package metrics

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

// Regression guard for https://github.com/migalabs/goteth/issues/271:
// when beacon committee data is missing (e.g. it failed to download), resolving
// a validator from a committee index must return an error instead of panicking
// with a nil/out-of-range slice access.
func TestGetValidatorFromCommitteeIndexMissingCommittee(t *testing.T) {
	base := StateMetricsBase{
		PrevState:    &spec.AgnosticState{Epoch: 0},
		CurrentState: &spec.AgnosticState{Epoch: 1}, // empty EpochStructs => no committees
		NextState:    &spec.AgnosticState{Epoch: 2},
	}
	m := AltairMetrics{}
	m.baseMetrics = base

	// A slot inside CurrentState's epoch, whose committee data is absent.
	slot := phase0.Slot(1*spec.SlotsPerEpoch + 1)

	if _, err := m.GetValidatorFromCommitteeIndex(slot, 0, 0); err == nil {
		t.Fatal("expected an error when the beacon committee is missing, got nil (would have panicked before the fix)")
	}
}

// GetValList must return nil (not panic) when the committee does not exist, which
// is the safe primitive the guards above rely on.
func TestGetValListMissingCommitteeReturnsNil(t *testing.T) {
	duties := spec.EpochDuties{} // no BeaconCommittees populated
	if got := duties.GetValList(phase0.Slot(33), 0); got != nil {
		t.Fatalf("expected nil validator list for a missing committee, got %v", got)
	}
}
