package gdax

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var g GDAX

// Please supply your APIKeys here for better testing
const (
	apiKey    = ""
	apiSecret = ""
	clientID  = "" //passphrase you made at API CREATION
)

func TestSetDefaults(t *testing.T) {
	g.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	gdxConfig, err := cfg.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Error("Test Failed - GDAX Setup() init error")
	}

	g.Setup(gdxConfig)
}

func TestGetFee(t *testing.T) {
	if g.GetFee(false) == 0 {
		t.Error("Test failed - GetFee() error")
	}
	if g.GetFee(true) != 0 {
		t.Error("Test failed - GetFee() error")
	}
}

func TestGetProducts(t *testing.T) {
	_, err := g.GetProducts()
	if err != nil {
		t.Error("Test failed - GetProducts() error")
	}
}

func TestGetTicker(t *testing.T) {
	_, err := g.GetTicker("BTC-USD")
	if err != nil {
		t.Error("Test failed - GetTicker() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := g.GetTrades("BTC-USD")
	if err != nil {
		t.Error("Test failed - GetTrades() error", err)
	}
}

func TestGetHistoricRates(t *testing.T) {
	_, err := g.GetHistoricRates("BTC-USD", 0, 0, 0)
	if err != nil {
		t.Error("Test failed - GetHistoricRates() error", err)
	}
}

func TestGetStats(t *testing.T) {
	_, err := g.GetStats("BTC-USD")
	if err != nil {
		t.Error("Test failed - GetStats() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := g.GetCurrencies()
	if err != nil {
		t.Error("Test failed - GetCurrencies() error", err)
	}
}

func TestGetServerTime(t *testing.T) {
	_, err := g.GetServerTime()
	if err != nil {
		t.Error("Test failed - GetServerTime() error", err)
	}
}

func TestAuthRequests(t *testing.T) {

	if g.APIKey != "" && g.APISecret != "" && g.ClientID != "" {

		_, err := g.GetAccounts()
		if err == nil {
			t.Error("Test failed - GetAccounts() error", err)
		}

		_, err = g.GetAccount("234cb213-ac6f-4ed8-b7b6-e62512930945")
		if err == nil {
			t.Error("Test failed - GetAccount() error", err)
		}

		_, err = g.GetAccountHistory("234cb213-ac6f-4ed8-b7b6-e62512930945")
		if err == nil {
			t.Error("Test failed - GetAccountHistory() error", err)
		}

		_, err = g.GetHolds("234cb213-ac6f-4ed8-b7b6-e62512930945")
		if err == nil {
			t.Error("Test failed - GetHolds() error", err)
		}

		_, err = g.PlaceLimitOrder("", 0, 0, "buy", "", "", "BTC-USD", "", false)
		if err == nil {
			t.Error("Test failed - PlaceLimitOrder() error", err)
		}

		_, err = g.PlaceMarketOrder("", 1, 0, "buy", "BTC-USD", "")
		if err == nil {
			t.Error("Test failed - PlaceMarketOrder() error", err)
		}

		err = g.CancelOrder("1337")
		if err == nil {
			t.Error("Test failed - CancelOrder() error", err)
		}

		_, err = g.CancelAllOrders("BTC-USD")
		if err == nil {
			t.Error("Test failed - CancelAllOrders() error", err)
		}

		_, err = g.GetOrders([]string{"open", "done"}, "BTC-USD")
		if err == nil {
			t.Error("Test failed - GetOrders() error", err)
		}

		_, err = g.GetOrder("1337")
		if err == nil {
			t.Error("Test failed - GetOrders() error", err)
		}

		_, err = g.GetFills("1337", "BTC-USD")
		if err == nil {
			t.Error("Test failed - GetFills() error", err)
		}
		_, err = g.GetFills("", "")
		if err == nil {
			t.Error("Test failed - GetFills() error", err)
		}

		_, err = g.GetFundingRecords("rejected")
		if err == nil {
			t.Error("Test failed - GetFundingRecords() error", err)
		}

		// 	_, err := g.RepayFunding("1", "BTC")
		// 	if err != nil {
		// 		t.Error("Test failed - RepayFunding() error", err)
		// 	}

		_, err = g.MarginTransfer(1, "withdraw", "45fa9e3b-00ba-4631-b907-8a98cbdf21be", "BTC")
		if err == nil {
			t.Error("Test failed - MarginTransfer() error", err)
		}

		_, err = g.GetPosition()
		if err == nil {
			t.Error("Test failed - GetPosition() error", err)
		}

		_, err = g.ClosePosition(false)
		if err == nil {
			t.Error("Test failed - ClosePosition() error", err)
		}

		_, err = g.GetPayMethods()
		if err == nil {
			t.Error("Test failed - GetPayMethods() error", err)
		}

		_, err = g.DepositViaPaymentMethod(1, "BTC", "1337")
		if err == nil {
			t.Error("Test failed - DepositViaPaymentMethod() error", err)
		}

		_, err = g.DepositViaCoinbase(1, "BTC", "1337")
		if err == nil {
			t.Error("Test failed - DepositViaCoinbase() error", err)
		}

		_, err = g.WithdrawViaPaymentMethod(1, "BTC", "1337")
		if err == nil {
			t.Error("Test failed - WithdrawViaPaymentMethod() error", err)
		}

		// 	_, err := g.WithdrawViaCoinbase(1, "BTC", "c13cd0fc-72ca-55e9-843b-b84ef628c198")
		// 	if err != nil {
		// 		t.Error("Test failed - WithdrawViaCoinbase() error", err)
		// 	}

		_, err = g.WithdrawCrypto(1, "BTC", "1337")
		if err == nil {
			t.Error("Test failed - WithdrawViaCoinbase() error", err)
		}

		_, err = g.GetCoinbaseAccounts()
		if err == nil {
			t.Error("Test failed - GetCoinbaseAccounts() error", err)
		}

		_, err = g.GetReportStatus("1337")
		if err == nil {
			t.Error("Test failed - GetReportStatus() error", err)
		}

		_, err = g.GetTrailingVolume()
		if err == nil {
			t.Error("Test failed - GetTrailingVolume() error", err)
		}
	}
}
