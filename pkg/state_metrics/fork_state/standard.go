package fork_state

import (
	"bytes"
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
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
	Balances                []phase0.Gwei                     // balance of each validator
	Validators              []*phase0.Validator               // list of validators
	TotalActiveBalance      phase0.Gwei                       // effective balance
	TotalActiveRealBalance  phase0.Gwei                       // real balance
	AttestingBalance        []phase0.Gwei                     // one attesting balance per flag (of the previous epoch attestations)
	MaxAttestingBalance     phase0.Gwei                       // the effective balance of validators that did attest in any manner
	EpochStructs            EpochData                         // structs about beacon committees, proposers and attestation
	CorrectFlags            [][]uint                          // one aray per flag
	AttestingVals           []bool                            // the number of validators that did attest in the last epoch
	PrevAttestations        []*phase0.PendingAttestation      // array of attestations (currently only for Phase0)
	NumAttestingVals        uint                              // number of validators that attested in the last epoch
	NumActiveVals           uint                              // number of active validators in the epoch
	ValAttestationInclusion map[phase0.ValidatorIndex]ValVote // one map per validator including which slots it had to attest and when it was included
	AttestedValsPerSlot     map[phase0.Slot][]uint64          // for each slot in the epoch, how many vals attested
	Epoch                   phase0.Epoch                      // Epoch of the state
	Slot                    phase0.Slot                       // Slot of the state
	BlockRoots              [][]byte                          // array of block roots at this point (8192)
	MissedBlocks            []phase0.Slot                     // blocks missed in the epoch until this point
	SyncCommittee           altair.SyncCommittee              // list of pubkeys in the current sync committe
}

func GetCustomState(bstate spec.VersionedBeaconState, iApi *http.Service) (ForkStateContentBase, error) {
	switch bstate.Version {

	case spec.DataVersionPhase0:
		return NewPhase0State(bstate, iApi), nil

	case spec.DataVersionAltair:
		return NewAltairState(bstate, iApi), nil

	case spec.DataVersionBellatrix:
		return NewBellatrixState(bstate, iApi), nil

	case spec.DataVersionCapella:
		return NewCapellaState(bstate, iApi), nil
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

	p.AttestingBalance = make([]phase0.Gwei, 3)
	p.AttestingVals = make([]bool, arrayLen)
	p.CorrectFlags = make([][]uint, 3)
	p.MissedBlocks = make([]phase0.Slot, 0)
	p.ValAttestationInclusion = make(map[phase0.ValidatorIndex]ValVote)
	p.AttestedValsPerSlot = make(map[phase0.Slot][]uint64)

	for i := range p.CorrectFlags {
		p.CorrectFlags[i] = make([]uint, arrayLen)
	}

	p.TotalActiveBalance = p.GetTotalActiveEffBalance()
	p.TotalActiveRealBalance = p.GetTotalActiveRealBalance()
	p.TrackMissingBlocks()
	return nil
}

// the length of the valList = number of validators
// each position represents a valIdx
// if the item has a number > 0, count it
func (p ForkStateContentBase) ValsEffectiveBalance(valList []phase0.Gwei) phase0.Gwei {

	resultBalance := phase0.Gwei(0)

	for valIdx, item := range valList { // loop over validators
		if item > 0 && valIdx < len(p.Validators) {
			resultBalance += p.Validators[valIdx].EffectiveBalance
		}
	}

	return resultBalance
}

func (p ForkStateContentBase) Balance(valIdx phase0.ValidatorIndex) (phase0.Gwei, error) {
	if uint64(len(p.Balances)) < uint64(valIdx) {
		err := fmt.Errorf("phase0 - validator index %d wasn't activated in slot %d", valIdx, p.Slot)
		return 0, err
	}
	balance := p.Balances[valIdx]

	return balance, nil
}

// Edit NumActiveVals
func (p *ForkStateContentBase) GetTotalActiveEffBalance() phase0.Gwei {

	val_array := make([]phase0.Gwei, len(p.Validators))
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
func (p ForkStateContentBase) GetTotalActiveRealBalance() phase0.Gwei {
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
func (p ForkStateContentBase) TrackPrevMissingBlock() phase0.Slot {
	firstIndex := (p.Slot - utils.SLOTS_PER_EPOCH) % utils.SLOTS_PER_HISTORICAL_ROOT

	lastItem := p.BlockRoots[firstIndex-1]
	item := p.BlockRoots[firstIndex]
	res := bytes.Compare(lastItem, item)

	if res == 0 {
		// both consecutive roots were the same ==> missed block
		slot := p.Slot - utils.SLOTS_PER_EPOCH
		return slot
	}

	return 0

}

// We use blockroots to track missed blocks. When there is a missed block, the block root is repeated
func (p *ForkStateContentBase) TrackMissingBlocks() {
	firstIndex := (p.Slot - utils.SLOTS_PER_EPOCH + 1) % utils.SLOTS_PER_HISTORICAL_ROOT
	lastIndex := (p.Slot) % utils.SLOTS_PER_HISTORICAL_ROOT

	for i := firstIndex; i <= lastIndex; i++ {
		if i == 0 {
			continue
		}
		lastItem := p.BlockRoots[i-1]
		item := p.BlockRoots[i]
		res := bytes.Compare(lastItem, item)

		if res == 0 {
			// both consecutive roots were the same ==> missed block
			slot := i - firstIndex + p.Slot - utils.SLOTS_PER_EPOCH + 1
			p.MissedBlocks = append(p.MissedBlocks, slot)
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

// List of validators that were in the epoch of the state
// Length of the list is variable, each position containing the valIdx
func (p ForkStateContentBase) GetAllVals() []phase0.ValidatorIndex {
	result := make([]phase0.ValidatorIndex, 0)

	for i := range p.Validators {
		result = append(result, phase0.ValidatorIndex(i))

	}
	return result
}

// Returns a list of missing flags for the corresponding valIdx
func (p ForkStateContentBase) MissingFlags(valIdx phase0.ValidatorIndex) []bool {
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

func (p ForkStateContentBase) GetValStatus(valIdx phase0.ValidatorIndex) model.ValidatorStatus {

	if p.Validators[valIdx].ExitEpoch <= phase0.Epoch(p.Epoch) {
		return model.EXIT_STATUS
	}

	if p.Validators[valIdx].Slashed {
		return model.SLASHED_STATUS
	}

	if p.Validators[valIdx].ActivationEpoch <= phase0.Epoch(p.Epoch) {
		return model.ACTIVE_STATUS
	}

	return model.QUEUE_STATUS

}
