package stream

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// // Manager defines functionality for different exchange streaming services and
// // related connections
// type Manager interface {
// 	// Connect() error
// 	// Disconnect() error
// 	// GenerateAuthSubscriptions()
// 	// GenerateMarketDataSubscriptions()
// 	// Subscribe()
// 	// UnSubscribe()
// 	// Refresh()
// 	// GetName() string
// 	// GetProxy() string
// 	Setup(*stream.WebsocketSetup) error
// 	SetupNewConnection(ConnectionSetup, bool) error
// 	SetupLocalOrderbook(cache.Config) error
// 	SubscribeToChannels([]ChannelSubscription)
// 	RemoveSubscribedChannels([]ChannelSubscription)
// 	GetSubscriptions() []ChannelSubscription
// 	SetProxyAddress(string) error
// 	IsEnabled() bool
// 	Wrapper
// }

// Connection defines a streaming services connection
type Connection interface {
	Dial(*websocket.Dialer, http.Header) error
	ReadMessage() (Response, error)
	SendJSONMessage(interface{}) error
	SetupPingHandler(PingHandler)
	IsIDWaitingForResponse(id int64) bool
	SetResponseIDAndData(id int64, data []byte)
	GenerateMessageID(useNano bool) int64
	SendMessageReturnResponse(id int64, request interface{}) ([]byte, error)
	SendRawMessage(messageType int, message []byte) error
}

// SubscriptionManager handles streaming subscription streaming
type SubscriptionManager interface{}

// Wrapper links websocket endpoint requests for account, positional, order
// information etc.
type Wrapper interface{}

// Response defines generalised data from the stream connection
type Response struct {
	Type int
	Raw  []byte
}

// ChannelSubscription container for streaming subscriptions
// Currently only a one at a time thing to avoid complexity
// TODO: SUB/UNSUB complete list
type ChannelSubscription struct {
	Channel    string
	Currency   currency.Pair
	Params     map[string]interface{}
	subscribed bool
}

// ConnectionSetup defines variables for an individual stream connection
type ConnectionSetup struct {
	ResponseCheckTimeout time.Duration
	ResponseMaxLimit     time.Duration
	RateLimit            int
	URL                  string
}

// PingHandler container for ping handler settings
type PingHandler struct {
	Websocket         bool
	UseGorillaHandler bool
	MessageType       int
	Message           []byte
	Delay             time.Duration
}

// TradeData defines trade data
type TradeData struct {
	Timestamp    time.Time
	CurrencyPair currency.Pair
	AssetType    asset.Item
	Exchange     string
	EventType    order.Type
	Price        float64
	Amount       float64
	Side         order.Side
}

// FundingData defines funding data
type FundingData struct {
	Timestamp    time.Time
	CurrencyPair currency.Pair
	AssetType    asset.Item
	Exchange     string
	Amount       float64
	Rate         float64
	Period       int64
	Side         order.Side
}

// KlineData defines kline feed
type KlineData struct {
	Timestamp  time.Time
	Pair       currency.Pair
	AssetType  asset.Item
	Exchange   string
	StartTime  time.Time
	CloseTime  time.Time
	Interval   string
	OpenPrice  float64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	Volume     float64
}

// WebsocketPositionUpdated reflects a change in orders/contracts on an exchange
type WebsocketPositionUpdated struct {
	Timestamp time.Time
	Pair      currency.Pair
	AssetType asset.Item
	Exchange  string
}
