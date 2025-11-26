package v8_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/thrasher-corp/gocryptotrader/config/versions/v8"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Bitmex"}, new(v8.Version).Exchanges())
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		in  string
		exp string
	}{
		{"wss://private.bitmex.com/realtimemd", `"WebsocketSpotURL": "wss://private.bitmex.com/realtimemd"`},
		{"wss://ws.bitmex.com/realtimemd", ""},
		{"wss://www.bitmex.com/realtimemd", ""},
		{"wss://www.bitmex.com/realtime", ""},
		{"wss://ws.testnet.bitmex.com/realtimemd", `"WebsocketSpotURL": "wss://ws.testnet.bitmex.com/realtime"`},
		{"wss://testnet.bitmex.com/realtimemd", `"WebsocketSpotURL": "wss://ws.testnet.bitmex.com/realtime"`},
		{"wss://testnet.bitmex.com/realtime", `"WebsocketSpotURL": "wss://ws.testnet.bitmex.com/realtime"`},
	} {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			in := []byte(`{"name":"Bitmex","api":{"urlEndpoints":{"WebsocketSpotURL": "` + tt.in + `"}}}`)
			out, err := new(v8.Version).UpgradeExchange(t.Context(), in)
			require.NoError(t, err)
			exp := `{"name":"Bitmex","api":{"urlEndpoints":{` + tt.exp + `}}}`
			assert.Equal(t, exp, string(out))
		})
	}

	in := []byte(`{"name":"Bitmex","api":{}`)
	out, err := new(v8.Version).UpgradeExchange(t.Context(), in)
	require.NoError(t, err, "UpgradeExchange must not error when urlEndpoints is missing")
	assert.Equal(t, string(in), string(out), "UpgradeExchange should return same input not error when urlEndpoints is missing")

	_, err = new(v8.Version).UpgradeExchange(t.Context(), []byte(`{"name":"Bitmex","api":{"urlEndpoints":{"WebsocketSpotURL": 42}}}`))
	require.ErrorContains(t, err, "Value is not a string", "UpgradeExchange must error correctly on string value")
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	in := []byte(`{"name":"Bitmex","api":{"urlEndpoints":{"WebsocketSpotURL": 42}}}`)
	out, err := new(v8.Version).DowngradeExchange(t.Context(), in)
	require.NoError(t, err)
	require.Equal(t, string(in), string(out), "DowngradeExchange must not change json")
}
