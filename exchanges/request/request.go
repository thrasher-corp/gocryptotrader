package request

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
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
func (r *Requester) SendPayload(ctx context.Context, i *Item) error {
	if !i.NonceEnabled {
		r.timedLock.LockForDuration()
	}

	req, err := i.validateRequest(ctx, r)
	if err != nil {
		r.timedLock.UnlockIfLocked()
		return err
	}

	if i.HTTPDebugging {
		// Err not evaluated due to validation check above
		dump, _ := httputil.DumpRequestOut(req, true)
		log.Debugf(log.RequestSys, "DumpRequest:\n%s", dump)
	}

	if atomic.LoadInt32(&r.jobs) >= MaxRequestJobs {
		r.timedLock.UnlockIfLocked()
		return errors.New("max request jobs reached")
	}

	atomic.AddInt32(&r.jobs, 1)
	err = r.doRequest(req, i)
	atomic.AddInt32(&r.jobs, -1)
	r.timedLock.UnlockIfLocked()

	return err
}

// validateRequest validates the requester item fields
func (i *Item) validateRequest(ctx context.Context, r *Requester) (*http.Request, error) {
	if r == nil || r.Name == "" {
		return nil, errors.New("not initialised, SetDefaults() called before making request?")
	}

	if i == nil {
		return nil, errors.New("request item cannot be nil")
	}

	if i.Path == "" {
		return nil, errors.New("invalid path")
	}

	if i.HeaderResponse != nil {
		if *i.HeaderResponse == nil {
			return nil, errors.New("header response is nil")
		}
	}

	req, err := http.NewRequestWithContext(ctx, i.Method, i.Path, i.Body)
	if err != nil {
		return nil, err
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
func (r *Requester) doRequest(req *http.Request, p *Item) error {
	if p == nil {
		return errors.New("request item cannot be nil")
	}

	if p.Verbose {
		log.Debugf(log.RequestSys,
			"%s request path: %s",
			r.Name,
			p.Path)

		for k, d := range req.Header {
			log.Debugf(log.RequestSys,
				"%s request header [%s]: %s",
				r.Name,
				k,
				d)
		}
		log.Debugf(log.RequestSys,
			"%s request type: %s",
			r.Name,
			req.Method)

		if p.Body != nil {
			log.Debugf(log.RequestSys,
				"%s request body: %v",
				r.Name,
				p.Body)
		}
	}

	for attempt := 1; ; attempt++ {
		// Initiate a rate limit reservation and sleep on requested endpoint
		err := r.InitiateRateLimit(p.Endpoint)
		if err != nil {
			return err
		}

		resp, err := r.HTTPClient.Do(req)
		if retry, checkErr := r.retryPolicy(resp, err); checkErr != nil {
			return checkErr
		} else if retry {
			if err == nil {
				// If the body isn't fully read, the connection cannot be re-used
				r.drainBody(resp.Body)
			}

			// Can't currently regenerate nonce and signatures with fresh values for retries, so for now, we must not retry
			if p.NonceEnabled {
				if timeoutErr, ok := err.(net.Error); !ok || !timeoutErr.Timeout() {
					return fmt.Errorf("unable to retry request using nonce, err: %v", err)
				}
			}

			if attempt > r.maxRetries {
				if err != nil {
					return fmt.Errorf("failed to retry request, err: %v", err)
				}
				return fmt.Errorf("failed to retry request, status: %s", resp.Status)
			}

			after := RetryAfter(resp, time.Now())
			backoff := r.backoff(attempt)
			delay := backoff
			if after > backoff {
				delay = after
			}

			if d, ok := req.Context().Deadline(); ok && d.After(time.Now()) && time.Now().Add(delay).After(d) {
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
	r.Nonce.Inc()
	return r.Nonce.Get()
}

// GetNonceMilli returns a nonce for requests. This locks and enforces concurrent
// nonce FIFO on the buffered job channel this is for millisecond
func (r *Requester) GetNonceMilli() nonce.Value {
	r.timedLock.LockForDuration()
	if r.Nonce.Get() == 0 {
		r.Nonce.Set(time.Now().UnixNano() / int64(time.Millisecond))
		return r.Nonce.Get()
	}
	r.Nonce.Inc()
	return r.Nonce.Get()
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
