package gemini

import (
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please enter sandbox API keys & assigned roles for better testing procedures

const (
	apiKey1           = ""
	apiSecret1        = ""
	apiKeyRole1       = ""
	sessionHeartBeat1 = false

	apiKey2           = ""
	apiSecret2        = ""
	apiKeyRole2       = ""
	sessionHeartBeat2 = false

	canManipulateRealOrders = false
)

func TestAddSession(t *testing.T) {
	var g1 Gemini
	err := AddSession(&g1, 1, apiKey1, apiSecret1, apiKeyRole1, true, false)
	if err != nil {
		t.Error("Test failed - AddSession() error")
	}
	err = AddSession(&g1, 1, apiKey1, apiSecret1, apiKeyRole1, true, false)
	if err == nil {
		t.Error("Test failed - AddSession() error")
	}
	var g2 Gemini
	err = AddSession(&g2, 2, apiKey2, apiSecret2, apiKeyRole2, false, true)
	if err != nil {
		t.Error("Test failed - AddSession() error")
	}
}

func TestSetDefaults(t *testing.T) {
	Session[1].SetDefaults()
	Session[2].SetDefaults()
}

func TestSetup(t *testing.T) {

	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	geminiConfig, err := cfg.GetExchangeConfig("Gemini")
	if err != nil {
		t.Error("Test Failed - Gemini Setup() init error")
	}

	geminiConfig.AuthenticatedAPISupport = true

	Session[1].Setup(geminiConfig)
	Session[2].Setup(geminiConfig)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetSymbols()
	if err != nil {
		t.Error("Test Failed - GetSymbols() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTicker("BTCUSD")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
	_, err = Session[1].GetTicker("bla")
	if err == nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetOrderbook("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTrades("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetNotionalVolume(t *testing.T) {
	if apiKey2 != "" && apiSecret2 != "" {
		t.Parallel()
		_, err := Session[2].GetNotionalVolume()
		if err != nil {
			t.Error("Test Failed - GetNotionalVolume() error", err)
		}
	}
}

func TestGetAuction(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetAuction("btcusd")
	if err != nil {
		t.Error("Test Failed - GetAuction() error", err)
	}
}

func TestGetAuctionHistory(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetAuctionHistory("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetAuctionHistory() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := Session[1].NewOrder("btcusd", 1, 4500, "buy", "exchange limit")
	if err == nil {
		t.Error("Test Failed - NewOrder() error", err)
	}
	_, err = Session[2].NewOrder("btcusd", 1, 4500, "buy", "exchange limit")
	if err == nil {
		t.Error("Test Failed - NewOrder() error", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := Session[1].CancelExistingOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelExistingOrder() error", err)
	}
}

func TestCancelExistingOrders(t *testing.T) {
	t.Parallel()
	_, err := Session[1].CancelExistingOrders(false)
	if err == nil {
		t.Error("Test Failed - CancelExistingOrders() error", err)
	}
	_, err = Session[2].CancelExistingOrders(true)
	if err == nil {
		t.Error("Test Failed - CancelExistingOrders() error", err)
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetOrderStatus(1337)
	if err == nil {
		t.Error("Test Failed - GetOrderStatus() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetOrders()
	if err == nil {
		t.Error("Test Failed - GetOrders() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetTradeHistory("btcusd", 0)
	if err == nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTradeVolume()
	if err == nil {
		t.Error("Test Failed - GetTradeVolume() error", err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetBalances()
	if err == nil {
		t.Error("Test Failed - GetBalances() error", err)
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetCryptoDepositAddress("LOL123", "btc")
	if err == nil {
		t.Error("Test Failed - GetCryptoDepositAddress() error", err)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := Session[1].WithdrawCrypto("LOL123", "btc", 1)
	if err == nil {
		t.Error("Test Failed - WithdrawCrypto() error", err)
	}
}

func TestPostHeartbeat(t *testing.T) {
	t.Parallel()
	_, err := Session[2].PostHeartbeat()
	if err == nil {
		t.Error("Test Failed - PostHeartbeat() error", err)
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

	var feeBuilder = setFeeBuilder()
	if apiKey1 != "" && apiSecret1 != "" {
		// CryptocurrencyTradeFee Basic
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0.01) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.01), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(100) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(100), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0.01) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.01), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	// Act
	withdrawPermissions := Session[1].FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Recieved: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func isRealOrderTestEnabled() bool {
	if Session[1].APIKey == "" || Session[1].APISecret == "" ||
		Session[1].APIKey == "Key" || Session[1].APISecret == "Secret" ||
		!canManipulateRealOrders {
		return false
	}
	return true
}

func TestSubmitOrder(t *testing.T) {
	Session[1].SetDefaults()
	TestSetup(t)
	Session[1].Verbose = true

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	var p = pair.CurrencyPair{
		Delimiter:      "_",
		FirstCurrency:  symbol.LTC,
		SecondCurrency: symbol.BTC,
	}
	response, err := Session[1].SubmitOrder(p, exchange.Buy, exchange.Market, 1, 10, "1234234")
	if err != nil || !response.IsOrderPlaced {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	Session[1].SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	Session[1].Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := Session[1].CancelOrder(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	Session[1].SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	Session[1].Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := Session[1].CancelAllOrders(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}
