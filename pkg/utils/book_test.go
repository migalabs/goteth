package utils

import (
	"sync"
	"testing"
	"time"
)

// Acquire looks like a per-key lock but is a counting semaphore that records
// what holds each slot. Two goroutines can acquire the same key, and each takes
// its own token. While a page was a bare marker the second acquire overwrote
// the first, so only one FreePage found a key to delete and one token was never
// returned: the pool shrank by one every time, until nothing could acquire at
// all (migalabs/goteth#292).

func TestAcquiringOneKeyTwiceReturnsBothTokens(t *testing.T) {
	book := NewRoutineBook(4, "test")

	book.Acquire("epoch=471766")
	book.Acquire("epoch=471766")
	if free := book.NumFreePages(); free != 2 {
		t.Fatalf("two acquires left %d of 4 tokens free, want 2", free)
	}

	book.FreePage("epoch=471766")
	book.FreePage("epoch=471766")

	if free := book.NumFreePages(); free != 4 {
		t.Errorf("after releasing both holders %d of 4 tokens are free, want 4; "+
			"the missing one is gone for the life of the process", free)
	}
}

// The consequence, stated as the pool sees it: repeat the leak and the book
// runs out of slots even though nothing is holding any.
func TestRepeatedDuplicateAcquiresDoNotExhaustThePool(t *testing.T) {
	book := NewRoutineBook(2, "test")

	for i := 0; i < 10; i++ {
		book.Acquire("epoch=100")
		book.Acquire("epoch=100")
		book.FreePage("epoch=100")
		book.FreePage("epoch=100")
	}

	if free := book.NumFreePages(); free != 2 {
		t.Errorf("after 10 paired acquire/free rounds %d of 2 tokens are free; "+
			"the pool is exhausted and the next Acquire blocks forever", free)
	}
	if active := book.ActivePages(); active != 0 {
		t.Errorf("book reports %d active holders with none outstanding", active)
	}
}

// A key stays active while any holder is still working, or a reorg would
// replace a state out from under a processor that is still reading it.
func TestAKeyStaysActiveUntilTheLastHolderReleasesIt(t *testing.T) {
	book := NewRoutineBook(4, "test")

	book.Acquire("epoch=10")
	book.Acquire("epoch=10")

	book.FreePage("epoch=10")
	if !book.CheckPageActive("epoch=10") {
		t.Error("the key went inactive while a second holder was still working")
	}

	book.FreePage("epoch=10")
	if book.CheckPageActive("epoch=10") {
		t.Error("the key stayed active after its last holder released it")
	}
}

// Freeing a key nobody holds must not push a token into the channel: the pool
// would grow past its size, and once the channel is full that send blocks
// forever.
func TestFreeingAKeyNobodyHoldsDoesNotInventTokens(t *testing.T) {
	book := NewRoutineBook(2, "test")

	done := make(chan struct{})
	go func() {
		defer close(done)
		book.FreePage("never-acquired")
		book.FreePage("never-acquired")
		book.FreePage("never-acquired")
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("FreePage blocked on a full channel; the pool was over-filled")
	}

	if free := book.NumFreePages(); free != 2 {
		t.Errorf("pool holds %d tokens, want 2", free)
	}
}

func TestActivePagesCountsHoldersNotKeys(t *testing.T) {
	// The shutdown check asks "is anything still running", so two routines on
	// one key must count twice.
	book := NewRoutineBook(4, "test")

	book.Acquire("epoch=10")
	book.Acquire("epoch=10")
	book.Acquire("epoch=11")

	if active := book.ActivePages(); active != 3 {
		t.Errorf("ActivePages reported %d, want 3; a shutdown check would think "+
			"fewer routines were running than there are", active)
	}
}

func TestConcurrentAcquireAndFreeKeepsTheAccountingExact(t *testing.T) {
	book := NewRoutineBook(8, "test")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			book.Acquire("epoch=1")
			book.FreePage("epoch=1")
		}()
	}
	wg.Wait()

	if free := book.NumFreePages(); free != 8 {
		t.Errorf("after 50 paired acquire/free rounds %d of 8 tokens are free", free)
	}
	if active := book.ActivePages(); active != 0 {
		t.Errorf("%d holders still recorded with none outstanding", active)
	}
}
