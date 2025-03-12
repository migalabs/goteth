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

func TestHexStringAddressIsValid(t *testing.T) {
	tests := []struct {
		name    string
		address string
		valid   bool
	}{
		{
			name:    "Empty",
			address: "",
			valid:   false,
		},
		{
			name:    "Short",
			address: "0x123",
			valid:   false,
		},
		{
			name:    "Long",
			address: "0x12345678901234567890123456789012345678901",
			valid:   false,
		},
		{
			name:    "Invalid",
			address: "0x123456789012345678901234567890123456790g",
			valid:   false,
		},
		{
			name:    "Valid (Mainnet)",
			address: "0x00000000219ab540356cBB839Cbe05303d7705Fa",
			valid:   true,
		},
		{
			name:    "Valid (Sepolia)",
			address: "0x7f02C3E3c98b133055B8B348B2Ac625669Ed295D",
			valid:   true,
		},
		{
			name:    "Valid (Holesky)",
			address: "0x4242424242424242424242424242424242424242",
			valid:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid := spec.HexStringAddressIsValid(test.address)
			if valid != test.valid {
				t.Errorf("HexStringAddressIsValid() returned %v, expected %v", valid, test.valid)
			}
		})
	}
}

func TestUint64Max(t *testing.T) {
	tests := []struct {
		name string
		a    uint64
		b    uint64
		max  uint64
	}{
		{
			name: "A greater than B",
			a:    10,
			b:    5,
			max:  10,
		},
		{
			name: "B greater than A",
			a:    5,
			b:    10,
			max:  10,
		},
		{
			name: "A equal to B",
			a:    5,
			b:    5,
			max:  5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			max := spec.Uint64Max(test.a, test.b)
			if max != test.max {
				t.Errorf("Uint64Max() returned %d, expected %d", max, test.max)
			}
		})
	}

}

func TestUint64Min(t *testing.T) {
	tests := []struct {
		name string
		a    uint64
		b    uint64
		min  uint64
	}{
		{
			name: "A greater than B",
			a:    10,
			b:    5,
			min:  5,
		},
		{
			name: "B greater than A",
			a:    5,
			b:    10,
			min:  5,
		},
		{
			name: "A equal to B",
			a:    5,
			b:    5,
			min:  5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			min := spec.Uint64Min(test.a, test.b)
			if min != test.min {
				t.Errorf("Uint64Min() returned %d, expected %d", min, test.min)
			}
		})
	}

}
