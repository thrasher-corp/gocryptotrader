package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// setupAPIServerManager checks and creates an api server manager
func setupAPIServerManager(remoteConfig *config.RemoteControlConfig, pprofConfig *config.Profiler, exchangeManager iExchangeManager, bot iBot, portfolioManager iPortfolioManager, configPath string) (*apiServerManager, error) {
	if remoteConfig == nil {
		return nil, errNilRemoteConfig
	}
	if pprofConfig == nil {
		return nil, errNilPProfConfig
	}
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if bot == nil {
		return nil, errNilBot
	}
	if configPath == "" {
		return nil, errEmptyConfigPath
	}
	return &apiServerManager{
		remoteConfig:           remoteConfig,
		pprofConfig:            pprofConfig,
		restListenAddress:      remoteConfig.DeprecatedRPC.ListenAddress,
		websocketListenAddress: remoteConfig.WebsocketRPC.ListenAddress,
		exchangeManager:        exchangeManager,
		bot:                    bot,
		gctConfigPath:          configPath,
		portfolioManager:       portfolioManager,
	}, nil
}

// IsRESTServerRunning safely checks whether the subsystem is running
func (m *apiServerManager) IsRESTServerRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.restStarted) == 1
}

// IsWebsocketServerRunning safely checks whether the subsystem is running
func (m *apiServerManager) IsWebsocketServerRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.websocketStarted) == 1
}

// StopRESTServer attempts to shutdown the subsystem
func (m *apiServerManager) StopRESTServer() error {
	if m == nil {
		return fmt.Errorf("api server %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.restStarted, 1, 0) {
		return fmt.Errorf("apiserver deprecated server %w", ErrSubSystemNotStarted)
	}
	err := m.restHTTPServer.Shutdown(context.Background())
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	m.wgRest.Wait()
	m.restRouter = nil
	return nil
}

func (m *apiServerManager) StopWebsocketServer() error {
	if m == nil {
		return fmt.Errorf("api server %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.websocketStarted, 1, 0) {
		return fmt.Errorf("apiserver websocket server %w", ErrSubSystemNotStarted)
	}

	err := m.websocketHTTPServer.Shutdown(context.Background())
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	m.websocketRouter = nil
	m.websocketHub = nil
	m.wgWebsocket.Wait()
	m.websocketHTTPServer = nil
	return nil
}

// newRouter takes in the exchange interfaces and returns a new multiplexer
// router
func (m *apiServerManager) newRouter(isREST bool) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	var routes []Route
	if common.ExtractPort(m.websocketListenAddress) == 80 {
		m.websocketListenAddress = common.ExtractHost(m.websocketListenAddress)
	} else {
		m.websocketListenAddress = strings.Join([]string{common.ExtractHost(m.websocketListenAddress),
			strconv.Itoa(common.ExtractPort(m.websocketListenAddress))}, ":")
	}

	if isREST {
		routes = []Route{
			{"", http.MethodGet, "/", m.getIndex},
			{"GetAllSettings", http.MethodGet, "/config/all", m.restGetAllSettings},
			{"SaveAllSettings", http.MethodPost, "/config/all/save", m.restSaveAllSettings},
			{"AllEnabledAccountInfo", http.MethodGet, "/exchanges/enabled/accounts/all", m.restGetAllEnabledAccountInfo},
			{"AllActiveExchangesAndCurrencies", http.MethodGet, "/exchanges/enabled/latest/all", m.restGetAllActiveTickers},
			{"GetPortfolio", http.MethodGet, "/portfolio/all", m.restGetPortfolio},
			{"AllActiveExchangesAndOrderbooks", http.MethodGet, "/exchanges/orderbook/latest/all", m.restGetAllActiveOrderbooks},
		}

		if m.pprofConfig.Enabled {
			if m.pprofConfig.MutexProfileFraction > 0 {
				runtime.SetMutexProfileFraction(m.pprofConfig.MutexProfileFraction)
			}
			log.Debugf(log.RESTSys,
				"HTTP Go performance profiler (pprof) endpoint enabled: http://%s:%d/debug/pprof/\n",
				common.ExtractHost(m.websocketListenAddress),
				common.ExtractPort(m.websocketListenAddress))
			router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		}
	} else {
		routes = []Route{
			{"ws", http.MethodGet, "/ws", m.WebsocketClientHandler},
		}
	}

	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(restLogger(route.HandlerFunc, route.Name)).
			Host(m.websocketListenAddress)
	}
	return router
}

// StartRESTServer starts a REST handler
func (m *apiServerManager) StartRESTServer() error {
	if !atomic.CompareAndSwapInt32(&m.restStarted, 0, 1) {
		return fmt.Errorf("rest server %w", errAlreadyRunning)
	}
	if !m.remoteConfig.DeprecatedRPC.Enabled {
		atomic.StoreInt32(&m.restStarted, 0)
		return fmt.Errorf("rest %w", errServerDisabled)
	}
	log.Debugf(log.RESTSys,
		"Deprecated RPC handler support enabled. Listen URL: http://%s:%d\n",
		common.ExtractHost(m.restListenAddress), common.ExtractPort(m.restListenAddress))
	m.restRouter = m.newRouter(true)
	if m.restHTTPServer == nil {
		m.restHTTPServer = &http.Server{
			Addr:    m.restListenAddress,
			Handler: m.restRouter,
		}
	}
	m.wgRest.Add(1)
	go func() {
		defer m.wgRest.Done()
		err := m.restHTTPServer.ListenAndServe()
		if err != nil {
			atomic.StoreInt32(&m.restStarted, 0)
			if !errors.Is(err, http.ErrServerClosed) {
				log.Error(log.APIServerMgr, err)
			}
		}
	}()
	return nil
}

// restLogger logs the requests internally
func restLogger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		inner.ServeHTTP(w, r)

		log.Debugf(log.RESTSys,
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}

// writeResponse outputs a JSON response of the response interface
func writeResponse(w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(response)
}

// handleError prints the REST method and error
func handleError(method string, err error) {
	log.Errorf(log.APIServerMgr, "RESTful %s: handler failed to send JSON response. Error %s\n",
		method, err)
}

// restGetAllSettings replies to a request with an encoded JSON response about the
// trading Bots configuration.
func (m *apiServerManager) restGetAllSettings(w http.ResponseWriter, r *http.Request) {
	err := writeResponse(w, config.GetConfig())
	if err != nil {
		handleError(r.Method, err)
	}
}

// restSaveAllSettings saves all current settings from request body as a JSON
// document then reloads state and returns the settings
func (m *apiServerManager) restSaveAllSettings(w http.ResponseWriter, r *http.Request) {
	// Get the data from the request
	decoder := json.NewDecoder(r.Body)
	var responseData config.Post
	err := decoder.Decode(&responseData)
	if err != nil {
		handleError(r.Method, err)
	}
	// Save change the settings
	cfg := config.GetConfig()
	err = cfg.UpdateConfig(m.gctConfigPath, &responseData.Data, false)
	if err != nil {
		handleError(r.Method, err)
	}

	err = writeResponse(w, cfg)
	if err != nil {
		handleError(r.Method, err)
	}
	err = m.bot.SetupExchanges()
	if err != nil {
		handleError(r.Method, err)
	}
}

// restGetAllActiveOrderbooks returns all enabled exchange orderbooks
func (m *apiServerManager) restGetAllActiveOrderbooks(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeOrderbooks
	response.Data = getAllActiveOrderbooks(m.exchangeManager)
	err := writeResponse(w, response)
	if err != nil {
		handleError(r.Method, err)
	}
}

// restGetPortfolio returns the Bot portfolio manager
func (m *apiServerManager) restGetPortfolio(w http.ResponseWriter, r *http.Request) {
	result := m.portfolioManager.GetPortfolioSummary()
	err := writeResponse(w, result)
	if err != nil {
		handleError(r.Method, err)
	}
}

// restGetAllActiveTickers returns all active tickers
func (m *apiServerManager) restGetAllActiveTickers(w http.ResponseWriter, r *http.Request) {
	var response AllEnabledExchangeCurrencies
	response.Data = getAllActiveTickers(m.exchangeManager)
	err := writeResponse(w, response)
	if err != nil {
		handleError(r.Method, err)
	}
}

// restGetAllEnabledAccountInfo via get request returns JSON response of account
// info
func (m *apiServerManager) restGetAllEnabledAccountInfo(w http.ResponseWriter, r *http.Request) {
	response := getAllActiveAccounts(m.exchangeManager)
	err := writeResponse(w, response)
	if err != nil {
		handleError(r.Method, err)
	}
}

// getIndex returns an HTML snippet for when a user requests the index URL
func (m *apiServerManager) getIndex(w http.ResponseWriter, _ *http.Request) {
	_, err := fmt.Fprint(w, restIndexResponse)
	if err != nil {
		log.Error(log.APIServerMgr, err)
	}
	w.WriteHeader(http.StatusOK)
}

// getAllActiveOrderbooks returns all enabled exchanges orderbooks
func getAllActiveOrderbooks(m iExchangeManager) []EnabledExchangeOrderbooks {
	var orderbookData []EnabledExchangeOrderbooks
	exchanges := m.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes(true)
		exchName := exchanges[x].GetName()
		var exchangeOB EnabledExchangeOrderbooks
		exchangeOB.ExchangeName = exchName

		for y := range assets {
			currencies, err := exchanges[x].GetEnabledPairs(assets[y])
			if err != nil {
				log.Errorf(log.APIServerMgr,
					"Exchange %s could not retrieve enabled currencies. Err: %s\n",
					exchName,
					err)
				continue
			}
			for z := range currencies {
				ob, err := exchanges[x].FetchOrderbook(currencies[z], assets[y])
				if err != nil {
					log.Errorf(log.APIServerMgr,
						"Exchange %s failed to retrieve %s orderbook. Err: %s\n", exchName,
						currencies[z].String(),
						err)
					continue
				}
				exchangeOB.ExchangeValues = append(exchangeOB.ExchangeValues, *ob)
			}
			orderbookData = append(orderbookData, exchangeOB)
		}
		orderbookData = append(orderbookData, exchangeOB)
	}
	return orderbookData
}

// getAllActiveTickers returns all enabled exchanges tickers
func getAllActiveTickers(m iExchangeManager) []EnabledExchangeCurrencies {
	var tickers []EnabledExchangeCurrencies
	exchanges := m.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes(true)
		exchName := exchanges[x].GetName()
		var exchangeTickers EnabledExchangeCurrencies
		exchangeTickers.ExchangeName = exchName

		for y := range assets {
			currencies, err := exchanges[x].GetEnabledPairs(assets[y])
			if err != nil {
				log.Errorf(log.APIServerMgr,
					"Exchange %s could not retrieve enabled currencies. Err: %s\n",
					exchName,
					err)
				continue
			}
			for z := range currencies {
				t, err := exchanges[x].FetchTicker(currencies[z], assets[y])
				if err != nil {
					log.Errorf(log.APIServerMgr,
						"Exchange %s failed to retrieve %s ticker. Err: %s\n", exchName,
						currencies[z].String(),
						err)
					continue
				}
				exchangeTickers.ExchangeValues = append(exchangeTickers.ExchangeValues, *t)
			}
			tickers = append(tickers, exchangeTickers)
		}
		tickers = append(tickers, exchangeTickers)
	}
	return tickers
}

// getAllActiveAccounts returns all enabled exchanges accounts
func getAllActiveAccounts(m iExchangeManager) []AllEnabledExchangeAccounts {
	var accounts []AllEnabledExchangeAccounts
	exchanges := m.GetExchanges()
	for x := range exchanges {
		assets := exchanges[x].GetAssetTypes(true)
		exchName := exchanges[x].GetName()
		var exchangeAccounts AllEnabledExchangeAccounts
		for y := range assets {
			a, err := exchanges[x].FetchAccountInfo(assets[y])
			if err != nil {
				log.Errorf(log.APIServerMgr,
					"Exchange %s failed to retrieve %s ticker. Err: %s\n",
					exchName,
					assets[y],
					err)
				continue
			}
			exchangeAccounts.Data = append(exchangeAccounts.Data, a)
		}
		accounts = append(accounts, exchangeAccounts)
	}
	return accounts
}

// StartWebsocketServer starts a Websocket handler
func (m *apiServerManager) StartWebsocketServer() error {
	if !atomic.CompareAndSwapInt32(&m.websocketStarted, 0, 1) {
		return fmt.Errorf("websocket server %w", errAlreadyRunning)
	}
	if !m.remoteConfig.WebsocketRPC.Enabled {
		atomic.StoreInt32(&m.websocketStarted, 0)
		return fmt.Errorf("websocket %w", errServerDisabled)
	}
	log.Debugf(log.APIServerMgr,
		"Websocket RPC support enabled. Listen URL: ws://%s:%d/ws\n",
		common.ExtractHost(m.websocketListenAddress), common.ExtractPort(m.websocketListenAddress))
	m.websocketRouter = m.newRouter(false)
	if m.websocketHTTPServer == nil {
		m.websocketHTTPServer = &http.Server{
			Addr:    m.websocketListenAddress,
			Handler: m.websocketRouter,
		}
	}

	m.wgWebsocket.Add(1)
	go func() {
		defer m.wgWebsocket.Done()
		err := m.websocketHTTPServer.ListenAndServe()
		if err != nil {
			atomic.StoreInt32(&m.websocketStarted, 0)
			if !errors.Is(err, http.ErrServerClosed) {
				log.Error(log.APIServerMgr, err)
			}
		}
	}()
	return nil
}

// newWebsocketHub Creates a new websocket hub
func newWebsocketHub() *websocketHub {
	return &websocketHub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *websocketClient),
		Unregister: make(chan *websocketClient),
		Clients:    make(map[*websocketClient]bool),
	}
}

func (h *websocketHub) run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				log.Debugln(log.APIServerMgr, "websocket: disconnected client")
				delete(h.Clients, client)
				close(client.Send)
			}
		case message := <-h.Broadcast:
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					log.Debugln(log.APIServerMgr, "websocket: disconnected client")
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}

// SendWebsocketMessage sends a websocket event to the client
func (c *websocketClient) SendWebsocketMessage(evt interface{}) error {
	data, err := json.Marshal(evt)
	if err != nil {
		log.Errorf(log.APIServerMgr, "websocket: failed to send message: %s\n", err)
		return err
	}

	c.Send <- data
	return nil
}

func (c *websocketClient) read() {
	defer func() {
		c.Hub.Unregister <- c
		conErr := c.Conn.Close()
		if conErr != nil {
			log.Error(log.APIServerMgr, conErr)
		}
	}()

	for {
		msgType, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf(log.APIServerMgr, "websocket: client disconnected, err: %s\n", err)
			}
			break
		}

		if msgType == websocket.TextMessage {
			var evt WebsocketEvent
			err := json.Unmarshal(message, &evt)
			if err != nil {
				log.Errorf(log.APIServerMgr, "websocket: failed to decode JSON sent from client %s\n", err)
				continue
			}

			if evt.Event == "" {
				log.Warnln(log.APIServerMgr, "websocket: client sent a blank event, disconnecting")
				continue
			}

			dataJSON, err := json.Marshal(evt.Data)
			if err != nil {
				log.Errorln(log.APIServerMgr, "websocket: client sent data we couldn't JSON decode")
				break
			}

			req := strings.ToLower(evt.Event)
			log.Debugf(log.APIServerMgr, "websocket: request received: %s\n", req)

			result, ok := wsHandlers[req]
			if !ok {
				log.Debugln(log.APIServerMgr, "websocket: unsupported event")
				continue
			}

			if result.authRequired && !c.Authenticated {
				log.Warnf(log.APIServerMgr, "Websocket: request %s failed due to unauthenticated request on an authenticated API\n", evt.Event)
				err = c.SendWebsocketMessage(WebsocketEventResponse{Event: evt.Event, Error: "unauthorised request on authenticated API"})
				if err != nil {
					log.Error(log.APIServerMgr, err)
				}
				continue
			}

			err = result.handler(c, dataJSON)
			if err != nil {
				log.Errorf(log.APIServerMgr, "websocket: request %s failed. Error %s\n", evt.Event, err)
				continue
			}
		}
	}
}

func (c *websocketClient) write() {
	defer func() {
		err := c.Conn.Close()
		if err != nil {
			log.Error(log.APIServerMgr, err)
		}
	}()
	for {
		message, ok := <-c.Send
		if !ok {
			err := c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			if err != nil {
				log.Error(log.APIServerMgr, err)
			}
			log.Debugln(log.APIServerMgr, "websocket: hub closed the channel")
			return
		}

		w, err := c.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			log.Errorf(log.APIServerMgr, "websocket: failed to create new io.writeCloser: %s\n", err)
			return
		}
		_, err = w.Write(message)
		if err != nil {
			log.Error(log.APIServerMgr, err)
		}

		// Add queued chat messages to the current websocket message
		n := len(c.Send)
		for i := 0; i < n; i++ {
			_, err = w.Write(<-c.Send)
			if err != nil {
				log.Error(log.APIServerMgr, err)
			}
		}

		if err := w.Close(); err != nil {
			log.Errorf(log.APIServerMgr, "websocket: failed to close io.WriteCloser: %s\n", err)
			return
		}
	}
}

// StartWebsocketHandler starts the websocket hub and routine which
// handles clients
func StartWebsocketHandler() {
	if !wsHubStarted {
		wsHubStarted = true
		wsHub = newWebsocketHub()
		go wsHub.run()
	}
}

// BroadcastWebsocketMessage meow
func BroadcastWebsocketMessage(evt WebsocketEvent) error {
	if !wsHubStarted {
		return ErrWebsocketServiceNotRunning
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
func (m *apiServerManager) WebsocketClientHandler(w http.ResponseWriter, r *http.Request) {
	if !wsHubStarted {
		StartWebsocketHandler()
	}

	connectionLimit := m.remoteConfig.WebsocketRPC.ConnectionLimit
	numClients := len(wsHub.Clients)

	if numClients >= connectionLimit {
		log.Warnf(log.APIServerMgr,
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
	if m.remoteConfig.WebsocketRPC.AllowInsecureOrigin {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(log.APIServerMgr, err)
		return
	}

	client := &websocketClient{
		Hub:              wsHub,
		Conn:             conn,
		Send:             make(chan []byte, 1024),
		maxAuthFailures:  m.remoteConfig.WebsocketRPC.MaxAuthFailures,
		username:         m.remoteConfig.Username,
		password:         m.remoteConfig.Password,
		configPath:       m.gctConfigPath,
		exchangeManager:  m.exchangeManager,
		bot:              m.bot,
		portfolioManager: m.portfolioManager,
	}

	client.Hub.Register <- client
	log.Debugf(log.APIServerMgr,
		"websocket: client connected. Connected clients: %d. Limit %d.\n",
		numClients+1, connectionLimit)

	go client.read()
	go client.write()
}

func wsAuth(client *websocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "auth",
	}

	var auth WebsocketAuth
	err := json.Unmarshal(data.([]byte), &auth)
	if err != nil {
		wsResp.Error = err.Error()
		sendErr := client.SendWebsocketMessage(wsResp)
		if sendErr != nil {
			log.Error(log.APIServerMgr, sendErr)
		}
		return err
	}

	hashPW := crypto.HexEncodeToString(crypto.GetSHA256([]byte(client.password)))
	if auth.Username == client.username && auth.Password == hashPW {
		client.Authenticated = true
		wsResp.Data = WebsocketResponseSuccess
		log.Debugln(log.APIServerMgr,
			"websocket: client authenticated successfully")
		return client.SendWebsocketMessage(wsResp)
	}

	wsResp.Error = "invalid username/password"
	client.authFailures++
	sendErr := client.SendWebsocketMessage(wsResp)
	if sendErr != nil {
		log.Error(log.APIServerMgr, sendErr)
	}
	if client.authFailures >= client.maxAuthFailures {
		log.Debugf(log.APIServerMgr,
			"websocket: disconnecting client, maximum auth failures threshold reached (failures: %d limit: %d)\n",
			client.authFailures, client.maxAuthFailures)
		wsHub.Unregister <- client
		return nil
	}

	log.Debugf(log.APIServerMgr,
		"websocket: client sent wrong username/password (failures: %d limit: %d)\n",
		client.authFailures, client.maxAuthFailures)
	return nil
}

func wsGetConfig(client *websocketClient, _ interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetConfig",
		Data:  config.GetConfig(),
	}
	return client.SendWebsocketMessage(wsResp)
}

func wsSaveConfig(client *websocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "SaveConfig",
	}
	var respCfg config.Config
	err := json.Unmarshal(data.([]byte), &respCfg)
	if err != nil {
		wsResp.Error = err.Error()
		sendErr := client.SendWebsocketMessage(wsResp)
		if sendErr != nil {
			log.Error(log.APIServerMgr, sendErr)
		}
		return err
	}

	cfg := config.GetConfig()
	err = cfg.UpdateConfig(client.configPath, &respCfg, false)
	if err != nil {
		wsResp.Error = err.Error()
		sendErr := client.SendWebsocketMessage(wsResp)
		if sendErr != nil {
			log.Error(log.APIServerMgr, sendErr)
		}
		return err
	}

	err = client.bot.SetupExchanges()
	if err != nil {
		wsResp.Error = err.Error()
		sendErr := client.SendWebsocketMessage(wsResp)
		if sendErr != nil {
			log.Error(log.APIServerMgr, sendErr)
		}
		return err
	}
	wsResp.Data = WebsocketResponseSuccess
	return client.SendWebsocketMessage(wsResp)
}

func wsGetAccountInfo(client *websocketClient, data interface{}) error {
	accountInfo := getAllActiveAccounts(client.exchangeManager)
	wsResp := WebsocketEventResponse{
		Event: "GetAccountInfo",
		Data:  accountInfo,
	}
	return client.SendWebsocketMessage(wsResp)
}

func wsGetTickers(client *websocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetTickers",
	}
	wsResp.Data = getAllActiveTickers(client.exchangeManager)
	return client.SendWebsocketMessage(wsResp)
}

func wsGetTicker(client *websocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetTicker",
	}
	var tickerReq WebsocketOrderbookTickerRequest
	err := json.Unmarshal(data.([]byte), &tickerReq)
	if err != nil {
		wsResp.Error = err.Error()
		sendErr := client.SendWebsocketMessage(wsResp)
		if sendErr != nil {
			log.Error(log.APIServerMgr, sendErr)
		}
		return err
	}

	p, err := currency.NewPairFromString(tickerReq.Currency)
	if err != nil {
		return err
	}

	a, err := asset.New(tickerReq.AssetType)
	if err != nil {
		return err
	}

	exch := client.exchangeManager.GetExchangeByName(tickerReq.Exchange)
	if exch == nil {
		wsResp.Error = exchange.ErrNoExchangeFound.Error()
		sendErr := client.SendWebsocketMessage(wsResp)
		if sendErr != nil {
			log.Error(log.APIServerMgr, sendErr)
		}
		return err
	}
	tick, err := exch.FetchTicker(p, a)
	if err != nil {
		wsResp.Error = err.Error()
		sendErr := client.SendWebsocketMessage(wsResp)
		if sendErr != nil {
			log.Error(log.APIServerMgr, sendErr)
		}
		return err
	}
	wsResp.Data = tick
	return client.SendWebsocketMessage(wsResp)
}

func wsGetOrderbooks(client *websocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetOrderbooks",
	}
	wsResp.Data = getAllActiveOrderbooks(client.exchangeManager)
	return client.SendWebsocketMessage(wsResp)
}

func wsGetOrderbook(client *websocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetOrderbook",
	}
	var orderbookReq WebsocketOrderbookTickerRequest
	err := json.Unmarshal(data.([]byte), &orderbookReq)
	if err != nil {
		wsResp.Error = err.Error()
		sendErr := client.SendWebsocketMessage(wsResp)
		if sendErr != nil {
			log.Error(log.APIServerMgr, sendErr)
		}
		return err
	}

	p, err := currency.NewPairFromString(orderbookReq.Currency)
	if err != nil {
		return err
	}

	a, err := asset.New(orderbookReq.AssetType)
	if err != nil {
		return err
	}

	exch := client.exchangeManager.GetExchangeByName(orderbookReq.Exchange)
	if exch == nil {
		wsResp.Error = exchange.ErrNoExchangeFound.Error()
		sendErr := client.SendWebsocketMessage(wsResp)
		if sendErr != nil {
			log.Error(log.APIServerMgr, sendErr)
		}
		return err
	}
	ob, err := exch.FetchOrderbook(p, a)
	if err != nil {
		wsResp.Error = err.Error()
		sendErr := client.SendWebsocketMessage(wsResp)
		if sendErr != nil {
			log.Error(log.APIServerMgr, sendErr)
		}
		return err
	}
	wsResp.Data = ob
	return nil
}

func wsGetExchangeRates(client *websocketClient, data interface{}) error {
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

func wsGetPortfolio(client *websocketClient, data interface{}) error {
	wsResp := WebsocketEventResponse{
		Event: "GetPortfolio",
	}

	wsResp.Data = client.portfolioManager.GetPortfolioSummary()
	return client.SendWebsocketMessage(wsResp)
}
