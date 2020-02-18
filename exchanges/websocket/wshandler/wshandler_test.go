package wshandler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

func TestTrafficMonitorTimeout(t *testing.T) {
	ws := New()
	err := ws.Setup(
		&WebsocketSetup{
			Enabled:                          true,
			AuthenticatedWebsocketAPISupport: true,
			WebsocketTimeout:                 10000,
			DefaultURL:                       "testDefaultURL",
			ExchangeName:                     "exchangeName",
			RunningURL:                       "testRunningURL",
			Connector:                        func() error { return nil },
			Subscriber:                       func(test WebsocketChannelSubscription) error { return nil },
			UnSubscriber:                     func(test WebsocketChannelSubscription) error { return nil },
		})
	if err != nil {
		t.Error(err)
	}
	ws.setConnectedStatus(true)
	ws.TrafficAlert = make(chan struct{}, 2)
	ws.ShutdownC = make(chan struct{})
	var anotherWG sync.WaitGroup
	anotherWG.Add(1)
	go ws.trafficMonitor(&anotherWG)
	anotherWG.Wait()
	ws.TrafficAlert <- struct{}{}
	trafficTimer := time.NewTimer(5 * time.Second)
	select {
	case <-trafficTimer.C:
		t.Error("should be exiting")
	default:
		ws.Wg.Wait()
	}
}

func TestIsDisconnectionError(t *testing.T) {
	isADisconnectionError := isDisconnectionError(errors.New("errorText"))
	if isADisconnectionError {
		t.Error("Its not")
	}
	isADisconnectionError = isDisconnectionError(&websocket.CloseError{
		Code: 1006,
		Text: "errorText",
	})
	if !isADisconnectionError {
		t.Error("It is")
	}

	isADisconnectionError = isDisconnectionError(&net.OpError{
		Op:     "",
		Net:    "",
		Source: nil,
		Addr:   nil,
		Err:    errors.New("errorText"),
	})
	if !isADisconnectionError {
		t.Error("It is")
	}
}

func TestConnectionMessageErrors(t *testing.T) {
	ws := New()
	ws.connected = true
	ws.enabled = true
	ws.ReadMessageErrors = make(chan error)
	ws.DataHandler = make(chan interface{})
	ws.ShutdownC = make(chan struct{})
	ws.connector = func() error { return nil }
	ws.features = &protocol.Features{}
	go ws.connectionMonitor()
	timer := time.NewTimer(900 * time.Millisecond)
	ws.ReadMessageErrors <- errors.New("errorText")
	select {
	case err := <-ws.DataHandler:
		if err.(error).Error() != "errorText" {
			t.Errorf("Expected 'errorText', received %v", err)
		}
	case <-timer.C:
		t.Error("Timeout waiting for datahandler to receive error")
	}
	timer = time.NewTimer(900 * time.Millisecond)
	ws.ReadMessageErrors <- &websocket.CloseError{
		Code: 1006,
		Text: "errorText",
	}
outer:
	for {
		select {
		case <-ws.DataHandler:
			t.Fatal("Error is a disconnection error")
		case <-timer.C:
			break outer
		}
	}
}

func TestWebsocket(t *testing.T) {
	ws := Websocket{}
	ws.setInit(true)
	err := ws.Setup(&WebsocketSetup{
		ExchangeName: "test",
		Enabled:      true,
	})
	if err != nil && err.Error() != "test Websocket already initialised" {
		t.Errorf("Expected 'test Websocket already initialised', received %v", err)
	}

	ws = *New()
	err = ws.SetProxyAddress("testProxy")
	if err != nil {
		t.Error("SetProxyAddress", err)
	}

	err = ws.Setup(
		&WebsocketSetup{
			Enabled:                          true,
			AuthenticatedWebsocketAPISupport: true,
			WebsocketTimeout:                 2,
			DefaultURL:                       "testDefaultURL",
			ExchangeName:                     "exchangeName",
			RunningURL:                       "testRunningURL",
			Connector:                        func() error { return nil },
			Subscriber:                       func(test WebsocketChannelSubscription) error { return nil },
			UnSubscriber:                     func(test WebsocketChannelSubscription) error { return nil },
			Features:                         &protocol.Features{},
		})
	if err != nil {
		t.Error(err)
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

	if ws.GetProxyAddress() != "testProxy" {
		t.Error("WebsocketSetup")
	}

	if ws.GetDefaultURL() != "testDefaultURL" {
		t.Error("WebsocketSetup")
	}

	if ws.GetWebsocketURL() != "testRunningURL" {
		t.Error("WebsocketSetup")
	}

	if ws.trafficTimeout != time.Duration(2) {
		t.Error("WebsocketSetup")
	}
	// -- Not connected shutdown
	err = ws.Shutdown()
	if err == nil {
		t.Fatal("should not be connected to able to shut down")
	}
	ws.Wg.Wait()
	// -- Normal connect
	err = ws.Connect()
	if err != nil {
		t.Fatal("WebsocketSetup", err)
	}
	ws.SetWebsocketURL("ws://demos.kaazing.com/echo")
	// -- Already connected connect
	err = ws.Connect()
	if err == nil {
		t.Fatal("should not connect, already connected")
	}
	// -- Normal shutdown
	err = ws.Shutdown()
	if err != nil {
		t.Fatal("WebsocketSetup", err)
	}
	ws.Wg.Wait()
}

// placeholderSubscriber basic function to test subscriptions
func placeholderSubscriber(channelToSubscribe WebsocketChannelSubscription) error {
	return nil
}

// TestSubscribe logic test
func TestSubscribe(t *testing.T) {
	w := Websocket{
		channelsToSubscribe: []WebsocketChannelSubscription{
			{
				Channel: "hello",
			},
		},
		subscribedChannels: []WebsocketChannelSubscription{},
	}
	w.SetChannelSubscriber(placeholderSubscriber)
	err := w.appendSubscribedChannels()
	if err != nil {
		t.Error(err)
	}
	if len(w.subscribedChannels) != 1 {
		t.Errorf("Subscription did not occur")
	}
}

// TestSubscribe logic test
func TestSubscribeToChannels(t *testing.T) {
	w := Websocket{
		channelsToSubscribe: []WebsocketChannelSubscription{
			{
				Channel: "hello",
			},
		},
		subscribedChannels: []WebsocketChannelSubscription{},
	}
	w.SetChannelSubscriber(placeholderSubscriber)
	w.SubscribeToChannels([]WebsocketChannelSubscription{{Channel: "hello"}, {Channel: "hello2"}})
	if len(w.channelsToSubscribe) != 2 {
		t.Errorf("Subscription did not occur")
	}
}

// TestUnsubscribe logic test
func TestUnsubscribe(t *testing.T) {
	w := Websocket{
		channelsToSubscribe: []WebsocketChannelSubscription{},
		subscribedChannels: []WebsocketChannelSubscription{
			{
				Channel: "hello",
			},
		},
	}
	w.SetChannelUnsubscriber(placeholderSubscriber)
	err := w.unsubscribeToChannels()
	if err != nil {
		t.Error(err)
	}
	if len(w.subscribedChannels) != 0 {
		t.Errorf("Unsubscription did not occur")
	}
}

// TestSubscriptionWithExistingEntry logic test
func TestSubscriptionWithExistingEntry(t *testing.T) {
	w := Websocket{
		channelsToSubscribe: []WebsocketChannelSubscription{
			{
				Channel: "hello",
			},
		},
		subscribedChannels: []WebsocketChannelSubscription{
			{
				Channel: "hello",
			},
		},
	}
	w.SetChannelSubscriber(placeholderSubscriber)
	err := w.appendSubscribedChannels()
	if err != nil {
		t.Error(err)
	}
	if len(w.subscribedChannels) != 1 {
		t.Errorf("Subscription should not have occurred")
	}
}

// TestUnsubscriptionWithExistingEntry logic test
func TestUnsubscriptionWithExistingEntry(t *testing.T) {
	w := Websocket{
		channelsToSubscribe: []WebsocketChannelSubscription{
			{
				Channel: "hello",
			},
		},
		subscribedChannels: []WebsocketChannelSubscription{
			{
				Channel: "hello",
			},
		},
	}
	w.SetChannelUnsubscriber(placeholderSubscriber)
	err := w.unsubscribeToChannels()
	if err != nil {
		t.Error(err)
	}
	if len(w.subscribedChannels) != 1 {
		t.Errorf("Unsubscription should not have occurred")
	}
}

// TestManageSubscriptionsStartStop logic test
func TestManageSubscriptionsStartStop(t *testing.T) {
	w := Websocket{
		ShutdownC: make(chan struct{}),
		features:  &protocol.Features{Subscribe: true, Unsubscribe: true},
	}
	w.Wg.Add(1)
	go w.manageSubscriptions()
	close(w.ShutdownC)
	w.Wg.Wait()
}

// TestManageSubscriptions logic test
func TestManageSubscriptions(t *testing.T) {
	w := Websocket{
		ShutdownC: make(chan struct{}),
		features:  &protocol.Features{Subscribe: true, Unsubscribe: true},
		subscribedChannels: []WebsocketChannelSubscription{
			{
				Channel: "hello",
			},
		},
	}
	w.SetChannelUnsubscriber(placeholderSubscriber)
	w.SetChannelSubscriber(placeholderSubscriber)
	w.setConnectedStatus(true)
	go w.manageSubscriptions()
	time.Sleep(8 * time.Second)
	w.setConnectedStatus(false)
	time.Sleep(manageSubscriptionsDelay)
	w.subscriptionMutex.Lock()
	if len(w.subscribedChannels) > 0 {
		t.Error("Expected empty subscribed channels")
	}
	w.subscriptionMutex.Unlock()
}

// TestConnectionMonitorNoConnection logic test
func TestConnectionMonitorNoConnection(t *testing.T) {
	ws := New()
	ws.DataHandler = make(chan interface{}, 1)
	ws.ShutdownC = make(chan struct{}, 1)
	ws.exchangeName = "hello"
	ws.trafficTimeout = 1
	go ws.connectionMonitor()
	if ws.IsConnectionMonitorRunning() {
		t.Fatal("Should have exited")
	}
}

// TestRemoveChannelToSubscribe logic test
func TestRemoveChannelToSubscribe(t *testing.T) {
	subscription := WebsocketChannelSubscription{
		Channel: "hello",
	}
	w := Websocket{
		channelsToSubscribe: []WebsocketChannelSubscription{
			subscription,
		},
	}
	w.SetChannelUnsubscriber(placeholderSubscriber)
	w.removeChannelToSubscribe(subscription)
	if len(w.subscribedChannels) != 0 {
		t.Errorf("Unsubscription did not occur")
	}
}

// TestRemoveChannelToSubscribeWithNoSubscription logic test
func TestRemoveChannelToSubscribeWithNoSubscription(t *testing.T) {
	subscription := WebsocketChannelSubscription{
		Channel: "hello",
	}
	w := Websocket{
		channelsToSubscribe: []WebsocketChannelSubscription{},
	}
	w.DataHandler = make(chan interface{}, 1)
	w.SetChannelUnsubscriber(placeholderSubscriber)
	go w.removeChannelToSubscribe(subscription)
	err := <-w.DataHandler
	if !strings.Contains(err.(error).Error(), "could not be removed because it was not found") {
		t.Error("Expected not found error")
	}
}

// TestResubscribeToChannel logic test
func TestResubscribeToChannel(t *testing.T) {
	subscription := WebsocketChannelSubscription{
		Channel: "hello",
	}
	w := Websocket{
		channelsToSubscribe: []WebsocketChannelSubscription{},
	}
	w.DataHandler = make(chan interface{}, 1)
	w.SetChannelUnsubscriber(placeholderSubscriber)
	w.SetChannelSubscriber(placeholderSubscriber)
	w.ResubscribeToChannel(subscription)
}

// TestSliceCopyDoesntImpactBoth logic test
func TestSliceCopyDoesntImpactBoth(t *testing.T) {
	w := Websocket{
		channelsToSubscribe: []WebsocketChannelSubscription{
			{
				Channel: "hello1",
			},
			{
				Channel: "hello2",
			},
		},
		subscribedChannels: []WebsocketChannelSubscription{
			{
				Channel: "hello3",
			},
		},
	}
	w.SetChannelUnsubscriber(placeholderSubscriber)
	err := w.unsubscribeToChannels()
	if err != nil {
		t.Error(err)
	}
	if len(w.subscribedChannels) != 2 {
		t.Errorf("Unsubscription did not occur")
	}
	w.subscribedChannels[0].Channel = "test"
	if strings.EqualFold(w.subscribedChannels[0].Channel, w.channelsToSubscribe[0].Channel) {
		t.Errorf("Slice has not been copied appropriately")
	}
}

// TestSliceCopyDoesntImpactBoth logic test
func TestGetSubscriptions(t *testing.T) {
	w := Websocket{
		subscribedChannels: []WebsocketChannelSubscription{
			{
				Channel: "hello3",
			},
		},
	}

	subs := w.GetSubscriptions()
	subs[0].Channel = "noHELLO"
	if strings.EqualFold(w.subscribedChannels[0].Channel, subs[0].Channel) {
		t.Error("Subscriptions was not copied properly")
	}
}

// TestSetCanUseAuthenticatedEndpoints logic test
func TestSetCanUseAuthenticatedEndpoints(t *testing.T) {
	ws := New()
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

func TestRemoveSubscribedChannels(t *testing.T) {
	w := Websocket{
		channelsToSubscribe: []WebsocketChannelSubscription{
			{
				Channel: "hello3",
			},
		},
	}

	w.RemoveSubscribedChannels([]WebsocketChannelSubscription{{Channel: "hello3"}})
	if len(w.channelsToSubscribe) == 1 {
		t.Error("Did not remove subscription")
	}
}

const (
	websocketTestURL  = "wss://www.bitmex.com/realtime"
	returnResponseURL = "wss://ws.kraken.com"
	useProxyTests     = false                     // Disabled by default. Freely available proxy servers that work all the time are difficult to find
	proxyURL          = "http://212.186.171.4:80" // Replace with a usable proxy server
)

var wc *WebsocketConnection
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

// TestMain setup test
func TestMain(m *testing.M) {
	wc = &WebsocketConnection{
		ExchangeName:         "test",
		URL:                  returnResponseURL,
		ResponseMaxLimit:     7000000000,
		ResponseCheckTimeout: 30000000,
	}
	os.Exit(m.Run())
}

// TestDial logic test
func TestDial(t *testing.T) {
	var testCases = []testStruct{
		{Error: nil, WC: WebsocketConnection{ExchangeName: "test1", Verbose: true, URL: websocketTestURL, RateLimit: 10, ResponseCheckTimeout: 30000000, ResponseMaxLimit: 7000000000}},
		{Error: errors.New(" Error: malformed ws or wss URL"), WC: WebsocketConnection{ExchangeName: "test2", Verbose: true, URL: "", ResponseCheckTimeout: 30000000, ResponseMaxLimit: 7000000000}},
		{Error: nil, WC: WebsocketConnection{ExchangeName: "test3", Verbose: true, URL: websocketTestURL, ProxyURL: proxyURL, ResponseCheckTimeout: 30000000, ResponseMaxLimit: 7000000000}},
	}
	for i := range testCases {
		testData := &testCases[i]
		t.Run(testData.WC.ExchangeName, func(t *testing.T) {
			if testData.WC.ProxyURL != "" && !useProxyTests {
				t.Skip("Proxy testing not enabled, skipping")
			}
			err := testData.WC.Dial(&dialer, http.Header{})
			if err != nil {
				if testData.Error != nil && err.Error() == testData.Error.Error() {
					return
				}
				t.Fatal(err)
			}
		})
	}
}

// TestSendMessage logic test
func TestSendMessage(t *testing.T) {
	var testCases = []testStruct{
		{Error: nil, WC: WebsocketConnection{ExchangeName: "test1", Verbose: true, URL: websocketTestURL, RateLimit: 10, ResponseCheckTimeout: 30000000, ResponseMaxLimit: 7000000000}},
		{Error: errors.New(" Error: malformed ws or wss URL"), WC: WebsocketConnection{ExchangeName: "test2", Verbose: true, URL: "", ResponseCheckTimeout: 30000000, ResponseMaxLimit: 7000000000}},
		{Error: nil, WC: WebsocketConnection{ExchangeName: "test3", Verbose: true, URL: websocketTestURL, ProxyURL: proxyURL, ResponseCheckTimeout: 30000000, ResponseMaxLimit: 7000000000}},
	}
	for i := range testCases {
		testData := &testCases[i]
		t.Run(testData.WC.ExchangeName, func(t *testing.T) {
			if testData.WC.ProxyURL != "" && !useProxyTests {
				t.Skip("Proxy testing not enabled, skipping")
			}
			err := testData.WC.Dial(&dialer, http.Header{})
			if err != nil {
				if testData.Error != nil && err.Error() == testData.Error.Error() {
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
	if wc.ProxyURL != "" && !useProxyTests {
		t.Skip("Proxy testing not enabled, skipping")
	}
	err := wc.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go readMessages(wc, t)

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

// TestSetupPingHandler logic test
func TestSetupPingHandler(t *testing.T) {
	if wc.ProxyURL != "" && !useProxyTests {
		t.Skip("Proxy testing not enabled, skipping")
	}
	wc.Shutdown = make(chan struct{})
	err := wc.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}

	wc.SetupPingHandler(WebsocketPingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PingMessage,
		Delay:             1000,
	})

	err = wc.Connection.Close()
	if err != nil {
		t.Error(err)
	}

	err = wc.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	wc.SetupPingHandler(WebsocketPingHandler{
		MessageType: websocket.TextMessage,
		Message:     []byte(Ping),
		Delay:       200,
	})
	time.Sleep(time.Millisecond * 500)
	close(wc.Shutdown)
	wc.Wg.Wait()
}

// TestParseBinaryResponse logic test
func TestParseBinaryResponse(t *testing.T) {
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
		t.Errorf("GZip conversion failed. Received: '%v', Expected: 'hello'", string(resp2))
	}
}

// TestSetResponseIDAndData logic test
func TestSetResponseIDAndData(t *testing.T) {
	wc.IDResponses = nil
	wc.SetResponseIDAndData(0, nil)
	wc.SetResponseIDAndData(1, []byte("hi"))
	if len(wc.IDResponses) != 2 {
		t.Error("Expected 2 entries")
	}
}

// TestIsIDWaitingForResponse logic test
func TestIsIDWaitingForResponse(t *testing.T) {
	wc.IDResponses = nil
	wc.SetResponseIDAndData(0, nil)
	wc.SetResponseIDAndData(1, []byte("hi"))
	if len(wc.IDResponses) != 2 {
		t.Error("Expected 2 entries")
	}
	if !wc.IsIDWaitingForResponse(0) {
		t.Error("Expected true")
	}
	if wc.IsIDWaitingForResponse(2) {
		t.Error("Expected false")
	}
	if wc.IsIDWaitingForResponse(1337) {
		t.Error("Expected false")
	}
}

// readMessages helper func
func readMessages(wc *WebsocketConnection, t *testing.T) {
	timer := time.NewTimer(20 * time.Second)
	for {
		select {
		case <-timer.C:
			return
		default:
			resp, err := wc.ReadMessage()
			if err != nil {
				t.Error(err)
				return
			}
			var incoming testResponse
			err = json.Unmarshal(resp.Raw, &incoming)
			if err != nil {
				t.Error(err)
				return
			}
			if incoming.RequestID > 0 {
				wc.SetResponseIDAndData(incoming.RequestID, resp.Raw)
				return
			}
		}
	}
}

// TestCanUseAuthenticatedWebsocketForWrapper logic test
func TestCanUseAuthenticatedWebsocketForWrapper(t *testing.T) {
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
