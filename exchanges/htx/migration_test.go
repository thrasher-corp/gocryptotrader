package htx

import (
	"os"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestHTXConfigurationLifecycle(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		subscriptions string
		disabled      bool
		custom        bool
	}{
		{name: "missing"},
		{name: "null", subscriptions: `null`},
		{name: "empty", subscriptions: `[]`},
		{name: "wildcards", custom: true, subscriptions: `[{"enabled":true,"channel":"ticker","asset":"all"},{"enabled":true,"channel":"orderbook","asset":"ALL"},{"enabled":true,"channel":"candles","asset":"all","interval":"1m"},{"enabled":true,"channel":"allTrades"},{"enabled":true,"channel":"myOrders","asset":"All","authenticated":true},{"enabled":true,"channel":"myTrades","authenticated":true},{"enabled":true,"channel":"myAccount","asset":"","authenticated":true}]`},
		{name: "disabled exchange", disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			original, err := os.ReadFile("../../config_example.json")
			require.NoError(t, err, "shipped configuration must load")
			var previous config.Config
			require.NoError(t, json.Unmarshal(original, &previous), "shipped configuration must decode")
			var legacy config.Exchange
			for _, ex := range previous.Exchanges {
				if ex.Name == "HTX" {
					legacy = ex
					break
				}
			}
			require.Equal(t, "HTX", legacy.Name, "shipped HTX configuration must exist")
			legacy.Name = "Huobi"
			legacy.Enabled = !tc.disabled
			legacy.API.AuthenticatedWebsocketSupport = true
			legacy.API.Endpoints["RestSpotURL"] = "https://custom.example"
			legacy.Features.Subscriptions = nil
			pair := currency.NewBTCUSDT()
			require.NoError(t, legacy.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{pair}, true), "spot enabled pairs must be configured")
			require.NoError(t, legacy.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{pair}, false), "spot available pairs must be configured")
			previous.Version = 17
			previous.Exchanges = []config.Exchange{legacy}
			raw, err := json.Marshal(previous)
			require.NoError(t, err, "previous configuration must encode")
			if tc.subscriptions != "" {
				raw, err = jsonparser.Set(raw, []byte(tc.subscriptions), "exchanges", "[0]", "features", "subscriptions")
				require.NoError(t, err, "legacy selection must install")
			}
			fixture, err := os.CreateTemp("testdata", "htx-lifecycle-*.json")
			require.NoError(t, err, "owned regression fixture must open")
			t.Cleanup(func() {
				_ = fixture.Close()
				assert.NoError(t, os.Remove(fixture.Name()), "owned regression fixture should be removed")
			})
			_, err = fixture.Write(raw)
			require.NoError(t, err, "previous configuration must persist")
			require.NoError(t, fixture.Close(), "fixture must close before loading")
			var loaded config.Config
			require.NoError(t, loaded.ReadConfigFromFile(fixture.Name(), true), "real loader must upgrade previous configuration")
			require.Equal(t, 18, loaded.Version, "loader must select the folded migration")
			require.Len(t, loaded.Exchanges, 1, "loader must retain the selected exchange")
			require.Equal(t, "HTX", loaded.Exchanges[0].Name, "loader must rename Huobi")
			h := new(Exchange)
			h.SetDefaults()
			require.NoError(t, h.Setup(&loaded.Exchanges[0]), "migrated configuration must set up")
			if tc.disabled {
				assert.False(t, h.IsEnabled(), "disabled exchange should remain disabled")
			} else {
				h.Websocket.SetCanUseAuthenticatedEndpoints(true)
				public, err := h.generateSubscriptionsForAsset(asset.Spot, false)
				require.NoError(t, err, "spot public subscriptions must expand")
				private, err := h.generateSubscriptionsForAsset(asset.Spot, true)
				require.NoError(t, err, "spot private subscriptions must expand")
				if tc.custom {
					channels := make([]string, 0, len(public)+len(private))
					for _, sub := range append(public, private...) {
						channels = append(channels, sub.QualifiedChannel)
					}
					assert.ElementsMatch(t, []string{"market.btcusdt.detail", "market.btcusdt.depth.step0", "market.btcusdt.kline.1min", "market.btcusdt.trade.detail", "orders#*", "trade.clearing#*#1", "accounts.update#2"}, channels, "wildcard migration should preserve every version 17 spot topic")
				} else {
					assert.NotEmpty(t, public, "implicit selections should retain spot public defaults")
					for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
						derivatives, err := h.generateSubscriptionsForAsset(a, false)
						require.NoError(t, err, "implicit derivative defaults must expand")
						assert.NotEmpty(t, derivatives, "implicit selections should retain derivative defaults")
					}
				}
			}
			if !tc.disabled {
				require.Contains(t, loaded.Exchanges[0].API.Endpoints, "WebsocketCoinMarginedPrivateURL", "setup must persist the new coin-margined private endpoint")
				require.Contains(t, loaded.Exchanges[0].API.Endpoints, "WebsocketUSDTMarginedPrivateURL", "setup must persist the new USDT-margined private endpoint")
			}
			require.NoError(t, loaded.SaveConfigToFile(fixture.Name()), "runtime configuration must persist")
			var reloaded config.Config
			require.NoError(t, reloaded.ReadConfigFromFile(fixture.Name(), true), "persisted configuration must reload")
			assert.Equal(t, loaded.Exchanges[0].API.Endpoints, reloaded.Exchanges[0].API.Endpoints, "reload should retain persisted endpoints")
			assert.Equal(t, loaded.Exchanges[0].Features.Subscriptions, reloaded.Exchanges[0].Features.Subscriptions, "reload should retain subscription choices")
			restored := new(Exchange)
			restored.SetDefaults()
			require.NoError(t, restored.Setup(&reloaded.Exchanges[0]), "reloaded configuration must set up again")
			saved, err := os.ReadFile(fixture.Name())
			require.NoError(t, err, "persisted configuration must read")
			downgraded, err := versions.Manager.Deploy(t.Context(), saved, 17)
			require.NoError(t, err, "persisted configuration must downgrade")
			var target config.Config
			require.NoError(t, json.Unmarshal(downgraded, &target), "version 17 output must decode")
			assert.Equal(t, 17, target.Version, "downgrade should restore version 17")
			assert.Equal(t, "Huobi", target.Exchanges[0].Name, "downgrade should restore Huobi")
			// These are the endpoint names accepted by upstream/master at 7469b6189.
			// This checks the historical contract; it does not execute an old Huobi binary.
			accepted := []string{
				"RestSpotURL",
				"RestSpotSupplementaryURL",
				"RestUSDTMarginedFuturesURL",
				"RestCoinMarginedFuturesURL",
				"RestUSDCMarginedFuturesURL",
				"RestFuturesURL",
				"RestFuturesSupplementaryURL",
				"RestSandboxURL",
				"RestSwapURL",
				"WebsocketSpotURL",
				"WebsocketCoinMarginedURL",
				"WebsocketUSDTMarginedURL",
				"WebsocketUSDCMarginedURL",
				"WebsocketOptionsURL",
				"WebsocketTradeURL",
				"WebsocketPrivateURL",
				"WebsocketSpotSupplementaryURL",
				"WebsocketFuturesURL",
				"WebsocketFuturesPrivateURL",
				"ChainAnalysisURL",
				"EdgeCase1URL",
				"EdgeCase2URL",
				"EdgeCase3URL",
			}
			for endpoint := range target.Exchanges[0].API.Endpoints {
				assert.Containsf(t, accepted, endpoint, "downgraded endpoint %s should be accepted by version 17", endpoint)
			}
			assert.Equal(t, "https://custom.example", target.Exchanges[0].API.Endpoints["RestSpotURL"], "downgrade should preserve user endpoint choices")
		})
	}
}
