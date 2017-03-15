package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

const (
	GDAX_WEBSOCKET_URL = "wss://ws-feed.exchange.gdax.com"
)

type GDAXWebsocketSubscribe struct {
	Type      string `json:"type"`
	ProductID string `json:"product_id"`
}

type GDAXWebsocketReceived struct {
	Type     string  `json:"type"`
	Time     string  `json:"time"`
	Sequence int     `json:"sequence"`
	OrderID  string  `json:"order_id"`
	Size     float64 `json:"size,string"`
	Price    float64 `json:"price,string"`
	Side     string  `json:"side"`
}

type GDAXWebsocketOpen struct {
	Type          string  `json:"type"`
	Time          string  `json:"time"`
	Sequence      int     `json:"sequence"`
	OrderID       string  `json:"order_id"`
	Price         float64 `json:"price,string"`
	RemainingSize float64 `json:"remaining_size,string"`
	Side          string  `json:"side"`
}

type GDAXWebsocketDone struct {
	Type          string  `json:"type"`
	Time          string  `json:"time"`
	Sequence      int     `json:"sequence"`
	Price         float64 `json:"price,string"`
	OrderID       string  `json:"order_id"`
	Reason        string  `json:"reason"`
	Side          string  `json:"side"`
	RemainingSize float64 `json:"remaining_size,string"`
}

type GDAXWebsocketMatch struct {
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

type GDAXWebsocketChange struct {
	Type     string  `json:"type"`
	Time     string  `json:"time"`
	Sequence int     `json:"sequence"`
	OrderID  string  `json:"order_id"`
	NewSize  float64 `json:"new_size,string"`
	OldSize  float64 `json:"old_size,string"`
	Price    float64 `json:"price,string"`
	Side     string  `json:"side"`
}

func (g *GDAX) WebsocketSubscribe(product string, conn *websocket.Conn) error {
	subscribe := GDAXWebsocketSubscribe{"subscribe", product}
	json, err := common.JSONEncode(subscribe)
	if err != nil {
		return err
	}

	err = conn.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		return err
	}
	return nil
}

func (g *GDAX) WebsocketClient() {
	for g.Enabled && g.Websocket {
		var Dialer websocket.Dialer
		conn, _, err := Dialer.Dial(GDAX_WEBSOCKET_URL, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", g.GetName(), err)
			continue
		}

		log.Printf("%s Connected to Websocket.\n", g.GetName())

		currencies := []string{}
		for _, x := range g.EnabledPairs {
			currency := x[0:3] + "-" + x[3:]
			currencies = append(currencies, currency)
		}

		for _, x := range currencies {
			err = g.WebsocketSubscribe(x, conn)
			if err != nil {
				log.Printf("%s Websocket subscription error: %s\n", g.GetName(), err)
				continue
			}
		}

		if g.Verbose {
			log.Printf("%s Subscribed to product messages.", g.GetName())
		}

		for g.Enabled && g.Websocket {
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
				err := common.JSONDecode(resp, &msgType)
				if err != nil {
					log.Println(err)
					continue
				}

				switch msgType.Type {
				case "error":
					log.Println(string(resp))
					break
				case "received":
					received := GDAXWebsocketReceived{}
					err := common.JSONDecode(resp, &received)
					if err != nil {
						log.Println(err)
						continue
					}
				case "open":
					open := GDAXWebsocketOpen{}
					err := common.JSONDecode(resp, &open)
					if err != nil {
						log.Println(err)
						continue
					}
				case "done":
					done := GDAXWebsocketDone{}
					err := common.JSONDecode(resp, &done)
					if err != nil {
						log.Println(err)
						continue
					}
				case "match":
					match := GDAXWebsocketMatch{}
					err := common.JSONDecode(resp, &match)
					if err != nil {
						log.Println(err)
						continue
					}
				case "change":
					change := GDAXWebsocketChange{}
					err := common.JSONDecode(resp, &change)
					if err != nil {
						log.Println(err)
						continue
					}
				}
			}
		}
		conn.Close()
		log.Printf("%s Websocket client disconnected.", g.GetName())
	}
}
