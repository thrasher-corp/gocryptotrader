package stream

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Connection defines a streaming services connection
type Connection interface {
	Dial(*websocket.Dialer, http.Header) error
	ReadMessage() Response
	SendJSONMessage(interface{}) error
	SetupPingHandler(PingHandler)
	GenerateMessageID(highPrecision bool) int64
	SendMessageReturnResponse(signature interface{}, request interface{}) ([]byte, error)
	SendRawMessage(messageType int, message []byte) error
	SetURL(string)
	SetProxy(string)
	GetURL() string
	Shutdown() error
}

// Response defines generalised data from the stream connection
type Response struct {
	Type int
	Raw  []byte
}

// ChannelSubscription container for streaming subscriptions
type ChannelSubscription struct {
	Channel  string
	Currency currency.Pair
	Asset    asset.Item
	Params   map[string]interface{}
}

// ConnectionSetup defines variables for an individual stream connection
type ConnectionSetup struct {
	ResponseCheckTimeout time.Duration
	ResponseMaxLimit     time.Duration
	RateLimit            int64
	URL                  string
	Authenticated        bool
}

// PingHandler container for ping handler settings
type PingHandler struct {
	Websocket         bool
	UseGorillaHandler bool
	MessageType       int
	Message           []byte
	Delay             time.Duration
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

// UnhandledMessageWarning defines a container for unhandled message warnings
type UnhandledMessageWarning struct {
	Message string
}
