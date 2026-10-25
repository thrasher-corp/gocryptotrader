package cryptodotcom

import (
	"bytes"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// accountFile holds the user channels' documented push examples by channel, laid out as testdata/wsPublic.json is and
// made consistent with each other and with the documented REST examples of the same schemas; the account risk and
// position balance channels, which have no documented examples, are composed from their schemas and those values
const accountFile = "testdata/wsAccount.json"

// accountFrames returns the frames testdata/wsAccount.json holds, loaded once
var accountFrames = recordedFrames(accountFile)

// documentedAccountID is the account the documented examples name
const documentedAccountID = "52e7c00f-1324-5a6z-bfgt-de445bde21a5"

// accountFrame returns the frame at index i of the frames answering a user channel's first subscription
func accountFrame(t *testing.T, channel string, i int) []byte {
	t.Helper()
	return recordedFrame(t, accountFrames, channel, i)
}

// enableFillsFeed has an exchange relay the fills its websocket receives through its fills feed
func enableFillsFeed(ex *Exchange) {
	ex.SetFillsFeedStatus(true)
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
}

// documentedOrderDetails returns the order details of the documented perpetual swap order and the margin order
// testdata/wsAccount.json holds, as an exchange relays them
func documentedOrderDetails(exchangeName string) (perpetual, marginOrder order.Detail) {
	perpetual = order.Detail{
		Exchange:           exchangeName,
		OrderID:            "19848525",
		ClientOrderID:      "1613571154900",
		AccountID:          documentedAccountID,
		Pair:               perpBTCUSD,
		AssetType:          asset.PerpetualSwap,
		Side:               order.Buy,
		Type:               order.Limit,
		Status:             order.Active,
		TimeInForce:        order.GoodTillCancel,
		Price:              50000,
		Amount:             0.01,
		RemainingAmount:    0.01,
		CostAsset:          currency.USD,
		FeeAsset:           currency.USD,
		SettlementCurrency: currency.USD,
		MarginType:         margin.Multi,
		Date:               time.Unix(0, 1613575617173123456),
		LastUpdated:        time.UnixMilli(1613575617173),
	}
	marginOrder = order.Detail{
		Exchange:             exchangeName,
		OrderID:              "19848530",
		ClientOrderID:        "1613571154910",
		AccountID:            documentedAccountID,
		Pair:                 spotBTCUSD,
		AssetType:            asset.Margin,
		Side:                 order.Sell,
		Type:                 order.Market,
		Status:               order.Filled,
		TimeInForce:          order.ImmediateOrCancel,
		Amount:               0.005,
		AverageExecutedPrice: 51200,
		ExecutedAmount:       0.005,
		Cost:                 256,
		CostAsset:            currency.USD,
		Fee:                  0.1024,
		FeeAsset:             currency.USD,
		MarginType:           margin.Multi,
		Date:                 time.Unix(0, 1613575617273123456),
		LastUpdated:          time.UnixMilli(1613575617280),
		CloseTime:            time.UnixMilli(1613575617280),
	}
	return perpetual, marginOrder
}

// documentedAdvancedOrderLegs returns the order details of the legs of the advanced order testdata/wsAccount.json holds,
// as an exchange relays them: the documented perpetual swap order and the stop loss and take profit orders closing it
func documentedAdvancedOrderLegs(exchangeName string) []order.Detail {
	entry, _ := documentedOrderDetails(exchangeName)
	stopLoss := entry
	stopLoss.OrderID, stopLoss.ClientOrderID = "19848526", "1613571154901"
	stopLoss.Side, stopLoss.Type, stopLoss.Price, stopLoss.TriggerPrice = order.Sell, order.Stop, 0, 45000
	takeProfit := stopLoss
	takeProfit.OrderID, takeProfit.ClientOrderID = "19848527", "1613571154902"
	takeProfit.Type, takeProfit.TriggerPrice = order.TakeProfit, 55000
	return []order.Detail{entry, stopLoss, takeProfit}
}

func TestUserChannel(t *testing.T) {
	t.Parallel()
	for channel, exp := range map[string]string{
		subscription.MyOrdersChannel:  "user.order",
		subscription.MyTradesChannel:  "user.trade",
		subscription.MyAccountChannel: "user.balance",
		AdvancedOrdersChannel:         "user.advance.order",
		TransactionsChannel:           "user.transactions",
		PositionsChannel:              "user.positions",
		AccountRiskChannel:            "user.account_risk",
		PositionBalanceChannel:        "user.position_balance",
		OTCQuoteRequestsChannel:       "user.otc_qr.requests",
		OTCQuotesChannel:              "user.otc_qr.quotes",
		OTCDealsChannel:               "user.otc.deals",
	} {
		got, err := userChannel(&subscription.Subscription{Channel: channel})
		require.NoErrorf(t, err, "userChannel must not error for %s", channel)
		assert.Equalf(t, exp, got, "userChannel should name the %s channel as Crypto.com does", channel)
	}
	for _, channel := range []string{"user.order", "user.trade", "user.balance"} {
		_, err := userChannel(&subscription.Subscription{Channel: channel})
		assert.ErrorIsf(t, err, subscription.ErrUseConstChannelName, "userChannel should refuse Crypto.com's name for %s", channel)
	}
	for _, channel := range []string{"nope", subscription.TickerChannel, "user.advanced.order"} {
		_, err := userChannel(&subscription.Subscription{Channel: channel})
		assert.ErrorIsf(t, err, subscription.ErrNotSupported, "userChannel should refuse the %s channel", channel)
	}
}

func TestUserChannelByInstrument(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel string
		a       asset.Item
		exp     bool
	}{
		{subscription.MyOrdersChannel, asset.Spot, true},
		{subscription.MyOrdersChannel, asset.Empty, false},
		{AdvancedOrdersChannel, asset.PerpetualSwap, true},
		{subscription.MyTradesChannel, asset.Futures, true},
		{subscription.MyAccountChannel, asset.Spot, false},
		{OTCDealsChannel, asset.Empty, false},
	} {
		got, err := userChannelByInstrument(&subscription.Subscription{Channel: tc.channel}, tc.a)
		require.NoErrorf(t, err, "userChannelByInstrument must not error for %s of %q", tc.channel, tc.a)
		assert.Equalf(t, tc.exp, got, "userChannelByInstrument should report whether %s of %q names a channel of each instrument", tc.channel, tc.a)
	}
	_, err := userChannelByInstrument(&subscription.Subscription{Channel: "nope"}, asset.Spot)
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "userChannelByInstrument should refuse a channel Crypto.com does not serve")
}

func TestUserInstrumentChannel(t *testing.T) {
	t.Parallel()
	got, err := e.userInstrumentChannel(&subscription.Subscription{Channel: subscription.MyTradesChannel}, asset.Futures, futureBTCUSD)
	require.NoError(t, err, "userInstrumentChannel must not error")
	assert.Equal(t, "user.trade.BTCUSD-261030", got, "userInstrumentChannel should name the channel of the instrument")
	got, err = e.userInstrumentChannel(&subscription.Subscription{Channel: subscription.MyOrdersChannel}, asset.Spot, currency.NewBTCUSD())
	require.NoError(t, err, "userInstrumentChannel must not error for a pair not in Crypto.com's format")
	assert.Equal(t, "user.order.BTC_USD", got, "userInstrumentChannel should name the instrument as Crypto.com does")
	_, err = e.userInstrumentChannel(&subscription.Subscription{Channel: "nope"}, asset.Spot, spotBTCUSD)
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "userInstrumentChannel should refuse a channel Crypto.com does not serve")
	_, err = e.userInstrumentChannel(&subscription.Subscription{Channel: subscription.MyOrdersChannel}, asset.Options, spotBTCUSD)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "userInstrumentChannel should refuse an asset Crypto.com does not trade")
}

func TestGenerateUserSubscriptions(t *testing.T) {
	t.Parallel()
	user := func(channel string, a asset.Item, pairs ...currency.Pair) *subscription.Subscription {
		return &subscription.Subscription{Enabled: true, Asset: a, Channel: channel, Pairs: pairs, Authenticated: true}
	}
	qualified := func(s *subscription.Subscription, a asset.Item, channel string, pairs ...currency.Pair) *subscription.Subscription {
		expanded := s.Clone()
		expanded.Asset, expanded.Pairs, expanded.QualifiedChannel = a, pairs, channel
		return expanded
	}
	defaults := make(subscription.List, 0, len(defaultSubscriptions))
	for _, s := range defaultSubscriptions {
		if s.Authenticated {
			defaults = append(defaults, s)
		}
	}
	require.Len(t, defaults, 5, "the default user subscriptions must be the orders, advanced orders, trades, balances and positions")
	orders, trades := user(subscription.MyOrdersChannel, asset.Spot), user(subscription.MyTradesChannel, asset.PerpetualSwap)
	everyOrder := user(subscription.MyOrdersChannel, asset.All)
	everyBalance := user(subscription.MyAccountChannel, asset.All)
	advanced, advancedOfPair := user(AdvancedOrdersChannel, asset.Empty), user(AdvancedOrdersChannel, asset.PerpetualSwap, perpBTCUSD)
	ticker := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}}
	accountChannels := subscription.List{
		user(TransactionsChannel, asset.Empty), user(PositionsChannel, asset.Empty), user(AccountRiskChannel, asset.Empty),
		user(PositionBalanceChannel, asset.Empty), user(OTCQuoteRequestsChannel, asset.Empty), user(OTCQuotesChannel, asset.Empty),
		user(OTCDealsChannel, asset.Empty),
	}
	for _, tc := range []struct {
		name string
		subs subscription.List
		exp  subscription.List
	}{
		{
			name: "the default user channels, each for every instrument",
			subs: defaultSubscriptions,
			exp: subscription.List{
				qualified(defaults[0], asset.Empty, "user.order"),
				qualified(defaults[1], asset.Empty, "user.advance.order"),
				qualified(defaults[2], asset.Empty, "user.trade"),
				qualified(defaults[3], asset.Empty, "user.balance"),
				qualified(defaults[4], asset.Empty, "user.positions"),
			},
		},
		{
			name: "orders and trades of each pair of an asset",
			subs: subscription.List{orders, trades},
			exp: subscription.List{
				qualified(orders, asset.Spot, "user.order.BTC_USD", spotBTCUSD),
				qualified(orders, asset.Spot, "user.order.BTC_USDT", spotBTCUSDT),
				qualified(orders, asset.Spot, "user.order.CRO_USD", spotCROUSD),
				qualified(orders, asset.Spot, "user.order.ETH_USD", spotETHUSD),
				qualified(trades, asset.PerpetualSwap, "user.trade.BTCUSD-PERP", perpBTCUSD),
				qualified(trades, asset.PerpetualSwap, "user.trade.ETHUSD-PERP", perpETHUSD),
			},
		},
		{
			name: "the orders of an instrument spot and margin share once",
			subs: subscription.List{everyOrder},
			exp: subscription.List{
				qualified(everyOrder, asset.Spot, "user.order.BTC_USD", spotBTCUSD),
				qualified(everyOrder, asset.Spot, "user.order.BTC_USDT", spotBTCUSDT),
				qualified(everyOrder, asset.Spot, "user.order.CRO_USD", spotCROUSD),
				qualified(everyOrder, asset.Spot, "user.order.ETH_USD", spotETHUSD),
				qualified(everyOrder, asset.Futures, "user.order.BTCUSD-261030", futureBTCUSD),
				qualified(everyOrder, asset.PerpetualSwap, "user.order.BTCUSD-PERP", perpBTCUSD),
				qualified(everyOrder, asset.PerpetualSwap, "user.order.ETHUSD-PERP", perpETHUSD),
			},
		},
		{
			name: "the balances once, whatever their subscription's asset",
			subs: subscription.List{everyBalance},
			exp:  subscription.List{qualified(everyBalance, asset.Spot, "user.balance", spotBTCUSD, spotBTCUSDT, spotCROUSD, spotETHUSD)},
		},
		{
			name: "the advanced orders of every instrument and of a pair",
			subs: subscription.List{advanced, advancedOfPair},
			exp: subscription.List{
				qualified(advanced, asset.Empty, "user.advance.order"),
				qualified(advancedOfPair, asset.PerpetualSwap, "user.advance.order.BTCUSD-PERP", perpBTCUSD),
			},
		},
		{
			name: "every other user channel",
			subs: accountChannels,
			exp: subscription.List{
				qualified(accountChannels[0], asset.Empty, "user.transactions"),
				qualified(accountChannels[1], asset.Empty, "user.positions"),
				qualified(accountChannels[2], asset.Empty, "user.account_risk"),
				qualified(accountChannels[3], asset.Empty, "user.position_balance"),
				qualified(accountChannels[4], asset.Empty, "user.otc_qr.requests"),
				qualified(accountChannels[5], asset.Empty, "user.otc_qr.quotes"),
				qualified(accountChannels[6], asset.Empty, "user.otc.deals"),
			},
		},
		{
			name: "user channels alongside market data, which the market data connections carry",
			subs: subscription.List{ticker, advanced},
			exp:  subscription.List{qualified(advanced, asset.Empty, "user.advance.order")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			ex.Features.Subscriptions = tc.subs.Clone()
			got, err := ex.generateUserSubscriptions()
			require.NoError(t, err, "generateUserSubscriptions must not error")
			testsubs.EqualLists(t, tc.exp, got)
		})
	}

	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		err  error
	}{
		{"Crypto.com's name of a channel GoCryptoTrader names", user("user.order", asset.Empty), subscription.ErrUseConstChannelName},
		{"a channel Crypto.com does not serve", user("nope", asset.Empty), subscription.ErrNotSupported},
		{"a market data channel", user(subscription.TickerChannel, asset.Spot), subscription.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			ex.Features.Subscriptions = subscription.List{tc.sub}
			_, err := ex.generateUserSubscriptions()
			assert.ErrorIs(t, err, tc.err, "generateUserSubscriptions should refuse the subscription")
		})
	}

	t.Run("without authentication", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
		ex.Features.Subscriptions = defaultSubscriptions.Clone()
		got, err := ex.generateUserSubscriptions()
		require.NoError(t, err, "generateUserSubscriptions must not error without authentication")
		assert.Empty(t, got, "generateUserSubscriptions should subscribe no user channels without authentication")
	})

	t.Run("a user channel left unauthenticated", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: subscription.MyOrdersChannel}}
		_, err := ex.generateMarketDataSubscriptions()
		assert.ErrorIs(t, err, subscription.ErrNotSupported, "generateMarketDataSubscriptions should refuse a user channel")
	})
}

func TestWebsocketUserOrders(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.order", 0)), "wsHandleChannelData must not error for the open orders")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.order", 1)), "wsHandleChannelData must not error for an order")
	perpetual, marginOrder := documentedOrderDetails(ex.Name)
	exp := []any{[]order.Detail{perpetual}, []order.Detail{marginOrder}}
	assert.Equal(t, exp, relayedData(ex), "the orders should be relayed as the details of the assets they trade as")

	// An order whose detail cannot be told is reported, and the others relayed
	push := `{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.order","channel":"user.order","data":[{"order_id":"1","order_type":"NOPE","side":"BUY","status":"ACTIVE","instrument_name":"BTC_USD"},{"order_id":"2","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"BTC"},{"order_id":"3","order_type":"LIMIT","side":"SELL","time_in_force":"GOOD_TILL_CANCEL","exec_inst":[],"quantity":"0.0100","limit_price":"60000.0","status":"ACTIVE","instrument_name":"BTC_USD","fee_instrument_name":"USD","create_time":1613575617173,"update_time":1613575617173}]}}`
	err := ex.wsHandleChannelData(t.Context(), nil, []byte(push))
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "wsHandleChannelData should report an order of a type it does not know")
	assert.ErrorIs(t, err, errMalformedInstrumentName, "wsHandleChannelData should report an order of an instrument it cannot name")
	exp = []any{[]order.Detail{{
		Exchange:        ex.Name,
		OrderID:         "3",
		Pair:            spotBTCUSD,
		AssetType:       asset.Spot,
		Side:            order.Sell,
		Type:            order.Limit,
		Status:          order.Active,
		TimeInForce:     order.GoodTillCancel,
		Price:           60000,
		Amount:          0.01,
		RemainingAmount: 0.01,
		CostAsset:       currency.USD,
		FeeAsset:        currency.USD,
		Date:            time.UnixMilli(1613575617173),
		LastUpdated:     time.UnixMilli(1613575617173),
	}}}
	assert.Equal(t, exp, relayedData(ex), "the orders whose details can be told should be relayed")

	push = `{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.order","channel":"user.order","data":[{"order_id":"1","order_type":"NOPE","side":"BUY","status":"ACTIVE","instrument_name":"BTC_USD"}]}}`
	assert.ErrorIs(t, ex.wsHandleChannelData(t.Context(), nil, []byte(push)), order.ErrUnrecognisedOrderType, "wsHandleChannelData should report orders none of whose details can be told")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(`{"id":1,"method":"subscribe","code":0,"result":{"subscription":"user.order","channel":"user.order","data":[]}}`)), "wsHandleChannelData must not error without open orders")
	assert.ErrorIs(t, ex.wsHandleChannelData(t.Context(), nil, []byte(`{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.order","channel":"user.order","data":{}}}`)), errMalformedWebsocketMessage, "wsHandleChannelData should refuse orders it cannot decode")
	assert.Empty(t, relayedData(ex), "nothing should be relayed without an order whose detail can be told")
}

func TestWebsocketAdvancedOrders(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	frame := accountFrame(t, "user.advance.order", 0)
	legs := documentedAdvancedOrderLegs(ex.Name)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, frame), "wsHandleChannelData must not error")
	assert.Equal(t, []any{legs}, relayedData(ex), "the legs of the advanced order should be relayed as order details")

	// Some of Crypto.com's docs name the channel user.advanced.order
	advanced := bytes.ReplaceAll(frame, []byte(`"channel":"user.advance.order"`), []byte(`"channel":"user.advanced.order"`))
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, advanced), "wsHandleChannelData must not error for a push of the user.advanced.order channel")
	assert.Equal(t, []any{legs}, relayedData(ex), "a push of the user.advanced.order channel should be relayed as an advanced order's")
}

func TestWebsocketUserTrades(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.trade", 0)), "wsHandleChannelData must not error without a fills feed")
	exp := []any{[]WebsocketUserTrade{{
		AccountID:         documentedAccountID,
		EventDate:         "2021-02-17",
		JournalType:       JournalTypeTrading,
		TradedQuantity:    preciseNumber("0.0500"),
		TradedPrice:       preciseNumber("51278.5"),
		Fees:              preciseNumber("-1.025570"),
		FeeCredits:        preciseNumber("-0.500000"),
		OrderID:           "19708564",
		TradeID:           "38554669",
		TradeMatchID:      "76423",
		ClientOrderID:     "6ac2421d-5078-4ef6-a9d5-9680602ce123",
		TakerSide:         TakerSideMaker,
		Side:              SideBuy,
		InstrumentName:    "BTCUSD-PERP",
		FeeInstrumentName: currency.USD,
		CreateTime:        types.Time(time.UnixMilli(1613570791060)),
		CreateTimeNano:    types.Time(time.Unix(0, 1613570791060123456)),
		TransactionTime:   types.Time(time.Unix(0, 1613570791060827635)),
	}}}
	assert.Equal(t, exp, relayedData(ex), "the trades should be relayed as Crypto.com reports them without a fills feed")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(`{"id":1,"method":"subscribe","code":0,"result":{"subscription":"user.trade","channel":"user.trade","data":[]}}`)), "wsHandleChannelData must not error without trades")
	assert.Empty(t, relayedData(ex), "nothing should be relayed without trades")

	enableFillsFeed(ex)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.trade", 0)), "wsHandleChannelData must not error")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.trade", 1)), "wsHandleChannelData must not error for a margin order's trade")
	marginOrderFill := fill.Data{
		Timestamp:     time.Unix(0, 1613575617280123456),
		Exchange:      ex.Name,
		AssetType:     asset.Spot,
		CurrencyPair:  spotBTCUSD,
		Side:          order.Sell,
		OrderID:       "19848530",
		ClientOrderID: "1613571154910",
		TradeID:       "38554670",
		Price:         51200,
		Amount:        0.005,
	}
	exp = []any{
		[]fill.Data{{
			Timestamp:     time.Unix(0, 1613570791060123456),
			Exchange:      ex.Name,
			AssetType:     asset.PerpetualSwap,
			CurrencyPair:  perpBTCUSD,
			Side:          order.Buy,
			OrderID:       "19708564",
			ClientOrderID: "6ac2421d-5078-4ef6-a9d5-9680602ce123",
			TradeID:       "38554669",
			Price:         51278.5,
			Amount:        0.05,
		}},
		// A trade carries no execution instructions, so the trade of a margin order not yet tracked is filled as spot
		[]fill.Data{marginOrderFill},
	}
	assert.Equal(t, exp, relayedData(ex), "the trades should be relayed as fills through the fills feed")

	// The order channel reports the order as a margin order, whose trades are then filled as margin
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.order", 1)), "wsHandleChannelData must not error for the margin order")
	require.Len(t, relayedData(ex), 1, "the margin order's details must be relayed")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.trade", 1)), "wsHandleChannelData must not error for a margin order's trade")
	marginOrderFill.AssetType = asset.Margin
	assert.Equal(t, []any{[]fill.Data{marginOrderFill}}, relayedData(ex), "the trade of an order tracked as a margin order should be filled as margin")

	// A trade whose fill cannot be told is reported, and the others filled
	push := `{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.trade","channel":"user.trade","data":[{"trade_id":"1","side":"NOPE","instrument_name":"BTC_USD"},{"trade_id":"2","side":"BUY","instrument_name":"BTCUSD"},{"trade_id":"3","traded_quantity":"0.0010","traded_price":"51000.00","side":"BUY","instrument_name":"BTC_USD","create_time":1613575617290}]}}`
	err := ex.wsHandleChannelData(t.Context(), nil, []byte(push))
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "wsHandleChannelData should report a trade of a side it does not know")
	assert.ErrorIs(t, err, errMalformedInstrumentName, "wsHandleChannelData should report a trade of an instrument it cannot name")
	exp = []any{[]fill.Data{{
		Timestamp:    time.UnixMilli(1613575617290),
		Exchange:     ex.Name,
		AssetType:    asset.Spot,
		CurrencyPair: spotBTCUSD,
		Side:         order.Buy,
		TradeID:      "3",
		Price:        51000,
		Amount:       0.001,
	}}}
	assert.Equal(t, exp, relayedData(ex), "the trades whose fills can be told should be filled, timed to the millisecond without a nanosecond time")

	for _, frame := range []string{
		`{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.trade","channel":"user.trade","data":[]}}`,
		`{"id":1,"method":"subscribe","code":0,"channel":"user.trade"}`,
	} {
		require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(frame)), "wsHandleChannelData must not error for %s", frame)
	}
	assert.Empty(t, relayedData(ex), "nothing should be relayed without trades")
	assert.ErrorIs(t, ex.wsHandleChannelData(t.Context(), nil, []byte(`{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.trade","channel":"user.trade","data":{}}}`)), errMalformedWebsocketMessage, "wsHandleChannelData should refuse trades it cannot decode")
}

func TestTrackMarginOrder(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.trackMarginOrder(asset.Margin, "1", "c1")
	assert.True(t, ex.isMarginOrder("1", ""), "isMarginOrder should report an order tracked as a margin order by its order ID")
	assert.True(t, ex.isMarginOrder("", "c1"), "isMarginOrder should report an order tracked as a margin order by its client order ID")
	assert.True(t, ex.isMarginOrder("2", "c1"), "isMarginOrder should report an order whose client order ID is tracked")
	assert.False(t, ex.isMarginOrder("c1", "1"), "isMarginOrder should keep order IDs and client order IDs apart")
	assert.False(t, ex.isMarginOrder("", ""), "isMarginOrder should not report an order without IDs")

	ex.trackMarginOrder(asset.PerpetualSwap, "3", "c3")
	ex.trackMarginOrder(asset.Futures, "1", "c1")
	assert.False(t, ex.isMarginOrder("3", "c3"), "trackMarginOrder should not track a derivative's order")
	assert.True(t, ex.isMarginOrder("1", "c1"), "trackMarginOrder should leave a margin order tracked for a derivative's order")

	ex.trackMarginOrder(asset.Spot, "1", "")
	assert.False(t, ex.isMarginOrder("1", ""), "trackMarginOrder should track an order trading as spot as no margin order by its order ID")
	assert.True(t, ex.isMarginOrder("", "c1"), "trackMarginOrder should go on tracking a client order ID it is not given")
	ex.trackMarginOrder(asset.Spot, "", "c1")
	assert.False(t, ex.isMarginOrder("", "c1"), "trackMarginOrder should stop tracking an order trading as spot by its client order ID")

	// Orders can share a client order ID, so an order known to trade as spot by its order ID is not taken for the margin
	// order that shares it, though an order not known by its order ID is
	ex.trackMarginOrder(asset.Spot, "101", "shared")
	ex.trackMarginOrder(asset.Margin, "102", "shared")
	assert.False(t, ex.isMarginOrder("101", "shared"), "isMarginOrder should tell an order known by its order ID by that alone")
	assert.True(t, ex.isMarginOrder("103", "shared"), "isMarginOrder should tell an order not known by its order ID by its client order ID")

	ex.trackMarginOrder(asset.Margin, "", "c4")
	ex.untrackClientOrderID("c4")
	assert.False(t, ex.isMarginOrder("", "c4"), "untrackClientOrderID should stop tracking a client order ID")

	var untracked Exchange
	untracked.untrackClientOrderID("c1")
	untracked.trackMarginOrder(asset.Margin, "1", "c1")
	assert.False(t, untracked.isMarginOrder("1", "c1"), "isMarginOrder should report no order without a margin order cache")
}

func TestWebsocketTransactions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.transactions", 0)), "wsHandleChannelData must not error for the acknowledgement")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.transactions", 1)), "wsHandleChannelData must not error")
	exp := []any{[]TransactionResponse{
		{
			AccountID:           documentedAccountID,
			EventDate:           "2021-02-18",
			JournalType:         JournalTypeTrading,
			JournalID:           "187078",
			TransactionQuantity: preciseNumber("-0.0005"),
			TransactionCost:     preciseNumber("-24.500000"),
			RealisedPNL:         preciseNumber("-0.006125"),
			OrderID:             "72062",
			TradeID:             "71497",
			TradeMatchID:        "8625",
			EventTime:           types.Time(time.UnixMilli(1613640752166)),
			EventTimeNano:       types.Time(time.Unix(0, 1613640752166234567)),
			ClientOrderID:       "6ac2421d-5078-4ef6-a9d5-9680602ce123",
			TakerSide:           TakerSideMaker,
			Side:                SideSell,
			InstrumentName:      "BTCUSD-PERP",
		},
		{
			AccountID:           documentedAccountID,
			EventDate:           "2021-02-18",
			JournalType:         JournalTypeSessionSettle,
			JournalID:           "186959",
			TransactionQuantity: preciseNumber("0"),
			TransactionCost:     preciseNumber("0.000000"),
			RealisedPNL:         preciseNumber("-0.007800"),
			TradeMatchID:        "0",
			EventTime:           types.Time(time.UnixMilli(1613638800002)),
			EventTimeNano:       types.Time(time.Unix(0, 1613638800002124563)),
			InstrumentName:      "BTCUSD-PERP",
			IsolationID:         "19848526",
			IsolationType:       IsolationTypeIsolatedMargin,
		},
	}}
	assert.Equal(t, exp, relayedData(ex), "the transactions should be relayed")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(`{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.transactions","channel":"user.transactions","data":[]}}`)), "wsHandleChannelData must not error without transactions")
	assert.Empty(t, relayedData(ex), "nothing should be relayed without transactions")
}

func TestWebsocketUserBalance(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	setTestCredentials(ex, &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")

	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.balance", 0)), "wsHandleChannelData must not error")
	relayed := relayedData(ex)
	require.Len(t, relayed, 1, "the balances must be relayed once")
	subAccts, ok := relayed[0].(accounts.SubAccounts)
	require.True(t, ok, "the balances must be relayed as sub accounts")
	for _, subAcct := range subAccts {
		for c, b := range subAcct.Balances {
			// The push leaves the update time out, so the account store stamps it
			assert.WithinDurationf(t, time.Now(), b.UpdatedAt, time.Minute, "the %s %s balance should be stamped when it was saved", subAcct.AssetType, c)
			b.UpdatedAt = time.Time{}
			subAcct.Balances[c] = b
		}
	}
	exp := make(accounts.SubAccounts, 0, 4)
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures, asset.PerpetualSwap} {
		exp = append(exp, &accounts.SubAccount{AssetType: a, Balances: accounts.CurrencyBalances{
			currency.CRO: {Currency: currency.CRO, Total: 24422.72427884, Free: 24422.72427884},
			currency.USD: {Currency: currency.USD, Total: 3113.50747209, Free: 3113.50747209},
		}})
	}
	assert.Equal(t, exp, subAccts, "the balances should be relayed for each enabled asset, as every asset trades from the one wallet")

	// Crypto.com's docs leave out whether a push holds every currency, so a push leaving one out keeps it
	usdOnly := `{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.balance","channel":"user.balance","data":[{"instrument_name":"USD","position_balances":[{"instrument_name":"USD","quantity":"3113.50747209","reserved_qty":"0.00000000"}]}]}}`
	for _, tc := range []struct {
		name  string
		frame []byte
		usd   accounts.Balance
	}{
		{"a change", accountFrame(t, "user.balance", 1), accounts.Balance{Currency: currency.USD, Total: 3113.50747209, Hold: 100, Free: 3013.50747209}},
		{"a change leaving a currency out", []byte(usdOnly), accounts.Balance{Currency: currency.USD, Total: 3113.50747209, Free: 3113.50747209}},
	} {
		require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, tc.frame), "wsHandleChannelData must not error for %s", tc.name)
		assert.Lenf(t, relayedData(ex), 1, "%s should be relayed once", tc.name)
		for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures, asset.PerpetualSwap} {
			stored, err := ex.Accounts.CurrencyBalances(creds, a)
			require.NoErrorf(t, err, "the %s balances must be stored", a)
			for c, b := range stored {
				b.UpdatedAt = time.Time{}
				stored[c] = b
			}
			exp := accounts.CurrencyBalances{
				currency.CRO: {Currency: currency.CRO, Total: 24422.72427884, Free: 24422.72427884},
				currency.USD: tc.usd,
			}
			assert.Equalf(t, exp, stored, "the %s balances should take %s, keeping any currency it leaves out", a, tc.name)
		}
	}

	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(`{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.balance","channel":"user.balance","data":[]}}`)), "wsHandleChannelData must not error without balances")
	assert.ErrorIs(t, ex.wsHandleChannelData(t.Context(), nil, []byte(`{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.balance","channel":"user.balance","data":{}}}`)), errMalformedWebsocketMessage, "wsHandleChannelData should refuse balances it cannot decode")
	assert.Empty(t, relayedData(ex), "nothing should be relayed without balances, or for balances that cannot be decoded")

	unauthenticated := newTestExchange(t)
	setTestCredentials(unauthenticated, &accounts.Credentials{})
	// Checking the credentials, which the mock skips, refuses empty ones with an error the account store's check does not
	// export
	unauthenticated.SkipAuthCheck = false
	assert.ErrorIs(t, unauthenticated.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.balance", 0)), exchange.ErrCredentialsAreEmpty, "wsHandleChannelData should report balances it cannot store")
	assert.Empty(t, relayedData(unauthenticated), "balances that cannot be stored should not be relayed")
}

// documentedPositions returns the documented positions testdata/wsAccount.json holds, as the user.positions channel
// reports them
func documentedPositions() (cross, isolated PositionResponse) {
	cross = PositionResponse{
		AccountID:                 documentedAccountID,
		InstrumentName:            "BTCUSD-PERP",
		InstrumentType:            InstrumentTypePerpetualSwap,
		Quantity:                  preciseNumber("-0.1984"),
		Cost:                      preciseNumber("-10159.573500"),
		OpenPositionPNL:           preciseNumber("-497.743736"),
		OpenPositionCost:          preciseNumber("-10159.352200"),
		SessionPNL:                preciseNumber("2.236145"),
		UpdateTime:                types.Time(time.UnixMilli(1613552240770)),
		SessionUnrealisedPNL:      preciseNumber("-497.743736"),
		PositionInitialMargin:     preciseNumber("1065.731724"),
		PositionMaintenanceMargin: preciseNumber("532.865862"),
		MarketValue:               preciseNumber("-10657.315904"),
		MarkPrice:                 53716.31,
		TargetLeverage:            10,
		Leverage:                  10,
		LiquidationPrice:          88765.80,
	}
	isolated = cross
	isolated.UpdateTime = types.Time(time.UnixMilli(1613552240771))
	isolated.IsolationID = "19848526"
	isolated.IsolationType = IsolationTypeIsolatedMargin
	isolated.LiquidationPrice = 58700.00
	isolated.IsolatedMarginBalance = preciseNumber("1100.00000000")
	return cross, isolated
}

// documentedPositionCores returns the documented positions' core fields, which the user.position_balance channel
// reports alone
func documentedPositionCores() (cross, isolated PositionResponse) {
	cross = PositionResponse{
		AccountID:        documentedAccountID,
		InstrumentName:   "BTCUSD-PERP",
		InstrumentType:   InstrumentTypePerpetualSwap,
		Quantity:         preciseNumber("-0.1984"),
		Cost:             preciseNumber("-10159.573500"),
		OpenPositionPNL:  preciseNumber("-497.743736"),
		OpenPositionCost: preciseNumber("-10159.352200"),
		SessionPNL:       preciseNumber("2.236145"),
		UpdateTime:       types.Time(time.UnixMilli(1613552240770)),
	}
	isolated = cross
	isolated.UpdateTime = types.Time(time.UnixMilli(1613552240771))
	isolated.IsolationID = "19848526"
	isolated.IsolationType = IsolationTypeIsolatedMargin
	return cross, isolated
}

func TestWebsocketPositions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.positions", 0)), "wsHandleChannelData must not error")
	cross, isolated := documentedPositions()
	exp := []any{[]PositionResponse{cross, isolated}}
	assert.Equal(t, exp, relayedData(ex), "the positions should be relayed")
}

func TestWebsocketAccountRisk(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.account_risk", 0)), "wsHandleChannelData must not error for the acknowledgement")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.account_risk", 1)), "wsHandleChannelData must not error")
	cross, isolated := documentedPositions()
	exp := []any{[]WebsocketAccountRisk{{
		AccountID:                  documentedAccountID,
		InstrumentName:             "USD",
		TotalAvailableBalance:      preciseNumber("4721.05898582"),
		TotalMarginBalance:         preciseNumber("7595.42571782"),
		TotalInitialMargin:         preciseNumber("2874.36673202"),
		TotalMaintenanceMargin:     preciseNumber("1437.18336601"),
		TotalPositionCost:          preciseNumber("14517.54641301"),
		TotalCashBalance:           preciseNumber("7890.00320721"),
		TotalCollateralValue:       preciseNumber("7651.18811483"),
		TotalSessionUnrealisedPNL:  preciseNumber("-55.76239701"),
		TotalSessionRealisedPNL:    preciseNumber("0.00000000"),
		TotalEffectiveLeverage:     1.9040123,
		PositionLimit:              preciseNumber("3000000.00000000"),
		UsedPositionLimit:          preciseNumber("40674.69622001"),
		TotalIsolatedCashBalance:   preciseNumber("1100.00000000"),
		TotalPositionInitialMargin: preciseNumber("486.31273202"),
		TotalHaircut:               preciseNumber("2388.054"),
		TotalRiskExposure:          preciseNumber("21314.63358080"),
		TotalBorrow:                preciseNumber("0"),
		MarginScore:                0.18922,
		CreditLimits:               []json.RawMessage{},
		Balances: []CollateralBalance{
			{
				InstrumentName:       "CRO",
				Quantity:             preciseNumber("24422.72427884"),
				MarketValue:          preciseNumber("4776.107959969951"),
				CollateralEligible:   true,
				Haircut:              0.5,
				CollateralAmount:     preciseNumber("2388.053979984976"),
				MaxWithdrawalBalance: preciseNumber("24422.72427884"),
				ReservedQuantity:     preciseNumber("0.00000000"),
			},
			{
				InstrumentName:       "USD",
				Quantity:             preciseNumber("3113.50747209"),
				MarketValue:          preciseNumber("3113.50747209"),
				CollateralEligible:   true,
				CollateralAmount:     preciseNumber("3113.50747209"),
				MaxWithdrawalBalance: preciseNumber("3113.50747209"),
				ReservedQuantity:     preciseNumber("0.00000000"),
			},
		},
		Positions: []PositionResponse{cross},
		IsolatedPositions: []AccountRiskIsolatedPosition{{
			InstrumentName:             "USD",
			TotalAvailableBalance:      preciseNumber("34.26827600"),
			TotalMarginBalance:         preciseNumber("602.25626400"),
			TotalInitialMargin:         preciseNumber("567.98798800"),
			TotalMaintenanceMargin:     preciseNumber("283.99399400"),
			TotalPositionCost:          preciseNumber("10159.57350000"),
			TotalCashBalance:           preciseNumber("1100.00000000"),
			TotalCollateralValue:       preciseNumber("1100.00000000"),
			TotalSessionUnrealisedPNL:  preciseNumber("-497.74373600"),
			TotalSessionRealisedPNL:    preciseNumber("0.00000000"),
			TotalEffectiveLeverage:     17.695111,
			PositionLimit:              preciseNumber("3000000.00000000"),
			UsedPositionLimit:          preciseNumber("10657.31679040"),
			TotalIsolatedCashBalance:   preciseNumber("0.00000000"),
			TotalPositionInitialMargin: preciseNumber("567.98798800"),
			TotalHaircut:               preciseNumber("0"),
			IsolationID:                "19848526",
			Leverage:                   10,
			IsolationType:              IsolationTypeIsolatedMargin,
			Balances: []CollateralBalance{{
				InstrumentName:       "USD",
				Quantity:             preciseNumber("1100.00000000"),
				MarketValue:          preciseNumber("1100.00000000"),
				CollateralEligible:   true,
				CollateralAmount:     preciseNumber("1100.00000000"),
				MaxWithdrawalBalance: preciseNumber("34.26827600"),
				ReservedQuantity:     preciseNumber("0.00000000"),
			}},
			Positions: []PositionResponse{isolated},
		}},
	}}}
	assert.Equal(t, exp, relayedData(ex), "the account's risk should be relayed with the account the push names")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(`{"id":-1,"method":"subscribe","code":0,"result":{"account_id":"`+documentedAccountID+`","subscription":"user.account_risk","channel":"user.account_risk","data":[]}}`)), "wsHandleChannelData must not error without risk")
	assert.Empty(t, relayedData(ex), "nothing should be relayed without risk")
	assert.ErrorIs(t, ex.wsHandleChannelData(t.Context(), nil, []byte(`{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.account_risk","channel":"user.account_risk","data":{}}}`)), errMalformedWebsocketMessage, "wsHandleChannelData should refuse risk it cannot decode")
}

func TestWebsocketPositionBalance(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.position_balance", 0)), "wsHandleChannelData must not error for the acknowledgement")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.position_balance", 1)), "wsHandleChannelData must not error")
	cross, isolated := documentedPositionCores()
	exp := []any{[]WebsocketPositionBalance{{
		Balances: []CollateralBalance{
			{InstrumentName: "CRO", Quantity: preciseNumber("24422.72427884"), UpdateTime: types.Time(time.UnixMilli(1613552240770))},
			{InstrumentName: "USD", Quantity: preciseNumber("3113.50747209"), UpdateTime: types.Time(time.UnixMilli(1613552240770))},
		},
		Positions: []PositionResponse{cross},
		IsolatedPositions: []PositionBalanceIsolatedPosition{{
			IsolationID:   "19848526",
			Leverage:      10,
			IsolationType: IsolationTypeIsolatedMargin,
			Balances:      []CollateralBalance{{InstrumentName: "USD", Quantity: preciseNumber("1100.00000000"), UpdateTime: types.Time(time.UnixMilli(1613552240771))}},
			Positions:     []PositionResponse{isolated},
		}},
	}}}
	assert.Equal(t, exp, relayedData(ex), "the collateral balances and positions should be relayed")
}

func TestWebsocketOTCQuoteRequests(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for i := range 3 {
		require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.otc_qr.requests", i)), "wsHandleChannelData must not error for frame %d", i)
	}
	exp := []any{
		[]OTCQuoteRequestResponse{{QuoteRequestID: "88997712345", ClientQuoteRequestID: "abcd13456", Status: OTCQuoteRequestStatusActive}},
		[]OTCQuoteRequestResponse{{QuoteRequestID: "88997712345", ClientQuoteRequestID: "abcd13456", Status: OTCQuoteRequestStatusCompleted}},
	}
	assert.Equal(t, exp, relayedData(ex), "the quote requests should be relayed as their statuses change")
}

func TestWebsocketOTCQuotes(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.otc_qr.quotes", 1)), "wsHandleChannelData must not error")
	exp := []any{[]OTCQuoteResponse{{
		QuoteID:              "11223344556677",
		QuoteRequestID:       "88997712345",
		ClientQuoteRequestID: "abcd13456",
		Status:               OTCQuoteStatusActive,
		Reason:               "NO_ERROR",
		ResponseTime:         types.Time(time.Unix(0, 1695153400000000000)),
		ExpiryTime:           types.Time(time.Unix(0, 1695153405000000000)),
		Legs:                 []OTCQuoteLeg{{InstrumentName: "BTC_USD", Quantity: 1, Bid: 95085.99, Ask: 96086.01, Type: OTCLegTypePrice}},
	}}}
	assert.Equal(t, exp, relayedData(ex), "the quote should be relayed")
}

func TestWebsocketOTCDeals(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for i := 1; i < 3; i++ {
		require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, accountFrame(t, "user.otc.deals", i)), "wsHandleChannelData must not error for frame %d", i)
	}
	deal := OTCDealResponse{
		DealID:         "111222333444555777",
		ClientDealID:   "external_deal_id_123",
		QuoteID:        "11223344556677",
		QuoteRequestID: "88997712345",
		DealType:       OTCDealTypeQuoteRequest,
		DealStatus:     OTCDealStatusAccepted,
		Reason:         "NO_ERROR",
		UpdateTime:     types.Time(time.Unix(0, 1695153401000000000)),
		Legs:           []OTCDealLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Price: preciseNumber("96086.01"), Quantity: preciseNumber("1"), Type: OTCLegTypePrice}},
	}
	settled := deal
	settled.DealStatus = OTCDealStatusSettled
	settled.UpdateTime = types.Time(time.Unix(0, 1695153402000000000))
	settled.Legs = []OTCDealLeg{{InstrumentName: "BTC_USD", Side: SideBuy, Price: preciseNumber("96086.01"), Quantity: preciseNumber("1"), ExecutedQuantity: preciseNumber("1"), ExecutedNotional: preciseNumber("96086.01"), Type: OTCLegTypePrice}}
	exp := []any{[]OTCDealResponse{deal}, []OTCDealResponse{settled}}
	assert.Equal(t, exp, relayedData(ex), "the deal should be relayed as its status changes")
}

func TestWebsocketUserChannelsUndecodable(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for _, channel := range []string{TransactionsChannel, PositionsChannel, PositionBalanceChannel, OTCQuoteRequestsChannel, OTCQuotesChannel, OTCDealsChannel} {
		frame := `{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"` + channel + `","channel":"` + channel + `","data":{}}}`
		assert.ErrorIsf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(frame)), errMalformedWebsocketMessage, "wsHandleChannelData should refuse %s data it cannot decode", channel)
	}
	assert.Empty(t, relayedData(ex), "nothing should be relayed for data that cannot be decoded")

	unknown := `{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.nope","channel":"user.nope","data":[]}}`
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(unknown)), "wsHandleChannelData must not error for a user channel it does not know")
	exp := []any{websocket.UnhandledMessageWarning{Message: ex.Name + websocket.UnhandledMessage + unknown}}
	assert.Equal(t, exp, relayedData(ex), "a user channel wsHandleChannelData does not know should be relayed as unhandled")
}

func TestWebsocketUserDataConnect(t *testing.T) {
	t.Parallel()
	venue := newUserDataVenue(t)
	subs := make(subscription.List, 0, len(defaultSubscriptions))
	for _, s := range defaultSubscriptions {
		if s.Authenticated {
			subs = append(subs, s.Clone())
		}
	}
	ex, err := newUserDataTestExchange(t, venue, subs)
	require.NoError(t, err, "Connect must subscribe the user channels")

	stored := ex.Websocket.GetSubscriptions()
	require.Len(t, stored, len(subs), "each user channel must be stored")
	for _, s := range stored {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "%s should be subscribed", s.QualifiedChannel)
	}
	assert.Len(t, venue.received(websocketMethodAuth), 2, "the user and trade connections should authenticate")
	requests := venue.received(websocketMethodSubscribe)
	require.Len(t, requests, 1, "the user channels must be subscribed in one request")
	assert.JSONEq(t, `{"channels":["user.order","user.advance.order","user.trade","user.balance","user.positions"]}`, string(requests[0].Params), "the user channels should be subscribed for every instrument")
	_, err = ex.Websocket.GetConnection(websocketMarketDataConnection)
	assert.Error(t, err, "no market data connection should be dialled for the user channels")

	// The positions are answered last, so every user channel's frames have been handled once they are relayed
	var relayed []any
	require.Eventually(t, func() bool {
		relayed = append(relayed, relayedData(ex)...)
		return slices.ContainsFunc(relayed, func(data any) bool {
			_, ok := data.([]PositionResponse)
			return ok
		})
	}, 10*time.Second, 10*time.Millisecond, "the positions must be relayed")
	var orders [][]order.Detail
	var tradeIDs []string
	for _, data := range relayed {
		switch d := data.(type) {
		case []order.Detail:
			orders = append(orders, d)
		case []WebsocketUserTrade:
			for i := range d {
				tradeIDs = append(tradeIDs, d[i].TradeID)
			}
		case accounts.SubAccounts, []PositionResponse:
		default:
			assert.Failf(t, "only the user channels' data should be relayed", "relayed %T: %v", data, data)
		}
	}
	perpetual, marginOrder := documentedOrderDetails(ex.Name)
	exp := [][]order.Detail{{perpetual}, {marginOrder}, documentedAdvancedOrderLegs(ex.Name)}
	assert.Equal(t, exp, orders, "the open orders each acknowledgement carries and the orders pushed after should be relayed")
	assert.Equal(t, []string{"38554669", "38554670"}, tradeIDs, "the trades the acknowledgement carries and the trade pushed after should be relayed")

	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	balances, err := ex.Accounts.CurrencyBalances(creds, asset.Spot)
	require.NoError(t, err, "the balances must be stored")
	usd := balances[currency.USD]
	usd.UpdatedAt = time.Time{}
	assert.Equal(t, accounts.Balance{Currency: currency.USD, Total: 3113.50747209, Hold: 100, Free: 3013.50747209}, usd, "the balances the acknowledgement carries should be stored, then changed by the push after")
}
