package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

var (
	WSConn *websocket.Conn
)

type WebsocketEvent struct {
	Event    string      `json:"event"`
	Data     interface{} `json:"data"`
	Exchange string      `json:"exchange,omitempty"`
}

type WebsocketAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type WebsocketEventResponse struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
}

type WebsocketTickerRequest struct {
	Exchange string `json:"exchangeName"`
	Currency string `json:"currency"`
}

func SendWebsocketAuth(username, password string) error {
	pwHash := common.HexEncodeToString(common.GetSHA256([]byte(password)))
	req := WebsocketEvent{
		Event: "auth",
		Data: WebsocketAuth{
			Username: username,
			Password: pwHash,
		},
	}

	return SendWebsocketMsg(req)
}

func SendWebsocketMsg(data interface{}) error {
	return WSConn.WriteJSON(data)
}

func GetWebsocketTicker(currency string) error {
	wsevt := WebsocketEvent{
		Event: "ticker",
		Data:  currency,
	}

	return SendWebsocketMsg(wsevt)
}

func main() {
	var Dialer websocket.Dialer
	var err error

	WSConn, _, err = Dialer.Dial("ws://localhost:9050/ws", http.Header{})

	if err != nil {
		log.Println("Unable to connect to websocket server")
		return
	}

	log.Println("Connected to websocket!")
	log.Println("Authenticating..")
	SendWebsocketAuth("blah", "blah")

	var wsResp WebsocketEventResponse
	err = WSConn.ReadJSON(&wsResp)
	if err != nil {
		log.Println(err)
		return
	}

	if wsResp.Error != "" {
		log.Fatal(wsResp.Error)
	}

	log.Println("Authenticated successfully")
	log.Println("Getting config..")

	req := WebsocketEvent{
		Event: "GetConfig",
	}

	err = WSConn.WriteJSON(req)
	if err != nil {
		log.Println(err)
		return
	}

	err = WSConn.ReadJSON(&wsResp)
	if err != nil {
		log.Println(err)
		return
	}

	if wsResp.Error != "" {
		log.Fatal(wsResp.Error)
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

	resultCfg.Name = "TEST"

	req = WebsocketEvent{
		Event: "SaveConfig",
		Data:  resultCfg,
	}

	log.Println("Saving config..")
	err = WSConn.WriteJSON(req)
	if err != nil {
		log.Fatal(err)
	}

	err = WSConn.ReadJSON(&wsResp)
	if err != nil {
		log.Println(err)
		return
	}

	if wsResp.Error != "" {
		log.Fatal(wsResp.Error)
	}

	log.Println("Saved config!")
	log.Println("Getting account info..")

	req = WebsocketEvent{
		Event: "GetAccountInfo",
	}

	err = WSConn.WriteJSON(req)
	if err != nil {
		log.Println(err)
		return
	}

	err = WSConn.ReadJSON(&wsResp)
	if err != nil {
		log.Println(err)
		return
	}

	if wsResp.Error != "" {
		log.Fatal(wsResp.Error)
	}

	log.Println("Getting tickers..")

	req = WebsocketEvent{
		Event: "GetTickers",
	}

	err = WSConn.WriteJSON(req)
	if err != nil {
		log.Println(err)
		return
	}

	err = WSConn.ReadJSON(&wsResp)
	if err != nil {
		log.Println(err)
		return
	}

	if wsResp.Error != "" {
		log.Fatal(wsResp.Error)
	}

	log.Println("Getting specific ticker..")

	var tickReq WebsocketTickerRequest
	tickReq.Currency = "LTCUSD"
	tickReq.Exchange = "Bitfinex"

	req = WebsocketEvent{
		Event: "GetTicker",
		Data:  tickReq,
	}

	err = WSConn.WriteJSON(req)
	if err != nil {
		log.Println(err)
		return
	}

	err = WSConn.ReadJSON(&wsResp)
	if err != nil {
		log.Println(err)
		return
	}

	if wsResp.Error != "" {
		log.Fatal(wsResp.Error)
	}

	log.Println(wsResp)

	for {
		msgType, resp, err := WSConn.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}

		log.Println(msgType)
		log.Println(string(resp))
	}
	time.Sleep(time.Second * 10)
	WSConn.Close()
}
