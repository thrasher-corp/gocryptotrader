package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

const (
	ALPHAPOINT_DEFAULT_WEBSOCKET_URL = "wss://sim3.alphapoint.com:8401/v1/GetTicker/"
)

type AlphapointWebsocketTicker struct {
	MessageType             string  `json:"messageType"`
	ProductPair             string  `json:"prodPair"`
	High                    float64 `json:"high"`
	Low                     float64 `json:"low"`
	Last                    float64 `json:"last"`
	Volume                  float64 `json:"volume"`
	Volume24Hrs             float64 `json:"volume24hrs"`
	Volume24HrsProduct2     float64 `json:"volume24hrsProduct2"`
	Total24HrQtyTraded      float64 `json:"Total24HrQtyTraded"`
	Total24HrProduct2Traded float64 `json:"Total24HrProduct2Traded"`
	Total24HrNumTrades      float64 `json:"Total24HrNumTrades"`
	Bid                     float64 `json:"bid"`
	Ask                     float64 `json:"ask"`
	BuyOrderCount           int     `json:"buyOrderCount"`
	SellOrderCount          int     `json:"sellOrderCount"`
}

func (a *Alphapoint) WebsocketClient() {
	for a.ExchangeEnabled && a.WebsocketEnabled {
		var Dialer websocket.Dialer
		var err error
		a.WebsocketConn, _, err = Dialer.Dial(a.WebsocketURL, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", a.ExchangeName, err)
			continue
		}

		if a.Verbose {
			log.Printf("%s Connected to Websocket.\n", a.ExchangeName)
		}

		err = a.WebsocketConn.WriteMessage(websocket.TextMessage, []byte(`{"messageType": "logon"}`))

		if err != nil {
			log.Println(err)
			return
		}

		for a.ExchangeEnabled && a.WebsocketEnabled {
			msgType, resp, err := a.WebsocketConn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}

			switch msgType {
			case websocket.TextMessage:
				type MsgType struct {
					MessageType string `json:"messageType"`
				}

				msgType := MsgType{}
				err := common.JSONDecode(resp, &msgType)
				if err != nil {
					log.Println(err)
					continue
				}

				switch msgType.MessageType {
				case "Ticker":
					ticker := AlphapointWebsocketTicker{}
					err = common.JSONDecode(resp, &ticker)
					if err != nil {
						log.Println(err)
						continue
					}
				}
			}
		}
		a.WebsocketConn.Close()
		log.Printf("%s Websocket client disconnected.", a.ExchangeName)
	}
}
