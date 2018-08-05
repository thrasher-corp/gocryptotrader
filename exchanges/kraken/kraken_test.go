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
	_, err := k.GetServerTime()
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
	args := TradeBalanceOptions{Asset: "ZEUR"}
	_, err := k.GetTradeBalance(args)
	if err == nil {
		t.Error("Test Failed - GetTradeBalance() error", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	args := OrderInfoOptions{Trades: true}
	_, err := k.GetOpenOrders(args)
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error", err)
	}
}

func TestGetClosedOrders(t *testing.T) {
	t.Parallel()
	args := GetClosedOrdersOptions{Trades: true, Start: "OE4KV4-4FVQ5-V7XGPU"}
	_, err := k.GetClosedOrders(args)
	if err == nil {
		t.Error("Test Failed - GetClosedOrders() error", err)
	}
}

func TestQueryOrdersInfo(t *testing.T) {
	t.Parallel()
	args := OrderInfoOptions{Trades: true}
	_, err := k.QueryOrdersInfo(args, "OR6ZFV-AA6TT-CKFFIW", "OAMUAJ-HLVKG-D3QJ5F")
	if err == nil {
		t.Error("Test Failed - QueryOrdersInfo() error", err)
	}
}

func TestGetTradesHistory(t *testing.T) {
	t.Parallel()
	args := GetTradesHistoryOptions{Trades: true, Start: "TMZEDR-VBJN2-NGY6DX", End: "TVRXG2-R62VE-RWP3UW"}
	_, err := k.GetTradesHistory(args)
	if err == nil {
		t.Error("Test Failed - GetTradesHistory() error", err)
	}
}

func TestQueryTrades(t *testing.T) {
	t.Parallel()
	_, err := k.QueryTrades(true, "TMZEDR-VBJN2-NGY6DX", "TFLWIB-KTT7L-4TWR3L", "TDVRAH-2H6OS-SLSXRX")
	if err == nil {
		t.Error("Test Failed - QueryTrades() error", err)
	}
}

func TestOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := k.OpenPositions(false)
	if err == nil {
		t.Error("Test Failed - OpenPositions() error", err)
	}
}

func TestGetLedgers(t *testing.T) {
	t.Parallel()
	args := GetLedgersOptions{Start: "LRUHXI-IWECY-K4JYGO", End: "L5NIY7-JZQJD-3J4M2V", Ofs: 15}
	_, err := k.GetLedgers(args)
	if err == nil {
		t.Error("Test Failed - GetLedgers() error", err)
	}
}

func TestQueryLedgers(t *testing.T) {
	t.Parallel()
	_, err := k.QueryLedgers("LVTSFS-NHZVM-EXNZ5M")
	if err == nil {
		t.Error("Test Failed - QueryLedgers() error", err)
	}
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := k.GetTradeVolume(true, "OAVY7T-MV5VK-KHDF5X")
	if err == nil {
		t.Error("Test Failed - GetTradeVolume() error", err)
	}
}

func TestAddOrder(t *testing.T) {
	t.Parallel()
	args := AddOrderOptions{Oflags: "fcib"}
	_, err := k.AddOrder("XXBTZUSD", "sell", "market", 0.00000001, 0, 0, 0, args)
	if err == nil {
		t.Error("Test Failed - AddOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := k.CancelOrder("OAVY7T-MV5VK-KHDF5X")
	if err == nil {
		t.Error("Test Failed - CancelOrder() error", err)
	}
}
