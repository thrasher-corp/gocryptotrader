package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type LakeBTCWebsocketTickerResponse struct {
	Blah float64
}

const (
	LAKEBTC_WEBSOCKET_URL = "wss://www.LakeBTC.com/websocket"
)

func WSRailsSubscribe(channel string, conn *websocket.Conn) {
	data := fmt.Sprintf(`["websocket_rails.subscribe", {"data":{"channel": "%s" }}]`, channel)
	err := conn.WriteMessage(websocket.TextMessage, []byte(data))

	if err != nil {
		log.Println(err)
		return
	}
}

func WSRailsUnsubscribe(channel string, conn *websocket.Conn) {
	data := fmt.Sprintf(`["websocket_rails.unsubscribe", {"data":{"channel": "%s" }}]`, channel)
	err := conn.WriteMessage(websocket.TextMessage, []byte(data))

	if err != nil {
		log.Println(err)
		return
	}
}

func WSRailsPong(id string, conn *websocket.Conn) {
	data := fmt.Sprintf(`["websocket_rails.pong", {"data":{"connection_id": %s}}]`, id)
	err := conn.WriteMessage(websocket.TextMessage, []byte(data))

	if err != nil {
		log.Println(err)
		return
	}
}

func (l *LakeBTC) WebsocketClient() {
	for l.Enabled && l.Websocket {
		var Dialer websocket.Dialer
		conn, _, err := Dialer.Dial(LAKEBTC_WEBSOCKET_URL, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", l.GetName(), err)
			continue
		}

		log.Printf("%s Connected to Websocket.\n", l.GetName())

		for l.Enabled && l.Websocket {
			msgType, resp, err := conn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}

			response := [][]interface{}{}
			err = JSONDecode(resp, &response)

			if err != nil {
				log.Println(err)
				break
			}

			if msgType == websocket.TextMessage {
				event := response[0][0]
				data := response[0][1]

				switch event {
				case "client_connected":
					WSRailsSubscribe("ticker", conn)
					for _, x := range l.EnabledPairs {
						currency := x[3:]
						WSRailsSubscribe(fmt.Sprintf("orderbook_%s", currency), conn)
					}
				case "websocket_rails.subscribe":
				case "websocket_rails.ping":
					WSRailsPong("null", conn)
				case "update":
					update := data.(map[string]interface{})
					channel := update["channel"]
					data = update["data"]
					dataJSON, err := JSONEncode(data)

					if err != nil {
						log.Println(err)
						continue
					}

					switch channel {
					case "ticker":
						ticker := LakeBTCTickerResponse{}
						err = JSONDecode(dataJSON, &ticker)

						if err != nil {
							log.Println(err)
							continue
						}
					case "orderbook_USD", "orderbook_CNY":
						orderbook := LakeBTCOrderbook{}
						err = JSONDecode(dataJSON, &orderbook)

						if err != nil {
							log.Println(err)
							continue
						}
					}
				}
			}
		}
		conn.Close()
		log.Printf("%s Websocket client disconnected.\n", l.GetName())
	}
}
