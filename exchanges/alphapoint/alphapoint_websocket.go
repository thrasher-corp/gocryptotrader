package alphapoint

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	alphapointDefaultWebsocketURL = "wss://sim3.alphapoint.com:8401/v1/GetTicker/"
)

// WebsocketClient starts a new webstocket connection
func (a *Alphapoint) WebsocketClient() {
	for a.Enabled {
		var Dialer websocket.Dialer
		var err error
		a.WebsocketConn, _, err = Dialer.Dial(a.API.Endpoints.WebsocketURL, http.Header{})

		if err != nil {
			log.Errorf("%s Unable to connect to Websocket. Error: %s\n", a.Name, err)
			continue
		}

		if a.Verbose {
			log.Debugf("%s Connected to Websocket.\n", a.Name)
		}

		err = a.WebsocketConn.WriteMessage(websocket.TextMessage, []byte(`{"messageType": "logon"}`))

		if err != nil {
			log.Error(err)
			return
		}

		for a.Enabled {
			msgType, resp, err := a.WebsocketConn.ReadMessage()
			if err != nil {
				log.Error(err)
				break
			}

			if msgType == websocket.TextMessage {
				type MsgType struct {
					MessageType string `json:"messageType"`
				}

				msgType := MsgType{}
				err := common.JSONDecode(resp, &msgType)
				if err != nil {
					log.Error(err)
					continue
				}

				if msgType.MessageType == "Ticker" {
					ticker := WebsocketTicker{}
					err = common.JSONDecode(resp, &ticker)
					if err != nil {
						log.Error(err)
						continue
					}
				}
			}
		}
		a.WebsocketConn.Close()
		log.Debugf("%s Websocket client disconnected.", a.Name)
	}
}
