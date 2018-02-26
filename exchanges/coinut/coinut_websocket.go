package coinut

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

const coinutWebsocketURL = "wss://wsapi.coinut.com"

// WebsocketClient initiates a websocket client
func (c *COINUT) WebsocketClient() {
	for c.Enabled && c.Websocket {
		var Dialer websocket.Dialer
		var err error
		c.WebsocketConn, _, err = Dialer.Dial(c.WebsocketURL, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", c.Name, err)
			continue
		}

		if c.Verbose {
			log.Printf("%s Connected to Websocket.\n", c.Name)
		}

		err = c.WebsocketConn.WriteMessage(websocket.TextMessage, []byte(`{"messageType": "hello_world"}`))

		if err != nil {
			log.Println(err)
			return
		}

		for c.Enabled && c.Websocket {
			msgType, resp, err := c.WebsocketConn.ReadMessage()
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
				log.Println(string(resp))
			}
		}
		c.WebsocketConn.Close()
		log.Printf("%s Websocket client disconnected.", c.Name)
	}
}
