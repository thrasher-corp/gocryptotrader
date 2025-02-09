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
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = "WSKMLKNW-JCKF6SGH-VWKQAUS8-RYYWFJYP"
	apiSecret               = "b1b11137b33e52bd7ae2df3a59e905141b40740edd0568f9141b63ed0cea6bdcab8b2ac8e307ca11a048493fd7d1528a26d7a1a9e3caae53fb82965b3ebf2b57"
	canManipulateRealOrders = false
)

var me = &MEXC{}

func TestMain(m *testing.M) {
	me.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Mexc")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = me.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
// func TestInterface(t *testing.T) {
// 	var e exchange.IBotExchange
// 	if e = new(MEXC); e == nil {
// 		t.Fatal("unable to allocate exchange")
// 	}
// }

// Implement tests for API endpoints below

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

func TestGetCandlestick(t *testing.T) {
	t.Parallel()
	_, err := me.GetCandlestick(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = me.GetCandlestick(context.Background(), "BTCUSDT", kline.TwelveHour, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := me.GetCandlestick(context.Background(), "BTCUSDT", kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := me.GetCurrentAveragePrice(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	me.Verbose = true
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
	me.Verbose = true
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
	result, err := me.GetSubAccountList(context.Background(), "SubAcc1", false, 1, 10)
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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me, canManipulateRealOrders)
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

	arg.OrderType, err = me.OrderTypeString(order.Limit)
	require.NoError(t, err)
	_, err = me.CreateBatchOrder(context.Background(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Quantity = 123478.5
	_, err = me.CreateBatchOrder(context.Background(), []BatchOrderCreationParam{arg})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.OrderType, err = me.OrderTypeString(order.Market)
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
	typesMap := map[order.Type]struct {
		String string
		Error  error
	}{
		order.Limit:             {String: "LIMIT_ORDER"},
		order.PostOnly:          {String: "POST_ONLY"},
		order.Market:            {String: "MARKET_ORDER"},
		order.ImmediateOrCancel: {String: "IMMEDIATE_OR_CANCEL"},
		order.FillOrKill:        {String: "FILL_OR_KILL"},
		order.StopLimit:         {String: "STOP_LIMIT"},
		order.OptimalLimitIOC:   {String: "", Error: order.ErrUnsupportedOrderType},
	}
	for a := range typesMap {
		value, err := me.OrderTypeString(a)
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
	result, err := me.GetWithdrawalHistory(context.Background(), currency.USDT, "APPLY", time.Now().Add(-10*time.Hour), time.Now(), 10)
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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, me)
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
