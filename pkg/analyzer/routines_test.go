package analyzer

import (
	"sync"
	"testing"
	"time"

	"github.com/migalabs/goteth/pkg/spec"
)

// flushRecorder is a thread-safe fake flush target for
// batchDataColumnSidecarEvents. It records call and row counts only; the
// batch slice itself is reused by the batcher and must not be retained.
type flushRecorder struct {
	mu    sync.Mutex
	calls int
	rows  int
}

func (r *flushRecorder) flush(batch []spec.DataColumnSidecarEventWrapper) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.rows += len(batch)
}

func (r *flushRecorder) snapshot() (calls int, rows int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls, r.rows
}

// waitFor polls cond until it is true or the timeout expires.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", msg)
}

type batcherHarness struct {
	events   chan spec.DataColumnSidecarEventWrapper
	tick     chan time.Time
	done     chan struct{}
	stopped  func() bool
	rec      *flushRecorder
	finished chan struct{}
}

func startBatcher(maxBatch int, stopped func() bool) *batcherHarness {
	h := &batcherHarness{
		events:   make(chan spec.DataColumnSidecarEventWrapper, 1024),
		tick:     make(chan time.Time),
		done:     make(chan struct{}),
		stopped:  stopped,
		rec:      &flushRecorder{},
		finished: make(chan struct{}),
	}
	go func() {
		batchDataColumnSidecarEvents(h.events, h.tick, h.done, h.stopped, maxBatch, h.rec.flush)
		close(h.finished)
	}()
	return h
}

func (h *batcherHarness) sendEvents(n int) {
	for i := 0; i < n; i++ {
		h.events <- spec.DataColumnSidecarEventWrapper{Timestamp: time.Now()}
	}
}

func (h *batcherHarness) waitFinished(t *testing.T) {
	t.Helper()
	select {
	case <-h.finished:
	case <-time.After(2 * time.Second):
		t.Fatal("batcher did not return")
	}
}

// A burst larger than maxBatch flushes in maxBatch-sized batches without any
// tick, and the tick then flushes the remainder: far fewer flushes than events.
func TestBatchDataColumnEventsFlushesAtMaxBatch(t *testing.T) {
	h := startBatcher(128, func() bool { return false })
	h.sendEvents(300)

	// The two full 128-event batches flush without any tick.
	waitFor(t, time.Second, func() bool { _, rows := h.rec.snapshot(); return rows == 256 },
		"two maxBatch flushes (256 rows)")

	// Wait until the loop has consumed the whole burst, then tick to flush the
	// remaining 44 (sending the tick earlier could reach the select while part
	// of the burst still sits in the channel, splitting the last flush).
	waitFor(t, time.Second, func() bool { return len(h.events) == 0 }, "channel drained")
	h.tick <- time.Time{}
	waitFor(t, time.Second, func() bool { _, rows := h.rec.snapshot(); return rows == 300 },
		"tick flush of the remainder (300 rows total)")

	calls, rows := h.rec.snapshot()
	if rows != 300 {
		t.Fatalf("expected 300 rows flushed, got %d", rows)
	}
	if calls != 3 {
		t.Fatalf("expected 3 flush calls (128+128+44), got %d", calls)
	}

	close(h.done)
	h.waitFinished(t)
}

// A partial buffer is flushed once per tick, not once per event.
func TestBatchDataColumnEventsTickFlushesPartialBuffer(t *testing.T) {
	h := startBatcher(128, func() bool { return false })
	h.sendEvents(5)

	// Once the channel is empty every sent event is either appended already or
	// being appended by the current loop iteration; the blocking tick send below
	// is only received after that iteration finishes.
	waitFor(t, time.Second, func() bool { return len(h.events) == 0 }, "channel drained")
	h.tick <- time.Time{}

	waitFor(t, time.Second, func() bool { calls, rows := h.rec.snapshot(); return calls == 1 && rows == 5 },
		"single flush of 5 rows")

	close(h.done)
	h.waitFinished(t)
}

// When stopped() reports true on a tick, pending channel events are drained and
// flushed before returning.
func TestBatchDataColumnEventsStopDrainsChannel(t *testing.T) {
	var mu sync.Mutex
	stop := false
	setStop := func(v bool) { mu.Lock(); stop = v; mu.Unlock() }
	getStop := func() bool { mu.Lock(); defer mu.Unlock(); return stop }

	h := startBatcher(128, getStop)
	h.sendEvents(10)
	setStop(true)
	h.tick <- time.Time{}

	h.waitFinished(t)
	_, rows := h.rec.snapshot()
	if rows != 10 {
		t.Fatalf("expected all 10 rows flushed on stop, got %d", rows)
	}
}

// When done fires, pending channel events are drained and flushed before
// returning.
func TestBatchDataColumnEventsDoneDrainsChannel(t *testing.T) {
	h := startBatcher(128, func() bool { return false })
	h.sendEvents(7)
	close(h.done)

	h.waitFinished(t)
	_, rows := h.rec.snapshot()
	if rows != 7 {
		t.Fatalf("expected all 7 rows flushed on done, got %d", rows)
	}
}
