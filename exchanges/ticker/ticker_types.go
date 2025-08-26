package ticker

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// const values for the ticker package
const (
	errPairNotSet       = "ticker currency pair not set"
	errAssetTypeNotSet  = "ticker asset type not set"
	errTickerPriceIsNil = "ticker price is nil"
)

// Vars for the ticker package
var (
	service *Service
)

// Service holds ticker information for each individual exchange
type Service struct {
	Tickers  map[key.ExchangeAssetPair]*Ticker
	Exchange map[string]uuid.UUID
	mux      *dispatch.Mux
	mu       sync.Mutex
}

// Price struct stores the currency pair and pricing information
type Price struct {
	Last         float64       `json:"Last"`
	High         float64       `json:"High"`
	Low          float64       `json:"Low"`
	Bid          float64       `json:"Bid"`
	BidSize      float64       `json:"BidSize"`
	Ask          float64       `json:"Ask"`
	AskSize      float64       `json:"AskSize"`
	Volume       float64       `json:"Volume"`
	QuoteVolume  float64       `json:"QuoteVolume"`
	PriceATH     float64       `json:"PriceATH"`
	Open         float64       `json:"Open"`
	Close        float64       `json:"Close"`
	OpenInterest float64       `json:"OpenInterest"`
	MarkPrice    float64       `json:"MarkPrice"`
	IndexPrice   float64       `json:"IndexPrice"`
	Pair         currency.Pair `json:"Pair"`
	ExchangeName string        `json:"exchangeName"`
	AssetType    asset.Item    `json:"assetType"`
	LastUpdated  time.Time

	// Funding rate field variables
	FlashReturnRate       float64
	BidPeriod             float64
	AskPeriod             float64
	FlashReturnRateAmount float64
}

// Ticker struct holds the ticker information for a currency pair and type
type Ticker struct {
	Price
	Main  uuid.UUID
	Assoc []uuid.UUID
}
