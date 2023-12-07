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
	Version                 spec.DataVersion
	StateRoot               phase0.Root
	Balances                []phase0.Gwei                     // balance of each validator
	Validators              []*phase0.Validator               // list of validators
	TotalActiveBalance      phase0.Gwei                       // effective balance
	TotalActiveRealBalance  phase0.Gwei                       // real balance
	AttestingBalance        []phase0.Gwei                     // one attesting balance per flag (of the previous epoch attestations)
	MaxAttestingBalance     phase0.Gwei                       // the effective balance of validators that did attest in any manner
	EpochStructs            EpochDuties                       // structs about beacon committees, proposers and attestation
	CorrectFlags            [][]uint                          // one aray per flag
	AttestingVals           []bool                            // the number of validators that did attest in the last epoch
	PrevAttestations        []*phase0.PendingAttestation      // array of attestations (currently only for Phase0)
	NumAttestingVals        uint                              // number of validators that attested in the last epoch
	NumActiveVals           uint                              // number of active validators in the epoch
	NumExitedVals           uint                              // number of exited validators in the epoch
	NumSlashedVals          uint                              // number of slashed validators in the epoch
	NumQueuedVals           uint                              // number of validators in the queue
	ValAttestationInclusion map[phase0.ValidatorIndex]ValVote // one map per validator including which slots it had to attest and when it was included
	AttestedValsPerSlot     map[phase0.Slot][]uint64          // for each slot in the epoch, how many vals attested
	Epoch                   phase0.Epoch                      // Epoch of the state
	Slot                    phase0.Slot                       // Slot of the state
	BlockRoots              []phase0.Root                     // array of block roots at this point (8192)
	MissedBlocks            []phase0.Slot                     // blocks missed in the epoch until this point
	SyncCommittee           altair.SyncCommittee              // list of pubkeys in the current sync committe
	Blocks                  []AgnosticBlock                   // list of blocks in the epoch
	Withdrawals             []phase0.Gwei                     // one position per validator
	GenesisTimestamp        uint64                            // genesis timestamp
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
	p.AttestingVals = make([]bool, arrayLen)
	p.CorrectFlags = make([][]uint, 3)
	p.MissedBlocks = make([]phase0.Slot, 0)
	p.ValAttestationInclusion = make(map[phase0.ValidatorIndex]ValVote)
	p.AttestedValsPerSlot = make(map[phase0.Slot][]uint64)
	p.Withdrawals = make([]phase0.Gwei, len(p.Validators))

	for i := range p.CorrectFlags {
		p.CorrectFlags[i] = make([]uint, arrayLen)
	}
	p.GetValsStateNums()
	p.TotalActiveBalance = p.GetTotalActiveEffBalance()
	p.TotalActiveRealBalance = p.GetTotalActiveRealBalance()
	p.TrackMissingBlocks()
	return nil
}

func (p *AgnosticState) AddBlocks(blockList []AgnosticBlock) {
	p.Blocks = blockList
	p.CalculateWithdrawals()
}

func (p *AgnosticState) CalculateWithdrawals() {

	p.Withdrawals = make([]phase0.Gwei, len(p.Validators))
	for _, block := range p.Blocks {
		for _, withdrawal := range block.ExecutionPayload.Withdrawals {
			p.Withdrawals[withdrawal.ValidatorIndex] += withdrawal.Amount
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

func (p AgnosticState) Balance(valIdx phase0.ValidatorIndex) (phase0.Gwei, error) {
	if uint64(len(p.Balances)) < uint64(valIdx) {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.Slot)
		return 0, err
	}
	balance := p.Balances[valIdx]

	return balance, nil
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

// check if there was a missed block at last slot of previous epoch
func (p AgnosticState) TrackPrevMissingBlock() phase0.Slot {
	firstIndex := (p.Slot - SlotsPerEpoch) % SlotsPerHistoricalRoot

	lastItem := p.BlockRoots[firstIndex-1]
	item := p.BlockRoots[firstIndex]
	res := bytes.Compare(lastItem[:], item[:])

	if res == 0 {
		// both consecutive roots were the same ==> missed block
		slot := p.Slot - SlotsPerEpoch
		return slot
	}

	return 0

}

// We use blockroots to track missed blocks. When there is a missed block, the block root is repeated
func (p *AgnosticState) TrackMissingBlocks() {

	firstIndex := phase0.Slot(p.Epoch*SlotsPerEpoch) % SlotsPerHistoricalRoot                // first slot of the epoch
	lastIndex := phase0.Slot(p.Epoch*SlotsPerEpoch+SlotsPerEpoch-1) % SlotsPerHistoricalRoot // last slot of the epoch

	for i := firstIndex; i <= lastIndex; i++ {
		if i == 0 {
			continue
		}
		lastItem := p.BlockRoots[i-1]              // prevBlock, starting at last slot of prevEpoch
		item := p.BlockRoots[i]                    // currentBlock, starting at slot0 of the epoch
		res := bytes.Compare(lastItem[:], item[:]) // if equal, currentBlock was missed

		if res == 0 {
			// both consecutive roots were the same ==> missed block
			slot := i - firstIndex + phase0.Slot(p.Epoch*SlotsPerEpoch) // delta + start of the epoch
			p.MissedBlocks = append(p.MissedBlocks, slot)
		}
	}
}

// List of validators that were active in the epoch of the state
// Lenght of the list is variable, each position containing the valIdx
func (p AgnosticState) GetActiveVals() []uint64 {
	result := make([]uint64, 0)

	for i, val := range p.Validators {
		if IsActive(*val, phase0.Epoch(p.Epoch)) {
			result = append(result, uint64(i))
		}

	}
	return result
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

// List of validators that were in the epoch of the state
// Length of the list is variable, each position containing the valIdx
func (p AgnosticState) GetAllVals() []phase0.ValidatorIndex {
	result := make([]phase0.ValidatorIndex, 0)

	for i := range p.Validators {
		result = append(result, phase0.ValidatorIndex(i))

	}
	return result
}

// Returns a list of missing flags for the corresponding valIdx
func (p AgnosticState) MissingFlags(valIdx phase0.ValidatorIndex) []bool {
	result := []bool{false, false, false}

	if int(valIdx) >= len(p.CorrectFlags[0]) {
		return result
	}

	for i, item := range p.CorrectFlags {
		if IsActive(*p.Validators[valIdx], phase0.Epoch(p.Epoch-1)) && item[valIdx] == 0 {
			if item[valIdx] == 0 {
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
	for idx, item := range p.CorrectFlags[flagIndex] {
		// if validator was active and no correct flag
		if IsActive(*p.Validators[idx], phase0.Epoch(p.Epoch-1)) && item == 0 {
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

// This Wrapper is meant to include all necessary data from the Phase0 Fork
func NewPhase0State(bstate spec.VersionedBeaconState, duties EpochDuties) AgnosticState {

	balances := make([]phase0.Gwei, 0)

	for _, item := range bstate.Phase0.Balances {
		balances = append(balances, phase0.Gwei(item))
	}

	phase0Obj := AgnosticState{
		Version:          bstate.Version,
		Balances:         balances,
		Validators:       bstate.Phase0.Validators,
		EpochStructs:     duties,
		Epoch:            phase0.Epoch(bstate.Phase0.Slot / SlotsPerEpoch),
		Slot:             phase0.Slot(bstate.Phase0.Slot),
		BlockRoots:       bstate.Phase0.BlockRoots,
		PrevAttestations: bstate.Phase0.PreviousEpochAttestations,
		GenesisTimestamp: bstate.Phase0.GenesisTime,
	}

	phase0Obj.Setup()

	return phase0Obj

}

// This Wrapper is meant to include all necessary data from the Altair Fork
func NewAltairState(bstate spec.VersionedBeaconState, duties EpochDuties) AgnosticState {

	altairObj := AgnosticState{
		Version:          bstate.Version,
		Balances:         bstate.Altair.Balances,
		Validators:       bstate.Altair.Validators,
		EpochStructs:     duties,
		Epoch:            phase0.Epoch(bstate.Altair.Slot / SlotsPerEpoch),
		Slot:             bstate.Altair.Slot,
		BlockRoots:       bstate.Altair.BlockRoots,
		SyncCommittee:    *bstate.Altair.CurrentSyncCommittee,
		GenesisTimestamp: bstate.Altair.GenesisTime,
	}

	altairObj.Setup()

	ProcessAltairAttestations(&altairObj, bstate.Altair.PreviousEpochParticipation)

	return altairObj
}

func ProcessAltairAttestations(customState *AgnosticState, participation []altair.ParticipationFlags) {
	// calculate attesting vals only once
	flags := []altair.ParticipationFlag{
		altair.TimelySourceFlagIndex,
		altair.TimelyTargetFlagIndex,
		altair.TimelyHeadFlagIndex}

	for participatingFlag := range flags {

		flag := altair.ParticipationFlags(math.Pow(2, float64(participatingFlag)))

		for valIndex, item := range participation {
			// Here we have one item per validator
			// Item is a 3-bit string
			// each bit represents a flag

			if (item & flag) == flag {
				// The attestation has a timely flag, therefore we consider it correct flag
				customState.CorrectFlags[participatingFlag][valIndex] += uint(1)

				// we sum the attesting balance in the corresponding flag index
				customState.AttestingBalance[participatingFlag] += customState.Validators[valIndex].EffectiveBalance

				// if this validator was not counted as attesting before, count it now
				if !customState.AttestingVals[valIndex] {
					customState.NumAttestingVals++
					customState.MaxAttestingBalance = customState.Validators[valIndex].EffectiveBalance
				}
				customState.AttestingVals[valIndex] = true
			}
		}
	}
}

// This Wrapper is meant to include all necessary data from the Bellatrix Fork
func NewBellatrixState(bstate spec.VersionedBeaconState, duties EpochDuties) AgnosticState {

	bellatrixObj := AgnosticState{
		Version:          bstate.Version,
		Balances:         bstate.Bellatrix.Balances,
		Validators:       bstate.Bellatrix.Validators,
		EpochStructs:     duties,
		Epoch:            phase0.Epoch(bstate.Bellatrix.Slot / SlotsPerEpoch),
		Slot:             bstate.Bellatrix.Slot,
		BlockRoots:       bstate.Bellatrix.BlockRoots,
		SyncCommittee:    *bstate.Bellatrix.CurrentSyncCommittee,
		GenesisTimestamp: bstate.Bellatrix.GenesisTime,
	}

	bellatrixObj.Setup()

	ProcessAltairAttestations(&bellatrixObj, bstate.Bellatrix.PreviousEpochParticipation)

	return bellatrixObj
}

// This Wrapper is meant to include all necessary data from the Capella Fork
func NewCapellaState(bstate spec.VersionedBeaconState, duties EpochDuties) AgnosticState {

	capellaObj := AgnosticState{
		Version:          bstate.Version,
		Balances:         bstate.Capella.Balances,
		Validators:       bstate.Capella.Validators,
		EpochStructs:     duties,
		Epoch:            phase0.Epoch(bstate.Capella.Slot / SlotsPerEpoch),
		Slot:             bstate.Capella.Slot,
		BlockRoots:       bstate.Capella.BlockRoots,
		SyncCommittee:    *bstate.Capella.CurrentSyncCommittee,
		GenesisTimestamp: bstate.Capella.GenesisTime,
	}

	capellaObj.Setup()

	ProcessAltairAttestations(&capellaObj, bstate.Capella.PreviousEpochParticipation)

	return capellaObj
}

// This Wrapper is meant to include all necessary data from the Capella Fork
func NewDenebState(bstate spec.VersionedBeaconState, duties EpochDuties) AgnosticState {

	denebObj := AgnosticState{
		Version:          bstate.Version,
		Balances:         bstate.Deneb.Balances,
		Validators:       bstate.Deneb.Validators,
		EpochStructs:     duties,
		Epoch:            phase0.Epoch(bstate.Deneb.Slot / SlotsPerEpoch),
		Slot:             bstate.Deneb.Slot,
		BlockRoots:       bstate.Deneb.BlockRoots,
		SyncCommittee:    *bstate.Deneb.CurrentSyncCommittee,
		GenesisTimestamp: bstate.Deneb.GenesisTime,
	}

	denebObj.Setup()

	ProcessAltairAttestations(&denebObj, bstate.Deneb.PreviousEpochParticipation)

	return denebObj
}
