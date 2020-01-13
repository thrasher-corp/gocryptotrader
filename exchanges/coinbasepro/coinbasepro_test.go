package coinbasepro

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/withdraw"
)

var c CoinbasePro

// Please supply your APIKeys here for better testing
const (
	apiKey                  = ""
	apiSecret               = ""
	clientID                = "" // passphrase you made at API CREATION
	canManipulateRealOrders = false
	testPair                = "BTC-USD"
)

func TestMain(m *testing.M) {
	c.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("coinbasepro load config error", err)
	}
	gdxConfig, err := cfg.GetExchangeConfig("CoinbasePro")
	if err != nil {
		log.Fatal("coinbasepro Setup() init error")
	}
	gdxConfig.API.Credentials.Key = apiKey
	gdxConfig.API.Credentials.Secret = apiSecret
	gdxConfig.API.Credentials.ClientID = clientID
	gdxConfig.API.AuthenticatedSupport = true
	gdxConfig.API.AuthenticatedWebsocketSupport = true
	err = c.Setup(gdxConfig)
	if err != nil {
		log.Fatal("CoinbasePro setup error", err)
	}

	os.Exit(m.Run())
}

func TestGetProducts(t *testing.T) {
	_, err := c.GetProducts()
	if err != nil {
		t.Errorf("Coinbase, GetProducts() Error: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	_, err := c.GetTicker(testPair)
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := c.GetTrades(testPair)
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetHistoricRatesApiCheck(t *testing.T) {
	e := expectedCandles(5, 300, 60)
	if e != nil {
		t.Error(e)
	}
	e = expectedCandles(2, 600, 300)
	if e != nil {
		t.Error(e)
	}
	e = expectedCandles(2, 1800, 900)
	if e != nil {
		t.Error(e)
	}
	e = expectedCandles(2, 7200, 3600)
	if e != nil {
		t.Error(e)
	}
	e = expectedCandles(2, 43200, 21600)
	if e != nil {
		t.Error(e)
	}
	e = expectedCandles(2, 172800, 86400)
	if e != nil {
		t.Error(e)
	}
}

// expectedCandles uses the previous candle time window because the current one might not be complete and if used the test would become non-deterministic
func expectedCandles(expectedCandles int, timeRange time.Duration, candleGranularity int64) error {
	end := time.Now().UTC().Add(-time.Second * timeRange) // the latest candle may not yet be ready, so skipping to the previous one
	start := end.Add(-time.Second * timeRange)
	resp, err := c.GetHistoricRates(testPair, start.Format(time.RFC3339), end.Format(time.RFC3339), candleGranularity)
	if err != nil {
		return err
	}
	if len(resp) != expectedCandles {
		err := fmt.Errorf("expected %d candles, returned: %d", expectedCandles, len(resp))
		return err
	}
	return nil
}

func TestGetHistoricRatesGranularityCheck(t *testing.T) {
	end := time.Now().UTC()
	start := time.Now().UTC().Add(-time.Second * 300)
	invalidGranularity := 11
	_, err := c.GetHistoricRates(testPair, start.Format(time.RFC3339), end.Format(time.RFC3339), int64(invalidGranularity))
	if err == nil {
		t.Error("granularity validation did not work as expected")
	}
}

func TestGetStats(t *testing.T) {
	_, err := c.GetStats(testPair)
	if err != nil {
		t.Error("GetStats() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := c.GetCurrencies()
	if err != nil {
		t.Error("GetCurrencies() error", err)
	}
}

func TestGetServerTime(t *testing.T) {
	_, err := c.GetServerTime()
	if err != nil {
		t.Error("GetServerTime() error", err)
	}
}

func TestAuthRequests(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := c.GetAccounts()
	if err != nil {
		t.Error("GetAccounts() error", err)
	}
	accountResponse, err := c.GetAccount("13371337-1337-1337-1337-133713371337")
	if accountResponse.ID != "" {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	accountHistoryResponse, err := c.GetAccountHistory("13371337-1337-1337-1337-133713371337")
	if len(accountHistoryResponse) > 0 {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	getHoldsResponse, err := c.GetHolds("13371337-1337-1337-1337-133713371337")
	if len(getHoldsResponse) > 0 {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	orderResponse, err := c.PlaceLimitOrder("", 0.001, 0.001,
		order.Buy.Lower(), "", "", testPair, "", false)
	if orderResponse != "" {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	marketOrderResponse, err := c.PlaceMarketOrder("", 1, 0,
		order.Buy.Lower(), testPair, "")
	if marketOrderResponse != "" {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	fillsResponse, err := c.GetFills("1337", testPair)
	if len(fillsResponse) > 0 {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	_, err = c.GetFills("", "")
	if err == nil {
		t.Error("Expecting error")
	}
	_, err = c.GetFundingRecords("rejected")
	if err == nil {
		t.Error("Expecting error")
	}
	marginTransferResponse, err := c.MarginTransfer(1, "withdraw", "13371337-1337-1337-1337-133713371337", "BTC")
	if marginTransferResponse.ID != "" {
		t.Error("Expecting no data returned")
	}
	if err == nil {
		t.Error("Expecting error")
	}
	_, err = c.GetPosition()
	if err == nil {
		t.Error("Expecting error")
	}
	_, err = c.ClosePosition(false)
	if err == nil {
		t.Error("Expecting error")
	}
	_, err = c.GetPayMethods()
	if err != nil {
		t.Error("GetPayMethods() error", err)
	}
	_, err = c.GetCoinbaseAccounts()
	if err != nil {
		t.Error("GetCoinbaseAccounts() error", err)
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
		if resp, err := c.GetFee(feeBuilder); resp != float64(0.003) || err != nil {
			t.Error(err)
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.003), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := c.GetFee(feeBuilder); resp != float64(3000) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(3000), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.01), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := c.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}
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
	if resp, err := c.GetFee(feeBuilder); resp != float64(0.15) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := c.GetFee(feeBuilder); resp != float64(25) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestCalculateTradingFee(t *testing.T) {
	t.Parallel()
	// uppercase
	var volume = []Volume{
		{
			ProductID: "BTC_USD",
			Volume:    100,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.003) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.003), resp)
	}

	// lowercase
	volume = []Volume{
		{
			ProductID: "btc_usd",
			Volume:    100,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.003) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.003), resp)
	}

	// mixedCase
	volume = []Volume{
		{
			ProductID: "btc_USD",
			Volume:    100,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.003) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.003), resp)
	}

	// medium volume
	volume = []Volume{
		{
			ProductID: "btc_USD",
			Volume:    10000001,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.002) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
	}

	// high volume
	volume = []Volume{
		{
			ProductID: "btc_USD",
			Volume:    100000010000,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0.001) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
	}

	// no match
	volume = []Volume{
		{
			ProductID: "btc_beeteesee",
			Volume:    100000010000,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, false); resp != float64(0) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}

	// taker
	volume = []Volume{
		{
			ProductID: "btc_USD",
			Volume:    100000010000,
		},
	}

	if resp := c.calculateTradingFee(volume, currency.BTC, currency.USD, "_", 1, 1, true); resp != float64(0) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.AutoWithdrawFiatWithAPIPermissionText
	withdrawPermissions := c.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
		Currencies: []currency.Pair{currency.NewPair(currency.BTC,
			currency.LTC)},
	}

	_, err := c.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
		Currencies: []currency.Pair{currency.NewPair(currency.BTC,
			currency.LTC)},
	}

	_, err := c.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
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
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		OrderSide: order.Buy,
		OrderType: order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
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

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
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
	withdrawCryptoRequest := withdraw.CryptoRequest{
		GenericInfo: withdraw.GenericInfo{
			Amount:      -1,
			Currency:    currency.BTC,
			Description: "WITHDRAW IT ALL",
		},
		Address: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := c.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
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

	var withdrawFiatRequest = withdraw.FiatRequest{
		GenericInfo: withdraw.GenericInfo{
			Amount:   100,
			Currency: currency.USD,
		},
		BankName: "Federal Reserve Bank",
	}

	_, err := c.WithdrawFiatFunds(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.FiatRequest{
		GenericInfo: withdraw.GenericInfo{
			Amount:   100,
			Currency: currency.USD,
		},
		BankName: "Federal Reserve Bank",
	}

	_, err := c.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := c.GetDepositAddress(currency.BTC, "")
	if err == nil {
		t.Error("GetDepositAddress() error", err)
	}
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	if !c.Websocket.IsEnabled() && !c.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	c.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         c.Name,
		URL:                  c.Websocket.GetWebsocketURL(),
		Verbose:              c.Verbose,
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
	err = c.Subscribe(wshandler.WebsocketChannelSubscription{
		Channel:  "user",
		Currency: currency.NewPairFromString(testPair),
	})
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case badResponse := <-c.Websocket.DataHandler:
		t.Error(badResponse)
	case <-timer.C:
	}
	timer.Stop()
}
