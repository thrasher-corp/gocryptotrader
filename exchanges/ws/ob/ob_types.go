package ob

import (
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

// WebsocketOrderbookLocal defines a local cache of orderbooks for amending,
// appending and deleting changes and updates the main store in ob.go
type WebsocketOrderbookLocal struct {
	ob     map[currency.Pair]map[string]*orderbook.Base
	buffer map[currency.Pair]map[string][]WebsocketOrderbookUpdate
	m      sync.Mutex
}

// WebsocketOrderbookUpdate contains cool info on how to update the websocket ob yeah man
type WebsocketOrderbookUpdate struct {
	BufferEnabled         bool
	SortBuffer            bool
	SortBufferByUpdateIDs bool  // When timestamps arent provided, an id can help sort
	UpdateEntriesByID     bool  // Use the update IDs to match ob entries
	UpdateID              int64 // Used when no time is provided
	UpdateTime            time.Time
	ExchangeName          string
	AssetType             string
	Action                string // Used in conjunction with UpdateEntriesByID
	Bids                  []orderbook.Item
	Asks                  []orderbook.Item
	CurrencyPair          currency.Pair
}
