package htx

import (
	"context"
	"strconv"
	"testing"
	"time"
	"uuid"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWebsocketHandleMyAccountMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "accounts.update#2", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.MyAccountChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	testexch.FixtureToDataHandler(t, "testdata/wsMyAccount.jsonl", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 3, "Must see correct number of records")
	exp := []WebsocketAccountUpdate{
		{Currency: currency.BTC.Lower(), AccountID: 123456, Balance: 23.111, ChangeType: "transfer", AccountType: "trade", ChangeTime: types.Time(time.UnixMilli(1568601800000))},
		{Currency: currency.BTC.Lower(), AccountID: 33385, Available: 2028.69, ChangeType: "order.match", AccountType: "trade", ChangeTime: types.Time(time.UnixMilli(1574393385167))},
		{Currency: currency.USDT.Lower(), AccountID: 14884859, Available: 20.29388158, Balance: 20.29388158, AccountType: "trade"},
	}
	for i, ex := range exp {
		var err error
		ex.SequenceNumber, err = types.NewPreciseNumberFromString(strconv.Itoa(i + 1))
		require.NoError(t, err, "expected sequence number must parse")
		uAny := <-e.Websocket.DataHandler.C
		u, ok := uAny.Data.(WebsocketAccountUpdate)
		require.True(t, ok, "Must get the correct type from DataHandler")
		require.NotNil(t, u)
		assert.Equal(t, ex, u)
	}
}

func TestWebsocketHandleMyOrdersMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "orders#*", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.MyOrdersChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	errs := testexch.FixtureToDataHandlerWithErrors(t, "testdata/wsMyOrders.jsonl", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	require.Equal(t, 1, len(errs), "Must receive the correct number of errors back")
	require.ErrorContains(t, errs[0].Err, "error with order \"test1\": invalid.client.order.id (NT) (2002)")
	require.Len(t, e.Websocket.DataHandler.C, 4, "Must see correct number of records")
	exp := []*order.Detail{
		{
			Exchange:      e.Name,
			Pair:          btcusdtPair,
			Side:          order.Buy,
			Status:        order.Rejected,
			ClientOrderID: "test1",
			AssetType:     asset.Spot,
			LastUpdated:   time.UnixMicro(1583853365586000),
		},
		{
			Exchange:      e.Name,
			Pair:          btcusdtPair,
			Side:          order.Buy,
			Status:        order.Cancelled,
			ClientOrderID: "test2",
			AssetType:     asset.Spot,
			LastUpdated:   time.UnixMicro(1583853365586000),
		},
		{
			Exchange:      e.Name,
			Pair:          btcusdtPair,
			Side:          order.Sell,
			Status:        order.New,
			ClientOrderID: "test3",
			AssetType:     asset.Spot,
			Price:         77,
			Amount:        2,
			Type:          order.Limit,
			OrderID:       "27163533",
			LastUpdated:   time.UnixMicro(1583853365586000),
		},
		{
			Exchange:    e.Name,
			Pair:        btcusdtPair,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Price:       70000,
			Amount:      0.000157,
			Type:        order.Limit,
			OrderID:     "1199329381585359",
			LastUpdated: time.UnixMicro(1731039387696000),
		},
	}
	for _, ex := range exp {
		m := <-e.Websocket.DataHandler.C
		require.IsType(t, &order.Detail{}, m.Data, "Must get the correct type from DataHandler")
		d, _ := m.Data.(*order.Detail)
		require.NotNil(t, d)
		assert.Equal(t, ex, d, "Order Detail should match")
	}
	t.Run("order quantity units", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, fields                                      string
			quote, executedQuote, amount, executed, remaining float64
			status                                            order.Status
		}{
			{name: "market buy creation", fields: `"eventType":"creation","type":"buy-market","orderValue":"100","orderStatus":"submitted"`, quote: 100, status: order.New},
			{name: "market buy trade", fields: `"eventType":"trade","type":"buy-market","orderValue":"100.3","execAmt":"60.1","remainAmt":"40.2","tradePrice":"60000","tradeVolume":"0.0005","orderStatus":"partial-filled"`, quote: 100.3, executedQuote: 60.1, status: order.PartiallyFilled},
			{name: "market buy partial cancellation", fields: `"eventType":"cancellation","type":"buy-market","orderValue":"100","execAmt":"60","remainAmt":"40","orderStatus":"partial-canceled"`, quote: 100, executedQuote: 60, status: order.PartiallyCancelled},
			{name: "market buy cancellation without fills", fields: `"eventType":"cancellation","type":"buy-market","orderValue":"100","execAmt":"0","remainAmt":"100","orderStatus":"canceled"`, quote: 100, status: order.Cancelled},
			{name: "market buy final trade", fields: `"eventType":"trade","type":"buy-market","orderValue":"100","execAmt":"100","remainAmt":"0","tradePrice":"60000","tradeVolume":"0.0007","orderStatus":"filled"`, quote: 100, executedQuote: 100, status: order.Filled},
			{name: "limit maker buy", fields: `"eventType":"trade","type":"buy-limit-maker","orderSize":"2","execAmt":"1.5","remainAmt":"0.5","orderStatus":"partial-filled"`, amount: 2, executed: 1.5, remaining: 0.5, status: order.PartiallyFilled},
			{name: "market sell", fields: `"eventType":"trade","type":"sell-market","orderSize":"2","execAmt":"1.5","remainAmt":"0.5","orderStatus":"partial-filled"`, amount: 2, executed: 1.5, remaining: 0.5, status: order.PartiallyFilled},
			{name: "IOC buy", fields: `"eventType":"trade","type":"buy-ioc","orderSize":"2","execAmt":"1.5","remainAmt":"0.5","orderStatus":"partial-filled"`, amount: 2, executed: 1.5, remaining: 0.5, status: order.PartiallyFilled},
			{name: "FOK buy", fields: `"eventType":"trade","type":"buy-limit-fok","orderSize":"2","execAmt":"1.5","remainAmt":"0.5","orderStatus":"partial-filled"`, amount: 2, executed: 1.5, remaining: 0.5, status: order.PartiallyFilled},
			{name: "market sell cancellation", fields: `"eventType":"cancellation","type":"sell-market","orderSize":"2","execAmt":"1.5","remainAmt":"0.5","orderStatus":"partial-canceled"`, amount: 2, executed: 1.5, remaining: 0.5, status: order.PartiallyCancelled},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				detail := readSpotOrderUpdate(t, ex, tc.fields)
				assert.Equal(t, tc.status, detail.Status, "status should match the venue update")
				assert.Equal(t, tc.quote, detail.QuoteAmount, "quote amount should hold the reported budget")
				assert.Equal(t, tc.executedQuote, detail.ExecutedQuoteAmount, "executed quote amount should hold cumulative quote execution")
				assert.Equal(t, tc.amount, detail.Amount, "amount should hold only the requested base quantity")
				assert.Equal(t, tc.executed, detail.ExecutedAmount, "executed amount should hold only cumulative base execution")
				assert.Equal(t, tc.remaining, detail.RemainingAmount, "remaining amount should hold only the unfilled base quantity")
			})
		}
	})

	t.Run("market buy merges stream and REST execution", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ws := readSpotOrderUpdate(t, ex, `"eventType":"trade","type":"buy-market","orderValue":"100","execAmt":"60","remainAmt":"40","orderStatus":"partial-filled"`)
		submit, err := (&order.Submit{Exchange: ex.Name, Pair: btcusdtPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 100}).DeriveSubmitResponse("27163536")
		require.NoError(t, err, "the acknowledgement must map to a submit response")
		for _, firstSeen := range []string{"stream", "submit"} {
			t.Run(firstSeen, func(t *testing.T) {
				t.Parallel()
				stored := ws.Copy()
				if firstSeen == "submit" {
					placed, err := submit.DeriveDetail(uuid.NewV4())
					require.NoError(t, err, "the acknowledgement must map to an order detail")
					stored = *placed
					require.NoError(t, stored.UpdateOrderFromDetail(ws), "the quote-only update must merge into the acknowledged order")
				}
				assert.True(t, stored.IsActive(), "the partial fill should stay eligible for polling")
				assert.Equal(t, 60.0, stored.ExecutedQuoteAmount, "quote execution should be recorded before REST reports the base fill")
				assert.Zero(t, stored.AverageExecutedPrice, "quote execution alone should leave average price unknown")

				require.NoError(t, stored.UpdateOrderFromDetail(&order.Detail{Status: order.PartiallyFilled, ExecutedAmount: 0.0005, ExecutedQuoteAmount: 30}), "an older REST snapshot must merge safely")
				assert.Zero(t, stored.ExecutedAmount, "the stale snapshot should not install a base fill from an older cumulative total")
				assert.Equal(t, 60.0, stored.ExecutedQuoteAmount, "the stale snapshot should not rewind cumulative quote execution")

				require.NoError(t, stored.UpdateOrderFromDetail(&order.Detail{Status: order.PartiallyFilled, ExecutedAmount: 0.001, ExecutedQuoteAmount: 60}), "a REST snapshot at the same quote total must merge")
				assert.Equal(t, 0.001, stored.ExecutedAmount, "REST should enrich the order with its base fills")
				assert.Equal(t, 60000.0, stored.AverageExecutedPrice, "matching base and quote totals should establish the average price")

				require.NoError(t, stored.UpdateOrderFromDetail(&order.Detail{Status: order.PartiallyFilled, ExecutedQuoteAmount: 80}), "the next quote-only update must merge")
				assert.Equal(t, 80.0, stored.ExecutedQuoteAmount, "the next update should retain quote progress")
				assert.Zero(t, stored.ExecutedAmount, "quote progress should invalidate the base fill from the earlier total")
				assert.Zero(t, stored.AverageExecutedPrice, "quote progress should invalidate the average from the earlier fill")

				require.NoError(t, stored.UpdateOrderFromDetail(&order.Detail{Status: order.Filled, ExecutedAmount: 0.002, ExecutedQuoteAmount: 100}), "the final REST snapshot must merge")
				assert.Equal(t, order.Filled, stored.Status, "the final REST status should take effect")
				assert.Equal(t, 0.002, stored.ExecutedAmount, "the final snapshot should report the base fill")
				assert.Equal(t, 100.0, stored.ExecutedQuoteAmount, "the final snapshot should report the quote fill")
				assert.False(t, stored.IsActive(), "the filled order should leave the polling set")
			})
		}
	})
}

// readSpotOrderUpdate reads exactly one order detail from the private stream handler.
func readSpotOrderUpdate(t *testing.T, ex *Exchange, fields string) *order.Detail {
	t.Helper()
	sub := &subscription.Subscription{Asset: asset.Spot}
	require.NoError(t, ex.wsHandleMyOrdersMsg(t.Context(), sub, []byte(`{"action":"push","ch":"orders#btcusdt","data":{"orderId":27163536,"symbol":"btcusdt",`+fields+`}}`)), "the handler must accept the order update")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "the handler must relay one update")
	detail, ok := (<-ex.Websocket.DataHandler.C).Data.(*order.Detail)
	require.True(t, ok, "the update must contain order details")
	return detail
}

func TestWebsocketHandleMyTradesMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "trade.clearing#btcusdt#1", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.MyTradesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	testexch.FixtureToDataHandler(t, "testdata/wsMyTrades.jsonl", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	m := <-e.Websocket.DataHandler.C
	exp := &order.Detail{
		Exchange:      e.Name,
		Pair:          btcusdtPair,
		Side:          order.Buy,
		Status:        order.PartiallyFilled,
		ClientOrderID: "a001",
		OrderID:       "99998888",
		AssetType:     asset.Spot,
		Date:          time.UnixMicro(1583853365586000),
		LastUpdated:   time.UnixMicro(1583853365996000),
		Price:         10000,
		Amount:        1,
		Trades: []order.TradeHistory{
			{
				Price:     9999.99,
				Amount:    0.96,
				Fee:       19.88,
				FeeAsset:  "btc",
				Exchange:  e.Name,
				TID:       "919219323232",
				Side:      order.Buy,
				IsMaker:   false,
				Timestamp: time.UnixMicro(1583853365996000),
			},
		},
	}
	require.IsType(t, &order.Detail{}, m.Data, "Must get the correct type from DataHandler")
	d, _ := m.Data.(*order.Detail)
	require.NotNil(t, d)
	assert.Equal(t, exp, d, "Order Detail should match")
}

func TestWebsocketGenerateSignature(t *testing.T) {
	t.Parallel()
	h := testexch.MockWebsocketInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketPrivate)
	require.NoError(t, err, "private spot websocket connection must be available")
	signature, err := h.wsGenerateSignature(conn, &accounts.Credentials{Key: "key", Secret: "secret"}, "2026-07-30T00:00:00")
	require.NoError(t, err, "spot websocket signature generation must not error")
	assert.NotEmpty(t, signature, "spot websocket signature should not be empty")
	_, err = h.wsGenerateSignature(nil, &accounts.Credentials{}, "")
	require.ErrorIs(t, err, common.ErrNilPointer, "spot websocket signature generation must require a connection")
	_, err = h.wsGenerateSignature(conn, nil, "")
	require.ErrorIs(t, err, common.ErrNilPointer, "spot websocket signature generation must require credentials")
}

func TestWebsocketLogin(t *testing.T) {
	t.Parallel()
	h := testexch.MockWebsocketInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	conn, err := h.Websocket.GetConnection(exchange.WebsocketPrivate)
	require.NoError(t, err, "private spot websocket connection must be available")
	h.Websocket.SetCanUseAuthenticatedEndpoints(false)
	require.NoError(t, h.wsLogin(t.Context(), conn), "spot websocket login must not error")
	assert.True(t, h.Websocket.CanUseAuthenticatedEndpoints(), "successful login should enable authenticated endpoints")
}

func TestWebsocketHandleV2Ping(t *testing.T) {
	t.Parallel()
	messages := make(chan []byte, 1)
	h := testexch.MockWebsocketInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, msg []byte, _ *gws.Conn) error {
		messages <- msg
		return nil
	}))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketPrivate)
	require.NoError(t, err, "private spot websocket connection must be available")
	require.NoError(t, h.wsHandleV2ping(t.Context(), conn, []byte(`{"action":"ping","data":{"ts":123}}`)), "V2 ping handler must not error")
	select {
	case msg := <-messages:
		assert.JSONEq(t, `{"action":"pong","data":{"ts":123}}`, string(msg), "V2 ping handler should send the matching pong")
	case <-time.After(time.Second):
		require.Fail(t, "V2 pong must be sent")
	}
	require.ErrorIs(t, h.wsHandleV2ping(t.Context(), nil, []byte(`{}`)), common.ErrNilPointer, "V2 ping handler must reject a nil connection")
	require.Error(t, h.wsHandleV2ping(t.Context(), conn, []byte(`{}`)), "V2 ping handler must require a timestamp")
}
