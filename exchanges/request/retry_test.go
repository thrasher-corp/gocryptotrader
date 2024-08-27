package request_test

import (
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestDefaultRetryPolicy(t *testing.T) {
	t.Parallel()
	type args struct {
		Error    error
		Response *http.Response
	}
	type want struct {
		Error error
		Retry bool
	}
	testTable := map[string]struct {
		Args args
		Want want
	}{
		"DNS Error": {
			Args: args{Error: &net.DNSError{Err: "fake"}},
			Want: want{Error: &net.DNSError{Err: "fake"}},
		},
		"DNS Timeout": {
			Args: args{Error: &net.DNSError{Err: "fake", IsTimeout: true}},
			Want: want{Retry: true},
		},
		"Too Many Requests": {
			Args: args{Response: &http.Response{StatusCode: http.StatusTooManyRequests}},
			Want: want{Retry: true},
		},
		"Not Found": {
			Args: args{Response: &http.Response{StatusCode: http.StatusNotFound}},
		},
		"Retry After": {
			Args: args{Response: &http.Response{StatusCode: http.StatusTeapot, Header: http.Header{"Retry-After": []string{"0.5"}}}},
			Want: want{Retry: true},
		},
	}

	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			retry, err := request.DefaultRetryPolicy(tt.Args.Response, tt.Args.Error)

			if exp := tt.Want.Error; exp != nil {
				if !reflect.DeepEqual(err, exp) {
					t.Fatalf("unexpected error\nexp: %#v, got: %#v", exp, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error\nexp: <nil>, got: %#v", err)
			}

			if tt.Want.Retry != retry {
				t.Fatalf("incorrect retry flag\nexp: %v, got: %v", tt.Want.Retry, retry)
			}
		})
	}
}

func TestRetryAfter(t *testing.T) {
	t.Parallel()
	now := time.Date(2020, time.April, 20, 13, 31, 13, 0, time.UTC)

	type args struct {
		Now      time.Time
		Response *http.Response
	}
	type want struct {
		Delay time.Duration
	}
	testTable := map[string]struct {
		Args args
		Want want
	}{
		"No Response": {},
		"Empty Header": {
			Args: args{Response: &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": []string{""}}}},
		},
		"Partial Seconds": {
			Args: args{Response: &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": []string{"0.5"}}}},
		},
		"Delay Seconds": {
			Args: args{Response: &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": []string{"3"}}}},
			Want: want{Delay: 3 * time.Second},
		},
		"Invalid HTTP Date RFC3339": {
			Args: args{
				Now:      now,
				Response: &http.Response{StatusCode: http.StatusTeapot, Header: http.Header{"Retry-After": []string{"2020-04-02T13:31:18Z"}}},
			},
		},
		"Valid HTTP Date": {
			Args: args{
				Now:      now,
				Response: &http.Response{StatusCode: http.StatusTeapot, Header: http.Header{"Retry-After": []string{"Mon, 20 Apr 2020 13:31:18 GMT"}}},
			},
			Want: want{Delay: 5 * time.Second},
		},
	}

	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			delay := request.RetryAfter(tt.Args.Response, tt.Args.Now)

			if exp := tt.Want.Delay; delay != exp {
				t.Fatalf("unexpected delay\nexp: %v, got: %v", exp, delay)
			}
		})
	}
}
