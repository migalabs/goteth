package clientapi

import (
	"errors"
	"net/http"

	"github.com/attestantio/go-eth2-client/api"
)

// isNotFound reports whether a beacon API call failed because the resource
// does not exist, as opposed to failing to reach the node at all.
//
// It asks the error for its status code rather than searching the rendered
// message for "404". The message is not a safe place to look: the beacon node
// is addressed by slot, so on a transport failure go-eth2-client joins in the
// error from net/http, which carries the full request URL. Every slot whose
// decimal form contains "404" (about one in two hundred) then produced a
// message matching a substring search, and an unreachable node was recorded as
// a missing block instead of being retried. The port, the peer address and any
// hex root echoed back in an error body could do the same.
//
// go-eth2-client declares Error() on the value receiver, so both api.Error and
// *api.Error satisfy the error interface. It returns the pointer form today;
// both are matched here so that changing which one is returned cannot quietly
// turn every genuine 404 into a retry loop.
func isNotFound(err error) bool {
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}

	var apiErrValue api.Error
	if errors.As(err, &apiErrValue) {
		return apiErrValue.StatusCode == http.StatusNotFound
	}

	return false
}
