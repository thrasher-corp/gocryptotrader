package lbank

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/kline"
	"github.com/thrasher-corp/gocryptotrader/exchange/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPing(t *testing.T) {
	t.Parallel()
	assert.NoError(t, e.Ping(t.Context()), "Ping should not error")
}

func TestGetMarketDepth(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	ctx := t.Context()
	for _, size := range []uint64{0, maxDepthSize + 1} {
		t.Run("size "+strconv.FormatUint(size, 10), func(t *testing.T) {
			_, err := e.GetMarketDepth(ctx, spotTestPair, size)
			require.ErrorIsf(t, err, errLimitOutOfRange, "GetMarketDepth must reject a size of %d", size)
		})
	}
	t.Run("no pair", func(t *testing.T) {
		_, err := e.GetMarketDepth(ctx, currency.EMPTYPAIR, 5)
		require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarketDepth must require a pair")
	})
	t.Run("order book", func(t *testing.T) {
		result, err := e.GetMarketDepth(ctx, spotTestPair, 5)
		require.NoError(t, err, "GetMarketDepth must not error")
		if mockTests {
			exp := &MarketDepthResponse{
				Asks: orderbook.LevelsArrayPriceAmount{
					{Price: 80796.79, Amount: 8.72803},
					{Price: 80796.8, Amount: 0.00046},
					{Price: 80796.86, Amount: 0.00249},
					{Price: 80796.87, Amount: 0.00514},
					{Price: 80797.08, Amount: 0.00017},
				},
				Bids: orderbook.LevelsArrayPriceAmount{
					{Price: 80796.78, Amount: 2.69408},
					{Price: 80796.77, Amount: 0.00061},
					{Price: 80796.52, Amount: 0.00008},
					{Price: 80795.81, Amount: 0.00031},
					{Price: 80795.62, Amount: 0.00372},
				},
				Timestamp: types.Time(time.UnixMilli(1792890784854)),
			}
			assert.Equal(t, exp, result, "GetMarketDepth should decode the order book")
			return
		}
		assert.Len(t, result.Asks, 5, "GetMarketDepth should return the depth asked for")
	})
}

func TestGetLatestPrices(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	ctx := t.Context()
	t.Run("one pair", func(t *testing.T) {
		result, err := e.GetLatestPrices(ctx, spotTestPair)
		require.NoError(t, err, "GetLatestPrices must not error")
		if mockTests {
			exp := []LatestPriceResponse{{Symbol: "btc_usdt", Price: 80796.78}}
			assert.Equal(t, exp, result, "GetLatestPrices should decode the pair's price")
		} else {
			require.Len(t, result, 1, "GetLatestPrices must return the pair asked for")
		}
	})
	t.Run("every pair", func(t *testing.T) {
		result, err := e.GetLatestPrices(ctx, currency.EMPTYPAIR)
		require.NoError(t, err, "GetLatestPrices must not error for every pair")
		if mockTests {
			exp := []LatestPriceResponse{{Symbol: "oph_usdt", Price: 0.000049}, {Symbol: "marschain_usdt", Price: 0.010097}, {Symbol: "esp3l_usdt", Price: 0.698001}}
			assert.Equal(t, exp, result, "GetLatestPrices should decode every pair's price")
			return
		}
		assert.Greater(t, len(result), 1, "GetLatestPrices should return every pair")
	})
}

func TestGetBookTicker(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	ctx := t.Context()
	t.Run("no pair", func(t *testing.T) {
		_, err := e.GetBookTicker(ctx, currency.EMPTYPAIR)
		require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetBookTicker must require a pair")
	})
	t.Run("best bid and ask", func(t *testing.T) {
		result, err := e.GetBookTicker(ctx, spotTestPair)
		require.NoError(t, err, "GetBookTicker must not error")
		if mockTests {
			exp := &BookTickerResponse{Symbol: "btc_usdt", BidPrice: 80796.78, BidQuantity: 2.69408, AskPrice: 80796.79, AskQuantity: 8.72803}
			assert.Equal(t, exp, result, "GetBookTicker should decode the best bid and ask")
			return
		}
		assert.Positive(t, result.AskPrice.Float64(), "GetBookTicker should return an ask")
	})
}

func TestGetTickers(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	ctx := t.Context()
	t.Run("one pair", func(t *testing.T) {
		result, err := e.GetTickers(ctx, spotTestPair)
		require.NoError(t, err, "GetTickers must not error")
		if mockTests {
			exp := []TickerResponse{{
				Symbol: "btc_usdt",
				Ticker: TickerStatistics{
					LastPrice: 80796.78, High24Hour: 81092.52, Low24Hour: 79270.99, Volume24Hour: 2863.0685, Turnover24Hour: 230249030.18,
					ChangePercentage24Hour: 0.04,
				},
				Timestamp: types.Time(time.UnixMilli(1792890786820)),
			}}
			assert.Equal(t, exp, result, "GetTickers should decode the pair's ticker")
		} else {
			require.Len(t, result, 1, "GetTickers must return the pair asked for")
		}
	})
	t.Run("every pair", func(t *testing.T) {
		result, err := e.GetTickers(ctx, currency.EMPTYPAIR)
		require.NoError(t, err, "GetTickers must not error for every pair")
		if mockTests {
			require.Len(t, result, 2, "GetTickers must return every recorded ticker")
			exp := TickerResponse{
				Symbol: "eth_usdt",
				Ticker: TickerStatistics{
					LastPrice: 2357.01, High24Hour: 2389.71, Low24Hour: 2283.18, Volume24Hour: 40538.4801, Turnover24Hour: 94806827.83,
					ChangePercentage24Hour: -0.89,
				},
				Timestamp: types.Time(time.UnixMilli(1792890784849)),
			}
			assert.Equal(t, exp, result[0], "GetTickers should decode every ticker")
			return
		}
		assert.Greater(t, len(result), 1, "GetTickers should return every pair")
	})
}

func TestGetLeveragedTokenTickers(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	ctx := t.Context()
	exp := TickerResponse{
		Symbol: "btc3l_usdt",
		Ticker: TickerStatistics{
			LastPrice: 2.678543, High24Hour: 2.721474, Low24Hour: 2.536929, Volume24Hour: 112529.32, Turnover24Hour: 300017.7961,
			ChangePercentage24Hour: 0.03,
		},
		Timestamp: types.Time(time.UnixMilli(1792890787880)),
	}
	t.Run("one token", func(t *testing.T) {
		pair := currency.NewPairWithDelimiter("btc3l", "usdt", currency.UnderscoreDelimiter)
		result, err := e.GetLeveragedTokenTickers(ctx, pair)
		require.NoError(t, err, "GetLeveragedTokenTickers must not error")
		if mockTests {
			assert.Equal(t, []TickerResponse{exp}, result, "GetLeveragedTokenTickers should decode the token's ticker")
		} else {
			require.Len(t, result, 1, "GetLeveragedTokenTickers must return the token asked for")
		}
	})
	t.Run("every token", func(t *testing.T) {
		result, err := e.GetLeveragedTokenTickers(ctx, currency.EMPTYPAIR)
		require.NoError(t, err, "GetLeveragedTokenTickers must not error for every token")
		if mockTests {
			require.Len(t, result, 2, "GetLeveragedTokenTickers must return every recorded ticker")
			assert.Equal(t, exp, result[1], "GetLeveragedTokenTickers should decode every ticker")
			return
		}
		assert.Greater(t, len(result), 1, "GetLeveragedTokenTickers should return every token")
	})
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	t.Run("after time sent in milliseconds", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v2/supplement/trades.do", r.URL.Path, "GetTrades should request the trades endpoint")
			assert.Equal(t, url.Values{"symbol": {"btc_usdt"}, "size": {"2"}, "time": {"1792884094962"}}, r.URL.Query(), "GetTrades should send the time in milliseconds")
			writeResponse(t, w, `{"result":"true","data":[],"error_code":0}`)
		}))
		result, err := ex.GetTrades(t.Context(), spotTestPair, 2, time.UnixMilli(1792884094962))
		require.NoError(t, err, "GetTrades must not error")
		assert.Empty(t, result, "GetTrades should return no trades")
	})

	// The cases below share the package exchange, so they run serially and in order
	ctx := t.Context()
	for _, size := range []uint64{0, maxTradesSize + 1} {
		t.Run("size "+strconv.FormatUint(size, 10), func(t *testing.T) {
			_, err := e.GetTrades(ctx, spotTestPair, size, time.Time{})
			require.ErrorIsf(t, err, errLimitOutOfRange, "GetTrades must reject a size of %d", size)
		})
	}
	t.Run("no pair", func(t *testing.T) {
		_, err := e.GetTrades(ctx, currency.EMPTYPAIR, 3, time.Time{})
		require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTrades must require a pair")
	})
	t.Run("latest trades", func(t *testing.T) {
		result, err := e.GetTrades(ctx, spotTestPair, 3, time.Time{})
		require.NoError(t, err, "GetTrades must not error")
		if mockTests {
			exp := []TradeResponse{
				{ID: "4f9ff15a834f4fec8b9e77393e01b9c0", Price: 80966.9, Quantity: 0.00328, QuoteQuantity: 265.571432, Time: types.Time(time.UnixMilli(1792897777298)), IsBuyerMaker: true},
				{ID: "4d900f90a1774904b60884537468edfe", Price: 80966.91, Quantity: 0.00017, QuoteQuantity: 13.7643747, Time: types.Time(time.UnixMilli(1792897777108)), IsBuyerMaker: true},
				{ID: "1356e49b40fa4c0993da1ac2794721d0", Price: 80966.61, Quantity: 0.00021, QuoteQuantity: 17.0029881, Time: types.Time(time.UnixMilli(1792897776502))},
			}
			assert.Equal(t, exp, result, "GetTrades should decode every trade")
			return
		}
		assert.Len(t, result, 3, "GetTrades should return the trades asked for")
	})
}

func TestGetKlines(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	ctx := t.Context()
	start := time.Unix(1769914800, 0)
	t.Run("nil request", func(t *testing.T) {
		_, err := e.GetKlines(ctx, nil)
		require.ErrorIs(t, err, common.ErrNilPointer, "GetKlines must reject a nil request")
	})
	for _, size := range []uint64{0, maxKlineSize + 1} {
		t.Run("size "+strconv.FormatUint(size, 10), func(t *testing.T) {
			_, err := e.GetKlines(ctx, &KlineRequest{Pair: spotTestPair, Interval: kline.OneDay, Start: start, Size: size})
			require.ErrorIsf(t, err, errLimitOutOfRange, "GetKlines must reject a size of %d", size)
		})
	}
	for _, interval := range []kline.Interval{kline.EightHour, kline.TwelveHour, kline.TwoHour} {
		t.Run(interval.Short()+" interval", func(t *testing.T) {
			_, err := e.GetKlines(ctx, &KlineRequest{Pair: spotTestPair, Interval: interval, Start: start, Size: 3})
			require.ErrorIsf(t, err, kline.ErrUnsupportedInterval, "GetKlines must reject a %s interval", interval)
		})
	}
	t.Run("no start", func(t *testing.T) {
		_, err := e.GetKlines(ctx, &KlineRequest{Pair: spotTestPair, Interval: kline.OneDay, Size: 3})
		require.ErrorIs(t, err, common.ErrDateUnset, "GetKlines must require a start")
	})
	t.Run("no pair", func(t *testing.T) {
		_, err := e.GetKlines(ctx, &KlineRequest{Interval: kline.OneDay, Start: start, Size: 3})
		require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetKlines must require a pair")
	})
	t.Run("candles", func(t *testing.T) {
		result, err := e.GetKlines(ctx, &KlineRequest{Pair: spotTestPair, Interval: kline.OneDay, Start: start, Size: 3})
		require.NoError(t, err, "GetKlines must not error")
		exp := []KlineResponse{
			{Time: time.Unix(1769875200, 0), Open: 81300.5, High: 82110.88, Low: 75855.7, Close: 77594.65, Volume: 16848.64871},
			{Time: time.Unix(1769961600, 0), Open: 77594.65, High: 79326.72, Low: 74609.01, Close: 78849.22, Volume: 19966.48606},
			{Time: time.Unix(1770048000, 0), Open: 78849.22, High: 79289.37, Low: 77053.36, Close: 78132, Volume: 12151.23631},
		}
		// These candles are long settled, so the live venue returns the same ones
		assert.Equal(t, exp, result, "GetKlines should decode every candle")
	})
}

func TestKlineResponseUnmarshalJSON(t *testing.T) { //nolint:tparallel // Cases share one decode target and run in order
	t.Parallel()
	var k KlineResponse
	t.Run("numbers and quoted numbers", func(t *testing.T) {
		require.NoError(t, k.UnmarshalJSON([]byte(`[1769875200,"81300.5",82110.88,75855.7,77594.65,"16848.64871"]`)), "UnmarshalJSON must accept numbers and quoted numbers")
		exp := KlineResponse{Time: time.Unix(1769875200, 0), Open: 81300.5, High: 82110.88, Low: 75855.7, Close: 77594.65, Volume: 16848.64871}
		assert.Equal(t, exp, k, "UnmarshalJSON should decode every field")
	})
	t.Run("missing field", func(t *testing.T) {
		assert.ErrorIs(t, k.UnmarshalJSON([]byte(`[1769875200,81300.5,82110.88,75855.7,77594.65]`)), errMalformedKline, "UnmarshalJSON should reject a candle missing a field")
	})
	t.Run("not a list", func(t *testing.T) {
		assert.Error(t, k.UnmarshalJSON([]byte(`{"t":1}`)), "UnmarshalJSON should reject a candle that is not a list")
	})
}

func TestSymbolParams(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	t.Run("no pair", func(t *testing.T) {
		_, err := e.symbolParams(currency.EMPTYPAIR)
		require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "symbolParams must require a pair")
	})
	t.Run("spot symbol", func(t *testing.T) {
		params, err := e.symbolParams(currency.NewBTCUSDT())
		require.NoError(t, err, "symbolParams must not error")
		assert.Equal(t, url.Values{"symbol": {"btc_usdt"}}, params, "symbolParams should format the pair as LBank's spot symbol")
	})
}
