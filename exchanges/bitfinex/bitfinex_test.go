package bitfinex

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
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
	os.Exit(m.Run())
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

func TestGetTickerBatch(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickerBatch()
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker("tBTCUSD")
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetTicker("fUSD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetTrades("tBTCUSD", 5, 0, 0, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook("tBTCUSD", "R0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook("fUSD", "R0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook("tBTCUSD", "P0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook("fUSD", "P0", 1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetStats("btcusd")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetFundingBook("usd")
	if err != nil {
		t.Error(err)
	}
}

func TestGetLends(t *testing.T) {
	t.Parallel()
	_, err := b.GetLends("usd", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandles(t *testing.T) {
	t.Parallel()
	_, err := b.GetCandles("fUSD", "1m", 0, 0, 10, true, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetAccountFees(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.UpdateAccountInfo()
	if err != nil {
		t.Error("GetAccountInfo error", err)
	}
}

func TestGetWithdrawalFee(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetWithdrawalFees()
	if err != nil {
		t.Error("GetAccountInfo error", err)
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
	b.Verbose = true
	_, err := b.NewDeposit("blabla", "testwallet", 0)
	if err == nil {
		t.Error("NewDeposit() Expected error")
	}

	_, err = b.NewDeposit("bitcoin", "testwallet", 0)
	if err == nil {
		t.Error("NewDeposit() Expected error")
	}

	_, err = b.NewDeposit("bitcoin", "exchange", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetKeyPermissions(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetKeyPermissions()
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginInfo(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetMarginInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetAccountBalance()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.FetchAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestWalletTransfer(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.WalletTransfer(0.01, "btc", "bla", "bla")
	if err == nil {
		t.Error("error cannot be nil")
	}
}

func TestNewOrder(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.NewOrder("BTCUSD",
		order.Limit.Lower(),
		1,
		2,
		false,
		true)
	if err == nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	_, err := b.UpdateTicker(currency.NewPairFromString("BTCUSD"), asset.Spot)
	if err != nil {
		t.Error(err)
	}
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
		WalletAddress: core.BitcoinDonationAddress,
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
		WalletAddress: core.BitcoinDonationAddress,
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

	withdrawCryptoRequest := withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: &withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
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

	var withdrawFiatRequest = withdraw.Request{
		Amount:      -1,
		Currency:    currency.USD,
		Description: "WITHDRAW IT ALL",
		Fiat: &withdraw.FiatRequest{
			Bank:         &banking.Account{},
			WireCurrency: currency.USD.String(),
		},
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

	var withdrawFiatRequest = withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Fiat: &withdraw.FiatRequest{
			Bank:                          &banking.Account{},
			WireCurrency:                  currency.USD.String(),
			RequiresIntermediaryBank:      true,
			IsExpressWire:                 false,
			IntermediaryBankAccountNumber: 12345,
			IntermediaryBankAddress:       "123 Fake St",
			IntermediaryBankCity:          "Tarry Town",
			IntermediaryBankCountry:       "Hyrule",
			IntermediaryBankName:          "Federal Reserve Bank",
			IntermediarySwiftCode:         "Taylor",
		},
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

func TestConvertSymbolToDepositMethod(t *testing.T) {
	s, err := b.ConvertSymbolToDepositMethod(currency.BTC)
	if err != nil {
		log.Fatal(err)
	}
	if s != "bitcoin" {
		t.Errorf("expected bitcoin but received %s", s)
	}

	_, err = b.ConvertSymbolToDepositMethod(currency.NewCode("CATS!"))
	if err == nil {
		log.Fatal("error cannot be nil")
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	err := b.UpdateTradablePairs(false)
	if err != nil {
		t.Error(err)
	}
}
