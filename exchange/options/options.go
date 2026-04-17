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
	LastUpdated           time.Time
	Delta                 float64
	Gamma                 float64
	Vega                  float64
	Theta                 float64
	Rho                   float64
	BidImpliedVolatility  float64
	AskImpliedVolatility  float64
	MarkImpliedVolatility float64
}
