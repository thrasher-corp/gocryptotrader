// Package v18 migrates MEXC configurations saved with the spot-only default websocket subscriptions to the defaults
// which also cover its perpetual futures contracts.
package v18

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // Used instead of gct encoding/json so that we can ensure consistent library functionality between versions
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/buger/jsonparser"
)

// The assets the default subscriptions name, as the config saves them
const (
	spotAsset    = "spot"
	futuresAsset = "futures"
)

// subscriptionKeys are the saved subscription fields the defaults use; an entry with any other field is customised
var subscriptionKeys = []string{"enabled", "channel", "asset", "interval", "levels", "authenticated"}

// defaultSubscription is an enabled default subscription
type defaultSubscription struct {
	channel       string
	asset         string
	interval      time.Duration
	levels        int64
	authenticated bool
}

// previousDefaults are the v17 default subscriptions, which only cover spot
var previousDefaults = []defaultSubscription{
	{channel: "orderbook", asset: spotAsset, levels: 20},
	{channel: "candles", asset: spotAsset, interval: 15 * time.Minute},
	{channel: "ticker", asset: spotAsset, interval: 100 * time.Millisecond},
	{channel: "public.miniTicker.v3.api.pb", asset: spotAsset},
	{channel: "allTrades", asset: spotAsset, interval: 100 * time.Millisecond},
	{channel: "myTrades", asset: spotAsset, authenticated: true},
	{channel: "myOrders", asset: spotAsset, authenticated: true},
	{channel: "myAccount", asset: spotAsset, authenticated: true},
}

// currentDefaults are the v18 default subscriptions, the previous defaults followed by those of the futures contracts
var currentDefaults = slices.Concat(previousDefaults, []defaultSubscription{
	{channel: "ticker", asset: futuresAsset},
	{channel: "orderbook", asset: futuresAsset, levels: 20},
	{channel: "allTrades", asset: futuresAsset},
	{channel: "candles", asset: futuresAsset, interval: 15 * time.Minute},
	{channel: "myOrders", asset: futuresAsset, authenticated: true},
	{channel: "myTrades", asset: futuresAsset, authenticated: true},
	{channel: "myAccount", asset: futuresAsset, authenticated: true},
	{channel: "position", asset: futuresAsset, authenticated: true},
})

// savedSubscription holds the saved subscription fields the config loader decodes
type savedSubscription struct {
	Enabled       *bool           `json:"enabled"`
	Channel       string          `json:"channel"`
	Asset         string          `json:"asset"`
	Interval      json.RawMessage `json:"interval"`
	Levels        *int64          `json:"levels"`
	Authenticated *bool           `json:"authenticated"`
}

// savedDefault is a default subscription in the form the config saves it
type savedDefault struct {
	Enabled       bool   `json:"enabled"`
	Channel       string `json:"channel"`
	Asset         string `json:"asset"`
	Interval      string `json:"interval,omitempty"`
	Levels        int64  `json:"levels,omitempty"`
	Authenticated bool   `json:"authenticated,omitempty"`
}

// Version implements ExchangeVersion for MEXC's futures support
type Version struct{}

// Exchanges returns just MEXC
func (*Version) Exchanges() []string { return []string{"MEXC"} }

// UpgradeExchange replaces the previous default subscriptions with the current defaults; subscriptions which differ
// from the previous defaults are the user's choice and are left unchanged. The futures subscriptions take effect once
// the futures asset is enabled
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return replaceSubscriptions(exchange, previousDefaults, currentDefaults)
}

// DowngradeExchange replaces the current default subscriptions with the previous defaults. The previous version
// supports no futures, so it expands no subscription for them and other futures entries are left unchanged
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return replaceSubscriptions(exchange, currentDefaults, previousDefaults)
}

// replaceSubscriptions replaces saved subscriptions which match one set of defaults, in any order, with another
func replaceSubscriptions(exchange []byte, from, to []defaultSubscription) ([]byte, error) {
	raw, valueType, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError), err == nil && valueType == jsonparser.Null:
		return exchange, nil
	case err != nil:
		return exchange, fmt.Errorf("error getting MEXC subscriptions: %w", err)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return exchange, fmt.Errorf("error decoding MEXC subscriptions: %w", err)
	}
	matched, err := matchDefaults(entries, from)
	if err != nil {
		return exchange, fmt.Errorf("error decoding MEXC subscriptions: %w", err)
	}
	if !matched {
		return exchange, nil
	}
	saved := make([]savedDefault, len(to))
	for i := range to {
		saved[i] = savedDefault{Enabled: true, Channel: to[i].channel, Asset: to[i].asset, Levels: to[i].levels, Authenticated: to[i].authenticated}
		if to[i].interval != 0 {
			saved[i].Interval = shortInterval(to[i].interval)
		}
	}
	updated, err := json.Marshal(saved)
	if err != nil {
		return exchange, fmt.Errorf("error encoding MEXC subscriptions: %w", err)
	}
	if exchange, err = jsonparser.Set(exchange, updated, "features", "subscriptions"); err != nil {
		return exchange, fmt.Errorf("error setting MEXC subscriptions: %w", err)
	}
	return exchange, nil
}

// shortInterval returns an interval as the config saves it, without the zero units time.Duration adds, such as "15m"
func shortInterval(d time.Duration) string {
	s := d.String()
	for _, zeros := range []string{"m0s", "h0m"} {
		if strings.HasSuffix(s, zeros) {
			s = s[:len(s)-2]
		}
	}
	return s
}

// matchDefaults reports whether each saved subscription resolves to a different one of the defaults
// Every entry is decoded first, so that a malformed entry is reported whichever defaults are matched
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
		// A channel and asset name one default at most, so an entry can match no other
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

// matches reports whether a saved subscription resolves to the default; an absent, false or null authenticated field
// is unauthenticated, and absent levels are none
func (d *defaultSubscription) matches(s *savedSubscription) bool {
	interval, ok := parseInterval(s.Interval)
	var levels int64
	if s.Levels != nil {
		levels = *s.Levels
	}
	return ok &&
		s.Enabled != nil && *s.Enabled &&
		interval == d.interval &&
		levels == d.levels &&
		(s.Authenticated != nil && *s.Authenticated) == d.authenticated
}

// parseInterval returns a saved interval as the config loader parses it, either as nanoseconds or as a duration such
// as "15m"; an absent interval is zero, and null is rejected as the loader rejects it
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
	d, err := time.ParseDuration(text)
	return d, err == nil
}
