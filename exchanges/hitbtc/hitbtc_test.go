package hitbtc

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var h HitBTC
var wsSetupRan bool

// Please supply your own APIKEYS here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

func TestMain(m *testing.M) {
	h.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("HitBTC load config error", err)
	}
	hitbtcConfig, err := cfg.GetExchangeConfig("HitBTC")
	if err != nil {
		log.Fatal("HitBTC Setup() init error")
	}
	hitbtcConfig.API.AuthenticatedSupport = true
	hitbtcConfig.API.AuthenticatedWebsocketSupport = true
	hitbtcConfig.API.Credentials.Key = apiKey
	hitbtcConfig.API.Credentials.Secret = apiSecret

	err = h.Setup(hitbtcConfig)
	if err != nil {
		log.Fatal("HitBTC setup error", err)
	}

	os.Exit(m.Run())
}

func TestGetOrderbook(t *testing.T) {
	_, err := h.GetOrderbook("BTCUSD", 50)
	if err != nil {
		t.Error("Test faild - HitBTC GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := h.GetTrades("BTCUSD", "", "", "", "", "", "")
	if err != nil {
		t.Error("Test faild - HitBTC GetTradeHistory() error", err)
	}
}

func TestGetChartCandles(t *testing.T) {
	_, err := h.GetCandles("BTCUSD", "", "")
	if err != nil {
		t.Error("Test faild - HitBTC GetChartData() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := h.GetCurrencies()
	if err != nil {
		t.Error("Test faild - HitBTC GetCurrencies() error", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPair(currency.ETH, currency.BTC),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	h.GetFeeByType(feeBuilder)
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

func TestUpdateTicker(t *testing.T) {
	h.CurrencyPairs.StorePairs(asset.Spot, currency.NewPairsFromStrings([]string{"BTC-USD", "XRP-USD"}), true)
	_, err := h.UpdateTicker(currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = h.FetchTicker(currency.NewPair(currency.XRP, currency.USD), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllTickers(t *testing.T) {
	_, err := h.GetTickers()
	if err != nil {
		t.Error(err)
	}
}

func TestGetSingularTicker(t *testing.T) {
	_, err := h.GetTicker("BTCUSD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	if areTestAPIKeysSet() {
		// CryptocurrencyTradeFee Basic
		if resp, err := h.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Error(err)
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := h.GetFee(feeBuilder); resp != float64(2000) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(2000), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := h.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}

		// CryptocurrencyWithdrawalFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if resp, err := h.GetFee(feeBuilder); resp != float64(0.042800) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.042800), resp)
			t.Error(err)
		}

		// CryptocurrencyWithdrawalFee Invalid currency
		feeBuilder = setFeeBuilder()
		feeBuilder.Pair.Base = currency.NewCode("hello")
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err == nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	feeBuilder.Pair.Base = currency.BTC
	feeBuilder.Pair.Quote = currency.LTC
	if resp, err := h.GetFee(feeBuilder); resp != float64(0.0006) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.0006), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := h.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType:  order.AnyType,
		Currencies: []currency.Pair{currency.NewPair(currency.ETH, currency.BTC)},
	}

	_, err := h.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType:  order.AnyType,
		Currencies: []currency.Pair{currency.NewPair(currency.ETH, currency.BTC)},
	}

	_, err := h.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return h.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.DGD,
			Quote: currency.BTC,
		},
		OrderSide: order.Buy,
		OrderType: order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
	}
	response, err := h.SubmitOrder(orderSubmission)
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

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := h.CancelOrder(orderCancellation)
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

	resp, err := h.CancelAllOrders(orderCancellation)

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
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := h.ModifyOrder(&order.Modify{})
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

	_, err := h.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := h.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := h.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := h.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("GetDepositAddress() error", err)
		}
	} else {
		_, err := h.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("GetDepositAddress() error cannot be nil")
		}
	}
}
func setupWsAuth(t *testing.T) {
	if wsSetupRan {
		return
	}
	if !h.Websocket.IsEnabled() && !h.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	h.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	h.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	h.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         h.Name,
		URL:                  hitbtcWebsocketAddress,
		Verbose:              h.Verbose,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := h.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go h.WsHandleData()
	h.wsLogin()
	timer := time.NewTimer(time.Second)
	select {
	case loginError := <-h.Websocket.DataHandler:
		t.Fatal(loginError)
	case <-timer.C:
	}
	timer.Stop()
	wsSetupRan = true
}

// TestWsCancelOrder dials websocket, sends cancel request.
func TestWsCancelOrder(t *testing.T) {
	setupWsAuth(t)
	if !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders false, skipping test")
	}
	_, err := h.wsCancelOrder("ImNotARealOrderID")
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsPlaceOrder dials websocket, sends order submission.
func TestWsPlaceOrder(t *testing.T) {
	setupWsAuth(t)
	if !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders false, skipping test")
	}
	_, err := h.wsPlaceOrder(currency.NewPair(currency.LTC, currency.BTC),
		order.Buy.String(),
		1,
		1)
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsReplaceOrder dials websocket, sends replace order request.
func TestWsReplaceOrder(t *testing.T) {
	setupWsAuth(t)
	if !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders false, skipping test")
	}
	_, err := h.wsReplaceOrder("ImNotARealOrderID", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetActiveOrders dials websocket, sends get active orders request.
func TestWsGetActiveOrders(t *testing.T) {
	setupWsAuth(t)
	_, err := h.wsGetActiveOrders()
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetTradingBalance dials websocket, sends get trading balance request.
func TestWsGetTradingBalance(t *testing.T) {
	setupWsAuth(t)
	_, err := h.wsGetTradingBalance()
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetTradingBalance dials websocket, sends get trading balance request.
func TestWsGetTrades(t *testing.T) {
	setupWsAuth(t)
	_, err := h.wsGetTrades(currency.NewPair(currency.ETH, currency.BTC), 1000, "ASC", "id")
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetTradingBalance dials websocket, sends get trading balance request.
func TestWsGetSymbols(t *testing.T) {
	setupWsAuth(t)
	_, err := h.wsGetSymbols(currency.NewPair(currency.ETH, currency.BTC))
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetTradingBalance dials websocket, sends get trading balance request.
func TestSsGetCurrencies(t *testing.T) {
	setupWsAuth(t)
	_, err := h.wsGetCurrencies(currency.BTC)
	if err != nil {
		t.Fatal(err)
	}
}
