package metrics

import (
	"errors"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

func (s StateMetricsBase) GetStateAtSlot(slot phase0.Slot) (*spec.AgnosticState, error) {
	if slot >= phase0.Slot(s.PrevState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(s.CurrentState.Epoch)*spec.SlotsPerEpoch {
		// slot in PrevEpoch
		return s.PrevState, nil
	}

	if slot >= phase0.Slot(s.CurrentState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(s.NextState.Epoch)*spec.SlotsPerEpoch {
		// slot in CurrentEpoch
		return s.CurrentState, nil
	}

	if slot >= phase0.Slot(s.NextState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(s.NextState.Epoch+1)*spec.SlotsPerEpoch {
		// slot in NextEpoch
		return s.NextState, nil
	}

	return nil, errors.New("could not get state from any epoch")
}

func (p AltairMetrics) GetValidatorFromCommitteeIndex(slot phase0.Slot, committeeIndex phase0.CommitteeIndex, idx int) (phase0.ValidatorIndex, error) {
	state, err := p.baseMetrics.GetStateAtSlot(slot)
	if err != nil {

		return 0, fmt.Errorf("could not get validator from any epoch (slot %d, committee %d, index %d): %w", slot, committeeIndex, idx, err)
	}
	valList := state.EpochStructs.GetValList(slot, committeeIndex)
	return valList[idx], nil
}

func (p AltairMetrics) GetJustifiedRootfromSlot(slot phase0.Slot) (phase0.Root, error) {
	state, err := p.baseMetrics.GetStateAtSlot(slot)
	if err != nil {
		return phase0.Root{}, fmt.Errorf("could not get justified root from any epoch: slot %d", slot)
	}
	return state.CurrentJustifiedCheckpoint.Root, nil
}

func (s StateMetricsBase) GetBlockFromSlot(slot phase0.Slot) (*spec.AgnosticBlock, error) {
	if slot >= phase0.Slot(s.PrevState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(s.CurrentState.Epoch)*spec.SlotsPerEpoch {
		// slot in PrevEpoch
		return s.PrevState.Blocks[slot%spec.SlotsPerEpoch], nil
	}

	if slot >= phase0.Slot(s.CurrentState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(s.NextState.Epoch)*spec.SlotsPerEpoch {
		// slot in CurrentEpochEpoch
		return s.CurrentState.Blocks[slot%spec.SlotsPerEpoch], nil
	}

	if slot >= phase0.Slot(s.NextState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(s.NextState.Epoch+1)*spec.SlotsPerEpoch {
		// slot in NextEpoch
		return s.NextState.Blocks[slot%spec.SlotsPerEpoch], nil
	}

	return &spec.AgnosticBlock{}, errors.New("could not get block from any epoch")
}

// Returns the closest proposed block backwards from the given slot
func (s StateMetricsBase) GetBestInclusionDelay(slot phase0.Slot) (int, error) {

	minSlot := phase0.Slot(s.PrevState.Epoch * spec.SlotsPerEpoch)

	for i := slot; i > minSlot; i-- {
		block, err := s.GetBlockFromSlot(i)
		if err != nil {
			return 0, err
		}

		if block.Proposed {
			return int(slot - i), nil
		}
	}

	return int(slot - minSlot), nil
}

func countTrue(arr []bool) int {
	result := 0

	for _, item := range arr {
		if item {
			result += 1
		}
	}
	return result
}

func slotInEpoch(slot phase0.Slot, epoch phase0.Epoch) bool {
	if slot >= phase0.Slot(epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(epoch+1)*spec.SlotsPerEpoch {
		return true
	}
	return false
}
