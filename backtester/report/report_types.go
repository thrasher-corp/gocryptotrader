package report

import (
	"errors"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
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
	UpdateItem(*kline.Item)
	UseDarkMode(bool)
}

// Data holds all statistical information required to output detailed backtesting results
type Data struct {
	OriginalCandles       []*kline.Item
	EnhancedCandles       []DetailedKline
	Statistics            *statistics.Statistic
	Config                *config.Config
	TemplatePath          string
	OutputPath            string
	Warnings              []Warning
	UseDarkTheme          bool
	USDTotalsChart        []TotalsChart
	HoldingsOverTimeChart []TotalsChart
	Prettify              PrettyNumbers
}

// TotalsChart holds chart plot data
// to render charts in the report
type TotalsChart struct {
	Name       string
	DataPoints []ChartPlot
}

// ChartPlot holds value data
// for a chart
type ChartPlot struct {
	Value     float64
	UnixMilli int64
	Flag      string
}

// Warning holds any candle warnings
type Warning struct {
	Exchange string
	Asset    asset.Item
	Pair     currency.Pair
	Message  string
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
	UnixMilli      int64
	Open           float64
	High           float64
	Low            float64
	Close          float64
	Volume         float64
	VolumeColour   string
	MadeOrder      bool
	OrderDirection order.Side
	OrderAmount    decimal.Decimal
	Shape          string
	Text           string
	Position       string
	Colour         string
	PurchasePrice  float64
}

// PrettyNumbers is used for report rendering
// one cannot access packages when rendering data in a template
// this struct exists purely to help make numbers look pretty
type PrettyNumbers struct{}

// Decimal2 renders a decimal nicely with 2 decimal places
func (p *PrettyNumbers) Decimal2(d decimal.Decimal) string {
	return convert.DecimalToCommaSeparatedString(d, 2, ".", ",")
}

// Decimal8 renders a decimal nicely with 8 decimal places
func (p *PrettyNumbers) Decimal8(d decimal.Decimal) string {
	return convert.DecimalToCommaSeparatedString(d, 8, ".", ",")
}

// Decimal64 renders a decimal nicely with the idea not to limit decimal places
// and to make you nostalgic for Nintendo
func (p *PrettyNumbers) Decimal64(d decimal.Decimal) string {
	return convert.DecimalToCommaSeparatedString(d, 64, ".", ",")
}

// Float8 renders a float nicely with 8 decimal places
func (p *PrettyNumbers) Float8(f float64) string {
	return convert.FloatToCommaSeparatedString(f, 8, ".", ",")
}

// Int renders an int nicely
func (p *PrettyNumbers) Int(i int64) string {
	return convert.IntToCommaSeparatedString(i, ",")
}
