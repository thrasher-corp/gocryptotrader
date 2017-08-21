package bitfinex

import (
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
)

func TestWebsocketPingHandler(t *testing.T) {
	wsPingHandler := Bitfinex{}
	var Dialer websocket.Dialer
	var err error

	wsPingHandler.WebsocketConn, _, err = Dialer.Dial(bitfinexWebsocket, http.Header{})
	if err != nil {
		t.Errorf("Test Failed - Bitfinex dialer error: %s", err)
	}
	err = wsPingHandler.WebsocketPingHandler()
	if err != nil {
		t.Errorf("Test Failed - Bitfinex WebsocketPingHandler() error: %s", err)
	}
	err = wsPingHandler.WebsocketConn.Close()
	if err != nil {
		t.Errorf("Test Failed - Bitfinex websocketConn.Close() error: %s", err)
	}
}

func TestWebsocketSubscribe(t *testing.T) {
	websocketSubcribe := Bitfinex{}
	var Dialer websocket.Dialer
	var err error
	params := make(map[string]string)
	params["pair"] = "BTCUSD"

	websocketSubcribe.WebsocketConn, _, err = Dialer.Dial(bitfinexWebsocket, http.Header{})
	if err != nil {
		t.Errorf("Test Failed - Bitfinex Dialer error: %s", err)
	}
	err = websocketSubcribe.WebsocketSubscribe("ticker", params)
	if err != nil {
		t.Errorf("Test Failed - Bitfinex WebsocketSubscribe() error: %s", err)
	}

	err = websocketSubcribe.WebsocketConn.Close()
	if err != nil {
		t.Errorf("Test Failed - Bitfinex websocketConn.Close() error: %s", err)
	}
}

func TestWebsocketSendAuth(t *testing.T) {
	wsSendAuth := Bitfinex{}
	var Dialer websocket.Dialer
	var err error

	wsSendAuth.WebsocketConn, _, err = Dialer.Dial(bitfinexWebsocket, http.Header{})
	if err != nil {
		t.Errorf("Test Failed - Bitfinex Dialer error: %s", err)
	}
	err = wsSendAuth.WebsocketSendAuth()
	if err != nil {
		t.Errorf("Test Failed - Bitfinex WebsocketSendAuth() error: %s", err)
	}
}

func TestWebsocketAddSubscriptionChannel(t *testing.T) {
	wsAddSubscriptionChannel := Bitfinex{}
	wsAddSubscriptionChannel.SetDefaults()
	var Dialer websocket.Dialer
	var err error

	wsAddSubscriptionChannel.WebsocketConn, _, err = Dialer.Dial(bitfinexWebsocket, http.Header{})
	if err != nil {
		t.Errorf("Test Failed - Bitfinex Dialer error: %s", err)
	}

	wsAddSubscriptionChannel.WebsocketAddSubscriptionChannel(1337, "ticker", "BTCUSD")
	if len(wsAddSubscriptionChannel.WebsocketSubdChannels) == 0 {
		t.Errorf("Test Failed - Bitfinex WebsocketAddSubscriptionChannel() error: %s", err)
	}
	if wsAddSubscriptionChannel.WebsocketSubdChannels[1337].Channel != "ticker" {
		t.Errorf("Test Failed - Bitfinex WebsocketAddSubscriptionChannel() error: %s", err)
	}
	if wsAddSubscriptionChannel.WebsocketSubdChannels[1337].Pair != "BTCUSD" {
		t.Errorf("Test Failed - Bitfinex WebsocketAddSubscriptionChannel() error: %s", err)
	}
}
