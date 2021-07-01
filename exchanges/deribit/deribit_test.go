package deribit

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var d Deribit

func TestMain(m *testing.M) {
	d.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Deribit")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = d.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return d.ValidateAPICredentials()
}

// Implement tests for API endpoints below

func TestGetBookSummaryByCurrency(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetBookSummaryByCurrency("BTC", "")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetBookSummaryByInstrument("BTC-25MAR22")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetContractSize(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetContractSize("BTC-25MAR22")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetCurrencies()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetFundingChartData(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetFundingChartData("BTC-PERPETUAL", "8h")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}
