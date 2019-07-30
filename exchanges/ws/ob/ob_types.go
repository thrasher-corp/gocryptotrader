package ob

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
	orderbookBuffer map[currency.Pair]map[string][]BufferUpdate
	lastUpdated     time.Time
	m               sync.Mutex
}

// BufferUpdate contains cool info on how to update the websocket orderbook yeah man
type BufferUpdate struct {
	OrderByIDs   bool  // When timestamps arent provided, an id can help sort
	UseUpdateIDs bool  // Use the update IDs to match ob entries
	UpdateID     int64 // Used when no time is provided
	Updated      time.Time
	ExchangeName string
	AssetType    string
	Action       string
	Bids         []orderbook.Item
	Asks         []orderbook.Item
	CurrencyPair currency.Pair
}
