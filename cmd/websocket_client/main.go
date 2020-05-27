package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

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
	Exchange  string     `json:"exchangeName"`
	Currency  string     `json:"currency"`
	AssetType asset.Item `json:"assetType"`
}

// SendWebsocketEvent sends a websocket event message
func SendWebsocketEvent(event string, reqData interface{}, result *WebsocketEventResponse) error {
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

func main() {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.File, true)
	if err != nil {
		log.Fatalf("Failed to load config file: %s", err)
	}

	listenAddr := cfg.RemoteControl.WebsocketRPC.ListenAddress
	wsHost := fmt.Sprintf("ws://%s:%d/ws", common.ExtractHost(listenAddr),
		common.ExtractPort(listenAddr))
	log.Printf("Connecting to websocket host: %s", wsHost)

	var dialer websocket.Dialer
	var resp *http.Response
	WSConn, resp, err = dialer.Dial(wsHost, http.Header{})
	if err != nil {
		log.Println("Unable to connect to websocket server")
		return
	}
	resp.Body.Close()
	log.Println("Connected to websocket!")

	log.Println("Authenticating..")
	var wsResp WebsocketEventResponse
	reqData := WebsocketAuth{
		Username: cfg.RemoteControl.Username,
		Password: crypto.HexEncodeToString(crypto.GetSHA256([]byte(cfg.RemoteControl.Password))),
	}
	err = SendWebsocketEvent("auth", reqData, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Authenticated successfully")

	log.Println("Getting config..")
	err = SendWebsocketEvent("GetConfig", nil, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Fetched config.")

	dataJSON, err := json.Marshal(&wsResp.Data)
	if err != nil {
		log.Fatal(err)
	}

	var resultCfg config.Config
	err = json.Unmarshal(dataJSON, &resultCfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Saving config..")
	origBotName := resultCfg.Name
	resultCfg.Name = "TEST"
	err = SendWebsocketEvent("SaveConfig", resultCfg, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Saved config!")
	resultCfg.Name = origBotName
	err = SendWebsocketEvent("SaveConfig", resultCfg, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Saved config (restored original bot name)!")

	log.Println("Getting account info..")
	err = SendWebsocketEvent("GetAccountInfo", nil, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Got account info!")

	log.Println("Getting tickers..")
	err = SendWebsocketEvent("GetTickers", nil, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Got tickers!")

	log.Println("Getting specific ticker..")
	dataReq := WebsocketOrderbookTickerRequest{
		Exchange:  "Bitfinex",
		Currency:  "BTCUSD",
		AssetType: asset.Spot,
	}

	err = SendWebsocketEvent("GetTicker", dataReq, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Got ticker!")

	log.Println("Getting orderbooks..")
	err = SendWebsocketEvent("GetOrderbooks", nil, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Got orderbooks!")

	log.Println("Getting specific orderbook..")
	err = SendWebsocketEvent("GetOrderbook", dataReq, &wsResp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Got orderbook!")

	for {
		var wsEvent WebsocketEvent
		err = WSConn.ReadJSON(&wsEvent)
		if err != nil {
			break
		}

		log.Printf("Recv'd: %s", wsEvent.Event)
	}
	WSConn.Close()
}
