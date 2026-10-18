package v18_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v18 "github.com/thrasher-corp/gocryptotrader/config/versions/v18"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Huobi", "HTX"}, new(v18.Version).Exchanges(), "Exchanges should return migrated exchange names")
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{name: "legacy", in: "Huobi", want: "HTX"},
		{name: "unrelated", in: "Kraken", want: "Kraken"},
		{name: "current", in: "HTX", want: "HTX"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(`{"name":"`+tt.in+`"}`))
			require.NoError(t, err, "UpgradeExchange must not error")
			require.NotEmpty(t, out, "UpgradeExchange must return output")
			var config struct {
				Name string `json:"name"`
			}
			require.NoError(t, json.Unmarshal(out, &config), "upgraded exchange must decode")
			assert.Equalf(t, tt.want, config.Name, "exchange name %s should migrate correctly", tt.in)
		})
	}

	for _, key := range []string{"usdtmarginedfutures", "USDTMarginedFutures", "USDTMARGINEDFUTURES"} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("existing %s enabled %t", key, enabled), func(t *testing.T) {
				t.Parallel()
				input := fmt.Appendf(nil, `{"name":"Huobi","currencyPairs":{"pairs":{%q:{"assetEnabled":%t,"enabled":"ETH-USDT","available":"ETH-USDT"}}}}`, key, enabled)
				out, err := new(v18.Version).UpgradeExchange(t.Context(), input)
				require.NoError(t, err, "existing pair settings must migrate")
				var raw struct {
					CurrencyPairs struct {
						Pairs map[string]json.RawMessage `json:"pairs"`
					} `json:"currencyPairs"`
				}
				require.NoError(t, json.Unmarshal(out, &raw), "migrated pair keys must decode")
				require.Len(t, raw.CurrencyPairs.Pairs, 1, "migration must not add a duplicate asset key")
				assert.Contains(t, raw.CurrencyPairs.Pairs, key, "existing asset key spelling should be preserved")
				var loaded struct {
					CurrencyPairs currency.PairsManager `json:"currencyPairs"`
				}
				require.NoError(t, json.Unmarshal(out, &loaded), "runtime pair manager must decode migrated settings")
				store := loaded.CurrencyPairs.Pairs[asset.USDTMarginedFutures]
				require.NotNil(t, store, "runtime must recognise the existing asset key")
				assert.Equal(t, enabled, store.AssetEnabled, "migration should preserve the user's asset selection")
				expected := currency.Pairs{currency.NewPairWithDelimiter("ETH", "USDT", "-")}
				assert.Equal(t, expected, store.Enabled, "migration should preserve enabled pairs")
				assert.Equal(t, expected, store.Available, "migration should preserve available pairs")
				second, err := new(v18.Version).UpgradeExchange(t.Context(), out)
				require.NoError(t, err, "existing pair settings must support repeated migration")
				assert.JSONEq(t, string(out), string(second), "repeated migration should preserve existing settings")
			})
		}
	}

	t.Run("preserve unrelated bytes", func(t *testing.T) {
		t.Parallel()
		input := []byte(`{"name":"Huobi", "z":9007199254740993, "nested":{"z":1,"a":2}, "currencyPairs":{"pairs":{}}, "features":{"subscriptions":[]}, "a":1e3}`)
		out, err := new(v18.Version).UpgradeExchange(t.Context(), input)
		require.NoError(t, err, "migration must succeed")
		assert.Contains(t, string(out), `"name":"HTX", "z":9007199254740993, "nested":{"z":1,"a":2}`, "unrelated keys and integer precision should be preserved")
		assert.Contains(t, string(out), `, "a":1e3}`, "unrelated number formatting should be preserved")
	})
	t.Run("derivative configuration", func(t *testing.T) {
		t.Parallel()
		input := []byte(`{"name":"HTX","currencyPairs":{"pairs":{}},"features":{"subscriptions":[{"enabled":true,"channel":"myAccount","authenticated":true}]}}`)
		out, err := new(v18.Version).UpgradeExchange(t.Context(), input)
		require.NoError(t, err, "UpgradeExchange must not error")
		var config struct {
			CurrencyPairs struct {
				Pairs map[string]any `json:"pairs"`
			} `json:"currencyPairs"`
			Features struct {
				Subscriptions []struct {
					Enabled       bool   `json:"enabled"`
					Channel       string `json:"channel"`
					Asset         string `json:"asset"`
					Authenticated bool   `json:"authenticated"`
				} `json:"subscriptions"`
			} `json:"features"`
		}
		require.NoError(t, json.Unmarshal(out, &config), "upgraded exchange must decode")
		assert.Contains(t, config.CurrencyPairs.Pairs, "usdtmarginedfutures", "USDT-margined pairs should be added")
		assert.Len(t, config.Features.Subscriptions, 30, "spot account and all derivative subscriptions should be retained")
		assert.Equal(t, "spot", config.Features.Subscriptions[0].Asset, "spot account subscription should retain its asset")
		privateCount := 0
		fundingRateCount := 0
		for _, sub := range config.Features.Subscriptions {
			if sub.Channel == "fundingRate" {
				fundingRateCount++
			}
			if sub.Authenticated && sub.Asset != "spot" {
				privateCount++
				assert.False(t, sub.Enabled, "private derivative subscriptions should default to disabled")
			}
		}
		assert.Equal(t, 17, privateCount, "all private derivative subscriptions should be added")
		assert.Zero(t, fundingRateCount, "unsupported funding defaults should not be added")

		second, err := new(v18.Version).UpgradeExchange(t.Context(), out)
		require.NoError(t, err, "repeated UpgradeExchange must not error")
		assert.JSONEq(t, string(out), string(second), "UpgradeExchange should be idempotent")
	})
	t.Run("implicit defaults", func(t *testing.T) {
		t.Parallel()
		for _, field := range []string{"", `"subscriptions":null`, `"subscriptions":[]`} {
			t.Run(field, func(t *testing.T) {
				t.Parallel()
				input := `{"name":"HTX","features":{` + field + `}}`
				out, err := new(v18.Version).UpgradeExchange(t.Context(), []byte(input))
				require.NoError(t, err, "implicit defaults must migrate")
				assert.JSONEq(t, input, string(out), "implicit defaults should remain unchanged")
			})
		}
	})
	t.Run("custom funding choices", func(t *testing.T) {
		t.Parallel()
		input := []byte(`{"name":"HTX","features":{"subscriptions":[{"enabled":false,"channel":"fundingRate","asset":"coinmarginedfutures"},{"enabled":true,"channel":"fundingRate","asset":"usdtmarginedfutures","pairs":["ETH-USDT"]},{"enabled":true,"channel":"fundingRate","asset":"coinmarginedfutures"},{"enabled":true,"channel":"fundingRate","asset":"usdtmarginedfutures"}]}}`)
		out, err := new(v18.Version).UpgradeExchange(t.Context(), input)
		require.NoError(t, err, "custom funding choices must migrate")
		var cfg struct {
			Features struct {
				Subscriptions []map[string]any `json:"subscriptions"`
			} `json:"features"`
		}
		require.NoError(t, json.Unmarshal(out, &cfg), "subscriptions must decode")
		var funding []map[string]any
		for _, sub := range cfg.Features.Subscriptions {
			if sub["channel"] == "fundingRate" {
				funding = append(funding, sub)
			}
		}
		assert.Equal(t, []map[string]any{{"enabled": false, "channel": "fundingRate", "asset": "coinmarginedfutures"}, {"enabled": true, "channel": "fundingRate", "asset": "usdtmarginedfutures", "pairs": []any{"ETH-USDT"}}}, funding, "only exact unsupported defaults should be removed")
	})
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{name: "current", in: "HTX", want: "Huobi"},
		{name: "unrelated", in: "Kraken", want: "Kraken"},
		{name: "legacy", in: "Huobi", want: "Huobi"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v18.Version).DowngradeExchange(t.Context(), []byte(`{"name":"`+tt.in+`"}`))
			require.NoError(t, err, "DowngradeExchange must not error")
			require.NotEmpty(t, out, "DowngradeExchange must return output")
			assert.Equalf(t, `{"name":"`+tt.want+`"}`, string(out), "exchange name %s should migrate correctly", tt.in)
		})
	}
	t.Run("persisted endpoints", func(t *testing.T) {
		t.Parallel()
		input := []byte(`{"name":"HTX","api":{"urlEndpoints":{"WebsocketCoinMarginedPrivateURL":"wss://coin.example","WebsocketUSDTMarginedPrivateURL":"wss://usdt.example","RestSpotURL":"https://custom.example","WebsocketPrivateURL":"wss://spot.example"}}}`)
		out, err := new(v18.Version).DowngradeExchange(t.Context(), input)
		require.NoError(t, err, "persisted endpoints must downgrade")
		assert.JSONEq(t, `{"name":"Huobi","api":{"urlEndpoints":{"RestSpotURL":"https://custom.example","WebsocketPrivateURL":"wss://spot.example"}}}`, string(out), "downgrade should remove only endpoints unavailable in version 17")
	})
}

func TestRegisteredUpgrade(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input string
	}{
		{name: "earlier migrations", input: `{"version":11,"gctscript":{"enabled":true},"exchanges":[{"name":"EXMO"},{"name":"BitMEX"},{"name":"Huobi","enabled":true},{"name":"Kraken","enabled":true}]}`},
		{name: "upstream version 17", input: `{"version":17,"exchanges":[{"name":"Huobi","enabled":true},{"name":"Kraken","enabled":true}]}`},
		{name: "upstream version 16", input: `{"version":16,"exchanges":[{"name":"Huobi","enabled":true},{"name":"Kraken","enabled":true}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := versions.Manager.Deploy(t.Context(), []byte(tc.input), 18)
			require.NoError(t, err, "Deploy must apply the registered upgrades")
			var config struct {
				Version   uint64 `json:"version"`
				Exchanges []struct {
					Name string `json:"name"`
				} `json:"exchanges"`
			}
			require.NoError(t, json.Unmarshal(out, &config), "deployed config must decode")
			assert.Equal(t, uint64(18), config.Version, "Deploy should update the config version")
			require.Len(t, config.Exchanges, 2, "Deploy must retain only the supported exchanges")
			assert.Equal(t, "HTX", config.Exchanges[0].Name, "Deploy should rename Huobi to HTX")
			assert.Equal(t, "Kraken", config.Exchanges[1].Name, "Deploy should preserve unrelated exchanges")
			assert.NotContains(t, string(out), `"gctscript"`, "Deploy should retain upstream's GCTScript removal")
		})
	}
}
