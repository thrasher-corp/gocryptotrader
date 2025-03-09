package websocket

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// Inspector is used to verify messages via SendMessageReturnResponsesWithInspection
// It inspects the []bytes websocket message and returns true if the message is the final message in a sequence of expected messages
type Inspector interface {
	IsFinal([]byte) bool
}

// Response defines generalised data from the stream connection
type Response struct {
	Type int
	Raw  []byte
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
