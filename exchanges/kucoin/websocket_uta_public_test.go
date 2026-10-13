package kucoin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// utaFixtureData decodes the data of each push in a websocket fixture into T
func utaFixtureData[T any](t *testing.T, path string) []T {
	t.Helper()
	lines := utaFixtureLines(t, path)
	out := make([]T, 0, len(lines))
	for i, line := range lines {
		var msg UTAWebsocketMessage
		require.NoErrorf(t, json.Unmarshal([]byte(line), &msg), "%s line %d must decode", path, i+1)
		var v T
		require.NoErrorf(t, json.Unmarshal(msg.Data, &v), "%s line %d data must decode", path, i+1)
		out = append(out, v)
	}
	return out
}

// utaMessage decodes a push frame, for tests passing pushes to their handlers directly
func utaMessage(t *testing.T, frame string) *UTAWebsocketMessage {
	t.Helper()
	msg := new(UTAWebsocketMessage)
	require.NoErrorf(t, json.Unmarshal([]byte(frame), msg), "json.Unmarshal must not error for push frame %s", frame)
	return msg
}

// utaFixtureMessages decodes each push of a websocket fixture, for tests passing pushes to their handlers directly
func utaFixtureMessages(t *testing.T, path string) []*UTAWebsocketMessage {
	t.Helper()
	lines := utaFixtureLines(t, path)
	msgs := make([]*UTAWebsocketMessage, len(lines))
	for i, line := range lines {
		msgs[i] = utaMessage(t, line)
	}
	return msgs
}

func TestUTAWebsocketTickerDecoding(t *testing.T) {
	t.Parallel()
	exp := []UTAWebsocketTicker{
		{
			Symbol:          "BTC-USDT",
			Sequence:        38625963537,
			BestBidPrice:    83403.3,
			BestBidSize:     0.68235477,
			BestAskPrice:    83403.4,
			BestAskSize:     0.53328968,
			LastTradedPrice: 83399.7,
			Side:            "SELL",
			LastTradedSize:  0.02395178,
			MatchTimestamp:  types.Time(time.Unix(0, 1791899343339000000)),
		},
		{
			Symbol:          "XBTUSDTM",
			Sequence:        1947310281969,
			BestBidPrice:    83327,
			BestBidSize:     299,
			BestAskPrice:    83327.1,
			BestAskSize:     586,
			LastTradedPrice: 83327,
			Side:            "sell",
			LastTradedSize:  1,
			MatchTimestamp:  types.Time(time.Unix(0, 1791899402314000000)),
		},
	}
	assert.Equal(t, exp, utaFixtureData[UTAWebsocketTicker](t, "testdata/wsUTATicker.json"), "ticker pushes should decode every field, telling upper and lower case keys apart")
}

func TestProcessUTATicker(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	stored := &ticker.Price{High: 84000.1, Low: 82000.2, BaseVolume: 1234.5, QuoteVolume: 102345678.9, Bid: 83000, Ask: 83000.1, Pair: utaSpotTestPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.Unix(1791899300, 0)}
	require.NoError(t, ticker.ProcessTicker(stored), "ProcessTicker must not error")
	for i, msg := range utaFixtureMessages(t, "testdata/wsUTATicker.json") {
		_, market, _ := strings.Cut(msg.Topic, ".")
		require.NoErrorf(t, ex.processUTATicker(t.Context(), msg, market), "processUTATicker must not error for fixture line %d", i+1)
	}
	spot := ticker.Price{Last: 83399.7, LastSize: 0.02395178, High: 84000.1, Low: 82000.2, Bid: 83403.3, BidSize: 0.68235477, Ask: 83403.4, AskSize: 0.53328968, BaseVolume: 1234.5, QuoteVolume: 102345678.9, Pair: utaSpotTestPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.Unix(0, 1791899343339000000)}
	exp := []any{
		[]ticker.Price{
			spot,
			{Last: 83399.7, LastSize: 0.02395178, Bid: 83403.3, BidSize: 0.68235477, Ask: 83403.4, AskSize: 0.53328968, Pair: utaSpotTestPair, ExchangeName: ex.Name, AssetType: asset.Margin, LastUpdated: time.Unix(0, 1791899343339000000)},
		},
		[]ticker.Price{
			{Last: 83327, LastSize: 1, Bid: 83327, BidSize: 299, Ask: 83327.1, AskSize: 586, Pair: utaFuturesTestPair, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: time.Unix(0, 1791899402314000000)},
		},
	}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTATicker should store and relay the tickers, keeping the fields a push lacks")
	got, err := ticker.GetTicker(ex.Name, utaSpotTestPair, asset.Spot)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, spot, *got, "processUTATicker should store the spot ticker it relays")

	err = ex.processUTATicker(t.Context(), utaMessage(t, `{"T":"ticker.SPOT", "P":1791899343343102882, "d":{"A":"1","B":"1","E":38625963538,"M":1791899343339000000,"S":"SELL","a":"83403.4","b":"83403.4","l":"83399.7","q":"0.02395178","s":"BTC-USDT"}}`), "SPOT")
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "processUTATicker should return the tickers that fail processing")
	assert.Empty(t, drainUTARelay(ex), "processUTATicker should not relay an empty batch")
	err = ex.processUTATicker(t.Context(), utaMessage(t, `{"T":"ticker.FUTURES", "P":1791899402317469972, "d":{"s":"NOTACONTRACTM"}}`), "FUTURES")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "processUTATicker should error for an unknown futures contract")
	assert.Error(t, ex.processUTATicker(t.Context(), utaMessage(t, `{"T":"ticker.SPOT", "P":1791899343343102882, "d":[]}`), "SPOT"), "processUTATicker should error for malformed data")
	ex.Name = ""
	err = ex.processUTATicker(t.Context(), utaMessage(t, `{"T":"ticker.SPOT", "P":1791899343343102882, "d":{"s":"BTC-USDT"}}`), "SPOT")
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet, "processUTATicker should return the stored ticker's lookup error")
}

func TestStoredUTATicker(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	got, err := ex.storedUTATicker(utaSpotTestPair, asset.Margin)
	require.NoError(t, err, "storedUTATicker must not error without a stored ticker")
	assert.Equal(t, &ticker.Price{ExchangeName: ex.Name, Pair: utaSpotTestPair, AssetType: asset.Margin}, got, "storedUTATicker should start a ticker when none is stored")

	stored := &ticker.Price{Last: 83327, High: 84000, Pair: utaFuturesTestPair, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: time.Unix(1791899300, 0)}
	require.NoError(t, ticker.ProcessTicker(stored), "ProcessTicker must not error")
	got, err = ex.storedUTATicker(utaFuturesTestPair, asset.Futures)
	require.NoError(t, err, "storedUTATicker must not error")
	assert.Equal(t, stored, got, "storedUTATicker should return the stored ticker")

	ex.Name = ""
	_, err = ex.storedUTATicker(utaFuturesTestPair, asset.Futures)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet, "storedUTATicker should return lookup errors other than a missing ticker")
}

func TestUTAWebsocketKlineDecoding(t *testing.T) {
	t.Parallel()
	exp := []UTAWebsocketKline{
		{Interval: "1min", Symbol: "BTC-USDT", StartTime: types.Time(time.Unix(1791899340, 0)), EndTime: types.Time(time.Unix(1791899400, 0)), OpenPrice: 83403.3, ClosePrice: 83403.4, HighPrice: 83403.4, LowPrice: 83397.5, TransactionVolume: 0.0194076, TransactionAmount: 1618.627552576},
		{Interval: "1min", Symbol: "XBTUSDTM", StartTime: types.Time(time.Unix(1791899340, 0)), EndTime: types.Time(time.Unix(1791899400, 0)), OpenPrice: 83365.4, ClosePrice: 83312.2, HighPrice: 83376.5, LowPrice: 83306.9, TransactionVolume: 2146, TransactionAmount: 178861.6938},
		{Interval: "1min", Symbol: "XBTUSDTM", StartTime: types.Time(time.Unix(1791899400, 0)), EndTime: types.Time(time.Unix(1791899460, 0)), OpenPrice: 83327, ClosePrice: 83327, HighPrice: 83327, LowPrice: 83327, TransactionVolume: 2, TransactionAmount: 166.654, StartIndicator: true},
	}
	assert.Equal(t, exp, utaFixtureData[UTAWebsocketKline](t, "testdata/wsUTAKline.json"), "kline pushes should decode every field, telling upper and lower case keys apart")
}

func TestProcessUTAKline(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	msgs := utaFixtureMessages(t, "testdata/wsUTAKline.json")
	for i, msg := range msgs {
		_, market, _ := strings.Cut(msg.Topic, ".")
		require.NoErrorf(t, ex.processUTAKline(t.Context(), msg, market), "processUTAKline must not error for fixture line %d", i+1)
	}
	spotCandle := []kline.Candle{{Time: time.Unix(1791899340, 0), Open: 83403.3, High: 83403.4, Low: 83397.5, Close: 83403.4, Volume: 0.0194076, QuoteVolume: 1618.627552576}}
	exp := []any{
		&kline.Item{Exchange: ex.Name, Pair: utaSpotTestPair, Asset: asset.Spot, Interval: kline.OneMin, Candles: spotCandle},
		&kline.Item{Exchange: ex.Name, Pair: utaSpotTestPair, Asset: asset.Margin, Interval: kline.OneMin, Candles: spotCandle},
		&kline.Item{Exchange: ex.Name, Pair: utaFuturesTestPair, Asset: asset.Futures, Interval: kline.OneMin, Candles: []kline.Candle{{Time: time.Unix(1791899340, 0), Open: 83365.4, High: 83376.5, Low: 83306.9, Close: 83312.2, Volume: 2146, QuoteVolume: 178861.6938}}},
		&kline.Item{Exchange: ex.Name, Pair: utaFuturesTestPair, Asset: asset.Futures, Interval: kline.OneMin, Candles: []kline.Candle{{Time: time.Unix(1791899400, 0), Open: 83327, High: 83327, Low: 83327, Close: 83327, Volume: 2, QuoteVolume: 166.654}}},
	}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTAKline should relay a candle for each asset the symbol trades as")

	ex.Websocket.DataHandler = stream.NewRelay(1)
	require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "filler"), "Send must not error")
	err := ex.processUTAKline(t.Context(), msgs[0], "SPOT")
	assert.Error(t, err, "processUTAKline should return the error relaying a candle")
	err = ex.processUTAKline(t.Context(), utaMessage(t, `{"T":"kline.SPOT", "P":1791899366825249413, "d":{"s":"BTC-USDT","i":"7min"}}`), "SPOT")
	assert.ErrorIs(t, err, kline.ErrInvalidInterval, "processUTAKline should error for an unknown interval")
	err = ex.processUTAKline(t.Context(), utaMessage(t, `{"T":"kline.FUTURES", "P":1791899366825249413, "d":{"s":"NOTACONTRACTM","i":"1min"}}`), "FUTURES")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "processUTAKline should error for an unknown futures contract")
	assert.Error(t, ex.processUTAKline(t.Context(), utaMessage(t, `{"T":"kline.SPOT", "P":1791899366825249413, "d":[]}`), "SPOT"), "processUTAKline should error for malformed data")
}

func TestUTAIntervalFromString(t *testing.T) {
	t.Parallel()
	for exp, name := range utaKlineIntervals {
		got, err := utaIntervalFromString(name)
		require.NoErrorf(t, err, "utaIntervalFromString must not error for %s", name)
		assert.Equalf(t, exp, got, "utaIntervalFromString should return the interval %s names", name)
	}
	_, err := utaIntervalFromString("7min")
	assert.ErrorIs(t, err, kline.ErrInvalidInterval, "utaIntervalFromString should error for an unknown interval")
}

func TestUTAWebsocketTradeDecoding(t *testing.T) {
	t.Parallel()
	exp := []UTAWebsocketTrade{
		{Symbol: "BTC-USDT", Sequence: 24806820952031232, TradeID: "24806820952031232", Price: 83403.4, Size: 0.00221691, Side: "sell", MatchTimestamp: types.Time(time.Unix(0, 1791899366799000000))},
		{Symbol: "XBTUSDTM", Sequence: 1947310281969, TradeID: "1947310281969", Price: 83327, Size: 1, Side: "sell", RPI: true, MatchTimestamp: types.Time(time.Unix(0, 1791899402314000000))},
	}
	assert.Equal(t, exp, utaFixtureData[UTAWebsocketTrade](t, "testdata/wsUTATrade.json"), "trade pushes should decode every field, telling upper and lower case keys apart")
}

func TestProcessUTATrade(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	newUTAWebsocket(t, ex)
	msgs := utaFixtureMessages(t, "testdata/wsUTATrade.json")
	for i, msg := range msgs {
		_, market, _ := strings.Cut(msg.Topic, ".")
		require.NoErrorf(t, ex.processUTATrade(msg, market), "processUTATrade must not error for fixture line %d while trades are neither saved nor fed", i+1)
	}
	assert.Empty(t, drainUTARelay(ex), "processUTATrade should relay nothing while trades are neither saved nor fed")

	ex.SetTradeFeedStatus(true)
	for i, msg := range msgs {
		_, market, _ := strings.Cut(msg.Topic, ".")
		require.NoErrorf(t, ex.processUTATrade(msg, market), "processUTATrade must not error for fixture line %d", i+1)
	}
	spot := trade.Data{TID: "24806820952031232", Exchange: ex.Name, CurrencyPair: utaSpotTestPair, AssetType: asset.Spot, Side: order.Sell, Price: 83403.4, Amount: 0.00221691, Timestamp: time.Unix(0, 1791899366799000000)}
	margin := spot
	margin.AssetType = asset.Margin
	exp := []any{
		[]trade.Data{spot, margin},
		[]trade.Data{{TID: "1947310281969", Exchange: ex.Name, CurrencyPair: utaFuturesTestPair, AssetType: asset.Futures, Side: order.Sell, Price: 83327, Amount: 1, Timestamp: time.Unix(0, 1791899402314000000)}},
	}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTATrade should relay a trade for each asset the symbol trades as")

	err := ex.processUTATrade(utaMessage(t, `{"T":"trade.SPOT", "P":1791899366800116304, "d":{"S":"sideways","s":"BTC-USDT"}}`), "SPOT")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "processUTATrade should error for an unknown side")
	err = ex.processUTATrade(utaMessage(t, `{"T":"trade.FUTURES", "P":1791899366800116304, "d":{"S":"buy","s":"NOTACONTRACTM"}}`), "FUTURES")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "processUTATrade should error for an unknown futures contract")
	assert.Error(t, ex.processUTATrade(utaMessage(t, `{"T":"trade.SPOT", "P":1791899366800116304, "d":[]}`), "SPOT"), "processUTATrade should error for malformed data")
}

func TestUTAWebsocketOrderbookDecoding(t *testing.T) {
	t.Parallel()
	exp := UTAWebsocketOrderbook{
		SequenceStart:  1738649731005,
		SequenceEnd:    1738649731005,
		MatchTimestamp: types.Time(time.Unix(0, 1791899416980000000)),
		Symbol:         "ETHUSDTM",
		Bids: []UTAWebsocketOrderbookLevel{
			{Price: 2508.97, Size: 102}, {Price: 2508.87, Size: 139}, {Price: 2508.86, RPISize: 718}, {Price: 2508.75, Size: 50}, {Price: 2508.7, Size: 21},
		},
		Asks: []UTAWebsocketOrderbookLevel{
			{Price: 2508.98, Size: 666}, {Price: 2509.01, Size: 36}, {Price: 2509.08, Size: 129}, {Price: 2509.09, Size: 327, RPISize: 1736}, {Price: 2509.1, Size: 36, RPISize: 126},
		},
	}
	assert.Equal(t, exp, utaFixtureData[UTAWebsocketOrderbook](t, "testdata/wsUTAOrderbook.json")[3], "an orderbook push including retail price improvement orders should decode every field")
}

func TestUTAWebsocketOrderbookLevelUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var l UTAWebsocketOrderbookLevel
	require.NoError(t, json.Unmarshal([]byte(`["2509.09","327","1736"]`), &l), "UnmarshalJSON must not error for three values")
	assert.Equal(t, UTAWebsocketOrderbookLevel{Price: 2509.09, Size: 327, RPISize: 1736}, l, "UnmarshalJSON should decode the price, size and RPI size")
	require.NoError(t, json.Unmarshal([]byte(`["83779.1","0.40597618"]`), &l), "UnmarshalJSON must not error for two values")
	assert.Equal(t, UTAWebsocketOrderbookLevel{Price: 83779.1, Size: 0.40597618}, l, "UnmarshalJSON should decode the price and size, clearing an earlier RPI size")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["83779.1"]`), &l), common.ErrMalformedData, "UnmarshalJSON should error for one value")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["1","2","3","4"]`), &l), common.ErrMalformedData, "UnmarshalJSON should error for four values")
	assert.Error(t, json.Unmarshal([]byte(`{"price":"1"}`), &l), "UnmarshalJSON should error for an object")
}

func TestProcessUTAOrderbook(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	ex.ValidateOrderbook = true
	ethPair := currency.NewPairWithDelimiter("ETH", "USDTM", currency.UnderscoreDelimiter)
	for i, msg := range utaFixtureMessages(t, "testdata/wsUTAOrderbook.json") {
		_, market, _ := strings.Cut(msg.Topic, ".")
		require.NoErrorf(t, ex.processUTAOrderbook(t.Context(), nil, msg, market), "processUTAOrderbook must not error for fixture line %d", i+1)
	}
	assert.Len(t, drainUTARelay(ex), 10, "processUTAOrderbook should relay the orderbook on each snapshot and delta, for spot and margin alike")
	spotBook := func(a asset.Item) *orderbook.Book {
		return &orderbook.Book{
			Bids:              orderbook.Levels{{Price: 83779, Amount: 0.03430859}, {Price: 83778.8, Amount: 0.02242723}, {Price: 83778.7, Amount: 0.00004322}, {Price: 83775.2, Amount: 0.03065}},
			Asks:              orderbook.Levels{{Price: 83779.1, Amount: 0.15034365}, {Price: 83779.2, Amount: 0.00304322}, {Price: 83779.3, Amount: 0.0035}, {Price: 83789.5, Amount: 0.2067}},
			Exchange:          ex.Name,
			Pair:              utaSpotTestPair,
			Asset:             a,
			LastUpdated:       time.Unix(0, 1791901037327000000),
			LastPushed:        time.Unix(0, 1791901037339674554),
			LastUpdateID:      38627190496,
			ValidateOrderbook: true,
		}
	}
	for _, exp := range []*orderbook.Book{
		spotBook(asset.Spot),
		spotBook(asset.Margin),
		{
			Bids:              orderbook.Levels{{Price: 2508.97, Amount: 102}, {Price: 2508.87, Amount: 139}, {Price: 2508.75, Amount: 50}, {Price: 2508.7, Amount: 21}},
			Asks:              orderbook.Levels{{Price: 2508.98, Amount: 666}, {Price: 2509.01, Amount: 36}, {Price: 2509.08, Amount: 129}, {Price: 2509.09, Amount: 327}, {Price: 2509.1, Amount: 36}},
			Exchange:          ex.Name,
			Pair:              ethPair,
			Asset:             asset.Futures,
			LastUpdated:       time.Unix(0, 1791899416980000000),
			LastPushed:        time.Unix(0, 1791899416995814336),
			LastUpdateID:      1738649731005,
			ValidateOrderbook: true,
		},
		{
			Bids:              orderbook.Levels{{Price: 83757.5, Amount: 295}, {Price: 83754.5, Amount: 11}, {Price: 83751.4, Amount: 39}},
			Asks:              orderbook.Levels{{Price: 83757.6, Amount: 168}, {Price: 83761.8, Amount: 88}, {Price: 83763.3, Amount: 77}, {Price: 83763.6, Amount: 41}, {Price: 83770.5, Amount: 72}, {Price: 84144.7, Amount: 63}},
			Exchange:          ex.Name,
			Pair:              utaFuturesTestPair,
			Asset:             asset.Futures,
			LastUpdated:       time.Unix(0, 1791901030071000000),
			LastPushed:        time.Unix(0, 1791901030080569976),
			LastUpdateID:      1748392655996,
			ValidateOrderbook: true,
		},
	} {
		got, err := ex.Websocket.Orderbook.GetOrderbook(exp.Pair, exp.Asset)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s %s", exp.Pair, exp.Asset)
		assert.Falsef(t, got.InsertedAt.IsZero(), "the %s %s orderbook should record when it was inserted", exp.Pair, exp.Asset)
		got.InsertedAt = time.Time{}
		assert.Equalf(t, exp, got, "processUTAOrderbook should build the %s %s orderbook from its snapshot and deltas, without retail price improvement orders", exp.Pair, exp.Asset)
	}

	err := ex.processUTAOrderbook(t.Context(), nil, utaMessage(t, `{"T":"obu.SPOT", "dp":"5", "t":"partial", "P":1791901035798337832, "d":{"C":1,"M":1791901035787000000,"O":1,"a":[],"b":[],"s":"BTC-USDT"}}`), "SPOT")
	assert.ErrorIs(t, err, errUTAUnknownTopic, "processUTAOrderbook should error for an unknown update type")
	err = ex.processUTAOrderbook(t.Context(), nil, utaMessage(t, `{"T":"obu.SPOT", "dp":"5", "t":"snapshot", "P":1791901035798337832, "d":{"C":1,"M":1791901035787000000,"O":1,"a":[["83779.1","1"]],"b":[["83778","1"],["83779","1"]],"s":"BTC-USDT"}}`), "SPOT")
	assert.Error(t, err, "processUTAOrderbook should error for a snapshot with bids out of order")
	err = ex.processUTAOrderbook(t.Context(), nil, utaMessage(t, `{"T":"obu.FUTURES", "dp":"5", "t":"snapshot", "P":1791901035798337832, "d":{"s":"NOTACONTRACTM"}}`), "FUTURES")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "processUTAOrderbook should error for an unknown futures contract")
	assert.Error(t, ex.processUTAOrderbook(t.Context(), nil, utaMessage(t, `{"T":"obu.SPOT", "dp":"5", "t":"snapshot", "P":1791901035798337832, "d":{"a":[["1"]]}}`), "SPOT"), "processUTAOrderbook should error for a malformed level")

	t.Run("sequence gap", func(t *testing.T) {
		t.Parallel()
		ex := newUTATestExchange(t)
		var mu sync.Mutex
		var actions []string
		wsURL := newUTAMockServer(t, true, func(msg map[string]any) []string {
			action, _ := msg["action"].(string)
			mu.Lock()
			actions = append(actions, action)
			mu.Unlock()
			return []string{utaAck(msg)}
		})
		newUTAWebsocket(t, ex, ex.newUTAFuturesPublicConnection(ex.Config, wsURL))
		ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 500}}
		require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
		conn, err := ex.Websocket.GetConnection(utaFuturesPublicConnection)
		require.NoError(t, err, "GetConnection must not error")
		snapshot := utaMessage(t, `{"T":"obu.FUTURES", "dp":"increment@10ms", "t":"snapshot", "P":1791901029951262471, "d":{"a":[["83757.6","168"]],"b":[["83757.5","295"]],"s":"XBTUSDTM","C":100,"M":1791901029947000000,"O":100}}`)
		require.NoError(t, ex.processUTAOrderbook(t.Context(), conn, snapshot, "FUTURES"), "processUTAOrderbook must not error for a snapshot")
		gap := utaMessage(t, `{"T":"obu.FUTURES", "dp":"increment@10ms", "t":"delta", "P":1791901030016290579, "d":{"a":[["84144.7","63"]],"b":[],"s":"XBTUSDTM","C":106,"M":1791901030005000000,"O":105}}`)
		err = ex.processUTAOrderbook(t.Context(), conn, gap, "FUTURES")
		assert.ErrorIs(t, err, errUTAOrderbookSequenceGap, "processUTAOrderbook should report a delta skipping sequences")
		// The resubscription runs in the background; it is complete once the orderbook is subscribed to again
		assert.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			subs := conn.Subscriptions().List()
			return len(actions) >= 3 && len(subs) == 1 && subs[0].State() == subscription.SubscribedState
		}, 10*time.Second, 10*time.Millisecond, "processUTAOrderbook should resubscribe to the orderbook")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{utaSubscribeAction, utaUnsubscribeAction, utaSubscribeAction}, actions, "processUTAOrderbook should unsubscribe from and subscribe to the orderbook again, for a fresh snapshot")
	})
}

func TestApplyUTAOrderbookDelta(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	msg := &UTAWebsocketMessage{PushTimestamp: types.Time(time.Unix(0, 1791901030016290579))}
	delta := &UTAWebsocketOrderbook{SequenceStart: 101, SequenceEnd: 103, MatchTimestamp: types.Time(time.Unix(0, 1791901030005000000)), Asks: []UTAWebsocketOrderbookLevel{{Price: 101, Size: 2}}}
	resync, err := ex.applyUTAOrderbookDelta(t.Context(), msg, delta, utaFuturesTestPair, asset.Futures)
	assert.NoError(t, err, "applyUTAOrderbookDelta should drop a delta for a missing orderbook without error")
	assert.True(t, resync, "applyUTAOrderbookDelta should ask for a fresh snapshot for a missing orderbook")
	resync, err = ex.applyUTAOrderbookDelta(t.Context(), msg, delta, currency.EMPTYPAIR, asset.Futures)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "applyUTAOrderbookDelta should return the error looking up the orderbook")
	assert.False(t, resync, "applyUTAOrderbookDelta should not ask for a fresh snapshot when the lookup fails")

	loadSnapshot := func() {
		require.NoError(t, ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
			Exchange:          ex.Name,
			Pair:              utaFuturesTestPair,
			Asset:             asset.Futures,
			Bids:              orderbook.Levels{{Price: 99, Amount: 1}},
			Asks:              orderbook.Levels{{Price: 100, Amount: 1}},
			LastUpdated:       time.Unix(1791901029, 0),
			LastUpdateID:      100,
			ValidateOrderbook: true,
		}), "LoadSnapshot must not error")
	}
	loadSnapshot()
	resync, err = ex.applyUTAOrderbookDelta(t.Context(), msg, delta, utaFuturesTestPair, asset.Futures)
	require.NoError(t, err, "applyUTAOrderbookDelta must not error for a delta following on")
	assert.False(t, resync, "applyUTAOrderbookDelta should not ask for a fresh snapshot for a delta following on")
	stale := &UTAWebsocketOrderbook{SequenceStart: 90, SequenceEnd: 95, MatchTimestamp: types.Time(time.Unix(0, 1791901030000000000)), Asks: []UTAWebsocketOrderbookLevel{{Price: 100, Size: 0}}}
	resync, err = ex.applyUTAOrderbookDelta(t.Context(), msg, stale, utaFuturesTestPair, asset.Futures)
	require.NoError(t, err, "applyUTAOrderbookDelta must not error for a delta already held")
	assert.False(t, resync, "applyUTAOrderbookDelta should not ask for a fresh snapshot for a delta already held")
	book, err := ex.Websocket.Orderbook.GetOrderbook(utaFuturesTestPair, asset.Futures)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, orderbook.Levels{{Price: 100, Amount: 1}, {Price: 101, Amount: 2}}, book.Asks, "applyUTAOrderbookDelta should apply the delta following on and skip the one already held")
	assert.Equal(t, int64(103), book.LastUpdateID, "applyUTAOrderbookDelta should move the orderbook to the delta's last sequence")

	gap := &UTAWebsocketOrderbook{SequenceStart: 105, SequenceEnd: 106, MatchTimestamp: types.Time(time.Unix(0, 1791901030100000000)), Asks: []UTAWebsocketOrderbookLevel{{Price: 102, Size: 1}}}
	resync, err = ex.applyUTAOrderbookDelta(t.Context(), msg, gap, utaFuturesTestPair, asset.Futures)
	assert.ErrorIs(t, err, errUTAOrderbookSequenceGap, "applyUTAOrderbookDelta should error for a delta skipping sequences")
	assert.True(t, resync, "applyUTAOrderbookDelta should ask for a fresh snapshot once changes were missed")
	_, err = ex.Websocket.Orderbook.GetOrderbook(utaFuturesTestPair, asset.Futures)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "applyUTAOrderbookDelta should invalidate an orderbook that missed changes")
	resync, err = ex.applyUTAOrderbookDelta(t.Context(), msg, gap, utaFuturesTestPair, asset.Futures)
	assert.NoError(t, err, "applyUTAOrderbookDelta should drop a delta to an invalidated orderbook without error")
	assert.True(t, resync, "applyUTAOrderbookDelta should keep asking for a fresh snapshot for an invalidated orderbook")

	loadSnapshot()
	zeroPrice := &UTAWebsocketOrderbook{SequenceStart: 101, SequenceEnd: 101, MatchTimestamp: types.Time(time.Unix(0, 1791901030200000000)), Bids: []UTAWebsocketOrderbookLevel{{Price: 0, Size: 1}}}
	resync, err = ex.applyUTAOrderbookDelta(t.Context(), msg, zeroPrice, utaFuturesTestPair, asset.Futures)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "applyUTAOrderbookDelta should return the error of a delta leaving the orderbook invalid")
	assert.True(t, resync, "applyUTAOrderbookDelta should ask for a fresh snapshot once a delta leaves the orderbook invalid")
}

func TestResyncUTAOrderbook(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	store := subscription.NewStore()
	tickerSub := &subscription.Subscription{Channel: UTATickerChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}}
	otherPair := &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{currency.NewPairWithDelimiter("ETH", "USDTM", currency.UnderscoreDelimiter)}, Levels: 500}
	resubscribing := &subscription.Subscription{Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 500}
	for _, s := range []*subscription.Subscription{tickerSub, otherPair, resubscribing} {
		require.NoError(t, s.SetState(subscription.SubscribedState), "SetState must not error")
		require.NoError(t, store.Add(s), "Add must not error")
	}
	require.NoError(t, resubscribing.SetState(subscription.ResubscribingState), "SetState must not error")
	ex.resyncUTAOrderbook(t.Context(), &utaFakeConnection{subs: store}, utaFuturesTestPair)
	for _, s := range []*subscription.Subscription{tickerSub, otherPair} {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "resyncUTAOrderbook should leave %s alone", s)
	}
	assert.Equal(t, subscription.ResubscribingState, resubscribing.State(), "resyncUTAOrderbook should leave a resubscription under way alone")
}

// newUTAResyncServer serves an orderbook subscription whose deltas skip sequences: after the first snapshot it pushes
// gapped deltas, as KuCoin keeps pushing deltas, until the client unsubscribes, and answers a second subscription with a
// fresh snapshot. It returns the server's URL and the subscription actions it received
func newUTAResyncServer(t *testing.T, rejectUnsubscribe bool) (wsURL string, actions func() []string) {
	t.Helper()
	frame := func(updateType string, start, end int64) []byte {
		return fmt.Appendf(nil, `{"T":"obu.FUTURES", "dp":"increment@10ms", "t":"%s", "P":1791901029951262471, "d":{"a":[["83757.6","168"]],"b":[["83757.5","295"]],"s":"XBTUSDTM","C":%d,"M":1791901029947000000,"O":%d}}`, updateType, end, start)
	}
	var mu sync.Mutex
	var received []string
	upgrader := gws.Upgrader{}
	s := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if !assert.NoError(t, err, "Upgrade should not error") {
			return
		}
		defer c.Close()
		var writeMu sync.Mutex
		write := func(frames ...[]byte) {
			writeMu.Lock()
			defer writeMu.Unlock()
			for _, f := range frames {
				if err := c.WriteMessage(gws.BinaryMessage, f); err != nil {
					return
				}
			}
		}
		stopDeltas := make(chan struct{})
		var stopOnce sync.Once
		defer stopOnce.Do(func() { close(stopDeltas) })
		for {
			_, p, err := c.ReadMessage()
			if err != nil {
				return
			}
			var msg UTAWebsocketSubscriptionRequest
			if !assert.NoError(t, json.Unmarshal(p, &msg), "client message should be a subscription request") {
				return
			}
			mu.Lock()
			received = append(received, msg.Action)
			subscribes := 0
			for _, a := range received {
				if a == utaSubscribeAction {
					subscribes++
				}
			}
			mu.Unlock()
			ack := []byte(`{"id":"` + msg.ID + `","result":true}`)
			switch {
			case msg.Action == utaUnsubscribeAction:
				stopOnce.Do(func() { close(stopDeltas) })
				if rejectUnsubscribe {
					ack = []byte(`{"id":"` + msg.ID + `","result":false,"message":"invalid request data"}`)
				}
				write(ack)
			case subscribes == 1:
				write(ack, frame("snapshot", 100, 100))
				go func() {
					for seq := int64(105); ; seq += 2 {
						select {
						case <-stopDeltas:
							return
						case <-time.After(20 * time.Millisecond):
							write(frame("delta", seq, seq+1))
						}
					}
				}()
			default:
				write(ack, frame("snapshot", 200, 200))
			}
		}
	}))
	s.Start()
	return "ws" + strings.TrimPrefix(s.URL, "http"), func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(received)
	}
}

// TestUTAOrderbookResync proves that an orderbook missing changes is resubscribed to and rebuilt from the fresh snapshot
// KuCoin pushes on subscribing, and that a failed resubscription leaves it invalid
func TestUTAOrderbookResync(t *testing.T) {
	t.Parallel()
	for _, rejectUnsubscribe := range []bool{false, true} {
		t.Run(fmt.Sprintf("rejected unsubscribe %t", rejectUnsubscribe), func(t *testing.T) {
			t.Parallel()
			ex := newUTATestExchange(t)
			wsURL, actions := newUTAResyncServer(t, rejectUnsubscribe)
			newUTAWebsocket(t, ex, ex.newUTAFuturesPublicConnection(ex.Config, wsURL))
			ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 500}}
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			expActions := []string{utaSubscribeAction, utaUnsubscribeAction, utaSubscribeAction}
			if rejectUnsubscribe {
				expActions = expActions[:2]
			}
			assert.Eventually(t, func() bool { return len(actions()) >= len(expActions) }, 10*time.Second, 10*time.Millisecond, "the orderbook subscription should be resubscribed to once the orderbook misses changes")
			assert.Equal(t, expActions, actions(), "the connection should unsubscribe and then subscribe again")
			if rejectUnsubscribe {
				_, err := ex.Websocket.Orderbook.GetOrderbook(utaFuturesTestPair, asset.Futures)
				assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the orderbook should stay invalid when the resubscription fails")
				return
			}
			assert.Eventually(t, func() bool {
				book, err := ex.Websocket.Orderbook.GetOrderbook(utaFuturesTestPair, asset.Futures)
				return err == nil && book.LastUpdateID == 200
			}, 10*time.Second, 10*time.Millisecond, "the orderbook should be rebuilt from the fresh snapshot")
		})
	}
}

func TestUTAOrderbookLevels(t *testing.T) {
	t.Parallel()
	levels := []UTAWebsocketOrderbookLevel{{Price: 2509.09, Size: 327, RPISize: 1736}, {Price: 2509.1, RPISize: 126}}
	assert.Equal(t, orderbook.Levels{{Price: 2509.09, Amount: 327}}, utaOrderbookLevels(levels, true), "utaOrderbookLevels should drop a snapshot level holding retail price improvement orders alone")
	assert.Equal(t, orderbook.Levels{{Price: 2509.09, Amount: 327}, {Price: 2509.1}}, utaOrderbookLevels(levels, false), "utaOrderbookLevels should keep a delta's zero size, which removes the level")
}

func TestUTAWebsocketMarkPriceDecoding(t *testing.T) {
	t.Parallel()
	exp := []UTAWebsocketMarkPrice{{Symbol: "XBTUSDTM", MarkPrice: 83334.39, IndexPrice: 83370.16, OpenInterest: 9511786, Timestamp: types.Time(time.UnixMilli(1791899401000))}}
	assert.Equal(t, exp, utaFixtureData[UTAWebsocketMarkPrice](t, "testdata/wsUTAMarkPrice.json"), "mark price pushes should decode every field")
}

func TestProcessUTAMarkPrice(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	stored := &ticker.Price{Last: 83327, Bid: 83327, Ask: 83327.1, Pair: utaFuturesTestPair, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: time.Unix(0, 1791899400314000000)}
	require.NoError(t, ticker.ProcessTicker(stored), "ProcessTicker must not error")
	require.NoError(t, ex.processUTAMarkPrice(t.Context(), utaFixtureMessages(t, "testdata/wsUTAMarkPrice.json")[0]), "processUTAMarkPrice must not error")
	exp := &ticker.Price{Last: 83327, Bid: 83327, Ask: 83327.1, MarkPrice: 83334.39, IndexPrice: 83370.16, OpenInterest: 9511786, Pair: utaFuturesTestPair, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: time.UnixMilli(1791899401000)}
	assert.Equal(t, []any{exp}, drainUTARelay(ex), "processUTAMarkPrice should relay the mark price, index price and open interest, keeping the ticker's other fields")
	got, err := ticker.GetTicker(ex.Name, utaFuturesTestPair, asset.Futures)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, exp, got, "processUTAMarkPrice should store the ticker it relays")

	err = ex.processUTAMarkPrice(t.Context(), utaMessage(t, `{"T":"mark-price", "P":1791899401284316208, "d":{"s":"NOTACONTRACTM"}}`))
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "processUTAMarkPrice should error for an unknown futures contract")
	assert.Error(t, ex.processUTAMarkPrice(t.Context(), utaMessage(t, `{"T":"mark-price", "P":1791899401284316208, "d":[]}`)), "processUTAMarkPrice should error for malformed data")
	ex.Name = ""
	err = ex.processUTAMarkPrice(t.Context(), utaMessage(t, `{"T":"mark-price", "P":1791899401284316208, "d":{"s":"XBTUSDTM"}}`))
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet, "processUTAMarkPrice should return the stored ticker's lookup error")
}

func TestProcessUTAFundingFee(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	require.NoError(t, ex.processUTAFundingFee(t.Context(), utaFixtureMessages(t, "testdata/wsUTAFundingFee.json")[0]), "processUTAFundingFee must not error")
	exp := []any{&UTAWebsocketFundingFee{
		Symbol:             "XBTUSDTM",
		FundingFeeRate:     0.000055,
		FundingTime:        types.Time(time.UnixMilli(1791878400000)),
		NextFundingTime:    types.Time(time.UnixMilli(1791907200000)),
		Granularity:        28800000,
		FundingRateCap:     0.003,
		FundingRateFloor:   -0.003,
		LastFundingFeeRate: 0.000072,
	}}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTAFundingFee should relay every field of the push")
	assert.Error(t, ex.processUTAFundingFee(t.Context(), utaMessage(t, `{"T":"funding-fee", "P":1791899460244168627, "d":[]}`)), "processUTAFundingFee should error for malformed data")
}

func TestProcessUTAAllFundingFeeRates(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	require.NoError(t, ex.processUTAAllFundingFeeRates(t.Context(), utaFixtureMessages(t, "testdata/wsUTAFundingFee.json")[1]), "processUTAAllFundingFeeRates must not error")
	exp := []any{[]UTAWebsocketFundingFee{
		{
			Symbol:             "ETHUSDTM",
			FundingFeeRate:     -0.000006,
			FundingTime:        types.Time(time.UnixMilli(1791878400000)),
			NextFundingTime:    types.Time(time.UnixMilli(1791907200000)),
			Granularity:        28800000,
			FundingRateCap:     0.00375,
			FundingRateFloor:   -0.00375,
			LastFundingFeeRate: 0.000079,
		},
		{
			Symbol:             "XBTUSDTM",
			FundingFeeRate:     0.000055,
			FundingTime:        types.Time(time.UnixMilli(1791878400000)),
			NextFundingTime:    types.Time(time.UnixMilli(1791907200000)),
			Granularity:        28800000,
			FundingRateCap:     0.003,
			FundingRateFloor:   -0.003,
			LastFundingFeeRate: 0.000072,
		},
	}}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTAAllFundingFeeRates should relay every contract's rates")
	assert.Error(t, ex.processUTAAllFundingFeeRates(t.Context(), utaMessage(t, `{"T":"funding-fee-all-symbols", "P":1791899461185742531, "d":{}}`)), "processUTAAllFundingFeeRates should error for malformed data")
}

func TestProcessUTACallAuctionInfo(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	require.NoError(t, ex.processUTACallAuctionInfo(t.Context(), utaFixtureMessages(t, "testdata/wsUTACallAuctionInfo.json")[0]), "processUTACallAuctionInfo must not error")
	exp := []any{&UTAWebsocketCallAuctionInfo{
		Symbol:                  "GROVE-USDT",
		EstimatedPrice:          0.05568,
		EstimatedSize:           9099.4,
		SellOrderRangeLowPrice:  0.05568,
		SellOrderRangeHighPrice: 0.19839,
		BuyOrderRangeLowPrice:   0.00006,
		BuyOrderRangeHighPrice:  0.19,
		Timestamp:               types.Time(time.UnixMilli(1783434842521)),
	}}
	assert.Equal(t, exp, drainUTARelay(ex), "processUTACallAuctionInfo should relay every field of the push")
	assert.Error(t, ex.processUTACallAuctionInfo(t.Context(), utaMessage(t, `{"T":"callAuctionInfo.SPOT", "P":1783434842522281439, "d":[]}`)), "processUTACallAuctionInfo should error for malformed data")
}

// utaAllSeen returns whether every wanted relay has been seen
func utaAllSeen(seen map[string]bool, want []string) bool {
	for _, w := range want {
		if !seen[w] {
			return false
		}
	}
	return true
}

// TestUTAPublicChannels subscribes to each public channel pushing within seconds on both markets and checks that each
// relays data, against mock servers replaying recorded pushes or, with -tags mock_test_off, against KuCoin
func TestUTAPublicChannels(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	ex.SetTradeFeedStatus(true)
	spotURL, futuresURL := kucoinUTASpotPublicWebsocketURL, kucoinUTAFuturesPublicWebsocketURL
	if mockTests {
		orderbooks := utaFixtureLines(t, "testdata/wsUTAOrderbook.json")
		spotPushes := map[any]string{
			"kline": utaFixtureLines(t, "testdata/wsUTAKline.json")[0],
			"trade": utaFixtureLines(t, "testdata/wsUTATrade.json")[0],
			"obu":   orderbooks[0],
		}
		futuresPushes := map[any]string{
			"obu":        orderbooks[4],
			"mark-price": utaFixtureLines(t, "testdata/wsUTAMarkPrice.json")[0],
		}
		spotURL = newUTAMockServer(t, true, func(msg map[string]any) []string { return []string{utaAck(msg), spotPushes[msg["channel"]]} })
		futuresURL = newUTAMockServer(t, true, func(msg map[string]any) []string { return []string{utaAck(msg), futuresPushes[msg["channel"]]} })
	}
	newUTAWebsocket(t, ex, ex.newUTASpotPublicConnection(ex.Config, spotURL), ex.newUTAFuturesPublicConnection(ex.Config, futuresURL))
	ex.Features.Subscriptions = []*subscription.Subscription{
		{Enabled: true, Channel: UTAKlineChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}, Interval: kline.OneMin},
		{Enabled: true, Channel: UTATradeChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}},
		{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{utaSpotTestPair}, Levels: 5},
		{Enabled: true, Channel: UTAOrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}, Levels: 500},
		{Enabled: true, Channel: UTAMarkPriceChannel, Asset: asset.Futures, Pairs: currency.Pairs{utaFuturesTestPair}},
	}
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	seen := map[string]bool{}
	want := []string{"kline", "trade", "spot orderbook", "futures orderbook", "mark price"}
	deadline := time.After(time.Minute)
	for !utaAllSeen(seen, want) {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			switch d := p.Data.(type) {
			case *kline.Item:
				seen["kline"] = true
			case []trade.Data:
				seen["trade"] = true
			case *orderbook.Depth:
				seen[d.Asset().String()+" orderbook"] = true
			case *ticker.Price:
				seen["mark price"] = true
			}
		case <-deadline:
			require.FailNowf(t, "public channels timed out", "every public channel must relay data within a minute, relayed: %v", seen)
		}
	}
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		pair := utaSpotTestPair
		if a == asset.Futures {
			pair = utaFuturesTestPair
		}
		book, err := ex.Websocket.Orderbook.GetOrderbook(pair, a)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
		assert.NotEmptyf(t, book.Bids, "the %s orderbook should hold bids", a)
		assert.NotEmptyf(t, book.Asks, "the %s orderbook should hold asks", a)
	}
}
