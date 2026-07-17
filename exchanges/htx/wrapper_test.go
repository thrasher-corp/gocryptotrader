package htx

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)
}

func TestUpdateTickerSpot(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("INV", "ALID", "-"), asset.Spot)
	assert.ErrorContains(t, err, "invalid symbol")
	_, err = e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Spot)
	require.NoError(t, err)
}

func TestUpdateTickerCMF(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("INV", "ALID", "_"), asset.CoinMarginedFutures)
	assert.ErrorContains(t, err, "symbol data error")
	_, err = e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("BTC", "USD", "_"), asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateTickerFutures(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), btccwPair, asset.Futures)
	require.NoError(t, err)
}

func TestUpdateTickerUSDTMarginedFutures(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), btcusdtPair, asset.USDTMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateOrderbookSpot(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btcusdtPair, asset.Spot)
	require.NoError(t, err)
}

func TestUpdateOrderbookCMF(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateOrderbookFuture(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btccwPair, asset.Futures)
	require.NoError(t, err)
	_, err = e.UpdateOrderbook(t.Context(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateOrderbookUSDTMarginedFutures(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btcusdtPair, asset.USDTMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateOrderbookUnsupportedAsset(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btcusdtPair, asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject unsupported assets")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	updatePairsOnce(t, e)
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewBTCUSDT()},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)

	getOrdersRequest.Pairs = []currency.Pair{btcusdPair}
	getOrdersRequest.AssetType = asset.CoinMarginedFutures
	_, err = e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)
	getOrdersRequest.Pairs = []currency.Pair{btcFutureDatedPair}
	getOrdersRequest.AssetType = asset.Futures
	_, err = e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures})
	require.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	updatePairsOnce(t, e)

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err := e.GetHistoricCandles(t.Context(), btcusdtPair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcusdtPair, asset.Spot, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	updatePairsOnce(t, e)

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err := e.GetHistoricCandlesExtended(t.Context(), btcusdtPair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	// demonstrate that adjusting time doesn't wreck non-day intervals
	_, err = e.GetHistoricCandlesExtended(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneHour, endTime.AddDate(0, 0, -1), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), time.Now())
	require.NoError(t, err)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneHour, endTime.AddDate(0, 0, -1), time.Now())
	require.NoError(t, err)
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickers(t.Context())
	require.NoError(t, err)
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	st, err := e.GetCurrentServerTime(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, st, "GetCurrentServerTime should return a time")
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, st, "GetServerTime should return a time")
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	_, err := h.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject nil fee builder")

	_, err = h.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")
	assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "fee type should fall back when credentials are not valid")
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	_, err := e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := e.FormatWithdrawPermissions()
	assert.Equal(t, expectedResult, withdrawPermissions)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewBTCUSDT()},
		Side:      order.AnySide,
	}

	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestGetActiveOrdersValidation(t *testing.T) {
	t.Parallel()

	getOrdersRequest := order.MultiOrderRequest{
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	require.ErrorContains(t, err, "currency must be supplied", "GetActiveOrders must require a currency pair for spot")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name        string
		pair        currency.Pair
		asset       asset.Item
		expectedErr error
	}{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
				Type:      order.AnyType,
				Pairs:     []currency.Pair{tt.pair},
				AssetType: tt.asset,
				Side:      order.AnySide,
			})
			require.ErrorIs(t, err, tt.expectedErr, "GetActiveOrders must return the expected branch error")
		})
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountFundingHistory(t.Context())
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetAccountFundingHistory must return unsupported")
}

func TestGetAccountID(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	_, err := h.GetAccountID(t.Context())
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "GetAccountID must require credentials")
}

// TestSubmitOrder and below can impact your orders on the exchange. Enable canManipulateRealOrders to run them
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	accounts, err := e.GetAccounts(t.Context())
	require.NoError(t, err, "GetAccounts must not error")

	orderSubmission := &order.Submit{
		Exchange: e.Name,
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.USDT,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     5,
		Amount:    1,
		ClientID:  strconv.FormatInt(accounts[0].ID, 10),
		AssetType: asset.Spot,
	}
	response, err := e.SubmitOrder(t.Context(), orderSubmission)
	require.NoError(t, err)
	assert.Equal(t, order.New, response.Status, "response status should be correct")
}

func TestSubmitOrderValidation(t *testing.T) {
	t.Parallel()

	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject nil submissions")

	_, err = e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  e.Name,
		Pair:      btcusdtPair,
		AssetType: asset.Spot,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "invalid",
	})
	require.Error(t, err, "SubmitOrder must reject invalid spot account IDs")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name        string
		pair        currency.Pair
		asset       asset.Item
		expectedErr error
	}{
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.SubmitOrder(t.Context(), &order.Submit{
				Exchange:  h.Name,
				Pair:      tt.pair,
				AssetType: tt.asset,
				Side:      order.Buy,
				Type:      order.Limit,
				Price:     1,
				Amount:    1,
				Leverage:  1,
			})
			require.ErrorIs(t, err, tt.expectedErr, "SubmitOrder must return the expected branch error")
		})
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      btcusdtPair,
		AssetType: asset.Spot,
	}

	err := e.CancelOrder(t.Context(), orderCancellation)
	require.NoError(t, err)
}

func TestCancelOrderValidation(t *testing.T) {
	t.Parallel()

	err := e.CancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelOrder must reject nil cancellations")

	err = e.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "invalid",
		Pair:      btcusdtPair,
		AssetType: asset.Spot,
	})
	require.Error(t, err, "CancelOrder must reject non-numeric spot order IDs")

	err = e.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "1",
		Pair:      btcusdtPair,
		AssetType: asset.Binary,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelOrder must reject unsupported assets")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name  string
		pair  currency.Pair
		asset asset.Item
	}{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures},
		{name: "futures", pair: btccwPair, asset: asset.Futures},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := h.CancelOrder(t.Context(), &order.Cancel{
				OrderID:   "1",
				Pair:      tt.pair,
				AssetType: tt.asset,
			})
			require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "CancelOrder must require credentials")
		})
	}
}

func TestCancelBatchOrdersValidation(t *testing.T) {
	t.Parallel()

	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must reject empty requests")

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrders must require an order ID")
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	_, err := e.CancelAllOrders(t.Context(), &orderCancellation)
	require.NoError(t, err)
}

func TestCancelAllOrdersValidation(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject nil cancellations")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name        string
		pair        currency.Pair
		asset       asset.Item
		expectedErr error
	}{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.CancelAllOrders(t.Context(), &order.Cancel{
				OrderID:   "1",
				Pair:      tt.pair,
				AssetType: tt.asset,
			})
			require.ErrorIs(t, err, tt.expectedErr, "CancelAllOrders must return the expected branch error")
		})
	}
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	for _, a := range []asset.Item{asset.Spot, asset.CoinMarginedFutures, asset.Futures} {
		_, err := e.UpdateAccountBalances(t.Context(), a)
		assert.NoErrorf(t, err, "UpdateAccountBalances should not error for asset %s", a)
	}
}

func TestUpdateAccountBalancesValidation(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name        string
		asset       asset.Item
		expectedErr error
	}{
		{name: "spot", asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "coin margined futures", asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "usdt margined futures", asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "futures", asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "unsupported asset", asset: asset.Binary, expectedErr: asset.ErrNotSupported},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.UpdateAccountBalances(t.Context(), tt.asset)
			require.ErrorIs(t, err, tt.expectedErr, "UpdateAccountBalances must return the expected branch error")
		})
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder must return unsupported")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdrawCryptoRequest)
	require.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero)
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestAuthenticateWebsocket(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	err := h.AuthenticateWebsocket(t.Context())
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "AuthenticateWebsocket must require credentials")
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	err := h.ValidateAPICredentials(t.Context(), asset.Spot)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "ValidateAPICredentials must report authentication errors")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.USDT, "", "uSdTeRc20")
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestGetDepositAddressAuthentication(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	_, err := h.GetDepositAddress(t.Context(), currency.USDT, "", "uSdTeRc20")
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "GetDepositAddress must require credentials")
}

func TestGetOrderInfoValidation(t *testing.T) {
	t.Parallel()

	_, err := e.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must reject empty pairs")

	_, err = e.GetOrderInfo(t.Context(), "1", btcusdtPair, asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject unsupported assets")

	_, err = e.GetOrderInfo(t.Context(), "invalid", btcusdtPair, asset.Spot)
	require.Error(t, err, "GetOrderInfo must reject non-numeric spot order IDs")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name  string
		pair  currency.Pair
		asset asset.Item
	}{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures},
		{name: "futures", pair: btccwPair, asset: asset.Futures},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.GetOrderInfo(t.Context(), "1", tt.pair, tt.asset)
			require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "GetOrderInfo must require credentials")
		})
	}
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		interval kline.Interval
		output   string
	}{
		{kline.OneMin, "1min"},
		{kline.FourHour, "4hour"},
		{kline.OneDay, "1day"},
		{kline.OneWeek, "1week"},
		{kline.OneMonth, "1mon"},
		{kline.OneYear, "1year"},
		{kline.TwoWeek, ""},
	} {
		assert.Equalf(t, tt.output, e.FormatExchangeKlineInterval(tt.interval), "FormatExchangeKlineInterval should return correctly for %s", tt.output)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), btcusdtPair, asset.Spot)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btccwPair, asset.Futures)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btcusdtPair, asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject unsupported assets")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), btcusdtPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetOrderHistoryValidation(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.ErrorContains(t, err, "currency must be supplied", "GetOrderHistory must require a currency pair for spot")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	testCases := []struct {
		name        string
		pair        currency.Pair
		asset       asset.Item
		expectedErr error
	}{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: asset.ErrNotSupported},
		{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
		{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
				Type:      order.AnyType,
				Pairs:     []currency.Pair{tt.pair},
				AssetType: tt.asset,
				Side:      order.AnySide,
				StartTime: time.Now().AddDate(0, 0, -1),
				EndTime:   time.Now(),
			})
			require.ErrorIs(t, err, tt.expectedErr, "GetOrderHistory must return the expected branch error")
		})
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	c, err := e.GetAvailableTransferChains(t.Context(), currency.USDT)
	require.NoError(t, err)
	require.Greater(t, len(c), 2, "Must get more than 2 chains")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err)
}

func TestGetWithdrawalsHistoryUnsupportedAsset(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetWithdrawalsHistory must reject unsupported assets")
}

func TestCompatibleVars(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		side           string
		orderPriceType string
		status         int64
		expectedSide   order.Side
		expectedType   order.Type
		expectedStatus order.Status
		expectedTIF    order.TimeInForce
		wantErr        bool
	}{
		{name: "buy limit active", side: "buy", orderPriceType: "limit", status: 3, expectedSide: order.Buy, expectedType: order.Limit, expectedStatus: order.Active},
		{name: "sell market filled", side: "sell", orderPriceType: "opponent", status: 6, expectedSide: order.Sell, expectedType: order.Market, expectedStatus: order.Filled},
		{name: "post only cancelled", side: "buy", orderPriceType: "post_only", status: 7, expectedSide: order.Buy, expectedType: order.Limit, expectedStatus: order.Cancelled, expectedTIF: order.PostOnly},
		{name: "invalid side", side: "hold", orderPriceType: "limit", status: 3, wantErr: true},
		{name: "invalid order price type", side: "buy", orderPriceType: "stop", status: 3, wantErr: true},
		{name: "invalid status", side: "buy", orderPriceType: "limit", status: 99, wantErr: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := compatibleVars(tt.side, tt.orderPriceType, tt.status)
			if tt.wantErr {
				require.Error(t, err, "compatibleVars must reject invalid input")
				return
			}
			require.NoError(t, err, "compatibleVars must not error")
			assert.Equal(t, tt.expectedSide, resp.Side, "side should match")
			assert.Equal(t, tt.expectedType, resp.OrderType, "order type should match")
			assert.Equal(t, tt.expectedStatus, resp.Status, "status should match")
			assert.Equal(t, tt.expectedTIF, resp.TimeInForce, "time in force should match")
		})
	}
}

func TestSetOrderSideStatusAndType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		state          string
		requestType    string
		expectedSide   order.Side
		expectedType   order.Type
		expectedStatus order.Status
	}{
		{
			name:           "buy market",
			state:          "filled",
			requestType:    string(SpotNewOrderRequestTypeBuyMarket),
			expectedSide:   order.Buy,
			expectedType:   order.Market,
			expectedStatus: order.Filled,
		},
		{
			name:           "sell market",
			state:          "partial-filled",
			requestType:    string(SpotNewOrderRequestTypeSellMarket),
			expectedSide:   order.Sell,
			expectedType:   order.Market,
			expectedStatus: order.PartiallyFilled,
		},
		{
			name:           "buy limit",
			state:          "submitted",
			requestType:    string(SpotNewOrderRequestTypeBuyLimit),
			expectedSide:   order.Buy,
			expectedType:   order.Limit,
			expectedStatus: order.New,
		},
		{
			name:           "sell limit",
			state:          "canceled",
			requestType:    string(SpotNewOrderRequestTypeSellLimit),
			expectedSide:   order.Sell,
			expectedType:   order.Limit,
			expectedStatus: order.Cancelled,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			detail := &order.Detail{Exchange: e.Name}
			setOrderSideStatusAndType(tt.state, tt.requestType, detail)
			assert.Equal(t, tt.expectedSide, detail.Side, "side should match")
			assert.Equal(t, tt.expectedType, detail.Type, "order type should match")
			assert.Equal(t, tt.expectedStatus, detail.Status, "status should match")
		})
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	resp, err := e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")
	updatePairsOnce(t, e)
	err := e.CurrencyPairs.EnablePair(asset.USDTMarginedFutures, currency.NewBTCUSDT())
	if err != nil {
		require.ErrorIs(t, err, currency.ErrPairAlreadyEnabled)
	}

	resp, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.CoinMarginedFutures,
		Pair:                 currency.NewBTCUSD(),
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	err = e.CurrencyPairs.EnablePair(asset.CoinMarginedFutures, currency.NewBTCUSD())
	require.ErrorIs(t, err, currency.ErrPairAlreadyEnabled)

	resp, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.CoinMarginedFutures,
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Binary, currency.NewBTCUSDT())
	require.NoError(t, err)
	assert.False(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewBTCUSDT())
	require.NoError(t, err)
	assert.True(t, is)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		err := e.UpdateTickers(t.Context(), a)
		require.NoErrorf(t, err, "asset %s", a)
		avail, err := e.GetAvailablePairs(a)
		require.NoError(t, err)
		for _, p := range avail {
			_, err = ticker.GetTicker(e.Name, p, a)
			assert.NoErrorf(t, err, "Could not get ticker for %s %s", a, p)
		}
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)

	resp, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USD.Item,
		Asset: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  btccwPair.Base.Item,
		Quote: btccwPair.Quote.Item,
		Asset: asset.Futures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.CurrencyPairs.GetPairs(a, false)
			require.NoError(t, err, "GetPairs must not error")
			require.NotEmpty(t, pairs, "GetPairs must return pairs")
			for _, p := range pairs {
				l, err := e.GetOrderExecutionLimits(a, p)
				require.NoError(t, err, "GetOrderExecutionLimits must not error")
				assert.Positive(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive")
				assert.Positive(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive")
				assert.Positive(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive")
			}
		})
	}
	t.Run("unsupported asset", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Binary), asset.ErrNotSupported)
	})
}

func TestBootstrap(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")

	c, err := e.Bootstrap(t.Context())
	require.NoError(t, err)
	assert.True(t, c, "Bootstrap should return true to continue")

	e.futureContractCodes = nil
	e.Features.Enabled.AutoPairUpdates = false
	_, err = e.Bootstrap(t.Context())
	require.NoError(t, err)
	require.NotNil(t, e.futureContractCodes)
}

var (
	updatePairsMutex         sync.Mutex
	futureContractCodesCache map[string]currency.Code
)

// updatePairsOnce updates the pairs once, and ensures a future dated contract is enabled
