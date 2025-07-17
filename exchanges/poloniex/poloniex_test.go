package poloniex

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
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

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	result, err := e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(e) {
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
		Type:      order.Limit,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
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
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOrderHistory(generateContext(), &order.MultiOrderRequest{
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.NotNil(t, result)
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

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		AssetType: asset.Options,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	arg := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      spotTradablePair,
		AssetType: asset.Spot,
	}
	e.Verbose = true
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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
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
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Amount = 1000
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.LTC
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	withdrawCryptoRequest := withdraw.Request{
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
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
	require.NotNil(t, result)

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
	require.NotNil(t, result)

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
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.UpdateTicker(t.Context(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateTicker(t.Context(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
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
	require.ErrorIs(t, err, errOrderAssetTypeMismatch)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, OrderID: "1233"}, {AssetType: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{Pair: futuresTradablePair, AssetType: asset.Futures, OrderID: "1233"}, {OrderID: "1233", AssetType: asset.Futures, Pair: spotTradablePair}})
	require.ErrorIs(t, err, errPairStringMismatch)

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
			Pair:      currency.NewPair(currency.BTC, currency.USD),
		},
		{
			OrderID:   "234",
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.BTC, currency.USD),
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)
	require.NotZero(t, st)

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

func TestGetSymbolInformation(t *testing.T) {
	t.Parallel()
	result, err := e.GetSymbolInformation(t.Context(), spotTradablePair)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.GetSymbolInformation(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrenciesInformation(t *testing.T) {
	t.Parallel()
	result, err := e.GetCurrenciesInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV2CurrencyInformation(t *testing.T) {
	t.Parallel()
	result, err := e.GetV2CurrencyInformation(t.Context())
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
	_, err := e.MarkPriceComponents(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := e.MarkPriceComponents(t.Context(), spotTradablePair)
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
	require.NotNil(t, result)

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
	result, err := e.GetCollateralInfos(t.Context())
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
	_, err := e.GetAllBalance(t.Context(), "", "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllBalance(generateContext(), "329455537441832960", "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.GetAllBalance(generateContext(), "329455537441832960", "SPOT")
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
	_, err := e.AccountsTransfer(t.Context(), &AccountTransferParams{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = e.AccountsTransfer(t.Context(), &AccountTransferParams{Amount: 1232.221})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.AccountsTransfer(t.Context(), &AccountTransferParams{
		Ccy: currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = e.AccountsTransfer(t.Context(), &AccountTransferParams{
		Amount:      1,
		Ccy:         currency.BTC,
		FromAccount: "219961623421431808",
	})
	require.ErrorIs(t, err, errAddressRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AccountsTransfer(generateContext(), &AccountTransferParams{
		Amount:      1,
		Ccy:         currency.BTC,
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
	_, err := e.GetAccountTransferRecords(generateContext(), time.Time{}, time.Time{}, "", currency.BTC, 0, 0)
	require.NoError(t, err)
}

func TestGetAccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountTransferRecord(generateContext(), "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetAccountTransferRecord(generateContext(), "329455537441832960")
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
	_, err := e.SubAccountTransfer(t.Context(), &SubAccountTransferParam{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = e.SubAccountTransfer(t.Context(), &SubAccountTransferParam{Amount: 12.34})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SubAccountTransfer(t.Context(), &SubAccountTransferParam{
		Currency: currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = e.SubAccountTransfer(t.Context(), &SubAccountTransferParam{
		Currency: currency.BTC,
		Amount:   1,
	})
	require.ErrorIs(t, err, errAccountIDRequired)
	_, err = e.SubAccountTransfer(t.Context(), &SubAccountTransferParam{
		Currency:      currency.BTC,
		Amount:        1,
		FromAccountID: "1234568",
		ToAccountID:   "1234567",
	})
	require.ErrorIs(t, err, errAccountTypeRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubAccountTransfer(generateContext(), &SubAccountTransferParam{
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
	_, err := e.GetSubAccountTransferRecords(generateContext(), currency.BTC, time.Time{}, time.Time{}, "", "", "", "", "", 0, 0)
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

func TestNewCurrencyDepoditAddress(t *testing.T) {
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
	_, err := e.WithdrawCurrency(t.Context(), &WithdrawCurrencyParam{})
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.WithdrawCurrency(t.Context(), &WithdrawCurrencyParam{
		Currency: currency.BTC.String() + "TRON", // Sends BTC through the TRON chain
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = e.WithdrawCurrency(t.Context(), &WithdrawCurrencyParam{Currency: currency.BTC.String(), Amount: 1})
	require.ErrorIs(t, err, errAddressRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WithdrawCurrency(generateContext(), &WithdrawCurrencyParam{Currency: currency.BTC.String(), Amount: 1, Address: "0xbb8d0d7c346daecc2380dabaa91f3ccf8ae232fb4"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCurrencyV2(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCurrencyV2(t.Context(), &WithdrawCurrencyV2Param{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = e.WithdrawCurrencyV2(t.Context(), &WithdrawCurrencyV2Param{Coin: currency.BTC})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = e.WithdrawCurrencyV2(t.Context(), &WithdrawCurrencyV2Param{Coin: currency.BTC, Amount: 1})
	require.ErrorIs(t, err, errInvalidWithdrawalChain)
	_, err = e.WithdrawCurrencyV2(t.Context(), &WithdrawCurrencyV2Param{Coin: currency.BTC, Amount: 1, Network: "BTC"})
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawCurrencyV2(t.Context(), &WithdrawCurrencyV2Param{Network: "BTC", Coin: currency.BTC, Amount: 1, Address: "0xbb8d0d7c346daecc2380dabaa91f3ccf8ae232fb4"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountMarginInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountMarginInformation(generateContext(), "SPOT")
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
	_, err := e.PlaceOrder(t.Context(), &PlaceOrderParams{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderParams{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderParams{Symbol: spotTradablePair})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceOrder(t.Context(), &PlaceOrderParams{
		Symbol:        spotTradablePair,
		Side:          order.Buy.String(),
		Type:          order.Market.String(),
		Quantity:      100,
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
	require.ErrorIs(t, err, errNilArgument)
	_, err = e.PlaceBatchOrders(t.Context(), []PlaceOrderParams{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.PlaceBatchOrders(t.Context(), []PlaceOrderParams{
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
	result, err := e.PlaceBatchOrders(t.Context(), []PlaceOrderParams{
		{
			Symbol:        pair,
			Side:          order.Buy.String(),
			Type:          order.Market.String(),
			Quantity:      100,
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
	_, err := e.CancelReplaceOrder(t.Context(), &CancelReplaceOrderParam{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = e.CancelReplaceOrder(t.Context(), &CancelReplaceOrderParam{
		TimeInForce: "GTC",
	})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelReplaceOrder(t.Context(), &CancelReplaceOrderParam{
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
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderDetail(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrderDetail(generateContext(), "12345536545645", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrderByID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelOrderByID(t.Context(), "12345536545645")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleOrdersByIDs(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleOrdersByIDs(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = e.CancelMultipleOrdersByIDs(t.Context(), &OrderCancellationParams{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelMultipleOrdersByIDs(t.Context(), &OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllTradeOrders(t.Context(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKillSwitch(t *testing.T) {
	t.Parallel()
	_, err := e.KillSwitch(t.Context(), "")
	require.ErrorIs(t, err, errInvalidTimeout)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.KillSwitch(generateContext(), "30")
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
	_, err := e.CreateSmartOrder(t.Context(), &SmartOrderRequestParam{})
	require.ErrorIs(t, err, errNilArgument)

	_, err = e.CreateSmartOrder(t.Context(), &SmartOrderRequestParam{Side: "BUY"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.CreateSmartOrder(t.Context(), &SmartOrderRequestParam{Symbol: spotTradablePair})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateSmartOrder(generateContext(), &SmartOrderRequestParam{
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
}

func TestCancelReplaceSmartOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelReplaceSmartOrder(t.Context(), &CancelReplaceSmartOrderParam{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = e.CancelReplaceSmartOrder(t.Context(), &CancelReplaceSmartOrderParam{Price: 18000})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelReplaceSmartOrder(t.Context(), &CancelReplaceSmartOrderParam{
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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelSmartOrderByID(t.Context(), "123313413", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleSmartOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleSmartOrders(t.Context(), nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = e.CancelMultipleSmartOrders(t.Context(), &OrderCancellationParams{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelMultipleSmartOrders(t.Context(), &OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllSmartOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllSmartOrders(t.Context(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrdersHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetOrdersHistory(generateContext(), spotTradablePair, "SPOT", "", "", "", "", 0, 10, time.Time{}, time.Time{}, false)
	require.NoError(t, err)
}

func TestGetSmartOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err := e.GetSmartOrderHistory(generateContext(), spotTradablePair, "SPOT", "", "", "", "", 0, 10, time.Time{}, time.Time{}, false)
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

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	result, err := e.GenerateDefaultSubscriptions()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHandlePayloads(t *testing.T) {
	t.Parallel()
	subscriptions, err := e.GenerateDefaultSubscriptions()
	require.NoError(t, err)
	require.NotEmpty(t, subscriptions)

	result, err := e.handleSubscriptions("subscribe", subscriptions)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

var pushMessages = map[string]string{
	"AccountBalance": `{ "channel": "balances", "data": [{ "changeTime": 1657312008411, "accountId": "1234", "accountType": "SPOT", "eventType": "place_order", "available": "9999999983.668", "currency": "BTC", "id": 60018450912695040, "userId": 12345, "hold": "16.332", "ts": 1657312008443 }] }`,
	"Orders":         `{ "channel": "orders", "data": [ { "symbol": "BTC_USDC", "type": "LIMIT", "quantity": "1", "orderId": "32471407854219264", "tradeFee": "0", "clientOrderId": "", "accountType": "SPOT", "feeCurrency": "", "eventType": "place", "source": "API", "side": "BUY", "filledQuantity": "0", "filledAmount": "0", "matchRole": "MAKER", "state": "NEW", "tradeTime": 0, "tradeAmount": "0", "orderAmount": "0", "createTime": 1648708186922, "price": "47112.1", "tradeQty": "0", "tradePrice": "0", "tradeId": "0", "ts": 1648708187469 } ] }`,
	"Candles":        `{"channel":"candles_minute_5","data":[{"symbol":"BTC_USDT","open":"25143.19","high":"25148.58","low":"25138.76","close":"25144.55","quantity":"0.860454","amount":"21635.20983974","tradeCount":20,"startTime":1694469000000,"closeTime":1694469299999,"ts":1694469049867}]}`,
	"BooksLV2":       `{"channel":"book_lv2","data":[{"symbol":"BTC_USDC","createTime":1694469187745,"asks":[],"bids":[["25148.81","0.02158"],["25088.11","0"]],"lastId":598273385,"id":598273386,"ts":1694469187760}],"action":"update"}`,
	"Books":          `{"channel":"book","data":[{"symbol":"BTC_USDC","createTime":1694469187686,"asks":[["25157.24","0.444294"],["25157.25","0.024357"],["25157.26","0.003204"],["25163.39","0.039476"],["25163.4","0.110047"]],"bids":[["25148.8","0.00692"],["25148.61","0.021581"],["25148.6","0.034504"],["25148.59","0.065405"],["25145.52","0.79537"]],"id":598273384,"ts":1694469187733}]}`,
	"Tickers":        `{"channel":"ticker","data":[{"symbol":"BTC_USDC","startTime":1694382780000,"open":"25866.3","high":"26008.47","low":"24923.65","close":"25153.02","quantity":"1626.444884","amount":"41496808.63699303","tradeCount":37124,"dailyChange":"-0.0276","markPrice":"25154.9","closeTime":1694469183664,"ts":1694469187081}]}`,
	"Trades":         `{"channel":"trades","data":[{"symbol":"BTC_USDC","amount":"52.821342","quantity":"0.0021","takerSide":"sell","createTime":1694469183664,"price":"25153.02","id":"71076055","ts":1694469183673}]}`,
	"Currencies":     `{"channel":"currencies","data":[[{"currency":"BTC","id":28,"name":"Bitcoin","description":"BTC Clone","type":"address","withdrawalFee":"0.0008","minConf":2,"depositAddress":null,"blockchain":"BTC","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["BTCTRON"]},{"currency":"XRP","id":243,"name":"XRP","description":"Payment ID","type":"address-payment-id","withdrawalFee":"0.2","minConf":2,"depositAddress":"rwU8rAiE2eyEPz3sikfbHuqCuiAtdXqa2v","blockchain":"XRP","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":false,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":[]},{"currency":"ETH","id":267,"name":"Ethereum","description":"Sweep to Main Account","type":"address","withdrawalFee":"0.00197556","minConf":64,"depositAddress":null,"blockchain":"ETH","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["ETHTRON"]},{"currency":"USDT","id":214,"name":"Tether USD","description":"Sweep to Main Account","type":"address","withdrawalFee":"0","minConf":2,"depositAddress":null,"blockchain":"OMNI","delisted":false,"tradingState":"NORMAL","walletState":"DISABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["USDTETH","USDTTRON"]},{"currency":"DOGE","id":59,"name":"Dogecoin","description":"BTC Clone","type":"address","withdrawalFee":"20","minConf":6,"depositAddress":null,"blockchain":"DOGE","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["DOGETRON"]},{"currency":"LTC","id":125,"name":"Litecoin","description":"BTC Clone","type":"address","withdrawalFee":"0.001","minConf":4,"depositAddress":null,"blockchain":"LTC","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":true,"isChildChain":false,"supportCollateral":true,"supportBorrow":true,"childChains":["LTCTRON"]},{"currency":"DASH","id":60,"name":"Dash","description":"BTC Clone","type":"address","withdrawalFee":"0.01","minConf":20,"depositAddress":null,"blockchain":"DASH","delisted":false,"tradingState":"NORMAL","walletState":"ENABLED","parentChain":null,"isMultiChain":false,"isChildChain":false,"supportCollateral":false,"supportBorrow":false,"childChains":[]}]],"action":"snapshot"}`,
	"Symbols":        `{"channel":"symbols","data":[[{"symbol":"BTC_USDC","baseCurrencyName":"BTC","quoteCurrencyName":"USDT","displayName":"BTC/USDT","state":"NORMAL","visibleStartTime":1659018819512,"tradableStartTime":1659018819512,"crossMargin":{"supportCrossMargin":true,"maxLeverage":"3"},"symbolTradeLimit":{"symbol":"BTC_USDT","priceScale":2,"quantityScale":6,"amountScale":2,"minQuantity":"0.000001","minAmount":"1","highestBid":"0","lowestAsk":"0"}}]],"action":"snapshot"}`,
}

func TestWsPushData(t *testing.T) {
	t.Parallel()
	for key, value := range pushMessages {
		err := e.wsHandleData([]byte(value))
		assert.NoErrorf(t, err, "%s error %s: %v", e.Name, key, err)
	}
}

func TestWsCreateOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsCreateOrder(&PlaceOrderParams{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = e.WsCreateOrder(&PlaceOrderParams{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.WsCreateOrder(&PlaceOrderParams{
		Symbol: spotTradablePair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WsCreateOrder(&PlaceOrderParams{
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
	if mockTests {
		t.SkipNow()
	}
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	testexch.SetupWs(t, e)
	result, err := e.WsCancelMultipleOrdersByIDs(&OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	testexch.SetupWs(t, e)
	result, err := e.WsCancelAllTradeOrders([]string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Futures)
	require.NoError(t, err)

	e.Verbose = true
	instrument, err := e.GetV3FuturesProductInfo(t.Context(), futuresTradablePair.String())
	require.NoError(t, err)
	require.NotNil(t, instrument)

	limits, err := e.GetOrderExecutionLimits(asset.Futures, futuresTradablePair)
	require.NoError(t, err)
	require.NotNil(t, limits)
	require.Equal(t, limits.PriceStepIncrementSize, instrument.TickSize.Float64())
	require.Equal(t, limits.MinimumBaseAmount, instrument.MinQuantity.Float64())
	assert.Equal(t, limits.MinimumQuoteAmount, instrument.MinSize.Float64())

	// sample test for spot instrument order execution limit

	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	require.NoError(t, err)

	spotInstruments, err := e.GetSymbolInformation(t.Context(), spotTradablePair)
	require.NoError(t, err)
	require.NotNil(t, instrument)

	limits, err = e.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
	require.NoError(t, err)
	require.Len(t, spotInstruments, 1)
	require.Equal(t, limits.PriceStepIncrementSize, spotInstruments[0].SymbolTradeLimit.PriceScale)
	require.Equal(t, limits.MinimumBaseAmount, spotInstruments[0].SymbolTradeLimit.MinQuantity.Float64())
	assert.Equal(t, limits.MinimumQuoteAmount, spotInstruments[0].SymbolTradeLimit.MinAmount.Float64())
}

// ---- Futures endpoints ---

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		assert.NoError(t, err, "cannot get pairs for %s", a)
		assert.NotEmpty(t, pairs, "no pairs for %s", a)

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
		err = e.wsFuturesHandleData([]byte(data))
		assert.NoErrorf(t, err, "%s: unexpected error %v", title, err)
	}
}

func TestGetCurrencyInformation(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyInformation(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.GetCurrencyInformation(t.Context(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
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

func TestPlaceV3FuturesOrder(t *testing.T) {
	t.Parallel()
	arg := &FuturesParams{}
	_, err := e.PlaceV3FuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.ReduceOnly = true
	_, err = e.PlaceV3FuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTC_USDT_PERP"
	_, err = e.PlaceV3FuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	_, err = e.PlaceV3FuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "LONG"
	_, err = e.PlaceV3FuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit_maker"
	_, err = e.PlaceV3FuturesOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.PlaceV3FuturesOrder(t.Context(), &FuturesParams{
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
	arg := FuturesParams{}
	_, err := e.PlaceV3FuturesMultipleOrders(t.Context(), []FuturesParams{arg})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.ReduceOnly = true
	_, err = e.PlaceV3FuturesMultipleOrders(t.Context(), []FuturesParams{arg})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTC_USDT_PERP"
	_, err = e.PlaceV3FuturesMultipleOrders(t.Context(), []FuturesParams{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	_, err = e.PlaceV3FuturesMultipleOrders(t.Context(), []FuturesParams{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "LONG"
	_, err = e.PlaceV3FuturesMultipleOrders(t.Context(), []FuturesParams{arg})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit_maker"
	_, err = e.PlaceV3FuturesMultipleOrders(t.Context(), []FuturesParams{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceV3FuturesMultipleOrders(t.Context(), []FuturesParams{
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

func TestCancelV3FuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelV3FuturesOrder(t.Context(), &CancelOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	e.Verbose = true
	_, err = e.CancelV3FuturesOrder(t.Context(), &CancelOrderParams{OrderID: "1234"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.CancelV3FuturesOrder(t.Context(), &CancelOrderParams{Symbol: futuresTradablePair.String()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.CancelV3FuturesOrder(generateContext(), &CancelOrderParams{Symbol: futuresTradablePair.String(), OrderID: "12345"})
	require.NoError(t, err)
}

func TestCancelMultipleV3FuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleV3FuturesOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.CancelMultipleV3FuturesOrders(t.Context(), &CancelOrdersParams{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleV3FuturesOrders(generateContext(), &CancelOrdersParams{Symbol: futuresTradablePair.String()})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllV3FuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllV3FuturesOrders(t.Context(), "", "BUY")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllV3FuturesOrders(generateContext(), futuresTradablePair.String(), "BUY")
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
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	_, err = e.CloseAtMarketPrice(generateContext(), futuresTradablePair.String(), "CROSS", "", "123123")
	require.NoError(t, err)
}

func TestCloseAllAtMarketPrice(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	_, err := e.CloseAllAtMarketPrice(generateContext())
	require.NoError(t, err)
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

func TestGetV3FuturesOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetV3FuturesOrderHistory(generateContext(), "", "LIMIT", "", "PARTIALLY_CANCELED", "", "", "PREV", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesCurrentPosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetV3FuturesCurrentPosition(generateContext(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesPositionHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetV3FuturesPositionHistory(generateContext(), "", "ISOLATED", "LONG", "NEXT", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAdjustMarginForIsolatedMarginTradingPositions(t *testing.T) {
	t.Parallel()
	_, err := e.AdjustMarginForIsolatedMarginTradingPositions(t.Context(), "", "", "ADD", 123)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.AdjustMarginForIsolatedMarginTradingPositions(t.Context(), "DOT_USDT_PERP", "", "ADD", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = e.AdjustMarginForIsolatedMarginTradingPositions(t.Context(), "DOT_USDT_PERP", "", "", 123)
	require.ErrorIs(t, err, errMarginAdjustTypeMissing)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.AdjustMarginForIsolatedMarginTradingPositions(generateContext(), "BTC_USDT_PERP", "", "ADD", 123)
	require.NoError(t, err)
}

func TestGetV3FuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetV3FuturesLeverage(t.Context(), "", "ISOLATED")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetV3FuturesLeverage(generateContext(), "BTC_USDT_PERP", "ISOLATED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetV3FuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetV3FuturesLeverage(t.Context(), "", "CROSS", "LONG", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.SetV3FuturesLeverage(t.Context(), "BTC_USDT_PERP", "", "LONG", 10)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = e.SetV3FuturesLeverage(t.Context(), "BTC_USDT_PERP", "CROSS", "", 10)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.SetV3FuturesLeverage(t.Context(), "BTC_USDT_PERP", "CROSS", "LONG", 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.SetV3FuturesLeverage(generateContext(), "BTC_USDT_PERP", "CROSS", "LONG", 10)
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
	_, err := e.GetV3FuturesOrderBook(t.Context(), "", 100, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetV3FuturesOrderBook(t.Context(), "BTC_USDT_PERP", 100, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetV3FuturesKlineData(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetV3FuturesKlineData(t.Context(), "BTC_USDT_PERP", kline.SixHour, time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetV3FuturesKlineData(t.Context(), "BTC_USDT_PERP", kline.FiveMin, time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesExecutionInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetV3FuturesExecutionInfo(t.Context(), "", 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetV3FuturesExecutionInfo(t.Context(), "BTC_USDT_PERP", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3LiquidationOrder(t *testing.T) {
	t.Parallel()
	result, err := e.GetV3LiquidiationOrder(t.Context(), "BTC_USDT_PERP", "NEXT", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesMarketInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetV3FuturesMarketInfo(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesIndexPrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetV3FuturesIndexPrice(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3IndexPriceComponents(t *testing.T) {
	t.Parallel()
	_, err := e.GetV3IndexPriceComponents(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetV3IndexPriceComponents(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceKlineData(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetIndexPriceKlineData(t.Context(), "BTC_USDT_PERP", kline.SixHour, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetIndexPriceKlineData(t.Context(), "BTC_USDT_PERP", kline.FourHour, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetV3FuturesMarkPrice(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPriceKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceKlineData(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = e.GetMarkPriceKlineData(t.Context(), "BTC_USDT_PERP", kline.SixHour, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetMarkPriceKlineData(t.Context(), "BTC_USDT_PERP", kline.FourHour, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesProductInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetV3FuturesProductInfo(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetV3FuturesProductInfo(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesCurrentFundingRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetV3FuturesCurrentFundingRate(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetV3FuturesCurrentFundingRate(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	result, err := e.GetV3FuturesHistoricalFundingRates(t.Context(), "", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesCurrentOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetV3FuturesCurrentOpenPositions(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetV3FuturesCurrentOpenPositions(t.Context(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInsuranceFundInformation(t *testing.T) {
	t.Parallel()
	result, err := e.GetInsuranceFundInformation(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesRiskLimit(t *testing.T) {
	t.Parallel()
	result, err := e.GetV3FuturesRiskLimit(t.Context(), "")
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
	var err error
	var is string
	for key, val := range params {
		is, err = IntervalString(key)
		require.Equal(t, val.IntervalString, is)
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
	ctx := context.Background()
	if mockTests {
		credStore := (&account.ContextCredentialsStore{})
		credStore.Load(&account.Credentials{
			Key:    "abcde",
			Secret: "fghij",
		})
		ctx = context.WithValue(ctx, account.ContextCredentialsFlag, credStore)
	}
	return ctx
}
