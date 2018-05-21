package base

import (
	"testing"
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

func TestSetup(t *testing.T) {
	i.Setup()
}

func TestPushEvent(t *testing.T) {
	i.PushEvent(Event{})
}

func TestGetEnabledCommunicationMediums(t *testing.T) {
	i.GetEnabledCommunicationMediums()
}
