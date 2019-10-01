package alphapoint

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	alphapointDefaultWebsocketURL = "wss://sim3.alphapoint.com:8401/v1/GetTicker/"
)

// WebsocketClient starts a new webstocket connection
func (a *Alphapoint) WebsocketClient() {
	for a.Enabled {
		var dialer websocket.Dialer
		var err error
		a.WebsocketConn, _, err = dialer.Dial(a.API.Endpoints.WebsocketURL, http.Header{})

		if err != nil {
			log.Errorf(log.ExchangeSys, "%s Unable to connect to Websocket. Error: %s\n", a.Name, err)
			continue
		}

		if a.Verbose {
			log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", a.Name)
		}

		err = a.WebsocketConn.WriteMessage(websocket.TextMessage, []byte(`{"messageType": "logon"}`))

		if err != nil {
			log.Error(log.ExchangeSys, err)
			return
		}

		for a.Enabled {
			msgType, resp, err := a.WebsocketConn.ReadMessage()
			if err != nil {
				a.Websocket.ReadMessageErrors <- err
				log.Error(log.ExchangeSys, err)
				break
			}

			if msgType == websocket.TextMessage {
				type MsgType struct {
					MessageType string `json:"messageType"`
				}

				msgType := MsgType{}
				err := common.JSONDecode(resp, &msgType)
				if err != nil {
					log.Error(log.ExchangeSys, err)
					continue
				}

				if msgType.MessageType == "Ticker" {
					ticker := WebsocketTicker{}
					err = common.JSONDecode(resp, &ticker)
					if err != nil {
						log.Error(log.ExchangeSys, err)
						continue
					}
				}
			}
		}
		a.WebsocketConn.Close()
		log.Debugf(log.ExchangeSys, "%s Websocket client disconnected.", a.Name)
	}
}
