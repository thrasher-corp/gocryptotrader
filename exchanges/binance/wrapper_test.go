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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
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
		tickers, err := e.GetEOptions24hrTickerPriceChangeStatistics(t.Context(), currency.EMPTYPAIR)
		if err != nil || len(tickers) == 0 {
			return
		}
		busiest := slices.MaxFunc(tickers, func(a, b EOptionTicker) int { return cmp.Compare(a.TradeCount, b.TradeCount) })
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

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		assert.Truef(t, ex.IsAssetWebsocketSupported(a), "SetDefaults should support %s websocket market data", a)
	}
	assert.False(t, ex.IsAssetWebsocketSupported(asset.Margin), "SetDefaults should fetch margin market data over REST")
	assert.Equal(t, map[kline.Interval]bool{kline.OneHour: true, kline.FourHour: true, kline.EightHour: true}, ex.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies, "SetDefaults should list every funding interval")
	assert.Equal(t, map[asset.Item]bool{asset.USDTMarginedFutures: true, asset.CoinMarginedFutures: true}, ex.Features.Supports.FuturesCapabilities.FundingRateBatching, "SetDefaults should batch funding rates of both futures assets")
	for _, interval := range []kline.Interval{kline.HundredMilliseconds, kline.FiveHundredMilliseconds} {
		assert.Falsef(t, ex.Features.Enabled.Kline.Intervals.ExchangeSupported(interval), "SetDefaults should not enable %s klines", interval)
	}
	for _, interval := range []kline.Interval{kline.ThousandMilliseconds, kline.OneMin, kline.EightHour, kline.ThreeDay, kline.OneMonth} {
		assert.Truef(t, ex.Features.Enabled.Kline.Intervals.ExchangeSupported(interval), "SetDefaults should enable %s klines", interval)
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
		empty := newRoutedTestExchange(t, map[string]string{tc.path: "[]"})
		_, err = empty.UpdateTicker(t.Context(), tc.pair, tc.a)
		require.ErrorIsf(t, err, ticker.ErrTickerNotFound, "UpdateTicker must report a missing %s ticker", tc.a)
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

func TestUFuturesSymbolPair(t *testing.T) {
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
		assert.Equalf(t, tc.exp, uFuturesSymbolPair(tc.symbol, tc.baseAsset, tc.quote), "uFuturesSymbolPair should split %s", tc.symbol)
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
					OrderID:       "2146764",
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
					TriggerPrice:         90000,
					AverageExecutedPrice: 90000,
					Exchange:             "Binance",
					OrderID:              "2146760",
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
					OrderID:       "2146764",
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
					TimeInForce:     order.GoodTillCrossing,
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
			resp, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: orderType, Side: order.AnySide, Pairs: currency.Pairs{currency.NewBTCUSDT()}})
			require.NoErrorf(t, err, "GetActiveOrders must list %s orders over the WebSocket API", orderType)
			if mockTests {
				assert.Equalf(t, exp, resp, "GetActiveOrders should return the %s orders listed over the WebSocket API", orderType)
			}
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
					OrderID:              "2146765",
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
			resp, err := e.GetOrderHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetOrderHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, resp, "GetOrderHistory should return the orders")
			}
		})
	}
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
		{"USDⓈ-M unsupported type", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.IOS, Side: order.Sell, Amount: 1}, order.ErrUnsupportedOrderType},
		{"USDⓈ-M quote amount", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Market, Side: order.Buy, QuoteAmount: 100}, order.ErrAmountIsInvalid},
		{"USDⓈ-M stop limit without price", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.StopLimit, Side: order.Sell, TriggerPrice: 1, Amount: 1}, order.ErrPriceMustBeSetIfLimitOrder},
		{"USDⓈ-M good till day without end time", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Sell, Price: 1, Amount: 1, TimeInForce: order.GoodTillDay}, errGoodTillDateRequired},
		{"USDⓈ-M stop market without trigger price", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.StopMarket, Side: order.Sell, Amount: 1}, errTriggerPriceRequired},
		{"USDⓈ-M trailing stop distance", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.TrailingStop, Side: order.Sell, Amount: 1, TrackingMode: order.Distance, TrackingValue: 100}, errTrailingStopPercentageRequired},
		{"USDⓈ-M index price trigger", &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.StopMarket, Side: order.Sell, Amount: 1, TriggerPrice: 1, TriggerPriceType: order.IndexPrice}, order.ErrUnknownPriceType},
		{"COIN-M stop", &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.Stop, Side: order.Sell, Price: 1, TriggerPrice: 1, Amount: 1}, order.ErrUnsupportedOrderType},
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
				Type:            order.Limit,
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
				Status:        order.Filled,
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
				Status:        order.Filled,
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
				OrderID:       "2146761",
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
				OrderID:       "2146762",
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
				OrderID:       "2146763",
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
				Status:        order.Filled,
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
			tc.submit.Exchange = ex.Name
			tc.submit.Pair = currency.NewBTCUSDT()
			tc.submit.AssetType = asset.Spot
			resp, err := ex.SubmitOrder(t.Context(), tc.submit)
			require.NoErrorf(t, err, "SubmitOrder must place the %s order over the WebSocket API", tc.submit.Type)
			if mockTests {
				assert.Equalf(t, tc.exp, resp, "SubmitOrder should return the %s order placed over the WebSocket API", tc.submit.Type)
			}
		}
	})
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject a nil cancellation")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must reject an unsupported asset")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Margin, MarginType: margin.Isolated})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllOrders must reject isolated margin without a pair")

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
		for _, pair := range []currency.Pair{currency.NewBTCUSDT(), currency.EMPTYPAIR} {
			resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: pair})
			require.NoErrorf(t, err, "CancelAllOrders must cancel %q orders over the WebSocket API", pair)
			if mockTests {
				assert.Equalf(t, exp, resp, "CancelAllOrders should return the statuses of %q orders cancelled over the WebSocket API", pair)
			}
		}
	})
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	ex := newMarketTestExchange(t)
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an unsupported asset")
	subAccountCtx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", SubAccount: "sub"})
	_, err = ex.UpdateAccountBalances(subAccountCtx, asset.Spot)
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "UpdateAccountBalances must reject spot sub-accounts")

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
				currency.USDT: {Currency: currency.USDT, Total: 10099.448, Hold: 1084.52138, Free: 8725.92524},
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

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must reject an empty pair")
	_, err = e.GetOrderInfo(t.Context(), "", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must reject an empty order ID")
	_, err = e.GetOrderInfo(t.Context(), "abc", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, errInvalidOrderID, "GetOrderInfo must reject an invalid order ID")
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
				OrderID:              "2146760",
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
				TimeInForce:          order.GoodTillCancel | order.PostOnly,
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

	// SubmitOrder identifies OCO orders by order list ID and USDⓈ-M conditional orders by algo ID
	const notFound = `{"code":-2013,"msg":"Order does not exist."}`
	listReply := func(symbol string) string {
		return `{"orderListId":1930,"contingencyType":"OCO","listStatusType":"EXEC_STARTED","listOrderStatus":"EXECUTING","listClientOrderId":"gct-oco","transactionTime":1791284812300,"symbol":"` + symbol + `"}`
	}
	for _, tc := range []struct {
		name    string
		a       asset.Item
		replies map[string]string
		exp     *order.Detail
		err     error
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
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), url), "SetRunningURL must not error")
			got, err := ex.GetOrderInfo(t.Context(), "1930", currency.NewBTCUSDT(), tc.a)
			require.ErrorIs(t, err, tc.err, "GetOrderInfo must return the expected error")
			assert.Equal(t, tc.exp, got, "GetOrderInfo should return the order list")
		})
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder must reject a nil modification")
	modify := &order.Modify{Pair: currency.NewBTCUSDT(), OrderID: "1234"}
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		modify.AssetType = a
		_, err = e.ModifyOrder(t.Context(), modify)
		require.ErrorIsf(t, err, asset.ErrNotSupported, "ModifyOrder must reject %s orders", a)
	}
	modify.AssetType = asset.USDTMarginedFutures
	modify.OrderID = "abc"
	_, err = e.ModifyOrder(t.Context(), modify)
	require.ErrorIs(t, err, errInvalidOrderID, "ModifyOrder must reject an invalid order ID")
	modify.OrderID = "20072994037"
	_, err = e.ModifyOrder(t.Context(), modify)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "ModifyOrder must reject a modification without a side")

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
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must reject an empty batch")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: a, Pair: currency.NewBTCUSDT(), OrderID: "1"}})
		require.ErrorIsf(t, err, asset.ErrNotSupported, "CancelBatchOrders must reject %s orders", a)
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
		assert.Equalf(t, exp, depositStatusString(status), "depositStatusString should name status %d", status)
	}
	for status, exp := range map[uint64]string{0: "Email sent", 2: "Awaiting approval", 3: "Rejected", 4: "Processing", 6: "Completed", 9: "9"} {
		assert.Equalf(t, exp, withdrawalStatusString(status), "withdrawalStatusString should name status %d", status)
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
		assert.Equalf(t, tc.exp, e.FormatExchangeKlineInterval(tc.interval), "FormatExchangeKlineInterval should format %s", tc.interval)
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
			result, err := e.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), a, start, end)
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

func TestFormatExchangeCurrency(t *testing.T) {
	t.Parallel()
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
		result, err := e.FormatExchangeCurrency(tc.pair, tc.a)
		require.NoErrorf(t, err, "FormatExchangeCurrency must not error for %s %s", tc.a, tc.pair)
		assert.Equalf(t, tc.exp, result.String(), "FormatExchangeCurrency should format %s %s", tc.a, tc.pair)
	}
}

func TestFormatSymbol(t *testing.T) {
	t.Parallel()
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
		result, err := e.FormatSymbol(tc.pair, tc.a)
		require.NoErrorf(t, err, "FormatSymbol must not error for %s %s", tc.a, tc.pair)
		assert.Equalf(t, tc.exp, result, "FormatSymbol should format %s %s", tc.a, tc.pair)
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
			result, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
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
		ex := newRoutedTestExchange(t, tc.replies)
		_, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: tc.a, Pair: tc.pair, StartDate: start, EndDate: end})
		assert.ErrorIsf(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates should report a missing %s mark price", tc.a)
	}
}

func TestFundingFeeWindows(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1744070400000)
	var result [][2]time.Time
	for windowStart, windowEnd := range fundingFeeWindows(start, start.Add(1200*time.Hour)) {
		result = append(result, [2]time.Time{windowStart, windowEnd})
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
		result, err := e.IsPerpetualFutureCurrency(tc.a, tc.pair)
		require.ErrorIsf(t, err, tc.err, "IsPerpetualFutureCurrency must return the expected error for %s %s", tc.a, tc.pair)
		assert.Equalf(t, tc.exp, result, "IsPerpetualFutureCurrency should report whether %s %s is perpetual", tc.a, tc.pair)
	}
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.CoinMarginedFutures, asset.Options} {
		_, err := e.GetCollateralMode(t.Context(), a)
		require.ErrorIsf(t, err, asset.ErrNotSupported, "GetCollateralMode must reject %s", a)
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

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	err := e.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.PortfolioMode)
	require.ErrorIs(t, err, order.ErrCollateralInvalid, "SetCollateralMode must reject portfolio mode")
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.CoinMarginedFutures, asset.Options} {
		err = e.SetCollateralMode(t.Context(), a, collateral.SingleMode)
		require.ErrorIsf(t, err, asset.ErrNotSupported, "SetCollateralMode must reject %s", a)
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
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		arg.Asset = a
		_, err = e.ChangePositionMargin(t.Context(), arg)
		require.ErrorIsf(t, err, asset.ErrNotSupported, "ChangePositionMargin must reject %s", a)
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
	for _, invalid := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err = e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Pair: arg.Pair, Asset: arg.Asset, MarginType: arg.MarginType, NewAllocatedMargin: invalid, OriginalAllocatedMargin: 1337})
		require.ErrorIsf(t, err, errInvalidMarginAmount, "ChangePositionMargin must reject a new allocated margin of %v", invalid)
		_, err = e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Pair: arg.Pair, Asset: arg.Asset, MarginType: arg.MarginType, NewAllocatedMargin: 1337, OriginalAllocatedMargin: invalid})
		require.ErrorIsf(t, err, errInvalidMarginAmount, "ChangePositionMargin must reject an original allocated margin of %v", invalid)
	}
	arg.OriginalAllocatedMargin = arg.NewAllocatedMargin
	_, err = e.ChangePositionMargin(t.Context(), arg)
	require.ErrorIs(t, err, errMarginAmountRequired, "ChangePositionMargin must reject an unchanged margin")

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
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: a, Pair: currency.NewBTCUSDT()})
		require.ErrorIsf(t, err, asset.ErrNotSupported, "GetFuturesPositionSummary must reject %s", a)
	}

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
		ex := newRoutedTestExchange(t, tc.replies)
		_, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: tc.a, Pair: tc.pair})
		assert.ErrorIsf(t, err, tc.err, "GetFuturesPositionSummary should report the missing %s", tc.name)
	}
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
		{name: "from an order ID", end: start.Add(72 * time.Hour), fromID: 7, maxSpan: time.Hour, requests: []window{{start, start.Add(72 * time.Hour)}}, orders: []testOrder{at(1, 0), at(2, 0), at(3, 0)}, exp: []uint64{1, 2}},
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
}

func TestFilterOrders(t *testing.T) {
	t.Parallel()
	orders := []order.Detail{
		{OrderID: "1", Type: order.StopLimit, Side: order.Sell},
		{OrderID: "2", Type: order.StopMarket, Side: order.Sell},
		{OrderID: "3", Type: order.TakeProfitLimit, Side: order.Sell},
		{OrderID: "4", Type: order.TakeProfitMarket, Side: order.Buy},
		{OrderID: "5", Type: order.TrailingStop, Side: order.Sell},
		{OrderID: "6", Type: order.Limit, Side: order.Sell},
	}
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
		got := e.filterOrders(&order.MultiOrderRequest{Type: tc.orderType, Side: tc.side}, orders)
		ids := make([]string, len(got))
		for i := range got {
			ids[i] = got[i].OrderID
		}
		assert.Equalf(t, tc.exp, ids, "filterOrders should keep the %s %s orders", tc.orderType, tc.side)
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
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		arg.Asset = a
		_, err = e.GetFuturesPositionOrders(t.Context(), arg)
		require.ErrorIsf(t, err, asset.ErrNotSupported, "GetFuturesPositionOrders must reject %s", a)
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
		resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: tc.a, Pairs: currency.Pairs{tc.pair}, StartDate: start, RespectOrderHistoryLimits: true})
		require.NoErrorf(t, err, "GetFuturesPositionOrders must not error for %s", tc.a)
		assert.Equalf(t, tc.exp, resp, "GetFuturesPositionOrders should return the %s position's orders", tc.a)
	}
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		err := e.SetMarginType(t.Context(), a, currency.NewBTCUSDT(), margin.Isolated)
		require.ErrorIsf(t, err, asset.ErrNotSupported, "SetMarginType must reject %s", a)
	}
	err := e.SetMarginType(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.SpotIsolated)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType, "SetMarginType must reject an unsupported margin type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	for _, tc := range []struct {
		a  asset.Item
		mt margin.Type
	}{
		{asset.USDTMarginedFutures, margin.Isolated},
		{asset.USDTMarginedFutures, margin.Multi},
		{asset.CoinMarginedFutures, margin.Isolated},
	} {
		assert.NoErrorf(t, e.SetMarginType(t.Context(), tc.a, assetToTradablePairMap[tc.a], tc.mt), "SetMarginType should not error for %s %s", tc.a, tc.mt)
	}
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverage(t.Context(), asset.USDTMarginedFutures, currency.EMPTYPAIR, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLeverage must reject an empty pair")
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Options} {
		_, err = e.GetLeverage(t.Context(), a, currency.NewBTCUSDT(), margin.Multi, order.UnknownSide)
		require.ErrorIsf(t, err, asset.ErrNotSupported, "GetLeverage must reject %s, whose leverage Binance does not report", a)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
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
		if !mockTests && tc.err != nil {
			continue
		}
		if !mockTests {
			tc.pair = assetToTradablePairMap[tc.a]
		}
		leverage, err := e.GetLeverage(t.Context(), tc.a, tc.pair, margin.Unset, order.UnknownSide)
		require.ErrorIsf(t, err, tc.err, "GetLeverage must return the expected error for %s %s", tc.a, tc.pair)
		if mockTests {
			assert.Equalf(t, tc.exp, leverage, "GetLeverage should return the %s %s leverage", tc.a, tc.pair)
		}
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Spot, asset.Options} {
		err := e.SetLeverage(t.Context(), a, currency.NewBTCUSDT(), margin.Multi, 5, order.UnknownSide)
		require.ErrorIsf(t, err, asset.ErrNotSupported, "SetLeverage must reject %s", a)
	}
	err := e.SetLeverage(t.Context(), asset.Margin, currency.NewBTCUSDT(), margin.Isolated, 5, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported, "SetLeverage must reject isolated margin")
	for _, amount := range []float64{0, 2.5} {
		err = e.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, amount, order.UnknownSide)
		require.ErrorIsf(t, err, errInvalidLeverage, "SetLeverage must reject a leverage of %v", amount)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
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
		assert.NoErrorf(t, e.SetLeverage(t.Context(), tc.a, assetToTradablePairMap[tc.a], tc.mt, tc.leverage, order.UnknownSide), "SetLeverage should not error for %s %s", tc.a, tc.mt)
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
		assert.Equalf(t, exp, e.futuresContractType(contractType, "BTCUSDT"), "futuresContractType should convert %s", contractType)
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

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL must reject an empty pair")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Futures, currency.NewBTCUSDT())
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must reject an unsupported asset")

	if !mockTests {
		for a, pair := range marketTestPairs(t) {
			result, err := e.GetCurrencyTradeURL(t.Context(), a, pair)
			require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
			assert.NotEmptyf(t, result, "GetCurrencyTradeURL should return a URL for %s", a)
		}
		return
	}
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.USDTMarginedFutures, currency.NewPairWithDelimiter("ETHUSDT", "261225", currency.UnderscoreDelimiter))
	require.ErrorIs(t, err, currency.ErrPairNotFound, "GetCurrencyTradeURL must reject a contract that is not listed")
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
		result, err := e.GetCurrencyTradeURL(t.Context(), tc.a, tc.pair)
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s %s", tc.a, tc.pair)
		assert.Equalf(t, tc.exp, result, "GetCurrencyTradeURL should return the trade page of %s %s", tc.a, tc.pair)
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
	server := httptest.NewTestServer(t, handler)
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

func TestRemainingAmountAndAveragePrice(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.006, remainingAmount(0.01, 0.004), "remainingAmount should subtract exactly")
	assert.Equal(t, 79990.0, averagePrice(39.995, 0.0005), "averagePrice should divide exactly")
	assert.Zero(t, averagePrice(39.995, 0), "averagePrice should be zero without fills")
}

func TestOrderSideString(t *testing.T) {
	t.Parallel()
	for side, exp := range map[order.Side]string{
		order.Buy: "BUY", order.Bid: "BUY", order.Long: "BUY", order.Sell: "SELL", order.Ask: "SELL", order.Short: "SELL",
	} {
		got, err := orderSideString(side)
		require.NoErrorf(t, err, "orderSideString must not error for %s", side)
		assert.Equalf(t, exp, got, "orderSideString should convert %s", side)
	}
	_, err := orderSideString(order.UnknownSide)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "orderSideString should reject an unknown side")
}

func TestSpotOrderTypeString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
		exp       string
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
		got, err := spotOrderTypeString(tc.orderType, tc.tif)
		require.NoErrorf(t, err, "spotOrderTypeString must not error for %s %s", tc.orderType, tc.tif)
		assert.Equalf(t, tc.exp, got, "spotOrderTypeString should convert %s %s", tc.orderType, tc.tif)
	}
	for _, orderType := range []order.Type{order.TrailingStop, order.Stop, order.TakeProfit} {
		_, err := spotOrderTypeString(orderType, order.UnknownTIF)
		assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "spotOrderTypeString should reject %s, which has no single spot type", orderType)
	}
}

func TestStringToSpotOrderType(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]order.Type{
		"LIMIT": order.Limit, "MARKET": order.Market, "STOP_LOSS": order.StopMarket, "STOP_LOSS_LIMIT": order.StopLimit,
		"TAKE_PROFIT": order.TakeProfitMarket, "TAKE_PROFIT_LIMIT": order.TakeProfitLimit, "LIMIT_MAKER": order.LimitMaker,
	} {
		got, err := stringToSpotOrderType(in)
		require.NoErrorf(t, err, "stringToSpotOrderType must not error for %s", in)
		assert.Equalf(t, exp, got, "stringToSpotOrderType should convert %s", in)
		if exp != order.LimitMaker {
			back, err := spotOrderTypeString(got, order.UnknownTIF)
			require.NoErrorf(t, err, "spotOrderTypeString must not error for %s", got)
			assert.Equalf(t, in, back, "spotOrderTypeString should invert stringToSpotOrderType for %s", in)
		}
	}
	_, err := stringToSpotOrderType("STOP_MARKET")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "stringToSpotOrderType should reject a futures type")
}

func TestFuturesOrderTypeString(t *testing.T) {
	t.Parallel()
	for in, exp := range map[order.Type]string{
		order.Limit: "LIMIT", order.Market: "MARKET", order.StopLimit: "STOP", order.StopMarket: "STOP_MARKET",
		order.TakeProfitLimit: "TAKE_PROFIT", order.TakeProfitMarket: "TAKE_PROFIT_MARKET", order.TrailingStop: "TRAILING_STOP_MARKET",
	} {
		got, err := futuresOrderTypeString(in)
		require.NoErrorf(t, err, "futuresOrderTypeString must not error for %s", in)
		assert.Equalf(t, exp, got, "futuresOrderTypeString should convert %s", in)
		back, err := derivativesOrderType(got)
		require.NoErrorf(t, err, "derivativesOrderType must not error for %s", got)
		assert.Equalf(t, in, back, "derivativesOrderType should invert futuresOrderTypeString for %s", in)
	}
	for _, orderType := range []order.Type{order.LimitMaker, order.Stop, order.TakeProfit} {
		_, err := futuresOrderTypeString(orderType)
		assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "futuresOrderTypeString should reject %s, which has no single futures type", orderType)
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
		assert.Equalf(t, tc.exp, resolveTriggerOrderType(tc.orderType, tc.price), "resolveTriggerOrderType should resolve %s with price %v", tc.orderType, tc.price)
	}
}

func TestIsConditionalOrderType(t *testing.T) {
	t.Parallel()
	for _, orderType := range []order.Type{order.Stop, order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitLimit, order.TakeProfitMarket, order.TrailingStop} {
		assert.Truef(t, isConditionalOrderType(orderType), "isConditionalOrderType should report %s", orderType)
	}
	for _, orderType := range []order.Type{order.Limit, order.Market, order.LimitMaker, order.OCO, order.AnyType} {
		assert.Falsef(t, isConditionalOrderType(orderType), "isConditionalOrderType should not report %s", orderType)
	}
}

func TestTimeInForceString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a   asset.Item
		tif order.TimeInForce
		exp string
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
		got, err := timeInForceString(tc.a, tc.tif)
		require.ErrorIsf(t, err, tc.err, "timeInForceString must return the expected error for %s %s", tc.a, tc.tif)
		assert.Equalf(t, tc.exp, got, "timeInForceString should convert %s %s", tc.a, tc.tif)
	}
}

func TestWorkingTypeString(t *testing.T) {
	t.Parallel()
	got, err := workingTypeString(order.LastPrice)
	require.NoError(t, err, "workingTypeString must not error for the last price")
	assert.Empty(t, got, "workingTypeString should leave the contract price, Binance's default, unsent")
	got, err = workingTypeString(order.MarkPrice)
	require.NoError(t, err, "workingTypeString must not error for the mark price")
	assert.Equal(t, "MARK_PRICE", got, "workingTypeString should convert the mark price")
	_, err = workingTypeString(order.IndexPrice)
	assert.ErrorIs(t, err, order.ErrUnknownPriceType, "workingTypeString should reject the index price")
}

func TestAlgoOrderStatus(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]order.Status{
		"NEW": order.New, "CANCELED": order.Cancelled, "TRIGGERING": order.Pending, "TRIGGERED": order.Active,
		"FINISHED": order.Closed, "REJECTED": order.Rejected, "EXPIRED": order.Expired,
	} {
		got, err := algoOrderStatus(in)
		require.NoErrorf(t, err, "algoOrderStatus must not error for %s", in)
		assert.Equalf(t, exp, got, "algoOrderStatus should convert %s", in)
	}
	_, err := algoOrderStatus("PAUSED")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "algoOrderStatus should reject an unknown status")
}

func TestOrderListStatus(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]order.Status{"EXECUTING": order.Active, "ALL_DONE": order.Closed, "REJECT": order.Rejected} {
		got, err := orderListStatus(in)
		require.NoErrorf(t, err, "orderListStatus must not error for %s", in)
		assert.Equalf(t, exp, got, "orderListStatus should convert %s", in)
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newSpotOrderValues(tc.submit)
			require.ErrorIs(t, err, tc.err, "newSpotOrderValues must return the expected error")
			assert.Equal(t, tc.exp, got, "newSpotOrderValues should work out the order's values")
		})
	}
}

func TestNewSpotOCOLegs(t *testing.T) {
	t.Parallel()
	_, err := newSpotOCOLegs(&order.Submit{Type: order.OCO, Price: 1, TriggerPrice: 1, Amount: 1})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "newSpotOCOLegs must reject an unknown side")
	_, err = newSpotOCOLegs(&order.Submit{Type: order.OCO, Side: order.Sell, TriggerPrice: 1, Amount: 1})
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder, "newSpotOCOLegs must reject a list without a price")
	_, err = newSpotOCOLegs(&order.Submit{Type: order.OCO, Side: order.Sell, Price: 1, Amount: 1})
	require.ErrorIs(t, err, errTriggerPriceRequired, "newSpotOCOLegs must reject a list without a trigger price")
	_, err = newSpotOCOLegs(&order.Submit{
		Type: order.OCO, Side: order.Sell, Price: 2, TriggerPrice: 1, Amount: 1, TimeInForce: order.GoodTillDay,
		RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{LimitPrice: 0.9}},
	})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "newSpotOCOLegs must reject an unsupported stop limit time in force")

	got, err := newSpotOCOLegs(&order.Submit{
		Type: order.OCO, Side: order.Sell, Price: 90000, TriggerPrice: 70000, Amount: 0.001, ClientOrderID: "gct-oco",
		TimeInForce:         order.PostOnly | order.GoodTillCancel,
		RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{LimitPrice: 69000}},
	})
	require.NoError(t, err, "newSpotOCOLegs must not error")
	exp := &spotOCOLegs{
		side:       "SELL",
		limitMaker: OCOOrderListLeg{Type: "LIMIT_MAKER", Price: 90000},
		stopLoss:   OCOOrderListLeg{Type: "STOP_LOSS_LIMIT", Price: 69000, StopPrice: 70000, TimeInForce: "GTC"},
	}
	assert.Equal(t, exp, got, "newSpotOCOLegs should work out a stop limit list's legs")

	got, err = newSpotOCOLegs(&order.Submit{Type: order.OCO, Side: order.Buy, Price: 60000, TriggerPrice: 95000, Amount: 0.002})
	require.NoError(t, err, "newSpotOCOLegs must not error")
	exp = &spotOCOLegs{
		side:       "BUY",
		limitMaker: OCOOrderListLeg{Type: "LIMIT_MAKER", Price: 60000},
		stopLoss:   OCOOrderListLeg{Type: "STOP_LOSS", StopPrice: 95000},
	}
	assert.Equal(t, exp, got, "newSpotOCOLegs should work out a stop loss list's legs without client order IDs")
}

func TestMarginSideEffect(t *testing.T) {
	t.Parallel()
	assert.Empty(t, marginSideEffect(&order.Submit{}), "marginSideEffect should send no side effect by default")
	assert.Equal(t, "MARGIN_BUY", marginSideEffect(&order.Submit{AutoBorrow: true}), "marginSideEffect should borrow")
	assert.Equal(t, "AUTO_REPAY", marginSideEffect(&order.Submit{AutoRepay: true}), "marginSideEffect should repay")
	assert.Equal(t, "AUTO_BORROW_REPAY", marginSideEffect(&order.Submit{AutoBorrow: true, AutoRepay: true}), "marginSideEffect should borrow and repay")
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
			for _, tc := range []struct {
				side           order.Side
				reduceOnly     bool
				hedgedSide     string
				reduceOnlySent bool
			}{
				{side: order.Buy, reduceOnly: true, reduceOnlySent: true},
				{side: order.Sell},
				{side: order.Long, hedgedSide: "LONG"},
				{side: order.Long, reduceOnly: true, hedgedSide: "SHORT", reduceOnlySent: true},
				{side: order.Short, hedgedSide: "SHORT"},
				{side: order.Short, reduceOnly: true, hedgedSide: "LONG", reduceOnlySent: true},
			} {
				positionSide, reduceOnly, err := ex.futuresPositionSide(t.Context(), &order.Submit{Side: tc.side, ReduceOnly: tc.reduceOnly})
				require.NoErrorf(t, err, "futuresPositionSide must not error for %s reduce only %t", tc.side, tc.reduceOnly)
				expSide, expReduceOnly := "", tc.reduceOnlySent
				if hedgeMode && tc.hedgedSide != "" {
					expSide, expReduceOnly = tc.hedgedSide, false
				}
				assert.Equalf(t, expSide, positionSide, "futuresPositionSide should return the position side for %s reduce only %t", tc.side, tc.reduceOnly)
				assert.Equalf(t, expReduceOnly, reduceOnly, "futuresPositionSide should return the reduce only flag for %s reduce only %t", tc.side, tc.reduceOnly)
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

	_, err = futuresOrdersInWindow(start, end, limit, func(time.Time, time.Time) ([]time.Time, error) {
		return nil, errAPIResponse
	})
	assert.ErrorIs(t, err, errAPIResponse, "futuresOrdersInWindow should return the request error")
}

func TestPositionSideFor(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "LONG", positionSideFor(order.Long), "positionSideFor should return LONG for a long direction")
	assert.Equal(t, "LONG", positionSideFor(order.Buy), "positionSideFor should return LONG for a buy direction")
	assert.Equal(t, "SHORT", positionSideFor(order.Short), "positionSideFor should return SHORT for a short direction")
	assert.Equal(t, "BOTH", positionSideFor(order.UnknownSide), "positionSideFor should return BOTH without a direction")
}
