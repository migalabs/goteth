package analyzer

import (
	"fmt"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
)

// HandleReorg replaces the state at the end of an epoch while a processor may
// still be reading it, so it waits for that processor first. It walks slots,
// while ProcessStateTransitionMetrics registers epochs, and the conversion was
// left to the call site and not done: the barrier asked for the slot number
// under the epoch tag. No such key is ever in the book, so the wait returned
// immediately every time it ran (migalabs/goteth#292).

// The key the processor actually registers, built the way process_state.go
// builds it. If that ever changes, this test fails and says so.
func processorEpochKey(epoch uint64) string {
	return fmt.Sprintf("%s%d", epochProcesserTag, epoch)
}

func processorSlotKey(slot uint64) string {
	return fmt.Sprintf("%s%d", slotProcesserTag, slot)
}

func TestTheEpochBarrierAsksForTheKeyTheProcessorRegistered(t *testing.T) {
	const epoch uint64 = 471766
	lastSlot := phase0.Slot((epoch+1)*spec.SlotsPerEpoch - 1)

	got := epochBarrierKey(lastSlot)
	want := processorEpochKey(epoch)

	if got != want {
		t.Errorf("barrier waits on %q while the processor registered %q; "+
			"the wait matches nothing and the state is replaced under a live reader",
			got, want)
	}
}

// Every slot of an epoch must resolve to that epoch's key, not just the last
// one: the barrier runs at the end-of-epoch boundary but the conversion has to
// hold wherever it is called from.
func TestEverySlotOfAnEpochResolvesToItsEpochKey(t *testing.T) {
	const epoch uint64 = 100
	want := processorEpochKey(epoch)

	for offset := uint64(0); offset < spec.SlotsPerEpoch; offset++ {
		slot := phase0.Slot(epoch*spec.SlotsPerEpoch + offset)
		if got := epochBarrierKey(slot); got != want {
			t.Fatalf("slot %d gave %q, want %q", slot, got, want)
		}
	}
}

func TestTheSlotBarrierAsksForTheSlotKey(t *testing.T) {
	const slot uint64 = 15096543

	if got, want := slotBarrierKey(phase0.Slot(slot)), processorSlotKey(slot); got != want {
		t.Errorf("slot barrier waits on %q, want %q", got, want)
	}
}

// The two barriers must not collapse onto one another: a slot key and an epoch
// key are different namespaces, and waiting on the wrong one is the bug.
func TestTheTwoBarriersDoNotShareAKey(t *testing.T) {
	slot := phase0.Slot(15096543)

	if slotBarrierKey(slot) == epochBarrierKey(slot) {
		t.Error("the slot and epoch barriers built the same key")
	}
}

// End to end through the book: a processor registered for the epoch must be
// visible to the barrier that is meant to wait for it.
func TestTheBarrierSeesALiveProcessor(t *testing.T) {
	const epoch uint64 = 471766
	lastSlot := phase0.Slot((epoch+1)*spec.SlotsPerEpoch - 1)

	book := utils.NewRoutineBook(4, "test")
	book.Acquire(processorEpochKey(epoch))

	if !book.CheckPageActive(epochBarrierKey(lastSlot)) {
		t.Error("the barrier cannot see the processor holding that epoch, so it " +
			"would replace the state while the processor is still reading it")
	}
}
