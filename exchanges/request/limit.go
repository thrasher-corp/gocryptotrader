package request

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// Const here define individual functionality sub types for rate limiting
const (
	Unset EndpointLimit = iota
	Auth
	UnAuth
)

// BasicLimit denotes basic rate limit that implements the Limiter interface
// does not need to set endpoint functionality.
type BasicLimit struct {
	r *rate.Limiter
}

// Limit executes a single rate limit set by NewRateLimit
func (b *BasicLimit) Limit(_ EndpointLimit) <-chan error {
	ch := make(chan error)
	go func(ch chan<- error) {
		time.Sleep(b.r.Reserve().Delay())
		ch <- nil
	}(ch)
	return ch
}

// EndpointLimit defines individual endpoint rate limits that are set when
// New is called.
type EndpointLimit int

// Limiter interface groups rate limit functionality defined in the REST
// wrapper for extended rate limiting configuration i.e. Shells of rate
// limits with a global rate for sub rates.
type Limiter interface {
	Limit(EndpointLimit) <-chan error
}

// NewRateLimit creates a new RateLimit based of time interval and how many
// actions allowed and breaks it down to an actions-per-second basis -- Burst
// rate is kept as one as this is not supported for out-bound requests.
func NewRateLimit(interval time.Duration, actions int) *rate.Limiter {
	if actions <= 0 || interval <= 0 {
		// Returns an un-restricted rate limiter
		return rate.NewLimiter(rate.Inf, 1)
	}

	i := 1 / interval.Seconds()
	rps := i * float64(actions)
	return rate.NewLimiter(rate.Limit(rps), 1)
}

// NewBasicRateLimit returns an object that implements the limiter interface
// for basic rate limit
func NewBasicRateLimit(interval time.Duration, actions int) *Limit {
	return &Limit{
		haltService:   make(chan struct{}),
		resumeService: make(chan struct{}),
		shutdown:      make(chan struct{}),
		Service:       &BasicLimit{NewRateLimit(interval, actions)},
	}
}

// NewLimit returns an object that implements the limiter interface
// for basic rate limit
func NewLimit(l Limiter) *Limit {
	return &Limit{
		Service: l,
	}
}

// Initiate determines rate limit for end service
func (l *Limit) Initiate(e EndpointLimit) error {
	if atomic.LoadInt32(&l.disableRateLimiter) == 1 {
		return nil
	}

	if l.Service != nil {
		l.wg.Add(1)
		defer l.wg.Done()

		for {
			fmt.Printf("service initiating limit. ID:%d\n", e)
			err := l.Service.Limit(e)
			select {
			case <-l.haltService:
				fmt.Printf("service halted. ID:%d\n", e)
				select {
				case <-l.shutdown:
					fmt.Printf("service shutdown. ID:%d\n", e)
					return errors.New("service shutdown")
				case <-l.resumeService:
					fmt.Printf("service resumed. ID: %d\n", e)
				}
			case err := <-err:
				fmt.Printf("service limit accessed. ID: %d\n", e)
				return err
			case <-l.shutdown:
				fmt.Printf("service shutdown. ID:%d\n", e)
				return errors.New("service shutdown")
			}
		}
	}

	return nil
}

// DisableRateLimiter disables the rate limiting system for the exchange
func (l *Limit) DisableRateLimiter() error {
	if !atomic.CompareAndSwapInt32(&l.disableRateLimiter, 0, 1) {
		return errors.New("rate limiter already disabled")
	}
	return nil
}

// EnableRateLimiter enables the rate limiting system for the exchange
func (l *Limit) EnableRateLimiter() error {
	if !atomic.CompareAndSwapInt32(&l.disableRateLimiter, 1, 0) {
		return errors.New("rate limiter already enabled")
	}
	return nil
}

// IsBackOff returns if we should be backing off sending requests
func (l *Limit) IsBackOff() bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	return l.backoff
}

// BackOff sets mode to back off
func (l *Limit) BackOff() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.backoff {
		return errors.New("already backing off")
	}
	l.backoff = true
	return nil
}

// BackOn sets mode to continue normal operations
func (l *Limit) BackOn() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if !l.backoff {
		return errors.New("already in normal operations")
	}
	l.backoff = false
	return nil
}

// Lock locks all outbound traffic for the service
func (l *Limit) Lock() {
	l.resumeService = make(chan struct{})
	close(l.haltService)
}

// Unlock enables all outbound traffic for the service
func (l *Limit) Unlock() {
	l.haltService = make(chan struct{})
	close(l.resumeService)
}

// Shutdown shuts out all outbound services
func (l *Limit) Shutdown() {
	close(l.shutdown)
	l.wg.Wait()
}
