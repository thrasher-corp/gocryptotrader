package bitflyer

import (
	"log"
	"testing"

	"github.com/thrasher-/gocryptotrader/currency/symbol"
	"github.com/thrasher-/gocryptotrader/exchanges"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
)

var b Bitflyer

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bitflyerConfig, err := cfg.GetExchangeConfig("Bitflyer")
	if err != nil {
		t.Error("Test Failed - bitflyer Setup() init error")
	}

	bitflyerConfig.AuthenticatedAPISupport = true
	bitflyerConfig.APIKey = testAPIKey
	bitflyerConfig.APISecret = testAPISecret

	b.Setup(bitflyerConfig)
}

func TestGetLatestBlockCA(t *testing.T) {
	t.Parallel()
	_, err := b.GetLatestBlockCA()
	if err != nil {
		t.Error("test failed - Bitflyer - GetLatestBlockCA() error:", err)
	}
}

func TestGetBlockCA(t *testing.T) {
	t.Parallel()
	_, err := b.GetBlockCA("000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f")
	if err != nil {
		t.Error("test failed - Bitflyer - GetBlockCA() error:", err)
	}
}

func TestGetBlockbyHeightCA(t *testing.T) {
	t.Parallel()
	_, err := b.GetBlockbyHeightCA(0)
	if err != nil {
		t.Error("test failed - Bitflyer - GetBlockbyHeightCA() error:", err)
	}
}

func TestGetTransactionByHashCA(t *testing.T) {
	t.Parallel()
	_, err := b.GetTransactionByHashCA("0562d1f063cd4127053d838b165630445af5e480ceb24e1fd9ecea52903cb772")
	if err != nil {
		t.Error("test failed - Bitflyer - GetTransactionByHashCA() error:", err)
	}
}

func TestGetAddressInfoCA(t *testing.T) {
	t.Parallel()
	v, err := b.GetAddressInfoCA("1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB")
	if err != nil {
		t.Error("test failed - Bitflyer - GetAddressInfoCA() error:", err)
	}
	if v.UnconfirmedBalance == 0 || v.ConfirmedBalance == 0 {
		log.Println("WARNING!: Donation wallet is empty :( - please consider donating")
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets()
	if err != nil {
		t.Error("test failed - Bitflyer - GetMarkets() error:", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook("BTC_JPY")
	if err != nil {
		t.Error("test failed - Bitflyer - GetOrderBook() error:", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker("BTC_JPY")
	if err != nil {
		t.Error("test failed - Bitflyer - GetTicker() error:", err)
	}
}

func TestGetExecutionHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetExecutionHistory("BTC_JPY")
	if err != nil {
		t.Error("test failed - Bitflyer - GetExecutionHistory() error:", err)
	}
}

func TestGetExchangeStatus(t *testing.T) {
	t.Parallel()
	_, err := b.GetExchangeStatus()
	if err != nil {
		t.Error("test failed - Bitflyer - GetExchangeStatus() error:", err)
	}
}

func TestCheckFXString(t *testing.T) {
	t.Parallel()
	p := pair.NewCurrencyPairDelimiter("FXBTC_JPY", "_")
	p = b.CheckFXString(p)
	if p.FirstCurrency.String() != "FX_BTC" {
		t.Error("test failed - Bitflyer - CheckFXString() error")
	}
}

func TestGetTickerPrice(t *testing.T) {
	t.Parallel()
	var p pair.CurrencyPair

	currencies := b.GetAvailableCurrencies()
	for _, pair := range currencies {
		if pair.Pair().String() == "FXBTC_JPY" {
			p = pair
			break
		}
	}

	_, err := b.GetTickerPrice(p, b.AssetTypes[0])
	if err != nil {
		t.Error("test failed - Bitflyer - GetTickerPrice() error", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "",
		FeeType:             exchange.CryptocurrencyTradeFee,
		FirstCurrency:       symbol.BTC,
		SecondCurrency:      symbol.LTC,
		IsMaker:             false,
		PurchasePrice:       1,
		CurrencyItem:        symbol.JPY,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFee(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	var feeBuilder = setFeeBuilder()

	if testAPIKey != "" || testAPISecret != "" {
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
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.1) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.1), resp)
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
	feeBuilder.CurrencyItem = symbol.JPY
	if resp, err := b.GetFee(feeBuilder); resp != float64(324) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(324), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.JPY
	if resp, err := b.GetFee(feeBuilder); resp != float64(540) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(540), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	b.SetDefaults()
	expectedResult := exchange.AutoWithdrawFiatText + " & " + exchange.WithdrawCryptoViaWebsiteOnlyText
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
		FirstCurrency:  symbol.LTC,
		SecondCurrency: symbol.BTC,
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
