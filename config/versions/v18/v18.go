// Package v18 migrates Coinbase configurations to its rebuilt websocket, whose default subscriptions stream every
// asset, the US derivatives and the Global Derivatives gateway's perpetuals among them, and subscribe heartbeats
// implicitly.
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

// allAssets and spotAsset are the assets the defaults subscribe
const (
	allAssets = "all"
	spotAsset = "spot"
)

// subscriptionKeys are the saved subscription fields the defaults use; an entry with any other field is customised
var subscriptionKeys = []string{"enabled", "channel", "asset", "params", "interval", "levels", "authenticated"}

// defaultSubscription is a default subscription, enabled unless disabled is set
type defaultSubscription struct {
	channel       string
	asset         string
	interval      time.Duration
	authenticated bool
	disabled      bool
}

// previousDefaults are the v17 default subscriptions
var previousDefaults = []defaultSubscription{
	{channel: "heartbeat"},
	{channel: "status", asset: allAssets},
	{channel: "ticker", asset: spotAsset},
	{channel: "candles", asset: spotAsset},
	{channel: "allTrades", asset: spotAsset},
	{channel: "orderbook", asset: spotAsset},
	{channel: "myAccount", asset: allAssets, authenticated: true},
	{channel: "ticker_batch", asset: spotAsset},
}

// previousExample is the subscription list the v17 config_example.json shipped, which differs from previousDefaults.
// Configurations copied from it hold GCT's choices rather than the user's, and its account channel is one neither
// version can generate, which fails every subscription once authenticated websocket support is enabled
var previousExample = []defaultSubscription{
	{channel: "heartbeat"},
	{channel: "status", authenticated: true, disabled: true},
	{channel: "ticker", asset: spotAsset},
	{channel: "candles", asset: spotAsset},
	{channel: "allTrades", asset: spotAsset},
	{channel: "orderbook", asset: spotAsset},
	{channel: "account", authenticated: true},
	{channel: "ticker_batch", asset: spotAsset, disabled: true},
}

// currentDefaults are the v18 default subscriptions, which stream every asset; each connection subscribes heartbeats
// itself, and the ticker channel covers what ticker_batch did
var currentDefaults = []defaultSubscription{
	{channel: "ticker", asset: allAssets},
	{channel: "candles", asset: allAssets, interval: 5 * time.Minute},
	{channel: "allTrades", asset: allAssets},
	{channel: "orderbook", asset: allAssets},
	{channel: "status", asset: allAssets},
	{channel: "myAccount", asset: allAssets, authenticated: true},
}

// legacyChannels are the Coinbase Pro channels the previous version replaced every subscription for with its defaults
// whenever it loaded a configuration holding one
var legacyChannels = []string{"level2_batch", "matches"}

// previousChannels are the channels the previous version subscribes; it errors generating any other
var previousChannels = []string{"heartbeat", "ticker", "candles", "allTrades", "orderbook", "myAccount", "status", "ticker_batch"}

// previousAssets are the assets the previous version trades; it expands a subscription for any other asset to nothing
var previousAssets = []string{"", allAssets, spotAsset, "futures"}

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
	Enabled       bool   `json:"enabled"`
	Channel       string `json:"channel"`
	Asset         string `json:"asset,omitempty"`
	Interval      string `json:"interval,omitempty"`
	Authenticated bool   `json:"authenticated,omitempty"`
}

// Version implements ExchangeVersion for Coinbase's rebuilt websocket
type Version struct{}

// Exchanges returns just Coinbase
func (*Version) Exchanges() []string { return []string{"Coinbase"} }

// UpgradeExchange replaces the previous default subscriptions, or the list the previous config_example.json shipped,
// with the current defaults, which stream every asset, as it does a subscription list holding a legacy Coinbase Pro
// channel, which the previous version replaced with its defaults on load; subscriptions which differ from both are the
// user's choice and are left unchanged
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	entries, err := subscriptions(exchange)
	if err != nil || entries == nil {
		return exchange, err
	}
	saved, err := decodeEntries(entries)
	if err != nil {
		return exchange, fmt.Errorf("error decoding Coinbase subscriptions: %w", err)
	}
	if slices.ContainsFunc(saved, func(s savedSubscription) bool { return slices.Contains(legacyChannels, s.Channel) }) {
		return setDefaults(exchange, currentDefaults, nil)
	}
	for _, previous := range [][]defaultSubscription{previousDefaults, previousExample} {
		if inert, ok := matchDefaults(entries, saved, previous); ok {
			return setDefaults(exchange, currentDefaults, inert)
		}
	}
	return exchange, nil
}

// DowngradeExchange replaces the current default subscriptions with the previous defaults, and disables the enabled
// subscriptions the previous version cannot generate: a channel it does not subscribe, for an asset it trades. It
// removes any subscription to a legacy Coinbase Pro channel, enabled or not, for which the previous version would
// replace every subscription with its defaults
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	entries, err := subscriptions(exchange)
	if err != nil || entries == nil {
		return exchange, err
	}
	saved, err := decodeEntries(entries)
	if err != nil {
		return exchange, fmt.Errorf("error decoding Coinbase subscriptions: %w", err)
	}
	if inert, ok := matchDefaults(entries, saved, currentDefaults); ok {
		return setDefaults(exchange, previousDefaults, inert)
	}
	for i := range saved {
		if saved[i].Enabled == nil || !*saved[i].Enabled || slices.Contains(previousChannels, saved[i].Channel) ||
			!slices.ContainsFunc(previousAssets, func(a string) bool { return strings.EqualFold(a, saved[i].Asset) }) {
			continue
		}
		if exchange, err = jsonparser.Set(exchange, []byte("false"), "features", "subscriptions", "["+strconv.Itoa(i)+"]", "enabled"); err != nil {
			return exchange, fmt.Errorf("error disabling Coinbase subscription %d: %w", i, err)
		}
	}
	// Removing from the end keeps the earlier entries' indices
	for i, s := range slices.Backward(saved) {
		if slices.Contains(legacyChannels, s.Channel) {
			exchange = jsonparser.Delete(exchange, "features", "subscriptions", "["+strconv.Itoa(i)+"]")
		}
	}
	return exchange, nil
}

// subscriptions returns a configuration's saved subscriptions, or none when it holds none
func subscriptions(exchange []byte) ([]json.RawMessage, error) {
	raw, valueType, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError), err == nil && valueType == jsonparser.Null:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("error getting Coinbase subscriptions: %w", err)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("error decoding Coinbase subscriptions: %w", err)
	}
	if entries == nil {
		entries = []json.RawMessage{}
	}
	return entries, nil
}

// decodeEntries decodes each saved subscription, so that a malformed entry is reported whichever defaults are matched
func decodeEntries(entries []json.RawMessage) ([]savedSubscription, error) {
	saved := make([]savedSubscription, len(entries))
	for i, entry := range entries {
		if err := json.Unmarshal(entry, &saved[i]); err != nil {
			return nil, err
		}
	}
	return saved, nil
}

// setDefaults replaces a configuration's subscriptions with a set of defaults, as the config saves them, followed by
// the disabled entries kept
func setDefaults(exchange []byte, defaults []defaultSubscription, keep []json.RawMessage) ([]byte, error) {
	saved := make([]any, 0, len(defaults)+len(keep))
	for i := range defaults {
		d := savedDefault{Enabled: true, Channel: defaults[i].channel, Asset: defaults[i].asset, Authenticated: defaults[i].authenticated}
		if defaults[i].interval != 0 {
			d.Interval = shortInterval(defaults[i].interval)
		}
		saved = append(saved, d)
	}
	for _, entry := range keep {
		saved = append(saved, entry)
	}
	updated, err := json.Marshal(saved)
	if err != nil {
		return exchange, fmt.Errorf("error encoding Coinbase subscriptions: %w", err)
	}
	if exchange, err = jsonparser.Set(exchange, updated, "features", "subscriptions"); err != nil {
		return exchange, fmt.Errorf("error setting Coinbase subscriptions: %w", err)
	}
	return exchange, nil
}

// shortInterval returns an interval as the config saves it, such as 5m
func shortInterval(d time.Duration) string {
	s := d.String()
	s = strings.TrimSuffix(s, "0s")
	return strings.TrimSuffix(s, "0m")
}

// matchDefaults reports whether each default is resolved to by a different saved subscription holding only the fields
// the defaults use, setting aside the disabled entries no default takes: the config loader drops a disabled
// subscription, so it changes nothing the defaults stream, and it is returned to be kept. An entry is matched to the
// defaults before the rest are set aside, so a disabled entry naming a channel another entry resolves is set aside
// wherever it comes in the list
func matchDefaults(entries []json.RawMessage, saved []savedSubscription, defaults []defaultSubscription) (inert []json.RawMessage, ok bool) {
	matched := make([]bool, len(defaults))
	resolved := make([]bool, len(saved))
	for i := range saved {
		j := slices.IndexFunc(defaults, func(d defaultSubscription) bool {
			return d.channel == saved[i].Channel && strings.EqualFold(d.asset, saved[i].Asset)
		})
		if j != -1 && !matched[j] && defaults[j].matches(&saved[i]) && onlyDefaultFields(entries[i]) {
			matched[j], resolved[i] = true, true
		}
	}
	if slices.Contains(matched, false) {
		return nil, false
	}
	for i := range saved {
		if resolved[i] {
			continue
		}
		if saved[i].Enabled != nil && *saved[i].Enabled {
			return nil, false
		}
		inert = append(inert, entries[i])
	}
	return inert, true
}

// onlyDefaultFields reports whether a saved subscription holds only the fields the defaults use
func onlyDefaultFields(entry json.RawMessage) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(entry, &fields) != nil {
		return false
	}
	for key := range fields {
		if !slices.Contains(subscriptionKeys, key) {
			return false
		}
	}
	return true
}

// matches reports whether a saved subscription resolves to the default: explicitly enabled, or disabled where the
// default is, authenticated as the default is, an absent, false or null authenticated field all being unauthenticated,
// without parameters, an absent, null or empty parameters object all being none, with the default's interval as the
// config loader parses it and no levels
func (d *defaultSubscription) matches(s *savedSubscription) bool {
	interval, ok := parseInterval(s.Interval)
	if !ok {
		return false
	}
	levels, ok := parseLevels(s.Levels)
	return ok &&
		s.Enabled != nil && *s.Enabled != d.disabled &&
		(s.Authenticated != nil && *s.Authenticated) == d.authenticated &&
		interval == d.interval &&
		levels == 0 &&
		(len(s.Params) == 0 || reflect.DeepEqual(s.Params, map[string]any{}))
}

// parseLevels returns saved levels as the config loader decodes them, an absent or null field being zero
func parseLevels(raw json.RawMessage) (int, bool) {
	var levels int
	if raw != nil && json.Unmarshal(raw, &levels) != nil {
		return 0, false
	}
	return levels, true
}

// parseInterval returns a saved interval as the config loader parses it: nanoseconds, a duration such as 5m0s, or a
// count of units such as 5min; an absent interval is zero, and null is rejected as the loader rejects it
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
