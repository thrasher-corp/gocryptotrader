package bingx

import (
	"io"
	"net/http"
	"net/url"
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

// spotTestRequest holds a request a spot test server received
type spotTestRequest struct {
	method   string
	path     string
	rawQuery string
	params   url.Values // Without timestamp and signature
}

// spotTestServer returns a test exchange whose REST requests are recorded on the returned channel and answered with
// body
func spotTestServer(tb testing.TB, body string) (ex *Exchange, received <-chan spotTestRequest) {
	tb.Helper()
	requests := make(chan spotTestRequest, 4)
	ex = newTestServerExchange(tb, func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		params.Del("timestamp")
		params.Del("signature")
		select {
		case requests <- spotTestRequest{method: r.Method, path: r.URL.Path, rawQuery: r.URL.RawQuery, params: params}:
		default:
		}
		_, _ = io.WriteString(w, body)
	})
	return ex, requests
}

// spotReceiveRequest returns the request a spot test server received during a completed call
func spotReceiveRequest(tb testing.TB, requests <-chan spotTestRequest) spotTestRequest {
	tb.Helper()
	select {
	case r := <-requests:
		return r
	default:
		require.FailNow(tb, "a request must have been received")
		return spotTestRequest{}
	}
}

func TestGetSpotSymbols(t *testing.T) {
	t.Parallel()
	t.Run("one symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSpotSymbols(t.Context(), currency.NewBTCUSDT())
		require.NoError(t, err, "GetSpotSymbols must not error")
		if mockTests {
			exp := []SpotSymbolResponse{
				{
					Symbol:            "BTC-USDT",
					TickSize:          0.01,
					StepSize:          0.000001,
					MinQuantity:       0.0001826,
					MaxQuantity:       18.2663756,
					MinNotional:       0.5,
					MaxNotional:       7588825.16352942,
					MaxMarketNotional: 7588825.16352942,
					Status:            SpotSymbolStatusOnline,
					APIStateBuy:       true,
					APIStateSell:      true,
					DisplayName:       "BTC-USDT",
				},
			}
			assert.Equal(t, exp, result, "GetSpotSymbols should decode every field")
			return
		}
		require.Len(t, result, 1, "GetSpotSymbols must return the requested symbol")
		assert.Equal(t, "BTC-USDT", result[0].Symbol, "Symbol should be the requested symbol")
		assert.Positive(t, result[0].TickSize, "TickSize should be positive")
	})

	t.Run("every symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSpotSymbols(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetSpotSymbols must not error")
		if mockTests {
			exp := []SpotSymbolResponse{
				{
					Symbol:            "EDGEX-USDT",
					TickSize:          0.0001,
					StepSize:          0.01,
					MinQuantity:       1.9,
					MaxQuantity:       18960.95,
					MinNotional:       1,
					MaxNotional:       3801,
					MaxMarketNotional: 3801,
					Status:            SpotSymbolStatusOffline,
					TimeOnline:        types.Time(time.UnixMilli(1774960200000)),
					OffTime:           types.Time(time.UnixMilli(1787281200000)),
					MaintainTime:      types.Time(time.UnixMilli(1787281200000)),
					DisplayName:       "EDGEX-USDT",
				},
				{
					Symbol:            "GPT-OLD-USDT",
					TickSize:          0.00001,
					StepSize:          0.1,
					MinQuantity:       211.9,
					MaxQuantity:       847816.9,
					MinNotional:       5,
					MaxNotional:       20000,
					MaxMarketNotional: -1,
					Status:            SpotSymbolStatusOffline,
					APIStateBuy:       true,
					APIStateSell:      true,
					TimeOnline:        types.Time(time.UnixMilli(1678446300000)),
					DisplayName:       "GPT-OLD",
				},
				{
					Symbol:            "ATOR-USDT",
					TickSize:          0.0001,
					StepSize:          0.01,
					MinQuantity:       10.08,
					MaxQuantity:       20155.49,
					MinNotional:       1,
					MaxNotional:       30000,
					MaxMarketNotional: -1,
					Status:            SpotSymbolStatusOffline,
					APIStateBuy:       true,
					APIStateSell:      true,
					TimeOnline:        types.Time(time.UnixMilli(1701331200000)),
					DisplayName:       "ATOR-USDT",
				},
			}
			assert.Equal(t, exp, result, "GetSpotSymbols should decode every listed symbol")
			return
		}
		assert.NotEmpty(t, result, "GetSpotSymbols should return symbols")
	})

	t.Run("unknown symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSpotSymbols(t.Context(), currency.NewPair(currency.NewCode("NOPE"), currency.USDT))
		require.NoError(t, err, "GetSpotSymbols must not error for an unknown symbol")
		assert.Empty(t, result, "GetSpotSymbols should return no symbols for an unknown symbol")
	})

	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetSpotSymbols(t.Context(), currency.NewBTCUSDT())
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotSymbols should return the symbol formatting error")
	})

	t.Run("every symbol request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"symbols":[]}}`)
		_, err := ex.GetSpotSymbols(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetSpotSymbols must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, "/openApi/spot/v1/common/symbols", r.path, "path should be the symbols endpoint")
		assert.Empty(t, r.params, "no parameters should be sent for every symbol")
	})
}

func TestGetSpotRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotRecentTrades(t.Context(), currency.EMPTYPAIR, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotRecentTrades must reject an empty pair")
	_, err = new(Exchange).GetSpotRecentTrades(t.Context(), currency.NewBTCUSDT(), 5)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotRecentTrades must return the symbol formatting error")

	result, err := e.GetSpotRecentTrades(t.Context(), currency.NewBTCUSDT(), 5)
	require.NoError(t, err, "GetSpotRecentTrades must not error")
	if mockTests {
		exp := []SpotRecentTradeResponse{
			{ID: 247152115, Price: 74450.01, Quantity: 0.002331, Time: types.Time(time.UnixMilli(1794423765865)), BuyerMaker: true},
			{ID: 247152114, Price: 74450.01, Quantity: 0.007322, Time: types.Time(time.UnixMilli(1794423765861)), BuyerMaker: true},
			{ID: 247152113, Price: 74450, Quantity: 0.010658, Time: types.Time(time.UnixMilli(1794423764862)), BuyerMaker: true},
			{ID: 247152112, Price: 74445.41, Quantity: 0.01346, Time: types.Time(time.UnixMilli(1794423763861))},
			{ID: 247152111, Price: 74445.4, Quantity: 0.001686, Time: types.Time(time.UnixMilli(1794423763370)), BuyerMaker: true},
		}
		assert.Equal(t, exp, result, "GetSpotRecentTrades should decode every field")
		return
	}
	require.Len(t, result, 5, "GetSpotRecentTrades must return the limit of trades")
	assert.NotZero(t, result[0].ID, "ID should be set")
	assert.Positive(t, result[0].Price, "Price should be positive")
}

func TestGetSpotOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOrderBook(t.Context(), currency.EMPTYPAIR, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotOrderBook must reject an empty pair")
	_, err = new(Exchange).GetSpotOrderBook(t.Context(), currency.NewBTCUSDT(), 5)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotOrderBook must return the symbol formatting error")

	result, err := e.GetSpotOrderBook(t.Context(), currency.NewBTCUSDT(), 5)
	require.NoError(t, err, "GetSpotOrderBook must not error")
	if mockTests {
		exp := &SpotOrderBookResponse{
			Bids: orderbook.LevelsArrayPriceAmount{
				{Price: 74452, Amount: 0.005314},
				{Price: 74447.96, Amount: 0.174095},
				{Price: 74447.19, Amount: 0.000383},
				{Price: 74447.11, Amount: 0.000129},
				{Price: 74446.8, Amount: 0.030268},
			},
			Asks: orderbook.LevelsArrayPriceAmount{
				{Price: 74456.96, Amount: 0.001785},
				{Price: 74456.51, Amount: 0.013364},
				{Price: 74456.06, Amount: 0.105006},
				{Price: 74456.05, Amount: 7.875292},
				{Price: 74452.02, Amount: 0.001846},
			},
			Timestamp:    types.Time(time.UnixMilli(1794423746568)),
			LastUpdateID: 16556488842,
		}
		assert.Equal(t, exp, result, "GetSpotOrderBook should decode every field")
		return
	}
	assert.Len(t, result.Bids, 5, "GetSpotOrderBook should return the limit of bids")
	assert.Len(t, result.Asks, 5, "GetSpotOrderBook should return the limit of asks")
	assert.NotZero(t, result.LastUpdateID, "LastUpdateID should be set")
}

func TestGetSpotKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotKlines(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotKlines must reject a nil request")
	_, err = e.GetSpotKlines(t.Context(), &SpotKlinesRequest{Interval: kline.OneMin})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotKlines must reject an empty pair")
	_, err = e.GetSpotKlines(t.Context(), &SpotKlinesRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetSpotKlines must reject an empty interval")
	_, err = e.GetSpotKlines(t.Context(), &SpotKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.TenMin})
	require.ErrorIs(t, err, errUnsupportedInterval, "GetSpotKlines must reject an unsupported interval")
	_, err = e.GetSpotKlines(t.Context(), &SpotKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneMin, StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotKlines must reject a start after the end")
	_, err = new(Exchange).GetSpotKlines(t.Context(), &SpotKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneMin})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotKlines must return the symbol formatting error")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"timestamp":1,"data":[]}`)
		_, err := ex.GetSpotKlines(t.Context(), &SpotKlinesRequest{
			Symbol:    currency.NewBTCUSDT(),
			Interval:  kline.OneDay,
			StartTime: time.UnixMilli(1794355200000),
			EndTime:   time.UnixMilli(1794441599999),
			TimeZone:  SpotKlineTimeZoneUTCPlus8,
			Limit:     3,
		})
		require.NoError(t, err, "GetSpotKlines must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodGet, r.method, "method should be GET")
		assert.Equal(t, "/openApi/spot/v2/quote/klines", r.path, "path should be the klines endpoint")
		exp := url.Values{
			"symbol":    {"BTC-USDT"},
			"interval":  {"1d"},
			"startTime": {"1794355200000"},
			"endTime":   {"1794441599999"},
			"timeZone":  {"8"},
			"limit":     {"3"},
		}
		assert.Equal(t, exp, r.params, "GetSpotKlines should send every parameter")
	})

	t.Run("latest candles", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSpotKlines(t.Context(), &SpotKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneMin, Limit: 2})
		require.NoError(t, err, "GetSpotKlines must not error")
		if mockTests {
			exp := []SpotKlineResponse{
				{
					OpenTime:    types.Time(time.UnixMilli(1794423720000)),
					Open:        74436.01,
					High:        74455.48,
					Low:         74436,
					Close:       74452,
					Volume:      0.472368,
					CloseTime:   types.Time(time.UnixMilli(1794423779999)),
					QuoteVolume: 35165.37,
				},
				{
					OpenTime:    types.Time(time.UnixMilli(1794423660000)),
					Open:        74441.71,
					High:        74456.01,
					Low:         74434.74,
					Close:       74436,
					Volume:      1.404755,
					CloseTime:   types.Time(time.UnixMilli(1794423719999)),
					QuoteVolume: 104577.06,
				},
			}
			assert.Equal(t, exp, result, "GetSpotKlines should decode every field")
			return
		}
		require.Len(t, result, 2, "GetSpotKlines must return the limit of candles")
		assert.True(t, result[0].OpenTime.Time().After(result[1].OpenTime.Time()), "candles should arrive newest first")
		openTime := result[0].OpenTime.Time()
		assert.Equal(t, openTime.Truncate(time.Minute), openTime, "a candle should open on a minute")
	})
}

func TestSpotKlineResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := SpotKlineResponse{
		OpenTime:    types.Time(time.UnixMilli(1794336480000)),
		Open:        73474.01400000001,
		High:        73477.68,
		Low:         73418.87,
		Close:       73438.98,
		Volume:      0.916659,
		CloseTime:   types.Time(time.UnixMilli(1794336539999)),
		QuoteVolume: 67315.312376,
	}
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "eight elements", raw: `[1794336480000,73474.01400000001,73477.68,73418.87,73438.98,0.916659,1794336539999,67315.312376]`},
		{name: "extra element", raw: `[1794336480000,73474.01400000001,73477.68,73418.87,73438.98,0.916659,1794336539999,67315.312376,42]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var k SpotKlineResponse
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &k), "Unmarshal must not error")
			assert.Equal(t, exp, k, "Unmarshal should decode every element in order")
			var direct SpotKlineResponse
			require.NoError(t, direct.UnmarshalJSON([]byte(tc.raw)), "UnmarshalJSON must not error")
			assert.Equal(t, exp, direct, "UnmarshalJSON should decode every element in order")
		})
	}

	t.Run("too few elements", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`[1794336480000,73474.01,73477.68,73418.87,73438.98,0.916659,1794336539999]`)
		var k SpotKlineResponse
		assert.ErrorIs(t, json.Unmarshal(raw, &k), errSpotKlineRowLength, "Unmarshal should reject a row with fewer than eight elements")
		assert.ErrorIs(t, k.UnmarshalJSON(raw), errSpotKlineRowLength, "UnmarshalJSON should reject a row with fewer than eight elements")
	})

	t.Run("bad element", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`[1794336480000,"x",73477.68,73418.87,73438.98,0.916659,1794336539999,67315.312376]`)
		var k SpotKlineResponse
		assert.Error(t, json.Unmarshal(raw, &k), "Unmarshal should reject an element of the wrong type")
		assert.ErrorContains(t, k.UnmarshalJSON(raw), "error decoding kline element 1", "UnmarshalJSON should name the element it cannot decode")
	})

	t.Run("not an array", func(t *testing.T) {
		t.Parallel()
		var k SpotKlineResponse
		assert.Error(t, json.Unmarshal([]byte(`{"open":1}`), &k), "Unmarshal should reject an object")
		assert.Error(t, k.UnmarshalJSON([]byte(`{"open":1}`)), "UnmarshalJSON should reject an object")
	})
}

func TestGetSpotTicker(t *testing.T) {
	t.Parallel()
	t.Run("one symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSpotTicker(t.Context(), currency.NewBTCUSDT())
		require.NoError(t, err, "GetSpotTicker must not error")
		if mockTests {
			exp := []SpotTickerResponse{
				{
					Symbol:             "BTC-USDT",
					OpenPrice:          73488.69,
					HighPrice:          74772.59,
					LowPrice:           73221.5,
					LastPrice:          74452.02,
					PriceChange:        963.33,
					Volume:             1804.96,
					QuoteVolume:        133296362.38,
					OpenTime:           types.Time(time.UnixMilli(1794337348498)),
					CloseTime:          types.Time(time.UnixMilli(1794423748498)),
					BidPrice:           74452,
					BidQuantity:        0.005314,
					AskPrice:           74452.02,
					AskQuantity:        0.001442,
					PriceChangePercent: 1.31,
				},
			}
			assert.Equal(t, exp, result, "GetSpotTicker should decode every field")
			return
		}
		require.Len(t, result, 1, "GetSpotTicker must return the requested symbol")
		assert.Equal(t, "BTC-USDT", result[0].Symbol, "Symbol should be the requested symbol")
		assert.Positive(t, result[0].LastPrice, "LastPrice should be positive")
	})

	t.Run("every symbol request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"timestamp":1,"data":[]}`)
		_, err := ex.GetSpotTicker(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetSpotTicker must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, "/openApi/spot/v2/quote/ticker", r.path, "path should be the ticker endpoint")
		assert.Empty(t, r.params, "no parameters should be sent for every symbol")
	})

	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetSpotTicker(t.Context(), currency.NewBTCUSDT())
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotTicker should return the symbol formatting error")
	})
}

func TestGetSpotAggregatedOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotAggregatedOrderBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotAggregatedOrderBook must reject a nil request")
	_, err = e.GetSpotAggregatedOrderBook(t.Context(), &SpotAggregatedOrderBookRequest{Type: OrderBookAggregationStep0})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotAggregatedOrderBook must reject an empty pair")
	_, err = e.GetSpotAggregatedOrderBook(t.Context(), &SpotAggregatedOrderBookRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, errSpotOrderBookAggregationEmpty, "GetSpotAggregatedOrderBook must reject an empty type, which BingX requires")
	_, err = new(Exchange).GetSpotAggregatedOrderBook(t.Context(), &SpotAggregatedOrderBookRequest{Symbol: currency.NewBTCUSDT(), Type: OrderBookAggregationStep0})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotAggregatedOrderBook must return the symbol formatting error")

	result, err := e.GetSpotAggregatedOrderBook(t.Context(), &SpotAggregatedOrderBookRequest{Symbol: currency.NewBTCUSDT(), Limit: 5, Type: OrderBookAggregationStep1})
	require.NoError(t, err, "GetSpotAggregatedOrderBook must not error")
	if mockTests {
		exp := &SpotAggregatedOrderBookResponse{
			Bids: orderbook.LevelsArrayPriceAmount{
				{Price: 74450, Amount: 0.00512},
				{Price: 74445.9, Amount: 0.13756},
				{Price: 74445.2, Amount: 0.000054},
				{Price: 74444.9, Amount: 0.000057},
				{Price: 74444.8, Amount: 0.000311},
			},
			Asks: orderbook.LevelsArrayPriceAmount{
				{Price: 74450.1, Amount: 0.002431},
				{Price: 74454.1, Amount: 8.110311},
				{Price: 74454.4, Amount: 0.877327},
				{Price: 74454.5, Amount: 0.015863},
				{Price: 74455, Amount: 3.649691},
			},
			Timestamp: types.Time(time.UnixMilli(1794423749322)),
		}
		assert.Equal(t, exp, result, "GetSpotAggregatedOrderBook should decode every field")
		return
	}
	assert.Len(t, result.Bids, 5, "GetSpotAggregatedOrderBook should return the limit of bids")
	assert.Len(t, result.Asks, 5, "GetSpotAggregatedOrderBook should return the limit of asks")
}

func TestGetSpotPriceTicker(t *testing.T) {
	t.Parallel()
	t.Run("one symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSpotPriceTicker(t.Context(), currency.NewBTCUSDT())
		require.NoError(t, err, "GetSpotPriceTicker must not error")
		if mockTests {
			exp := []SpotPriceTickerResponse{
				{Price: 74450, Symbol: "BTC-USDT", Timestamp: types.Time(time.UnixMilli(1794423749861))},
			}
			assert.Equal(t, exp, result, "GetSpotPriceTicker should decode every field")
			return
		}
		require.Len(t, result, 1, "GetSpotPriceTicker must return the requested symbol")
		assert.Positive(t, result[0].Price.Float64(), "Price should be positive")
	})

	t.Run("every symbol request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"timestamp":1,"data":[]}`)
		_, err := ex.GetSpotPriceTicker(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetSpotPriceTicker must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, "/openApi/spot/v2/quote/price", r.path, "path should be the price endpoint")
		assert.Empty(t, r.params, "no parameters should be sent for every symbol")
	})

	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetSpotPriceTicker(t.Context(), currency.NewBTCUSDT())
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotPriceTicker should return the symbol formatting error")
	})
}

func TestGetSpotBookTicker(t *testing.T) {
	t.Parallel()
	t.Run("one symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSpotBookTicker(t.Context(), currency.NewBTCUSDT())
		require.NoError(t, err, "GetSpotBookTicker must not error")
		if mockTests {
			exp := []SpotBookTickerResponse{
				{
					Time:        types.Time(time.UnixMilli(1794423750451)),
					Symbol:      "BTC-USDT",
					BidPrice:    74444.01,
					BidQuantity: 0.001884,
					AskPrice:    74444.03,
					AskQuantity: 0.003724,
				},
			}
			assert.Equal(t, exp, result, "GetSpotBookTicker should decode every field")
			return
		}
		require.Len(t, result, 1, "GetSpotBookTicker must return the requested symbol")
		assert.Positive(t, result[0].AskPrice.Float64(), "AskPrice should be positive")
	})

	t.Run("every symbol request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"timestamp":1,"data":[]}`)
		_, err := ex.GetSpotBookTicker(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetSpotBookTicker must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, "/openApi/spot/v2/quote/bookTicker", r.path, "path should be the book ticker endpoint")
		assert.Empty(t, r.params, "no parameters should be sent for every symbol")
	})

	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetSpotBookTicker(t.Context(), currency.NewBTCUSDT())
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotBookTicker should return the symbol formatting error")
	})
}

func TestGetSpotHistoricalKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotHistoricalKlines(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotHistoricalKlines must reject a nil request")
	_, err = e.GetSpotHistoricalKlines(t.Context(), &SpotHistoricalKlinesRequest{Interval: kline.OneMin})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotHistoricalKlines must reject an empty pair")
	_, err = e.GetSpotHistoricalKlines(t.Context(), &SpotHistoricalKlinesRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetSpotHistoricalKlines must reject an empty interval")
	_, err = e.GetSpotHistoricalKlines(t.Context(), &SpotHistoricalKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.TenMin})
	require.ErrorIs(t, err, errUnsupportedInterval, "GetSpotHistoricalKlines must reject an unsupported interval")
	_, err = e.GetSpotHistoricalKlines(t.Context(), &SpotHistoricalKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneMin, StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotHistoricalKlines must reject a start after the end")
	_, err = new(Exchange).GetSpotHistoricalKlines(t.Context(), &SpotHistoricalKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneMin})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotHistoricalKlines must return the symbol formatting error")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"timestamp":1,"data":[]}`)
		_, err := ex.GetSpotHistoricalKlines(t.Context(), &SpotHistoricalKlinesRequest{
			Symbol:    currency.NewBTCUSDT(),
			Interval:  kline.OneMin,
			StartTime: time.UnixMilli(1794200000000),
			EndTime:   time.UnixMilli(1794200300000),
			Limit:     3,
		})
		require.NoError(t, err, "GetSpotHistoricalKlines must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, "/openApi/spot/v2/quote/historicalKlines", r.path, "path should be the historical klines endpoint")
		exp := url.Values{
			"symbol":    {"BTC-USDT"},
			"interval":  {"1m"},
			"startTime": {"1794200000000"},
			"endTime":   {"1794200300000"},
			"limit":     {"3"},
		}
		assert.Equal(t, exp, r.params, "GetSpotHistoricalKlines should send every parameter")
	})

	t.Run("older candles", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSpotHistoricalKlines(t.Context(), &SpotHistoricalKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneMin, Limit: 2})
		require.NoError(t, err, "GetSpotHistoricalKlines must not error")
		if mockTests {
			exp := []SpotKlineResponse{
				{
					OpenTime:    types.Time(time.UnixMilli(1794336480000)),
					Open:        73474.01400000001,
					High:        73477.68,
					Low:         73418.87,
					Close:       73438.98,
					Volume:      0.916659,
					CloseTime:   types.Time(time.UnixMilli(1794336539999)),
					QuoteVolume: 67315.312376,
				},
				{
					OpenTime:    types.Time(time.UnixMilli(1794336420000)),
					Open:        73514.342,
					High:        73514.35,
					Low:         73474,
					Close:       73474.01400000001,
					Volume:      0.469901,
					CloseTime:   types.Time(time.UnixMilli(1794336479999)),
					QuoteVolume: 34534.130438,
				},
			}
			assert.Equal(t, exp, result, "GetSpotHistoricalKlines should decode every field")
			return
		}
		require.Len(t, result, 2, "GetSpotHistoricalKlines must return the limit of candles")
		assert.True(t, result[0].OpenTime.Time().After(result[1].OpenTime.Time()), "candles should arrive newest first without a time range")
	})
}

func TestGetSpotHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotHistoricalTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotHistoricalTrades must reject a nil request")
	_, err = e.GetSpotHistoricalTrades(t.Context(), &SpotHistoricalTradesRequest{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotHistoricalTrades must reject an empty pair")
	_, err = new(Exchange).GetSpotHistoricalTrades(t.Context(), &SpotHistoricalTradesRequest{Symbol: currency.NewBTCUSDT(), Limit: 5})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotHistoricalTrades must return the symbol formatting error")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"timestamp":1,"data":[]}`)
		_, err := ex.GetSpotHistoricalTrades(t.Context(), &SpotHistoricalTradesRequest{Symbol: currency.NewBTCUSDT(), Limit: 3, FromID: "1794423042362247150641"})
		require.NoError(t, err, "GetSpotHistoricalTrades must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, "/openApi/spot/v2/quote/historicalTrades", r.path, "path should be the historical trades endpoint")
		exp := url.Values{"symbol": {"BTC-USDT"}, "limit": {"3"}, "fromId": {"1794423042362247150641"}}
		assert.Equal(t, exp, r.params, "GetSpotHistoricalTrades should send every parameter")
	})

	t.Run("latest trades", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSpotHistoricalTrades(t.Context(), &SpotHistoricalTradesRequest{Symbol: currency.NewBTCUSDT(), Limit: 5})
		require.NoError(t, err, "GetSpotHistoricalTrades must not error")
		if mockTests {
			exp := []SpotHistoricalTradeResponse{
				{TradeID: "1794423749861247152084", Time: types.Time(time.UnixMilli(1794423749861)), TradeSide: SpotTradeSideBuy, Symbol: "BTC-USDT", Price: 74450.01, Quantity: 0.000445},
				{TradeID: "1794423749397247152083", Time: types.Time(time.UnixMilli(1794423749397)), TradeSide: SpotTradeSideSell, Symbol: "BTC-USDT", Price: 74450.01, Quantity: 0.005195},
				{TradeID: "1794423748861247152082", Time: types.Time(time.UnixMilli(1794423748861)), TradeSide: SpotTradeSideBuy, Symbol: "BTC-USDT", Price: 74450.01, Quantity: 0.01576},
				{TradeID: "1794423747869247152081", Time: types.Time(time.UnixMilli(1794423747869)), TradeSide: SpotTradeSideBuy, Symbol: "BTC-USDT", Price: 74452.02, Quantity: 0.011148},
				{TradeID: "1794423747865247152080", Time: types.Time(time.UnixMilli(1794423747865)), TradeSide: SpotTradeSideBuy, Symbol: "BTC-USDT", Price: 74452.02, Quantity: 0.024674},
			}
			assert.Equal(t, exp, result, "GetSpotHistoricalTrades should decode every field")
			return
		}
		require.Len(t, result, 5, "GetSpotHistoricalTrades must return the limit of trades")
		assert.NotEmpty(t, result[0].TradeID, "TradeID should be set")
		assert.Contains(t, []SpotTradeSide{SpotTradeSideBuy, SpotTradeSideSell}, result[0].TradeSide, "TradeSide should be buy or sell")
	})
}

func TestSpotSetSymbol(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	require.NoError(t, e.spotSetSymbol(params, currency.NewBTCUSDT()), "spotSetSymbol must not error")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}}, params, "spotSetSymbol should set the symbol in the spot request format")

	err := new(Exchange).spotSetSymbol(url.Values{}, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "spotSetSymbol should return the format error")
}
