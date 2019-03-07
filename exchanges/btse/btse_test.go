package btse

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey    = ""
	testAPISecret = ""
)

var b BTSE

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	btseConfig, err := cfg.GetExchangeConfig("BTSE")
	if err != nil {
		t.Error("Test Failed - BTSE Setup() init error")
	}

	btseConfig.AuthenticatedAPISupport = true
	btseConfig.APIKey = testAPIKey
	btseConfig.APISecret = testAPISecret

	b.Setup(btseConfig)
}

func TestGetMarkets(t *testing.T) {
	b.SetDefaults()
	_, err := b.GetMarkets()
	if err != nil {
		t.Fatalf("Test failed. Err: %s", err)
	}
}

func TestGetTrades(t *testing.T) {
	b.SetDefaults()
	_, err := b.GetTrades("BTC-USD")
	if err != nil {
		t.Fatalf("Test failed. Err: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	b.SetDefaults()
	_, err := b.GetTicker("BTC-USD")
	if err != nil {
		t.Fatalf("Test failed. Err: %s", err)
	}
}

func TestGetMarketStatistics(t *testing.T) {
	b.SetDefaults()
	_, err := b.GetMarketStatistics("BTC-USD")
	if err != nil {
		t.Fatalf("Test failed. Err: %s", err)
	}
}

func TestGetServerTime(t *testing.T) {
	b.SetDefaults()
	_, err := b.GetServerTime()
	if err != nil {
		t.Fatalf("Test failed. Err: %s", err)
	}
}
