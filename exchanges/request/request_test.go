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

	"github.com/thrasher-corp/gocryptotrader/common"
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
		_, err := io.WriteString(w, `{"response":true}`)
		if err != nil {
			log.Fatal(err)
		}
	})
	sm.HandleFunc("/error", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, `{"error":true}`)
		if err != nil {
			log.Fatal(err)
		}
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
			_, err := io.WriteString(w, `{"response":false}`)
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		_, err := io.WriteString(w, `{"response":true}`)
		if err != nil {
			log.Fatal(err)
		}
	})
	sm.HandleFunc("/rate-retry", func(w http.ResponseWriter, req *http.Request) {
		if !serverLimitRetry.Allow() {
			w.Header().Add("Retry-After", strconv.Itoa(int(math.Round(serverLimitInterval.Seconds()))))
			http.Error(w,
				http.StatusText(http.StatusTooManyRequests),
				http.StatusTooManyRequests)
			_, err := io.WriteString(w, `{"response":false}`)
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		_, err := io.WriteString(w, `{"response":true}`)
		if err != nil {
			log.Fatal(err)
		}
	})
	sm.HandleFunc("/always-retry", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Retry-After", time.Now().Format(time.RFC1123))
		w.WriteHeader(http.StatusTooManyRequests)
		_, err := io.WriteString(w, `{"response":false}`)
		if err != nil {
			log.Fatal(err)
		}
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

	r, err := New("TestRequest",
		new(http.Client))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	var check *Item
	_, err = check.validateRequest(ctx, &Requester{})
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
	r.userAgent = "r00t axxs"
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

var errEndpointLimitNotFound = errors.New("endpoint limit not found")

func (g *GlobalLimitTest) Limit(ctx context.Context, e EndpointLimit) error {
	switch e {
	case Auth:
		if g.Auth == nil {
			return errors.New("auth rate not set")
		}
		return g.Auth.Wait(ctx)
	case UnAuth:
		if g.UnAuth == nil {
			return errors.New("unauth rate not set")
		}
		return g.UnAuth.Wait(ctx)
	default:
		return fmt.Errorf("cannot execute functionality: %d %w",
			e,
			errEndpointLimitNotFound)
	}
}

var globalshell = GlobalLimitTest{
	Auth:   NewRateLimit(time.Millisecond*600, 1),
	UnAuth: NewRateLimit(time.Second*1, 100)}

func TestDoRequest(t *testing.T) {
	t.Parallel()
	r, err := New("test", new(http.Client), WithLimiter(&globalshell))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = (*Requester)(nil).SendPayload(ctx, Unset, nil, UnauthenticatedRequest)
	if !errors.Is(ErrRequestSystemIsNil, err) {
		t.Fatalf("expected: %v but received: %v", ErrRequestSystemIsNil, err)
	}
	err = r.SendPayload(ctx, Unset, nil, UnauthenticatedRequest)
	if !errors.Is(errRequestFunctionIsNil, err) {
		t.Fatalf("expected: %v but received: %v", errRequestFunctionIsNil, err)
	}

	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) { return nil, nil }, UnauthenticatedRequest)
	if !errors.Is(errRequestItemNil, err) {
		t.Fatalf("expected: %v but received: %v", errRequestItemNil, err)
	}

	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) { return &Item{}, nil }, UnauthenticatedRequest)
	if !errors.Is(errInvalidPath, err) {
		t.Fatalf("expected: %v but received: %v", errInvalidPath, err)
	}

	var nilHeader http.Header
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{
			Path:           testURL,
			HeaderResponse: &nilHeader,
		}, nil
	}, UnauthenticatedRequest)
	if !errors.Is(errHeaderResponseMapIsNil, err) {
		t.Fatalf("expected: %v but received: %v", errHeaderResponseMapIsNil, err)
	}

	// Invalid/missing endpoint limit
	err = r.SendPayload(ctx, Unset, func() (*Item, error) {
		return &Item{
			Path: testURL,
		}, nil
	}, UnauthenticatedRequest)
	if !errors.Is(err, errEndpointLimitNotFound) {
		t.Fatalf("expected: %v but received: %v", errEndpointLimitNotFound, err)
	}

	// Force debug
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{
			Path: testURL,
			Headers: map[string]string{
				"test": "supertest",
			},
			Body:          strings.NewReader("test"),
			HTTPDebugging: true,
			Verbose:       true,
		}, nil
	}, UnauthenticatedRequest)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	// Fail new request call
	newError := errors.New("request item failure")
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return nil, newError
	}, UnauthenticatedRequest)
	if !errors.Is(err, newError) {
		t.Fatalf("received: %v but expected: %v", err, newError)
	}

	// max request job ceiling
	r.jobs = MaxRequestJobs
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{Path: testURL}, nil
	}, UnauthenticatedRequest)
	if !errors.Is(err, errMaxRequestJobs) {
		t.Fatalf("received: %v but expected: %v", err, errMaxRequestJobs)
	}
	// reset jobs
	r.jobs = 0

	r._HTTPClient, err = newProtectedClient(common.NewHTTPClientWithTimeout(0))
	if err != nil {
		t.Fatal(err)
	}

	// timeout checker
	err = r._HTTPClient.setHTTPClientTimeout(time.Millisecond * 50)
	if err != nil {
		t.Fatal(err)
	}
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{Path: testURL + "/timeout"}, nil
	}, UnauthenticatedRequest)
	if !errors.Is(err, errFailedToRetryRequest) {
		t.Fatalf("received: %v but expected: %v", err, errFailedToRetryRequest)
	}
	// reset timeout
	err = r._HTTPClient.setHTTPClientTimeout(0)
	if err != nil {
		t.Fatal(err)
	}

	// Check JSON
	var resp struct {
		Response bool `json:"response"`
	}

	// Check header contents
	var passback = http.Header{}
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{
			Method:         http.MethodGet,
			Path:           testURL,
			Result:         &resp,
			HeaderResponse: &passback,
		}, nil
	}, UnauthenticatedRequest)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
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
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   testURL,
			Result: &respErr,
		}, nil
	}, UnauthenticatedRequest)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if respErr.Error {
		t.Fatal("unexpected value")
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
			payloadError := r.SendPayload(ctx, Auth, func() (*Item, error) {
				return &Item{
					Method: http.MethodGet,
					Path:   testURL + "/rate",
					Result: &resp,
				}, nil
			}, AuthenticatedRequest)
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
	r, err := New("test", new(http.Client), WithBackoff(backoff))
	if err != nil {
		t.Fatal(err)
	}
	var failed int32
	var wg sync.WaitGroup
	wg.Add(4)
	for i := 0; i < 4; i++ {
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			var resp struct {
				Response bool `json:"response"`
			}
			payloadError := r.SendPayload(context.Background(), Auth, func() (*Item, error) {
				return &Item{
					Method: http.MethodGet,
					Path:   testURL + "/rate-retry",
					Result: &resp,
				}, nil
			}, AuthenticatedRequest)
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
	r, err := New("test", new(http.Client), WithBackoff(backoff))
	if err != nil {
		t.Fatal(err)
	}
	err = r.SendPayload(context.Background(), Unset, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   testURL + "/always-retry",
		}, nil
	}, UnauthenticatedRequest)
	if !errors.Is(err, errFailedToRetryRequest) {
		t.Fatalf("received: %v but expected: %v", err, errFailedToRetryRequest)
	}
}

func TestDoRequest_NotRetryable(t *testing.T) {
	t.Parallel()

	notRetryErr := errors.New("not retryable")
	retry := func(resp *http.Response, err error) (bool, error) {
		return false, notRetryErr
	}
	backoff := func(n int) time.Duration {
		return time.Duration(n) * time.Millisecond
	}
	r, err := New("test", new(http.Client), WithRetryPolicy(retry), WithBackoff(backoff))
	if err != nil {
		t.Fatal(err)
	}
	err = r.SendPayload(context.Background(), Unset, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   testURL + "/always-retry",
		}, nil
	}, UnauthenticatedRequest)
	if !errors.Is(err, notRetryErr) {
		t.Fatalf("received: %v but expected: %v", err, notRetryErr)
	}
}

func TestGetNonce(t *testing.T) {
	t.Parallel()
	r, err := New("test",
		new(http.Client),
		WithLimiter(&globalshell))
	if err != nil {
		t.Fatal(err)
	}
	if n1, n2 := r.GetNonce(false), r.GetNonce(false); n1 == n2 {
		t.Fatal(unexpected)
	}

	r2, err := New("test",
		new(http.Client),
		WithLimiter(&globalshell))
	if err != nil {
		t.Fatal(err)
	}
	if n1, n2 := r2.GetNonce(true), r2.GetNonce(true); n1 == n2 {
		t.Fatal(unexpected)
	}
}

func TestGetNonceMillis(t *testing.T) {
	t.Parallel()
	r, err := New("test",
		new(http.Client),
		WithLimiter(&globalshell))
	if err != nil {
		t.Fatal(err)
	}
	if m1, m2 := r.GetNonceMilli(), r.GetNonceMilli(); m1 == m2 {
		log.Fatal(unexpected)
	}
}

func TestSetProxy(t *testing.T) {
	t.Parallel()
	var r *Requester
	err := r.SetProxy(nil)
	if !errors.Is(err, ErrRequestSystemIsNil) {
		t.Fatalf("received: '%v', but expected: '%v'", err, ErrRequestSystemIsNil)
	}
	r, err = New("test",
		&http.Client{Transport: new(http.Transport)},
		WithLimiter(&globalshell))
	if err != nil {
		t.Fatal(err)
	}
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
	r, err := New("test",
		new(http.Client),
		WithLimiter(NewBasicRateLimit(time.Second, 1)))
	if err != nil {
		t.Fatal(err)
	}
	i := Item{
		Path:   "http://www.google.com",
		Method: http.MethodGet,
	}
	ctx := context.Background()

	tn := time.Now()
	err = r.SendPayload(ctx, Unset, func() (*Item, error) { return &i, nil }, UnauthenticatedRequest)
	if err != nil {
		t.Fatal(err)
	}
	err = r.SendPayload(ctx, Unset, func() (*Item, error) { return &i, nil }, UnauthenticatedRequest)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(tn) < time.Second {
		t.Error("rate limit issues")
	}

	ctx, cancel := context.WithDeadline(ctx, tn.Add(time.Nanosecond))
	defer cancel()
	err = r.SendPayload(ctx, Unset, func() (*Item, error) { return &i, nil }, UnauthenticatedRequest)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("received: %v but expected: %v", err, context.DeadlineExceeded)
	}
}

func TestEnableDisableRateLimit(t *testing.T) {
	r, err := New("TestRequest",
		new(http.Client),
		WithLimiter(NewBasicRateLimit(time.Minute, 1)))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	var resp interface{}
	err = r.SendPayload(ctx, Auth, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   testURL,
			Result: &resp,
		}, nil
	}, AuthenticatedRequest)
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

	err = r.SendPayload(ctx, Auth, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   testURL,
			Result: &resp,
		}, nil
	}, AuthenticatedRequest)
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
		err = r.SendPayload(ctx, Auth, func() (*Item, error) {
			return &Item{
				Method: http.MethodGet,
				Path:   testURL,
				Result: &resp,
			}, nil
		}, AuthenticatedRequest)
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

func TestSetHTTPClient(t *testing.T) {
	var r *Requester
	err := r.SetHTTPClient(nil)
	if !errors.Is(err, ErrRequestSystemIsNil) {
		t.Fatalf("received: '%v', but expected: '%v'", err, ErrRequestSystemIsNil)
	}
	client := new(http.Client)
	r = new(Requester)
	err = r.SetHTTPClient(client)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected: '%v'", err, nil)
	}
	err = r.SetHTTPClient(client)
	if !errors.Is(err, errCannotReuseHTTPClient) {
		t.Fatalf("received: '%v', but expected: '%v'", err, errCannotReuseHTTPClient)
	}
}

func TestSetHTTPClientTimeout(t *testing.T) {
	var r *Requester
	err := r.SetHTTPClientTimeout(0)
	if !errors.Is(err, ErrRequestSystemIsNil) {
		t.Fatalf("received: '%v', but expected: '%v'", err, ErrRequestSystemIsNil)
	}
	r = new(Requester)
	err = r.SetHTTPClient(common.NewHTTPClientWithTimeout(2))
	if err != nil {
		t.Fatal(err)
	}
	err = r.SetHTTPClientTimeout(time.Second)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected: '%v'", err, nil)
	}
}

func TestSetHTTPClientUserAgent(t *testing.T) {
	var r *Requester
	err := r.SetHTTPClientUserAgent("")
	if !errors.Is(err, ErrRequestSystemIsNil) {
		t.Fatalf("received: '%v', but expected: '%v'", err, ErrRequestSystemIsNil)
	}
	r = new(Requester)
	err = r.SetHTTPClientUserAgent("")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected: '%v'", err, nil)
	}
}

func TestGetHTTPClientUserAgent(t *testing.T) {
	var r *Requester
	_, err := r.GetHTTPClientUserAgent()
	if !errors.Is(err, ErrRequestSystemIsNil) {
		t.Fatalf("received: '%v', but expected: '%v'", err, ErrRequestSystemIsNil)
	}
	r = new(Requester)
	err = r.SetHTTPClientUserAgent("sillyness")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected: '%v'", err, nil)
	}
	ua, err := r.GetHTTPClientUserAgent()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v', but expected: '%v'", err, nil)
	}
	if ua != "sillyness" {
		t.Fatal("unexpected value")
	}
}

func TestContextVerbosity(t *testing.T) {
	t.Parallel()
	if isVerbose(context.Background(), false) {
		t.Fatal("unexpected value")
	}

	if !isVerbose(context.Background(), true) {
		t.Fatal("unexpected value")
	}

	ctx := context.Background()
	ctx = WithVerbose(ctx)
	if !isVerbose(ctx, false) {
		t.Fatal("unexpected value")
	}

	ctx = context.Background()
	ctx = context.WithValue(ctx, contextVerboseFlag, false)
	if isVerbose(ctx, false) {
		t.Fatal("unexpected value")
	}

	ctx = context.Background()
	ctx = context.WithValue(ctx, contextVerboseFlag, "bruh")
	if isVerbose(ctx, false) {
		t.Fatal("unexpected value")
	}
}
