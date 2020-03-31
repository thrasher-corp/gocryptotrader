package ftx

import (
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

var f Ftx

func TestMain(m *testing.M) {
	f.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Ftx")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = f.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return f.ValidateAPICredentials()
}

// Implement tests for API endpoints below

func TestGetMarkets(t *testing.T) {
	a, err := f.GetMarkets()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarket(t *testing.T) {
	a, err := f.GetMarket("FTT/BTC")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	a, err := f.GetOrderbook("FTT/BTC", 5)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	a, err := f.GetTrades("FTT/BTC", "10234032", "5234343433", 5)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricalData(t *testing.T) {
	a, err := f.GetHistoricalData("FTT/BTC", "86400", "5", "", "")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFutures(t *testing.T) {
	a, err := f.GetFutures()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuture(t *testing.T) {
	a, err := f.GetFuture("LEO-0327")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFutureStats(t *testing.T) {
	a, err := f.GetFutureStats("LEO-0327")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRates(t *testing.T) {
	a, err := f.GetFundingRates()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	a, err := f.GetAccountInfo()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPositions(t *testing.T) {
	a, err := f.GetPositions()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestChangeAccountLeverage(t *testing.T) {
	f.Verbose = true
	err := f.ChangeAccountLeverage(50)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCoins(t *testing.T) {
	a, err := f.GetCoins()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBalances(t *testing.T) {
	f.Verbose = true
	a, err := f.GetBalances()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllWalletBalances(t *testing.T) {
	f.Verbose = true
	a, err := f.GetAllWalletBalances()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchDepositAddress(t *testing.T) {
	f.Verbose = true
	a, err := f.FetchDepositAddress("TUSD")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchDepositHistory(t *testing.T) {
	f.Verbose = true
	a, err := f.FetchDepositHistory()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchWithdrawalHistory(t *testing.T) {
	f.Verbose = true
	a, err := f.FetchWithdrawalHistory()
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdraw(t *testing.T) {
	a, err := f.Withdraw("BTC", "38eyTMFHvo5UjPR91zwYYKuCtdF2uhtdxS", "", "", "642606", 0.01)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestOpenOrders(t *testing.T) {
	a, err := f.GetOpenOrders("FTT/BTC")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderHistory(t *testing.T) {
	a, err := f.FetchOrderHistory("FTT/BTC", "", "", "2")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenTriggerOrders(t *testing.T) {
	a, err := f.GetOpenTriggerOrders("FTT/BTC", "")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTriggerOrderTriggers(t *testing.T) {
	a, err := f.GetTriggerOrderTriggers("alkdjfkajdsf")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTriggerOrderHistory(t *testing.T) {
	a, err := f.GetTriggerOrderHistory("FTT/BTC")
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}
