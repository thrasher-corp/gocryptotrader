package kraken

import (
	"log"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
)

var k Kraken
var wsSetupRan bool

// Please add your own APIkeys to do correct due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	clientID                = ""
	canManipulateRealOrders = false
)

// TestSetDefaults setup func
func TestSetDefaults(t *testing.T) {
	k.SetDefaults()
}

// TestSetup setup func
func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Kraken load config error", err)
	}
	krakenConfig, err := cfg.GetExchangeConfig("Kraken")
	if err != nil {
		t.Error("kraken Setup() init error", err)
	}
	krakenConfig.API.AuthenticatedSupport = true
	krakenConfig.API.Credentials.Key = apiKey
	krakenConfig.API.Credentials.Secret = apiSecret
	krakenConfig.API.Credentials.ClientID = clientID
	krakenConfig.API.Endpoints.WebsocketURL = k.API.Endpoints.WebsocketURL
	subscribeToDefaultChannels = false

	err = k.Setup(krakenConfig)
	if err != nil {
		t.Fatal("Kraken setup error", err)
	}
}

// TestGetServerTime API endpoint test
func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := k.GetServerTime()
	if err != nil {
		t.Error("GetServerTime() error", err)
	}
}

// TestGetAssets API endpoint test
func TestGetAssets(t *testing.T) {
	t.Parallel()
	_, err := k.GetAssets()
	if err != nil {
		t.Error("GetAssets() error", err)
	}
}

// TestGetAssetPairs API endpoint test
func TestGetAssetPairs(t *testing.T) {
	t.Parallel()
	_, err := k.GetAssetPairs()
	if err != nil {
		t.Error("GetAssetPairs() error", err)
	}
}

// TestGetTicker API endpoint test
func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := k.GetTicker("BCHEUR")
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

// TestGetTickers API endpoint test
func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := k.GetTickers("LTCUSD,ETCUSD")
	if err != nil {
		t.Error("GetTickers() error", err)
	}
}

// TestGetOHLC API endpoint test
func TestGetOHLC(t *testing.T) {
	t.Parallel()
	_, err := k.GetOHLC("BCHEUR")
	if err != nil {
		t.Error("GetOHLC() error", err)
	}
}

// TestGetDepth API endpoint test
func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := k.GetDepth("BCHEUR")
	if err != nil {
		t.Error("GetDepth() error", err)
	}
}

// TestGetTrades API endpoint test
func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := k.GetTrades("BCHEUR")
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

// TestGetSpread API endpoint test
func TestGetSpread(t *testing.T) {
	t.Parallel()
	_, err := k.GetSpread("BCHEUR")
	if err != nil {
		t.Error("GetSpread() error", err)
	}
}

// TestGetBalance API endpoint test
func TestGetBalance(t *testing.T) {
	t.Parallel()
	_, err := k.GetBalance()
	if err == nil {
		t.Error("GetBalance() error", err)
	}
}

// TestGetTradeBalance API endpoint test
func TestGetTradeBalance(t *testing.T) {
	t.Parallel()
	args := TradeBalanceOptions{Asset: "ZEUR"}
	_, err := k.GetTradeBalance(args)
	if err == nil {
		t.Error("GetTradeBalance() error", err)
	}
}

// TestGetOpenOrders API endpoint test
func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	args := OrderInfoOptions{Trades: true}
	_, err := k.GetOpenOrders(args)
	if err == nil {
		t.Error("GetOpenOrders() error", err)
	}
}

// TestGetClosedOrders API endpoint test
func TestGetClosedOrders(t *testing.T) {
	t.Parallel()
	args := GetClosedOrdersOptions{Trades: true, Start: "OE4KV4-4FVQ5-V7XGPU"}
	_, err := k.GetClosedOrders(args)
	if err == nil {
		t.Error("GetClosedOrders() error", err)
	}
}

// TestQueryOrdersInfo API endpoint test
func TestQueryOrdersInfo(t *testing.T) {
	t.Parallel()
	args := OrderInfoOptions{Trades: true}
	_, err := k.QueryOrdersInfo(args, "OR6ZFV-AA6TT-CKFFIW", "OAMUAJ-HLVKG-D3QJ5F")
	if err == nil {
		t.Error("QueryOrdersInfo() error", err)
	}
}

// TestGetTradesHistory API endpoint test
func TestGetTradesHistory(t *testing.T) {
	t.Parallel()
	args := GetTradesHistoryOptions{Trades: true, Start: "TMZEDR-VBJN2-NGY6DX", End: "TVRXG2-R62VE-RWP3UW"}
	_, err := k.GetTradesHistory(args)
	if err == nil {
		t.Error("GetTradesHistory() error", err)
	}
}

// TestQueryTrades API endpoint test
func TestQueryTrades(t *testing.T) {
	t.Parallel()
	_, err := k.QueryTrades(true, "TMZEDR-VBJN2-NGY6DX", "TFLWIB-KTT7L-4TWR3L", "TDVRAH-2H6OS-SLSXRX")
	if err == nil {
		t.Error("QueryTrades() error", err)
	}
}

// TestOpenPositions API endpoint test
func TestOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := k.OpenPositions(false)
	if err == nil {
		t.Error("OpenPositions() error", err)
	}
}

// TestGetLedgers API endpoint test
func TestGetLedgers(t *testing.T) {
	t.Parallel()
	args := GetLedgersOptions{Start: "LRUHXI-IWECY-K4JYGO", End: "L5NIY7-JZQJD-3J4M2V", Ofs: 15}
	_, err := k.GetLedgers(args)
	if err == nil {
		t.Error("GetLedgers() error", err)
	}
}

// TestQueryLedgers API endpoint test
func TestQueryLedgers(t *testing.T) {
	t.Parallel()
	_, err := k.QueryLedgers("LVTSFS-NHZVM-EXNZ5M")
	if err == nil {
		t.Error("QueryLedgers() error", err)
	}
}

// TestGetTradeVolume API endpoint test
func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := k.GetTradeVolume(true, "OAVY7T-MV5VK-KHDF5X")
	if err == nil {
		t.Error("GetTradeVolume() error", err)
	}
}

// TestAddOrder API endpoint test
func TestAddOrder(t *testing.T) {
	t.Parallel()
	args := AddOrderOptions{OrderFlags: "fcib"}
	_, err := k.AddOrder("XXBTZUSD",
		exchange.SellOrderSide.ToLower().ToString(), exchange.LimitOrderType.ToLower().ToString(),
		0.00000001, 0, 0, 0, &args)
	if err == nil {
		t.Error("AddOrder() error", err)
	}
}

// TestCancelExistingOrder API endpoint test
func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := k.CancelExistingOrder("OAVY7T-MV5VK-KHDF5X")
	if err == nil {
		t.Error("CancelExistingOrder() error", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPair(currency.XXBT, currency.ZUSD),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFee logic test

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	k.GetFeeByType(feeBuilder)
	if apiKey == "" || apiSecret == "" {
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
	k.SetDefaults()
	TestSetup(t)
	var feeBuilder = setFeeBuilder()

	if areTestAPIKeysSet() {
		// CryptocurrencyTradeFee Basic
		if resp, err := k.GetFee(feeBuilder); resp != float64(0.0026) || err != nil {
			t.Error(err)
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.0026), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := k.GetFee(feeBuilder); resp != float64(2600) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(2600), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := k.GetFee(feeBuilder); resp != float64(0.0016) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.0016), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := k.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}

		// InternationalBankDepositFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.InternationalBankDepositFee
		if resp, err := k.GetFee(feeBuilder); resp != float64(5) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(5), resp)
			t.Error(err)
		}
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	feeBuilder.Pair.Base = currency.XXBT
	if resp, err := k.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(5), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := k.GetFee(feeBuilder); resp != float64(0.0005) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.0005), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := k.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := k.GetFee(feeBuilder); resp != float64(5) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(5), resp)
		t.Error(err)
	}
}

// TestFormatWithdrawPermissions logic test
func TestFormatWithdrawPermissions(t *testing.T) {
	k.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.WithdrawCryptoWith2FAText + " & " + exchange.AutoWithdrawFiatWithSetupText + " & " + exchange.WithdrawFiatWith2FAText

	withdrawPermissions := k.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

// TestGetActiveOrders wrapper test
func TestGetActiveOrders(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := k.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// TestGetOrderHistory wrapper test
func TestGetOrderHistory(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := k.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return k.ValidateAPICredentials()
}

// TestSubmitOrder wrapper test
func TestSubmitOrder(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &exchange.OrderSubmission{
		Pair: currency.Pair{
			Base:  currency.XBT,
			Quote: currency.USD,
		},
		OrderSide: exchange.BuyOrderSide,
		OrderType: exchange.LimitOrderType,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
	}
	response, err := k.SubmitOrder(orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// TestCancelExchangeOrder wrapper test
func TestCancelExchangeOrder(t *testing.T) {
	k.SetDefaults()
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

	err := k.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

// TestCancelAllExchangeOrders wrapper test
func TestCancelAllExchangeOrders(t *testing.T) {
	k.SetDefaults()
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

	resp, err := k.CancelAllOrders(orderCancellation)

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

// TestGetAccountInfo wrapper test
func TestGetAccountInfo(t *testing.T) {
	if apiKey != "" || apiSecret != "" || clientID != "" {
		_, err := k.GetAccountInfo()
		if err != nil {
			t.Error("GetAccountInfo() error", err)
		}
	} else {
		_, err := k.GetAccountInfo()
		if err == nil {
			t.Error("GetAccountInfo() error")
		}
	}
}

// TestModifyOrder wrapper test
func TestModifyOrder(t *testing.T) {
	_, err := k.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("ModifyOrder() error")
	}
}

// TestWithdraw wrapper test
func TestWithdraw(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	withdrawCryptoRequest := exchange.CryptoWithdrawRequest{
		GenericWithdrawRequestInfo: exchange.GenericWithdrawRequestInfo{
			Amount:        -1,
			Currency:      currency.XXBT,
			Description:   "WITHDRAW IT ALL",
			TradePassword: "Key",
		},
		Address: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := k.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

// TestWithdrawFiat wrapper test
func TestWithdrawFiat(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.FiatWithdrawRequest{
		GenericWithdrawRequestInfo: exchange.GenericWithdrawRequestInfo{
			Amount:        -1,
			Currency:      currency.EUR,
			Description:   "WITHDRAW IT ALL",
			TradePassword: "someBank",
		},
	}

	_, err := k.WithdrawFiatFunds(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

// TestWithdrawInternationalBank wrapper test
func TestWithdrawInternationalBank(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.FiatWithdrawRequest{
		GenericWithdrawRequestInfo: exchange.GenericWithdrawRequestInfo{
			Amount:        -1,
			Currency:      currency.EUR,
			Description:   "WITHDRAW IT ALL",
			TradePassword: "someBank",
		},
	}

	_, err := k.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

// TestGetDepositAddress wrapper test
func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := k.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("GetDepositAddress() error", err)
		}
	} else {
		_, err := k.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("GetDepositAddress() error can not be nil")
		}
	}
}

// TestWithdrawStatus wrapper test
func TestWithdrawStatus(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() {
		_, err := k.WithdrawStatus(currency.BTC, "")
		if err != nil {
			t.Error("WithdrawStatus() error", err)
		}
	} else {
		_, err := k.WithdrawStatus(currency.BTC, "")
		if err == nil {
			t.Error("GetDepositAddress() error can not be nil", err)
		}
	}
}

// TestWithdrawCancel wrapper test
func TestWithdrawCancel(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)
	_, err := k.WithdrawCancel(currency.BTC, "")
	if areTestAPIKeysSet() && err == nil {
		t.Error("WithdrawCancel() error cannot be nil")
	} else if !areTestAPIKeysSet() && err == nil {
		t.Errorf("WithdrawCancel() error - expecting an error when no keys are set but received nil")
	}
}

// ---------------------------- Websocket tests -----------------------------------------

func setupWsTests(t *testing.T) {
	if wsSetupRan {
		return
	}
	TestSetDefaults(t)
	TestSetup(t)
	if !k.Websocket.IsEnabled() && !k.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	k.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	k.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	k.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         k.Name,
		URL:                  krakenWSURL,
		Verbose:              k.Verbose,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := k.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go k.WsHandleData()
	wsSetupRan = true
}

// TestWebsocketSubscribe tests returning a message with an id
func TestWebsocketSubscribe(t *testing.T) {
	setupWsTests(t)
	err := k.Subscribe(wshandler.WebsocketChannelSubscription{
		Channel:  defaultSubscribedChannels[0],
		Currency: currency.NewPairWithDelimiter("XBT", "USD", "/"),
	})
	if err != nil {
		t.Error(err)
	}
}
