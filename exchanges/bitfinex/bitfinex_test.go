package bitfinex

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply your own keys here to do better tests
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
)

var b Bitfinex

func TestSetup(t *testing.T) {
	b.SetDefaults()
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bfxConfig, err := cfg.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Error("Test Failed - Bitfinex Setup() init error")
	}
	b.Setup(bfxConfig)
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret
	if !b.Enabled || b.AuthenticatedAPISupport || b.RESTPollingDelay != time.Duration(10) ||
		b.Verbose || b.Websocket.IsEnabled() || len(b.BaseCurrencies) < 1 ||
		len(b.AvailablePairs) < 1 || len(b.EnabledPairs) < 1 {
		t.Error("Test Failed - Bitfinex Setup values not set correctly")
	}
	b.AuthenticatedAPISupport = true
	// custom rate limit for testing
	b.Requester.SetRateLimit(true, time.Millisecond*300, 1)
	b.Requester.SetRateLimit(false, time.Millisecond*300, 1)
}

func TestGetPlatformStatus(t *testing.T) {
	t.Parallel()

	result, err := b.GetPlatformStatus()
	if err != nil {
		t.Errorf("TestGetPlatformStatus error: %s", err)
	}

	if result != bitfinexOperativeMode && result != bitfinexMaintenanceMode {
		t.Errorf("TestGetPlatformStatus unexpected response code")
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetLatestSpotPrice("BTCUSD")
	if err != nil {
		t.Error("Bitfinex GetLatestSpotPrice error: ", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker("BTCUSD")
	if err != nil {
		t.Error("BitfinexGetTicker init error: ", err)
	}

	_, err = b.GetTicker("wigwham")
	if err == nil {
		t.Error("Test Failed - GetTicker() error")
	}
}

func TestGetTickerV2(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickerV2("tBTCUSD")
	if err != nil {
		t.Errorf("GetTickerV2 error: %s", err)
	}

	_, err = b.GetTickerV2("fUSD")
	if err != nil {
		t.Errorf("GetTickerV2 error: %s", err)
	}
}

func TestGetTickersV2(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickersV2("tBTCUSD,fUSD")
	if err != nil {
		t.Errorf("GetTickersV2 error: %s", err)
	}
}

func TestGetStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetStats("BTCUSD")
	if err != nil {
		t.Error("BitfinexGetStatsTest init error: ", err)
	}

	_, err = b.GetStats("wigwham")
	if err == nil {
		t.Error("Test Failed - GetStats() error")
	}
}

func TestGetFundingBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetFundingBook("USD")
	if err != nil {
		t.Error("Testing Failed - GetFundingBook() error")
	}
	_, err = b.GetFundingBook("wigwham")
	if err == nil {
		t.Error("Testing Failed - GetFundingBook() error")
	}
}

func TestGetLendbook(t *testing.T) {
	t.Parallel()

	_, err := b.GetLendbook("BTCUSD", url.Values{})
	if err != nil {
		t.Error("Testing Failed - GetLendbook() error: ", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderbook("BTCUSD", url.Values{})
	if err != nil {
		t.Error("BitfinexGetOrderbook init error: ", err)
	}
}

func TestGetOrderbookV2(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderbookV2("tBTCUSD", "P0", url.Values{})
	if err != nil {
		t.Errorf("GetOrderbookV2 error: %s", err)
	}

	_, err = b.GetOrderbookV2("fUSD", "P0", url.Values{})
	if err != nil {
		t.Errorf("GetOrderbookV2 error: %s", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetTrades("BTCUSD", url.Values{})
	if err != nil {
		t.Error("BitfinexGetTrades init error: ", err)
	}
}

func TestGetTradesv2(t *testing.T) {
	t.Parallel()

	_, err := b.GetTradesV2("tBTCUSD", 0, 0, true)
	if err != nil {
		t.Error("BitfinexGetTrades init error: ", err)
	}
}

func TestGetLends(t *testing.T) {
	t.Parallel()

	_, err := b.GetLends("BTC", url.Values{})
	if err != nil {
		t.Error("BitfinexGetLends init error: ", err)
	}
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()

	symbols, err := b.GetSymbols()
	if err != nil {
		t.Fatal("BitfinexGetSymbols init error: ", err)
	}
	if reflect.TypeOf(symbols[0]).String() != "string" {
		t.Error("Bitfinex GetSymbols is not a string")
	}

	expectedCurrencies := []string{
		"rrtbtc",
		"zecusd",
		"zecbtc",
		"xmrusd",
		"xmrbtc",
		"dshusd",
		"dshbtc",
		"bccbtc",
		"bcubtc",
		"bccusd",
		"bcuusd",
		"btcusd",
		"ltcusd",
		"ltcbtc",
		"ethusd",
		"ethbtc",
		"etcbtc",
		"etcusd",
		"bfxusd",
		"bfxbtc",
		"rrtusd",
	}
	if len(expectedCurrencies) <= len(symbols) {

		for _, explicitSymbol := range expectedCurrencies {
			if common.StringDataCompare(expectedCurrencies, explicitSymbol) {
				break
			} else {
				t.Error("BitfinexGetSymbols currency mismatch with: ", explicitSymbol)
			}
		}
	} else {
		t.Error("BitfinexGetSymbols currency mismatch, Expected Currencies < Exchange Currencies")
	}
}

func TestGetSymbolsDetails(t *testing.T) {
	t.Parallel()

	_, err := b.GetSymbolsDetails()
	if err != nil {
		t.Error("BitfinexGetSymbolsDetails init error: ", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetAccountInfo()
	if err != nil {
		t.Error("Test Failed - GetAccountInfo error", err)
	}
}

func TestGetAccountFees(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetAccountFees()
	if err == nil {
		t.Error("Test Failed - GetAccountFees error")
	}
}

func TestGetAccountSummary(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetAccountSummary()
	if err == nil {
		t.Error("Test Failed - GetAccountSummary() error:")
	}
}

func TestNewDeposit(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.NewDeposit("blabla", "testwallet", 1)
	if err == nil {
		t.Error("Test Failed - NewDeposit() error:", err)
	}
}

func TestGetKeyPermissions(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetKeyPermissions()
	if err == nil {
		t.Error("Test Failed - GetKeyPermissions() error:")
	}
}

func TestGetMarginInfo(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetMarginInfo()
	if err == nil {
		t.Error("Test Failed - GetMarginInfo() error")
	}
}

func TestGetAccountBalance(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetAccountBalance()
	if err == nil {
		t.Error("Test Failed - GetAccountBalance() error")
	}
}

func TestWalletTransfer(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.WalletTransfer(0.01, "bla", "bla", "bla")
	if err == nil {
		t.Error("Test Failed - WalletTransfer() error")
	}
}

func TestNewOrder(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.NewOrder("BTCUSD", 1, 2, true, "market", false)
	if err == nil {
		t.Error("Test Failed - NewOrder() error")
	}
}

func TestNewOrderMulti(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	newOrder := []PlaceOrder{
		{
			Symbol:   "BTCUSD",
			Amount:   1,
			Price:    1,
			Exchange: "bitfinex",
			Side:     "buy",
			Type:     "market",
		},
	}

	_, err := b.NewOrderMulti(newOrder)
	if err == nil {
		t.Error("Test Failed - NewOrderMulti() error")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelExistingOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelExistingOrder() error")
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelMultipleOrders([]int64{1337, 1336})
	if err == nil {
		t.Error("Test Failed - CancelMultipleOrders() error")
	}
}

func TestCancelAllExistingOrders(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelAllExistingOrders()
	if err == nil {
		t.Error("Test Failed - CancelAllExistingOrders() error")
	}
}

func TestReplaceOrder(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.ReplaceOrder(1337, "BTCUSD", 1, 1, true, "market", false)
	if err == nil {
		t.Error("Test Failed - ReplaceOrder() error")
	}
}

func TestGetOrderStatus(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetOrderStatus(1337)
	if err == nil {
		t.Error("Test Failed - GetOrderStatus() error")
	}
}

func TestGetOpenOrders(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetOpenOrders()
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error")
	}
}

func TestGetActivePositions(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActivePositions()
	if err == nil {
		t.Error("Test Failed - GetActivePositions() error")
	}
}

func TestClaimPosition(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.ClaimPosition(1337)
	if err == nil {
		t.Error("Test Failed - ClaimPosition() error")
	}
}

func TestGetBalanceHistory(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetBalanceHistory("USD", time.Time{}, time.Time{}, 1, "deposit")
	if err == nil {
		t.Error("Test Failed - GetBalanceHistory() error")
	}
}

func TestGetMovementHistory(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetMovementHistory("USD", "bitcoin", time.Time{}, time.Time{}, 1)
	if err == nil {
		t.Error("Test Failed - GetMovementHistory() error")
	}
}

func TestGetTradeHistory(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetTradeHistory("BTCUSD", time.Time{}, time.Time{}, 1, 0)
	if err == nil {
		t.Error("Test Failed - GetTradeHistory() error")
	}
}

func TestNewOffer(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.NewOffer("BTC", 1, 1, 1, "loan")
	if err == nil {
		t.Error("Test Failed - NewOffer() error")
	}
}

func TestCancelOffer(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelOffer(1337)
	if err == nil {
		t.Error("Test Failed - CancelOffer() error")
	}
}

func TestGetOfferStatus(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetOfferStatus(1337)
	if err == nil {
		t.Error("Test Failed - NewOffer() error")
	}
}

func TestGetActiveCredits(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActiveCredits()
	if err == nil {
		t.Error("Test Failed - GetActiveCredits() error", err)
	}
}

func TestGetActiveOffers(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActiveOffers()
	if err == nil {
		t.Error("Test Failed - GetActiveOffers() error", err)
	}
}

func TestGetActiveMarginFunding(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActiveMarginFunding()
	if err == nil {
		t.Error("Test Failed - GetActiveMarginFunding() error", err)
	}
}

func TestGetUnusedMarginFunds(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetUnusedMarginFunds()
	if err == nil {
		t.Error("Test Failed - GetUnusedMarginFunds() error", err)
	}
}

func TestGetMarginTotalTakenFunds(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetMarginTotalTakenFunds()
	if err == nil {
		t.Error("Test Failed - GetMarginTotalTakenFunds() error", err)
	}
}

func TestCloseMarginFunding(t *testing.T) {
	if b.APIKey == "" || b.APISecret == "" {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CloseMarginFunding(1337)
	if err == nil {
		t.Error("Test Failed - CloseMarginFunding() error")
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

	if testAPIKey != "" || testAPISecret != "" {
		// CryptocurrencyTradeFee Basic
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := b.GetFee(feeBuilder); resp != float64(2000) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2000), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
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
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.0004) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0004), resp)
			t.Error(err)
		}
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
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	b.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.AutoWithdrawFiatWithAPIPermissionText
	// Act
	withdrawPermissions := b.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	b.Verbose = true

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := b.GetActiveOrders(getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	b.Verbose = true

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := b.GetOrderHistory(getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	if b.APIKey != "" && b.APIKey != "Key" &&
		b.APISecret != "" && b.APISecret != "Secret" {
		return true
	}
	return false
}

func TestSubmitOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	var p = pair.CurrencyPair{
		Delimiter:      "",
		FirstCurrency:  symbol.LTC,
		SecondCurrency: symbol.BTC,
	}
	response, err := b.SubmitOrder(p, exchange.BuyOrderSide, exchange.MarketOrderType, 1, 1, "clientId")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

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
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrdera(t *testing.T) {
	// Arrange
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

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
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := b.ModifyOrder(exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    symbol.BTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
	}

	_, err := b.WithdrawCryptocurrencyFunds(withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
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
		Currency:                 symbol.BTC,
		Description:              "WITHDRAW IT ALL",
		BankAccountName:          "Satoshi Nakamoto",
		BankAccountNumber:        12345,
		BankAddress:              "123 Fake St",
		BankCity:                 "Tarry Town",
		BankCountry:              "Hyrule",
		BankName:                 "Federal Reserve Bank",
		WireCurrency:             symbol.USD,
		SwiftCode:                "Taylor",
		RequiresIntermediaryBank: false,
		IsExpressWire:            false,
	}

	_, err := b.WithdrawFiatFunds(withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
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
		Currency:                      symbol.BTC,
		Description:                   "WITHDRAW IT ALL",
		BankAccountName:               "Satoshi Nakamoto",
		BankAccountNumber:             12345,
		BankAddress:                   "123 Fake St",
		BankCity:                      "Tarry Town",
		BankCountry:                   "Hyrule",
		BankName:                      "Federal Reserve Bank",
		WireCurrency:                  symbol.USD,
		SwiftCode:                     "Taylor",
		RequiresIntermediaryBank:      true,
		IsExpressWire:                 false,
		IntermediaryBankAccountNumber: 12345,
		IntermediaryBankAddress:       "123 Fake St",
		IntermediaryBankCity:          "Tarry Town",
		IntermediaryBankCountry:       "Hyrule",
		IntermediaryBankName:          "Federal Reserve Bank",
		IntermediarySwiftCode:         "Taylor",
	}

	_, err := b.WithdrawFiatFundsToInternationalBank(withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := b.GetDepositAddress(symbol.BTC, "deposit")
		if err != nil {
			t.Error("Test Failed - GetDepositAddress() error", err)
		}
	} else {
		_, err := b.GetDepositAddress(symbol.BTC, "deposit")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress() error cannot be nil")
		}
	}
}
