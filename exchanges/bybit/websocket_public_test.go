package bybit

import (
	"bytes"
	"errors"
	"maps"
	"strconv"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	fixtureOptionPair        = currency.NewPairWithDelimiter("BTC", "24SEP27-170000-P-USDT", currency.DashDelimiter)
	fixtureUSDTFuturesPair   = currency.NewPair(currency.BTC, currency.NewCode("USDT-16OCT26"))
	fixturePreListingPair    = currency.NewPair(currency.NewCode("NEW"), currency.USDT)
	fixtureUSDCPerpetualPair = currency.NewPair(currency.BTC, currency.PERP)
)

// streamTestInstance returns an exchange of its own for a stream test, holding the pairs the fixtures use,
// or the live pairs for a live test, named for the test to keep its tickers and orderbooks apart
func streamTestInstance(t *testing.T) *Exchange {
	t.Helper()
	if mockTests {
		return fixtureStreamInstance(t)
	}
	ex := websocketTestInstance(t)
	testexch.UpdatePairsOnce(t, ex)
	ex.Name = t.Name()
	return ex
}

// fixtureStreamInstance returns an exchange of its own holding the pairs the fixtures use, named for the test to keep
// its tickers and orderbooks apart, for tests that process recorded frames in mock and live runs alike
func fixtureStreamInstance(t *testing.T) *Exchange {
	t.Helper()
	ex := websocketTestInstance(t)
	for a, pairs := range map[asset.Item]currency.Pairs{
		asset.Spot:                {currency.NewBTCUSDT()},
		asset.USDTMarginedFutures: {currency.NewBTCUSDT(), fixtureUSDTFuturesPair, fixturePreListingPair},
		asset.USDCMarginedFutures: {fixtureUSDCPerpetualPair},
		asset.CoinMarginedFutures: {currency.NewBTCUSD()},
		asset.Options:             {fixtureOptionPair},
	} {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, false), "StorePairs must not error for %s available pairs", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, true), "StorePairs must not error for %s enabled pairs", a)
	}
	withMockCredentials(ex)
	ex.Name = t.Name()
	return ex
}

// liveOptionPair returns an enabled option pair of a live test instance
func liveOptionPair(t *testing.T, ex *Exchange) currency.Pair {
	t.Helper()
	pairs, err := ex.GetEnabledPairs(asset.Options)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.NotEmpty(t, pairs, "an option pair must be enabled")
	return pairs[0]
}

// relayed drains the data handler of ex
func relayed(ex *Exchange) []any {
	var data []any
	for {
		select {
		case payload := <-ex.Websocket.DataHandler.C:
			data = append(data, payload.Data)
		default:
			return data
		}
	}
}

// fillDataHandler gives ex a data handler whose buffer is full, and returns the error the handler reports for a
// payload it cannot take
func fillDataHandler(t *testing.T, ex *Exchange) error {
	t.Helper()
	full := stream.NewRelay(1)
	require.NoError(t, full.Send(t.Context(), "filler"), "Send must not error for the payload that fills the buffer")
	errBufferFull := errors.Unwrap(full.Send(t.Context(), "overflow"))
	require.Error(t, errBufferFull, "Send must error for a payload beyond the buffer")
	ex.Websocket.DataHandler = full
	return errBufferFull
}

func TestWebsocketHandleDataTopics(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	err := ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"liquidation.BTCUSDT","ts":1673251091822,"type":"snapshot","data":{}}`))
	require.ErrorIs(t, err, errUnhandledStreamData, "wsHandleData must error for a topic Bybit retired")
	assert.ErrorContains(t, err, "liquidation.BTCUSDT", "error should name the topic")
	err = ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"tickers_lt.BTC3LUSDT","ts":1672325446847,"type":"snapshot","data":{}}`))
	require.ErrorIs(t, err, errUnhandledStreamData, "wsHandleData must error for the retired leveraged token topics")
	err = ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":1}`))
	require.Error(t, err, "wsHandleData must error for a topic that is not a string")
	err = ex.wsHandleData(t.Context(), &FixtureConnection{}, asset.Spot, []byte(`{"success":true,"ret_msg":"pong","conn_id":"0970e817-426e-429a-a679-ff7f55e0b16a","op":"ping"}`))
	require.NoError(t, err, "wsHandleData must handle a pong")
	err = ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"kline.1","data":[],"ts":1672324988882,"type":"snapshot"}`))
	require.ErrorIs(t, err, errMalformedTopic, "wsHandleData must error for a kline topic without a symbol")
	for _, topic := range []string{"orderbook.50.BTCUSDT", "orderbook.rpi.BTCUSDT", "orderbook.full.BTCUSDT", "tickers.BTCUSDT", "kline.1.BTCUSDT", "allLiquidation.BTCUSDT", "insurance.USDT", "priceLimit.BTCUSDT", "adlAlert.USDT"} {
		err = ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"`+topic+`","data":"bad"}`))
		assert.Errorf(t, err, "wsHandleData should error for an undecodable %s push", topic)
	}
	ex.SetTradeFeedStatus(true)
	err = ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"publicTrade.BTCUSDT","data":"bad"}`))
	assert.Error(t, err, "wsHandleData should error for an undecodable trade push")
}

func TestRelayWebsocketResponse(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	frames := fixtureFrames(t, "testdata/wsPublicPriceLimit.json")
	require.NoError(t, relayWebsocketResponse[WebsocketPriceLimitUpdate](t.Context(), ex, frames[0]), "relayWebsocketResponse must not error")
	exp := []any{
		&WebsocketPriceLimitUpdate{
			Topic:     "priceLimit.BTCUSDT",
			Timestamp: types.Time(time.UnixMilli(1791535781983)),
			Data: WebsocketPriceLimit{
				Symbol:    "BTCUSDT",
				BuyLimit:  83385.2,
				SellLimit: 81734.1,
			},
		},
	}
	assert.Equal(t, exp, relayed(ex), "relayWebsocketResponse should relay the push decoded")

	err := relayWebsocketResponse[WebsocketPriceLimitUpdate](t.Context(), ex, []byte(`{"topic":"priceLimit.BTCUSDT","data":"bad"}`))
	require.Error(t, err, "relayWebsocketResponse must error for a push that does not decode")
	assert.Empty(t, relayed(ex), "relayWebsocketResponse should not relay a push that does not decode")

	errBufferFull := fillDataHandler(t, ex)
	err = relayWebsocketResponse[WebsocketPriceLimitUpdate](t.Context(), ex, frames[0])
	require.ErrorIs(t, err, errBufferFull, "relayWebsocketResponse must return the data handler's error")
}

func TestWebsocketOrderbook(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	if !mockTests {
		option := liveOptionPair(t, ex)
		optionSymbol, err := ex.symbol(option, asset.Options)
		require.NoError(t, err, "symbol must not error")
		for _, tc := range []struct {
			streamURL string
			topic     string
			asset     asset.Item
			pair      currency.Pair
			frames    int
		}{
			{spotPublic, "orderbook.50.BTCUSDT", asset.Spot, currency.NewBTCUSDT(), 2},
			{optionPublic, "orderbook.25." + optionSymbol, asset.Options, option, 1},
		} {
			frames, _ := liveStreamFrames(t, tc.streamURL, 20*time.Second, tc.frames, tc.topic)
			require.NotEmptyf(t, frames, "%s must push", tc.topic)
			for _, frame := range frames {
				require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, tc.asset, frame), "wsHandleData must not error for %s", tc.topic)
			}
			_, err := ex.Websocket.Orderbook.GetOrderbook(tc.pair, tc.asset)
			require.NoErrorf(t, err, "GetOrderbook must not error for %s", tc.topic)
		}
		return
	}
	frames := fixtureFrames(t, "testdata/wsPublicOrderbook.json")
	pushes := decodeFramesStrictly[WebsocketOrderbookUpdate](t, frames)
	expPushes := []WebsocketOrderbookUpdate{
		{
			Topic:     "orderbook.50.BTCUSDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791535765476)),
			Data: WebsocketOrderbook{
				Symbol: "BTCUSDT",
				Bids: []orderbook.Level{
					{
						Amount: 2.080933,
						Price:  82590.5,
					},
					{
						Amount: 0.968648,
						Price:  82589.4,
					},
					{
						Amount: 0.025557,
						Price:  82589.1,
					},
				},
				Asks: []orderbook.Level{
					{
						Amount: 0.412778,
						Price:  82590.6,
					},
					{
						Amount: 0.0242,
						Price:  82591.2,
					},
					{
						Amount: 0.125,
						Price:  82592,
					},
				},
				UpdateID: 117779840,
				Sequence: 115179399571,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1791535765473)),
		},
		{
			Topic:     "orderbook.50.BTCUSDT",
			Type:      "delta",
			Timestamp: types.Time(time.UnixMilli(1791535765496)),
			Data: WebsocketOrderbook{
				Symbol: "BTCUSDT",
				Bids: []orderbook.Level{
					{
						Amount: 1.002651,
						Price:  82589.4,
					},
					{
						Price: 82589.1,
					},
				},
				Asks: []orderbook.Level{
					{
						Amount: 0.398812,
						Price:  82590.6,
					},
					{
						Amount: 0.004,
						Price:  82591.9,
					},
				},
				UpdateID: 117779841,
				Sequence: 115179399630,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1791535765494)),
		},
		{
			Topic:     "orderbook.25.BTC-24SEP27-170000-P-USDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791535794734)),
			ID:        "orderbook.25.BTC-24SEP27-170000-P-USDT-77632995738-1791535794734",
			Data: WebsocketOrderbook{
				Symbol: "BTC-24SEP27-170000-P-USDT",
				Bids: []orderbook.Level{
					{
						Amount: 18.55,
						Price:  83310,
					},
					{
						Amount: 0.96,
						Price:  5,
					},
				},
				Asks: []orderbook.Level{
					{
						Amount: 18.55,
						Price:  85565,
					},
				},
				UpdateID: 162,
				Sequence: 77632995738,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1791533657213)),
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every orderbook field")

	for i, a := range []asset.Item{asset.Spot, asset.Spot, asset.Options} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, a, frames[i]), "wsHandleData must not error for frame %d", i)
	}
	books := make([]*orderbook.Book, 0, 2)
	for _, k := range []struct {
		pair  currency.Pair
		asset asset.Item
	}{{currency.NewBTCUSDT(), asset.Spot}, {fixtureOptionPair, asset.Options}} {
		book, err := ex.Websocket.Orderbook.GetOrderbook(k.pair, k.asset)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", k.asset)
		book.InsertedAt = time.Time{}
		books = append(books, book)
	}
	exp := []*orderbook.Book{
		{
			Bids: []orderbook.Level{
				{
					Amount: 2.080933,
					Price:  82590.5,
				},
				{
					Amount: 1.002651,
					Price:  82589.4,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 0.398812,
					Price:  82590.6,
				},
				{
					Amount: 0.0242,
					Price:  82591.2,
				},
				{
					Amount: 0.004,
					Price:  82591.9,
				},
				{
					Amount: 0.125,
					Price:  82592,
				},
			},
			Exchange:     "TestWebsocketOrderbook",
			Pair:         currency.NewBTCUSDT(),
			Asset:        asset.Spot,
			LastUpdated:  time.UnixMilli(1791535765494),
			LastPushed:   time.UnixMilli(1791535765496),
			LastUpdateID: 117779841,
		},
		{
			Bids: []orderbook.Level{
				{
					Amount: 18.55,
					Price:  83310,
				},
				{
					Amount: 0.96,
					Price:  5,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 18.55,
					Price:  85565,
				},
			},
			Exchange:     "TestWebsocketOrderbook",
			Pair:         currency.NewPairWithDelimiter("BTC", "24SEP27-170000-P-USDT", "-"),
			Asset:        asset.Options,
			LastUpdated:  time.UnixMilli(1791533657213),
			LastPushed:   time.UnixMilli(1791535794734),
			LastUpdateID: 162,
		},
	}
	assert.Equal(t, exp, books, "wsHandleData should store each snapshot with its deltas applied")
	err := ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"orderbook.50.ZZZQUSDT","ts":1791535765476,"type":"snapshot","data":{"s":"ZZZQUSDT","b":[],"a":[],"u":1,"seq":1},"cts":1791535765473}`))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "wsHandleData must error for an orderbook of an unknown pair")
}

func TestWebsocketProcessOrderbook(t *testing.T) {
	t.Parallel()
	frames := fixtureFrames(t, "testdata/wsPublicOrderbook.json")
	require.Len(t, frames, 3, "the orderbook fixture must hold a spot snapshot, a spot delta and an option snapshot")
	ex := fixtureStreamInstance(t)
	for i, a := range []asset.Item{asset.Spot, asset.Spot, asset.Options} {
		require.NoErrorf(t, ex.wsProcessOrderbook(t.Context(), a, frames[i]), "wsProcessOrderbook must not error for frame %d", i)
	}
	assert.Len(t, relayed(ex), len(frames), "wsProcessOrderbook should publish the book after each push")
	spot, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error for the spot book")
	option, err := ex.Websocket.Orderbook.GetOrderbook(fixtureOptionPair, asset.Options)
	require.NoError(t, err, "GetOrderbook must not error for the option book")
	spot.InsertedAt, option.InsertedAt = time.Time{}, time.Time{}
	exp := []*orderbook.Book{
		{
			Bids: []orderbook.Level{
				{
					Amount: 2.080933,
					Price:  82590.5,
				},
				{
					Amount: 1.002651,
					Price:  82589.4,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 0.398812,
					Price:  82590.6,
				},
				{
					Amount: 0.0242,
					Price:  82591.2,
				},
				{
					Amount: 0.004,
					Price:  82591.9,
				},
				{
					Amount: 0.125,
					Price:  82592,
				},
			},
			Exchange:     "TestWebsocketProcessOrderbook",
			Pair:         currency.NewBTCUSDT(),
			Asset:        asset.Spot,
			LastUpdated:  time.UnixMilli(1791535765494),
			LastPushed:   time.UnixMilli(1791535765496),
			LastUpdateID: 117779841,
		},
		{
			Bids: []orderbook.Level{
				{
					Amount: 18.55,
					Price:  83310,
				},
				{
					Amount: 0.96,
					Price:  5,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 18.55,
					Price:  85565,
				},
			},
			Exchange:     "TestWebsocketProcessOrderbook",
			Pair:         fixtureOptionPair,
			Asset:        asset.Options,
			LastUpdated:  time.UnixMilli(1791533657213),
			LastPushed:   time.UnixMilli(1791535794734),
			LastUpdateID: 162,
		},
	}
	assert.Equal(t, exp, []*orderbook.Book{spot, option}, "wsProcessOrderbook should load each snapshot and apply its deltas, timed by the matching engine")

	for _, tc := range []struct {
		name  string
		frame []byte
		err   error
	}{
		{name: "delta without a snapshot", frame: frames[1], err: orderbook.ErrDepthNotFound},
		{name: "unknown pair", frame: bytes.ReplaceAll(frames[0], []byte("BTCUSDT"), []byte("ZZZQUSDT")), err: currency.ErrPairNotFound},
		{name: "undecodable push", frame: []byte(`{"topic":"orderbook.50.BTCUSDT","type":"snapshot","data":"bad"}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := fixtureStreamInstance(t)
			err := ex.wsProcessOrderbook(t.Context(), asset.Spot, tc.frame)
			if tc.err == nil {
				require.Error(t, err, "wsProcessOrderbook must error for a push that does not decode")
			} else {
				require.ErrorIs(t, err, tc.err, "wsProcessOrderbook must return the expected error")
			}
			_, err = ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.Spot)
			assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "wsProcessOrderbook should store no book for a push it rejects")
			assert.Empty(t, relayed(ex), "wsProcessOrderbook should publish nothing for a push it rejects")
		})
	}
}

func TestWebsocketRPIOrderbook(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	frames := liveOrFixtureFrames(t, spotPublic, "testdata/wsPublicRPIOrderbook.json", "orderbook.rpi.BTCUSDT")
	for _, frame := range frames {
		require.NoError(t, ex.wsHandleData(t.Context(), nil, asset.Spot, frame), "wsHandleData must not error")
	}
	data := relayed(ex)
	require.Len(t, data, len(frames), "every push must be relayed")
	if !mockTests {
		assert.IsType(t, (*WebsocketRPIOrderbookUpdate)(nil), data[0], "pushes should be relayed decoded")
		return
	}
	pushes := decodeFramesStrictly[WebsocketRPIOrderbookUpdate](t, frames)
	expPushes := []WebsocketRPIOrderbookUpdate{
		{
			Topic:     "orderbook.rpi.BTCUSDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791535764116)),
			Data: WebsocketRPIOrderbook{
				Symbol: "BTCUSDT",
				Bids: []RPIOrderbookLevel{
					{
						Price:      82590.5,
						NonRPISize: 1.976699,
						RPISize:    0.260218,
					},
					{
						Price:   82590.2,
						RPISize: 0.29,
					},
				},
				Asks: []RPIOrderbookLevel{
					{
						Price:      82590.6,
						NonRPISize: 0.412778,
					},
					{
						Price:   82592.4,
						RPISize: 0.175695,
					},
				},
				UpdateID: 114982077,
				Sequence: 115179398712,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1791535764110)),
		},
		{
			Topic:     "orderbook.rpi.BTCUSDT",
			Type:      "delta",
			Timestamp: types.Time(time.UnixMilli(1791535764216)),
			Data: WebsocketRPIOrderbook{
				Symbol: "BTCUSDT",
				Bids: []RPIOrderbookLevel{
					{
						Price:      82590.5,
						NonRPISize: 1.982752,
						RPISize:    0.260218,
					},
					{
						Price: 82590.2,
					},
				},
				Asks: []RPIOrderbookLevel{
					{
						Price:      82602.7,
						NonRPISize: 0.00089,
						RPISize:    0.175695,
					},
				},
				UpdateID: 114982078,
				Sequence: 115179398744,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1791535764211)),
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every RPI orderbook field")
	for i := range pushes {
		assert.Equalf(t, &pushes[i], data[i], "push %d should be relayed decoded", i)
	}
}

// liveOrFixtureFrames returns the fixture's frames for a mock test, or the frames a live stream pushes for topic
func liveOrFixtureFrames(t *testing.T, streamURL, fixture, topic string) [][]byte {
	t.Helper()
	if mockTests {
		return fixtureFrames(t, fixture)
	}
	frames, _ := liveStreamFrames(t, streamURL, 20*time.Second, 2, topic)
	require.NotEmptyf(t, frames, "%s must push", topic)
	return frames
}

func TestWebsocketFullOrderbook(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	frames := liveOrFixtureFrames(t, linearPublic, "testdata/wsPublicFullOrderbook.json", "orderbook.full.BTCUSDT")
	for _, frame := range frames {
		require.NoError(t, ex.wsHandleData(t.Context(), nil, asset.USDTMarginedFutures, frame), "wsHandleData must not error")
	}
	data := relayed(ex)
	require.Len(t, data, len(frames), "every push must be relayed")
	_, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "full depth deltas must not build an orderbook without a snapshot")
	if !mockTests {
		assert.IsType(t, (*WebsocketOrderbookUpdate)(nil), data[0], "pushes should be relayed decoded")
		return
	}
	pushes := decodeFramesStrictly[WebsocketOrderbookUpdate](t, frames)
	expPushes := []WebsocketOrderbookUpdate{
		{
			Topic:     "orderbook.full.BTCUSDT",
			Type:      "delta",
			Timestamp: types.Time(time.UnixMilli(1791535782472)),
			Data: WebsocketOrderbook{
				Symbol: "BTCUSDT",
				Bids: []orderbook.Level{
					{
						Amount: 9.4,
						Price:  82560.9,
					},
					{
						Price: 82548.5,
					},
				},
				Asks: []orderbook.Level{
					{
						Amount: 0.345,
						Price:  82561,
					},
					{
						Price: 82567.3,
					},
				},
				UpdateID: 27680112,
				Sequence: 822947987140,
			},
			CreationTimestamp: types.Time(time.UnixMilli(1791535782467)),
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every full orderbook field")
	for i := range pushes {
		assert.Equalf(t, &pushes[i], data[i], "push %d should be relayed decoded", i)
	}
}

func TestWebsocketPublicTrade(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	ex.SetTradeFeedStatus(true)
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	if !mockTests {
		frames, _ := liveStreamFrames(t, linearPublic, 20*time.Second, 1, "publicTrade.BTCUSDT")
		require.NotEmpty(t, frames, "the trade topic must push")
		require.NoError(t, ex.wsHandleData(t.Context(), nil, asset.USDTMarginedFutures, frames[0]), "wsHandleData must not error")
		data := relayed(ex)
		require.Len(t, data, 1, "trades must be relayed")
		assert.IsType(t, []trade.Data{}, data[0], "trades should be relayed as trade data")
		// Option trades are pushed by base coin, for contracts that may not be enabled here. They can be more than 20
		// seconds apart, and the option stream accepts any topic name, so the wait is longer and only a pushed trade
		// proves the topic
		frames, _ = liveStreamFrames(t, optionPublic, time.Minute, 1, "publicTrade."+liveOptionPair(t, ex).Base.Upper().String())
		require.NotEmpty(t, frames, "the option trade topic must push")
		require.NoError(t, ex.wsHandleData(t.Context(), nil, asset.Options, frames[0]), "wsHandleData must not error for option trades")
		return
	}
	frames := fixtureFrames(t, "testdata/wsPublicTrade.json")
	pushes := decodeFramesStrictly[WebsocketPublicTradeUpdate](t, frames)
	expPushes := []WebsocketPublicTradeUpdate{
		{
			Topic:     "publicTrade.BTCUSDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791535765239)),
			Data: []WebsocketPublicTrade{
				{
					OrderFillTimestamp: types.Time(time.UnixMilli(1791535765237)),
					Symbol:             "BTCUSDT",
					Side:               "Sell",
					Size:               0.023863,
					Price:              82590.5,
					TradeID:            "2290000001228958155",
					RPI:                true,
					Sequence:           115179399571,
				},
			},
		},
		{
			Topic:     "publicTrade.BTCUSDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791535782604)),
			Data: []WebsocketPublicTrade{
				{
					OrderFillTimestamp: types.Time(time.UnixMilli(1791535782603)),
					Symbol:             "BTCUSDT",
					Side:               "Buy",
					Size:               0.002,
					Price:              82560.9,
					TickDirection:      "ZeroMinusTick",
					TradeID:            "1b240f9b-dbb1-56d6-b380-c4f64a92420d",
					BlockTrade:         true,
					Sequence:           822947990317,
				},
			},
		},
		{
			ID:        "publicTrade.BTC-77634219074-1791535798091",
			Topic:     "publicTrade.BTC",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791535798091)),
			Data: []WebsocketPublicTrade{
				{
					OrderFillTimestamp:    types.Time(time.UnixMilli(1791535798090)),
					Symbol:                "BTC-24SEP27-170000-P-USDT",
					Side:                  "Buy",
					Size:                  0.01,
					Price:                 84130,
					TradeID:               "20350fa2-6942-57bc-8eb1-07705dd4bab3",
					Sequence:              77634219074,
					MarkPrice:             84134.38654221,
					IndexPrice:            82611.83292604,
					MarkImpliedVolatility: 0.4254,
					ImpliedVolatility:     0.4249,
				},
				{
					OrderFillTimestamp:    types.Time(time.UnixMilli(1791535798198)),
					Symbol:                "BTC-10OCT26-82750-C-USDT",
					Side:                  "Sell",
					Size:                  0.32,
					Price:                 415,
					TradeID:               "88a3f502-9202-59c0-84d4-fb1638d923fc",
					Sequence:              77634219303,
					MarkPrice:             419.0386432,
					IndexPrice:            82615.69696287,
					MarkImpliedVolatility: 0.281,
					ImpliedVolatility:     0.2785,
				},
			},
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every trade field")
	for i, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.Options} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, a, frames[i]), "wsHandleData must not error for frame %d", i)
	}
	exp := []any{
		[]trade.Data{
			{
				TID:          "2290000001228958155",
				Exchange:     "TestWebsocketPublicTrade",
				CurrencyPair: currency.NewBTCUSDT(),
				AssetType:    asset.Spot,
				Side:         order.Sell,
				Price:        82590.5,
				Amount:       0.023863,
				Timestamp:    time.UnixMilli(1791535765237),
			},
		},
		[]trade.Data{
			{
				TID:          "1b240f9b-dbb1-56d6-b380-c4f64a92420d",
				Exchange:     "TestWebsocketPublicTrade",
				CurrencyPair: currency.NewBTCUSDT(),
				AssetType:    asset.USDTMarginedFutures,
				Side:         order.Buy,
				Price:        82560.9,
				Amount:       0.002,
				Timestamp:    time.UnixMilli(1791535782603),
			},
		},
		[]trade.Data{
			{
				TID:          "20350fa2-6942-57bc-8eb1-07705dd4bab3",
				Exchange:     "TestWebsocketPublicTrade",
				CurrencyPair: currency.NewPairWithDelimiter("BTC", "24SEP27-170000-P-USDT", "-"),
				AssetType:    asset.Options,
				Side:         order.Buy,
				Price:        84130,
				Amount:       0.01,
				Timestamp:    time.UnixMilli(1791535798090),
			},
		},
	}
	assert.Equal(t, exp, relayed(ex), "wsHandleData should relay the trades of known and enabled pairs")

	ex.SetTradeFeedStatus(false)
	require.NoError(t, ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"publicTrade.BTCUSDT","data":"not decoded while trades are neither relayed nor saved"}`)), "wsHandleData must skip trades when they are neither relayed nor saved")
	err := ex.wsHandleData(t.Context(), nil, asset.Spot, frames[0])
	require.NoError(t, err, "wsHandleData must not error with the trade feed disabled")
	assert.Empty(t, relayed(ex), "trades should not be relayed with the trade feed disabled")

	ex.SetTradeFeedStatus(true)
	err = ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"publicTrade.ETHUSDT","data":[{"T":1791535765237,"s":"ZZZQUSDT","S":"Sell","v":"0.1","p":"1","i":"1"}]}`))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "wsHandleData must error for a trade of an unknown spot pair")
	err = ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"publicTrade.BTCUSDT","data":[{"T":1791535765237,"s":"BTCUSDT","S":"Hold","v":"0.1","p":"1","i":"1"}]}`))
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "wsHandleData must error for an unknown trade side")
}

func TestWebsocketProcessPublicTrade(t *testing.T) {
	t.Parallel()
	frames := fixtureFrames(t, "testdata/wsPublicTrade.json")
	require.Len(t, frames, 3, "the trade fixture must hold a spot, a linear and an option push")
	ex := fixtureStreamInstance(t)
	err := ex.wsProcessPublicTrade(asset.Spot, []byte(`{"topic":"publicTrade.BTCUSDT","data":"not decoded while trades are neither relayed nor saved"}`))
	require.NoError(t, err, "wsProcessPublicTrade must skip trades that are neither relayed nor saved")
	assert.Empty(t, relayed(ex), "wsProcessPublicTrade should relay nothing while the trade feed is disabled")

	ex.SetTradeFeedStatus(true)
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	for i, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.Options} {
		require.NoErrorf(t, ex.wsProcessPublicTrade(a, frames[i]), "wsProcessPublicTrade must not error for frame %d", i)
	}
	// The option push also holds a trade of BTC-10OCT26-82750-C-USDT, a contract the instance does not hold
	exp := []any{
		[]trade.Data{
			{
				TID:          "2290000001228958155",
				Exchange:     "TestWebsocketProcessPublicTrade",
				CurrencyPair: currency.NewBTCUSDT(),
				AssetType:    asset.Spot,
				Side:         order.Sell,
				Price:        82590.5,
				Amount:       0.023863,
				Timestamp:    time.UnixMilli(1791535765237),
			},
		},
		[]trade.Data{
			{
				TID:          "1b240f9b-dbb1-56d6-b380-c4f64a92420d",
				Exchange:     "TestWebsocketProcessPublicTrade",
				CurrencyPair: currency.NewBTCUSDT(),
				AssetType:    asset.USDTMarginedFutures,
				Side:         order.Buy,
				Price:        82560.9,
				Amount:       0.002,
				Timestamp:    time.UnixMilli(1791535782603),
			},
		},
		[]trade.Data{
			{
				TID:          "20350fa2-6942-57bc-8eb1-07705dd4bab3",
				Exchange:     "TestWebsocketProcessPublicTrade",
				CurrencyPair: fixtureOptionPair,
				AssetType:    asset.Options,
				Side:         order.Buy,
				Price:        84130,
				Amount:       0.01,
				Timestamp:    time.UnixMilli(1791535798090),
			},
		},
	}
	assert.Equal(t, exp, relayed(ex), "wsProcessPublicTrade should relay the trades of known pairs and enabled option contracts")
	// A push can carry several trades, and every one of them is relayed
	spotTrade := frames[0][bytes.IndexByte(frames[0], '[')+1 : bytes.LastIndexByte(frames[0], ']')]
	nextTrade := bytes.Replace(spotTrade, []byte(`"i":"2290000001228958155"`), []byte(`"i":"2290000001228958156"`), 1)
	require.NoError(t, ex.wsProcessPublicTrade(asset.Spot, bytes.Replace(frames[0], spotTrade, bytes.Join([][]byte{spotTrade, nextTrade}, []byte(",")), 1)), "wsProcessPublicTrade must not error for a push of two trades")
	first, ok := exp[0].([]trade.Data)
	require.True(t, ok, "the spot push's expected trades must be trade data")
	second := first[0]
	second.TID = "2290000001228958156"
	assert.Equal(t, []any{[]trade.Data{first[0], second}}, relayed(ex), "wsProcessPublicTrade should relay every trade of a push")

	noEnabledContract := bytes.ReplaceAll(frames[2], []byte("BTC-24SEP27-170000-P-USDT"), []byte("BTC-25SEP27-170000-P-USDT"))
	require.NoError(t, ex.wsProcessPublicTrade(asset.Options, noEnabledContract), "wsProcessPublicTrade must not error for an option push without an enabled contract")
	assert.Empty(t, relayed(ex), "wsProcessPublicTrade should not relay an empty batch")

	for _, tc := range []struct {
		name  string
		asset asset.Item
		frame []byte
		err   error
	}{
		{name: "an unknown pair", asset: asset.Spot, frame: bytes.Replace(frames[0], []byte(`"s":"BTCUSDT"`), []byte(`"s":"ZZZQUSDT"`), 1), err: currency.ErrPairNotFound},
		{name: "an unknown side", asset: asset.Spot, frame: bytes.Replace(frames[0], []byte(`"S":"Sell"`), []byte(`"S":"Hold"`), 1), err: order.ErrSideIsInvalid},
		{name: "an option trade without a symbol", asset: asset.Options, frame: bytes.Replace(frames[2], []byte(`"s":"BTC-24SEP27-170000-P-USDT"`), []byte(`"s":""`), 1), err: currency.ErrSymbolStringEmpty},
		{name: "an undecodable push", asset: asset.Spot, frame: []byte(`{"topic":"publicTrade.BTCUSDT","data":"bad"}`)},
	} {
		err := ex.wsProcessPublicTrade(tc.asset, tc.frame)
		if tc.err == nil {
			require.Errorf(t, err, "wsProcessPublicTrade must error for %s", tc.name)
		} else {
			require.ErrorIsf(t, err, tc.err, "wsProcessPublicTrade must error for %s", tc.name)
		}
		assert.Emptyf(t, relayed(ex), "wsProcessPublicTrade should relay nothing for %s", tc.name)
	}
}

func TestPublicTradePair(t *testing.T) {
	t.Parallel()
	ex := fixtureStreamInstance(t)
	listedContract := currency.NewPairWithDelimiter("BTC", "10OCT26-82750-C-USDT", currency.DashDelimiter)
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, currency.Pairs{fixtureOptionPair, listedContract}, false), "StorePairs must not error for the listed option contracts")
	datedFutures := currency.NewPair(currency.BTC, currency.NewCode("25DEC26"))
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDCMarginedFutures, currency.Pairs{fixtureUSDCPerpetualPair, datedFutures}, false), "StorePairs must not error for the listed USDC contracts")
	for _, tc := range []struct {
		symbol string
		asset  asset.Item
		exp    currency.Pair
		err    error
	}{
		{symbol: "BTCUSDT", asset: asset.Spot, exp: currency.NewBTCUSDT()},
		{symbol: "BTC-25DEC26", asset: asset.USDCMarginedFutures, exp: datedFutures},
		{symbol: "ZZZQUSDT", asset: asset.Spot, err: currency.ErrPairNotFound},
		{symbol: "BTC-24SEP27-170000-P-USDT", asset: asset.Options, exp: fixtureOptionPair},
		{symbol: "BTC-10OCT26-82750-C-USDT", asset: asset.Options},  // Listed but not enabled
		{symbol: "BTC-25SEP27-170000-P-USDT", asset: asset.Options}, // Not listed when the pairs were last updated
		{symbol: "", asset: asset.Options, err: currency.ErrSymbolStringEmpty},
	} {
		p, err := ex.publicTradePair(tc.symbol, tc.asset)
		require.ErrorIsf(t, err, tc.err, "publicTradePair must return the expected error for %s %q", tc.asset, tc.symbol)
		assert.Equalf(t, tc.exp, p, "publicTradePair should return the pair of %s %q, or an empty pair for an option trade to skip", tc.asset, tc.symbol)
	}
}

func TestWebsocketTicker(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	if !mockTests {
		option := liveOptionPair(t, ex)
		optionSymbol, err := ex.symbol(option, asset.Options)
		require.NoError(t, err, "symbol must not error")
		for _, tc := range []struct {
			streamURL string
			topic     string
			asset     asset.Item
			pair      currency.Pair
		}{
			{spotPublic, "tickers.BTCUSDT", asset.Spot, currency.NewBTCUSDT()},
			{linearPublic, "tickers.BTCUSDT", asset.USDTMarginedFutures, currency.NewBTCUSDT()},
			{inversePublic, "tickers.BTCUSD", asset.CoinMarginedFutures, currency.NewBTCUSD()},
			{optionPublic, "tickers." + optionSymbol, asset.Options, option},
		} {
			frames, _ := liveStreamFrames(t, tc.streamURL, 20*time.Second, 1, tc.topic)
			require.NotEmptyf(t, frames, "%s must push", tc.topic)
			for _, frame := range frames {
				require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, tc.asset, frame), "wsHandleData must not error for %s", tc.asset)
			}
			tick, err := ticker.GetTicker(ex.Name, tc.pair, tc.asset)
			require.NoErrorf(t, err, "GetTicker must not error for %s", tc.asset)
			assert.NotZerof(t, tick.LastUpdated, "%s ticker should hold the push time", tc.asset)
		}
		return
	}
	frames := fixtureFrames(t, "testdata/wsPublicTicker.json")
	pushes := decodeFramesStrictly[WebsocketTickerUpdate](t, frames)
	expPushes := []WebsocketTickerUpdate{
		{
			Topic:         "tickers.BTCUSDT",
			Type:          "snapshot",
			Timestamp:     types.Time(time.UnixMilli(1791535765477)),
			CrossSequence: 115179399822,
			Data: WebsocketTicker{
				Symbol:              "BTCUSDT",
				Price24HourPercent:  -0.0037,
				LastPrice:           82590.6,
				PreviousPrice24Hour: 82896.8,
				HighPrice24Hour:     83174,
				LowPrice24Hour:      80393.9,
				Turnover24Hour:      1118450408.4968019,
				Volume24Hour:        13646.082843,
				USDIndexPrice:       82523.544958,
			},
		},
		{
			Topic:         "tickers.BTCUSDT",
			Type:          "snapshot",
			Timestamp:     types.Time(time.UnixMilli(1791535782383)),
			CrossSequence: 822947985112,
			Data: WebsocketTicker{
				Symbol:                  "BTCUSDT",
				TickDirection:           "ZeroPlusTick",
				Price24HourPercent:      -0.003753,
				LastPrice:               82561,
				PreviousPrice24Hour:     82872.1,
				HighPrice24Hour:         83143.2,
				LowPrice24Hour:          80350.1,
				PreviousPrice1Hour:      82475,
				MarkPrice:               82559.7,
				IndexPrice:              82590.92,
				OpenInterest:            57991.212,
				OpenInterestValue:       4787737065.36,
				SingleOpenInterest:      28995.606,
				SingleOpenInterestValue: 2393868532.68,
				Turnover24Hour:          6866421328.479,
				Volume24Hour:            83950.362,
				NextFundingTime:         types.Time(time.Unix(1791561600, 0)),
				FundingRate:             0.00004412,
				Bid1Price:               82560.9,
				Bid1Size:                9.537,
				Ask1Price:               82561,
				Ask1Size:                0.751,
				FundingIntervalHour:     8,
				FundingCap:              0.00333,
			},
		},
		{
			Topic:         "tickers.BTCUSDT",
			Type:          "delta",
			Timestamp:     types.Time(time.UnixMilli(1791535782483)),
			CrossSequence: 822947989392,
			Data: WebsocketTicker{
				Symbol:             "BTCUSDT",
				TickDirection:      "MinusTick",
				Price24HourPercent: -0.003755,
				LastPrice:          82560.9,
				Turnover24Hour:     6866421411.0399,
				Volume24Hour:       83950.363,
				Bid1Price:          82560.9,
				Bid1Size:           9.4,
				Ask1Price:          82561,
				Ask1Size:           0.755,
			},
		},
		{
			Topic:         "tickers.BTCUSDT-16OCT26",
			Type:          "snapshot",
			Timestamp:     types.Time(time.UnixMilli(1791535781868)),
			CrossSequence: 16608446312,
			Data: WebsocketTicker{
				Symbol:                  "BTCUSDT-16OCT26",
				TickDirection:           "ZeroPlusTick",
				Price24HourPercent:      -0.003952,
				LastPrice:               82783.2,
				PreviousPrice24Hour:     83111.7,
				HighPrice24Hour:         83111.7,
				LowPrice24Hour:          80593,
				PreviousPrice1Hour:      82638.3,
				MarkPrice:               82696.6,
				IndexPrice:              82594.9,
				OpenInterest:            99.452,
				OpenInterestValue:       8224342.26,
				SingleOpenInterest:      49.726,
				SingleOpenInterestValue: 4112171.13,
				Turnover24Hour:          135063.2496,
				Volume24Hour:            1.644,
				Bid1Price:               82680.4,
				Bid1Size:                0.484,
				Ask1Price:               82705,
				Ask1Size:                0.484,
				DeliveryTime:            time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC),
				BasisRate:               0.0010603,
				DeliveryFeeRate:         0.0005,
				PredictedDeliveryPrice:  82611.5,
				Basis:                   193.57,
				BasisRateYear:           0.0628413,
			},
		},
		{
			Topic:         "tickers.NEWUSDT",
			Type:          "snapshot",
			Timestamp:     types.Time(time.UnixMilli(1791535781950)),
			CrossSequence: 1203345,
			Data: WebsocketTicker{
				Symbol:                 "NEWUSDT",
				TickDirection:          "PlusTick",
				MarkPrice:              0.0512,
				Bid1Price:              0.0511,
				Bid1Size:               52000,
				Ask1Price:              0.0513,
				Ask1Size:               61000,
				PreOpenPrice:           0.0512,
				PreQuantity:            120000,
				CurrentPreListingPhase: "CallAuction",
				FundingIntervalHour:    4,
				FundingCap:             0.02,
			},
		},
		{
			Topic:     "tickers.BTC-24SEP27-170000-P-USDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791535794028)),
			ID:        "tickers.BTC-24SEP27-170000-P-USDT-77634215397-1791535794028",
			Data: WebsocketTicker{
				Symbol:                     "BTC-24SEP27-170000-P-USDT",
				LastPrice:                  84120,
				HighPrice24Hour:            84305,
				LowPrice24Hour:             83950,
				MarkPrice:                  84134.38654221,
				IndexPrice:                 82611.83292604,
				OpenInterest:               1.2,
				Turnover24Hour:             1652.27,
				Volume24Hour:               0.02,
				PredictedDeliveryPrice:     84112.5,
				BidPrice:                   83310,
				BidSize:                    18.55,
				BidImpliedVolatility:       0.337,
				AskPrice:                   85565,
				AskSize:                    18.55,
				AskImpliedVolatility:       0.5147,
				MarkPriceImpliedVolatility: 0.4254,
				UnderlyingPrice:            87001.2,
				TotalVolume:                3,
				TotalTurnover:              247836,
				Delta:                      -0.91927665,
				Gamma:                      0.00000414,
				Vega:                       127.51390405,
				Theta:                      -7.74856765,
				Change24Hour:               0.00167063,
			},
		},
		{
			Topic:         "tickers.BTCUSD",
			Type:          "snapshot",
			Timestamp:     types.Time(time.UnixMilli(1791535789460)),
			CrossSequence: 118955330421,
			Data: WebsocketTicker{
				Symbol:                  "BTCUSD",
				TickDirection:           "ZeroPlusTick",
				Price24HourPercent:      -0.003757,
				LastPrice:               82502.6,
				PreviousPrice24Hour:     82813.8,
				HighPrice24Hour:         83092.3,
				LowPrice24Hour:          80302.9,
				PreviousPrice1Hour:      82414.6,
				MarkPrice:               82508.23,
				IndexPrice:              82536.06,
				OpenInterest:            516533432,
				OpenInterestValue:       6260.39,
				SingleOpenInterest:      258266716,
				SingleOpenInterestValue: 3130.19,
				Turnover24Hour:          4571.2059,
				Volume24Hour:            373935316,
				NextFundingTime:         types.Time(time.Unix(1791561600, 0)),
				FundingRate:             0.0001,
				Bid1Price:               82508,
				Bid1Size:                119791,
				Ask1Price:               82508.1,
				Ask1Size:                3500,
				FundingIntervalHour:     8,
				FundingCap:              0.005,
			},
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every ticker field")
	assets := []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.USDTMarginedFutures, asset.USDTMarginedFutures, asset.USDTMarginedFutures, asset.Options, asset.CoinMarginedFutures}
	require.Len(t, frames, len(assets), "every frame must have an asset")
	for i, a := range assets {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, a, frames[i]), "wsHandleData must not error for frame %d", i)
	}
	exp := []any{
		&ticker.Price{
			Last:         82590.6,
			High:         83174,
			Low:          80393.9,
			BaseVolume:   13646.082843,
			QuoteVolume:  1118450408.4968019,
			Pair:         currency.NewBTCUSDT(),
			ExchangeName: "TestWebsocketTicker",
			AssetType:    asset.Spot,
			LastUpdated:  time.UnixMilli(1791535765477),
		},
		&ticker.Price{
			Last:              82561,
			High:              83143.2,
			Low:               80350.1,
			Bid:               82560.9,
			BidSize:           9.537,
			Ask:               82561,
			AskSize:           0.751,
			BaseVolume:        83950.362,
			QuoteVolume:       6866421328.479,
			OpenInterest:      28995.606,
			OpenInterestValue: 2393868532.68,
			MarkPrice:         82559.7,
			IndexPrice:        82590.92,
			Pair:              currency.NewBTCUSDT(),
			ExchangeName:      "TestWebsocketTicker",
			AssetType:         asset.USDTMarginedFutures,
			LastUpdated:       time.UnixMilli(1791535782383),
		},
		&ticker.Price{
			Last:              82560.9,
			High:              83143.2,
			Low:               80350.1,
			Bid:               82560.9,
			BidSize:           9.4,
			Ask:               82561,
			AskSize:           0.755,
			BaseVolume:        83950.363,
			QuoteVolume:       6866421411.0399,
			OpenInterest:      28995.606,
			OpenInterestValue: 2393868532.68,
			MarkPrice:         82559.7,
			IndexPrice:        82590.92,
			Pair:              currency.NewBTCUSDT(),
			ExchangeName:      "TestWebsocketTicker",
			AssetType:         asset.USDTMarginedFutures,
			LastUpdated:       time.UnixMilli(1791535782483),
		},
		&ticker.Price{
			Last:              82783.2,
			High:              83111.7,
			Low:               80593,
			Bid:               82680.4,
			BidSize:           0.484,
			Ask:               82705,
			AskSize:           0.484,
			BaseVolume:        1.644,
			QuoteVolume:       135063.2496,
			OpenInterest:      49.726,
			OpenInterestValue: 4112171.13,
			MarkPrice:         82696.6,
			IndexPrice:        82594.9,
			Pair:              currency.NewPairWithDelimiter("BTC", "USDT-16OCT26", ""),
			ExchangeName:      "TestWebsocketTicker",
			AssetType:         asset.USDTMarginedFutures,
			LastUpdated:       time.UnixMilli(1791535781868),
		},
		&ticker.Price{
			Bid:          0.0511,
			BidSize:      52000,
			Ask:          0.0513,
			AskSize:      61000,
			MarkPrice:    0.0512,
			Pair:         currency.NewPairWithDelimiter("NEW", "USDT", ""),
			ExchangeName: "TestWebsocketTicker",
			AssetType:    asset.USDTMarginedFutures,
			LastUpdated:  time.UnixMilli(1791535781950),
		},
		&ticker.Price{
			Last:         84120,
			High:         84305,
			Low:          83950,
			Bid:          83310,
			BidSize:      18.55,
			Ask:          85565,
			AskSize:      18.55,
			BaseVolume:   0.02,
			QuoteVolume:  1652.27,
			OpenInterest: 1.2,
			MarkPrice:    84134.38654221,
			IndexPrice:   82611.83292604,
			Pair:         currency.NewPairWithDelimiter("BTC", "24SEP27-170000-P-USDT", "-"),
			ExchangeName: "TestWebsocketTicker",
			AssetType:    asset.Options,
			LastUpdated:  time.UnixMilli(1791535794028),
		},
		&ticker.Price{
			Last:              82502.6,
			High:              83092.3,
			Low:               80302.9,
			Bid:               82508,
			BidSize:           119791,
			Ask:               82508.1,
			AskSize:           3500,
			BaseVolume:        4571.2059,
			QuoteVolume:       373935316,
			OpenInterest:      258266716,
			OpenInterestValue: 3130.19,
			MarkPrice:         82508.23,
			IndexPrice:        82536.06,
			Pair:              currency.NewBTCUSD(),
			ExchangeName:      "TestWebsocketTicker",
			AssetType:         asset.CoinMarginedFutures,
			LastUpdated:       time.UnixMilli(1791535789460),
		},
	}
	assert.Equal(t, exp, relayed(ex), "wsHandleData should store and relay each ticker, merging deltas into the snapshot")

	err := ex.wsHandleData(t.Context(), nil, asset.USDCMarginedFutures, []byte(`{"topic":"tickers.BTCPERP","type":"delta","data":{"symbol":"BTCPERP","bid1Price":"82499.90"},"cs":43446182173,"ts":1791535783174}`))
	require.ErrorIs(t, err, ticker.ErrTickerNotFound, "wsHandleData must error for a delta without a snapshot")
	err = ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"tickers.ZZZQUSDT","type":"snapshot","data":{"symbol":"ZZZQUSDT"},"ts":1791535783174}`))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "wsHandleData must error for an unknown pair")
	err = ex.wsHandleData(t.Context(), nil, asset.USDCMarginedFutures, []byte(`{"topic":"tickers.BTCPERP","type":"snapshot","data":{"symbol":"BTCPERP","bid1Price":"2","ask1Price":"1"},"cs":43446182173,"ts":1791535783174}`))
	require.Error(t, err, "wsHandleData must not relay a ticker the store rejects")
	assert.Empty(t, relayed(ex), "a rejected ticker should not be relayed")
}

func TestWebsocketProcessPublicTicker(t *testing.T) {
	t.Parallel()
	frames := fixtureFrames(t, "testdata/wsPublicTicker.json")
	require.GreaterOrEqual(t, len(frames), 3, "the ticker fixture must start with a spot snapshot and a linear snapshot and delta")
	ex := fixtureStreamInstance(t)
	for i, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.USDTMarginedFutures} {
		require.NoErrorf(t, ex.wsProcessPublicTicker(t.Context(), a, frames[i]), "wsProcessPublicTicker must not error for frame %d", i)
	}
	exp := []any{
		&ticker.Price{
			Last:         82590.6,
			High:         83174,
			Low:          80393.9,
			BaseVolume:   13646.082843,
			QuoteVolume:  1118450408.4968019,
			Pair:         currency.NewBTCUSDT(),
			ExchangeName: "TestWebsocketProcessPublicTicker",
			AssetType:    asset.Spot,
			LastUpdated:  time.UnixMilli(1791535765477),
		},
		&ticker.Price{
			Last:              82561,
			High:              83143.2,
			Low:               80350.1,
			Bid:               82560.9,
			BidSize:           9.537,
			Ask:               82561,
			AskSize:           0.751,
			BaseVolume:        83950.362,
			QuoteVolume:       6866421328.479,
			OpenInterest:      28995.606,
			OpenInterestValue: 2393868532.68,
			MarkPrice:         82559.7,
			IndexPrice:        82590.92,
			Pair:              currency.NewBTCUSDT(),
			ExchangeName:      "TestWebsocketProcessPublicTicker",
			AssetType:         asset.USDTMarginedFutures,
			LastUpdated:       time.UnixMilli(1791535782383),
		},
		// The delta holds only the fields that changed, so the rest stay as the snapshot set them
		&ticker.Price{
			Last:              82560.9,
			High:              83143.2,
			Low:               80350.1,
			Bid:               82560.9,
			BidSize:           9.4,
			Ask:               82561,
			AskSize:           0.755,
			BaseVolume:        83950.363,
			QuoteVolume:       6866421411.0399,
			OpenInterest:      28995.606,
			OpenInterestValue: 2393868532.68,
			MarkPrice:         82559.7,
			IndexPrice:        82590.92,
			Pair:              currency.NewBTCUSDT(),
			ExchangeName:      "TestWebsocketProcessPublicTicker",
			AssetType:         asset.USDTMarginedFutures,
			LastUpdated:       time.UnixMilli(1791535782483),
		},
	}
	assert.Equal(t, exp, relayed(ex), "wsProcessPublicTicker should relay each ticker, merging a delta into the snapshot")
	stored := make([]any, 0, 2)
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures} {
		tick, err := ticker.GetTicker(ex.Name, currency.NewBTCUSDT(), a)
		require.NoErrorf(t, err, "GetTicker must not error for %s", a)
		stored = append(stored, tick)
	}
	assert.Equal(t, []any{exp[0], exp[2]}, stored, "wsProcessPublicTicker should store each ticker it relays")

	// A delta can take a field to zero, and an empty one leaves the field as it was
	zeroed := []byte(`{"topic":"tickers.BTCUSDT","type":"delta","data":{"symbol":"BTCUSDT","singleOpenInterest":"0","singleOpenInterestValue":"0","markPrice":"","bid1Price":""},"cs":414513823617,"ts":1791535782583}`)
	require.NoError(t, ex.wsProcessPublicTicker(t.Context(), asset.USDTMarginedFutures, zeroed), "wsProcessPublicTicker must not error for a delta taking a field to zero")
	expZeroed, ok := exp[2].(*ticker.Price)
	require.True(t, ok, "the delta's ticker must be a price")
	zeroedTick := *expZeroed
	zeroedTick.OpenInterest, zeroedTick.OpenInterestValue, zeroedTick.LastUpdated = 0, 0, time.UnixMilli(1791535782583)
	assert.Equal(t, []any{&zeroedTick}, relayed(ex), "wsProcessPublicTicker should apply a delta's zeroes and ignore its empty fields")

	for _, tc := range []struct {
		name  string
		asset asset.Item
		frame []byte
		err   error
	}{
		{name: "delta without a snapshot", asset: asset.USDTMarginedFutures, frame: frames[2], err: ticker.ErrTickerNotFound},
		{name: "unknown pair", asset: asset.Spot, frame: bytes.ReplaceAll(frames[0], []byte("BTCUSDT"), []byte("ZZZQUSDT")), err: currency.ErrPairNotFound},
		{name: "ticker the store rejects", asset: asset.USDTMarginedFutures, frame: bytes.Replace(frames[1], []byte(`"ask1Price":"82561.00"`), []byte(`"ask1Price":"82560.90"`), 1), err: ticker.ErrBidEqualsAsk},
		{name: "undecodable push", asset: asset.Spot, frame: []byte(`{"topic":"tickers.BTCUSDT","type":"snapshot","data":"bad"}`), err: jsonparser.MalformedObjectError},
		{name: "undecodable field", asset: asset.Spot, frame: []byte(`{"topic":"tickers.BTCUSDT","type":"snapshot","data":{"symbol":"BTCUSDT","lastPrice":"bad"}}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := fixtureStreamInstance(t)
			err := ex.wsProcessPublicTicker(t.Context(), tc.asset, tc.frame)
			if tc.err == nil {
				require.Error(t, err, "wsProcessPublicTicker must error for a push that does not decode")
			} else {
				require.ErrorIs(t, err, tc.err, "wsProcessPublicTicker must return the expected error")
			}
			assert.Empty(t, relayed(ex), "wsProcessPublicTicker should not relay a ticker it could not process")
			_, err = ticker.GetTicker(ex.Name, currency.NewBTCUSDT(), tc.asset)
			assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "wsProcessPublicTicker should store no ticker it could not process")
		})
	}
}

func TestUpdateTickerDelta(t *testing.T) {
	t.Parallel()
	carried := func(fields ...string) map[string]bool {
		m := make(map[string]bool, len(fields))
		for _, f := range fields {
			m[f] = true
		}
		return m
	}
	tick := &ticker.Price{AssetType: asset.USDTMarginedFutures, Last: 1, Bid: 2, BidSize: 7, Ask: 3, MarkPrice: 4}
	updateTicker(tick, &WebsocketTicker{Bid1Price: 2.5, Volume24Hour: 10, Turnover24Hour: 20}, carried("bid1Price", "bid1Size", "volume24h", "turnover24h"))
	assert.Equal(t, &ticker.Price{AssetType: asset.USDTMarginedFutures, Last: 1, Bid: 2.5, Ask: 3, MarkPrice: 4, BaseVolume: 10, QuoteVolume: 20}, tick, "a delta should change only the fields it carries, to zero too")
	inverse := &ticker.Price{AssetType: asset.CoinMarginedFutures}
	updateTicker(inverse, &WebsocketTicker{Volume24Hour: 373935316, Turnover24Hour: 4571.2059}, carried("volume24h", "turnover24h"))
	assert.Equal(t, &ticker.Price{AssetType: asset.CoinMarginedFutures, BaseVolume: 4571.2059, QuoteVolume: 373935316}, inverse, "an inverse ticker's turnover should be its base volume")
	updateTicker(inverse, &WebsocketTicker{Turnover24Hour: 4572}, carried("turnover24h"))
	assert.Equal(t, &ticker.Price{AssetType: asset.CoinMarginedFutures, BaseVolume: 4572, QuoteVolume: 373935316}, inverse, "an inverse delta carrying only turnover should change only the base volume")
	option := &ticker.Price{AssetType: asset.Options}
	updateTicker(option, &WebsocketTicker{BidPrice: 83310, BidSize: 18.55, AskPrice: 85565, AskSize: 2.5}, carried("bidPrice", "bidSize", "askPrice", "askSize"))
	assert.Equal(t, &ticker.Price{AssetType: asset.Options, Bid: 83310, BidSize: 18.55, Ask: 85565, AskSize: 2.5}, option, "an option ticker's best bid and ask should be mapped")

	// Linear and inverse tickers count open interest on both sides of each contract and on one side, which GCT reports;
	// an option ticker's open interest, its only figure, counts one side
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures} {
		contract := &ticker.Price{AssetType: a}
		updateTicker(contract, &WebsocketTicker{OpenInterest: 57991.212, OpenInterestValue: 4787737065.36, SingleOpenInterest: 28995.606, SingleOpenInterestValue: 2393868532.68}, carried("openInterest", "openInterestValue", "singleOpenInterest", "singleOpenInterestValue"))
		assert.Equalf(t, &ticker.Price{AssetType: a, OpenInterest: 28995.606, OpenInterestValue: 2393868532.68}, contract, "a %s ticker should report its single side open interest", a)
		// A delta in which only the figure for both sides changed leaves the single side figure as it was
		updateTicker(contract, &WebsocketTicker{OpenInterest: 57991.213, OpenInterestValue: 4787737147.92}, carried("openInterest", "openInterestValue"))
		assert.Equalf(t, &ticker.Price{AssetType: a, OpenInterest: 28995.606, OpenInterestValue: 2393868532.68}, contract, "a %s ticker should not report open interest for both sides", a)
		// The last contract closing takes open interest to zero
		updateTicker(contract, &WebsocketTicker{}, carried("openInterest", "openInterestValue", "singleOpenInterest", "singleOpenInterestValue"))
		assert.Equalf(t, &ticker.Price{AssetType: a}, contract, "a %s ticker should report open interest falling to zero", a)
	}
	option = &ticker.Price{AssetType: asset.Options}
	updateTicker(option, &WebsocketTicker{OpenInterest: 1.2, OpenInterestValue: 99, SingleOpenInterest: 0.6}, carried("openInterest", "openInterestValue", "singleOpenInterest"))
	assert.Equal(t, &ticker.Price{AssetType: asset.Options, OpenInterest: 1.2, OpenInterestValue: 99}, option, "an option ticker's open interest should be mapped")

	// A delta can take any field to zero
	full := func(a asset.Item) *ticker.Price {
		return &ticker.Price{AssetType: a, Last: 1, High: 2, Low: 3, Bid: 4, BidSize: 5, Ask: 6, AskSize: 7, BaseVolume: 8, QuoteVolume: 9, MarkPrice: 10, IndexPrice: 11, OpenInterest: 12, OpenInterestValue: 13}
	}
	contract := full(asset.USDTMarginedFutures)
	updateTicker(contract, &WebsocketTicker{}, carried("lastPrice", "highPrice24h", "lowPrice24h", "bid1Price", "bid1Size", "ask1Price", "ask1Size", "volume24h", "turnover24h", "markPrice", "indexPrice", "singleOpenInterest", "singleOpenInterestValue"))
	assert.Equal(t, &ticker.Price{AssetType: asset.USDTMarginedFutures}, contract, "a delta should take every field it carries as zero to zero")
	option = full(asset.Options)
	updateTicker(option, &WebsocketTicker{}, carried("lastPrice", "highPrice24h", "lowPrice24h", "bidPrice", "bidSize", "askPrice", "askSize", "volume24h", "turnover24h", "markPrice", "indexPrice", "openInterest", "openInterestValue"))
	assert.Equal(t, &ticker.Price{AssetType: asset.Options}, option, "an option delta should take every field it carries as zero to zero")

	// A delta taking one field to zero leaves every other field as it was
	shared := map[string]func(*ticker.Price) *float64{
		"lastPrice":    func(p *ticker.Price) *float64 { return &p.Last },
		"highPrice24h": func(p *ticker.Price) *float64 { return &p.High },
		"lowPrice24h":  func(p *ticker.Price) *float64 { return &p.Low },
		"markPrice":    func(p *ticker.Price) *float64 { return &p.MarkPrice },
		"indexPrice":   func(p *ticker.Price) *float64 { return &p.IndexPrice },
	}
	with := func(fields map[string]func(*ticker.Price) *float64) map[string]func(*ticker.Price) *float64 {
		maps.Copy(fields, shared)
		return fields
	}
	for a, fields := range map[asset.Item]map[string]func(*ticker.Price) *float64{
		asset.USDTMarginedFutures: with(map[string]func(*ticker.Price) *float64{
			"bid1Price": func(p *ticker.Price) *float64 { return &p.Bid }, "bid1Size": func(p *ticker.Price) *float64 { return &p.BidSize },
			"ask1Price": func(p *ticker.Price) *float64 { return &p.Ask }, "ask1Size": func(p *ticker.Price) *float64 { return &p.AskSize },
			"volume24h": func(p *ticker.Price) *float64 { return &p.BaseVolume }, "turnover24h": func(p *ticker.Price) *float64 { return &p.QuoteVolume },
			"singleOpenInterest": func(p *ticker.Price) *float64 { return &p.OpenInterest }, "singleOpenInterestValue": func(p *ticker.Price) *float64 { return &p.OpenInterestValue },
		}),
		asset.CoinMarginedFutures: with(map[string]func(*ticker.Price) *float64{
			"bid1Price": func(p *ticker.Price) *float64 { return &p.Bid }, "bid1Size": func(p *ticker.Price) *float64 { return &p.BidSize },
			"ask1Price": func(p *ticker.Price) *float64 { return &p.Ask }, "ask1Size": func(p *ticker.Price) *float64 { return &p.AskSize },
			"volume24h": func(p *ticker.Price) *float64 { return &p.QuoteVolume }, "turnover24h": func(p *ticker.Price) *float64 { return &p.BaseVolume },
			"singleOpenInterest": func(p *ticker.Price) *float64 { return &p.OpenInterest }, "singleOpenInterestValue": func(p *ticker.Price) *float64 { return &p.OpenInterestValue },
		}),
		asset.Options: with(map[string]func(*ticker.Price) *float64{
			"bidPrice": func(p *ticker.Price) *float64 { return &p.Bid }, "bidSize": func(p *ticker.Price) *float64 { return &p.BidSize },
			"askPrice": func(p *ticker.Price) *float64 { return &p.Ask }, "askSize": func(p *ticker.Price) *float64 { return &p.AskSize },
			"volume24h": func(p *ticker.Price) *float64 { return &p.BaseVolume }, "turnover24h": func(p *ticker.Price) *float64 { return &p.QuoteVolume },
			"openInterest": func(p *ticker.Price) *float64 { return &p.OpenInterest }, "openInterestValue": func(p *ticker.Price) *float64 { return &p.OpenInterestValue },
		}),
	} {
		for field, value := range fields {
			got, exp := full(a), full(a)
			*value(exp) = 0
			updateTicker(got, &WebsocketTicker{}, carried(field))
			assert.Equalf(t, exp, got, "a %s delta taking %s to zero should change that field alone", a, field)
		}
	}
}

func TestTickerFields(t *testing.T) {
	t.Parallel()
	fields, err := tickerFields([]byte(`{"topic":"tickers.BTCUSDT","type":"delta","data":{"symbol":"BTCUSDT","bid1Price":"","bid1Size":"0","lastPrice":"95000.5","basis":"","nextFundingTime":"1792108800000","isPreListing":false},"ts":1792108700000}`))
	require.NoError(t, err, "tickerFields must not error")
	assert.Equal(t, map[string]bool{"symbol": true, "bid1Size": true, "lastPrice": true, "nextFundingTime": true, "isPreListing": true}, fields, "tickerFields should return every field carrying a value, zero included, and no empty one")
	_, err = tickerFields([]byte(`{"topic":"tickers.BTCUSDT","data":"bad"}`))
	assert.Error(t, err, "tickerFields should error for a push whose data is not an object")
}

func TestWebsocketKline(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	if !mockTests {
		frames, _ := liveStreamFrames(t, inversePublic, 20*time.Second, 1, "kline.1.BTCUSD")
		require.NotEmpty(t, frames, "the kline topic must push")
		require.NoError(t, ex.wsHandleData(t.Context(), nil, asset.CoinMarginedFutures, frames[0]), "wsHandleData must not error")
		data := relayed(ex)
		require.NotEmpty(t, data, "candles must be relayed")
		assert.IsType(t, kline.Item{}, data[0], "candles should be relayed as kline items")
		return
	}
	frames := fixtureFrames(t, "testdata/wsPublicKline.json")
	pushes := decodeFramesStrictly[WebsocketKlineUpdate](t, frames)
	expPushes := []WebsocketKlineUpdate{
		{
			Topic:     "kline.1.BTCUSDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791535765563)),
			Data: []WebsocketKline{
				{
					Start:     types.Time(time.Unix(1791535740, 0)),
					End:       types.Time(time.UnixMilli(1791535799999)),
					Interval:  "1",
					Open:      82585.8,
					Close:     82590.6,
					High:      82590.6,
					Low:       82585.8,
					Volume:    0.768965,
					Turnover:  63508.6685769,
					Timestamp: types.Time(time.UnixMilli(1791535765563)),
				},
			},
		},
		{
			Topic:     "kline.1.BTCUSD",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791535740021)),
			Data: []WebsocketKline{
				{
					Start:     types.Time(time.Unix(1791535680, 0)),
					End:       types.Time(time.UnixMilli(1791535739999)),
					Interval:  "1",
					Open:      82479.2,
					Close:     82494.5,
					High:      82501.3,
					Low:       82475.1,
					Volume:    184522,
					Turnover:  2.23688106,
					Confirm:   true,
					Timestamp: types.Time(time.UnixMilli(1791535739912)),
				},
			},
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every kline field")
	for i, a := range []asset.Item{asset.Spot, asset.CoinMarginedFutures} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, a, frames[i]), "wsHandleData must not error for frame %d", i)
	}
	exp := []any{
		kline.Item{
			Exchange: "TestWebsocketKline",
			Pair:     currency.NewBTCUSDT(),
			Asset:    asset.Spot,
			Interval: kline.OneMin,
			Candles: []kline.Candle{
				{
					Time:             time.Unix(1791535740, 0),
					Open:             82585.8,
					High:             82590.6,
					Low:              82585.8,
					Close:            82590.6,
					Volume:           0.768965,
					QuoteVolume:      63508.6685769,
					ValidationIssues: kline.PartialCandle,
				},
			},
		},
		kline.Item{
			Exchange: "TestWebsocketKline",
			Pair:     currency.NewBTCUSD(),
			Asset:    asset.CoinMarginedFutures,
			Interval: kline.OneMin,
			Candles: []kline.Candle{
				{
					Time:        time.Unix(1791535680, 0),
					Open:        82479.2,
					High:        82501.3,
					Low:         82475.1,
					Close:       82494.5,
					Volume:      2.23688106,
					QuoteVolume: 184522,
				},
			},
		},
	}
	assert.Equal(t, exp, relayed(ex), "wsHandleData should relay each candle")
	err := ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"kline.1.ZZZQUSDT","data":[],"ts":1672324988882,"type":"snapshot"}`))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "wsHandleData must error for an unknown pair")
	err = ex.wsHandleData(t.Context(), nil, asset.Spot, []byte(`{"topic":"kline.2.BTCUSDT","data":[{"start":1672324800000,"interval":"2"}],"ts":1672324988882,"type":"snapshot"}`))
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "wsHandleData must error for an unknown interval")
}

func TestWebsocketProcessKline(t *testing.T) {
	t.Parallel()
	frames := fixtureFrames(t, "testdata/wsPublicKline.json")
	require.Len(t, frames, 2, "the kline fixture must hold a spot and an inverse push")
	ex := fixtureStreamInstance(t)
	for i, a := range []asset.Item{asset.Spot, asset.CoinMarginedFutures} {
		require.NoErrorf(t, ex.wsProcessKline(t.Context(), a, frames[i]), "wsProcessKline must not error for frame %d", i)
	}
	exp := []any{
		kline.Item{
			Exchange: "TestWebsocketProcessKline",
			Pair:     currency.NewBTCUSDT(),
			Asset:    asset.Spot,
			Interval: kline.OneMin,
			Candles: []kline.Candle{
				{
					Time:             time.Unix(1791535740, 0),
					Open:             82585.8,
					High:             82590.6,
					Low:              82585.8,
					Close:            82590.6,
					Volume:           0.768965,
					QuoteVolume:      63508.6685769,
					ValidationIssues: kline.PartialCandle,
				},
			},
		},
		// An inverse candle's volume counts USD contracts and its turnover BTC, so the two swap
		kline.Item{
			Exchange: "TestWebsocketProcessKline",
			Pair:     currency.NewBTCUSD(),
			Asset:    asset.CoinMarginedFutures,
			Interval: kline.OneMin,
			Candles: []kline.Candle{
				{
					Time:        time.Unix(1791535680, 0),
					Open:        82479.2,
					High:        82501.3,
					Low:         82475.1,
					Close:       82494.5,
					Volume:      2.23688106,
					QuoteVolume: 184522,
				},
			},
		},
	}
	assert.Equal(t, exp, relayed(ex), "wsProcessKline should relay each candle with the pair its topic names")

	// Whether a candle is partial follows its confirm flag, whichever asset it is of
	for i, a := range []asset.Item{asset.Spot, asset.CoinMarginedFutures} {
		for _, confirmed := range []bool{false, true} {
			frame := bytes.Replace(frames[i], []byte(`"confirm":`+strconv.FormatBool(i == 1)), []byte(`"confirm":`+strconv.FormatBool(confirmed)), 1)
			require.NoErrorf(t, ex.wsProcessKline(t.Context(), a, frame), "wsProcessKline must not error for a %s candle confirmed %t", a, confirmed)
			got := relayed(ex)
			require.Lenf(t, got, 1, "wsProcessKline must relay a %s candle confirmed %t", a, confirmed)
			item, ok := got[0].(kline.Item)
			require.Truef(t, ok, "wsProcessKline must relay a %s candle confirmed %t as a kline item", a, confirmed)
			require.Lenf(t, item.Candles, 1, "wsProcessKline must relay the %s candle confirmed %t", a, confirmed)
			expIssues := kline.PartialCandle
			if confirmed {
				expIssues = ""
			}
			assert.Equalf(t, expIssues, item.Candles[0].ValidationIssues, "wsProcessKline should mark a %s candle confirmed %t partial only while it is open", a, confirmed)
		}
	}

	for _, tc := range []struct {
		name  string
		frame []byte
		err   error
	}{
		{name: "a topic without a symbol", frame: []byte(`{"topic":"kline.1","data":[],"ts":1791535765563,"type":"snapshot"}`), err: errMalformedTopic},
		{name: "an unknown pair", frame: bytes.ReplaceAll(frames[0], []byte("BTCUSDT"), []byte("ZZZQUSDT")), err: currency.ErrPairNotFound},
		{name: "an unknown interval", frame: bytes.Replace(frames[0], []byte(`"interval":"1"`), []byte(`"interval":"2"`), 1), err: kline.ErrInvalidInterval},
		{name: "an undecodable push", frame: []byte(`{"topic":"kline.1.BTCUSDT","data":"bad"}`)},
	} {
		err := ex.wsProcessKline(t.Context(), asset.Spot, tc.frame)
		if tc.err == nil {
			require.Errorf(t, err, "wsProcessKline must error for %s", tc.name)
		} else {
			require.ErrorIsf(t, err, tc.err, "wsProcessKline must error for %s", tc.name)
		}
		assert.Emptyf(t, relayed(ex), "wsProcessKline should relay nothing for %s", tc.name)
	}

	errBufferFull := fillDataHandler(t, ex)
	require.ErrorIs(t, ex.wsProcessKline(t.Context(), asset.Spot, frames[0]), errBufferFull, "wsProcessKline must return the data handler's error")
}

func TestWebsocketAllLiquidation(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	if !mockTests {
		topics := []string{"allLiquidation.BTCUSDT", "allLiquidation.ETHUSDT", "allLiquidation.SOLUSDT", "allLiquidation.XRPUSDT", "allLiquidation.DOGEUSDT"}
		frames, reply := liveStreamFrames(t, linearPublic, 15*time.Second, 1, topics...)
		var resp SubscriptionResponse
		require.NoError(t, json.Unmarshal(reply, &resp), "subscription reply must decode")
		require.True(t, resp.Success, "liquidation topics must be accepted")
		// Liquidations are rare, so a quiet window proves only the subscription
		for _, frame := range frames {
			require.NoError(t, ex.wsHandleData(t.Context(), nil, asset.USDTMarginedFutures, frame), "wsHandleData must not error")
		}
		return
	}
	testRelayedPublicStream[WebsocketAllLiquidationUpdate](t, ex, "testdata/wsPublicAllLiquidation.json", asset.USDTMarginedFutures, []WebsocketAllLiquidationUpdate{
		{
			Topic:     "allLiquidation.BTCUSDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.UnixMilli(1791534260980)),
			Data: []WebsocketLiquidation{
				{
					UpdatedTime:     types.Time(time.UnixMilli(1791534260544)),
					Symbol:          "BTCUSDT",
					Side:            "Buy",
					Size:            0.041,
					BankruptcyPrice: 82361.4,
				},
			},
		},
	})
}

func TestWebsocketInsurancePool(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	if !mockTests {
		frames, _ := liveStreamFrames(t, linearPublic, 20*time.Second, 1, "insurance.USDT")
		require.NotEmpty(t, frames, "the insurance topic must push its snapshot")
		require.NoError(t, ex.wsHandleData(t.Context(), nil, asset.USDTMarginedFutures, frames[0]), "wsHandleData must not error")
		return
	}
	testRelayedPublicStream[WebsocketInsurancePoolUpdate](t, ex, "testdata/wsPublicInsurancePool.json", asset.USDTMarginedFutures, []WebsocketInsurancePoolUpdate{
		{
			Topic:     "insurance.USDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.Unix(1791535781, 0)),
			Data: []WebsocketInsurancePool{
				{
					Coin:       currency.USDT,
					Symbols:    "HYPEUSDT-23OCT26,HYPEUSDT-16OCT26,HYPEUSDT-27NOV26,HYPEUSDT-30OCT26",
					Balance:    50044.12025128,
					UpdateTime: types.Time(time.Unix(1791535781, 0)),
				},
				{
					Coin:       currency.USDT,
					Symbols:    "PRLUSDT,BASEDUSDT,GWEIUSDT",
					Balance:    2170104.46560389,
					UpdateTime: types.Time(time.Unix(1791535781, 0)),
				},
			},
		},
		{
			Topic:     "insurance.USDT",
			Type:      "delta",
			Timestamp: types.Time(time.Unix(1791535782, 0)),
			Data: []WebsocketInsurancePool{
				{
					Coin:       currency.USDT,
					Symbols:    "PRLUSDT,BASEDUSDT,GWEIUSDT",
					Balance:    2170101.02210391,
					UpdateTime: types.Time(time.Unix(1791535782, 0)),
				},
			},
		},
	})
}

func TestWebsocketPriceLimit(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	if !mockTests {
		frames, _ := liveStreamFrames(t, spotPublic, 20*time.Second, 1, "priceLimit.BTCUSDT")
		require.NotEmpty(t, frames, "the price limit topic must push")
		require.NoError(t, ex.wsHandleData(t.Context(), nil, asset.Spot, frames[0]), "wsHandleData must not error")
		return
	}
	testRelayedPublicStream[WebsocketPriceLimitUpdate](t, ex, "testdata/wsPublicPriceLimit.json", asset.Spot, []WebsocketPriceLimitUpdate{
		{
			Topic:     "priceLimit.BTCUSDT",
			Timestamp: types.Time(time.UnixMilli(1791535781983)),
			Data: WebsocketPriceLimit{
				Symbol:    "BTCUSDT",
				BuyLimit:  83385.2,
				SellLimit: 81734.1,
			},
		},
	})
}

func TestWebsocketADLAlert(t *testing.T) {
	t.Parallel()
	ex := streamTestInstance(t)
	if !mockTests {
		frames, _ := liveStreamFrames(t, inversePublic, 20*time.Second, 1, "adlAlert.inverse")
		require.NotEmpty(t, frames, "the ADL alert topic must push its snapshot")
		require.NoError(t, ex.wsHandleData(t.Context(), nil, asset.CoinMarginedFutures, frames[0]), "wsHandleData must not error")
		return
	}
	testRelayedPublicStream[WebsocketADLAlertUpdate](t, ex, "testdata/wsPublicADLAlert.json", asset.USDTMarginedFutures, []WebsocketADLAlertUpdate{
		{
			Topic:     "adlAlert.USDT",
			Type:      "snapshot",
			Timestamp: types.Time(time.Unix(1791535778, 0)),
			Data: []WebsocketADLAlert{
				{
					Coin:                currency.USDT,
					Symbol:              "DGAIUSDT",
					Balance:             17651927.40423898,
					InsurancePNLRatio:   -1,
					ADLTriggerThreshold: 20000,
					ADLStopRatio:        -0.95,
				},
				{
					Coin:                currency.USDT,
					Symbol:              "FWOGUSDT",
					Balance:             -5421.29889888,
					MaxBalance:          -5421.29889888,
					InsurancePNLRatio:   -0.3,
					PNLRatio:            0.000174,
					ADLTriggerThreshold: 10000,
					ADLStopRatio:        -0.25,
				},
			},
		},
	})
}

// testRelayedPublicStream feeds a fixture's frames to the public handler, checks they decode to exp and that each
// push is relayed decoded
func testRelayedPublicStream[T any](t *testing.T, ex *Exchange, fixture string, a asset.Item, exp []T) {
	t.Helper()
	frames := fixtureFrames(t, fixture)
	pushes := decodeFramesStrictly[T](t, frames)
	assert.Equal(t, exp, pushes, "fixture frames should decode every field")
	for i, frame := range frames {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), nil, a, frame), "wsHandleData must not error for frame %d", i)
	}
	data := relayed(ex)
	require.Len(t, data, len(frames), "every push must be relayed")
	for i := range pushes {
		assert.Equalf(t, &pushes[i], data[i], "push %d should be relayed decoded", i)
	}
}

func TestPublicTopic(t *testing.T) {
	t.Parallel()
	btc := currency.Pairs{currency.NewBTCUSDT()}
	for _, tc := range []struct {
		sub   *subscription.Subscription
		asset asset.Item
		exp   string
		err   error
	}{
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Pairs: btc, Levels: 200}, asset: asset.USDTMarginedFutures, exp: "orderbook.200.BTCUSDT"},
		{sub: &subscription.Subscription{Channel: chanOrderbook, Pairs: btc}, asset: asset.USDTMarginedFutures, exp: "orderbook.50.BTCUSDT"},
		{sub: &subscription.Subscription{Channel: chanOrderbook, Pairs: currency.Pairs{fixtureOptionPair}}, asset: asset.Options, exp: "orderbook.25.BTC-24SEP27-170000-P-USDT"},
		{sub: &subscription.Subscription{Channel: chanRPIOrderbook, Pairs: btc}, asset: asset.USDTMarginedFutures, exp: "orderbook.rpi.BTCUSDT"},
		{sub: &subscription.Subscription{Channel: chanFullOrderbook, Pairs: currency.Pairs{currency.NewBTCUSD()}}, asset: asset.CoinMarginedFutures, exp: "orderbook.full.BTCUSD"},
		{sub: &subscription.Subscription{Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{fixtureUSDCPerpetualPair}}, asset: asset.USDCMarginedFutures, exp: "publicTrade.BTCPERP"},
		{sub: &subscription.Subscription{Channel: chanPublicTrade, Pairs: currency.Pairs{fixtureOptionPair, currency.NewPairWithDelimiter("BTC", "10OCT26-82750-C-USDT", "-")}}, asset: asset.Options, exp: "publicTrade.BTC"},
		{sub: &subscription.Subscription{Channel: subscription.TickerChannel, Pairs: currency.Pairs{fixtureUSDTFuturesPair}}, asset: asset.USDTMarginedFutures, exp: "tickers.BTCUSDT-16OCT26"},
		{sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Pairs: btc, Interval: kline.FifteenMin}, asset: asset.USDTMarginedFutures, exp: "kline.15.BTCUSDT"},
		{sub: &subscription.Subscription{Channel: chanAllLiquidation, Pairs: btc}, asset: asset.USDTMarginedFutures, exp: "allLiquidation.BTCUSDT"},
		{sub: &subscription.Subscription{Channel: chanPriceLimit, Pairs: currency.Pairs{currency.NewBTCUSD()}}, asset: asset.CoinMarginedFutures, exp: "priceLimit.BTCUSD"},
		{sub: &subscription.Subscription{Channel: chanInsurancePool}, asset: asset.USDTMarginedFutures, exp: "insurance.USDT"},
		{sub: &subscription.Subscription{Channel: chanInsurancePool}, asset: asset.USDCMarginedFutures, exp: "insurance.USDC"},
		{sub: &subscription.Subscription{Channel: chanADLAlert}, asset: asset.CoinMarginedFutures, exp: "adlAlert.inverse"},
		{sub: &subscription.Subscription{Channel: chanADLAlert}, asset: asset.Options, err: errContractGroupNotSupported},
		{sub: &subscription.Subscription{Channel: chanPublicTrade}, asset: asset.Options, err: currency.ErrCurrencyPairsEmpty},
		{sub: &subscription.Subscription{Channel: chanPublicTrade, Pairs: currency.Pairs{fixtureOptionPair, currency.NewPairWithDelimiter("ETH", "10OCT26-2500-C-USDT", "-")}}, asset: asset.Options, err: errOptionTradesNotOneBaseCoin},
		{sub: &subscription.Subscription{Channel: chanPublicTicker, Pairs: currency.Pairs{currency.NewBTCUSDT(), currency.NewBTCUSD()}}, asset: asset.USDTMarginedFutures, err: subscription.ErrNotSinglePair},
		{sub: &subscription.Subscription{Channel: chanKline, Pairs: btc}, asset: asset.USDTMarginedFutures, err: subscription.ErrInvalidInterval},
		{sub: &subscription.Subscription{Channel: chanPublicTicker, Pairs: btc}, asset: asset.Binary, err: asset.ErrNotSupported},
	} {
		topic, err := e.publicTopic(tc.sub, tc.asset)
		require.ErrorIsf(t, err, tc.err, "publicTopic must return the expected error for %s %s", tc.asset, tc.sub.Channel)
		assert.Equalf(t, tc.exp, topic, "publicTopic should return the topic for %s %s", tc.asset, tc.sub.Channel)
	}
}

func TestOrderbookLevels(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		levels int
		asset  asset.Item
		exp    int
	}{
		{levels: 200, asset: asset.Spot, exp: 200},
		{levels: 100, asset: asset.Options, exp: 100},
		{asset: asset.Spot, exp: defaultOrderbookLevels},
		{asset: asset.USDTMarginedFutures, exp: defaultOrderbookLevels},
		{asset: asset.CoinMarginedFutures, exp: defaultOrderbookLevels},
		{asset: asset.Options, exp: defaultOptionOrderbookLevels},
	} {
		got := orderbookLevels(&subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: tc.levels}, tc.asset)
		assert.Equalf(t, tc.exp, got, "orderbookLevels should return the depth for %s with levels %d", tc.asset, tc.levels)
	}
}

func TestContractGroup(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset asset.Item
		exp   string
		err   error
	}{
		{asset: asset.USDTMarginedFutures, exp: "USDT"},
		{asset: asset.USDCMarginedFutures, exp: "USDC"},
		{asset: asset.CoinMarginedFutures, exp: "inverse"},
		{asset: asset.Spot, err: errContractGroupNotSupported},
		{asset: asset.Options, err: errContractGroupNotSupported},
	} {
		group, err := contractGroup(tc.asset)
		require.ErrorIsf(t, err, tc.err, "contractGroup must return the expected error for %s", tc.asset)
		assert.Equalf(t, tc.exp, group, "contractGroup should name the %s group as the insurance and ADL alert topics do", tc.asset)
	}
}

func TestOptionTradeTopic(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		pairs currency.Pairs
		exp   string
		err   error
	}{
		{name: "one contract", pairs: currency.Pairs{fixtureOptionPair}, exp: "publicTrade.BTC"},
		{name: "contracts of one base coin in either case", pairs: currency.Pairs{currency.NewPairWithDelimiter("btc", "10oct26-82750-c-usdt", currency.DashDelimiter), fixtureOptionPair}, exp: "publicTrade.BTC"},
		{name: "no contracts", err: currency.ErrCurrencyPairsEmpty},
		{name: "contracts of two base coins", pairs: currency.Pairs{fixtureOptionPair, currency.NewPairWithDelimiter("ETH", "10OCT26-2500-C-USDT", currency.DashDelimiter)}, err: errOptionTradesNotOneBaseCoin},
	} {
		topic, err := optionTradeTopic(tc.pairs)
		require.ErrorIsf(t, err, tc.err, "optionTradeTopic must return the expected error for %s", tc.name)
		assert.Equalf(t, tc.exp, topic, "optionTradeTopic should return the base coin's trade topic for %s", tc.name)
	}
}

func TestGeneratePublicDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	options := currency.Pairs{fixtureOptionPair, currency.NewPairWithDelimiter("BTC", "10OCT26-82750-C-USDT", "-"), currency.NewPairWithDelimiter("ETH", "10OCT26-2500-C-USDT", "-")}
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, options, false), "StorePairs must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, options, true), "StorePairs must not error")
	subs, err := ex.GenerateOptionsDefaultSubscriptions()
	require.NoError(t, err, "GenerateOptionsDefaultSubscriptions must not error")
	assert.ElementsMatch(t, []string{
		"orderbook.25.BTC-24SEP27-170000-P-USDT", "tickers.BTC-24SEP27-170000-P-USDT",
		"orderbook.25.BTC-10OCT26-82750-C-USDT", "tickers.BTC-10OCT26-82750-C-USDT",
		"orderbook.25.ETH-10OCT26-2500-C-USDT", "tickers.ETH-10OCT26-2500-C-USDT",
		"publicTrade.BTC", "publicTrade.ETH",
	}, subs.QualifiedChannels(), "option subscriptions should take 25 levels and one trade topic per base coin")
	for _, s := range subs {
		if s.QualifiedChannel == "publicTrade.BTC" {
			assert.Len(t, s.Pairs, 2, "the BTC trade subscription should carry both BTC contracts")
		}
	}

	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, currency.Pairs{currency.NewBTCUSD()}, false), "StorePairs must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, currency.Pairs{currency.NewBTCUSD()}, true), "StorePairs must not error")
	subs, err = ex.GenerateInverseDefaultSubscriptions()
	require.NoError(t, err, "GenerateInverseDefaultSubscriptions must not error")
	assert.ElementsMatch(t, []string{"orderbook.50.BTCUSD", "tickers.BTCUSD", "publicTrade.BTCUSD"}, subs.QualifiedChannels(), "inverse subscriptions should cover the orderbook, ticker and trades")

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.CoinMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = ex.GenerateInverseDefaultSubscriptions()
	require.NoError(t, err, "GenerateInverseDefaultSubscriptions must not error for a disabled asset")
	assert.Empty(t, subs, "a disabled asset should generate no subscriptions")

	ex = websocketTestInstance(t)
	linear := currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, linear, false), "StorePairs must not error for the available linear pairs")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, linear, true), "StorePairs must not error for the enabled linear pairs")
	subs, err = ex.generatePublicDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err, "generatePublicDefaultSubscriptions must not error")
	// Enabled pairs come in the asset's configured format
	btc, eth := currency.NewPairWithDelimiter("BTC", "USDT", "_"), currency.NewPairWithDelimiter("ETH", "USDT", "_")
	exp := subscription.List{
		{Enabled: true, Channel: chanOrderbook, Pairs: currency.Pairs{btc}, Asset: asset.USDTMarginedFutures, Levels: 50, QualifiedChannel: "orderbook.50.BTCUSDT"},
		{Enabled: true, Channel: chanPublicTicker, Pairs: currency.Pairs{btc}, Asset: asset.USDTMarginedFutures, QualifiedChannel: "tickers.BTCUSDT"},
		{Enabled: true, Channel: chanPublicTrade, Pairs: currency.Pairs{btc}, Asset: asset.USDTMarginedFutures, QualifiedChannel: "publicTrade.BTCUSDT"},
		{Enabled: true, Channel: chanOrderbook, Pairs: currency.Pairs{eth}, Asset: asset.USDTMarginedFutures, Levels: 50, QualifiedChannel: "orderbook.50.ETHUSDT"},
		{Enabled: true, Channel: chanPublicTicker, Pairs: currency.Pairs{eth}, Asset: asset.USDTMarginedFutures, QualifiedChannel: "tickers.ETHUSDT"},
		{Enabled: true, Channel: chanPublicTrade, Pairs: currency.Pairs{eth}, Asset: asset.USDTMarginedFutures, QualifiedChannel: "publicTrade.ETHUSDT"},
	}
	assert.Equal(t, exp, subs, "generatePublicDefaultSubscriptions should subscribe each enabled pair's 50 level orderbook, ticker and trades")

	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, options, false), "StorePairs must not error for the available option pairs")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Options, options, true), "StorePairs must not error for the enabled option pairs")
	subs, err = ex.generatePublicDefaultSubscriptions(asset.Options)
	require.NoError(t, err, "generatePublicDefaultSubscriptions must not error for options")
	exp = subscription.List{
		{Enabled: true, Channel: chanOrderbook, Pairs: currency.Pairs{options[0]}, Asset: asset.Options, Levels: 25, QualifiedChannel: "orderbook.25.BTC-24SEP27-170000-P-USDT"},
		{Enabled: true, Channel: chanPublicTicker, Pairs: currency.Pairs{options[0]}, Asset: asset.Options, QualifiedChannel: "tickers.BTC-24SEP27-170000-P-USDT"},
		{Enabled: true, Channel: chanOrderbook, Pairs: currency.Pairs{options[1]}, Asset: asset.Options, Levels: 25, QualifiedChannel: "orderbook.25.BTC-10OCT26-82750-C-USDT"},
		{Enabled: true, Channel: chanPublicTicker, Pairs: currency.Pairs{options[1]}, Asset: asset.Options, QualifiedChannel: "tickers.BTC-10OCT26-82750-C-USDT"},
		{Enabled: true, Channel: chanOrderbook, Pairs: currency.Pairs{options[2]}, Asset: asset.Options, Levels: 25, QualifiedChannel: "orderbook.25.ETH-10OCT26-2500-C-USDT"},
		{Enabled: true, Channel: chanPublicTicker, Pairs: currency.Pairs{options[2]}, Asset: asset.Options, QualifiedChannel: "tickers.ETH-10OCT26-2500-C-USDT"},
		{Enabled: true, Channel: chanPublicTrade, Pairs: currency.Pairs{options[0], options[1]}, Asset: asset.Options, QualifiedChannel: "publicTrade.BTC"},
		{Enabled: true, Channel: chanPublicTrade, Pairs: currency.Pairs{options[2]}, Asset: asset.Options, QualifiedChannel: "publicTrade.ETH"},
	}
	assert.Equal(t, exp, subs, "generatePublicDefaultSubscriptions should subscribe each option's 25 level orderbook and ticker, and each base coin's trades")

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = ex.generatePublicDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err, "generatePublicDefaultSubscriptions must not error for a disabled asset")
	assert.Nil(t, subs, "generatePublicDefaultSubscriptions should generate nothing for a disabled asset")

	subs, err = ex.generatePublicDefaultSubscriptions(asset.Binary)
	require.ErrorIs(t, err, currency.ErrAssetNotFound, "generatePublicDefaultSubscriptions must error for an asset without pairs")
	assert.Nil(t, subs, "generatePublicDefaultSubscriptions should generate nothing for an asset without pairs")

	usdc := &currency.PairStore{AssetEnabled: true, Available: currency.Pairs{fixtureUSDCPerpetualPair}, Enabled: currency.Pairs{fixtureUSDCPerpetualPair}, ConfigFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}}
	require.NoError(t, ex.CurrencyPairs.Store(asset.USDCMarginedFutures, usdc), "Store must not error for a pair store without a request format")
	subs, err = ex.generatePublicDefaultSubscriptions(asset.USDCMarginedFutures)
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "generatePublicDefaultSubscriptions must report subscriptions it cannot name a topic for")
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "generatePublicDefaultSubscriptions must return why it cannot name the topics")
	assert.Nil(t, subs, "generatePublicDefaultSubscriptions should generate nothing when it cannot name every topic")
}

func TestQualifyPublicSubscriptions(t *testing.T) {
	t.Parallel()
	good := &subscription.Subscription{Channel: chanPublicTicker, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
	done := &subscription.Subscription{Channel: chanPublicTicker, QualifiedChannel: "tickers.ETHUSDT"}
	bad := &subscription.Subscription{Channel: chanKline, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
	valid, err := e.qualifyPublicSubscriptions(asset.USDTMarginedFutures, subscription.List{good, done, bad})
	require.ErrorIs(t, err, subscription.ErrInvalidInterval, "qualifyPublicSubscriptions must report a subscription without a topic")
	require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "qualifyPublicSubscriptions must report a subscription failure")
	assert.Equal(t, subscription.List{good, done}, valid, "qualifyPublicSubscriptions should keep the subscriptions with a topic")
	assert.Equal(t, "tickers.BTCUSDT", good.QualifiedChannel, "qualifyPublicSubscriptions should set the topic")
	assert.Equal(t, "tickers.ETHUSDT", done.QualifiedChannel, "qualifyPublicSubscriptions should keep a set topic")
}
