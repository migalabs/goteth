package metrics

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

// The deposit counters are accumulated while a state is NextState and read an
// epoch later, while it is CurrentState. A state that was re-downloaded or
// refreshed in between carries zeros, and the epoch row is then written with no
// deposits while t_deposits holds the rows (migalabs/goteth#287).

const depositAmount = phase0.Gwei(32_000_000_000)

// stateWithOnePendingDeposit builds the smallest state that processes a single
// deposit: the queue is finalized, the Eth1 bridge check is satisfied, and the
// balance available for processing covers the amount.
func stateWithOnePendingDeposit() *spec.AgnosticState {
	var pubkey phase0.BLSPubKey
	pubkey[0] = 0x01

	return &spec.AgnosticState{
		Epoch:                     10,
		Validators:                []*phase0.Validator{{PublicKey: pubkey, ExitEpoch: phase0.Epoch(spec.FarFutureEpoch), WithdrawableEpoch: phase0.Epoch(spec.FarFutureEpoch)}},
		PendingDeposits:           []*electra.PendingDeposit{{Pubkey: pubkey, Amount: depositAmount, Slot: 0}},
		DepositBalanceToConsume:   depositAmount * 4,
		Eth1DepositIndex:          1,
		DepositRequestsStartIndex: 0,
		DepositedAmounts:          make(map[phase0.ValidatorIndex]phase0.Gwei),
	}
}

// The fix rests on this: a state that lost its counters can have them derived
// again from the state alone, reaching the numbers the original run reached.
func TestRecomputingAReplacedStateReproducesTheCounters(t *testing.T) {
	p := &ElectraMetrics{}

	accumulated := stateWithOnePendingDeposit()
	p.processPendingDepositsFor(accumulated)

	// The same state as it comes back from the beacon node, counters at zero.
	redownloaded := stateWithOnePendingDeposit()
	p.processPendingDepositsFor(redownloaded)

	if accumulated.DepositsNum == 0 {
		t.Fatal("the fixture processed no deposits, so it cannot show anything")
	}
	if redownloaded.DepositsNum != accumulated.DepositsNum {
		t.Errorf("recomputed %d deposits, the original run reached %d",
			redownloaded.DepositsNum, accumulated.DepositsNum)
	}
	if redownloaded.TotalDepositsAmount != accumulated.TotalDepositsAmount {
		t.Errorf("recomputed %d gwei, the original run reached %d",
			redownloaded.TotalDepositsAmount, accumulated.TotalDepositsAmount)
	}
}

func TestProcessingMarksTheStateAccumulated(t *testing.T) {
	p := &ElectraMetrics{}
	state := stateWithOnePendingDeposit()

	p.processPendingDepositsFor(state)

	if !state.PendingDepositsProcessed {
		t.Error("the state was processed but not marked, so it would be recomputed again")
	}
}

// An epoch with an empty queue must still be marked, otherwise every epoch
// without deposits, which is most of them, is recomputed on every pass.
func TestAnEmptyQueueStillMarksTheState(t *testing.T) {
	p := &ElectraMetrics{}
	state := &spec.AgnosticState{Epoch: 10, DepositedAmounts: make(map[phase0.ValidatorIndex]phase0.Gwei)}

	p.processPendingDepositsFor(state)

	if !state.PendingDepositsProcessed {
		t.Error("a state with no pending deposits was left unmarked")
	}
	if state.DepositsNum != 0 {
		t.Errorf("an empty queue produced %d deposits", state.DepositsNum)
	}
}

// Why PendingDepositsProcessed has to gate the call: the counters accumulate
// with +=, so a second run on the same object doubles them. This pins the
// hazard the flag exists to prevent.
func TestRunningTwiceOnOneStateLeavesTheCountersUnchanged(t *testing.T) {
	// The counters accumulate with +=, so a second pass over the same state
	// object would double them. The same object legitimately reaches this
	// function twice - once as NextState during its own epoch, once as
	// CurrentState an epoch later - so idempotence is what makes the second
	// call safe rather than a guard at one call site.
	p := &ElectraMetrics{}
	state := stateWithOnePendingDeposit()

	p.processPendingDepositsFor(state)
	num, amount, deposits := state.DepositsNum, state.TotalDepositsAmount, len(state.DepositsProcessed)

	p.processPendingDepositsFor(state)

	if state.DepositsNum != num {
		t.Errorf("DepositsNum went from %d to %d on a second run", num, state.DepositsNum)
	}
	if state.TotalDepositsAmount != amount {
		t.Errorf("TotalDepositsAmount went from %d to %d on a second run", amount, state.TotalDepositsAmount)
	}
	if len(state.DepositsProcessed) != deposits {
		t.Errorf("DepositsProcessed went from %d to %d entries on a second run", deposits, len(state.DepositsProcessed))
	}
}

func TestPerValidatorAmountsDoNotDoubleOnASecondRun(t *testing.T) {
	// DepositedAmounts is accumulated per validator with += as well, and it
	// feeds the per-validator deposit rows rather than the epoch counters, so
	// it needs its own assertion.
	p := &ElectraMetrics{}
	state := stateWithOnePendingDeposit()

	p.processPendingDepositsFor(state)
	before := make(map[phase0.ValidatorIndex]phase0.Gwei, len(state.DepositedAmounts))
	for idx, amount := range state.DepositedAmounts {
		before[idx] = amount
	}
	if len(before) == 0 {
		t.Fatal("no per-validator amounts were recorded; the fixture no longer exercises this")
	}

	p.processPendingDepositsFor(state)

	for idx, amount := range before {
		if state.DepositedAmounts[idx] != amount {
			t.Errorf("validator %d went from %d to %d on a second run", idx, amount, state.DepositedAmounts[idx])
		}
	}
}

func TestTheNextStatePathIsGuardedToo(t *testing.T) {
	// The failure the guard-at-one-call-site version missed: processPendingDeposits
	// runs unconditionally on every ProcessStateTransitionMetrics call, so an
	// epoch reprocessed against a state still held in cache - a dependency pass
	// in AdvanceFinalized, a carried reprocess, a refresh that failed - went
	// through this path a second time and doubled the counters.
	state := stateWithOnePendingDeposit()
	p := &ElectraMetrics{}
	p.baseMetrics.NextState = state

	p.processPendingDeposits()
	num := state.DepositsNum

	p.processPendingDeposits()

	if state.DepositsNum != num {
		t.Errorf("reprocessing through the NextState path doubled %d to %d", num, state.DepositsNum)
	}
}

func TestRefreshingAStateAllowsItToBeCountedAgain(t *testing.T) {
	// Idempotence must not become "counted once, ever". RefreshBlocks zeroes
	// the counters, so it clears the flag, and the next pass has to refill them
	// or the epoch row reports no deposits (#287 in the other direction).
	p := &ElectraMetrics{}
	state := stateWithOnePendingDeposit()

	p.processPendingDepositsFor(state)
	num := state.DepositsNum
	if num == 0 {
		t.Fatal("fixture processed no deposits")
	}

	state.RefreshBlocks(nil)
	if state.PendingDepositsProcessed {
		t.Fatal("RefreshBlocks left the state marked as already accumulated")
	}

	p.processPendingDepositsFor(state)
	if state.DepositsNum != num {
		t.Errorf("after a refresh the counters came back as %d, want %d", state.DepositsNum, num)
	}
}

func TestARefreshResetsThePerValidatorAmountsToo(t *testing.T) {
	// RefreshBlocks promises to reset the accumulators so that recalculating
	// does not double-count. DepositedAmounts accumulates with += and was left
	// populated, so the scalar counters came back correct while the
	// per-validator amounts came back doubled.
	p := &ElectraMetrics{}
	state := stateWithOnePendingDeposit()

	p.processPendingDepositsFor(state)
	want := make(map[phase0.ValidatorIndex]phase0.Gwei, len(state.DepositedAmounts))
	for idx, amount := range state.DepositedAmounts {
		want[idx] = amount
	}
	if len(want) == 0 {
		t.Fatal("no per-validator amounts were recorded; the fixture no longer exercises this")
	}

	state.RefreshBlocks(nil)
	p.processPendingDepositsFor(state)

	for idx, amount := range want {
		if state.DepositedAmounts[idx] != amount {
			t.Errorf("validator %d: %d after a refresh and recompute, want %d",
				idx, state.DepositedAmounts[idx], amount)
		}
	}
}
