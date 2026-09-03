package db

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// Processing epoch E writes the epoch row for E-1, which is what
// DeleteStateMetrics clears before the rewrite. Epoch 0 is the exception:
// there is no epoch before genesis.

func TestProcessingAnEpochWritesThePreviousEpochsRow(t *testing.T) {
	row, ok := EpochRowWrittenBy(10)

	if !ok {
		t.Fatal("processing epoch 10 should write an epoch row")
	}
	if row != 9 {
		t.Errorf("epoch row = %d, want 9", row)
	}
}

func TestGenesisWritesNoEpochRow(t *testing.T) {
	// phase0.Epoch is unsigned, so 0-1 wraps to the maximum uint64 and would
	// address a row that cannot exist. Nothing matches it today, which is why
	// this never surfaced, but that is not a property to rely on.
	row, ok := EpochRowWrittenBy(0)

	if ok {
		t.Errorf("processing epoch 0 claimed to write epoch row %d", row)
	}
}

func TestNoEpochEverAddressesTheWrappedRow(t *testing.T) {
	// Written through a variable on purpose. The compiler rejects the constant
	// form as an overflow, which is exactly why this went unnoticed in the
	// original code: there the epoch is a variable, so it wraps silently.
	var zero phase0.Epoch
	wrapped := zero - 1

	for _, epoch := range []phase0.Epoch{0, 1, 2, 100, 471766} {
		row, ok := EpochRowWrittenBy(epoch)
		if ok && row == wrapped {
			t.Errorf("epoch %d addresses the wrapped row %d", epoch, row)
		}
	}
}
