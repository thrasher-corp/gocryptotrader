package stream

import (
	"errors"
	"fmt"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

var (
	passConnectionFunc     = func(c Connection) error { return nil }
	failConnectionFunc     = func(c Connection) error { return errors.New("fail") }
	passGenerateConnection = func(c ConnectionSetup, sub []ChannelSubscription) ([]Connection, error) {
		return []Connection{
			Connection(&WebsocketConnection{}),
		}, nil
	}
	passGenerateSubscriptions = func(_ SubscriptionOptions) ([]ChannelSubscription, error) {
		return []ChannelSubscription{
			{
				Channel: "Test Subscription",
			},
		}, nil
	}
)

func TestNewConnectionManager(t *testing.T) {
	_, err := NewConnectionManager(nil)
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	_, err = NewConnectionManager(&ConnectionManagerConfig{
		ExchangeConnector: passConnectionFunc,
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	_, err = NewConnectionManager(&ConnectionManagerConfig{
		ExchangeConnector:          passConnectionFunc,
		ExchangeGenerateConnection: passGenerateConnection,
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	_, err = NewConnectionManager(&ConnectionManagerConfig{
		ExchangeConnector:             passConnectionFunc,
		ExchangeGenerateSubscriptions: passGenerateSubscriptions,
		ExchangeGenerateConnection:    passGenerateConnection,
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	conManager, err := NewConnectionManager(&ConnectionManagerConfig{
		ExchangeConnector:             passConnectionFunc,
		ExchangeGenerateSubscriptions: passGenerateSubscriptions,
		ExchangeGenerateConnection:    passGenerateConnection,
		Features:                      &protocol.Features{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if conManager == nil {
		t.Fatal("connection manager should not be nil")
	}
}

func TestGenerateConnections(t *testing.T) {
	conManager, err := NewConnectionManager(&ConnectionManagerConfig{
		ExchangeConnector:          passConnectionFunc,
		ExchangeGenerateConnection: passGenerateConnection,
		Features:                   &protocol.Features{},
	})
	if err != nil {
		t.Fatal(err)
	}

	subs, err := conManager.GenerateSubscriptions()
	if err != nil {
		t.Fatal(err)
	}

	m, err := conManager.GenerateConnections(false, subs)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(m)
}
