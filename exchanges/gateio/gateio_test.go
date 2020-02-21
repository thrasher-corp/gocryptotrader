package gateio

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

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var g Gateio
var wsSetupRan bool

func TestMain(m *testing.M) {
	g.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("GateIO load config error", err)
	}
	gConf, err := cfg.GetExchangeConfig("GateIO")
	if err != nil {
		log.Fatal("GateIO Setup() init error")
	}
	gConf.API.AuthenticatedSupport = true
	gConf.API.AuthenticatedWebsocketSupport = true
	gConf.API.Credentials.Key = apiKey
	gConf.API.Credentials.Secret = apiSecret

	err = g.Setup(gConf)
	if err != nil {
		log.Fatal("GateIO setup error", err)
	}

	os.Exit(m.Run())
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := g.GetSymbols()
	if err != nil {
		t.Errorf("Gateio TestGetSymbols: %s", err)
	}
}

func TestGetMarketInfo(t *testing.T) {
	t.Parallel()
	_, err := g.GetMarketInfo()
	if err != nil {
		t.Errorf("Gateio GetMarketInfo: %s", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	_, err := g.SpotNewOrder(SpotNewOrderRequestParams{
		Symbol: "btc_usdt",
		Amount: 1.1,
		Price:  10.1,
		Type:   order.Sell.Lower(),
	})
	if err != nil {
		t.Errorf("Gateio SpotNewOrder: %s", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	_, err := g.CancelExistingOrder(917591554, "btc_usdt")
	if err != nil {
		t.Errorf("Gateio CancelExistingOrder: %s", err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() {
		t.Skip()
	}

	_, err := g.GetBalances()
	if err != nil {
		t.Errorf("Gateio GetBalances: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := g.GetLatestSpotPrice("btc_usdt")
	if err != nil {
		t.Errorf("Gateio GetLatestSpotPrice: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := g.GetTicker("btc_usdt")
	if err != nil {
		t.Errorf("Gateio GetTicker: %s", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := g.GetTickers()
	if err != nil {
		t.Errorf("Gateio GetTicker: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderbook("btc_usdt")
	if err != nil {
		t.Errorf("Gateio GetTicker: %s", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()

	_, err := g.GetSpotKline(KlinesRequestParams{
		Symbol:   "btc_usdt",
		GroupSec: TimeIntervalFiveMinutes, // 5 minutes or less
		HourSize: 1,                       // 1 hour data
	})

	if err != nil {
		t.Errorf("Gateio GetSpotKline: %s", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.USDT.String(), "_"),
		IsMaker:             false,
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	g.GetFeeByType(feeBuilder)
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
	if areTestAPIKeysSet() {
		// CryptocurrencyTradeFee Basic
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Error(err)
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := g.GetFee(feeBuilder); resp != float64(2000) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(2000), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := g.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType,
	}

	_, err := g.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType,
	}

	currPair := currency.NewPair(currency.LTC, currency.BTC)
	currPair.Delimiter = "_"
	getOrdersRequest.Pairs = []currency.Pair{currPair}

	_, err := g.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return g.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.LTC,
			Quote:     currency.BTC,
		},
		Side:     order.Buy,
		Type:     order.Limit,
		Price:    1,
		Amount:   1,
		ClientID: "meowOrder",
	}
	response, err := g.SubmitOrder(orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
	}

	err := g.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
	}

	resp, err := g.CancelAllOrders(orderCancellation)

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
	if apiSecret == "" || apiKey == "" {
		_, err := g.UpdateAccountInfo()
		if err == nil {
			t.Error("GetAccountInfo() Expected error")
		}
	} else {
		_, err := g.UpdateAccountInfo()
		if err != nil {
			t.Error("GetAccountInfo() error", err)
		}
	}
}

func TestModifyOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := g.ModifyOrder(&order.Modify{})
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

	_, err := g.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
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
	_, err := g.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := g.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := g.GetDepositAddress(currency.ETC, "")
		if err != nil {
			t.Error("Test Fail - GetDepositAddress error", err)
		}
	} else {
		_, err := g.GetDepositAddress(currency.ETC, "")
		if err == nil {
			t.Error("Test Fail - GetDepositAddress error cannot be nil")
		}
	}
}
func TestGetOrderInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("no API keys set skipping test")
	}

	_, err := g.GetOrderInfo("917591554")
	if err != nil {
		if err.Error() != "no order found with id 917591554" && err.Error() != "failed to get open orders" {
			t.Fatalf("GetOrderInfo() returned an error skipping test: %v", err)
		}
	}
}

// TestWsGetBalance dials websocket, sends balance request.
func TestWsGetBalance(t *testing.T) {
	if !g.Websocket.IsEnabled() && !g.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	g.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         g.Name,
		URL:                  gateioWebsocketEndpoint,
		Verbose:              g.Verbose,
		RateLimit:            gateioWebsocketRateLimit,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := g.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go g.wsReadData()
	resp, err := g.wsServerSignIn()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Result.Status != "success" {
		t.Fatal("Unsuccessful login")
	}
	_, err = g.wsGetBalance([]string{"EOS", "BTC"})
	if err != nil {
		t.Error(err)
	}
	_, err = g.wsGetBalance([]string{})
	if err != nil {
		t.Error(err)
	}
}

// TestWsGetOrderInfo dials websocket, sends order info request.
func TestWsGetOrderInfo(t *testing.T) {
	if !g.Websocket.IsEnabled() && !g.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	g.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         g.Name,
		URL:                  gateioWebsocketEndpoint,
		Verbose:              g.Verbose,
		RateLimit:            gateioWebsocketRateLimit,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := g.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go g.wsReadData()
	resp, err := g.wsServerSignIn()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Result.Status != "success" {
		t.Fatal("Unsuccessful login")
	}
	_, err = g.wsGetOrderInfo("EOS_USDT", 0, 1000)
	if err != nil {
		t.Error(err)
	}
}

func setupWSTestAuth(t *testing.T) {
	if wsSetupRan {
		return
	}
	if !g.Websocket.IsEnabled() && !g.API.AuthenticatedWebsocketSupport {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	g.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         g.Name,
		URL:                  gateioWebsocketEndpoint,
		Verbose:              g.Verbose,
		RateLimit:            gateioWebsocketRateLimit,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := g.WebsocketConn.Dial(&dialer, http.Header{})

	g.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	g.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	if err != nil {
		t.Fatal(err)
	}
	go g.wsReadData()
	wsSetupRan = true
}

// TestWsUnsubscribe dials websocket, sends an unsubscribe request.
func TestWsUnsubscribe(t *testing.T) {
	setupWSTestAuth(t)
	g.Verbose = true
	err := g.Unsubscribe(wshandler.WebsocketChannelSubscription{
		Channel:  "ticker.subscribe",
		Currency: currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_"),
	})
	if err != nil {
		t.Error(err)
	}
}

// TestWsSubscribe dials websocket, sends a subscribe request.
func TestWsSubscribe(t *testing.T) {
	setupWSTestAuth(t)
	err := g.Subscribe(wshandler.WebsocketChannelSubscription{
		Channel:  "ticker.subscribe",
		Currency: currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_"),
	})
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "ticker.update", 
    "params": 
        [
            "BTC_USDT", 
                {
                    "period": 86400, 
                    "open": "0",
                    "close": "0",
                    "high": "0",
                    "low": "0",
                    "last": "0.2844",
                    "change": "0",
                    "quoteVolume": "0",
                    "baseVolume": "0"
                }
     ],
     "id": null
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrade(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "trades.update",
    "params": 
        [
             "BTC_USDT", 
             [
                 {
                 "id": 7172173,
                 "time": 1523339279.761838,
                 "price": "398.59",
                 "amount": "0.027",
                 "type": "buy"
                 }
             ]
         ],
     "id": null
 }
`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsDepth(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "depth.update", 
    "params": [
        true, 
        {
            "asks": [
                [                    
                    "8000.00",
                    "9.6250"
                ]
            ],
            "bids": [                
                [                    
                    "8000.00",
                    "9.6250"
                ]                
            ]
         }, 
         "BTC_USDT"
    ],
    "id": null
 }`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsKLine(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "kline.update",
    "params":
        [
            [
                1492358400,
                "7000.00",
                "8000.0",
                "8100.00",
                "6800.00",
                "1000.00",
                "123456.00",
                "BTC_USDT"
            ]
        ],
    "id": null
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
  "method": "order.update",
  "params": [
    3,
    {
      "id": 34628963,
      "market": "BTC_USDT",
      "orderType": 1,
      "type": 2,
      "user": 602123,
      "ctime": 1523013969.6271579,
      "mtime": 1523013969.6271579,
      "price": "0.1",
      "amount": "1000",
      "left": "1000",
      "filledAmount": "0",
      "filledTotal": "0",
      "dealFee": "0"
    }
  ],
  "id": null
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsBalanceUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "balance.update", 
    "params": [{"EOS": {"available": "96.765323611874", "freeze": "11"}}],
    "id": 1234
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}
