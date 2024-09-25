package stream

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Connection defines a streaming services connection
type Connection interface {
	Dial(*websocket.Dialer, http.Header) error
	ReadMessage() Response
	SetupPingHandler(request.EndpointLimit, PingHandler)
	// GenerateMessageID generates a message ID for the individual connection. If a bespoke function is set
	// (by using SetupNewConnection) it will use that, otherwise it will use the defaultGenerateMessageID function
	// defined in websocket_connection.go.
	GenerateMessageID(highPrecision bool) int64
	// SendMessageReturnResponse will send a WS message to the connection and wait for response
	SendMessageReturnResponse(ctx context.Context, epl request.EndpointLimit, signature any, request any) ([]byte, error)
	// SendMessageReturnResponses will send a WS message to the connection and wait for N responses
	SendMessageReturnResponses(ctx context.Context, epl request.EndpointLimit, signature any, request any, expected int) ([][]byte, error)
	// SendRawMessage sends a message over the connection without JSON encoding it
	SendRawMessage(ctx context.Context, epl request.EndpointLimit, messageType int, message []byte) error
	// SendJSONMessage sends a JSON encoded message over the connection
	SendJSONMessage(ctx context.Context, epl request.EndpointLimit, payload any) error
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

// ConnectionSetup defines variables for an individual stream connection
type ConnectionSetup struct {
	ResponseCheckTimeout    time.Duration
	ResponseMaxLimit        time.Duration
	RateLimit               *request.RateLimiterWithWeight
	URL                     string
	Authenticated           bool
	ConnectionLevelReporter Reporter
	// BespokeGenerateMessageID is a function that returns a unique message ID.
	// This is useful for when an exchange connection requires a unique or
	// structured message ID for each message sent.
	BespokeGenerateMessageID func(highPrecision bool) int64
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

// Reporter interface groups observability functionality over
// Websocket request latency.
type Reporter interface {
	Latency(name string, message []byte, t time.Duration)
}
