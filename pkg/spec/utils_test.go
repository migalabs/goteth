package spec_test

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

func TestEpochAtSlot(t *testing.T) {
	tests := []struct {
		name  string
		slot  phase0.Slot
		epoch phase0.Epoch
	}{
		{
			name:  "Genesis",
			slot:  0,
			epoch: 0,
		},
		{
			name:  "Slot 1",
			slot:  1,
			epoch: 0,
		},
		{
			name:  "Slot 32",
			slot:  32,
			epoch: 1,
		},
		{
			name:  "Slot 33",
			slot:  33,
			epoch: 1,
		},
		{
			name:  "Slot 64",
			slot:  64,
			epoch: 2,
		},
		{
			name:  "Slot 65",
			slot:  65,
			epoch: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			epoch := spec.EpochAtSlot(test.slot)
			if epoch != test.epoch {
				t.Errorf("EpochAtSlot() returned %d, expected %d", epoch, test.epoch)
			}
		})
	}

}

func TestFirstSlotInEpoch(t *testing.T) {
	tests := []struct {
		name  string
		slot  phase0.Slot
		first phase0.Slot
	}{
		{
			name:  "Genesis",
			slot:  0,
			first: 0,
		},
		{
			name:  "Slot 1",
			slot:  1,
			first: 0,
		},
		{
			name:  "Slot 32",
			slot:  32,
			first: 32,
		},
		{
			name:  "Slot 33",
			slot:  33,
			first: 32,
		},
		{
			name:  "Slot 64",
			slot:  64,
			first: 64,
		},
		{
			name:  "Slot 65",
			slot:  65,
			first: 64,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := spec.FirstSlotInEpoch(test.slot)
			if first != test.first {
				t.Errorf("FirstSlotInEpoch() returned %d, expected %d", first, test.first)
			}
		})
	}

}
