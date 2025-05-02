package poloniex

import (
<<<<<<< HEAD
	"context"
=======
	"errors"
	"net/http"
	"strings"
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
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

<<<<<<< HEAD
var (
	p                                     = &Poloniex{}
	spotTradablePair, futuresTradablePair currency.Pair
)
=======
var p = &Poloniex{}

func TestTimestamp(t *testing.T) {
	t.Parallel()
	_, err := p.GetTimestamp(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := p.GetTicker(t.Context())
	if err != nil {
		t.Error("Poloniex GetTicker() error", err)
	}
}

func TestGetVolume(t *testing.T) {
	t.Parallel()
	_, err := p.GetVolume(t.Context())
	if err != nil {
		t.Error("Test failed - Poloniex GetVolume() error")
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := p.GetOrderbook(t.Context(), "BTC_XMR", 50)
	if err != nil {
		t.Error("Test failed - Poloniex GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := p.GetTradeHistory(t.Context(), "BTC_XMR", 0, 0)
	if err != nil {
		t.Error("Test failed - Poloniex GetTradeHistory() error", err)
	}
}

func TestGetChartData(t *testing.T) {
	t.Parallel()
	_, err := p.GetChartData(t.Context(),
		"BTC_XMR",
		time.Unix(1405699200, 0), time.Unix(1405699400, 0), "300")
	if err != nil {
		t.Error("Test failed - Poloniex GetChartData() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := p.GetCurrencies(t.Context())
	if err != nil {
		t.Error("Test failed - Poloniex GetCurrencies() error", err)
	}
}

func TestGetLoanOrders(t *testing.T) {
	t.Parallel()
	_, err := p.GetLoanOrders(t.Context(), "BTC")
	if err != nil {
		t.Error("Test failed - Poloniex GetLoanOrders() error", err)
	}
}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651

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
<<<<<<< HEAD
	result, err := p.GetFeeByType(context.Background(), feeBuilder)
	require.NoError(t, err)

=======
	_, err := p.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	if !sharedtestvalues.AreAPICredentialsSet(p) {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		assert.NotNil(t, result)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()

	if sharedtestvalues.AreAPICredentialsSet(p) || mockTests {
		// CryptocurrencyTradeFee Basic
<<<<<<< HEAD
		if _, err := p.GetFee(generateContext(), feeBuilder); err != nil {
=======
		if _, err := p.GetFee(t.Context(), feeBuilder); err != nil {
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
<<<<<<< HEAD
		if _, err := p.GetFee(generateContext(), feeBuilder); err != nil {
=======
		if _, err := p.GetFee(t.Context(), feeBuilder); err != nil {
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
<<<<<<< HEAD
		if _, err := p.GetFee(generateContext(), feeBuilder); err != nil {
=======
		if _, err := p.GetFee(t.Context(), feeBuilder); err != nil {
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
<<<<<<< HEAD
	result, err := p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
=======
	if _, err := p.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
<<<<<<< HEAD
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
=======
	if _, err := p.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
<<<<<<< HEAD
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
=======
	if _, err := p.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
<<<<<<< HEAD
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
=======
	if _, err := p.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
<<<<<<< HEAD
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	assert.NotNil(t, result)
=======
	if _, err := p.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}
}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	// CryptocurrencyTradeFee Basic
	feeBuilder = setFeeBuilder()
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	result, err = p.GetFee(context.Background(), feeBuilder)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := p.GetActiveOrders(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

<<<<<<< HEAD
	_, err = p.GetActiveOrders(context.Background(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
=======
	_, err := p.GetActiveOrders(t.Context(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
		t.Error("GetActiveOrders() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetActiveOrders() err", err)
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	}
	result, err := p.GetActiveOrders(generateContext(), &order.MultiOrderRequest{
		AssetType: asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = p.GetActiveOrders(generateContext(), &order.MultiOrderRequest{
		AssetType: asset.Futures,
		Side:      order.Buy,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := p.GetOrderHistory(context.Background(), &order.MultiOrderRequest{
		Type:      order.Limit,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = p.GetOrderHistory(context.Background(), &order.MultiOrderRequest{
		Type:      order.Liquidation,
		AssetType: asset.Spot,
		Side:      order.Buy,
	})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

<<<<<<< HEAD
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
=======
	_, err := p.GetOrderHistory(t.Context(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
		t.Errorf("Could not get order history: %s", err)
	case !sharedtestvalues.AreAPICredentialsSet(p) && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not mock get order history: %s", err)
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	}
	result, err := p.GetOrderHistory(generateContext(), &order.MultiOrderRequest{
		Type:      order.Limit,
		AssetType: asset.Spot,
		Side:      order.Buy,
	})
	assert.NoErrorf(t, err, "error: %v", err)
	assert.NotNil(t, result)

	result, err = p.GetOrderHistory(generateContext(), &order.MultiOrderRequest{
		Type:      order.Limit,
		AssetType: asset.Spot,
		Side:      order.Sell,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = p.GetOrderHistory(generateContext(), &order.MultiOrderRequest{
		Type:      order.Limit,
		AssetType: asset.Futures,
		Side:      order.Buy,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

<<<<<<< HEAD
=======
func TestGetOrderStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mock           bool
		orderID        string
		errExpected    bool
		errMsgExpected string
	}{
		{
			name:           "correct order ID",
			mock:           true,
			orderID:        "96238912841",
			errExpected:    false,
			errMsgExpected: "",
		},
		{
			name:           "wrong order ID",
			mock:           true,
			orderID:        "96238912842",
			errExpected:    true,
			errMsgExpected: "Order not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.mock != mockTests {
				t.Skip("mock mismatch, skipping")
			}

			_, err := p.GetAuthenticatedOrderStatus(t.Context(),
				tt.orderID)
			switch {
			case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
				t.Errorf("Could not get order status: %s", err)
			case !sharedtestvalues.AreAPICredentialsSet(p) && err == nil && !mockTests:
				t.Error("Expecting an error when no keys are set")
			case mockTests && err != nil:
				if !tt.errExpected {
					t.Errorf("Could not mock get order status: %s", err.Error())
				} else if !(strings.Contains(err.Error(), tt.errMsgExpected)) {
					t.Errorf("Could not mock get order status: %s", err.Error())
				}
			case mockTests:
				if tt.errExpected {
					t.Errorf("Mock get order status expect an error '%s', get no error", tt.errMsgExpected)
				}
			}
		})
	}
}

func TestGetOrderTrades(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mock           bool
		orderID        string
		errExpected    bool
		errMsgExpected string
	}{
		{
			name:           "correct order ID",
			mock:           true,
			orderID:        "96238912841",
			errExpected:    false,
			errMsgExpected: "",
		},
		{
			name:           "wrong order ID",
			mock:           true,
			orderID:        "96238912842",
			errExpected:    true,
			errMsgExpected: "Order not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.mock != mockTests {
				t.Skip("mock mismatch, skipping")
			}

			_, err := p.GetAuthenticatedOrderTrades(t.Context(), tt.orderID)
			switch {
			case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
				t.Errorf("Could not get order trades: %s", err)
			case !sharedtestvalues.AreAPICredentialsSet(p) && err == nil && !mockTests:
				t.Error("Expecting an error when no keys are set")
			case mockTests && err != nil:
				assert.ErrorContains(t, err, tt.errMsgExpected)
			}
		})
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := p.SubmitOrder(context.Background(), &order.Submit{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := &order.Submit{AssetType: asset.Futures, TimeInForce: order.GoodTillCrossing}
	_, err = p.SubmitOrder(context.Background(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	arg.Pair = futuresTradablePair
	_, err = p.SubmitOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce)
	arg.TimeInForce = order.GoodTillCancel
	arg.AssetType = asset.Options
	_, err = p.SubmitOrder(context.Background(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	arg.AssetType = asset.Spot
	arg.Type = order.Liquidation
	arg.Pair = spotTradablePair
	_, err = p.SubmitOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.SubmitOrder(generateContext(), &order.Submit{
		Exchange:  p.Name,
		Pair:      spotTradablePair,
		Side:      order.Buy,
		Type:      order.Market,
		Price:     10,
		Amount:    10000000,
		AssetType: asset.Spot,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

<<<<<<< HEAD
	result, err = p.SubmitOrder(generateContext(), &order.Submit{
		Exchange:     p.Name,
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

	result, err = p.SubmitOrder(generateContext(), &order.Submit{
		Exchange:     p.Name,
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

	result, err = p.SubmitOrder(generateContext(), &order.Submit{
		Exchange:     p.Name,
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
=======
	response, err := p.SubmitOrder(t.Context(), orderSubmission)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(p) && (err != nil || response.Status != order.Filled):
		t.Errorf("Order failed to be placed: %v", err)
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock SubmitOrder() err", err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	arg := &order.Cancel{
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
=======
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currency.NewPair(currency.LTC, currency.BTC),
		AssetType: asset.Spot,
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	}
	err := p.CancelOrder(context.Background(), arg)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

<<<<<<< HEAD
	arg.OrderID = "123"
	err = p.CancelOrder(context.Background(), arg)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
=======
	err := p.CancelOrder(t.Context(), orderCancellation)
	switch {
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Error("Mock CancelExchangeOrder() err", err)
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	}
	arg.AssetType = asset.Spot
	err = p.CancelOrder(generateContext(), arg)
	assert.NoError(t, err)

	arg.Type = order.StopLimit
	err = p.CancelOrder(generateContext(), arg)
	assert.NoError(t, err)

	err = p.CancelOrder(generateContext(), &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Futures,
	})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	err = p.CancelOrder(generateContext(), &order.Cancel{
		OrderID:   "1",
		AssetType: asset.Futures,
		Pair:      futuresTradablePair,
	})
	assert.NoError(t, err)
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	_, err := p.CancelAllOrders(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = p.CancelAllOrders(context.Background(), &order.Cancel{
		AssetType: asset.Options,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}
<<<<<<< HEAD
	arg := &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          spotTradablePair,
		AssetType:     asset.Spot,
=======

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	}
	p.Verbose = true
	arg.Type = order.Stop
	result, err := p.CancelAllOrders(generateContext(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)

<<<<<<< HEAD
	arg.Type = order.Limit
	result, err = p.CancelAllOrders(generateContext(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)

	arg.Pair = futuresTradablePair
	result, err = p.CancelAllOrders(generateContext(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.StopLimit
	result, err = p.CancelAllOrders(generateContext(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
=======
	resp, err := p.CancelAllOrders(t.Context(), orderCancellation)
	switch {
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Error("Mock CancelAllExchangeOrders() err", err)
	}
	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := p.ModifyOrder(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

<<<<<<< HEAD
	arg := &order.Modify{
		OrderID: "1337",
		Price:   1337,
=======
	_, err := p.ModifyOrder(t.Context(), &order.Modify{
		OrderID:   "1337",
		Price:     1337,
		AssetType: asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
	})
	switch {
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil && mockTests:
		t.Error("ModifyOrder() error", err)
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("ModifyOrder() error cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock ModifyOrder() err", err)
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	}
	_, err = p.ModifyOrder(context.Background(), arg)
	assert.ErrorIs(t, err, order.ErrPairIsEmpty)

	arg.Pair = spotTradablePair
	_, err = p.ModifyOrder(context.Background(), arg)
	assert.ErrorIs(t, err, order.ErrAssetNotSet)

	arg.AssetType = asset.Futures
	_, err = p.ModifyOrder(context.Background(), arg)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	_, err = p.ModifyOrder(context.Background(), arg)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.Type = order.Limit
	arg.TimeInForce = order.GoodTillTime
	_, err = p.ModifyOrder(context.Background(), arg)
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	arg.TimeInForce = order.GoodTillCancel
	result, err := p.ModifyOrder(context.Background(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	_, err := p.WithdrawCryptocurrencyFunds(context.Background(), nil)
	assert.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil)

	arg := &withdraw.Request{
		Crypto: withdraw.CryptoRequest{
			FeeAmount: 0,
		},
	}
	_, err = p.WithdrawCryptocurrencyFunds(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

<<<<<<< HEAD
	arg.Amount = 1000
	_, err = p.WithdrawCryptocurrencyFunds(context.Background(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.LTC
	_, err = p.WithdrawCryptocurrencyFunds(context.Background(), arg)
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	withdrawCryptoRequest := withdraw.Request{
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
		Amount:   1,
		Currency: currency.BTC,
=======
	_, err := p.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Error("should error due to invalid amount")
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	}
	result, err := p.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := p.UpdateAccountInfo(context.Background(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.UpdateAccountInfo(generateContext(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = p.UpdateAccountInfo(generateContext(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	var withdrawFiatRequest withdraw.Request
<<<<<<< HEAD
	_, err := p.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
=======
	_, err := p.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported, err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	_, err := p.WithdrawFiatFundsToInternationalBank(context.Background(), &withdraw.Request{})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
=======
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}

	var withdrawFiatRequest withdraw.Request
	_, err := p.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := p.GetDepositAddress(t.Context(), currency.USDT, "", "USDTETH")
	switch {
	case sharedtestvalues.AreAPICredentialsSet(p) && err != nil:
		t.Error("GetDepositAddress()", err)
	case !sharedtestvalues.AreAPICredentialsSet(p) && !mockTests && err == nil:
		t.Error("GetDepositAddress() cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock GetDepositAddress() err", err)
	}
}

func TestGenerateNewAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)

	_, err := p.GenerateNewAddress(t.Context(), currency.XRP.String())
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsAuth dials websocket, sends login request.
// Will receive a message only on failure
func TestWsAuth(t *testing.T) {
	t.Parallel()
	if !p.Websocket.IsEnabled() && !p.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(p) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	var dialer gws.Dialer
	err := p.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go p.wsReadData()
	creds, err := p.GetCredentials(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	err = p.wsSendAuthorisedCommand(creds.Secret, creds.Key, "subscribe")
	if err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case response := <-p.Websocket.DataHandler:
		t.Error(response)
	case <-timer.C:
	}
	timer.Stop()
}

func TestWsSubAck(t *testing.T) {
	pressXToJSON := []byte(`[1002, 1]`)
	err := p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	err := p.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[1002, null, [ 50, "382.98901522", "381.99755898", "379.41296309", "-0.04312950", "14969820.94951828", "38859.58435407", 0, "412.25844455", "364.56122072" ] ]`)
	err = p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsExchangeVolume(t *testing.T) {
	err := p.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[1003,null,["2018-11-07 16:26",5804,{"BTC":"3418.409","ETH":"2645.921","USDT":"10832502.689","USDC":"1578020.908"}]]`)
	err = p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrades(t *testing.T) {
	p.SetSaveTradeDataStatus(true)
	err := p.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[14, 8768, [["t", "42706057", 1, "0.05567134", "0.00181421", 1522877119]]]`)
	err = p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsPriceAggregateOrderbook(t *testing.T) {
	err := p.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}
	pressXToJSON := []byte(`[50,141160924,[["i",{"currencyPair":"BTC_LTC","orderBook":[{"0.002784":"17.55","0.002786":"1.47","0.002792":"13.25","0.0028":"0.21","0.002804":"0.02","0.00281":"1.5","0.002811":"258.82","0.002812":"3.81","0.002817":"0.06","0.002824":"3","0.002825":"0.02","0.002836":"18.01","0.002837":"0.03","0.00284":"0.03","0.002842":"12.7","0.00285":"0.02","0.002852":"0.02","0.002855":"1.3","0.002857":"15.64","0.002864":"0.01"},{"0.002782":"45.93","0.002781":"1.46","0.002774":"13.34","0.002773":"0.04","0.002771":"0.05","0.002765":"6.21","0.002764":"3","0.00276":"10.77","0.002758":"3.11","0.002754":"0.02","0.002751":"288.94","0.00275":"24.06","0.002745":"187.27","0.002743":"0.04","0.002742":"0.96","0.002731":"0.06","0.00273":"12.13","0.002727":"0.02","0.002725":"0.03","0.002719":"1.09"}]}, "1692080077892"]]]`)
	err = p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`[50,141160925,[["o",1,"0.002742","0", "1692080078806"],["o",1,"0.002718","0.02", "1692080078806"]]]`)
	err = p.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
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
	result, err := p.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.FiveMin, start.UTC(), end.UTC())
	require.NoError(t, err)
	require.NotNil(t, result)

<<<<<<< HEAD
	result, err = p.GetHistoricCandles(context.Background(), futuresTradablePair, asset.Futures, kline.FiveMin, start.UTC(), end.UTC())
	require.NoError(t, err)
	assert.NotNil(t, result)
=======
	pair, err := currency.NewPairFromString("BTC_LTC")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Unix(1588741402, 0)
	_, err = p.GetHistoricCandles(t.Context(), pair, asset.Spot, kline.FiveMin, start, time.Unix(1588745003, 0))
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1744103854944)
	end := time.UnixMilli(1744190254944)
	if !mockTests {
		start = time.Now().Add(-time.Hour * 24)
		end = time.Now()
	}
	result, err := p.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.OneHour, start, end)
	require.NoError(t, err)
	require.NotNil(t, result)

<<<<<<< HEAD
	result, err = p.GetHistoricCandlesExtended(context.Background(), futuresTradablePair, asset.Futures, kline.FiveMin, start, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
=======
	_, err = p.GetHistoricCandlesExtended(t.Context(), pair, asset.Spot, kline.FiveMin, time.Unix(1588741402, 0), time.Unix(1588745003, 0))
	if !errors.Is(err, nil) {
		t.Fatal(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	result, err := p.GetRecentTrades(context.Background(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = p.GetRecentTrades(context.Background(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
=======
	currencyPair, err := currency.NewPairFromString("BTC_XMR")
	if err != nil {
		t.Fatal(err)
	}
	if mockTests {
		t.Skip("relies on time.Now()")
	}
	_, err = p.GetRecentTrades(t.Context(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
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
<<<<<<< HEAD
	_, err := p.GetHistoricTrades(context.Background(),
		spotTradablePair, asset.Spot, tStart, tEnd)
	require.NoError(t, err)

	_, err = p.GetHistoricTrades(context.Background(),
		futuresTradablePair, asset.Futures, tStart, tEnd)
	require.NoError(t, err)
=======
	_, err = p.GetHistoricTrades(t.Context(),
		currencyPair, asset.Spot, tStart, tEnd)
	if err != nil {
		t.Error(err)
	}
}

func TestProcessAccountMarginPosition(t *testing.T) {
	err := p.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}

	margin := []byte(`[1000,"",[["m", 23432933, 28, "-0.06000000"]]]`)
	err = p.wsHandleData(margin)
	if !errors.Is(err, errNotEnoughData) {
		t.Fatalf("expected: %v but received: %v", errNotEnoughData, err)
	}

	margin = []byte(`[1000,"",[["m", "23432933", 28, "-0.06000000", null]]]`)
	err = p.wsHandleData(margin)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	margin = []byte(`[1000,"",[["m", 23432933, "28", "-0.06000000", null]]]`)
	err = p.wsHandleData(margin)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	margin = []byte(`[1000,"",[["m", 23432933, 28, -0.06000000, null]]]`)
	err = p.wsHandleData(margin)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	margin = []byte(`[1000,"",[["m", 23432933, 28, "-0.06000000", null]]]`)
	err = p.wsHandleData(margin)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountPendingOrder(t *testing.T) {
	err := p.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}

	pending := []byte(`[1000,"",[["p",431682155857,127,"1000.00000000","1.00000000","0"]]]`)
	err = p.wsHandleData(pending)
	if !errors.Is(err, errNotEnoughData) {
		t.Fatalf("expected: %v but received: %v", errNotEnoughData, err)
	}

	pending = []byte(`[1000,"",[["p","431682155857",127,"1000.00000000","1.00000000","0",null]]]`)
	err = p.wsHandleData(pending)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	pending = []byte(`[1000,"",[["p",431682155857,"127","1000.00000000","1.00000000","0",null]]]`)
	err = p.wsHandleData(pending)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	pending = []byte(`[1000,"",[["p",431682155857,127,1000.00000000,"1.00000000","0",null]]]`)
	err = p.wsHandleData(pending)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	pending = []byte(`[1000,"",[["p",431682155857,127,"1000.00000000",1.00000000,"0",null]]]`)
	err = p.wsHandleData(pending)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	pending = []byte(`[1000,"",[["p",431682155857,127,"1000.00000000","1.00000000",0,null]]]`)
	err = p.wsHandleData(pending)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	pending = []byte(`[1000,"",[["p",431682155857,127,"1000.00000000","1.00000000","0",null]]]`)
	err = p.wsHandleData(pending)
	if err != nil {
		t.Fatal(err)
	}

	// Unmatched pair in system
	pending = []byte(`[1000,"",[["p",431682155857,666,"1000.00000000","1.00000000","0",null]]]`)
	err = p.wsHandleData(pending)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountOrderUpdate(t *testing.T) {
	orderUpdate := []byte(`[1000,"",[["o",431682155857,"0.00000000","f"]]]`)
	err := p.wsHandleData(orderUpdate)
	if !errors.Is(err, errNotEnoughData) {
		t.Fatalf("expected: %v but received: %v", errNotEnoughData, err)
	}

	orderUpdate = []byte(`[1000,"",[["o","431682155857","0.00000000","f",null]]]`)
	err = p.wsHandleData(orderUpdate)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	orderUpdate = []byte(`[1000,"",[["o",431682155857,0.00000000,"f",null]]]`)
	err = p.wsHandleData(orderUpdate)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.00000000",123,null]]]`)
	err = p.wsHandleData(orderUpdate)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.00000000","c",null]]]`)
	err = p.wsHandleData(orderUpdate)
	if !errors.Is(err, errNotEnoughData) {
		t.Fatalf("expected: %v but received: %v", errNotEnoughData, err)
	}

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.50000000","c",null,"0.50000000"]]]`)
	err = p.wsHandleData(orderUpdate)
	if err != nil {
		t.Fatal(err)
	}

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.00000000","c",null,"1.00000000"]]]`)
	err = p.wsHandleData(orderUpdate)
	if err != nil {
		t.Fatal(err)
	}

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.50000000","f",null]]]`)
	err = p.wsHandleData(orderUpdate)
	if err != nil {
		t.Fatal(err)
	}

	orderUpdate = []byte(`[1000,"",[["o",431682155857,"0.00000000","s",null]]]`)
	err = p.wsHandleData(orderUpdate)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountOrderLimit(t *testing.T) {
	err := p.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}

	accountTrade := []byte(`[1000,"",[["n",127,431682155857,"0","1000.00000000","1.00000000","2021-04-13 07:19:56","1.00000000"]]]`)
	err = p.wsHandleData(accountTrade)
	if !errors.Is(err, errNotEnoughData) {
		t.Fatalf("expected: %v but received: %v", errNotEnoughData, err)
	}

	accountTrade = []byte(`[1000,"",[["n","127",431682155857,"0","1000.00000000","1.00000000","2021-04-13 07:19:56","1.00000000",null]]]`)
	err = p.wsHandleData(accountTrade)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrade = []byte(`[1000,"",[["n",127,"431682155857","0","1000.00000000","1.00000000","2021-04-13 07:19:56","1.00000000",null]]]`)
	err = p.wsHandleData(accountTrade)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,0,"1000.00000000","1.00000000","2021-04-13 07:19:56","1.00000000",null]]]`)
	err = p.wsHandleData(accountTrade)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,"0",1000.00000000,"1.00000000","2021-04-13 07:19:56","1.00000000",null]]]`)
	err = p.wsHandleData(accountTrade)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,"0","1000.00000000",1.00000000,"2021-04-13 07:19:56","1.00000000",null]]]`)
	err = p.wsHandleData(accountTrade)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,"0","1000.00000000","1.00000000",1234,"1.00000000",null]]]`)
	err = p.wsHandleData(accountTrade)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,"0","1000.00000000","1.00000000","2021-04-13 07:19:56",1.00000000,null]]]`)
	err = p.wsHandleData(accountTrade)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrade = []byte(`[1000,"",[["n",127,431682155857,"0","1000.00000000","1.00000000","2021-04-13 07:19:56","1.00000000",null]]]`)
	err = p.wsHandleData(accountTrade)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountBalanceUpdate(t *testing.T) {
	err := p.loadCurrencyDetails(t.Context())
	if err != nil {
		t.Error(err)
	}

	balance := []byte(`[1000,"",[["b",243,"e"]]]`)
	err = p.wsHandleData(balance)
	if !errors.Is(err, errNotEnoughData) {
		t.Fatalf("expected: %v but received: %v", errNotEnoughData, err)
	}

	balance = []byte(`[1000,"",[["b","243","e","-1.00000000"]]]`)
	err = p.wsHandleData(balance)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	balance = []byte(`[1000,"",[["b",243,1234,"-1.00000000"]]]`)
	err = p.wsHandleData(balance)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	balance = []byte(`[1000,"",[["b",243,"e",-1.00000000]]]`)
	err = p.wsHandleData(balance)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	balance = []byte(`[1000,"",[["b",243,"e","-1.00000000"]]]`)
	err = p.wsHandleData(balance)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountTrades(t *testing.T) {
	accountTrades := []byte(`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345"]]]`)
	err := p.wsHandleData(accountTrades)
	if !errors.Is(err, errNotEnoughData) {
		t.Fatalf("expected: %v but received: %v", errNotEnoughData, err)
	}

	accountTrades = []byte(`[1000,"",[["t", "12345", "0.03000000", "0.50000000", "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = p.wsHandleData(accountTrades)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrades = []byte(`[1000,"",[["t", 12345, 0.03000000, "0.50000000", "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = p.wsHandleData(accountTrades)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrades = []byte(`[1000,"",[["t", 12345, "0.03000000", 0.50000000, "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = p.wsHandleData(accountTrades)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrades = []byte(`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, 0.00000375, "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = p.wsHandleData(accountTrades)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrades = []byte(`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, 0.0000037, "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = p.wsHandleData(accountTrades)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrades = []byte(`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, "0.00000375", 12345, "12345", 0.015]]]`)
	err = p.wsHandleData(accountTrades)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	accountTrades = []byte(`[1000,"",[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09", "12345", "0.015"]]]`)
	err = p.wsHandleData(accountTrades)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessAccountKilledOrder(t *testing.T) {
	kill := []byte(`[1000,"",[["k", 1337]]]`)
	err := p.wsHandleData(kill)
	if !errors.Is(err, errNotEnoughData) {
		t.Fatalf("expected: %v but received: %v", errNotEnoughData, err)
	}

	kill = []byte(`[1000,"",[["k", "1337", null]]]`)
	err = p.wsHandleData(kill)
	if !errors.Is(err, errTypeAssertionFailure) {
		t.Fatalf("expected: %v but received: %v", errTypeAssertionFailure, err)
	}

	kill = []byte(`[1000,"",[["k", 1337, null]]]`)
	err = p.wsHandleData(kill)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetCompleteBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetCompleteBalances(t.Context())
	if err != nil {
		t.Fatal(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	_, err := p.UpdateTicker(context.Background(), spotTradablePair, asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := p.UpdateTicker(context.Background(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = p.UpdateTicker(context.Background(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
=======
	cp, err := currency.NewPairFromString("BTC_LTC")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.UpdateTicker(t.Context(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	err := p.UpdateTickers(context.Background(), asset.Options)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	err = p.UpdateTickers(context.Background(), asset.Spot)
	assert.NoError(t, err)

	err = p.UpdateTickers(context.Background(), asset.Futures)
	assert.NoError(t, err)
=======
	err := p.UpdateTickers(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	_, err := p.GetAvailableTransferChains(context.Background(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := p.GetAvailableTransferChains(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
=======
	_, err := p.GetAvailableTransferChains(t.Context(), currency.USDT)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWalletActivity(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)

	_, err := p.WalletActivity(t.Context(), time.Now().Add(-time.Minute), time.Now(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelMultipleOrdersByIDs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.CancelMultipleOrdersByIDs(t.Context(), []string{"1234"}, []string{"5678"})
	if err != nil {
		t.Error(err)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	if mockTests {
		t.SkipNow()
=======
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	_, err := p.GetAccountFundingHistory(t.Context())
	if err != nil {
		t.Error(err)
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetAccountFundingHistory(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	if mockTests {
		t.SkipNow()
=======
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)

	_, err := p.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error(err)
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	result, err := p.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := p.CancelBatchOrders(context.Background(), []order.Cancel{})
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	_, err = p.CancelBatchOrders(context.Background(), []order.Cancel{{AssetType: asset.Options}})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = p.CancelBatchOrders(context.Background(), []order.Cancel{{AssetType: asset.Futures}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = p.CancelBatchOrders(context.Background(), []order.Cancel{{AssetType: asset.Futures, OrderID: "1233", Pair: futuresTradablePair}, {AssetType: asset.Spot, Pair: futuresTradablePair}})
	require.ErrorIs(t, err, errOrderAssetTypeMismatch)

	_, err = p.CancelBatchOrders(context.Background(), []order.Cancel{{AssetType: asset.Futures, OrderID: "1233"}, {AssetType: asset.Futures}})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = p.CancelBatchOrders(context.Background(), []order.Cancel{{Pair: futuresTradablePair, AssetType: asset.Futures, OrderID: "1233"}, {OrderID: "1233", AssetType: asset.Futures, Pair: spotTradablePair}})
	require.ErrorIs(t, err, errPairStringMismatch)

	_, err = p.CancelBatchOrders(context.Background(), []order.Cancel{{AssetType: asset.Spot, OrderID: "1233", Type: order.Liquidation}, {AssetType: asset.Spot, OrderID: "123444", Type: order.StopLimit}})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
<<<<<<< HEAD
	_, err = p.CancelBatchOrders(generateContext(), []order.Cancel{{
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

	result, err := p.CancelBatchOrders(generateContext(), []order.Cancel{
=======
	_, err := p.CancelBatchOrders(t.Context(), []order.Cancel{
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
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
<<<<<<< HEAD
	require.NoError(t, err)
	assert.NotNil(t, result)
=======
	if err != nil {
		t.Error(err)
	}
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	st, err := p.GetTimestamp(t.Context())
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if st.IsZero() {
		t.Error("expected a time")
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	st, err := p.GetServerTime(context.Background(), asset.Spot)
	require.NoError(t, err)
	require.NotZero(t, st)
=======
	st, err := p.GetServerTime(t.Context(), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651

	st, err = p.GetServerTime(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotZero(t, st)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := p.GetFuturesContractDetails(context.Background(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = p.GetFuturesContractDetails(context.Background(), asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := p.GetFuturesContractDetails(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := p.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = p.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{Pair: currency.NewPair(currency.BTC, currency.LTC)})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	_, err = p.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.Spot,
		Pair:                 spotTradablePair,
		IncludePredictedRate: false,
	})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	result, err := p.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.Futures,
		Pair:                 futuresTradablePair,
		IncludePredictedRate: false,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := p.IsPerpetualFutureCurrency(asset.Spot, spotTradablePair)
	require.NoError(t, err)
	require.False(t, is)

	is, err = p.IsPerpetualFutureCurrency(asset.Futures, futuresTradablePair)
	require.NoError(t, err)
	assert.True(t, is)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	_, err := p.FetchTradablePairs(context.Background(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := p.FetchTradablePairs(context.Background(), asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.FetchTradablePairs(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSymbolInformation(t *testing.T) {
	t.Parallel()
	result, err := p.GetSymbolInformation(context.Background(), spotTradablePair)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetSymbolInformation(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrenciesInformation(t *testing.T) {
	t.Parallel()
	result, err := p.GetCurrenciesInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV2CurrencyInformation(t *testing.T) {
	t.Parallel()
	result, err := p.GetV2CurrencyInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTimestamp(t *testing.T) {
	t.Parallel()
	result, err := p.GetSystemTimestamp(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketPrices(t *testing.T) {
	t.Parallel()
	result, err := p.GetMarketPrices(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketPrice(t *testing.T) {
	t.Parallel()
	_, err := p.GetMarketPrice(context.Background(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := p.GetMarketPrice(context.Background(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrices(t *testing.T) {
	t.Parallel()
	result, err := p.GetMarkPrices(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := p.GetMarkPrice(context.Background(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := p.GetMarkPrice(context.Background(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarkPriceComponents(t *testing.T) {
	t.Parallel()
	_, err := p.MarkPriceComponents(context.Background(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := p.MarkPriceComponents(context.Background(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := p.GetOrderbook(context.Background(), currency.EMPTYPAIR, 0, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := p.GetOrderbook(context.Background(), spotTradablePair, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := p.UpdateOrderbook(context.Background(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = p.UpdateOrderbook(context.Background(), spotTradablePair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := p.UpdateOrderbook(context.Background(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.UpdateOrderbook(context.Background(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := p.GetCandlesticks(context.Background(), currency.EMPTYPAIR, kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = p.GetCandlesticks(context.Background(), spotTradablePair, kline.HundredMilliseconds, time.Now().Add(-time.Hour*48), time.Time{}, 0)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := p.GetCandlesticks(context.Background(), spotTradablePair, kline.FiveMin, time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := p.GetTrades(context.Background(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := p.GetTrades(context.Background(), spotTradablePair, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	result, err := p.GetTickers(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := p.GetTicker(context.Background(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err := p.GetTicker(context.Background(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralInfos(t *testing.T) {
	t.Parallel()
	result, err := p.GetCollateralInfos(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralInfo(t *testing.T) {
	t.Parallel()
	result, err := p.GetCollateralInfo(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowRateInfo(t *testing.T) {
	t.Parallel()
	result, err := p.GetBorrowRateInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAccountInformation(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAllBalances(generateContext(), "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetAllBalances(generateContext(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllBalance(t *testing.T) {
	t.Parallel()
	_, err := p.GetAllBalance(context.Background(), "", "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAllBalance(generateContext(), "329455537441832960", "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = p.GetAllBalance(generateContext(), "329455537441832960", "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllAccountActivities(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetAllAccountActivities(generateContext(), time.Time{}, time.Time{}, 0, 0, 0, "", currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestAccountsTransfer(t *testing.T) {
	t.Parallel()
	_, err := p.AccountsTransfer(context.Background(), &AccountTransferParams{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.AccountsTransfer(context.Background(), &AccountTransferParams{Amount: 1232.221})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = p.AccountsTransfer(context.Background(), &AccountTransferParams{
		Ccy: currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = p.AccountsTransfer(context.Background(), &AccountTransferParams{
		Amount:      1,
		Ccy:         currency.BTC,
		FromAccount: "219961623421431808",
	})
	require.ErrorIs(t, err, errAddressRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	result, err := p.AccountsTransfer(generateContext(), &AccountTransferParams{
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetAccountTransferRecords(generateContext(), time.Time{}, time.Time{}, "", currency.BTC, 0, 0)
	require.NoError(t, err)
}

func TestGetAccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := p.GetAccountTransferRecord(generateContext(), "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err = p.GetAccountTransferRecord(generateContext(), "329455537441832960")
	require.NoError(t, err)
}

func TestGetFeeInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetFeeInfo(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetInterestHistory(generateContext(), time.Time{}, time.Time{}, "", 0, 0)
	require.NoError(t, err)
}

func TestGetSubAccountInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetSubAccountInformation(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetSubAccountBalances(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBalance(t *testing.T) {
	t.Parallel()
	_, err := p.GetSubAccountBalance(context.Background(), "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err = p.GetSubAccountBalance(generateContext(), "2d45301d-5f08-4a2b-a763-f9199778d854")
	require.NoError(t, err)
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	_, err := p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{Amount: 12.34})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency: currency.BTC,
	})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency: currency.BTC,
		Amount:   1,
	})
	require.ErrorIs(t, err, errAccountIDRequired)
	_, err = p.SubAccountTransfer(context.Background(), &SubAccountTransferParam{
		Currency:      currency.BTC,
		Amount:        1,
		FromAccountID: "1234568",
		ToAccountID:   "1234567",
	})
	require.ErrorIs(t, err, errAccountTypeRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	result, err := p.SubAccountTransfer(generateContext(), &SubAccountTransferParam{
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetSubAccountTransferRecords(generateContext(), currency.BTC, time.Time{}, time.Time{}, "", "", "", "", "", 0, 0)
	require.NoError(t, err)
}

func TestGetSubAccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := p.GetSubAccountTransferRecord(context.Background(), "")
	require.ErrorIs(t, err, errAccountIDRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err = p.GetSubAccountTransferRecord(generateContext(), "329455537441832960")
	require.NoError(t, err)
}

func TestGetDepositAddresses(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetDepositAddresses(generateContext(), currency.LTC)
	require.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := p.GetOrderInfo(generateContext(), "", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetOrderInfo(generateContext(), "1234", spotTradablePair, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = p.GetOrderInfo(generateContext(), "12345", futuresTradablePair, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetDepositAddress(generateContext(), currency.BTC, "", "TON")
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
		start = time.Now().Add(-time.Hour * 2)
		end = time.Now()
	}
	result, err := p.WalletActivity(generateContext(), start, end, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewCurrencyDepoditAddress(t *testing.T) {
	t.Parallel()
	_, err := p.NewCurrencyDepositAddress(context.Background(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	result, err := p.NewCurrencyDepositAddress(generateContext(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	_, err := p.WithdrawCurrency(context.Background(), &WithdrawCurrencyParam{})
	require.ErrorIs(t, err, errNilArgument)

	_, err = p.WithdrawCurrency(context.Background(), &WithdrawCurrencyParam{
		Currency: currency.BTC.String() + "TRON", // Sends BTC through the TRON chain
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = p.WithdrawCurrency(context.Background(), &WithdrawCurrencyParam{Currency: currency.BTC.String(), Amount: 1})
	require.ErrorIs(t, err, errAddressRequired)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	result, err := p.WithdrawCurrency(generateContext(), &WithdrawCurrencyParam{Currency: currency.BTC.String(), Amount: 1, Address: "0xbb8d0d7c346daecc2380dabaa91f3ccf8ae232fb4"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCurrencyV2(t *testing.T) {
	t.Parallel()
	_, err := p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{Coin: currency.BTC})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{Coin: currency.BTC, Amount: 1})
	require.ErrorIs(t, err, errInvalidWithdrawalChain)
	_, err = p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{Coin: currency.BTC, Amount: 1, Network: "BTC"})
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.WithdrawCurrencyV2(context.Background(), &WithdrawCurrencyV2Param{Network: "BTC", Coin: currency.BTC, Amount: 1, Address: "0xbb8d0d7c346daecc2380dabaa91f3ccf8ae232fb4"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountMarginInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAccountMarginInformation(generateContext(), "SPOT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetBorrowStatus(generateContext(), currency.USDT)
	require.NoError(t, err)
}

func TestMaximumBuySellAmount(t *testing.T) {
	t.Parallel()
	_, err := p.MaximumBuySellAmount(context.Background(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.MaximumBuySellAmount(generateContext(), spotTradablePair)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := p.PlaceOrder(context.Background(), &PlaceOrderParams{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.PlaceOrder(context.Background(), &PlaceOrderParams{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = p.PlaceOrder(context.Background(), &PlaceOrderParams{Symbol: spotTradablePair})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.PlaceOrder(context.Background(), &PlaceOrderParams{
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
	_, err := p.PlaceBatchOrders(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{
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

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.PlaceBatchOrders(context.Background(), []PlaceOrderParams{
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
	_, err := p.CancelReplaceOrder(context.Background(), &CancelReplaceOrderParam{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.CancelReplaceOrder(context.Background(), &CancelReplaceOrderParam{
		TimeInForce: "GTC",
	})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelReplaceOrder(context.Background(), &CancelReplaceOrderParam{
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetOpenOrders(generateContext(), spotTradablePair, "", "NEXT", "", 10)
	require.NoError(t, err)
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := p.GetOrderDetail(context.Background(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetOrderDetail(generateContext(), "12345536545645", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderByID(t *testing.T) {
	t.Parallel()
	_, err := p.CancelOrderByID(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelOrderByID(context.Background(), "12345536545645")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleOrdersByIDs(t *testing.T) {
	t.Parallel()
	_, err := p.CancelMultipleOrdersByIDs(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.CancelMultipleOrdersByIDs(context.Background(), &OrderCancellationParams{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelMultipleOrdersByIDs(context.Background(), &OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelAllTradeOrders(context.Background(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestKillSwitch(t *testing.T) {
	t.Parallel()
	_, err := p.KillSwitch(context.Background(), "")
	require.ErrorIs(t, err, errInvalidTimeout)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.KillSwitch(generateContext(), "30")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetKillSwitchStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetKillSwitchStatus(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSmartOrder(t *testing.T) {
	t.Parallel()
	_, err := p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{})
	require.ErrorIs(t, err, errNilArgument)

	_, err = p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{Side: "BUY"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = p.CreateSmartOrder(context.Background(), &SmartOrderRequestParam{Symbol: spotTradablePair})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CreateSmartOrder(generateContext(), &SmartOrderRequestParam{
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
	_, err := p.CancelReplaceSmartOrder(context.Background(), &CancelReplaceSmartOrderParam{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.CancelReplaceSmartOrder(context.Background(), &CancelReplaceSmartOrderParam{Price: 18000})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelReplaceSmartOrder(context.Background(), &CancelReplaceSmartOrderParam{
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
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetSmartOpenOrders(generateContext(), 10)
	require.NoError(t, err)
}

func TestGetSmartOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := p.GetSmartOrderDetail(generateContext(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetSmartOrderDetail(generateContext(), "123313413", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSmartOrderByID(t *testing.T) {
	t.Parallel()
	_, err := p.CancelSmartOrderByID(context.Background(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelSmartOrderByID(context.Background(), "123313413", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleSmartOrders(t *testing.T) {
	t.Parallel()
	_, err := p.CancelMultipleSmartOrders(context.Background(), nil)
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.CancelMultipleSmartOrders(context.Background(), &OrderCancellationParams{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelMultipleSmartOrders(context.Background(), &OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllSmartOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.CancelAllSmartOrders(context.Background(), []string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"}, []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrdersHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetOrdersHistory(generateContext(), spotTradablePair, "SPOT", "", "", "", "", 0, 10, time.Time{}, time.Time{}, false)
	require.NoError(t, err)
}

func TestGetSmartOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetSmartOrderHistory(generateContext(), spotTradablePair, "SPOT", "", "", "", "", 0, 10, time.Time{}, time.Time{}, false)
	require.NoError(t, err)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetTradeHistory(generateContext(), currency.Pairs{spotTradablePair}, "", 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
}

func TestGetTradeOrderID(t *testing.T) {
	t.Parallel()
	_, err := p.GetTradesByOrderID(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err = p.GetTradesByOrderID(generateContext(), "13123242323")
	require.NoError(t, err)
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	result, err := p.GenerateDefaultSubscriptions()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHandlePayloads(t *testing.T) {
	t.Parallel()
	subscriptions, err := p.GenerateDefaultSubscriptions()
	require.NoError(t, err)
	require.NotEmpty(t, subscriptions)

	result, err := p.handleSubscriptions("subscribe", subscriptions)
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
		err := p.wsHandleData([]byte(value))
		assert.NoErrorf(t, err, "%s error %s: %v", p.Name, key, err)
=======
	_, err := p.FetchTradablePairs(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
	}
}

func TestWsCreateOrder(t *testing.T) {
	t.Parallel()
	_, err := p.WsCreateOrder(&PlaceOrderParams{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = p.WsCreateOrder(&PlaceOrderParams{Amount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = p.WsCreateOrder(&PlaceOrderParams{
		Symbol: spotTradablePair,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.WsCreateOrder(&PlaceOrderParams{
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
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	testexch.SetupWs(t, p)
	result, err := p.WsCancelMultipleOrdersByIDs(&OrderCancellationParams{OrderIDs: []string{"1234"}, ClientOrderIDs: []string{"5678"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.SkipNow()
	}
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	testexch.SetupWs(t, p)
	result, err := p.WsCancelAllTradeOrders([]string{"BTC_USDT", "ETH_USDT"}, []string{"SPOT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := p.UpdateOrderExecutionLimits(context.Background(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = p.UpdateOrderExecutionLimits(context.Background(), asset.Futures)
	require.NoError(t, err)

	p.Verbose = true
	instrument, err := p.GetV3FuturesProductInfo(context.Background(), futuresTradablePair.String())
	require.NoError(t, err)
	require.NotNil(t, instrument)

	limits, err := p.GetOrderExecutionLimits(asset.Futures, futuresTradablePair)
	require.NoError(t, err)
	require.NotNil(t, limits)
	require.Equal(t, limits.PriceStepIncrementSize, instrument.TickSize.Float64())
	require.Equal(t, limits.MinimumBaseAmount, instrument.MinQuantity.Float64())
	assert.Equal(t, limits.MinimumQuoteAmount, instrument.MinSize.Float64())

	// sample test for spot instrument order execution limit

	err = p.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	require.NoError(t, err)

	spotInstruments, err := p.GetSymbolInformation(context.Background(), spotTradablePair)
	require.NoError(t, err)
	require.NotNil(t, instrument)

	limits, err = p.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
	require.NoError(t, err)
	require.Len(t, spotInstruments, 1)
	require.Equal(t, limits.PriceStepIncrementSize, spotInstruments[0].SymbolTradeLimit.PriceScale)
	require.Equal(t, limits.MinimumBaseAmount, spotInstruments[0].SymbolTradeLimit.MinQuantity.Float64())
	assert.Equal(t, limits.MinimumQuoteAmount, spotInstruments[0].SymbolTradeLimit.MinAmount.Float64())
}

// ---- Futures endpoints ---

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, p)
	for _, a := range p.GetAssetTypes(false) {
		pairs, err := p.CurrencyPairs.GetPairs(a, false)
<<<<<<< HEAD
		assert.NoError(t, err, "cannot get pairs for %s", a)
		assert.NotEmpty(t, pairs, "no pairs for %s", a)

		resp, err := p.GetCurrencyTradeURL(context.Background(), a, pairs[0])
		assert.NoError(t, err)
=======
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := p.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
>>>>>>> bea16af380a26e7706d97dde4016c72c84d71651
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
		err = p.wsFuturesHandleData([]byte(data))
		assert.NoErrorf(t, err, "%s: unexpected error %v", title, err)
	}
}

func TestGetCurrencyInformation(t *testing.T) {
	t.Parallel()
	_, err := p.GetCurrencyInformation(context.Background(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := p.GetCurrencyInformation(context.Background(), currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	_, err := p.GetAccountBalance(generateContext())
	require.NoError(t, err)
}

func TestGetAccountBills(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetAccountBills(generateContext(), time.Time{}, time.Time{}, 0, 0, "NEXT", "PNL")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceV3FuturesOrder(t *testing.T) {
	t.Parallel()
	arg := &FuturesParams{}
	_, err := p.PlaceV3FuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.ReduceOnly = true
	_, err = p.PlaceV3FuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTC_USDT_PERP"
	_, err = p.PlaceV3FuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	_, err = p.PlaceV3FuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "LONG"
	_, err = p.PlaceV3FuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit_maker"
	_, err = p.PlaceV3FuturesOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	_, err = p.PlaceV3FuturesOrder(context.Background(), &FuturesParams{
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
	_, err := p.PlaceV3FuturesMultipleOrders(context.Background(), []FuturesParams{arg})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.ReduceOnly = true
	_, err = p.PlaceV3FuturesMultipleOrders(context.Background(), []FuturesParams{arg})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	arg.Symbol = "BTC_USDT_PERP"
	_, err = p.PlaceV3FuturesMultipleOrders(context.Background(), []FuturesParams{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	_, err = p.PlaceV3FuturesMultipleOrders(context.Background(), []FuturesParams{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "LONG"
	_, err = p.PlaceV3FuturesMultipleOrders(context.Background(), []FuturesParams{arg})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit_maker"
	_, err = p.PlaceV3FuturesMultipleOrders(context.Background(), []FuturesParams{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	result, err := p.PlaceV3FuturesMultipleOrders(context.Background(), []FuturesParams{
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
	_, err := p.CancelV3FuturesOrder(context.Background(), &CancelOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	p.Verbose = true
	_, err = p.CancelV3FuturesOrder(context.Background(), &CancelOrderParams{OrderID: "1234"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = p.CancelV3FuturesOrder(context.Background(), &CancelOrderParams{Symbol: futuresTradablePair.String()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	_, err = p.CancelV3FuturesOrder(generateContext(), &CancelOrderParams{Symbol: futuresTradablePair.String(), OrderID: "12345"})
	require.NoError(t, err)
}

func TestCancelMultipleV3FuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := p.CancelMultipleV3FuturesOrders(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = p.CancelMultipleV3FuturesOrders(context.Background(), &CancelOrdersParams{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}
	result, err := p.CancelMultipleV3FuturesOrders(generateContext(), &CancelOrdersParams{Symbol: futuresTradablePair.String()})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllV3FuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := p.CancelAllV3FuturesOrders(context.Background(), "", "BUY")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}
	result, err := p.CancelAllV3FuturesOrders(generateContext(), futuresTradablePair.String(), "BUY")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCloseAtMarketPrice(t *testing.T) {
	t.Parallel()
	_, err := p.CloseAtMarketPrice(context.Background(), "", "", "", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = p.CloseAtMarketPrice(context.Background(), futuresTradablePair.String(), "", "", "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = p.CloseAtMarketPrice(context.Background(), futuresTradablePair.String(), "CROSS", "", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}
	_, err = p.CloseAtMarketPrice(generateContext(), futuresTradablePair.String(), "CROSS", "", "123123")
	require.NoError(t, err)
}

func TestCloseAllAtMarketPrice(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, p, canManipulateRealOrders)
	}
	_, err := p.CloseAllAtMarketPrice(generateContext())
	require.NoError(t, err)
}

func TestGetCurrentOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetCurrentFuturesOrders(generateContext(), futuresTradablePair.String(), "SELL", "", "", "NEXT", 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderExecutionDetails(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1743615790295), time.UnixMilli(1743702190295)
	if !mockTests {
		startTime, endTime = time.Now().Add(-time.Hour*24), time.Now()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetOrderExecutionDetails(generateContext(), "", "", "", "NEXT", startTime, endTime, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetV3FuturesOrderHistory(generateContext(), "", "LIMIT", "", "PARTIALLY_CANCELED", "", "", "PREV", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesCurrentPosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetV3FuturesCurrentPosition(generateContext(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesPositionHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetV3FuturesPositionHistory(generateContext(), "", "ISOLATED", "LONG", "NEXT", time.Time{}, time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAdjustMarginForIsolatedMarginTradingPositions(t *testing.T) {
	t.Parallel()
	_, err := p.AdjustMarginForIsolatedMarginTradingPositions(context.Background(), "", "", "ADD", 123)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = p.AdjustMarginForIsolatedMarginTradingPositions(context.Background(), "DOT_USDT_PERP", "", "ADD", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = p.AdjustMarginForIsolatedMarginTradingPositions(context.Background(), "DOT_USDT_PERP", "", "", 123)
	require.ErrorIs(t, err, errMarginAdjustTypeMissing)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	_, err = p.AdjustMarginForIsolatedMarginTradingPositions(generateContext(), "BTC_USDT_PERP", "", "ADD", 123)
	require.NoError(t, err)
}

func TestGetV3FuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := p.GetV3FuturesLeverage(context.Background(), "", "ISOLATED")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetV3FuturesLeverage(generateContext(), "BTC_USDT_PERP", "ISOLATED")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetV3FuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := p.SetV3FuturesLeverage(context.Background(), "", "CROSS", "LONG", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = p.SetV3FuturesLeverage(context.Background(), "BTC_USDT_PERP", "", "LONG", 10)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = p.SetV3FuturesLeverage(context.Background(), "BTC_USDT_PERP", "CROSS", "", 10)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = p.SetV3FuturesLeverage(context.Background(), "BTC_USDT_PERP", "CROSS", "LONG", 0)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	_, err = p.SetV3FuturesLeverage(generateContext(), "BTC_USDT_PERP", "CROSS", "LONG", 10)
	require.NoError(t, err)
}

func TestSwitchPositionMode(t *testing.T) {
	t.Parallel()
	err := p.SwitchPositionMode(context.Background(), "")
	require.ErrorIs(t, err, errPositionModeInvalid)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p, canManipulateRealOrders)
	}
	err = p.SwitchPositionMode(generateContext(), "HEDGE")
	require.NoError(t, err)
}

func TestGetPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, p)
	}
	result, err := p.GetPositionMode(generateContext())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := p.GetV3FuturesOrderBook(context.Background(), "", 100, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := p.GetV3FuturesOrderBook(context.Background(), "BTC_USDT_PERP", 100, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesKlineData(t *testing.T) {
	t.Parallel()
	_, err := p.GetV3FuturesKlineData(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = p.GetV3FuturesKlineData(context.Background(), "BTC_USDT_PERP", kline.SixHour, time.Time{}, time.Time{}, 100)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := p.GetV3FuturesKlineData(context.Background(), "BTC_USDT_PERP", kline.FiveMin, time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesExecutionInfo(t *testing.T) {
	t.Parallel()
	_, err := p.GetV3FuturesExecutionInfo(context.Background(), "", 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := p.GetV3FuturesExecutionInfo(context.Background(), "BTC_USDT_PERP", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3LiquidationOrder(t *testing.T) {
	t.Parallel()
	result, err := p.GetV3LiquidiationOrder(context.Background(), "BTC_USDT_PERP", "NEXT", time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesMarketInfo(t *testing.T) {
	t.Parallel()
	result, err := p.GetV3FuturesMarketInfo(context.Background(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesIndexPrice(t *testing.T) {
	t.Parallel()
	result, err := p.GetV3FuturesIndexPrice(context.Background(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3IndexPriceComponents(t *testing.T) {
	t.Parallel()
	_, err := p.GetV3IndexPriceComponents(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := p.GetV3IndexPriceComponents(context.Background(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPriceKlineData(t *testing.T) {
	t.Parallel()
	_, err := p.GetIndexPriceKlineData(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = p.GetIndexPriceKlineData(context.Background(), "BTC_USDT_PERP", kline.SixHour, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := p.GetIndexPriceKlineData(context.Background(), "BTC_USDT_PERP", kline.FourHour, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := p.GetV3FuturesMarkPrice(context.Background(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPriceKlineData(t *testing.T) {
	t.Parallel()
	_, err := p.GetMarkPriceKlineData(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = p.GetMarkPriceKlineData(context.Background(), "BTC_USDT_PERP", kline.SixHour, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := p.GetMarkPriceKlineData(context.Background(), "BTC_USDT_PERP", kline.FourHour, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesProductInfo(t *testing.T) {
	t.Parallel()
	_, err := p.GetV3FuturesProductInfo(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := p.GetV3FuturesProductInfo(context.Background(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesCurrentFundingRate(t *testing.T) {
	t.Parallel()
	_, err := p.GetV3FuturesCurrentFundingRate(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := p.GetV3FuturesCurrentFundingRate(context.Background(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	result, err := p.GetV3FuturesHistoricalFundingRates(context.Background(), "", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesCurrentOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := p.GetV3FuturesCurrentOpenPositions(context.Background(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := p.GetV3FuturesCurrentOpenPositions(context.Background(), "BTC_USDT_PERP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInsuranceFundInformation(t *testing.T) {
	t.Parallel()
	result, err := p.GetInsuranceFundInformation(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetV3FuturesRiskLimit(t *testing.T) {
	t.Parallel()
	result, err := p.GetV3FuturesRiskLimit(context.Background(), "")
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
