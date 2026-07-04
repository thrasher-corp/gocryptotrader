package lbank

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWsHandleKbar(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "kbar",
		"pair": "btc_usdt",
		"kbar": {
			"o": "29000.0",
			"h": "29500.0",
			"l": "28800.0",
			"c": "29200.0",
			"v": "100.5",
			"t": 1704067200000,
			"slot": "1min"
		},
		"SERVER": "V2",
		"TS": 1704067200000
	}`))
	assert.NoError(t, err, "wsHandleData kbar should not error")
}

func TestWsHandleOrderUpdate(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "orderUpdate",
		"pair": "btc_usdt",
		"orderUpdate": {
			"amount": "0.5",
			"orderStatus": 2,
			"price": "29000.0",
			"role": "maker",
			"updateTime": 1704067200000,
			"uuid": "test-order-uuid",
			"txUuid": "test-tx-uuid",
			"volumePrice": "14500.0"
		},
		"SERVER": "V2",
		"TS": 1704067200000
	}`))
	assert.NoError(t, err, "wsHandleData orderUpdate should not error")
}

func TestLbankOrderStatusToOrderStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int64
		expected order.Status
		wantErr  bool
	}{
		{0, order.New, false},
		{1, order.PartiallyFilled, false},
		{2, order.Filled, false},
		{3, order.PartiallyCancelled, false},
		{4, order.PendingCancel, false},
		{99, order.UnknownStatus, true},
	}
	for _, tt := range tests {
		status, err := lbankOrderStatusToOrderStatus(tt.input)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, status)
		}
	}
}

func TestKlineIntervalFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected kline.Interval
		wantErr  bool
	}{
		{"1min", kline.OneMin, false},
		{"5min", kline.FiveMin, false},
		{"1hr", kline.OneHour, false},
		{"day", kline.OneDay, false},
		{"invalid", 0, true},
	}
	for _, tt := range tests {
		interval, err := klineIntervalFromString(tt.input)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, interval)
		}
	}
}

func TestWsHandleDataServerError(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{
		"SERVER": "V2",
		"message": "Missing parameter ['pair']",
		"status": "error",
		"TS": 1704067200000
	}`))
	assert.Error(t, err, "server error message should return error")
	assert.Contains(t, err.Error(), "Missing parameter")
}

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
	assert.NoError(t, err, "wsHandleData ticker should not error")
}

func TestWsHandleTrades(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.SetSaveTradeDataStatus(true)

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "eth_usdt",
		"trade": {
			"volume": "0.5",
			"price": "2100.0",
			"direction": "buy",
			"TS": "1704067200000"
		},
		"SERVER": "V2",
		"TS": "1704067200000"
	}`))
	assert.NoError(t, err, "wsHandleData trades should not error")
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
	assert.Len(t, ob.Asks, 2, "orderbook should have 2 asks")
	assert.Len(t, ob.Bids, 2, "orderbook should have 2 bids")
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	assert.NotEmpty(t, subs, "generateSubscriptions should return subscriptions")
}

func TestWsHandleTickerErrors(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "tick",
		"pair": "btc_usdt",
		"tick": "not an object"
	}`))
	assert.Error(t, err, "invalid tick JSON should return error")
}

func TestWsHandleTradesErrors(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.SetSaveTradeDataStatus(true)

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "btc_usdt",
		"trade": {"volume": "0.5", "price": "100", "direction": "invalid", "TS": 1704067200000}
	}`))
	assert.Error(t, err, "invalid direction should return error")
}

func TestWsHandleKbarErrors(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "kbar",
		"pair": "btc_usdt",
		"kbar": {"o": "100", "h": "110", "l": "90", "c": "105", "v": "50", "t": 1704067200000, "slot": "invalid"}
	}`))
	assert.Error(t, err, "invalid interval should return error")
}

func TestWsHandleOrderUpdateErrors(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "orderUpdate",
		"pair": "btc_usdt",
		"orderUpdate": {"amount": "0.5", "orderStatus": 99, "price": "100", "updateTime": 1704067200000, "uuid": "test"}
	}`))
	assert.Error(t, err, "invalid order status should return error")
}
