package coinbaseinternational

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var co = &CoinbaseInternational{}

func TestMain(m *testing.M) {
	co.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Coinbaseinternational")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = co.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(CoinbaseInternational); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

// Implement tests for API endpoints below

func TestListAssets(t *testing.T) {
	t.Parallel()
	_, err := co.ListAssets(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetAssetDetails(context.Background(), currency.EMPTYCODE, "", "207597618027560960")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSupportedNetworksPerAsset(t *testing.T) {
	t.Parallel()
	_, err := co.GetSupportedNetworksPerAsset(context.Background(), currency.BTC, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	_, err := co.GetInstruments(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstrumentDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetInstrumentDetails(context.Background(), "BTC-PERP", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetQuotePerInstrument(t *testing.T) {
	t.Parallel()
	_, err := co.GetQuotePerInstrument(context.Background(), "BTC-PERP", "", "")
	if err != nil {
		t.Error(err)
	}
}
