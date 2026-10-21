package kucoin

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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

// spotTestPair is the pair the spot fixtures record, enabled for spot in the test config
var spotTestPair = currency.NewPairWithDelimiter("BTC", "USDT", "-")

// TestWebsocketSpotPayloads decodes the recorded spot pushes whose fields the handlers map onto shared types
func TestWebsocketSpotPayloads(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		key string
		got any
		exp any
	}{
		{key: "ticker", got: new(WebsocketTicker), exp: &WebsocketTicker{
			Sequence:    38626076751,
			Price:       83296.2,
			Size:        0.00002779,
			BestAsk:     83296.2,
			BestAskSize: 0.00874259,
			BestBid:     83296.1,
			BestBidSize: 0.83775345,
			Time:        types.Time(time.UnixMilli(1791899504669)),
		}},
		{key: "level1", got: new(WebsocketOrderbookLevel1), exp: &WebsocketOrderbookLevel1{
			Asks:      [2]types.Number{83296.2, 0.17744259},
			Bids:      [2]types.Number{83296.1, 0.83775345},
			Timestamp: types.Time(time.UnixMilli(1791899505902)),
		}},
		{key: "level50", got: new(WebsocketOrderbookDepth), exp: &WebsocketOrderbookDepth{
			Asks:      orderbook.LevelsArrayPriceAmount{{Amount: 0.00874259, Price: 83296.2}, {Amount: 0.0085, Price: 83296.3}, {Amount: 0.0085, Price: 83296.4}},
			Bids:      orderbook.LevelsArrayPriceAmount{{Amount: 0.83775345, Price: 83296.1}, {Amount: 0.07352707, Price: 83296}, {Amount: 0.0005, Price: 83295.9}},
			Timestamp: types.Time(time.UnixMilli(1791899505798)),
		}},
		{key: "increment", got: new(WebsocketOrderbookIncrement), exp: &WebsocketOrderbookIncrement{
			Changes: WebsocketOrderbookChanges{
				Asks: [][3]types.Number{{83708.9, 0.07939911, 38627404122}},
				Bids: [][3]types.Number{{83692.5, 0.04827727, 38627404123}},
			},
			SequenceEnd:   38627404123,
			SequenceStart: 38627404122,
			Symbol:        "BTC-USDT",
			Time:          types.Time(time.UnixMilli(1791901473976)),
		}},
		{key: "klines", got: new(WebsocketCandlestick), exp: &WebsocketCandlestick{
			Symbol: "BTC-USDT",
			Candles: WebsocketCandle{
				StartTime:         types.Time(time.Unix(1791899460, 0)),
				OpenPrice:         83310.5,
				ClosePrice:        83296.2,
				HighPrice:         83310.5,
				LowPrice:          83281.5,
				TransactionVolume: 3.86976157,
				TransactionAmount: 322303.932634086,
			},
			Time: types.Time(time.Unix(0, 1791899507203818046)),
		}},
		{key: "trade", got: new(WebsocketTrade), exp: &WebsocketTrade{
			MakerOrderID: "6ace375c8dd1990007ab3d6d",
			Price:        83296.2,
			Sequence:     "24806846522605568",
			Side:         "buy",
			Size:         0.00534199,
			Symbol:       "BTC-USDT",
			TakerOrderID: "499484166153404418",
			Time:         types.Time(time.UnixMilli(1791899507179)),
			TradeID:      "24806846522605568",
			Type:         "match",
		}},
		{key: "symbolSnapshot", got: new(WebsocketSnapshot), exp: &WebsocketSnapshot{
			Sequence: 38626078076,
			Data: WebsocketSnapshotData{
				AskSize:          0.0055,
				AveragePrice:     83541.67284915,
				BaseCurrency:     currency.BTC,
				BidSize:          0.4781167,
				Board:            1,
				Buy:              83299.8,
				ChangePrice:      520.3,
				ChangeRate:       0.0062,
				Close:            83296.2,
				Datetime:         types.Time(time.UnixMilli(1791899507490)),
				High:             84149,
				LastSize:         0.00002779,
				LastTradedPrice:  83296.2,
				Low:              81035.5,
				MakerCoefficient: 1,
				MakerFeeRate:     0.001,
				MarginTrade:      true,
				Mark:             1,
				Market:           "USDS",
				MarketChange1Hour: WebsocketSnapshotMarketChange{
					ChangePrice: -42.2,
					ChangeRate:  -0.0005,
					High:        83678.3,
					Low:         83180.4,
					Open:        83338.4,
					Volume:      396.39534308,
					VolumeValue: 33057188.091801945,
				},
				MarketChange24Hour: WebsocketSnapshotMarketChange{
					ChangePrice: 520.3,
					ChangeRate:  0.0062,
					High:        84149,
					Low:         81035.5,
					Open:        82775.9,
					Volume:      6381.269114425228,
					VolumeValue: 529005967.528381,
				},
				MarketChange4Hour: WebsocketSnapshotMarketChange{
					ChangePrice: -316.6,
					ChangeRate:  -0.0037,
					High:        83738.4,
					Low:         83130.4,
					Open:        83612.8,
					Volume:      978.7129054,
					VolumeValue: 81592661.56820641,
				},
				Markets:          []string{"USDS", "Majors", "PoW", "Layer1"},
				Open:             82775.9,
				QuoteCurrency:    currency.USDT,
				Sell:             83300.3,
				Sequence:         38626078076,
				SiteTypes:        []string{"global"},
				Sort:             100,
				Symbol:           "BTC-USDT",
				SymbolCode:       "BTC-USDT",
				TakerCoefficient: 1,
				TakerFeeRate:     0.001,
				Trading:          true,
				Volume:           6381.269114425228,
				VolumeValue:      529005967.528381,
			},
		}},
		{key: "orderV2Match", got: new(WebsocketSpotOrderChange), exp: &WebsocketSpotOrderChange{
			ClientOrderID: "5c52e11203aa677f33e493fc",
			FeeType:       "takerFee",
			FilledSize:    0.00001,
			Liquidity:     "taker",
			MatchPrice:    71171.9,
			MatchSize:     0.00001,
			OrderID:       "6720da3fa30a360007f5f832",
			OrderTime:     types.Time(time.UnixMilli(1730206271588)),
			OrderType:     "market",
			OriginSize:    0.00001,
			Side:          "buy",
			Size:          0.00001,
			Status:        "match",
			Symbol:        "BTC-USDT",
			TradeID:       "11116472408358913",
			Timestamp:     types.Time(time.UnixMilli(1730206271616)),
			Type:          "match",
		}},
		{key: "orderV2Update", got: new(WebsocketSpotOrderChange), exp: &WebsocketSpotOrderChange{
			CanceledSize:  0.00001,
			ClientOrderID: "5c52e11203aa677f33e493fb",
			OldSize:       0.00002,
			OrderID:       "6720df7640e6fe0007b57696",
			OrderTime:     types.Time(time.UnixMilli(1730207606848)),
			OrderType:     "limit",
			OriginSize:    0.00002,
			Price:         50000,
			RemainSize:    0.00001,
			Side:          "buy",
			Size:          0.00001,
			Status:        "open",
			Symbol:        "BTC-USDT",
			Timestamp:     types.Time(time.UnixMilli(1730207616617)),
			Type:          "update",
		}},
		{key: "orderV2Filled", got: new(WebsocketSpotOrderChange), exp: &WebsocketSpotOrderChange{
			ClientOrderID: "5c52e11203aa677f33e493fc",
			FilledSize:    0.00001,
			OrderID:       "6720da3fa30a360007f5f832",
			OrderTime:     types.Time(time.UnixMilli(1730206271588)),
			OrderType:     "market",
			OriginSize:    0.00001,
			RemainFunds:   0.28828,
			Side:          "buy",
			Size:          0.00001,
			Status:        "done",
			Symbol:        "BTC-USDT",
			Timestamp:     types.Time(time.UnixMilli(1730206271616)),
			Type:          "filled",
		}},
		{key: "balanceTrade", got: new(WebsocketAccountBalance), exp: &WebsocketAccountBalance{
			AccountID:       "548674591753",
			Currency:        currency.USDT,
			Total:           21.133773386762,
			Available:       20.132773386762,
			Hold:            1.001,
			AvailableChange: -0.5005,
			HoldChange:      0.5005,
			RelationContext: WebsocketBalanceRelationContext{Symbol: "BTC-USDT", OrderID: "6721d0632db25b0007071fdc", TradeID: "11116472408358913"},
			RelationEvent:   "trade.hold",
			RelationEventID: "354689988084000",
			Time:            types.Time(time.UnixMilli(1730269283892)),
		}},
		{key: "stopOrder", got: new(WebsocketStopOrder), exp: &WebsocketStopOrder{
			OrderID:    "vs93gpupfa48anof003u85mb",
			OrderPrice: 70000,
			OrderType:  "stop",
			Side:       "buy",
			Size:       0.00007142,
			Stop:       "loss",
			StopPrice:  71000,
			Symbol:     "BTC-USDT",
			TradeType:  "TRADE",
			Type:       "open",
			CreatedAt:  types.Time(time.UnixMilli(1742305928064)),
			Timestamp:  types.Time(time.Unix(0, 1742305928091268493)),
		}},
	} {
		var msg WebsocketMessage
		require.NoErrorf(t, json.Unmarshal(wsFixture(t, "wsSpot.json", tc.key), &msg), "%s must decode", tc.key)
		require.NoErrorf(t, json.Unmarshal(msg.Data, tc.got), "%s data must decode", tc.key)
		assert.Equalf(t, tc.exp, tc.got, "%s should decode every field", tc.key)
	}
}

func TestWebsocketCandleUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var got WebsocketCandle
	require.NoError(t, json.Unmarshal([]byte(`["1791899460","83310.5","83296.2","83310.5","83281.5","3.86976157","322303.932634086"]`), &got), "UnmarshalJSON must not error")
	exp := WebsocketCandle{
		StartTime:         types.Time(time.Unix(1791899460, 0)),
		OpenPrice:         83310.5,
		ClosePrice:        83296.2,
		HighPrice:         83310.5,
		LowPrice:          83281.5,
		TransactionVolume: 3.86976157,
		TransactionAmount: 322303.932634086,
	}
	assert.Equal(t, exp, got, "UnmarshalJSON should decode every element")
	require.Error(t, json.Unmarshal([]byte(`["1791899460","x"]`), &got), "UnmarshalJSON must reject an undecodable element")
	require.Error(t, json.Unmarshal([]byte(`{}`), &got), "UnmarshalJSON must reject a non-array")
}

func TestProcessTicker(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[[]ticker.Price](t, nil, &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}})
		require.NotEmpty(t, got, "the ticker topic must relay a ticker")
		assert.Positive(t, got[0].Last, "a live ticker should carry the last price")
		assert.Positive(t, got[0].Bid, "a live ticker should carry the best bid")
		assert.Positive(t, got[0].Ask, "a live ticker should carry the best ask")
		return
	}
	ex := wsTestInstance(t)
	// The 24 hour statistics come from the REST ticker and must survive the push, which does not carry them
	require.NoError(t, ticker.ProcessTicker(&ticker.Price{ExchangeName: ex.Name, Pair: spotTestPair, AssetType: asset.Spot, High: 84149, Low: 81035.5, BaseVolume: 6381.27, QuoteVolume: 529005967.53}), "seeding the stored ticker must not error")
	exp := []ticker.Price{
		{
			Last: 83296.2, LastSize: 0.00002779, Bid: 83296.1, BidSize: 0.83775345, Ask: 83296.2, AskSize: 0.00874259,
			High: 84149, Low: 81035.5, BaseVolume: 6381.27, QuoteVolume: 529005967.53,
			Pair: spotTestPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1791899504669),
		},
		{
			// The mock harness enables the pair for margin too
			Last: 83296.2, LastSize: 0.00002779, Bid: 83296.1, BidSize: 0.83775345, Ask: 83296.2, AskSize: 0.00874259,
			Pair: spotTestPair, ExchangeName: ex.Name, AssetType: asset.Margin, LastUpdated: time.UnixMilli(1791899504669),
		},
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsSpot.json", "ticker"), "processTicker should merge the push onto each enabled asset's ticker")
	stored, err := ticker.GetTicker(ex.Name, spotTestPair, asset.Spot)
	require.NoError(t, err, "processTicker must store the ticker")
	assert.Equal(t, &exp[0], stored, "processTicker should store the ticker it relays")

	exp = []ticker.Price{{
		Last: 2522.35, LastSize: 0.008, Bid: 2522.82, BidSize: 8.6026149, Ask: 2522.83, AskSize: 0.032,
		Pair: currency.NewPairWithDelimiter("ETH", "USDT", "-"), ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1791901506268),
	}}
	assert.Equal(t, exp, wsPush(t, ex, "wsSpot.json", "allTickers"), "processTicker should read an All Tickers push for the symbol in its subject")

	require.NoError(t, ex.processTicker(t.Context(), []byte(`{"price":"1","time":1789624664495}`), "UNTRACKED-USDT"), "processTicker must ignore an untracked pair")
	assert.Empty(t, ex.Websocket.DataHandler.C, "processTicker should not relay an empty batch")
	require.Error(t, ex.processTicker(t.Context(), []byte(`{"price":[]}`), "BTC-USDT"), "processTicker must reject an undecodable push")
	require.ErrorIs(t, ex.processTicker(t.Context(), []byte(`{}`), "a"), currency.ErrCreatingPair, "processTicker must reject a symbol that is not a pair")
}

func TestProcessSpotTickers(t *testing.T) {
	t.Parallel()
	ex := wsTestInstance(t)
	// The pair is enabled for spot and margin in the test config
	pair := currency.NewPairWithDelimiter("ETH", "BTC", "-")
	require.NoError(t, ex.processSpotTickers(t.Context(), pair, marketTickerChannel, func(p *ticker.Price) { p.Bid, p.Ask = 1, 2 }), "processSpotTickers must not error")
	got, ok := (<-ex.Websocket.DataHandler.C).Data.([]ticker.Price)
	require.True(t, ok, "processSpotTickers must relay a ticker batch")
	assert.Len(t, got, 2, "processSpotTickers should relay a ticker for each enabled asset")

	err := ex.processSpotTickers(t.Context(), pair, marketTickerChannel, func(p *ticker.Price) { p.Bid, p.Ask = 2, 2 })
	require.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "processSpotTickers must return the tickers the store rejects")
	assert.Empty(t, ex.Websocket.DataHandler.C, "processSpotTickers should not relay an empty batch")

	// A pair's pushes can arrive on several connections at once, so a merge must not undo another
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		require.NoErrorf(t, ticker.ProcessTicker(&ticker.Price{ExchangeName: ex.Name, Pair: pair, AssetType: a, Bid: 1, Ask: 2}), "seeding the stored %s ticker must not error", a)
	}
	const writers, merges = 4, 250
	ex.Websocket.DataHandler = stream.NewRelay(2 * writers * merges)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			<-start
			for range merges {
				assert.NoError(t, ex.processSpotTickers(t.Context(), pair, marketTickerChannel, func(p *ticker.Price) { p.Last++ }), "processSpotTickers should not error")
			}
		})
		wg.Go(func() {
			<-start
			for range merges {
				assert.NoError(t, ex.processMergedTicker(t.Context(), pair, asset.Spot, func(p *ticker.Price) { p.Last++ }), "processMergedTicker should not error")
			}
		})
	}
	close(start)
	wg.Wait()
	stored, err := ticker.GetTicker(ex.Name, pair, asset.Spot)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, float64(2*writers*merges), stored.Last, "every merge of processSpotTickers and processMergedTicker should be kept")
	ex.Websocket.DataHandler = stream.NewRelay(1)

	require.NoError(t, ex.DisableAssetWebsocketSupport(asset.Margin), "DisableAssetWebsocketSupport must not error")
	require.NoError(t, ex.processSpotTickers(t.Context(), pair, marketTickerChannel, func(p *ticker.Price) { p.Bid, p.Ask = 1, 2 }), "processSpotTickers must not error")
	got, ok = (<-ex.Websocket.DataHandler.C).Data.([]ticker.Price)
	require.True(t, ok, "processSpotTickers must relay a ticker batch")
	assert.Len(t, got, 1, "processSpotTickers should leave out an asset without websocket support")

	ex.Name = ""
	require.ErrorIs(t, ex.processSpotTickers(t.Context(), pair, marketTickerChannel, func(*ticker.Price) {}), common.ErrExchangeNameNotSet, "processSpotTickers must return a store error")
	require.ErrorIs(t, new(Exchange).processSpotTickers(t.Context(), pair, marketTickerChannel, func(*ticker.Price) {}), currency.ErrPairManagerNotInitialised, "processSpotTickers must return an asset lookup error")
}

func TestProcessOrderbookLevel1(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*orderbook.Depth](t, nil, &subscription.Subscription{Channel: marketOrderbookDepth1Channel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}})
		book, err := got.Retrieve()
		require.NoError(t, err, "Retrieve must not error")
		assert.Len(t, book.Bids, 1, "a level 1 book should have one bid")
		assert.Len(t, book.Asks, 1, "a level 1 book should have one ask")
		return
	}
	ex := wsTestInstance(t)
	require.NoError(t, ex.wsHandleData(t.Context(), nil, wsFixture(t, "wsSpot.json", "level1")), "wsHandleData must not error")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		book, err := ex.Websocket.Orderbook.GetOrderbook(spotTestPair, a)
		require.NoErrorf(t, err, "the level 1 book must be stored for %s", a)
		exp := &orderbook.Book{
			Bids:        orderbook.Levels{{Price: 83296.1, Amount: 0.83775345}},
			Asks:        orderbook.Levels{{Price: 83296.2, Amount: 0.17744259}},
			Exchange:    ex.Name,
			Pair:        spotTestPair,
			Asset:       a,
			LastUpdated: time.UnixMilli(1791899505902),
			InsertedAt:  book.InsertedAt,
		}
		assert.Equalf(t, exp, book, "the level 1 push should load a one level book for %s", a)
	}

	require.NoError(t, ex.processOrderbookLevel1(t.Context(), []byte(`{"asks":["83296.2","0.17744259"],"bids":[],"timestamp":1791899505902}`), "BTC-USDT"), "processOrderbookLevel1 must load a book with one empty side")
	book, err := ex.Websocket.Orderbook.GetOrderbook(spotTestPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Empty(t, book.Bids, "an empty side should load without levels")
	require.ErrorIs(t, ex.processOrderbookLevel1(t.Context(), []byte(`{"asks":[],"bids":[]}`), "BTC-USDT"), errEmptyOrderbookLevelOne, "processOrderbookLevel1 must reject a push without prices")
	require.Error(t, ex.processOrderbookLevel1(t.Context(), []byte(`{"asks":{}}`), "BTC-USDT"), "processOrderbookLevel1 must reject an undecodable push")
	require.ErrorIs(t, ex.processOrderbookLevel1(t.Context(), []byte(`{}`), "a"), currency.ErrCreatingPair, "processOrderbookLevel1 must reject a symbol that is not a pair")
	require.ErrorIs(t, new(Exchange).processOrderbookLevel1(t.Context(), wsFixtureData(t, "wsSpot.json", "level1"), "BTC-USDT"), currency.ErrPairManagerNotInitialised, "processOrderbookLevel1 must return an asset lookup error")
	unnamed := testInstance(t)
	unnamed.Name = ""
	require.ErrorIs(t, unnamed.processOrderbookLevel1(t.Context(), wsFixtureData(t, "wsSpot.json", "level1"), "BTC-USDT"), common.ErrExchangeNameNotSet, "processOrderbookLevel1 must return a load error")
}

// wsFixtureData returns the data of a recorded frame
func wsFixtureData(tb testing.TB, file, key string) []byte {
	tb.Helper()
	var msg WebsocketMessage
	require.NoErrorf(tb, json.Unmarshal(wsFixture(tb, file, key), &msg), "%s must decode", key)
	return msg.Data
}

// fillWebsocketDataHandler gives the exchange a full data handler and returns the error a relay to it wraps
func fillWebsocketDataHandler(t *testing.T, ex *Exchange) error {
	t.Helper()
	full := stream.NewRelay(1)
	require.NoError(t, full.Send(t.Context(), "filler"), "Send must not error for the payload that fills the buffer")
	errBufferFull := errors.Unwrap(full.Send(t.Context(), "overflow"))
	require.Error(t, errBufferFull, "Send must error for a payload beyond the buffer")
	ex.Websocket.DataHandler = full
	return errBufferFull
}

func TestProcessOrderbook(t *testing.T) {
	t.Parallel()
	if !mockTests {
		for _, channel := range []string{marketOrderbookDepth5Channel, marketOrderbookDepth50Channel} {
			got := liveWsData[*orderbook.Depth](t, nil, &subscription.Subscription{Channel: channel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}})
			book, err := got.Retrieve()
			require.NoErrorf(t, err, "Retrieve must not error for %s", channel)
			assert.NotEmptyf(t, book.Bids, "%s should load bids", channel)
			assert.NotEmptyf(t, book.Asks, "%s should load asks", channel)
		}
		return
	}
	ex := wsTestInstance(t)
	require.NoError(t, ex.wsHandleData(t.Context(), nil, wsFixture(t, "wsSpot.json", "level50")), "wsHandleData must not error")
	book, err := ex.Websocket.Orderbook.GetOrderbook(spotTestPair, asset.Spot)
	require.NoError(t, err, "the level 50 book must be stored")
	exp := &orderbook.Book{
		Bids:        orderbook.Levels{{Price: 83296.1, Amount: 0.83775345}, {Price: 83296, Amount: 0.07352707}, {Price: 83295.9, Amount: 0.0005}},
		Asks:        orderbook.Levels{{Price: 83296.2, Amount: 0.00874259}, {Price: 83296.3, Amount: 0.0085}, {Price: 83296.4, Amount: 0.0085}},
		Exchange:    ex.Name,
		Pair:        spotTestPair,
		Asset:       asset.Spot,
		LastUpdated: time.UnixMilli(1791899505798),
		InsertedAt:  book.InsertedAt,
	}
	assert.Equal(t, exp, book, "the level 50 push should load the book")

	pair := currency.NewPairWithDelimiter("ETH", "BTC", "-")
	require.NoError(t, ex.processOrderbook(t.Context(), []byte(`{"asks":[["0.0500","1.5"],["0.0500","0.5"],["0.0600","2"]],"bids":[["0.0400","3"],["0.0400","1"],["0.0300","4"]],"timestamp":1700555340197}`), "ETH-BTC", marketOrderbookDepth5Channel), "processOrderbook must not error")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		merged, err := ex.Websocket.Orderbook.GetOrderbook(pair, a)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
		assert.Equal(t, orderbook.Levels{{Price: 0.05, Amount: 2}, {Price: 0.06, Amount: 2}}, merged.Asks, "processOrderbook should merge levels KuCoin rounds to one price")
		assert.Equal(t, orderbook.Levels{{Price: 0.04, Amount: 4}, {Price: 0.03, Amount: 4}}, merged.Bids, "processOrderbook should merge levels KuCoin rounds to one price")
	}

	before := time.Now()
	require.NoError(t, ex.processOrderbook(t.Context(), []byte(`{"asks":[["0.0500","1"]],"bids":[["0.0400","1"]]}`), "ETH-BTC", marketOrderbookDepth5Channel), "processOrderbook must not error without a timestamp")
	book, err = ex.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.False(t, book.LastUpdated.Before(before), "a push without a timestamp should be stamped when loaded")

	require.Error(t, ex.processOrderbook(t.Context(), []byte(`{"asks":`), "ETH-BTC", marketOrderbookDepth5Channel), "processOrderbook must reject an undecodable push")
	require.ErrorIs(t, ex.processOrderbook(t.Context(), []byte(`{}`), "a", marketOrderbookDepth5Channel), currency.ErrCreatingPair, "processOrderbook must reject a symbol that is not a pair")
	require.ErrorIs(t, new(Exchange).processOrderbook(t.Context(), []byte(`{}`), "ETH-BTC", marketOrderbookDepth5Channel), currency.ErrPairManagerNotInitialised, "processOrderbook must return an asset lookup error")
	unnamed := testInstance(t)
	unnamed.Name = ""
	require.ErrorIs(t, unnamed.processOrderbook(t.Context(), []byte(`{"asks":[["0.0500","1"]],"bids":[["0.0400","1"]],"timestamp":1700555340197}`), "ETH-BTC", marketOrderbookDepth5Channel), common.ErrExchangeNameNotSet, "processOrderbook must return a load error")
}

// newTestUpdateManager returns an update manager that syncs books from a snapshot at the given update ID
func newTestUpdateManager(ex *Exchange, lastUpdateID int64) *orderbookmanager.UpdateManager {
	return orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
		FetchDeadline: orderbookmanager.DefaultWSOrderbookUpdateDeadline,
		FetchOrderbook: func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
			return &orderbook.Book{
				Exchange: ex.Name, Pair: p, Asset: a,
				Bids: orderbook.Levels{{Price: 80000, Amount: 1}}, Asks: orderbook.Levels{{Price: 90000, Amount: 1}},
				LastUpdateID: lastUpdateID, LastUpdated: time.UnixMilli(1791901473000),
			}, nil
		},
		CheckPendingUpdate: checkPendingUpdate,
		Orderbook:          &ex.Websocket.Orderbook,
	})
}

func TestProcessSpotOrderbookIncrement(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		got := liveWsData[*orderbook.Depth](t, nil, &subscription.Subscription{Channel: marketOrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}})
		book, err := got.Retrieve()
		require.NoError(t, err, "Retrieve must not error")
		assert.NotEmpty(t, book.Bids, "the realtime book should hold bids")
		return
	}
	ex := wsTestInstance(t)
	ex.wsOBUpdateMgr = newTestUpdateManager(ex, 38627404121)
	require.NoError(t, ex.wsHandleData(t.Context(), nil, wsFixture(t, "wsSpot.json", "increment")), "wsHandleData must not error")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		require.EventuallyWithTf(t, func(c *assert.CollectT) {
			book, err := ex.Websocket.Orderbook.GetOrderbook(spotTestPair, a)
			require.NoError(c, err, "GetOrderbook must return the realtime book")
			assert.Equal(c, int64(38627404123), book.LastUpdateID, "the book should carry the push's last sequence")
			assert.Equal(c, orderbook.Levels{{Price: 83692.5, Amount: 0.04827727, ID: 38627404123}, {Price: 80000, Amount: 1}}, book.Bids, "the push's bid should be applied")
			assert.Equal(c, orderbook.Levels{{Price: 83708.9, Amount: 0.07939911, ID: 38627404122}, {Price: 90000, Amount: 1}}, book.Asks, "the push's ask should be applied")
		}, time.Second, 10*time.Millisecond, "the increment must reach the %s book", a)
	}

	pair := currency.NewPairWithDelimiter("DOGE", "USDT", "-")
	require.NoError(t, ex.processSpotOrderbookIncrement(t.Context(), []byte(`{"changes":{"asks":[],"bids":[["0.2","10","38627404123"]]},"sequenceEnd":38627404123,"sequenceStart":38627404122,"symbol":"DOGE-USDT","time":1791901473976}`), "DOGE-USDT"), "a subscribed pair outside the enabled lists must not error")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		book, err := ex.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
		require.NoError(c, err, "a pair outside the enabled lists must be booked as spot")
		assert.Equal(c, int64(38627404123), book.LastUpdateID, "the update should apply to the spot book")
	}, time.Second, 10*time.Millisecond, "the increment must reach the spot book")

	require.ErrorIs(t, ex.processSpotOrderbookIncrement(t.Context(), []byte(`{}`), "a"), currency.ErrCreatingPair, "processSpotOrderbookIncrement must reject a symbol that is not a pair")
	require.Error(t, ex.processSpotOrderbookIncrement(t.Context(), []byte(`{"changes":`), "BTC-USDT"), "processSpotOrderbookIncrement must reject an undecodable push")
	require.ErrorIs(t, new(Exchange).processSpotOrderbookIncrement(t.Context(), []byte(`{}`), "BTC-USDT"), currency.ErrPairManagerNotInitialised, "processSpotOrderbookIncrement must return an asset lookup error")

	t.Run("update manager error", func(t *testing.T) {
		t.Parallel()
		ex := wsTestInstance(t)
		ex.wsOBUpdateMgr = newTestUpdateManager(ex, 38627404121)
		// XRP-USDT is enabled for neither asset, so only its spot book is synced
		xrpUSDT := currency.NewPairWithDelimiter("XRP", "USDT", "-")
		require.NoError(t, ex.processSpotOrderbookIncrement(t.Context(), []byte(`{"changes":{"asks":[],"bids":[["80500","10","38627404122"]]},"sequenceEnd":38627404122,"sequenceStart":38627404122,"symbol":"XRP-USDT","time":1791901473976}`), "XRP-USDT"), "processSpotOrderbookIncrement must not error")
		// The sync relays the book when it loads the snapshot and again when it applies the push, its last use of the store,
		// so the store is only set up again below once both have arrived
		for range 2 {
			select {
			case resp := <-ex.Websocket.DataHandler.C:
				require.IsType(t, &orderbook.Depth{}, resp.Data, "the update manager must relay the synced book")
			case <-time.After(5 * time.Second):
				require.FailNow(t, "the update manager did not sync the book")
			}
		}
		// Setting the store up again, as setting the websocket up again does, drops the book the update manager has synced
		require.NoError(t, ex.Websocket.Orderbook.Setup(ex.Name, ex.Websocket.DataHandler, false), "Setup must not error")
		err := ex.processSpotOrderbookIncrement(t.Context(), []byte(`{"changes":{"asks":[["89500","5","38627404123"]],"bids":[]},"sequenceEnd":38627404123,"sequenceStart":38627404122,"symbol":"XRP-USDT","time":1791901474976}`), "XRP-USDT")
		require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "processSpotOrderbookIncrement must return the update manager's error")
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			book, err := ex.Websocket.Orderbook.GetOrderbook(xrpUSDT, asset.Spot)
			require.NoError(c, err, "the update manager must sync the book again")
			assert.Equal(c, int64(38627404123), book.LastUpdateID, "the book should be synced again up to the push")
		}, 5*time.Second, 10*time.Millisecond, "the update manager must sync the book again from a new snapshot")
	})
}

func TestProcessCallAuctionOrderbook(t *testing.T) {
	t.Parallel()
	if !mockTests {
		// Pushes only flow during a call auction, so live only proves the topic is served
		liveWsSubscribe(t, &subscription.Subscription{Channel: callAuctionOrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}})
		return
	}
	ex := wsTestInstance(t)
	exp := &WebsocketCallAuctionOrderbook{
		Symbol:    "BTC-USDT",
		Asks:      orderbook.LevelsArrayPriceAmount{{Amount: 0.08168874, Price: 95964.3}, {Amount: 0.00985094, Price: 95967.9}},
		Bids:      orderbook.LevelsArrayPriceAmount{{Amount: 1.35483359, Price: 95964.2}, {Amount: 0.01117492, Price: 95964.1}},
		Timestamp: types.Time(time.UnixMilli(1733124805073)),
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsSpot.json", "callAuctionLevel50"), "processCallAuctionOrderbook should relay the push with the topic's symbol")
	require.Error(t, ex.processCallAuctionOrderbook(t.Context(), []byte(`{"asks":{}}`), "BTC-USDT"), "processCallAuctionOrderbook must reject an undecodable push")
}

func TestProcessCallAuctionData(t *testing.T) {
	t.Parallel()
	if !mockTests {
		liveWsSubscribe(t, &subscription.Subscription{Channel: callAuctionDataChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}})
		return
	}
	ex := wsTestInstance(t)
	exp := &WebsocketCallAuctionInfo{
		Symbol:                  "BTC-USDT",
		EstimatedPrice:          0.17,
		EstimatedSize:           0.03715004,
		SellOrderRangeLowPrice:  1.788,
		SellOrderRangeHighPrice: 2.788,
		BuyOrderRangeLowPrice:   1.788,
		BuyOrderRangeHighPrice:  2.788,
		Time:                    types.Time(time.UnixMilli(1550653727731)),
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsSpot.json", "callAuctionData"), "processCallAuctionData should relay the push")

	require.NoError(t, ex.processCallAuctionData(t.Context(), wsFixtureData(t, "wsSpot.json", "callAuctionData")), "processCallAuctionData must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "processCallAuctionData must relay the push")
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processCallAuctionData should relay every field of the push")
	require.Error(t, ex.processCallAuctionData(t.Context(), []byte(`{"estimatedPrice":[]}`)), "processCallAuctionData must reject an undecodable push")
	assert.Empty(t, ex.Websocket.DataHandler.C, "processCallAuctionData should not relay an undecodable push")
}

func TestProcessCandlesticks(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*kline.Item](t, nil, &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}, Interval: kline.OneMin})
		require.Len(t, got.Candles, 1, "a candle push must relay one candle")
		assert.Positive(t, got.Candles[0].Close, "a live candle should carry a close price")
		assert.Equal(t, kline.OneMin, got.Interval, "the candle should carry the subscribed interval")
		return
	}
	ex := wsTestInstance(t)
	require.NoError(t, ex.wsHandleData(t.Context(), nil, wsFixture(t, "wsSpot.json", "klines")), "wsHandleData must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 2, "wsHandleData must relay a candle for each enabled asset")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		exp := &kline.Item{
			Pair:     spotTestPair,
			Asset:    a,
			Exchange: ex.Name,
			Interval: kline.OneMin,
			Candles:  []kline.Candle{{Time: time.Unix(1791899460, 0), Open: 83310.5, Close: 83296.2, High: 83310.5, Low: 83281.5, Volume: 3.86976157}},
		}
		assert.Equalf(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processCandlesticks should relay the %s candle", a)
	}

	data := wsFixtureData(t, "wsSpot.json", "klines")
	require.ErrorIs(t, ex.processCandlesticks(t.Context(), data, "BTC-USDT"), errMalformedCandleTopic, "processCandlesticks must reject a topic without an interval")
	require.ErrorIs(t, ex.processCandlesticks(t.Context(), data, "BTC-USDT_2min"), kline.ErrInvalidInterval, "processCandlesticks must reject an unknown interval")
	require.ErrorIs(t, ex.processCandlesticks(t.Context(), data, "a_1min"), currency.ErrCreatingPair, "processCandlesticks must reject a symbol that is not a pair")
	require.Error(t, ex.processCandlesticks(t.Context(), []byte(`{"candles":{}}`), "BTC-USDT_1min"), "processCandlesticks must reject an undecodable push")
	require.ErrorIs(t, new(Exchange).processCandlesticks(t.Context(), data, "BTC-USDT_1min"), currency.ErrPairManagerNotInitialised, "processCandlesticks must return an asset lookup error")
	require.NoError(t, ex.DisableAssetWebsocketSupport(asset.Spot), "DisableAssetWebsocketSupport must not error")
	require.NoError(t, ex.DisableAssetWebsocketSupport(asset.Margin), "DisableAssetWebsocketSupport must not error")
	require.NoError(t, ex.processCandlesticks(t.Context(), data, "BTC-USDT_1min"), "processCandlesticks must not error")
	assert.Empty(t, ex.Websocket.DataHandler.C, "processCandlesticks should not relay a candle for an asset without websocket support")

	full := wsTestInstance(t)
	errBufferFull := fillWebsocketDataHandler(t, full)
	require.ErrorIs(t, full.processCandlesticks(t.Context(), data, "BTC-USDT_1min"), errBufferFull, "processCandlesticks must return the error relaying a candle")
}

func TestProcessTradeData(t *testing.T) {
	t.Parallel()
	enableTradeFeed := func(ex *Exchange) {
		ex.SetTradeFeedStatus(true)
		ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	}
	if !mockTests {
		got := liveWsData[[]trade.Data](t, enableTradeFeed, &subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}})
		require.NotEmpty(t, got, "the trade topic must relay a trade")
		assert.Positive(t, got[0].Price, "a live trade should carry a price")
		assert.NotEmpty(t, got[0].TID, "a live trade should carry an ID")
		return
	}
	ex := wsTestInstance(t)
	require.NoError(t, ex.processTradeData(wsFixtureData(t, "wsSpot.json", "trade"), "BTC-USDT"), "processTradeData must not error without a trade feed")
	assert.Empty(t, ex.Websocket.DataHandler.C, "processTradeData should relay nothing without a trade feed")

	enableTradeFeed(ex)
	require.NoError(t, ex.wsHandleData(t.Context(), nil, wsFixture(t, "wsSpot.json", "trade")), "wsHandleData must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 2, "wsHandleData must relay a trade for each enabled asset")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		exp := []trade.Data{{
			TID:          "24806846522605568",
			Exchange:     ex.Name,
			CurrencyPair: spotTestPair,
			AssetType:    a,
			Side:         order.Buy,
			Price:        83296.2,
			Amount:       0.00534199,
			Timestamp:    time.UnixMilli(1791899507179),
		}}
		assert.Equalf(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processTradeData should relay the %s trade", a)
	}

	data := wsFixtureData(t, "wsSpot.json", "trade")
	require.Error(t, ex.processTradeData([]byte(`{"price":[]}`), "BTC-USDT"), "processTradeData must reject an undecodable push")
	require.ErrorIs(t, ex.processTradeData(data, "a"), currency.ErrCreatingPair, "processTradeData must reject a symbol that is not a pair")
	require.ErrorIs(t, ex.processTradeData([]byte(`{"side":"sideways"}`), "BTC-USDT"), order.ErrSideIsInvalid, "processTradeData must reject an unknown side")
	unready := wsTestInstance(t)
	enableTradeFeed(unready)
	unready.CurrencyPairs = currency.PairsManager{}
	require.ErrorIs(t, unready.processTradeData(data, "BTC-USDT"), currency.ErrPairManagerNotInitialised, "processTradeData must return an asset lookup error")

	full := wsTestInstance(t)
	errBufferFull := fillWebsocketDataHandler(t, full)
	enableTradeFeed(full)
	require.ErrorIs(t, full.processTradeData(data, "BTC-USDT"), errBufferFull, "processTradeData must return the error relaying a trade")
}

func TestProcessMarketSnapshot(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[[]ticker.Price](t, nil, &subscription.Subscription{Channel: marketSnapshotChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}})
		require.NotEmpty(t, got, "the snapshot topic must relay a ticker")
		assert.Positive(t, got[0].High, "a live snapshot should carry the 24 hour high")
		assert.Positive(t, got[0].BaseVolume, "a live snapshot should carry the 24 hour volume")
		return
	}
	ex := wsTestInstance(t)
	require.NoError(t, ticker.ProcessTicker(&ticker.Price{ExchangeName: ex.Name, Pair: spotTestPair, AssetType: asset.Spot, OpenInterest: 7}), "seeding the stored ticker must not error")
	exp := []ticker.Price{
		{
			Last: 83296.2, LastSize: 0.00002779, Bid: 83299.8, BidSize: 0.4781167, Ask: 83300.3, AskSize: 0.0055,
			High: 84149, Low: 81035.5, Open: 82775.9, Close: 83296.2, BaseVolume: 6381.269114425228, QuoteVolume: 529005967.528381, OpenInterest: 7,
			Pair: spotTestPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1791899507490),
		},
		{
			Last: 83296.2, LastSize: 0.00002779, Bid: 83299.8, BidSize: 0.4781167, Ask: 83300.3, AskSize: 0.0055,
			High: 84149, Low: 81035.5, Open: 82775.9, Close: 83296.2, BaseVolume: 6381.269114425228, QuoteVolume: 529005967.528381,
			Pair: spotTestPair, ExchangeName: ex.Name, AssetType: asset.Margin, LastUpdated: time.UnixMilli(1791899507490),
		},
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsSpot.json", "symbolSnapshot"), "processMarketSnapshot should merge every field the snapshot carries onto each enabled asset's ticker")

	ethBTC := currency.NewPairWithDelimiter("ETH", "BTC", "-")
	exp = []ticker.Price{
		{
			Last: 0.03011, LastSize: 0.0020015, Bid: 0.03012, BidSize: 1.02017, Ask: 0.03013, AskSize: 3.1237805,
			High: 0.03062, Low: 0.0297, Open: 0.03052, Close: 0.03011, BaseVolume: 1603.2892686, QuoteVolume: 48.545435885492,
			Pair: ethBTC, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1791901508163),
		},
		{
			Last: 0.03011, LastSize: 0.0020015, Bid: 0.03012, BidSize: 1.02017, Ask: 0.03013, AskSize: 3.1237805,
			High: 0.03062, Low: 0.0297, Open: 0.03052, Close: 0.03011, BaseVolume: 1603.2892686, QuoteVolume: 48.545435885492,
			Pair: ethBTC, ExchangeName: ex.Name, AssetType: asset.Margin, LastUpdated: time.UnixMilli(1791901508163),
		},
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsSpot.json", "marketSnapshot"), "processMarketSnapshot should merge a market snapshot onto the ticker of the symbol it carries")

	require.NoError(t, ex.processMarketSnapshot(t.Context(), []byte(`{"data":{"symbol":"UNTRACKED-USDT"}}`)), "processMarketSnapshot must ignore an untracked pair")
	assert.Empty(t, ex.Websocket.DataHandler.C, "processMarketSnapshot should not relay an empty batch")
	require.Error(t, ex.processMarketSnapshot(t.Context(), []byte(`{"data":[]}`)), "processMarketSnapshot must reject an undecodable push")
	require.ErrorIs(t, ex.processMarketSnapshot(t.Context(), []byte(`{"data":{"symbol":"a"}}`)), currency.ErrCreatingPair, "processMarketSnapshot must reject a symbol that is not a pair")
}

func TestProcessOrderChange(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: spotOrderV2Channel, Authenticated: true}, &subscription.Subscription{Channel: spotOrderV1Channel, Authenticated: true})
		return
	}
	ex := wsTestInstance(t)
	open := &order.Detail{
		Price: 50000, Amount: 0.00001, RemainingAmount: 0.00001, Exchange: ex.Name, OrderID: "6720ecd9ec71f4000747731a", ClientOrderID: "5c52e11203aa677f33e493fb",
		Type: order.Limit, Side: order.Buy, Status: order.Open, AssetType: asset.Spot, Date: time.UnixMilli(1730211033305), LastUpdated: time.UnixMilli(1730211033335), Pair: spotTestPair,
	}
	match := &order.Detail{
		Amount: 0.00001, ExecutedAmount: 0.00001, Exchange: ex.Name, OrderID: "6720da3fa30a360007f5f832", ClientOrderID: "5c52e11203aa677f33e493fc",
		Type: order.Market, Side: order.Buy, Status: order.Filled, AssetType: asset.Spot, Date: time.UnixMilli(1730206271588), LastUpdated: time.UnixMilli(1730206271616), Pair: spotTestPair,
		Trades: []order.TradeHistory{{Price: 71171.9, Amount: 0.00001, Exchange: ex.Name, TID: "11116472408358913", Type: order.Market, Side: order.Buy, Timestamp: time.UnixMilli(1730206271616)}},
	}
	for _, tc := range []struct {
		key string
		exp *order.Detail
	}{
		{key: "orderV2Received", exp: &order.Detail{
			Amount: 0.00001, Exchange: ex.Name, OrderID: "6720da3fa30a360007f5f832", ClientOrderID: "5c52e11203aa677f33e493fc",
			Type: order.Market, Side: order.Buy, Status: order.New, AssetType: asset.Spot, Date: time.UnixMilli(1730206271588), LastUpdated: time.UnixMilli(1730206271616), Pair: spotTestPair,
		}},
		{key: "orderV2Open", exp: open},
		{key: "orderV1Open", exp: open},
		{key: "orderV2Update", exp: &order.Detail{
			Price: 50000, Amount: 0.00001, RemainingAmount: 0.00001, Exchange: ex.Name, OrderID: "6720df7640e6fe0007b57696", ClientOrderID: "5c52e11203aa677f33e493fb",
			Type: order.Limit, Side: order.Buy, Status: order.Open, AssetType: asset.Spot, Date: time.UnixMilli(1730207606848), LastUpdated: time.UnixMilli(1730207616617), Pair: spotTestPair,
		}},
		{key: "orderV2Match", exp: match},
		{key: "orderV1Match", exp: match},
		{key: "orderV2Filled", exp: &order.Detail{
			Amount: 0.00001, ExecutedAmount: 0.00001, Exchange: ex.Name, OrderID: "6720da3fa30a360007f5f832", ClientOrderID: "5c52e11203aa677f33e493fc",
			Type: order.Market, Side: order.Buy, Status: order.Filled, AssetType: asset.Spot, Date: time.UnixMilli(1730206271588), LastUpdated: time.UnixMilli(1730206271616), Pair: spotTestPair,
		}},
		{key: "orderV2Canceled", exp: &order.Detail{
			Price: 50000, Amount: 0.00001, Exchange: ex.Name, OrderID: "6720df7640e6fe0007b57696", ClientOrderID: "5c52e11203aa677f33e493fb",
			Type: order.Limit, Side: order.Buy, Status: order.Cancelled, AssetType: asset.Spot, Date: time.UnixMilli(1730207606848), LastUpdated: time.UnixMilli(1730207624559), Pair: spotTestPair,
		}},
	} {
		assert.Equalf(t, tc.exp, wsPush(t, ex, "wsSpot.json", tc.key), "processOrderChange should relay %s as an order update", tc.key)
	}

	// The fixtures' pair is enabled for spot and margin, so only the order ID tells a margin order from a spot one
	submitted, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spotTestPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Limit, Price: 50000, Amount: 0.001, MarginType: margin.Isolated, TimeInForce: order.PostOnly, AutoBorrow: true, AutoRepay: true, ClientOrderID: "gct-margin-limit"})
	require.NoError(t, err, "SubmitOrder must not error")
	crossSubmitted, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spotTestPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, QuoteAmount: 1, ClientOrderID: "5c52e11203aa677f33e493fc"})
	require.NoError(t, err, "SubmitOrder must not error for a cross margin order")
	listed, err := ex.GetOrderInfo(t.Context(), "671667306afcdb000723107f", spotTestPair, asset.Margin)
	require.NoError(t, err, "GetOrderInfo must not error")
	pushAsset := func(key, orderID string) asset.Item {
		t.Helper()
		data := bytes.Replace(wsFixtureData(t, "wsSpot.json", key), []byte("6720da3fa30a360007f5f832"), []byte(orderID), 1)
		require.NoErrorf(t, ex.processOrderChange(t.Context(), data), "processOrderChange must not error for %s", key)
		d, ok := (<-ex.Websocket.DataHandler.C).Data.(*order.Detail)
		require.Truef(t, ok, "%s must relay an order detail", key)
		return d.AssetType
	}
	for _, orderID := range []string{submitted.OrderID, crossSubmitted.OrderID, listed.OrderID} {
		assert.Equalf(t, asset.Margin, pushAsset("orderV2Received", orderID), "a push for margin order %s should be a margin order update", orderID)
		assert.Equalf(t, asset.Margin, pushAsset("orderV2Filled", orderID), "the push finishing margin order %s should be a margin order update", orderID)
		assert.Equalf(t, asset.Spot, pushAsset("orderV2Received", orderID), "a push for finished margin order %s should be read as for an order placed elsewhere", orderID)
	}

	// REST reporting a margin order finished forgets it, as its last push may have been missed
	_, err = ex.marginHFOrderDetail(&MarginHFOrderResponse{ID: "gct-margin-live", Symbol: "BTC-USDT", Type: "limit", Side: "buy", Active: true})
	require.NoError(t, err, "marginHFOrderDetail must not error")
	for _, active := range []bool{true, false} {
		_, err = ex.marginHFOrderDetail(&MarginHFOrderResponse{ID: "gct-margin-finished", Symbol: "BTC-USDT", Type: "limit", Side: "buy", Active: active})
		require.NoError(t, err, "marginHFOrderDetail must not error")
	}
	assert.Equal(t, asset.Spot, pushAsset("orderV2Received", "gct-margin-finished"), "a push for a margin order REST reported finished should be read as for an order placed elsewhere")
	assert.Equal(t, asset.Margin, pushAsset("orderV2Received", "gct-margin-live"), "a push for a margin order still live should be a margin order update")

	// Orders are placed and listed while their pushes arrive
	ex.Websocket.DataHandler = stream.NewRelay(100)
	var wg sync.WaitGroup
	for i := range 50 {
		orderID := "gct-concurrent-" + strconv.Itoa(i)
		wg.Go(func() {
			_, err := ex.marginHFOrderDetail(&MarginHFOrderResponse{ID: orderID, Symbol: "BTC-USDT", Type: "limit", Side: "buy", Active: true})
			assert.NoError(t, err, "marginHFOrderDetail should not error")
		})
		wg.Go(func() {
			data := bytes.Replace(wsFixtureData(t, "wsSpot.json", "orderV2Filled"), []byte("6720da3fa30a360007f5f832"), []byte(orderID), 1)
			assert.NoError(t, ex.processOrderChange(t.Context(), data), "processOrderChange should not error")
		})
	}
	wg.Wait()

	require.Error(t, ex.processOrderChange(t.Context(), []byte(`{"size":[]}`)), "processOrderChange must reject an undecodable push")
	require.ErrorIs(t, ex.processOrderChange(t.Context(), []byte(`{"symbol":"a"}`)), currency.ErrCreatingPair, "processOrderChange must reject a symbol that is not a pair")
	require.ErrorIs(t, ex.processOrderChange(t.Context(), []byte(`{"symbol":"BTC-USDT","orderType":"iceberg"}`)), order.ErrUnrecognisedOrderType, "processOrderChange must reject an unknown order type")
	require.ErrorIs(t, ex.processOrderChange(t.Context(), []byte(`{"symbol":"BTC-USDT","orderType":"limit","side":"sideways"}`)), order.ErrSideIsInvalid, "processOrderChange must reject an unknown side")
	require.ErrorIs(t, ex.processOrderChange(t.Context(), []byte(`{"symbol":"BTC-USDT","orderType":"limit","side":"buy","type":"expired"}`)), errUnknownOrderChangeType, "processOrderChange must reject an unknown message type")
}

func TestSpotOrderAsset(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	ex.marginOrderIDs.Store("gct-margin", struct{}{})
	for _, tc := range []struct {
		name    string
		pair    currency.Pair
		orderID string
		exp     asset.Item
	}{
		{name: "an order placed elsewhere on a pair enabled for spot and margin", pair: spotTestPair, orderID: "gct-spot", exp: asset.Spot},
		{name: "a known margin order on a pair enabled for spot and margin", pair: spotTestPair, orderID: "gct-margin", exp: asset.Margin},
		{name: "a known margin order on a pair enabled for spot alone", pair: currency.NewPairWithDelimiter("ETH", "USDT", "-"), orderID: "gct-margin", exp: asset.Spot},
		{name: "an order on a pair enabled for margin alone", pair: currency.NewPairWithDelimiter("TRX", "BTC", "-"), orderID: "gct-spot", exp: asset.Margin},
		{name: "a known margin order on a pair enabled for neither", pair: currency.NewPairWithDelimiter("DOGE", "USDT", "-"), orderID: "gct-margin", exp: asset.Spot},
	} {
		assert.Equalf(t, tc.exp, ex.spotOrderAsset(tc.pair, tc.orderID), "spotOrderAsset should read %s as a %s order", tc.name, tc.exp)
	}
}

func TestOrderChangeStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status, messageType string
		filled, remaining   float64
		exp                 order.Status
	}{
		{status: "new", messageType: "received", exp: order.New},
		{status: "open", messageType: "open", remaining: 1, exp: order.Open},
		{status: "open", messageType: "open", filled: 1, remaining: 1, exp: order.PartiallyFilled},
		{status: "open", messageType: "match", filled: 1, remaining: 1, exp: order.PartiallyFilled},
		{status: "match", messageType: "match", filled: 2, exp: order.Filled},
		{status: "done", messageType: "filled", filled: 2, exp: order.Filled},
		{status: "done", messageType: "canceled", exp: order.Cancelled},
		{status: "done", messageType: "canceled", filled: 1, exp: order.PartiallyFilledCancelled},
		{status: "open", messageType: "update", remaining: 1, exp: order.Open},
		{status: "open", messageType: "update", filled: 1, remaining: 1, exp: order.PartiallyFilled},
		{status: "done", messageType: "update", exp: order.Cancelled},
		{status: "done", messageType: "update", filled: 1, exp: order.PartiallyFilledCancelled},
	} {
		got, err := orderChangeStatus(tc.status, tc.messageType, tc.filled, tc.remaining)
		require.NoErrorf(t, err, "orderChangeStatus must not error for %s %s", tc.status, tc.messageType)
		assert.Equalf(t, tc.exp, got, "orderChangeStatus should map %s %s filled %v remaining %v", tc.status, tc.messageType, tc.filled, tc.remaining)
	}
	_, err := orderChangeStatus("done", "expired", 0, 0)
	require.ErrorIs(t, err, errUnknownOrderChangeType, "orderChangeStatus must reject an unknown message type")
}

func TestProcessAccountBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: accountBalanceChannel, Authenticated: true})
		return
	}
	ex := wsTestInstance(t)
	for _, tc := range []struct {
		key string
		a   asset.Item
	}{
		{key: "balanceTrade", a: asset.Spot},
		{key: "balanceMargin", a: asset.Margin},
	} {
		got, ok := wsPush(t, ex, "wsSpot.json", tc.key).(accounts.SubAccounts)
		require.Truef(t, ok, "%s must relay sub-accounts", tc.key)
		exp := accounts.SubAccounts{accounts.NewSubAccount(tc.a, "")}
		exp[0].Balances.Set(currency.USDT, accounts.Balance{Total: 21.133773386762, Hold: 1.001, Free: 20.132773386762, UpdatedAt: time.UnixMilli(1730269283892)})
		assert.Equalf(t, exp, got, "%s should relay the %s balance", tc.key, tc.a)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		stored, err := ex.Accounts.GetBalance("", creds, tc.a, currency.USDT)
		require.NoErrorf(t, err, "the %s balance must be stored", tc.a)
		assert.Equalf(t, 20.132773386762, stored.Free, "the stored %s balance should be the pushed one", tc.a)
	}

	// The main account's push adds its share to the trade account's, as REST saves the spot balance
	got, ok := wsPush(t, ex, "wsSpot.json", "balanceMain").(accounts.SubAccounts)
	require.True(t, ok, "balanceMain must relay sub-accounts")
	spot := accounts.SubAccounts{accounts.NewSubAccount(asset.Spot, "")}
	spot[0].Balances.Set(currency.USDT, accounts.Balance{Total: 42.267546773524, Hold: 2.002, Free: 40.265546773524, UpdatedAt: time.UnixMilli(1730269283892)})
	assert.Equal(t, spot, got, "a main account push should relay the spot balance with the trade account's share")

	// An isolated margin account is the margin sub-account of its symbol
	isolated := bytes.Replace(wsFixtureData(t, "wsSpot.json", "balanceMain"), []byte("main.deposit"), []byte("isolated_BTC-USDT.hold"), 1)
	require.NoError(t, ex.processAccountBalance(t.Context(), isolated), "processAccountBalance must not error for an isolated margin account change")
	isolatedExp := accounts.SubAccounts{accounts.NewSubAccount(asset.Margin, "BTC-USDT")}
	isolatedExp[0].Balances.Set(currency.USDT, accounts.Balance{Total: 21.133773386762, Hold: 1.001, Free: 20.132773386762, UpdatedAt: time.UnixMilli(1730269283892)})
	assert.Equal(t, isolatedExp, (<-ex.Websocket.DataHandler.C).Data, "an isolated margin account change should be relayed as its symbol's margin balance")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	isolatedStored, err := ex.Accounts.GetBalance("BTC-USDT", creds, asset.Margin, currency.USDT)
	require.NoError(t, err, "the isolated margin balance must be stored")
	assert.Equal(t, 20.132773386762, isolatedStored.Free, "the stored isolated margin balance should be the pushed one")
	otherAccount := bytes.Replace(wsFixtureData(t, "wsSpot.json", "balanceMain"), []byte(`"main.deposit"`), []byte(`"other"`), 1)
	require.NoError(t, ex.processAccountBalance(t.Context(), otherAccount), "processAccountBalance must not error for an account without a sub-account")
	otherPush, ok := (<-ex.Websocket.DataHandler.C).Data.(*WebsocketAccountBalance)
	require.True(t, ok, "an account without a sub-account must be relayed as it comes")
	assert.Equal(t, "other", otherPush.RelationEvent, "an account without a sub-account should be relayed as it comes")

	require.Error(t, ex.processAccountBalance(t.Context(), []byte(`{"total":[]}`)), "processAccountBalance must reject an undecodable push")
	require.ErrorIs(t, ex.processAccountBalance(t.Context(), []byte(`{"relationEvent":"vault.hold"}`)), errUnknownBalanceRelation, "processAccountBalance must reject an unknown account")
	ex.Accounts = nil
	require.ErrorIs(t, ex.processAccountBalance(t.Context(), wsFixtureData(t, "wsSpot.json", "balanceTrade")), common.ErrNilPointer, "processAccountBalance must return a save error")
	require.ErrorIs(t, ex.processAccountBalance(t.Context(), wsFixtureData(t, "wsSpot.json", "balanceMargin")), common.ErrNilPointer, "processAccountBalance must return the error reading the saved margin balance")
	ex.SkipAuthCheck = false // The mock instance skips credential checks
	ex.SetCredentials(&accounts.Credentials{})
	require.ErrorIs(t, ex.processAccountBalance(t.Context(), wsFixtureData(t, "wsSpot.json", "balanceMargin")), exchange.ErrCredentialsAreEmpty, "processAccountBalance must return a credentials error")

	// REST saves the main and trade accounts together and the cross margin liability; the pushes that follow keep to both
	ex = marketTestServerExchange(t, map[string]string{
		"/api/v1/accounts": `{"code":"200000","data":[
			{"id":"548674591753","currency":"USDT","type":"trade","balance":"21.5","available":"20.5","holds":"1"},
			{"id":"548674591753","currency":"BTC","type":"trade","balance":"0.5","available":"0.4","holds":"0.1"},
			{"id":"63355cd156298d0001b66e61","currency":"USDT","type":"main","balance":"10.75","available":"9.5","holds":"1.25"}]}`,
		"/api/v3/margin/accounts":   `{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"38.5","available":"20","hold":"18.5","liability":"12.5"}]}}`,
		"/api/v3/isolated/accounts": `{"code":"200000","data":{"assets":[]}}`,
	})
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		_, err := ex.UpdateAccountBalances(t.Context(), a)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
	}
	creds, err = ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored := func(a asset.Item, c currency.Code) accounts.Balance {
		t.Helper()
		b, err := ex.Accounts.GetBalance("", creds, a, c)
		require.NoErrorf(t, err, "the %s %s balance must be stored", a, c)
		return b
	}
	restBTC, restMargin := stored(asset.Spot, currency.BTC), stored(asset.Margin, currency.USDT)
	// REST stamps the balances it saves with their arrival time, so the pushes that follow its snapshot carry later times
	pushed := time.UnixMilli(time.Now().Add(time.Minute).UnixMilli())
	frame := func(fields string, at time.Time) []byte {
		return []byte(`{"currency":"USDT",` + fields + `,"relationEventId":"354689988084000","time":"` + strconv.FormatInt(at.UnixMilli(), 10) + `"}`)
	}
	for _, tc := range []struct {
		name, fields string
		at           time.Time
		exp          accounts.Balance
	}{
		{name: "a trade account push", fields: `"total":"21.5","available":"20","hold":"1.5","relationEvent":"trade.hold"`, at: pushed, exp: accounts.Balance{Currency: currency.USDT, Total: 32.25, Hold: 2.75, Free: 29.5, UpdatedAt: pushed}},
		{name: "a main account push", fields: `"total":"15.75","available":"14.5","hold":"1.25","relationEvent":"main.deposit"`, at: pushed.Add(time.Second), exp: accounts.Balance{Currency: currency.USDT, Total: 37.25, Hold: 2.75, Free: 34.5, UpdatedAt: pushed.Add(time.Second)}},
		{name: "the same main account push again", fields: `"total":"15.75","available":"14.5","hold":"1.25","relationEvent":"main.deposit"`, at: pushed.Add(time.Second), exp: accounts.Balance{Currency: currency.USDT, Total: 37.25, Hold: 2.75, Free: 34.5, UpdatedAt: pushed.Add(time.Second)}},
		{name: "an HF trade account push", fields: `"total":"20.5","available":"19.5","hold":"1","relationEvent":"trade_hf.setted"`, at: pushed.Add(2 * time.Second), exp: accounts.Balance{Currency: currency.USDT, Total: 36.25, Hold: 2.25, Free: 34, UpdatedAt: pushed.Add(2 * time.Second)}}, // codespell:ignore setted
	} {
		require.NoErrorf(t, ex.processAccountBalance(t.Context(), frame(tc.fields, tc.at)), "processAccountBalance must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, stored(asset.Spot, currency.USDT), "%s should replace its account's share of the spot balance alone", tc.name)
		assert.Equalf(t, restBTC, stored(asset.Spot, currency.BTC), "%s should leave the other currencies", tc.name)
	}

	err = ex.processAccountBalance(t.Context(), frame(`"total":"1","available":"1","hold":"0","relationEvent":"trade.hold"`, pushed.Add(-time.Second)))
	require.ErrorIs(t, err, errBalancePushOutOfSequence, "processAccountBalance must reject a push older than its account's last balance")
	// A push for one account older than the other account's last is still its own account's newest balance
	require.NoError(t, ex.processAccountBalance(t.Context(), frame(`"total":"16.75","available":"15.5","hold":"1.25","relationEvent":"main.deposit"`, pushed.Add(1500*time.Millisecond))), "processAccountBalance must not error for a main account push older than the trade account's")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 37.25, Hold: 2.25, Free: 35, UpdatedAt: pushed.Add(2 * time.Second)}, stored(asset.Spot, currency.USDT), "a main account push older than the trade account's last should still replace the main account's share")
	require.NoError(t, ex.processAccountBalance(t.Context(), frame(`"total":"15.75","available":"14.5","hold":"1.25","relationEvent":"main.deposit"`, pushed.Add(3*time.Second))), "processAccountBalance must not error for a main account push")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 36.25, Hold: 2.25, Free: 34, UpdatedAt: pushed.Add(3 * time.Second)}, stored(asset.Spot, currency.USDT), "a rejected push should not change its account's share")

	// A REST listing older than the pushes keeps the shares they left, and refreshes the currencies no push is newer for
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error when pushes are newer than its listing")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 36.25, Hold: 2.25, Free: 34, UpdatedAt: pushed.Add(3 * time.Second)}, stored(asset.Spot, currency.USDT), "a REST listing older than the pushes should keep their shares")
	btc := stored(asset.Spot, currency.BTC)
	assert.True(t, btc.UpdatedAt.After(restBTC.UpdatedAt), "a REST listing should refresh the currencies no push is newer for")
	restBTC.UpdatedAt = btc.UpdatedAt
	assert.Equal(t, restBTC, btc, "a REST listing should save the balances it lists for the currencies no push is newer for")
	require.NoError(t, ex.processAccountBalance(t.Context(), frame(`"total":"15.75","available":"14.5","hold":"1.25","relationEvent":"main.deposit"`, pushed.Add(4*time.Second))), "processAccountBalance must not error for a main account push")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 36.25, Hold: 2.25, Free: 34, UpdatedAt: pushed.Add(4 * time.Second)}, stored(asset.Spot, currency.USDT), "a push after an older REST listing should add the pushed trade share, not the listed one")

	// Each set of credentials has balances of its own
	other := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-api-key", Secret: "other-api-secret", ClientID: "other-passphrase"})
	require.NoError(t, ex.processAccountBalance(other, frame(`"total":"100","available":"100","hold":"0","relationEvent":"trade.hold"`, pushed.Add(5*time.Second))), "processAccountBalance must not error for other credentials")
	require.NoError(t, ex.processAccountBalance(t.Context(), frame(`"total":"15.75","available":"14.5","hold":"1.25","relationEvent":"main.deposit"`, pushed.Add(6*time.Second))), "processAccountBalance must not error for a main account push")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 36.25, Hold: 2.25, Free: 34, UpdatedAt: pushed.Add(6 * time.Second)}, stored(asset.Spot, currency.USDT), "a push should add the shares of its own credentials alone")
	otherCreds, err := ex.GetCredentials(other)
	require.NoError(t, err, "GetCredentials must not error for other credentials")
	otherSpot, err := ex.Accounts.GetBalance("", otherCreds, asset.Spot, currency.USDT)
	require.NoError(t, err, "the other credentials' spot balance must be stored")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 100, Free: 100, UpdatedAt: pushed.Add(5 * time.Second)}, otherSpot, "the other credentials' spot balance should be their own push alone")

	marginHold := `"total":"38.5","available":"19.5","hold":"19","relationEvent":"marginV2.hold"`
	err = ex.processAccountBalance(t.Context(), frame(marginHold, time.Now().Add(-time.Minute)))
	assert.ErrorContains(t, err, "out of sequence", "processAccountBalance should reject a margin push older than the REST snapshot")
	assert.Equal(t, restMargin, stored(asset.Margin, currency.USDT), "a margin push older than the REST snapshot should not replace its balance")
	require.NoError(t, ex.processAccountBalance(t.Context(), frame(marginHold, pushed)), "processAccountBalance must not error for a margin push")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 38.5, Hold: 19, Free: 19.5, Borrowed: 12.5, UpdatedAt: pushed}, stored(asset.Margin, currency.USDT), "a margin push should replace the margin balance and keep the liability REST saved")

	// REST polls while pushes arrive, which can land older than one already saved for their account
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			_, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
			assert.NoError(t, err, "UpdateAccountBalances should not error")
		})
		wg.Go(func() {
			if err := ex.processAccountBalance(t.Context(), frame(`"total":"15.75","available":"14.5","hold":"1.25","relationEvent":"main.deposit"`, time.Now().Add(time.Hour+time.Duration(i)*time.Millisecond))); err != nil {
				assert.ErrorIs(t, err, errBalancePushOutOfSequence, "processAccountBalance should only refuse a push older than its account's last balance")
			}
		})
	}
	wg.Wait()
}

// TestMarginBalanceSavesWaitForEachOther covers a margin push reading the saved liability and saving it back, which REST's
// margin snapshot must not land between, or the snapshot's liability is undone: both wait for marginBalanceMu
func TestMarginBalanceSavesWaitForEachOther(t *testing.T) {
	t.Parallel()
	ex := marketTestServerExchange(t, map[string]string{
		"/api/v3/margin/accounts":   `{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"38.5","available":"20","hold":"18.5","liability":"12.5"}]}}`,
		"/api/v3/isolated/accounts": `{"code":"200000","data":{"assets":[]}}`,
	})
	push := []byte(`{"currency":"USDT","total":"38.5","available":"19.5","hold":"19","relationEvent":"marginV2.hold","time":"` + strconv.FormatInt(time.Now().Add(time.Minute).UnixMilli(), 10) + `"}`)
	for _, tc := range []struct {
		name string
		save func() error
	}{
		{name: "REST margin snapshot", save: func() error { _, err := ex.UpdateAccountBalances(t.Context(), asset.Margin); return err }},
		{name: "margin balance push", save: func() error { return ex.processAccountBalance(t.Context(), push) }},
	} {
		ex.marginBalanceMu.Lock()
		done := make(chan error, 1)
		go func() { done <- tc.save() }()
		select {
		case err := <-done:
			ex.marginBalanceMu.Unlock()
			require.FailNowf(t, "save did not wait", "the %s must wait for the margin balance lock, but returned %v", tc.name, err)
		case <-time.After(100 * time.Millisecond):
		}
		ex.marginBalanceMu.Unlock()
		select {
		case err := <-done:
			require.NoErrorf(t, err, "the %s must not error once the margin balance lock is free", tc.name)
		case <-time.After(5 * time.Second):
			require.FailNowf(t, "save never finished", "the %s must finish once the margin balance lock is free", tc.name)
		}
	}
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	b, err := ex.Accounts.GetBalance("", creds, asset.Margin, currency.USDT)
	require.NoError(t, err, "the margin balance must be stored")
	assert.Equal(t, 12.5, b.Borrowed, "the margin push should keep the liability the REST snapshot saved")
}

func TestBalanceAccount(t *testing.T) {
	t.Parallel()
	for event, exp := range map[string]struct {
		a       asset.Item
		account string
	}{
		"trade.hold":                 {a: asset.Spot, account: "trade"},
		"trade_hf.setted":            {a: asset.Spot, account: "trade"}, // codespell:ignore setted
		"main.deposit":               {a: asset.Spot, account: "main"},
		"margin.transfer":            {a: asset.Margin},
		"marginV2.other":             {a: asset.Margin},
		"isolated_BTC-USDT.hold":     {a: asset.Margin, account: "BTC-USDT"},
		"isolatedV2_ETH-USDT.setted": {a: asset.Margin, account: "ETH-USDT"}, // codespell:ignore setted
		"other":                      {a: asset.Empty},
	} {
		a, account, err := balanceAccount(event)
		require.NoErrorf(t, err, "balanceAccount must not error for %s", event)
		assert.Equalf(t, exp.a, a, "balanceAccount should map %s to its account's asset", event)
		assert.Equalf(t, exp.account, account, "balanceAccount should map %s to its sub-account", event)
	}
	// An isolated margin account without a symbol would be saved over the cross margin account, sub-account ""
	for _, event := range []string{"vault.hold", "other.hold", "", "isolated_.hold", "isolatedV2_.hold"} {
		_, _, err := balanceAccount(event)
		require.ErrorIsf(t, err, errUnknownBalanceRelation, "balanceAccount must reject %q", event)
	}
}

func TestProcessStopOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: spotMarketAdvancedChannel, Authenticated: true})
		return
	}
	ex := wsTestInstance(t)
	for _, tc := range []struct {
		key        string
		a          asset.Item
		marginType margin.Type
		status     order.Status
	}{
		{key: "stopOrder", a: asset.Spot, status: order.Open},
		{key: "stopOrderMargin", a: asset.Margin, marginType: margin.Multi, status: order.Closed},
		{key: "stopOrderIsolated", a: asset.Margin, marginType: margin.Isolated, status: order.Cancelled},
	} {
		exp := &order.Detail{
			Price:        70000,
			TriggerPrice: 71000,
			Amount:       0.00007142,
			Exchange:     ex.Name,
			OrderID:      "vs93gpupfa48anof003u85mb",
			Type:         order.StopLimit,
			Side:         order.Buy,
			Status:       tc.status,
			AssetType:    tc.a,
			MarginType:   tc.marginType,
			Date:         time.UnixMilli(1742305928064),
			LastUpdated:  time.Unix(0, 1742305928091268493),
			Pair:         spotTestPair,
		}
		assert.Equalf(t, exp, wsPush(t, ex, "wsSpot.json", tc.key), "processStopOrder should relay %s as an order update", tc.key)
	}
	for _, tc := range []struct {
		name, orderPrice string
	}{
		{name: "without an order price"},
		{name: "with an empty order price", orderPrice: `"orderPrice":"",`},
		{name: "with a zero order price", orderPrice: `"orderPrice":"0",`},
	} {
		push := `{"orderId":"vs93gpupfa48anof003u85mb",` + tc.orderPrice + `"orderType":"stop","side":"buy","size":"0.00007142","stop":"loss","stopPrice":"71000","symbol":"BTC-USDT","tradeType":"TRADE","type":"open","createdAt":1742305928064,"ts":1742305928091268493}`
		require.NoErrorf(t, ex.processStopOrder(t.Context(), []byte(push)), "processStopOrder must not error for a stop order push %s", tc.name)
		exp := &order.Detail{
			TriggerPrice: 71000,
			Amount:       0.00007142,
			Exchange:     ex.Name,
			OrderID:      "vs93gpupfa48anof003u85mb",
			Type:         order.StopMarket,
			Side:         order.Buy,
			Status:       order.Open,
			AssetType:    asset.Spot,
			Date:         time.UnixMilli(1742305928064),
			LastUpdated:  time.Unix(0, 1742305928091268493),
			Pair:         spotTestPair,
		}
		assert.Equalf(t, exp, (<-ex.Websocket.DataHandler.C).Data, "processStopOrder should relay a stop order push %s as a stop market order", tc.name)
	}

	require.Error(t, ex.processStopOrder(t.Context(), []byte(`{"size":[]}`)), "processStopOrder must reject an undecodable push")
	require.ErrorIs(t, ex.processStopOrder(t.Context(), []byte(`{"symbol":"a"}`)), currency.ErrCreatingPair, "processStopOrder must reject a symbol that is not a pair")
	require.ErrorIs(t, ex.processStopOrder(t.Context(), []byte(`{"symbol":"BTC-USDT","orderType":"iceberg"}`)), order.ErrUnrecognisedOrderType, "processStopOrder must reject an unknown order type")
	require.ErrorIs(t, ex.processStopOrder(t.Context(), []byte(`{"symbol":"BTC-USDT","orderType":"stop","side":"sideways"}`)), order.ErrSideIsInvalid, "processStopOrder must reject an unknown side")
	require.ErrorIs(t, ex.processStopOrder(t.Context(), []byte(`{"symbol":"BTC-USDT","orderType":"stop","side":"buy","type":"expired"}`)), errUnknownStopOrderType, "processStopOrder must reject an unknown message type")
	require.ErrorIs(t, ex.processStopOrder(t.Context(), []byte(`{"symbol":"BTC-USDT","orderType":"stop","side":"buy","type":"open","tradeType":"OTC"}`)), errUnknownMarginTradeType, "processStopOrder must reject an unknown trade type")
}

func TestStopPushOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType  string
		orderPrice float64
		exp        order.Type
	}{
		{orderType: "stop", orderPrice: 70000, exp: order.StopLimit},
		{orderType: "stop", exp: order.StopMarket},
		{orderType: "limit", orderPrice: 70000, exp: order.Limit},
		{orderType: "market", exp: order.Market},
	} {
		got, err := stopPushOrderType(tc.orderType, tc.orderPrice)
		require.NoErrorf(t, err, "stopPushOrderType must not error for %s", tc.orderType)
		assert.Equalf(t, tc.exp, got, "stopPushOrderType should map %s with order price %v", tc.orderType, tc.orderPrice)
	}
	_, err := stopPushOrderType("iceberg", 0)
	require.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "stopPushOrderType must reject an unknown order type")
}

func TestStopOrderStatus(t *testing.T) {
	t.Parallel()
	for messageType, exp := range map[string]order.Status{"open": order.Open, "triggered": order.Closed, "cancel": order.Cancelled} {
		got, err := stopOrderStatus(messageType)
		require.NoErrorf(t, err, "stopOrderStatus must not error for %s", messageType)
		assert.Equalf(t, exp, got, "stopOrderStatus should map %s", messageType)
	}
	_, err := stopOrderStatus("expired")
	require.ErrorIs(t, err, errUnknownStopOrderType, "stopOrderStatus must reject an unknown message type")
}
