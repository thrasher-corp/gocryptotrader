package event

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type Event struct {
	Exchange     string         `json:"exchange"`
	Time         time.Time      `json:"timestamp"`
	Interval     kline.Interval `json:"interval-size"`
	CurrencyPair currency.Pair  `json:"pair"`
	AssetType    asset.Item     `json:"asset"`
	Why          string         `json:"why"`
}
