package request

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

const unexpected = "unexpected values"

var testURL string
var serverLimit *rate.Limiter

func TestMain(m *testing.M) {
	serverLimit = NewRateLimit(time.Millisecond*500, 1)
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
		new(http.Client),
		nil)

	var check *Item
	_, err := check.validateRequest(&Requester{})
	if err == nil {
		t.Fatal(unexpected)
	}

	_, err = check.validateRequest(nil)
	if err == nil {
		t.Fatal(unexpected)
	}

	_, err = check.validateRequest(r)
	if err == nil {
		t.Fatal(unexpected)
	}

	check = &Item{}
	_, err = check.validateRequest(r)
	if err == nil {
		t.Fatal(unexpected)
	}

	check.Path = testURL
	check.Method = " " // Forces method check; "" automatically converts to GET
	_, err = check.validateRequest(r)
	if err == nil {
		t.Fatal(unexpected)
	}

	check.Method = http.MethodPost
	_, err = check.validateRequest(r)
	if err != nil {
		t.Fatal(err)
	}

	// Test setting headers
	check.Headers = map[string]string{
		"Content-Type": "Super awesome HTTP party experience",
	}

	// Test user agent set
	r.UserAgent = "r00t axxs"
	req, err := check.validateRequest(r)
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

func (g *GlobalLimitTest) Limit(f Functionality) error {
	switch f {
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
		return fmt.Errorf("cannot execute functionality: %d not found", f)
	}
}

var globalshell = GlobalLimitTest{
	Auth:   NewRateLimit(time.Millisecond*600, 1),
	UnAuth: NewRateLimit(time.Second*1, 100)}

func TestDoRequest(t *testing.T) {
	t.Parallel()
	r := New("test",
		new(http.Client),
		&globalshell)

	err := r.SendPayload(&Item{})
	if err == nil {
		t.Fatal(unexpected)
	}

	err = r.SendPayload(&Item{Method: http.MethodGet})
	if err == nil {
		t.Fatal(unexpected)
	}

	err = r.SendPayload(&Item{
		Method: http.MethodGet,
		Path:   testURL,
	})
	if err == nil {
		t.Fatal(unexpected)
	}

	// force debug
	err = r.SendPayload(&Item{
		Method:        http.MethodGet,
		Path:          testURL,
		HTTPDebugging: true,
		Verbose:       true,
	})
	if err == nil {
		t.Fatal(unexpected)
	}

	// max request job ceiling
	r.jobs = MaxRequestJobs
	err = r.SendPayload(&Item{
		Method: http.MethodGet,
		Path:   testURL,
	})
	if err == nil {
		t.Fatal(unexpected)
	}
	// reset jobs
	r.jobs = 0

	// timeout checker
	r.HTTPClient.Timeout = time.Millisecond * 50
	err = r.SendPayload(&Item{
		Method: http.MethodGet,
		Path:   testURL + "/timeout",
	})
	if err == nil {
		t.Fatal(unexpected)
	}
	// reset timeout
	r.HTTPClient.Timeout = 0

	// Check JSON
	var resp struct {
		Response bool `json:"response"`
	}
	err = r.SendPayload(&Item{
		Method:   http.MethodGet,
		Path:     testURL,
		Result:   &resp,
		Endpoint: UnAuth,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Response {
		t.Fatal(unexpected)
	}

	// Check error
	var respErr struct {
		Error bool `json:"error"`
	}
	err = r.SendPayload(&Item{
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

	// Check rate limit
	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func(wg *sync.WaitGroup) {
			var resp struct {
				Response bool `json:"response"`
			}
			err = r.SendPayload(&Item{
				Method:      http.MethodGet,
				Path:        testURL + "/rate",
				Result:      &resp,
				AuthRequest: true,
				Endpoint:    Auth,
			})
			wg.Done()
			if err != nil {
				log.Fatal(err)
			}
			if !resp.Response {
				log.Fatal(unexpected)
			}
		}(&wg)
	}
	wg.Wait()
}

func TestGetNonce(t *testing.T) {
	t.Parallel()
	r := New("test",
		new(http.Client),
		&globalshell)

	n1 := r.GetNonce(false)
	n2 := r.GetNonce(false)
	if n1 == n2 {
		t.Fatal(unexpected)
	}

	r2 := New("test",
		new(http.Client),
		&globalshell)
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
		&globalshell)
	m1 := r.GetNonceMilli()
	m2 := r.GetNonceMilli()
	if m1 == m2 {
		log.Fatal(unexpected)
	}
}

func TestSetProxy(t *testing.T) {
	t.Parallel()
	r := New("test",
		new(http.Client),
		&globalshell)
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

func TestEnableDisableRateLimiter(t *testing.T) {
	t.Parallel()
	r := New("test",
		new(http.Client),
		&globalshell)

	r.DisableRateLimit()
	if !r.DisableRateLimiter {
		t.Fatal(unexpected)
	}

	r.EnableRateLimit()
	if r.DisableRateLimiter {
		t.Fatal(unexpected)
	}
}
