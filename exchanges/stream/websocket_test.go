package stream

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	websocketTestURL = "wss://www.bitmex.com/realtime"
	useProxyTests    = false                     // Disabled by default. Freely available proxy servers that work all the time are difficult to find
	proxyURL         = "http://212.186.171.4:80" // Replace with a usable proxy server
)

var (
	errDastardlyReason = errors.New("some dastardly reason")
)

var dialer websocket.Dialer

type testStruct struct {
	Error error
	WC    WebsocketConnection
}

type testRequest struct {
	Event        string          `json:"event"`
	RequestID    int64           `json:"reqid,omitempty"`
	Pairs        []string        `json:"pair"`
	Subscription testRequestData `json:"subscription,omitempty"`
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

var defaultSetup = &WebsocketSetup{
	ExchangeConfig: &config.Exchange{
		Features: &config.FeaturesConfig{
			Enabled: config.FeaturesEnabledConfig{Websocket: true},
		},
		API: config.APIConfig{
			AuthenticatedWebsocketSupport: true,
		},
		WebsocketTrafficTimeout: time.Second * 5,
		Name:                    "GTX",
	},
	DefaultURL:   "testDefaultURL",
	RunningURL:   "wss://testRunningURL",
	Connector:    func() error { return nil },
	Subscriber:   func([]subscription.Subscription) error { return nil },
	Unsubscriber: func([]subscription.Subscription) error { return nil },
	GenerateSubscriptions: func() ([]subscription.Subscription, error) {
		return []subscription.Subscription{
			{Channel: "TestSub"},
			{Channel: "TestSub2", Key: "purple"},
			{Channel: "TestSub3", Key: testSubKey{"mauve"}},
			{Channel: "TestSub4", Key: 42},
		}, nil
	},
	Features: &protocol.Features{Subscribe: true, Unsubscribe: true},
}

type dodgyConnection struct {
	WebsocketConnection
}

// override websocket connection method to produce a wicked terrible error
func (d *dodgyConnection) Shutdown() error {
	return fmt.Errorf("%w: %w", errCannotShutdown, errDastardlyReason)
}

// override websocket connection method to produce a wicked terrible error
func (d *dodgyConnection) Connect() error {
	return fmt.Errorf("cannot connect: %w", errDastardlyReason)
}

func TestMain(m *testing.M) {
	// Change trafficCheckInterval for TestTrafficMonitorTimeout before parallel tests to avoid racing
	trafficCheckInterval = 50 * time.Millisecond
	os.Exit(m.Run())
}

func TestSetup(t *testing.T) {
	t.Parallel()
	var w *Websocket
	err := w.Setup(nil)
	assert.ErrorIs(t, err, errWebsocketIsNil, "Setup should error correctly")

	w = &Websocket{DataHandler: make(chan interface{})}
	err = w.Setup(nil)
	assert.ErrorIs(t, err, errWebsocketSetupIsNil, "Setup should error correctly")

	websocketSetup := &WebsocketSetup{}

	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errExchangeConfigIsNil, "Setup should error correctly")

	websocketSetup.ExchangeConfig = &config.Exchange{}
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errExchangeConfigNameEmpty, "Setup should error correctly")

	websocketSetup.ExchangeConfig.Name = "testname"
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errWebsocketFeaturesIsUnset, "Setup should error correctly")

	websocketSetup.Features = &protocol.Features{}
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errConfigFeaturesIsNil, "Setup should error correctly")

	websocketSetup.ExchangeConfig.Features = &config.FeaturesConfig{}
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errWebsocketConnectorUnset, "Setup should error correctly")

	websocketSetup.Connector = func() error { return nil }
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errWebsocketSubscriberUnset, "Setup should error correctly")

	websocketSetup.Subscriber = func([]subscription.Subscription) error { return nil }
	websocketSetup.Features.Unsubscribe = true
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errWebsocketUnsubscriberUnset, "Setup should error correctly")

	websocketSetup.Unsubscriber = func([]subscription.Subscription) error { return nil }
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errWebsocketSubscriptionsGeneratorUnset, "Setup should error correctly")

	websocketSetup.GenerateSubscriptions = func() ([]subscription.Subscription, error) { return nil, nil }
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errDefaultURLIsEmpty, "Setup should error correctly")

	websocketSetup.DefaultURL = "test"
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errRunningURLIsEmpty, "Setup should error correctly")

	websocketSetup.RunningURL = "http://www.google.com"
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errInvalidWebsocketURL, "Setup should error correctly")

	websocketSetup.RunningURL = "wss://www.google.com"
	websocketSetup.RunningURLAuth = "http://www.google.com"
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errInvalidWebsocketURL, "Setup should error correctly")

	websocketSetup.RunningURLAuth = "wss://www.google.com"
	err = w.Setup(websocketSetup)
	assert.ErrorIs(t, err, errInvalidTrafficTimeout, "Setup should error correctly")

	websocketSetup.ExchangeConfig.WebsocketTrafficTimeout = time.Minute
	err = w.Setup(websocketSetup)
	assert.NoError(t, err, "Setup should not error")
}

// TestTrafficMonitorTrafficAlerts ensures multiple traffic alerts work and only process one trafficAlert per interval
// ensures shutdown works after traffic alerts
func TestTrafficMonitorTrafficAlerts(t *testing.T) {
	t.Parallel()
	ws := NewWebsocket()
	err := ws.Setup(defaultSetup)
	require.NoError(t, err, "Setup must not error")

	signal := struct{}{}
	patience := 10 * time.Millisecond
	ws.trafficTimeout = 200 * time.Millisecond
	ws.ShutdownC = make(chan struct{})
	ws.state.Store(connected)

	thenish := time.Now()
	ws.trafficMonitor()

	assert.True(t, ws.IsTrafficMonitorRunning(), "traffic monitor should be running")
	require.Equal(t, connected, ws.state.Load(), "websocket must be connected")

	for i := 0; i < 6; i++ { // Timeout will happen at 200ms so we want 6 * 50ms checks to pass
		select {
		case ws.TrafficAlert <- signal:
			if i == 0 {
				require.WithinDurationf(t, time.Now(), thenish, trafficCheckInterval, "First Non-blocking test must happen before the traffic is checked")
			}
		default:
			require.Failf(t, "", "TrafficAlert should not block; Check #%d", i)
		}

		select {
		case ws.TrafficAlert <- signal:
			require.Failf(t, "", "TrafficAlert should block after first slot used; Check #%d", i)
		default:
			if i == 0 {
				require.WithinDuration(t, time.Now(), thenish, trafficCheckInterval, "First Blocking test must happen before the traffic is checked")
			}
		}

		require.Eventuallyf(t, func() bool { return len(ws.TrafficAlert) == 0 }, 5*time.Second, patience, "trafficAlert should be drained; Check #%d", i)
		assert.Truef(t, ws.IsConnected(), "state should still be connected; Check #%d", i)
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Equal(c, disconnected, ws.state.Load(), "websocket must be disconnected")
		assert.False(c, ws.IsTrafficMonitorRunning(), "trafficMonitor should be shut down")
	}, 2*ws.trafficTimeout, patience, "trafficTimeout should trigger a shutdown once we stop feeding trafficAlerts")
}

// TestTrafficMonitorConnecting ensures connecting status doesn't trigger shutdown
func TestTrafficMonitorConnecting(t *testing.T) {
	t.Parallel()
	ws := NewWebsocket()
	err := ws.Setup(defaultSetup)
	require.NoError(t, err, "Setup must not error")

	ws.ShutdownC = make(chan struct{})
	ws.state.Store(connecting)
	ws.trafficTimeout = 50 * time.Millisecond
	ws.trafficMonitor()
	require.True(t, ws.IsTrafficMonitorRunning(), "traffic monitor should be running")
	require.Equal(t, connecting, ws.state.Load(), "websocket must be connecting")
	<-time.After(4 * ws.trafficTimeout)
	require.Equal(t, connecting, ws.state.Load(), "websocket must still be connecting after several checks")
	ws.state.Store(connected)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Equal(c, disconnected, ws.state.Load(), "websocket must be disconnected")
		assert.False(c, ws.IsTrafficMonitorRunning(), "trafficMonitor should be shut down")
	}, 4*ws.trafficTimeout, 10*time.Millisecond, "trafficTimeout should trigger a shutdown after connecting status changes")
}

// TestTrafficMonitorShutdown ensures shutdown is processed and waitgroup is cleared
func TestTrafficMonitorShutdown(t *testing.T) {
	t.Parallel()
	ws := NewWebsocket()
	err := ws.Setup(defaultSetup)
	require.NoError(t, err, "Setup must not error")

	ws.ShutdownC = make(chan struct{})
	ws.state.Store(connected)
	ws.trafficTimeout = time.Minute
	ws.trafficMonitor()
	assert.True(t, ws.IsTrafficMonitorRunning(), "traffic monitor should be running")

	wgReady := make(chan bool)
	go func() {
		ws.Wg.Wait()
		close(wgReady)
	}()
	select {
	case <-wgReady:
		require.Failf(t, "", "WaitGroup should be blocking still")
	case <-time.After(trafficCheckInterval):
	}

	close(ws.ShutdownC)

	<-time.After(2 * trafficCheckInterval)
	assert.False(t, ws.IsTrafficMonitorRunning(), "traffic monitor should be shutdown")
	select {
	case <-wgReady:
	default:
		require.Failf(t, "", "WaitGroup should be freed now")
	}
}

func TestIsDisconnectionError(t *testing.T) {
	t.Parallel()
	assert.False(t, IsDisconnectionError(errors.New("errorText")), "IsDisconnectionError should return false")
	assert.True(t, IsDisconnectionError(&websocket.CloseError{Code: 1006, Text: "errorText"}), "IsDisconnectionError should return true")
	assert.False(t, IsDisconnectionError(&net.OpError{Err: errClosedConnection}), "IsDisconnectionError should return false")
	assert.True(t, IsDisconnectionError(&net.OpError{Err: errors.New("errText")}), "IsDisconnectionError should return true")
}

func TestConnectionMessageErrors(t *testing.T) {
	t.Parallel()
	var wsWrong = &Websocket{}
	err := wsWrong.Connect()
	assert.ErrorIs(t, err, errNoConnectFunc, "Connect should error correctly")

	wsWrong.connector = func() error { return nil }
	err = wsWrong.Connect()
	assert.ErrorIs(t, err, ErrWebsocketNotEnabled, "Connect should error correctly")

	wsWrong.setEnabled(true)
	wsWrong.setState(connecting)
	wsWrong.Wg = &sync.WaitGroup{}
	err = wsWrong.Connect()
	assert.ErrorIs(t, err, errAlreadyReconnecting, "Connect should error correctly")

	wsWrong.setState(disconnected)
	wsWrong.connector = func() error { return errDastardlyReason }
	err = wsWrong.Connect()
	assert.ErrorIs(t, err, errDastardlyReason, "Connect should error correctly")

	ws := NewWebsocket()
	err = ws.Setup(defaultSetup)
	require.NoError(t, err, "Setup must not error")
	ws.trafficTimeout = time.Minute
	ws.connector = func() error { return nil }

	err = ws.Connect()
	require.NoError(t, err, "Connect must not error")

	c := func(tb *assert.CollectT) {
		select {
		case v, ok := <-ws.ToRoutine:
			require.True(tb, ok, "ToRoutine should not be closed on us")
			switch err := v.(type) {
			case *websocket.CloseError:
				assert.Equal(tb, "SpecialText", err.Text, "Should get correct Close Error")
			case error:
				assert.ErrorIs(tb, err, errDastardlyReason, "Should get the correct error")
			default:
				assert.Failf(tb, "Wrong data type sent to ToRoutine", "Got type: %T", err)
			}
		default:
			assert.Fail(tb, "Nothing available on ToRoutine")
		}
	}

	ws.TrafficAlert <- struct{}{}
	ws.ReadMessageErrors <- errDastardlyReason
	assert.EventuallyWithT(t, c, 2*time.Second, 10*time.Millisecond, "Should get an error down the routine")

	ws.ReadMessageErrors <- &websocket.CloseError{Code: 1006, Text: "SpecialText"}
	assert.EventuallyWithT(t, c, 2*time.Second, 10*time.Millisecond, "Should get an error down the routine")
}

func TestWebsocket(t *testing.T) {
	t.Parallel()

	ws := NewWebsocket()

	err := ws.SetProxyAddress("garbagio")
	assert.ErrorContains(t, err, "invalid URI for request", "SetProxyAddress should error correctly")

	ws.Conn = &dodgyConnection{}
	ws.AuthConn = &WebsocketConnection{}
	ws.setEnabled(true)

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

	err = ws.SetProxyAddress("https://192.168.0.1:1337")
	assert.NoError(t, err, "SetProxyAddress should not error when not yet connected")

	ws.setState(connected)

	err = ws.SetProxyAddress("https://192.168.0.1:1336")
	assert.ErrorIs(t, err, errDastardlyReason, "SetProxyAddress should call Connect and error from there")

	err = ws.SetProxyAddress("https://192.168.0.1:1336")
	assert.ErrorIs(t, err, errSameProxyAddress, "SetProxyAddress should error correctly")

	// removing proxy
	err = ws.SetProxyAddress("")
	assert.ErrorIs(t, err, errDastardlyReason, "SetProxyAddress should call Shutdown and error from there")
	assert.ErrorIs(t, err, errCannotShutdown, "SetProxyAddress should call Shutdown and error from there")

	ws.Conn = &WebsocketConnection{}
	ws.setEnabled(true)

	// reinstate proxy
	err = ws.SetProxyAddress("http://localhost:1337")
	assert.NoError(t, err, "SetProxyAddress should not error")
	assert.Equal(t, "http://localhost:1337", ws.GetProxyAddress(), "GetProxyAddress should return correctly")
	assert.Equal(t, "wss://testRunningURL", ws.GetWebsocketURL(), "GetWebsocketURL should return correctly")
	assert.Equal(t, time.Second*5, ws.trafficTimeout, "trafficTimeout should default correctly")

	ws.setState(connected)
	ws.AuthConn = &dodgyConnection{}
	err = ws.Shutdown()
	assert.ErrorIs(t, err, errDastardlyReason, "Shutdown should error correctly with a dodgy authConn")
	assert.ErrorIs(t, err, errCannotShutdown, "Shutdown should error correctly with a dodgy authConn")

	ws.AuthConn = &WebsocketConnection{}
	ws.setState(disconnected)

	err = ws.Connect()
	assert.NoError(t, err, "Connect should not error")

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
	err = ws.Connect()
	assert.NoError(t, err, "ReConnect called manually should not error")

	err = ws.Connect()
	assert.ErrorIs(t, err, errAlreadyConnected, "ReConnect should error when already connected")

	err = ws.Shutdown()
	assert.NoError(t, err, "Shutdown should not error")
	ws.Wg.Wait()
}

// TestSubscribe logic test
func TestSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	ws := NewWebsocket()
	assert.NoError(t, ws.Setup(defaultSetup), "WS Setup should not error")

	fnSub := func(subs []subscription.Subscription) error {
		ws.AddSuccessfulSubscriptions(subs...)
		return nil
	}
	fnUnsub := func(unsubs []subscription.Subscription) error {
		ws.RemoveSubscriptions(unsubs...)
		return nil
	}
	ws.Subscriber = fnSub
	ws.Unsubscriber = fnUnsub

	subs, err := ws.GenerateSubs()
	assert.NoError(t, err, "Generating test subscriptions should not error")
	assert.ErrorIs(t, ws.UnsubscribeChannels(nil), errNoSubscriptionsSupplied, "Unsubscribing from nil should error")
	assert.ErrorIs(t, ws.UnsubscribeChannels(subs), ErrSubscriptionNotFound, "Unsubscribing should error when not subscribed")
	assert.Nil(t, ws.GetSubscription(42), "GetSubscription on empty internal map should return")
	assert.NoError(t, ws.SubscribeToChannels(subs), "Basic Subscribing should not error")
	assert.Len(t, ws.GetSubscriptions(), 4, "Should have 4 subscriptions")
	byDefKey := ws.GetSubscription(subscription.DefaultKey{Channel: "TestSub"})
	if assert.NotNil(t, byDefKey, "GetSubscription by default key should find a channel") {
		assert.Equal(t, "TestSub", byDefKey.Channel, "GetSubscription by default key should return a pointer a copy of the right channel")
		assert.NotSame(t, byDefKey, ws.subscriptions["TestSub"], "GetSubscription returns a fresh pointer")
	}
	if assert.NotNil(t, ws.GetSubscription("purple"), "GetSubscription by string key should find a channel") {
		assert.Equal(t, "TestSub2", ws.GetSubscription("purple").Channel, "GetSubscription by string key should return a pointer a copy of the right channel")
	}
	if assert.NotNil(t, ws.GetSubscription(testSubKey{"mauve"}), "GetSubscription by type key should find a channel") {
		assert.Equal(t, "TestSub3", ws.GetSubscription(testSubKey{"mauve"}).Channel, "GetSubscription by type key should return a pointer a copy of the right channel")
	}
	if assert.NotNil(t, ws.GetSubscription(42), "GetSubscription by int key should find a channel") {
		assert.Equal(t, "TestSub4", ws.GetSubscription(42).Channel, "GetSubscription by int key should return a pointer a copy of the right channel")
	}
	assert.Nil(t, ws.GetSubscription(nil), "GetSubscription by nil should return nil")
	assert.Nil(t, ws.GetSubscription(45), "GetSubscription by invalid key should return nil")
	assert.ErrorIs(t, ws.SubscribeToChannels(subs), errChannelAlreadySubscribed, "Subscribe should error when already subscribed")
	assert.ErrorIs(t, ws.SubscribeToChannels(nil), errNoSubscriptionsSupplied, "Subscribe to nil should error")
	assert.NoError(t, ws.UnsubscribeChannels(subs), "Unsubscribing should not error")
}

// TestResubscribe tests Resubscribing to existing subscriptions
func TestResubscribe(t *testing.T) {
	t.Parallel()
	ws := NewWebsocket()

	wackedOutSetup := *defaultSetup
	wackedOutSetup.MaxWebsocketSubscriptionsPerConnection = -1
	err := ws.Setup(&wackedOutSetup)
	assert.ErrorIs(t, err, errInvalidMaxSubscriptions, "Invalid MaxWebsocketSubscriptionsPerConnection should error")

	err = ws.Setup(defaultSetup)
	assert.NoError(t, err, "WS Setup should not error")

	fnSub := func(subs []subscription.Subscription) error {
		ws.AddSuccessfulSubscriptions(subs...)
		return nil
	}
	fnUnsub := func(unsubs []subscription.Subscription) error {
		ws.RemoveSubscriptions(unsubs...)
		return nil
	}
	ws.Subscriber = fnSub
	ws.Unsubscriber = fnUnsub

	channel := []subscription.Subscription{{Channel: "resubTest"}}

	assert.ErrorIs(t, ws.ResubscribeToChannel(&channel[0]), ErrSubscriptionNotFound, "Resubscribe should error when channel isn't subscribed yet")
	assert.NoError(t, ws.SubscribeToChannels(channel), "Subscribe should not error")
	assert.NoError(t, ws.ResubscribeToChannel(&channel[0]), "Resubscribe should not error now the channel is subscribed")
}

// TestSubscriptionState tests Subscription state changes
func TestSubscriptionState(t *testing.T) {
	t.Parallel()
	ws := NewWebsocket()

	c := &subscription.Subscription{Key: 42, Channel: "Gophers", State: subscription.SubscribingState}
	assert.ErrorIs(t, ws.SetSubscriptionState(c, subscription.UnsubscribingState), ErrSubscriptionNotFound, "Setting an imaginary sub should error")

	assert.NoError(t, ws.AddSubscription(c), "Adding first subscription should not error")
	found := ws.GetSubscription(42)
	assert.NotNil(t, found, "Should find the subscription")
	assert.Equal(t, subscription.SubscribingState, found.State, "Subscription should be Subscribing")
	assert.ErrorIs(t, ws.AddSubscription(c), ErrSubscribedAlready, "Adding an already existing sub should error")
	assert.ErrorIs(t, ws.SetSubscriptionState(c, subscription.SubscribingState), ErrChannelInStateAlready, "Setting Same state should error")
	assert.ErrorIs(t, ws.SetSubscriptionState(c, subscription.UnsubscribingState+1), errInvalidChannelState, "Setting an invalid state should error")

	ws.AddSuccessfulSubscriptions(*c)
	found = ws.GetSubscription(42)
	assert.NotNil(t, found, "Should find the subscription")
	assert.Equal(t, subscription.SubscribedState, found.State, "Subscription should be subscribed state")

	assert.NoError(t, ws.SetSubscriptionState(c, subscription.UnsubscribingState), "Setting Unsub state should not error")
	found = ws.GetSubscription(42)
	assert.Equal(t, subscription.UnsubscribingState, found.State, "Subscription should be unsubscribing state")
}

// TestRemoveSubscriptions tests removing a subscription
func TestRemoveSubscriptions(t *testing.T) {
	t.Parallel()
	ws := NewWebsocket()

	c := &subscription.Subscription{Key: 42, Channel: "Unite!"}
	assert.NoError(t, ws.AddSubscription(c), "Adding first subscription should not error")
	assert.NotNil(t, ws.GetSubscription(42), "Added subscription should be findable")

	ws.RemoveSubscriptions(*c)
	assert.Nil(t, ws.GetSubscription(42), "Remove should have removed the sub")
}

// TestConnectionMonitorNoConnection logic test
func TestConnectionMonitorNoConnection(t *testing.T) {
	t.Parallel()
	ws := NewWebsocket()
	ws.connectionMonitorDelay = 500
	ws.DataHandler = make(chan interface{}, 1)
	ws.ShutdownC = make(chan struct{}, 1)
	ws.exchangeName = "hello"
	ws.Wg = &sync.WaitGroup{}
	ws.setEnabled(true)
	err := ws.connectionMonitor()
	require.NoError(t, err, "connectionMonitor must not error")
	assert.True(t, ws.IsConnectionMonitorRunning(), "IsConnectionMonitorRunning should return true")
	err = ws.connectionMonitor()
	assert.ErrorIs(t, err, errAlreadyRunning, "connectionMonitor should error correctly")
}

// TestGetSubscription logic test
func TestGetSubscription(t *testing.T) {
	t.Parallel()
	assert.Nil(t, (*Websocket).GetSubscription(nil, "imaginary"), "GetSubscription on a nil Websocket should return nil")
	assert.Nil(t, (&Websocket{}).GetSubscription("empty"), "GetSubscription on a Websocket with no sub map should return nil")
	w := Websocket{
		subscriptions: subscriptionMap{
			42: {
				Channel: "hello3",
			},
		},
	}
	assert.Nil(t, w.GetSubscription(43), "GetSubscription with an invalid key should return nil")
	c := w.GetSubscription(42)
	if assert.NotNil(t, c, "GetSubscription with an valid key should return a channel") {
		assert.Equal(t, "hello3", c.Channel, "GetSubscription should return the correct channel details")
	}
}

// TestGetSubscriptions logic test
func TestGetSubscriptions(t *testing.T) {
	t.Parallel()
	w := Websocket{
		subscriptions: subscriptionMap{
			42: {
				Channel: "hello3",
			},
		},
	}
	assert.Equal(t, "hello3", w.GetSubscriptions()[0].Channel, "GetSubscriptions should return the correct channel details")
}

// TestSetCanUseAuthenticatedEndpoints logic test
func TestSetCanUseAuthenticatedEndpoints(t *testing.T) {
	t.Parallel()
	ws := NewWebsocket()
	assert.False(t, ws.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints should return false")
	ws.SetCanUseAuthenticatedEndpoints(true)
	assert.True(t, ws.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints should return true")
}

// TestDial logic test
func TestDial(t *testing.T) {
	t.Parallel()
	var testCases = []testStruct{
		{Error: nil,
			WC: WebsocketConnection{
				ExchangeName:     "test1",
				Verbose:          true,
				URL:              websocketTestURL,
				RateLimit:        10,
				ResponseMaxLimit: 7000000000,
			},
		},
		{Error: errors.New(" Error: malformed ws or wss URL"),
			WC: WebsocketConnection{
				ExchangeName:     "test2",
				Verbose:          true,
				URL:              "",
				ResponseMaxLimit: 7000000000,
			},
		},
		{Error: nil,
			WC: WebsocketConnection{
				ExchangeName:     "test3",
				Verbose:          true,
				URL:              websocketTestURL,
				ProxyURL:         proxyURL,
				ResponseMaxLimit: 7000000000,
			},
		},
	}
	for i := range testCases {
		testData := &testCases[i]
		t.Run(testData.WC.ExchangeName, func(t *testing.T) {
			t.Parallel()
			if testData.WC.ProxyURL != "" && !useProxyTests {
				t.Skip("Proxy testing not enabled, skipping")
			}
			err := testData.WC.Dial(&dialer, http.Header{})
			if err != nil {
				if testData.Error != nil && strings.Contains(err.Error(), testData.Error.Error()) {
					return
				}
				t.Fatal(err)
			}
		})
	}
}

// TestSendMessage logic test
func TestSendMessage(t *testing.T) {
	t.Parallel()
	var testCases = []testStruct{
		{Error: nil, WC: WebsocketConnection{
			ExchangeName:     "test1",
			Verbose:          true,
			URL:              websocketTestURL,
			RateLimit:        10,
			ResponseMaxLimit: 7000000000,
		},
		},
		{Error: errors.New(" Error: malformed ws or wss URL"),
			WC: WebsocketConnection{
				ExchangeName:     "test2",
				Verbose:          true,
				URL:              "",
				ResponseMaxLimit: 7000000000,
			},
		},
		{Error: nil,
			WC: WebsocketConnection{
				ExchangeName:     "test3",
				Verbose:          true,
				URL:              websocketTestURL,
				ProxyURL:         proxyURL,
				ResponseMaxLimit: 7000000000,
			},
		},
	}
	for i := range testCases {
		testData := &testCases[i]
		t.Run(testData.WC.ExchangeName, func(t *testing.T) {
			t.Parallel()
			if testData.WC.ProxyURL != "" && !useProxyTests {
				t.Skip("Proxy testing not enabled, skipping")
			}
			err := testData.WC.Dial(&dialer, http.Header{})
			if err != nil {
				if testData.Error != nil && strings.Contains(err.Error(), testData.Error.Error()) {
					return
				}
				t.Fatal(err)
			}
			err = testData.WC.SendJSONMessage(Ping)
			if err != nil {
				t.Error(err)
			}
			err = testData.WC.SendRawMessage(websocket.TextMessage, []byte(Ping))
			if err != nil {
				t.Error(err)
			}
		})
	}
}

// TestSendMessageWithResponse logic test
func TestSendMessageWithResponse(t *testing.T) {
	t.Parallel()
	wc := &WebsocketConnection{
		Verbose:          true,
		URL:              "wss://ws.kraken.com",
		ResponseMaxLimit: time.Second * 5,
		Match:            NewMatch(),
	}
	if wc.ProxyURL != "" && !useProxyTests {
		t.Skip("Proxy testing not enabled, skipping")
	}

	err := wc.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}

	go readMessages(t, wc)

	request := testRequest{
		Event: "subscribe",
		Pairs: []string{currency.NewPairWithDelimiter("XBT", "USD", "/").String()},
		Subscription: testRequestData{
			Name: "ticker",
		},
		RequestID: wc.GenerateMessageID(false),
	}

	_, err = wc.SendMessageReturnResponse(request.RequestID, request)
	if err != nil {
		t.Error(err)
	}
}

type reporter struct {
	name string
	msg  []byte
	t    time.Duration
}

func (r *reporter) Latency(name string, message []byte, t time.Duration) {
	r.name = name
	r.msg = message
	r.t = t
}

// readMessages helper func
func readMessages(t *testing.T, wc *WebsocketConnection) {
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
	wc := &WebsocketConnection{
		URL:              websocketTestURL,
		ResponseMaxLimit: time.Second * 5,
		Match:            NewMatch(),
		Wg:               &sync.WaitGroup{},
	}

	if wc.ProxyURL != "" && !useProxyTests {
		t.Skip("Proxy testing not enabled, skipping")
	}
	wc.ShutdownC = make(chan struct{})
	err := wc.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}

	wc.SetupPingHandler(PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PingMessage,
		Delay:             100,
	})

	err = wc.Connection.Close()
	if err != nil {
		t.Error(err)
	}

	err = wc.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	wc.SetupPingHandler(PingHandler{
		MessageType: websocket.TextMessage,
		Message:     []byte(Ping),
		Delay:       200,
	})
	time.Sleep(time.Millisecond * 201)
	close(wc.ShutdownC)
	wc.Wg.Wait()
}

// TestParseBinaryResponse logic test
func TestParseBinaryResponse(t *testing.T) {
	t.Parallel()
	wc := &WebsocketConnection{
		URL:              websocketTestURL,
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
	ws := &Websocket{}
	assert.False(t, ws.CanUseAuthenticatedWebsocketForWrapper(), "CanUseAuthenticatedWebsocketForWrapper should return false")

	ws.setState(connected)
	require.True(t, ws.IsConnected(), "IsConnected must return true")
	assert.False(t, ws.CanUseAuthenticatedWebsocketForWrapper(), "CanUseAuthenticatedWebsocketForWrapper should return false")

	ws.SetCanUseAuthenticatedEndpoints(true)
	assert.True(t, ws.CanUseAuthenticatedWebsocketForWrapper(), "CanUseAuthenticatedWebsocketForWrapper should return true")
}

func TestGenerateMessageID(t *testing.T) {
	t.Parallel()
	wc := WebsocketConnection{}
	const spins = 1000
	ids := make([]int64, spins)
	for i := 0; i < spins; i++ {
		id := wc.GenerateMessageID(true)
		assert.NotContains(t, ids, id, "GenerateMessageID must not generate the same ID twice")
		ids[i] = id
	}
}

// BenchmarkGenerateMessageID-8   	 2850018	       408 ns/op	      56 B/op	       4 allocs/op
func BenchmarkGenerateMessageID_High(b *testing.B) {
	wc := WebsocketConnection{}
	for i := 0; i < b.N; i++ {
		_ = wc.GenerateMessageID(true)
	}
}

// BenchmarkGenerateMessageID_Low-8   	 2591596	       447 ns/op	      56 B/op	       4 allocs/op
func BenchmarkGenerateMessageID_Low(b *testing.B) {
	wc := WebsocketConnection{}
	for i := 0; i < b.N; i++ {
		_ = wc.GenerateMessageID(false)
	}
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

func TestGetChannelDifference(t *testing.T) {
	t.Parallel()
	web := Websocket{}

	newChans := []subscription.Subscription{
		{
			Channel: "Test1",
		},
		{
			Channel: "Test2",
		},
		{
			Channel: "Test3",
		},
	}
	subs, unsubs := web.GetChannelDifference(newChans)
	assert.Len(t, subs, 3, "Should get the correct number of subs")
	assert.Empty(t, unsubs, "Should get the correct number of unsubs")

	web.AddSuccessfulSubscriptions(subs...)

	flushedSubs := []subscription.Subscription{
		{
			Channel: "Test2",
		},
	}

	subs, unsubs = web.GetChannelDifference(flushedSubs)
	assert.Empty(t, subs, "Should get the correct number of subs")
	assert.Len(t, unsubs, 2, "Should get the correct number of unsubs")

	flushedSubs = []subscription.Subscription{
		{
			Channel: "Test2",
		},
		{
			Channel: "Test4",
		},
	}

	subs, unsubs = web.GetChannelDifference(flushedSubs)
	if assert.Len(t, subs, 1, "Should get the correct number of subs") {
		assert.Equal(t, "Test4", subs[0].Channel, "Should subscribe to the right channel")
	}
	if assert.Len(t, unsubs, 2, "Should get the correct number of unsubs") {
		sort.Slice(unsubs, func(i, j int) bool { return unsubs[i].Channel <= unsubs[j].Channel })
		assert.Equal(t, "Test1", unsubs[0].Channel, "Should unsubscribe from the right channels")
		assert.Equal(t, "Test3", unsubs[1].Channel, "Should unsubscribe from the right channels")
	}
}

// GenSubs defines a theoretical exchange with pair management
type GenSubs struct {
	EnabledPairs currency.Pairs
	subscribos   []subscription.Subscription
	unsubscribos []subscription.Subscription
}

// generateSubs default subs created from the enabled pairs list
func (g *GenSubs) generateSubs() ([]subscription.Subscription, error) {
	superduperchannelsubs := make([]subscription.Subscription, len(g.EnabledPairs))
	for i := range g.EnabledPairs {
		superduperchannelsubs[i] = subscription.Subscription{
			Channel: "TEST:" + strconv.FormatInt(int64(i), 10),
			Pair:    g.EnabledPairs[i],
		}
	}
	return superduperchannelsubs, nil
}

func (g *GenSubs) SUBME(subs []subscription.Subscription) error {
	if len(subs) == 0 {
		return errors.New("WOW")
	}
	g.subscribos = subs
	return nil
}

func (g *GenSubs) UNSUBME(unsubs []subscription.Subscription) error {
	if len(unsubs) == 0 {
		return errors.New("WOW")
	}
	g.unsubscribos = unsubs
	return nil
}

// sneaky connect func
func connect() error { return nil }

func TestFlushChannels(t *testing.T) {
	t.Parallel()
	// Enabled pairs/setup system
	newgen := GenSubs{EnabledPairs: []currency.Pair{
		currency.NewPair(currency.BTC, currency.AUD),
		currency.NewPair(currency.BTC, currency.USDT),
	}}

	dodgyWs := Websocket{}
	err := dodgyWs.FlushChannels()
	assert.ErrorIs(t, err, ErrWebsocketNotEnabled, "FlushChannels should error correctly")

	dodgyWs.setEnabled(true)
	err = dodgyWs.FlushChannels()
	assert.ErrorIs(t, err, ErrNotConnected, "FlushChannels should error correctly")

	w := Websocket{
		connector:    connect,
		ShutdownC:    make(chan struct{}),
		Subscriber:   newgen.SUBME,
		Unsubscriber: newgen.UNSUBME,
		Wg:           new(sync.WaitGroup),
		features:     &protocol.Features{
			// No features
		},
		trafficTimeout: time.Second * 30, // Added for when we utilise connect()
		// in FlushChannels() so the traffic monitor doesn't time out and turn
		// this to an unconnected state
	}
	w.setEnabled(true)
	w.setState(connected)

	problemFunc := func() ([]subscription.Subscription, error) {
		return nil, errDastardlyReason
	}

	noSub := func() ([]subscription.Subscription, error) {
		return nil, nil
	}

	// Disable pair and flush system
	newgen.EnabledPairs = []currency.Pair{
		currency.NewPair(currency.BTC, currency.AUD)}
	w.GenerateSubs = func() ([]subscription.Subscription, error) {
		return []subscription.Subscription{{Channel: "test"}}, nil
	}
	err = w.FlushChannels()
	assert.NoError(t, err, "FlushChannels should not error")

	w.features.FullPayloadSubscribe = true
	w.GenerateSubs = problemFunc
	err = w.FlushChannels() // error on full subscribeToChannels
	assert.ErrorIs(t, err, errDastardlyReason, "FlushChannels should error correctly")

	w.GenerateSubs = noSub
	err = w.FlushChannels() // No subs to unsub
	assert.NoError(t, err, "FlushChannels should not error")

	w.GenerateSubs = newgen.generateSubs
	subs, err := w.GenerateSubs()
	require.NoError(t, err, "GenerateSubs must not error")

	w.AddSuccessfulSubscriptions(subs...)
	err = w.FlushChannels()
	assert.NoError(t, err, "FlushChannels should not error")
	w.features.FullPayloadSubscribe = false
	w.features.Subscribe = true

	w.GenerateSubs = problemFunc
	err = w.FlushChannels()
	assert.ErrorIs(t, err, errDastardlyReason, "FlushChannels should error correctly")

	w.GenerateSubs = newgen.generateSubs
	err = w.FlushChannels()
	assert.NoError(t, err, "FlushChannels should not error")
	w.subscriptionMutex.Lock()
	w.subscriptions = subscriptionMap{
		41: {
			Key:     41,
			Channel: "match channel",
			Pair:    currency.NewPair(currency.BTC, currency.AUD),
		},
		42: {
			Key:     42,
			Channel: "unsub channel",
			Pair:    currency.NewPair(currency.THETA, currency.USDT),
		},
	}
	w.subscriptionMutex.Unlock()

	err = w.FlushChannels()
	assert.NoError(t, err, "FlushChannels should not error")

	err = w.FlushChannels()
	assert.NoError(t, err, "FlushChannels should not error")

	w.setState(connected)
	w.features.Unsubscribe = true
	err = w.FlushChannels()
	assert.NoError(t, err, "FlushChannels should not error")
}

func TestDisable(t *testing.T) {
	t.Parallel()
	w := Websocket{
		ShutdownC: make(chan struct{}),
	}
	w.setEnabled(true)
	w.setState(connected)
	require.NoError(t, w.Disable(), "Disable must not error")
	assert.ErrorIs(t, w.Disable(), ErrAlreadyDisabled, "Disable should error correctly")
}

func TestEnable(t *testing.T) {
	t.Parallel()
	w := Websocket{
		connector: connect,
		Wg:        new(sync.WaitGroup),
		ShutdownC: make(chan struct{}),
		GenerateSubs: func() ([]subscription.Subscription, error) {
			return []subscription.Subscription{{Channel: "test"}}, nil
		},
		Subscriber: func([]subscription.Subscription) error { return nil },
	}

	require.NoError(t, w.Enable(), "Enable must not error")
	assert.ErrorIs(t, w.Enable(), errWebsocketAlreadyEnabled, "Enable should error correctly")
}

func TestSetupNewConnection(t *testing.T) {
	t.Parallel()
	var nonsenseWebsock *Websocket
	err := nonsenseWebsock.SetupNewConnection(ConnectionSetup{URL: "urlstring"})
	assert.ErrorIs(t, err, errWebsocketIsNil, "SetupNewConnection should error correctly")

	nonsenseWebsock = &Websocket{}
	err = nonsenseWebsock.SetupNewConnection(ConnectionSetup{URL: "urlstring"})
	assert.ErrorIs(t, err, errExchangeConfigNameEmpty, "SetupNewConnection should error correctly")

	nonsenseWebsock = &Websocket{exchangeName: "test"}
	err = nonsenseWebsock.SetupNewConnection(ConnectionSetup{URL: "urlstring"})
	assert.ErrorIs(t, err, errTrafficAlertNil, "SetupNewConnection should error correctly")

	nonsenseWebsock.TrafficAlert = make(chan struct{}, 1)
	err = nonsenseWebsock.SetupNewConnection(ConnectionSetup{URL: "urlstring"})
	assert.ErrorIs(t, err, errReadMessageErrorsNil, "SetupNewConnection should error correctly")

	web := NewWebsocket()

	err = web.Setup(defaultSetup)
	assert.NoError(t, err, "Setup should not error")

	err = web.SetupNewConnection(ConnectionSetup{})
	assert.ErrorIs(t, err, errExchangeConfigEmpty, "SetupNewConnection should error correctly")

	err = web.SetupNewConnection(ConnectionSetup{URL: "urlstring"})
	assert.NoError(t, err, "SetupNewConnection should not error")

	err = web.SetupNewConnection(ConnectionSetup{URL: "urlstring", Authenticated: true})
	assert.NoError(t, err, "SetupNewConnection should not error")
}

func TestWebsocketConnectionShutdown(t *testing.T) {
	t.Parallel()
	wc := WebsocketConnection{}
	err := wc.Shutdown()
	assert.NoError(t, err, "Shutdown should not error")

	err = wc.Dial(&websocket.Dialer{}, nil)
	assert.ErrorContains(t, err, "malformed ws or wss URL", "Dial must error correctly")

	wc.URL = websocketTestURL

	err = wc.Dial(&websocket.Dialer{}, nil)
	require.NoError(t, err, "Dial must not error")

	err = wc.Shutdown()
	require.NoError(t, err, "Shutdown must not error")
}

// TestLatency logic test
func TestLatency(t *testing.T) {
	t.Parallel()
	r := &reporter{}
	exch := "Kraken"
	wc := &WebsocketConnection{
		ExchangeName:     exch,
		Verbose:          true,
		URL:              "wss://ws.kraken.com",
		ResponseMaxLimit: time.Second * 5,
		Match:            NewMatch(),
		Reporter:         r,
	}
	if wc.ProxyURL != "" && !useProxyTests {
		t.Skip("Proxy testing not enabled, skipping")
	}

	err := wc.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}

	go readMessages(t, wc)

	request := testRequest{
		Event: "subscribe",
		Pairs: []string{currency.NewPairWithDelimiter("XBT", "USD", "/").String()},
		Subscription: testRequestData{
			Name: "ticker",
		},
		RequestID: wc.GenerateMessageID(false),
	}

	_, err = wc.SendMessageReturnResponse(request.RequestID, request)
	if err != nil {
		t.Error(err)
	}

	if r.t == 0 {
		t.Error("expected a nonzero duration, got zero")
	}

	if r.name != exch {
		t.Errorf("expected %v, got %v", exch, r.name)
	}
}

func TestCheckSubscriptions(t *testing.T) {
	t.Parallel()
	ws := Websocket{}
	err := ws.checkSubscriptions(nil)
	assert.ErrorIs(t, err, errNoSubscriptionsSupplied, "checkSubscriptions should error correctly")

	ws.MaxSubscriptionsPerConnection = 1

	err = ws.checkSubscriptions([]subscription.Subscription{{}, {}})
	assert.ErrorIs(t, err, errSubscriptionsExceedsLimit, "checkSubscriptions should error correctly")

	ws.MaxSubscriptionsPerConnection = 2

	ws.subscriptions = subscriptionMap{42: {Key: 42, Channel: "test"}}
	err = ws.checkSubscriptions([]subscription.Subscription{{Key: 42, Channel: "test"}})
	assert.ErrorIs(t, err, errChannelAlreadySubscribed, "checkSubscriptions should error correctly")

	err = ws.checkSubscriptions([]subscription.Subscription{{}})
	assert.NoError(t, err, "checkSubscriptions should not error")
}
