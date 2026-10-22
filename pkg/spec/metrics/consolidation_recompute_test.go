package metrics

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
)

// The consolidation counters have the same shape as the deposit ones fixed in
// migalabs/goteth#287: ConsolidationsProcessed is appended to and
// ConsolidationsProcessedAmount accumulated with +=, both while the state is
// NextState, and both read from CurrentState an epoch later by standard.go. A
// state re-downloaded in between reports zero while t_consolidations holds the
// rows; a state reprocessed against a retained object reports double.

const consolidatedBalance = phase0.Gwei(32_000_000_000)

// eth1Credentials returns a 0x01 withdrawal credential. The prefix byte is read
// unconditionally by processExcessActiveBalanceRestructuring, and keeping both
// states on the same prefix means no switch-to-compounding is detected.
func eth1Credentials() []byte {
	credentials := make([]byte, 32)
	credentials[0] = spec.Eth1AddressWithdrawalPrefix
	return credentials
}

// stateWithOnePendingConsolidation builds the smallest state that processes a
// single consolidation: the source is unslashed and already withdrawable, so
// neither the skip nor the break branch is taken.
func stateWithOnePendingConsolidation() *spec.AgnosticState {
	return &spec.AgnosticState{
		Epoch: 10,
		Validators: []*phase0.Validator{
			{EffectiveBalance: consolidatedBalance, WithdrawableEpoch: 5, WithdrawalCredentials: eth1Credentials()},
			{EffectiveBalance: consolidatedBalance, WithdrawableEpoch: phase0.Epoch(spec.FarFutureEpoch), WithdrawalCredentials: eth1Credentials()},
		},
		Balances:                []phase0.Gwei{consolidatedBalance, consolidatedBalance},
		PendingConsolidations:   []*electra.PendingConsolidation{{SourceIndex: 0, TargetIndex: 1}},
		ConsolidationsProcessed: make([]spec.ConsolidationProcessed, 0),
		ConsolidatedAmounts:     make(map[phase0.ValidatorIndex]phase0.Gwei),
	}
}

// The fix rests on this: a state that lost its counters can have them derived
// again from the state alone, reaching the numbers the original run reached.
func TestRecomputingAReplacedStateReproducesTheConsolidationCounters(t *testing.T) {
	p := &ElectraMetrics{}

	accumulated := stateWithOnePendingConsolidation()
	p.processPendingConsolidationsFor(accumulated)

	// The same state as it comes back from the beacon node, counters at zero.
	redownloaded := stateWithOnePendingConsolidation()
	p.processPendingConsolidationsFor(redownloaded)

	if len(accumulated.ConsolidationsProcessed) == 0 {
		t.Fatal("the fixture processed no consolidations, so it cannot show anything")
	}
	if len(redownloaded.ConsolidationsProcessed) != len(accumulated.ConsolidationsProcessed) {
		t.Errorf("recomputed %d consolidations, the original run reached %d",
			len(redownloaded.ConsolidationsProcessed), len(accumulated.ConsolidationsProcessed))
	}
	if redownloaded.ConsolidationsProcessedAmount != accumulated.ConsolidationsProcessedAmount {
		t.Errorf("recomputed %d gwei, the original run reached %d",
			redownloaded.ConsolidationsProcessedAmount, accumulated.ConsolidationsProcessedAmount)
	}
}

func TestProcessingMarksTheStateConsolidated(t *testing.T) {
	p := &ElectraMetrics{}
	state := stateWithOnePendingConsolidation()

	p.processPendingConsolidationsFor(state)

	if !state.PendingConsolidationsProcessed {
		t.Error("the state was processed but not marked, so it would be recomputed again")
	}
}

// An epoch with an empty queue must still be marked, otherwise every epoch
// without consolidations, which is most of them, is recomputed on every pass.
func TestAnEmptyConsolidationQueueStillMarksTheState(t *testing.T) {
	p := &ElectraMetrics{}
	state := &spec.AgnosticState{Epoch: 10, ConsolidatedAmounts: make(map[phase0.ValidatorIndex]phase0.Gwei)}

	p.processPendingConsolidationsFor(state)

	if !state.PendingConsolidationsProcessed {
		t.Error("a state with no pending consolidations was left unmarked")
	}
	if len(state.ConsolidationsProcessed) != 0 {
		t.Errorf("an empty queue produced %d consolidations", len(state.ConsolidationsProcessed))
	}
}

// Why the flag has to gate the call: the counters accumulate with += and
// append, so a second run on the same object doubles them. This pins the
// hazard the flag exists to prevent.
func TestRunningConsolidationsTwiceLeavesTheCountersUnchanged(t *testing.T) {
	p := &ElectraMetrics{}
	state := stateWithOnePendingConsolidation()

	p.processPendingConsolidationsFor(state)
	num, amount := len(state.ConsolidationsProcessed), state.ConsolidationsProcessedAmount

	p.processPendingConsolidationsFor(state)

	if len(state.ConsolidationsProcessed) != num {
		t.Errorf("ConsolidationsProcessed went from %d to %d entries on a second run",
			num, len(state.ConsolidationsProcessed))
	}
	if state.ConsolidationsProcessedAmount != amount {
		t.Errorf("ConsolidationsProcessedAmount went from %d to %d on a second run",
			amount, state.ConsolidationsProcessedAmount)
	}
}

func TestPerValidatorConsolidatedAmountsDoNotDoubleOnASecondRun(t *testing.T) {
	// ConsolidatedAmounts is accumulated per validator with += as well. It is
	// rebuilt from scratch by processConsolidationsForRewardCalculation on the
	// PreProcessBundle path, but this function is what populates it on the
	// NextState path, so it needs its own assertion.
	p := &ElectraMetrics{}
	state := stateWithOnePendingConsolidation()

	p.processPendingConsolidationsFor(state)
	before := make(map[phase0.ValidatorIndex]phase0.Gwei, len(state.ConsolidatedAmounts))
	for idx, amount := range state.ConsolidatedAmounts {
		before[idx] = amount
	}
	if len(before) == 0 {
		t.Fatal("no per-validator amounts were recorded; the fixture no longer exercises this")
	}

	p.processPendingConsolidationsFor(state)

	for idx, amount := range before {
		if state.ConsolidatedAmounts[idx] != amount {
			t.Errorf("validator %d went from %d to %d on a second run", idx, amount, state.ConsolidatedAmounts[idx])
		}
	}
}

func TestTheNextStateConsolidationPathIsGuardedToo(t *testing.T) {
	// processPendingConsolidations runs on every ProcessStateTransitionMetrics
	// call, so an epoch reprocessed against a state still held in cache - a
	// dependency pass in AdvanceFinalized, a carried reprocess - went through
	// this path a second time and doubled the counters.
	state := stateWithOnePendingConsolidation()
	p := &ElectraMetrics{}
	p.baseMetrics.NextState = state

	p.processPendingConsolidations()
	amount := state.ConsolidationsProcessedAmount

	p.processPendingConsolidations()

	if state.ConsolidationsProcessedAmount != amount {
		t.Errorf("reprocessing through the NextState path doubled %d to %d", amount, state.ConsolidationsProcessedAmount)
	}
}

func TestRefreshingAStateAllowsConsolidationsToBeCountedAgain(t *testing.T) {
	// Idempotence must not become "counted once, ever". RefreshBlocks zeroes
	// the counters, so it clears the flag, and the next pass has to refill them
	// or the epoch row reports no consolidations.
	p := &ElectraMetrics{}
	state := stateWithOnePendingConsolidation()

	p.processPendingConsolidationsFor(state)
	amount := state.ConsolidationsProcessedAmount
	if amount == 0 {
		t.Fatal("fixture processed no consolidations")
	}

	state.RefreshBlocks(nil)
	if state.PendingConsolidationsProcessed {
		t.Fatal("RefreshBlocks left the state marked as already accumulated")
	}

	p.processPendingConsolidationsFor(state)
	if state.ConsolidationsProcessedAmount != amount {
		t.Errorf("after a refresh the counters came back as %d, want %d", state.ConsolidationsProcessedAmount, amount)
	}
}

func TestARefreshClearsTheConsolidationEntries(t *testing.T) {
	// RefreshBlocks promises to reset the accumulators. It zeroed the deposit
	// counters but left ConsolidationsProcessed and ConsolidationsProcessedAmount
	// populated, so a recompute after a refresh appended a second set of rows on
	// top of the first.
	p := &ElectraMetrics{}
	state := stateWithOnePendingConsolidation()

	p.processPendingConsolidationsFor(state)
	num, amount := len(state.ConsolidationsProcessed), state.ConsolidationsProcessedAmount
	if num == 0 {
		t.Fatal("fixture processed no consolidations")
	}

	state.RefreshBlocks(nil)
	p.processPendingConsolidationsFor(state)

	if len(state.ConsolidationsProcessed) != num {
		t.Errorf("after a refresh and recompute there are %d entries, want %d", len(state.ConsolidationsProcessed), num)
	}
	if state.ConsolidationsProcessedAmount != amount {
		t.Errorf("after a refresh and recompute the amount is %d, want %d", state.ConsolidationsProcessedAmount, amount)
	}
}

// The tests above pin the function's own behaviour. This one pins the wiring:
// that PreProcessBundle recomputes against CurrentState, which is the object
// standard.go reads the epoch row from. Without that call site the function is
// correct and the epoch row is still written with zeros.
func TestPreProcessBundleFillsAReDownloadedCurrentState(t *testing.T) {
	// CurrentState as it comes back from the beacon node after a reorg: the
	// pending queue is there, the counters accumulated an epoch earlier are not.
	currentState := stateWithOnePendingConsolidation()
	currentState.StateRoot = phase0.Root{0x01} // not empty, or the bundle is a no-op
	currentState.Blocks = nil                  // short-circuits ProcessAttestations
	currentState.DepositedAmounts = make(map[phase0.ValidatorIndex]phase0.Gwei)
	currentState.ConsolidatedOutAmounts = make(map[phase0.ValidatorIndex]phase0.Gwei)

	nextState := stateWithOnePendingConsolidation()
	nextState.Epoch = currentState.Epoch + 1
	nextState.ConsolidatedOutAmounts = make(map[phase0.ValidatorIndex]phase0.Gwei)
	nextState.DepositedAmounts = make(map[phase0.ValidatorIndex]phase0.Gwei)

	p := &ElectraMetrics{}
	p.baseMetrics.CurrentState = currentState
	p.baseMetrics.NextState = nextState
	// PrevState carries an empty state root, so the block-reward tail of the
	// bundle, which needs a far larger fixture, does not run. It still has to be
	// non-nil: EmptyStateRoot has a value receiver.
	p.baseMetrics.PrevState = &spec.AgnosticState{}

	if err := p.PreProcessBundle(); err != nil {
		t.Fatalf("PreProcessBundle: %v", err)
	}

	if currentState.ConsolidationsProcessedAmount == 0 {
		t.Error("CurrentState came out of the bundle with no consolidated amount; " +
			"the epoch row would be written as zero while t_consolidations holds the rows")
	}
	if len(currentState.ConsolidationsProcessed) == 0 {
		t.Error("CurrentState came out of the bundle with no consolidation entries")
	}
}
