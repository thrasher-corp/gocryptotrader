package alphapoint

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	alphapointDefaultWebsocketURL = "wss://sim3.alphapoint.com:8401/v1/GetTicker/"
)

// WebsocketClient starts a new webstocket connection
func (a *Alphapoint) WebsocketClient() {
	for a.Enabled {
		var dialer websocket.Dialer
		var err error
		var httpResp *http.Response
		endpoint, err := a.API.Endpoints.GetURL(exchange.WebsocketSpot)
		if err != nil {
			log.Error(log.WebsocketMgr, err)
		}
		a.WebsocketConn, httpResp, err = dialer.Dial(endpoint, http.Header{})
		httpResp.Body.Close() // not used, so safely free the body

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
				log.Error(log.ExchangeSys, err)
				break
			}

			if msgType == websocket.TextMessage {
				type MsgType struct {
					MessageType string `json:"messageType"`
				}

				msgType := MsgType{}
				err := json.Unmarshal(resp, &msgType)
				if err != nil {
					log.Error(log.ExchangeSys, err)
					continue
				}

				if msgType.MessageType == "Ticker" {
					ticker := WebsocketTicker{}
					err = json.Unmarshal(resp, &ticker)
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
