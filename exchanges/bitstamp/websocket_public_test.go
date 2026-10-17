package bitstamp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestHandleWebsocketOrderbook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSOrderbook(t.Context(), []byte(`{"bids":"wibble"}`), wsSpotPair, asset.Spot), "handleWSOrderbook should error on malformed data")

	for _, tc := range []struct {
		data string
		exp  *orderbook.Book
	}{
		{
			data: `{"timestamp":"1580336834","microtimestamp":"1580336834607546","bids":[["9328.28","0.05925332"],["9327.34","0.43120000"],["0","69"]],"asks":[["9337.10","0.03000000"],["9340.85","2.67820000"]]}`,
			exp: &orderbook.Book{
				Bids:              orderbook.Levels{{Price: 9328.28, Amount: 0.05925332}, {Price: 9327.34, Amount: 0.4312}},
				Asks:              orderbook.Levels{{Price: 9337.1, Amount: 0.03}, {Price: 9340.85, Amount: 2.6782}},
				Exchange:          ex.Name,
				Pair:              wsSpotPair,
				Asset:             asset.Spot,
				LastUpdated:       time.UnixMicro(1580336834607546),
				ValidateOrderbook: ex.ValidateOrderbook,
			},
		},
		{
			data: `{"timestamp":"1790318551","microtimestamp":"1790318551908216","bids":[["84010","0.01190"],["84006","0.33643"]],"asks":[["84018","0.33440"],["84021","1.07000"]]}`,
			exp: &orderbook.Book{
				Bids:              orderbook.Levels{{Price: 84010, Amount: 0.0119}, {Price: 84006, Amount: 0.33643}},
				Asks:              orderbook.Levels{{Price: 84018, Amount: 0.3344}, {Price: 84021, Amount: 1.07}},
				Exchange:          ex.Name,
				Pair:              wsPerpetualPair,
				Asset:             asset.PerpetualContract,
				LastUpdated:       time.UnixMicro(1790318551908216),
				ValidateOrderbook: ex.ValidateOrderbook,
			},
		},
	} {
		require.NoErrorf(t, ex.handleWSOrderbook(t.Context(), []byte(tc.data), tc.exp.Pair, tc.exp.Asset), "handleWSOrderbook must not error for %s", tc.exp.Asset)
		ob, err := ex.Websocket.Orderbook.GetOrderbook(tc.exp.Pair, tc.exp.Asset)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", tc.exp.Asset)
		assert.NotZerof(t, ob.InsertedAt, "InsertedAt should be set for %s", tc.exp.Asset)
		ob.InsertedAt = time.Time{}
		assert.Equalf(t, tc.exp, ob, "handleWSOrderbook should load the snapshot without zero priced bids for %s", tc.exp.Asset)
	}
}

func TestHandleWebsocketTrade(t *testing.T) {
	t.Parallel()
	data := []byte(`{"microtimestamp":"1580336751488517","amount":0.00598803,"buy_order_id":4621328909,"sell_order_id":4621329035,"amount_str":"0.00598803","price_str":"9334.73","timestamp":"1580336751","price":9334.73,"type":1,"id":104007706}`)
	ex := newWebsocketTestInstance(t)
	require.NoError(t, ex.handleWSTrade(data, wsSpotPair, asset.Spot), "handleWSTrade must not error with the trade feed disabled")
	assert.Empty(t, dataHandlerPayloads(ex), "trades should not be sent with the trade feed disabled")

	ex = newWebsocketTestInstance(t)
	ex.SetTradeFeedStatus(true)
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	assert.Error(t, ex.handleWSTrade([]byte(`{"price_str":[]}`), wsSpotPair, asset.Spot), "handleWSTrade should error on malformed data")
	require.NoError(t, ex.handleWSTrade(data, wsSpotPair, asset.Spot), "handleWSTrade must not error")
	assert.Equal(t, []any{[]trade.Data{{
		TID:          "104007706",
		Exchange:     ex.Name,
		CurrencyPair: wsSpotPair,
		AssetType:    asset.Spot,
		Side:         order.Sell,
		Price:        9334.73,
		Amount:       0.00598803,
		Timestamp:    time.UnixMicro(1580336751488517),
	}}}, dataHandlerPayloads(ex), "trades should be sent to the data handler")
}

func TestHandleWebsocketFundingRate(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSFundingRate(t.Context(), []byte(`{"funding_rate":[]}`), wsPerpetualPair, asset.PerpetualContract), "handleWSFundingRate should error on malformed data")
	data := []byte(`{"market":"btcusd-perp","mark_price":"83997.21486186","index_price":"83991.59300000002","funding_rate":"0.0001","timestamp":"1790318552","next_funding_time":"1790323200"}`)
	require.NoError(t, ex.handleWSFundingRate(t.Context(), data, wsPerpetualPair, asset.PerpetualContract), "handleWSFundingRate must not error")
	assert.Equal(t, []any{
		websocket.FundingData{
			Timestamp:    time.Unix(1790318552, 0),
			CurrencyPair: wsPerpetualPair,
			AssetType:    asset.PerpetualContract,
			Exchange:     ex.Name,
			Rate:         0.0001,
		},
		&WebsocketFundingRate{
			Pair:            wsPerpetualPair,
			AssetType:       asset.PerpetualContract,
			Market:          "btcusd-perp",
			MarkPrice:       83997.21486186,
			IndexPrice:      83991.59300000002,
			FundingRate:     0.0001,
			Timestamp:       types.Time(time.Unix(1790318552, 0)),
			NextFundingTime: types.Time(time.Unix(1790323200, 0)),
		},
	}, dataHandlerPayloads(ex), "funding rates and updates should be sent to the data handler")
}

func TestHandleWebsocketDetailOrderbook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSDetailOrderbook(t.Context(), []byte(`{"bids":"wibble"}`), wsSpotPair, asset.Spot), "handleWSDetailOrderbook should error on malformed data")
	err := ex.handleWSDetailOrderbook(t.Context(), []byte(`{"bids":[["1","1","9223372036854775808"]]}`), wsSpotPair, asset.Spot)
	assert.ErrorIs(t, err, errOrderIDOverflow, "handleWSDetailOrderbook should error on an order ID beyond an orderbook level ID")
	err = ex.handleWSDetailOrderbook(t.Context(), []byte(`{"asks":[["1","1","9223372036854775808"]]}`), wsSpotPair, asset.Spot)
	assert.ErrorIs(t, err, errOrderIDOverflow, "handleWSDetailOrderbook should error on an ask order ID beyond an orderbook level ID")

	data := `{"timestamp":"1791848800","microtimestamp":"1791848800102345","bids":[["81730.34","0.01000000","2060344614850561"],["81730.34","0.02000000","2060344614850562"],["0","1","1"]],"asks":[["81758.04","0.03000000","2060344614850560"]]}`
	require.NoError(t, ex.handleWSDetailOrderbook(t.Context(), []byte(data), wsSpotPair, asset.Spot), "handleWSDetailOrderbook must not error")
	ob, err := ex.Websocket.Orderbook.GetOrderbook(wsSpotPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.NotZero(t, ob.InsertedAt, "InsertedAt should be set")
	ob.InsertedAt = time.Time{}
	// Each level holds an order, so prices repeat and the zero priced bid is removed
	exp := &orderbook.Book{
		Bids:              orderbook.Levels{{Price: 81730.34, Amount: 0.01, ID: 2060344614850561}, {Price: 81730.34, Amount: 0.02, ID: 2060344614850562}},
		Asks:              orderbook.Levels{{Price: 81758.04, Amount: 0.03, ID: 2060344614850560}},
		Exchange:          ex.Name,
		Pair:              wsSpotPair,
		Asset:             asset.Spot,
		LastUpdated:       time.UnixMicro(1791848800102345),
		PriceDuplication:  true,
		ValidateOrderbook: ex.ValidateOrderbook,
	}
	assert.Equal(t, exp, ob, "handleWSDetailOrderbook should load the snapshot of orders")
}

func TestHandleWebsocketAnnouncement(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSAnnouncement(t.Context(), "published", []byte(`{"announcement_id":"wibble"}`)), "handleWSAnnouncement should error on malformed data")
	data := []byte(`{"announcement_id":12345,"announcement_type":"trading_halt","status":"in_progress","published_at":"2026-08-25T10:00:00Z","updated_at":"2026-08-25T10:00:00Z","product_type":"TOKENIZED_SECURITY","product":["TSLA/USD"],"trading_halt_details":{"reason":"circuit_breaker","start_time":"2026-08-25T10:00:00Z","end_time":null}}`)
	require.NoError(t, ex.handleWSAnnouncement(t.Context(), "in_progress", data), "handleWSAnnouncement must not error")
	haltStart := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, []any{&WebsocketAnnouncement{
		Event:              "in_progress",
		AnnouncementID:     12345,
		AnnouncementType:   "trading_halt",
		Status:             "in_progress",
		PublishedAt:        haltStart,
		UpdatedAt:          haltStart,
		ProductType:        "TOKENIZED_SECURITY",
		Product:            []string{"TSLA/USD"},
		TradingHaltDetails: &TradingHaltDetails{Reason: "circuit_breaker", StartTime: haltStart},
	}}, dataHandlerPayloads(ex), "announcements should be sent to the data handler")
}
