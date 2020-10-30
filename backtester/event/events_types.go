package event

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type Event struct {
	Time         time.Time
	CurrencyPair currency.Pair
	AssetType    asset.Item
}
