package event

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type Event struct {
	Exchange     string
	Time         time.Time
	Interval     kline.Interval
	CurrencyPair currency.Pair
	AssetType    asset.Item
	MakerFee     float64
	TakerFee     float64
	FeeRate      float64
}
