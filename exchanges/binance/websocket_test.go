package binance

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// fixtureLines returns the lines of a websocket fixture file
func fixtureLines(tb testing.TB, path string) [][]byte {
	tb.Helper()
	f, err := os.Open(path)
	require.NoErrorf(tb, err, "Open must not error for %s", path)
	defer f.Close()
	var lines [][]byte
	s := bufio.NewScanner(f)
	for s.Scan() {
		lines = append(lines, append([]byte(nil), s.Bytes()...))
	}
	require.NoError(tb, s.Err(), "Scanner must not error")
	return lines
}

// newTestExchange returns an exchange set up from the test configuration under the test's name, so stores it writes
// do not collide with other tests'. In mock tests its REST requests go to the shared mock server and it signs with the
// placeholder credentials
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	ex.Name = tb.Name()
	if mockTests {
		require.NoError(tb, testexch.MockHTTPInstance(ex), "MockHTTPInstance must not error")
		ex.API.AuthenticatedSupport = true
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	} else if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		ex.API.AuthenticatedSupport = true
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(apiCredentials)
	}
	return ex
}

// useMockWebsocket replaces the exchange's websocket connections with one built from setup and served by mock, which
// the connection monitor does not reconnect within the test
func useMockWebsocket(tb testing.TB, ex *Exchange, setup *websocket.ConnectionSetup, mock mockws.WsMockFunc) {
	tb.Helper()
	s := httptest.NewTestServer(tb, mockws.CurryWsMockUpgrader(tb, mock))
	s.Start()
	setup.URL = "ws" + strings.TrimPrefix(s.URL, "http")
	useWebsocket(tb, ex, setup)
}

// useWebsocket replaces the exchange's websocket connections with one built from setup, which the connection monitor
// does not reconnect within the test
func useWebsocket(tb testing.TB, ex *Exchange, setup *websocket.ConnectionSetup) {
	tb.Helper()
	ex.Config.ConnectionMonitorDelay = time.Hour
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(tb, ex.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         ex.Config,
		Features:                               &ex.Features.Supports.WebsocketCapabilities,
		TradeFeed:                              ex.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement:           true,
		MaxWebsocketSubscriptionsPerConnection: 1024,
	}), "Websocket Setup must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	require.NoError(tb, ex.Websocket.SetupNewConnection(setup), "SetupNewConnection must not error")
	tb.Cleanup(func() {
		if ex.Websocket.IsConnected() {
			assert.NoError(tb, ex.Websocket.Shutdown(), "Shutdown should not error")
		}
	})
}

func BenchmarkWsHandleData(b *testing.B) {
	b.ReportAllocs()
	lines := fixtureLines(b, "testdata/wsSpot.json")
	ex := newTestExchange(b)
	useTestOrderbookSync(ex, func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
		return nil, errTestNoSnapshot
	})
	for b.Loop() {
		for x := range lines {
			assert.NoError(b, ex.wsHandleData(b.Context(), nil, lines[x]), "wsHandleData should not error")
		}
		for len(ex.Websocket.DataHandler.C) > 0 {
			<-ex.Websocket.DataHandler.C
		}
	}
}

func TestStreamPayloads(t *testing.T) {
	t.Parallel()
	exp := []struct {
		stream  string
		payload any
	}{
		{"btcusdt@trade", &TradeStream{
			EventType:    "trade",
			EventTime:    types.Time(time.UnixMilli(1791284980615)),
			Symbol:       "BTCUSDT",
			TradeID:      6739446511,
			Price:        86104.02,
			Quantity:     0.00011,
			TradeTime:    types.Time(time.UnixMilli(1791284980614)),
			IsBuyerMaker: true,
			Ignore:       true,
		}},
		{"btcusdt@ticker", &TickerStream{
			EventType:                   "24hrTicker",
			EventTime:                   types.Time(time.UnixMilli(1791284981016)),
			Symbol:                      "BTCUSDT",
			PriceChange:                 156.02,
			PriceChangePercent:          0.182,
			WeightedAveragePrice:        85748.34072691,
			PreviousClosePrice:          85948,
			LastPrice:                   86104.02,
			LastQuantity:                0.00011,
			BestBidPrice:                86104.01,
			BestBidQuantity:             7.31683,
			BestAskPrice:                86104.02,
			BestAskQuantity:             5.55365,
			OpenPrice:                   85948,
			HighPrice:                   86725.12,
			LowPrice:                    84972.01,
			TotalTradedBaseAssetVolume:  13625.56497,
			TotalTradedQuoteAssetVolume: 1168369587.644197,
			StatisticsOpenTime:          types.Time(time.UnixMilli(1791198581003)),
			StatisticsCloseTime:         types.Time(time.UnixMilli(1791284981003)),
			FirstTradeID:                6736717189,
			LastTradeID:                 6739446511,
			TotalNumberOfTrades:         2729323,
		}},
		{"btcusdt@kline_1m", &KlineStream{
			EventType: "kline",
			EventTime: types.Time(time.UnixMilli(1791284982042)),
			Symbol:    "BTCUSDT",
			Kline: KlineStreamData{
				StartTime:                types.Time(time.UnixMilli(1791284940000)),
				CloseTime:                types.Time(time.UnixMilli(1791284999999)),
				Symbol:                   "BTCUSDT",
				Interval:                 "1m",
				FirstTradeID:             6739444698,
				LastTradeID:              6739446515,
				OpenPrice:                86118.01,
				ClosePrice:               86104.02,
				HighPrice:                86118.01,
				LowPrice:                 86104.01,
				BaseAssetVolume:          4.97348,
				NumberOfTrades:           1818,
				QuoteAssetVolume:         428264.1671163,
				TakerBuyBaseAssetVolume:  2.19449,
				TakerBuyQuoteAssetVolume: 188961.5203966,
				Ignore:                   123456,
			},
		}},
		{"btcusdt@kline_1s", &KlineStream{
			EventType: "kline",
			EventTime: types.Time(time.UnixMilli(1791284981014)),
			Symbol:    "BTCUSDT",
			Kline: KlineStreamData{
				StartTime:                types.Time(time.UnixMilli(1791284980000)),
				CloseTime:                types.Time(time.UnixMilli(1791284980999)),
				Symbol:                   "BTCUSDT",
				Interval:                 "1s",
				FirstTradeID:             6739446508,
				LastTradeID:              6739446511,
				OpenPrice:                86104.01,
				ClosePrice:               86104.02,
				HighPrice:                86104.03,
				LowPrice:                 86104,
				BaseAssetVolume:          0.00835,
				NumberOfTrades:           4,
				IsClosed:                 true,
				QuoteAssetVolume:         718.968509,
				TakerBuyBaseAssetVolume:  0.00255,
				TakerBuyQuoteAssetVolume: 219.565251,
				Ignore:                   123456,
			},
		}},
		{"btcusdt@depth5", &WsPartialDepth{
			LastUpdateID: 101096708675,
			Bids:         orderbook.LevelsArrayPriceAmount{{Price: 86104.01, Amount: 7.31683}, {Price: 86104, Amount: 0.02991}},
			Asks:         orderbook.LevelsArrayPriceAmount{{Price: 86104.02, Amount: 5.55365}, {Price: 86104.03, Amount: 0.00072}},
		}},
		{"btcusdt@depth10@100ms", &WsPartialDepth{
			LastUpdateID: 101096708778,
			Bids:         orderbook.LevelsArrayPriceAmount{{Price: 86104.01, Amount: 7.31528}, {Price: 86104, Amount: 0.02997}},
			Asks:         orderbook.LevelsArrayPriceAmount{{Price: 86104.02, Amount: 5.48779}, {Price: 86104.03, Amount: 0.00072}},
		}},
		{"btcusdt@depth@100ms", &DiffDepthStream{
			EventType:     "depthUpdate",
			EventTime:     types.Time(time.UnixMilli(1791284980512)),
			Symbol:        "BTCUSDT",
			FirstUpdateID: 101096708651,
			FinalUpdateID: 101096708675,
			Bids:          orderbook.LevelsArrayPriceAmount{{Price: 86104, Amount: 0.02991}},
			Asks:          orderbook.LevelsArrayPriceAmount{{Price: 86104.04, Amount: 1}},
		}},
		{"btcusdt@depth@100ms", &DiffDepthStream{
			EventType:     "depthUpdate",
			EventTime:     types.Time(time.UnixMilli(1791284980612)),
			Symbol:        "BTCUSDT",
			FirstUpdateID: 101096708676,
			FinalUpdateID: 101096708700,
			Bids:          orderbook.LevelsArrayPriceAmount{{Price: 86104.01, Amount: 7.316}, {Price: 86103.5, Amount: 0}},
			Asks:          orderbook.LevelsArrayPriceAmount{{Price: 86104.02, Amount: 5.5}},
		}},
		{"btcusdt@depth@100ms", &DiffDepthStream{
			EventType:     "depthUpdate",
			EventTime:     types.Time(time.UnixMilli(1791284980712)),
			Symbol:        "BTCUSDT",
			FirstUpdateID: 101096708701,
			FinalUpdateID: 101096708778,
			Bids:          orderbook.LevelsArrayPriceAmount{{Price: 86104.01, Amount: 7.31528}, {Price: 86104, Amount: 0.02997}},
			Asks:          orderbook.LevelsArrayPriceAmount{{Price: 86104.02, Amount: 5.48779}, {Price: 86104.05, Amount: 0.12}},
		}},
	}
	lines := fixtureLines(t, "testdata/wsSpot.json")
	require.Len(t, lines, len(exp), "fixture must have one frame per expected payload")
	for i, line := range lines {
		stream, err := jsonparser.GetString(line, "stream")
		require.NoErrorf(t, err, "frame %d must have a stream", i)
		require.Equalf(t, exp[i].stream, stream, "frame %d must be from the expected stream", i)
		data, _, _, err := jsonparser.Get(line, "data")
		require.NoErrorf(t, err, "frame %d must have data", i)
		got := reflect.New(reflect.TypeOf(exp[i].payload).Elem()).Interface()
		require.NoErrorf(t, json.Unmarshal(data, got), "Unmarshal must not error for frame %d", i)
		assert.Equalf(t, exp[i].payload, got, "frame %d of %s should decode every field", i, stream)
	}
}

func TestWsHandleData(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.Features.Enabled.TradeFeed = true
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	// Diff depth frames only wait for a snapshot here; TestSpotOrderbookSync synchronises them
	useTestOrderbookSync(ex, func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
		return nil, errTestNoSnapshot
	})
	for _, line := range fixtureLines(t, "testdata/wsSpot.json") {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, line), "wsHandleData must not error for %s", line)
	}
	pair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	exp := []any{
		[]trade.Data{{
			Exchange:     t.Name(),
			CurrencyPair: pair,
			AssetType:    asset.Spot,
			Side:         order.Sell,
			TID:          "6739446511",
			Price:        86104.02,
			Amount:       0.00011,
			Timestamp:    time.UnixMilli(1791284980614),
		}},
		&ticker.Price{
			Last:                       86104.02,
			LastSize:                   0.00011,
			VolumeWeightedAveragePrice: 85748.34072691,
			High:                       86725.12,
			Low:                        84972.01,
			Bid:                        86104.01,
			BidSize:                    7.31683,
			Ask:                        86104.02,
			AskSize:                    5.55365,
			BaseVolume:                 13625.56497,
			QuoteVolume:                1168369587.644197,
			Open:                       85948,
			PercentChange24Hour:        0.182,
			Close:                      85948,
			Pair:                       pair,
			ExchangeName:               t.Name(),
			AssetType:                  asset.Spot,
			LastUpdated:                time.UnixMilli(1791284981016),
		},
		kline.Item{
			Exchange: t.Name(),
			Pair:     pair,
			Asset:    asset.Spot,
			Interval: kline.OneMin,
			Candles: []kline.Candle{{
				Time:             time.UnixMilli(1791284940000),
				Open:             86118.01,
				High:             86118.01,
				Low:              86104.01,
				Close:            86104.02,
				Volume:           4.97348,
				ValidationIssues: kline.PartialCandle,
			}},
		},
		kline.Item{
			Exchange: t.Name(),
			Pair:     pair,
			Asset:    asset.Spot,
			Interval: kline.ThousandMilliseconds,
			Candles: []kline.Candle{{
				Time:   time.UnixMilli(1791284980000),
				Open:   86104.01,
				High:   86104.03,
				Low:    86104,
				Close:  86104.02,
				Volume: 0.00835,
			}},
		},
	}
	for i := range exp {
		require.NotEmptyf(t, ex.Websocket.DataHandler.C, "relay must hold payload %d", i)
		got := (<-ex.Websocket.DataHandler.C).Data
		assert.Equalf(t, exp[i], got, "relayed payload %d should match", i)
	}
	stored, err := ticker.GetTicker(t.Name(), pair, asset.Spot)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, exp[1], stored, "the ticker should be stored as it was relayed")

	// The partial depth payloads load snapshots, which the order book relays
	for range 2 {
		require.NotEmpty(t, ex.Websocket.DataHandler.C, "relay must hold the order book snapshots")
		<-ex.Websocket.DataHandler.C
	}
	assert.Empty(t, ex.Websocket.DataHandler.C, "relay should hold nothing else")
	book, err := ex.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, int64(101096708778), book.LastUpdateID, "the last partial depth payload should be the book")
	assert.Equal(t, orderbook.Levels{{Price: 86104.01, Amount: 7.31528}, {Price: 86104, Amount: 0.02997}}, book.Bids, "bids should be the last payload's")
	assert.Equal(t, orderbook.Levels{{Price: 86104.02, Amount: 5.48779}, {Price: 86104.03, Amount: 0.00072}}, book.Asks, "asks should be the last payload's")
}

func TestWsHandleDataErrors(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.Features.Enabled.TradeFeed = true
	for _, tc := range []struct {
		name string
		msg  string
		err  error
	}{
		{name: "no stream", msg: `{"data":{"e":"trade"}}`, err: errUnhandledMessage},
		{name: "no data", msg: `{"stream":"btcusdt@trade"}`, err: errUnhandledMessage},
		{name: "unknown channel", msg: `{"stream":"btcusdt@aggTrade","data":{"e":"aggTrade"}}`, err: errUnhandledMessage},
		{name: "unknown kline interval", msg: `{"stream":"btcusdt@kline_2s","data":{"e":"kline","s":"BTCUSDT","k":{"i":"2s"}}}`, err: kline.ErrInvalidInterval},
		{name: "unknown symbol", msg: `{"stream":"moonusdt@ticker","data":{"e":"24hrTicker","s":"MOONUSDT"}}`, err: currency.ErrPairNotFound},
		{name: "unknown partial depth symbol", msg: `{"stream":"moonusdt@depth5","data":{"lastUpdateId":1}}`, err: currency.ErrPairNotFound},
		{name: "unknown diff depth symbol", msg: `{"stream":"moonusdt@depth","data":{"e":"depthUpdate","s":"MOONUSDT","U":1,"u":2}}`, err: currency.ErrPairNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, ex.wsHandleData(t.Context(), nil, []byte(tc.msg)), tc.err, "wsHandleData should return the expected error")
		})
	}
	for _, msg := range []string{
		`{"stream":"btcusdt@trade","data":"bad"}`,
		`{"stream":"btcusdt@ticker","data":"bad"}`,
		`{"stream":"btcusdt@kline_1m","data":"bad"}`,
		`{"stream":"btcusdt@depth","data":"bad"}`,
		`{"stream":"btcusdt@depth5","data":"bad"}`,
		`{"stream":"!serverShutdown","data":"bad"}`,
	} {
		assert.Errorf(t, ex.wsHandleData(t.Context(), nil, []byte(msg)), "wsHandleData should reject %s", msg)
	}
}

func TestWsHandleDataDisabledPair(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.Features.Enabled.TradeFeed = true
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	ethusdt := currency.NewPairWithDelimiter("ETH", "USDT", "-")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "-"), ethusdt}, false), "StorePairs must not error")
	// Without a synchronisation a diff depth frame that reached it would error
	ex.orderbookSync = nil
	for _, msg := range []string{
		`{"stream":"ethusdt@trade","data":{"e":"trade","s":"ETHUSDT","t":1,"p":"1","q":"1"}}`,
		`{"stream":"ethusdt@ticker","data":{"e":"24hrTicker","s":"ETHUSDT","c":"1"}}`,
		`{"stream":"ethusdt@kline_1m","data":{"e":"kline","s":"ETHUSDT","k":{"i":"1m"}}}`,
		`{"stream":"ethusdt@depth5","data":{"lastUpdateId":1,"bids":[["1","1"]],"asks":[["2","1"]]}}`,
		`{"stream":"ethusdt@depth@100ms","data":{"e":"depthUpdate","s":"ETHUSDT","U":1,"u":2,"b":[["1","1"]],"a":[]}}`,
	} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, []byte(msg)), "wsHandleData must not error for a disabled pair's %s", msg)
	}
	assert.Empty(t, ex.Websocket.DataHandler.C, "a disabled pair's payloads should not be relayed")
}

func TestHandleServerShutdown(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	setup := ex.spotStreamConnectionSetup(ex.Config)
	setup.SubscriptionsNotRequired = true
	useMockWebsocket(t, ex, setup, func(testing.TB, []byte, *gws.Conn) error { return nil })
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	require.NoError(t, ex.wsHandleData(t.Context(), nil, []byte(`{"stream":"!serverShutdown","data":{"e":"serverShutdown","E":1770123456789}}`)), "wsHandleData must not error for serverShutdown")
	assert.Eventually(t, func() bool { return !ex.Websocket.IsConnected() }, 5*time.Second, 10*time.Millisecond, "serverShutdown should disconnect, so the connection monitor reconnects")
}

func TestSpotOrderbookSync(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	pair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	// The snapshot falls inside the second frame, so the first frame is dropped and the second applied
	useTestOrderbookSync(ex, func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
		return &orderbook.Book{
			Exchange:     ex.Name,
			Pair:         p,
			Asset:        a,
			Bids:         orderbook.Levels{{Price: 86104.01, Amount: 7.31683}, {Price: 86104, Amount: 0.02991}, {Price: 86103.5, Amount: 0.5}},
			Asks:         orderbook.Levels{{Price: 86104.02, Amount: 5.55365}, {Price: 86104.03, Amount: 0.00072}},
			LastUpdateID: 101096708690,
			LastUpdated:  time.UnixMilli(1791284980650),
		}, nil
	})
	for _, line := range fixtureLines(t, "testdata/wsSpot.json") {
		if stream, err := jsonparser.GetString(line, "stream"); err == nil && stream == "btcusdt@depth@100ms" {
			require.NoError(t, ex.wsHandleData(t.Context(), nil, line), "wsHandleData must not error for a diff depth frame")
		}
	}
	var got *orderbookState
	require.Eventually(t, func() bool {
		var err error
		got, err = storedOrderbookState(ex, pair, asset.Spot)
		return err == nil && got.LastUpdateID == 101096708778
	}, 5*time.Second, 10*time.Millisecond, "the order book must synchronise to the last frame")
	exp := &orderbookState{
		Bids:         orderbook.Levels{{Price: 86104.01, Amount: 7.31528}, {Price: 86104, Amount: 0.02997}},
		Asks:         orderbook.Levels{{Price: 86104.02, Amount: 5.48779}, {Price: 86104.03, Amount: 0.00072}, {Price: 86104.05, Amount: 0.12}},
		LastUpdateID: 101096708778,
		LastUpdated:  time.UnixMilli(1791284980712),
	}
	assert.Equal(t, exp, got, "the book should be the snapshot with the frames after it applied")
}

func TestProcessOrderbookUpdate(t *testing.T) {
	t.Parallel()
	err := new(Exchange).processOrderbookUpdate(t.Context(), 1, &orderbook.Update{})
	assert.ErrorIs(t, err, errOrderbookSyncNotSetUp, "processOrderbookUpdate should require the order book synchronisation")
}

func TestProcessOrderbookUpdateOverlap(t *testing.T) {
	t.Parallel()
	// Spot applies an event that starts within the book and ends after it, while the derivatives procedures need each
	// event to chain to the last by pu, so a new snapshot is taken instead
	for a, applied := range map[asset.Item]bool{asset.Spot: true, asset.USDTMarginedFutures: false} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			var fetches atomic.Int32
			useTestOrderbookSync(ex, func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
				fetches.Add(1)
				return &orderbook.Book{
					Exchange: ex.Name, Pair: p, Asset: a, Bids: orderbook.Levels{{Price: 100, Amount: 1}}, Asks: orderbook.Levels{{Price: 101, Amount: 1}},
					LastUpdateID: 100, LastUpdated: spotDepthFrameTime(100),
				}, nil
			})
			pair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
			update := func(finalUpdateID int64) *orderbook.Update {
				return &orderbook.Update{
					UpdateID: finalUpdateID, UpdateTime: spotDepthFrameTime(finalUpdateID), Asset: a, Pair: pair,
					Bids: orderbook.Levels{{Price: 100, Amount: float64(finalUpdateID)}},
				}
			}
			require.NoError(t, ex.processOrderbookUpdate(t.Context(), 101, update(105)), "processOrderbookUpdate must not error")
			require.Eventually(t, func() bool {
				id, err := ex.Websocket.Orderbook.LastUpdateID(pair, a)
				return err == nil && id == 105
			}, 5*time.Second, 10*time.Millisecond, "the book must synchronise")

			require.NoError(t, ex.processOrderbookUpdate(t.Context(), 103, update(110)), "processOrderbookUpdate must not error for an overlapping event")
			got, err := storedOrderbookState(ex, pair, a)
			if !applied {
				assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "an overlapping derivatives event should invalidate the book")
				return
			}
			require.NoError(t, err, "an overlapping spot event must leave the book valid")
			exp := &orderbookState{
				Bids: orderbook.Levels{{Price: 100, Amount: 110}}, Asks: orderbook.Levels{{Price: 101, Amount: 1}},
				LastUpdateID: 110, LastUpdated: spotDepthFrameTime(110),
			}
			assert.Equal(t, exp, got, "an overlapping spot event should apply")
			assert.Equal(t, int32(1), fetches.Load(), "an overlapping spot event should not take a new snapshot")
		})
	}
}

func TestCheckSpotPendingUpdate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		firstUpdateID int64
		finalUpdateID int64
		skip          bool
		err           error
	}{
		{name: "ends before the snapshot", firstUpdateID: 90, finalUpdateID: 99, skip: true},
		{name: "ends at the snapshot", firstUpdateID: 95, finalUpdateID: 100, skip: true},
		{name: "spans the snapshot", firstUpdateID: 95, finalUpdateID: 110},
		{name: "follows the snapshot", firstUpdateID: 101, finalUpdateID: 110},
		{name: "starts after a gap", firstUpdateID: 102, finalUpdateID: 110, err: orderbookmanager.ErrOrderbookSnapshotOutdated},
	} {
		skip, err := checkSpotPendingUpdate(100, tc.firstUpdateID, &orderbook.Update{UpdateID: tc.finalUpdateID, Asset: asset.Spot})
		assert.ErrorIsf(t, err, tc.err, "checkSpotPendingUpdate should return the expected error for an event that %s", tc.name)
		assert.Equalf(t, tc.skip, skip, "checkSpotPendingUpdate should decide whether to skip an event that %s", tc.name)
	}
}

func TestCheckPendingUpdate(t *testing.T) {
	t.Parallel()
	// An event ending at the snapshot's update ID is already in a spot snapshot, while the derivatives procedures apply it
	for a, exp := range map[asset.Item]bool{
		asset.Spot:                true,
		asset.USDTMarginedFutures: false,
		asset.CoinMarginedFutures: false,
		asset.Options:             false,
	} {
		skip, err := checkPendingUpdate(100, 95, &orderbook.Update{UpdateID: 100, Asset: a})
		require.NoErrorf(t, err, "checkPendingUpdate must not error for a %s event", a)
		assert.Equalf(t, exp, skip, "checkPendingUpdate should follow the %s procedure", a)
	}
	_, err := checkPendingUpdate(100, 102, &orderbook.Update{UpdateID: 110, Asset: asset.Spot})
	assert.ErrorIs(t, err, orderbookmanager.ErrOrderbookSnapshotOutdated, "checkPendingUpdate should reject a spot event after a gap")
	_, err = checkPendingUpdate(100, 102, &orderbook.Update{UpdateID: 110, Asset: asset.Options})
	assert.ErrorIs(t, err, orderbookmanager.ErrOrderbookSnapshotOutdated, "checkPendingUpdate should reject an options event after a gap")
}

func TestOrderbookSnapshotLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a         asset.Item
		limit     uint64
		limitRate func(uint64) request.EndpointLimit
		cost      endpointCost
	}{
		{asset.Spot, 1000, spotOrderbookLimit, endpointCost{spotIPPool, 50}},
		{asset.USDTMarginedFutures, 1000, uFuturesOrderbookLimit, endpointCost{derivativesIPPool, 20}},
		{asset.CoinMarginedFutures, 1000, cFuturesOrderbookLimit, endpointCost{derivativesIPPool, 20}},
		// Options orders share a 400 a minute pool, so their snapshots take the deepest book at the least weight
		{asset.Options, 50, optionsOrderbookLimit, endpointCost{optionsIPPool, 1}},
	} {
		limit := orderbookSnapshotLimit(tc.a)
		assert.Equalf(t, tc.limit, limit, "orderbookSnapshotLimit should give the %s snapshot depth", tc.a)
		assert.Equalf(t, tc.cost, endpointCosts[tc.limitRate(limit)], "the %s snapshot should cost its documented weight", tc.a)
	}
}

func TestFetchOrderbookSnapshot(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var limit, body string
		switch r.URL.Path {
		case "/api/v3/depth":
			limit, body = "1000", `{"lastUpdateId":101096708690,"bids":[["86104.01000000","7.31683000"]],"asks":[["86104.02000000","5.55365000"]]}`
		case "/fapi/v1/depth", "/dapi/v1/depth":
			limit, body = "1000", `{"lastUpdateId":1027024,"E":1589436922972,"T":1589436922959,"bids":[["4.00000000","431.00000000"]],"asks":[["4.00000200","12.00000000"]]}`
		case "/eapi/v1/depth":
			limit, body = "50", `{"T":1762866729358,"lastUpdateId":42788078000,"bids":[["3645.000","1.00"]],"asks":[["3675.000","5.00"]]}`
		default:
			http.NotFound(w, r)
			return
		}
		assert.Equalf(t, limit, r.URL.Query().Get("limit"), "%s should request the snapshot depth", r.URL.Path)
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Write should not error")
	}))
	srv.Start()
	for _, u := range []exchange.URL{exchange.RestSpotSupplementary, exchange.RestUSDTMargined, exchange.RestCoinMargined, exchange.RestOptions} {
		require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(u.String(), srv.URL), "SetRunningURL must not error for %s", u)
	}
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")

	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  *orderbook.Book
	}{
		{asset.Spot, currency.NewBTCUSDT(), &orderbook.Book{Bids: orderbook.Levels{{Price: 86104.01, Amount: 7.31683}}, Asks: orderbook.Levels{{Price: 86104.02, Amount: 5.55365}}, LastUpdateID: 101096708690}},
		{asset.USDTMarginedFutures, currency.NewBTCUSDT(), &orderbook.Book{Bids: orderbook.Levels{{Price: 4, Amount: 431}}, Asks: orderbook.Levels{{Price: 4.000002, Amount: 12}}, LastUpdateID: 1027024, LastUpdated: time.UnixMilli(1589436922959)}},
		{asset.CoinMarginedFutures, testCFuturesPair, &orderbook.Book{Bids: orderbook.Levels{{Price: 4, Amount: 431}}, Asks: orderbook.Levels{{Price: 4.000002, Amount: 12}}, LastUpdateID: 1027024, LastUpdated: time.UnixMilli(1589436922959)}},
		{asset.Options, testOptionsPair, &orderbook.Book{Bids: orderbook.Levels{{Price: 3645, Amount: 1}}, Asks: orderbook.Levels{{Price: 3675, Amount: 5}}, LastUpdateID: 42788078000, LastUpdated: time.UnixMilli(1762866729358)}},
	} {
		book, err := ex.fetchOrderbookSnapshot(t.Context(), tc.pair, tc.a)
		require.NoErrorf(t, err, "fetchOrderbookSnapshot must not error for %s", tc.a)
		if tc.a == asset.Spot {
			// The spot snapshot carries no time, so it is stamped when it arrives
			assert.WithinDuration(t, time.Now(), book.LastUpdated, time.Minute, "the spot snapshot should be stamped with its arrival time")
			book.LastUpdated = time.Time{}
		}
		tc.exp.Exchange, tc.exp.Pair, tc.exp.Asset, tc.exp.ValidateOrderbook = ex.Name, tc.pair, tc.a, ex.ValidateOrderbook
		assert.Equalf(t, tc.exp, book, "fetchOrderbookSnapshot should convert the %s snapshot", tc.a)
	}
	_, err := ex.fetchOrderbookSnapshot(t.Context(), currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "fetchOrderbookSnapshot should return the spot request's error")
	_, err = ex.fetchOrderbookSnapshot(t.Context(), currency.NewBTCUSDT(), asset.Margin)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "fetchOrderbookSnapshot should reject an asset without diff depth streams")
}

// spotDepthStreams generates diff depth frames the way Binance's streams send them: each continues its symbol's chain
// of update IDs and sets the bid at price 100 to its final update ID, so a book shows which frame it has reached. A
// snapshot is the book at the latest frame generated
type spotDepthStreams struct {
	mu   sync.Mutex
	last map[string]int64
}

func newSpotDepthStreams() *spotDepthStreams {
	return &spotDepthStreams{last: make(map[string]int64)}
}

// spotDepthFrameTime is the event time of the frame ending at an update ID
func spotDepthFrameTime(finalUpdateID int64) time.Time {
	return time.UnixMilli(1791284980000 + finalUpdateID)
}

// lastLocked returns a symbol's latest final update ID; its chain starts at 1000
func (s *spotDepthStreams) lastLocked(symbol string) int64 {
	if _, ok := s.last[symbol]; !ok {
		s.last[symbol] = 1000
	}
	return s.last[symbol]
}

// frame returns the next frame of a symbol's stream
func (s *spotDepthStreams) frame(symbol string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	first := s.lastLocked(symbol) + 1
	final := first + 4
	s.last[symbol] = final
	return fmt.Appendf(nil, `{"stream":%q,"data":{"e":"depthUpdate","E":%d,"s":%q,"U":%d,"u":%d,"b":[["100","%d"]],"a":[]}}`,
		strings.ToLower(symbol)+"@depth@100ms", spotDepthFrameTime(final).UnixMilli(), symbol, first, final, final)
}

// skip moves a symbol's stream on without frames, as it moves on while frames are lost or the stream is unsubscribed
func (s *spotDepthStreams) skip(symbol string, updates int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last[symbol] = s.lastLocked(symbol) + updates
}

// finalUpdateID returns the final update ID of a symbol's latest frame
func (s *spotDepthStreams) finalUpdateID(symbol string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastLocked(symbol)
}

// snapshotJSON returns a REST snapshot of a symbol's book
func (s *spotDepthStreams) snapshotJSON(symbol string) []byte {
	last := s.finalUpdateID(symbol)
	return fmt.Appendf(nil, `{"lastUpdateId":%d,"bids":[["100","%d"],["99","1"]],"asks":[["101","1"]]}`, last, last)
}

// snapshotBook returns a snapshot of a pair's book as the order book synchronisation takes it
func (s *spotDepthStreams) snapshotBook(ex *Exchange, p currency.Pair, a asset.Item) *orderbook.Book {
	last := s.finalUpdateID(p.Base.String() + p.Quote.String())
	return &orderbook.Book{
		Exchange:     ex.Name,
		Pair:         p,
		Asset:        a,
		Bids:         orderbook.Levels{{Price: 100, Amount: float64(last)}, {Price: 99, Amount: 1}},
		Asks:         orderbook.Levels{{Price: 101, Amount: 1}},
		LastUpdateID: last,
		LastUpdated:  spotDepthFrameTime(last),
	}
}

// expectedState returns a symbol's book once it follows its stream to the latest frame
func (s *spotDepthStreams) expectedState(symbol string) *orderbookState {
	last := s.finalUpdateID(symbol)
	return &orderbookState{
		Bids:         orderbook.Levels{{Price: 100, Amount: float64(last)}, {Price: 99, Amount: 1}},
		Asks:         orderbook.Levels{{Price: 101, Amount: 1}},
		LastUpdateID: last,
		LastUpdated:  spotDepthFrameTime(last),
	}
}

// syncSpotBook sends frames of a pair's stream until its book follows them
func syncSpotBook(t *testing.T, ex *Exchange, streams *spotDepthStreams, pair currency.Pair) {
	t.Helper()
	symbol := pair.Base.String() + pair.Quote.String()
	require.Eventually(t, func() bool {
		if err := ex.wsHandleData(t.Context(), nil, streams.frame(symbol)); err != nil {
			return false
		}
		book, err := ex.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
		return err == nil && book.LastUpdateID == streams.finalUpdateID(symbol)
	}, 5*time.Second, 10*time.Millisecond, "the book must synchronise with its stream")
}

func TestSpotOrderbookResyncsAfterGap(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	streams := newSpotDepthStreams()
	var fetches atomic.Int32
	resync := make(chan struct{})
	useTestOrderbookSync(ex, func(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
		if fetches.Add(1) == 2 {
			// The second snapshot waits, so the test sees the book while it is resynchronised
			select {
			case <-resync:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return streams.snapshotBook(ex, p, a), nil
	})
	pair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	syncSpotBook(t, ex, streams, pair)
	require.Equal(t, int32(1), fetches.Load(), "the first synchronisation must take one snapshot")

	// Frames lost in transit leave a gap in the update IDs
	streams.skip("BTCUSDT", 10)
	require.NoError(t, ex.wsHandleData(t.Context(), nil, streams.frame("BTCUSDT")), "wsHandleData must not error for a frame after a gap")
	_, err := ex.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "a gap should invalidate the book")
	require.Eventually(t, func() bool { return fetches.Load() == 2 }, 5*time.Second, 10*time.Millisecond, "a gap must start a new synchronisation")
	close(resync)
	syncSpotBook(t, ex, streams, pair)
	assert.Equal(t, int32(2), fetches.Load(), "the gap should take one new snapshot")
	got, err := storedOrderbookState(ex, pair, asset.Spot)
	require.NoError(t, err, "the resynchronised book must be stored")
	assert.Equal(t, streams.expectedState("BTCUSDT"), got, "the resynchronised book should follow its stream")
}

func TestSpotOrderbookRetriesFailedSnapshot(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	streams := newSpotDepthStreams()
	var fetches atomic.Int32
	useTestOrderbookSync(ex, func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
		if fetches.Add(1) == 1 {
			return nil, errTestNoSnapshot
		}
		return streams.snapshotBook(ex, p, a), nil
	})
	pair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	syncSpotBook(t, ex, streams, pair)
	assert.Equal(t, int32(2), fetches.Load(), "a failed snapshot should be fetched again for a following frame")
	got, err := storedOrderbookState(ex, pair, asset.Spot)
	require.NoError(t, err, "the synchronised book must be stored")
	assert.Equal(t, streams.expectedState("BTCUSDT"), got, "the book should follow its stream after a failed snapshot")
}

func TestSpotOrderbookDropsStaleEvents(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	streams := newSpotDepthStreams()
	var fetches atomic.Int32
	useTestOrderbookSync(ex, func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
		fetches.Add(1)
		return streams.snapshotBook(ex, p, a), nil
	})
	pair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	syncSpotBook(t, ex, streams, pair)
	older, latest := streams.frame("BTCUSDT"), streams.frame("BTCUSDT")
	// Both frames apply, then arrive again late; a late event ends at or before the book's update ID, and Binance's
	// procedure ignores it
	for _, frame := range [][]byte{older, latest, latest, older} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, frame), "wsHandleData must not error for %s", frame)
	}
	got, err := storedOrderbookState(ex, pair, asset.Spot)
	require.NoError(t, err, "late events must leave the book valid")
	assert.Equal(t, streams.expectedState("BTCUSDT"), got, "late events should leave the book as its stream left it")
	assert.Equal(t, int32(1), fetches.Load(), "late events should not take a new snapshot")
}

// spotDepthServer serves spot diff depth streams over websocket and their snapshots over REST, as Binance does: each
// subscribed stream sends a frame every 20ms, and a resubscribed stream resumes after the updates it missed
type spotDepthServer struct {
	streams *spotDepthStreams
	done    chan struct{}

	mu         sync.Mutex
	conns      map[*gws.Conn]*spotDepthConn
	subscribed map[string]bool
	fetches    map[string]int
	hold       chan struct{}
	held       int
}

// spotDepthConn is a connection's subscribed symbols
type spotDepthConn struct {
	writeMu sync.Mutex
	symbols []string
}

func newSpotDepthServer(t *testing.T) *spotDepthServer {
	t.Helper()
	s := &spotDepthServer{
		streams:    newSpotDepthStreams(),
		done:       make(chan struct{}),
		conns:      make(map[*gws.Conn]*spotDepthConn),
		subscribed: make(map[string]bool),
		fetches:    make(map[string]int),
	}
	t.Cleanup(func() { close(s.done) })
	return s
}

// use connects an exchange's spot diff depth streams, at most maxSubscriptions a connection, to the server
func (s *spotDepthServer) use(t *testing.T, ex *Exchange, maxSubscriptions int) {
	t.Helper()
	rest := httptest.NewTestServer(t, http.HandlerFunc(s.serveSnapshot))
	rest.Start()
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpotSupplementary.String(), rest.URL), "SetRunningURL must not error")
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds}}

	streams := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, s.handleRequest))
	streams.Start()
	setup := ex.spotStreamConnectionSetup(ex.Config)
	setup.URL = "ws" + strings.TrimPrefix(streams.URL, "http")
	ex.Config.ConnectionMonitorDelay = time.Hour
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(t, ex.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         ex.Config,
		Features:                               &ex.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement:           true,
		MaxWebsocketSubscriptionsPerConnection: maxSubscriptions,
	}), "Websocket Setup must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	require.NoError(t, ex.Websocket.SetupNewConnection(setup), "SetupNewConnection must not error")
	t.Cleanup(func() {
		if ex.Websocket.IsConnected() {
			assert.NoError(t, ex.Websocket.Shutdown(), "Shutdown should not error")
		}
	})
	useTestOrderbookSync(ex, ex.fetchOrderbookSnapshot)
	// The relay is drained so it never blocks the readers
	relay := ex.Websocket.DataHandler
	go func() {
		for {
			select {
			case <-s.done:
				return
			case <-relay.C:
			}
		}
	}()
}

// handleRequest answers SUBSCRIBE and UNSUBSCRIBE requests and streams a connection's subscribed symbols
func (s *spotDepthServer) handleRequest(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	var req WsPayload
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	s.mu.Lock()
	c, ok := s.conns[w]
	if !ok {
		c = new(spotDepthConn)
		s.conns[w] = c
		go s.stream(w, c)
	}
	for _, stream := range req.Params {
		name, _, _ := strings.Cut(stream, "@")
		symbol := strings.ToUpper(name)
		if req.Method != wsSubscribeMethod {
			c.symbols = slices.DeleteFunc(c.symbols, func(s string) bool { return s == symbol })
			continue
		}
		if s.subscribed[symbol] {
			s.streams.skip(symbol, 1000)
		}
		s.subscribed[symbol] = true
		c.symbols = append(c.symbols, symbol)
	}
	s.mu.Unlock()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"result":null,"id":%q}`, req.ID))
}

// stream sends a frame of each of a connection's symbols every 20ms until the connection or the test ends
func (s *spotDepthServer) stream(w *gws.Conn, c *spotDepthConn) {
	frameTicker := time.NewTicker(20 * time.Millisecond)
	defer frameTicker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-frameTicker.C:
		}
		s.mu.Lock()
		frames := make([][]byte, len(c.symbols))
		for i, symbol := range c.symbols {
			frames[i] = s.streams.frame(symbol)
		}
		s.mu.Unlock()
		for _, frame := range frames {
			c.writeMu.Lock()
			err := w.WriteMessage(gws.TextMessage, frame)
			c.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// serveSnapshot answers a REST order book snapshot request, once holdSnapshots releases it if it is holding them
func (s *spotDepthServer) serveSnapshot(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	s.mu.Lock()
	s.fetches[symbol]++
	hold := s.hold
	if hold != nil {
		s.held++
	}
	s.mu.Unlock()
	if hold != nil {
		select {
		case <-hold:
		case <-r.Context().Done():
			return
		}
	}
	_, _ = w.Write(s.streams.snapshotJSON(symbol))
}

// holdSnapshots makes snapshot requests wait, as they do behind a busy rate limiter, until the returned func is called
func (s *spotDepthServer) holdSnapshots() (release func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hold := make(chan struct{})
	s.hold = hold
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.hold = nil
		close(hold)
	}
}

// heldSnapshots returns how many snapshot requests have been held
func (s *spotDepthServer) heldSnapshots() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held
}

// snapshotFetches returns how many snapshots of each pair were requested
func (s *spotDepthServer) snapshotFetches(pairs currency.Pairs) map[currency.Pair]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	fetches := make(map[currency.Pair]int, len(pairs))
	for _, p := range pairs {
		fetches[p] = s.fetches[p.Base.String()+p.Quote.String()]
	}
	return fetches
}

// requireBooksFollow waits until every pair's book follows its stream beyond the update IDs in after, and returns the
// books' update IDs. A book follows when its bid at 100 is its last update ID, the frame that set it
func requireBooksFollow(t *testing.T, ex *Exchange, pairs currency.Pairs, after map[currency.Pair]int64) map[currency.Pair]int64 {
	t.Helper()
	updateIDs := make(map[currency.Pair]int64, len(pairs))
	require.Eventually(t, func() bool {
		for _, p := range pairs {
			book, err := ex.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
			if err != nil || len(book.Bids) == 0 || book.Bids[0].Amount != float64(book.LastUpdateID) || book.LastUpdateID <= after[p] {
				return false
			}
			updateIDs[p] = book.LastUpdateID
		}
		return true
	}, 10*time.Second, 10*time.Millisecond, "every book must follow its stream")
	return updateIDs
}

// spotDepthTestPairs returns n spot pairs, all available and the first enabled of them enabled
func spotDepthTestPairs(t *testing.T, ex *Exchange, n, enabled int) currency.Pairs {
	t.Helper()
	pairs := make(currency.Pairs, n)
	for i := range pairs {
		pairs[i] = currency.NewPairWithDelimiter(fmt.Sprintf("COIN%02d", i), "USDT", "-")
	}
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, pairs, false), "StorePairs must not error for the available pairs")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, pairs[:enabled], true), "StorePairs must not error for the enabled pairs")
	return pairs
}

func TestSpotOrderbookSyncAcrossConnections(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	srv := newSpotDepthServer(t)
	pairs := spotDepthTestPairs(t, ex, 12, 8)
	srv.use(t, ex, 4)
	orderbookSync := ex.orderbookSync
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	updateIDs := requireBooksFollow(t, ex, pairs[:8], nil)
	exp := make(map[currency.Pair]int, len(pairs))
	for _, p := range pairs[:8] {
		exp[p] = 1
	}
	require.Equal(t, exp, srv.snapshotFetches(pairs[:8]), "each book must synchronise from one snapshot")

	// Four more streams need a third connection, whose connector must leave the other books alone
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, pairs, true), "StorePairs must not error")
	require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error")
	requireBooksFollow(t, ex, pairs[:8], updateIDs)
	requireBooksFollow(t, ex, pairs[8:], nil)
	for _, p := range pairs[8:] {
		exp[p] = 1
	}
	assert.Equal(t, exp, srv.snapshotFetches(pairs), "opening a connection should not fetch the other connections' books again")
	assert.Same(t, orderbookSync, ex.orderbookSync, "connections should share the one order book synchronisation")
}

func TestSpotOrderbookSyncAfterReconnect(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	srv := newSpotDepthServer(t)
	pairs := spotDepthTestPairs(t, ex, 4, 4)
	srv.use(t, ex, 1024)
	release := srv.holdSnapshots()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	require.Eventually(t, func() bool { return srv.heldSnapshots() == len(pairs) }, 5*time.Second, 10*time.Millisecond, "every book must wait for a snapshot")

	// Shutting down while snapshots are pending, then reconnecting to streams that moved on, must not leave a book
	// without synchronisation
	require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
	release()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	requireBooksFollow(t, ex, pairs, nil)
	for p, fetches := range srv.snapshotFetches(pairs) {
		assert.Equalf(t, 2, fetches, "%s should synchronise again from one new snapshot after the outdated one", p)
	}
}

func TestStringToOrderStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]order.Status{
		"NEW":              order.New,
		"PENDING_NEW":      order.Pending,
		"PARTIALLY_FILLED": order.PartiallyFilled,
		"FILLED":           order.Filled,
		"CANCELED":         order.Cancelled,
		"PENDING_CANCEL":   order.PendingCancel,
		"REJECTED":         order.Rejected,
		"EXPIRED":          order.Expired,
		"EXPIRED_IN_MATCH": order.Expired,
	} {
		got, err := stringToOrderStatus(status)
		require.NoErrorf(t, err, "stringToOrderStatus must not error for %s", status)
		assert.Equalf(t, exp, got, "stringToOrderStatus should convert %s", status)
	}
	_, err := stringToOrderStatus("LOL")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "stringToOrderStatus should reject an unknown status")
}

// expectedSpotSubscriptions returns the subscriptions of the enabled spot pairs for channels, keyed by stream name
func expectedSpotSubscriptions(tb testing.TB, ex *Exchange, channels ...*subscription.Subscription) subscription.List {
	tb.Helper()
	pairs, err := ex.GetEnabledPairs(asset.Spot)
	require.NoError(tb, err, "GetEnabledPairs must not error")
	exp := make(subscription.List, 0, len(channels)*len(pairs))
	for _, p := range pairs {
		for _, c := range channels {
			sub := c.Clone()
			sub.Asset = asset.Spot
			sub.Pairs = currency.Pairs{p}
			sub.QualifiedChannel = strings.ToLower(p.Base.String()+p.Quote.String()) + "@" + c.QualifiedChannel
			exp = append(exp, sub)
		}
	}
	return exp
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, expectedSpotSubscriptions(
		t, ex,
		&subscription.Subscription{Channel: subscription.CandlesChannel, QualifiedChannel: "kline_1m", Interval: kline.OneMin},
		&subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: "depth@100ms", Interval: kline.HundredMilliseconds},
		&subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "ticker"},
		&subscription.Subscription{Channel: subscription.AllTradesChannel, QualifiedChannel: "trade"},
	), subs)

	// Subscriptions without an asset are spot's, and the other assets' are left to their own connections
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel},
		{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures},
		{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.OneDay},
	}
	subs, err = ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, expectedSpotSubscriptions(
		t, ex,
		&subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "ticker"},
		&subscription.Subscription{Channel: subscription.CandlesChannel, QualifiedChannel: "kline_1d", Interval: kline.OneDay},
	), subs)

	// A subscription that cannot be expanded leaves the others, as partial generation
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot},
		{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.TwoDay},
	}
	subs, err = ex.generateSubscriptions()
	require.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generateSubscriptions must report partial generation")
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "generateSubscriptions must report the failed subscription's error")
	testsubs.EqualLists(t, expectedSpotSubscriptions(
		t, ex,
		&subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "ticker"},
	), subs)

	// Partial depth would overwrite the books diff depth keeps, so only the diff depth subscription is made
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Levels: 20, Interval: kline.HundredMilliseconds},
		{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Interval: kline.HundredMilliseconds},
	}
	subs, err = ex.generateSubscriptions()
	require.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generateSubscriptions must report partial generation")
	require.ErrorIs(t, err, errOrderbookSubscriptionConflict, "generateSubscriptions must report the conflicting order book subscription")
	testsubs.EqualLists(t, expectedSpotSubscriptions(
		t, ex,
		&subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: "depth@100ms", Interval: kline.HundredMilliseconds},
	), subs)

	// Spot disabled leaves nothing to subscribe to, so the connection is not made
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
	subs, err = ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error with spot disabled")
	assert.Empty(t, subs, "generateSubscriptions should generate nothing with spot disabled")
}

func TestOneOrderbookSubscription(t *testing.T) {
	t.Parallel()
	// Subscriptions arrive expanded, with one pair each
	btc, eth := currency.Pairs{currency.NewBTCUSDT()}, currency.Pairs{currency.NewPair(currency.ETH, currency.USDT)}
	tickerSub := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: btc, QualifiedChannel: "btcusdt@ticker"}
	diff := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: btc, Interval: kline.HundredMilliseconds, QualifiedChannel: "btcusdt@depth@100ms"}
	diffEverySecond := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: btc, QualifiedChannel: "btcusdt@depth"}
	partial5 := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: btc, Levels: 5, QualifiedChannel: "btcusdt@depth5"}
	partial20 := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: btc, Levels: 20, Interval: kline.HundredMilliseconds, QualifiedChannel: "btcusdt@depth20@100ms"}
	otherPairDiff := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: eth, Interval: kline.HundredMilliseconds, QualifiedChannel: "ethusdt@depth@100ms"}
	for _, tc := range []struct {
		name     string
		subs     subscription.List
		exp      subscription.List
		conflict bool
	}{
		{name: "no order book", subs: subscription.List{tickerSub}, exp: subscription.List{tickerSub}},
		{name: "partial depth alone", subs: subscription.List{partial5, tickerSub}, exp: subscription.List{partial5, tickerSub}},
		{name: "depths of different pairs", subs: subscription.List{partial5, otherPairDiff}, exp: subscription.List{partial5, otherPairDiff}},
		{name: "diff depth after partial depth", subs: subscription.List{partial20, tickerSub, diff, otherPairDiff}, exp: subscription.List{tickerSub, diff, otherPairDiff}, conflict: true},
		{name: "two diff depths", subs: subscription.List{diff, diffEverySecond}, exp: subscription.List{diff}, conflict: true},
		{name: "two partial depths", subs: subscription.List{partial5, partial20}, exp: subscription.List{partial5}, conflict: true},
	} {
		got, err := oneOrderbookSubscription(tc.subs)
		assert.Equalf(t, tc.exp, got, "oneOrderbookSubscription should keep one order book subscription for each pair with %s", tc.name)
		if tc.conflict {
			assert.ErrorIsf(t, err, errOrderbookSubscriptionConflict, "oneOrderbookSubscription should report the conflict with %s", tc.name)
		} else {
			assert.NoErrorf(t, err, "oneOrderbookSubscription should not error with %s", tc.name)
		}
	}
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp string
		err error
	}{
		{sub: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: "ticker"},
		{sub: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: "trade"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThousandMilliseconds}, exp: "kline_1s"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.FifteenMin}, exp: "kline_15m"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneDay}, exp: "kline_1d"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThreeDay}, exp: "kline_3d"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneWeek}, exp: "kline_1w"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneMonth}, exp: "kline_1M"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.TwoDay}, err: kline.ErrUnsupportedInterval},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel}, exp: "depth"},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.ThousandMilliseconds}, exp: "depth"},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds}, exp: "depth@100ms"},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 5}, exp: "depth5"},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 20, Interval: kline.HundredMilliseconds}, exp: "depth20@100ms"},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 7}, err: errUnsupportedLevels},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.FiveHundredMilliseconds}, err: errUnsupportedFrequency},
		{sub: &subscription.Subscription{Channel: subscription.MyOrdersChannel}, err: errUnsupportedChannel},
	} {
		got, err := channelName(tc.sub)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "channelName should error for %v", tc.sub)
			continue
		}
		require.NoErrorf(t, err, "channelName must not error for %v", tc.sub)
		assert.Equalf(t, tc.exp, got, "channelName should name %v", tc.sub)
	}
}

func TestIntervalToString(t *testing.T) {
	t.Parallel()
	for _, i := range klineIntervalList {
		got, err := intervalToString(i.Interval)
		require.NoErrorf(t, err, "intervalToString must not error for %s", i.Interval)
		assert.Equalf(t, i.String, got, "intervalToString should format %s", i.Interval)
	}
	_, err := intervalToString(kline.TwoDay)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "intervalToString should reject an unsupported interval")
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	exp := expectedSpotSubscriptions(
		t, ex,
		&subscription.Subscription{Channel: subscription.CandlesChannel, QualifiedChannel: "kline_1m", Interval: kline.OneMin},
		&subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: "depth@100ms", Interval: kline.HundredMilliseconds},
		&subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "ticker"},
		&subscription.Subscription{Channel: subscription.AllTradesChannel, QualifiedChannel: "trade"},
	)
	if !mockTests {
		// Binance streams the subscriptions, so the test checks them against the real server
		useWebsocket(t, ex, ex.spotStreamConnectionSetup(ex.Config))
	} else {
		mock := func(tb testing.TB, msg []byte, w *gws.Conn) error {
			tb.Helper()
			var req WsPayload
			require.NoError(tb, json.Unmarshal(msg, &req), "Unmarshal must not error")
			assert.Contains(tb, []string{wsSubscribeMethod, wsUnsubscribeMethod}, req.Method, "method should be SUBSCRIBE or UNSUBSCRIBE")
			assert.ElementsMatch(tb, exp.QualifiedChannels(), req.Params, "request should name every stream")
			return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"result":null,"id":%q}`, req.ID))
		}
		useMockWebsocket(t, ex, ex.spotStreamConnectionSetup(ex.Config), mock)
	}
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must subscribe without error")
	subs := ex.Websocket.GetSubscriptions()
	testsubs.EqualLists(t, exp, subs.Clone())
	for _, s := range subs {
		assert.Equal(t, subscription.SubscribedState, s.State(), "subscription should be subscribed")
	}
	conn, err := ex.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err, "GetConnection must not error")
	require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, subs), "UnsubscribeChannels must not error")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "UnsubscribeChannels should remove every subscription")
}

func TestSubscribeBadResp(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	mock := func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		var req WsPayload
		require.NoError(tb, json.Unmarshal(msg, &req), "Unmarshal must not error")
		return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"result":{"error":"carrots"},"id":%q}`, req.ID))
	}
	setup := ex.spotStreamConnectionSetup(ex.Config)
	setup.SubscriptionsNotRequired = true
	useMockWebsocket(t, ex, setup, mock)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	conn, err := ex.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err, "GetConnection must not error")
	err = ex.Subscribe(t.Context(), conn, subscription.List{{Channel: subscription.TickerChannel, QualifiedChannel: "moons@ticker"}})
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "Subscribe must error ErrSubscriptionFailure")
	require.ErrorIs(t, err, common.ErrUnknownError, "Subscribe must error ErrUnknownError")
	assert.ErrorContains(t, err, "carrots", "Subscribe should error containing the response")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "a failed subscription should be removed")
}

func TestSetupSpotConnections(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.Websocket.CreateTestConnection(asset.Spot)
	require.NoError(t, err, "the spot stream connection must be registered")
	_, err = ex.Websocket.CreateTestConnection(spotWebsocketAPI)
	require.NoError(t, err, "the WebSocket API connection must be registered with spot and margin enabled")
	assert.NotNil(t, ex.orderbookSync, "Setup should create the order book synchronisation every connection shares")

	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	cfg := &config.Config{}
	require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true), "LoadConfig must not error")
	exchCfg, err := cfg.GetExchangeConfig("Binance")
	require.NoError(t, err, "GetExchangeConfig must not error")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		require.NoErrorf(t, exchCfg.CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error for %s", a)
	}
	ex = new(Exchange)
	ex.SetDefaults()
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(t, ex.Setup(exchCfg), "Setup must not error with spot and margin disabled")
	_, err = ex.Websocket.CreateTestConnection(spotWebsocketAPI)
	assert.ErrorIs(t, err, websocket.ErrRequestRouteNotFound, "the WebSocket API connection should not be registered with spot and margin disabled")
}
