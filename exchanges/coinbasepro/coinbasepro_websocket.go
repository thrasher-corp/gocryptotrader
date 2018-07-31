package coinbasepro

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

const (
	coinbaseproWebsocketURL = "wss://ws-feed.pro.coinbase.com"
)

// WebsocketSubscribe subscribes to a websocket connection
func (c *CoinbasePro) WebsocketSubscribe(product string, conn *websocket.Conn) error {
	subscribe := WebsocketSubscribe{"subscribe", product}
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

// WebsocketClient initiates a websocket client
func (c *CoinbasePro) WebsocketClient() {
	for c.Enabled && c.Websocket {
		var Dialer websocket.Dialer
		conn, _, err := Dialer.Dial(coinbaseproWebsocketURL, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", c.GetName(), err)
			continue
		}

		log.Printf("%s Connected to Websocket.\n", c.GetName())

		currencies := []string{}
		for _, x := range c.EnabledPairs {
			currency := x[0:3] + "-" + x[3:]
			currencies = append(currencies, currency)
		}

		for _, x := range currencies {
			err = c.WebsocketSubscribe(x, conn)
			if err != nil {
				log.Printf("%s Websocket subscription error: %s\n", c.GetName(), err)
				continue
			}
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
					received := WebsocketReceived{}
					err := common.JSONDecode(resp, &received)
					if err != nil {
						log.Println(err)
						continue
					}
				case "open":
					open := WebsocketOpen{}
					err := common.JSONDecode(resp, &open)
					if err != nil {
						log.Println(err)
						continue
					}
				case "done":
					done := WebsocketDone{}
					err := common.JSONDecode(resp, &done)
					if err != nil {
						log.Println(err)
						continue
					}
				case "match":
					match := WebsocketMatch{}
					err := common.JSONDecode(resp, &match)
					if err != nil {
						log.Println(err)
						continue
					}
				case "change":
					change := WebsocketChange{}
					err := common.JSONDecode(resp, &change)
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
