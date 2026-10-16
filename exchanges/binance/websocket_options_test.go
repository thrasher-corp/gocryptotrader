package binance

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestGenerateOptionsSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	subs, err := ex.generateOptionsPublicSubscriptions()
	require.NoError(t, err, "generateOptionsPublicSubscriptions must not error")
	testsubs.EqualLists(t, subscription.List{
		{Channel: subscription.TickerChannel, Asset: asset.Options, Pairs: currency.Pairs{testOptionsPair}, QualifiedChannel: "btc-261030-85000-c@optionTicker"},
		{Channel: subscription.AllTradesChannel, Asset: asset.Options, Pairs: currency.Pairs{testOptionsPair}, QualifiedChannel: "btc-261030-85000-c@optionTrade"},
		{Channel: subscription.OrderbookChannel, Asset: asset.Options, Pairs: currency.Pairs{testOptionsPair}, Interval: kline.HundredMilliseconds, QualifiedChannel: "btc-261030-85000-c@depth@100ms"},
	}, subs)

	subs, err = ex.generateOptionsMarketSubscriptions()
	require.NoError(t, err, "generateOptionsMarketSubscriptions must not error")
	testsubs.EqualLists(t, subscription.List{
		{Channel: subscription.CandlesChannel, Asset: asset.Options, Pairs: currency.Pairs{testOptionsPair}, Interval: kline.OneMin, QualifiedChannel: "btc-261030-85000-c@kline_1m"},
	}, subs)

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Options, false), "SetAssetEnabled must not error")
	for _, generate := range []func() (subscription.List, error){ex.generateOptionsPublicSubscriptions, ex.generateOptionsMarketSubscriptions} {
		subs, err = generate()
		require.NoError(t, err, "generating subscriptions for a disabled asset must not error")
		assert.Empty(t, subs, "a disabled asset should generate no subscriptions, so its connection is skipped")
	}
}

func TestWsHandleOptionsData(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	useTestOrderbookSync(ex, func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
		return nil, errTestNoSnapshot
	})
	for _, frame := range fixtureFrames(t, "testdata/wsOptions.json") {
		require.NoErrorf(t, ex.wsHandleOptionsData(t.Context(), nil, frame), "wsHandleOptionsData must not error for %s", frame)
	}
	exp := []any{
		&OptionsBookTicker{
			EventType: "bookTicker", UpdateID: 42788078076, Symbol: "BTC-261030-85000-C", BestBidPrice: 3645, BestBidQuantity: 2.17, BestAskPrice: 3675,
			BestAskQuantity: 6.18, TransactionTime: fixtureTime(1791283337764), EventTime: fixtureTime(1791283337765),
		},
		&ticker.Price{
			Last: 3660, LastSize: 1.25, VolumeWeightedAveragePrice: 3650.092, High: 4085, Low: 3540, BaseVolume: 175, QuoteVolume: 643766.05, Open: 3904,
			PercentChange24Hour: -6.25, Pair: testOptionsPair, ExchangeName: ex.Name, AssetType: asset.Options, LastUpdated: time.UnixMilli(1791283461766),
		},
		[]trade.Data{{
			TID: "467", Exchange: ex.Name, CurrencyPair: testOptionsPair, AssetType: asset.Options, Side: order.Buy, Price: 3660, Amount: 1,
			Timestamp: time.UnixMilli(1791283461746),
		}},
		[]OptionsIndexPrice{
			{EventType: "indexPrice", EventTime: fixtureTime(1791283430989), UnderlyingSymbol: "BTCUSDT", IndexPrice: 86176.8923913},
			{EventType: "indexPrice", EventTime: fixtureTime(1791283430989), UnderlyingSymbol: "ETHUSDT", IndexPrice: 2716.86162791},
		},
		kline.Item{
			Exchange: ex.Name, Pair: testOptionsPair, Asset: asset.Options, Interval: kline.OneMin,
			Candles: []kline.Candle{{Time: time.UnixMilli(1791283440000), Open: 3610, High: 3620, Low: 3605, Close: 3615, Volume: 1.2}},
		},
		[]OptionsMarkPrice{{
			Symbol: "BTC-261030-85000-C", MarkPrice: 3889.71, EventTime: fixtureTime(1791283430983), EventType: "markPrice", IndexPrice: 86176.8923913,
			EstimatedSettlePrice: 3880.5, BestBuyPrice: 3885, BestSellPrice: 3895, BestBuyQuantity: 2.4, BestSellQuantity: 4.62, BuyImpliedVolatility: 0.36169887,
			SellImpliedVolatility: 0.36233783, BuyMaximumPrice: 7000, SellMinimumPrice: 780, Volatility: 0.362, RiskFreeRate: 0.0525, Delta: 0.40766128,
			Theta: -35.46280967, Gamma: 0.0000266, Vega: 156.51083319,
		}},
		&OptionsNewSymbol{
			EventType: "optionSymbol", EventTime: fixtureTime(1669356423908), Symbol: "BTC-261127-90000-C", Underlying: "BTCUSDT", QuotationAsset: currency.USDT,
			OptionType: "CALL", StrikePrice: 90000, DeliveryDateTime: fixtureTime(1798012800000), Unit: 1, OnboardDateTime: fixtureTime(1791283200000),
			ContractStatus: "TRADING",
		},
		[]WsOptionsOpenInterest{{
			EventType: "openInterest", EventTime: fixtureTime(1791283200045), Symbol: "BTC-261030-85000-C", OpenInterest: 1580.87, OpenInterestUSDT: 136129592.178168204,
		}},
	}
	assert.Equal(t, exp, relayedPayloads(ex), "wsHandleOptionsData should relay every stream of enabled pairs")

	got, err := storedOrderbookState(ex, testOptionsPair, asset.Options)
	require.NoError(t, err, "the partial depth snapshot must be stored")
	assert.Equal(t, &orderbookState{
		Bids:         orderbook.Levels{{Price: 3645, Amount: 2.17}, {Price: 3640, Amount: 5.12}},
		Asks:         orderbook.Levels{{Price: 3670, Amount: 0.56}, {Price: 3675, Amount: 5.62}},
		LastUpdateID: 42788078248,
		LastUpdated:  time.UnixMilli(1791283337855),
		LastPushed:   time.UnixMilli(1791283337870),
	}, got, "partial depth should load the top levels into the options order book")
}

func TestHandleOptionsErrors(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	conn := &matchRecordingConnection{}
	require.NoError(t, ex.wsHandleOptionsData(t.Context(), conn, []byte(`{"result":null,"id":11}`)), "a request reply must be routed")
	require.NoError(t, ex.wsHandleOptionsUserData(t.Context(), conn, []byte(`{"result":null,"id":12}`)), "a request reply must be routed")
	assert.Equal(t, []any{int64(11), int64(12)}, conn.signatures, "request replies should be routed by their id")

	for frame, wantErr := range map[string]error{
		`{"stream":"btcusdt@unknown","data":{"e":"unknownEvent"}}`:                             errUnhandledStreamEvent,
		`{"stream":"x@optionTicker","data":{"e":"24hrTicker","s":"NOT-LISTED"}}`:               currency.ErrPairNotFound,
		`{"stream":"x@optionTrade","data":{"e":"trade","s":"NOT-LISTED"}}`:                     currency.ErrPairNotFound,
		`{"stream":"x@optionTrade","data":{"e":"trade","s":"BTC-261030-85000-C","S":"UP"}}`:    order.ErrSideIsInvalid,
		`{"stream":"x@kline_1m","data":{"e":"kline","s":"NOT-LISTED","k":{"i":"1m"}}}`:         currency.ErrPairNotFound,
		`{"stream":"x@kline_1m","data":{"e":"kline","s":"BTC-261030-85000-C","k":{"i":"1x"}}}`: kline.ErrInvalidInterval,
		`{"stream":"x@depth@100ms","data":{"e":"depthUpdate","s":"NOT-LISTED"}}`:               currency.ErrPairNotFound,
		`{"stream":"` + testListenKey + `","data":{"e":"UNKNOWN_EVENT"}}`:                      errUnhandledStreamEvent,
		`{"stream":"` + testListenKey + `","data":{"E":1}}`:                                    errUnhandledStreamEvent,
		`{"stream":"` + testListenKey + `"}`:                                                   errUnhandledStreamEvent,
	} {
		handle := ex.wsHandleOptionsData
		if strings.HasPrefix(frame, `{"stream":"`+testListenKey) {
			handle = ex.wsHandleOptionsUserData
		}
		err := handle(t.Context(), conn, []byte(frame))
		require.ErrorIsf(t, err, wantErr, "options handlers must reject %s", frame)
		assert.NotContains(t, err.Error(), testListenKey, "errors should never quote the listen key")
	}
	for _, frame := range []string{
		`{"stream":"x@depth@100ms","data":{"e":"depthUpdate","U":"invalid"}}`,
		`{"stream":"x@optionTicker","data":{"e":"24hrTicker","E":"invalid"}}`,
		`{"stream":"x@optionTrade","data":{"e":"trade","t":"invalid"}}`,
		`{"stream":"x@kline_1m","data":{"e":"kline","E":"invalid"}}`,
		`{"stream":"!index@arr","data":[{"e":"indexPrice","E":"invalid"}]}`,
	} {
		assert.Errorf(t, ex.wsHandleOptionsData(t.Context(), conn, []byte(frame)), "wsHandleOptionsData should report the decoding error of %s", frame)
	}
	for data, wantErr := range map[string]error{
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTC-261030-85000-C","S":"UP"}}`:                                   order.ErrSideIsInvalid,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTC-261030-85000-C","S":"SELL","o":"ICEBERG"}}`:                   order.ErrUnrecognisedOrderType,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTC-261030-85000-C","S":"SELL","o":"LIMIT","X":"SUSPENDED"}}`:     errUnknownOrderStatus,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTC-261030-85000-C","S":"SELL","o":"LIMIT","X":"NEW","f":"GTE"}}`: order.ErrInvalidTimeInForce,
	} {
		assert.ErrorIsf(t, ex.wsHandleOptionsUserData(t.Context(), conn, []byte(`{"stream":"`+testListenKey+`","data":`+data+`}`)), wantErr, "wsHandleOptionsUserData should reject %s", data)
	}
	for _, data := range []string{
		`{"e":"BALANCE_POSITION_UPDATE","E":"invalid"}`,
		`{"e":"ORDER_TRADE_UPDATE","E":"invalid"}`,
	} {
		assert.Errorf(t, ex.wsHandleOptionsUserData(t.Context(), conn, []byte(`{"stream":"`+testListenKey+`","data":`+data+`}`)), "wsHandleOptionsUserData should reject %s", data)
	}
	// Disabled pairs are skipped
	require.NoError(t, ex.wsHandleOptionsData(t.Context(), conn, []byte(`{"stream":"x@kline_1m","data":{"e":"kline","s":"ETH-261009-2700-C","k":{"i":"1m"}}}`)), "a disabled pair must be skipped")
	assert.Empty(t, relayedPayloads(ex), "nothing should be relayed for rejected or disabled frames")
}

func TestWsHandleOptionsUserData(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	for _, frame := range fixtureFrames(t, "testdata/wsOptionsUserData.json") {
		if isEventFrame(frame, "listenKeyExpired") {
			continue // TestProcessListenKeyExpired renews the stream over a connection
		}
		require.NoErrorf(t, ex.wsHandleOptionsUserData(t.Context(), nil, frame), "wsHandleOptionsUserData must not error for %s", frame)
	}
	payloads := relayedPayloads(ex)
	normaliseBalanceTimes(t, payloads)
	exp := []any{
		&OptionsAccountUpdate{
			EventType: "ACCOUNT_UPDATE", EventTime: fixtureTime(1762914568643), TransactionTime: fixtureTime(1762914568619), Equity: 10000371.61462086,
			AdjustedEquity: 10000475.51032086, WalletBalance: 10000475.51032086, PositionValue: -103.8957, UnrealizedPNL: 16.1043,
			InitialMargin: 32354.38562539, MaintenanceMargin: 6089.28766956,
		},
		accounts.SubAccounts{{AssetType: asset.Options, Balances: accounts.CurrencyBalances{
			currency.USDT: {Currency: currency.USDT, Total: 10000471.379409, Free: 10000471.379409},
		}}},
		[]futures.Position{{
			Exchange: ex.Name, Asset: asset.Options, Pair: testOptionsPair, NotionalSize: decimal.MustFromFloat(-366).Abs(), Status: order.Open,
			OpeningPrice: decimal.MustFromFloat(3660), OpeningDirection: order.Short, LatestSize: decimal.MustFromFloat(-0.1).Abs(),
			LatestDirection: order.Short, LastUpdated: time.UnixMilli(1762917544206),
		}},
		&order.Detail{
			TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 3660, Amount: 0.2, AverageExecutedPrice: 3660, ExecutedAmount: 0.1, Cost: 366,
			RemainingAmount: 0.1, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "4611869636869226548", ClientOrderID: "TEST",
			Type: order.Limit, Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.Options, Date: time.UnixMilli(1762917544206),
			LastUpdated: time.UnixMilli(1762917544206), Pair: testOptionsPair,
			Trades: []order.TradeHistory{{
				Price: 3660, Amount: 0.1, Fee: 0.0732, Exchange: ex.Name, TID: "467", Type: order.Limit, Side: order.Sell,
				Timestamp: time.UnixMilli(1762917544206), IsMaker: true, FeeAsset: "USDT",
			}},
		},
		&OptionsGreekUpdate{
			EventType: "GREEK_UPDATE", EventTime: fixtureTime(1762917544216), TransactionTime: fixtureTime(1762917544215),
			Greeks: []OptionsGreekUpdateItem{{Underlying: "BTCUSDT", Delta: -0.01304097, Gamma: -0.00000124, Theta: 16.116481, Vega: -3.83444011}},
		},
		&OptionsRiskLevelChange{EventType: "RISK_LEVEL_CHANGE", EventTime: fixtureTime(1587727187525), RiskLevel: "REDUCE_ONLY", MarginBalance: 1534.11708371, MaintenanceMargin: 254789.11708371},
	}
	assert.Equal(t, exp, payloads, "wsHandleOptionsUserData should relay every user data event")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.GetBalance("", creds, asset.Options, currency.USDT)
	require.NoError(t, err, "the balance update must store the balance")
	assert.Equal(t, 10000471.379409, stored.Total, "the balance update should store the account balance")
}

func TestOptionsOrderTradeUpdateFills(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	// A limit sell of 0.3 is placed, then filled by two trades, each charged a fee that options report as negative
	for _, u := range []struct {
		execution, status, last, cumulative, averagePrice, commission string
		tradeID                                                       uint64
		tradeTime                                                     int64
	}{
		{"NEW", "NEW", "0", "0", "0", "0", 0, 1791300000000},
		{"TRADE", "PARTIALLY_FILLED", "0.1", "0.1", "3660", "-0.1098", 468, 1791300000100},
		{"TRADE", "FILLED", "0.2", "0.3", "3660", "-0.2196", 469, 1791300000200},
	} {
		frame := fmt.Appendf(nil, `{"stream":%q,"data":{"e":"ORDER_TRADE_UPDATE","E":%d,"T":%d,"o":{"s":"BTC-261030-85000-C","c":"two-fills","S":"SELL","o":"LIMIT","f":"GTC","q":"0.3","p":"3660","ap":%q,"x":%q,"X":%q,"i":4611869636869226549,"l":%q,"z":%q,"L":"3660","N":"USDT","n":%q,"T":%d,"t":%d,"b":"0","a":"0.3","m":true,"R":false,"ot":"LIMIT","rp":"0","V":"NONE"}}}`,
			testListenKey, u.tradeTime+1, u.tradeTime, u.averagePrice, u.execution, u.status, u.last, u.cumulative, u.commission, u.tradeTime, u.tradeID)
		require.NoErrorf(t, ex.wsHandleOptionsUserData(t.Context(), nil, frame), "wsHandleOptionsUserData must not error for the %s update", u.status)
	}
	trades := []order.TradeHistory{
		{
			Price: 3660, Amount: 0.1, Fee: 0.1098, Exchange: ex.Name, TID: "468", Type: order.Limit, Side: order.Sell,
			Timestamp: time.UnixMilli(1791300000100), IsMaker: true, FeeAsset: "USDT",
		},
		{
			Price: 3660, Amount: 0.2, Fee: 0.2196, Exchange: ex.Name, TID: "469", Type: order.Limit, Side: order.Sell,
			Timestamp: time.UnixMilli(1791300000200), IsMaker: true, FeeAsset: "USDT",
		},
	}
	exp := []any{
		&order.Detail{
			TimeInForce: order.GoodTillCancel, Price: 3660, Amount: 0.3, RemainingAmount: 0.3, FeeAsset: currency.USDT, Exchange: ex.Name,
			OrderID: "4611869636869226549", ClientOrderID: "two-fills", Type: order.Limit, Side: order.Sell, Status: order.New,
			AssetType: asset.Options, Date: time.UnixMilli(1791300000000), LastUpdated: time.UnixMilli(1791300000000), Pair: testOptionsPair,
		},
		&order.Detail{
			TimeInForce: order.GoodTillCancel, Price: 3660, Amount: 0.3, AverageExecutedPrice: 3660, ExecutedAmount: 0.1, Cost: 366,
			RemainingAmount: 0.2, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "4611869636869226549", ClientOrderID: "two-fills", Type: order.Limit,
			Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.Options, Date: time.UnixMilli(1791300000100),
			LastUpdated: time.UnixMilli(1791300000100), Pair: testOptionsPair, Trades: trades[:1],
		},
		&order.Detail{
			TimeInForce: order.GoodTillCancel, Price: 3660, Amount: 0.3, AverageExecutedPrice: 3660, ExecutedAmount: 0.3, Cost: 1098,
			FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "4611869636869226549", ClientOrderID: "two-fills", Type: order.Limit, Side: order.Sell,
			Status: order.Filled, AssetType: asset.Options, Date: time.UnixMilli(1791300000200), LastUpdated: time.UnixMilli(1791300000200),
			Pair: testOptionsPair, Trades: trades[1:],
		},
	}
	payloads := relayedPayloads(ex)
	require.Equal(t, exp, payloads, "wsHandleOptionsUserData must relay each update with its own fill")

	// Order stores keep the first detail of an order and merge later ones into it
	first, ok := payloads[0].(*order.Detail)
	require.True(t, ok, "the first payload must be an order detail")
	stored := first.Copy()
	for _, p := range payloads[1:] {
		d, ok := p.(*order.Detail)
		require.True(t, ok, "every payload must be an order detail")
		require.NoError(t, stored.UpdateOrderFromDetail(d), "UpdateOrderFromDetail must not error")
	}
	assert.Equal(t, trades, stored.Trades, "the stored order should keep every fill with its own fee")
	assert.Zero(t, stored.Fee, "the stored order should leave the fees to its trades")
	assert.Equal(t, 1098.0, stored.Cost, "the stored order should keep the cost of its latest update")
	assert.Equal(t, time.UnixMilli(1791300000000), stored.Date, "the stored order should be dated when it was placed")
}

func TestOptionsPayloadDecoding(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		fixture string
		key     string
		target  any
		exp     any
	}{
		{
			name: "diff depth", fixture: "testdata/wsOptions.json", key: "btc-261030-85000-c@depth@100ms", target: new(OptionsDepthUpdate),
			exp: &OptionsDepthUpdate{
				EventType: "depthUpdate", EventTime: fixtureTime(1791283337817), TransactionTime: fixtureTime(1791283337764), Symbol: "BTC-261030-85000-C",
				FirstUpdateID: 42788077974, FinalUpdateID: 42788078076, PreviousFinalUpdateID: 42788074837,
				Bids: orderbook.LevelsArrayPriceAmount{{Price: 3645, Amount: 2.17}},
				Asks: orderbook.LevelsArrayPriceAmount{{Price: 3675, Amount: 6.18}, {Price: 3680, Amount: 4.01}},
			},
		},
		{
			name: "ticker", fixture: "testdata/wsOptions.json", key: "btc-261030-85000-c@optionTicker", target: new(OptionsTicker),
			exp: &OptionsTicker{
				EventType: "24hrTicker", EventTime: fixtureTime(1791283461766), Symbol: "BTC-261030-85000-C", PriceChange: -244, PriceChangePercent: -0.0625,
				WeightedAveragePrice: 3650.092, LastPrice: 3660, LastQuantity: 1.25, OpenPrice: 3904, HighPrice: 4085, LowPrice: 3540, TradingVolume: 175,
				TradeAmount: 643766.05, StatisticsOpenTime: fixtureTime(1791201120000), StatisticsCloseTime: fixtureTime(1791283461766), FirstTradeID: 62,
				LastTradeID: 468, TotalNumberOfTrades: 407,
			},
		},
		{
			name: "trade", fixture: "testdata/wsOptions.json", key: "btcusdt@optionTrade", target: new(WsOptionsTrade),
			exp: &WsOptionsTrade{
				EventType: "trade", EventTime: fixtureTime(1791283461747), TradeCompletedTime: fixtureTime(1791283461746), Symbol: "BTC-261030-85000-C",
				TradeID: 467, Price: 3660, Quantity: 1, TradeType: "MARKET", Direction: "BUY", IsBuyerMaker: true,
			},
		},
		{
			name: "kline", fixture: "testdata/wsOptions.json", key: "btc-261030-85000-c@kline_1m", target: new(OptionsKline),
			exp: &OptionsKline{
				EventType: "kline", EventTime: fixtureTime(1791283502013), Symbol: "BTC-261030-85000-C",
				Kline: OptionsKlineData{
					StartTime: fixtureTime(1791283440000), EndTime: fixtureTime(1791283499999), Symbol: "BTC-261030-85000-C", CandlePeriod: "1m",
					FirstTradeID: -1, LastTradeID: -1, Open: 3610, Close: 3615, High: 3620, Low: 3605, Volume: 1.2, NumberOfTrades: 2, IsCompleted: true,
					CompletedTradeAmount: 4335, TakerCompletedTradeVolume: 0.8, TakerTradeAmount: 2892,
				},
			},
		},
		{
			name: "balance and position update", fixture: "testdata/wsOptionsUserData.json", key: "BALANCE_POSITION_UPDATE", target: new(OptionsBalancePositionUpdate),
			exp: &OptionsBalancePositionUpdate{
				EventType: "BALANCE_POSITION_UPDATE", EventTime: fixtureTime(1762917544216), TransactionTime: fixtureTime(1762917544206), EventReasonType: "ORDER",
				Balances:  []OptionsBalanceUpdate{{MarginAsset: currency.USDT, AccountBalance: 10000471.379409, BalanceChange: -120.5}},
				Positions: []OptionsPositionUpdate{{Symbol: "BTC-261030-85000-C", PositionQuantity: -0.1, PositionValue: -366, AverageEntryPrice: 3660}},
			},
		},
		{
			name: "order trade update", fixture: "testdata/wsOptionsUserData.json", key: "ORDER_TRADE_UPDATE", target: new(OptionsOrderTradeUpdate),
			exp: &OptionsOrderTradeUpdate{
				EventType: "ORDER_TRADE_UPDATE", EventTime: fixtureTime(1762917544216), TransactionTime: fixtureTime(1762917544206),
				Order: OptionsOrderTradeUpdateOrder{
					Symbol: "BTC-261030-85000-C", ClientOrderID: "TEST", Side: "SELL", OrderType: "LIMIT", TimeInForce: "GTC", OriginalQuantity: 0.2,
					OriginalPrice: 3660, AveragePrice: 3660, ExecutionType: "TRADE", OrderStatus: "PARTIALLY_FILLED", OrderID: 4611869636869226548,
					OrderLastFilledQuantity: 0.1, OrderFilledAccumulatedQuantity: 0.1, LastFilledPrice: 3660, CommissionAsset: currency.USDT, Commission: -0.0732,
					OrderTradeTime: fixtureTime(1762917544206), TradeID: 467, BidsQuantity: 0.1, AskQuantity: 0.3, IsMaker: true, IsReduceOnly: true,
					OriginalOrderType: "LIMIT", RealizedProfit: 1.5, SelfTradePreventionMode: "EXPIRE_BOTH",
				},
			},
		},
	} {
		require.NoErrorf(t, json.Unmarshal(fixtureFrameData(t, tc.fixture, tc.key), tc.target), "Unmarshal must not error for the %s", tc.name)
		assert.Equalf(t, tc.exp, tc.target, "the %s should decode every field", tc.name)
	}
}
