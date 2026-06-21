package lbank

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWsHandleData(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	t.Run("unknown type returns no error", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"unknown","pair":"eth_usdt"}`))
		assert.NoError(t, err)
	})

	t.Run("missing type returns no error", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"pair":"eth_usdt"}`))
		assert.NoError(t, err)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`not json`))
		assert.Error(t, err)
	})
}

func TestWsHandleTicker(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "tick",
		"pair": "eth_usdt",
		"tick": {
			"high": "2149.67",
			"low": "2010.26",
			"latest": "2124.36",
			"vol": "51774.0345",
			"change": "2.66"
		},
		"SERVER": "V2",
		"TS": "2024-01-01T00:00:00.000"
	}`))
	assert.NoError(t, err, "wsHandleData ticker must not error")
}

func TestWsHandleTrades(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "eth_usdt",
		"trade": [
			{
				"date_ms": 1700000000000,
				"amount": "0.5",
				"price": "2100.0",
				"type": "buy",
				"tid": "abc123"
			},
			{
				"date_ms": 1700000001000,
				"amount": "1.0",
				"price": "2099.0",
				"type": "sell",
				"tid": "def456"
			}
		],
		"SERVER": "V2",
		"TS": "2024-01-01T00:00:00.000"
	}`))
	assert.NoError(t, err, "wsHandleData trades must not error")
}

func TestWsHandleOrderbook(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "depth",
		"pair": "btc_usdt",
		"depth": {
			"asks": [
				["2125.0", "1.5"],
				["2126.0", "2.0"]
			],
			"bids": [
				["2124.0", "1.0"],
				["2123.0", "3.0"]
			]
		},
		"SERVER": "V2",
		"TS": "2024-01-01T00:00:00.000"
	}`))
	require.NoError(t, err, "wsHandleData orderbook must not error")

	p, err := currency.NewPairFromString("btc_usdt")
	require.NoError(t, err)
	ob, err := orderbook.Get(ex.Name, p, asset.Spot)
	require.NoError(t, err, "orderbook.Get must not error")
	assert.Len(t, ob.Asks, 2, "orderbook must have 2 asks")
	assert.Len(t, ob.Bids, 2, "orderbook must have 2 bids")
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	assert.NotEmpty(t, subs, "generateSubscriptions must return subscriptions")
}
