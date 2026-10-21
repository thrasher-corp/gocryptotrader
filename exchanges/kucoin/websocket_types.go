package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WebsocketTokenResponse is the token and the servers a Classic websocket connection is made with
type WebsocketTokenResponse struct {
	Token           string           `json:"token"`
	InstanceServers []InstanceServer `json:"instanceServers"`
}

// InstanceServer is a Classic websocket server
type InstanceServer struct {
	Endpoint     string `json:"endpoint"`
	Encrypt      bool   `json:"encrypt"`
	Protocol     string `json:"protocol"`
	PingInterval uint64 `json:"pingInterval"` // Milliseconds
	PingTimeout  uint64 `json:"pingTimeout"`  // Milliseconds
}

// WebsocketPingMessage is the heartbeat the client sends on a Classic websocket connection
type WebsocketPingMessage struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// WebsocketSubscriptionRequest is a subscribe or unsubscribe request
type WebsocketSubscriptionRequest struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Topic          string `json:"topic"`
	PrivateChannel bool   `json:"privateChannel"`
	Response       bool   `json:"response"`
}

// WebsocketSubscriptionResponse is the ack or error a subscribe or unsubscribe request is answered with
type WebsocketSubscriptionResponse struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Code uint64          `json:"code"`
	Data json.RawMessage `json:"data"` // The error message on an error
}

// WebsocketMessage is the envelope of every frame a Classic websocket connection receives: welcome, pong, ack, error and
// topic pushes
type WebsocketMessage struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Topic          string          `json:"topic"`
	Subject        string          `json:"subject"`
	UserID         string          `json:"userId"`
	ChannelType    string          `json:"channelType"`
	SequenceNumber uint64          `json:"sn"`
	Code           uint64          `json:"code"`
	Timestamp      types.Time      `json:"timestamp"` // Pong server time
	Data           json.RawMessage `json:"data"`
}
