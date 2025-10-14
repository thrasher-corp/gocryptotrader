package poloniex

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply your own APIKEYS here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var (
	e                                     *Exchange
	spotTradablePair, futuresTradablePair currency.Pair
)

func (e *Exchange) setAPICredential(apiKey, apiSecret string) {
	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.LTC.String(),
			currency.BTC.String(),
			"-"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	result, err := e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(e) || e.SkipAuthCheck {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		assert.NotNil(t, result)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()

	if sharedtestvalues.AreAPICredentialsSet(e) || mockTests {
		// CryptocurrencyTradeFee Basic
		_, err := e.GetFee(generateContext(), feeBuilder)
		assert.NoError(t, err)

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		_, err = e.GetFee(generateContext(), feeBuilder)
		assert.NoError(t, err)

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := e.GetFee(generateContext(), feeBuilder); err != nil {
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	result, err := e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	result, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	result, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	result, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	result, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)
	assert.NotNil(t, result)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// CryptocurrencyTradeFee Basic
	feeBuilder = setFeeBuilder()
	result, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	result, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	result, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetActiveOrders(generateContext(), &order.MultiOrderRequest{
		AssetType: asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetActiveOrders(generateContext(), &order.MultiOrderRequest{
		AssetType: asset.Futures,
		Side:      order.Buy,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Type:      order.Liquidation,
		AssetType: asset.Spot,
		Side:      order.Buy,
	})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrderHistory(generateContext(), &order.MultiOrderRequest{
		Type:      order.Limit,
		AssetType: asset.Spot,
		Side:      order.Buy,
	})
	assert.NoErrorf(t, err, "error: %v", err)
	assert.NotNil(t, result)

	result, err = e.GetOrderHistory(generateContext(), &order.MultiOrderRequest{
		Type:      order.Limit,
		AssetType: asset.Spot,
		Side:      order.Sell,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOrderHistory(generateContext(), &order.MultiOrderRequest{
		Type:      order.Limit,
		AssetType: asset.Futures,
		Side:      order.Buy,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), &order.Submit{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := &order.Submit{AssetType: asset.Futures, TimeInForce: order.GoodTillCrossing}
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	arg.Pair = futuresTradablePair
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce)
	arg.TimeInForce = order.GoodTillCancel
	arg.AssetType = asset.Options
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	arg.AssetType = asset.Spot
	arg.Type = order.Liquidation
	arg.Pair = spotTradablePair
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitOrder(generateContext(), &order.Submit{
		Exchange:  e.Name,
		Pair:      spotTradablePair,
		Side:      order.Buy,
		Type:      order.Market,
		Price:     10,
		Amount:    10000000,
		AssetType: asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.SubmitOrder(generateContext(), &order.Submit{
		Exchange:     e.Name,
		Pair:         spotTradablePair,
		Side:         order.Buy,
		Type:         order.StopLimit,
		TriggerPrice: 11,
		Price:        10,
		Amount:       10000000,
		ClientID:     "hi",
		AssetType:    asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.SubmitOrder(generateContext(), &order.Submit{
		Exchange:     e.Name,
		Pair:         spotTradablePair,
		Side:         order.Buy,
		Type:         order.Market,
		TriggerPrice: 11,
		Price:        10,
		Amount:       10000000,
		ClientID:     "hi",
		AssetType:    asset.Futures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.SubmitOrder(generateContext(), &order.Submit{
		Exchange:     e.Name,
		Pair:         futuresTradablePair,
		Side:         order.Buy,
		Type:         order.TrailingStop,
		TriggerPrice: 11,
		Price:        10,
		Amount:       10000000,
		AssetType:    asset.Futures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	arg := &order.Submit{AssetType: asset.Futures, TimeInForce: order.GoodTillCrossing}
	_, err := e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Pair = futuresTradablePair
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce)

	arg.TimeInForce = order.GoodTillCancel
	arg.AssetType = asset.Options
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	arg.Type = order.Liquidation
	arg.Pair = spotTradablePair
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	if !mockTests && !e.Websocket.IsEnabled() && !e.Websocket.IsConnected() {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWs(t, e)
	} else {
		t.SkipNow()
	}
	result, err := e.WebsocketSubmitOrder(generateContext(), &order.Submit{
		Exchange:  e.Name,
		Pair:      spotTradablePair,
		Side:      order.Buy,
		Type:      order.Market,
		Price:     10,
		Amount:    10000000,
		AssetType: asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.WebsocketSubmitOrder(generateContext(), &order.Submit{
		Exchange:     e.Name,
		Pair:         spotTradablePair,
		Side:         order.Buy,
		Type:         order.StopLimit,
		TriggerPrice: 11,
		Price:        10,
		Amount:       10000000,
		ClientID:     "hi",
		AssetType:    asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.WebsocketSubmitOrder(generateContext(), &order.Submit{
		Exchange:     e.Name,
		Pair:         spotTradablePair,
		Side:         order.Buy,
		Type:         order.Market,
		TriggerPrice: 11,
		Price:        10,
		Amount:       10000000,
		ClientID:     "hi",
		AssetType:    asset.Futures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	err := e.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: "", ClientOrderID: ""})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests && !e.Websocket.IsEnabled() && !e.Websocket.IsConnected() {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWs(t, e)
	} else {
		t.SkipNow()
	}
	err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: "2312", ClientOrderID: "23123121231"})
	assert.NoError(t, err)
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	arg := &order.Cancel{
		AccountID: "1",
	}
	err := e.CancelOrder(t.Context(), arg)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = "123"
	err = e.CancelOrder(t.Context(), arg)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.AssetType = asset.Spot
	err = e.CancelOrder(generateContext(), arg)
	assert.NoError(t, err)

	arg.Type = order.StopLimit
	err = e.CancelOrder(generateContext(), arg)
	assert.NoError(t, err)

	err = e.CancelOrder(generateContext(), &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Futures,
	})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	err = e.CancelOrder(generateContext(), &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Futures,
		Pair:      futuresTradablePair,
	})
	assert.NoError(t, err)
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      spotTradablePair,
		AssetType: asset.Spot,
	}
	arg.Type = order.Stop
	result, err := e.CancelAllOrders(generateContext(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.Limit
	result, err = e.CancelAllOrders(generateContext(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)

	arg.Pair = futuresTradablePair
	result, err = e.CancelAllOrders(generateContext(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.StopLimit
	result, err = e.CancelAllOrders(generateContext(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	arg := &order.Modify{
		OrderID: "1337",
		Price:   1337,
	}
	_, err = e.ModifyOrder(t.Context(), arg)
	assert.ErrorIs(t, err, order.ErrPairIsEmpty)

	arg.Pair = spotTradablePair
	_, err = e.ModifyOrder(t.Context(), arg)
	assert.ErrorIs(t, err, order.ErrAssetNotSet)

	arg.AssetType = asset.Futures
	_, err = e.ModifyOrder(t.Context(), arg)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	_, err = e.ModifyOrder(t.Context(), arg)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.Type = order.Limit
	arg.TimeInForce = order.GoodTillTime
	_, err = e.ModifyOrder(t.Context(), arg)
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.TimeInForce = order.GoodTillCancel
	result, err := e.ModifyOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	assert.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil)

	arg := &withdraw.Request{
		Crypto: withdraw.CryptoRequest{
			FeeAmount: 0,
		},
	}
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1000
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.LTC
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidWithdrawalChain)

	arg.Crypto.Chain = "ERP"
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorIs(t, err, errAddressRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	withdrawCryptoRequest := withdraw.Request{
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
			Chain:   "ERP",
		},
		Amount:   1,
		Currency: currency.BTC,
	}
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdrawCryptoRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateAccountInfo(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UpdateAccountInfo(generateContext(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateAccountInfo(generateContext(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	var withdrawFiatRequest withdraw.Request
	_, err := e.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	var start, end time.Time
	if mockTests {
		start = time.UnixMilli(1744183959258)
		end = time.UnixMilli(1744191159258)
	} else {
		start = time.Now().Add(-time.Hour * 2)
		end = time.Now()
	}
	result, err := e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, start.UTC(), end.UTC())
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.FiveMin, start.UTC(), end.UTC())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1744103854944)
	end := time.UnixMilli(1744190254944)
	if !mockTests {
		start = time.Now().Add(-time.Hour * 24)
		end = time.Now()
	}
	result, err := e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.OneHour, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, kline.FiveMin, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	result, err := e.GetRecentTrades(t.Context(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetRecentTrades(t.Context(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	tStart := time.Date(2020, 6, 6, 0, 0, 0, 0, time.UTC)
	tEnd := time.Date(2020, 6, 6, 1, 0, 0, 0, time.UTC)
	if !mockTests {
		tmNow := time.Now()
		tStart = time.Date(tmNow.Year(), tmNow.Month()-3, 6, 0, 0, 0, 0, time.UTC)
		tEnd = time.Date(tmNow.Year(), tmNow.Month()-3, 7, 0, 0, 0, 0, time.UTC)
	}
	_, err := e.GetHistoricTrades(t.Context(),
		spotTradablePair, asset.Spot, tStart, tEnd)
	require.NoError(t, err)

	_, err = e.GetHistoricTrades(t.Context(),
		futuresTradablePair, asset.Futures, tStart, tEnd)
	require.NoError(t, err)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), spotTradablePair, asset.Binary)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := e.UpdateTickers(t.Context(), asset.Options)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.UpdateTickers(t.Context(), asset.Spot)
	assert.NoError(t, err)

	err = e.UpdateTickers(t.Context(), asset.Futures)
	assert.NoError(t, err)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{})
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Options}})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, OrderID: "1233", Pair: futuresTradablePair}, {AssetType: asset.Spot, Pair: futuresTradablePair}})
	require.ErrorIs(t, err, asset.ErrInvalidAsset)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, OrderID: "1233"}, {AssetType: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{Pair: futuresTradablePair, AssetType: asset.Futures, OrderID: "1233"}, {OrderID: "1233", AssetType: asset.Futures, Pair: spotTradablePair}})
	require.ErrorIs(t, err, currency.ErrPairNotFound)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Spot, OrderID: "1233", Type: order.Liquidation}, {AssetType: asset.Spot, OrderID: "123444", Type: order.StopLimit}})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelBatchOrders(generateContext(), []order.Cancel{{
		Pair:      futuresTradablePair,
		AssetType: asset.Futures,
		OrderID:   "1233",
		Type:      order.StopLimit,
	}, {
		Pair:      futuresTradablePair,
		AssetType: asset.Futures,
		OrderID:   "123444",
		Type:      order.StopLimit,
	}})
	require.NoError(t, err)

	result, err := e.CancelBatchOrders(generateContext(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      spotTradablePair,
		},
		{
			OrderID:   "134",
			AssetType: asset.Spot,
			Pair:      currency.NewBTCUSD(),
		},
		{
			OrderID:   "234",
			AssetType: asset.Spot,
			Pair:      currency.NewBTCUSD(),
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotZero(t, st)

	st, err = e.GetServerTime(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotZero(t, st)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Pair: currency.NewPair(currency.BTC, currency.LTC)})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.Spot,
		Pair:                 spotTradablePair,
		IncludePredictedRate: false,
	})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	result, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.Futures,
		Pair:                 futuresTradablePair,
		IncludePredictedRate: false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Spot, spotTradablePair)
	require.NoError(t, err)
	require.False(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.Futures, futuresTradablePair)
	require.NoError(t, err)
	assert.True(t, is)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbolInformation(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetSymbolInformation(t.Context(), spotTradablePair)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestGetExecutionLimits(t *testing.T) {
	t.Parallel()
	result, err := e.GetAllSymbolInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetCurrencies(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetCurrency(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTimestamp(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemTimestamp(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketPrices(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarketPrices(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetMarketPrice(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrices(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarkPrices(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetMarkPrice(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarkPriceComponents(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceComponents(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetMarkPriceComponents(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetOrderbook(t.Context(), spotTradablePair, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
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

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlesticks(t.Context(), currency.EMPTYPAIR, kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetCandlesticks(t.Context(), spotTradablePair, kline.HundredMilliseconds, time.Now().Add(-time.Hour*48), time.Time{}, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetCandlesticks(t.Context(), spotTradablePair, kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetTrades(t.Context(), spotTradablePair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	result, err := e.GetTickers(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetTicker(t.Context(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralInfos(t *testing.T) {
	t.Parallel()
	_, err := e.GetCollateralInfo(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetCollateralInfo(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralsInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetCollateralsInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetCollateralInfo(t.Context(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowRateInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetBorrowRateInfo(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountInformation(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllBalances(generateContext(), "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.GetAllBalances(generateContext(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllBalancesByID(t.Context(), "", "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllBalancesByID(generateContext(), "329455537441832960", "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.GetAllBalancesByID(generateContext(), "329455537441832960", "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllAccountActivities(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetAllAccountActivities(generateContext(), time.Time{}, time.Time{}, 0, 0, 0, "", currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestAccountsTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.AccountsTransfer(t.Context(), &AccountTransferRequest{Amount: 1232.221})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.AccountsTransfer(t.Context(), &AccountTransferRequest{
		Currency: currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = e.AccountsTransfer(t.Context(), &AccountTransferRequest{
		Amount:      1,
		Currency:    currency.BTC,
		FromAccount: "219961623421431808",
	})
	require.ErrorIs(t, err, errAddressRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AccountsTransfer(generateContext(), &AccountTransferRequest{
		Amount:      1,
		Currency:    currency.BTC,
		FromAccount: "329455537441832960",
		ToAccount:   "329455537441832960",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountTransferRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetAccountsTransferRecordsByTransferID(generateContext(), time.Time{}, time.Time{}, "", currency.BTC, 0, 0)
	require.NoError(t, err)
}

func TestGetAccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountsTransferRecordByTransferID(generateContext(), "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetAccountsTransferRecordByTransferID(generateContext(), "329455537441832960")
	require.NoError(t, err)
}

func TestGetFeeInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFeeInfo(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetInterestHistory(generateContext(), time.Time{}, time.Time{}, "", 0, 0)
	require.NoError(t, err)
}

func TestGetSubAccountInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountInformation(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountBalances(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountBalance(t.Context(), "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetSubAccountBalance(generateContext(), "2d45301d-5f08-4a2b-a763-f9199778d854")
	require.NoError(t, err)
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.SubAccountTransfer(t.Context(), &SubAccountTransferRequest{Amount: 12.34})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SubAccountTransfer(t.Context(), &SubAccountTransferRequest{
		Currency: currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = e.SubAccountTransfer(t.Context(), &SubAccountTransferRequest{
		Currency: currency.BTC,
		Amount:   1,
	})
	require.ErrorIs(t, err, errAccountIDRequired)
	_, err = e.SubAccountTransfer(t.Context(), &SubAccountTransferRequest{
		Currency:      currency.BTC,
		Amount:        1,
		FromAccountID: "1234568",
		ToAccountID:   "1234567",
	})
	require.ErrorIs(t, err, errAccountTypeRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubAccountTransfer(generateContext(), &SubAccountTransferRequest{
		Currency:        currency.BTC,
		Amount:          1,
		FromAccountID:   "329455537441832960",
		ToAccountID:     "329455537441832961",
		FromAccountType: "SPOT",
		ToAccountType:   "SPOT",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountTransferRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetSubAccountTransferRecords(generateContext(), &SubAccountTransferRecordRequest{Currency: currency.BTC})
	require.NoError(t, err)
}

func TestGetSubAccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountTransferRecord(t.Context(), "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetSubAccountTransferRecord(generateContext(), "329455537441832960")
	require.NoError(t, err)
}

func TestGetDepositAddresses(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetDepositAddresses(generateContext(), currency.LTC)
	require.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(generateContext(), "", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrderInfo(generateContext(), "1234", spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOrderInfo(generateContext(), "12345", futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDepositAddress(generateContext(), currency.BTC, "", "TON")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWalletActivity(t *testing.T) {
	t.Parallel()
	var start, end time.Time
	if mockTests {
		start = time.UnixMilli(1743575750138)
		end = time.UnixMilli(1743582950138)
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		start = time.Now().Add(-time.Hour * 2)
		end = time.Now()
	}
	result, err := e.WalletActivity(generateContext(), start, end, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.NewCurrencyDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.NewCurrencyDepositAddress(generateContext(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCurrency(t.Context(), &WithdrawCurrencyRequest{Coin: currency.BTC})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.WithdrawCurrency(t.Context(), &WithdrawCurrencyRequest{Coin: currency.BTC, Amount: 1})
	require.ErrorIs(t, err, errInvalidWithdrawalChain)
	_, err = e.WithdrawCurrency(t.Context(), &WithdrawCurrencyRequest{Coin: currency.BTC, Amount: 1, Network: "BTC"})
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawCurrency(t.Context(), &WithdrawCurrencyRequest{Network: "BTC", Coin: currency.BTC, Amount: 1, Address: "0xbb8d0d7c346daecc2380dabaa91f3ccf8ae232fb4"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountMarginInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountMarginInfo(generateContext(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetBorrowStatus(generateContext(), currency.USDT)
	require.NoError(t, err)
}

func TestMaximumBuySellAmount(t *testing.T) {
	t.Parallel()
	_, err := e.MaximumBuySellAmount(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.MaximumBuySellAmount(generateContext(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceOrder(t.Context(), &PlaceOrderRequest{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{Symbol: spotTradablePair})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{Symbol: spotTradablePair, Side: order.Sell.Lower()})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceOrder(t.Context(), &PlaceOrderRequest{
		Symbol:        spotTradablePair,
		Side:          order.Buy.String(),
		Type:          order.Market.String(),
		Amount:        100,
		Price:         40000.50000,
		TimeInForce:   "GTC",
		ClientOrderID: "1234Abc",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceBatchOrders(t.Context(), []PlaceOrderRequest{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.PlaceBatchOrders(t.Context(), []PlaceOrderRequest{
		{
			Symbol: spotTradablePair,
		},
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	var pair currency.Pair
	getPairFromString := func(pairString string) currency.Pair {
		pair, err = currency.NewPairFromString(pairString)
		if err != nil {
			return currency.EMPTYPAIR
		}
		return pair
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceBatchOrders(t.Context(), []PlaceOrderRequest{
		{
			Symbol:        pair,
			Side:          order.Buy.String(),
			Type:          order.Market.String(),
			Quantity:      1,
			Price:         40000.50000,
			TimeInForce:   "GTC",
			ClientOrderID: "1234Abc",
		},
		{
			Symbol: getPairFromString("BTC_USDT"),
			Amount: 100,
			Side:   "BUY",
		},
		{
			Symbol:        getPairFromString("BTC_USDT"),
			Type:          "LIMIT",
			Quantity:      100,
			Side:          "BUY",
			Price:         40000.50000,
			TimeInForce:   "IOC",
			ClientOrderID: "1234Abc",
		},
		{
			Symbol: getPairFromString("ETH_USDT"),
			Amount: 1000,
			Side:   "BUY",
		},
		{
			Symbol:        getPairFromString("TRX_USDT"),
			Type:          "LIMIT",
			Quantity:      15000,
			Side:          "SELL",
			Price:         0.0623423423,
			TimeInForce:   "IOC",
			ClientOrderID: "456Xyz",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelReplaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelReplaceOrder(t.Context(), &CancelReplaceOrderRequest{TimeInForce: "GTC"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelReplaceOrder(t.Context(), &CancelReplaceOrderRequest{
		orderID:       "29772698821328896",
		ClientOrderID: "1234Abc",
		Price:         18000,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetOpenOrders(generateContext(), spotTradablePair, "", "NEXT", "", 10)
	require.NoError(t, err)

	_, err = e.GetOpenOrders(generateContext(), spotTradablePair, "SELL", "NEXT", "24993088082542592", 10)
	require.NoError(t, err)
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrder(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrder(generateContext(), "12345536545645", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrderByID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOrderByID(t.Context(), "12345536545645")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleOrdersByIDs(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrdersByIDs(t.Context(), nil, nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOrdersByIDs(t.Context(), []string{"1234"}, []string{"5678"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err := e.CancelAllTradeOrders(t.Context(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	require.NoError(t, err)
}

func TestKillSwitch(t *testing.T) {
	t.Parallel()
	_, err := e.KillSwitch(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidTimeout)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.KillSwitch(generateContext(), time.Second*30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDisableKillSwitch(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DisableKillSwitch(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKillSwitchStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetKillSwitchStatus(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSmartOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSmartOrder(t.Context(), &SmartOrderRequestRequest{Side: "BUY"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.CreateSmartOrder(t.Context(), &SmartOrderRequestRequest{Symbol: spotTradablePair})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSmartOrder(generateContext(), &SmartOrderRequestRequest{
		Symbol:        spotTradablePair,
		Type:          "STOP_LIMIT",
		Price:         40000.50000,
		ClientOrderID: "1234Abc",
		Side:          "BUY",
		TimeInForce:   "GTC",
		Quantity:      100,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equalf(t, int64(200), result.Code, "CreateSmartOrder error with code: %d message: %s", result.Code, result.Message)
}

func TestCancelReplaceSmartOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelReplaceSmartOrder(t.Context(), &CancelReplaceSmartOrderRequest{Price: 18000})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelReplaceSmartOrder(t.Context(), &CancelReplaceSmartOrderRequest{
		orderID:       "29772698821328896",
		ClientOrderID: "1234Abc",
		Price:         18000,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmartOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetSmartOpenOrders(generateContext(), 10)
	require.NoError(t, err)
}

func TestGetSmartOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetSmartOrderDetail(generateContext(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSmartOrderDetail(generateContext(), "123313413", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSmartOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSmartOrderByID(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSmartOrderByID(t.Context(), "123313413", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleSmartOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleSmartOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.CancelMultipleSmartOrders(t.Context(), &CancelOrdersRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleSmartOrders(t.Context(), &CancelOrdersRequest{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllSmartOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllSmartOrders(t.Context(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrdersHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetOrdersHistory(generateContext(), &OrdersHistoryRequest{Symbol: spotTradablePair, AccountType: "SPOT", OrderType: "", Side: "", Direction: "", States: "", From: 0, Limit: 10, StartTime: time.Time{}, EndTime: time.Time{}, HideCancel: false})
	require.NoError(t, err)
}

func TestGetSmartOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetSmartOrderHistory(generateContext(), &OrdersHistoryRequest{Symbol: spotTradablePair, AccountType: "SPOT", OrderType: "", Side: "", Direction: "", States: "", From: 0, Limit: 10, StartTime: time.Time{}, EndTime: time.Time{}, HideCancel: false})
	require.NoError(t, err)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetTradeHistory(generateContext(), currency.Pairs{spotTradablePair}, "", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
}

func TestGetTradeOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradesByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetTradesByOrderID(generateContext(), "13123242323")
	require.NoError(t, err)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	subs, err := e.generateSubscriptions()
	require.NoError(t, err)
	exp := []string{"candles_minute_5", "trades", "ticker", "book_lv2"}

	creds, err := e.GetCredentials(t.Context())
	if assert.True(t, err == nil || errors.Is(err, exchange.ErrAuthenticationSupportNotEnabled)) {
		if !creds.IsEmpty() {
			println("\n\nerr\n\n")
			exp = append(exp, "orders", "balances")
		}
	}

	got := make([]string, len(subs))
	for i := range subs {
		got[i] = subs[i].QualifiedChannel
	}
	assert.Equal(t, exp, got)
}

func TestHandleSubscription(t *testing.T) {
	t.Parallel()
	subs, err := e.generateSubscriptions()
	require.NoError(t, err)
	require.NotEmpty(t, subs)

	for _, s := range subs {
		if s.Authenticated {
			continue // Skip authenticated channels
		}
		t.Run(s.QualifiedChannel, func(t *testing.T) {
			t.Parallel()
			payload, err := e.handleSubscription("subscribe", s)
			require.NoError(t, err, "handleSubscription must not error")
			assert.NotEmpty(t, payload.Channel, "Channel should not be empty")
		})
	}
}

var pushMessages = map[string]string{
	"AccountBalance": `{ "channel": "balances", "data": [{ "changeTime": 1657312008411, "accountId": "1234", "accountType": "SPOT", "eventType": "place_order", "available": "9999999983.668", "currency": "BTC", "id": 60018450912695040, "userId": 12345, "hold": "16.332", "ts": 1657312008443 }] }`,
	"Orders":         `{ "channel": "orders", "data": [ { "symbol": "BTC_USDC", "type": "LIMIT", "quantity": "1", "orderId": "32471407854219264", "tradeFee": "0", "clientOrderId": "", "accountType": "SPOT", "feeCurrency": "", "eventType": "place", "source": "API", "side": "BUY", "filledQuantity": "0", "filledAmount": "0", "matchRole": "MAKER", "state": "NEW", "tradeTime": 0, "tradeAmount": "0", "orderAmount": "0", "createTime": 1648708186922, "price": "47112.1", "tradeQty": "0", "tradePrice": "0", "tradeId": "0", "ts": 1648708187469 } ] }`,
	"Candles":        `{"channel":"candles_minute_5","data":[{"symbol":"BTC_USDT","open":"25143.19","high":"25148.58","low":"25138.76","close":"25144.55","quantity":"0.860454","amount":"21635.20983974","tradeCount":20,"startTime":1694469000000,"closeTime":1694469299999,"ts":1694469049867}]}`,
	"Books":          `{"channel":"book","data":[{"symbol":"BTC_USDC","createTime":1694469187686,"asks":[["25157.24","0.444294"],["25157.25","0.024357"],["25157.26","0.003204"],["25163.39","0.039476"],["25163.4","0.110047"]],"bids":[["25148.8","0.00692"],["25148.61","0.021581"],["25148.6","0.034504"],["25148.59","0.065405"],["25145.52","0.79537"]],"id":598273384,"ts":1694469187733}]}`,
	"Tickers":        `{"channel":"ticker","data":[{"symbol":"BTC_USDC","startTime":1694382780000,"open":"25866.3","high":"26008.47","low":"24923.65","close":"25153.02","quantity":"1626.444884","amount":"41496808.63699303","tradeCount":37124,"dailyChange":"-0.0276","markPrice":"25154.9","closeTime":1694469183664,"ts":1694469187081}]}`,
	"Trades":         `{"channel":"trades","data":[{"symbol":"BTC_USDC","amount":"52.821342","quantity":"0.0021","takerSide":"sell","createTime":1694469183664,"price":"25153.02","id":"71076055","ts":1694469183673}]}`,
	"Currencies":     `{"channel":"currencies","data":[[{"currency":"BTC","id":28,"name":"Bitcoin","description":"BTC Clone","type":"address","withdrawalFee":"0.0008","minConf":2,"depositAddress":null,"blockchain":"BTC","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["BTCTRON"]},{"currency":"XRP","id":243,"name":"XRP","description":"Payment ID","type":"address-payment-id","withdrawalFee":"0.2","minConf":2,"depositAddress":"rwU8rAiE2eyEPz3sikfbHuqCuiAtdXqa2v","blockchain":"XRP","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":false,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":[]},{"currency":"ETH","id":267,"name":"Ethereum","description":"Sweep to Main Account","type":"address","withdrawalFee":"0.00197556","minConf":64,"depositAddress":null,"blockchain":"ETH","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["ETHTRON"]},{"currency":"USDT","id":214,"name":"Tether USD","description":"Sweep to Main Account","type":"address","withdrawalFee":"0","minConf":2,"depositAddress":null,"blockchain":"OMNI","delisted":false,"tradingState":"NORMAL","walletState":"DISABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["USDTETH","USDTTRON"]},{"currency":"DOGE","id":59,"name":"Dogecoin","description":"BTC Clone","type":"address","withdrawalFee":"20","minConf":6,"depositAddress":null,"blockchain":"DOGE","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["DOGETRON"]},{"currency":"LTC","id":125,"name":"Litecoin","description":"BTC Clone","type":"address","withdrawalFee":"0.001","minConf":4,"depositAddress":null,"blockchain":"LTC","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["LTCTRON"]},{"currency":"DASH","id":60,"name":"Dash","description":"BTC Clone","type":"address","withdrawalFee":"0.01","minConf":20,"depositAddress":null,"blockchain":"DASH","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":false,"isChildChain":false,"supportCollateral":false,"supportBorrow":false,"childChains":[]}]],"action":"snapshot"}`,
	"Symbols":        `{"channel":"symbols","data":[[{"symbol":"BTC_USDC","baseCurrencyName":"BTC","quoteCurrencyName":"USDT","displayName":"BTC/USDT","state":"NORMAL","visibleStartTime":1659018819512,"tradableStartTime":1659018819512,"crossMargin":{"supportCrossMargin":true,"maxLeverage":"3"},"symbolTradeLimit":{"symbol":"BTC_USDT","priceScale":2,"quantityScale":6,"amountScale":2,"minQuantity":"0.000001","minAmount":"1","highestBid":"0","lowestAsk":"0"}}]],"action":"snapshot"}`,
}

func TestWsPushData(t *testing.T) {
	t.Parallel()
	for key, value := range pushMessages {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			err := e.wsHandleData(t.Context(), e.Websocket.Conn, []byte(value))
			assert.NoError(t, err)
		})
	}
	// Since running test parallelly shuffles the order of execution
	// We run book_lv2 data handling, ensuring the snapshot is processed before the update as follows
	err := e.wsHandleData(t.Context(), e.Websocket.Conn, []byte(`{"channel":"book_lv2","data":[{"symbol":"BTC_USDC","createTime":1694469187745,"asks":[],"bids":[["25148.81","0.02158"],["25088.11","0"]],"lastId":598273385,"id":598273386,"ts":1694469187760}],"action":"snapshot"}`))
	require.NoError(t, err, "book_lv2 snapshot must not error")
	err = e.wsHandleData(t.Context(), e.Websocket.Conn, []byte(`{"channel":"book_lv2","data":[{"symbol":"BTC_USDC","createTime":1694469187745,"asks":[],"bids":[["25148.81","0.02158"],["25088.11","0"]],"lastId":598273385,"id":598273386,"ts":1694469187760}],"action":"update"}`))
	assert.NoError(t, err, "book_lv2 update should not error")
}

func TestWsCreateOrder(t *testing.T) {
	t.Parallel()
	e := new(Exchange) //nolint:govet // Intentional shadow
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	_, err := e.WsCreateOrder(t.Context(), &PlaceOrderRequest{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.WsCreateOrder(t.Context(), &PlaceOrderRequest{
		Symbol: spotTradablePair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	e.setAPICredential(apiKey, apiSecret)
	require.True(t, e.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")

	if !mockTests && !e.Websocket.IsEnabled() && !e.Websocket.IsConnected() {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWs(t, e)
	} else {
		t.SkipNow()
	}
	result, err := e.WsCreateOrder(generateContext(), &PlaceOrderRequest{
		Symbol:        spotTradablePair,
		Side:          order.Buy.String(),
		Type:          order.Market.String(),
		Amount:        1232432,
		Quantity:      100,
		Price:         40000.50000,
		TimeInForce:   "GTC",
		ClientOrderID: "1234Abc",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelMultipleOrdersByIDs(t *testing.T) {
	t.Parallel()
	e := new(Exchange) //nolint:govet // Intentional shadow
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	if !mockTests && !e.Websocket.IsEnabled() && !e.Websocket.IsConnected() {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWs(t, e)
	} else {
		t.SkipNow()
	}
	result, err := e.WsCancelMultipleOrdersByIDs(t.Context(), []string{"1234"}, []string{"5678"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	if !mockTests && !e.Websocket.IsEnabled() && !e.Websocket.IsConnected() {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWs(t, e)
	} else {
		t.SkipNow()
	}
	result, err := e.WsCancelAllTradeOrders(t.Context(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Futures)
	require.NoError(t, err)

	instrument, err := e.GetFuturesProductInfo(t.Context(), futuresTradablePair.String())
	require.NoError(t, err)
	require.NotNil(t, instrument)

	lms, err := e.GetOrderExecutionLimits(asset.Futures, futuresTradablePair)
	require.NoError(t, err)
	require.NotNil(t, lms)
	assert.Equal(t, lms.PriceStepIncrementSize, instrument.TickSize.Float64())
	assert.Equal(t, lms.MinimumBaseAmount, instrument.MinQuantity.Float64())
	assert.Equal(t, lms.MinimumQuoteAmount, instrument.MinSize.Float64())

	// sample test for spot instrument order execution limit

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	require.NoError(t, err)

	spotInstruments, err := e.GetSymbolInformation(t.Context(), spotTradablePair)
	require.NoError(t, err)
	require.NotNil(t, spotInstruments)

	lms, err = e.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
	require.NoError(t, err)
	require.Len(t, spotInstruments, 1)
	require.Equal(t, lms.PriceStepIncrementSize, spotInstruments[0].SymbolTradeLimit.PriceScale)
	require.Equal(t, lms.MinimumBaseAmount, spotInstruments[0].SymbolTradeLimit.MinQuantity.Float64())
	assert.Equal(t, lms.MinimumQuoteAmount, spotInstruments[0].SymbolTradeLimit.MinAmount.Float64())
}

// ---- Futures endpoints ---

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		assert.NoError(t, err)
		assert.NotEmpty(t, pairs)

		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		assert.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

var futuresPushDataMap = map[string]string{
	"Product Info":            `{"channel": "symbol", "data":[{"symbol": "BTC_USDT_PERP", "visibleStartTime": "1584721775000", "tradableStartTime": "1584721775000", "pxScale": "0.01,0.1,1,10,100", "lotSz": 1, "minSz": 1, "ctVal": "0.001", "status": "OPEN", "maxPx": "1000000", "minPx": "0.01", "marketMaxQty": 100000, "limitMaxQty": 100000, "maxQty": "1000000", "minQty": "1", "maxLever": "75", "lever": "20", "ctType": "LINEAR", "alias": "", "bAsset": ".PXBTUSDT", "bCcy": "BTC", "qCcy": "USDT", "sCcy": "USDT", "tSz": "0.01","oDate": "1547435912000", "iM": "0.0133", "mR": "5000", "mM": "0.006" } ], "action": "snapshot"}`,
	"Orderbook":               `{"channel": "book", "data": [ { "asks": [ ["46100", "9284"] ], "bids": [ ["34400.089", "1"] ], "id": 954, "ts": 1718869676586, "s": "BTC_USDT_PERP", "cT": 1718869676555}]}`,
	"Orderbook Lvl2":          `{"channel": "book_lv2", "data": [ { "asks": [["46100", "9284"]], "bids": [["34400.089", "1"]], "lid": 953, "id": 954, "ts": 1718870001418, "s": "BTC_USDT_PERP", "cT": 1718869676555 } ], "action": "snapshot"}`,
	"K-Line Data":             `{"channel": "candles_minute_1", "data": [ ["BTC_USDT_PERP","91883.46","91958.73","91883.46","91958.73","367.68438","4",2,1741243200000,1741243259999,1741243218348]]}`,
	"Tickers":                 `{"channel": "tickers", "data": [ { "s": "BTC_USDT_PERP", "o": "46000", "l": "26829.541", "h": "46100", "c": "46100", "qty": "18736", "amt": "8556118.81658", "tC": 44, "sT": 1718785800000, "cT": 1718872244268, "dC": "0.0022", "bPx": "46000", "bSz": "46000", "aPx": "46100", "aSz": "9279", "ts": 1718872247385}]}`,
	"Trades":                  `{"channel":"trades", "data": [ { "id": 291, "ts": 1718871802553, "s": "BTC_USDT_PERP", "px": "46100", "qty": "1", "amt": "461", "side": "buy", "cT": 1718871802534}]}`,
	"Index Price":             `{"channel": "index_price", "data": [ { "ts": 1719226453000, "s": "BTC_USDT_PERP", "iPx": "34400"}]}`,
	"Mark Price":              `{"channel":"mark_price", "data": [ { "ts": 1719226453000, "s": "BTC_USDT_PERP", "mPx": "34400"}]}`,
	"Mark Price K-line Data":  `{"channel": "mark_price_candles_minute_1", "data": [["BTC_USDT_PERP","57800.17","57815.95","57809.65","57800.17",1725264900000,1725264959999,1725264919140]]}`,
	"Index Price K-line Data": `{"channel": "index_candles_minute_1", "data": [ ["BTC_USDT_PERP","57520.09","57614.9","57520.09","57609.89",1725248760000,1725248819999,1725248813187]]}`,
	"Funding Rate":            `{"channel":"funding_rate", "data": [ { "ts": 1718874420000, "s": "BTC_USDT_PERP", "nFR": "0.000003", "fR": "0.000619", "fT": 1718874000000, "nFT": 1718874900000}]}`,
	"Positions":               `{"channel":"positions", "data": [ { "symbol": "BTC_USDT_PERP", "posSide": "BOTH", "side": "buy", "mgnMode": "CROSS", "openAvgPx": "64999", "qty": "1", "oldQty": "0", "availQty": "1", "lever": 1, "fee": "-0.259996", "adl": "0", "liqPx": "-965678126.114070339063390145", "mgn": "604.99", "im": "604.99", "mm": "3.327445", "upl": "-45", "uplRatio": "-0.0743", "pnl": "0", "markPx": "60499", "mgnRatio": "0.000007195006959591", "state": "NORMAL", "ffee": "0", "fpnl": "0", "cTime": 1723459553457, "uTime": 1725330697439, "ts": 1725330697459}]}`,
	"Orders":                  `{"channel": "orders", "data": [ { "symbol": "BTC_USDT_PERP", "side": "BUY", "type": "LIMIT", "mgnMode": "CROSS", "timeInForce": "GTC", "clOrdId": "polo353849510130364416", "sz": "1", "px": "64999", "reduceOnly": false, "posSide": "BOTH", "ordId": "353849510130364416", "state": "NEW", "source": "WEB", "avgPx": "0", "execQty": "0", "execAmt": "0", "feeCcy": "", "feeAmt": "0", "deductCcy": "", "deductAmt": "0", "actType": "TRADING", "qCcy": "USDT", "cTime": 1725330697421, "uTime": 1725330697421, "ts": 1725330697451}]}`,
	"Trade":                   `{"channel": "trade", "data": [ { "symbol": "BTC_USDT_PERP", "side": "BUY", "ordId": "353849510130364416", "clOrdId": "polo353849510130364416", "role": "TAKER", "trdId": "48", "feeCcy": "USDT", "feeAmt": "0.259996", "deductCcy": "", "deductAmt": "0", "fpx": "64999", "fqty": "1", "uTime": 1725330697559, "ts": 1725330697579}]}`,
	"Account Change":          `{"channel": "account", "data": [ { "state": "NORMAL", "eq": "9604385.495986629521985415", "isoEq": "0", "im": "281.27482", "mm": "65.7758462", "mmr": "0.000006848522086861", "upl": "702.005423182573616772", "availMgn": "9604104.221166629521985415", "details": [ { "ccy": "USDT", "eq": "9604385.495986629521985415", "isoEq": "0", "avail": "9603683.490563446948368643", "upl": "702.005423182573616772", "isoAvail": "0", "isoHold": "0", "isoUpl": "0", "im": "281.27482", "imr": "0.000029286081875569", "mm": "65.7758462", "mmr": "0.000006848522086861", "cTime": 1723431998599, "uTime": 1725329576649 } ], "cTime": 1689326308656, "uTime": 1725329576649, "ts": 1725329576659}]}`,
}

func TestWsFuturesHandleData(t *testing.T) {
	t.Parallel()
	var err error
	for title, data := range futuresPushDataMap {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			err = e.wsFuturesHandleData(t.Context(), e.Websocket.Conn, []byte(data))
			assert.NoError(t, err)
		})
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetAccountBalance(generateContext())
	require.NoError(t, err)
}

func TestGetAccountBills(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountBills(generateContext(), time.Time{}, time.Time{}, 0, 0, "NEXT", "PNL")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	arg := &FuturesOrderRequest{
		ReduceOnly: true,
	}
	_, err := e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTC_USDT_PERP"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "LONG"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit_maker"
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.PlaceFuturesOrder(t.Context(), &FuturesOrderRequest{
		ClientOrderID:           "939a9d51-8f32-443a-9fb8-ff0852010487",
		Symbol:                  "BTC_USDT_PERP",
		Side:                    "buy",
		MarginMode:              "CROSS",
		PositionSide:            "LONG",
		OrderType:               "limit_maker",
		Price:                   46050,
		Size:                    10,
		TimeInForce:             "GTC",
		SelfTradePreventionMode: "EXPIRE_TAKER",
		ReduceOnly:              false,
	})
	require.NoError(t, err)
}

func TestPlaceMultipleOrders(t *testing.T) {
	t.Parallel()
	arg := FuturesOrderRequest{
		ReduceOnly: true,
	}
	_, err := e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{arg})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTC_USDT_PERP"
	_, err = e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	_, err = e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "LONG"
	_, err = e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit_maker"
	_, err = e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{
		{
			ClientOrderID:           "939a9d51",
			Symbol:                  "BTC_USDT_PERP",
			Side:                    "buy",
			MarginMode:              "CROSS",
			PositionSide:            "LONG",
			OrderType:               "limit_maker",
			Price:                   46050,
			Size:                    10,
			TimeInForce:             "GTC",
			SelfTradePreventionMode: "EXPIRE_TAKER",
			ReduceOnly:              false,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesOrder(t.Context(), &CancelOrderRequest{OrderID: "1234"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.CancelFuturesOrder(t.Context(), &CancelOrderRequest{Symbol: futuresTradablePair.String()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.CancelFuturesOrder(generateContext(), &CancelOrderRequest{Symbol: futuresTradablePair.String(), OrderID: "12345"})
	require.NoError(t, err)
}

func TestCancelMultipleFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleFuturesOrders(t.Context(), &CancelOrdersRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelMultipleFuturesOrders(t.Context(), &CancelOrdersRequest{Symbol: futuresTradablePair})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleFuturesOrders(generateContext(), &CancelOrdersRequest{Symbol: futuresTradablePair, OrderIDs: []string{"331378951169769472", "331378951182352384", "331378951199129601"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllFuturesOrders(t.Context(), "", "BUY")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllFuturesOrders(generateContext(), futuresTradablePair.String(), "BUY")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCloseAtMarketPrice(t *testing.T) {
	t.Parallel()
	_, err := e.CloseAtMarketPrice(t.Context(), "", "", "", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.CloseAtMarketPrice(t.Context(), futuresTradablePair.String(), "", "", "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = e.CloseAtMarketPrice(t.Context(), futuresTradablePair.String(), "CROSS", "", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.CloseAtMarketPrice(generateContext(), futuresTradablePair.String(), "CROSS", "", "123123")
	require.NoError(t, err)
}

func TestCloseAllAtMarketPrice(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseAllAtMarketPrice(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrentFuturesOrders(generateContext(), futuresTradablePair.String(), "SELL", "", "", "NEXT", 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderExecutionDetails(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1743615790295), time.UnixMilli(1743702190295)
	if !mockTests {
		startTime, endTime = time.Now().Add(-time.Hour*24), time.Now()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrderExecutionDetails(generateContext(), "", "", "", "NEXT", startTime, endTime, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOrderHistory(generateContext(), "", "LIMIT", "", "PARTIALLY_CANCELED", "", "", "PREV", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesCurrentPosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesCurrentPosition(generateContext(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositionHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesPositionHistory(generateContext(), "", "ISOLATED", "LONG", "NEXT", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAdjustMarginForIsolatedMarginTradingPositions(t *testing.T) {
	t.Parallel()
	_, err := e.AdjustMarginForIsolatedMarginTradingPositions(t.Context(), "", "", "ADD", 123)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.AdjustMarginForIsolatedMarginTradingPositions(t.Context(), "DOT_USDT_PERP", "", "ADD", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.AdjustMarginForIsolatedMarginTradingPositions(t.Context(), "DOT_USDT_PERP", "", "", 123)
	require.ErrorIs(t, err, errMarginAdjustTypeMissing)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.AdjustMarginForIsolatedMarginTradingPositions(generateContext(), "BTC_USDT_PERP", "", "ADD", 123)
	require.NoError(t, err)
}

func TestGetFuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesLeverage(t.Context(), "", "ISOLATED")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesLeverage(generateContext(), "BTC_USDT_PERP", "ISOLATED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetFuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetFuturesLeverage(t.Context(), "", "CROSS", "LONG", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.SetFuturesLeverage(t.Context(), "BTC_USDT_PERP", "", "LONG", 10)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = e.SetFuturesLeverage(t.Context(), "BTC_USDT_PERP", "CROSS", "", 10)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.SetFuturesLeverage(t.Context(), "BTC_USDT_PERP", "CROSS", "LONG", 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.SetFuturesLeverage(generateContext(), "BTC_USDT_PERP", "CROSS", "LONG", 10)
	require.NoError(t, err)
}

func TestSwitchPositionMode(t *testing.T) {
	t.Parallel()
	err := e.SwitchPositionMode(t.Context(), "")
	require.ErrorIs(t, err, errPositionModeInvalid)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.SwitchPositionMode(generateContext(), "HEDGE")
	require.NoError(t, err)
}

func TestGetPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPositionMode(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderBook(t.Context(), "", 100, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesOrderBook(t.Context(), "BTC_USDT_PERP", 100, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesKlineData(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetFuturesKlineData(t.Context(), "BTC_USDT_PERP", kline.HundredMilliseconds, time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetFuturesKlineData(t.Context(), "BTC_USDT_PERP", kline.FiveMin, time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesExecutionInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesExecutionInfo(t.Context(), "", 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesExecutionInfo(t.Context(), "BTC_USDT_PERP", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLiquidationOrder(t *testing.T) {
	t.Parallel()
	result, err := e.GetLiquidiationOrder(t.Context(), "BTC_USDT_PERP", "NEXT", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesMarketInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesMarketInfo(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesIndexPrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesIndexPrice(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceComponents(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceComponents(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetIndexPriceComponents(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceKlineData(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetIndexPriceKlineData(t.Context(), "BTC_USDT_PERP", kline.HundredMilliseconds, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetIndexPriceKlineData(t.Context(), "BTC_USDT_PERP", kline.FourHour, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesMarkPrice(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPriceKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceKlineData(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetMarkPriceKlineData(t.Context(), "BTC_USDT_PERP", kline.HundredMilliseconds, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetMarkPriceKlineData(t.Context(), "BTC_USDT_PERP", kline.FourHour, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesProductInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesProductInfo(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesProductInfo(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesCurrentFundingRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCurrentFundingRate(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesCurrentFundingRate(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesHistoricalFundingRates(t.Context(), "", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesCurrentOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCurrentOpenPositions(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesCurrentOpenPositions(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInsuranceFundInformation(t *testing.T) {
	t.Parallel()
	result, err := e.GetInsuranceFundInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesRiskLimit(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesRiskLimit(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIntervalString(t *testing.T) {
	t.Parallel()
	params := map[kline.Interval]struct {
		IntervalString string
		Error          error
	}{
		kline.OneMin:     {IntervalString: "MINUTE_1"},
		kline.FiveMin:    {IntervalString: "MINUTE_5"},
		kline.FifteenMin: {IntervalString: "MINUTE_15"},
		kline.ThirtyMin:  {IntervalString: "MINUTE_30"},
		kline.OneHour:    {IntervalString: "HOUR_1"},
		kline.TwoHour:    {IntervalString: "HOUR_2"},
		kline.FourHour:   {IntervalString: "HOUR_4"},
		kline.TwelveHour: {IntervalString: "HOUR_12"},
		kline.OneDay:     {IntervalString: "DAY_1"},
		kline.ThreeDay:   {IntervalString: "DAY_3"},
		kline.OneWeek:    {IntervalString: "WEEK_1"},
		kline.TwoWeek:    {Error: kline.ErrUnsupportedInterval},
	}
	for key, val := range params {
		s, err := intervalToString(key)
		require.Equal(t, val.IntervalString, s)
		require.ErrorIs(t, err, val.Error, err)
	}
}

func TestTimeInForceString(t *testing.T) {
	t.Parallel()
	timeInForceStringMap := map[order.TimeInForce]struct {
		String string
		Error  error
	}{
		order.GoodTillCancel:    {String: "GTC"},
		order.FillOrKill:        {String: "FOK"},
		order.ImmediateOrCancel: {String: "IOC"},
		order.GoodTillCrossing:  {Error: order.ErrInvalidTimeInForce},
	}
	for k, v := range timeInForceStringMap {
		result, err := TimeInForceString(k)
		assert.ErrorIs(t, err, v.Error)
		assert.Equal(t, v.String, result)
	}
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	orderStringMap := map[order.Type]struct {
		String string
		Error  error
	}{
		order.Market:       {String: order.Market.String()},
		order.Limit:        {String: order.Limit.String()},
		order.LimitMaker:   {String: order.LimitMaker.String()},
		order.StopLimit:    {String: "STOP_LIMIT"},
		order.AnyType:      {},
		order.UnknownType:  {},
		order.TrailingStop: {Error: order.ErrUnsupportedOrderType},
	}
	for k, v := range orderStringMap {
		result, err := OrderTypeString(k)
		require.ErrorIs(t, err, v.Error)
		assert.Equal(t, v.String, result)
	}
}

func TestStringToOrderType(t *testing.T) {
	t.Parallel()
	orderTypeStringToTypeMap := map[string]order.Type{
		"":                    order.Limit,
		"STOP":                order.Stop,
		"STOP_LIMIT":          order.StopLimit,
		"TRAILING_STOP":       order.TrailingStop,
		"TRAILING_STOP_LIMIT": order.TrailingStopLimit,
	}
	for k, v := range orderTypeStringToTypeMap {
		result := StringToOrderType(k)
		assert.Equal(t, result, v)
	}
}

func TestOrderStateString(t *testing.T) {
	t.Parallel()
	orderStatusToStringMap := map[string]order.Status{
		"NEW":                order.New,
		"FAILED":             order.Closed,
		"FILLED":             order.Filled,
		"CANCELED":           order.Cancelled,
		"abcd":               order.UnknownStatus,
		"PARTIALLY_FILLED":   order.PartiallyFilled,
		"PARTIALLY_CANCELED": order.PartiallyCancelled,
	}
	for k, v := range orderStatusToStringMap {
		result := orderStateFromString(k)
		assert.Equal(t, v, result)
	}
}

func TestStringToOrderSide(t *testing.T) {
	t.Parallel()
	stringToOrderSideMap := map[string]order.Side{
		order.Sell.String():  order.Sell,
		order.Buy.String():   order.Buy,
		order.Short.String(): order.Short,
		order.Long.String():  order.Long,
		"":                   order.UnknownSide,
	}
	for k, v := range stringToOrderSideMap {
		result := stringToOrderSide(k)
		assert.Equal(t, v, result)
	}
}

func generateContext() context.Context {
	if mockTests {
		return account.DeployCredentialsToContext(context.Background(), &account.Credentials{
			Key:    "abcde",
			Secret: "fghij",
		})
	}
	return context.Background()
}

func TestUnmarshalToFuturesCandle(t *testing.T) {
	t.Parallel()
	data := []byte(`[ [ "58651", "58651", "58651", "58651", "0", "0", "0", "1719975420000", "1719975479999" ], [ "58651", "58651", "58651", "58651", "0", "0", "0", "1719975480000", "1719975539999" ]]`)
	var resp []FuturesCandle
	err := json.Unmarshal(data, &resp)
	require.NoError(t, err)
	require.Len(t, resp, 2)
	assert.Equal(t, 58651.0, resp[0].LowestPrice.Float64())
	assert.Equal(t, 58651.0, resp[0].HighestPrice.Float64())
	assert.Equal(t, 58651.0, resp[0].OpeningPrice.Float64())
	assert.Equal(t, 58651.0, resp[0].ClosingPrice.Float64())
	assert.Equal(t, 0.0, resp[0].QuoteAmount.Float64())
	assert.Equal(t, 0.0, resp[0].BaseAmount.Float64())
	assert.Equal(t, 0.0, resp[0].Trades.Float64())
	assert.Equal(t, time.UnixMilli(1719975420000), resp[0].StartTime.Time())
	assert.Equal(t, time.UnixMilli(1719975479999), resp[0].EndTime.Time())
}

func TestGenerateFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	result, err := e.GenerateFuturesDefaultSubscriptions(true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHandleFuturesSubscriptions(t *testing.T) {
	t.Parallel()
	enabledPairs, err := e.GetEnabledPairs(asset.Futures)
	require.NoError(t, err)

	subscs := subscription.List{
		{
			Asset:   asset.Futures,
			Channel: channelFuturesTickers,
			Pairs:   enabledPairs,
		},
		{
			Asset:   asset.Futures,
			Channel: channelFuturesOrderbookLvl2,
			Pairs:   enabledPairs,
		},
	}
	payloads := []SubscriptionPayload{
		{Event: "subscribe", Channel: []string{"tickers"}, Symbols: enabledPairs.Strings()},
		{Event: "subscribe", Channel: []string{"book_lv2"}, Symbols: enabledPairs.Strings()},
	}
	result := e.handleFuturesSubscriptions("subscribe", subscs)
	require.Len(t, payloads, 2)
	for i := range subscs {
		require.Equal(t, payloads[i], result[i])
	}
}

func TestWebsocketSubmitOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSubmitOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestOrderbookLevelFromSlice(t *testing.T) {
	t.Parallel()
	var obData []types.Number
	data := []byte(`["88350.22","0.019937","88376.19","0.000203","88376.58","0.000696"]`)
	err := json.Unmarshal(data, &obData)
	require.NoError(t, err)

	target := []orderbook.Level{{Price: 88350.22, Amount: 0.019937}, {Price: 88376.19, Amount: 0.000203}, {Price: 88376.58, Amount: 0.000696}}
	obLevels := orderbookLevelFromSlice(obData)
	require.Len(t, obLevels, len(target))
	for x := range obLevels {
		require.Equal(t, target[x].Price, obLevels[x].Price)
		require.Equal(t, target[x].Amount, obLevels[x].Amount)
	}
}
