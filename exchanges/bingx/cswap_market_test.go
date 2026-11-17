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
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetCoinMarginedContracts(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetCoinMarginedContracts(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedContracts should return the symbol formatting error")

	t.Run("one contract", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetCoinMarginedContracts(t.Context(), currency.NewBTCUSD())
		require.NoError(t, err, "GetCoinMarginedContracts must not error")
		if mockTests {
			exp := []CoinMarginedContractResponse{
				{Symbol: "BTC-USD", PricePrecision: 1, MinTickSize: 100, MinTradeValue: 100, MinQuantity: 1, Status: 1, TimeOnline: types.Time(time.UnixMilli(1710738000000)), DisplayName: "BTC-USD"},
			}
			assert.Equal(t, exp, result, "GetCoinMarginedContracts should decode every field")
			return
		}
		require.Len(t, result, 1, "GetCoinMarginedContracts must return the requested contract")
		assert.Equal(t, "BTC-USD", result[0].Symbol, "Symbol should be the requested symbol")
		assert.Positive(t, result[0].MinTickSize.Float64(), "MinTickSize should be positive")
	})

	t.Run("every contract", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetCoinMarginedContracts(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetCoinMarginedContracts must not error")
		if mockTests {
			exp := []CoinMarginedContractResponse{
				{Symbol: "BNB-USD", PricePrecision: 2, MinTickSize: 10, MinTradeValue: 10, MinQuantity: 1, Status: 1, TimeOnline: types.Time(time.UnixMilli(1713175200000)), DisplayName: "BNB-USD"},
				{Symbol: "SUI-USD", PricePrecision: 4, MinTickSize: 10, MinTradeValue: 10, MinQuantity: 1, Status: 1, TimeOnline: types.Time(time.UnixMilli(1741921200000)), DisplayName: "SUI-USD"},
				{Symbol: "UNI-USD", PricePrecision: 3, MinTickSize: 10, MinTradeValue: 10, MinQuantity: 1, Status: 1, TimeOnline: types.Time(time.UnixMilli(1710907200000)), DisplayName: "UNI-USD"},
			}
			assert.Equal(t, exp, result, "GetCoinMarginedContracts should decode every contract")
			return
		}
		assert.Greater(t, len(result), 1, "GetCoinMarginedContracts should return every contract")
	})
}

func TestGetCoinMarginedPriceAndFundingRate(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetCoinMarginedPriceAndFundingRate(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedPriceAndFundingRate should return the symbol formatting error")

	t.Run("one symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetCoinMarginedPriceAndFundingRate(t.Context(), currency.NewBTCUSD())
		require.NoError(t, err, "GetCoinMarginedPriceAndFundingRate must not error")
		if mockTests {
			exp := []CoinMarginedPriceAndFundingRateResponse{
				{Symbol: "BTC-USD", MarkPrice: 74341.3, IndexPrice: 74372.1, LastFundingRate: 0.0001, NextFundingTime: types.Time(time.UnixMilli(1794441600000))},
			}
			assert.Equal(t, exp, result, "GetCoinMarginedPriceAndFundingRate should decode every field")
			return
		}
		require.Len(t, result, 1, "GetCoinMarginedPriceAndFundingRate must return the requested symbol")
		assert.Equal(t, "BTC-USD", result[0].Symbol, "Symbol should be the requested symbol")
		assert.Positive(t, result[0].MarkPrice.Float64(), "MarkPrice should be positive")
	})

	t.Run("every symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetCoinMarginedPriceAndFundingRate(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetCoinMarginedPriceAndFundingRate must not error")
		if mockTests {
			exp := []CoinMarginedPriceAndFundingRateResponse{
				{Symbol: "BNB-USD", MarkPrice: 686.38, IndexPrice: 685.95, LastFundingRate: 0.000009, NextFundingTime: types.Time(time.UnixMilli(1794441600000))},
				{Symbol: "SOL-USD", MarkPrice: 84.801, IndexPrice: 84.855, LastFundingRate: 0.000003, NextFundingTime: types.Time(time.UnixMilli(1794441600000))},
			}
			assert.Equal(t, exp, result, "GetCoinMarginedPriceAndFundingRate should decode every symbol")
			return
		}
		assert.Greater(t, len(result), 1, "GetCoinMarginedPriceAndFundingRate should return every symbol")
	})
}

func TestGetCoinMarginedOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetCoinMarginedOpenInterest(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedOpenInterest should return the symbol formatting error")

	t.Run("one symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetCoinMarginedOpenInterest(t.Context(), currency.NewBTCUSD())
		require.NoError(t, err, "GetCoinMarginedOpenInterest must not error")
		if mockTests {
			exp := []CoinMarginedOpenInterestResponse{
				{Symbol: "BTC-USD", OpenInterest: 698.8058, Timestamp: types.Time(time.UnixMilli(1794441600000))},
			}
			assert.Equal(t, exp, result, "GetCoinMarginedOpenInterest should decode every field")
			return
		}
		require.Len(t, result, 1, "GetCoinMarginedOpenInterest must return the requested symbol")
		assert.Equal(t, "BTC-USD", result[0].Symbol, "Symbol should be the requested symbol")
		assert.Positive(t, result[0].OpenInterest.Float64(), "OpenInterest should be positive")
	})

	t.Run("every symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetCoinMarginedOpenInterest(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetCoinMarginedOpenInterest must not error")
		if mockTests {
			exp := []CoinMarginedOpenInterestResponse{
				{Symbol: "BTC-USD", OpenInterest: 698.8058, Timestamp: types.Time(time.UnixMilli(1794441600000))},
				{Symbol: "DOT-USD", OpenInterest: 408573.5186, Timestamp: types.Time(time.UnixMilli(1794441600000))},
			}
			assert.Equal(t, exp, result, "GetCoinMarginedOpenInterest should decode every symbol")
			return
		}
		assert.Greater(t, len(result), 1, "GetCoinMarginedOpenInterest should return every symbol")
	})
}

func TestGetCoinMarginedKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedKlineData(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetCoinMarginedKlineData should reject a nil request")
	_, err = e.GetCoinMarginedKlineData(t.Context(), &CoinMarginedKlineDataRequest{Interval: kline.OneHour})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedKlineData should reject an empty symbol")
	_, err = e.GetCoinMarginedKlineData(t.Context(), &CoinMarginedKlineDataRequest{Symbol: currency.NewBTCUSD(), Interval: kline.TenMin})
	assert.ErrorIs(t, err, errUnsupportedInterval, "GetCoinMarginedKlineData should reject an unsupported interval")
	_, err = e.GetCoinMarginedKlineData(t.Context(), &CoinMarginedKlineDataRequest{Symbol: currency.NewBTCUSD(), Interval: kline.OneHour, StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCoinMarginedKlineData should reject a start time after the end time")
	_, err = new(Exchange).GetCoinMarginedKlineData(t.Context(), &CoinMarginedKlineDataRequest{Symbol: currency.NewBTCUSD(), Interval: kline.OneHour})
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedKlineData should return the symbol formatting error")

	t.Run("sends every parameter", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method, "method should be GET")
			assert.Equal(t, "/openApi/cswap/v1/market/klines", r.URL.Path, "path should be the klines path")
			exp := url.Values{
				"symbol":    {"BTC-USD"},
				"interval":  {"1h"},
				"startTime": {"1794416400000"},
				"endTime":   {"1794427200000"},
				"timeZone":  {"8"},
				"limit":     {"2"},
			}
			assert.Equal(t, exp, r.URL.Query(), "query should hold every parameter")
			_, _ = io.WriteString(w, `{"code":0,"msg":"","data":[]}`)
		})
		_, err := ex.GetCoinMarginedKlineData(t.Context(), &CoinMarginedKlineDataRequest{
			Symbol:    currency.NewBTCUSD(),
			Interval:  kline.OneHour,
			StartTime: time.UnixMilli(1794416400000),
			EndTime:   time.UnixMilli(1794427200000),
			TimeZone:  8,
			Limit:     2,
		})
		assert.NoError(t, err, "GetCoinMarginedKlineData should not error")
	})

	t.Run("candles", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetCoinMarginedKlineData(t.Context(), &CoinMarginedKlineDataRequest{Symbol: currency.NewBTCUSD(), Interval: kline.OneHour, Limit: 3})
		require.NoError(t, err, "GetCoinMarginedKlineData must not error")
		if mockTests {
			exp := []CoinMarginedKlineResponse{
				{Open: 74393.1, Close: 74339.4, High: 74411.1, Low: 74329.8, Volume: 895, Time: types.Time(time.UnixMilli(1794423600000))},
				{Open: 74419.9, Close: 74393.1, High: 74543.1, Low: 74294.4, Volume: 16849, Time: types.Time(time.UnixMilli(1794420000000))},
				{Open: 74153.7, Close: 74419.9, High: 74674.4, Low: 74054.3, Volume: 37794, Time: types.Time(time.UnixMilli(1794416400000))},
			}
			assert.Equal(t, exp, result, "GetCoinMarginedKlineData should decode every field")
			return
		}
		require.Len(t, result, 3, "GetCoinMarginedKlineData must return the requested number of candles")
		assert.True(t, result[0].Time.Time().After(result[2].Time.Time()), "candles should arrive newest first")
	})
}

func TestGetCoinMarginedOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedOrderBook(t.Context(), currency.EMPTYPAIR, 5)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedOrderBook should reject an empty symbol")
	_, err = new(Exchange).GetCoinMarginedOrderBook(t.Context(), currency.NewBTCUSD(), 5)
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMarginedOrderBook should return the symbol formatting error")

	result, err := e.GetCoinMarginedOrderBook(t.Context(), currency.NewBTCUSD(), 5)
	require.NoError(t, err, "GetCoinMarginedOrderBook must not error")
	require.NotNil(t, result, "GetCoinMarginedOrderBook must return a book")
	if mockTests {
		exp := &CoinMarginedOrderBookResponse{
			Time: types.Time(time.UnixMilli(1794423767928)),
			Bids: orderbook.LevelsArrayPriceAmount{
				{Price: 74348.6, Amount: 3124},
				{Price: 74348.5, Amount: 1},
				{Price: 74343, Amount: 3261},
				{Price: 74342.4, Amount: 2379},
				{Price: 74341.8, Amount: 6793},
			},
			Asks: orderbook.LevelsArrayPriceAmount{
				{Price: 74354.6, Amount: 3887},
				{Price: 74354, Amount: 2325},
				{Price: 74350.6, Amount: 1165},
				{Price: 74348.8, Amount: 16},
				{Price: 74348.7, Amount: 2380},
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedOrderBook should decode every field")
		return
	}
	require.Len(t, result.Bids, 5, "GetCoinMarginedOrderBook must return the requested bid levels")
	require.Len(t, result.Asks, 5, "GetCoinMarginedOrderBook must return the requested ask levels")
	assert.Greater(t, result.Asks[0].Price, result.Asks[4].Price, "asks should arrive highest price first")
	assert.False(t, result.Time.Time().IsZero(), "Time should be set")
}

func TestGetCoinMargined24HourPriceChange(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetCoinMargined24HourPriceChange(t.Context(), currency.NewBTCUSD())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCoinMargined24HourPriceChange should return the symbol formatting error")

	t.Run("one symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetCoinMargined24HourPriceChange(t.Context(), currency.NewBTCUSD())
		require.NoError(t, err, "GetCoinMargined24HourPriceChange must not error")
		if mockTests {
			exp := []CoinMargined24HourPriceChangeResponse{
				{
					Symbol:             "BTC-USD",
					PriceChange:        972.7,
					PriceChangePercent: 1.33,
					LastPrice:          74348.6,
					LastQuantity:       9,
					HighPrice:          74674.4,
					LowPrice:           73100.4,
					Volume:             604402,
					QuoteVolume:        819.6,
					OpenPrice:          73375.9,
					CloseTime:          types.Time(time.UnixMilli(1794423768967)),
					BidPrice:           74348.6,
					BidQuantity:        3480,
					AskPrice:           74348.7,
					AskQuantity:        2670,
				},
			}
			assert.Equal(t, exp, result, "GetCoinMargined24HourPriceChange should decode every field")
			return
		}
		require.Len(t, result, 1, "GetCoinMargined24HourPriceChange must return the requested symbol")
		assert.Equal(t, "BTC-USD", result[0].Symbol, "Symbol should be the requested symbol")
		assert.Positive(t, result[0].LastPrice.Float64(), "LastPrice should be positive")
	})

	t.Run("every symbol", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetCoinMargined24HourPriceChange(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetCoinMargined24HourPriceChange must not error")
		if mockTests {
			exp := []CoinMargined24HourPriceChangeResponse{
				{
					Symbol:             "SUI-USD",
					PriceChange:        0.0222,
					PriceChangePercent: 2.72,
					LastPrice:          0.8371,
					LastQuantity:       3,
					HighPrice:          0.8524,
					LowPrice:           0.8121,
					Volume:             409824,
					QuoteVolume:        4950589.62,
					OpenPrice:          0.8147,
					CloseTime:          types.Time(time.UnixMilli(1794423770805)),
					BidPrice:           0.8371,
					BidQuantity:        16,
					AskPrice:           0.8372,
					AskQuantity:        13,
				},
				{
					Symbol:             "BCH-USD",
					PriceChange:        2.87,
					PriceChangePercent: 1.26,
					LastPrice:          231.31,
					LastQuantity:       2,
					HighPrice:          233.47,
					LowPrice:           227.18,
					Volume:             84473,
					QuoteVolume:        3675.02,
					OpenPrice:          228.35,
					CloseTime:          types.Time(time.UnixMilli(1794423766549)),
					BidPrice:           231.31,
					BidQuantity:        725,
					AskPrice:           231.32,
					AskQuantity:        1092,
				},
			}
			assert.Equal(t, exp, result, "GetCoinMargined24HourPriceChange should decode every symbol")
			return
		}
		assert.Greater(t, len(result), 1, "GetCoinMargined24HourPriceChange should return every symbol")
	})
}

func TestCoinMarginedSymbolParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    url.Values
	}{
		{"empty symbol", currency.EMPTYPAIR, url.Values{}},
		{"symbol without a delimiter", currency.NewBTCUSD(), url.Values{"symbol": {"BTC-USD"}}},
		{"symbol with another delimiter", currency.NewPairWithDelimiter("eth", "usd", currency.UnderscoreDelimiter), url.Values{"symbol": {"ETH-USD"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := e.coinMarginedSymbolParams(tc.symbol)
			require.NoError(t, err, "coinMarginedSymbolParams must not error")
			assert.Equal(t, tc.exp, params, "coinMarginedSymbolParams should return the formatted symbol")
		})
	}
	t.Run("unformattable symbol", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).coinMarginedSymbolParams(currency.NewBTCUSD())
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "coinMarginedSymbolParams should return the formatting error")
	})
}
