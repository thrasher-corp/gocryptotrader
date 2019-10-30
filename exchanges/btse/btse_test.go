package btse

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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

	b.Setup(btseConfig)
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
		OrderType: order.AnyType,
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
		OrderType: order.AnyType,
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
	actual := parseOrderTime("2018-08-20 19:20:46").Unix()
	if expected != actual {
		t.Errorf("TestParseOrderTime expected: %d, got %d", expected, actual)
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
		OrderSide: order.Buy,
		OrderType: order.Limit,
		Price:     100000,
		Amount:    0.1,
		ClientID:  "meowOrder",
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
		OrderID:       "b334ecef-2b42-4998-b8a4-b6b14f6d2671",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
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
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
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
