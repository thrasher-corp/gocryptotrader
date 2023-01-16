package cryptodotcom

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var cr Cryptodotcom

func TestMain(m *testing.M) {
	cr.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Cryptodotcom")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = cr.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(Cryptodotcom); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func areTestAPIKeysSet() bool {
	return cr.ValidateAPICredentials(cr.GetDefaultCredentials()) == nil
}

// Implement tests for API endpoints below

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := cr.GetSymbols(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTickersInAllAvailableMarkets(t *testing.T) {
	t.Parallel()
	_, err := cr.GetTickersInAllAvailableMarkets(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTickerForParticularMarket(t *testing.T) {
	t.Parallel()
	pairs, err := cr.GetSymbols(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = cr.GetTickerForParticularMarket(context.Background(), pairs[0].Symbol)
	if err != nil {
		t.Error(err)
	}
}

func TestGetKlineDataOverSpecifiedPeriod(t *testing.T) {
	t.Parallel()
	pairs, err := cr.GetSymbols(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = cr.GetKlineDataOverSpecifiedPeriod(context.Background(), kline.FiveMin, pairs[0].Symbol)
	if err != nil {
		t.Error(err)
	}
}
