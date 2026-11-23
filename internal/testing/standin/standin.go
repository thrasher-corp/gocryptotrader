// Package standin answers HTTP requests with canned replies, so tests run without the services they call
package standin

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// ErrNoReply is returned for a request that no canned reply covers
var ErrNoReply = errors.New("no stand-in reply")

// Reply is a canned HTTP reply
type Reply struct {
	Status int
	Body   string
}

// Transport answers each request with the reply its function returns
type Transport func(*http.Request) (Reply, error)

// RoundTrip closes the request body and answers with the function's reply
func (t Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		if err := r.Body.Close(); err != nil {
			return nil, err
		}
	}
	reply, err := t(r)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: reply.Status,
		Status:     strconv.Itoa(reply.Status) + " " + http.StatusText(reply.Status),
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(reply.Body)),
		Request:    r,
	}, nil
}

// Client returns an HTTP client whose every request the transport answers
func Client(t Transport) *http.Client {
	return &http.Client{Transport: t}
}
