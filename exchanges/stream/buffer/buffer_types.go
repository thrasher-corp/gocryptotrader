package buffer

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// timerDefault defines the amount of time between alerting the sync manager of
// an update.
var timerDefault = time.Second * 10

// Orderbook defines a local cache of orderbooks for amending, appending
// and deleting changes and updates the main store for a stream
type Orderbook struct {
	ob                    map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder
	obBufferLimit         int
	bufferEnabled         bool
	sortBuffer            bool
	sortBufferByUpdateIDs bool // When timestamps aren't provided, an id can help sort
	updateEntriesByID     bool // Use the update IDs to match ob entries
	exchangeName          string
	dataHandler           chan interface{}
	verbose               bool
	m                     sync.Mutex
}

// orderbookHolder defines a store of pending updates and a pointer to the
// orderbook depth
type orderbookHolder struct {
	ob     *orderbook.Depth
	buffer *[]Update
	// Reduces the amount of outbound alerts to the data handler for example
	// coinbasepro can have up too 100 updates per second introducing overhead.
	// The sync agent only requires an alert every 15 seconds for a specific
	// currency.
	ticker *time.Ticker
}

// Update stores orderbook updates and dictates what features to use when processing
type Update struct {
	UpdateID   int64 // Used when no time is provided
	UpdateTime time.Time
	Asset      asset.Item
	Action
	Bids []orderbook.Item
	Asks []orderbook.Item
	Pair currency.Pair

	// Determines if there is a max depth of orderbooks and after an append we
	// should remove any items that are outside of this scope. Kraken is the
	// only exchange utilising this field.
	MaxDepth int
}

// Action defines a set of differing states required to implement an incoming
// orderbook update used in conjunction with UpdateEntriesByID
type Action string

const (
	// Amend applies amount adjustment by ID
	Amend Action = "update"
	// Delete removes price level from book by ID
	Delete Action = "delete"
	// Insert adds price level to book
	Insert Action = "insert"
	// UpdateInsert on conflict applies amount adjustment or appends new amount
	// to book
	UpdateInsert Action = "update/insert"
)
