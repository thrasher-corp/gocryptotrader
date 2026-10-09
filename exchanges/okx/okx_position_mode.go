package okx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	positionModeCacheTTL   = 2 * time.Second
	positionModeCacheLimit = 32
)

// contractPositionMode coalesces account lookups so the configuration endpoint's
// lower rate limit does not throttle each order. Cache reuse expires two seconds
// after a successful GET; external mode changes can remain unseen during that interval.
func (e *Exchange) contractPositionMode(ctx context.Context, a asset.Item) (string, error) {
	if a != asset.Futures && a != asset.PerpetualSwap {
		return "", nil
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return "", err
	}
	store := &accounts.ContextCredentialsStore{}
	store.Load(creds)
	ctx = context.WithValue(ctx, accounts.ContextCredentialsFlag, store)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		e.positionModeMu.Lock()
		if e.positionModes == nil {
			e.positionModes = make(map[accounts.Credentials]*positionModeCacheEntry)
		}
		entry := e.positionModes[*creds]
		if entry == nil {
			entry = &positionModeCacheEntry{}
			e.positionModes[*creds] = entry
		}
		wait := entry.changed
		var lookup *positionModeLookup
		if wait == nil {
			wait = entry.ready
			lookup = entry.lookup
		}
		if wait != nil {
			e.positionModeMu.Unlock()
			select {
			case <-wait:
				if err := ctx.Err(); err != nil {
					return "", err
				}
				e.positionModeMu.Lock()
				current := lookup != nil && e.positionModes[*creds] == entry && lookup.generation == entry.generation && entry.changing == 0
				if current && lookup.err == nil {
					current = entry.lookup == lookup && time.Now().Before(entry.expires)
				}
				e.positionModeMu.Unlock()
				if current && !errors.Is(lookup.err, context.Canceled) && !errors.Is(lookup.err, context.DeadlineExceeded) {
					return lookup.mode, lookup.err
				}
				continue
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		if time.Now().Before(entry.expires) {
			mode := entry.mode
			e.positionModeMu.Unlock()
			return mode, nil
		}
		entry.ready = make(chan struct{})
		lookup = &positionModeLookup{generation: entry.generation}
		entry.lookup = lookup
		e.positionModeMu.Unlock()

		configuration, err := e.GetAccountConfiguration(ctx)
		mode := ""
		switch {
		case err != nil:
			err = fmt.Errorf("error fetching account position mode: %w", err)
		case configuration == nil:
			err = common.ErrNoResponse
		default:
			mode = configuration.PositionMode
			if mode != "net_mode" && mode != "long_short_mode" {
				err = fmt.Errorf("%w: %q", errInvalidPositionMode, mode)
			}
		}
		e.positionModeMu.Lock()
		superseded := lookup.generation != entry.generation
		if err == nil && !superseded {
			entry.mode = mode
			entry.expires = time.Now().Add(positionModeCacheTTL)
		}
		lookup.err = err
		if err == nil {
			lookup.mode = mode
		}
		close(entry.ready)
		entry.ready = nil
		e.trimPositionModeCache()
		e.positionModeMu.Unlock()
		if superseded {
			continue
		}
		if err != nil {
			return "", err
		}
		return mode, nil
	}
}

// changePositionModeCache invalidates even failed mutations because their remote
// outcome can be uncertain. In-flight lookups cannot repopulate the earlier mode.
func (e *Exchange) changePositionModeCache(creds *accounts.Credentials, starting bool) {
	e.positionModeMu.Lock()
	defer e.positionModeMu.Unlock()
	if e.positionModes == nil {
		e.positionModes = make(map[accounts.Credentials]*positionModeCacheEntry)
	}
	entry := e.positionModes[*creds]
	if entry == nil {
		entry = &positionModeCacheEntry{}
		e.positionModes[*creds] = entry
	}
	entry.expires = time.Time{}
	entry.generation++
	if starting {
		if entry.changing == 0 {
			entry.changed = make(chan struct{})
		}
		entry.changing++
	} else {
		entry.changing--
		if entry.changing == 0 {
			close(entry.changed)
			entry.changed = nil
		}
		e.trimPositionModeCache()
	}
}

// trimPositionModeCache bounds retained credential entries, keeping active
// lookups and mode changes alive until their callers finish. The mutex must be held.
func (e *Exchange) trimPositionModeCache() {
	for len(e.positionModes) > positionModeCacheLimit {
		var oldest accounts.Credentials
		var found *positionModeCacheEntry
		for creds, entry := range e.positionModes {
			if entry.ready != nil || entry.changing != 0 {
				continue
			}
			if found == nil || entry.expires.Before(found.expires) {
				oldest, found = creds, entry
			}
		}
		if found == nil {
			return
		}
		delete(e.positionModes, oldest)
	}
}
