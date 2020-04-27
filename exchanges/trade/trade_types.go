package trade

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// History holds exchange history data
type History struct {
	Timestamp       time.Time
	TID             string
	Price           float64
	Amount          float64
	Exchange        string
	Side            order.Side
	Fee             float64
	Description     string
	Asset           asset.Item
	AggregatedTrade bool
	Maker           bool
	FirstTradeID    string
	LastTradeID     string
	FillType        string
	Type            order.Type
}

// HistoryRequest defines params for fetching of the exchange trade history
type HistoryRequest struct {
	Pair           currency.Pair
	Asset          asset.Item
	TimestampStart time.Time // Some exchanges need a starting timestamp
	TradeID        string    // Some exchanges require a trade ID
}
