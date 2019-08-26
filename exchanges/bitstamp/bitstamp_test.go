package bitstamp

import (
	"net/url"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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
	if b.APIKey != "" && b.APIKey != "Key" &&
		b.APISecret != "" && b.APISecret != "Secret" {
		return true
	}
	return false
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
	if !areTestAPIKeysSet() || mockTests {
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
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(7.5) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(7.5),
			resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(15) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(15),
			resp)
		t.Error(err)
	}
}

func TestCalculateTradingFee(t *testing.T) {
	t.Parallel()

	var newBalance = new(Balances)
	newBalance.BTCUSDFee = 1
	newBalance.BTCEURFee = 0

	if resp := b.CalculateTradingFee(currency.BTC, currency.USD, 0, 0, newBalance); resp != 0 {
		t.Error("Test Failed - GetFee() error")
	}
	if resp := b.CalculateTradingFee(currency.BTC, currency.USD, 2, 2, newBalance); resp != float64(4) {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(4), resp)
	}
	if resp := b.CalculateTradingFee(currency.BTC, currency.EUR, 2, 2, newBalance); resp != float64(0) {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}

	dummy1, dummy2 := currency.NewCode(""), currency.NewCode("")
	if resp := b.CalculateTradingFee(dummy1, dummy2, 0, 0, newBalance); resp != 0 {
		t.Error("Test Failed - GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()

	_, err := b.GetTicker(currency.BTC.String()+currency.USD.String(), false)
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderbook(currency.BTC.String() + currency.USD.String())
	if err != nil {
		t.Error("Test Failed - GetOrderbook() error", err)
	}
}

func TestGetTradingPairs(t *testing.T) {
	t.Parallel()

	_, err := b.GetTradingPairs()
	if err != nil {
		t.Error("Test Failed - GetTradingPairs() error", err)
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()

	value := url.Values{}
	value.Set("time", "hour")

	_, err := b.GetTransactions(currency.BTC.String()+currency.USD.String(), value)
	if err != nil {
		t.Error("Test Failed - GetTransactions() error", err)
	}
}

func TestGetEURUSDConversionRate(t *testing.T) {
	t.Parallel()

	_, err := b.GetEURUSDConversionRate()
	if err != nil {
		t.Error("Test Failed - GetEURUSDConversionRate() error", err)
	}
}

func TestGetBalance(t *testing.T) {
	t.Parallel()

	_, err := b.GetBalance()
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("Test Failed - GetBalance() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - GetBalance() error", err)
	}
}

func TestGetUserTransactions(t *testing.T) {
	t.Parallel()

	_, err := b.GetUserTransactions("btcusd")
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("Test Failed - GetUserTransactions() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - GetUserTransactions() error", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenOrders("btcusd")
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("Test Failed - GetOpenOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - GetOpenOrders() error", err)
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderStatus(1337)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("Test Failed - GetOrderStatus() error", err)
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
		t.Error("Test Failed - GetWithdrawalRequests() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - GetWithdrawalRequests() error", err)
	}
}

func TestGetUnconfirmedBitcoinDeposits(t *testing.T) {
	t.Parallel()

	_, err := b.GetUnconfirmedBitcoinDeposits()
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("Test Failed - GetUnconfirmedBitcoinDeposits() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - GetUnconfirmedBitcoinDeposits() error", err)
	}
}

func TestTransferAccountBalance(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() && !mockTests {
		t.Skip()
	}

	err := b.TransferAccountBalance(0.01, "btc", "testAccount", true)
	if !mockTests && err != nil {
		t.Error("Test Failed - TransferAccountBalance() error", err)
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

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
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

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
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

	var p = currency.Pair{
		Delimiter: "",
		Base:      currency.BTC,
		Quote:     currency.USD,
	}
	response, err := b.SubmitOrder(p,
		exchange.BuyOrderSide,
		exchange.MarketOrderType,
		1,
		1,
		"clientId")
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

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
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

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := b.CancelAllOrders(orderCancellation)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()

	_, err := b.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    currency.BTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
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

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:                   100,
		Currency:                 currency.USD,
		Description:              "WITHDRAW IT ALL",
		BankAccountName:          "Satoshi Nakamoto",
		BankAccountNumber:        12345,
		BankAddress:              "123 Fake St",
		BankCity:                 "Tarry Town",
		BankCountry:              "AU",
		BankName:                 "Federal Reserve Bank",
		WireCurrency:             currency.USD.String(),
		SwiftCode:                "CTBAAU2S",
		RequiresIntermediaryBank: false,
		IsExpressWire:            false,
		BankPostalCode:           "2088",
		IBAN:                     "IT60X0542811101000000123456",
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

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:                        100,
		Currency:                      currency.USD,
		Description:                   "WITHDRAW IT ALL",
		BankAccountName:               "Satoshi Nakamoto",
		BankAccountNumber:             12345,
		BankAddress:                   "123 Fake St",
		BankCity:                      "Tarry Town",
		BankCountry:                   "AU",
		BankName:                      "Federal Reserve Bank",
		WireCurrency:                  currency.USD.String(),
		SwiftCode:                     "CTBAAU2S",
		RequiresIntermediaryBank:      false,
		IsExpressWire:                 false,
		BankPostalCode:                "2088",
		IBAN:                          "IT60X0542811101000000123456",
		IntermediaryBankAccountNumber: 12345,
		IntermediaryBankAddress:       "123 Fake St",
		IntermediaryBankCity:          "Tarry Town",
		IntermediaryBankCountry:       "AU",
		IntermediaryBankName:          "Federal Reserve Bank",
		IntermediaryBankPostalCode:    "2088",
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
		t.Error("Test Failed - GetDepositAddress error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - GetDepositAddress error cannot be nil")
	case mockTests && err != nil:
		t.Error("Test Failed - GetDepositAddress error", err)
	}
}
