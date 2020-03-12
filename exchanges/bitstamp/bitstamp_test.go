package bitstamp

import (
	"net/url"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please add your private keys and customerID for better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	customerID              = "" // This is the customer id you use to log in
	canManipulateRealOrders = false
)

var b Bitstamp

func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials()
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()

	var feeBuilder = setFeeBuilder()
	b.GetFeeByType(feeBuilder)
	if !areTestAPIKeysSet() {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v",
				exchange.OfflineTradeFee,
				feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v",
				exchange.CryptocurrencyTradeFee,
				feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()

	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Error(err)
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(7.5) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(7.5),
			resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(15) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(15),
			resp)
		t.Error(err)
	}
}

func TestCalculateTradingFee(t *testing.T) {
	t.Parallel()

	newBalance := make(Balances)
	newBalance["BTC"] = Balance{
		USDFee: 1,
		EURFee: 0,
	}

	if resp := b.CalculateTradingFee(currency.BTC, currency.USD, 0, 0, newBalance); resp != 0 {
		t.Error("GetFee() error")
	}
	if resp := b.CalculateTradingFee(currency.BTC, currency.USD, 2, 2, newBalance); resp != float64(4) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(4), resp)
	}
	if resp := b.CalculateTradingFee(currency.BTC, currency.EUR, 2, 2, newBalance); resp != float64(0) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}

	dummy1, dummy2 := currency.NewCode(""), currency.NewCode("")
	if resp := b.CalculateTradingFee(dummy1, dummy2, 0, 0, newBalance); resp != 0 {
		t.Error("GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()

	_, err := b.GetTicker(currency.BTC.String()+currency.USD.String(), false)
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderbook(currency.BTC.String() + currency.USD.String())
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetTradingPairs(t *testing.T) {
	t.Parallel()

	_, err := b.GetTradingPairs()
	if err != nil {
		t.Error("GetTradingPairs() error", err)
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()

	value := url.Values{}
	value.Set("time", "hour")

	_, err := b.GetTransactions(currency.BTC.String()+currency.USD.String(), value)
	if err != nil {
		t.Error("GetTransactions() error", err)
	}
}

func TestGetEURUSDConversionRate(t *testing.T) {
	t.Parallel()

	_, err := b.GetEURUSDConversionRate()
	if err != nil {
		t.Error("GetEURUSDConversionRate() error", err)
	}
}

func TestGetBalance(t *testing.T) {
	t.Parallel()

	_, err := b.GetBalance()
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetBalance() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetBalance() error", err)
	}
}

func TestGetUserTransactions(t *testing.T) {
	t.Parallel()

	_, err := b.GetUserTransactions("btcusd")
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetUserTransactions() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetUserTransactions() error", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenOrders("btcusd")
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetOpenOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetOpenOrders() error", err)
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderStatus(1337)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetOrderStatus() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Error("Expecting an error until a QA pass can be completed")
	}
}

func TestGetWithdrawalRequests(t *testing.T) {
	t.Parallel()

	_, err := b.GetWithdrawalRequests(0)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetWithdrawalRequests() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetWithdrawalRequests() error", err)
	}
}

func TestGetUnconfirmedBitcoinDeposits(t *testing.T) {
	t.Parallel()

	_, err := b.GetUnconfirmedBitcoinDeposits()
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetUnconfirmedBitcoinDeposits() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetUnconfirmedBitcoinDeposits() error", err)
	}
}

func TestTransferAccountBalance(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() && !mockTests {
		t.Skip()
	}

	err := b.TransferAccountBalance(0.01, "btc", "testAccount", true)
	if !mockTests && err != nil {
		t.Error("TransferAccountBalance() error", err)
	}
	if mockTests && err == nil {
		t.Error("Expecting an error until a QA pass can be completed")
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()

	expectedResult := exchange.AutoWithdrawCryptoText +
		" & " +
		exchange.AutoWithdrawFiatText
	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s",
			expectedResult,
			withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()

	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not get open orders: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not get open orders: %s", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()

	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}

	_, err := b.GetOrderHistory(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not get order history: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not get order history: %s", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.USD,
		},
		OrderSide: order.Buy,
		OrderType: order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
	}
	response, err := b.SubmitOrder(orderSubmission)
	switch {
	case areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) && !mockTests:
		t.Errorf("Order failed to be placed: %v", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass is completed")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	orderCancellation := &order.Cancel{
		OrderID: "1234",
	}
	err := b.CancelOrder(orderCancellation)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass is completed")
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	resp, err := b.CancelAllOrders(&order.Cancel{})
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()

	_, err := b.ModifyOrder(&order.Modify{})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	withdrawCryptoRequest := withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: &withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := b.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass is completed")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{
		Fiat: &withdraw.FiatRequest{
			Bank: &banking.Account{
				AccountName:    "Satoshi Nakamoto",
				AccountNumber:  "12345",
				BankAddress:    "123 Fake St",
				BankPostalCity: "Tarry Town",
				BankCountry:    "AU",
				BankName:       "Federal Reserve Bank",
				SWIFTCode:      "CTBAAU2S",
				BankPostalCode: "2088",
				IBAN:           "IT60X0542811101000000123456",
			},
			WireCurrency:             currency.USD.String(),
			RequiresIntermediaryBank: false,
			IsExpressWire:            false,
		},
		Amount:      -1,
		Currency:    currency.USD,
		Description: "WITHDRAW IT ALL",
	}

	_, err := b.WithdrawFiatFunds(&withdrawFiatRequest)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass is completed")
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{
		Fiat: &withdraw.FiatRequest{
			Bank: &banking.Account{
				AccountName:    "Satoshi Nakamoto",
				AccountNumber:  "12345",
				BankAddress:    "123 Fake St",
				BankPostalCity: "Tarry Town",
				BankCountry:    "AU",
				BankName:       "Federal Reserve Bank",
				SWIFTCode:      "CTBAAU2S",
				BankPostalCode: "2088",
				IBAN:           "IT60X0542811101000000123456",
			},
			WireCurrency:                  currency.USD.String(),
			RequiresIntermediaryBank:      false,
			IsExpressWire:                 false,
			IntermediaryBankAccountNumber: 12345,
			IntermediaryBankAddress:       "123 Fake St",
			IntermediaryBankCity:          "Tarry Town",
			IntermediaryBankCountry:       "AU",
			IntermediaryBankName:          "Federal Reserve Bank",
			IntermediaryBankPostalCode:    "2088",
		},
		Amount:      -1,
		Currency:    currency.USD,
		Description: "WITHDRAW IT ALL",
	}

	_, err := b.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass is completed")
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := b.GetDepositAddress(currency.BTC, "")
	switch {
	case areTestAPIKeysSet() && customerID != "" && err != nil && !mockTests:
		t.Error("GetDepositAddress error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetDepositAddress error cannot be nil")
	case mockTests && err != nil:
		t.Error("GetDepositAddress error", err)
	}
}

func TestParseTime(t *testing.T) {
	t.Parallel()

	tm, err := parseTime("2019-10-18 01:55:14")
	if err != nil {
		t.Error(err)
	}

	if tm.Year() != 2019 ||
		tm.Month() != 10 ||
		tm.Day() != 18 ||
		tm.Hour() != 1 ||
		tm.Minute() != 55 ||
		tm.Second() != 14 {
		t.Error("invalid time values")
	}
}
