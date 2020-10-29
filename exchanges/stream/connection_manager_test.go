package stream

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

var (
	errGenerateSubs  = errors.New("failed to generate subs")
	errGenerateConns = errors.New("failed to generate connections")

	passConnectionFunc     = func(c Connection) error { return nil }
	failConnectionFunc     = func(c Connection) error { return errors.New("fail") }
	passGenerateConnection = func(_ string, _ bool) (Connection, error) { return &WebsocketConnection{}, nil }
	failGenerateConnection = func(_ string, _ bool) (Connection, error) { return nil, errGenerateConns }

	passGenerateSubscriptions = func(_ SubscriptionOptions) ([]ChannelSubscription, error) {
		return []ChannelSubscription{
			{
				Channel: "Test Subscription",
			},
		}, nil
	}

	passGenerateSubscriptionsTwoSubs = func(_ SubscriptionOptions) ([]ChannelSubscription, error) {
		return []ChannelSubscription{
			{
				Channel: "Test Subscription",
			},
			{
				Channel: "Test Subscription 2",
			},
		}, nil
	}

	failGenerateSubscriptions = func(_ SubscriptionOptions) ([]ChannelSubscription, error) {
		return nil, errGenerateSubs
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
	_, err = NewConnectionManager(&ConnectionManagerConfig{
		ExchangeConnector:             passConnectionFunc,
		ExchangeGenerateSubscriptions: passGenerateSubscriptions,
		ExchangeGenerateConnection:    passGenerateConnection,
		Features:                      &protocol.Features{},
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	_, err = NewConnectionManager(&ConnectionManagerConfig{
		ExchangeConnector:             passConnectionFunc,
		ExchangeGenerateSubscriptions: passGenerateSubscriptions,
		ExchangeGenerateConnection:    passGenerateConnection,
		Features:                      &protocol.Features{},
		Configurations: []ConnectionSetup{
			{},
		},
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	_, err = NewConnectionManager(&ConnectionManagerConfig{
		ExchangeConnector:             passConnectionFunc,
		ExchangeGenerateSubscriptions: passGenerateSubscriptions,
		ExchangeGenerateConnection:    passGenerateConnection,
		Features:                      &protocol.Features{},
		Configurations: []ConnectionSetup{
			{
				URL: "TEST URL",
			},
		},
	})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	conManager, err := NewConnectionManager(&ConnectionManagerConfig{
		ExchangeConnector:             passConnectionFunc,
		ExchangeGenerateSubscriptions: passGenerateSubscriptions,
		ExchangeGenerateConnection:    passGenerateConnection,
		Features:                      &protocol.Features{},
		Configurations: []ConnectionSetup{
			{
				URL: "TEST URL",
			},
		},
		// AuthConfiguration: ConnectionSetup{URL: "TEST URL"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if conManager == nil {
		t.Fatal("connection manager should not be nil")
	}
}

var (
	blankConfig     = []ConnectionSetup{{}}
	OneConfigNoMax  = []ConnectionSetup{{URL: "TEST URL"}}
	OneConfigMaxSub = []ConnectionSetup{{URL: "TEST URL", MaxSubscriptions: 1}}
)

func TestGenerateConnections(t *testing.T) {
	tests := []struct {
		Name                  string
		GenerateSubscriptions func(SubscriptionOptions) ([]ChannelSubscription, error)
		GenerateConnection    func(url string, auth bool) (Connection, error)
		ConfigurationSet      []ConnectionSetup
		ConnectionCount       int
		SubscriptionCount     int
		Error                 error
	}{
		{
			Name:  "No Subscription Generator Function ",
			Error: errNoGenerateSubsFunc,
		},
		{
			Name:                  "No Conn Generator Function",
			GenerateSubscriptions: passGenerateSubscriptions,
			Error:                 errNoGenerateConnFunc,
		},
		{
			Name:                  "No Configurations",
			GenerateSubscriptions: passGenerateSubscriptions,
			GenerateConnection:    passGenerateConnection,
			Error:                 errNoConfigurations,
		},
		{
			Name:                  "Missing Connection URL",
			GenerateSubscriptions: passGenerateSubscriptions,
			GenerateConnection:    passGenerateConnection,
			ConfigurationSet:      blankConfig,
			Error:                 errMissingURLInConfig,
		},
		{
			Name:                  "Dodgy Generate Subscriptions",
			GenerateSubscriptions: failGenerateSubscriptions,
			GenerateConnection:    passGenerateConnection,
			ConfigurationSet:      OneConfigNoMax,
			Error:                 errGenerateSubs,
		},
		{
			Name:                  "Dodgy Generate Connections",
			GenerateSubscriptions: passGenerateSubscriptions,
			GenerateConnection:    failGenerateConnection,
			ConfigurationSet:      OneConfigNoMax,
			Error:                 errGenerateConns,
		},
		{
			Name:                  "Dodgy Generate Connections Subscriptions Exceed Max",
			GenerateSubscriptions: passGenerateSubscriptionsTwoSubs,
			GenerateConnection:    failGenerateConnection,
			ConfigurationSet:      OneConfigMaxSub,
			Error:                 errGenerateConns,
		},
		{
			Name:                  "Generate Connection based on subscriptions with no max",
			GenerateSubscriptions: passGenerateSubscriptionsTwoSubs,
			GenerateConnection:    passGenerateConnection,
			ConfigurationSet:      OneConfigNoMax,
			ConnectionCount:       1,
			SubscriptionCount:     2,
			Error:                 nil,
		},
		{
			Name:                  "Generate Connection based on subscriptions with max",
			GenerateSubscriptions: passGenerateSubscriptionsTwoSubs,
			GenerateConnection:    passGenerateConnection,
			ConfigurationSet:      OneConfigMaxSub,
			ConnectionCount:       2,
			SubscriptionCount:     2,
			Error:                 nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			man, err := NewConnectionManager(&ConnectionManagerConfig{
				ExchangeConnector:             passConnectionFunc,
				ExchangeGenerateSubscriptions: tt.GenerateSubscriptions,
				ExchangeGenerateConnection:    tt.GenerateConnection,
				Configurations:                tt.ConfigurationSet,
				Features:                      &protocol.Features{},
			})
			if err != nil {
				if !errors.Is(err, tt.Error) {
					t.Fatalf("expecting error [%v] but received [%v]", tt.Error, err)
				}
				return
			}
			subs, err := man.GenerateSubscriptions()
			if err != nil {
				if !errors.Is(err, tt.Error) {
					t.Fatalf("expecting error [%v] but received [%v]", tt.Error, err)
				}
				return
			}

			m, err := man.GenerateConnections(subs)
			if err != nil {
				if !errors.Is(err, tt.Error) {
					t.Fatalf("expecting error [%v] but received [%v]", tt.Error, err)
				}
				return
			}

			if !errors.Is(err, tt.Error) {
				if !errors.Is(err, tt.Error) {
					t.Fatalf("expecting error [%v] but received [%v]", tt.Error, err)
				}
				return
			}

			if len(m) != tt.ConnectionCount {
				t.Fatalf("expecting [%d] connections but received [%d]", tt.ConnectionCount, len(m))
			}

			var subCount int
			for _, v := range m {
				subCount += len(v)
			}

			if subCount != tt.SubscriptionCount {
				t.Fatalf("expecting [%d] subscriptions but received [%d]", tt.SubscriptionCount, subCount)
			}
		})
	}

	// conManager, err := NewConnectionManager(&cfg)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// subs, err := conManager.GenerateSubscriptions()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// m, err := conManager.GenerateConnections(false, subs)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// fmt.Println(m)
}
