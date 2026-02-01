package buffer

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// Config defines the configuration variables for the websocket buffer; snapshot
// and incremental update orderbook processing.
type Config struct {
	// SortBuffer enables a websocket to sort incoming updates before processing.
	SortBuffer bool
	// SortBufferByUpdateIDs allows the sorting of the buffered updates by their
	// corresponding update IDs.
	SortBufferByUpdateIDs bool
}

// Orderbook defines a local cache of orderbooks for amending, appending
// and deleting changes and updates the main store for a stream
type Orderbook struct {
	ob                    map[key.PairAsset]*orderbookHolder
	obBufferLimit         int
	bufferEnabled         bool
	sortBuffer            bool
	sortBufferByUpdateIDs bool // When timestamps aren't provided, an id can help sort
	exchangeName          string
	dataHandler           *stream.Relay
	verbose               bool

	m sync.RWMutex
}

// orderbookHolder defines a store of pending updates and a pointer to the
// orderbook depth
type orderbookHolder struct {
	ob     *orderbook.Depth
	buffer []orderbook.Update
}
