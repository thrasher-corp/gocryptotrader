package options

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Option is a normalized options greeks payload for websocket data handling.
type Option struct {
	ExchangeName string
	Pair         currency.Pair
	AssetType    asset.Item
	LastUpdated  time.Time
	Delta        float64
	Gamma        float64
	Vega         float64
	Theta        float64
	Rho          float64
	BidIV        float64
	AskIV        float64
	MarkIV       float64
}
