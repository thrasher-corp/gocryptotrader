package request

import (
	"net/http"
	"testing"
	"time"
)

func TestNewRateLimit(t *testing.T) {
	r := NewRateLimit(time.Second*10, 5)

	if r.Duration != time.Second*10 && r.Rate != 5 {
		t.Fatal("unexpected values")
	}
}

func TestSetRate(t *testing.T) {
	r := NewRateLimit(time.Second*10, 5)

	r.SetRate(40)
	if r.GetRate() != 40 {
		t.Fatal("unexpected values")
	}
}

func TestSetDuration(t *testing.T) {
	r := NewRateLimit(time.Second*10, 5)

	r.SetDuration(time.Second)
	if r.GetDuration() != time.Second {
		t.Fatal("unexpected values")
	}
}

func TestDecerementRequests(t *testing.T) {
	r := New("bitfinex", NewRateLimit(time.Second*10, 5), NewRateLimit(time.Second*20, 100), new(http.Client))

	r.AuthLimit.SetRequests(99)
	r.DecrementRequests(true)

	if r.AuthLimit.GetRequests() != 98 {
		t.Fatal("unexpected values")
	}
}
func TestStartCycle(t *testing.T) {
	r := New("bitfinex", NewRateLimit(time.Second*10, 5), NewRateLimit(time.Second*20, 100), new(http.Client))

	if r.AuthLimit.Duration != time.Second*10 && r.AuthLimit.Rate != 5 {
		t.Fatal("unexpected values")
	}

	if r.UnauthLimit.Duration != time.Second*20 && r.UnauthLimit.Rate != 100 {
		t.Fatal("unexpected values")
	}

	r.AuthLimit.SetRequests(1)
	r.UnauthLimit.SetRequests(1)
	r.StartCycle()
	if r.Cycle.IsZero() || r.AuthLimit.GetRequests() != 0 || r.UnauthLimit.GetRequests() != 0 {
		t.Fatal("unexpcted values")
	}
}

func TestIsRateLimited(t *testing.T) {
	r := New("bitfinex", NewRateLimit(time.Second*10, 5), NewRateLimit(time.Second*20, 100), new(http.Client))
	r.StartCycle()

	if r.AuthLimit.ToString() != "Rate limiter set to 5 requests per 10s" {
		t.Fatal("unexcpted values")
	}

	if r.UnauthLimit.ToString() != "Rate limiter set to 100 requests per 20s" {
		t.Fatal("unexpected values")
	}

	if r.AuthLimit.ToString() != "Rate limiter set to 5 requests per 10s" {
		t.Fatal("unexcpted values")
	}

	// FIXME: Need to account for unauth/auth/total requests
	r.AuthLimit.SetRequests(4)
	if r.AuthLimit.GetRequests() != 4 {
		t.Fatal("unexpected values")
	}

	// test that we're not rate limited since 4 < 5
	if r.IsRateLimited(true) {
		t.Fatal("unexpected values")
	}

	// bump requests counter to 6 which would exceed the rate limiter
	r.AuthLimit.SetRequests(6)
	if !r.IsRateLimited(true) {
		t.Fatal("unexpected values")
	}

	// FIXME: Need to account for unauth/auth/total requests
	r.UnauthLimit.SetRequests(99)
	if r.UnauthLimit.GetRequests() != 99 {
		t.Fatal("unexpected values")
	}

	// test that we're not rate limited since 99 < 100
	if r.IsRateLimited(false) {
		t.Fatal("unexpected values")
	}

	// bump requests counter to 100 which would exceed the rate limiter
	r.UnauthLimit.SetRequests(100)
	if !r.IsRateLimited(false) {
		t.Fatal("unexpected values")
	}
}

func TestRequiresRateLimiter(t *testing.T) {
	r := New("bitfinex", NewRateLimit(time.Second*10, 5), NewRateLimit(time.Second*20, 100), new(http.Client))
	if !r.RequiresRateLimiter() {
		t.Fatal("unexpected values")
	}

	r.AuthLimit.Rate = 0
	r.UnauthLimit.Rate = 0

	if r.RequiresRateLimiter() {
		t.Fatal("unexpected values")
	}
}

func TestSetLimit(t *testing.T) {
	r := New("bitfinex", NewRateLimit(time.Second*10, 5), NewRateLimit(time.Second*20, 100), new(http.Client))

	r.SetRateLimit(true, time.Minute, 20)
	if r.AuthLimit.Rate != 20 && r.AuthLimit.Duration != time.Minute*20 {
		t.Fatal("unexpected values")
	}

	r.SetRateLimit(false, time.Minute, 40)
	if r.UnauthLimit.Rate != 40 && r.UnauthLimit.Duration != time.Minute {
		t.Fatal("unexpected values")
	}
}

func TestGetLimit(t *testing.T) {
	r := New("bitfinex", NewRateLimit(time.Second*10, 5), NewRateLimit(time.Second*20, 100), new(http.Client))

	if r.GetRateLimit(true).Duration != time.Second*10 && r.GetRateLimit(true).Rate != 5 {
		t.Fatal("unexpected values")
	}

	if r.GetRateLimit(false).Duration != time.Second*10 && r.GetRateLimit(false).Rate != 100 {
		t.Fatal("unexpected values")
	}
}

func TestIsValidMethod(t *testing.T) {
	for x := range supportedMethods {
		if !IsValidMethod(supportedMethods[x]) {
			t.Fatal("unexpected values")
		}
	}

	if IsValidMethod("BLAH") {
		t.Fatal("unexpected values")
	}
}

func TestIsValidCycle(t *testing.T) {
	r := New("bitfinex", NewRateLimit(time.Second*10, 5), NewRateLimit(time.Second*20, 100), new(http.Client))
	r.Cycle = time.Now().Add(-9 * time.Second)

	if !r.IsValidCycle(true) {
		t.Fatal("unexpected values")
	}

	r.Cycle = time.Now().Add(-11 * time.Second)
	if r.IsValidCycle(true) {
		t.Fatal("unexpected values")
	}

	r.Cycle = time.Now().Add(-19 * time.Second)

	if !r.IsValidCycle(false) {
		t.Fatal("unexpected values")
	}

	r.Cycle = time.Now().Add(-21 * time.Second)
	if r.IsValidCycle(false) {
		t.Fatal("unexpected values")
	}
}

func TestCheckRequest(t *testing.T) {
	r := New("", NewRateLimit(time.Second*10, 5), NewRateLimit(time.Second*20, 100), new(http.Client))
	_, err := r.checkRequest("bad method, bad", "http://www.google.com", nil, nil)
	if err == nil {
		t.Fatal("unexpected values")
	}
}

func TestDoRequest(t *testing.T) {
	var test *Requester
	err := test.SendPayload("GET", "https://www.google.com", nil, nil, nil, false, true)
	if err == nil {
		t.Fatal("not iniitalised")
	}

	r := New("", NewRateLimit(time.Second*10, 5), NewRateLimit(time.Second*20, 100), new(http.Client))
	if err == nil {
		t.Fatal("unexpected values")
	}

	r.Name = "bitfinex"
	err = r.SendPayload("BLAH", "https://www.google.com", nil, nil, nil, false, true)
	if err == nil {
		t.Fatal("unexpected values")
	}

	err = r.SendPayload("GET", "", nil, nil, nil, false, true)
	if err == nil {
		t.Fatal("unexpected values")
	}

	err = r.SendPayload("GET", "https://www.google.com", nil, nil, nil, false, true)
	if err != nil {
		t.Fatal("unexpected values")
	}

	if !r.RequiresRateLimiter() {
		t.Fatal("unexpcted values")
	}

	r.SetRateLimit(false, time.Second, 0)
	r.SetRateLimit(true, time.Second, 0)

	err = r.SendPayload("GET", "https://www.google.com", nil, nil, nil, false, true)
	if err != nil {
		t.Fatal("unexpected values")
	}

	if r.RequiresRateLimiter() {
		t.Fatal("unexpected values")
	}

	r.SetRateLimit(false, time.Millisecond*200, 100)
	r.SetRateLimit(true, time.Millisecond*100, 100)
	r.Cycle = time.Now().Add(time.Millisecond * -201)

	if r.IsValidCycle(false) {
		t.Fatal("unexepcted values")
	}

	err = r.SendPayload("GET", "https://www.google.com", nil, nil, nil, false, true)
	if err != nil {
		t.Fatal("unexpected values")
	}

	r.Cycle = time.Now().Add(time.Millisecond * -101)

	if r.IsValidCycle(true) {
		t.Fatal("unexepcted values")
	}

	err = r.SendPayload("GET", "https://www.google.com", nil, nil, nil, true, true)
	if err != nil {
		t.Fatal("unexpected values")
	}

	var result interface{}
	err = r.SendPayload("GET", "https://www.google.com", nil, nil, result, false, true)
	if err != nil {
		t.Fatal(err)
	}

	headers := make(map[string]string)
	headers["content-type"] = "content/text"
	err = r.SendPayload("POST", "https://api.bitfinex.com", headers, nil, result, false, true)
	if err != nil {
		t.Fatal(err)
	}

	r.StartCycle()
	r.UnauthLimit.SetRequests(100)
	err = r.SendPayload("GET", "https://www.google.com", nil, nil, result, false, false)
	if err != nil {
		t.Fatal("unexpected values")
	}
}
