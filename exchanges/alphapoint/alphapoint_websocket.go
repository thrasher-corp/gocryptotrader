package alphapoint

import (
	"net/http"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	alphapointDefaultWebsocketURL = "wss://sim3.alphapoint.com:8401/v1/GetTicker/"
)

// WebsocketClient starts a new webstocket connection
func (e *Exchange) WebsocketClient() {
	for e.Enabled {
		var dialer gws.Dialer
		var err error
		var httpResp *http.Response
		endpoint, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
		if err != nil {
			log.Errorln(log.WebsocketMgr, err)
		}
		e.WebsocketConn, httpResp, err = dialer.Dial(endpoint, http.Header{})
		httpResp.Body.Close() // not used, so safely free the body

		if err != nil {
			log.Errorf(log.ExchangeSys, "%s Unable to connect to Websocket. Error: %s\n", e.Name, err)
			continue
		}

		if e.Verbose {
			log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", e.Name)
		}

		err = e.WebsocketConn.WriteMessage(gws.TextMessage, []byte(`{"messageType": "logon"}`))
		if err != nil {
			log.Errorln(log.ExchangeSys, err)
			return
		}

		for e.Enabled {
			msgType, resp, err := e.WebsocketConn.ReadMessage()
			if err != nil {
				log.Errorln(log.ExchangeSys, err)
				break
			}

			if msgType == gws.TextMessage {
				type MsgType struct {
					MessageType string `json:"messageType"`
				}

				msgType := MsgType{}
				err := json.Unmarshal(resp, &msgType)
				if err != nil {
					log.Errorln(log.ExchangeSys, err)
					continue
				}

				if msgType.MessageType == "Ticker" {
					ticker := WebsocketTicker{}
					err = json.Unmarshal(resp, &ticker)
					if err != nil {
						log.Errorln(log.ExchangeSys, err)
						continue
					}
				}
			}
		}
		e.WebsocketConn.Close()
		log.Debugf(log.ExchangeSys, "%s Websocket client disconnected.", e.Name)
	}
}
