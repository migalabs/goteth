package clientapi

import (
	"errors"
	"fmt"
	"net"
	nethttp "net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/attestantio/go-eth2-client/api"
)

// apiError builds the error go-eth2-client returns for a non-2xx response.
// It constructs the pointer form because that is what http.go returns.
func apiError(status int, body string) error {
	return &api.Error{
		Method:     nethttp.MethodGet,
		StatusCode: status,
		Endpoint:   "/eth/v2/beacon/blocks/12345",
		Data:       []byte(body),
	}
}

// transportError builds the error go-eth2-client returns when the request
// never reached the node: it joins its own note to the error from net/http,
// which is a *url.Error carrying the full request URL. The slot is in that URL
// because blocks are addressed by slot.
//
// TestTransportErrorMatchesARealFailedRequest proves this construction still
// matches what a real failed request produces.
func transportError(slot string) error {
	return errors.Join(
		errors.New("failed to call GET endpoint"),
		&url.Error{
			Op:  "Get",
			URL: "http://beacon:5052/eth/v2/beacon/blocks/" + slot,
			Err: errors.New("dial tcp 10.0.0.5:5052: connect: connection refused"),
		},
	)
}

// A missing block is the one case that may be recorded as missing. Everything
// else has to stay an error so the caller retries.
func TestIsNotFoundAcceptsOnlyAGenuine404(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"a 404 with no body", apiError(404, ""), true},
		{"a 404 with the node's body", apiError(404, `{"code":404,"message":"NOT_FOUND: beacon block"}`), true},
		{"a 500", apiError(500, `{"code":500,"message":"internal error"}`), false},
		{"a 503 while syncing", apiError(503, `{"code":503,"message":"syncing"}`), false},
		{"a 400", apiError(400, `{"code":400,"message":"invalid block ID"}`), false},
		{"no error at all", nil, false},
		{"an unrelated error", errors.New("something went wrong"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNotFound(tc.err); got != tc.want {
				t.Fatalf("isNotFound(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// These are the cases the previous implementation got wrong. It searched the
// rendered message for the substring "404", and the slot is part of the
// request URL, so an unreachable beacon node was reported as a missing block
// for roughly one slot in two hundred. Each of these returned true before the
// fix and must return false now.
func TestIsNotFoundIgnoresA404ThatIsNotAStatusCode(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"the node is unreachable while fetching slot 404", transportError("404")},
		{"the node is unreachable while fetching slot 4040", transportError("4040")},
		{"the node is unreachable while fetching slot 1404000", transportError("1404000")},
		{"a 500 whose body names slot 4040", apiError(500, `{"code":500,"message":"error processing slot 4040"}`)},
		{"a 500 whose body names a root containing 404", apiError(500, `{"message":"bad root 0x404a1b"}`)},
		{"a timeout on a 404-shaped slot", errors.Join(
			errors.New("failed to call GET endpoint"),
			&url.Error{Op: "Get", URL: "http://beacon:5052/eth/v2/beacon/blocks/404", Err: errors.New("context deadline exceeded")},
		)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.err.Error(), "404") {
				t.Fatalf("this case is meant to contain the substring 404 but does not: %v", tc.err)
			}
			if isNotFound(tc.err) {
				t.Errorf("isNotFound reported a missing block for a failure that is not a 404: %v", tc.err)
			}
		})
	}
}

// The error reaches isNotFound through whatever the call stack wrapped it in,
// so the status has to be found through the chain rather than only at the top.
func TestIsNotFoundLooksThroughWrapping(t *testing.T) {
	notFound := apiError(404, "")

	for _, tc := range []struct {
		name string
		err  error
	}{
		{"fmt.Errorf with %w", fmt.Errorf("downloading block at slot 12345: %w", notFound)},
		{"two levels deep", fmt.Errorf("epoch 380: %w", fmt.Errorf("slot 12345: %w", notFound))},
		{"joined with another error", errors.Join(errors.New("failed to call GET endpoint"), notFound)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !isNotFound(tc.err) {
				t.Errorf("isNotFound did not find the 404 through the wrapping: %v", tc.err)
			}
		})
	}
}

// go-eth2-client declares Error() on the value receiver, so api.Error and
// *api.Error both satisfy the error interface. It returns the pointer today.
// If that ever changes, a check that matched only the pointer would stop
// recognising any 404 and every missing slot would retry to exhaustion, so
// both forms are accepted.
func TestIsNotFoundAcceptsTheValueFormOfTheError(t *testing.T) {
	byValue := api.Error{Method: nethttp.MethodGet, StatusCode: 404}
	if !isNotFound(byValue) {
		t.Error("isNotFound did not recognise a 404 delivered as a value rather than a pointer")
	}

	byValueOther := api.Error{Method: nethttp.MethodGet, StatusCode: 500}
	if isNotFound(byValueOther) {
		t.Error("isNotFound accepted a 500 delivered as a value")
	}
}

// transportError above is a hand-built stand-in for what net/http produces.
// This makes a request that really fails and checks the stand-in still matches
// it: same error text shape, same answer from isNotFound. Without this the
// other tests could drift into asserting against a fiction.
//
// The request goes to a closed port on the loopback interface, so it needs no
// network and cannot reach anything.
func TestTransportErrorMatchesARealFailedRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("could not reserve a local port: %s", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	_, realErr := nethttp.Get(fmt.Sprintf("http://%s/eth/v2/beacon/blocks/404", addr))
	if realErr == nil {
		t.Skip("the request unexpectedly succeeded; something is listening on the reserved port")
	}
	joined := errors.Join(errors.New("failed to call GET endpoint"), realErr)

	if !strings.Contains(joined.Error(), "404") {
		t.Fatalf("a real transport failure for slot 404 no longer carries the slot, so the bug this guards has changed shape: %v", joined)
	}
	if isNotFound(joined) {
		t.Errorf("isNotFound reported a missing block for a real transport failure: %v", joined)
	}

	var urlErr *url.Error
	if !errors.As(realErr, &urlErr) {
		t.Errorf("net/http no longer returns a *url.Error, so transportError() is no longer faithful: %T", realErr)
	}
}
