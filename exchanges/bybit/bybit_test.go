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
