// Package v21 replaces Binance's previous default subscriptions, which stream spot alone, with the current defaults,
// which also stream USDT-margined futures, coin-margined futures and options.
package v21

import (
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
)

// Exchanges returns just Binance
func (*Version) Exchanges() []string { return []string{"Binance"} }

// UpgradeExchange replaces the previous default subscriptions with the current defaults
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return replaceDefaults(exchange, previousDefaults, currentDefaults)
}

// DowngradeExchange replaces the current default subscriptions with the previous defaults
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return replaceDefaults(exchange, currentDefaults, previousDefaults)
}

// replaceDefaults replaces saved subscriptions which are one set of defaults, in any order, with another set. Any other
// list is the user's choice and stays as saved, as does a list the loader would reject, or one in a configuration with
// more than one field the loader could read its name, features or subscriptions from, as it ignores case in field names
func replaceDefaults(exchange []byte, from, to []defaultSubscription) ([]byte, error) {
	if _, ok := soleField(exchange, "name", jsonparser.String); !ok {
		return exchange, nil
	}
	features, ok := soleField(exchange, "features", jsonparser.Object)
	if !ok {
		return exchange, nil
	}
	if subscriptions, ok := soleField(features, "subscriptions", jsonparser.Array); !ok || !isDefaults(subscriptions, from) {
		return exchange, nil
	}
	return jsonparser.Set(exchange, marshal(to), "features", "subscriptions")
}

// soleField returns the value of an object's field, and whether name, spelled as GCT saves it, is the only field the
// loader could read as name ignoring case and its value has the type given. The default build's decoder folds field
// names, matching ſ to s, and sonic's lower-cases them, matching İ to i, so a field matching either way counts
func soleField(object []byte, name string, valueType jsonparser.ValueType) ([]byte, bool) {
	var matches int
	var exact bool
	var value []byte
	err := jsonparser.ObjectEach(object, func(key, v []byte, dataType jsonparser.ValueType, _ int) error {
		if k := string(key); strings.EqualFold(k, name) || strings.ToLower(k) == name { //nolint:gocritic // EqualFold alone misses what sonic's lower-casing matches
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
		b = append(b, `{"enabled":true,"channel":"`+s.channel+`","asset":"`+s.asset+`"`...)
		if s.interval != "" {
			b = append(b, `,"interval":"`+s.interval+`"`...)
		}
		b = append(b, '}')
	}
	return append(b, ']')
}
