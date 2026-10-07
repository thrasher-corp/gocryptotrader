package binance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
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

func TestGenerateCFuturesSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	subs, err := ex.generateCFuturesSubscriptions()
	require.NoError(t, err, "generateCFuturesSubscriptions must not error")
	testsubs.EqualLists(t, subscription.List{
		{Channel: subscription.TickerChannel, Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{testCFuturesPair}, QualifiedChannel: "btcusd_perp@ticker"},
		{Channel: subscription.AllTradesChannel, Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{testCFuturesPair}, QualifiedChannel: "btcusd_perp@aggTrade"},
		{Channel: subscription.CandlesChannel, Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{testCFuturesPair}, Interval: kline.OneMin, QualifiedChannel: "btcusd_perp@kline_1m"},
		{Channel: subscription.OrderbookChannel, Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{testCFuturesPair}, Interval: kline.HundredMilliseconds, QualifiedChannel: "btcusd_perp@depth@100ms"},
	}, subs)

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.CoinMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = ex.generateCFuturesSubscriptions()
	require.NoError(t, err, "generateCFuturesSubscriptions must not error for a disabled asset")
	assert.Empty(t, subs, "a disabled asset should generate no subscriptions, so its connection is skipped")
}

func TestWsHandleCFuturesData(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	useTestOrderbookSync(ex, func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
		return nil, errTestNoSnapshot
	})
	for _, frame := range fixtureFrames(t, "testdata/wsCFutures.json") {
		require.NoErrorf(t, ex.wsHandleCFuturesData(t.Context(), nil, frame), "wsHandleCFuturesData must not error for %s", frame)
	}
	miniTicker := ticker.Price{
		Last: 86137.8, High: 86664.7, Low: 84911.4, Open: 85973.7, BaseVolume: 8584, Pair: testCFuturesPair, ExchangeName: ex.Name,
		AssetType: asset.CoinMarginedFutures, LastUpdated: time.UnixMilli(1791283463856),
	}
	tick := ticker.Price{
		Last: 86137.8, LastSize: 4, VolumeWeightedAveragePrice: 85677.87248792, High: 86664.7, Low: 84911.4, Open: 85973.7, PercentChange24Hour: 0.191,
		BaseVolume: 8584, Pair: testCFuturesPair, ExchangeName: ex.Name, AssetType: asset.CoinMarginedFutures, LastUpdated: time.UnixMilli(1791283463856),
	}
	markPrice := FuturesMarkPrice{
		EventType: "markPriceUpdate", EventTime: fixtureTime(1791283451000), Symbol: "BTCUSD_PERP", MarkPrice: 86135.5, IndexPrice: 86169.88180356,
		EstimatedSettlePrice: 86066.64151075, FundingRate: 0.00007058, MarkPriceMovingAverage: 86135.4, NextFundingTime: fixtureTime(1791302400000), SymbolType: 2,
	}
	exp := []any{
		[]trade.Data{{
			TID: "499297335", Exchange: ex.Name, CurrencyPair: testCFuturesPair, AssetType: asset.CoinMarginedFutures, Side: order.Sell,
			Price: 86137.7, Amount: 3, Timestamp: time.UnixMilli(1791283466336),
		}},
		&FuturesBookTicker{
			EventType: "bookTicker", UpdateID: 11747023575855, EventTime: fixtureTime(1791283450746), TransactionTime: fixtureTime(1791283450745),
			Symbol: "BTCUSD_PERP", Pair: "BTCUSD", BestBidPrice: 86135.4, BestBidQuantity: 2649, BestAskPrice: 86135.5, BestAskQuantity: 3010, SymbolType: 2,
		},
		&FuturesLiquidationOrder{
			EventType: "forceOrder", EventTime: fixtureTime(1591154240950),
			Order: FuturesLiquidationOrderDetail{
				Symbol: "BTCUSD_PERP", Pair: "BTCUSD", Side: "SELL", OrderType: "LIMIT", TimeInForce: "IOC", OriginalQuantity: 2, Price: 85425.5,
				AveragePrice: 85496.5, OrderStatus: "FILLED", OrderLastFilledQuantity: 1, OrderFilledAccumulatedQuantity: 2,
				OrderTradeTime: fixtureTime(1591154240949), SymbolType: 2,
			},
		},
		[]ticker.Price{miniTicker},
		&miniTicker,
		[]ticker.Price{tick},
		&tick,
		&FuturesContinuousKline{
			EventType: "continuous_kline", EventTime: fixtureTime(1791283466336), Pair: "BTCUSD", ContractType: "PERPETUAL",
			Kline: FuturesContinuousKlineData{
				StartTime: fixtureTime(1791283440000), CloseTime: fixtureTime(1791283499999), Interval: "1m", FirstUpdateID: 11747022867989,
				LastUpdateID: 11747024833030, OpenPrice: 86111, ClosePrice: 86137.7, HighPrice: 86137.8, LowPrice: 86111, Volume: 546, NumberOfTrades: 53,
				IsClosed: true, QuoteAssetVolume: 0.63398896, TakerBuyVolume: 543, TakerBuyQuoteAssetVolume: 0.63050617, Ignore: 1,
			},
		},
		&FuturesContractInfo{
			EventType: "contractInfo", EventTime: fixtureTime(1669647330375), Symbol: "BTCUSD_PERP", Pair: "BTCUSD", ContractType: "PERPETUAL",
			DeliveryDateTime: fixtureTime(4133404800000), OnboardDateTime: fixtureTime(1666594800000), ContractStatus: "TRADING",
			NotionalBrackets: []FuturesContractInfoBracket{
				{NotionalBracket: 1, BracketFloorNotional: 0.5, BracketNotionalCap: 500000, MaintenanceRatio: 0.0065, AuxiliaryNumber: 75, MinLeverage: 51, MaxLeverage: 75},
			},
			SymbolType: 2,
		},
		kline.Item{
			Exchange: ex.Name, Pair: testCFuturesPair, Asset: asset.CoinMarginedFutures, Interval: kline.OneMin,
			Candles: []kline.Candle{{
				Time: time.UnixMilli(1791283440000), Open: 86111, High: 86137.8, Low: 86111, Close: 86137.7, Volume: 546, ValidationIssues: kline.PartialCandle,
			}},
		},
		&markPrice,
		[]FuturesMarkPrice{markPrice, {
			EventType: "markPriceUpdate", EventTime: fixtureTime(1791283451000), Symbol: "BTCUSD_261225", MarkPrice: 87125.1, IndexPrice: 86169.88180356,
			EstimatedSettlePrice: 87020.64151075, MarkPriceMovingAverage: 87124.9, SymbolType: 2,
		}},
		&CFuturesIndexPrice{EventType: "IndexUpdate", EventTime: fixtureTime(1791283451000), Pair: "BTCUSD", IndexPrice: 86169.89067719},
		&CFuturesPriceKline{
			EventType: "indexPrice_kline", EventTime: fixtureTime(1791283451057), Pair: "BTCUSD",
			Kline: CFuturesPriceKlineData{
				StartTime: fixtureTime(1791283440000), CloseTime: fixtureTime(1791283499999), Symbol: "0", Interval: "1m", FirstTradeID: 1791283440000,
				LastTradeID: 1791283451000, OpenPrice: 86151.64923644, ClosePrice: 86169.89067719, HighPrice: 86170.98258838, LowPrice: 86151.64923644,
				Volume: 1, NumberOfBasicData: 12, IsClosed: true, QuoteAssetVolume: 2, TakerBuyBaseAssetVolume: 3, TakerBuyQuoteAssetVolume: 4, Ignore: 5,
			},
		},
		&CFuturesPriceKline{
			EventType: "markPrice_kline", EventTime: fixtureTime(1791283451080), Pair: "BTCUSD",
			Kline: CFuturesPriceKlineData{
				StartTime: fixtureTime(1791283440000), CloseTime: fixtureTime(1791283499999), Symbol: "BTCUSD_PERP", Interval: "1m", FirstTradeID: 1791283440000,
				LastTradeID: 1791283451000, OpenPrice: 86111, ClosePrice: 86135.5, HighPrice: 86135.6, LowPrice: 86110.9, Volume: 1, NumberOfBasicData: 12,
				QuoteAssetVolume: 2, TakerBuyBaseAssetVolume: 3, TakerBuyQuoteAssetVolume: 4, Ignore: 5,
			},
		},
	}
	assert.Equal(t, exp, relayedPayloads(ex), "wsHandleCFuturesData should relay every stream of the connection's product and enabled pairs")

	got, err := storedOrderbookState(ex, testCFuturesPair, asset.CoinMarginedFutures)
	require.NoError(t, err, "the partial depth snapshot must be stored")
	assert.Equal(t, &orderbookState{
		Bids:         orderbook.Levels{{Price: 86135.4, Amount: 2649}, {Price: 86135.3, Amount: 21}},
		Asks:         orderbook.Levels{{Price: 86135.5, Amount: 3502}, {Price: 86135.6, Amount: 32}},
		LastUpdateID: 11747023540358,
		LastUpdated:  time.UnixMilli(1791283450210),
		LastPushed:   time.UnixMilli(1791283450213),
	}, got, "partial depth should load the top levels into the COIN-M order book")
}

func TestWsHandleCFuturesUserData(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	for _, frame := range fixtureFrames(t, "testdata/wsCFuturesUserData.json") {
		if isEventFrame(frame, "listenKeyExpired") {
			continue // TestProcessListenKeyExpired renews the stream over a connection
		}
		require.NoErrorf(t, ex.wsHandleCFuturesUserData(t.Context(), nil, frame), "wsHandleCFuturesUserData must not error for %s", frame)
	}
	payloads := relayedPayloads(ex)
	normaliseBalanceTimes(t, payloads)
	exp := []any{
		&FuturesMarginCall{
			EventType: "MARGIN_CALL", EventTime: fixtureTime(1587727187525), AccountAlias: "SfsR", CrossWalletBalance: 3.16812045,
			Positions: []FuturesMarginCallPosition{{
				Symbol: "BTCUSD_PERP", PositionSide: "LONG", PositionAmount: 132, MarginType: "CROSSED", IsolatedWallet: 0.1, MarkPrice: 86187.17127,
				UnrealizedPNL: -0.0001166074, MaintenanceMarginRequired: 0.0001614445,
			}},
		},
		accounts.SubAccounts{{AssetType: asset.CoinMarginedFutures, Balances: accounts.CurrencyBalances{
			currency.BTC: {Currency: currency.BTC, Total: 1.12345678, Free: 1.12345678},
		}}},
		[]futures.Position{{
			Exchange: ex.Name, Asset: asset.CoinMarginedFutures, Pair: testCFuturesPair, PositionMargin: decimal.MustFromFloat(0.00002325),
			RealisedPNL: decimal.MustFromFloat(0.0002), UnrealisedPNL: decimal.MustFromFloat(0.00001), Status: order.Open,
			OpeningPrice: decimal.MustFromFloat(86000), OpeningDirection: order.Long, LatestSize: decimal.MustFromFloat(20).Abs(),
			LatestDirection: order.Long, LastUpdated: time.UnixMilli(1564745798938),
		}},
		&order.Detail{
			TimeInForce: order.GoodTillCrossing, ReduceOnly: true, Price: 86000.1, Amount: 2, TriggerPrice: 86100.1, AverageExecutedPrice: 86000.1,
			ExecutedAmount: 2, FeeAsset: currency.BTC, Exchange: ex.Name, OrderID: "8888888", ClientOrderID: "TEST", Type: order.Limit,
			Side: order.Buy, Status: order.Filled, AssetType: asset.CoinMarginedFutures, Date: time.UnixMilli(1591274595442),
			LastUpdated: time.UnixMilli(1591274595442), Pair: testCFuturesPair,
			Trades: []order.TradeHistory{{
				Price: 86000.1, Amount: 1, Fee: 0.00000116, Exchange: ex.Name, TID: "1157454910", Type: order.Limit, Side: order.Buy,
				Timestamp: time.UnixMilli(1591274595442), IsMaker: true, FeeAsset: "BTC",
			}},
		},
		&FuturesAccountConfigUpdate{
			EventType: "ACCOUNT_CONFIG_UPDATE", EventTime: fixtureTime(1611646737479), TransactionTime: fixtureTime(1611646737476),
			AccountConfiguration: &FuturesAccountConfiguration{Symbol: "BTCUSD_PERP", Leverage: 25},
		},
		&FuturesStrategyUpdate{
			EventType: "STRATEGY_UPDATE", TransactionTime: fixtureTime(1669261797627), EventTime: fixtureTime(1669261797628),
			StrategyUpdate: FuturesStrategyUpdateData{StrategyID: 176054595, StrategyType: "GRID", StrategyStatus: "WORKING", Symbol: "BTCUSD_PERP", UpdateTime: fixtureTime(1669261797627), OpCode: 8001},
		},
		&FuturesGridUpdate{
			EventType: "GRID_UPDATE", TransactionTime: fixtureTime(1669262908216), EventTime: fixtureTime(1669262908218),
			GridUpdate: FuturesGridUpdateData{
				StrategyID: 176057040, StrategyType: "GRID", StrategyStatus: "WORKING", Symbol: "BTCUSD_PERP", RealizedPNL: -0.00000716,
				UnmatchedAveragePrice: 86720, UnmatchedQuantity: -1, UnmatchedFee: -0.00000716, MatchedPNL: 0.00000001, UpdateTime: fixtureTime(1669262908197),
			},
		},
	}
	assert.Equal(t, exp, payloads, "wsHandleCFuturesUserData should relay every user data event")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.GetBalance("", creds, asset.CoinMarginedFutures, currency.BTC)
	require.NoError(t, err, "the account update must store the balance")
	assert.Equal(t, 1.12345678, stored.Total, "the account update should store the wallet balance")
}
