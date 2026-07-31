package htx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWSHandleCandleMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.kline.1min", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.CandlesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsCandles.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	cAny := <-e.Websocket.DataHandler.C
	c, ok := cAny.Data.(kline.Item)
	require.True(t, ok, "Must get the correct type from DataHandler")
	require.Len(t, c.Candles, 1, "Candles must contain a single candle")
	exp := kline.Item{
		Exchange: e.Name,
		Asset:    asset.Spot,
		Pair:     btcusdtPair,
		Interval: 0,
		Candles: []kline.Candle{{
			Time:   time.UnixMilli(1489474082831),
			Open:   7962.62,
			Close:  8014.56,
			High:   14962.77,
			Low:    5110.14,
			Volume: 4.4,
		}},
	}
	assert.Equal(t, exp, c)
}

func TestWSHandleOrderbookMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.depth.step0", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.OrderbookChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsOrderbook.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	dAny := <-e.Websocket.DataHandler.C
	d, ok := dAny.Data.(*orderbook.Depth)
	require.True(t, ok, "Must get the correct type from DataHandler")
	require.NotNil(t, d)
	l, err := d.GetAskLength()
	require.NoError(t, err, "GetAskLength must not error")
	assert.Equal(t, 2, l, "Ask length should be correct")
	liq, _, err := d.TotalAskAmounts()
	require.NoError(t, err, "TotalAskAmount must not error")
	assert.Equal(t, 0.502591, liq, "Ask Liquidity should be correct")
	l, err = d.GetBidLength()
	require.NoError(t, err, "GetBidLength must not error")
	assert.Equal(t, 2, l, "Bid length should be correct")
	liq, _, err = d.TotalBidAmounts()
	require.NoError(t, err, "TotalBidAmount must not error")
	assert.Equal(t, 0.56281, liq, "Bid Liquidity should be correct")
}

// TestWSHandleAllTradesMsg ensures wsHandleAllTrades sends trade.Data to the ws.DataHandler
func TestWSHandleAllTradesMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.trade.detail", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.AllTradesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	exp := []trade.Data{
		{
			Exchange:     e.Name,
			CurrencyPair: btcusdtPair,
			Timestamp:    time.UnixMilli(1630994963173).UTC(),
			Price:        52648.62,
			Amount:       0.006754,
			Side:         order.Buy,
			TID:          "102523573486",
			AssetType:    asset.Spot,
		},
		{
			Exchange:     e.Name,
			CurrencyPair: btcusdtPair,
			Timestamp:    time.UnixMilli(1630994963184).UTC(),
			Price:        52648.73,
			Amount:       0.006755,
			Side:         order.Sell,
			TID:          "102523573487",
			AssetType:    asset.Spot,
		},
	}
	require.Len(t, e.Websocket.DataHandler.C, 2, "Must see correct number of trades")
	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case trade.Data:
			i := 1 - len(e.Websocket.DataHandler.C)
			require.Equalf(t, exp[i], v, "Trade [%d] must be correct", i)
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T(%s)", v, v)
		}
	}
	require.Empty(t, e.Websocket.DataHandler.C, "Must not see any errors going to datahandler")
}

func TestWSHandleTickerMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.detail", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.TickerChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsTicker.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	tickAny := <-e.Websocket.DataHandler.C
	tick, ok := tickAny.Data.(*ticker.Price)
	require.True(t, ok, "Must get the correct type from DataHandler")
	require.NotNil(t, tick)
	exp := &ticker.Price{
		High:         52924.14,
		Low:          51000,
		Bid:          0,
		Volume:       13991.028076056185,
		QuoteVolume:  7.27676440200527e+08,
		Open:         51823.62,
		Close:        52379.99,
		Pair:         btcusdtPair,
		ExchangeName: e.Name,
		AssetType:    asset.Spot,
		LastUpdated:  time.UnixMilli(1630998026649),
	}
	assert.Equal(t, exp, tick)
}

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

func TestStringToOrderSide(t *testing.T) {
	t.Parallel()
	type TestCases struct {
		Case   string
		Result order.Side
	}
	testCases := []TestCases{
		{Case: "buy-limit", Result: order.Buy},
		{Case: "sell-limit", Result: order.Sell},
		{Case: "woah-nelly", Result: order.UnknownSide},
	}
	for i := range testCases {
		result, _ := stringToOrderSide(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestStringToOrderType(t *testing.T) {
	t.Parallel()
	type TestCases struct {
		Case   string
		Result order.Type
	}
	testCases := []TestCases{
		{Case: "buy-limit", Result: order.Limit},
		{Case: "sell-market", Result: order.Market},
		{Case: "woah-nelly", Result: order.UnknownType},
	}
	for i := range testCases {
		result, _ := stringToOrderType(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestGenerateSpotSubscriptions(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	e.Features.Subscriptions = defaultSubscriptions.Clone()

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := e.generateSubscriptionsForAsset(asset.Spot, false)
	require.NoError(t, err, "public generateSubscriptionsForAsset must not error")
	privateSubs, err := e.generateSubscriptionsForAsset(asset.Spot, true)
	require.NoError(t, err, "private generateSubscriptionsForAsset must not error")
	subs = append(subs, privateSubs...)
	exp := subscription.List{}
	for _, s := range e.Features.Subscriptions {
		if s.Asset == asset.Empty {
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.QualifiedChannel = channelName(s)
			exp = append(exp, s)
			continue
		}
		for _, a := range e.GetAssetTypes(true) {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			pairs = common.SortStrings(pairs).Format(currency.PairFormat{Uppercase: false, Delimiter: ""})
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.Asset = a
			if isWildcardChannel(s) {
				s.Pairs = pairs
				s.QualifiedChannel = channelName(s)
				exp = append(exp, s)
				continue
			}
			for i, p := range pairs {
				s := s.Clone() //nolint:govet // Intentional lexical scope shadow
				s.QualifiedChannel = channelName(s, p)
				switch s.Channel {
				case subscription.OrderbookChannel:
					s.QualifiedChannel += ".step0"
				case subscription.CandlesChannel:
					s.QualifiedChannel += ".1min"
				}
				s.Pairs = pairs[i : i+1]
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	tmpl, err := h.GetSubscriptionTemplate(&subscription.Subscription{})
	require.NoError(t, err, "GetSubscriptionTemplate must not error")
	assert.NotNil(t, tmpl, "subscription template should be returned")
}

func wsFixture(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	operation, _ := jsonparser.GetString(msg, "op")
	topic, _ := jsonparser.GetString(msg, "topic")
	if operation == wsAuthChannel {
		return w.WriteMessage(gws.TextMessage, []byte(`{"op":"auth","type":"api","ts":1489474081631,"err-code":0,"data":{"user-id":"12345678"}}`))
	}
	if operation == wsSubOp || operation == wsUnsubOp {
		return w.WriteMessage(gws.TextMessage, []byte(`{"op":"`+operation+`","topic":"`+topic+`","ts":1489474081631,"err-code":0}`))
	}
	if operation == "pong" {
		return nil
	}
	cid, _ := jsonparser.GetString(msg, "cid")
	switch operation {
	case "place_order", "cancel_order":
		return w.WriteMessage(gws.TextMessage, []byte(`{"code":200,"message":"Success","cid":"`+cid+`","data":{"order_id":"1","client_order_id":"2"},"rate_limit":{"limit":"24","interval":"3000","remaining":"19","reset":"1772779491561"}}`))
	case "place_batch_orders", "cancel_batch_orders", "cancel_all_orders":
		return w.WriteMessage(gws.TextMessage, []byte(`{"code":200,"message":"Success","cid":"`+cid+`","data":[{"order_id":"1","client_order_id":"2"}],"rate_limit":{"limit":"24","interval":"3000","remaining":"19","reset":"1772779491561"}}`))
	}
	action, _ := jsonparser.GetString(msg, "action")
	ch, _ := jsonparser.GetString(msg, "ch")
	if action == "req" && ch == "auth" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"action":"req","code":200,"ch":"auth","data":{}}`))
	}
	if action == "sub" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"action":"sub","code":200,"ch":"`+ch+`"}`))
	}
	id, _ := jsonparser.GetString(msg, "id")
	sub, _ := jsonparser.GetString(msg, "sub")
	if id != "" && sub != "" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"id":"`+id+`","status":"ok","subbed":"`+sub+`"}`))
	}
	unsub, _ := jsonparser.GetString(msg, "unsub")
	if id != "" && unsub != "" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"id":"`+id+`","status":"ok","unsubbed":"`+unsub+`"}`))
	}
	return fmt.Errorf("%w: %s", errUnhandledMockWebsocketMessage, msg)
}

func TestWSHandleData(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.wsHandleData(t.Context(), nil, []byte(`{"unexpected":true}`)), "unmatched websocket data must be forwarded")
	message := <-h.Websocket.DataHandler.C
	assert.IsType(t, websocket.UnhandledMessageWarning{}, message.Data, "unmatched websocket data should produce a warning")
	require.ErrorIs(t, h.wsHandleData(t.Context(), nil, []byte(`{"ping":123}`)), common.ErrNilPointer, "websocket ping data must require a connection")
}

func TestWSHandleV2SubResp(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketPrivate)
	require.NoError(t, err, "private spot websocket connection must be available")
	require.NoError(t, h.wsHandleV2subResp(conn, wsSubOp, []byte(`{}`)), "subscription response without a channel must be ignored")
	require.ErrorIs(t, h.wsHandleV2subResp(nil, wsSubOp, []byte(`{}`)), common.ErrNilPointer, "subscription response must require a connection")
}

func TestWSHandleChannelMsgs(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	sub := &subscription.Subscription{
		Asset:            asset.Spot,
		Channel:          subscription.CandlesChannel,
		Pairs:            currency.Pairs{currency.NewBTCUSDT()},
		Interval:         kline.OneMin,
		QualifiedChannel: "market.btcusdt.kline.1min",
	}
	testexch.FixtureToDataHandler(t, "testdata/wsCandles.json", func(ctx context.Context, data []byte) error {
		return h.wsHandleChannelMsgs(ctx, sub, data)
	})
}

func TestManageSubs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		channel      string
		topic        string
		contractCode string
	}{
		{name: "orders", channel: subscription.MyOrdersChannel, topic: "orders", contractCode: "*"},
		{name: "account", channel: subscription.MyAccountChannel, topic: "account"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, func(tb testing.TB, msg []byte, w *gws.Conn) error {
				tb.Helper()
				operation, err := jsonparser.GetString(msg, "op")
				require.NoError(t, err, "subscription operation must decode")
				topic, err := jsonparser.GetString(msg, "topic")
				require.NoError(t, err, "subscription topic must decode")
				contractCode, _ := jsonparser.GetString(msg, "contract_code")
				assert.Equal(t, tc.topic, topic, "V5 subscription topic should match")
				assert.Equal(t, tc.contractCode, contractCode, "V5 subscription contract code should match")
				return w.WriteMessage(gws.TextMessage, []byte(`{"op":"`+operation+`","topic":"`+topic+`","ts":1489474081631,"err-code":0}`))
			}))
			conn, err := h.Websocket.GetConnection(exchange.WebsocketUSDTMarginedPrivate)
			require.NoError(t, err, "private USDT-margined websocket connection must be available")
			sub := &subscription.Subscription{
				Asset:            asset.USDTMarginedFutures,
				Channel:          tc.channel,
				Authenticated:    true,
				QualifiedChannel: tc.topic,
			}
			require.NoError(t, h.manageSubs(t.Context(), conn, wsSubOp, subscription.List{sub}), "V5 subscription must not error")
			assert.Equal(t, subscription.SubscribedState, sub.State(), "V5 subscription should be active")
		})
	}

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.ErrorIs(t, h.manageSubs(t.Context(), nil, wsSubOp, nil), subscription.ErrBatchingNotSupported, "empty subscription batches must be rejected")
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
	require.NoError(t, h.wsLogin(t.Context(), conn), "spot websocket login must not error")
}

func TestWSHandleV1Ping(t *testing.T) {
	t.Parallel()
	messages := make(chan []byte, 1)
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, msg []byte, _ *gws.Conn) error {
		messages <- msg
		return nil
	}))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot websocket connection must be available")
	require.NoError(t, h.wsHandleV1ping(t.Context(), conn, 123), "V1 ping handler must not error")
	select {
	case msg := <-messages:
		assert.JSONEq(t, `{"pong":123}`, string(msg), "V1 ping handler should send the matching pong")
	case <-time.After(time.Second):
		require.Fail(t, "V1 pong must be sent")
	}
	require.ErrorIs(t, h.wsHandleV1ping(t.Context(), nil, 123), common.ErrNilPointer, "V1 ping handler must reject a nil connection")
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

// TestSubscribe exercises public subscriptions against the deterministic websocket fixture.
func TestSubscribe(t *testing.T) {
	t.Parallel()
	e := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	subs, err := defaultSubscriptions.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	err = e.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")
	conn, err := e.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot websocket connection must be available")
	got := conn.Subscriptions().List()
	require.Equal(t, 8, len(got), "Must get correct number of subscriptions")
	for _, s := range got {
		assert.Equal(t, subscription.SubscribedState, s.State())
	}
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	subs, err := defaultSubscriptions.ExpandTemplates(h)
	require.NoError(t, err, "ExpandTemplates must not error")
	require.NoError(t, h.Subscribe(subs), "Subscribe must not error")
	require.NoError(t, h.Unsubscribe(subs), "Unsubscribe must not error")
	for _, endpoint := range []exchange.URL{exchange.WebsocketSpot, exchange.WebsocketPrivate} {
		conn, err := h.Websocket.GetConnection(endpoint)
		require.NoErrorf(t, err, "%s connection must be available", endpoint)
		assert.Emptyf(t, conn.Subscriptions().List(), "%s subscriptions should be empty", endpoint)
	}
}

// TestAuthSubscribe exercises mock subscriptions including private
func TestAuthSubscribe(t *testing.T) {
	t.Parallel()
	subCfg := defaultSubscriptions.Clone()
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	h.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := subCfg.ExpandTemplates(h)
	require.NoError(t, err, "ExpandTemplates must not error")
	err = h.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")
	publicConn, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot public websocket connection must be available")
	privateConn, err := h.Websocket.GetConnection(exchange.WebsocketPrivate)
	require.NoError(t, err, "spot private websocket connection must be available")
	got := append(publicConn.Subscriptions().List(), privateConn.Subscriptions().List()...)
	require.Equal(t, 11, len(got), "Must get correct number of subscriptions")
	for _, s := range got {
		assert.Equal(t, subscription.SubscribedState, s.State())
	}
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "market.BTC-USD.kline", channelName(&subscription.Subscription{Channel: subscription.CandlesChannel}, btcusdPair))
	assert.Equal(t, "trade.clearing#*#1", channelName(&subscription.Subscription{Channel: subscription.MyTradesChannel}, btcusdPair))
	for _, tt := range []struct {
		channel string
		want    string
	}{
		{channel: subscription.MyOrdersChannel, want: "orders"},
		{channel: wsTradeUpdatesChannel, want: "trade"},
		{channel: wsExecutionDetailsChannel, want: "trade_detail"},
		{channel: wsPositionsChannel, want: "positions"},
		{channel: subscription.MyAccountChannel, want: "account"},
		{channel: subscription.MyTradesChannel, want: "match_orders"},
		{channel: wsTriggerOrdersChannel, want: "algo_orders"},
	} {
		t.Run(tt.channel, func(t *testing.T) {
			t.Parallel()
			sub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: tt.channel, Authenticated: true}
			assert.Equal(t, tt.want, channelName(sub), "private USDT-margined channel should use the documented V5 topic")
		})
	}
	assert.Equal(t, "orders.*", channelName(&subscription.Subscription{
		Asset:         asset.CoinMarginedFutures,
		Channel:       subscription.MyOrdersChannel,
		Authenticated: true,
	}), "coin-margined private channels should retain the documented wildcard topic")
	assert.Panics(t, func() { channelName(&subscription.Subscription{Channel: wsOrderbookChannel}, btcusdPair) })
}

func TestIsWildcardChannel(t *testing.T) {
	t.Parallel()
	assert.False(t, isWildcardChannel(&subscription.Subscription{Channel: subscription.CandlesChannel}))
	assert.True(t, isWildcardChannel(&subscription.Subscription{Channel: subscription.MyOrdersChannel}))
	assert.True(t, isWildcardChannel(&subscription.Subscription{Channel: subscription.MyAccountChannel}))
	assert.True(t, isWildcardChannel(&subscription.Subscription{Channel: wsPositionsChannel, Authenticated: true}))
	assert.Panics(t, func() { channelName(&subscription.Subscription{Channel: wsOrderbookChannel}) })
}

func TestGetErrResp(t *testing.T) {
	err := getErrResp([]byte(`{"status":"error","err-code":"bad-request","err-msg":"invalid topic promiscuous.drop🐻s.nearby"}`))
	assert.ErrorContains(t, err, "invalid topic promiscuous.drop🐻s.nearby (bad-request)", "V1 errors should return correctly")
	err = getErrResp([]byte(`{"status":"ok","subbed":"market.btcusdt.trade.detail"}`))
	assert.NoError(t, err, "V1 success should not error")

	err = getErrResp([]byte(`{"action":"sub","code":2001,"ch":"naughty.drop🐻s.locally","message":"invalid.ch"}`))
	assert.ErrorContains(t, err, "invalid.ch (2001)", "V2 errors should return correctly")

	err = getErrResp([]byte(`{"action":"sub","code":200,"ch":"orders#btcusdt","data":{}}`))
	assert.NoError(t, err, "V2 success should not error")

	err = getErrResp([]byte(`{"op":"auth","err-code":0}`))
	assert.NoError(t, err, "derivative websocket success should not error")
	err = getErrResp([]byte(`{"op":"auth","err-code":2001,"err-msg":"invalid authentication"}`))
	assert.ErrorContains(t, err, "invalid authentication (2001)", "derivative websocket errors should include their numeric code")
}
