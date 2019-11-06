package coinbene

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false

	btcusdt = "BTC/USDT"
)

var c Coinbene

func TestMain(m *testing.M) {
	c.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json")
	if err != nil {
		log.Fatalf("Test Failed - Coinbene Setup() init error:, %v", err)
	}
	coinbeneConfig, err := cfg.GetExchangeConfig("Coinbene")
	if err != nil {
		log.Fatalf("Test Failed - Coinbene Setup() init error: %v", err)
	}
	coinbeneConfig.Websocket = true
	coinbeneConfig.AuthenticatedAPISupport = true
	coinbeneConfig.APISecret = testAPISecret
	coinbeneConfig.APIKey = testAPIKey
	c.Setup(&coinbeneConfig)

	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	if c.APIKey != "" && c.APIKey != "Key" &&
		c.APISecret != "" && c.APISecret != "Secret" {
		return true
	}
	return false
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	_, err := c.FetchTicker(btcusdt)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderbooks(t *testing.T) {
	t.Parallel()
	_, err := c.FetchOrderbooks(btcusdt, 100)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := c.GetTrades(btcusdt)
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
	_, err := c.GetPairInfo(btcusdt)
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
	_, err := c.PlaceOrder(140, 1, btcusdt, "buy", "")
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
	_, err := c.FetchOpenOrders(btcusdt)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchClosedOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchClosedOrders(btcusdt, "")
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
