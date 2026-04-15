package yobit

import (
	"log"
	"math"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var e *Exchange

// Please supply your own keys for better unit testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var testPair = currency.NewBTCUSD().Format(currency.PairFormat{Delimiter: "_"})

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Yobit Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	os.Exit(m.Run())
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	if err != nil {
		t.Errorf("FetchTradablePairs err: %s", err)
	}
}

func TestGetInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetInfo(t.Context())
	if err != nil {
		t.Error("GetInfo() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), testPair.String())
	assert.NoError(t, err, "GetTicker should not error")
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepth(t.Context(), testPair.String())
	assert.NoError(t, err, "GetDepth should not error")
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), testPair.String())
	assert.NoError(t, err, "GetTrades should not error")
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), "")
	if err == nil {
		t.Error("GetOpenOrders() Expected error")
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetOrderInfo(t.Context(), "1337", currency.NewBTCUSD(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetCryptoDepositAddress(t.Context(), "bTc", false)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := e.CancelExistingOrder(t.Context(), 1337)
	if err == nil {
		t.Error("CancelOrder() Expected error")
	}
}

func TestTrade(t *testing.T) {
	t.Parallel()
	_, err := e.Trade(t.Context(), "", order.Buy.String(), 0, 0)
	if err == nil {
		t.Error("Trade() Expected error")
	}
}

func TestWithdrawCoinsToAddress(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCoinsToAddress(t.Context(), "", 0, "")
	if err == nil {
		t.Error("WithdrawCoinsToAddress() Expected error")
	}
}

func TestCreateYobicode(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCoupon(t.Context(), "bla", 0)
	if err == nil {
		t.Error("CreateYobicode() Expected error")
	}
}

func TestRedeemYobicode(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemCoupon(t.Context(), "bla2")
	if err == nil {
		t.Error("RedeemYobicode() Expected error")
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
	feeBuilder := setFeeBuilder()
	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	feeBuilder := setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee QIWI
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.Qiwi
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Wire
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.WireTransfer
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Payeer
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.Payeer
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Capitalist
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.RUR
	feeBuilder.BankTransactionType = exchange.Capitalist
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee AdvCash
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	feeBuilder.BankTransactionType = exchange.AdvCash
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee PerfectMoney
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.RUR
	feeBuilder.BankTransactionType = exchange.PerfectMoney
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := e.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Pairs:     []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(math.MaxInt64, 0),
		Side:      order.AnySide,
	}

	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// TestSubmitOrder and below can impact your orders on the exchange. Enable canManipulateRealOrders to run them
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	orderSubmission := &order.Submit{
		Exchange: e.Name,
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := e.SubmitOrder(t.Context(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(e) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	err := e.CancelOrder(t.Context(), orderCancellation)
	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	resp, err := e.CancelAllOrders(t.Context(), orderCancellation)

	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	_, err := e.ModifyOrder(t.Context(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}
	_, err := e.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if sharedtestvalues.AreAPICredentialsSet(e) {
		_, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
		if err != nil {
			t.Error(err)
		}
	} else {
		_, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
		if err == nil {
			t.Error("GetDepositAddress() error")
		}
	}
}

func TestGetRecentTrades(t *testing.T) {
	_, err := e.GetRecentTrades(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err, "GetRecentTrades should not error")
}

func TestGetHistoricTrades(t *testing.T) {
	_, err := e.GetHistoricTrades(t.Context(), testPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err, "UpdateTicker should not error")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := e.UpdateTickers(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)

	if st.IsZero() {
		t.Fatal("expected a time")
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}
