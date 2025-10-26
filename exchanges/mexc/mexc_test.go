package mexc

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
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

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	if err := populateTradablePairs(); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func populateTradablePairs() error {
	if err := e.UpdateTradablePairs(context.Background()); err != nil {
		return err
	}
	tradablePairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	spotTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Spot)
	if err != nil {
		return err
	}
	tradablePairs, err = e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	futuresTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Futures)
	return err
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
	_, err := e.GetOrderbook(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetOrderbook(t.Context(), "BTCUSDT", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTradesList(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTradesList(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetRecentTradesList(t.Context(), "BTCUSDT", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetAggregatedTrades(t.Context(), "", time.Now().Add(-time.Hour*1), time.Now(), 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetAggregatedTrades(t.Context(), "BTCUSDT", time.Now().Add(-time.Hour*1), time.Now(), 0)
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

	_, err = e.GetCandlestick(t.Context(), "", intervalString, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetCandlestick(t.Context(), "BTCUSDT", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetCandlestick(t.Context(), "BTCUSDT", "5m", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentAveragePrice(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetCurrentAveragePrice(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGet24HourTickerPriceChangeStatistics(t *testing.T) {
	t.Parallel()
	result, err := e.Get24HourTickerPriceChangeStatistics(t.Context(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.Get24HourTickerPriceChangeStatistics(t.Context(), []string{"BTCUSDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	result, err := e.GetSymbolPriceTicker(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetSymbolPriceTicker(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolsPriceTicker(t *testing.T) {
	result, err := e.GetSymbolsPriceTicker(t.Context(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetSymbolsPriceTicker(t.Context(), []string{"BTCUSDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbolOrderbookTicker(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetSymbolOrderbookTicker(t.Context(), "BTCUSDT")
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
	assert.ErrorIs(t, err, errInvalidSubAccountName)

	_, err = e.CreateSubAccount(t.Context(), "Test1", "")
	assert.ErrorIs(t, err, errInvalidSubAccountNote)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateSubAccount(t.Context(), "Test1", "sub-account notes")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountList(t.Context(), "", false, 1, 10)
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
	assert.ErrorIs(t, err, errInvalidSubAccountName)

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
	_, err := e.NewTestOrder(t.Context(), "", "123123", "SELL", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.NewTestOrder(t.Context(), "BTCUSDT", "123123", "", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.NewTestOrder(t.Context(), "BTCUSDT", "123123", "SELL", "", 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.NewTestOrder(t.Context(), "BTCUSDT", "123123", "SELL", typeLimit, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.NewTestOrder(t.Context(), "BTCUSDT", "123123", "SELL", typeLimit, 1, 0, 0)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.NewTestOrder(t.Context(), "BTCUSDT", "123123", "SELL", typeMarket, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.NewTestOrder(t.Context(), "BTCUSDT", "123123", "SELL", typeLimit, 1, 0, 123456.78)
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
	_, err := e.NewOrder(t.Context(), "", "123123", "SELL", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.NewOrder(t.Context(), "BTCUSDT", "123123", "", typeLimit, 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.NewOrder(t.Context(), "BTCUSDT", "123123", "SELL", "", 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.NewOrder(t.Context(), "BTCUSDT", "123123", "SELL", typeLimit, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.NewOrder(t.Context(), "BTCUSDT", "123123", "SELL", typeLimit, 1, 0, 0)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.NewOrder(t.Context(), "BTCUSDT", "123123", "SELL", typeMarket, 0, 0, 123456.78)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewOrder(t.Context(), spotTradablePair.String(), "123123", "BUY", typeLimit, 1, 0, 123456.78)
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

	arg.Symbol = "BTCUSDT"
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
			assert.ErrorIs(t, err, typesMap[a].Error)
		})
	}
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelTradeOrder(t.Context(), "", "", "", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.CancelTradeOrder(t.Context(), "BTCUSDT", "", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelTradeOrder(t.Context(), "BTCUSDT", "1234", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOpenOrdersBySymbol(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOpenOrdersBySymbol(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderByID(t.Context(), "", "123455", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetOrderByID(t.Context(), "BTCUSDT", "", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderByID(t.Context(), "BTCUSDT", "1234", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrders(t.Context(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllOrders(t.Context(), "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllOrders(t.Context(), "BTCUSDT", time.Time{}, time.Time{}, 10)
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
	_, err := e.GetAccountTradeList(t.Context(), "", "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountTradeList(t.Context(), "BTCUSDT", "", time.Time{}, time.Time{}, 10)
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
	result, err := e.GetSymbolTradingFee(t.Context(), "BTCUSDT")
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundDepositHistory(t.Context(), currency.BTC, "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalHistory(t.Context(), currency.USDT, time.Time{}, time.Time{}, 0, 10)
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
	_, err := e.UserUniversalTransfer(t.Context(), "", "SPOT", currency.USDT, 1000)
	require.ErrorIs(t, err, errAccountTypeRequired)
	_, err = e.UserUniversalTransfer(t.Context(), "FUTURE", "", currency.USDT, 1000)
	require.ErrorIs(t, err, errAccountTypeRequired)
	_, err = e.UserUniversalTransfer(t.Context(), "FUTURE", "SPOT", currency.EMPTYCODE, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.UserUniversalTransfer(t.Context(), "FUTURE", "SPOT", currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UserUniversalTransfer(t.Context(), "FUTURE", "SPOT", currency.USDT, 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUniversalTransferHistory(t.Context(), "", "FUTURES", time.Now().Add(-time.Hour*20), time.Now(), 0, 10)
	require.ErrorIs(t, err, errAccountTypeRequired)
	_, err = e.GetUniversalTransferHistory(t.Context(), "SPOT", "", time.Now().Add(-time.Hour*20), time.Now(), 0, 10)
	require.ErrorIs(t, err, errAccountTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUniversalTransferHistory(t.Context(), "SPOT", "FUTURES", time.Now().Add(-time.Hour*20), time.Now(), 0, 10)
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
	_, err := e.DustLog(t.Context(), time.Time{}, time.Time{}, 0, 0)
	require.ErrorIs(t, err, errPaginationLimitIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.DustLog(t.Context(), time.Time{}, time.Time{}, 0, 10)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetInternalTransferHistory(t.Context(), "11945860693", time.Time{}, time.Time{}, 0, 10)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRebateHistoryRecords(t.Context(), time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRebateRecordsDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRebateRecordsDetail(t.Context(), time.Now().Add(-time.Hour*48), time.Now(), 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSelfRebateRecordsDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSelfRebateRecordsDetail(t.Context(), time.Time{}, time.Time{}, 10)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateCommissionRecord(t.Context(), time.Time{}, time.Time{}, "abcdef", 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateWithdrawRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateWithdrawRecord(t.Context(), time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCommissionDetailRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateCommissionDetailRecord(t.Context(), time.Time{}, time.Time{}, "", "1", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCampaignData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateCampaignData(t.Context(), time.Now().Add(-time.Hour*480), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateReferralData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAffiliateReferralData(t.Context(), time.Time{}, time.Time{}, "", "", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAffiliateData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAffiliateData(t.Context(), time.Time{}, time.Time{}, "", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractsDetail(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesContracts(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetFuturesContracts(t.Context(), result.Data[0].Symbol)
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
	_, err := e.GetContractOrderbook(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractOrderbook(t.Context(), "BTC_USDT", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepthSnapshotOfContract(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepthSnapshotOfContract(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetDepthSnapshotOfContract(t.Context(), "BTC_USDT", 0)
	require.ErrorIs(t, err, errPaginationLimitIsRequired)

	result, err := e.GetDepthSnapshotOfContract(t.Context(), "BTC_USDT", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractIndexPrice(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractIndexPrice(t.Context(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractFairPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractFairPrice(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractFairPrice(t.Context(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractFundingPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractFundingPrice(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractFundingPrice(t.Context(), "BTC_USDT")
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
	_, err := e.GetContractsCandlestickData(t.Context(), "", 0, time.Now().Add(-time.Hour*480), time.Now())
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractsCandlestickData(t.Context(), "BTC_USDT", kline.FifteenMin, time.Now().Add(-time.Hour*480), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKlineDataOfIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetKlineDataOfIndexPrice(t.Context(), "", 0, time.Now().Add(-time.Hour*480), time.Now())
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetKlineDataOfIndexPrice(t.Context(), "BTC_USDT", kline.FifteenMin, time.Now().Add(-time.Hour*480), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKlineDataOfFairPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetKlineDataOfFairPrice(t.Context(), "", 0, time.Now().Add(-time.Hour*480), time.Now())
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetKlineDataOfFairPrice(t.Context(), "BTC_USDT", kline.FifteenMin, time.Now().Add(-time.Hour*480), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractTransactionData(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractTransactionData(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractTransactionData(t.Context(), "BTC_USDT", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractTrendData(t *testing.T) {
	t.Parallel()
	result, err := e.GetContractTickers(t.Context(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	result, err = e.GetContractTickers(t.Context(), "")
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
	_, err := e.GetContractRiskFundBalanceHistory(t.Context(), "", 1, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetContractRiskFundBalanceHistory(t.Context(), "BTC_USDT", 0, 10)
	require.ErrorIs(t, err, errPageNumberRequired)
	_, err = e.GetContractRiskFundBalanceHistory(t.Context(), "BTC_USDT", 1, 0)
	require.ErrorIs(t, err, errPageSizeRequired)

	result, err := e.GetContractRiskFundBalanceHistory(t.Context(), "BTC_USDT", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractFundingRateHistory(t.Context(), "", 1, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractFundingRateHistory(t.Context(), "BTC_USDT", 1, 10)
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
	result, err := e.GetUserPositionHistory(t.Context(), "BTC_USDT", "1", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersCurrentHoldingPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersCurrentHoldingPositions(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetUsersCurrentHoldingPositions(t.Context(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersFundingRateDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersFundingRateDetails(t.Context(), "BTC_USDT", 123123, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserCurrentPendingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserCurrentPendingOrder(t.Context(), "", 0, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserCurrentPendingOrder(t.Context(), "BTC_USDT", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUserHistoricalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllUserHistoricalOrders(t.Context(), "BTC_USDT", "1", "1", "1", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderBasedOnExternalNumber(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBasedOnExternalNumber(t.Context(), "", "12312312")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetOrderBasedOnExternalNumber(t.Context(), "BTC_USDT", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderBasedOnExternalNumber(t.Context(), "BTC_USDT", "12312312")
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
	_, err := e.GetUserOrderAllTransactionDetails(t.Context(), "", time.Time{}, time.Time{}, 1, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserOrderAllTransactionDetails(t.Context(), "BTC_USDT", time.Time{}, time.Time{}, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTriggerOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTriggerOrderList(t.Context(), "", "1", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesStopLimitOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesStopLimitOrderList(t.Context(), "BTC_USDT", false, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesRiskLimit(t.Context(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesCurrentTradingFeeRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCurrentTradingFeeRate(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesCurrentTradingFeeRate(t.Context(), "BTC_USDT")
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
	_, err := e.GetContractLeverage(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetContractLeverage(t.Context(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSwitchLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SwitchLeverage(t.Context(), 0, 25, 2, 1, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.SwitchLeverage(t.Context(), 123333, 0, 2, 1, "")
	require.ErrorIs(t, err, errMissingLeverage)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SwitchLeverage(t.Context(), 123333, 25, 2, 1, "")
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
	_, err := e.CancelOrderByClientOrderID(t.Context(), "", "12345")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.CancelOrderByClientOrderID(t.Context(), "BTC_USDT", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelOrderByClientOrderID(t.Context(), "BTC_USDT", "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CancelAllOpenOrders(t.Context(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerUniversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBrokerUniversalTransferHistory(t.Context(), "", "", "test1@thrasher.io", "test2@thrasher.io", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = e.GetBrokerUniversalTransferHistory(t.Context(), "FUTURES", "", "test1@thrasher.io", "test2@thrasher.io", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBrokerUniversalTransferHistory(t.Context(), "SPOT", "FUTURES", "test1@thrasher.io", "test2@thrasher.io", time.Time{}, time.Time{}, 0, 10)
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
	result, err := e.CreateBrokerSubAccountAPIKey(t.Context(), &BrokerSubAccountAPIKeyParams{SubAccount: "my-subaccount-name", Permissions: []string{"SPOT_ACCOUNT_READ", "SPOT_ACCOUNT_WRITE"}, Note: "note-here"})
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountDepositHistory(t.Context(), currency.ETH, "1", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllRecentSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllRecentSubAccountDepositHistory(t.Context(), currency.ETH, "1", time.Time{}, time.Time{}, 0, 10)
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
		assert.ErrorIs(t, err, v)
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
	_, err := e.GetHistoricCandles(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.TenMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	_, err = e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Options, kline.TenMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.FiveMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*5))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandlesExtended(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.TenMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	_, err = e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Options, kline.TenMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2))
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, kline.FiveMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*5))
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

	fInstrumentDetail, err := e.GetFuturesContracts(t.Context(), futuresTradablePair.String())
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
	_, err := e.GetHistoricTrades(t.Context(), currency.EMPTYPAIR, asset.Options, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Options, time.Time{}, time.Time{})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Spot, time.Now().Add(-time.Minute*4), time.Now().Add(-time.Minute*2))
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricTrades(t.Context(), futuresTradablePair, asset.Futures, time.Now().Add(-time.Minute*4), time.Now().Add(-time.Minute*2))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "TON")
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

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

func TestWsHandle(t *testing.T) {
	t.Parallel()
	pushDataMap := map[string]string{
		"spot@public.aggre.deals.v3.api.pb":          `{"channel": "spot@public.aggre.deals.v3.api.pb@100ms@BTCUSDT", "publicdeals": { "dealsList": [ { "price": "93220.00", "quantity": "0.04438243", "tradetype": 2, "time": 1736409765051 } ], "eventtype": "spot@public.aggre.deals.v3.api.pb@100ms" }, "symbol": "BTCUSDT", "sendtime": 1736409765052 }`,
		"spot@public.kline.v3.api.pb":                `{"channel": "spot@public.kline.v3.api.pb@BTCUSDT@Min15", "publicspotkline": { "interval": "Min15", "windowstart": 1736410500, "openingprice": "92925", "closingprice": "93158.47", "highestprice": "93158.47", "lowestprice": "92800", "volume": "36.83803224", "amount": "3424811.05", "windowend": 1736411400 }, "symbol": "BTCUSDT", "symbolid": "2fb942154ef44a4ab2ef98c8afb6a4a7", "createtime": 1736410707571}`,
		"spot@public.aggre.depth.v3.api.pb":          `{"channel": "spot@public.aggre.depth.v3.api.pb@100ms@BTCUSDT", "publicincreasedepths": { "asksList": [], "bidsList": [ { "price": "92877.58", "quantity": "0.00000000" } ], "eventtype": "spot@public.aggre.depth.v3.api.pb@100ms", "version": "36913293511" }, "symbol": "BTCUSDT", "sendtime": 1736411507002}`,
		"spot@public.increase.depth.batch.v3.api.pb": `{"channel" : "spot@public.increase.depth.batch.v3.api.pb@BTCUSDT", "symbol" : "BTCUSDT", "sendTime" : "1739502064578", "publicIncreaseDepthsBatch" : { "items" : [ { "asks" : [ ], "bids" : [ { "price" : "96578.48", "quantity" : "0.00000000" } ], "eventType" : "", "version" : "39003145507" }, { "asks" : [ ], "bids" : [ { "price" : "96578.90", "quantity" : "0.00000000" } ], "eventType" : "", "version" : "39003145508" }, { "asks" : [ ], "bids" : [ { "price" : "96579.31", "quantity" : "0.00000000" } ], "eventType" : "", "version" : "39003145509" }, { "asks" : [ ], "bids" : [ { "price" : "96579.84", "quantity" : "0.00000000" } ], "eventType" : "", "version" : "39003145510" }, { "asks" : [ ], "bids" : [ { "price" : "96576.69", "quantity" : "4.88725694" } ], "eventType" : "", "version" : "39003145511" } ], "eventType" : "spot@public.increase.depth.batch.v3.api.pb"}}`,
		"spot@public.limit.depth.v3.api.pb":          `{"channel": "spot@public.limit.depth.v3.api.pb@BTCUSDT@5", "publiclimitdepths": { "asksList": [ { "price": "93180.18", "quantity": "0.21976424" } ], "bidsList": [ { "price": "93179.98", "quantity": "2.82651000" } ], "eventtype": "spot@public.limit.depth.v3.api.pb", "version": "36913565463" }, "symbol": "BTCUSDT", "sendtime": 1736411838730}`,
		"spot@public.aggre.bookTicker.v3.api.pb":     `{"channel": "spot@public.aggre.bookTicker.v3.api.pb@100ms@BTCUSDT", "publicbookticker": { "bidprice": "93387.28", "bidquantity": "3.73485", "askprice": "93387.29", "askquantity": "7.669875" }, "symbol": "BTCUSDT", "sendtime": 1736412092433 }`,
		"spot@public.bookTicker.batch.v3.api.pb":     `{"channel" : "spot@public.bookTicker.batch.v3.api.pb@BTCUSDT", "symbol" : "BTCUSDT", "sendTime" : "1739503249114", "publicBookTickerBatch" : { "items" : [ { "bidPrice" : "96567.37", "bidQuantity" : "3.362925", "askPrice" : "96567.38", "askQuantity":"1.545255"}]}}`,
		"spot@private.deals.v3.api.pb":               `{channel: "spot@private.deals.v3.api.pb", symbol: "MXUSDT", sendTime: 1736417034332, privateDeals { price: "3.6962", quantity: "1", amount: "3.6962", tradeType: 2, tradeId: "505979017439002624X1", orderId: "C02__505979017439002624115", feeAmount: "0.0003998377369698171", feeCurrency: "MX", time: 1736417034280}}`,
		"spot@private.orders.v3.api.pb":              `{channel: "spot@private.orders.v3.api.pb", symbol: "MXUSDT", sendTime: 1736417034281, privateOrders { id: "C02__505979017439002624115", price: "3.5121", quantity: "1", amount: "0", avgPrice: "3.6962", orderType: 5, tradeType: 2, remainAmount: "0", remainQuantity: "0", lastDealQuantity: "1", cumulativeQuantity: "1", cumulativeAmount: "3.6962", status: 2, createTime: 1736417034259}}`,
		"spot@private.account.v3.api.pb":             `{channel: "spot@private.account.v3.api.pb", createTime: 1736417034305, sendTime: 1736417034307, privateAccount { vcoinName: "USDT", coinId: "128f589271cb4951b03e71e6323eb7be", balanceAmount: "21.94210356004384", balanceAmountChange: "10", frozenAmount: "0", frozenAmountChange: "0", type: "CONTRACT_TRANSFER", time: 1736416910000}}`,
	}
	for elem := range pushDataMap {
		err := e.WsHandleData([]byte(pushDataMap[elem]))
		assert.NoErrorf(t, err, "%v: %s", err, elem)
	}
}

func TestWsHandleFuturesData(t *testing.T) {
	t.Parallel()
	futuresWsPushDataMap := map[string]string{
		"sub.tickers":                 `{"channel": "push.tickers", "data": [ { "fairPrice": 183.01, "lastPrice": 183, "riseFallRate": -0.0708, "symbol": "BSV_USDT", "volume24": 200 }, { "fairPrice": 220.22, "lastPrice": 220.4, "riseFallRate": -0.0686, "symbol": "BCH_USDT", "volume24": 200 } ], "ts": 1587442022003}`,
		"push.ticker":                 `{"symbol":"LINK_USDT","data":{"symbol":"LINK_USDT","lastPrice":14.022,"riseFallRate":-0.0270,"fairPrice":14.022,"indexPrice":14.028,"volume24":104524120,"amount24":149228107.8277,"maxBidPrice":16.833,"minAskPrice":11.222,"lower24Price":13.967,"high24Price":14.518,"timestamp":1746351275382,"bid1":14.02,"ask1":14.021,"holdVol":14558875,"riseFallValue":-0.390,"fundingRate":-0.000045,"zone":"UTC+8","riseFallRates":[-0.0270,-0.0594,0.1172,-0.3674,0.3499,0.0065],"riseFallRatesOfTimezone":[-0.0238,-0.0153,-0.0270]},"channel":"push.ticker","ts":1746351275382}`,
		"push.deal":                   `{"symbol":"IOTA_USDT","data":[{"p":0.1834,"v":97,"T":1,"O":1,"M":2,"t":1748810708074}],"channel":"push.deal","ts":1748810708074}`,
		"sub.depth":                   `{"channel":"push.depth", "data":{ "asks":[ [ 6859.5, 3251, 1 ] ], "bids":[ ], "version":96801927 }, "symbol":"BTC_USDT", "ts":1587442022003}`,
		"push.kline":                  `{"symbol":"CHEEMS_USDT","data":{"symbol":"CHEEMS_USDT","interval":"Min15","t":1746351000,"o":0.0000015036,"c":0.0000014988,"h":0.0000015036,"l":0.0000014962,"a":1183.078,"q":79,"ro":0.0000015021,"rc":0.0000014988,"rh":0.0000015021,"rl":0.0000014962},"channel":"push.kline","ts":1746351123147}`,
		"sub.funding.rate":            `{"channel":"push.funding.rate", "data":{ "rate":0.001, "symbol":"BTC_USDT" }, "symbol":"BTC_USDT", "ts":1587442022003 }`,
		"push.index.price":            `{"symbol":"BSV_USDT","data":{"symbol":"BSV_USDT","price":36.64},"channel":"push.index.price","ts":1746351370315}`,
		"push.fair.price":             `{"symbol":"YZYSOL_USDT","data":{"symbol":"YZYSOL_USDT","price":0.00278},"channel":"push.fair.price","ts":1746351543720}`,
		"push.personal.order":         `{"channel":"push.personal.order", "data":{ "category":1, "createTime":1610005069976, "dealAvgPrice":0.731, "dealVol":1, "errorCode":0, "externalOid":"_m_95bc2b72d3784bce8f9efecbdef9fe35", "feeCurrency":"USDT", "leverage":0, "makerFee":0, "openType":1, "orderId":"102067003631907840", "orderMargin":0, "orderType":5, "positionId":1397818, "price":0.707, "profit":-0.0005, "remainVol":0, "side":4, "state":3, "symbol":"CRV_USDT", "takerFee":0.00004386, "updateTime":1610005069983, "usedMargin":0, "version":2, "vol":1 }, "ts":1610005069989}`,
		"push.personal.asset":         `{"channel":"push.personal.asset", "data":{ "availableBalance":0.7514236, "bonus":0, "currency":"USDT", "frozenBalance":0, "positionMargin":0 }, "ts":1610005070083}`,
		"push.personal.position":      `{"channel":"push.personal.position", "data":{ "autoAddIm":false, "closeAvgPrice":0.731, "closeVol":1, "frozenVol":0, "holdAvgPrice":0.736, "holdFee":0, "holdVol":0, "im":0, "leverage":15, "liquidatePrice":0, "oim":0, "openAvgPrice":0.736, "openType":1, "positionId":1397818, "positionType":1, "realised":-0.0005, "state":3, "symbol":"CRV_USDT" },"ts":1610005070157}`,
		"push.personal.adl.level":     `{"channel":"push.personal.adl.level", "data":{ "adlLevel":0, "positionId":1397818 }, "ts":1610005032231 }`,
		"push.personal.position.mode": `{"channel":"push.personal.position.mode", "data":{ "positionMode": 1 }, "ts":1610005070157}`,
		"push.fullDepth":              `{"symbol":"INIT_USDT","data":{"asks":[[0.7542,1484,1],[0.7543,4676,2],[0.7544,11626,2],[0.7545,8247,1],[0.7546,20469,1],[0.7547,10241,1],[0.7548,26518,1],[0.7549,10490,1],[0.755,21088,1],[0.7551,16653,1],[0.7552,22110,1],[0.7553,26518,1],[0.7554,26252,1],[0.7555,16962,1],[0.7556,26518,1],[0.7557,16926,1],[0.7558,18085,1],[0.7559,16484,1],[0.756,26518,1],[0.7561,9654,1]],"bids":[[0.7541,374,1],[0.754,3186,3],[0.7539,3995,1],[0.7538,10560,1],[0.7537,12689,1],[0.7536,14731,1],[0.7535,18077,1],[0.7534,11203,1],[0.7533,9609,1],[0.7532,20530,1],[0.7531,10936,1],[0.753,11492,1],[0.7529,13563,1],[0.7528,15658,1],[0.7527,10737,1],[0.7526,15113,1],[0.7525,20870,1],[0.7524,13257,1],[0.7523,16629,1],[0.7522,10854,1]],"version":197614550},"channel":"push.depth.full","ts":1748810839220}`,
	}
	for elem := range futuresWsPushDataMap {
		t.Run(elem, func(t *testing.T) {
			t.Parallel()
			err := e.WsHandleFuturesData([]byte(futuresWsPushDataMap[elem]))
			assert.NoErrorf(t, err, "%v: %s", err, elem)
		})
	}
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
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	arg.Pair = spotTradablePair
	_, err = e.SubmitOrder(t.Context(), arg)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	arg.Pair = spotTradablePair
	arg.AssetType = asset.Spot
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	// Spot orders test
	arg.Type = order.Limit
	arg.Side = order.Sell
	_, err = e.SubmitOrder(t.Context(), arg)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = .1
	_, err = e.SubmitOrder(t.Context(), arg)
	assert.ErrorIs(t, err, limits.ErrPriceBelowMin)

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
	assert.ErrorIs(t, err, margin.ErrInvalidMarginType)

	arg.MarginType = margin.Multi
	result, err = e.SubmitOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelOrder(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	err = e.CancelOrder(t.Context(), &order.Cancel{})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654"})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", AssetType: asset.Spot})
	assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", Pair: spotTradablePair, AssetType: asset.Spot})
	assert.NoError(t, err)

	err = e.CancelOrder(t.Context(), &order.Cancel{OrderID: "987654", Pair: futuresTradablePair, AssetType: asset.Futures})
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345"})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345", AssetType: asset.Spot})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345", AssetType: asset.Spot, Pair: spotTradablePair})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.CancelAllOrders(t.Context(), &order.Cancel{OrderID: "12345", AssetType: asset.Futures, Pair: futuresTradablePair})
	require.NoError(t, err)
	assert.NotNil(t, result)
}
