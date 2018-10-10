package btcc

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply your own APIkeys here to do better tests
const (
	apiKey    = ""
	apiSecret = ""
)

var b BTCC

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("BTCC")
	if err != nil {
		t.Error("Test Failed - BTCC Setup() init error")
	}
	b.Setup(bConfig)

	if !b.IsEnabled() || b.AuthenticatedAPISupport ||
		b.RESTPollingDelay != time.Duration(10) || b.Verbose ||
		b.Websocket.IsEnabled() || len(b.BaseCurrencies) < 1 ||
		len(b.AvailablePairs) < 1 || len(b.EnabledPairs) < 1 {
		t.Error("Test Failed - BTCC Setup values not set correctly")
	}
}

// func TestGetTicker(t *testing.T) {
// 	t.Skip()
// 	_, err := b.GetTicker("BTCUSD")
// 	if err != nil {
// 		t.Error("Test failed - GetTicker() error", err)
// 	}
// }

// func TestGetTradeHistory(t *testing.T) {
// 	t.Skip()
// 	_, err := b.GetTradeHistory("BTCUSD", 0, 0, time.Time{})
// 	if err != nil {
// 		t.Error("Test failed - GetTradeHistory() error", err)
// 	}
// }

// func TestGetOrderBook(t *testing.T) {
// 	t.Skip()
// 	_, err := b.GetOrderBook("BTCUSD", 100)
// 	if err != nil {
// 		t.Error("Test failed - GetOrderBook() error", err)
// 	}
// 	_, err = b.GetOrderBook("BTCUSD", 0)
// 	if err != nil {
// 		t.Error("Test failed - GetOrderBook() error", err)
// 	}
// }

// func TestGetAccountInfo(t *testing.T) {
// 	t.Skip()
// 	err := b.GetAccountInfo("")
// 	if err == nil {
// 		t.Error("Test failed - GetAccountInfo() error", err)
// 	}
// }
func TestGetFee(t *testing.T) {
	t.Parallel()
	b.SetDefaults()
	TestSetup(t)

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 1, 1, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC, 100, 100, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 10000000000, -1000000000, true, true); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 1, 1, true, false); resp != float64(0.00000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.00000), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 1, 1, false, true); resp != float64(0.0000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.00000), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 10000000000, -1000000000, false, true); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyWithdrawalFee, symbol.BTC, 1, 5, false, false); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.001), resp)
	}

	if resp, err := b.GetFee(exchange.CyptocurrencyDepositFee, symbol.BTC, 1, 0.001, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.CyptocurrencyDepositFee, symbol.BTC, 1, 555, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.InternationalBankDepositFee, symbol.BTC, 1, 1, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.InternationalBankWithdrawalFee, symbol.HKD, 1, 1, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

}
