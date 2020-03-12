package ticker

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// const values for the ticker package
const (
	errExchangeNameUnset = "ticker exchange name not set"
	errPairNotSet        = "ticker currency pair not set"
	errAssetTypeNotSet   = "ticker asset type not set"
	errTickerPriceIsNil  = "ticker price is nil"
)

// Vars for the ticker package
var (
	service *Service
)

// Service holds ticker information for each individual exchange
type Service struct {
	Tickers  map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Ticker
	Exchange map[string]uuid.UUID
	mux      *dispatch.Mux
	sync.RWMutex
}

// Price struct stores the currency pair and pricing information
type Price struct {
	Last         float64       `json:"Last"`
	High         float64       `json:"High"`
	Low          float64       `json:"Low"`
	Bid          float64       `json:"Bid"`
	Ask          float64       `json:"Ask"`
	Volume       float64       `json:"Volume"`
	QuoteVolume  float64       `json:"QuoteVolume"`
	PriceATH     float64       `json:"PriceATH"`
	Open         float64       `json:"Open"`
	Close        float64       `json:"Close"`
	Pair         currency.Pair `json:"Pair"`
	ExchangeName string        `json:"exchangeName"`
	AssetType    asset.Item    `json:"assetType"`
	LastUpdated  time.Time
}

// Ticker struct holds the ticker information for a currency pair and type
type Ticker struct {
	Price
	Main  uuid.UUID
	Assoc []uuid.UUID
}
