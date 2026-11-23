// Package standin answers HTTP requests with canned replies, so tests run without the services they call
package standin

import (
	"context"
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

// RoundTrip answers with the function's reply, or with the context's cause if the request is already cancelled. It then
// closes the body the request was sent with, and returns the close error if answering succeeded.
func (t Transport) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	if body := r.Body; body != nil {
		defer func() {
			if closeErr := body.Close(); closeErr != nil && err == nil {
				resp, err = nil, closeErr
			}
		}()
	}
	if ctx := r.Context(); ctx.Err() != nil {
		return nil, context.Cause(ctx)
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
