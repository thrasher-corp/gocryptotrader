package kraken

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"sync"
	"time"
)

// wsRefreshInterval is the least time between the REST refreshes one kind of websocket message starts, so a burst of
// messages costs one refresh an interval. Every Kraken REST request shares one rate limit of a request a second, so
// refreshes are kept sparse enough to leave most of it to the requests the engine makes
const wsRefreshInterval = 5 * time.Second

// wsRefreshAttempts bounds the refreshes of a key Kraken has not settled, such as an order REST still reports as open
// after the websocket reported it finished, or one whose request failed
const wsRefreshAttempts = 5

// wsRefresher coalesces the REST refreshes websocket messages start. One refresh runs at a time, and the keys asked for
// while it runs wait for the next, which starts an interval after it
type wsRefresher struct {
	mu      sync.Mutex
	running bool
	// pending holds the keys waiting for a refresh, with the refreshes each has had without being settled
	pending map[string]int
	// settled holds when each key was settled, for a key settled within settledRetention to be passed over when asked
	// for again
	settled map[string]time.Time
	// interval is the time between refreshes, wsRefreshInterval when zero
	interval time.Duration
	// settledRetention is how long a settled key is passed over when asked for again, which a key whose refresh gives
	// the same answer every time needs; zero refreshes a key whenever it is asked for
	settledRetention time.Duration
	// active reports whether the websocket whose messages start the refreshes is still connected or connecting. A
	// refresh after the first is made only while it is, so a refresher stops once the websocket disconnects; nil means
	// always
	active func() bool
}

// refresh asks for keys to be refreshed by fn, starting a refresh when none is running. fn returns the keys it could not
// settle, which are refreshed again until each has had wsRefreshAttempts refreshes
func (r *wsRefresher) refresh(ctx context.Context, fn func(context.Context, []string) []string, keys ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pending == nil {
		r.pending = make(map[string]int)
	}
	now := time.Now()
	for k, at := range r.settled {
		if now.Sub(at) > r.settledRetention {
			delete(r.settled, k)
		}
	}
	for _, k := range keys {
		if _, ok := r.settled[k]; ok {
			continue
		}
		// A key asked for again comes with news its earlier refreshes could not have seen, so its attempts start afresh
		r.pending[k] = 0
	}
	if r.running || len(r.pending) == 0 {
		return
	}
	r.running = true
	// The refresh outlives the message that started it, whose context may end once the message is handled
	go r.run(context.WithoutCancel(ctx), fn, cmp.Or(r.interval, wsRefreshInterval))
}

// run refreshes the pending keys, then waits an interval, until no key is pending or the websocket has disconnected
func (r *wsRefresher) run(ctx context.Context, fn func(context.Context, []string) []string, interval time.Duration) {
	for first := true; ; first = false {
		r.mu.Lock()
		if len(r.pending) == 0 || !first && r.active != nil && !r.active() {
			clear(r.pending)
			r.running = false
			r.mu.Unlock()
			return
		}
		attempts := r.pending
		r.pending = make(map[string]int)
		r.mu.Unlock()
		unsettled := fn(ctx, slices.Sorted(maps.Keys(attempts)))
		r.mu.Lock()
		for k := range attempts {
			if slices.Contains(unsettled, k) {
				// A key asked for again while the refresh ran keeps its fresh attempts
				if _, asked := r.pending[k]; !asked && attempts[k]+1 < wsRefreshAttempts {
					r.pending[k] = attempts[k] + 1
				}
				continue
			}
			if r.settledRetention > 0 {
				if r.settled == nil {
					r.settled = make(map[string]time.Time)
				}
				r.settled[k] = time.Now()
				// Asked for again while it was being settled, the key has had its answer
				delete(r.pending, k)
			}
		}
		r.mu.Unlock()
		time.Sleep(interval)
	}
}
