// Package v25 replaces Binance's previous default subscriptions, which stream spot alone, with the current defaults,
// which also stream USDT-margined futures, coin-margined futures and options. A customised list keeps every
// subscription as saved, all-assets ones included, as the exchange now streams those for every asset. A USDT-margined or
// coin-margined futures asset it names, directly or through an all-assets subscription other than myOrders, gains the
// default order book subscription unless an order book subscription covers it, and a futures asset it does not name
// streams the exchange's runtime defaults.
package v25

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // Used instead of gct encoding/json so that we can ensure consistent library functionality between versions
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
)

var errNotDefault = errors.New("not a default subscription")

// Version implements ExchangeVersion for Binance's default subscriptions
type Version struct{}

// defaultSubscription is an enabled default subscription with its fields as GCT saves them
type defaultSubscription struct {
	channel, asset, interval string
}

func assetDefaults(a string) []defaultSubscription {
	return []defaultSubscription{
		{channel: "ticker", asset: a},
		{channel: "allTrades", asset: a},
		{channel: "candles", asset: a, interval: "1m"},
		{channel: "orderbook", asset: a, interval: "100ms"},
	}
}

var (
	previousDefaults = assetDefaults("spot")
	currentDefaults  = slices.Concat(assetDefaults("spot"), assetDefaults("usdtmarginedfutures"), assetDefaults("coinmarginedfutures"), assetDefaults("options"))
	// futuresOrderbooks are the default order book subscriptions of the USDT-margined and coin-margined futures assets
	futuresOrderbooks = []defaultSubscription{
		{channel: "orderbook", asset: "usdtmarginedfutures", interval: "100ms"},
		{channel: "orderbook", asset: "coinmarginedfutures", interval: "100ms"},
	}
)

// Exchanges returns just Binance
func (*Version) Exchanges() []string { return []string{"Binance"} }

// UpgradeExchange replaces the previous default subscriptions with the current defaults, and has a customised list
// stream the USDT-margined and coin-margined futures order books. A downgrade leaves such a list as upgraded, as the
// previous version streams nothing for the entries it gained
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	subscriptions, path, ok := savedSubscriptions(exchange)
	if !ok {
		return exchange, nil
	}
	upgraded, ok := streamFuturesOrderbooks(subscriptions)
	if !ok {
		return exchange, nil
	}
	if isDefaults(upgraded, previousDefaults) {
		upgraded = marshal(currentDefaults)
	}
	if bytes.Equal(upgraded, subscriptions) {
		return exchange, nil
	}
	return jsonparser.Set(exchange, upgraded, path...)
}

// DowngradeExchange replaces the current default subscriptions with the previous defaults
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return replaceDefaults(exchange, currentDefaults, previousDefaults)
}

// replaceDefaults replaces saved subscriptions which are one set of defaults, in any order, with another set. Any other
// list is the user's choice and stays as saved, as does a list the loader would reject, or one in a configuration whose
// name, features or subscriptions the loader could read from more than one field, as it ignores case in field names, or
// from a field only one of its decoders reads
func replaceDefaults(exchange []byte, from, to []defaultSubscription) ([]byte, error) {
	subscriptions, path, ok := savedSubscriptions(exchange)
	if !ok || !isDefaults(subscriptions, from) {
		return exchange, nil
	}
	return jsonparser.Set(exchange, marshal(to), path...)
}

// savedSubscriptions returns an exchange configuration's saved subscriptions and the field names of their path, and
// whether its name, features and subscriptions are each the only field the loader could read them from, read as such by
// both decoders
func savedSubscriptions(exchange []byte) (subscriptions []byte, path []string, ok bool) {
	if _, _, ok = soleField(exchange, "name", jsonparser.String); !ok {
		return nil, nil, false
	}
	features, featuresKey, ok := soleField(exchange, "features", jsonparser.Object)
	if !ok {
		return nil, nil, false
	}
	subscriptions, subscriptionsKey, ok := soleField(features, "subscriptions", jsonparser.Array)
	if !ok {
		return nil, nil, false
	}
	return subscriptions, []string{featuresKey, subscriptionsKey}, true
}

// savedEntry is a saved subscription's asset and channel as the loader reads them, and, for an all-assets order book
// subscription, whether the USDT-margined and coin-margined futures streams take its update speed and depth
type savedEntry struct {
	asset, channel string
	futuresBook    bool
}

// streamFuturesOrderbooks returns saved subscriptions with the default order book subscription of each USDT-margined or
// coin-margined futures asset they name, directly or through an all-assets subscription, myOrders aside, but give no
// order book subscription covering it, enabled or not: one of the asset, or an all-assets one at an update speed and
// depth the futures streams take. It reports false, leaving the list as saved, for a list that is not valid JSON, holds
// an entry that is not an object or whose asset or channel is neither text nor null, or holds an entry whose asset or
// channel, or on an all-assets order book subscription whose interval or levels, the default build's and sonic's
// decoders could read differently
func streamFuturesOrderbooks(subscriptions []byte) ([]byte, bool) {
	if !json.Valid(subscriptions) {
		return nil, false
	}
	var entries []savedEntry
	valid := true
	_, err := jsonparser.ArrayEach(subscriptions, func(raw []byte, dataType jsonparser.ValueType, _ int, _ error) {
		entry, ok := readEntry(raw, dataType)
		valid = valid && ok
		entries = append(entries, entry)
	})
	if err != nil || !valid {
		return nil, false
	}
	var added []byte
	for _, book := range futuresOrderbooks {
		named := slices.ContainsFunc(entries, func(e savedEntry) bool {
			return e.channel != "myOrders" && (e.asset == book.asset || e.asset == "all")
		})
		covered := slices.ContainsFunc(entries, func(e savedEntry) bool {
			return e.channel == book.channel && (e.asset == book.asset || e.futuresBook)
		})
		if named && !covered {
			added = append(append(added, ','), marshalEntry(book)...)
		}
	}
	if added == nil {
		return subscriptions, true
	}
	end := bytes.LastIndexByte(subscriptions, ']')
	return slices.Concat(subscriptions[:end], added, subscriptions[end:]), true
}

// entryFields are the fields of a saved subscription the upgrade reads
var entryFields = []string{"asset", "channel", "interval", "levels"}

// savedValue is a value saved under a field name the loader could read as one of entryFields, as jsonparser returns it,
// which leaves text escaped
type savedValue struct {
	raw       string
	valueType jsonparser.ValueType
	// bothDecoders reports whether both decoders read the field name as the field
	bothDecoders bool
}

// readEntry reads a saved subscription as the loader does: a missing or null asset or channel is empty, which is spot
// for an asset, and an asset is lower-cased. The interval and levels are read only for an all-assets order book
// subscription. It reports false when the decoders could read the entry differently: when a field it reads is given more
// than once with different values, or under a name only one decoder reads as that field, as the default build's decoder
// folds case and sonic's lower-cases
func readEntry(raw []byte, dataType jsonparser.ValueType) (savedEntry, bool) {
	var entry savedEntry
	if dataType != jsonparser.Object {
		return entry, false
	}
	values := make(map[string][]savedValue, len(entryFields))
	err := jsonparser.ObjectEach(raw, func(key, value []byte, valueType jsonparser.ValueType, _ int) error {
		for _, name := range entryFields {
			if k := string(key); matchesField(k, name) {
				values[name] = append(values[name], savedValue{raw: string(value), valueType: valueType, bothDecoders: matchesBoth(k, name)})
			}
		}
		return nil
	})
	if err != nil {
		return entry, false
	}
	assets, ok := texts(values["asset"])
	if !ok {
		return entry, false
	}
	for i := range assets {
		assets[i] = strings.ToLower(assets[i])
	}
	channels, ok := texts(values["channel"])
	if !ok || !allSame(assets) || !allSame(channels) {
		return entry, false
	}
	if len(assets) > 0 {
		entry.asset = assets[0]
	}
	if len(channels) > 0 {
		entry.channel = channels[0]
	}
	if entry.asset != "all" || entry.channel != "orderbook" {
		return entry, true
	}
	interval, intervalOK := soleValue(values["interval"])
	levels, levelsOK := soleValue(values["levels"])
	if !intervalOK || !levelsOK {
		return entry, false
	}
	entry.futuresBook = takesFuturesInterval(interval) && takesFuturesLevels(levels)
	return entry, true
}

// texts returns the text of each value, a null being empty, and reports false for a value that is neither text nor null
// or saved under a field name only one decoder reads as the field
func texts(values []savedValue) ([]string, bool) {
	out := make([]string, len(values))
	for i, v := range values {
		if !v.bothDecoders {
			return nil, false
		}
		switch v.valueType {
		case jsonparser.String:
			text, err := jsonparser.ParseString([]byte(v.raw))
			if err != nil {
				return nil, false
			}
			out[i] = text
		case jsonparser.Null:
		default:
			return nil, false
		}
	}
	return out, true
}

// soleValue returns the value saved for a field, which is jsonparser.NotExist when none is, and reports false for a
// field given more than once with different values or under a field name only one decoder reads as the field
func soleValue(values []savedValue) (savedValue, bool) {
	if len(values) == 0 {
		return savedValue{valueType: jsonparser.NotExist, bothDecoders: true}, true
	}
	return values[0], values[0].bothDecoders && allSame(values)
}

// takesFuturesInterval reports whether the futures order book streams take the update speed a saved interval reads as:
// none, as they push every 250ms without one, 100ms or 500ms. Values the loader accepts are read as it reads them; one it
// refuses fails the configuration's load, so what this reports for it changes nothing
func takesFuturesInterval(v savedValue) bool {
	var d time.Duration
	switch v.valueType {
	case jsonparser.NotExist:
		return true
	case jsonparser.Number:
		n, err := strconv.ParseInt(v.raw, 10, 64)
		if err != nil {
			return false
		}
		d = time.Duration(n)
	case jsonparser.String:
		var err error
		if d, err = time.ParseDuration(v.raw); err != nil {
			// The loader also reads a whole count of a unit of a second or longer, which is a speed the streams take only
			// when the count is zero
			digits := len(v.raw) - len(strings.TrimLeft(v.raw, "0123456789"))
			return digits > 0 && digits < len(v.raw) && strings.Trim(v.raw[:digits], "0") == ""
		}
	default:
		return false
	}
	return d == 0 || d == 100*time.Millisecond || d == 500*time.Millisecond
}

// takesFuturesLevels reports whether the futures order book streams take the depth saved levels read as: none, as a
// null or missing value reads, or 5, 10 or 20 levels a side
func takesFuturesLevels(v savedValue) bool {
	switch v.valueType {
	case jsonparser.NotExist, jsonparser.Null:
		return true
	case jsonparser.Number:
		n, err := strconv.ParseInt(v.raw, 10, 64)
		return err == nil && (n == 0 || n == 5 || n == 10 || n == 20)
	default:
		return false
	}
}

// allSame reports whether every value is the first
func allSame[T comparable](values []T) bool {
	return !slices.ContainsFunc(values, func(v T) bool { return v != values[0] })
}

// matchesBoth reports whether both decoders read a field name as name
func matchesBoth(key, name string) bool {
	return strings.EqualFold(key, name) && strings.ToLower(key) == name //nolint:gocritic // EqualFold alone misses what sonic's lower-casing matches
}

// matchesField reports whether a field name is one the loader reads as name: the default build's decoder folds field
// names, matching ſ to s, and sonic's lower-cases them, matching İ to i
func matchesField(key, name string) bool {
	return strings.EqualFold(key, name) || strings.ToLower(key) == name //nolint:gocritic // EqualFold alone misses what sonic's lower-casing matches
}

// soleField returns the value of an object's field and the name it is saved under, and whether that field is the only one
// the loader could read as name, both decoders read it as name, as the runtime ignores case in field names, and its value
// has the type given. The default build's decoder folds field names, matching ſ to s, and sonic's lower-cases them,
// matching İ to i, so a field matching either way counts
func soleField(object []byte, name string, valueType jsonparser.ValueType) (value []byte, key string, ok bool) {
	var matches int
	err := jsonparser.ObjectEach(object, func(k, v []byte, dataType jsonparser.ValueType, _ int) error {
		if s := string(k); matchesField(s, name) {
			matches++
			ok = matchesBoth(s, name) && dataType == valueType
			value, key = v, s
		}
		return nil
	})
	return value, key, err == nil && matches == 1 && ok
}

// isDefaults reports whether saved subscriptions are each a different one of the defaults, and every one of them, in
// JSON the loader accepts
func isDefaults(subscriptions []byte, defaults []defaultSubscription) bool {
	if !json.Valid(subscriptions) {
		return false
	}
	matched := make([]bool, len(defaults))
	var count int
	_, err := jsonparser.ArrayEach(subscriptions, func(entry []byte, _ jsonparser.ValueType, _ int, _ error) {
		count++
		if i := slices.IndexFunc(defaults, func(d defaultSubscription) bool { return d.savedAs(entry) }); i != -1 {
			matched[i] = true
		}
	})
	return err == nil && count == len(defaults) && !slices.Contains(matched, false)
}

// savedAs reports whether a saved subscription is the default as GCT saves it: an object with each of the default's
// fields given once and no other, with its asset in any case, as the loader reads assets, or absent for spot, as saved
// before subscriptions had assets
func (d defaultSubscription) savedAs(entry []byte) bool {
	var enabled, channel, asset, interval bool
	err := jsonparser.ObjectEach(entry, func(key, value []byte, dataType jsonparser.ValueType, _ int) error {
		var ok bool
		switch string(key) {
		case "enabled":
			ok = !enabled && dataType == jsonparser.Boolean && string(value) == "true"
			enabled = true
		case "channel":
			ok = !channel && string(value) == d.channel
			channel = true
		case "asset":
			ok = !asset && strings.ToLower(string(value)) == d.asset //nolint:gocritic // asset.New lower-cases; EqualFold's Unicode folding would accept assets it refuses
			asset = true
		case "interval":
			ok = !interval && string(value) == d.interval
			interval = true
		}
		if !ok {
			return errNotDefault
		}
		return nil
	})
	return err == nil && enabled && channel && (asset || d.asset == "spot") && interval == (d.interval != "")
}

// marshal returns subscriptions as GCT saves them
func marshal(subscriptions []defaultSubscription) []byte {
	b := []byte{'['}
	for i, s := range subscriptions {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, marshalEntry(s)...)
	}
	return append(b, ']')
}

// marshalEntry returns a subscription as GCT saves it
func marshalEntry(s defaultSubscription) []byte {
	b := []byte(`{"enabled":true,"channel":"` + s.channel + `","asset":"` + s.asset + `"`)
	if s.interval != "" {
		b = append(b, `,"interval":"`+s.interval+`"`...)
	}
	return append(b, '}')
}
