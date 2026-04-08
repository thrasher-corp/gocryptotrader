package websocket

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

const (
	Ping               = "ping"
	useProxyTests      = false                     // Disabled by default. Freely available proxy servers that work all the time are difficult to find
	proxyURL           = "http://212.186.171.4:80" // Replace with a usable proxy server
	testTrafficTimeout = time.Second
)

var errDastardlyReason = errors.New("some dastardly reason")

func noopConnect() error { return nil }

func closeChanNoPanic(ch chan struct{}) {
	if ch == nil {
		return
	}
	defer func() { _ = recover() }()
	close(ch)
}

func restoreShutdownChannel(ws *Manager) {
	if ws == nil {
		return
	}
	ws.m.Lock()
	defer ws.m.Unlock()
	ws.shutdownC = make(chan struct{})
	for i := range ws.connectionManager {
		for j := range ws.connectionManager[i].connections {
			if conn, ok := ws.connectionManager[i].connections[j].(*connection); ok {
				conn.shutdown = ws.shutdownC
			}
		}
	}
}

// resetManagerForNextConnectAttempt waits for monitor goroutines to drain during test cleanup.
// It intentionally avoids mutating ShutdownC directly.
func resetManagerForNextConnectAttempt(t *testing.T, ws *Manager) {
	t.Helper()
	if ws == nil {
		return
	}
	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		ws.Wg.Wait()
	}()
	require.Eventually(t, func() bool {
		select {
		case <-waitDone:
			return true
		default:
			return false
		}
	}, 5*time.Second, 20*time.Millisecond, "manager cleanup must wait for monitor goroutines")
}

func cleanupManagerMonitors(t *testing.T, ws *Manager) {
	t.Helper()
	if ws == nil {
		return
	}
	ws.setEnabled(false)
	err := ws.Shutdown()
	if err != nil {
		if errors.Is(err, ErrNotConnected) || errors.Is(err, errAlreadyReconnecting) {
			closeChanNoPanic(ws.shutdownC)
			resetManagerForNextConnectAttempt(t, ws)
			require.Eventually(t, func() bool {
				return !ws.connectionMonitorRunning.Load()
			}, 5*time.Second, 20*time.Millisecond, "connection monitor must stop during cleanup")
			restoreShutdownChannel(ws)
			return
		}
		t.Fatalf("manager shutdown cleanup failed: %v", err)
	}
	resetManagerForNextConnectAttempt(t, ws)
	require.Eventually(t, func() bool {
		return !ws.connectionMonitorRunning.Load()
	}, 5*time.Second, 20*time.Millisecond, "connection monitor must stop during cleanup")
}

type testStruct struct {
	Error error
	WC    connection
}

type testRequest struct {
	Event        string          `json:"event"`
	RequestID    int64           `json:"reqid,omitempty"`
	Pairs        []string        `json:"pair"`
	Subscription testRequestData `json:"subscription"`
}

// testRequestData contains details on WS channel
type testRequestData struct {
	Name     string `json:"name,omitempty"`
	Interval int64  `json:"interval,omitempty"`
	Depth    int64  `json:"depth,omitempty"`
}

type testResponse struct {
	RequestID int64 `json:"reqid,omitempty"`
}

type testSubKey struct {
	Mood string
}

func newDefaultSetup() *ManagerSetup {
	return &ManagerSetup{
		ExchangeConfig: &config.Exchange{
			Features: &config.FeaturesConfig{
				Enabled: config.FeaturesEnabledConfig{Websocket: true},
			},
			API: config.APIConfig{
				AuthenticatedWebsocketSupport: true,
			},
			WebsocketTrafficTimeout: testTrafficTimeout,
			Name:                    "GTX",
		},
		DefaultURL:   "testDefaultURL",
		RunningURL:   "wss://testRunningURL",
		Connector:    func() error { return nil },
		Subscriber:   func(subscription.List) error { return nil },
		Unsubscriber: func(subscription.List) error { return nil },
		GenerateSubscriptions: func() (subscription.List, error) {
			return subscription.List{
				{Channel: "TestSub"},
				{Channel: "TestSub2", Key: "purple"},
				{Channel: "TestSub3", Key: testSubKey{"mauve"}},
				{Channel: "TestSub4", Key: 42},
			}, nil
		},
		Features: &protocol.Features{Subscribe: true, Unsubscribe: true},
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	var w *Manager
	err := w.Setup(nil)
	assert.ErrorContains(t, err, "nil pointer: *websocket.Manager")

	w = &Manager{DataHandler: stream.NewRelay(1)}
	err = w.Setup(nil)
	assert.ErrorContains(t, err, "nil pointer: *websocket.ManagerSetup")

	websocketSetup := &ManagerSetup{}
	err = w.Setup(websocketSetup)
	assert.ErrorContains(t, err, "nil pointer: ManagerSetup.Exchange")

	websocketSetup.ExchangeConfig = &config.Exchange{}
	err = w.Setup(websocketSetup)
	assert.ErrorContains(t, err, "nil pointer: ManagerSetup.ExchangeConfig.Features")

	websocketSetup.ExchangeConfig.Features = &config.FeaturesConfig{}
	err = w.Setup(websocketSetup)
	assert.ErrorContains(t, err, "nil pointer: ManagerSetup.Features")

	websocketSetup.Features = &protocol.Features{}
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errExchangeConfigNameEmpty)

	websocketSetup.ExchangeConfig.Name = "testname"
	websocketSetup.Subscriber = func(subscription.List) error { return nil } // kicks off the setup
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errWebsocketConnectorUnset)
	websocketSetup.Subscriber = nil

	websocketSetup.Connector = func() error { return nil }
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errWebsocketSubscriberUnset)

	websocketSetup.Subscriber = func(subscription.List) error { return nil }
	w.features.Unsubscribe = true
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errWebsocketUnsubscriberUnset)

	websocketSetup.Unsubscriber = func(subscription.List) error { return nil }
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errWebsocketSubscriptionsGeneratorUnset)

	websocketSetup.GenerateSubscriptions = func() (subscription.List, error) { return nil, nil }
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errDefaultURLIsEmpty)

	websocketSetup.DefaultURL = "test"
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errRunningURLIsEmpty)

	websocketSetup.RunningURL = "http://www.google.com"
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errInvalidWebsocketURL)

	websocketSetup.RunningURL = "wss://www.google.com"
	websocketSetup.RunningURLAuth = "http://www.google.com"
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errInvalidWebsocketURL)

	websocketSetup.RunningURLAuth = "wss://www.google.com"
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errInvalidTrafficTimeout)

	websocketSetup.ExchangeConfig.WebsocketTrafficTimeout = time.Minute
	err = w.Setup(websocketSetup)
	assert.NoError(t, err, "Setup should not error")
}

func TestConnectionMessageErrors(t *testing.T) { //nolint:tparallel // top-level parallel is safe; serial subtests limit websocket CI contention
	t.Parallel()

	newSingleManager := func(t *testing.T) *Manager {
		t.Helper()

		ws := NewManager()
		t.Cleanup(func() { cleanupManagerMonitors(t, ws) })
		return ws
	}

	newConfiguredSingleManager := func(t *testing.T) *Manager {
		t.Helper()

		ws := newSingleManager(t)
		err := ws.Setup(newDefaultSetup())
		require.NoError(t, err, "Setup must not error")
		ws.trafficTimeout = time.Minute
		return ws
	}

	newConfiguredMultiManager := func(t *testing.T, connSetup *ConnectionSetup) *Manager {
		t.Helper()

		ws := newSingleManager(t)
		setup := newDefaultSetup()
		setup.UseMultiConnectionManagement = true
		err := ws.Setup(setup)
		require.NoError(t, err, "Setup must not error")
		ws.SetCanUseAuthenticatedEndpoints(true)
		if connSetup != nil {
			ws.connectionManager = []*websocket{{setup: connSetup}}
		}
		return ws
	}

	t.Run("single connection preflight", func(t *testing.T) {
		t.Run("disabled websocket", func(t *testing.T) {
			ws := newSingleManager(t)
			ws.connector = noopConnect

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, ErrWebsocketNotEnabled, "Connect must error correctly")
		})

		t.Run("already reconnecting", func(t *testing.T) {
			ws := newSingleManager(t)
			ws.setEnabled(true)
			ws.setState(connectingState)

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, errAlreadyReconnecting, "Connect must error correctly")
		})

		t.Run("nil subscriptions", func(t *testing.T) {
			ws := newSingleManager(t)
			ws.setEnabled(true)
			ws.setState(disconnectedState)
			ws.connector = noopConnect
			ws.subscriptions = nil

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, common.ErrNilPointer, "Connect must get a nil pointer error")
			require.ErrorContains(t, err, "subscriptions", "Connect must get a nil pointer error about subscriptions")
		})

		t.Run("connector error", func(t *testing.T) {
			ws := newSingleManager(t)
			ws.setEnabled(true)
			ws.setState(disconnectedState)
			ws.connector = func() error { return errDastardlyReason }

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, errDastardlyReason, "Connect must error correctly")
		})
	})

	t.Run("single connection requires subscriptions", func(t *testing.T) {
		ws := newConfiguredSingleManager(t)

		require.ErrorIs(t, ws.Connect(t.Context()), ErrSubscriptionsNotAdded)
		require.NoError(t, ws.Shutdown())
	})

	t.Run("single connection forwards read errors to data handler", func(t *testing.T) {
		ws := newConfiguredSingleManager(t)
		ws.Subscriber = func(subs subscription.List) error {
			for _, sub := range subs {
				if err := ws.subscriptions.Add(sub); err != nil {
					return err
				}
			}
			return nil
		}

		require.NoError(t, ws.Connect(t.Context()), "Connect must not error")

		checkToRoutineResult := func(t *testing.T) {
			t.Helper()

			v, ok := <-ws.DataHandler.C
			require.True(t, ok, "ToRoutine must not be closed on us")

			switch err := v.Data.(type) {
			case *gws.CloseError:
				assert.Equal(t, "SpecialText", err.Text, "Should get correct Close Error")
			case error:
				assert.ErrorIs(t, err, errDastardlyReason, "Should get the correct error")
			default:
				assert.Failf(t, "Wrong data type sent to ToRoutine", "Got type: %T", err)
			}
		}

		ws.TrafficAlert <- struct{}{}
		ws.ReadMessageErrors <- errDastardlyReason
		checkToRoutineResult(t)

		ws.ReadMessageErrors <- &gws.CloseError{Code: 1006, Text: "SpecialText"}
		checkToRoutineResult(t)
	})

	t.Run("multi connection", func(t *testing.T) {
		mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		t.Cleanup(mock.Close)

		mockURL := "ws" + mock.URL[len("http"):] + "/ws"
		dial := func(ctx context.Context, conn Connection) error {
			return conn.Dial(ctx, gws.DefaultDialer, nil, nil)
		}
		noopHandler := func(context.Context, Connection, []byte) error { return nil }
		testSubs := subscription.List{{Channel: "test"}}

		t.Run("no pending connections", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, nil)

			err := ws.Connect(t.Context())
			assert.ErrorIs(t, err, errNoPendingConnections, "Connect should error correctly")
		})

		t.Run("missing generate subscriptions", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{URL: mockURL})

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, errWebsocketSubscriptionsGeneratorUnset)
		})

		t.Run("generate subscriptions error", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL: mockURL,
				GenerateSubscriptions: func() (subscription.List, error) {
					return nil, errDastardlyReason
				},
			})

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, errDastardlyReason)
		})

		t.Run("missing connector", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL: mockURL,
				GenerateSubscriptions: func() (subscription.List, error) {
					return testSubs, nil
				},
			})

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, errNoConnectFunc)
		})

		t.Run("missing handler", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL: mockURL,
				GenerateSubscriptions: func() (subscription.List, error) {
					return testSubs, nil
				},
				Connector: func(context.Context, Connection) error {
					return errDastardlyReason
				},
			})

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, errWebsocketDataHandlerUnset)
		})

		t.Run("missing subscriber", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL: mockURL,
				GenerateSubscriptions: func() (subscription.List, error) {
					return testSubs, nil
				},
				Connector: func(context.Context, Connection) error {
					return errDastardlyReason
				},
				Handler: noopHandler,
			})

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, errWebsocketSubscriberUnset)
		})

		t.Run("connector error", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL: mockURL,
				GenerateSubscriptions: func() (subscription.List, error) {
					return testSubs, nil
				},
				Connector: func(context.Context, Connection) error {
					return errDastardlyReason
				},
				Handler:    noopHandler,
				Subscriber: func(context.Context, Connection, subscription.List) error { return nil },
			})

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, errDastardlyReason)
		})

		t.Run("authenticate error", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL: mockURL,
				Authenticate: func(context.Context, Connection) error {
					return errDastardlyReason
				},
				GenerateSubscriptions: func() (subscription.List, error) {
					return testSubs, nil
				},
				Connector:  dial,
				Handler:    noopHandler,
				Subscriber: func(context.Context, Connection, subscription.List) error { return nil },
			})

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, errDastardlyReason)
		})

		t.Run("subscriber error", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL: mockURL,
				GenerateSubscriptions: func() (subscription.List, error) {
					return testSubs, nil
				},
				Connector: dial,
				Handler:   noopHandler,
				Subscriber: func(context.Context, Connection, subscription.List) error {
					return errDastardlyReason
				},
			})

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, errDastardlyReason)
			require.NoError(t, ws.Shutdown())
		})

		t.Run("missing recorded subscriptions", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL: mockURL,
				GenerateSubscriptions: func() (subscription.List, error) {
					return testSubs, nil
				},
				Connector:  dial,
				Handler:    noopHandler,
				Subscriber: func(context.Context, Connection, subscription.List) error { return nil },
			})
			ws.connectionManager[0].subscriptions = subscription.NewStore()

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, ErrSubscriptionsNotAdded)
			require.NoError(t, ws.Shutdown())
		})

		t.Run("successful connect and send raw message", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL: mockURL,
				GenerateSubscriptions: func() (subscription.List, error) {
					return testSubs, nil
				},
				Connector: dial,
				Handler:   noopHandler,
			})
			ws.connectionManager[0].subscriptions = subscription.NewStore()
			ws.connectionManager[0].setup.Subscriber = func(context.Context, Connection, subscription.List) error {
				return ws.connectionManager[0].subscriptions.Add(&subscription.Subscription{Channel: "test"})
			}

			err := ws.Connect(t.Context())
			require.NoError(t, err)

			err = ws.connectionManager[0].connections[0].SendRawMessage(t.Context(), request.Unset, gws.TextMessage, []byte("test"))
			require.NoError(t, err)
			require.NoError(t, ws.Shutdown())
		})

		t.Run("subscriptions not required", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL:                      mockURL,
				SubscriptionsNotRequired: true,
				Connector:                dial,
				Handler:                  noopHandler,
			})

			err := ws.Connect(t.Context())
			require.NoError(t, err, "must not error when connection when no subscriptions are required")
			require.NoError(t, ws.Shutdown())
		})

		t.Run("subscriptions not required connector failure", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, &ConnectionSetup{
				URL:                      mockURL,
				SubscriptionsNotRequired: true,
				Connector: func(context.Context, Connection) error {
					return errors.New("no connect")
				},
				Handler: noopHandler,
			})

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, common.ErrFatal, "must error on connect when no subscriptions are required")
		})
	})
}

func TestConnectTrackOnExistingConnectionManagerRecordsTrackedSubscriptions(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	setup := newDefaultSetup()
	setup.UseMultiConnectionManagement = true
	require.NoError(t, mgr.Setup(setup))
	trackedSub := &subscription.Subscription{Channel: "tracked-only"}

	require.NoError(t, mgr.SetupNewConnection(&ConnectionSetup{
		URL: "wss://tracked-only.example/ws",
		Connector: func(context.Context, Connection) error {
			return errors.New("connector should not be called for tracked-only batch")
		},
		GenerateSubscriptions: func() (subscription.List, error) {
			return subscription.List{trackedSub}, nil
		},
		Subscriber:   func(context.Context, Connection, subscription.List) error { return nil },
		Unsubscriber: func(context.Context, Connection, subscription.List) error { return nil },
		Handler:      func(context.Context, Connection, []byte) error { return nil },
		TrackOnExistingConnection: func(context.Context, Connection, subscription.List) (subscription.List, subscription.List, error) {
			return nil, subscription.List{trackedSub}, nil
		},
	}))

	existingConn := &fakeConnection{subscriptions: subscription.NewStore()}
	mgr.trackConnection(existingConn, mgr.connectionManager[0])

	require.NoError(t, mgr.Connect(t.Context()))
	require.NotNil(t, mgr.connectionManager[0].subscriptions.Get(trackedSub), "tracked subscriptions must be recorded by manager")

	mgr.setEnabled(false)
	require.NoError(t, mgr.Shutdown())
}

func TestCreateConnectAndSubscribe(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	mgr.MaxSubscriptionsPerConnection = 1

	ws := &websocket{subscriptions: subscription.NewStore(), setup: &ConnectionSetup{}}
	subs := subscription.List{{Channel: "one"}, {Channel: "two"}}
	err := mgr.createConnectAndSubscribe(t.Context(), ws, subs)
	require.ErrorIs(t, err, common.ErrFatal, "must return fatal error when exceeding max subscriptions")
	assert.ErrorIs(t, err, errSubscriptionsExceedsLimit, "should return the subscriptions exceeds limit error")

	mgr.MaxSubscriptionsPerConnection = 0
	ws.setup.Connector = func(context.Context, Connection) error { return errConnectionFault }
	err = mgr.createConnectAndSubscribe(t.Context(), ws, subs)
	require.ErrorIs(t, err, common.ErrFatal, "must return fatal error when calling ws.setup.Connector")
	assert.ErrorIs(t, err, errConnectionFault, "should return the correct error when calling ws.setup.Connector")

	ws.setup.Connector = func(context.Context, Connection) error { return nil }
	err = mgr.createConnectAndSubscribe(t.Context(), ws, subs)
	require.ErrorIs(t, err, common.ErrFatal, "must return fatal error when not connected after a potential failed ws.setup.Connector call")
	assert.ErrorIs(t, err, ErrNotConnected, "should signal connection not established")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	t.Cleanup(server.Close)

	ws.setup.URL = "ws" + server.URL[len("http"):] + "/ws"
	ws.setup.Handler = func(context.Context, Connection, []byte) error { return nil }
	ws.setup.Connector = func(ctx context.Context, conn Connection) error {
		return conn.Dial(ctx, gws.DefaultDialer, nil, nil)
	}
	ws.setup.Authenticate = func(context.Context, Connection) error { return errConnectionFault }
	mgr.SetCanUseAuthenticatedEndpoints(true)

	err = mgr.createConnectAndSubscribe(t.Context(), ws, subs)
	require.ErrorIs(t, err, common.ErrFatal, "authenticate failure must be fatal")
	assert.ErrorIs(t, err, errConnectionFault, "should wrap authentication failure reason")
	assert.ErrorIs(t, err, errFailedToAuthenticate, "should wrap authentication failure")
	require.Len(t, ws.connections, 1, "connection must be tracked by websocket")
	require.Len(t, mgr.connections, 1, "websocket connection association must be tracked by manager")
	require.Equal(t, mgr.connections[ws.connections[0]], ws, "manager connections map must track the websocket owner")
	require.NoError(t, ws.connections[0].Shutdown())
	delete(mgr.connections, ws.connections[0])
	ws.connections = nil
	mgr.Wg.Wait()

	ws.setup.Authenticate = func(context.Context, Connection) error { return nil }
	ws.setup.SubscriptionsNotRequired = true
	err = mgr.createConnectAndSubscribe(t.Context(), ws, subs)
	require.ErrorIs(t, err, common.ErrFatal, "subscriptions not required must error when subscriptions are provided")
	require.ErrorIs(t, err, ErrSubscriptionFailure, "subscriptions not required must error when subscriptions are provided")
	require.Len(t, ws.connections, 1, "connection must be tracked by websocket")
	require.Len(t, mgr.connections, 1, "websocket connection association must be tracked by manager")
	require.Equal(t, mgr.connections[ws.connections[0]], ws, "manager connections map must track the websocket owner")
	require.NoError(t, ws.connections[0].Shutdown())
	delete(mgr.connections, ws.connections[0])
	ws.connections = nil
	mgr.Wg.Wait()

	err = mgr.createConnectAndSubscribe(t.Context(), ws, nil)
	require.NoError(t, err, "subscriptions not required with no subscriptions must not error")
	require.Len(t, ws.connections, 1, "connection must be tracked by websocket")
	require.Len(t, mgr.connections, 1, "websocket connection association must be tracked by manager")
	require.Equal(t, mgr.connections[ws.connections[0]], ws, "manager connections map must track the websocket owner")
	require.NoError(t, ws.connections[0].Shutdown())
	delete(mgr.connections, ws.connections[0])
	ws.connections = nil
	mgr.Wg.Wait()

	ws.setup.SubscriptionsNotRequired = false
	ws.setup.Subscriber = func(context.Context, Connection, subscription.List) error {
		return errConnectionFault
	}
	err = mgr.createConnectAndSubscribe(t.Context(), ws, subs)
	require.ErrorIs(t, err, ErrSubscriptionFailure, "subscriber error must bubble as subscription failure")
	assert.ErrorIs(t, err, errConnectionFault, "should include wrapped error")
	require.Len(t, ws.connections, 1, "connection must be tracked by websocket")
	require.Len(t, mgr.connections, 1, "websocket connection association must be tracked by manager")
	require.Equal(t, mgr.connections[ws.connections[0]], ws, "manager connections map must track the websocket owner")
	require.NoError(t, ws.connections[0].Shutdown())
	delete(mgr.connections, ws.connections[0])
	ws.connections = nil
	mgr.Wg.Wait()

	ws.setup.Subscriber = func(context.Context, Connection, subscription.List) error {
		return nil
	}
	err = mgr.createConnectAndSubscribe(t.Context(), ws, subs)
	require.ErrorIs(t, err, ErrSubscriptionFailure, "missing added subscriptions must return subscription failure error")
	require.ErrorIs(t, err, ErrSubscriptionsNotAdded, "missing added subscriptions must return subs not added error")
	require.Len(t, ws.connections, 1, "connection must be tracked by websocket")
	require.Len(t, mgr.connections, 1, "websocket connection association must be tracked by manager")
	require.Equal(t, mgr.connections[ws.connections[0]], ws, "manager connections map must track the websocket owner")
	require.NoError(t, ws.connections[0].Shutdown())
	delete(mgr.connections, ws.connections[0])
	ws.connections = nil
	mgr.Wg.Wait()

	ws.setup.Subscriber = func(context.Context, Connection, subscription.List) error {
		for _, sub := range subs {
			if err := ws.subscriptions.Add(sub); err != nil {
				return err
			}
		}
		return nil
	}
	err = mgr.createConnectAndSubscribe(t.Context(), ws, subs)
	require.NoError(t, err, "createConnectAndSubscribe must succeed")
	require.Len(t, ws.connections, 1, "connection must be tracked by websocket")
	require.Len(t, mgr.connections, 1, "websocket connection association must be tracked by manager")
	require.Equal(t, mgr.connections[ws.connections[0]], ws, "manager connections map must track the websocket owner")
	require.Len(t, ws.connections[0].Subscriptions().List(), len(subs), "connection subscription store must mirror websocket store")
	require.NoError(t, ws.connections[0].Shutdown())
	delete(mgr.connections, ws.connections[0])
	ws.connections = nil
	mgr.Wg.Wait()
}

func TestConnectIncludesCallerName(t *testing.T) {
	t.Parallel()

	ws := NewManager()
	ws.setEnabled(true)
	ws.useMultiConnectionManagement = true
	ws.connectionManager = []*websocket{{
		setup:         &ConnectionSetup{URL: "wss://example.invalid/ws"},
		subscriptions: subscription.NewStore(),
	}}

	err := ws.Connect(request.WithCallerName(t.Context(), t.Name()))
	require.ErrorIs(t, err, errWebsocketSubscriptionsGeneratorUnset)
	assert.ErrorContains(t, err, "cannot connect to [conn:1] [URL:wss://example.invalid/ws]")
}

func TestObserveConnectionStopsOnShutdown(t *testing.T) {
	t.Parallel()

	ws := NewManager()
	ws.verbose = true
	ws.connectionMonitorRunning.Store(true)
	close(ws.shutdownC)
	synctest.Test(t, func(t *testing.T) {
		timer := time.NewTimer(time.Minute)
		t.Cleanup(func() {
			timer.Stop()
		})

		require.True(t, ws.observeConnection(t.Context(), timer))
		assert.False(t, ws.connectionMonitorRunning.Load())
	})
}

func TestObserveTrafficStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	ws := NewManager()
	ws.verbose = true
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	assert.True(t, ws.observeTraffic(ctx, make(<-chan time.Time), func() {}))
}

func TestTrackConnection(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	conn := &connection{}
	first := &websocket{}
	second := &websocket{}

	mgr.trackConnection(conn, first)
	mgr.trackConnection(conn, first)

	require.Len(t, mgr.connections, 1, "manager connection association must stay deduplicated")
	require.Len(t, first.connections, 1, "websocket connection list must not append duplicates")
	assert.Same(t, first, mgr.connections[conn], "manager connection association should stay with the original websocket")
	assert.Same(t, conn, first.connections[0], "websocket should retain the tracked connection")

	assert.PanicsWithValue(t,
		"trackConnection called with connection already associated with a different websocket",
		func() { mgr.trackConnection(conn, second) },
		"trackConnection should panic when the same connection is associated with a different websocket")
	assert.Same(t, first, mgr.connections[conn], "manager connection association should remain unchanged after panic")
	require.Len(t, first.connections, 1, "original websocket must retain the tracked connection after panic")
	assert.Same(t, conn, first.connections[0], "original websocket should still retain the tracked connection")
	assert.Empty(t, second.connections, "new websocket should not gain the tracked connection after panic")
}

func TestSetSubscriptionsNotRequired(t *testing.T) {
	t.Parallel()

	singleConn := NewManager()
	singleConn.GenerateSubs = func() (subscription.List, error) {
		return subscription.List{{Channel: "single"}}, nil
	}

	singleConn.SetSubscriptionsNotRequired()

	subs, err := singleConn.GenerateSubs()
	require.NoError(t, err, "GenerateSubs must not error after subscriptions are disabled")
	assert.Empty(t, subs, "GenerateSubs should return no subscriptions after subscriptions are disabled")

	multiConn := NewManager()
	multiConn.useMultiConnectionManagement = true
	multiConn.connectionManager = []*websocket{
		{setup: nil},
		{setup: &ConnectionSetup{}},
		{setup: &ConnectionSetup{SubscriptionsNotRequired: true}},
	}

	multiConn.SetSubscriptionsNotRequired()

	for i := range multiConn.connectionManager {
		require.NotNil(t,
			multiConn.connectionManager[i].setup,
			"connection setup must be initialised when missing")
		assert.True(t,
			multiConn.connectionManager[i].setup.SubscriptionsNotRequired,
			"connection setup should not require subscriptions after override")
	}
}

func TestSetAllConnectionURLs(t *testing.T) {
	t.Parallel()

	singleConn := NewManager()
	singleConn.Conn = &connection{URL: "ws://old-public.example.com"}
	singleConn.AuthConn = &connection{URL: "ws://old-auth.example.com"}

	err := singleConn.SetAllConnectionURLs("ws://mock.example.com/ws")
	require.NoError(t, err, "SetAllConnectionURLs must not error for single-connection managers")
	assert.Equal(t, "ws://mock.example.com/ws", singleConn.runningURL, "runningURL should be updated for single-connection managers")
	assert.Equal(t, "ws://mock.example.com/ws", singleConn.runningURLAuth, "runningURLAuth should be updated for single-connection managers")
	assert.Equal(t, "ws://mock.example.com/ws", singleConn.Conn.GetURL(), "Conn URL should be updated for single-connection managers")
	assert.Equal(t, "ws://mock.example.com/ws", singleConn.AuthConn.GetURL(), "AuthConn URL should be updated for single-connection managers")

	multiConn := NewManager()
	multiConn.useMultiConnectionManagement = true
	multiConn.connectionManager = []*websocket{
		{setup: nil},
		{setup: &ConnectionSetup{URL: "ws://first.example.com"}},
		{setup: &ConnectionSetup{URL: "ws://second.example.com"}, connections: []Connection{&connection{URL: "ws://live.example.com"}}},
	}

	err = multiConn.SetAllConnectionURLs("ws://mock.example.com/ws")
	require.NoError(t, err, "SetAllConnectionURLs must not error for multi-connection managers")

	for i := range multiConn.connectionManager {
		require.NotNil(t,
			multiConn.connectionManager[i].setup,
			"connection setup must be initialised when missing")
		assert.Equal(t,
			"ws://mock.example.com/ws",
			multiConn.connectionManager[i].setup.URL,
			"connection setup URL should be updated for each multi-connection setup")
	}
	assert.Equal(t,
		"ws://live.example.com",
		multiConn.connectionManager[2].connections[0].GetURL(),
		"existing live connection URL should not be mutated by the pre-connect helper")
}

func TestSetAllConnectionURLsErrorsAfterConnect(t *testing.T) {
	t.Parallel()

	ws := NewManager()

	err := ws.SetAllConnectionURLs("ws://mock.example.com/ws")
	require.NoError(t, err, "SetAllConnectionURLs must allow pre-connect configuration")

	ws.setState(connectingState)
	err = ws.SetAllConnectionURLs("ws://mock.example.com/ws")
	require.ErrorIs(t, err, errAlreadyReconnecting, "SetAllConnectionURLs must error once Connect has started")
	require.ErrorContains(t, err, "SetAllConnectionURLs must be called before Connect")

	ws.setState(connectedState)
	err = ws.SetAllConnectionURLs("ws://mock.example.com/ws")
	require.ErrorIs(t, err, errAlreadyConnected, "SetAllConnectionURLs must error after connect")
	require.ErrorContains(t, err, "SetAllConnectionURLs must be called before Connect")
}

func TestManager(t *testing.T) {
	t.Parallel()

	ws := NewManager()

	err := ws.SetProxyAddress(t.Context(), "garbagio")
	assert.ErrorContains(t, err, "invalid URI for request", "SetProxyAddress should error correctly")

	ws.setEnabled(true)
	defaultSetup := newDefaultSetup()
	err = ws.Setup(defaultSetup) // Sets to enabled again
	require.NoError(t, err, "Setup may not error")

	err = ws.Setup(defaultSetup)
	assert.ErrorIs(t, err, ErrWebsocketAlreadyInitialised, "Setup should error correctly if called twice")

	assert.Equal(t, "GTX", ws.GetName(), "GetName should return correctly")
	assert.True(t, ws.IsEnabled(), "Websocket should be enabled by Setup")

	ws.setEnabled(false)
	assert.False(t, ws.IsEnabled(), "Websocket should be disabled by setEnabled(false)")

	ws.setEnabled(true)
	assert.True(t, ws.IsEnabled(), "Websocket should be enabled by setEnabled(true)")

	err = ws.SetProxyAddress(t.Context(), "https://192.168.0.1:1337")
	assert.NoError(t, err, "SetProxyAddress should not error when not yet connected")

	ws.setState(connectedState)

	ws.connector = func() error { return errDastardlyReason }
	err = ws.SetProxyAddress(t.Context(), "https://192.168.0.1:1336")
	assert.ErrorIs(t, err, errDastardlyReason, "SetProxyAddress should call Connect and error from there")

	err = ws.SetProxyAddress(t.Context(), "https://192.168.0.1:1336")
	assert.ErrorIs(t, err, errSameProxyAddress, "SetProxyAddress should error correctly")

	// removing proxy
	assert.NoError(t, ws.SetProxyAddress(t.Context(), ""))

	ws.setEnabled(true)
	// reinstate proxy
	err = ws.SetProxyAddress(t.Context(), "http://localhost:1337")
	assert.NoError(t, err, "SetProxyAddress should not error")
	assert.Equal(t, "http://localhost:1337", ws.GetProxyAddress(), "GetProxyAddress should return correctly")
	assert.Equal(t, "wss://testRunningURL", ws.GetWebsocketURL(), "GetWebsocketURL should return correctly")
	assert.Equal(t, testTrafficTimeout, ws.trafficTimeout, "trafficTimeout should default correctly")

	assert.ErrorIs(t, ws.Shutdown(), ErrNotConnected)
	ws.setState(connectedState)
	assert.NoError(t, ws.Shutdown())

	ws.connector = func() error { return nil }

	require.ErrorIs(t, ws.Connect(t.Context()), ErrSubscriptionsNotAdded)
	require.NoError(t, ws.Shutdown())

	ws.Subscriber = func(subs subscription.List) error {
		for _, sub := range subs {
			if err := ws.subscriptions.Add(sub); err != nil {
				return err
			}
		}
		return nil
	}
	assert.NoError(t, ws.Connect(t.Context()), "Connect should not error")

	ws.defaultURL = "ws://demos.kaazing.com/echo"
	ws.defaultURLAuth = "ws://demos.kaazing.com/echo"

	err = ws.SetWebsocketURL("", false, false)
	assert.NoError(t, err, "SetWebsocketURL should not error")

	err = ws.SetWebsocketURL("ws://demos.kaazing.com/echo", false, false)
	assert.NoError(t, err, "SetWebsocketURL should not error")

	err = ws.SetWebsocketURL("", true, false)
	assert.NoError(t, err, "SetWebsocketURL should not error")

	err = ws.SetWebsocketURL("ws://demos.kaazing.com/echo", true, false)
	assert.NoError(t, err, "SetWebsocketURL should not error")

	err = ws.SetWebsocketURL("ws://demos.kaazing.com/echo", true, true)
	assert.NoError(t, err, "SetWebsocketURL should not error on reconnect")

	// -- initiate the reconnect which is usually handled by connection monitor
	err = ws.Connect(t.Context())
	assert.NoError(t, err, "ReConnect called manually should not error")

	err = ws.Connect(t.Context())
	assert.ErrorIs(t, err, errAlreadyConnected, "ReConnect should error when already connected")

	err = ws.Shutdown()
	assert.NoError(t, err, "Shutdown should not error")
	ws.Wg.Wait()

	ws.useMultiConnectionManagement = true

	ws.connectionManager = []*websocket{{setup: &ConnectionSetup{URL: "ws://demos.kaazing.com/echo"}, connections: []Connection{&connection{subscriptions: subscription.NewStore()}}}}
	err = ws.SetProxyAddress(t.Context(), "https://192.168.0.1:1337")
	require.NoError(t, err)
}

// TestSetCanUseAuthenticatedEndpoints logic test
func TestSetCanUseAuthenticatedEndpoints(t *testing.T) {
	t.Parallel()
	ws := NewManager()
	assert.False(t, ws.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints should return false")
	ws.SetCanUseAuthenticatedEndpoints(true)
	assert.True(t, ws.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints should return true")
}

// TestDial logic test
func TestDial(t *testing.T) {
	t.Parallel()

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	defer mock.Close()

	testCases := []testStruct{
		{
			WC: connection{
				ExchangeName:     "test1",
				Verbose:          true,
				URL:              "ws" + mock.URL[len("http"):] + "/ws",
				RateLimit:        request.NewWeightedRateLimitByDuration(10 * time.Millisecond),
				ResponseMaxLimit: 7000000000,
			},
		},
		{
			Error: errors.New(" Error: malformed ws or wss URL"),
			WC: connection{
				ExchangeName:     "test2",
				Verbose:          true,
				URL:              "",
				ResponseMaxLimit: 7000000000,
			},
		},
		{
			WC: connection{
				ExchangeName:     "test3",
				Verbose:          true,
				URL:              "ws" + mock.URL[len("http"):] + "/ws",
				ProxyURL:         proxyURL,
				ResponseMaxLimit: 7000000000,
			},
		},
	}
	// Mock server rejects parallel connections
	for i := range testCases {
		if testCases[i].WC.ProxyURL != "" && !useProxyTests {
			t.Log("Proxy testing not enabled, skipping")
			continue
		}
		err := testCases[i].WC.Dial(t.Context(), &gws.Dialer{}, http.Header{}, nil)
		if err != nil {
			if testCases[i].Error != nil && strings.Contains(err.Error(), testCases[i].Error.Error()) {
				return
			}
			t.Fatal(err)
		}
	}
}

// TestSendMessage logic test
func TestSendMessage(t *testing.T) {
	t.Parallel()

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	defer mock.Close()

	testCases := []testStruct{
		{
			WC: connection{
				ExchangeName:     "test1",
				Verbose:          true,
				URL:              "ws" + mock.URL[len("http"):] + "/ws",
				RateLimit:        request.NewWeightedRateLimitByDuration(10 * time.Millisecond),
				ResponseMaxLimit: 7000000000,
			},
		},
		{
			Error: errors.New(" Error: malformed ws or wss URL"),
			WC: connection{
				ExchangeName:     "test2",
				Verbose:          true,
				URL:              "",
				ResponseMaxLimit: 7000000000,
			},
		},
		{
			WC: connection{
				ExchangeName:     "test3",
				Verbose:          true,
				URL:              "ws" + mock.URL[len("http"):] + "/ws",
				ProxyURL:         proxyURL,
				ResponseMaxLimit: 7000000000,
			},
		},
	}
	// Mock server rejects parallel connections
	for x := range testCases {
		if testCases[x].WC.ProxyURL != "" && !useProxyTests {
			t.Log("Proxy testing not enabled, skipping")
			continue
		}
		err := testCases[x].WC.Dial(t.Context(), &gws.Dialer{}, http.Header{}, nil)
		if err != nil {
			if testCases[x].Error != nil && strings.Contains(err.Error(), testCases[x].Error.Error()) {
				return
			}
			t.Fatal(err)
		}
		err = testCases[x].WC.SendJSONMessage(t.Context(), request.Unset, Ping)
		require.NoError(t, err)
		err = testCases[x].WC.SendRawMessage(t.Context(), request.Unset, gws.TextMessage, []byte(Ping))
		require.NoError(t, err)
	}
}

func TestSendMessageReturnResponse(t *testing.T) {
	t.Parallel()

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	defer mock.Close()

	wc := &connection{
		Verbose:          true,
		URL:              "ws" + mock.URL[len("http"):] + "/ws",
		ResponseMaxLimit: time.Second * 5,
		Match:            NewMatch(),
	}
	if wc.ProxyURL != "" && !useProxyTests {
		t.Skip("Proxy testing not enabled, skipping")
	}

	err := wc.Dial(t.Context(), &gws.Dialer{}, http.Header{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	go readMessages(t, wc)

	req := testRequest{
		Event: "subscribe",
		Pairs: []string{currency.NewPairWithDelimiter("XBT", "USD", "/").String()},
		Subscription: testRequestData{
			Name: "ticker",
		},
		RequestID: 12345,
	}

	_, err = wc.SendMessageReturnResponse(t.Context(), request.Unset, req.RequestID, req)
	if err != nil {
		t.Error(err)
	}

	cancelledCtx, fn := context.WithDeadline(t.Context(), time.Now())
	fn()
	_, err = wc.SendMessageReturnResponse(cancelledCtx, request.Unset, "123", req)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// with timeout
	wc.ResponseMaxLimit = 1
	_, err = wc.SendMessageReturnResponse(t.Context(), request.Unset, "123", req)
	assert.ErrorIs(t, err, ErrSignatureTimeout, "SendMessageReturnResponse should error when request ID not found")

	_, err = wc.SendMessageReturnResponsesWithInspector(t.Context(), request.Unset, "123", req, 1, inspection{})
	assert.ErrorIs(t, err, ErrSignatureTimeout, "SendMessageReturnResponse should error when request ID not found")
}

func TestWaitForResponses(t *testing.T) {
	t.Parallel()
	dummy := &connection{
		ResponseMaxLimit: time.Nanosecond,
		Match:            NewMatch(),
	}
	_, err := dummy.waitForResponses(t.Context(), "silly", nil, 1, inspection{})
	require.ErrorIs(t, err, ErrSignatureTimeout)

	dummy.ResponseMaxLimit = time.Second
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = dummy.waitForResponses(ctx, "silly", nil, 1, inspection{})
	require.ErrorIs(t, err, context.Canceled)

	// test break early and hit verbose path
	ch := make(chan []byte, 1)
	ch <- []byte("hello")
	ctx = request.WithVerbose(t.Context())

	got, err := dummy.waitForResponses(ctx, "silly", ch, 2, inspection{breakEarly: true})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "hello", string(got[0]))
}

type inspection struct {
	breakEarly bool
}

func (i inspection) IsFinal([]byte) bool { return i.breakEarly }

type reporter struct {
	name string
	msg  []byte
	t    time.Duration
}

func (r *reporter) Latency(name string, payload []byte, t time.Duration) {
	r.name = name
	r.msg = payload
	r.t = t
}

// readMessages helper func
func readMessages(t *testing.T, wc *connection) {
	t.Helper()
	timer := time.NewTimer(20 * time.Second)
	for {
		select {
		case <-timer.C:
			return
		default:
			resp := wc.ReadMessage()
			if resp.Raw == nil {
				t.Error("connection has closed")
				return
			}
			var incoming testResponse
			err := json.Unmarshal(resp.Raw, &incoming)
			if err != nil {
				t.Error(err)
				return
			}
			if incoming.RequestID > 0 {
				wc.Match.IncomingWithData(incoming.RequestID, resp.Raw)
				return
			}
		}
	}
}

// TestSetupPingHandler logic test
func TestSetupPingHandler(t *testing.T) {
	t.Parallel()

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	defer mock.Close()

	wc := &connection{
		URL:              "ws" + mock.URL[len("http"):] + "/ws",
		ResponseMaxLimit: time.Second * 5,
		Match:            NewMatch(),
		Wg:               &sync.WaitGroup{},
	}

	if wc.ProxyURL != "" && !useProxyTests {
		t.Skip("Proxy testing not enabled, skipping")
	}
	wc.shutdown = make(chan struct{})
	err := wc.Dial(t.Context(), &gws.Dialer{}, http.Header{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	wc.SetupPingHandler(request.Unset, PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PingMessage,
		Delay:             100,
	})

	err = wc.Connection.Close()
	if err != nil {
		t.Error(err)
	}

	err = wc.Dial(t.Context(), &gws.Dialer{}, http.Header{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	wc.SetupPingHandler(request.Unset, PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(Ping),
		Delay:       200,
	})
	time.Sleep(time.Millisecond * 201)
	close(wc.shutdown)
	wc.Wg.Wait()
}

// TestParseBinaryResponse logic test
func TestParseBinaryResponse(t *testing.T) {
	t.Parallel()

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	defer mock.Close()

	wc := &connection{
		URL:              "ws" + mock.URL[len("http"):] + "/ws",
		ResponseMaxLimit: time.Second * 5,
		Match:            NewMatch(),
	}

	var b bytes.Buffer
	g := gzip.NewWriter(&b)
	_, err := g.Write([]byte("hello"))
	require.NoError(t, err, "gzip.Write must not error")
	assert.NoError(t, g.Close(), "Close should not error")

	resp, err := wc.parseBinaryResponse(b.Bytes())
	assert.NoError(t, err, "parseBinaryResponse should not error parsing gzip")
	assert.EqualValues(t, "hello", resp, "parseBinaryResponse should decode gzip")

	b.Reset()
	f, err := flate.NewWriter(&b, 1)
	require.NoError(t, err, "flate.NewWriter must not error")
	_, err = f.Write([]byte("goodbye"))
	require.NoError(t, err, "flate.Write must not error")
	assert.NoError(t, f.Close(), "Close should not error")

	resp, err = wc.parseBinaryResponse(b.Bytes())
	assert.NoError(t, err, "parseBinaryResponse should not error parsing inflate")
	assert.EqualValues(t, "goodbye", resp, "parseBinaryResponse should deflate")

	_, err = wc.parseBinaryResponse([]byte{})
	assert.ErrorContains(t, err, "unexpected EOF", "parseBinaryResponse should error on empty input")
}

// TestCanUseAuthenticatedWebsocketForWrapper logic test
func TestCanUseAuthenticatedWebsocketForWrapper(t *testing.T) {
	t.Parallel()
	ws := &Manager{}
	assert.False(t, ws.CanUseAuthenticatedWebsocketForWrapper(), "CanUseAuthenticatedWebsocketForWrapper should return false")

	ws.setState(connectedState)
	require.True(t, ws.IsConnected(), "IsConnected must return true")
	assert.False(t, ws.CanUseAuthenticatedWebsocketForWrapper(), "CanUseAuthenticatedWebsocketForWrapper should return false")

	ws.SetCanUseAuthenticatedEndpoints(true)
	assert.True(t, ws.CanUseAuthenticatedWebsocketForWrapper(), "CanUseAuthenticatedWebsocketForWrapper should return true")
}

func TestCheckWebsocketURL(t *testing.T) {
	err := checkWebsocketURL("")
	assert.ErrorIs(t, err, errInvalidWebsocketURL, "checkWebsocketURL should error correctly on empty string")

	err = checkWebsocketURL("wowowow:wowowowo")
	assert.ErrorIs(t, err, errInvalidWebsocketURL, "checkWebsocketURL should error correctly on bad format")

	err = checkWebsocketURL("://")
	assert.ErrorContains(t, err, "missing protocol scheme", "checkWebsocketURL should error correctly on bad proto")

	err = checkWebsocketURL("http://www.google.com")
	assert.ErrorIs(t, err, errInvalidWebsocketURL, "checkWebsocketURL should error correctly on wrong proto")

	err = checkWebsocketURL("wss://websocketconnection.place")
	assert.NoError(t, err, "checkWebsocketURL should not error")

	err = checkWebsocketURL("ws://websocketconnection.place")
	assert.NoError(t, err, "checkWebsocketURL should not error")
}

// GenSubs defines a theoretical exchange with pair management
type GenSubs struct {
	EnabledPairs currency.Pairs
	subscribos   subscription.List
	unsubscribos subscription.List
}

// generateSubs default subs created from the enabled pairs list
func (g *GenSubs) generateSubs() (subscription.List, error) {
	superduperchannelsubs := make(subscription.List, len(g.EnabledPairs))
	for i := range g.EnabledPairs {
		superduperchannelsubs[i] = &subscription.Subscription{
			Channel: "TEST:" + strconv.FormatInt(int64(i), 10),
			Pairs:   currency.Pairs{g.EnabledPairs[i]},
		}
	}
	return superduperchannelsubs, nil
}

func (g *GenSubs) SUBME(subs subscription.List) error {
	if len(subs) == 0 {
		return errors.New("WOW")
	}
	g.subscribos = subs
	return nil
}

func (g *GenSubs) UNSUBME(unsubs subscription.List) error {
	if len(unsubs) == 0 {
		return errors.New("WOW")
	}
	g.unsubscribos = unsubs
	return nil
}

func TestDisable(t *testing.T) {
	t.Parallel()
	w := NewManager()
	w.setEnabled(true)
	w.setState(connectedState)
	require.NoError(t, w.Disable(), "Disable must not error")
	assert.ErrorIs(t, w.Disable(), ErrAlreadyDisabled, "Disable should error correctly")
}

func TestEnable(t *testing.T) {
	t.Parallel()
	w := NewManager()
	w.connector = noopConnect
	w.Subscriber = func(subscription.List) error { return nil }
	w.Unsubscriber = func(subscription.List) error { return nil }
	w.GenerateSubs = func() (subscription.List, error) { return nil, nil }
	require.NoError(t, w.Enable(t.Context()), "Enable must not error")
	assert.ErrorIs(t, w.Enable(t.Context()), ErrWebsocketAlreadyEnabled, "Enable should error correctly")
}

func TestSetupNewConnection(t *testing.T) {
	t.Parallel()
	var nonsenseWebsock *Manager
	err := nonsenseWebsock.SetupNewConnection(&ConnectionSetup{URL: "urlstring"})
	assert.ErrorContains(t, err, "nil pointer: *websocket.Manager")

	nonsenseWebsock = &Manager{}
	err = nonsenseWebsock.SetupNewConnection(&ConnectionSetup{URL: "urlstring"})
	assert.ErrorIs(t, err, errExchangeConfigNameEmpty, "SetupNewConnection should error correctly")

	nonsenseWebsock = &Manager{exchangeName: "test"}
	err = nonsenseWebsock.SetupNewConnection(&ConnectionSetup{URL: "urlstring"})
	assert.ErrorIs(t, err, errTrafficAlertNil, "SetupNewConnection should error correctly")

	nonsenseWebsock.TrafficAlert = make(chan struct{}, 1)
	err = nonsenseWebsock.SetupNewConnection(&ConnectionSetup{URL: "urlstring"})
	assert.ErrorIs(t, err, errReadMessageErrorsNil, "SetupNewConnection should error correctly")

	web := NewManager()

	err = web.Setup(newDefaultSetup())
	assert.NoError(t, err, "Setup should not error")

	err = web.SetupNewConnection(&ConnectionSetup{URL: "urlstring"})
	assert.NoError(t, err, "SetupNewConnection should not error")

	err = web.SetupNewConnection(&ConnectionSetup{URL: "urlstring", Authenticated: true})
	assert.NoError(t, err, "SetupNewConnection should not error")

	// Test connection candidates for multi connection tracking.
	multi := NewManager()
	set := newDefaultSetup()
	set.UseMultiConnectionManagement = true
	require.NoError(t, multi.Setup(set))

	err = multi.SetupNewConnection(nil)
	assert.ErrorContains(t, err, "nil pointer: *websocket.ConnectionSetup")

	connSetup := &ConnectionSetup{ResponseCheckTimeout: time.Millisecond}
	err = multi.SetupNewConnection(connSetup)
	require.ErrorIs(t, err, errDefaultURLIsEmpty)

	connSetup.URL = "urlstring"
	err = multi.SetupNewConnection(connSetup)
	require.ErrorIs(t, err, errWebsocketConnectorUnset)

	connSetup.Connector = func(context.Context, Connection) error { return nil }
	err = multi.SetupNewConnection(connSetup)
	require.ErrorIs(t, err, errWebsocketSubscriptionsGeneratorUnset)

	connSetup.GenerateSubscriptions = func() (subscription.List, error) { return nil, nil }
	err = multi.SetupNewConnection(connSetup)
	require.ErrorIs(t, err, errWebsocketSubscriberUnset)

	connSetup.Subscriber = func(context.Context, Connection, subscription.List) error { return nil }
	err = multi.SetupNewConnection(connSetup)
	require.ErrorIs(t, err, errWebsocketUnsubscriberUnset)

	connSetup.Unsubscriber = func(context.Context, Connection, subscription.List) error { return nil }
	err = multi.SetupNewConnection(connSetup)
	require.ErrorIs(t, err, errWebsocketDataHandlerUnset)

	connSetup.Handler = func(context.Context, Connection, []byte) error { return nil }
	connSetup.MessageFilter = []string{"slices are super naughty and not comparable"}
	err = multi.SetupNewConnection(connSetup)
	require.ErrorIs(t, err, errMessageFilterNotComparable)

	connSetup.MessageFilter = "comparable string signature"
	err = multi.SetupNewConnection(connSetup)
	require.NoError(t, err)

	require.Len(t, multi.connectionManager, 1)

	require.Nil(t, multi.AuthConn)
	require.Nil(t, multi.Conn)

	err = multi.SetupNewConnection(connSetup)
	require.ErrorIs(t, err, errDuplicateConnectionSetup)
}

func TestGetConfiguredWebsocketURLs(t *testing.T) {
	t.Parallel()

	var nilManager *Manager
	urls, err := nilManager.GetConfiguredWebsocketURLs()
	assert.ErrorIs(t, err, common.ErrNilPointer)
	assert.Nil(t, urls)

	single := NewManager()
	require.NoError(t, single.Setup(newDefaultSetup()))
	single.runningURL = "wss://single-running"
	urls, err = single.GetConfiguredWebsocketURLs()
	require.NoError(t, err)
	assert.Equal(t, []string{"wss://single-running"}, urls)

	single.runningURL = ""
	urls, err = single.GetConfiguredWebsocketURLs()
	require.NoError(t, err)
	assert.Equal(t, []string{single.defaultURL}, urls)

	single.defaultURL = ""
	urls, err = single.GetConfiguredWebsocketURLs()
	require.NoError(t, err)
	assert.Nil(t, urls, "Configured websocket URLs should be nil when no URLs are set")

	multi := NewManager()
	setup := newDefaultSetup()
	setup.UseMultiConnectionManagement = true
	require.NoError(t, multi.Setup(setup))

	connSetupOne := &ConnectionSetup{
		URL:                   "wss://one.example/ws",
		Connector:             func(context.Context, Connection) error { return nil },
		GenerateSubscriptions: func() (subscription.List, error) { return nil, nil },
		Subscriber:            func(context.Context, Connection, subscription.List) error { return nil },
		Unsubscriber:          func(context.Context, Connection, subscription.List) error { return nil },
		Handler:               func(context.Context, Connection, []byte) error { return nil },
		MessageFilter:         "one",
	}
	require.NoError(t, multi.SetupNewConnection(connSetupOne))

	connSetupTwo := &ConnectionSetup{
		URL:                   "wss://two.example/ws",
		Connector:             func(context.Context, Connection) error { return nil },
		GenerateSubscriptions: func() (subscription.List, error) { return nil, nil },
		Subscriber:            func(context.Context, Connection, subscription.List) error { return nil },
		Unsubscriber:          func(context.Context, Connection, subscription.List) error { return nil },
		Handler:               func(context.Context, Connection, []byte) error { return nil },
		MessageFilter:         "two",
	}
	require.NoError(t, multi.SetupNewConnection(connSetupTwo))

	urls, err = multi.GetConfiguredWebsocketURLs()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"wss://one.example/ws", "wss://two.example/ws"}, urls)
}

func TestConnectionShutdown(t *testing.T) {
	t.Parallel()
	wc := connection{shutdown: make(chan struct{})}
	err := wc.Shutdown()
	assert.NoError(t, err, "Shutdown should not error when connection.Connection is nil")

	err = wc.Dial(t.Context(), &gws.Dialer{}, nil, nil)
	assert.ErrorContains(t, err, "malformed ws or wss URL", "Dial should error correctly")

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	defer mock.Close()

	wc.URL = "ws" + mock.URL[len("http"):] + "/ws"

	err = wc.Dial(t.Context(), &gws.Dialer{}, nil, nil)
	require.NoError(t, err, "Dial must not error")

	err = wc.Shutdown()
	require.NoError(t, err, "Shutdown must not error")
}

// TestLatency logic test
func TestLatency(t *testing.T) {
	t.Parallel()

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	defer mock.Close()

	r := &reporter{}
	exch := "Kraken"
	wc := &connection{
		ExchangeName:     exch,
		Verbose:          true,
		URL:              "ws" + mock.URL[len("http"):] + "/ws",
		ResponseMaxLimit: time.Second * 1,
		Match:            NewMatch(),
		Reporter:         r,
	}
	if wc.ProxyURL != "" && !useProxyTests {
		t.Skip("Proxy testing not enabled, skipping")
	}

	err := wc.Dial(t.Context(), &gws.Dialer{}, http.Header{}, nil)
	require.NoError(t, err)

	go readMessages(t, wc)

	req := testRequest{
		Event:        "subscribe",
		Pairs:        []string{currency.NewPairWithDelimiter("XBT", "USD", "/").String()},
		Subscription: testRequestData{Name: "ticker"},
		RequestID:    12346,
	}

	_, err = wc.SendMessageReturnResponse(t.Context(), request.Unset, req.RequestID, req)
	require.NoError(t, err)
	require.NotEmpty(t, r.t, "Latency must have a duration")
	require.Equal(t, exch, r.name, "Latency must have the correct exchange name")
}

func TestRemoveURLQueryString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "https://www.google.com", removeURLQueryString("https://www.google.com?test=1"), "removeURLQueryString should remove query string")
	assert.Equal(t, "https://www.google.com", removeURLQueryString("https://www.google.com"), "removeURLQueryString should not change URL")
	assert.Empty(t, removeURLQueryString(""), "removeURLQueryString should be empty")
}

func TestWriteToConn(t *testing.T) {
	t.Parallel()
	wc := connection{}
	require.ErrorIs(t, wc.writeToConn(t.Context(), request.Unset, func() error { return nil }), errWebsocketIsDisconnected)
	wc.setConnectedStatus(true)
	// No rate limits set
	require.NoError(t, wc.writeToConn(t.Context(), request.Unset, func() error { return nil }))
	// connection rate limit set
	// Use a longer interval so the second call always requires delay and hits ctx deadline checks deterministically.
	wc.RateLimit = request.NewWeightedRateLimitByDuration(time.Second)
	require.NoError(t, wc.writeToConn(t.Context(), request.Unset, func() error { return nil }))
	ctx, cancel := context.WithTimeout(t.Context(), 0) // deadline exceeded
	cancel()
	require.ErrorIs(t, wc.writeToConn(ctx, request.Unset, func() error { return nil }), context.DeadlineExceeded)
	wc.RateLimit = request.NewWeightedRateLimitByDuration(time.Millisecond)
	// definitions set but with fallover
	wc.RateLimitDefinitions = request.RateLimitDefinitions{
		request.Auth: request.NewWeightedRateLimitByDuration(time.Millisecond),
	}
	require.NoError(t, wc.writeToConn(t.Context(), request.Unset, func() error { return nil }))
	// match with global rate limit
	require.NoError(t, wc.writeToConn(t.Context(), request.Auth, func() error { return nil }))
	// definitions set but connection rate limiter not set
	wc.RateLimit = nil
	require.ErrorIs(t, wc.writeToConn(ctx, request.Unset, func() error { return nil }), errRateLimitNotFound)
}

func TestDrain(t *testing.T) {
	t.Parallel()
	drain(nil)
	ch := make(chan error)
	drain(ch)
	require.Empty(t, ch, "Drain must empty the channel")
	ch = make(chan error, 10)
	for range 10 {
		ch <- errors.New("test")
	}
	drain(ch)
	require.Empty(t, ch, "Drain must empty the channel")
}

func TestMonitorFrame(t *testing.T) {
	t.Parallel()
	ws := Manager{}
	require.Panics(t, func() { ws.monitorFrame(t.Context(), nil, nil) }, "monitorFrame must panic on nil frame")
	require.Panics(t, func() { ws.monitorFrame(t.Context(), nil, func(context.Context) func() bool { return nil }) }, "monitorFrame must panic on nil function")
	ws.Wg.Add(1)
	ws.monitorFrame(t.Context(), &ws.Wg, func(context.Context) func() bool { return func() bool { return true } })
	ws.Wg.Wait()
}

func TestMonitorConnection(t *testing.T) {
	t.Parallel()
	ws := Manager{verbose: true, ReadMessageErrors: make(chan error, 1), shutdownC: make(chan struct{})}
	// Handle timer expired and websocket disabled, shutdown everything.
	timer := time.NewTimer(0)
	ws.setState(connectedState)
	ws.connectionMonitorRunning.Store(true)
	require.True(t, ws.observeConnection(t.Context(), timer))
	require.False(t, ws.connectionMonitorRunning.Load())
	require.Equal(t, disconnectedState, ws.state.Load())
	// Handle timer expired and everything is great, reset the timer.
	ws.setEnabled(true)
	ws.setState(connectedState)
	ws.connectionMonitorRunning.Store(true)
	timer = time.NewTimer(0)
	require.False(t, ws.observeConnection(t.Context(), timer)) // Not shutting down
	// Handle timer expired and for reason its not connected, so lets happily connect again.
	ws.setState(disconnectedState)
	require.False(t, ws.observeConnection(t.Context(), timer)) // Connect is intentionally erroring
	// Handle error from a connection which will then trigger a reconnect
	ws.setState(connectedState)
	ws.DataHandler = stream.NewRelay(1)
	ws.ReadMessageErrors <- errConnectionFault
	timer = time.NewTimer(time.Second)
	require.False(t, ws.observeConnection(t.Context(), timer))
	payload := <-ws.DataHandler.C
	err, ok := payload.Data.(error)
	require.True(t, ok)
	require.ErrorIs(t, err, errConnectionFault)

	// Handle error while still in connecting state; state should be reset so reconnect can proceed.
	ws.setState(connectingState)
	ws.ReadMessageErrors <- errConnectionFault
	timer = time.NewTimer(time.Second)
	require.False(t, ws.observeConnection(t.Context(), timer))
	require.Equal(t, disconnectedState, ws.state.Load())

	// Handle outta closure shell
	innerShell := ws.monitorConnection(t.Context())
	ws.setState(connectedState)
	ws.ReadMessageErrors <- errConnectionFault
	require.False(t, innerShell())
}

func TestMonitorTraffic(t *testing.T) { //nolint:tparallel // top-level parallel is safe; serial subtests limit websocket CI contention
	t.Parallel()

	newTimeoutSignal := func() <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	newManager := func() *Manager {
		return &Manager{
			verbose:      true,
			shutdownC:    make(chan struct{}),
			TrafficAlert: make(chan struct{}, 1),
		}
	}

	t.Run("shutdown signal exits", func(t *testing.T) {
		ws := newManager()
		close(ws.shutdownC)

		require.True(t, ws.observeTraffic(t.Context(), make(chan time.Time), nil))
	})

	t.Run("connecting keeps monitor alive", func(t *testing.T) {
		ws := newManager()
		ws.setState(connectingState)

		require.False(t, ws.observeTraffic(t.Context(), newTimeoutSignal(), nil))
	})

	t.Run("traffic keeps monitor alive", func(t *testing.T) {
		ws := newManager()
		ws.setState(connectedState)
		ws.TrafficAlert <- struct{}{}

		require.False(t, ws.observeTraffic(t.Context(), newTimeoutSignal(), nil))
	})

	t.Run("timeout invokes shutdown handler", func(t *testing.T) {
		ws := newManager()
		ws.setState(connectedState)

		shutdownCalled := false
		require.True(t, ws.observeTraffic(t.Context(), newTimeoutSignal(), func() {
			shutdownCalled = true
			ws.setState(disconnectedState)
		}))
		require.True(t, shutdownCalled, "timeout handler must be called when traffic is missing")
		require.Equal(t, disconnectedState, ws.state.Load())
	})

	t.Run("monitor traffic shell", func(t *testing.T) {
		ws := newManager()
		ws.trafficTimeout = time.Hour
		close(ws.shutdownC)

		innerShell := ws.monitorTraffic(t.Context())
		require.True(t, innerShell())
	})
}

func TestGetConnection(t *testing.T) {
	t.Parallel()
	var ws *Manager
	_, err := ws.GetConnection(nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	require.ErrorContains(t, err, fmt.Sprintf("%T", ws))

	ws = &Manager{}

	_, err = ws.GetConnection(nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	require.ErrorContains(t, err, "messageFilter")

	_, err = ws.GetConnection("testURL")
	require.ErrorIs(t, err, errCannotObtainOutboundConnection)

	ws.useMultiConnectionManagement = true

	_, err = ws.GetConnection("testURL")
	require.ErrorIs(t, err, ErrNotConnected)

	ws.setState(connectedState)

	_, err = ws.GetConnection("testURL")
	require.ErrorIs(t, err, ErrRequestRouteNotFound)

	ws.connectionManager = []*websocket{{
		setup: &ConnectionSetup{MessageFilter: "testURL", URL: "testURL"},
	}}

	_, err = ws.GetConnection("testURL")
	require.ErrorIs(t, err, ErrNotConnected)

	expected := &connection{subscriptions: subscription.NewStore()}
	ws.connectionManager[0].connections = []Connection{expected}

	conn, err := ws.GetConnection("testURL")
	require.NoError(t, err)
	assert.Same(t, expected, conn)
}

func TestShutdown(t *testing.T) {
	t.Parallel()
	m := Manager{}
	m.setState(connectingState)
	require.ErrorIs(t, m.Shutdown(), errAlreadyReconnecting, "Shutdown must error correctly")
	m.setState(disconnectedState)
	require.ErrorIs(t, m.Shutdown(), ErrNotConnected, "Shutdown must error correctly")
	m.setState(connectedState)
	require.Panics(t, func() { _ = m.Shutdown() }, "Shutdown must panic on nil shutdown channel")
	m.resetShutdownSignal()
	require.NoError(t, m.Shutdown(), "Shutdown must not error with no connections")
	m.setState(connectedState)
	m.Conn = &struct{ *connection }{&connection{}}
	m.AuthConn = &struct{ *connection }{&connection{}}
	require.ErrorIs(t, m.Shutdown(), common.ErrTypeAssertFailure, "Shutdown must error with unhandled connection type")

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	defer mock.Close()

	wsURL := "ws" + mock.URL[len("http"):] + "/ws"
	conn, resp, err := gws.DefaultDialer.DialContext(t.Context(), wsURL, nil)
	require.NoError(t, err, "DialContext must not error")
	defer resp.Body.Close()

	m.AuthConn = nil
	m.Conn = nil
	m.connectionManager = []*websocket{
		{connections: []Connection{&connection{Connection: nil, subscriptions: subscription.NewStore()}}},
		{connections: []Connection{&connection{Connection: conn, subscriptions: subscription.NewStore()}}},
	}
	m.setState(connectedState)
	require.NoError(t, m.Shutdown(), "Shutdown must not error with faulty connection in connectionManager")

	gwsConnAuth, respAuth, err := gws.DefaultDialer.DialContext(t.Context(), wsURL, nil)
	require.NoError(t, err, "DialContext must not error")
	defer respAuth.Body.Close()

	gwsConnUnAuth, respUnAuth, err := gws.DefaultDialer.DialContext(t.Context(), wsURL, nil)
	require.NoError(t, err, "DialContext must not error")
	defer respUnAuth.Body.Close()

	m.connectionManager = nil
	authConn := &connection{Connection: gwsConnAuth, shutdown: m.ShutdownSignal()}
	m.AuthConn = authConn
	unauthConn := &connection{Connection: gwsConnUnAuth, shutdown: m.ShutdownSignal()}
	m.Conn = unauthConn

	m.setState(connectedState)
	require.NoError(t, m.Shutdown(), "Shutdown must not error with good connections")

	require.Equal(t, m.ShutdownSignal(), authConn.shutdown, "shutdown channels must be the same after original shutdown channel is closed")
	require.Equal(t, m.ShutdownSignal(), unauthConn.shutdown, "shutdown channels must be the same after original shutdown channel is closed")
}
