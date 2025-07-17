package apexpro

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey    = ""
	apiSecret = ""
	clientID  = ""

	starkKey            = ""
	starkSecret         = ""
	starkKeyYCoordinate = ""

	ethereumAddress = ""

	canManipulateRealOrders = true
)

var e *Exchange

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" && clientID != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.API.CredentialsValidator.RequiresBase64DecodeSecret = false
		e.SetCredentials(apiKey, apiSecret, clientID, ethereumAddress, "", "", starkKey, starkSecret, starkKeyYCoordinate)
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	if err := e.UpdateTradablePairs(context.Background(), true); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestGetSystemTimeV3(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemTimeV3(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTimeV2(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemTimeV2(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTimeV1(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemTimeV1(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllConfigDataV3(t *testing.T) {
	t.Parallel()
	result, err := e.GetAllConfigDataV3(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllSymbolsConfigDataV1(t *testing.T) {
	t.Parallel()
	result, err := e.GetAllSymbolsConfigDataV1(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketDepthV3(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarketDepthV3(t.Context(), "BTC-USDC", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketDepthV2(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarketDepthV2(t.Context(), "BTC-USDC", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketDepthV1(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarketDepthV1(t.Context(), "BTC-USDC", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNewestTradingDataV3(t *testing.T) {
	t.Parallel()
	result, err := e.GetNewestTradingDataV3(t.Context(), "BTC-USDC", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNewestTradingDataV2(t *testing.T) {
	t.Parallel()
	result, err := e.GetNewestTradingDataV2(t.Context(), "BTC-USDC", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNewestTradingDataV1(t *testing.T) {
	t.Parallel()
	result, err := e.GetNewestTradingDataV1(t.Context(), "BTC-USDC", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlestickChartDataV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlestickChartDataV3(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetCandlestickChartDataV3(t.Context(), "BTC-USDC", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlestickChartDataV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlestickChartDataV2(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetCandlestickChartDataV2(t.Context(), "BTC-USDC", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlestickChartDataV1(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlestickChartDataV1(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetCandlestickChartDataV1(t.Context(), "BTC-USDC", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickerDataV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickerDataV3(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetTickerDataV3(t.Context(), "BTC-USDC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickerDataV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickerDataV2(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetTickerDataV2(t.Context(), "BTC-USDC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingHistoryRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingHistoryRateV3(t.Context(), "", time.Time{}, time.Time{}, 10, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFundingHistoryRateV3(t.Context(), "BTC-USDC", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetFundingHistoryRateV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingHistoryRateV2(t.Context(), "", time.Time{}, time.Time{}, 10, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFundingHistoryRateV2(t.Context(), "BTC-USDC", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetFundingHistoryRateV1(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingHistoryRateV1(t.Context(), "", time.Time{}, time.Time{}, 10, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFundingHistoryRateV1(t.Context(), "BTC-USDC", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetAllConfigDataV2(t *testing.T) {
	t.Parallel()
	result, err := e.GetAllConfigDataV2(t.Context())
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetCheckIfUserExistsV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetCheckIfUserExistsV2(t.Context(), "")
	require.ErrorIs(t, err, errEthereumAddressMissing)

	result, err := e.GetCheckIfUserExistsV2(t.Context(), "0x0330eBB5e894720e6746070371F9Fd797BE9D074")
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetCheckIfUserExistsV1(t *testing.T) {
	t.Parallel()
	_, err := e.GetCheckIfUserExistsV1(t.Context(), "")
	require.ErrorIs(t, err, errEthereumAddressMissing)

	result, err := e.GetCheckIfUserExistsV1(t.Context(), "0x0330eBB5e894720e6746070371F9Fd797BE9D074")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateNonce(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateNonceV3(t.Context(), "", "0x0330eBB5e894720e6746070371F9Fd797BE9D074", "9")
	require.ErrorIs(t, err, errL2KeyMissing)
	_, err = e.GenerateNonceV3(t.Context(), "0x06c98993ca62f5e71dbe721f743045eff7475711b359681cd64364a60e677505", "", "9")
	require.ErrorIs(t, err, errEthereumAddressMissing)
	_, err = e.GenerateNonceV3(t.Context(), "0x06c98993ca62f5e71dbe721f743045eff7475711b359681cd64364a60e677505", "0x0330eBB5e894720e6746070371F9Fd797BE9D074", "")
	require.ErrorIs(t, err, errChainIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GenerateNonceV3(t.Context(), starkKey, ethereumAddress, "9")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateNonceV2(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateNonceV2(t.Context(), "", "0x0330eBB5e894720e6746070371F9Fd797BE9D074", "9")
	require.ErrorIs(t, err, errL2KeyMissing)
	_, err = e.GenerateNonceV2(t.Context(), "0x06c98993ca62f5e71dbe721f743045eff7475711b359681cd64364a60e677505", "", "9")
	require.ErrorIs(t, err, errEthereumAddressMissing)
	_, err = e.GenerateNonceV2(t.Context(), "0x06c98993ca62f5e71dbe721f743045eff7475711b359681cd64364a60e677505", "0x0330eBB5e894720e6746070371F9Fd797BE9D074", "")
	require.ErrorIs(t, err, errChainIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GenerateNonceV2(t.Context(), starkKey, ethereumAddress, "9")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateNonceV1(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateNonceV1(t.Context(), "", "0x0330eBB5e894720e6746070371F9Fd797BE9D074", "9")
	require.ErrorIs(t, err, errL2KeyMissing)
	_, err = e.GenerateNonceV1(t.Context(), "0x06c98993ca62f5e71dbe721f743045eff7475711b359681cd64364a60e677505", "", "9")
	require.ErrorIs(t, err, errEthereumAddressMissing)
	_, err = e.GenerateNonceV1(t.Context(), "0x06c98993ca62f5e71dbe721f743045eff7475711b359681cd64364a60e677505", "0x0330eBB5e894720e6746070371F9Fd797BE9D074", "")
	require.ErrorIs(t, err, errChainIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GenerateNonceV1(t.Context(), starkKey, ethereumAddress, "9")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersDataV3(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersDataV2GetUsersDataV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersDataV2(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUsersDataV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUsersDataV1(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEditUserData(t *testing.T) {
	t.Parallel()
	_, err := e.EditUserDataV3(t.Context(), &EditUserDataParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EditUserDataV3(t.Context(), &EditUserDataParams{
		Email:                    "someone@thrasher.io",
		UserData:                 "",
		Username:                 "Thrasher",
		IsSharingUsername:        true,
		Country:                  "Ethiopia",
		EmailNotifyGeneralEnable: true,
		EmailNotifyTradingEnable: true,
		EmailNotifyAccountEnable: true,
		PopupNotifyTradingEnable: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestEditUserDataV2(t *testing.T) {
	t.Parallel()
	_, err := e.EditUserDataV2(t.Context(), &EditUserDataParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EditUserDataV2(t.Context(), &EditUserDataParams{
		Email:                    "samuaeladnew@gmail.com",
		UserData:                 "",
		Username:                 "Username",
		IsSharingUsername:        true,
		Country:                  "Ethiopia",
		EmailNotifyGeneralEnable: true,
		EmailNotifyTradingEnable: true,
		EmailNotifyAccountEnable: true,
		PopupNotifyTradingEnable: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAccountData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAccountDataV3(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAccountDataV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAccountDataV2(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAccountDataV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAccountDataV1(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAccountBalance(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAccountBalanceV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAccountBalanceV2(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAccountBalanceV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAccountBalanceV1(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTransferDataV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTransferDataV2(t.Context(), currency.USDT, time.Now().Add(-time.Hour*50), time.Now(), "DEPOSIT", nil, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserTransferDataV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTransferDataV1(t.Context(), currency.USDT, time.Time{}, time.Time{}, "", nil, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserWithdrawalListV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserWithdrawalListV2(t.Context(), "WITHDRAWAL", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserWithdrawalListV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserWithdrawalListV1(t.Context(), "WITHDRAWAL", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFastAndCrossChainWithdrawalFees(t *testing.T) {
	t.Parallel()
	_, err := e.GetFastAndCrossChainWithdrawalFeesV2(t.Context(), 1, "1", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFastAndCrossChainWithdrawalFeesV2(t.Context(), 1.32, "1", currency.USDC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFastAndCrossChainWithdrawalFeesV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFastAndCrossChainWithdrawalFeesV1(t.Context(), 1.32, "1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetWithdrawalAndTransferLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetAssetWithdrawalAndTransferLimitV2(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAssetWithdrawalAndTransferLimitV2(t.Context(), currency.USDC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetWithdrawalAndTransferLimitV1(t *testing.T) {
	t.Parallel()
	_, err := e.GetAssetWithdrawalAndTransferLimitV1(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAssetWithdrawalAndTransferLimitV1(t.Context(), currency.USDC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserDepositWithdrawData(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTransferData(t.Context(), 0, 10, "", "DEPOSIT", "", "", time.Time{}, time.Now(), []string{"1"})
	require.ErrorIs(t, err, errInvalidTimestamp)
	_, err = e.GetUserTransferData(t.Context(), 0, 10, "", "DEPOSIT", "", "", time.Now().Add(time.Hour*30), time.Time{}, []string{"1"})
	require.ErrorIs(t, err, errInvalidTimestamp)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserTransferData(t.Context(), 0, 10, "", "DEPOSIT", "", "", time.Now().Add(time.Hour*30), time.Now(), []string{"1"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalFees(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalFees(t.Context(), 12, []string{"1"}, 140)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractAccountTransferLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractAccountTransferLimits(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetContractAccountTransferLimits(t.Context(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTradeHistory(t.Context(), "BTC-USD", order.Sell.String(), "LIMIT", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeHistoryV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeHistoryV2(t.Context(), "BTC-USD", order.Sell.String(), "LIMIT", currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTradeHistoryV2(t.Context(), "BTC-USD", order.Sell.String(), "LIMIT", currency.USDC, time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeHistoryV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTradeHistoryV1(t.Context(), "BTC-USD", order.Sell.String(), "LIMIT", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWorstPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetWorstPriceV3(t.Context(), "", "SELL", 1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetWorstPriceV3(t.Context(), "BTC-USDC", "", 1)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.GetWorstPriceV3(t.Context(), "BTC-USDC", "SELL", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
}

func TestGetWorstPriceV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetWorstPriceV3(t.Context(), "", "SELL", 1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetWorstPriceV3(t.Context(), "BTC-USDC", "", 1)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.GetWorstPriceV3(t.Context(), "BTC-USDC", "SELL", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWorstPriceV3(t.Context(), "BTC-USDC", "SELL", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWorstPriceV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetWorstPriceV2(t.Context(), "", "SELL", 1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetWorstPriceV2(t.Context(), "BTC-USDC", "", 1)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.GetWorstPriceV2(t.Context(), "BTC-USDC", "SELL", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWorstPriceV2(t.Context(), "BTC-USDC", "SELL", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWorstPriceV1(t *testing.T) {
	t.Parallel()
	_, err := e.GetWorstPriceV2(t.Context(), "", "SELL", 1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetWorstPriceV2(t.Context(), "BTC-USDC", "", 1)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.GetWorstPriceV2(t.Context(), "BTC-USDC", "SELL", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWorstPriceV1(t.Context(), "BTC-USDC", "SELL", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	futuresTradablePair, err := currency.NewPairFromString("ETH-USDC")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if e.UserAccountDetail == nil {
		e.UserAccountDetail, err = e.GetUserAccountDataV2(t.Context())
		assert.NoError(t, err)
		assert.NotNil(t, e.UserAccountDetail)
	}
	result, err := e.CreateOrderV3(t.Context(), &CreateOrderParams{
		Symbol:          futuresTradablePair,
		Side:            order.Buy.String(),
		OrderType:       "LIMIT",
		Size:            0.01,
		Price:           2250,
		TimeInForce:     "GOOD_TIL_CANCEL",
		TriggerPrice:    0,
		TrailingPercent: 0,
		ReduceOnly:      false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelPerpOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPerpOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelPerpOrder(t.Context(), "123231")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelPerpOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPerpOrderByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelPerpOrderByClientOrderID(t.Context(), "2312312")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrdersV3(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.CancelAllOpenOrdersV3(t.Context(), []string{"BTC-USDC"})
	assert.NoError(t, err)
}

func TestCancelPerpOrderV2(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPerpOrderV2(t.Context(), "", currency.USDT)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CancelPerpOrderV2(t.Context(), "123231", currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrders(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrdersV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrdersV2(t.Context(), currency.USDC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrdersV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrdersV1(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllOrderHistory(t.Context(), "BTC-USDC", "SELL", "MARKET", "OPEN", "HISTORY", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOrderHistoryV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllOrderHistoryV2(t.Context(), currency.EMPTYCODE, "BTC-USDC", "SELL", "MARKET", "OPEN", "HISTORY", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllOrderHistoryV2(t.Context(), currency.USDT, "BTC-USDC", "SELL", "MARKET", "OPEN", "HISTORY", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOrderHistoryV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllOrderHistoryV1(t.Context(), "BTC-USDC", "SELL", "MARKET", "OPEN", "HISTORY", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderID(t.Context(), "12343")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSingleOrderV2(t *testing.T) {
	t.Parallel()
	_, err := e.getSingleOrder(t.Context(), "", "", currency.USDC)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
}

func TestGetSingleOrderByOrderIDV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleOrderByOrderIDV2(t.Context(), "", currency.USDT)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.GetSingleOrderByOrderIDV2(t.Context(), "231232341", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSingleOrderByOrderIDV2(t.Context(), "231232341", currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSingleOrderByClientOrderIDV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleOrderByClientOrderIDV2(t.Context(), "231232341", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetSingleOrderByClientOrderIDV2(t.Context(), "", currency.USDT)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSingleOrderByClientOrderIDV2(t.Context(), "231232341", currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSingleOrderByOrderIDV1(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleOrderByOrderIDV1(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSingleOrderByOrderIDV1(t.Context(), "231232341")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSingleOrderByClientOrderIDV1(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleOrderByClientOrderIDV1(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSingleOrderByClientOrderIDV1(t.Context(), "231232341")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVerificationEmailLink(t *testing.T) {
	t.Parallel()
	err := e.GetVerificationEmailLink(t.Context(), "", currency.USDC)
	require.ErrorIs(t, err, errUserIDRequired)
	err = e.GetVerificationEmailLink(t.Context(), "123123", currency.USDC)
	assert.NoError(t, err)
}

func TestLinkDevice(t *testing.T) {
	t.Parallel()
	err := e.LinkDevice(t.Context(), currency.EMPTYCODE, "1")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	err = e.LinkDevice(t.Context(), currency.USDT, "")
	require.ErrorIs(t, err, errDeviceTypeIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.LinkDevice(t.Context(), currency.USDT, "2")
	require.NoError(t, err)
}

func TestGetOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderByClientOrderID(t.Context(), "12343")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundingRateV3(t.Context(), "BTC-USDC", "LONG", "", time.Now().Add(-time.Hour*50), time.Now(), 10, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundingRateV1(t.Context(), "BTC-USDC", "LONG", "", time.Now().Add(-time.Hour*50), time.Now(), 10, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingRateV2(t.Context(), currency.EMPTYCODE, "BTC-USDC", "LONG", "", time.Now().Add(-time.Hour*50), time.Now(), 10, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundingRateV2(t.Context(), currency.USDT, "BTC-USDC", "LONG", "", time.Now().Add(-time.Hour*50), time.Now(), 10, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserHistorialProfitAndLoss(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserHistorialProfitAndLoss(t.Context(), "BTC-USDC", "LONG", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserHistorialProfitAndLossV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserHistorialProfitAndLossV1(t.Context(), "BTC-USDC", "LONG", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserHistorialProfitAndLossV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserHistorialProfitAndLossV2(t.Context(), currency.EMPTYCODE, "BTC-USDC", "LONG", time.Time{}, time.Time{}, 0, 100)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserHistorialProfitAndLossV2(t.Context(), currency.USDT, "BTC-USDC", "LONG", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetYesterdaysPNL(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetYesterdaysPNL(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetYesterdaysPNLV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetYesterdaysPNLV1(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetYesterdaysPNLV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetYesterdaysPNLV2(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetYesterdaysPNLV2(t.Context(), currency.USDC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalAssetValue(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetHistoricalAssetValue(t.Context(), time.Now().Add(-time.Hour*50), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalAssetValueV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetHistoricalAssetValueV1(t.Context(), time.Now().Add(-time.Hour*50), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalAssetValueV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalAssetValueV2(t.Context(), currency.EMPTYCODE, time.Now().Add(-time.Hour*50), time.Now())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetHistoricalAssetValueV2(t.Context(), currency.USDC, time.Now().Add(-time.Hour*50), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetInitialMarginRateInfo(t *testing.T) {
	t.Parallel()
	err := e.SetInitialMarginRateInfo(t.Context(), "", 200)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	err = e.SetInitialMarginRateInfo(t.Context(), "BTC-USDC", 0)
	require.ErrorIs(t, err, errInitialMarginRateRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.SetInitialMarginRateInfo(t.Context(), "BTC-USDC", 200)
	assert.NoError(t, err)
}

func TestSetInitialMarginRateInfoV1(t *testing.T) {
	t.Parallel()
	err := e.SetInitialMarginRateInfoV1(t.Context(), "", 200)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	err = e.SetInitialMarginRateInfoV1(t.Context(), "BTC-USDC", 0)
	require.ErrorIs(t, err, errInitialMarginRateRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.SetInitialMarginRateInfoV1(t.Context(), "BTC-USDC", 200)
	assert.NoError(t, err)
}

func TestSetInitialMarginRateInfoV2(t *testing.T) {
	t.Parallel()
	err := e.SetInitialMarginRateInfoV2(t.Context(), "", 200)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	err = e.SetInitialMarginRateInfoV2(t.Context(), "BTC-USDC", 0)
	require.ErrorIs(t, err, errInitialMarginRateRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err = e.SetInitialMarginRateInfoV2(t.Context(), "BTC-USDC", 200)
	assert.NoError(t, err)
}

func TestWithdrawAsset(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawAsset(t.Context(), &AssetWithdrawalParams{
		ClientWithdrawID: "123123",
		Timestamp:        time.Now(),
		EthereumAddress:  ethereumAddress,
		L2Key:            starkKey,
		ToChainID:        "3",
		L2SourceTokenID:  currency.USDC,
		L1TargetTokenID:  currency.USDC,
		IsFastWithdraw:   false})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	_, err = e.WithdrawAsset(t.Context(), &AssetWithdrawalParams{
		Amount:          1,
		Timestamp:       time.Now(),
		EthereumAddress: ethereumAddress,
		L2Key:           starkKey,
		ToChainID:       "3",
		L2SourceTokenID: currency.USDC,
		L1TargetTokenID: currency.USDC,
		IsFastWithdraw:  false})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	_, err = e.WithdrawAsset(t.Context(), &AssetWithdrawalParams{
		Amount:           1,
		ClientWithdrawID: "123123",
		EthereumAddress:  ethereumAddress,
		L2Key:            starkKey,
		ToChainID:        "3",
		L2SourceTokenID:  currency.USDC,
		L1TargetTokenID:  currency.USDC,
		IsFastWithdraw:   false})
	require.ErrorIs(t, err, errInvalidTimestamp)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawAsset(t.Context(), &AssetWithdrawalParams{
		Amount:           1,
		ClientWithdrawID: "123123",
		Timestamp:        time.Now(),
		EthereumAddress:  ethereumAddress,
		L2Key:            starkKey,
		ToChainID:        "3",
		L2SourceTokenID:  currency.USDC,
		L1TargetTokenID:  currency.USDC,
		IsFastWithdraw:   false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUserWithdrawalV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UserWithdrawalV2(t.Context(),
		&WithdrawalParams{
			Amount:          1,
			Asset:           currency.USDC,
			EthereumAddress: "0x0330eBB5e894720e6746070371F9Fd797BE9D074",
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawalToAddressV2(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawalToAddressV2(t.Context(), &WithdrawalToAddressParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.WithdrawalToAddressV2(t.Context(), &WithdrawalToAddressParams{Asset: currency.ETH})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = e.WithdrawalToAddressV2(t.Context(), &WithdrawalToAddressParams{
		Amount: .1,
	})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WithdrawalToAddressV2(t.Context(), &WithdrawalToAddressParams{
		Amount:        1,
		ClientOrderID: "12334",
		Asset:         currency.BTC,
	})
	require.ErrorIs(t, err, errEthereumAddressMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawalToAddressV2(t.Context(), &WithdrawalToAddressParams{
		Amount:          1,
		Asset:           currency.USDC,
		EthereumAddress: "0x0330eBB5e894720e6746070371F9Fd797BE9D074",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawalToAddressV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawalToAddressV1(t.Context(), &WithdrawalToAddressParams{
		Amount:          1,
		Asset:           currency.USDC,
		EthereumAddress: "0x0330eBB5e894720e6746070371F9Fd797BE9D074",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrderCreationParamsFilter(t *testing.T) {
	t.Parallel()
	_, err := e.orderCreationParamsFilter(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrOrderDetailIsNil)
	_, err = e.orderCreationParamsFilter(t.Context(), &CreateOrderParams{Side: order.Buy.String()})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	futuresTradablePair, err := currency.NewPairFromString("BTC-USDC")
	require.NoError(t, err)
	arg := &CreateOrderParams{Symbol: futuresTradablePair}
	_, err = e.orderCreationParamsFilter(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	arg.Side = order.Buy.String()
	_, err = e.orderCreationParamsFilter(t.Context(), &CreateOrderParams{Symbol: futuresTradablePair, Side: order.Buy.String()})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	arg.OrderType = order.Limit.String()
	_, err = e.orderCreationParamsFilter(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	arg.Size = 2
	_, err = e.orderCreationParamsFilter(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	arg.Price = 123
	arg.LimitFee = -1
	_, err = e.orderCreationParamsFilter(t.Context(), arg)
	require.ErrorIs(t, err, errLimitFeeRequired)
	arg.LimitFee = 0.003
	_, err = e.orderCreationParamsFilter(t.Context(), arg)
	require.ErrorIs(t, err, errExpirationTimeRequired)
}

func TestCreateOrderV1(t *testing.T) {
	t.Parallel()
	futuresTradablePair, err := currency.NewPairFromString("ETH-USDC")
	require.NoError(t, err)

	if e.UserAccountDetail == nil {
		e.UserAccountDetail, err = e.GetUserAccountDataV2(t.Context())
		require.NoError(t, err)
		require.NotNil(t, e.UserAccountDetail)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	e.Verbose = true
	result, err := e.CreateOrderV1(t.Context(), &CreateOrderParams{
		Symbol:          futuresTradablePair,
		Side:            order.Buy.String(),
		OrderType:       "LIMIT",
		Size:            0.01,
		Price:           2250,
		TimeInForce:     "GOOD_TIL_CANCEL",
		TriggerPrice:    0,
		TrailingPercent: 0,
		ReduceOnly:      false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateOrderV2(t *testing.T) {
	t.Parallel()
	futuresTradablePair, err := currency.NewPairFromString("ETH-USDC")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	if e.UserAccountDetail == nil {
		e.UserAccountDetail, err = e.GetUserAccountDataV2(t.Context())
		require.NoError(t, err)
		require.NotNil(t, e.UserAccountDetail)
	}
	e.Verbose = true
	result, err := e.CreateOrderV2(t.Context(), &CreateOrderParams{
		Symbol:          futuresTradablePair,
		Side:            order.Buy.String(),
		OrderType:       "LIMIT",
		Size:            0.01,
		Price:           2250,
		TimeInForce:     "GOOD_TIL_CANCEL",
		TriggerPrice:    0,
		TrailingPercent: 0,
		ReduceOnly:      false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFastWithdrawalV2(t *testing.T) {
	t.Parallel()
	e.Verbose = true
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FastWithdrawalV2(t.Context(), &FastWithdrawalParams{
		Amount:       1,
		ClientID:     "123213",
		Expiration:   time.Now().Add(time.Hour * 45).UnixMilli(),
		Asset:        currency.USDC,
		ERC20Address: "0x0330eBB5e894720e6746070371F9Fd797BE9D074",
		ChainID:      "56",
		Fees:         0,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFastWithdrawalV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FastWithdrawalV1(t.Context(), &FastWithdrawalParams{
		Amount:  1,
		Asset:   currency.USDC,
		ChainID: "1",
		Fees:    0,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	pairs, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	assert.NoErrorf(t, err, "FetchTradablePairs should not error for %s", asset.Futures)
	assert.NotEmptyf(t, pairs, "Should get some pairs for %s", asset.Futures)

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Futures)
	require.NoError(t, err)

	limits, err := e.GetOrderExecutionLimits(asset.Futures, pairs[0])
	assert.NoErrorf(t, err, "GetOrderExecutionLimits should not error for %s pair %s", asset.Futures, pairs[0])
	assert.Positivef(t, limits.MinPrice, "MinPrice must be positive for %s pair %s", asset.Futures, pairs[0])
	assert.Positivef(t, limits.MaxPrice, "MaxPrice must be positive for %s pair %s", asset.Futures, pairs[0])
	assert.Positivef(t, limits.PriceStepIncrementSize, "PriceStepIncrementSize must be positive for %s pair %s", asset.Futures, pairs[0])
	assert.Positivef(t, limits.MinimumBaseAmount, "MinimumBaseAmount must be positive for %s pair %s", asset.Futures, pairs[0])
	assert.Positivef(t, limits.MaximumBaseAmount, "MaximumBaseAmount must be positive for %s pair %s", asset.Futures, pairs[0])
	assert.Positivef(t, limits.AmountStepIncrementSize, "AmountStepIncrementSize must be positive for %s pair %s", asset.Futures, pairs[0])
	assert.Positivef(t, limits.MarketMaxQty, "MarketMaxQty must be positive for %s pair %s", asset.Futures, pairs[0])
	assert.Positivef(t, limits.MaxTotalOrders, "MaxTotalOrders must be positive for %s pair %s", asset.Futures, pairs[0])
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Futures, currency.NewPair(currency.BTC, currency.USDC))
	require.NoError(t, err)
	assert.True(t, is)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC-USDC")
	require.NoError(t, err)

	result, err := e.GetHistoricCandles(t.Context(), pair, asset.Futures, kline.OneMin, time.Now().Add(-time.Minute*3), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC-USDC")
	require.NoError(t, err)

	result, err := e.GetHistoricCandlesExtended(t.Context(), pair, asset.Futures, kline.OneMin, time.Now().Add(-time.Minute*3), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	result, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := e.UpdateTradablePairs(t.Context(), true)
	assert.NoError(t, err)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC-USDC")
	require.NoError(t, err)

	result, err := e.UpdateTicker(t.Context(), pair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	require.NoError(t, err)

	result, err := e.UpdateOrderbook(t.Context(), pair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UpdateAccountInfo(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.USDC, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTCUSD")
	require.NoError(t, err)

	result, err := e.GetRecentTrades(t.Context(), pair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetServerTime(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOrders(t.Context(), &order.Cancel{
		AssetType:  asset.Futures,
		MarginType: margin.Isolated,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "", currency.EMPTYPAIR, asset.Futures)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderInfo(t.Context(), "614463889001677573", currency.EMPTYPAIR, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Futures,
		Type:      order.Limit,
		Side:      order.Buy,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.WsConnect()
	assert.NoError(t, err)
}

func TestGetTransferErc20Fact(t *testing.T) {
	fact, err := GetTransferErc20Fact(
		3, "0x1234567890123456789012345678901234567890",
		"123.456", "0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa",
		"0x1234567890abcdef")
	assert.NoError(t, err)
	assert.Equal(t, "34052387b5efb6132a42b244cff52a85a507ab319c414564d7a89207d4473672", fact)
}

func TestOrderTypeStrings(t *testing.T) {
	t.Parallel()
	orderMap := map[order.Type]string{
		order.Limit:            "LIMIT",
		order.Market:           "MARKET",
		order.StopLimit:        "STOP_LIMIT",
		order.StopMarket:       "STOP_MARKET",
		order.TakeProfit:       "TAKE_PROFIT_LIMIT",
		order.TakeProfitMarket: "TAKE_PROFIT_MARKET",
	}
	for k := range orderMap {
		assert.Equal(t, orderTypeString(k), orderMap[k])
	}
}

func TestGetRepaymentPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetRepaymentPrice(t.Context(), []RepaymentTokenAndAmount{}, "client-id-here")
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.GetRepaymentPrice(t.Context(), []RepaymentTokenAndAmount{{}}, "")
	require.ErrorIs(t, err, errClientIDMissing)
	_, err = e.GetRepaymentPrice(t.Context(), []RepaymentTokenAndAmount{{}}, "client-id-here")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetRepaymentPrice(t.Context(), []RepaymentTokenAndAmount{{
		Token: currency.ETH,
	}}, "client-id-here")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRepaymentPrice(t.Context(), []RepaymentTokenAndAmount{
		{
			Token:  currency.BTC,
			Amount: 123.4,
		},
	}, "client-id-here")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarshalJSON(t *testing.T) {
	t.Parallel()
	data := LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}}
	marshalData, err := json.Marshal(data)
	require.NoError(t, err)
	assert.NotNil(t, marshalData)
}

func TestUserManualRepayment(t *testing.T) {
	t.Parallel()
	_, err := e.UserManualRepayment(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.UserManualRepayment(t.Context(), &UserManualRepaymentParams{ExpiryTime: time.Now().Add(-time.Hour * 48)})
	require.ErrorIs(t, err, errClientIDMissing)
	_, err = e.UserManualRepayment(t.Context(), &UserManualRepaymentParams{ClientID: "1234567"})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.UserManualRepayment(t.Context(), &UserManualRepaymentParams{ClientID: "1234567",
		LoanRepaymentTokenAndAmount: LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}}})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.UserManualRepayment(t.Context(), &UserManualRepaymentParams{ClientID: "1234567",
		LoanRepaymentTokenAndAmount: LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}},
		PoolRepaymentTokensDetail:   LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}}})
	require.ErrorIs(t, err, errExpirationTimeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UserManualRepayment(t.Context(), &UserManualRepaymentParams{
		ClientID:                    "1234567",
		ExpiryTime:                  time.Now().Add(-time.Hour * 48),
		PoolRepaymentTokensDetail:   LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}},
		LoanRepaymentTokenAndAmount: LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}
