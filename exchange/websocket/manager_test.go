package websocket

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	ws.ShutdownC = make(chan struct{})
	for i := range ws.connectionManager {
		for j := range ws.connectionManager[i].connections {
			if conn, ok := ws.connectionManager[i].connections[j].(*connection); ok {
				conn.shutdown = ws.ShutdownC
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
			closeChanNoPanic(ws.ShutdownC)
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
		// Failed connects start the connection monitor, and cleanup waits up to one delay for it to exit
		ws.connectionMonitorDelay = 100 * time.Millisecond
		ws.SetCanUseAuthenticatedEndpoints(true)
		if connSetup != nil {
			ws.connectionManager = []*websocket{{setup: connSetup}}
		}
		return ws
	}

	t.Run("single connection preflight", func(t *testing.T) {
		t.Run("pre-connect hook", func(t *testing.T) {
			ws := newConfiguredSingleManager(t)
			var calls int
			lockFree := true
			ws.preConnect = func(context.Context) {
				calls++
				// Preparation must not hold m.m, or a slow exchange lookup queues order requests behind it.
				if ws.m.TryLock() {
					ws.m.Unlock()
				} else {
					lockFree = false
				}
			}

			require.ErrorIs(t, ws.Connect(t.Context()), ErrSubscriptionsNotAdded, "Connect must continue after preparation")
			assert.Equal(t, 1, calls, "Connect should run preparation before connecting")
			require.NoError(t, ws.Shutdown(), "Shutdown must not error")
			require.ErrorIs(t, ws.Connect(t.Context()), ErrSubscriptionsNotAdded, "Connect must retry after preparation")
			assert.Equal(t, 2, calls, "Connect should retry preparation on a later connection attempt")
			assert.True(t, lockFree, "Connect should run preparation without holding the manager lock")
			require.NoError(t, ws.Shutdown(), "Shutdown must not error")
		})

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
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))

		mockURL := "ws" + mock.URL[len("http"):] + "/ws"
		dial := func(ctx context.Context, conn Connection) error {
			return conn.Dial(ctx, dialer, nil, nil)
		}
		noopHandler := func(context.Context, Connection, []byte) error { return nil }
		testSubs := subscription.List{{Channel: "test"}}

		t.Run("no pending connections", func(t *testing.T) {
			ws := newConfiguredMultiManager(t, nil)

			err := ws.Connect(t.Context())
			assert.ErrorIs(t, err, errNoPendingConnections, "Connect should error correctly")
			assert.False(t, ws.connectionMonitorRunning.Load(), "connection monitor should not start without pending connections")
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

		t.Run("partial generate subscriptions error", func(t *testing.T) {
			partialSubs := make(subscription.List, 1, 2)
			partialSubs[0] = &subscription.Subscription{Channel: "partial-connect"}
			healthySubs := make(subscription.List, 1, 2)
			healthySubs[0] = &subscription.Subscription{Channel: "healthy-connect"}
			var partialCalls, healthyCalls int
			ws := newConfiguredMultiManager(t, nil)
			partialSetup := &ConnectionSetup{
				URL: mockURL,
				GenerateSubscriptions: func() (subscription.List, error) {
					partialCalls++
					return partialSubs, fmt.Errorf("%w: %w", ErrSubscriptionPartial, errDastardlyReason)
				},
				Connector: dial,
				Handler:   noopHandler,
			}
			healthySetup := &ConnectionSetup{
				URL: mockURL,
				GenerateSubscriptions: func() (subscription.List, error) {
					healthyCalls++
					return healthySubs, nil
				},
				Connector: dial,
				Handler:   noopHandler,
			}
			subscriber := func(_ context.Context, conn Connection, subs subscription.List) error {
				return ws.AddSuccessfulSubscriptions(conn, subs...)
			}
			partialSetup.Subscriber = subscriber
			healthySetup.Subscriber = subscriber
			ws.connectionManager = []*websocket{{setup: partialSetup}, {setup: healthySetup}}

			err := ws.Connect(t.Context())
			require.ErrorIs(t, err, ErrSubscriptionPartial, "Connect must surface partial subscription generation")
			assert.True(t, ws.IsConnected(), "websocket should remain connected after partial subscription generation")
			require.NotNil(t, ws.connectionManager[0].subscriptions.Get(partialSubs[0]),
				"Connect must retain subscriptions returned with a partial generation error")
			require.NotNil(t, ws.connectionManager[1].subscriptions.Get(healthySubs[0]),
				"Connect must continue through healthy generators after a partial generation error")

			partialRefresh := &subscription.Subscription{Channel: "partial-refresh"}
			healthyRefresh := &subscription.Subscription{Channel: "healthy-refresh"}
			partialSubs = append(partialSubs, partialRefresh)
			healthySubs = append(healthySubs, healthyRefresh)
			err = ws.FlushChannels(t.Context())
			require.ErrorIs(t, err, ErrSubscriptionPartial, "FlushChannels must surface partial subscription generation")
			assert.Equal(t, 2, partialCalls, "partial generator should run during connect and flush")
			assert.Equal(t, 2, healthyCalls, "healthy generator should run during connect and flush")
			require.NotNil(t, ws.connectionManager[0].subscriptions.Get(partialRefresh),
				"FlushChannels must retain subscriptions returned with a partial generation error")
			require.NotNil(t, ws.connectionManager[1].subscriptions.Get(healthyRefresh),
				"FlushChannels must continue through healthy generators after a partial generation error")

			require.NoError(t, ws.Shutdown(), "Shutdown must not error")
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

	server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))

	ws.setup.URL = "ws" + server.URL[len("http"):] + "/ws"
	ws.setup.Handler = func(context.Context, Connection, []byte) error { return nil }
	ws.setup.Connector = func(ctx context.Context, conn Connection) error {
		return conn.Dial(ctx, dialer, nil, nil)
	}
	ws.setup.Authenticate = func(context.Context, Connection) error { return errConnectionFault }
	mgr.SetCanUseAuthenticatedEndpoints(true)

	err = mgr.createConnectAndSubscribe(t.Context(), ws, subs)
	require.ErrorIs(t, err, common.ErrFatal, "authenticate failure must be fatal")
	assert.ErrorIs(t, err, errConnectionFault, "should wrap authentication failure reason")
	assert.ErrorIs(t, err, errFailedToAuthenticate, "should wrap authentication failure")
	require.Empty(t, ws.connections, "a connection failing to authenticate must not be tracked by websocket")
	assert.Empty(t, mgr.connections, "a connection failing to authenticate should not be tracked by manager")
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

// newConnectionCountingServer returns a mock server URL and dialer, along with the number of websocket connections the
// server currently holds open, so that tests can tell whether a client socket was released
func newConnectionCountingServer(t *testing.T) (string, *gws.Dialer, *atomic.Int64) {
	t.Helper()
	var open atomic.Int64
	server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		open.Add(1)
		defer open.Add(-1)
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	return "ws" + server.URL[len("http"):] + "/ws", dialer, &open
}

func TestConnectReleasesConnectionWhenConnectorFails(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		wsURL, dialer, openConnections := newConnectionCountingServer(t)
		ws := NewManager()
		t.Cleanup(func() { cleanupManagerMonitors(t, ws) })
		setup := newDefaultSetup()
		setup.RunningURL = wsURL
		require.NoError(t, ws.Setup(setup), "Setup must not error")
		require.NoError(t, ws.SetupNewConnection(&ConnectionSetup{URL: wsURL}), "SetupNewConnection must not error")
		require.NoError(t, ws.SetupNewConnection(&ConnectionSetup{URL: wsURL, Authenticated: true}), "SetupNewConnection must not error for the authenticated connection")

		readerDone := make(chan struct{})
		// Mirrors an exchange whose login on the authenticated socket fails after the public socket and its reader are
		// already running
		ws.connector = func() error {
			if err := ws.Conn.Dial(t.Context(), dialer, http.Header{}, nil); err != nil {
				return err
			}
			ws.Wg.Go(func() {
				defer close(readerDone)
				for {
					if resp := ws.Conn.ReadMessage(); resp.Raw == nil {
						return
					}
				}
			})
			if err := ws.AuthConn.Dial(t.Context(), dialer, http.Header{}, nil); err != nil {
				return err
			}
			return errDastardlyReason
		}

		require.ErrorIs(t, ws.Connect(t.Context()), errDastardlyReason, "Connect must return the connector error")
		assert.False(t, ws.IsConnected(), "IsConnected should return false after a failed connect")
		assert.False(t, ws.IsConnecting(), "IsConnecting should return false after a failed connect")
		select {
		case <-readerDone:
		default:
			assert.Fail(t, "reader should be stopped by the time Connect returns")
		}
		assert.Eventually(t, func() bool { return openConnections.Load() == 0 }, 5*time.Second, 10*time.Millisecond, "server should see both sockets closed")
	})
}

func TestConnectDoesNotStartMonitorWithoutConnector(t *testing.T) {
	t.Parallel()

	ws := NewManager()
	t.Cleanup(func() { cleanupManagerMonitors(t, ws) })
	require.NoError(t, ws.Setup(newDefaultSetup()), "Setup must not error")
	ws.connector = nil // Setup rejects a nil connector, so it is cleared afterwards
	require.ErrorIs(t, ws.Connect(t.Context()), errNoConnectFunc, "Connect must error without a connector")
	assert.False(t, ws.connectionMonitorRunning.Load(), "connection monitor should not start without a connector")
}

func TestConnectDoesNotStartMonitorAfterMultiConnectionAttemptWithEndedContext(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		ws := newRetryTestManager(t, true, "ws"+mock.URL[len("http"):]+"/ws", dialer)
		setup := ws.connectionManager[0].setup
		connector := setup.Connector
		// The in-memory dialer does not watch the context, so the connector does, as a network dial would
		setup.Connector = func(ctx context.Context, conn Connection) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return connector(ctx, conn)
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, ws.Connect(ctx), context.Canceled, "Connect must error once its context has ended")
		assert.False(t, ws.connectionMonitorRunning.Load(), "connection monitor should not start when every retry would dial with an ended context")
	})
}

func TestConnectStartsMonitorAfterMultiConnectionSuccessWithEndedContext(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		ws := newRetryTestManager(t, true, "ws"+mock.URL[len("http"):]+"/ws", dialer)
		ws.trafficTimeout = time.Hour // Stops the traffic monitor closing the idle connection on its own
		ctx, cancel := context.WithCancel(t.Context())
		setup := ws.connectionManager[0].setup
		connector := setup.Connector
		// Ends the context once the socket is up, as a deadline expiring late in a successful attempt would
		setup.Connector = func(ctx context.Context, conn Connection) error {
			defer cancel()
			return connector(ctx, conn)
		}

		require.NoError(t, ws.Connect(ctx), "Connect must not error")
		require.NoError(t, ws.Disable(), "Disable must not error")
		time.Sleep(2 * ws.connectionMonitorDelay)
		synctest.Wait()
		assert.False(t, ws.IsConnected(), "IsConnected should return false once the monitor sees the websocket was disabled")
	})
}

func TestConnectRetriesSingleConnectionAfterContextEnds(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var upgradeAttempts atomic.Int64
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if upgradeAttempts.Add(1) == 1 {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
				return
			}
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		ws := newRetryTestManager(t, false, "ws"+mock.URL[len("http"):]+"/ws", dialer)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		require.ErrorIs(t, ws.Connect(ctx), gws.ErrBadHandshake, "Connect must error while the server is unavailable")
		time.Sleep(ws.connectionMonitorDelay)
		synctest.Wait()
		// Single connection connectors and subscribers take no context, so the retry can still succeed
		assert.True(t, ws.IsConnected(), "IsConnected should return true once the server is available, though the context has ended")
	})
}

func TestConnectReturnsTeardownError(t *testing.T) {
	t.Parallel()

	ws := NewManager()
	require.NoError(t, ws.Setup(newDefaultSetup()), "Setup must not error")
	// The failed Connect below starts the connection monitor, which retries until the websocket is disabled
	t.Cleanup(func() { assert.NoError(t, ws.Disable(), "Disable should not error") })
	ws.Conn = &struct{ *connection }{&connection{}}
	ws.connector = func() error { return errDastardlyReason }

	err := ws.Connect(t.Context())
	require.ErrorIs(t, err, errDastardlyReason, "Connect must return the connector error")
	assert.ErrorIs(t, err, common.ErrTypeAssertFailure, "Connect should return the teardown error")
}

func TestCreateConnectAndSubscribeClosesUntrackedConnection(t *testing.T) {
	t.Parallel()

	newWebsocket := func(t *testing.T, connector func(context.Context, Connection, *gws.Dialer) error) (*websocket, *atomic.Int64) {
		t.Helper()
		wsURL, dialer, openConnections := newConnectionCountingServer(t)
		return &websocket{
			subscriptions: subscription.NewStore(),
			setup: &ConnectionSetup{
				URL: wsURL,
				Connector: func(ctx context.Context, conn Connection) error {
					return connector(ctx, conn, dialer)
				},
			},
		}, openConnections
	}

	t.Run("connector error after dial", func(t *testing.T) {
		t.Parallel()
		ws, openConnections := newWebsocket(t, func(ctx context.Context, conn Connection, dialer *gws.Dialer) error {
			require.NoError(t, conn.Dial(ctx, dialer, nil, nil), "Dial must not error")
			return errDastardlyReason
		})

		err := NewManager().createConnectAndSubscribe(t.Context(), ws, nil)
		require.ErrorIs(t, err, common.ErrFatal, "createConnectAndSubscribe must return a fatal error")
		assert.ErrorIs(t, err, errDastardlyReason, "createConnectAndSubscribe should return the connector error")
		assert.Eventually(t, func() bool { return openConnections.Load() == 0 }, 5*time.Second, 10*time.Millisecond, "server should see the socket closed")
	})

	t.Run("dialled connection no longer connected", func(t *testing.T) {
		t.Parallel()
		ws, openConnections := newWebsocket(t, func(ctx context.Context, conn Connection, dialer *gws.Dialer) error {
			require.NoError(t, conn.Dial(ctx, dialer, nil, nil), "Dial must not error")
			c, ok := conn.(*connection)
			require.True(t, ok, "connection must be a *connection")
			c.setConnectedStatus(false) // As if a read error was seen before the connector returned
			return nil
		})

		err := NewManager().createConnectAndSubscribe(t.Context(), ws, nil)
		require.ErrorIs(t, err, common.ErrFatal, "createConnectAndSubscribe must return a fatal error")
		assert.ErrorIs(t, err, ErrNotConnected, "createConnectAndSubscribe should return ErrNotConnected")
		assert.Eventually(t, func() bool { return openConnections.Load() == 0 }, 5*time.Second, 10*time.Millisecond, "server should see the socket closed")
	})
}

func TestConnectStartsTrafficMonitorOnceConnected(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		ws := NewManager()
		t.Cleanup(func() { cleanupManagerMonitors(t, ws) })
		setup := newDefaultSetup()
		setup.UseMultiConnectionManagement = true
		// Cleanup gives the connection monitor five seconds to stop, so its cycle must not follow the default
		setup.ExchangeConfig.ConnectionMonitorDelay = time.Second
		require.NoError(t, ws.Setup(setup), "Setup must not error")

		connectorErr := errDastardlyReason
		var dials atomic.Int64
		require.NoError(t, ws.SetupNewConnection(&ConnectionSetup{
			URL: "ws" + mock.URL[len("http"):] + "/ws",
			Connector: func(ctx context.Context, conn Connection) error {
				if connectorErr != nil {
					return connectorErr
				}
				dials.Add(1)
				return conn.Dial(ctx, dialer, nil, nil)
			},
			GenerateSubscriptions: func() (subscription.List, error) {
				return subscription.List{{Channel: "test"}}, nil
			},
			Subscriber: func(_ context.Context, conn Connection, subs subscription.List) error {
				return ws.AddSuccessfulSubscriptions(conn, subs...)
			},
			Unsubscriber: func(context.Context, Connection, subscription.List) error { return nil },
			Handler:      func(context.Context, Connection, []byte) error { return nil },
		}), "SetupNewConnection must not error")

		for range 5 {
			require.ErrorIs(t, ws.Connect(t.Context()), errDastardlyReason, "Connect must return the connector error")
		}
		connectorErr = nil
		require.NoError(t, ws.Connect(t.Context()), "Connect must not error")

		// Traffic arrives well inside the timeout, which a single traffic monitor accepts indefinitely
		for range 5 {
			time.Sleep(ws.trafficTimeout * 6 / 10)
			select {
			case ws.TrafficAlert <- struct{}{}:
			default:
			}
		}
		synctest.Wait()
		assert.True(t, ws.IsConnected(), "IsConnected should return true while traffic keeps arriving after failed connection attempts")
		assert.Equal(t, int64(1), dials.Load(), "connection should not be dropped and dialled again")

		// Once traffic stops, the traffic monitor must drop the connection so that it is dialled again
		time.Sleep(3*ws.trafficTimeout + ws.connectionMonitorDelay)
		synctest.Wait()
		assert.Greater(t, dials.Load(), int64(1), "connection should be dropped and dialled again once traffic stops")
	})
}

func TestConnectSingleConnectionTrafficMonitorDropsSilentConnection(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		wsURL := "ws" + mock.URL[len("http"):] + "/ws"
		ws := NewManager()
		t.Cleanup(func() { cleanupManagerMonitors(t, ws) })
		setup := newDefaultSetup()
		setup.RunningURL = wsURL
		// Cleanup gives the connection monitor five seconds to stop, so its cycle must not follow the default
		setup.ExchangeConfig.ConnectionMonitorDelay = time.Second
		setup.GenerateSubscriptions = func() (subscription.List, error) { return nil, nil }
		require.NoError(t, ws.Setup(setup), "Setup must not error")
		require.NoError(t, ws.SetupNewConnection(&ConnectionSetup{URL: wsURL}), "SetupNewConnection must not error")

		var dials atomic.Int64
		ws.connector = func() error {
			dials.Add(1)
			if err := ws.Conn.Dial(t.Context(), dialer, http.Header{}, nil); err != nil {
				return err
			}
			ws.Wg.Go(func() {
				for {
					if resp := ws.Conn.ReadMessage(); resp.Raw == nil {
						return
					}
				}
			})
			return nil
		}
		require.NoError(t, ws.Connect(t.Context()), "Connect must not error")

		// No traffic arrives, so the traffic monitor must drop the connection and the connection monitor dial it again
		time.Sleep(3*ws.trafficTimeout + ws.connectionMonitorDelay)
		synctest.Wait()
		assert.Greater(t, dials.Load(), int64(1), "connection should be dropped and dialled again when no traffic arrives")
	})
}

// newRetryTestManager returns a manager that dials wsURL through dialer, using multi connection management if requested
func newRetryTestManager(t *testing.T, multiConnection bool, wsURL string, dialer *gws.Dialer) *Manager {
	t.Helper()
	ws := NewManager()
	t.Cleanup(func() { cleanupManagerMonitors(t, ws) })
	setup := newDefaultSetup()
	setup.RunningURL = wsURL
	setup.UseMultiConnectionManagement = multiConnection
	require.NoError(t, ws.Setup(setup), "Setup must not error")

	if multiConnection {
		require.NoError(t, ws.SetupNewConnection(&ConnectionSetup{
			URL: wsURL,
			Connector: func(ctx context.Context, conn Connection) error {
				return conn.Dial(ctx, dialer, nil, nil)
			},
			GenerateSubscriptions: func() (subscription.List, error) {
				return subscription.List{{Channel: "test"}}, nil
			},
			Subscriber: func(_ context.Context, conn Connection, subs subscription.List) error {
				return ws.AddSuccessfulSubscriptions(conn, subs...)
			},
			Unsubscriber: func(context.Context, Connection, subscription.List) error { return nil },
			Handler:      func(context.Context, Connection, []byte) error { return nil },
		}), "SetupNewConnection must not error")
		return ws
	}

	require.NoError(t, ws.SetupNewConnection(&ConnectionSetup{URL: wsURL}), "SetupNewConnection must not error")
	ws.connector = func() error {
		if err := ws.Conn.Dial(t.Context(), dialer, http.Header{}, nil); err != nil {
			return err
		}
		ws.Wg.Add(1)
		go ws.Reader(t.Context(), ws.Conn, func(context.Context, Connection, []byte) error { return nil })
		return nil
	}
	ws.Subscriber = func(subs subscription.List) error {
		for _, sub := range subs {
			if err := ws.subscriptions.Add(sub); err != nil {
				return err
			}
		}
		return nil
	}
	return ws
}

func TestConnectRetriesFailedInitialConnection(t *testing.T) {
	t.Parallel()

	testRetry := func(t *testing.T, multiConnection bool) {
		t.Helper()
		synctest.Test(t, func(t *testing.T) {
			var upgradeAttempts atomic.Int64
			mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if upgradeAttempts.Add(1) == 1 {
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				}
				mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
			}))
			ws := newRetryTestManager(t, multiConnection, "ws"+mock.URL[len("http"):]+"/ws", dialer)

			require.ErrorIs(t, ws.Connect(t.Context()), gws.ErrBadHandshake, "Connect must error while the server is unavailable")
			time.Sleep(ws.connectionMonitorDelay)
			synctest.Wait()
			assert.True(t, ws.IsConnected(), "IsConnected should return true once the server is available, without another Connect call")
			assert.Equal(t, int64(2), upgradeAttempts.Load(), "server should see exactly one retry")
		})
	}

	t.Run("single connection", func(t *testing.T) {
		t.Parallel()
		testRetry(t, false)
	})
	t.Run("multi connection", func(t *testing.T) {
		t.Parallel()
		testRetry(t, true)
	})
}

func TestConnectShutsDownConnectionCompletedAfterDisable(t *testing.T) {
	t.Parallel()

	testDisable := func(t *testing.T, multiConnection bool) {
		t.Helper()
		synctest.Test(t, func(t *testing.T) {
			mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
			}))
			ws := newRetryTestManager(t, multiConnection, "ws"+mock.URL[len("http"):]+"/ws", dialer)
			ws.trafficTimeout = time.Hour // Stops the traffic monitor closing the idle connection on its own
			// Mirrors a slow login that outlasts the connection monitor's first check
			if multiConnection {
				setup := ws.connectionManager[0].setup
				connector := setup.Connector
				setup.Connector = func(ctx context.Context, conn Connection) error {
					time.Sleep(ws.connectionMonitorDelay * 3 / 2)
					return connector(ctx, conn)
				}
			} else {
				connector := ws.connector
				ws.connector = func() error {
					time.Sleep(ws.connectionMonitorDelay * 3 / 2)
					return connector()
				}
			}

			go func() {
				time.Sleep(ws.connectionMonitorDelay / 2)
				assert.NoError(t, ws.Disable(), "Disable should not error")
			}()
			require.NoError(t, ws.Connect(t.Context()), "Connect must not error")
			time.Sleep(2 * ws.connectionMonitorDelay)
			synctest.Wait()
			assert.False(t, ws.IsConnected(), "IsConnected should return false once the monitor sees the websocket was disabled while connecting")
		})
	}

	t.Run("single connection", func(t *testing.T) {
		t.Parallel()
		testDisable(t, false)
	})
	t.Run("multi connection", func(t *testing.T) {
		t.Parallel()
		testDisable(t, true)
	})
}

func TestConnectRetriesStopWhenDisabled(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var upgradeAttempts atomic.Int64
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			upgradeAttempts.Add(1)
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		}))
		ws := newRetryTestManager(t, false, "ws"+mock.URL[len("http"):]+"/ws", dialer)

		require.ErrorIs(t, ws.Connect(t.Context()), gws.ErrBadHandshake, "Connect must error while the server is unavailable")
		time.Sleep(2 * ws.connectionMonitorDelay)
		synctest.Wait()
		require.Equal(t, int64(3), upgradeAttempts.Load(), "connection monitor must retry once per delay")

		require.NoError(t, ws.Disable(), "Disable must not error")
		time.Sleep(3 * ws.connectionMonitorDelay)
		synctest.Wait()
		assert.Equal(t, int64(3), upgradeAttempts.Load(), "connection monitor should not retry once disabled")
		assert.False(t, ws.connectionMonitorRunning.Load(), "connection monitor should exit once disabled")
	})
}

func TestConnectRetriesReleaseConnectionsWhenConnectorFails(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		wsURL, dialer, openConnections := newConnectionCountingServer(t)
		ws := newRetryTestManager(t, false, wsURL, dialer)
		var attempts atomic.Int64
		connect := ws.connector
		// Mirrors an exchange whose login keeps failing after the socket and its reader are running
		ws.connector = func() error {
			attempts.Add(1)
			if err := connect(); err != nil {
				return err
			}
			return errDastardlyReason
		}

		require.ErrorIs(t, ws.Connect(t.Context()), errDastardlyReason, "Connect must return the connector error")
		time.Sleep(5*ws.connectionMonitorDelay + ws.connectionMonitorDelay/2)
		synctest.Wait()
		assert.Equal(t, int64(6), attempts.Load(), "connection monitor should retry once per delay")
		assert.Zero(t, openConnections.Load(), "server should see every failed attempt's socket closed")
		assert.False(t, ws.IsConnected(), "IsConnected should return false while the connector keeps failing")
	})
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
	assert.ErrorIs(t, err, errWebsocketAlreadyInitialised, "Setup should error correctly if called twice")

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

	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))

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
		err := testCases[i].WC.Dial(t.Context(), dialer, http.Header{}, nil)
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

	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))

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
		err := testCases[x].WC.Dial(t.Context(), dialer, http.Header{}, nil)
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

	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))

	wc := &connection{
		Verbose:          true,
		URL:              "ws" + mock.URL[len("http"):] + "/ws",
		ResponseMaxLimit: time.Second * 5,
		Match:            NewMatch(),
	}
	if wc.ProxyURL != "" && !useProxyTests {
		t.Skip("Proxy testing not enabled, skipping")
	}

	err := wc.Dial(t.Context(), dialer, http.Header{}, nil)
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

	synctest.Test(t, func(t *testing.T) {
		var pings atomic.Int64
		mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, _ *gws.Conn) error {
				assert.Equal(t, Ping, string(msg), "mock server should only receive the configured ping message")
				pings.Add(1)
				return nil
			})
		}))

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
		require.NoError(t, wc.Dial(t.Context(), dialer, http.Header{}, nil), "Dial must not error")
		first := wc.Connection
		// the bubble waits for every goroutine, including the mock server's handler, which only returns once the connection closes
		t.Cleanup(func() {
			close(wc.shutdown)
			wc.Wg.Wait()
			// the test closes the first connection itself, which is still current only if the second dial failed
			if wc.Connection != first {
				assert.NoError(t, wc.Connection.Close(), "Close should not error")
			}
		})

		wc.SetupPingHandler(request.Unset, PingHandler{
			UseGorillaHandler: true,
			MessageType:       gws.PingMessage,
			Delay:             100 * time.Millisecond,
		})
		require.NoError(t, wc.Connection.Close(), "Close must not error")

		require.NoError(t, wc.Dial(t.Context(), dialer, http.Header{}, nil), "Dial must not error")
		wc.SetupPingHandler(request.Unset, PingHandler{
			MessageType: gws.TextMessage,
			Message:     []byte(Ping),
			Delay:       200 * time.Millisecond,
		})

		// the ping handler wakes at the same instant, and Sleep also waits for its ping to reach the server
		for sent := range int64(3) {
			synctest.Sleep(200*time.Millisecond - time.Nanosecond)
			assert.Equal(t, sent, pings.Load(), "ping handler should not send before its delay elapses")
			synctest.Sleep(time.Nanosecond)
			assert.Equal(t, sent+1, pings.Load(), "ping handler should send one ping each time its delay elapses")
		}
	})
}

// TestParseBinaryResponse logic test
func TestParseBinaryResponse(t *testing.T) {
	t.Parallel()

	var gzipBuffer bytes.Buffer
	g := gzip.NewWriter(&gzipBuffer)
	_, err := g.Write([]byte("hello"))
	require.NoError(t, err, "gzip.Write must not error")
	require.NoError(t, g.Close(), "gzip.Close must not error")
	corruptGZIPPayload := bytes.Clone(gzipBuffer.Bytes())
	corruptGZIPPayload[len(corruptGZIPPayload)-1] ^= 0xff

	// Captured from MEXC's public spot websocket for
	// spot@public.aggre.bookTicker.v3.api.pb@100ms@KASUSDT.
	mexcFrame, err := hex.DecodeString("0a3473706f74407075626c69632e61676772652e626f6f6b5469636b657" +
		"22e76332e6170692e7062403130306d73404b4153555344541a074b41535553445430d4d882c2fc33da1335" +
		"0a08302e3032363239331204313430371a08302e30323633353722053237372e352a0b3133303438393134" +
		"3133393090d482c2fc33")
	require.NoError(t, err, "captured MEXC frame must decode")

	testCases := []struct {
		name   string
		input  []byte
		expect []byte
		err    error
	}{
		{name: "gzip", input: gzipBuffer.Bytes(), expect: []byte("hello")},
		{name: "uncompressed MEXC protobuf", input: mexcFrame, expect: bytes.Clone(mexcFrame)},
		{name: "unrecognised binary", input: []byte{0x01, 0x02, 0x03}, expect: []byte{0x01, 0x02, 0x03}},
		{name: "nil"},
		{name: "empty frame", input: []byte{}, expect: []byte{}},
		{name: "single byte", input: []byte{0x1f}, expect: []byte{0x1f}},
		{name: "partial gzip magic", input: []byte{0x1f, 0x00}, expect: []byte{0x1f, 0x00}},
		{name: "corrupt gzip header", input: []byte{0x1f, 0x8b, 0x08}, err: io.ErrUnexpectedEOF},
		{name: "corrupt gzip payload", input: corruptGZIPPayload, err: gzip.ErrChecksum},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := parseBinaryResponse(tc.input)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "parseBinaryResponse should return the expected error")
				assert.Nil(t, resp, "parseBinaryResponse should not return a payload with an error")
				return
			}
			require.NoError(t, err, "parseBinaryResponse must not error")
			assert.Equal(t, tc.expect, resp, "parseBinaryResponse should return the expected response")
		})
	}
}

// TestReadMessageBinary ensures binary frames reach the caller as the exchange handler expects
func TestReadMessageBinary(t *testing.T) {
	// Subtests share the one connection and write then read in order, so neither this test nor its subtests call t.Parallel.
	mock, dialer := mockws.NewTestServer(t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, p []byte, c *gws.Conn) error {
		return c.WriteMessage(gws.BinaryMessage, p)
	}))

	wc := &connection{
		ExchangeName:     "test",
		URL:              "ws" + mock.URL[len("http"):],
		ResponseMaxLimit: time.Second * 5,
		Match:            NewMatch(),
	}
	require.NoError(t, wc.Dial(t.Context(), dialer, http.Header{}, nil), "Dial must not error")
	defer func() { assert.NoError(t, wc.Connection.Close(), "Close should not error") }()

	var gzipBuffer bytes.Buffer
	g := gzip.NewWriter(&gzipBuffer)
	_, err := g.Write([]byte("hello"))
	require.NoError(t, err, "gzip.Write must not error")
	require.NoError(t, g.Close(), "gzip.Close must not error")

	mexcFrame, err := hex.DecodeString("0a3473706f74407075626c69632e61676772652e626f6f6b5469636b657" +
		"22e76332e6170692e7062403130306d73404b4153555344541a074b41535553445430d4d882c2fc33da1335" +
		"0a08302e3032363239331204313430371a08302e30323633353722053237372e352a0b3133303438393134" +
		"3133393090d482c2fc33")
	require.NoError(t, err, "captured MEXC frame must decode")

	for _, tc := range []struct {
		name   string
		send   []byte
		expect []byte
	}{
		{name: "gzip is decompressed", send: gzipBuffer.Bytes(), expect: []byte("hello")},
		{name: "uncompressed protobuf is preserved", send: mexcFrame, expect: mexcFrame},
		{name: "malformed gzip yields an empty non-nil response", send: []byte{0x1f, 0x8b, 0x08}, expect: []byte("")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, wc.Connection.WriteMessage(gws.BinaryMessage, tc.send), "WriteMessage must not error")
			resp := wc.ReadMessage()
			require.NotNil(t, resp.Raw, "Raw must not be nil; a nil response stops the reader")
			assert.Equal(t, tc.expect, resp.Raw, "ReadMessage should return the expected payload")
		})
	}
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

	connSetupThree := *connSetupOne
	connSetupThree.MessageFilter = "three"
	require.NoError(t, multi.SetupNewConnection(&connSetupThree), "SetupNewConnection must accept a shared URL under another message filter")
	urls, err = multi.GetConfiguredWebsocketURLs()
	require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
	assert.ElementsMatch(t, []string{"wss://one.example/ws", "wss://two.example/ws"}, urls, "GetConfiguredWebsocketURLs should list a shared URL once")
}

func TestConnectionShutdown(t *testing.T) {
	t.Parallel()
	var unset *connection
	assert.NoError(t, unset.Shutdown(), "Shutdown should not error for a nil connection")

	wc := connection{shutdown: make(chan struct{})}
	err := wc.Shutdown()
	assert.NoError(t, err, "Shutdown should not error when connection.Connection is nil")

	err = wc.Dial(t.Context(), &gws.Dialer{}, nil, nil)
	assert.ErrorContains(t, err, "malformed ws or wss URL", "Dial should error correctly")

	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))

	wc.URL = "ws" + mock.URL[len("http"):] + "/ws"

	err = wc.Dial(t.Context(), dialer, nil, nil)
	require.NoError(t, err, "Dial must not error")

	err = wc.Shutdown()
	require.NoError(t, err, "Shutdown must not error")
}

// stalledWriteConn simulates a peer which has stopped reading; once stalled, writes block until the connection is
// closed, as they do once the socket send buffer is full
type stalledWriteConn struct {
	net.Conn
	stalled   atomic.Bool
	blocked   chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
}

func (c *stalledWriteConn) Write(b []byte) (int, error) {
	if !c.stalled.Load() {
		return c.Conn.Write(b)
	}
	select {
	case c.blocked <- struct{}{}:
	default:
	}
	<-c.closed
	return 0, net.ErrClosed
}

func (c *stalledWriteConn) Close() error {
	err := net.ErrClosed
	c.closeOnce.Do(func() {
		close(c.closed)
		err = c.Conn.Close()
	})
	return err
}

// dialStalledWriteConn dials a connection to an echo server through a stalledWriteConn, beneath TLS if useTLS is set
func dialStalledWriteConn(t *testing.T, useTLS bool) (*connection, *stalledWriteConn) {
	t.Helper()
	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	conn := &stalledWriteConn{blocked: make(chan struct{}, 1), closed: make(chan struct{})}
	dialContext, scheme := dialer.NetDialContext, "ws"
	if useTLS {
		tr, ok := mock.Client().Transport.(*http.Transport)
		require.True(t, ok, "test server client transport must be an *http.Transport")
		dialer.TLSClientConfig, scheme = tr.TLSClientConfig, "wss"
		// The in-memory TLS dial returns its client before the handshake, so its connection can be stalled beneath TLS
		dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := tr.DialTLSContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			tlsConn, ok := c.(*tls.Conn)
			if !ok {
				return nil, fmt.Errorf("DialTLSContext returned %T, not *tls.Conn", c)
			}
			return tlsConn.NetConn(), nil
		}
	}
	dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		var err error
		conn.Conn, err = dialContext(ctx, network, addr)
		return conn, err
	}

	wc := &connection{URL: scheme + mock.URL[len("http"):] + "/ws"}
	require.NoError(t, wc.Dial(t.Context(), dialer, nil, nil), "Dial must not error")
	t.Cleanup(func() { _ = conn.Close() }) // Releases the write and Shutdown if Shutdown fails to close the connection
	return wc, conn
}

func TestConnectionShutdownWithStalledWrite(t *testing.T) {
	t.Parallel()

	wc, conn := dialStalledWriteConn(t, false)
	conn.stalled.Store(true)
	writeErr := make(chan error, 1)
	go func() { writeErr <- wc.SendRawMessage(t.Context(), request.Unset, gws.TextMessage, []byte("test")) }()
	select {
	case <-conn.blocked:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "SendRawMessage must reach the stalled write")
	}

	shutdownErr := make(chan error, 1)
	go func() { shutdownErr <- wc.Shutdown() }()
	select {
	case err := <-shutdownErr:
		require.NoError(t, err, "Shutdown must not error")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "Shutdown must not wait for a stalled write")
	}
	select {
	case err := <-writeErr:
		assert.ErrorIs(t, err, net.ErrClosed, "stalled write should error once the connection is closed")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "stalled write must return once the connection is closed")
	}
}

func TestConnectionShutdownTLSWithStalledPeer(t *testing.T) {
	t.Parallel()

	wc, conn := dialStalledWriteConn(t, true)
	conn.stalled.Store(true) // The peer has stopped reading, and no write is in progress
	shutdownErr := make(chan error, 1)
	go func() { shutdownErr <- wc.Shutdown() }()
	select {
	case err := <-shutdownErr:
		require.NoError(t, err, "Shutdown must not error")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "Shutdown must not write to a peer which has stopped reading")
	}
	select {
	case <-conn.closed:
	default:
		assert.Fail(t, "Shutdown should close the connection beneath TLS before returning")
	}
}

func TestConnectionShutdownWaitsForWriteInProgress(t *testing.T) {
	t.Parallel()

	for _, useTLS := range []bool{false, true} {
		t.Run(fmt.Sprintf("tls=%t", useTLS), func(t *testing.T) {
			t.Parallel()

			wc, conn := dialStalledWriteConn(t, useTLS)
			conn.stalled.Store(true) // The peer has stopped reading, so a TLS close_notify would block
			// Holding writeControl stands in for a write in progress, which a reconnect must not be able to overtake
			wc.writeControl.Lock()
			shutdownErr := make(chan error, 1)
			go func() { shutdownErr <- wc.Shutdown() }()
			select {
			case <-conn.closed:
			case <-time.After(5 * time.Second):
				wc.writeControl.Unlock()
				require.FailNow(t, "Shutdown must close the connection without waiting for writeControl")
			}
			select {
			case <-shutdownErr:
				wc.writeControl.Unlock()
				require.FailNow(t, "Shutdown must not return while a write holds writeControl")
			case <-time.After(100 * time.Millisecond):
			}
			wc.writeControl.Unlock()
			select {
			case err := <-shutdownErr:
				require.NoError(t, err, "Shutdown must not error")
			case <-time.After(5 * time.Second):
				require.FailNow(t, "Shutdown must return once writeControl is released")
			}
		})
	}
}

// TestLatency logic test
func TestLatency(t *testing.T) {
	t.Parallel()

	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))

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

	err := wc.Dial(t.Context(), dialer, http.Header{}, nil)
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

func TestWriteToConnRateLimitBarrierWithoutLimiter(t *testing.T) {
	t.Parallel()
	wc := connection{}
	wc.setConnectedStatus(true)
	contexts, err := request.NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	errs := make(chan error, 2)
	writes := make(chan struct{}, 2)
	go func() {
		errs <- wc.writeToConn(contexts[0], request.Unset, func() error {
			writes <- struct{}{}
			return nil
		})
	}()
	require.Never(t, func() bool { return len(writes) != 0 }, 10*time.Millisecond, time.Millisecond, "first participant must not write before its peer arrives")
	go func() {
		errs <- wc.writeToConn(contexts[1], request.Unset, func() error {
			writes <- struct{}{}
			return nil
		})
	}()
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
	require.Len(t, writes, 2)
}

func TestSendMessageBarrierRejectionRemovesSignature(t *testing.T) {
	t.Parallel()
	wc := connection{Match: NewMatch()}
	wc.setConnectedStatus(true)
	contexts, err := request.NewRateLimitBarrierContexts(t.Context(), 2)
	require.NoError(t, err)
	request.AbortRateLimitBarrier(contexts[1])
	_, err = wc.SendMessageReturnResponse(contexts[0], request.Unset, "signature", struct{}{})
	require.ErrorIs(t, err, request.ErrRateLimitBarrierRejected)
	_, err = wc.Match.Set("signature", 1)
	require.NoError(t, err)
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
	ws := Manager{verbose: true, ReadMessageErrors: make(chan error, 1), ShutdownC: make(chan struct{})}
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
			ShutdownC:    make(chan struct{}),
			TrafficAlert: make(chan struct{}, 1),
		}
	}

	t.Run("shutdown signal exits", func(t *testing.T) {
		ws := newManager()
		close(ws.ShutdownC)

		require.True(t, ws.observeTraffic(make(chan time.Time), nil))
	})

	t.Run("connecting keeps monitor alive", func(t *testing.T) {
		ws := newManager()
		ws.setState(connectingState)

		require.False(t, ws.observeTraffic(newTimeoutSignal(), nil))
	})

	t.Run("traffic keeps monitor alive", func(t *testing.T) {
		ws := newManager()
		ws.setState(connectedState)
		ws.TrafficAlert <- struct{}{}

		require.False(t, ws.observeTraffic(newTimeoutSignal(), nil))
	})

	t.Run("timeout invokes shutdown handler", func(t *testing.T) {
		ws := newManager()
		ws.setState(connectedState)

		shutdownCalled := false
		require.True(t, ws.observeTraffic(newTimeoutSignal(), func() {
			shutdownCalled = true
			ws.setState(disconnectedState)
		}))
		require.True(t, shutdownCalled, "timeout handler must be called when traffic is missing")
		require.Equal(t, disconnectedState, ws.state.Load())
	})

	t.Run("monitor traffic shell", func(t *testing.T) {
		ws := newManager()
		ws.trafficTimeout = time.Hour
		close(ws.ShutdownC)

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
	m.ShutdownC = make(chan struct{})
	require.NoError(t, m.Shutdown(), "Shutdown must not error with no connections")
	m.setState(connectedState)
	m.Conn = &struct{ *connection }{&connection{}}
	m.AuthConn = &struct{ *connection }{&connection{}}
	require.ErrorIs(t, m.Shutdown(), common.ErrTypeAssertFailure, "Shutdown must error with unhandled connection type")

	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))

	wsURL := "ws" + mock.URL[len("http"):] + "/ws"
	conn, resp, err := dialer.DialContext(t.Context(), wsURL, nil)
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

	gwsConnAuth, respAuth, err := dialer.DialContext(t.Context(), wsURL, nil)
	require.NoError(t, err, "DialContext must not error")
	defer respAuth.Body.Close()

	gwsConnUnAuth, respUnAuth, err := dialer.DialContext(t.Context(), wsURL, nil)
	require.NoError(t, err, "DialContext must not error")
	defer respUnAuth.Body.Close()

	m.connectionManager = nil
	authConn := &connection{Connection: gwsConnAuth, shutdown: m.ShutdownC}
	m.AuthConn = authConn
	unauthConn := &connection{Connection: gwsConnUnAuth, shutdown: m.ShutdownC}
	m.Conn = unauthConn

	m.setState(connectedState)
	require.NoError(t, m.Shutdown(), "Shutdown must not error with good connections")

	require.Equal(t, m.ShutdownC, authConn.shutdown, "shutdown channels must be the same after original shutdown channel is closed")
	require.Equal(t, m.ShutdownC, unauthConn.shutdown, "shutdown channels must be the same after original shutdown channel is closed")
}

func TestCreateConnectAndSubscribeRecordsPartialSubscriptions(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	mgr.useMultiConnectionManagement = true
	server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))

	ws := &websocket{subscriptions: subscription.NewStore(), setup: &ConnectionSetup{
		URL: "ws" + server.URL[len("http"):] + "/ws",
		Connector: func(ctx context.Context, conn Connection) error {
			return conn.Dial(ctx, dialer, nil, nil)
		},
		Subscriber: partialSubscriber(mgr),
		Handler:    func(context.Context, Connection, []byte) error { return nil },
	}}
	t.Cleanup(func() { cleanupManagedConnectionReaders(t, mgr, ws) })

	accepted := &subscription.Subscription{Channel: "accepted"}
	rejected := &subscription.Subscription{Channel: "rejected"}
	err := mgr.createConnectAndSubscribe(t.Context(), ws, subscription.List{accepted, rejected})
	require.ErrorIs(t, err, ErrSubscriptionFailure, "subscriber error must bubble as subscription failure")
	require.ErrorIs(t, err, errSubscriptionRejected, "must include wrapped subscriber error")
	require.Len(t, ws.connections, 1, "connection must be tracked by websocket")
	connStore := ws.connections[0].Subscriptions()
	assert.NotNil(t, connStore.Get(accepted), "accepted subscription should be recorded against the connection")
	assert.Nil(t, connStore.Get(rejected), "rejected subscription should not be recorded against the connection")
	assert.Equal(t, 1, connStore.Len(), "connection store should only hold the accepted subscription")

	require.NoError(t, ws.connections[0].Shutdown())
	delete(mgr.connections, ws.connections[0])
	ws.connections = nil
	mgr.Wg.Wait()

	accepted = &subscription.Subscription{Channel: "accepted-again"}
	rejected = &subscription.Subscription{Channel: "missing"}
	ws.setup.Subscriber = func(_ context.Context, c Connection, subs subscription.List) error {
		return mgr.AddSuccessfulSubscriptions(c, subs[0])
	}
	err = mgr.createConnectAndSubscribe(t.Context(), ws, subscription.List{accepted, rejected})
	require.ErrorIs(t, err, ErrSubscriptionFailure, "missing subscriptions must return subscription failure")
	require.ErrorIs(t, err, ErrSubscriptionsNotAdded, "missing subscriptions must return subs not added error")
	require.Len(t, ws.connections, 1, "connection must be tracked by websocket")
	connStore = ws.connections[0].Subscriptions()
	assert.NotNil(t, connStore.Get(accepted), "accepted subscription should be recorded against the connection")
	assert.Nil(t, connStore.Get(rejected), "missing subscription should not be recorded against the connection")
}

// lifecycleTestServer returns an echo server's websocket URL and a connector dialling it
func lifecycleTestServer(t *testing.T) (wsURL string, dial func(context.Context, Connection) error) {
	t.Helper()
	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	return "ws" + mock.URL[len("http"):] + "/ws", func(ctx context.Context, conn Connection) error { return conn.Dial(ctx, dialer, nil, nil) }
}

// lifecycleDropServer returns a server's websocket URL and a connector dialling it; the server closes a connection that
// sends "drop", as a venue dropping it would
func lifecycleDropServer(t *testing.T) (wsURL string, dial func(context.Context, Connection) error) {
	t.Helper()
	mock, dialer := mockws.NewTestServer(t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, p []byte, c *gws.Conn) error {
		if string(p) == "drop" {
			return c.Close()
		}
		return nil
	}))
	return "ws" + mock.URL[len("http"):] + "/ws", func(ctx context.Context, conn Connection) error { return conn.Dial(ctx, dialer, nil, nil) }
}

// newLifecycleTestManager returns a multi-connection manager with the given setups, and an hour's monitor delay and
// traffic timeout. Its connection monitor never starts, so tests drive optional retries themselves, and it shuts down
// when the test ends
func newLifecycleTestManager(t *testing.T, setups ...*ConnectionSetup) *Manager {
	t.Helper()
	m := NewManager()
	m.connectionMonitorRunning.Store(true)
	t.Cleanup(func() {
		m.setEnabled(false)
		// Bounded, as a manager left holding its lock by a failed test would otherwise block the cleanup
		err, ok := lifecycleWithin(t, func() error {
			err := m.Shutdown()
			m.Wg.Wait()
			return err
		})
		if !ok {
			assert.Fail(t, "Shutdown should return within five seconds")
		} else if err != nil {
			assert.ErrorIs(t, err, ErrNotConnected, "Shutdown should succeed or find the manager disconnected")
		}
	})
	setup := newDefaultSetup()
	setup.UseMultiConnectionManagement = true
	setup.ExchangeConfig.ConnectionMonitorDelay = time.Hour
	// A lull in traffic would otherwise shut the manager down mid-test, as newDefaultSetup's traffic timeout is a second
	setup.ExchangeConfig.WebsocketTrafficTimeout = time.Hour
	require.NoError(t, m.Setup(setup), "Setup must not error")
	for _, s := range setups {
		require.NoError(t, m.SetupNewConnection(s), "SetupNewConnection must not error")
	}
	return m
}

// lifecycleWithin returns what call returns, and false once call has not returned within five seconds: a call that
// retires connections waits for their readers while holding the manager's lock, so a reader that never reports it has
// stopped would otherwise block the test until go test's timeout. Call reads whatever the test checks before it returns,
// so the bound gives the manager's goroutines no extra time to change it
func lifecycleWithin[T any](t *testing.T, call func() T) (T, bool) {
	t.Helper()
	result := make(chan T, 1)
	go func() { result <- call() }()
	select {
	case r := <-result:
		return r, true
	case <-time.After(5 * time.Second):
		var zero T
		return zero, false
	}
}

// lifecycleTestSetup returns a connection setup on wsURL subscribing whatever generate returns through the manager's
// stores
func lifecycleTestSetup(m **Manager, wsURL string, filter any, connector func(context.Context, Connection) error, generate func() (subscription.List, error)) *ConnectionSetup {
	return &ConnectionSetup{
		URL:                   wsURL,
		MessageFilter:         filter,
		Connector:             connector,
		GenerateSubscriptions: generate,
		Handler:               func(context.Context, Connection, []byte) error { return nil },
		Subscriber: func(_ context.Context, conn Connection, subs subscription.List) error {
			return (*m).AddSuccessfulSubscriptions(conn, subs...)
		},
		Unsubscriber: func(_ context.Context, conn Connection, subs subscription.List) error {
			return (*m).RemoveSubscriptions(conn, subs...)
		},
	}
}

func TestConnectOptionalSetup(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var accountDials atomic.Int32
	var accountUp atomic.Bool
	accountDial := func(ctx context.Context, conn Connection) error {
		accountDials.Add(1)
		if !accountUp.Load() {
			return errDastardlyReason
		}
		return dial(ctx, conn)
	}
	publicSub := &subscription.Subscription{Channel: "public"}
	accountSub := &subscription.Subscription{Channel: "account"}
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", accountDial, func() (subscription.List, error) { return subscription.List{accountSub.Clone()}, nil })
	account.Optional = true
	m = newLifecycleTestManager(t,
		lifecycleTestSetup(&m, wsURL, "public", dial, func() (subscription.List, error) { return subscription.List{publicSub.Clone()}, nil }),
		account,
	)
	public, optional := m.connectionManager[0], m.connectionManager[1]

	start := time.Now()
	require.NoError(t, m.Connect(t.Context()), "Connect must not fail for an optional connection's failure")
	end := time.Now()
	assert.Equal(t, int32(1), accountDials.Load(), "Connect should dial the optional setup")
	assert.True(t, m.IsConnected(), "Connect should leave the required connections up")
	assert.Len(t, m.snapshotManagedConnections(public), 1, "Connect should keep the required connection")
	assert.NotNil(t, public.subscriptions.Get(publicSub), "Connect should keep the required connection's subscriptions")
	assert.Empty(t, m.snapshotManagedConnections(optional), "Connect should not keep a connection for the failed optional setup")
	assert.Equal(t, 1, optional.retryFailures, "Connect should count the optional setup's failure")
	assert.WithinRange(t, optional.retryAt, start.Add(time.Hour), end.Add(time.Hour), "Connect should retry the optional setup after the monitor delay")

	require.NoError(t, m.Shutdown(), "Shutdown must not error")
	require.NoError(t, m.Connect(t.Context()), "Connect must not error while the optional setup's retry is not due")
	assert.Equal(t, int32(1), accountDials.Load(), "Connect should not dial an optional setup before its retry is due")
	assert.Equal(t, 1, optional.retryFailures, "Connect should keep the optional setup's failures across a reconnection")

	m.m.Lock()
	optional.retryAt = time.Now().Add(-time.Second)
	m.m.Unlock()
	start = time.Now()
	m.retryOptionalSetups(t.Context())
	end = time.Now()
	assert.Equal(t, int32(2), accountDials.Load(), "retryOptionalSetups should dial a due optional setup")
	assert.Equal(t, 2, optional.retryFailures, "retryOptionalSetups should count a further failure")
	assert.WithinRange(t, optional.retryAt, start.Add(time.Hour), end.Add(time.Hour), "retryOptionalSetups should schedule the next retry")

	accountUp.Store(true)
	m.retryOptionalSetups(t.Context())
	assert.Equal(t, int32(2), accountDials.Load(), "retryOptionalSetups should not dial before the retry is due")
	m.m.Lock()
	optional.retryAt = time.Now().Add(-time.Second)
	m.m.Unlock()
	m.retryOptionalSetups(t.Context())
	assert.Equal(t, int32(3), accountDials.Load(), "retryOptionalSetups should dial a due optional setup")
	assert.Len(t, m.snapshotManagedConnections(optional), 1, "retryOptionalSetups should connect the optional setup")
	assert.NotNil(t, optional.subscriptions.Get(accountSub), "retryOptionalSetups should subscribe the optional setup")
	assert.Zero(t, optional.retryFailures, "retryOptionalSetups should clear the failures once connected")
	assert.Zero(t, optional.retryAt, "retryOptionalSetups should clear the retry once connected")

	m.retryOptionalSetups(t.Context())
	assert.Equal(t, int32(3), accountDials.Load(), "retryOptionalSetups should leave a connected optional setup alone")
}

func TestConnectionMonitorRetriesOptionalSetup(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var accountUp atomic.Bool
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", func(ctx context.Context, conn Connection) error {
		if !accountUp.Load() {
			return errDastardlyReason
		}
		return dial(ctx, conn)
	}, func() (subscription.List, error) { return subscription.List{{Channel: "account"}}, nil })
	account.Optional = true
	m = newLifecycleTestManager(t, lifecycleTestSetup(&m, wsURL, "public", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "public"}}, nil
	}), account)
	m.connectionMonitorDelay = 10 * time.Millisecond
	m.connectionMonitorRunning.Store(false)
	t.Cleanup(func() {
		m.setEnabled(false)
		assert.Eventually(t, func() bool { return !m.connectionMonitorRunning.Load() }, 5*time.Second, 10*time.Millisecond, "the connection monitor should stop once the manager is disabled")
	})
	public, optional := m.connectionManager[0], m.connectionManager[1]

	require.NoError(t, m.Connect(t.Context()), "Connect must not fail for an optional connection's failure")
	publicConnections := m.snapshotManagedConnections(public)
	accountUp.Store(true)
	assert.Eventually(t, func() bool {
		m.m.Lock()
		defer m.m.Unlock()
		return optional.retryFailures == 0 && len(m.snapshotManagedConnections(optional)) == 1
	}, 5*time.Second, 10*time.Millisecond, "the connection monitor should connect an optional setup once its retry is due")
	assert.Equal(t, publicConnections, m.snapshotManagedConnections(public), "the connection monitor should connect an optional setup without reconnecting the others")
}

func TestObserveConnectionIsolatesFaults(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "account"}}, nil
	})
	account.Optional = true
	m = newLifecycleTestManager(t, lifecycleTestSetup(&m, wsURL, "public", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "public"}}, nil
	}), account)
	public, optional := m.connectionManager[0], m.connectionManager[1]
	require.NoError(t, m.Connect(t.Context()), "Connect must not error")
	publicConnections := m.snapshotManagedConnections(public)
	accountConnections := m.snapshotManagedConnections(optional)
	require.Len(t, accountConnections, 1, "Connect must connect the optional setup")
	timer := time.NewTimer(time.Hour)
	t.Cleanup(func() { timer.Stop() })
	fault := func(conn Connection) error {
		return &connectionFaultError{connection: conn, err: fmt.Errorf("%w: %w", errDastardlyReason, errConnectionFault)}
	}

	m.ReadMessageErrors <- fault(accountConnections[0])
	assert.False(t, m.observeConnection(t.Context(), timer), "observeConnection should keep monitoring")
	assert.Empty(t, m.snapshotManagedConnections(optional), "observeConnection should retire the lost connection of an optional setup")
	assert.Equal(t, 1, optional.retryFailures, "observeConnection should retry an optional setup that lost its connection")
	assert.True(t, m.IsConnected(), "observeConnection should leave the manager connected when an optional connection is lost")
	assert.Equal(t, publicConnections, m.snapshotManagedConnections(public), "observeConnection should leave the other setups' connections up")

	m.ReadMessageErrors <- fault(accountConnections[0])
	assert.False(t, m.observeConnection(t.Context(), timer), "observeConnection should keep monitoring")
	assert.Equal(t, 1, optional.retryFailures, "observeConnection should ignore the fault of a connection already retired")
	assert.Equal(t, publicConnections, m.snapshotManagedConnections(public), "observeConnection should not reconnect for a connection already retired")

	m.ReadMessageErrors <- fault(publicConnections[0])
	assert.False(t, m.observeConnection(t.Context(), timer), "observeConnection should keep monitoring")
	assert.True(t, m.IsConnected(), "observeConnection should reconnect when a required connection is lost")
	assert.NotEqual(t, publicConnections, m.snapshotManagedConnections(public), "observeConnection should replace a lost required connection")
}

func TestObserveConnectionReconcilesDroppedFaults(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleDropServer(t)
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "account"}}, nil
	})
	account.Optional = true
	m = newLifecycleTestManager(t, lifecycleTestSetup(&m, wsURL, "public", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "public"}}, nil
	}), account)
	public, optional := m.connectionManager[0], m.connectionManager[1]
	require.NoError(t, m.Connect(t.Context()), "Connect must not error")
	publicConnections, accountConnections := m.snapshotManagedConnections(public), m.snapshotManagedConnections(optional)
	require.Len(t, publicConnections, 1, "Connect must connect the required setup")
	require.Len(t, accountConnections, 1, "Connect must connect the optional setup")
	drop := func(conn Connection) {
		require.NoError(t, conn.SendRawMessage(t.Context(), request.Unset, gws.TextMessage, []byte("drop")), "SendRawMessage must not error")
	}

	// The lost optional connection's fault fills the fault channel, so the required connection's fault is dropped
	drop(accountConnections[0])
	require.Eventually(t, func() bool { return len(m.ReadMessageErrors) == 1 }, 5*time.Second, time.Millisecond, "the lost optional connection must report its fault")
	drop(publicConnections[0])
	required, ok := publicConnections[0].(*connection)
	require.True(t, ok, "the required connection must be a *connection")
	require.Eventually(t, func() bool { return !required.IsConnected() }, 5*time.Second, time.Millisecond, "the required connection must be lost")
	timer := time.NewTimer(time.Hour)
	t.Cleanup(func() { timer.Stop() })
	assert.False(t, m.observeConnection(t.Context(), timer), "observeConnection should keep monitoring")
	assert.True(t, m.IsConnected(), "observeConnection should reconnect")
	assert.NotEqual(t, publicConnections, m.snapshotManagedConnections(public), "observeConnection should replace a lost required connection whose fault was dropped")
}

func TestConnectRollbackLeavesHandlersUnawaited(t *testing.T) {
	t.Parallel()
	upgrader := gws.Upgrader{}
	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if !assert.NoError(t, err, "Upgrade should not error") {
			return
		}
		defer c.Close()
		if err := c.WriteMessage(gws.TextMessage, []byte("greeting")); err != nil {
			return
		}
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	wsURL := "ws" + mock.URL[len("http"):] + "/ws"
	dial := func(ctx context.Context, conn Connection) error { return conn.Dial(ctx, dialer, nil, nil) }
	handling, abandon := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var m *Manager
	feeds := lifecycleTestSetup(&m, wsURL, "feeds", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "feeds"}}, nil
	})
	// A handler taking the manager's lock waits while Connect holds it, so Connect must not wait for its reader; the
	// handler gives up when abandoned, so a Connect that does wait still returns
	feeds.Handler = func(context.Context, Connection, []byte) error {
		once.Do(func() { close(handling) })
		found := make(chan error, 1)
		go func() {
			_, err := m.GetConnection("feeds")
			found <- err
		}()
		select {
		case <-found:
		case <-abandon:
		}
		return nil
	}
	m = newLifecycleTestManager(t, feeds, lifecycleTestSetup(&m, wsURL, "failing", func(context.Context, Connection) error {
		// Abandoning ends this wait too, so a greeting that never reaches the handler fails the test rather than
		// holding Connect until go test's timeout
		select {
		case <-handling:
		case <-abandon:
		}
		return errDastardlyReason
	}, func() (subscription.List, error) {
		return subscription.List{{Channel: "failing"}}, nil
	}))

	connected := make(chan error, 1)
	go func() { connected <- m.Connect(t.Context()) }()
	select {
	case err := <-connected:
		assert.ErrorIs(t, err, errDastardlyReason, "Connect should return the required connection's failure")
	case <-time.After(5 * time.Second):
		close(abandon)
		_, returned := lifecycleWithin(t, func() error { return <-connected })
		require.True(t, returned, "Connect must return once its handler and connector are abandoned")
		select {
		case <-handling:
			assert.Fail(t, "Connect should not wait for a reader whose handler waits on the manager's lock")
		default:
			assert.Fail(t, "The feeds connection's reader should hand its greeting to the handler")
		}
	}
	assert.Empty(t, m.snapshotManagedConnections(m.connectionManager[0]), "Connect should close the connections it made")
}

func TestConnectOptionalSetupRetiresConnections(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var dialled *connection
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", func(ctx context.Context, conn Connection) error {
		dialled, _ = conn.(*connection)
		return dial(ctx, conn)
	}, func() (subscription.List, error) {
		return subscription.List{{Channel: "account"}}, nil
	})
	account.Optional = true
	account.Authenticate = func(context.Context, Connection) error { return errDastardlyReason }
	m = newLifecycleTestManager(t, lifecycleTestSetup(&m, wsURL, "public", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "public"}}, nil
	}), account)
	m.SetCanUseAuthenticatedEndpoints(true)

	type connected struct {
		err        error
		account    []Connection
		registered int
	}
	got, ok := lifecycleWithin(t, func() connected {
		got := connected{err: m.Connect(t.Context()), account: m.snapshotManagedConnections(m.connectionManager[1])}
		m.connectionManagerMu.RLock()
		got.registered = len(m.connections)
		m.connectionManagerMu.RUnlock()
		return got
	})
	require.True(t, ok, "Connect must return within five seconds")
	require.NoError(t, got.err, "Connect must not fail for an optional connection's failure")
	require.NotNil(t, dialled, "Connect must dial the optional setup")
	assert.Empty(t, got.account, "Connect should close the connection of an optional setup that failed after dialling")
	assert.Equal(t, 1, got.registered, "Connect should unregister the closed connection of an optional setup")
	assert.True(t, socketClosed(dialled), "Connect should close the socket of an optional setup that failed after dialling")
	assert.Equal(t, 1, m.connectionManager[1].retryFailures, "Connect should count the failure of an optional setup that failed after dialling")
}

func TestFlushChannelsOptionalSetup(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var accountGenerations atomic.Int32
	var generationFails atomic.Bool
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", func(context.Context, Connection) error {
		return errDastardlyReason
	}, func() (subscription.List, error) {
		accountGenerations.Add(1)
		if generationFails.Load() {
			return nil, errDastardlyReason
		}
		return subscription.List{{Channel: "account"}}, nil
	})
	account.Optional = true
	publicSub := &subscription.Subscription{Channel: "public"}
	var publicSubs subscription.List
	m = newLifecycleTestManager(t,
		account,
		lifecycleTestSetup(&m, wsURL, "public", dial, func() (subscription.List, error) { return publicSubs.Clone(), nil }),
	)
	optional, public := m.connectionManager[0], m.connectionManager[1]
	publicSubs = subscription.List{{Channel: "first"}}

	require.NoError(t, m.Connect(t.Context()), "Connect must not fail for an optional connection's failure")
	require.Equal(t, int32(1), accountGenerations.Load(), "Connect must generate the optional setup's subscriptions")

	publicSubs = append(publicSubs, publicSub)
	require.NoError(t, m.FlushChannels(t.Context()), "FlushChannels must not error while the optional setup's retry is not due")
	assert.Equal(t, int32(1), accountGenerations.Load(), "FlushChannels should leave an optional setup to its retry")
	assert.NotNil(t, public.subscriptions.Get(publicSub), "FlushChannels should flush the other setups")

	m.m.Lock()
	optional.retryAt = time.Now().Add(-time.Second)
	m.m.Unlock()
	publicSubs = publicSubs[:1]
	require.NoError(t, m.FlushChannels(t.Context()), "FlushChannels must not fail for a due optional setup's failure")
	assert.Equal(t, int32(2), accountGenerations.Load(), "FlushChannels should retry a due optional setup")
	assert.Nil(t, public.subscriptions.Get(publicSub), "FlushChannels should flush the setups after a failed optional one")
	assert.Equal(t, 2, optional.retryFailures, "FlushChannels should count the optional setup's failure")
	assert.True(t, m.IsConnected(), "FlushChannels should leave the other connections up")

	generationFails.Store(true)
	m.m.Lock()
	optional.retryAt = time.Now().Add(-time.Second)
	m.m.Unlock()
	publicSubs = append(publicSubs, publicSub)
	require.NoError(t, m.FlushChannels(t.Context()), "FlushChannels must not fail for a due optional setup's failure to generate its subscriptions")
	assert.Equal(t, int32(3), accountGenerations.Load(), "FlushChannels should generate a due optional setup's subscriptions")
	assert.NotNil(t, public.subscriptions.Get(publicSub), "FlushChannels should flush the setups after an optional one that failed to generate its subscriptions")
	assert.Equal(t, 3, optional.retryFailures, "FlushChannels should count an optional setup's failure to generate its subscriptions")
}

func TestFlushChannelsConnectedOptionalSetup(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	errAccountSubscribe, errPublicGeneration := errors.New("account subscribe failed"), errors.New("public generation failed")
	errPublicSubscribe := errors.New("public subscribe failed")
	var accountGenerationErr, accountSubscribeErr, publicGenerationErr, publicSubscribeErr error
	accountSubs := subscription.List{{Channel: "account"}}
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", dial, func() (subscription.List, error) {
		if accountGenerationErr != nil {
			return nil, accountGenerationErr
		}
		return accountSubs.Clone(), nil
	})
	account.Optional = true
	subscriber := account.Subscriber
	account.Subscriber = func(ctx context.Context, conn Connection, subs subscription.List) error {
		if accountSubscribeErr != nil {
			return accountSubscribeErr
		}
		return subscriber(ctx, conn, subs)
	}
	publicSub, laterPublicSub := &subscription.Subscription{Channel: "public"}, &subscription.Subscription{Channel: "later"}
	var publicSubs subscription.List
	publicSetup := lifecycleTestSetup(&m, wsURL, "public", dial, func() (subscription.List, error) {
		return publicSubs.Clone(), publicGenerationErr
	})
	publicSubscriber := publicSetup.Subscriber
	publicSetup.Subscriber = func(ctx context.Context, conn Connection, subs subscription.List) error {
		if publicSubscribeErr != nil {
			return publicSubscribeErr
		}
		return publicSubscriber(ctx, conn, subs)
	}
	m = newLifecycleTestManager(t, account, publicSetup)
	optional, public := m.connectionManager[0], m.connectionManager[1]
	publicSubs = subscription.List{{Channel: "first"}}
	require.NoError(t, m.Connect(t.Context()), "Connect must not error")
	accountConnections := m.snapshotManagedConnections(optional)
	require.Len(t, accountConnections, 1, "Connect must connect the optional setup")

	accountGenerationErr = errDastardlyReason
	publicSubs = subscription.List{{Channel: "first"}, publicSub}
	assert.ErrorIs(t, m.FlushChannels(t.Context()), errDastardlyReason, "FlushChannels should report a connected optional setup's failure to generate its subscriptions")
	assert.NotNil(t, public.subscriptions.Get(publicSub), "FlushChannels should flush the setups after a connected optional one that failed")
	assert.Equal(t, accountConnections, m.snapshotManagedConnections(optional), "FlushChannels should leave a connected optional setup's connections up")
	assert.Zero(t, optional.retryFailures, "FlushChannels should not retry an optional setup that is still connected")

	accountGenerationErr, accountSubscribeErr = nil, errAccountSubscribe
	accountSubs = subscription.List{{Channel: "account"}, {Channel: "account two"}}
	publicSubs = subscription.List{{Channel: "first"}, publicSub, laterPublicSub}
	assert.ErrorIs(t, m.FlushChannels(t.Context()), errAccountSubscribe, "FlushChannels should report a connected optional setup's failure to subscribe")
	assert.NotNil(t, public.subscriptions.Get(laterPublicSub), "FlushChannels should flush the setups after a connected optional one that failed to subscribe")
	assert.Equal(t, accountConnections, m.snapshotManagedConnections(optional), "FlushChannels should leave a connected optional setup's connections up after a failure to subscribe")

	accountGenerationErr, publicGenerationErr = errDastardlyReason, errPublicGeneration
	err := m.FlushChannels(t.Context())
	assert.ErrorIs(t, err, errPublicGeneration, "FlushChannels should return a required setup's failure to generate its subscriptions")
	assert.ErrorIs(t, err, errDastardlyReason, "FlushChannels should keep an earlier optional setup's failure beside a required setup's failure to generate")

	publicGenerationErr, publicSubscribeErr = nil, errPublicSubscribe
	publicSubs = subscription.List{{Channel: "first"}, publicSub, laterPublicSub, {Channel: "last"}}
	err = m.FlushChannels(t.Context())
	assert.ErrorIs(t, err, errPublicSubscribe, "FlushChannels should return a required setup's failure to subscribe")
	assert.ErrorIs(t, err, errDastardlyReason, "FlushChannels should keep an earlier optional setup's failure beside a required setup's failure to subscribe")
}

func TestOptionalRetryDelay(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		monitorDelay time.Duration
		failures     int
		exp          time.Duration
	}{
		{monitorDelay: 2 * time.Second, failures: 1, exp: 2 * time.Second},
		{monitorDelay: 2 * time.Second, failures: 2, exp: 4 * time.Second},
		{monitorDelay: 2 * time.Second, failures: 5, exp: 32 * time.Second},
		{monitorDelay: 2 * time.Second, failures: 6, exp: time.Minute},
		{monitorDelay: 2 * time.Second, failures: 1000, exp: time.Minute},
		{monitorDelay: 25 * time.Second, failures: 2, exp: 50 * time.Second},
		{monitorDelay: 25 * time.Second, failures: 3, exp: time.Minute},
		{monitorDelay: 90 * time.Second, failures: 1, exp: 90 * time.Second},
		{monitorDelay: 90 * time.Second, failures: 4, exp: 90 * time.Second},
	} {
		m := &Manager{connectionMonitorDelay: tc.monitorDelay}
		assert.Equalf(t, tc.exp, m.optionalRetryDelay(tc.failures), "optionalRetryDelay should wait %s after %d failures with a %s monitor delay", tc.exp, tc.failures, tc.monitorDelay)
	}
}

// TestOptionalSetupFailed fails an optional setup four times in a row with a monitor delay short of the minute ceiling,
// so each retry's delay depends on the failures counted
func TestOptionalSetupFailed(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "account"}}, nil
	})
	account.Optional = true
	m = newLifecycleTestManager(t, account)
	m.connectionMonitorDelay = 20 * time.Second
	optional := m.connectionManager[0]
	for i, delay := range []time.Duration{20 * time.Second, 40 * time.Second, time.Minute, time.Minute} {
		m.m.Lock()
		start := time.Now()
		err := m.optionalSetupFailed(0, optional, errDastardlyReason)
		end := time.Now()
		failures, retryAt := optional.retryFailures, optional.retryAt
		m.m.Unlock()
		assert.ErrorIsf(t, err, errDastardlyReason, "optionalSetupFailed should return the cause of failure %d", i+1)
		assert.ErrorContainsf(t, err, "retried in "+delay.String()+":", "optionalSetupFailed should report the retry it schedules after failure %d", i+1)
		assert.Equalf(t, i+1, failures, "optionalSetupFailed should count failure %d", i+1)
		assert.WithinRangef(t, retryAt, start.Add(delay), end.Add(delay), "optionalSetupFailed should retry %s after failure %d", delay, i+1)
	}
}

// shutdownWaitingConnection is a connection whose Shutdown waits for the test's release, so a test can read the clock
// while an optional setup's connections are being retired
type shutdownWaitingConnection struct {
	*fakeConnection
	shuttingDown, release chan struct{}
}

func (c *shutdownWaitingConnection) Shutdown() error {
	close(c.shuttingDown)
	<-c.release
	return nil
}

// TestOptionalSetupFailedSchedulesAfterRetiring fails an optional setup whose connection is slow to shut down, and
// checks that the delay before its retry counts from no earlier than that shutdown, not from when the call began
func TestOptionalSetupFailedSchedulesAfterRetiring(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "account"}}, nil
	})
	account.Optional = true
	m = newLifecycleTestManager(t, account)
	optional := m.connectionManager[0]
	conn := &shutdownWaitingConnection{
		fakeConnection: &fakeConnection{subscriptions: subscription.NewStore()},
		shuttingDown:   make(chan struct{}),
		release:        make(chan struct{}),
	}
	release := sync.OnceFunc(func() { close(conn.release) })
	t.Cleanup(release)
	m.trackConnection(conn, optional)
	failed := make(chan error, 1)
	go func() {
		m.m.Lock()
		defer m.m.Unlock()
		failed <- m.optionalSetupFailed(0, optional, errDastardlyReason)
	}()
	select {
	case <-conn.shuttingDown:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "optionalSetupFailed must shut the optional setup's connection down")
	}
	retired := time.Now()
	release()
	err, ok := lifecycleWithin(t, func() error { return <-failed })
	require.True(t, ok, "optionalSetupFailed must return once the connection has shut down")
	end := time.Now()
	assert.ErrorIs(t, err, errDastardlyReason, "optionalSetupFailed should return the cause")
	assert.Empty(t, m.snapshotManagedConnections(optional), "optionalSetupFailed should retire the optional setup's connection")
	m.m.Lock()
	retryAt := optional.retryAt
	m.m.Unlock()
	assert.WithinRange(t, retryAt, retired.Add(time.Hour), end.Add(time.Hour), "optionalSetupFailed should count the monitor delay from no earlier than the connection's shutdown")
}

func TestConnectionShutdownStopsPingHandler(t *testing.T) {
	t.Parallel()
	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	m := NewManager()
	m.useMultiConnectionManagement = true
	conn := m.createConnectionFromSetup(&ConnectionSetup{URL: "ws" + mock.URL[len("http"):] + "/ws"})
	var pings sync.WaitGroup
	conn.Wg = &pings
	require.NoError(t, conn.Dial(t.Context(), dialer, nil, nil), "Dial must not error")
	conn.SetupPingHandler(request.Unset, PingHandler{MessageType: gws.TextMessage, Message: []byte(Ping), Delay: time.Hour})
	require.NoError(t, conn.Shutdown(), "Shutdown must not error")
	_ = conn.Shutdown() // A repeated shutdown must not close the closed signal again; the socket is already closed
	done := make(chan struct{})
	go func() {
		pings.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "a managed connection's ping work must stop when the connection shuts down")
	}
}

func TestLegacyConnectionPingsAfterRedial(t *testing.T) {
	t.Parallel()
	received := make(chan string, 8)
	mock, dialer := mockws.NewTestServer(t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, p []byte, _ *gws.Conn) error {
		select {
		case received <- string(p):
		default:
		}
		return nil
	}))
	m := NewManager()
	conn := m.createConnectionFromSetup(&ConnectionSetup{URL: "ws" + mock.URL[len("http"):] + "/ws"})
	var pings sync.WaitGroup
	conn.Wg = &pings
	t.Cleanup(func() {
		close(m.ShutdownC)
		_ = conn.Shutdown()
		pings.Wait()
	})
	// A legacy connection is dialled again on each reconnection, so shutting it down must not stop its later pings
	require.NoError(t, conn.Dial(t.Context(), dialer, nil, nil), "Dial must not error")
	require.NoError(t, conn.Shutdown(), "Shutdown must not error")
	require.NoError(t, conn.Dial(t.Context(), dialer, nil, nil), "Dial must not error after a shutdown")
	conn.SetupPingHandler(request.Unset, PingHandler{MessageType: gws.TextMessage, Message: []byte(Ping), Delay: time.Millisecond})
	select {
	case message := <-received:
		assert.Equal(t, Ping, message, "a legacy connection should ping once dialled again")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "a legacy connection must ping once dialled again after a shutdown")
	}
}

// retryDialTest holds the retry of an optional setup that failed to connect inside its connector, so a test can act on
// the manager while the retry is dialling
type retryDialTest struct {
	m                 *Manager
	public, optional  *websocket
	publicConnections []Connection
	// dialled is the connection the retry's connector dialled, and ownGroup whether the connector ran with a wait group
	// other than the manager's
	dialled  *connection
	ownGroup bool
	// routineDone is closed once the routine the retry's connector started on the connection's wait group finishes
	routineDone chan struct{}
	// release ends the retry's connector with the error it returns, and retried is closed once the retry returns
	release chan error
	retried chan struct{}
	// laterDialsFail fails the dials after the retry's, and generationFails and generatesNothing fail the optional
	// setup's subscriptions to generate or leave it nothing to subscribe
	laterDialsFail, generationFails, generatesNothing atomic.Bool
}

// newRetryDialTest returns a manager whose required setup connected and whose optional setup, adjusted by configure when
// given, failed to, once the optional setup's retry has dialled. The retry's connector then starts a routine on the
// connection's wait group, which finishes a little after the connection closes, and waits to be released
func newRetryDialTest(t *testing.T, configure func(*ConnectionSetup)) *retryDialTest {
	t.Helper()
	wsURL, dial := lifecycleDropServer(t)
	h := &retryDialTest{
		routineDone: make(chan struct{}),
		release:     make(chan error, 1),
		retried:     make(chan struct{}),
	}
	entered := make(chan *connection, 1)
	var dials atomic.Int32
	account := lifecycleTestSetup(&h.m, wsURL, "account", func(ctx context.Context, conn Connection) error {
		switch dials.Add(1) {
		case 1:
			return errDastardlyReason // Connect's dial fails, so the setup is retried
		case 2:
			return h.retryDial(ctx, conn, dial, entered)
		}
		if h.laterDialsFail.Load() {
			return errDastardlyReason
		}
		return dial(ctx, conn)
	}, func() (subscription.List, error) {
		switch {
		case h.generationFails.Load():
			return nil, errDastardlyReason
		case h.generatesNothing.Load():
			return nil, nil
		}
		return subscription.List{{Channel: "account"}}, nil
	})
	account.Optional = true
	if configure != nil {
		configure(account)
	}
	h.m = newLifecycleTestManager(t, lifecycleTestSetup(&h.m, wsURL, "public", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "public"}}, nil
	}), account)
	h.public, h.optional = h.m.connectionManager[0], h.m.connectionManager[1]
	require.NoError(t, h.m.Connect(t.Context()), "Connect must not error for an optional setup's failure")
	require.Equal(t, 1, h.optional.retryFailures, "Connect must count the optional setup's failure")
	h.publicConnections = h.m.snapshotManagedConnections(h.public)
	require.Len(t, h.publicConnections, 1, "Connect must connect the required setup")
	h.m.m.Lock()
	h.optional.retryAt = time.Now().Add(-time.Second)
	h.m.m.Unlock()

	// Released here too, so a failed test cannot leave the retry waiting, and awaited before the manager shuts down
	t.Cleanup(func() {
		select {
		case h.release <- nil:
		default:
		}
		select {
		case <-h.retried:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "the retry should return once its connector is released")
		}
	})
	ctx := t.Context()
	go func() {
		defer close(h.retried)
		h.m.retryOptionalSetups(ctx)
	}()
	select {
	case h.dialled = <-entered:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the retry must dial the optional setup")
	}
	return h
}

// retryDial is the retry's connector: it dials, starts a routine on the connection's wait group and waits to be released
func (h *retryDialTest) retryDial(ctx context.Context, conn Connection, dial func(context.Context, Connection) error, entered chan<- *connection) error {
	c, ok := conn.(*connection)
	if !ok {
		return common.GetTypeAssertError("*connection", conn)
	}
	if err := dial(ctx, c); err != nil {
		return err
	}
	h.ownGroup = c.Wg != &h.m.Wg
	c.Wg.Go(func() {
		select {
		case <-c.closed:
		case <-c.shutdown:
		}
		time.Sleep(100 * time.Millisecond) // A routine still finishing after its connection closes
		close(h.routineDone)
	})
	entered <- c
	select {
	case err := <-h.release:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// within returns what call returns, which must be within five seconds while the retry dials
func (h *retryDialTest) within(t *testing.T, name string, call func() error) error {
	t.Helper()
	err, ok := lifecycleWithin(t, call)
	require.Truef(t, ok, "%s must return while the retry dials", name)
	return err
}

func TestRetryOptionalSetupsDialsOutsideLock(t *testing.T) {
	t.Parallel()
	shutdown := func(t *testing.T, h *retryDialTest) {
		t.Helper()
		require.NoError(t, h.within(t, "Shutdown", h.m.Shutdown), "Shutdown must not error")
	}
	for _, tc := range []struct {
		name      string
		configure func(*ConnectionSetup)
		// act runs while the retry dials, and dialErr is what the retry's connector then returns
		act     func(*testing.T, *retryDialTest)
		dialErr error
		// kept is whether the optional setup keeps the retry's connection
		kept                   bool
		connections, failures  int
		rescheduled            bool
		connected, reconnected bool
	}{
		{
			name: "GetConnection meanwhile",
			act: func(t *testing.T, h *retryDialTest) {
				t.Helper()
				type found struct {
					public                Connection
					publicErr, accountErr error
				}
				got, ok := lifecycleWithin(t, func() (f found) {
					f.public, f.publicErr = h.m.GetConnection("public")
					_, f.accountErr = h.m.GetConnection("account")
					return f
				})
				require.True(t, ok, "GetConnection must return while the retry dials")
				require.NoError(t, got.publicErr, "GetConnection must not error for a connected setup")
				assert.Equal(t, h.publicConnections[0], got.public, "GetConnection should return another setup's connection while the retry dials")
				assert.ErrorIs(t, got.accountErr, ErrNotConnected, "GetConnection should not return a connection still being dialled")
			},
			kept: true, connections: 1, connected: true,
		},
		{
			name: "setup without subscriptions",
			configure: func(s *ConnectionSetup) {
				s.SubscriptionsNotRequired, s.GenerateSubscriptions = true, nil
			},
			kept: true, connections: 1, connected: true,
		},
		{
			name:     "Shutdown meanwhile",
			act:      shutdown,
			failures: 1,
		},
		{
			name: "FlushChannels past its deadline meanwhile",
			act: func(t *testing.T, h *retryDialTest) {
				t.Helper()
				ctx, cancel := context.WithDeadline(t.Context(), time.Now())
				defer cancel()
				assert.ErrorIs(t, h.within(t, "FlushChannels", func() error { return h.m.FlushChannels(ctx) }), context.DeadlineExceeded, "FlushChannels should return its context's error")
			},
			kept: true, connections: 1, connected: true,
		},
		{
			name: "SetProxyAddress meanwhile",
			act: func(t *testing.T, h *retryDialTest) {
				t.Helper()
				require.NoError(t, h.within(t, "SetProxyAddress", func() error { return h.m.SetProxyAddress(t.Context(), "") }), "SetProxyAddress must not error")
			},
			connections: 1, connected: true, reconnected: true,
		},
		{
			name: "Disable meanwhile",
			act: func(t *testing.T, h *retryDialTest) {
				t.Helper()
				require.NoError(t, h.m.Disable(), "Disable must not error")
			},
			failures: 1, connected: true,
		},
		{
			name: "FlushChannels connecting the setup meanwhile",
			act: func(t *testing.T, h *retryDialTest) {
				t.Helper()
				require.NoError(t, h.within(t, "FlushChannels", func() error { return h.m.FlushChannels(t.Context()) }), "FlushChannels must not error")
			},
			connections: 1, connected: true,
		},
		{
			name:     "failed dial",
			dialErr:  errDastardlyReason,
			failures: 2, rescheduled: true, connected: true,
		},
		{
			name:     "failed dial after a shutdown meanwhile",
			act:      shutdown,
			dialErr:  errDastardlyReason,
			failures: 1,
		},
		{
			name: "reconnection meanwhile leaving the setup due",
			act: func(t *testing.T, h *retryDialTest) {
				t.Helper()
				shutdown(t, h)
				h.laterDialsFail.Store(true)
				require.NoError(t, h.within(t, "Connect", func() error { return h.m.Connect(t.Context()) }), "Connect must not error for an optional setup's failure")
				h.m.m.Lock()
				failures := h.optional.retryFailures
				h.optional.retryAt = time.Now().Add(-time.Second) // As if the delay before its next retry passed meanwhile
				h.m.m.Unlock()
				require.Equal(t, 2, failures, "Connect must count the optional setup's failure")
			},
			failures: 2, connected: true, reconnected: true,
		},
		{
			name: "connection lost meanwhile",
			act: func(t *testing.T, h *retryDialTest) {
				t.Helper()
				read := make(chan Response, 1)
				go func() { read <- h.dialled.ReadMessage() }()
				require.NoError(t, h.dialled.SendRawMessage(t.Context(), request.Unset, gws.TextMessage, []byte("drop")), "SendRawMessage must not error")
				select {
				case resp := <-read:
					require.Nil(t, resp.Raw, "the connection being dialled must be lost")
				case <-time.After(5 * time.Second):
					require.FailNow(t, "the connection being dialled must be lost")
				}
				timer := time.NewTimer(time.Hour)
				t.Cleanup(func() { timer.Stop() })
				exit, ok := lifecycleWithin(t, func() bool { return h.m.observeConnection(t.Context(), timer) })
				require.True(t, ok, "observeConnection must return while the retry dials")
				assert.False(t, exit, "observeConnection should keep monitoring")
				assert.True(t, h.m.IsConnected(), "observeConnection should leave the manager connected when a connection being dialled is lost")
				assert.Equal(t, h.publicConnections, h.m.snapshotManagedConnections(h.public), "observeConnection should leave the other setups' connections up when a connection being dialled is lost")
			},
			failures: 2, rescheduled: true, connected: true,
		},
		{
			name: "nothing generated once dialled",
			act: func(_ *testing.T, h *retryDialTest) {
				h.generatesNothing.Store(true)
			},
			connected: true,
		},
		{
			name: "generation failing once dialled",
			act: func(_ *testing.T, h *retryDialTest) {
				h.generationFails.Store(true)
			},
			failures: 2, rescheduled: true, connected: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newRetryDialTest(t, tc.configure)
			assert.True(t, h.ownGroup, "the retry's connector should start routines on a wait group of its own, as a shutdown meanwhile waits on the manager's")
			if tc.act != nil {
				tc.act(t, h)
			}
			h.release <- tc.dialErr
			select {
			case <-h.retried:
			case <-time.After(5 * time.Second):
				require.FailNow(t, "the retry must return once its connector is released")
			}

			m := h.m
			assert.Same(t, &m.Wg, h.dialled.Wg, "the retry should leave the routines its connection starts once dialled to the manager's wait group")
			connections := m.snapshotManagedConnections(h.optional)
			assert.Len(t, connections, tc.connections, "the optional setup should hold the connections made for it")
			assert.Equal(t, tc.kept, slices.Contains(connections, Connection(h.dialled)), "the retry should keep its connection only while nothing has changed the manager or the setup")
			_, registered := m.managedWebsocket(h.dialled)
			assert.Equal(t, tc.kept, registered, "the retry should register only a connection it keeps")
			var closed bool
			select {
			case <-h.dialled.closed:
				closed = true
			default:
			}
			assert.Equal(t, !tc.kept, closed, "the retry should shut down a connection it does not keep")
			if !tc.kept {
				// Its socket is closed too, not just the channel its routines stop on
				assert.True(t, socketClosed(h.dialled), "the retry should close the socket of a connection it does not keep")
			}
			if tc.kept && !h.optional.setup.SubscriptionsNotRequired {
				assert.NotNil(t, h.optional.subscriptions.Get(&subscription.Subscription{Channel: "account"}), "the retry should subscribe the optional setup's generated subscriptions")
			}
			m.m.Lock()
			failures, retryAt := h.optional.retryFailures, h.optional.retryAt
			m.m.Unlock()
			assert.Equal(t, tc.failures, failures, "the retry should count a failure only for a connection it would keep")
			assert.Equal(t, tc.rescheduled, retryAt.After(time.Now()), "the retry should schedule the next retry only after a failure it counts")
			if tc.failures == 0 {
				assert.Zero(t, retryAt, "the retry time should be cleared once the setup connects")
			}
			assert.Equal(t, tc.connected, m.IsConnected(), "the manager should be connected only when nothing shut it down")
			if tc.connected && tc.reconnected {
				assert.NotEqual(t, h.publicConnections, m.snapshotManagedConnections(h.public), "the manager should have reconnected the other setups")
			} else if tc.connected {
				assert.Equal(t, h.publicConnections, m.snapshotManagedConnections(h.public), "the retry should leave the other setups' connections up")
			}

			type stopped struct {
				err         error
				routineDone bool
			}
			got, ok := lifecycleWithin(t, func() (s stopped) {
				if m.IsConnected() {
					s.err = m.Shutdown()
				}
				m.Wg.Wait()
				select {
				case <-h.routineDone:
					s.routineDone = true
				default:
				}
				return s
			})
			require.True(t, ok, "the manager must stop within five seconds")
			require.NoError(t, got.err, "Shutdown must not error")
			assert.True(t, got.routineDone, "stopping the manager should wait for the routines the retry's connector started")
		})
	}
}

func TestRetryOptionalSetupsSettlesWithoutDialling(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var accountDials atomic.Int32
	var generationFails, generatesNothing atomic.Bool
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", func(context.Context, Connection) error {
		accountDials.Add(1)
		return errDastardlyReason
	}, func() (subscription.List, error) {
		switch {
		case generationFails.Load():
			return nil, errDastardlyReason
		case generatesNothing.Load():
			return nil, nil
		}
		return subscription.List{{Channel: "account"}}, nil
	})
	account.Optional = true
	m = newLifecycleTestManager(t, lifecycleTestSetup(&m, wsURL, "public", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "public"}}, nil
	}), account)
	optional := m.connectionManager[1]
	require.NoError(t, m.Connect(t.Context()), "Connect must not error for an optional setup's failure")
	require.Equal(t, int32(1), accountDials.Load(), "Connect must dial the optional setup")
	due := func() {
		m.m.Lock()
		optional.retryAt = time.Now().Add(-time.Second)
		m.m.Unlock()
	}

	generationFails.Store(true)
	due()
	m.retryOptionalSetups(t.Context())
	assert.Equal(t, int32(1), accountDials.Load(), "retryOptionalSetups should not dial a setup whose subscriptions fail to generate")
	assert.Equal(t, 2, optional.retryFailures, "retryOptionalSetups should count a setup's failure to generate its subscriptions")
	assert.True(t, optional.retryAt.After(time.Now()), "retryOptionalSetups should schedule the next retry of a setup whose subscriptions fail to generate")

	generationFails.Store(false)
	generatesNothing.Store(true)
	due()
	m.retryOptionalSetups(t.Context())
	assert.Equal(t, int32(1), accountDials.Load(), "retryOptionalSetups should not dial a setup with nothing to subscribe")
	assert.Zero(t, optional.retryFailures, "retryOptionalSetups should clear the failures of a setup with nothing to subscribe")
	assert.Zero(t, optional.retryAt, "retryOptionalSetups should clear the retry of a setup with nothing to subscribe")
}

// socketClosed reports whether a connection's socket is closed: a read then fails at once rather than waiting out its
// deadline
func socketClosed(c *connection) bool {
	netConn := c.Connection.NetConn()
	if err := netConn.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		return true
	}
	_, err := netConn.Read(make([]byte, 1))
	switch {
	case err == nil, errors.Is(err, os.ErrDeadlineExceeded):
		return false
	default:
		return true
	}
}

// TestConnectVerboseSetups connects, with verbose logging on, a setup split over connections whose second batch is
// tracked on the first connection, a setup without subscriptions, one generating none, one whose subscriptions are all
// tracked on a connection it already has, and an optional setup whose retry is not due
func TestConnectVerboseSetups(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	first, second := &subscription.Subscription{Channel: "first"}, &subscription.Subscription{Channel: "second"}
	var m *Manager
	batched := lifecycleTestSetup(&m, wsURL, "batched", dial, func() (subscription.List, error) {
		return subscription.List{first, second}, nil
	})
	batched.TrackOnExistingConnection = func(_ context.Context, _ Connection, subs subscription.List) (remaining, tracked subscription.List, err error) {
		return nil, subs, nil
	}
	requests := lifecycleTestSetup(&m, wsURL, "requests", dial, nil)
	requests.SubscriptionsNotRequired = true
	empty := lifecycleTestSetup(&m, wsURL, "empty", dial, func() (subscription.List, error) { return nil, nil })
	trackedSub := &subscription.Subscription{Channel: "tracked"}
	tracked := lifecycleTestSetup(&m, wsURL, "tracked", func(context.Context, Connection) error {
		return errors.New("a setup whose subscriptions are all tracked should not be dialled")
	}, func() (subscription.List, error) { return subscription.List{trackedSub}, nil })
	tracked.TrackOnExistingConnection = batched.TrackOnExistingConnection
	optional := lifecycleTestSetup(&m, wsURL, "optional", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "optional"}}, nil
	})
	optional.Optional = true
	m = newLifecycleTestManager(t, batched, requests, empty, tracked, optional)
	m.verbose = true
	m.MaxSubscriptionsPerConnection = 1
	existing := &fakeConnection{subscriptions: subscription.NewStore()}
	m.trackConnection(existing, m.connectionManager[3])
	m.connectionManager[4].retryFailures, m.connectionManager[4].retryAt = 1, time.Now().Add(time.Hour)

	require.NoError(t, m.Connect(t.Context()), "Connect must not error")
	batchedConnections := m.snapshotManagedConnections(m.connectionManager[0])
	require.Len(t, batchedConnections, 1, "Connect must dial one connection for a batch tracked on the first")
	assert.NotNil(t, batchedConnections[0].Subscriptions().Get(second), "Connect should track the second batch on the first connection")
	assert.Len(t, m.snapshotManagedConnections(m.connectionManager[1]), 1, "Connect should connect a setup without subscriptions")
	assert.Empty(t, m.snapshotManagedConnections(m.connectionManager[2]), "Connect should skip a setup generating no subscriptions")
	assert.NotNil(t, m.connectionManager[3].subscriptions.Get(trackedSub), "Connect should track the subscriptions of a setup on its existing connection")
	assert.Empty(t, m.snapshotManagedConnections(m.connectionManager[4]), "Connect should skip an optional setup whose retry is not due")
	assert.Equal(t, 1, m.connectionManager[4].retryFailures, "Connect should keep the failures of an optional setup whose retry is not due")
}

// TestConnectSetupTrackingFailures fails tracking subscriptions on a setup's existing connections, before the setup's
// subscriptions are batched and for a later batch, which connect reports as subscription errors
func TestConnectSetupTrackingFailures(t *testing.T) {
	t.Parallel()
	refuse := func(context.Context, Connection, subscription.List) (remaining, tracked subscription.List, err error) {
		return nil, nil, errDastardlyReason
	}
	t.Run("existing connection", func(t *testing.T) {
		t.Parallel()
		wsURL, dial := lifecycleTestServer(t)
		var m *Manager
		setup := lifecycleTestSetup(&m, wsURL, "feeds", dial, func() (subscription.List, error) {
			return subscription.List{{Channel: "feeds"}}, nil
		})
		setup.TrackOnExistingConnection = refuse
		m = newLifecycleTestManager(t, setup)
		m.trackConnection(&fakeConnection{subscriptions: subscription.NewStore()}, m.connectionManager[0])

		assert.ErrorIs(t, m.Connect(t.Context()), errDastardlyReason, "Connect should report a failure to track subscriptions on an existing connection")
		assert.True(t, m.IsConnected(), "Connect should stay connected after a subscription error")
		assert.Len(t, m.snapshotManagedConnections(m.connectionManager[0]), 1, "Connect should dial nothing for subscriptions it failed to track")
	})
	t.Run("later batch", func(t *testing.T) {
		t.Parallel()
		wsURL, dial := lifecycleTestServer(t)
		first, second := &subscription.Subscription{Channel: "first"}, &subscription.Subscription{Channel: "second"}
		var m *Manager
		setup := lifecycleTestSetup(&m, wsURL, "feeds", dial, func() (subscription.List, error) {
			return subscription.List{first, second}, nil
		})
		setup.TrackOnExistingConnection = refuse
		m = newLifecycleTestManager(t, setup)
		m.MaxSubscriptionsPerConnection = 1

		assert.ErrorIs(t, m.Connect(t.Context()), errDastardlyReason, "Connect should report a failure to track a later batch on the first connection")
		assert.True(t, m.IsConnected(), "Connect should stay connected after a subscription error")
		assert.Len(t, m.snapshotManagedConnections(m.connectionManager[0]), 1, "Connect should dial no connection for a batch it failed to track")
		assert.NotNil(t, m.connectionManager[0].subscriptions.Get(first), "Connect should subscribe the first batch")
		assert.Nil(t, m.connectionManager[0].subscriptions.Get(second), "Connect should not subscribe a batch it failed to track")
	})
}

// TestRetireConnectionsShutdownErrors retires connections whose sockets are already closed, by a rollback and directly,
// which reports the shutdown errors and still unregisters the connections
func TestRetireConnectionsShutdownErrors(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var m *Manager
	failing := lifecycleTestSetup(&m, wsURL, "failing", func(context.Context, Connection) error {
		// The first setup's connection is closed under the rollback, so retiring it fails to shut it down
		for _, conn := range m.snapshotManagedConnections(m.connectionManager[0]) {
			if err := conn.Shutdown(); err != nil {
				return err
			}
		}
		return errDastardlyReason
	}, func() (subscription.List, error) { return subscription.List{{Channel: "failing"}}, nil })
	m = newLifecycleTestManager(t, lifecycleTestSetup(&m, wsURL, "first", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "first"}}, nil
	}), failing)

	assert.ErrorIs(t, m.Connect(t.Context()), errDastardlyReason, "Connect should return the required setup's failure")
	assert.False(t, m.IsConnected(), "Connect should leave the manager disconnected")
	assert.Empty(t, m.snapshotManagedConnections(m.connectionManager[0]), "Connect should unregister a connection whose shutdown fails during the rollback")
	m.connectionManagerMu.RLock()
	registered := len(m.connections)
	m.connectionManagerMu.RUnlock()
	assert.Zero(t, registered, "Connect should leave no connection registered after the rollback")

	m.connectionManager[1].setup.Connector = dial
	require.NoError(t, m.Connect(t.Context()), "Connect must not error")
	ws := m.connectionManager[0]
	connections := m.snapshotManagedConnections(ws)
	require.Len(t, connections, 1, "Connect must connect the first setup")
	require.NoError(t, connections[0].Shutdown(), "Shutdown must not error")
	m.m.Lock()
	err := m.retireConnections(ws, connections)
	m.m.Unlock()
	assert.ErrorIs(t, err, net.ErrClosed, "retireConnections should report a connection it fails to shut down")
	assert.Empty(t, m.snapshotManagedConnections(ws), "retireConnections should unregister a connection it fails to shut down")
	_, stillRegistered := m.managedWebsocket(connections[0])
	assert.False(t, stillRegistered, "retireConnections should remove a connection it fails to shut down from the manager")
}

// TestObserveConnectionRetiresOtherLostOptionalSetups loses the connections of two optional setups: the fault channel
// holds the first's fault and drops the second's, so the second is found lost and retired with the first
func TestObserveConnectionRetiresOtherLostOptionalSetups(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleDropServer(t)
	var m *Manager
	optionalSetup := func(filter string) *ConnectionSetup {
		s := lifecycleTestSetup(&m, wsURL, filter, dial, func() (subscription.List, error) {
			return subscription.List{{Channel: filter}}, nil
		})
		s.Optional = true
		return s
	}
	m = newLifecycleTestManager(t, lifecycleTestSetup(&m, wsURL, "public", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "public"}}, nil
	}), optionalSetup("first"), optionalSetup("second"))
	public, first, second := m.connectionManager[0], m.connectionManager[1], m.connectionManager[2]
	require.NoError(t, m.Connect(t.Context()), "Connect must not error")
	publicConnections := m.snapshotManagedConnections(public)
	firstConnections, secondConnections := m.snapshotManagedConnections(first), m.snapshotManagedConnections(second)
	require.Len(t, firstConnections, 1, "Connect must connect the first optional setup")
	require.Len(t, secondConnections, 1, "Connect must connect the second optional setup")
	require.Equal(t, 1, firstConnections[0].Subscriptions().Len(), "Connect must store the first optional connection's subscription")
	require.Equal(t, 1, secondConnections[0].Subscriptions().Len(), "Connect must store the second optional connection's subscription")
	drop := func(conn Connection) {
		require.NoError(t, conn.SendRawMessage(t.Context(), request.Unset, gws.TextMessage, []byte("drop")), "SendRawMessage must not error")
	}

	drop(firstConnections[0])
	require.Eventually(t, func() bool { return len(m.ReadMessageErrors) == 1 }, 5*time.Second, time.Millisecond, "the first lost connection must report its fault")
	drop(secondConnections[0])
	lost, ok := secondConnections[0].(*connection)
	require.True(t, ok, "the second optional connection must be a *connection")
	require.Eventually(t, func() bool { return !lost.IsConnected() }, 5*time.Second, time.Millisecond, "the second optional connection must be lost")
	timer := time.NewTimer(time.Hour)
	t.Cleanup(func() { timer.Stop() })
	assert.False(t, m.observeConnection(t.Context(), timer), "observeConnection should keep monitoring")
	assert.True(t, m.IsConnected(), "observeConnection should leave the manager connected")
	assert.Equal(t, publicConnections, m.snapshotManagedConnections(public), "observeConnection should leave the required connection up")
	assert.Empty(t, m.snapshotManagedConnections(first), "observeConnection should retire the optional connection whose fault it read")
	assert.Empty(t, m.snapshotManagedConnections(second), "observeConnection should retire an optional connection whose fault was dropped")
	assert.Equal(t, 1, first.retryFailures, "observeConnection should retry the optional setup whose fault it read")
	assert.Equal(t, 1, second.retryFailures, "observeConnection should retry the optional setup whose fault was dropped")
	// A caller can still hold a retired connection, which lists no subscriptions once retired
	assert.Zero(t, firstConnections[0].Subscriptions().Len(), "observeConnection should clear the store of the optional connection whose fault it read")
	assert.Zero(t, secondConnections[0].Subscriptions().Len(), "observeConnection should clear the store of an optional connection whose fault was dropped")
}

// TestRetryOptionalSetupsKeepsPartlySubscribedConnection retries an optional setup whose connection comes up while its
// subscriptions are refused: the retry counts as a recovery, keeping the connection and clearing the retry, and the
// refused subscriptions wait for the next flush or reconnection
func TestRetryOptionalSetupsKeepsPartlySubscribedConnection(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var accountUp atomic.Bool
	var m *Manager
	accountSub := &subscription.Subscription{Channel: "account"}
	account := lifecycleTestSetup(&m, wsURL, "account", func(ctx context.Context, conn Connection) error {
		if !accountUp.Load() {
			return errDastardlyReason
		}
		return dial(ctx, conn)
	}, func() (subscription.List, error) { return subscription.List{accountSub}, nil })
	account.Optional = true
	account.Subscriber = func(context.Context, Connection, subscription.List) error { return errDastardlyReason }
	m = newLifecycleTestManager(t, lifecycleTestSetup(&m, wsURL, "public", dial, func() (subscription.List, error) {
		return subscription.List{{Channel: "public"}}, nil
	}), account)
	optional := m.connectionManager[1]
	require.NoError(t, m.Connect(t.Context()), "Connect must not error for an optional setup's failure")
	require.Equal(t, 1, optional.retryFailures, "Connect must count the optional setup's failure")

	accountUp.Store(true)
	m.m.Lock()
	optional.retryAt = time.Now().Add(-time.Second)
	m.m.Unlock()
	m.retryOptionalSetups(t.Context())
	assert.Len(t, m.snapshotManagedConnections(optional), 1, "retryOptionalSetups should keep a connection whose subscriptions were refused")
	assert.Nil(t, optional.subscriptions.Get(accountSub), "retryOptionalSetups should not store a refused subscription")
	assert.Zero(t, optional.retryFailures, "retryOptionalSetups should clear the failures of a setup whose connection came up")
	assert.Zero(t, optional.retryAt, "retryOptionalSetups should clear the retry of a setup whose connection came up")
}

// TestRetryOptionalSetupsDiscardsClosedDial discards, with verbose logging on, a retry's connection whose socket closed
// while it was dialled and whose manager shut down meanwhile
func TestRetryOptionalSetupsDiscardsClosedDial(t *testing.T) {
	t.Parallel()
	h := newRetryDialTest(t, nil)
	h.m.verbose = true
	require.NoError(t, h.dialled.Connection.NetConn().Close(), "closing the dialled socket must not error")
	require.NoError(t, h.within(t, "Shutdown", h.m.Shutdown), "Shutdown must not error")
	h.release <- nil
	select {
	case <-h.retried:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the retry must return once its connector is released")
	}
	assert.Empty(t, h.m.snapshotManagedConnections(h.optional), "the retry should not keep a connection dialled across a shutdown")
	_, registered := h.m.managedWebsocket(h.dialled)
	assert.False(t, registered, "the retry should not register a connection dialled across a shutdown")
	select {
	case <-h.dialled.closed:
	default:
		assert.Fail(t, "the retry should shut down a connection dialled across a shutdown")
	}
	h.m.m.Lock()
	failures := h.optional.retryFailures
	h.m.m.Unlock()
	assert.Equal(t, 1, failures, "the retry should count no failure for a connection it discards")
}

// TestFlushChannelsRetiresConnectionFailingToAuthenticate flushes a connected setup, required or optional, whose new
// subscription needs another connection, which fails to authenticate. The flush retires that connection, so the next
// flush dials and authenticates another rather than subscribing on a connection that never signed in
func TestFlushChannelsRetiresConnectionFailingToAuthenticate(t *testing.T) {
	t.Parallel()
	for _, optional := range []bool{false, true} {
		t.Run(fmt.Sprintf("optional %t", optional), func(t *testing.T) {
			t.Parallel()
			wsURL, dial := lifecycleTestServer(t)
			errRefused, errNotSignedIn := errors.New("authentication refused"), errors.New("connection never signed in")
			first, second := &subscription.Subscription{Channel: "first"}, &subscription.Subscription{Channel: "second"}
			wanted := subscription.List{first}
			var m *Manager
			setup := lifecycleTestSetup(&m, wsURL, "account", dial, func() (subscription.List, error) { return slices.Clone(wanted), nil })
			setup.Optional = optional
			var authentications int
			signedIn := make(map[Connection]bool)
			setup.Authenticate = func(_ context.Context, conn Connection) error {
				authentications++
				if authentications == 2 {
					return errRefused
				}
				signedIn[conn] = true
				return nil
			}
			subscriber := setup.Subscriber
			setup.Subscriber = func(ctx context.Context, conn Connection, subs subscription.List) error {
				if !signedIn[conn] {
					return errNotSignedIn
				}
				return subscriber(ctx, conn, subs)
			}
			m = newLifecycleTestManager(t, setup)
			m.MaxSubscriptionsPerConnection = 1
			m.SetCanUseAuthenticatedEndpoints(true)
			require.NoError(t, m.Connect(t.Context()), "Connect must not error")
			ws := m.connectionManager[0]
			connected := m.snapshotManagedConnections(ws)
			require.Len(t, connected, 1, "Connect must connect the setup")

			wanted = subscription.List{first, second}
			assert.ErrorIs(t, m.FlushChannels(t.Context()), errRefused, "FlushChannels should report the failure to authenticate")
			assert.Equal(t, connected, m.snapshotManagedConnections(ws), "FlushChannels should retire the connection failing to authenticate")
			assert.NoError(t, m.FlushChannels(t.Context()), "FlushChannels should not error once another connection authenticates")
			assert.Equal(t, 3, authentications, "FlushChannels should authenticate another connection")
			assert.NotNil(t, ws.subscriptions.Get(second), "FlushChannels should subscribe on the connection that authenticated")
			assert.Zero(t, ws.retryFailures, "FlushChannels should count no failure for a setup that stayed connected")
		})
	}
}

// TestFlushChannelsConnectedOptionalSetupFailedDial flushes a connected optional setup whose new subscription needs
// another connection, which fails to dial: the setup keeps the connection it has and counts no failure, as only an
// optional setup left without connections is retried
func TestFlushChannelsConnectedOptionalSetupFailedDial(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var dials atomic.Int32
	first, second := &subscription.Subscription{Channel: "first"}, &subscription.Subscription{Channel: "second"}
	wanted := subscription.List{first}
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", func(ctx context.Context, conn Connection) error {
		if dials.Add(1) > 1 {
			return errDastardlyReason
		}
		return dial(ctx, conn)
	}, func() (subscription.List, error) { return slices.Clone(wanted), nil })
	account.Optional = true
	m = newLifecycleTestManager(t, account)
	m.MaxSubscriptionsPerConnection = 1
	optional := m.connectionManager[0]
	require.NoError(t, m.Connect(t.Context()), "Connect must not error")
	connected := m.snapshotManagedConnections(optional)
	require.Len(t, connected, 1, "Connect must connect the optional setup")

	wanted = subscription.List{first, second}
	err := m.FlushChannels(t.Context())
	assert.ErrorIs(t, err, errDastardlyReason, "FlushChannels should report the failed dial of another connection")
	assert.ErrorIs(t, err, common.ErrFatal, "FlushChannels should report the failed dial as fatal")
	assert.Equal(t, int32(2), dials.Load(), "FlushChannels should dial another connection for the new subscription")
	assert.Equal(t, connected, m.snapshotManagedConnections(optional), "FlushChannels should keep the connection a connected optional setup has")
	assert.NotNil(t, optional.subscriptions.Get(first), "FlushChannels should keep the subscription the kept connection holds")
	assert.Zero(t, optional.retryFailures, "FlushChannels should count no failure for an optional setup that stayed connected")
	assert.True(t, m.IsConnected(), "FlushChannels should leave the manager connected")
}

// TestFlushChannelsCancelledKeepsGatheredErrors ends a flush's context while the first setup's subscriptions are
// generated, partly: the flush stops before the second setup and returns the partial error with the context's
func TestFlushChannelsCancelledKeepsGatheredErrors(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var cancelling atomic.Bool
	var secondGenerations atomic.Int32
	var m *Manager
	first := lifecycleTestSetup(&m, wsURL, "first", dial, func() (subscription.List, error) {
		if cancelling.Load() {
			cancel()
			return subscription.List{{Channel: "first"}}, fmt.Errorf("%w: one left out", ErrSubscriptionPartial)
		}
		return subscription.List{{Channel: "first"}}, nil
	})
	m = newLifecycleTestManager(t, first, lifecycleTestSetup(&m, wsURL, "second", dial, func() (subscription.List, error) {
		secondGenerations.Add(1)
		return subscription.List{{Channel: "second"}}, nil
	}))
	require.NoError(t, m.Connect(t.Context()), "Connect must not error")
	require.Equal(t, int32(1), secondGenerations.Load(), "Connect must generate the second setup's subscriptions")

	cancelling.Store(true)
	err := m.FlushChannels(ctx)
	assert.ErrorIs(t, err, ErrSubscriptionPartial, "FlushChannels should keep the error gathered before its context ended")
	assert.ErrorIs(t, err, context.Canceled, "FlushChannels should return its context's error")
	require.Error(t, err, "FlushChannels must return an error")
	assert.Equal(t, 1, strings.Count(err.Error(), "one left out"), "FlushChannels should report the gathered error once")
	assert.Equal(t, 1, strings.Count(err.Error(), context.Canceled.Error()), "FlushChannels should report its context's error once")
	assert.Equal(t, int32(1), secondGenerations.Load(), "FlushChannels should not generate a setup's subscriptions once its context has ended")
}

// TestFlushChannelsConnectsDueOptionalSetup flushes an optional setup whose retry is due, so the flush dials it: once
// its connection comes up the flush keeps it and clears the retry, as a retry would, whether its subscription is
// accepted or refused, and reports a refusal
func TestFlushChannelsConnectsDueOptionalSetup(t *testing.T) {
	t.Parallel()
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprintf("subscription refused %t", refused), func(t *testing.T) {
			t.Parallel()
			wsURL, dial := lifecycleTestServer(t)
			errRefused := errors.New("subscription refused")
			var accountUp atomic.Bool
			var m *Manager
			accountSub := &subscription.Subscription{Channel: "account"}
			account := lifecycleTestSetup(&m, wsURL, "account", func(ctx context.Context, conn Connection) error {
				if !accountUp.Load() {
					return errDastardlyReason
				}
				return dial(ctx, conn)
			}, func() (subscription.List, error) { return subscription.List{accountSub}, nil })
			account.Optional = true
			subscriber := account.Subscriber
			account.Subscriber = func(ctx context.Context, conn Connection, subs subscription.List) error {
				if refused {
					return errRefused
				}
				return subscriber(ctx, conn, subs)
			}
			m = newLifecycleTestManager(t, account)
			optional := m.connectionManager[0]
			require.NoError(t, m.Connect(t.Context()), "Connect must not error for an optional setup's failure")
			require.Equal(t, 1, optional.retryFailures, "Connect must count the optional setup's failure")

			accountUp.Store(true)
			m.m.Lock()
			optional.retryAt = time.Now().Add(-time.Second)
			m.m.Unlock()
			err := m.FlushChannels(t.Context())
			if refused {
				assert.ErrorIs(t, err, errRefused, "FlushChannels should report a subscription refused once the setup connected")
			} else {
				assert.NoError(t, err, "FlushChannels should not error once the setup connected")
			}
			assert.Len(t, m.snapshotManagedConnections(optional), 1, "FlushChannels should keep the connection it dialled for the setup")
			assert.Equal(t, !refused, optional.subscriptions.Get(accountSub) != nil, "FlushChannels should store only an accepted subscription")
			assert.Zero(t, optional.retryFailures, "FlushChannels should clear the failures of a setup it connected")
			assert.Zero(t, optional.retryAt, "FlushChannels should clear the retry of a setup it connected")
		})
	}
}

// TestFlushChannelsDueOptionalSetupPartlyConnected flushes an optional setup whose retry is due and whose subscriptions
// need two connections, the second failing to dial: the setup is retired whole and retried later, as Connect leaves it,
// rather than kept on its first connection
func TestFlushChannelsDueOptionalSetupPartlyConnected(t *testing.T) {
	t.Parallel()
	wsURL, dial := lifecycleTestServer(t)
	var dials atomic.Int32
	var m *Manager
	account := lifecycleTestSetup(&m, wsURL, "account", func(ctx context.Context, conn Connection) error {
		if dials.Add(1) != 2 {
			return errDastardlyReason
		}
		return dial(ctx, conn)
	}, func() (subscription.List, error) {
		return subscription.List{{Channel: "first"}, {Channel: "second"}}, nil
	})
	account.Optional = true
	m = newLifecycleTestManager(t, account)
	m.MaxSubscriptionsPerConnection = 1
	optional := m.connectionManager[0]
	require.NoError(t, m.Connect(t.Context()), "Connect must not error for an optional setup's failure")
	require.Equal(t, 1, optional.retryFailures, "Connect must count the optional setup's failure")

	m.m.Lock()
	optional.retryAt = time.Now().Add(-time.Second)
	m.m.Unlock()
	assert.NoError(t, m.FlushChannels(t.Context()), "FlushChannels should not return an optional setup's failure to connect")
	assert.Equal(t, int32(3), dials.Load(), "FlushChannels should dial a connection for each subscription")
	assert.Empty(t, m.snapshotManagedConnections(optional), "FlushChannels should retire the first connection once the second fails to dial")
	assert.Zero(t, optional.subscriptions.Len(), "FlushChannels should clear the subscriptions of a setup it retired")
	assert.Equal(t, 2, optional.retryFailures, "FlushChannels should count the setup's failure to connect")
	assert.True(t, optional.retryAt.After(time.Now()), "FlushChannels should schedule the setup's next retry")
}
