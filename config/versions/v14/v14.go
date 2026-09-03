// Package v14 migrates Huobi exchange configurations to HTX and adds derivative defaults.
package v14

import (
	"context"
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"strings"

	"github.com/buger/jsonparser"
)

// Version implements ExchangeVersion to migrate Huobi configurations to HTX.
type Version struct{}

// Exchanges returns the legacy and current exchange names handled by this migration.
func (*Version) Exchanges() []string { return []string{"Huobi", "HTX"} }

// UpgradeExchange changes the legacy Huobi exchange name to HTX and adds its derivative configuration.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	if name, err := jsonparser.GetString(exchange, "name"); err == nil && name == "Huobi" {
		var setErr error
		exchange, setErr = jsonparser.Set(exchange, []byte(`"HTX"`), "name")
		if setErr != nil {
			return exchange, setErr
		}
	}

	var config map[string]any
	if err := json.Unmarshal(exchange, &config); err != nil {
		return exchange, err
	}
	name, _ := config["name"].(string)
	if !strings.EqualFold(name, "HTX") {
		return exchange, nil
	}

	addUSDTMarginedPair(config)
	addDerivativeSubscriptions(config)
	return json.Marshal(config)
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
	if _, found := pairs["usdtmarginedfutures"]; found {
		return
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
	for _, sub := range subscriptions {
		entry, ok := sub.(map[string]any)
		if !ok {
			continue
		}
		channel, _ := entry["channel"].(string)
		assetName, _ := entry["asset"].(string)
		if channel == "myAccount" && assetName == "" {
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
			publicChannels:  []string{"ticker", "candles", "orderbook", "allTrades", "public.%s.funding_rate"},
			privateChannels: []string{"myOrders", "myTrades", "myAccount", "positions", "triggerOrders"},
		},
		{
			asset:          "usdtmarginedfutures",
			publicChannels: []string{"ticker", "candles", "orderbook", "allTrades", "public.%s.funding_rate"},
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
					if assetName == item.asset && channelName == channel && authenticated == group.authenticated {
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
	if name, err := jsonparser.GetString(exchange, "name"); err == nil && name == "HTX" {
		return jsonparser.Set(exchange, []byte(`"Huobi"`), "name")
	}
	return exchange, nil
}
