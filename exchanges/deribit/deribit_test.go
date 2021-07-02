package deribit

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

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

func TestGetFundingRateValue(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetFundingRateValue("BTC-PERPETUAL", time.Now().Add(-time.Hour*8), time.Now())
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

// func TestGetHistoricalVolatility(t *testing.T) {
// 	t.Parallel()
// 	d.Verbose = true
// 	data, err := d.GetHistoricalVolatility("BTC-PERPETUAL", time.Now().Add(-time.Hour*8), time.Now())
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	fmt.Println(data)
// }

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetIndexPrice("btc_usd")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetIndexPriceNames()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetInstrumentData(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetInstrumentData("BTC-25MAR22")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetInstrumentsData(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetInstrumentsData("BTC", "", false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetLastSettlementsByCurrency("BTC", "", "", 0, time.Time{})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetLastSettlementsByInstrument("BTC-25MAR22", "", "", 0, time.Time{})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetLastTradesByCurrency("BTC", "", "", "", "", 0, false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetLastTradesByCurrencyAndTime("BTC", "", "", 0, false,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetLastTradesByInstrument("BTC-25MAR22", "", "", "", "", 0, false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetLastTradesByInstrumentAndTime("BTC-25MAR22", "", "", 0, false,
		time.Now().Add(-8*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetOrderbookData(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetOrderbookData("BTC-25MAR22", 0)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}

func TestGetTradeVolumes(t *testing.T) {
	t.Parallel()
	d.Verbose = true
	data, err := d.GetTradeVolumes(false)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)
}
