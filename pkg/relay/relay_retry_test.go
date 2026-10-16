package relay

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	v1 "github.com/attestantio/go-relay-client/api/v1"
)

// A relay's delivered bids are fetched once per epoch and nothing fetches them
// again, so one failed request used to lose the epoch's bids from that relay.

// scriptedRelay answers each request with the next error of its script, nil
// meaning a successful answer, and repeats the last one when the script runs out.
type scriptedRelay struct {
	script []error
	calls  int
	empty  bool // answer successfully with no bids, as the client does for []
}

func (s *scriptedRelay) Name() string              { return "scripted" }
func (s *scriptedRelay) Address() string           { return "https://scripted.relay" }
func (s *scriptedRelay) Pubkey() *phase0.BLSPubKey { return nil }

func (s *scriptedRelay) DeliveredBidTrace(context.Context, phase0.Slot) (*v1.BidTrace, error) {
	return nil, errors.New("not used")
}

func (s *scriptedRelay) DeliveredBulkBidTrace(ctx context.Context, slot phase0.Slot, limit int) ([]*v1.BidTrace, error) {
	err := s.script[min(s.calls, len(s.script)-1)]
	s.calls++
	if err != nil || s.empty {
		return nil, err
	}
	return []*v1.BidTrace{{Slot: slot}}, nil
}

// scriptedClient is a relay client over a scripted relay. The waits between
// attempts are recorded instead of slept; onWait, if set, runs in each.
func scriptedClient(t *testing.T, ctx context.Context, onWait func(), script ...error) (*RelayClient, *scriptedRelay, *[]time.Duration) {
	t.Helper()
	var waits []time.Duration
	wait := waitBeforeRetry
	waitBeforeRetry = func(_ context.Context, d time.Duration) {
		waits = append(waits, d)
		if onWait != nil {
			onWait()
		}
	}
	t.Cleanup(func() { waitBeforeRetry = wait })
	relay := &scriptedRelay{script: script}
	return &RelayClient{ctx: ctx, client: relay, address: relay.Address()}, relay, &waits
}

var errRelay = errors.New("context deadline exceeded")

func TestATransientFailureIsRetried(t *testing.T) {
	r, relay, waits := scriptedClient(t, context.Background(), nil, errRelay, errRelay, nil)

	bids, err := r.GetDeliveredBidsPerSlotRange(15373855, 32)

	if err != nil || len(bids) != 1 {
		t.Fatalf("bids = %v, err = %v, want the third attempt's answer", bids, err)
	}
	if relay.calls != 3 {
		t.Errorf("requests = %d, want 3", relay.calls)
	}
	if len(*waits) != 2 || (*waits)[0] != relayRetryBackoff || (*waits)[1] != 2*relayRetryBackoff {
		t.Errorf("waits = %v, want %v then twice that", *waits, relayRetryBackoff)
	}
	if r.failing() {
		t.Error("the breaker counts a failure for a request that succeeded on a retry")
	}
}

// A relay that fails every attempt is one failure for the breaker, and the
// error keeps the message operators grep for.
func TestARequestFailingEveryAttemptIsOneFailure(t *testing.T) {
	r, relay, _ := scriptedClient(t, context.Background(), nil, errRelay)

	_, err := r.GetDeliveredBidsPerSlotRange(15373855, 32)

	if err == nil || !strings.Contains(err.Error(), "error obtaining delivered bid trace from https://scripted.relay: context deadline exceeded") {
		t.Fatalf("err = %v, want the delivered bid trace error", err)
	}
	if relay.calls != relayRequestAttempts {
		t.Errorf("requests = %d, want %d", relay.calls, relayRequestAttempts)
	}
	if r.consecutiveFail != 1 {
		t.Errorf("consecutive failures = %d, want 1", r.consecutiveFail)
	}
}

// The client turns an empty list into no bids and no error: the relay
// delivered nothing up to the cursor. That is an answer, not a failure.
func TestAnEmptyAnswerIsAnAnswer(t *testing.T) {
	r, relay, _ := scriptedClient(t, context.Background(), nil, nil)
	relay.empty = true

	bids, err := r.GetDeliveredBidsPerSlotRange(15373855, 32)

	if err != nil || bids != nil {
		t.Errorf("bids = %v, err = %v, want neither", bids, err)
	}
	if relay.calls != 1 || r.failing() {
		t.Errorf("%d requests, failing %v: want one request and no failure", relay.calls, r.failing())
	}
}

// A relay whose last request failed gets one attempt per request, so a relay
// that is down does not hold every epoch up for the retries.
func TestARelayAlreadyFailingGetsOneAttempt(t *testing.T) {
	r, relay, _ := scriptedClient(t, context.Background(), nil, errRelay)
	r.consecutiveFail = 1

	if _, err := r.GetDeliveredBidsPerSlotRange(15373855, 32); err == nil {
		t.Fatal("want an error")
	}
	if relay.calls != 1 {
		t.Errorf("requests = %d, want 1", relay.calls)
	}
}

// The probe after the breaker's cooldown is a request of a relay known to be
// failing: one attempt, and the breaker opens again if it fails.
func TestTheHalfOpenProbeGetsOneAttempt(t *testing.T) {
	r, relay, _ := scriptedClient(t, context.Background(), nil, errRelay)
	r.consecutiveFail = cbFailureThreshold
	r.openUntil = time.Now().Add(-time.Second)

	if _, err := r.GetDeliveredBidsPerSlotRange(15373855, 32); err == nil || strings.Contains(err.Error(), "circuit breaker open") {
		t.Fatalf("err = %v, want the probe's own failure", err)
	}
	if relay.calls != 1 {
		t.Errorf("requests = %d, want 1", relay.calls)
	}
	if !r.isOpen() {
		t.Error("a failed probe leaves the breaker closed")
	}
}

// Three failed requests still open the breaker, and an open breaker sends none.
func TestThreeFailedRequestsStillOpenTheBreaker(t *testing.T) {
	r, relay, _ := scriptedClient(t, context.Background(), nil, errRelay)

	for i := 0; i < cbFailureThreshold; i++ {
		r.GetDeliveredBidsPerSlotRange(15373855, 32)
	}
	sent := relay.calls
	_, err := r.GetDeliveredBidsPerSlotRange(15373855, 32)

	if err == nil || !strings.Contains(err.Error(), "circuit breaker open") {
		t.Fatalf("err = %v, want the breaker open", err)
	}
	if relay.calls != sent {
		t.Errorf("an open breaker sent %d requests", relay.calls-sent)
	}
}

// Shutting down stops the retries, whether it comes before the request or
// during the wait for the next attempt.
func TestShuttingDownStopsTheRetries(t *testing.T) {
	t.Run("before the request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r, relay, waits := scriptedClient(t, ctx, nil, errRelay)
		if _, err := r.GetDeliveredBidsPerSlotRange(15373855, 32); err == nil {
			t.Fatal("want an error")
		}
		if relay.calls != 1 || len(*waits) != 0 {
			t.Errorf("%d requests and %d waits, want one request and no wait", relay.calls, len(*waits))
		}
	})
	t.Run("during the wait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		r, relay, _ := scriptedClient(t, ctx, cancel, errRelay)
		if _, err := r.GetDeliveredBidsPerSlotRange(15373855, 32); err == nil {
			t.Fatal("want an error")
		}
		if relay.calls != 1 {
			t.Errorf("requests = %d, want 1", relay.calls)
		}
	})
}

// A relay that failed before and answers now is healthy again: the next request
// gets its retries back.
func TestAnAnswerAfterAFailureClearsIt(t *testing.T) {
	r, _, _ := scriptedClient(t, context.Background(), nil, nil)
	r.consecutiveFail = 1

	if _, err := r.GetDeliveredBidsPerSlotRange(15373855, 32); err != nil {
		t.Fatal(err)
	}
	if r.failing() {
		t.Error("the relay still counts as failing after a successful request")
	}
}
