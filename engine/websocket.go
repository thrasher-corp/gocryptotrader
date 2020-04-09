package engine

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Const vars for websocket
const (
	WebsocketResponseSuccess = "OK"
)

var (
	wsHub        *WebsocketHub
	wsHubStarted bool
)

type wsCommandHandler struct {
	authRequired bool
	handler      func(client *WebsocketClient, data interface{}) error
}

var wsHandlers = map[string]wsCommandHandler{
	"auth":             {authRequired: false, handler: wsAuth},
	"getconfig":        {authRequired: true, handler: wsGetConfig},
	"saveconfig":       {authRequired: true, handler: wsSaveConfig},
	"getaccountinfo":   {authRequired: true, handler: wsGetAccountInfo},
	"gettickers":       {authRequired: false, handler: wsGetTickers},
	"getticker":        {authRequired: false, handler: wsGetTicker},
	"getorderbooks":    {authRequired: false, handler: wsGetOrderbooks},
	"getorderbook":     {authRequired: false, handler: wsGetOrderbook},
	"getexchangerates": {authRequired: false, handler: wsGetExchangeRates},
	"getportfolio":     {authRequired: true, handler: wsGetPortfolio},
}

// NewWebsocketHub Creates a new websocket hub
func NewWebsocketHub() *WebsocketHub {
	return &WebsocketHub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *WebsocketClient),
		Unregister: make(chan *WebsocketClient),
		Clients:    make(map[*WebsocketClient]bool),
	}
}

func (h *WebsocketHub) run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				log.Debugln(log.WebsocketMgr, "websocket: disconnected client")
				delete(h.Clients, client)
				close(client.Send)
			}
		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					log.Debugln(log.WebsocketMgr, "websocket: disconnected client")
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}

// SendWebsocketMessage sends a websocket event to the client
func (c *WebsocketClient) SendWebsocketMessage(evt interface{}) error {
	data, err := json.Marshal(evt)
	if err != nil {
		log.Errorf(log.WebsocketMgr, "websocket: failed to send message: %s\n", err)
		return err
	}

	c.Send <- data
	return nil
}

func (c *WebsocketClient) read() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		msgType, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf(log.WebsocketMgr, "websocket: client disconnected, err: %s\n", err)
			}
			break
		}

		if msgType == websocket.TextMessage {
			var evt WebsocketEvent
			err := json.Unmarshal(message, &evt)
			if err != nil {
				log.Errorf(log.WebsocketMgr, "websocket: failed to decode JSON sent from client %s\n", err)
				continue
			}

			if evt.Event == "" {
				log.Warnln(log.WebsocketMgr, "websocket: client sent a blank event, disconnecting")
				continue
			}

			dataJSON, err := json.Marshal(evt.Data)
			if err != nil {
				log.Errorln(log.WebsocketMgr, "websocket: client sent data we couldn't JSON decode")
				break
			}

			req := strings.ToLower(evt.Event)
			log.Debugf(log.WebsocketMgr, "websocket: request received: %s\n", req)

			result, ok := wsHandlers[req]
			if !ok {
				log.Debugln(log.WebsocketMgr, "websocket: unsupported event")
				continue
			}

			if result.authRequired && !c.Authenticated {
				log.Warnf(log.WebsocketMgr, "Websocket: request %s failed due to unauthenticated request on an authenticated API\n", evt.Event)
				c.SendWebsocketMessage(WebsocketEventResponse{Event: evt.Event, Error: "unauthorised request on authenticated API"})
				continue
			}

			err = result.handler(c, dataJSON)
			if err != nil {
				log.Errorf(log.WebsocketMgr, "websocket: request %s failed. Error %s\n", evt.Event, err)
				continue
			}
		}
	}
}

func (c *WebsocketClient) write() {
	defer func() {
		c.Conn.Close()
	}()
	for { // nolint // ws client write routine loop
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Debugln(log.WebsocketMgr, "websocket: hub closed the channel")
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Errorf(log.WebsocketMgr, "websocket: failed to create new io.writeCloser: %s\n", err)
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				log.Errorf(log.WebsocketMgr, "websocket: failed to close io.WriteCloser: %s\n", err)
				return
			}
		}
	}
}

// StartWebsocketHandler starts the websocket hub and routine which
// handles clients
func StartWebsocketHandler() {
	if !wsHubStarted {
		wsHubStarted = true
		wsHub = NewWebsocketHub()
		go wsHub.run()
	}
}

// BroadcastWebsocketMessage meow
func BroadcastWebsocketMessage(evt WebsocketEvent) error {
	if !wsHubStarted {
		return errors.New("websocket service not started")
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	wsHub.Broadcast <- data
	return nil
}

// WebsocketClientHandler upgrades the HTTP connection to a websocket
// compatible one
func WebsocketClientHandler(w http.ResponseWriter, r *http.Request) {
	if !wsHubStarted {
		StartWebsocketHandler()
	}

	connectionLimit := Bot.Config.RemoteControl.WebsocketRPC.ConnectionLimit
	numClients := len(wsHub.Clients)

	if numClients >= connectionLimit {
		log.Warnf(log.WebsocketMgr,
			"websocket: client rejected due to websocket client limit reached. Number of clients %d. Limit %d.\n",
			numClients, connectionLimit)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	upgrader := websocket.Upgrader{
		WriteBufferSize: 1024,
		ReadBufferSize:  1024,
	}

	// Allow insecure origin if the Origin request header is present and not
	// equal to the Host request header. Default to false
	if Bot.Config.RemoteControl.WebsocketRPC.AllowInsecureOrigin {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(log.WebsocketMgr, err)
		return
	}

	client := &WebsocketClient{Hub: wsHub, Conn: conn, Send: make(chan []byte, 1024)}
	client.Hub.Register <- client
	log.Debugf(log.WebsocketMgr,
		"websocket: client connected. Connected clients: %d. Limit %d.\n",
		numClients+1, connectionLimit)

	go client.read()
	go client.write()
}

func wsAuth(client *WebsocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "auth",
	}

	var auth WebsocketAuth
	err := json.Unmarshal(data.([]byte), &auth)
	if err != nil {
		wsResp.Error = err.Error()
		client.SendWebsocketMessage(wsResp)
		return err
	}

	hashPW := crypto.HexEncodeToString(crypto.GetSHA256([]byte(Bot.Config.RemoteControl.Password)))
	if auth.Username == Bot.Config.RemoteControl.Username && auth.Password == hashPW {
		client.Authenticated = true
		wsResp.Data = WebsocketResponseSuccess
		log.Debugln(log.WebsocketMgr,
			"websocket: client authenticated successfully")
		return client.SendWebsocketMessage(wsResp)
	}

	wsResp.Error = "invalid username/password"
	client.authFailures++
	client.SendWebsocketMessage(wsResp)
	if client.authFailures >= Bot.Config.RemoteControl.WebsocketRPC.MaxAuthFailures {
		log.Debugf(log.WebsocketMgr,
			"websocket: disconnecting client, maximum auth failures threshold reached (failures: %d limit: %d)\n",
			client.authFailures, Bot.Config.RemoteControl.WebsocketRPC.MaxAuthFailures)
		wsHub.Unregister <- client
		return nil
	}

	log.Debugf(log.WebsocketMgr,
		"websocket: client sent wrong username/password (failures: %d limit: %d)\n",
		client.authFailures, Bot.Config.RemoteControl.WebsocketRPC.MaxAuthFailures)
	return nil
}

func wsGetConfig(client *WebsocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetConfig",
		Data:  Bot.Config,
	}
	return client.SendWebsocketMessage(wsResp)
}

func wsSaveConfig(client *WebsocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "SaveConfig",
	}
	var cfg config.Config
	err := json.Unmarshal(data.([]byte), &cfg)
	if err != nil {
		wsResp.Error = err.Error()
		client.SendWebsocketMessage(wsResp)
		return err
	}

	err = Bot.Config.UpdateConfig(Bot.Settings.ConfigFile, &cfg, Bot.Settings.EnableDryRun)
	if err != nil {
		wsResp.Error = err.Error()
		client.SendWebsocketMessage(wsResp)
		return err
	}

	SetupExchanges()
	wsResp.Data = WebsocketResponseSuccess
	return client.SendWebsocketMessage(wsResp)
}

func wsGetAccountInfo(client *WebsocketClient, data interface{}) error {
	accountInfo := GetAllEnabledExchangeAccountInfo()
	wsResp := WebsocketEventResponse{
		Event: "GetAccountInfo",
		Data:  accountInfo,
	}
	return client.SendWebsocketMessage(wsResp)
}

func wsGetTickers(client *WebsocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetTickers",
	}
	wsResp.Data = GetAllActiveTickers()
	return client.SendWebsocketMessage(wsResp)
}

func wsGetTicker(client *WebsocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetTicker",
	}
	var tickerReq WebsocketOrderbookTickerRequest
	err := json.Unmarshal(data.([]byte), &tickerReq)
	if err != nil {
		wsResp.Error = err.Error()
		client.SendWebsocketMessage(wsResp)
		return err
	}

	result, err := GetSpecificTicker(currency.NewPairFromString(tickerReq.Currency),
		tickerReq.Exchange, asset.Item(tickerReq.AssetType))

	if err != nil {
		wsResp.Error = err.Error()
		client.SendWebsocketMessage(wsResp)
		return err
	}
	wsResp.Data = result
	return client.SendWebsocketMessage(wsResp)
}

func wsGetOrderbooks(client *WebsocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetOrderbooks",
	}
	wsResp.Data = GetAllActiveOrderbooks()
	return client.SendWebsocketMessage(wsResp)
}

func wsGetOrderbook(client *WebsocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetOrderbook",
	}
	var orderbookReq WebsocketOrderbookTickerRequest
	err := json.Unmarshal(data.([]byte), &orderbookReq)
	if err != nil {
		wsResp.Error = err.Error()
		client.SendWebsocketMessage(wsResp)
		return err
	}

	result, err := GetSpecificOrderbook(currency.NewPairFromString(orderbookReq.Currency),
		orderbookReq.Exchange, asset.Item(orderbookReq.AssetType))

	if err != nil {
		wsResp.Error = err.Error()
		client.SendWebsocketMessage(wsResp)
		return err
	}
	wsResp.Data = result
	return client.SendWebsocketMessage(wsResp)
}

func wsGetExchangeRates(client *WebsocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetExchangeRates",
	}

	var err error
	wsResp.Data, err = currency.GetExchangeRates()
	if err != nil {
		return err
	}

	return client.SendWebsocketMessage(wsResp)
}

func wsGetPortfolio(client *WebsocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetPortfolio",
	}
	wsResp.Data = Bot.Portfolio.GetPortfolioSummary()
	return client.SendWebsocketMessage(wsResp)
}
