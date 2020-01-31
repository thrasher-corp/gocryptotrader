package coinbene

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := c.GetOrderbook(spotTestPair, 100)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := c.GetTicker(spotTestPair)
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

func TestGetAcounntBalances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetAccountBalances()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountAssetBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetAccountAssetBalance(currency.BTC.String())
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := c.PlaceSpotOrder(
		1,
		1,
		spotTestPair,
		order.Buy.Lower(),
		order.Limit.Lower(),
		"Sup3rAw3s0m3Cl13ntiDH",
		0,
	)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.PlaceSpotOrders(
		[]PlaceOrderRequest{
			{
				1,
				1,
				spotTestPair,
				order.Buy.Lower(),
				order.Limit.Lower(),
				"Sup3rAw3s0m3Cl13ntiDH",
				0,
			},
		})
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchOpenSpotOrders(spotTestPair)
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

func TestFetchOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchSpotOrderInfo("adfjashjgsag")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSpotOrderFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetSpotOrderFills("1912131427156307968")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSpotOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := c.CancelSpotOrder("adfjashjgsag")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSpotOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.CancelSpotOrders([]string{"578639816552972288", "578639902896914432"})
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter("BTC", "USDT", "/")
	_, err := c.UpdateTicker(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = c.UpdateTicker(cp, asset.PerpetualSwap)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.UpdateAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter("BTC", "USDT", "/")
	_, err := c.UpdateOrderbook(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = c.UpdateOrderbook(cp, asset.PerpetualSwap)
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
	_, err := c.GetSwapOrderbook(swapTestPair, 100)
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
	_, err := c.GetSwapTrades(swapTestPair, 10)
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
		order.Buy.Lower(),
		"limit",
		"fixed",
		"12345",
		1,
		1,
		2)
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

	_, err := c.GetSwapOpenOrdersByPage(swapTestPair, 0)
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

	_, err := c.GetSwapOrderHistory("", "", swapTestPair, 1, 10, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderHistoryByOrderID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOrderHistoryByOrderID("", "", swapTestPair, "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSwapOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.CancelSwapOrders([]string{"578639816552972288", "578639902896914432"})
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapOrderFills(swapTestPair, "5807143157122003", 580714315825905664)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapFundingRates(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}

	_, err := c.GetSwapFundingRates(1, 2)
	if err != nil {
		t.Error(err)
	}
}
