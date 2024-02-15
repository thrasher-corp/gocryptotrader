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
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
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
		Name:                    "exchangeName",
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
	return errors.New("cannot shutdown due to some dastardly reason")
}

// override websocket connection method to produce a wicked terrible error
func (d *dodgyConnection) Connect() error {
	return errors.New("cannot connect due to some dastardly reason")
}

func TestSetup(t *testing.T) {
	t.Parallel()
	var w *Websocket
	err := w.Setup(nil)
	if !errors.Is(err, errWebsocketIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWebsocketIsNil)
	}

	w = &Websocket{DataHandler: make(chan interface{})}
	err = w.Setup(nil)
	if !errors.Is(err, errWebsocketSetupIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWebsocketSetupIsNil)
	}

	websocketSetup := &WebsocketSetup{}
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errWebsocketAlreadyInitialised) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWebsocketAlreadyInitialised)
	}

	w.Init = true
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errExchangeConfigIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeConfigIsNil)
	}

	websocketSetup.ExchangeConfig = &config.Exchange{}
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errExchangeConfigNameUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeConfigNameUnset)
	}
	websocketSetup.ExchangeConfig.Name = "testname"

	err = w.Setup(websocketSetup)
	if !errors.Is(err, errWebsocketFeaturesIsUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWebsocketFeaturesIsUnset)
	}

	websocketSetup.Features = &protocol.Features{}
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errConfigFeaturesIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errConfigFeaturesIsNil)
	}

	websocketSetup.ExchangeConfig.Features = &config.FeaturesConfig{}
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errWebsocketConnectorUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWebsocketConnectorUnset)
	}

	websocketSetup.Connector = func() error { return nil }
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errWebsocketSubscriberUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWebsocketSubscriberUnset)
	}

	websocketSetup.Subscriber = func([]subscription.Subscription) error { return nil }
	websocketSetup.Features.Unsubscribe = true
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errWebsocketUnsubscriberUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWebsocketUnsubscriberUnset)
	}

	websocketSetup.Unsubscriber = func([]subscription.Subscription) error { return nil }
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errWebsocketSubscriptionsGeneratorUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWebsocketSubscriptionsGeneratorUnset)
	}

	websocketSetup.GenerateSubscriptions = func() ([]subscription.Subscription, error) { return nil, nil }
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errDefaultURLIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errDefaultURLIsEmpty)
	}

	websocketSetup.DefaultURL = "test"
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errRunningURLIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errRunningURLIsEmpty)
	}

	websocketSetup.RunningURL = "http://www.google.com"
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errInvalidWebsocketURL) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidWebsocketURL)
	}

	websocketSetup.RunningURL = "wss://www.google.com"
	websocketSetup.RunningURLAuth = "http://www.google.com"
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errInvalidWebsocketURL) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidWebsocketURL)
	}

	websocketSetup.RunningURLAuth = "wss://www.google.com"
	err = w.Setup(websocketSetup)
	if !errors.Is(err, errInvalidTrafficTimeout) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidTrafficTimeout)
	}

	websocketSetup.ExchangeConfig.WebsocketTrafficTimeout = time.Minute
	err = w.Setup(websocketSetup)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestTrafficMonitorTimeout(t *testing.T) {
	t.Parallel()
	ws := *New()
	if err := ws.Setup(defaultSetup); err != nil {
		t.Fatal(err)
	}
	ws.trafficTimeout = time.Second * 2
	ws.ShutdownC = make(chan struct{})
	ws.trafficMonitor()
	if !ws.IsTrafficMonitorRunning() {
		t.Fatal("traffic monitor should be running")
	}
	// Deploy traffic alert
	ws.TrafficAlert <- struct{}{}
	// try to add another traffic monitor
	ws.trafficMonitor()
	if !ws.IsTrafficMonitorRunning() {
		t.Fatal("traffic monitor should be running")
	}
	// prevent shutdown routine
	ws.setConnectedStatus(false)
	// await timeout closure
	ws.Wg.Wait()
	if ws.IsTrafficMonitorRunning() {
		t.Error("should be dead")
	}
}

func TestIsDisconnectionError(t *testing.T) {
	t.Parallel()
	isADisconnectionError := IsDisconnectionError(errors.New("errorText"))
	if isADisconnectionError {
		t.Error("Its not")
	}
	isADisconnectionError = IsDisconnectionError(&websocket.CloseError{
		Code: 1006,
		Text: "errorText",
	})
	if !isADisconnectionError {
		t.Error("It is")
	}

	isADisconnectionError = IsDisconnectionError(&net.OpError{
		Err: errClosedConnection,
	})
	if isADisconnectionError {
		t.Error("It's not")
	}

	isADisconnectionError = IsDisconnectionError(&net.OpError{
		Err: errors.New("errText"),
	})
	if !isADisconnectionError {
		t.Error("It is")
	}
}

func TestConnectionMessageErrors(t *testing.T) {
	t.Parallel()
	var wsWrong = &Websocket{}
	err := wsWrong.Connect()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	wsWrong.connector = func() error { return nil }
	err = wsWrong.Connect()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	wsWrong.setEnabled(true)
	wsWrong.setConnectingStatus(true)
	wsWrong.Wg = &sync.WaitGroup{}
	err = wsWrong.Connect()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	wsWrong.setConnectedStatus(false)
	wsWrong.connector = func() error { return errors.New("edge case error of dooooooom") }
	err = wsWrong.Connect()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	ws := *New()
	err = ws.Setup(defaultSetup)
	if err != nil {
		t.Fatal(err)
	}
	ws.trafficTimeout = time.Minute
	ws.connector = func() error { return nil }

	err = ws.Connect()
	if err != nil {
		t.Fatal(err)
	}

	ws.TrafficAlert <- struct{}{}

	timer := time.NewTimer(900 * time.Millisecond)
	ws.ReadMessageErrors <- errors.New("errorText")
	select {
	case err := <-ws.ToRoutine:
		errText, ok := err.(error)
		if !ok {
			t.Error("unable to type assert error")
		} else if errText.Error() != "errorText" {
			t.Errorf("Expected 'errorText', received %v", err)
		}
	case <-timer.C:
		t.Error("Timeout waiting for datahandler to receive error")
	}
	ws.ReadMessageErrors <- &websocket.CloseError{
		Code: 1006,
		Text: "errorText",
	}
outer:
	for {
		select {
		case err := <-ws.ToRoutine:
			if _, ok := err.(*websocket.CloseError); !ok {
				t.Errorf("Error is not a disconnection error: %v", err)
			}
		case <-timer.C:
			break outer
		}
	}
}

func TestWebsocket(t *testing.T) {
	t.Parallel()
	wsInit := Websocket{}
	err := wsInit.Setup(&WebsocketSetup{
		ExchangeConfig: &config.Exchange{
			Features: &config.FeaturesConfig{
				Enabled: config.FeaturesEnabledConfig{Websocket: true},
			},
			Name: "test",
		},
	})
	if !errors.Is(err, errWebsocketAlreadyInitialised) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWebsocketAlreadyInitialised)
	}

	ws := *New()
	err = ws.SetProxyAddress("garbagio")
	if err == nil {
		t.Error("error cannot be nil")
	}

	ws.Conn = &WebsocketConnection{}
	ws.AuthConn = &WebsocketConnection{}
	ws.setEnabled(true)
	err = ws.SetProxyAddress("https://192.168.0.1:1337")
	if err == nil {
		t.Error("error cannot be nil")
	}
	ws.setConnectedStatus(true)
	ws.ShutdownC = make(chan struct{})
	ws.Wg = &sync.WaitGroup{}
	err = ws.SetProxyAddress("https://192.168.0.1:1336")
	if err == nil {
		t.Error("SetProxyAddress", err)
	}

	err = ws.SetProxyAddress("https://192.168.0.1:1336")
	if err == nil {
		t.Error("SetProxyAddress", err)
	}
	ws.setEnabled(false)

	// removing proxy
	err = ws.SetProxyAddress("")
	if err != nil {
		t.Error(err)
	}
	// reinstate proxy
	err = ws.SetProxyAddress("http://localhost:1337")
	if err != nil {
		t.Error(err)
	}
	// conflict proxy
	err = ws.SetProxyAddress("http://localhost:1337")
	if err == nil {
		t.Error("error cannot be nil")
	}
	err = ws.Setup(defaultSetup)
	if err != nil {
		t.Fatal(err)
	}
	if ws.GetName() != "exchangeName" {
		t.Error("WebsocketSetup")
	}

	if !ws.IsEnabled() {
		t.Error("WebsocketSetup")
	}

	ws.setEnabled(false)
	if ws.IsEnabled() {
		t.Error("WebsocketSetup")
	}
	ws.setEnabled(true)
	if !ws.IsEnabled() {
		t.Error("WebsocketSetup")
	}

	if ws.GetProxyAddress() != "http://localhost:1337" {
		t.Error("WebsocketSetup")
	}

	if ws.GetWebsocketURL() != "wss://testRunningURL" {
		t.Error("WebsocketSetup")
	}
	if ws.trafficTimeout != time.Second*5 {
		t.Error("WebsocketSetup")
	}
	// -- Not connected shutdown
	err = ws.Shutdown()
	if err == nil {
		t.Fatal("should not be connected to able to shut down")
	}

	ws.setConnectedStatus(true)
	ws.Conn = &dodgyConnection{}
	err = ws.Shutdown()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	ws.Conn = &WebsocketConnection{}

	ws.setConnectedStatus(true)
	ws.AuthConn = &dodgyConnection{}
	err = ws.Shutdown()
	if err == nil {
		t.Fatal("error cannot be nil ")
	}

	ws.AuthConn = &WebsocketConnection{}
	ws.setConnectedStatus(false)

	// -- Normal connect
	err = ws.Connect()
	if err != nil {
		t.Fatal("WebsocketSetup", err)
	}

	ws.defaultURL = "ws://demos.kaazing.com/echo"
	ws.defaultURLAuth = "ws://demos.kaazing.com/echo"

	err = ws.SetWebsocketURL("", false, false)
	if err != nil {
		t.Fatal(err)
	}
	err = ws.SetWebsocketURL("ws://demos.kaazing.com/echo", false, false)
	if err != nil {
		t.Fatal(err)
	}
	err = ws.SetWebsocketURL("", true, false)
	if err != nil {
		t.Fatal(err)
	}
	err = ws.SetWebsocketURL("ws://demos.kaazing.com/echo", true, false)
	if err != nil {
		t.Fatal(err)
	}
	// Attempt reconnect
	err = ws.SetWebsocketURL("ws://demos.kaazing.com/echo", true, true)
	if err != nil {
		t.Fatal(err)
	}
	// -- initiate the reconnect which is usually handled by connection monitor
	err = ws.Connect()
	if err != nil {
		t.Fatal(err)
	}
	err = ws.Connect()
	if err == nil {
		t.Fatal("should already be connected")
	}
	// -- Normal shutdown
	err = ws.Shutdown()
	if err != nil {
		t.Fatal("WebsocketSetup", err)
	}
	ws.Wg.Wait()
}

// TestSubscribe logic test
func TestSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	ws := *New()
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
	ws := *New()

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
	ws := New()

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
	ws := New()

	c := &subscription.Subscription{Key: 42, Channel: "Unite!"}
	assert.NoError(t, ws.AddSubscription(c), "Adding first subscription should not error")
	assert.NotNil(t, ws.GetSubscription(42), "Added subscription should be findable")

	ws.RemoveSubscriptions(*c)
	assert.Nil(t, ws.GetSubscription(42), "Remove should have removed the sub")
}

// TestConnectionMonitorNoConnection logic test
func TestConnectionMonitorNoConnection(t *testing.T) {
	t.Parallel()
	ws := *New()
	ws.connectionMonitorDelay = 500
	ws.DataHandler = make(chan interface{}, 1)
	ws.ShutdownC = make(chan struct{}, 1)
	ws.exchangeName = "hello"
	ws.Wg = &sync.WaitGroup{}
	ws.enabled = true
	err := ws.connectionMonitor()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}
	if !ws.IsConnectionMonitorRunning() {
		t.Fatal("Should not have exited")
	}
	err = ws.connectionMonitor()
	if !errors.Is(err, errAlreadyRunning) {
		t.Fatalf("received: %v, but expected: %v", err, errAlreadyRunning)
	}
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
	ws := *New()
	result := ws.CanUseAuthenticatedEndpoints()
	if result {
		t.Error("expected `canUseAuthenticatedEndpoints` to be false")
	}
	ws.SetCanUseAuthenticatedEndpoints(true)
	result = ws.CanUseAuthenticatedEndpoints()
	if !result {
		t.Error("expected `canUseAuthenticatedEndpoints` to be true")
	}
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
	w := gzip.NewWriter(&b)
	_, err := w.Write([]byte("hello"))
	if err != nil {
		t.Error(err)
	}
	err = w.Close()
	if err != nil {
		t.Error(err)
	}
	var resp []byte
	resp, err = wc.parseBinaryResponse(b.Bytes())
	if err != nil {
		t.Error(err)
	}
	if !strings.EqualFold(string(resp), "hello") {
		t.Errorf("GZip conversion failed. Received: '%v', Expected: 'hello'", string(resp))
	}

	var b2 bytes.Buffer
	w2, err2 := flate.NewWriter(&b2, 1)
	if err2 != nil {
		t.Error(err2)
	}
	_, err2 = w2.Write([]byte("hello"))
	if err2 != nil {
		t.Error(err)
	}
	err2 = w2.Close()
	if err2 != nil {
		t.Error(err)
	}
	resp2, err3 := wc.parseBinaryResponse(b2.Bytes())
	if err3 != nil {
		t.Error(err3)
	}
	if !strings.EqualFold(string(resp2), "hello") {
		t.Errorf("Deflate conversion failed. Received: '%v', Expected: 'hello'", string(resp2))
	}

	_, err4 := wc.parseBinaryResponse([]byte{})
	if err4 == nil || err4.Error() != "unexpected EOF" {
		t.Error("Expected error 'unexpected EOF'")
	}
}

// TestCanUseAuthenticatedWebsocketForWrapper logic test
func TestCanUseAuthenticatedWebsocketForWrapper(t *testing.T) {
	t.Parallel()
	ws := &Websocket{}
	resp := ws.CanUseAuthenticatedWebsocketForWrapper()
	if resp {
		t.Error("Expected false, `connected` is false")
	}
	ws.setConnectedStatus(true)
	resp = ws.CanUseAuthenticatedWebsocketForWrapper()
	if resp {
		t.Error("Expected false, `connected` is true and `CanUseAuthenticatedEndpoints` is false")
	}
	ws.canUseAuthenticatedEndpoints = true
	resp = ws.CanUseAuthenticatedWebsocketForWrapper()
	if !resp {
		t.Error("Expected true, `connected` and `CanUseAuthenticatedEndpoints` is true")
	}
}

func TestGenerateMessageID(t *testing.T) {
	t.Parallel()
	wc := WebsocketConnection{}
	var id int64
	for i := 0; i < 10; i++ {
		newID := wc.GenerateMessageID(true)
		if id == newID {
			t.Fatal("ID generation is not unique")
		}
		id = newID
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
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = checkWebsocketURL("wowowow:wowowowo")
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = checkWebsocketURL("://")
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = checkWebsocketURL("http://www.google.com")
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = checkWebsocketURL("wss://websocketconnection.place")
	if err != nil {
		t.Fatal(err)
	}

	err = checkWebsocketURL("ws://websocketconnection.place")
	if err != nil {
		t.Fatal(err)
	}
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
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	dodgyWs.setEnabled(true)
	err = dodgyWs.FlushChannels()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	web := Websocket{
		enabled:      true,
		connected:    true,
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

	problemFunc := func() ([]subscription.Subscription, error) {
		return nil, errors.New("problems")
	}

	noSub := func() ([]subscription.Subscription, error) {
		return nil, nil
	}

	// Disable pair and flush system
	newgen.EnabledPairs = []currency.Pair{
		currency.NewPair(currency.BTC, currency.AUD)}
	web.GenerateSubs = func() ([]subscription.Subscription, error) {
		return []subscription.Subscription{{Channel: "test"}}, nil
	}
	err = web.FlushChannels()
	if err != nil {
		t.Fatal(err)
	}

	web.features.FullPayloadSubscribe = true
	web.GenerateSubs = problemFunc
	err = web.FlushChannels() // error on full subscribeToChannels
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	web.GenerateSubs = noSub
	err = web.FlushChannels() // No subs to sub
	if err != nil {
		t.Fatal(err)
	}

	web.GenerateSubs = newgen.generateSubs
	subs, err := web.GenerateSubs()
	if err != nil {
		t.Fatal(err)
	}
	web.AddSuccessfulSubscriptions(subs...)
	err = web.FlushChannels()
	if err != nil {
		t.Fatal(err)
	}
	web.features.FullPayloadSubscribe = false
	web.features.Subscribe = true

	web.GenerateSubs = problemFunc
	err = web.FlushChannels()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	web.GenerateSubs = newgen.generateSubs
	err = web.FlushChannels()
	if err != nil {
		t.Fatal(err)
	}
	web.subscriptionMutex.Lock()
	web.subscriptions = subscriptionMap{
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
	web.subscriptionMutex.Unlock()

	err = web.FlushChannels()
	if err != nil {
		t.Fatal(err)
	}

	err = web.FlushChannels()
	if err != nil {
		t.Fatal(err)
	}

	web.setConnectedStatus(true)
	web.features.Unsubscribe = true
	err = web.FlushChannels()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDisable(t *testing.T) {
	t.Parallel()
	web := Websocket{
		enabled:   true,
		connected: true,
		ShutdownC: make(chan struct{}),
	}
	err := web.Disable()
	if err != nil {
		t.Fatal(err)
	}
	err = web.Disable()
	if err == nil {
		t.Fatal("should already be disabled")
	}
}

func TestEnable(t *testing.T) {
	t.Parallel()
	web := Websocket{
		connector: connect,
		Wg:        new(sync.WaitGroup),
		ShutdownC: make(chan struct{}),
		GenerateSubs: func() ([]subscription.Subscription, error) {
			return []subscription.Subscription{{Channel: "test"}}, nil
		},
		Subscriber: func([]subscription.Subscription) error { return nil },
	}

	err := web.Enable()
	if err != nil {
		t.Fatal(err)
	}

	err = web.Enable()
	if err == nil {
		t.Fatal("should already be enabled")
	}

	fmt.Print()
}

func TestSetupNewConnection(t *testing.T) {
	t.Parallel()
	var nonsenseWebsock *Websocket
	err := nonsenseWebsock.SetupNewConnection(ConnectionSetup{URL: "urlstring"})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	nonsenseWebsock = &Websocket{}
	err = nonsenseWebsock.SetupNewConnection(ConnectionSetup{URL: "urlstring"})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	nonsenseWebsock = &Websocket{exchangeName: "test"}
	err = nonsenseWebsock.SetupNewConnection(ConnectionSetup{URL: "urlstring"})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	nonsenseWebsock.TrafficAlert = make(chan struct{})
	err = nonsenseWebsock.SetupNewConnection(ConnectionSetup{URL: "urlstring"})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	web := Websocket{
		connector:         connect,
		Wg:                new(sync.WaitGroup),
		ShutdownC:         make(chan struct{}),
		Init:              true,
		TrafficAlert:      make(chan struct{}),
		ReadMessageErrors: make(chan error),
		DataHandler:       make(chan interface{}),
	}

	err = web.Setup(defaultSetup)
	if err != nil {
		t.Fatal(err)
	}
	err = web.SetupNewConnection(ConnectionSetup{})
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	err = web.SetupNewConnection(ConnectionSetup{URL: "urlstring"})
	if err != nil {
		t.Fatal(err)
	}
	err = web.SetupNewConnection(ConnectionSetup{URL: "urlstring",
		Authenticated: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebsocketConnectionShutdown(t *testing.T) {
	t.Parallel()
	wc := WebsocketConnection{}
	err := wc.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	err = wc.Dial(&websocket.Dialer{}, nil)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	wc.URL = websocketTestURL

	err = wc.Dial(&websocket.Dialer{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = wc.Shutdown()
	if err != nil {
		t.Fatal(err)
	}
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
	if !errors.Is(err, errNoSubscriptionsSupplied) {
		t.Fatalf("received: %v, but expected: %v", err, errNoSubscriptionsSupplied)
	}

	ws.MaxSubscriptionsPerConnection = 1

	err = ws.checkSubscriptions([]subscription.Subscription{{}, {}})
	if !errors.Is(err, errSubscriptionsExceedsLimit) {
		t.Fatalf("received: %v, but expected: %v", err, errSubscriptionsExceedsLimit)
	}

	ws.MaxSubscriptionsPerConnection = 2

	ws.subscriptions = subscriptionMap{42: {Key: 42, Channel: "test"}}
	err = ws.checkSubscriptions([]subscription.Subscription{{Key: 42, Channel: "test"}})
	if !errors.Is(err, errChannelAlreadySubscribed) {
		t.Fatalf("received: %v, but expected: %v", err, errChannelAlreadySubscribed)
	}

	err = ws.checkSubscriptions([]subscription.Subscription{{}})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, but expected: %v", err, nil)
	}
}
