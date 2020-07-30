package buffer

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// Orderbook defines a local cache of orderbooks for amending, appending
// and deleting changes and updates the main store for a stream
type Orderbook struct {
	ob                    map[currency.Pair]map[asset.Item]*orderbook.Base
	buffer                map[currency.Pair]map[asset.Item][]*Update
	obBufferLimit         int
	bufferEnabled         bool
	sortBuffer            bool
	sortBufferByUpdateIDs bool // When timestamps aren't provided, an id can help sort
	updateEntriesByID     bool // Use the update IDs to match ob entries
	exchangeName          string
	dataHandler           chan interface{}
	m                     sync.Mutex
}

// Update stores orderbook updates and dictates what features to use when processing
type Update struct {
	UpdateID   int64 // Used when no time is provided
	UpdateTime time.Time
	Asset      asset.Item
	Action     string // Used in conjunction with UpdateEntriesByID
	Bids       []orderbook.Item
	Asks       []orderbook.Item
	Pair       currency.Pair
}
