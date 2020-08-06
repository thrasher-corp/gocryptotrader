package trade

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	buffer []Data
	candles []kline.Candle
)

// BasicData is not yet converted stuff, so its
// much cleaner to pass-by-value using base types
// I dont want any kind of pointers here to allow for
// easy memory reclamation and not needing to put so much
// on the stack
type BasicData struct {
	Timestamp    int64
	CurrencyPair string
	Delimiter string
	AssetType    asset.Item
	Exchange     string
	EventType    order.Type
	Price        float64
	Amount       float64
	Side         order.Side
}

// Data defines trade data
type Data struct {
	Timestamp    time.Time
	CurrencyPair currency.Pair
	AssetType    asset.Item
	Exchange     string
	EventType    order.Type
	Price        float64
	Amount       float64
	Side         order.Side
}

// Traderino is a holder of trades right now
type Traderino struct {
	mutex sync.Mutex
	shutdown chan struct{}
}

type ByDate []Data

func (b ByDate) Len() int {
	return len(b)
}

func (b ByDate) Less(i, j int) bool {
	return b[i].Timestamp.Before(b[j].Timestamp)
}

func (b ByDate) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
