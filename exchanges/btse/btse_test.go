package btse

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	testPair                = "BTC-USD"
)

var b BTSE

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	btseConfig, err := cfg.GetExchangeConfig("BTSE")
	if err != nil {
		log.Fatal(err)
	}

	btseConfig.API.AuthenticatedSupport = true
	btseConfig.API.Credentials.Key = apiKey
	btseConfig.API.Credentials.Secret = apiSecret

	err = b.Setup(btseConfig)
	if err != nil {
		log.Fatal(err)
	}
	b.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	b.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials()
}

func TestGetMarketsSummary(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketsSummary()
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets()
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.FetchOrderBook(testPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrades(testPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker(testPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketStatistics(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketStatistics(testPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetServerTime()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetAccountBalance()
	if err != nil {
		t.Error(err)
	}
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetFills("", testPair, "", "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := b.CreateOrder(0.1,
		10000,
		order.Sell.String(),
		order.Limit.String(),
		testPair,
		"",
		"")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetOrders("")
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType,
	}
	_, err := b.GetOrderHistory(&getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expected := exchange.NoAPIWithdrawalMethodsText
	actual := b.FormatWithdrawPermissions()
	if actual != expected {
		t.Errorf("Expected: %s, Received: %s", expected, actual)
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.USD),
		IsMaker:       true,
		Amount:        1,
		PurchasePrice: 1000,
	}

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
	t.Parallel()

	feeBuilder := &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.USD),
		IsMaker:       true,
		Amount:        1,
		PurchasePrice: 1000,
	}

	if resp, err := b.GetFee(feeBuilder); resp != 0.500000 || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", 0.500000, resp)
		t.Error(err)
	}

	feeBuilder.IsMaker = false
	if resp, err := b.GetFee(feeBuilder); resp != 1.00000 || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", 1.00000, resp)
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != 0.0005 || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", 0.0005, resp)
		t.Error(err)
	}

	feeBuilder.Pair.Base = currency.USDT
	if resp, err := b.GetFee(feeBuilder); resp != 1.080000 || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", 1.080000, resp)
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(3) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(3), resp)
		t.Error(err)
	}

	feeBuilder.Amount = 1000000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(900) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(900), resp)
		t.Error(err)
	}

	feeBuilder.Amount = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(25) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(25), resp)
		t.Error(err)
	}
}

func TestParseOrderTime(t *testing.T) {
	expected := int64(1534794360)
	actual, err := parseOrderTime("2018-08-20 19:20:46")
	if err != nil {
		t.Fatal(err)
	}
	if expected != actual.Unix() {
		t.Errorf("TestParseOrderTime expected: %d, got %d", expected, actual.Unix())
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.USD,
		},
		Side:     order.Buy,
		Type:     order.Limit,
		Price:    100000,
		Amount:   0.1,
		ClientID: "meowOrder",
	}
	response, err := b.SubmitOrder(orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	currencyPair := currency.NewPairWithDelimiter(currency.BTC.String(),
		currency.USD.String(),
		"-")
	var orderCancellation = &order.Cancel{
		ID:            "b334ecef-2b42-4998-b8a4-b6b14f6d2671",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
	}
	err := b.CancelOrder(orderCancellation)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	currencyPair := currency.NewPairWithDelimiter(currency.BTC.String(),
		currency.USD.String(),
		"-")
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
	}
	resp, err := b.CancelAllOrders(orderCancellation)

	if err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
	for k, v := range resp.Status {
		if strings.Contains(v, "Failed") {
			t.Errorf("order id: %s failed to cancel: %v", k, v)
		}
	}
}

func TestWsOrderbook(t *testing.T) {
	pressXToJSON := []byte(`{"topic":"orderBookApi:BTC-USD_0","data":{"buyQuote":[{"price":"9272.0","size":"0.077"},{"price":"9271.0","size":"1.122"},{"price":"9270.0","size":"2.548"},{"price":"9267.5","size":"1.015"},{"price":"9265.5","size":"0.930"},{"price":"9265.0","size":"0.475"},{"price":"9264.5","size":"2.216"},{"price":"9264.0","size":"9.709"},{"price":"9263.5","size":"3.667"},{"price":"9263.0","size":"8.481"},{"price":"9262.5","size":"7.660"},{"price":"9262.0","size":"9.689"},{"price":"9261.5","size":"4.213"},{"price":"9261.0","size":"1.491"},{"price":"9260.5","size":"6.264"},{"price":"9260.0","size":"1.690"},{"price":"9259.5","size":"5.718"},{"price":"9259.0","size":"2.706"},{"price":"9258.5","size":"0.192"},{"price":"9258.0","size":"1.592"},{"price":"9257.5","size":"1.749"},{"price":"9257.0","size":"8.104"},{"price":"9256.0","size":"0.161"},{"price":"9252.0","size":"1.544"},{"price":"9249.5","size":"1.462"},{"price":"9247.5","size":"1.833"},{"price":"9247.0","size":"0.168"},{"price":"9245.5","size":"1.941"},{"price":"9244.0","size":"1.423"},{"price":"9243.5","size":"0.175"}],"currency":"USD","sellQuote":[{"price":"9303.5","size":"1.839"},{"price":"9303.0","size":"2.067"},{"price":"9302.0","size":"0.117"},{"price":"9298.5","size":"1.569"},{"price":"9297.0","size":"1.527"},{"price":"9295.0","size":"0.184"},{"price":"9294.0","size":"1.785"},{"price":"9289.0","size":"1.673"},{"price":"9287.5","size":"4.194"},{"price":"9287.0","size":"6.622"},{"price":"9286.5","size":"2.147"},{"price":"9286.0","size":"3.348"},{"price":"9285.5","size":"5.655"},{"price":"9285.0","size":"10.423"},{"price":"9284.5","size":"6.233"},{"price":"9284.0","size":"8.860"},{"price":"9283.5","size":"9.441"},{"price":"9283.0","size":"3.455"},{"price":"9282.5","size":"11.033"},{"price":"9282.0","size":"11.471"},{"price":"9281.5","size":"4.742"},{"price":"9281.0","size":"14.789"},{"price":"9280.5","size":"11.117"},{"price":"9280.0","size":"0.807"},{"price":"9279.5","size":"1.651"},{"price":"9279.0","size":"0.244"},{"price":"9278.5","size":"0.533"},{"price":"9277.0","size":"1.447"},{"price":"9273.0","size":"1.976"},{"price":"9272.5","size":"0.093"}]}}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrades(t *testing.T) {
	pressXToJSON := []byte(`{"topic":"tradeHistory:BTC-USD","data":[{"amount":0.09,"gain":1,"newest":0,"price":9273.6,"serialId":0,"transactionUnixtime":1580349090693}]}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderNotification(t *testing.T) {
	status := []string{"ORDER_INSERTED", "ORDER_CANCELLED", "TRIGGER_INSERTED", "ORDER_FULL_TRANSACTED", "ORDER_PARTIALLY_TRANSACTED", "INSUFFICIENT_BALANCE", "TRIGGER_ACTIVATED", "MARKET_UNAVAILABLE"}
	for i := range status {
		pressXToJSON := []byte(`{"topic": "notificationApi","data": [{"symbol": "BTC-USD","orderID": "1234","orderMode": "MODE_BUY","orderType": "TYPE_LIMIT","price": "1","size": "1","status": "` + status[i] + `","timestamp": "1580349090693","type": "STOP","triggerPrice": "1"}]}`)
		err := b.wsHandleData(pressXToJSON)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestStatusToStandardStatus(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []TestCases{
		{Case: "ORDER_INSERTED", Result: order.New},
		{Case: "TRIGGER_INSERTED", Result: order.New},
		{Case: "ORDER_CANCELLED", Result: order.Cancelled},
		{Case: "ORDER_FULL_TRANSACTED", Result: order.Filled},
		{Case: "ORDER_PARTIALLY_TRANSACTED", Result: order.PartiallyFilled},
		{Case: "TRIGGER_ACTIVATED", Result: order.Active},
		{Case: "INSUFFICIENT_BALANCE", Result: order.InsufficientBalance},
		{Case: "MARKET_UNAVAILABLE", Result: order.MarketUnavailable},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := stringToOrderStatus(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Exepcted: %v, received: %v", testCases[i].Result, result)
		}
	}
}
