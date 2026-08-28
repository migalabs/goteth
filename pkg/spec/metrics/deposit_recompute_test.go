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
func TestRunningTwiceOnOneStateDoublesTheCounters(t *testing.T) {
	p := &ElectraMetrics{}
	state := stateWithOnePendingDeposit()

	p.processPendingDepositsFor(state)
	once := state.DepositsNum
	p.processPendingDepositsFor(state)

	if state.DepositsNum != once*2 {
		t.Errorf("expected a second run to double %d to %d, got %d; if this no longer "+
			"holds, revisit whether PendingDepositsProcessed still needs to gate the call",
			once, once*2, state.DepositsNum)
	}
}
