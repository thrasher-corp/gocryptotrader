package poloniex

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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

	websocketMockTestsSkipped = "skipped websocket test while mock testing is enabled"
)

var (
	e                                     *Exchange
	spotTradablePair, futuresTradablePair currency.Pair
)

func (e *Exchange) setAPICredential(apiKey, apiSecret string) { //nolint:unparam // Intentional suppress 'apiKey always receives apiKey ("")' error
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
	require.NoError(t, err)

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
		_, err := e.GetFee(generateContext(t), feeBuilder)
		assert.NoError(t, err)

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		_, err = e.GetFee(generateContext(t), feeBuilder)
		assert.NoError(t, err)

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := e.GetFee(generateContext(t), feeBuilder); err != nil {
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
	assert.NoError(t, err)
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
	_, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetActiveOrders(generateContext(t), &order.MultiOrderRequest{
		AssetType: asset.Spot,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetActiveOrders(generateContext(t), &order.MultiOrderRequest{
		AssetType: asset.Futures,
		Side:      order.Buy,
	})
	assert.NoError(t, err)
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
	result, err := e.GetOrderHistory(generateContext(t), &order.MultiOrderRequest{
		Type:      order.Limit,
		AssetType: asset.Spot,
		Side:      order.Buy,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOrderHistory(generateContext(t), &order.MultiOrderRequest{
		Type:      order.StopLimit,
		AssetType: asset.Spot,
		Side:      order.Sell,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOrderHistory(generateContext(t), &order.MultiOrderRequest{
		Type:      order.Limit,
		AssetType: asset.Futures,
		Side:      order.Buy,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()

	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = e.SubmitOrder(t.Context(), &order.Submit{})
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	arg := &order.Submit{Exchange: e.Name}
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	arg.Pair = spotTradablePair
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAssetNotSet)

	arg.AssetType = asset.Spot
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Bid
	arg.Type = order.Type(65537)
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.Type = order.Limit
	arg.TimeInForce = order.GoodTillCancel
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	arg.Amount = 1
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder)

	arg = &order.Submit{Exchange: e.Name, AssetType: asset.Options, Side: order.Long, Type: order.Market, Amount: 1, TimeInForce: order.GoodTillCrossing, Pair: futuresTradablePair}
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	// unit tests specific to spot
	arg.AssetType = asset.Spot
	arg.Type = order.Liquidation
	arg.Pair = spotTradablePair
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	// spot smart orders validation
	arg.Type = order.TrailingStopLimit
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTrailingOffset)

	arg.Side = order.Sell
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTrailingOffset)

	arg.TrackingValue = 5
	arg.TrackingMode = order.Percentage

	// Futures place order
	arg = &order.Submit{Exchange: e.Name, AssetType: asset.Futures, Type: order.Market, Amount: 1, TimeInForce: order.GoodTillCrossing, Pair: futuresTradablePair, MarginType: margin.Isolated}
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell
	arg.Type = order.TrailingStopLimit
	_, err = e.SubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubmitOrder(generateContext(t), &order.Submit{
		Exchange:    e.Name,
		Pair:        spotTradablePair,
		Side:        order.Buy,
		Type:        order.Market,
		Price:       10,
		QuoteAmount: 10000000,
		AssetType:   asset.Spot,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.SubmitOrder(generateContext(t), &order.Submit{
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
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.SubmitOrder(generateContext(t), &order.Submit{
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
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.SubmitOrder(generateContext(t), &order.Submit{
		Exchange:  e.Name,
		Pair:      futuresTradablePair,
		Side:      order.Buy,
		Type:      order.Market,
		Price:     10,
		Amount:    10000000,
		AssetType: asset.Futures,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.SubmitOrder(generateContext(t), &order.Submit{
		Exchange:   e.Name,
		Pair:       futuresTradablePair,
		Side:       order.Buy,
		MarginType: margin.Multi,
		Type:       order.Limit,
		Price:      10,
		Amount:     10000000,
		AssetType:  asset.Futures,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()

	_, err := e.WebsocketSubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{})
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	arg := &order.Submit{Exchange: e.Name}
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	arg.Pair = spotTradablePair
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAssetNotSet)

	arg.AssetType = asset.Spot
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.Type = order.TrailingStop
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	arg.Amount = 1
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.Type = order.Limit
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder)

	arg.AssetType = asset.Futures
	arg.Price = 10
	_, err = e.WebsocketSubmitOrder(t.Context(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	if mockTests {
		t.Skip(websocketMockTestsSkipped)
	}
	e.setAPICredential(apiKey, apiSecret)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	require.True(t, e.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")

	testexch.SetupWs(t, e)
	result, err := e.WebsocketSubmitOrder(generateContext(t), &order.Submit{
		Exchange:    e.Name,
		Pair:        spotTradablePair,
		Side:        order.Buy,
		Type:        order.Market,
		Price:       10,
		QuoteAmount: 1000000,
		AssetType:   asset.Spot,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.WebsocketSubmitOrder(generateContext(t), &order.Submit{
		Exchange:     e.Name,
		Pair:         spotTradablePair,
		Side:         order.Sell,
		Type:         order.Limit,
		TriggerPrice: 11,
		Price:        10,
		Amount:       1,
		ClientID:     "hi",
		AssetType:    asset.Spot,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	err := e.WebsocketCancelOrder(t.Context(), &order.Cancel{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if mockTests {
		t.Skip(websocketMockTestsSkipped)
	}
	e.setAPICredential(apiKey, apiSecret)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	require.True(t, e.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")

	testexch.SetupWs(t, e)
	err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: "2312", ClientOrderID: "23123121231"})
	assert.NoError(t, err)
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	arg := &order.Cancel{
		AccountID: "1",
	}
	err := e.CancelOrder(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	err = e.CancelOrder(t.Context(), arg)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = "123"
	err = e.CancelOrder(t.Context(), arg)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.AssetType = asset.Spot
	err = e.CancelOrder(generateContext(t), arg)
	assert.NoError(t, err)

	arg.Type = order.StopLimit
	err = e.CancelOrder(generateContext(t), arg)
	assert.NoError(t, err)

	err = e.CancelOrder(generateContext(t), &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Futures,
	})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	err = e.CancelOrder(generateContext(t), &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Futures,
		Pair:      futuresTradablePair,
	})
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		testexch.SetupWs(t, e)
	}
	arg := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      spotTradablePair,
		AssetType: asset.Spot,
	}
	arg.Type = order.Stop
	result, err := e.CancelAllOrders(generateContext(t), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.Limit
	result, err = e.CancelAllOrders(generateContext(t), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.AssetType = asset.Futures
	arg.Pair = futuresTradablePair
	result, err = e.CancelAllOrders(generateContext(t), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.StopLimit
	result, err = e.CancelAllOrders(generateContext(t), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	arg := &order.Modify{
		OrderID: "1337",
		Price:   1337,
	}
	_, err := e.ModifyOrder(t.Context(), arg)
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
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.StopLimit
	result, err = e.ModifyOrder(t.Context(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil)

	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{})
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: e.Name,
	})
	require.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero)

	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: e.Name,
		Amount:   1,
	})
	require.ErrorContains(t, err, withdraw.ErrStrNoCurrencySet)

	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: e.Name,
		Amount:   1,
		Type:     withdraw.Crypto,
		Currency: currency.USD,
	})
	require.ErrorContains(t, err, withdraw.ErrStrCurrencyNotCrypto)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: e.Name,
		Crypto: withdraw.CryptoRequest{
			Address: "bc1qk0jareu4jytc0cfrhr5wgshsq8",
			Chain:   "ETH",
		},
		Amount:   0.0000001,
		Currency: currency.BTC,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UpdateAccountBalances(generateContext(t), asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateAccountBalances(generateContext(t), asset.Futures)
	assert.NoError(t, err)
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
	startTime, endTime := time.UnixMilli(1744183959258), time.UnixMilli(1744191159258)
	if !mockTests {
		startTime, endTime = time.Now().Add(-time.Hour*2), time.Now()
	}
	result, err := e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.FiveMin, startTime.UTC(), endTime.UTC())
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.FiveMin, startTime.UTC(), endTime.UTC())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1744103854944), time.UnixMilli(1744190254944)
	if !mockTests {
		startTime, endTime = time.Now().Add(-time.Hour*24), time.Now()
	}
	result, err := e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.OneHour, startTime, endTime)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, kline.FiveMin, startTime, endTime)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	result, err := e.GetRecentTrades(t.Context(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetRecentTrades(t.Context(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(),
		spotTradablePair, asset.Spot, time.Time{}, time.Time{})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
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
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip("skipped mock test because GetAccountFundingHistory uses dynamic timestamp data")
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingHistory(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip("skipped mock test because GetWithdrawalsHistory uses dynamic timestamp data")
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Options}})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, OrderID: "1234"}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, Pair: futuresTradablePair}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, OrderID: "1233"}, {AssetType: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}

	resp, err := e.CancelBatchOrders(generateContext(t), []order.Cancel{
		{
			Pair:      futuresTradablePair,
			AssetType: asset.Futures,
			OrderID:   "1233",
		},
		{
			Pair:      futuresTradablePair,
			AssetType: asset.Futures,
			OrderID:   "123444",
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	resp, err = e.CancelBatchOrders(generateContext(t), []order.Cancel{
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
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	resp, err = e.CancelBatchOrders(generateContext(t), []order.Cancel{
		{
			OrderID:   "134",
			AssetType: asset.Spot,
			Type:      order.StopLimit,
		},
		{
			OrderID:   "234",
			AssetType: asset.Spot,
			Type:      order.TrailingStop,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	assert.NoError(t, err)
	assert.NotZero(t, st)

	st, err = e.GetServerTime(t.Context(), asset.Futures)
	assert.NoError(t, err)
	assert.NotZero(t, st)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Spot, spotTradablePair)
	assert.NoError(t, err)
	require.False(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.Futures, futuresTradablePair)
	assert.NoError(t, err)
	assert.True(t, is)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	assert.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.FetchTradablePairs(t.Context(), asset.Futures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbol(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetSymbol(t.Context(), spotTradablePair)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	result, err := e.GetSymbols(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetCurrencies(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetCurrency(t.Context(), currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTimestamp(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemTimestamp(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketPrices(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarketPrices(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetMarketPrice(t.Context(), spotTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrices(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarkPrices(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetMarkPrice(t.Context(), spotTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarkPriceComponents(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceComponents(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetMarkPriceComponents(t.Context(), spotTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetOrderbook(t.Context(), spotTradablePair, 0, 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOrderbook(t.Context(), spotTradablePair, 1, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateOrderbook(t.Context(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlesticks(t.Context(), currency.EMPTYPAIR, kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	startTime, endTime := time.UnixMilli(1743615790295), time.UnixMilli(1743702190295)
	_, err = e.GetCandlesticks(t.Context(), spotTradablePair, kline.HundredMilliseconds, startTime, time.Time{}, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	_, err = e.GetCandlesticks(t.Context(), spotTradablePair, kline.FiveMin, endTime, startTime, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		startTime, endTime = endTime.Add(-time.Hour*24), time.Now()
	}
	result, err := e.GetCandlesticks(t.Context(), spotTradablePair, kline.FiveMin, startTime, endTime, 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetTrades(t.Context(), spotTradablePair, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	result, err := e.GetTickers(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.GetTicker(t.Context(), spotTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateral(t *testing.T) {
	t.Parallel()
	_, err := e.GetCollateral(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetCollateral(t.Context(), currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollaterals(t *testing.T) {
	t.Parallel()
	result, err := e.GetCollaterals(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowRate(t *testing.T) {
	t.Parallel()
	result, err := e.GetBorrowRate(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccount(generateContext(t))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBalances(generateContext(t), "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.GetBalances(generateContext(t), "SPOT")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBalancesByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountBalances(t.Context(), "", "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountBalances(generateContext(t), "329455537441832960", "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.GetAccountBalances(generateContext(t), "329455537441832960", "SPOT")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountActivities(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1764664401864), time.UnixMilli(1764668001864)
	_, err := e.GetAccountActivities(generateContext(t), endTime, startTime, 0, 0, 0, "", currency.EMPTYCODE)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour), time.Now()
	}
	_, err = e.GetAccountActivities(generateContext(t), startTime, endTime, 0, 0, 0, "", currency.EMPTYCODE)
	require.NoError(t, err)

	_, err = e.GetAccountActivities(generateContext(t), time.Time{}, time.Time{}, 200, 0, 0, "", currency.EMPTYCODE)
	require.NoError(t, err)

	_, err = e.GetAccountActivities(generateContext(t), time.Time{}, time.Time{}, 0, 10, 100, "PRE", currency.EMPTYCODE)
	require.NoError(t, err)

	_, err = e.GetAccountActivities(generateContext(t), time.Time{}, time.Time{}, 0, 0, 0, "NEXT", currency.BTC)
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

	_, err = e.AccountsTransfer(t.Context(), &AccountTransferRequest{
		Amount:    1,
		Currency:  currency.BTC,
		ToAccount: "219961623421431808",
	})
	require.ErrorIs(t, err, errAddressRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AccountsTransfer(generateContext(t), &AccountTransferRequest{
		Amount:      1,
		Currency:    currency.BTC,
		FromAccount: "329455537441832960",
		ToAccount:   "329455537441832960",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountsTransferRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1764664401864), time.UnixMilli(1764668001864)
	_, err := e.GetAccountsTransferRecords(generateContext(t), endTime, startTime, "", currency.BTC, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour), time.Now()
	}
	_, err = e.GetAccountsTransferRecords(generateContext(t), startTime, endTime, "", currency.BTC, 0, 0)
	require.NoError(t, err)

	_, err = e.GetAccountsTransferRecords(generateContext(t), time.Time{}, time.Time{}, "NEXT", currency.BTC, 1, 100)
	require.NoError(t, err)

	_, err = e.GetAccountsTransferRecords(generateContext(t), startTime, endTime, "", currency.EMPTYCODE, 0, 0)
	require.NoError(t, err)
}

func TestGetAccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountsTransferRecordByTransferID(generateContext(t), "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetAccountsTransferRecordByTransferID(generateContext(t), "329455537441832960")
	require.NoError(t, err)
}

func TestGetFeeInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFeeInfo(generateContext(t))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1764664401864), time.UnixMilli(1764668001864)
	_, err := e.GetInterestHistory(generateContext(t), endTime, startTime, "", 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour), time.Now()
	}
	_, err = e.GetInterestHistory(generateContext(t), startTime, endTime, "", 0, 0)
	require.NoError(t, err)

	_, err = e.GetInterestHistory(generateContext(t), time.Time{}, time.Time{}, "NEXT", 0, 0)
	require.NoError(t, err)

	_, err = e.GetInterestHistory(generateContext(t), time.Time{}, time.Time{}, "NEXT", 1, 100)
	require.NoError(t, err)

	_, err = e.GetInterestHistory(generateContext(t), time.Time{}, time.Time{}, "", 0, 0)
	require.NoError(t, err)
}

func TestGetSubAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccount(generateContext(t))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubAccountBalances(generateContext(t))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountBalance(t.Context(), "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetSubAccountBalance(generateContext(t), "2d45301d-5f08-4a2b-a763-f9199778d854")
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
	})
	require.ErrorIs(t, err, errAccountIDRequired)
	_, err = e.SubAccountTransfer(t.Context(), &SubAccountTransferRequest{
		Currency:      currency.BTC,
		Amount:        1,
		FromAccountID: "1234568",
		ToAccountID:   "1234567",
	})
	require.ErrorIs(t, err, errAccountTypeRequired)

	_, err = e.SubAccountTransfer(t.Context(), &SubAccountTransferRequest{
		Currency:        currency.BTC,
		Amount:          1,
		FromAccountID:   "1234568",
		ToAccountID:     "1234567",
		FromAccountType: "SPOT",
	})
	require.ErrorIs(t, err, errAccountTypeRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubAccountTransfer(generateContext(t), &SubAccountTransferRequest{
		Currency:        currency.BTC,
		Amount:          1,
		FromAccountID:   "329455537441832960",
		ToAccountID:     "329455537441832961",
		FromAccountType: "SPOT",
		ToAccountType:   "SPOT",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountTransferRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1764664401864), time.UnixMilli(1764668001864)
	_, err := e.GetSubAccountTransferRecords(generateContext(t), &SubAccountTransferRecordRequest{Currency: currency.BTC, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour), time.Now()
	}
	_, err = e.GetSubAccountTransferRecords(generateContext(t), &SubAccountTransferRecordRequest{
		Currency:        currency.BTC,
		FromAccountType: "SPOT",
		ToAccountType:   "FUTURES",
		ToAccountID:     "32b323201-e832-2270-78t4-e25ak408946",
		FromAccountID:   "32b323201-e832-2270-78t4-e25ak408941",
	})
	require.NoError(t, err)

	_, err = e.GetSubAccountTransferRecords(generateContext(t), &SubAccountTransferRecordRequest{Currency: currency.BTC, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err)

	_, err = e.GetSubAccountTransferRecords(generateContext(t), &SubAccountTransferRecordRequest{Currency: currency.BTC, StartTime: startTime, From: 1, Limit: 100})
	require.NoError(t, err)

	_, err = e.GetSubAccountTransferRecords(generateContext(t), &SubAccountTransferRecordRequest{Currency: currency.BTC, Direction: "NEXT"})
	require.NoError(t, err)
}

func TestGetSubAccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountTransferRecord(t.Context(), "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetSubAccountTransferRecord(generateContext(t), "329455537441832960")
	require.NoError(t, err)
}

func TestGetDepositAddresses(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetDepositAddresses(generateContext(t), currency.LTC)
	require.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(generateContext(t), "1234", spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrderInfo(generateContext(t), "1234", spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOrderInfo(generateContext(t), "12345", futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDepositAddress(generateContext(t), currency.BTC, "", "TON")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWalletActivity(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1743575750138), time.UnixMilli(1743582950138)
	_, err := e.WalletActivity(generateContext(t), endTime, startTime, "")
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour*2), time.Now()
	}
	_, err = e.WalletActivity(generateContext(t), startTime, endTime, "deposits")
	require.NoError(t, err)

	result, err := e.WalletActivity(generateContext(t), startTime, endTime, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.NewCurrencyDepositAddress(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.NewCurrencyDepositAddress(generateContext(t), currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCurrency(t.Context(), &WithdrawCurrencyRequest{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawCurrencyRequest{Coin: currency.BTC})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawCurrencyRequest{Coin: currency.BTC, Amount: 1})
	require.ErrorIs(t, err, errInvalidWithdrawalChain)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawCurrencyRequest{Coin: currency.BTC, Amount: 1, Network: "BTC"})
	require.ErrorIs(t, err, errAddressRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WithdrawCurrency(t.Context(), &WithdrawCurrencyRequest{Network: "ERP", Coin: currency.BTC, Amount: 1, Address: "bc1qk0jareu4jytc0cfrhr5wgshsq8"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountMargin(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetAccountMargin(generateContext(t), "")
	require.ErrorIs(t, err, errAccountTypeRequired)

	result, err := e.GetAccountMargin(generateContext(t), "SPOT")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetBorrowStatus(generateContext(t), currency.USDT)
	require.NoError(t, err)
}

func TestMaximumBuySellAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginBuySellAmounts(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginBuySellAmounts(generateContext(t), spotTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceOrder(t.Context(), &PlaceOrderRequest{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{Symbol: spotTradablePair})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{Symbol: spotTradablePair, Side: order.Sell})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{
		Symbol:        spotTradablePair,
		Side:          order.Buy,
		Type:          OrderType(order.Market),
		Quantity:      1,
		Price:         40000.50000,
		TimeInForce:   TimeInForce(order.GoodTillCancel),
		ClientOrderID: "1234Abc",
	})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{
		Symbol:        spotTradablePair,
		Side:          order.Sell,
		Type:          OrderType(order.Market),
		Amount:        1,
		Price:         40000.50000,
		TimeInForce:   TimeInForce(order.GoodTillCancel),
		ClientOrderID: "1234Abc",
	})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceOrder(t.Context(), &PlaceOrderRequest{
		Symbol:        spotTradablePair,
		Side:          order.Buy,
		Type:          OrderType(order.Market),
		Amount:        100,
		Price:         100000.50000,
		TimeInForce:   TimeInForce(order.GoodTillCancel),
		ClientOrderID: "1234Abc",
	})
	assert.NoError(t, err)
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

	_, err = e.PlaceBatchOrders(t.Context(), []PlaceOrderRequest{{
		Symbol:        spotTradablePair,
		Side:          order.Buy,
		Type:          OrderType(order.Market),
		Quantity:      1,
		Price:         40000.50000,
		TimeInForce:   TimeInForce(order.GoodTillCancel),
		ClientOrderID: "1234Abc",
	}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.PlaceBatchOrders(t.Context(), []PlaceOrderRequest{{
		Symbol:        spotTradablePair,
		Side:          order.Sell,
		Type:          OrderType(order.Market),
		Amount:        1,
		Price:         40000.50000,
		TimeInForce:   TimeInForce(order.GoodTillCancel),
		ClientOrderID: "1234Abc",
	}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

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
			Side:          order.Buy,
			Type:          OrderType(order.Market),
			Amount:        1,
			Price:         40000.50000,
			TimeInForce:   TimeInForce(order.GoodTillCancel),
			ClientOrderID: "1234Abc",
		},
		{
			Symbol: getPairFromString("BTC_USDT"),
			Amount: 100,
			Side:   order.Buy,
		},
		{
			Symbol:        getPairFromString("BTC_USDT"),
			Type:          OrderType(order.Limit),
			Quantity:      100,
			Side:          order.Buy,
			Price:         40000.50000,
			TimeInForce:   TimeInForce(order.ImmediateOrCancel),
			ClientOrderID: "1234Abc",
		},
		{
			Symbol: getPairFromString("ETH_USDT"),
			Amount: 1000,
			Side:   order.Buy,
		},
		{
			Symbol:        getPairFromString("TRX_USDT"),
			Type:          OrderType(order.Limit),
			Quantity:      15000,
			Side:          order.Sell,
			Price:         0.0623423423,
			TimeInForce:   TimeInForce(order.ImmediateOrCancel),
			ClientOrderID: "456Xyz",
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelReplaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelReplaceOrder(t.Context(), &CancelReplaceOrderRequest{TimeInForce: TimeInForce(order.GoodTillCancel)})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelReplaceOrder(t.Context(), &CancelReplaceOrderRequest{
		OrderID:       "29772698821328896",
		ClientOrderID: "1234Abc",
		Price:         18000,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetOpenOrders(generateContext(t), spotTradablePair, "", "NEXT", "", 10)
	require.NoError(t, err)

	_, err = e.GetOpenOrders(generateContext(t), spotTradablePair, "SELL", "NEXT", "24993088082542592", 10)
	require.NoError(t, err)
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrder(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.GetOrder(generateContext(t), "12345536545645", "")
	require.ErrorIs(t, err, order.ErrGetFailed)

	_, err = e.GetOrder(generateContext(t), "", "12345")
	require.ErrorIs(t, err, order.ErrGetFailed)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrder(generateContext(t), "12345", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrderByID(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.CancelOrderByID(t.Context(), "", "12345536545645")
	require.ErrorIs(t, err, order.ErrCancelFailed)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOrderByID(t.Context(), "12345536545645", "")
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelTradeOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err := e.CancelTradeOrders(t.Context(), []string{"BTC_USDT", "ETH_USDT"}, []AccountType{AccountType(asset.Spot)})
	require.NoError(t, err)
}

func TestKillSwitch(t *testing.T) {
	t.Parallel()
	_, err := e.KillSwitch(t.Context(), 0)
	require.ErrorIs(t, err, errInvalidTimeout)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.KillSwitch(generateContext(t), time.Second*30)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDisableKillSwitch(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DisableKillSwitch(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKillSwitchStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetKillSwitchStatus(generateContext(t))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSmartOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSmartOrder(t.Context(), &SmartOrderRequest{Side: order.Buy})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.CreateSmartOrder(t.Context(), &SmartOrderRequest{Symbol: spotTradablePair})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = e.CreateSmartOrder(t.Context(), &SmartOrderRequest{Symbol: spotTradablePair, Side: order.Buy})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	_, err = e.CreateSmartOrder(t.Context(), &SmartOrderRequest{Symbol: spotTradablePair, Side: order.Buy, Quantity: 10, Type: OrderType(order.StopLimit)})
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder)

	_, err = e.CreateSmartOrder(t.Context(), &SmartOrderRequest{Symbol: spotTradablePair, Side: order.Buy, Quantity: 10, Type: OrderType(order.TrailingStopLimit), Price: 1234, TrailingOffset: "1%"})
	require.ErrorIs(t, err, errInvalidOffsetLimit)

	_, err = e.CreateSmartOrder(t.Context(), &SmartOrderRequest{Symbol: spotTradablePair, Side: order.Buy, Quantity: 10, Type: OrderType(order.TrailingStopLimit), Price: 1234})
	require.ErrorIs(t, err, errInvalidTrailingOffset)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSmartOrder(generateContext(t), &SmartOrderRequest{
		Symbol:        spotTradablePair,
		Type:          OrderType(order.StopLimit),
		Price:         100000.5,
		ClientOrderID: "1234Abc",
		Side:          order.Buy,
		TimeInForce:   TimeInForce(order.GoodTillCancel),
		Quantity:      100,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equalf(t, int64(200), result.Code, "CreateSmartOrder error with code: %d message: %s", result.Code, result.Message)

	result, err = e.CreateSmartOrder(generateContext(t), &SmartOrderRequest{
		Symbol:         spotTradablePair,
		Type:           OrderType(order.TrailingStopLimit),
		Price:          100000.5,
		ClientOrderID:  "55667798abcd",
		Side:           order.Buy,
		TimeInForce:    TimeInForce(order.GoodTillCancel),
		Quantity:       100,
		TrailingOffset: "2%",
		LimitOffset:    "1%",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equalf(t, int64(0), result.Code, "CreateSmartOrder error with code: %d message: %s", result.Code, result.Message)
}

func TestCancelReplaceSmartOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelReplaceSmartOrder(t.Context(), &CancelReplaceSmartOrderRequest{Price: 18000})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.CancelReplaceSmartOrder(t.Context(), &CancelReplaceSmartOrderRequest{NewClientOrderID: "1234Abc", Price: 18000})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelReplaceSmartOrder(t.Context(), &CancelReplaceSmartOrderRequest{
		OrderID:          "29772698821328896",
		NewClientOrderID: "1234Abc",
		Price:            18000,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSmartOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetSmartOpenOrders(generateContext(t), 10, []string{"TRAILING_STOP", "TRAILING_STOP_LIMIT", "STOP", "STOP_LIMIT"})
	require.NoError(t, err)
}

func TestGetSmartOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSmartOrderDetails(generateContext(t), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSmartOrderDetails(generateContext(t), "123313413", "")
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleSmartOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleSmartOrders(t.Context(), &CancelOrdersRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleSmartOrders(t.Context(), &CancelOrdersRequest{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSmartOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSmartOrders(t.Context(), []currency.Pair{{Base: currency.BTC, Delimiter: "_", Quote: currency.USDT}, {Base: currency.ETH, Delimiter: "_", Quote: currency.USDT}}, []AccountType{AccountType(asset.Spot)}, []OrderType{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrdersHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1764930174763), time.UnixMilli(1765290174763)
	_, err := e.GetOrdersHistory(generateContext(t), &OrdersHistoryRequest{Symbol: spotTradablePair, AccountType: "SPOT", Limit: 10, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour*100), time.Now()
	}
	_, err = e.GetOrdersHistory(generateContext(t), &OrdersHistoryRequest{Symbol: spotTradablePair, AccountType: "SPOT", Limit: 10, OrderTypes: []string{"LIMIT", "LIMIT_MAKER"}})
	require.NoError(t, err)

	_, err = e.GetOrdersHistory(generateContext(t), &OrdersHistoryRequest{
		Symbol:      spotTradablePair,
		AccountType: "SPOT",
		From:        228530000,
		Limit:       10,
		Direction:   "NEXT",
		Side:        order.Sell,
		HideCancel:  true,
		StartTime:   startTime,
		EndTime:     endTime,
		OrderType:   "LIMIT_MAKER",
		States:      "FAILED",
	})
	require.NoError(t, err)
}

func TestGetSmartOrderHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1764930174763), time.UnixMilli(1765290174763)
	_, err := e.GetSmartOrderHistory(generateContext(t), &OrdersHistoryRequest{Limit: 10, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour*100), time.Now()
	}
	_, err = e.GetSmartOrderHistory(generateContext(t), &OrdersHistoryRequest{
		Symbol:      spotTradablePair,
		AccountType: "SPOT",
		OrderType:   "LIMIT",
		Side:        order.Sell,
		Direction:   "PRE",
		From:        12323123,
		Limit:       100,
		StartTime:   startTime,
		EndTime:     endTime,
		HideCancel:  true,
	})
	require.NoError(t, err)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1764664401864), time.UnixMilli(1764668001864)
	_, err := e.GetTradeHistory(generateContext(t), currency.Pairs{spotTradablePair}, "", 0, 0, endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour), time.Now()
	}
	_, err = e.GetTradeHistory(generateContext(t), currency.Pairs{spotTradablePair}, "", 0, 0, startTime, endTime)
	require.NoError(t, err)

	_, err = e.GetTradeHistory(generateContext(t), currency.Pairs{spotTradablePair}, "", 1, 100, startTime, endTime)
	require.NoError(t, err)

	_, err = e.GetTradeHistory(generateContext(t), currency.Pairs{spotTradablePair}, "NEXT", 10, 100, startTime, endTime)
	require.NoError(t, err)

	_, err = e.GetTradeHistory(generateContext(t), currency.Pairs{spotTradablePair}, "", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
}

func TestGetTradeOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradesByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetTradesByOrderID(generateContext(t), "13123242323")
	require.NoError(t, err)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	var privateQualifiedChannels []string
	if e.ValidateAPICredentials(t.Context(), asset.Spot) == nil {
		privateQualifiedChannels = append(privateQualifiedChannels, "orders", "balances")
	}

	for _, input := range []struct {
		gen func() (subscription.List, error)
		exp []string
	}{
		{
			gen: e.generateSubscriptions,
			exp: []string{"candles_minute_5", "trades", "ticker", "book_lv2"},
		},
		{
			gen: e.generatePrivateSubscriptions,
			exp: privateQualifiedChannels,
		},
	} {
		got, err := input.gen()
		require.NoError(t, err)

		var gotQualifiedChannels []string
		for _, inp := range got {
			gotQualifiedChannels = append(gotQualifiedChannels, inp.QualifiedChannel)
		}
		assert.Equal(t, input.exp, gotQualifiedChannels)
	}
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
	"Trades":         `{"channel":"trades","data":[{"symbol":"BTC_USDC","amount":"52.821342","quantity":"0.0021","takerSide":"sell","createTime":1694469183664,"price":"25153.02","id":71076055,"ts":1694469183673}]}`,
	"Currencies":     `{"channel":"currencies","data":[{"currency":"BTC","id":28,"name":"Bitcoin","description":"BTC Clone","type":"address","withdrawalFee":"0.0008","minConf":2,"depositAddress":null,"blockchain":"BTC","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["BTCTRON"]},{"currency":"XRP","id":243,"name":"XRP","description":"Payment ID","type":"address-payment-id","withdrawalFee":"0.2","minConf":2,"depositAddress":"rwU8rAiE2eyEPz3sikfbHuqCuiAtdXqa2v","blockchain":"XRP","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":false,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":[]},{"currency":"ETH","id":267,"name":"Ethereum","description":"Sweep to Main Account","type":"address","withdrawalFee":"0.00197556","minConf":64,"depositAddress":null,"blockchain":"ETH","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["ETHTRON"]},{"currency":"USDT","id":214,"name":"Tether USD","description":"Sweep to Main Account","type":"address","withdrawalFee":"0","minConf":2,"depositAddress":null,"blockchain":"OMNI","delisted":false,"tradingState":"NORMAL","walletState":"DISABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["USDTETH","USDTTRON"]},{"currency":"DOGE","id":59,"name":"Dogecoin","description":"BTC Clone","type":"address","withdrawalFee":"20","minConf":6,"depositAddress":null,"blockchain":"DOGE","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["DOGETRON"]},{"currency":"LTC","id":125,"name":"Litecoin","description":"BTC Clone","type":"address","withdrawalFee":"0.001","minConf":4,"depositAddress":null,"blockchain":"LTC","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["LTCTRON"]},{"currency":"DASH","id":60,"name":"Dash","description":"BTC Clone","type":"address","withdrawalFee":"0.01","minConf":20,"depositAddress":null,"blockchain":"DASH","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":false,"isChildChain":false,"supportCollateral":false,"supportBorrow":false,"childChains":[]}],"action":"snapshot"}`,
	"Symbols":        `{"channel":"symbols","data":[{"symbol":"BTC_USDC","baseCurrencyName":"BTC","quoteCurrencyName":"USDT","displayName":"BTC/USDT","state":"NORMAL","visibleStartTime":1659018819512,"tradableStartTime":1659018819512,"crossMargin":{"supportCrossMargin":true,"maxLeverage":"3"},"symbolTradeLimit":{"symbol":"BTC_USDT","priceScale":2,"quantityScale":6,"amountScale":2,"minQuantity":"0.000001","minAmount":"1","highestBid":"0","lowestAsk":"0"}}],"action":"snapshot"}`,
}

func TestWsPushData(t *testing.T) {
	t.Parallel()
	for key, value := range pushMessages {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			err := e.wsHandleData(generateContext(t), e.Websocket.Conn, []byte(value))
			assert.NoError(t, err)
		})
	}
	// Since running test in parallel shuffles the order of execution
	// We run book_lv2 data handling, ensuring the snapshot is processed before the update as follows
	err := e.wsHandleData(generateContext(t), e.Websocket.Conn, []byte(`{"channel":"book_lv2","data":[{"symbol":"BTC_USDC","createTime":1694469187745,"asks":[],"bids":[["25148.81","0.02158"],["25088.11","0"]],"lastId":598273385,"id":598273386,"ts":1694469187760}],"action":"snapshot"}`))
	require.NoError(t, err, "book_lv2 snapshot must not error")
	err = e.wsHandleData(generateContext(t), e.Websocket.Conn, []byte(`{"channel":"book_lv2","data":[{"symbol":"BTC_USDC","createTime":1694469187745,"asks":[],"bids":[["25148.81","0.02158"],["25088.11","0"]],"lastId":598273385,"id":598273386,"ts":1694469187760}],"action":"update"}`))
	assert.NoError(t, err, "book_lv2 update should not error")
}

func TestWsCreateOrder(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	_, err := e.WsCreateOrder(t.Context(), &PlaceOrderRequest{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.WsCreateOrder(t.Context(), &PlaceOrderRequest{
		Symbol: spotTradablePair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = e.WsCreateOrder(t.Context(), &PlaceOrderRequest{
		Symbol: spotTradablePair,
		Side:   order.Sell,
	})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	if mockTests {
		t.Skip(websocketMockTestsSkipped)
	}

	e.setAPICredential(apiKey, apiSecret)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	require.True(t, e.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")

	testexch.SetupWs(t, e)
	result, err := e.WsCreateOrder(generateContext(t), &PlaceOrderRequest{
		Symbol:        spotTradablePair,
		Side:          order.Buy,
		Type:          OrderType(order.Market),
		Amount:        400050.0,
		Quantity:      100,
		Price:         40000.50000,
		TimeInForce:   TimeInForce(order.GoodTillCancel),
		ClientOrderID: "1234Abcde",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelMultipleOrdersByIDs(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	if mockTests {
		t.Skip(websocketMockTestsSkipped)
	}
	e.setAPICredential(apiKey, apiSecret)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	require.True(t, e.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")

	testexch.SetupWs(t, e)
	result, err := e.WsCancelMultipleOrdersByIDs(t.Context(), []string{"1234"}, []string{"5678"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelTradeOrders(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	if mockTests {
		t.Skip(websocketMockTestsSkipped)
	}
	e.setAPICredential(apiKey, apiSecret)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	require.True(t, e.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")

	testexch.SetupWs(t, e)
	result, err := e.WsCancelTradeOrders(t.Context(), []string{"BTC_USDT", "ETH_USDT"}, []AccountType{AccountType(asset.Spot)})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Futures)
	require.NoError(t, err)

	instrument, err := e.GetFuturesProduct(t.Context(), futuresTradablePair)
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

	spotInstruments, err := e.GetSymbol(t.Context(), spotTradablePair)
	require.NoError(t, err)
	require.NotNil(t, spotInstruments)

	lms, err = e.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
	require.NoError(t, err)
	require.Len(t, spotInstruments, 1)
	require.Equal(t, lms.PriceStepIncrementSize, priceScaleMultipliers[spotInstruments[0].SymbolTradeLimit.PriceScale])
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
	"Orderbook":               `{"channel":"book","data":[{"asks":[["87854.92","9"],["87854.93","1"],["87861.31","1"],["87864.51","530"],["87866.84","21"]],"bids":[["87851.73","1"],["87851.72","1"],["87811.92","158"],["87810.84","9"],["87809.78","446"]],"id":3206990058,"ts":1765921966039,"s":"BTC_USDT_PERP","cT":1765921965951}]}`,
	"K-Line Data":             `{"channel": "candles_minute_1", "data": [ ["BTC_USDT_PERP","91883.46","91958.73","91883.46","91958.73","367.68438","4",2,1741243200000,1741243259999,1741243218348]]}`,
	"K-Line Five Min Data":    `{"channel": "candles_minute_5", "data": [ ["BTC_USDT_PERP","91883.46","91958.73","91883.46","91958.73","367.68438","4",2,1741243200000,1741243259999,1741243218348]]}`,
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
	for title, data := range futuresPushDataMap {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			err := e.wsFuturesHandleData(t.Context(), e.Websocket.Conn, []byte(data))
			assert.NoError(t, err)
		})
	}
	// Since running test in parallel shuffles the order of execution
	// We run book_lv2 data handling, ensuring the snapshot is processed before the update as follows
	err := e.wsFuturesHandleData(generateContext(t), e.Websocket.Conn, []byte(`{"channel":"book_lv2","data":[{"asks":[["87845.63","1"],["87850.4","1"],["87859.98","9"],["87866.84","21"],["87888.32","33"],["87891.40","106"],["87894.48","705"],["87897.56","238"],["87900.64","762"],["87905.16","8"],["87911.17","34"],["87913.95","4040"],["87915.13","470"],["87919.09","199"],["87922.74","2141"],["87923.05","758"],["87931.52","1568"],["87937.09","10"],["87940.31","1392"],["87940.61","64"]],"bids":[["87842.42","1"],["87842.41","1"],["87835.72","1138"],["87834.64","226"],["87833.61","446"],["87833.56","26"],["87832.48","13"],["87831.40","130"],["87807.27","69"],["87798.48","935"],["87794.18","1932"],["87791.10","296"],["87789.69","1596"],["87788.02","320"],["87784.94","131"],["87781.86","242"],["87780.90","768"],["87772.11","1737"],["87771.33","910"],["87767.37","261"]],"lid":3206980818,"id":3206980819,"ts":1765921828543,"s":"BTC_USDT_PERP","cT":1765921827839}],"action":"snapshot"}`))
	require.NoError(t, err, "Futures Orderbook Level-2 Snapshot must not error")
	err = e.wsFuturesHandleData(t.Context(), e.Websocket.Conn, []byte(`{"channel":"book_lv2","data":[{"asks":[],"bids":[["87807.27","4"]],"lid":3206980819,"id":3206980843,"ts":1765921828626,"s":"BTC_USDT_PERP","cT":1765921828619}],"action":"update"}`))
	assert.NoError(t, err, "Futures Orderbook Level-2 Update should not error")
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetAccountBalance(generateContext(t))
	require.NoError(t, err)
}

func TestGetAccountBills(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1764930174763), time.UnixMilli(1765290174763)
	_, err := e.GetAccountBills(generateContext(t), endTime, startTime, 0, 0, "NEXT", "PNL")
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = endTime.Add(-time.Hour*24), time.Now()
	}
	result, err := e.GetAccountBills(generateContext(t), startTime, endTime, 0, 0, "NEXT", "PNL")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	arg := &FuturesOrderRequest{
		ReduceOnly: true,
	}
	_, err := e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = futuresTradablePair
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	arg.MarginMode = MarginMode(margin.Multi)
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = order.Long
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = OrderType(order.LimitMaker)
	_, err = e.PlaceFuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.PlaceFuturesOrder(t.Context(), &FuturesOrderRequest{
		ClientOrderID:           "939a9d51-8f32-443a-9fb8-ff0852010487",
		Symbol:                  futuresTradablePair,
		Side:                    "buy",
		MarginMode:              MarginMode(margin.Multi),
		PositionSide:            order.Long,
		OrderType:               OrderType(order.LimitMaker),
		Price:                   46050,
		Size:                    10,
		TimeInForce:             TimeInForce(order.GoodTillCancel),
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
	_, err := e.PlaceFuturesMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{arg})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = futuresTradablePair
	_, err = e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	arg.MarginMode = MarginMode(margin.Multi)
	_, err = e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = order.Long
	_, err = e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = OrderType(order.LimitMaker)
	_, err = e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceFuturesMultipleOrders(t.Context(), []FuturesOrderRequest{
		{
			ClientOrderID:           "939a9d51",
			Symbol:                  futuresTradablePair,
			Side:                    "buy",
			MarginMode:              MarginMode(margin.Multi),
			PositionSide:            order.Long,
			OrderType:               OrderType(order.LimitMaker),
			Price:                   46050,
			Size:                    10,
			TimeInForce:             TimeInForce(order.GoodTillCancel),
			SelfTradePreventionMode: "EXPIRE_TAKER",
			ReduceOnly:              false,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesOrder(t.Context(), &CancelOrderRequest{OrderID: "1234"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.CancelFuturesOrder(t.Context(), &CancelOrderRequest{Symbol: futuresTradablePair})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.CancelFuturesOrder(generateContext(t), &CancelOrderRequest{Symbol: futuresTradablePair, OrderID: "12345"})
	require.NoError(t, err)
}

func TestCancelMultipleFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleFuturesOrders(t.Context(), &CancelFuturesOrdersRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelMultipleFuturesOrders(t.Context(), &CancelFuturesOrdersRequest{Symbol: futuresTradablePair})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleFuturesOrders(generateContext(t), &CancelFuturesOrdersRequest{Symbol: futuresTradablePair, OrderIDs: []string{"331378951169769472", "331378951182352384", "331378951199129601"}})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesOrders(t.Context(), currency.EMPTYPAIR, "BUY")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelFuturesOrders(generateContext(t), futuresTradablePair, "BUY")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCloseAtMarketPrice(t *testing.T) {
	t.Parallel()
	_, err := e.CloseAtMarketPrice(t.Context(), currency.EMPTYPAIR, "", "", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.CloseAtMarketPrice(t.Context(), futuresTradablePair, "", "", "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.CloseAtMarketPrice(generateContext(t), futuresTradablePair, "CROSS", "", "")
	require.NoError(t, err)
}

func TestCloseAllPositionsAtMarketPrice(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseAllPositionsAtMarketPrice(generateContext(t))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentFuturesOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrentFuturesOrders(generateContext(t), futuresTradablePair, "SELL", "NEXT", "", 0, 0, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderExecutionDetails(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1743615790295), time.UnixMilli(1743702190295)
	_, err := e.GetOrderExecutionDetails(generateContext(t), order.Sell, currency.EMPTYPAIR, "", "", "NEXT", endTime, startTime, 0, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour*24), time.Now()
	}
	result, err := e.GetOrderExecutionDetails(generateContext(t), order.Buy, futuresTradablePair, "331381604734197760", "polo331381602863284224", "NEXT", startTime, endTime, 1, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1743615790295), time.UnixMilli(1743702190295)
	_, err := e.GetFuturesOrderHistory(generateContext(t), currency.EMPTYPAIR, order.UnknownSide, "LIMIT", "PARTIALLY_CANCELED", "", "", "PREV", endTime, startTime, 0, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour*24), time.Now()
	}
	result, err := e.GetFuturesOrderHistory(generateContext(t), futuresTradablePair, order.Sell, "LIMIT", "PARTIALLY_CANCELED", "331381604734197760", "polo331381602863284224", "PREV", startTime, endTime, 1, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesCurrentPosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesCurrentPosition(generateContext(t), currency.EMPTYPAIR)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetFuturesCurrentPosition(generateContext(t), futuresTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositionHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1743615790295), time.UnixMilli(1743702190295)
	_, err := e.GetFuturesPositionHistory(generateContext(t), currency.EMPTYPAIR, "ISOLATED", "LONG", "NEXT", endTime, startTime, 0, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		startTime, endTime = time.Now().Add(-time.Hour*24), time.Now()
	}
	result, err := e.GetFuturesPositionHistory(generateContext(t), futuresTradablePair, "ISOLATED", "LONG", "NEXT", startTime, endTime, 1, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAdjustMarginForIsolatedMarginTradingPositions(t *testing.T) {
	t.Parallel()
	_, err := e.AdjustMarginForIsolatedMarginTradingPositions(t.Context(), currency.EMPTYPAIR, "", "ADD", 123)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.AdjustMarginForIsolatedMarginTradingPositions(t.Context(), futuresTradablePair, "", "ADD", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.AdjustMarginForIsolatedMarginTradingPositions(t.Context(), futuresTradablePair, "", "", 123)
	require.ErrorIs(t, err, errInvalidMarginAdjustType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.AdjustMarginForIsolatedMarginTradingPositions(generateContext(t), futuresTradablePair, "", "ADD", 123)
	require.NoError(t, err)
}

func TestGetFuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesLeverage(t.Context(), currency.EMPTYPAIR, "ISOLATED")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesLeverage(generateContext(t), futuresTradablePair, "ISOLATED")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetFuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetFuturesLeverage(t.Context(), currency.EMPTYPAIR, "CROSS", "LONG", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.SetFuturesLeverage(t.Context(), futuresTradablePair, "", "LONG", 10)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = e.SetFuturesLeverage(t.Context(), futuresTradablePair, "CROSS", "", 10)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.SetFuturesLeverage(t.Context(), futuresTradablePair, "CROSS", "LONG", 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.SetFuturesLeverage(generateContext(t), futuresTradablePair, "CROSS", "LONG", 10)
	require.NoError(t, err)
}

func TestSwitchPositionMode(t *testing.T) {
	t.Parallel()
	err := e.SwitchPositionMode(t.Context(), "")
	require.ErrorIs(t, err, errInvalidPositionMode)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.SwitchPositionMode(generateContext(t), "HEDGE")
	require.NoError(t, err)
}

func TestGetPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPositionMode(generateContext(t))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserPositionRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserPositionRiskLimit(t.Context(), currency.EMPTYPAIR, "CROSS", "LONG")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserPositionRiskLimit(t.Context(), futuresTradablePair, "CROSS", "LONG")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderBook(t.Context(), currency.EMPTYPAIR, 100, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesOrderBook(t.Context(), futuresTradablePair, 100, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesKlineData(t.Context(), currency.EMPTYPAIR, kline.FiveMin, time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	startTime, endTime := time.UnixMilli(1743615790295), time.UnixMilli(1743702190295)
	_, err = e.GetFuturesKlineData(t.Context(), futuresTradablePair, kline.FiveMin, endTime, startTime, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	_, err = e.GetFuturesKlineData(t.Context(), futuresTradablePair, kline.HundredMilliseconds, startTime, endTime, 100)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	_, err = e.GetFuturesKlineData(t.Context(), futuresTradablePair, kline.HundredMilliseconds, endTime, startTime, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		startTime, endTime = endTime.Add(-time.Hour*24), time.Now()
	}
	result, err := e.GetFuturesKlineData(t.Context(), futuresTradablePair, kline.FiveMin, startTime, endTime, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesExecution(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesExecution(t.Context(), currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesExecution(t.Context(), futuresTradablePair, 5)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLiquidationOrder(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1764664401864), time.UnixMilli(1764668001864)
	_, err := e.GetLiquidationOrder(t.Context(), futuresTradablePair, "NEXT", endTime, startTime, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		startTime, endTime = time.Now().Add(-time.Hour), time.Now()
	}
	result, err := e.GetLiquidationOrder(t.Context(), futuresTradablePair, "NEXT", startTime, endTime, 1, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesMarket(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesMarket(t.Context(), futuresTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesIndexPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesIndexPrice(t.Context(), futuresTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesIndexPrices(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesIndexPrices(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceComponents(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceComponents(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetIndexPriceComponents(t.Context(), futuresTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstrumentsIndexPriceComponents(t *testing.T) {
	t.Parallel()
	result, err := e.GetAllIndexPriceComponents(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceKlineData(t.Context(), currency.EMPTYPAIR, kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetIndexPriceKlineData(t.Context(), futuresTradablePair, kline.HundredMilliseconds, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	startTime, endTime := time.UnixMilli(1764930174763), time.UnixMilli(1765290174763)
	_, err = e.GetIndexPriceKlineData(t.Context(), futuresTradablePair, kline.FourHour, endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		startTime, endTime = endTime.Add(-time.Hour*24), time.Now()
	}
	result, err := e.GetIndexPriceKlineData(t.Context(), futuresTradablePair, kline.FourHour, startTime, endTime, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesMarkPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesMarkPrice(t.Context(), futuresTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesMarkPrices(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesMarkPrices(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPriceKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceKlineData(t.Context(), currency.EMPTYPAIR, kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetMarkPriceKlineData(t.Context(), futuresTradablePair, kline.HundredMilliseconds, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	startTime, endTime := time.UnixMilli(1764930174763), time.UnixMilli(1765290174763)
	_, err = e.GetMarkPriceKlineData(t.Context(), futuresTradablePair, kline.FourHour, endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		startTime, endTime = endTime.Add(-time.Hour*24), time.Now()
	}
	result, err := e.GetMarkPriceKlineData(t.Context(), futuresTradablePair, kline.FourHour, startTime, endTime, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesProduct(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesProduct(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesProduct(t.Context(), futuresTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesCurrentFundingRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCurrentFundingRate(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesCurrentFundingRate(t.Context(), futuresTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesHistoricalFundingRates(t.Context(), currency.EMPTYPAIR, time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	startTime, endTime := time.UnixMilli(1743615790295), time.UnixMilli(1743702190295)
	_, err = e.GetFuturesHistoricalFundingRates(t.Context(), futuresTradablePair, endTime, startTime, 100)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		startTime, endTime = endTime.Add(-time.Hour*24), time.Now()
	}
	result, err := e.GetFuturesHistoricalFundingRates(t.Context(), futuresTradablePair, startTime, endTime, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractOpenInterest(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractOpenInterest(t.Context(), futuresTradablePair)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInsuranceFund(t *testing.T) {
	t.Parallel()
	result, err := e.GetInsuranceFund(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesRiskLimit(t.Context(), currency.EMPTYPAIR, "", 1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesRiskLimit(t.Context(), futuresTradablePair, "", 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetFuturesRiskLimit(t.Context(), futuresTradablePair, "CROSS", 1)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractLimitPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractLimitPrice(t.Context(), []currency.Pair{currency.EMPTYPAIR})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetContractLimitPrice(t.Context(), []currency.Pair{currency.NewPairWithDelimiter("DOT", "USDT_PERP", "_"), currency.EMPTYPAIR, futuresTradablePair})
	assert.NoError(t, err)
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
		kline.TenMin:     {IntervalString: "MINUTE_10"},
		kline.FifteenMin: {IntervalString: "MINUTE_15"},
		kline.ThirtyMin:  {IntervalString: "MINUTE_30"},
		kline.OneHour:    {IntervalString: "HOUR_1"},
		kline.TwoHour:    {IntervalString: "HOUR_2"},
		kline.FourHour:   {IntervalString: "HOUR_4"},
		kline.SixHour:    {IntervalString: "HOUR_6"},
		kline.EightHour:  {IntervalString: "HOUR_8"},
		kline.TwelveHour: {IntervalString: "HOUR_12"},
		kline.OneDay:     {IntervalString: "DAY_1"},
		kline.ThreeDay:   {IntervalString: "DAY_3"},
		kline.OneWeek:    {IntervalString: "WEEK_1"},
		kline.OneMonth:   {IntervalString: "MONTH_1"},
		kline.TwoWeek:    {Error: kline.ErrUnsupportedInterval},
	}
	for key, val := range params {
		s, err := intervalToString(key)
		require.Equal(t, val.IntervalString, s)
		require.ErrorIs(t, err, val.Error, err)
	}
}

func TestOrderStateString(t *testing.T) {
	t.Parallel()
	orderStatusToStringMap := map[string]order.Status{
		"NEW":                order.New,
		"FAILED":             order.Rejected,
		"FILLED":             order.Filled,
		"CANCELED":           order.Cancelled,
		"PENDING_Cancel":     order.PendingCancel,
		"abcd":               order.UnknownStatus,
		"PARTIALLY_FILLED":   order.PartiallyFilled,
		"PARTIALLY_CANCELED": order.PartiallyCancelled,
	}
	for k, v := range orderStatusToStringMap {
		result := orderStateFromString(k)
		assert.Equal(t, v, result)
	}
}

func generateContext(tb testing.TB) context.Context {
	tb.Helper()
	if mockTests {
		return accounts.DeployCredentialsToContext(tb.Context(), &accounts.Credentials{
			Key:    "abcde",
			Secret: "fghij",
		})
	}
	return tb.Context()
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

func TestHandleFuturesSubscriptions(t *testing.T) {
	t.Parallel()
	enabledPairs, err := e.GetEnabledPairs(asset.Futures)
	require.NoError(t, err)

	subscs, err := subscription.List{
		{
			Asset:   asset.Futures,
			Channel: subscription.TickerChannel,
			Pairs:   enabledPairs,
		},
		{
			Asset:   asset.Futures,
			Channel: subscription.OrderbookChannel,
			Pairs:   enabledPairs,
		},
	}.ExpandTemplates(e)
	require.NoError(t, err)

	payloads := []*SubscriptionPayload{
		{Event: "subscribe", Channel: []string{"tickers"}, Symbols: enabledPairs.Strings()},
		{Event: "subscribe", Channel: []string{"book_lv2"}, Symbols: enabledPairs.Strings()},
	}
	for i, s := range subscs {
		result, err := e.handleSubscription("subscribe", s)
		require.NoError(t, err)
		require.Equal(t, payloads[i], result)
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

var channelIntervals = []*struct {
	input    string
	channel  string
	interval kline.Interval
	err      error
}{
	{input: "mark_candles", channel: "mark_candles", err: kline.ErrInvalidInterval},
	{input: "mark_candles_hour_1", channel: "mark_candles", interval: kline.OneHour},
	{input: "mark_price_candles_minute_1", channel: "mark_price_candles", interval: kline.OneMin},
	{input: "mark_candles_minute_30", channel: "mark_candles", interval: kline.ThirtyMin},
	{input: "index_candles_hour_4", channel: "index_candles", interval: kline.FourHour},
	{input: "candles_minute_30", channel: "candles", interval: kline.ThirtyMin},
	{input: "candles_minute_15", channel: "candles", interval: kline.FifteenMin},
	{input: "candles_minute_10", channel: "candles", interval: kline.TenMin},
	{input: "candles_minute_5", channel: "candles", interval: kline.FiveMin},
	{input: "mark_candles_day_3", channel: "mark_candles", interval: kline.ThreeDay},
	{input: "mark_candles_week_1", channel: "mark_candles", interval: kline.OneWeek},
	{input: "mark_candles_hour_abc", channel: "mark_candles", interval: kline.Interval(0), err: kline.ErrUnsupportedInterval},
}

func TestChannelToIntervalSplit(t *testing.T) {
	t.Parallel()
	for _, chd := range channelIntervals {
		t.Run(chd.input, func(t *testing.T) {
			t.Parallel()
			c, i, err := channelToIntervalSplit(chd.input)
			require.ErrorIs(t, err, chd.err)
			require.Equal(t, chd.channel, c)
			assert.Equal(t, chd.interval, i)
		})
	}
}

func TestStatusResponseError(t *testing.T) {
	t.Parallel()
	var p *OrderIDResponse
	require.NoError(t, json.Unmarshal([]byte(`{"id": "4"}`), &p))
	require.NoError(t, json.Unmarshal([]byte(`{"id": "4","code":200}`), &p))
	require.NoError(t, p.Error())
	require.NoError(t, json.Unmarshal([]byte(`{"id": "4","code":400,"message":"this works"}`), &p))
	err, ok := any(p).(interface{ Error() error })
	require.True(t, ok)
	require.ErrorContains(t, err.Error(), "this works")
}

func TestConnect(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip(websocketMockTestsSkipped)
	}
	require.NoError(t, e.Websocket.Connect())
	assert.True(t, e.Websocket.IsConnected(), "websocket should be connected")
}

func TestWebsocketSliceErrorCheck(t *testing.T) {
	t.Parallel()
	results := []struct {
		in       string
		hasError bool
		sliceLen int
	}{
		{in: `{"data":[{ "orderId": 205343650954092544, "clientOrderId": "", "message": "", "code": 200 }]}`, sliceLen: 1},
		{in: `{ "id": "123457", "data": [{ "orderId": 0, "clientOrderId": null, "message": "Currency trade disabled", "code": 21352 }] }`, hasError: true, sliceLen: 1},
		{in: `{ "id": "123457", "data": [{ "orderId": 205343650954092544, "clientOrderId": "", "message": "", "code": 200 }, { "orderId": 0, "clientOrderId": null, "message": "Currency trade disabled", "code": 21352 }] }`, hasError: true, sliceLen: 2},
	}

	response := []*WsCancelOrderResponse{}
	for _, elem := range results {
		require.NoError(t, json.Unmarshal([]byte(elem.in), &WebsocketResponse{Data: &response}))
		assert.NotNil(t, response)
		assert.Len(t, response, elem.sliceLen)
		if elem.hasError {
			assert.Error(t, checkForErrorInSliceResponse(response))
		} else {
			assert.NoError(t, checkForErrorInSliceResponse(response))
		}
	}
}
