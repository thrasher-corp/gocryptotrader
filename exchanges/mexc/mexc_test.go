package mexc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                    = ""
	apiSecret                 = ""
	canManipulateRealOrders   = false
	canManipulateAPIEndpoints = false
)

var (
	e *Exchange

	assetsAndErrors = map[asset.Item]error{
		asset.Spot:    nil,
		asset.Futures: nil,
		asset.Options: asset.ErrNotSupported,
	}

	spotTradablePair, futuresTradablePair currency.Pair
)

func (e *Exchange) setEnabledPairs(spotTradablePair, futuresTradablePair currency.Pair) error {
	if err := e.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair}, false); err != nil {
		return err
	}
	if err := e.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair}, true); err != nil {
		return err
	}
	if err := e.CurrencyPairs.StorePairs(asset.Futures, []currency.Pair{futuresTradablePair}, false); err != nil {
		return err
	}
	return e.CurrencyPairs.StorePairs(asset.Futures, []currency.Pair{futuresTradablePair}, true)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbols(t.Context(), nil)
	assert.NoError(t, err)

	result, err := e.GetSymbols(t.Context(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemTime(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDefaultSumbols(t *testing.T) {
	t.Parallel()
	result, err := e.GetDefaultSumbols(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetOrderbook(t.Context(), spotTradablePair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTradesList(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTradesList(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetRecentTradesList(t.Context(), spotTradablePair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetAggregatedTrades(t.Context(), currency.EMPTYPAIR, endTime, startTime, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetAggregatedTrades(t.Context(), spotTradablePair, endTime, startTime, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetAggregatedTrades(t.Context(), spotTradablePair, startTime, endTime, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIntervalToString(t *testing.T) {
	t.Parallel()
	_, err := intervalToString(kline.TenMin)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	intervalString, err := intervalToString(kline.FiveMin)
	require.NoError(t, err)
	assert.NotEmpty(t, intervalString)
}

func TestGetCandlestick(t *testing.T) {
	t.Parallel()
	intervalString, err := intervalToString(kline.FiveMin)
	require.NoError(t, err)

	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err = e.GetCandlestick(t.Context(), currency.EMPTYPAIR, intervalString, startTime, endTime, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetCandlestick(t.Context(), spotTradablePair, "", startTime, endTime, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetCandlestick(t.Context(), spotTradablePair, "5m", startTime, endTime, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentAveragePrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetCurrentAveragePrice(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGet24HourTickerPriceChangeStatistics(t *testing.T) {
	t.Parallel()
	result, err := e.Get24HourTickerPriceChangeStatistics(t.Context(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.Get24HourTickerPriceChangeStatistics(t.Context(), []string{spotTradablePair.String()})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbolPriceTicker(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetSymbolPriceTicker(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolsPriceTicker(t *testing.T) {
	result, err := e.GetSymbolsPriceTicker(t.Context(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetSymbolsPriceTicker(t.Context(), []string{spotTradablePair.String()})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbolOrderbookTicker(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetSymbolOrderbookTicker(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbookTickers(t *testing.T) {
	t.Parallel()
	result, err := e.GetOrderbookTickers(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubAccount(t.Context(), "", "sub-account notes")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	_, err = e.CreateSubAccount(t.Context(), "Test1", "")
	require.ErrorIs(t, err, errInvalidSubAccountNote)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateSubAccount(t.Context(), "Test1", "sub-account notes")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountList(t.Context(), "sam", true, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateAPIKeyForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAPIKeyForSubAccount(t.Context(), "", "123", "SPOT_DEAL_WRITE", "")
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = e.CreateAPIKeyForSubAccount(t.Context(), "SubAcc2", "", "SPOT_DEAL_WRITE", "")
	require.ErrorIs(t, err, errInvalidSubAccountNote)
	_, err = e.CreateAPIKeyForSubAccount(t.Context(), "SubAcc2", "123", "", "")
	require.ErrorIs(t, err, errUnsupportedPermissionValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateAPIKeyForSubAccount(t.Context(), "SubAcc2", "123", "SPOT_DEAL_WRITE", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAPIKey(t.Context(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountAPIKey(t.Context(), "SubAcc1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteAPIKeySubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteAPIKeySubAccount(t.Context(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.DeleteAPIKeySubAccount(t.Context(), "SubAcc1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Empty, asset.Futures, currency.USDT, 1234.)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Empty, currency.USDT, 1234.)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Futures, currency.EMPTYCODE, 1234.)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Futures, currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubAccountUniversalTransfer(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Futures, currency.USDT, 1234.)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountUnversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountUnversalTransferHistory(t.Context(), "master@test.com", "subaccount@test.com", asset.Empty, asset.Futures, time.Now().Add(-time.Hour*50), time.Now().Add(-time.Hour*20), 10, 20)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.GetSubAccountUnversalTransferHistory(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Empty, time.Now().Add(-time.Hour*50), time.Now().Add(-time.Hour*20), 10, 20)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountUnversalTransferHistory(t.Context(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Futures, time.Now().Add(-time.Hour*50), time.Now().Add(-time.Hour*20), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAsset(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountAsset(t.Context(), "", asset.Spot)
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = e.GetSubAccountAsset(t.Context(), "thesubaccount@test.com", asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountAsset(t.Context(), "thesubaccount@test.com", asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKYCStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetKYCStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUseAPIDefaultSymbols(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UseAPIDefaultSymbols(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewTestOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewTestOrder(t.Context(), currency.EMPTYPAIR, "123123", "SELL", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.NewTestOrder(t.Context(), spotTradablePair, "123123", "", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.NewTestOrder(t.Context(), spotTradablePair, "123123", "SELL", "", 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.NewTestOrder(t.Context(), spotTradablePair, "123123", "SELL", typeLimit, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.NewTestOrder(t.Context(), spotTradablePair, "123123", "SELL", typeLimit, 1, 0, 0)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.NewTestOrder(t.Context(), spotTradablePair, "123123", "SELL", typeMarket, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.NewTestOrder(t.Context(), spotTradablePair, "123123", "SELL", typeLimit, 1, 0, 123456.78)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSpotOrderStringFromOrderTypeAndTimeInForce(t *testing.T) {
	t.Parallel()
	orderTypeAndTimeInForceToOrderTypeString := []struct {
		OType       order.Type
		TimeInForce order.TimeInForce
		String      string
		Error       error
	}{
		{order.Limit, order.PostOnly, typeLimitMaker, nil},
		{order.Limit, order.UnknownTIF, typeLimit, nil},
		{order.Market, order.UnknownTIF, typeMarket, nil},
		{order.UnknownType, order.PostOnly, typeLimitMaker, nil},
		{order.Market, order.FillOrKill, typeFillOrKill, nil},
		{order.Market, order.ImmediateOrCancel, typeImmediateOrCancel, nil},
		{order.UnknownType, order.FillOrKill, typeFillOrKill, nil},
		{order.UnknownType, order.ImmediateOrCancel, typeImmediateOrCancel, nil},
		{order.UnknownType, order.UnknownTIF, "", order.ErrTypeIsInvalid},
	}
	for x := range orderTypeAndTimeInForceToOrderTypeString {
		t.Run(orderTypeAndTimeInForceToOrderTypeString[x].String, func(t *testing.T) {
			t.Parallel()
			result, err := SpotOrderStringFromOrderTypeAndTimeInForce(orderTypeAndTimeInForceToOrderTypeString[x].OType, orderTypeAndTimeInForceToOrderTypeString[x].TimeInForce)
			require.ErrorIs(t, err, orderTypeAndTimeInForceToOrderTypeString[x].Error)
			assert.Equal(t, orderTypeAndTimeInForceToOrderTypeString[x].String, result)
		})
	}
}

func TestStringToOrderTypeAndTimeInForce(t *testing.T) {
	t.Parallel()
	orderTypeAndTimeInForceFromOrderTypeString := []struct {
		String      string
		Type        order.Type
		TimeInForce order.TimeInForce
		Error       error
	}{
		{typeLimit, order.Limit, order.GoodTillCancel, nil},
		{typeLimitMaker, order.Limit, order.PostOnly, nil},
		{typePostOnly, order.Limit, order.PostOnly, nil},
		{typeMarket, order.Market, order.UnknownTIF, nil},
		{typeImmediateOrCancel, order.Market, order.ImmediateOrCancel, nil},
		{typeFillOrKill, order.Market, order.FillOrKill, nil},
		{typeStopLimit, order.StopLimit, order.UnknownTIF, nil},
		{"", order.UnknownType, order.UnknownTIF, order.ErrUnsupportedOrderType},
	}
	for x := range orderTypeAndTimeInForceFromOrderTypeString {
		t.Run(orderTypeAndTimeInForceFromOrderTypeString[x].String, func(t *testing.T) {
			t.Parallel()
			oType, tif, err := e.StringToOrderTypeAndTimeInForce(orderTypeAndTimeInForceFromOrderTypeString[x].String)
			require.ErrorIs(t, err, orderTypeAndTimeInForceFromOrderTypeString[x].Error)
			assert.Equal(t, orderTypeAndTimeInForceFromOrderTypeString[x].Type, oType)
			assert.Equal(t, orderTypeAndTimeInForceFromOrderTypeString[x].TimeInForce, tif)
		})
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewOrder(t.Context(), currency.EMPTYPAIR, "123123", "SELL", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", "", 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", typeLimit, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", typeLimit, 1, 0, 0)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.NewOrder(t.Context(), spotTradablePair, "123123", "SELL", typeMarket, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewOrder(t.Context(), spotTradablePair, "123123", "BUY", typeLimit, 1, 0, 123456.78)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBatchOrder(t *testing.T) {
	t.Parallel()
	arg := BatchOrderCreationParam{
		NewClientOrderID: 1234,
	}
	_, err := e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = spotTradablePair
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.OrderType, err = e.OrderTypeStringFromOrderTypeAndTimeInForce(order.Limit, order.UnknownTIF)
	require.NoError(t, err)
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Quantity = 123478.5
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.OrderType, err = e.OrderTypeStringFromOrderTypeAndTimeInForce(order.Market, order.UnknownTIF)
	require.NoError(t, err)

	arg.Quantity = 0
	_, err = e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.Quantity = 1231231
	result, err := e.CreateBatchOrder(t.Context(), []BatchOrderCreationParam{arg})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	typesMap := map[struct {
		Type        order.Type
		TimeInForce order.TimeInForce
	}]struct {
		String string
		Error  error
	}{
		{Type: order.Limit}:                    {String: typeLimit},
		{TimeInForce: order.PostOnly}:          {String: typePostOnly},
		{Type: order.Market}:                   {String: typeMarket},
		{TimeInForce: order.ImmediateOrCancel}: {String: typeImmediateOrCancel},
		{TimeInForce: order.FillOrKill}:        {String: typeFillOrKill},
		{Type: order.StopLimit}:                {String: typeStopLimit},
		{Type: order.MarketMakerProtection}:    {String: "", Error: order.ErrUnsupportedOrderType},
	}
	for a := range typesMap {
		t.Run(typesMap[a].String, func(t *testing.T) {
			t.Parallel()
			value, err := e.OrderTypeStringFromOrderTypeAndTimeInForce(a.Type, a.TimeInForce)
			require.Equal(t, typesMap[a].String, value)
			require.ErrorIs(t, err, typesMap[a].Error)
		})
	}
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelTradeOrder(t.Context(), currency.EMPTYPAIR, "", "", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.CancelTradeOrder(t.Context(), spotTradablePair, "", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelTradeOrder(t.Context(), spotTradablePair, "1234", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOpenOrdersBySymbol(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOpenOrdersBySymbol(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderByID(t.Context(), currency.EMPTYPAIR, "123455", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetOrderByID(t.Context(), spotTradablePair, "", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderByID(t.Context(), spotTradablePair, "1234", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrders(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllOrders(t.Context(), currency.EMPTYPAIR, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err = e.GetAllOrders(t.Context(), spotTradablePair, endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllOrders(t.Context(), spotTradablePair, startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountTradeList(t.Context(), currency.EMPTYPAIR, "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err = e.GetAccountTradeList(t.Context(), spotTradablePair, "", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountTradeList(t.Context(), spotTradablePair, "", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableMXDeduct(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableMXDeduct(t.Context(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMXDeductStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMXDeductStatus(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolTradingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSymbolTradingFee(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrencyInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCapital(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCapital(t.Context(), 1.2, currency.EMPTYCODE, "", "BNB", "1234", core.BitcoinDonationAddress, "abcd", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WithdrawCapital(t.Context(), 1.2, currency.BTC, "", "BNB", "", "", "abcd", "")
	require.ErrorIs(t, err, errAddressRequired)
	_, err = e.WithdrawCapital(t.Context(), 0, currency.BTC, "", "BNB", "1234", core.BitcoinDonationAddress, "abcd", "")
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawCapital(t.Context(), 1.2, currency.BTC, "", "BNB", "1234", core.BitcoinDonationAddress, "abcd", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawal(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelWithdrawal(t.Context(), "1231212")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundDepositHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetFundDepositHistory(t.Context(), currency.BTC, "", endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundDepositHistory(t.Context(), currency.BTC, "", startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetWithdrawalHistory(t.Context(), currency.USDT, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalHistory(t.Context(), currency.USDT, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateDepositAddress(t.Context(), currency.EMPTYCODE, "TRC20")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GenerateDepositAddress(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, errNetworkNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GenerateDepositAddress(t.Context(), currency.USDT, "TRC20")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressOfCoin(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddressOfCoin(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositAddressOfCoin(t.Context(), currency.USDT, "ERC20")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalAddress(t.Context(), currency.USDT, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUserUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.UserUniversalTransfer(t.Context(), asset.Empty, asset.Spot, currency.USDT, 1000)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)
	_, err = e.UserUniversalTransfer(t.Context(), asset.Futures, asset.Empty, currency.USDT, 1000)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)
	_, err = e.UserUniversalTransfer(t.Context(), asset.Futures, asset.Spot, currency.EMPTYCODE, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.UserUniversalTransfer(t.Context(), asset.Futures, asset.Spot, currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UserUniversalTransfer(t.Context(), asset.Futures, asset.Spot, currency.USDT, 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnversalTransferHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetUniversalTransferHistory(t.Context(), asset.Empty, asset.Futures, startTime, endTime, 0, 10)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)
	_, err = e.GetUniversalTransferHistory(t.Context(), asset.Spot, asset.Empty, startTime, endTime, 0, 10)
	require.ErrorIs(t, err, asset.ErrInvalidAsset)
	_, err = e.GetUniversalTransferHistory(t.Context(), asset.Spot, asset.Futures, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUniversalTransferHistory(t.Context(), asset.Spot, asset.Futures, startTime, endTime, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUniversalTransferDetailByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetUniversalTransferDetailByID(t.Context(), "")
	require.ErrorIs(t, err, errTransactionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUniversalTransferDetailByID(t.Context(), "12345678")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetThatCanBeConvertedintoMX(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAssetThatCanBeConvertedintoMX(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDustTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.DustTransfer(t.Context(), []currency.Code{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.DustTransfer(t.Context(), []currency.Code{currency.EMPTYCODE, currency.ETH})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.DustTransfer(t.Context(), []currency.Code{currency.BTC, currency.ETH})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDustLog(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.DustLog(t.Context(), endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)
	_, err = e.DustLog(t.Context(), startTime, endTime, 0, 0)
	require.ErrorIs(t, err, errPaginationLimitIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.DustLog(t.Context(), startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInternalTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.InternalTransfer(t.Context(), "", "someone@example.com", "+251", currency.USDT, 1.2)
	require.ErrorIs(t, err, errAccountTypeRequired)
	_, err = e.InternalTransfer(t.Context(), "EMAIL", "", "+251", currency.USDT, 1.2)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = e.InternalTransfer(t.Context(), "EMAIL", "someone@example.com", "+251", currency.EMPTYCODE, 1.2)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.InternalTransfer(t.Context(), "EMAIL", "someone@example.com", "+251", currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.InternalTransfer(t.Context(), "EMAIL", "someone@example.com", "+251", currency.USDT, 1.2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInternalTransferHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetInternalTransferHistory(t.Context(), "11945860693", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetInternalTransferHistory(t.Context(), "11945860693", startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCapitalWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CapitalWithdrawal(t.Context(), currency.EMPTYCODE, "1234", "TRC20", core.BitcoinDonationAddress, "", "", 1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.CapitalWithdrawal(t.Context(), currency.BTC, "12345678", "TRC20", "", "", "", 1234)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = e.CapitalWithdrawal(t.Context(), currency.BTC, "1234", "TRC20", core.BitcoinDonationAddress, "", "", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CapitalWithdrawal(t.Context(), currency.BTC, "1234", "TRC20", core.BitcoinDonationAddress, "", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRebateHistoryRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetRebateHistoryRecords(t.Context(), endTime, startTime, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRebateHistoryRecords(t.Context(), startTime, endTime, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRebateRecordsDetail(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetRebateRecordsDetail(t.Context(), endTime, startTime, 1000)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRebateRecordsDetail(t.Context(), startTime, endTime, 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSelfRebateRecordsDetail(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetSelfRebateRecordsDetail(t.Context(), endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSelfRebateRecordsDetail(t.Context(), startTime, endTime, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetReferCode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetReferCode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCommissionRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetAffiliateCommissionRecord(t.Context(), endTime, startTime, "abcdef", 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateCommissionRecord(t.Context(), startTime, endTime, "abcdef", 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateWithdrawRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetAffiliateWithdrawRecord(t.Context(), endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateWithdrawRecord(t.Context(), startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCommissionDetailRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetAffiliateCommissionDetailRecord(t.Context(), endTime, startTime, "", "1", 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateCommissionDetailRecord(t.Context(), startTime, endTime, "", "1", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCampaignData(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetAffiliateCampaignData(t.Context(), endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateCampaignData(t.Context(), startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateReferralData(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetAffiliateReferralData(t.Context(), endTime, startTime, "", "", 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateReferralData(t.Context(), startTime, endTime, "350882", "abc", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAffiliateData(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetSubAffiliateData(t.Context(), endTime, startTime, "", 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAffiliateData(t.Context(), startTime, endTime, "", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractsDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContracts(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)

	result, err := e.GetFuturesContracts(t.Context(), futuresTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTransferableCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetTransferableCurrencies(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractDepthInformation(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractOrderbook(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractOrderbook(t.Context(), spotTradablePair, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepthSnapshotOfContract(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepthSnapshotOfContract(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetDepthSnapshotOfContract(t.Context(), futuresTradablePair, 0)
	require.ErrorIs(t, err, errPaginationLimitIsRequired)

	result, err := e.GetDepthSnapshotOfContract(t.Context(), futuresTradablePair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractIndexPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractIndexPrice(t.Context(), futuresTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractFairPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractFairPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractFairPrice(t.Context(), futuresTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractFundingPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractFundingPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractFundingPrice(t.Context(), futuresTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestContractIntervalString(t *testing.T) {
	t.Parallel()
	intervalToStringMap := map[kline.Interval]struct {
		String string
		Error  error
	}{
		kline.OneMin:     {"Min1", nil},
		kline.FiveMin:    {"Min5", nil},
		kline.FifteenMin: {"Min15", nil},
		kline.ThirtyMin:  {"Min30", nil},
		kline.OneHour:    {"Min60", nil},
		kline.FourHour:   {"Hour4", nil},
		kline.EightHour:  {"Hour8", nil},
		kline.OneDay:     {"Day1", nil},
		kline.OneWeek:    {"Week1", nil},
		kline.OneMonth:   {"Month1", nil},
		kline.SixMonth:   {"", kline.ErrUnsupportedInterval},
	}
	for key, result := range intervalToStringMap {
		value, err := ContractIntervalString(key)
		require.ErrorIs(t, err, result.Error)
		assert.Equal(t, result.String, value)
	}
}

func TestGetContractsCandlestickData(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetContractsCandlestickData(t.Context(), currency.EMPTYPAIR, 0, startTime, endTime)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetContractsCandlestickData(t.Context(), currency.EMPTYPAIR, 0, endTime, startTime)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractsCandlestickData(t.Context(), futuresTradablePair, kline.FifteenMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKlineDataOfIndexPrice(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetKlineDataOfIndexPrice(t.Context(), currency.EMPTYPAIR, 0, startTime, endTime)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetKlineDataOfIndexPrice(t.Context(), futuresTradablePair, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetKlineDataOfIndexPrice(t.Context(), futuresTradablePair, kline.FifteenMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKlineDataOfFairPrice(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetKlineDataOfFairPrice(t.Context(), currency.EMPTYPAIR, 0, startTime, endTime)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetKlineDataOfFairPrice(t.Context(), futuresTradablePair, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetKlineDataOfFairPrice(t.Context(), futuresTradablePair, kline.FifteenMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractTransactionData(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractTransactionData(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractTransactionData(t.Context(), futuresTradablePair, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractTickers(t *testing.T) {
	t.Parallel()
	result, err := e.GetContractTickers(t.Context(), futuresTradablePair)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	result, err = e.GetContractTickers(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetAllContractRiskFundBalance(t *testing.T) {
	t.Parallel()
	result, err := e.GetAllContractRiskFundBalance(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractRiskFundBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractRiskFundBalanceHistory(t.Context(), currency.EMPTYPAIR, 1, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetContractRiskFundBalanceHistory(t.Context(), futuresTradablePair, 0, 10)
	require.ErrorIs(t, err, errPageNumberRequired)
	_, err = e.GetContractRiskFundBalanceHistory(t.Context(), futuresTradablePair, 1, 0)
	require.ErrorIs(t, err, errPageSizeRequired)

	result, err := e.GetContractRiskFundBalanceHistory(t.Context(), futuresTradablePair, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractFundingRateHistory(t.Context(), currency.EMPTYPAIR, 1, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractFundingRateHistory(t.Context(), futuresTradablePair, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUserAssetsInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllUserAssetsInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserSingleCurrencyAssetInformation(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserSingleCurrencyAssetInformation(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserSingleCurrencyAssetInformation(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAssetTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAssetTransferRecords(t.Context(), currency.ETH, "WAIT", "IN", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserPositionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserPositionHistory(t.Context(), futuresTradablePair, "1", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersCurrentHoldingPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersCurrentHoldingPositions(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetUsersCurrentHoldingPositions(t.Context(), futuresTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersFundingRateDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersFundingRateDetails(t.Context(), futuresTradablePair, 123123, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserCurrentPendingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserCurrentPendingOrder(t.Context(), currency.EMPTYPAIR, 0, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserCurrentPendingOrder(t.Context(), futuresTradablePair, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUserHistoricalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllUserHistoricalOrders(t.Context(), futuresTradablePair, "1", "1", "1", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderBasedOnExternalNumber(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBasedOnExternalNumber(t.Context(), currency.EMPTYPAIR, "12312312")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetOrderBasedOnExternalNumber(t.Context(), futuresTradablePair, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderBasedOnExternalNumber(t.Context(), futuresTradablePair, "12312312")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderByOrderNumber(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderByOrderID(t.Context(), "12312312")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBatchOrdersByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetBatchOrdersByOrderID(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBatchOrdersByOrderID(t.Context(), []string{"123123"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderTransactionDetailsByOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderTransactionDetailsByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderTransactionDetailsByOrderID(t.Context(), "1232131")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserOrderAllTransactionDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserOrderAllTransactionDetails(t.Context(), currency.EMPTYPAIR, time.Time{}, time.Time{}, 1, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserOrderAllTransactionDetails(t.Context(), futuresTradablePair, time.Time{}, time.Time{}, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTriggerOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTriggerOrderList(t.Context(), currency.EMPTYPAIR, "1", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesStopLimitOrderList(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetFuturesStopLimitOrderList(t.Context(), futuresTradablePair, false, endTime, startTime, 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesStopLimitOrderList(t.Context(), futuresTradablePair, true, startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesRiskLimit(t.Context(), futuresTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesCurrentTradingFeeRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCurrentTradingFeeRate(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesCurrentTradingFeeRate(t.Context(), futuresTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIncreaseDecreaseMargin(t *testing.T) {
	t.Parallel()
	err := e.IncreaseDecreaseMargin(t.Context(), 0, 0, "ADD")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	err = e.IncreaseDecreaseMargin(t.Context(), 12312312, 0, "ADD")
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	err = e.IncreaseDecreaseMargin(t.Context(), 12312312, 1.5, "")
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.IncreaseDecreaseMargin(t.Context(), 1231231, 123.45, "SUB")
	assert.NoError(t, err)
}

func TestGetContractLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractLeverage(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetContractLeverage(t.Context(), futuresTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSwitchLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SwitchLeverage(t.Context(), 0, 25, 2, 1, currency.EMPTYPAIR)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.SwitchLeverage(t.Context(), 123333, 0, 2, 1, currency.EMPTYPAIR)
	require.ErrorIs(t, err, errMissingLeverage)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SwitchLeverage(t.Context(), 123333, 25, 2, 1, currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPositionMode(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangePositionMode(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePositionMode(t.Context(), 0)
	require.ErrorIs(t, err, errPositionModeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ChangePositionMode(t.Context(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	arg := &PlaceFuturesOrderParams{
		ReduceOnly: true,
	}
	_, err := e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	arg.Symbol = "BTC_USDT"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	arg.Price = 1234
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	arg.Volume = 3
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell
	arg.OrderType = order.Limit.String()
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.MarginType = margin.Multi
	result, err := e.PlaceFuturesOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrdersByID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrdersByID(t.Context())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CancelOrdersByID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelOrdersByID(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrderByClientOrderID(t.Context(), currency.EMPTYPAIR, "12345")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.CancelOrderByClientOrderID(t.Context(), spotTradablePair, "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelOrderByClientOrderID(t.Context(), spotTradablePair, "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CancelAllOpenOrders(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerUniversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerUniversalTransferHistory(t.Context(), asset.Empty, asset.Empty, "test1@thrasher.io", "test2@thrasher.io", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = e.GetBrokerUniversalTransferHistory(t.Context(), asset.Empty, asset.Empty, "test1@thrasher.io", "test2@thrasher.io", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errAddressRequired)

	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err = e.GetBrokerUniversalTransferHistory(t.Context(), asset.Futures, asset.Empty, "test1@thrasher.io", "test2@thrasher.io", startTime, endTime, 0, 10)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = e.GetBrokerUniversalTransferHistory(t.Context(), asset.Futures, asset.Spot, "test1@thrasher.io", "test2@thrasher.io", endTime, startTime, 1, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerUniversalTransferHistory(t.Context(), asset.Spot, asset.Futures, "test1@thrasher.io", "test2@thrasher.io", startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBrokerSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateBrokerSubAccount(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerAccountSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerAccountSubAccountList(t.Context(), "my-subaccount-name", 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountStatus(t.Context(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountStatus(t.Context(), "my-subaccount-name")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBrokerSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	arg := &BrokerSubAccountAPIKeyParams{
		IP: []string{"127.0.0.1"},
	}
	_, err := e.CreateBrokerSubAccountAPIKey(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	arg.SubAccount = "my-subaccount-name"
	_, err = e.CreateBrokerSubAccountAPIKey(t.Context(), arg)
	require.ErrorIs(t, err, errUnsupportedPermissionValue)

	arg.Permissions = []string{"SPOT_ACCOUNT_READ", "SPOT_ACCOUNT_WRITE"}
	_, err = e.CreateBrokerSubAccountAPIKey(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountNote)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.CreateBrokerSubAccountAPIKey(t.Context(), &BrokerSubAccountAPIKeyParams{
		SubAccount:  "my-subaccount-name",
		Permissions: []string{"SPOT_ACCOUNT_READ", "SPOT_ACCOUNT_WRITE"},
		Note:        "note-here",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerSubAccountAPIKey(t.Context(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerSubAccountAPIKey(t.Context(), "my-subaccount-name")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarshalStringList(t *testing.T) {
	t.Parallel()
	data := &struct {
		Data StringList `json:"data"`
	}{
		Data: []string{"SPOT_ACCOUNT_READ", "SPOT_ACCOUNT_WRITE"},
	}
	result, err := json.Marshal(data)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.JSONEq(t, `{"data":"SPOT_ACCOUNT_READ,SPOT_ACCOUNT_WRITE"}`, string(result))
}

func TestDeleteBrokerAPIKeySubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteBrokerAPIKeySubAccount(t.Context(), &BrokerSubAccountAPIKeyDeletionParams{APIKey: "api-key-here"})
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = e.DeleteBrokerAPIKeySubAccount(t.Context(), &BrokerSubAccountAPIKeyDeletionParams{SubAccount: "sub-account-detail-here"})
	require.ErrorIs(t, err, errAPIKeyMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateAPIEndpoints)
	result, err := e.DeleteBrokerAPIKeySubAccount(t.Context(), &BrokerSubAccountAPIKeyDeletionParams{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateBrokerSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateBrokerSubAccountDepositAddress(t.Context(), &BrokerSubAccountDepositAddressCreationParams{Network: "ERC20"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GenerateBrokerSubAccountDepositAddress(t.Context(), &BrokerSubAccountDepositAddressCreationParams{Coin: currency.ETH})
	require.ErrorIs(t, err, errNetworkNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GenerateBrokerSubAccountDepositAddress(t.Context(), &BrokerSubAccountDepositAddressCreationParams{Coin: currency.ETH, Network: "ERC20"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerSubAccountDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerSubAccountDepositAddress(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetSubAccountDepositHistory(t.Context(), currency.ETH, "1", endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountDepositHistory(t.Context(), currency.ETH, "1", startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllRecentSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetAllRecentSubAccountDepositHistory(t.Context(), currency.ETH, "1", endTime, startTime, 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllRecentSubAccountDepositHistory(t.Context(), currency.ETH, "1", startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	for k, v := range assetsAndErrors {
		result, err := e.FetchTradablePairs(t.Context(), k)
		require.ErrorIs(t, err, v)
		if v == nil {
			assert.NotNil(t, result)
		}
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), futuresTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.UpdateTicker(t.Context(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateTicker(t.Context(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	for k, v := range assetsAndErrors {
		err := e.UpdateTickers(t.Context(), k)
		require.ErrorIs(t, err, v)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateOrderbook(t.Context(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetHistoricCandles(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.TenMin, startTime, endTime)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	result, err := e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandlesExtended(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.TenMin, startTime, endTime)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Options, kline.TenMin, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, kline.FiveMin, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	sTime, err := e.GetServerTime(t.Context(), asset.Empty)
	require.NoError(t, err)
	assert.NotEmpty(t, sTime)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	require.NoErrorf(t, err, "Error fetching %s pairs for test: %v", asset.Spot, err)

	instrumentInfo, err := e.GetSymbols(t.Context(), []string{spotTradablePair.String()})
	require.NoError(t, err)
	require.NotEmpty(t, instrumentInfo.Symbols[0])

	lms, err := e.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
	require.NoError(t, err)
	require.NotNil(t, lms)

	symbolDetail := instrumentInfo.Symbols[0]
	require.NotNil(t, symbolDetail, "instrument required to be found")
	require.Equal(t, symbolDetail.QuoteAmountPrecision.Float64(), lms.PriceStepIncrementSize)
	assert.Equal(t, symbolDetail.BaseSizePrecision.Float64(), lms.MinimumBaseAmount)
	assert.Equal(t, symbolDetail.MaxQuoteAmount.Float64(), lms.MaximumQuoteAmount)

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Futures)
	require.NoErrorf(t, err, "Error fetching %s pairs for test: %v", asset.Spot, err)

	fInstrumentDetail, err := e.GetFuturesContracts(t.Context(), futuresTradablePair)
	require.NoError(t, err)
	require.NotEmpty(t, fInstrumentDetail.Data[0])

	lms, err = e.GetOrderExecutionLimits(asset.Futures, futuresTradablePair)
	require.NoError(t, err)

	fsymbolDetail := fInstrumentDetail.Data[0]
	require.NotNil(t, fsymbolDetail)
	assert.Equal(t, fsymbolDetail.PriceScale, lms.PriceStepIncrementSize)
	assert.Equal(t, fsymbolDetail.MinVol, lms.MinimumBaseAmount)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.Options,
		Pair:                 spotTradablePair,
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  futuresTradablePair,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.IsPerpetualFutureCurrency(asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.IsPerpetualFutureCurrency(asset.Spot, spotTradablePair)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	result, err := e.IsPerpetualFutureCurrency(asset.Futures, futuresTradablePair)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Binary)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateAccountBalances(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Empty)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), currency.EMPTYPAIR, asset.Options)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetRecentTrades(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetRecentTrades(t.Context(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetRecentTrades(t.Context(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1767204283384), time.UnixMilli(1767204403384)
	_, err := e.GetHistoricTrades(t.Context(), currency.EMPTYPAIR, asset.Options, startTime, endTime)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Options, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Spot, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricTrades(t.Context(), futuresTradablePair, asset.Futures, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "TON")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDepositAddress(t.Context(), currency.BTC, "", "TON")
	require.True(t, err != nil || err == deposit.ErrAddressNotFound)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	arg := &order.MultiOrderRequest{AssetType: asset.Options}
	_, err := e.GetActiveOrders(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	arg.Pairs = currency.Pairs{}
	_, err = e.GetActiveOrders(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	arg.Pairs = currency.Pairs{spotTradablePair}
	_, err = e.GetActiveOrders(t.Context(), arg)
	require.NoError(t, err)

	arg.AssetType = asset.Futures
	arg.Pairs = currency.Pairs{futuresTradablePair}
	_, err = e.GetActiveOrders(t.Context(), arg)
	require.NoError(t, err)
}

func TestGenerateListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	listenKey, err := e.GenerateListenKey(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, listenKey)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "12342", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetOrderInfo(t.Context(), "12342", spotTradablePair, asset.Spot)
	assert.NoError(t, err)

	_, err = e.GetOrderInfo(t.Context(), "12342", futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = e.SubmitOrder(t.Context(), &order.Submit{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg := &order.Submit{
		Pair:      spotTradablePair,
		AssetType: asset.Options,
		Type:      order.Liquidation,
		Side:      order.Long,
	}
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrAssetNotFound)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.Pair = spotTradablePair
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.Pair = spotTradablePair
	arg.AssetType = asset.Spot
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	// Spot orders test
	arg.Type = order.Limit
	arg.Side = order.Sell
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = .1
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.Price = 1234567
	result, err := e.SubmitOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Futures orders test
	arg.AssetType = asset.Futures
	arg.Pair = futuresTradablePair
	arg.Amount = 1
	arg.TimeInForce = order.ImmediateOrCancel
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	arg.MarginType = margin.Multi
	result, err = e.SubmitOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	err = e.CancelOrder(t.Context(), &order.Cancel{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654"})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", AssetType: asset.Spot})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", Pair: spotTradablePair, AssetType: asset.Spot})
	assert.NoError(t, err)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", Pair: futuresTradablePair, AssetType: asset.Futures})
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345"})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345", AssetType: asset.Spot})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345", AssetType: asset.Spot, Pair: spotTradablePair})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345", AssetType: asset.Futures, Pair: futuresTradablePair})
	require.NoError(t, err)
	assert.NotNil(t, result)
}
