package oex

import (
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
)

var o Oex
var setupRan bool
var m sync.Mutex

func TestSetup(t *testing.T) {
	t.Parallel()
	m.Lock()
	defer m.Unlock()

	if setupRan {
		return
	}
	o.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json")
	if err != nil {
		t.Errorf("Test Failed - Oex Setup() init error:, %v", err)
	}
	oexConfig, err := cfg.GetExchangeConfig("Oex")
	if err != nil {
		t.Errorf("Test Failed - Oex Setup() init error: %v", err)
	}
	oexConfig.Websocket = true
	oexConfig.AuthenticatedAPISupport = true
	oexConfig.APISecret = testAPISecret
	oexConfig.APIKey = testAPIKey
	o.Setup(&oexConfig)
	setupRan = true
}

func areTestAPIKeysSet() bool {
	if o.APIKey != "" && o.APIKey != "Key" &&
		o.APISecret != "" && o.APISecret != "Secret" {
		return true
	}
	return false
}

func TestGetTicker(t *testing.T) {
	TestSetup(t)
	_, err := o.GetTicker("btcusdt")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllTicker(t *testing.T) {
	TestSetup(t)
	_, err := o.GetAllTicker()
	if err != nil {
		t.Error(err)
	}
}

func TestGetKline(t *testing.T) {
	TestSetup(t)
	_, err := o.GetKline("btcusdt", "1")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	TestSetup(t)
	_, err := o.GetTrades("btcusdt")
	if err != nil {
		t.Error(err)
	}
}

func TestLatestCurrencyPrices(t *testing.T) {
	TestSetup(t)
	_, err := o.LatestCurrencyPrices(time.Now().String())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketDepth(t *testing.T) {
	TestSetup(t)
	_, err := o.GetMarketDepth("ethbtc", "step1")
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	TestSetup(t)
	cp := currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	_, err := o.UpdateTicker(cp, "")
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderBook(t *testing.T) {
	TestSetup(t)
	cp := currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	_, err := o.UpdateOrderbook(cp, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllPairs(t *testing.T) {
	TestSetup(t)
	_, err := o.GetAllPairs()
	if err != nil {
		t.Log(err)
	}
}

func TestGetUserInfo(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := o.GetUserInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllOrders(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := o.GetAllOrders("btcusdt", "", "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestFindOrderHistory(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := o.FindOrderHistory("btcusdt", "", "", "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveOrder(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := o.RemoveOrder("", "btcusdt")
	if err != nil {
		t.Errorf("expected err due to wrong order id but got: %v", err)
	}
}

func TestRemoveAllOrders(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := o.RemoveAllOrders("btceth")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateOrder(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := o.CreateOrder("buy", "limit order", "1", "1000", "btcusdt", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := o.GetOpenOrders("btcusdt", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestSelfTrade(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := o.SelfTrade("buy", "limit order", "1", "", "ethbtc", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserAssetData(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := o.GetUserAssetData("", "0412437212", "")
	if err != nil {
		t.Log(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := o.GetAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderInfo(t *testing.T) {
	TestSetup(t)
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := o.FetchOrderInfo("ksldfkaig", "btcusdt")
	if err != nil {
		t.Error(err)
	}
}
