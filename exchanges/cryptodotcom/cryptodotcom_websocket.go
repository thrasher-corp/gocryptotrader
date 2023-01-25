package cryptodotcom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	publicHeartbeat         = "public/heartbeat"
	publicResponsdHeartbeat = "public/respond-heartbeat"
)

var websocketSubscriptionEndpointsURL = []string{
	publicAuth,
	publicInstruments,

	privateSetCancelOnDisconnect,
	privateGetCancelOnDisconnect,

	postWithdrawal,
	privateGetWithdrawalHistory,
	privateGetAccountSummary,

	privateCreateOrder,
	privateCancelOrder,
	privateCreateOrderList,
	privateCancelOrderList,
	privateCancelAllOrders,
	privateGetOrderHistory,
	privateGetOpenOrders,
	privateGetOrderDetail,
	privateGetTrades,
}

// websocket subscriptions channels list

const (
	// private subscription channels
	userOrderCnl   = "user.order.%s" // user.order.{instrument_name}
	userTradeCnl   = "user.trade.%s" // user.trade.{instrument_name}
	userBalanceCnl = "user.balance"

	// public subscription channels

	instrumentOrderbookCnl = "book.%s"           // book.{instrument_name}
	tickerCnl              = "ticker.%s"         // ticker.{instrument_name}
	tradeCnl               = "trade.%s"          // trade.{instrument_name}
	candlestickCnl         = "candlestick.%s.%s" // candlestick.{time_frame}.{instrument_name}
)

var defaultSubscriptions = []string{
	instrumentOrderbookCnl,
	tickerCnl,
	tradeCnl,
	candlestickCnl,
}

// responseStream a channel thought which the data coming from the two websocket connection will go through.
var responseStream = make(chan stream.Response)

func (cr *Cryptodotcom) WsConnect() error {
	if !cr.Websocket.IsEnabled() || !cr.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192
	err := cr.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	cr.Websocket.Wg.Add(2)
	go cr.wsFunnelConnectionData(cr.Websocket.Conn)
	go cr.WsReadData()
	if cr.Verbose {
		log.Debugf(log.ExchangeSys, "Successful connection to %v\n",
			cr.Websocket.GetWebsocketURL())
	}
	cr.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PingMessage,
		Delay:             time.Second * 10,
	})
	if cr.IsWebsocketAuthenticationSupported() {
		var authDialer websocket.Dialer
		authDialer.ReadBufferSize = 8192
		authDialer.WriteBufferSize = 8192
		err = cr.WsAuthConnect(context.TODO(), authDialer)
		if err != nil {
			cr.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	subscriptions, _ := cr.GenerateDefaultSubscriptions()
	return cr.Subscribe(subscriptions)
}

// wsFunnelConnectionData receives data from multiple connection and pass the data
// to wsRead through a channel responseStream
func (cr *Cryptodotcom) wsFunnelConnectionData(ws stream.Connection) {
	defer cr.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		responseStream <- stream.Response{Raw: resp.Raw}
	}
}

// WsReadData read coming messages thought the websocket connection and process the data.
func (cr *Cryptodotcom) WsReadData() {
	defer cr.Websocket.Wg.Done()
	for {
		select {
		case <-cr.Websocket.ShutdownC:
			select {
			case resp := <-responseStream:
				err := cr.WsHandleData(resp.Raw)
				if err != nil {
					select {
					case cr.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr, "%s websocket handle data error: %v", cr.Name, err)
					}
				}
			default:
			}
			return
		case resp := <-responseStream:
			err := cr.WsHandleData(resp.Raw)
			if err != nil {
				cr.Websocket.DataHandler <- err
			}
		}
	}
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (cr *Cryptodotcom) WsHandleData(respRaw []byte) error {
	println(string(respRaw))
	var resp *SubscriptionResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	if resp.ID != 0 {
		cr.Websocket.Match.IncomingWithData(resp.ID, respRaw)
	}
	return nil
}

// WsAuthConnect represents an authenticated connection to a websocket server
func (cr *Cryptodotcom) WsAuthConnect(ctx context.Context, dialer websocket.Dialer) error {
	if !cr.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%v AuthenticatedWebsocketAPISupport not enabled", cr.Name)
	}
	err := cr.Websocket.AuthConn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v Websocket connection %v error. Error %v", cr.Name, cryptodotcomWebsocketUserAPI, err)
	}
	return nil
}

// WsAuthenticate sends authentication request through the websocket connection.
func (cr *Cryptodotcom) WsAuthenticate(ctx context.Context) error {
	return nil
}

// func (cr)

func (cr *Cryptodotcom) Subscribe(subscriptions []stream.ChannelSubscription) error {
	return cr.handleSubscriptions("subscribe", subscriptions)
}

func (cr *Cryptodotcom) Unsubscribe(subscriptions []stream.ChannelSubscription) error {
	return cr.handleSubscriptions("unsubscribe", subscriptions)
}

// GenerateDefaultSubscriptions ...
func (cr *Cryptodotcom) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	subscriptions := []stream.ChannelSubscription{}
	channels := defaultSubscriptions
	for x := range channels {
		if channels[x] == userBalanceCnl {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel: channels[x],
			})
			continue
		}
		enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}
		for p := range enabledPairs {
			switch channels[x] {
			case instrumentOrderbookCnl,
				tickerCnl,
				tradeCnl:
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[x],
					Currency: enabledPairs[p],
				})
			case candlestickCnl:
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Channel:  channels[x],
					Currency: enabledPairs[p],
					Params: map[string]interface{}{
						"interval": "5m",
					},
				})
			default:
				continue
			}
		}
	}
	return subscriptions, nil
}

func (cr *Cryptodotcom) handleSubscriptions(operation string, subscriptions []stream.ChannelSubscription) error {
	subscriptionPayloads, err := cr.generatePayload(operation, subscriptions)
	if err != nil {
		return err
	}
	for p := range subscriptionPayloads {
		val, _ := json.Marshal(subscriptionPayloads[p])
		println("Payload: ", string(val))
		err := cr.Websocket.Conn.SendJSONMessage(subscriptionPayloads[p])
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
	return nil
}

func (cr *Cryptodotcom) generatePayload(operation string, subscription []stream.ChannelSubscription) ([]SubscriptionPayload, error) {
	subscriptionPayloads := make([]SubscriptionPayload, len(subscription))
	timestamp := time.Now()
	for x := range subscription {
		subscriptionPayloads[x] = SubscriptionPayload{
			ID:     int(cr.Websocket.Conn.GenerateMessageID(false)),
			Method: operation,
			Nonce:  timestamp.UnixMilli(),
		}
		switch subscription[x].Channel {
		case userOrderCnl,
			userTradeCnl,
			instrumentOrderbookCnl,
			tickerCnl,
			tradeCnl:
			subscriptionPayloads[x].Params = map[string][]string{"channels": {fmt.Sprintf(subscription[x].Channel, subscription[x].Currency.String())}}
		case candlestickCnl:
			subscriptionPayloads[x].Params = map[string][]string{"channels": {fmt.Sprintf(subscription[x].Channel, subscription[x].Params["interval"].(string), subscription[x].Currency.String())}}
		case userBalanceCnl:
			subscriptionPayloads[x].Params = map[string][]string{"channels": {subscription[x].Channel}}
		}
	}
	return subscriptionPayloads, nil
}
