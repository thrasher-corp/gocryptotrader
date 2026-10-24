package mexc

import (
	"bytes"
	"io"
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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	`"price":109777.5,"vol":5,"leverage":2,"side":1,"category":1,"orderType":5,"dealAvgPrice":109777.5,"dealVol":5,` +
	`"orderMargin":27.444375,"takerFee":0.0329,"makerFee":0,"profit":0,"feeCurrency":"USDT","openType":1,"state":3,` +
	`"externalOid":"gct-1","errorCode":0,"usedMargin":27.444375,"createTime":1761887134000,"updateTime":1761887135000,` +
	`"positionMode":1,"version":2}`

// futuresOpenPositionReply is an isolated long position
const futuresOpenPositionReply = `{"positionId":1109973831,"symbol":"BTC_USDT","positionType":1,"openType":1,"state":1,` +
	`"holdVol":5,"holdAvgPrice":109777.5,"openAvgPrice":109777.5,"liquidatePrice":55020.5,"oim":27.5,"im":27.5,` +
	`"realised":-0.0329,"leverage":2,"createTime":1761887133854,"updateTime":1761887133854,"version":1,"unRealizedPnl":-0.0039}`

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err, "GetFuturesContractDetails must not error")
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
	if mockTests {
		hooked, ok := contracts["HOOKED_USDT"]
		require.True(t, ok, "a contract closed to API trading must still be listed")
		assert.False(t, hooked.IsActive, "a contract closed to API trading should not be active")
	}
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
	assert.False(t, result[0].LatestRate.Time.IsZero(), "the latest settled rate should carry its time")
	assert.True(t, result[0].TimeOfNextRate.After(result[0].LatestRate.Time), "the next funding should follow the latest")
	assert.Equal(t, result[0].TimeOfNextRate, result[0].PredictedUpcomingRate.Time, "the predicted rate should be settled at the next funding")

	result, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures})
	require.NoError(t, err, "GetLatestFundingRates must not error for the enabled pairs")
	assert.Len(t, result, 2, "every enabled contract's rate should be returned")
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
	require.NotEmpty(t, result.FundingRates, "settled rates must be returned")
	for i := 1; i < len(result.FundingRates); i++ {
		assert.Truef(t, result.FundingRates[i].Time.After(result.FundingRates[i-1].Time), "rate %d should follow the one before it", i)
	}
	assert.Equal(t, result.FundingRates[len(result.FundingRates)-1], result.LatestRate, "the latest rate should be the newest settled")
	assert.True(t, result.TimeOfNextRate.After(result.LatestRate.Time), "the next funding should follow the latest")

	start = start.Add(-7 * 24 * time.Hour)
	result, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: futuresTradablePair, StartDate: start, EndDate: end})
	require.NoError(t, err, "GetHistoricalFundingRates must not error within a window")
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
	assert.True(t, result[0].Key.Pair().Equal(futuresTradablePair), "the contract asked for should be returned")
	assert.Positive(t, result[0].OpenInterest, "the open interest should be positive")

	for _, keys := range [][]key.PairAsset{{linear, inverse}, nil} {
		result, err = e.GetOpenInterest(t.Context(), keys...)
		require.NoErrorf(t, err, "GetOpenInterest must not error for %d contracts", len(keys))
		assert.Lenf(t, result, 2, "both contracts' open interest should be returned for %d contracts asked for", len(keys))
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
	cost, code := futuresCost(&FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USDT, SettleCoin: currency.USDT, ContractSize: 0.0001}, 5, 100000)
	assert.InDelta(t, 50.0, cost, 1e-9, "a linear contract should cost its contracts times its size and price")
	assert.Equal(t, currency.USDT, code, "a linear contract should cost its settlement currency")

	cost, code = futuresCost(&FuturesContractResponse{BaseCoin: currency.BTC, QuoteCoin: currency.USD, SettleCoin: currency.BTC, ContractSize: 100}, 5, 100000)
	assert.InDelta(t, 0.005, cost, 1e-12, "an inverse contract should cost its contracts times its size over the price")
	assert.Equal(t, currency.BTC, code, "an inverse contract should cost its settlement currency")
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
	windows := futuresHistoryWindows(start, end)
	require.Len(t, windows, 3, "a 200 day window must be read in three parts")
	assert.Equal(t, start, windows[0].start, "the first part should start the window")
	assert.Equal(t, end, windows[2].end, "the last part should end the window")
	for i := range windows {
		assert.LessOrEqualf(t, windows[i].end.Sub(windows[i].start), futuresHistoryWindow, "part %d should span at most 90 days", i)
		if i > 0 {
			assert.Equalf(t, windows[i-1].end, windows[i].start, "part %d should follow on from the one before", i)
		}
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
	b := futuresBalance(10, 2, 1.5)
	assert.Equal(t, 10.0, b.Free, "the available balance should be free")
	assert.Equal(t, 3.5, b.Hold, "frozen funds and position margin should be held")
	assert.Equal(t, 13.5, b.Total, "the total should be what is free and held")
}

func TestSideFromMarginSide(t *testing.T) {
	t.Parallel()
	assert.Equal(t, order.Long, sideFromMarginSide("long"), "a long margin side should be long")
	assert.Equal(t, order.Short, sideFromMarginSide("SHORT"), "a short margin side should be short")
	assert.Equal(t, order.UnknownSide, sideFromMarginSide(""), "an unset margin side should name no side")
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
	require.Len(t, subAccounts[0].Balances, 1, "a balance without a currency must be skipped")
	b := subAccounts[0].Balances[currency.USDT]
	assert.Equal(t, 10.0, b.Free, "the available balance should be free")
	assert.Equal(t, 3.5, b.Hold, "frozen funds and position margin should be held")
	assert.Equal(t, 13.5, b.Total, "the total should be what is free and held")
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
			body:   `{"symbol":"BTC_USDT","price":50000,"vol":1,"leverage":20,"side":1,"type":1,"openType":1,"externalOid":"gct-1","stpMode":0}`,
			data:   `{"orderId":"739113577038255616","ts":1761888808839}`,
		}), liveUnavailable)
		resp, err := submit(ex, order.Submit{Side: order.Buy, Type: order.Limit, Price: 50000, Amount: 1, ClientOrderID: "gct-1"})
		require.NoError(t, err, "SubmitOrder must not error for futures")
		assert.Equal(t, order.New, resp.Status, "a placed order should be new")
		assert.Equal(t, "739113577038255616", resp.OrderID, "the order ID should be returned")
		assert.Equal(t, margin.Isolated, resp.MarginType, "an order without a margin type should be isolated")
		assert.Equal(t, 20.0, resp.Leverage, "the account's leverage should be used")
		assert.Equal(t, int64(1761888808839), resp.Date.UnixMilli(), "the venue's placement time should be used")
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

	// A close names the position it closes, and takes the position's margin mode when it names none
	isolatedLong := futuresOpenPositionReply
	crossLong := strings.NewReplacer(`"positionId":1109973831`, `"positionId":1109973832`, `"openType":1`, `"openType":2`).Replace(futuresOpenPositionReply)
	for name, tc := range map[string]struct {
		positions  string
		marginType margin.Type
		body       string
		exp        margin.Type
		err        error
	}{
		"cross close": {
			positions: "[" + isolatedLong + "," + crossLong + "]", marginType: margin.Multi,
			body: `{"symbol":"BTC_USDT","vol":2,"side":4,"type":5,"openType":2,"positionId":1109973832,"stpMode":0}`, exp: margin.Multi,
		},
		"close of the position's margin mode": {
			positions: "[" + crossLong + "]",
			body:      `{"symbol":"BTC_USDT","vol":2,"side":4,"type":5,"openType":2,"positionId":1109973832,"stpMode":0}`, exp: margin.Multi,
		},
		"close without a position": {
			positions: "[]",
			body:      `{"symbol":"BTC_USDT","vol":2,"side":4,"type":5,"openType":1,"stpMode":0}`, exp: margin.Isolated,
		},
		"close of either margin mode": {positions: "[" + isolatedLong + "," + crossLong + "]", err: errMarginTypeAmbiguous},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			routes := []futuresRoute{{method: http.MethodGet, path: "private/position/open_positions", query: "symbol=BTC_USDT", data: tc.positions}}
			if tc.body != "" {
				routes = append(routes, futuresRoute{method: http.MethodPost, path: "private/order/create", body: tc.body, data: `{"orderId":"739113577038255617","ts":1761888808840}`})
			}
			ex := testExchangeFor(t, futuresHandler(t, routes...), liveUnavailable)
			resp, err := submit(ex, order.Submit{Side: order.Sell, Type: order.Market, Amount: 2, ReduceOnly: true, MarginType: tc.marginType})
			require.ErrorIs(t, err, tc.err, "SubmitOrder must return the expected error for a closing order")
			if tc.err != nil {
				return
			}
			assert.Equal(t, tc.exp, resp.MarginType, "the closed position's margin type should be returned")
			assert.True(t, resp.ReduceOnly, "a closing order should reduce only")
			assert.Zero(t, resp.Leverage, "a closing order should send no leverage")
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
	require.Len(t, resp.Status, len(orders), "every order must carry an outcome")
	assert.Equal(t, order.Cancelled.String(), resp.Status["1000"], "a cancelled order should be reported cancelled")
	assert.Equal(t, "order state cannot be cancelled", resp.Status["1050"], "a refused order should carry the venue's reason")
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
			assert.Empty(t, resp.Status, "the venue names no orders, so none should be reported")
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
	assert.Equal(t, "739106551624717312", detail.OrderID, "the order ID should be returned")
	assert.Equal(t, "gct-1", detail.ClientOrderID, "the external order ID should be the client order ID")
	assert.True(t, detail.Pair.Equal(futuresTradablePair), "the contract should be returned")
	assert.Equal(t, order.Market, detail.Type, "the order type should be decoded")
	assert.Equal(t, order.Buy, detail.Side, "an opening long should be a buy")
	assert.Equal(t, order.Filled, detail.Status, "a filled order should be filled")
	assert.Equal(t, 5.0, detail.ContractAmount, "the order's contracts should be returned")
	assert.Equal(t, 5.0, detail.ExecutedAmount, "the filled contracts should be returned")
	assert.Zero(t, detail.RemainingAmount, "a filled order should have nothing remaining")
	assert.InDelta(t, 54.88875, detail.Cost, 1e-9, "the cost should be the contracts' value in the settlement currency")
	assert.Equal(t, currency.USDT, detail.CostAsset, "the cost should be in the settlement currency")
	assert.Equal(t, 0.0329, detail.Fee, "the fee should be returned")
	assert.Equal(t, margin.Isolated, detail.MarginType, "the margin type should be decoded")
	assert.Equal(t, 2.0, detail.Leverage, "the leverage should be decoded")
	assert.Equal(t, int64(1761887135000), detail.LastUpdated.UnixMilli(), "the update time should be decoded")
	require.Len(t, detail.Trades, 1, "the order's fill must be returned")
	assert.Equal(t, "9466823344", detail.Trades[0].TID, "the fill ID should be decoded")
	assert.False(t, detail.Trades[0].IsMaker, "a taker fill should not be a maker's")
	assert.Equal(t, "USDT", detail.Trades[0].FeeAsset, "the fill's fee currency should be decoded")
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
	require.Len(t, orders, 1, "only the pair asked for must be returned")
	assert.Equal(t, "750330663772702720", orders[0].OrderID, "the pair's order should be returned")
	assert.Equal(t, order.New, orders[0].Status, "an unfilled order should be new")
	assert.Equal(t, order.Limit, orders[0].Type, "the order type should be decoded")
	assert.Equal(t, order.GoodTillCancel, orders[0].TimeInForce, "a limit order should rest until cancelled")
	assert.Equal(t, margin.Multi, orders[0].MarginType, "a cross order should be multi")
	assert.Equal(t, 10.0, orders[0].RemainingAmount, "an unfilled order should have every contract remaining")
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
	require.Len(t, orders, 2, "each order must be returned once")
	byID := map[string]order.Detail{orders[0].OrderID: orders[0], orders[1].OrderID: orders[1]}
	assert.Equal(t, order.Filled, byID["739211854408619008"].Status, "a filled order should be filled")
	assert.Equal(t, order.Sell, byID["739211854408619008"].Side, "an opening short should be a sell")
	assert.InDelta(t, 10.0, byID["739211854408619008"].Cost, 1e-9, "a filled order should carry its cost")
	assert.Equal(t, order.Cancelled, byID["739211257974395392"].Status, "an unfilled cancelled order should be cancelled")
	assert.True(t, byID["739211257974395392"].ReduceOnly, "a closing order should reduce only")
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

	ex := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
		method: http.MethodGet,
		path:   "private/position/open_positions",
		query:  "symbol=BTC_USDT",
		data:   "[" + futuresOpenPositionReply + "]",
	}), liveUnavailable)
	summary, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: futuresTradablePair})
	require.NoError(t, err, "GetFuturesPositionSummary must not error")
	assert.Equal(t, 5.0, summary.CurrentSize.InexactFloat64(), "a long position's size should be positive")
	assert.InDelta(t, 54.88875, summary.NotionalSize.InexactFloat64(), 1e-9, "the notional size should be the position's value")
	assert.Equal(t, 2.0, summary.Leverage.InexactFloat64(), "the leverage should be returned")
	assert.Equal(t, margin.Isolated, summary.MarginType, "the margin type should be returned")
	assert.Equal(t, 27.5, summary.IsolatedMargin.InexactFloat64(), "an isolated position should report its margin")
	assert.Equal(t, currency.USDT, summary.Currency, "the settlement currency should be returned")
	assert.Equal(t, futures.Linear, summary.ContractSettlementType, "the contract should be linear")
	assert.Equal(t, -0.0039, summary.UnrealisedPNL.InexactFloat64(), "the unrealised PNL should be returned")
	assert.Equal(t, 55020.5, summary.EstimatedLiquidationPrice.InexactFloat64(), "the liquidation price should be returned")

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
	assert.Equal(t, -5.0, summary.CurrentSize.InexactFloat64(), "a short position's size should be negative")
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
	require.Len(t, resp, 1, "the contract's orders must be returned")
	assert.Equal(t, futures.Linear, resp[0].ContractSettlementType, "the contract should be linear")
	require.Len(t, resp[0].Orders, 1, "the contract's order must be returned")
	assert.Equal(t, "739106551624717312", resp[0].Orders[0].OrderID, "the order should be returned")
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

	positions := futuresRoute{method: http.MethodGet, path: "private/position/open_positions", query: "symbol=BTC_USDT", data: "[" + futuresOpenPositionReply + "]"}
	for name, tc := range map[string]struct {
		newMargin float64
		body      string
	}{
		"add":       {newMargin: 30, body: `{"positionId":1109973831,"amount":2.5,"type":"ADD"}`},
		"subtract":  {newMargin: 25, body: `{"positionId":1109973831,"amount":2.5,"type":"SUB"}`},
		"unchanged": {newMargin: 27.5},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			routes := []futuresRoute{positions}
			if tc.body != "" {
				routes = append(routes, futuresRoute{method: http.MethodPost, path: "private/position/change_margin", body: tc.body})
			}
			ex := testExchangeFor(t, futuresHandler(t, routes...), liveUnavailable)
			resp, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
				Exchange:           ex.Name,
				Asset:              asset.Futures,
				Pair:               futuresTradablePair,
				MarginType:         margin.Isolated,
				NewAllocatedMargin: tc.newMargin,
			})
			require.NoError(t, err, "ChangePositionMargin must not error")
			assert.Equal(t, tc.newMargin, resp.AllocatedMargin, "the position's new margin should be returned")
		})
	}
}

func TestGetHistoricCandlesFuturesMalformed(t *testing.T) {
	t.Parallel()
	ex := testExchangeFor(t, futuresHandler(t, futuresContractsRoute, futuresRoute{
		method: http.MethodGet,
		path:   "contract/kline/BTC_USDT",
		data:   `{"time":[1792800000,1792803600],"open":[1],"close":[1,2],"high":[1,2],"low":[1,2],"vol":[1,2],"amount":[1,2]}`,
	}), liveUnavailable)
	start, end := futuresWindow()
	_, err := ex.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.OneHour, start, end)
	assert.ErrorIs(t, err, errMalformedCandles, "candle fields of unequal lengths should be refused")
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
