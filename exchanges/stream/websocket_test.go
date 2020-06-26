package stream

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

var defaultSetup = &WebsocketSetup{
	Enabled:                          true,
	AuthenticatedWebsocketAPISupport: true,
	WebsocketTimeout:                 time.Second * 5,
	DefaultURL:                       "testDefaultURL",
	ExchangeName:                     "exchangeName",
	RunningURL:                       "testRunningURL",
	Connector:                        func() error { return nil },
	Subscriber:                       func(_ []ChannelSubscription) error { return nil },
	UnSubscriber:                     func(_ []ChannelSubscription) error { return nil },
	GenerateSubscriptions: func() ([]ChannelSubscription, error) {
		return []ChannelSubscription{
			{Channel: "TestSub"},
			{Channel: "TestSub2"},
			{Channel: "TestSub3"},
			{Channel: "TestSub4"},
		}, nil
	},
	Features: &protocol.Features{Subscribe: true, Unsubscribe: true},
}

func TestTrafficMonitorTimeout(t *testing.T) {
	ws := *New()
	err := ws.Setup(defaultSetup)
	if err != nil {
		t.Fatal(err)
	}
	ws.ShutdownC = make(chan struct{})
	err = ws.trafficMonitor()
	if err != nil {
		t.Fatal(err)
	}
	// Deploy traffic alert
	ws.TrafficAlert <- struct{}{}

	// Check if traffic monitor has started
	err = ws.trafficMonitor()
	if err != nil {
		t.Fatal(err)
	}

	ws.Wg.Wait()

	// Start new instance then simulate shutdown
	err = ws.trafficMonitor()
	if err != nil {
		t.Fatal(err)
	}

	ws.Wg.Wait()
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
	ws := *New()
	ws.connected = true
	ws.enabled = true
	ws.readMessageErrors = make(chan error)
	ws.DataHandler = make(chan interface{})
	ws.ShutdownC = make(chan struct{})
	ws.connector = func() error { return nil }
	ws.features = &protocol.Features{}
	go ws.connectionMonitor()
	timer := time.NewTimer(900 * time.Millisecond)
	ws.readMessageErrors <- errors.New("errorText")
	select {
	case err := <-ws.DataHandler:
		if err.(error).Error() != "errorText" {
			t.Errorf("Expected 'errorText', received %v", err)
		}
	case <-timer.C:
		t.Error("Timeout waiting for datahandler to receive error")
	}
	timer = time.NewTimer(900 * time.Millisecond)
	ws.readMessageErrors <- &websocket.CloseError{
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

	if ws.GetProxyAddress() != "testProxy" {
		t.Error("WebsocketSetup")
	}

	if ws.GetWebsocketURL() != "testRunningURL" {
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
	ws.Wg.Wait()
	// -- Normal connect
	err = ws.Connect()
	if err != nil {
		t.Fatal("WebsocketSetup", err)
	}
	err = ws.SetWebsocketURL("ws://demos.kaazing.com/echo", false)
	if err != nil {
		t.Fatal(err)
	}
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

// TestSubscribe logic test
func TestSubscribeUnsubscribe(t *testing.T) {
	ws := *New()
	err := ws.Setup(defaultSetup)
	if err != nil {
		t.Fatal(err)
	}

	err = ws.UnsubscribeChannels(nil)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	// Generate test sub
	subs, err := ws.GenerateSubs()
	if err != nil {
		t.Fatal(err)
	}

	// unsub when no subscribed channel
	err = ws.UnsubscribeChannels(subs)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = ws.SubscribeToChannels(subs)
	if err != nil {
		t.Fatal(err)
	}

	// subscribe when already subscribed
	err = ws.SubscribeToChannels(subs)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = ws.UnsubscribeChannels(subs)
	if err != nil {
		t.Fatal(err)
	}
}

func TestResubscribe(t *testing.T) {
	ws := *New()
	err := ws.Setup(defaultSetup)
	if err != nil {
		t.Fatal(err)
	}

	channel := []ChannelSubscription{{Channel: "resubTest"}}
	err = ws.ResubscribeToChannel(&channel[0])
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = ws.SubscribeToChannels(channel)
	if err != nil {
		t.Fatal(err)
	}

	err = ws.ResubscribeToChannel(&channel[0])
	if err != nil {
		t.Fatal("error cannot be nil")
	}
}

// TestConnectionMonitorNoConnection logic test
func TestConnectionMonitorNoConnection(t *testing.T) {
	ws := *New()
	ws.DataHandler = make(chan interface{}, 1)
	ws.ShutdownC = make(chan struct{}, 1)
	ws.exchangeName = "hello"
	ws.trafficTimeout = 1
	go ws.connectionMonitor()
	time.Sleep(time.Second)
	if ws.IsConnectionMonitorRunning() {
		t.Fatal("Should have exited")
	}
}

// TestSliceCopyDoesntImpactBoth logic test
func TestGetSubscriptions(t *testing.T) {
	w := Websocket{
		subscriptions: []ChannelSubscription{
			{
				Channel: "hello3",
			},
		},
	}
	if !strings.EqualFold("hello3", w.GetSubscriptions()[0].Channel) {
		t.Error("Subscriptions was not copied properly")
	}
}

// TestSetCanUseAuthenticatedEndpoints logic test
func TestSetCanUseAuthenticatedEndpoints(t *testing.T) {
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

const (
	websocketTestURL  = "wss://www.bitmex.com/realtime"
	returnResponseURL = "wss://ws.kraken.com"
	useProxyTests     = false                     // Disabled by default. Freely available proxy servers that work all the time are difficult to find
	proxyURL          = "http://212.186.171.4:80" // Replace with a usable proxy server
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

// TestDial logic test
func TestDial(t *testing.T) {
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
	wc := &WebsocketConnection{
		URL:              "wss://echo.websocket.org",
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

// readMessages helper func
func readMessages(wc *WebsocketConnection, t *testing.T) {
	timer := time.NewTimer(20 * time.Second)
	for {
		select {
		case <-timer.C:
			return
		default:
			resp := wc.ReadMessage()
			if resp.Raw != nil {
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
	wc := &WebsocketConnection{
		URL:              "wss://echo.websocket.org",
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
	wc.SetupPingHandler(PingHandler{
		MessageType: websocket.TextMessage,
		Message:     []byte(Ping),
		Delay:       200,
	})
	time.Sleep(time.Millisecond * 500)
	close(wc.ShutdownC)
	wc.Wg.Wait()
}

// TestParseBinaryResponse logic test
func TestParseBinaryResponse(t *testing.T) {
	wc := &WebsocketConnection{
		URL:              "wss://echo.websocket.org",
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
		t.Errorf("GZip conversion failed. Received: '%v', Expected: 'hello'", string(resp2))
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

func TestGenerateMessageID(t *testing.T) {
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
