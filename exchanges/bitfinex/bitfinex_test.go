package bitfinex

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/withdraw"
)

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b Bitfinex
var wsAuthExecuted bool

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Bitfinex load config error", err)
	}
	bfxConfig, err := cfg.GetExchangeConfig("Bitfinex")
	if err != nil {
		log.Fatal("Bitfinex Setup() init error")
	}
	err = b.Setup(bfxConfig)
	if err != nil {
		log.Fatal("Bitfinex setup error", err)
	}
	b.API.Credentials.Key = apiKey
	b.API.Credentials.Secret = apiSecret
	if !b.Enabled || b.API.AuthenticatedSupport ||
		b.Verbose || b.Websocket.IsEnabled() || len(b.BaseCurrencies) < 1 {
		log.Fatal("Bitfinex Setup values not set correctly")
	}

	if areTestAPIKeysSet() {
		b.API.AuthenticatedSupport = true
		b.API.AuthenticatedWebsocketSupport = true
	}

	// custom rate limit for testing
	b.Requester.SetRateLimit(true, time.Millisecond*300, 1)
	b.Requester.SetRateLimit(false, time.Millisecond*300, 1)
	os.Exit(m.Run())
}

func TestAppendOptionalDelimiter(t *testing.T) {
	t.Parallel()
	curr1 := currency.NewPairFromString("BTCUSD")
	b.appendOptionalDelimiter(&curr1)
	if curr1.Delimiter != "" {
		t.Errorf("Expected no delimiter, received %v", curr1.Delimiter)
	}
	curr2 := currency.NewPairFromString("DUSK:USD")
	curr2.Delimiter = ""
	b.appendOptionalDelimiter(&curr2)
	if curr2.Delimiter != ":" {
		t.Errorf("Expected \"-\" as a delimiter, received %v", curr2.Delimiter)
	}
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
		t.Error("GetTicker() Expected error")
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
		t.Error("GetStats() Expected error")
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
		t.Error("Testing Failed - GetFundingBook() Expected error")
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
			}
			t.Error("BitfinexGetSymbols currency mismatch with: ", explicitSymbol)
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
		t.Error("GetAccountInfo error", err)
	}
}

func TestGetAccountFees(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetAccountFees()
	if err == nil {
		t.Error("GetAccountFees Expected error")
	}
}

func TestGetAccountSummary(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetAccountSummary()
	if err == nil {
		t.Error("GetAccountSummary() Expected error")
	}
}

func TestNewDeposit(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.NewDeposit("blabla", "testwallet", 1)
	if err == nil {
		t.Error("NewDeposit() Expected error")
	}
}

func TestGetKeyPermissions(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetKeyPermissions()
	if err == nil {
		t.Error("GetKeyPermissions() Expected error")
	}
}

func TestGetMarginInfo(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetMarginInfo()
	if err == nil {
		t.Error("GetMarginInfo() Expected error")
	}
}

func TestGetAccountBalance(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetAccountBalance()
	if err == nil {
		t.Error("GetAccountBalance() Expected error")
	}
}

func TestWalletTransfer(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.WalletTransfer(0.01, "bla", "bla", "bla")
	if err == nil {
		t.Error("WalletTransfer() Expected error")
	}
}

func TestNewOrder(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.NewOrder("BTCUSD",
		1,
		2,
		true,
		order.Limit.Lower(),
		false)
	if err == nil {
		t.Error("NewOrder() Expected error")
	}
}

func TestNewOrderMulti(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	newOrder := []PlaceOrder{
		{
			Symbol:   "BTCUSD",
			Amount:   1,
			Price:    1,
			Exchange: "bitfinex",
			Side:     order.Buy.Lower(),
			Type:     order.Limit.Lower(),
		},
	}

	_, err := b.NewOrderMulti(newOrder)
	if err == nil {
		t.Error("NewOrderMulti() Expected error")
	}
}

func TestCancelOrder(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelExistingOrder(1337)
	if err == nil {
		t.Error("CancelExistingOrder() Expected error")
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelMultipleOrders([]int64{1337, 1336})
	if err == nil {
		t.Error("CancelMultipleOrders() Expected error")
	}
}

func TestCancelAllOrders(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelAllExistingOrders()
	if err == nil {
		t.Error("CancelAllExistingOrders() Expected error")
	}
}

func TestReplaceOrder(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.ReplaceOrder(1337, "BTCUSD",
		1, 1, true, order.Limit.Lower(), false)
	if err == nil {
		t.Error("ReplaceOrder() Expected error")
	}
}

func TestGetOrderStatus(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetOrderStatus(1337)
	if err == nil {
		t.Error("GetOrderStatus() Expected error")
	}
}

func TestGetOpenOrders(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetOpenOrders()
	if err == nil {
		t.Error("GetOpenOrders() Expectederror")
	}
}

func TestGetActivePositions(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActivePositions()
	if err == nil {
		t.Error("GetActivePositions() Expected error")
	}
}

func TestClaimPosition(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.ClaimPosition(1337)
	if err == nil {
		t.Error("ClaimPosition() Expected error")
	}
}

func TestGetBalanceHistory(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetBalanceHistory("USD", time.Time{}, time.Time{}, 1, "deposit")
	if err == nil {
		t.Error("GetBalanceHistory() Expected error")
	}
}

func TestGetMovementHistory(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetMovementHistory("USD", "bitcoin", time.Time{}, time.Time{}, 1)
	if err == nil {
		t.Error("GetMovementHistory() Expected error")
	}
}

func TestGetTradeHistory(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetTradeHistory("BTCUSD", time.Time{}, time.Time{}, 1, 0)
	if err == nil {
		t.Error("GetTradeHistory() Expected error")
	}
}

func TestNewOffer(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.NewOffer("BTC", 1, 1, 1, "loan")
	if err == nil {
		t.Error("NewOffer() Expected error")
	}
}

func TestCancelOffer(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelOffer(1337)
	if err == nil {
		t.Error("CancelOffer() Expected error")
	}
}

func TestGetOfferStatus(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetOfferStatus(1337)
	if err == nil {
		t.Error("NewOffer() Expected error")
	}
}

func TestGetActiveCredits(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActiveCredits()
	if err == nil {
		t.Error("GetActiveCredits() Expected error")
	}
}

func TestGetActiveOffers(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActiveOffers()
	if err == nil {
		t.Error("GetActiveOffers() Expected error")
	}
}

func TestGetActiveMarginFunding(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActiveMarginFunding()
	if err == nil {
		t.Error("GetActiveMarginFunding() Expected error")
	}
}

func TestGetUnusedMarginFunds(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetUnusedMarginFunds()
	if err == nil {
		t.Error("GetUnusedMarginFunds() Expected error")
	}
}

func TestGetMarginTotalTakenFunds(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetMarginTotalTakenFunds()
	if err == nil {
		t.Error("GetMarginTotalTakenFunds() Expected error")
	}
}

func TestCloseMarginFunding(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CloseMarginFunding(1337)
	if err == nil {
		t.Error("CloseMarginFunding() Expected error")
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
	if !areTestAPIKeysSet() {
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
	var feeBuilder = setFeeBuilder()
	t.Parallel()

	if areTestAPIKeysSet() {
		// CryptocurrencyTradeFee Basic
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Error(err)
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := b.GetFee(feeBuilder); resp != float64(2000) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(2000), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}

		// CryptocurrencyWithdrawalFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.0004) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.0004), resp)
			t.Error(err)
		}
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.AutoWithdrawFiatWithAPIPermissionText
	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
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
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		OrderSide: order.Buy,
		OrderType: order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
	}
	response, err := b.SubmitOrder(orderSubmission)

	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
	if areTestAPIKeysSet() && !response.IsOrderPlaced {
		t.Error("Order not placed")
	}
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
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

func TestCancelAllExchangeOrdera(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
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

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.ModifyOrder(&order.Modify{})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	withdrawCryptoRequest := withdraw.CryptoWithdrawRequest{
		GenericWithdrawRequestInfo: withdraw.GenericWithdrawRequestInfo{
			Amount:      -1,
			Currency:    currency.BTC,
			Description: "WITHDRAW IT ALL",
		},
		Address: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
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
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.FiatWithdrawRequest{
		GenericWithdrawRequestInfo: withdraw.GenericWithdrawRequestInfo{
			Amount:      -1,
			Currency:    currency.USD,
			Description: "WITHDRAW IT ALL",
		},
		BankAccountName:          "Satoshi Nakamoto",
		BankAccountNumber:        "12345",
		BankAddress:              "123 Fake St",
		BankCity:                 "Tarry Town",
		BankCountry:              "Hyrule",
		BankName:                 "Federal Reserve Bank",
		WireCurrency:             currency.USD.String(),
		SwiftCode:                "Taylor",
		RequiresIntermediaryBank: false,
		IsExpressWire:            false,
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
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.FiatWithdrawRequest{
		GenericWithdrawRequestInfo: withdraw.GenericWithdrawRequestInfo{
			Amount:      -1,
			Currency:    currency.BTC,
			Description: "WITHDRAW IT ALL",
		},
		BankAccountName:               "Satoshi Nakamoto",
		BankAccountNumber:             "12345",
		BankAddress:                   "123 Fake St",
		BankCity:                      "Tarry Town",
		BankCountry:                   "Hyrule",
		BankName:                      "Federal Reserve Bank",
		WireCurrency:                  currency.USD.String(),
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

	_, err := b.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() {
		_, err := b.GetDepositAddress(currency.BTC, "deposit")
		if err != nil {
			t.Error("GetDepositAddress() error", err)
		}
	} else {
		_, err := b.GetDepositAddress(currency.BTC, "deposit")
		if err == nil {
			t.Error("GetDepositAddress() error cannot be nil")
		}
	}
}

func setupWs() {
	b.AuthenticatedWebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         b.Name,
		URL:                  authenticatedBitfinexWebsocketEndpoint,
		Verbose:              b.Verbose,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := b.AuthenticatedWebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		log.Fatal(err)
	}
	b.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	b.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	go b.WsReadData(b.AuthenticatedWebsocketConn)
	go b.WsDataHandler()
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	runAuth(t)
}

func runAuth(t *testing.T) {
	setupWs()
	err := b.WsSendAuth()
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case resp := <-b.Websocket.DataHandler:
		if logResponse, ok := resp.(map[string]interface{}); ok {
			if logResponse["event"] != "auth" && logResponse["status"] != "OK" {
				t.Error("expected successful login")
			}
		} else {
			t.Error("Unexpected response")
		}
	case <-timer.C:
		t.Error("Have not received a response")
	}
	timer.Stop()
	wsAuthExecuted = true
}

// TestWsPlaceOrder dials websocket, sends order request.
func TestWsPlaceOrder(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	_, err := b.WsNewOrder(&WsNewOrderRequest{
		CustomID: 1337,
		Type:     order.Buy.String(),
		Symbol:   "tBTCUSD",
		Amount:   10,
		Price:    -10,
	})
	if err != nil {
		t.Error(err)
	}
}

// TestWsCancelOrder dials websocket, sends cancel request.
func TestWsCancelOrder(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsCancelOrder(1234)
	if err != nil {
		t.Error(err)
	}
}

// TestWsCancelOrder dials websocket, sends modify request.
func TestWsUpdateOrder(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsModifyOrder(&WsUpdateOrderRequest{
		OrderID: 1234,
		Price:   -111,
		Amount:  111,
	})
	if err != nil {
		t.Error(err)
	}
}

// TestWsCancelAllOrders dials websocket, sends cancel all request.
func TestWsCancelAllOrders(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsCancelAllOrders()
	if err != nil {
		t.Error(err)
	}
}

// TestWsCancelAllOrders dials websocket, sends cancel all request.
func TestWsCancelMultiOrders(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsCancelMultiOrders([]int64{1, 2, 3, 4})
	if err != nil {
		t.Error(err)
	}
}

// TestWsNewOffer dials websocket, sends new offer request.
func TestWsNewOffer(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsNewOffer(&WsNewOfferRequest{
		Type:   order.Limit.String(),
		Symbol: "fBTC",
		Amount: -10,
		Rate:   10,
		Period: 30,
	})
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second)
}

// TestWsCancelOffer dials websocket, sends cancel offer request.
func TestWsCancelOffer(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsCancelOffer(1234)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second)
}
