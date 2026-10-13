// Package v18 migrates Bitstamp configurations to its v2 API, whose authentication needs no client ID and whose default
// websocket subscriptions cover perpetual contracts.
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

const clientIDRequirement = "requiresClientID"

// subscriptionKeys are the saved subscription fields the defaults use; an entry with any other field is customised
var subscriptionKeys = []string{"enabled", "channel", "asset", "interval", "authenticated"}

// defaultSubscription is an enabled default subscription
type defaultSubscription struct {
	channel       string
	asset         string
	interval      time.Duration
	authenticated bool
}

// previousDefaults are the v17 default subscriptions, which only cover spot
var previousDefaults = []defaultSubscription{
	{channel: "orderbook", asset: "spot", interval: 100 * time.Millisecond},
	{channel: "allTrades", asset: "spot"},
	{channel: "myOrders", asset: "spot", authenticated: true},
	{channel: "myTrades", asset: "spot", authenticated: true},
}

// currentDefaults are the v18 default subscriptions, which cover every enabled asset and perpetual funding rates
var currentDefaults = []defaultSubscription{
	{channel: "orderbook", asset: "all", interval: 100 * time.Millisecond},
	{channel: "allTrades", asset: "all"},
	{channel: "funding_rate", asset: "perpetualcontract"},
	{channel: "myOrders", asset: "all", authenticated: true},
	{channel: "myTrades", asset: "all", authenticated: true},
}

// savedSubscription holds the saved subscription fields the config loader decodes
type savedSubscription struct {
	Enabled       *bool           `json:"enabled"`
	Channel       string          `json:"channel"`
	Asset         string          `json:"asset"`
	Interval      json.RawMessage `json:"interval"`
	Authenticated *bool           `json:"authenticated"`
}

// savedDefault is a default subscription in the form the config saves it
type savedDefault struct {
	Enabled       bool   `json:"enabled"`
	Channel       string `json:"channel"`
	Asset         string `json:"asset"`
	Interval      string `json:"interval,omitempty"`
	Authenticated bool   `json:"authenticated,omitempty"`
}

// Version implements ExchangeVersion for Bitstamp's v2 API update
type Version struct{}

// Exchanges returns just Bitstamp
func (*Version) Exchanges() []string { return []string{"Bitstamp"} }

// UpgradeExchange removes the client ID requirement and replaces the previous default subscriptions with the current
// defaults; subscriptions which differ from the previous defaults are the user's choice and are left unchanged
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	// The exchange overwrites the saved credentials validator during setup, but the config loader checks the saved
	// validator first, so a client ID requirement would disable authentication for credentials without a client ID
	keys, err := clientIDRequirementKeys(exchange)
	if err != nil {
		return exchange, err
	}
	for _, key := range keys {
		exchange = jsonparser.Delete(exchange, "api", "credentialsValidator", key)
	}
	return replaceSubscriptions(exchange, previousDefaults, currentDefaults)
}

// DowngradeExchange restores the client ID requirement and replaces the current default subscriptions with the
// previous defaults
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	_, valueType, _, err := jsonparser.Get(exchange, "api", "credentialsValidator")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
	case err != nil:
		return exchange, fmt.Errorf("error getting Bitstamp credentials validator: %w", err)
	case valueType == jsonparser.Object:
		if exchange, err = jsonparser.Set(exchange, []byte("true"), "api", "credentialsValidator", clientIDRequirement); err != nil {
			return exchange, fmt.Errorf("error setting Bitstamp client ID requirement: %w", err)
		}
	}
	return replaceSubscriptions(exchange, currentDefaults, previousDefaults)
}

// clientIDRequirementKeys returns each credentials validator key the config loader decodes as the client ID
// requirement, since it matches keys case insensitively; a key repeated in the validator is returned for each repeat
func clientIDRequirementKeys(exchange []byte) ([]string, error) {
	validator, valueType, _, err := jsonparser.Get(exchange, "api", "credentialsValidator")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("error getting Bitstamp credentials validator: %w", err)
	case valueType != jsonparser.Object:
		return nil, nil
	}
	var keys []string
	if err := jsonparser.ObjectEach(validator, func(key, _ []byte, _ jsonparser.ValueType, _ int) error {
		if strings.EqualFold(string(key), clientIDRequirement) {
			keys = append(keys, string(key))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("error decoding Bitstamp credentials validator: %w", err)
	}
	return keys, nil
}

// replaceSubscriptions replaces saved subscriptions which match one set of defaults, in any order, with another
func replaceSubscriptions(exchange []byte, from, to []defaultSubscription) ([]byte, error) {
	raw, valueType, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError), err == nil && valueType == jsonparser.Null:
		return exchange, nil
	case err != nil:
		return exchange, fmt.Errorf("error getting Bitstamp subscriptions: %w", err)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return exchange, fmt.Errorf("error decoding Bitstamp subscriptions: %w", err)
	}
	matched, err := matchDefaults(entries, from)
	if err != nil {
		return exchange, fmt.Errorf("error decoding Bitstamp subscriptions: %w", err)
	}
	if !matched {
		return exchange, nil
	}
	saved := make([]savedDefault, len(to))
	for i := range to {
		saved[i] = savedDefault{Enabled: true, Channel: to[i].channel, Asset: to[i].asset, Authenticated: to[i].authenticated}
		if to[i].interval != 0 {
			saved[i].Interval = to[i].interval.String()
		}
	}
	updated, err := json.Marshal(saved)
	if err != nil {
		return exchange, fmt.Errorf("error encoding Bitstamp subscriptions: %w", err)
	}
	if exchange, err = jsonparser.Set(exchange, updated, "features", "subscriptions"); err != nil {
		return exchange, fmt.Errorf("error setting Bitstamp subscriptions: %w", err)
	}
	return exchange, nil
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
		j := slices.IndexFunc(defaults, func(d defaultSubscription) bool { return d.channel == saved[i].Channel })
		if j == -1 || matched[j] || !defaults[j].matches(&saved[i]) {
			return false, nil
		}
		matched[j] = true
	}
	return true, nil
}

// matches reports whether a saved subscription resolves to the default; assets match case insensitively as the loader
// parses them, and an absent, false or null authenticated field are all unauthenticated
func (d *defaultSubscription) matches(s *savedSubscription) bool {
	interval, ok := parseInterval(s.Interval)
	return ok &&
		s.Enabled != nil && *s.Enabled &&
		strings.EqualFold(s.Asset, d.asset) &&
		interval == d.interval &&
		(s.Authenticated != nil && *s.Authenticated) == d.authenticated
}

// parseInterval returns a saved interval as the config loader parses it, either as nanoseconds or as a duration such
// as "100ms"; an absent interval is zero, and null is rejected as the loader rejects it
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
