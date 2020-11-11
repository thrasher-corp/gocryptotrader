package protocol

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

func TestSetup(t *testing.T) {
	f := &Features{}
	err := f.SetupREST(nil)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = f.SetupREST(&State{})
	if err != nil {
		t.Fatal(err)
	}

	err = f.SetupREST(&State{})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = f.SetupWebsocket(nil)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = f.SetupWebsocket(&State{})
	if err != nil {
		t.Fatal(err)
	}

	err = f.SetupWebsocket(&State{})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
}

func TestSet(t *testing.T) {
	f := &Features{}
	// Setup with no components supported
	err := f.SetupREST(&State{})
	if err != nil {
		t.Fatal(err)
	}
	err = f.SetupWebsocket(&State{})
	if err != nil {
		t.Fatal(err)
	}
	// No change of state
	err = f.SetREST(State{})
	if err != nil {
		t.Fatal(err)
	}
	err = f.SetWebsocket(State{})
	if err != nil {
		t.Fatal(err)
	}
	// state change but not supported
	err = f.SetREST(State{
		AccountBalance: convert.BoolPtrT,
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetREST(State{
		AccountBalance: convert.BoolPtrF,
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetWebsocket(State{
		AccountBalance: convert.BoolPtrT,
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetWebsocket(State{
		AccountBalance: convert.BoolPtrF,
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	// test state change with supported component
	f.REST = nil
	f.Websocket = nil
	err = f.SetupREST(&State{AccountBalance: convert.BoolPtrF})
	if err != nil {
		t.Fatal(err)
	}
	err = f.SetREST(State{
		AccountBalance: convert.BoolPtrT,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = f.SetupWebsocket(&State{TickerBatching: convert.BoolPtrT})
	if err != nil {
		t.Fatal(err)
	}
	err = f.SetWebsocket(State{
		TickerBatching: convert.BoolPtrF,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWithdrawalPermissions(t *testing.T) {
	f := &Features{}
	if f.GetWithdrawalPermissions() != 0 {
		t.Fatal("unexpected value")
	}
	f.SetWithdrawalPermissions(123)
	if f.GetWithdrawalPermissions() != 123 {
		t.Fatal("unexpected value")
	}
}

func TestIsSupported(t *testing.T) {
	f := &Features{}
	if f.IsWebsocketSupported() {
		t.Fatal("websocket should not be supported")
	}
	if f.IsRESTSupported() {
		t.Fatal("rest should not be supported")
	}
	err := f.SetupREST(&State{})
	if err != nil {
		t.Fatal(err)
	}
	if !f.IsRESTSupported() {
		t.Fatal("rest should be supported")
	}
	err = f.SetupWebsocket(&State{})
	if err != nil {
		t.Fatal(err)
	}
	if !f.IsWebsocketSupported() {
		t.Fatal("websocket should be supported")
	}
}

func TestIsEnabled(t *testing.T) {
	f := &Features{}
	_, err := f.IsWebsocketEnabled()
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetupWebsocket(&State{ProtocolEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	enabled, err := f.IsWebsocketEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("websocket should be enabled")
	}

	_, err = f.IsRESTEnabled()
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetupREST(&State{ProtocolEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	enabled, err = f.IsRESTEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("rest should be enabled")
	}
}

func TestCheckAuthentication(t *testing.T) {
	f := &Features{}
	_, err := f.IsRestAuthenticationEnabled()
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetupREST(&State{AuthenticationEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	auth, err := f.IsRestAuthenticationEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !auth {
		t.Fatal("rest auth should be enabled")
	}

	_, err = f.IsWebsocketAuthenticationEnabled()
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetupWebsocket(&State{AuthenticationEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	auth, err = f.IsWebsocketAuthenticationEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !auth {
		t.Fatal("websocket auth should be enabled")
	}
}

func TestCheckComponent(t *testing.T) {
	f := &Features{}
	_, err := f.RESTEnabled(TickerBatching)
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetupREST(&State{TickerBatching: convert.BoolPtrT})
	if err != nil {
		t.Fatal(err)
	}
	enabled, err := f.RESTEnabled(TickerBatching)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("ticker batching on rest should be enabled")
	}

	_, err = f.WebsocketEnabled(TickerBatching)
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetupWebsocket(&State{TickerBatching: convert.BoolPtrF})
	if err != nil {
		t.Fatal(err)
	}
	enabled, err = f.WebsocketEnabled(TickerBatching)
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("ticker batching on websocket should not be enabled")
	}
}

func TestSupports(t *testing.T) {
	f := &Features{}
	_, err := f.RESTSupports(TickerBatching)
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetupREST(&State{TickerBatching: convert.BoolPtrF})
	if err != nil {
		t.Fatal(err)
	}
	supported, err := f.RESTSupports(TickerBatching)
	if err != nil {
		t.Fatal(err)
	}
	if !supported {
		t.Fatal("ticker batching on rest should be supported")
	}
	supported, err = f.RESTSupports(AccountBalance)
	if err != nil {
		t.Fatal(err)
	}
	if supported {
		t.Fatal("account balance on rest should not be supported")
	}

	_, err = f.WebsocketSupports(TickerBatching)
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = f.SetupWebsocket(&State{TickerBatching: convert.BoolPtrT})
	if err != nil {
		t.Fatal(err)
	}
	supported, err = f.WebsocketSupports(TickerBatching)
	if err != nil {
		t.Fatal(err)
	}
	if !supported {
		t.Fatal("ticker batching on websocket should be supported")
	}
	supported, err = f.WebsocketSupports(AccountBalance)
	if err != nil {
		t.Fatal(err)
	}
	if supported {
		t.Fatal("account balance on websocket should not be supported")
	}
}

func TestJSON(t *testing.T) {
	f := &Features{}
	err := f.SetupREST(&State{TickerBatching: convert.BoolPtrF})
	if err != nil {
		t.Fatal(err)
	}

	err = f.SetupWebsocket(&State{AccountInfo: convert.BoolPtrF})
	if err != nil {
		t.Fatal(err)
	}

	payload, err := f.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	newF := &Features{}
	err = newF.UnmarshalJSON(payload)
	if err != nil {
		t.Fatal(err)
	}

	if !newF.IsRESTSupported() || !newF.IsWebsocketSupported() {
		t.Fatal("unexpected values")
	}
}
