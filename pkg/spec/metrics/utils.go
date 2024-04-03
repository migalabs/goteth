package metrics

import (
	"errors"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

func (p AltairMetrics) GetValidatorFromCommitteeIndex(slot phase0.Slot, committeeIndex phase0.CommitteeIndex, idx int) (phase0.ValidatorIndex, error) {
	if slot >= phase0.Slot(p.baseMetrics.PrevState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(p.baseMetrics.CurrentState.Epoch)*spec.SlotsPerEpoch {
		// slot in PrevEpoch
		valList := p.baseMetrics.PrevState.EpochStructs.GetValList(slot, committeeIndex)
		return valList[idx], nil
	}

	if slot >= phase0.Slot(p.baseMetrics.CurrentState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(p.baseMetrics.NextState.Epoch)*spec.SlotsPerEpoch {
		// slot in CurrentEpoch
		valList := p.baseMetrics.CurrentState.EpochStructs.GetValList(slot, committeeIndex)
		return valList[idx], nil
	}

	if slot >= phase0.Slot(p.baseMetrics.NextState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(p.baseMetrics.NextState.Epoch+1)*spec.SlotsPerEpoch {
		// slot in NextEpoch
		valList := p.baseMetrics.NextState.EpochStructs.GetValList(slot, committeeIndex)
		return valList[idx], nil
	}

	return 0, fmt.Errorf("could not get validator from any epoch: slot %d, committee %d, index %d", slot, committeeIndex, idx)
}

func (p AltairMetrics) GetJustifiedRootfromSlot(slot phase0.Slot) (phase0.Root, error) {
	if slot >= phase0.Slot(p.baseMetrics.PrevState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(p.baseMetrics.CurrentState.Epoch)*spec.SlotsPerEpoch {
		// slot in PrevEpoch
		return p.baseMetrics.PrevState.CurrentJustifiedCheckpoint.Root, nil
	}

	if slot >= phase0.Slot(p.baseMetrics.CurrentState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(p.baseMetrics.NextState.Epoch)*spec.SlotsPerEpoch {
		// slot in CurrentEpochEpoch
		return p.baseMetrics.CurrentState.CurrentJustifiedCheckpoint.Root, nil
	}

	if slot >= phase0.Slot(p.baseMetrics.NextState.Epoch)*spec.SlotsPerEpoch &&
		slot < phase0.Slot(p.baseMetrics.NextState.Epoch+1)*spec.SlotsPerEpoch {
		// slot in NextEpoch
		return p.baseMetrics.NextState.CurrentJustifiedCheckpoint.Root, nil
	}

	return phase0.Root{}, errors.New("could not get justified checkpoint from any epoch")

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
