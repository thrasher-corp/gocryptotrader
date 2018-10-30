package request

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/exchanges/nonce"
	log "github.com/thrasher-/gocryptotrader/logger"
)

var supportedMethods = []string{http.MethodGet, http.MethodPost, http.MethodHead,
	http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodConnect}

// Const vars for rate limiter
const (
	DefaultMaxRequestJobs       = 50
	DefaultTimeoutRetryAttempts = 3

	proxyTLSTimeout = 15 * time.Second
)

// Vars for rate limiter
var (
	MaxRequestJobs       = DefaultMaxRequestJobs
	TimeoutRetryAttempts = DefaultTimeoutRetryAttempts
	DisableRateLimiter   bool
)

// Requester struct for the request client
type Requester struct {
	HTTPClient           *http.Client
	UnauthLimit          *RateLimit
	AuthLimit            *RateLimit
	Name                 string
	UserAgent            string
	Cycle                time.Time
	timeoutRetryAttempts int
	m                    sync.Mutex
	Jobs                 chan Job
	disengage            chan struct{}
	WorkerStarted        bool
	Nonce                nonce.Nonce
	fifoLock             sync.Mutex
	DisableRateLimiter   bool
}

// RateLimit struct
type RateLimit struct {
	Duration time.Duration
	Rate     int
	Requests int
	Mutex    sync.Mutex
}

// JobResult holds a request job result
type JobResult struct {
	Error  error
	Result interface{}
}

// Job holds a request job
type Job struct {
	Request     *http.Request
	Method      string
	Path        string
	Headers     map[string]string
	Body        io.Reader
	Result      interface{}
	JobResult   chan *JobResult
	AuthRequest bool
	Verbose     bool
}

// NewRateLimit creates a new RateLimit
func NewRateLimit(d time.Duration, rate int) *RateLimit {
	return &RateLimit{Duration: d, Rate: rate}
}

// ToString returns the rate limiter in string notation
func (r *RateLimit) ToString() string {
	return fmt.Sprintf("Rate limiter set to %d requests per %v", r.Rate, r.Duration)
}

// GetRate returns the ratelimit rate
func (r *RateLimit) GetRate() int {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.Rate
}

// SetRate sets the ratelimit rate
func (r *RateLimit) SetRate(rate int) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Rate = rate
}

// GetRequests returns the number of requests for the ratelimit
func (r *RateLimit) GetRequests() int {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.Requests
}

// SetRequests sets requests counter for the rateliit
func (r *RateLimit) SetRequests(l int) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Requests = l
}

// SetDuration sets the duration for the ratelimit
func (r *RateLimit) SetDuration(d time.Duration) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Duration = d
}

// GetDuration gets the duration for the ratelimit
func (r *RateLimit) GetDuration() time.Duration {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.Duration
}

// StartCycle restarts the cycle time and requests counters
func (r *Requester) StartCycle() {
	r.Cycle = time.Now()
	r.AuthLimit.SetRequests(0)
	r.UnauthLimit.SetRequests(0)
}

// IsRateLimited returns whether or not the request Requester is rate limited
func (r *Requester) IsRateLimited(auth bool) bool {
	if auth {
		if r.AuthLimit.GetRequests() >= r.AuthLimit.GetRate() && r.IsValidCycle(auth) {
			return true
		}
	} else {
		if r.UnauthLimit.GetRequests() >= r.UnauthLimit.GetRate() && r.IsValidCycle(auth) {
			return true
		}
	}
	return false
}

// RequiresRateLimiter returns whether or not the request Requester requires a rate limiter
func (r *Requester) RequiresRateLimiter() bool {
	if DisableRateLimiter {
		return false
	}

	if r.AuthLimit.GetRate() != 0 || r.UnauthLimit.GetRate() != 0 {
		return true
	}
	return false
}

// IncrementRequests increments the ratelimiter request counter for either auth or unauth
// requests
func (r *Requester) IncrementRequests(auth bool) {
	if auth {
		reqs := r.AuthLimit.GetRequests()
		reqs++
		r.AuthLimit.SetRequests(reqs)
		return
	}

	reqs := r.UnauthLimit.GetRequests()
	reqs++
	r.UnauthLimit.SetRequests(reqs)
}

// DecrementRequests decrements the ratelimiter request counter for either auth or unauth
// requests
func (r *Requester) DecrementRequests(auth bool) {
	if auth {
		reqs := r.AuthLimit.GetRequests()
		reqs--
		r.AuthLimit.SetRequests(reqs)
		return
	}

	reqs := r.AuthLimit.GetRequests()
	reqs--
	r.UnauthLimit.SetRequests(reqs)
}

// SetRateLimit sets the request Requester ratelimiter
func (r *Requester) SetRateLimit(auth bool, duration time.Duration, rate int) {
	if auth {
		r.AuthLimit.SetRate(rate)
		r.AuthLimit.SetDuration(duration)
		return
	}
	r.UnauthLimit.SetRate(rate)
	r.UnauthLimit.SetDuration(duration)
}

// GetRateLimit gets the request Requester ratelimiter
func (r *Requester) GetRateLimit(auth bool) *RateLimit {
	if auth {
		return r.AuthLimit
	}
	return r.UnauthLimit
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
func New(name string, authLimit, unauthLimit *RateLimit, httpRequester *http.Client) *Requester {
	return &Requester{
		HTTPClient:           httpRequester,
		UnauthLimit:          unauthLimit,
		AuthLimit:            authLimit,
		Name:                 name,
		Jobs:                 make(chan Job, MaxRequestJobs),
		disengage:            make(chan struct{}, 1),
		timeoutRetryAttempts: TimeoutRetryAttempts,
	}
}

// IsValidMethod returns whether the supplied method is supported
func IsValidMethod(method string) bool {
	return common.StringDataCompareInsensitive(supportedMethods, method)
}

// IsValidCycle checks to see whether the current request cycle is valid or not
func (r *Requester) IsValidCycle(auth bool) bool {
	if auth {
		if time.Since(r.Cycle) < r.AuthLimit.GetDuration() {
			return true
		}
	} else {
		if time.Since(r.Cycle) < r.UnauthLimit.GetDuration() {
			return true
		}
	}

	r.StartCycle()
	return false
}

func (r *Requester) checkRequest(method, path string, body io.Reader, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	if r.UserAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Add("User-Agent", r.UserAgent)
	}

	return req, nil
}

// DoRequest performs a HTTP/HTTPS request with the supplied params
func (r *Requester) DoRequest(req *http.Request, path string, body io.Reader, result interface{}, authRequest, verbose bool) error {
	if verbose {
		log.Debugf("%s exchange request path: %s requires rate limiter: %v", r.Name, path, r.RequiresRateLimiter())
		for k, d := range req.Header {
			log.Debugf("%s exchange request header [%s]: %s", r.Name, k, d)
		}
		log.Debug(body)
	}

	var timeoutError error
	for i := 0; i < r.timeoutRetryAttempts+1; i++ {
		resp, err := r.HTTPClient.Do(req)
		if err != nil {
			if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
				if verbose {
					log.Errorf("%s request has timed-out retrying request, count %d",
						r.Name,
						i)
				}
				timeoutError = err
				continue
			}

			if r.RequiresRateLimiter() {
				r.DecrementRequests(authRequest)
			}
			return err
		}
		if resp == nil {
			if r.RequiresRateLimiter() {
				r.DecrementRequests(authRequest)
			}
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
			switch {
			case common.StringContains(resp.Header.Get("Content-Type"), "application/json"):
				reader = resp.Body

			default:
				log.Warnf("%s request response content type differs from JSON; received %v [path: %s]",
					r.Name, resp.Header.Get("Content-Type"), path)
				reader = resp.Body
			}
		}

		contents, err := ioutil.ReadAll(reader)
		if err != nil {
			return err
		}

		if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
			err = fmt.Errorf("unsuccessful HTTP status code: %d", resp.StatusCode)

			if verbose {
				err = fmt.Errorf("%s\n%s", err.Error(),
					fmt.Sprintf("%s exchange raw response: %s", r.Name, string(contents)))
			}

			return err
		}

		resp.Body.Close()
		if verbose {
			log.Debugf("HTTP status: %s, Code: %v", resp.Status, resp.StatusCode)
			log.Debugf("%s exchange raw response: %s", r.Name, string(contents))
		}

		if result != nil {
			return common.JSONDecode(contents, result)
		}

		return nil
	}
	return fmt.Errorf("request.go error - failed to retry request %s",
		timeoutError)
}

func (r *Requester) worker() {
	for {
		for x := range r.Jobs {
			if !r.IsRateLimited(x.AuthRequest) {
				r.IncrementRequests(x.AuthRequest)

				err := r.DoRequest(x.Request, x.Path, x.Body, x.Result, x.AuthRequest, x.Verbose)
				x.JobResult <- &JobResult{
					Error:  err,
					Result: x.Result,
				}
			} else {
				limit := r.GetRateLimit(x.AuthRequest)
				diff := limit.GetDuration() - time.Since(r.Cycle)
				if x.Verbose {
					log.Debugf("%s request. Rate limited! Sleeping for %v", r.Name, diff)
				}
				time.Sleep(diff)

				for {
					if r.IsRateLimited(x.AuthRequest) {
						time.Sleep(time.Millisecond)
						continue
					}
					r.IncrementRequests(x.AuthRequest)

					if x.Verbose {
						log.Debugf("%s request. No longer rate limited! Doing request", r.Name)
					}

					err := r.DoRequest(x.Request, x.Path, x.Body, x.Result, x.AuthRequest, x.Verbose)
					x.JobResult <- &JobResult{
						Error:  err,
						Result: x.Result,
					}
					break
				}
			}
		}
	}
}

// SendPayload handles sending HTTP/HTTPS requests
func (r *Requester) SendPayload(method, path string, headers map[string]string, body io.Reader, result interface{}, authRequest, nonceEnabled, verbose bool) error {
	if !nonceEnabled {
		r.lock()
	}

	if r == nil || r.Name == "" {
		r.unlock()
		return errors.New("not initiliased, SetDefaults() called before making request?")
	}

	if !IsValidMethod(method) {
		r.unlock()
		return fmt.Errorf("incorrect method supplied %s: supported %s", method, supportedMethods)
	}

	if path == "" {
		r.unlock()
		return errors.New("invalid path")
	}

	req, err := r.checkRequest(method, path, body, headers)
	if err != nil {
		r.unlock()
		return err
	}

	if !r.RequiresRateLimiter() {
		r.unlock()
		return r.DoRequest(req, path, body, result, authRequest, verbose)
	}

	if len(r.Jobs) == MaxRequestJobs {
		r.unlock()
		return errors.New("max request jobs reached")
	}

	r.m.Lock()
	if !r.WorkerStarted {
		r.StartCycle()
		r.WorkerStarted = true
		go r.worker()
	}
	r.m.Unlock()

	jobResult := make(chan *JobResult)

	newJob := Job{
		Request:     req,
		Method:      method,
		Path:        path,
		Headers:     headers,
		Body:        body,
		Result:      result,
		JobResult:   jobResult,
		AuthRequest: authRequest,
		Verbose:     verbose,
	}

	if verbose {
		log.Debugf("%s request. Attaching new job.", r.Name)
	}
	r.Jobs <- newJob
	r.unlock()

	if verbose {
		log.Debugf("%s request. Waiting for job to complete.", r.Name)
	}
	resp := <-newJob.JobResult

	if verbose {
		log.Debugf("%s request. Job complete.", r.Name)
	}
	return resp.Error
}

// GetNonce returns a nonce for requests. This locks and enforces concurrent
// nonce FIFO on the buffered job channel
func (r *Requester) GetNonce(isNano bool) nonce.Value {
	r.lock()
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
	r.lock()
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

// lock locks and sets up an issue timer, if something errors out of scope it
// automatically unlocks
func (r *Requester) lock() {
	if r.disengage == nil {
		r.disengage = make(chan struct{}, 1)
	}
	var wg sync.WaitGroup
	r.fifoLock.Lock()
	wg.Add(1)
	go func() {
		timer := time.NewTimer(50 * time.Millisecond)
		wg.Done()
		select {
		case <-timer.C:
			log.Errorf("Unlocking due to possible error for %s", r.Name)
			r.fifoLock.Unlock()

		case <-r.disengage:
			return
		}
	}()
	wg.Wait()
}

// unlock unlocks mtx and shuts down a timer
func (r *Requester) unlock() {
	r.disengage <- struct{}{}
	r.fifoLock.Unlock()
}
