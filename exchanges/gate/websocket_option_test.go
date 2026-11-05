package gate

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestProcessOptionsContractTickers(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		payload string
		wantErr bool
	}{
		{
			name:    "mocked mark and index prices",
			payload: `{"name":"BTC_USDT-20261225-100000-C","last_price":"118.4","mark_price":"118.35","index_price":"100000.25"}`,
		},
		{
			name:    "mocked malformed response",
			payload: `{`,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			err := ex.processOptionsContractTickers(t.Context(), []byte(tc.payload))
			if tc.wantErr {
				require.Error(t, err, "malformed response must fail")
				return
			}
			require.NoError(t, err, "options ticker processing must succeed")
			select {
			case msg := <-ex.Websocket.DataHandler.C:
				got, ok := msg.Data.(*ticker.Price)
				require.True(t, ok, "message must contain a ticker price")
				assert.Equal(t, 118.35, got.MarkPrice, "mark price should match")
				assert.Equal(t, 100000.25, got.IndexPrice, "index price should match")
				assert.Equal(t, asset.Options, got.AssetType, "asset should be options")
				assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT-20261225-100000-C", currency.UnderscoreDelimiter), got.Pair, "contract should match")
			default:
				require.Fail(t, "options ticker must be emitted")
			}
		})
	}
}

func TestProcessResponse(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	var malformed websocketResponse[[]*WebsocketPositionClose]
	require.Error(t, ex.processResponse(t.Context(), []byte(`{`), &malformed), "processResponse must error on malformed JSON")

	var data websocketResponse[[]*WebsocketPositionClose]
	require.NoError(t, ex.processResponse(t.Context(), []byte(`{"time":1541505434,"channel":"options.position_closes","event":"update","result":[{"contract":"BTC_USDT-20211130-65000-C","pnl":-0.0001,"side":"long","text":"web","time_ms":1547198562123,"user":"9xxx","settle_size":"1","underlying":"BTC_USDT"}]}`), &data), "processResponse must not error")
	select {
	case msg := <-ex.Websocket.DataHandler.C:
		exp := &websocketResponse[[]*WebsocketPositionClose]{
			Time:    types.Time(time.Unix(1541505434, 0)),
			Channel: "options.position_closes",
			Event:   "update",
			Result: []*WebsocketPositionClose{{
				Contract:      currency.NewPairWithDelimiter("BTC", "USDT-20211130-65000-C", currency.UnderscoreDelimiter),
				ProfitAndLoss: -0.0001,
				Side:          "long",
				Text:          "web",
				Time:          types.Time(time.UnixMilli(1547198562123)),
				User:          "9xxx",
				SettleSize:    1,
				Underlying:    "BTC_USDT",
			}},
		}
		assert.Equal(t, exp, msg.Data, "processResponse should relay the decoded payload")
	default:
		require.Fail(t, "processResponse must relay the decoded payload")
	}
}

func TestWebsocketHandleOptionsDataRelaysResult(t *testing.T) {
	t.Parallel()
	contract := currency.NewPairWithDelimiter("BTC", "USDT-20211130-50000-P", currency.UnderscoreDelimiter)
	// The envelope's times differ from the result's, so relaying the envelope instead of the result shows
	for _, tc := range []struct {
		channel string
		result  string
		exp     any
	}{
		{
			channel: optionsUnderlyingTickersChannel,
			result:  `{"trade_put":800,"trade_call":41700,"index_price":"50695.43","name":"BTC_USDT"}`,
			exp:     &WebsocketOptionUnderlyingTicker{TradePut: 800, TradeCall: 41700, IndexPrice: 50695.43, Name: "BTC_USDT"},
		},
		{
			channel: optionsUnderlyingPriceChannel,
			result:  `{"underlying":"BTC_USDT","price":49653.24,"time":1639143988,"time_ms":1639143988931}`,
			exp:     &WebsocketOptionsUnderlyingPrice{Underlying: "BTC_USDT", Price: 49653.24, UpdateTime: types.Time(time.UnixMilli(1639143988931))},
		},
		{
			channel: optionsMarkPriceChannel,
			result:  `{"contract":"BTC_USDT-20211130-50000-P","price":11021.27,"time":1639143401,"time_ms":1639143401676}`,
			exp:     &WebsocketOptionsMarkPrice{Contract: contract, Price: 11021.27, UpdateTime: types.Time(time.UnixMilli(1639143401676))},
		},
		{
			channel: optionsSettlementChannel,
			result:  `{"contract":"BTC_USDT-20211130-50000-P","orderbook_id":2,"position_size":1,"profit":0.5,"settle_price":70000,"strike_price":65000,"tag":"WEEK","trade_id":3,"trade_size":4,"underlying":"BTC_USDT","time":1639051907,"time_ms":1639051907000}`,
			exp:     &WebsocketOptionsSettlement{Contract: contract, OrderbookID: 2, PositionSize: 1, Profit: 0.5, SettlePrice: 70000, StrikePrice: 65000, Tag: "WEEK", TradeID: 3, TradeSize: 4, Underlying: "BTC_USDT", UpdateTime: types.Time(time.UnixMilli(1639051907000))},
		},
		{
			channel: optionsContractsChannel,
			result:  `{"contract":"BTC_USDT-20211130-50000-P","create_time":1637917026,"expiration_time":1638230400,"init_margin_high":0.15,"init_margin_low":0.1,"is_call":true,"maint_margin_base":0.075,"maker_fee_rate":0.0004,"mark_price_round":0.1,"min_balance_short":0.5,"min_order_margin":0.1,"multiplier":0.0001,"order_price_deviate":0.2,"order_price_round":0.1,"order_size_max":1,"order_size_min":10,"orders_limit":100000,"ref_discount_rate":0.1,"ref_rebate_rate":0.3,"strike_price":50000,"tag":"WEEK","taker_fee_rate":0.0004,"underlying":"BTC_USDT","time":1639051907,"time_ms":1639051907000}`,
			exp: &WebsocketOptionsContract{
				Contract: contract, CreateTime: types.Time(time.Unix(1637917026, 0)), ExpirationTime: types.Time(time.Unix(1638230400, 0)),
				InitMarginHigh: 0.15, InitMarginLow: 0.1, IsCall: true, MaintMarginBase: 0.075, MakerFeeRate: 0.0004, MarkPriceRound: 0.1,
				MinBalanceShort: 0.5, MinOrderMargin: 0.1, Multiplier: 0.0001, OrderPriceDeviate: 0.2, OrderPriceRound: 0.1, OrderSizeMax: 1,
				OrderSizeMin: 10, OrdersLimit: 100000, RefDiscountRate: 0.1, RefRebateRate: 0.3, StrikePrice: 50000, Tag: "WEEK",
				TakerFeeRate: 0.0004, Underlying: "BTC_USDT", Time: types.Time(time.UnixMilli(1639051907000)),
			},
		},
		{
			channel: optionsOrderbookTickerChannel,
			result:  `{"t":1615366379123,"u":2517661076,"s":"BTC_USDT-20211130-50000-C","b":"54696.6","B":37000,"a":"54696.7","A":47061}`,
			exp:     &WebsocketOptionsOrderbookTicker{UpdateTimestamp: types.Time(time.UnixMilli(1615366379123)), UpdateID: 2517661076, ContractName: "BTC_USDT-20211130-50000-C", BidPrice: 54696.6, BidSize: 37000, AskPrice: 54696.7, AskSize: 47061},
		},
	} {
		t.Run(tc.channel, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			push := `{"time":1630576300,"time_ms":1630576300123,"channel":"` + tc.channel + `","event":"update","result":` + tc.result + `}`
			require.NoError(t, ex.WebsocketHandleOptionsData(t.Context(), nil, []byte(push)), "WebsocketHandleOptionsData must not error")
			require.Len(t, ex.Websocket.DataHandler.C, 1, "WebsocketHandleOptionsData must relay the push")
			assert.Equal(t, tc.exp, (<-ex.Websocket.DataHandler.C).Data, "WebsocketHandleOptionsData should relay the decoded result")
		})
	}
}

func TestGenerateOptionsPayload(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	contract := currency.NewPairWithDelimiter("BTC", "USDT-20211130-65000-C", currency.UnderscoreDelimiter)

	_, err := ex.generateOptionsPayload(t.Context(), subscribeEvent, nil)
	require.ErrorIs(t, err, errNoChannelsSupplied, "generateOptionsPayload must reject an empty list")
	_, err = ex.generateOptionsPayload(t.Context(), subscribeEvent, subscription.List{{Channel: optionsTradesChannel, Pairs: currency.Pairs{contract, contract}}})
	require.ErrorIs(t, err, subscription.ErrNotSinglePair, "generateOptionsPayload must reject a subscription to several contracts")

	for _, tc := range []struct {
		name    string
		channel string
		pair    currency.Pair
		params  map[string]any
		payload []string
		auth    bool
		err     error
		errText string
	}{
		{name: "contract channel", channel: optionsTradesChannel, pair: contract, payload: []string{"BTC_USDT-20211130-65000-C"}},
		{name: "underlying channel", channel: optionsUnderlyingTickersChannel, pair: contract, payload: []string{"BTC_USDT"}},
		{name: "underlying channel without an underlying", channel: optionsUnderlyingPriceChannel, pair: currency.Pair{Base: currency.BTC}, errText: "invalid currency pair"},
		{name: "balances", channel: optionsBalancesChannel, pair: contract, params: map[string]any{"user_id": uint64(666)}, payload: []string{"666"}, auth: true},
		{name: "user channel", channel: optionsOrdersChannel, pair: contract, params: map[string]any{"user_id": uint64(666)}, payload: []string{"666", "BTC_USDT-20211130-65000-C"}, auth: true},
		{name: "user channel without a user ID", channel: optionsOrdersChannel, pair: contract, err: errUserIDRequired},
		{name: "balances without a user ID", channel: optionsBalancesChannel, pair: contract, err: errUserIDRequired},
		{name: "user trades with a mistyped user ID", channel: optionsUserTradesChannel, pair: contract, params: map[string]any{"user_id": "666"}, err: errUserIDRequired},
		{name: "liquidates without a user ID", channel: optionsLiquidatesChannel, pair: contract, err: errUserIDRequired},
		{name: "user settlements without a user ID", channel: optionsUserSettlementChannel, pair: contract, err: errUserIDRequired},
		{name: "position closes without a user ID", channel: optionsPositionCloseChannel, pair: contract, err: errUserIDRequired},
		{name: "positions without a user ID", channel: optionsPositionsChannel, pair: contract, err: errUserIDRequired},
		{name: "order book", channel: optionsOrderbookChannel, pair: contract, params: map[string]any{"accuracy": "0", "level": "20"}, payload: []string{"BTC_USDT-20211130-65000-C", "20", "0"}},
		{name: "order book without accuracy", channel: optionsOrderbookChannel, pair: contract, params: map[string]any{"level": "20"}, err: orderbook.ErrOrderbookInvalid},
		{name: "order book without level", channel: optionsOrderbookChannel, pair: contract, params: map[string]any{"accuracy": "0"}, err: orderbook.ErrOrderbookInvalid},
		{name: "order book update", channel: optionsOrderbookUpdateChannel, pair: contract, params: map[string]any{"interval": kline.HundredMilliseconds, "level": 20}, payload: []string{"BTC_USDT-20211130-65000-C", "100ms", "20"}},
		{name: "order book update without level", channel: optionsOrderbookUpdateChannel, pair: contract, params: map[string]any{"interval": kline.HundredMilliseconds}, payload: []string{"BTC_USDT-20211130-65000-C", "100ms"}},
		{name: "order book update without interval", channel: optionsOrderbookUpdateChannel, pair: contract, err: orderbook.ErrOrderbookInvalid},
		{name: "order book update with an unsupported interval", channel: optionsOrderbookUpdateChannel, pair: contract, params: map[string]any{"interval": kline.ThreeMin}, err: kline.ErrUnsupportedInterval},
		{name: "contract candles", channel: optionsContractCandlesticksChannel, pair: contract, params: map[string]any{"interval": kline.FiveMin}, payload: []string{"5m", "BTC_USDT-20211130-65000-C"}},
		{name: "underlying candles", channel: optionsUnderlyingCandlesticksChannel, pair: contract, params: map[string]any{"interval": kline.FiveMin}, payload: []string{"5m", "BTC_USDT"}},
		{name: "candles without interval", channel: optionsContractCandlesticksChannel, pair: contract, errText: "missing options underlying candlesticks interval"},
		{name: "candles with an unsupported interval", channel: optionsContractCandlesticksChannel, pair: contract, params: map[string]any{"interval": kline.ThreeMin}, err: kline.ErrUnsupportedInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payloads, err := ex.generateOptionsPayload(t.Context(), subscribeEvent, subscription.List{{Channel: tc.channel, Pairs: currency.Pairs{tc.pair}, Asset: asset.Options, Params: tc.params}})
			switch {
			case tc.errText != "":
				require.ErrorContains(t, err, tc.errText, "generateOptionsPayload must return the expected error")
				return
			default:
				require.ErrorIs(t, err, tc.err, "generateOptionsPayload must return the expected error")
				if tc.err != nil {
					return
				}
			}
			require.Len(t, payloads, 1, "generateOptionsPayload must return one payload")
			assert.Equal(t, tc.channel, payloads[0].Channel, "generateOptionsPayload should name the channel")
			assert.Equal(t, subscribeEvent, payloads[0].Event, "generateOptionsPayload should set the event")
			assert.Equal(t, tc.payload, payloads[0].Payload, "generateOptionsPayload should build the payload")
			if !tc.auth {
				assert.Nil(t, payloads[0].Auth, "generateOptionsPayload should not sign a public channel")
				return
			}
			require.NotNil(t, payloads[0].Auth, "generateOptionsPayload must sign a user channel")
			assert.Equal(t, "mock-api-key", payloads[0].Auth.Key, "generateOptionsPayload should sign with the API key")
		})
	}

	noCredentials := new(Exchange)
	require.NoError(t, testexch.Setup(noCredentials), "Setup must not error")
	_, err = noCredentials.generateOptionsPayload(t.Context(), subscribeEvent, subscription.List{{Channel: optionsOrdersChannel, Pairs: currency.Pairs{contract}, Asset: asset.Options, Params: map[string]any{"user_id": uint64(666)}}})
	assert.Error(t, err, "generateOptionsPayload should need credentials to sign a user channel")
}

func TestGenerateOptionsDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := ex.GenerateOptionsDefaultSubscriptions(t.Context())
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")
	var userChannels int
	for _, s := range subs {
		if s.Channel != optionsUserTradesChannel && s.Channel != optionsBalancesChannel {
			continue
		}
		userChannels++
		userID, ok := s.Params["user_id"].(uint64)
		require.Truef(t, ok, "GenerateOptionsDefaultSubscriptions must set a user ID for %s", s.Channel)
		if mockTests {
			assert.Equalf(t, uint64(666), userID, "GenerateOptionsDefaultSubscriptions should use the options account's user ID for %s", s.Channel)
		}
	}
	assert.NotZero(t, userChannels, "GenerateOptionsDefaultSubscriptions should add the authenticated channels")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = ex.GenerateOptionsDefaultSubscriptions(ctx)
	assert.ErrorIs(t, err, context.Canceled, "GenerateOptionsDefaultSubscriptions should request the options account under the context it is given")
}

func TestWebsocketHandleOptionsDataNullResult(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel     string
		event       string
		err         error
		errContains string
	}{
		{channel: optionsContractTickersChannel, event: "update", errContains: "currency pair not set"},
		{channel: optionsOrderbookUpdateChannel, event: "update", err: currency.ErrCurrencyPairEmpty},
		{channel: optionsOrderbookChannel, event: "all", errContains: "currency pair not set"},
	} {
		t.Run(tc.channel, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			var err error
			require.NotPanics(t, func() {
				err = ex.WebsocketHandleOptionsData(t.Context(), nil, []byte(`{"time":1,"channel":"`+tc.channel+`","event":"`+tc.event+`","result":null}`))
			}, "WebsocketHandleOptionsData must not panic on a null result")
			if tc.errContains != "" {
				assert.ErrorContains(t, err, tc.errContains, "WebsocketHandleOptionsData should reject a null result")
				return
			}
			assert.ErrorIs(t, err, tc.err, "WebsocketHandleOptionsData should return the expected error for a null result")
		})
	}
}

func TestProcessOptionsTradesPushData(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.SetTradeFeedStatus(false)
	ex.SetSaveTradeDataStatus(false)
	require.NoError(t, ex.processOptionsTradesPushData([]byte(`{`)), "processOptionsTradesPushData must ignore pushes while the trade feed and saving are off")
	ex.SetTradeFeedStatus(true)
	assert.Error(t, ex.processOptionsTradesPushData([]byte(`{`)), "processOptionsTradesPushData should reject malformed JSON")
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	// Gate signs the size by the taker's side, so a negative size is a taker sell
	require.NoError(t, ex.processOptionsTradesPushData([]byte(`{"time":1630576356,"channel":"options.trades","event":"update","result":[{"id":1,"create_time":1639144526,"create_time_ms":1639144526597,"contract":"BTC_USDT-20211231-59800-C","size":-1,"price":"0.1","underlying":"BTC_USDT","is_call":true},{"id":2,"create_time_ms":1639144527597,"contract":"BTC_USDT-20211231-59800-C","size":3,"price":"0.2","underlying":"BTC_USDT","is_call":true}]}`)), "processOptionsTradesPushData must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "processOptionsTradesPushData must relay the trades")
	contract := currency.NewPairWithDelimiter("BTC", "USDT-20211231-59800-C", currency.UnderscoreDelimiter)
	exp := []trade.Data{
		{TID: "1", Exchange: ex.Name, CurrencyPair: contract, AssetType: asset.Options, Side: order.Sell, Price: 0.1, Amount: 1, Timestamp: time.UnixMilli(1639144526597)},
		{TID: "2", Exchange: ex.Name, CurrencyPair: contract, AssetType: asset.Options, Side: order.Buy, Price: 0.2, Amount: 3, Timestamp: time.UnixMilli(1639144527597)},
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processOptionsTradesPushData should relay the taker side and an unsigned amount")
}

func TestProcessOptionsOrderPushData(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	assert.Error(t, ex.processOptionsOrderPushData(t.Context(), []byte(`{`)), "processOptionsOrderPushData should reject malformed JSON")
	assert.ErrorContains(t, ex.processOptionsOrderPushData(t.Context(), []byte(`{"time":1630654851,"channel":"options.orders","event":"update","result":[{"id":106,"status":"bogus"}]}`)), "unrecognised order status", "processOptionsOrderPushData should reject an unknown status")
	require.Empty(t, ex.Websocket.DataHandler.C, "processOptionsOrderPushData must not relay a rejected push")
	contract := currency.NewPairWithDelimiter("BTC", "USDT-20211130-65000-C", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name     string
		order    string
		status   order.Status
		side     order.Side
		executed float64
	}{
		{name: "open bid", order: `"size":10,"left":4,"status":"open","finish_as":""`, status: order.Open, side: order.Long, executed: 6},
		{name: "filled ask", order: `"size":-10,"left":0,"status":"finished","finish_as":"filled"`, status: order.Filled, side: order.Short, executed: 10},
		{name: "cancelled ask", order: `"size":-10,"left":-7,"status":"finished","finish_as":"cancelled"`, status: order.Cancelled, side: order.Short, executed: 3},
		{name: "market maker protection", order: `"size":5,"left":5,"status":"finished","finish_as":"mmp_cancelled"`, status: order.Cancelled, side: order.Long},
		{name: "finished without finish_as", order: `"size":5,"left":5,"status":"finished","finish_as":""`, status: order.Cancelled, side: order.Long},
	} {
		push := `{"time":1630654851,"channel":"options.orders","event":"update","result":[{"contract":"BTC_USDT-20211130-65000-C","create_time":1637897000,"fill_price":0,"id":106,"is_close":false,"is_liq":false,"is_reduce_only":false,"price":100,"text":"-","tif":"gtc","underlying":"BTC_USDT","user":"9xxx","time_ms":1637897000123,` + tc.order + `}]}`
		require.NoErrorf(t, ex.processOptionsOrderPushData(t.Context(), []byte(push)), "processOptionsOrderPushData must not error for %s", tc.name)
		require.Lenf(t, ex.Websocket.DataHandler.C, 1, "processOptionsOrderPushData must relay the %s order", tc.name)
		amount := 10.0
		if tc.name == "market maker protection" || tc.name == "finished without finish_as" {
			amount = 5
		}
		exp := []order.Detail{{
			Amount:          amount,
			Exchange:        ex.Name,
			OrderID:         "106",
			Side:            tc.side,
			Status:          tc.status,
			Pair:            contract,
			Date:            time.UnixMilli(1637897000123),
			ExecutedAmount:  tc.executed,
			RemainingAmount: amount - tc.executed,
			Price:           100,
			AssetType:       asset.Options,
			AccountID:       "9xxx",
		}}
		assert.Equalf(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processOptionsOrderPushData should relay the %s order", tc.name)
	}
}

func TestProcessOptionsUserTradesPushData(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.SetFillsFeedStatus(false)
	require.NoError(t, ex.processOptionsUserTradesPushData([]byte(`{`)), "processOptionsUserTradesPushData must ignore pushes while the fills feed is off")
	ex.SetFillsFeedStatus(true)
	assert.Error(t, ex.processOptionsUserTradesPushData([]byte(`{`)), "processOptionsUserTradesPushData should reject malformed JSON")
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	// The size is signed by the user's side of the trade
	require.NoError(t, ex.processOptionsUserTradesPushData([]byte(`{"time":1639144214,"channel":"options.usertrades","event":"update","result":[{"id":"1","underlying":"BTC_USDT","order":"557940","contract":"BTC_USDT-20211216-44800-C","create_time":1639144214,"create_time_ms":1639144214583,"price":"4999","role":"taker","size":-2,"fee":"0.001","text":"t-xer01sax4yu"},{"id":"2","underlying":"BTC_USDT","order":"557941","contract":"BTC_USDT-20211216-44800-C","create_time_ms":1639144215583,"price":"5000","role":"maker","size":3}]}`)), "processOptionsUserTradesPushData must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "processOptionsUserTradesPushData must relay the fills")
	contract := currency.NewPairWithDelimiter("BTC", "USDT-20211216-44800-C", currency.UnderscoreDelimiter)
	exp := []fill.Data{
		{Timestamp: time.UnixMilli(1639144214583), Exchange: ex.Name, AssetType: asset.Options, CurrencyPair: contract, Side: order.Short, OrderID: "557940", TradeID: "1", Price: 4999, Amount: 2},
		{Timestamp: time.UnixMilli(1639144215583), Exchange: ex.Name, AssetType: asset.Options, CurrencyPair: contract, Side: order.Long, OrderID: "557941", TradeID: "2", Price: 5000, Amount: 3},
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processOptionsUserTradesPushData should relay each fill's side and unsigned amount")
}

// matchRecorderConnection records the reply it is asked to route
type matchRecorderConnection struct {
	websocket.Connection
	signature any
	data      []byte
}

func (c *matchRecorderConnection) RequireMatchWithData(signature any, data []byte) error {
	c.signature, c.data = signature, data
	return nil
}

func TestWebsocketHandleOptionsDataErrors(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	assert.ErrorIs(t, ex.WebsocketHandleOptionsData(t.Context(), nil, []byte(`{"time_ms":"x","channel":"options.ul_price"}`)), strconv.ErrSyntax, "WebsocketHandleOptionsData should reject an invalid header")

	conn := &matchRecorderConnection{}
	reply := []byte(`{"time":1630576352,"id":7,"channel":"options.ul_price","event":"subscribe","result":{"status":"success"}}`)
	require.NoError(t, ex.WebsocketHandleOptionsData(t.Context(), conn, reply), "WebsocketHandleOptionsData must route a subscription reply")
	assert.Equal(t, int64(7), conn.signature, "WebsocketHandleOptionsData should route the reply by its ID")
	assert.Equal(t, reply, conn.data, "WebsocketHandleOptionsData should route the whole reply")
}

func TestProcessOptionsOrderbookSnapshotPushData(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	call := currency.NewPairWithDelimiter("BTC", "USDT-20211130-65000-C", currency.UnderscoreDelimiter)
	put := currency.NewPairWithDelimiter("BTC", "USDT-20211130-65000-P", currency.UnderscoreDelimiter)

	assert.Error(t, ex.processOptionsOrderbookSnapshotPushData(t.Context(), "all", []byte(`{`), time.Now()), "processOptionsOrderbookSnapshotPushData should reject a malformed snapshot")
	require.NoError(t, ex.processOptionsOrderbookSnapshotPushData(t.Context(), "all", []byte(`{"t":1639144526597,"contract":"BTC_USDT-20211130-65000-C","id":2,"asks":[{"p":"6","s":1}],"bids":[{"p":"5","s":2}]}`), time.Now()), "processOptionsOrderbookSnapshotPushData must load a full snapshot")
	ob, err := ex.Websocket.Orderbook.GetOrderbook(call, asset.Options)
	require.NoError(t, err, "GetOrderbook must find the snapshot")
	assert.Equal(t, orderbook.Levels{{Price: 5, Amount: 2}}, ob.Bids, "processOptionsOrderbookSnapshotPushData should load the snapshot's bids")
	assert.Equal(t, orderbook.Levels{{Price: 6, Amount: 1}}, ob.Asks, "processOptionsOrderbookSnapshotPushData should load the snapshot's asks")

	assert.Error(t, ex.processOptionsOrderbookSnapshotPushData(t.Context(), "update", []byte(`{`), time.Now()), "processOptionsOrderbookSnapshotPushData should reject a malformed update")
	assert.ErrorContains(t, ex.processOptionsOrderbookSnapshotPushData(t.Context(), "update", []byte(`[]`), time.Now()), "missing orderbook ask and bid data", "processOptionsOrderbookSnapshotPushData should reject an empty update")
	assert.ErrorIs(t, ex.processOptionsOrderbookSnapshotPushData(t.Context(), "update", []byte(`[{"p":"1","s":1,"c":"_"}]`), time.Now()), currency.ErrCreatingPair, "processOptionsOrderbookSnapshotPushData should reject an invalid contract")
	// A positive size is a bid and a negative one an ask; each contract gets its own book
	require.NoError(t, ex.processOptionsOrderbookSnapshotPushData(t.Context(), "update", []byte(`[{"p":"7","s":3,"c":"BTC_USDT-20211130-65000-P"},{"p":"8","s":-4,"c":"BTC_USDT-20211130-65000-P"},{"p":"9","s":-1,"c":"BTC_USDT-20211130-65000-C"},{"p":"6","s":5,"c":"BTC_USDT-20211130-65000-P"},{"p":"8.5","s":-2,"c":"BTC_USDT-20211130-65000-P"}]`), time.Now()), "processOptionsOrderbookSnapshotPushData must load an update event")
	ob, err = ex.Websocket.Orderbook.GetOrderbook(put, asset.Options)
	require.NoError(t, err, "GetOrderbook must find the put's book")
	assert.Equal(t, orderbook.Levels{{Price: 7, Amount: 3}, {Price: 6, Amount: 5}}, ob.Bids, "processOptionsOrderbookSnapshotPushData should load every positive size as a bid")
	assert.Equal(t, orderbook.Levels{{Price: 8, Amount: 4}, {Price: 8.5, Amount: 2}}, ob.Asks, "processOptionsOrderbookSnapshotPushData should load every negative size as an unsigned ask")
	ob, err = ex.Websocket.Orderbook.GetOrderbook(call, asset.Options)
	require.NoError(t, err, "GetOrderbook must find the call's book")
	assert.Equal(t, orderbook.Levels{{Price: 9, Amount: 1}}, ob.Asks, "processOptionsOrderbookSnapshotPushData should replace the call's asks")

	ex.ValidateOrderbook = true
	assert.ErrorIs(t, ex.processOptionsOrderbookSnapshotPushData(t.Context(), "update", []byte(`[{"p":"0","s":1,"c":"BTC_USDT-20211130-65000-C"}]`), time.Now()), orderbook.ErrPriceZero, "processOptionsOrderbookSnapshotPushData should return a failure to load the book")
}

func TestProcessOptionsOrderbookUpdate(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	assert.Error(t, ex.processOptionsOrderbookUpdate(t.Context(), []byte(`{`), asset.Options, time.Now()), "processOptionsOrderbookUpdate should reject malformed JSON")
	assert.ErrorIs(t, ex.processOptionsOrderbookUpdate(t.Context(), []byte(`{"t":1615366381417,"s":"","U":2517661101,"u":2517661113}`), asset.Options, time.Now()), currency.ErrCurrencyPairEmpty, "processOptionsOrderbookUpdate should reject an update without a contract")
}

var wsOptionsPushDataHandler = map[string]string{
	"options.contract_tickers":      `{"time": 1630576352,	"channel": "options.contract_tickers",	"event": "update",	"result": {"name": "BTC_USDT-20211231-59800-P",    "last_price": "11349.5",    "mark_price": "11170.19",    "index_price": "",    "position_size": 993,    "bid1_price": "10611.7",    "bid1_size": 100,    "ask1_price": "11728.7",    "ask1_size": 100,    "vega": "34.8731",    "theta": "-72.80588",    "rho": "-28.53331",    "gamma": "0.00003",    "delta": "-0.78311",    "mark_iv": "0.86695",    "bid_iv": "0.65481",    "ask_iv": "0.88145",    "leverage": "3.5541112718136"	}}`,
	"options.ul_tickers":            `{"time": 1630576352,	"channel": "options.ul_tickers",	"event": "update",	"result": {"trade_put": 800,	   "trade_call": 41700,	   "index_price": "50695.43",	   "name": "BTC_USDT"	}}`,
	"options.trades":                `{"time": 1630576356,	"channel": "options.trades",	"event": "update",	"result": [{"contract": "BTC_USDT-20211231-59800-C",        "create_time": 1639144526,        "id": 12279,        "price": 997.8,        "size": -100,        "create_time_ms": 1639144526597,        "underlying": "BTC_USDT"    }	]}`,
	"options.ul_trades":             `{"time": 1630576356,	"channel": "options.ul_trades",	"event": "update",	"result": [{"contract": "BTC_USDT-20211231-59800-C","create_time": 1639144526,"id": 12279,"price": 997.8,"size": -100,"create_time_ms": 1639144526597,"underlying": "BTC_USDT","is_call": true}	]}`,
	"options.ul_price":              `{"time": 1630576356,	"channel": "options.ul_price",	"event": "update",	"result": {"underlying": "BTC_USDT",	   "price": 49653.24,"time": 1639143988,"time_ms": 1639143988931}}`,
	"options.mark_price":            `{"time": 1630576356,	"channel": "options.mark_price",	"event": "update",	"result": {"contract": "BTC_USDT-20211231-59800-P",    "price": 11021.27,    "time": 1639143401,    "time_ms": 1639143401676}}`,
	"options.settlements":           `{"time": 1630576356,	"channel": "options.settlements",	"event": "update",	"result": {"contract": "BTC_USDT-20211130-55000-P",	   "orderbook_id": 2,	   "position_size": 1,	   "profit": 0.5,"settle_price": 70000,	   "strike_price": 65000,	   "tag": "WEEK",	   "trade_id": 1,	   "trade_size": 1,	   "underlying": "BTC_USDT",	   "time": 1639051907,	   "time_ms": 1639051907000}}`,
	"options.contracts":             `{"time": 1630576356,	"channel": "options.contracts",	"event": "update",	"result": {	   "contract": "BTC_USDT-20211130-50000-P",	   "create_time": 1637917026,	   "expiration_time": 1638230400,"init_margin_high": 0.15,	   "init_margin_low": 0.1,	   "is_call": false,	   "maint_margin_base": 0.075,	   "maker_fee_rate": 0.0004,	   "mark_price_round": 0.1,	   "min_balance_short": 0.5,	   "min_order_margin": 0.1,	   "multiplier": 0.0001,	   "order_price_deviate": 0,	   "order_price_round": 0.1,	   "order_size_max": 1,	   "order_size_min": 10,	   "orders_limit": 100000,	   "ref_discount_rate": 0.1,	   "ref_rebate_rate": 0,	   "strike_price": 50000,	   "tag": "WEEK",	   "taker_fee_rate": 0.0004,	   "underlying": "BTC_USDT",	   "time": 1639051907,	   "time_ms": 1639051907000}}`,
	"options.contract_candlesticks": `{	"time": 1630650451,	"channel": "options.contract_candlesticks",	"event": "update",	"result": [{"t": 1639039260,       "v": 100,       "c": "1041.4",       "h": "1041.4",       "l": "1041.4","o": "1041.4",       "a": "0",       "n": "10s_BTC_USDT-20211231-59800-C"   }	]}`,
	"options.ul_candlesticks":       `{"time": 1630650451,	"channel": "options.ul_candlesticks",	"event": "update",	"result": [{"t": 1639039260,        "v": 100,        "c": "1041.4",        "h": "1041.4",        "l": "1041.4","o": "1041.4",        "a": "0",        "n": "10s_BTC_USDT"    }	]}`,
	"options.book_ticker":           `{"time": 1630650452,	"channel": "options.book_ticker",	"event": "update",	"result": {    "t": 1615366379123,    "u": 2517661076,    "s": "BTC_USDT-20211130-50000-C",    "b": "54696.6",    "B": 37000,    "a": "54696.7",    "A": 47061	}}`,
	"options.order_book_update":     `{"time": 1630650445,	"channel": "options.order_book_update",	"event": "update",	"result": {    "t": 1615366381417,    "s": "BTC_USDT-20211130-50000-C",    "U": 2517661101,    "u": 2517661113,    "b": [{"p": "54672.1","s": 95        },        {            "p": "54664.5",            "s": 58794        }    ],    "a": [        {            "p": "54743.6",            "s": 95        },        {            "p": "54742",            "s": 95        }    ]	}}`,
	"options.order_book":            `{"time": 1630650445,	"channel": "options.order_book",	"event": "all",	"result": {"t": 1541500161123,    "contract": "BTC_USDT-20211130-50000-C",    "id": 93973511,    "asks": [{"p": "97.1","s": 2245},{"p": "97.2",            "s": 2245        }    ],    "bids": [		{            "p": "97.2",            "s": 2245        },        {            "p": "97.1",            "s": 2245        }    ]	}}`,
	"options.order_book-1":          `{"channel": "options.order_book",	"event": "update",	"time": 1630650445,	"result": [{"p": "49525.6",		"s": 7726,		"c": "BTC_USDT-20211130-50000-C",		"id": 93973511	  }	]}`,
	"options.book_ticker-2":         `{"time": 1630650452,	"channel": "options.book_ticker",	"event": "update",	"result": {"t": 1615366379123,    "u": 2517661076,    "s": "BTC_USDT-20211130-50000-C",    "b": "54696.6",    "B": 37000,    "a": "54696.7",    "A": 47061	}}`,
	"options.order_book_update-3":   `{"time": 1630650445,	"channel": "options.order_book_update",	"event": "update",	"result": {"t": 1615366381417,    "s": "BTC_USDT-20211130-50000-C",    "U": 2517661101,"u": 2517661113,    "b": [{"p": "54672.1","s": 95},{"p": "54664.5","s": 58794}],"a": [{"p": "54743.6","s": 95},{"p": "54742","s": 95}]}}`,
	"options.order_book-4":          `{"time": 1630650445,	"channel": "options.order_book",	"event": "all",	"result": {"t": 1541500161123,    "contract": "BTC_USDT-20211130-50000-C",    "id": 93973511,    "asks": [{"p": "97.1","s": 2245},{"p": "97.2","s": 2245}    ],    "bids": [{"p": "97.2",            "s": 2245        },        {            "p": "97.1",            "s": 2245        }    ]	}}`,
	"options.order_book-5":          `{"channel": "options.order_book",	"event": "update",	"time": 1630650445,	"result": [{"p": "49525.6",		"s": 7726,		"c": "BTC_USDT-20211130-50000-C",		"id": 93973511	  }	]}`,
	"options.liquidates":            `{"channel": "options.liquidates",	"event": "update",	"time": 1630654851,	"result": [{"user": "1xxxx",		  "init_margin": 1190,		  "maint_margin": 1042.5,		  "order_margin": 0,		  "time": 1639051907,		  "time_ms": 1639051907000}	]}`,
	"options.positions":             `{"time": 1630654851,	"channel": "options.positions",	"event": "update",	"error": null,	"result": [	   {		  "entry_price": 0,		  "realised_pnl": -13.028,		  "size": 0,		  "contract": "BTC_USDT-20211130-65000-C",		  "user": "9010",		  "time": 1639051907,		  "time_ms": 1639051907000}	]}`,
	"options.pong":                  `{"time":1756700469,"channel":"options.pong","event":"","result":null}`,
}

func TestWebsocketHandleOptionsData(t *testing.T) {
	t.Parallel()
	for k, v := range wsOptionsPushDataHandler {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			err := e.WebsocketHandleOptionsData(t.Context(), nil, []byte(v))
			require.NoError(t, err)
		})
	}
}

var wsOptionsAuthenticatedDataHandler = map[string]string{
	"options.orders":           `{"time": 1630654851,"channel": "options.orders",	"event": "update",	"result": [{"contract": "BTC_USDT-20211130-65000-C",		  "create_time": 1637897000,		  "fill_price": 0,"finish_as": "cancelled",		  "iceberg": 0,		  "id": 106,		  "is_close": false,		  "is_liq": false,		  "is_reduce_only": false,		  "left": -10,		  "mkfr": 0.0004,		  "price": 15000,		  "refr": 0,		  "refu": 0,		  "size": -10,		  "status": "finished",		  "text": "web",		  "tif": "gtc",		  "tkfr": 0.0004,		  "underlying": "BTC_USDT",		  "user": "9xxx",		  "time": 1639051907,"time_ms": 1639051907000}]}`,
	"options.usertrades":       `{"time": 1639144214,	"channel": "options.usertrades",	"event": "update",	"result": [{"id": "1","underlying": "BTC_USDT","order": "557940","contract": "BTC_USDT-20211216-44800-C","create_time": 1639144214,"create_time_ms": 1639144214583,"price": "4999","role": "taker","size": -1}]}`,
	"options.user_settlements": `{"channel": "options.user_settlements","event": "update",	"time": 1639051907,	"result": [{"contract": "BTC_USDT-20211130-65000-C","realised_pnl": -13.028,"settle_price": 70000,"settle_profit": 5,"size": 10,"strike_price": 65000,"underlying": "BTC_USDT","user": "9xxx","time": 1639051907,"time_ms": 1639051907000}]}`,
	"options.position_closes":  `{"channel": "options.position_closes","event": "update",	"time": 1630654851,	"result": [{"contract": "BTC_USDT-20211130-50000-C","pnl": -0.0056,"settle_size": 0,"side": "long","text": "web","underlying": "BTC_USDT","user": "11xxxxx","time": 1639051907,"time_ms": 1639051907000}]}`,
	"options.balances":         `{"channel": "options.balances",	"event": "update",	"time": 1630654851,	"result": [{"balance": 60.79009,"change": -0.5,"text": "BTC_USDT-20211130-55000-P","type": "set","user": "11xxxx","time": 1639051907,"time_ms": 1639051907000}]}`,
}

func TestPrivateWebsocketHandleOptionsData(t *testing.T) {
	t.Parallel()
	for k, v := range wsOptionsAuthenticatedDataHandler {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "test", Secret: "test"})
			err := e.WebsocketHandleOptionsData(ctx, nil, []byte(v))
			require.NoError(t, err)
		})
	}
}

func TestGenerateOptionsDefaultSubscriptionsSkipsDisabledAssetBeforeAuth(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	ex.SetDefaults()
	ex.API.AuthenticatedWebsocketSupport = true
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Options, false), "SetAssetEnabled must not error")
	got, err := ex.GenerateOptionsDefaultSubscriptions(t.Context())
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error for disabled asset")
	require.Empty(t, got, "disabled options asset must not generate subscriptions")
	require.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticated endpoints must not be changed when disabled options are skipped before auth")
}
