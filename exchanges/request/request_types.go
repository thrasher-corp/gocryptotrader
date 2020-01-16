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
	DefaultMaxRequestJobs       int32 = 50
	DefaultTimeoutRetryAttempts       = 3
	DefaultMutexLockTimeout           = 50 * time.Millisecond
	proxyTLSTimeout                   = 15 * time.Second
	userAgent                         = "User-Agent"
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
	Limiter              Limiter
	Name                 string
	UserAgent            string
	timeoutRetryAttempts int
	jobs                 int32
	Nonce                nonce.Nonce
	DisableRateLimiter   bool
	timedLock            *timedmutex.TimedMutex
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
	Endpoint      EndpointLimit
}
