package request

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
)

var supportedMethods = []string{http.MethodGet, http.MethodPost, http.MethodHead,
	http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodConnect}

// Const vars for rate limiter
const (
	DefaultMaxRequestJobs       = 50
	DefaultTimeoutRetryAttempts = 3

	proxyTLSTimeout = 15 * time.Second
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
	UnauthLimit          *RateLimit
	AuthLimit            *RateLimit
	Name                 string
	UserAgent            string
	Cycle                time.Time
	timeoutRetryAttempts int
	m                    sync.Mutex
	Jobs                 chan Job
	WorkerStarted        bool
	Nonce                nonce.Nonce
	fifoLock             sync.Mutex
	DisableRateLimiter   bool
}

// RateLimit struct
type RateLimit struct {
	Duration time.Duration
	Rate     int
	Requests int
	Mutex    sync.Mutex
}

// JobResult holds a request job result
type JobResult struct {
	Error  error
	Result interface{}
}

// Job holds a request job
type Job struct {
	Request       *http.Request
	Method        string
	Path          string
	Headers       map[string]string
	Body          io.Reader
	Result        interface{}
	JobResult     chan *JobResult
	AuthRequest   bool
	Verbose       bool
	HTTPDebugging bool
	Record        bool
}
