package dydx

import (
	"context"
	"encoding/json"
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

var dy DYDX

func TestMain(m *testing.M) {
	dy.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("DYDX")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = dy.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(DYDX); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func areTestAPIKeysSet() bool {
	return dy.ValidateAPICredentials(dy.GetDefaultCredentials()) == nil
}

// Implement tests for API endpoints below

var instrumentJSON = `{	"markets": {	  "LINK-USD": {	  "market": "LINK-USD",	  "status": "ONLINE",	  "baseAsset": "LINK",	  "quoteAsset": "USD",	  "stepSize": "0.1",	  "tickSize": "0.01",	  "indexPrice": "12",	  "oraclePrice": "101",	  "priceChange24H": "0",	  "nextFundingRate": "0.0000125000",	  "nextFundingAt": "2021-03-01T18:00:00.000Z",	  "minOrderSize": "1",	  "type": "PERPETUAL",	  "initialMarginFraction": "0.10",	  "maintenanceMarginFraction": "0.05",	  "baselinePositionSize": "1000",	  "incrementalPositionSize": "1000",	  "incrementalInitialMarginFraction": "0.2",	  "volume24H": "0",	  "trades24H": "0",	  "openInterest": "0",	  "maxPositionSize": "10000",	  "assetResolution": "10000000",	  "syntheticAssetId": "0x4c494e4b2d37000000000000000000"	}	}}`

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	var instrumentData InstrumentDatas
	err := json.Unmarshal([]byte(instrumentJSON), &instrumentData)
	if err != nil {
		t.Error(err)
	}
	if _, err := dy.GetMarkets(context.Background(), ""); err != nil {
		t.Error(err)
	}
}

func TestGetOrderbooks(t *testing.T) {
	t.Parallel()
	if _, err := dy.GetOrderbooks(context.Background(), "CRV-USD"); err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	if result, err := dy.GetTrades(context.Background(), "CRV-USD", time.Now().Add(time.Hour*-1), 5); err != nil {
		t.Error(err)
	} else {
		value, _ := json.Marshal(result)
		println(value)
	}
}
