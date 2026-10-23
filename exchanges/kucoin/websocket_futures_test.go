package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	// futuresTestPair is the contract the futures fixtures record, as the available pairs spell it
	futuresTestPair = currency.NewPairWithDelimiter("XBT", "USDTM", "_")
	// futuresLivePair is a contract enabled in the test config, which live subscriptions use
	futuresLivePair = currency.NewPairWithDelimiter("SOL", "USDTM", "_")
)

// TestWebsocketFuturesPayloads decodes the recorded futures pushes whose fields the handlers map onto shared types
func TestWebsocketFuturesPayloads(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		key string
		got any
		exp any
	}{
		{key: "tickerV2", got: new(WebsocketFuturesTickerV2), exp: &WebsocketFuturesTickerV2{
			Symbol:       "XBTUSDTM",
			Sequence:     1748392084691,
			BestBidSize:  284,
			BestBidPrice: 83258.2,
			BestAskPrice: 83258.3,
			BestAskSize:  629,
			Timestamp:    types.Time(time.UnixMilli(1791899571611)),
		}},
		{key: "tickerV1", got: new(WebsocketFuturesTickerV1), exp: &WebsocketFuturesTickerV1{
			Symbol:       "XBTUSDTM",
			Sequence:     1947310382240,
			Side:         "buy",
			Size:         12,
			Price:        83258.3,
			BestBidSize:  3386,
			BestBidPrice: 83258.2,
			BestAskPrice: 83268.3,
			TradeID:      "1947310382240",
			BestAskSize:  5,
			Timestamp:    types.Time(time.UnixMilli(1791899571745)),
		}},
		{key: "level50", got: new(WebsocketFuturesOrderbookDepth), exp: &WebsocketFuturesOrderbookDepth{
			Sequence:      1748392084690,
			Asks:          orderbook.LevelsArrayPriceAmount{{Amount: 634, Price: 83258.3}, {Amount: 76, Price: 83258.4}, {Amount: 84, Price: 83259.4}},
			Bids:          orderbook.LevelsArrayPriceAmount{{Amount: 284, Price: 83258.2}, {Amount: 45, Price: 83253.4}, {Amount: 100, Price: 83251.7}},
			Timestamp:     types.Time(time.UnixMilli(1791899571610)),
			PushTimestamp: types.Time(time.UnixMilli(1791899571610)),
		}},
		{key: "increment", got: new(WebsocketFuturesOrderbookIncrement), exp: &WebsocketFuturesOrderbookIncrement{
			Sequence:  1748392084691,
			Change:    "83258.3,sell,629",
			Timestamp: types.Time(time.UnixMilli(1791899571611)),
		}},
		{key: "klines", got: new(WebsocketCandlestick), exp: &WebsocketCandlestick{
			Symbol: "XBTUSDTM",
			Candles: WebsocketCandle{
				StartTime:         types.Time(time.Unix(1791899520, 0)),
				OpenPrice:         83259.3,
				ClosePrice:        83258.3,
				HighPrice:         83260.1,
				LowPrice:          83238.1,
				TransactionVolume: 8658.8084,
				TransactionAmount: 104,
			},
			Time: types.Time(time.UnixMilli(1791899571770)),
		}},
		{key: "trade", got: new(WebsocketFuturesExecution), exp: &WebsocketFuturesExecution{
			Symbol:       "XBTUSDTM",
			Sequence:     1947310382240,
			Side:         "buy",
			Size:         12,
			Price:        83258.3,
			TakerOrderID: "499484436958625792",
			MakerOrderID: "499484431849955328",
			TradeID:      "1947310382240",
			Timestamp:    types.Time(time.UnixMilli(1791899571745)),
		}},
		{key: "snapshot", got: new(WebsocketFuturesSnapshot), exp: &WebsocketFuturesSnapshot{
			FundingRate:           0.000055,
			HighPrice:             84133.2,
			LastPrice:             83276.5,
			LowPrice:              80985.9,
			Price24HoursBefore:    82697.7,
			PriceChange:           578.8,
			PriceChangePercentage: 0.0069,
			Symbol:                "XBTUSDTM",
			Timestamp:             types.Time(time.Unix(0, 1791899575028643636)),
			Turnover:              204740196.2154,
			Volume:                2474.442,
		}},
		{key: "orderMatch", got: new(WebsocketFuturesOrderChange), exp: &WebsocketFuturesOrderChange{
			Symbol:       "XBTUSDTM",
			OrderType:    "limit",
			Side:         "sell",
			OrderID:      "440761625608192",
			Liquidity:    "taker",
			MarginMode:   "CROSS",
			Type:         "match",
			OrderTime:    types.Time(time.UnixMilli(1743146786640)),
			Size:         3840,
			FilledSize:   1116,
			Price:        84603.44,
			MatchPrice:   85739.69,
			MatchSize:    1000,
			RemainSize:   2724,
			TradeID:      "1740800012709",
			Status:       "match",
			Timestamp:    types.Time(time.UnixMilli(1743146786746)),
			FeeType:      "takerFee",
			TradeType:    "liquid",
			PositionSide: "BOTH",
		}},
		{key: "orderUpdate", got: new(WebsocketFuturesOrderChange), exp: &WebsocketFuturesOrderChange{
			Symbol:        "XBTUSDTM",
			OrderType:     "limit",
			Side:          "buy",
			CanceledSize:  1037,
			OrderID:       "228685469427204099",
			Liquidity:     "maker",
			MarginMode:    "ISOLATED",
			Type:          "update",
			OrderTime:     types.Time(time.Unix(0, 1727336066682194084)),
			Size:          19982,
			FilledSize:    7327,
			Price:         5.029,
			RemainSize:    11618,
			ClientOrderID: "10496pp066R679264",
			OldSize:       19982,
			Status:        "open",
			Timestamp:     types.Time(time.UnixMilli(1727336066766)),
			TradeType:     "trade",
			UserID:        "669a61642857ca000186f626",
		}},
		{key: "orderCanceled", got: new(WebsocketFuturesOrderChange), exp: &WebsocketFuturesOrderChange{
			Symbol:          "XBTUSDTM",
			OrderType:       "limit",
			Side:            "buy",
			CanceledSize:    1,
			OrderID:         "247901211536203776",
			MarginMode:      "ISOLATED",
			Type:            "canceled",
			OrderTime:       types.Time(time.Unix(0, 1731917456611809239)),
			Size:            1,
			Price:           90000,
			AllCanceledSize: 1,
			Status:          "done",
			Timestamp:       types.Time(time.UnixMilli(1731917460806)),
			TradeType:       "adl",
		}},
		{key: "walletBalance", got: new(WebsocketFuturesWalletBalance), exp: &WebsocketFuturesWalletBalance{
			CrossPositionMargin:      17.551016,
			IsolatedOrderMargin:      1.2,
			HoldBalance:              2.3,
			Equity:                   387.224858816,
			Version:                  2118,
			AvailableBalance:         285.652001096,
			IsolatedPositionMargin:   63.98342359,
			MaxWithdrawAmount:        285.645841096,
			WalletBalance:            371.394298816,
			IsolatedFundingFeeMargin: 2.89220199,
			CrossUnrealisedPNL:       2.95996,
			TotalCrossMargin:         310.370835226,
			Currency:                 currency.USDT,
			IsolatedUnrealisedPNL:    12.8706,
			CrossOrderMargin:         10.06002012,
			Timestamp:                types.Time(time.UnixMilli(1741164936624)),
		}},
		{key: "availableBalance", got: new(WebsocketFuturesAvailableBalance), exp: &WebsocketFuturesAvailableBalance{
			AvailableBalance: 5923,
			HoldBalance:      2312,
			Currency:         currency.XBT,
			Timestamp:        types.Time(time.UnixMilli(1553842862614)),
		}},
		{key: "stopOrder", got: new(WebsocketFuturesStopOrder), exp: &WebsocketFuturesStopOrder{
			MarginMode:    "ISOLATED",
			OrderID:       "240673378116083712",
			OrderPrice:    0.1,
			OrderType:     "stop",
			Side:          "buy",
			Size:          1,
			Stop:          "down",
			StopPrice:     1000,
			StopPriceType: "TP",
			Symbol:        "XBTUSDTM",
			Type:          "open",
			CreatedAt:     types.Time(time.UnixMilli(1730194206837)),
			Timestamp:     types.Time(time.UnixMicro(1730194206843133)),
		}},
	} {
		var msg WebsocketMessage
		require.NoErrorf(t, json.Unmarshal(wsFixture(t, "wsFutures.json", tc.key), &msg), "%s must decode", tc.key)
		require.NoErrorf(t, json.Unmarshal(msg.Data, tc.got), "%s data must decode", tc.key)
		assert.Equalf(t, tc.exp, tc.got, "%s should decode every field", tc.key)
	}
}

func TestProcessFuturesTickerV2(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*ticker.Price](t, nil, &subscription.Subscription{Channel: futuresTickerV2Channel, Asset: asset.Futures, Pairs: currency.Pairs{futuresLivePair}})
		assert.Positive(t, got.Bid, "a live tickerV2 push should carry the best bid")
		assert.Positive(t, got.Ask, "a live tickerV2 push should carry the best ask")
		return
	}
	ex := wsTestInstance(t)
	exp := &ticker.Price{
		Bid: 83258.2, BidSize: 284, Ask: 83258.3, AskSize: 629,
		Pair: futuresTestPair, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: time.UnixMilli(1791899571611),
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsFutures.json", "tickerV2"), "a tickerV2 push should set the best bid and ask")
	stored, err := ticker.GetTicker(ex.Name, futuresTestPair, asset.Futures)
	require.NoError(t, err, "the tickerV2 push must be stored")
	assert.Equal(t, exp, stored, "the stored ticker should be the relayed one")

	require.Error(t, ex.processFuturesTickerV2(t.Context(), []byte(`{"bestBidSize":-1}`)), "processFuturesTickerV2 must reject an undecodable push")
	require.ErrorIs(t, ex.processFuturesTickerV2(t.Context(), []byte(`{"symbol":"NOTACONTRACT"}`)), currency.ErrPairNotFound, "processFuturesTickerV2 must reject an unknown contract")
}

// TestProcessFuturesTickerV2KeepsStoredSnapshot covers the tickerV2 channel reporting only the best bid and ask, where
// replacing the stored ticker would clear the snapshot the REST ticker request left behind
func TestProcessFuturesTickerV2KeepsStoredSnapshot(t *testing.T) {
	t.Parallel()
	ku := wsTestInstance(t)
	pair := currency.NewPairWithDelimiter("ETH", "USDCM", "_")
	require.NoError(t, ticker.ProcessTicker(&ticker.Price{
		ExchangeName: ku.Name,
		AssetType:    asset.Futures,
		Pair:         pair,
		Last:         3551,
		LastSize:     12,
		High:         3600,
		Low:          3400,
		BaseVolume:   30449670,
		QuoteVolume:  845169919063,
		OpenInterest: 1234567,
	}), "seeding the stored ticker must not error")

	msg := []byte(`{"topic":"/contractMarket/tickerV2:ETHUSDCM","type":"message","subject":"tickerV2","data":{"symbol":"ETHUSDCM","sequence":1739524604832,"bestBidSize":795,"bestBidPrice":"3200","bestAskPrice":"3600","bestAskSize":284,"ts":1789627572494000000}}`)
	require.NoError(t, ku.wsHandleData(t.Context(), nil, msg), "wsHandleData must not error")
	require.Len(t, ku.Websocket.DataHandler.C, 1, "the push must emit a single ticker")

	got, ok := (<-ku.Websocket.DataHandler.C).Data.(*ticker.Price)
	require.True(t, ok, "the push must emit a ticker price")

	stored, err := ticker.GetTicker(ku.Name, pair, asset.Futures)
	require.NoError(t, err, "the ticker must remain stored")
	exp := &ticker.Price{
		Last:         3551,
		LastSize:     12,
		High:         3600,
		Low:          3400,
		Bid:          3200,
		BidSize:      795,
		Ask:          3600,
		AskSize:      284,
		BaseVolume:   30449670,
		QuoteVolume:  845169919063,
		OpenInterest: 1234567,
		Pair:         pair,
		ExchangeName: ku.Name,
		AssetType:    asset.Futures,
		LastUpdated:  time.Unix(0, 1789627572494000000),
	}
	assert.Equal(t, exp, stored, "the best bid/ask push should keep every stored field it does not report")
	assert.Equal(t, stored, got, "the stored ticker should match the dispatched ticker")
}

// TestProcessFuturesTickerV2WithoutTimestamp covers a tickerV2 frame that carries no ts. The stored snapshot's
// timestamp must not be carried onto the fresh bid and ask, which would make the exchange store them under a stale time
func TestProcessFuturesTickerV2WithoutTimestamp(t *testing.T) {
	t.Parallel()
	ku := wsTestInstance(t)
	pair := currency.NewPairWithDelimiter("ETH", "USDCM", "_")
	require.NoError(t, ticker.ProcessTicker(&ticker.Price{
		ExchangeName: ku.Name,
		AssetType:    asset.Futures,
		Pair:         pair,
		Last:         3551,
		High:         3600,
		Low:          3400,
		LastUpdated:  time.Now().Add(-time.Hour),
	}), "seeding the stored ticker must not error")

	msg := []byte(`{"topic":"/contractMarket/tickerV2:ETHUSDCM","type":"message","subject":"tickerV2","data":{"symbol":"ETHUSDCM","sequence":1739524604832,"bestBidSize":795,"bestBidPrice":"3200","bestAskPrice":"3600","bestAskSize":284}}`)
	before := time.Now()
	require.NoError(t, ku.wsHandleData(t.Context(), nil, msg), "wsHandleData must not error")
	require.Len(t, ku.Websocket.DataHandler.C, 1, "the push must emit a single ticker")

	got, ok := (<-ku.Websocket.DataHandler.C).Data.(*ticker.Price)
	require.True(t, ok, "the push must emit a ticker price")
	assert.False(t, got.LastUpdated.Before(before), "a frame without ts should carry its own receipt time")
	stored, err := ticker.GetTicker(ku.Name, pair, asset.Futures)
	require.NoError(t, err, "the ticker must remain stored")
	assert.False(t, stored.LastUpdated.Before(before), "the untimestamped push should be stored under its own store time")
	exp := &ticker.Price{
		Last:         3551,
		High:         3600,
		Low:          3400,
		Bid:          3200,
		BidSize:      795,
		Ask:          3600,
		AskSize:      284,
		Pair:         pair,
		ExchangeName: ku.Name,
		AssetType:    asset.Futures,
		LastUpdated:  stored.LastUpdated,
	}
	assert.Equal(t, exp, stored, "the untimestamped push should keep every stored field it does not report")
}

func TestProcessFuturesTickerV1(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*ticker.Price](t, nil, &subscription.Subscription{Channel: futuresTickerV1Channel, Asset: asset.Futures, Pairs: currency.Pairs{futuresLivePair}})
		assert.Positive(t, got.Last, "a live Ticker V1 push should carry the last price")
		assert.Positive(t, got.LastSize, "a live Ticker V1 push should carry the last size")
		return
	}
	ex := wsTestInstance(t)
	// Ticker V1's best bid and ask are those at the fill, so the stored ones from Ticker V2 must stay
	require.NoError(t, ticker.ProcessTicker(&ticker.Price{ExchangeName: ex.Name, Pair: futuresTestPair, AssetType: asset.Futures, Bid: 83258.2, Ask: 83258.3, BidSize: 284, AskSize: 629}), "seeding the stored ticker must not error")
	exp := &ticker.Price{
		Last: 83258.3, LastSize: 12, Bid: 83258.2, BidSize: 284, Ask: 83258.3, AskSize: 629,
		Pair: futuresTestPair, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: time.UnixMilli(1791899571745),
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsFutures.json", "tickerV1"), "a Ticker V1 push should set the last fill and keep the stored best bid and ask")

	require.Error(t, ex.processFuturesTickerV1(t.Context(), []byte(`{"size":-1}`)), "processFuturesTickerV1 must reject an undecodable push")
	require.ErrorIs(t, ex.processFuturesTickerV1(t.Context(), []byte(`{"symbol":"NOTACONTRACT"}`)), currency.ErrPairNotFound, "processFuturesTickerV1 must reject an unknown contract")
}

func TestProcessFuturesSnapshot(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*ticker.Price](t, nil, &subscription.Subscription{Channel: futuresSnapshotChannel, Asset: asset.Futures, Pairs: currency.Pairs{futuresLivePair}})
		assert.Positive(t, got.High, "a live snapshot should carry the 24 hour high")
		assert.Positive(t, got.BaseVolume, "a live snapshot should carry the 24 hour volume")
		return
	}
	ex := wsTestInstance(t)
	// The snapshot's last price lags Ticker V1's, so the stored one must stay
	require.NoError(t, ticker.ProcessTicker(&ticker.Price{ExchangeName: ex.Name, Pair: futuresTestPair, AssetType: asset.Futures, Last: 83300, OpenInterest: 5}), "seeding the stored ticker must not error")
	exp := &ticker.Price{
		Last: 83300, High: 84133.2, Low: 80985.9, Open24Hour: 82697.7, BaseVolume: 2474.442, QuoteVolume: 204740196.2154, OpenInterest: 5,
		Pair: futuresTestPair, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: time.Unix(0, 1791899575028643636),
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsFutures.json", "snapshot"), "a snapshot push should set the 24 hour statistics")

	inverse := currency.NewPairWithDelimiter("XBT", "USDM", "_")
	exp = &ticker.Price{
		High: 84027.9, Low: 80888, Open24Hour: 82632.6, BaseVolume: 67.0725106591, QuoteVolume: 5561697,
		Pair: inverse, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: time.Unix(0, 1791899575050723597),
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsFutures.json", "snapshotInverse"), "an inverse contract's snapshot should count its turnover as the base volume")

	require.Error(t, ex.processFuturesSnapshot(t.Context(), []byte(`{"volume":"x"}`)), "processFuturesSnapshot must reject an undecodable push")
	require.ErrorIs(t, ex.processFuturesSnapshot(t.Context(), []byte(`{"symbol":"NOTACONTRACT"}`)), currency.ErrPairNotFound, "processFuturesSnapshot must reject an unknown contract")
}

// TestFuturesTickerFeed covers the futures ticker the default ticker subscription feeds: Ticker V2 sets the best bid
// and ask, Ticker V1 the last fill and the symbol snapshot the 24 hour statistics, so together they keep every field
// of the stored ticker current, whichever arrives first
func TestFuturesTickerFeed(t *testing.T) {
	t.Parallel()
	if !mockTests {
		ex := liveWsConnect(t, nil, &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{futuresLivePair}})
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			stored, err := ticker.GetTicker(ex.Name, futuresLivePair, asset.Futures)
			require.NoError(c, err, "the ticker must be stored")
			assert.Positive(c, stored.Last, "Ticker V1 should set the last price")
			assert.Positive(c, stored.LastSize, "Ticker V1 should set the last size")
			assert.Positive(c, stored.Bid, "Ticker V2 should set the best bid")
			assert.Positive(c, stored.Ask, "Ticker V2 should set the best ask")
			assert.Positive(c, stored.High, "the snapshot should set the 24 hour high")
			assert.Positive(c, stored.BaseVolume, "the snapshot should set the 24 hour volume")
		}, time.Minute, time.Second, "the futures ticker subscription must keep every field of the stored ticker current")
		return
	}
	exp := func(ex *Exchange, lastUpdated time.Time) *ticker.Price {
		return &ticker.Price{
			Last: 83258.3, LastSize: 12, Bid: 83258.2, BidSize: 284, Ask: 83258.3, AskSize: 629,
			High: 84133.2, Low: 80985.9, Open24Hour: 82697.7, BaseVolume: 2474.442, QuoteVolume: 204740196.2154,
			Pair: futuresTestPair, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: lastUpdated,
		}
	}
	for name, keys := range map[string][]string{
		"V2, V1, snapshot": {"tickerV2", "tickerV1", "snapshot"},
		"snapshot, V1, V2": {"snapshot", "tickerV1", "tickerV2"},
		"V1, snapshot, V2": {"tickerV1", "snapshot", "tickerV2"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex := wsTestInstance(t)
			var last *ticker.Price
			for _, key := range keys {
				var ok bool
				last, ok = wsPush(t, ex, "wsFutures.json", key).(*ticker.Price)
				require.Truef(t, ok, "%s must relay a ticker", key)
			}
			stored, err := ticker.GetTicker(ex.Name, futuresTestPair, asset.Futures)
			require.NoError(t, err, "the ticker must be stored")
			assert.Equal(t, exp(ex, last.LastUpdated), stored, "the stored ticker should carry every field the three topics report")
		})
	}
}

func TestIsInverseContract(t *testing.T) {
	t.Parallel()
	for pair, exp := range map[currency.Pair]bool{
		currency.NewPair(currency.XBT, currency.USDTM):           false,
		currency.NewPair(currency.ETH, currency.USDCM):           false,
		currency.NewPair(currency.XBT, currency.USDM):            true,
		currency.NewPair(currency.XBT, currency.NewCode("MZ26")): true,
	} {
		assert.Equalf(t, exp, isInverseContract(pair), "isInverseContract should be %t for %s", exp, pair)
	}
}

func TestProcessFuturesOrderbookSnapshot(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*orderbook.Depth](t, nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{futuresLivePair}})
		book, err := got.Retrieve()
		require.NoError(t, err, "Retrieve must not error")
		assert.Len(t, book.Bids, 5, "a level 5 book should have five bids")
		return
	}
	ex := wsTestInstance(t)
	require.NoError(t, ex.wsHandleData(t.Context(), nil, wsFixture(t, "wsFutures.json", "level50")), "wsHandleData must not error")
	book, err := ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
	require.NoError(t, err, "the level 50 book must be stored")
	exp := &orderbook.Book{
		Bids:         orderbook.Levels{{Price: 83258.2, Amount: 284}, {Price: 83253.4, Amount: 45}, {Price: 83251.7, Amount: 100}},
		Asks:         orderbook.Levels{{Price: 83258.3, Amount: 634}, {Price: 83258.4, Amount: 76}, {Price: 83259.4, Amount: 84}},
		Exchange:     ex.Name,
		Pair:         futuresTestPair,
		Asset:        asset.Futures,
		LastUpdated:  time.UnixMilli(1791899571610),
		LastPushed:   time.UnixMilli(1791899571610),
		LastUpdateID: 1748392084690,
		InsertedAt:   book.InsertedAt,
	}
	assert.Equal(t, exp, book, "the level 50 push should load the book")

	require.Error(t, ex.processFuturesOrderbookSnapshot(t.Context(), nil, []byte(`{"asks":{}}`), "XBTUSDTM"), "processFuturesOrderbookSnapshot must reject an undecodable push")
	require.ErrorIs(t, ex.processFuturesOrderbookSnapshot(t.Context(), nil, []byte(`{}`), "NOTACONTRACT"), currency.ErrPairNotFound, "processFuturesOrderbookSnapshot must reject an unknown contract")
}

func TestProcessFuturesOrderbookIncrement(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*orderbook.Depth](t, nil, &subscription.Subscription{Channel: futuresOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{futuresLivePair}})
		book, err := got.Retrieve()
		require.NoError(t, err, "Retrieve must not error")
		assert.NotEmpty(t, book.Bids, "the realtime book should hold bids")
		return
	}
	for _, tc := range []struct {
		key        string
		bids, asks orderbook.Levels
	}{
		{key: "increment", bids: orderbook.Levels{{Price: 80000, Amount: 1}}, asks: orderbook.Levels{{Price: 83258.3, Amount: 629, ID: 1748392084691}, {Price: 90000, Amount: 1}}},
	} {
		ex := wsTestInstance(t)
		ex.wsOBUpdateMgr = newTestUpdateManager(ex, 1748392084690)
		require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, wsFixture(t, "wsFutures.json", tc.key)), "wsHandleData must not error for %s", tc.key)
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			book, err := ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
			require.NoError(c, err, "GetOrderbook must return the realtime book")
			assert.Equal(c, int64(1748392084691), book.LastUpdateID, "the book should carry the push's sequence")
			assert.Equal(c, tc.bids, book.Bids, "the book's bids should be the snapshot's")
			assert.Equal(c, tc.asks, book.Asks, "the push's ask should be applied")
		}, time.Second, 10*time.Millisecond, "the increment must reach the futures book")
	}

	ex := wsTestInstance(t)
	ex.wsOBUpdateMgr = newTestUpdateManager(ex, 17)
	require.NoError(t, ex.processFuturesOrderbookIncrement(t.Context(), nil, []byte(`{"sequence":18,"change":"85000,buy,83","timestamp":1551770400000}`), "XBTUSDTM"), "processFuturesOrderbookIncrement must not error for a buy")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		book, err := ex.Websocket.Orderbook.GetOrderbook(futuresTestPair, asset.Futures)
		require.NoError(c, err, "GetOrderbook must return the realtime book")
		assert.Equal(c, orderbook.Levels{{Price: 85000, Amount: 83, ID: 18}, {Price: 80000, Amount: 1}}, book.Bids, "a buy change should update the bids")
	}, time.Second, 10*time.Millisecond, "the buy must reach the futures book")

	for _, tc := range []struct {
		name, change string
	}{
		{name: "two parts", change: "5000.0,buy"},
		{name: "price", change: "bad,buy,83"},
		{name: "size", change: "5000.0,buy,bad"},
		{name: "side", change: "5000.0,hold,83"},
	} {
		err := ex.processFuturesOrderbookIncrement(t.Context(), nil, []byte(`{"sequence":19,"change":"`+tc.change+`","timestamp":1551770400000}`), "XBTUSDTM")
		require.ErrorIsf(t, err, errMalformedOrderbookChange, "processFuturesOrderbookIncrement must reject a change with a bad %s", tc.name)
	}
	require.Error(t, ex.processFuturesOrderbookIncrement(t.Context(), nil, []byte(`{"sequence":`), "XBTUSDTM"), "processFuturesOrderbookIncrement must reject an undecodable push")
	require.ErrorIs(t, ex.processFuturesOrderbookIncrement(t.Context(), nil, []byte(`{}`), "NOTACONTRACT"), currency.ErrPairNotFound, "processFuturesOrderbookIncrement must reject an unknown contract")
}

func TestProcessFuturesKline(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*kline.Item](t, nil, &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Futures, Pairs: currency.Pairs{futuresLivePair}, Interval: kline.OneMin})
		require.Len(t, got.Candles, 1, "a candle push must relay one candle")
		assert.Positive(t, got.Candles[0].Close, "a live candle should carry a close price")
		return
	}
	ex := wsTestInstance(t)
	exp := &kline.Item{
		Asset:    asset.Futures,
		Exchange: ex.Name,
		Pair:     futuresTestPair,
		Interval: kline.OneMin,
		Candles:  []kline.Candle{{Time: time.Unix(1791899520, 0), Open: 83259.3, Close: 83258.3, High: 83260.1, Low: 83238.1, Volume: 104}},
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsFutures.json", "klines"), "processFuturesKline should relay the candle with the traded contracts as its volume")

	data := wsFixtureData(t, "wsFutures.json", "klines")
	require.ErrorIs(t, ex.processFuturesKline(t.Context(), data, "XBTUSDTM"), errMalformedCandleTopic, "processFuturesKline must reject a topic without an interval")
	require.ErrorIs(t, ex.processFuturesKline(t.Context(), data, "XBTUSDTM_2min"), kline.ErrInvalidInterval, "processFuturesKline must reject an unknown interval")
	require.ErrorIs(t, ex.processFuturesKline(t.Context(), data, "NOTACONTRACT_1min"), currency.ErrPairNotFound, "processFuturesKline must reject an unknown contract")
	require.Error(t, ex.processFuturesKline(t.Context(), []byte(`{"candles":{}}`), "XBTUSDTM_1min"), "processFuturesKline must reject an undecodable push")
}

func TestProcessFuturesExecution(t *testing.T) {
	t.Parallel()
	enableTradeFeed := func(ex *Exchange) {
		ex.SetTradeFeedStatus(true)
		ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	}
	if !mockTests {
		got := liveWsData[[]trade.Data](t, enableTradeFeed, &subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.Futures, Pairs: currency.Pairs{futuresLivePair}})
		require.NotEmpty(t, got, "the trade topic must relay a trade")
		assert.Positive(t, got[0].Price, "a live trade should carry a price")
		return
	}
	ex := wsTestInstance(t)
	require.NoError(t, ex.processFuturesExecution(wsFixtureData(t, "wsFutures.json", "trade")), "processFuturesExecution must not error without a trade feed")
	assert.Empty(t, ex.Websocket.DataHandler.C, "processFuturesExecution should relay nothing without a trade feed")
	enableTradeFeed(ex)
	exp := []trade.Data{{
		TID:          "1947310382240",
		Exchange:     ex.Name,
		CurrencyPair: futuresTestPair,
		AssetType:    asset.Futures,
		Side:         order.Buy,
		Price:        83258.3,
		Amount:       12,
		Timestamp:    time.UnixMilli(1791899571745),
	}}
	assert.Equal(t, exp, wsPush(t, ex, "wsFutures.json", "trade"), "processFuturesExecution should relay the trade")

	require.Error(t, ex.processFuturesExecution([]byte(`{"size":-1}`)), "processFuturesExecution must reject an undecodable push")
	require.ErrorIs(t, ex.processFuturesExecution([]byte(`{"symbol":"NOTACONTRACT"}`)), currency.ErrPairNotFound, "processFuturesExecution must reject an unknown contract")
	require.ErrorIs(t, ex.processFuturesExecution([]byte(`{"symbol":"XBTUSDTM","side":"sideways"}`)), order.ErrSideIsInvalid, "processFuturesExecution must reject an unknown side")
}

func TestProcessFuturesInstrument(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*WebsocketFuturesMarkIndexPrice](t, nil, &subscription.Subscription{Channel: futuresInstrumentChannel, Asset: asset.Futures, Pairs: currency.Pairs{futuresLivePair}})
		assert.Equal(t, "SOLUSDTM", got.Symbol, "the mark price should name the topic's symbol")
		assert.Positive(t, got.MarkPrice, "the live mark price should be positive")
		return
	}
	ex := wsTestInstance(t)
	exp := &WebsocketFuturesMarkIndexPrice{
		Symbol:      "XBTUSDTM",
		MarkPrice:   83262.36,
		IndexPrice:  83300.44,
		Granularity: 1000,
		Timestamp:   types.Time(time.Unix(1791899572, 0)),
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsFutures.json", "markIndexPrice"), "a mark.index.price push should be relayed with the topic's symbol")
	expFunding := &WebsocketFuturesFundingRate{
		Symbol:      "XBTUSDTM",
		Period:      1,
		Granularity: 60000,
		FundingRate: 0.000055,
		Timestamp:   types.Time(time.Unix(1791899580, 0)),
	}
	assert.Equal(t, expFunding, wsPush(t, ex, "wsFutures.json", "fundingRate"), "a funding.rate push should be relayed with the topic's symbol")

	require.ErrorIs(t, ex.processFuturesInstrument(t.Context(), &WebsocketMessage{Topic: futuresInstrumentChannel + ":XBTUSDTM", Subject: "premium.index"}, "XBTUSDTM"), errUnhandledSubject, "processFuturesInstrument must reject an unknown subject")
	require.Error(t, ex.processFuturesInstrument(t.Context(), &WebsocketMessage{Subject: "funding.rate", Data: []byte(`{"granularity":-1}`)}, "XBTUSDTM"), "processFuturesInstrument must reject an undecodable push")
}

func TestProcessFuturesAnnouncement(t *testing.T) {
	t.Parallel()
	if !mockTests {
		// Settlements happen every 8 hours, so live only proves the topic is served
		liveWsSubscribe(t, &subscription.Subscription{Channel: futuresAnnouncementChannel, Asset: asset.Futures})
		return
	}
	ex := wsTestInstance(t)
	assert.Equal(t, &WebsocketFuturesFundingBegin{
		Symbol:      "XBTUSDTM",
		FundingTime: types.Time(time.Unix(1551770400, 0)),
		FundingRate: -0.002966,
		Timestamp:   types.Time(time.Unix(1551770400, 0)),
	}, wsPush(t, ex, "wsFutures.json", "fundingBegin"), "a funding.begin push should be relayed as it comes")
	assert.Equal(t, &WebsocketFuturesFundingEnd{
		Symbol:      "XBTUSDTM",
		FundingTime: types.Time(time.Unix(1551770400, 0)),
		FundingRate: -0.002966,
		Timestamp:   types.Time(time.Unix(1551770410, 0)),
	}, wsPush(t, ex, "wsFutures.json", "fundingEnd"), "a funding.end push should be relayed as it comes")
	require.ErrorIs(t, ex.processFuturesAnnouncement(t.Context(), &WebsocketMessage{Topic: futuresAnnouncementChannel, Subject: "funding.delay"}), errUnhandledSubject, "processFuturesAnnouncement must reject an unknown subject")
}

func TestProcessFuturesOrderChange(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: futuresTradeOrderChannel, Asset: asset.Futures, Authenticated: true})
		return
	}
	ex := wsTestInstance(t)
	for _, tc := range []struct {
		key string
		exp *order.Detail
	}{
		{key: "orderOpen", exp: &order.Detail{
			Price: 91670, Amount: 1, RemainingAmount: 1, Exchange: ex.Name, OrderID: "247899236673269761", ClientOrderID: "5c52e11203aa677f33e493fb",
			Type: order.Limit, Side: order.Buy, Status: order.Open, AssetType: asset.Futures, MarginType: margin.Isolated,
			Date: time.Unix(0, 1731916985768138917), LastUpdated: time.UnixMilli(1731916985789), Pair: futuresTestPair,
		}},
		{key: "orderUpdate", exp: &order.Detail{
			Price: 5.029, Amount: 19982, ExecutedAmount: 7327, RemainingAmount: 11618, Exchange: ex.Name, OrderID: "228685469427204099", ClientOrderID: "10496pp066R679264",
			Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, AssetType: asset.Futures, MarginType: margin.Isolated,
			Date: time.Unix(0, 1727336066682194084), LastUpdated: time.UnixMilli(1727336066766), Pair: futuresTestPair,
		}},
		{key: "orderMatch", exp: &order.Detail{
			Price: 84603.44, Amount: 3840, ExecutedAmount: 1116, RemainingAmount: 2724, Exchange: ex.Name, OrderID: "440761625608192",
			Type: order.Limit, Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.Futures, MarginType: margin.Multi,
			Date: time.UnixMilli(1743146786640), LastUpdated: time.UnixMilli(1743146786746), Pair: futuresTestPair,
			Trades: []order.TradeHistory{{Price: 85739.69, Amount: 1000, Exchange: ex.Name, TID: "1740800012709", Type: order.Limit, Side: order.Sell, Timestamp: time.UnixMilli(1743146786746)}},
		}},
		{key: "orderFilled", exp: &order.Detail{
			Price: 76000, Amount: 1, ExecutedAmount: 1, Exchange: ex.Name, OrderID: "435278894182961153", ClientOrderID: "5c52e11203aa677f33e493fb",
			Type: order.Limit, Side: order.Buy, Status: order.Filled, AssetType: asset.Futures, MarginType: margin.Multi,
			Date: time.Unix(0, 1776591777464496313), LastUpdated: time.UnixMilli(1776591777474), Pair: futuresTestPair,
		}},
		{key: "orderCanceled", exp: &order.Detail{
			Price: 90000, Amount: 1, Exchange: ex.Name, OrderID: "247901211536203776",
			Type: order.Limit, Side: order.Buy, Status: order.Cancelled, AssetType: asset.Futures, MarginType: margin.Isolated,
			Date: time.Unix(0, 1731917456611809239), LastUpdated: time.UnixMilli(1731917460806), Pair: futuresTestPair,
		}},
	} {
		assert.Equalf(t, tc.exp, wsPush(t, ex, "wsFutures.json", tc.key), "processFuturesOrderChange should relay %s as an order update", tc.key)
	}

	// The docs' open push, of an order resting on the book, carries no order type; a stored order keeps the one it has
	docsOpen := []byte(`{"symbol":"XBTUSDTM","side":"buy","canceledSize":"0","orderId":"247899236673269761","liquidity":"maker","marginMode":"ISOLATED","type":"open","orderTime":1731916985768138917,"size":"1","filledSize":"0","price":"91670","remainSize":"1","clientOid":"5c52e11203aa677f33e493fb","status":"open","ts":1731916985789000000}`)
	require.NoError(t, ex.processFuturesOrderChange(t.Context(), docsOpen), "processFuturesOrderChange must not error for an open push without an order type")
	got, ok := (<-ex.Websocket.DataHandler.C).Data.(*order.Detail)
	require.True(t, ok, "processFuturesOrderChange must relay an order detail")
	exp := &order.Detail{
		Price: 91670, Amount: 1, RemainingAmount: 1, Exchange: ex.Name, OrderID: "247899236673269761", ClientOrderID: "5c52e11203aa677f33e493fb",
		Type: order.UnknownType, Side: order.Buy, Status: order.Open, AssetType: asset.Futures, MarginType: margin.Isolated,
		Date: time.Unix(0, 1731916985768138917), LastUpdated: time.UnixMilli(1731916985789), Pair: futuresTestPair,
	}
	assert.Equal(t, exp, got, "processFuturesOrderChange should relay an open push without an order type with the type unknown")
	stored := &order.Detail{Exchange: ex.Name, OrderID: "247899236673269761", Type: order.StopLimit, Side: order.Buy, Status: order.New, AssetType: asset.Futures, Pair: futuresTestPair, Amount: 1}
	require.NoError(t, stored.UpdateOrderFromDetail(got), "UpdateOrderFromDetail must not error")
	assert.Equal(t, order.StopLimit, stored.Type, "an open push without an order type should leave the stored order's type")

	require.Error(t, ex.processFuturesOrderChange(t.Context(), []byte(`{"size":[]}`)), "processFuturesOrderChange must reject an undecodable push")
	require.ErrorIs(t, ex.processFuturesOrderChange(t.Context(), []byte(`{"symbol":"NOTACONTRACT"}`)), currency.ErrPairNotFound, "processFuturesOrderChange must reject an unknown contract")
	require.ErrorIs(t, ex.processFuturesOrderChange(t.Context(), []byte(`{"symbol":"XBTUSDTM","orderType":"iceberg"}`)), order.ErrUnrecognisedOrderType, "processFuturesOrderChange must reject an unknown order type")
	require.ErrorIs(t, ex.processFuturesOrderChange(t.Context(), []byte(`{"symbol":"XBTUSDTM","orderType":"limit","side":"sideways"}`)), order.ErrSideIsInvalid, "processFuturesOrderChange must reject an unknown side")
	require.ErrorIs(t, ex.processFuturesOrderChange(t.Context(), []byte(`{"symbol":"XBTUSDTM","orderType":"limit","side":"buy","type":"expired"}`)), errUnknownOrderChangeType, "processFuturesOrderChange must reject an unknown message type")
	require.ErrorIs(t, ex.processFuturesOrderChange(t.Context(), []byte(`{"symbol":"XBTUSDTM","orderType":"limit","side":"buy","type":"open","marginMode":"PORTFOLIO"}`)), errUnknownMarginMode, "processFuturesOrderChange must reject an unknown margin mode")
}

// TestProcessFuturesOrderChangeStopOrderType checks an Orders push gives a stop order the type REST and its stop pushes
// give it, as KuCoin follows a triggered stop order on the Orders topic with only the limit or market type it places
func TestProcessFuturesOrderChangeStopOrderType(t *testing.T) {
	t.Parallel()
	relay := func(t *testing.T, ex *Exchange, push string, stop bool) *order.Detail {
		t.Helper()
		if stop {
			require.NoError(t, ex.processFuturesStopOrder(t.Context(), []byte(push)), "processFuturesStopOrder must not error")
		} else {
			require.NoError(t, ex.processFuturesOrderChange(t.Context(), []byte(push)), "processFuturesOrderChange must not error")
		}
		d, ok := relayed(t, ex).(*order.Detail)
		require.True(t, ok, "the push must be relayed as an order detail")
		return d
	}
	// A stop push without an order price is for a stop market order
	stopPush := func(id, price, side, stop, event string) string {
		return `{"marginMode":"ISOLATED","orderId":"` + id + `","orderPrice":"` + price + `","orderType":"stop","side":"` + side + `","size":3,"stop":"` + stop + `","stopPrice":"65500","stopPriceType":"TP","symbol":"XBTUSDTM","type":"` + event + `","createdAt":1730194206837,"ts":1730194206843133000}`
	}
	// An Orders push without an order type leaves out the field, as the docs' open push does
	orderPush := func(id, orderType, side, event, status string) string {
		if orderType != "" {
			orderType = `"orderType":"` + orderType + `",`
		}
		return `{"symbol":"XBTUSDTM",` + orderType + `"side":"` + side + `","canceledSize":"0","orderId":"` + id + `","marginMode":"ISOLATED","type":"` + event + `","orderTime":1731916985768138917,"size":"3","filledSize":"1","price":"65000.1","matchPrice":"65000.1","matchSize":"1","remainSize":"2","tradeId":"1794175373644","status":"` + status + `","ts":1731916996762000000}`
	}
	remembered := func(ex *Exchange) []any {
		var ids []any
		ex.futuresStopOrderTypes.Range(func(id, _ any) bool {
			ids = append(ids, id)
			return true
		})
		return ids
	}

	t.Run("triggered stop order", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		assert.Equal(t, order.TakeProfit, relay(t, ex, stopPush("tp", "65000.1", "buy", "down", "open"), true).Type, "processFuturesStopOrder should relay a buy triggered down as a take profit order")
		assert.Equal(t, order.TakeProfit, relay(t, ex, stopPush("tp", "65000.1", "buy", "down", "triggered"), true).Type, "processFuturesStopOrder should relay the triggered take profit order as a take profit order")
		assert.Equal(t, order.TakeProfit, relay(t, ex, orderPush("tp", "", "buy", "open", "open"), false).Type, "processFuturesOrderChange should relay an open push without an order type with the stop order's type")
		match := relay(t, ex, orderPush("tp", "limit", "buy", "match", "open"), false)
		assert.Equal(t, order.TakeProfit, match.Type, "processFuturesOrderChange should relay a triggered take profit order's match with its stop push's type")
		require.Len(t, match.Trades, 1, "processFuturesOrderChange must relay the match's trade")
		assert.Equal(t, order.TakeProfit, match.Trades[0].Type, "processFuturesOrderChange should give the match's trade the order's type")
		assert.Equal(t, order.TakeProfit, relay(t, ex, orderPush("tp", "limit", "buy", "match", "open"), false).Type, "processFuturesOrderChange should relay each later match with the stop order's type")
		rest, err := ex.futuresOrderDetail(&FuturesOrderDetailResponse{ID: "tp", Symbol: "XBTUSDTM", Type: "limit", Side: "buy", Stop: "down", StopTriggered: true, Size: 3, DealSize: 1, IsActive: true})
		require.NoError(t, err, "futuresOrderDetail must not error")
		assert.Equal(t, order.TakeProfit, rest.Type, "futuresOrderDetail should report the triggered take profit order as a take profit order")
		assert.Equal(t, order.TakeProfit, relay(t, ex, orderPush("tp", "limit", "buy", "filled", "done"), false).Type, "processFuturesOrderChange should relay the fill of a triggered take profit order with its type")
		assert.Empty(t, remembered(ex), "processFuturesOrderChange should forget a stop order once it is done")
	})

	t.Run("order pushes before the triggered pushes", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		relay(t, ex, stopPush("take-profit", "65000.1", "buy", "down", "open"), true)
		relay(t, ex, stopPush("stop-loss", "65000.1", "sell", "down", "open"), true)
		assert.Equal(t, order.TakeProfit, relay(t, ex, orderPush("take-profit", "limit", "buy", "match", "open"), false).Type, "processFuturesOrderChange should relay a stop order's match with its stop push's type before the triggered push")
		assert.Equal(t, order.StopLimit, relay(t, ex, orderPush("stop-loss", "limit", "sell", "match", "open"), false).Type, "processFuturesOrderChange should relay each stop order's match with its own stop push's type")
	})

	t.Run("stop orders REST lists", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		untriggered, err := ex.futuresOrderDetail(&FuturesOrderDetailResponse{ID: "untriggered", Symbol: "XBTUSDTM", Type: "limit", Side: "buy", Stop: "down", Size: 3, IsActive: true})
		require.NoError(t, err, "futuresOrderDetail must not error")
		require.Equal(t, order.TakeProfit, untriggered.Type, "futuresOrderDetail must report a buy triggered down as a take profit order")
		triggered, err := ex.futuresOrderDetail(&FuturesOrderDetailResponse{ID: "triggered", Symbol: "XBTUSDTM", Type: "market", Side: "sell", Stop: "up", StopTriggered: true, Size: 3, DealSize: 1, IsActive: true})
		require.NoError(t, err, "futuresOrderDetail must not error")
		require.Equal(t, order.TakeProfitMarket, triggered.Type, "futuresOrderDetail must report a sell market order triggered up as a take profit market order")
		assert.Equal(t, order.TakeProfit, relay(t, ex, orderPush("untriggered", "limit", "buy", "match", "open"), false).Type, "processFuturesOrderChange should relay an order REST listed as an untriggered stop order with its stop type")
		assert.Equal(t, order.TakeProfitMarket, relay(t, ex, orderPush("triggered", "market", "sell", "match", "open"), false).Type, "processFuturesOrderChange should relay a market order REST listed as a triggered stop order with its stop type")
		_, err = ex.futuresOrderDetail(&FuturesOrderDetailResponse{ID: "untriggered", Symbol: "XBTUSDTM", Type: "limit", Side: "buy", Stop: "down", StopTriggered: true, Size: 3, DealSize: 3})
		require.NoError(t, err, "futuresOrderDetail must not error")
		assert.ElementsMatch(t, []any{"triggered"}, remembered(ex), "futuresOrderDetail should forget the stop order REST reports finished, and that alone")
		assert.Equal(t, order.TakeProfitMarket, relay(t, ex, orderPush("triggered", "market", "sell", "match", "open"), false).Type, "processFuturesOrderChange should still relay the other stop order with its stop type")
		_, err = ex.futuresOrderDetail(&FuturesOrderDetailResponse{ID: "triggered", Symbol: "XBTUSDTM", Type: "market", Side: "sell", Size: 3, DealSize: 3})
		require.NoError(t, err, "futuresOrderDetail must not error")
		assert.Empty(t, remembered(ex), "futuresOrderDetail should forget an order REST reports finished without its stop condition too")
	})

	t.Run("stop order REST reads as just ended before its done push", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		relay(t, ex, stopPush("tp", "65000.1", "buy", "down", "triggered"), true)
		_, err := ex.futuresOrderDetail(&FuturesOrderDetailResponse{ID: "tp", Symbol: "XBTUSDTM", Type: "limit", Side: "buy", Stop: "down", StopTriggered: true, Size: 3, DealSize: 3, EndAt: types.Time(time.Now())})
		require.NoError(t, err, "futuresOrderDetail must not error")
		assert.Equal(t, order.TakeProfit, relay(t, ex, orderPush("tp", "limit", "buy", "filled", "done"), false).Type, "processFuturesOrderChange should relay the done push of a stop order REST read as just ended with its stop type")
		assert.Empty(t, remembered(ex), "processFuturesOrderChange should forget the stop order once it is done")
	})

	t.Run("stop order cancelled before it triggers", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		relay(t, ex, stopPush("cancelled", "65000.1", "sell", "up", "open"), true)
		assert.Equal(t, order.Cancelled, relay(t, ex, stopPush("cancelled", "65000.1", "sell", "up", "cancel"), true).Status, "processFuturesStopOrder should relay a cancelled stop order as cancelled")
		assert.Empty(t, remembered(ex), "processFuturesStopOrder should forget a cancelled stop order")
		assert.Equal(t, order.Limit, relay(t, ex, orderPush("cancelled", "limit", "sell", "match", "open"), false).Type, "processFuturesOrderChange should relay an order no longer known as a stop order with its own type")
	})

	t.Run("order placed elsewhere", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		assert.Equal(t, order.Limit, relay(t, ex, orderPush("plain", "limit", "buy", "match", "open"), false).Type, "processFuturesOrderChange should relay an order not known as a stop order with its own type")
		assert.Empty(t, remembered(ex), "processFuturesOrderChange should remember no type for an order not known as a stop order")
	})
}

func TestFuturesMarginType(t *testing.T) {
	t.Parallel()
	for mode, exp := range map[string]margin.Type{"ISOLATED": margin.Isolated, "CROSS": margin.Multi, "": margin.Unset} {
		got, err := futuresMarginType(mode)
		require.NoErrorf(t, err, "futuresMarginType must not error for %q", mode)
		assert.Equalf(t, exp, got, "futuresMarginType should map %q", mode)
	}
	_, err := futuresMarginType("PORTFOLIO")
	require.ErrorIs(t, err, errUnknownMarginMode, "futuresMarginType must reject an unknown margin mode")
}

func TestProcessFuturesWallet(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: futuresAccountBalanceEventChannel, Asset: asset.Futures, Authenticated: true})
		return
	}
	ex := wsTestInstance(t)
	for _, tc := range []struct {
		key                    string
		code                   currency.Code
		total, hold, available float64
		updatedAt              time.Time
	}{
		{key: "walletBalance", code: currency.USDT, total: 387.224858816, hold: 17.551016 + 10.06002012 + 63.98342359 + 1.2 + 2.3, available: 285.652001096, updatedAt: time.UnixMilli(1741164936624)},
		{key: "availableBalance", code: currency.XBT, total: 5923 + 2312, hold: 2312, available: 5923, updatedAt: time.UnixMilli(1553842862614)},
	} {
		exp := accounts.SubAccounts{accounts.NewSubAccount(asset.Futures, "")}
		exp[0].Balances.Set(tc.code, accounts.Balance{Total: tc.total, Hold: tc.hold, Free: tc.available, UpdatedAt: tc.updatedAt})
		assert.Equalf(t, exp, wsPush(t, ex, "wsFutures.json", tc.key), "a %s push should be saved as the futures balance", tc.key)
	}
	// The docs' walletBalance.change example
	docs := []byte(`{"topic":"/contractAccount/wallet","type":"message","subject":"walletBalance.change","id":"67c811885b87be0001a4880e","userId":"633559791e1cbc0001f319bc","channelType":"private","data":{"crossPosMargin":"17.551016","isolatedOrderMargin":"0","holdBalance":"0","equity":"387.224858816","version":"2118","availableBalance":"285.652001096","isolatedPosMargin":"63.98342359","maxWithdrawAmount":"285.645841096","walletBalance":"371.394298816","isolatedFundingFeeMargin":"2.89220199","crossUnPnl":"2.95996","totalCrossMargin":"310.370835226","currency":"USDT","isolatedUnPnl":"12.8706","crossOrderMargin":"10.06002012","timestamp":"1741164936624"}}`)
	require.NoError(t, ex.wsHandleData(t.Context(), nil, docs), "wsHandleData must not error for the docs' walletBalance.change example")
	exp := accounts.SubAccounts{accounts.NewSubAccount(asset.Futures, "")}
	exp[0].Balances.Set(currency.USDT, accounts.Balance{Total: 387.224858816, Hold: 17.551016 + 10.06002012 + 63.98342359, Free: 285.652001096, UpdatedAt: time.UnixMilli(1741164936624)})
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "a walletBalance.change push should be saved with its equity as the total")
	assert.Equal(t, &WebsocketFuturesOrderMargin{OrderMargin: 5923, Currency: currency.USDT, Timestamp: types.Time(time.UnixMilli(1553842862614))}, wsPush(t, ex, "wsFutures.json", "orderMargin"), "an orderMargin.change push should be relayed as it comes")
	assert.Equal(t, &WebsocketFuturesWithdrawHold{WithdrawHold: 5923, Currency: currency.USDT, Timestamp: types.Time(time.UnixMilli(1553842862614))}, wsPush(t, ex, "wsFutures.json", "withdrawHold"), "a withdrawHold.change push should be relayed as it comes")

	require.ErrorIs(t, ex.processFuturesWallet(t.Context(), &WebsocketMessage{Topic: futuresAccountBalanceEventChannel, Subject: "bonus.change"}), errUnhandledSubject, "processFuturesWallet must reject an unknown subject")
	for _, subject := range []string{"walletBalance.change", "availableBalance.change"} {
		require.Errorf(t, ex.processFuturesWallet(t.Context(), &WebsocketMessage{Subject: subject, Data: []byte(`{"currency":[]}`)}), "processFuturesWallet must reject an undecodable %s push", subject)
	}
}

func TestSaveFuturesBalance(t *testing.T) {
	t.Parallel()
	ex := accountTestInstance(t)
	require.NoError(t, ex.saveFuturesBalance(t.Context(), currency.USDT, 4, 1, 3, time.UnixMilli(1741164936624)), "saveFuturesBalance must not error")
	exp := accounts.SubAccounts{accounts.NewSubAccount(asset.Futures, "")}
	exp[0].Balances.Set(currency.USDT, accounts.Balance{Total: 4, Hold: 1, Free: 3, UpdatedAt: time.UnixMilli(1741164936624)})
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "saveFuturesBalance should relay the balance it saves")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.GetBalance("", creds, asset.Futures, currency.USDT)
	require.NoError(t, err, "the futures balance must be stored")
	assert.Equal(t, 4.0, stored.Total, "the stored total should be the one saved")
	ex.Accounts = nil
	require.ErrorIs(t, ex.saveFuturesBalance(t.Context(), currency.USDT, 4, 1, 3, time.Now()), common.ErrNilPointer, "saveFuturesBalance must return a save error")
}

func TestProcessFuturesPosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: futuresPositionAllChannel, Asset: asset.Futures, Authenticated: true}, &subscription.Subscription{Channel: futuresPositionChannel, Asset: asset.Futures, Pairs: currency.Pairs{futuresLivePair}, Authenticated: true})
		return
	}
	ex := wsTestInstance(t)
	exp := &WebsocketFuturesPosition{
		Symbol:                       "XBTUSDTM",
		MaintenanceMarginRequirement: 0.004,
		RiskLimit:                    50000000,
		RealLeverage:                 19.1376874933,
		DeleveragePercentage:         0.87,
		OpeningTimestamp:             types.Time(time.UnixMilli(1771400783360)),
		AutoDeposit:                  true,
		CurrentTimestamp:             types.Time(time.UnixMilli(1771474169458)),
		CurrentQuantity:              -1,
		CurrentCost:                  -68.0942,
		CurrentCommission:            0.03794569,
		UnrealisedCost:               -68.0942,
		RealisedCost:                 0.03794569,
		IsOpen:                       true,
		MarkPrice:                    66954.6,
		MarkValue:                    -66.9546,
		PositionCost:                 -68.0942,
		PositionCross:                1,
		PositionInitial:              1.361884,
		PositionCommission:           0.0013,
		PositionLoss:                 0.00291083,
		PositionMargin:               2.35897317,
		PositionFunding:              -0.00291083,
		PositionMaintenance:          0.30799116,
		MaintenanceMargin:            3.49857317,
		AverageEntryPrice:            68094.2,
		LiquidationPrice:             70130.5725363,
		BankruptPrice:                70453.17317,
		SettleCurrency:               currency.USDT,
		ChangeReason:                 "changeRiskLimit",
		RiskLimitLevel:               5,
		RealisedGrossCost:            0.1639,
		RealisedGrossPNL:             -0.1639,
		RealisedPNL:                  -0.04376735,
		UnrealisedPNL:                1.1396,
		UnrealisedPNLPercentage:      0.0167,
		UnrealisedROEPercentage:      0.8368,
		Leverage:                     19.1376874933,
		MarginMode:                   "ISOLATED",
		PositionSide:                 "SHORT",
		Tax:                          0.0001,
		DealCommission:               -0.04085652,
		FundingFee:                   -0.00291083,
		AggregateRate:                0.0046,
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsFutures.json", "positionChange"), "a position.change push should be relayed as it comes")
	exp = &WebsocketFuturesPosition{
		Symbol:                  "XBTUSDTM",
		CrossMode:               true,
		DeleveragePercentage:    0.06,
		OpeningTimestamp:        types.Time(time.UnixMilli(1717639498983)),
		CurrentTimestamp:        types.Time(time.UnixMilli(1717724686618)),
		CurrentQuantity:         -2,
		CurrentCost:             -136.002,
		CurrentCommission:       0.06739824,
		UnrealisedCost:          -136.002,
		RealisedCost:            0.06739824,
		IsOpen:                  true,
		MarkPrice:               70778.04,
		MarkValue:               -141.55608,
		PositionCost:            -136.002,
		PositionInitial:         5.4509819612,
		PositionMargin:          5.6735903779,
		AverageEntryPrice:       68001,
		LiquidationPrice:        80700.49720065,
		BankruptPrice:           81152.42267235,
		SettleCurrency:          currency.USDT,
		ChangeReason:            "positionChange",
		RealisedPNL:             -0.09580416,
		UnrealisedPNL:           -5.55408,
		UnrealisedPNLPercentage: -0.0408,
		UnrealisedROEPercentage: -1.0189,
		Leverage:                24.95,
		MarginMode:              "CROSS",
		PositionSide:            "BOTH",
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsFutures.json", "positionChangeCross"), "a cross margin position.change push should be relayed as it comes")
	assert.Equal(t, &WebsocketFuturesPositionSettlement{
		Symbol:         "XBTUSDTM",
		MarkPrice:      67198.3,
		Quantity:       -2,
		PositionSide:   "SHORT",
		SettleCurrency: currency.USDT,
		FundingTime:    types.Time(time.UnixMilli(1771488000000)),
		MarginMode:     "ISOLATED",
		FundingFee:     -0.00309113,
		FundingRate:    -0.000023,
		Timestamp:      types.Time(time.Unix(0, 1771488018495030863)),
	}, wsPush(t, ex, "wsFutures.json", "positionSettlement"), "a position.settlement push should be relayed as it comes")
	assert.Equal(t, &WebsocketFuturesRiskLimitAdjustment{Message: "Insufficient balance. Cannot increase margin.", Success: true, RiskLimitLevel: 9}, wsPush(t, ex, "wsFutures.json", "positionAdjustRiskLimit"), "a position.adjustRiskLimit push should be relayed as it comes")
	require.ErrorIs(t, ex.processFuturesPosition(t.Context(), &WebsocketMessage{Topic: futuresPositionAllChannel, Subject: "position.unknown"}), errUnhandledSubject, "processFuturesPosition must reject an unknown subject")
}

func TestProcessFuturesMarginMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: futuresMarginModeChannel, Asset: asset.Futures, Authenticated: true})
		return
	}
	exp := &WebsocketFuturesMarginMode{"XBTUSDTM": "CROSS", "ETHUSDTM": "ISOLATED"}
	assert.Equal(t, exp, wsPush(t, wsTestInstance(t), "wsFutures.json", "marginMode"), "a Margin Mode push should be relayed as it comes")
}

func TestProcessFuturesCrossLeverage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: futuresCrossLeverageChannel, Asset: asset.Futures, Authenticated: true})
		return
	}
	exp := &WebsocketFuturesCrossLeverage{"XBTUSDTM": {Leverage: 51}}
	assert.Equal(t, exp, wsPush(t, wsTestInstance(t), "wsFutures.json", "crossLeverage"), "a Cross Margin Leverage push should be relayed as it comes")
}

func TestProcessFuturesStopOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: futuresStopOrdersLifecycleEventChannel, Asset: asset.Futures, Authenticated: true})
		return
	}
	ex := wsTestInstance(t)
	exp := &order.Detail{
		Price:        0.1,
		TriggerPrice: 1000,
		Amount:       1,
		Exchange:     ex.Name,
		OrderID:      "240673378116083712",
		Type:         order.TakeProfit,
		Side:         order.Buy,
		Status:       order.Open,
		AssetType:    asset.Futures,
		MarginType:   margin.Isolated,
		Date:         time.UnixMilli(1730194206837),
		LastUpdated:  time.UnixMicro(1730194206843133),
		Pair:         futuresTestPair,
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsFutures.json", "stopOrder"), "processFuturesStopOrder should relay the push as an order update")
	for _, tc := range []struct {
		name, orderPrice string
	}{
		{name: "without an order price"},
		{name: "with an empty order price", orderPrice: `"orderPrice":"",`},
		{name: "with a zero order price", orderPrice: `"orderPrice":"0",`},
	} {
		push := `{"marginMode":"ISOLATED","orderId":"240673378116083712",` + tc.orderPrice + `"orderType":"stop","side":"buy","size":1,"stop":"down","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"open","createdAt":1730194206837,"ts":1730194206843133000}`
		require.NoErrorf(t, ex.processFuturesStopOrder(t.Context(), []byte(push)), "processFuturesStopOrder must not error for a stop order push %s", tc.name)
		got := *exp
		got.Price, got.Type = 0, order.TakeProfitMarket
		assert.Equalf(t, &got, (<-ex.Websocket.DataHandler.C).Data, "processFuturesStopOrder should relay a take profit order push %s as a take profit market order", tc.name)
	}
	// The docs' example is a buy triggered as the price falls, a take profit order; an order triggered as the price moves
	// against its side is a stop order
	for _, tc := range []struct {
		name, event, side, stop, orderPrice string
		oSide                               order.Side
		price                               float64
		exp                                 order.Type
	}{
		{name: "sell triggered as the price falls, with an order price", event: "open", side: "sell", stop: "down", orderPrice: `"orderPrice":"0.1",`, oSide: order.Sell, price: 0.1, exp: order.StopLimit},
		{name: "sell triggered as the price falls, without an order price", event: "open", side: "sell", stop: "down", oSide: order.Sell, exp: order.StopMarket},
		{name: "sell triggered as the price rises, with an order price", event: "open", side: "sell", stop: "up", orderPrice: `"orderPrice":"0.1",`, oSide: order.Sell, price: 0.1, exp: order.TakeProfit},
		{name: "sell triggered as the price rises, without an order price", event: "open", side: "sell", stop: "up", oSide: order.Sell, exp: order.TakeProfitMarket},
		{name: "buy triggered as the price rises, with an order price", event: "open", side: "buy", stop: "up", orderPrice: `"orderPrice":"0.1",`, oSide: order.Buy, price: 0.1, exp: order.StopLimit},
		{name: "buy triggered as the price rises", event: "open", side: "buy", stop: "up", oSide: order.Buy, exp: order.StopMarket},
		{name: "triggered sell taking profit as the price rises", event: "triggered", side: "sell", stop: "up", orderPrice: `"orderPrice":"0.1",`, oSide: order.Sell, price: 0.1, exp: order.TakeProfit},
		{name: "cancelled buy taking profit as the price falls", event: "cancel", side: "buy", stop: "down", oSide: order.Buy, exp: order.TakeProfitMarket},
	} {
		push := `{"marginMode":"ISOLATED","orderId":"240673378116083712",` + tc.orderPrice + `"orderType":"stop","side":"` + tc.side + `","size":1,"stop":"` + tc.stop + `","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"` + tc.event + `","createdAt":1730194206837,"ts":1730194206843133000}`
		require.NoErrorf(t, ex.processFuturesStopOrder(t.Context(), []byte(push)), "processFuturesStopOrder must not error for a %s", tc.name)
		got, ok := (<-ex.Websocket.DataHandler.C).Data.(*order.Detail)
		require.Truef(t, ok, "processFuturesStopOrder must relay an order detail for a %s", tc.name)
		assert.Equalf(t, tc.exp, got.Type, "processFuturesStopOrder should relay a %s as a %s order", tc.name, tc.exp)
		assert.Equalf(t, tc.oSide, got.Side, "processFuturesStopOrder should relay a %s with its side", tc.name)
		assert.Equalf(t, tc.price, got.Price, "processFuturesStopOrder should relay a %s with its order price", tc.name)
	}
	for _, tc := range []struct {
		name       string
		push       string
		side       order.Side
		status     order.Status
		marginType margin.Type
	}{
		{name: "triggered isolated margin stop order", push: `{"marginMode":"ISOLATED","orderId":"240673378116083712","orderPrice":"0.1","orderType":"stop","side":"buy","size":1,"stop":"down","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"triggered","createdAt":1730194206837,"ts":1730194206843133000}`, side: order.Buy, status: order.Open, marginType: margin.Isolated},
		{name: "triggered cross margin stop order", push: `{"marginMode":"CROSS","orderId":"240673378116083712","orderPrice":"0.1","orderType":"stop","side":"buy","size":1,"stop":"down","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"triggered","createdAt":1730194206837,"ts":1730194206843133000}`, side: order.Buy, status: order.Open, marginType: margin.Multi},
		{name: "triggered stop order without a margin mode", push: `{"orderId":"240673378116083712","orderPrice":"0.1","orderType":"stop","side":"buy","size":1,"stop":"down","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"triggered","createdAt":1730194206837,"ts":1730194206843133000}`, side: order.Buy, status: order.Open, marginType: margin.Unset},
		{name: "cancelled isolated margin stop order", push: `{"marginMode":"ISOLATED","orderId":"240673378116083712","orderPrice":"0.1","orderType":"stop","side":"buy","size":1,"stop":"down","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"cancel","createdAt":1730194206837,"ts":1730194206843133000}`, side: order.Buy, status: order.Cancelled, marginType: margin.Isolated},
		{name: "triggered sell stop order", push: `{"marginMode":"ISOLATED","orderId":"240673378116083712","orderPrice":"0.1","orderType":"stop","side":"sell","size":1,"stop":"up","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"triggered","createdAt":1730194206837,"ts":1730194206843133000}`, side: order.Sell, status: order.Open, marginType: margin.Isolated},
		{name: "triggered sell stop loss order", push: `{"marginMode":"ISOLATED","orderId":"240673378116083712","orderPrice":"0.1","orderType":"stop","side":"sell","size":1,"stop":"down","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"triggered","createdAt":1730194206837,"ts":1730194206843133000}`, side: order.Sell, status: order.Open, marginType: margin.Isolated},
		{name: "triggered cross margin sell stop order", push: `{"marginMode":"CROSS","orderId":"240673378116083712","orderPrice":"0.1","orderType":"stop","side":"sell","size":1,"stop":"down","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"triggered","createdAt":1730194206837,"ts":1730194206843133000}`, side: order.Sell, status: order.Open, marginType: margin.Multi},
		{name: "triggered sell stop order without a margin mode", push: `{"orderId":"240673378116083712","orderPrice":"0.1","orderType":"stop","side":"sell","size":1,"stop":"up","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"triggered","createdAt":1730194206837,"ts":1730194206843133000}`, side: order.Sell, status: order.Open, marginType: margin.Unset},
		{name: "triggered market stop order", push: `{"marginMode":"ISOLATED","orderId":"240673378116083712","orderType":"stop","side":"buy","size":1,"stop":"down","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"triggered","createdAt":1730194206837,"ts":1730194206843133000}`, side: order.Buy, status: order.Open, marginType: margin.Isolated},
		{name: "cancelled sell stop order", push: `{"marginMode":"CROSS","orderId":"240673378116083712","orderPrice":"0.1","orderType":"stop","side":"sell","size":1,"stop":"up","stopPrice":"1000","stopPriceType":"TP","symbol":"XBTUSDTM","type":"cancel","createdAt":1730194206837,"ts":1730194206843133000}`, side: order.Sell, status: order.Cancelled, marginType: margin.Multi},
	} {
		require.NoErrorf(t, ex.processFuturesStopOrder(t.Context(), []byte(tc.push)), "processFuturesStopOrder must not error for a %s", tc.name)
		got, ok := (<-ex.Websocket.DataHandler.C).Data.(*order.Detail)
		require.Truef(t, ok, "processFuturesStopOrder must relay an order detail for a %s", tc.name)
		assert.Equalf(t, tc.status, got.Status, "processFuturesStopOrder should relay a %s with status %s", tc.name, tc.status)
		assert.Equalf(t, tc.side, got.Side, "processFuturesStopOrder should relay a %s with its side", tc.name)
		assert.Equalf(t, tc.marginType, got.MarginType, "processFuturesStopOrder should relay a %s with its margin type", tc.name)
	}

	require.Error(t, ex.processFuturesStopOrder(t.Context(), []byte(`{"size":-1}`)), "processFuturesStopOrder must reject an undecodable push")
	require.ErrorIs(t, ex.processFuturesStopOrder(t.Context(), []byte(`{"symbol":"NOTACONTRACT"}`)), currency.ErrPairNotFound, "processFuturesStopOrder must reject an unknown contract")
	require.ErrorIs(t, ex.processFuturesStopOrder(t.Context(), []byte(`{"symbol":"XBTUSDTM","orderType":"iceberg"}`)), order.ErrUnrecognisedOrderType, "processFuturesStopOrder must reject an unknown order type")
	require.ErrorIs(t, ex.processFuturesStopOrder(t.Context(), []byte(`{"symbol":"XBTUSDTM","orderType":"stop","side":"sideways"}`)), order.ErrSideIsInvalid, "processFuturesStopOrder must reject an unknown side")
	require.ErrorIs(t, ex.processFuturesStopOrder(t.Context(), []byte(`{"symbol":"XBTUSDTM","orderType":"stop","side":"buy","type":"expired"}`)), errUnknownStopOrderType, "processFuturesStopOrder must reject an unknown message type")
	require.ErrorIs(t, ex.processFuturesStopOrder(t.Context(), []byte(`{"symbol":"XBTUSDTM","orderType":"stop","side":"buy","type":"open","marginMode":"PORTFOLIO"}`)), errUnknownMarginMode, "processFuturesStopOrder must reject an unknown margin mode")
}
