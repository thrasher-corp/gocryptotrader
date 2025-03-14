package base

import (
	"testing"
	"time"
)

var b Base

func TestStart(_ *testing.T) {
	b = Base{
		Name:      "test",
		Enabled:   true,
		Verbose:   true,
		Connected: true,
	}
}

func TestIsEnabled(t *testing.T) {
	if !b.IsEnabled() {
		t.Error("base IsEnabled() error")
	}
}

func TestIsConnected(t *testing.T) {
	if !b.IsConnected() {
		t.Error("base IsConnected() error")
	}
}

func TestGetName(t *testing.T) {
	if b.GetName() != "test" {
		t.Error("base GetName() error")
	}
}

func TestSetServiceStarted(t *testing.T) {
	b = Base{}
	tt := time.Now()
	if b.ServiceStarted.Equal(tt) {
		t.Errorf("expected '%v', received '%v'", time.Time{}, tt)
	}
	b.SetServiceStarted(tt)
	if !b.ServiceStarted.Equal(tt) {
		t.Errorf("expected '%v', received '%v'", tt, b.ServiceStarted)
	}
}

type CommunicationProvider struct {
	ICommunicate

	isEnabled        bool
	isConnected      bool
	ConnectCalled    bool
	PushEventCalled  bool
	ServiceStartTime time.Time
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

func (p *CommunicationProvider) PushEvent(_ Event) error {
	p.PushEventCalled = true
	return nil
}

func (p *CommunicationProvider) GetName() string {
	return "someTestProvider"
}

func (p *CommunicationProvider) SetServiceStarted(t time.Time) {
	p.ServiceStartTime = t
}

func TestSetup(t *testing.T) {
	var ic IComm
	testConfigs := []struct {
		isEnabled           bool
		isConnected         bool
		shouldConnectCalled bool
		provider            ICommunicate
	}{
		{false, true, false, nil},
		{false, false, false, nil},
		{true, true, false, nil},
		{true, false, true, nil},
	}
	for _, config := range testConfigs {
		config.provider = &CommunicationProvider{
			isEnabled:   config.isEnabled,
			isConnected: config.isConnected,
		}
		ic = append(ic, config.provider)
	}

	ic.Setup()

	for idx, provider := range ic {
		exp := testConfigs[idx].shouldConnectCalled
		act, ok := provider.(*CommunicationProvider)
		if !ok {
			t.Fatal("unable to type assert provider")
		}
		if exp != act.ConnectCalled {
			t.Fatalf("provider should be enabled and not be connected: exp=%v, act=%v", exp, act)
		}
	}
}

func TestPushEvent(t *testing.T) {
	var ic IComm
	testConfigs := []struct {
		Enabled         bool
		Connected       bool
		PushEventCalled bool
		provider        ICommunicate
	}{
		{false, true, false, nil},
		{false, false, false, nil},
		{true, false, false, nil},
		{true, true, true, nil},
	}
	for _, config := range testConfigs {
		config.provider = &CommunicationProvider{
			isEnabled:   config.Enabled,
			isConnected: config.Connected,
		}
		ic = append(ic, config.provider)
	}

	ic.PushEvent(Event{})

	for idx, provider := range ic {
		exp := testConfigs[idx].PushEventCalled
		act, ok := provider.(*CommunicationProvider)
		if !ok {
			t.Fatal("unable to type assert provider")
		}
		if exp != act.PushEventCalled {
			t.Fatalf("provider should be enabled and connected: exp=%v, act=%v", exp, act)
		}
	}
}
