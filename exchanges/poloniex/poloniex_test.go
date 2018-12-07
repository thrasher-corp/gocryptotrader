package poloniex

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var p Poloniex

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

func TestSetDefaults(t *testing.T) {
	p.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	poloniexConfig, err := cfg.GetExchangeConfig("Poloniex")
	if err != nil {
		t.Error("Test Failed - Poloniex Setup() init error")
	}

	poloniexConfig.AuthenticatedAPISupport = true
	poloniexConfig.APIKey = apiKey
	poloniexConfig.APISecret = apiSecret

	p.Setup(poloniexConfig)
}

func TestGetTicker(t *testing.T) {
	_, err := p.GetTicker()
	if err != nil {
		t.Error("Test faild - Poloniex GetTicker() error")
	}
}

func TestGetVolume(t *testing.T) {
	_, err := p.GetVolume()
	if err != nil {
		t.Error("Test faild - Poloniex GetVolume() error")
	}
}

func TestGetOrderbook(t *testing.T) {
	_, err := p.GetOrderbook("BTC_XMR", 50)
	if err != nil {
		t.Error("Test faild - Poloniex GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	_, err := p.GetTradeHistory("BTC_XMR", "", "")
	if err != nil {
		t.Error("Test faild - Poloniex GetTradeHistory() error", err)
	}
}

func TestGetChartData(t *testing.T) {
	_, err := p.GetChartData("BTC_XMR", "1405699200", "1405699400", "300")
	if err != nil {
		t.Error("Test faild - Poloniex GetChartData() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := p.GetCurrencies()
	if err != nil {
		t.Error("Test faild - Poloniex GetCurrencies() error", err)
	}
}

func TestGetLoanOrders(t *testing.T) {
	_, err := p.GetLoanOrders("BTC")
	if err != nil {
		t.Error("Test faild - Poloniex GetLoanOrders() error", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "-",
		FeeType:             exchange.CryptocurrencyTradeFee,
		FirstCurrency:       symbol.LTC,
		SecondCurrency:      symbol.BTC,
		IsMaker:             false,
		PurchasePrice:       1,
		CurrencyItem:        symbol.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFee(t *testing.T) {
	p.SetDefaults()
	TestSetup(t)
	var feeBuilder = setFeeBuilder()

	if apiKey != "" && apiSecret != "" {
		// CryptocurrencyTradeFee Basic
		if resp, err := p.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.0015), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := p.GetFee(feeBuilder); resp != float64(2000) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(2000), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := p.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.001), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	p.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText
	// Act
	withdrawPermissions := p.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Recieved: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func isRealOrderTestEnabled() bool {
	if p.APIKey == "" || p.APISecret == "" ||
		p.APIKey == "Key" || p.APISecret == "Secret" ||
		!canManipulateRealOrders {
		return false
	}
	return true
}

func TestSubmitOrder(t *testing.T) {
	p.SetDefaults()
	TestSetup(t)
	p.Verbose = true

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	var pair = pair.CurrencyPair{
		Delimiter:      "_",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.LTC,
	}
	response, err := p.SubmitOrder(pair, exchange.Buy, exchange.Market, 1, 10, "hi")
	if err != nil || !response.IsOrderPlaced {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	p.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	p.Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := p.CancelOrder(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	p.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	p.Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := p.CancelAllOrders(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}
