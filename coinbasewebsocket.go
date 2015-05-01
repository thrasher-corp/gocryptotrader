package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

const (
	COINBASE_WEBSOCKET_URL = "wss://ws-feed.exchange.coinbase.com"
)

type CoinbaseWebsocketSubscribe struct {
	Type      string `json:"type"`
	ProductID string `json:"product_id"`
}

type CoinbaseWebsocketReceived struct {
	Type     string  `json:"type"`
	Time     string  `json:"time"`
	Sequence int     `json:"sequence"`
	OrderID  string  `json:"order_id"`
	Size     float64 `json:"size,string"`
	Price    float64 `json:"price,string"`
	Side     string  `json:"side"`
}

type CoinbaseWebsocketOpen struct {
	Type          string  `json:"type"`
	Time          string  `json:"time"`
	Sequence      int     `json:"sequence"`
	OrderID       string  `json:"order_id"`
	Price         float64 `json:"price,string"`
	RemainingSize float64 `json:"remaining_size,string"`
	Side          string  `json:"side"`
}

type CoinbaseWebsocketDone struct {
	Type          string  `json:"type"`
	Time          string  `json:"time"`
	Sequence      int     `json:"sequence"`
	Price         float64 `json:"price,string"`
	OrderID       string  `json:"order_id"`
	Reason        string  `json:"reason"`
	Side          string  `json:"side"`
	RemainingSize float64 `json:"remaining_size,string"`
}

type CoinbaseWebsocketMatch struct {
	Type         string  `json:"type"`
	TradeID      int     `json:"trade_id"`
	Sequence     int     `json:"sequence"`
	MakerOrderID string  `json:"maker_order_id"`
	TakerOrderID string  `json:"taker_order_id"`
	Time         string  `json:"time"`
	Size         float64 `json:"size,string"`
	Price        float64 `json:"price,string"`
	Side         string  `json:"side"`
}

type CoinbaseWebsocketChange struct {
	Type     string  `json:"type"`
	Time     string  `json:"time"`
	Sequence int     `json:"sequence"`
	OrderID  string  `json:"order_id"`
	NewSize  float64 `json:"new_size,string"`
	OldSize  float64 `json:"old_size,string"`
	Price    float64 `json:"price,string"`
	Side     string  `json:"side"`
}

func (c *Coinbase) WebsocketSubscribe(product string, conn *websocket.Conn) error {
	subscribe := CoinbaseWebsocketSubscribe{"subscribe", "BTC-USD"}
	json, err := JSONEncode(subscribe)
	if err != nil {
		return err
	}

	err = conn.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		return err
	}
	return nil
}

func (c *Coinbase) WebsocketClient() {
	for c.Enabled && c.Websocket {
		var Dialer websocket.Dialer
		conn, _, err := Dialer.Dial(COINBASE_WEBSOCKET_URL, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", c.GetName(), err)
			continue
		}

		log.Printf("%s Connected to Websocket.\n", c.GetName())

		err = c.WebsocketSubscribe("BTC-USD", conn)
		if err != nil {
			log.Printf("%s Websocket subscription error: %s\n", c.GetName(), err)
			continue
		}

		if c.Verbose {
			log.Printf("%s Subscribed to product messages.", c.GetName())
		}

		for c.Enabled && c.Websocket {
			msgType, resp, err := conn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}

			switch msgType {
			case websocket.TextMessage:
				type MsgType struct {
					Type string `json:"type"`
				}

				msgType := MsgType{}
				err := JSONDecode(resp, &msgType)
				if err != nil {
					log.Println(err)
					continue
				}

				switch msgType.Type {
				case "error":
					log.Println(string(resp))
					break
				case "received":
					received := CoinbaseWebsocketReceived{}
					err := JSONDecode(resp, &received)
					if err != nil {
						log.Println(err)
						continue
					}
				case "open":
					open := CoinbaseWebsocketOpen{}
					err := JSONDecode(resp, &open)
					if err != nil {
						log.Println(err)
						continue
					}
				case "done":
					done := CoinbaseWebsocketDone{}
					err := JSONDecode(resp, &done)
					if err != nil {
						log.Println(err)
						continue
					}
				case "match":
					match := CoinbaseWebsocketMatch{}
					err := JSONDecode(resp, &match)
					if err != nil {
						log.Println(err)
						continue
					}
				case "change":
					change := CoinbaseWebsocketChange{}
					err := JSONDecode(resp, &change)
					if err != nil {
						log.Println(err)
						continue
					}
				}
			}
		}
		conn.Close()
		log.Printf("%s Websocket client disconnected.", c.GetName())
	}
}
