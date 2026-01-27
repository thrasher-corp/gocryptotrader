package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
)

const unexpected = "unexpected values"

var (
	testURL     string
	serverLimit *RateLimiterWithWeight
)

func TestMain(m *testing.M) {
	serverLimitInterval := time.Millisecond * 500
	serverLimit = NewWeightedRateLimitByDuration(serverLimitInterval)
	serverLimitRetry := NewWeightedRateLimitByDuration(serverLimitInterval)
	sm := http.NewServeMux()
	sm.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"response":true}`)
		if err != nil {
			log.Fatal(err)
		}
	})
	sm.HandleFunc("/error", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, `{"error":true}`)
		if err != nil {
			log.Fatal(err)
		}
	})
	sm.HandleFunc("/timeout", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(time.Millisecond * 100)
		w.WriteHeader(http.StatusGatewayTimeout)
	})
	sm.HandleFunc("/rate", func(w http.ResponseWriter, _ *http.Request) {
		if !serverLimit.limiter.Allow() {
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
	sm.HandleFunc("/rate-retry", func(w http.ResponseWriter, _ *http.Request) {
		if !serverLimitRetry.limiter.Allow() {
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
	sm.HandleFunc("/always-retry", func(w http.ResponseWriter, _ *http.Request) {
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

func TestCheckRequest(t *testing.T) {
	t.Parallel()

	r, err := New("TestRequest",
		new(http.Client))
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

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

var globalshell = RateLimitDefinitions{
	Auth:   NewWeightedRateLimitByDuration(time.Millisecond * 600),
	UnAuth: NewRateLimitWithWeight(time.Second*1, 100, 1),
}

func TestDoRequest(t *testing.T) {
	t.Parallel()
	r, err := New("test", new(http.Client), WithLimiter(globalshell))
	require.NoError(t, err, "New requester must not error")

	ctx := t.Context()
	err = (*Requester)(nil).SendPayload(ctx, Unset, nil, UnauthenticatedRequest)
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	err = r.SendPayload(ctx, Unset, nil, UnauthenticatedRequest)
	require.ErrorIs(t, err, errRequestFunctionIsNil)

	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) { return nil, nil }, UnauthenticatedRequest)
	require.ErrorIs(t, err, errRequestItemNil)

	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) { return &Item{}, nil }, UnauthenticatedRequest)
	require.ErrorIs(t, err, errInvalidPath)

	var nilHeader http.Header
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{
			Path:           testURL,
			HeaderResponse: &nilHeader,
		}, nil
	}, UnauthenticatedRequest)
	require.ErrorIs(t, err, errHeaderResponseMapIsNil)

	// Invalid/missing endpoint limit
	err = r.SendPayload(ctx, Unset, func() (*Item, error) { return &Item{Path: testURL}, nil }, UnauthenticatedRequest)
	require.ErrorIs(t, err, common.ErrNilPointer)

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
	require.NoError(t, err, "SendPayload must not error")

	// Fail new request call
	newError := errors.New("request item failure")
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return nil, newError
	}, UnauthenticatedRequest)
	require.ErrorIs(t, err, newError)

	r._HTTPClient, err = newProtectedClient(common.NewHTTPClientWithTimeout(0))
	require.NoError(t, err, "newProtectedClient must not error")

	// timeout checker
	err = r._HTTPClient.setHTTPClientTimeout(time.Millisecond * 50)
	require.NoError(t, err, "setHTTPClientTimeout must not error")
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{Path: testURL + "/timeout"}, nil
	}, UnauthenticatedRequest)
	require.ErrorIs(t, err, errFailedToRetryRequest)
	// reset timeout
	err = r._HTTPClient.setHTTPClientTimeout(0)
	require.NoError(t, err, "setHTTPClientTimeout must not error")

	// Check JSON
	var resp struct {
		Response bool `json:"response"`
	}

	// Check header contents
	passback := http.Header{}
	err = r.SendPayload(ctx, UnAuth, func() (*Item, error) {
		return &Item{
			Method:         http.MethodGet,
			Path:           testURL,
			Result:         &resp,
			HeaderResponse: &passback,
		}, nil
	}, UnauthenticatedRequest)
	require.NoError(t, err, "SendPayload must not error")

	require.Equal(t, "17", passback.Get("Content-Length"), "Content-Length must have the correct value")
	require.Equal(t, "application/json", passback.Get("Content-Type"), "Content-Type must have the correct value")

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
	require.NoError(t, err, "SendPayload must not error")
	require.False(t, respErr.Error, "Error must be false")

	// Check client side rate limit
	var ec common.ErrorCollector
	for range 5 {
		ec.Go(func() error {
			var resp struct {
				Response bool `json:"response"`
			}
			if err := r.SendPayload(ctx, Auth, func() (*Item, error) {
				return &Item{
					Method: http.MethodGet,
					Path:   testURL + "/rate",
					Result: &resp,
				}, nil
			}, AuthenticatedRequest); err != nil {
				return fmt.Errorf("SendPayload error: %w", err)
			}
			if !resp.Response {
				return fmt.Errorf("unexpected response: %+v", resp)
			}
			return nil
		})
	}

	require.NoError(t, ec.Collect(), "Collect must return no errors")
}

func TestDoRequest_Retries(t *testing.T) {
	t.Parallel()

	r, err := New("test", new(http.Client), WithBackoff(func(int) time.Duration { return 0 }))
	require.NoError(t, err, "New requester must not error")

	var ec common.ErrorCollector
	for range 4 {
		ec.Go(func() error {
			var resp struct {
				Response bool `json:"response"`
			}
			itemFn := func() (*Item, error) {
				return &Item{
					Method: http.MethodGet,
					Path:   testURL + "/rate-retry",
					Result: &resp,
				}, nil
			}

			if err := r.SendPayload(t.Context(), Auth, itemFn, AuthenticatedRequest); err != nil {
				return fmt.Errorf("SendPayload error: %w", err)
			}
			if !resp.Response {
				return fmt.Errorf("unexpected response: %+v", resp)
			}
			return nil
		})
	}

	require.NoError(t, ec.Collect(), "Collect must return no errors")
}

func TestDoRequest_RetryNonRecoverable(t *testing.T) {
	t.Parallel()

	backoff := func(int) time.Duration {
		return 0
	}
	r, err := New("test", new(http.Client), WithBackoff(backoff))
	if err != nil {
		t.Fatal(err)
	}
	err = r.SendPayload(t.Context(), Unset, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   testURL + "/always-retry",
		}, nil
	}, UnauthenticatedRequest)
	require.ErrorIs(t, err, errFailedToRetryRequest)
}

func TestDoRequest_NotRetryable(t *testing.T) {
	t.Parallel()

	notRetryErr := errors.New("not retryable")
	retry := func(*http.Response, error) (bool, error) {
		return false, notRetryErr
	}
	backoff := func(n int) time.Duration {
		return time.Duration(n) * time.Millisecond
	}
	r, err := New("test", new(http.Client), WithRetryPolicy(retry), WithBackoff(backoff))
	if err != nil {
		t.Fatal(err)
	}
	err = r.SendPayload(t.Context(), Unset, func() (*Item, error) {
		return &Item{
			Method: http.MethodGet,
			Path:   testURL + "/always-retry",
		}, nil
	}, UnauthenticatedRequest)
	require.ErrorIs(t, err, notRetryErr)
}

func TestEvaluateRetry(t *testing.T) {
	t.Parallel()

	r := Requester{}
	retry, err := r.evaluateRetry(WithRetryNotAllowed(t.Context()), nil, errInvalidPath, 1, false)
	require.ErrorIs(t, err, errInvalidPath, "must return incoming error when retry not allowed")
	require.False(t, retry, "must not retry when retry not allowed")

	r.retryPolicy = DefaultRetryPolicy
	retry, err = r.evaluateRetry(t.Context(), nil, errInvalidPath, 1, false)
	require.ErrorIs(t, err, errInvalidPath, "must return incoming error when using default retry policy")
	require.False(t, retry, "must not retry when using default retry policy and the error is non-timeout")

	retry, err = r.evaluateRetry(t.Context(), &http.Response{StatusCode: http.StatusOK}, nil, 1, false)
	require.NoError(t, err, "must not error when response is OK")
	require.False(t, retry, "must not retry on 200 status")

	errTimeout := &net.DNSError{IsTimeout: true}
	retry, err = r.evaluateRetry(t.Context(), nil, errTimeout, 1, false)
	require.ErrorIs(t, err, errFailedToRetryRequest, "must return error when attempt is higher than max retries")
	require.ErrorIs(t, err, errExceedsMaxRetries, "must return error when attempt is higher than max retries")
	require.ErrorIs(t, err, errTimeout, "must wrap original error")
	require.False(t, retry, "must not retry when max attempts exceeded")

	retry, err = r.evaluateRetry(t.Context(), &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429", Body: io.NopCloser(strings.NewReader(""))}, nil, 1, false)
	require.ErrorContains(t, err, "failed to retry request exceeds maximum retry attempts: status \"429\"", "must return error and status code when attempt is higher than max retries")
	require.False(t, retry, "must not retry when max attempts exceeded")

	r.maxRetries = 1
	r.backoff = func(int) time.Duration { return time.Millisecond * 10 }
	ctx, cancel := context.WithDeadline(t.Context(), time.Now())
	defer cancel()
	retry, err = r.evaluateRetry(ctx, nil, errTimeout, 1, false)
	require.ErrorIs(t, err, errFailedToRetryRequest, "must return error when deadline would be exceeded")
	require.ErrorIs(t, err, context.DeadlineExceeded, "must return error when deadline would be exceeded")
	require.ErrorIs(t, err, errTimeout, "must wrap original error")
	require.False(t, retry, "must not retry when deadline would be exceeded")

	retry, err = r.evaluateRetry(ctx, &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429", Body: io.NopCloser(strings.NewReader(""))}, nil, 1, false)
	require.ErrorContains(t, err, "failed to retry request context deadline exceeded: status \"429\"", "must return error and status code when attempt is higher than max retries")
	require.False(t, retry, "must not retry when deadline would be exceeded")

	ctx, cancel = context.WithCancel(t.Context())
	cancel()
	retry, err = r.evaluateRetry(ctx, &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429", Body: io.NopCloser(strings.NewReader(""))}, nil, 1, true)
	require.ErrorIs(t, err, errFailedToRetryRequest, "must return error when context is cancelled")
	require.ErrorIs(t, err, context.Canceled, "must return error when context is cancelled")
	require.False(t, retry, "must not retry when context is cancelled")

	retry, err = r.evaluateRetry(t.Context(), &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429", Body: io.NopCloser(strings.NewReader(""))}, nil, 1, true)
	require.NoError(t, err, "must not error")
	require.True(t, retry, "must retry on 429 response")
}

func TestGetNonce(t *testing.T) {
	t.Parallel()
	r, err := New("test", new(http.Client), WithLimiter(globalshell))
	require.NoError(t, err)
	n1 := r.GetNonce(nonce.Unix)
	assert.NotZero(t, n1)
	n2 := r.GetNonce(nonce.Unix)
	assert.NotZero(t, n2)
	assert.NotEqual(t, n1, n2)

	r2, err := New("test", new(http.Client), WithLimiter(globalshell))
	require.NoError(t, err)
	n3 := r2.GetNonce(nonce.UnixNano)
	assert.NotZero(t, n3)
	n4 := r2.GetNonce(nonce.UnixNano)
	assert.NotZero(t, n4)
	assert.NotEqual(t, n3, n4)

	assert.NotEqual(t, n1, n3)
	assert.NotEqual(t, n2, n4)
}

// 40532461	       30.29 ns/op	       0 B/op	       0 allocs/op (prev)
// 45329203	       26.53 ns/op	       0 B/op	       0 allocs/op
func BenchmarkGetNonce(b *testing.B) {
	r, err := New("test", new(http.Client), WithLimiter(globalshell))
	require.NoError(b, err)
	for b.Loop() {
		r.GetNonce(nonce.UnixNano)
		r.timedLock.UnlockIfLocked()
	}
}

func TestSetProxy(t *testing.T) {
	t.Parallel()
	var r *Requester
	err := r.SetProxy(nil)
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	r, err = New("test", &http.Client{Transport: new(http.Transport)}, WithLimiter(globalshell))
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
	r, err := New("test", new(http.Client), WithLimiter(NewBasicRateLimit(time.Second, 1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	i := Item{Path: "http://www.google.com", Method: http.MethodGet}
	ctx := t.Context()

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
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestEnableDisableRateLimit(t *testing.T) {
	r, err := New("TestRequest", new(http.Client), WithLimiter(NewBasicRateLimit(50*time.Millisecond, 1, 1)))
	require.NoError(t, err, "New requester must not error")

	sendIt := func() error {
		return r.SendPayload(t.Context(), Auth, func() (*Item, error) {
			return &Item{
				Method: http.MethodGet,
				Path:   testURL,
				Result: new(any),
			}, nil
		}, AuthenticatedRequest)
	}

	// allow initial request
	require.NoError(t, sendIt(), "sendIt must not error")

	// error on redundant enable
	assert.ErrorIs(t, r.EnableRateLimiter(), ErrRateLimiterAlreadyEnabled)

	// error on redundant disable
	require.NoError(t, r.DisableRateLimiter(), "DisableRateLimiter must not error")
	assert.ErrorIs(t, r.DisableRateLimiter(), ErrRateLimiterAlreadyDisabled)

	// allow requests when disabled
	require.NoError(t, sendIt(), "sendIt must not error")

	// allow when re-enabled
	require.NoError(t, r.EnableRateLimiter(), "EnableRateLimiter must succeed")
	require.NoError(t, sendIt(), "sendIt must not error")

	// block excess requests
	require.NoError(t, sendIt(), "sendIt must not error") // consume the one token
	start := time.Now()
	err = sendIt() // this should block until a token is refilled
	require.NoError(t, err, "sendIt must not error")
	elapsed := time.Since(start)
	assert.GreaterOrEqualf(t, elapsed.Milliseconds(), int64(20), "Expected sendIt to block for at least 20ms, but it returned after %dms", elapsed.Milliseconds())
}

func TestSetHTTPClient(t *testing.T) {
	var r *Requester
	err := r.SetHTTPClient(nil)
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	client := new(http.Client)
	r = new(Requester)
	err = r.SetHTTPClient(client)
	require.NoError(t, err)

	err = r.SetHTTPClient(client)
	require.ErrorIs(t, err, errCannotReuseHTTPClient)
}

func TestSetHTTPClientTimeout(t *testing.T) {
	var r *Requester
	err := r.SetHTTPClientTimeout(0)
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	r = new(Requester)
	err = r.SetHTTPClient(common.NewHTTPClientWithTimeout(2))
	if err != nil {
		t.Fatal(err)
	}
	err = r.SetHTTPClientTimeout(time.Second)
	require.NoError(t, err)
}

func TestSetHTTPClientUserAgent(t *testing.T) {
	var r *Requester
	err := r.SetHTTPClientUserAgent("")
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	r = new(Requester)
	err = r.SetHTTPClientUserAgent("")
	require.NoError(t, err)
}

func TestGetHTTPClientUserAgent(t *testing.T) {
	var r *Requester
	_, err := r.GetHTTPClientUserAgent()
	require.ErrorIs(t, err, ErrRequestSystemIsNil)

	r = new(Requester)
	err = r.SetHTTPClientUserAgent("sillyness")
	require.NoError(t, err)

	ua, err := r.GetHTTPClientUserAgent()
	require.NoError(t, err)

	if ua != "sillyness" {
		t.Fatal("unexpected value")
	}
}

func TestGetRateLimiterDefinitions(t *testing.T) {
	t.Parallel()
	require.Equal(t, RateLimitDefinitions(nil), (*Requester)(nil).GetRateLimiterDefinitions())
	r, err := New("test", new(http.Client), WithLimiter(globalshell))
	require.NoError(t, err)
	require.NotEmpty(t, r.GetRateLimiterDefinitions())
	assert.Equal(t, globalshell, r.GetRateLimiterDefinitions())
}
