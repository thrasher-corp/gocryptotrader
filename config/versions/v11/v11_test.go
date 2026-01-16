package v11_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v11 "github.com/thrasher-corp/gocryptotrader/config/versions/v11"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Poloniex"}, new(v11.Version).Exchanges())
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		in      string
		urlType string
		exp     string
	}{
		{"https://poloniex.com", "RestSpotURL", ""},
		{"https://poloniex.private-proxy.com", "RestSpotURL", `"RestSpotURL": "https://poloniex.private-proxy.com"`},
		{"wss://api2.poloniex.com", "WebsocketSpotURL", ""},
		{"wss://poloniex.private-proxy.com", "WebsocketSpotURL", `"WebsocketSpotURL": "wss://poloniex.private-proxy.com"`},
	} {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			in := []byte(`{"name":"Poloniex","api":{"urlEndpoints":{"` + tt.urlType + `": "` + tt.in + `"}}}`)
			out, err := new(v11.Version).UpgradeExchange(t.Context(), in)
			require.NoError(t, err)
			exp := `{"name":"Poloniex","api":{"urlEndpoints":{` + tt.exp + `}}}`
			assert.Equal(t, exp, string(out))
		})
	}

	in := []byte(`{"name":"Poloniex","api":{}`)
	out, err := new(v11.Version).UpgradeExchange(t.Context(), in)
	require.NoError(t, err, "UpgradeExchange must not error when urlEndpoints is missing")
	assert.Equal(t, string(in), string(out), "UpgradeExchange should return same input and no error when urlEndpoints is missing")

	_, err = new(v11.Version).UpgradeExchange(t.Context(), []byte(`{"name":"Poloniex","api":{"urlEndpoints":{"WebsocketSpotURL": 42}}}`))
	require.ErrorContains(t, err, "Value is not a string", "UpgradeExchange must error correctly on string value")
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	in := []byte(`{"name":"Poloniex","api":{"urlEndpoints":{"WebsocketSpotURL": 42}}}`)
	out, err := new(v11.Version).DowngradeExchange(t.Context(), in)
	require.NoError(t, err)
	require.Equal(t, string(in), string(out), "DowngradeExchange must not change json")
}
