package v19_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v18 "github.com/thrasher-corp/gocryptotrader/config/versions/v18"
	v19 "github.com/thrasher-corp/gocryptotrader/config/versions/v19"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"HTX"}, new(v19.Version).Exchanges(), "migration should target HTX only")
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, subscriptions string
	}{
		{name: "missing"},
		{name: "null", subscriptions: `"subscriptions":null`},
		{name: "empty", subscriptions: `"subscriptions":[]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := []byte(`{"name":"Huobi","enabled":true,"features":{` + tc.subscriptions + `},"currencyPairs":{"pairs":{}},"precision":9007199254740993}`)
			previous, err := new(v18.Version).UpgradeExchange(t.Context(), input)
			require.NoError(t, err, "v18 must produce its historical defaults")
			out, err := new(v19.Version).UpgradeExchange(t.Context(), previous)
			require.NoError(t, err, "v19 must repair generated defaults")
			var migrated config.Exchange
			require.NoError(t, json.Unmarshal(out, &migrated), "exchange must decode through real config types")
			assert.Empty(t, migrated.Features.Subscriptions, "absent spot defaults should retain runtime default selection")
			assert.Contains(t, string(out), `"precision":9007199254740993`, "unrelated integer precision should be retained")
			again, err := new(v19.Version).UpgradeExchange(t.Context(), out)
			require.NoError(t, err, "repeated migration must succeed")
			assert.Equal(t, out, again, "migration should be idempotent")
		})
	}
	t.Run("custom entries", func(t *testing.T) {
		t.Parallel()
		input := []byte(`{"name":"HTX","features":{"subscriptions":[{"enabled":false,"channel":"ticker","asset":"spot"},{"enabled":false,"channel":"fundingRate","asset":"coinmarginedfutures"},{"enabled":true,"channel":"fundingRate","asset":"usdtmarginedfutures","pairs":["ETH-USDT"]},{"enabled":true,"channel":"fundingRate","asset":"coinmarginedfutures"},{"enabled":true,"channel":"fundingRate","asset":"usdtmarginedfutures"}]}}`)
		out, err := new(v19.Version).UpgradeExchange(t.Context(), input)
		require.NoError(t, err, "custom list must migrate")
		assert.JSONEq(t, `{"name":"HTX","features":{"subscriptions":[{"enabled":false,"channel":"ticker","asset":"spot"},{"enabled":false,"channel":"fundingRate","asset":"coinmarginedfutures"},{"enabled":true,"channel":"fundingRate","asset":"usdtmarginedfutures","pairs":["ETH-USDT"]}]}}`, string(out), "only exact unsupported defaults should be removed")
	})
	for _, input := range []string{`{"name":"HTX"}`, `{"name":"HTX","features":{"subscriptions":null}}`, `{"name":"HTX","features":{"subscriptions":[]}}`} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			out, err := new(v19.Version).UpgradeExchange(t.Context(), []byte(input))
			require.NoError(t, err, "unchanged defaults must migrate")
			assert.Equal(t, input, string(out), "implicit and empty selections should remain unchanged")
		})
	}
	for _, input := range []string{`{`, `[]`, `{"features":{"subscriptions":[1]}}`} {
		t.Run("invalid "+input, func(t *testing.T) {
			t.Parallel()
			_, err := new(v19.Version).UpgradeExchange(t.Context(), []byte(input))
			assert.Error(t, err, "invalid config should report a decoding error")
		})
	}
	t.Run("registered upgrade and downgrade", func(t *testing.T) {
		t.Parallel()
		input := []byte(`{"version":17,"exchanges":[{"name":"Huobi","enabled":true,"features":{"subscriptions":null},"currencyPairs":{"pairs":{}}},{"name":"Kraken","features":{"subscriptions":[{"enabled":false,"channel":"ticker","asset":"spot"}]}}]}`)
		out, err := versions.Manager.Deploy(t.Context(), input, 19)
		require.NoError(t, err, "registered v17 to v19 upgrade must succeed")
		var migrated config.Config
		require.NoError(t, json.Unmarshal(out, &migrated), "registered output must decode through Config")
		assert.Equal(t, 19, migrated.Version, "loader should advance to version 19")
		assert.Equal(t, "HTX", migrated.Exchanges[0].Name, "legacy exchange should be renamed")
		assert.Empty(t, migrated.Exchanges[0].Features.Subscriptions, "registered upgrade should preserve spot runtime defaults")
		assert.Len(t, migrated.Exchanges[1].Features.Subscriptions, 1, "unrelated exchanges should retain subscriptions")
		back, err := versions.Manager.Deploy(t.Context(), out, 17)
		require.NoError(t, err, "downgrade to upstream v17 must succeed")
		require.NoError(t, json.Unmarshal(back, &migrated), "downgraded config must decode")
		assert.Equal(t, 17, migrated.Version, "downgrade should restore the upstream version")
		assert.Equal(t, "Huobi", migrated.Exchanges[0].Name, "historical downgrade should restore the exchange name")
	})
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	input := []byte(`{"name":"HTX","features":{"subscriptions":[{"enabled":false,"channel":"ticker","asset":"spot"}]}}`)
	out, err := new(v19.Version).DowngradeExchange(t.Context(), input)
	require.NoError(t, err, "compatible downgrade must succeed")
	assert.Equal(t, input, out, "downgrade should retain repaired compatible subscriptions")
}
