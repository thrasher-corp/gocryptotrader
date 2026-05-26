package options

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Greeks is a normalised options greeks payload for websocket data handling.
type Greeks struct {
	ExchangeName          string
	Pair                  currency.Pair
	AssetType             asset.Item
	InstrumentID          string
	LastUpdated           time.Time
	ExchangeTimestamp     time.Time
	ReceivedAt            time.Time
	Sequence              int64
	Delta                 float64
	Gamma                 float64
	Vega                  float64
	Theta                 float64
	Rho                   float64
	Bid                   float64
	Ask                   float64
	BidSize               float64
	AskSize               float64
	MarkPrice             float64
	IndexPrice            float64
	UnderlyingPrice       float64
	LastTradePrice        float64
	LastTradeSize         float64
	LastTradeAt           time.Time
	OpenInterest          float64
	Volume24h             float64
	BidImpliedVolatility  float64
	AskImpliedVolatility  float64
	MarkImpliedVolatility float64
}
