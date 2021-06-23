package engine

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// Const vars for websocket
const (
	WebsocketResponseSuccess = "OK"
	restIndexResponse        = "<html>GoCryptoTrader RESTful interface. For the web GUI, please visit the <a href=https://github.com/thrasher-corp/gocryptotrader/blob/master/web/README.md>web GUI readme.</a></html>"
	DeprecatedName           = "deprecated_rpc"
	WebsocketName            = "websocket_rpc"
)

var (
	wsHub              *websocketHub
	wsHubStarted       bool
	errNilRemoteConfig = errors.New("received nil remote config")
	errNilPProfConfig  = errors.New("received nil pprof config")
	errNilBot          = errors.New("received nil engine bot")
	errEmptyConfigPath = errors.New("received empty config path")
	errServerDisabled  = errors.New("server disabled")
	errAlreadyRunning  = errors.New("already running")
	// ErrWebsocketServiceNotRunning occurs when a message is sent to be broadcast via websocket
	// and its not running
	ErrWebsocketServiceNotRunning = errors.New("websocket service not started")
)

// apiServerManager holds all relevant fields to manage both REST and websocket
// api servers
type apiServerManager struct {
	restStarted            int32
	websocketStarted       int32
	restListenAddress      string
	websocketListenAddress string
	gctConfigPath          string
	restHTTPServer         *http.Server
	websocketHTTPServer    *http.Server
	wgRest                 sync.WaitGroup
	wgWebsocket            sync.WaitGroup

	restRouter      *mux.Router
	websocketRouter *mux.Router
	websocketHub    *websocketHub

	remoteConfig     *config.RemoteControlConfig
	pprofConfig      *config.Profiler
	exchangeManager  iExchangeManager
	bot              iBot
	portfolioManager iPortfolioManager
}

// websocketClient stores information related to the websocket client
type websocketClient struct {
	Hub              *websocketHub
	Conn             *websocket.Conn
	Authenticated    bool
	authFailures     int
	Send             chan []byte
	username         string
	password         string
	maxAuthFailures  int
	exchangeManager  iExchangeManager
	bot              iBot
	portfolioManager iPortfolioManager
	configPath       string
}

// websocketHub stores the data for managing websocket clients
type websocketHub struct {
	Clients    map[*websocketClient]bool
	Broadcast  chan []byte
	Register   chan *websocketClient
	Unregister chan *websocketClient
}

// WebsocketEvent is the struct used for websocket events
type WebsocketEvent struct {
	Exchange  string `json:"exchange,omitempty"`
	AssetType string `json:"assetType,omitempty"`
	Event     string
	Data      interface{}
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

// WebsocketAuth is a struct used for
type WebsocketAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Route is a sub type that holds the request routes
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// AllEnabledExchangeOrderbooks holds the enabled exchange orderbooks
type AllEnabledExchangeOrderbooks struct {
	Data []EnabledExchangeOrderbooks `json:"data"`
}

// EnabledExchangeOrderbooks is a sub type for singular exchanges and respective
// orderbooks
type EnabledExchangeOrderbooks struct {
	ExchangeName   string           `json:"exchangeName"`
	ExchangeValues []orderbook.Base `json:"exchangeValues"`
}

// AllEnabledExchangeCurrencies holds the enabled exchange currencies
type AllEnabledExchangeCurrencies struct {
	Data []EnabledExchangeCurrencies `json:"data"`
}

// EnabledExchangeCurrencies is a sub type for singular exchanges and respective
// currencies
type EnabledExchangeCurrencies struct {
	ExchangeName   string         `json:"exchangeName"`
	ExchangeValues []ticker.Price `json:"exchangeValues"`
}

// AllEnabledExchangeAccounts holds all enabled accounts info
type AllEnabledExchangeAccounts struct {
	Data []account.Holdings `json:"data"`
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

type wsCommandHandler struct {
	authRequired bool
	handler      func(client *websocketClient, data interface{}) error
}
