package base

import (
	"fmt"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

var (
	b Base
	i IComm
)

func TestStart(t *testing.T) {
	b = Base{
		Name:      "test",
		Enabled:   true,
		Verbose:   true,
		Connected: true,
	}
}

func TestIsEnabled(t *testing.T) {
	if !b.IsEnabled() {
		t.Error("test failed - base IsEnabled() error")
	}
}

func TestIsConnected(t *testing.T) {
	if !b.IsConnected() {
		t.Error("test failed - base IsConnected() error")
	}
}

func TestGetName(t *testing.T) {
	if b.GetName() != "test" {
		t.Error("test failed - base GetName() error")
	}
}

func TestGetTicker(t *testing.T) {
	v := b.GetTicker("ANX")
	if v != "" {
		t.Error("test failed - base GetTicker() error")
	}

	exchangeName := "exchange"
	savedTickerStaged := TickerStaged
	defer func() { TickerStaged = savedTickerStaged }()
	TickerStaged = map[string]map[string]map[string]ticker.Price{
		exchangeName: {
			"testAsset": {
				"BTCUSD": ticker.Price{
					Pair:     currency.NewPairFromString("BTCUSD"),
					Ask:      0.000001,
					Bid:      0.000002,
					High:     0.000003,
					Last:     0.000004,
					Low:      0.000005,
					PriceATH: 0.000006,
					Volume:   0.000007,
				},
			},
		},
	}

	exp := "Currency Pair: BTCUSD Ask: 0.000001, Bid: 0.000002 High: 0.000003 Last: 0.000004 Low: 0.000005 ATH: 0.000006 Volume: 0.000007"
	act := b.GetTicker(exchangeName)
	if exp != act {
		t.Errorf("\n'%s'\n is not\n '%s'", act, exp)
	}
}

func TestGetOrderbook(t *testing.T) {
	v := b.GetOrderbook("ANX")
	if v != "" {
		t.Error("test failed - base GetOrderbook() error")
	}

	exchangeName := "exchange"
	lastUpdated := time.Now().String()
	savedOrderbookStaged := OrderbookStaged
	defer func() { OrderbookStaged = savedOrderbookStaged }()
	OrderbookStaged = map[string]map[string]map[string]Orderbook{
		exchangeName: {
			"testAsset": {
				"BTCUSD": Orderbook{
					CurrencyPair: "BTCUSD",
					AssetType:    ticker.Spot,
					TotalAsks:    0.000001,
					TotalBids:    0.000002,
					LastUpdated:  lastUpdated,
				},
			},
		},
	}

	exp := fmt.Sprintf("Currency Pair: BTCUSD AssetType: %s, LastUpdated: %s TotalAsks: 0.000001 TotalBids: 0.000002", ticker.Spot, lastUpdated)
	act := b.GetOrderbook(exchangeName)
	if exp != act {
		t.Errorf("\n'%s'\n is not\n '%s'", act, exp)
	}
}

func TestGetPortfolio(t *testing.T) {
	v := b.GetPortfolio()
	if v != "{}" {
		t.Error("test failed - base GetPortfolio() error")
	}
}

func TestGetSettings(t *testing.T) {
	v := b.GetSettings()
	if v != "{ }" {
		t.Error("test failed - base GetSettings() error")
	}
}

func TestGetStatus(t *testing.T) {
	v := b.GetStatus()
	if v == "" {
		t.Error("test failed - base GetStatus() error")
	}
}

func TestSetup(t *testing.T) {
	i.Setup()
}

func TestPushEvent(t *testing.T) {
	i.PushEvent(Event{})
}

func TestGetEnabledCommunicationMediums(t *testing.T) {
	i.GetEnabledCommunicationMediums()
}
