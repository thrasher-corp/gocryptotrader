package request

import (
	"io"
	"net/http"
	"sync"
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
)

// Requester struct for the request client
type Requester struct {
	HTTPClient           *http.Client
	Limit                *Limit
	Name                 string
	UserAgent            string
	timeoutRetryAttempts int
	jobs                 int32
	Nonce                nonce.Nonce
	timedLock            *timedmutex.TimedMutex
	shutdown             chan struct{}
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

// Limit defines and determines a request routes path limits
type Limit struct {
	disableRateLimiter int32
	Service            Limiter
	outboundTraffic    sync.Mutex

	backoff bool

	haltService   chan struct{}
	upperShell    sync.WaitGroup
	lowerShell    sync.WaitGroup
	resumeService chan struct{}
	shutdown      chan struct{}
	// makes sure all items pending on stack get cancelled
	wg     sync.WaitGroup
	mtx    sync.Mutex
	inside sync.RWMutex
}
