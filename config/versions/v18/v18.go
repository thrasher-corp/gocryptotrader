// Package v18 migrates Huobi exchange configurations to HTX and adds derivative defaults.
package v18

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"slices"
	"strings"

	"github.com/buger/jsonparser"
)

// Version implements ExchangeVersion to migrate Huobi configurations to HTX.
type Version struct{}

// Exchanges uses wildcard dispatch because exchange names are case-insensitive at runtime.
func (*Version) Exchanges() []string { return []string{"*"} }

// UpgradeExchange changes the legacy Huobi exchange name to HTX and adds its derivative configuration.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	name, err := jsonparser.GetString(exchange, "name")
	if err != nil {
		return exchange, err
	}
	if !strings.EqualFold(name, "Huobi") && !strings.EqualFold(name, "HTX") {
		return exchange, nil
	}
	exchange, err = jsonparser.Set(exchange, []byte(`"HTX"`), "name")
	if err != nil {
		return exchange, err
	}

	var config map[string]any
	decoder := json.NewDecoder(bytes.NewReader(exchange))
	decoder.UseNumber()
	if err := decoder.Decode(&config); err != nil {
		return exchange, err
	}
	addUSDTMarginedPair(config)
	addDerivativeSubscriptions(config)
	// Patch only the changed subtrees, preserving unrelated key order and numeric text.
	for _, path := range [][]string{{"currencyPairs", "pairs", "usdtmarginedfutures"}, {"features", "subscriptions"}} {
		var value any = config
		for _, key := range path {
			object, ok := value.(map[string]any)
			if !ok {
				value = nil
				break
			}
			value = object[key]
		}
		if value == nil {
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return exchange, err
		}
		exchange, err = jsonparser.Set(exchange, raw, path...)
		if err != nil {
			return exchange, err
		}
	}
	return exchange, nil
}

// addUSDTMarginedPair adds the pair configuration required by the V5 wrapper without replacing user settings.
func addUSDTMarginedPair(config map[string]any) {
	currencyPairs, ok := config["currencyPairs"].(map[string]any)
	if !ok {
		return
	}
	pairs, ok := currencyPairs["pairs"].(map[string]any)
	if !ok {
		return
	}
	for key := range pairs {
		if strings.EqualFold(key, "usdtmarginedfutures") {
			return
		}
	}
	pairs["usdtmarginedfutures"] = map[string]any{
		"assetEnabled": true,
		"enabled":      "BTC-USDT",
		"available":    "BTC-USDT",
		"requestFormat": map[string]any{
			"uppercase": true,
			"delimiter": "-",
		},
		"configFormat": map[string]any{
			"uppercase": true,
			"delimiter": "-",
		},
	}
}

// addDerivativeSubscriptions adds public defaults, leaves private defaults disabled and preserves user entries.
func addDerivativeSubscriptions(config map[string]any) {
	features, hasFeatures := config["features"].(map[string]any)
	if !hasFeatures {
		return
	}
	subscriptions, _ := features["subscriptions"].([]any)
	if len(subscriptions) == 0 {
		// Missing, null and empty selections use runtime defaults, including spot.
		return
	}
	// V17 ignored derivative funding defaults; the market sockets still cannot deliver them.
	subscriptions = slices.DeleteFunc(subscriptions, func(sub any) bool {
		entry, ok := sub.(map[string]any)
		assetName, _ := entry["asset"].(string)
		return ok && len(entry) == 3 && entry["enabled"] == true && entry["channel"] == "fundingRate" &&
			(strings.EqualFold(assetName, "coinmarginedfutures") || strings.EqualFold(assetName, "usdtmarginedfutures"))
	})
	for _, sub := range subscriptions {
		entry, ok := sub.(map[string]any)
		if !ok {
			continue
		}
		assetName, _ := entry["asset"].(string)
		// V17 served only spot, so wildcard scope meant spot for every channel.
		if assetName == "" || strings.EqualFold(assetName, "all") {
			entry["asset"] = "spot"
		}
	}
	for _, item := range []struct {
		asset           string
		publicChannels  []string
		privateChannels []string
	}{
		{
			asset:           "futures",
			publicChannels:  []string{"ticker", "candles", "orderbook", "allTrades"},
			privateChannels: []string{"myOrders", "myTrades", "myAccount", "positions", "triggerOrders"},
		},
		{
			asset:           "coinmarginedfutures",
			publicChannels:  []string{"ticker", "candles", "orderbook", "allTrades"},
			privateChannels: []string{"myOrders", "myTrades", "myAccount", "positions", "triggerOrders"},
		},
		{
			asset:          "usdtmarginedfutures",
			publicChannels: []string{"ticker", "candles", "orderbook", "allTrades"},
			privateChannels: []string{
				"myOrders",
				"tradeUpdates",
				"executionDetails",
				"myAccount",
				"positions",
				"myTrades",
				"triggerOrders",
			},
		},
	} {
		for _, group := range []struct {
			channels      []string
			enabled       bool
			authenticated bool
		}{
			{channels: item.publicChannels, enabled: true},
			{channels: item.privateChannels, authenticated: true},
		} {
			for _, channel := range group.channels {
				found := false
				for _, sub := range subscriptions {
					entry, ok := sub.(map[string]any)
					if !ok {
						continue
					}
					assetName, _ := entry["asset"].(string)
					channelName, _ := entry["channel"].(string)
					authenticated, _ := entry["authenticated"].(bool)
					if strings.EqualFold(assetName, item.asset) && channelName == channel && authenticated == group.authenticated {
						found = true
						break
					}
				}
				if found {
					continue
				}
				sub := map[string]any{
					"enabled": group.enabled,
					"channel": channel,
					"asset":   item.asset,
				}
				if group.authenticated {
					sub["authenticated"] = true
				}
				if channel == "candles" {
					sub["interval"] = "1m"
				}
				subscriptions = append(subscriptions, sub)
			}
		}
	}
	features["subscriptions"] = subscriptions
}

// DowngradeExchange changes the HTX exchange name back to Huobi.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	if name, err := jsonparser.GetString(exchange, "name"); err == nil && strings.EqualFold(name, "HTX") {
		// Setup persists new endpoint keys that version 17 cannot load.
		for _, key := range []string{"WebsocketCoinMarginedPrivateURL", "WebsocketUSDTMarginedPrivateURL"} {
			exchange = jsonparser.Delete(exchange, "api", "urlEndpoints", key)
		}
		return jsonparser.Set(exchange, []byte(`"Huobi"`), "name")
	}
	return exchange, nil
}
