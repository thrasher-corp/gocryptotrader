package base

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
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
}

func TestGetOrderbook(t *testing.T) {
	v := b.GetOrderbook("ANX")
	if v != "" {
		t.Error("test failed - base GetOrderbook() error")
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

type CommunicationProvider struct {
	ICommunicate

	isEnabled       bool
	isConnected     bool
	ConnectCalled   bool
	PushEventCalled bool
}

func (p *CommunicationProvider) IsEnabled() bool {
	return p.isEnabled
}

func (p *CommunicationProvider) IsConnected() bool {
	return p.isConnected
}

func (p *CommunicationProvider) Connect() error {
	p.ConnectCalled = true
	return nil
}

func (p *CommunicationProvider) PushEvent(e Event) error {
	p.PushEventCalled = true
	return nil
}

func (p *CommunicationProvider) GetName() string {
	return "someTestProvider"
}

func TestSetup(t *testing.T) {
	var ic IComm
	testConfigs := []struct {
		isEnabled          bool
		isConnected        bool
		shouldConnectCaled bool
		provider           ICommunicate
	}{
		{false, true, false, nil},
		{false, false, false, nil},
		{true, true, false, nil},
		{true, false, true, nil},
	}
	for _, config := range testConfigs {
		config.provider = &CommunicationProvider{
			isEnabled:   config.isEnabled,
			isConnected: config.isConnected}
		ic = append(ic, config.provider)
	}

	ic.Setup()

	for idx, provider := range ic {
		exp := testConfigs[idx].shouldConnectCaled
		act := provider.(*CommunicationProvider).ConnectCalled
		if exp != act {
			t.Fatalf("provider should be enabled and not be connected: exp=%v, act=%v", exp, act)
		}
	}
}

func TestPushEvent(t *testing.T) {
	var ic IComm
	testConfigs := []struct {
		Enabled        bool
		Connected      bool
		PushEventCaled bool
		provider       ICommunicate
	}{
		{false, true, false, nil},
		{false, false, false, nil},
		{true, false, false, nil},
		{true, true, true, nil},
	}
	for _, config := range testConfigs {
		config.provider = &CommunicationProvider{
			isEnabled:   config.Enabled,
			isConnected: config.Connected}
		ic = append(ic, config.provider)
	}

	ic.PushEvent(Event{})

	for idx, provider := range ic {
		exp := testConfigs[idx].PushEventCaled
		act := provider.(*CommunicationProvider).PushEventCalled
		if exp != act {
			t.Fatalf("provider should be enabled and connected: exp=%v, act=%v", exp, act)
		}
	}
}

func TestStageTickerData(t *testing.T) {
	_, ok := TickerStaged["bitstamp"]["someAsset"]["BTCUSD"]
	if ok {
		t.Fatalf("key should not exists")
	}

	price := ticker.Price{}
	var i IComm
	i.Setup()

	i.StageTickerData("bitstamp", "someAsset", &price)

	_, ok = TickerStaged["bitstamp"]["someAsset"][price.Pair.String()]
	if !ok {
		t.Fatalf("key should exists")
	}
}

func TestOrderbookData(t *testing.T) {
	_, ok := OrderbookStaged["bitstamp"]["someAsset"]["someOrderbook"]
	if ok {
		t.Fatal("key should not exists")
	}

	ob := orderbook.Base{
		Asks: []orderbook.Item{
			{1, 2, 3},
			{4, 5, 6},
		},
	}
	var i IComm
	i.Setup()

	i.StageOrderbookData("bitstamp", "someAsset", &ob)

	orderbook, ok := OrderbookStaged["bitstamp"]["someAsset"][ob.Pair.String()]
	if !ok {
		t.Fatal("key should exists")
	}

	if ob.Pair.String() != orderbook.CurrencyPair {
		t.Fatal("currency missmatched")
	}

	_, totalAsks := ob.TotalAsksAmount()
	if totalAsks != orderbook.TotalAsks {
		t.Fatal("total asks missmatched")
	}

	_, totalBids := ob.TotalBidsAmount()
	if totalBids != orderbook.TotalBids {
		t.Fatal("total bids missmatched")
	}
}
