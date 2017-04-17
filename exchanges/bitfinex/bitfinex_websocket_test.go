package bitfinex

import (
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

func TestWebsocketPingHandler(t *testing.T) {
	wsPingHandler := Bitfinex{}
	var Dialer websocket.Dialer
	var err error

	wsPingHandler.WebsocketConn, _, err = Dialer.Dial(BITFINEX_WEBSOCKET, http.Header{})
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

func TestWebsocketSend(t *testing.T) {
	wsSend := Bitfinex{}
	var Dialer websocket.Dialer
	var err error

	type WebsocketHandshake struct {
		Event   string  `json:"event"`
		Code    int64   `json:"code"`
		Version float64 `json:"version"`
	}

	request, dodgyrequest := make(map[string]string), make(map[string]string)
	request["event"] = "ping"
	dodgyrequest["dodgyEvent"] = "didgereedodge"

	hs := WebsocketHandshake{}

	for {
		wsSend.WebsocketConn, _, err = Dialer.Dial(BITFINEX_WEBSOCKET, http.Header{})
		if err != nil {
			if err.Error() == "websocket: close 1006 (abnormal closure): unexpected EOF" {
				err = wsSend.WebsocketConn.Close()
				if err != nil {
					t.Errorf("Test Failed - Bitfinex websocketConn.Close() error: %s", err)
				}
				continue
			} else {
				t.Errorf("Test Failed - Bitfinex websocket connection error: %s", err)
			}
		}
		mType, resp, err := wsSend.WebsocketConn.ReadMessage()
		if err != nil {
			t.Errorf("Test Failed - Bitfinex websocketconn.ReadMessage() error: %s", err)
		}
		if mType != websocket.TextMessage {
			t.Errorf("Test Failed - Bitfinex websocketconn.ReadMessage() mType error: %d", mType)
		}
		err = common.JSONDecode(resp, &hs)
		if err != nil {
			t.Errorf("Test Failed - Bitfinex JSONDecode error: %s", err)
		}
		if hs.Code != 0 {
			t.Errorf("Test Failed - Bitfinex hs.Code incorrect: %d", hs.Code)
		}
		if hs.Event != "info" {
			t.Errorf("Test Failed - Bitfinex hs.Event incorrect: %s", hs.Event)
		}
		if hs.Version != 1.1 {
			t.Errorf("Test Failed - Bitfinex hs.Version incorrect: %f", hs.Version)
		}

		err = wsSend.WebsocketSend(request)
		if err != nil {
			t.Errorf("Test Failed - Bitfinex websocket send error: %s", err)
		}
		mType, resp, err = wsSend.WebsocketConn.ReadMessage()
		if err != nil {
			if err.Error() == "websocket: close 1006 (abnormal closure): unexpected EOF" {
				err = wsSend.WebsocketConn.Close()
				if err != nil {
					t.Errorf("Test Failed - Bitfinex websocketConn.Close() error: %s", err)
				}
				continue
			} else {
				t.Errorf("Test Failed - Bitfinex websocketConn.ReadMessage() error: %s", err)
			}
		}
		if mType != websocket.TextMessage {
			t.Errorf("Test Failed - Bitfinex websocketconn.ReadMessage() mType error: %d", mType)
		}
		err = common.JSONDecode(resp, &hs)
		if err != nil {
			t.Errorf("Test Failed - Bitfinex JSONDecode error: %s", err)
		}
		if hs.Code != 0 {
			t.Errorf("Test Failed - Bitfinex hs.Code incorrect: %d", hs.Code)
		}
		if hs.Event != "pong" {
			t.Errorf("Test Failed - Bitfinex hs.Event incorrect: %s", hs.Event)
		}
		if hs.Version != 1.1 {
			t.Errorf("Test Failed - Bitfinex hs.Version incorrect: %f", hs.Version)
		}

		err = wsSend.WebsocketSend(dodgyrequest)
		if err != nil {
			t.Errorf("Test Failed - Bitfinex websocket send error: %s", err)
		}
		mType, resp, err = wsSend.WebsocketConn.ReadMessage()
		if err != nil {
			if err.Error() == "websocket: close 1006 (abnormal closure): unexpected EOF" {
				err = wsSend.WebsocketConn.Close()
				if err != nil {
					t.Errorf("Test Failed - Bitfinex websocketConn.Close() error: %s", err)
				}
				continue
			} else {
				t.Errorf("Test Failed - Bitfinex websocketConn.ReadMessage() error: %s", err)
			}
		}
		if mType != websocket.TextMessage {
			t.Errorf("Test Failed - Bitfinex websocketconn.ReadMessage() mType error: %d", mType)
		}
		err = common.JSONDecode(resp, &hs)
		if err != nil {
			t.Errorf("Test Failed - Bitfinex JSONDecode error: %s", err)
		}
		if hs.Code != 10000 {
			t.Errorf("Test Failed - Bitfinex hs.Code incorrect: %d", hs.Code)
		}
		if hs.Event != "error" {
			t.Errorf("Test Failed - Bitfinex hs.Event incorrect: %s", hs.Event)
		}
		if hs.Version != 1.1 {
			t.Errorf("Test Failed - Bitfinex hs.Version incorrect: %f", hs.Version)
		}

		err = wsSend.WebsocketConn.Close()
		if err != nil {
			t.Errorf("Test Failed - Bitfinex websocketConn.Close() error: %s", err)
		}
		break
	}
}

func TestWebsocketSubscribe(t *testing.T) {
	websocketSubcribe := Bitfinex{}
	var Dialer websocket.Dialer
	var err error
	params := make(map[string]string)
	params["pair"] = "BTCUSD"

	websocketSubcribe.WebsocketConn, _, err = Dialer.Dial(BITFINEX_WEBSOCKET, http.Header{})
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

	wsSendAuth.WebsocketConn, _, err = Dialer.Dial(BITFINEX_WEBSOCKET, http.Header{})
	if err != nil {
		t.Errorf("Test Failed - Bitfinex Dialer error: %s", err)
	}
	err = wsSendAuth.WebsocketSendAuth()
	if err != nil {
		t.Errorf("Test Failed - Bitfinex WebsocketSendAuth() error: %s", err)
	}
}

func TestWebsocketSendUnauth(t *testing.T) {
	wsSendUnauth := Bitfinex{}
	var Dialer websocket.Dialer
	var err error

	wsSendUnauth.WebsocketConn, _, err = Dialer.Dial(BITFINEX_WEBSOCKET, http.Header{})
	if err != nil {
		t.Errorf("Test Failed - Bitfinex Dialer error: %s", err)
	}
	err = wsSendUnauth.WebsocketSendUnauth()
	if err != nil {
		t.Errorf("Test Failed - Bitfinex WebsocketSendAuth() error: %s", err)
	}
}

func TestWebsocketAddSubscriptionChannel(t *testing.T) {
	wsAddSubscriptionChannel := Bitfinex{}
	wsAddSubscriptionChannel.SetDefaults()
	var Dialer websocket.Dialer
	var err error

	wsAddSubscriptionChannel.WebsocketConn, _, err = Dialer.Dial(BITFINEX_WEBSOCKET, http.Header{})
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

// func TestWebsocketClient(t *testing.T) {
//
// }
