package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	WebsocketResponseSuccess = "OK"
)

var WebsocketRoutes = Routes{
	Route{
		"ws",
		"GET",
		"/ws",
		WebsocketClientHandler,
	},
}

type WebsocketClient struct {
	ID            int
	Conn          *websocket.Conn
	LastRecv      time.Time
	Authenticated bool
}

type WebsocketEvent struct {
	Exchange  string `json:"exchange,omitempty"`
	AssetType string `json:"assetType,omitempty"`
	Event     string
	Data      interface{}
}

type WebsocketEventResponse struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
}

type WebsocketTickerRequest struct {
	Exchange  string `json:"exchangeName"`
	Currency  string `json:"currency"`
	AssetType string `json:"assetType"`
}

var WebsocketClientHub []WebsocketClient

func WebsocketClientHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		WriteBufferSize: 1024,
		ReadBufferSize:  1024,
	}

	newClient := WebsocketClient{
		ID: len(WebsocketClientHub),
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	newClient.Conn = conn
	WebsocketClientHub = append(WebsocketClientHub, newClient)
	log.Println("New websocket client connected.")
}

func DisconnectWebsocketClient(id int, err error) {
	for i := range WebsocketClientHub {
		if WebsocketClientHub[i].ID == id {
			WebsocketClientHub[i].Conn.Close()
			WebsocketClientHub = append(WebsocketClientHub[:i], WebsocketClientHub[i+1:]...)
			log.Printf("Disconnected Websocket client, error: %s", err)
			return
		}
	}
}

func SendWebsocketMessage(id int, data interface{}) error {
	for _, x := range WebsocketClientHub {
		if x.ID == id {
			return x.Conn.WriteJSON(data)
		}
	}
	return nil
}

func BroadcastWebsocketMessage(evt WebsocketEvent) error {
	log.Println(evt)
	for _, x := range WebsocketClientHub {
		x.Conn.WriteJSON(evt)
	}
	return nil
}

type WebsocketAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func WebsocketHandler() {
	for {
		for x := range WebsocketClientHub {
			msgType, msg, err := WebsocketClientHub[x].Conn.ReadMessage()
			if err != nil {
				DisconnectWebsocketClient(x, err)
				continue
			}

			if msgType != websocket.TextMessage {
				DisconnectWebsocketClient(x, err)
				continue
			}

			var evt WebsocketEvent
			err = common.JSONDecode(msg, &evt)
			if err != nil {
				log.Println(err)
				continue
			}

			if evt.Event == "" {
				DisconnectWebsocketClient(x, errors.New("Websocket client sent data we did not understand"))
				continue
			}

			dataJSON, err := common.JSONEncode(evt.Data)
			if err != nil {
				log.Println(err)
				continue
			}

			if !WebsocketClientHub[x].Authenticated && evt.Event != "auth" {
				wsResp := WebsocketEventResponse{
					Event: "auth",
					Error: "you must authenticate first",
				}
				SendWebsocketMessage(x, wsResp)
				DisconnectWebsocketClient(x, errors.New("Websocket client did not auth"))
				continue
			} else if !WebsocketClientHub[x].Authenticated && evt.Event == "auth" {
				var auth WebsocketAuth
				err = common.JSONDecode(dataJSON, &auth)
				if err != nil {
					log.Println(err)
					continue
				}
				hashPW := common.HexEncodeToString(common.GetSHA256([]byte("password")))
				if auth.Username == "username" && auth.Password == hashPW {
					WebsocketClientHub[x].Authenticated = true
					wsResp := WebsocketEventResponse{
						Event: "auth",
						Data:  WebsocketResponseSuccess,
					}
					SendWebsocketMessage(x, wsResp)
					log.Println("Websocket client authenticated successfully")
					continue
				} else {
					wsResp := WebsocketEventResponse{
						Event: "auth",
						Error: "invalid username/password",
					}
					SendWebsocketMessage(x, wsResp)
					DisconnectWebsocketClient(x, errors.New("Websocket client sent wrong username/password"))
					continue
				}
			}
			switch evt.Event {
			case "GetConfig":
				wsResp := WebsocketEventResponse{
					Event: "GetConfig",
					Data:  bot.config,
				}
				SendWebsocketMessage(x, wsResp)
				continue
			case "SaveConfig":
				wsResp := WebsocketEventResponse{
					Event: "SaveConfig",
				}
				var cfg config.Config
				err := common.JSONDecode(dataJSON, &cfg)
				if err != nil {
					wsResp.Error = err.Error()
					SendWebsocketMessage(x, wsResp)
					log.Println(err)
					continue
				}

				//Save change the settings
				for x := range bot.config.Exchanges {
					for i := 0; i < len(cfg.Exchanges); i++ {
						if cfg.Exchanges[i].Name == bot.config.Exchanges[x].Name {
							bot.config.Exchanges[x].Enabled = cfg.Exchanges[i].Enabled
							bot.config.Exchanges[x].APIKey = cfg.Exchanges[i].APIKey
							bot.config.Exchanges[x].APISecret = cfg.Exchanges[i].APISecret
							bot.config.Exchanges[x].EnabledPairs = cfg.Exchanges[i].EnabledPairs
						}
					}
				}

				//Reload the configuration
				err = bot.config.SaveConfig(bot.configFile)
				if err != nil {
					wsResp.Error = err.Error()
					SendWebsocketMessage(x, wsResp)
					continue
				}
				err = bot.config.LoadConfig(bot.configFile)
				if err != nil {
					wsResp.Error = err.Error()
					SendWebsocketMessage(x, wsResp)
					continue
				}
				setupBotExchanges()
				wsResp.Data = WebsocketResponseSuccess
				SendWebsocketMessage(x, wsResp)
				continue
			case "GetAccountInfo":
				accountInfo := GetAllEnabledExchangeAccountInfo()
				wsResp := WebsocketEventResponse{
					Event: "GetAccountInfo",
					Data:  accountInfo,
				}
				SendWebsocketMessage(x, wsResp)
				continue
			case "GetTicker":
				wsResp := WebsocketEventResponse{
					Event: "GetTicker",
				}
				var tickerReq WebsocketTickerRequest
				err := common.JSONDecode(dataJSON, &tickerReq)
				if err != nil {
					wsResp.Error = err.Error()
					SendWebsocketMessage(x, wsResp)
					log.Println(err)
					continue
				}

				data, err := GetSpecificTicker(tickerReq.Currency,
					tickerReq.Exchange, tickerReq.AssetType)

				if err != nil {
					wsResp.Error = err.Error()
					SendWebsocketMessage(x, wsResp)
					log.Println(err)
					continue
				}
				wsResp.Data = data
				SendWebsocketMessage(x, wsResp)
				continue

			case "GetTickers":
				wsResp := WebsocketEventResponse{
					Event: "GetTickers",
				}
				tickers := GetAllActiveTickers()
				wsResp.Data = tickers
				SendWebsocketMessage(x, wsResp)
				continue
			}
		}
		time.Sleep(time.Millisecond)
	}
}
