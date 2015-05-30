package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

/* Implementation based off Bitfinex's bfws.bitfinex.com websocket service */

const (
	BITFINEX_WEBSOCKET           = "ws://websocket.bitfinex.com:8086/WSGateway/"
	BTIFINEX_WEBSOCKET_TRADES    = "SubscribeTrades"
	BITFINEX_WEBSOCKET_ORDERBOOK = "SubscribeLevel2"
)

type BitfinexWebsocketFrameRequest struct {
	Type         string `json:"type"`
	M            int    `json:"m"`
	I            int    `json:"i"`
	Subscription string `json:"n"`
	Params       string `json:"o"`
}

type BitfinexWebsocketResponse struct {
	M            int    `json:"m"`
	I            int    `json:"i"`
	Subscription string `json:"n"`
	Params       string `json:"o"`
}

func (b *Bitfinex) Subscribe(counter *int, subscription string, params map[string]interface{}) error {
	msg := BitfinexWebsocketFrameRequest{}
	msg.Type = "anws_frame"
	msg.M = 0
	msg.I = *counter
	msg.Subscription = subscription

	paramsEncoded, err := JSONEncode(params)
	if err != nil {
		return err
	}
	msg.Params = string(paramsEncoded)

	msgEncoded, err := JSONEncode(msg)
	if err != nil {
		return err
	}

	err = b.WebsocketConn.WriteMessage(websocket.TextMessage, msgEncoded)

	if err != nil {
		return err
	}

	*counter += 2
	return nil
}

func (b *Bitfinex) WebsocketClient() {
	msgCounter := 0
	for b.Enabled && b.Websocket {
		var Dialer websocket.Dialer
		var err error
		b.WebsocketConn, _, err = Dialer.Dial(BITFINEX_WEBSOCKET, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.GetName(), err)
			continue
		}

		if b.Verbose {
			log.Printf("%s Connected to Websocket.\n", b.GetName())
		}

		err = b.Subscribe(&msgCounter, BITFINEX_WEBSOCKET_ORDERBOOK, map[string]interface{}{"ExchangeID": "0", "ProductPairID": "1", "Depth": 25, "RoundToDecimals": 2})
		if err != nil {
			log.Println(err)
			continue
		}

		err = b.Subscribe(&msgCounter, BTIFINEX_WEBSOCKET_TRADES, map[string]interface{}{"ExchangeID": "0", "ProductPairID": "1", "IncludeLastCount": "100"})
		if err != nil {
			log.Println(err)
			continue
		}

		if b.Verbose {
			log.Printf("%s Subscribed to channels.\n", b.GetName())
		}

		for b.Enabled && b.Websocket {
			msgType, resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}

			switch msgType {
			case websocket.TextMessage:
				msg := BitfinexWebsocketResponse{}
				err := JSONDecode(resp, &msg)
				if err != nil {
					log.Println(err)
					continue
				}
				log.Println(string(resp))
				log.Println(msg)
			}
		}
		b.WebsocketConn.Close()
		log.Printf("%s Websocket client disconnected.\n", b.GetName())
	}
}
