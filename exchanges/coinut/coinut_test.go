package coinut

import (
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var c COINUT
var wsSetupRan bool

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	clientID                = ""
	canManipulateRealOrders = false
)

func TestMain(m *testing.M) {
	c.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Coinut load config error", err)
	}
	bConfig, err := cfg.GetExchangeConfig("COINUT")
	if err != nil {
		log.Fatal("Coinut Setup() init error")
	}
	bConfig.API.AuthenticatedSupport = true
	bConfig.API.AuthenticatedWebsocketSupport = true
	bConfig.API.Credentials.Key = apiKey
	bConfig.API.Credentials.ClientID = clientID
	err = c.Setup(bConfig)
	if err != nil {
		log.Fatal("Coinut setup error", err)
	}

	c.SeedInstruments()

	os.Exit(m.Run())
}

func setupWSTestAuth(t *testing.T) {
	if wsSetupRan {
		return
	}

	if !c.Websocket.IsEnabled() && !c.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	if areTestAPIKeysSet() {
		c.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	c.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         c.Name,
		URL:                  coinutWebsocketURL,
		Verbose:              c.Verbose,
		RateLimit:            coinutWebsocketRateLimit,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}

	var dialer websocket.Dialer
	err := c.WebsocketConn.Dial(&dialer, http.Header{})
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
	wsSetupRan = true
	_, err = c.WsGetInstruments()
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstruments(t *testing.T) {
	_, err := c.GetInstruments()
	if err != nil {
		t.Error("GetInstruments() error", err)
	}
}

func TestSeedInstruments(t *testing.T) {
	err := c.SeedInstruments()
	if err != nil {
		// No point checking the next condition
		t.Fatal(err)
	}

	if len(c.instrumentMap.GetInstrumentIDs()) == 0 {
		t.Error("instrument map hasn't been seeded")
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
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if resp, err := c.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Error(err)
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.0010), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := c.GetFee(feeBuilder); resp != float64(1000) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(1000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.EUR
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := c.GetFee(feeBuilder); resp != float64(10) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(10), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.SGD
	if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := c.GetFee(feeBuilder); resp != float64(10) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(10), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.CAD
	if resp, err := c.GetFee(feeBuilder); resp != float64(2) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(2), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.SGD
	if resp, err := c.GetFee(feeBuilder); resp != float64(10) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(10), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.CAD
	if resp, err := c.GetFee(feeBuilder); resp != float64(2) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(2), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.WithdrawCryptoViaWebsiteOnlyText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := c.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}
	_, err := c.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	}
}

func TestGetOrderHistoryWrapper(t *testing.T) {
	setupWSTestAuth(t)
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
		Currencies: []currency.Pair{currency.NewPair(currency.BTC,
			currency.USD)},
	}

	_, err := c.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return c.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.USD,
		},
		OrderSide: order.Buy,
		OrderType: order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "123",
	}
	response, err := c.SubmitOrder(orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	currencyPair := currency.NewPair(currency.BTC, currency.USD)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
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

	resp, err := c.CancelAllOrders(orderCancellation)

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

func TestGetAccountInfo(t *testing.T) {
	if apiKey != "" || clientID != "" {
		_, err := c.UpdateAccountInfo()
		if err != nil {
			t.Error("GetAccountInfo() error", err)
		}
	} else {
		_, err := c.UpdateAccountInfo()
		if err == nil {
			t.Error("GetAccountInfo() Expected error")
		}
	}
}

func TestModifyOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := c.ModifyOrder(&order.Modify{})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	withdrawCryptoRequest := withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: &withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
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
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := c.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := c.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := c.GetDepositAddress(currency.BTC, "")
	if err == nil {
		t.Error("GetDepositAddress() function unsupported cannot be nil")
	}
}

// TestWsAuthGetAccountBalance dials websocket, retrieves account balance
func TestWsAuthGetAccountBalance(t *testing.T) {
	setupWSTestAuth(t)
	_, err := c.wsGetAccountBalance()
	if err != nil {
		t.Error(err)
	}
}

// TestWsAuthSubmitOrder dials websocket, submit order
func TestWsAuthSubmitOrder(t *testing.T) {
	setupWSTestAuth(t)
	if !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	ord := WsSubmitOrderParameters{
		Amount:   1,
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
		Price:    1,
		Side:     order.Buy,
	}
	_, err := c.wsSubmitOrder(&ord)
	if err != nil {
		t.Error(err)
	}
}

// TestWsAuthCancelOrders dials websocket, submit orders
func TestWsAuthSubmitOrders(t *testing.T) {
	setupWSTestAuth(t)
	if !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	order1 := WsSubmitOrderParameters{
		Amount:   1,
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
		Price:    1,
		Side:     order.Buy,
	}
	order2 := WsSubmitOrderParameters{
		Amount:   3,
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  2,
		Price:    2,
		Side:     order.Buy,
	}
	_, err := c.wsSubmitOrders([]WsSubmitOrderParameters{order1, order2})
	if err != nil {
		t.Error(err)
	}
}

// TestWsAuthCancelOrders dials websocket, cancels orders
// doesn't care about if the order cancellations fail
func TestWsAuthCancelOrders(t *testing.T) {
	setupWSTestAuth(t)
	if !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	ord := WsCancelOrderParameters{
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
	}
	order2 := WsCancelOrderParameters{
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  2,
	}
	resp, err := c.wsCancelOrders([]WsCancelOrderParameters{ord, order2})
	if err != nil {
		t.Error(err)
	}
	if resp.Status[0] != "OK" {
		t.Error("Order failed to cancel")
	}
}

// TestWsAuthCancelOrders dials websocket, cancels orders
// Checks that the wrapper oversight works
func TestWsAuthCancelOrdersWrapper(t *testing.T) {
	setupWSTestAuth(t)
	if !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	orderDetails := order.Cancel{
		CurrencyPair: currency.NewPair(currency.LTC, currency.BTC),
	}
	_, err := c.CancelAllOrders(&orderDetails)
	if err != nil {
		t.Error(err)
	}
}

// TestWsAuthCancelOrder dials websocket, cancels order
func TestWsAuthCancelOrder(t *testing.T) {
	setupWSTestAuth(t)
	if !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	ord := &WsCancelOrderParameters{
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
	}
	resp, err := c.wsCancelOrder(ord)
	if err != nil {
		t.Error(err)
	}
	if len(resp.Status) >= 1 && resp.Status[0] != "OK" {
		t.Errorf("Failed to cancel order")
	}
}

// TestWsAuthGetOpenOrders dials websocket, retrieves open orders
func TestWsAuthGetOpenOrders(t *testing.T) {
	setupWSTestAuth(t)
	_, err := c.wsGetOpenOrders(currency.NewPair(currency.LTC, currency.BTC).String())
	if err != nil {
		t.Error(err)
	}
}

func TestCurrencyMapIsLoaded(t *testing.T) {
	t.Parallel()
	var i instrumentMap
	if l := i.IsLoaded(); l {
		t.Error("unexpected result")
	}

	i.Seed("BTCUSD", 1337)
	if l := i.IsLoaded(); !l {
		t.Error("unexpected result")
	}
}

func TestCurrencyMapSeed(t *testing.T) {
	t.Parallel()
	var i instrumentMap
	// Test non-seeded lookups
	if id := i.LookupInstrument(1234); id != "" {
		t.Error("unexpected result")
	}
	if id := i.LookupID("BLAH"); id != 0 {
		t.Error("unexpected result")
	}

	// Test seeded lookups
	i.Seed("BTCUSD", 1337)
	if id := i.LookupID("BTCUSD"); id != 1337 {
		t.Error("unexpected result")
	}
	if id := i.LookupInstrument(1337); id != "BTCUSD" {
		t.Error("unexpected result")
	}

	// Test invalid lookups
	if id := i.LookupInstrument(1234); id != "" {
		t.Error("unexpected result")
	}
	if id := i.LookupID("BLAH"); id != 0 {
		t.Error("unexpected result")
	}

	// Test seeding existing item
	i.Seed("BTCUSD", 1234)
	if id := i.LookupID("BTCUSD"); id != 1337 {
		t.Error("unexpected result")
	}
	if id := i.LookupInstrument(1337); id != "BTCUSD" {
		t.Error("unexpected result")
	}
}

func TestCurrencyMapInstrumentIDs(t *testing.T) {
	t.Parallel()

	var i instrumentMap
	if r := i.GetInstrumentIDs(); len(r) > 0 {
		t.Error("non initialised instrument map shouldn't return any ids")
	}

	// Seed the instrument map
	i.Seed("BTCUSD", 1234)
	i.Seed("LTCUSD", 1337)

	f := func(ids []int64, target int64) bool {
		for x := range ids {
			if ids[x] == target {
				return true
			}
		}
		return false
	}

	// Test 2 valid instruments and one invalid
	ids := i.GetInstrumentIDs()
	if r := f(ids, 1234); !r {
		t.Error("unexpected result")
	}
	if r := f(ids, 1337); !r {
		t.Error("unexpected result")
	}
	if r := f(ids, 4321); r {
		t.Error("unexpected result")
	}
}

func TestGetNonce(t *testing.T) {
	result := getNonce()
	for x := 0; x < 100000; x++ {
		if result <= 0 || result > coinutMaxNonce {
			t.Fatal("invalid nonce value")
		}
	}
}
