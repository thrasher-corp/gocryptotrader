package bingx

import (
	"errors"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// wrapperPerpetualAssets are the perpetual assets BingX offers
var wrapperPerpetualAssets = []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures}

// wrapperLiveMarginType returns the margin type of a perpetual asset's live test pair
func wrapperLiveMarginType(t *testing.T, ex *Exchange, a asset.Item) margin.Type {
	t.Helper()
	var mt MarginType
	if a == asset.USDTMarginedFutures {
		resp, err := ex.GetSwapMarginType(t.Context(), assetToTradablePairMap[a])
		require.NoError(t, err, "GetSwapMarginType must not error")
		mt = resp.MarginType
	} else {
		resp, err := ex.GetCoinMarginedMarginType(t.Context(), assetToTradablePairMap[a])
		require.NoError(t, err, "GetCoinMarginedMarginType must not error")
		require.NotEmpty(t, resp, "GetCoinMarginedMarginType must return the pair's margin type")
		mt = resp[0].MarginType
	}
	converted, err := convertMarginType(mt)
	require.NoError(t, err, "convertMarginType must not error")
	return converted
}

// wrapperNextFundingTime is the next funding time the recorded mark price and funding rate replies report
var wrapperNextFundingTime = time.UnixMilli(1794441600000)

// wrapperDecimals returns decimals as text, so values held in different forms compare equal
func wrapperDecimals(values ...decimal.Decimal) []string {
	s := make([]string, len(values))
	for i := range values {
		s[i] = values[i].String()
	}
	return s
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must reject a spot asset")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must reject an unsupported futures asset")
	_, err = new(Exchange).GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetFuturesContractDetails must return the pair format error")

	ex := newWrapperTestExchange(t)
	t.Run(asset.USDTMarginedFutures.String(), func(t *testing.T) {
		t.Parallel()
		contracts, err := ex.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
		require.NoError(t, err, "GetFuturesContractDetails must not error")
		require.NotEmpty(t, contracts, "GetFuturesContractDetails must return contracts")
		if !mockTests {
			return
		}
		require.Len(t, contracts, 2, "GetFuturesContractDetails must return every recorded contract")
		c := contracts[0]
		assert.Equal(t, []string{"-0.003", "0.003", "0.000056"}, wrapperDecimals(c.FundingRateFloor, c.FundingRateCeiling, c.LatestRate.Rate), "GetFuturesContractDetails should take the funding rate limits and current rate")
		c.FundingRateFloor, c.FundingRateCeiling, c.LatestRate.Rate = decimal.Decimal{}, decimal.Decimal{}, decimal.Decimal{}
		assert.Equal(t, futures.Contract{
			Exchange:           ex.Name,
			Name:               currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter),
			Underlying:         currency.NewBTCUSDT(),
			Asset:              asset.USDTMarginedFutures,
			StartDate:          time.UnixMilli(1586275200000),
			IsActive:           true,
			Status:             "online",
			Type:               futures.Perpetual,
			SettlementType:     futures.Linear,
			SettlementCurrency: currency.USDT,
			MarginCurrency:     currency.USDT,
			LatestRate:         fundingrate.Rate{Time: wrapperNextFundingTime.Add(-8 * time.Hour)},
		}, c, "GetFuturesContractDetails should convert the USDT-M contract")
	})
	t.Run(asset.CoinMarginedFutures.String(), func(t *testing.T) {
		t.Parallel()
		contracts, err := ex.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
		require.NoError(t, err, "GetFuturesContractDetails must not error")
		require.NotEmpty(t, contracts, "GetFuturesContractDetails must return contracts")
		if !mockTests {
			return
		}
		require.Len(t, contracts, 3, "GetFuturesContractDetails must return every recorded contract")
		c := contracts[0]
		assert.Equal(t, "0.000009", c.LatestRate.Rate.String(), "GetFuturesContractDetails should take the current rate")
		c.LatestRate.Rate = decimal.Decimal{}
		bnb := currency.NewPairWithDelimiter("BNB", "USD", currency.DashDelimiter)
		assert.Equal(t, futures.Contract{
			Exchange:           ex.Name,
			Name:               bnb,
			Underlying:         bnb,
			Asset:              asset.CoinMarginedFutures,
			StartDate:          time.UnixMilli(1713175200000),
			IsActive:           true,
			Status:             "1",
			Type:               futures.Perpetual,
			SettlementType:     futures.Inverse,
			SettlementCurrency: currency.BNB,
			MarginCurrency:     currency.BNB,
			Multiplier:         10,
		}, c, "GetFuturesContractDetails should convert the coin-M contract, its multiplier its USD value")
		assert.True(t, contracts[1].LatestRate.Rate.IsZero(), "GetFuturesContractDetails should leave out the rate of a contract without one")
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/quote/contracts":      {`{"code":100500,"msg":"busy"}`, `{"code":0,"data":[{"symbol":"NODASH"}]}`, `{"code":0,"data":[]}`},
			"GET /openApi/swap/v2/quote/premiumIndex":   {`{"code":0,"data":[]}`, `{"code":100500,"msg":"busy"}`},
			"GET /openApi/cswap/v1/market/contracts":    {`{"code":100500,"msg":"busy"}`, `{"code":0,"data":[{"symbol":"NODASH"}]}`, `{"code":0,"data":[]}`},
			"GET /openApi/cswap/v1/market/premiumIndex": {`{"code":0,"data":[]}`, `{"code":100500,"msg":"busy"}`},
		})
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			_, err := ex.GetFuturesContractDetails(t.Context(), a)
			assert.ErrorIsf(t, err, errAPIResponse, "GetFuturesContractDetails should return the %s contracts request's error", a)
			contracts, err := ex.GetFuturesContractDetails(t.Context(), a)
			require.NoErrorf(t, err, "GetFuturesContractDetails must skip a %s symbol it cannot read", a)
			assert.Emptyf(t, contracts, "GetFuturesContractDetails should leave out a %s symbol it cannot read", a)
			_, err = ex.GetFuturesContractDetails(t.Context(), a)
			assert.ErrorIsf(t, err, errAPIResponse, "GetFuturesContractDetails should return the %s funding rate request's error", a)
		}
	})
}

func TestSwapContractStatusString(t *testing.T) {
	t.Parallel()
	for status, exp := range map[SwapContractStatus]string{
		SwapContractStatusOffline:                  "offline",
		SwapContractStatusOnline:                   "online",
		SwapContractStatusPreOnline:                "pre-online",
		SwapContractStatusForbiddenToOpenPositions: "forbidden to open positions",
		9: "9",
	} {
		assert.Equalf(t, exp, swapContractStatusString(status), "swapContractStatusString should name status %d", status)
	}
}

func TestSwapFundingPeriodStart(t *testing.T) {
	t.Parallel()
	r := &SwapMarkPriceAndFundingRateResponse{NextFundingTime: types.Time(wrapperNextFundingTime), FundingIntervalHours: 4}
	assert.Equal(t, wrapperNextFundingTime.Add(-4*time.Hour), swapFundingPeriodStart(r), "swapFundingPeriodStart should start the period an interval before the next funding time")
	r.FundingIntervalHours = 0
	assert.Equal(t, wrapperNextFundingTime, swapFundingPeriodStart(r), "swapFundingPeriodStart should start a period without an interval at the next funding time")
}

func TestSwapFundingInterval(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 8*time.Hour, swapFundingInterval(&SwapMarkPriceAndFundingRateResponse{FundingIntervalHours: 8}), "swapFundingInterval should return the interval in hours")
	longest := swapFundingInterval(&SwapMarkPriceAndFundingRateResponse{FundingIntervalHours: math.MaxUint64})
	assert.Positive(t, longest, "swapFundingInterval should not overflow an interval beyond any duration")
	assert.Equal(t, time.Duration(math.MaxInt64).Truncate(time.Hour), longest, "swapFundingInterval should clamp an interval to the longest duration in hours")
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		isPerpetual, err := e.IsPerpetualFutureCurrency(a, currency.NewBTCUSDT())
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", a)
		assert.Truef(t, isPerpetual, "IsPerpetualFutureCurrency should report every %s contract perpetual", a)
		_, err = e.IsPerpetualFutureCurrency(a, currency.EMPTYPAIR)
		assert.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "IsPerpetualFutureCurrency should reject an empty %s pair", a)
	}
	isPerpetual, err := e.IsPerpetualFutureCurrency(asset.Spot, currency.NewBTCUSDT())
	require.NoError(t, err, "IsPerpetualFutureCurrency must not error for spot")
	assert.False(t, isPerpetual, "IsPerpetualFutureCurrency should report a spot pair not perpetual")
	_, err = e.IsPerpetualFutureCurrency(asset.Futures, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, asset.ErrNotSupported, "IsPerpetualFutureCurrency should reject an unsupported asset")
}

func TestGetCollateralCurrencyForContract(t *testing.T) {
	t.Parallel()
	code, a, err := e.GetCollateralCurrencyForContract(asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.USDC))
	require.NoError(t, err, "GetCollateralCurrencyForContract must not error")
	assert.Equal(t, currency.USDC, code, "GetCollateralCurrencyForContract should margin a USDT-M contract in its settlement currency")
	assert.Equal(t, asset.USDTMarginedFutures, a, "GetCollateralCurrencyForContract should hold the collateral in the USDT-M account")
	_, _, err = e.GetCollateralCurrencyForContract(asset.Spot, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetCollateralCurrencyForContract should reject a spot pair")
}

func TestGetCurrencyForRealisedPNL(t *testing.T) {
	t.Parallel()
	code, a, err := e.GetCurrencyForRealisedPNL(asset.CoinMarginedFutures, currency.NewPairWithDelimiter("ETH", "USD", currency.DashDelimiter))
	require.NoError(t, err, "GetCurrencyForRealisedPNL must not error")
	assert.Equal(t, currency.ETH, code, "GetCurrencyForRealisedPNL should pay a coin-M contract's profit in its base coin")
	assert.Equal(t, asset.CoinMarginedFutures, a, "GetCurrencyForRealisedPNL should pay into the coin-M account")
	_, _, err = e.GetCurrencyForRealisedPNL(asset.Futures, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyForRealisedPNL should reject an unsupported asset")
}

func TestContractSettlement(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  currency.Code
	}{
		{asset.USDTMarginedFutures, currency.NewBTCUSDT(), currency.USDT},
		{asset.USDTMarginedFutures, currency.NewPair(currency.ETH, currency.USDC), currency.USDC},
		{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), currency.BTC},
	} {
		code, a, err := contractSettlement(tc.a, tc.pair)
		require.NoErrorf(t, err, "contractSettlement must not error for %s %s", tc.a, tc.pair)
		assert.Equalf(t, tc.exp, code, "contractSettlement should settle %s %s in %s", tc.a, tc.pair, tc.exp)
		assert.Equalf(t, tc.a, a, "contractSettlement should settle %s into its own account", tc.a)
	}
	_, _, err := contractSettlement(asset.USDTMarginedFutures, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "contractSettlement should reject an empty pair")
	_, _, err = contractSettlement(asset.Spot, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset, "contractSettlement should reject a spot pair")
	_, _, err = contractSettlement(asset.Futures, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, asset.ErrNotSupported, "contractSettlement should reject an unsupported asset")
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLatestFundingRates must reject an unsupported asset")

	ex := newWrapperTestExchange(t)
	for _, tc := range []struct {
		a           asset.Item
		periodStart time.Time
		rate        string
	}{
		{asset.USDTMarginedFutures, wrapperNextFundingTime.Add(-8 * time.Hour), "0.000056"},
		// Coin-M reports no funding interval, so the period's start is unknown
		{asset.CoinMarginedFutures, time.Time{}, "0.0001"},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			pair := assetToTradablePairMap[tc.a]
			started := time.Now()
			rates, err := ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: tc.a, Pair: pair, IncludePredictedRate: true})
			require.NoError(t, err, "GetLatestFundingRates must not error")
			require.Len(t, rates, 1, "GetLatestFundingRates must return the pair's rate")
			r := rates[0]
			assert.WithinRange(t, r.TimeChecked, started, time.Now(), "GetLatestFundingRates should stamp the time the rate was checked")
			if !mockTests {
				return
			}
			assert.Equal(t, []string{tc.rate, tc.rate}, wrapperDecimals(r.LatestRate.Rate, r.PredictedUpcomingRate.Rate), "GetLatestFundingRates should take the current rate as the latest and predicted rates")
			r.LatestRate.Rate, r.PredictedUpcomingRate.Rate, r.TimeChecked = decimal.Decimal{}, decimal.Decimal{}, time.Time{}
			assert.Equal(t, fundingrate.LatestRateResponse{
				Exchange:              ex.Name,
				Asset:                 tc.a,
				Pair:                  pair,
				LatestRate:            fundingrate.Rate{Time: tc.periodStart},
				PredictedUpcomingRate: fundingrate.Rate{Time: wrapperNextFundingTime},
				TimeOfNextRate:        wrapperNextFundingTime,
			}, r, "GetLatestFundingRates should date the latest rate by its period and the predicted rate by the next funding time")
		})
	}

	t.Run("every enabled pair", func(t *testing.T) {
		t.Parallel()
		for a, expSymbols := range map[asset.Item][]string{
			// Only BTC-USDT is enabled among the recorded USDT-M rates, and no recorded coin-M rate is enabled
			asset.USDTMarginedFutures: {"BTC-USDT"},
			asset.CoinMarginedFutures: {},
		} {
			rates, err := ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: a})
			require.NoErrorf(t, err, "GetLatestFundingRates must not error for %s", a)
			if !mockTests {
				assert.NotEmptyf(t, rates, "GetLatestFundingRates should return the enabled %s pairs' rates", a)
				continue
			}
			symbols := make([]string, 0, len(rates))
			for i := range rates {
				symbols = append(symbols, rates[i].Pair.String())
				assert.Truef(t, rates[i].PredictedUpcomingRate.Time.IsZero(), "GetLatestFundingRates should leave out a %s predicted rate not requested", a)
			}
			assert.Equalf(t, expSymbols, symbols, "GetLatestFundingRates should return the enabled %s pairs' rates", a)
		}
	})

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/quote/premiumIndex":   {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/cswap/v1/market/premiumIndex": {`{"code":100500,"msg":"busy"}`},
		})
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			_, err := ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: a})
			assert.ErrorIsf(t, err, errAPIResponse, "GetLatestFundingRates should return the %s request's error", a)
		}
	})

	t.Run("pairs never loaded", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/quote/premiumIndex":   {`{"code":0,"data":[{"symbol":"BTC-USDT","lastFundingRate":"0.0001"}]}`},
			"GET /openApi/cswap/v1/market/premiumIndex": {`{"code":0,"data":[{"symbol":"BTC-USD","lastFundingRate":"0.0001"}]}`},
		})
		ex.CurrencyPairs = currency.PairsManager{}
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			_, err := ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: a})
			assert.ErrorContainsf(t, err, "pair matcher is nil", "GetLatestFundingRates should return the %s pair lookup's error", a)
		}
	})
}

func TestFundingRatePair(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	requested := currency.NewPairWithDelimiter("XRP", "USDT", currency.DashDelimiter)
	pair, ok, err := ex.fundingRatePair(&fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: requested}, "ANY-USDT")
	require.NoError(t, err, "fundingRatePair must not error")
	assert.True(t, ok, "fundingRatePair should return a requested pair")
	assert.Equal(t, requested, pair, "fundingRatePair should return the pair requested")
	every := &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures}
	pair, ok, err = ex.fundingRatePair(every, usdtMarginedTradablePair.String())
	require.NoError(t, err, "fundingRatePair must not error")
	assert.True(t, ok, "fundingRatePair should return an enabled pair")
	assert.Equal(t, usdtMarginedTradablePair.String(), pair.String(), "fundingRatePair should return the symbol's pair")
	for _, symbol := range []string{"NOPE-USDT", "NODASH"} {
		_, ok, err = ex.fundingRatePair(every, symbol)
		require.NoErrorf(t, err, "fundingRatePair must not error for %q", symbol)
		assert.Falsef(t, ok, "fundingRatePair should leave out %q", symbol)
	}
	_, _, err = new(Exchange).fundingRatePair(every, "BTC-USDT")
	assert.Error(t, err, "fundingRatePair should return the error of a pair manager without pairs")
}

func TestLatestFundingRate(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	start, next, checked := time.UnixMilli(1), time.UnixMilli(2), time.UnixMilli(3)
	r := latestFundingRate("ex", &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, IncludePredictedRate: true}, pair, start, next, 0.0001, checked)
	assert.Equal(t, []string{"0.0001", "0.0001"}, wrapperDecimals(r.LatestRate.Rate, r.PredictedUpcomingRate.Rate), "latestFundingRate should take the rate as the latest and predicted rates")
	assert.Equal(t, start, r.LatestRate.Time, "latestFundingRate should date the latest rate by its period's start")
	assert.Equal(t, next, r.PredictedUpcomingRate.Time, "latestFundingRate should date the predicted rate by the next funding time")
	assert.Equal(t, next, r.TimeOfNextRate, "latestFundingRate should report the next funding time")
	assert.Equal(t, checked, r.TimeChecked, "latestFundingRate should report when the rate was checked")
	assert.Equal(t, "ex", r.Exchange, "latestFundingRate should name the exchange")
	r = latestFundingRate("ex", &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures}, pair, start, next, 0.0001, checked)
	assert.Equal(t, fundingrate.Rate{}, r.PredictedUpcomingRate, "latestFundingRate should leave out a predicted rate not requested")
}

// wrapperFundingRatePage returns a funding rate history reply holding a rate at each time, newest first as BingX sends
// them, or BingX's null data without any
func wrapperFundingRatePage(times ...time.Time) string {
	if len(times) == 0 {
		return `{"code":0,"msg":"","data":null}`
	}
	rows := make([]string, len(times))
	for i := range times {
		rows[len(times)-1-i] = `{"symbol":"BTC-USDT","fundingRate":"0.0001","fundingTime":` + wrapperMilliseconds(times[i]) + `}`
	}
	return `{"code":0,"data":[` + strings.Join(rows, ",") + `]}`
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates must reject a nil request")
	for _, tc := range []struct {
		name string
		r    *fundingrate.HistoricalRatesRequest
		err  error
	}{
		{"coin-M", &fundingrate.HistoricalRatesRequest{Asset: asset.CoinMarginedFutures}, common.ErrFunctionNotSupported},
		{"unsupported asset", &fundingrate.HistoricalRatesRequest{Asset: asset.Spot}, asset.ErrNotSupported},
		{"empty pair", &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures}, currency.ErrCurrencyPairEmpty},
		{"start after end", &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), StartDate: wrapperFixtureEnd, EndDate: wrapperFixtureStart}, common.ErrStartAfterEnd},
		{"payment currency", &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), StartDate: wrapperFixtureStart, EndDate: wrapperFixtureEnd, PaymentCurrency: currency.USDC}, common.ErrFunctionNotSupported},
	} {
		_, err := e.GetHistoricalFundingRates(t.Context(), tc.r)
		require.ErrorIsf(t, err, tc.err, "GetHistoricalFundingRates must reject a request with an invalid %s", tc.name)
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the recorded rates are a past window, which a live test cannot predict")
		}
		ex := newWrapperTestExchange(t)
		rates, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), StartDate: wrapperFixtureStart, EndDate: wrapperFixtureEnd, IncludePredictedRate: true})
		require.NoError(t, err, "GetHistoricalFundingRates must not error")
		require.Len(t, rates.FundingRates, 10, "GetHistoricalFundingRates must return every rate in the range")
		for i := range rates.FundingRates {
			assert.Equalf(t, wrapperFixtureStart.Add(time.Duration(i)*8*time.Hour), rates.FundingRates[i].Time.UTC(), "GetHistoricalFundingRates should return rate %d in ascending time", i)
		}
		assert.Equal(t, "0.000058", rates.FundingRates[3].Rate.String(), "GetHistoricalFundingRates should convert each rate")
		assert.Equal(t, []string{"0.000056", "0.000056"}, wrapperDecimals(rates.LatestRate.Rate, rates.PredictedUpcomingRate.Rate), "GetHistoricalFundingRates should take the current rate as the latest and predicted rates")
		assert.Equal(t, wrapperNextFundingTime.Add(-8*time.Hour), rates.LatestRate.Time, "GetHistoricalFundingRates should date the latest rate by its period")
		assert.Equal(t, wrapperNextFundingTime, rates.TimeOfNextRate, "GetHistoricalFundingRates should report the next funding time")
		assert.True(t, rates.PaymentCurrency.IsEmpty(), "GetHistoricalFundingRates should leave out payments not requested")
	})

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the recorded rates are a past window; the live case fetches the latest")
		}
		ex := newWrapperTestExchange(t)
		pair := assetToTradablePairMap[asset.USDTMarginedFutures]
		end := time.Now().UTC().Truncate(time.Hour)
		start := end.Add(-72 * time.Hour)
		rates, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: pair, StartDate: start, EndDate: end})
		require.NoError(t, err, "GetHistoricalFundingRates must not error")
		require.NotEmpty(t, rates.FundingRates, "GetHistoricalFundingRates must return the range's rates")
		assert.True(t, slices.IsSortedFunc(rates.FundingRates, func(a, b fundingrate.Rate) int { return a.Time.Compare(b.Time) }), "GetHistoricalFundingRates should return the rates in ascending time")
		for i := range rates.FundingRates {
			assert.WithinRangef(t, rates.FundingRates[i].Time, start, end, "GetHistoricalFundingRates should return rate %d within the range", i)
		}
		assert.False(t, rates.LatestRate.Time.IsZero(), "GetHistoricalFundingRates should report the latest rate")
	})

	start := time.Now().UTC().Truncate(time.Hour).Add(-72 * time.Hour)
	end := start.Add(24 * time.Hour)
	premium := `{"code":0,"data":{"symbol":"BTC-USDT","lastFundingRate":"0.0002","nextFundingTime":` + wrapperMilliseconds(end.Add(8*time.Hour)) + `,"fundingIntervalHours":8}}`
	t.Run("payments", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/quote/fundingRate":  {wrapperFundingRatePage(start.Add(8*time.Hour), start.Add(16*time.Hour), end), wrapperFundingRatePage()},
			"GET /openApi/swap/v2/quote/premiumIndex": {premium},
			"GET /openApi/swap/v2/user/income": {`{"code":0,"data":[
				{"symbol":"BTC-USDT","incomeType":"FUNDING_FEE","income":"-0.5","asset":"USDT","time":` + wrapperMilliseconds(start.Add(8*time.Hour+time.Second)) + `},
				{"symbol":"BTC-USDT","incomeType":"FUNDING_FEE","income":"0.25","asset":"USDT","time":` + wrapperMilliseconds(start.Add(16*time.Hour)) + `},
				{"symbol":"BTC-USDT","incomeType":"FUNDING_FEE","income":"9","asset":"USDT","time":` + wrapperMilliseconds(start.Add(20*time.Hour)) + `}]}`},
		})
		rates, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), StartDate: start, EndDate: end, IncludePayments: true})
		require.NoError(t, err, "GetHistoricalFundingRates must not error")
		require.Len(t, rates.FundingRates, 3, "GetHistoricalFundingRates must return every rate in the range")
		assert.Equal(t, []string{"-0.5", "0.25", "0"}, wrapperDecimals(rates.FundingRates[0].Payment, rates.FundingRates[1].Payment, rates.FundingRates[2].Payment), "GetHistoricalFundingRates should add each fee to the rate it was paid at, leaving out fees half an interval from every rate")
		assert.Equal(t, "-0.25", rates.PaymentSum.String(), "GetHistoricalFundingRates should sum the fees paid at the rates")
		assert.Equal(t, currency.USDT, rates.PaymentCurrency, "GetHistoricalFundingRates should report payments in the settlement currency")
		assert.True(t, rates.PredictedUpcomingRate.Time.IsZero(), "GetHistoricalFundingRates should leave out a predicted rate not requested")
		got := requests()
		require.Len(t, got, 4, "GetHistoricalFundingRates must request the rates back to the range's start, the current rate and the payments")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "currency": {"USDT"}, "incomeType": {"FUNDING_FEE"}, "startTime": {wrapperMilliseconds(start)}, "endTime": {wrapperMilliseconds(end)}, "limit": {"1000"}}, got[3].params, "GetHistoricalFundingRates should request the range's funding fees")
	})

	t.Run("payment retention", func(t *testing.T) {
		t.Parallel()
		old := time.Now().Add(-100 * 24 * time.Hour).Truncate(time.Hour)
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/quote/fundingRate":  {wrapperFundingRatePage(), wrapperFundingRatePage()},
			"GET /openApi/swap/v2/quote/premiumIndex": {premium, premium},
			"GET /openApi/swap/v2/user/income":        {`{"code":0,"data":[]}`},
		})
		r := &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), StartDate: old, EndDate: old.Add(11 * 24 * time.Hour), IncludePayments: true}
		_, err := ex.GetHistoricalFundingRates(t.Context(), r)
		require.ErrorIs(t, err, errFundingPaymentsNotKept, "GetHistoricalFundingRates must report payments older than BingX keeps")
		r.RespectHistoryLimits = true
		started := time.Now()
		_, err = ex.GetHistoricalFundingRates(t.Context(), r)
		require.NoError(t, err, "GetHistoricalFundingRates must not error when respecting the history limits")
		got := requests()
		require.Len(t, got, 5, "GetHistoricalFundingRates must request the payments BingX keeps")
		paymentsStart, err := strconv.ParseInt(got[4].params.Get("startTime"), 10, 64)
		require.NoError(t, err, "startTime must be a number")
		assert.WithinRange(t, time.UnixMilli(paymentsStart), started.Add(-fundingFlowRetention).Truncate(time.Millisecond), time.Now().Add(-fundingFlowRetention), "GetHistoricalFundingRates should request payments from the oldest BingX keeps")
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/quote/fundingRate":  {`{"code":100500,"msg":"busy"}`, wrapperFundingRatePage(), wrapperFundingRatePage(), wrapperFundingRatePage()},
			"GET /openApi/swap/v2/quote/premiumIndex": {`{"code":100500,"msg":"busy"}`, `{"code":0,"data":[]}`, premium},
			"GET /openApi/swap/v2/user/income":        {`{"code":100500,"msg":"busy"}`},
		})
		r := &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), StartDate: start, EndDate: end, IncludePayments: true}
		_, err := ex.GetHistoricalFundingRates(t.Context(), r)
		assert.ErrorIs(t, err, errAPIResponse, "GetHistoricalFundingRates should return the rates request's error")
		_, err = ex.GetHistoricalFundingRates(t.Context(), r)
		assert.ErrorIs(t, err, errAPIResponse, "GetHistoricalFundingRates should return the current rate request's error")
		_, err = ex.GetHistoricalFundingRates(t.Context(), r)
		assert.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates should report a pair without a current rate")
		_, err = ex.GetHistoricalFundingRates(t.Context(), r)
		assert.ErrorIs(t, err, errAPIResponse, "GetHistoricalFundingRates should return the payments request's error")
	})
}

func TestSwapFundingRates(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1700000000000).UTC().Truncate(time.Hour)
	full := make([]time.Time, fundingRateHistoryLimit)
	for i := range full {
		full[i] = start.Add(time.Duration(i+3) * time.Hour)
	}
	end := full[len(full)-1]
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/swap/v2/quote/fundingRate": {
			wrapperFundingRatePage(full...),
			// The next page repeats the oldest rate of the last
			wrapperFundingRatePage(start, start.Add(2*time.Hour), full[0]),
			`{"code":100500,"msg":"busy"}`,
		},
	})
	rates, err := ex.swapFundingRates(t.Context(), currency.NewBTCUSDT(), start, end)
	require.NoError(t, err, "swapFundingRates must not error")
	require.Len(t, rates, fundingRateHistoryLimit+2, "swapFundingRates must return every rate once")
	assert.Equal(t, start, rates[0].Time.UTC(), "swapFundingRates should return the rates in ascending time")
	assert.Equal(t, end, rates[len(rates)-1].Time.UTC(), "swapFundingRates should end at the range's last rate")
	got := requests()
	require.Len(t, got, 2, "swapFundingRates must page back until a page is not full")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "startTime": {wrapperMilliseconds(start)}, "endTime": {wrapperMilliseconds(end)}, "limit": {"1000"}}, got[0].params, "swapFundingRates should request the latest rates first")
	assert.Equal(t, wrapperMilliseconds(full[0].Add(-time.Millisecond)), got[1].params.Get("endTime"), "swapFundingRates should page back from before the oldest rate fetched")
	_, err = ex.swapFundingRates(t.Context(), currency.NewBTCUSDT(), start, end)
	assert.ErrorIs(t, err, errAPIResponse, "swapFundingRates should return a page's error")

	t.Run("pages back to the range's start", func(t *testing.T) {
		t.Parallel()
		at := func(hours ...int) []time.Time {
			times := make([]time.Time, len(hours))
			for i, h := range hours {
				times[i] = start.Add(time.Duration(h) * time.Hour)
			}
			return times
		}
		for _, tc := range []struct {
			name     string
			end      time.Time
			pages    [][]time.Time
			expected []time.Time
		}{
			{"short page then an older page", start.Add(12 * time.Hour), [][]time.Time{at(10, 11, 12), at(0, 1)}, at(0, 1, 10, 11, 12)},
			{"short page then an empty page", start.Add(12 * time.Hour), [][]time.Time{at(10, 12), {}}, at(10, 12)},
			{"four-hourly settlements", start.Add(8 * time.Hour), [][]time.Time{at(4, 8), at(0)}, at(0, 4, 8)},
			{"three pages ending with a single rate", start.Add(5 * time.Hour), [][]time.Time{at(4, 5), at(2, 3), at(0)}, at(0, 2, 3, 4, 5)},
			{"a single rate before older pages", start.Add(5 * time.Hour), [][]time.Time{at(5), at(2, 3), at(0)}, at(0, 2, 3, 5)},
			{"a page that cannot move the cursor", start.Add(2 * time.Hour), [][]time.Time{{start.Add(2*time.Hour + time.Millisecond)}}, []time.Time{start.Add(2*time.Hour + time.Millisecond)}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				replies := make([]string, len(tc.pages))
				for i := range tc.pages {
					replies[i] = wrapperFundingRatePage(tc.pages[i]...)
				}
				ex, requests := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v2/quote/fundingRate": replies})
				rates, err := ex.swapFundingRates(t.Context(), currency.NewBTCUSDT(), start, tc.end)
				require.NoError(t, err, "swapFundingRates must not error")
				got := make([]time.Time, len(rates))
				for i := range rates {
					got[i] = rates[i].Time.UTC()
				}
				assert.Equal(t, tc.expected, got, "swapFundingRates should return each rate of every page once, in ascending time")
				sent := requests()
				require.Len(t, sent, len(tc.pages), "swapFundingRates must request each page once")
				for i := 1; i < len(sent); i++ {
					oldest := slices.MinFunc(tc.pages[i-1], func(a, b time.Time) int { return a.Compare(b) })
					assert.Equalf(t, wrapperMilliseconds(oldest.Add(-time.Millisecond)), sent[i].params.Get("endTime"), "request %d should page back from before the oldest rate fetched", i)
				}
			})
		}
	})

	t.Run("a later page's error", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, body string
			err        error
		}{
			{"an error", `{"code":100500,"msg":"busy"}`, errAPIResponse},
			{"a null body", `null`, common.ErrNoResponse},
		} {
			ex, requests := wrapperTestServer(t, map[string][]string{
				"GET /openApi/swap/v2/quote/fundingRate": {wrapperFundingRatePage(start.Add(8*time.Hour), start.Add(16*time.Hour)), tc.body},
			})
			_, err := ex.swapFundingRates(t.Context(), currency.NewBTCUSDT(), start, start.Add(16*time.Hour))
			assert.ErrorIsf(t, err, tc.err, "swapFundingRates should return a later page's error rather than the rates before it, for %s", tc.name)
			assert.Lenf(t, requests(), 2, "swapFundingRates should stop paging at %s", tc.name)
		}
	})

	t.Run("no progress", func(t *testing.T) {
		t.Parallel()
		times := make([]time.Time, fundingRateHistoryLimit)
		for i := range times {
			times[i] = start
		}
		ex, requests := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v2/quote/fundingRate": {wrapperFundingRatePage(times...)}})
		rates, err := ex.swapFundingRates(t.Context(), currency.NewBTCUSDT(), start, start)
		require.NoError(t, err, "swapFundingRates must not error")
		assert.Len(t, rates, 1, "swapFundingRates should return a repeated rate once")
		assert.Len(t, requests(), 1, "swapFundingRates should stop paging once a page reaches the range's start")
		for i := range times {
			times[i] = start.Add(2 * time.Hour)
		}
		ex, requests = wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v2/quote/fundingRate": {wrapperFundingRatePage(times...)}})
		_, err = ex.swapFundingRates(t.Context(), currency.NewBTCUSDT(), start, start.Add(time.Hour))
		require.NoError(t, err, "swapFundingRates must not error")
		assert.Len(t, requests(), 1, "swapFundingRates should stop paging once a page holds no rate before the page's end")
	})
}

func TestAddFundingPayment(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1700000000000)
	interval := 8 * time.Hour
	resp := &fundingrate.HistoricalRates{FundingRates: []fundingrate.Rate{{Time: start}, {Time: start.Add(interval)}, {Time: start.Add(2 * interval)}}}
	addFundingPayment(resp, start.Add(interval+time.Second), decimal.NewFromInt(2), interval)
	addFundingPayment(resp, start.Add(-time.Minute), decimal.NewFromInt(-1), interval)
	addFundingPayment(resp, start.Add(2*interval+time.Hour), decimal.NewFromInt(3), interval)
	addFundingPayment(resp, start.Add(interval/2), decimal.NewFromInt(100), interval)
	addFundingPayment(resp, start.Add(-interval), decimal.NewFromInt(100), interval)
	assert.Equal(t, []string{"-1", "2", "3"}, wrapperDecimals(resp.FundingRates[0].Payment, resp.FundingRates[1].Payment, resp.FundingRates[2].Payment), "addFundingPayment should add each fee to the nearest rate within half an interval")
	assert.Equal(t, "4", resp.PaymentSum.String(), "addFundingPayment should sum the fees added")
	empty := &fundingrate.HistoricalRates{}
	addFundingPayment(empty, start, decimal.NewFromInt(1), interval)
	assert.True(t, empty.PaymentSum.IsZero(), "addFundingPayment should add nothing without rates")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	btcUSD := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	usdtMargined := key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.USDTMarginedFutures}
	coinMargined := key.PairAsset{Base: currency.BTC.Item, Quote: currency.USD.Item, Asset: asset.CoinMarginedFutures}
	_, err := e.GetOpenInterest(t.Context(), usdtMargined, key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest must reject an unsupported asset")

	ex := newWrapperTestExchange(t)
	t.Run("pairs", func(t *testing.T) {
		t.Parallel()
		// Copies, as live tests replace them while the other subtests read them
		usdtMargined, coinMargined := usdtMargined, coinMargined
		if !mockTests {
			usdtMargined = key.PairAsset{Base: usdtMarginedTradablePair.Base.Item, Quote: usdtMarginedTradablePair.Quote.Item, Asset: asset.USDTMarginedFutures}
			coinMargined = key.PairAsset{Base: coinMarginedTradablePair.Base.Item, Quote: coinMarginedTradablePair.Quote.Item, Asset: asset.CoinMarginedFutures}
		}
		resp, err := ex.GetOpenInterest(t.Context(), usdtMargined, coinMargined)
		require.NoError(t, err, "GetOpenInterest must not error")
		require.Len(t, resp, 2, "GetOpenInterest must return each pair's open interest")
		if !mockTests {
			return
		}
		assert.Equal(t, []futures.OpenInterest{
			{Key: key.NewExchangeAssetPair(ex.Name, asset.USDTMarginedFutures, usdtMargined.Pair()), OpenInterest: 634048539.6},
			{Key: key.NewExchangeAssetPair(ex.Name, asset.CoinMarginedFutures, btcUSD), OpenInterest: 698.8058},
		}, resp, "GetOpenInterest should return the USDT-M value and the coin-M base coin amount")
	})
	t.Run("enabled pairs", func(t *testing.T) {
		t.Parallel()
		resp, err := ex.GetOpenInterest(t.Context())
		require.NoError(t, err, "GetOpenInterest must not error")
		assert.NotEmpty(t, resp, "GetOpenInterest should return the enabled pairs' open interest")
	})

	t.Run("several coin-M pairs", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/cswap/v1/market/openInterest": {
				`{"code":0,"data":[{"symbol":"BTC-USD","openInterest":"698.8058"},{"symbol":"ETH-USD","openInterest":"12"}]}`,
				`{"code":0,"data":[{"symbol":"BTC-USD","openInterest":"698.8058"}]}`,
				`{"code":100500,"msg":"busy"}`,
				`{"code":0,"data":[]}`,
				`{"code":100500,"msg":"busy"}`,
			},
			"GET /openApi/swap/v2/quote/openInterest": {`{"code":100500,"msg":"busy"}`},
		})
		eth := key.PairAsset{Base: currency.ETH.Item, Quote: currency.USD.Item, Asset: asset.CoinMarginedFutures}
		resp, err := ex.GetOpenInterest(t.Context(), coinMargined, eth)
		require.NoError(t, err, "GetOpenInterest must not error")
		require.Len(t, resp, 2, "GetOpenInterest must return each pair's open interest")
		assert.Equal(t, 12.0, resp[1].OpenInterest, "GetOpenInterest should find each pair in the reply for every symbol")
		assert.Empty(t, requests()[0].params, "GetOpenInterest should request every symbol's open interest at once")
		_, err = ex.GetOpenInterest(t.Context(), coinMargined, eth)
		assert.ErrorIs(t, err, currency.ErrPairNotFound, "GetOpenInterest should report a pair the reply leaves out")
		_, err = ex.GetOpenInterest(t.Context(), coinMargined, eth)
		assert.ErrorIs(t, err, errAPIResponse, "GetOpenInterest should return the request for every symbol's error")
		_, err = ex.GetOpenInterest(t.Context(), coinMargined)
		assert.ErrorIs(t, err, currency.ErrPairNotFound, "GetOpenInterest should report a pair without open interest")
		_, err = ex.GetOpenInterest(t.Context(), coinMargined)
		assert.ErrorIs(t, err, errAPIResponse, "GetOpenInterest should return a coin-M request's error")
		_, err = ex.GetOpenInterest(t.Context(), usdtMargined)
		assert.ErrorIs(t, err, errAPIResponse, "GetOpenInterest should return a USDT-M request's error")
	})

	t.Run("disabled asset", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/cswap/v1/market/openInterest": {`{"code":0,"data":[{"symbol":"BTC-USD","openInterest":"698.8058"}]}`},
		})
		require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
		resp, err := ex.GetOpenInterest(t.Context())
		require.NoError(t, err, "GetOpenInterest must not error with a futures asset disabled")
		assert.Equal(t, []futures.OpenInterest{{Key: key.NewExchangeAssetPair(ex.Name, asset.CoinMarginedFutures, btcUSD), OpenInterest: 698.8058}}, resp, "GetOpenInterest should return the enabled asset's enabled pairs alone")
		got := requests()
		require.Len(t, got, 1, "GetOpenInterest must request the enabled asset's pair alone")
		assert.Equal(t, url.Values{"symbol": {"BTC-USD"}}, got[0].params, "GetOpenInterest should request the enabled coin-M pair")
	})

	t.Run("pair store missing", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		ex.CurrencyPairs.Delete(asset.USDTMarginedFutures)
		_, err := ex.GetOpenInterest(t.Context())
		assert.ErrorIs(t, err, currency.ErrAssetNotFound, "GetOpenInterest should return the enabled pairs lookup's error")
	})

	t.Run("coin-M symbol formatting error", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetOpenInterest(t.Context(), coinMargined)
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetOpenInterest should return the coin-M symbol formatting error")
	})
}

func TestLeveragePositionSides(t *testing.T) {
	t.Parallel()
	hedge, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v1/positionSide/dual": {wrapperHedgeModeReply}})
	oneWay, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v1/positionSide/dual": {wrapperOneWayModeReply}})
	for _, tc := range []struct {
		ex   *Exchange
		a    asset.Item
		side order.Side
		exp  []PositionSide
	}{
		{hedge, asset.USDTMarginedFutures, order.Long, []PositionSide{PositionSideLong}},
		{hedge, asset.USDTMarginedFutures, order.Sell, []PositionSide{PositionSideShort}},
		{hedge, asset.USDTMarginedFutures, order.UnknownSide, []PositionSide{PositionSideLong, PositionSideShort}},
		{oneWay, asset.USDTMarginedFutures, order.Long, []PositionSide{PositionSideBoth}},
		{oneWay, asset.CoinMarginedFutures, order.AnySide, []PositionSide{PositionSideLong, PositionSideShort}},
		{oneWay, asset.CoinMarginedFutures, order.Short, []PositionSide{PositionSideShort}},
	} {
		sides, err := tc.ex.leveragePositionSides(t.Context(), tc.a, tc.side)
		require.NoErrorf(t, err, "leveragePositionSides must not error for %s %s", tc.a, tc.side)
		assert.Equalf(t, tc.exp, sides, "leveragePositionSides should set %s %s leverage on %v", tc.a, tc.side, tc.exp)
	}
	failing, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v1/positionSide/dual": {`{"code":100500,"msg":"busy"}`}})
	_, err := failing.leveragePositionSides(t.Context(), asset.USDTMarginedFutures, order.Long)
	assert.ErrorIs(t, err, errAPIResponse, "leveragePositionSides should return the position mode lookup's error")
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.Spot, currency.NewBTCUSDT(), margin.Isolated, 2, order.Long), asset.ErrNotSupported, "SetLeverage must reject an unsupported asset")
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.EMPTYPAIR, margin.Isolated, 2, order.Long), currency.ErrCurrencyPairEmpty, "SetLeverage must reject an empty pair")
	for _, amount := range []float64{0, 0.5, 2.5, math.NaN(), math.Inf(1), math.MaxUint32 + 1} {
		require.ErrorIsf(t, e.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Isolated, amount, order.Long), errInvalidLeverage, "SetLeverage must reject leverage %v", amount)
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		if !mockTests {
			t.Skip("the recorded leverage changes are documented examples on other pairs")
		}
		require.NoError(t, ex.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter), margin.Multi, 8, order.Short), "SetLeverage must not error for USDT-M")
		require.NoError(t, ex.SetLeverage(t.Context(), asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), margin.Isolated, 4, order.Long), "SetLeverage must not error for coin-M")
	})

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the recorded leverage changes are documented examples; the live case sets the live pairs' current leverage")
		}
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		for _, a := range wrapperPerpetualAssets {
			leverage, err := ex.GetLeverage(t.Context(), a, assetToTradablePairMap[a], margin.Unset, order.Long)
			require.NoErrorf(t, err, "GetLeverage must not error for %s", a)
			require.NoErrorf(t, ex.SetLeverage(t.Context(), a, assetToTradablePairMap[a], margin.Unset, leverage, order.Long), "SetLeverage must not error for %s", a)
		}
	})

	t.Run("sides", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v1/positionSide/dual": {wrapperOneWayModeReply},
			"POST /openApi/swap/v2/trade/leverage":   {`{"code":0,"data":{"leverage":3,"symbol":"BTC-USDT"}}`, `{"code":100500,"msg":"busy"}`},
			"POST /openApi/cswap/v1/trade/leverage":  {`{"code":0,"data":{"symbol":"BTC-USD","longLeverage":3,"shortLeverage":3}}`, `{"code":0,"data":{"symbol":"BTC-USD","longLeverage":3,"shortLeverage":3}}`},
		})
		require.NoError(t, ex.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 3, order.Long), "SetLeverage must not error")
		require.NoError(t, ex.SetLeverage(t.Context(), asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), margin.Multi, 3, order.UnknownSide), "SetLeverage must not error")
		got := requests()
		require.Len(t, got, 4, "SetLeverage must look the mode up and set each side's leverage")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "side": {"BOTH"}, "leverage": {"3"}}, got[1].params, "SetLeverage should set a one-way account's leverage on BOTH")
		assert.Equal(t, "LONG", got[2].params.Get("side"), "SetLeverage should set the long side's leverage without a direction")
		assert.Equal(t, "SHORT", got[3].params.Get("side"), "SetLeverage should set the short side's leverage without a direction")
		assert.ErrorIs(t, ex.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 3, order.Long), errAPIResponse, "SetLeverage should return the request's error")
		failing, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v1/positionSide/dual": {`{"code":100500,"msg":"busy"}`}})
		assert.ErrorIs(t, failing.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 3, order.Long), errAPIResponse, "SetLeverage should return the position mode lookup's error")
	})

	t.Run("position mode rejection", func(t *testing.T) {
		t.Parallel()
		rejection := `{"code":109400,"msg":"In the one-way mode, the 'PositionSide' field can only be set to BOTH."}`
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v1/positionSide/dual": {wrapperHedgeModeReply, wrapperOneWayModeReply, wrapperOneWayModeReply},
			"POST /openApi/swap/v2/trade/leverage":   {rejection, `{"code":0,"data":{"leverage":3,"symbol":"BTC-USDT"}}`, rejection, rejection, `{"code":101204,"msg":"leverage too high"}`},
		})
		require.NoError(t, ex.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 3, order.Long), "SetLeverage must not error")
		got := requests()
		require.Len(t, got, 4, "SetLeverage must look the mode up again and set the leverage once more")
		assert.Equal(t, "LONG", got[1].params.Get("side"), "SetLeverage should first set the leverage for the cached hedge mode")
		assert.Equal(t, "BOTH", got[3].params.Get("side"), "SetLeverage should set the leverage again for the mode looked up")
		assert.ErrorIs(t, ex.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 3, order.Long), errAPIResponse, "SetLeverage should return a second refusal")
		assert.Len(t, requests(), 7, "SetLeverage should set the leverage twice at most")
		assert.ErrorIs(t, ex.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 3, order.Long), errAPIResponse, "SetLeverage should return another refusal")
		assert.Len(t, requests(), 8, "SetLeverage should not set the leverage again after a refusal for another reason")
	})

	t.Run("credentials changing during the change", func(t *testing.T) {
		t.Parallel()
		rejection := `{"code":109400,"msg":"In the one-way mode, the 'PositionSide' field can only be set to BOTH."}`
		for _, tc := range []struct {
			name     string
			a        asset.Item
			pair     currency.Pair
			routes   map[string][]string
			requests int
		}{
			{
				name: "USDT-M", a: asset.USDTMarginedFutures, pair: currency.NewBTCUSDT(),
				routes: map[string][]string{
					"GET /openApi/swap/v1/positionSide/dual": {wrapperCredentialsChange + wrapperHedgeModeReply, wrapperOneWayModeReply},
					"POST /openApi/swap/v2/trade/leverage":   {wrapperCredentialsChange + rejection, `{"code":0,"data":{"leverage":3,"symbol":"BTC-USDT"}}`},
				},
				requests: 4,
			},
			{
				name: "coin-M", a: asset.CoinMarginedFutures, pair: currency.NewBTCUSD(),
				routes: map[string][]string{
					"POST /openApi/cswap/v1/trade/leverage": {
						wrapperCredentialsChange + `{"code":0,"data":{"symbol":"BTC-USD","longLeverage":3,"shortLeverage":2}}`,
						`{"code":0,"data":{"symbol":"BTC-USD","longLeverage":3,"shortLeverage":3}}`,
					},
				},
				requests: 2,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, requests := wrapperTestServer(t, tc.routes)
				require.NoError(t, ex.SetLeverage(t.Context(), tc.a, tc.pair, margin.Multi, 3, order.UnknownSide), "SetLeverage must not error")
				got := requests()
				require.Lenf(t, got, tc.requests, "SetLeverage must send each %s request", tc.name)
				for _, r := range got {
					assert.Equalf(t, testAPIKey, r.apiKey, "SetLeverage should send %s %s for the account it started with", r.method, r.path)
				}
			})
		}
	})

	t.Run("without credentials", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{})
		ex.SetCredentials(&accounts.Credentials{})
		assert.ErrorIs(t, ex.SetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.Multi, 3, order.Long), exchange.ErrCredentialsAreEmpty, "SetLeverage should return the credentials error")
		assert.Empty(t, requests(), "SetLeverage should send nothing without credentials")
	})
}

func TestSetPerpetualLeverage(t *testing.T) {
	t.Parallel()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/swap/v1/positionSide/dual": {wrapperHedgeModeReply},
		"POST /openApi/swap/v2/trade/leverage":   {`{"code":0,"data":{"leverage":4,"symbol":"BTC-USDT"}}`, `{"code":100500,"msg":"busy"}`},
		"POST /openApi/cswap/v1/trade/leverage":  {`{"code":0,"data":{"symbol":"BTC-USD","longLeverage":4,"shortLeverage":3}}`, `{"code":0,"data":{"symbol":"BTC-USD","longLeverage":4,"shortLeverage":4}}`},
	})
	require.NoError(t, ex.setPerpetualLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), 4, order.Short), "setPerpetualLeverage must not error")
	require.NoError(t, ex.setPerpetualLeverage(t.Context(), asset.CoinMarginedFutures, currency.NewBTCUSD(), 4, order.UnknownSide), "setPerpetualLeverage must not error")
	got := requests()
	require.Len(t, got, 4, "setPerpetualLeverage must look the USDT-M mode up and set each side's leverage")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "side": {"SHORT"}, "leverage": {"4"}}, got[1].params, "setPerpetualLeverage should set the short side's leverage in hedge mode")
	assert.Equal(t, "LONG", got[2].params.Get("side"), "setPerpetualLeverage should set the long side's leverage without a direction")
	assert.Equal(t, "SHORT", got[3].params.Get("side"), "setPerpetualLeverage should set the short side's leverage without a direction")
	assert.ErrorIs(t, ex.setPerpetualLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), 4, order.Short), errAPIResponse, "setPerpetualLeverage should return the request's error")
	failing, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v1/positionSide/dual": {`{"code":100500,"msg":"busy"}`}})
	assert.ErrorIs(t, failing.setPerpetualLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), 4, order.Short), errAPIResponse, "setPerpetualLeverage should return the position mode lookup's error")
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverage(t.Context(), asset.Spot, currency.NewBTCUSDT(), margin.Isolated, order.Long)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLeverage must reject an unsupported asset")
	_, err = e.GetLeverage(t.Context(), asset.CoinMarginedFutures, currency.EMPTYPAIR, margin.Isolated, order.Long)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLeverage must reject an empty pair")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		if !mockTests {
			for a, p := range map[asset.Item]currency.Pair{asset.USDTMarginedFutures: usdtMarginedTradablePair, asset.CoinMarginedFutures: coinMarginedTradablePair} {
				leverage, err := ex.GetLeverage(t.Context(), a, p, margin.Multi, order.Long)
				require.NoErrorf(t, err, "GetLeverage must not error for %s", a)
				assert.Positivef(t, leverage, "GetLeverage should return the %s leverage", a)
			}
			return
		}
		leverage, err := ex.GetLeverage(t.Context(), asset.USDTMarginedFutures, currency.NewPairWithDelimiter("BCH", "USDT", currency.DashDelimiter), margin.Multi, order.UnknownSide)
		require.NoError(t, err, "GetLeverage must not error")
		assert.Equal(t, 50.0, leverage, "GetLeverage should return the leverage both sides share")
		btcUSD := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
		for side, exp := range map[order.Side]float64{order.Long: 4, order.Sell: 8} {
			leverage, err = ex.GetLeverage(t.Context(), asset.CoinMarginedFutures, btcUSD, margin.Multi, side)
			require.NoErrorf(t, err, "GetLeverage must not error for %s", side)
			assert.Equalf(t, exp, leverage, "GetLeverage should return the %s side's leverage", side)
		}
		_, err = ex.GetLeverage(t.Context(), asset.CoinMarginedFutures, btcUSD, margin.Multi, order.UnknownSide)
		assert.ErrorIs(t, err, errPositionDirectionRequired, "GetLeverage should require a direction while the sides' leverage differs")
	})

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/leverage":  {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/cswap/v1/trade/leverage": {`{"code":100500,"msg":"busy"}`},
		})
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			_, err := ex.GetLeverage(t.Context(), a, currency.NewBTCUSDT(), margin.Multi, order.Long)
			assert.ErrorIsf(t, err, errAPIResponse, "GetLeverage should return the %s request's error", a)
		}
	})
}

func TestFormatMarginType(t *testing.T) {
	t.Parallel()
	for mt, exp := range map[margin.Type]MarginType{margin.Isolated: MarginTypeIsolated, margin.Multi: MarginTypeCrossed} {
		got, err := formatMarginType(mt)
		require.NoErrorf(t, err, "formatMarginType must not error for %s", mt)
		assert.Equalf(t, exp, got, "formatMarginType should format %s", mt)
	}
	for _, mt := range []margin.Type{margin.Unset, margin.SpotIsolated, margin.NoMargin} {
		_, err := formatMarginType(mt)
		assert.ErrorIsf(t, err, margin.ErrInvalidMarginType, "formatMarginType should reject %s", mt)
	}
}

func TestConvertMarginType(t *testing.T) {
	t.Parallel()
	for mt, exp := range map[MarginType]margin.Type{MarginTypeIsolated: margin.Isolated, MarginTypeSeparateIsolated: margin.Isolated, MarginTypeCrossed: margin.Multi} {
		got, err := convertMarginType(mt)
		require.NoErrorf(t, err, "convertMarginType must not error for %s", mt)
		assert.Equalf(t, exp, got, "convertMarginType should convert %s", mt)
	}
	got, err := convertMarginType("CROSS")
	assert.ErrorIs(t, err, margin.ErrInvalidMarginType, "convertMarginType should reject an unknown margin type")
	assert.Equal(t, margin.Unset, got, "convertMarginType should return no margin type for an unknown one")
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.SetMarginType(t.Context(), asset.Spot, currency.NewBTCUSDT(), margin.Isolated), asset.ErrNotSupported, "SetMarginType must reject an unsupported asset")
	require.ErrorIs(t, e.SetMarginType(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT(), margin.SpotIsolated), margin.ErrInvalidMarginType, "SetMarginType must reject a margin type BingX does not take")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		if !mockTests {
			t.Skip("the recorded margin type changes are documented examples on other pairs")
		}
		require.NoError(t, ex.SetMarginType(t.Context(), asset.USDTMarginedFutures, currency.NewPairWithDelimiter("MINA", "USDT", currency.DashDelimiter), margin.Multi), "SetMarginType must not error for USDT-M")
		require.NoError(t, ex.SetMarginType(t.Context(), asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), margin.Isolated), "SetMarginType must not error for coin-M")
	})

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the recorded margin type changes are documented examples; the live case switches the live pairs' margin type and back")
		}
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		for _, a := range wrapperPerpetualAssets {
			current := wrapperLiveMarginType(t, ex, a)
			other := margin.Isolated
			if current == margin.Isolated {
				other = margin.Multi
			}
			require.NoErrorf(t, ex.SetMarginType(t.Context(), a, assetToTradablePairMap[a], other), "SetMarginType must not error for %s %s", a, other)
			assert.Equalf(t, other, wrapperLiveMarginType(t, ex, a), "SetMarginType should set the %s margin type", a)
			require.NoErrorf(t, ex.SetMarginType(t.Context(), a, assetToTradablePairMap[a], current), "SetMarginType must not error for %s %s", a, current)
		}
	})

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"POST /openApi/swap/v2/trade/marginType":  {`{"code":100500,"msg":"busy"}`},
			"POST /openApi/cswap/v1/trade/marginType": {`{"code":100500,"msg":"busy"}`},
		})
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			assert.ErrorIsf(t, ex.SetMarginType(t.Context(), a, currency.NewBTCUSDT(), margin.Isolated), errAPIResponse, "SetMarginType should return the %s request's error", a)
		}
	})
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangePositionMargin must reject a nil request")
	btcUSD := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	for _, tc := range []struct {
		name string
		req  *margin.PositionChangeRequest
		err  error
	}{
		{"asset", &margin.PositionChangeRequest{Asset: asset.Spot, Pair: currency.NewBTCUSDT()}, asset.ErrNotSupported},
		{"pair", &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures}, currency.ErrCurrencyPairEmpty},
		{"margin type", &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), MarginType: margin.Multi, OriginalAllocatedMargin: 1, NewAllocatedMargin: 2}, margin.ErrMarginTypeUnsupported},
		{"new margin", &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OriginalAllocatedMargin: 1}, margin.ErrNewAllocatedMarginRequired},
		{"original margin", &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), NewAllocatedMargin: 1}, margin.ErrOriginalPositionMarginRequired},
		{"NaN margin", &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OriginalAllocatedMargin: 1, NewAllocatedMargin: math.NaN()}, errInvalidMarginAmount},
		{"infinite margin", &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OriginalAllocatedMargin: math.Inf(-1), NewAllocatedMargin: 1}, errInvalidMarginAmount},
		{"unchanged margin", &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OriginalAllocatedMargin: 1.1, NewAllocatedMargin: 1.1}, errMarginChangeRequired},
		{"coin-M side", &margin.PositionChangeRequest{Asset: asset.CoinMarginedFutures, Pair: btcUSD, OriginalAllocatedMargin: 1, NewAllocatedMargin: 2}, errPositionSideRequired},
	} {
		_, err := e.ChangePositionMargin(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "ChangePositionMargin must reject a request with an invalid %s", tc.name)
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		if !mockTests {
			t.Skip("the recorded margin changes are documented examples, which need an isolated position")
		}
		for _, req := range []*margin.PositionChangeRequest{
			{Exchange: ex.Name, Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), MarginType: margin.Isolated, OriginalAllocatedMargin: 10, NewAllocatedMargin: 13, MarginSide: "long"},
			{Exchange: ex.Name, Asset: asset.CoinMarginedFutures, Pair: btcUSD, MarginType: margin.Isolated, OriginalAllocatedMargin: 1, NewAllocatedMargin: 1.01, MarginSide: "LONG"},
		} {
			resp, err := ex.ChangePositionMargin(t.Context(), req)
			require.NoErrorf(t, err, "ChangePositionMargin must not error for %s", req.Asset)
			assert.Equalf(t, &margin.PositionChangeResponse{Exchange: ex.Name, Pair: req.Pair, Asset: req.Asset, MarginType: margin.Isolated, AllocatedMargin: req.NewAllocatedMargin}, resp, "ChangePositionMargin should report the %s position's new margin", req.Asset)
		}
	})

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the recorded margin changes are documented examples; the live case adds margin to the live pairs' long position, if any")
		}
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		for _, tc := range []struct {
			a      asset.Item
			margin float64
		}{
			{asset.USDTMarginedFutures, 10.1},
			{asset.CoinMarginedFutures, 1.0001},
		} {
			sides, err := ex.leveragePositionSides(t.Context(), tc.a, order.Long)
			require.NoErrorf(t, err, "leveragePositionSides must not error for %s", tc.a)
			resp, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Exchange: ex.Name, Asset: tc.a, Pair: assetToTradablePairMap[tc.a], MarginType: margin.Isolated, OriginalAllocatedMargin: math.Trunc(tc.margin), NewAllocatedMargin: tc.margin, MarginSide: string(sides[0])})
			if err != nil {
				requireAPIErrorCode(t, err, livePositionStateCodes...)
				continue
			}
			assert.Equalf(t, tc.margin, resp.AllocatedMargin, "ChangePositionMargin should report the %s position's new margin", tc.a)
		}
	})

	t.Run("decrease", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"POST /openApi/swap/v2/trade/positionMargin":  {`{"code":0,"msg":"","amount":0.3,"type":2}`, `{"code":100500,"msg":"busy"}`},
			"POST /openApi/cswap/v1/trade/positionMargin": {`{"code":0,"msg":""}`},
		})
		_, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OriginalAllocatedMargin: 0.4, NewAllocatedMargin: 0.1})
		require.NoError(t, err, "ChangePositionMargin must not error")
		_, err = ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.CoinMarginedFutures, Pair: btcUSD, OriginalAllocatedMargin: 0.4, NewAllocatedMargin: 0.1, MarginSide: "Short"})
		require.NoError(t, err, "ChangePositionMargin must not error")
		got := requests()
		require.Len(t, got, 2, "ChangePositionMargin must make a request for each change")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "amount": {"0.3"}, "type": {"2"}}, got[0].params, "ChangePositionMargin should remove the exact difference from a one-way position")
		assert.Equal(t, url.Values{"symbol": {"BTC-USD"}, "amount": {"0.3"}, "type": {"Decrease"}, "positionSide": {"SHORT"}}, got[1].params, "ChangePositionMargin should remove the exact difference from a coin-M position side")
		_, err = ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OriginalAllocatedMargin: 1, NewAllocatedMargin: 2})
		assert.ErrorIs(t, err, errAPIResponse, "ChangePositionMargin should return the request's error")
	})
}

func TestSummaryPositionSide(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		direction order.Side
		sides     []PositionSide
		exp       PositionSide
	}{
		{order.Long, nil, PositionSideLong},
		{order.Sell, []PositionSide{PositionSideLong}, PositionSideShort},
		{order.UnknownSide, []PositionSide{PositionSideBoth}, PositionSideBoth},
		{order.UnknownSide, nil, PositionSideBoth},
		{order.AnySide, []PositionSide{PositionSideLong}, PositionSideLong},
		{order.UnknownSide, []PositionSide{PositionSideShort}, PositionSideShort},
	} {
		side, err := summaryPositionSide(tc.direction, tc.sides)
		require.NoErrorf(t, err, "summaryPositionSide must not error for %s %v", tc.direction, tc.sides)
		assert.Equalf(t, tc.exp, side, "summaryPositionSide should summarise %s with sides %v open on %s", tc.direction, tc.sides, tc.exp)
	}
	_, err := summaryPositionSide(order.UnknownSide, []PositionSide{PositionSideLong, PositionSideShort})
	assert.ErrorIs(t, err, errPositionDirectionRequired, "summaryPositionSide should require a direction while both sides are open")
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionSummary must reject a nil request")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), CalculateOffline: true})
	require.ErrorIs(t, err, common.ErrCannotCalculateOffline, "GetFuturesPositionSummary must refuse an offline calculation")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesPositionSummary must reject an empty pair")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Spot, Pair: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesPositionSummary must reject an unsupported asset")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		if !mockTests {
			t.Skip("the recorded positions are documented examples, which need open positions")
		}
		btcUSD := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
		summary, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: btcUSD})
		require.NoError(t, err, "GetFuturesPositionSummary must not error")
		assert.Equal(t, btcUSD, summary.Pair, "GetFuturesPositionSummary should summarise the pair")
		assert.Equal(t, margin.Multi, summary.MarginType, "GetFuturesPositionSummary should report a cross margin position")
		assert.Equal(t, collateral.SingleMode, summary.CollateralMode, "GetFuturesPositionSummary should report coin-M collateral in the base coin alone")
		assert.Equal(t, currency.BTC, summary.Currency, "GetFuturesPositionSummary should report the coin-M position in its base coin")
		assert.Equal(t, futures.Inverse, summary.ContractSettlementType, "GetFuturesPositionSummary should report an inverse contract")
		assert.Equal(t,
			[]string{"4", "0.00110845", "2024.7812708419876", "0.00110845", "66098.9", "3", "67662", "-0.00010485", "0.14438227", "0.14317271", "0.00120956", "0"},
			wrapperDecimals(summary.Leverage, summary.InitialMarginRequirement, summary.EstimatedLiquidationPrice, summary.CollateralUsed, summary.MarkPrice, summary.CurrentSize, summary.AverageOpenPrice, summary.UnrealisedPNL, summary.TotalCollateral, summary.FreeCollateral, summary.FrozenBalance, summary.IsolatedMargin),
			"GetFuturesPositionSummary should convert the coin-M position and balance")
		// The recorded USDT-M balances hold no USDC, the recorded position's settlement currency
		_, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewPairWithDelimiter("BTC", "USDC", currency.DashDelimiter)})
		assert.ErrorIs(t, err, currency.ErrCurrencyNotFound, "GetFuturesPositionSummary should report a position without collateral")
	})

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the recorded positions are documented examples; the live case summarises the live pairs' positions, if any")
		}
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		for _, a := range wrapperPerpetualAssets {
			summary, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: a, Pair: assetToTradablePairMap[a]})
			if errors.Is(err, futures.ErrNoPositionsFound) || errors.Is(err, errPositionDirectionRequired) {
				continue
			}
			require.NoErrorf(t, err, "GetFuturesPositionSummary must not error for %s", a)
			assert.Equalf(t, assetToTradablePairMap[a], summary.Pair, "GetFuturesPositionSummary should summarise the %s pair", a)
			assert.Equalf(t, a, summary.Asset, "GetFuturesPositionSummary should summarise the %s position", a)
		}
	})
}

// wrapperSwapPositionsReply is a USDT-M positions reply holding BTC-USDT's long and short positions
const wrapperSwapPositionsReply = `{"code":0,"data":[
	{"symbol":"BTC-USDT","currency":"USDT","positionSide":"LONG","isolated":true,"positionAmt":"0.5","unrealizedProfit":"1.5","realisedProfit":"-0.2","initialMargin":"20","margin":"21","liquidationPrice":60000,"avgPrice":"70000","leverage":20,"positionValue":"35000","markPrice":"70003"},
	{"symbol":"BTC-USDT","currency":"USDT","positionSide":"SHORT","isolated":false,"positionAmt":"0.1","margin":"5","leverage":10},
	{"symbol":"ETH-USDT","currency":"USDT","positionSide":"LONG","positionAmt":"1"}]}`

func TestSwapPositionSummary(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/swap/v2/user/positions": {wrapperSwapPositionsReply, wrapperSwapPositionsReply, wrapperSwapPositionsReply, wrapperSwapPositionsReply, `{"code":100500,"msg":"busy"}`, wrapperSwapPositionsReply, wrapperSwapPositionsReply},
		"GET /openApi/swap/v3/user/balance": {
			`{"code":0,"data":[{"asset":"USDC","balance":"5"},{"asset":"USDT","balance":"100","availableMargin":"60"}]}`,
			`{"code":0,"data":[{"asset":"USDC","balance":"5"}]}`,
			`{"code":100500,"msg":"busy"}`,
			`{"code":0,"data":[{"asset":"USDT","balance":"100","availableMargin":"60"}]}`,
		},
		"GET /openApi/swap/v1/trade/assetMode": {`{"code":0,"data":{"assetMode":"singleAssetMode"}}`, `{"code":100500,"msg":"busy"}`},
	})
	summary, err := ex.swapPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: pair, Direction: order.Long})
	require.NoError(t, err, "swapPositionSummary must not error")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "currency": {"USDT"}}, requests()[0].params, "swapPositionSummary should request the pair's positions in its settlement currency")
	assert.Equal(t, margin.Isolated, summary.MarginType, "swapPositionSummary should report an isolated position")
	assert.Equal(t, collateral.SingleMode, summary.CollateralMode, "swapPositionSummary should report the account's collateral mode")
	assert.Equal(t, currency.USDT, summary.Currency, "swapPositionSummary should report the position in its settlement currency")
	assert.Equal(t, futures.Linear, summary.ContractSettlementType, "swapPositionSummary should report a linear contract")
	assert.Equal(t,
		[]string{"35000", "20", "20", "60000", "21", "70003", "0.5", "70000", "1.5", "-0.2", "100", "60", "40", "21"},
		wrapperDecimals(summary.NotionalSize, summary.Leverage, summary.InitialMarginRequirement, summary.EstimatedLiquidationPrice, summary.CollateralUsed, summary.MarkPrice, summary.CurrentSize, summary.AverageOpenPrice, summary.UnrealisedPNL, summary.RealisedPNL, summary.TotalCollateral, summary.FreeCollateral, summary.FrozenBalance, summary.IsolatedMargin),
		"swapPositionSummary should convert the long position and the USDT balance")

	_, err = ex.swapPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: pair, Direction: order.Short})
	assert.ErrorIs(t, err, currency.ErrCurrencyNotFound, "swapPositionSummary should report a position without collateral")
	_, err = ex.swapPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: pair, Direction: order.Short})
	assert.ErrorIs(t, err, errAPIResponse, "swapPositionSummary should return the balance request's error")
	_, err = ex.swapPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: pair, Direction: order.Short})
	assert.ErrorIs(t, err, errAPIResponse, "swapPositionSummary should return the collateral mode request's error")
	_, err = ex.swapPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: pair})
	assert.ErrorIs(t, err, errAPIResponse, "swapPositionSummary should return the positions request's error")
	_, err = ex.swapPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: pair})
	assert.ErrorIs(t, err, errPositionDirectionRequired, "swapPositionSummary should require a direction while both sides are open")
	_, err = ex.swapPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewPairWithDelimiter("SOL", "USDT", currency.DashDelimiter)})
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "swapPositionSummary should report a pair without a position")
	_, err = new(Exchange).swapPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: pair})
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "swapPositionSummary should return the symbol formatting error")
}

func TestCoinMarginedPositionSummary(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	positions := `{"code":0,"data":[{"symbol":"BTC-USD","positionSide":"Short","isolated":true,"positionAmt":"7","initialMargin":"0.002","liquidationPrice":90000,"avgPrice":"70000","leverage":5,"markPrice":"70001","unrealizedProfit":"0.0001"}]}`
	bothSides := `{"code":0,"data":[{"symbol":"BTC-USD","positionSide":"LONG","positionAmt":"1"},{"symbol":"BTC-USD","positionSide":"Short","positionAmt":"2"}]}`
	ex, _ := wrapperTestServer(t, map[string][]string{
		"GET /openApi/cswap/v1/user/positions": {positions, positions, positions, `{"code":100500,"msg":"busy"}`, bothSides},
		"GET /openApi/cswap/v1/user/balance":   {`{"code":0,"data":[{"asset":"BTC","balance":"0.1","availableMargin":"0.09"}]}`, `{"code":0,"data":[]}`, `{"code":100500,"msg":"busy"}`},
	})
	summary, err := ex.coinMarginedPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: pair})
	require.NoError(t, err, "coinMarginedPositionSummary must not error")
	assert.Equal(t, margin.Isolated, summary.MarginType, "coinMarginedPositionSummary should report an isolated position")
	assert.Equal(t,
		[]string{"5", "0.002", "90000", "0.002", "70001", "7", "70000", "0.0001", "0.1", "0.09", "0.01", "0.002"},
		wrapperDecimals(summary.Leverage, summary.InitialMarginRequirement, summary.EstimatedLiquidationPrice, summary.CollateralUsed, summary.MarkPrice, summary.CurrentSize, summary.AverageOpenPrice, summary.UnrealisedPNL, summary.TotalCollateral, summary.FreeCollateral, summary.FrozenBalance, summary.IsolatedMargin),
		"coinMarginedPositionSummary should convert the position, its side in any case, and the balance")
	_, err = ex.coinMarginedPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: pair, Direction: order.Short})
	assert.ErrorIs(t, err, currency.ErrCurrencyNotFound, "coinMarginedPositionSummary should report a position without collateral")
	_, err = ex.coinMarginedPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: pair, Direction: order.Long})
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "coinMarginedPositionSummary should report a side without a position")
	_, err = ex.coinMarginedPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: pair})
	assert.ErrorIs(t, err, errAPIResponse, "coinMarginedPositionSummary should return the positions request's error")
	_, err = ex.coinMarginedPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: pair})
	assert.ErrorIs(t, err, errPositionDirectionRequired, "coinMarginedPositionSummary should require a direction while both sides are open")
	ex, _ = wrapperTestServer(t, map[string][]string{
		"GET /openApi/cswap/v1/user/positions": {positions},
		"GET /openApi/cswap/v1/user/balance":   {`{"code":100500,"msg":"busy"}`},
	})
	_, err = ex.coinMarginedPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: pair})
	assert.ErrorIs(t, err, errAPIResponse, "coinMarginedPositionSummary should return the balance request's error")
	_, err = new(Exchange).coinMarginedPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: pair})
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "coinMarginedPositionSummary should return the symbol formatting error")
}

func TestIsolatedMarginType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, margin.Isolated, isolatedMarginType(true), "isolatedMarginType should report an isolated position")
	assert.Equal(t, margin.Multi, isolatedMarginType(false), "isolatedMarginType should report a cross margin position")
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionOrders must reject a nil request")
	start := time.UnixMilli(1785503625603)
	for _, tc := range []struct {
		name string
		req  *futures.PositionsRequest
		err  error
	}{
		{"asset", &futures.PositionsRequest{Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, StartDate: start}, asset.ErrNotSupported},
		{"pairs", &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, StartDate: start}, currency.ErrCurrencyPairsEmpty},
		{"start", &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}}, common.ErrDateUnset},
		{"range", &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}, StartDate: start, EndDate: start.Add(-time.Hour)}, common.ErrStartAfterEnd},
	} {
		_, err := e.GetFuturesPositionOrders(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetFuturesPositionOrders must reject a request with an invalid %s", tc.name)
	}

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		if !mockTests {
			t.Skip("the recorded orders are documented examples in a past window")
		}
		btcUSD := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
		resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{btcUSD}, StartDate: start, EndDate: time.UnixMilli(1786021625603)})
		require.NoError(t, err, "GetFuturesPositionOrders must not error")
		require.Len(t, resp, 1, "GetFuturesPositionOrders must return the pair's position")
		assert.Equal(t, btcUSD, resp[0].Pair, "GetFuturesPositionOrders should return the pair's position")
		assert.Equal(t, futures.Inverse, resp[0].ContractSettlementType, "GetFuturesPositionOrders should report an inverse contract")
		require.Len(t, resp[0].Orders, 1, "GetFuturesPositionOrders must return the position's order")
		assert.Equal(t, margin.Isolated, resp[0].Orders[0].MarginType, "GetFuturesPositionOrders should give each order the pair's margin type")
		assert.Equal(t, 0.002585, resp[0].Orders[0].Fee, "GetFuturesPositionOrders should convert each order")
	})

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("the recorded orders are documented examples in a past window; the live case fetches the live pairs' last day")
		}
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		settlement := map[asset.Item]futures.ContractSettlementType{asset.USDTMarginedFutures: futures.Linear, asset.CoinMarginedFutures: futures.Inverse}
		for _, a := range wrapperPerpetualAssets {
			pair := assetToTradablePairMap[a]
			resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: a, Pairs: currency.Pairs{pair}, StartDate: time.Now().Add(-24 * time.Hour)})
			require.NoErrorf(t, err, "GetFuturesPositionOrders must not error for %s", a)
			require.Lenf(t, resp, 1, "GetFuturesPositionOrders must return the %s pair's position", a)
			assert.Equalf(t, pair, resp[0].Pair, "GetFuturesPositionOrders should return the %s pair's position", a)
			assert.Equalf(t, a, resp[0].Asset, "GetFuturesPositionOrders should return the %s position", a)
			assert.Equalf(t, settlement[a], resp[0].ContractSettlementType, "GetFuturesPositionOrders should report the %s contract's settlement", a)
		}
	})

	t.Run("windows and margin types", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/trade/allOrders":  {wrapperPerpetualHistoryPage(0, 2, "BTC-USDT"), wrapperPerpetualHistoryPage(2, 1, "BTC-USDT"), wrapperPerpetualHistoryPage(0, 1, "BTC-USDT"), `{"code":100500,"msg":"busy"}`, wrapperPerpetualHistoryPage(0, 1, "BTC-USDT")},
			"GET /openApi/swap/v2/trade/marginType": {`{"code":0,"data":{"marginType":"CROSSED"}}`, `{"code":0,"data":{"marginType":"ODD"}}`, `{"code":100500,"msg":"busy"}`},
		})
		end := start.Add(perpetualOrderHistorySpan + time.Hour)
		resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}, StartDate: start, EndDate: end})
		require.NoError(t, err, "GetFuturesPositionOrders must not error")
		require.Len(t, resp, 1, "GetFuturesPositionOrders must return the pair's position")
		require.Len(t, resp[0].Orders, 3, "GetFuturesPositionOrders must return every window's orders")
		assert.Equal(t, margin.Multi, resp[0].Orders[2].MarginType, "GetFuturesPositionOrders should give each order the pair's margin type")
		assert.Equal(t, futures.Linear, resp[0].ContractSettlementType, "GetFuturesPositionOrders should report a linear contract")
		got := requests()
		require.Len(t, got, 3, "GetFuturesPositionOrders must request each seven day window and the margin type")
		assert.Equal(t, wrapperMilliseconds(end), got[1].params.Get("endTime"), "GetFuturesPositionOrders should end the last window at the request's end")
		resp, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}, StartDate: start, EndDate: start.Add(time.Hour)})
		require.NoError(t, err, "GetFuturesPositionOrders must not error for a margin type it cannot convert")
		assert.Equal(t, margin.Unset, resp[0].Orders[0].MarginType, "GetFuturesPositionOrders should leave a margin type it cannot convert unset")
		_, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}, StartDate: start, EndDate: start.Add(time.Hour)})
		assert.ErrorIs(t, err, errAPIResponse, "GetFuturesPositionOrders should return the orders request's error")
		_, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSDT()}, StartDate: start, EndDate: start.Add(time.Hour)})
		assert.ErrorIs(t, err, errAPIResponse, "GetFuturesPositionOrders should return the margin type request's error")
	})

	t.Run("coin-M", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/cswap/v1/trade/orderHistory": {wrapperPerpetualHistoryPage(0, 1, "BTC-USD"), wrapperPerpetualHistoryPage(0, 1, "BTC-USD"), wrapperPerpetualHistoryPage(0, 1, "BTC-USD"), `{"code":100500,"msg":"busy"}`},
			"GET /openApi/cswap/v1/trade/marginType":   {`{"code":0,"data":{"symbol":"BTC-USD","marginType":"CROSSED"}}`, `{"code":0,"data":[{"symbol":"BTC-USD","marginType":"ODD"}]}`, `{"code":100500,"msg":"busy"}`},
		})
		pairs := currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)}
		started := time.Now()
		resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.CoinMarginedFutures, Pairs: pairs, StartDate: started.Add(-time.Hour)})
		require.NoError(t, err, "GetFuturesPositionOrders must not error")
		assert.Equal(t, margin.Multi, resp[0].Orders[0].MarginType, "GetFuturesPositionOrders should give each order the pair's margin type")
		endTime, err := strconv.ParseInt(requests()[0].params.Get("endTime"), 10, 64)
		require.NoError(t, err, "endTime must be a number")
		assert.WithinRange(t, time.UnixMilli(endTime), started.Truncate(time.Millisecond), time.Now(), "GetFuturesPositionOrders should end a request without an end now")
		resp, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.CoinMarginedFutures, Pairs: pairs, StartDate: started.Add(-time.Hour)})
		require.NoError(t, err, "GetFuturesPositionOrders must not error for a margin type it cannot convert")
		assert.Equal(t, margin.Unset, resp[0].Orders[0].MarginType, "GetFuturesPositionOrders should leave a margin type it cannot convert unset")
		_, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.CoinMarginedFutures, Pairs: pairs, StartDate: started.Add(-time.Hour)})
		assert.ErrorIs(t, err, errAPIResponse, "GetFuturesPositionOrders should return the margin type request's error")
		_, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.CoinMarginedFutures, Pairs: pairs, StartDate: started.Add(-time.Hour)})
		assert.ErrorIs(t, err, errAPIResponse, "GetFuturesPositionOrders should return the orders request's error")
	})
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.SetCollateralMode(t.Context(), asset.CoinMarginedFutures, collateral.MultiMode), common.ErrFunctionNotSupported, "SetCollateralMode must report that coin-M is margined in the base coin alone")
	require.ErrorIs(t, e.SetCollateralMode(t.Context(), asset.Spot, collateral.MultiMode), asset.ErrNotSupported, "SetCollateralMode must reject an unsupported asset")
	require.ErrorIs(t, e.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.PortfolioMode), order.ErrCollateralInvalid, "SetCollateralMode must reject a mode BingX does not offer")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		require.NoError(t, ex.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.MultiMode), "SetCollateralMode must not error")
	})

	t.Run("single asset mode", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{"POST /openApi/swap/v1/trade/assetMode": {`{"code":0,"data":{"assetMode":"singleAssetMode"}}`, `{"code":100500,"msg":"busy"}`}})
		require.NoError(t, ex.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.SingleMode), "SetCollateralMode must not error")
		assert.Equal(t, url.Values{"assetMode": {"singleAssetMode"}}, requests()[0].params, "SetCollateralMode should switch the account to single asset mode")
		assert.ErrorIs(t, ex.SetCollateralMode(t.Context(), asset.USDTMarginedFutures, collateral.SingleMode), errAPIResponse, "SetCollateralMode should return the request's error")
	})
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	_, err := e.GetCollateralMode(t.Context(), asset.CoinMarginedFutures)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetCollateralMode must report that coin-M is margined in the base coin alone")
	_, err = e.GetCollateralMode(t.Context(), asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCollateralMode must reject an unsupported asset")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		mode, err := ex.GetCollateralMode(t.Context(), asset.USDTMarginedFutures)
		require.NoError(t, err, "GetCollateralMode must not error")
		if mockTests {
			assert.Equal(t, collateral.MultiMode, mode, "GetCollateralMode should report multi-assets mode")
		}
	})

	t.Run("modes", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v1/trade/assetMode": {`{"code":0,"data":{"assetMode":"singleAssetMode"}}`, `{"code":0,"data":{"assetMode":"portfolioMode"}}`, `{"code":100500,"msg":"busy"}`},
		})
		mode, err := ex.GetCollateralMode(t.Context(), asset.USDTMarginedFutures)
		require.NoError(t, err, "GetCollateralMode must not error")
		assert.Equal(t, collateral.SingleMode, mode, "GetCollateralMode should report single asset mode")
		_, err = ex.GetCollateralMode(t.Context(), asset.USDTMarginedFutures)
		assert.ErrorIs(t, err, order.ErrCollateralInvalid, "GetCollateralMode should reject a mode it does not know")
		_, err = ex.GetCollateralMode(t.Context(), asset.USDTMarginedFutures)
		assert.ErrorIs(t, err, errAPIResponse, "GetCollateralMode should return the request's error")
	})
}

func TestGetMarginRatesHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginRatesHistory(t.Context(), &margin.RateHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetMarginRatesHistory should report that BingX offers no margin lending")
}
