package report

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// lightweight charts can ony render 1100 candles
const maxChartLimit = 1100

var (
	errNoCandles       = errors.New("no candles to enhance")
	errStatisticsUnset = errors.New("unable to proceed with unset Statistics property")
)

// Handler contains all functions required to generate statistical reporting for backtesting results
type Handler interface {
	GenerateReport() error
	AddKlineItem(*kline.Item)
}

// Data holds all statistical information required to output detailed backtesting results
type Data struct {
	OriginalCandles []*kline.Item
	EnhancedCandles []DetailedKline
	Statistics      *statistics.Statistic
	Config          *config.Config
	TemplatePath    string
	OutputPath      string
}

// DetailedKline enhances kline details for the purpose of rich reporting results
type DetailedKline struct {
	IsOverLimit bool
	Watermark   string
	Exchange    string
	Asset       asset.Item
	Pair        currency.Pair
	Interval    kline.Interval
	Candles     []DetailedCandle
}

// DetailedCandle contains extra details to enable rich reporting results
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
