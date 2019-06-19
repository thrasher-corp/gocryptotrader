package lbank

import (
	"fmt"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey    = "9820cd63-6f97-4086-a370-0fb89b7ec3c3"
	testAPISecret = "MIICeQIBADANBgkqhkiG9w0BAQEFAASCAmMwggJfAgEAAoGBAPTBCsEhejCYzRZ9WvxvZzdueQFscHCxNJ9fFoHfeLPpXtBQM/929aKZT72zkIxkif7NUDAZufvIm9ejlyENs0QT3bs9YRjc6dJ/uhuUGhsN/sHokU0DpQ8i0S7sQL5P0LEBjbsR/+e0L43YDMnRVof5yvd3KZ9DVgVhoGfivYfZAgMBAAECgYEAqAIXWsmbMd7B8W0tVtk2FhPsVnDUolbSE5BXR+FZ3s4UepSDjRpgtTPeTA8F64lcPJ89KzeNtmtXpuex50ubQIlih8NaNOiCivEA5jnlOTuJgWx+zXmywlMCt91JXi+2+C4BG0PemArwhUIl1miW5WPEiTfLMHrp7t9eFrT4qs0CQQD82UGBqAj475uj9WoPAEwkkU+lIEylGavO+frJBu417a1cFAWG7g+E5Uv+3s+Ua+RYIg9yjIwT6kLyTX79/dcrAkEA9831mYu1AMHurq1t4ifRTxQ5hewwtSFbLhKETNP3fB2pVnjdjD4lUR347RK6Yc56E0/AuReaGIaLyus9O4tbCwJBALmxyPUy9lv0hSa1/w1DV6hne8m23fNG1jIszuzChUHf6zi7j4+X2JfuWpC1DFhhoJLFePjUla+ulTokhgZ9XX8CQQCRYYryb01cyWovnt31raiVvWbWFDCrQ4uL5x8pN75dWcWMTtKjwZ4BDhWJeNBSK2HhTIvjy14Df4QqI4LEGUjrAkEA/DIKkFIUMU3ZhAtvGi0EB3TetxtMiwHttktWMdaBnvAAAR8pF7oqmaSTJ97isVILf/814er2WmahLBLEplUddA=="
)

var l Lbank

func TestSetDefaults(t *testing.T) {
	l.SetDefaults()
}

func TestSetup(t *testing.T) {
	t.Parallel()
	l.SetDefaults()
	l.APIKey = testAPIKey
	l.APISecret = testAPISecret
	l.loadPrivKey()
	l.Verbose = true
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json")
	if err != nil {
		t.Fatal("Test Failed - Lbank Setup() init error", err)
	}
	lbankConfig, err := cfg.GetExchangeConfig("Lbank")
	if err != nil {
		t.Fatal("Test Failed - Lbank Setup() init error", err)
	}

	// lbankConfig.AuthenticatedAPISupport = true
	// lbankConfig.APIKey = testAPIKey
	// lbankConfig.APISecret = testAPISecret

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
	an, err := l.GetMarketDepths("btc_usdt", "60", "1")
	t.Log(an)
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
		Base:      currency.BTC,
		Quote:     currency.USD}

	_, err := l.UpdateOrderbook(p, "spot")
	if err != nil {
		t.Fatalf("Update for orderbook failed: %v", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	TestSetup(t)
	meow, err := l.GetUserInfo()
	if err != nil {
		t.Error("invalid key or sign", err)
	}

	log.Println(meow)

	for key, val := range meow.Freeze {
		log.Println(key, val)
	}

	t.Error()
}

func TestCreateOrder(t *testing.T) {
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

func TestCancelOrder(t *testing.T) {
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.CancelOrder(cp.Lower().String(), "lkjsdaflka1238dsfj7")
	if err != nil {
		t.Error("Test failed - expected an error due to wrong orderID input", err)
	}
}

func TestQueryOrder(t *testing.T) {
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrder(cp.Lower().String(), "1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQueryOrderHistory(t *testing.T) {
	TestSetup(t)
	l.Verbose = true
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrderHistory(cp.Lower().String(), "1", "50")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPairInfo(t *testing.T) {
	TestSetup(t)
	l.Verbose = true
	_, err := l.GetPairInfo()
	if err != nil {
		t.Error("somethings wrong")
	}
}

func TestGetOpeningOrders(t *testing.T) {
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.GetOpeningOrders(cp.Lower().String(), "1", "50")
	if err != nil {
		t.Error("unexpected error")
	}
}

func TestUSD2RMBRate(t *testing.T) {
	TestSetup(t)
	l.Verbose = true
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

func TestGetWithdrawRecords(t *testing.T) {
	TestSetup(t)
	l.Verbose = true
	_, err := l.GetWithdrawlRecords("eth", "1")
}

func TestLoadPrivKey(t *testing.T) {
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
	l.SetDefaults()
	l.APISecret = testAPISecret
	l.loadPrivKey()
	a, err := l.sign("wtf", l.privateKey)
	fmt.Printf(a)
	if err != nil {
		t.Error(err)
	}
}
