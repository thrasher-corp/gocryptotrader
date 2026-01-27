package request

import (
	"io"
	"net/http"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/timedmutex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
)

// Const vars for rate limiter
const (
	DefaultMaxRetryAttempts = 3
	DefaultMutexLockTimeout = 50 * time.Millisecond
	drainBodyLimit          = 100000
	proxyTLSTimeout         = 15 * time.Second
	userAgent               = "User-Agent"
)

// Vars for rate limiter
var (
	MaxRetryAttempts = DefaultMaxRetryAttempts
	globalReporter   Reporter
)

// Requester struct for the request client
type Requester struct {
	_HTTPClient        *client
	limiter            RateLimitDefinitions
	reporter           Reporter
	name               string
	userAgent          string
	maxRetries         int
	Nonce              nonce.Nonce
	disableRateLimiter int32
	backoff            Backoff
	retryPolicy        RetryPolicy
	timedLock          *timedmutex.TimedMutex
}

// Item is a temp item for requests
type Item struct {
	Method                 string
	Path                   string
	Headers                map[string]string
	Body                   io.Reader
	Result                 any
	NonceEnabled           bool
	Verbose                bool
	HTTPDebugging          bool
	HTTPRecording          bool
	HTTPMockDataSliceLimit int // Limits slices per HTTP record to reduce mock data size.
	IsReserved             bool
	// HeaderResponse for inspection of header contents package side useful for
	// pagination
	HeaderResponse *http.Header
}

// Backoff determines how long to wait between request attempts.
type Backoff func(n int) time.Duration

// RetryPolicy determines whether the request should be retried.
type RetryPolicy func(resp *http.Response, err error) (bool, error)

// RequesterOption is a function option that can be applied to configure a Requester when creating it.
type RequesterOption func(*Requester)

// Generate defines a closure for functionality outside the requester to
// generate a new *http.Request on every attempt. This minimizes the chance of
// being outside the receive window if application rate limiting reduces outbound
// requests.
type Generate func() (*Item, error)
