package mexc

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	wsURL = "wss://wbs-api.mexc.com/ws"
)

// WsConnect initiates a websocket connection
func (me *MEXC) WsConnect() error {
	me.Websocket.Enable()
	if !me.Websocket.IsEnabled() || !me.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer = websocket.Dialer{
		EnableCompression: true,
		ReadBufferSize:    8192,
		WriteBufferSize:   8192,
	}

	err := me.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	me.Websocket.Wg.Add(1)
	go me.wsReadData(me.Websocket.Conn)
	if me.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			me.Websocket.GetWebsocketURL())
	}
	me.Websocket.Conn.SetupPingHandler(request.Unset, stream.PingHandler{
		MessageType: websocket.TextMessage,
		Message:     []byte(`{"method": "PING"}`),
		Delay:       time.Second * 20,
	})
	if me.Websocket.CanUseAuthenticatedEndpoints() {
		// err = me.WsAuth(context.TODO())
		// if err != nil {
		// 	log.Errorf(log.ExchangeSys, "Error connecting auth socket: %s\n", err.Error())
		// 	me.Websocket.SetCanUseAuthenticatedEndpoints(false)
		// }
	}
	return me.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, &WsSubscriptionPayload{Method: "SUBSCRIPTION", Params: []string{"spot@public.aggre.bookTicker.v3.api.pb@100ms@BTCUSDT"}})
}

// wsReadData sends msgs from public and auth websockets to data handler
func (me *MEXC) wsReadData(ws stream.Connection) {
	defer me.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		if err := me.WsHandleData(resp.Raw); err != nil {
			panic(err.Error())
			me.Websocket.DataHandler <- err
		}
	}
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (me *MEXC) generateSubscriptions() (subscription.List, error) {
	return subscription.List{}, nil
	// return me.Features.Subscriptions.ExpandTemplates(me)
}

func (me *MEXC) Subscribe(channelsToSubscribe subscription.List) error {
	return me.handleSubscription("SUBSCRIPTION", channelsToSubscribe)
}

func (me *MEXC) Unsubscribe(channelsToSubscribe subscription.List) error {
	return me.handleSubscription("UNSUBSCRIPTION", channelsToSubscribe)
}

func (me *MEXC) handleSubscription(method string, subs subscription.List) error {
	var payload []WsSubscriptionPayload
	data, err := me.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, "abc", payload)
	if err != nil {
		return err
	}
	println(string(data))
	return nil
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (me *MEXC) WsHandleData(respRaw []byte) error {
	println(string(respRaw))
	return nil
}
