package wshandler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

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
		Verbose:              true,
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
	for i := 0; i < len(testCases); i++ {
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
	for i := 0; i < len(testCases); i++ {
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
			err = testData.WC.SendMessage("ping")
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
	go readMesages(wc, t)

	request := testRequest{
		Event: "subscribe",
		Pairs: []string{currency.NewPairWithDelimiter("XBT", "USD", "/").String()},
		Subscription: testRequestData{
			Name: "ticker",
		},
		RequestID: wc.GenerateMessageID(true),
	}
	_, err = wc.SendMessageReturnResponse(request.RequestID, request)
	if err != nil {
		t.Error(err)
	}
}

// TestParseBinaryResponse logic test
func TestParseBinaryResponse(t *testing.T) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write([]byte("hello"))
	w.Close()
	resp, err := wc.parseBinaryResponse(b.Bytes())
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
	w2.Write([]byte("hello"))
	w2.Close()
	resp2, err3 := wc.parseBinaryResponse(b2.Bytes())
	if err3 != nil {
		t.Error(err3)
	}
	if !strings.EqualFold(string(resp2), "hello") {
		t.Errorf("GZip conversion failed. Received: '%v', Expected: 'hello'", string(resp2))
	}
}

// TestAddResponseWithID logic test
func TestAddResponseWithID(t *testing.T) {
	wc.IDResponses = nil
	wc.AddResponseWithID(0, []byte("hi"))
	wc.AddResponseWithID(1, []byte("hi"))
}

// readMesages helper func
func readMesages(wc *WebsocketConnection, t *testing.T) {
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
			err = common.JSONDecode(resp.Raw, &incoming)
			if err != nil {
				t.Error(err)
				return
			}
			if incoming.RequestID > 0 {
				wc.AddResponseWithID(incoming.RequestID, resp.Raw)
				return
			}
		}
	}
}
