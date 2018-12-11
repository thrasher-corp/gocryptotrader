package binance

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"

	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
)

var b Binance

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	binanceConfig, err := cfg.GetExchangeConfig("Binance")
	if err != nil {
		t.Error("Test Failed - Binance Setup() init error")
	}

	binanceConfig.AuthenticatedAPISupport = true
	binanceConfig.APIKey = testAPIKey
	binanceConfig.APISecret = testAPISecret
	b.Setup(binanceConfig)
}

func TestGetExchangeValidCurrencyPairs(t *testing.T) {
	t.Parallel()
	_, err := b.GetExchangeValidCurrencyPairs()
	if err != nil {
		t.Error("Test Failed - Binance GetExchangeValidCurrencyPairs() error", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: "BTCUSDT",
		Limit:  10,
	})

	if err != nil {
		t.Error("Test Failed - Binance GetOrderBook() error", err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetRecentTrades(RecentTradeRequestParams{
		Symbol: "BTCUSDT",
		Limit:  15,
	})

	if err != nil {
		t.Error("Test Failed - Binance GetRecentTrades() error", err)
	}
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetHistoricalTrades("BTCUSDT", 5, 1337)
	if err == nil {
		t.Error("Test Failed - Binance GetHistoricalTrades() error", err)
	}
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetAggregatedTrades("BTCUSDT", 5)
	if err != nil {
		t.Error("Test Failed - Binance GetAggregatedTrades() error", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := b.GetSpotKline(KlinesRequestParams{
		Symbol:   "BTCUSDT",
		Interval: TimeIntervalFiveMinutes,
		Limit:    24,
	})
	if err != nil {
		t.Error("Test Failed - Binance GetSpotKline() error", err)
	}
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetAveragePrice("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetAveragePrice() error", err)
	}
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetPriceChangeStats("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetPriceChangeStats() error", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickers()
	if err != nil {
		t.Error("Test Failed - Binance TestGetTickers error", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetLatestSpotPrice("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetLatestSpotPrice() error", err)
	}
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetBestPrice("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetBestPrice() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()

	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}
	_, err := b.NewOrder(NewOrderRequest{
		Symbol:      "BTCUSDT",
		Side:        BinanceRequestParamsSideSell,
		TradeType:   BinanceRequestParamsOrderLimit,
		TimeInForce: BinanceRequestParamsTimeGTC,
		Quantity:    0.01,
		Price:       1536.1,
	})

	if err == nil {
		t.Error("Test Failed - Binance NewOrder() error", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	_, err := b.CancelExistingOrder("BTCUSDT", 82584683, "")
	if err != nil {
		t.Error("Test Failed - Binance CancelExistingOrder() error", err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()

	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	_, err := b.QueryOrder("BTCUSDT", "", 1337)
	if err == nil {
		t.Error("Test Failed - Binance QueryOrder() error", err)
	}
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()

	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	_, err := b.OpenOrders("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance OpenOrders() error", err)
	}
}

func TestAllOrders(t *testing.T) {
	t.Parallel()

	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	_, err := b.AllOrders("BTCUSDT", "", "")
	if err != nil {
		t.Error("Test Failed - Binance AllOrders() error", err)
	}
}

func TestGetAccount(t *testing.T) {
	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}
	t.Parallel()
	b.SetDefaults()
	TestSetup(t)
	account, err := b.GetAccount()
	if err != nil {
		t.Fatal("Test Failed - Binance GetAccount() error", err)
	}
	if account.MakerCommission <= 0 {
		t.Fatalf("Test Failed. Expected > 0, Recieved %d", account.MakerCommission)
	}
	if account.TakerCommission <= 0 {
		t.Fatalf("Test Failed. Expected > 0, Recieved %d", account.TakerCommission)
	}

	t.Logf("Current makerFee: %d", account.MakerCommission)
	t.Logf("Current takerFee: %d", account.TakerCommission)
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

	if testAPIKey != "" || testAPISecret != "" {
		// CryptocurrencyTradeFee Basic
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.1) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := b.GetFee(feeBuilder); resp != float64(100000) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(100000), resp)
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
	expectedResult := exchange.AutoWithdrawCryptoText
	// Act
	withdrawPermissions := b.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Recieved: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// -----------------------------------------------------------------------------------------------------------------------------

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
	resp, err := b.CancelAllOrders(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	_, err := b.GetAccountInfo()
	if err != nil {
		t.Error("test failed - GetAccountInfo() error:", err)
	}
}
