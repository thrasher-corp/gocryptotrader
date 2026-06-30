package apexpro

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
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
	if err := e.UpdateTradablePairs(context.Background()); err != nil {
		log.Fatal(err)
	}
	if err := e.enablePairs(); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

var enabledAssetPair map[asset.Item]currency.Pair

func (e *Exchange) enablePairs() error {
	var err error
	enabledAssetPair = make(map[asset.Item]currency.Pair, 7)
	enabledAssetPair[asset.RealWorldAsset], err = e.FormatExchangeCurrency(currency.Pair{Base: currency.NewCode("AAPL"), Quote: currency.USDT}, asset.RealWorldAsset)
	if err != nil {
		log.Fatal(err)
	}
	enabledAssetPair[asset.PerpetualContract], err = e.FormatExchangeCurrency(currency.Pair{Base: currency.BTC, Quote: currency.USDT}, asset.PerpetualContract)
	if err != nil {
		log.Fatal(err)
	}
	// store the pairs into the enabled pairs
	return storeTestPairs(e)
}

func storeTestPairs(e *Exchange) error {
	for a, p := range enabledAssetPair {
		if err := e.CurrencyPairs.StorePairs(a, []currency.Pair{p}, false); err != nil {
			return err
		}
		if err := e.CurrencyPairs.StorePairs(a, []currency.Pair{p}, true); err != nil {
			return err
		}
	}
	return nil
}

func getPair(assetType asset.Item) currency.Pair {
	pair, ok := enabledAssetPair[assetType]
	if !ok {
		return pair
	}
	pairs, err := e.GetEnabledPairs(assetType)
	if err != nil {
		log.Fatal(err)
	}
	return pairs[0]
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
	_, err := e.GetMarketDepthV3(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetMarketDepthV3(t.Context(), "BTC-USDC", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNewestTradingDataV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetNewestTradingDataV3(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetNewestTradingDataV3(t.Context(), "BTC-USDC", 10)
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

func TestGetTickerDataV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickerDataV3(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetTickerDataV3(t.Context(), "BTC-USDC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingHistoryRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingHistoryRateV3(t.Context(), "", time.Time{}, time.Time{}, 10, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFundingHistoryRateV3(t.Context(), "BTC-USDT", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetFundingHistoryRateV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingHistoryRateV2(t.Context(), "", time.Time{}, time.Time{}, 10, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFundingHistoryRateV2(t.Context(), "BTC-USDT", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetFundingHistoryRateV1(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingHistoryRateV1(t.Context(), "", time.Time{}, time.Time{}, 10, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFundingHistoryRateV1(t.Context(), "BTC-USDT", time.Time{}, time.Time{}, 0, 0)
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
	_, err := e.GetCheckIfUserExistsV2(t.Context(), "")
	require.ErrorIs(t, err, errEthereumAddressMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
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
	_, err := e.EditUserDataV3(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EditUserDataV3(t.Context(), &EditUserDataRequest{
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
	_, err := e.EditUserDataV2(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EditUserDataV2(t.Context(), &EditUserDataRequest{
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
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
}

func TestGetWorstPriceV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetWorstPriceV3(t.Context(), "", "SELL", 1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetWorstPriceV3(t.Context(), "BTC-USDC", "", 1)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.GetWorstPriceV3(t.Context(), "BTC-USDC", "SELL", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

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
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

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
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

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
	result, err := e.CreateOrderV3(t.Context(), &CreateOrderRequest{
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
	_, err = e.CancelPerpOrderV2(t.Context(), "12345", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

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

func TestGetOpenOrdersV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrdersV1(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllOrderHistory(t.Context(), "BTC-USDC", "SELL", "MARKET", "OPEN", "HISTORY", time.Now(), time.Now().Add(-time.Hour), 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllOrderHistory(t.Context(), "BTC-USDC", "SELL", "MARKET", "OPEN", "HISTORY", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOrderHistoryV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllOrderHistoryV2(t.Context(), currency.EMPTYCODE, "BTC-USDC", "SELL", "MARKET", "OPEN", "HISTORY", time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetAllOrderHistoryV2(t.Context(), currency.USDT, "BTC-USDC", "SELL", "MARKET", "OPEN", "HISTORY", time.Now(), time.Now().Add(-time.Hour), 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllOrderHistoryV2(t.Context(), currency.USDT, "BTC-USDC", "SELL", "MARKET", "OPEN", "HISTORY", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllOrderHistoryV1(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllOrderHistoryV1(t.Context(), "BTC-USDC", "SELL", "MARKET", "OPEN", "HISTORY", time.Now(), time.Now().Add(-time.Hour), 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

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
	_, err := e.WithdrawAsset(t.Context(), &AssetWithdrawalRequest{
		ClientWithdrawID: "123123",
		Timestamp:        time.Now(),
		EthereumAddress:  ethereumAddress,
		L2Key:            starkKey,
		ToChainID:        "3",
		L2SourceTokenID:  currency.USDC,
		L1TargetTokenID:  currency.USDC,
		IsFastWithdraw:   false,
	})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.WithdrawAsset(t.Context(), &AssetWithdrawalRequest{
		Amount:          1,
		Timestamp:       time.Now(),
		EthereumAddress: ethereumAddress,
		L2Key:           starkKey,
		ToChainID:       "3",
		L2SourceTokenID: currency.USDC,
		L1TargetTokenID: currency.USDC,
		IsFastWithdraw:  false,
	})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	_, err = e.WithdrawAsset(t.Context(), &AssetWithdrawalRequest{
		Amount:           1,
		ClientWithdrawID: "123123",
		EthereumAddress:  ethereumAddress,
		L2Key:            starkKey,
		ToChainID:        "3",
		L2SourceTokenID:  currency.USDC,
		L1TargetTokenID:  currency.USDC,
		IsFastWithdraw:   false,
	})
	require.ErrorIs(t, err, errInvalidTimestamp)

	// The following validations execute after GetCredentials, so they require credentials to be set.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.WithdrawAsset(t.Context(), &AssetWithdrawalRequest{
		Amount:           1,
		ClientWithdrawID: "123123",
		Timestamp:        time.Now(),
		EthereumAddress:  "0x0330eBB5e894720e6746070371F9Fd797BE9D074",
		L2Key:            "0x1",
	})
	require.ErrorIs(t, err, errChainIDMissing)
	_, err = e.WithdrawAsset(t.Context(), &AssetWithdrawalRequest{
		Amount:           1,
		ClientWithdrawID: "123123",
		Timestamp:        time.Now(),
		EthereumAddress:  "0x0330eBB5e894720e6746070371F9Fd797BE9D074",
		L2Key:            "0x1",
		ToChainID:        "3",
	})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WithdrawAsset(t.Context(), &AssetWithdrawalRequest{
		Amount:           1,
		ClientWithdrawID: "123123",
		Timestamp:        time.Now(),
		EthereumAddress:  "0x0330eBB5e894720e6746070371F9Fd797BE9D074",
		L2Key:            "0x1",
		ToChainID:        "3",
		L2SourceTokenID:  currency.USDC,
	})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.WithdrawAsset(t.Context(), &AssetWithdrawalRequest{
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
	_, err := e.UserWithdrawalV2(t.Context(), &WithdrawalRequest{})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.UserWithdrawalV2(t.Context(), &WithdrawalRequest{Amount: 1})
	require.ErrorIs(t, err, errEthereumAddressMissing)
	_, err = e.UserWithdrawalV2(t.Context(), &WithdrawalRequest{Amount: 1, EthereumAddress: "0x0330eBB5e894720e6746070371F9Fd797BE9D074"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UserWithdrawalV2(t.Context(),
		&WithdrawalRequest{
			Amount:          1,
			Asset:           currency.USDC,
			EthereumAddress: "0x0330eBB5e894720e6746070371F9Fd797BE9D074",
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawalToAddressV2(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawalToAddressV2(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.WithdrawalToAddressV2(t.Context(), &WithdrawalToAddressRequest{Asset: currency.ETH})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.WithdrawalToAddressV2(t.Context(), &WithdrawalToAddressRequest{
		Amount: .1,
	})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.WithdrawalToAddressV2(t.Context(), &WithdrawalToAddressRequest{
		Amount:        1,
		ClientOrderID: "12334",
		Asset:         currency.BTC,
	})
	require.ErrorIs(t, err, errEthereumAddressMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawalToAddressV2(t.Context(), &WithdrawalToAddressRequest{
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
	result, err := e.WithdrawalToAddressV1(t.Context(), &WithdrawalToAddressRequest{
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
	_, err = e.orderCreationParamsFilter(t.Context(), &CreateOrderRequest{Side: order.Buy.String()})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	futuresTradablePair, err := currency.NewPairFromString("BTC-USDC")
	require.NoError(t, err)
	arg := &CreateOrderRequest{Symbol: futuresTradablePair}
	_, err = e.orderCreationParamsFilter(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	arg.Side = order.Buy.String()
	_, err = e.orderCreationParamsFilter(t.Context(), &CreateOrderRequest{Symbol: futuresTradablePair, Side: order.Buy.String()})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	arg.OrderType = order.Limit.String()
	_, err = e.orderCreationParamsFilter(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	arg.Size = 2
	_, err = e.orderCreationParamsFilter(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	arg.Price = 123
	arg.LimitFee = -1
	_, err = e.orderCreationParamsFilter(t.Context(), arg)
	require.ErrorIs(t, err, errLimitFeeRequired)
	arg.LimitFee = 0.003
	_, err = e.orderCreationParamsFilter(t.Context(), arg)
	require.ErrorIs(t, err, errExpirationTimeRequired)
}

func TestOrderCreationParamsFilterV3(t *testing.T) {
	t.Parallel()
	_, err := e.orderCreationParamsFilterV3(t.Context(), &CreateOrderRequest{})
	require.ErrorIs(t, err, order.ErrOrderDetailIsNil)
	_, err = e.orderCreationParamsFilterV3(t.Context(), &CreateOrderRequest{Side: order.Buy.String()})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	futuresTradablePair, err := currency.NewPairFromString("BTC-USDC")
	require.NoError(t, err)
	arg := &CreateOrderRequest{Symbol: futuresTradablePair}
	_, err = e.orderCreationParamsFilterV3(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	arg.Side = order.Buy.String()
	_, err = e.orderCreationParamsFilterV3(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	arg.OrderType = order.Limit.String()
	_, err = e.orderCreationParamsFilterV3(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	arg.Size = 2
	_, err = e.orderCreationParamsFilterV3(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
}

func TestIntervalToString(t *testing.T) {
	t.Parallel()
	_, err := intervalToString(kline.Interval(0))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	result, err := intervalToString(kline.FiveMin)
	require.NoError(t, err)
	assert.Equal(t, "5", result)
}

func TestIntervalFromString(t *testing.T) {
	t.Parallel()
	_, err := intervalFromString("unsupported")
	require.ErrorIs(t, err, kline.ErrInvalidInterval)
	result, err := intervalFromString("5")
	require.NoError(t, err)
	assert.Equal(t, kline.FiveMin, result)
}

func TestCreateOrderV1(t *testing.T) {
	t.Parallel()
	futuresTradablePair, err := currency.NewPairFromString("ETH-USDC")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	if e.UserAccountDetail == nil {
		e.UserAccountDetail, err = e.GetUserAccountDataV2(t.Context())
		require.NoError(t, err)
		require.NotNil(t, e.UserAccountDetail)
	}

	result, err := e.CreateOrderV1(t.Context(), &CreateOrderRequest{
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
	result, err := e.CreateOrderV2(t.Context(), &CreateOrderRequest{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FastWithdrawalV2(t.Context(), &FastWithdrawalRequest{
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
	result, err := e.FastWithdrawalV1(t.Context(), &FastWithdrawalRequest{
		Amount:  1,
		Asset:   currency.USDC,
		ChainID: "1",
		Fees:    0,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFastWithdrawalV3(t *testing.T) {
	t.Parallel()
	_, err := e.FastWithdrawalV3(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.FastWithdrawalV3(t.Context(), &FastWithdrawalRequest{Asset: currency.USDC, ChainID: "1"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.FastWithdrawalV3(t.Context(), &FastWithdrawalRequest{Amount: 1, ChainID: "1"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.FastWithdrawalV3(t.Context(), &FastWithdrawalRequest{Amount: 1, Asset: currency.USDC})
	require.ErrorIs(t, err, errChainIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FastWithdrawalV3(t.Context(), &FastWithdrawalRequest{
		Amount:       1,
		ClientID:     "123213",
		Expiration:   time.Now().Add(time.Hour * 45).UnixMilli(),
		Asset:        currency.USDC,
		ERC20Address: ethereumAddress,
		ChainID:      "56",
		Fees:         0,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCrossChainWithdrawalsV3(t *testing.T) {
	t.Parallel()
	_, err := e.CrossChainWithdrawalsV3(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CrossChainWithdrawalsV3(t.Context(), &FastWithdrawalRequest{Asset: currency.USDC, ChainID: "1"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.CrossChainWithdrawalsV3(t.Context(), &FastWithdrawalRequest{Amount: 1, ChainID: "1"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CrossChainWithdrawalsV3(t.Context(), &FastWithdrawalRequest{Amount: 1, Asset: currency.USDC})
	require.ErrorIs(t, err, errChainIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CrossChainWithdrawalsV3(t.Context(), &FastWithdrawalRequest{
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
	pairs, err := e.GetEnabledPairs(asset.PerpetualContract)
	assert.NoErrorf(t, err, "FetchTradablePairs should not error for %s", asset.PerpetualContract)
	assert.NotEmptyf(t, pairs, "Should get some pairs for %s", asset.PerpetualContract)

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.PerpetualContract)
	require.NoError(t, err)

	execLimits, err := e.GetOrderExecutionLimits(asset.PerpetualContract, pairs[0])
	assert.NoErrorf(t, err, "GetOrderExecutionLimits should not error for %s pair %s", asset.PerpetualContract, pairs[0])
	assert.Positivef(t, execLimits.MinPrice, "MinPrice should be positive for %s pair %s", asset.PerpetualContract, pairs[0])
	assert.Positivef(t, execLimits.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s pair %s", asset.PerpetualContract, pairs[0])
	assert.Positivef(t, execLimits.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s pair %s", asset.PerpetualContract, pairs[0])
	assert.Positivef(t, execLimits.MaximumBaseAmount, "MaximumBaseAmount should be positive for %s pair %s", asset.PerpetualContract, pairs[0])
	assert.Positivef(t, execLimits.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s pair %s", asset.PerpetualContract, pairs[0])
	assert.Positivef(t, execLimits.MarketMaxQty, "MarketMaxQty should be positive for %s pair %s", asset.PerpetualContract, pairs[0])
	assert.Positivef(t, execLimits.MaxTotalOrders, "MaxTotalOrders should be positive for %s pair %s", asset.PerpetualContract, pairs[0])
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.PerpetualContract, currency.NewBTCUSDT())
	require.NoError(t, err)
	assert.True(t, is)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesContractDetails(t.Context(), asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	for _, assetType := range e.GetAssetTypes(true) {
		result, err := e.GetHistoricCandles(t.Context(), getPair(assetType), assetType, kline.FifteenMin, time.Now().Add(-time.Hour*6), time.Now())
		require.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	for _, assetType := range e.GetAssetTypes(true) {
		result, err := e.GetHistoricCandlesExtended(t.Context(), getPair(assetType), assetType, kline.FifteenMin, time.Now().Add(-time.Hour*6), time.Now())
		require.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	result, err := e.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.FetchTradablePairs(t.Context(), asset.RealWorldAsset)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	// Use a dedicated instance so mutating the stored pairs does not race with parallel
	// tests reading the shared exchange's enabled and available pairs.
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	assert.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs should not error")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	for _, assetType := range e.GetAssetTypes(true) {
		result, err := e.UpdateTicker(t.Context(), getPair(assetType), assetType)
		require.NoError(t, err)
		assert.NotNil(t, result)
	}
}

// perpetualContractOnce bootstraps the PerpetualContract asset with live pairs.
var perpetualContractOnce sync.Once

func setupPerpetualContract(tb testing.TB) {
	tb.Helper()
	perpetualContractOnce.Do(func() {
		pairs, err := e.FetchTradablePairs(context.Background(), asset.PerpetualContract)
		require.NoError(tb, err, "FetchTradablePairs must not error")
		require.NotEmpty(tb, pairs, "FetchTradablePairs must return perpetual contract pairs")
		require.NoError(tb, e.CurrencyPairs.SetAssetEnabled(asset.PerpetualContract, true), "SetAssetEnabled must not error")
		require.NoError(tb, e.SetPairs(pairs, asset.PerpetualContract, false), "SetPairs available must not error")
		require.NoError(tb, e.SetPairs(pairs[:1], asset.PerpetualContract, true), "SetPairs enabled must not error")
	})
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()

	setupPerpetualContract(t)

	for _, assetType := range e.GetAssetTypes(true) {
		err := e.UpdateTickers(t.Context(), assetType)
		require.NoError(t, err)

		result, err := ticker.GetTicker(e.Name, getPair(assetType), assetType)
		require.NoError(t, err)
		assert.Positive(t, result.Last, "last price should be positive")
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	setupPerpetualContract(t)
	enabledPairs, err := e.GetEnabledPairs(asset.PerpetualContract)
	require.NoError(t, err)
	require.NotEmpty(t, enabledPairs, "must have enabled perpetual contract pairs")

	perperualContractPair := getPair(asset.PerpetualContract)
	result, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  perperualContractPair.Base.Item,
		Quote: perperualContractPair.Quote.Item,
		Asset: asset.PerpetualContract,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result, "should return open interest for the requested pair")

	result, err = e.GetOpenInterest(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, result, "should return open interest for all enabled pairs")

	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{Asset: asset.Spot})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "non-perpetual asset should error")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	setupPerpetualContract(t)
	enabledPairs, err := e.GetEnabledPairs(asset.PerpetualContract)
	require.NoError(t, err)
	require.NotEmpty(t, enabledPairs, "must have enabled perpetual contract pairs")

	result, err := e.GetCurrencyTradeURL(t.Context(), asset.PerpetualContract, getPair(asset.PerpetualContract))
	require.NoError(t, err)
	assert.NotEmpty(t, result, "trade URL should not be empty")

	_, err = e.GetCurrencyTradeURL(t.Context(), asset.PerpetualContract, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "empty pair should error")
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetAvailableTransferChains(t.Context(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	for _, assetType := range e.GetAssetTypes(true) {
		result, err := e.UpdateOrderbook(t.Context(), getPair(assetType), assetType)
		require.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateAccountBalances(t.Context(), asset.Spot)
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
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.USDC, asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	for _, assetType := range e.GetAssetTypes(true) {
		result, err := e.GetRecentTrades(t.Context(), getPair(assetType), assetType)
		require.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetServerTime(t.Context(), asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOrders(t.Context(), &order.Cancel{
		AssetType:  asset.PerpetualContract,
		MarginType: margin.Isolated,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "", currency.EMPTYPAIR, asset.PerpetualContract)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderInfo(t.Context(), "614463889001677573", currency.EMPTYPAIR, asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.PerpetualContract,
		Type:      order.Limit,
		Side:      order.Buy,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTransferErc20Fact(t *testing.T) {
	fact, err := GetTransferErc20Fact(
		3, "0x1234567890123456789012345678901234567890",
		"123.456", "0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa",
		"0x1234567890abcdef",
	)
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
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

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
	_, err = e.UserManualRepayment(t.Context(), &UserManualRepaymentRequest{ExpiryTime: time.Now().Add(-time.Hour * 48)})
	require.ErrorIs(t, err, errClientIDMissing)
	_, err = e.UserManualRepayment(t.Context(), &UserManualRepaymentRequest{ClientID: "1234567"})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.UserManualRepayment(t.Context(), &UserManualRepaymentRequest{
		ClientID:                    "1234567",
		LoanRepaymentTokenAndAmount: LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}},
	})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.UserManualRepayment(t.Context(), &UserManualRepaymentRequest{
		ClientID:                    "1234567",
		LoanRepaymentTokenAndAmount: LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}},
		PoolRepaymentTokensDetail:   LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}},
	})
	require.ErrorIs(t, err, errExpirationTimeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UserManualRepayment(t.Context(), &UserManualRepaymentRequest{
		ClientID:                    "1234567",
		ExpiryTime:                  time.Now().Add(-time.Hour * 48),
		PoolRepaymentTokensDetail:   LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}},
		LoanRepaymentTokenAndAmount: LoanRepaymentTokenAndAmountList{{Token: currency.BTC, Amount: 123.4}},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRegisterRWAAccount(t *testing.T) {
	t.Parallel()
	_, err := e.RegisterRWAAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.RegisterRWAAccount(t.Context(), &RWARegisterAccountRequest{MasterAccountID: "123", Signature: "sig"})
	require.ErrorIs(t, err, errL2KeyMissing)

	_, err = e.RegisterRWAAccount(t.Context(), &RWARegisterAccountRequest{L2Key: "0xabc", Signature: "sig"})
	require.ErrorIs(t, err, errMasterAccountIDMissing)

	_, err = e.RegisterRWAAccount(t.Context(), &RWARegisterAccountRequest{L2Key: "0xabc", MasterAccountID: "123"})
	require.ErrorIs(t, err, errSignatureMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RegisterRWAAccount(t.Context(), &RWARegisterAccountRequest{
		L2Key:           "0xabc",
		MasterAccountID: "123",
		Signature:       "sig",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateRWAAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateRWAAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.GenerateRWAAPIKey(t.Context(), &RWAGenerateAPIKeyRequest{WalletName: "w", AccountID: "1", EthAddress: "0xeth", Signature: "sig"})
	require.ErrorIs(t, err, errL2KeyMissing)

	_, err = e.GenerateRWAAPIKey(t.Context(), &RWAGenerateAPIKeyRequest{L2Key: "0xabc", AccountID: "1", EthAddress: "0xeth", Signature: "sig"})
	require.ErrorIs(t, err, errWalletNameMissing)

	_, err = e.GenerateRWAAPIKey(t.Context(), &RWAGenerateAPIKeyRequest{L2Key: "0xabc", WalletName: "w", EthAddress: "0xeth", Signature: "sig"})
	require.ErrorIs(t, err, errAccountIDMissing)

	_, err = e.GenerateRWAAPIKey(t.Context(), &RWAGenerateAPIKeyRequest{L2Key: "0xabc", WalletName: "w", AccountID: "1", Signature: "sig"})
	require.ErrorIs(t, err, errEthereumAddressMissing)

	_, err = e.GenerateRWAAPIKey(t.Context(), &RWAGenerateAPIKeyRequest{L2Key: "0xabc", WalletName: "w", AccountID: "1", EthAddress: "0xeth"})
	require.ErrorIs(t, err, errSignatureMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GenerateRWAAPIKey(t.Context(), &RWAGenerateAPIKeyRequest{
		L2Key:      "0xabc",
		WalletName: "w",
		AccountID:  "1",
		EthAddress: "0xeth",
		Signature:  "sig",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRWAAccountData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRWAAccountData(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferContractToRWA(t *testing.T) {
	t.Parallel()
	_, err := e.TransferContractToRWA(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.TransferContractToRWA(t.Context(), &RWATransferRequest{Asset: currency.USDT, ReceiverAccountID: "1", ReceiverL2Key: "0xabc", ReceiverAddress: "0xeth", ClientID: "c"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.TransferContractToRWA(t.Context(), &RWATransferRequest{Amount: 1, ReceiverAccountID: "1", ReceiverL2Key: "0xabc", ReceiverAddress: "0xeth", ClientID: "c"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.TransferContractToRWA(t.Context(), &RWATransferRequest{Amount: 1, Asset: currency.USDT, ReceiverL2Key: "0xabc", ReceiverAddress: "0xeth", ClientID: "c"})
	require.ErrorIs(t, err, errReceiverAccountIDMissing)

	_, err = e.TransferContractToRWA(t.Context(), &RWATransferRequest{Amount: 1, Asset: currency.USDT, ReceiverAccountID: "1", ReceiverAddress: "0xeth", ClientID: "c"})
	require.ErrorIs(t, err, errReceiverL2KeyMissing)

	_, err = e.TransferContractToRWA(t.Context(), &RWATransferRequest{Amount: 1, Asset: currency.USDT, ReceiverAccountID: "1", ReceiverL2Key: "0xabc", ClientID: "c"})
	require.ErrorIs(t, err, errReceiverAddressMissing)

	_, err = e.TransferContractToRWA(t.Context(), &RWATransferRequest{Amount: 1, Asset: currency.USDT, ReceiverAccountID: "1", ReceiverL2Key: "0xabc", ReceiverAddress: "0xeth"})
	require.ErrorIs(t, err, errClientIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.TransferContractToRWA(t.Context(), &RWATransferRequest{
		Amount:            1,
		Asset:             currency.USDT,
		ReceiverAccountID: "850000000000000001",
		ReceiverL2Key:     "0xabc",
		ReceiverAddress:   ethereumAddress,
		ClientID:          "1234567",
		Signature:         "sig",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferRWAToContract(t *testing.T) {
	t.Parallel()
	_, err := e.TransferRWAToContract(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.TransferRWAToContract(t.Context(), &RWATransferRequest{Amount: 1, Asset: currency.USDT, ReceiverAccountID: "1", ReceiverL2Key: "0xabc", ReceiverAddress: "0xeth"})
	require.ErrorIs(t, err, errClientIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.TransferRWAToContract(t.Context(), &RWATransferRequest{
		Amount:            1,
		Asset:             currency.USDT,
		ReceiverAccountID: "123",
		ReceiverL2Key:     "0xabc",
		ReceiverAddress:   ethereumAddress,
		ClientID:          "1234567",
		Signature:         "sig",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateRWAOrder(t *testing.T) {
	t.Parallel()
	rwaPair, err := currency.NewPairFromString("AAPL-USDT")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateRWAOrder(t.Context(), &CreateOrderRequest{
		Symbol:      rwaPair,
		Side:        order.Buy.String(),
		OrderType:   "LIMIT",
		Size:        0.01,
		Price:       150,
		TimeInForce: "GOOD_TIL_CANCEL",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAssetTypeFromSymbol(t *testing.T) {
	t.Parallel()
	rwa := &Exchange{SymbolsConfig: &AllSymbolsConfigs{}}
	rwa.SymbolsConfig.ContractConfig.StockContract = []*StockContractDetail{{Symbol: "AAPL-USDT"}}
	assert.Equal(t, asset.RealWorldAsset, rwa.assetTypeFromSymbol("AAPL-USDT"), "stock symbol should resolve to RWA")
	assert.Equal(t, asset.PerpetualContract, rwa.assetTypeFromSymbol("BTC-USDT"), "unknown symbol should default to perpetual")

	empty := &Exchange{}
	assert.Equal(t, asset.PerpetualContract, empty.assetTypeFromSymbol("AAPL-USDT"), "nil config should default to perpetual")
}

func TestUpdateOrderExecutionLimitsRealWorldAsset(t *testing.T) {
	t.Parallel()
	rwa := &Exchange{SymbolsConfig: &AllSymbolsConfigs{}}
	rwa.Name = "Apexpro-RWA-Test"
	rwa.SymbolsConfig.ContractConfig.StockContract = []*StockContractDetail{{
		Symbol:                   "AAPL-USDT",
		TickSize:                 0.01,
		StepSize:                 0.001,
		MinOrderSize:             0.001,
		MaxOrderSize:             1000,
		IncrementalPositionValue: 1,
		MaxPositionValue:         100000,
		MaxPositionSize:          500,
	}}
	err := rwa.UpdateOrderExecutionLimits(t.Context(), asset.RealWorldAsset)
	require.NoError(t, err)

	cp, err := currency.NewPairFromString("AAPL-USDT")
	require.NoError(t, err)
	lim, err := rwa.GetOrderExecutionLimits(asset.RealWorldAsset, cp)
	require.NoError(t, err)
	assert.Positive(t, lim.MinimumBaseAmount, "MinimumBaseAmount should be positive")
	assert.Positive(t, lim.PriceStepIncrementSize, "PriceStepIncrementSize should be positive")
	assert.Equal(t, int64(500), lim.MaxTotalOrders, "MaxTotalOrders should match config")

	// Unsupported assets should be a no-op rather than an error.
	require.NoError(t, rwa.UpdateOrderExecutionLimits(t.Context(), asset.Spot))
}

func TestUpdateAccountBalancesRealWorldAsset(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UpdateAccountBalances(t.Context(), asset.RealWorldAsset)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEditUserDataV1(t *testing.T) {
	t.Parallel()
	_, err := e.EditUserDataV1(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EditUserDataV1(t.Context(), &EditUserDataRequest{
		Email:                    "someone@thrasher.io",
		Username:                 "Thrasher",
		Country:                  "Ethiopia",
		EmailNotifyGeneralEnable: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCrossChainWithdrawalsV1(t *testing.T) {
	t.Parallel()
	_, err := e.CrossChainWithdrawalsV1(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CrossChainWithdrawalsV1(t.Context(), &FastWithdrawalRequest{Asset: currency.USDC, ChainID: "1"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.CrossChainWithdrawalsV1(t.Context(), &FastWithdrawalRequest{Amount: 1, ChainID: "1"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CrossChainWithdrawalsV1(t.Context(), &FastWithdrawalRequest{Amount: 1, Asset: currency.USDC})
	require.ErrorIs(t, err, errChainIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CrossChainWithdrawalsV1(t.Context(), &FastWithdrawalRequest{
		Amount:  1,
		Asset:   currency.USDC,
		ChainID: "1",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCrossChainWithdrawalsV2(t *testing.T) {
	t.Parallel()
	_, err := e.CrossChainWithdrawalsV2(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CrossChainWithdrawalsV2(t.Context(), &FastWithdrawalRequest{Asset: currency.USDC, ChainID: "1"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.CrossChainWithdrawalsV2(t.Context(), &FastWithdrawalRequest{Amount: 1, ChainID: "1"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.CrossChainWithdrawalsV2(t.Context(), &FastWithdrawalRequest{Amount: 1, Asset: currency.USDC})
	require.ErrorIs(t, err, errChainIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CrossChainWithdrawalsV2(t.Context(), &FastWithdrawalRequest{
		Amount:  1,
		Asset:   currency.USDC,
		ChainID: "1",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTokenByID(t *testing.T) {
	t.Parallel()
	// An unknown token ID must resolve to an empty token rather than panicking.
	assert.Empty(t, e.GetTokenByID("non-existent-token-id"), "unknown token ID should return an empty token")
}
