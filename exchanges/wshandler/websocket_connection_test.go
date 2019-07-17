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
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
)

const (
	websocketTestURL  = "wss://www.bitmex.com/realtime"
	returnResponseURL = "wss://ws.kraken.com"
	useProxyTests     = false                     // Disabled by default. Freely available proxy servers that work all the time are difficult to find
	proxyURL          = "http://212.186.171.4:80" // Replace with a usable proxy server
)

type TestStruct struct {
	Error error
	WC    WebsocketConnection
}

var wc *WebsocketConnection
var dialer websocket.Dialer

type WebsocketSubscriptionEventRequest struct {
	Event        string                    `json:"event"`
	RequestID    int64                     `json:"reqid,omitempty"`
	Pairs        []string                  `json:"pair"`
	Subscription WebsocketSubscriptionData `json:"subscription,omitempty"`
}

// WebsocketSubscriptionData contains details on WS channel
type WebsocketSubscriptionData struct {
	Name     string `json:"name,omitempty"`
	Interval int64  `json:"interval,omitempty"`
	Depth    int64  `json:"depth,omitempty"`
}

type TestWebsocketResponse struct {
	RequestID int64 `json:"reqid,omitempty"`
}

func TestMain(m *testing.M) {
	wc = &WebsocketConnection{
		ExchangeName: "test",
		Verbose:      true,
		URL:          returnResponseURL,
	}
	os.Exit(m.Run())
}

func TestDial(t *testing.T) {
	var testCases = []TestStruct{
		{Error: nil, WC: WebsocketConnection{ExchangeName: "test1", Verbose: true, URL: websocketTestURL, RateLimit: 10}},
		{Error: errors.New(" Error: malformed ws or wss URL"), WC: WebsocketConnection{ExchangeName: "test2", Verbose: true, URL: ""}},
		{Error: nil, WC: WebsocketConnection{ExchangeName: "test3", Verbose: true, URL: websocketTestURL, ProxyURL: proxyURL}},
	}
	for i := range testCases {
		t.Run(testCases[i].WC.ExchangeName, func(t *testing.T) {
			if testCases[i].WC.ProxyURL != "" && !useProxyTests {
				t.Skip("Proxy testing not enabled, skipping")
			}
			err := testCases[i].WC.Dial(&dialer, http.Header{})
			if err != nil {
				if testCases[i].Error != nil && err.Error() == testCases[i].Error.Error() {
					return
				}
				t.Fatal(err)
			}
		})
	}
}

func TestSendMessage(t *testing.T) {
	var testCases = []TestStruct{
		{Error: nil, WC: WebsocketConnection{ExchangeName: "test1", Verbose: true, URL: websocketTestURL, RateLimit: 10}},
		{Error: errors.New(" Error: malformed ws or wss URL"), WC: WebsocketConnection{ExchangeName: "test2", Verbose: true, URL: ""}},
		{Error: nil, WC: WebsocketConnection{ExchangeName: "test3", Verbose: true, URL: websocketTestURL, ProxyURL: proxyURL}},
	}
	for i := range testCases {
		t.Run(testCases[i].WC.ExchangeName, func(t *testing.T) {
			if testCases[i].WC.ProxyURL != "" && !useProxyTests {
				t.Skip("Proxy testing not enabled, skipping")
			}
			err := testCases[i].WC.Dial(&dialer, http.Header{})
			if err != nil {
				if testCases[i].Error != nil && err.Error() == testCases[i].Error.Error() {
					return
				}
				t.Fatal(err)
			}
			err = testCases[i].WC.SendMessage("ping")
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestSendMessageWithResponse(t *testing.T) {
	if wc.ProxyURL != "" && !useProxyTests {
		t.Skip("Proxy testing not enabled, skipping")
	}
	err := wc.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go readMesages(wc, t)

	request := WebsocketSubscriptionEventRequest{
		Event: "subscribe",
		Pairs: []string{currency.NewPairWithDelimiter("XBT", "USD", "/").String()},
		Subscription: WebsocketSubscriptionData{
			Name: "ticker",
		},
		RequestID: wc.GenerateMessageID(true),
	}
	_, err = wc.SendMessageReturnResponse(request.RequestID, request)
	if err != nil {
		t.Error(err)
	}
}

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
	w2, err := flate.NewWriter(&b2, 1)
	w2.Write([]byte("hello"))
	w2.Close()
	resp, err = wc.parseBinaryResponse(b2.Bytes())
	if err != nil {
		t.Error(err)
	}
	if !strings.EqualFold(string(resp), "hello") {
		t.Errorf("GZip conversion failed. Received: '%v', Expected: 'hello'", string(resp))
	}
}

func TestAddResponseWithID(t *testing.T) {
	wc.IDResponses = nil
	wc.AddResponseWithID(0, []byte("hi"))
	wc.AddResponseWithID(1, []byte("hi"))
}

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
			var incoming TestWebsocketResponse
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
