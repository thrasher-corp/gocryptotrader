package bitstamp

import (
	"net/url"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	"github.com/thrasher-/gocryptotrader/exchanges"

	"github.com/thrasher-/gocryptotrader/config"
)

// Please add your private keys and customerID for better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	customerID              = ""
	canManipulateRealOrders = false
)

var b Bitstamp

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()

	if b.Name != "Bitstamp" {
		t.Error("Test Failed - SetDefaults() error")
	}
	if b.Enabled != false {
		t.Error("Test Failed - SetDefaults() error")
	}
	if b.Verbose != false {
		t.Error("Test Failed - SetDefaults() error")
	}
	if b.Websocket.IsEnabled() != false {
		t.Error("Test Failed - SetDefaults() error")
	}
	if b.RESTPollingDelay != 10 {
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
	bConfig.APIKey = apiKey
	bConfig.APISecret = apiSecret
	bConfig.ClientID = customerID

	b.Setup(bConfig)

	if !b.IsEnabled() || b.RESTPollingDelay != time.Duration(10) ||
		b.Verbose || b.Websocket.IsEnabled() || len(b.BaseCurrencies) < 1 ||
		len(b.AvailablePairs) < 1 || len(b.EnabledPairs) < 1 {
		t.Error("Test Failed - Bitstamp Setup values not set correctly")
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:         1,
		Delimiter:      "",
		FeeType:        exchange.CryptocurrencyTradeFee,
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.LTC,
		IsMaker:        false,
		PurchasePrice:  1,
	}
}

func TestGetFee(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(7.5) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(7.5), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(15) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(15), resp)
		t.Error(err)
	}
}

func TestCalculateTradingFee(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	b.Balance = Balances{}
	b.Balance.BTCUSDFee = 1
	b.Balance.BTCEURFee = 0

	if resp := b.CalculateTradingFee(symbol.BTC+symbol.USD, 0, 0); resp != 0 {
		t.Error("Test Failed - GetFee() error")
	}
	if resp := b.CalculateTradingFee(symbol.BTC+symbol.USD, 2, 2); resp != float64(4) {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(4), resp)
	}
	if resp := b.CalculateTradingFee(symbol.BTC+symbol.EUR, 2, 2); resp != float64(0) {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}
	if resp := b.CalculateTradingFee("bla", 0, 0); resp != 0 {
		t.Error("Test Failed - GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker(symbol.BTC+symbol.USD, false)
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
	_, err = b.GetTicker(symbol.BTC+symbol.USD, true)
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook(symbol.BTC + symbol.USD)
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

	_, err := b.GetTransactions(symbol.BTC+symbol.USD, value)
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
	if b.APIKey == "" || b.APISecret == "" ||
		b.APIKey == "Key" || b.APISecret == "Secret" {
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
	if err == nil || resp != false {
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
	if b.APIKey == "" || b.APISecret == "" ||
		b.APIKey == "Key" || b.APISecret == "Secret" {
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
	if b.APIKey == "" || b.APISecret == "" ||
		b.APIKey == "Key" || b.APISecret == "Secret" {
		t.Skip()
	}

	_, err := b.CryptoWithdrawal(0, "bla", "btc", "", true)
	if err == nil {
		t.Error("Test Failed - CryptoWithdrawal() error", err)
	}
}

func TestGetBitcoinDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := b.GetCryptoDepositAddress("btc")
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
	if b.APIKey == "" || b.APISecret == "" ||
		b.APIKey == "Key" || b.APISecret == "Secret" {
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
	// Arrange
	b.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.AutoWithdrawFiatText
	// Act
	withdrawPermissions := b.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Recieved: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func isRealOrderTestEnabled() bool {
	if b.APIKey == "" || b.APISecret == "" ||
		b.APIKey == "Key" || b.APISecret == "Secret" ||
		!canManipulateRealOrders {
		return false
	}
	return true
}

func TestSubmitOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	var p = pair.CurrencyPair{
		Delimiter:      "",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.USD,
	}
	response, err := b.SubmitOrder(p, exchange.Buy, exchange.Market, 1, 1, "clientId")
	if err != nil || !response.IsOrderPlaced {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	b.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	b.Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := b.CancelOrder(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	b.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	b.Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := b.CancelAllOrders(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}
