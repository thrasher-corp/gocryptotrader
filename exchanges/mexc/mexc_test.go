package mexc

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	me = &MEXC{}

	assetsAndErrors = map[asset.Item]error{
		asset.Spot:    nil,
		asset.Futures: nil,
		asset.Options: asset.ErrNotSupported,
	}

	spotTradablePair, futuresTradablePair currency.Pair
)

func TestMain(m *testing.M) {
	me = new(MEXC)
	if err := testexch.Setup(me); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		me.API.AuthenticatedSupport = true
		me.API.AuthenticatedWebsocketSupport = true
		me.SetCredentials(apiKey, apiSecret, "", "", "", "")
		me.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	err := populateTradablePairs()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func populateTradablePairs() error {
	err := me.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		return err
	}
	tradablePairs, err := me.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	spotTradablePair, err = me.FormatExchangeCurrency(tradablePairs[0], asset.Spot)
	if err != nil {
		return err
	}
	tradablePairs, err = me.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	futuresTradablePair, err = me.FormatExchangeCurrency(tradablePairs[0], asset.Futures)
	if err != nil {
		return err
	}
	return nil
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	result, err := me.GetSymbols(context.Background(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	result, err := me.GetSystemTime(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDefaultSumbols(t *testing.T) {
	t.Parallel()
	result, err := me.GetDefaultSumbols(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := me.GetOrderbook(context.Background(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetOrderbook(context.Background(), "BTCUSDT", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTradesList(t *testing.T) {
	t.Parallel()
	_, err := me.GetRecentTradesList(context.Background(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetRecentTradesList(context.Background(), "BTCUSDT", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := me.GetAggregatedTrades(context.Background(), "", time.Now().Add(-time.Hour*1), time.Now(), 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetAggregatedTrades(context.Background(), "BTCUSDT", time.Now().Add(-time.Hour*1), time.Now(), 0)
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

	_, err = me.GetCandlestick(context.Background(), "", intervalString, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = me.GetCandlestick(context.Background(), "BTCUSDT", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := me.GetCandlestick(context.Background(), "BTCUSDT", "5m", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := me.GetCurrentAveragePrice(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetCurrentAveragePrice(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGet24HourTickerPriceChangeStatistics(t *testing.T) {
	t.Parallel()
	result, err := me.Get24HourTickerPriceChangeStatistics(context.Background(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.Get24HourTickerPriceChangeStatistics(context.Background(), []string{"BTCUSDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	result, err := me.GetSymbolPriceTicker(context.Background(), []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetSymbolPriceTicker(context.Background(), []string{"BTCUSDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	result, err := me.GetSymbolOrderbookTicker(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetSymbolOrderbookTicker(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSubAccount(t *testing.T) {
	t.Parallel()
	_, err := me.CreateSubAccount(context.Background(), "", "sub-account notes")
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = me.CreateSubAccount(context.Background(), "Test1", "")
	require.ErrorIs(t, err, errInvalidSubAccountNote)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CreateSubAccount(context.Background(), "Test1", "sub-account notes")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSubAccountList(context.Background(), "", false, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateAPIKeyForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := me.CreateAPIKeyForSubAccount(context.Background(), "", "123", "SPOT_DEAL_WRITE", "")
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = me.CreateAPIKeyForSubAccount(context.Background(), "SubAcc2", "", "SPOT_DEAL_WRITE", "")
	require.ErrorIs(t, err, errInvalidSubAccountNote)
	_, err = me.CreateAPIKeyForSubAccount(context.Background(), "SubAcc2", "123", "", "")
	require.ErrorIs(t, err, errUnsupportedPermissionValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CreateAPIKeyForSubAccount(context.Background(), "SubAcc2", "123", "SPOT_DEAL_WRITE", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := me.GetSubAccountAPIKey(context.Background(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSubAccountAPIKey(context.Background(), "SubAcc1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteAPIKeySubAccount(t *testing.T) {
	t.Parallel()
	_, err := me.DeleteAPIKeySubAccount(context.Background(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.DeleteAPIKeySubAccount(context.Background(), "SubAcc1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := me.SubAccountUniversalTransfer(context.Background(), "master@test.com", "subaccount@test.com", asset.Empty, asset.Futures, currency.USDT, 1234.)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = me.SubAccountUniversalTransfer(context.Background(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Empty, currency.USDT, 1234.)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = me.SubAccountUniversalTransfer(context.Background(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Futures, currency.EMPTYCODE, 1234.)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = me.SubAccountUniversalTransfer(context.Background(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Futures, currency.USDT, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.SubAccountUniversalTransfer(context.Background(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Futures, currency.USDT, 1234.)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountUnversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := me.GetSubAccountUnversalTransferHistory(context.Background(), "master@test.com", "subaccount@test.com", asset.Empty, asset.Futures, time.Now().Add(-time.Hour*50), time.Now().Add(-time.Hour*20), 10, 20)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = me.GetSubAccountUnversalTransferHistory(context.Background(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Empty, time.Now().Add(-time.Hour*50), time.Now().Add(-time.Hour*20), 10, 20)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSubAccountUnversalTransferHistory(context.Background(), "master@test.com", "subaccount@test.com", asset.Spot, asset.Futures, time.Now().Add(-time.Hour*50), time.Now().Add(-time.Hour*20), 10, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountAsset(t *testing.T) {
	t.Parallel()
	_, err := me.GetSubAccountAsset(context.Background(), "", asset.Spot)
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = me.GetSubAccountAsset(context.Background(), "thesubaccount@test.com", asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSubAccountAsset(context.Background(), "thesubaccount@test.com", asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKYCStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetKYCStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUseAPIDefaultSymbols(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.UseAPIDefaultSymbols(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewTestOrder(t *testing.T) {
	t.Parallel()
	_, err := me.NewTestOrder(context.Background(), "", "123123", "SELL", "LIMIT_ORDER", 1, 0, 123456.78)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = me.NewTestOrder(context.Background(), "BTCUSDT", "123123", "", "LIMIT_ORDER", 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = me.NewTestOrder(context.Background(), "BTCUSDT", "123123", "SELL", "", 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = me.NewTestOrder(context.Background(), "BTCUSDT", "123123", "SELL", "LIMIT_ORDER", 0, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = me.NewTestOrder(context.Background(), "BTCUSDT", "123123", "SELL", "LIMIT_ORDER", 1, 0, 0)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = me.NewTestOrder(context.Background(), "BTCUSDT", "123123", "SELL", "MARKET_ORDER", 0, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.NewTestOrder(context.Background(), "BTCUSDT", "123123", "SELL", "LIMIT_ORDER", 1, 0, 123456.78)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := me.NewOrder(context.Background(), "", "123123", "SELL", "LIMIT_ORDER", 1, 0, 123456.78)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = me.NewOrder(context.Background(), "BTCUSDT", "123123", "", "LIMIT_ORDER", 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = me.NewOrder(context.Background(), "BTCUSDT", "123123", "SELL", "", 1, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = me.NewOrder(context.Background(), "BTCUSDT", "123123", "SELL", "LIMIT_ORDER", 0, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = me.NewOrder(context.Background(), "BTCUSDT", "123123", "SELL", "LIMIT_ORDER", 1, 0, 0)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = me.NewOrder(context.Background(), "BTCUSDT", "123123", "SELL", "MARKET_ORDER", 0, 0, 123456.78)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	// sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.NewOrder(context.Background(), "BTCUSDT", "123123", "SELL", "LIMIT_ORDER", 1, 0, 123456.78)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBatchOrder(t *testing.T) {
	t.Parallel()
	_, err := me.CreateBatchOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = me.CreateBatchOrder(context.Background(), []BatchOrderCreationParam{{}})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := BatchOrderCreationParam{
		NewClientOrderID: 1234,
	}
	_, err = me.CreateBatchOrder(context.Background(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTCUSDT"
	_, err = me.CreateBatchOrder(context.Background(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.String()
	_, err = me.CreateBatchOrder(context.Background(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.OrderType, err = me.OrderTypeStringFromOrderTypeAndTimeInForce(order.Limit, order.UnknownTIF)
	require.NoError(t, err)
	_, err = me.CreateBatchOrder(context.Background(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Quantity = 123478.5
	_, err = me.CreateBatchOrder(context.Background(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.OrderType, err = me.OrderTypeStringFromOrderTypeAndTimeInForce(order.Market, order.UnknownTIF)
	require.NoError(t, err)

	arg.Quantity = 0
	_, err = me.CreateBatchOrder(context.Background(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	arg.Quantity = 1231231
	result, err := me.CreateBatchOrder(context.Background(), []BatchOrderCreationParam{arg})
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
		{Type: order.Limit}:                    {String: "LIMIT_ORDER"},
		{TimeInForce: order.PostOnly}:          {String: "POST_ONLY"},
		{Type: order.Market}:                   {String: "MARKET_ORDER"},
		{TimeInForce: order.ImmediateOrCancel}: {String: "IMMEDIATE_OR_CANCEL"},
		{TimeInForce: order.FillOrKill}:        {String: "FILL_OR_KILL"},
		{Type: order.StopLimit}:                {String: "STOP_LIMIT"},
		{Type: order.OptimalLimitIOC}:          {String: "", Error: order.ErrUnsupportedOrderType},
	}
	for a := range typesMap {
		value, err := me.OrderTypeStringFromOrderTypeAndTimeInForce(a.Type, a.TimeInForce)
		assert.Equal(t, typesMap[a].String, value)
		assert.ErrorIs(t, err, typesMap[a].Error)
	}
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := me.CancelTradeOrder(context.Background(), "", "", "", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = me.CancelTradeOrder(context.Background(), "BTCUSDT", "", "", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CancelTradeOrder(context.Background(), "BTCUSDT", "1234", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := me.CancelAllOpenOrdersBySymbol(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CancelAllOpenOrdersBySymbol(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	_, err := me.GetOrderByID(context.Background(), "", "123455", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = me.GetOrderByID(context.Background(), "BTCUSDT", "", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetOrderByID(context.Background(), "BTCUSDT", "1234", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := me.GetOpenOrders(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetOpenOrders(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOrders(t *testing.T) {
	t.Parallel()
	_, err := me.GetAllOrders(context.Background(), "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAllOrders(context.Background(), "BTCUSDT", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAccountInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := me.GetAccountTradeList(context.Background(), "", "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAccountTradeList(context.Background(), "BTCUSDT", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableMXDeduct(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.EnableMXDeduct(context.Background(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMXDeductStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetMXDeductStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolTradingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSymbolTradingFee(context.Background(), "BTCUSDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetCurrencyInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCapital(t *testing.T) {
	t.Parallel()
	_, err := me.WithdrawCapital(context.Background(), 1.2, currency.EMPTYCODE, "", "BNB", "1234", core.BitcoinDonationAddress, "abcd", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = me.WithdrawCapital(context.Background(), 1.2, currency.BTC, "", "BNB", "", "", "abcd", "")
	require.ErrorIs(t, err, errAddressRequired)
	_, err = me.WithdrawCapital(context.Background(), 0, currency.BTC, "", "BNB", "1234", core.BitcoinDonationAddress, "abcd", "")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.WithdrawCapital(context.Background(), 1.2, currency.BTC, "", "BNB", "1234", core.BitcoinDonationAddress, "abcd", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := me.CancelWithdrawal(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CancelWithdrawal(context.Background(), "1231212")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetFundDepositHistory(context.Background(), currency.BTC, "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetWithdrawalHistory(context.Background(), currency.USDT, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := me.GenerateDepositAddress(context.Background(), currency.EMPTYCODE, "TRC20")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = me.GenerateDepositAddress(context.Background(), currency.USDT, "")
	require.ErrorIs(t, err, errNetworkNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.GenerateDepositAddress(context.Background(), currency.USDT, "TRC20")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressOfCoin(t *testing.T) {
	t.Parallel()
	_, err := me.GetDepositAddressOfCoin(context.Background(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetDepositAddressOfCoin(context.Background(), currency.USDT, "ERC20")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetWithdrawalAddress(context.Background(), currency.USDT, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUserUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := me.UserUniversalTransfer(context.Background(), "", "SPOT", currency.USDT, 1000)
	require.ErrorIs(t, err, errAccountTypeRequired)
	_, err = me.UserUniversalTransfer(context.Background(), "FUTURE", "", currency.USDT, 1000)
	require.ErrorIs(t, err, errAccountTypeRequired)
	_, err = me.UserUniversalTransfer(context.Background(), "FUTURE", "SPOT", currency.EMPTYCODE, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = me.UserUniversalTransfer(context.Background(), "FUTURE", "SPOT", currency.USDT, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.UserUniversalTransfer(context.Background(), "FUTURE", "SPOT", currency.USDT, 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := me.GetUniversalTransferHistory(context.Background(), "", "FUTURE", time.Now().Add(-time.Hour*20), time.Now(), 0, 10)
	require.ErrorIs(t, err, errAccountTypeRequired)
	_, err = me.GetUniversalTransferHistory(context.Background(), "SPOT", "", time.Now().Add(-time.Hour*20), time.Now(), 0, 10)
	require.ErrorIs(t, err, errAccountTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetUniversalTransferHistory(context.Background(), "SPOT", "FUTURE", time.Now().Add(-time.Hour*20), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUniversalTransferDetailByID(t *testing.T) {
	t.Parallel()
	_, err := me.GetUniversalTransferDetailByID(context.Background(), "")
	require.ErrorIs(t, err, errTransactionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetUniversalTransferDetailByID(context.Background(), "12345678")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetThatCanBeConvertedintoMX(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAssetThatCanBeConvertedintoMX(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDustTransfer(t *testing.T) {
	t.Parallel()
	_, err := me.DustTransfer(context.Background(), []currency.Code{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = me.DustTransfer(context.Background(), []currency.Code{currency.EMPTYCODE, currency.ETH})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.DustTransfer(context.Background(), []currency.Code{currency.BTC, currency.ETH})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDustLog(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.DustLog(context.Background(), time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInternalTransfer(t *testing.T) {
	t.Parallel()
	_, err := me.InternalTransfer(context.Background(), "", "someone@example.com", "+251", currency.USDT, 1.2)
	require.ErrorIs(t, err, errAccountTypeRequired)
	_, err = me.InternalTransfer(context.Background(), "EMAIL", "", "+251", currency.USDT, 1.2)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = me.InternalTransfer(context.Background(), "EMAIL", "someone@example.com", "+251", currency.EMPTYCODE, 1.2)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = me.InternalTransfer(context.Background(), "EMAIL", "someone@example.com", "+251", currency.USDT, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.InternalTransfer(context.Background(), "EMAIL", "someone@example.com", "+251", currency.USDT, 1.2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInternalTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetInternalTransferHistory(context.Background(), "11945860693", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCapitalWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := me.CapitalWithdrawal(context.Background(), currency.EMPTYCODE, "1234", "TRC20", core.BitcoinDonationAddress, "", "", 1234)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = me.CapitalWithdrawal(context.Background(), currency.BTC, "12345678", "TRC20", "", "", "", 1234)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = me.CapitalWithdrawal(context.Background(), currency.BTC, "1234", "TRC20", core.BitcoinDonationAddress, "", "", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CapitalWithdrawal(context.Background(), currency.BTC, "1234", "TRC20", core.BitcoinDonationAddress, "", "", 1234)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRebateHistoryRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetRebateHistoryRecords(context.Background(), time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRebateRecordsDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetRebateRecordsDetail(context.Background(), time.Now().Add(-time.Hour*48), time.Now(), 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSelfRebateRecordsDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSelfRebateRecordsDetail(context.Background(), time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetReferCode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetReferCode(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCommissionRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAffiliateCommissionRecord(context.Background(), time.Time{}, time.Time{}, "abcdef", 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateWithdrawRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAffiliateWithdrawRecord(context.Background(), time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCommissionDetailRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAffiliateCommissionDetailRecord(context.Background(), time.Time{}, time.Time{}, "", "1", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateCampaignData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAffiliateCampaignData(context.Background(), time.Now().Add(-time.Hour*480), time.Now(), 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffiliateReferralData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAffiliateReferralData(context.Background(), time.Time{}, time.Time{}, "", "", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAffiliateData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSubAffiliateData(context.Background(), time.Time{}, time.Time{}, "", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractsDetail(t *testing.T) {
	t.Parallel()
	result, err := me.GetFuturesContracts(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetFuturesContracts(context.Background(), result.Data[0].Symbol)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()
	detail := `{"symbol":"BTC_USDT","displayName":"BTC_USDT永续","displayNameEn":"BTC_USDT PERPETUAL","positionOpenType":3,"baseCoin":"BTC","quoteCoin":"USDT","baseCoinName":"BTC","quoteCoinName":"USDT","futureType":1,"settleCoin":"USDT","contractSize":0.0001,"minLeverage":1,"maxLeverage":400,"countryConfigContractMaxLeverage":0,"priceScale":1,"volScale":0,"amountScale":4,"priceUnit":0.1,"volUnit":1,"minVol":1,"maxVol":1500000,"bidLimitPriceRate":0.1,"askLimitPriceRate":0.1,"takerFeeRate":0.0002,"makerFeeRate":0,"maintenanceMarginRate":0.0015,"initialMarginRate":0.0025,"riskBaseVol":15000,"riskIncrVol":200000,"riskLongShortSwitch":0,"riskIncrMmr":0.005,"riskIncrImr":0.008,"riskLevelLimit":13,"priceCoefficientVariation":0.004,"indexOrigin":["BITGET","BYBIT","BINANCE","HTX","OKX","MEXC","KUCOIN"],"state":0,"isNew":false,"isHot":false,"isHidden":false,"conceptPlate":["mc-trade-zone-pow"],"conceptPlateId":[12],"riskLimitType":"BY_VOLUME","maxNumOrders":[200,50],"marketOrderMaxLevel":20,"marketOrderPriceLimitRate1":0.2,"marketOrderPriceLimitRate2":0.005,"triggerProtect":0.1,"appraisal":0,"showAppraisalCountdown":0,"automaticDelivery":0,"apiAllowed":false,"depthStepList":["0.1","1","10","100"],"limitMaxVol":10000000,"threshold":0,"baseCoinIconUrl":"https://public.mocortech.com/coin/F20210514192151938ROhGjOFp2Fpgb7.png","id":10,"vid":"128f589271cb4951b03e71e6323eb7be","baseCoinId":"febc9973be4d4d53bb374476239eb219","createTime":1591242684000,"openingTime":0,"openingCountdownOption":1,"showBeforeOpen":true,"isMaxLeverage":true,"isZeroFeeRate":false}`
	details := `[` + detail + ",{}]"
	var target FuturesContractsList
	err := json.Unmarshal([]byte(detail), &target)
	assert.NoError(t, err)
	assert.Len(t, target, 1)

	var targets FuturesContractsList
	err = json.Unmarshal([]byte(details), &targets)
	assert.NoError(t, err)
	assert.Len(t, targets, 2)
}

func TestGetTransferableCurrencies(t *testing.T) {
	t.Parallel()
	result, err := me.GetTransferableCurrencies(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractDepthInformation(t *testing.T) {
	t.Parallel()
	_, err := me.GetContractDepthInformation(context.Background(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetContractDepthInformation(context.Background(), "BTC_USDT", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepthSnapshotOfContract(t *testing.T) {
	t.Parallel()
	_, err := me.GetDepthSnapshotOfContract(context.Background(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = me.GetDepthSnapshotOfContract(context.Background(), "BTC_USDT", 0)
	require.ErrorIs(t, err, errLimitIsRequired)

	result, err := me.GetDepthSnapshotOfContract(context.Background(), "BTC_USDT", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := me.GetContractIndexPrice(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetContractIndexPrice(context.Background(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractFairPrice(t *testing.T) {
	t.Parallel()
	_, err := me.GetContractFairPrice(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetContractFairPrice(context.Background(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractFundingPrice(t *testing.T) {
	t.Parallel()
	_, err := me.GetContractFundingPrice(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetContractFundingPrice(context.Background(), "BTC_USDT")
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
	_, err := me.GetContractsCandlestickData(context.Background(), "", 0, time.Now().Add(-time.Hour*480), time.Now())
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetContractsCandlestickData(context.Background(), "BTC_USDT", kline.FifteenMin, time.Now().Add(-time.Hour*480), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKlineDataOfIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := me.GetKlineDataOfIndexPrice(context.Background(), "", 0, time.Now().Add(-time.Hour*480), time.Now())
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetKlineDataOfIndexPrice(context.Background(), "BTC_USDT", kline.FifteenMin, time.Now().Add(-time.Hour*480), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKlineDataOfFairPrice(t *testing.T) {
	t.Parallel()
	_, err := me.GetKlineDataOfFairPrice(context.Background(), "", 0, time.Now().Add(-time.Hour*480), time.Now())
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetKlineDataOfFairPrice(context.Background(), "BTC_USDT", kline.FifteenMin, time.Now().Add(-time.Hour*480), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractTransactionData(t *testing.T) {
	t.Parallel()
	_, err := me.GetContractTransactionData(context.Background(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetContractTransactionData(context.Background(), "BTC_USDT", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractTrendData(t *testing.T) {
	t.Parallel()
	result, err := me.GetContractTickers(context.Background(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	result, err = me.GetContractTickers(context.Background(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetAllContractRiskFundBalance(t *testing.T) {
	t.Parallel()
	result, err := me.GetAllContractRiskFundBalance(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractRiskFundBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := me.GetContractRiskFundBalanceHistory(context.Background(), "", 1, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = me.GetContractRiskFundBalanceHistory(context.Background(), "BTC_USDT", 0, 10)
	require.ErrorIs(t, err, errPageNumberRequired)
	_, err = me.GetContractRiskFundBalanceHistory(context.Background(), "BTC_USDT", 1, 0)
	require.ErrorIs(t, err, errPageSizeRequired)

	result, err := me.GetContractRiskFundBalanceHistory(context.Background(), "BTC_USDT", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := me.GetContractFundingRateHistory(context.Background(), "", 1, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := me.GetContractFundingRateHistory(context.Background(), "BTC_USDT", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUserAssetsInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAllUserAssetsInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserSingleCurrencyAssetInformation(t *testing.T) {
	t.Parallel()
	_, err := me.GetUserSingleCurrencyAssetInformation(context.Background(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetUserSingleCurrencyAssetInformation(context.Background(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAssetTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetUserAssetTransferRecords(context.Background(), currency.ETH, "WAIT", "IN", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserPositionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetUserPositionHistory(context.Background(), "BTC_USDT", "1", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersCurrentHoldingPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetUsersCurrentHoldingPositions(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetUsersCurrentHoldingPositions(context.Background(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersFundingRateDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetUsersFundingRateDetails(context.Background(), "BTC_USDT", 123123, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserCurrentPendingOrder(t *testing.T) {
	t.Parallel()
	_, err := me.GetUserCurrentPendingOrder(context.Background(), "", 0, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetUserCurrentPendingOrder(context.Background(), "BTC_USDT", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllUserHistoricalOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAllUserHistoricalOrders(context.Background(), "BTC_USDT", "1", "1", "1", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderBasedOnExternalNumber(t *testing.T) {
	t.Parallel()
	_, err := me.GetOrderBasedOnExternalNumber(context.Background(), "", "12312312")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = me.GetOrderBasedOnExternalNumber(context.Background(), "BTC_USDT", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetOrderBasedOnExternalNumber(context.Background(), "BTC_USDT", "12312312")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderByOrderNumber(t *testing.T) {
	t.Parallel()
	_, err := me.GetOrderByOrderID(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetOrderByOrderID(context.Background(), "12312312")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBatchOrdersByOrderID(t *testing.T) {
	t.Parallel()
	_, err := me.GetBatchOrdersByOrderID(context.Background(), nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetBatchOrdersByOrderID(context.Background(), []string{"123123"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderTransactionDetailsByOrderID(t *testing.T) {
	t.Parallel()
	_, err := me.GetOrderTransactionDetailsByOrderID(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetOrderTransactionDetailsByOrderID(context.Background(), "1232131")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserOrderAllTransactionDetails(t *testing.T) {
	t.Parallel()
	_, err := me.GetUserOrderAllTransactionDetails(context.Background(), "", time.Time{}, time.Time{}, 1, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetUserOrderAllTransactionDetails(context.Background(), "BTC_USDT", time.Time{}, time.Time{}, 1, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTriggerOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetTriggerOrderList(context.Background(), "", "1", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesStopLimitOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetFuturesStopLimitOrderList(context.Background(), "BTC_USDT", false, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetFuturesRiskLimit(context.Background(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesCurrentTradingFeeRate(t *testing.T) {
	t.Parallel()
	_, err := me.GetFuturesCurrentTradingFeeRate(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetFuturesCurrentTradingFeeRate(context.Background(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIncreaseDecreaseMargin(t *testing.T) {
	t.Parallel()
	err := me.IncreaseDecreaseMargin(context.Background(), 0, 0, "ADD")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	err = me.IncreaseDecreaseMargin(context.Background(), 12312312, 0, "ADD")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	err = me.IncreaseDecreaseMargin(context.Background(), 12312312, 1.5, "")
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	err = me.IncreaseDecreaseMargin(context.Background(), 1231231, 123.45, "SUB")
	assert.NoError(t, err)
}

func TestGetContractLeverage(t *testing.T) {
	t.Parallel()
	_, err := me.GetContractLeverage(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetContractLeverage(context.Background(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSwitchLeverage(t *testing.T) {
	t.Parallel()
	_, err := me.SwitchLeverage(context.Background(), 0, 25, 2, 1, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = me.SwitchLeverage(context.Background(), 123333, 0, 2, 1, "")
	require.ErrorIs(t, err, errMissingLeverage)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.SwitchLeverage(context.Background(), 123333, 25, 2, 1, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetPositionMode(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestChangePositionMode(t *testing.T) {
	t.Parallel()
	_, err := me.ChangePositionMode(context.Background(), 0)
	require.ErrorIs(t, err, errPositionModeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)

	result, err := me.ChangePositionMode(context.Background(), 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := me.PlaceFuturesOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &PlaceFuturesOrderParams{
		ReduceOnly: true,
	}
	_, err = me.PlaceFuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	arg.Symbol = "BTC_USDT"
	_, err = me.PlaceFuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	arg.Price = 1234
	_, err = me.PlaceFuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	arg.Volume = 3
	_, err = me.PlaceFuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell
	arg.OrderType = order.Limit.String()
	_, err = me.PlaceFuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.PlaceFuturesOrder(context.Background(), &PlaceFuturesOrderParams{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrdersByID(t *testing.T) {
	t.Parallel()
	_, err := me.CancelOrdersByID(context.Background())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = me.CancelOrdersByID(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CancelOrdersByID(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := me.CancelOrderByClientOrderID(context.Background(), "", "12345")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = me.CancelOrderByClientOrderID(context.Background(), "BTC_USDT", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CancelOrderByClientOrderID(context.Background(), "BTC_USDT", "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.CancelAllOpenOrders(context.Background(), "BTC_USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerUniversalTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := me.GetBrokerUniversalTransferHistory(context.Background(), "", "", "test1@thrasher.io", "test2@thrasher.io", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = me.GetBrokerUniversalTransferHistory(context.Background(), "FUTURES", "", "test1@thrasher.io", "test2@thrasher.io", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetBrokerUniversalTransferHistory(context.Background(), "SPOT", "FUTURES", "test1@thrasher.io", "test2@thrasher.io", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBrokerSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.CreateBrokerSubAccount(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerAccountSubAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetBrokerAccountSubAccountList(context.Background(), "my-subaccount-name", 10, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountStatus(t *testing.T) {
	t.Parallel()
	_, err := me.GetSubAccountStatus(context.Background(), "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSubAccountStatus(context.Background(), "my-subaccount-name")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateBrokerSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := me.CreateBrokerSubAccountAPIKey(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &BrokerSubAccountAPIKeyParams{
		IP: []string{"127.0.0.1"},
	}
	_, err = me.CreateBrokerSubAccountAPIKey(context.Background(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	arg.SubAccount = "my-subaccount-name"
	_, err = me.CreateBrokerSubAccountAPIKey(context.Background(), arg)
	require.ErrorIs(t, err, errUnsupportedPermissionValue)

	arg.Permissions = []string{"SPOT_ACCOUNT_READ", "SPOT_ACCOUNT_WRITE"}
	_, err = me.CreateBrokerSubAccountAPIKey(context.Background(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountNote)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateAPIEndpoints)
	result, err := me.CreateBrokerSubAccountAPIKey(context.Background(), &BrokerSubAccountAPIKeyParams{SubAccount: "my-subaccount-name", Permissions: []string{"SPOT_ACCOUNT_READ", "SPOT_ACCOUNT_WRITE"}, Note: "note-here"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetBrokerSubAccountAPIKey(context.Background(), "my-subaccount-name")
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
	_, err := me.DeleteBrokerAPIKeySubAccount(context.Background(), &BrokerSubAccountAPIKeyDeletionParams{APIKey: "api-key-here"})
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = me.DeleteBrokerAPIKeySubAccount(context.Background(), &BrokerSubAccountAPIKeyDeletionParams{SubAccount: "sub-account-detail-here"})
	require.ErrorIs(t, err, errAPIKeyMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateAPIEndpoints)
	result, err := me.DeleteBrokerAPIKeySubAccount(context.Background(), &BrokerSubAccountAPIKeyDeletionParams{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateBrokerSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := me.GenerateBrokerSubAccountDepositAddress(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = me.GenerateBrokerSubAccountDepositAddress(context.Background(), &BrokerSubAccountDepositAddressCreationParams{Network: "ERC20"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = me.GenerateBrokerSubAccountDepositAddress(context.Background(), &BrokerSubAccountDepositAddressCreationParams{Coin: currency.ETH})
	require.ErrorIs(t, err, errNetworkNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
	result, err := me.GenerateBrokerSubAccountDepositAddress(context.Background(), &BrokerSubAccountDepositAddressCreationParams{Coin: currency.ETH, Network: "ERC20"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBrokerSubAccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := me.GetBrokerSubAccountDepositAddress(context.Background(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetBrokerSubAccountDepositAddress(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetSubAccountDepositHistory(context.Background(), currency.ETH, "1", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllRecentSubAccountDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAllRecentSubAccountDepositHistory(context.Background(), currency.ETH, "1", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	for k, v := range assetsAndErrors {
		result, err := me.FetchTradablePairs(context.Background(), k)
		require.ErrorIs(t, err, v)
		if v == nil {
			assert.NotNil(t, result)
		}
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := me.UpdateTicker(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT}, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = me.UpdateTicker(context.Background(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := me.UpdateTicker(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT}, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.UpdateTicker(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT}, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	for k, v := range assetsAndErrors {
		err := me.UpdateTickers(context.Background(), k)
		assert.ErrorIs(t, err, v)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := me.UpdateOrderbook(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT}, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = me.UpdateOrderbook(context.Background(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := me.UpdateOrderbook(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT}, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.UpdateOrderbook(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT}, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := me.GetHistoricCandles(context.Background(), currency.EMPTYPAIR, asset.Spot, kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = me.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.TenMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := me.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.FiveMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetHistoricCandles(context.Background(), futuresTradablePair, asset.Futures, kline.FiveMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*5))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := me.GetHistoricCandlesExtended(context.Background(), currency.EMPTYPAIR, asset.Spot, kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = me.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.TenMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*2))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := me.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.FiveMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetHistoricCandlesExtended(context.Background(), futuresTradablePair, asset.Futures, kline.FiveMin, time.Now().Add(-time.Hour*48), time.Now().Add(-time.Hour*5))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	sTime, err := me.GetServerTime(context.Background(), asset.Empty)
	require.NoError(t, err)
	assert.NotEmpty(t, sTime)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := me.UpdateOrderExecutionLimits(context.Background(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = me.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	require.NoErrorf(t, err, "Error fetching %s pairs for test: %v", asset.Spot, err)

	instrumentInfo, err := me.GetSymbols(context.Background(), []string{spotTradablePair.String()})
	require.NoError(t, err)
	require.NotEmpty(t, instrumentInfo.Symbols[0])

	limits, err := me.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
	require.NoError(t, err)

	symbolDetail := instrumentInfo.Symbols[0]
	require.NotNil(t, symbolDetail, "instrument required to be found")
	require.Equal(t, symbolDetail.QuoteAmountPrecision.Float64(), limits.PriceStepIncrementSize)
	assert.Equal(t, symbolDetail.BaseSizePrecision.Float64(), limits.MinimumBaseAmount)
	assert.Equal(t, symbolDetail.MaxQuoteAmount.Float64(), limits.MaximumQuoteAmount)

	err = me.UpdateOrderExecutionLimits(context.Background(), asset.Futures)
	require.NoErrorf(t, err, "Error fetching %s pairs for test: %v", asset.Spot, err)

	fInstrumentDetail, err := me.GetFuturesContracts(context.Background(), futuresTradablePair.String())
	require.NoError(t, err)
	require.NotEmpty(t, fInstrumentDetail.Data[0])

	limits, err = me.GetOrderExecutionLimits(asset.Futures, futuresTradablePair)
	require.NoError(t, err)

	fsymbolDetail := fInstrumentDetail.Data[0]
	require.NotNil(t, fsymbolDetail)
	assert.Equal(t, fsymbolDetail.PriceScale, limits.PriceStepIncrementSize)
	assert.Equal(t, fsymbolDetail.MinVol, limits.MinimumBaseAmount)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := me.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.Options,
		Pair:                 currency.NewPair(currency.BTC, currency.USDT),
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := me.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  futuresTradablePair,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	_, err := me.IsPerpetualFutureCurrency(asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = me.IsPerpetualFutureCurrency(asset.Spot, spotTradablePair)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = me.IsPerpetualFutureCurrency(asset.Futures, futuresTradablePair)
	assert.NoError(t, err)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := me.GetFuturesContractDetails(context.Background(), asset.Binary)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = me.GetFuturesContractDetails(context.Background(), asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := me.GetFuturesContractDetails(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.UpdateAccountInfo(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.UpdateAccountInfo(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Empty)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetAccountFundingHistory(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := me.GetRecentTrades(context.Background(), currency.EMPTYPAIR, asset.Options)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = me.GetRecentTrades(context.Background(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := me.GetRecentTrades(context.Background(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetRecentTrades(context.Background(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := me.GetHistoricTrades(context.Background(), currency.EMPTYPAIR, asset.Options, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = me.GetHistoricTrades(context.Background(), spotTradablePair, asset.Options, time.Time{}, time.Time{})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := me.GetHistoricTrades(context.Background(), spotTradablePair, asset.Spot, time.Now().Add(-time.Minute*4), time.Now().Add(-time.Minute*2))
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetHistoricTrades(context.Background(), futuresTradablePair, asset.Futures, time.Now().Add(-time.Minute*4), time.Now().Add(-time.Minute*2))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := me.GetDepositAddress(context.Background(), currency.EMPTYCODE, "", "TON")
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	_, err = me.GetDepositAddress(context.Background(), currency.BTC, "", "TON")
	require.True(t, err != nil || err == deposit.ErrAddressNotFound)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	arg := &order.MultiOrderRequest{AssetType: asset.Options}
	_, err := me.GetActiveOrders(context.Background(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	arg.Pairs = currency.Pairs{}
	_, err = me.GetActiveOrders(context.Background(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	arg.Pairs = currency.Pairs{spotTradablePair}
	result, err := me.GetActiveOrders(context.Background(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)

	arg.AssetType = asset.Futures
	arg.Pairs = currency.Pairs{futuresTradablePair}
	result, err = me.GetActiveOrders(context.Background(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateListenKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	listenKey, err := me.GenerateListenKey(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, listenKey)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := me.GetOrderInfo(context.Background(), "12342", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
	result, err := me.GetOrderInfo(context.Background(), "12342", spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = me.GetOrderInfo(context.Background(), "12342", futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
