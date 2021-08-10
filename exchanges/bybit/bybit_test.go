package bybit

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var by Bybit

func TestMain(m *testing.M) {
	by.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Bybit")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = by.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(Bybit); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func areTestAPIKeysSet() bool {
	return by.ValidateAPICredentials()
}

func TestGetAllPairs(t *testing.T) {
	by.Verbose = true
	t.Parallel()

	_, err := by.GetAllPairs()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()

	_, err := by.GetOrderBook("BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()

	_, err := by.GetTrades("BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetKlines(t *testing.T) {
	t.Parallel()

	_, err := by.GetKlines("BTCUSDT", "5m", 2000, time.Now().Add(-time.Hour*1), time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGet24HrsChange(t *testing.T) {
	t.Parallel()

	_, err := by.Get24HrsChange("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.Get24HrsChange("")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetLastTradedPrice(t *testing.T) {
	t.Parallel()

	_, err := by.GetLastTradedPrice("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetLastTradedPrice("")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetBestBidAskPrice(t *testing.T) {
	t.Parallel()

	_, err := by.GetBestBidAskPrice("BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}

	_, err = by.GetBestBidAskPrice("")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreatePostOrder(t *testing.T) {
	t.Parallel()

	r, err := by.CreatePostOrder(&PlaceOrderRequest{
		Symbol:      "BTCUSDT",
		Quantity:    1,
		Side:        "BUY",
		TradeType:   "LIMIT",
		TimeInForce: "GTC",
		Price:       100,
		OrderLinkID: "linkID",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", r)
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()

	r, err := by.QueryOrder("0", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", r)
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	r, err := by.CancelExistingOrder("", "linkID")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", r)
}

func TestBatchCancelOrder(t *testing.T) {
	t.Parallel()

	r, err := by.BatchCancelOrder("", "BUY", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", r)
}

func TestListOpenOrders(t *testing.T) {
	t.Parallel()

	r, err := by.ListOpenOrders("", "BUY", 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", r)
}

func TestListPastOrders(t *testing.T) {
	t.Parallel()

	r, err := by.ListPastOrders("", "BUY", 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", r)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()

	r, err := by.GetTradeHistory("", 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", r)
}

func TestGetWalletBalance(t *testing.T) {
	t.Parallel()

	r, err := by.GetWalletBalance()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", r)
}
