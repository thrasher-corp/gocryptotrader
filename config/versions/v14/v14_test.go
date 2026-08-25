package v14_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v14 "github.com/thrasher-corp/gocryptotrader/config/versions/v14"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Huobi", "HTX"}, new(v14.Version).Exchanges(), "Exchanges should return migrated exchange names")
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
			out, err := new(v14.Version).UpgradeExchange(t.Context(), []byte(`{"name":"`+tt.in+`"}`))
			require.NoError(t, err, "UpgradeExchange must not error")
			require.NotEmpty(t, out, "UpgradeExchange must return output")
			var config struct {
				Name string `json:"name"`
			}
			require.NoError(t, json.Unmarshal(out, &config), "upgraded exchange must decode")
			assert.Equalf(t, tt.want, config.Name, "exchange name %s should migrate correctly", tt.in)
		})
	}

	t.Run("derivative configuration", func(t *testing.T) {
		t.Parallel()
		input := []byte(`{"name":"HTX","currencyPairs":{"pairs":{}},"features":{"subscriptions":[{"enabled":true,"channel":"myAccount","authenticated":true}]}}`)
		out, err := new(v14.Version).UpgradeExchange(t.Context(), input)
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
		assert.Len(t, config.Features.Subscriptions, 32, "spot account and all derivative subscriptions should be retained")
		assert.Equal(t, "spot", config.Features.Subscriptions[0].Asset, "spot account subscription should retain its asset")
		privateCount := 0
		for _, sub := range config.Features.Subscriptions {
			if sub.Authenticated && sub.Asset != "spot" {
				privateCount++
				assert.False(t, sub.Enabled, "private derivative subscriptions should default to disabled")
			}
		}
		assert.Equal(t, 17, privateCount, "all private derivative subscriptions should be added")

		second, err := new(v14.Version).UpgradeExchange(t.Context(), out)
		require.NoError(t, err, "repeated UpgradeExchange must not error")
		assert.JSONEq(t, string(out), string(second), "UpgradeExchange should be idempotent")
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
			out, err := new(v14.Version).DowngradeExchange(t.Context(), []byte(`{"name":"`+tt.in+`"}`))
			require.NoError(t, err, "DowngradeExchange must not error")
			require.NotEmpty(t, out, "DowngradeExchange must return output")
			assert.Equalf(t, `{"name":"`+tt.want+`"}`, string(out), "exchange name %s should migrate correctly", tt.in)
		})
	}
}

func TestRegisteredUpgrade(t *testing.T) {
	t.Parallel()
	input := []byte(`{"version":11,"exchanges":[{"name":"EXMO"},{"name":"Huobi","enabled":true},{"name":"Kraken","enabled":true}]}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 14)
	require.NoError(t, err, "Deploy must apply the registered v12, v13 and v14 upgrades")
	var config struct {
		Version   uint64 `json:"version"`
		Exchanges []struct {
			Name string `json:"name"`
		} `json:"exchanges"`
	}
	require.NoError(t, json.Unmarshal(out, &config), "deployed config must decode")
	assert.Equal(t, uint64(14), config.Version, "Deploy should update the config version")
	require.Len(t, config.Exchanges, 2, "Deploy must remove EXMO")
	assert.Equal(t, "HTX", config.Exchanges[0].Name, "Deploy should rename Huobi to HTX")
	assert.Equal(t, "Kraken", config.Exchanges[1].Name, "Deploy should preserve unrelated exchanges")
}
