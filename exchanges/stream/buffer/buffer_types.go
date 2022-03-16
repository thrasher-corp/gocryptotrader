package buffer

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// Config defines the configuration variables for the websocket buffer; snapshot
// and incremental update orderbook processing.
type Config struct {
	// SortBuffer enables a websocket to buffer incoming updates before
	// processing. NOTE: This is set by config.json under
	// "websocketBufferEnabled" for each individual exchange.
	SortBuffer bool
	// SortBufferByUpdateIDs allows the sorting of the buffered updates by their
	// corresponding update IDs. This is package defined.
	SortBufferByUpdateIDs bool
	// UpdateEntriesByID will match by IDs instead of price to perform the a set
	// of actions. e.g. update, delete, insert
	UpdateEntriesByID bool
	// updateIDProgression requires that the new update ID be greater than the
	// prior ID. This will skip processing and not error.
	UpdateIDProgression bool
	// checksum is a package defined checksum calculation for updated books.
	Checksum func(state *orderbook.Base, checksum uint32) error
}

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
	dataHandler           chan<- interface{}
	verbose               bool

	// updateIDProgression requires that the new update ID be greater than the
	// prior ID. This will skip processing and not error.
	updateIDProgression bool
	// checksum is a package defined checksum calculation for updated books.
	checksum func(state *orderbook.Base, checksum uint32) error

	publishPeriod time.Duration
	m             sync.Mutex
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
	ticker   *time.Ticker
	updateID int64
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

	// Checksum defines the expected value when the books have been verified
	Checksum uint32

	// UpdateIDProgression defines if the ID is used as counter and needs to be
	// greater than last ID.
	UpdateIDProgression bool

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
