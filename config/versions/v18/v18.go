// Package v18 moves Kraken's websocket endpoints to the paths websocket v2 is served on, as v1 is no longer used, and
// adds the default futures websocket subscriptions to configs holding the previous default subscriptions.
package v18

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
)

// v1Endpoints are the websocket v1 endpoints Kraken serves websocket v2 from at /v2
var v1Endpoints = []string{
	"wss://ws.kraken.com",
	"wss://ws-auth.kraken.com",
	"wss://beta-ws.kraken.com",
	"wss://beta-ws-auth.kraken.com",
}

// endpointKeys are the configured endpoints Kraken's websockets connect to
var endpointKeys = []string{"WebsocketSpotURL", "WebsocketSpotSupplementaryURL"}

// subscription is a default subscription, matched against a saved subscription as the runtime reads it
type subscription struct {
	channel       string
	asset         string
	interval      time.Duration
	levels        int
	authenticated bool
}

// previousSubscriptions are Kraken's default subscriptions before futures websocket support
var previousSubscriptions = []subscription{
	{channel: "ticker", asset: "spot"},
	{channel: "allTrades", asset: "spot"},
	{channel: "candles", asset: "spot", interval: time.Minute},
	{channel: "orderbook", asset: "spot", levels: 1000},
	{channel: "myOrders", authenticated: true},
	{channel: "myTrades", authenticated: true},
}

// futuresSubscriptions are the default subscriptions futures websocket support adds
var futuresSubscriptions = []subscription{
	{channel: "ticker", asset: "futures"},
	{channel: "allTrades", asset: "futures"},
	{channel: "orderbook", asset: "futures"},
	{channel: "myOrders", asset: "futures", authenticated: true},
	{channel: "myTrades", asset: "futures", authenticated: true},
}

// savedSubscription is a subscription as a config saves it
type savedSubscription struct {
	Enabled       bool            `json:"enabled"`
	Channel       string          `json:"channel,omitempty"`
	Pairs         string          `json:"pairs,omitempty"`
	Asset         string          `json:"asset,omitempty"`
	Params        map[string]any  `json:"params,omitempty"`
	Interval      json.RawMessage `json:"interval,omitempty"`
	Levels        int             `json:"levels,omitempty"`
	Authenticated bool            `json:"authenticated,omitempty"`
}

// Version implements ExchangeVersion for Kraken's websocket v2 endpoints and futures websocket subscriptions.
type Version struct{}

// Exchanges returns just Kraken.
func (*Version) Exchanges() []string { return []string{"Kraken"} }

// UpgradeExchange moves Kraken's websocket v1 endpoints to their v2 paths, and adds the futures subscriptions to the
// previous default subscriptions. Other endpoints and subscriptions are kept, as they were chosen by the user.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	exchange, err := migrateEndpoints(exchange, func(endpoint string) (string, bool) {
		endpoint = strings.TrimSuffix(endpoint, "/")
		return endpoint + "/v2", slices.Contains(v1Endpoints, endpoint)
	})
	if err != nil {
		return exchange, err
	}
	return migrateSubscriptions(exchange, true)
}

// DowngradeExchange moves Kraken's websocket v2 endpoints back to their v1 paths, and removes the futures
// subscriptions from the default subscriptions.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	exchange, err := migrateEndpoints(exchange, func(endpoint string) (string, bool) {
		endpoint, ok := strings.CutSuffix(strings.TrimSuffix(endpoint, "/"), "/v2")
		return endpoint, ok && slices.Contains(v1Endpoints, endpoint)
	})
	if err != nil {
		return exchange, err
	}
	return migrateSubscriptions(exchange, false)
}

// migrateEndpoints replaces each websocket endpoint migrate returns a replacement for
func migrateEndpoints(exchange []byte, migrate func(string) (string, bool)) ([]byte, error) {
	for _, key := range endpointKeys {
		value, valueType, _, err := jsonparser.Get(exchange, "api", "urlEndpoints", key)
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			continue
		}
		if err != nil {
			return exchange, fmt.Errorf("error getting Kraken %s: %w", key, err)
		}
		if valueType != jsonparser.String {
			continue
		}
		endpoint, err := jsonparser.ParseString(value)
		if err != nil {
			return exchange, fmt.Errorf("error parsing Kraken %s: %w", key, err)
		}
		replacement, ok := migrate(endpoint)
		if !ok {
			continue
		}
		if exchange, err = jsonparser.Set(exchange, []byte(strconv.Quote(replacement)), "api", "urlEndpoints", key); err != nil {
			return exchange, fmt.Errorf("error setting Kraken %s: %w", key, err)
		}
	}
	return exchange, nil
}

// migrateSubscriptions adds the futures subscriptions to subscriptions that are exactly the previous defaults on
// upgrade, and removes them from subscriptions that are exactly the current defaults on downgrade. Kraken gives a config
// without subscriptions the defaults when it is set up, so such a config needs neither
func migrateSubscriptions(exchange []byte, upgrade bool) ([]byte, error) {
	value, _, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	if errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return exchange, nil
	}
	if err != nil {
		return exchange, fmt.Errorf("error getting Kraken subscriptions: %w", err)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(value, &entries); err != nil {
		return exchange, fmt.Errorf("error decoding Kraken subscriptions: %w", err)
	}
	saved := make([]savedSubscription, len(entries))
	for i := range entries {
		if err := json.Unmarshal(entries[i], &saved[i]); err != nil {
			return exchange, fmt.Errorf("error decoding Kraken subscription: %w", err)
		}
	}

	if upgrade {
		if _, ok := matchDefaults(saved, previousSubscriptions); !ok {
			return exchange, nil
		}
		for _, s := range futuresSubscriptions {
			entry, err := json.Marshal(savedSubscription{Enabled: true, Channel: s.channel, Asset: s.asset, Authenticated: s.authenticated})
			if err != nil {
				return exchange, fmt.Errorf("error encoding Kraken futures subscription: %w", err)
			}
			entries = append(entries, entry)
		}
	} else {
		matched, ok := matchDefaults(saved, slices.Concat(previousSubscriptions, futuresSubscriptions))
		if !ok {
			return exchange, nil
		}
		kept := make([]json.RawMessage, 0, len(previousSubscriptions))
		for i, d := range matched {
			if d < len(previousSubscriptions) {
				kept = append(kept, entries[i])
			}
		}
		entries = kept
	}

	updated, err := json.Marshal(entries)
	if err != nil {
		return exchange, fmt.Errorf("error encoding Kraken subscriptions: %w", err)
	}
	if exchange, err = jsonparser.Set(exchange, updated, "features", "subscriptions"); err != nil {
		return exchange, fmt.Errorf("error setting Kraken subscriptions: %w", err)
	}
	return exchange, nil
}

// matchDefaults returns the index of the default each saved subscription is, if the saved subscriptions are exactly the
// defaults in any order. No saved subscription can be two defaults, as each default has its own channel and asset
func matchDefaults(saved []savedSubscription, defaults []subscription) ([]int, bool) {
	if len(saved) != len(defaults) {
		return nil, false
	}
	matched := make([]int, len(saved))
	used := make([]bool, len(defaults))
	for i := range saved {
		matched[i] = slices.IndexFunc(defaults, func(d subscription) bool { return d.matches(&saved[i]) })
		if matched[i] == -1 || used[matched[i]] {
			return nil, false
		}
		used[matched[i]] = true
	}
	return matched, true
}

// matches returns whether a saved subscription is s as the runtime reads it, which folds the case of assets and
// treats empty pairs and parameters as none
func (s subscription) matches(saved *savedSubscription) bool {
	interval, ok := parseInterval(saved.Interval)
	return ok && saved.Enabled && saved.Channel == s.channel && strings.EqualFold(saved.Asset, s.asset) && saved.Pairs == "" &&
		len(saved.Params) == 0 && interval == s.interval && saved.Levels == s.levels && saved.Authenticated == s.authenticated
}

// intervalUnits are the interval units the runtime accepts besides Go's, in lower case
var intervalUnits = map[string]time.Duration{
	"s": time.Second, "sec": time.Second, "secs": time.Second, "second": time.Second, "seconds": time.Second,
	"m": time.Minute, "min": time.Minute, "mins": time.Minute, "minute": time.Minute, "minutes": time.Minute,
	"h": time.Hour, "hr": time.Hour, "hrs": time.Hour, "hour": time.Hour, "hours": time.Hour,
	"d": 24 * time.Hour, "day": 24 * time.Hour, "days": 24 * time.Hour,
	"w": 7 * 24 * time.Hour, "wk": 7 * 24 * time.Hour, "wks": 7 * 24 * time.Hour, "week": 7 * 24 * time.Hour, "weeks": 7 * 24 * time.Hour,
	"mo": 30 * 24 * time.Hour, "month": 30 * 24 * time.Hour, "months": 30 * 24 * time.Hour,
}

// parseInterval parses a saved interval as the runtime does: an integer counts nanoseconds, and a string is a Go
// duration or a count with a unit spelt in any case, except "M", which is a month. It returns false for an interval
// without a duration, such as "raw", which no default uses
func parseInterval(raw json.RawMessage) (time.Duration, bool) {
	if len(raw) == 0 {
		return 0, true
	}
	if n, err := strconv.ParseInt(string(raw), 10, 64); err == nil {
		return time.Duration(n), true
	}
	s := string(bytes.Trim(raw, `"`))
	if d, err := time.ParseDuration(s); err == nil {
		return d, true
	}
	split := strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' })
	if split <= 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(s[:split], 10, 64)
	if err != nil {
		return 0, false
	}
	unit := s[split:]
	if unit == "M" {
		unit = "month"
	}
	scale, ok := intervalUnits[strings.ToLower(unit)]
	if !ok || n > math.MaxInt64/int64(scale) {
		return 0, false
	}
	return time.Duration(n) * scale, true
}
