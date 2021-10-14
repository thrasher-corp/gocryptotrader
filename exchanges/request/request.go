package request

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/timedmutex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errRequestSystemIsNil     = errors.New("request system is nil")
	errMaxRequestJobs         = errors.New("max request jobs reached")
	errRequestFunctionIsNil   = errors.New("request function is nil")
	errServiceNameUnset       = errors.New("service name unset")
	errRequestItemNil         = errors.New("request item is nil")
	errInvalidPath            = errors.New("invalid path")
	errHeaderResponseMapIsNil = errors.New("header response map is nil")
	errFailedToRetryRequest   = errors.New("failed to retry request")
	errContextRequired        = errors.New("context is required")
)

// New returns a new Requester
func New(name string, httpRequester *http.Client, opts ...RequesterOption) *Requester {
	r := &Requester{
		HTTPClient:  httpRequester,
		Name:        name,
		backoff:     DefaultBackoff(),
		retryPolicy: DefaultRetryPolicy,
		maxRetries:  MaxRetryAttempts,
		timedLock:   timedmutex.NewTimedMutex(DefaultMutexLockTimeout),
	}

	for _, o := range opts {
		o(r)
	}

	return r
}

// SendPayload handles sending HTTP/HTTPS requests
func (r *Requester) SendPayload(ctx context.Context, ep EndpointLimit, newRequest Generate) error {
	if r == nil {
		return errRequestSystemIsNil
	}

	if ctx == nil {
		return errContextRequired
	}

	defer r.timedLock.UnlockIfLocked()

	if newRequest == nil {
		return errRequestFunctionIsNil
	}

	if atomic.LoadInt32(&r.jobs) >= MaxRequestJobs {
		return errMaxRequestJobs
	}

	atomic.AddInt32(&r.jobs, 1)
	err := r.doRequest(ctx, ep, newRequest)
	atomic.AddInt32(&r.jobs, -1)
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

	if r.UserAgent != "" && req.Header.Get(userAgent) == "" {
		req.Header.Add(userAgent, r.UserAgent)
	}

	return req, nil
}

// DoRequest performs a HTTP/HTTPS request with the supplied params
func (r *Requester) doRequest(ctx context.Context, endpoint EndpointLimit, newRequest Generate) error {
	for attempt := 1; ; attempt++ {
		// Check if context has finished before executing new attempt.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Initiate a rate limit reservation and sleep on requested endpoint
		err := r.InitiateRateLimit(ctx, endpoint)
		if err != nil {
			return fmt.Errorf("failed to rate limit HTTP request: %w", err)
		}

		p, err := newRequest()
		if err != nil {
			return err
		}

		req, err := p.validateRequest(ctx, r)
		if err != nil {
			return err
		}

		if p.Verbose {
			log.Debugf(log.RequestSys, "%s attempt %d request path: %s", r.Name, attempt, p.Path)
			for k, d := range req.Header {
				log.Debugf(log.RequestSys, "%s request header [%s]: %s", r.Name, k, d)
			}
			log.Debugf(log.RequestSys, "%s request type: %s", r.Name, p.Method)
			if p.Body != nil {
				log.Debugf(log.RequestSys, "%s request body: %v", r.Name, p.Body)
			}
		}

		resp, err := r.HTTPClient.Do(req)
		if retry, checkErr := r.retryPolicy(resp, err); checkErr != nil {
			return checkErr
		} else if retry {
			if err == nil {
				// If the body isn't fully read, the connection cannot be re-used
				r.drainBody(resp.Body)
			}

			if attempt > r.maxRetries {
				if err != nil {
					return fmt.Errorf("%w, err: %v", errFailedToRetryRequest, err)
				}
				return fmt.Errorf("%w, status: %s", errFailedToRetryRequest, resp.Status)
			}

			after := RetryAfter(resp, time.Now())
			backoff := r.backoff(attempt)
			delay := backoff
			if after > backoff {
				delay = after
			}

			if dl, ok := req.Context().Deadline(); ok && dl.Before(time.Now().Add(delay)) {
				if err != nil {
					return fmt.Errorf("deadline would be exceeded by retry, err: %v", err)
				}
				return fmt.Errorf("deadline would be exceeded by retry, status: %s", resp.Status)
			}

			if p.Verbose {
				log.Errorf(log.RequestSys,
					"%s request has failed. Retrying request in %s, attempt %d",
					r.Name,
					delay,
					attempt)
			}

			time.Sleep(delay)
			continue
		}

		contents, err := ioutil.ReadAll(resp.Body)
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
			err = mock.HTTPRecord(resp, r.Name, contents)
			if err != nil {
				return fmt.Errorf("mock recording failure %s", err)
			}
		}

		if p.HeaderResponse != nil {
			for k, v := range resp.Header {
				(*p.HeaderResponse)[k] = v
			}
		}

		if resp.StatusCode < http.StatusOK ||
			resp.StatusCode > http.StatusAccepted {
			return fmt.Errorf("%s unsuccessful HTTP status code: %d raw response: %s",
				r.Name,
				resp.StatusCode,
				string(contents))
		}

		if p.HTTPDebugging {
			dump, err := httputil.DumpResponse(resp, false)
			if err != nil {
				log.Errorf(log.RequestSys, "DumpResponse invalid response: %v:", err)
			}
			log.Debugf(log.RequestSys, "DumpResponse Headers (%v):\n%s", p.Path, dump)
			log.Debugf(log.RequestSys, "DumpResponse Body (%v):\n %s", p.Path, string(contents))
		}

		resp.Body.Close()
		if p.Verbose {
			log.Debugf(log.RequestSys,
				"HTTP status: %s, Code: %v",
				resp.Status,
				resp.StatusCode)
			if !p.HTTPDebugging {
				log.Debugf(log.RequestSys,
					"%s raw response: %s",
					r.Name,
					string(contents))
			}
		}
		return unmarshallError
	}
}

// GetNonce returns a nonce for requests. This locks and enforces concurrent
// nonce FIFO on the buffered job channel
func (r *Requester) GetNonce(isNano bool) nonce.Value {
	r.timedLock.LockForDuration()
	if r.Nonce.Get() == 0 {
		if isNano {
			r.Nonce.Set(time.Now().UnixNano())
		} else {
			r.Nonce.Set(time.Now().Unix())
		}
		return r.Nonce.Get()
	}
	return r.Nonce.GetInc()
}

// GetNonceMilli returns a nonce for requests. This locks and enforces concurrent
// nonce FIFO on the buffered job channel this is for millisecond
func (r *Requester) GetNonceMilli() nonce.Value {
	r.timedLock.LockForDuration()
	if r.Nonce.Get() == 0 {
		r.Nonce.Set(time.Now().UnixMilli())
		return r.Nonce.Get()
	}
	return r.Nonce.GetInc()
}

// SetProxy sets a proxy address to the client transport
func (r *Requester) SetProxy(p *url.URL) error {
	if p.String() == "" {
		return errors.New("no proxy URL supplied")
	}

	t, ok := r.HTTPClient.Transport.(*http.Transport)
	if !ok {
		return errors.New("transport not set, cannot set proxy")
	}
	t.Proxy = http.ProxyURL(p)
	t.TLSHandshakeTimeout = proxyTLSTimeout
	return nil
}

func (r *Requester) drainBody(body io.ReadCloser) {
	defer body.Close()
	if _, err := io.Copy(ioutil.Discard, io.LimitReader(body, drainBodyLimit)); err != nil {
		log.Errorf(log.RequestSys,
			"%s failed to drain request body %s",
			r.Name,
			err)
	}
}
