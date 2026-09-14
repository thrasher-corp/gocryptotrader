package lbank

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWsHandleKbar(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "kbar",
		"pair": "btc_usdt",
		"kbar": {"o": "29000.0","h": "29500.0","l": "28800.0","c": "29200.0","v": "100.5","t": "2026-07-07T12:30:00.000","slot": "1min"},
		"SERVER": "V2",
		"TS": "2026-07-07T12:30:00.000"
	}`))
	require.NoError(t, err, "wsHandleData kbar must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one kbar update")
	exp := kline.Item{
		Exchange: ex.Name,
		Pair:     currency.NewPairWithDelimiter("btc", "usdt", "_"),
		Asset:    asset.Spot,
		Interval: kline.OneMin,
		Candles: []kline.Candle{{
			Time:   time.Date(2026, 7, 7, 12, 30, 0, 0, wsTimeLocation).UTC(),
			Open:   29000.0,
			High:   29500.0,
			Low:    28800.0,
			Close:  29200.0,
			Volume: 100.5,
		}},
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "wsHandleData should send the expected kbar")

	t.Run("invalid interval", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{
			"type": "kbar",
			"pair": "btc_usdt",
			"kbar": {"o": "100","h": "110","l": "90","c": "105","v": "50","t": "2026-07-07T12:30:00.000","slot": "invalid"}
		}`))
		assert.Error(t, err, "invalid interval should return error")
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"kbar","pair":"btc_usdt","kbar":!!!}`))
		assert.Error(t, err, "malformed JSON should return error")
	})
}

func TestWebsocketTimeUnmarshal(t *testing.T) {
	t.Parallel()

	t.Run("valid timestamp", func(t *testing.T) {
		t.Parallel()
		var wt websocketTime
		err := json.Unmarshal([]byte(`"2026-09-06T13:29:00.000"`), &wt)
		require.NoError(t, err)
		assert.False(t, wt.Time().IsZero(), "time should not be zero")
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		var wt websocketTime
		err := json.Unmarshal([]byte(`""`), &wt)
		assert.NoError(t, err, "empty string should not error")
	})

	t.Run("invalid timestamp", func(t *testing.T) {
		t.Parallel()
		var wt websocketTime
		err := json.Unmarshal([]byte(`"not-a-time"`), &wt)
		assert.ErrorIs(t, err, errInvalidWebsocketTime, "should return errInvalidWebsocketTime")
	})
}

func TestWsHandleOrderUpdate(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{"orderUpdate":{"accAmt":"0.4","amount":"0.2","avgPrice":"99.5","orderAmt":"1","orderPrice":"100","orderStatus":1,"price":"99","remainAmt":"0.6","type":"buy","updateTime":1704067200000,"uuid":"test-order-uuid"},"pair":"btc_usdt","type":"orderUpdate","SERVER":"V2","TS":"2024-01-01T08:00:00.000"}`))
	require.NoError(t, err, "wsHandleData must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one order update")
	exp := &order.Detail{
		Exchange:             ex.Name,
		AssetType:            asset.Spot,
		Pair:                 currency.NewPairWithDelimiter("btc", "usdt", "_"),
		Price:                100,
		Amount:               1,
		ExecutedAmount:       0.4,
		RemainingAmount:      0.6,
		AverageExecutedPrice: 99.5,
		Side:                 order.Buy,
		OrderID:              "test-order-uuid",
		Status:               order.PartiallyFilled,
		LastUpdated:          time.UnixMilli(1704067200000),
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "wsHandleData should send the expected order detail")

	t.Run("invalid order status", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{
			"type": "orderUpdate",
			"pair": "btc_usdt",
			"orderUpdate": {"orderAmt": "1.0","orderStatus": 99,"orderPrice": "100","type": "buy","updateTime": 1704067200000,"uuid": "test"}
		}`))
		assert.Error(t, err, "invalid order status should return error")
	})

	t.Run("invalid order side", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{
			"type": "orderUpdate",
			"pair": "btc_usdt",
			"orderUpdate": {"orderAmt": "1.0","orderStatus": 2,"orderPrice": "100","type": "notaside","updateTime": 1704067200000,"uuid": "test"}
		}`))
		assert.Error(t, err, "invalid order side should return error")
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"orderUpdate","pair":"btc_usdt","orderUpdate":!!!}`))
		assert.Error(t, err, "malformed JSON should return error")
	})
}

func TestLbankOrderStatusToOrderStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int64
		expected order.Status
		wantErr  bool
	}{
		{-1, order.Cancelled, false},
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
		{lbankWsKline1Min, kline.OneMin, false},
		{lbankWsKline5Min, kline.FiveMin, false},
		{lbankWsKline15Min, kline.FifteenMin, false},
		{lbankWsKline30Min, kline.ThirtyMin, false},
		{lbankWsKline1Hr, kline.OneHour, false},
		{lbankWsKline4Hr, kline.FourHour, false},
		{lbankWsKlineDay, kline.OneDay, false},
		{lbankWsKlineWeek, kline.OneWeek, false},
		{lbankWsKlineMonth, kline.OneMonth, false},
		{lbankWsKlineYear, kline.OneYear, false},
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
		"TS": "2021-07-26T19:48:03.270"
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
	ex.Name = t.Name()

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "tick",
		"pair": "eth_usdt",
		"tick": {"high": "2149.67","low": "2010.26","latest": "2124.36","vol": "51774.0345"},
		"SERVER": "V2",
		"TS": "2024-01-01T00:00:00.000"
	}`))
	require.NoError(t, err, "wsHandleData ticker must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one ticker update")
	exp := &ticker.Price{
		ExchangeName: ex.Name,
		Pair:         currency.NewPairWithDelimiter("eth", "usdt", "_"),
		AssetType:    asset.Spot,
		High:         2149.67,
		Low:          2010.26,
		Last:         2124.36,
		Volume:       51774.0345,
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "wsHandleData should send the expected ticker")

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"tick","pair":"btc_usdt","tick":!!!}`))
		assert.Error(t, err, "malformed JSON should return error")
	})
}

func TestWsHandleAssetUpdate(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "assetUpdate",
		"data": {
			"asset": "114548.31881315",
			"assetCode": "usdt",
			"free": "97430.6739041",
			"freeze": "17117.64490905",
			"time": 1627300043270,
			"type": "ORDER_CREATE"
		},
		"SERVER": "V2",
		"TS": "2021-07-26T19:48:03.270"
	}`))
	require.NoError(t, err, "wsHandleData assetUpdate must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one asset update")
	exp := accounts.Change{
		AssetType: asset.Spot,
		Balance: accounts.Balance{
			Currency: currency.NewCode("usdt"),
			Total:    114548.31881315,
			Free:     97430.6739041,
			Hold:     17117.64490905,
		},
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "wsHandleData should send the expected asset update")
}

func TestWsHandleTrades(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.SetSaveTradeDataStatus(true)
	ex.SetTradeFeedStatus(true)

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "eth_usdt",
		"trade": {"volume": "0.5","price": "2100.0","direction": "buy","TS": "2026-07-07T12:30:00.000"},
		"SERVER": "V2",
		"TS": "2021-07-26T19:48:03.270"
	}`))
	require.NoError(t, err, "wsHandleData trades must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one trade update")
	exp := trade.Data{
		Exchange:     ex.Name,
		AssetType:    asset.Spot,
		CurrencyPair: currency.NewPairWithDelimiter("eth", "usdt", "_"),
		Price:        2100.0,
		Amount:       0.5,
		Timestamp:    time.Date(2026, 7, 7, 12, 30, 0, 0, wsTimeLocation).UTC(),
		Side:         order.Buy,
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "wsHandleData should send the expected trade")

	t.Run("invalid direction", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{
			"type": "trade",
			"pair": "btc_usdt",
			"trade": {"volume": "0.5","price": "100","direction": "invalid","TS": "2026-07-07T12:30:00.000"}
		}`))
		assert.Error(t, err, "invalid direction should return error")
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"trade","pair":"btc_usdt","trade":!!!}`))
		assert.Error(t, err, "malformed JSON should return error")
	})
}

func TestWsHandleOrderbook(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	ex.Name = t.Name()

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
