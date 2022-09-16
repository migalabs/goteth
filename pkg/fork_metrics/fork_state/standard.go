package fork_state

import (
	"bytes"
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.WithField(
		"module", "FoskStateContent",
	)
)

// This Wrapper is meant to include all common objects across Ethereum Hard Fork Specs
type ForkStateContentBase struct {
	Version                 spec.DataVersion
	Balances                []uint64                     // balance of each validator
	Validators              []*phase0.Validator          // list of validators
	TotalActiveBalance      uint64                       // effective balance
	TotalActiveRealBalance  uint64                       // real balance
	AttestingBalance        []uint64                     // one attesting balance per flag
	MaxAttestingBalance     uint64                       // the effective balance of validators that did attest in any manner
	EpochStructs            EpochData                    // structs about beacon committees, proposers and attestation
	CorrectFlags            [][]uint64                   // one aray per flag
	AttestingVals           []bool                       // the number of validators that did attest in the last epoch
	PrevAttestations        []*phase0.PendingAttestation // array of attestations (currently only for Phase0)
	NumAttestingVals        uint64                       // number of validators that attested in the last epoch
	NumActiveVals           uint64                       // number of active validators in the epoch
	ValAttestationInclusion map[uint64]ValVote           // one map per validator including which slots it had to attest and when it was included
	AttestedValsPerSlot     map[uint64][]uint64          // for each slot in the epoch, how many vals attested
	Epoch                   uint64                       // Epoch of the state
	Slot                    uint64                       // Slot of the state
	BlockRoots              [][]byte                     // array of block roots at this point (8192)
	MissedBlocks            []uint64                     // blocks missed in the epoch until this point
	SyncCommittee           altair.SyncCommittee         // list of pubkeys in the current sync committe
}

func GetCustomState(bstate spec.VersionedBeaconState, iApi *http.Service) (ForkStateContentBase, error) {
	switch bstate.Version {

	case spec.DataVersionPhase0:
		return NewPhase0State(bstate, iApi), nil

	case spec.DataVersionAltair:
		return NewAltairState(bstate, iApi), nil

	case spec.DataVersionBellatrix:
		return NewBellatrixState(bstate, iApi), nil
	default:
		return ForkStateContentBase{}, fmt.Errorf("could not figure out the Beacon State Fork Version: %s", bstate.Version)
	}
}

// Initialize all necessary arrays and process anything standard
func (p *ForkStateContentBase) Setup() error {
	if p.Validators == nil {
		return fmt.Errorf("validator list not provided, cannot create")
	}
	arrayLen := len(p.Validators)
	if p.PrevAttestations == nil {
		p.PrevAttestations = make([]*phase0.PendingAttestation, 0)
	}

	p.AttestingBalance = make([]uint64, 3)
	p.AttestingVals = make([]bool, arrayLen)
	p.CorrectFlags = make([][]uint64, 3)
	p.MissedBlocks = make([]uint64, 0)

	for i := range p.CorrectFlags {
		p.CorrectFlags[i] = make([]uint64, arrayLen)
	}

	p.TotalActiveBalance = p.GetTotalActiveEffBalance()
	p.TrackMissingBlocks()
	return nil
}

// the length of the valList = number of validators
// each position represents a valIdx
// if the item has a number > 0, count it
func (p ForkStateContentBase) ValsEffectiveBalance(valList []uint64) uint64 {

	resultBalance := uint64(0)

	for valIdx, item := range valList { // loop over validators
		if item > 0 && valIdx < len(p.Validators) {
			resultBalance += uint64(p.Validators[valIdx].EffectiveBalance)
		}
	}

	return uint64(resultBalance)
}

func (p ForkStateContentBase) Balance(valIdx uint64) (uint64, error) {
	if uint64(len(p.Balances)) < valIdx {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.Slot)
		return 0, err
	}
	balance := p.Balances[valIdx]

	return balance, nil
}

// Edit NumActiveVals
func (p *ForkStateContentBase) GetTotalActiveEffBalance() uint64 {

	val_array := make([]uint64, len(p.Validators))
	p.NumActiveVals = 0 // any time we calculate total effective balance, the number of active vals is refreshed and recalculated
	for idx := range val_array {
		if IsActive(*p.Validators[idx], phase0.Epoch(p.Epoch)) {
			val_array[idx] += 1
			p.NumActiveVals++
		}

	}

	return p.ValsEffectiveBalance(val_array)
}

// Not effective balance, but balance
func (p ForkStateContentBase) GetTotalActiveRealBalance() uint64 {
	totalBalance := uint64(0)

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
func (p *ForkStateContentBase) TrackMissingBlocks() {
	firstIndex := (p.Slot - SLOTS_PER_EPOCH + 1) % SLOTS_PER_HISTORICAL_ROOT
	lastIndex := (p.Slot) % SLOTS_PER_HISTORICAL_ROOT

	for i := firstIndex; i <= lastIndex; i++ {
		if i == 0 {
			continue
		}
		lastItem := p.BlockRoots[i-1]
		item := p.BlockRoots[i]
		res := bytes.Compare(lastItem, item)

		if res == 0 {
			// both consecutive roots were the same ==> missed block
			slot := i - firstIndex + p.Slot - SLOTS_PER_EPOCH + 1
			p.MissedBlocks = append(p.MissedBlocks, uint64(slot))
		}
	}
}

// List of validators that were active in the epoch of the state
// Lenght of the list is variable, each position containing the valIdx
func (p ForkStateContentBase) GetActiveVals() []uint64 {
	result := make([]uint64, 0)

	for i, val := range p.Validators {
		if IsActive(*val, phase0.Epoch(p.Epoch)) {
			result = append(result, uint64(i))
		}

	}
	return result
}

// Returns a list of missing flags for the corresponding valIdx
func (p ForkStateContentBase) MissingFlags(valIdx uint64) []bool {
	result := []bool{true, true, true}

	if int(valIdx) >= len(p.CorrectFlags[0]) {
		return result
	}

	for i, item := range p.CorrectFlags {
		if item[valIdx] == 0 {
			// missing flag
			result[i] = true
		}

	}
	return result
}

// Argument: 0 for source, 1 for target and 2 for head
// Return the count of missing flag in the previous epoch participation / attestations
func (p ForkStateContentBase) GetMissingFlagCount(flagIndex int) uint64 {
	result := uint64(0)
	for idx, item := range p.CorrectFlags[flagIndex] {
		// if validator was active and no correct flag
		if IsActive(*p.Validators[idx], phase0.Epoch(p.Epoch-1)) && item == 0 {
			result += 1
		}
	}

	return result
}
