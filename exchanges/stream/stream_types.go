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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// Connection defines a streaming services connection
type Connection interface {
	Dial(*websocket.Dialer, http.Header) error
	DialContext(context.Context, *websocket.Dialer, http.Header) error
	ReadMessage() Response
	SetupPingHandler(request.EndpointLimit, PingHandler)
	// GenerateMessageID generates a message ID for the individual connection. If a bespoke function is set
	// (by using SetupNewConnection) it will use that, otherwise it will use the defaultGenerateMessageID function
	// defined in websocket_connection.go.
	GenerateMessageID(highPrecision bool) int64
	// SendMessageReturnResponse will send a WS message to the connection and wait for response
	SendMessageReturnResponse(ctx context.Context, epl request.EndpointLimit, signature any, request any) ([]byte, error)
	// SendMessageReturnResponses will send a WS message to the connection and wait for N responses
	SendMessageReturnResponses(ctx context.Context, epl request.EndpointLimit, signature any, request any, expected int, isFinalMessage ...Inspector) ([][]byte, error)
	// SendRawMessage sends a message over the connection without JSON encoding it
	SendRawMessage(ctx context.Context, epl request.EndpointLimit, messageType int, message []byte) error
	// SendJSONMessage sends a JSON encoded message over the connection
	SendJSONMessage(ctx context.Context, epl request.EndpointLimit, payload any) error
	SetURL(string)
	SetProxy(string)
	GetURL() string
	Shutdown() error
}

// Inspector is a hook that allows for custom message inspection
type Inspector func([]byte) bool

// Response defines generalised data from the stream connection
type Response struct {
	Type int
	Raw  []byte
}

// ConnectionSetup defines variables for an individual stream connection
type ConnectionSetup struct {
	ResponseCheckTimeout time.Duration
	ResponseMaxLimit     time.Duration
	RateLimit            *request.RateLimiterWithWeight
	// Authenticated indicates if the connection can be authenticated, this is not used for multi-connection.
	Authenticated           bool
	ConnectionLevelReporter Reporter

	// URL defines the websocket server URL to connect to
	URL string
	// Connector is the function that will be called to connect to the
	// exchange's websocket server. This will be called once when the stream
	// service is started. Any bespoke connection logic should be handled here.
	Connector func(ctx context.Context, conn Connection) error
	// GenerateSubscriptions is a function that will be called to generate a
	// list of subscriptions to be made to the exchange's websocket server.
	GenerateSubscriptions func() (subscription.List, error)
	// Subscriber is a function that will be called to send subscription
	// messages based on the exchange's websocket server requirements to
	// subscribe to specific channels.
	Subscriber func(ctx context.Context, conn Connection, sub subscription.List) error
	// Unsubscriber is a function that will be called to send unsubscription
	// messages based on the exchange's websocket server requirements to
	// unsubscribe from specific channels. NOTE: IF THE FEATURE IS ENABLED.
	Unsubscriber func(ctx context.Context, conn Connection, unsub subscription.List) error
	// Handler defines the function that will be called when a message is
	// received from the exchange's websocket server. This function should
	// handle the incoming message and pass it to the appropriate data handler.
	Handler func(ctx context.Context, incoming []byte) error
	// BespokeGenerateMessageID is a function that returns a unique message ID.
	// This is useful for when an exchange connection requires a unique or
	// structured message ID for each message sent.
	BespokeGenerateMessageID func(highPrecision bool) int64
	// Authenticate is a function that will be called to authenticate the
	// connection to the exchange's websocket server. This function should
	// handle the authentication process and return an error if the
	// authentication fails.
	Authenticate func(ctx context.Context, conn Connection) error
	// OutboundRequestSignature is any type that will match outbound
	// requests to this specific connection. This could be an asset type
	// `asset.Spot`, a string type denoting the individual URL, an
	// authenticated or unauthenticated string or a mixture of these.
	OutboundRequestSignature any
	// ConnectionDoesNotRequireSubscriptions is for when this is a dedicated connection for only outbound requests.
	// Subscription generation and handling is not required. Please use dummy functions `SubscriberNotRequired` &
	// `SubscriptionGenerationNotRequired` for fields.
	ConnectionDoesNotRequireSubscriptions bool
}

// SubscriberNotRequired is a default subscriber function that can be used when a connection does not require
// subscription functionality.
func SubscriberNotRequired(ctx context.Context, conn Connection, sub subscription.List) error {
	return nil
}

// SubscriptionGenerationNotRequired is a default subscription generation function that can be used when a connection
// does not require subscription functionality. This returns a single dummy subscription to spawn a connection.
func SubscriptionGenerationNotRequired() (subscription.List, error) {
	return nil, nil
}

// ConnectionWrapper contains the connection setup details to be used when
// attempting a new connection. It also contains the subscriptions that are
// associated with the specific connection.
type ConnectionWrapper struct {
	// Setup contains the connection setup details
	Setup *ConnectionSetup
	// Subscriptions contains the subscriptions that are associated with the
	// specific connection(s)
	Subscriptions *subscription.Store
	// Connection contains the active connection based off the connection
	// details above.
	Connection Connection // TODO: Upgrade to slice of connections.
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
