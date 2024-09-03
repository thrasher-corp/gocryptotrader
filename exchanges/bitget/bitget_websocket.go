package bitget

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bitgetPublicWSURL  = "wss://ws.bitget.com/v2/ws/public/"
	bitgetPrivateWSURL = "wss://ws.bitget.com/v2/ws/private/"
)

// WsConnect connects to a websocket feed
func (bi *Bitget) WsConnect() error {
	if !bi.Websocket.IsEnabled() || !bi.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	err := bi.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if bi.Verbose {
		log.Debugf(log.ExchangeSys, "%s connected to Websocket.\n", bi.Name)
	}
	bi.Websocket.Wg.Add(1)
	go bi.wsReadData()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (bi *Bitget) wsReadData() {
	defer bi.Websocket.Wg.Done()
	for {
		resp := bi.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := bi.wsHandleData(resp.Raw)
		if err != nil {
			bi.Websocket.DataHandler <- err
		}
	}
}

func (bi *Bitget) wsHandleData(respRaw []byte) error {
	var wsResponse WsResponse
	err := json.Unmarshal(respRaw, &wsResponse)
	if err != nil {
		return err
	}
	switch wsResponse.Event {
	case "pong":
		if bi.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket %v\n", bi.Name, wsResponse.Event)
		}
	case "subscribe":
		if bi.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket %v succeeded for %v\n", bi.Name, wsResponse.Event,
				wsResponse.Arg)
		}
	case "error":
		return fmt.Errorf("%v - Websocket error, code: %v message: %v", bi.Name, wsResponse.Code, wsResponse.Message)
	default:
		bi.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: bi.Name + stream.UnhandledMessage +
			string(respRaw)}
	}
	return nil
}

func (bi *Bitget) generateDefaultSubscriptions() (subscription.List, error) {
	channels := []string{bitgetTickerChannel}
	enabledPairs, err := bi.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions subscription.List
	for i := range channels {
		subscriptions = append(subscriptions, &subscription.Subscription{
			Channel: channels[i],
			Pairs:   enabledPairs,
			Asset:   asset.Spot,
		})
	}
	return subscriptions, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (bi *Bitget) Subscribe(subs subscription.List) error {
	baseReq := &WsRequest{
		Operation: "subscribe",
	}
	for _, s := range subs {
		for i := range s.Pairs {
			baseReq.Arguments = append(baseReq.Arguments, WsArgument{
				Channel:        s.Channel,
				InstrumentType: s.Asset.String(),
				InstrumentID:   s.Pairs[i].String(),
			})
		}
	}
	cap := (len(baseReq.Arguments) / 47)
	reqSlice := make([]WsRequest, cap)
	for i := 0; i < cap; i++ {
		reqSlice[i].Operation = baseReq.Operation
		if i == cap-1 {
			reqSlice[i].Arguments = baseReq.Arguments[i*47:]
			break
		}
		reqSlice[i].Arguments = baseReq.Arguments[i*47 : (i+1)*47]
	}
	for i := range reqSlice {
		err := bi.Websocket.Conn.SendJSONMessage(reqSlice[i])
		if err != nil {
		}
	}
	return common.ErrNotYetImplemented
}

// SendJSONMessage sends a JSON message to the connected websocket
