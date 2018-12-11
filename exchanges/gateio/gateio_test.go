package gateio

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var g Gateio

func TestSetDefaults(t *testing.T) {
	g.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	gateioConfig, err := cfg.GetExchangeConfig("GateIO")
	if err != nil {
		t.Error("Test Failed - GateIO Setup() init error")
	}

	gateioConfig.AuthenticatedAPISupport = true
	gateioConfig.APIKey = apiKey
	gateioConfig.APISecret = apiSecret

	g.Setup(gateioConfig)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := g.GetSymbols()
	if err != nil {
		t.Errorf("Test failed - Gateio TestGetSymbols: %s", err)
	}
}

func TestGetMarketInfo(t *testing.T) {
	t.Parallel()
	_, err := g.GetMarketInfo()
	if err != nil {
		t.Errorf("Test failed - Gateio GetMarketInfo: %s", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()

	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}

	_, err := g.SpotNewOrder(SpotNewOrderRequestParams{
		Symbol: "btc_usdt",
		Amount: 1.1,
		Price:  10.1,
		Type:   SpotNewOrderRequestParamsTypeSell,
	})
	if err != nil {
		t.Errorf("Test failed - Gateio SpotNewOrder: %s", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}

	_, err := g.CancelExistingOrder(917591554, "btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio CancelExistingOrder: %s", err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()

	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}

	_, err := g.GetBalances()
	if err != nil {
		t.Errorf("Test failed - Gateio GetBalances: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := g.GetLatestSpotPrice("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio GetLatestSpotPrice: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := g.GetTicker("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio GetTicker: %s", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := g.GetTickers()
	if err != nil {
		t.Errorf("Test failed - Gateio GetTicker: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderbook("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio GetTicker: %s", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()

	_, err := g.GetSpotKline(KlinesRequestParams{
		Symbol:   "btc_usdt",
		GroupSec: TimeIntervalFiveMinutes, // 5 minutes or less
		HourSize: 1,                       // 1 hour data
	})

	if err != nil {
		t.Errorf("Test failed - Gateio GetSpotKline: %s", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "_",
		FeeType:             exchange.CryptocurrencyTradeFee,
		FirstCurrency:       symbol.BTC,
		SecondCurrency:      symbol.USDT,
		IsMaker:             false,
		PurchasePrice:       1,
		CurrencyItem:        symbol.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFee(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)

	var feeBuilder = setFeeBuilder()
	if apiKey != "" && apiSecret != "" {
		// CryptocurrencyTradeFee Basic
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := g.GetFee(feeBuilder); resp != float64(2000) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(2000), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	g.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText
	// Act
	withdrawPermissions := g.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Recieved: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func isRealOrderTestEnabled() bool {
	if g.APIKey == "" || g.APISecret == "" ||
		g.APIKey == "Key" || g.APISecret == "Secret" ||
		!canManipulateRealOrders {
		return false
	}
	return true
}

func TestSubmitOrder(t *testing.T) {
	g.SetDefaults()
	TestSetup(t)
	g.Verbose = true

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	var p = pair.CurrencyPair{
		Delimiter:      "_",
		FirstCurrency:  symbol.LTC,
		SecondCurrency: symbol.BTC,
	}
	response, err := g.SubmitOrder(p, exchange.Buy, exchange.Market, 1, 10, "1234234")
	if err != nil || !response.IsOrderPlaced {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	g.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	g.Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := g.CancelOrder(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	g.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	g.Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	resp, err := g.CancelAllOrders(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if apiSecret == "" || apiKey == "" {
		_, err := g.GetAccountInfo()
		if err == nil {
			t.Error("Test Failed - GetAccountInfo() error")
		}
	} else {
		_, err := g.GetAccountInfo()
		if err != nil {
			t.Error("Test Failed - GetAccountInfo() error", err)
		}
	}
}
