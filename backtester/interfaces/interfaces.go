package interfaces

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// EventHandler interface implements required GetTime() & Pair() return
type EventHandler interface {
	IsEvent() bool
	GetTime() time.Time
	Pair() currency.Pair
	GetExchange() string
	GetInterval() kline.Interval
	GetAssetType() asset.Item

	GetWhy() string
	AppendWhy(string)
}

// DataHandler interface used for loading and interacting with Data
type DataEventHandler interface {
	EventHandler
	DataType() DataType
	Price() float64
}

type DataType uint8

// Directioner dictates the side of an order
type Directioner interface {
	SetDirection(side order.Side)
	GetDirection() order.Side
}
