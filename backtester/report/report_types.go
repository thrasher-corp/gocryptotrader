package report

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// lightweight charts can ony render 1100 candles
const maxChartLimit = 1099

type Handler interface {
	GenerateReport() error
	AddCandles(*kline.Item)
}

type Data struct {
	OriginalCandles []*kline.Item
	Candles         []DetailedKline
	Statistics      *statistics.Statistic
	Config          *config.Config
}

type DetailedKline struct {
	IsOverLimit bool
	Watermark   string
	Exchange    string
	Asset       asset.Item
	Pair        currency.Pair
	Interval    kline.Interval
	Candles     []DetailedCandle
}

type DetailedCandle struct {
	Time           int64
	Open           float64
	High           float64
	Low            float64
	Close          float64
	Volume         float64
	VolumeColour   string
	MadeOrder      bool
	OrderDirection order.Side
	OrderAmount    float64
	Shape          string
	Text           string
	Position       string
	Colour         string
	PurchasePrice  float64
}
