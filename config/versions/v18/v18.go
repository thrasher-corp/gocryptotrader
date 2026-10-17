// Package v18 migrates Bitfinex configurations to its perpetual contracts, whose default websocket subscriptions add
// their candles and status channels.
package v18

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // Used instead of gct encoding/json so that we can ensure consistent library functionality between versions
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
)

// allAssets and candlesChannel are the asset and channel several defaults share
const (
	allAssets      = "all"
	candlesChannel = "candles"
)

// subscriptionKeys are the saved subscription fields the defaults use; an entry with any other field is customised
var subscriptionKeys = []string{"enabled", "channel", "asset", "params", "interval", "levels", "authenticated"}

// defaultSubscription is an enabled, unauthenticated default subscription
type defaultSubscription struct {
	channel  string
	asset    string
	params   map[string]any
	interval time.Duration
	levels   int
}

// previousDefaults are the v17 default subscriptions
var previousDefaults = []defaultSubscription{
	{channel: "ticker", asset: allAssets},
	{channel: "allTrades", asset: allAssets},
	{channel: candlesChannel, asset: "spot", interval: time.Minute},
	{channel: candlesChannel, asset: "margin", interval: time.Minute},
	{channel: candlesChannel, asset: "marginfunding", interval: time.Minute, params: map[string]any{"_period": "p30"}},
	{channel: "orderbook", asset: allAssets, levels: 100, params: map[string]any{"prec": "R0"}},
}

// currentDefaults are the v18 default subscriptions, which add the perpetual contracts' candles and status channels
var currentDefaults = []defaultSubscription{
	{channel: "ticker", asset: allAssets},
	{channel: "allTrades", asset: allAssets},
	{channel: candlesChannel, asset: "spot", interval: time.Minute},
	{channel: candlesChannel, asset: "margin", interval: time.Minute},
	{channel: candlesChannel, asset: "marginfunding", interval: time.Minute, params: map[string]any{"_period": "p30"}},
	{channel: candlesChannel, asset: "usdtmarginedfutures", interval: time.Minute},
	{channel: candlesChannel, asset: "coinmarginedfutures", interval: time.Minute},
	{channel: "orderbook", asset: allAssets, levels: 100, params: map[string]any{"prec": "R0"}},
	{channel: "status", asset: "usdtmarginedfutures"},
	{channel: "status", asset: "coinmarginedfutures"},
}

// perpetualAssets are the assets of the perpetual contracts, which the previous version does not trade
var perpetualAssets = []string{"usdtmarginedfutures", "coinmarginedfutures"}

// savedSubscription holds the saved subscription fields the config loader decodes
type savedSubscription struct {
	Enabled       *bool           `json:"enabled"`
	Channel       string          `json:"channel"`
	Asset         string          `json:"asset"`
	Params        map[string]any  `json:"params"`
	Interval      json.RawMessage `json:"interval"`
	Levels        json.RawMessage `json:"levels"`
	Authenticated *bool           `json:"authenticated"`
}

// savedDefault is a default subscription in the form the config saves it
type savedDefault struct {
	Enabled  bool           `json:"enabled"`
	Channel  string         `json:"channel"`
	Asset    string         `json:"asset"`
	Params   map[string]any `json:"params,omitempty"`
	Interval string         `json:"interval,omitempty"`
	Levels   int            `json:"levels,omitempty"`
}

// Version implements ExchangeVersion for Bitfinex's perpetual contracts
type Version struct{}

// Exchanges returns just Bitfinex
func (*Version) Exchanges() []string { return []string{"Bitfinex"} }

// UpgradeExchange replaces the previous default subscriptions with the current defaults, which cover the perpetual
// contracts; subscriptions which differ from the previous defaults are the user's choice and are left unchanged
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return replaceSubscriptions(exchange, previousDefaults, currentDefaults)
}

// DowngradeExchange replaces the current default subscriptions with the previous defaults, and disables subscriptions
// to the status channel the previous version cannot subscribe to
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	exchange, err := replaceSubscriptions(exchange, currentDefaults, previousDefaults)
	if err != nil {
		return exchange, err
	}
	return disableStatusSubscriptions(exchange)
}

// disableStatusSubscriptions disables the enabled subscriptions to the status channel, which the previous version sends
// as a channel needing a symbol, failing to subscribe. It never expands one for a perpetual contract asset, which it
// does not trade, so those are left as they are
func disableStatusSubscriptions(exchange []byte) ([]byte, error) {
	raw, valueType, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	if err != nil || valueType != jsonparser.Array {
		return exchange, nil //nolint:nilerr // replaceSubscriptions has already reported malformed subscriptions
	}
	var i int
	var errs error
	// Set returns a new document, so the entries are read from a copy of the original
	if _, err := jsonparser.ArrayEach(bytes.Clone(raw), func(entry []byte, _ jsonparser.ValueType, _ int, _ error) {
		defer func() { i++ }()
		channel, _ := jsonparser.GetString(entry, "channel")
		a, _ := jsonparser.GetString(entry, "asset")
		if enabled, err := jsonparser.GetBoolean(entry, "enabled"); err != nil || !enabled || channel != "status" || slices.ContainsFunc(perpetualAssets, func(p string) bool { return strings.EqualFold(p, a) }) {
			return
		}
		var setErr error
		if exchange, setErr = jsonparser.Set(exchange, []byte("false"), "features", "subscriptions", "["+strconv.Itoa(i)+"]", "enabled"); setErr != nil {
			errs = setErr
		}
	}); err != nil {
		return exchange, err
	}
	return exchange, errs
}

// replaceSubscriptions replaces saved subscriptions which match one set of defaults, in any order, with another
func replaceSubscriptions(exchange []byte, from, to []defaultSubscription) ([]byte, error) {
	raw, valueType, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError), err == nil && valueType == jsonparser.Null:
		return exchange, nil
	case err != nil:
		return exchange, fmt.Errorf("error getting Bitfinex subscriptions: %w", err)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return exchange, fmt.Errorf("error decoding Bitfinex subscriptions: %w", err)
	}
	matched, err := matchDefaults(entries, from)
	if err != nil {
		return exchange, fmt.Errorf("error decoding Bitfinex subscriptions: %w", err)
	}
	if !matched {
		return exchange, nil
	}
	saved := make([]savedDefault, len(to))
	for i := range to {
		saved[i] = savedDefault{Enabled: true, Channel: to[i].channel, Asset: to[i].asset, Params: to[i].params, Levels: to[i].levels}
		if to[i].interval != 0 {
			saved[i].Interval = shortInterval(to[i].interval)
		}
	}
	updated, err := json.Marshal(saved)
	if err != nil {
		return exchange, fmt.Errorf("error encoding Bitfinex subscriptions: %w", err)
	}
	if exchange, err = jsonparser.Set(exchange, updated, "features", "subscriptions"); err != nil {
		return exchange, fmt.Errorf("error setting Bitfinex subscriptions: %w", err)
	}
	return exchange, nil
}

// shortInterval returns an interval as the config saves it, such as 1m
func shortInterval(d time.Duration) string {
	s := d.String()
	s = strings.TrimSuffix(s, "0s")
	return strings.TrimSuffix(s, "0m")
}

// matchDefaults reports whether each saved subscription resolves to a different one of the defaults. Every entry is
// decoded first, so that a malformed entry is reported whichever defaults are matched
func matchDefaults(entries []json.RawMessage, defaults []defaultSubscription) (bool, error) {
	saved := make([]savedSubscription, len(entries))
	var customised bool
	for i, entry := range entries {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(entry, &fields); err != nil {
			return false, err
		}
		for key := range fields {
			if !slices.Contains(subscriptionKeys, key) {
				customised = true
			}
		}
		if err := json.Unmarshal(entry, &saved[i]); err != nil {
			return false, err
		}
	}
	if customised || len(saved) != len(defaults) {
		return false, nil
	}
	matched := make([]bool, len(defaults))
	for i := range saved {
		j := slices.IndexFunc(defaults, func(d defaultSubscription) bool {
			return d.channel == saved[i].Channel && strings.EqualFold(d.asset, saved[i].Asset)
		})
		if j == -1 || matched[j] || !defaults[j].matches(&saved[i]) {
			return false, nil
		}
		matched[j] = true
	}
	return true, nil
}

// matches reports whether a saved subscription resolves to the default: enabled and unauthenticated, an absent, false
// or null authenticated field all being unauthenticated, with the default's parameters, an absent, null or empty
// parameters object all being none, and its interval and levels as the config loader parses them
func (d *defaultSubscription) matches(s *savedSubscription) bool {
	interval, ok := parseInterval(s.Interval)
	if !ok {
		return false
	}
	levels, ok := parseLevels(s.Levels)
	return ok &&
		s.Enabled != nil && *s.Enabled &&
		(s.Authenticated == nil || !*s.Authenticated) &&
		interval == d.interval &&
		levels == d.levels &&
		(len(s.Params) == 0 && len(d.params) == 0 || reflect.DeepEqual(s.Params, d.params))
}

// parseLevels returns saved levels as the config loader decodes them, an absent or null field being zero
func parseLevels(raw json.RawMessage) (int, bool) {
	var levels int
	if raw != nil && json.Unmarshal(raw, &levels) != nil {
		return 0, false
	}
	return levels, true
}

// parseInterval returns a saved interval as the config loader parses it: nanoseconds, a duration such as 1m0s, or a
// count of units such as 1min; an absent interval is zero, and null is rejected as the loader rejects it
func parseInterval(raw json.RawMessage) (time.Duration, bool) {
	if raw == nil {
		return 0, true
	}
	if bytes.Equal(raw, []byte("null")) {
		return 0, false
	}
	var nanoseconds int64
	if err := json.Unmarshal(raw, &nanoseconds); err == nil {
		return time.Duration(nanoseconds), true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, false
	}
	if d, err := time.ParseDuration(text); err == nil {
		return d, true
	}
	split := strings.IndexFunc(text, func(r rune) bool { return r < '0' || r > '9' })
	if split <= 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(text[:split], 10, 64)
	if err != nil {
		return 0, false
	}
	unit := text[split:]
	if unit == "M" {
		// A month, before folding case would make it a minute
		unit = "month"
	}
	var scale time.Duration
	switch strings.ToLower(unit) {
	case "s", "sec", "secs", "second", "seconds":
		scale = time.Second
	case "m", "min", "mins", "minute", "minutes":
		scale = time.Minute
	case "h", "hr", "hrs", "hour", "hours":
		scale = time.Hour
	case "d", "day", "days":
		scale = 24 * time.Hour
	case "w", "wk", "wks", "week", "weeks":
		scale = 7 * 24 * time.Hour
	case "mo", "month", "months":
		scale = 30 * 24 * time.Hour
	default:
		return 0, false
	}
	if n > math.MaxInt64/int64(scale) {
		return 0, false
	}
	return time.Duration(n) * scale, true
}
