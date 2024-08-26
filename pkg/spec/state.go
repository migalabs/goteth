package spec

import (
	"bytes"
	"fmt"
	"math"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type AgnosticState struct {
	Version                    spec.DataVersion
	GenesisTimestamp           uint64 // genesis timestamp
	StateRoot                  phase0.Root
	Epoch                      phase0.Epoch                 // Epoch of the state
	Slot                       phase0.Slot                  // Slot of the state
	Balances                   []phase0.Gwei                // balance of each validator
	Validators                 []*phase0.Validator          // list of validators
	TotalActiveBalance         phase0.Gwei                  // effective balance
	TotalActiveRealBalance     phase0.Gwei                  // real balance
	AttestingBalance           []phase0.Gwei                // one attesting balance per flag (of the previous epoch attestations)
	EpochStructs               EpochDuties                  // structs about beacon committees, proposers and attestation
	PrevEpochCorrectFlags      [][]bool                     // one aray per flag
	PrevAttestations           []*phase0.PendingAttestation // array of attestations (currently only for Phase0)
	NumActiveVals              uint                         // number of active validators in the epoch
	NumExitedVals              uint                         // number of exited validators in the epoch
	NumSlashedVals             uint                         // number of slashed validators in the epoch
	NumQueuedVals              uint                         // number of validators in the queue
	BlockRoots                 []phase0.Root                // array of block roots at this point (8192)
	MissedBlocks               []phase0.Slot                // blocks missed in the epoch until this point
	SyncCommittee              altair.SyncCommittee         // list of pubkeys in the current sync committe
	Blocks                     []*AgnosticBlock             // list of blocks in the epoch
	Withdrawals                []phase0.Gwei                // one position per validator
	Deposits                   []phase0.Gwei                // one per validator index
	CurrentJustifiedCheckpoint phase0.Checkpoint            // the latest justified checkpoint
	LatestBlockHeader          *phase0.BeaconBlockHeader
}

func GetCustomState(bstate spec.VersionedBeaconState, duties EpochDuties) (AgnosticState, error) {
	switch bstate.Version {

	case spec.DataVersionPhase0:
		return NewPhase0State(bstate, duties), nil

	case spec.DataVersionAltair:
		return NewAltairState(bstate, duties), nil

	case spec.DataVersionBellatrix:
		return NewBellatrixState(bstate, duties), nil

	case spec.DataVersionCapella:
		return NewCapellaState(bstate, duties), nil
	case spec.DataVersionDeneb:
		return NewDenebState(bstate, duties), nil
	default:
		return AgnosticState{}, fmt.Errorf("could not figure out the Beacon State Fork Version: %s", bstate.Version)
	}
}

// Initialize all necessary arrays and process anything standard
func (p *AgnosticState) Setup() error {
	if p.Validators == nil {
		return fmt.Errorf("validator list not provided, cannot create")
	}
	arrayLen := len(p.Validators)
	if p.PrevAttestations == nil {
		p.PrevAttestations = make([]*phase0.PendingAttestation, 0)
	}

	p.AttestingBalance = make([]phase0.Gwei, 3)
	p.PrevEpochCorrectFlags = make([][]bool, 3) // matrix of 3 x n_validators
	for i := range p.PrevEpochCorrectFlags {
		p.PrevEpochCorrectFlags[i] = make([]bool, arrayLen)
	}
	p.GetValsStateNums()
	p.TotalActiveBalance = p.GetTotalActiveEffBalance()
	p.TotalActiveRealBalance = p.GetTotalActiveRealBalance()
	p.TrackMissingBlocks()
	return nil
}

func (p *AgnosticState) AddBlocks(blockList []*AgnosticBlock) {
	p.Blocks = blockList
	p.CalculateWithdrawals()
	p.CalculateDeposits()
}

func (p *AgnosticState) CalculateWithdrawals() {

	p.Withdrawals = make([]phase0.Gwei, len(p.Validators))
	for _, block := range p.Blocks {
		for _, withdrawal := range block.ExecutionPayload.Withdrawals {
			p.Withdrawals[withdrawal.ValidatorIndex] += withdrawal.Amount
		}

	}
}

func (p *AgnosticState) CalculateDeposits() {

	p.Deposits = make([]phase0.Gwei, len(p.Validators))
	for _, block := range p.Blocks {
		for _, deposit := range block.Deposits {

			for valIDx, validator := range p.Validators {
				if deposit.Data.PublicKey == validator.PublicKey {
					p.Deposits[valIDx] += deposit.Data.Amount
				}
			}
		}

	}
}

// the length of the valList = number of validators
// each position represents a valIdx
// if the item has a number > 0, count it
func (p AgnosticState) ValsEffectiveBalance(valList []phase0.Gwei) phase0.Gwei {

	resultBalance := phase0.Gwei(0)

	for valIdx, item := range valList { // loop over validators
		if item > 0 && valIdx < len(p.Validators) {
			resultBalance += p.Validators[valIdx].EffectiveBalance
		}
	}

	return resultBalance
}

func (p *AgnosticState) GetTotalActiveEffBalance() phase0.Gwei {

	val_array := make([]phase0.Gwei, len(p.Validators))
	for idx := range val_array {
		if IsActive(*p.Validators[idx], phase0.Epoch(p.Epoch)) {
			val_array[idx] += 1
		}
	}

	return p.ValsEffectiveBalance(val_array)
}

func (p *AgnosticState) GetValsStateNums() {
	result := p.GetValsPerStatus()
	p.NumActiveVals = uint(len(result[ACTIVE_STATUS]))
	p.NumExitedVals = uint(len(result[EXIT_STATUS]))
	p.NumSlashedVals = uint(len(result[SLASHED_STATUS]))
	p.NumQueuedVals = uint(len(result[QUEUE_STATUS]))
}

// Not effective balance, but balance
func (p AgnosticState) GetTotalActiveRealBalance() phase0.Gwei {
	totalBalance := phase0.Gwei(0)

	for idx := range p.Validators {
		if IsActive(*p.Validators[idx], phase0.Epoch(p.Epoch)) {
			totalBalance += p.Balances[idx]
		}

	}
	return totalBalance
}

func IsActive(validator phase0.Validator, epoch phase0.Epoch) bool {
	if validator.ActivationEpoch <= epoch &&
		epoch < validator.ExitEpoch {
		return true
	}
	return false
}

// We use blockroots to track missed blocks. When there is a missed block, the block root is repeated
func (p *AgnosticState) TrackMissingBlocks() {
	firstSlotOfEpoch := phase0.Slot(p.Epoch * SlotsPerEpoch)
	lastSlotOfEpoch := phase0.Slot(p.Epoch*SlotsPerEpoch + SlotsPerEpoch - 1)
	firstIndex := firstSlotOfEpoch % SlotsPerHistoricalRoot // first slot of the epoch
	lastIndex := lastSlotOfEpoch % SlotsPerHistoricalRoot   // last slot of the epoch
	p.MissedBlocks = make([]phase0.Slot, 0)

	for i := firstIndex; i < lastIndex; i++ {
		var previousItem phase0.Root
		if i == 0 {
			previousItem = p.BlockRoots[SlotsPerHistoricalRoot-1] // wrap around
		} else {
			previousItem = p.BlockRoots[i-1] // prevBlock, starting at previous slot of prevEpoch
		}
		item := p.BlockRoots[i]                        // currentBlock, starting at slot0 of the epoch
		res := bytes.Compare(previousItem[:], item[:]) // if equal, currentBlock was missed

		if res == 0 {
			// both consecutive roots were the same ==> missed block
			slot := i - firstIndex + phase0.Slot(p.Epoch*SlotsPerEpoch) // delta + start of the epoch
			p.MissedBlocks = append(p.MissedBlocks, slot)
		}
	}

	// Handle the last slot of the epoch separately since the block root of the last slot of the epoch is not included in the block roots list
	if p.LatestBlockHeader.Slot != lastSlotOfEpoch {
		p.MissedBlocks = append(p.MissedBlocks, lastSlotOfEpoch)
	}
}

func (p AgnosticState) GetValsPerStatus() [][]uint64 {
	result := make([][]uint64, NUMBER_OF_STATUS)

	for i := range result {
		result[i] = make([]uint64, 0)
	}

	for i := range p.Validators {
		status := p.GetValStatus(phase0.ValidatorIndex(i))
		result[status] = append(result[status], uint64(i))
	}

	return result
}

// Returns a list of missing flags for the corresponding valIdx
func (p AgnosticState) MissingFlags(valIdx phase0.ValidatorIndex) []bool {
	result := []bool{false, false, false}

	if int(valIdx) >= len(p.PrevEpochCorrectFlags[0]) {
		return result
	}

	if IsActive(*p.Validators[valIdx], phase0.Epoch(p.Epoch-1)) {
		for i, item := range p.PrevEpochCorrectFlags {
			if !item[valIdx] {
				// no missing flag
				result[i] = true
			}
		}
	}
	return result
}

// Argument: 0 for source, 1 for target and 2 for head
// Return the count of missing flag in the previous epoch participation / attestations
func (p AgnosticState) GetMissingFlagCount(flagIndex int) uint64 {
	result := uint64(0)
	for idx, item := range p.PrevEpochCorrectFlags[flagIndex] {
		// if validator was active and no correct flag
		if IsActive(*p.Validators[idx], phase0.Epoch(p.Epoch-1)) && !item {
			result += 1
		}
	}

	return result
}

func (p AgnosticState) GetValStatus(valIdx phase0.ValidatorIndex) ValidatorStatus {

	if p.Validators[valIdx].Slashed {
		return SLASHED_STATUS
	}

	if p.Validators[valIdx].ExitEpoch <= phase0.Epoch(p.Epoch) {
		return EXIT_STATUS
	}

	if p.Validators[valIdx].ActivationEpoch <= phase0.Epoch(p.Epoch) {
		return ACTIVE_STATUS
	}

	return QUEUE_STATUS

}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#get_block_root
func (p AgnosticState) GetBlockRoot(epoch phase0.Epoch) phase0.Root {

	firstSlotInEpoch := phase0.Slot(epoch * SlotsPerEpoch)

	return p.GetBlockRootAtSlot(firstSlotInEpoch)
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#get_block_root_at_slot
func (p AgnosticState) GetBlockRootAtSlot(slot phase0.Slot) phase0.Root {

	return p.BlockRoots[slot%SlotsPerHistoricalRoot]
}

// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#get_block_root_at_slot
func (p AgnosticState) EmptyStateRoot() bool {

	return p.StateRoot == phase0.Root{}
}

// This Wrapper is meant to include all necessary data from the Phase0 Fork
func NewPhase0State(bstate spec.VersionedBeaconState, duties EpochDuties) AgnosticState {

	balances := make([]phase0.Gwei, 0)

	for _, item := range bstate.Phase0.Balances {
		balances = append(balances, phase0.Gwei(item))
	}

	phase0Obj := AgnosticState{

		Version:                    bstate.Version,
		Balances:                   balances,
		Validators:                 bstate.Phase0.Validators,
		EpochStructs:               duties,
		Epoch:                      phase0.Epoch(bstate.Phase0.Slot / SlotsPerEpoch),
		Slot:                       phase0.Slot(bstate.Phase0.Slot),
		BlockRoots:                 bstate.Phase0.BlockRoots,
		PrevAttestations:           bstate.Phase0.PreviousEpochAttestations,
		GenesisTimestamp:           bstate.Phase0.GenesisTime,
		CurrentJustifiedCheckpoint: *bstate.Phase0.CurrentJustifiedCheckpoint,
		LatestBlockHeader:          bstate.Phase0.LatestBlockHeader,
	}

	phase0Obj.Setup()

	return phase0Obj

}

// This Wrapper is meant to include all necessary data from the Altair Fork
func NewAltairState(bstate spec.VersionedBeaconState, duties EpochDuties) AgnosticState {

	altairObj := AgnosticState{
		Version:                    bstate.Version,
		Balances:                   bstate.Altair.Balances,
		Validators:                 bstate.Altair.Validators,
		EpochStructs:               duties,
		Epoch:                      phase0.Epoch(bstate.Altair.Slot / SlotsPerEpoch),
		Slot:                       bstate.Altair.Slot,
		BlockRoots:                 bstate.Altair.BlockRoots,
		SyncCommittee:              *bstate.Altair.CurrentSyncCommittee,
		GenesisTimestamp:           bstate.Altair.GenesisTime,
		CurrentJustifiedCheckpoint: *bstate.Altair.CurrentJustifiedCheckpoint,
		LatestBlockHeader:          bstate.Altair.LatestBlockHeader,
	}

	altairObj.Setup()

	ProcessAltairAttestations(&altairObj, bstate.Altair.PreviousEpochParticipation)

	return altairObj
}

func ProcessAltairAttestations(customState *AgnosticState, prevEpochParticipation []altair.ParticipationFlags) {
	// calculate attesting vals only once
	flags := []altair.ParticipationFlag{
		altair.TimelySourceFlagIndex,
		altair.TimelyTargetFlagIndex,
		altair.TimelyHeadFlagIndex}

	for participatingFlag := range flags {

		flag := altair.ParticipationFlags(math.Pow(2, float64(participatingFlag)))

		for valIndex, item := range prevEpochParticipation {
			// Here we have one item per validator
			// Item is a 3-bit string
			// each bit represents a flag

			if (item & flag) == flag {
				// The attestation has a timely flag, therefore we consider it correct flag
				customState.PrevEpochCorrectFlags[participatingFlag][valIndex] = true

				// we sum the attesting balance in the corresponding flag index
				customState.AttestingBalance[participatingFlag] += customState.Validators[valIndex].EffectiveBalance
			}
		}
	}
}

// This Wrapper is meant to include all necessary data from the Bellatrix Fork
func NewBellatrixState(bstate spec.VersionedBeaconState, duties EpochDuties) AgnosticState {

	bellatrixObj := AgnosticState{
		Version:                    bstate.Version,
		Balances:                   bstate.Bellatrix.Balances,
		Validators:                 bstate.Bellatrix.Validators,
		EpochStructs:               duties,
		Epoch:                      phase0.Epoch(bstate.Bellatrix.Slot / SlotsPerEpoch),
		Slot:                       bstate.Bellatrix.Slot,
		BlockRoots:                 bstate.Bellatrix.BlockRoots,
		SyncCommittee:              *bstate.Bellatrix.CurrentSyncCommittee,
		GenesisTimestamp:           bstate.Bellatrix.GenesisTime,
		CurrentJustifiedCheckpoint: *bstate.Bellatrix.CurrentJustifiedCheckpoint,
		LatestBlockHeader:          bstate.Bellatrix.LatestBlockHeader,
	}

	bellatrixObj.Setup()

	ProcessAltairAttestations(&bellatrixObj, bstate.Bellatrix.PreviousEpochParticipation)

	return bellatrixObj
}

// This Wrapper is meant to include all necessary data from the Capella Fork
func NewCapellaState(bstate spec.VersionedBeaconState, duties EpochDuties) AgnosticState {

	capellaObj := AgnosticState{
		Version:                    bstate.Version,
		Balances:                   bstate.Capella.Balances,
		Validators:                 bstate.Capella.Validators,
		EpochStructs:               duties,
		Epoch:                      phase0.Epoch(bstate.Capella.Slot / SlotsPerEpoch),
		Slot:                       bstate.Capella.Slot,
		BlockRoots:                 bstate.Capella.BlockRoots,
		SyncCommittee:              *bstate.Capella.CurrentSyncCommittee,
		GenesisTimestamp:           bstate.Capella.GenesisTime,
		CurrentJustifiedCheckpoint: *bstate.Capella.CurrentJustifiedCheckpoint,
		LatestBlockHeader:          bstate.Capella.LatestBlockHeader,
	}

	capellaObj.Setup()

	ProcessAltairAttestations(&capellaObj, bstate.Capella.PreviousEpochParticipation)

	return capellaObj
}

// This Wrapper is meant to include all necessary data from the Capella Fork
func NewDenebState(bstate spec.VersionedBeaconState, duties EpochDuties) AgnosticState {

	denebObj := AgnosticState{
		Version:                    bstate.Version,
		Balances:                   bstate.Deneb.Balances,
		Validators:                 bstate.Deneb.Validators,
		EpochStructs:               duties,
		Epoch:                      phase0.Epoch(bstate.Deneb.Slot / SlotsPerEpoch),
		Slot:                       bstate.Deneb.Slot,
		BlockRoots:                 bstate.Deneb.BlockRoots,
		SyncCommittee:              *bstate.Deneb.CurrentSyncCommittee,
		GenesisTimestamp:           bstate.Deneb.GenesisTime,
		CurrentJustifiedCheckpoint: *bstate.Deneb.CurrentJustifiedCheckpoint,
		LatestBlockHeader:          bstate.Deneb.LatestBlockHeader,
	}

	denebObj.Setup()

	ProcessAltairAttestations(&denebObj, bstate.Deneb.PreviousEpochParticipation)

	return denebObj
}
