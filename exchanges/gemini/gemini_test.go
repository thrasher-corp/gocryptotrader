package gemini

import (
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
)

// Please enter sandbox API keys & assigned roles for better testing procedures
const (
	apiKey                  = ""
	apiSecret               = ""
	apiKeyRole              = ""
	sessionHeartBeat        = false
	canManipulateRealOrders = false
)

const testCurrency = "btcusd"

var g Gemini

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := g.GetSymbols()
	if err != nil {
		t.Error("Test Failed - GetSymbols() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := g.GetTicker("BTCUSD")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
	_, err = g.GetTicker("bla")
	if err == nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderbook(testCurrency, url.Values{})
	if err != nil {
		t.Error("Test Failed - GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := g.GetTrades(testCurrency, url.Values{})
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetNotionalVolume(t *testing.T) {
	t.Parallel()
	_, err := g.GetNotionalVolume()
	if err != nil && mockTests {
		t.Error("Test Failed - GetNotionalVolume() error", err)
	} else if err == nil && !mockTests {
		t.Error("Test Failed - GetNotionalVolume() error cannot be nil")
	}
}

func TestGetAuction(t *testing.T) {
	t.Parallel()
	_, err := g.GetAuction(testCurrency)
	if err != nil {
		t.Error("Test Failed - GetAuction() error", err)
	}
}

func TestGetAuctionHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetAuctionHistory(testCurrency, url.Values{})
	if err != nil {
		t.Error("Test Failed - GetAuctionHistory() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := g.NewOrder(testCurrency, 1, 9000, "buy", "exchange limit")
	if err != nil && mockTests {
		t.Error("Test Failed - NewOrder() error", err)
	} else if err == nil && !mockTests {
		t.Error("Test Failed - NewOrder() error cannot be nil")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := g.CancelExistingOrder(265555413)
	if err != nil && mockTests {
		t.Error("Test Failed - CancelExistingOrder() error", err)
	} else if err == nil && !mockTests {
		t.Error("Test Failed - CancelExistingOrder() error cannot be nil")
	}
}

func TestCancelExistingOrders(t *testing.T) {
	t.Parallel()
	_, err := g.CancelExistingOrders(false)
	if err != nil && mockTests {
		t.Error("Test Failed - CancelExistingOrders() error", err)
	} else if err == nil && !mockTests {
		t.Error("Test Failed - CancelExistingOrders() error cannot be nil")
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderStatus(265563260)
	if err != nil && mockTests {
		t.Error("Test Failed - GetOrderStatus() error", err)
	} else if err == nil && !mockTests {
		t.Error("Test Failed - GetOrderStatus() error cannot be nil")
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrders()
	if err != nil && mockTests {
		t.Error("Test Failed - GetOrders() error", err)
	} else if err == nil && !mockTests {
		t.Error("Test Failed - GetOrders() error cannot be nil")
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetTradeHistory(testCurrency, 0)
	if err != nil && mockTests {
		t.Error("Test Failed - GetTradeHistory() error", err)
	} else if err == nil && !mockTests {
		t.Error("Test Failed - GetTradeHistory() error cannot be nil")
	}
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := g.GetTradeVolume()
	if err != nil && mockTests {
		t.Error("Test Failed - GetTradeVolume() error", err)
	} else if err == nil && !mockTests {
		t.Error("Test Failed - GetTradeVolume() error cannot be nil")
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()
	_, err := g.GetBalances()
	if err != nil && mockTests {
		t.Error("Test Failed - GetBalances() error", err)
	} else if err == nil && !mockTests {
		t.Error("Test Failed - GetBalances() error cannot be nil")
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := g.GetCryptoDepositAddress("LOL123", "btc")
	if err == nil {
		t.Error("Test Failed - GetCryptoDepositAddress() error", err)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := g.WithdrawCrypto("LOL123", "btc", 1)
	if err == nil {
		t.Error("Test Failed - WithdrawCrypto() error", err)
	}
}

func TestPostHeartbeat(t *testing.T) {
	t.Parallel()
	_, err := g.PostHeartbeat()
	if err != nil && mockTests {
		t.Error("Test Failed - PostHeartbeat() error", err)
	} else if err == nil && !mockTests {
		t.Error("Test Failed - PostHeartbeat() error cannot be nil")
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.LTC.String(),
			"_"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	g.GetFeeByType(feeBuilder)

	if !areTestAPIKeysSet() {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v",
				exchange.OfflineTradeFee,
				feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v",
				exchange.CryptocurrencyTradeFee,
				feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	if areTestAPIKeysSet() || mockTests {
		// CryptocurrencyTradeFee Basic
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.0035) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
				float64(0.0035),
				resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := g.GetFee(feeBuilder); resp != float64(3500) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
				float64(3500),
				resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
				float64(0.001),
				resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
				float64(0),
				resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText +
		" & " +
		exchange.AutoWithdrawCryptoWithSetupText +
		" & " +
		exchange.WithdrawFiatViaWebsiteOnlyText

	withdrawPermissions := g.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s",
			expectedResult,
			withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
		Currencies: []currency.Pair{
			currency.NewPair(currency.LTC, currency.BTC),
		},
	}

	_, err := g.GetActiveOrders(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not get open orders: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not get open orders: %s", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType:  exchange.AnyOrderType,
		Currencies: []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
	}

	_, err := g.GetOrderHistory(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Errorf("Could not get order history: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case err != nil && mockTests:
		t.Errorf("Could not get order history: %s", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	if g.APIKey != "" && g.APIKey != "Key" &&
		g.APISecret != "" && g.APISecret != "Secret" {
		return true
	}
	return false
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = currency.Pair{
		Delimiter: "_",
		Base:      currency.LTC,
		Quote:     currency.BTC,
	}

	response, err := g.SubmitOrder(p,
		exchange.BuyOrderSide,
		exchange.LimitOrderType,
		1,
		10,
		"1234234")
	switch {
	case areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced):
		t.Errorf("Order failed to be placed: %v", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderCancellation = &exchange.OrderCancellation{
		OrderID: "266029865",
	}

	err := g.CancelOrder(orderCancellation)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case err != nil && mockTests:
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := g.CancelAllOrders(orderCancellation)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := g.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    currency.BTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := g.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil && !mockTests {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
	if areTestAPIKeysSet() && err == nil && mockTests {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}
	_, err := g.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}
	_, err := g.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := g.GetDepositAddress(currency.BTC, "")
	if err == nil {
		t.Error("Test Failed - GetDepositAddress error cannot be nil")
	}
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	t.Parallel()
	g.WebsocketURL = geminiWebsocketSandboxEndpoint

	if !g.Websocket.IsEnabled() &&
		!g.AuthenticatedWebsocketAPISupport ||
		!areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	go g.WsHandleData()
	err := g.WsSecureSubscribe(&dialer, geminiWsOrderEvents)
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case resp := <-g.Websocket.DataHandler:
		if resp.(WsSubscriptionAcknowledgementResponse).Type != "subscription_ack" {
			t.Error("Login failed")
		}
	case <-timer.C:
		t.Error("Expected response")
	}
	timer.Stop()
}
