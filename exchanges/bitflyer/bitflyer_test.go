package bitflyer

import (
	"log"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey    = ""
	testAPISecret = ""
)

var b Bitflyer

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bitflyerConfig, err := cfg.GetExchangeConfig("Bitflyer")
	if err != nil {
		t.Error("Test Failed - bitflyer Setup() init error")
	}

	bitflyerConfig.AuthenticatedAPISupport = true
	bitflyerConfig.APIKey = testAPIKey
	bitflyerConfig.APISecret = testAPISecret

	b.Setup(bitflyerConfig)
}

func TestGetLatestBlockCA(t *testing.T) {
	t.Parallel()
	_, err := b.GetLatestBlockCA()
	if err != nil {
		t.Error("test failed - Bitflyer - GetLatestBlockCA() error:", err)
	}
}

func TestGetBlockCA(t *testing.T) {
	t.Parallel()
	_, err := b.GetBlockCA("000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f")
	if err != nil {
		t.Error("test failed - Bitflyer - GetBlockCA() error:", err)
	}
}

func TestGetBlockbyHeightCA(t *testing.T) {
	t.Parallel()
	_, err := b.GetBlockbyHeightCA(0)
	if err != nil {
		t.Error("test failed - Bitflyer - GetBlockbyHeightCA() error:", err)
	}
}

func TestGetTransactionByHashCA(t *testing.T) {
	t.Parallel()
	_, err := b.GetTransactionByHashCA("0562d1f063cd4127053d838b165630445af5e480ceb24e1fd9ecea52903cb772")
	if err != nil {
		t.Error("test failed - Bitflyer - GetTransactionByHashCA() error:", err)
	}
}

func TestGetAddressInfoCA(t *testing.T) {
	t.Parallel()
	v, err := b.GetAddressInfoCA("1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB")
	if err != nil {
		t.Error("test failed - Bitflyer - GetAddressInfoCA() error:", err)
	}
	if v.UnconfirmedBalance == 0 || v.ConfirmedBalance == 0 {
		log.Println("WARNING!: Donation wallet is empty :( - please consider donating")
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets()
	if err != nil {
		t.Error("test failed - Bitflyer - GetMarkets() error:", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook("BTC_JPY")
	if err != nil {
		t.Error("test failed - Bitflyer - GetOrderBook() error:", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker("BTC_JPY")
	if err != nil {
		t.Error("test failed - Bitflyer - GetTicker() error:", err)
	}
}

func TestGetExecutionHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetExecutionHistory("BTC_JPY")
	if err != nil {
		t.Error("test failed - Bitflyer - GetExecutionHistory() error:", err)
	}
}

func TestGetExchangeStatus(t *testing.T) {
	t.Parallel()
	_, err := b.GetExchangeStatus()
	if err != nil {
		t.Error("test failed - Bitflyer - GetExchangeStatus() error:", err)
	}
}

// func TestGetChats(t *testing.T) {
// 	t.Parallel()
// 	time := time.Now().Format(time.RFC3339)
// 	_, err := b.GetChats(time)
// 	if err != nil {
// 		t.Error("test failed - Bitflyer - GetChats() error:", err)
// 	}
// }

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	p := pair.NewCurrencyPairFromString("BTC_JPY")
	_, err := b.UpdateTicker(p, "SPOT")
	if err != nil {
		t.Error("test failed - Bitflyer - UpdateTicker() error:", err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	p := pair.NewCurrencyPairFromString("BTC_JPY")
	_, err := b.UpdateOrderbook(p, "SPOT")
	if err != nil {
		t.Error("test failed - Bitflyer - UpdateOrderbook() error:", err)
	}
}

func TestCheckFXString(t *testing.T) {
	t.Parallel()
	p := pair.NewCurrencyPairDelimiter("FXBTC_JPY", "_")
	p = b.CheckFXString(p)
	if p.GetFirstCurrency().String() != "FX_BTC" {
		t.Error("test failed - Bitflyer - CheckFXString() error")
	}
}

func TestGetTickerPrice(t *testing.T) {
	t.Parallel()
	var p pair.CurrencyPair

	currencies := b.GetAvailableCurrencies()
	for _, pair := range currencies {
		if pair.Pair().String() == "FXBTC_JPY" {
			p = pair
			break
		}
	}

	_, err := b.GetTickerPrice(p, b.AssetTypes[0])
	if err != nil {
		t.Error("test failed - Bitflyer - GetTickerPrice() error", err)
	}
}
