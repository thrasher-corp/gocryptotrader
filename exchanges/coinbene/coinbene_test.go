package coinbene

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
	spotTestPair            = "BTC/USDT"
	swapTestPair            = "BTCUSDT"
)

var c Coinbene

func TestMain(m *testing.M) {
	c.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	coinbeneConfig, err := cfg.GetExchangeConfig("Coinbene")
	if err != nil {
		log.Fatal(err)
	}
	coinbeneConfig.API.AuthenticatedWebsocketSupport = true
	coinbeneConfig.API.AuthenticatedSupport = true
	coinbeneConfig.API.Credentials.Secret = testAPISecret
	coinbeneConfig.API.Credentials.Key = testAPIKey

	err = c.Setup(coinbeneConfig)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return c.AllowAuthenticatedRequest()
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := c.GetTicker(spotTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := c.GetOrderbook(spotTestPair, 100)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := c.GetTrades(spotTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllPairs(t *testing.T) {
	t.Parallel()
	_, err := c.GetAllPairs()
	if err != nil {
		t.Error(err)
	}
}

func TestGetPairInfo(t *testing.T) {
	t.Parallel()
	_, err := c.GetPairInfo(spotTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetUserBalance()
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := c.PlaceOrder(1, 1, spotTestPair, order.Buy.Lower(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchOrderInfo("adfjashjgsag")
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := c.RemoveOrder("adfjashjgsag")
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchOpenOrders(spotTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchClosedOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchClosedOrders(spotTestPair, "")
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter("BTC", "USDT", "/")
	_, err := c.UpdateTicker(cp, "spot")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter("BTC", "USDT", "/")
	_, err := c.UpdateOrderbook(cp, "spot")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapTickers(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapTickers()
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapTicker(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapTicker(swapTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderbook(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapOrderbook(swapTestPair, "100")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapKlines(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapKlines(swapTestPair,
		"1573184608",
		"1573184808",
		"1")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapTrades(t *testing.T) {
	t.Parallel()
	_, err := c.GetSwapTrades(swapTestPair, "10")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetSwapAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapPositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetSwapPositions(swapTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceSwapOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.PlaceSwapOrder(swapTestPair,
		"openLong",
		"limit",
		"fixed",
		"12345",
		100000,
		1,
		1)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSwapOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.CancelSwapOrder("1337")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenSwapOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOpenOrders(swapTestPair, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOpenOrdersByPage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOpenOrdersByPage(swapTestPair, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOrderInfo("1337")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOrderHistory("", "", swapTestPair, "1", "10", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderHistoryByOrderID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOrderHistoryByOrderID("", "", swapTestPair, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelMultipleSwapOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.CancelMultipleSwapOrders([]string{"578639816552972288", "578639902896914432"})
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOrderFills(swapTestPair, "5807143157122003", "580714315825905664")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapFundingRates(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapFundingRates("1", "2")
	if err != nil {
		t.Error(err)
	}
}
