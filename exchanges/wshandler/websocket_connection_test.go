package wshandler

import (
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/gorilla/websocket"
)

const (
	websocketTestURL = "wss://www.bitmex.com/realtime"
	proxyURL         = "http://103.89.253.249:3128"
)

type TestStruct struct {
	Error error
	WC    WebsocketConnection
}

var wc *WebsocketConnection
var testCases = []TestStruct{
	{Error: nil, WC: WebsocketConnection{ExchangeName: "test1", Verbose: true, URL: websocketTestURL, RateLimit: 10}},
	{Error: errors.New(" Error: malformed ws or wss URL"), WC: WebsocketConnection{ExchangeName: "test2", Verbose: true, URL: ""}},
	{Error: nil, WC: WebsocketConnection{ExchangeName: "test3", Verbose: true, URL: websocketTestURL, ProxyURL: proxyURL}},
}
var dialer websocket.Dialer

func TestMain(m *testing.M) {
	wc = &WebsocketConnection{
		ExchangeName: "butts",
		Verbose:      true,
		URL:          "wss://echo.websocket.org",
	}
	os.Exit(m.Run())
}

func TestDial(t *testing.T) {
	for _, tests := range testCases {
		test := tests
		t.Run(test.WC.ExchangeName, func(t *testing.T) {
			err := test.WC.Dial(&dialer, http.Header{})
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestSendMessage(t *testing.T) {
	for _, tests := range testCases {
		test := tests
		t.Run(test.WC.ExchangeName, func(t *testing.T) {
			err := test.WC.Dial(&dialer, http.Header{})
			if err != nil {
				if err.Error() == test.Error.Error() {
					return
				}
				t.Fatal(err)
			}
			err = test.WC.SendMessage("ping")
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestSendMessageWithResponse(t *testing.T) {
	for _, tests := range testCases {
		test := tests
		t.Run(test.WC.ExchangeName, func(t *testing.T) {
			err := test.WC.Dial(&dialer, http.Header{})
			if err != nil {
				if err != test.Error {
					t.Fatal(err)
				}
			}
			resp, err := test.WC.SendMessageReturnResponse(1, "ping")
			if err != nil {
				t.Error(err)
			}
			t.Log(string(resp))
		})
	}
}
