package htx

import (
	"context"
	"testing"
	"time"

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

func TestWSHandleMyAccountMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "accounts.update#2", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.MyAccountChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	testexch.FixtureToDataHandler(t, "testdata/wsMyAccount.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 3, "Must see correct number of records")
	exp := []WsAccountUpdate{
		{Currency: "btc", AccountID: 123456, Balance: 23.111, ChangeType: "transfer", AccountType: "trade", ChangeTime: types.Time(time.UnixMilli(1568601800000)), SeqNum: 1},
		{Currency: "btc", AccountID: 33385, Available: 2028.69, ChangeType: "order.match", AccountType: "trade", ChangeTime: types.Time(time.UnixMilli(1574393385167)), SeqNum: 2},
		{Currency: "usdt", AccountID: 14884859, Available: 20.29388158, Balance: 20.29388158, AccountType: "trade", SeqNum: 3},
	}
	for _, ex := range exp {
		uAny := <-e.Websocket.DataHandler.C
		u, ok := uAny.Data.(WsAccountUpdate)
		require.True(t, ok, "Must get the correct type from DataHandler")
		require.NotNil(t, u)
		assert.Equal(t, ex, u)
	}
}

func TestWSHandleMyOrdersMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "orders#*", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.MyOrdersChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	errs := testexch.FixtureToDataHandlerWithErrors(t, "testdata/wsMyOrders.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
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
}

func TestWSHandleMyTradesMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "trade.clearing#btcusdt#1", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.MyTradesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	testexch.FixtureToDataHandler(t, "testdata/wsMyTrades.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
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

func TestWSGenerateSignature(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
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

func TestWSLogin(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	conn, err := h.Websocket.GetConnection(exchange.WebsocketPrivate)
	require.NoError(t, err, "private spot websocket connection must be available")
	h.Websocket.SetCanUseAuthenticatedEndpoints(false)
	require.NoError(t, h.wsLogin(t.Context(), conn), "spot websocket login must not error")
	assert.True(t, h.Websocket.CanUseAuthenticatedEndpoints(), "successful login should enable authenticated endpoints")
}

func TestWSHandleV2Ping(t *testing.T) {
	t.Parallel()
	messages := make(chan []byte, 1)
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, msg []byte, _ *gws.Conn) error {
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
