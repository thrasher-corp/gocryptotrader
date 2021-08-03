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
	DefaultMaxRequestJobs   int32 = 50
	DefaultMaxRetryAttempts       = 3
	DefaultMutexLockTimeout       = 50 * time.Millisecond
	drainBodyLimit                = 100000
	proxyTLSTimeout               = 15 * time.Second
	userAgent                     = "User-Agent"
)

// Vars for rate limiter
var (
	MaxRequestJobs   = DefaultMaxRequestJobs
	MaxRetryAttempts = DefaultMaxRetryAttempts
)

// Requester struct for the request client
type Requester struct {
	HTTPClient         *http.Client
	limiter            Limiter
	Name               string
	UserAgent          string
	maxRetries         int
	jobs               int32
	Nonce              nonce.Nonce
	disableRateLimiter int32
	backoff            Backoff
	retryPolicy        RetryPolicy
	timedLock          *timedmutex.TimedMutex
}

// Item is a temp item for requests
type Item struct {
	Method        string
	Path          string
	Headers       map[string]string
	Body          io.Reader
	Result        interface{}
	AuthRequest   bool
	NonceEnabled  bool
	Verbose       bool
	HTTPDebugging bool
	HTTPRecording bool
	IsReserved    bool
	// HeaderResponse for inspection of header contents package side useful for
	// pagination
	HeaderResponse *http.Header
	Endpoint       EndpointLimit
}

// Backoff determines how long to wait between request attempts.
type Backoff func(n int) time.Duration

// RetryPolicy determines whether the request should be retried.
type RetryPolicy func(resp *http.Response, err error) (bool, error)

// RequesterOption is a function option that can be applied to configure a Requester when creating it.
type RequesterOption func(*Requester)
