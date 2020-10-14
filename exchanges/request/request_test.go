package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

const unexpected = "unexpected values"

var testURL string
var serverLimit *rate.Limiter

func TestMain(m *testing.M) {
	serverLimitInterval := time.Millisecond * 500
	serverLimit = NewRateLimit(serverLimitInterval, 1)
	serverLimitRetry := NewRateLimit(serverLimitInterval, 1)
	sm := http.NewServeMux()
	sm.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"response":true}`)
	})
	sm.HandleFunc("/error", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":true}`)
	})
	sm.HandleFunc("/timeout", func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(time.Millisecond * 100)
		w.WriteHeader(http.StatusGatewayTimeout)
	})
	sm.HandleFunc("/rate", func(w http.ResponseWriter, req *http.Request) {
		if !serverLimit.Allow() {
			http.Error(w,
				http.StatusText(http.StatusTooManyRequests),
				http.StatusTooManyRequests)
			io.WriteString(w, `{"response":false}`)
			return
		}
		io.WriteString(w, `{"response":true}`)
	})
	sm.HandleFunc("/rate-retry", func(w http.ResponseWriter, req *http.Request) {
		if !serverLimitRetry.Allow() {
			w.Header().Add("Retry-After", strconv.Itoa(int(math.Round(serverLimitInterval.Seconds()))))
			http.Error(w,
				http.StatusText(http.StatusTooManyRequests),
				http.StatusTooManyRequests)
			io.WriteString(w, `{"response":false}`)
			return
		}
		io.WriteString(w, `{"response":true}`)
	})
	sm.HandleFunc("/always-retry", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Retry-After", time.Now().Format(time.RFC1123))
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"response":false}`)
	})

	server := httptest.NewServer(sm)
	testURL = server.URL
	issues := m.Run()
	server.Close()
	os.Exit(issues)
}

func TestNewRateLimit(t *testing.T) {
	t.Parallel()
	r := NewRateLimit(time.Second*10, 5)
	if r.Limit() != 0.5 {
		t.Fatal(unexpected)
	}

	// Ensures rate limiting factor is the same
	r = NewRateLimit(time.Second*2, 1)
	if r.Limit() != 0.5 {
		t.Fatal(unexpected)
	}

	// Test for open rate limit
	r = NewRateLimit(time.Second*2, 0)
	if r.Limit() != rate.Inf {
		t.Fatal(unexpected)
	}

	r = NewRateLimit(0, 69)
	if r.Limit() != rate.Inf {
		t.Fatal(unexpected)
	}
}

func TestCheckRequest(t *testing.T) {
	t.Parallel()

	r := New("TestRequest",
		new(http.Client))
	ctx := context.Background()

	var check *Item
	_, err := check.validateRequest(ctx, &Requester{})
	if err == nil {
		t.Fatal(unexpected)
	}

	_, err = check.validateRequest(ctx, nil)
	if err == nil {
		t.Fatal(unexpected)
	}

	_, err = check.validateRequest(ctx, r)
	if err == nil {
		t.Fatal(unexpected)
	}

	check = &Item{}
	_, err = check.validateRequest(ctx, r)
	if err == nil {
		t.Fatal(unexpected)
	}

	check.Path = testURL
	check.Method = " " // Forces method check; "" automatically converts to GET
	_, err = check.validateRequest(ctx, r)
	if err == nil {
		t.Fatal(unexpected)
	}

	check.Method = http.MethodPost
	_, err = check.validateRequest(ctx, r)
	if err != nil {
		t.Fatal(err)
	}

	var passback http.Header
	check.HeaderResponse = &passback
	_, err = check.validateRequest(ctx, r)
	if err == nil {
		t.Fatal("expected error when underlying memory is not allocated")
	}
	passback = http.Header{}

	// Test setting headers
	check.Headers = map[string]string{
		"Content-Type": "Super awesome HTTP party experience",
	}

	// Test user agent set
	r.UserAgent = "r00t axxs"
	req, err := check.validateRequest(ctx, r)
	if err != nil {
		t.Fatal(err)
	}

	if req.Header.Get("Content-Type") != "Super awesome HTTP party experience" {
		t.Fatal(unexpected)
	}

	if req.UserAgent() != "r00t axxs" {
		t.Fatal(unexpected)
	}
}

type GlobalLimitTest struct {
	Auth   *rate.Limiter
	UnAuth *rate.Limiter
}

func (g *GlobalLimitTest) Limit(e EndpointLimit) error {
	switch e {
	case Auth:
		if g.Auth == nil {
			return errors.New("auth rate not set")
		}
		time.Sleep(g.Auth.Reserve().Delay())
		return nil
	case UnAuth:
		if g.UnAuth == nil {
			return errors.New("unauth rate not set")
		}
		time.Sleep(g.UnAuth.Reserve().Delay())
		return nil
	default:
		return fmt.Errorf("cannot execute functionality: %d not found", e)
	}
}

var globalshell = GlobalLimitTest{
	Auth:   NewRateLimit(time.Millisecond*600, 1),
	UnAuth: NewRateLimit(time.Second*1, 100)}

func TestDoRequest(t *testing.T) {
	t.Parallel()
	r := New("test",
		new(http.Client),
		WithLimiter(&globalshell))
	ctx := context.Background()

	err := r.SendPayload(ctx, &Item{})
	if err == nil {
		t.Fatal(unexpected)
	}
	if !strings.Contains(err.Error(), "invalid path") {
		t.Fatal(err)
	}

	err = r.SendPayload(ctx, &Item{Method: http.MethodGet})
	if err == nil {
		t.Fatal(unexpected)
	}
	if !strings.Contains(err.Error(), "invalid path") {
		t.Fatal(err)
	}

	// Invalid/missing endpoint limit
	err = r.SendPayload(ctx, &Item{
		Method: http.MethodGet,
		Path:   testURL,
	})
	if err == nil {
		t.Fatal(unexpected)
	}
	if !strings.Contains(err.Error(), "cannot execute functionality") {
		t.Fatal(err)
	}

	// force debug
	err = r.SendPayload(ctx, &Item{
		Method:        http.MethodGet,
		Path:          testURL,
		HTTPDebugging: true,
		Verbose:       true,
	})
	if err == nil {
		t.Fatal(unexpected)
	}
	if !strings.Contains(err.Error(), "cannot execute functionality") {
		t.Fatal(err)
	}

	// max request job ceiling
	r.jobs = MaxRequestJobs
	err = r.SendPayload(ctx, &Item{
		Method:   http.MethodGet,
		Path:     testURL,
		Endpoint: UnAuth,
	})
	if err == nil {
		t.Fatal(unexpected)
	}
	if !strings.Contains(err.Error(), "max request jobs reached") {
		t.Fatal(err)
	}
	// reset jobs
	r.jobs = 0

	// timeout checker
	r.HTTPClient.Timeout = time.Millisecond * 50
	err = r.SendPayload(ctx, &Item{
		Method:   http.MethodGet,
		Path:     testURL + "/timeout",
		Endpoint: UnAuth,
	})
	if err == nil {
		t.Fatal(unexpected)
	}
	if !strings.Contains(err.Error(), "failed to retry request") {
		t.Fatal(err)
	}
	// reset timeout
	r.HTTPClient.Timeout = 0

	// Check JSON
	var resp struct {
		Response bool `json:"response"`
	}

	// Check header contents
	var passback = http.Header{}
	err = r.SendPayload(ctx, &Item{
		Method:         http.MethodGet,
		Path:           testURL,
		Result:         &resp,
		Endpoint:       UnAuth,
		HeaderResponse: &passback,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Response {
		t.Fatal(unexpected)
	}

	if passback.Get("Content-Length") != "17" {
		t.Fatal("incorrect header value")
	}

	if passback.Get("Content-Type") != "application/json" {
		t.Fatal("incorrect header value")
	}

	// Check error
	var respErr struct {
		Error bool `json:"error"`
	}
	err = r.SendPayload(ctx, &Item{
		Method:   http.MethodGet,
		Path:     testURL,
		Result:   &respErr,
		Endpoint: UnAuth,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Response {
		t.Fatal(unexpected)
	}

	// Check client side rate limit
	var failed int32
	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func(wg *sync.WaitGroup) {
			var resp struct {
				Response bool `json:"response"`
			}
			payloadError := r.SendPayload(ctx, &Item{
				Method:      http.MethodGet,
				Path:        testURL + "/rate",
				Result:      &resp,
				AuthRequest: true,
				Endpoint:    Auth,
			})
			wg.Done()
			if payloadError != nil {
				atomic.StoreInt32(&failed, 1)
				log.Fatal(payloadError)
			}
			if !resp.Response {
				atomic.StoreInt32(&failed, 1)
				log.Fatal(unexpected)
			}
		}(&wg)
	}
	wg.Wait()

	if failed != 0 {
		t.Fatal("request failed")
	}
}

func TestDoRequest_Retries(t *testing.T) {
	t.Parallel()

	backoff := func(n int) time.Duration {
		return 0
	}
	r := New("test", new(http.Client), WithBackoff(backoff))
	var failed int32
	var wg sync.WaitGroup
	wg.Add(4)
	for i := 0; i < 4; i++ {
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			var resp struct {
				Response bool `json:"response"`
			}
			payloadError := r.SendPayload(context.Background(), &Item{
				Method:      http.MethodGet,
				Path:        testURL + "/rate-retry",
				Result:      &resp,
				AuthRequest: true,
				Endpoint:    Auth,
			})
			if payloadError != nil {
				atomic.StoreInt32(&failed, 1)
				log.Fatal(payloadError)
			}
			if !resp.Response {
				atomic.StoreInt32(&failed, 1)
				log.Fatal(unexpected)
			}
		}(&wg)
	}
	wg.Wait()

	if failed != 0 {
		t.Fatal("request failed")
	}
}

func TestDoRequest_RetryNonRecoverable(t *testing.T) {
	t.Parallel()

	backoff := func(n int) time.Duration {
		return 0
	}
	r := New("test", new(http.Client), WithBackoff(backoff))
	payloadError := r.SendPayload(context.Background(), &Item{
		Method: http.MethodGet,
		Path:   testURL + "/always-retry",
	})
	if payloadError == nil {
		t.Fatal("expected an error")
	}
}

func TestDoRequest_NotRetryable(t *testing.T) {
	t.Parallel()

	retry := func(resp *http.Response, err error) (bool, error) {
		return false, errors.New("not retryable")
	}
	backoff := func(n int) time.Duration {
		return time.Duration(n) * time.Millisecond
	}
	r := New("test", new(http.Client), WithRetryPolicy(retry), WithBackoff(backoff))
	payloadError := r.SendPayload(context.Background(), &Item{
		Method: http.MethodGet,
		Path:   testURL + "/always-retry",
	})
	if payloadError == nil {
		t.Fatal("expected an error")
	}
}

func TestGetNonce(t *testing.T) {
	t.Parallel()
	r := New("test",
		new(http.Client),
		WithLimiter(&globalshell))

	n1 := r.GetNonce(false)
	n2 := r.GetNonce(false)
	if n1 == n2 {
		t.Fatal(unexpected)
	}

	r2 := New("test",
		new(http.Client),
		WithLimiter(&globalshell))
	n3 := r2.GetNonce(true)
	n4 := r2.GetNonce(true)
	if n3 == n4 {
		t.Fatal(unexpected)
	}
}

func TestGetNonceMillis(t *testing.T) {
	t.Parallel()
	r := New("test",
		new(http.Client),
		WithLimiter(&globalshell))
	m1 := r.GetNonceMilli()
	m2 := r.GetNonceMilli()
	if m1 == m2 {
		log.Fatal(unexpected)
	}
}

func TestSetProxy(t *testing.T) {
	t.Parallel()
	r := New("test",
		&http.Client{Transport: new(http.Transport)},
		WithLimiter(&globalshell))
	u, err := url.Parse("http://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	err = r.SetProxy(u)
	if err != nil {
		t.Fatal(err)
	}
	u, err = url.Parse("")
	if err != nil {
		t.Fatal(err)
	}
	err = r.SetProxy(u)
	if err == nil {
		t.Fatal("error cannot be nil")
	}
}

func TestBasicLimiter(t *testing.T) {
	r := New("test",
		new(http.Client),
		WithLimiter(NewBasicRateLimit(time.Second, 1)))
	i := Item{
		Path:   "http://www.google.com",
		Method: http.MethodGet,
	}
	ctx := context.Background()

	tn := time.Now()
	_ = r.SendPayload(ctx, &i)
	_ = r.SendPayload(ctx, &i)
	if time.Since(tn) < time.Second {
		t.Error("rate limit issues")
	}
}

func TestEnableDisableRateLimit(t *testing.T) {
	r := New("TestRequest",
		new(http.Client),
		WithLimiter(NewBasicRateLimit(time.Minute, 1)))
	ctx := context.Background()

	var resp interface{}
	err := r.SendPayload(ctx, &Item{
		Method:      http.MethodGet,
		Path:        testURL,
		Result:      &resp,
		AuthRequest: true,
		Endpoint:    Auth,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = r.EnableRateLimiter()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = r.DisableRateLimiter()
	if err != nil {
		t.Fatal(err)
	}

	err = r.SendPayload(ctx, &Item{
		Method:      http.MethodGet,
		Path:        testURL,
		Result:      &resp,
		AuthRequest: true,
		Endpoint:    Auth,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = r.DisableRateLimiter()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = r.EnableRateLimiter()
	if err != nil {
		t.Fatal(err)
	}

	ti := time.NewTicker(time.Second)
	c := make(chan struct{})
	go func(c chan struct{}) {
		err = r.SendPayload(ctx, &Item{
			Method:      http.MethodGet,
			Path:        testURL,
			Result:      &resp,
			AuthRequest: true,
			Endpoint:    Auth,
		})
		if err != nil {
			log.Fatal(err)
		}
		c <- struct{}{}
	}(c)

	select {
	case <-c:
		t.Fatal("rate limiting failure")
	case <-ti.C:
		// Correct test
	}
}
