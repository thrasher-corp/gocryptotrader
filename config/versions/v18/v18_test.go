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
	assert.Equal(t, []string{"*"}, new(v18.Version).Exchanges(), "Exchanges should allow case-insensitive name matching within the migration")
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{name: "legacy", in: "Huobi", want: "HTX"},
		{name: "lowercase legacy", in: "huobi", want: "HTX"},
		{name: "uppercase legacy", in: "HUOBI", want: "HTX"},
		{name: "mixed-case legacy", in: "hUoBi", want: "HTX"},
		{name: "unrelated", in: "Kraken", want: "Kraken"},
		{name: "current", in: "HTX", want: "HTX"},
		{name: "lowercase current", in: "htx", want: "HTX"},
		{name: "mixed-case current", in: "hTx", want: "HTX"},
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
	t.Run("unrelated exchange bytes", func(t *testing.T) {
		t.Parallel()
		input := []byte(`{"name":"Kraken", "z":9007199254740993,"features":{"subscriptions":[{"enabled":true,"channel":"ticker"}]},"currencyPairs":{"pairs":{}}, "a":1e3}`)
		out, err := new(v18.Version).UpgradeExchange(t.Context(), input)
		require.NoError(t, err, "unrelated exchanges must pass through wildcard dispatch")
		assert.Equal(t, input, out, "unrelated exchanges should remain byte-identical")
	})

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
		const defaults = `{"enabled":true,"channel":"ticker","asset":"futures"},
{"enabled":true,"channel":"candles","asset":"futures","interval":"1m"},
{"enabled":true,"channel":"orderbook","asset":"futures"},
{"enabled":true,"channel":"allTrades","asset":"futures"},
{"enabled":false,"channel":"myOrders","asset":"futures","authenticated":true},
{"enabled":false,"channel":"myTrades","asset":"futures","authenticated":true},
{"enabled":false,"channel":"myAccount","asset":"futures","authenticated":true},
{"enabled":false,"channel":"positions","asset":"futures","authenticated":true},
{"enabled":false,"channel":"triggerOrders","asset":"futures","authenticated":true},
{"enabled":true,"channel":"ticker","asset":"coinmarginedfutures"},
{"enabled":true,"channel":"candles","asset":"coinmarginedfutures","interval":"1m"},
{"enabled":true,"channel":"orderbook","asset":"coinmarginedfutures"},
{"enabled":true,"channel":"allTrades","asset":"coinmarginedfutures"},
{"enabled":false,"channel":"myOrders","asset":"coinmarginedfutures","authenticated":true},
{"enabled":false,"channel":"myTrades","asset":"coinmarginedfutures","authenticated":true},
{"enabled":false,"channel":"myAccount","asset":"coinmarginedfutures","authenticated":true},
{"enabled":false,"channel":"positions","asset":"coinmarginedfutures","authenticated":true},
{"enabled":false,"channel":"triggerOrders","asset":"coinmarginedfutures","authenticated":true},
{"enabled":true,"channel":"ticker","asset":"usdtmarginedfutures"},
{"enabled":true,"channel":"candles","asset":"usdtmarginedfutures","interval":"1m"},
{"enabled":true,"channel":"orderbook","asset":"usdtmarginedfutures"},
{"enabled":true,"channel":"allTrades","asset":"usdtmarginedfutures"},
{"enabled":false,"channel":"myOrders","asset":"usdtmarginedfutures","authenticated":true},
{"enabled":false,"channel":"tradeUpdates","asset":"usdtmarginedfutures","authenticated":true},
{"enabled":false,"channel":"executionDetails","asset":"usdtmarginedfutures","authenticated":true},
{"enabled":false,"channel":"myAccount","asset":"usdtmarginedfutures","authenticated":true},
{"enabled":false,"channel":"positions","asset":"usdtmarginedfutures","authenticated":true},
{"enabled":false,"channel":"myTrades","asset":"usdtmarginedfutures","authenticated":true},
{"enabled":false,"channel":"triggerOrders","asset":"usdtmarginedfutures","authenticated":true}`
		for _, tc := range []struct{ name, subscriptions, preserved string }{
			{name: "defaults", subscriptions: `{"enabled":true,"channel":"myAccount","authenticated":true}`, preserved: `{"enabled":true,"channel":"myAccount","authenticated":true,"asset":"spot"}`},
			{name: "unauthenticated private channel", subscriptions: `{"enabled":true,"channel":"myOrders","asset":"futures"}`, preserved: `{"enabled":true,"channel":"myOrders","asset":"futures"}`},
			{name: "mixed-case funding defaults", subscriptions: `{"enabled":true,"channel":"myAccount","authenticated":true},{"enabled":true,"channel":"fundingRate","asset":"USDTMarginedFutures"},{"enabled":true,"channel":"fundingRate","asset":"COINMARGINEDFUTURES"}`, preserved: `{"enabled":true,"channel":"myAccount","authenticated":true,"asset":"spot"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				input := []byte(`{"name":"HTX","currencyPairs":{"pairs":{}},"features":{"subscriptions":[` + tc.subscriptions + `]}}`)
				out, err := new(v18.Version).UpgradeExchange(t.Context(), input)
				require.NoError(t, err, "derivative configuration must migrate")
				var cfg struct {
					CurrencyPairs currency.PairsManager `json:"currencyPairs"`
					Features      struct {
						Subscriptions json.RawMessage `json:"subscriptions"`
					} `json:"features"`
				}
				require.NoError(t, json.Unmarshal(out, &cfg), "runtime pair manager and subscriptions must decode")
				assert.Contains(t, cfg.CurrencyPairs.Pairs, asset.USDTMarginedFutures, "USDT-margined pairs should be added")
				assert.JSONEq(t, `[`+tc.preserved+`,`+defaults+`]`, string(cfg.Features.Subscriptions), "complete subscription defaults should retain intervals, authentication and asset scope")
				second, err := new(v18.Version).UpgradeExchange(t.Context(), out)
				require.NoError(t, err, "repeated migration must succeed")
				assert.JSONEq(t, string(out), string(second), "derivative migration should be idempotent")
			})
		}
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
		{name: "lowercase current", in: "htx", want: "Huobi"},
		{name: "mixed-case current", in: "hTx", want: "Huobi"},
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
	t.Run("unrelated exchange bytes", func(t *testing.T) {
		t.Parallel()
		input := []byte(`{"name":"Kraken", "api":{"urlEndpoints":{"WebsocketUSDTMarginedPrivateURL":"wss://custom.example"}}, "a":1e3}`)
		out, err := new(v18.Version).DowngradeExchange(t.Context(), input)
		require.NoError(t, err, "unrelated exchanges must pass through wildcard dispatch")
		assert.Equal(t, input, out, "unrelated endpoint settings and bytes should remain unchanged")
	})
	t.Run("persisted endpoints", func(t *testing.T) {
		t.Parallel()
		input := []byte(`{"name":"HTX","api":{"urlEndpoints":{"WebsocketCoinMarginedPrivateURL":"wss://coin.example","WebsocketUSDTMarginedPrivateURL":"wss://usdt.example","RestSpotURL":"https://custom.example","WebsocketPrivateURL":"wss://spot.example"}}}`)
		out, err := new(v18.Version).DowngradeExchange(t.Context(), input)
		require.NoError(t, err, "persisted endpoints must downgrade")
		assert.JSONEq(t, `{"name":"Huobi","api":{"urlEndpoints":{"RestSpotURL":"https://custom.example","WebsocketPrivateURL":"wss://spot.example"}}}`, string(out), "downgrade should remove only endpoints unavailable in version 17")
	})
}

func TestRegisteredMigration(t *testing.T) {
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
	for _, transition := range []struct {
		name    string
		from    uint16
		to      uint16
		names   []string
		want    string
		wantURL bool
	}{
		{name: "upgrade", from: 17, to: 18, names: []string{"Huobi", "huobi", "HUOBI", "hUoBi", "HTX", "htx", "hTx"}, want: "HTX", wantURL: true},
		{name: "downgrade", from: 18, to: 17, names: []string{"HTX", "htx", "hTx"}, want: "Huobi"},
	} {
		for _, name := range transition.names {
			t.Run(transition.name+" "+name, func(t *testing.T) {
				t.Parallel()
				const unrelated = `{"name":"Kraken","currencyPairs":{"pairs":{}},"features":{"subscriptions":[{"enabled":true,"channel":"ticker"}]},"api":{"urlEndpoints":{"WebsocketUSDTMarginedPrivateURL":"wss://unrelated.example"}},"precision":9007199254740993}`
				input := fmt.Appendf(nil, `{"version":%d,"exchanges":[{"name":%q,"currencyPairs":{"pairs":{}},"features":{"subscriptions":[{"enabled":true,"channel":"ticker"}]},"api":{"urlEndpoints":{"WebsocketUSDTMarginedPrivateURL":"wss://usdt.example","RestSpotURL":"https://custom.example"}}},%s]}`, transition.from, name, unrelated)
				out, err := versions.Manager.Deploy(t.Context(), input, transition.to)
				require.NoError(t, err, "registered migration must handle every accepted exchange name")
				var cfg struct {
					Version   uint16 `json:"version"`
					Exchanges []struct {
						Name          string `json:"name"`
						CurrencyPairs struct {
							Pairs map[string]json.RawMessage `json:"pairs"`
						} `json:"currencyPairs"`
						API struct {
							URLEndpoints map[string]string `json:"urlEndpoints"`
						} `json:"api"`
					} `json:"exchanges"`
				}
				require.NoError(t, json.Unmarshal(out, &cfg), "migrated configuration must decode")
				assert.Equal(t, transition.to, cfg.Version, "migration should advance to the requested version")
				require.Len(t, cfg.Exchanges, 2, "migration must retain both exchanges")
				assert.Equal(t, transition.want, cfg.Exchanges[0].Name, "migration should canonicalise the exchange name")
				assert.Equal(t, "https://custom.example", cfg.Exchanges[0].API.URLEndpoints["RestSpotURL"], "existing spot endpoint should remain unchanged")
				if transition.wantURL {
					assert.Contains(t, cfg.Exchanges[0].API.URLEndpoints, "WebsocketUSDTMarginedPrivateURL", "upgrade should preserve the private endpoint")
					assert.Contains(t, cfg.Exchanges[0].CurrencyPairs.Pairs, "usdtmarginedfutures", "upgrade should apply derivative defaults to every accepted name")
				} else {
					assert.NotContains(t, cfg.Exchanges[0].API.URLEndpoints, "WebsocketUSDTMarginedPrivateURL", "downgrade should remove the unsupported private endpoint")
				}
				var raw struct {
					Exchanges []json.RawMessage `json:"exchanges"`
				}
				require.NoError(t, json.Unmarshal(out, &raw), "unrelated exchange must decode")
				assert.JSONEq(t, unrelated, string(raw.Exchanges[1]), "wildcard dispatch should preserve unrelated exchange settings")
				assert.Contains(t, string(raw.Exchanges[1]), "9007199254740993", "unrelated integer precision should be preserved")
			})
		}
	}
}
