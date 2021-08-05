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
	r, err := by.GetAllPairs()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(r)
}

func TestGetOrderBook(t *testing.T) {
	r, err := by.GetOrderBook("BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(r)
}

func TestGetTrades(t *testing.T) {
	r, err := by.GetTrades("BTCUSDT", 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(r)
}

func TestGetKlines(t *testing.T) {
	r, err := by.GetKlines("BTCUSDT", "5m", 2000, time.Now().Add(-time.Hour*1), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(r)
}
