// Package v19 repairs HTX subscription defaults introduced by version 18.
package v19

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // Migrations retain stable standard-library JSON behaviour.
	"reflect"

	"github.com/buger/jsonparser"
)

// Version preserves implicit spot defaults and removes unsupported funding defaults.
type Version struct{}

// Exchanges returns the exchange whose subscription defaults need repairing.
func (*Version) Exchanges() []string { return []string{"HTX"} }

// UpgradeExchange recognises the complete v18-generated list before restoring
// implicit defaults. Custom lists retain all entries except exact funding defaults
// that cannot be delivered by the configured market sockets.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(exchange, &object); err != nil {
		return exchange, err
	}
	raw, kind, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	if kind != jsonparser.Array {
		return exchange, nil
	}
	if err != nil {
		return exchange, err
	}
	var subs []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&subs); err != nil {
		return exchange, err
	}
	var defaults []map[string]any
	for _, scope := range []struct {
		asset   string
		private []string
	}{
		{"futures", []string{"myOrders", "myTrades", "myAccount", "positions", "triggerOrders"}},
		{"coinmarginedfutures", []string{"myOrders", "myTrades", "myAccount", "positions", "triggerOrders"}},
		{"usdtmarginedfutures", []string{"myOrders", "tradeUpdates", "executionDetails", "myAccount", "positions", "myTrades", "triggerOrders"}},
	} {
		for _, channel := range []string{"ticker", "candles", "orderbook", "allTrades"} {
			entry := map[string]any{"enabled": true, "channel": channel, "asset": scope.asset}
			if channel == "candles" {
				entry["interval"] = "1m"
			}
			defaults = append(defaults, entry)
		}
		if scope.asset != "futures" {
			defaults = append(defaults, map[string]any{"enabled": true, "channel": "fundingRate", "asset": scope.asset})
		}
		for _, channel := range scope.private {
			defaults = append(defaults, map[string]any{"enabled": false, "channel": channel, "asset": scope.asset, "authenticated": true})
		}
	}
	if reflect.DeepEqual(subs, defaults) {
		// v18 added this exact list to missing, null and empty subscriptions,
		// replacing runtime defaults with derivative-only subscriptions.
		return jsonparser.Delete(exchange, "features", "subscriptions"), nil
	}
	kept := make([]map[string]any, 0, len(subs))
	for _, sub := range subs {
		if len(sub) == 3 && sub["enabled"] == true && sub["channel"] == "fundingRate" &&
			(sub["asset"] == "coinmarginedfutures" || sub["asset"] == "usdtmarginedfutures") {
			continue
		}
		kept = append(kept, sub)
	}
	if len(kept) == len(subs) {
		return exchange, nil
	}
	encoded, err := json.Marshal(kept)
	if err != nil {
		return exchange, err
	}
	return jsonparser.Set(exchange, encoded, "features", "subscriptions")
}

// DowngradeExchange retains the compatible repaired list rather than restoring
// subscriptions that the previous market sockets could not deliver either.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return exchange, nil
}
