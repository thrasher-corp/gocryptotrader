package trade

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// DefaultProcessorIntervalTime is the default timer
// to process queued trades and save them to the database
const DefaultProcessorIntervalTime = time.Second * 15

var (
	processor Processor
	// BufferProcessorIntervalTime is the interval to save trade buffer data to the database.
	// Change this by changing the runtime param `-tradeprocessinginterval=15s`
	BufferProcessorIntervalTime = DefaultProcessorIntervalTime
	// ErrNoTradesSupplied is returned when an attempt is made to process trades, but is an empty slice
	ErrNoTradesSupplied = errors.New("no trades supplied")
)

// Trade used to hold data and methods related to trade dissemination and
// storage
type Trade struct {
	dataHandler      *stream.Relay
	tradeFeedEnabled bool
}

// Data defines trade data
type Data struct {
	ID           uuid.UUID `json:"ID,omitempty"`
	TID          string
	Exchange     string
	CurrencyPair currency.Pair
	AssetType    asset.Item
	Side         order.Side
	Price        float64
	Amount       float64
	Timestamp    time.Time
}

// Processor used for processing trade data in batches
// and saving them to the database
type Processor struct {
	mutex                   sync.Mutex
	started                 atomic.Bool
	bufferProcessorInterval time.Duration
	buffer                  []Data
}
