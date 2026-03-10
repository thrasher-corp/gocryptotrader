package websocket

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// PingHandler container for ping handler settings
type PingHandler struct {
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

// UnhandledMessageWarning defines a container for unhandled message warnings
type UnhandledMessageWarning struct {
	Message string
}

// Reporter interface groups observability functionality over Websocket request latency.
type Reporter interface {
	Latency(name string, message []byte, t time.Duration)
}
