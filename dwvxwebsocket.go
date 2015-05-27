package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

const (
	DWVX_WEBSOCKET_URL = "wss://api.dwvx.com.au:8401/v1/GetTicker/"
)

func (d *DWVX) WebsocketClient() {
	for d.Enabled && d.Websocket {
		var Dialer websocket.Dialer
		var err error
		d.WebsocketConn, _, err = Dialer.Dial(DWVX_WEBSOCKET_URL, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", d.Name, err)
			continue
		}

		if d.Verbose {
			log.Printf("%s Connected to Websocket.\n", d.Name)
		}

		err = d.WebsocketConn.WriteMessage(websocket.TextMessage, []byte(`{"messageType": "logon"}`))

		if err != nil {
			log.Println(err)
			return
		}

		for d.Enabled && d.Websocket {
			msgType, resp, err := d.WebsocketConn.ReadMessage()
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
				err := JSONDecode(resp, &msgType)
				if err != nil {
					log.Println(err)
					continue
				}

				switch msgType.MessageType {
				case "Ticker":
					ticker := AlphapointWebsocketTicker{}
					err = JSONDecode(resp, &ticker)
					if err != nil {
						log.Println(err)
						continue
					}
				}
			}
		}
		d.WebsocketConn.Close()
		log.Printf("%s Websocket client disconnected.", d.Name)
	}
}
