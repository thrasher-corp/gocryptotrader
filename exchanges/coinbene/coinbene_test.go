package coinbene

import (
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
)

var c Coinbene
var setupRan bool
var m sync.Mutex

func TestSetup(t *testing.T) {
	t.Parallel()
	m.Lock()
	defer m.Unlock()

	if setupRan {
		return
	}
	c.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json")
	if err != nil {
		t.Errorf("Test Failed - Coinbene Setup() init error:, %v", err)
	}
	coinbeneConfig, err := cfg.GetExchangeConfig("Coinbene")
	if err != nil {
		t.Errorf("Test Failed - Coinbene Setup() init error: %v", err)
	}
	coinbeneConfig.Websocket = true
	coinbeneConfig.AuthenticatedAPISupport = true
	coinbeneConfig.APISecret = testAPISecret
	coinbeneConfig.APIKey = testAPIKey
	c.Setup(&coinbeneConfig)
	setupRan = true
}

func areTestAPIKeysSet() bool {
	if c.APIKey != "" && c.APIKey != "Key" &&
		c.APISecret != "" && c.APISecret != "Secret" {
		return true
	}
	return false
}

func TestFetchTicker(t *testing.T) {
	TestSetup(t)
	_, err := c.FetchTicker("BTC/USDT")
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderbooks(t *testing.T) {
	TestSetup(t)
	_, err := c.FetchOrderbooks("BTC/USDT", 100)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	TestSetup(t)
	_, err := c.GetTrades("BTC/USDT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllPairs(t *testing.T) {
	TestSetup(t)
	_, err := c.GetAllPairs()
	if err != nil {
		t.Error(err)
	}
}

func TestGetPairInfo(t *testing.T) {
	TestSetup(t)
	_, err := c.GetPairInfo("BTC/USDT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserBalance(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetUserBalance()
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceOrder(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := c.PlaceOrder(140, 1, "BTC/USDT", "1", "")
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderInfo(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchOrderInfo("adfjashjgsag")
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveOrder(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := c.RemoveOrder("adfjashjgsag")
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOpenOrders(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchOpenOrders("BTC/USDT")
	if err != nil {
		t.Error(err)
	}
}

func TestFetchClosedOrders(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.FetchClosedOrders("BTC/USDT", "")
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	TestSetup(t)
	cp := currency.NewPairWithDelimiter("BTC", "USDT", "/")
	_, err := c.UpdateTicker(cp, "spot")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := c.GetAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	TestSetup(t)
	cp := currency.NewPairWithDelimiter("BTC", "USDT", "/")
	_, err := c.UpdateOrderbook(cp, "spot")
	if err != nil {
		t.Error(err)
	}
}
