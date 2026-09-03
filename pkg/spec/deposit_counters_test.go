package spec

import "testing"

// The deposit counters on a state are accumulated while it is NextState and
// read one epoch later, while it is CurrentState. Anything that replaces or
// resets the object in between leaves them at zero, and the epoch row is then
// written with no deposits while t_deposits holds the rows
// (migalabs/goteth#287).

func TestRefreshBlocksClearsTheDepositAccumulationFlag(t *testing.T) {
	state := &AgnosticState{
		DepositsNum:              4,
		TotalDepositsAmount:      128_000_000_000,
		PendingDepositsProcessed: true,
	}

	state.RefreshBlocks(nil)

	if state.DepositsNum != 0 {
		t.Fatalf("RefreshBlocks left DepositsNum at %d; the fixture assumes it resets",
			state.DepositsNum)
	}
	if state.PendingDepositsProcessed {
		t.Error("the counters were reset but the state still claims they were accumulated, " +
			"so nothing would recompute them and the epoch row would report zero deposits")
	}
}

// A freshly downloaded state has never been accumulated, which is what marks it
// for recomputation rather than being taken at face value.
func TestAFreshStateIsNotMarkedAccumulated(t *testing.T) {
	if (&AgnosticState{}).PendingDepositsProcessed {
		t.Error("a new state claims its deposit counters were accumulated")
	}
}

// Zero deposits is a legitimate outcome, and must stay distinguishable from
// "never counted": most epochs genuinely have none.
func TestZeroDepositsIsDistinctFromNeverCounted(t *testing.T) {
	counted := &AgnosticState{DepositsNum: 0, PendingDepositsProcessed: true}
	never := &AgnosticState{DepositsNum: 0}

	if !counted.PendingDepositsProcessed {
		t.Error("an epoch with genuinely no deposits should not be recomputed")
	}
	if never.PendingDepositsProcessed {
		t.Error("a state that was never counted must be recomputed")
	}
}
