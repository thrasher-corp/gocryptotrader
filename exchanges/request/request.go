package request

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/timedmutex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"golang.org/x/time/rate"
)

// NewRateLimit creates a new RateLimit based of time interval and how many
// actions allowed down to a APS level -- Burst rate is kept as one action per
// rate of interval
func NewRateLimit(interval time.Duration, actions int) *rate.Limiter {
	if actions == 0 || interval == 0 {
		// Just gives you an open rate limiter
		return rate.NewLimiter(rate.Inf, 1)
	}

	invSeconds := 1 / interval.Seconds()
	actualRate := invSeconds * float64(actions)
	return rate.NewLimiter(rate.Limit(actualRate), 1)
}

// SetTimeoutRetryAttempts sets the amount of times the job will be retried
// if it times out
func (r *Requester) SetTimeoutRetryAttempts(n int) error {
	if n < 0 {
		return errors.New("routines.go error - timeout retry attempts cannot be less than zero")
	}
	r.timeoutRetryAttempts = n
	return nil
}

// New returns a new Requester
func New(name string, authLimit, unauthLimit *rate.Limiter, httpRequester *http.Client) *Requester {
	return &Requester{
		HTTPClient:           httpRequester,
		UnauthLimit:          unauthLimit,
		AuthLimit:            authLimit,
		Name:                 name,
		timeoutRetryAttempts: TimeoutRetryAttempts,
		timedLock:            timedmutex.NewTimedMutex(DefaultMutexLockTimeout),
	}
}

// IsValidMethod returns whether the supplied method is supported
func IsValidMethod(method string) bool {
	return common.StringDataCompareInsensitive(supportedMethods, method)
}

func (i *Item) checkRequest(r *Requester) (*http.Request, error) {
	if r == nil || r.Name == "" {
		return nil, errors.New("not initiliased, SetDefaults() called before making request?")
	}

	if i == nil {
		return nil, errors.New("request item cannot be nil")
	}

	if i.Path == "" {
		return nil, errors.New("invalid path")
	}

	req, err := http.NewRequest(i.Method, i.Path, i.Body)
	if err != nil {
		return nil, err
	}

	for k, v := range i.Headers {
		req.Header.Add(k, v)
	}

	if r.UserAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Add("User-Agent", r.UserAgent)
	}

	return req, nil
}

// DoRequest performs a HTTP/HTTPS request with the supplied params
func (r *Requester) DoRequest(req *http.Request, p *Item) error {
	if p.Verbose {
		log.Debugf(log.Global,
			"%s exchange request path: %s requires rate limiter: %v",
			r.Name,
			p.Path,
			// r.RequiresRateLimiter())
			false)

		for k, d := range req.Header {
			log.Debugf(log.Global,
				"%s exchange request header [%s]: %s",
				r.Name,
				k,
				d)
		}
		log.Debugf(log.Global,
			"%s exchange request type: %s",
			r.Name,
			req.Method)
		log.Debugf(log.Global,
			"%s exchange request body: %v",
			r.Name,
			p.Body)
	}

	var timeoutError error
	for i := 0; i < r.timeoutRetryAttempts+1; i++ {
		err := r.InitiateRateLimit(p.AuthRequest, p.Verbose)
		if err != nil {
			return err
		}

		resp, err := r.HTTPClient.Do(req)
		if err != nil {
			if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
				if p.Verbose {
					log.Errorf(log.ExchangeSys,
						"%s request has timed-out retrying request, count %d",
						r.Name,
						i)
				}
				timeoutError = err
				continue
			}
			return err
		}

		if resp == nil {
			return errors.New("resp is nil")
		}

		var reader io.ReadCloser
		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err = gzip.NewReader(resp.Body)
			defer reader.Close()
			if err != nil {
				return err
			}

		case "json":
			reader = resp.Body

		default:
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				if p.Verbose {
					log.Warnf(log.ExchangeSys,
						"%s request response content type differs from JSON; received %v [path: %s]\n",
						r.Name,
						contentType,
						p.Path)
				}
			}
			reader = resp.Body
		}

		contents, err := ioutil.ReadAll(reader)
		if err != nil {
			return err
		}

		if p.HTTPRecording {
			// This dumps http responses for future mocking implementations
			err = mock.HTTPRecord(resp, r.Name, contents)
			if err != nil {
				return fmt.Errorf("mock recording failure %s", err)
			}
		}

		if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
			return fmt.Errorf("%s exchange unsuccessful HTTP status code: %d  raw response: %s",
				r.Name,
				resp.StatusCode,
				string(contents))
		}

		if p.HTTPDebugging {
			dump, err := httputil.DumpResponse(resp, false)
			if err != nil {
				log.Errorf(log.Global, "DumpResponse invalid response: %v:", err)
			}
			log.Debugf(log.Global, "DumpResponse Headers (%v):\n%s", p.Path, dump)
			log.Debugf(log.Global, "DumpResponse Body (%v):\n %s", p.Path, string(contents))
		}

		resp.Body.Close()
		if p.Verbose {
			log.Debugf(log.ExchangeSys,
				"HTTP status: %s, Code: %v",
				resp.Status,
				resp.StatusCode)
			if !p.HTTPDebugging {
				log.Debugf(log.ExchangeSys,
					"%s exchange raw response: %s",
					r.Name,
					string(contents))
			}
		}
		return json.Unmarshal(contents, p.Result)
	}
	return fmt.Errorf("request.go error - failed to retry request %s",
		timeoutError)
}

// InitiateRateLimit sets call for auth, unauth or global rate limit shells
func (r *Requester) InitiateRateLimit(auth, verbose bool) error {
	if DisableRateLimiter {
		return nil
	}

	var limit *rate.Limiter
	if auth {
		limit = r.AuthLimit
	} else {
		limit = r.UnauthLimit
	}

	reserved := limit.Reserve()

	delay := reserved.Delay()
	if delay == rate.InfDuration {
		return errors.New("issues")
	}
	time.Sleep(reserved.Delay())
	return nil
}

// SendPayload handles sending HTTP/HTTPS requests
func (r *Requester) SendPayload(i *Item) error {
	if !i.NonceEnabled {
		r.timedLock.LockForDuration()
	}

	req, err := i.checkRequest(r)
	if err != nil {
		r.timedLock.UnlockIfLocked()
		return err
	}

	if i.HTTPDebugging {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.Errorf(log.Global,
				"DumpRequest invalid response %v:", err)
		}
		log.Debugf(log.Global,
			"DumpRequest:\n%s", dump)
	}

	if atomic.LoadInt32(&r.jobs) >= MaxRequestJobs {
		r.timedLock.UnlockIfLocked()
		return errors.New("max request jobs reached")
	}

	atomic.AddInt32(&r.jobs, 1)

	err = r.DoRequest(req, i)

	atomic.AddInt32(&r.jobs, -1)
	r.timedLock.UnlockIfLocked()

	return err
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

	r.HTTPClient.Transport = &http.Transport{
		Proxy:               http.ProxyURL(p),
		TLSHandshakeTimeout: proxyTLSTimeout,
	}
	return nil
}

// DisableRateLimit disables rate limiting on the requester side so it can be
// handled by the work management system
func (r *Requester) DisableRateLimit() {
	r.DisableRateLimiter = false
}

// EnableRateLimit enables rate limiting on the requester side so it can be rate
// limit calls
func (r *Requester) EnableRateLimit() {
	r.DisableRateLimiter = true
}
