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
