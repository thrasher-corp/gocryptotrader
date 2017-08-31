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
	cfg.LoadConfig("../../testdata/configtest.dat")
	gdxConfig, err := cfg.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Error("Test Failed - GDAX Setup() init error")
	}

	g.Setup(gdxConfig)
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if g.GetFee(false) == 0 {
		t.Error("Test failed - GetFee() error")
	}
	if g.GetFee(true) != 0 {
		t.Error("Test failed - GetFee() error")
	}
}

func TestGetProducts(t *testing.T) {
	t.Parallel()
	_, err := g.GetProducts()
	if err != nil {
		t.Error("Test failed - GetProducts() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := g.GetTicker("BTC-USD")
	if err != nil {
		t.Error("Test failed - GetTicker() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := g.GetTrades("BTC-USD")
	if err != nil {
		t.Error("Test failed - GetTrades() error", err)
	}
}

func TestGetHistoricRates(t *testing.T) {
	t.Parallel()
	_, err := g.GetHistoricRates("BTC-USD", 0, 0, 0)
	if err != nil {
		t.Error("Test failed - GetHistoricRates() error", err)
	}
}

func TestGetStats(t *testing.T) {
	t.Parallel()
	_, err := g.GetStats("BTC-USD")
	if err != nil {
		t.Error("Test failed - GetStats() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := g.GetCurrencies()
	if err != nil {
		t.Error("Test failed - GetCurrencies() error", err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := g.GetServerTime()
	if err != nil {
		t.Error("Test failed - GetServerTime() error", err)
	}
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	_, err := g.GetAccounts()
	if err == nil {
		t.Error("Test failed - GetAccounts() error", err)
	}
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	_, err := g.GetAccount("234cb213-ac6f-4ed8-b7b6-e62512930945")
	if err == nil {
		t.Error("Test failed - GetAccount() error", err)
	}
}

func TestGetAccountHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetAccountHistory("234cb213-ac6f-4ed8-b7b6-e62512930945")
	if err == nil {
		t.Error("Test failed - GetAccountHistory() error", err)
	}
}

func TestGetHolds(t *testing.T) {
	t.Parallel()
	_, err := g.GetHolds("234cb213-ac6f-4ed8-b7b6-e62512930945")
	if err == nil {
		t.Error("Test failed - GetHolds() error", err)
	}
}

func TestPlaceLimitOrder(t *testing.T) {
	t.Parallel()
	_, err := g.PlaceLimitOrder("", 0, 0, "buy", "", "", "BTC-USD", "", false)
	if err == nil {
		t.Error("Test failed - PlaceLimitOrder() error", err)
	}
}

func TestPlaceMarketOrder(t *testing.T) {
	t.Parallel()
	_, err := g.PlaceMarketOrder("", 1, 0, "buy", "BTC-USD", "")
	if err == nil {
		t.Error("Test failed - PlaceMarketOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := g.CancelOrder("1337")
	if err == nil {
		t.Error("Test failed - CancelOrder() error", err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := g.CancelAllOrders("BTC-USD")
	if err == nil {
		t.Error("Test failed - CancelAllOrders() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrders([]string{"open", "done"}, "BTC-USD")
	if err == nil {
		t.Error("Test failed - GetOrders() error", err)
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrder("1337")
	if err == nil {
		t.Error("Test failed - GetOrders() error", err)
	}
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	_, err := g.GetFills("1337", "BTC-USD")
	if err == nil {
		t.Error("Test failed - GetFills() error", err)
	}
	_, err = g.GetFills("", "")
	if err == nil {
		t.Error("Test failed - GetFills() error", err)
	}
}

func TestGetFundingRecords(t *testing.T) {
	t.Parallel()
	_, err := g.GetFundingRecords("rejected")
	if err == nil {
		t.Error("Test failed - GetFundingRecords() error", err)
	}
}

// func TestRepayFunding(t *testing.T) {
// 	g.Verbose = true
// 	_, err := g.RepayFunding("1", "BTC")
// 	if err != nil {
// 		t.Error("Test failed - RepayFunding() error", err)
// 	}
// }

func TestMarginTransfer(t *testing.T) { //invalid sig issue
	t.Parallel()
	_, err := g.MarginTransfer(1, "withdraw", "45fa9e3b-00ba-4631-b907-8a98cbdf21be", "BTC")
	if err == nil {
		t.Error("Test failed - MarginTransfer() error", err)
	}
}

func TestGetPosition(t *testing.T) {
	t.Parallel()
	_, err := g.GetPosition()
	if err == nil {
		t.Error("Test failed - GetPosition() error", err)
	}
}

func TestClosePosition(t *testing.T) {
	t.Parallel()
	_, err := g.ClosePosition(false)
	if err == nil {
		t.Error("Test failed - ClosePosition() error", err)
	}
}

func TestGetPayMethods(t *testing.T) {
	t.Parallel()
	_, err := g.GetPayMethods()
	if err == nil {
		t.Error("Test failed - GetPayMethods() error", err)
	}
}

func TestDepositViaPaymentMethod(t *testing.T) {
	t.Parallel()
	_, err := g.DepositViaPaymentMethod(1, "BTC", "1337")
	if err == nil {
		t.Error("Test failed - DepositViaPaymentMethod() error", err)
	}
}

func TestDepositViaCoinbase(t *testing.T) {
	t.Parallel()
	_, err := g.DepositViaCoinbase(1, "BTC", "1337")
	if err == nil {
		t.Error("Test failed - DepositViaCoinbase() error", err)
	}
}

func TestWithdrawViaPaymentMethod(t *testing.T) {
	t.Parallel()
	_, err := g.WithdrawViaPaymentMethod(1, "BTC", "1337")
	if err == nil {
		t.Error("Test failed - WithdrawViaPaymentMethod() error", err)
	}
}

// func TestWithdrawViaCoinbase(t *testing.T) { // No Route found error
// 	_, err := g.WithdrawViaCoinbase(1, "BTC", "c13cd0fc-72ca-55e9-843b-b84ef628c198")
// 	if err != nil {
// 		t.Error("Test failed - WithdrawViaCoinbase() error", err)
// 	}
// }

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := g.WithdrawCrypto(1, "BTC", "1337")
	if err == nil {
		t.Error("Test failed - WithdrawViaCoinbase() error", err)
	}
}

func TestGetCoinbaseAccounts(t *testing.T) {
	t.Parallel()
	_, err := g.GetCoinbaseAccounts()
	if err == nil {
		t.Error("Test failed - GetCoinbaseAccounts() error", err)
	}
}

func TestGetReportStatus(t *testing.T) {
	t.Parallel()
	_, err := g.GetReportStatus("1337")
	if err == nil {
		t.Error("Test failed - GetReportStatus() error", err)
	}
}

func TestGetTrailingVolume(t *testing.T) {
	t.Parallel()
	_, err := g.GetTrailingVolume()
	if err == nil {
		t.Error("Test failed - GetTrailingVolume() error", err)
	}
}
