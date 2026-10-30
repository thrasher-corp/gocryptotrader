package mexc

import (
	"bytes"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// futuresContractsReply holds the venue's details of a linear and an inverse contract, trimmed to the fields read
const futuresContractsReply = `[{"symbol":"BTC_USDT","baseCoin":"BTC","quoteCoin":"USDT","settleCoin":"USDT",` +
	`"contractSize":0.0001,"minLeverage":1,"maxLeverage":500,"priceUnit":0.1,"volUnit":1,"minVol":1,"maxVol":400000,` +
	`"limitMaxVol":2500000,"state":0,"apiAllowed":true,"futureType":1,"createTime":1591242684000},{"symbol":"BTC_USD",` +
	`"baseCoin":"BTC","quoteCoin":"USD","settleCoin":"BTC","contractSize":100,"minLeverage":1,"maxLeverage":125,` +
	`"priceUnit":0.1,"volUnit":1,"minVol":1,"maxVol":2800,"limitMaxVol":2800,"state":0,"apiAllowed":true,"futureType":1,` +
	`"createTime":1608718223000}]`

// futuresContractsRoute serves the contract details the wrapper reads contract sizes and settlement currencies from
var futuresContractsRoute = futuresRoute{method: http.MethodGet, path: "contract/detail/country", data: futuresContractsReply}

// futuresFilledOrderReply is a filled isolated market order opening a long position
const futuresFilledOrderReply = `{"orderId":"739106551624717312","symbol":"BTC_USDT","positionId":1109973831,` +
	`"price":109777.5,"priceStr":"109777.500000000000000000","vol":5,"leverage":2,"side":1,"category":1,"orderType":5,` +
	`"dealAvgPrice":109777.5,"dealAvgPriceStr":"109777.500000000000000000","dealVol":5,"orderMargin":27.444375,` +
	`"takerFee":0.0329,"makerFee":0,"profit":0,"feeCurrency":"USDT","openType":1,"state":3,"externalOid":"gct-1",` +
	`"errorCode":0,"usedMargin":27.444375,"createTime":1761887134000,"updateTime":1761887135000,"positionMode":1,` +
	`"version":2,"showCancelReason":0,"showProfitRateShare":0,"bboTypeNum":0,"totalFee":0.0329,` +
	`"zeroSaveTotalFeeBinance":0,"zeroTradeTotalFeeBinance":0.0329}`

// futuresFilledOrder returns futuresFilledOrderReply as it decodes
func futuresFilledOrder() *FuturesOrderResponse {
	return &FuturesOrderResponse{
		OrderID:                  "739106551624717312",
		Symbol:                   "BTC_USDT",
		PositionID:               "1109973831",
		Price:                    109777.5,
		PriceExact:               109777.5,
		Volume:                   5,
		Leverage:                 2,
		Side:                     FuturesSideOpenLong,
		Category:                 FuturesOrderCategoryLimit,
		OrderType:                FuturesOrderTypeMarket,
		DealAveragePrice:         109777.5,
		DealAveragePriceExact:    109777.5,
		DealVolume:               5,
		OrderMargin:              27.444375,
		UsedMargin:               27.444375,
		TakerFee:                 0.0329,
		FeeCurrency:              currency.USDT,
		OpenType:                 FuturesOpenTypeIsolated,
		State:                    FuturesOrderStateFilled,
		ExternalOrderID:          "gct-1",
		CreateTime:               types.Time(time.UnixMilli(1761887134000)),
		UpdateTime:               types.Time(time.UnixMilli(1761887135000)),
		PositionMode:             FuturesPositionModeHedge,
		Version:                  2,
		TotalFee:                 0.0329,
		ZeroTradeTotalFeeBinance: 0.0329,
	}
}

// futuresFilledOrderDetail returns futuresFilledOrderReply as an exchange's wrapper maps it
func futuresFilledOrderDetail(exchangeName string) *order.Detail {
	return &order.Detail{
		Leverage:             2,
		Price:                109777.5,
		Amount:               5,
		ContractAmount:       5,
		AverageExecutedPrice: 109777.5,
		ExecutedAmount:       5,
		Cost:                 54.88875,
		CostAsset:            currency.USDT,
		Fee:                  0.0329,
		FeeAsset:             currency.USDT,
		Exchange:             exchangeName,
		OrderID:              "739106551624717312",
		ClientOrderID:        "gct-1",
		Type:                 order.Market,
		Side:                 order.Buy,
		Status:               order.Filled,
		AssetType:            asset.Futures,
		Date:                 time.UnixMilli(1761887134000),
		LastUpdated:          time.UnixMilli(1761887135000),
		Pair:                 futuresTradablePair,
		MarginType:           margin.Isolated,
	}
}

// futuresOpenPositionReply is an isolated long position
const futuresOpenPositionReply = `{"positionId":1109973831,"symbol":"BTC_USDT","positionType":1,"openType":1,"state":1,` +
	`"holdVol":5,"holdAvgPrice":109777.5,"openAvgPrice":109777.5,"liquidatePrice":55020.5,"oim":27.444375,"im":27.444375,` +
	`"realised":-0.0329,"leverage":2,"createTime":1761887133854,"updateTime":1761887133854,"version":1,"unRealizedPnl":-0.0039}`

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err, "GetFuturesContractDetails must not error")
	if mockTests {
		// HOOKED_USDT is closed to API trading, so it is listed but not active
		exp := []futures.Contract{
			{
				Exchange:           "MEXC",
				Name:               futuresTradablePair,
				Underlying:         currency.NewBTCUSDT(),
				Asset:              asset.Futures,
				StartDate:          time.Unix(1591242684, 0),
				IsActive:           true,
				Status:             "enabled",
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USDT,
				MarginCurrency:     currency.USDT,
				Multiplier:         0.0001,
				MaxLeverage:        500,
			},
			{
				Exchange:           "MEXC",
				Name:               currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter),
				Underlying:         currency.NewPair(currency.ETH, currency.USDT),
				Asset:              asset.Futures,
				StartDate:          time.Unix(1591243107, 0),
				IsActive:           true,
				Status:             "enabled",
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USDT,
				MarginCurrency:     currency.USDT,
				Multiplier:         0.01,
				MaxLeverage:        500,
			},
			{
				Exchange:           "MEXC",
				Name:               futuresInversePair,
				Underlying:         currency.NewBTCUSD(),
				Asset:              asset.Futures,
				StartDate:          time.Unix(1608718223, 0),
				IsActive:           true,
				Status:             "enabled",
				Type:               futures.Perpetual,
				SettlementType:     futures.Inverse,
				SettlementCurrency: currency.BTC,
				MarginCurrency:     currency.BTC,
				Multiplier:         100,
				MaxLeverage:        125,
			},
			{
				Exchange:           "MEXC",
				Name:               currency.NewPairWithDelimiter("HOOKED", "USDT", currency.UnderscoreDelimiter),
				Underlying:         currency.NewPair(currency.NewCode("HOOKED"), currency.USDT),
				Asset:              asset.Futures,
				StartDate:          time.Unix(1791098820, 0),
				Status:             "enabled",
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USDT,
				MarginCurrency:     currency.USDT,
				Multiplier:         10,
				MaxLeverage:        20,
			},
		}
		assert.Equal(t, exp, result, "every recorded contract should be mapped")
		return
	}
	contracts := make(map[string]futures.Contract, len(result))
	for i := range result {
		contracts[result[i].Name.String()] = result[i]
	}
	linear, ok := contracts[futuresTradablePair.String()]
	require.True(t, ok, "the linear contract must be listed")
	assert.Equal(t, futures.Perpetual, linear.Type, "the linear contract should be perpetual")
	assert.Equal(t, futures.Linear, linear.SettlementType, "a USDT margined contract should be linear")
	assert.Equal(t, currency.USDT, linear.SettlementCurrency, "the linear contract should settle in USDT")
	assert.True(t, linear.IsActive, "the linear contract should be active")
	assert.Positive(t, linear.Multiplier, "the linear contract should carry its size")
	inverse, ok := contracts[futuresInversePair.String()]
	require.True(t, ok, "the inverse contract must be listed")
	assert.Equal(t, futures.Inverse, inverse.SettlementType, "a coin margined contract should be inverse")
	assert.Equal(t, currency.BTC, inverse.SettlementCurrency, "the inverse contract should settle in its base currency")
}

func TestFuturesContractStatus(t *testing.T) {
	t.Parallel()
	for state, exp := range map[uint8]string{
		FuturesContractStateEnabled:   "enabled",
		FuturesContractStateDelivery:  "delivery",
		FuturesContractStateDelivered: "delivered",
		FuturesContractStateOffline:   "offline",
		FuturesContractStatePaused:    "paused",
		9:                             "unknown",
	} {
		assert.Equalf(t, exp, futuresContractStatus(state), "state %d should be named", state)
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Spot, Pair: spotTradablePair})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	result, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: futuresTradablePair, IncludePredictedRate: true})
	require.NoError(t, err, "GetLatestFundingRates must not error")
	require.Len(t, result, 1, "the contract's rate must be returned")
	if mockTests {
		assert.False(t, result[0].TimeChecked.IsZero(), "the rate should record when it was checked")
		result[0].TimeChecked = time.Time{}
		// The predicted rate is the current one, settled at the next funding
		exp := fundingrate.LatestRateResponse{
			Exchange:              "MEXC",
			Asset:                 asset.Futures,
			Pair:                  futuresTradablePair,
			LatestRate:            fundingrate.Rate{Time: time.Unix(1792828800, 0), Rate: decimal.MustFromFloat(0.000035)},
			PredictedUpcomingRate: fundingrate.Rate{Time: time.Unix(1792857600, 0), Rate: decimal.MustFromFloat(0.000034)},
			TimeOfNextRate:        time.Unix(1792857600, 0),
		}
		assert.Equal(t, exp, result[0], "the latest and predicted rates should be mapped")
	} else {
		assert.False(t, result[0].LatestRate.Time.IsZero(), "the latest settled rate should carry its time")
		assert.True(t, result[0].TimeOfNextRate.After(result[0].LatestRate.Time), "the next funding should follow the latest")
		assert.Equal(t, result[0].TimeOfNextRate, result[0].PredictedUpcomingRate.Time, "the predicted rate should be settled at the next funding")
	}

	result, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures})
	require.NoError(t, err, "GetLatestFundingRates must not error for the enabled pairs")
	require.Len(t, result, 2, "every enabled contract's rate must be returned")
	if mockTests {
		for i := range result {
			assert.Falsef(t, result[i].TimeChecked.IsZero(), "rate %d should record when it was checked", i)
			result[i].TimeChecked = time.Time{}
		}
		exp := []fundingrate.LatestRateResponse{
			{
				Exchange:       "MEXC",
				Asset:          asset.Futures,
				Pair:           futuresTradablePair,
				LatestRate:     fundingrate.Rate{Time: time.Unix(1792828800, 0), Rate: decimal.MustFromFloat(0.000035)},
				TimeOfNextRate: time.Unix(1792857600, 0),
			},
			{
				Exchange:       "MEXC",
				Asset:          asset.Futures,
				Pair:           futuresInversePair,
				LatestRate:     fundingrate.Rate{Time: time.Unix(1792828800, 0), Rate: decimal.MustFromFloat(0.000098)},
				TimeOfNextRate: time.Unix(1792857600, 0),
			},
		}
		assert.Equal(t, exp, result, "every enabled contract's rate should be mapped")
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Spot, Pair: spotTradablePair})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture)
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: futuresTradablePair, IncludePayments: true})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
	start, end := futuresWindow()
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: futuresTradablePair, StartDate: end, EndDate: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: futuresTradablePair, IncludePredictedRate: true})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	// The recorded page of settled rates is trimmed to its newest five
	rate := func(settled int64, value float64) fundingrate.Rate {
		return fundingrate.Rate{Time: time.Unix(settled, 0), Rate: decimal.MustFromFloat(value)}
	}
	recorded := []fundingrate.Rate{
		rate(1792713600, 0.000052), rate(1792742400, 0.000072), rate(1792771200, 0.0001), rate(1792800000, 0.000039), rate(1792828800, 0.000035),
	}
	if mockTests {
		exp := &fundingrate.HistoricalRates{
			Exchange:              "MEXC",
			Asset:                 asset.Futures,
			Pair:                  futuresTradablePair,
			StartDate:             time.Unix(1792713600, 0),
			EndDate:               time.Unix(1792828800, 0),
			LatestRate:            recorded[4],
			PredictedUpcomingRate: rate(1792857600, 0.000034),
			FundingRates:          recorded,
			TimeOfNextRate:        time.Unix(1792857600, 0),
		}
		assert.Equal(t, exp, result, "the settled rates should be mapped oldest first, with the predicted rate")
	} else {
		require.NotEmpty(t, result.FundingRates, "settled rates must be returned")
		for i := 1; i < len(result.FundingRates); i++ {
			assert.Truef(t, result.FundingRates[i].Time.After(result.FundingRates[i-1].Time), "rate %d should follow the one before it", i)
		}
		assert.Equal(t, result.FundingRates[len(result.FundingRates)-1], result.LatestRate, "the latest rate should be the newest settled")
		assert.True(t, result.TimeOfNextRate.After(result.LatestRate.Time), "the next funding should follow the latest")
	}

	start = start.Add(-7 * 24 * time.Hour)
	result, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: futuresTradablePair, StartDate: start, EndDate: end})
	require.NoError(t, err, "GetHistoricalFundingRates must not error within a window")
	if mockTests {
		// The window ends before the newest recorded rate, and its dates are those of the rates returned
		exp := &fundingrate.HistoricalRates{
			Exchange:     "MEXC",
			Asset:        asset.Futures,
			Pair:         futuresTradablePair,
			StartDate:    time.Unix(1792713600, 0),
			EndDate:      time.Unix(1792800000, 0),
			LatestRate:   recorded[3],
			FundingRates: recorded[:4],
		}
		assert.Equal(t, exp, result, "the window's rates should be mapped")
		return
	}
	require.NotEmpty(t, result.FundingRates, "the window's rates must be returned")
	for i := range result.FundingRates {
		assert.Falsef(t, result.FundingRates[i].Time.Before(start), "rate %d should not precede the window", i)
		assert.Falsef(t, result.FundingRates[i].Time.After(end), "rate %d should not follow the window", i)
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Futures, futuresTradablePair)
	require.NoError(t, err, "IsPerpetualFutureCurrency must not error")
	assert.True(t, is, "every futures contract should be perpetual")
	is, err = e.IsPerpetualFutureCurrency(asset.Spot, spotTradablePair)
	require.NoError(t, err, "IsPerpetualFutureCurrency must not error")
	assert.False(t, is, "a spot pair should not be perpetual")
}

func TestGetCollateralCurrencyForContract(t *testing.T) {
	t.Parallel()
	_, _, err := e.GetCollateralCurrencyForContract(asset.Spot, spotTradablePair)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, _, err = e.GetCollateralCurrencyForContract(asset.Futures, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	ex := newPrivateTestExchange(t, futuresHandler(t, futuresContractsRoute))
	_, _, err = ex.GetCollateralCurrencyForContract(asset.Futures, futuresTradablePair)
	require.ErrorIs(t, err, errContractNotFound, "an unread contract must not be guessed")

	_, err = ex.GetFuturesContracts(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesContracts must not error")
	for p, exp := range map[currency.Pair]currency.Code{futuresTradablePair: currency.USDT, futuresInversePair: currency.BTC} {
		code, a, err := ex.GetCollateralCurrencyForContract(asset.Futures, p)
		require.NoErrorf(t, err, "GetCollateralCurrencyForContract must not error for %s", p)
		assert.Equalf(t, exp, code, "%s should settle in %s", p, exp)
		assert.Equalf(t, asset.Futures, a, "%s collateral should be held in the futures account", p)
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: spotTradablePair.Base.Item, Quote: spotTradablePair.Quote.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	linear := key.PairAsset{Base: futuresTradablePair.Base.Item, Quote: futuresTradablePair.Quote.Item, Asset: asset.Futures}
	inverse := key.PairAsset{Base: futuresInversePair.Base.Item, Quote: futuresInversePair.Quote.Item, Asset: asset.Futures}
	result, err := e.GetOpenInterest(t.Context(), linear)
	require.NoError(t, err, "GetOpenInterest must not error for a contract")
	require.Len(t, result, 1, "the contract's open interest must be returned")
	// The open interest is the contracts held times each contract's size, in the base currency of a linear contract
	// and the quote currency of an inverse one
	if mockTests {
		exp := []futures.OpenInterest{{Key: key.NewExchangeAssetPair("MEXC", asset.Futures, futuresTradablePair), OpenInterest: 48989.418000000005}}
		assert.Equal(t, exp, result, "the contract's open interest should be mapped")
	} else {
		assert.True(t, result[0].Key.Pair().Equal(futuresTradablePair), "the contract asked for should be returned")
		assert.Positive(t, result[0].OpenInterest, "the open interest should be positive")
	}

	for _, keys := range [][]key.PairAsset{{linear, inverse}, nil} {
		result, err = e.GetOpenInterest(t.Context(), keys...)
		require.NoErrorf(t, err, "GetOpenInterest must not error for %d contracts", len(keys))
		if !mockTests {
			assert.Lenf(t, result, 2, "both contracts' open interest should be returned for %d contracts asked for", len(keys))
			continue
		}
		exp := []futures.OpenInterest{
			{Key: key.NewExchangeAssetPair("MEXC", asset.Futures, futuresTradablePair), OpenInterest: 49021.712400000004},
			{Key: key.NewExchangeAssetPair("MEXC", asset.Futures, futuresInversePair), OpenInterest: 172493300},
		}
		assert.Equalf(t, exp, result, "both contracts' open interest should be mapped for %d contracts asked for", len(keys))
	}
}

func TestFuturesVolumes(t *testing.T) {
	t.Parallel()
	linear := &FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USDT, SettleCoin: currency.USDT, ContractSize: 0.0001}
	base, quote := futuresVolumes(linear, 1000, 5000)
	assert.InDelta(t, 0.1, base, 1e-12, "a linear contract's base volume should be its contracts times its size")
	assert.Equal(t, 5000.0, quote, "a linear contract's quote volume should be its turnover")

	inverse := &FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USD, SettleCoin: currency.BTC, ContractSize: 100}
	base, quote = futuresVolumes(inverse, 10, 0.02)
	assert.Equal(t, 0.02, base, "an inverse contract's base volume should be its turnover")
	assert.Equal(t, 1000.0, quote, "an inverse contract's quote volume should be its contracts times its size")
}

func TestFuturesCost(t *testing.T) {
	t.Parallel()
	cost := futuresCost(&FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USDT, SettleCoin: currency.USDT, ContractSize: 0.0001}, 5, 100000)
	assert.InDelta(t, 50.0, cost, 1e-9, "a linear contract should be worth its contracts times its size and price in its settlement currency")

	cost = futuresCost(&FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USD, SettleCoin: currency.BTC, ContractSize: 100}, 5, 100000)
	assert.InDelta(t, 0.005, cost, 1e-12, "an inverse contract should be worth its contracts times its size over the price in its settlement currency")
}

func TestFuturesQuoteCost(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		contract *FuturesContractResponse
		exp      float64
	}{
		{&FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USDT, SettleCoin: currency.USDT, ContractSize: 0.0001}, 50},
		{&FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USDC, SettleCoin: currency.USDC, ContractSize: 0.0001}, 50},
		{&FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USD, SettleCoin: currency.BTC, ContractSize: 100}, 500},
	} {
		assert.InDeltaf(t, tc.exp, futuresQuoteCost(tc.contract, 5, 100000), 1e-9, "five %s margined contracts at 100000 should cost %v %s", tc.contract.SettleCoin, tc.exp, tc.contract.QuoteCoin)
	}
}

func TestFuturesOrderDetailInverseCost(t *testing.T) {
	t.Parallel()
	ex := newFuturesWebsocketTestExchange(t)
	o := futuresFilledOrder()
	o.Symbol = "BTC_USD"
	o.Price, o.Volume, o.DealVolume, o.DealAveragePrice, o.State = 51000, 10, 4, 50000, FuturesOrderStateUnfilled
	o.FeeCurrency = currency.BTC
	detail, err := ex.futuresOrderDetail(t.Context(), o, currency.EMPTYPAIR)
	require.NoError(t, err, "futuresOrderDetail must not error")
	// Four of ten contracts of 100 USD each cost 400 USD, whatever the price they filled at
	exp := futuresFilledOrderDetail(ex.Name)
	exp.Pair = futuresInversePair
	exp.Price, exp.Amount, exp.ContractAmount, exp.ExecutedAmount, exp.RemainingAmount = 51000, 10, 10, 4, 6
	exp.AverageExecutedPrice, exp.Cost, exp.CostAsset = 50000, 400, currency.USD
	exp.Status, exp.FeeAsset = order.PartiallyFilled, currency.BTC
	assert.Equal(t, exp, &detail, "a partly filled inverse order should cost its filled contracts' value in its quote currency")
}

func TestFuturesOrderDetailUnlistedSymbol(t *testing.T) {
	t.Parallel()
	unformatted := new(Exchange)
	unformatted.CurrencyPairs.UseGlobalFormat = true
	o := futuresFilledOrder()
	o.DealVolume = 0
	_, err := unformatted.futuresOrderDetail(t.Context(), o, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "futuresOrderDetail must return why a symbol has no pair without a fallback pair")

	ex := newFuturesWebsocketTestExchange(t)
	o.Symbol = "UNLISTED"
	_, err = ex.futuresOrderDetail(t.Context(), o, currency.EMPTYPAIR)
	require.ErrorContains(t, err, "delimiter not found", "futuresOrderDetail must return why a symbol which names no pair has none")
	detail, err := ex.futuresOrderDetail(t.Context(), o, futuresTradablePair)
	require.NoError(t, err, "futuresOrderDetail must not error with a fallback pair")
	assert.Equal(t, futuresTradablePair, detail.Pair, "a symbol which names no pair should take the fallback pair")

	ex = testExchangeFor(t, futuresHandler(t, futuresContractsRoute), liveUnavailable)
	o = futuresFilledOrder()
	o.Symbol = "ETH_USDT"
	detail, err = ex.futuresOrderDetail(t.Context(), o, currency.EMPTYPAIR)
	require.NoError(t, err, "futuresOrderDetail must not error for a contract the venue no longer lists")
	exp := futuresFilledOrderDetail(ex.Name)
	exp.Pair = currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter)
	exp.Cost, exp.CostAsset = 0, currency.EMPTYCODE
	assert.Equal(t, exp, &detail, "an order on a contract the venue no longer lists should be reported without its cost")
}

func TestFuturesCurrentLeverage(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/leverage",
		query:  "symbol=BTC_USDT",
		data: `[{"positionType":1,"openType":1,"leverage":2},{"positionType":2,"openType":1,"leverage":3},` +
			`{"positionType":1,"openType":2,"leverage":5},{"positionType":2,"openType":2,"leverage":0}]`,
	}), liveUnavailable)
	for _, tc := range []struct {
		side, openType uint8
		exp            uint64
		err            error
	}{
		{FuturesSideOpenLong, FuturesOpenTypeIsolated, 2, nil},
		{FuturesSideOpenShort, FuturesOpenTypeIsolated, 3, nil},
		{FuturesSideOpenLong, FuturesOpenTypeCross, 5, nil},
		{FuturesSideOpenShort, FuturesOpenTypeCross, 0, errInvalidLeverage},
	} {
		leverage, err := ex.futuresCurrentLeverage(t.Context(), futuresTradablePair, tc.side, tc.openType)
		require.ErrorIsf(t, err, tc.err, "futuresCurrentLeverage must return the expected error for side %d and open type %d", tc.side, tc.openType)
		assert.Equalf(t, tc.exp, leverage, "side %d and open type %d should take the leverage of their positions", tc.side, tc.openType)
	}

	ex = testExchangeFor(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"code":401,"message":"Not logged in"}`))
	}, liveUnavailable)
	_, err := ex.futuresCurrentLeverage(t.Context(), futuresTradablePair, FuturesSideOpenLong, FuturesOpenTypeIsolated)
	assert.ErrorIs(t, err, errFuturesRequestFailed, "futuresCurrentLeverage should return a refused leverage request")
}

func TestFuturesContractIsInverse(t *testing.T) {
	t.Parallel()
	assert.False(t, (&FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USDT, SettleCoin: currency.USDT}).IsInverse(), "a contract settled in its quote currency should be linear")
	assert.True(t, (&FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USD, SettleCoin: currency.BTC}).IsInverse(), "a contract settled in its base currency should be inverse")
	assert.False(t, (&FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USD}).IsInverse(), "a contract without a settlement currency should not be inverse")
}

func TestFuturesHistoryWindows(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1750000000000)
	assert.Equal(t, []historyWindow{{start: start}}, futuresHistoryWindows(start, time.Time{}), "an open window should be sent as it is")
	assert.Equal(t, []historyWindow{{}}, futuresHistoryWindows(time.Time{}, time.Time{}), "an unset window should be sent as it is")

	end := start.Add(200 * 24 * time.Hour)
	exp := []historyWindow{
		{start: start, end: start.Add(futuresHistoryWindow)},
		{start: start.Add(futuresHistoryWindow), end: start.Add(2 * futuresHistoryWindow)},
		{start: start.Add(2 * futuresHistoryWindow), end: end},
	}
	assert.Equal(t, exp, futuresHistoryWindows(start, end), "a 200 day window should be read in parts of at most 90 days, each following on from the one before")

	now := time.Now()
	day := 24 * time.Hour
	for _, tc := range []struct {
		name       string
		start, end time.Time
		parts      int
	}{
		{name: "a window ending in the future", start: now.Add(-30 * day), end: now.Add(61 * day), parts: 1},
		{name: "a later part starting before now and ending after it", start: now.Add(-100 * day), end: now.Add(100 * day), parts: 2},
		{name: "a later part starting a minute before now", start: now.Add(-futuresHistoryWindow - time.Minute), end: now.Add(10 * day), parts: 2},
		{name: "a later part starting a minute after now", start: now.Add(-futuresHistoryWindow + time.Minute), end: now.Add(10 * day), parts: 1},
		{name: "a window starting after now, for the request to refuse", start: now.Add(day), end: now.Add(2 * day), parts: 1},
	} {
		exp := make([]historyWindow, tc.parts)
		for i := range exp {
			from := tc.start.Add(time.Duration(i) * futuresHistoryWindow)
			to := from.Add(futuresHistoryWindow)
			if to.After(tc.end) {
				to = tc.end
			}
			exp[i] = historyWindow{start: from, end: to}
		}
		assert.Equalf(t, exp, futuresHistoryWindows(tc.start, tc.end), "%s should be read in its parts starting before now", tc.name)
	}
}

func TestFuturesOrderSide(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		side       order.Side
		reduceOnly bool
		exp        uint8
	}{
		{order.Buy, false, FuturesSideOpenLong},
		{order.Long, false, FuturesSideOpenLong},
		{order.Buy, true, FuturesSideCloseShort},
		{order.Sell, false, FuturesSideOpenShort},
		{order.Short, false, FuturesSideOpenShort},
		{order.Sell, true, FuturesSideCloseLong},
	} {
		side, err := futuresOrderSide(tc.side, tc.reduceOnly)
		require.NoErrorf(t, err, "futuresOrderSide must not error for %s", tc.side)
		assert.Equalf(t, tc.exp, side, "%s reducing %t should map to side %d", tc.side, tc.reduceOnly, tc.exp)
		back, reduceOnly := orderSideFromFuturesSide(side)
		assert.Equalf(t, tc.side.IsLong(), back.IsLong(), "side %d should map back to the same direction", side)
		assert.Equalf(t, tc.reduceOnly, reduceOnly, "side %d should map back to the same reduction", side)
	}
	_, err := futuresOrderSide(order.AnySide, false)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	side, reduceOnly := orderSideFromFuturesSide(9)
	assert.Equal(t, order.UnknownSide, side, "an unknown side should not be guessed")
	assert.False(t, reduceOnly, "an unknown side should not reduce")
}

func TestFuturesSideType(t *testing.T) {
	t.Parallel()
	for side, exp := range map[uint8]string{
		FuturesSideOpenLong:   FuturesSideTypeBuy,
		FuturesSideCloseShort: FuturesSideTypeBuy,
		FuturesSideOpenShort:  FuturesSideTypeSell,
		FuturesSideCloseLong:  FuturesSideTypeSell,
		0:                     "",
		5:                     "",
	} {
		assert.Equalf(t, exp, futuresSideType(side), "side %d should have side type %q", side, exp)
	}
}

func TestFuturesPositionSide(t *testing.T) {
	t.Parallel()
	assert.Equal(t, order.Long, futuresPositionSide(FuturesPositionTypeLong), "a long position should be long")
	assert.Equal(t, order.Short, futuresPositionSide(FuturesPositionTypeShort), "a short position should be short")
	assert.Equal(t, order.UnknownSide, futuresPositionSide(0), "an unknown position type should not be guessed")
}

func TestFuturesTradeSide(t *testing.T) {
	t.Parallel()
	assert.Equal(t, order.Buy, futuresTradeSide(1), "a taker buy should be a buy")
	assert.Equal(t, order.Sell, futuresTradeSide(2), "a taker sell should be a sell")
	assert.Equal(t, order.UnknownSide, futuresTradeSide(0), "an unknown side should not be guessed")
}

func TestFuturesPositionTypes(t *testing.T) {
	t.Parallel()
	for side, exp := range map[order.Side][]uint8{
		order.UnknownSide: {FuturesPositionTypeLong, FuturesPositionTypeShort},
		order.Buy:         {FuturesPositionTypeLong},
		order.Long:        {FuturesPositionTypeLong},
		order.Sell:        {FuturesPositionTypeShort},
		order.Short:       {FuturesPositionTypeShort},
	} {
		positionTypes, err := futuresPositionTypes(side)
		require.NoErrorf(t, err, "futuresPositionTypes must not error for %s", side)
		assert.Equalf(t, exp, positionTypes, "%s should name its position types", side)
	}
	_, err := futuresPositionTypes(order.ClosePosition)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
}

func TestFuturesOrderTypeFromOrderTypeAndTimeInForce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		oType order.Type
		tif   order.TimeInForce
		exp   uint8
	}{
		{order.Limit, order.UnknownTIF, FuturesOrderTypeLimit},
		{order.Limit, order.GoodTillCancel, FuturesOrderTypeLimit},
		{order.Limit, order.PostOnly, FuturesOrderTypePostOnly},
		{order.Limit, order.GoodTillCancel | order.PostOnly, FuturesOrderTypePostOnly},
		{order.Limit, order.ImmediateOrCancel, FuturesOrderTypeImmediateOrCancel},
		{order.Limit, order.FillOrKill, FuturesOrderTypeFillOrKill},
		{order.Market, order.UnknownTIF, FuturesOrderTypeMarket},
		{order.Market, order.ImmediateOrCancel, FuturesOrderTypeMarket},
	} {
		oType, err := futuresOrderTypeFromOrderTypeAndTimeInForce(tc.oType, tc.tif)
		require.NoErrorf(t, err, "futuresOrderTypeFromOrderTypeAndTimeInForce must not error for %s %s", tc.oType, tc.tif)
		assert.Equalf(t, tc.exp, oType, "%s %s should map to type %d", tc.oType, tc.tif, tc.exp)
	}
	_, err := futuresOrderTypeFromOrderTypeAndTimeInForce(order.Limit, order.GoodTillTime)
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)
	_, err = futuresOrderTypeFromOrderTypeAndTimeInForce(order.Market, order.FillOrKill)
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)
	_, err = futuresOrderTypeFromOrderTypeAndTimeInForce(order.Stop, order.UnknownTIF)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType)
}

func TestOrderTypeFromFuturesOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType, category uint8
		oType               order.Type
		tif                 order.TimeInForce
	}{
		{FuturesOrderTypeLimit, FuturesOrderCategoryLimit, order.Limit, order.GoodTillCancel},
		{FuturesOrderTypePostOnly, FuturesOrderCategoryLimit, order.Limit, order.PostOnly},
		{FuturesOrderTypeImmediateOrCancel, FuturesOrderCategoryLimit, order.Limit, order.ImmediateOrCancel},
		{FuturesOrderTypeFillOrKill, FuturesOrderCategoryLimit, order.Limit, order.FillOrKill},
		{FuturesOrderTypeMarket, FuturesOrderCategoryLimit, order.Market, order.UnknownTIF},
		{FuturesOrderTypeMarketToLimit, FuturesOrderCategoryLimit, order.Market, order.UnknownTIF},
		{FuturesOrderTypeLimit, FuturesOrderCategoryLiquidationCustody, order.Liquidation, order.UnknownTIF},
		{FuturesOrderTypeMarket, FuturesOrderCategoryCustodyClose, order.Liquidation, order.UnknownTIF},
		{FuturesOrderTypeLimit, FuturesOrderCategoryLiquidationReduction, order.Liquidation, order.UnknownTIF},
		{FuturesOrderTypeLimit, FuturesOrderCategoryAutoDeleverage, order.Limit, order.GoodTillCancel},
		{9, FuturesOrderCategoryLimit, order.UnknownType, order.UnknownTIF},
	} {
		oType, tif := orderTypeFromFuturesOrder(tc.orderType, tc.category)
		assert.Equalf(t, tc.oType, oType, "type %d in category %d should map to %s", tc.orderType, tc.category, tc.oType)
		assert.Equalf(t, tc.tif, tif, "type %d in category %d should map to %s", tc.orderType, tc.category, tc.tif)
	}
}

func TestOrderStatusFromFuturesState(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		state  uint8
		filled float64
		exp    order.Status
	}{
		{FuturesOrderStatePending, 0, order.New},
		{FuturesOrderStateUnfilled, 0, order.New},
		{FuturesOrderStateUnfilled, 1, order.PartiallyFilled},
		{FuturesOrderStateFilled, 5, order.Filled},
		{FuturesOrderStateCancelled, 0, order.Cancelled},
		{FuturesOrderStateCancelled, 1, order.PartiallyCancelled},
		{FuturesOrderStateInvalid, 0, order.Rejected},
		{9, 0, order.UnknownStatus},
	} {
		assert.Equalf(t, tc.exp, orderStatusFromFuturesState(tc.state, tc.filled), "state %d with %v filled should map to %s", tc.state, tc.filled, tc.exp)
	}
}

func TestFuturesOpenTypeFromMarginType(t *testing.T) {
	t.Parallel()
	for m, exp := range map[margin.Type]uint8{margin.Unset: FuturesOpenTypeIsolated, margin.Isolated: FuturesOpenTypeIsolated, margin.Multi: FuturesOpenTypeCross} {
		openType, err := futuresOpenTypeFromMarginType(m)
		require.NoErrorf(t, err, "futuresOpenTypeFromMarginType must not error for %s", m)
		assert.Equalf(t, exp, openType, "%s should map to open type %d", m, exp)
	}
	_, err := futuresOpenTypeFromMarginType(margin.SpotIsolated)
	assert.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)

	assert.Equal(t, margin.Isolated, futuresMarginType(FuturesOpenTypeIsolated), "an isolated open type should be isolated")
	assert.Equal(t, margin.Multi, futuresMarginType(FuturesOpenTypeCross), "a cross open type should be multi")
	assert.Equal(t, margin.Unset, futuresMarginType(0), "an unknown open type should not be guessed")
}

func TestFuturesBalance(t *testing.T) {
	t.Parallel()
	// Frozen funds and position margin are held, and the total is what is free and held
	assert.Equal(t, accounts.Balance{Total: 13.5, Hold: 3.5, Free: 10}, futuresBalance(10, 2, 1.5), "futuresBalance should count margin as held")
}

func TestFuturesMarginChange(t *testing.T) {
	t.Parallel()
	documented := 27.444375
	ulpsAbove := func(x float64, n int) float64 {
		for range n {
			x = math.Nextafter(x, math.Inf(1))
		}
		return x
	}
	for _, tc := range []struct {
		newMargin, currentMargin, exp float64
	}{
		{30, documented, 2.555625},
		{25, documented, -2.444375},
		{documented + 0.1, documented, 0.1},
		{documented + 64, documented, 64},
		{0.004, 0.003125, 0.000875},
		{100.1, 0.2, 99.9},
		{0.2, 100.1, -99.9},
		{1000000.3, 0.1, 1000000.2},
		{10000000.00390625, 10000000, 0.00390625},
		{30, 27.5, 2.5},
		{documented, documented, 0},
		{ulpsAbove(documented, 1), documented, 0},
		{ulpsAbove(documented, 2), documented, 0},
		{ulpsAbove(documented, 3), documented, 1e-14},
	} {
		assert.Equalf(t, tc.exp, futuresMarginChange(tc.newMargin, tc.currentMargin), "moving a margin of %v to %v should change it by %v", tc.currentMargin, tc.newMargin, tc.exp)
	}
	assert.True(t, math.IsNaN(futuresMarginChange(math.NaN(), documented)), "a new margin which is not a number should give no change amount")
}

func TestSelectFuturesPosition(t *testing.T) {
	t.Parallel()
	crossLong := &FuturesPositionResponse{PositionID: "1", Symbol: "BTC_USDT", PositionType: FuturesPositionTypeLong, OpenType: FuturesOpenTypeCross}
	isolatedLong := &FuturesPositionResponse{PositionID: "2", Symbol: "BTC_USDT", PositionType: FuturesPositionTypeLong, OpenType: FuturesOpenTypeIsolated}
	short := &FuturesPositionResponse{PositionID: "3", Symbol: "BTC_USDT", PositionType: FuturesPositionTypeShort, OpenType: FuturesOpenTypeCross}
	closed := &FuturesPositionResponse{PositionID: "4", Symbol: "BTC_USDT", PositionType: FuturesPositionTypeShort, State: FuturesPositionStateClosed}
	other := &FuturesPositionResponse{PositionID: "5", Symbol: "ETH_USDT", PositionType: FuturesPositionTypeShort}

	pos, err := selectFuturesPosition([]*FuturesPositionResponse{crossLong, isolatedLong, closed, other, nil}, "BTC_USDT", order.UnknownSide)
	require.NoError(t, err, "selectFuturesPosition must not error with one side open")
	assert.Equal(t, isolatedLong, pos, "the isolated position should be preferred")

	_, err = selectFuturesPosition([]*FuturesPositionResponse{crossLong, short}, "BTC_USDT", order.UnknownSide)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "both sides open must need a direction")
	pos, err = selectFuturesPosition([]*FuturesPositionResponse{crossLong, short}, "BTC_USDT", order.Short)
	require.NoError(t, err, "selectFuturesPosition must not error with a direction")
	assert.Equal(t, short, pos, "the side named should be selected")

	_, err = selectFuturesPosition([]*FuturesPositionResponse{closed, other}, "BTC_USDT", order.UnknownSide)
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound)
}

func TestRecordFuturesCancelOutcomes(t *testing.T) {
	t.Parallel()
	requested := []order.Cancel{{OrderID: "101716841474621953"}, {OrderID: "108885377779302912"}, {OrderID: "108886241042563584"}}
	resp := &order.CancelBatchResponse{Status: make(map[string]string)}
	err := recordFuturesCancelOutcomes(resp, requested, []*FuturesCancelOrderResponse{
		{OrderID: "101716841474621953", ErrorCode: 2040, ErrorMessage: "order not exist"},
		{OrderID: "108886241042563584", ErrorMessage: "success"},
	}, nil)
	require.ErrorIs(t, err, errCancelOutcomeMissing, "an order without an outcome must be reported")
	assert.Equal(t, map[string]string{
		"101716841474621953": "order not exist",
		"108885377779302912": errCancelOutcomeMissing.Error(),
		"108886241042563584": order.Cancelled.String(),
	}, resp.Status, "every requested order should carry its outcome")

	resp = &order.CancelBatchResponse{Status: make(map[string]string)}
	err = recordFuturesCancelOutcomes(resp, requested[:1], nil, errFuturesRequestFailed)
	require.ErrorIs(t, err, errFuturesRequestFailed, "a failed request must be returned")
	assert.Equal(t, errFuturesRequestFailed.Error(), resp.Status["101716841474621953"], "a failed request should be reported against its orders")
}

func TestUpdateAccountBalancesFutures(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/account/assets",
		data:   `[{"currency":"USDT","positionMargin":1.5,"frozenBalance":2,"availableBalance":10},{"currency":"","availableBalance":1}]`,
	}), liveReadOnly)
	subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
	require.NoError(t, err, "UpdateAccountBalances must not error for futures")
	require.Len(t, subAccounts, 1, "the futures account must be returned")
	assert.Equal(t, asset.Futures, subAccounts[0].AssetType, "the futures account should be named")
	if !mockTests {
		return
	}
	b := subAccounts[0].Balances[currency.USDT]
	assert.False(t, b.UpdatedAt.IsZero(), "the balance should record when it was updated")
	b.UpdatedAt = time.Time{}
	subAccounts[0].Balances[currency.USDT] = b
	// A balance without a currency is skipped
	exp := accounts.SubAccounts{{
		AssetType: asset.Futures,
		Balances:  accounts.CurrencyBalances{currency.USDT: {Currency: currency.USDT, Total: 13.5, Hold: 3.5, Free: 10}},
	}}
	assert.Equal(t, exp, subAccounts, "the futures balances should be mapped")
}

func TestSubmitOrderFutures(t *testing.T) {
	t.Parallel()
	submit := func(ex *Exchange, s order.Submit) (*order.SubmitResponse, error) {
		s.Exchange, s.AssetType, s.Pair = ex.Name, asset.Futures, futuresTradablePair
		return ex.SubmitOrder(t.Context(), &s)
	}
	_, err := submit(e, order.Submit{Side: order.Buy, Type: order.Market, QuoteAmount: 100})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "a futures order sized in quote currency must be refused")
	_, err = submit(e, order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, MarginType: margin.SpotIsolated})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = submit(e, order.Submit{Side: order.Buy, Type: order.Market, Amount: 1, Leverage: 2.5})
	require.ErrorIs(t, err, errInvalidLeverage, "fractional leverage must be refused")
	_, err = submit(e, order.Submit{Side: order.Buy, Type: order.Limit, Price: 1, Amount: 1, TimeInForce: order.GoodTillTime})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	t.Run("isolated open takes the account's leverage", func(t *testing.T) {
		t.Parallel()
		ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
			method: http.MethodGet,
			path:   "private/position/leverage",
			query:  "symbol=BTC_USDT",
			data:   `[{"positionType":1,"openType":1,"leverage":20},{"positionType":2,"openType":1,"leverage":10}]`,
		}, futuresRoute{
			method: http.MethodPost,
			path:   "private/order/create",
			body:   `{"symbol":"BTC_USDT","price":50000,"vol":1,"leverage":20,"side":1,"sideType":"buy","type":1,"openType":1,"externalOid":"gct-1","stpMode":0}`,
			data:   `{"orderId":"739113577038255616","ts":1761888808839}`,
		}), liveUnavailable)
		resp, err := submit(ex, order.Submit{Side: order.Buy, Type: order.Limit, Price: 50000, Amount: 1, ClientOrderID: "gct-1"})
		require.NoError(t, err, "SubmitOrder must not error for futures")
		// An order without a margin type is isolated, at the account's leverage, and stamped with the venue's time
		exp := &order.SubmitResponse{
			Exchange:      ex.Name,
			Type:          order.Limit,
			Side:          order.Buy,
			Pair:          futuresTradablePair,
			AssetType:     asset.Futures,
			Leverage:      20,
			Price:         50000,
			Amount:        1,
			ClientOrderID: "gct-1",
			LastUpdated:   time.UnixMilli(1761888808839),
			Date:          time.UnixMilli(1761888808839),
			Status:        order.New,
			OrderID:       "739113577038255616",
			MarginType:    margin.Isolated,
		}
		assert.Equal(t, exp, resp, "the placed order should be returned")
	})

	t.Run("isolated open without leverage set", func(t *testing.T) {
		t.Parallel()
		ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
			method: http.MethodGet,
			path:   "private/position/leverage",
			query:  "symbol=BTC_USDT",
			data:   `[{"positionType":1,"openType":2,"leverage":20}]`,
		}), liveUnavailable)
		_, err := submit(ex, order.Submit{Side: order.Sell, Type: order.Market, Amount: 1})
		assert.ErrorIs(t, err, errInvalidLeverage, "an isolated open without leverage should be refused before it is sent")
	})

	// A close names the position it closes, when one is open, and takes that position's margin mode when it names none.
	// On a one-way account it is also sent reduce-only
	isolatedLong := futuresOpenPositionReply
	crossLong := strings.NewReplacer(`"positionId":1109973831`, `"positionId":1109973832`, `"openType":1`, `"openType":2`).Replace(futuresOpenPositionReply)
	for name, tc := range map[string]struct {
		positionMode string
		positions    string
		marginType   margin.Type
		body         string
		exp          margin.Type
		err          error
	}{
		"cross close": {
			positionMode: "1", positions: "[" + isolatedLong + "," + crossLong + "]", marginType: margin.Multi,
			body: `{"symbol":"BTC_USDT","vol":2,"side":4,"sideType":"sell","type":5,"openType":2,"positionId":1109973832,"stpMode":0}`, exp: margin.Multi,
		},
		"close of the position's margin mode": {
			positionMode: "1", positions: "[" + crossLong + "]",
			body: `{"symbol":"BTC_USDT","vol":2,"side":4,"sideType":"sell","type":5,"openType":2,"positionId":1109973832,"stpMode":0}`, exp: margin.Multi,
		},
		"close without a position": {
			positionMode: "1", positions: "[]",
			body: `{"symbol":"BTC_USDT","vol":2,"side":4,"sideType":"sell","type":5,"openType":1,"stpMode":0}`, exp: margin.Isolated,
		},
		"close of either margin mode": {positionMode: "1", positions: "[" + isolatedLong + "," + crossLong + "]", err: errMarginTypeAmbiguous},
		"one-way cross close": {
			positionMode: "2", positions: "[" + crossLong + "]", marginType: margin.Multi,
			body: `{"symbol":"BTC_USDT","vol":2,"side":4,"sideType":"sell","type":5,"openType":2,"positionId":1109973832,"positionMode":2,"reduceOnly":true,"stpMode":0}`,
			exp:  margin.Multi,
		},
		"one-way close without a position": {
			positionMode: "2", positions: "[]",
			body: `{"symbol":"BTC_USDT","vol":2,"side":4,"sideType":"sell","type":5,"openType":1,"positionMode":2,"reduceOnly":true,"stpMode":0}`,
			exp:  margin.Isolated,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			routes := []futuresRoute{
				{method: http.MethodGet, path: "private/position/position_mode", data: tc.positionMode},
				{method: http.MethodGet, path: "private/position/open_positions", query: "symbol=BTC_USDT", data: tc.positions},
			}
			if tc.body != "" {
				routes = append(routes, futuresRoute{method: http.MethodPost, path: "private/order/create", body: tc.body, data: `{"orderId":"739113577038255617","ts":1761888808840}`})
			}
			ex := testExchangeFor(t, futuresHandler(t, routes...), liveUnavailable)
			resp, err := submit(ex, order.Submit{Side: order.Sell, Type: order.Market, Amount: 2, ReduceOnly: true, MarginType: tc.marginType})
			require.ErrorIs(t, err, tc.err, "SubmitOrder must return the expected error for a closing order")
			if tc.err != nil {
				return
			}
			// A closing order is returned as reduce-only, and sends no leverage
			exp := &order.SubmitResponse{
				Exchange:    ex.Name,
				Type:        order.Market,
				Side:        order.Sell,
				Pair:        futuresTradablePair,
				AssetType:   asset.Futures,
				ReduceOnly:  true,
				Amount:      2,
				LastUpdated: time.UnixMilli(1761888808840),
				Date:        time.UnixMilli(1761888808840),
				Status:      order.New,
				OrderID:     "739113577038255617",
				MarginType:  tc.exp,
			}
			assert.Equal(t, exp, resp, "the closing order should be returned with the closed position's margin type")
		})
	}
}

func TestSubmitFuturesOrder(t *testing.T) {
	t.Parallel()
	submit := func(ex *Exchange, s order.Submit) (*order.SubmitResponse, error) {
		s.Exchange, s.AssetType, s.Pair, s.Amount = ex.Name, asset.Futures, futuresTradablePair, 1
		if s.Type == order.Limit {
			s.Price = 100.5
		}
		return ex.submitFuturesOrder(t.Context(), &s)
	}
	const placed = `{"orderId":"739113577038255618","ts":1761888808841}`
	const (
		readMode      = "GET private/position/position_mode"
		readPositions = "GET private/position/open_positions"
		place         = "POST private/order/create"
	)
	// record passes each request to next, and returns the requests it received, as method and path, and the body of the
	// last order placement
	record := func(next http.HandlerFunc) (http.HandlerFunc, func() ([]string, []byte)) {
		var mu sync.Mutex
		var requests []string
		var body []byte
		return func(w http.ResponseWriter, r *http.Request) {
				sent, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the request body should not error")
				r.Body = io.NopCloser(bytes.NewReader(sent))
				mu.Lock()
				requests = append(requests, r.Method+" "+strings.TrimPrefix(r.URL.Path, "/api/v1/"))
				if r.URL.Path == "/api/v1/private/order/create" {
					body = sent
				}
				mu.Unlock()
				next(w, r)
			}, func() ([]string, []byte) {
				mu.Lock()
				defer mu.Unlock()
				return append([]string(nil), requests...), body
			}
	}
	// refuse answers requests to path as the venue refuses one, and passes the rest to next
	refuse := func(path string, next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/"+path {
				next(w, r)
				return
			}
			_, _ = w.Write([]byte(`{"success":false,"code":510,"message":"Requests are too frequent, please try again later"}`))
		}
	}

	// An opening order names its side type, buy to open a long and sell to open a short, and reads no position mode
	for _, tc := range []struct {
		name string
		s    order.Submit
		body string
	}{
		{
			name: "cross market long", s: order.Submit{Side: order.Buy, Type: order.Market, MarginType: margin.Multi},
			body: `{"symbol":"BTC_USDT","vol":1,"side":1,"sideType":"buy","type":5,"openType":2,"stpMode":0}`,
		},
		{
			name: "cross market short", s: order.Submit{Side: order.Sell, Type: order.Market, MarginType: margin.Multi},
			body: `{"symbol":"BTC_USDT","vol":1,"side":3,"sideType":"sell","type":5,"openType":2,"stpMode":0}`,
		},
		{
			name: "cross limit long", s: order.Submit{Side: order.Buy, Type: order.Limit, MarginType: margin.Multi},
			body: `{"symbol":"BTC_USDT","price":100.5,"vol":1,"side":1,"sideType":"buy","type":1,"openType":2,"stpMode":0}`,
		},
		{
			name: "cross limit short", s: order.Submit{Side: order.Sell, Type: order.Limit, MarginType: margin.Multi},
			body: `{"symbol":"BTC_USDT","price":100.5,"vol":1,"side":3,"sideType":"sell","type":1,"openType":2,"stpMode":0}`,
		},
		{
			name: "isolated market long", s: order.Submit{Side: order.Buy, Type: order.Market, MarginType: margin.Isolated, Leverage: 5},
			body: `{"symbol":"BTC_USDT","vol":1,"leverage":5,"side":1,"sideType":"buy","type":5,"openType":1,"stpMode":0}`,
		},
		{
			name: "isolated market short", s: order.Submit{Side: order.Sell, Type: order.Market, MarginType: margin.Isolated, Leverage: 5},
			body: `{"symbol":"BTC_USDT","vol":1,"leverage":5,"side":3,"sideType":"sell","type":5,"openType":1,"stpMode":0}`,
		},
	} {
		t.Run("open "+tc.name, func(t *testing.T) {
			t.Parallel()
			handler, requests := record(futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/order/create", body: tc.body, data: placed}))
			ex := testExchangeFor(t, handler, liveUnavailable)
			resp, err := submit(ex, tc.s)
			require.NoError(t, err, "submitFuturesOrder must not error")
			assert.Equal(t, "739113577038255618", resp.OrderID, "submitFuturesOrder should return the placed order's ID")
			sent, _ := requests()
			assert.Equal(t, []string{place}, sent, "an opening order should only be placed")
		})
	}

	// A reducing order reads the account's position mode once, then the position it closes. On a hedge account its body
	// is the one listed; on a one-way account it also carries positionMode 2 and reduceOnly
	position := func(id, positionType, openType string) string {
		return strings.NewReplacer(`"positionId":1109973831`, `"positionId":`+id, `"positionType":1`, `"positionType":`+positionType, `"openType":1`, `"openType":`+openType).Replace(futuresOpenPositionReply)
	}
	isolatedLong, crossLong := position("1109973831", "1", "1"), position("1109973832", "1", "2")
	isolatedShort, crossShort := position("1109973833", "2", "1"), position("1109973834", "2", "2")
	for _, tc := range []struct {
		name      string
		side      order.Side
		orderType order.Type
		positions string
		body      string
	}{
		{
			name: "market close of no short", side: order.Buy, orderType: order.Market, positions: "[]",
			body: `{"symbol":"BTC_USDT","vol":1,"side":2,"sideType":"buy","type":5,"openType":1,"stpMode":0}`,
		},
		{
			name: "market close of an isolated short", side: order.Buy, orderType: order.Market, positions: "[" + isolatedShort + "]",
			body: `{"symbol":"BTC_USDT","vol":1,"side":2,"sideType":"buy","type":5,"openType":1,"positionId":1109973833,"stpMode":0}`,
		},
		{
			name: "market close of a cross short", side: order.Buy, orderType: order.Market, positions: "[" + crossShort + "]",
			body: `{"symbol":"BTC_USDT","vol":1,"side":2,"sideType":"buy","type":5,"openType":2,"positionId":1109973834,"stpMode":0}`,
		},
		{
			name: "limit close of no short", side: order.Buy, orderType: order.Limit, positions: "[]",
			body: `{"symbol":"BTC_USDT","price":100.5,"vol":1,"side":2,"sideType":"buy","type":1,"openType":1,"stpMode":0}`,
		},
		{
			name: "limit close of an isolated short", side: order.Buy, orderType: order.Limit, positions: "[" + isolatedShort + "]",
			body: `{"symbol":"BTC_USDT","price":100.5,"vol":1,"side":2,"sideType":"buy","type":1,"openType":1,"positionId":1109973833,"stpMode":0}`,
		},
		{
			name: "limit close of a cross short", side: order.Buy, orderType: order.Limit, positions: "[" + crossShort + "]",
			body: `{"symbol":"BTC_USDT","price":100.5,"vol":1,"side":2,"sideType":"buy","type":1,"openType":2,"positionId":1109973834,"stpMode":0}`,
		},
		{
			name: "market close of no long", side: order.Sell, orderType: order.Market, positions: "[]",
			body: `{"symbol":"BTC_USDT","vol":1,"side":4,"sideType":"sell","type":5,"openType":1,"stpMode":0}`,
		},
		{
			name: "market close of an isolated long", side: order.Sell, orderType: order.Market, positions: "[" + isolatedLong + "]",
			body: `{"symbol":"BTC_USDT","vol":1,"side":4,"sideType":"sell","type":5,"openType":1,"positionId":1109973831,"stpMode":0}`,
		},
		{
			name: "market close of a cross long", side: order.Sell, orderType: order.Market, positions: "[" + crossLong + "]",
			body: `{"symbol":"BTC_USDT","vol":1,"side":4,"sideType":"sell","type":5,"openType":2,"positionId":1109973832,"stpMode":0}`,
		},
		{
			name: "limit close of no long", side: order.Sell, orderType: order.Limit, positions: "[]",
			body: `{"symbol":"BTC_USDT","price":100.5,"vol":1,"side":4,"sideType":"sell","type":1,"openType":1,"stpMode":0}`,
		},
		{
			name: "limit close of an isolated long", side: order.Sell, orderType: order.Limit, positions: "[" + isolatedLong + "]",
			body: `{"symbol":"BTC_USDT","price":100.5,"vol":1,"side":4,"sideType":"sell","type":1,"openType":1,"positionId":1109973831,"stpMode":0}`,
		},
		{
			name: "limit close of a cross long", side: order.Sell, orderType: order.Limit, positions: "[" + crossLong + "]",
			body: `{"symbol":"BTC_USDT","price":100.5,"vol":1,"side":4,"sideType":"sell","type":1,"openType":2,"positionId":1109973832,"stpMode":0}`,
		},
	} {
		for positionMode, body := range map[string]string{
			"1": tc.body,
			"2": strings.Replace(tc.body, `"stpMode":0}`, `"positionMode":2,"reduceOnly":true,"stpMode":0}`, 1),
		} {
			t.Run("position mode "+positionMode+" "+tc.name, func(t *testing.T) {
				t.Parallel()
				handler, requests := record(futuresHandler(t,
					futuresRoute{method: http.MethodGet, path: "private/position/position_mode", data: positionMode},
					futuresRoute{method: http.MethodGet, path: "private/position/open_positions", query: "symbol=BTC_USDT", data: tc.positions},
					futuresRoute{method: http.MethodPost, path: "private/order/create", body: body, data: placed},
				))
				ex := testExchangeFor(t, handler, liveUnavailable)
				resp, err := submit(ex, order.Submit{Side: tc.side, Type: tc.orderType, ReduceOnly: true})
				require.NoError(t, err, "submitFuturesOrder must not error")
				assert.Equal(t, "739113577038255618", resp.OrderID, "submitFuturesOrder should return the placed order's ID")
				sent, _ := requests()
				assert.Equal(t, []string{readMode, readPositions, place}, sent, "a reducing order should read the position mode once, then its position, before it is placed")
			})
		}
	}

	// Whatever else a reducing order names, both position modes send it with its own type, side, price and closing
	// position, the one-way body adding only positionMode 2 and reduceOnly, and both fail it alike. One exchange serves
	// each subtest's submissions in turn, as one for each would double the package's race run
	for _, side := range []order.Side{order.Buy, order.Sell} {
		isolatedID, crossID, wantSide, wantSideType := "1109973831", "1109973832", "4", "sell"
		closing, other := isolatedLong, crossLong
		if side == order.Buy {
			isolatedID, crossID, wantSide, wantSideType = "1109973833", "1109973834", "2", "buy"
			closing, other = isolatedShort, crossShort
		}
		for _, kind := range []struct {
			name        string
			orderType   order.Type
			timeInForce order.TimeInForce
			wantType    string
		}{
			{"market", order.Market, order.UnknownTIF, "5"},
			{"limit", order.Limit, order.UnknownTIF, "1"},
			{"post-only", order.Limit, order.PostOnly, "2"},
			{"immediate-or-cancel", order.Limit, order.ImmediateOrCancel, "3"},
			{"fill-or-kill", order.Limit, order.FillOrKill, "4"},
		} {
			for marginName, marginType := range map[string]margin.Type{"unnamed margin": margin.Unset, "isolated": margin.Isolated, "cross": margin.Multi} {
				t.Run("position modes agree "+side.String()+" "+kind.name+" "+marginName, func(t *testing.T) {
					t.Parallel()
					var serving atomic.Pointer[http.HandlerFunc]
					ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) { (*serving.Load())(w, r) }, liveUnavailable)
					for _, venue := range []struct {
						name      string
						positions string
						held      string
						refuse    string
					}{
						{name: "no position", positions: "[]"},
						{name: "isolated position", positions: "[" + closing + "]", held: "isolated"},
						{name: "cross position", positions: "[" + other + "]", held: "cross"},
						{name: "positions refused", refuse: "private/position/open_positions"},
						{name: "placement refused", positions: "[]", refuse: "private/order/create"},
					} {
						send := func(positionMode string) ([]string, []byte, error) {
							handler := futuresHandler(t,
								futuresRoute{method: http.MethodGet, path: "private/position/position_mode", data: positionMode},
								futuresRoute{method: http.MethodGet, path: "private/position/open_positions", query: "symbol=BTC_USDT", data: venue.positions},
								futuresRoute{method: http.MethodPost, path: "private/order/create", data: placed},
							)
							if venue.refuse != "" {
								handler = refuse(venue.refuse, handler)
							}
							handler, requests := record(handler)
							serving.Store(&handler)
							_, err := submit(ex, order.Submit{Side: side, Type: kind.orderType, TimeInForce: kind.timeInForce, MarginType: marginType, ReduceOnly: true})
							sent, body := requests()
							return sent, body, err
						}
						hedgeRequests, hedgeBody, hedgeErr := send("1")
						oneWayRequests, oneWayBody, oneWayErr := send("2")
						wantRequests := []string{readMode, readPositions, place}
						if venue.refuse != "" {
							require.ErrorIsf(t, hedgeErr, errFuturesRequestFailed, "with %s, the refusal must be returned in hedge mode", venue.name)
							require.ErrorIsf(t, oneWayErr, errFuturesRequestFailed, "with %s, the refusal must be returned in one-way mode", venue.name)
							assert.Equalf(t, hedgeErr.Error(), oneWayErr.Error(), "with %s, both position modes should return the same error", venue.name)
							if venue.refuse == "private/position/open_positions" {
								wantRequests = wantRequests[:2]
							}
						} else {
							require.NoErrorf(t, hedgeErr, "with %s, submitFuturesOrder must not error in hedge mode", venue.name)
							require.NoErrorf(t, oneWayErr, "with %s, submitFuturesOrder must not error in one-way mode", venue.name)
						}
						assert.Equalf(t, wantRequests, hedgeRequests, "with %s, a hedge mode reducing order should read the mode, then its position, before it is placed", venue.name)
						assert.Equalf(t, wantRequests, oneWayRequests, "with %s, a one-way mode reducing order should read the mode, then its position, before it is placed", venue.name)
						if venue.refuse == "private/position/open_positions" {
							continue
						}
						// A position of the margin type named, or of either when none is, is named and lends its margin type
						openType := map[margin.Type]string{margin.Unset: "1", margin.Isolated: "1", margin.Multi: "2"}[marginType]
						positionID := ""
						switch {
						case venue.held == "isolated" && marginType != margin.Multi:
							openType, positionID = "1", isolatedID
						case venue.held == "cross" && marginType != margin.Isolated:
							openType, positionID = "2", crossID
						}
						exp := map[string]any{
							"symbol":   "BTC_USDT",
							"vol":      exactNumber("1"),
							"side":     exactNumber(wantSide),
							"sideType": wantSideType,
							"type":     exactNumber(kind.wantType),
							"openType": exactNumber(openType),
							"stpMode":  exactNumber("0"),
						}
						if kind.orderType == order.Limit {
							exp["price"] = exactNumber("201/2")
						}
						if positionID != "" {
							exp["positionId"] = exactNumber(positionID)
						}
						hedge, err := exactJSON(hedgeBody)
						require.NoErrorf(t, err, "with %s, the hedge mode body must be JSON", venue.name)
						assert.Equalf(t, exp, hedge, "with %s, the hedge mode body should be sent", venue.name)
						exp["positionMode"], exp["reduceOnly"] = exactNumber("2"), true
						oneWay, err := exactJSON(oneWayBody)
						require.NoErrorf(t, err, "with %s, the one-way mode body must be JSON", venue.name)
						assert.Equalf(t, exp, oneWay, "with %s, the one-way mode body should add positionMode 2 and reduceOnly", venue.name)
					}
				})
			}
		}
	}

	// A reducing order makes no other request where the position mode can't be read or is neither hedge nor one-way
	for positionMode, exp := range map[string]error{
		"refused": errFuturesRequestFailed,
		"0":       errInvalidPositionMode,
		"3":       errInvalidPositionMode,
		"4":       errInvalidPositionMode,
		"255":     errInvalidPositionMode,
		"null":    errInvalidPositionMode,
	} {
		t.Run("position mode "+positionMode, func(t *testing.T) {
			t.Parallel()
			handler := futuresHandler(t, futuresRoute{method: http.MethodGet, path: "private/position/position_mode", data: positionMode})
			if positionMode == "refused" {
				handler = refuse("private/position/position_mode", futuresHandler(t))
			}
			var serving atomic.Pointer[http.HandlerFunc]
			ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) { (*serving.Load())(w, r) }, liveUnavailable)
			for _, side := range []order.Side{order.Buy, order.Sell} {
				for _, orderType := range []order.Type{order.Market, order.Limit} {
					for marginName, marginType := range map[string]margin.Type{"unnamed margin": margin.Unset, "isolated": margin.Isolated, "cross": margin.Multi} {
						recorded, requests := record(handler)
						serving.Store(&recorded)
						_, err := submit(ex, order.Submit{Side: side, Type: orderType, MarginType: marginType, ReduceOnly: true})
						assert.ErrorIsf(t, err, exp, "a %s %s order with %s should be stopped by the position mode", side, orderType, marginName)
						sent, _ := requests()
						assert.Equalf(t, []string{readMode}, sent, "a %s %s order with %s should make no request after the position mode stops it", side, orderType, marginName)
					}
				}
			}
		})
	}

	// A reducing order is not sent in either position mode when its matching position has no ID or two positions match
	// it. The venue holds one margin type a side (its error 2027), so the isolated and cross positions are a reply it
	// shouldn't send
	for _, positionMode := range []string{"1", "2"} {
		for _, tc := range []struct {
			name      string
			positions string
			err       error
		}{
			{name: "position without an ID", positions: "[" + position("0", "1", "2") + "]", err: errInvalidFuturesID},
			{name: "isolated and cross positions", positions: "[" + isolatedLong + "," + crossLong + "]", err: errMarginTypeAmbiguous},
		} {
			t.Run("position mode "+positionMode+" "+tc.name, func(t *testing.T) {
				t.Parallel()
				handler, requests := record(futuresHandler(t,
					futuresRoute{method: http.MethodGet, path: "private/position/position_mode", data: positionMode},
					futuresRoute{method: http.MethodGet, path: "private/position/open_positions", query: "symbol=BTC_USDT", data: tc.positions},
				))
				ex := testExchangeFor(t, handler, liveUnavailable)
				_, err := submit(ex, order.Submit{Side: order.Sell, Type: order.Market, ReduceOnly: true})
				assert.ErrorIs(t, err, tc.err, "submitFuturesOrder should return why the closing position stops the order")
				sent, _ := requests()
				assert.Equal(t, []string{readMode, readPositions}, sent, "a reducing order should not be placed when its position can't be named")
			})
		}
	}

	for _, tc := range []struct {
		name    string
		side    order.Side
		handler func(*testing.T) http.HandlerFunc
		err     error
	}{
		{
			name:    "unknown side",
			side:    order.AnySide,
			handler: func(t *testing.T) http.HandlerFunc { t.Helper(); return futuresHandler(t) },
			err:     order.ErrSideIsInvalid,
		},
		{
			name: "order refused",
			side: order.Buy,
			handler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return refuse("private/order/create", futuresHandler(t))
			},
			err: errFuturesRequestFailed,
		},
		{
			name: "order without an ID",
			side: order.Buy,
			handler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/order/create", data: `{}`})
			},
			err: common.ErrNoResponse,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := testExchangeFor(t, tc.handler(t), liveUnavailable)
			_, err := submit(ex, order.Submit{Side: tc.side, Type: order.Market, MarginType: margin.Multi})
			assert.ErrorIs(t, err, tc.err, "submitFuturesOrder should return why the order was not placed")
		})
	}
}

func TestCancelOrderFutures(t *testing.T) {
	t.Parallel()
	cancel := func(ex *Exchange, c order.Cancel) error {
		c.AssetType = asset.Futures
		return ex.CancelOrder(t.Context(), &c)
	}
	require.ErrorIs(t, cancel(e, order.Cancel{}), order.ErrOrderIDNotSet)
	require.ErrorIs(t, cancel(e, order.Cancel{ClientOrderID: "gct-1"}), currency.ErrCurrencyPairEmpty, "a client order ID must be named with its pair")
	require.ErrorIs(t, cancel(e, order.Cancel{OrderID: "abc"}), errInvalidFuturesID)

	const orderID = "739106551624717312"
	for name, tc := range map[string]struct {
		data string
		err  error
	}{
		"cancelled": {data: `[{"orderId":739106551624717312,"errorCode":0,"errorMsg":"success"}]`},
		"refused":   {data: `[{"orderId":739106551624717312,"errorCode":2040,"errorMsg":"order not exist"}]`, err: errFuturesRequestFailed},
		"missing":   {data: `[]`, err: errCancelOutcomeMissing},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var sent []byte
			replay := futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/order/cancel", body: `[739106551624717312]`, data: tc.data})
			ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
				var err error
				sent, err = io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the request body should not error")
				r.Body = io.NopCloser(bytes.NewReader(sent))
				replay(w, r)
			}, liveUnavailable)
			err := cancel(ex, order.Cancel{OrderID: orderID})
			require.ErrorIs(t, err, tc.err, "the cancellation's outcome must be returned")
			assert.Equal(t, "["+orderID+"]", string(sent), "the order ID should be sent exactly, as a number")
		})
	}

	for data, exp := range map[string]error{
		`[{"orderId":739106551624717312,"errorCode":0,"errorMsg":"success"}]`:            nil,
		`{"orderId":739106551624717312,"errorCode":0,"errorMsg":"success"}`:              nil,
		`[{"orderId":739106551624717312,"errorCode":2040,"errorMsg":"order not exist"}]`: errFuturesRequestFailed,
		`[]`:   errCancelOutcomeMissing,
		`null`: errCancelOutcomeMissing,
	} {
		ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
			method: http.MethodPost,
			path:   "private/order/cancel_with_external",
			body:   `{"symbol":"BTC_USDT","externalOid":"gct-1"}`,
			data:   data,
		}), liveUnavailable)
		assert.ErrorIsf(t, cancel(ex, order.Cancel{ClientOrderID: "gct-1", Pair: futuresTradablePair}), exp, "a cancellation by client order ID answered %s should report its outcome", data)
	}
}

func TestCancelBatchOrdersFutures(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, OrderID: "x"}})
	require.ErrorIs(t, err, errInvalidFuturesID)

	// 51 orders take two requests, of 50 orders and of one
	orders := make([]order.Cancel, futuresBatchLimit+1)
	firstIDs, firstOutcomes := make([]string, futuresBatchLimit), make([]string, futuresBatchLimit)
	for i := range orders {
		id := strconv.Itoa(1000 + i)
		orders[i] = order.Cancel{OrderID: id, AssetType: asset.Futures, Pair: futuresTradablePair}
		if i < futuresBatchLimit {
			firstIDs[i] = id
			firstOutcomes[i] = `{"orderId":` + id + `,"errorCode":0}`
		}
	}
	var requests atomic.Int32
	replay := futuresHandler(t, futuresRoute{
		method: http.MethodPost,
		path:   "private/order/cancel",
		body:   "[" + strings.Join(firstIDs, ",") + "]",
		data:   "[" + strings.Join(firstOutcomes, ",") + "]",
	}, futuresRoute{
		method: http.MethodPost,
		path:   "private/order/cancel",
		body:   `[1050]`,
		data:   `[{"orderId":1050,"errorCode":2041,"errorMsg":"order state cannot be cancelled"}]`,
	})
	ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		replay(w, r)
	}, liveUnavailable)
	resp, err := ex.CancelBatchOrders(t.Context(), orders)
	require.NoError(t, err, "a refused cancellation must be reported in its status, not as an error")
	assert.EqualValues(t, 2, requests.Load(), "the orders should be cancelled 50 at a time")
	// A refused order carries the venue's reason
	exp := &order.CancelBatchResponse{Status: map[string]string{"1050": "order state cannot be cancelled"}}
	for _, id := range firstIDs {
		exp.Status[id] = order.Cancelled.String()
	}
	assert.Equal(t, exp, resp, "every order should carry its outcome")
}

func TestCancelAllOrdersFutures(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		pair currency.Pair
		body string
	}{
		"contract":  {pair: futuresTradablePair, body: `{"symbol":"BTC_USDT"}`},
		"account":   {body: `{"symbol":""}`},
		"formatted": {pair: currency.NewBTCUSDT(), body: `{"symbol":"BTC_USDT"}`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex := testExchangeFor(t, futuresHandler(t, futuresRoute{method: http.MethodPost, path: "private/order/cancel_all", body: tc.body}), liveUnavailable)
			resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures, Pair: tc.pair})
			require.NoError(t, err, "CancelAllOrders must not error for futures")
			assert.Equal(t, order.CancelAllResponse{Status: map[string]string{}}, resp, "the venue names no orders, so none should be reported")
		})
	}
}

func TestGetOrderInfoFutures(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "", futuresTradablePair, asset.Futures)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	ex := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/get/739106551624717312",
		data:   futuresFilledOrderReply,
	}, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/deal_details/739106551624717312",
		data: `[{"id":"9466823344","symbol":"BTC_USDT","side":1,"vol":5,"price":109777.5,"fee":0.0329,"feeCurrency":"USDT",` +
			`"profit":0,"category":1,"orderId":"739106551624717312","timestamp":1761887134000,"taker":true}]`,
	}), liveUnavailable)
	detail, err := ex.GetOrderInfo(t.Context(), "739106551624717312", futuresTradablePair, asset.Futures)
	require.NoError(t, err, "GetOrderInfo must not error for futures")
	// An opening long is a buy, its amounts are in contracts and its cost is their value in the quote currency
	exp := futuresFilledOrderDetail(ex.Name)
	exp.Trades = []order.TradeHistory{{
		Price:     109777.5,
		Amount:    5,
		Fee:       0.0329,
		Exchange:  ex.Name,
		TID:       "9466823344",
		Side:      order.Buy,
		Timestamp: time.UnixMilli(1761887134000),
		FeeAsset:  "USDT",
	}}
	assert.Equal(t, exp, detail, "the order and its fill should be mapped")
}

func TestGetActiveOrdersFutures(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/list/open_orders",
		query:  "page_num=1&page_size=100",
		data: `[{"orderId":"750330744886358528","symbol":"ETH_USDT","price":2500,"vol":10,"leverage":1,"side":1,"category":1,` +
			`"orderType":1,"dealVol":0,"openType":2,"state":2,"createTime":1764563189993,"updateTime":1764563190059},` +
			`{"orderId":"750330663772702720","symbol":"BTC_USDT","price":80000,"vol":10,"leverage":1,"side":1,"category":1,` +
			`"orderType":1,"dealVol":0,"openType":2,"state":2,"createTime":1764563170653,"updateTime":1764563170723}]`,
	}), liveReadOnly)
	orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Pairs: currency.Pairs{futuresTradablePair}, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error for futures")
	if !mockTests {
		return
	}
	// Only the pair asked for is returned; an unfilled cross limit order is new, multi margined and rests until
	// cancelled, with every contract remaining
	exp := order.FilteredOrders{{
		TimeInForce:     order.GoodTillCancel,
		Leverage:        1,
		Price:           80000,
		Amount:          10,
		ContractAmount:  10,
		RemainingAmount: 10,
		Exchange:        ex.Name,
		OrderID:         "750330663772702720",
		Type:            order.Limit,
		Side:            order.Buy,
		Status:          order.New,
		AssetType:       asset.Futures,
		Date:            time.UnixMilli(1764563170653),
		LastUpdated:     time.UnixMilli(1764563170723),
		Pair:            futuresTradablePair,
		MarginType:      margin.Multi,
	}}
	assert.Equal(t, exp, orders, "the pair's order should be mapped")
}

// futuresOrderPage returns a page of open orders with consecutive IDs
func futuresOrderPage(firstID, n int) string {
	page := make([]string, n)
	for i := range page {
		page[i] = `{"orderId":"` + strconv.Itoa(firstID+i) + `","symbol":"BTC_USDT","price":80000,"vol":1,"side":1,"category":1,` +
			`"orderType":1,"openType":2,"state":2,"createTime":1758000000000}`
	}
	return "[" + strings.Join(page, ",") + "]"
}

func TestGetActiveOrdersFuturesPaging(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		second string
		exp    int
	}{
		"next page": {second: futuresOrderPage(6000, 20), exp: futuresDefaultPageSizeLimit + 20},
		// A page number the venue ignores serves the first page again, which must end the paging
		"repeated page": {second: futuresOrderPage(5000, futuresDefaultPageSizeLimit), exp: futuresDefaultPageSizeLimit},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
				method: http.MethodGet, path: "private/order/list/open_orders", query: "page_num=1&page_size=100", data: futuresOrderPage(5000, futuresDefaultPageSizeLimit),
			}, futuresRoute{
				method: http.MethodGet, path: "private/order/list/open_orders", query: "page_num=2&page_size=100", data: tc.second,
			}), liveUnavailable)
			orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
			require.NoError(t, err, "GetActiveOrders must not error")
			assert.Len(t, orders, tc.exp, "every order should be returned once")
		})
	}
}

func TestGetOrderHistoryFuturesPaging(t *testing.T) {
	t.Parallel()
	const start, end int64 = 1757900000000, 1758100000000
	query := func(page int) string {
		return "end_time=" + strconv.FormatInt(end, 10) + "&page_num=" + strconv.Itoa(page) + "&page_size=100&start_time=" + strconv.FormatInt(start, 10) + "&states=3%2C4%2C5&symbol=BTC_USDT"
	}
	for name, tc := range map[string]struct {
		second string
		exp    int
	}{
		"next page": {second: futuresOrderPage(6000, 1), exp: futuresDefaultPageSizeLimit + 1},
		// A page number the venue ignores serves the first page again, which must end the paging
		"repeated page": {second: futuresOrderPage(5000, futuresDefaultPageSizeLimit), exp: futuresDefaultPageSizeLimit},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
				method: http.MethodGet, path: "private/order/list/history_orders", query: query(1), data: futuresOrderPage(5000, futuresDefaultPageSizeLimit),
			}, futuresRoute{
				method: http.MethodGet, path: "private/order/list/history_orders", query: query(2), data: tc.second,
			}), liveUnavailable)
			orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
				AssetType: asset.Futures,
				Pairs:     currency.Pairs{futuresTradablePair},
				Side:      order.AnySide,
				Type:      order.AnyType,
				StartTime: time.UnixMilli(start),
				EndTime:   time.UnixMilli(end),
			})
			require.NoError(t, err, "GetOrderHistory must not error")
			assert.Len(t, orders, tc.exp, "every order should be returned once")
		})
	}
}

// fundingRatePage returns a page of settled funding rates, newest first, settled 8 hours apart from the newest
func fundingRatePage(newest time.Time, first, n int) string {
	rates := make([]string, n)
	for i := range rates {
		rates[i] = `{"symbol":"BTC_USDT","fundingRate":0.0001,"collectCycle":8,"settleTime":` + strconv.FormatInt(newest.Add(-time.Duration(first+i)*8*time.Hour).UnixMilli(), 10) + `}`
	}
	return `{"pageSize":1000,"currentPage":1,"resultList":[` + strings.Join(rates, ",") + `]}`
}

func TestGetHistoricalFundingRatesPaging(t *testing.T) {
	t.Parallel()
	newest := time.UnixMilli(1792800000000).UTC()
	settled := func(i int) time.Time { return newest.Add(-time.Duration(i) * 8 * time.Hour) }
	page := func(n int, data string) futuresRoute {
		return futuresRoute{method: http.MethodGet, path: "contract/funding_rate/history", query: "page_num=" + strconv.Itoa(n) + "&page_size=1000&symbol=BTC_USDT", data: data}
	}
	for name, tc := range map[string]struct {
		start, end time.Time
		routes     []futuresRoute
		count      int
		oldest     time.Time
		latest     time.Time
	}{
		"latest": {
			routes: []futuresRoute{page(1, fundingRatePage(newest, 0, 1000))},
			count:  1000, oldest: settled(999), latest: settled(0),
		},
		"start on the second page": {
			start: settled(1500), end: settled(0),
			routes: []futuresRoute{page(1, fundingRatePage(newest, 0, 1000)), page(2, fundingRatePage(newest, 1000, 1000))},
			count:  1501, oldest: settled(1500), latest: settled(0),
		},
		"end on the second page": {
			end:    settled(1200),
			routes: []futuresRoute{page(1, fundingRatePage(newest, 0, 1000)), page(2, fundingRatePage(newest, 1000, 1000)), page(3, fundingRatePage(newest, 2000, 0))},
			count:  800, oldest: settled(1999), latest: settled(1200),
		},
		"end before a full page": {
			end:    settled(500),
			routes: []futuresRoute{page(1, fundingRatePage(newest, 0, 1000)), page(2, fundingRatePage(newest, 1000, 1000))},
			count:  1000, oldest: settled(1499), latest: settled(500),
		},
		// A page number the venue ignores serves the first page again, which must end the paging
		"repeated page": {
			start: settled(1500), end: settled(0),
			routes: []futuresRoute{page(1, fundingRatePage(newest, 0, 1000)), page(2, fundingRatePage(newest, 0, 1000))},
			count:  1000, oldest: settled(999), latest: settled(0),
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex := testExchangeFor(t, futuresHandler(t, tc.routes...), liveUnavailable)
			result, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: futuresTradablePair, StartDate: tc.start, EndDate: tc.end})
			require.NoError(t, err, "GetHistoricalFundingRates must not error")
			require.Len(t, result.FundingRates, tc.count, "the rates asked for must be returned")
			assert.Equal(t, tc.oldest, result.FundingRates[0].Time.UTC(), "the rates should start at the oldest asked for")
			assert.Equal(t, tc.latest, result.LatestRate.Time.UTC(), "the latest rate should be the newest asked for")
		})
	}
}

func TestGetOrderHistoryFutures(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	// A 100 day window is read as 90 days and 10, and an order on the boundary is returned once
	const start, middle, end int64 = 1750000000000, 1757776000000, 1758640000000
	window := func(from, to int64) string {
		return "end_time=" + strconv.FormatInt(to, 10) + "&page_num=1&page_size=100&start_time=" + strconv.FormatInt(from, 10) + "&states=3%2C4%2C5&symbol=BTC_USDT"
	}
	boundary := `{"orderId":"739211854408619008","symbol":"BTC_USDT","price":0,"vol":1,"side":3,"category":1,"orderType":5,` +
		`"dealAvgPrice":100000,"dealVol":1,"openType":1,"state":3,"createTime":1757776000000,"updateTime":1757776000000}`
	ex := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/list/history_orders",
		query:  window(start, middle),
		data:   "[" + boundary + "]",
	}, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/list/history_orders",
		query:  window(middle, end),
		data: "[" + boundary + `,{"orderId":"739211257974395392","symbol":"BTC_USDT","price":90000,"vol":1,"side":2,` +
			`"category":1,"orderType":1,"dealVol":0,"openType":1,"state":4,"createTime":1758000000000,"updateTime":1758000001000}]`,
	}), liveUnavailable)
	orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Futures,
		Pairs:     currency.Pairs{futuresTradablePair},
		Side:      order.AnySide,
		Type:      order.AnyType,
		StartTime: time.UnixMilli(start),
		EndTime:   time.UnixMilli(end),
	})
	require.NoError(t, err, "GetOrderHistory must not error for futures")
	// An opening short is a sell, and a closing order reduces only
	exp := order.FilteredOrders{
		{
			Amount:               1,
			ContractAmount:       1,
			AverageExecutedPrice: 100000,
			ExecutedAmount:       1,
			Cost:                 10,
			CostAsset:            currency.USDT,
			Exchange:             ex.Name,
			OrderID:              "739211854408619008",
			Type:                 order.Market,
			Side:                 order.Sell,
			Status:               order.Filled,
			AssetType:            asset.Futures,
			Date:                 time.UnixMilli(1757776000000),
			LastUpdated:          time.UnixMilli(1757776000000),
			Pair:                 futuresTradablePair,
			MarginType:           margin.Isolated,
		},
		{
			TimeInForce:     order.GoodTillCancel,
			ReduceOnly:      true,
			Price:           90000,
			Amount:          1,
			ContractAmount:  1,
			RemainingAmount: 1,
			Exchange:        ex.Name,
			OrderID:         "739211257974395392",
			Type:            order.Limit,
			Side:            order.Buy,
			Status:          order.Cancelled,
			AssetType:       asset.Futures,
			Date:            time.UnixMilli(1758000000000),
			LastUpdated:     time.UnixMilli(1758000001000),
			Pair:            futuresTradablePair,
			MarginType:      margin.Isolated,
		},
	}
	assert.Equal(t, exp, orders, "each order should be mapped once")
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.Spot, spotTradablePair, margin.Isolated, 2, order.Buy), futures.ErrNotFuturesAsset)
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.Futures, futuresTradablePair, margin.Isolated, 0, order.Buy), errInvalidLeverage)
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.Futures, futuresTradablePair, margin.Isolated, 2.5, order.Buy), errInvalidLeverage)
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.Futures, futuresTradablePair, margin.SpotIsolated, 2, order.Buy), margin.ErrMarginTypeUnsupported)
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.Futures, futuresTradablePair, margin.Isolated, 2, order.ClosePosition), order.ErrSideIsInvalid)

	// The venue asks for an open position to be named by its ID; a side without one has its leverage set by contract
	positions := `[` + futuresOpenPositionReply + `,` + strings.NewReplacer(`"positionId":1109973831`, `"positionId":1109973832`, `"openType":1`, `"openType":2`).Replace(futuresOpenPositionReply) + `]`
	for name, tc := range map[string]struct {
		marginType margin.Type
		side       order.Side
		bodies     []string
	}{
		"isolated long position": {margin.Isolated, order.Buy, []string{`{"positionId":1109973831,"leverage":5}`}},
		"isolated sides":         {margin.Isolated, order.UnknownSide, []string{`{"positionId":1109973831,"leverage":5}`, `{"leverage":5,"openType":1,"symbol":"BTC_USDT","positionType":2}`}},
		"cross long position":    {margin.Multi, order.Long, []string{`{"positionId":1109973832,"leverage":5}`}},
		"cross short":            {margin.Multi, order.Sell, []string{`{"leverage":5,"openType":2,"symbol":"BTC_USDT","positionType":2}`}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var sent []string
			routes := []futuresRoute{{method: http.MethodGet, path: "private/position/open_positions", query: "symbol=BTC_USDT", data: positions}}
			for _, body := range tc.bodies {
				routes = append(routes, futuresRoute{method: http.MethodPost, path: "private/position/change_leverage", body: body})
			}
			replay := futuresHandler(t, routes...)
			ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the request body should not error")
					r.Body = io.NopCloser(bytes.NewReader(body))
					mu.Lock()
					sent = append(sent, string(body))
					mu.Unlock()
				}
				replay(w, r)
			}, liveUnavailable)
			require.NoError(t, ex.SetLeverage(t.Context(), asset.Futures, futuresTradablePair, tc.marginType, 5, tc.side), "SetLeverage must not error")
			assert.Len(t, sent, len(tc.bodies), "each side named should have its leverage set once")
		})
	}

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/open_positions",
		data:   `[` + strings.Replace(futuresOpenPositionReply, `"positionId":1109973831`, `"positionId":0`, 1) + `]`,
	}), liveUnavailable)
	assert.ErrorIs(t, ex.SetLeverage(t.Context(), asset.Futures, futuresTradablePair, margin.Isolated, 5, order.Buy), errInvalidFuturesID, "a position without an ID should not be guessed")
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverage(t.Context(), asset.Spot, spotTradablePair, margin.Isolated, order.Buy)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = e.GetLeverage(t.Context(), asset.Futures, currency.EMPTYPAIR, margin.Isolated, order.Buy)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	ex := testExchangeFor(t, futuresHandler(t, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/leverage",
		query:  "symbol=BTC_USDT",
		data: `[{"positionType":1,"openType":1,"leverage":20},{"positionType":2,"openType":1,"leverage":10},` +
			`{"positionType":1,"openType":2,"leverage":5},{"positionType":2,"openType":2,"leverage":5}]`,
	}), liveUnavailable)
	for _, tc := range []struct {
		marginType margin.Type
		side       order.Side
		exp        float64
	}{
		{margin.Isolated, order.Buy, 20},
		{margin.Isolated, order.Short, 10},
		{margin.Multi, order.UnknownSide, 5},
	} {
		leverage, err := ex.GetLeverage(t.Context(), asset.Futures, futuresTradablePair, tc.marginType, tc.side)
		require.NoErrorf(t, err, "GetLeverage must not error for %s %s", tc.marginType, tc.side)
		assert.Equalf(t, tc.exp, leverage, "%s %s should have its own leverage", tc.marginType, tc.side)
	}
	_, err = ex.GetLeverage(t.Context(), asset.Futures, futuresTradablePair, margin.Isolated, order.UnknownSide)
	assert.ErrorIs(t, err, errInvalidLeverage, "sides of different leverage should need a side named")
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Spot, Pair: spotTradablePair})
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	unformatted := new(Exchange)
	unformatted.CurrencyPairs.UseGlobalFormat = true
	_, err = unformatted.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: futuresTradablePair})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetFuturesPositionSummary must refuse a pair it cannot format")

	ex := testExchangeFor(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"code":401,"message":"Not logged in"}`))
	}, liveUnavailable)
	_, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: futuresTradablePair})
	require.ErrorIs(t, err, errFuturesRequestFailed, "GetFuturesPositionSummary must return a refused positions request")

	ex = testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/open_positions",
		query:  "symbol=ETH_USDT",
		data:   "[" + strings.Replace(futuresOpenPositionReply, `"symbol":"BTC_USDT"`, `"symbol":"ETH_USDT"`, 1) + "]",
	}), liveUnavailable)
	_, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter)})
	require.ErrorIs(t, err, errContractNotFound, "GetFuturesPositionSummary must refuse a position on a contract the venue does not list")

	ex = testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/open_positions",
		query:  "symbol=BTC_USDT",
		data:   "[" + futuresOpenPositionReply + "]",
	}), liveUnavailable)
	summary, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: futuresTradablePair})
	require.NoError(t, err, "GetFuturesPositionSummary must not error")
	// A long position's size is positive, its notional size is its value, and an isolated position reports its margin
	exp := &futures.PositionSummary{
		Pair:                      futuresTradablePair,
		Asset:                     asset.Futures,
		MarginType:                margin.Isolated,
		Currency:                  currency.USDT,
		IsolatedMargin:            decimal.MustFromFloat(27.444375),
		NotionalSize:              decimal.MustFromFloat(54.88875),
		Leverage:                  decimal.NewFromInt(2),
		EstimatedLiquidationPrice: decimal.MustFromFloat(55020.5),
		CollateralUsed:            decimal.MustFromFloat(27.444375),
		CurrentSize:               decimal.MustFromFloat(5),
		ContractSize:              decimal.MustFromFloat(0.0001),
		ContractMultiplier:        decimal.MustFromFloat(0.0001),
		ContractSettlementType:    futures.Linear,
		AverageOpenPrice:          decimal.MustFromFloat(109777.5),
		UnrealisedPNL:             decimal.MustFromFloat(-0.0039),
		RealisedPNL:               decimal.MustFromFloat(-0.0329),
	}
	assert.Equal(t, exp, summary, "the position should be summarised")

	ex = testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/open_positions",
		query:  "symbol=BTC_USDT",
		data:   "[" + futuresOpenPositionReply + "," + strings.Replace(futuresOpenPositionReply, `"positionType":1`, `"positionType":2`, 1) + "]",
	}), liveUnavailable)
	_, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: futuresTradablePair})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "both sides open must need a direction")
	summary, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: futuresTradablePair, Direction: order.Short})
	require.NoError(t, err, "GetFuturesPositionSummary must not error with a direction")
	exp.CurrentSize = decimal.MustFromFloat(-5)
	assert.Equal(t, exp, summary, "a short position's size should be negative")

	ex = testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/open_positions",
		query:  "symbol=BTC_USD",
		data: `[{"positionId":1109973840,"symbol":"BTC_USD","positionType":1,"openType":1,"state":1,"holdVol":5,` +
			`"holdAvgPrice":50000,"openAvgPrice":50000,"liquidatePrice":33500,"oim":0.005,"im":0.005,"realised":-0.000003,` +
			`"leverage":2,"createTime":1761887133854,"updateTime":1761887133854,"version":1,"unRealizedPnl":0.0000012}]`,
	}), liveUnavailable)
	summary, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: futuresInversePair})
	require.NoError(t, err, "GetFuturesPositionSummary must not error for an inverse contract")
	// Five contracts of 100 USD at 50000 are worth 0.01 BTC, the settlement currency the summary is in
	exp = &futures.PositionSummary{
		Pair:                      futuresInversePair,
		Asset:                     asset.Futures,
		MarginType:                margin.Isolated,
		Currency:                  currency.BTC,
		IsolatedMargin:            decimal.MustFromFloat(0.005),
		NotionalSize:              decimal.MustFromFloat(0.01),
		Leverage:                  decimal.NewFromInt(2),
		EstimatedLiquidationPrice: decimal.MustFromFloat(33500),
		CollateralUsed:            decimal.MustFromFloat(0.005),
		CurrentSize:               decimal.MustFromFloat(5),
		ContractSize:              decimal.MustFromFloat(100),
		ContractMultiplier:        decimal.MustFromFloat(100),
		ContractSettlementType:    futures.Inverse,
		AverageOpenPrice:          decimal.MustFromFloat(50000),
		UnrealisedPNL:             decimal.MustFromFloat(0.0000012),
		RealisedPNL:               decimal.MustFromFloat(-0.000003),
	}
	assert.Equal(t, exp, summary, "an inverse position should be summarised in its settlement currency")
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.Futures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)
	start := time.UnixMilli(1761850000000)
	_, err = e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{futuresTradablePair}, StartDate: start, EndDate: start.Add(-time.Hour)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	ex := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
		method: http.MethodGet,
		path:   "private/order/list/history_orders",
		query:  "end_time=1761936400000&page_num=1&page_size=100&start_time=1761850000000&states=3%2C4%2C5&symbol=BTC_USDT",
		data:   "[" + futuresFilledOrderReply + "]",
	}), liveUnavailable)
	resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{futuresTradablePair}, StartDate: start, EndDate: start.Add(24 * time.Hour)})
	require.NoError(t, err, "GetFuturesPositionOrders must not error")
	exp := []futures.PositionResponse{{
		Pair:                   futuresTradablePair,
		Asset:                  asset.Futures,
		ContractSettlementType: futures.Linear,
		Orders:                 []order.Detail{*futuresFilledOrderDetail(ex.Name)},
	}}
	assert.Equal(t, exp, resp, "the contract's orders should be mapped")
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Spot, Pair: spotTradablePair})
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: futuresTradablePair, MarginType: margin.Multi, NewAllocatedMargin: 1})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: futuresTradablePair, MarginType: margin.Isolated})
	require.ErrorIs(t, err, margin.ErrNewAllocatedMarginRequired)
	unformatted := new(Exchange)
	unformatted.CurrencyPairs.UseGlobalFormat = true
	_, err = unformatted.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: futuresTradablePair, MarginType: margin.Isolated, NewAllocatedMargin: 1})
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "ChangePositionMargin must refuse a pair it cannot format")

	const (
		add               = `{"positionId":1109973831,"amount":2.555625,"type":"ADD"}`
		addShort          = `{"positionId":1109973832,"amount":2.555625,"type":"ADD"}`
		notLoggedIn       = `{"success":false,"code":401,"message":"Not logged in"}`
		precisionRejected = `{"success":false,"code":2015,"message":"Price or quantity precision error"}`
	)
	bothSides := futuresOpenPositionReply + "," + strings.NewReplacer(`"positionId":1109973831`, `"positionId":1109973832`, `"positionType":1`, `"positionType":2`).Replace(futuresOpenPositionReply)
	for name, tc := range map[string]struct {
		newMargin  float64
		marginSide string
		position   string
		body       string
		refusals   map[string]string
		err        error
		allocated  float64
	}{
		"add":                  {newMargin: 30, position: futuresOpenPositionReply, body: add, allocated: 30},
		"subtract":             {newMargin: 25, position: futuresOpenPositionReply, body: `{"positionId":1109973831,"amount":2.444375,"type":"SUB"}`, allocated: 25},
		"unchanged":            {newMargin: 27.444375, position: futuresOpenPositionReply, allocated: 27.444375},
		"within float64 error": {newMargin: math.Nextafter(27.444375, 30), position: futuresOpenPositionReply, allocated: 27.444375},
		"buy side":             {newMargin: 30, marginSide: "Buy", position: futuresOpenPositionReply, body: add, allocated: 30},
		"long side":            {newMargin: 30, marginSide: "LONG", position: futuresOpenPositionReply, body: add, allocated: 30},
		"short side":           {newMargin: 30, marginSide: "short", position: futuresOpenPositionReply, err: futures.ErrNoPositionsFound},
		"misspelt side":        {newMargin: 30, marginSide: "shrot", err: order.ErrSideIsInvalid},
		"both open, short":     {newMargin: 30, marginSide: "short", position: bothSides, body: addShort, allocated: 30},
		"both open, sell":      {newMargin: 30, marginSide: "Sell", position: bothSides, body: addShort, allocated: 30},
		"both open, buy":       {newMargin: 30, marginSide: "buy", position: bothSides, body: add, allocated: 30},
		"both open, no side":   {newMargin: 30, position: bothSides, err: order.ErrSideIsInvalid},
		"both open, any side":  {newMargin: 30, marginSide: "any", position: bothSides, err: order.ErrSideIsInvalid},
		"position without id":  {newMargin: 30, position: strings.Replace(futuresOpenPositionReply, `"positionId":1109973831`, `"positionId":0`, 1), err: errInvalidFuturesID},
		"positions refused":    {newMargin: 30, refusals: map[string]string{"private/position/open_positions": notLoggedIn}, err: errFuturesRequestFailed},
		"change refused": {
			newMargin: 30, position: futuresOpenPositionReply, body: add,
			refusals: map[string]string{"private/position/change_margin": precisionRejected}, err: errFuturesRequestFailed,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var routes []futuresRoute
			if tc.position != "" {
				routes = append(routes, futuresRoute{method: http.MethodGet, path: "private/position/open_positions", query: "symbol=BTC_USDT", data: "[" + tc.position + "]"})
			}
			if tc.body != "" {
				routes = append(routes, futuresRoute{method: http.MethodPost, path: "private/position/change_margin", body: tc.body})
			}
			serve := futuresHandler(t, routes...)
			var changes atomic.Int32
			ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/private/position/change_margin" {
					changes.Add(1)
				}
				if refusal, ok := tc.refusals[strings.TrimPrefix(r.URL.Path, "/api/v1/")]; ok {
					_, _ = w.Write([]byte(refusal))
					return
				}
				serve(w, r)
			}, liveUnavailable)
			resp, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
				Exchange:           ex.Name,
				Asset:              asset.Futures,
				Pair:               futuresTradablePair,
				MarginType:         margin.Isolated,
				NewAllocatedMargin: tc.newMargin,
				MarginSide:         tc.marginSide,
			})
			var sent int32
			if tc.body != "" {
				sent = 1
			}
			assert.Equal(t, sent, changes.Load(), "a margin change should be sent only when the selected position's margin moves")
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "ChangePositionMargin must return the expected error")
				assert.Nil(t, resp, "no response should be returned with an error")
				return
			}
			require.NoError(t, err, "ChangePositionMargin must not error")
			exp := &margin.PositionChangeResponse{Exchange: ex.Name, Pair: futuresTradablePair, Asset: asset.Futures, AllocatedMargin: tc.allocated, MarginType: margin.Isolated}
			assert.Equal(t, exp, resp, "the position's new margin should be returned")
		})
	}
}

type historicCandlesFailure struct {
	name    string
	handler func(*testing.T) http.HandlerFunc
	// setup readies the exchange for the request, where it needs more than the test exchange holds
	setup func(*testing.T, *Exchange)
	pair  currency.Pair
	a     asset.Item
	err   error
}

func historicCandlesFailures() []historicCandlesFailure {
	return []historicCandlesFailure{
		{
			name: "spot request refused",
			handler: func(*testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, `{"msg":"Invalid symbol.","code":-1121,"_extend":null}`, http.StatusBadRequest)
				}
			},
			pair: spotTradablePair,
			a:    asset.Spot,
			err:  request.ErrBadStatus,
		},
		{
			name: "futures candle fields of unequal lengths",
			handler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return futuresHandler(t, futuresContractsRoute, futuresRoute{
					method: http.MethodGet,
					path:   "contract/kline/BTC_USDT",
					data:   `{"time":[1792800000,1792803600],"open":[1],"close":[1,2],"high":[1,2],"low":[1,2],"vol":[1,2],"amount":[1,2]}`,
				})
			},
			pair: futuresTradablePair,
			a:    asset.Futures,
			err:  errMalformedCandles,
		},
		{
			name: "asset not handled",
			handler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return futuresHandler(t)
			},
			setup: func(t *testing.T, ex *Exchange) {
				t.Helper()
				require.NoError(t, ex.SetAssetPairStore(asset.Margin, currency.PairStore{
					AssetEnabled:  true,
					RequestFormat: &currency.PairFormat{Uppercase: true},
					ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
				}), "SetAssetPairStore must not error")
				for _, enabled := range []bool{false, true} {
					require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Margin, currency.Pairs{spotTradablePair}, enabled), "StorePairs must not error")
				}
			},
			pair: spotTradablePair,
			a:    asset.Margin,
			err:  asset.ErrNotSupported,
		},
	}
}

func TestGetHistoricCandlesFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range historicCandlesFailures() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := testExchangeFor(t, tc.handler(t), liveUnavailable)
			if tc.setup != nil {
				tc.setup(t, ex)
			}
			start, end := futuresWindow()
			_, err := ex.GetHistoricCandles(t.Context(), tc.pair, tc.a, kline.OneHour, start, end)
			assert.ErrorIs(t, err, tc.err, "GetHistoricCandles should return why no candles were returned")
		})
	}
}

func TestGetHistoricCandlesExtendedFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range historicCandlesFailures() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := testExchangeFor(t, tc.handler(t), liveUnavailable)
			if tc.setup != nil {
				tc.setup(t, ex)
			}
			start, end := futuresWindow()
			_, err := ex.GetHistoricCandlesExtended(t.Context(), tc.pair, tc.a, kline.OneHour, start, end)
			assert.ErrorIs(t, err, tc.err, "GetHistoricCandlesExtended should return why no candles were returned")
		})
	}
}

func TestGetHistoricCandlesExtendedFuturesPages(t *testing.T) {
	t.Parallel()
	// The venue's replies, taken live, to futuresWindow's pages at four candles a request: a full page and a shorter last one
	firstPage := futuresRoute{
		method: http.MethodGet,
		path:   "contract/kline/BTC_USDT",
		query:  "end=1792814399&interval=Min60&start=1792800000",
		data: `{"time":[1792800000,1792803600,1792807200,1792810800],"open":[81159.9,80588.3,80898.9,80963.4],` +
			`"close":[80588.3,80898.9,80963.4,80943.9],"high":[81188.2,80935.0,80965.0,81068.6],` +
			`"low":[80574.2,80586.4,80801.5,80811.0],"vol":[1.6499022E7,1.0006081E7,4030439.0,5675471.0],` +
			`"amount":[1.3328937401748E8,8.081974471053E7,3.260778647078E7,4.592740441558E7],` +
			`"realOpen":[81160.0,80588.3,80899.0,80963.5],"realClose":[80588.3,80898.9,80963.4,80943.9],` +
			`"realHigh":[81188.2,80935.0,80965.0,81068.6],"realLow":[80574.2,80586.4,80801.5,80811.0]}`,
	}
	lastPage := futuresRoute{
		method: http.MethodGet,
		path:   "contract/kline/BTC_USDT",
		query:  "end=1792821599&interval=Min60&start=1792814400",
		data: `{"time":[1792814400,1792818000],"open":[80943.9,80857.2],"close":[80857.2,80732.4],"high":[81069.4,80859.8],` +
			`"low":[80812.0,80681.7],"vol":[5345322.0,7363386.0],"amount":[4.32526066287E7,5.945572418861E7],` +
			`"realOpen":[80943.9,80857.1],"realClose":[80857.2,80732.4],"realHigh":[81069.4,80859.8],` +
			`"realLow":[80812.0,80681.7]}`,
	}
	ex := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, firstPage, lastPage), liveUnavailable)
	ex.Features.Enabled.Kline.GlobalResultLimit = 4
	start, end := futuresWindow()
	exp, err := e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandles must not error")
	result, err := ex.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must request each page of the window")
	assert.Equal(t, exp.Candles, result.Candles, "GetHistoricCandlesExtended should return the candles of every page")

	serveFirstPage := futuresHandler(t, futuresContractsRoute, firstPage)
	refused := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/"+lastPage.path && r.URL.RawQuery == lastPage.query {
			_, _ = w.Write([]byte(`{"success":false,"code":510,"message":"Requests are too frequent, please try again later"}`))
			return
		}
		serveFirstPage(w, r)
	}, liveUnavailable)
	refused.Features.Enabled.Kline.GlobalResultLimit = 4
	_, err = refused.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, kline.OneHour, start, end)
	assert.ErrorIs(t, err, errFuturesRequestFailed, "GetHistoricCandlesExtended should return a later page's refusal rather than zero candles")

	emptyFirstPage := firstPage
	emptyFirstPage.data = futuresNoCandlesReply
	gap := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, emptyFirstPage, lastPage), liveUnavailable)
	gap.Features.Enabled.Kline.GlobalResultLimit = 4
	result, err = gap.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must request a later page after a page without candles")
	require.Len(t, result.Candles, len(exp.Candles), "GetHistoricCandlesExtended must return as many candles as the window has hours")
	assert.Equal(t, exp.Candles[4:], result.Candles[4:], "GetHistoricCandlesExtended should return a later page's candles after a page without any")

	// A contract listed part-way through a page: the venue returns only that page's later candles
	shortFirstPage := firstPage
	shortFirstPage.data = `{"time":[1792807200,1792810800],"open":[80898.9,80963.4],"close":[80963.4,80943.9],` +
		`"high":[80965.0,81068.6],"low":[80801.5,80811.0],"vol":[4030439.0,5675471.0],` +
		`"amount":[3.260778647078E7,4.592740441558E7],"realOpen":[80899.0,80963.5],"realClose":[80963.4,80943.9],` +
		`"realHigh":[80965.0,81068.6],"realLow":[80801.5,80811.0]}`
	listed := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, shortFirstPage, lastPage), liveUnavailable)
	listed.Features.Enabled.Kline.GlobalResultLimit = 4
	result, err = listed.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must request a later page after a short one")
	require.Len(t, result.Candles, len(exp.Candles), "GetHistoricCandlesExtended must return as many candles as the window has hours")
	assert.Equal(t, exp.Candles[2:], result.Candles[2:], "GetHistoricCandlesExtended should return a short page's candles and a later page's")
}

func TestGetHistoricCandlesExtendedSpotPages(t *testing.T) {
	t.Parallel()
	// The venue's replies, taken live, to futuresWindow's hourly spot candles in one request, and in pages of four
	// candles: a full page and a shorter last one
	const (
		firstHours = `[1792800000000,"81191.74","81212.9","80620","80630","147.64025561",1792803600000,"11935230.06"],` +
			`[1792803600000,"80630","80953","80630","80933.23","80.673352",1792807200000,"6517384.56"],` +
			`[1792807200000,"80933.23","80983.04","80850","80983.04","49.38711222",1792810800000,"3996840.34"],` +
			`[1792810800000,"80983.04","81085.32","80850","80969.06","111.40396556",1792814400000,"9018551.56"]`
		lastHours = `[1792814400000,"80969.06","81080.49","80843.25","80876.59","111.92291902",1792818000000,"9056775.13"],` +
			`[1792818000000,"80876.59","80889.6","80735.77","80769.99","227.05720215",1792821600000,"18342770.26"]`
	)
	replies := map[string]string{
		"endTime=1792821600000&interval=60m&startTime=1792800000000&symbol=BTCUSDT":         "[" + firstHours + "," + lastHours + "]",
		"endTime=1792814400000&interval=60m&limit=4&startTime=1792800000000&symbol=BTCUSDT": "[" + firstHours + "]",
		"endTime=1792821600000&interval=60m&limit=4&startTime=1792814400000&symbol=BTCUSDT": "[" + lastHours + "]",
	}
	ex := testExchangeFor(t, func(w http.ResponseWriter, r *http.Request) {
		reply, ok := replies[r.URL.RawQuery]
		if r.URL.Path != "/api/v3/klines" || !ok {
			http.Error(w, `{"msg":"unexpected request","code":-1}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}, liveUnavailable)
	start, end := futuresWindow()
	exp, err := ex.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandles must not error")
	ex.Features.Enabled.Kline.GlobalResultLimit = 4
	result, err := ex.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must request each spot page of the window")
	assert.Equal(t, exp.Candles, result.Candles, "GetHistoricCandlesExtended should return the candles of every spot page")
}

// futuresNoCandlesReply is the venue's candle data for a window without candles
const futuresNoCandlesReply = `{"time":[],"open":[],"close":[],"high":[],"low":[],"vol":[],"amount":[],"realOpen":[],"realClose":[],` +
	`"realHigh":[],"realLow":[]}`

// candleSource is an interval the venue doesn't serve and the venue's name for the interval its candles are built from
type candleSource struct {
	a        asset.Item
	pair     currency.Pair
	interval kline.Interval
	source   string
}

func candleSources() []candleSource {
	return []candleSource{
		{a: asset.Spot, pair: spotTradablePair, interval: kline.ThreeMin, source: "1m"},
		{a: asset.Spot, pair: spotTradablePair, interval: kline.TenMin, source: "5m"},
		{a: asset.Spot, pair: spotTradablePair, interval: kline.TwoHour, source: "60m"},
		{a: asset.Spot, pair: spotTradablePair, interval: kline.SixHour, source: "60m"},
		{a: asset.Spot, pair: spotTradablePair, interval: kline.TwelveHour, source: "4h"},
		{a: asset.Spot, pair: spotTradablePair, interval: kline.ThreeDay, source: "1d"},
		{a: asset.Spot, pair: spotTradablePair, interval: kline.TwoWeek, source: "1W"},
		{a: asset.Spot, pair: spotTradablePair, interval: kline.ThreeMonth, source: "1M"},
		{a: asset.Futures, pair: futuresTradablePair, interval: kline.ThreeMin, source: "Min1"},
		{a: asset.Futures, pair: futuresTradablePair, interval: kline.TenMin, source: "Min5"},
		{a: asset.Futures, pair: futuresTradablePair, interval: kline.TwoHour, source: "Min60"},
		{a: asset.Futures, pair: futuresTradablePair, interval: kline.SixHour, source: "Min60"},
		{a: asset.Futures, pair: futuresTradablePair, interval: kline.TwelveHour, source: "Hour4"},
		{a: asset.Futures, pair: futuresTradablePair, interval: kline.ThreeDay, source: "Day1"},
		{a: asset.Futures, pair: futuresTradablePair, interval: kline.TwoWeek, source: "Week1"},
		{a: asset.Futures, pair: futuresTradablePair, interval: kline.ThreeMonth, source: "Month1"},
	}
}

// candleIntervals records the interval each candle request asks for, serving the venue's replies for a window without
// candles so that no interval needs recorded candles
type candleIntervals struct {
	mu        sync.Mutex
	requested []string
}

func (c *candleIntervals) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	serveContracts := futuresHandler(t, futuresContractsRoute)
	return func(w http.ResponseWriter, r *http.Request) {
		var reply string
		switch r.URL.Path {
		case "/api/v3/klines":
			reply = `[]`
		case "/api/v1/contract/kline/BTC_USDT":
			reply = `{"success":true,"code":0,"data":` + futuresNoCandlesReply + `}`
		default:
			serveContracts(w, r)
			return
		}
		c.mu.Lock()
		c.requested = append(c.requested, r.URL.Query().Get("interval"))
		c.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}
}

func (c *candleIntervals) intervals() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requested
}

func TestGetHistoricCandlesSourceIntervals(t *testing.T) {
	t.Parallel()
	for _, tc := range candleSources() {
		t.Run(tc.a.String()+" "+tc.interval.Word(), func(t *testing.T) {
			t.Parallel()
			var venue candleIntervals
			ex := testExchangeFor(t, venue.handler(t), liveUnavailable)
			start, end := futuresWindow()
			_, err := ex.GetHistoricCandles(t.Context(), tc.pair, tc.a, tc.interval, start, end)
			require.ErrorIs(t, err, kline.ErrNoTimeSeriesDataToConvert, "GetHistoricCandles must return that the venue sent no candles")
			assert.Equal(t, []string{tc.source}, venue.intervals(), "GetHistoricCandles should request the interval the candles are built from")
		})
	}
}

func TestGetHistoricCandlesExtendedSourceIntervals(t *testing.T) {
	t.Parallel()
	for _, tc := range candleSources() {
		t.Run(tc.a.String()+" "+tc.interval.Word(), func(t *testing.T) {
			t.Parallel()
			var venue candleIntervals
			ex := testExchangeFor(t, venue.handler(t), liveUnavailable)
			start, end := futuresWindow()
			_, err := ex.GetHistoricCandlesExtended(t.Context(), tc.pair, tc.a, tc.interval, start, end)
			require.ErrorIs(t, err, kline.ErrNoTimeSeriesDataToConvert, "GetHistoricCandlesExtended must return that the venue sent no candles")
			assert.Equal(t, []string{tc.source}, venue.intervals(), "GetHistoricCandlesExtended should request the interval the candles are built from")
		})
	}
}

func TestFuturesCapabilities(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Equal(t, exchange.FuturesCapabilities{
		FundingRates:        true,
		FundingRateBatching: map[asset.Item]bool{asset.Futures: true},
		SupportedFundingRateFrequencies: map[kline.Interval]bool{
			kline.OneHour:   true,
			kline.FourHour:  true,
			kline.EightHour: true,
			kline.OneDay:    true,
		},
		Positions: true,
		Leverage:  true,
		OpenInterest: exchange.OpenInterestSupport{
			Supported:          true,
			SupportedViaTicker: true,
			SupportsRestBatch:  true,
		},
	}, ex.Features.Supports.FuturesCapabilities, "the futures capabilities should match the implemented wrapper methods")
	_, err := ex.CurrencyPairs.Get(asset.Futures)
	assert.NoError(t, err, "the futures pair store should be set up")
}
