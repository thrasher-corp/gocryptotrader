package engine

import "github.com/gorilla/websocket"

// WebsocketClient stores information related to the websocket client
type WebsocketClient struct {
	Hub           *WebsocketHub
	Conn          *websocket.Conn
	Authenticated bool
	authFailures  int
	Send          chan []byte
}

// WebsocketHub stores the data for managing websocket clients
type WebsocketHub struct {
	Clients    map[*WebsocketClient]bool
	Broadcast  chan []byte
	Register   chan *WebsocketClient
	Unregister chan *WebsocketClient
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
