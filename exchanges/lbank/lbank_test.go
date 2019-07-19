package lbank

import (
	"fmt"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey          = ""
	testAPISecret       = ""
	canManipulateOrders = false
)

var l Lbank
var setupRan bool

func TestSetup(t *testing.T) {
	if setupRan {
		return
	}
	setupRan = true

	t.Parallel()
	l.SetDefaults()
	l.APIKey = testAPIKey
	l.APISecret = testAPISecret
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json")
	if err != nil {
		t.Errorf("Test Failed - Lbank Setup() init error:, %v", err)
	}
	lbankConfig, err := cfg.GetExchangeConfig("Lbank")
	if err != nil {
		t.Errorf("Test Failed - Lbank Setup() init error: %v", err)
	}
	lbankConfig.Websocket = true
	l.Setup(&lbankConfig)
}

func areTestAPIKeysSet() bool {
	if l.APIKey != "" && l.APIKey != "Key" &&
		l.APISecret != "" && l.APISecret != "Secret" {
		return true
	}
	return false
}

func TestGetTicker(t *testing.T) {
	TestSetup(t)
	_, err := l.GetTicker("btc_usdt")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyPairs(t *testing.T) {
	TestSetup(t)
	_, err := l.GetCurrencyPairs()
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketDepths(t *testing.T) {
	TestSetup(t)
	_, err := l.GetMarketDepths("btc_usdt", "60", "1")
	if err != nil {
		t.Errorf("GetMarketDepth failed: %v", err)
	}
	a, _ := l.GetMarketDepths("btc_usdt", "60", "0")
	if len(a.Asks) != 60 {
		t.Errorf("length requested doesnt match the output")
	}
	_, err = l.GetMarketDepths("btc_usdt", "61", "0")
	if err == nil {
		t.Errorf("size is greater than the maximum allowed")
	}
}

func TestGetTrades(t *testing.T) {
	TestSetup(t)
	_, err := l.GetTrades("btc_usdt", "600", fmt.Sprintf("%v", time.Now().Unix()))
	if err != nil {
		t.Error(err)
	}
	a, err := l.GetTrades("btc_usdt", "600", "0")
	if len(a) != 600 && err != nil {
		t.Error(err)
	}
}

func TestGetKlines(t *testing.T) {
	TestSetup(t)
	_, err := l.GetKlines("btc_usdt", "600", "minute1", fmt.Sprintf("%v", time.Now().Unix()))
	if err != nil {
		t.Error(err)
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
		t.Errorf("Update for orderbook failed: %v", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	_, err := l.GetUserInfo()
	if err != nil {
		t.Error("invalid key or sign", err)
	}
}

func TestCreateOrder(t *testing.T) {
	areTestAPIKeysSet()
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
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.ETH.String(), currency.BTC.String(), "_")
	_, err := l.RemoveOrder(cp.Lower().String(), "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	if err != nil {
		t.Errorf("unable to remove order: %v", err)
	}
}

func TestQueryOrder(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrder(cp.Lower().String(), "1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQueryOrderHistory(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrderHistory(cp.Lower().String(), "1", "50")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPairInfo(t *testing.T) {
	TestSetup(t)
	_, err := l.GetPairInfo()
	if err != nil {
		t.Errorf("couldnt get pair info: %v", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	areTestAPIKeysSet()
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
		t.Error("unable to acquire the rate")
	}
}

func TestGetWithdrawConfig(t *testing.T) {
	TestSetup(t)
	_, err := l.GetWithdrawConfig("eth")
	if err != nil {
		t.Errorf("unable to get withdraw config: %v", err)
	}
}

func TestWithdraw(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	_, err := l.Withdraw("", "", "", "", "")
	if err != nil {
		t.Errorf("unable to withdraw: %v", err)
	}
}

func TestGetWithdrawRecords(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	_, err := l.GetWithdrawalRecords("eth", "0", "1", "20")
	if err != nil {
		t.Errorf("unable to get withdrawal records: %v", err)
	}
}

func TestLoadPrivKey(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
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
	TestSetup(t)
	areTestAPIKeysSet()

	l.APISecret = testAPISecret
	l.loadPrivKey()
	_, err := l.sign("hello123")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.SubmitOrder(cp.Lower(), "BUY", "ANY", 2, 1312, "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	areTestAPIKeysSet()
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
	areTestAPIKeysSet()
	TestSetup(t)
	_, err := l.GetOrderInfo("9ead39f5-701a-400b-b635-d7349eb0f6b")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllOpenOrderID(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	_, err := l.GetAllOpenOrderID()
	if err != nil {
		t.Error(err)
	}
}

func TestGetFeeByType(t *testing.T) {
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	var input exchange.FeeBuilder
	input.Amount = 2
	input.FeeType = exchange.CryptocurrencyWithdrawalFee
	input.Pair = cp
	a, err := l.GetFeeByType(&input)
	if err != nil {
		t.Error(err)
	}
	if a != 0.0005 {
		t.Errorf("testGetFeeByType failed. Expected: 0.0005, Received: %v", a)
	}
}
