package binanceus

import (
	"cmp"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// btcUSDTAggregatedTrades are the aggregate trades the market fixtures record
var btcUSDTAggregatedTrades = []AggregatedTradeResponse{
	{AggregateTradeID: 30522835, Price: 73687.07, Quantity: 0.00075, FirstTradeID: 32183706, LastTradeID: 32183707, Timestamp: types.Time(time.UnixMilli(1794464066712)), IsBuyerMaker: true, IsBestMatch: true},
	{AggregateTradeID: 30522836, Price: 73683.77, Quantity: 0.00098, FirstTradeID: 32183708, LastTradeID: 32183708, Timestamp: types.Time(time.UnixMilli(1794464409788)), IsBuyerMaker: true, IsBestMatch: true},
	{AggregateTradeID: 30522837, Price: 73716.55, Quantity: 0.00198, FirstTradeID: 32183709, LastTradeID: 32183709, Timestamp: types.Time(time.UnixMilli(1794464635319)), IsBestMatch: true},
}

// btcUSDTTrades are the trades the recent and historical trades fixtures record
var btcUSDTTrades = []RecentTradeResponse{
	{ID: 32183707, Price: 73687.07, Quantity: 0.00042, QuoteQuantity: 30.9485694, Time: types.Time(time.UnixMilli(1794464066712)), IsBuyerMaker: true, IsBestMatch: true},
	{ID: 32183708, Price: 73683.77, Quantity: 0.00098, QuoteQuantity: 72.2100946, Time: types.Time(time.UnixMilli(1794464409788)), IsBuyerMaker: true, IsBestMatch: true},
	{ID: 32183709, Price: 73716.55, Quantity: 0.00198, QuoteQuantity: 145.958769, Time: types.Time(time.UnixMilli(1794464635319)), IsBestMatch: true},
}

func TestGetMostRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetMostRecentTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMostRecentTrades must reject a nil request")
	_, err = e.GetMostRecentTrades(t.Context(), &RecentTradeRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMostRecentTrades must reject an empty pair")

	t.Run("leaves an unset limit out", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			limit uint64
			exp   url.Values
		}{
			{exp: url.Values{"symbol": {"BTCUSDT"}}},
			{limit: 1000, exp: url.Values{"symbol": {"BTCUSDT"}, "limit": {"1000"}}},
		} {
			t.Run(strconv.FormatUint(tc.limit, 10), func(t *testing.T) {
				t.Parallel()
				ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/api/v3/trades", r.URL.Path, "GetMostRecentTrades should request the trades path")
					assert.Equal(t, tc.exp, r.URL.Query(), "GetMostRecentTrades should send a limit only when set")
					writeResponse(t, w, `[]`)
				})
				_, err := ex.GetMostRecentTrades(t.Context(), &RecentTradeRequest{Symbol: currency.NewBTCUSDT(), Limit: tc.limit})
				require.NoError(t, err, "GetMostRecentTrades must not error")
			})
		}
	})

	ensureTradablePairs(t)
	result, err := e.GetMostRecentTrades(t.Context(), &RecentTradeRequest{Symbol: spotTradablePair, Limit: 3})
	require.NoError(t, err, "GetMostRecentTrades must not error")
	if mockTests {
		assert.Equal(t, btcUSDTTrades, result, "GetMostRecentTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetMostRecentTrades should return trades")
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalTrades(t.Context(), currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricalTrades must reject an empty pair")

	t.Run("sends the limit and starting trade ID when set", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name          string
			limit, fromID uint64
			exp           url.Values
		}{
			{name: "neither", exp: url.Values{"symbol": {"BTCUSDT"}}},
			{name: "both", limit: 1000, fromID: 32183706, exp: url.Values{"symbol": {"BTCUSDT"}, "limit": {"1000"}, "fromId": {"32183706"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/api/v3/historicalTrades", r.URL.Path, "GetHistoricalTrades should request the historical trades path")
					assert.Equal(t, tc.exp, r.URL.Query(), "GetHistoricalTrades should send the set parameters")
					assert.Empty(t, r.Header.Get("X-MBX-APIKEY"), "GetHistoricalTrades should not send the API key, as the endpoint no longer takes one")
					writeResponse(t, w, `[]`)
				})
				_, err := ex.GetHistoricalTrades(t.Context(), currency.NewBTCUSDT(), tc.limit, tc.fromID)
				require.NoError(t, err, "GetHistoricalTrades must not error")
			})
		}
	})

	ensureTradablePairs(t)
	result, err := e.GetHistoricalTrades(t.Context(), spotTradablePair, 3, 0)
	require.NoError(t, err, "GetHistoricalTrades must not error")
	if mockTests {
		assert.Equal(t, btcUSDTTrades, result, "GetHistoricalTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetHistoricalTrades should return trades")
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetAggregatedTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAggregatedTrades must reject a nil request")
	_, err = e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAggregatedTrades must reject an empty pair")
	start := time.UnixMilli(1794464066712)
	_, err = e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), StartTime: start, FromID: 2})
	require.ErrorIs(t, err, errInvalidParameterCombination, "GetAggregatedTrades must reject fromId with a start time")
	_, err = e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), EndTime: start, FromID: 2})
	require.ErrorIs(t, err, errInvalidParameterCombination, "GetAggregatedTrades must reject fromId with an end time")
	_, err = e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), StartTime: start, EndTime: start.Add(-time.Millisecond)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAggregatedTrades must reject a start after the end")
	_, err = e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), Limit: 1001})
	require.ErrorIs(t, err, errAggregatedTradesStartRequired, "GetAggregatedTrades must reject a limit above 1000 without fromId or startTime")

	t.Run("sends a symbol alone or with the set parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *AggregatedTradeRequest
			exp  url.Values
		}{
			{name: "symbol only", req: &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT()}, exp: url.Values{"symbol": {"BTCUSDT"}}},
			{name: "from ID", req: &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), FromID: 30522835, Limit: 1000}, exp: url.Values{"symbol": {"BTCUSDT"}, "fromId": {"30522835"}, "limit": {"1000"}}},
			{name: "time window", req: &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), StartTime: start, EndTime: start.Add(time.Hour)}, exp: url.Values{"symbol": {"BTCUSDT"}, "startTime": {"1794464066712"}, "endTime": {"1794467666712"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/api/v3/aggTrades", r.URL.Path, "GetAggregatedTrades should request the aggregate trades path")
					assert.Equal(t, tc.exp, r.URL.Query(), "GetAggregatedTrades should send the set parameters")
					writeResponse(t, w, `[]`)
				})
				_, err := ex.GetAggregatedTrades(t.Context(), tc.req)
				require.NoError(t, err, "GetAggregatedTrades must not error")
			})
		}
	})

	ensureTradablePairs(t)
	result, err := e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: spotTradablePair, Limit: 3})
	require.NoError(t, err, "GetAggregatedTrades must not error")
	if mockTests {
		assert.Equal(t, btcUSDTAggregatedTrades, result, "GetAggregatedTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetAggregatedTrades should return trades")
}

// aggregatedTradesVenue serves aggregate trades with IDs from first to last, a millisecond apart from base, the way
// Binance.US pages them: fromId and startTime are inclusive, endTime too, and a page holds at most limit trades. It
// records the query of each request
type aggregatedTradesVenue struct {
	first, last uint64
	baseMilli   uint64
	mu          sync.Mutex
	queries     []url.Values
}

func (v *aggregatedTradesVenue) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		v.mu.Lock()
		v.queries = append(v.queries, q)
		v.mu.Unlock()
		limit, err := strconv.ParseUint(q.Get("limit"), 10, 64)
		if !assert.NoError(t, err, "aggregatedTradesVenue should be sent a limit") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		from := v.first
		if fromID := q.Get("fromId"); fromID != "" {
			if from, err = strconv.ParseUint(fromID, 10, 64); !assert.NoError(t, err, "aggregatedTradesVenue should be sent a numeric fromId") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		if startTime := q.Get("startTime"); startTime != "" {
			ms, err := strconv.ParseUint(startTime, 10, 64)
			if !assert.NoError(t, err, "aggregatedTradesVenue should be sent a numeric startTime") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if ms > v.baseMilli {
				from = max(from, v.first+ms-v.baseMilli)
			}
		}
		to := v.last
		if endTime := q.Get("endTime"); endTime != "" {
			ms, err := strconv.ParseUint(endTime, 10, 64)
			if !assert.NoError(t, err, "aggregatedTradesVenue should be sent a numeric endTime") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if ms < v.baseMilli {
				ms = v.baseMilli
			}
			to = min(to, v.first+ms-v.baseMilli)
		}
		var page strings.Builder
		page.WriteByte('[')
		for id, n := from, uint64(0); id <= to && n < limit; id, n = id+1, n+1 {
			if n > 0 {
				page.WriteByte(',')
			}
			page.WriteString(`{"a":` + strconv.FormatUint(id, 10) + `,"T":` + strconv.FormatUint(v.baseMilli+id-v.first, 10) + `}`)
		}
		page.WriteByte(']')
		writeResponse(t, w, page.String())
	}
}

func TestBatchAggregateTrades(t *testing.T) {
	t.Parallel()
	base := time.UnixMilli(1794464066000)
	for _, tc := range []struct {
		name     string
		req      *AggregatedTradeRequest
		first    uint64
		last     uint64
		expFirst uint64
		expLen   int
		requests int
	}{
		// The second page starts at the last trade of the first, which fromId returns again
		{name: "limit over two pages", req: &AggregatedTradeRequest{FromID: 100, Limit: 1500}, first: 100, last: 5000, expFirst: 100, expLen: 1500, requests: 2},
		{name: "limit ending a page early", req: &AggregatedTradeRequest{FromID: 100, Limit: 1001}, first: 100, last: 5000, expFirst: 100, expLen: 1001, requests: 2},
		{name: "fewer trades than the limit", req: &AggregatedTradeRequest{FromID: 100, Limit: 5000}, first: 100, last: 1599, expFirst: 100, expLen: 1500, requests: 3},
		{name: "start time to the end time", req: &AggregatedTradeRequest{StartTime: base.Add(10 * time.Millisecond), EndTime: base.Add(1209 * time.Millisecond), Limit: 5000}, first: 100, last: 5000, expFirst: 110, expLen: 1200, requests: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			venue := &aggregatedTradesVenue{first: tc.first, last: tc.last, baseMilli: 1794464066000}
			ex := newTestServerExchange(t, venue.handler(t))
			req := *tc.req
			req.Symbol = currency.NewBTCUSDT()
			result, err := ex.GetAggregatedTrades(t.Context(), &req)
			require.NoError(t, err, "GetAggregatedTrades must not error")
			require.Len(t, result, tc.expLen, "GetAggregatedTrades must collect the expected number of trades")
			for i := range result {
				require.Equalf(t, tc.expFirst+uint64(i), result[i].AggregateTradeID, "GetAggregatedTrades must not repeat or skip trade %d", i)
			}
			assert.Len(t, venue.queries, tc.requests, "GetAggregatedTrades should request the expected number of pages")
			for i, q := range venue.queries {
				assert.Equalf(t, "1000", q.Get("limit"), "GetAggregatedTrades should request page %d at the maximum limit", i)
				if i > 0 {
					assert.Emptyf(t, q.Get("startTime"), "GetAggregatedTrades should page %d on by fromId alone", i)
					assert.Emptyf(t, q.Get("endTime"), "GetAggregatedTrades should page %d on by fromId alone", i)
				}
			}
		})
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderBook must reject a nil request")
	_, err = e.GetOrderBook(t.Context(), &OrderBookRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderBook must reject an empty pair")

	t.Run("sends the set parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *OrderBookRequest
			exp  url.Values
		}{
			{name: "symbol only", req: &OrderBookRequest{Symbol: currency.NewBTCUSDT()}, exp: url.Values{"symbol": {"BTCUSDT"}}},
			{name: "limit and status", req: &OrderBookRequest{Symbol: currency.NewBTCUSDT(), Limit: 5000, SymbolStatus: SymbolStatusTrading}, exp: url.Values{"symbol": {"BTCUSDT"}, "limit": {"5000"}, "symbolStatus": {"TRADING"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/api/v3/depth", r.URL.Path, "GetOrderBook should request the depth path")
					assert.Equal(t, tc.exp, r.URL.Query(), "GetOrderBook should send the set parameters")
					writeResponse(t, w, `{}`)
				})
				_, err := ex.GetOrderBook(t.Context(), tc.req)
				require.NoError(t, err, "GetOrderBook must not error")
			})
		}
	})

	ensureTradablePairs(t)
	result, err := e.GetOrderBook(t.Context(), &OrderBookRequest{Symbol: spotTradablePair, Limit: 5})
	require.NoError(t, err, "GetOrderBook must not error")
	require.NotNil(t, result, "GetOrderBook must return a result")
	if mockTests {
		exp := &OrderBookResponse{
			LastUpdateID: 3417864065,
			Bids:         orderbook.LevelsArrayPriceAmount{{Price: 73735.39, Amount: 0.00626}, {Price: 73729.03, Amount: 0.03395}, {Price: 73728.61, Amount: 0.02837}, {Price: 73726.92, Amount: 0.00394}, {Price: 73722.83, Amount: 0.04973}},
			Asks:         orderbook.LevelsArrayPriceAmount{{Price: 73735.4, Amount: 0.02715}, {Price: 73736.12, Amount: 0.00198}, {Price: 73736.21, Amount: 0.00198}, {Price: 73736.23, Amount: 0.00198}, {Price: 73736.49, Amount: 0.00033}},
		}
		assert.Equal(t, exp, result, "GetOrderBook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "GetOrderBook should return bids")
	assert.NotEmpty(t, result.Asks, "GetOrderBook should return asks")
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotKline(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotKline must reject a nil request")
	_, err = e.GetSpotKline(t.Context(), &KlinesRequest{Interval: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotKline must reject an empty pair")
	_, err = e.GetSpotKline(t.Context(), &KlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.ThousandMilliseconds})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetSpotKline must reject an interval Binance.US does not serve")
	start := time.UnixMilli(1794459600000)
	_, err = e.GetSpotKline(t.Context(), &KlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneHour, StartTime: start, EndTime: start.Add(-time.Hour)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotKline must reject a start after the end")

	t.Run("sends the set parameters", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v3/klines", r.URL.Path, "GetSpotKline should request the klines path")
			exp := url.Values{"symbol": {"BTCUSDT"}, "interval": {"1M"}, "startTime": {"1794459600000"}, "endTime": {"1794463200000"}, "timeZone": {"+10:00"}, "limit": {"1000"}}
			assert.Equal(t, exp, r.URL.Query(), "GetSpotKline should send the set parameters")
			writeResponse(t, w, `[]`)
		})
		_, err := ex.GetSpotKline(t.Context(), &KlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneMonth, StartTime: start, EndTime: start.Add(time.Hour), TimeZone: "+10:00", Limit: 1000})
		require.NoError(t, err, "GetSpotKline must not error")
	})

	ensureTradablePairs(t)
	result, err := e.GetSpotKline(t.Context(), &KlinesRequest{Symbol: spotTradablePair, Interval: kline.OneHour, Limit: 2})
	require.NoError(t, err, "GetSpotKline must not error")
	if mockTests {
		exp := []CandleStickResponse{
			{OpenTime: types.Time(time.UnixMilli(1794459600000)), OpenPrice: 73729.46, HighPrice: 73729.46, LowPrice: 73538.15, ClosePrice: 73583.11, Volume: 0.20483, CloseTime: types.Time(time.UnixMilli(1794463199999)), QuoteAssetVolume: 15076.9394201, NumberOfTrades: 132, TakerBuyBaseAssetVolume: 0.10514, TakerBuyQuoteAssetVolume: 7737.6648209},
			{OpenTime: types.Time(time.UnixMilli(1794463200000)), OpenPrice: 73598.16, HighPrice: 73716.55, LowPrice: 73558.83, ClosePrice: 73716.55, Volume: 0.10383, CloseTime: types.Time(time.UnixMilli(1794466799999)), QuoteAssetVolume: 7646.0624178, NumberOfTrades: 50, TakerBuyBaseAssetVolume: 0.0333, TakerBuyQuoteAssetVolume: 2451.0655633},
		}
		assert.Equal(t, exp, result, "GetSpotKline should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetSpotKline should return the limit of klines")
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestSpotPrice(t.Context(), currency.Pairs{currency.EMPTYPAIR}, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLatestSpotPrice must reject an empty pair")

	t.Run("sends symbols and status as set", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v3/ticker/price", r.URL.Path, "GetLatestSpotPrice should request the price ticker path")
			assert.Equal(t, url.Values{"symbols": {`["BTCUSDT","ETHUSDT"]`}, "symbolStatus": {"HALT"}}, r.URL.Query(), "GetLatestSpotPrice should send the symbols and status")
			writeResponse(t, w, `[{"symbol":"BTCUSDT","price":"73716.55"},{"symbol":"ETHUSDT","price":"2713.26"}]`)
		})
		result, err := ex.GetLatestSpotPrice(t.Context(), currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, SymbolStatusHalt)
		require.NoError(t, err, "GetLatestSpotPrice must not error")
		assert.Equal(t, []SymbolPriceResponse{{Symbol: "BTCUSDT", Price: 73716.55}, {Symbol: "ETHUSDT", Price: 2713.26}}, result, "GetLatestSpotPrice should decode every symbol's price")
	})

	ensureTradablePairs(t)
	result, err := e.GetLatestSpotPrice(t.Context(), currency.Pairs{spotTradablePair}, "")
	require.NoError(t, err, "GetLatestSpotPrice must not error")
	if mockTests {
		assert.Equal(t, []SymbolPriceResponse{{Symbol: "BTCUSDT", Price: 73716.55}}, result, "GetLatestSpotPrice should decode the price")
		return
	}
	assert.Len(t, result, 1, "GetLatestSpotPrice should return the symbol's price")
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetAveragePrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAveragePrice must reject an empty pair")

	ensureTradablePairs(t)
	result, err := e.GetAveragePrice(t.Context(), spotTradablePair)
	require.NoError(t, err, "GetAveragePrice must not error")
	require.NotNil(t, result, "GetAveragePrice must return a result")
	if mockTests {
		assert.Equal(t, &AveragePriceResponse{IntervalInMinutes: 5, Price: 73705.69716216, CloseTime: types.Time(time.UnixMilli(1794464635319))}, result, "GetAveragePrice should decode every field")
		return
	}
	assert.Positive(t, result.Price.Float64(), "GetAveragePrice should return a price")
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetBestPrice(t.Context(), currency.Pairs{currency.EMPTYPAIR}, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetBestPrice must reject an empty pair")

	t.Run("sends no symbol for every symbol", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v3/ticker/bookTicker", r.URL.Path, "GetBestPrice should request the book ticker path")
			assert.Equal(t, url.Values{"symbolStatus": {"TRADING"}}, r.URL.Query(), "GetBestPrice should send only the status")
			writeResponse(t, w, `[{"symbol":"BTCUSDT","bidPrice":"1","bidQty":"2","askPrice":"3","askQty":"4"}]`)
		})
		result, err := ex.GetBestPrice(t.Context(), nil, SymbolStatusTrading)
		require.NoError(t, err, "GetBestPrice must not error")
		assert.Equal(t, []BookTickerResponse{{Symbol: "BTCUSDT", BidPrice: 1, BidQuantity: 2, AskPrice: 3, AskQuantity: 4}}, result, "GetBestPrice should decode every symbol's book ticker")
	})

	ensureTradablePairs(t)
	result, err := e.GetBestPrice(t.Context(), currency.Pairs{spotTradablePair}, "")
	require.NoError(t, err, "GetBestPrice must not error")
	if mockTests {
		assert.Equal(t, []BookTickerResponse{{Symbol: "BTCUSDT", BidPrice: 73735.39, BidQuantity: 0.00626, AskPrice: 73735.4, AskQuantity: 0.02715}}, result, "GetBestPrice should decode every field")
		return
	}
	assert.Len(t, result, 1, "GetBestPrice should return the symbol's book ticker")
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetPriceChangeStats(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPriceChangeStats must reject a nil request")
	_, err = e.GetPriceChangeStats(t.Context(), &PriceChangeStatsRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPriceChangeStats must reject an empty pair")

	t.Run("sends the type and status as set", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v3/ticker/24hr", r.URL.Path, "GetPriceChangeStats should request the 24hr ticker path")
			assert.Equal(t, url.Values{"type": {"MINI"}, "symbolStatus": {"BREAK"}}, r.URL.Query(), "GetPriceChangeStats should send the type and status")
			writeResponse(t, w, `[]`)
		})
		_, err := ex.GetPriceChangeStats(t.Context(), &PriceChangeStatsRequest{Type: TickerTypeMini, SymbolStatus: SymbolStatusBreak})
		require.NoError(t, err, "GetPriceChangeStats must not error")
	})

	ensureTradablePairs(t)
	result, err := e.GetPriceChangeStats(t.Context(), &PriceChangeStatsRequest{Symbols: currency.Pairs{spotTradablePair}})
	require.NoError(t, err, "GetPriceChangeStats must not error")
	if mockTests {
		exp := []PriceChangeStatsResponse{{
			Symbol:               "BTCUSDT",
			PriceChange:          -127.77,
			PriceChangePercent:   -0.173,
			WeightedAveragePrice: 73982.27701898,
			PreviousClosePrice:   73814.69,
			LastPrice:            73716.55,
			LastQuantity:         0.00198,
			BidPrice:             73735.39,
			BidQuantity:          0.00626,
			AskPrice:             73735.4,
			AskQuantity:          0.02715,
			OpenPrice:            73844.32,
			HighPrice:            74797.33,
			LowPrice:             73398.96,
			Volume:               52.65434,
			QuoteVolume:          3895487.9681315,
			OpenTime:             types.Time(time.UnixMilli(1794378380886)),
			CloseTime:            types.Time(time.UnixMilli(1794464780886)),
			FirstID:              32175295,
			LastID:               32183709,
			Count:                8415,
		}}
		assert.Equal(t, exp, result, "GetPriceChangeStats should decode every field")
		return
	}
	assert.Len(t, result, 1, "GetPriceChangeStats should return the symbol's statistics")
}

func TestGetTickerData(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickerData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTickerData must reject a nil request")
	_, err = e.GetTickerData(t.Context(), &RollingWindowTickerRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetTickerData must reject a request without symbols")
	_, err = e.GetTickerData(t.Context(), &RollingWindowTickerRequest{Symbols: make(currency.Pairs, tickerSymbolsLimit+1)})
	require.ErrorIs(t, err, errTooManySymbols, "GetTickerData must reject more than 100 symbols")
	_, err = e.GetTickerData(t.Context(), &RollingWindowTickerRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTickerData must reject an empty pair")
	_, err = e.GetTickerData(t.Context(), &RollingWindowTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}, WindowSize: 90 * time.Second})
	require.ErrorIs(t, err, errInvalidWindowSize, "GetTickerData must reject a window that is not whole minutes, hours or days")

	t.Run("sends the set parameters", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v3/ticker", r.URL.Path, "GetTickerData should request the rolling window ticker path")
			exp := url.Values{"symbols": {`["BTCUSDT","ETHUSDT"]`}, "windowSize": {"7d"}, "type": {"MINI"}, "symbolStatus": {"TRADING"}}
			assert.Equal(t, exp, r.URL.Query(), "GetTickerData should send the set parameters")
			writeResponse(t, w, `[]`)
		})
		_, err := ex.GetTickerData(t.Context(), &RollingWindowTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, WindowSize: 7 * 24 * time.Hour, Type: TickerTypeMini, SymbolStatus: SymbolStatusTrading})
		require.NoError(t, err, "GetTickerData must not error")
	})

	ensureTradablePairs(t)
	result, err := e.GetTickerData(t.Context(), &RollingWindowTickerRequest{Symbols: currency.Pairs{spotTradablePair}, WindowSize: time.Hour})
	require.NoError(t, err, "GetTickerData must not error")
	if mockTests {
		exp := []TickerStatisticsResponse{{
			Symbol:               "BTCUSDT",
			PriceChange:          106.25,
			PriceChangePercent:   0.144,
			WeightedAveragePrice: 73624.9438031,
			OpenPrice:            73610.3,
			HighPrice:            73716.55,
			LowPrice:             73538.15,
			LastPrice:            73716.55,
			Volume:               0.18832,
			QuoteVolume:          13865.049417,
			OpenTime:             types.Time(time.UnixMilli(1794461160000)),
			CloseTime:            types.Time(time.UnixMilli(1794464797714)),
			FirstID:              32183606,
			LastID:               32183709,
			Count:                104,
		}}
		assert.Equal(t, exp, result, "GetTickerData should decode every field")
		return
	}
	assert.Len(t, result, 1, "GetTickerData should return the symbol's statistics")
}

func TestGetTradingDayTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradingDayTicker(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradingDayTicker must reject a nil request")
	_, err = e.GetTradingDayTicker(t.Context(), &TradingDayTickerRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetTradingDayTicker must reject a request without symbols")
	_, err = e.GetTradingDayTicker(t.Context(), &TradingDayTickerRequest{Symbols: make(currency.Pairs, tickerSymbolsLimit+1)})
	require.ErrorIs(t, err, errTooManySymbols, "GetTradingDayTicker must reject more than 100 symbols")
	_, err = e.GetTradingDayTicker(t.Context(), &TradingDayTickerRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTradingDayTicker must reject an empty pair")

	t.Run("sends the set parameters", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v3/ticker/tradingDay", r.URL.Path, "GetTradingDayTicker should request the trading day ticker path")
			exp := url.Values{"symbol": {"BTCUSDT"}, "timeZone": {"-5"}, "type": {"FULL"}, "symbolStatus": {"HALT"}}
			assert.Equal(t, exp, r.URL.Query(), "GetTradingDayTicker should send the set parameters")
			writeResponse(t, w, `{}`)
		})
		_, err := ex.GetTradingDayTicker(t.Context(), &TradingDayTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}, TimeZone: "-5", Type: TickerTypeFull, SymbolStatus: SymbolStatusHalt})
		require.NoError(t, err, "GetTradingDayTicker must not error")
	})

	ensureTradablePairs(t)
	result, err := e.GetTradingDayTicker(t.Context(), &TradingDayTickerRequest{Symbols: currency.Pairs{spotTradablePair}})
	require.NoError(t, err, "GetTradingDayTicker must not error")
	if mockTests {
		exp := []TickerStatisticsResponse{{
			Symbol:               "BTCUSDT",
			PriceChange:          -316.36,
			PriceChangePercent:   -0.427,
			WeightedAveragePrice: 73811.70886223,
			OpenPrice:            74032.91,
			HighPrice:            74217.25,
			LowPrice:             73538.15,
			LastPrice:            73716.55,
			Volume:               15.3253,
			QuoteVolume:          1131186.5818263,
			OpenTime:             types.Time(time.UnixMilli(1794441600000)),
			CloseTime:            types.Time(time.UnixMilli(1794527999999)),
			FirstID:              32181814,
			LastID:               32183709,
			Count:                1896,
		}}
		assert.Equal(t, exp, result, "GetTradingDayTicker should decode every field")
		return
	}
	assert.Len(t, result, 1, "GetTradingDayTicker should return the symbol's statistics")
}

func TestObjectOrArrayUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := objectOrArray[SymbolPriceResponse]{{Symbol: "BTCUSDT", Price: 73716.55}, {Symbol: "ETHUSDT", Price: 2713.26}}
	var result objectOrArray[SymbolPriceResponse]
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"BTCUSDT","price":"73716.55000000"},{"symbol":"ETHUSDT","price":"2713.26000000"}]`), &result), "Unmarshal must not error for an array")
	assert.Equal(t, exp, result, "Unmarshal should decode every element of an array")

	result = nil
	require.NoError(t, json.Unmarshal([]byte(` {"symbol":"BTCUSDT","price":"73716.55000000"}`), &result), "Unmarshal must not error for an object")
	assert.Equal(t, exp[:1], result, "Unmarshal should decode an object into one element")

	assert.Error(t, json.Unmarshal([]byte(`{"symbol":1}`), &result), "Unmarshal should error for an invalid object")
	assert.Error(t, json.Unmarshal([]byte(`"BTCUSDT"`), &result), "Unmarshal should error for neither an object nor an array")
}

func TestCandleStickResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result CandleStickResponse
	require.NoError(t, json.Unmarshal([]byte(`[1794459600000,"73729.46000000","73729.46000000","73538.15000000","73583.11000000","0.20483000",1794463199999,"15076.93942010",132,"0.10514000","7737.66482090","0"]`), &result), "Unmarshal must not error")
	exp := CandleStickResponse{OpenTime: types.Time(time.UnixMilli(1794459600000)), OpenPrice: 73729.46, HighPrice: 73729.46, LowPrice: 73538.15, ClosePrice: 73583.11, Volume: 0.20483, CloseTime: types.Time(time.UnixMilli(1794463199999)), QuoteAssetVolume: 15076.9394201, NumberOfTrades: 132, TakerBuyBaseAssetVolume: 0.10514, TakerBuyQuoteAssetVolume: 7737.6648209}
	assert.Equal(t, exp, result, "Unmarshal should decode every element")
	assert.Error(t, json.Unmarshal([]byte(`{"openTime":1}`), &result), "Unmarshal should error for an object")
}

func TestWindowSizeToString(t *testing.T) {
	t.Parallel()
	const day = 24 * time.Hour
	for _, tc := range []struct {
		window time.Duration
		exp    string
	}{
		{time.Minute, "1m"}, {59 * time.Minute, "59m"}, {time.Hour, "1h"}, {23 * time.Hour, "23h"}, {day, "1d"}, {7 * day, "7d"},
	} {
		t.Run(tc.exp, func(t *testing.T) {
			t.Parallel()
			got, err := windowSizeToString(tc.window)
			require.NoErrorf(t, err, "windowSizeToString must not error for %s", tc.window)
			assert.Equalf(t, tc.exp, got, "windowSizeToString should format %s", tc.window)
		})
	}
	for _, window := range []time.Duration{0, time.Second, 90 * time.Second, 61 * time.Minute, 25 * time.Hour, 8 * day, -time.Hour} {
		t.Run(window.String(), func(t *testing.T) {
			t.Parallel()
			_, err := windowSizeToString(window)
			assert.ErrorIsf(t, err, errInvalidWindowSize, "windowSizeToString should reject %s", window)
		})
	}
}

func TestFormatToInterval(t *testing.T) {
	t.Parallel()
	for _, i := range klineIntervalList {
		t.Run(i.String, func(t *testing.T) {
			t.Parallel()
			got, err := formatToInterval(i.String)
			require.NoErrorf(t, err, "formatToInterval must not error for %q", i.String)
			assert.Equalf(t, i.Interval, got, "formatToInterval should return the interval of %q", i.String)
		})
	}
	for _, s := range []string{"", "1s", "1H", "10m"} {
		t.Run(cmp.Or(s, "empty"), func(t *testing.T) {
			t.Parallel()
			_, err := formatToInterval(s)
			assert.ErrorIsf(t, err, kline.ErrInvalidInterval, "formatToInterval should reject %q", s)
		})
	}
}

func TestIntervalToString(t *testing.T) {
	t.Parallel()
	for _, i := range klineIntervalList {
		t.Run(i.String, func(t *testing.T) {
			t.Parallel()
			got, err := intervalToString(i.Interval)
			require.NoErrorf(t, err, "intervalToString must not error for %s", i.Interval)
			assert.Equalf(t, i.String, got, "intervalToString should name %s", i.Interval)
		})
	}
	for _, interval := range []kline.Interval{0, kline.ThousandMilliseconds, kline.TenMin} {
		t.Run(interval.String(), func(t *testing.T) {
			t.Parallel()
			_, err := intervalToString(interval)
			assert.ErrorIsf(t, err, kline.ErrUnsupportedInterval, "intervalToString should reject %s", interval)
		})
	}
}

func TestCheckTickerSymbols(t *testing.T) {
	t.Parallel()
	assert.NoError(t, checkTickerSymbols(tickerSymbolsLimit), "checkTickerSymbols should accept the maximum")
	assert.ErrorIs(t, checkTickerSymbols(tickerSymbolsLimit+1), errTooManySymbols, "checkTickerSymbols should refuse more than the maximum")
}
