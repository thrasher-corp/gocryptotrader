package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/timedmutex"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// UnsetRequest is an unset request authentication level
	UnsetRequest AuthType = 0
	// UnauthenticatedRequest denotes a request with no credentials
	UnauthenticatedRequest = iota << 1
	// AuthenticatedRequest denotes a request using API credentials
	AuthenticatedRequest
)

// AuthType helps distinguish the purpose of a HTTP request
type AuthType uint8

// Public errors
var (
	ErrRequestSystemIsNil = errors.New("request system is nil")
	ErrAuthRequestFailed  = errors.New("authenticated request failed")
	ErrBadStatus          = errors.New("unsuccessful HTTP status code")
)

var (
	errRequestFunctionIsNil   = errors.New("request function is nil")
	errRequestItemNil         = errors.New("request item is nil")
	errInvalidPath            = errors.New("invalid path")
	errHeaderResponseMapIsNil = errors.New("header response map is nil")
	errFailedToRetryRequest   = errors.New("failed to retry request")
	errContextRequired        = errors.New("context is required")
	errTransportNotSet        = errors.New("transport not set, cannot set timeout")
	errRequestTypeUnpopulated = errors.New("request type bool is not populated")
	errExceedsMaxRetries      = errors.New("exceeds maximum retry attempts")
)

// New returns a new Requester
func New(name string, httpRequester *http.Client, opts ...RequesterOption) (*Requester, error) {
	protectedClient, err := newProtectedClient(httpRequester)
	if err != nil {
		return nil, fmt.Errorf("cannot set up a new requester for %s: %w", name, err)
	}
	r := &Requester{
		_HTTPClient: protectedClient,
		name:        name,
		backoff:     DefaultBackoff(),
		retryPolicy: DefaultRetryPolicy,
		maxRetries:  MaxRetryAttempts,
		timedLock:   timedmutex.NewTimedMutex(DefaultMutexLockTimeout),
		reporter:    globalReporter,
	}

	for _, o := range opts {
		o(r)
	}

	return r, nil
}

// SendPayload handles sending HTTP/HTTPS requests
func (r *Requester) SendPayload(ctx context.Context, ep EndpointLimit, newRequest Generate, requestType AuthType) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}

	if ctx == nil {
		return errContextRequired
	}
	if requestType == UnsetRequest {
		return errRequestTypeUnpopulated
	}

	defer r.timedLock.UnlockIfLocked()

	if newRequest == nil {
		return errRequestFunctionIsNil
	}

	err := r.doRequest(ctx, ep, newRequest)
	if err != nil && requestType == AuthenticatedRequest {
		err = common.AppendError(err, ErrAuthRequestFailed)
	}
	return err
}

// validateRequest validates the requester item fields
func (i *Item) validateRequest(ctx context.Context, r *Requester) (*http.Request, error) {
	if i == nil {
		return nil, errRequestItemNil
	}

	if i.Path == "" {
		return nil, errInvalidPath
	}

	if i.HeaderResponse != nil && *i.HeaderResponse == nil {
		return nil, errHeaderResponseMapIsNil
	}

	if !i.NonceEnabled {
		r.timedLock.LockForDuration()
	}
	req, err := http.NewRequestWithContext(ctx, i.Method, i.Path, i.Body)
	if err != nil {
		return nil, err
	}

	if i.HTTPDebugging {
		// Err not evaluated due to validation check above
		dump, _ := httputil.DumpRequestOut(req, true)
		log.Debugf(log.RequestSys, "DumpRequest:\n%s", dump)
	}

	for k, v := range i.Headers {
		req.Header.Add(k, v)
	}

	if r.userAgent != "" && req.Header.Get(userAgent) == "" {
		req.Header.Add(userAgent, r.userAgent)
	}

	return req, nil
}

// doRequest performs a HTTP/HTTPS request with the supplied params
func (r *Requester) doRequest(ctx context.Context, endpoint EndpointLimit, newRequest Generate) error {
	for attempt := 1; ; attempt++ {
		// Check if context has finished before executing new attempt.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if r.limiter != nil {
			// Initiate a rate limit reservation and sleep on requested endpoint
			err := r.InitiateRateLimit(ctx, endpoint)
			if err != nil {
				return fmt.Errorf("failed to rate limit HTTP request: %w", err)
			}
		}

		p, err := newRequest()
		if err != nil {
			return err
		}

		req, err := p.validateRequest(ctx, r)
		if err != nil {
			return err
		}

		verbose := IsVerbose(ctx, p.Verbose)

		if verbose {
			log.Debugf(log.RequestSys, "%s attempt %d request path: %s", r.name, attempt, p.Path)
			for k, d := range req.Header {
				log.Debugf(log.RequestSys, "%s request header [%s]: %s", r.name, k, d)
			}
			log.Debugf(log.RequestSys, "%s request type: %s", r.name, p.Method)
			if req.GetBody != nil {
				bodyCopy, bodyErr := req.GetBody()
				if bodyErr != nil {
					return bodyErr
				}
				payload, bodyErr := io.ReadAll(bodyCopy)
				err = bodyCopy.Close()
				if err != nil {
					log.Errorf(log.RequestSys, "%s failed to close request body %s", r.name, err)
				}
				if bodyErr != nil {
					return bodyErr
				}
				log.Debugf(log.RequestSys, "%s request body: %s", r.name, payload)
			}
		}

		start := time.Now()

		resp, err := r._HTTPClient.do(req)

		if r.reporter != nil && err == nil {
			r.reporter.Latency(r.name, p.Method, p.Path, time.Since(start))
		}

		if retry, err := r.evaluateRetry(ctx, resp, err, attempt, verbose); err != nil {
			return err
		} else if retry {
			continue
		}

		contents, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// Even in the case of an erroneous condition below, yield the parsed
		// response to caller.
		var unmarshallError error
		if p.Result != nil {
			unmarshallError = json.Unmarshal(contents, p.Result)
		}

		if p.HTTPRecording {
			// This dumps http responses for future mocking implementations
			err = mock.HTTPRecord(resp, r.name, contents, p.HTTPMockDataSliceLimit)
			if err != nil {
				return fmt.Errorf("mock recording failure %w, request %v: resp: %v", err, req, resp)
			}
		}

		if p.HeaderResponse != nil {
			maps.Copy(*p.HeaderResponse, resp.Header)
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusNoContent {
			return fmt.Errorf("%s %w: %d raw response: %s", r.name, ErrBadStatus, resp.StatusCode, string(contents))
		}

		if p.HTTPDebugging {
			dump, dumpErr := httputil.DumpResponse(resp, false)
			if dumpErr != nil {
				log.Errorf(log.RequestSys, "DumpResponse invalid response: %v:", dumpErr)
			} else {
				log.Debugf(log.RequestSys, "DumpResponse (%v):\n%s", p.Path, dump)
			}
			log.Debugf(log.RequestSys, "DumpResponse Body (%v):\n %s", p.Path, string(contents))
		}

		err = resp.Body.Close()
		if err != nil {
			log.Errorf(log.RequestSys, "%s failed to close request body %s", r.name, err)
		}
		if verbose {
			for k, d := range resp.Header {
				log.Debugf(log.RequestSys, "%s response header [%s]: %s", r.name, k, d)
			}
			log.Debugf(log.RequestSys, "HTTP status: %s, Code: %v", resp.Status, resp.StatusCode)
			if !p.HTTPDebugging {
				log.Debugf(log.RequestSys, "%s raw response: %s", r.name, string(contents))
			}
		}
		return unmarshallError
	}
}

// evaluateRetry checks whether a request should be retried based on the retry policy and context.
func (r *Requester) evaluateRetry(ctx context.Context, resp *http.Response, incomingErr error, attempt int, verbose bool) (bool, error) {
	if hasRetryNotAllowed(ctx) {
		return false, incomingErr
	}

	retry, err := r.retryPolicy(resp, incomingErr)
	if err != nil {
		return false, err
	}

	if !retry {
		return false, nil
	}

	if incomingErr == nil {
		// If the body isn't fully read, the connection cannot be reused
		r.drainBody(resp.Body)
	}

	if attempt > r.maxRetries {
		if incomingErr != nil {
			return false, fmt.Errorf("%w %w: err: %w", errFailedToRetryRequest, errExceedsMaxRetries, incomingErr)
		}
		return false, fmt.Errorf("%w %w: status %q", errFailedToRetryRequest, errExceedsMaxRetries, resp.Status)
	}

	after := RetryAfter(resp, time.Now())
	backoff := r.backoff(attempt)
	delay := max(backoff, after)

	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(delay)) {
		if incomingErr != nil {
			return false, fmt.Errorf("%w %w: err: %w", errFailedToRetryRequest, context.DeadlineExceeded, incomingErr)
		}
		return false, fmt.Errorf("%w %w: status %q", errFailedToRetryRequest, context.DeadlineExceeded, resp.Status)
	}

	if verbose {
		log.Errorf(log.RequestSys, "%s request has failed. Retrying request in %s, attempt %d", r.name, delay, attempt)
	}

	if delay > 0 {
		// Allow for context cancellation while delaying the retry.
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return false, fmt.Errorf("%w %w", errFailedToRetryRequest, ctx.Err())
		}
	}

	return true, nil
}

func (r *Requester) drainBody(body io.ReadCloser) {
	if _, err := io.Copy(io.Discard, io.LimitReader(body, drainBodyLimit)); err != nil {
		log.Errorf(log.RequestSys, "%s failed to drain request body %s", r.name, err)
	}
	if err := body.Close(); err != nil {
		log.Errorf(log.RequestSys, "%s failed to close request body %s", r.name, err)
	}
}

// GetNonce returns a nonce for requests. This locks and enforces concurrent
// nonce FIFO on the buffered job channel
func (r *Requester) GetNonce(set nonce.Setter) nonce.Value {
	r.timedLock.LockForDuration()
	return r.Nonce.GetAndIncrement(set)
}

// SetProxy sets a proxy address for the client transport
func (r *Requester) SetProxy(p *url.URL) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	return r._HTTPClient.setProxy(p)
}

// SetHTTPClient sets exchanges HTTP client
func (r *Requester) SetHTTPClient(newClient *http.Client) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	protectedClient, err := newProtectedClient(newClient)
	if err != nil {
		return err
	}
	r._HTTPClient = protectedClient
	return nil
}

// SetHTTPClientTimeout sets the timeout value for the exchanges HTTP Client and
// also the underlying transports idle connection timeout
func (r *Requester) SetHTTPClientTimeout(timeout time.Duration) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	return r._HTTPClient.setHTTPClientTimeout(timeout)
}

// SetHTTPClientUserAgent sets the exchanges HTTP user agent
func (r *Requester) SetHTTPClientUserAgent(userAgent string) error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	r.userAgent = userAgent
	return nil
}

// GetHTTPClientUserAgent gets the exchanges HTTP user agent
func (r *Requester) GetHTTPClientUserAgent() (string, error) {
	if r == nil {
		return "", ErrRequestSystemIsNil
	}
	return r.userAgent, nil
}

// Shutdown releases persistent memory for garbage collection.
func (r *Requester) Shutdown() error {
	if r == nil {
		return ErrRequestSystemIsNil
	}
	return r._HTTPClient.release()
}
