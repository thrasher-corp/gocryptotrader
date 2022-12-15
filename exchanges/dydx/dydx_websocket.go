package dydx

import (
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (

	// channels

	accountsChannel  = "v3_accounts"
	orderbookChannel = "v3_orderbook"
	tradesChannel    = "v3_trades"
	marketsChannel   = "v3_markets"
)

// WsConnect connect to dydx websocket server.
func (dy *DYDX) WsConnect() error {
	if !dy.Websocket.IsEnabled() || !dy.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := dy.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if dy.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", dy.Name)
	}
	dy.Websocket.Wg.Add(1)
	go dy.wsReadData()
	return nil
}

func (dy *DYDX) wsReadData() {
	defer dy.Websocket.Wg.Done()
	for {
		resp := dy.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := dy.wsHandleData(resp.Raw)
		if err != nil {
			dy.Websocket.DataHandler <- err
		}
	}
}

func (dy *DYDX) wsHandleData(respRaw []byte) error {
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (dy *DYDX) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {

	return nil, nil
}

// Subscribe sends a subscriptions requests through the websocket connection.
func (dy *DYDX) Subscribe(subscriptions []stream.ChannelSubscription) error {
	return nil
}

// Unsubscribe sends unsubscription to channels through the websocket connection.
func (dy *DYDX) Unsubscribe(subscriptions []stream.ChannelSubscription) error {
	return nil
}
