package okx

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var ok Okx

func TestMain(m *testing.M) {
	ok.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Okx")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = ok.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(Okx); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func areTestAPIKeysSet() bool {
	return ok.ValidateAPICredentials(ok.GetDefaultCredentials()) == nil
}

// Implement tests for API endpoints below

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, er := ok.GetTickers(context.Background(), "", "", "BTC-USD-SWAP")
	if er != nil {
		t.Error("Okx GetTickers() error", er)
	}
}

// TestGetIndexTickers
func TestGetIndexTickers(t *testing.T) {
	t.Parallel()
	_, er := ok.GetIndexTickers(context.Background(), "USDT", "")
	if er != nil {
		t.Error("OKX GetIndexTickers() error", er)
	}
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	_, er := ok.GetOrderBookDepth(context.Background(), currency.NewPair(currency.BTC, currency.USDT), 10)
	if er != nil {
		t.Error("OKX GetOrderBookDepth() error", er)
	}
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, er := ok.GetCandlesticks(context.Background(), "BTC-USDT", kline.OneHour, time.Unix(time.Now().Unix()-3600, 0), time.Now(), 30)
	if er != nil {
		t.Error("Okx GetCandlesticks() error", er)
	}
}

func TestGetCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, er := ok.GetCandlesticksHistory(context.Background(), "BTC-USDT", kline.OneHour, time.Unix(time.Now().Unix()-3600, 0), time.Now(), 30)
	if er != nil {
		t.Error("Okx GetCandlesticksHistory() error", er)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, er := ok.GetTrades(context.Background(), "BTC-USDT", 30)
	if er != nil {
		t.Error("Okx GetTrades() error", er)
	}
}

func TestGet24HTotalVolume(t *testing.T) {
	t.Parallel()
	_, er := ok.Get24HTotalVolume(context.Background())
	if er != nil {
		t.Error("Okx Get24HTotalVolume() error", er)
	}
}

func TestGetOracle(t *testing.T) {
	t.Parallel()
	_, er := ok.GetOracle(context.Background())
	if er != nil {
		t.Error("Okx GetOracle() error", er)
	}
}

func TestGetExchangeRate(t *testing.T) {
	t.Parallel()
	_, er := ok.GetExchangeRate(context.Background())
	if er != nil {
		t.Error("Okx GetExchangeRate() error", er)
	}
}

func TestGetIndexComponents(t *testing.T) {
	t.Parallel()
	_, er := ok.GetIndexComponents(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	if er != nil {
		t.Error("Okx GetIndexComponents() error", er)
	}
}

func TestGetinstrument(t *testing.T) {
	t.Parallel()
	_, er := ok.GetInstruments(context.Background(), &InstrumentsFetchParams{
		InstrumentType: "SPOT",
	})
	if er != nil {
		t.Error("Okx GetInstruments() error", er)
	}
}

func TestGetDeliveryHistory(t *testing.T) {
	t.Parallel()
	_, er := ok.GetDeliveryHistory(context.Background(), "BTC-USDT", "", time.Time{}, time.Time{}, 200)
	if er != nil {
		t.Error("okx GetDeliveryHistory() error", er)
	}
}
