package binance

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
	"golang.org/x/time/rate"
)

var (
	mostTradedOptionOnce sync.Once
	mostTradedOptionPair currency.Pair
)

// mostTradedOption returns the option with the most trades in the last day, which has candles and a populated order
// book for the live tests
func mostTradedOption(t *testing.T) currency.Pair {
	t.Helper()
	mostTradedOptionOnce.Do(func() {
		tickers, err := e.GetEOptions24HourTickerPriceChangeStatistics(t.Context(), currency.EMPTYPAIR)
		if err != nil || len(tickers) == 0 {
			return
		}
		busiest := slices.MaxFunc(tickers, func(a, b EOptionTickerResponse) int { return cmp.Compare(a.TradeCount, b.TradeCount) })
		mostTradedOptionPair, _ = currency.NewPairDelimiter(busiest.Symbol, currency.DashDelimiter)
	})
	require.False(t, mostTradedOptionPair.IsEmpty(), "mostTradedOption must find an option")
	return mostTradedOptionPair
}

// marketTestPairs returns each asset's pair for the market data tests: the pairs the fixtures record in mock tests, or
// the first enabled pairs and the most traded option when testing live
func marketTestPairs(t *testing.T) map[asset.Item]currency.Pair {
	t.Helper()
	ensureTradablePairs(t)
	if !mockTests {
		pairs := maps.Clone(assetToTradablePairMap)
		pairs[asset.Options] = mostTradedOption(t)
		return pairs
	}
	return map[asset.Item]currency.Pair{
		asset.Spot:                currency.NewBTCUSDT(),
		asset.Margin:              currency.NewBTCUSDT(),
		asset.USDTMarginedFutures: currency.NewBTCUSDT(),
		asset.CoinMarginedFutures: currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter),
		asset.Options:             currency.NewPairWithDelimiter("BTC", "261225-85000-C", currency.DashDelimiter),
	}
}

// newMarketTestExchange returns an exchange named after the test, so the ticker, order book, limits and account stores
// it writes do not collide with other tests', with the market test pairs enabled
func newMarketTestExchange(t *testing.T) *Exchange {
	t.Helper()
	pairs := marketTestPairs(t)
	ex := newTestExchange(t)
	if mockTests {
		// Mock responses are served locally, so throttling protects nothing
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	}
	ex.CurrencyPairs.Load(&e.CurrencyPairs)
	for a, p := range pairs {
		if mockTests {
			require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, false), "StorePairs must not error for available %s pairs", a)
			require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, true), "StorePairs must not error for enabled %s pairs", a)
			continue
		}
		err := ex.CurrencyPairs.EnablePair(a, p)
		require.Truef(t, err == nil || errors.Is(err, currency.ErrPairAlreadyEnabled), "EnablePair must not error for %s %s: %v", a, p, err)
	}
	return ex
}

func TestSetDefaults(t *testing.T) { //nolint:tparallel // the cases share the test's exchange with the checks around them
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	// The cases share ex with the checks after them, so they run in order
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			assert.Truef(t, ex.IsAssetWebsocketSupported(a), "SetDefaults should support %s websocket market data", a)
		})
	}
	assert.False(t, ex.IsAssetWebsocketSupported(asset.Margin), "SetDefaults should fetch margin market data over REST")
	assert.Equal(t, map[kline.Interval]bool{kline.OneHour: true, kline.FourHour: true, kline.EightHour: true}, ex.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies, "SetDefaults should list every funding interval")
	assert.Equal(t, map[asset.Item]bool{asset.USDTMarginedFutures: true, asset.CoinMarginedFutures: true}, ex.Features.Supports.FuturesCapabilities.FundingRateBatching, "SetDefaults should batch funding rates of both futures assets")
	// The cases share ex, so they run in order
	for _, interval := range []kline.Interval{kline.HundredMilliseconds, kline.FiveHundredMilliseconds} {
		t.Run(interval.Short(), func(t *testing.T) {
			assert.Falsef(t, ex.Features.Enabled.Kline.Intervals.ExchangeSupported(interval), "SetDefaults should not enable %s klines", interval)
		})
	}
	// The cases share ex, so they run in order
	for _, interval := range []kline.Interval{kline.ThousandMilliseconds, kline.OneMin, kline.EightHour, kline.ThreeDay, kline.OneMonth} {
		t.Run(interval.Short(), func(t *testing.T) {
			assert.Truef(t, ex.Features.Enabled.Kline.Intervals.ExchangeSupported(interval), "SetDefaults should enable %s klines", interval)
		})
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetServerTime must reject an unsupported asset")

	for _, tc := range []struct {
		a   asset.Item
		exp time.Time
	}{
		{asset.Spot, time.UnixMilli(1775571261666)},
		{asset.Margin, time.UnixMilli(1775571261666)},
		{asset.USDTMarginedFutures, time.UnixMilli(1791283479731)},
		{asset.CoinMarginedFutures, time.UnixMilli(1791298531317)},
		{asset.Options, time.UnixMilli(1791285187287)},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetServerTime(t.Context(), tc.a)
			require.NoError(t, err, "GetServerTime must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetServerTime should return the asset's server time")
				return
			}
			assert.WithinDuration(t, time.Now(), result, time.Minute, "GetServerTime should return the current time")
		})
	}
}

// mockContext bounds a request the recorded responses serve, which answer every request, so a pager that never stops
// fails at the deadline; a live request keeps the test's context
func mockContext(t *testing.T) context.Context {
	t.Helper()
	if !mockTests {
		return t.Context()
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// newRoutedTestExchange serves every REST host from replies keyed by request path, answering Binance error objects
// with a 400 status
func newRoutedTestExchange(t *testing.T, replies map[string]string) *Exchange {
	t.Helper()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply, ok := replies[r.URL.Path]
		if !assert.Truef(t, ok, "%s should not be requested", r.URL.Path) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.Contains(reply, `"code":-`) {
			w.WriteHeader(http.StatusBadRequest)
		}
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	})
	url, err := ex.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
	require.NoError(t, err, "GetURL must not error")
	for _, u := range []exchange.URL{exchange.RestUSDTMargined, exchange.RestCoinMargined, exchange.RestOptions} {
		require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(u.String(), url), "SetRunningURL must not error for %s", u)
	}
	return ex
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	ex := newMarketTestExchange(t)
	_, err := ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must reject an empty pair")
	_, err = ex.UpdateTicker(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must reject an unsupported asset")
	// The cases run in order, before the parallel cases below, which call UpdateTicker too
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		path string
	}{
		{asset.Spot, currency.NewBTCUSDT(), "/api/v3/ticker/24hr"},
		{asset.Margin, currency.NewBTCUSDT(), "/api/v3/ticker/24hr"},
		{asset.USDTMarginedFutures, currency.NewBTCUSDT(), "/fapi/v1/ticker/24hr"},
		{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), "/dapi/v1/ticker/24hr"},
		{asset.Options, currency.Pair{Base: currency.BTC, Quote: currency.NewCode("270924-90000-C"), Delimiter: currency.DashDelimiter}, "/eapi/v1/ticker"},
	} {
		t.Run("missing "+tc.a.String()+" ticker", func(t *testing.T) {
			empty := newRoutedTestExchange(t, map[string]string{tc.path: "[]"})
			_, err = empty.UpdateTicker(t.Context(), tc.pair, tc.a)
			require.ErrorIsf(t, err, ticker.ErrTickerNotFound, "UpdateTicker must report a missing %s ticker", tc.a)
		})
	}

	pairs := marketTestPairs(t)
	btcusdtSpot := ticker.Price{
		Last:                       67773.99,
		LastSize:                   0.00904,
		VolumeWeightedAveragePrice: 69011.84878802,
		High:                       70351.46,
		Low:                        67763.64,
		Bid:                        67773.99,
		BidSize:                    3.21885,
		Ask:                        67774,
		AskSize:                    3.28593,
		BaseVolume:                 16084.48849,
		QuoteVolume:                1110020287.5044677,
		Open:                       69415.23,
		PercentChange24Hour:        -2.364,
		Close:                      69415.23,
		Pair:                       currency.NewBTCUSDT(),
		ExchangeName:               ex.Name,
		LastUpdated:                time.UnixMilli(1775571563002),
	}
	optionsPriceChangePercent := 0.119
	for _, tc := range []struct {
		a   asset.Item
		exp *ticker.Price
	}{
		{a: asset.Spot, exp: new(btcusdtSpot)},
		{a: asset.Margin, exp: new(btcusdtSpot)},
		{
			a: asset.USDTMarginedFutures,
			exp: &ticker.Price{
				Last:                       86163,
				LastSize:                   0.013,
				VolumeWeightedAveragePrice: 85686.16,
				High:                       86683.9,
				Low:                        84910,
				BaseVolume:                 119574.386,
				QuoteVolume:                10245870317.88,
				Open:                       85993.8,
				PercentChange24Hour:        0.197,
				Pair:                       currency.NewBTCUSDT(),
				ExchangeName:               ex.Name,
				LastUpdated:                time.UnixMilli(1791283483051),
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: &ticker.Price{
				Last:                       85994.6,
				LastSize:                   1,
				VolumeWeightedAveragePrice: 85677.28191708,
				High:                       86664.7,
				Low:                        84911.4,
				BaseVolume:                 8571.2,
				Open:                       86035.6,
				PercentChange24Hour:        -0.048,
				Pair:                       currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter),
				ExchangeName:               ex.Name,
				LastUpdated:                time.UnixMilli(1791282919746),
			},
		},
		{
			a: asset.Options,
			exp: &ticker.Price{
				Last:                6965,
				LastSize:            0.03,
				High:                6985,
				Low:                 6225,
				Bid:                 7165,
				Ask:                 7240,
				BaseVolume:          0.3,
				QuoteVolume:         2087.7,
				Open:                6225,
				PercentChange24Hour: optionsPriceChangePercent * 100,
				Pair:                currency.NewPairWithDelimiter("BTC", "261225-85000-C", currency.DashDelimiter),
				ExchangeName:        ex.Name,
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			result, err := ex.UpdateTicker(t.Context(), pairs[tc.a], tc.a)
			require.NoError(t, err, "UpdateTicker must not error")
			if !mockTests {
				assert.Positive(t, result.Last+result.Bid+result.Ask, "UpdateTicker should return prices")
				return
			}
			tc.exp.AssetType = tc.a
			if tc.a == asset.Options {
				// Options tickers have no time of their statistics, so they are stamped when stored
				assert.WithinRange(t, result.LastUpdated, started, time.Now(), "UpdateTicker should stamp an options ticker when it is stored")
				tc.exp.LastUpdated = result.LastUpdated
			}
			assert.Equal(t, tc.exp, result, "UpdateTicker should return every field of the ticker")
		})
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ex := newMarketTestExchange(t)
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Futures), asset.ErrNotSupported, "UpdateTickers must reject an unsupported asset")

	if mockTests {
		// The spot tickers include a multibyte symbol, which must match its pair, and every asset's tickers include a
		// symbol that is not available, which must be skipped
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.NewCode("币安人生"), currency.USDT)}, false), "StorePairs must not error")
	}
	btcusdtSpot := ticker.Price{
		Last:                       86465.22,
		LastSize:                   0.00069,
		VolumeWeightedAveragePrice: 85770.60543635,
		High:                       86538.46,
		Low:                        84972.01,
		Bid:                        86465.21,
		BidSize:                    2.83802,
		Ask:                        86465.22,
		AskSize:                    3.18862,
		BaseVolume:                 12534.70327,
		QuoteVolume:                1075109088.4329567,
		Open:                       85609.36,
		PercentChange24Hour:        1,
		Close:                      85609.36,
		Pair:                       currency.NewBTCUSDT(),
		ExchangeName:               ex.Name,
		LastUpdated:                time.UnixMilli(1791298528013),
	}
	optionsPriceChangePercent := 0.041
	for _, tc := range []struct {
		a   asset.Item
		exp []*ticker.Price
	}{
		{
			a: asset.Spot,
			exp: []*ticker.Price{
				new(btcusdtSpot),
				{
					Last:                       0.4914,
					LastSize:                   186,
					VolumeWeightedAveragePrice: 0.49309468,
					High:                       0.514,
					Low:                        0.4795,
					Bid:                        0.4913,
					BidSize:                    241.9,
					Ask:                        0.4916,
					AskSize:                    519.7,
					BaseVolume:                 7692943.7,
					QuoteVolume:                3793349.59396,
					Open:                       0.5059,
					PercentChange24Hour:        -2.866,
					Close:                      0.5055,
					Pair:                       currency.NewPair(currency.NewCode("币安人生"), currency.USDT),
					ExchangeName:               ex.Name,
					LastUpdated:                time.UnixMilli(1791298524097),
				},
			},
		},
		{
			a:   asset.Margin,
			exp: []*ticker.Price{new(btcusdtSpot)},
		},
		{
			a: asset.USDTMarginedFutures,
			exp: []*ticker.Price{
				{
					Last:                       86163.1,
					LastSize:                   0.11,
					VolumeWeightedAveragePrice: 85686.17,
					High:                       86683.9,
					Low:                        84910,
					BaseVolume:                 119575.491,
					QuoteVolume:                10245965528.02,
					Open:                       85993.8,
					PercentChange24Hour:        0.197,
					Pair:                       currency.NewBTCUSDT(),
					ExchangeName:               ex.Name,
					LastUpdated:                time.UnixMilli(1791283486974),
				},
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: []*ticker.Price{
				{
					Last:                       86110.7,
					LastSize:                   104,
					VolumeWeightedAveragePrice: 85683.47028219,
					High:                       86664.7,
					Low:                        84911.4,
					BaseVolume:                 8668.3,
					Open:                       85881,
					PercentChange24Hour:        0.267,
					Pair:                       currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter),
					ExchangeName:               ex.Name,
					LastUpdated:                time.UnixMilli(1791285170154),
				},
			},
		},
		{
			a: asset.Options,
			exp: []*ticker.Price{
				{
					Last:                6480,
					LastSize:            0.02,
					High:                6480,
					Low:                 6225,
					Bid:                 6935,
					Ask:                 6985,
					BaseVolume:          0.02,
					QuoteVolume:         129.6,
					Open:                6225,
					PercentChange24Hour: optionsPriceChangePercent * 100,
					Pair:                currency.NewPairWithDelimiter("BTC", "261225-85000-C", currency.DashDelimiter),
					ExchangeName:        ex.Name,
				},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			require.NoError(t, ex.UpdateTickers(t.Context(), tc.a), "UpdateTickers must not error")
			result, err := ticker.GetExchangeTickers(ex.Name)
			require.NoError(t, err, "GetExchangeTickers must not error")
			result = slices.DeleteFunc(result, func(p *ticker.Price) bool { return p.AssetType != tc.a })
			slices.SortFunc(result, func(a, b *ticker.Price) int { return strings.Compare(a.Pair.String(), b.Pair.String()) })
			if !mockTests {
				assert.NotEmpty(t, result, "UpdateTickers should store tickers")
				return
			}
			require.Len(t, result, len(tc.exp), "UpdateTickers must store a ticker of every available symbol")
			for i := range tc.exp {
				tc.exp[i].AssetType = tc.a
				if tc.a == asset.Options {
					// Options tickers have no time of their statistics, so they are stamped when stored
					assert.WithinRange(t, result[i].LastUpdated, started, time.Now(), "UpdateTickers should stamp an options ticker when it is stored")
					tc.exp[i].LastUpdated = result[i].LastUpdated
				}
			}
			assert.Equal(t, tc.exp, result, "UpdateTickers should store every field of the tickers")
		})
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newMarketTestExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must reject an empty pair")
	_, err = ex.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject an unsupported asset")

	pairs := marketTestPairs(t)
	spotBook := orderbook.Book{
		Bids: orderbook.Levels{
			{Amount: 0.57234, Price: 67773.99},
			{Amount: 0.00008, Price: 67773.98},
			{Amount: 0.00008, Price: 67773.83},
			{Amount: 0.00008, Price: 67772.87},
			{Amount: 0.00008, Price: 67772.04},
		},
		Asks: orderbook.Levels{
			{Amount: 3.39871, Price: 67774},
			{Amount: 0.00008, Price: 67774.01},
			{Amount: 0.01452, Price: 67774.53},
			{Amount: 0.18939, Price: 67774.55},
			{Amount: 0.48717, Price: 67774.56},
		},
		Pair:         currency.NewBTCUSDT(),
		LastUpdateID: 91566051557,
	}
	for _, tc := range []struct {
		a   asset.Item
		exp *orderbook.Book
	}{
		{a: asset.Spot, exp: new(spotBook)},
		{a: asset.Margin, exp: new(spotBook)},
		{
			a: asset.USDTMarginedFutures,
			exp: &orderbook.Book{
				Bids: orderbook.Levels{
					{Amount: 13.582, Price: 86163},
					{Amount: 0.021, Price: 86162.9},
					{Amount: 0.034, Price: 86162.8},
				},
				Asks: orderbook.Levels{
					{Amount: 5.38, Price: 86163.1},
					{Amount: 0.957, Price: 86163.2},
					{Amount: 0.041, Price: 86163.3},
				},
				Pair:         currency.NewBTCUSDT(),
				LastUpdated:  time.UnixMilli(1791283481757),
				LastUpdateID: 11747025852863,
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: &orderbook.Book{
				Bids: orderbook.Levels{
					{Amount: 4217, Price: 86110.6},
					{Amount: 13, Price: 86110.5},
					{Amount: 1, Price: 86109.3},
				},
				Asks: orderbook.Levels{
					{Amount: 768, Price: 86110.7},
					{Amount: 1, Price: 86111.4},
					{Amount: 2, Price: 86112.6},
				},
				Pair:         currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter),
				LastUpdated:  time.UnixMilli(1791285170281),
				LastUpdateID: 11747177501814,
			},
		},
		{
			a: asset.Options,
			exp: &orderbook.Book{
				Bids: orderbook.Levels{
					{Amount: 6.92, Price: 7165},
					{Amount: 3.01, Price: 7160},
					{Amount: 0.01, Price: 1320},
					{Amount: 0.1, Price: 10},
					{Amount: 0.2, Price: 5},
				},
				Asks: orderbook.Levels{
					{Amount: 9.93, Price: 7240},
					{Amount: 4.22, Price: 7265},
					{Amount: 0.5, Price: 100000},
				},
				Pair:         currency.NewPairWithDelimiter("BTC", "261225-85000-C", currency.DashDelimiter),
				LastUpdated:  time.UnixMilli(1791298525867),
				LastUpdateID: 42821617338,
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			result, err := ex.UpdateOrderbook(t.Context(), pairs[tc.a], tc.a)
			require.NoError(t, err, "UpdateOrderbook must not error")
			if !mockTests {
				assert.NotEmpty(t, result.Bids, "UpdateOrderbook should return bids")
				return
			}
			tc.exp.Exchange, tc.exp.Asset, tc.exp.ValidateOrderbook, tc.exp.RestSnapshot = ex.Name, tc.a, true, true
			assert.WithinRange(t, result.InsertedAt, started, time.Now(), "UpdateOrderbook should insert the book now")
			tc.exp.InsertedAt = result.InsertedAt
			if tc.a == asset.Spot || tc.a == asset.Margin {
				// Spot order books have no timestamp, so they are stamped when stored
				assert.WithinRange(t, result.LastUpdated, started, time.Now(), "UpdateOrderbook should stamp a spot book when it is stored")
				tc.exp.LastUpdated = result.LastUpdated
			}
			assert.Equal(t, tc.exp, result, "UpdateOrderbook should return every level of the book")
		})
	}
}

// mockTradablePairs holds each asset's tradable pairs in the exchange information fixtures
var mockTradablePairs = map[asset.Item]currency.Pairs{
	asset.Spot: currency.Pairs{
		currency.NewPair(currency.ETH, currency.BTC),
		currency.NewPair(currency.LTC, currency.BTC),
		currency.NewBTCUSDT(),
		currency.NewPair(currency.NewCode("币安人生"), currency.USDT),
	}.Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}),
	asset.Margin: currency.Pairs{
		currency.NewPair(currency.ETH, currency.BTC),
		currency.NewPair(currency.BNB, currency.ETH),
		currency.NewBTCUSDT(),
		currency.NewPair(currency.NewCode("币安人生"), currency.USDT),
	}.Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}),
	asset.USDTMarginedFutures: currency.Pairs{
		currency.NewBTCUSDT(),
		currency.NewPair(currency.NewCode("BTCUSDT"), currency.NewCode("261225")),
	}.Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}),
	asset.CoinMarginedFutures: currency.Pairs{
		currency.NewPair(currency.NewCode("BTCUSD"), currency.PERP),
		currency.NewPair(currency.NewCode("BTCUSD"), currency.NewCode("261225")),
	}.Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}),
	asset.Options: currency.Pairs{
		currency.NewPair(currency.BTC, currency.NewCode("260410-72000-P")),
		currency.NewPair(currency.BTC, currency.NewCode("261225-85000-C")),
	}.Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}),
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must reject an unsupported asset")

	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.FetchTradablePairs(t.Context(), a)
			require.NoError(t, err, "FetchTradablePairs must not error")
			if mockTests {
				// Halted symbols, and spot or margin symbols not allowed to trade as the asset, are left out
				assert.Equal(t, mockTradablePairs[a], result, "FetchTradablePairs should return the trading symbols of the asset")
				return
			}
			assert.NotEmpty(t, result, "FetchTradablePairs should return pairs")
		})
	}
}

func TestUSDTMarginedFuturesSymbolPair(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		symbol           string
		baseAsset, quote currency.Code
		exp              currency.Pair
	}{
		{"BTCUSDT", currency.BTC, currency.USDT, currency.NewBTCUSDT()},
		{"1000SATSUSDT", currency.NewCode("1000SATS"), currency.USDT, currency.NewPair(currency.NewCode("1000SATS"), currency.USDT)},
		{"TSLAUSDT", currency.NewCode("TSLA"), currency.USDT, currency.NewPair(currency.NewCode("TSLA"), currency.USDT)},
		{"币安人生USDT", currency.NewCode("币安人生"), currency.USDT, currency.NewPair(currency.NewCode("币安人生"), currency.USDT)},
		{"BTCUSDC", currency.BTC, currency.USDC, currency.NewPair(currency.BTC, currency.USDC)},
		{"BTCUSDT_261225", currency.BTC, currency.USDT, currency.NewPair(currency.NewCode("BTCUSDT"), currency.NewCode("261225"))},
	} {
		t.Run(subtestName(tc.symbol), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, usdtMarginedFuturesSymbolPair(tc.symbol, tc.baseAsset, tc.quote), "usdtMarginedFuturesSymbolPair should split %s", tc.symbol)
		})
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := ex.UpdateTradablePairs(ctx)
	for _, a := range ex.GetAssetTypes(false) {
		assert.ErrorContainsf(t, err, a.String()+": ", "UpdateTradablePairs should report the %s failure and go on to the other assets", a)
	}
	assert.ErrorIs(t, err, context.Canceled, "UpdateTradablePairs should return the request errors")

	unloaded := new(Exchange)
	unloaded.SetDefaults()
	err = unloaded.UpdateTradablePairs(ctx)
	assert.ErrorIs(t, err, context.Canceled, "UpdateTradablePairs should return the request errors without stored pairs")
	assert.NotErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "UpdateTradablePairs should not report missing pairs when no asset updated")

	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	for _, a := range ex.GetAssetTypes(false) {
		pairs, err := ex.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		if mockTests {
			assert.Equalf(t, mockTradablePairs[a], pairs, "UpdateTradablePairs should store the %s tradable pairs", a)
		} else {
			assert.NotEmptyf(t, pairs, "UpdateTradablePairs should store %s pairs", a)
		}
		enabled, err := ex.GetEnabledPairs(a)
		require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
		assert.NotEmptyf(t, enabled, "UpdateTradablePairs should leave a %s pair enabled", a)
	}
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")

	ex := newTestExchange(t)
	ex.SetCredentials(&accounts.Credentials{})
	feeBuilder := setFeeBuilder()
	result, err := ex.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error without credentials")
	assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "GetFeeByType should estimate the trade fee offline without credentials")
	assert.Equal(t, 0.002, result, "GetFeeByType should return the offline trade fee")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	feeBuilder = setFeeBuilder()
	result, err = e.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")
	assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType, "GetFeeByType should fetch the trade fee with credentials")
	if mockTests {
		assert.Equal(t, 0.001, result, "GetFeeByType should return the account's taker fee")
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetFee(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFee must reject a nil fee builder")

	for _, tc := range []struct {
		name     string
		builder  *exchange.FeeBuilder
		exp      float64
		err      error
		mockOnly bool
	}{
		{name: "offline trade fee", builder: &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, PurchasePrice: 1000, Amount: 2}, exp: 4},
		// The account's commission rates are 0.1% for takers and 0.09% for makers
		{name: "taker trade fee", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, PurchasePrice: 1000, Amount: 2}, exp: 2},
		{name: "maker trade fee", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, PurchasePrice: 1000, Amount: 2, IsMaker: true}, exp: 1.8},
		{name: "negative trade fee", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, PurchasePrice: -1000, Amount: 2}},
		// BTC withdrawals cost 0.00002 on the default BTC network and 0.0000035 on BSC
		{name: "withdrawal fee", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewBTCUSDT()}, exp: 0.00002},
		{name: "withdrawal fee of a coin without a default network", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewPair(currency.EUR, currency.USDT)}, err: errDefaultNetworkNotFound, mockOnly: true},
		{name: "withdrawal fee of an unlisted coin", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewPair(currency.NewCode("GCTNOTLISTED"), currency.USDT)}, err: currency.ErrCurrencyNotFound},
		{name: "withdrawal fee of an empty coin", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee}, err: currency.ErrCurrencyCodeEmpty},
		{name: "deposit fee", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyDepositFee, Pair: currency.NewBTCUSDT()}},
		{name: "international bank withdrawal fee", builder: &exchange.FeeBuilder{FeeType: exchange.InternationalBankWithdrawalFee, FiatCurrency: currency.HKD}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests && tc.mockOnly {
				t.Skip("the fixtures record a coin without a default network")
			}
			fetched := tc.builder.FeeType == exchange.CryptocurrencyTradeFee || tc.builder.FeeType == exchange.CryptocurrencyWithdrawalFee && !tc.builder.Pair.Base.IsEmpty()
			if !mockTests && fetched {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetFee(t.Context(), tc.builder)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "GetFee must return the expected error")
				return
			}
			require.NoError(t, err, "GetFee must not error")
			if mockTests || !fetched {
				assert.Equal(t, tc.exp, result, "GetFee should return the fee")
			}
		})
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders must reject a nil request")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders must reject an unsupported asset")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin, MarginType: margin.Isolated, Side: order.AnySide, Type: order.OCO})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetActiveOrders must reject isolated margin order lists without pairs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	for _, tc := range []struct {
		name      string
		req       *order.MultiOrderRequest
		everyPair bool
		exp       order.FilteredOrders
	}{
		{
			name: "spot",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot},
			exp: []order.Detail{
				{
					TimeInForce:     order.GoodTillCancel,
					Price:           84000,
					Amount:          0.01,
					TriggerPrice:    85000,
					RemainingAmount: 0.01,
					Exchange:        "Binance",
					OrderID:         "2",
					ClientOrderID:   "myOrder2",
					Type:            order.StopLimit,
					Side:            order.Sell,
					Status:          order.New,
					AssetType:       asset.Spot,
					Date:            time.Unix(1744106000, 0),
					LastUpdated:     time.Unix(1744106000, 0),
					Pair:            currency.NewBTCUSDT(),
				},
				{
					TimeInForce:          order.GoodTillCancel,
					Price:                86000,
					Amount:               0.01,
					AverageExecutedPrice: 86000,
					ExecutedAmount:       0.001,
					RemainingAmount:      0.009,
					Cost:                 86,
					CostAsset:            currency.USDT,
					Exchange:             "Binance",
					OrderID:              "5",
					ClientOrderID:        "myOrder5",
					Type:                 order.Limit,
					Side:                 order.Buy,
					Status:               order.PartiallyFilled,
					AssetType:            asset.Spot,
					Date:                 time.Unix(1744109000, 0),
					LastUpdated:          time.Unix(1744109000, 0),
					Pair:                 currency.NewBTCUSDT(),
				},
			},
		},
		{
			name:      "spot every symbol",
			req:       &order.MultiOrderRequest{AssetType: asset.Spot},
			everyPair: true,
			exp: []order.Detail{
				{
					TimeInForce:     order.GoodTillCancel,
					Price:           84000,
					Amount:          0.01,
					TriggerPrice:    85000,
					RemainingAmount: 0.01,
					Exchange:        "Binance",
					OrderID:         "2",
					ClientOrderID:   "myOrder2",
					Type:            order.StopLimit,
					Side:            order.Sell,
					Status:          order.New,
					AssetType:       asset.Spot,
					Date:            time.Unix(1744106000, 0),
					LastUpdated:     time.Unix(1744106000, 0),
					Pair:            currency.NewBTCUSDT(),
				},
			},
		},
		{
			name: "margin isolated",
			req:  &order.MultiOrderRequest{AssetType: asset.Margin, MarginType: margin.Isolated},
			exp: []order.Detail{
				{
					TimeInForce:          order.GoodTillCancel,
					Price:                76005,
					Amount:               0.0003,
					TriggerPrice:         76000,
					AverageExecutedPrice: 76005,
					ExecutedAmount:       0.0001,
					RemainingAmount:      0.0002,
					Cost:                 7.6005,
					CostAsset:            currency.USDT,
					Exchange:             "Binance",
					OrderID:              "211842552",
					ClientOrderID:        "qhcZw71gAkCCTv0t0k8LUK",
					Type:                 order.TakeProfitLimit,
					Side:                 order.Sell,
					Status:               order.PartiallyFilled,
					AssetType:            asset.Margin,
					Date:                 time.Unix(1744150000, 0),
					LastUpdated:          time.Unix(1744150100, 0),
					Pair:                 currency.NewBTCUSDT(),
					MarginType:           margin.Isolated,
				},
			},
		},
		{
			// The cross margin order lists of every symbol are requested, and the ETHUSDT list is left out
			name: "margin cross OCO",
			req:  &order.MultiOrderRequest{AssetType: asset.Margin, Type: order.OCO},
			exp: []order.Detail{
				{
					Exchange:      "Binance",
					OrderID:       "33",
					ClientOrderID: "gct-margin-cross-open-oco",
					Type:          order.OCO,
					Status:        order.Active,
					AssetType:     asset.Margin,
					Date:          time.Unix(1744151000, 0),
					LastUpdated:   time.Unix(1744151000, 0),
					Pair:          currency.NewBTCUSDT(),
					MarginType:    margin.Multi,
				},
			},
		},
		{
			name: "margin isolated OCO",
			req:  &order.MultiOrderRequest{AssetType: asset.Margin, MarginType: margin.Isolated, Type: order.OCO},
			exp: []order.Detail{
				{
					Exchange:      "Binance",
					OrderID:       "31",
					ClientOrderID: "wuB13fmulKj3YjdqWEcsnp",
					Type:          order.OCO,
					Status:        order.Active,
					AssetType:     asset.Margin,
					Date:          time.Unix(1744150000, 0),
					LastUpdated:   time.Unix(1744150000, 0),
					Pair:          currency.NewBTCUSDT(),
					MarginType:    margin.Isolated,
				},
			},
		},
		{
			name: "USDⓈ-M",
			req:  &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures},
			exp: []order.Detail{
				{
					TimeInForce:          order.GoodTillDay,
					ReduceOnly:           true,
					Price:                80000,
					Amount:               0.002,
					TriggerPrice:         79000,
					AverageExecutedPrice: 80000,
					ExecutedAmount:       0.001,
					RemainingAmount:      0.001,
					Cost:                 80,
					Exchange:             "Binance",
					OrderID:              "22542179",
					ClientOrderID:        "testOrder",
					Type:                 order.Limit,
					Side:                 order.Buy,
					Status:               order.PartiallyFilled,
					AssetType:            asset.USDTMarginedFutures,
					Date:                 time.Unix(1744118000, 0),
					LastUpdated:          time.Unix(1744118400, 0),
					Pair:                 currency.NewBTCUSDT(),
				},
				{
					TimeInForce:   order.GoodTillCancel,
					ReduceOnly:    true,
					Price:         95000,
					Amount:        0.002,
					TriggerPrice:  94000,
					Exchange:      "Binance",
					OrderID:       "algo-2146764",
					ClientOrderID: "gct-um-take-profit",
					Type:          order.TakeProfitLimit,
					Side:          order.Sell,
					Status:        order.Active,
					AssetType:     asset.USDTMarginedFutures,
					Date:          time.Unix(1750485496, 0),
					LastUpdated:   time.Unix(1750485497, 0),
					Pair:          currency.NewBTCUSDT(),
				},
			},
		},
		{
			name:      "USDⓈ-M every symbol",
			req:       &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures},
			everyPair: true,
			exp: []order.Detail{
				{
					TimeInForce:          order.GoodTillDay,
					ReduceOnly:           true,
					Price:                80000,
					Amount:               0.002,
					TriggerPrice:         79000,
					AverageExecutedPrice: 80000,
					ExecutedAmount:       0.001,
					RemainingAmount:      0.001,
					Cost:                 80,
					Exchange:             "Binance",
					OrderID:              "22542179",
					ClientOrderID:        "testOrder",
					Type:                 order.Limit,
					Side:                 order.Buy,
					Status:               order.PartiallyFilled,
					AssetType:            asset.USDTMarginedFutures,
					Date:                 time.Unix(1744118000, 0),
					LastUpdated:          time.Unix(1744118400, 0),
					Pair:                 currency.NewBTCUSDT(),
				},
				{
					TimeInForce:          order.GoodTillCancel,
					ReduceOnly:           true,
					Price:                90000,
					Amount:               0.01,
					RemainingAmount:      0.01,
					TriggerPrice:         90000,
					AverageExecutedPrice: 90000,
					Exchange:             "Binance",
					OrderID:              "algo-2146760",
					ClientOrderID:        "6B2I9XVcJpCjqPAJ4YoFX7",
					Type:                 order.TakeProfitLimit,
					Side:                 order.Sell,
					Status:               order.New,
					AssetType:            asset.USDTMarginedFutures,
					Date:                 time.UnixMilli(1750485492076),
					LastUpdated:          time.UnixMilli(1750485492076),
					Pair:                 currency.NewBTCUSDT(),
				},
			},
		},
		{
			name: "USDⓈ-M limit",
			req:  &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Type: order.Limit},
			exp: []order.Detail{
				{
					TimeInForce:          order.GoodTillDay,
					ReduceOnly:           true,
					Price:                80000,
					Amount:               0.002,
					TriggerPrice:         79000,
					AverageExecutedPrice: 80000,
					ExecutedAmount:       0.001,
					RemainingAmount:      0.001,
					Cost:                 80,
					Exchange:             "Binance",
					OrderID:              "22542179",
					ClientOrderID:        "testOrder",
					Type:                 order.Limit,
					Side:                 order.Buy,
					Status:               order.PartiallyFilled,
					AssetType:            asset.USDTMarginedFutures,
					Date:                 time.Unix(1744118000, 0),
					LastUpdated:          time.Unix(1744118400, 0),
					Pair:                 currency.NewBTCUSDT(),
				},
			},
		},
		{
			name: "USDⓈ-M take profit",
			req:  &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Type: order.TakeProfitLimit},
			exp: []order.Detail{
				{
					TimeInForce:   order.GoodTillCancel,
					ReduceOnly:    true,
					Price:         95000,
					Amount:        0.002,
					TriggerPrice:  94000,
					Exchange:      "Binance",
					OrderID:       "algo-2146764",
					ClientOrderID: "gct-um-take-profit",
					Type:          order.TakeProfitLimit,
					Side:          order.Sell,
					Status:        order.Active,
					AssetType:     asset.USDTMarginedFutures,
					Date:          time.Unix(1750485496, 0),
					LastUpdated:   time.Unix(1750485497, 0),
					Pair:          currency.NewBTCUSDT(),
				},
			},
		},
		{
			name: "COIN-M",
			req:  &order.MultiOrderRequest{AssetType: asset.CoinMarginedFutures},
			exp: []order.Detail{
				{
					TimeInForce:          order.GoodTillCancel,
					ReduceOnly:           true,
					Price:                9020.5,
					Amount:               10,
					TriggerPrice:         9020,
					AverageExecutedPrice: 9022.5,
					ExecutedAmount:       4,
					RemainingAmount:      6,
					Cost:                 0.04432225,
					Exchange:             "Binance",
					OrderID:              "1917641",
					ClientOrderID:        "abc",
					Type:                 order.TrailingStop,
					Side:                 order.Buy,
					Status:               order.PartiallyFilled,
					AssetType:            asset.CoinMarginedFutures,
					Date:                 time.UnixMilli(1579276756075),
					LastUpdated:          time.UnixMilli(1579276756075),
					Pair:                 currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
				},
				{
					TimeInForce:     order.PostOnly,
					Price:           9020.5,
					Amount:          10,
					RemainingAmount: 10,
					Exchange:        "Binance",
					OrderID:         "1917643",
					ClientOrderID:   "abe",
					Type:            order.Limit,
					Side:            order.Buy,
					Status:          order.New,
					AssetType:       asset.CoinMarginedFutures,
					Date:            time.UnixMilli(1579276756075),
					LastUpdated:     time.UnixMilli(1579276756075),
					Pair:            currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
				},
				{
					TimeInForce:     order.GoodTillCancel,
					ReduceOnly:      true,
					Amount:          1,
					RemainingAmount: 1,
					TriggerPrice:    70000,
					Exchange:        "Binance",
					OrderID:         "algo-3100001",
					ClientOrderID:   "gct-cm-stop",
					Type:            order.StopMarket,
					Side:            order.Sell,
					Status:          order.New,
					AssetType:       asset.CoinMarginedFutures,
					Date:            time.UnixMilli(1744120800000),
					LastUpdated:     time.UnixMilli(1744120800000),
					Pair:            currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
				},
			},
		},
		{
			name: "options",
			req:  &order.MultiOrderRequest{AssetType: asset.Options},
			exp: []order.Detail{
				{
					TimeInForce:     order.GoodTillCancel,
					Price:           1300,
					Amount:          0.01,
					RemainingAmount: 0.01,
					CostAsset:       currency.USDT,
					Exchange:        "Binance",
					OrderID:         "4611875134427365380",
					ClientOrderID:   "gct-options-open",
					Type:            order.Limit,
					Side:            order.Sell,
					Status:          order.New,
					AssetType:       asset.Options,
					Date:            time.Unix(1744121300, 0),
					LastUpdated:     time.Unix(1744121300, 0),
					Pair:            currency.NewPairWithDelimiter("BTC", "260410-72000-P", "-"),
				},
				{
					TimeInForce:          order.GoodTillCancel,
					Price:                1300,
					Amount:               0.01,
					AverageExecutedPrice: 1300,
					ExecutedAmount:       0.004,
					RemainingAmount:      0.006,
					Cost:                 5.2,
					CostAsset:            currency.USDT,
					Exchange:             "Binance",
					OrderID:              "4611875134427365381",
					ClientOrderID:        "gct-options-partial",
					Type:                 order.Limit,
					Side:                 order.Sell,
					Status:               order.PartiallyFilled,
					AssetType:            asset.Options,
					Date:                 time.Unix(1744121400, 0),
					LastUpdated:          time.Unix(1744121500, 0),
					Pair:                 currency.NewPairWithDelimiter("BTC", "260410-72000-P", "-"),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !tc.everyPair {
				tc.req.Pairs = currency.Pairs{assetToTradablePairMap[tc.req.AssetType]}
			}
			if tc.req.Type == order.UnknownType {
				tc.req.Type = order.AnyType
			}
			tc.req.Side = order.AnySide
			resp, err := e.GetActiveOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetActiveOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, resp, "GetActiveOrders should return the open orders")
			}
		})
	}

	t.Run("spot OCO", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		}
		// The fixture's open order list is on LTCBTC, which this exchange has as an available pair
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.LTC, currency.BTC)}, false), "StorePairs must not error")
		resp, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.OCO, Side: order.AnySide})
		require.NoError(t, err, "GetActiveOrders must not error")
		if mockTests {
			exp := order.FilteredOrders{
				{
					Exchange:      ex.Name,
					OrderID:       "31",
					ClientOrderID: "wuB13fmulKj3YjdqWEcsnp",
					Type:          order.OCO,
					Status:        order.Active,
					AssetType:     asset.Spot,
					Date:          time.UnixMilli(1791284811009),
					LastUpdated:   time.UnixMilli(1791284811009),
					Pair:          currency.NewPair(currency.LTC, currency.BTC),
				},
			}
			assert.Equal(t, exp, resp, "GetActiveOrders should return the open order lists")
		}
	})

	t.Run("websocket API", func(t *testing.T) {
		t.Parallel()
		ex := newWsAPITestExchange(t, true)
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT()}, false), "StorePairs must not error")
		// The cases share ex and its WebSocket API connection, so each also checks a request keeps nothing from the last
		for orderType, exp := range map[order.Type]order.FilteredOrders{
			order.AnyType: {
				{
					TimeInForce:          order.GoodTillCancel,
					Price:                23416.1,
					Amount:               0.00847,
					TriggerPrice:         23400,
					AverageExecutedPrice: 23416.1,
					QuoteAmount:          198.335215,
					ExecutedAmount:       0.00847,
					Cost:                 198.334367,
					CostAsset:            currency.USDT,
					Exchange:             ex.Name,
					OrderID:              "12569099453",
					ClientOrderID:        "4d96324ff9d44481926157",
					Type:                 order.Limit,
					Side:                 order.Sell,
					Status:               order.Filled,
					AssetType:            asset.Spot,
					Date:                 time.UnixMilli(1660801715639),
					LastUpdated:          time.UnixMilli(1660801717945),
					Pair:                 currency.NewBTCUSDT(),
				},
			},
			order.OCO: {
				{
					Exchange:      ex.Name,
					OrderID:       "1274512",
					ClientOrderID: "08985fedd9ea2cf6b28996",
					Type:          order.OCO,
					Status:        order.Active,
					AssetType:     asset.Spot,
					Date:          time.UnixMilli(1660801713793),
					LastUpdated:   time.UnixMilli(1660801713793),
					Pair:          currency.NewBTCUSDT(),
				},
			},
		} {
			t.Run(orderType.String(), func(t *testing.T) {
				resp, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: orderType, Side: order.AnySide, Pairs: currency.Pairs{currency.NewBTCUSDT()}})
				require.NoErrorf(t, err, "GetActiveOrders must list %s orders over the WebSocket API", orderType)
				if mockTests {
					assert.Equalf(t, exp, resp, "GetActiveOrders should return the %s orders listed over the WebSocket API", orderType)
				}
			})
		}
	})
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory must reject a nil request")
	req := &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType}
	_, err = e.GetOrderHistory(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetOrderHistory must reject a request without pairs")
	req.Pairs = currency.Pairs{currency.NewBTCUSDT()}
	req.FromOrderID = "abc"
	_, err = e.GetOrderHistory(t.Context(), req)
	require.ErrorIs(t, err, errInvalidOrderID, "GetOrderHistory must reject an invalid from order ID")
	req.FromOrderID = ""
	req.AssetType = asset.Futures
	_, err = e.GetOrderHistory(t.Context(), req)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory must reject an unsupported asset")

	t.Run("orders by asset", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		}
		ensureTradablePairs(t)
		startTime, endTime := getTime()
		for _, tc := range []struct {
			name string
			req  *order.MultiOrderRequest
			exp  order.FilteredOrders
		}{
			{
				name: "spot",
				req:  &order.MultiOrderRequest{AssetType: asset.Spot, StartTime: startTime, EndTime: endTime},
				exp: []order.Detail{
					{
						TimeInForce:          order.GoodTillCancel,
						Amount:               0.002,
						AverageExecutedPrice: 86000,
						ExecutedAmount:       0.002,
						Cost:                 172,
						CostAsset:            currency.USDT,
						Exchange:             "Binance",
						OrderID:              "50",
						ClientOrderID:        "gct-history-filled",
						Type:                 order.Market,
						Side:                 order.Sell,
						Status:               order.Filled,
						AssetType:            asset.Spot,
						Date:                 time.Unix(1744110000, 0),
						LastUpdated:          time.Unix(1744110000, 0),
						Pair:                 currency.NewBTCUSDT(),
					},
					{
						TimeInForce:     order.GoodTillCancel,
						Price:           95000,
						Amount:          0.002,
						TriggerPrice:    94000,
						RemainingAmount: 0.002,
						Exchange:        "Binance",
						OrderID:         "51",
						ClientOrderID:   "gct-history-cancelled",
						Type:            order.TakeProfitLimit,
						Side:            order.Sell,
						Status:          order.Cancelled,
						AssetType:       asset.Spot,
						Date:            time.Unix(1744111000, 0),
						LastUpdated:     time.Unix(1744112000, 0),
						Pair:            currency.NewBTCUSDT(),
					},
					{
						TimeInForce:    order.GoodTillCancel,
						Price:          60000,
						Amount:         0.001,
						ExecutedAmount: 0.001,
						Exchange:       "Binance",
						OrderID:        "52",
						ClientOrderID:  "gct-history-legacy",
						Type:           order.Limit,
						Side:           order.Buy,
						Status:         order.Filled,
						AssetType:      asset.Spot,
						Date:           time.Unix(1744113000, 0),
						LastUpdated:    time.Unix(1744113000, 0),
						Pair:           currency.NewBTCUSDT(),
					},
				},
			},
			{
				name: "spot OCO",
				req:  &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.OCO, FromOrderID: "29"},
				exp: []order.Detail{
					{
						Exchange:      "Binance",
						OrderID:       "29",
						ClientOrderID: "amEEAXryFzFwYF1FeRpUoZ",
						Type:          order.OCO,
						Status:        order.Closed,
						AssetType:     asset.Spot,
						Date:          time.Unix(1744114000, 0),
						LastUpdated:   time.Unix(1744114000, 0),
						Pair:          currency.NewBTCUSDT(),
					},
				},
			},
			{
				name: "margin",
				req:  &order.MultiOrderRequest{AssetType: asset.Margin, FromOrderID: "1"},
				exp: []order.Detail{
					{
						TimeInForce:          order.GoodTillCancel,
						Price:                80000,
						Amount:               0.001,
						AverageExecutedPrice: 79990,
						ExecutedAmount:       0.001,
						Cost:                 79.99,
						CostAsset:            currency.USDT,
						Exchange:             "Binance",
						OrderID:              "28457",
						ClientOrderID:        "gct-margin-limit",
						Type:                 order.Limit,
						Side:                 order.Buy,
						Status:               order.Filled,
						AssetType:            asset.Margin,
						Date:                 time.Unix(1791284814, 0),
						LastUpdated:          time.UnixMilli(1791284814200),
						Pair:                 currency.NewBTCUSDT(),
						MarginType:           margin.Multi,
					},
					{
						TimeInForce:     order.GoodTillCancel,
						Amount:          0.001,
						TriggerPrice:    70000,
						RemainingAmount: 0.001,
						Exchange:        "Binance",
						OrderID:         "28462",
						ClientOrderID:   "gct-margin-stop",
						Type:            order.StopMarket,
						Side:            order.Sell,
						Status:          order.Expired,
						AssetType:       asset.Margin,
						Date:            time.Unix(1791284818, 0),
						LastUpdated:     time.Unix(1791284819, 0),
						Pair:            currency.NewBTCUSDT(),
						MarginType:      margin.Multi,
					},
				},
			},
			{
				// The cross margin order lists of every symbol are requested, and the ETHUSDT list is left out
				name: "margin cross OCO",
				req:  &order.MultiOrderRequest{AssetType: asset.Margin, Type: order.OCO},
				exp: []order.Detail{
					{
						Exchange:      "Binance",
						OrderID:       "1003",
						ClientOrderID: "gct-margin-cross-oco",
						Type:          order.OCO,
						Status:        order.Closed,
						AssetType:     asset.Margin,
						Date:          time.Unix(1791284821, 0),
						LastUpdated:   time.Unix(1791284821, 0),
						Pair:          currency.NewBTCUSDT(),
						MarginType:    margin.Multi,
					},
				},
			},
			{
				name: "margin isolated OCO",
				req:  &order.MultiOrderRequest{AssetType: asset.Margin, MarginType: margin.Isolated, Type: order.OCO},
				exp: []order.Detail{
					{
						Exchange:      "Binance",
						OrderID:       "1002",
						ClientOrderID: "gct-margin-isolated-oco",
						Type:          order.OCO,
						Status:        order.Closed,
						AssetType:     asset.Margin,
						Date:          time.Unix(1791284820, 0),
						LastUpdated:   time.Unix(1791284820, 0),
						Pair:          currency.NewBTCUSDT(),
						MarginType:    margin.Isolated,
					},
				},
			},
			{
				name: "USDⓈ-M",
				req:  &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, StartTime: startTime, EndTime: endTime},
				exp: []order.Detail{
					{
						TimeInForce:          order.GoodTillCancel,
						Price:                80000,
						Amount:               0.002,
						AverageExecutedPrice: 80000,
						ExecutedAmount:       0.002,
						Cost:                 160,
						Exchange:             "Binance",
						OrderID:              "22542184",
						ClientOrderID:        "gct-um-filled",
						Type:                 order.Limit,
						Side:                 order.Buy,
						Status:               order.Filled,
						AssetType:            asset.USDTMarginedFutures,
						Date:                 time.Unix(1744120000, 0),
						LastUpdated:          time.Unix(1744120500, 0),
						Pair:                 currency.NewBTCUSDT(),
					},
					{
						TimeInForce:     order.GoodTillCancel,
						Price:           80000,
						Amount:          0.002,
						RemainingAmount: 0.002,
						Exchange:        "Binance",
						OrderID:         "22542185",
						ClientOrderID:   "gct-um-stp",
						Type:            order.Market,
						Side:            order.Sell,
						Status:          order.Expired,
						AssetType:       asset.USDTMarginedFutures,
						Date:            time.Unix(1744121000, 0),
						LastUpdated:     time.Unix(1744121000, 0),
						Pair:            currency.NewBTCUSDT(),
					},
					{
						TimeInForce:     order.PostOnly,
						Price:           80000,
						Amount:          0.002,
						RemainingAmount: 0.002,
						Exchange:        "Binance",
						OrderID:         "22542186",
						ClientOrderID:   "gct-um-rpi",
						Type:            order.Limit,
						Side:            order.Buy,
						Status:          order.New,
						AssetType:       asset.USDTMarginedFutures,
						Date:            time.Unix(1744122000, 0),
						LastUpdated:     time.Unix(1744122000, 0),
						Pair:            currency.NewBTCUSDT(),
					},
					{
						TimeInForce:          order.GoodTillCancel,
						ReduceOnly:           true,
						Amount:               0.002,
						TriggerPrice:         70000,
						AverageExecutedPrice: 69990,
						Exchange:             "Binance",
						OrderID:              "algo-2146765",
						ClientOrderID:        "gct-um-finished",
						Type:                 order.StopMarket,
						Side:                 order.Sell,
						Status:               order.Closed,
						AssetType:            asset.USDTMarginedFutures,
						Date:                 time.Unix(1744123000, 0),
						LastUpdated:          time.Unix(1744124000, 0),
						Pair:                 currency.NewBTCUSDT(),
					},
				},
			},
			{
				name: "COIN-M",
				req:  &order.MultiOrderRequest{AssetType: asset.CoinMarginedFutures, FromOrderID: "1573346959"},
				exp: []order.Detail{
					{
						TimeInForce:          order.GoodTillCancel,
						Price:                80000,
						Amount:               1,
						AverageExecutedPrice: 80000,
						ExecutedAmount:       1,
						Cost:                 0.00125,
						Exchange:             "Binance",
						OrderID:              "1573346959",
						ClientOrderID:        "gct-cm-filled",
						Type:                 order.Limit,
						Side:                 order.Buy,
						Status:               order.Filled,
						AssetType:            asset.CoinMarginedFutures,
						Date:                 time.Unix(1744120400, 0),
						LastUpdated:          time.Unix(1744120500, 0),
						Pair:                 currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
					},
					{
						TimeInForce:     order.GoodTillCancel,
						Price:           80000,
						Amount:          1,
						RemainingAmount: 1,
						Exchange:        "Binance",
						OrderID:         "1573346963",
						ClientOrderID:   "gct-cm-trailing",
						Type:            order.TrailingStop,
						Side:            order.Buy,
						Status:          order.Cancelled,
						AssetType:       asset.CoinMarginedFutures,
						Date:            time.Unix(1744120600, 0),
						LastUpdated:     time.Unix(1744120700, 0),
						Pair:            currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
					},
					{
						TimeInForce:          order.GoodTillCancel,
						ReduceOnly:           true,
						Amount:               1,
						TriggerPrice:         90000,
						AverageExecutedPrice: 90010,
						Exchange:             "Binance",
						OrderID:              "algo-3100002",
						ClientOrderID:        "gct-cm-take-profit",
						Type:                 order.TakeProfitMarket,
						Side:                 order.Sell,
						Status:               order.Closed,
						AssetType:            asset.CoinMarginedFutures,
						Date:                 time.UnixMilli(1744120900000),
						LastUpdated:          time.UnixMilli(1744121000000),
						Pair:                 currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
					},
				},
			},
			{
				name: "options",
				req:  &order.MultiOrderRequest{AssetType: asset.Options},
				exp: []order.Detail{
					{
						TimeInForce:          order.GoodTillCancel,
						Price:                1300,
						Amount:               0.01,
						AverageExecutedPrice: 1300,
						ExecutedAmount:       0.01,
						Cost:                 13,
						CostAsset:            currency.USDT,
						Exchange:             "Binance",
						OrderID:              "4611875134427365382",
						ClientOrderID:        "gct-options-filled",
						Type:                 order.Limit,
						Side:                 order.Sell,
						Status:               order.Filled,
						AssetType:            asset.Options,
						Date:                 time.Unix(1744121600, 0),
						LastUpdated:          time.Unix(1744121700, 0),
						Pair:                 currency.NewPairWithDelimiter("BTC", "260410-72000-P", "-"),
					},
					{
						TimeInForce:     order.ImmediateOrCancel,
						Price:           1300,
						Amount:          0.01,
						RemainingAmount: 0.01,
						CostAsset:       currency.USDT,
						Exchange:        "Binance",
						OrderID:         "4611875134427365383",
						ClientOrderID:   "gct-options-cancelled",
						Type:            order.Limit,
						Side:            order.Sell,
						Status:          order.Cancelled,
						AssetType:       asset.Options,
						Date:            time.Unix(1744121800, 0),
						LastUpdated:     time.Unix(1744121900, 0),
						Pair:            currency.NewPairWithDelimiter("BTC", "260410-72000-P", "-"),
					},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				tc.req.Pairs = currency.Pairs{assetToTradablePairMap[tc.req.AssetType]}
				if tc.req.Type == order.UnknownType {
					tc.req.Type = order.AnyType
				}
				tc.req.Side = order.AnySide
				resp, err := e.GetOrderHistory(mockContext(t), tc.req)
				require.NoError(t, err, "GetOrderHistory must not error")
				if mockTests {
					assert.Equal(t, tc.exp, resp, "GetOrderHistory should return the orders")
				}
			})
		}
	})

	t.Run("from an order ID", func(t *testing.T) {
		t.Parallel()
		// Binance lets a window override the order ID, or refuses the two together, so a request from an order ID carries no
		// window, even one longer than the endpoints take
		var mu sync.Mutex
		var queries []map[string][]string
		ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			queries = append(queries, r.URL.Query())
			mu.Unlock()
			_, err := w.Write([]byte(`[]`))
			assert.NoError(t, err, "Write should not error")
		})
		end := time.Now()
		// The cases share ex and its recorded queries in order, so each also checks that a request keeps nothing from the last
		for _, tc := range []struct {
			a         asset.Item
			orderType order.Type
			pair      currency.Pair
			idParam   string
		}{
			{asset.Spot, order.Limit, currency.NewBTCUSDT(), "orderId"},
			{asset.Spot, order.OCO, currency.NewBTCUSDT(), "fromId"},
			{asset.Margin, order.Limit, currency.NewBTCUSDT(), "orderId"},
			{asset.Margin, order.OCO, currency.NewBTCUSDT(), "fromId"},
			{asset.USDTMarginedFutures, order.Limit, currency.NewBTCUSDT(), "orderId"},
			{asset.CoinMarginedFutures, order.Limit, currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), "orderId"},
			{asset.Options, order.Limit, currency.NewPairWithDelimiter("BTC", "270924-90000-C", currency.DashDelimiter), "orderId"},
		} {
			t.Run(tc.a.String()+" "+tc.orderType.String(), func(t *testing.T) {
				mu.Lock()
				queries = nil
				mu.Unlock()
				_, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
					AssetType: tc.a, Type: tc.orderType, Side: order.AnySide, Pairs: currency.Pairs{tc.pair},
					FromOrderID: "5", StartTime: end.AddDate(0, 0, -30), EndTime: end,
				})
				require.NoErrorf(t, err, "GetOrderHistory must not error for %s %s orders from an order ID", tc.a, tc.orderType)
				mu.Lock()
				defer mu.Unlock()
				require.Lenf(t, queries, 1, "GetOrderHistory must send one request for %s %s orders from an order ID", tc.a, tc.orderType)
				assert.Equalf(t, []string{"5"}, queries[0][tc.idParam], "GetOrderHistory should send the %s %s order ID", tc.a, tc.orderType)
				assert.NotContainsf(t, queries[0], "startTime", "GetOrderHistory should send no start time with the %s %s order ID", tc.a, tc.orderType)
				assert.NotContainsf(t, queries[0], "endTime", "GetOrderHistory should send no end time with the %s %s order ID", tc.a, tc.orderType)
			})
		}
	})

	// FromOrderID applies to the kind of order it names: an algo order ID to algo orders, which it starts from, and any
	// other ID to orders, while the other kind is listed by the window
	t.Run("from an algo order ID", func(t *testing.T) {
		t.Parallel()
		end := time.Now()
		for _, m := range []struct {
			a                   asset.Item
			pair                currency.Pair
			orderPath, algoPath string
		}{
			{asset.USDTMarginedFutures, currency.NewBTCUSDT(), "/fapi/v1/allOrders", "/fapi/v1/allAlgoOrders"},
			{asset.CoinMarginedFutures, testCoinMarginedFuturesPair, "/dapi/v1/allOrders", "/dapi/v1/allAlgoOrders"},
		} {
			for _, tc := range []struct {
				fromID                  string
				orderParams, algoParams map[string]string
				orderWindow, algoWindow bool
			}{
				{fromID: "algo-5", orderParams: map[string]string{"orderId": ""}, algoParams: map[string]string{"algoId": "5"}, orderWindow: true},
				{fromID: "5", orderParams: map[string]string{"orderId": "5"}, algoParams: map[string]string{"algoId": ""}, algoWindow: true},
			} {
				t.Run(m.a.String()+" from "+tc.fromID, func(t *testing.T) {
					t.Parallel()
					var mu sync.Mutex
					queries := make(map[string]map[string][]string)
					ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
						mu.Lock()
						queries[r.URL.Path] = r.URL.Query()
						mu.Unlock()
						_, err := w.Write([]byte(`[]`))
						assert.NoError(t, err, "Write should not error")
					})
					_, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
						AssetType: m.a, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{m.pair},
						FromOrderID: tc.fromID, StartTime: end.Add(-time.Hour), EndTime: end,
					})
					require.NoError(t, err, "GetOrderHistory must not error")
					mu.Lock()
					defer mu.Unlock()
					// The cases read queries under the lock held above, so they run in order
					for path, exp := range map[string]struct {
						params map[string]string
						window bool
					}{m.orderPath: {tc.orderParams, tc.orderWindow}, m.algoPath: {tc.algoParams, tc.algoWindow}} {
						t.Run(subtestName(path), func(t *testing.T) {
							q, ok := queries[path]
							require.Truef(t, ok, "GetOrderHistory must request %s", path)
							for k, v := range exp.params {
								if v == "" {
									assert.NotContainsf(t, q, k, "GetOrderHistory should not send %s to %s", k, path)
								} else {
									assert.Equalf(t, []string{v}, q[k], "GetOrderHistory should send %s %q to %s", k, v, path)
								}
							}
							_, hasWindow := q["startTime"]
							assert.Equalf(t, exp.window, hasWindow, "GetOrderHistory should send %s a window only when the ID is of the other kind", path)
						})
					}
				})
			}
		}
		_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{currency.NewBTCUSDT()}, FromOrderID: "algo-x"})
		assert.ErrorIs(t, err, errInvalidOrderID, "GetOrderHistory should reject an invalid algo order ID")
		_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{currency.NewBTCUSDT()}, FromOrderID: "algo-5"})
		assert.ErrorIs(t, err, errInvalidOrderID, "GetOrderHistory should reject an algo order ID on spot")
	})
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// -----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject a nil submission")
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		err    error
	}{
		{"unsupported asset", &order.Submit{AssetType: asset.Futures, Type: order.Market, Side: order.Buy, Amount: 1}, asset.ErrNotSupported},
		{"leverage", &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Buy, Amount: 1, Leverage: 2}, order.ErrSubmitLeverageNotSupported},
		{"spot trailing stop", &order.Submit{AssetType: asset.Spot, Type: order.TrailingStop, Side: order.Sell, Amount: 1}, order.ErrUnsupportedOrderType},
		{"spot stop limit without price", &order.Submit{AssetType: asset.Spot, Type: order.StopLimit, Side: order.Sell, TriggerPrice: 1, Amount: 1}, order.ErrPriceMustBeSetIfLimitOrder},
		{"spot stop without trigger price", &order.Submit{AssetType: asset.Spot, Type: order.Stop, Side: order.Sell, Amount: 1}, errTriggerPriceRequired},
		{"spot good till crossing", &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.GoodTillCrossing}, order.ErrUnsupportedTimeInForce},
		{"spot limit quote amount", &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 1, QuoteAmount: 100}, order.ErrAmountIsInvalid},
		{"spot OCO without price", &order.Submit{AssetType: asset.Spot, Type: order.OCO, Side: order.Sell, TriggerPrice: 1, Amount: 1}, order.ErrPriceMustBeSetIfLimitOrder},
		{"margin OCO without trigger price", &order.Submit{AssetType: asset.Margin, Type: order.OCO, Side: order.Sell, Price: 1, Amount: 1}, errTriggerPriceRequired},
		{"margin unsupported type", &order.Submit{AssetType: asset.Margin, Type: order.IOS, Side: order.Sell, Price: 1, Amount: 1}, order.ErrUnsupportedOrderType},
		{"spot limit maker immediate or cancel", &order.Submit{AssetType: asset.Spot, Type: order.LimitMaker, Side: order.Sell, Price: 1, Amount: 1, TimeInForce: order.ImmediateOrCancel}, order.ErrUnsupportedTimeInForce},
		{"margin stop loss good till day", &order.Submit{AssetType: asset.Margin, Type: order.Stop, Side: order.Sell, TriggerPrice: 1, Amount: 1, TimeInForce: order.GoodTillDay}, order.ErrUnsupportedTimeInForce},
		{"spot OCO fill or kill without a stop limit order", &order.Submit{AssetType: asset.Spot, Type: order.OCO, Side: order.Sell, Price: 2, TriggerPrice: 1, Amount: 1, TimeInForce: order.FillOrKill}, order.ErrUnsupportedTimeInForce},
		{"margin OCO good till day without a stop limit order", &order.Submit{AssetType: asset.Margin, Type: order.OCO, Side: order.Sell, Price: 2, TriggerPrice: 1, Amount: 1, TimeInForce: order.GoodTillDay}, order.ErrUnsupportedTimeInForce},
		{"USDⓈ-M unsupported type", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.IOS, Side: order.Sell, Amount: 1}, order.ErrUnsupportedOrderType},
		{"USDⓈ-M quote amount", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Market, Side: order.Buy, QuoteAmount: 100}, order.ErrAmountIsInvalid},
		{"USDⓈ-M stop limit without price", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.StopLimit, Side: order.Sell, TriggerPrice: 1, Amount: 1}, order.ErrPriceMustBeSetIfLimitOrder},
		{"USDⓈ-M good till day without end time", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Sell, Price: 1, Amount: 1, TimeInForce: order.GoodTillDay}, errGoodTillDateRequired},
		{"USDⓈ-M stop market without trigger price", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.StopMarket, Side: order.Sell, Amount: 1}, errTriggerPriceRequired},
		{"USDⓈ-M trailing stop distance", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.TrailingStop, Side: order.Sell, Amount: 1, TrackingMode: order.Distance, TrackingValue: 100}, errTrailingStopPercentageRequired},
		{"USDⓈ-M index price trigger", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.StopMarket, Side: order.Sell, Amount: 1, TriggerPrice: 1, TriggerPriceType: order.IndexPrice}, order.ErrUnknownPriceType},
		{"COIN-M unsupported type", &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.IOS, Side: order.Sell, Amount: 1}, order.ErrUnsupportedOrderType},
		{"COIN-M stop market without trigger price", &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.StopMarket, Side: order.Sell, Amount: 1}, errTriggerPriceRequired},
		{"COIN-M trailing stop distance", &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.TrailingStop, Side: order.Sell, Amount: 1, TrackingMode: order.Distance, TrackingValue: 100}, errTrailingStopPercentageRequired},
		{"COIN-M good till day", &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.Limit, Side: order.Sell, Price: 1, Amount: 1, TimeInForce: order.GoodTillDay}, order.ErrUnsupportedTimeInForce},
		{"options market", &order.Submit{AssetType: asset.Options, Type: order.Market, Side: order.Buy, Amount: 1}, order.ErrUnsupportedOrderType},
		{"options good till day", &order.Submit{AssetType: asset.Options, Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.GoodTillDay}, order.ErrUnsupportedTimeInForce},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.submit.Exchange = e.Name
			tc.submit.Pair = currency.NewBTCUSDT()
			_, err := e.SubmitOrder(t.Context(), tc.submit)
			assert.ErrorIs(t, err, tc.err, "SubmitOrder should reject the submission")
		})
	}

	t.Run("places orders", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		ensureTradablePairs(t)
		for _, tc := range []struct {
			name   string
			submit *order.Submit
			exp    *order.SubmitResponse
		}{
			{
				name:   "spot limit",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 0.001, ClientOrderID: "gct-spot-limit"},
				exp: &order.SubmitResponse{
					Exchange:             "Binance",
					Type:                 order.Limit,
					Side:                 order.Buy,
					Pair:                 currency.NewBTCUSDT(),
					AssetType:            asset.Spot,
					Price:                80000,
					Amount:               0.001,
					RemainingAmount:      0.0005,
					ClientOrderID:        "gct-spot-limit",
					AverageExecutedPrice: 79990,
					LastUpdated:          time.Unix(1791284812, 0),
					Date:                 time.Unix(1791284812, 0),
					Status:               order.PartiallyFilled,
					OrderID:              "41",
					Trades: []order.TradeHistory{
						{
							Price:     79990,
							Amount:    0.0005,
							Fee:       0.0000005,
							Exchange:  "Binance",
							TID:       "6739441160",
							Type:      order.Limit,
							Side:      order.Buy,
							Timestamp: time.Unix(1791284812, 0),
							FeeAsset:  "BTC",
						},
					},
					Cost: 39.995,
				},
			},
			{
				name:   "spot market quote amount",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Buy, QuoteAmount: 100},
				exp: &order.SubmitResponse{
					Exchange:             "Binance",
					Type:                 order.Market,
					Side:                 order.Buy,
					Pair:                 currency.NewBTCUSDT(),
					AssetType:            asset.Spot,
					QuoteAmount:          100,
					ClientOrderID:        "x0RTQ8DxnmQpYfFZ1zG6aP",
					AverageExecutedPrice: 86190,
					LastUpdated:          time.UnixMilli(1791284812100),
					Date:                 time.UnixMilli(1791284812100),
					Status:               order.Filled,
					OrderID:              "42",
					Trades: []order.TradeHistory{
						{
							Price:     86190,
							Amount:    0.00058,
							Fee:       0.00000058,
							Exchange:  "Binance",
							TID:       "6739441161",
							Type:      order.Market,
							Side:      order.Buy,
							Timestamp: time.UnixMilli(1791284812100),
							FeeAsset:  "BTC",
						},
						{
							Price:     86190,
							Amount:    0.00058,
							Fee:       0.00000058,
							Exchange:  "Binance",
							TID:       "6739441162",
							Type:      order.Market,
							Side:      order.Buy,
							Timestamp: time.UnixMilli(1791284812100),
							FeeAsset:  "BTC",
						},
					},
					Cost: 99.9804,
				},
			},
			{
				name:   "spot post only",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Sell, Price: 90000, Amount: 0.001, TimeInForce: order.PostOnly},
				exp: &order.SubmitResponse{
					Exchange:        "Binance",
					Type:            order.LimitMaker,
					Side:            order.Sell,
					Pair:            currency.NewBTCUSDT(),
					AssetType:       asset.Spot,
					TimeInForce:     order.PostOnly,
					Price:           90000,
					Amount:          0.001,
					RemainingAmount: 0.001,
					ClientOrderID:   "pWbZoydWdBoBHkGxnDqeHq",
					LastUpdated:     time.UnixMilli(1791284812200),
					Date:            time.UnixMilli(1791284812200),
					Status:          order.New,
					OrderID:         "43",
				},
			},
			{
				name:   "spot stop limit",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.StopLimit, Side: order.Sell, Price: 70000, TriggerPrice: 71000, Amount: 0.001},
				exp: &order.SubmitResponse{
					Exchange:        "Binance",
					Type:            order.StopLimit,
					Side:            order.Sell,
					Pair:            currency.NewBTCUSDT(),
					AssetType:       asset.Spot,
					Price:           70000,
					Amount:          0.001,
					RemainingAmount: 0.001,
					TriggerPrice:    71000,
					ClientOrderID:   "Kvqn4AIjCk1bh2Ec9jT5gA",
					LastUpdated:     time.UnixMilli(1791284812300),
					Date:            time.UnixMilli(1791284812300),
					Status:          order.New,
					OrderID:         "44",
				},
			},
			{
				name:   "spot stop with a price",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.Stop, Side: order.Sell, Price: 70000, TriggerPrice: 71000, Amount: 0.001},
				exp: &order.SubmitResponse{
					Exchange:        "Binance",
					Type:            order.StopLimit,
					Side:            order.Sell,
					Pair:            currency.NewBTCUSDT(),
					AssetType:       asset.Spot,
					Price:           70000,
					Amount:          0.001,
					RemainingAmount: 0.001,
					TriggerPrice:    71000,
					ClientOrderID:   "Kvqn4AIjCk1bh2Ec9jT5gA",
					LastUpdated:     time.UnixMilli(1791284812300),
					Date:            time.UnixMilli(1791284812300),
					Status:          order.New,
					OrderID:         "44",
				},
			},
			{
				name: "spot sell OCO",
				submit: &order.Submit{
					AssetType: asset.Spot, Type: order.OCO, Side: order.Sell, Price: 90000, TriggerPrice: 70000, Amount: 0.001, ClientOrderID: "gct-oco",
					RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{LimitPrice: 69000}},
				},
				exp: &order.SubmitResponse{
					Exchange:      "Binance",
					Type:          order.OCO,
					Side:          order.Sell,
					Pair:          currency.NewBTCUSDT(),
					AssetType:     asset.Spot,
					Price:         90000,
					Amount:        0.001,
					TriggerPrice:  70000,
					ClientOrderID: "gct-oco",
					LastUpdated:   time.Unix(1791284813, 0),
					Date:          time.Unix(1791284813, 0),
					Status:        order.Active,
					OrderID:       "1930",
				},
			},
			{
				name:   "spot buy OCO",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.OCO, Side: order.Buy, Price: 60000, TriggerPrice: 95000, Amount: 0.002},
				exp: &order.SubmitResponse{
					Exchange:      "Binance",
					Type:          order.OCO,
					Side:          order.Buy,
					Pair:          currency.NewBTCUSDT(),
					AssetType:     asset.Spot,
					Price:         60000,
					Amount:        0.002,
					TriggerPrice:  95000,
					ClientOrderID: "Ci4nT6aNYQ9xOq4VBxJcVS",
					LastUpdated:   time.UnixMilli(1791284813100),
					Date:          time.UnixMilli(1791284813100),
					Status:        order.Active,
					OrderID:       "1931",
				},
			},
			{
				name:   "margin isolated borrowing limit",
				submit: &order.Submit{AssetType: asset.Margin, Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 0.001, ClientOrderID: "gct-margin-limit", MarginType: margin.Isolated, AutoBorrow: true},
				exp: &order.SubmitResponse{
					Exchange:             "Binance",
					Type:                 order.Limit,
					Side:                 order.Buy,
					Pair:                 currency.NewBTCUSDT(),
					AssetType:            asset.Margin,
					Price:                80000,
					Amount:               0.001,
					ClientOrderID:        "gct-margin-limit",
					AverageExecutedPrice: 80000,
					LastUpdated:          time.Unix(1791284814, 0),
					Date:                 time.Unix(1791284814, 0),
					Status:               order.Filled,
					OrderID:              "28457",
					Trades: []order.TradeHistory{
						{
							Price:     80000,
							Amount:    0.001,
							Fee:       0.08,
							Exchange:  "Binance",
							TID:       "39882",
							Type:      order.Limit,
							Side:      order.Buy,
							Timestamp: time.Unix(1791284814, 0),
							FeeAsset:  "USDT",
						},
					},
					Cost:       80,
					BorrowSize: 50,
					MarginType: margin.Isolated,
				},
			},
			{
				name:   "margin repaying market",
				submit: &order.Submit{AssetType: asset.Margin, Type: order.Market, Side: order.Sell, Amount: 0.001, AutoRepay: true},
				exp: &order.SubmitResponse{
					Exchange:             "Binance",
					Type:                 order.Market,
					Side:                 order.Sell,
					Pair:                 currency.NewBTCUSDT(),
					AssetType:            asset.Margin,
					Amount:               0.001,
					ClientOrderID:        "dHkB4fdq7Lcz9Xm1PoRtUw",
					AverageExecutedPrice: 86100,
					LastUpdated:          time.UnixMilli(1791284814100),
					Date:                 time.UnixMilli(1791284814100),
					Status:               order.Filled,
					OrderID:              "28458",
					Trades: []order.TradeHistory{
						{
							Price:     86100,
							Amount:    0.001,
							Fee:       0.0861,
							Exchange:  "Binance",
							TID:       "39883",
							Type:      order.Market,
							Side:      order.Sell,
							Timestamp: time.UnixMilli(1791284814100),
							FeeAsset:  "USDT",
						},
					},
					Cost: 86.1,
				},
			},
			{
				name: "margin OCO",
				submit: &order.Submit{
					AssetType: asset.Margin, Type: order.OCO, Side: order.Sell, Price: 90000, TriggerPrice: 70000, Amount: 0.001, ClientOrderID: "gct-margin-oco",
					AutoBorrow: true, AutoRepay: true, RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{LimitPrice: 69000}},
				},
				exp: &order.SubmitResponse{
					Exchange:      "Binance",
					Type:          order.OCO,
					Side:          order.Sell,
					Pair:          currency.NewBTCUSDT(),
					AssetType:     asset.Margin,
					Price:         90000,
					Amount:        0.001,
					TriggerPrice:  70000,
					ClientOrderID: "gct-margin-oco",
					LastUpdated:   time.Unix(1791284815, 0),
					Date:          time.Unix(1791284815, 0),
					Status:        order.Active,
					OrderID:       "1001",
					BorrowSize:    0.001,
				},
			},
			{
				name:   "USDⓈ-M post only limit",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 0.002, ClientOrderID: "gct-um-limit", TimeInForce: order.PostOnly},
				exp: &order.SubmitResponse{
					Exchange:        "Binance",
					Type:            order.Limit,
					Side:            order.Buy,
					Pair:            currency.NewBTCUSDT(),
					AssetType:       asset.USDTMarginedFutures,
					TimeInForce:     order.PostOnly,
					Price:           80000,
					Amount:          0.002,
					RemainingAmount: 0.002,
					ClientOrderID:   "gct-um-limit",
					LastUpdated:     time.Unix(1744118500, 0),
					Date:            time.Unix(1744118500, 0),
					Status:          order.New,
					OrderID:         "22542180",
				},
			},
			{
				name:   "USDⓈ-M hedge mode long",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Market, Side: order.Long, Amount: 0.002},
				exp: &order.SubmitResponse{
					Exchange:      "Binance",
					Type:          order.Market,
					Side:          order.Long,
					Pair:          currency.NewBTCUSDT(),
					AssetType:     asset.USDTMarginedFutures,
					Amount:        0.002,
					ClientOrderID: "Wq5zEdT0b9xS2kVhLm3nPc",
					LastUpdated:   time.Unix(1744118600, 0),
					Date:          time.Unix(1744118600, 0),
					Status:        order.New,
					OrderID:       "22542181",
				},
			},
			{
				name:   "USDⓈ-M hedge mode long close",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Market, Side: order.Short, Amount: 0.002, ReduceOnly: true},
				exp: &order.SubmitResponse{
					Exchange:      "Binance",
					Type:          order.Market,
					Side:          order.Short,
					Pair:          currency.NewBTCUSDT(),
					AssetType:     asset.USDTMarginedFutures,
					ReduceOnly:    true,
					Amount:        0.002,
					ClientOrderID: "Hn4cYpR8vB1sE6kTqZ2xWd",
					LastUpdated:   time.Unix(1744118700, 0),
					Date:          time.Unix(1744118700, 0),
					Status:        order.New,
					OrderID:       "22542182",
				},
			},
			{
				name:   "USDⓈ-M good till date",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Sell, Price: 90000, Amount: 0.002, ReduceOnly: true, TimeInForce: order.GoodTillDay, EndTime: time.UnixMilli(1744300800000)},
				exp: &order.SubmitResponse{
					Exchange:        "Binance",
					Type:            order.Limit,
					Side:            order.Sell,
					Pair:            currency.NewBTCUSDT(),
					AssetType:       asset.USDTMarginedFutures,
					TimeInForce:     order.GoodTillDay,
					ReduceOnly:      true,
					Price:           90000,
					Amount:          0.002,
					RemainingAmount: 0.002,
					ClientOrderID:   "Lm8xQ2wE5rT7yU1iO3pAsD",
					LastUpdated:     time.Unix(1744118800, 0),
					Date:            time.Unix(1744118800, 0),
					Status:          order.New,
					OrderID:         "22542183",
				},
			},
			{
				name:   "USDⓈ-M stop market",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.StopMarket, Side: order.Sell, TriggerPrice: 70000, Amount: 0.002, ReduceOnly: true, ClientOrderID: "gct-um-stop", TriggerPriceType: order.MarkPrice},
				exp: &order.SubmitResponse{
					Exchange:      "Binance",
					Type:          order.StopMarket,
					Side:          order.Sell,
					Pair:          currency.NewBTCUSDT(),
					AssetType:     asset.USDTMarginedFutures,
					ReduceOnly:    true,
					Amount:        0.002,
					TriggerPrice:  70000,
					ClientOrderID: "gct-um-stop",
					LastUpdated:   time.Unix(1750485493, 0),
					Date:          time.Unix(1750485493, 0),
					Status:        order.New,
					OrderID:       "algo-2146761",
				},
			},
			{
				name:   "USDⓈ-M take profit limit",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.TakeProfitLimit, Side: order.Sell, Price: 95000, TriggerPrice: 94000, Amount: 0.002},
				exp: &order.SubmitResponse{
					Exchange:      "Binance",
					Type:          order.TakeProfitLimit,
					Side:          order.Sell,
					Pair:          currency.NewBTCUSDT(),
					AssetType:     asset.USDTMarginedFutures,
					Price:         95000,
					Amount:        0.002,
					TriggerPrice:  94000,
					ClientOrderID: "YtR4eW2qA8sD5fG1hJ7kLz",
					LastUpdated:   time.Unix(1750485494, 0),
					Date:          time.Unix(1750485494, 0),
					Status:        order.New,
					OrderID:       "algo-2146762",
				},
			},
			{
				name:   "USDⓈ-M trailing stop",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.TrailingStop, Side: order.Sell, TriggerPrice: 92000, Amount: 0.002, TrackingMode: order.Percentage, TrackingValue: 1.5},
				exp: &order.SubmitResponse{
					Exchange:      "Binance",
					Type:          order.TrailingStop,
					Side:          order.Sell,
					Pair:          currency.NewBTCUSDT(),
					AssetType:     asset.USDTMarginedFutures,
					Amount:        0.002,
					TriggerPrice:  92000,
					ClientOrderID: "Pq9oI8uY7tR6eW5qA4sD3f",
					LastUpdated:   time.Unix(1750485495, 0),
					Date:          time.Unix(1750485495, 0),
					Status:        order.New,
					OrderID:       "algo-2146763",
				},
			},
			{
				name:   "COIN-M limit",
				submit: &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 1, ClientOrderID: "gct-cm-limit", TimeInForce: order.ImmediateOrCancel},
				exp: &order.SubmitResponse{
					Exchange:        "Binance",
					Type:            order.Limit,
					Side:            order.Buy,
					Pair:            currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
					AssetType:       asset.CoinMarginedFutures,
					TimeInForce:     order.ImmediateOrCancel,
					Price:           80000,
					Amount:          1,
					RemainingAmount: 1,
					ClientOrderID:   "gct-cm-limit",
					LastUpdated:     time.Unix(1744120000, 0),
					Date:            time.Unix(1744120000, 0),
					Status:          order.Expired,
					OrderID:         "1573346960",
				},
			},
			{
				name:   "COIN-M hedge mode short",
				submit: &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.Market, Side: order.Short, Amount: 1},
				exp: &order.SubmitResponse{
					Exchange:      "Binance",
					Type:          order.Market,
					Side:          order.Short,
					Pair:          currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
					AssetType:     asset.CoinMarginedFutures,
					Amount:        1,
					ClientOrderID: "Zx1cV2bN3mA4sD5fG6hJ7k",
					LastUpdated:   time.Unix(1744120100, 0),
					Date:          time.Unix(1744120100, 0),
					Status:        order.New,
					OrderID:       "1573346961",
				},
			},
			{
				name:   "options post only",
				submit: &order.Submit{AssetType: asset.Options, Type: order.Limit, Side: order.Sell, Price: 1300, Amount: 0.01, ClientOrderID: "gct-options-limit", ReduceOnly: true, TimeInForce: order.PostOnly},
				exp: &order.SubmitResponse{
					Exchange:             "Binance",
					Type:                 order.Limit,
					Side:                 order.Sell,
					Pair:                 currency.NewPairWithDelimiter("BTC", "260410-72000-P", "-"),
					AssetType:            asset.Options,
					TimeInForce:          order.PostOnly,
					ReduceOnly:           true,
					Price:                1300,
					Amount:               0.01,
					RemainingAmount:      0.008,
					ClientOrderID:        "gct-options-limit",
					AverageExecutedPrice: 1300,
					LastUpdated:          time.UnixMilli(1744121000500),
					Date:                 time.Unix(1744121000, 0),
					Status:               order.PartiallyFilled,
					OrderID:              "4611875134427365378",
					Fee:                  0.03,
					FeeAsset:             currency.USDT,
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				tc.submit.Exchange = e.Name
				tc.submit.Pair = assetToTradablePairMap[tc.submit.AssetType]
				resp, err := e.SubmitOrder(t.Context(), tc.submit)
				require.NoError(t, err, "SubmitOrder must not error")
				if mockTests {
					assert.Equal(t, tc.exp, resp, "SubmitOrder should return the placed order")
				}
			})
		}

		t.Run("websocket API", func(t *testing.T) {
			t.Parallel()
			ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			// The cases share ex and its connection in order, so each also checks that a request keeps nothing from the last
			for _, tc := range []struct {
				submit *order.Submit
				exp    *order.SubmitResponse
			}{
				{
					submit: &order.Submit{Type: order.Market, Side: order.Buy, Amount: 0.001, ClientOrderID: "gct-market-order"},
					exp: &order.SubmitResponse{
						Exchange:             ex.Name,
						Type:                 order.Market,
						Side:                 order.Buy,
						Pair:                 currency.NewBTCUSDT(),
						AssetType:            asset.Spot,
						Amount:               0.001,
						ClientOrderID:        "gct-market-order",
						AverageExecutedPrice: 23416.1,
						LastUpdated:          time.UnixMilli(1660801715700),
						Date:                 time.UnixMilli(1660801715700),
						Status:               order.Filled,
						OrderID:              "12569099460",
						Trades: []order.TradeHistory{
							{
								Price:     23416.1,
								Amount:    0.001,
								Fee:       0.000001,
								Exchange:  ex.Name,
								TID:       "1650422490",
								Type:      order.Market,
								Side:      order.Buy,
								Timestamp: time.UnixMilli(1660801715700),
								FeeAsset:  "BTC",
							},
						},
						Cost: 23.4161,
					},
				},
				{
					submit: &order.Submit{Type: order.StopLimit, Side: order.Sell, Price: 23416.1, TriggerPrice: 23400, Amount: 0.00847, ClientOrderID: "gct-stop-limit-order"},
					exp: &order.SubmitResponse{
						Exchange:        ex.Name,
						Type:            order.StopLimit,
						Side:            order.Sell,
						Pair:            currency.NewBTCUSDT(),
						AssetType:       asset.Spot,
						Price:           23416.1,
						Amount:          0.00847,
						RemainingAmount: 0.00847,
						TriggerPrice:    23400,
						ClientOrderID:   "gct-stop-limit-order",
						LastUpdated:     time.UnixMilli(1660801715800),
						Date:            time.UnixMilli(1660801715800),
						Status:          order.New,
						OrderID:         "12569099461",
					},
				},
				{
					submit: &order.Submit{
						Type: order.OCO, Side: order.Sell, Price: 23420, TriggerPrice: 23410, Amount: 0.0065, ClientOrderID: "gct-oco-order",
						RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{LimitPrice: 23405}},
					},
					exp: &order.SubmitResponse{
						Exchange:      ex.Name,
						Type:          order.OCO,
						Side:          order.Sell,
						Pair:          currency.NewBTCUSDT(),
						AssetType:     asset.Spot,
						Price:         23420,
						Amount:        0.0065,
						TriggerPrice:  23410,
						ClientOrderID: "gct-oco-order",
						LastUpdated:   time.UnixMilli(1711062760700),
						Date:          time.UnixMilli(1711062760700),
						Status:        order.Active,
						OrderID:       "1274514",
					},
				},
			} {
				t.Run(tc.submit.Type.String(), func(t *testing.T) {
					tc.submit.Exchange = ex.Name
					tc.submit.Pair = currency.NewBTCUSDT()
					tc.submit.AssetType = asset.Spot
					resp, err := ex.SubmitOrder(t.Context(), tc.submit)
					require.NoErrorf(t, err, "SubmitOrder must place the %s order over the WebSocket API", tc.submit.Type)
					if mockTests {
						assert.Equalf(t, tc.exp, resp, "SubmitOrder should return the %s order placed over the WebSocket API", tc.submit.Type)
					}
				})
			}
		})
	})

	// A spot or margin order Binance finishes at placement after executing part of its quantity is reported New, over REST
	// and the WebSocket API alike, as a submit response carries no executed amount and the order manager settles only
	// active orders. Orders finished with nothing or everything executed, orders still working and quote sized market
	// orders keep Binance's status. A USDⓈ-M, COIN-M or options order is reported New once it has executed any of its
	// quantity, filled ones included, as a futures or options reply carries no fills either
	t.Run("finished at placement", func(t *testing.T) {
		t.Parallel()
		placed := time.UnixMilli(1791284980615)
		for _, tc := range []struct {
			name   string
			submit *order.Submit
			reply  string
			exp    *order.SubmitResponse
		}{
			{
				name:   "spot IOC limit part filled and expired",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 100, Amount: 1, TimeInForce: order.ImmediateOrCancel},
				reply:  `{"symbol":"BTCUSDT","orderId":30,"orderListId":-1,"clientOrderId":"ioc-part","transactTime":1791284980615,"price":"100","origQty":"1","executedQty":"0.5","origQuoteOrderQty":"0","cummulativeQuoteQty":"50","status":"EXPIRED","timeInForce":"IOC","type":"LIMIT","side":"BUY","workingTime":1791284980615,"fills":[{"price":"100","qty":"0.5","commission":"0.0005","commissionAsset":"BTC","tradeId":7}],"selfTradePreventionMode":"EXPIRE_MAKER"}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, TimeInForce: order.ImmediateOrCancel, Price: 100, Amount: 1, RemainingAmount: 0.5,
					ClientOrderID: "ioc-part", AverageExecutedPrice: 100, LastUpdated: placed, Date: placed, Status: order.New, OrderID: "30", Cost: 50,
					Trades: []order.TradeHistory{{Price: 100, Amount: 0.5, Fee: 0.0005, TID: "7", Type: order.Limit, Side: order.Buy, Timestamp: placed, FeeAsset: "BTC"}},
				},
			},
			{
				name:   "spot market part filled and expired",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Sell, Amount: 1},
				reply:  `{"symbol":"BTCUSDT","orderId":31,"orderListId":-1,"clientOrderId":"market-part","transactTime":1791284980615,"price":"0","origQty":"1","executedQty":"0.4","origQuoteOrderQty":"0","cummulativeQuoteQty":"40","status":"EXPIRED","timeInForce":"GTC","type":"MARKET","side":"SELL","workingTime":1791284980615,"fills":[{"price":"100","qty":"0.4","commission":"0.04","commissionAsset":"USDT","tradeId":8}],"selfTradePreventionMode":"EXPIRE_MAKER"}`,
				exp: &order.SubmitResponse{
					Type: order.Market, Side: order.Sell, AssetType: asset.Spot, Amount: 1, RemainingAmount: 0.6, ClientOrderID: "market-part",
					AverageExecutedPrice: 100, LastUpdated: placed, Date: placed, Status: order.New, OrderID: "31", Cost: 40,
					Trades: []order.TradeHistory{{Price: 100, Amount: 0.4, Fee: 0.04, TID: "8", Type: order.Market, Side: order.Sell, Timestamp: placed, FeeAsset: "USDT"}},
				},
			},
			{
				name:   "spot quote sized market part filled and expired",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Buy, QuoteAmount: 100},
				reply:  `{"symbol":"BTCUSDT","orderId":32,"orderListId":-1,"clientOrderId":"quote-part","transactTime":1791284980615,"price":"0","origQty":"1","executedQty":"0.5","origQuoteOrderQty":"100","cummulativeQuoteQty":"50","status":"EXPIRED","timeInForce":"GTC","type":"MARKET","side":"BUY","workingTime":1791284980615,"fills":[{"price":"100","qty":"0.5","commission":"0.0005","commissionAsset":"BTC","tradeId":9}],"selfTradePreventionMode":"EXPIRE_MAKER"}`,
				exp: &order.SubmitResponse{
					Type: order.Market, Side: order.Buy, AssetType: asset.Spot, QuoteAmount: 100, RemainingAmount: 0.5, ClientOrderID: "quote-part",
					AverageExecutedPrice: 100, LastUpdated: placed, Date: placed, Status: order.Expired, OrderID: "32", Cost: 50,
					Trades: []order.TradeHistory{{Price: 100, Amount: 0.5, Fee: 0.0005, TID: "9", Type: order.Market, Side: order.Buy, Timestamp: placed, FeeAsset: "BTC"}},
				},
			},
			{
				name:   "spot IOC limit expired unfilled",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 100, Amount: 1, TimeInForce: order.ImmediateOrCancel},
				reply:  `{"symbol":"BTCUSDT","orderId":33,"orderListId":-1,"clientOrderId":"ioc-none","transactTime":1791284980615,"price":"100","origQty":"1","executedQty":"0","origQuoteOrderQty":"0","cummulativeQuoteQty":"0","status":"EXPIRED","timeInForce":"IOC","type":"LIMIT","side":"BUY","workingTime":1791284980615,"fills":[],"selfTradePreventionMode":"EXPIRE_MAKER"}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, TimeInForce: order.ImmediateOrCancel, Price: 100, Amount: 1, RemainingAmount: 1,
					ClientOrderID: "ioc-none", LastUpdated: placed, Date: placed, Status: order.Expired, OrderID: "33",
				},
			},
			{
				name:   "spot limit filled",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 100, Amount: 1},
				reply:  `{"symbol":"BTCUSDT","orderId":34,"orderListId":-1,"clientOrderId":"limit-filled","transactTime":1791284980615,"price":"100","origQty":"1","executedQty":"1","origQuoteOrderQty":"0","cummulativeQuoteQty":"100","status":"FILLED","timeInForce":"GTC","type":"LIMIT","side":"BUY","workingTime":1791284980615,"fills":[{"price":"100","qty":"1","commission":"0.001","commissionAsset":"BTC","tradeId":10}],"selfTradePreventionMode":"EXPIRE_MAKER"}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, Price: 100, Amount: 1, ClientOrderID: "limit-filled", AverageExecutedPrice: 100,
					LastUpdated: placed, Date: placed, Status: order.Filled, OrderID: "34", Cost: 100,
					Trades: []order.TradeHistory{{Price: 100, Amount: 1, Fee: 0.001, TID: "10", Type: order.Limit, Side: order.Buy, Timestamp: placed, FeeAsset: "BTC"}},
				},
			},
			{
				name:   "spot limit part filled and working",
				submit: &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 100, Amount: 1},
				reply:  `{"symbol":"BTCUSDT","orderId":35,"orderListId":-1,"clientOrderId":"limit-part","transactTime":1791284980615,"price":"100","origQty":"1","executedQty":"0.5","origQuoteOrderQty":"0","cummulativeQuoteQty":"50","status":"PARTIALLY_FILLED","timeInForce":"GTC","type":"LIMIT","side":"BUY","workingTime":1791284980615,"fills":[{"price":"100","qty":"0.5","commission":"0.0005","commissionAsset":"BTC","tradeId":11}],"selfTradePreventionMode":"EXPIRE_MAKER"}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, Price: 100, Amount: 1, RemainingAmount: 0.5, ClientOrderID: "limit-part",
					AverageExecutedPrice: 100, LastUpdated: placed, Date: placed, Status: order.PartiallyFilled, OrderID: "35", Cost: 50,
					Trades: []order.TradeHistory{{Price: 100, Amount: 0.5, Fee: 0.0005, TID: "11", Type: order.Limit, Side: order.Buy, Timestamp: placed, FeeAsset: "BTC"}},
				},
			},
			{
				name:   "margin IOC limit part filled and expired",
				submit: &order.Submit{AssetType: asset.Margin, Type: order.Limit, Side: order.Sell, Price: 100, Amount: 1, TimeInForce: order.ImmediateOrCancel},
				reply:  `{"symbol":"BTCUSDT","orderId":36,"clientOrderId":"margin-part","isIsolated":false,"transactTime":1791284980615,"price":"100","origQty":"1","executedQty":"0.25","cummulativeQuoteQty":"25","status":"EXPIRED","timeInForce":"IOC","type":"LIMIT","side":"SELL","selfTradePreventionMode":"EXPIRE_MAKER","fills":[{"price":"100","qty":"0.25","commission":"0.025","commissionAsset":"USDT","tradeId":12}]}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Sell, AssetType: asset.Margin, TimeInForce: order.ImmediateOrCancel, Price: 100, Amount: 1, RemainingAmount: 0.75,
					ClientOrderID: "margin-part", AverageExecutedPrice: 100, LastUpdated: placed, Date: placed, Status: order.New, OrderID: "36", Cost: 25,
					Trades: []order.TradeHistory{{Price: 100, Amount: 0.25, Fee: 0.025, TID: "12", Type: order.Limit, Side: order.Sell, Timestamp: placed, FeeAsset: "USDT"}},
				},
			},
			{
				name:   "USDⓈ-M IOC limit part filled and expired",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Buy, Price: 100, Amount: 1, TimeInForce: order.ImmediateOrCancel},
				reply:  `{"orderId":40,"symbol":"BTCUSDT","status":"EXPIRED","clientOrderId":"um-ioc-part","price":"100","avgPrice":"100","origQty":"1","executedQty":"0.5","cumQty":"0.5","cumQuote":"50","timeInForce":"IOC","type":"LIMIT","reduceOnly":false,"closePosition":false,"side":"BUY","positionSide":"BOTH","stopPrice":"0","workingType":"CONTRACT_PRICE","priceProtect":false,"origType":"LIMIT","priceMatch":"NONE","selfTradePreventionMode":"EXPIRE_MAKER","goodTillDate":0,"updateTime":1791284980615}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Buy, AssetType: asset.USDTMarginedFutures, TimeInForce: order.ImmediateOrCancel, Price: 100, Amount: 1, RemainingAmount: 0.5,
					ClientOrderID: "um-ioc-part", LastUpdated: placed, Date: placed, Status: order.New, OrderID: "40",
				},
			},
			{
				name:   "USDⓈ-M IOC limit filled",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Buy, Price: 100, Amount: 1, TimeInForce: order.ImmediateOrCancel},
				reply:  `{"orderId":41,"symbol":"BTCUSDT","status":"FILLED","clientOrderId":"um-ioc-filled","price":"100","avgPrice":"100","origQty":"1","executedQty":"1","cumQty":"1","cumQuote":"100","timeInForce":"IOC","type":"LIMIT","reduceOnly":false,"closePosition":false,"side":"BUY","positionSide":"BOTH","stopPrice":"0","workingType":"CONTRACT_PRICE","priceProtect":false,"origType":"LIMIT","priceMatch":"NONE","selfTradePreventionMode":"EXPIRE_MAKER","goodTillDate":0,"updateTime":1791284980615}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Buy, AssetType: asset.USDTMarginedFutures, TimeInForce: order.ImmediateOrCancel, Price: 100, Amount: 1,
					ClientOrderID: "um-ioc-filled", LastUpdated: placed, Date: placed, Status: order.New, OrderID: "41",
				},
			},
			{
				name:   "USDⓈ-M IOC limit expired unfilled",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Buy, Price: 100, Amount: 1, TimeInForce: order.ImmediateOrCancel},
				reply:  `{"orderId":42,"symbol":"BTCUSDT","status":"EXPIRED","clientOrderId":"um-ioc-none","price":"100","avgPrice":"0","origQty":"1","executedQty":"0","cumQty":"0","cumQuote":"0","timeInForce":"IOC","type":"LIMIT","reduceOnly":false,"closePosition":false,"side":"BUY","positionSide":"BOTH","stopPrice":"0","workingType":"CONTRACT_PRICE","priceProtect":false,"origType":"LIMIT","priceMatch":"NONE","selfTradePreventionMode":"EXPIRE_MAKER","goodTillDate":0,"updateTime":1791284980615}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Buy, AssetType: asset.USDTMarginedFutures, TimeInForce: order.ImmediateOrCancel, Price: 100, Amount: 1, RemainingAmount: 1,
					ClientOrderID: "um-ioc-none", LastUpdated: placed, Date: placed, Status: order.Expired, OrderID: "42",
				},
			},
			{
				name:   "USDⓈ-M limit part filled and working",
				submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Sell, Price: 100, Amount: 1},
				reply:  `{"orderId":43,"symbol":"BTCUSDT","status":"PARTIALLY_FILLED","clientOrderId":"um-limit-part","price":"100","avgPrice":"100","origQty":"1","executedQty":"0.25","cumQty":"0.25","cumQuote":"25","timeInForce":"GTC","type":"LIMIT","reduceOnly":false,"closePosition":false,"side":"SELL","positionSide":"BOTH","stopPrice":"0","workingType":"CONTRACT_PRICE","priceProtect":false,"origType":"LIMIT","priceMatch":"NONE","selfTradePreventionMode":"EXPIRE_MAKER","goodTillDate":0,"updateTime":1791284980615}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Sell, AssetType: asset.USDTMarginedFutures, Price: 100, Amount: 1, RemainingAmount: 0.75,
					ClientOrderID: "um-limit-part", LastUpdated: placed, Date: placed, Status: order.PartiallyFilled, OrderID: "43",
				},
			},
			{
				name:   "COIN-M market filled",
				submit: &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.Market, Side: order.Sell, Amount: 2},
				reply:  `{"orderId":44,"symbol":"BTCUSD_PERP","pair":"BTCUSD","status":"FILLED","clientOrderId":"cm-market","price":"0","avgPrice":"100","origQty":"2","executedQty":"2","cumQty":"2","cumBase":"2","timeInForce":"GTC","type":"MARKET","reduceOnly":false,"closePosition":false,"side":"SELL","positionSide":"BOTH","stopPrice":"0","workingType":"CONTRACT_PRICE","priceProtect":false,"origType":"MARKET","updateTime":1791284980615}`,
				exp: &order.SubmitResponse{
					Type: order.Market, Side: order.Sell, AssetType: asset.CoinMarginedFutures, Amount: 2,
					ClientOrderID: "cm-market", LastUpdated: placed, Date: placed, Status: order.New, OrderID: "44",
				},
			},
			{
				name:   "COIN-M IOC limit part filled and expired",
				submit: &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.Limit, Side: order.Buy, Price: 100, Amount: 4, TimeInForce: order.ImmediateOrCancel},
				reply:  `{"orderId":45,"symbol":"BTCUSD_PERP","pair":"BTCUSD","status":"EXPIRED","clientOrderId":"cm-ioc-part","price":"100","avgPrice":"100","origQty":"4","executedQty":"1","cumQty":"1","cumBase":"1","timeInForce":"IOC","type":"LIMIT","reduceOnly":false,"closePosition":false,"side":"BUY","positionSide":"BOTH","stopPrice":"0","workingType":"CONTRACT_PRICE","priceProtect":false,"origType":"LIMIT","updateTime":1791284980615}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Buy, AssetType: asset.CoinMarginedFutures, TimeInForce: order.ImmediateOrCancel, Price: 100, Amount: 4, RemainingAmount: 3,
					ClientOrderID: "cm-ioc-part", LastUpdated: placed, Date: placed, Status: order.New, OrderID: "45",
				},
			},
			{
				name:   "options IOC limit filled",
				submit: &order.Submit{AssetType: asset.Options, Type: order.Limit, Side: order.Buy, Price: 100, Amount: 0.01, TimeInForce: order.ImmediateOrCancel},
				reply:  `{"orderId":4611875134427365390,"symbol":"BTC-261030-85000-C","price":"100","quantity":"0.01","executedQty":"0.01","fee":"0.03","side":"BUY","type":"LIMIT","timeInForce":"IOC","reduceOnly":false,"postOnly":false,"createTime":1791284980600,"updateTime":1791284980615,"status":"FILLED","avgPrice":"100","source":"API","clientOrderId":"options-ioc-filled","priceScale":0,"quantityScale":2,"optionSide":"CALL","quoteAsset":"USDT","mmp":false,"selfTradePreventionMode":"EXPIRE_MAKER"}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Buy, AssetType: asset.Options, TimeInForce: order.ImmediateOrCancel, Price: 100, Amount: 0.01, ClientOrderID: "options-ioc-filled",
					AverageExecutedPrice: 100, LastUpdated: placed, Date: time.UnixMilli(1791284980600), Status: order.New, OrderID: "4611875134427365390", Fee: 0.03, FeeAsset: currency.USDT,
				},
			},
			{
				name:   "options IOC limit part filled and cancelled",
				submit: &order.Submit{AssetType: asset.Options, Type: order.Limit, Side: order.Sell, Price: 100, Amount: 0.01, TimeInForce: order.ImmediateOrCancel},
				reply:  `{"orderId":4611875134427365391,"symbol":"BTC-261030-85000-C","price":"100","quantity":"0.01","executedQty":"0.004","fee":"0.012","side":"SELL","type":"LIMIT","timeInForce":"IOC","reduceOnly":false,"postOnly":false,"createTime":1791284980600,"updateTime":1791284980615,"status":"CANCELLED","avgPrice":"100","source":"API","clientOrderId":"options-ioc-part","priceScale":0,"quantityScale":2,"optionSide":"CALL","quoteAsset":"USDT","mmp":false,"selfTradePreventionMode":"EXPIRE_MAKER"}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Sell, AssetType: asset.Options, TimeInForce: order.ImmediateOrCancel, Price: 100, Amount: 0.01, RemainingAmount: 0.006,
					ClientOrderID: "options-ioc-part", AverageExecutedPrice: 100, LastUpdated: placed, Date: time.UnixMilli(1791284980600), Status: order.New,
					OrderID: "4611875134427365391", Fee: 0.012, FeeAsset: currency.USDT,
				},
			},
			{
				name:   "options IOC limit cancelled unfilled",
				submit: &order.Submit{AssetType: asset.Options, Type: order.Limit, Side: order.Buy, Price: 100, Amount: 0.01, TimeInForce: order.ImmediateOrCancel},
				reply:  `{"orderId":4611875134427365392,"symbol":"BTC-261030-85000-C","price":"100","quantity":"0.01","executedQty":"0","fee":"0","side":"BUY","type":"LIMIT","timeInForce":"IOC","reduceOnly":false,"postOnly":false,"createTime":1791284980600,"updateTime":1791284980615,"status":"CANCELLED","avgPrice":"0","source":"API","clientOrderId":"options-ioc-none","priceScale":0,"quantityScale":2,"optionSide":"CALL","quoteAsset":"USDT","mmp":false,"selfTradePreventionMode":"EXPIRE_MAKER"}`,
				exp: &order.SubmitResponse{
					Type: order.Limit, Side: order.Buy, AssetType: asset.Options, TimeInForce: order.ImmediateOrCancel, Price: 100, Amount: 0.01, RemainingAmount: 0.01,
					ClientOrderID: "options-ioc-none", LastUpdated: placed, Date: time.UnixMilli(1791284980600), Status: order.Cancelled, OrderID: "4611875134427365392", FeeAsset: currency.USDT,
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				check := func(t *testing.T, ex *Exchange) {
					t.Helper()
					submit := *tc.submit
					submit.Exchange, submit.Pair = ex.Name, currency.NewBTCUSDT()
					switch submit.AssetType {
					case asset.CoinMarginedFutures:
						submit.Pair = testCoinMarginedFuturesPair
					case asset.Options:
						submit.Pair = testOptionsPair
					}
					resp, err := ex.SubmitOrder(t.Context(), &submit)
					require.NoError(t, err, "SubmitOrder must not error")
					exp := *tc.exp
					exp.Exchange, exp.Pair = ex.Name, submit.Pair
					exp.Trades = slices.Clone(tc.exp.Trades)
					for i := range exp.Trades {
						exp.Trades[i].Exchange = ex.Name
					}
					assert.Equal(t, &exp, resp, "SubmitOrder should report the order Binance placed")
				}
				t.Run("REST", func(t *testing.T) {
					t.Parallel()
					path := "/api/v3/order"
					switch tc.submit.AssetType {
					case asset.Margin:
						path = "/sapi/v1/margin/order"
					case asset.USDTMarginedFutures:
						path = "/fapi/v1/order"
					case asset.CoinMarginedFutures:
						path = "/dapi/v1/order"
					case asset.Options:
						path = "/eapi/v1/order"
					}
					check(t, newRoutedTestExchange(t, map[string]string{path: tc.reply}))
				})
				if tc.submit.AssetType != asset.Spot {
					return
				}
				t.Run("websocket API", func(t *testing.T) {
					t.Parallel()
					ex := newTestExchange(t)
					ex.SkipAuthCheck = true
					ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
					useMockWebsocket(t, ex, ex.spotWebsocketAPIConnectionSetup(ex.Config), func(tb testing.TB, msg []byte, w *gws.Conn) error {
						tb.Helper()
						var req WebsocketAPIRequest
						if err := json.Unmarshal(msg, &req); err != nil {
							return err
						}
						assert.Equal(tb, "order.place", req.Method, "SubmitOrder should place the order")
						return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,"status":200,"result":%s}`, req.ID, tc.reply))
					})
					require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
					ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
					require.True(t, ex.canUseWebsocketAPIForOrders(), "orders must go through the WebSocket API")
					check(t, ex)
				})
			})
		}
	})

	// A post only order is reported post only whichever post only time in force it was submitted with, as every read of
	// the order reports it, while other orders keep theirs
	t.Run("reports post only", func(t *testing.T) {
		t.Parallel()
		ex := newRoutedTestExchange(t, map[string]string{
			"/api/v3/order":         `{"symbol":"BTCUSDT","orderId":40,"orderListId":-1,"clientOrderId":"post-only","transactTime":1791284980615,"origQty":"1","executedQty":"0","cummulativeQuoteQty":"0","status":"NEW","fills":[]}`,
			"/sapi/v1/margin/order": `{"symbol":"BTCUSDT","orderId":44,"clientOrderId":"post-only","isIsolated":false,"transactTime":1791284980615,"origQty":"1","executedQty":"0","cummulativeQuoteQty":"0","status":"NEW","fills":[]}`,
			"/fapi/v1/order":        `{"orderId":41,"symbol":"BTCUSDT","status":"NEW","clientOrderId":"post-only","origQty":"1","executedQty":"0","updateTime":1791284980615}`,
			"/dapi/v1/order":        `{"orderId":42,"symbol":"BTCUSD_PERP","status":"NEW","clientOrderId":"post-only","origQty":"1","executedQty":"0","updateTime":1791284980615}`,
			"/eapi/v1/order":        `{"orderId":43,"symbol":"BTC-260410-72000-P","quantity":"1","executedQty":"0","status":"ACCEPTED","clientOrderId":"post-only","createTime":1791284980615,"updateTime":1791284980615,"quoteAsset":"USDT"}`,
		})
		optionsPair := currency.Pair{Base: currency.BTC, Quote: currency.NewCode("260410-72000-P"), Delimiter: currency.DashDelimiter}
		for _, tc := range []struct {
			name      string
			a         asset.Item
			pair      currency.Pair
			orderType order.Type
			tif       order.TimeInForce
			exp       order.TimeInForce
		}{
			{"spot limit maker good till cancel", asset.Spot, currency.NewBTCUSDT(), order.LimitMaker, order.GoodTillCancel, order.PostOnly},
			{"margin post only good till cancel limit", asset.Margin, currency.NewBTCUSDT(), order.Limit, order.PostOnly | order.GoodTillCancel, order.PostOnly},
			{"spot good till cancel limit", asset.Spot, currency.NewBTCUSDT(), order.Limit, order.GoodTillCancel, order.GoodTillCancel},
			{"USDⓈ-M good till crossing limit", asset.USDTMarginedFutures, currency.NewBTCUSDT(), order.Limit, order.GoodTillCrossing, order.PostOnly},
			{"USDⓈ-M immediate or cancel limit", asset.USDTMarginedFutures, currency.NewBTCUSDT(), order.Limit, order.ImmediateOrCancel, order.ImmediateOrCancel},
			{"COIN-M post only good till cancel limit", asset.CoinMarginedFutures, testCoinMarginedFuturesPair, order.Limit, order.PostOnly | order.GoodTillCancel, order.PostOnly},
			{"options good till crossing limit", asset.Options, optionsPair, order.Limit, order.GoodTillCrossing, order.PostOnly},
			{"options good till cancel limit", asset.Options, optionsPair, order.Limit, order.GoodTillCancel, order.GoodTillCancel},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				resp, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange: ex.Name, Pair: tc.pair, AssetType: tc.a, Type: tc.orderType, Side: order.Sell, Price: 90000, Amount: 1, TimeInForce: tc.tif,
				})
				require.NoError(t, err, "SubmitOrder must not error")
				assert.Equal(t, tc.exp, resp.TimeInForce, "SubmitOrder should report the time in force every read of the order reports")
			})
		}
	})
}

// TestPostOnlyOrdersReadAsPostOnly ensures that order queries read a spot or margin LIMIT_MAKER order, which Binance
// reports as GTC, and a USDⓈ-M GTX order as post only, as SubmitOrder reports them
func TestPostOnlyOrdersReadAsPostOnly(t *testing.T) {
	t.Parallel()
	ex := newRoutedTestExchange(t, map[string]string{
		"/api/v3/order":         `{"symbol":"BTCUSDT","orderId":50,"orderListId":-1,"clientOrderId":"spot-maker","price":"90000","origQty":"0.001","executedQty":"0","cummulativeQuoteQty":"0","status":"NEW","timeInForce":"GTC","type":"LIMIT_MAKER","side":"SELL","stopPrice":"0","time":1791284980615,"updateTime":1791284980615,"isWorking":true,"workingTime":1791284980615,"origQuoteOrderQty":"0","selfTradePreventionMode":"EXPIRE_MAKER"}`,
		"/sapi/v1/margin/order": `{"clientOrderId":"margin-maker","cummulativeQuoteQty":"0","executedQty":"0","icebergQty":"0","isWorking":true,"orderId":51,"origQty":"0.001","price":"90000","side":"SELL","status":"NEW","stopPrice":"0","symbol":"BTCUSDT","isIsolated":false,"time":1791284980615,"timeInForce":"GTC","type":"LIMIT_MAKER","selfTradePreventionMode":"EXPIRE_MAKER","updateTime":1791284980615}`,
		"/fapi/v1/order":        `{"orderId":52,"symbol":"BTCUSDT","status":"NEW","clientOrderId":"usdtm-gtx","price":"90000","avgPrice":"0","origQty":"0.002","executedQty":"0","cumQuote":"0","timeInForce":"GTX","type":"LIMIT","reduceOnly":false,"closePosition":false,"side":"SELL","positionSide":"BOTH","stopPrice":"0","workingType":"CONTRACT_PRICE","priceProtect":false,"origType":"LIMIT","priceMatch":"NONE","selfTradePreventionMode":"EXPIRE_MAKER","goodTillDate":0,"time":1791284980615,"updateTime":1791284980615}`,
		"/fapi/v1/algoOrder":    `{"code":-2013,"msg":"Order does not exist."}`,
	})
	placed := time.UnixMilli(1791284980615)
	pair := currency.NewBTCUSDT()
	for _, tc := range []struct {
		a       asset.Item
		orderID string
		exp     *order.Detail
	}{
		{
			a: asset.Spot, orderID: "50",
			exp: &order.Detail{
				TimeInForce: order.PostOnly, Price: 90000, Amount: 0.001, RemainingAmount: 0.001, Exchange: ex.Name, OrderID: "50", ClientOrderID: "spot-maker",
				Type: order.LimitMaker, Side: order.Sell, Status: order.New, AssetType: asset.Spot, Date: placed, LastUpdated: placed, Pair: pair,
			},
		},
		{
			a: asset.Margin, orderID: "51",
			exp: &order.Detail{
				TimeInForce: order.PostOnly, Price: 90000, Amount: 0.001, RemainingAmount: 0.001, Exchange: ex.Name, OrderID: "51", ClientOrderID: "margin-maker",
				Type: order.LimitMaker, Side: order.Sell, Status: order.New, AssetType: asset.Margin, Date: placed, LastUpdated: placed, Pair: pair, MarginType: margin.Multi,
			},
		},
		{
			a: asset.USDTMarginedFutures, orderID: "52",
			exp: &order.Detail{
				TimeInForce: order.PostOnly, Price: 90000, Amount: 0.002, RemainingAmount: 0.002, Exchange: ex.Name, OrderID: "52", ClientOrderID: "usdtm-gtx",
				Type: order.Limit, Side: order.Sell, Status: order.New, AssetType: asset.USDTMarginedFutures, Date: placed, LastUpdated: placed, Pair: pair,
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			d, err := ex.GetOrderInfo(t.Context(), tc.orderID, pair, tc.a)
			require.NoError(t, err, "GetOrderInfo must not error")
			assert.Equal(t, tc.exp, d, "GetOrderInfo should read the post only order as post only")
		})
	}
}

func TestOptionsListingsReadPostOnly(t *testing.T) {
	t.Parallel()
	// Binance documents postOnly on the placement and single order query replies only, so a post only order listed
	// without it reads as GTC; the listings' PostOnly fields are kept, and read, in case Binance sends the flag
	const listed = `"orderId":4611875134427365377,"symbol":"BTC-260410-72000-P","price":"1300","quantity":"0.01","executedQty":"0","side":"SELL","type":"LIMIT","timeInForce":"GTC","reduceOnly":false,"createTime":1791284980615,"updateTime":1791284980616,"status":"ACCEPTED","avgPrice":"0","clientOrderId":"post-only","priceScale":1,"quantityScale":2,"optionSide":"PUT","quoteAsset":"USDT","mmp":false`
	pair := currency.Pair{Base: currency.BTC, Quote: currency.NewCode("260410-72000-P"), Delimiter: currency.DashDelimiter}
	for _, tc := range []struct {
		name   string
		listed string
		exp    order.TimeInForce
	}{
		{"flagged post only", `{` + listed + `,"postOnly":true}`, order.PostOnly},
		{"without the flag", `{` + listed + `}`, order.GoodTillCancel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newRoutedTestExchange(t, map[string]string{
				"/eapi/v1/openOrders":    "[" + tc.listed + "]",
				"/eapi/v1/historyOrders": "[" + tc.listed + "]",
			})
			exp := order.Detail{
				TimeInForce: tc.exp, Price: 1300, Amount: 0.01, RemainingAmount: 0.01, CostAsset: currency.USDT, Exchange: ex.Name,
				OrderID: "4611875134427365377", ClientOrderID: "post-only", Type: order.Limit, Side: order.Sell, Status: order.New,
				AssetType: asset.Options, Date: time.UnixMilli(1791284980615), LastUpdated: time.UnixMilli(1791284980616), Pair: pair,
			}
			req := &order.MultiOrderRequest{AssetType: asset.Options, Pairs: currency.Pairs{pair}, Type: order.AnyType, Side: order.AnySide}
			active, err := ex.GetActiveOrders(t.Context(), req)
			require.NoError(t, err, "GetActiveOrders must not error")
			assert.Equal(t, order.FilteredOrders{exp}, active, "GetActiveOrders should read the listed order's time in force")
			history, err := ex.GetOrderHistory(t.Context(), req)
			require.NoError(t, err, "GetOrderHistory must not error")
			assert.Equal(t, order.FilteredOrders{exp}, history, "GetOrderHistory should read the listed order's time in force")
		})
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject a nil cancellation")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must reject an unsupported asset")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Margin, MarginType: margin.Isolated})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllOrders must reject isolated margin without a pair")

	t.Run("cancels open orders", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		ensureTradablePairs(t)
		for _, tc := range []struct {
			name      string
			cancel    *order.Cancel
			everyPair bool
			exp       order.CancelAllResponse
		}{
			{
				name:   "spot",
				cancel: &order.Cancel{AssetType: asset.Spot},
				exp: order.CancelAllResponse{
					Status: map[string]string{
						"11": "CANCELED",
						"13": "CANCELED",
						"20": "CANCELED",
						"21": "CANCELED",
					},
				},
			},
			{
				name:      "spot every symbol",
				cancel:    &order.Cancel{AssetType: asset.Spot},
				everyPair: true,
				exp: order.CancelAllResponse{
					Status: map[string]string{
						"11": "CANCELED",
						"13": "CANCELED",
						"20": "CANCELED",
						"21": "CANCELED",
						"4":  "CANCELED",
					},
				},
			},
			{
				name:   "margin isolated",
				cancel: &order.Cancel{AssetType: asset.Margin, MarginType: margin.Isolated},
				exp: order.CancelAllResponse{
					Status: map[string]string{
						"11": "CANCELED",
						"20": "CANCELED",
						"21": "CANCELED",
					},
				},
			},
			{
				name:      "margin every symbol",
				cancel:    &order.Cancel{AssetType: asset.Margin},
				everyPair: true,
				exp: order.CancelAllResponse{
					Status: map[string]string{
						"28459": "CANCELED",
						"28460": "CANCELED",
						"28461": "CANCELED",
					},
				},
			},
			{
				name:   "USDⓈ-M",
				cancel: &order.Cancel{AssetType: asset.USDTMarginedFutures},
				exp: order.CancelAllResponse{
					Status: map[string]string{},
				},
			},
			{
				name:      "USDⓈ-M every symbol",
				cancel:    &order.Cancel{AssetType: asset.USDTMarginedFutures},
				everyPair: true,
				exp: order.CancelAllResponse{
					Status: map[string]string{},
				},
			},
			{
				name:   "COIN-M",
				cancel: &order.Cancel{AssetType: asset.CoinMarginedFutures},
				exp: order.CancelAllResponse{
					Status: map[string]string{},
				},
			},
			{
				name:      "COIN-M every symbol",
				cancel:    &order.Cancel{AssetType: asset.CoinMarginedFutures},
				everyPair: true,
				exp: order.CancelAllResponse{
					Status: map[string]string{},
				},
			},
			{
				name:   "options",
				cancel: &order.Cancel{AssetType: asset.Options},
				exp: order.CancelAllResponse{
					Status: map[string]string{},
				},
			},
			{
				name:      "options every symbol",
				cancel:    &order.Cancel{AssetType: asset.Options},
				everyPair: true,
				exp: order.CancelAllResponse{
					Status: map[string]string{},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				if !tc.everyPair {
					tc.cancel.Pair = assetToTradablePairMap[tc.cancel.AssetType]
				}
				resp, err := e.CancelAllOrders(t.Context(), tc.cancel)
				require.NoError(t, err, "CancelAllOrders must not error")
				if mockTests {
					assert.Equal(t, tc.exp, resp, "CancelAllOrders should return the cancelled orders' statuses")
				}
			})
		}

		t.Run("websocket API", func(t *testing.T) {
			t.Parallel()
			ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			// The open orders of every symbol are BTCUSDT's, so both requests cancel the same orders
			exp := order.CancelAllResponse{Status: map[string]string{"12569099453": "CANCELED", "12569099454": "CANCELED"}}
			// The cases share ex and its connection in order, so each also checks that a request keeps nothing from the last
			for _, pair := range []currency.Pair{currency.NewBTCUSDT(), currency.EMPTYPAIR} {
				t.Run(subtestName(pair.String()), func(t *testing.T) {
					resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: pair})
					require.NoErrorf(t, err, "CancelAllOrders must cancel %q orders over the WebSocket API", pair)
					if mockTests {
						assert.Equalf(t, exp, resp, "CancelAllOrders should return the statuses of %q orders cancelled over the WebSocket API", pair)
					}
				})
			}
		})
	})

	t.Run("without open orders", func(t *testing.T) {
		t.Parallel()
		// Binance rejects cancelling every open order of a symbol that has none
		unknownOrder := `{"code":-2011,"msg":"Unknown order sent."}`
		for _, tc := range []struct {
			name   string
			cancel *order.Cancel
			path   string
		}{
			{"spot", &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT()}, "/api/v3/openOrders"},
			{"cross margin", &order.Cancel{AssetType: asset.Margin, Pair: currency.NewBTCUSDT()}, "/sapi/v1/margin/openOrders"},
			{"isolated margin", &order.Cancel{AssetType: asset.Margin, MarginType: margin.Isolated, Pair: currency.NewBTCUSDT()}, "/sapi/v1/margin/openOrders"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRoutedTestExchange(t, map[string]string{tc.path: unknownOrder})
				resp, err := ex.CancelAllOrders(t.Context(), tc.cancel)
				require.NoError(t, err, "CancelAllOrders must treat a pair without open orders as having nothing to cancel")
				assert.Empty(t, resp.Status, "CancelAllOrders should report no cancelled orders")
			})
		}

		t.Run("websocket API", func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.API.AuthenticatedSupport = true
			ex.API.AuthenticatedWebsocketSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
			var cancelAllRequests atomic.Int32
			useMockWebsocket(t, ex, ex.spotWebsocketAPIConnectionSetup(ex.Config), func(tb testing.TB, msg []byte, w *gws.Conn) error {
				tb.Helper()
				var req struct {
					ID     string `json:"id"`
					Method string `json:"method"`
				}
				require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
				assert.Equal(tb, "openOrders.cancelAll", req.Method, "CancelAllOrders should only cancel open orders")
				cancelAllRequests.Add(1)
				return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,"status":400,"error":{"code":-2011,"msg":"Unknown order sent."}}`, req.ID))
			})
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT()})
			require.NoError(t, err, "CancelAllOrders must treat a pair without open orders as having nothing to cancel over the WebSocket API")
			assert.Empty(t, resp.Status, "CancelAllOrders should report no cancelled orders over the WebSocket API")
			assert.Equal(t, int32(1), cancelAllRequests.Load(), "CancelAllOrders should cancel over the WebSocket API")
		})

		ex := newRoutedTestExchange(t, map[string]string{"/api/v3/openOrders": `{"code":-2011,"msg":"Order was not canceled due to cancel restrictions."}`})
		_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT()})
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "CancelAllOrders must return a cancel restriction")
		assert.Equal(t, "Order was not canceled due to cancel restrictions.", apiErr.Message, "CancelAllOrders should return the cancel restriction")
	})

	t.Run("listing rate", func(t *testing.T) {
		t.Parallel()
		// Order book snapshots can hold a host's IP pool for seconds, so the listing a cancellation of every symbol's orders
		// starts with draws on the host's cancel pool, as the cancellations do, while it can take the listing at once, and
		// otherwise waits on whichever of the two pools admits it sooner
		replies := map[string]string{
			"GET /api/v3/openOrders":         `[{"symbol":"BTCUSDT","orderId":11}]`,
			"DELETE /api/v3/openOrders":      `[{"symbol":"BTCUSDT","orderId":11,"status":"CANCELED"}]`,
			"GET /fapi/v1/openOrders":        `[{"symbol":"BTCUSDT","orderId":12}]`,
			"GET /fapi/v1/openAlgoOrders":    `[{"symbol":"BTCUSDT","algoId":13}]`,
			"DELETE /fapi/v1/allOpenOrders":  `{"code":200,"msg":"done"}`,
			"DELETE /fapi/v1/algoOpenOrders": `{"code":200,"msg":"done"}`,
			"GET /dapi/v1/openOrders":        `[{"symbol":"BTCUSD_PERP","orderId":14}]`,
			"GET /dapi/v1/openAlgoOrders":    `[{"symbol":"BTCUSD_PERP","algoId":15}]`,
			"DELETE /dapi/v1/allOpenOrders":  `{"code":200,"msg":"done"}`,
			"DELETE /dapi/v1/algoOpenOrders": `{"code":200,"msg":"done"}`,
			"GET /eapi/v1/openOrders":        `[{"symbol":"BTC-260410-72000-P","orderId":16}]`,
			"DELETE /eapi/v1/allOpenOrders":  `{"code":0,"msg":"success"}`,
		}
		exhausted := request.NewRateLimitWithWeight(time.Hour, 1, 1)
		require.NoError(t, exhausted.RateLimit(t.Context()), "RateLimit must take the limiter's only token")
		shares := []struct {
			name string
			epl  request.EndpointLimit
		}{
			{"spot listing", spotCancelAllListingRate},
			{"USDⓈ-M listing", usdtMarginedFuturesCancelAllListingRate},
			{"COIN-M listing", coinMarginedFuturesCancelAllListingRate},
			{"options listing", optionsCancelAllListingRate},
		}
		overflows := []request.EndpointLimit{spotOpenOrdersAllRate, usdtMarginedFuturesGetAllOpenOrdersRate, usdtMarginedFuturesOpenAlgoOrdersAllRate, coinMarginedFuturesGetAllOpenOrdersRate, coinMarginedFuturesOpenAlgoOrdersAllRate, optionsAllQueryOpenOrdersRate}
		for _, tc := range []struct {
			name  string
			limit func(*testing.T, request.RateLimitDefinitions)
		}{
			{"IP pools held", func(_ *testing.T, definitions request.RateLimitDefinitions) {
				for epl, cost := range endpointCosts {
					if cost.pool == spotIPPool || cost.pool == derivativesIPPool || cost.pool == optionsIPPool {
						definitions[epl] = exhausted
					}
				}
			}},
			// The IP pools refill evenly with a burst of one, so here the overflow listings are given limiters that take them
			// at once
			{"cancel pools spent", func(_ *testing.T, definitions request.RateLimitDefinitions) {
				for _, share := range shares {
					definitions[share.epl] = exhausted
				}
				for _, epl := range overflows {
					definitions[epl] = request.GetRateLimiterWithWeight(rate.NewLimiter(rate.Inf, 1), 1)
				}
			}},
			{"cancel pools a token short", func(t *testing.T, definitions request.RateLimitDefinitions) {
				t.Helper()
				// The cases write the exchange's rate limit definitions, so they run in order
				for _, share := range shares {
					t.Run(share.name, func(t *testing.T) {
						weight := definitions[share.epl].Weight()
						limiter := rate.NewLimiter(1000, int(weight))
						require.True(t, limiter.AllowN(time.Now(), 1), "AllowN must take a token")
						definitions[share.epl] = request.GetRateLimiterWithWeight(limiter, weight)
					})
				}
				for _, epl := range overflows {
					definitions[epl] = exhausted
				}
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				sent := make(map[string]int)
				ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					reply, ok := replies[r.Method+" "+r.URL.Path]
					if !assert.Truef(t, ok, "%s %s should not be requested", r.Method, r.URL.Path) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					mu.Lock()
					sent[r.Method+" "+r.URL.Path]++
					mu.Unlock()
					_, err := w.Write([]byte(reply))
					assert.NoError(t, err, "Write should not error")
				})
				require.NoError(t, ex.EnableRateLimiter(), "EnableRateLimiter must not error")
				tc.limit(t, ex.Requester.GetRateLimiterDefinitions())
				// A pool held for an hour exceeds the deadline, so a listing waiting on it fails
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				// The cases share ex, whose requests the check below counts, so they run in order before it
				for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
					t.Run(a.String(), func(t *testing.T) {
						_, err := ex.CancelAllOrders(ctx, &order.Cancel{AssetType: a})
						assert.NoErrorf(t, err, "CancelAllOrders should list and cancel %s orders on the pool that admits them sooner", a)
					})
				}
				exp := make(map[string]int, len(replies))
				for k := range replies {
					exp[k] = 1
				}
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, exp, sent, "CancelAllOrders should list and cancel every symbol's open orders")
			})
		}
	})

	t.Run("COIN-M algo order symbols", func(t *testing.T) {
		t.Parallel()
		// A cancellation of every COIN-M symbol's orders cancels the algo orders of a symbol with no open orders, and no
		// algo orders of a symbol it has none of
		const allCancelled = `{"code":200,"msg":"The operation of cancel all open order is done."}`
		ex, sent := newCancelStepsExchange(t, []cancelStep{
			{http.MethodGet, "/dapi/v1/openOrders", map[string]string{"symbol": ""}, `[{"symbol":"BTCUSD_PERP","orderId":1573346962}]`},
			{http.MethodGet, "/dapi/v1/openAlgoOrders", map[string]string{"symbol": ""}, `[{"symbol":"ETHUSD_PERP","algoId":3100001}]`},
			{http.MethodDelete, "/dapi/v1/allOpenOrders", map[string]string{"symbol": "BTCUSD_PERP"}, allCancelled},
			{http.MethodDelete, "/dapi/v1/algoOpenOrders", map[string]string{"symbol": "ETHUSD_PERP"}, allCancelled},
		})
		_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.CoinMarginedFutures})
		require.NoError(t, err, "CancelAllOrders must cancel every symbol's orders and algo orders")
		assert.Equal(t, 4, sent(), "CancelAllOrders should send every expected request")
	})
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	t.Run("REST snapshot ordering", func(t *testing.T) {
		t.Parallel()
		testRESTBalanceSnapshotOrdering(t, false)
	})
	t.Run("keeps streamed balance", func(t *testing.T) {
		t.Parallel()
		// A REST poll's snapshot from before a stream event, however late the poll stores it, leaves the event's balance
		// stored, while a snapshot of the event or later is stored as REST reports it
		for _, tc := range balanceEvents {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				var current atomic.Bool
				ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
						return
					}
					reply := tc.flatReply
					if current.Load() {
						reply = balanceRefreshReplies[tc.path]
					}
					_, err := w.Write([]byte(reply))
					assert.NoError(t, err, "Write should not error")
				})
				stored := accounts.NewSubAccount(tc.a, "")
				stored.Balances.Set(tc.code, accounts.Balance{Total: tc.restTotal + 1, Free: tc.restTotal + 1})
				require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{stored}, true), "Save must not error")
				require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.eventTotal)), "processing the balance event must not error")
				streamed := storedBalance(t, ex, tc.a, tc.code)
				resp, err := ex.UpdateAccountBalances(t.Context(), tc.a)
				require.NoError(t, err, "UpdateAccountBalances must not error")
				require.Len(t, resp, 1, "UpdateAccountBalances must return the asset's balances")
				assert.Equal(t, streamed, resp[0].Balances[tc.code], "the poll should report the stream event's balance")
				assert.Equal(t, streamed, storedBalance(t, ex, tc.a, tc.code), "a poll older than the stream event should leave its balance stored")

				// The user data streams are the configured account's, so another account's poll stores its snapshot
				other := &accounts.Credentials{Key: "other-key", Secret: "other-secret"}
				otherCtx := accounts.DeployCredentialsToContext(t.Context(), other)
				otherStored := accounts.NewSubAccount(tc.a, "")
				otherStored.Balances.Set(tc.code, accounts.Balance{Total: 7, Free: 7})
				require.NoError(t, ex.Accounts.Save(otherCtx, accounts.SubAccounts{otherStored}, true), "Save must not error for another account")
				_, err = ex.UpdateAccountBalances(otherCtx, tc.a)
				require.NoError(t, err, "UpdateAccountBalances must not error for another account")
				otherBalance, err := ex.Accounts.GetBalance("", other, tc.a, tc.code)
				require.NoError(t, err, "GetBalance must not error for another account")
				assert.Equal(t, tc.restTotal, otherBalance.Total, "another account's poll should store the wallet balance REST reports")
				assert.Equal(t, tc.restTotal, otherBalance.Free, "another account's poll should store the available balance REST reports")

				current.Store(true)
				_, err = ex.UpdateAccountBalances(t.Context(), tc.a)
				require.NoError(t, err, "UpdateAccountBalances must not error")
				got := storedBalance(t, ex, tc.a, tc.code)
				assert.Equal(t, tc.restTotal, got.Total, "a poll of the stream event should store the wallet balance REST reports")
				assert.Equal(t, tc.restFree, got.Free, "a poll of the stream event should store the available balance REST reports")
				assert.Equal(t, tc.restOnlyTotal, storedBalance(t, ex, tc.a, tc.restOnly).Total, "a poll should store a currency no event reported")
			})
		}
	})

	t.Run("checks under the store lock", func(t *testing.T) {
		t.Parallel()
		// A poll checks its snapshot against the stream's events under the balance store lock, as it stores it, so no event
		// is stored between the check and the store
		for _, tc := range balanceEvents {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				var answered atomic.Bool
				ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					answered.Store(true)
					serveBalanceRefreshReplies(t)(w, r)
				})
				require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.restTotal)), "processing the balance event must not error")
				var lookups atomic.Int32
				ctx := credentialsWatchingContext{Context: t.Context(), onCredentials: func() {
					if !answered.Load() {
						return
					}
					lookups.Add(1)
					if ex.balanceRefresh.storeMu.TryLock() {
						ex.balanceRefresh.storeMu.Unlock()
						assert.Fail(t, "a poll should check and store its snapshot under the balance store lock")
					}
				}}
				_, err := ex.UpdateAccountBalances(ctx, tc.a)
				require.NoError(t, err, "UpdateAccountBalances must not error")
				assert.GreaterOrEqual(t, lookups.Load(), int32(2), "the poll's check and its store should both look the credentials up")
			})
		}
	})

	t.Run("keeps omitted streamed currency", func(t *testing.T) {
		t.Parallel()
		// A snapshot that leaves out a currency the stream reported removes it only when it shows an update at or after the
		// currency's latest event; otherwise it may predate the event, so the event's balance stands
		const eventTime = 1791300000000
		for _, tc := range balanceEvents {
			for _, c := range []struct {
				name       string
				updateTime int64
				kept       bool
			}{
				{name: "snapshot without an update after the event", updateTime: eventTime - 1, kept: true},
				{name: "snapshot with an update at the event", updateTime: eventTime},
				{name: "snapshot with an update after the event", updateTime: eventTime + 1},
			} {
				t.Run(tc.a.String()+" "+c.name, func(t *testing.T) {
					t.Parallel()
					ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
						if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
							return
						}
						_, err := fmt.Fprintf(w, tc.restOnlyReply, c.updateTime)
						assert.NoError(t, err, "Fprintf should not error")
					})
					require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.eventTotal)), "processing the balance event must not error")
					streamed := storedBalance(t, ex, tc.a, tc.code)
					_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
					require.NoError(t, err, "UpdateAccountBalances must not error")
					assert.Equal(t, tc.restOnlyTotal, storedBalance(t, ex, tc.a, tc.restOnly).Total, "the poll should store the currency it reports")
					creds, err := ex.GetCredentials(t.Context())
					require.NoError(t, err, "GetCredentials must not error")
					got, err := ex.Accounts.GetBalance("", creds, tc.a, tc.code)
					if c.kept {
						require.NoError(t, err, "the streamed currency must stay stored")
						assert.Equal(t, streamed, got, "the streamed currency should keep its stored balance")
						return
					}
					assert.ErrorIs(t, err, accounts.ErrNoBalances, "the snapshot should remove the currency it leaves out")
				})
			}
		}
	})

	t.Run("empty snapshot", func(t *testing.T) {
		t.Parallel()
		// An empty snapshot shows no update of its own, so it removes a streamed currency only when it has a time of its own,
		// as options snapshots do, at or after the currency's latest event
		const eventTime = 1791300000000
		for _, tc := range balanceEvents {
			for _, snapshotTime := range []int64{eventTime - 1, eventTime, eventTime + 1} {
				t.Run(fmt.Sprintf("%s %+d ms", tc.a, snapshotTime-eventTime), func(t *testing.T) {
					t.Parallel()
					var reply string
					switch tc.a {
					case asset.USDTMarginedFutures:
						reply = `[]`
					case asset.CoinMarginedFutures:
						reply = `{"assets":[],"positions":[],"canTrade":true}`
					default:
						reply = fmt.Sprintf(`{"asset":[],"time":%d}`, snapshotTime)
					}
					ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
						if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
							return
						}
						_, err := w.Write([]byte(reply))
						assert.NoError(t, err, "Write should not error")
					})
					require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.eventTotal)), "processing the balance event must not error")
					streamed := storedBalance(t, ex, tc.a, tc.code)
					_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
					require.NoError(t, err, "UpdateAccountBalances must not error")
					creds, err := ex.GetCredentials(t.Context())
					require.NoError(t, err, "GetCredentials must not error")
					got, err := ex.Accounts.GetBalance("", creds, tc.a, tc.code)
					if tc.a == asset.Options && snapshotTime >= eventTime {
						assert.ErrorIs(t, err, accounts.ErrNoBalances, "an options snapshot taken at or after the event should remove the currency it leaves out")
						return
					}
					require.NoError(t, err, "the streamed currency must stay stored")
					assert.Equal(t, streamed, got, "the streamed currency should keep its stored balance")
				})
			}
		}
	})
	ex := newMarketTestExchange(t)
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an unsupported asset")
	subAccountCtx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", SubAccount: "sub"})
	_, err = ex.UpdateAccountBalances(subAccountCtx, asset.Spot)
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "UpdateAccountBalances must reject spot sub-accounts")

	// Serial: the cases poll ex, which the checks above set up and use
	t.Run("balances by asset", func(t *testing.T) {
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		}
		for _, tc := range []struct {
			a   asset.Item
			exp accounts.CurrencyBalances
		}{
			{
				a: asset.Spot,
				exp: accounts.CurrencyBalances{
					currency.BTC:  {Currency: currency.BTC, Total: 1.26, Hold: 0.01, Free: 1.25},
					currency.USDT: {Currency: currency.USDT, Total: 86288.63, Hold: 120.5, Free: 86168.13},
				},
			},
			{
				a: asset.Margin,
				exp: accounts.CurrencyBalances{
					currency.BTC: {Currency: currency.BTC, Total: 6.82728457, Hold: 0.01, Free: 6.81728457, AvailableWithoutBorrow: 6.2309524199999995, Borrowed: 0.58633215},
				},
			},
			{
				// The USDⓈ-M account alias is not a sub-account
				a: asset.USDTMarginedFutures,
				exp: accounts.CurrencyBalances{
					currency.USDT: {Currency: currency.USDT, Total: 122607.35137903, Hold: 122583.62668697, Free: 23.72469206},
				},
			},
			{
				a: asset.CoinMarginedFutures,
				exp: accounts.CurrencyBalances{
					currency.BTC: {Currency: currency.BTC, Total: 0.00241969, Hold: 0.00004614000000000007, Free: 0.00237355},
				},
			},
			{
				a: asset.Options,
				exp: accounts.CurrencyBalances{
					// Binance's available balance is not the margin balance less the initial margin, so Hold is what is not
					// available, as for USDⓈ-M and COIN-M: the float64 difference of the two
					currency.USDT: {Currency: currency.USDT, Total: 10099.448, Hold: 1373.5227599999998, Free: 8725.92524},
				},
			},
		} {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				started := time.Now()
				result, err := ex.UpdateAccountBalances(t.Context(), tc.a)
				require.NoError(t, err, "UpdateAccountBalances must not error")
				if !mockTests {
					assert.Len(t, result, 1, "UpdateAccountBalances should return the main account")
					return
				}
				require.Len(t, result, 1, "UpdateAccountBalances must return the main account")
				for c, b := range tc.exp {
					// Balances are stamped when they are saved
					assert.WithinRangef(t, result[0].Balances[c].UpdatedAt, started, time.Now(), "UpdateAccountBalances should stamp the %s balance", c)
					b.UpdatedAt = result[0].Balances[c].UpdatedAt
					tc.exp[c] = b
				}
				assert.Equal(t, accounts.SubAccounts{{AssetType: tc.a, Balances: tc.exp}}, result, "UpdateAccountBalances should return every balance")
			})
		}
	})
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must reject a nil cancellation")
	cancel := &order.Cancel{AssetType: asset.Spot, OrderID: "1"}
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), order.ErrPairIsEmpty, "CancelOrder must reject a cancellation without a pair")
	cancel.Pair = currency.NewBTCUSDT()
	cancel.AssetType = asset.Empty
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), order.ErrAssetNotSet, "CancelOrder must reject a cancellation without an asset")
	cancel.AssetType = asset.Spot
	cancel.OrderID = ""
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), order.ErrOrderIDNotSet, "CancelOrder must reject a cancellation without an ID")
	cancel.OrderID = "abc"
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), errInvalidOrderID, "CancelOrder must reject an invalid order ID")
	cancel.OrderID = "algo-1"
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), errInvalidOrderID, "CancelOrder must reject an algo order ID on spot")
	cancel.AssetType = asset.USDTMarginedFutures
	cancel.OrderID = "algo-abc"
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), errInvalidOrderID, "CancelOrder must reject an invalid algo order ID")
	cancel.OrderID = "1"
	cancel.AssetType = asset.Futures
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), asset.ErrNotSupported, "CancelOrder must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
	}{
		{"spot", &order.Cancel{AssetType: asset.Spot, OrderID: "28"}},
		{"spot OCO", &order.Cancel{AssetType: asset.Spot, Type: order.OCO, OrderID: "1929"}},
		{"margin isolated", &order.Cancel{AssetType: asset.Margin, MarginType: margin.Isolated, OrderID: "12314234"}},
		{"margin OCO by client ID", &order.Cancel{AssetType: asset.Margin, Type: order.OCO, ClientOrderID: "gct-margin-oco"}},
		{"USDⓈ-M", &order.Cancel{AssetType: asset.USDTMarginedFutures, Type: order.Limit, OrderID: "123"}},
		{"USDⓈ-M algo", &order.Cancel{AssetType: asset.USDTMarginedFutures, Type: order.StopMarket, OrderID: "2146760"}},
		{"COIN-M", &order.Cancel{AssetType: asset.CoinMarginedFutures, OrderID: "283194212", ClientOrderID: "myOrder1"}},
		{"COIN-M stop placed as an order", &order.Cancel{AssetType: asset.CoinMarginedFutures, Type: order.StopMarket, OrderID: "283194212", ClientOrderID: "myOrder1"}},
		{"options", &order.Cancel{AssetType: asset.Options, OrderID: "4611875134427365377", ClientOrderID: "213123"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.cancel.Pair = assetToTradablePairMap[tc.cancel.AssetType]
			assert.NoError(t, e.CancelOrder(t.Context(), tc.cancel), "CancelOrder should not error")
		})
	}

	t.Run("websocket API", func(t *testing.T) {
		t.Parallel()
		ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: "12569099454", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot}), "CancelOrder must cancel an order over the WebSocket API")
		assert.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{Type: order.OCO, OrderID: "1274514", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot}), "CancelOrder should cancel an order list over the WebSocket API")
	})
}

// cancelStep is a request a cancellation must send, the parameters it must carry, "" for one it must not, and the reply
type cancelStep struct {
	method, path string
	params       map[string]string
	reply        string
}

// newCancelStepsExchange returns an exchange whose requests must be exactly steps, in order, and a function reporting
// how many were sent
func newCancelStepsExchange(t *testing.T, steps []cancelStep) (ex *Exchange, sent func() int) {
	t.Helper()
	var mu sync.Mutex
	var count int
	ex = newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		i := count
		count++
		mu.Unlock()
		if !assert.Lessf(t, i, len(steps), "%s %s should not be sent", r.Method, r.URL.Path) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		step := steps[i]
		assert.Equalf(t, step.method+" "+step.path, r.Method+" "+r.URL.Path, "request %d should be the expected one", i)
		q := r.URL.Query()
		for k, v := range step.params {
			if v == "" {
				assert.NotContainsf(t, q, k, "request %d should not carry %s", i, k)
			} else {
				assert.Equalf(t, v, q.Get(k), "request %d should carry %s", i, k)
			}
		}
		// A batch reply carries its orders' rejections with a success status
		if strings.HasPrefix(step.reply, `{"code":-`) {
			w.WriteHeader(http.StatusBadRequest)
		}
		_, err := w.Write([]byte(step.reply))
		assert.NoError(t, err, "Write should not error")
	})
	url, err := ex.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
	require.NoError(t, err, "GetURL must not error")
	for _, u := range []exchange.URL{exchange.RestUSDTMargined, exchange.RestCoinMargined} {
		require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(u.String(), url), "SetRunningURL must not error for %s", u)
	}
	return ex, func() int {
		mu.Lock()
		defer mu.Unlock()
		return count
	}
}

func TestCancelOrderUntyped(t *testing.T) {
	t.Parallel()
	// The RPC server and order manager cancel without a type, and orders drop their margin type, while SubmitOrder
	// identifies OCO orders by order list ID and USDⓈ-M conditional orders by algo ID
	const (
		unknownOrder  = `{"code":-2011,"msg":"Unknown order sent."}`
		unknownList   = `{"code":-2011,"msg":"Order list does not exist."}`
		notFound      = `{"code":-2013,"msg":"Order does not exist."}`
		restricted    = `{"code":-2011,"msg":"Order was not canceled due to cancel restrictions."}`
		outsideWindow = `{"code":-1021,"msg":"Timestamp for this request is outside of the recvWindow."}`
		cancelled     = `{"symbol":"BTCUSDT","orderId":28,"status":"CANCELED"}`
		listCancelled = `{"orderListId":1930,"contingencyType":"OCO","listStatusType":"ALL_DONE","listOrderStatus":"ALL_DONE","symbol":"BTCUSDT"}`
		algoCancelled = `{"algoId":3000000123,"clientAlgoId":"gct-algo","code":"200","msg":"success"}`
	)
	algo := func(symbol string, id uint64) string {
		return fmt.Sprintf(`{"algoId":%d,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":%q,"side":"SELL","positionSide":"BOTH","quantity":"0.01","algoStatus":"NEW","triggerPrice":"3000","icebergQuantity":"null"}`, id, symbol)
	}
	spotOrder := func(params map[string]string, reply string) cancelStep {
		return cancelStep{http.MethodDelete, "/api/v3/order", params, reply}
	}
	spotList := func(params map[string]string, reply string) cancelStep {
		return cancelStep{http.MethodDelete, "/api/v3/orderList", params, reply}
	}
	marginOrder := func(isolated, reply string) cancelStep {
		return cancelStep{http.MethodDelete, "/sapi/v1/margin/order", map[string]string{"orderId": "28", "isIsolated": isolated}, reply}
	}
	marginList := func(isolated, reply string) cancelStep {
		return cancelStep{http.MethodDelete, "/sapi/v1/margin/orderList", map[string]string{"orderListId": "28", "isIsolated": isolated}, reply}
	}
	usdtMarginedOrder := func(params map[string]string, reply string) cancelStep {
		return cancelStep{http.MethodDelete, "/fapi/v1/order", params, reply}
	}
	usdtMarginedOrderLookup := func(params map[string]string, reply string) cancelStep {
		return cancelStep{http.MethodGet, "/fapi/v1/order", params, reply}
	}
	usdtMarginedOrderReply := `{"orderId":3000000123,"symbol":"BTCUSDT","status":"NEW","clientOrderId":"gct-algo","price":"3000","origQty":"0.01","executedQty":"0","type":"LIMIT","side":"SELL","time":1791284980615,"updateTime":1791284980615}`
	byID, byAlgoID := map[string]string{"orderId": "3000000123"}, map[string]string{"algoId": "3000000123"}
	for _, tc := range []struct {
		name   string
		cancel order.Cancel
		steps  []cancelStep
		errMsg string
		err    error
	}{
		{
			name:   "spot order list by order list ID",
			cancel: order.Cancel{AssetType: asset.Spot, OrderID: "1930"},
			steps:  []cancelStep{spotOrder(map[string]string{"orderId": "1930"}, unknownOrder), spotList(map[string]string{"orderListId": "1930"}, listCancelled)},
		},
		{
			name:   "spot order list by list client order ID",
			cancel: order.Cancel{AssetType: asset.Spot, ClientOrderID: "gct-oco"},
			steps:  []cancelStep{spotOrder(map[string]string{"origClientOrderId": "gct-oco"}, unknownOrder), spotList(map[string]string{"listClientOrderId": "gct-oco"}, listCancelled)},
		},
		{
			name:   "spot order list of any type",
			cancel: order.Cancel{AssetType: asset.Spot, Type: order.AnyType, OrderID: "1930"},
			steps:  []cancelStep{spotOrder(nil, unknownOrder), spotList(map[string]string{"orderListId": "1930"}, listCancelled)},
		},
		{
			name:   "spot unknown ID",
			cancel: order.Cancel{AssetType: asset.Spot, OrderID: "1930"},
			steps:  []cancelStep{spotOrder(nil, unknownOrder), spotList(nil, unknownList)},
			errMsg: "Order list does not exist.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "spot limit order",
			cancel: order.Cancel{AssetType: asset.Spot, Type: order.Limit, OrderID: "1930"},
			steps:  []cancelStep{spotOrder(nil, unknownOrder)},
			errMsg: "Unknown order sent.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "spot OCO order list",
			cancel: order.Cancel{AssetType: asset.Spot, Type: order.OCO, OrderID: "1930"},
			steps:  []cancelStep{spotList(map[string]string{"orderListId": "1930"}, unknownList)},
			errMsg: "Order list does not exist.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "spot order cancellation restricted",
			cancel: order.Cancel{AssetType: asset.Spot, OrderID: "1930"},
			steps:  []cancelStep{spotOrder(nil, restricted)},
			errMsg: "cancel restrictions",
		},
		{
			name:   "spot order not found then a failure",
			cancel: order.Cancel{AssetType: asset.Spot, OrderID: "1930"},
			steps:  []cancelStep{spotOrder(nil, notFound), spotList(nil, outsideWindow)},
			errMsg: "outside of the recvWindow",
		},
		{
			name:   "margin isolated order",
			cancel: order.Cancel{AssetType: asset.Margin, OrderID: "28"},
			steps:  []cancelStep{marginOrder("", unknownOrder), marginList("", unknownList), marginOrder("TRUE", cancelled)},
		},
		{
			name:   "margin isolated order list",
			cancel: order.Cancel{AssetType: asset.Margin, OrderID: "28"},
			steps:  []cancelStep{marginOrder("", unknownOrder), marginList("", unknownList), marginOrder("TRUE", unknownOrder), marginList("TRUE", listCancelled)},
		},
		{
			// The cross margin account's order list is tried before the isolated margin account, which the pair may not have
			name:   "margin cross order list",
			cancel: order.Cancel{AssetType: asset.Margin, OrderID: "28"},
			steps:  []cancelStep{marginOrder("", unknownOrder), marginList("", listCancelled)},
		},
		{
			name:   "margin without an isolated margin account",
			cancel: order.Cancel{AssetType: asset.Margin, OrderID: "28"},
			steps:  []cancelStep{marginOrder("", unknownOrder), marginList("", unknownList), marginOrder("TRUE", `{"code":-3003,"msg":"Margin account does not exist."}`)},
			errMsg: "Margin account does not exist.",
		},
		{
			name:   "margin isolated OCO without a margin type",
			cancel: order.Cancel{AssetType: asset.Margin, Type: order.OCO, OrderID: "28"},
			steps:  []cancelStep{marginList("", unknownList), marginList("TRUE", listCancelled)},
		},
		{
			name:   "margin isolated limit order without a margin type",
			cancel: order.Cancel{AssetType: asset.Margin, Type: order.Limit, OrderID: "28"},
			steps:  []cancelStep{marginOrder("", unknownOrder), marginOrder("TRUE", cancelled)},
		},
		{
			// A client order ID is unique only among an account's open orders, so the isolated account is not tried
			name:   "margin limit order by client order ID",
			cancel: order.Cancel{AssetType: asset.Margin, Type: order.Limit, ClientOrderID: "gct-margin"},
			steps:  []cancelStep{{http.MethodDelete, "/sapi/v1/margin/order", map[string]string{"origClientOrderId": "gct-margin", "isIsolated": ""}, unknownOrder}},
			errMsg: "Unknown order sent.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "margin order list by list client order ID",
			cancel: order.Cancel{AssetType: asset.Margin, ClientOrderID: "gct-margin"},
			steps: []cancelStep{
				{http.MethodDelete, "/sapi/v1/margin/order", map[string]string{"origClientOrderId": "gct-margin", "isIsolated": ""}, unknownOrder},
				{http.MethodDelete, "/sapi/v1/margin/orderList", map[string]string{"listClientOrderId": "gct-margin", "isIsolated": ""}, unknownList},
			},
			errMsg: "Order list does not exist.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "margin isolated order by client order ID with its margin type",
			cancel: order.Cancel{AssetType: asset.Margin, Type: order.Limit, MarginType: margin.Isolated, ClientOrderID: "gct-margin"},
			steps:  []cancelStep{{http.MethodDelete, "/sapi/v1/margin/order", map[string]string{"origClientOrderId": "gct-margin", "isIsolated": "TRUE"}, cancelled}},
		},
		{
			name:   "margin isolated order list with its margin type",
			cancel: order.Cancel{AssetType: asset.Margin, MarginType: margin.Isolated, OrderID: "28"},
			steps:  []cancelStep{marginOrder("TRUE", unknownOrder), marginList("TRUE", listCancelled)},
		},
		{
			name:   "margin cross limit order",
			cancel: order.Cancel{AssetType: asset.Margin, Type: order.Limit, MarginType: margin.Multi, OrderID: "28"},
			steps:  []cancelStep{marginOrder("", unknownOrder)},
			errMsg: "Unknown order sent.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "margin isolated limit order",
			cancel: order.Cancel{AssetType: asset.Margin, Type: order.Limit, MarginType: margin.Isolated, OrderID: "28"},
			steps:  []cancelStep{marginOrder("TRUE", cancelled)},
		},
		{
			name:   "margin failure",
			cancel: order.Cancel{AssetType: asset.Margin, OrderID: "28"},
			steps:  []cancelStep{marginOrder("", outsideWindow)},
			errMsg: "outside of the recvWindow",
		},
		{
			name:   "USDⓈ-M algo order",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, OrderID: "3000000123"},
			steps: []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algo("BTCUSDT", 3000000123)},
				usdtMarginedOrderLookup(byID, notFound),
				{http.MethodDelete, "/fapi/v1/algoOrder", byAlgoID, algoCancelled},
			},
		},
		{
			name:   "USDⓈ-M algo order by algo order ID",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, OrderID: "algo-3000000123"},
			steps:  []cancelStep{{http.MethodDelete, "/fapi/v1/algoOrder", byAlgoID, algoCancelled}},
		},
		{
			name:   "USDⓈ-M algo order by client ID",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, ClientOrderID: "gct-algo"},
			steps: []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-algo"}, algo("BTCUSDT", 77)},
				usdtMarginedOrderLookup(map[string]string{"origClientOrderId": "gct-algo"}, notFound),
				{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77", "clientAlgoId": ""}, algoCancelled},
			},
		},
		{
			name:   "USDⓈ-M order and algo order",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, OrderID: "3000000123"},
			steps: []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algo("BTCUSDT", 3000000123)},
				usdtMarginedOrderLookup(byID, usdtMarginedOrderReply),
			},
			errMsg: "the algo order as algo-3000000123",
			err:    errAmbiguousOrderID,
		},
		{
			name:   "USDⓈ-M algo order of another symbol",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, OrderID: "3000000123"},
			steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", nil, algo("ETHUSDT", 3000000123)}, usdtMarginedOrder(byID, unknownOrder)},
			errMsg: "Unknown order sent.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "USDⓈ-M unknown ID",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, OrderID: "3000000123"},
			steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", nil, notFound}, usdtMarginedOrder(byID, unknownOrder)},
			errMsg: "Unknown order sent.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "USDⓈ-M limit order",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, Type: order.Limit, OrderID: "3000000123"},
			steps:  []cancelStep{usdtMarginedOrder(byID, unknownOrder)},
			errMsg: "Unknown order sent.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "USDⓈ-M stop market order",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, Type: order.StopMarket, OrderID: "3000000123"},
			steps:  []cancelStep{{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "3000000123"}, unknownOrder}},
			errMsg: "Unknown order sent.",
			err:    order.ErrOrderNotFound,
		},
		{
			// The algo order is triggered or cancelled between its lookup and its cancellation
			name:   "USDⓈ-M algo order gone before its cancellation",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, OrderID: "3000000123"},
			steps: []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algo("BTCUSDT", 3000000123)},
				usdtMarginedOrderLookup(byID, notFound),
				{http.MethodDelete, "/fapi/v1/algoOrder", byAlgoID, unknownOrder},
			},
			errMsg: "error cancelling algo order",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "USDⓈ-M algo order lookup failure",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, OrderID: "3000000123"},
			steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", nil, outsideWindow}},
			errMsg: "outside of the recvWindow",
		},
		{
			name:   "USDⓈ-M algo order found then an order lookup failure",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, OrderID: "3000000123"},
			steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algo("BTCUSDT", 3000000123)}, usdtMarginedOrderLookup(byID, outsideWindow)},
			errMsg: "outside of the recvWindow",
		},
		{
			name:   "COIN-M limit order",
			cancel: order.Cancel{AssetType: asset.CoinMarginedFutures, Type: order.Limit, OrderID: "3000000123"},
			steps:  []cancelStep{{http.MethodDelete, "/dapi/v1/order", byID, unknownOrder}},
			errMsg: "Unknown order sent.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "COIN-M unknown ID",
			cancel: order.Cancel{AssetType: asset.CoinMarginedFutures, OrderID: "3000000123"},
			steps: []cancelStep{
				{http.MethodGet, "/dapi/v1/algoOrder", byAlgoID, notFound},
				{http.MethodDelete, "/dapi/v1/order", byID, unknownOrder},
			},
			errMsg: "Unknown order sent.",
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "COIN-M algo order by algo order ID",
			cancel: order.Cancel{AssetType: asset.CoinMarginedFutures, OrderID: "algo-3000000123"},
			steps:  []cancelStep{{http.MethodDelete, "/dapi/v1/algoOrder", byAlgoID, algoCancelled}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, sent := newCancelStepsExchange(t, tc.steps)
			tc.cancel.Pair = currency.NewBTCUSDT()
			err := ex.CancelOrder(t.Context(), &tc.cancel)
			if tc.errMsg == "" {
				assert.NoError(t, err, "CancelOrder should cancel the order")
			} else {
				assert.ErrorContains(t, err, tc.errMsg, "CancelOrder should return the cancellation error")
			}
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "CancelOrder should return the expected error")
			} else {
				assert.NotErrorIs(t, err, order.ErrOrderNotFound, "CancelOrder should not report a rejection for another reason as no order having the ID")
			}
			assert.Equal(t, len(tc.steps), sent(), "CancelOrder should send every expected request")
		})
	}

	for _, tc := range []struct {
		name    string
		cancel  order.Cancel
		methods []string
		replies []string
		errMsg  string
		err     error
	}{
		{
			name:    "limit order",
			cancel:  order.Cancel{Type: order.Limit},
			methods: []string{"order.cancel"},
			replies: []string{unknownOrder},
			errMsg:  "Unknown order sent.",
			err:     order.ErrOrderNotFound,
		},
		{
			name:    "unknown ID",
			methods: []string{"order.cancel", "orderList.cancel"},
			replies: []string{unknownOrder, unknownList},
			errMsg:  "Order list does not exist.",
			err:     order.ErrOrderNotFound,
		},
		{
			name:    "unknown order then a failure",
			methods: []string{"order.cancel", "orderList.cancel"},
			replies: []string{unknownOrder, outsideWindow},
			errMsg:  "outside of the recvWindow",
		},
		{
			name:    "OCO order list",
			cancel:  order.Cancel{Type: order.OCO},
			methods: []string{"orderList.cancel"},
			replies: []string{unknownList},
			errMsg:  "Order list does not exist.",
			err:     order.ErrOrderNotFound,
		},
	} {
		t.Run("websocket API "+tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.API.AuthenticatedSupport = true
			ex.API.AuthenticatedWebsocketSupport = true
			ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
			var sent atomic.Int32
			useMockWebsocket(t, ex, ex.spotWebsocketAPIConnectionSetup(ex.Config), func(tb testing.TB, msg []byte, w *gws.Conn) error {
				tb.Helper()
				var req struct {
					ID     string `json:"id"`
					Method string `json:"method"`
				}
				require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
				i := int(sent.Add(1)) - 1
				reply := `{"code":-1,"msg":"unexpected request"}`
				if assert.Lessf(tb, i, len(tc.methods), "%s should not be sent", req.Method) {
					assert.Equalf(tb, tc.methods[i], req.Method, "request %d should be the expected one", i)
					reply = tc.replies[i]
				}
				return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,"status":400,"error":%s}`, req.ID, reply))
			})
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			tc.cancel.AssetType, tc.cancel.OrderID, tc.cancel.Pair = asset.Spot, "1930", currency.NewBTCUSDT()
			err := ex.CancelOrder(t.Context(), &tc.cancel)
			assert.ErrorContains(t, err, tc.errMsg, "CancelOrder should return the cancellation error over the WebSocket API")
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "CancelOrder should return the expected error over the WebSocket API")
			} else {
				assert.NotErrorIs(t, err, order.ErrOrderNotFound, "CancelOrder should not report a rejection for another reason as no order having the ID over the WebSocket API")
			}
			assert.Equal(t, len(tc.methods), int(sent.Load()), "CancelOrder should send every expected request over the WebSocket API")
		})
	}

	// An untyped batch entry's untagged ID is looked up before anything is cancelled, so a rejection of an entry the
	// lookup found no algo order for is its outcome, whatever Binance rejected it with
	for _, tc := range []struct {
		name      string
		rejection string
		status    string
	}{
		{name: "unknown order", rejection: unknownOrder, status: "Unknown order sent."},
		{name: "order not found", rejection: notFound, status: "Order does not exist."},
		{name: "cancellation restricted", rejection: restricted, status: "Order was not canceled due to cancel restrictions."},
	} {
		t.Run("USDⓈ-M batch "+tc.name, func(t *testing.T) {
			t.Parallel()
			steps := []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": "1"}, notFound},
				{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": "3000000123"}, notFound},
				{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[1,3000000123]"}, `[{"orderId":1,"symbol":"BTCUSDT","status":"CANCELED"},` + tc.rejection + `]`},
			}
			ex, sent := newCancelStepsExchange(t, steps)
			resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
				{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "1"},
				{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "3000000123"},
			})
			require.NoError(t, err, "CancelBatchOrders must not error")
			assert.Equal(t, map[string]string{"1": "CANCELED", "3000000123": tc.status}, resp.Status, "CancelBatchOrders should report each entry's outcome")
			assert.Equal(t, len(steps), sent(), "CancelBatchOrders should send every expected request")
		})
	}

	// An untagged ID is looked up once, before the batch, however many entries name it and whichever is untyped
	for _, tc := range []struct {
		name    string
		entries []order.Cancel
		replies string
	}{
		{
			name:    "untyped entry cancelled before a typed repeat",
			entries: []order.Cancel{{OrderID: "77"}, {Type: order.Limit, OrderID: "77"}},
			replies: `[{"orderId":77,"symbol":"BTCUSDT","status":"CANCELED"},` + notFound + `]`,
		},
		{
			name:    "untyped repeat of a cancelled ID",
			entries: []order.Cancel{{OrderID: "77"}, {OrderID: "077"}},
			replies: `[{"orderId":77,"symbol":"BTCUSDT","status":"CANCELED"},` + unknownOrder + `]`,
		},
		{
			name:    "untyped entry rejected before a typed one cancels its ID",
			entries: []order.Cancel{{OrderID: "77"}, {Type: order.Limit, OrderID: "77"}},
			replies: `[` + unknownOrder + `,{"orderId":77,"symbol":"BTCUSDT","status":"CANCELED"}]`,
		},
	} {
		t.Run("USDⓈ-M batch "+tc.name, func(t *testing.T) {
			t.Parallel()
			ex, sent := newCancelStepsExchange(t, []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, notFound},
				{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[77,77]"}, tc.replies},
			})
			for i := range tc.entries {
				tc.entries[i].AssetType, tc.entries[i].Pair = asset.USDTMarginedFutures, currency.NewBTCUSDT()
			}
			resp, err := ex.CancelBatchOrders(t.Context(), tc.entries)
			require.NoError(t, err, "CancelBatchOrders must not error")
			assert.Equal(t, map[string]string{"77": "CANCELED"}, resp.Status, "CancelBatchOrders should keep the order's cancellation")
			assert.Equal(t, 2, sent(), "CancelBatchOrders should look the untagged ID up once")
		})
	}

	t.Run("USDⓈ-M batch order named by both its IDs", func(t *testing.T) {
		t.Parallel()
		ex, sent := newCancelStepsExchange(t, []cancelStep{
			{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": "77", "clientAlgoId": ""}, notFound},
			{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "ordinary-client", "algoId": ""}, notFound},
			{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[77]"}, `[{"orderId":77,"clientOrderId":"ordinary-client","symbol":"BTCUSDT","status":"CANCELED"}]`},
			{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["ordinary-client"]`}, `[` + notFound + `]`},
		})
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "77"},
			{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), ClientOrderID: "ordinary-client"},
		})
		require.NoError(t, err, "CancelBatchOrders must not error")
		assert.Equal(t, map[string]string{"77": "CANCELED", "ordinary-client": "Order does not exist."}, resp.Status, "CancelBatchOrders should report each entry's outcome")
		assert.Equal(t, 4, sent(), "CancelBatchOrders should send every expected request")
	})

	t.Run("USDⓈ-M batch zero order ID", func(t *testing.T) {
		t.Parallel()
		// Zero names no order, so there is no algo order to look up by it
		ex, sent := newCancelStepsExchange(t, []cancelStep{
			{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[0]"}, `[` + notFound + `]`},
		})
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "0"},
		})
		require.NoError(t, err, "CancelBatchOrders must not error")
		assert.Equal(t, map[string]string{"0": "Order does not exist."}, resp.Status, "CancelBatchOrders should report the rejection alone")
		assert.Equal(t, 1, sent(), "CancelBatchOrders should not look up an algo order for a zero order ID")
	})

	t.Run("USDⓈ-M batch algo order ahead of an order", func(t *testing.T) {
		t.Parallel()
		ex, sent := newCancelStepsExchange(t, []cancelStep{
			{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": "3000000123"}, algo("BTCUSDT", 3000000123)},
			usdtMarginedOrderLookup(map[string]string{"orderId": "3000000123"}, notFound),
			{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": "1"}, notFound},
			{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[1]"}, `[{"orderId":1,"symbol":"BTCUSDT","status":"CANCELED"}]`},
			{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "3000000123"}, algoCancelled},
		})
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "3000000123"},
			{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "1"},
		})
		require.NoError(t, err, "CancelBatchOrders must not error")
		assert.Equal(t, map[string]string{"1": "CANCELED", "3000000123": "CANCELED"}, resp.Status, "CancelBatchOrders should cancel the algo order the entry names")
		assert.Equal(t, 5, sent(), "CancelBatchOrders should send every expected request")
	})

	t.Run("USDⓈ-M batch order ID sharing a client order ID's text", func(t *testing.T) {
		t.Parallel()
		// The client order ID's rejection is not hidden by the cancellation of the order with the same text as its ID
		ex, sent := newCancelStepsExchange(t, []cancelStep{
			{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": "77", "clientAlgoId": ""}, notFound},
			{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "77", "algoId": ""}, notFound},
			{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[77]"}, `[{"orderId":77,"symbol":"BTCUSDT","status":"CANCELED"}]`},
			{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["77"]`}, `[` + notFound + `]`},
		})
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "77"},
			{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), ClientOrderID: "77"},
		})
		require.NoError(t, err, "CancelBatchOrders must not error")
		assert.Equal(t, map[string]string{"77": "Order does not exist."}, resp.Status, "CancelBatchOrders should report the client order ID's rejection")
		assert.Equal(t, 4, sent(), "CancelBatchOrders should send every expected request")
	})
}

func TestCancelError(t *testing.T) {
	t.Parallel()
	for _, apiErr := range []*APIError{
		{Code: -2011, Message: "Unknown order sent."},
		{Code: -2011, Message: "Order list does not exist."},
	} {
		t.Run(subtestName(apiErr.Message), func(t *testing.T) {
			t.Parallel()
			err := cancelError(apiErr)
			assert.ErrorIsf(t, err, order.ErrOrderNotFound, "cancelError should mark %q as no order having the ID", apiErr.Message)
			assert.ErrorIsf(t, err, apiErr, "cancelError should keep Binance's %q", apiErr.Message)
		})
	}
	// -2013 already maps to order.ErrOrderNotFound, so marking it again would repeat that in its message
	for name, err := range map[string]error{
		"nil":                         nil,
		"code -2013":                  &APIError{Code: -2013, Message: "Order does not exist."},
		"already marked as not found": fmt.Errorf("%w: %w", order.ErrOrderNotFound, &APIError{Code: -2011, Message: "Unknown order sent."}),
		"cancel restrictions":         &APIError{Code: -2011, Message: "Order was not canceled due to cancel restrictions."},
		"code -1021":                  &APIError{Code: -1021, Message: "Timestamp for this request is outside of the recvWindow."},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, err, cancelError(err), "cancelError should return %v as it is", err)
		})
	}
}

func TestCancelFirstFound(t *testing.T) {
	t.Parallel()
	unknownOrder := &APIError{Code: -2011, Message: "Unknown order sent."}
	unknownList := &APIError{Code: -2011, Message: "Order list does not exist."}
	failure := &APIError{Code: -1021, Message: "Timestamp for this request is outside of the recvWindow."}
	for _, tc := range []struct {
		name     string
		results  []error
		attempts int
		err      error
		notFound []string
	}{
		{name: "first attempt succeeds", results: []error{nil, cancelError(unknownList)}, attempts: 1},
		{name: "later attempt succeeds", results: []error{cancelError(unknownOrder), nil}, attempts: 2},
		{
			name:     "no attempt finds the order",
			results:  []error{cancelError(unknownOrder), cancelError(unknownList)},
			attempts: 2,
			notFound: []string{unknownOrder.Message, unknownList.Message},
		},
		{name: "failure stops the attempts", results: []error{cancelError(unknownOrder), failure, cancelError(unknownList)}, attempts: 2, err: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var attempts int
			fns := make([]func() error, len(tc.results))
			for i, result := range tc.results {
				fns[i] = func() error {
					attempts++
					return result
				}
			}
			err := cancelFirstFound(fns...)
			assert.Equal(t, tc.attempts, attempts, "cancelFirstFound should stop after the expected attempt")
			switch {
			case tc.err != nil:
				assert.Equal(t, tc.err, err, "cancelFirstFound should return the failure alone")
			case len(tc.notFound) != 0:
				assert.ErrorIs(t, err, order.ErrOrderNotFound, "cancelFirstFound should report no order having the ID")
				for _, msg := range tc.notFound {
					assert.ErrorContains(t, err, msg, "cancelFirstFound should return every attempt's error")
				}
			default:
				assert.NoError(t, err, "cancelFirstFound should succeed")
			}
		})
	}
}

func TestCancelSpotOrder(t *testing.T) {
	t.Parallel()
	unknownOrder := cancelStep{http.MethodDelete, "/api/v3/order", map[string]string{"orderId": "1930"}, `{"code":-2011,"msg":"Unknown order sent."}`}
	unknownList := cancelStep{http.MethodDelete, "/api/v3/orderList", map[string]string{"orderListId": "1930"}, `{"code":-2011,"msg":"Order list does not exist."}`}
	for _, tc := range []struct {
		orderType order.Type
		steps     []cancelStep
	}{
		{order.Limit, []cancelStep{unknownOrder}},
		{order.OCO, []cancelStep{unknownList}},
		{order.UnknownType, []cancelStep{unknownOrder, unknownList}},
	} {
		t.Run(tc.orderType.String(), func(t *testing.T) {
			t.Parallel()
			ex, sent := newCancelStepsExchange(t, tc.steps)
			err := ex.cancelSpotOrder(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT(), Type: tc.orderType, OrderID: "1930"}, 1930)
			assert.ErrorIs(t, err, order.ErrOrderNotFound, "cancelSpotOrder should report no order having the ID")
			assert.Equal(t, len(tc.steps), sent(), "cancelSpotOrder should send every expected request")
		})
	}
}

func TestCancelMarginOrder(t *testing.T) {
	t.Parallel()
	marginOrder := func(isolated string) cancelStep {
		return cancelStep{http.MethodDelete, "/sapi/v1/margin/order", map[string]string{"orderId": "28", "isIsolated": isolated}, `{"code":-2011,"msg":"Unknown order sent."}`}
	}
	marginList := func(isolated string) cancelStep {
		return cancelStep{http.MethodDelete, "/sapi/v1/margin/orderList", map[string]string{"orderListId": "28", "isIsolated": isolated}, `{"code":-2011,"msg":"Order list does not exist."}`}
	}
	for _, tc := range []struct {
		name   string
		cancel order.Cancel
		steps  []cancelStep
	}{
		{name: "cross margin limit order", cancel: order.Cancel{Type: order.Limit, MarginType: margin.Multi}, steps: []cancelStep{marginOrder("")}},
		{name: "isolated margin OCO order list", cancel: order.Cancel{Type: order.OCO, MarginType: margin.Isolated}, steps: []cancelStep{marginList("TRUE")}},
		{name: "untyped without a margin type", steps: []cancelStep{marginOrder(""), marginList(""), marginOrder("TRUE"), marginList("TRUE")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, sent := newCancelStepsExchange(t, tc.steps)
			tc.cancel.AssetType, tc.cancel.Pair, tc.cancel.OrderID = asset.Margin, currency.NewBTCUSDT(), "28"
			err := ex.cancelMarginOrder(t.Context(), &tc.cancel, 28)
			assert.ErrorIs(t, err, order.ErrOrderNotFound, "cancelMarginOrder should report no order having the ID")
			assert.Equal(t, len(tc.steps), sent(), "cancelMarginOrder should send every expected request")
		})
	}
}

func TestCancelFuturesOrder(t *testing.T) {
	t.Parallel()
	const (
		unknownOrder  = `{"code":-2011,"msg":"Unknown order sent."}`
		notFound      = `{"code":-2013,"msg":"Order does not exist."}`
		outsideWindow = `{"code":-1021,"msg":"Timestamp for this request is outside of the recvWindow."}`
		algoOrder     = `{"algoId":3000000123,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSDT","side":"SELL","positionSide":"BOTH","quantity":"0.01","algoStatus":"NEW","triggerPrice":"3000","icebergQuantity":"null"}`
		usdtOrder     = `{"orderId":3000000123,"symbol":"BTCUSDT","status":"NEW","clientOrderId":"gct-order","price":"3000","origQty":"0.01","executedQty":"0","type":"LIMIT","side":"BUY","time":1791284980615,"updateTime":1791284980615}`
		algoCancelled = `{"algoId":3000000123,"clientAlgoId":"gct-algo","code":"200","msg":"success"}`
		cancelled     = `{"orderId":3000000123,"symbol":"BTCUSDT","status":"CANCELED"}`
	)
	byID, byAlgoID := map[string]string{"orderId": "3000000123"}, map[string]string{"algoId": "3000000123"}
	for _, tc := range []struct {
		name   string
		cancel order.Cancel
		isAlgo bool
		steps  []cancelStep
		err    error
		errMsg string
	}{
		{
			name:   "USDⓈ-M stop market order",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, Type: order.StopMarket},
			steps:  []cancelStep{{http.MethodDelete, "/fapi/v1/algoOrder", byAlgoID, unknownOrder}},
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "USDⓈ-M algo order ID of a limit type",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, Type: order.Limit},
			isAlgo: true,
			steps:  []cancelStep{{http.MethodDelete, "/fapi/v1/algoOrder", byAlgoID, algoCancelled}},
		},
		{
			name:   "USDⓈ-M algo order ID with a client order ID",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures, ClientOrderID: "gct-algo"},
			isAlgo: true,
			steps:  []cancelStep{{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "3000000123", "clientAlgoId": ""}, algoCancelled}},
		},
		{
			name:   "COIN-M limit order",
			cancel: order.Cancel{AssetType: asset.CoinMarginedFutures, Type: order.Limit},
			steps:  []cancelStep{{http.MethodDelete, "/dapi/v1/order", byID, unknownOrder}},
			err:    order.ErrOrderNotFound,
		},
		{
			name:   "USDⓈ-M untyped without an algo order",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures},
			steps: []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, notFound},
				{http.MethodDelete, "/fapi/v1/order", byID, cancelled},
			},
		},
		{
			name:   "USDⓈ-M untyped unknown ID",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures},
			steps: []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, notFound},
				{http.MethodDelete, "/fapi/v1/order", byID, unknownOrder},
			},
			err: order.ErrOrderNotFound,
		},
		{
			name:   "USDⓈ-M untyped algo order",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures},
			steps: []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoOrder},
				{http.MethodGet, "/fapi/v1/order", byID, notFound},
				{http.MethodDelete, "/fapi/v1/algoOrder", byAlgoID, algoCancelled},
			},
		},
		{
			name:   "USDⓈ-M untyped algo order gone before its cancellation",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures},
			steps: []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoOrder},
				{http.MethodGet, "/fapi/v1/order", byID, notFound},
				{http.MethodDelete, "/fapi/v1/algoOrder", byAlgoID, unknownOrder},
			},
			err:    order.ErrOrderNotFound,
			errMsg: "error cancelling algo order",
		},
		{
			name:   "USDⓈ-M untyped ID of an order and an algo order",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures},
			steps: []cancelStep{
				{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoOrder},
				{http.MethodGet, "/fapi/v1/order", byID, usdtOrder},
			},
			err:    errAmbiguousOrderID,
			errMsg: "algo-3000000123",
		},
		{
			name:   "USDⓈ-M untyped algo order lookup failure",
			cancel: order.Cancel{AssetType: asset.USDTMarginedFutures},
			steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, outsideWindow}},
			errMsg: "outside of the recvWindow",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, sent := newCancelStepsExchange(t, tc.steps)
			tc.cancel.Pair, tc.cancel.OrderID = currency.NewBTCUSDT(), "3000000123"
			err := ex.cancelFuturesOrder(t.Context(), &tc.cancel, 3000000123, tc.isAlgo)
			if tc.errMsg != "" {
				assert.ErrorContains(t, err, tc.errMsg, "cancelFuturesOrder should return the expected error")
			}
			switch {
			case tc.err != nil:
				assert.ErrorIs(t, err, tc.err, "cancelFuturesOrder should return the expected error")
			case tc.errMsg != "":
				assert.NotErrorIs(t, err, order.ErrOrderNotFound, "cancelFuturesOrder should not report a failed lookup as no order having the ID")
			default:
				assert.NoError(t, err, "cancelFuturesOrder should cancel the order")
			}
			assert.Equal(t, len(tc.steps), sent(), "cancelFuturesOrder should send every expected request")
		})
	}

	// A USDⓈ-M stop, take profit or trailing stop type names an algo order, while COIN-M, which takes such orders through
	// New Order until Binance's cut-over, looks the ID up as it does an untyped request's
	for _, orderType := range append([]order.Type{order.AnyType}, conditionalOrderTypes...) {
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			t.Run(a.String()+" "+orderType.String()+" type", func(t *testing.T) {
				t.Parallel()
				var steps []cancelStep
				switch {
				case a == asset.CoinMarginedFutures:
					steps = []cancelStep{
						{http.MethodGet, "/dapi/v1/algoOrder", byAlgoID, notFound},
						{http.MethodDelete, "/dapi/v1/order", byID, cancelled},
					}
				case orderType == order.AnyType:
					steps = []cancelStep{
						{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, notFound},
						{http.MethodDelete, "/fapi/v1/order", byID, cancelled},
					}
				default:
					steps = []cancelStep{{http.MethodDelete, "/fapi/v1/algoOrder", byAlgoID, algoCancelled}}
				}
				ex, sent := newCancelStepsExchange(t, steps)
				c := &order.Cancel{AssetType: a, Pair: currency.NewBTCUSDT(), Type: orderType, OrderID: "3000000123"}
				require.NoErrorf(t, ex.cancelFuturesOrder(t.Context(), c, 3000000123, false), "cancelFuturesOrder must cancel a %s %s order", a, orderType)
				assert.Equalf(t, len(steps), sent(), "cancelFuturesOrder should send every expected request for a %s %s order", a, orderType)
			})
		}
	}

	// A stop, take profit or trailing stop named by its client order ID alone goes by its client algo ID on USDⓈ-M, and
	// is looked up by it on COIN-M
	byClientAlgoID := map[string]string{"clientAlgoId": "gct-stop", "algoId": ""}
	for _, orderType := range conditionalOrderTypes {
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			t.Run(a.String()+" "+orderType.String()+" type by client order ID", func(t *testing.T) {
				t.Parallel()
				steps := []cancelStep{{http.MethodDelete, "/fapi/v1/algoOrder", byClientAlgoID, algoCancelled}}
				if a == asset.CoinMarginedFutures {
					steps = []cancelStep{
						{http.MethodGet, "/dapi/v1/algoOrder", byClientAlgoID, notFound},
						{http.MethodDelete, "/dapi/v1/order", map[string]string{"origClientOrderId": "gct-stop", "orderId": ""}, cancelled},
					}
				}
				ex, sent := newCancelStepsExchange(t, steps)
				c := &order.Cancel{AssetType: a, Pair: currency.NewBTCUSDT(), Type: orderType, ClientOrderID: "gct-stop"}
				require.NoErrorf(t, ex.cancelFuturesOrder(t.Context(), c, 0, false), "cancelFuturesOrder must cancel a %s %s order by its client order ID", a, orderType)
				assert.Equalf(t, len(steps), sent(), "cancelFuturesOrder should send every expected request for a %s %s order named by its client order ID", a, orderType)
			})
		}
	}

	// An algo order ID names an algo order whatever type comes with it
	for _, orderType := range append([]order.Type{order.UnknownType, order.AnyType, order.Limit, order.Market}, conditionalOrderTypes...) {
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			t.Run(a.String()+" algo order ID with a "+orderType.String()+" type", func(t *testing.T) {
				t.Parallel()
				path := "/fapi/v1/algoOrder"
				if a == asset.CoinMarginedFutures {
					path = "/dapi/v1/algoOrder"
				}
				steps := []cancelStep{{http.MethodDelete, path, byAlgoID, algoCancelled}}
				ex, sent := newCancelStepsExchange(t, steps)
				c := &order.Cancel{AssetType: a, Pair: currency.NewBTCUSDT(), Type: orderType, OrderID: "algo-3000000123"}
				require.NoErrorf(t, ex.cancelFuturesOrder(t.Context(), c, 3000000123, true), "cancelFuturesOrder must cancel a %s algo order named with a %s type", a, orderType)
				assert.Equalf(t, len(steps), sent(), "cancelFuturesOrder should send every expected request for a %s algo order named with a %s type", a, orderType)
			})
		}
	}
}

func TestPairAlgoOrder(t *testing.T) {
	t.Parallel()
	algoReply := func(symbol string) string {
		return `{"algoId":3000000123,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"` + symbol + `","side":"SELL","positionSide":"BOTH","quantity":"0.01","algoStatus":"NEW","triggerPrice":"3000","icebergQuantity":"null"}`
	}
	byAlgoID := map[string]string{"algoId": "3000000123"}
	for _, tc := range []struct {
		name   string
		a      asset.Item
		pair   currency.Pair
		steps  []cancelStep
		err    error
		errMsg string
	}{
		{
			name:  "pair's algo order",
			a:     asset.USDTMarginedFutures,
			pair:  currency.NewBTCUSDT(),
			steps: []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoReply("BTCUSDT")}},
		},
		{
			name:  "COIN-M pair's algo order",
			a:     asset.CoinMarginedFutures,
			pair:  testCoinMarginedFuturesPair,
			steps: []cancelStep{{http.MethodGet, "/dapi/v1/algoOrder", byAlgoID, algoReply("BTCUSD_PERP")}},
		},
		{
			name:  "another pair's algo order",
			a:     asset.USDTMarginedFutures,
			pair:  currency.NewBTCUSDT(),
			steps: []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoReply("ETHUSDT")}},
			err:   order.ErrOrderNotFound,
		},
		{
			name:  "no algo order",
			a:     asset.USDTMarginedFutures,
			pair:  currency.NewBTCUSDT(),
			steps: []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, `{"code":-2013,"msg":"Order does not exist."}`}},
			err:   order.ErrOrderNotFound,
		},
		{
			name:   "lookup failure",
			a:      asset.USDTMarginedFutures,
			pair:   currency.NewBTCUSDT(),
			steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, `{"code":-1021,"msg":"Timestamp for this request is outside of the recvWindow."}`}},
			errMsg: "error fetching algo order",
		},
		{
			name:  "asset without pairs",
			a:     asset.Binary,
			pair:  currency.NewBTCUSDT(),
			steps: []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoReply("BTCUSDT")}},
			err:   asset.ErrNotSupported,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, sent := newCancelStepsExchange(t, tc.steps)
			algo, err := ex.pairAlgoOrder(t.Context(), tc.a, tc.pair, 3000000123, "", usdtMarginedFuturesCancelRate)
			switch {
			case tc.err != nil:
				assert.ErrorIs(t, err, tc.err, "pairAlgoOrder should return the expected error")
				assert.Nil(t, algo, "pairAlgoOrder should return no algo order")
			case tc.errMsg != "":
				assert.ErrorContains(t, err, tc.errMsg, "pairAlgoOrder should return the lookup error")
				assert.NotErrorIs(t, err, order.ErrOrderNotFound, "pairAlgoOrder should not report a failed lookup as no algo order")
			default:
				require.NoError(t, err, "pairAlgoOrder must not error")
				require.NotNil(t, algo, "pairAlgoOrder must return the algo order")
				assert.Equal(t, uint64(3000000123), algo.AlgoID, "pairAlgoOrder should return the algo order")
			}
			assert.Equal(t, len(tc.steps), sent(), "pairAlgoOrder should send every expected request")
		})
	}
}

// TestUntaggedAlgoOrder ensures that an untyped request's untagged ID is found to be an algo order's only when the pair
// has an algo order with it and no order with it, as the ID may name either
func TestUntaggedAlgoOrder(t *testing.T) {
	t.Parallel()
	const (
		notFound      = `{"code":-2013,"msg":"Order does not exist."}`
		unknownOrder  = `{"code":-2011,"msg":"Unknown order sent."}`
		outsideWindow = `{"code":-1021,"msg":"Timestamp for this request is outside of the recvWindow."}`
	)
	algoReply := func(symbol string) string {
		return `{"algoId":77,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"` + symbol + `","side":"SELL","positionSide":"BOTH","quantity":"0.01","algoStatus":"NEW","triggerPrice":"3000","icebergQuantity":"null"}`
	}
	usdtOrder := `{"orderId":77,"symbol":"BTCUSDT","status":"FILLED","clientOrderId":"gct-algo","price":"3000","origQty":"0.01","executedQty":"0.01","type":"LIMIT","side":"BUY","time":1791284980615,"updateTime":1791284980615}`
	coinOrder := `{"orderId":77,"symbol":"BTCUSD_PERP","pair":"BTCUSD","status":"FILLED","clientOrderId":"gct-algo","price":"3000","origQty":"1","executedQty":"1","type":"LIMIT","side":"BUY","time":1791284980615,"updateTime":1791284980615}`
	byAlgoID, byID := map[string]string{"algoId": "77", "clientAlgoId": ""}, map[string]string{"orderId": "77", "origClientOrderId": ""}
	byClientAlgoID, byClientID := map[string]string{"clientAlgoId": "gct-algo", "algoId": ""}, map[string]string{"origClientOrderId": "gct-algo", "orderId": ""}
	for _, tc := range []struct {
		name          string
		a             asset.Item
		orderID       uint64
		clientOrderID string
		steps         []cancelStep
		algo          bool
		err           error
		errMsg        string
	}{
		{
			name: "no algo order", a: asset.USDTMarginedFutures, orderID: 77,
			steps: []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, notFound}},
		},
		{
			name: "algo order lookup rejected as unknown", a: asset.USDTMarginedFutures, orderID: 77,
			steps: []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, unknownOrder}},
		},
		{
			name: "algo order alone with the order lookup rejected as unknown", a: asset.USDTMarginedFutures, orderID: 77,
			steps: []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoReply("BTCUSDT")}, {http.MethodGet, "/fapi/v1/order", byID, unknownOrder}},
			algo:  true,
		},
		{
			name: "another pair's algo order", a: asset.USDTMarginedFutures, orderID: 77,
			steps: []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoReply("ETHUSDT")}},
		},
		{
			name: "algo order lookup failure", a: asset.USDTMarginedFutures, orderID: 77,
			steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, outsideWindow}},
			errMsg: "error fetching algo order",
		},
		{
			name: "algo order alone", a: asset.USDTMarginedFutures, orderID: 77,
			steps: []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoReply("BTCUSDT")}, {http.MethodGet, "/fapi/v1/order", byID, notFound}},
			algo:  true,
		},
		{
			name: "algo order alone by client ID", a: asset.USDTMarginedFutures, clientOrderID: "gct-algo",
			steps: []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byClientAlgoID, algoReply("BTCUSDT")}, {http.MethodGet, "/fapi/v1/order", byClientID, notFound}},
			algo:  true,
		},
		{
			name: "algo order and order", a: asset.USDTMarginedFutures, orderID: 77,
			steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoReply("BTCUSDT")}, {http.MethodGet, "/fapi/v1/order", byID, usdtOrder}},
			err:    errAmbiguousOrderID,
			errMsg: "77 on usdtmarginedfutures BTCUSDT; name the order by its client order ID alone, or the algo order as algo-77",
		},
		{
			name: "algo order and order by client ID", a: asset.USDTMarginedFutures, clientOrderID: "gct-algo",
			steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byClientAlgoID, algoReply("BTCUSDT")}, {http.MethodGet, "/fapi/v1/order", byClientID, usdtOrder}},
			err:    errAmbiguousOrderID,
			errMsg: "gct-algo on usdtmarginedfutures BTCUSDT; name the order by its order ID, or the algo order as algo-77",
		},
		{
			name: "order lookup failure", a: asset.USDTMarginedFutures, orderID: 77,
			steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoReply("BTCUSDT")}, {http.MethodGet, "/fapi/v1/order", byID, outsideWindow}},
			errMsg: "error fetching order",
		},
		{
			name: "COIN-M algo order alone", a: asset.CoinMarginedFutures, orderID: 77,
			steps: []cancelStep{{http.MethodGet, "/dapi/v1/algoOrder", byAlgoID, algoReply("BTCUSD_PERP")}, {http.MethodGet, "/dapi/v1/order", byID, notFound}},
			algo:  true,
		},
		{
			name: "COIN-M algo order and order", a: asset.CoinMarginedFutures, orderID: 77,
			steps:  []cancelStep{{http.MethodGet, "/dapi/v1/algoOrder", byAlgoID, algoReply("BTCUSD_PERP")}, {http.MethodGet, "/dapi/v1/order", byID, coinOrder}},
			err:    errAmbiguousOrderID,
			errMsg: "77 on coinmarginedfutures BTCUSD_PERP; name the order by its client order ID alone, or the algo order as algo-77",
		},
		{
			name: "COIN-M algo order and order by client ID", a: asset.CoinMarginedFutures, clientOrderID: "gct-algo",
			steps:  []cancelStep{{http.MethodGet, "/dapi/v1/algoOrder", byClientAlgoID, algoReply("BTCUSD_PERP")}, {http.MethodGet, "/dapi/v1/order", byClientID, coinOrder}},
			err:    errAmbiguousOrderID,
			errMsg: "gct-algo on coinmarginedfutures BTCUSD_PERP; name the order by its order ID, or the algo order as algo-77",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, sent := newCancelStepsExchange(t, tc.steps)
			pair := currency.NewBTCUSDT()
			if tc.a == asset.CoinMarginedFutures {
				pair = testCoinMarginedFuturesPair
			}
			algo, err := ex.untaggedAlgoOrder(t.Context(), tc.a, pair, tc.orderID, tc.clientOrderID)
			if tc.errMsg != "" {
				assert.ErrorContains(t, err, tc.errMsg, "untaggedAlgoOrder should return the expected error")
			}
			switch {
			case tc.err != nil:
				assert.ErrorIs(t, err, tc.err, "untaggedAlgoOrder should return the expected error")
			case tc.errMsg != "":
				assert.NotErrorIs(t, err, order.ErrOrderNotFound, "untaggedAlgoOrder should not report a failed lookup as no order having the ID")
			default:
				assert.NoError(t, err, "untaggedAlgoOrder should not error")
			}
			if tc.algo {
				require.NotNil(t, algo, "untaggedAlgoOrder must return the algo order")
				assert.Equal(t, uint64(77), algo.AlgoID, "untaggedAlgoOrder should return the pair's algo order")
			} else {
				assert.Nil(t, algo, "untaggedAlgoOrder should return no algo order")
			}
			assert.Equal(t, len(tc.steps), sent(), "untaggedAlgoOrder should send every expected request")
		})
	}
}

// TestCancelOrderAlgoOrderID ensures that cancelling a stop by the ID SubmitOrder returned for it cancels the stop and
// not a regular order of the pair numbered alike, as Binance numbers algo orders apart from other orders. An untyped
// cancellation by the bare number, which names both, is refused, while a typed one cancels the kind its type names
func TestCancelOrderAlgoOrderID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                   string
		cancelType             order.Type
		orderID                string // the submitted stop's ID when empty
		regular, algo          bool
		err                    error
		sentRegular, sentQuery bool
	}{
		{name: "cancel the stop when a regular order has the same ID", algo: true},
		{name: "refuse the untagged ID of a regular order and a stop", orderID: "77", err: errAmbiguousOrderID, sentRegular: true, sentQuery: true},
		{name: "cancel the stop by its type and untagged ID", cancelType: order.StopMarket, orderID: "77", algo: true},
		{name: "cancel the regular order by its type and untagged ID", cancelType: order.Limit, orderID: "77", regular: true, sentRegular: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var regularCancelled, algoCancelled, regularQueried, algoQueried atomic.Bool
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/fapi/v1/algoOrder":
					switch r.Method {
					case http.MethodDelete:
						algoCancelled.Store(true)
						_, err := w.Write([]byte(`{"algoId":77,"code":200,"msg":"success"}`))
						assert.NoError(t, err, "Write should not error")
						return
					case http.MethodGet:
						algoQueried.Store(true)
					}
					_, err := w.Write([]byte(`{"algoId":77,"clientAlgoId":"stop-77","symbol":"BTCUSDT","side":"SELL","orderType":"STOP_MARKET","quantity":"1","algoStatus":"NEW","triggerPrice":"90"}`))
					assert.NoError(t, err, "Write should not error")
				case "/fapi/v1/order":
					switch r.Method {
					case http.MethodDelete:
						regularCancelled.Store(true)
					case http.MethodGet:
						regularQueried.Store(true)
					}
					_, err := w.Write([]byte(`{"orderId":77,"symbol":"BTCUSDT","status":"CANCELED"}`))
					assert.NoError(t, err, "Write should not error")
				default:
					assert.Fail(t, "unexpected endpoint", "the request should use an order endpoint")
				}
			})
			placed, err := ex.SubmitOrder(t.Context(), &order.Submit{
				Exchange: ex.Name, Pair: currency.NewBTCUSDT(),
				AssetType: asset.USDTMarginedFutures, Type: order.StopMarket,
				Side: order.Sell, Amount: 1, TriggerPrice: 90,
			})
			require.NoError(t, err, "SubmitOrder must place the stop")
			require.Equal(t, "algo-77", placed.OrderID, "SubmitOrder must identify the stop by its algo order ID")
			err = ex.CancelOrder(t.Context(), &order.Cancel{
				Exchange: ex.Name, Pair: placed.Pair, Type: tc.cancelType,
				AssetType: placed.AssetType, OrderID: cmp.Or(tc.orderID, placed.OrderID),
			})
			require.ErrorIs(t, err, tc.err, "CancelOrder must return the expected error")
			assert.Equal(t, tc.regular, regularCancelled.Load(), "CancelOrder should cancel the regular order only when it is meant")
			assert.Equal(t, tc.algo, algoCancelled.Load(), "CancelOrder should cancel the stop only when it is meant")
			assert.Equal(t, tc.sentQuery, algoQueried.Load(), "CancelOrder should look the stop up only for an untyped untagged ID")
			assert.Equal(t, tc.sentQuery, regularQueried.Load(), "CancelOrder should look the regular order up only for an untyped untagged ID")
		})
	}
}

// TestAlgoOrderIDRoundTrips ensures that the algo order ID each read of a USDⓈ-M or COIN-M algo order reports reaches
// the algo order when it is passed back, with no lookup of either kind of order
func TestAlgoOrderIDRoundTrips(t *testing.T) {
	t.Parallel()
	algoReply := func(symbol string) string {
		return `{"algoId":3000000123,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"` + symbol + `","side":"SELL","positionSide":"BOTH","timeInForce":"GTC","quantity":"1","algoStatus":"NEW","triggerPrice":"3000","icebergQuantity":"null","createTime":1791284812300,"updateTime":1791284812400}`
	}
	algoCancelled := `{"algoId":3000000123,"clientAlgoId":"gct-algo","code":"200","msg":"success"}`
	byAlgoID := map[string]string{"algoId": "3000000123", "symbol": ""}
	for _, tc := range []struct {
		name  string
		a     asset.Item
		read  cancelStep
		ids   func(*testing.T, *Exchange, currency.Pair) []string
		paths [2]string // the algo order endpoint and the order endpoint
	}{
		{
			name: "USDⓈ-M open algo orders",
			a:    asset.USDTMarginedFutures,
			read: cancelStep{http.MethodGet, "/fapi/v1/openAlgoOrders", map[string]string{"symbol": "BTCUSDT"}, `[` + algoReply("BTCUSDT") + `]`},
			ids: func(t *testing.T, ex *Exchange, pair currency.Pair) []string {
				t.Helper()
				orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Pairs: currency.Pairs{pair}, Type: order.StopMarket, Side: order.AnySide})
				require.NoError(t, err, "GetActiveOrders must not error")
				return orderIDs(orders)
			},
			paths: [2]string{"/fapi/v1/algoOrder", "/fapi/v1/order"},
		},
		{
			name: "USDⓈ-M algo order history",
			a:    asset.USDTMarginedFutures,
			read: cancelStep{http.MethodGet, "/fapi/v1/allAlgoOrders", map[string]string{"symbol": "BTCUSDT"}, `[` + algoReply("BTCUSDT") + `]`},
			ids: func(t *testing.T, ex *Exchange, pair currency.Pair) []string {
				t.Helper()
				orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Pairs: currency.Pairs{pair}, Type: order.StopMarket, Side: order.AnySide})
				require.NoError(t, err, "GetOrderHistory must not error")
				return orderIDs(orders)
			},
			paths: [2]string{"/fapi/v1/algoOrder", "/fapi/v1/order"},
		},
		{
			name: "COIN-M algo order",
			a:    asset.CoinMarginedFutures,
			read: cancelStep{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"algoId": "3000000123"}, algoReply("BTCUSD_PERP")},
			ids: func(t *testing.T, ex *Exchange, pair currency.Pair) []string {
				t.Helper()
				d, err := ex.GetOrderInfo(t.Context(), "algo-3000000123", pair, asset.CoinMarginedFutures)
				require.NoError(t, err, "GetOrderInfo must not error")
				return []string{d.OrderID}
			},
			paths: [2]string{"/dapi/v1/algoOrder", "/dapi/v1/order"},
		},
		{
			name: "USDⓈ-M algo order placement",
			a:    asset.USDTMarginedFutures,
			read: cancelStep{http.MethodPost, "/fapi/v1/algoOrder", map[string]string{"type": "STOP_MARKET"}, algoReply("BTCUSDT")},
			ids: func(t *testing.T, ex *Exchange, pair currency.Pair) []string {
				t.Helper()
				placed, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: pair, AssetType: asset.USDTMarginedFutures, Type: order.StopMarket, Side: order.Sell, Amount: 1, TriggerPrice: 3000})
				require.NoError(t, err, "SubmitOrder must not error")
				return []string{placed.OrderID}
			},
			paths: [2]string{"/fapi/v1/algoOrder", "/fapi/v1/order"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pair := currency.NewBTCUSDT()
			if tc.a == asset.CoinMarginedFutures {
				pair = testCoinMarginedFuturesPair
			}
			ex, sent := newCancelStepsExchange(t, []cancelStep{
				tc.read,
				{http.MethodGet, tc.paths[0], byAlgoID, algoReply(map[asset.Item]string{asset.USDTMarginedFutures: "BTCUSDT", asset.CoinMarginedFutures: "BTCUSD_PERP"}[tc.a])},
				{http.MethodDelete, tc.paths[0], byAlgoID, algoCancelled},
				{http.MethodDelete, tc.paths[0], byAlgoID, algoCancelled},
			})
			ids := tc.ids(t, ex, pair)
			require.Equal(t, []string{"algo-3000000123"}, ids, "the read must report the algo order ID")
			d, err := ex.GetOrderInfo(t.Context(), ids[0], pair, tc.a)
			require.NoError(t, err, "GetOrderInfo must look the algo order up")
			assert.Equal(t, ids[0], d.OrderID, "GetOrderInfo should report the algo order ID it was given")
			require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{AssetType: tc.a, Pair: pair, OrderID: ids[0]}), "CancelOrder must cancel the algo order")
			resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: tc.a, Pair: pair, OrderID: ids[0]}})
			require.NoError(t, err, "CancelBatchOrders must cancel the algo order")
			assert.Equal(t, map[string]string{ids[0]: "CANCELED"}, resp.Status, "CancelBatchOrders should key the status by the algo order ID")
			_, err = ex.ModifyOrder(t.Context(), &order.Modify{AssetType: tc.a, Pair: pair, OrderID: ids[0], Side: order.Sell, Price: 1, Amount: 1})
			assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "ModifyOrder should refuse the algo order without a request")
			assert.Equal(t, 4, sent(), "the algo order ID should reach the algo order endpoints alone")
		})
	}

	t.Run("ALGO_UPDATE", func(t *testing.T) {
		t.Parallel()
		ex, sent := newCancelStepsExchange(t, []cancelStep{{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "2148720"}, algoCancelled}})
		err := ex.processFuturesAlgoUpdate(t.Context(), []byte(`{"e":"ALGO_UPDATE","T":1791300000000,"E":1791300000006,"o":{"caid":"stop-loss","aid":2148720,"at":"CONDITIONAL","o":"STOP_MARKET","s":"BTCUSDT","S":"SELL","ps":"BOTH","f":"GTC","q":"0.01","X":"NEW","tp":"85000","p":"0"}}`), asset.USDTMarginedFutures)
		require.NoError(t, err, "processFuturesAlgoUpdate must not error")
		payloads := relayedPayloads(ex)
		require.Len(t, payloads, 2, "processFuturesAlgoUpdate must relay the event and its algo order")
		d, ok := payloads[1].(*order.Detail)
		require.True(t, ok, "processFuturesAlgoUpdate must relay the algo order")
		require.Equal(t, "algo-2148720", d.OrderID, "processFuturesAlgoUpdate must report the algo order ID")
		require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{AssetType: d.AssetType, Pair: d.Pair, OrderID: d.OrderID}), "CancelOrder must cancel the algo order")
		assert.Equal(t, 1, sent(), "CancelOrder should cancel the algo order with no lookup")
	})
}

// orderIDs returns the IDs of orders
func orderIDs(orders []order.Detail) []string {
	ids := make([]string, len(orders))
	for i := range orders {
		ids[i] = orders[i].OrderID
	}
	return ids
}

func TestCancelAlgoOrderLookupRate(t *testing.T) {
	t.Parallel()
	// Order book snapshots can hold the derivatives IP pool for seconds, so the lookups an untyped cancellation or
	// modification makes draw on the cancel pool as the cancellation does, while other lookups stay on the IP pool
	const algoOrder = `{"algoId":3000000123,"clientAlgoId":"gct-algo","symbol":"BTCUSDT","algoStatus":"NEW"}`
	ex, sent := newCancelStepsExchange(t, []cancelStep{
		{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": "3000000123"}, algoOrder},
		{http.MethodGet, "/fapi/v1/order", map[string]string{"orderId": "3000000123"}, `{"code":-2013,"msg":"Order does not exist."}`},
		{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "3000000123"}, `{"algoId":3000000123,"clientAlgoId":"gct-algo","code":"200","msg":"success"}`},
		{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": "3000000123"}, algoOrder},
		{http.MethodGet, "/fapi/v1/order", map[string]string{"orderId": "3000000123"}, `{"code":-2013,"msg":"Order does not exist."}`},
	})
	require.NoError(t, ex.EnableRateLimiter(), "EnableRateLimiter must not error")
	exhausted := request.NewRateLimitWithWeight(time.Hour, 1, 1)
	require.NoError(t, exhausted.RateLimit(t.Context()), "RateLimit must take the limiter's only token")
	definitions := ex.Requester.GetRateLimiterDefinitions()
	for epl, cost := range endpointCosts {
		if cost.pool == derivativesIPPool {
			definitions[epl] = exhausted
		}
	}
	ctx := request.WithDelayNotAllowed(t.Context())
	_, err := ex.USDTMarginedFuturesQueryAlgoOrder(ctx, 3000000123, "")
	require.ErrorIs(t, err, request.ErrDelayNotAllowed, "USDTMarginedFuturesQueryAlgoOrder must draw on the IP pool")
	_, err = ex.USDTMarginedFuturesGetOrderData(ctx, currency.NewBTCUSDT(), 3000000123, "")
	require.ErrorIs(t, err, request.ErrDelayNotAllowed, "USDTMarginedFuturesGetOrderData must draw on the IP pool")
	err = ex.CancelOrder(ctx, &order.Cancel{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "3000000123"})
	require.NoError(t, err, "CancelOrder must not wait on the IP pool")
	_, err = ex.ModifyOrder(ctx, &order.Modify{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "3000000123", Side: order.Sell, Price: 1, Amount: 1})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "ModifyOrder must refuse the algo order without waiting on the IP pool")
	assert.Equal(t, 5, sent(), "CancelOrder and ModifyOrder should send every expected request")
}

func TestCoinMarginedAlgoOrders(t *testing.T) {
	t.Parallel()
	// COIN-M stop, take profit and trailing stop orders are algo orders since the CM-UM integration, which SubmitOrder
	// identifies by algo order ID, while an untagged ID is looked up as both an order's and an algo order's, as on USDⓈ-M,
	// and so is one with a stop type, as COIN-M also takes such orders through New Order until Binance's cut-over
	const (
		unknownOrder  = `{"code":-2011,"msg":"Unknown order sent."}`
		notFound      = `{"code":-2013,"msg":"Order does not exist."}`
		algoCancelled = `{"algoId":3100001,"clientAlgoId":"gct-cm-stop","code":"200","msg":"success"}`
		allCancelled  = `{"code":200,"msg":"The operation of cancel all open order is done."}`
	)
	pair := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	algo := func(symbol string, id uint64) string {
		return fmt.Sprintf(`{"algoId":%d,"clientAlgoId":"gct-cm-stop","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":%q,"side":"SELL","positionSide":"BOTH","timeInForce":"GTE_GTC","quantity":"1","algoStatus":"FINISHED","triggerPrice":"70000","price":"0","icebergQuantity":"null","reduceOnly":true,"createTime":1744120800000,"updateTime":1744120900000,"actualOrderId":"1573346965","actualPrice":"69990","actualQty":"1"}`, id, symbol)
	}
	placed := func(id uint64, clientAlgoID, orderType string) string {
		return fmt.Sprintf(`{"algoId":%d,"clientAlgoId":%q,"algoType":"CONDITIONAL","orderType":%q,"symbol":"BTCUSD_PERP","side":"SELL","algoStatus":"NEW","createTime":1744120800000,"updateTime":1744120800100}`, id, clientAlgoID, orderType)
	}

	t.Run("SubmitOrder", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			submit order.Submit
			step   cancelStep
			exp    *order.SubmitResponse
		}{
			{
				submit: order.Submit{Type: order.StopMarket, Side: order.Sell, Amount: 1, TriggerPrice: 70000, TriggerPriceType: order.MarkPrice, ReduceOnly: true, ClientOrderID: "gct-cm-stop"},
				step: cancelStep{http.MethodPost, "/dapi/v1/algoOrder", map[string]string{
					"algoType": "CONDITIONAL", "symbol": "BTCUSD_PERP", "side": "SELL", "type": "STOP_MARKET", "quantity": "1", "triggerPrice": "70000",
					"workingType": "MARK_PRICE", "reduceOnly": "true", "clientAlgoId": "gct-cm-stop", "newOrderRespType": "RESULT", "price": "", "timeInForce": "",
				}, placed(3100001, "gct-cm-stop", "STOP_MARKET")},
				exp: &order.SubmitResponse{
					Exchange: "", Type: order.StopMarket, Side: order.Sell, Pair: pair, AssetType: asset.CoinMarginedFutures, ReduceOnly: true, Amount: 1, TriggerPrice: 70000,
					ClientOrderID: "gct-cm-stop", Date: time.UnixMilli(1744120800000), LastUpdated: time.UnixMilli(1744120800100), Status: order.New, OrderID: "algo-3100001",
				},
			},
			{
				submit: order.Submit{Type: order.TakeProfitLimit, Side: order.Sell, Amount: 1, Price: 95000, TriggerPrice: 94000},
				step: cancelStep{http.MethodPost, "/dapi/v1/algoOrder", map[string]string{
					"algoType": "CONDITIONAL", "symbol": "BTCUSD_PERP", "side": "SELL", "type": "TAKE_PROFIT", "quantity": "1", "price": "95000", "triggerPrice": "94000", "timeInForce": "GTC",
				}, placed(3100002, "gct-cm-take-profit", "TAKE_PROFIT")},
				exp: &order.SubmitResponse{
					Type: order.TakeProfitLimit, Side: order.Sell, Pair: pair, AssetType: asset.CoinMarginedFutures, Amount: 1, Price: 95000, TriggerPrice: 94000,
					ClientOrderID: "gct-cm-take-profit", Date: time.UnixMilli(1744120800000), LastUpdated: time.UnixMilli(1744120800100), Status: order.New, OrderID: "algo-3100002",
				},
			},
			{
				submit: order.Submit{Type: order.TrailingStop, Side: order.Sell, Amount: 1, TriggerPrice: 92000, TrackingMode: order.Percentage, TrackingValue: 1.5},
				step: cancelStep{http.MethodPost, "/dapi/v1/algoOrder", map[string]string{
					"algoType": "CONDITIONAL", "symbol": "BTCUSD_PERP", "side": "SELL", "type": "TRAILING_STOP_MARKET", "quantity": "1", "callbackRate": "1.5", "activatePrice": "92000", "triggerPrice": "",
				}, placed(3100003, "gct-cm-trailing", "TRAILING_STOP_MARKET")},
				exp: &order.SubmitResponse{
					Type: order.TrailingStop, Side: order.Sell, Pair: pair, AssetType: asset.CoinMarginedFutures, Amount: 1, TriggerPrice: 92000,
					ClientOrderID: "gct-cm-trailing", Date: time.UnixMilli(1744120800000), LastUpdated: time.UnixMilli(1744120800100), Status: order.New, OrderID: "algo-3100003",
				},
			},
		} {
			t.Run(tc.submit.Type.String(), func(t *testing.T) {
				t.Parallel()
				ex, sent := newCancelStepsExchange(t, []cancelStep{tc.step})
				tc.submit.Exchange, tc.submit.AssetType, tc.submit.Pair = ex.Name, asset.CoinMarginedFutures, pair
				tc.exp.Exchange = ex.Name
				resp, err := ex.SubmitOrder(t.Context(), &tc.submit)
				require.NoErrorf(t, err, "SubmitOrder must place a COIN-M %s order as an algo order", tc.submit.Type)
				assert.Equalf(t, tc.exp, resp, "SubmitOrder should identify a COIN-M %s order by its algo order ID", tc.submit.Type)
				assert.Equalf(t, 1, sent(), "SubmitOrder should send one request for a COIN-M %s order", tc.submit.Type)
			})
		}
	})

	t.Run("CancelOrder", func(t *testing.T) {
		t.Parallel()
		coinMarginedOrder := func(reply string) cancelStep {
			return cancelStep{http.MethodDelete, "/dapi/v1/order", map[string]string{"orderId": "3100001", "symbol": "BTCUSD_PERP"}, reply}
		}
		for _, tc := range []struct {
			name   string
			cancel order.Cancel
			steps  []cancelStep
			errMsg string
		}{
			{
				name:   "stop order",
				cancel: order.Cancel{Type: order.StopMarket, OrderID: "3100001"},
				steps: []cancelStep{
					{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001"}, algo("BTCUSD_PERP", 3100001)},
					{http.MethodGet, "/dapi/v1/order", map[string]string{"orderId": "3100001", "symbol": "BTCUSD_PERP"}, notFound},
					{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001", "symbol": ""}, algoCancelled},
				},
			},
			{
				name:   "algo order by algo order ID",
				cancel: order.Cancel{OrderID: "algo-3100001"},
				steps:  []cancelStep{{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001", "symbol": ""}, algoCancelled}},
			},
			{
				name:   "untyped algo order",
				cancel: order.Cancel{OrderID: "3100001"},
				steps: []cancelStep{
					{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001"}, algo("BTCUSD_PERP", 3100001)},
					{http.MethodGet, "/dapi/v1/order", map[string]string{"orderId": "3100001", "symbol": "BTCUSD_PERP"}, notFound},
					{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001"}, algoCancelled},
				},
			},
			{
				name:   "untyped algo order of another symbol",
				cancel: order.Cancel{OrderID: "3100001"},
				steps:  []cancelStep{{http.MethodGet, "/dapi/v1/algoOrder", nil, algo("ETHUSD_PERP", 3100001)}, coinMarginedOrder(unknownOrder)},
				errMsg: "Unknown order sent.",
			},
			{
				name:   "limit order",
				cancel: order.Cancel{Type: order.Limit, OrderID: "3100001"},
				steps:  []cancelStep{coinMarginedOrder(unknownOrder)},
				errMsg: "Unknown order sent.",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, sent := newCancelStepsExchange(t, tc.steps)
				tc.cancel.AssetType, tc.cancel.Pair = asset.CoinMarginedFutures, pair
				err := ex.CancelOrder(t.Context(), &tc.cancel)
				if tc.errMsg == "" {
					assert.NoErrorf(t, err, "CancelOrder should cancel the %s", tc.name)
				} else {
					assert.ErrorContainsf(t, err, tc.errMsg, "CancelOrder should return the %s's cancellation error", tc.name)
				}
				assert.Equalf(t, len(tc.steps), sent(), "CancelOrder should send every expected request for the %s", tc.name)
			})
		}
	})

	t.Run("CancelBatchOrders", func(t *testing.T) {
		t.Parallel()
		ex, sent := newCancelStepsExchange(t, []cancelStep{
			{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001"}, algo("BTCUSD_PERP", 3100001)},
			{http.MethodGet, "/dapi/v1/order", map[string]string{"orderId": "3100001", "symbol": "BTCUSD_PERP"}, notFound},
			{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100002"}, algo("BTCUSD_PERP", 3100002)},
			{http.MethodGet, "/dapi/v1/order", map[string]string{"orderId": "3100002", "symbol": "BTCUSD_PERP"}, notFound},
			{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001"}, algoCancelled},
			{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100002"}, algoCancelled},
		})
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.CoinMarginedFutures, Pair: pair, Type: order.StopMarket, OrderID: "3100001"},
			{AssetType: asset.CoinMarginedFutures, Pair: pair, OrderID: "3100002"},
		})
		require.NoError(t, err, "CancelBatchOrders must not error")
		assert.Equal(t, map[string]string{"3100001": "CANCELED", "3100002": "CANCELED"}, resp.Status, "CancelBatchOrders should cancel COIN-M algo orders")
		assert.Equal(t, 6, sent(), "CancelBatchOrders should send every expected request")
	})

	t.Run("CancelAllOrders", func(t *testing.T) {
		t.Parallel()
		ex, sent := newCancelStepsExchange(t, []cancelStep{
			{http.MethodGet, "/dapi/v1/openOrders", map[string]string{"symbol": ""}, `[{"symbol":"BTCUSD_PERP","orderId":1573346962}]`},
			{http.MethodGet, "/dapi/v1/openAlgoOrders", map[string]string{"symbol": ""}, `[{"symbol":"BTCUSD_PERP","algoId":3100001}]`},
			{http.MethodDelete, "/dapi/v1/allOpenOrders", map[string]string{"symbol": "BTCUSD_PERP"}, allCancelled},
			{http.MethodDelete, "/dapi/v1/algoOpenOrders", map[string]string{"symbol": "BTCUSD_PERP"}, allCancelled},
			{http.MethodDelete, "/dapi/v1/allOpenOrders", map[string]string{"symbol": "BTCUSD_PERP"}, allCancelled},
			{http.MethodDelete, "/dapi/v1/algoOpenOrders", map[string]string{"symbol": "BTCUSD_PERP"}, allCancelled},
		})
		_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.CoinMarginedFutures})
		require.NoError(t, err, "CancelAllOrders must cancel every symbol's orders and algo orders")
		_, err = ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.CoinMarginedFutures, Pair: pair})
		require.NoError(t, err, "CancelAllOrders must cancel a pair's orders and algo orders")
		assert.Equal(t, 6, sent(), "CancelAllOrders should send every expected request")
	})

	t.Run("GetOrderInfo", func(t *testing.T) {
		t.Parallel()
		ex, sent := newCancelStepsExchange(t, []cancelStep{
			{http.MethodGet, "/dapi/v1/order", map[string]string{"orderId": "3100001", "symbol": "BTCUSD_PERP"}, notFound},
			{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001"}, algo("BTCUSD_PERP", 3100001)},
		})
		d, err := ex.GetOrderInfo(t.Context(), "3100001", pair, asset.CoinMarginedFutures)
		require.NoError(t, err, "GetOrderInfo must look an ID no order has up as an algo ID")
		exp := &order.Detail{
			TimeInForce: order.GoodTillCancel, ReduceOnly: true, Amount: 1, TriggerPrice: 70000, AverageExecutedPrice: 69990, ExecutedAmount: 1,
			Exchange: ex.Name, OrderID: "algo-3100001", ClientOrderID: "gct-cm-stop", Type: order.StopMarket, Side: order.Sell, Status: order.Closed,
			AssetType: asset.CoinMarginedFutures, Date: time.UnixMilli(1744120800000), LastUpdated: time.UnixMilli(1744120900000), Pair: pair,
		}
		assert.Equal(t, exp, d, "GetOrderInfo should return the COIN-M algo order")
		assert.Equal(t, 2, sent(), "GetOrderInfo should send every expected request")
	})

	t.Run("conditional orders placed as orders", func(t *testing.T) {
		t.Parallel()
		// Conditional orders placed before the move to algo orders are orders, which a request for conditional orders
		// lists beside the algo orders
		const plainStop = `[{"orderId":1573346966,"clientOrderId":"gct-cm-plain-stop","symbol":"BTCUSD_PERP","side":"SELL","type":"STOP_MARKET","origType":"STOP_MARKET","status":"NEW","timeInForce":"GTC","origQty":"1","executedQty":"0","price":"0","stopPrice":"71000","time":1744120700000,"updateTime":1744120700000}]`
		ex, sent := newCancelStepsExchange(t, []cancelStep{
			{http.MethodGet, "/dapi/v1/openOrders", map[string]string{"symbol": "BTCUSD_PERP"}, plainStop},
			{http.MethodGet, "/dapi/v1/openAlgoOrders", map[string]string{"symbol": "BTCUSD_PERP"}, `[` + algo("BTCUSD_PERP", 3100001) + `]`},
			{http.MethodGet, "/dapi/v1/allOrders", map[string]string{"symbol": "BTCUSD_PERP", "limit": "100"}, plainStop},
			{http.MethodGet, "/dapi/v1/allAlgoOrders", map[string]string{"symbol": "BTCUSD_PERP", "limit": "100"}, `[` + algo("BTCUSD_PERP", 3100001) + `]`},
		})
		req := &order.MultiOrderRequest{AssetType: asset.CoinMarginedFutures, Pairs: currency.Pairs{pair}, Type: order.StopMarket, Side: order.AnySide}
		active, err := ex.GetActiveOrders(t.Context(), req)
		require.NoError(t, err, "GetActiveOrders must not error")
		history, err := ex.GetOrderHistory(t.Context(), req)
		require.NoError(t, err, "GetOrderHistory must not error")
		for name, orders := range map[string]order.FilteredOrders{"GetActiveOrders": active, "GetOrderHistory": history} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				ids := make([]string, len(orders))
				for i := range orders {
					ids[i] = orders[i].OrderID
				}
				assert.Equalf(t, []string{"1573346966", "algo-3100001"}, ids, "%s should list conditional orders and algo orders", name)
			})
		}
		assert.Equal(t, 4, sent(), "the listings should send every expected request")
	})

	t.Run("conditional orders placed as orders cancelled as listed", func(t *testing.T) {
		t.Parallel()
		// A stop, take profit or trailing stop placed through New Order is listed by its order ID and type, which cancel
		// it, while another symbol's algo order with that number is left alone and the pair's is refused as ambiguous
		plainStop := func(orderType, status string) string {
			return fmt.Sprintf(`{"orderId":1573346966,"clientOrderId":"gct-cm-plain-stop","symbol":"BTCUSD_PERP","side":"SELL","type":%q,"origType":%q,"status":%q,"timeInForce":"GTC","origQty":"1","executedQty":"0","price":"0","stopPrice":"71000","time":1744120700000,"updateTime":1744120700000}`, orderType, orderType, status)
		}
		const limitOrder = `{"orderId":1573346960,"clientOrderId":"gct-cm-limit","symbol":"BTCUSD_PERP","side":"BUY","type":"LIMIT","origType":"LIMIT","status":"CANCELED","timeInForce":"GTC","origQty":"1","executedQty":"0","price":"60000","updateTime":1744120800000}`
		algoLookup := func(reply string) cancelStep {
			return cancelStep{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"algoId": "1573346966", "clientAlgoId": ""}, reply}
		}
		plainStopRequest := func(method, orderType, status string) cancelStep {
			return cancelStep{method, "/dapi/v1/order", map[string]string{"orderId": "1573346966", "symbol": "BTCUSD_PERP"}, plainStop(orderType, status)}
		}
		batchCancel := func(orderType string) cancelStep {
			return cancelStep{http.MethodDelete, "/dapi/v1/batchOrders", map[string]string{"orderIdList": "[1573346966]", "symbol": "BTCUSD_PERP"}, `[` + plainStop(orderType, "CANCELED") + `]`}
		}
		for _, orderType := range []string{"STOP", "STOP_MARKET", "TAKE_PROFIT", "TAKE_PROFIT_MARKET", "TRAILING_STOP_MARKET"} {
			for _, tc := range []struct {
				name  string
				batch bool
				steps []cancelStep
				err   error
			}{
				{name: "no algo order has the ID", steps: []cancelStep{algoLookup(notFound), plainStopRequest(http.MethodDelete, orderType, "CANCELED")}},
				{name: "another symbol's algo order has the ID", steps: []cancelStep{algoLookup(algo("ETHUSD_PERP", 1573346966)), plainStopRequest(http.MethodDelete, orderType, "CANCELED")}},
				{name: "the pair's algo order has the ID too", steps: []cancelStep{algoLookup(algo("BTCUSD_PERP", 1573346966)), plainStopRequest(http.MethodGet, orderType, "NEW")}, err: errAmbiguousOrderID},
				{name: "batch, no algo order has the ID", batch: true, steps: []cancelStep{algoLookup(notFound), batchCancel(orderType)}},
				{name: "batch, another symbol's algo order has the ID", batch: true, steps: []cancelStep{algoLookup(algo("ETHUSD_PERP", 1573346966)), batchCancel(orderType)}},
				{name: "batch, the pair's algo order has the ID too", batch: true, steps: []cancelStep{algoLookup(algo("BTCUSD_PERP", 1573346966)), plainStopRequest(http.MethodGet, orderType, "NEW")}, err: errAmbiguousOrderID},
			} {
				t.Run(orderType+" "+tc.name, func(t *testing.T) {
					t.Parallel()
					steps := append([]cancelStep{
						{http.MethodGet, "/dapi/v1/openOrders", map[string]string{"symbol": "BTCUSD_PERP"}, `[` + plainStop(orderType, "NEW") + `]`},
						{http.MethodGet, "/dapi/v1/openAlgoOrders", map[string]string{"symbol": "BTCUSD_PERP"}, `[]`},
					}, tc.steps...)
					ex, sent := newCancelStepsExchange(t, steps)
					listed, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.CoinMarginedFutures, Pairs: currency.Pairs{pair}, Type: order.AnyType, Side: order.AnySide})
					require.NoError(t, err, "GetActiveOrders must not error")
					require.Len(t, listed, 1, "GetActiveOrders must list the order")
					require.Truef(t, isConditionalOrderType(listed[0].Type), "GetActiveOrders must list the %s order with its conditional type", orderType)
					cancel, err := listed[0].DeriveCancel()
					require.NoError(t, err, "DeriveCancel must not error")
					if tc.batch {
						resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{*cancel})
						require.ErrorIs(t, err, tc.err, "CancelBatchOrders must return the expected error")
						require.NotNil(t, resp, "CancelBatchOrders must return the order's outcome")
						if tc.err == nil {
							assert.Equal(t, map[string]string{"1573346966": "CANCELED"}, resp.Status, "CancelBatchOrders should report the order cancelled")
						} else {
							assert.Len(t, resp.Status, 1, "CancelBatchOrders should report the order alone")
							assert.Contains(t, resp.Status["1573346966"], tc.err.Error(), "CancelBatchOrders should report why the order was not cancelled")
						}
					} else {
						assert.ErrorIs(t, ex.CancelOrder(t.Context(), cancel), tc.err, "CancelOrder should cancel the order unless the pair's algo order has its ID")
					}
					assert.Equal(t, len(steps), sent(), "the cancellation should send every expected request")
				})
			}
		}

		// A stop or take profit type that leaves open which of Binance's types is meant is routed alike, while a market
		// type names an order
		for _, tc := range []struct {
			cancelType order.Type
			steps      []cancelStep
		}{
			{cancelType: order.Stop, steps: []cancelStep{algoLookup(notFound), plainStopRequest(http.MethodDelete, "STOP_MARKET", "CANCELED")}},
			{cancelType: order.TakeProfit, steps: []cancelStep{algoLookup(notFound), plainStopRequest(http.MethodDelete, "TAKE_PROFIT_MARKET", "CANCELED")}},
			{cancelType: order.Market, steps: []cancelStep{plainStopRequest(http.MethodDelete, "MARKET", "CANCELED")}},
		} {
			t.Run(tc.cancelType.String()+" type", func(t *testing.T) {
				t.Parallel()
				ex, sent := newCancelStepsExchange(t, tc.steps)
				err := ex.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.CoinMarginedFutures, Pair: pair, Type: tc.cancelType, OrderID: "1573346966"})
				require.NoErrorf(t, err, "CancelOrder must cancel the order named with a %s type", tc.cancelType)
				assert.Equalf(t, len(tc.steps), sent(), "CancelOrder should send every expected request for a %s type", tc.cancelType)
			})
		}

		t.Run("stop named by its client order ID", func(t *testing.T) {
			t.Parallel()
			lookup := cancelStep{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-cm-plain-stop", "algoId": ""}, notFound}
			steps := []cancelStep{
				lookup,
				{http.MethodDelete, "/dapi/v1/order", map[string]string{"origClientOrderId": "gct-cm-plain-stop", "orderId": "", "symbol": "BTCUSD_PERP"}, plainStop("STOP_MARKET", "CANCELED")},
				lookup,
				{http.MethodDelete, "/dapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["gct-cm-plain-stop"]`, "orderIdList": "", "symbol": "BTCUSD_PERP"}, `[` + plainStop("STOP_MARKET", "CANCELED") + `]`},
			}
			ex, sent := newCancelStepsExchange(t, steps)
			c := order.Cancel{AssetType: asset.CoinMarginedFutures, Pair: pair, Type: order.StopMarket, ClientOrderID: "gct-cm-plain-stop"}
			require.NoError(t, ex.CancelOrder(t.Context(), &c), "CancelOrder must cancel the order by its client order ID")
			resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{c})
			require.NoError(t, err, "CancelBatchOrders must cancel the order by its client order ID")
			assert.Equal(t, map[string]string{"gct-cm-plain-stop": "CANCELED"}, resp.Status, "CancelBatchOrders should report the order cancelled by its client order ID")
			assert.Equal(t, len(steps), sent(), "CancelOrder and CancelBatchOrders should send every expected request")
		})

		t.Run("batch, an order beside the stop", func(t *testing.T) {
			t.Parallel()
			steps := []cancelStep{
				algoLookup(notFound),
				{http.MethodDelete, "/dapi/v1/batchOrders", map[string]string{"orderIdList": "[1573346960,1573346966]", "symbol": "BTCUSD_PERP"}, `[` + limitOrder + `,` + plainStop("STOP_MARKET", "CANCELED") + `]`},
			}
			ex, sent := newCancelStepsExchange(t, steps)
			resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
				{AssetType: asset.CoinMarginedFutures, Pair: pair, Type: order.Limit, OrderID: "1573346960"},
				{AssetType: asset.CoinMarginedFutures, Pair: pair, Type: order.StopMarket, OrderID: "1573346966"},
			})
			require.NoError(t, err, "CancelBatchOrders must cancel both orders")
			assert.Equal(t, map[string]string{"1573346960": "CANCELED", "1573346966": "CANCELED"}, resp.Status, "CancelBatchOrders should report both orders cancelled")
			assert.Equal(t, len(steps), sent(), "CancelBatchOrders should look up only the stop")
		})

		t.Run("algo order ID with a stop type", func(t *testing.T) {
			t.Parallel()
			cancelAlgo := cancelStep{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001"}, algoCancelled}
			ex, sent := newCancelStepsExchange(t, []cancelStep{cancelAlgo, cancelAlgo})
			c := order.Cancel{AssetType: asset.CoinMarginedFutures, Pair: pair, Type: order.StopMarket, OrderID: "algo-3100001"}
			require.NoError(t, ex.CancelOrder(t.Context(), &c), "CancelOrder must cancel the algo order")
			resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{c})
			require.NoError(t, err, "CancelBatchOrders must cancel the algo order")
			assert.Equal(t, map[string]string{"algo-3100001": "CANCELED"}, resp.Status, "CancelBatchOrders should report the algo order cancelled")
			assert.Equal(t, 2, sent(), "an algo order ID should go straight to Cancel Algo Order")
		})

		t.Run("batch, stop whose ID cannot be parsed beside an order", func(t *testing.T) {
			t.Parallel()
			steps := []cancelStep{{http.MethodDelete, "/dapi/v1/batchOrders", map[string]string{"orderIdList": "[1573346960]", "symbol": "BTCUSD_PERP"}, `[` + limitOrder + `]`}}
			ex, sent := newCancelStepsExchange(t, steps)
			resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
				{AssetType: asset.CoinMarginedFutures, Pair: pair, Type: order.StopMarket, OrderID: "x77"},
				{AssetType: asset.CoinMarginedFutures, Pair: pair, Type: order.Limit, OrderID: "1573346960"},
			})
			assert.ErrorIs(t, err, errInvalidOrderID, "CancelBatchOrders should return the stop's parse error")
			require.NotNil(t, resp, "CancelBatchOrders must return each order's outcome")
			assert.Equal(t, map[string]string{"1573346960": "CANCELED", "x77": `invalid order ID "x77": strconv.ParseUint: parsing "x77": invalid syntax`}, resp.Status, "CancelBatchOrders should fail the stop alone")
			assert.Equal(t, len(steps), sent(), "CancelBatchOrders should send nothing for the stop")
		})

		t.Run("batch, the stop's number as another algo order's client algo ID", func(t *testing.T) {
			t.Parallel()
			// Successes are kept apart by kind of ID, so an algo order named by a client algo ID with the text of the stop's
			// number reports its own failure after the stop's algo order is cancelled
			const (
				namedAlgo = `{"algoId":3100002,"clientAlgoId":"1573346966","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSD_PERP","side":"SELL","positionSide":"BOTH","quantity":"1","algoStatus":"NEW","triggerPrice":"70000"}`
				cancelled = `{"algoId":1573346966,"clientAlgoId":"gct-cm-stop","code":"200","msg":"success"}`
			)
			steps := []cancelStep{
				algoLookup(algo("BTCUSD_PERP", 1573346966)),
				{http.MethodGet, "/dapi/v1/order", map[string]string{"orderId": "1573346966", "origClientOrderId": "", "symbol": "BTCUSD_PERP"}, notFound},
				{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"clientAlgoId": "1573346966", "algoId": ""}, namedAlgo},
				{http.MethodGet, "/dapi/v1/order", map[string]string{"origClientOrderId": "1573346966", "orderId": "", "symbol": "BTCUSD_PERP"}, notFound},
				{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "1573346966"}, cancelled},
				{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100002"}, unknownOrder},
			}
			ex, sent := newCancelStepsExchange(t, steps)
			resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
				{AssetType: asset.CoinMarginedFutures, Pair: pair, Type: order.StopMarket, OrderID: "1573346966"},
				{AssetType: asset.CoinMarginedFutures, Pair: pair, Type: order.StopMarket, ClientOrderID: "1573346966"},
			})
			assert.ErrorContains(t, err, "Unknown order sent.", "CancelBatchOrders should return the second algo order's rejection")
			require.NotNil(t, resp, "CancelBatchOrders must return each order's outcome")
			assert.Len(t, resp.Status, 1, "CancelBatchOrders should report one status for the shared text")
			assert.Contains(t, resp.Status["1573346966"], "Unknown order sent.", "CancelBatchOrders should report the second algo order's rejection")
			assert.Equal(t, len(steps), sent(), "CancelBatchOrders should send every expected request")
		})
	})
}

func TestCancelCoinMarginedAlgoOrderLookupRate(t *testing.T) {
	t.Parallel()
	// The lookups an untyped COIN-M cancellation makes draw on the cancel pool, as the cancellation does, rather than
	// queue behind order book snapshots on the IP pool
	ex, sent := newCancelStepsExchange(t, []cancelStep{
		{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001"}, `{"algoId":3100001,"clientAlgoId":"gct-cm-stop","symbol":"BTCUSD_PERP","algoStatus":"NEW"}`},
		{http.MethodGet, "/dapi/v1/order", map[string]string{"orderId": "3100001"}, `{"code":-2013,"msg":"Order does not exist."}`},
		{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001"}, `{"algoId":3100001,"clientAlgoId":"gct-cm-stop","code":"200","msg":"success"}`},
	})
	require.NoError(t, ex.EnableRateLimiter(), "EnableRateLimiter must not error")
	exhausted := request.NewRateLimitWithWeight(time.Hour, 1, 1)
	require.NoError(t, exhausted.RateLimit(t.Context()), "RateLimit must take the limiter's only token")
	definitions := ex.Requester.GetRateLimiterDefinitions()
	for epl, cost := range endpointCosts {
		if cost.pool == derivativesIPPool {
			definitions[epl] = exhausted
		}
	}
	ctx := request.WithDelayNotAllowed(t.Context())
	_, err := ex.CoinMarginedFuturesGetOrderData(ctx, testCoinMarginedFuturesPair, 3100001, "")
	require.ErrorIs(t, err, request.ErrDelayNotAllowed, "CoinMarginedFuturesGetOrderData must draw on the IP pool")
	err = ex.CancelOrder(ctx, &order.Cancel{AssetType: asset.CoinMarginedFutures, Pair: currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), OrderID: "3100001"})
	require.NoError(t, err, "CancelOrder must not wait on the IP pool")
	assert.Equal(t, 3, sent(), "CancelOrder should send every expected request")
}

func TestCountdownCancelAllRates(t *testing.T) {
	t.Parallel()
	// A countdown heartbeat reaching Binance after its countdown has run out has every order it covers cancelled, so
	// heartbeats and countdown settings wait on neither market data nor cancellations
	ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply, ok := map[string]string{
			"/fapi/v1/countdownCancelAll":          `{"symbol":"BTCUSDT","countdownTime":"0"}`,
			"/dapi/v1/countdownCancelAll":          `{"symbol":"BTCUSD_PERP","countdownTime":"0"}`,
			"/eapi/v1/countdownCancelAllHeartBeat": `{"underlyings":["BTCUSDT"]}`,
			"/eapi/v1/countdownCancelAll":          `{"underlying":"BTCUSDT","countdownTime":0}`,
		}[r.URL.Path]
		if !assert.Truef(t, ok && r.Method == http.MethodPost, "%s %s should not be requested", r.Method, r.URL.Path) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	})
	require.NoError(t, ex.EnableRateLimiter(), "EnableRateLimiter must not error")
	exhausted := request.NewRateLimitWithWeight(time.Hour, 1, 1)
	require.NoError(t, exhausted.RateLimit(t.Context()), "RateLimit must take the limiter's only token")
	definitions := ex.Requester.GetRateLimiterDefinitions()
	for epl, cost := range endpointCosts {
		if cost.pool != derivativesHeartbeatPool && cost.pool != optionsHeartbeatPool {
			definitions[epl] = exhausted
		}
	}
	ctx := request.WithDelayNotAllowed(t.Context())
	_, err := ex.USDTMarginedFuturesAutoCancelAllOpenOrders(ctx, currency.NewBTCUSDT(), time.Minute)
	assert.NoError(t, err, "USDTMarginedFuturesAutoCancelAllOpenOrders should draw on the heartbeat pool alone")
	_, err = ex.CoinMarginedFuturesAutoCancelAllOpenOrders(ctx, currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), time.Minute)
	assert.NoError(t, err, "CoinMarginedFuturesAutoCancelAllOpenOrders should draw on the heartbeat pool alone")
	_, err = ex.SendOptionsAutoCancelAllOpenOrdersHeartbeat(ctx, []string{"BTCUSDT"})
	assert.NoError(t, err, "SendOptionsAutoCancelAllOpenOrdersHeartbeat should draw on the heartbeat pool alone")
	_, err = ex.SetOptionsAutoCancelAllOpenOrders(ctx, "BTCUSDT", 0)
	assert.NoError(t, err, "SetOptionsAutoCancelAllOpenOrders should turn the countdown off on the heartbeat pool alone")

	// Heartbeats beyond what a heartbeat pool can take at once queue on the IP pool as they did before the pools, rather
	// than wait for the share to refill, which heartbeats sent faster than it refills would do ever longer
	ex = newTradingTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{}`))
		assert.NoError(t, err, "Write should not error")
	})
	require.NoError(t, ex.EnableRateLimiter(), "EnableRateLimiter must not error")
	slow := request.NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
	require.NoError(t, slow.RateLimit(t.Context()), "RateLimit must take the limiter's only token")
	definitions = ex.Requester.GetRateLimiterDefinitions()
	for epl, cost := range endpointCosts {
		switch cost.pool {
		case derivativesIPPool, optionsIPPool:
			definitions[epl] = slow
		default:
			definitions[epl] = exhausted
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	_, err = ex.USDTMarginedFuturesAutoCancelAllOpenOrders(ctx, currency.NewBTCUSDT(), time.Minute)
	assert.NoError(t, err, "USDTMarginedFuturesAutoCancelAllOpenOrders should queue on the IP pool beyond the heartbeat pool")
	_, err = ex.CoinMarginedFuturesAutoCancelAllOpenOrders(ctx, currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), time.Minute)
	assert.NoError(t, err, "CoinMarginedFuturesAutoCancelAllOpenOrders should queue on the IP pool beyond the heartbeat pool")
	_, err = ex.SendOptionsAutoCancelAllOpenOrdersHeartbeat(ctx, []string{"BTCUSDT"})
	assert.NoError(t, err, "SendOptionsAutoCancelAllOpenOrdersHeartbeat should queue on the IP pool beyond the heartbeat pool")
	_, err = ex.SetOptionsAutoCancelAllOpenOrders(ctx, "BTCUSDT", 0)
	assert.NoError(t, err, "SetOptionsAutoCancelAllOpenOrders should queue on the IP pool beyond the heartbeat pool")
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must reject an empty pair")
	_, err = e.GetOrderInfo(t.Context(), "", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must reject an empty order ID")
	_, err = e.GetOrderInfo(t.Context(), "abc", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, errInvalidOrderID, "GetOrderInfo must reject an invalid order ID")
	_, err = e.GetOrderInfo(t.Context(), "algo-1", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, errInvalidOrderID, "GetOrderInfo must reject an algo order ID on spot")
	_, err = e.GetOrderInfo(t.Context(), "algo-abc", currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, errInvalidOrderID, "GetOrderInfo must reject an invalid algo order ID")
	_, err = e.GetOrderInfo(t.Context(), "1", currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	for _, tc := range []struct {
		name    string
		orderID string
		a       asset.Item
		exp     *order.Detail
		err     error
	}{
		{
			name:    "spot",
			orderID: "41",
			a:       asset.Spot,
			exp: &order.Detail{
				TimeInForce:          order.GoodTillCancel,
				Price:                80000,
				Amount:               0.001,
				AverageExecutedPrice: 79990,
				ExecutedAmount:       0.0005,
				RemainingAmount:      0.0005,
				Cost:                 39.995,
				CostAsset:            currency.USDT,
				Exchange:             "Binance",
				OrderID:              "41",
				ClientOrderID:        "gct-spot-limit",
				Type:                 order.Limit,
				Side:                 order.Buy,
				Status:               order.PartiallyFilled,
				AssetType:            asset.Spot,
				Date:                 time.Unix(1791284812, 0),
				LastUpdated:          time.UnixMilli(1791284812500),
				Pair:                 currency.NewBTCUSDT(),
			},
		},
		{
			name:    "spot unknown status",
			orderID: "999",
			a:       asset.Spot,
			err:     errUnknownOrderStatus,
			exp:     nil,
		},
		{
			name:    "margin cross",
			orderID: "28457",
			a:       asset.Margin,
			exp: &order.Detail{
				TimeInForce:          order.GoodTillCancel,
				Price:                80000,
				Amount:               0.001,
				AverageExecutedPrice: 79990,
				ExecutedAmount:       0.001,
				Cost:                 79.99,
				CostAsset:            currency.USDT,
				Exchange:             "Binance",
				OrderID:              "28457",
				ClientOrderID:        "gct-margin-limit",
				Type:                 order.Limit,
				Side:                 order.Buy,
				Status:               order.Filled,
				AssetType:            asset.Margin,
				Date:                 time.Unix(1791284814, 0),
				LastUpdated:          time.UnixMilli(1791284814200),
				Pair:                 currency.NewBTCUSDT(),
				MarginType:           margin.Multi,
			},
		},
		{
			name:    "margin isolated",
			orderID: "112233424",
			a:       asset.Margin,
			exp: &order.Detail{
				TimeInForce:          order.GoodTillCancel,
				Price:                75800,
				Amount:               0.0005,
				TriggerPrice:         75900,
				AverageExecutedPrice: 75800,
				ExecutedAmount:       0.0002,
				RemainingAmount:      0.0003,
				Cost:                 15.16,
				CostAsset:            currency.USDT,
				Exchange:             "Binance",
				OrderID:              "112233424",
				ClientOrderID:        "ZwfQzuDIGpceVhKW5DvCmO",
				Type:                 order.StopLimit,
				Side:                 order.Sell,
				Status:               order.PartiallyFilled,
				AssetType:            asset.Margin,
				Date:                 time.Unix(1744150000, 0),
				LastUpdated:          time.Unix(1744150500, 0),
				Pair:                 currency.NewBTCUSDT(),
				MarginType:           margin.Isolated,
			},
		},
		{
			name:    "USDⓈ-M",
			orderID: "123",
			a:       asset.USDTMarginedFutures,
			exp: &order.Detail{
				TimeInForce:          order.GoodTillDay,
				ReduceOnly:           true,
				Price:                80000,
				Amount:               0.002,
				TriggerPrice:         79000,
				AverageExecutedPrice: 80000,
				ExecutedAmount:       0.001,
				RemainingAmount:      0.001,
				Cost:                 80,
				Exchange:             "Binance",
				OrderID:              "123",
				ClientOrderID:        "testOrder",
				Type:                 order.Limit,
				Side:                 order.Buy,
				Status:               order.PartiallyFilled,
				AssetType:            asset.USDTMarginedFutures,
				Date:                 time.Unix(1744118000, 0),
				LastUpdated:          time.Unix(1744118400, 0),
				Pair:                 currency.NewBTCUSDT(),
			},
		},
		{
			name:    "USDⓈ-M algo by algo order ID",
			orderID: "algo-2146760",
			a:       asset.USDTMarginedFutures,
			exp: &order.Detail{
				TimeInForce:          order.GoodTillCancel,
				ReduceOnly:           true,
				Price:                90000,
				Amount:               0.01,
				TriggerPrice:         90000,
				AverageExecutedPrice: 90000,
				ExecutedAmount:       0.01,
				Exchange:             "Binance",
				OrderID:              "algo-2146760",
				ClientOrderID:        "6B2I9XVcJpCjqPAJ4YoFX7",
				Type:                 order.TakeProfitLimit,
				Side:                 order.Sell,
				Status:               order.Closed,
				AssetType:            asset.USDTMarginedFutures,
				Date:                 time.UnixMilli(1750485492076),
				LastUpdated:          time.UnixMilli(1750485492076),
				Pair:                 currency.NewBTCUSDT(),
			},
		},
		{
			name:    "USDⓈ-M algo",
			orderID: "2146760",
			a:       asset.USDTMarginedFutures,
			exp: &order.Detail{
				TimeInForce:          order.GoodTillCancel,
				ReduceOnly:           true,
				Price:                90000,
				Amount:               0.01,
				TriggerPrice:         90000,
				AverageExecutedPrice: 90000,
				ExecutedAmount:       0.01,
				Exchange:             "Binance",
				OrderID:              "algo-2146760",
				ClientOrderID:        "6B2I9XVcJpCjqPAJ4YoFX7",
				Type:                 order.TakeProfitLimit,
				Side:                 order.Sell,
				Status:               order.Closed,
				AssetType:            asset.USDTMarginedFutures,
				Date:                 time.UnixMilli(1750485492076),
				LastUpdated:          time.UnixMilli(1750485492076),
				Pair:                 currency.NewBTCUSDT(),
			},
		},
		{
			name:    "USDⓈ-M missing",
			orderID: "1",
			a:       asset.USDTMarginedFutures,
			err:     errAPIResponse,
			exp:     nil,
		},
		{
			name:    "COIN-M",
			orderID: "1573346959",
			a:       asset.CoinMarginedFutures,
			exp: &order.Detail{
				TimeInForce:          order.GoodTillCancel,
				ReduceOnly:           true,
				Price:                9020.5,
				Amount:               10,
				TriggerPrice:         9020,
				AverageExecutedPrice: 9022.5,
				ExecutedAmount:       4,
				RemainingAmount:      6,
				Cost:                 0.04432225,
				Exchange:             "Binance",
				OrderID:              "1573346959",
				ClientOrderID:        "abc",
				Type:                 order.TrailingStop,
				Side:                 order.Buy,
				Status:               order.PartiallyFilled,
				AssetType:            asset.CoinMarginedFutures,
				Date:                 time.UnixMilli(1579276756075),
				LastUpdated:          time.UnixMilli(1579276756075),
				Pair:                 currency.NewPairWithDelimiter("BTCUSD", "PERP", "_"),
			},
		},
		{
			name:    "options",
			orderID: "4611875134427365377",
			a:       asset.Options,
			exp: &order.Detail{
				TimeInForce:          order.PostOnly,
				ReduceOnly:           true,
				Price:                1300,
				Amount:               0.01,
				AverageExecutedPrice: 1300,
				ExecutedAmount:       0.01,
				Cost:                 13,
				CostAsset:            currency.USDT,
				Exchange:             "Binance",
				OrderID:              "4611875134427365377",
				ClientOrderID:        "the-client-order-id",
				Type:                 order.Limit,
				Side:                 order.Sell,
				Status:               order.Filled,
				AssetType:            asset.Options,
				Date:                 time.Unix(1744120800, 0),
				LastUpdated:          time.Unix(1744120801, 0),
				Pair:                 currency.NewPairWithDelimiter("BTC", "260410-72000-P", "-"),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.GetOrderInfo(t.Context(), tc.orderID, assetToTradablePairMap[tc.a], tc.a)
			require.ErrorIs(t, err, tc.err, "GetOrderInfo must return the expected error")
			if mockTests {
				assert.Equal(t, tc.exp, resp, "GetOrderInfo should return the order")
			}
		})
	}

	t.Run("websocket API", func(t *testing.T) {
		t.Parallel()
		ex := newWsAPITestExchange(t, true)
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		resp, err := ex.GetOrderInfo(t.Context(), "12569099460", currency.NewBTCUSDT(), asset.Spot)
		require.NoError(t, err, "GetOrderInfo must query the order over the WebSocket API")
		if mockTests {
			exp := &order.Detail{
				TimeInForce:          order.GoodTillCancel,
				Amount:               0.001,
				AverageExecutedPrice: 23416.1,
				ExecutedAmount:       0.001,
				Cost:                 23.4161,
				CostAsset:            currency.USDT,
				Exchange:             ex.Name,
				OrderID:              "12569099460",
				ClientOrderID:        "gct-market-order",
				Type:                 order.Market,
				Side:                 order.Buy,
				Status:               order.Filled,
				AssetType:            asset.Spot,
				Date:                 time.UnixMilli(1660801715700),
				LastUpdated:          time.UnixMilli(1660801715700),
				Pair:                 currency.NewBTCUSDT(),
			}
			assert.Equal(t, exp, resp, "GetOrderInfo should return the order queried over the WebSocket API")
		}
	})

	// SubmitOrder identifies OCO orders by order list ID and USDⓈ-M and COIN-M conditional orders by algo order ID, while
	// an untagged USDⓈ-M or COIN-M ID may name an order or an algo order
	const (
		notFound      = `{"code":-2013,"msg":"Order does not exist."}`
		outsideWindow = `{"code":-1021,"msg":"Timestamp for this request is outside of the recvWindow."}`
		usdtOrder     = `{"orderId":1930,"symbol":"BTCUSDT","status":"NEW","clientOrderId":"gct-order","price":"3000","avgPrice":"0","origQty":"0.01","executedQty":"0","cumQuote":"0","timeInForce":"GTC","type":"LIMIT","reduceOnly":false,"side":"BUY","positionSide":"BOTH","stopPrice":"0","origType":"LIMIT","time":1791284812300,"updateTime":1791284812400}`
		coinOrder     = `{"orderId":1930,"symbol":"BTCUSD_PERP","pair":"BTCUSD","status":"NEW","clientOrderId":"gct-order","price":"3000","avgPrice":"0","origQty":"1","executedQty":"0","cumBase":"0","timeInForce":"GTC","type":"LIMIT","reduceOnly":false,"side":"BUY","positionSide":"BOTH","stopPrice":"0","origType":"LIMIT","time":1791284812300,"updateTime":1791284812400}`
	)
	listReply := func(symbol string) string {
		return `{"orderListId":1930,"contingencyType":"OCO","listStatusType":"EXEC_STARTED","listOrderStatus":"EXECUTING","listClientOrderId":"gct-oco","transactionTime":1791284812300,"symbol":"` + symbol + `"}`
	}
	algoReply := func(symbol string) string {
		return `{"algoId":1930,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"` + symbol + `","side":"SELL","positionSide":"BOTH","timeInForce":"GTC","quantity":"1","algoStatus":"NEW","triggerPrice":"3000","icebergQuantity":"null","createTime":1791284812300,"updateTime":1791284812400}`
	}
	for _, tc := range []struct {
		name    string
		a       asset.Item
		orderID string
		replies map[string]string
		exp     *order.Detail
		err     error
		errMsg  string
	}{
		{
			name:    "spot order list",
			a:       asset.Spot,
			replies: map[string]string{"/api/v3/order": notFound, "/api/v3/orderList": listReply("BTCUSDT")},
			exp:     &order.Detail{Exchange: "TestGetOrderInfo/spot_order_list", OrderID: "1930", ClientOrderID: "gct-oco", Type: order.OCO, Status: order.Active, AssetType: asset.Spot, Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812300), Pair: currency.NewBTCUSDT()},
		},
		{
			name:    "spot order list of another symbol",
			a:       asset.Spot,
			replies: map[string]string{"/api/v3/order": notFound, "/api/v3/orderList": listReply("ETHUSDT")},
			err:     order.ErrOrderNotFound,
		},
		{
			name:    "spot order list not found",
			a:       asset.Spot,
			replies: map[string]string{"/api/v3/order": notFound, "/api/v3/orderList": `{"code":-2011,"msg":"Order list does not exist."}`},
			err:     order.ErrOrderNotFound,
		},
		{
			name:    "margin order list",
			a:       asset.Margin,
			replies: map[string]string{"/sapi/v1/margin/order": notFound, "/sapi/v1/margin/orderList": listReply("BTCUSDT")},
			exp:     &order.Detail{Exchange: "TestGetOrderInfo/margin_order_list", OrderID: "1930", ClientOrderID: "gct-oco", Type: order.OCO, Status: order.Active, AssetType: asset.Margin, Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812300), Pair: currency.NewBTCUSDT(), MarginType: margin.Multi},
		},
		{
			name:    "isolated margin order list",
			a:       asset.Margin,
			replies: map[string]string{"/sapi/v1/margin/order": notFound, "/sapi/v1/margin/orderList": `{"code":-2011,"msg":"Order list does not exist."}`, "/sapi/v1/margin/orderList isolated": strings.TrimSuffix(listReply("BTCUSDT"), "}") + `,"isIsolated":true}`},
			exp:     &order.Detail{Exchange: "TestGetOrderInfo/isolated_margin_order_list", OrderID: "1930", ClientOrderID: "gct-oco", Type: order.OCO, Status: order.Active, AssetType: asset.Margin, Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812300), Pair: currency.NewBTCUSDT(), MarginType: margin.Isolated},
		},
		{
			name:    "margin order list not found",
			a:       asset.Margin,
			replies: map[string]string{"/sapi/v1/margin/order": notFound, "/sapi/v1/margin/orderList": `{"code":-2011,"msg":"Order list does not exist."}`},
			err:     order.ErrOrderNotFound,
		},
		{
			// An OTO order list is not an OCO order, and its orders are found by their own order IDs
			name:    "spot OTO order list",
			a:       asset.Spot,
			replies: map[string]string{"/api/v3/order": notFound, "/api/v3/orderList": strings.Replace(listReply("BTCUSDT"), `"OCO"`, `"OTO"`, 1)},
			err:     order.ErrOrderNotFound,
		},
		{
			name:    "margin OTO order list",
			a:       asset.Margin,
			replies: map[string]string{"/sapi/v1/margin/order": notFound, "/sapi/v1/margin/orderList": strings.Replace(listReply("BTCUSDT"), `"OCO"`, `"OTO"`, 1)},
			err:     order.ErrOrderNotFound,
		},
		{
			name:    "USDⓈ-M algo order with an unknown status",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": notFound, "/fapi/v1/algoOrder": `{"algoId":1930,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSDT","side":"SELL","positionSide":"BOTH","quantity":"0.01","algoStatus":"WEIRD","triggerPrice":"3000","icebergQuantity":"null"}`},
			err:     errUnknownOrderStatus,
		},
		{
			name:    "USDⓈ-M algo order of another symbol",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": notFound, "/fapi/v1/algoOrder": `{"algoId":1930,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"ETHUSDT","side":"SELL","positionSide":"BOTH","quantity":"0.01","algoStatus":"NEW","triggerPrice":"3000","icebergQuantity":"null"}`},
			err:     order.ErrOrderNotFound,
		},
		{
			name:    "USDⓈ-M algo order part filled",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": notFound, "/fapi/v1/algoOrder": `{"algoId":1930,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSDT","side":"SELL","positionSide":"BOTH","timeInForce":"GTC","quantity":"1","algoStatus":"FINISHED","actualOrderId":"7","actualPrice":"2990","actualType":"MARKET","actualQty":"0.4","triggerPrice":"3000","icebergQuantity":"null","createTime":1791284812300,"updateTime":1791284812400}`},
			exp:     &order.Detail{TimeInForce: order.GoodTillCancel, Amount: 1, TriggerPrice: 3000, AverageExecutedPrice: 2990, ExecutedAmount: 0.4, RemainingAmount: 0.6, Exchange: "TestGetOrderInfo/USDⓈ-M_algo_order_part_filled", OrderID: "algo-1930", ClientOrderID: "gct-algo", Type: order.StopMarket, Side: order.Sell, Status: order.Closed, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812400), Pair: currency.NewBTCUSDT()},
		},
		{
			name:    "USDⓈ-M close position algo order filled",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": notFound, "/fapi/v1/algoOrder": `{"algoId":1930,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSDT","side":"SELL","positionSide":"BOTH","timeInForce":"GTC","quantity":"0","algoStatus":"FINISHED","actualOrderId":"8","actualPrice":"2990","actualType":"MARKET","actualQty":"0.5","triggerPrice":"3000","icebergQuantity":"null","closePosition":true,"createTime":1791284812300,"updateTime":1791284812400}`},
			exp:     &order.Detail{TimeInForce: order.GoodTillCancel, TriggerPrice: 3000, AverageExecutedPrice: 2990, ExecutedAmount: 0.5, Exchange: "TestGetOrderInfo/USDⓈ-M_close_position_algo_order_filled", OrderID: "algo-1930", ClientOrderID: "gct-algo", Type: order.StopMarket, Side: order.Sell, Status: order.Closed, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812400), Pair: currency.NewBTCUSDT()},
		},
		{
			name:    "USDⓈ-M algo order ID",
			a:       asset.USDTMarginedFutures,
			orderID: "algo-1930",
			replies: map[string]string{"/fapi/v1/algoOrder": algoReply("BTCUSDT")},
			exp:     &order.Detail{TimeInForce: order.GoodTillCancel, Amount: 1, TriggerPrice: 3000, RemainingAmount: 1, Exchange: "TestGetOrderInfo/USDⓈ-M_algo_order_ID", OrderID: "algo-1930", ClientOrderID: "gct-algo", Type: order.StopMarket, Side: order.Sell, Status: order.New, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812400), Pair: currency.NewBTCUSDT()},
		},
		{
			name:    "COIN-M algo order ID",
			a:       asset.CoinMarginedFutures,
			orderID: "algo-1930",
			replies: map[string]string{"/dapi/v1/algoOrder": algoReply("BTCUSD_PERP")},
			exp:     &order.Detail{TimeInForce: order.GoodTillCancel, Amount: 1, TriggerPrice: 3000, RemainingAmount: 1, Exchange: "TestGetOrderInfo/COIN-M_algo_order_ID", OrderID: "algo-1930", ClientOrderID: "gct-algo", Type: order.StopMarket, Side: order.Sell, Status: order.New, AssetType: asset.CoinMarginedFutures, Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812400), Pair: testCoinMarginedFuturesPair},
		},
		{
			name:    "USDⓈ-M algo order ID of another symbol",
			a:       asset.USDTMarginedFutures,
			orderID: "algo-1930",
			replies: map[string]string{"/fapi/v1/algoOrder": algoReply("ETHUSDT")},
			err:     order.ErrOrderNotFound,
		},
		{
			name:    "USDⓈ-M unknown algo order ID",
			a:       asset.USDTMarginedFutures,
			orderID: "algo-1930",
			replies: map[string]string{"/fapi/v1/algoOrder": notFound},
			err:     order.ErrOrderNotFound,
		},
		{
			name:    "USDⓈ-M algo order with an unknown status by algo order ID",
			a:       asset.USDTMarginedFutures,
			orderID: "algo-1930",
			replies: map[string]string{"/fapi/v1/algoOrder": strings.Replace(algoReply("BTCUSDT"), `"NEW"`, `"WEIRD"`, 1)},
			err:     errUnknownOrderStatus,
			errMsg:  "order algo-1930",
		},
		{
			name:    "USDⓈ-M order and algo order",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": usdtOrder, "/fapi/v1/algoOrder": algoReply("BTCUSDT")},
			err:     errAmbiguousOrderID,
			errMsg:  "look the algo order up as algo-1930",
		},
		{
			name:    "COIN-M order and algo order",
			a:       asset.CoinMarginedFutures,
			replies: map[string]string{"/dapi/v1/order": coinOrder, "/dapi/v1/algoOrder": algoReply("BTCUSD_PERP")},
			err:     errAmbiguousOrderID,
		},
		{
			name:    "USDⓈ-M order and another symbol's algo order",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": usdtOrder, "/fapi/v1/algoOrder": algoReply("ETHUSDT")},
			exp:     &order.Detail{TimeInForce: order.GoodTillCancel, Price: 3000, Amount: 0.01, RemainingAmount: 0.01, Exchange: "TestGetOrderInfo/USDⓈ-M_order_and_another_symbol's_algo_order", OrderID: "1930", ClientOrderID: "gct-order", Type: order.Limit, Side: order.Buy, Status: order.New, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812400), Pair: currency.NewBTCUSDT()},
		},
		{
			name:    "USDⓈ-M order and an algo order lookup rejected as unknown",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": usdtOrder, "/fapi/v1/algoOrder": `{"code":-2011,"msg":"Unknown order sent."}`},
			exp:     &order.Detail{TimeInForce: order.GoodTillCancel, Price: 3000, Amount: 0.01, RemainingAmount: 0.01, Exchange: "TestGetOrderInfo/USDⓈ-M_order_and_an_algo_order_lookup_rejected_as_unknown", OrderID: "1930", ClientOrderID: "gct-order", Type: order.Limit, Side: order.Buy, Status: order.New, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812400), Pair: currency.NewBTCUSDT()},
		},
		{
			name:    "COIN-M order without an algo order",
			a:       asset.CoinMarginedFutures,
			replies: map[string]string{"/dapi/v1/order": coinOrder, "/dapi/v1/algoOrder": notFound},
			exp:     &order.Detail{TimeInForce: order.GoodTillCancel, Price: 3000, Amount: 1, RemainingAmount: 1, Exchange: "TestGetOrderInfo/COIN-M_order_without_an_algo_order", OrderID: "1930", ClientOrderID: "gct-order", Type: order.Limit, Side: order.Buy, Status: order.New, AssetType: asset.CoinMarginedFutures, Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812400), Pair: testCoinMarginedFuturesPair},
		},
		{
			name:    "USDⓈ-M order with an unknown status",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": strings.Replace(usdtOrder, `"NEW"`, `"WEIRD"`, 1), "/fapi/v1/algoOrder": notFound},
			err:     errUnknownOrderStatus,
			errMsg:  "error converting usdtmarginedfutures order 1930",
		},
		{
			name:    "USDⓈ-M order and an algo order lookup failure",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": usdtOrder, "/fapi/v1/algoOrder": outsideWindow},
			errMsg:  "error fetching algo order",
		},
		{
			name:    "USDⓈ-M order lookup failure",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": outsideWindow},
			errMsg:  "outside of the recvWindow",
		},
		{
			name:    "USDⓈ-M unknown ID",
			a:       asset.USDTMarginedFutures,
			replies: map[string]string{"/fapi/v1/order": notFound, "/fapi/v1/algoOrder": notFound},
			err:     order.ErrOrderNotFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				reply, ok := tc.replies[r.URL.Path+" isolated"]
				if !ok || r.URL.Query().Get("isIsolated") != "TRUE" {
					reply, ok = tc.replies[r.URL.Path]
				}
				if !assert.Truef(t, ok, "GetOrderInfo should not request %s", r.URL.Path) {
					return
				}
				if strings.Contains(reply, `"code":-`) {
					w.WriteHeader(http.StatusBadRequest)
				}
				_, err := w.Write([]byte(reply))
				assert.NoError(t, err, "Write should not error")
			})
			url, err := ex.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
			require.NoError(t, err, "GetURL must not error")
			for _, u := range []exchange.URL{exchange.RestUSDTMargined, exchange.RestCoinMargined} {
				require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(u.String(), url), "SetRunningURL must not error for %s", u)
			}
			orderID, pair := cmp.Or(tc.orderID, "1930"), currency.NewBTCUSDT()
			if tc.a == asset.CoinMarginedFutures {
				pair = testCoinMarginedFuturesPair
			}
			got, err := ex.GetOrderInfo(t.Context(), orderID, pair, tc.a)
			if tc.errMsg != "" {
				require.ErrorContains(t, err, tc.errMsg, "GetOrderInfo must return the expected error")
			}
			if tc.err != nil || tc.errMsg == "" {
				require.ErrorIs(t, err, tc.err, "GetOrderInfo must return the expected error")
			} else {
				assert.NotErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should not report a failed lookup as no order having the ID")
			}
			assert.Equal(t, tc.exp, got, "GetOrderInfo should return the order")
		})
	}
}

func TestListedOrderListsAreOneCancelsTheOther(t *testing.T) { //nolint:tparallel // the cases share one exchange and its server in order
	t.Parallel()
	// The order list endpoints return every contingency type, while GCT tracks only OCO order lists as orders; the orders
	// of an OTO list are listed as orders
	const lists = `[{"orderListId":1,"contingencyType":"OCO","listStatusType":"EXEC_STARTED","listOrderStatus":"EXECUTING","listClientOrderId":"gct-oco","transactionTime":1791284812300,"symbol":"BTCUSDT"},` +
		`{"orderListId":2,"contingencyType":"OTO","listStatusType":"EXEC_STARTED","listOrderStatus":"EXECUTING","listClientOrderId":"gct-oto","transactionTime":1791284812300,"symbol":"BTCUSDT"}]`
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if !assert.Contains(t, []string{"/api/v3/openOrderList", "/api/v3/allOrderList", "/sapi/v1/margin/openOrderList", "/sapi/v1/margin/allOrderList"}, r.URL.Path, "only the order list endpoints should be requested") {
			return
		}
		_, err := w.Write([]byte(lists))
		assert.NoError(t, err, "Write should not error")
	})
	// The cases share ex and its server in order, so each also checks that a listing keeps nothing from the last
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		t.Run(a.String(), func(t *testing.T) {
			exp := order.FilteredOrders{{
				Exchange: ex.Name, OrderID: "1", ClientOrderID: "gct-oco", Type: order.OCO, Status: order.Active, AssetType: a,
				Date: time.UnixMilli(1791284812300), LastUpdated: time.UnixMilli(1791284812300), Pair: currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter),
			}}
			if a == asset.Margin {
				exp[0].MarginType = margin.Multi
			}
			active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Type: order.OCO, Side: order.AnySide})
			require.NoErrorf(t, err, "GetActiveOrders must not error for %s", a)
			assert.Equalf(t, exp, active, "GetActiveOrders should list only the %s OCO order lists", a)
			history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: a, Type: order.OCO, Side: order.AnySide, Pairs: currency.Pairs{currency.NewBTCUSDT()}})
			require.NoErrorf(t, err, "GetOrderHistory must not error for %s", a)
			assert.Equalf(t, exp, history, "GetOrderHistory should list only the %s OCO order lists", a)
		})
	}

	ws := newTestExchange(t)
	oneTriggersTheOtherStatus := &WebsocketListStatus{EventType: "listStatus", Symbol: "BTCUSDT", OrderListID: 2, ContingencyType: "OTO", ListOrderStatus: "EXECUTING"}
	require.NoError(t, ws.processListStatus(t.Context(), asset.Spot, []byte(`{"e":"listStatus","s":"BTCUSDT","g":2,"c":"OTO","L":"EXECUTING"}`)), "processListStatus must not error")
	assert.Equal(t, []any{oneTriggersTheOtherStatus}, relayedPayloads(ws), "processListStatus should relay an OTO list status without reporting the list as an order")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder must reject a nil modification")
	modify := &order.Modify{Pair: currency.NewBTCUSDT(), OrderID: "1234"}
	// The cases share modify and err with the checks after them, so they run in order
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			modify.AssetType = a
			_, err = e.ModifyOrder(t.Context(), modify)
			require.ErrorIsf(t, err, asset.ErrNotSupported, "ModifyOrder must reject %s orders", a)
		})
	}
	modify.AssetType = asset.USDTMarginedFutures
	modify.OrderID = "abc"
	_, err = e.ModifyOrder(t.Context(), modify)
	require.ErrorIs(t, err, errInvalidOrderID, "ModifyOrder must reject an invalid order ID")
	modify.OrderID = "20072994037"
	_, err = e.ModifyOrder(t.Context(), modify)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "ModifyOrder must reject a modification without a side")

	// Serial: it continues with modify, which the checks above set up
	t.Run("modifies orders", func(t *testing.T) {
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		modify.Exchange = e.Name
		modify.Side = order.Buy
		modify.Amount = 1
		modify.Price = 30005
		resp, err := e.ModifyOrder(t.Context(), modify)
		require.NoError(t, err, "ModifyOrder must not error")
		if mockTests {
			exp := &order.ModifyResponse{
				Exchange:        e.Name,
				OrderID:         "20072994037",
				ClientOrderID:   "LJ9R4QZDihCaS8UAOOLpgW",
				Pair:            currency.NewBTCUSDT(),
				Type:            order.Limit,
				Side:            order.Buy,
				Status:          order.PartiallyFilled,
				AssetType:       asset.USDTMarginedFutures,
				Price:           30005,
				Amount:          1,
				RemainingAmount: 0.6,
				LastUpdated:     time.UnixMilli(1629182711600),
			}
			assert.Equal(t, exp, resp, "ModifyOrder should return the modified order")
		}

		ensureTradablePairs(t)
		coinM := &order.Modify{Exchange: e.Name, AssetType: asset.CoinMarginedFutures, Pair: coinmTradablePair, OrderID: "20072994037", Side: order.Buy, Amount: 2, Price: 30005}
		resp, err = e.ModifyOrder(t.Context(), coinM)
		require.NoError(t, err, "ModifyOrder must not error for a COIN-M order")
		if mockTests {
			exp := &order.ModifyResponse{
				Exchange:        e.Name,
				OrderID:         "20072994037",
				ClientOrderID:   "LJ9R4QZDihCaS8UAOOLpgW",
				Pair:            coinmTradablePair,
				Type:            order.Limit,
				Side:            order.Buy,
				Status:          order.PartiallyFilled,
				AssetType:       asset.CoinMarginedFutures,
				Price:           30005,
				Amount:          2,
				RemainingAmount: 1,
				LastUpdated:     time.UnixMilli(1629182711600),
			}
			assert.Equal(t, exp, resp, "ModifyOrder should return the modified COIN-M order")
		}
	})

	// An algo order, which Binance cannot modify, is refused before any modification is sent, whether it is named by its
	// algo order ID, by a stop, take profit or trailing stop type, or by an untyped request's untagged ID that is the
	// pair's algo order's alone; an untagged ID of an order and an algo order is refused
	t.Run("algo orders", func(t *testing.T) {
		t.Parallel()
		const (
			notFound      = `{"code":-2013,"msg":"Order does not exist."}`
			outsideWindow = `{"code":-1021,"msg":"Timestamp for this request is outside of the recvWindow."}`
			algoOrder     = `{"algoId":20072994037,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSDT","side":"BUY","algoStatus":"NEW"}`
			usdtOrder     = `{"orderId":20072994037,"symbol":"BTCUSDT","status":"NEW","clientOrderId":"gct-order","price":"30000","origQty":"1","executedQty":"0","type":"LIMIT","side":"BUY","time":1629182711600,"updateTime":1629182711600}`
			modified      = `{"orderId":20072994037,"symbol":"BTCUSDT","status":"NEW","clientOrderId":"gct-order","price":"30005","origQty":"1","executedQty":"0","type":"LIMIT","side":"BUY","updateTime":1629182711600}`
		)
		byAlgoID, byID := map[string]string{"algoId": "20072994037"}, map[string]string{"orderId": "20072994037"}
		for _, tc := range []struct {
			name    string
			modify  order.Modify
			steps   []cancelStep
			err     error
			errMsg  string
			success bool
		}{
			{name: "algo order ID", modify: order.Modify{OrderID: "algo-20072994037"}, err: order.ErrUnsupportedOrderType},
			{name: "algo order ID of a limit type", modify: order.Modify{Type: order.Limit, OrderID: "algo-20072994037"}, err: order.ErrUnsupportedOrderType},
			{name: "stop type", modify: order.Modify{Type: order.StopMarket, OrderID: "20072994037"}, err: order.ErrUnsupportedOrderType},
			{name: "invalid algo order ID", modify: order.Modify{OrderID: "algo-x"}, err: errInvalidOrderID},
			{
				name:   "untyped algo order",
				modify: order.Modify{OrderID: "20072994037"},
				steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoOrder}, {http.MethodGet, "/fapi/v1/order", byID, notFound}},
				err:    order.ErrUnsupportedOrderType,
				errMsg: "the ID names algo order algo-20072994037",
			},
			{
				name:   "untyped order and algo order",
				modify: order.Modify{OrderID: "20072994037"},
				steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, algoOrder}, {http.MethodGet, "/fapi/v1/order", byID, usdtOrder}},
				err:    errAmbiguousOrderID,
			},
			{
				name:   "untyped lookup failure",
				modify: order.Modify{OrderID: "20072994037"},
				steps:  []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, outsideWindow}},
				errMsg: "error fetching algo order",
			},
			{
				name:    "untyped order",
				modify:  order.Modify{OrderID: "20072994037"},
				steps:   []cancelStep{{http.MethodGet, "/fapi/v1/algoOrder", byAlgoID, notFound}, {http.MethodPut, "/fapi/v1/order", byID, modified}},
				success: true,
			},
			{
				name:    "limit order",
				modify:  order.Modify{Type: order.Limit, OrderID: "20072994037"},
				steps:   []cancelStep{{http.MethodPut, "/fapi/v1/order", byID, modified}},
				success: true,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, sent := newCancelStepsExchange(t, tc.steps)
				tc.modify.AssetType, tc.modify.Pair, tc.modify.Side, tc.modify.Price, tc.modify.Amount = asset.USDTMarginedFutures, currency.NewBTCUSDT(), order.Buy, 30005, 1
				resp, err := ex.ModifyOrder(t.Context(), &tc.modify)
				if tc.errMsg != "" {
					assert.ErrorContains(t, err, tc.errMsg, "ModifyOrder should return the expected error")
				}
				switch {
				case tc.success:
					require.NoError(t, err, "ModifyOrder must modify the order")
					assert.Equal(t, "20072994037", resp.OrderID, "ModifyOrder should report the modified order")
				case tc.err != nil:
					assert.ErrorIs(t, err, tc.err, "ModifyOrder should return the expected error")
				}
				assert.Equal(t, len(tc.steps), sent(), "ModifyOrder should send every expected request")
			})
		}
	})
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must reject an empty batch")
	// The cases assign the test's err, as the checks around them do, so they run in order
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		t.Run(a.String(), func(t *testing.T) {
			_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: a, Pair: currency.NewBTCUSDT(), OrderID: "1"}})
			require.ErrorIsf(t, err, asset.ErrNotSupported, "CancelBatchOrders must reject %s orders", a)
		})
	}
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.USDTMarginedFutures, OrderID: "1"}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelBatchOrders must reject a batch without a pair")
	p := currency.NewBTCUSDT()
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{
		{AssetType: asset.USDTMarginedFutures, Pair: p, OrderID: "1"},
		{AssetType: asset.CoinMarginedFutures, Pair: p, OrderID: "2"},
	})
	require.ErrorIs(t, err, errBatchCancelRequiresSamePair, "CancelBatchOrders must reject a batch of several assets")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.USDTMarginedFutures, Pair: p}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrders must reject an order without an ID")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.USDTMarginedFutures, Pair: p, OrderID: "abc"}})
	require.ErrorIs(t, err, errInvalidOrderID, "CancelBatchOrders must reject an invalid order ID")

	t.Run("cancels batches", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		ensureTradablePairs(t)
		usdtm := make([]order.Cancel, 0, 13)
		for i := 1; i <= 11; i++ {
			usdtm = append(usdtm, order.Cancel{AssetType: asset.USDTMarginedFutures, OrderID: strconv.Itoa(i)})
		}
		usdtm = append(
			usdtm,
			order.Cancel{AssetType: asset.USDTMarginedFutures, ClientOrderID: "myOrder2"},
			order.Cancel{AssetType: asset.USDTMarginedFutures, Type: order.TakeProfit, OrderID: "2146760"},
		)
		for _, tc := range []struct {
			name   string
			orders []order.Cancel
			exp    *order.CancelBatchResponse
			err    error
		}{
			{
				name:   "USDⓈ-M",
				orders: usdtm,
				exp: &order.CancelBatchResponse{
					Status: map[string]string{
						"1":        "CANCELED",
						"10":       "Unknown order sent.",
						"11":       "CANCELED",
						"2":        "CANCELED",
						"2146760":  "CANCELED",
						"3":        "CANCELED",
						"4":        "CANCELED",
						"5":        "CANCELED",
						"6":        "CANCELED",
						"7":        "CANCELED",
						"8":        "CANCELED",
						"9":        "CANCELED",
						"myOrder2": "CANCELED",
					},
				},
			},
			{
				name:   "COIN-M",
				orders: []order.Cancel{{AssetType: asset.CoinMarginedFutures, OrderID: "283194212"}},
				exp: &order.CancelBatchResponse{
					Status: map[string]string{
						"283194212": "CANCELED",
					},
				},
			},
			{
				name:   "COIN-M stop placed as an order",
				orders: []order.Cancel{{AssetType: asset.CoinMarginedFutures, Type: order.StopMarket, OrderID: "283194212"}},
				exp: &order.CancelBatchResponse{
					Status: map[string]string{
						"283194212": "CANCELED",
					},
				},
			},
			{
				name:   "options",
				orders: []order.Cancel{{AssetType: asset.Options, ClientOrderID: "the-client-order-id-2"}},
				exp: &order.CancelBatchResponse{
					Status: map[string]string{
						"the-client-order-id-2": "CANCELLED",
					},
				},
			},
			{
				name:   "options rejected request",
				orders: []order.Cancel{{AssetType: asset.Options, OrderID: "1"}},
				err:    errAPIResponse,
				exp: &order.CancelBatchResponse{
					Status: map[string]string{
						"1": "API error response: pair not found: code -1121 msg Invalid symbol.",
					},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				for i := range tc.orders {
					tc.orders[i].Pair = assetToTradablePairMap[tc.orders[i].AssetType]
				}
				resp, err := e.CancelBatchOrders(t.Context(), tc.orders)
				require.ErrorIs(t, err, tc.err, "CancelBatchOrders must return the expected error")
				if mockTests {
					assert.Equal(t, tc.exp, resp, "CancelBatchOrders should return each order's outcome")
				}
			})
		}
	})

	t.Run("algo order fallback", func(t *testing.T) {
		t.Parallel()
		const (
			unknownOrder = `{"code":-2011,"msg":"Unknown order sent."}`
			notFound     = `{"code":-2013,"msg":"Order does not exist."}`
		)
		algo := func(id uint64, clientAlgoID string) string {
			return fmt.Sprintf(`{"algoId":%d,"clientAlgoId":%q,"algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSDT","side":"SELL","positionSide":"BOTH","quantity":"0.01","algoStatus":"NEW","triggerPrice":"3000","icebergQuantity":"null"}`, id, clientAlgoID)
		}
		algoCancelled := func(id uint64, clientAlgoID string) string {
			return fmt.Sprintf(`{"algoId":%d,"clientAlgoId":%q,"code":"200","msg":"success"}`, id, clientAlgoID)
		}
		lookup := func(path string, params map[string]string, reply string) cancelStep {
			return cancelStep{http.MethodGet, path, params, reply}
		}

		// The RPC server reports every entry cancelled unless the call errors, so a failed lookup or algo order cancellation
		// is returned
		for _, tc := range []struct {
			operation, errMsg string
		}{
			{"query", "error cancelling order 2: error fetching algo order"},
			{"cancel", "error cancelling orders [2]"},
		} {
			t.Run(tc.operation+" failure", func(t *testing.T) {
				t.Parallel()
				const failure = `{"code":-1002,"msg":"Not authorised."}`
				steps := []cancelStep{
					lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "1"}, notFound),
					lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "2"}, failure),
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[1]"}, `[{"orderId":1,"symbol":"BTCUSDT","status":"CANCELED"}]`},
				}
				if tc.operation == "cancel" {
					steps[1].reply = algo(2, "gct-algo")
					steps = slices.Insert(steps, 2, lookup("/fapi/v1/order", map[string]string{"orderId": "2"}, notFound))
					steps = append(steps, cancelStep{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "2"}, failure})
				}
				ex, sent := newCancelStepsExchange(t, steps)
				resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
					{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "1"},
					{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "2"},
				})
				require.ErrorIs(t, err, errAPIResponse, "CancelBatchOrders must return the failure")
				assert.ErrorContains(t, err, tc.errMsg, "CancelBatchOrders should identify the failed entry")
				require.NotNil(t, resp, "CancelBatchOrders must retain the partial response")
				assert.Equal(t, "CANCELED", resp.Status["1"], "CancelBatchOrders should retain the successful cancellation")
				assert.Contains(t, resp.Status["2"], "Not authorised.", "CancelBatchOrders should retain the failure")
				assert.Equal(t, len(steps), sent(), "CancelBatchOrders should send the expected requests")
			})
		}

		t.Run("lookup failure beside a cancellation of the same text", func(t *testing.T) {
			t.Parallel()
			// The other entry's cancellation, keyed by the same text, must not hide the failed lookup
			ex, sent := newCancelStepsExchange(t, []cancelStep{
				lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, `{"code":-1002,"msg":"Not authorised."}`),
				{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[77]"}, `[{"orderId":77,"symbol":"BTCUSDT","status":"CANCELED"}]`},
			})
			resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
				{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "77"},
				{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Limit, OrderID: "77"},
			})
			require.ErrorIs(t, err, errAPIResponse, "CancelBatchOrders must return the failed lookup")
			require.NotNil(t, resp, "CancelBatchOrders must return each entry's outcome")
			assert.Contains(t, resp.Status["77"], "Not authorised.", "CancelBatchOrders should report the failed lookup")
			assert.Equal(t, 2, sent(), "CancelBatchOrders should send the expected requests")
		})

		// An algo order one entry cancelled is not cancelled again for another entry naming it, by its algo ID or by an ID
		// Binance reported for it, while an entry naming another algo order by a client algo ID that went unused is
		for _, tc := range []struct {
			name    string
			entries []order.Cancel
			steps   []cancelStep
			exp     map[string]string
			refused string // the entry refused for naming both an order and an algo order
		}{
			{
				name:    "algo ID named twice",
				entries: []order.Cancel{{OrderID: "77"}, {OrderID: "77"}},
				steps: []cancelStep{
					lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algo(77, "gct-algo")),
					lookup("/fapi/v1/order", map[string]string{"orderId": "77"}, notFound),
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algoCancelled(77, "gct-algo")},
				},
				exp: map[string]string{"77": "CANCELED"},
			},
			{
				name:    "algo order named by its algo ID and its client algo ID",
				entries: []order.Cancel{{OrderID: "77"}, {ClientOrderID: "gct-algo"}},
				steps: []cancelStep{
					lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algo(77, "gct-algo")),
					lookup("/fapi/v1/order", map[string]string{"orderId": "77"}, notFound),
					lookup("/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-algo"}, algo(77, "gct-algo")),
					lookup("/fapi/v1/order", map[string]string{"origClientOrderId": "gct-algo"}, notFound),
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algoCancelled(77, "gct-algo")},
				},
				exp: map[string]string{"77": "CANCELED", "gct-algo": "CANCELED"},
			},
			{
				name:    "client algo ID sent beside another algo order's algo ID",
				entries: []order.Cancel{{OrderID: "77", ClientOrderID: "other-algo"}, {ClientOrderID: "other-algo"}},
				steps: []cancelStep{
					lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "77", "clientAlgoId": ""}, algo(77, "gct-algo")),
					lookup("/fapi/v1/order", map[string]string{"orderId": "77", "origClientOrderId": ""}, notFound),
					lookup("/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "other-algo", "algoId": ""}, algo(88, "other-algo")),
					lookup("/fapi/v1/order", map[string]string{"origClientOrderId": "other-algo", "orderId": ""}, notFound),
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algoCancelled(77, "gct-algo")},
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "88"}, algoCancelled(88, "other-algo")},
				},
				exp: map[string]string{"77": "CANCELED", "other-algo": "CANCELED"},
			},
			{
				name:    "untyped repeat of a typed algo order",
				entries: []order.Cancel{{Type: order.Stop, OrderID: "77"}, {OrderID: "77"}},
				steps: []cancelStep{
					lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algo(77, "gct-algo")),
					lookup("/fapi/v1/order", map[string]string{"orderId": "77"}, notFound),
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algoCancelled(77, "gct-algo")},
				},
				exp: map[string]string{"77": "CANCELED"},
			},
			{
				name:    "untyped repeat by algo ID of a typed algo order cancelled by its client algo ID",
				entries: []order.Cancel{{Type: order.Stop, ClientOrderID: "gct-algo"}, {OrderID: "77"}},
				steps: []cancelStep{
					lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algo(77, "gct-algo")),
					lookup("/fapi/v1/order", map[string]string{"orderId": "77"}, notFound),
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-algo", "algoId": ""}, algoCancelled(77, "gct-algo")},
				},
				exp: map[string]string{"77": "CANCELED", "gct-algo": "CANCELED"},
			},
			{
				name:    "untyped repeat of a client algo ID a typed entry sent beside another algo order's algo ID",
				entries: []order.Cancel{{Type: order.Stop, OrderID: "77", ClientOrderID: "other-algo"}, {ClientOrderID: "other-algo"}},
				steps: []cancelStep{
					lookup("/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "other-algo", "algoId": ""}, algo(88, "other-algo")),
					lookup("/fapi/v1/order", map[string]string{"origClientOrderId": "other-algo", "orderId": ""}, notFound),
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77", "clientAlgoId": ""}, algoCancelled(77, "gct-algo")},
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "88"}, algoCancelled(88, "other-algo")},
				},
				exp: map[string]string{"77": "CANCELED", "other-algo": "CANCELED"},
			},
			{
				name:    "algo order ID",
				entries: []order.Cancel{{OrderID: "algo-77"}, {OrderID: "77"}},
				steps: []cancelStep{
					lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algo(77, "gct-algo")),
					lookup("/fapi/v1/order", map[string]string{"orderId": "77"}, notFound),
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algoCancelled(77, "gct-algo")},
				},
				exp: map[string]string{"algo-77": "CANCELED", "77": "CANCELED"},
			},
			{
				name:    "ID of an order and an algo order",
				entries: []order.Cancel{{OrderID: "77"}, {OrderID: "1"}},
				steps: []cancelStep{
					lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, algo(77, "gct-algo")),
					lookup("/fapi/v1/order", map[string]string{"orderId": "77"}, `{"orderId":77,"symbol":"BTCUSDT","status":"NEW","clientOrderId":"gct-order","price":"3000","origQty":"0.01","executedQty":"0","type":"LIMIT","side":"BUY","time":1791284980615,"updateTime":1791284980615}`),
					lookup("/fapi/v1/algoOrder", map[string]string{"algoId": "1"}, notFound),
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[1]"}, `[{"orderId":1,"symbol":"BTCUSDT","status":"CANCELED"}]`},
				},
				exp:     map[string]string{"1": "CANCELED"},
				refused: "77",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, sent := newCancelStepsExchange(t, tc.steps)
				for i := range tc.entries {
					tc.entries[i].AssetType, tc.entries[i].Pair = asset.USDTMarginedFutures, currency.NewBTCUSDT()
				}
				resp, err := ex.CancelBatchOrders(t.Context(), tc.entries)
				if tc.refused != "" {
					require.ErrorIs(t, err, errAmbiguousOrderID, "CancelBatchOrders must refuse the ID")
					require.NotNil(t, resp, "CancelBatchOrders must return each entry's outcome")
					assert.Contains(t, resp.Status[tc.refused], "the algo order as algo-77", "CancelBatchOrders should report the refusal")
					delete(resp.Status, tc.refused)
				} else {
					require.NoError(t, err, "CancelBatchOrders must not error")
				}
				assert.Equal(t, tc.exp, resp.Status, "CancelBatchOrders should report each entry's outcome")
				assert.Equal(t, len(tc.steps), sent(), "CancelBatchOrders should send every expected request")
			})
		}

		// USDⓈ-M takes stop, take profit and trailing stop orders as algo orders alone, so those entries go straight to Cancel
		// Algo Order, while COIN-M ones are looked up as untyped entries are; limit and market entries name orders, and an
		// algo order ID names an algo order whatever its type
		entryTypes := append([]order.Type{order.Limit, order.Market, order.AnyType}, conditionalOrderTypes...)
		lookups := func(path string, ids ...uint64) []cancelStep {
			steps := make([]cancelStep, len(ids))
			for i, id := range ids {
				steps[i] = lookup(path, map[string]string{"algoId": strconv.FormatUint(id, 10)}, notFound)
			}
			return steps
		}
		algoCancels := func(path string, ids ...uint64) []cancelStep {
			steps := make([]cancelStep, len(ids))
			for i, id := range ids {
				steps[i] = cancelStep{http.MethodDelete, path, map[string]string{"algoId": strconv.FormatUint(id, 10)}, algoCancelled(id, "")}
			}
			return steps
		}
		batchCancel := func(path, symbol string, ids ...uint64) cancelStep {
			list, replies := make([]string, len(ids)), make([]string, len(ids))
			for i, id := range ids {
				list[i], replies[i] = strconv.FormatUint(id, 10), fmt.Sprintf(`{"orderId":%d,"symbol":%q,"status":"CANCELED"}`, id, symbol)
			}
			return cancelStep{http.MethodDelete, path, map[string]string{"orderIdList": "[" + strings.Join(list, ",") + "]"}, "[" + strings.Join(replies, ",") + "]"}
		}
		// Entries 1 to 10 have entryTypes' types and entry 11 is an algo order ID with a limit type; each row lists the
		// entries looked up, those cancelled ten at a time as orders and those cancelled as algo orders
		for _, tc := range []struct {
			name, api, symbol            string
			a                            asset.Item
			pair                         currency.Pair
			lookedUp, orders, algoOrders []uint64
		}{
			{
				name: "USDⓈ-M entry types", api: "/fapi/v1/", symbol: "BTCUSDT", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				lookedUp: []uint64{3}, orders: []uint64{1, 2, 3}, algoOrders: []uint64{4, 5, 6, 7, 8, 9, 10, 11},
			},
			{
				name: "COIN-M entry types", api: "/dapi/v1/", symbol: "BTCUSD_PERP", a: asset.CoinMarginedFutures, pair: testCoinMarginedFuturesPair,
				lookedUp: []uint64{3, 4, 5, 6, 7, 8, 9, 10}, orders: []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, algoOrders: []uint64{11},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				steps := slices.Concat(lookups(tc.api+"algoOrder", tc.lookedUp...), []cancelStep{batchCancel(tc.api+"batchOrders", tc.symbol, tc.orders...)}, algoCancels(tc.api+"algoOrder", tc.algoOrders...))
				ex, sent := newCancelStepsExchange(t, steps)
				entries := make([]order.Cancel, 0, len(entryTypes)+1)
				exp := make(map[string]string, len(entryTypes)+1)
				for i, orderType := range entryTypes {
					entries = append(entries, order.Cancel{AssetType: tc.a, Pair: tc.pair, Type: orderType, OrderID: strconv.Itoa(i + 1)})
					exp[strconv.Itoa(i+1)] = "CANCELED"
				}
				entries = append(entries, order.Cancel{AssetType: tc.a, Pair: tc.pair, Type: order.Limit, OrderID: "algo-11"})
				exp["algo-11"] = "CANCELED"
				resp, err := ex.CancelBatchOrders(t.Context(), entries)
				require.NoError(t, err, "CancelBatchOrders must cancel every entry")
				assert.Equal(t, exp, resp.Status, "CancelBatchOrders should report every entry cancelled")
				assert.Equal(t, len(steps), sent(), "CancelBatchOrders should send every expected request")
			})
			t.Run(tc.name+" as algo order IDs", func(t *testing.T) {
				t.Parallel()
				algoTypes := append([]order.Type{order.UnknownType, order.AnyType, order.Limit, order.Market}, conditionalOrderTypes...)
				entries := make([]order.Cancel, len(algoTypes))
				exp := make(map[string]string, len(algoTypes))
				ids := make([]uint64, len(algoTypes))
				var id uint64
				for i, orderType := range algoTypes {
					id++
					ids[i] = id
					entries[i] = order.Cancel{AssetType: tc.a, Pair: tc.pair, Type: orderType, OrderID: "algo-" + strconv.FormatUint(id, 10)}
					// A client order ID beside the algo order ID changes nothing, as the algo ID goes first
					if id%2 == 0 {
						entries[i].ClientOrderID = "gct-algo-" + strconv.FormatUint(id, 10)
					}
					exp[entries[i].OrderID] = "CANCELED"
				}
				steps := algoCancels(tc.api+"algoOrder", ids...)
				for i := range steps {
					steps[i].params["clientAlgoId"] = ""
				}
				ex, sent := newCancelStepsExchange(t, steps)
				resp, err := ex.CancelBatchOrders(t.Context(), entries)
				require.NoError(t, err, "CancelBatchOrders must cancel every algo order")
				assert.Equal(t, exp, resp.Status, "CancelBatchOrders should report every algo order cancelled")
				assert.Equal(t, len(steps), sent(), "CancelBatchOrders should cancel each algo order by its algo ID alone")
			})
			t.Run(tc.name+" with IDs that cannot be parsed", func(t *testing.T) {
				t.Parallel()
				steps := []cancelStep{batchCancel(tc.api+"batchOrders", tc.symbol, 1)}
				ex, sent := newCancelStepsExchange(t, steps)
				entries := make([]order.Cancel, 0, len(conditionalOrderTypes)+1)
				entries = append(entries, order.Cancel{AssetType: tc.a, Pair: tc.pair, Type: order.Limit, OrderID: "1"})
				exp := map[string]string{"1": "CANCELED"}
				for i, orderType := range conditionalOrderTypes {
					id := "x" + strconv.Itoa(i+1)
					entries = append(entries, order.Cancel{AssetType: tc.a, Pair: tc.pair, Type: orderType, OrderID: id})
					exp[id] = fmt.Sprintf(`invalid order ID %q: strconv.ParseUint: parsing %q: invalid syntax`, id, id)
				}
				resp, err := ex.CancelBatchOrders(t.Context(), entries)
				assert.ErrorIs(t, err, errInvalidOrderID, "CancelBatchOrders should return each unparsed entry's error")
				require.NotNil(t, resp, "CancelBatchOrders must return each entry's outcome")
				assert.Equal(t, exp, resp.Status, "CancelBatchOrders should fail each unparsed entry alone")
				assert.Equal(t, len(steps), sent(), "CancelBatchOrders should cancel the parsed entry alone")
			})
		}
	})

	// An order named twice keeps the status of its cancellation, which Binance follows by rejecting the repeat as unknown,
	// and a rejected options entry reports its rejection
	t.Run("entry outcomes", func(t *testing.T) {
		t.Parallel()
		const unknownOrder = `{"code":-2011,"msg":"Unknown order sent."}`
		optionsPair := currency.Pair{Base: currency.BTC, Quote: currency.NewCode("260410-72000-P"), Delimiter: currency.DashDelimiter}
		tenCancelled := make([]string, 10)
		tenOrders := make([]order.Cancel, 0, 10)
		// An untyped entry's untagged ID is looked up once before the batch, and these name no algo order
		tenLookups := make([]cancelStep, 0, 10)
		for i := range tenCancelled {
			tenCancelled[i] = fmt.Sprintf(`{"orderId":%d,"symbol":"BTCUSDT","status":"CANCELED"}`, i+1)
			tenOrders = append(tenOrders, order.Cancel{OrderID: strconv.Itoa(i + 1)})
			tenLookups = append(tenLookups, cancelStep{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": strconv.Itoa(i + 1)}, `{"code":-2013,"msg":"Order does not exist."}`})
		}
		noAlgoOrder := func(path string, params map[string]string) cancelStep {
			return cancelStep{http.MethodGet, path, params, `{"code":-2013,"msg":"Order does not exist."}`}
		}
		for _, tc := range []struct {
			name    string
			a       asset.Item
			pair    currency.Pair
			entries []order.Cancel
			steps   []cancelStep
			exp     map[string]string
			err     error
		}{
			{
				name: "USDⓈ-M order ID named twice", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				entries: []order.Cancel{{OrderID: "9"}, {OrderID: "9"}},
				steps: []cancelStep{
					noAlgoOrder("/fapi/v1/algoOrder", map[string]string{"algoId": "9"}),
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[9,9]"}, `[{"orderId":9,"symbol":"BTCUSDT","status":"CANCELED"},` + unknownOrder + `]`},
				},
				exp: map[string]string{"9": "CANCELED"},
			},
			{
				name: "USDⓈ-M client order ID named twice", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				entries: []order.Cancel{{ClientOrderID: "gct-order"}, {ClientOrderID: "gct-order"}},
				steps: []cancelStep{
					noAlgoOrder("/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-order"}),
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["gct-order","gct-order"]`}, `[{"orderId":5,"clientOrderId":"gct-order","symbol":"BTCUSDT","status":"CANCELED"},` + unknownOrder + `]`},
				},
				exp: map[string]string{"gct-order": "CANCELED"},
			},
			{
				// The lookup goes by the order ID, so a second lookup that could fail and report the order otherwise is
				// not sent
				name: "COIN-M stop's order ID named twice with different client order IDs", a: asset.CoinMarginedFutures, pair: testCoinMarginedFuturesPair,
				entries: []order.Cancel{{Type: order.StopMarket, OrderID: "77", ClientOrderID: "gct-first"}, {Type: order.StopMarket, OrderID: "77", ClientOrderID: "gct-second"}},
				steps: []cancelStep{
					noAlgoOrder("/dapi/v1/algoOrder", map[string]string{"algoId": "77", "clientAlgoId": ""}),
					{http.MethodDelete, "/dapi/v1/batchOrders", map[string]string{"orderIdList": "[77,77]"}, `[{"orderId":77,"symbol":"BTCUSD_PERP","status":"CANCELED"},` + unknownOrder + `]`},
				},
				exp: map[string]string{"77": "CANCELED"},
			},
			{
				name: "USDⓈ-M client order IDs of an algo order and an order", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				entries: []order.Cancel{{ClientOrderID: "gct-algo"}, {ClientOrderID: "gct-order"}},
				steps: []cancelStep{
					{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-algo", "algoId": ""}, `{"algoId":77,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSDT","side":"SELL","positionSide":"BOTH","quantity":"0.01","algoStatus":"NEW","triggerPrice":"3000"}`},
					{http.MethodGet, "/fapi/v1/order", map[string]string{"origClientOrderId": "gct-algo", "orderId": ""}, `{"code":-2013,"msg":"Order does not exist."}`},
					noAlgoOrder("/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-order", "algoId": ""}),
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["gct-order"]`}, `[{"orderId":5,"clientOrderId":"gct-order","symbol":"BTCUSDT","status":"CANCELED"}]`},
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, `{"algoId":77,"clientAlgoId":"gct-algo","code":"200","msg":"success"}`},
				},
				exp: map[string]string{"gct-algo": "CANCELED", "gct-order": "CANCELED"},
			},
			{
				name: "USDⓈ-M client order IDs differing only in case", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				entries: []order.Cancel{{ClientOrderID: "gct-case"}, {ClientOrderID: "GCT-CASE"}},
				steps: []cancelStep{
					noAlgoOrder("/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-case", "algoId": ""}),
					noAlgoOrder("/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "GCT-CASE", "algoId": ""}),
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["gct-case","GCT-CASE"]`}, `[{"orderId":6,"clientOrderId":"gct-case","symbol":"BTCUSDT","status":"CANCELED"},{"orderId":7,"clientOrderId":"GCT-CASE","symbol":"BTCUSDT","status":"CANCELED"}]`},
				},
				exp: map[string]string{"gct-case": "CANCELED", "GCT-CASE": "CANCELED"},
			},
			{
				// The algo order ID prefix marks order IDs alone, so a client order ID starting with it is another client order ID
				name: "USDⓈ-M client order IDs differing by the algo order ID prefix", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				entries: []order.Cancel{{ClientOrderID: "gct-stop"}, {ClientOrderID: "algo-gct-stop"}},
				steps: []cancelStep{
					noAlgoOrder("/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-stop", "algoId": ""}),
					noAlgoOrder("/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "algo-gct-stop", "algoId": ""}),
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["gct-stop","algo-gct-stop"]`}, `[{"orderId":8,"clientOrderId":"gct-stop","symbol":"BTCUSDT","status":"CANCELED"},{"orderId":9,"clientOrderId":"algo-gct-stop","symbol":"BTCUSDT","status":"CANCELED"}]`},
				},
				exp: map[string]string{"gct-stop": "CANCELED", "algo-gct-stop": "CANCELED"},
			},
			{
				// An order ID of 0 names no order, so these are looked up by their client order IDs, each its own algo order
				name: "COIN-M stops with a zero order ID and different client order IDs", a: asset.CoinMarginedFutures, pair: testCoinMarginedFuturesPair,
				entries: []order.Cancel{{Type: order.StopMarket, OrderID: "0", ClientOrderID: "gct-first"}, {Type: order.StopMarket, OrderID: "0", ClientOrderID: "gct-second"}},
				steps: []cancelStep{
					{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-first", "algoId": ""}, `{"algoId":31,"clientAlgoId":"gct-first","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSD_PERP","side":"SELL","positionSide":"BOTH","quantity":"1","algoStatus":"NEW","triggerPrice":"70000"}`},
					noAlgoOrder("/dapi/v1/order", map[string]string{"origClientOrderId": "gct-first", "orderId": ""}),
					{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-second", "algoId": ""}, `{"algoId":32,"clientAlgoId":"gct-second","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSD_PERP","side":"SELL","positionSide":"BOTH","quantity":"1","algoStatus":"NEW","triggerPrice":"70000"}`},
					noAlgoOrder("/dapi/v1/order", map[string]string{"origClientOrderId": "gct-second", "orderId": ""}),
					{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "31"}, `{"algoId":31,"clientAlgoId":"gct-first","code":"200","msg":"success"}`},
					{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"algoId": "32"}, `{"algoId":32,"clientAlgoId":"gct-second","code":"200","msg":"success"}`},
				},
				exp: map[string]string{"0": "CANCELED"},
			},
			{
				name: "USDⓈ-M order ID named again in a request that fails", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				entries: append(slices.Clone(tenOrders), order.Cancel{OrderID: "1"}),
				steps: append(
					slices.Clone(tenLookups),
					cancelStep{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[1,2,3,4,5,6,7,8,9,10]"}, `[` + strings.Join(tenCancelled, ",") + `]`},
					cancelStep{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[1]"}, `{"code":-1000,"msg":"An unknown error occurred while processing the request."}`},
				),
				exp: map[string]string{
					"1": "CANCELED", "2": "CANCELED", "3": "CANCELED", "4": "CANCELED", "5": "CANCELED", "6": "CANCELED", "7": "CANCELED", "8": "CANCELED",
					"9": "CANCELED", "10": "CANCELED",
				},
				err: errAPIResponse,
			},
			{
				name: "USDⓈ-M order ID cancelled when named again after a rejection", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				entries: append(slices.Clone(tenOrders), order.Cancel{OrderID: "1"}),
				steps: append(
					slices.Clone(tenLookups),
					cancelStep{
						http.MethodDelete, "/fapi/v1/batchOrders",
						map[string]string{"orderIdList": "[1,2,3,4,5,6,7,8,9,10]"},
						`[{"code":-1001,"msg":"Internal error; unable to process your request. Please try again."},` + strings.Join(tenCancelled[1:], ",") + `]`,
					},
					cancelStep{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[1]"}, `[` + tenCancelled[0] + `]`},
				),
				exp: map[string]string{
					"1": "CANCELED", "2": "CANCELED", "3": "CANCELED", "4": "CANCELED", "5": "CANCELED", "6": "CANCELED", "7": "CANCELED", "8": "CANCELED",
					"9": "CANCELED", "10": "CANCELED",
				},
			},
			{
				// Statuses are keyed by the ID's text, so this order's success must not hide another order's rejection
				name: "USDⓈ-M order ID and another order's client order ID sharing its text", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				entries: []order.Cancel{{Type: order.Limit, OrderID: "123"}, {Type: order.Limit, ClientOrderID: "123"}},
				steps: []cancelStep{
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[123]"}, `[{"orderId":123,"clientOrderId":"gct-order","symbol":"BTCUSDT","status":"CANCELED"}]`},
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["123"]`}, `[` + unknownOrder + `]`},
				},
				exp: map[string]string{"123": "Unknown order sent."},
			},
			{
				name: "USDⓈ-M algo order named twice", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				entries: []order.Cancel{{Type: order.Stop, OrderID: "77"}, {Type: order.Stop, OrderID: "77"}},
				steps: []cancelStep{
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, `{"algoId":77,"clientAlgoId":"gct-algo","code":"200","msg":"success"}`},
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, unknownOrder},
				},
				exp: map[string]string{"77": "CANCELED"},
				// Each algo order is cancelled by a request of its own, whose failure is returned
				err: errAPIResponse,
			},
			{
				// An algo order's entry is cancelled by a request of its own, so an ID that cannot be parsed fails that entry alone
				name: "USDⓈ-M algo order IDs that cannot be parsed beside an order", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				entries: []order.Cancel{{Type: order.Limit, OrderID: "1"}, {Type: order.Stop, OrderID: "broken-id", ClientOrderID: "gct-stop"}, {OrderID: "algo-x", ClientOrderID: "gct-algo"}},
				steps: []cancelStep{
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[1]"}, `[{"orderId":1,"symbol":"BTCUSDT","status":"CANCELED"}]`},
				},
				exp: map[string]string{
					"1":         "CANCELED",
					"broken-id": `invalid order ID "broken-id": strconv.ParseUint: parsing "broken-id": invalid syntax`,
					"algo-x":    `invalid order ID "algo-x": strconv.ParseUint: parsing "x": invalid syntax`,
				},
				err: errInvalidOrderID,
			},
			{
				name: "COIN-M order ID named twice", a: asset.CoinMarginedFutures, pair: testCoinMarginedFuturesPair,
				entries: []order.Cancel{{OrderID: "9"}, {OrderID: "9"}},
				steps: []cancelStep{
					noAlgoOrder("/dapi/v1/algoOrder", map[string]string{"algoId": "9"}),
					{http.MethodDelete, "/dapi/v1/batchOrders", map[string]string{"orderIdList": "[9,9]"}, `[{"orderId":9,"symbol":"BTCUSD_PERP","status":"CANCELED"},` + unknownOrder + `]`},
				},
				exp: map[string]string{"9": "CANCELED"},
			},
			{
				name: "options entry rejected", a: asset.Options, pair: optionsPair,
				entries: []order.Cancel{{OrderID: "4611875134427365377"}, {OrderID: "4611875134427365378"}},
				steps: []cancelStep{
					{http.MethodDelete, "/eapi/v1/batchOrders", map[string]string{"orderIds": "[4611875134427365377,4611875134427365378]"}, `[{"orderId":4611875134427365377,"symbol":"BTC-260410-72000-P","status":"CANCELLED"},` + unknownOrder + `]`},
				},
				exp: map[string]string{"4611875134427365377": "CANCELLED", "4611875134427365378": "Unknown order sent."},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, sent := newCancelStepsExchange(t, tc.steps)
				url, err := ex.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
				require.NoError(t, err, "GetURL must not error")
				for _, u := range []exchange.URL{exchange.RestCoinMargined, exchange.RestOptions} {
					require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(u.String(), url), "SetRunningURL must not error for %s", u)
				}
				entries := slices.Clone(tc.entries)
				for i := range entries {
					entries[i].AssetType, entries[i].Pair = tc.a, tc.pair
				}
				resp, err := ex.CancelBatchOrders(t.Context(), entries)
				require.ErrorIs(t, err, tc.err, "CancelBatchOrders must return the expected error")
				require.NotNil(t, resp, "CancelBatchOrders must return each entry's outcome")
				assert.Equal(t, tc.exp, resp.Status, "CancelBatchOrders should report each order's outcome")
				assert.Equal(t, len(tc.steps), sent(), "CancelBatchOrders should send the expected requests")
			})
		}

		t.Run("USDⓈ-M order ID and another algo order's algo ID sharing its text", func(t *testing.T) {
			t.Parallel()
			ex, sent := newCancelStepsExchange(t, []cancelStep{
				{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[77]"}, `[{"orderId":77,"symbol":"BTCUSDT","status":"CANCELED"}]`},
				{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, unknownOrder},
			})
			resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
				{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Limit, OrderID: "77"},
				{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Stop, OrderID: "77"},
			})
			require.ErrorIs(t, err, errAPIResponse, "CancelBatchOrders must return the algo order's cancellation error")
			require.NotNil(t, resp, "CancelBatchOrders must return each entry's outcome")
			// Statuses are keyed by the ID's text, so the other order's success must not hide the algo order's rejection
			assert.Len(t, resp.Status, 1, "CancelBatchOrders should report one status for the shared ID text")
			assert.Contains(t, resp.Status["77"], "Unknown order sent.", "CancelBatchOrders should report the algo order's rejection")
			assert.Equal(t, 2, sent(), "CancelBatchOrders should send the expected requests")
		})

		t.Run("options entry of a conditional type with an ID that cannot be parsed", func(t *testing.T) {
			t.Parallel()
			// Options orders are cancelled ten at a time whatever their type, so such an ID refuses the whole batch
			ex, sent := newCancelStepsExchange(t, nil)
			url, err := ex.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
			require.NoError(t, err, "GetURL must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestOptions.String(), url), "SetRunningURL must not error")
			resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
				{AssetType: asset.Options, Pair: optionsPair, OrderID: "4611875134427365377"},
				{AssetType: asset.Options, Pair: optionsPair, Type: order.Stop, OrderID: "broken-id"},
			})
			require.ErrorIs(t, err, errInvalidOrderID, "CancelBatchOrders must refuse the batch")
			assert.Nil(t, resp, "CancelBatchOrders should return no outcomes")
			assert.Zero(t, sent(), "CancelBatchOrders should send no request")
		})

		// Statuses are keyed by the ID's text, so one order's cancellation must not hide the failed cancellation of an algo
		// order named after it by that text as another kind of ID, whether that entry is typed or found by its lookup
		const outsideWindow = `{"code":-1021,"msg":"Timestamp for this request is outside of the recvWindow."}`
		algo77Lookup := []cancelStep{
			{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"algoId": "77", "clientAlgoId": ""}, `{"algoId":77,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSDT","side":"SELL","positionSide":"BOTH","quantity":"0.01","algoStatus":"NEW","triggerPrice":"3000","icebergQuantity":"null"}`},
			{http.MethodGet, "/fapi/v1/order", map[string]string{"orderId": "77", "origClientOrderId": ""}, `{"code":-2013,"msg":"Order does not exist."}`},
		}
		algo88Lookup := []cancelStep{
			{http.MethodGet, "/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "77", "algoId": ""}, `{"algoId":88,"clientAlgoId":"77","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSDT","side":"SELL","positionSide":"BOTH","quantity":"0.01","algoStatus":"NEW","triggerPrice":"3000","icebergQuantity":"null"}`},
			{http.MethodGet, "/fapi/v1/order", map[string]string{"origClientOrderId": "77", "orderId": ""}, `{"code":-2013,"msg":"Order does not exist."}`},
		}
		algo77Cancelled := cancelStep{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, `{"algoId":77,"clientAlgoId":"gct-algo","code":"200","msg":"success"}`}
		algo88Failed := cancelStep{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "88"}, outsideWindow}
		for _, tc := range []struct {
			name    string
			entries []order.Cancel
			steps   []cancelStep
			key     string
		}{
			{
				name:    "USDⓈ-M algo ID and another algo order's typed client algo ID sharing its text",
				entries: []order.Cancel{{Type: order.Stop, OrderID: "77"}, {Type: order.Stop, ClientOrderID: "77"}},
				steps:   []cancelStep{algo77Cancelled, {http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "77", "algoId": ""}, outsideWindow}},
				key:     "77",
			},
			{
				name:    "USDⓈ-M algo ID and another algo order's untyped client algo ID sharing its text",
				entries: []order.Cancel{{Type: order.Stop, OrderID: "77"}, {ClientOrderID: "77"}},
				steps:   slices.Concat(algo88Lookup, []cancelStep{algo77Cancelled, algo88Failed}),
				key:     "77",
			},
			{
				name:    "USDⓈ-M untyped algo ID and another algo order's untyped client algo ID sharing its text",
				entries: []order.Cancel{{OrderID: "77"}, {ClientOrderID: "77"}},
				steps:   slices.Concat(algo77Lookup, algo88Lookup, []cancelStep{algo77Cancelled, algo88Failed}),
				key:     "77",
			},
			{
				name:    "USDⓈ-M client order ID and an algo order's typed client algo ID sharing its text",
				entries: []order.Cancel{{Type: order.Limit, ClientOrderID: "shared"}, {Type: order.Stop, ClientOrderID: "shared"}},
				steps: []cancelStep{
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["shared"]`}, `[{"orderId":5,"clientOrderId":"shared","symbol":"BTCUSDT","status":"CANCELED"}]`},
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "shared", "algoId": ""}, outsideWindow},
				},
				key: "shared",
			},
			{
				name:    "USDⓈ-M client order ID and an algo order's typed algo ID sharing its text",
				entries: []order.Cancel{{Type: order.Limit, ClientOrderID: "77"}, {Type: order.Stop, OrderID: "77"}},
				steps: []cancelStep{
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["77"]`}, `[{"orderId":5,"clientOrderId":"77","symbol":"BTCUSDT","status":"CANCELED"}]`},
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, outsideWindow},
				},
				key: "77",
			},
			{
				name:    "USDⓈ-M order ID and an algo order's typed client algo ID sharing its text",
				entries: []order.Cancel{{Type: order.Limit, OrderID: "77"}, {Type: order.Stop, ClientOrderID: "77"}},
				steps: []cancelStep{
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[77]"}, `[{"orderId":77,"clientOrderId":"gct-order","symbol":"BTCUSDT","status":"CANCELED"}]`},
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"clientAlgoId": "77", "algoId": ""}, outsideWindow},
				},
				key: "77",
			},
			{
				name:    "USDⓈ-M client order ID and an algo order's untyped algo ID sharing its text",
				entries: []order.Cancel{{Type: order.Limit, ClientOrderID: "77"}, {OrderID: "77"}},
				steps: slices.Concat(algo77Lookup, []cancelStep{
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"origClientOrderIdList": `["77"]`}, `[{"orderId":5,"clientOrderId":"77","symbol":"BTCUSDT","status":"CANCELED"}]`},
					{http.MethodDelete, "/fapi/v1/algoOrder", map[string]string{"algoId": "77"}, outsideWindow},
				}),
				key: "77",
			},
			{
				name:    "USDⓈ-M order ID and an algo order's untyped client algo ID sharing its text",
				entries: []order.Cancel{{Type: order.Limit, OrderID: "77"}, {ClientOrderID: "77"}},
				steps: slices.Concat(algo88Lookup, []cancelStep{
					{http.MethodDelete, "/fapi/v1/batchOrders", map[string]string{"orderIdList": "[77]"}, `[{"orderId":77,"clientOrderId":"gct-order","symbol":"BTCUSDT","status":"CANCELED"}]`},
					algo88Failed,
				}),
				key: "77",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, sent := newCancelStepsExchange(t, tc.steps)
				entries := slices.Clone(tc.entries)
				for i := range entries {
					entries[i].AssetType, entries[i].Pair = asset.USDTMarginedFutures, currency.NewBTCUSDT()
				}
				resp, err := ex.CancelBatchOrders(t.Context(), entries)
				require.ErrorIs(t, err, errAPIResponse, "CancelBatchOrders must return the failed cancellation")
				require.NotNil(t, resp, "CancelBatchOrders must return each entry's outcome")
				assert.Len(t, resp.Status, 1, "CancelBatchOrders should report one status for the shared ID text")
				assert.Contains(t, resp.Status[tc.key], "outside of the recvWindow", "CancelBatchOrders should report the algo order's failed cancellation")
				assert.Equal(t, len(tc.steps), sent(), "CancelBatchOrders should send the expected requests")
			})
		}
	})
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	t.Run("pages", func(t *testing.T) {
		t.Parallel()
		var offsets []string
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			records := []string{}
			if r.URL.Path == "/sapi/v1/capital/deposit/hisrec" {
				offsets = append(offsets, r.URL.Query().Get("offset"))
				count := 1
				if r.URL.Query().Get("offset") == "" {
					count = fundingHistoryPageLimit
				}
				for i := range count {
					records = append(records, `{"id":"`+strconv.Itoa(i)+`","coin":"BTC","status":1}`)
				}
			}
			_, err := w.Write([]byte("[" + strings.Join(records, ",") + "]"))
			assert.NoError(t, err, "Write should not error")
		})
		result, err := ex.GetAccountFundingHistory(t.Context())
		require.NoError(t, err, "GetAccountFundingHistory must not error")
		assert.Len(t, result, fundingHistoryPageLimit+1, "GetAccountFundingHistory should collect every page")
		assert.Equal(t, []string{"", "1000"}, offsets, "GetAccountFundingHistory should request the next page from the records collected")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetAccountFundingHistory should return records")
		return
	}
	exp := []exchange.FundingHistory{
		{
			ExchangeName:      e.Name,
			Status:            "Success",
			TransferID:        "769800519366885376",
			Timestamp:         time.UnixMilli(1661493146000),
			Currency:          "BNB",
			Amount:            0.001,
			TransferType:      "deposit",
			CryptoToAddress:   "bnb136ns6lfw4zs5hg4n85vdthaad7hq5m4gtkgf23",
			CryptoFromAddress: "bnb1grpf0955h0ykzq3ar5nmum7y6gdfl6lxfn46h2",
			CryptoTxID:        "98A3EA560C6B3336D348B6C83F0F95ECE4F1F5919E94BD006E5BF3BF264FACFC",
			CryptoChain:       "BNB",
		},
		{
			ExchangeName:    e.Name,
			Status:          "Waiting user confirm",
			TransferID:      "769800519366885377",
			Timestamp:       time.UnixMilli(1661493200000),
			Currency:        "XRP",
			Amount:          25,
			TransferType:    "deposit",
			CryptoToAddress: "rEb8TK3gBgk5auZkwc6sHnwrGVJH8DuaLh",
			CryptoTxID:      "Off-chain transfer 9d3c1b7a5e2f4d6c",
			CryptoChain:     "XRP",
		},
		{
			ExchangeName:    e.Name,
			Status:          "Completed",
			TransferID:      "b6ae22b3aa844210a7041aee7589627c",
			Timestamp:       time.Date(2019, 10, 12, 11, 12, 2, 0, time.UTC),
			Currency:        "USDT",
			Amount:          8.91,
			Fee:             0.004,
			TransferType:    "withdrawal",
			CryptoToAddress: "0x94df8b352de7f46f64b01d3666bf6e936e44ce60",
			CryptoTxID:      "0xb5ef8c13b968a406cc62a93a8bd80f9e9a906ef1b3fcf20a2e48573c17659268",
			CryptoChain:     "ETH",
		},
		{
			ExchangeName:    e.Name,
			Status:          "Rejected",
			TransferID:      "c8f1a3b5d7e94b2c8a6f0e1d2c3b4a59",
			Description:     "The address is not valid. Please confirm with the recipient",
			Timestamp:       time.Date(2019, 10, 13, 9, 1, 0, 0, time.UTC),
			Currency:        "ETH",
			Amount:          0.5,
			Fee:             0.0005,
			TransferType:    "withdrawal",
			CryptoToAddress: "0x8d12a197cb00d4747a1fe03395095ce2a5cc6819",
			CryptoChain:     "ARBITRUM",
		},
	}
	assert.Equal(t, exp, result, "GetAccountFundingHistory should return every deposit and withdrawal")
}

func TestFundingStatusStrings(t *testing.T) {
	t.Parallel()
	for status, exp := range map[uint64]string{0: "Pending", 1: "Success", 2: "Rejected", 6: "Credited but cannot withdraw", 7: "Wrong deposit", 8: "Waiting user confirm", 9: "9"} {
		t.Run("deposit "+strconv.FormatUint(status, 10), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, depositStatusString(status), "depositStatusString should name status %d", status)
		})
	}
	for status, exp := range map[uint64]string{0: "Email sent", 2: "Awaiting approval", 3: "Rejected", 4: "Processing", 6: "Completed", 9: "9"} {
		t.Run("withdrawal "+strconv.FormatUint(status, 10), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, withdrawalStatusString(status), "withdrawalStatusString should name status %d", status)
		})
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "WithdrawCryptocurrencyFunds must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:      e.Name,
		Currency:      currency.USDT,
		Description:   "not sent, as it would save the address",
		Amount:        8.91,
		Type:          withdraw.Crypto,
		ClientOrderID: "gct-withdrawal-1",
		Crypto: withdraw.CryptoRequest{
			Address: "0x94df8b352de7f46f64b01d3666bf6e936e44ce60",
			Chain:   "ETH",
		},
	})
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	if mockTests {
		assert.Equal(t, &withdraw.ExchangeResponse{ID: "7213fea8e94b4a5593d507237e5a555b"}, result, "WithdrawCryptocurrencyFunds should return the withdrawal ID")
		return
	}
	assert.NotEmpty(t, result.ID, "WithdrawCryptocurrencyFunds should return the withdrawal ID")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.ETH, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetWithdrawalsHistory should return withdrawals")
		return
	}
	exp := []exchange.WithdrawalHistory{
		{
			Status:          "Rejected",
			TransferID:      "c8f1a3b5d7e94b2c8a6f0e1d2c3b4a59",
			Description:     "The address is not valid. Please confirm with the recipient",
			Timestamp:       time.Date(2019, 10, 13, 9, 1, 0, 0, time.UTC),
			Currency:        "ETH",
			Amount:          0.5,
			Fee:             0.0005,
			TransferType:    "withdrawal",
			CryptoToAddress: "0x8d12a197cb00d4747a1fe03395095ce2a5cc6819",
			CryptoChain:     "ARBITRUM",
		},
	}
	assert.Equal(t, exp, result, "GetWithdrawalsHistory should return every withdrawal of the currency")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should not be supported")
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should not be supported")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDepositAddress(t.Context(), currency.USDT, "", currency.BNB.String())
	require.NoError(t, err, "GetDepositAddress must not error")
	if mockTests {
		exp := &deposit.Address{
			Address: "bnb136ns6lfw4zs5hg4n85vdthaad7hq5m4gtkgf23",
			Tag:     "101764890",
			Chain:   "BNB",
		}
		assert.Equal(t, exp, result, "GetDepositAddress should return the address and tag")
		return
	}
	assert.NotEmpty(t, result.Address, "GetDepositAddress should return an address")
}

// klineTestWindow returns the window of the kline tests, four five-minute candles
func klineTestWindow() (start, end time.Time) {
	if mockTests {
		return time.UnixMilli(1791127800000).UTC(), time.UnixMilli(1791129000000).UTC()
	}
	end = time.Now().UTC().Truncate(5 * time.Minute).Add(-5 * time.Minute)
	return end.Add(-20 * time.Minute), end
}

// klineTestCandles returns the candles the fixtures record for an asset in the kline test window
func klineTestCandles(a asset.Item) []kline.Candle {
	at := func(minute int) time.Time { return time.Date(2026, 10, 4, 15, minute, 0, 0, time.UTC) }
	switch a {
	case asset.Spot, asset.Margin:
		return []kline.Candle{
			{Time: at(30), Open: 85320.01, High: 85320.01, Low: 85288.01, Close: 85299.39, Volume: 15.37019},
			{Time: at(35), Open: 85299.38, High: 85300, Low: 85284.98, Close: 85299.99, Volume: 9.8598},
			{Time: at(40), Open: 85299.99, High: 85349.19, Low: 85299.99, Close: 85342.78, Volume: 16.97926},
			{Time: at(45), Open: 85342.77, High: 85398.33, Low: 85342.61, Close: 85342.62, Volume: 26.23221},
		}
	case asset.USDTMarginedFutures:
		return []kline.Candle{
			{Time: at(30), Open: 85276.5, High: 85276.6, Low: 85241.6, Close: 85254.1, Volume: 149.465},
			{Time: at(35), Open: 85254.1, High: 85272.6, Low: 85239.4, Close: 85272.5, Volume: 95.438},
			{Time: at(40), Open: 85272.6, High: 85309.2, Low: 85272.5, Close: 85299.9, Volume: 101.978},
			{Time: at(45), Open: 85299.9, High: 85359.7, Low: 85299.9, Close: 85303.9, Volume: 117.292},
		}
	case asset.CoinMarginedFutures:
		// COIN-M volumes count contracts
		return []kline.Candle{
			{Time: at(30), Open: 85260.4, High: 85260.4, Low: 85231.9, Close: 85247.3, Volume: 5282},
			{Time: at(35), Open: 85247.3, High: 85251.3, Low: 85229.6, Close: 85251.2, Volume: 2981},
			{Time: at(40), Open: 85251.3, High: 85283, Low: 85251.2, Close: 85283, Volume: 1250},
			{Time: at(45), Open: 85283, High: 85339.2, Low: 85282.9, Close: 85289.4, Volume: 10801},
		}
	default:
		return []kline.Candle{
			{Time: at(30), Open: 6225, High: 6225, Low: 6225, Close: 6225},
			{Time: at(35), Open: 6225, High: 6225, Low: 6225, Close: 6225},
			{Time: at(40), Open: 6480, High: 6480, Low: 6480, Close: 6480, Volume: 0.02},
			{Time: at(45), Open: 6480, High: 6480, Low: 6480, Close: 6480},
		}
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	ex := newMarketTestExchange(t)
	pairs := marketTestPairs(t)
	start, end := klineTestWindow()
	_, err := ex.GetHistoricCandles(t.Context(), pairs[asset.Spot], asset.Spot, kline.FiveMin, start.Add(-1000*kline.FiveMin.Duration()), end)
	require.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits, "GetHistoricCandles must reject a window beyond one request")
	_, err = ex.GetHistoricCandles(t.Context(), pairs[asset.USDTMarginedFutures], asset.USDTMarginedFutures, kline.ThousandMilliseconds, end.Add(-time.Minute), end)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetHistoricCandles must reject one second futures candles")

	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := ex.GetHistoricCandles(t.Context(), pairs[a], a, kline.FiveMin, start, end)
			require.NoError(t, err, "GetHistoricCandles must not error")
			if !mockTests {
				assert.Len(t, result.Candles, 4, "GetHistoricCandles should return every candle of the window")
				return
			}
			// The candle at the end of the window, which the request also returns, is left out
			exp := &kline.Item{Exchange: ex.Name, Pair: pairs[a], Asset: a, Interval: kline.FiveMin, Candles: klineTestCandles(a)}
			assert.Equal(t, exp, result, "GetHistoricCandles should return every candle of the window")
		})
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	ex := newMarketTestExchange(t)
	// Two candles a request split the window into two requests
	ex.Features.Enabled.Kline.GlobalResultLimit = 2
	pairs := marketTestPairs(t)
	start, end := klineTestWindow()
	_, err := ex.GetHistoricCandlesExtended(t.Context(), pairs[asset.CoinMarginedFutures], asset.CoinMarginedFutures, kline.ThousandMilliseconds, end.Add(-2*time.Second), end)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetHistoricCandlesExtended must reject one second futures candles")

	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := ex.GetHistoricCandlesExtended(t.Context(), pairs[a], a, kline.FiveMin, start, end)
			require.NoError(t, err, "GetHistoricCandlesExtended must not error")
			if !mockTests {
				assert.Len(t, result.Candles, 4, "GetHistoricCandlesExtended should return every candle of the window")
				return
			}
			exp := &kline.Item{Exchange: ex.Name, Pair: pairs[a], Asset: a, Interval: kline.FiveMin, Candles: klineTestCandles(a)}
			assert.Equal(t, exp, result, "GetHistoricCandlesExtended should return the candles of every request")
		})
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a    asset.Item
		path string
	}{
		{asset.Spot, "/api/v3/account"},
		{asset.Margin, "/sapi/v1/margin/account"},
		{asset.USDTMarginedFutures, "/fapi/v3/balance"},
		{asset.CoinMarginedFutures, "/dapi/v1/account"},
		{asset.Options, "/eapi/v1/marginAccount"},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assert.Equal(t, tc.path, r.URL.Path, "credential validation should query the asset's own account endpoint")
				reply := balanceRefreshReplies[r.URL.Path]
				switch r.URL.Path {
				case "/api/v3/account":
					reply = `{"balances":[{"asset":"USDT","free":"100","locked":"0"}]}`
				case "/sapi/v1/margin/account":
					reply = `{"userAssets":[{"asset":"USDT","free":"100","locked":"0","borrowed":"0","interest":"0"}]}`
				}
				_, err := w.Write([]byte(reply))
				assert.NoError(t, err, "the account response should be written")
			})
			assert.NoError(t, ex.ValidateAPICredentials(t.Context(), tc.a), "valid account credentials should be accepted")
			assert.Equal(t, int32(1), requests.Load(), "credential validation should send one account request")
		})
	}
	t.Run("rejected credentials", func(t *testing.T) {
		t.Parallel()
		ex := newTradingTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write([]byte(`{"code":-2015,"msg":"Invalid API-key, IP, or permissions for action."}`))
			assert.NoError(t, err, "the rejection should be written")
		})
		assert.ErrorIs(t, ex.ValidateAPICredentials(t.Context(), asset.Spot), errAPIResponse, "credential validation should preserve the venue rejection")
	})
	t.Run("unsupported asset", func(t *testing.T) {
		t.Parallel()
		ex := newTradingTestExchange(t, func(_ http.ResponseWriter, _ *http.Request) {
			assert.Fail(t, "an unsupported asset should make no request")
		})
		assert.ErrorIs(t, ex.ValidateAPICredentials(t.Context(), asset.Empty), asset.ErrNotSupported, "credential validation should reject an unsupported asset")
	})
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		exp      string
	}{
		{kline.ThousandMilliseconds, "1s"},
		{kline.OneMin, "1m"},
		{kline.FifteenMin, "15m"},
		{kline.OneHour, "1h"},
		{kline.TwelveHour, "12h"},
		{kline.OneDay, "1d"},
		{kline.ThreeDay, "3d"},
		{kline.OneWeek, "1w"},
		{kline.OneMonth, "1M"},
	} {
		t.Run(tc.interval.Short(), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, e.FormatExchangeKlineInterval(tc.interval), "FormatExchangeKlineInterval should format %s", tc.interval)
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject an unsupported asset")

	pairs := marketTestPairs(t)
	spotTrades := func() []trade.Data {
		return []trade.Data{
			{TID: "6122214010", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), Side: order.Sell, Price: 71657.63, Amount: 0.00008, Timestamp: time.UnixMilli(1773844903807)},
			{TID: "6122214011", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), Side: order.Sell, Price: 71657.63, Amount: 0.00008, Timestamp: time.UnixMilli(1773844903807)},
			{TID: "6122214012", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), Side: order.Sell, Price: 71657.63, Amount: 0.00008, Timestamp: time.UnixMilli(1773844903807)},
			{TID: "6122214013", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), Side: order.Sell, Price: 71657.63, Amount: 0.00007, Timestamp: time.UnixMilli(1773844903807)},
			{TID: "6122214014", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), Side: order.Sell, Price: 71657.63, Amount: 0.00008, Timestamp: time.UnixMilli(1773844903807)},
		}
	}
	coinmPair := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	optionsPair := currency.NewPairWithDelimiter("BTC", "261225-85000-C", currency.DashDelimiter)
	for _, tc := range []struct {
		a   asset.Item
		exp []trade.Data
	}{
		{a: asset.Spot, exp: spotTrades()},
		{a: asset.Margin, exp: spotTrades()},
		{
			a: asset.USDTMarginedFutures,
			exp: []trade.Data{
				{TID: "8149534980", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), Side: order.Sell, Price: 86065.5, Amount: 0.001, Timestamp: time.UnixMilli(1791283323016)},
				{TID: "8149534982", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), Side: order.Buy, Price: 86065.6, Amount: 0.001, Timestamp: time.UnixMilli(1791283324072)},
			},
		},
		{
			// COIN-M quantities count contracts, so the amounts are the base asset quantities
			a: asset.CoinMarginedFutures,
			exp: []trade.Data{
				{TID: "1106358704", Exchange: e.Name, CurrencyPair: coinmPair, Side: order.Sell, Price: 71627.2, Amount: 5, Timestamp: time.UnixMilli(1773844786716)},
				{TID: "1106358705", Exchange: e.Name, CurrencyPair: coinmPair, Side: order.Sell, Price: 71627.2, Amount: 10, Timestamp: time.UnixMilli(1773844786716)},
				{TID: "1106358706", Exchange: e.Name, CurrencyPair: coinmPair, Side: order.Sell, Price: 71626.8, Amount: 1, Timestamp: time.UnixMilli(1773844786716)},
				{TID: "1106358707", Exchange: e.Name, CurrencyPair: coinmPair, Side: order.Sell, Price: 71624.5, Amount: 2, Timestamp: time.UnixMilli(1773844786738)},
				{TID: "1106358708", Exchange: e.Name, CurrencyPair: coinmPair, Side: order.Sell, Price: 71624, Amount: 1, Timestamp: time.UnixMilli(1773844786738)},
			},
		},
		{
			a: asset.Options,
			exp: []trade.Data{
				{TID: "119", Exchange: e.Name, CurrencyPair: optionsPair, Side: order.Sell, Price: 4180, Amount: 0.01, Timestamp: time.UnixMilli(1788312683522)},
				{TID: "120", Exchange: e.Name, CurrencyPair: optionsPair, Side: order.Buy, Price: 4370, Amount: 0.05, Timestamp: time.UnixMilli(1788327662839)},
				{TID: "218", Exchange: e.Name, CurrencyPair: optionsPair, Side: order.Sell, Price: 6965, Amount: 0.03, Timestamp: time.UnixMilli(1791296963967)},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRecentTrades(t.Context(), pairs[tc.a], tc.a)
			require.NoError(t, err, "GetRecentTrades must not error")
			if !mockTests {
				assert.NotNil(t, result, "GetRecentTrades should return trades")
				return
			}
			for i := range tc.exp {
				tc.exp[i].AssetType = tc.a
			}
			assert.Equal(t, tc.exp, result, "GetRecentTrades should return every trade")
		})
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.USDTMarginedFutures, time.Now().Add(-time.Minute), time.Now())
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricTrades must reject an unsupported asset")

	start, end := time.UnixMilli(1791284810000), time.UnixMilli(1791284811500)
	if !mockTests {
		end = time.Now().Truncate(time.Second)
		start = end.Add(-5 * time.Second)
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetHistoricTrades(mockContext(t), currency.NewBTCUSDT(), a, start, end)
			require.NoError(t, err, "GetHistoricTrades must not error")
			if !mockTests {
				for i := range result {
					assert.WithinRangef(t, result[i].Timestamp, start, end, "GetHistoricTrades should return trade %d from the window", i)
				}
				return
			}
			// The window ends before the last aggregate trade of the second page
			exp := []trade.Data{
				{TID: "4081863688", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: a, Side: order.Sell, Price: 86168.13, Amount: 0.06453, Timestamp: time.UnixMilli(1791284810512)},
				{TID: "4081863689", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: a, Side: order.Buy, Price: 86168.14, Amount: 0.0032, Timestamp: time.UnixMilli(1791284810933)},
				{TID: "4081863690", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: a, Side: order.Sell, Price: 86168.13, Amount: 0.00116, Timestamp: time.UnixMilli(1791284810941)},
				{TID: "4081863691", Exchange: e.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: a, Side: order.Sell, Price: 86168.13, Amount: 0.00167, Timestamp: time.UnixMilli(1791284811009)},
			}
			assert.Equal(t, exp, result, "GetHistoricTrades should return every aggregate trade of the window")
		})
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetAvailableTransferChains(t.Context(), currency.NewCode("GCTNOTLISTED"))
	require.ErrorIs(t, err, currency.ErrCurrencyNotFound, "GetAvailableTransferChains must reject a currency that is not listed")
	if mockTests {
		_, err = e.GetAvailableTransferChains(t.Context(), currency.EUR)
		require.ErrorIs(t, err, currency.ErrCurrencyNotFound, "GetAvailableTransferChains must reject a currency without networks")
	}
	result, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	if mockTests {
		assert.Equal(t, []string{"BTC", "BSC"}, result, "GetAvailableTransferChains should return every network of the coin")
		return
	}
	assert.NotEmpty(t, result, "GetAvailableTransferChains should return chains")
}

func TestFormatExchangeCurrency(t *testing.T) { //nolint:tparallel // the cases share e, so they run in order while the test runs in parallel
	t.Parallel()
	// The cases share e in order, so each also checks that a request keeps nothing from the last
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  string
	}{
		{asset.Spot, currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), "BTCUSDT"},
		{asset.Margin, currency.NewPairWithDelimiter("LTO", "USDT", currency.UnderscoreDelimiter), "LTOUSDT"},
		{asset.USDTMarginedFutures, currency.NewPairWithDelimiter("btc", "usdt", currency.DashDelimiter), "BTCUSDT"},
		{asset.USDTMarginedFutures, currency.NewPair(currency.NewCode("BTCUSDT"), currency.NewCode("261225")), "BTCUSDT_261225"},
		{asset.CoinMarginedFutures, currency.NewPair(currency.NewCode("BTCUSD"), currency.PERP), "BTCUSD_PERP"},
		{asset.CoinMarginedFutures, currency.NewPair(currency.NewCode("BTCUSD"), currency.NewCode("211231")), "BTCUSD_211231"},
		{asset.Options, currency.NewPair(currency.BTC, currency.NewCode("261225-85000-C")), "BTC-261225-85000-C"},
	} {
		t.Run(tc.a.String()+" "+tc.pair.String(), func(t *testing.T) {
			result, err := e.FormatExchangeCurrency(tc.pair, tc.a)
			require.NoErrorf(t, err, "FormatExchangeCurrency must not error for %s %s", tc.a, tc.pair)
			assert.Equalf(t, tc.exp, result.String(), "FormatExchangeCurrency should format %s %s", tc.a, tc.pair)
		})
	}
}

func TestFormatSymbol(t *testing.T) { //nolint:tparallel // the cases share e, so they run in order while the test runs in parallel
	t.Parallel()
	// The cases share e in order, so each also checks that a request keeps nothing from the last
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  string
	}{
		{asset.Spot, currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), "BTCUSDT"},
		{asset.Margin, currency.NewPairWithDelimiter("LTO", "USDT", currency.UnderscoreDelimiter), "LTOUSDT"},
		{asset.USDTMarginedFutures, currency.NewPairWithDelimiter("btc", "usdt", currency.DashDelimiter), "BTCUSDT"},
		{asset.USDTMarginedFutures, currency.NewBTCUSDT(), "BTCUSDT"},
		{asset.USDTMarginedFutures, currency.NewPair(currency.NewCode("BTCUSDT"), currency.NewCode("261225")), "BTCUSDT_261225"},
		{asset.CoinMarginedFutures, currency.NewPair(currency.NewCode("BTCUSD"), currency.PERP), "BTCUSD_PERP"},
		{asset.CoinMarginedFutures, currency.NewPair(currency.NewCode("BTCUSD"), currency.NewCode("211231")), "BTCUSD_211231"},
		{asset.Options, currency.NewPair(currency.BTC, currency.NewCode("261225-85000-C")), "BTC-261225-85000-C"},
	} {
		t.Run(tc.a.String()+" "+tc.pair.String(), func(t *testing.T) {
			result, err := e.FormatSymbol(tc.pair, tc.a)
			require.NoErrorf(t, err, "FormatSymbol must not error for %s %s", tc.a, tc.pair)
			assert.Equalf(t, tc.exp, result, "FormatSymbol should format %s %s", tc.a, tc.pair)
		})
	}
}

func TestFormatUSDTMarginedFuturesPair(t *testing.T) {
	t.Parallel()
	pairFormat := currency.PairFormat{Uppercase: true}
	resp := e.formatUSDTMarginedFuturesPair(currency.NewPair(currency.DOGE, currency.USDT), pairFormat)
	assert.Equal(t, "DOGEUSDT", resp.String(), "formatUSDTMarginedFuturesPair should not delimit a perpetual")

	resp = e.formatUSDTMarginedFuturesPair(currency.NewPair(currency.DOGE, currency.NewCode("1234567890")), pairFormat)
	assert.Equal(t, "DOGE_1234567890", resp.String(), "formatUSDTMarginedFuturesPair should delimit a delivery date")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	ex := newMarketTestExchange(t)
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Futures), asset.ErrNotSupported, "UpdateOrderExecutionLimits must reject an unsupported asset")

	pairs := marketTestPairs(t)
	spotLimits := limits.MinMaxLevel{
		MinPrice:                0.01,
		MaxPrice:                1000000,
		PriceStepIncrementSize:  0.01,
		MultiplierUp:            2,
		MultiplierDown:          0.5,
		AveragePriceMinutes:     5,
		MinimumBaseAmount:       0.00001,
		MaximumBaseAmount:       9000,
		AmountStepIncrementSize: 0.00001,
		MinNotional:             5,
		MaxIcebergParts:         100,
		MarketMaxQty:            188.9393325,
		MaxTotalOrders:          200,
		MaxAlgoOrders:           5,
	}
	for _, tc := range []struct {
		a   asset.Item
		exp limits.MinMaxLevel
	}{
		{a: asset.Spot, exp: spotLimits},
		{a: asset.Margin, exp: spotLimits},
		{
			a: asset.USDTMarginedFutures,
			exp: limits.MinMaxLevel{
				MinPrice:                556.8,
				MaxPrice:                4529764,
				PriceStepIncrementSize:  0.1,
				MultiplierUp:            1.05,
				MultiplierDown:          0.95,
				MultiplierDecimal:       4,
				MinimumBaseAmount:       0.001,
				MaximumBaseAmount:       1000,
				AmountStepIncrementSize: 0.001,
				MinNotional:             50,
				MarketMinQty:            0.001,
				MarketMaxQty:            120,
				MarketStepIncrementSize: 0.001,
				MaxTotalOrders:          200,
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: limits.MinMaxLevel{
				MinPrice:                1000,
				MaxPrice:                4520958,
				PriceStepIncrementSize:  0.1,
				MultiplierUp:            1.05,
				MultiplierDown:          0.95,
				MultiplierDecimal:       4,
				MinimumBaseAmount:       1,
				MaximumBaseAmount:       1000000,
				AmountStepIncrementSize: 1,
				MarketMinQty:            1,
				MarketMaxQty:            60000,
				MarketStepIncrementSize: 1,
				MaxTotalOrders:          200,
				MaxAlgoOrders:           20,
			},
		},
		{
			a: asset.Options,
			exp: limits.MinMaxLevel{
				MinPrice:                1375,
				MaxPrice:                12360,
				PriceStepIncrementSize:  5,
				MinimumBaseAmount:       0.01,
				MaximumBaseAmount:       200,
				AmountStepIncrementSize: 0.01,
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), tc.a), "UpdateOrderExecutionLimits must not error")
			result, err := ex.GetOrderExecutionLimits(tc.a, pairs[tc.a])
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			if !mockTests {
				assert.Positive(t, result.PriceStepIncrementSize, "UpdateOrderExecutionLimits should load the tick size")
				return
			}
			assert.WithinRange(t, result.UpdatedAt, started, time.Now(), "UpdateOrderExecutionLimits should stamp the limits when loading them")
			tc.exp.UpdatedAt = result.UpdatedAt
			tc.exp.Key = key.NewExchangeAssetPair(ex.Name, tc.a, pairs[tc.a])
			assert.Equal(t, tc.exp, result, "UpdateOrderExecutionLimits should load the pair's limits")
		})
	}
}

func TestFetchExchangeLimits(t *testing.T) {
	t.Parallel()
	_, err := e.FetchExchangeLimits(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchExchangeLimits must reject an unsupported asset")

	ethbtc := limits.MinMaxLevel{
		MinPrice:                0.00001,
		MaxPrice:                922327,
		PriceStepIncrementSize:  0.00001,
		MultiplierUp:            2,
		MultiplierDown:          0.5,
		AveragePriceMinutes:     5,
		MinimumBaseAmount:       0.0001,
		MaximumBaseAmount:       100000,
		AmountStepIncrementSize: 0.0001,
		MinNotional:             0.0001,
		MaxIcebergParts:         100,
		MarketMaxQty:            1293.83077291,
		MaxTotalOrders:          200,
		MaxAlgoOrders:           5,
	}
	btcusdt := limits.MinMaxLevel{
		MinPrice:                0.01,
		MaxPrice:                1000000,
		PriceStepIncrementSize:  0.01,
		MultiplierUp:            2,
		MultiplierDown:          0.5,
		AveragePriceMinutes:     5,
		MinimumBaseAmount:       0.00001,
		MaximumBaseAmount:       9000,
		AmountStepIncrementSize: 0.00001,
		MinNotional:             5,
		MaxIcebergParts:         100,
		MarketMaxQty:            188.9393325,
		MaxTotalOrders:          200,
		MaxAlgoOrders:           5,
	}
	// The bid and ask price bands of the percent price by side filter differ, and the limits take the widest
	multibyte := limits.MinMaxLevel{
		MinPrice:                0.0001,
		MaxPrice:                1000,
		PriceStepIncrementSize:  0.0001,
		MultiplierUp:            2,
		MultiplierDown:          0.5,
		AveragePriceMinutes:     5,
		MinimumBaseAmount:       0.1,
		MaximumBaseAmount:       92141578,
		AmountStepIncrementSize: 0.1,
		MinNotional:             1,
		MaxIcebergParts:         100,
		MarketMaxQty:            63442.60541666,
		MaxTotalOrders:          200,
		MaxAlgoOrders:           5,
	}
	multibytePair := currency.NewPair(currency.NewCode("币安人生"), currency.USDT)
	limitsOf := func(a asset.Item, pair currency.Pair, l limits.MinMaxLevel) limits.MinMaxLevel {
		l.Key = key.NewExchangeAssetPair(e.Name, a, pair)
		return l
	}
	for _, tc := range []struct {
		a   asset.Item
		exp []limits.MinMaxLevel
	}{
		{
			// Spot limits include halted symbols, whose trading may resume, but not symbols that only trade as margin
			a: asset.Spot,
			exp: []limits.MinMaxLevel{
				limitsOf(asset.Spot, currency.NewPair(currency.ETH, currency.BTC), ethbtc),
				limitsOf(asset.Spot, currency.NewPair(currency.LTC, currency.BTC), limits.MinMaxLevel{
					MinPrice:                0.000001,
					MaxPrice:                100000,
					PriceStepIncrementSize:  0.000001,
					MultiplierUp:            5,
					MultiplierDown:          0.2,
					AveragePriceMinutes:     5,
					MinimumBaseAmount:       0.001,
					MaximumBaseAmount:       100000,
					AmountStepIncrementSize: 0.001,
					MinNotional:             0.0001,
					MaxIcebergParts:         100,
					MarketMaxQty:            2758.74901666,
					MaxTotalOrders:          200,
					MaxAlgoOrders:           5,
				}),
				limitsOf(asset.Spot, currency.NewPair(currency.NEO, currency.BTC), limits.MinMaxLevel{
					MinPrice:                0.00000001,
					MaxPrice:                100000,
					PriceStepIncrementSize:  0.00000001,
					MultiplierUp:            2,
					MultiplierDown:          0.5,
					AveragePriceMinutes:     5,
					MinimumBaseAmount:       0.01,
					MaximumBaseAmount:       100000,
					AmountStepIncrementSize: 0.01,
					MinNotional:             0.0001,
					MaxIcebergParts:         100,
					MarketMaxQty:            4136.18283333,
					MaxTotalOrders:          200,
					MaxAlgoOrders:           5,
				}),
				limitsOf(asset.Spot, currency.NewBTCUSDT(), btcusdt),
				limitsOf(asset.Spot, multibytePair, multibyte),
			},
		},
		{
			a: asset.Margin,
			exp: []limits.MinMaxLevel{
				limitsOf(asset.Margin, currency.NewPair(currency.ETH, currency.BTC), ethbtc),
				limitsOf(asset.Margin, currency.NewPair(currency.BNB, currency.ETH), limits.MinMaxLevel{
					MinPrice:                0.0001,
					MaxPrice:                1000,
					PriceStepIncrementSize:  0.0001,
					MultiplierUp:            2,
					MultiplierDown:          0.5,
					AveragePriceMinutes:     5,
					MinimumBaseAmount:       0.001,
					MaximumBaseAmount:       9000000,
					AmountStepIncrementSize: 0.001,
					MinNotional:             0.001,
					MaxIcebergParts:         100,
					MarketMaxQty:            760.528525,
					MaxTotalOrders:          200,
					MaxAlgoOrders:           5,
				}),
				limitsOf(asset.Margin, currency.NewBTCUSDT(), btcusdt),
				limitsOf(asset.Margin, multibytePair, multibyte),
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.FetchExchangeLimits(t.Context(), tc.a)
			require.NoError(t, err, "FetchExchangeLimits must not error")
			if !mockTests {
				assert.NotEmpty(t, result, "FetchExchangeLimits should return limits")
				return
			}
			assert.Equal(t, tc.exp, result, "FetchExchangeLimits should apply every filter of the asset's symbols")
		})
	}
}

func TestFetchUSDTMarginExchangeLimits(t *testing.T) {
	t.Parallel()
	t.Run("every filter", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/fapi/v1/exchangeInfo", r.URL.Path, "FetchUSDTMarginExchangeLimits should request the exchange information")
			_, err := w.Write([]byte(`{"symbols":[{"symbol":"TSLAUSDT","contractType":"TRADIFI_PERPETUAL","baseAsset":"TSLA","quoteAsset":"USDT","filters":[` +
				`{"filterType":"PRICE_FILTER","minPrice":"0.01","maxPrice":"100000","tickSize":"0.01"},` +
				`{"filterType":"LOT_SIZE","minQty":"0.01","maxQty":"10000","stepSize":"0.01"},` +
				`{"filterType":"MARKET_LOT_SIZE","minQty":"0.01","maxQty":"1000","stepSize":"0.01"},` +
				`{"filterType":"MAX_NUM_ORDERS","limit":200},{"filterType":"MAX_NUM_ALGO_ORDERS","limit":10},` +
				`{"filterType":"MIN_NOTIONAL","notional":"5"},` +
				`{"filterType":"PERCENT_PRICE","multiplierUp":"1.1500","multiplierDown":"0.8500","multiplierDecimal":"4"},` +
				`{"filterType":"POSITION_RISK_CONTROL","positionControlSide":"NONE"}]}]}`))
			assert.NoError(t, err, "Write should not error")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "SetRunningURL must not error")
		result, err := ex.FetchUSDTMarginExchangeLimits(t.Context())
		require.NoError(t, err, "FetchUSDTMarginExchangeLimits must not error")
		exp := []limits.MinMaxLevel{
			{
				Key:                     key.NewExchangeAssetPair(ex.Name, asset.USDTMarginedFutures, currency.NewPair(currency.NewCode("TSLA"), currency.USDT)),
				MinPrice:                0.01,
				MaxPrice:                100000,
				PriceStepIncrementSize:  0.01,
				MultiplierUp:            1.15,
				MultiplierDown:          0.85,
				MultiplierDecimal:       4,
				MinimumBaseAmount:       0.01,
				MaximumBaseAmount:       10000,
				AmountStepIncrementSize: 0.01,
				MinNotional:             5,
				MarketMinQty:            0.01,
				MarketMaxQty:            1000,
				MarketStepIncrementSize: 0.01,
				MaxTotalOrders:          200,
				MaxAlgoOrders:           10,
			},
		}
		assert.Equal(t, exp, result, "FetchUSDTMarginExchangeLimits should apply every documented filter")
	})

	result, err := e.FetchUSDTMarginExchangeLimits(t.Context())
	require.NoError(t, err, "FetchUSDTMarginExchangeLimits must not error")
	if !mockTests {
		assert.NotEmpty(t, result, "FetchUSDTMarginExchangeLimits should return limits")
		return
	}
	// A perpetual pairs its base and quote assets, a delivery contract its contract pair and delivery date
	exp := []limits.MinMaxLevel{
		{
			Key:                     key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, currency.NewBTCUSDT()),
			MinPrice:                556.8,
			MaxPrice:                4529764,
			PriceStepIncrementSize:  0.1,
			MultiplierUp:            1.05,
			MultiplierDown:          0.95,
			MultiplierDecimal:       4,
			MinimumBaseAmount:       0.001,
			MaximumBaseAmount:       1000,
			AmountStepIncrementSize: 0.001,
			MinNotional:             50,
			MarketMinQty:            0.001,
			MarketMaxQty:            120,
			MarketStepIncrementSize: 0.001,
			MaxTotalOrders:          200,
		},
		{
			Key:                     key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, currency.NewPair(currency.NewCode("BTCUSDT"), currency.NewCode("261225"))),
			MinPrice:                576.3,
			MaxPrice:                1000000,
			PriceStepIncrementSize:  0.1,
			MultiplierUp:            1.05,
			MultiplierDown:          0.95,
			MultiplierDecimal:       4,
			MinimumBaseAmount:       0.001,
			MaximumBaseAmount:       500,
			AmountStepIncrementSize: 0.001,
			MinNotional:             5,
			MarketMinQty:            0.001,
			MarketMaxQty:            1,
			MarketStepIncrementSize: 0.001,
			MaxTotalOrders:          200,
		},
	}
	assert.Equal(t, exp, result, "FetchUSDTMarginExchangeLimits should apply every filter of every symbol")
}

func TestFetchCoinMarginExchangeLimits(t *testing.T) {
	t.Parallel()
	result, err := e.FetchCoinMarginExchangeLimits(t.Context())
	require.NoError(t, err, "FetchCoinMarginExchangeLimits must not error")
	if !mockTests {
		assert.NotEmpty(t, result, "FetchCoinMarginExchangeLimits should return limits")
		return
	}
	exp := []limits.MinMaxLevel{
		{
			Key:                     key.NewExchangeAssetPair(e.Name, asset.CoinMarginedFutures, currency.NewPair(currency.NewCode("BTCUSD"), currency.PERP)),
			MinPrice:                1000,
			MaxPrice:                4520958,
			PriceStepIncrementSize:  0.1,
			MultiplierUp:            1.05,
			MultiplierDown:          0.95,
			MultiplierDecimal:       4,
			MinimumBaseAmount:       1,
			MaximumBaseAmount:       1000000,
			AmountStepIncrementSize: 1,
			MarketMinQty:            1,
			MarketMaxQty:            60000,
			MarketStepIncrementSize: 1,
			MaxTotalOrders:          200,
			MaxAlgoOrders:           20,
		},
		{
			Key:                     key.NewExchangeAssetPair(e.Name, asset.CoinMarginedFutures, currency.NewPair(currency.NewCode("BTCUSD"), currency.NewCode("261225"))),
			MinPrice:                1000,
			MaxPrice:                4671848,
			PriceStepIncrementSize:  0.1,
			MultiplierUp:            1.05,
			MultiplierDown:          0.95,
			MultiplierDecimal:       4,
			MinimumBaseAmount:       1,
			MaximumBaseAmount:       1000000,
			AmountStepIncrementSize: 1,
			MarketMinQty:            1,
			MarketMaxQty:            1000,
			MarketStepIncrementSize: 1,
			MaxTotalOrders:          200,
			MaxAlgoOrders:           20,
		},
	}
	assert.Equal(t, exp, result, "FetchCoinMarginExchangeLimits should apply every filter of every symbol")
}

func TestFetchOptionsExchangeLimits(t *testing.T) {
	t.Parallel()
	result, err := e.FetchOptionsExchangeLimits(t.Context())
	require.NoError(t, err, "FetchOptionsExchangeLimits must not error")
	if !mockTests {
		assert.NotEmpty(t, result, "FetchOptionsExchangeLimits should return limits")
		return
	}
	exp := []limits.MinMaxLevel{
		{
			Key:                     key.NewExchangeAssetPair(e.Name, asset.Options, currency.NewPair(currency.BTC, currency.NewCode("260410-72000-P"))),
			MinPrice:                965,
			MaxPrice:                8675,
			PriceStepIncrementSize:  5,
			MinimumBaseAmount:       0.01,
			MaximumBaseAmount:       200,
			AmountStepIncrementSize: 0.01,
		},
		{
			Key:                     key.NewExchangeAssetPair(e.Name, asset.Options, currency.NewPair(currency.BTC, currency.NewCode("261225-85000-C"))),
			MinPrice:                1375,
			MaxPrice:                12360,
			PriceStepIncrementSize:  5,
			MinimumBaseAmount:       0.01,
			MaximumBaseAmount:       200,
			AmountStepIncrementSize: 0.01,
		},
	}
	assert.Equal(t, exp, result, "FetchOptionsExchangeLimits should apply every filter of every symbol")
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates must reject a nil request")
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), StartDate: start, EndDate: end, IncludePredictedRate: true})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricalFundingRates must reject a predicted rate request")
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), StartDate: start, EndDate: end, PaymentCurrency: currency.DOGE})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricalFundingRates must reject a payment currency")
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Spot, Pair: currency.NewBTCUSDT(), StartDate: start, EndDate: end})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricalFundingRates must reject an unsupported asset")
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, StartDate: start, EndDate: end})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricalFundingRates must reject an empty pair")
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), StartDate: end, EndDate: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricalFundingRates must reject a start after the end")
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewPairWithDelimiter("BTCUSDT", "261225", currency.UnderscoreDelimiter), StartDate: start, EndDate: end})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetHistoricalFundingRates must reject a delivery contract")

	coinmPair := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		a   asset.Item
		exp *fundingrate.HistoricalRates
	}{
		{
			a: asset.USDTMarginedFutures,
			exp: &fundingrate.HistoricalRates{
				Exchange:   e.Name,
				Asset:      asset.USDTMarginedFutures,
				Pair:       currency.NewBTCUSDT(),
				StartDate:  start,
				EndDate:    end,
				LatestRate: fundingrate.Rate{Time: time.UnixMilli(1791273600000), Rate: decimal.MustFromFloat(-0.0000349)},
				FundingRates: []fundingrate.Rate{
					{Time: time.UnixMilli(1744128000000), Rate: decimal.MustFromFloat(0.00003134), Payment: decimal.MustFromFloat(-0.0125)},
					{Time: time.UnixMilli(1744156800000), Rate: decimal.MustFromFloat(0.00009568)},
				},
				PaymentSum:      decimal.MustFromFloat(-0.0125),
				PaymentCurrency: currency.USDT,
				TimeOfNextRate:  time.UnixMilli(1791302400000),
			},
		},
		{
			// Each payment is a few milliseconds after its funding time and goes to the nearest rate
			a: asset.CoinMarginedFutures,
			exp: &fundingrate.HistoricalRates{
				Exchange:   e.Name,
				Asset:      asset.CoinMarginedFutures,
				Pair:       coinmPair,
				StartDate:  start,
				EndDate:    end,
				LatestRate: fundingrate.Rate{Time: time.UnixMilli(1775548800000), Rate: decimal.MustFromFloat(0.00009543)},
				FundingRates: []fundingrate.Rate{
					{Time: time.UnixMilli(1744128000000), Rate: decimal.MustFromFloat(0.00005596)},
					{Time: time.UnixMilli(1744156800001), Rate: decimal.MustFromFloat(0.0001), Payment: decimal.MustFromFloat(-0.00000135)},
					{Time: time.UnixMilli(1744185600000), Rate: decimal.MustFromFloat(0.00008702), Payment: decimal.MustFromFloat(0.00000117)},
				},
				PaymentSum:      decimal.MustFromFloat(-0.00000135).Add(decimal.MustFromFloat(0.00000117)),
				PaymentCurrency: currency.BTC,
				TimeOfNextRate:  time.UnixMilli(1775577600000),
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetHistoricalFundingRates(mockContext(t), &fundingrate.HistoricalRatesRequest{
				Asset:           tc.a,
				Pair:            tc.exp.Pair,
				StartDate:       start,
				EndDate:         end,
				IncludePayments: sharedtestvalues.AreAPICredentialsSet(e),
			})
			require.NoError(t, err, "GetHistoricalFundingRates must not error")
			if !mockTests {
				assert.NotEmpty(t, result.FundingRates, "GetHistoricalFundingRates should return rates")
				return
			}
			assert.Equal(t, tc.exp, result, "GetHistoricalFundingRates should return every rate and payment")
		})
	}

	for _, tc := range []struct {
		a       asset.Item
		pair    currency.Pair
		replies map[string]string
	}{
		{asset.USDTMarginedFutures, currency.NewBTCUSDT(), map[string]string{"/fapi/v1/fundingInfo": "[]", "/fapi/v1/fundingRate": "[]", "/fapi/v1/premiumIndex": "[]"}},
		{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), map[string]string{"/dapi/v1/fundingInfo": "[]", "/dapi/v1/fundingRate": "[]", "/dapi/v1/premiumIndex": "[]"}},
	} {
		t.Run(tc.a.String()+" missing mark price", func(t *testing.T) {
			t.Parallel()
			ex := newRoutedTestExchange(t, tc.replies)
			_, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: tc.a, Pair: tc.pair, StartDate: start, EndDate: end})
			assert.ErrorIsf(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates should report a missing %s mark price", tc.a)
		})
	}

	t.Run("window end", func(t *testing.T) {
		t.Parallel()
		// A full page of rates ending at the window's end, or a millisecond before it, needs no request after the end, or one
		// of the last millisecond alone. A window of exactly 500 hours ends in a funding fee window of that millisecond too
		end := time.UnixMilli(1791273600000)
		start := end.Add(-500 * time.Hour)
		for _, tc := range []struct {
			name            string
			lastRate        time.Time
			includePayments bool
		}{
			{"full page ending at the end", end, false},
			{"full page ending a millisecond before the end", end.Add(-time.Millisecond), false},
			{"payments over 500 hours", time.Time{}, true},
		} {
			for _, product := range []struct {
				a      asset.Item
				pair   currency.Pair
				prefix string
			}{
				{asset.USDTMarginedFutures, currency.NewBTCUSDT(), "/fapi/v1/"},
				{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), "/dapi/v1/"},
			} {
				t.Run(tc.name+"/"+product.a.String(), func(t *testing.T) {
					t.Parallel()
					symbol := strings.ToUpper(product.pair.Base.String() + product.pair.Delimiter + product.pair.Quote.String())
					var mu sync.Mutex
					var ratePages int
					ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
						reply := "[]"
						q := r.URL.Query()
						switch strings.TrimPrefix(r.URL.Path, product.prefix) {
						case "fundingRate", "income":
							assert.LessOrEqual(t, q.Get("startTime"), q.Get("endTime"), "a request should not start after its end")
							assert.LessOrEqual(t, q.Get("endTime"), strconv.FormatInt(end.UnixMilli(), 10), "a request should not end after the window")
							if r.URL.Path != product.prefix+"fundingRate" || tc.lastRate.IsZero() {
								break
							}
							mu.Lock()
							ratePages++
							first := ratePages == 1
							mu.Unlock()
							if !first {
								break
							}
							rates := make([]string, fundingRatePageLimit)
							for i := range rates {
								rates[i] = fmt.Sprintf(`{"symbol":%q,"fundingTime":%d,"fundingRate":"0.0001"}`, symbol, tc.lastRate.Add(time.Duration(i+1-fundingRatePageLimit)*time.Millisecond).UnixMilli())
							}
							reply = "[" + strings.Join(rates, ",") + "]"
						case "premiumIndex":
							reply = fmt.Sprintf(`[{"symbol":%q,"lastFundingRate":"0.0001","nextFundingTime":1791302400000,"time":1791273600000}]`, symbol)
						case "fundingInfo":
						default:
							assert.Failf(t, "unexpected request", "%s should not be requested", r.URL.Path)
						}
						_, err := w.Write([]byte(reply))
						assert.NoError(t, err, "Write should not error")
					})
					url, err := ex.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
					require.NoError(t, err, "GetURL must not error")
					require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), url), "SetRunningURL must not error")
					require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestCoinMargined.String(), url), "SetRunningURL must not error")
					result, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: product.a, Pair: product.pair, StartDate: start, EndDate: end, IncludePayments: tc.includePayments})
					require.NoError(t, err, "GetHistoricalFundingRates must not error")
					if !tc.lastRate.IsZero() {
						assert.Len(t, result.FundingRates, fundingRatePageLimit, "GetHistoricalFundingRates should keep every rate fetched")
					}
				})
			}
		}
	})

	t.Run("paging", func(t *testing.T) {
		t.Parallel()
		// Binance returns at most limit rates from startTime to endTime, both inclusive and oldest first, so the page after a
		// full one starts after its last rate. Its funding times fall up to tens of milliseconds after the hour, its rates often
		// repeat or are negative and a contract's funding interval can be adjusted, so these do the same. Each window starts on
		// a rate or a millisecond after one and ends on a rate or a millisecond before one, with a rate outside it at each end
		first := time.UnixMilli(1704067200000)
		rateAt := func(i int) float64 { return float64(2*(i/2%7)-7) / 1e5 }
		for _, tc := range []struct {
			total                  int
			gap                    time.Duration
			startOnRate, endOnRate bool
		}{
			{fundingRatePageLimit - 1, 8 * time.Hour, true, false},
			{fundingRatePageLimit, 8 * time.Hour, false, false},
			{fundingRatePageLimit + 1, 8 * time.Hour, true, true},
			{2*fundingRatePageLimit - 1, time.Hour, false, false},
			{2*fundingRatePageLimit + 1, 8 * time.Hour, true, true},
		} {
			rates := make([]fundingrate.Rate, tc.total+2)
			for i := range rates {
				rates[i] = fundingrate.Rate{Time: first.Add(time.Duration(i)*tc.gap + time.Duration((16*i+10)%27)*time.Millisecond), Rate: decimal.MustFromFloat(rateAt(i))}
			}
			expRates := rates[1 : tc.total+1]
			start, end := rates[0].Time.Add(time.Millisecond), rates[tc.total+1].Time.Add(-time.Millisecond)
			if tc.startOnRate {
				start = expRates[0].Time
			}
			if tc.endOnRate {
				end = expRates[len(expRates)-1].Time
			}
			for _, product := range []struct {
				a      asset.Item
				pair   currency.Pair
				prefix string
			}{
				{asset.USDTMarginedFutures, currency.NewBTCUSDT(), "/fapi/v1/"},
				{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), "/dapi/v1/"},
			} {
				t.Run(fmt.Sprintf("%s/%d rates", product.a, tc.total), func(t *testing.T) {
					t.Parallel()
					symbol := strings.ToUpper(product.pair.Base.String() + product.pair.Delimiter + product.pair.Quote.String())
					var ratePages atomic.Int32
					ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
						reply := "[]"
						q := r.URL.Query()
						switch strings.TrimPrefix(r.URL.Path, product.prefix) {
						case "fundingRate":
							ratePages.Add(1)
							from, err := strconv.ParseInt(q.Get("startTime"), 10, 64)
							assert.NoError(t, err, "startTime should be a number")
							to, err := strconv.ParseInt(q.Get("endTime"), 10, 64)
							assert.NoError(t, err, "endTime should be a number")
							limit, err := strconv.Atoi(q.Get("limit"))
							assert.NoError(t, err, "limit should be a number")
							var page []string
							for i := range rates {
								if at := rates[i].Time.UnixMilli(); at >= from && at <= to && len(page) < limit {
									page = append(page, fmt.Sprintf(`{"symbol":%q,"fundingTime":%d,"fundingRate":%q}`, symbol, at, strconv.FormatFloat(rateAt(i), 'f', -1, 64)))
								}
							}
							reply = "[" + strings.Join(page, ",") + "]"
						case "premiumIndex":
							reply = fmt.Sprintf(`[{"symbol":%q,"lastFundingRate":"0.0001","nextFundingTime":1791302400000,"time":1791273600000}]`, symbol)
						case "fundingInfo":
						default:
							assert.Failf(t, "unexpected request", "%s should not be requested", r.URL.Path)
						}
						_, err := w.Write([]byte(reply))
						assert.NoError(t, err, "Write should not error")
					})
					serverURL, err := ex.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
					require.NoError(t, err, "GetURL must not error")
					require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), serverURL), "SetRunningURL must not error")
					require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestCoinMargined.String(), serverURL), "SetRunningURL must not error")
					result, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: product.a, Pair: product.pair, StartDate: start, EndDate: end})
					require.NoError(t, err, "GetHistoricalFundingRates must not error")
					require.Len(t, result.FundingRates, tc.total, "GetHistoricalFundingRates must return each rate in the window once")
					for i := range expRates {
						require.Equalf(t, expRates[i], result.FundingRates[i], "GetHistoricalFundingRates must return the window's rate %d in order", i)
					}
					assert.EqualValues(t, tc.total/fundingRatePageLimit+1, ratePages.Load(), "GetHistoricalFundingRates should stop after the page that is not full")
				})
			}
		}
	})
}

func TestFundingFeeWindows(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1744070400000)
	var result [][2]time.Time
	for windowStart, windowEnd := range fundingFeeWindows(start, start.Add(1200*time.Hour)) {
		result = append(result, [2]time.Time{windowStart, windowEnd})
		// A range that stops advancing fails here rather than growing until the test runs out of memory or time
		if !assert.Less(t, len(result), 10, "fundingFeeWindows should end") {
			break
		}
	}
	exp := [][2]time.Time{
		{start, start.Add(500*time.Hour - time.Millisecond)},
		{start.Add(500 * time.Hour), start.Add(1000*time.Hour - time.Millisecond)},
		{start.Add(1000 * time.Hour), start.Add(1200 * time.Hour)},
	}
	assert.Equal(t, exp, result, "fundingFeeWindows should split the range into windows of 500 funding times")
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, IncludePredictedRate: true})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetLatestFundingRates must reject a predicted rate request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLatestFundingRates must reject an unsupported asset")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures, Pair: currency.NewPairWithDelimiter("BTCUSD", "261225", currency.UnderscoreDelimiter)})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLatestFundingRates must reject a delivery contract")

	ex := newMarketTestExchange(t)
	pairs := marketTestPairs(t)
	// Contracts without an adjusted funding interval settle every eight hours. The two USDⓈ-M mark price recordings
	// were made a second apart
	usdtmRate := func(checked int64) fundingrate.LatestRateResponse {
		return fundingrate.LatestRateResponse{
			Exchange:       ex.Name,
			Asset:          asset.USDTMarginedFutures,
			Pair:           pairs[asset.USDTMarginedFutures],
			LatestRate:     fundingrate.Rate{Time: time.UnixMilli(1791273600000), Rate: decimal.MustFromFloat(-0.0000349)},
			TimeOfNextRate: time.UnixMilli(1791302400000),
			TimeChecked:    time.UnixMilli(checked),
		}
	}
	for _, tc := range []struct {
		name string
		req  *fundingrate.LatestRateRequest
		exp  []fundingrate.LatestRateResponse
	}{
		{
			name: "USDT-M pair",
			req:  &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: pairs[asset.USDTMarginedFutures]},
			exp:  []fundingrate.LatestRateResponse{usdtmRate(1791283486000)},
		},
		{
			// The mark prices of contracts that are not enabled are left out
			name: "USDT-M enabled pairs",
			req:  &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures},
			exp:  []fundingrate.LatestRateResponse{usdtmRate(1791283487000)},
		},
		{
			name: "COIN-M pair",
			req:  &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures, Pair: pairs[asset.CoinMarginedFutures]},
			exp: []fundingrate.LatestRateResponse{
				{
					Exchange:       ex.Name,
					Asset:          asset.CoinMarginedFutures,
					Pair:           pairs[asset.CoinMarginedFutures],
					LatestRate:     fundingrate.Rate{Time: time.UnixMilli(1775548800000), Rate: decimal.MustFromFloat(0.00009543)},
					TimeOfNextRate: time.UnixMilli(1775577600000),
					TimeChecked:    time.UnixMilli(1775571265000),
				},
			},
		},
		{
			// Delivery contracts and contracts that are not enabled are left out
			name: "COIN-M enabled pairs",
			req:  &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures},
			exp: []fundingrate.LatestRateResponse{
				{
					Exchange:       ex.Name,
					Asset:          asset.CoinMarginedFutures,
					Pair:           pairs[asset.CoinMarginedFutures],
					LatestRate:     fundingrate.Rate{Time: time.UnixMilli(1791273600000), Rate: decimal.MustFromFloat(0.00006375)},
					TimeOfNextRate: time.UnixMilli(1791302400000),
					TimeChecked:    time.UnixMilli(1791298529001),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := ex.GetLatestFundingRates(t.Context(), tc.req)
			require.NoError(t, err, "GetLatestFundingRates must not error")
			if !mockTests {
				assert.NotEmpty(t, result, "GetLatestFundingRates should return rates")
				return
			}
			assert.Equal(t, tc.exp, result, "GetLatestFundingRates should return every field of the rates")
		})
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  bool
		err  error
	}{
		{a: asset.Spot, pair: currency.NewBTCUSDT()},
		{a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(), exp: true},
		{a: asset.USDTMarginedFutures, pair: currency.NewPair(currency.BTC, currency.USDC), exp: true},
		{a: asset.USDTMarginedFutures, pair: currency.NewPair(currency.NewCode("TSLA"), currency.USDT), exp: true},
		{a: asset.USDTMarginedFutures, pair: currency.NewPairWithDelimiter("BTCUSDT", "261225", currency.UnderscoreDelimiter)},
		{a: asset.USDTMarginedFutures, err: currency.ErrCurrencyPairEmpty},
		{a: asset.CoinMarginedFutures, pair: currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), exp: true},
		{a: asset.CoinMarginedFutures, pair: currency.NewPairWithDelimiter("BTCUSD", "261225", currency.UnderscoreDelimiter)},
		{a: asset.CoinMarginedFutures, err: currency.ErrCurrencyPairEmpty},
	} {
		t.Run(tc.a.String()+" "+subtestName(tc.pair.String()), func(t *testing.T) {
			t.Parallel()
			result, err := e.IsPerpetualFutureCurrency(tc.a, tc.pair)
			require.ErrorIsf(t, err, tc.err, "IsPerpetualFutureCurrency must return the expected error for %s %s", tc.a, tc.pair)
			assert.Equalf(t, tc.exp, result, "IsPerpetualFutureCurrency should report whether %s %s is perpetual", tc.a, tc.pair)
		})
	}
}

func TestGetCollateralMode(t *testing.T) { //nolint:tparallel // the rejections run in order before the GetCollateralMode request
	t.Parallel()
	// The rejections run first, in order, so the GetCollateralMode request below also checks they leave nothing behind
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.CoinMarginedFutures, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			_, err := e.GetCollateralMode(t.Context(), a)
			require.ErrorIsf(t, err, asset.ErrNotSupported, "GetCollateralMode must reject %s", a)
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCollateralMode(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "GetCollateralMode must not error")
	if mockTests {
		assert.Equal(t, collateral.MultiMode, result, "GetCollateralMode should return Multi-Assets Mode")
	}
}

func TestSetCollateralMode(t *testing.T) { //nolint:tparallel // the rejections run in order before the SetCollateralMode request
	t.Parallel()
	err := e.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.PortfolioMode)
	require.ErrorIs(t, err, order.ErrCollateralInvalid, "SetCollateralMode must reject portfolio mode")
	// The rejections run first, in order, so the SetCollateralMode request below also checks they leave nothing behind
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.CoinMarginedFutures, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			err = e.SetCollateralMode(t.Context(), a, collateral.SingleMode)
			require.ErrorIsf(t, err, asset.ErrNotSupported, "SetCollateralMode must reject %s", a)
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.MultiMode), "SetCollateralMode should not error")
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangePositionMargin must reject a nil request")
	arg := &margin.PositionChangeRequest{}
	_, err = e.ChangePositionMargin(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangePositionMargin must reject an empty pair")
	arg.Pair = currency.NewBTCUSDT()
	// The cases share arg, which the checks below go on to change, so they run in order
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			arg.Asset = a
			_, err = e.ChangePositionMargin(t.Context(), arg)
			require.ErrorIsf(t, err, asset.ErrNotSupported, "ChangePositionMargin must reject %s", a)
		})
	}
	arg.Asset = asset.USDTMarginedFutures
	_, err = e.ChangePositionMargin(t.Context(), arg)
	require.ErrorIs(t, err, margin.ErrNewAllocatedMarginRequired, "ChangePositionMargin must require the new allocated margin")
	arg.NewAllocatedMargin = 1333337
	_, err = e.ChangePositionMargin(t.Context(), arg)
	require.ErrorIs(t, err, margin.ErrOriginalPositionMarginRequired, "ChangePositionMargin must require the original allocated margin")
	arg.OriginalAllocatedMargin = 1337
	arg.MarginType = margin.Multi
	_, err = e.ChangePositionMargin(t.Context(), arg)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "ChangePositionMargin must reject cross margin")
	arg.MarginType = margin.Isolated
	// The cases read arg and must precede the ChangePositionMargin calls below, so they run in order
	for _, invalid := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		t.Run(strconv.FormatFloat(invalid, 'f', -1, 64), func(t *testing.T) {
			_, err = e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Pair: arg.Pair, Asset: arg.Asset, MarginType: arg.MarginType, NewAllocatedMargin: invalid, OriginalAllocatedMargin: 1337})
			require.ErrorIsf(t, err, errInvalidMarginAmount, "ChangePositionMargin must reject a new allocated margin of %v", invalid)
			_, err = e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Pair: arg.Pair, Asset: arg.Asset, MarginType: arg.MarginType, NewAllocatedMargin: 1337, OriginalAllocatedMargin: invalid})
			require.ErrorIsf(t, err, errInvalidMarginAmount, "ChangePositionMargin must reject an original allocated margin of %v", invalid)
		})
	}
	arg.OriginalAllocatedMargin = arg.NewAllocatedMargin
	_, err = e.ChangePositionMargin(t.Context(), arg)
	require.ErrorIs(t, err, errMarginAmountRequired, "ChangePositionMargin must reject an unchanged margin")
	_, err = new(Exchange).ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Pair: currency.NewBTCUSDT(), Asset: asset.USDTMarginedFutures, MarginType: margin.Isolated, OriginalAllocatedMargin: 10, NewAllocatedMargin: 15})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "ChangePositionMargin must return the error changing the margin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	for _, tc := range []struct {
		name               string
		a                  asset.Item
		original, newValue float64
		side               string
	}{
		{"USDⓈ-M add", asset.USDTMarginedFutures, 10, 15, "long"},
		{"USDⓈ-M reduce", asset.USDTMarginedFutures, 5, 2.5, "SHORT"},
		{"COIN-M add", asset.CoinMarginedFutures, 100, 200, "BOTH"},
		{"COIN-M reduce", asset.CoinMarginedFutures, 0.0002, 0.0001, "BOTH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pair := assetToTradablePairMap[tc.a]
			resp, err := e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
				Pair:                    pair,
				Asset:                   tc.a,
				MarginType:              margin.Isolated,
				OriginalAllocatedMargin: tc.original,
				NewAllocatedMargin:      tc.newValue,
				MarginSide:              tc.side,
			})
			require.NoError(t, err, "ChangePositionMargin must not error")
			exp := &margin.PositionChangeResponse{Exchange: e.Name, Pair: pair, Asset: tc.a, MarginType: margin.Isolated, AllocatedMargin: tc.newValue}
			assert.Equal(t, exp, resp, "ChangePositionMargin should return the position's new margin")
		})
	}
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionSummary must reject a nil request")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{CalculateOffline: true})
	require.ErrorIs(t, err, common.ErrCannotCalculateOffline, "GetFuturesPositionSummary must reject an offline calculation")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesPositionSummary must reject an empty pair")
	// The cases assign the test's err, as the checks before them do, so they run in order
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: a, Pair: currency.NewBTCUSDT()})
			require.ErrorIsf(t, err, asset.ErrNotSupported, "GetFuturesPositionSummary must reject %s", a)
		})
	}

	t.Run("position summaries", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		}
		ensureTradablePairs(t)
		d := decimal.MustFromFloat
		usdtmTotal, usdtmFree, usdtmMaintenance := d(126.72469206), d(126.26497206), d(0.405)
		usdtm := &futures.PositionSummary{
			Pair:                         currency.NewBTCUSDT(),
			Asset:                        asset.USDTMarginedFutures,
			MarginType:                   margin.Isolated,
			CollateralMode:               collateral.MultiMode,
			Currency:                     currency.USDT,
			IsolatedMargin:               d(8),
			NotionalSize:                 d(81),
			Leverage:                     d(100),
			MaintenanceMarginRequirement: usdtmMaintenance,
			InitialMarginRequirement:     d(8.1),
			EstimatedLiquidationPrice:    d(72400.5),
			CollateralUsed:               d(8),
			MarkPrice:                    d(81000),
			CurrentSize:                  d(0.001),
			ContractSettlementType:       futures.Linear,
			AverageOpenPrice:             d(80000),
			UnrealisedPNL:                d(1),
			MaintenanceMarginFraction:    usdtmMaintenance.Div(usdtmTotal).Mul(decimal.NewFromInt(100)),
			FreeCollateral:               usdtmFree,
			TotalCollateral:              usdtmTotal,
			FrozenBalance:                usdtmTotal.Sub(usdtmFree),
		}
		coinmTotal, coinmFree, coinmMaintenance := d(0.00241969), d(0.00237355), d(0.00001102)
		coinm := &futures.PositionSummary{
			Pair:                         currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter),
			Asset:                        asset.CoinMarginedFutures,
			MarginType:                   margin.Isolated,
			CollateralMode:               collateral.SingleMode,
			Currency:                     currency.BTC,
			IsolatedMargin:               d(0.0000453),
			NotionalSize:                 d(0.00348561),
			Leverage:                     d(8),
			MaintenanceMarginRequirement: coinmMaintenance,
			InitialMarginRequirement:     d(0.0000441),
			EstimatedLiquidationPrice:    d(74502.8),
			CollateralUsed:               d(0.0000453),
			MarkPrice:                    d(86068.2),
			CurrentSize:                  d(3),
			ContractSettlementType:       futures.Inverse,
			AverageOpenPrice:             d(85642.5),
			UnrealisedPNL:                d(-0.00000081),
			MaintenanceMarginFraction:    coinmMaintenance.Div(coinmTotal).Mul(decimal.NewFromInt(100)),
			FreeCollateral:               coinmFree,
			TotalCollateral:              coinmTotal,
			FrozenBalance:                coinmTotal.Sub(coinmFree),
		}
		for _, tc := range []struct {
			name string
			req  *futures.PositionSummaryRequest
			exp  *futures.PositionSummary
			err  error
		}{
			{name: "USDⓈ-M", req: &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures}, exp: usdtm},
			{name: "USDⓈ-M hedge mode long", req: &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Direction: order.Long}, err: futures.ErrNoPositionsFound},
			{name: "COIN-M", req: &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, UnderlyingPair: currency.NewBTCUSD()}, exp: coinm},
			{name: "COIN-M margin asset from exchange information", req: &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures}, exp: coinm},
			{name: "COIN-M unknown collateral", req: &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, UnderlyingPair: currency.NewPair(currency.ETH, currency.USD)}, err: currency.ErrCurrencyNotFound},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				tc.req.Pair = assetToTradablePairMap[tc.req.Asset]
				resp, err := e.GetFuturesPositionSummary(t.Context(), tc.req)
				require.ErrorIs(t, err, tc.err, "GetFuturesPositionSummary must return the expected error")
				if mockTests {
					assert.Equal(t, tc.exp, resp, "GetFuturesPositionSummary should return the position's summary")
				}
			})
		}

		t.Run("USDⓈ-M single asset mode", func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				var body string
				switch r.URL.Path {
				case "/fapi/v2/account":
					body = `{"multiAssetsMargin":false,"totalWalletBalance":"126.7","availableBalance":"126.2","assets":[{"asset":"USDC","walletBalance":"100","availableBalance":"90"},{"asset":"USDT","walletBalance":"23.72469206","availableBalance":"23.26497206"}],"positions":[{"symbol":"BTCUSDT","leverage":"100","isolated":false,"positionSide":"BOTH"}]}`
				case "/fapi/v3/positionRisk":
					body = `[{"symbol":"BTCUSDT","positionSide":"BOTH","positionAmt":"0.001","entryPrice":"80000","markPrice":"81000","unRealizedProfit":"1","liquidationPrice":"72400.5","notional":"81","marginAsset":"USDT","maintMargin":"0.405","positionInitialMargin":"8.1"}]`
				default:
					assert.Failf(t, "GetFuturesPositionSummary should only request the account and its positions", "requested %s", r.URL.Path)
				}
				_, err := w.Write([]byte(body))
				assert.NoError(t, err, "Writing the response should not error")
			})
			resp, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT()})
			require.NoError(t, err, "GetFuturesPositionSummary must not error")
			d := decimal.MustFromFloat
			total, free, maintenance := d(23.72469206), d(23.26497206), d(0.405)
			exp := &futures.PositionSummary{
				Pair:                         currency.NewBTCUSDT(),
				Asset:                        asset.USDTMarginedFutures,
				MarginType:                   margin.Multi,
				CollateralMode:               collateral.SingleMode,
				Currency:                     currency.USDT,
				NotionalSize:                 d(81),
				Leverage:                     d(100),
				MaintenanceMarginRequirement: maintenance,
				InitialMarginRequirement:     d(8.1),
				EstimatedLiquidationPrice:    d(72400.5),
				CollateralUsed:               d(8.1),
				MarkPrice:                    d(81000),
				CurrentSize:                  d(0.001),
				ContractSettlementType:       futures.Linear,
				AverageOpenPrice:             d(80000),
				UnrealisedPNL:                d(1),
				MaintenanceMarginFraction:    maintenance.Div(total).Mul(decimal.NewFromInt(100)),
				FreeCollateral:               free,
				TotalCollateral:              total,
				FrozenBalance:                total.Sub(free),
			}
			assert.Equal(t, exp, resp, "GetFuturesPositionSummary should take single asset mode collateral from the margin asset")
		})

		coinM := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
		const (
			cmInfo     = `{"symbols":[{"symbol":"BTCUSD_PERP","marginAsset":"BTC"}]}`
			cmAccount  = `{"assets":[{"asset":"BTC","walletBalance":"1"}],"positions":[{"symbol":"BTCUSD_PERP","positionSide":"BOTH","leverage":"10"}]}`
			umAccount  = `{"assets":[],"positions":[{"symbol":"BTCUSDT","positionSide":"BOTH","leverage":"10"}]}`
			noPosition = `{"assets":[{"asset":"BTC","walletBalance":"1"}],"positions":[]}`
		)
		for _, tc := range []struct {
			name    string
			a       asset.Item
			pair    currency.Pair
			replies map[string]string
			err     error
		}{
			{"USDⓈ-M position information", asset.USDTMarginedFutures, currency.NewBTCUSDT(), map[string]string{"/fapi/v2/account": umAccount, "/fapi/v3/positionRisk": "[]"}, futures.ErrNoPositionsFound},
			{"COIN-M margin asset", asset.CoinMarginedFutures, coinM, map[string]string{"/dapi/v1/exchangeInfo": `{"symbols":[]}`}, currency.ErrCurrencyNotFound},
			{"COIN-M account position", asset.CoinMarginedFutures, coinM, map[string]string{"/dapi/v1/exchangeInfo": cmInfo, "/dapi/v1/account": noPosition}, futures.ErrNoPositionsFound},
			{"COIN-M position information", asset.CoinMarginedFutures, coinM, map[string]string{"/dapi/v1/exchangeInfo": cmInfo, "/dapi/v1/account": cmAccount, "/dapi/v1/positionRisk": "[]"}, futures.ErrNoPositionsFound},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRoutedTestExchange(t, tc.replies)
				_, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: tc.a, Pair: tc.pair})
				assert.ErrorIsf(t, err, tc.err, "GetFuturesPositionSummary should report the missing %s", tc.name)
			})
		}

		// Single-asset mode collateral comes from the margin asset's own row; rows of assets that BTCUSDT contains or ends
		// with must not stand in for it. Multi-Assets Mode collateral is the account's and needs no row
		const nearMisses = `[{"asset":"DT","walletBalance":"7"},{"asset":"USD","walletBalance":"8"},{"asset":"T","walletBalance":"9"}]`
		for _, tc := range []struct {
			name        string
			multiAssets bool
			assets      string
			err         error
		}{
			{"single asset mode without asset rows", false, `[]`, currency.ErrCurrencyNotFound},
			{"single asset mode with near miss rows", false, nearMisses, currency.ErrCurrencyNotFound},
			{"Multi-Assets Mode with near miss rows", true, nearMisses, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRoutedTestExchange(t, map[string]string{
					"/fapi/v2/account":      `{"multiAssetsMargin":` + strconv.FormatBool(tc.multiAssets) + `,"assets":` + tc.assets + `,"positions":[{"symbol":"BTCUSDT","positionSide":"BOTH","leverage":"10"}]}`,
					"/fapi/v3/positionRisk": `[{"symbol":"BTCUSDT","positionSide":"BOTH","positionAmt":"0.001","marginAsset":"USDT"}]`,
				})
				_, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT()})
				assert.ErrorIsf(t, err, tc.err, "GetFuturesPositionSummary should return the expected error for assets %s in Multi-Assets Mode %t", tc.assets, tc.multiAssets)
			})
		}
	})

	// A Hedge Mode position summary without a direction, as the RPC server requests it, is of the symbol's one open side,
	// and both sides being open requires a direction
	t.Run("hedge mode", func(t *testing.T) {
		t.Parallel()
		d := decimal.MustFromFloat
		hundred := decimal.NewFromInt(100)
		// position returns a position information entry, whose values Binance reports only while the side is open
		position := func(symbol, side, amount, values string) string {
			if amount == "0" {
				return `{"symbol":"` + symbol + `","positionSide":"` + side + `","positionAmt":"0"}`
			}
			return `{"symbol":"` + symbol + `","positionSide":"` + side + `","positionAmt":"` + amount + `",` + values + `}`
		}
		// Another symbol with both sides open, listed after the requested one, must not stand in for it
		usdtmExchange := func(t *testing.T, long, short string) *Exchange {
			t.Helper()
			return newRoutedTestExchange(t, map[string]string{
				"/fapi/v2/account": `{"multiAssetsMargin":false,"totalWalletBalance":"1000","availableBalance":"900","assets":[{"asset":"USDT","walletBalance":"1000","availableBalance":"900"}],"positions":[` +
					`{"symbol":"BTCUSDT","positionSide":"LONG","positionAmt":"` + long + `","leverage":"10","isolated":false},` +
					`{"symbol":"BTCUSDT","positionSide":"SHORT","positionAmt":"` + short + `","leverage":"20","isolated":true},` +
					`{"symbol":"ETHUSDT","positionSide":"LONG","positionAmt":"1","leverage":"5"},{"symbol":"ETHUSDT","positionSide":"SHORT","positionAmt":"-1","leverage":"5"}]}`,
				"/fapi/v3/positionRisk": `[` +
					position("BTCUSDT", "LONG", long, `"entryPrice":"80000","markPrice":"81000","unRealizedProfit":"10","liquidationPrice":"70000","notional":"810","marginAsset":"USDT","maintMargin":"4","positionInitialMargin":"81"`) + `,` +
					position("BTCUSDT", "SHORT", short, `"entryPrice":"82000","markPrice":"81000","unRealizedProfit":"20","liquidationPrice":"90000","notional":"-1620","marginAsset":"USDT","maintMargin":"8","positionInitialMargin":"162","isolatedWallet":"170"`) + `]`,
			})
		}
		longSummary := &futures.PositionSummary{
			Pair: currency.NewBTCUSDT(), Asset: asset.USDTMarginedFutures, MarginType: margin.Multi, CollateralMode: collateral.SingleMode, Currency: currency.USDT,
			NotionalSize: d(810), Leverage: d(10), MaintenanceMarginRequirement: d(4), InitialMarginRequirement: d(81), EstimatedLiquidationPrice: d(70000),
			CollateralUsed: d(81), MarkPrice: d(81000), CurrentSize: d(0.01), ContractSettlementType: futures.Linear, AverageOpenPrice: d(80000),
			UnrealisedPNL: d(10), MaintenanceMarginFraction: d(4).Div(d(1000)).Mul(hundred), FreeCollateral: d(900), TotalCollateral: d(1000), FrozenBalance: d(100),
		}
		shortSummary := &futures.PositionSummary{
			Pair: currency.NewBTCUSDT(), Asset: asset.USDTMarginedFutures, MarginType: margin.Isolated, CollateralMode: collateral.SingleMode, Currency: currency.USDT,
			IsolatedMargin: d(170), NotionalSize: d(-1620), Leverage: d(20), MaintenanceMarginRequirement: d(8), InitialMarginRequirement: d(162),
			EstimatedLiquidationPrice: d(90000), CollateralUsed: d(170), MarkPrice: d(81000), CurrentSize: d(-0.02), ContractSettlementType: futures.Linear,
			AverageOpenPrice: d(82000), UnrealisedPNL: d(20), MaintenanceMarginFraction: d(8).Div(d(1000)).Mul(hundred), FreeCollateral: d(900),
			TotalCollateral: d(1000), FrozenBalance: d(100),
		}
		for _, tc := range []struct {
			name        string
			long, short string
			direction   order.Side
			exp         *futures.PositionSummary
			err         error
		}{
			{name: "long open", long: "0.01", short: "0", exp: longSummary},
			{name: "short open", long: "0", short: "-0.02", exp: shortSummary},
			{name: "both open", long: "0.01", short: "-0.02", err: errPositionDirectionRequired},
			{name: "both open, short requested", long: "0.01", short: "-0.02", direction: order.Short, exp: shortSummary},
			{name: "both open, long requested", long: "0.01", short: "-0.02", direction: order.Long, exp: longSummary},
			{name: "none open", long: "0", short: "0", err: futures.ErrNoPositionsFound},
		} {
			t.Run("USDⓈ-M "+tc.name, func(t *testing.T) {
				t.Parallel()
				ex := usdtmExchange(t, tc.long, tc.short)
				resp, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Direction: tc.direction})
				require.ErrorIs(t, err, tc.err, "GetFuturesPositionSummary must return the expected error")
				assert.Equal(t, tc.exp, resp, "GetFuturesPositionSummary should summarise the expected side's position")
			})
		}

		coinmExchange := func(t *testing.T, long, short string) *Exchange {
			t.Helper()
			return newRoutedTestExchange(t, map[string]string{
				"/dapi/v1/account": `{"assets":[{"asset":"BTC","walletBalance":"1","availableBalance":"0.9"}],"positions":[` +
					`{"symbol":"BTCUSD_PERP","positionSide":"LONG","positionAmt":"` + long + `","isolated":false,"maintMargin":"0.0001","positionInitialMargin":"0.0003"},` +
					`{"symbol":"BTCUSD_PERP","positionSide":"SHORT","positionAmt":"` + short + `","isolated":false,"maintMargin":"0.0002","positionInitialMargin":"0.0006"},` +
					`{"symbol":"ETHUSD_PERP","positionSide":"LONG","positionAmt":"10"},{"symbol":"ETHUSD_PERP","positionSide":"SHORT","positionAmt":"-10"}]}`,
				"/dapi/v1/positionRisk": `[` +
					position("BTCUSD_PERP", "LONG", long, `"entryPrice":"86000","markPrice":"86100","unRealizedProfit":"0.00001","liquidationPrice":"70000","leverage":"10","notionalValue":"0.0035"`) + `,` +
					position("BTCUSD_PERP", "SHORT", short, `"entryPrice":"87000","markPrice":"86100","unRealizedProfit":"0.00002","liquidationPrice":"99000","leverage":"10","notionalValue":"-0.007"`) + `]`,
			})
		}
		for _, tc := range []struct {
			name        string
			long, short string
			exp         *futures.PositionSummary
			err         error
		}{
			{
				name: "short open", long: "0", short: "-6",
				exp: &futures.PositionSummary{
					Pair: testCoinMarginedFuturesPair, Asset: asset.CoinMarginedFutures, MarginType: margin.Multi, CollateralMode: collateral.SingleMode, Currency: currency.BTC,
					NotionalSize: d(-0.007), Leverage: d(10), MaintenanceMarginRequirement: d(0.0002), InitialMarginRequirement: d(0.0006),
					EstimatedLiquidationPrice: d(99000), CollateralUsed: d(0.0006), MarkPrice: d(86100), CurrentSize: d(-6), ContractSettlementType: futures.Inverse,
					AverageOpenPrice: d(87000), UnrealisedPNL: d(0.00002), MaintenanceMarginFraction: d(0.0002).Div(d(1)).Mul(hundred), FreeCollateral: d(0.9),
					TotalCollateral: d(1), FrozenBalance: d(1).Sub(d(0.9)),
				},
			},
			{name: "both open", long: "3", short: "-6", err: errPositionDirectionRequired},
		} {
			t.Run("COIN-M "+tc.name, func(t *testing.T) {
				t.Parallel()
				ex := coinmExchange(t, tc.long, tc.short)
				resp, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: testCoinMarginedFuturesPair, UnderlyingPair: currency.NewBTCUSD()})
				require.ErrorIs(t, err, tc.err, "GetFuturesPositionSummary must return the expected error")
				assert.Equal(t, tc.exp, resp, "GetFuturesPositionSummary should summarise the expected side's position")
			})
		}
	})
}

func TestPagedOrderHistory(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1744103854944)
	type testOrder struct {
		id   uint64
		time time.Time
	}
	type window struct{ start, end time.Time }
	timeOf := func(o *testOrder) time.Time { return o.time }
	idOf := func(o *testOrder) uint64 { return o.id }
	at := func(id uint64, offset time.Duration) testOrder { return testOrder{id, start.Add(offset)} }
	for _, tc := range []struct {
		name     string
		begin    time.Time
		end      time.Time
		fromID   uint64
		maxSpan  time.Duration
		orders   []testOrder
		requests []window
		exp      []uint64
	}{
		{name: "no window", requests: []window{{}}, orders: []testOrder{at(1, 0)}, exp: []uint64{1}},
		{name: "from an order ID", end: start.Add(72 * time.Hour), fromID: 7, maxSpan: time.Hour, requests: []window{{}}, orders: []testOrder{at(1, 0), at(2, 0), at(3, 0)}, exp: []uint64{1, 2}},
		{name: "one span", end: start.Add(time.Hour), maxSpan: 24 * time.Hour, requests: []window{{start, start.Add(time.Hour)}}, orders: []testOrder{at(1, 0)}, exp: []uint64{1}},
		{
			name: "split", end: start.Add(50 * time.Hour), maxSpan: 24 * time.Hour,
			orders: []testOrder{at(1, 0), at(2, 24*time.Hour), at(3, 49*time.Hour)},
			requests: []window{
				{start, start.Add(24 * time.Hour)},
				{start.Add(24*time.Hour - time.Millisecond), start.Add(24 * time.Hour)},
				{start.Add(24 * time.Hour), start.Add(48 * time.Hour)},
				{start.Add(48 * time.Hour), start.Add(50 * time.Hour)},
			},
			exp: []uint64{1, 2, 3},
		},
		{
			name: "full page ending at the window's end", end: start.Add(time.Hour), maxSpan: 24 * time.Hour,
			orders: []testOrder{at(1, 0), at(2, time.Hour), at(3, time.Hour)},
			requests: []window{
				{start, start.Add(time.Hour)},
				{start.Add(time.Hour - time.Millisecond), start.Add(time.Hour)},
			},
			exp: []uint64{1, 2, 3},
		},
		{
			name: "page reaching back before its start", end: start.Add(time.Hour), maxSpan: 24 * time.Hour,
			orders: []testOrder{at(1, time.Minute), at(2, time.Minute), at(3, 0)},
			requests: []window{
				{start, start.Add(time.Hour)},
				{start.Add(time.Minute), start.Add(time.Hour)},
				{start.Add(time.Minute + time.Millisecond), start.Add(time.Hour)},
			},
			exp: []uint64{1, 2},
		},
		{
			name: "paged across a shared millisecond", end: start.Add(time.Hour), maxSpan: 24 * time.Hour,
			orders: []testOrder{at(1, 0), at(2, time.Minute), at(3, time.Minute), at(4, 2*time.Minute), at(5, 3*time.Minute)},
			requests: []window{
				{start, start.Add(time.Hour)},
				{start.Add(time.Minute), start.Add(time.Hour)},
				{start.Add(time.Minute), start.Add(time.Hour)},
				{start.Add(time.Minute + time.Millisecond), start.Add(time.Hour)},
				{start.Add(3 * time.Minute), start.Add(time.Hour)},
			},
			exp: []uint64{1, 2, 3, 4, 5},
		},
		{
			// Binance takes milliseconds, so paging from the last order's millisecond to an end later in that millisecond
			// would send a window starting where it ends
			name: "bounds within milliseconds", begin: start.Add(300 * time.Microsecond), end: start.Add(time.Hour + 500*time.Microsecond), maxSpan: 24 * time.Hour,
			orders: []testOrder{at(1, 0), at(2, time.Hour), at(3, time.Hour)},
			requests: []window{
				{start, start.Add(time.Hour)},
				{start.Add(time.Hour - time.Millisecond), start.Add(time.Hour)},
			},
			exp: []uint64{1, 2, 3},
		},
		{
			name: "window within one millisecond", begin: start.Add(100 * time.Microsecond), end: start.Add(600 * time.Microsecond), maxSpan: 24 * time.Hour,
			orders:   []testOrder{at(1, 0), at(2, 0)},
			requests: []window{{start.Add(100 * time.Microsecond), start.Add(600 * time.Microsecond)}},
			exp:      []uint64{1, 2},
		},
		{name: "unsplit", end: start.Add(50 * time.Hour), requests: []window{{start, start.Add(50 * time.Hour)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			windowStart := start
			switch {
			case tc.end.IsZero():
				windowStart = time.Time{}
			case !tc.begin.IsZero():
				windowStart = tc.begin
			}
			var requests []window
			// Serve like Binance: the first two orders within the window's milliseconds, or from the order ID
			got, err := pagedOrderHistory(windowStart, tc.end, tc.fromID, tc.maxSpan, 2, func(from, to time.Time) ([]testOrder, error) {
				requests = append(requests, window{from, to})
				// A pager that keeps requesting fails here rather than hanging the test
				if !assert.LessOrEqual(t, len(requests), 100, "pagedOrderHistory should stop requesting") {
					return nil, errAPIResponse
				}
				var page []testOrder
				for _, o := range tc.orders {
					if len(page) < 2 && (from.IsZero() || tc.fromID != 0 || (o.time.UnixMilli() >= from.UnixMilli() && o.time.UnixMilli() <= to.UnixMilli())) {
						page = append(page, o)
					}
				}
				return page, nil
			}, timeOf, idOf)
			require.NoError(t, err, "pagedOrderHistory must not error")
			assert.Equal(t, tc.requests, requests, "pagedOrderHistory should request the expected windows")
			var ids []uint64
			for i := range got {
				ids = append(ids, got[i].id)
			}
			assert.Equal(t, tc.exp, ids, "pagedOrderHistory should return every order once")
		})
	}
	_, err := pagedOrderHistory(start, start.Add(time.Hour), 0, time.Hour, 2, func(time.Time, time.Time) ([]testOrder, error) {
		return nil, errAPIResponse
	}, timeOf, idOf)
	assert.ErrorIs(t, err, errAPIResponse, "pagedOrderHistory should return a fetch error")

	// A window ending more than a span in the future stops before a span starting after now, which the requests'
	// validation refuses
	now := time.Now()
	var requests []window
	got, err := pagedOrderHistory(now.Add(-time.Hour), now.Add(48*time.Hour), 0, 24*time.Hour, 2, func(from, to time.Time) ([]testOrder, error) {
		requests = append(requests, window{from, to})
		if !assert.LessOrEqual(t, len(requests), 100, "pagedOrderHistory should stop requesting") {
			return nil, errAPIResponse
		}
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
		return []testOrder{{1, now}}, nil
	}, timeOf, idOf)
	require.NoError(t, err, "pagedOrderHistory must not error for a window ending in the future")
	assert.Equal(t, []testOrder{{1, now}}, got, "pagedOrderHistory should return the orders up to now")
	assert.Equal(t, []window{{now.Add(-time.Hour).Truncate(time.Millisecond), now.Add(23 * time.Hour).Truncate(time.Millisecond)}}, requests, "pagedOrderHistory should request no span starting after now")

	// A spot history longer than Binance's 24 hours is requested in spans it accepts
	var spans []time.Duration
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		startTime, err := strconv.ParseInt(r.URL.Query().Get("startTime"), 10, 64)
		assert.NoError(t, err, "startTime should be a number")
		endTime, err := strconv.ParseInt(r.URL.Query().Get("endTime"), 10, 64)
		assert.NoError(t, err, "endTime should be a number")
		spans = append(spans, time.Duration(endTime-startTime)*time.Millisecond)
		_, err = w.Write([]byte("[]"))
		assert.NoError(t, err, "Write should not error")
	})
	_, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Type: order.AnyType, Side: order.AnySide, StartTime: start, EndTime: start.Add(50 * time.Hour)})
	require.NoError(t, err, "GetOrderHistory must not error")
	require.Len(t, spans, 3, "GetOrderHistory must request three spans")
	for _, span := range spans {
		assert.LessOrEqual(t, span, spotOrderHistorySpan, "GetOrderHistory should request spans Binance accepts")
	}

	t.Run("windows from an order ID", func(t *testing.T) {
		t.Parallel()
		// A request from an order ID carries no window, which Binance would let override the ID or refuse, so the window
		// applies to the orders from the ID in the whole milliseconds Binance compares
		start := time.UnixMilli(1744103854944)
		type testOrder struct {
			id   uint64
			time time.Time
		}
		timeOf := func(o *testOrder) time.Time { return o.time }
		idOf := func(o *testOrder) uint64 { return o.id }
		orders := []testOrder{{5, start}, {6, start.Add(time.Millisecond)}, {7, start.Add(2 * time.Millisecond)}}
		for _, tc := range []struct {
			name       string
			begin, end time.Time
			exp        []uint64
		}{
			{name: "no window", exp: []uint64{5, 6, 7}},
			{name: "window within a millisecond", begin: start.Add(time.Millisecond + 300*time.Microsecond), end: start.Add(time.Millisecond + 600*time.Microsecond), exp: []uint64{6}},
			{name: "start only", begin: start.Add(time.Millisecond), exp: []uint64{6, 7}},
			{name: "end only", end: start.Add(time.Millisecond), exp: []uint64{5, 6}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var requests int
				got, err := pagedOrderHistory(tc.begin, tc.end, 5, time.Hour, 1000, func(from, to time.Time) ([]testOrder, error) {
					requests++
					assert.Zero(t, from, "a request from an order ID should carry no start")
					assert.Zero(t, to, "a request from an order ID should carry no end")
					return slices.Clone(orders), nil
				}, timeOf, idOf)
				require.NoError(t, err, "pagedOrderHistory must not error")
				ids := make([]uint64, len(got))
				for i := range got {
					ids[i] = got[i].id
				}
				assert.Equal(t, tc.exp, ids, "pagedOrderHistory should return the orders from the ID within the window")
				assert.Equal(t, 1, requests, "pagedOrderHistory should make one request from the order ID")
			})
		}
		var requests int
		_, err := pagedOrderHistory(start.Add(time.Millisecond), start, 5, time.Hour, 1000, func(time.Time, time.Time) ([]testOrder, error) {
			requests++
			return nil, nil
		}, timeOf, idOf)
		assert.ErrorIs(t, err, common.ErrStartAfterEnd, "pagedOrderHistory should reject a window ending before it starts")
		assert.Zero(t, requests, "pagedOrderHistory should not request orders for a window ending before it starts")
	})
}

func TestFilterOrders(t *testing.T) { //nolint:tparallel // the cases share the orders slice the last check reads
	t.Parallel()
	orders := []order.Detail{
		{OrderID: "1", Type: order.StopLimit, Side: order.Sell},
		{OrderID: "2", Type: order.StopMarket, Side: order.Sell},
		{OrderID: "3", Type: order.TakeProfitLimit, Side: order.Sell},
		{OrderID: "4", Type: order.TakeProfitMarket, Side: order.Buy},
		{OrderID: "5", Type: order.TrailingStop, Side: order.Sell},
		{OrderID: "6", Type: order.Limit, Side: order.Sell},
	}
	// The cases filter one orders slice, which the check after them reads, so they run in order
	for _, tc := range []struct {
		orderType order.Type
		side      order.Side
		exp       []string
	}{
		{order.Stop, order.AnySide, []string{"1", "2"}},
		{order.TakeProfit, order.AnySide, []string{"3", "4"}},
		{order.TakeProfit, order.Sell, []string{"3"}},
		{order.StopLimit, order.AnySide, []string{"1"}},
		{order.Limit, order.AnySide, []string{"6"}},
		{order.AnyType, order.AnySide, []string{"1", "2", "3", "4", "5", "6"}},
	} {
		t.Run(tc.orderType.String()+" "+tc.side.String(), func(t *testing.T) {
			got := e.filterOrders(&order.MultiOrderRequest{Type: tc.orderType, Side: tc.side}, orders)
			ids := make([]string, len(got))
			for i := range got {
				ids[i] = got[i].OrderID
			}
			assert.Equalf(t, tc.exp, ids, "filterOrders should keep the %s %s orders", tc.orderType, tc.side)
		})
	}
	assert.Len(t, orders, 6, "filterOrders should leave the listed orders alone")
}

func TestKlineCandles(t *testing.T) {
	t.Parallel()
	_, err := e.klineCandles(t.Context(), asset.Futures, currency.NewBTCUSDT(), kline.OneHour, time.Now().Add(-time.Hour), time.Now(), 1)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "klineCandles should reject an unsupported asset")
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionOrders must reject a nil request")
	arg := &futures.PositionsRequest{RespectOrderHistoryLimits: true}
	_, err = e.GetFuturesPositionOrders(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetFuturesPositionOrders must reject a request without pairs")
	arg.Pairs = currency.Pairs{currency.NewBTCUSDT()}
	// The cases share arg and err with the checks after them, so they run in order
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			arg.Asset = a
			_, err = e.GetFuturesPositionOrders(t.Context(), arg)
			require.ErrorIsf(t, err, asset.ErrNotSupported, "GetFuturesPositionOrders must reject %s", a)
		})
	}
	arg.Asset = asset.USDTMarginedFutures
	arg.RespectOrderHistoryLimits = false
	_, err = e.GetFuturesPositionOrders(t.Context(), arg)
	require.ErrorIs(t, err, futures.ErrOrderHistoryTooLarge, "GetFuturesPositionOrders must reject a start beyond the order history")
	arg.StartDate = time.Now().Add(-time.Hour)
	arg.EndDate = arg.StartDate.Add(-time.Minute)
	_, err = e.GetFuturesPositionOrders(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesPositionOrders must reject an end before the start")

	// The requests carry the current time, so the replies come from a test server checking the window's bounds
	start := time.Now().Add(-48 * time.Hour)
	ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var body string
		switch r.URL.Path {
		case "/fapi/v2/positionRisk":
			assert.Equal(t, "BTCUSDT", q.Get("symbol"), "GetFuturesPositionOrders should request the USDⓈ-M symbol's positions")
			body = `[{"symbol":"BTCUSDT","marginType":"isolated","leverage":"10","positionSide":"BOTH"}]`
		case "/fapi/v1/allOrders":
			assert.Equal(t, "BTCUSDT", q.Get("symbol"), "GetFuturesPositionOrders should request the USDⓈ-M symbol's orders")
			assert.Equal(t, "1000", q.Get("limit"), "GetFuturesPositionOrders should request the most USDⓈ-M orders a request returns")
			assert.Equal(t, strconv.FormatInt(start.UnixMilli(), 10), q.Get("startTime"), "GetFuturesPositionOrders should request orders from the start")
			body = `[{"clientOrderId":"gct-um-filled","executedQty":"0.002","orderId":22542184,"origQty":"0.002","price":"80000","side":"BUY","status":"FILLED","symbol":"BTCUSDT","timeInForce":"GTC","type":"LIMIT","updateTime":1744120500000,"avgPrice":"80000","cumQuote":"160","time":1744120000000}]`
		case "/dapi/v1/positionRisk":
			assert.Equal(t, "BTCUSD", q.Get("pair"), "GetFuturesPositionOrders should request the COIN-M contract pair's positions")
			body = `[{"symbol":"BTCUSD_261225","marginType":"cross","leverage":"20","positionSide":"BOTH"},{"symbol":"BTCUSD_PERP","marginType":"isolated","leverage":"8","positionSide":"BOTH"}]`
		case "/dapi/v1/allOrders":
			assert.Equal(t, "BTCUSD_PERP", q.Get("symbol"), "GetFuturesPositionOrders should request the COIN-M symbol's orders")
			assert.Equal(t, "100", q.Get("limit"), "GetFuturesPositionOrders should request the most COIN-M orders a request returns")
			body = `[{"avgPrice":"80000","clientOrderId":"gct-cm-filled","cumBase":"0.00125","executedQty":"1","orderId":1573346959,"origQty":"1","price":"80000","side":"BUY","status":"FILLED","symbol":"BTCUSD_PERP","time":1744120400000,"timeInForce":"GTC","type":"LIMIT","updateTime":1744120500000},{"orderId":1573346964,"side":"BUY","status":"NEW","symbol":"BTCUSD_PERP","type":"ICEBERG","time":1744120600000}]`
		default:
			assert.Failf(t, "GetFuturesPositionOrders should only request positions and orders", "requested %s", r.URL.Path)
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	})
	coinmPair := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	// The cases share ex in order, so each also checks that a request keeps nothing from the last
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  []futures.PositionResponse
	}{
		{
			a:    asset.USDTMarginedFutures,
			pair: currency.NewBTCUSDT(),
			exp: []futures.PositionResponse{
				{
					Pair:                   currency.NewBTCUSDT(),
					Asset:                  asset.USDTMarginedFutures,
					ContractSettlementType: futures.Linear,
					Orders: []order.Detail{
						{
							TimeInForce:          order.GoodTillCancel,
							Leverage:             10,
							Price:                80000,
							Amount:               0.002,
							AverageExecutedPrice: 80000,
							ExecutedAmount:       0.002,
							Cost:                 160,
							Exchange:             ex.Name,
							OrderID:              "22542184",
							ClientOrderID:        "gct-um-filled",
							Type:                 order.Limit,
							Side:                 order.Buy,
							Status:               order.Filled,
							AssetType:            asset.USDTMarginedFutures,
							Date:                 time.Unix(1744120000, 0),
							LastUpdated:          time.Unix(1744120500, 0),
							Pair:                 currency.NewBTCUSDT(),
							MarginType:           margin.Isolated,
						},
					},
				},
			},
		},
		{
			a:    asset.CoinMarginedFutures,
			pair: coinmPair,
			exp: []futures.PositionResponse{
				{
					Pair:                   coinmPair,
					Asset:                  asset.CoinMarginedFutures,
					ContractSettlementType: futures.Inverse,
					Orders: []order.Detail{
						{
							TimeInForce:          order.GoodTillCancel,
							Leverage:             8,
							Price:                80000,
							Amount:               1,
							AverageExecutedPrice: 80000,
							ExecutedAmount:       1,
							Cost:                 0.00125,
							Exchange:             ex.Name,
							OrderID:              "1573346959",
							ClientOrderID:        "gct-cm-filled",
							Type:                 order.Limit,
							Side:                 order.Buy,
							Status:               order.Filled,
							AssetType:            asset.CoinMarginedFutures,
							Date:                 time.Unix(1744120400, 0),
							LastUpdated:          time.Unix(1744120500, 0),
							Pair:                 coinmPair,
							MarginType:           margin.Isolated,
						},
						{
							// The order's unknown type is left unset and logged rather than failing the listing
							Leverage:   8,
							Exchange:   ex.Name,
							OrderID:    "1573346964",
							Side:       order.Buy,
							Status:     order.New,
							AssetType:  asset.CoinMarginedFutures,
							Date:       time.Unix(1744120600, 0),
							Pair:       coinmPair,
							MarginType: margin.Isolated,
						},
					},
				},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: tc.a, Pairs: currency.Pairs{tc.pair}, StartDate: start, RespectOrderHistoryLimits: true})
			require.NoErrorf(t, err, "GetFuturesPositionOrders must not error for %s", tc.a)
			assert.Equalf(t, tc.exp, resp, "GetFuturesPositionOrders should return the %s position's orders", tc.a)
		})
	}

	t.Run("clamped window", func(t *testing.T) {
		t.Parallel()
		// RespectOrderHistoryLimits clamps an older start to exactly seven days ago, a window split at the longest the
		// endpoints take into one ending a millisecond before now and one of that last millisecond
		ex := newTradingTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte(`[]`))
			assert.NoError(t, err, "Write should not error")
		})
		// The cases share ex in order, so each also checks that a request keeps nothing from the last
		for _, tc := range []struct {
			a    asset.Item
			pair currency.Pair
		}{
			{asset.USDTMarginedFutures, currency.NewBTCUSDT()},
			{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)},
		} {
			t.Run(tc.a.String(), func(t *testing.T) {
				_, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: tc.a, Pairs: currency.Pairs{tc.pair}, StartDate: time.Now().AddDate(-1, 0, 0), RespectOrderHistoryLimits: true})
				assert.NoErrorf(t, err, "GetFuturesPositionOrders should collect the orders of a clamped %s window", tc.a)
			})
		}
	})
}

func TestSetMarginType(t *testing.T) { //nolint:tparallel // the rejections, then the account's margin type changes, run in order
	t.Parallel()
	// The rejections run first, in order, so the SetMarginType calls below also check they leave nothing behind
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			err := e.SetMarginType(t.Context(), a, currency.NewBTCUSDT(), margin.Isolated)
			require.ErrorIsf(t, err, asset.ErrNotSupported, "SetMarginType must reject %s", a)
		})
	}
	err := e.SetMarginType(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.SpotIsolated)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType, "SetMarginType must reject an unsupported margin type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	// Each case changes the account's margin type, two of them the same symbol's, so they run in order
	for _, tc := range []struct {
		a  asset.Item
		mt margin.Type
	}{
		{asset.USDTMarginedFutures, margin.Isolated},
		{asset.USDTMarginedFutures, margin.Multi},
		{asset.CoinMarginedFutures, margin.Isolated},
	} {
		t.Run(tc.a.String()+" "+subtestName(tc.mt.String()), func(t *testing.T) {
			assert.NoErrorf(t, e.SetMarginType(t.Context(), tc.a, assetToTradablePairMap[tc.a], tc.mt), "SetMarginType should not error for %s %s", tc.a, tc.mt)
		})
	}
}

func TestGetLeverage(t *testing.T) { //nolint:tparallel // the cases share e, so they run in order while the test runs in parallel
	t.Parallel()
	_, err := e.GetLeverage(t.Context(), asset.USDTMarginedFutures, currency.EMPTYPAIR, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLeverage must reject an empty pair")
	// The rejections run first, in order, so the GetLeverage calls below also check they leave nothing behind
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			_, err = e.GetLeverage(t.Context(), a, currency.NewBTCUSDT(), margin.Multi, order.UnknownSide)
			require.ErrorIsf(t, err, asset.ErrNotSupported, "GetLeverage must reject %s, whose leverage Binance does not report", a)
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	// The cases share e in order, so each also checks that a request keeps nothing from the last
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  float64
		err  error
	}{
		{a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(), exp: 10},
		{a: asset.USDTMarginedFutures, pair: currency.NewPair(currency.ETH, currency.USDT), err: futures.ErrPositionNotFound},
		{a: asset.CoinMarginedFutures, pair: currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), exp: 8},
		{a: asset.CoinMarginedFutures, pair: currency.NewPairWithDelimiter("ETHUSD", "PERP", currency.UnderscoreDelimiter), err: futures.ErrPositionNotFound},
	} {
		t.Run(tc.a.String()+" "+tc.pair.String(), func(t *testing.T) {
			if !mockTests && tc.err != nil {
				return
			}
			if !mockTests {
				tc.pair = assetToTradablePairMap[tc.a]
			}
			leverage, err := e.GetLeverage(t.Context(), tc.a, tc.pair, margin.Unset, order.UnknownSide)
			require.ErrorIsf(t, err, tc.err, "GetLeverage must return the expected error for %s %s", tc.a, tc.pair)
			if mockTests {
				assert.Equalf(t, tc.exp, leverage, "GetLeverage should return the %s %s leverage", tc.a, tc.pair)
			}
		})
	}
}

func TestSetLeverage(t *testing.T) { //nolint:tparallel // the rejections, then the account's leverage changes, run in order
	t.Parallel()
	// The rejections run first, in order, so the SetLeverage calls below also check they leave nothing behind
	for _, a := range []asset.Item{asset.Spot, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			err := e.SetLeverage(t.Context(), a, currency.NewBTCUSDT(), margin.Multi, 5, order.UnknownSide)
			require.ErrorIsf(t, err, asset.ErrNotSupported, "SetLeverage must reject %s", a)
		})
	}
	err := e.SetLeverage(t.Context(), asset.Margin, currency.NewBTCUSDT(), margin.Isolated, 5, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported, "SetLeverage must reject isolated margin")
	// The rejections run in order, before the SetLeverage calls below, which also check they leave nothing behind
	for _, amount := range []float64{0, 2.5} {
		t.Run(strconv.FormatFloat(amount, 'f', -1, 64), func(t *testing.T) {
			err = e.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, amount, order.UnknownSide)
			require.ErrorIsf(t, err, errInvalidLeverage, "SetLeverage must reject a leverage of %v", amount)
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	// Each case changes the account's leverage, two of them the same margin maximum, so they run in order
	for _, tc := range []struct {
		a        asset.Item
		mt       margin.Type
		leverage float64
	}{
		{asset.USDTMarginedFutures, margin.Isolated, 2},
		{asset.CoinMarginedFutures, margin.Multi, 21},
		{asset.Margin, margin.Multi, 5},
		{asset.Margin, margin.Unset, 5},
	} {
		t.Run(tc.a.String()+" "+subtestName(tc.mt.String()), func(t *testing.T) {
			assert.NoErrorf(t, e.SetLeverage(t.Context(), tc.a, assetToTradablePairMap[tc.a], tc.mt, tc.leverage, order.UnknownSide), "SetLeverage should not error for %s %s", tc.a, tc.mt)
		})
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must reject a spot asset")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must reject an unsupported futures asset")

	futuresFormat := currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}
	for _, tc := range []struct {
		a   asset.Item
		exp []futures.Contract
	}{
		{
			a: asset.USDTMarginedFutures,
			exp: []futures.Contract{
				{
					Exchange:           e.Name,
					Name:               currency.NewBTCUSDT().Format(futuresFormat),
					Underlying:         currency.NewBTCUSDT(),
					Asset:              asset.USDTMarginedFutures,
					StartDate:          time.UnixMilli(1567965300000),
					IsActive:           true,
					Status:             "TRADING",
					Type:               futures.Perpetual,
					SettlementType:     futures.Linear,
					SettlementCurrency: currency.USDT,
					MarginCurrency:     currency.USDT,
				},
				{
					Exchange:           e.Name,
					Name:               currency.NewPair(currency.NewCode("BTCUSDT"), currency.NewCode("261225")).Format(futuresFormat),
					Underlying:         currency.NewBTCUSDT(),
					Asset:              asset.USDTMarginedFutures,
					StartDate:          time.UnixMilli(1782460800000),
					EndDate:            time.UnixMilli(1798185600000),
					IsActive:           true,
					Status:             "TRADING",
					Type:               futures.Quarterly,
					SettlementType:     futures.Linear,
					SettlementCurrency: currency.USDT,
					MarginCurrency:     currency.USDT,
				},
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: []futures.Contract{
				{
					Exchange:           e.Name,
					Name:               currency.NewPair(currency.NewCode("BTCUSD"), currency.PERP).Format(futuresFormat),
					Underlying:         currency.NewBTCUSD(),
					Asset:              asset.CoinMarginedFutures,
					StartDate:          time.UnixMilli(1597042800000),
					IsActive:           true,
					Status:             "TRADING",
					Type:               futures.Perpetual,
					SettlementType:     futures.Inverse,
					SettlementCurrency: currency.BTC,
					MarginCurrency:     currency.BTC,
					Multiplier:         100,
					FundingRateFloor:   decimal.MustFromFloat(-0.025),
					FundingRateCeiling: decimal.MustFromFloat(0.025),
				},
				{
					Exchange:           e.Name,
					Name:               currency.NewPair(currency.NewCode("BTCUSD"), currency.NewCode("261225")).Format(futuresFormat),
					Underlying:         currency.NewBTCUSD(),
					Asset:              asset.CoinMarginedFutures,
					StartDate:          time.UnixMilli(1782460800000),
					EndDate:            time.UnixMilli(1798185600000),
					IsActive:           true,
					Status:             "TRADING",
					Type:               futures.Quarterly,
					SettlementType:     futures.Inverse,
					SettlementCurrency: currency.BTC,
					MarginCurrency:     currency.BTC,
					Multiplier:         100,
				},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesContractDetails(t.Context(), tc.a)
			require.NoError(t, err, "GetFuturesContractDetails must not error")
			if !mockTests {
				assert.NotEmpty(t, result, "GetFuturesContractDetails should return contracts")
				return
			}
			assert.Equal(t, tc.exp, result, "GetFuturesContractDetails should return every contract")
		})
	}
}

func TestFuturesContractType(t *testing.T) {
	t.Parallel()
	for contractType, exp := range map[string]futures.ContractType{
		"PERPETUAL":            futures.Perpetual,
		"TRADIFI_PERPETUAL":    futures.Perpetual,
		"PERPETUAL_DELIVERING": futures.Perpetual,
		"CURRENT_MONTH":        futures.Monthly,
		"NEXT_MONTH":           futures.Monthly,
		"CURRENT_QUARTER":      futures.Quarterly,
		"NEXT_QUARTER":         futures.Quarterly,
		"UNDOCUMENTED":         futures.Unknown,
	} {
		t.Run(subtestName(contractType), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, e.futuresContractType(contractType, "BTCUSDT"), "futuresContractType should convert %s", contractType)
		})
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context())
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetOpenInterest must reject a request without pairs")
	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest must reject an unsupported asset")

	pairs := marketTestPairs(t)
	result, err := e.GetOpenInterest(
		t.Context(),
		key.PairAsset{Base: pairs[asset.USDTMarginedFutures].Base.Item, Quote: pairs[asset.USDTMarginedFutures].Quote.Item, Asset: asset.USDTMarginedFutures},
		key.PairAsset{Base: pairs[asset.CoinMarginedFutures].Base.Item, Quote: pairs[asset.CoinMarginedFutures].Quote.Item, Asset: asset.CoinMarginedFutures},
	)
	require.NoError(t, err, "GetOpenInterest must not error")
	if !mockTests {
		assert.Len(t, result, 2, "GetOpenInterest should return the open interest of each pair")
		return
	}
	exp := []futures.OpenInterest{
		{Key: key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, pairs[asset.USDTMarginedFutures]), OpenInterest: 94598.71},
		// COIN-M open interest counts contracts
		{Key: key.NewExchangeAssetPair(e.Name, asset.CoinMarginedFutures, pairs[asset.CoinMarginedFutures]), OpenInterest: 12669867},
	}
	assert.Equal(t, exp, result, "GetOpenInterest should return the open interest of each pair")
}

func TestGetCurrencyTradeURL(t *testing.T) { //nolint:tparallel // the cases share e, so they run in order while the test runs in parallel
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL must reject an empty pair")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Futures, currency.NewBTCUSDT())
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must reject an unsupported asset")

	if !mockTests {
		for a, pair := range marketTestPairs(t) {
			t.Run(a.String(), func(t *testing.T) {
				result, err := e.GetCurrencyTradeURL(t.Context(), a, pair)
				require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
				assert.NotEmptyf(t, result, "GetCurrencyTradeURL should return a URL for %s", a)
			})
		}
		return
	}
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.USDTMarginedFutures, currency.NewPairWithDelimiter("ETHUSDT", "261225", currency.UnderscoreDelimiter))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "GetCurrencyTradeURL must reject a contract that is not listed")
	// The cases share e in order, so each also checks that a request keeps nothing from the last
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  string
	}{
		{asset.Spot, currency.NewBTCUSDT(), "https://www.binance.com/en/trade/BTCUSDT?type=spot"},
		{asset.Margin, currency.NewBTCUSDT(), "https://www.binance.com/en/trade/BTCUSDT?type=cross"},
		{asset.USDTMarginedFutures, currency.NewBTCUSDT(), "https://www.binance.com/en/futures/BTCUSDT"},
		{asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.USDC), "https://www.binance.com/en/futures/BTCUSDC"},
		{asset.USDTMarginedFutures, currency.NewPairWithDelimiter("BTCUSDT", "261225", currency.UnderscoreDelimiter), "https://www.binance.com/en/futures/BTCUSDT_QUARTER"},
		{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), "https://www.binance.com/en/delivery/BTCUSD"},
		{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTCUSD", "261225", currency.UnderscoreDelimiter), "https://www.binance.com/en/delivery/BTCUSD_QUARTER"},
		{asset.Options, currency.NewPairWithDelimiter("BTC", "261225-85000-C", currency.DashDelimiter), "https://www.binance.com/en/eoptions/BTCUSDT/BTC-261225-85000-C"},
	} {
		t.Run(tc.a.String()+" "+tc.pair.String(), func(t *testing.T) {
			result, err := e.GetCurrencyTradeURL(t.Context(), tc.a, tc.pair)
			require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s %s", tc.a, tc.pair)
			assert.Equalf(t, tc.exp, result, "GetCurrencyTradeURL should return the trade page of %s %s", tc.a, tc.pair)
		})
	}
}

// newTradingTestExchange returns an exchange named after the test whose REST requests handler answers, for tests whose
// requests carry the current time or need replies the shared fixtures cannot give
func newTradingTestExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret"})
	server := httptest.NewTestServer(t, capRequests(t, handler))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	for endpoint := range ex.API.Endpoints.GetURLMap() {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint, server.URL), "SetRunningURL must not error")
	}
	return ex
}

func TestParseOrderID(t *testing.T) {
	t.Parallel()
	id, err := parseOrderID("")
	require.NoError(t, err, "parseOrderID must not error for an empty ID")
	assert.Zero(t, id, "parseOrderID should return 0 for an empty ID")
	id, err = parseOrderID("4611875134427365377")
	require.NoError(t, err, "parseOrderID must not error")
	assert.Equal(t, uint64(4611875134427365377), id, "parseOrderID should return the ID")
	_, err = parseOrderID("abc")
	assert.ErrorIs(t, err, errInvalidOrderID, "parseOrderID should reject a non-numeric ID")
}

func TestAlgoOrderID(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "algo-3000000123", algoOrderID("3000000123"), "algoOrderID should tag the algo ID")
	assert.Equal(t, "algo-3000000123", formatAlgoOrderID(3000000123), "formatAlgoOrderID should tag the algo ID")
	for _, tc := range []struct {
		orderID, algoID string
		isAlgo          bool
	}{
		{"algo-3000000123", "3000000123", true},
		{"algo-", "", true},
		{"3000000123", "3000000123", false},
		{"Algo-3000000123", "Algo-3000000123", false},
		{"", "", false},
	} {
		t.Run(subtestName(tc.orderID), func(t *testing.T) {
			t.Parallel()
			algoID, isAlgo := cutAlgoOrderID(tc.orderID)
			assert.Equalf(t, tc.algoID, algoID, "cutAlgoOrderID should return the algo ID of %q", tc.orderID)
			assert.Equalf(t, tc.isAlgo, isAlgo, "cutAlgoOrderID should report whether %q is an algo order's", tc.orderID)
		})
	}
}

func TestParseFuturesOrderID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderID string
		id      uint64
		isAlgo  bool
		err     error
	}{
		{orderID: "", id: 0},
		{orderID: "3000000123", id: 3000000123},
		{orderID: "algo-3000000123", id: 3000000123, isAlgo: true},
		{orderID: "algo-", isAlgo: true, err: errInvalidOrderID},
		{orderID: "algo-abc", isAlgo: true, err: errInvalidOrderID},
		{orderID: "abc", err: errInvalidOrderID},
	} {
		t.Run(subtestName(tc.orderID), func(t *testing.T) {
			t.Parallel()
			id, isAlgo, err := parseFuturesOrderID(tc.orderID)
			require.ErrorIsf(t, err, tc.err, "parseFuturesOrderID must return the expected error for %q", tc.orderID)
			assert.Equalf(t, tc.id, id, "parseFuturesOrderID should return the ID of %q", tc.orderID)
			assert.Equalf(t, tc.isAlgo, isAlgo, "parseFuturesOrderID should report whether %q is an algo order's", tc.orderID)
		})
	}
	_, _, err := parseFuturesOrderID("algo-abc")
	assert.ErrorContains(t, err, `"algo-abc"`, "parseFuturesOrderID should quote the whole ID")
}

func TestRemainingAmountAndAveragePrice(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.006, remainingAmount(0.01, 0.004), "remainingAmount should subtract exactly")
	assert.Equal(t, 79990.0, averagePrice(39.995, 0.0005), "averagePrice should divide exactly")
	assert.Zero(t, averagePrice(39.995, 0), "averagePrice should be zero without fills")
}

func TestOrderSideString(t *testing.T) {
	t.Parallel()
	for side, exp := range map[order.Side]Side{
		order.Buy: "BUY", order.Bid: "BUY", order.Long: "BUY", order.Sell: "SELL", order.Ask: "SELL", order.Short: "SELL",
	} {
		t.Run(side.String(), func(t *testing.T) {
			t.Parallel()
			got, err := orderSideString(side)
			require.NoErrorf(t, err, "orderSideString must not error for %s", side)
			assert.Equalf(t, exp, got, "orderSideString should convert %s", side)
		})
	}
	_, err := orderSideString(order.UnknownSide)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "orderSideString should reject an unknown side")
}

func TestSpotOrderTypeString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
		exp       OrderType
	}{
		{order.Limit, order.GoodTillCancel, "LIMIT"},
		{order.Limit, order.PostOnly, "LIMIT_MAKER"},
		{order.Limit, order.PostOnly | order.GoodTillCancel, "LIMIT_MAKER"},
		{order.LimitMaker, order.UnknownTIF, "LIMIT_MAKER"},
		{order.Market, order.UnknownTIF, "MARKET"},
		{order.StopMarket, order.UnknownTIF, "STOP_LOSS"},
		{order.StopLimit, order.UnknownTIF, "STOP_LOSS_LIMIT"},
		{order.TakeProfitMarket, order.UnknownTIF, "TAKE_PROFIT"},
		{order.TakeProfitLimit, order.UnknownTIF, "TAKE_PROFIT_LIMIT"},
	} {
		t.Run(tc.orderType.String()+" "+subtestName(tc.tif.String()), func(t *testing.T) {
			t.Parallel()
			got, err := spotOrderTypeString(tc.orderType, tc.tif)
			require.NoErrorf(t, err, "spotOrderTypeString must not error for %s %s", tc.orderType, tc.tif)
			assert.Equalf(t, tc.exp, got, "spotOrderTypeString should convert %s %s", tc.orderType, tc.tif)
		})
	}
	for _, orderType := range []order.Type{order.TrailingStop, order.Stop, order.TakeProfit} {
		t.Run(orderType.String(), func(t *testing.T) {
			t.Parallel()
			_, err := spotOrderTypeString(orderType, order.UnknownTIF)
			assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "spotOrderTypeString should reject %s, which has no single spot type", orderType)
		})
	}
}

func TestStringToSpotOrderType(t *testing.T) {
	t.Parallel()
	for in, exp := range map[OrderType]order.Type{
		"LIMIT": order.Limit, "MARKET": order.Market, "STOP_LOSS": order.StopMarket, "STOP_LOSS_LIMIT": order.StopLimit,
		"TAKE_PROFIT": order.TakeProfitMarket, "TAKE_PROFIT_LIMIT": order.TakeProfitLimit, "LIMIT_MAKER": order.LimitMaker,
	} {
		t.Run(subtestName(string(in)), func(t *testing.T) {
			t.Parallel()
			got, err := stringToSpotOrderType(in)
			require.NoErrorf(t, err, "stringToSpotOrderType must not error for %s", in)
			assert.Equalf(t, exp, got, "stringToSpotOrderType should convert %s", in)
			if exp != order.LimitMaker {
				back, err := spotOrderTypeString(got, order.UnknownTIF)
				require.NoErrorf(t, err, "spotOrderTypeString must not error for %s", got)
				assert.Equalf(t, in, back, "spotOrderTypeString should invert stringToSpotOrderType for %s", in)
			}
		})
	}
	_, err := stringToSpotOrderType("STOP_MARKET")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "stringToSpotOrderType should reject a futures type")
}

func TestFuturesOrderTypeString(t *testing.T) {
	t.Parallel()
	for in, exp := range map[order.Type]OrderType{
		order.Limit: "LIMIT", order.Market: "MARKET", order.StopLimit: "STOP", order.StopMarket: "STOP_MARKET",
		order.TakeProfitLimit: "TAKE_PROFIT", order.TakeProfitMarket: "TAKE_PROFIT_MARKET", order.TrailingStop: "TRAILING_STOP_MARKET",
	} {
		t.Run(in.String(), func(t *testing.T) {
			t.Parallel()
			got, err := futuresOrderTypeString(in)
			require.NoErrorf(t, err, "futuresOrderTypeString must not error for %s", in)
			assert.Equalf(t, exp, got, "futuresOrderTypeString should convert %s", in)
			back, err := derivativesOrderType(got)
			require.NoErrorf(t, err, "derivativesOrderType must not error for %s", got)
			assert.Equalf(t, in, back, "derivativesOrderType should invert futuresOrderTypeString for %s", in)
		})
	}
	for _, orderType := range []order.Type{order.LimitMaker, order.Stop, order.TakeProfit} {
		t.Run(orderType.String(), func(t *testing.T) {
			t.Parallel()
			_, err := futuresOrderTypeString(orderType)
			assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "futuresOrderTypeString should reject %s, which has no single futures type", orderType)
		})
	}
}

func TestResolveTriggerOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		price     float64
		exp       order.Type
	}{
		{order.Stop, 0, order.StopMarket},
		{order.Stop, 70000, order.StopLimit},
		{order.TakeProfit, 0, order.TakeProfitMarket},
		{order.TakeProfit, 70000, order.TakeProfitLimit},
		{order.StopMarket, 70000, order.StopMarket},
		{order.Limit, 70000, order.Limit},
	} {
		t.Run(tc.orderType.String()+" "+strconv.FormatFloat(tc.price, 'f', -1, 64), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, resolveTriggerOrderType(tc.orderType, tc.price), "resolveTriggerOrderType should resolve %s with price %v", tc.orderType, tc.price)
		})
	}
}

func TestSubmittedOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a         asset.Item
		orderType order.Type
		tif       order.TimeInForce
		price     float64
		exp       order.Type
	}{
		{asset.Spot, order.Limit, order.PostOnly, 70000, order.LimitMaker},
		{asset.Margin, order.Limit, order.PostOnly | order.GoodTillCancel, 70000, order.LimitMaker},
		{asset.Spot, order.Limit, order.GoodTillCancel, 70000, order.Limit},
		{asset.Spot, order.LimitMaker, order.UnknownTIF, 70000, order.LimitMaker},
		{asset.Spot, order.Stop, order.PostOnly, 70000, order.StopLimit},
		{asset.USDTMarginedFutures, order.Limit, order.PostOnly, 70000, order.Limit},
		{asset.CoinMarginedFutures, order.Limit, order.PostOnly, 70000, order.Limit},
		{asset.Options, order.Limit, order.PostOnly, 100, order.Limit},
		{asset.USDTMarginedFutures, order.TakeProfit, order.UnknownTIF, 0, order.TakeProfitMarket},
	} {
		t.Run(tc.a.String()+" "+tc.orderType.String()+" "+subtestName(tc.tif.String()), func(t *testing.T) {
			t.Parallel()
			s := &order.Submit{AssetType: tc.a, Type: tc.orderType, TimeInForce: tc.tif, Price: tc.price}
			assert.Equalf(t, tc.exp, submittedOrderType(s), "submittedOrderType should place a %s %s %s order as its type", tc.a, tc.tif, tc.orderType)
		})
	}
}

func TestPlacesPostOnly(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a         asset.Item
		orderType order.Type
		tif       order.TimeInForce
		exp       bool
	}{
		{asset.Spot, order.LimitMaker, order.UnknownTIF, true},
		{asset.Margin, order.LimitMaker, order.GoodTillCancel, true},
		{asset.Spot, order.Limit, order.GoodTillCancel, false},
		{asset.Spot, order.OCO, order.PostOnly, false},
		{asset.USDTMarginedFutures, order.Limit, order.PostOnly, true},
		{asset.USDTMarginedFutures, order.StopLimit, order.GoodTillCrossing, true},
		{asset.CoinMarginedFutures, order.TakeProfitLimit, order.PostOnly | order.GoodTillCancel, true},
		{asset.CoinMarginedFutures, order.Limit, order.PostOnly | order.GoodTillCrossing, true},
		{asset.USDTMarginedFutures, order.Limit, order.ImmediateOrCancel, false},
		{asset.USDTMarginedFutures, order.Limit, order.GoodTillDay, false},
		{asset.USDTMarginedFutures, order.StopMarket, order.PostOnly, false},
		{asset.USDTMarginedFutures, order.TrailingStop, order.PostOnly, false},
		{asset.Options, order.Limit, order.GoodTillCrossing, true},
		{asset.Options, order.Limit, order.PostOnly | order.GoodTillCancel, true},
		{asset.Options, order.Limit, order.GoodTillCancel, false},
	} {
		t.Run(tc.a.String()+" "+tc.orderType.String()+" "+subtestName(tc.tif.String()), func(t *testing.T) {
			t.Parallel()
			s := &order.Submit{AssetType: tc.a, Type: tc.orderType, TimeInForce: tc.tif}
			assert.Equalf(t, tc.exp, placesPostOnly(s), "placesPostOnly should report whether a %s %s %s order is placed post only", tc.a, tc.tif, tc.orderType)
		})
	}
}

func TestIsConditionalOrderType(t *testing.T) {
	t.Parallel()
	for _, orderType := range []order.Type{order.Stop, order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitLimit, order.TakeProfitMarket, order.TrailingStop} {
		t.Run(orderType.String(), func(t *testing.T) {
			t.Parallel()
			assert.Truef(t, isConditionalOrderType(orderType), "isConditionalOrderType should report %s", orderType)
		})
	}
	for _, orderType := range []order.Type{order.Limit, order.Market, order.LimitMaker, order.OCO, order.AnyType} {
		t.Run(orderType.String(), func(t *testing.T) {
			t.Parallel()
			assert.Falsef(t, isConditionalOrderType(orderType), "isConditionalOrderType should not report %s", orderType)
		})
	}
}

// conditionalOrderTypes are the stop, take profit and trailing stop types, which USDⓈ-M takes as algo orders alone
var conditionalOrderTypes = []order.Type{order.Stop, order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitLimit, order.TakeProfitMarket, order.TrailingStop}

func TestIsAlgoOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		a     asset.Item
		types []order.Type
		exp   bool
	}{
		{name: "USDⓈ-M stop types", a: asset.USDTMarginedFutures, types: conditionalOrderTypes, exp: true},
		{name: "COIN-M stop types", a: asset.CoinMarginedFutures, types: conditionalOrderTypes},
		{name: "USDⓈ-M order types", a: asset.USDTMarginedFutures, types: []order.Type{order.Limit, order.Market}},
		{name: "USDⓈ-M untyped", a: asset.USDTMarginedFutures, types: []order.Type{order.UnknownType, order.AnyType}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, orderType := range tc.types {
				t.Run(orderType.String(), func(t *testing.T) {
					t.Parallel()
					assert.Equalf(t, tc.exp, isAlgoOrderType(tc.a, orderType), "isAlgoOrderType should report whether a %s %s type names an algo order", tc.a, orderType)
				})
			}
		})
	}
}

func TestNamesOrderOrAlgoOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		a     asset.Item
		types []order.Type
		exp   bool
	}{
		{name: "COIN-M stop types", a: asset.CoinMarginedFutures, types: conditionalOrderTypes, exp: true},
		{name: "COIN-M untyped", a: asset.CoinMarginedFutures, types: []order.Type{order.UnknownType, order.AnyType}, exp: true},
		{name: "USDⓈ-M untyped", a: asset.USDTMarginedFutures, types: []order.Type{order.UnknownType, order.AnyType}, exp: true},
		{name: "USDⓈ-M stop types", a: asset.USDTMarginedFutures, types: conditionalOrderTypes},
		{name: "COIN-M order types", a: asset.CoinMarginedFutures, types: []order.Type{order.Limit, order.Market}},
		{name: "USDⓈ-M order types", a: asset.USDTMarginedFutures, types: []order.Type{order.Limit, order.Market}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, orderType := range tc.types {
				t.Run(orderType.String(), func(t *testing.T) {
					t.Parallel()
					assert.Equalf(t, tc.exp, namesOrderOrAlgoOrder(tc.a, orderType), "namesOrderOrAlgoOrder should report whether a %s %s type may name either kind", tc.a, orderType)
				})
			}
		})
	}
}

func TestTimeInForceString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a   asset.Item
		tif order.TimeInForce
		exp TimeInForce
		err error
	}{
		{asset.Spot, order.UnknownTIF, "GTC", nil},
		{asset.Spot, order.GoodTillCancel, "GTC", nil},
		{asset.Margin, order.ImmediateOrCancel, "IOC", nil},
		{asset.Options, order.FillOrKill, "FOK", nil},
		{asset.USDTMarginedFutures, order.PostOnly, "GTX", nil},
		{asset.USDTMarginedFutures, order.PostOnly | order.GoodTillCancel, "GTX", nil},
		{asset.CoinMarginedFutures, order.GoodTillCrossing, "GTX", nil},
		{asset.CoinMarginedFutures, order.PostOnly | order.GoodTillCrossing, "GTX", nil},
		{asset.USDTMarginedFutures, order.GoodTillDay, "GTD", nil},
		{asset.Spot, order.PostOnly, "", order.ErrUnsupportedTimeInForce},
		{asset.Options, order.GoodTillCrossing, "", order.ErrUnsupportedTimeInForce},
		{asset.CoinMarginedFutures, order.GoodTillDay, "", order.ErrUnsupportedTimeInForce},
		{asset.USDTMarginedFutures, order.GoodTillTime, "", order.ErrUnsupportedTimeInForce},
		{asset.USDTMarginedFutures, order.StopOrReduce, "", order.ErrUnsupportedTimeInForce},
	} {
		t.Run(tc.a.String()+" "+subtestName(tc.tif.String()), func(t *testing.T) {
			t.Parallel()
			got, err := timeInForceString(tc.a, tc.tif)
			require.ErrorIsf(t, err, tc.err, "timeInForceString must return the expected error for %s %s", tc.a, tc.tif)
			assert.Equalf(t, tc.exp, got, "timeInForceString should convert %s %s", tc.a, tc.tif)
		})
	}
}

func TestWorkingTypeString(t *testing.T) {
	t.Parallel()
	got, err := workingTypeString(order.LastPrice)
	require.NoError(t, err, "workingTypeString must not error for the last price")
	assert.Empty(t, got, "workingTypeString should leave the contract price, Binance's default, unsent")
	got, err = workingTypeString(order.MarkPrice)
	require.NoError(t, err, "workingTypeString must not error for the mark price")
	assert.Equal(t, WorkingTypeMarkPrice, got, "workingTypeString should convert the mark price")
	_, err = workingTypeString(order.IndexPrice)
	assert.ErrorIs(t, err, order.ErrUnknownPriceType, "workingTypeString should reject the index price")
}

func TestAlgoOrderStatus(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]order.Status{
		"NEW": order.New, "CANCELED": order.Cancelled, "TRIGGERING": order.Pending, "TRIGGERED": order.Active,
		"FINISHED": order.Closed, "REJECTED": order.Rejected, "EXPIRED": order.Expired,
	} {
		t.Run(subtestName(in), func(t *testing.T) {
			t.Parallel()
			got, err := algoOrderStatus(in)
			require.NoErrorf(t, err, "algoOrderStatus must not error for %s", in)
			assert.Equalf(t, exp, got, "algoOrderStatus should convert %s", in)
		})
	}
	_, err := algoOrderStatus("PAUSED")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "algoOrderStatus should reject an unknown status")
}

func TestAlgoOrderDetail(t *testing.T) {
	t.Parallel()
	t.Run("remaining amount", func(t *testing.T) {
		t.Parallel()
		// An algo order that never triggered, or that the matching engine refused, has its whole quantity remaining,
		// as no part of it can have executed, while a triggering or triggered one's remainder is left for the single
		// order query, the only conversion told what it executed
		ex := new(Exchange)
		ex.Name = "Binance"
		// The cases share ex in order, so each also checks that a conversion keeps nothing from the last
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			t.Run(a.String(), func(t *testing.T) {
				for status, exp := range map[string]float64{
					"NEW": 1, "CANCELED": 1, "REJECTED": 1, "EXPIRED": 1, "TRIGGERING": 0, "TRIGGERED": 0, "FINISHED": 0, "PAUSED": 0,
				} {
					t.Run(subtestName(status), func(t *testing.T) {
						d, err := ex.algoOrderDetail(a, &USDTMarginedFuturesAlgoOrderBase{
							AlgoID: 77, OrderType: OrderTypeStopMarket, Side: SideSell, TimeInForce: TimeInForceGoodTillCancel, Quantity: 1, AlgoStatus: status,
						}, 0, currency.NewBTCUSDT())
						if status == "PAUSED" {
							assert.ErrorIsf(t, err, errUnknownOrderStatus, "algoOrderDetail should report the unknown %s status", a)
						} else {
							require.NoErrorf(t, err, "algoOrderDetail must not error for a %s %s order", a, status)
						}
						assert.Zerof(t, d.ExecutedAmount, "algoOrderDetail should report no executed amount for a %s %s order", a, status)
						assert.Equalf(t, exp, d.RemainingAmount, "algoOrderDetail should report the remaining amount of a %s %s order", a, status)
					})
				}
			})
		}
	})
}

func TestOrderListStatus(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]order.Status{"EXECUTING": order.Active, "ALL_DONE": order.Closed, "REJECT": order.Rejected} {
		t.Run(subtestName(in), func(t *testing.T) {
			t.Parallel()
			got, err := orderListStatus(in)
			require.NoErrorf(t, err, "orderListStatus must not error for %s", in)
			assert.Equalf(t, exp, got, "orderListStatus should convert %s", in)
		})
	}
	_, err := orderListStatus("EXEC_STARTED")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "orderListStatus should reject a list status type")
}

func TestNewSpotOrderValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    *spotOrderValues
		err    error
	}{
		{
			name:   "limit",
			submit: &order.Submit{Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 0.001, TriggerPrice: 1},
			exp:    &spotOrderValues{side: "BUY", orderType: "LIMIT", timeInForce: "GTC", price: 80000, quantity: 0.001},
		},
		{
			name:   "post only limit",
			submit: &order.Submit{Type: order.Limit, Side: order.Sell, Price: 90000, Amount: 0.001, TimeInForce: order.PostOnly},
			exp:    &spotOrderValues{side: "SELL", orderType: "LIMIT_MAKER", price: 90000, quantity: 0.001},
		},
		{
			name:   "market quote amount",
			submit: &order.Submit{Type: order.Market, Side: order.Long, QuoteAmount: 100, Price: 1, TimeInForce: order.ImmediateOrCancel},
			exp:    &spotOrderValues{side: "BUY", orderType: "MARKET", quoteOrderQuantity: 100},
		},
		{
			name:   "stop loss",
			submit: &order.Submit{Type: order.Stop, Side: order.Sell, TriggerPrice: 71000, Amount: 0.001},
			exp:    &spotOrderValues{side: "SELL", orderType: "STOP_LOSS", stopPrice: 71000, quantity: 0.001},
		},
		{
			name:   "take profit limit",
			submit: &order.Submit{Type: order.TakeProfitLimit, Side: order.Sell, Price: 95000, TriggerPrice: 94000, Amount: 0.001, TimeInForce: order.FillOrKill},
			exp:    &spotOrderValues{side: "SELL", orderType: "TAKE_PROFIT_LIMIT", timeInForce: "FOK", price: 95000, stopPrice: 94000, quantity: 0.001},
		},
		{name: "unknown side", submit: &order.Submit{Type: order.Market, Amount: 1}, err: order.ErrSideIsInvalid},
		{name: "unsupported type", submit: &order.Submit{Type: order.TrailingStop, Side: order.Sell, Amount: 1}, err: order.ErrUnsupportedOrderType},
		{name: "unsupported time in force", submit: &order.Submit{Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.GoodTillCrossing}, err: order.ErrUnsupportedTimeInForce},
		{name: "stop limit without price", submit: &order.Submit{Type: order.StopLimit, Side: order.Sell, TriggerPrice: 1, Amount: 1}, err: order.ErrPriceMustBeSetIfLimitOrder},
		{name: "limit maker without price", submit: &order.Submit{Type: order.LimitMaker, Side: order.Sell, Amount: 1}, err: order.ErrPriceMustBeSetIfLimitOrder},
		{name: "stop without trigger price", submit: &order.Submit{Type: order.Stop, Side: order.Sell, Amount: 1}, err: errTriggerPriceRequired},
		{name: "limit quote amount", submit: &order.Submit{Type: order.Limit, Side: order.Buy, Price: 1, QuoteAmount: 100}, err: order.ErrAmountIsInvalid},
		{
			name:   "limit maker good till cancel",
			submit: &order.Submit{Type: order.LimitMaker, Side: order.Sell, Price: 90000, Amount: 0.001, TimeInForce: order.GoodTillCancel},
			exp:    &spotOrderValues{side: "SELL", orderType: "LIMIT_MAKER", price: 90000, quantity: 0.001},
		},
		{
			name:   "limit maker post only",
			submit: &order.Submit{Type: order.LimitMaker, Side: order.Sell, Price: 90000, Amount: 0.001, TimeInForce: order.PostOnly | order.GoodTillCancel},
			exp:    &spotOrderValues{side: "SELL", orderType: "LIMIT_MAKER", price: 90000, quantity: 0.001},
		},
		{
			name:   "stop loss good till cancel",
			submit: &order.Submit{Type: order.StopMarket, Side: order.Sell, TriggerPrice: 71000, Amount: 0.001, TimeInForce: order.GoodTillCancel},
			exp:    &spotOrderValues{side: "SELL", orderType: "STOP_LOSS", stopPrice: 71000, quantity: 0.001},
		},
		{
			name:   "take profit",
			submit: &order.Submit{Type: order.TakeProfit, Side: order.Sell, TriggerPrice: 95000, Amount: 0.001},
			exp:    &spotOrderValues{side: "SELL", orderType: "TAKE_PROFIT", stopPrice: 95000, quantity: 0.001},
		},
		{name: "limit maker immediate or cancel", submit: &order.Submit{Type: order.LimitMaker, Side: order.Sell, Price: 1, Amount: 1, TimeInForce: order.ImmediateOrCancel}, err: order.ErrUnsupportedTimeInForce},
		{name: "limit maker fill or kill", submit: &order.Submit{Type: order.LimitMaker, Side: order.Sell, Price: 1, Amount: 1, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce},
		{name: "limit maker good till day", submit: &order.Submit{Type: order.LimitMaker, Side: order.Sell, Price: 1, Amount: 1, TimeInForce: order.GoodTillDay}, err: order.ErrUnsupportedTimeInForce},
		{name: "limit maker good till crossing", submit: &order.Submit{Type: order.LimitMaker, Side: order.Sell, Price: 1, Amount: 1, TimeInForce: order.GoodTillCrossing}, err: order.ErrUnsupportedTimeInForce},
		{name: "post only limit with an unknown flag", submit: &order.Submit{Type: order.Limit, Side: order.Sell, Price: 1, Amount: 1, TimeInForce: order.PostOnly | 1<<15}, err: order.ErrUnsupportedTimeInForce},
		{name: "stop loss post only", submit: &order.Submit{Type: order.StopMarket, Side: order.Sell, TriggerPrice: 1, Amount: 1, TimeInForce: order.PostOnly}, err: order.ErrUnsupportedTimeInForce},
		{name: "stop loss good till day", submit: &order.Submit{Type: order.Stop, Side: order.Sell, TriggerPrice: 1, Amount: 1, TimeInForce: order.GoodTillDay}, err: order.ErrUnsupportedTimeInForce},
		{name: "stop loss with an unknown flag", submit: &order.Submit{Type: order.StopMarket, Side: order.Sell, TriggerPrice: 1, Amount: 1, TimeInForce: order.GoodTillCancel | 1<<15}, err: order.ErrUnsupportedTimeInForce},
		{name: "take profit immediate or cancel", submit: &order.Submit{Type: order.TakeProfitMarket, Side: order.Sell, TriggerPrice: 1, Amount: 1, TimeInForce: order.ImmediateOrCancel}, err: order.ErrUnsupportedTimeInForce},
		{name: "take profit fill or kill", submit: &order.Submit{Type: order.TakeProfit, Side: order.Sell, TriggerPrice: 1, Amount: 1, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce},
		{
			name:   "market good till cancel",
			submit: &order.Submit{Type: order.Market, Side: order.Sell, Amount: 0.001, TimeInForce: order.GoodTillCancel},
			exp:    &spotOrderValues{side: "SELL", orderType: "MARKET", quantity: 0.001},
		},
		{name: "market fill or kill", submit: &order.Submit{Type: order.Market, Side: order.Sell, Amount: 1, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce},
		{name: "market good till day", submit: &order.Submit{Type: order.Market, Side: order.Sell, Amount: 1, TimeInForce: order.GoodTillDay}, err: order.ErrUnsupportedTimeInForce},
		{name: "market good till crossing", submit: &order.Submit{Type: order.Market, Side: order.Sell, Amount: 1, TimeInForce: order.GoodTillCrossing}, err: order.ErrUnsupportedTimeInForce},
		{name: "market post only", submit: &order.Submit{Type: order.Market, Side: order.Sell, Amount: 1, TimeInForce: order.PostOnly}, err: order.ErrUnsupportedTimeInForce},
		{name: "market with an unknown flag", submit: &order.Submit{Type: order.Market, Side: order.Sell, Amount: 1, TimeInForce: order.ImmediateOrCancel | 1<<15}, err: order.ErrUnsupportedTimeInForce},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newSpotOrderValues(tc.submit)
			require.ErrorIs(t, err, tc.err, "newSpotOrderValues must return the expected error")
			assert.Equal(t, tc.exp, got, "newSpotOrderValues should work out the order's values")
		})
	}
}

func TestNewSpotOneCancelsTheOtherLegs(t *testing.T) {
	t.Parallel()
	_, err := newSpotOneCancelsTheOtherLegs(&order.Submit{Type: order.OCO, Price: 1, TriggerPrice: 1, Amount: 1})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "newSpotOneCancelsTheOtherLegs must reject an unknown side")
	_, err = newSpotOneCancelsTheOtherLegs(&order.Submit{Type: order.OCO, Side: order.Sell, TriggerPrice: 1, Amount: 1})
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder, "newSpotOneCancelsTheOtherLegs must reject a list without a price")
	_, err = newSpotOneCancelsTheOtherLegs(&order.Submit{Type: order.OCO, Side: order.Sell, Price: 1, Amount: 1})
	require.ErrorIs(t, err, errTriggerPriceRequired, "newSpotOneCancelsTheOtherLegs must reject a list without a trigger price")
	_, err = newSpotOneCancelsTheOtherLegs(&order.Submit{
		Type: order.OCO, Side: order.Sell, Price: 2, TriggerPrice: 1, Amount: 1, TimeInForce: order.GoodTillDay,
		RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{LimitPrice: 0.9}},
	})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "newSpotOneCancelsTheOtherLegs must reject an unsupported stop limit time in force")
	for _, tif := range []order.TimeInForce{order.ImmediateOrCancel, order.FillOrKill, order.GoodTillDay, order.GoodTillCrossing, order.PostOnly | 1<<15} {
		t.Run(tif.String(), func(t *testing.T) {
			t.Parallel()
			_, err := newSpotOneCancelsTheOtherLegs(&order.Submit{Type: order.OCO, Side: order.Sell, Price: 2, TriggerPrice: 1, Amount: 1, TimeInForce: tif})
			require.ErrorIsf(t, err, order.ErrUnsupportedTimeInForce, "newSpotOneCancelsTheOtherLegs must reject %s for a list without a stop limit order, whose orders take no time in force", tif)
		})
	}

	got, err := newSpotOneCancelsTheOtherLegs(&order.Submit{
		Type: order.OCO, Side: order.Sell, Price: 90000, TriggerPrice: 70000, Amount: 0.001, ClientOrderID: "gct-oco",
		TimeInForce:         order.PostOnly | order.GoodTillCancel,
		RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{LimitPrice: 69000}},
	})
	require.NoError(t, err, "newSpotOneCancelsTheOtherLegs must not error")
	exp := &spotOneCancelsTheOtherLegs{
		side:       "SELL",
		limitMaker: OneCancelsTheOtherOrderListLeg{Type: "LIMIT_MAKER", Price: 90000},
		stopLoss:   OneCancelsTheOtherOrderListLeg{Type: "STOP_LOSS_LIMIT", Price: 69000, StopPrice: 70000, TimeInForce: "GTC"},
	}
	assert.Equal(t, exp, got, "newSpotOneCancelsTheOtherLegs should work out a stop limit list's legs")

	got, err = newSpotOneCancelsTheOtherLegs(&order.Submit{Type: order.OCO, Side: order.Buy, Price: 60000, TriggerPrice: 95000, Amount: 0.002})
	require.NoError(t, err, "newSpotOneCancelsTheOtherLegs must not error")
	exp = &spotOneCancelsTheOtherLegs{
		side:       "BUY",
		limitMaker: OneCancelsTheOtherOrderListLeg{Type: "LIMIT_MAKER", Price: 60000},
		stopLoss:   OneCancelsTheOtherOrderListLeg{Type: "STOP_LOSS", StopPrice: 95000},
	}
	assert.Equal(t, exp, got, "newSpotOneCancelsTheOtherLegs should work out a stop loss list's legs without client order IDs")

	got, err = newSpotOneCancelsTheOtherLegs(&order.Submit{Type: order.OCO, Side: order.Buy, Price: 60000, TriggerPrice: 95000, Amount: 0.002, TimeInForce: order.PostOnly | order.GoodTillCancel})
	require.NoError(t, err, "newSpotOneCancelsTheOtherLegs must not error for a post only, good till cancel list without a stop limit order")
	assert.Equal(t, exp, got, "newSpotOneCancelsTheOtherLegs should take a post only, good till cancel list without a stop limit order as one without a time in force")
}

func TestMarginSideEffect(t *testing.T) {
	t.Parallel()
	assert.Empty(t, marginSideEffect(&order.Submit{}), "marginSideEffect should send no side effect by default")
	assert.Equal(t, SideEffectTypeMarginBuy, marginSideEffect(&order.Submit{AutoBorrow: true}), "marginSideEffect should borrow")
	assert.Equal(t, SideEffectTypeAutoRepay, marginSideEffect(&order.Submit{AutoRepay: true}), "marginSideEffect should repay")
	assert.Equal(t, SideEffectTypeAutoBorrowRepay, marginSideEffect(&order.Submit{AutoBorrow: true, AutoRepay: true}), "marginSideEffect should borrow and repay")
}

func TestFuturesPositionSide(t *testing.T) {
	t.Parallel()
	for _, hedgeMode := range []bool{false, true} {
		t.Run(fmt.Sprintf("hedge mode %t", hedgeMode), func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/fapi/v1/positionSide/dual", r.URL.Path, "futuresPositionSide should fetch the position mode")
				_, err := fmt.Fprintf(w, `{"dualSidePosition":%t}`, hedgeMode)
				assert.NoError(t, err, "Writing the response should not error")
			})
			// The cases share ex and its server in order, so each also checks that a call keeps nothing from the last
			for _, tc := range []struct {
				side           order.Side
				reduceOnly     bool
				hedgedSide     PositionSide
				reduceOnlySent bool
			}{
				{side: order.Buy, reduceOnly: true, reduceOnlySent: true},
				{side: order.Sell},
				{side: order.Long, hedgedSide: "LONG"},
				{side: order.Long, reduceOnly: true, hedgedSide: "SHORT", reduceOnlySent: true},
				{side: order.Short, hedgedSide: "SHORT"},
				{side: order.Short, reduceOnly: true, hedgedSide: "LONG", reduceOnlySent: true},
			} {
				t.Run(tc.side.String()+" reduce only "+strconv.FormatBool(tc.reduceOnly), func(t *testing.T) {
					positionSide, reduceOnly, err := ex.futuresPositionSide(t.Context(), &order.Submit{Side: tc.side, ReduceOnly: tc.reduceOnly})
					require.NoErrorf(t, err, "futuresPositionSide must not error for %s reduce only %t", tc.side, tc.reduceOnly)
					var expSide PositionSide
					expReduceOnly := tc.reduceOnlySent
					if hedgeMode && tc.hedgedSide != "" {
						expSide, expReduceOnly = tc.hedgedSide, false
					}
					assert.Equalf(t, expSide, positionSide, "futuresPositionSide should return the position side for %s reduce only %t", tc.side, tc.reduceOnly)
					assert.Equalf(t, expReduceOnly, reduceOnly, "futuresPositionSide should return the reduce only flag for %s reduce only %t", tc.side, tc.reduceOnly)
				})
			}
		})
	}
	t.Run("error", func(t *testing.T) {
		t.Parallel()
		ex := newTradingTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte(`{"code":-2015,"msg":"Invalid API-key, IP, or permissions for action."}`))
			assert.NoError(t, err, "Writing the response should not error")
		})
		_, _, err := ex.futuresPositionSide(t.Context(), &order.Submit{Side: order.Long})
		assert.ErrorIs(t, err, errAPIResponse, "futuresPositionSide should return the position mode error")
	})
}

// positionModeVenue is a fake USDⓈ-M and COIN-M venue holding each account's position mode, keyed by API key, which it
// reports and changes, and checks orders against the way Binance does
type positionModeVenue struct {
	mu      sync.Mutex
	hedge   map[string]bool
	lookups map[string]int
	sides   []string // the position side and reduce only flag of every order placed or rejected, in turn
	reject  string   // the reply every order gets when set
}

func (v *positionModeVenue) setHedgeMode(apiKey string, hedgeMode bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.hedge[apiKey] = hedgeMode
}

func (v *positionModeVenue) lookupCounts() map[string]int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return maps.Clone(v.lookups)
}

func (v *positionModeVenue) positionSides() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.sides)
}

func (v *positionModeVenue) serve(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		v.mu.Lock()
		defer v.mu.Unlock()
		apiKey, q := r.Header.Get("X-MBX-APIKEY"), r.URL.Query()
		reply := `{"code":200,"msg":"success"}`
		switch r.Method + " " + r.URL.Path {
		case "GET /fapi/v1/positionSide/dual":
			if v.lookups == nil {
				v.lookups = make(map[string]int)
			}
			v.lookups[apiKey]++
			reply = fmt.Sprintf(`{"dualSidePosition":%t}`, v.hedge[apiKey])
		case "POST /fapi/v1/positionSide/dual":
			v.hedge[apiKey] = q.Get("dualSidePosition") == "true"
		case "POST /fapi/v1/order", "POST /dapi/v1/order", "POST /fapi/v1/algoOrder", "POST /dapi/v1/algoOrder":
			v.sides = append(v.sides, q.Get("positionSide")+"/"+q.Get("reduceOnly"))
			switch {
			case v.reject != "":
				reply = v.reject
			case v.hedge[apiKey] && q.Get("reduceOnly") != "":
				reply = `{"code":-1106,"msg":"Parameter 'reduceonly' sent when not required."}`
			case v.hedge[apiKey] && q.Get("positionSide") == "":
				reply = `{"code":-4061,"msg":"Order's position side does not match user's setting."}`
			case !v.hedge[apiKey] && q.Get("positionSide") != "":
				reply = `{"code":-4061,"msg":"Order's position side does not match user's setting."}`
			case strings.HasSuffix(r.URL.Path, "algoOrder"):
				reply = `{"algoId":3000000123,"clientAlgoId":"gct-algo","algoType":"CONDITIONAL","orderType":"STOP_MARKET","algoStatus":"NEW","createTime":1744118600000,"updateTime":1744118600000}`
			default:
				reply = `{"orderId":22542181,"clientOrderId":"gct-order","status":"FILLED","origQty":"1","executedQty":"1","updateTime":1744118600000}`
			}
		default:
			assert.Failf(t, "unexpected request", "%s %s should not be requested", r.Method, r.URL.Path)
		}
		if strings.HasPrefix(reply, `{"code":-`) {
			w.WriteHeader(http.StatusBadRequest)
		}
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	}
}

func TestPositionModeCache(t *testing.T) {
	t.Parallel()
	var c positionModeCache
	_, ok, generation := c.get("a")
	assert.False(t, ok, "get should report no mode before a lookup")
	c.set("a", generation, true)
	hedgeMode, ok, _ := c.get("a")
	assert.True(t, ok && hedgeMode, "get should return the mode a lookup set")
	_, ok, generation = c.get("b")
	assert.False(t, ok, "get should keep each account's mode apart")
	c.set("b", generation, false)
	hedgeMode, ok, _ = c.get("b")
	assert.True(t, ok && !hedgeMode, "get should return each account's own mode")

	// A lookup in flight when the mode is forgotten, as one overlapping a change of mode, sets nothing
	_, _, stale := c.get("a")
	c.forget("a")
	c.set("a", stale, true)
	_, ok, generation = c.get("a")
	assert.False(t, ok, "set should not keep the mode of a lookup the account's mode was forgotten during")
	c.set("a", generation, false)
	hedgeMode, ok, _ = c.get("a")
	assert.True(t, ok && !hedgeMode, "set should keep the mode of a lookup started after the account's mode was forgotten")
	_, ok, _ = c.get("b")
	assert.True(t, ok, "forget should leave another account's mode")
}

func TestFuturesPositionModeLookups(t *testing.T) {
	t.Parallel()
	// The position mode is looked up once per account, as context credentials can switch accounts per call, and again
	// after the account changes its mode
	venue := &positionModeVenue{hedge: map[string]bool{"hedge-key": true, "one-way-key": false}}
	ex := newTradingTestExchange(t, venue.serve(t))
	hedgeCtx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "hedge-key", Secret: "secret"})
	oneWayCtx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "one-way-key", Secret: "secret"})
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			side, _, err := ex.futuresPositionSide(hedgeCtx, &order.Submit{Side: order.Long})
			assert.NoError(t, err, "futuresPositionSide should not error")
			assert.Equal(t, PositionSideLong, side, "futuresPositionSide should follow the hedge mode account's mode")
			side, _, err = ex.futuresPositionSide(oneWayCtx, &order.Submit{Side: order.Long})
			assert.NoError(t, err, "futuresPositionSide should not error")
			assert.Empty(t, side, "futuresPositionSide should follow the one-way mode account's mode")
		})
	}
	wg.Wait()
	lookups := venue.lookupCounts()
	assert.Positive(t, lookups["hedge-key"], "futuresPositionSide should look the hedge mode account's mode up")
	assert.Positive(t, lookups["one-way-key"], "futuresPositionSide should look the one-way mode account's mode up")
	for range 3 {
		_, _, err := ex.futuresPositionSide(hedgeCtx, &order.Submit{Side: order.Short})
		require.NoError(t, err, "futuresPositionSide must not error")
	}
	assert.Equal(t, lookups, venue.lookupCounts(), "futuresPositionSide should keep each account's mode")

	require.NoError(t, ex.USDTMarginedFuturesChangePositionMode(hedgeCtx, false), "USDTMarginedFuturesChangePositionMode must not error")
	side, _, err := ex.futuresPositionSide(hedgeCtx, &order.Submit{Side: order.Long})
	require.NoError(t, err, "futuresPositionSide must not error")
	assert.Empty(t, side, "futuresPositionSide should follow the account's changed mode")
	_, _, err = ex.futuresPositionSide(oneWayCtx, &order.Submit{Side: order.Long})
	require.NoError(t, err, "futuresPositionSide must not error")
	exp := maps.Clone(lookups)
	exp["hedge-key"]++
	assert.Equal(t, exp, venue.lookupCounts(), "USDTMarginedFuturesChangePositionMode should have only the changed account's mode looked up again")
}

func TestSubmitFuturesOrderPositionMode(t *testing.T) {
	t.Parallel()
	// The cached position mode goes stale when the account's mode is changed elsewhere, which Binance answers by
	// rejecting the order's position side, or the reduce only flag Hedge Mode does not take
	const marginInsufficient = `{"code":-2019,"msg":"Margin is insufficient."}`
	long := order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Market, Side: order.Long, Amount: 0.002}
	for _, tc := range []struct {
		name           string
		hedgeMode      bool         // the mode the first order caches
		changedMode    bool         // the account's mode when the second order is placed
		reject         string       // the reply every second order gets, when set
		second         order.Submit // the second order
		expSides       []string     // the position side and reduce only flag of every order sent, in turn
		expSecondLooks int          // the lookups the second order makes
		expErr         bool
	}{
		{
			name: "stale one-way mode", changedMode: true, second: long,
			expSides: []string{"/", "/", "LONG/"}, expSecondLooks: 1,
		},
		{
			name: "stale one-way mode with reduce only", changedMode: true,
			second:   order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Market, Side: order.Long, Amount: 0.002, ReduceOnly: true},
			expSides: []string{"/", "/true", "SHORT/"}, expSecondLooks: 1,
		},
		{
			name: "stale hedge mode", hedgeMode: true, second: long,
			expSides: []string{"LONG/", "LONG/", "/"}, expSecondLooks: 1,
		},
		{
			name: "stale COIN-M one-way mode", changedMode: true,
			second:   order.Submit{AssetType: asset.CoinMarginedFutures, Pair: currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), Type: order.Market, Side: order.Short, Amount: 1},
			expSides: []string{"/", "/", "SHORT/"}, expSecondLooks: 1,
		},
		{
			name: "stale one-way mode of a conditional order", changedMode: true,
			second:   order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.StopMarket, Side: order.Short, Amount: 0.002, TriggerPrice: 70000},
			expSides: []string{"/", "/", "SHORT/"}, expSecondLooks: 1,
		},
		{
			name: "rejected again", reject: `{"code":-4061,"msg":"Order's position side does not match user's setting."}`, second: long,
			expSides: []string{"/", "/", "/"}, expSecondLooks: 1, expErr: true,
		},
		{
			name: "rejected otherwise", reject: marginInsufficient, second: long,
			expSides: []string{"/", "/"}, expErr: true,
		},
		{
			name: "rejected for another parameter", reject: `{"code":-1106,"msg":"Parameter 'timeInForce' sent when not required."}`, second: long,
			expSides: []string{"/", "/"}, expErr: true,
		},
		{
			name: "buy order", hedgeMode: true, changedMode: true,
			second:   order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Market, Side: order.Buy, Amount: 0.002},
			expSides: []string{"LONG/", "/"}, expErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			venue := &positionModeVenue{hedge: map[string]bool{"test-key": tc.hedgeMode}}
			ex := newTradingTestExchange(t, venue.serve(t))
			first := long
			first.Exchange = ex.Name
			_, err := ex.SubmitOrder(t.Context(), &first)
			require.NoError(t, err, "SubmitOrder must place the first order")
			venue.setHedgeMode("test-key", tc.changedMode)
			venue.mu.Lock()
			venue.reject = tc.reject
			venue.mu.Unlock()
			second := tc.second
			second.Exchange = ex.Name
			_, err = ex.SubmitOrder(t.Context(), &second)
			if tc.expErr {
				assert.ErrorIs(t, err, errAPIResponse, "SubmitOrder should return the rejection")
			} else {
				assert.NoError(t, err, "SubmitOrder should place the order with the account's current mode")
			}
			assert.Equal(t, tc.expSides, venue.positionSides(), "SubmitOrder should send each order with the expected position side")
			assert.Equal(t, 1+tc.expSecondLooks, venue.lookupCounts()["test-key"], "SubmitOrder should look the mode up again only after a rejection for it")
			if tc.reject == marginInsufficient {
				_, err = ex.SubmitOrder(t.Context(), &second)
				require.ErrorIs(t, err, errAPIResponse, "SubmitOrder must return the rejection")
				assert.Equal(t, 2, venue.lookupCounts()["test-key"], "a rejected Long or Short order should have the mode looked up again for the next")
			}
		})
	}
}

func TestPositionModeChangeOverlappingLookup(t *testing.T) {
	t.Parallel()
	// A lookup made while USDTMarginedFuturesChangePositionMode is in flight can read the mode the change replaces, so its
	// mode is not kept once the change returns
	changeStarted, lookupAnswered := make(chan struct{}), make(chan struct{})
	startChange, answerLookup := sync.OnceFunc(func() { close(changeStarted) }), sync.OnceFunc(func() { close(lookupAnswered) })
	t.Cleanup(answerLookup)
	var mu sync.Mutex
	hedge, lookups := false, 0
	ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			startChange()
			select {
			case <-lookupAnswered:
			case <-time.After(30 * time.Second):
			}
			mu.Lock()
			hedge = r.URL.Query().Get("dualSidePosition") == "true"
			mu.Unlock()
			_, err := w.Write([]byte(`{"code":200,"msg":"success"}`))
			assert.NoError(t, err, "Write should not error")
			return
		}
		mu.Lock()
		lookups++
		reply := fmt.Sprintf(`{"dualSidePosition":%t}`, hedge)
		mu.Unlock()
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	})
	looked := make(chan struct{})
	go func() {
		defer close(looked)
		defer answerLookup()
		select {
		case <-changeStarted:
		case <-time.After(30 * time.Second):
			assert.Fail(t, "USDTMarginedFuturesChangePositionMode should reach Binance")
			return
		}
		side, _, err := ex.futuresPositionSide(t.Context(), &order.Submit{Side: order.Long})
		assert.NoError(t, err, "futuresPositionSide should not error")
		assert.Empty(t, side, "futuresPositionSide should read the mode the change replaces")
	}()
	require.NoError(t, ex.USDTMarginedFuturesChangePositionMode(t.Context(), true), "USDTMarginedFuturesChangePositionMode must not error")
	select {
	case <-looked:
	case <-time.After(30 * time.Second):
		require.FailNow(t, "the lookup made during the change must return")
	}
	side, _, err := ex.futuresPositionSide(t.Context(), &order.Submit{Side: order.Long})
	require.NoError(t, err, "futuresPositionSide must not error")
	assert.Equal(t, PositionSideLong, side, "futuresPositionSide should follow the changed mode, not a lookup made during the change")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, lookups, "futuresPositionSide should look the mode up again after the change")
}

func TestPositionModeLookupRate(t *testing.T) {
	t.Parallel()
	// Order book snapshots can hold the derivatives IP pool for seconds, so the position mode lookup a Long or Short order
	// makes draws on the cancel pool, while Get Current Position Mode stays on the IP pool
	venue := &positionModeVenue{hedge: map[string]bool{"test-key": true}}
	ex := newTradingTestExchange(t, venue.serve(t))
	require.NoError(t, ex.EnableRateLimiter(), "EnableRateLimiter must not error")
	exhausted := request.NewRateLimitWithWeight(time.Hour, 1, 1)
	require.NoError(t, exhausted.RateLimit(t.Context()), "RateLimit must take the limiter's only token")
	definitions := ex.Requester.GetRateLimiterDefinitions()
	for epl, cost := range endpointCosts {
		if cost.pool == derivativesIPPool {
			definitions[epl] = exhausted
		}
	}
	ctx := request.WithDelayNotAllowed(t.Context())
	_, err := ex.GetUSDTMarginedFuturesCurrentPositionMode(ctx)
	require.ErrorIs(t, err, request.ErrDelayNotAllowed, "GetUSDTMarginedFuturesCurrentPositionMode must draw on the IP pool")
	_, err = ex.SubmitOrder(ctx, &order.Submit{Exchange: ex.Name, AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Market, Side: order.Long, Amount: 0.002})
	require.NoError(t, err, "SubmitOrder must not wait on the IP pool to look up a Long order's position mode")
	assert.Equal(t, []string{"LONG/"}, venue.positionSides(), "SubmitOrder should place the order in the account's position mode")

	// With the cancel pool taken, as by a cancellation of every order, the lookup queues on the IP pool as it did before
	venue = &positionModeVenue{hedge: map[string]bool{"test-key": true}}
	ex = newTradingTestExchange(t, venue.serve(t))
	require.NoError(t, ex.EnableRateLimiter(), "EnableRateLimiter must not error")
	slow := request.NewRateLimitWithWeight(100*time.Millisecond, 1, 1)
	require.NoError(t, slow.RateLimit(t.Context()), "RateLimit must take the limiter's only token")
	definitions = ex.Requester.GetRateLimiterDefinitions()
	for epl, cost := range endpointCosts {
		switch cost.pool {
		case derivativesCancelPool:
			definitions[epl] = exhausted
		case derivativesIPPool:
			definitions[epl] = slow
		}
	}
	deadline, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	_, err = ex.SubmitOrder(deadline, &order.Submit{Exchange: ex.Name, AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Market, Side: order.Long, Amount: 0.002})
	require.NoError(t, err, "SubmitOrder must look a Long order's position mode up on the IP pool when the cancel pool is taken")
	assert.Equal(t, []string{"LONG/"}, venue.positionSides(), "SubmitOrder should place the order in the account's position mode")
}

func TestCompatibleOrderVars(t *testing.T) {
	t.Parallel()
	got, err := compatibleOrderVars(asset.Spot, "SELL", "EXPIRED_IN_MATCH", "TAKE_PROFIT_LIMIT", "FOK")
	require.NoError(t, err, "compatibleOrderVars must not error for spot values")
	assert.Equal(t, OrderVars{Side: order.Sell, Status: order.Expired, OrderType: order.TakeProfitLimit, TimeInForce: order.FillOrKill}, got, "compatibleOrderVars should convert spot values")

	got, err = compatibleOrderVars(asset.Margin, "BUY", "PENDING_NEW", "LIMIT_MAKER", "")
	require.NoError(t, err, "compatibleOrderVars must not error for margin values")
	assert.Equal(t, OrderVars{Side: order.Buy, Status: order.Pending, OrderType: order.LimitMaker}, got, "compatibleOrderVars should convert margin values and leave an empty time in force unset")

	got, err = compatibleOrderVars(asset.USDTMarginedFutures, "SELL", "EXPIRED_IN_MATCH", "TRAILING_STOP_MARKET", "RPI")
	require.NoError(t, err, "compatibleOrderVars must not error for USDⓈ-M values")
	assert.Equal(t, OrderVars{Side: order.Sell, Status: order.Expired, OrderType: order.TrailingStop, TimeInForce: order.PostOnly}, got, "compatibleOrderVars should convert USDⓈ-M values")

	got, err = compatibleOrderVars(asset.Options, "BUY", "ACCEPTED", "LIMIT", "GTC")
	require.NoError(t, err, "compatibleOrderVars must not error for options values")
	assert.Equal(t, OrderVars{Side: order.Buy, Status: order.New, OrderType: order.Limit, TimeInForce: order.GoodTillCancel}, got, "compatibleOrderVars should convert options values")

	got, err = compatibleOrderVars(asset.CoinMarginedFutures, "SIDEWAYS", "SUSPENDED", "ICEBERG", "DAY")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "compatibleOrderVars should return the side error")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "compatibleOrderVars should return the type error")
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce, "compatibleOrderVars should return the time in force error")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "compatibleOrderVars should return the status error")
	assert.Equal(t, OrderVars{}, got, "compatibleOrderVars should leave unconvertible values unset")

	_, err = compatibleOrderVars(asset.Spot, "BUY", "NEW", "STOP_MARKET", "GTX")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "compatibleOrderVars should reject a futures type on spot")
	_, err = compatibleOrderVars(asset.Spot, "BUY", "NEW_INSURANCE", "LIMIT", "GTC")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "compatibleOrderVars should reject a futures status on spot")
}

func TestMarginOrderListPairs(t *testing.T) {
	t.Parallel()
	pairs := currency.Pairs{currency.NewBTCUSDT()}
	got, err := marginOrderListPairs(pairs, false)
	require.NoError(t, err, "marginOrderListPairs must not error for the cross margin account")
	assert.Equal(t, currency.Pairs{currency.EMPTYPAIR}, got, "marginOrderListPairs should request every cross margin symbol at once")
	got, err = marginOrderListPairs(pairs, true)
	require.NoError(t, err, "marginOrderListPairs must not error for isolated margin pairs")
	assert.Equal(t, pairs, got, "marginOrderListPairs should request isolated margin pairs one by one")
	_, err = marginOrderListPairs(nil, true)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "marginOrderListPairs should require isolated margin pairs")
}

func TestOpenOrdersPairs(t *testing.T) {
	t.Parallel()
	pairs := func(n int) currency.Pairs {
		p := make(currency.Pairs, n)
		for i := range p {
			p[i] = currency.NewPair(currency.NewCode(fmt.Sprintf("COIN%d", i)), currency.USDT)
		}
		return p
	}
	every := currency.Pairs{currency.EMPTYPAIR}
	assert.Equal(t, every, openOrdersPairs(asset.Spot, nil), "openOrdersPairs should request every spot symbol without pairs")
	assert.Equal(t, pairs(13), openOrdersPairs(asset.Spot, pairs(13)), "openOrdersPairs should request 13 spot pairs one by one")
	assert.Equal(t, every, openOrdersPairs(asset.Spot, pairs(14)), "openOrdersPairs should request every spot symbol for 14 pairs")
	assert.Equal(t, pairs(39), openOrdersPairs(asset.USDTMarginedFutures, pairs(39)), "openOrdersPairs should request 39 USDⓈ-M pairs one by one")
	assert.Equal(t, every, openOrdersPairs(asset.Options, pairs(40)), "openOrdersPairs should request every options symbol for 40 pairs")
	assert.Equal(t, pairs(500), openOrdersPairs(asset.Margin, pairs(500)), "openOrdersPairs should request margin pairs one by one")
	assert.Equal(t, every, openOrdersPairs(asset.Margin, nil), "openOrdersPairs should request every margin symbol without pairs")
}

func TestFuturesOrdersInWindow(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1744000000000)
	orderTimes := make([]time.Time, 0, 27)
	for i := range 25 {
		orderTimes = append(orderTimes, start.Add(time.Duration(i)*time.Hour))
	}
	// Two orders share a millisecond with the window's end, so they must both fall in the same window
	orderTimes = append(orderTimes, start.Add(10*24*time.Hour), start.Add(10*24*time.Hour))
	end := start.Add(10 * 24 * time.Hour)
	const limit = 4
	var windows int
	got, err := futuresOrdersInWindow(start, end, limit, func(windowStart, windowEnd time.Time) ([]time.Time, error) {
		windows++
		// A pager that keeps splitting fails here rather than hanging the test
		if !assert.LessOrEqual(t, windows, 1000, "futuresOrdersInWindow should stop requesting") {
			return nil, errAPIResponse
		}
		assert.Less(t, windowEnd.Sub(windowStart), 7*24*time.Hour, "futuresOrdersInWindow should request windows shorter than seven days")
		var orders []time.Time
		for _, orderTime := range orderTimes {
			if !orderTime.Before(windowStart) && !orderTime.After(windowEnd) && len(orders) < limit {
				orders = append(orders, orderTime)
			}
		}
		return orders, nil
	})
	require.NoError(t, err, "futuresOrdersInWindow must not error")
	assert.Equal(t, orderTimes, got, "futuresOrdersInWindow should collect every order once, in order")
	assert.Greater(t, windows, 2, "futuresOrdersInWindow should split windows whose reply is full")

	// Only a window starting and ending on the same millisecond can't be split, so a window of two milliseconds that
	// together hold more than a page is split, and a millisecond holding a full page is returned as it is
	split := []time.Time{start, start.Add(time.Millisecond), start.Add(time.Millisecond)}
	windows = 0
	got, err = futuresOrdersInWindow(start, start.Add(time.Millisecond), 2, func(windowStart, windowEnd time.Time) ([]time.Time, error) {
		windows++
		if !assert.LessOrEqual(t, windows, 1000, "futuresOrdersInWindow should stop requesting") {
			return nil, errAPIResponse
		}
		var orders []time.Time
		for _, orderTime := range split {
			if !orderTime.Before(windowStart) && !orderTime.After(windowEnd) && len(orders) < 2 {
				orders = append(orders, orderTime)
			}
		}
		return orders, nil
	})
	require.NoError(t, err, "futuresOrdersInWindow must not error for a window it can split")
	assert.Equal(t, split, got, "futuresOrdersInWindow should split a window down to single milliseconds")

	_, err = futuresOrdersInWindow(start, end, limit, func(time.Time, time.Time) ([]time.Time, error) {
		return nil, errAPIResponse
	})
	assert.ErrorIs(t, err, errAPIResponse, "futuresOrdersInWindow should return the request error")
}

func TestPositionSideFor(t *testing.T) {
	t.Parallel()
	assert.Equal(t, PositionSideLong, positionSideFor(order.Long), "positionSideFor should return LONG for a long direction")
	assert.Equal(t, PositionSideLong, positionSideFor(order.Buy), "positionSideFor should return LONG for a buy direction")
	assert.Equal(t, PositionSideShort, positionSideFor(order.Short), "positionSideFor should return SHORT for a short direction")
	assert.Equal(t, PositionSideBoth, positionSideFor(order.UnknownSide), "positionSideFor should return BOTH without a direction")
}
