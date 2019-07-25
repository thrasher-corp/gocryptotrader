package wshandler

import (
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

// IWSOrderbook defines what is required to manage orderbooks for websockets
type IWSOrderbook interface {
	ValidateWsOrderbookEntry() error
	AppendWsOrderbookEntry()
	UpdateWsOrderbook()
}

// WebsocketOrderbookLocal defines a local cache of orderbooks for amending,
// appending and deleting changes and updates the main store in orderbook.go
type WebsocketOrderbookLocal struct {
	orderbook       map[currency.Pair]map[string]*orderbook.Base
	orderbookBuffer map[currency.Pair]map[string][]orderbook.Base
	lastUpdated     time.Time
	m               sync.Mutex
}
