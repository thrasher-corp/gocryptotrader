package services

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

// Websocket tests the websocket connections
type Websocket struct{}

// Vars for the websocket client
var (
	WSConn *websocket.Conn
)

// WebsocketEvent is the struct used for websocket events
type WebsocketEvent struct {
	Exchange  string `json:"exchange,omitempty"`
	AssetType string `json:"assetType,omitempty"`
	Event     string
	Data      interface{}
}

// WebsocketAuth is the struct used for a websocket auth request
type WebsocketAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// WebsocketEventResponse is the struct used for websocket event responses
type WebsocketEventResponse struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
}

// WebsocketOrderbookTickerRequest is a struct used for ticker and orderbook
// requests
type WebsocketOrderbookTickerRequest struct {
	Exchange  string `json:"exchangeName"`
	Currency  string `json:"currency"`
	AssetType string `json:"assetType"`
}

// SendWebsocketEvent sends a websocket event message
func (w *Websocket) SendWebsocketEvent(event string, reqData interface{}, result *WebsocketEventResponse) error {
	req := WebsocketEvent{
		Event: event,
	}

	if reqData != nil {
		req.Data = reqData
	}

	err := WSConn.WriteJSON(req)
	if err != nil {
		return err
	}

	err = WSConn.ReadJSON(&result)
	if err != nil {
		return err
	}

	if result.Error != "" {
		return errors.New(result.Error)
	}

	return nil
}

// Run starts the websocket service
func (w *Websocket) Run() {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to load config file: %s", err)
	}

	listenAddr := cfg.Webserver.ListenAddress
	wsHost := fmt.Sprintf("ws://%s:%d/ws", common.ExtractHost(listenAddr),
		common.ExtractPort(listenAddr))
	log.Printf("Connecting to websocket host: %s", wsHost)

	var Dialer websocket.Dialer
	WSConn, _, err = Dialer.Dial(wsHost, http.Header{})
	if err != nil {
		log.Println("Unable to connect to websocket server")
		return
	}
	log.Println("Connected to websocket!")

	log.Println("Authenticating..")
	var wsResp WebsocketEventResponse
	reqData := WebsocketAuth{
		Username: cfg.Webserver.AdminUsername,
		Password: common.HexEncodeToString(common.GetSHA256([]byte(cfg.Webserver.AdminPassword))),
	}
	err = w.SendWebsocketEvent("auth", reqData, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Authenticated successfully")

	log.Println("Getting config..")
	err = w.SendWebsocketEvent("GetConfig", nil, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Fetched config.")

	dataJSON, err := common.JSONEncode(&wsResp.Data)
	if err != nil {
		log.Fatal(err)
	}

	var resultCfg config.Config
	err = common.JSONDecode(dataJSON, &resultCfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Saving config..")
	origBotName := resultCfg.Name
	resultCfg.Name = "TEST"
	err = w.SendWebsocketEvent("SaveConfig", resultCfg, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Saved config!")
	resultCfg.Name = origBotName
	err = w.SendWebsocketEvent("SaveConfig", resultCfg, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Saved config (restored original bot name)!")

	log.Println("Getting account info..")
	err = w.SendWebsocketEvent("GetAccountInfo", nil, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Got account info!")

	log.Println("Getting tickers..")
	err = w.SendWebsocketEvent("GetTickers", nil, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Got tickers!")

	log.Println("Getting specific ticker..")
	dataReq := WebsocketOrderbookTickerRequest{
		Exchange:  "Bitfinex",
		Currency:  "BTCUSD",
		AssetType: "SPOT",
	}

	err = w.SendWebsocketEvent("GetTicker", dataReq, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Got ticker!")

	log.Println("Getting orderbooks..")
	err = w.SendWebsocketEvent("GetOrderbooks", nil, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Got orderbooks!")

	log.Println("Getting specific orderbook..")
	err = w.SendWebsocketEvent("GetOrderbook", dataReq, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Got orderbook!")

	for {
		var wsEvent WebsocketEventResponse
		err = WSConn.ReadJSON(&wsEvent)
		if err != nil {
			break
		}

		log.Printf("Recv'd: %s", wsEvent.Event)
	}
	WSConn.Close()
}
