package coinut

import (
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/idoall/gocryptotrader/common"
	"github.com/idoall/gocryptotrader/config"
	"github.com/idoall/gocryptotrader/currency"
	exchange "github.com/idoall/gocryptotrader/exchanges"
	"github.com/idoall/gocryptotrader/exchanges/sharedtestvalues"
)

var c COINUT
var wsSetupRan bool

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	clientID                = ""
	canManipulateRealOrders = false
)

func TestSetDefaults(t *testing.T) {
	c.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("COINUT")
	if err != nil {
		t.Error("Test Failed - Coinut Setup() init error")
	}
	bConfig.AuthenticatedAPISupport = true
	bConfig.AuthenticatedWebsocketAPISupport = true
	bConfig.APIKey = apiKey
	c.Setup(&bConfig)
	c.ClientID = clientID

	if !c.IsEnabled() ||
		c.RESTPollingDelay != time.Duration(10) ||
		c.Websocket.IsEnabled() || len(c.BaseCurrencies) < 1 ||
		len(c.AvailablePairs) < 1 || len(c.EnabledPairs) < 1 {
		t.Error("Test Failed - Coinut Setup values not set correctly")
	}
}

func setupWSTestAuth(t *testing.T) {
	if wsSetupRan {
		return
	}
	c.SetDefaults()
	TestSetup(t)
	if !c.Websocket.IsEnabled() && !c.AuthenticatedWebsocketAPISupport || !areTestAPIKeysSet() {
		t.Skip(exchange.WebsocketNotEnabled)
	}
	var err error
	var dialer websocket.Dialer
	c.WebsocketConn, _, err = dialer.Dial(c.Websocket.GetWebsocketURL(),
		http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	c.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	c.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	go c.WsHandleData()
	err = c.wsAuthenticate()
	if err != nil {
		t.Error(err)
	}

	timer := time.NewTimer(5 * time.Second)
	select {
	case resp := <-c.Websocket.DataHandler:
		if resp.(WsLoginResponse).Username != clientID {
			t.Fatal("Unsuccessful login")
		}
	case <-timer.C:
		t.Fatal("Expected response")
	}
	timer.Stop()
	time.Sleep(2 * time.Second)
	instrumentListByString = make(map[string]int64)
	instrumentListByString[currency.NewPair(currency.LTC, currency.BTC).String()] = 1
	wsSetupRan = true
}

func TestGetInstruments(t *testing.T) {
	_, err := c.GetInstruments()
	if err != nil {
		t.Error("Test failed - GetInstruments() error", err)
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
	c.GetFeeByType(feeBuilder)
	if apiKey == "" {
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
	c.SetDefaults()
	TestSetup(t)
	t.Parallel()

	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := c.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0010), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := c.GetFee(feeBuilder); resp != float64(1000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(1000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.EUR
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := c.GetFee(feeBuilder); resp != float64(10) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(10), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.SGD
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := c.GetFee(feeBuilder); resp != float64(10) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(10), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.CAD
	if resp, err := c.GetFee(feeBuilder); resp != float64(2) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.SGD
	if resp, err := c.GetFee(feeBuilder); resp != float64(10) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(10), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.CAD
	if resp, err := c.GetFee(feeBuilder); resp != float64(2) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {

	c.SetDefaults()
	expectedResult := exchange.WithdrawCryptoViaWebsiteOnlyText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText

	withdrawPermissions := c.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := c.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
		Currencies: []currency.Pair{currency.NewPair(currency.BTC,
			currency.LTC)},
	}

	_, err := c.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	if c.APIKey != "" && c.APIKey != "Key" {
		return true
	}
	return false
}

func TestSubmitOrder(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = currency.Pair{
		Delimiter: "",
		Base:      currency.BTC,
		Quote:     currency.USD,
	}
	response, err := c.SubmitOrder(p, exchange.BuyOrderSide, exchange.LimitOrderType, 1, 10, "1234234")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {

	c.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := c.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {

	c.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := c.CancelAllOrders(orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if apiKey != "" || clientID != "" {
		_, err := c.GetAccountInfo()
		if err != nil {
			t.Error("Test Failed - GetAccountInfo() error", err)
		}
	} else {
		_, err := c.GetAccountInfo()
		if err == nil {
			t.Error("Test Failed - GetAccountInfo() error")
		}
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := c.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    currency.LTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := c.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected 'Not supported', received %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := c.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := c.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := c.GetDepositAddress(currency.BTC, "")
	if err == nil {
		t.Error("Test Failed - GetDepositAddress() function unsupported cannot be nil")
	}
}

// TestWsAuthGetAccountBalance dials websocket, sends login request.
func TestWsAuthGetAccountBalance(t *testing.T) {
	setupWSTestAuth(t)
	err := c.wsGetAccountBalance()
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseExtendedTimeout)
	select {
	case resp := <-c.Websocket.DataHandler:
		if resp.(WsUserBalanceResponse).Status[0] != "OK" {
			t.Error("Expected successful response")
		}
	case <-timer.C:
		t.Error("Expected response")
	}
	timer.Stop()
}

// TestWsAuthSubmitOrders dials websocket, sends login request.
func TestWsAuthSubmitOrders(t *testing.T) {
	setupWSTestAuth(t)
	order := WsSubmitOrderParameters{
		Amount:   1,
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
		Price:    1,
		Side:     exchange.BuyOrderSide,
	}
	err := c.wsSubmitOrders([]WsSubmitOrderParameters{order, order})
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseExtendedTimeout)
	select {
	case <-c.Websocket.DataHandler:
	case <-timer.C:
		t.Error("Expected response")
	}
	timer.Stop()
}

// TestWsAuthCancelOrders dials websocket, sends login request.
func TestWsAuthCancelOrders(t *testing.T) {
	setupWSTestAuth(t)
	order := WsCancelOrderParameters{
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
	}
	err := c.wsCancelOrders([]WsCancelOrderParameters{order, order})
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseExtendedTimeout)
	select {
	case <-c.Websocket.DataHandler:
	case <-timer.C:
		t.Error("Expected response")
	}
	timer.Stop()
}

// TestWsAuthCancelOrder dials websocket, sends login request.
func TestWsAuthCancelOrder(t *testing.T) {
	setupWSTestAuth(t)
	order := WsCancelOrderParameters{
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
	}
	err := c.wsCancelOrder(order)
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseExtendedTimeout)
	select {
	case <-c.Websocket.DataHandler:
	case <-timer.C:
		t.Error("Expected response")
	}
	timer.Stop()
}

// TestWsAuthGetOpenOrders dials websocket, sends login request.
func TestWsAuthGetOpenOrders(t *testing.T) {
	setupWSTestAuth(t)
	err := c.wsGetOpenOrders(currency.NewPair(currency.LTC, currency.BTC))
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseExtendedTimeout)
	select {
	case <-c.Websocket.DataHandler:
	case <-timer.C:
		t.Error("Expected response")
	}
	timer.Stop()
}
