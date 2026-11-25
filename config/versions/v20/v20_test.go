package v20_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v20 "github.com/thrasher-corp/gocryptotrader/config/versions/v20"
)

const (
	previousDefaults = `{"RestSpotURL":"https://api.lbkex.com"}`
	currentDefaults  = `{"RestSpotURL":"https://api.lbank.info","RestUSDTMarginedFuturesURL":"https://lbkperp.lbank.com","WebsocketSpotURL":"wss://api.lbank.info/ws/V2/"}`
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"*"}, new(v20.Version).Exchanges(), "Exchanges should return every exchange, so LBank matches in any case")
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	// Cases outside the table run serially and in order, so a migration keeping state between calls fails them
	ctx := t.Context()
	for _, name := range []string{"Lbank", "lbank", "LBANK", "LBank"} {
		t.Run("LBank named "+name+" is migrated", func(t *testing.T) {
			in := []byte(`{"name":"` + name + `","api":{"urlEndpoints":` + previousDefaults + `}}`)
			out, err := new(v20.Version).UpgradeExchange(ctx, in)
			require.NoErrorf(t, err, "UpgradeExchange must not error for %s", name)
			assert.JSONEqf(t, `{"name":"`+name+`","api":{"urlEndpoints":{}}}`, string(out), "UpgradeExchange should migrate %s", name)
		})
	}
	for _, name := range []string{"Bitstamp", "Lbankx", "L bank"} {
		t.Run("exchange named "+name+" is left unchanged", func(t *testing.T) {
			in := []byte(`{"name":"` + name + `","api":{"urlEndpoints":` + previousDefaults + `}}`)
			out, err := new(v20.Version).UpgradeExchange(ctx, in)
			require.NoErrorf(t, err, "UpgradeExchange must not error for %s", name)
			assert.Equalf(t, string(in), string(out), "UpgradeExchange should leave %s unchanged", name)
		})
	}

	for _, tc := range []struct {
		name string
		in   string
		exp  string
	}{
		{
			name: "the previous default is removed",
			in:   `{"RestSpotURL":"https://api.lbkex.com"}`,
			exp:  `{}`,
		},
		{
			name: "a custom endpoint is kept",
			in:   `{"RestSpotURL":"https://proxy.example/lbank"}`,
			exp:  `{"RestSpotURL":"https://proxy.example/lbank"}`,
		},
		{
			name: "a variant of the previous default is the user's choice and is kept",
			in:   `{"RestSpotURL":"https://api.lbkex.com/"}`,
			exp:  `{"RestSpotURL":"https://api.lbkex.com/"}`,
		},
		{
			name: "the current default and other endpoints are kept",
			in:   `{"RestSpotURL":"https://api.lbank.info","EdgeCase1URL":"https://api.lbkex.com"}`,
			exp:  `{"RestSpotURL":"https://api.lbank.info","EdgeCase1URL":"https://api.lbkex.com"}`,
		},
		{
			name: "a placeholder is kept",
			in:   `{"RestSpotURL":"NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"}`,
			exp:  `{"RestSpotURL":"NON_DEFAULT_HTTP_LINK_TO_EXCHANGE_API"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v20.Version).UpgradeExchange(t.Context(), []byte(`{"name":"Lbank","api":{"urlEndpoints":`+tc.in+`}}`))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, `{"name":"Lbank","api":{"urlEndpoints":`+tc.exp+`}}`, string(out), "UpgradeExchange should migrate the endpoint")
		})
	}

	t.Run("no urlEndpoints", func(t *testing.T) {
		in := []byte(`{"name":"Lbank","api":{}}`)
		out, err := new(v20.Version).UpgradeExchange(ctx, in)
		require.NoError(t, err, "UpgradeExchange must not error when urlEndpoints is missing")
		assert.Equal(t, string(in), string(out), "UpgradeExchange should not change a config without urlEndpoints")
	})
	t.Run("non-string endpoint", func(t *testing.T) {
		_, err := new(v20.Version).UpgradeExchange(ctx, []byte(`{"name":"Lbank","api":{"urlEndpoints":{"RestSpotURL":42}}}`))
		assert.ErrorContains(t, err, "Value is not a string", "UpgradeExchange should error on a non-string endpoint")
	})
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	// Cases outside the table run serially and in order, so a migration keeping state between calls fails them
	ctx := t.Context()
	for _, name := range []string{"Lbank", "lbank", "LBANK", "LBank"} {
		t.Run("LBank named "+name+" is migrated", func(t *testing.T) {
			in := []byte(`{"name":"` + name + `","api":{"urlEndpoints":` + currentDefaults + `}}`)
			out, err := new(v20.Version).DowngradeExchange(ctx, in)
			require.NoErrorf(t, err, "DowngradeExchange must not error for %s", name)
			assert.JSONEqf(t, `{"name":"`+name+`","api":{"urlEndpoints":{}}}`, string(out), "DowngradeExchange should migrate %s", name)
		})
	}
	for _, name := range []string{"Bitstamp", "Lbankx", "L bank"} {
		t.Run("exchange named "+name+" is left unchanged", func(t *testing.T) {
			in := []byte(`{"name":"` + name + `","api":{"urlEndpoints":` + currentDefaults + `}}`)
			out, err := new(v20.Version).DowngradeExchange(ctx, in)
			require.NoErrorf(t, err, "DowngradeExchange must not error for %s", name)
			assert.Equalf(t, string(in), string(out), "DowngradeExchange should leave %s unchanged", name)
		})
	}

	for _, tc := range []struct {
		name string
		in   string
		exp  string
	}{
		{
			name: "the current defaults are removed",
			in:   currentDefaults,
			exp:  `{}`,
		},
		{
			name: "custom endpoints are kept",
			in:   `{"RestSpotURL":"https://proxy.example/lbank","RestUSDTMarginedFuturesURL":"https://proxy.example/perp","WebsocketSpotURL":"wss://proxy.example/ws"}`,
			exp:  `{"RestSpotURL":"https://proxy.example/lbank","RestUSDTMarginedFuturesURL":"https://proxy.example/perp","WebsocketSpotURL":"wss://proxy.example/ws"}`,
		},
		{
			name: "the previous default and other endpoints are kept",
			in:   `{"RestSpotURL":"https://api.lbkex.com","EdgeCase1URL":"https://api.lbank.info"}`,
			exp:  `{"RestSpotURL":"https://api.lbkex.com","EdgeCase1URL":"https://api.lbank.info"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v20.Version).DowngradeExchange(t.Context(), []byte(`{"name":"Lbank","api":{"urlEndpoints":`+tc.in+`}}`))
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, `{"name":"Lbank","api":{"urlEndpoints":`+tc.exp+`}}`, string(out), "DowngradeExchange should migrate the endpoints")
		})
	}

	t.Run("no api", func(t *testing.T) {
		in := []byte(`{"name":"Lbank"}`)
		out, err := new(v20.Version).DowngradeExchange(ctx, in)
		require.NoError(t, err, "DowngradeExchange must not error when api is missing")
		assert.Equal(t, string(in), string(out), "DowngradeExchange should not change a config without endpoints")
	})
	t.Run("non-string endpoint", func(t *testing.T) {
		_, err := new(v20.Version).DowngradeExchange(ctx, []byte(`{"name":"Lbank","api":{"urlEndpoints":{"WebsocketSpotURL":true}}}`))
		assert.ErrorContains(t, err, "Value is not a string", "DowngradeExchange should error on a non-string endpoint")
	})
}

// TestRoundTrip checks a custom endpoint survives an upgrade and downgrade unchanged, and the previous default comes back
// as the previous version's default
func TestRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	for _, tc := range []struct {
		name string
		in   string
	}{
		{"custom endpoint", `{"name":"Lbank","api":{"urlEndpoints":{"RestSpotURL":"https://proxy.example/lbank"}}}`},
		{"default endpoint", `{"name":"Lbank","api":{"urlEndpoints":{}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			up, err := new(v20.Version).UpgradeExchange(ctx, []byte(tc.in))
			require.NoError(t, err, "UpgradeExchange must not error")
			down, err := new(v20.Version).DowngradeExchange(ctx, up)
			require.NoError(t, err, "DowngradeExchange must not error")
			assert.JSONEq(t, tc.in, string(down), "a config should round trip unchanged")
		})
	}
}
