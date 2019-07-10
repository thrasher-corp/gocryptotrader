package lbank

import (
	"fmt"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey    = ""
	testAPISecret = ""
)

var l Lbank
var setupRan bool

func TestSetDefaults(t *testing.T) {
	l.SetDefaults()
}

func TestSetup(t *testing.T) {
	if setupRan {
		return
	}
	setupRan = true

	t.Parallel()
	l.SetDefaults()
	l.APIKey = testAPIKey
	l.APISecret = testAPISecret
	l.loadPrivKey()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json")
	if err != nil {
		t.Fatal("Test Failed - Lbank Setup() init error", err)
	}
	lbankConfig, err := cfg.GetExchangeConfig("Lbank")
	lbankConfig.Websocket = true
	if err != nil {
		t.Fatal("Test Failed - Lbank Setup() init error", err)
	}

	l.Setup(&lbankConfig)
}

func TestGetTicker(t *testing.T) {
	TestSetup(t)
	_, err := l.GetTicker("btc_usdt")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetCurrencyPairs(t *testing.T) {
	TestSetup(t)
	_, err := l.GetCurrencyPairs()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetMarketDepths(t *testing.T) {
	TestSetup(t)
	_, err := l.GetMarketDepths("btc_usdt", "60", "1")
	if err != nil {
		t.Fatalf("GetMarketDepth failed: %v", err)
	}
	a, _ := l.GetMarketDepths("btc_usdt", "60", "0")
	if len(a.Asks) != 60 {
		t.Fatal("length requested doesnt match the output")
	}
	_, err = l.GetMarketDepths("btc_usdt", "61", "0")
	if err == nil {
		t.Fatal("size is greater than the maximum allowed")
	}
}

func TestGetTrades(t *testing.T) {
	TestSetup(t)
	_, err := l.GetTrades("btc_usdt", "600", fmt.Sprintf("%v", time.Now().Unix()))
	if err != nil {
		t.Fatal(err)
	}
	a, err := l.GetTrades("btc_usdt", "600", "0")
	if len(a) != 600 && err != nil {
		t.Fatal(err)
	}
}

func TestGetKlines(t *testing.T) {
	TestSetup(t)
	_, err := l.GetKlines("btc_usdt", "600", "minute1", fmt.Sprintf("%v", time.Now().Unix()))
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	TestSetup(t)
	p := currency.Pair{
		Delimiter: "_",
		Base:      currency.ETH,
		Quote:     currency.BTC}

	_, err := l.UpdateOrderbook(p.Lower(), "spot")
	if err != nil {
		t.Fatalf("Update for orderbook failed: %v", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := l.GetUserInfo()
	if err != nil {
		t.Error("invalid key or sign", err)
	}
}

func TestCreateOrder(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.CreateOrder(cp.Lower().String(), "what", 1231, 12314)
	if err == nil {
		t.Error("Test Failed - CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(cp.Lower().String(), "buy", 0, 0)
	if err == nil {
		t.Error("Test Failed - CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(cp.Lower().String(), "sell", 1231, 0)
	if err == nil {
		t.Error("Test Failed - CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(cp.Lower().String(), "buy", 58, 681)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRemoveOrder(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.ETH.String(), currency.BTC.String(), "_")
	_, err := l.RemoveOrder(cp.Lower().String(), "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	if err != nil {
		t.Error(err)
	}
}

func TestQueryOrder(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrder(cp.Lower().String(), "1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQueryOrderHistory(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	havealook, err := l.QueryOrderHistory(cp.Lower().String(), "1", "50")
	if err != nil {
		t.Error(err)
	}

	log.Println(havealook)
}

func TestGetPairInfo(t *testing.T) {
	TestSetup(t)
	_, err := l.GetPairInfo()
	if err != nil {
		t.Error("somethings wrong")
	}
}

func TestGetOpenOrders(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.GetOpenOrders(cp.Lower().String(), 1, 50)
	if err != nil {
		t.Error("unexpected error", err)
	}
}

func TestUSD2RMBRate(t *testing.T) {
	TestSetup(t)
	_, err := l.USD2RMBRate()
	if err != nil {
		t.Error("wtf")
	}
}

func TestGetWithdrawConfig(t *testing.T) {
	TestSetup(t)
	curr := "eth"
	_, err := l.GetWithdrawConfig(curr)
	if err != nil {
		t.Error("wtf", err)
	}
}

func TestWithdraw(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := l.Withdraw("", "", "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawRecords(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := l.GetWithdrawalRecords("eth", "0", "1", "20")
	if err != nil {
		t.Error(err)
	}
}

func TestLoadPrivKey(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	l.SetDefaults()
	l.APISecret = testAPISecret
	err := l.loadPrivKey()
	if err != nil {
		t.Error(err)
	}
	l.privKeyLoaded = false
	l.APISecret = "errortest"
	err = l.loadPrivKey()
	if err == nil {
		t.Errorf("expected error due to pemblock nil, got err: %v", err)
	}
}

func TestSign(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	l.SetDefaults()
	l.APISecret = testAPISecret
	l.loadPrivKey()
	_, err := l.sign("wtf", l.privateKey)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.SubmitOrder(cp.Lower(), "BUY", "ANY", 2, 1312, "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.ETH.String(), currency.BTC.String(), "_")
	var a exchange.OrderCancellation
	a.CurrencyPair = cp
	a.OrderID = "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23"
	err := l.CancelOrder(&a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := l.GetOrderInfo("9ead39f5-701a-400b-b635-d7349eb0f6b")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllOpenOrderID(t *testing.T) {
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := l.GetAllOpenOrderID()
	if err != nil {
		t.Error(err)
	}
}
