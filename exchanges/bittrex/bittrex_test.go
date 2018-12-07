package bittrex

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply you own test keys here to run better tests.
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b Bittrex

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
	if b.GetName() != "Bittrex" {
		t.Error("Test Failed - Bittrex - SetDefaults() error")
	}
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("Bittrex")
	if err != nil {
		t.Error("Test Failed - Bittrex Setup() init error")
	}
	bConfig.APIKey = apiKey
	bConfig.APISecret = apiSecret
	bConfig.AuthenticatedAPISupport = true

	b.Setup(bConfig)

	if !b.IsEnabled() ||
		b.RESTPollingDelay != time.Duration(10) || b.Verbose ||
		b.Websocket.IsEnabled() || len(b.BaseCurrencies) < 1 ||
		len(b.AvailablePairs) < 1 || len(b.EnabledPairs) < 1 {
		t.Error("Test Failed - Bittrex Setup values not set correctly")
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets()
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarkets() error: %s", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := b.GetCurrencies()
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetCurrencies() error: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	btc := "btc-ltc"

	_, err := b.GetTicker(btc)
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetTicker() error: %s", err)
	}
}

func TestGetMarketSummaries(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketSummaries()
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarketSummaries() error: %s", err)
	}
}

func TestGetMarketSummary(t *testing.T) {
	t.Parallel()
	pairOne := "BTC-LTC"

	_, err := b.GetMarketSummary(pairOne)
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarketSummary() error: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderbook("btc-ltc")
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetOrderbook() error: %s", err)
	}
}

func TestGetMarketHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetMarketHistory("btc-ltc")
	if err != nil {
		t.Errorf("Test Failed - Bittrex - GetMarketHistory() error: %s", err)
	}
}

func TestPlaceBuyLimit(t *testing.T) {
	t.Parallel()

	_, err := b.PlaceBuyLimit("btc-ltc", 1, 1)
	if err == nil {
		t.Error("Test Failed - Bittrex - PlaceBuyLimit() error")
	}
}

func TestPlaceSellLimit(t *testing.T) {
	t.Parallel()

	_, err := b.PlaceSellLimit("btc-ltc", 1, 1)
	if err == nil {
		t.Error("Test Failed - Bittrex - PlaceSellLimit() error")
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenOrders("")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrder() error")
	}
	_, err = b.GetOpenOrders("btc-ltc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrder() error")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	_, err := b.CancelExistingOrder("blaaaaaaa")
	if err == nil {
		t.Error("Test Failed - Bittrex - CancelExistingOrder() error")
	}
}

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountBalances()
	if err == nil {
		t.Error("Test Failed - Bittrex - GetAccountBalances() error")
	}
}

func TestGetAccountBalanceByCurrency(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountBalanceByCurrency("btc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetAccountBalanceByCurrency() error")
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := b.GetDepositAddress("btc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetDepositAddress() error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	_, err := b.Withdraw("btc", "something", "someplace", 1)
	if err == nil {
		t.Error("Test Failed - Bittrex - Withdraw() error")
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrder("0cb4c4e4-bdc7-4e13-8c13-430e587d2cc1")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrder() error")
	}
	_, err = b.GetOrder("")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrder() error")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderHistory("")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrderHistory() error")
	}
	_, err = b.GetOrderHistory("btc-ltc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetOrderHistory() error")
	}
}

func TestGetwithdrawalHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetWithdrawalHistory("")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetWithdrawalHistory() error")
	}
	_, err = b.GetWithdrawalHistory("btc-ltc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetWithdrawalHistory() error")
	}
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetDepositHistory("")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetDepositHistory() error")
	}
	_, err = b.GetDepositHistory("btc-ltc")
	if err == nil {
		t.Error("Test Failed - Bittrex - GetDepositHistory() error")
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
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.0025), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(2500) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(2500), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.0025), resp)
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
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0005) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.0005), resp)
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
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	b.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText
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
		Delimiter:      "-",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.LTC,
	}
	response, err := b.SubmitOrder(p, exchange.Buy, exchange.Limit, 1, 1, "clientId")
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
