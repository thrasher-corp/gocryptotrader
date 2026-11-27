// Package v25 replaces Binance's previous default subscriptions, which stream spot alone, with the current defaults,
// which also stream USDT-margined futures, coin-margined futures and options. A customised list keeps its choices but
// gains the USDT-margined and coin-margined futures order books: its all-assets subscriptions, which streamed nothing in
// the previous version, are dropped, which leaves each futures asset it does not name to the exchange's runtime
// defaults, and a futures asset it names with no order book subscription gains the default one.
package v25

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // Used instead of gct encoding/json so that we can ensure consistent library functionality between versions
	"errors"
	"slices"
	"strings"

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
	// futuresOrderbooks are the default order book subscriptions of the futures assets every upgraded list streams
	futuresOrderbooks = []defaultSubscription{
		{channel: "orderbook", asset: "usdtmarginedfutures", interval: "100ms"},
		{channel: "orderbook", asset: "coinmarginedfutures", interval: "100ms"},
	}
)

// Exchanges returns just Binance
func (*Version) Exchanges() []string { return []string{"Binance"} }

// UpgradeExchange replaces the previous default subscriptions with the current defaults, and has a customised list
// stream the USDT-margined and coin-margined futures order books. A downgrade leaves such a list as upgraded, as the
// previous version streams nothing for the entries it gained and nothing for those it lost
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	subscriptions, ok := savedSubscriptions(exchange)
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
	return jsonparser.Set(exchange, upgraded, "features", "subscriptions")
}

// DowngradeExchange replaces the current default subscriptions with the previous defaults
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return replaceDefaults(exchange, currentDefaults, previousDefaults)
}

// replaceDefaults replaces saved subscriptions which are one set of defaults, in any order, with another set. Any other
// list is the user's choice and stays as saved, as does a list the loader would reject, or one in a configuration with
// more than one field the loader could read its name, features or subscriptions from, as it ignores case in field names
func replaceDefaults(exchange []byte, from, to []defaultSubscription) ([]byte, error) {
	if subscriptions, ok := savedSubscriptions(exchange); !ok || !isDefaults(subscriptions, from) {
		return exchange, nil
	}
	return jsonparser.Set(exchange, marshal(to), "features", "subscriptions")
}

// savedSubscriptions returns an exchange configuration's saved subscriptions, and whether its name, features and
// subscriptions are each the only field the loader could read them from
func savedSubscriptions(exchange []byte) ([]byte, bool) {
	if _, ok := soleField(exchange, "name", jsonparser.String); !ok {
		return nil, false
	}
	features, ok := soleField(exchange, "features", jsonparser.Object)
	if !ok {
		return nil, false
	}
	return soleField(features, "subscriptions", jsonparser.Array)
}

// savedEntry is a saved subscription's asset and channel as the loader reads them
type savedEntry struct {
	raw            []byte
	asset, channel string
}

// streamFuturesOrderbooks returns saved subscriptions without their all-assets subscriptions, myOrders aside, and with
// the default order book subscription of each USDT-margined or coin-margined futures asset they name, myOrders aside,
// but give no order book subscription of, enabled or not. It reports false, leaving the list as saved, for a list that
// is not valid JSON, holds an entry that is not an object or whose asset or channel is not text, or holds an entry
// whose asset or channel the default build's and sonic's decoders could read differently
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
	kept := slices.DeleteFunc(slices.Clone(entries), func(e savedEntry) bool { return e.asset == "all" && e.channel != "myOrders" })
	changed := len(kept) != len(entries)
	entries = kept
	for _, book := range futuresOrderbooks {
		named := slices.ContainsFunc(entries, func(e savedEntry) bool { return e.asset == book.asset && e.channel != "myOrders" })
		if named && !slices.ContainsFunc(entries, func(e savedEntry) bool { return e.asset == book.asset && e.channel == book.channel }) {
			entries = append(entries, savedEntry{raw: marshalEntry(book), asset: book.asset, channel: book.channel})
			changed = true
		}
	}
	if !changed {
		return subscriptions, true
	}
	out := []byte{'['}
	for i, e := range entries {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, e.raw...)
	}
	return append(out, ']'), true
}

// readEntry reads a saved subscription's asset and channel as the loader does: a missing or null field is empty, which
// is spot for an asset, and an asset is lower-cased. A field given more than once must hold the same value each time,
// and a field name must match asset or channel under both the default build's decoder, which folds case, and sonic's,
// which lower-cases, or under neither; otherwise the decoders could read the entry differently and it reports false
func readEntry(raw []byte, dataType jsonparser.ValueType) (savedEntry, bool) {
	entry := savedEntry{raw: raw}
	if dataType != jsonparser.Object {
		return entry, false
	}
	var assets, channels []string
	ok := true
	err := jsonparser.ObjectEach(raw, func(key, value []byte, valueType jsonparser.ValueType, _ int) error {
		k := string(key)
		isAsset, isChannel := matchesField(k, "asset"), matchesField(k, "channel")
		if !isAsset && !isChannel {
			return nil
		}
		if !matchesBoth(k, "asset") && !matchesBoth(k, "channel") {
			ok = false
			return nil
		}
		var text string
		switch valueType {
		case jsonparser.String:
			s, err := jsonparser.ParseString(value)
			if err != nil {
				return err
			}
			text = s
		case jsonparser.Null:
		default:
			ok = false
		}
		if isAsset {
			assets = append(assets, strings.ToLower(text))
		} else {
			channels = append(channels, text)
		}
		return nil
	})
	if err != nil || !ok || !allSame(assets) || !allSame(channels) {
		return entry, false
	}
	if len(assets) > 0 {
		entry.asset = assets[0]
	}
	if len(channels) > 0 {
		entry.channel = channels[0]
	}
	return entry, true
}

// allSame reports whether every value is the first
func allSame(values []string) bool {
	return !slices.ContainsFunc(values, func(v string) bool { return v != values[0] })
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

// soleField returns the value of an object's field, and whether name, spelled as GCT saves it, is the only field the
// loader could read as name ignoring case and its value has the type given. The default build's decoder folds field
// names, matching ſ to s, and sonic's lower-cases them, matching İ to i, so a field matching either way counts
func soleField(object []byte, name string, valueType jsonparser.ValueType) ([]byte, bool) {
	var matches int
	var exact bool
	var value []byte
	err := jsonparser.ObjectEach(object, func(key, v []byte, dataType jsonparser.ValueType, _ int) error {
		if k := string(key); matchesField(k, name) {
			matches++
			exact = k == name && dataType == valueType
			value = v
		}
		return nil
	})
	return value, err == nil && matches == 1 && exact
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
