package coinbene

import (
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey    = ""
	testAPISecret = ""
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

func TestFetchTicker(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.FetchTicker("btcusdt")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderbooks(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.FetchOrderbooks("btcusdt")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.GetTrades("btcusdt", "")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllPairs(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.GetAllPairs()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserBalance(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.GetUserBalance()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceOrder(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.PlaceOrder(140, 1, "btcusdt", "buy-limit")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderInfo(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.FetchOrderInfo("adfjashjgsag")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveOrder(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.RemoveOrder("adfjashjgsag")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOpenOrders(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.FetchOpenOrders("btcusdt")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawApply(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.WithdrawApply(1000, "usd", "2394u23fm", "WTF", "usd-btc-ltc-usd")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	cp := currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	_, err := c.UpdateTicker(cp, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	TestSetup(t)
	c.Verbose = true
	a, err := c.GetAccountInfo()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}
