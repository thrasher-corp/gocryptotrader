package request

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/idoall/gocryptotrader/common"
)

var supportedMethods = []string{"GET", "POST", "HEAD", "PUT", "DELETE", "OPTIONS", "CONNECT"}

// Requester struct for the request client
type Requester struct {
	HTTPClient  *http.Client
	UnauthLimit RateLimit
	AuthLimit   RateLimit
	Name        string
	Cycle       time.Time
	m           sync.Mutex
}

// RateLimit struct
type RateLimit struct {
	Duration time.Duration
	Rate     int
	Requests int
	Mutex    sync.Mutex
}

// NewRateLimit creates a new RateLimit
func NewRateLimit(d time.Duration, rate int) RateLimit {
	return RateLimit{Duration: d, Rate: rate}
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
		// fmt.Println(r.AuthLimit.GetRequests())
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
	if r.AuthLimit.GetRate() != 0 || r.UnauthLimit.GetRate() != 0 {
		return true
	}
	return false
}

// IncrementRequests increments the ratelimiter request counter for either auth or unauth
// requests 增量计数器
func (r *Requester) IncrementRequests(auth bool) {
	if auth {
		reqs := r.AuthLimit.GetRequests()
		reqs++
		r.AuthLimit.SetRequests(reqs)
		return
	}

	reqs := r.AuthLimit.GetRequests()
	reqs++
	r.UnauthLimit.SetRequests(reqs)
}

// DecrementRequests decrements the ratelimiter request counter for either auth or unauth
// requests 减量计数器
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
func (r *Requester) GetRateLimit(auth bool) RateLimit {
	if auth {
		return r.AuthLimit
	}
	return r.UnauthLimit
}

// New returns a new Requester
func New(name string, authLimit, unauthLimit RateLimit, httpRequester *http.Client) *Requester {
	r := &Requester{HTTPClient: httpRequester, UnauthLimit: unauthLimit, AuthLimit: authLimit, Name: name}
	return r
}

// IsValidMethod returns whether the supplied method is supported
func IsValidMethod(method string) bool {
	return common.StringDataCompareUpper(supportedMethods, method)
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

	return req, nil
}

// DoRequest performs a HTTP/HTTPS request with the supplied params
func (r *Requester) DoRequest(req *http.Request, method, path string, headers map[string]string, body io.Reader, result interface{}, authRequest, verbose bool) error {
	if verbose {
		log.Printf("%s exchange request path: %s", r.Name, path)
	}

	resp, err := r.HTTPClient.Do(req)

	if err != nil {
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

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	resp.Body.Close()
	if verbose {
		log.Printf("%s exchange raw response: %s", r.Name, string(contents[:]))
	}

	// fmt.Println(common.JSONDecode(contents, result))

	if result != nil {
		return common.JSONDecode(contents, result)
	}

	return nil
}

// SendPayload handles sending HTTP/HTTPS requests
func (r *Requester) SendPayload(method, path string, headers map[string]string, body io.Reader, result interface{}, authRequest, verbose bool) error {
	if r == nil || r.Name == "" {
		return errors.New("not initiliased, SetDefaults() called before making request?")
	}

	if !IsValidMethod(method) {
		return fmt.Errorf("incorrect method supplied %s: supported %s", method, supportedMethods)
	}

	if path == "" {
		return errors.New("invalid path")
	}

	req, err := r.checkRequest(method, path, body, headers)
	if err != nil {
		return err
	}

	if !r.RequiresRateLimiter() {
		return r.DoRequest(req, method, path, headers, body, result, authRequest, verbose)
	}

	r.m.Lock()
	if r.Cycle.IsZero() || !r.IsValidCycle(authRequest) {
		r.StartCycle()
	}
	r.m.Unlock()

	if !r.IsRateLimited(authRequest) && r.IsValidCycle(authRequest) {
		r.IncrementRequests(authRequest)
		err := r.DoRequest(req, method, path, headers, body, result, authRequest, verbose)
		return err

	}

	r.m.Lock()
	for r.IsRateLimited(authRequest) {
		limit := r.GetRateLimit(authRequest)
		diff := limit.GetDuration() - time.Since(r.Cycle)
		log.Printf("%s IS RATE LIMITED. SLEEPING FOR %v", r.Name, diff)
		time.Sleep(diff)

		if !r.IsValidCycle(authRequest) {
			r.StartCycle()
		}

		if !r.IsRateLimited(authRequest) && r.IsValidCycle(authRequest) {
			r.IncrementRequests(authRequest)
			r.m.Unlock()
			return r.DoRequest(req, method, path, headers, body, result, authRequest, verbose)
		}
	}
	return nil
}
