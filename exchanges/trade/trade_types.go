package trade

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	buffer []Data
	candles []kline.Candle
)

// Data defines trade data
type Data struct {
	ID uuid.UUID
	Timestamp    time.Time
	Exchange     string
	EventType    order.Type
	CurrencyPair currency.Pair
	AssetType    asset.Item
	Price        float64
	Amount       float64
	Side         order.Side
}

type CandleHolder struct {
	candle kline.Candle
	trades []Data
}

// Traderino is a holder of trades right now
type Traderino struct {
	mutex sync.Mutex
	shutdown chan struct{}
	Name string
	started    int32
	lastCandleTime time.Time
	previousCandles []CandleHolder
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

type ByDate2 []CandleHolder

func (b ByDate2) Len() int {
	return len(b)
}

func (b ByDate2) Less(i, j int) bool {
	return b[i].candle.Time.Before(b[j].candle.Time)
}

func (b ByDate2) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
