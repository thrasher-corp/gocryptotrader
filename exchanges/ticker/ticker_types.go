package ticker

import (
	"sync"
	"time"
	"uuid"

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
	Last                       float64
	LastSize                   float64
	VolumeWeightedAveragePrice float64
	High                       float64
	Low                        float64
	Bid                        float64
	BidSize                    float64
	Ask                        float64
	AskSize                    float64
	BaseVolume                 float64
	QuoteVolume                float64
	Open                       float64
	Open24Hour                 float64
	PercentChange24Hour        float64
	Close                      float64
	OpenInterest               float64
	OpenInterestValue          float64
	MarkPrice                  float64
	IndexPrice                 float64
	Pair                       currency.Pair
	ExchangeName               string
	AssetType                  asset.Item
	LastUpdated                time.Time
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
