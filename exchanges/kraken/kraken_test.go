package kraken

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var k Kraken

// Please add your own APIkeys to do correct due diligence testing.
const (
	apiKey    = ""
	apiSecret = ""
	clientID  = ""
)

func TestSetDefaults(t *testing.T) {
	k.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	krakenConfig, err := cfg.GetExchangeConfig("Kraken")
	if err != nil {
		t.Error("Test Failed - kraken Setup() init error", err)
	}

	krakenConfig.AuthenticatedAPISupport = true
	krakenConfig.APIKey = apiKey
	krakenConfig.APISecret = apiSecret
	krakenConfig.ClientID = clientID

	k.Setup(krakenConfig)
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if k.GetFee(true) != 0.1 {
		t.Error("Test Failed - kraken GetFee() error")
	}
	if k.GetFee(false) != 0.35 {
		t.Error("Test Failed - kraken GetFee() error")
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := k.GetServerTime(false)
	if err != nil {
		t.Error("Test Failed - GetServerTime() error", err)
	}
	_, err = k.GetServerTime(true)
	if err != nil {
		t.Error("Test Failed - GetServerTime() error", err)
	}
}

func TestGetAssets(t *testing.T) {
	t.Parallel()
	_, err := k.GetAssets()
	if err != nil {
		t.Error("Test Failed - GetAssets() error", err)
	}
}

func TestGetAssetPairs(t *testing.T) {
	t.Parallel()
	_, err := k.GetAssetPairs()
	if err != nil {
		t.Error("Test Failed - GetAssetPairs() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := k.GetTicker("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOHLC(t *testing.T) {
	t.Parallel()
	_, err := k.GetOHLC("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetOHLC() error", err)
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := k.GetDepth("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetDepth() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := k.GetTrades("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetSpread(t *testing.T) {
	t.Parallel()
	_, err := k.GetSpread("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetSpread() error", err)
	}
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	_, err := k.GetBalance()
	if err == nil {
		t.Error("Test Failed - GetBalance() error", err)
	}
}

func TestGetTradeBalance(t *testing.T) {
	t.Parallel()
	_, err := k.GetTradeBalance("", "")
	if err == nil {
		t.Error("Test Failed - GetTradeBalance() error", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := k.GetOpenOrders(true, 0)
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error", err)
	}
}

func TestGetClosedOrders(t *testing.T) {
	t.Parallel()
	_, err := k.GetClosedOrders(true, 0, 0, 0, 0, "")
	if err == nil {
		t.Error("Test Failed - GetClosedOrders() error", err)
	}
}

func TestQueryOrdersInfo(t *testing.T) {
	t.Parallel()
	_, err := k.QueryOrdersInfo(false, 0, 0)
	if err == nil {
		t.Error("Test Failed - QueryOrdersInfo() error", err)
	}
}

func TestGetTradesHistory(t *testing.T) {
	t.Parallel()
	_, err := k.GetTradesHistory("", false, 0, 0, 0)
	if err == nil {
		t.Error("Test Failed - GetTradesHistory() error", err)
	}
}

func TestQueryTrades(t *testing.T) {
	t.Parallel()
	_, err := k.QueryTrades(0, false)
	if err == nil {
		t.Error("Test Failed - QueryTrades() error", err)
	}
}

func TestOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := k.OpenPositions(0, false)
	if err == nil {
		t.Error("Test Failed - OpenPositions() error", err)
	}
}

func TestGetLedgers(t *testing.T) {
	t.Parallel()
	_, err := k.GetLedgers("bla", "bla", "bla", 0, 0, 0)
	if err == nil {
		t.Error("Test Failed - GetLedgers() error", err)
	}
}

func TestQueryLedgers(t *testing.T) {
	t.Parallel()
	_, err := k.QueryLedgers("1337")
	if err == nil {
		t.Error("Test Failed - QueryLedgers() error", err)
	}
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := k.GetTradeVolume("BCHEUR")
	if err == nil {
		t.Error("Test Failed - GetTradeVolume() error", err)
	}
}

func TestAddOrder(t *testing.T) {
	t.Parallel()
	_, err := k.AddOrder("bla", "bla", "bla", 0, 0, 0, 0, 0)
	if err == nil {
		t.Error("Test Failed - AddOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := k.CancelOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelOrder() error", err)
	}
}
