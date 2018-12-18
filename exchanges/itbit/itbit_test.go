package itbit

import (
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var i ItBit

// Please provide your own keys to do proper testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	clientID                = ""
	canManipulateRealOrders = false
)

func TestSetDefaults(t *testing.T) {
	i.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	itbitConfig, err := cfg.GetExchangeConfig("ITBIT")
	if err != nil {
		t.Error("Test Failed - Gemini Setup() init error")
	}

	itbitConfig.AuthenticatedAPISupport = true
	itbitConfig.APIKey = testAPIKey
	itbitConfig.APISecret = testAPISecret
	itbitConfig.ClientID = clientID

	i.Setup(itbitConfig)
}

// TestAreAPIKeysSet is part of a pre-commit hook to prevent commiting your API keys
func TestAreAPIKeysSet(t *testing.T) {
	var errMsg string
	// Local keys
	if testAPIKey != "" && testAPIKey != "Key" {
		errMsg += "Cannot commit populated testAPIKey. "
	}
	if testAPISecret != "" && testAPISecret != "Secret" {
		errMsg += "Cannot commit populated testAPISecret. "
	}
	if clientID != "" {
		errMsg += "Cannot commit populated clientID. "
	}
	if canManipulateRealOrders {
		errMsg += "Cannot commit with canManipulateRealOrders enabled."
	}
	//configtest.json keys
	i.SetDefaults()
	TestSetup(t)
	if i.APIKey != "" && i.APIKey != "Key" {
		errMsg += "API key present in testconfig.json"
	}
	if i.APISecret != "" && i.APISecret != "Key" {
		errMsg += "API secret key present in testconfig.json"
	}
	if len(errMsg) > 0 {
		t.Error(errMsg)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := i.GetTicker("XBTUSD")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := i.GetOrderbook("XBTSGD")
	if err != nil {
		t.Error("Test Failed - GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := i.GetTradeHistory("XBTUSD", "0")
	if err != nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestGetWallets(t *testing.T) {
	_, err := i.GetWallets(url.Values{})
	if err == nil {
		t.Error("Test Failed - GetWallets() error", err)
	}
}

func TestCreateWallet(t *testing.T) {
	_, err := i.CreateWallet("test")
	if err == nil {
		t.Error("Test Failed - CreateWallet() error", err)
	}
}

func TestGetWallet(t *testing.T) {
	_, err := i.GetWallet("1337")
	if err == nil {
		t.Error("Test Failed - GetWallet() error", err)
	}
}

func TestGetWalletBalance(t *testing.T) {
	_, err := i.GetWalletBalance("1337", "XRT")
	if err == nil {
		t.Error("Test Failed - GetWalletBalance() error", err)
	}
}

func TestGetWalletTrades(t *testing.T) {
	_, err := i.GetWalletTrades("1337", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetWalletTrades() error", err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	_, err := i.GetFundingHistoryForWallet("1337", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetFundingHistory() error", err)
	}
}

func TestPlaceOrder(t *testing.T) {
	_, err := i.PlaceOrder("1337", "buy", "limit", "USD", 1, 0.2, "banjo", "sauce")
	if err == nil {
		t.Error("Test Failed - PlaceOrder() error", err)
	}
}

func TestGetOrder(t *testing.T) {
	_, err := i.GetOrder("1337", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetOrder() error", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Skip()
	err := i.CancelExistingOrder("1337", "1337order")
	if err == nil {
		t.Error("Test Failed - CancelOrder() error", err)
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	_, err := i.GetCryptoDepositAddress("1337", "AUD")
	if err == nil {
		t.Error("Test Failed - GetCryptoDepositAddress() error", err)
	}
}

func TestWalletTransfer(t *testing.T) {
	_, err := i.WalletTransfer("1337", "mywallet", "anotherwallet", 200, "USD")
	if err == nil {
		t.Error("Test Failed - WalletTransfer() error", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "_",
		FeeType:             exchange.CryptocurrencyTradeFee,
		FirstCurrency:       symbol.BTC,
		SecondCurrency:      symbol.LTC,
		IsMaker:             false,
		PurchasePrice:       1,
		CurrencyItem:        symbol.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if resp, err := i.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := i.GetFee(feeBuilder); resp != float64(2500) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(2500), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := i.GetFee(feeBuilder); resp != float64(40) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(40), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	i.SetDefaults()
	expectedResult := exchange.WithdrawCryptoViaWebsiteOnlyText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	// Act
	withdrawPermissions := i.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Recieved: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func isRealOrderTestEnabled() bool {
	if i.APIKey == "" || i.APISecret == "" ||
		i.APIKey == "Key" || i.APISecret == "Secret" ||
		!canManipulateRealOrders {
		return false
	}
	return true
}

func TestSubmitOrder(t *testing.T) {
	i.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	var p = pair.CurrencyPair{
		Delimiter:      "",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.USDT,
	}
	response, err := i.SubmitOrder(p, exchange.Buy, exchange.Limit, 1, 10, "hi")
	if err != nil || !response.IsOrderPlaced {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	i.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := i.CancelOrder(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	i.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	resp, err := i.CancelAllOrders(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if testAPIKey != "" || testAPISecret != "" || clientID != "" {
		_, err := i.GetAccountInfo()
		if err == nil {
			t.Error("Test Failed - GetAccountInfo() error")
		}
	}
}
