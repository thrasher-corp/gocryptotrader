package bitstamp

import (
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please add your private keys and customerID for better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	customerID              = "" // This is the customer id you use to log in
	canManipulateRealOrders = false
)

var b Bitstamp

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()

	if b.Name != "Bitstamp" {
		t.Error("Test Failed - SetDefaults() error")
	}
	if !b.Enabled {
		t.Error("Test Failed - SetDefaults() error")
	}
	if !b.Verbose {
		t.Error("Test Failed - SetDefaults() error")
	}
	if b.Websocket.IsEnabled() {
		t.Error("Test Failed - SetDefaults() error")
	}
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("Bitstamp")
	if err != nil {
		t.Error("Test Failed - Bitstamp Setup() init error")
	}
	bConfig.API.Credentials.Key = apiKey
	bConfig.API.Credentials.Secret = apiSecret
	bConfig.API.Credentials.ClientID = customerID

	b.Setup(bConfig)

	if !b.IsEnabled() || b.API.AuthenticatedSupport ||
		b.Verbose || b.Websocket.IsEnabled() || len(b.BaseCurrencies) < 1 {
		t.Error("Test Failed - Bitstamp Setup values not set correctly")
	}
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
	var feeBuilder = setFeeBuilder()
	b.GetFeeByType(feeBuilder)
	if apiKey == "" || apiSecret == "" {
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
	b.SetDefaults()
	TestSetup(t)

	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(7.5) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(7.5), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(15) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(15), resp)
		t.Error(err)
	}
}

func TestCalculateTradingFee(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	b.Balance = Balances{}
	b.Balance.BTCUSDFee = 1
	b.Balance.BTCEURFee = 0

	if resp := b.CalculateTradingFee(currency.BTC, currency.USD, 0, 0); resp != 0 {
		t.Error("Test Failed - GetFee() error")
	}
	if resp := b.CalculateTradingFee(currency.BTC, currency.USD, 2, 2); resp != float64(4) {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(4), resp)
	}
	if resp := b.CalculateTradingFee(currency.BTC, currency.EUR, 2, 2); resp != float64(0) {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}

	dummy1, dummy2 := currency.NewCode(""), currency.NewCode("")
	if resp := b.CalculateTradingFee(dummy1, dummy2, 0, 0); resp != 0 {
		t.Error("Test Failed - GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker(currency.BTC.String()+currency.USD.String(), false)
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
	_, err = b.GetTicker(currency.BTC.String()+currency.USD.String(), true)
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
	_, err = b.GetTransactions("wigwham", value)
	if err == nil {
		t.Error("Test Failed - GetTransactions() error")
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
	if err != nil {
		t.Error("Test Failed - GetBalance() error", err)
	}
}

func TestGetUserTransactions(t *testing.T) {
	t.Parallel()
	_, err := b.GetUserTransactions("")
	if err == nil {
		t.Error("Test Failed - GetUserTransactions() error", err)
	}

	_, err = b.GetUserTransactions("btcusd")
	if err == nil {
		t.Error("Test Failed - GetUserTransactions() error", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenOrders("btcusd")
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error", err)
	}
	_, err = b.GetOpenOrders("wigwham")
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error")
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	if !b.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := b.GetOrderStatus(1337)
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	resp, err := b.CancelExistingOrder(1337)
	if err == nil || resp {
		t.Error("Test Failed - CancelExistingOrder() error")
	}
}

func TestCancelAllExistingOrders(t *testing.T) {
	t.Parallel()

	_, err := b.CancelAllExistingOrders()
	if err == nil {
		t.Error("Test Failed - CancelAllExistingOrders() error", err)
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	if !b.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := b.PlaceOrder("btcusd", 0.01, 1, true, true)
	if err == nil {
		t.Error("Test Failed - PlaceOrder() error")
	}
}

func TestGetWithdrawalRequests(t *testing.T) {
	t.Parallel()

	_, err := b.GetWithdrawalRequests(0)
	if err == nil {
		t.Error("Test Failed - GetWithdrawalRequests() error", err)
	}
	_, err = b.GetWithdrawalRequests(-1)
	if err == nil {
		t.Error("Test Failed - GetWithdrawalRequests() error")
	}
}

func TestCryptoWithdrawal(t *testing.T) {
	t.Parallel()
	if !b.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := b.CryptoWithdrawal(0, "bla", "btc", "", true)
	if err == nil {
		t.Error("Test Failed - CryptoWithdrawal() error", err)
	}
}

func TestGetBitcoinDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := b.GetCryptoDepositAddress(currency.BTC)
	if err == nil {
		t.Error("Test Failed - GetCryptoDepositAddress() error", err)
	}
}

func TestGetUnconfirmedBitcoinDeposits(t *testing.T) {
	t.Parallel()

	_, err := b.GetUnconfirmedBitcoinDeposits()
	if err == nil {
		t.Error("Test Failed - GetUnconfirmedBitcoinDeposits() error", err)
	}
}

func TestTransferAccountBalance(t *testing.T) {
	t.Parallel()
	if !b.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := b.TransferAccountBalance(1, "", "", true)
	if err == nil {
		t.Error("Test Failed - TransferAccountBalance() error", err)
	}

	_, err = b.TransferAccountBalance(1, "btc", "", false)
	if err == nil {
		t.Error("Test Failed - TransferAccountBalance() error", err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	b.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.AutoWithdrawFiatText

	withdrawPermissions := b.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := b.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = currency.Pair{
		Delimiter: "",
		Base:      currency.BTC,
		Quote:     currency.USD,
	}
	response, err := b.SubmitOrder(p,
		exchange.BuyOrderSide, exchange.LimitOrderType, 1, 1, "clientId")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
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
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
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

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := b.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    currency.BTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := b.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
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
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
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
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() && customerID != "" {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("Test Failed - GetDepositAddress error", err)
		}
	} else {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress error cannot be nil")
		}
	}
}
