package report

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Handler interface {
	GenerateReport() error
}

type Data struct {
	OriginalCandles []*kline.Item
	Candles         []DetailedKline
	Statistics      *statistics.Statistic
}

type DetailedKline struct {
	Exchange string
	Asset    asset.Item
	Pair     currency.Pair
	Interval kline.Interval
	Candles  []DetailedCandle
}

type DetailedCandle struct {
	Time           time.Time
	Open           float64
	High           float64
	Low            float64
	Close          float64
	Volume         float64
	MadeOrder      bool
	OrderDirection order.Side
	OrderAmount    float64
	PurchasePrice  float64
}
