package stats

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// items holds stat items
	items     []item
	statMutex sync.RWMutex
)

// item holds various fields for storing currency pair stats
type item struct {
	Exchange  string
	Pair      currency.Pair
	AssetType asset.Item
	Price     float64
	Volume    float64
}

// byPrice allows sorting by price
type byPrice []item

// byVolume allows sorting by volume
type byVolume []item
