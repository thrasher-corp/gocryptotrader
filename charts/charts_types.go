package charts

import "io"

const (
	watermark         = "GoCryptoTrader"
	tvScriptName      = "lightweight-charts.standalone.production.js"
	chartjsScriptName = "Chart.bundle.min.js"
)

var tempByte []byte

// Chart configuration options
type Chart struct {
	Config
	Output
}

// Config handles chart configurable options
type Config struct {
	Template     string
	TemplatePath string
	File         string
	Path         string

	w         io.ReadWriter
	WriteFile bool
}

// Output holds page related configuration data that is passed to Template generation
type Output struct {
	Page     Page
	Exchange string
	Pair     string
	Data     interface{}
}

type Page struct {
	PageTitle string
	Watermark
	Width     float64
	Height    float64
}

type Watermark struct {
	Name string
	Visible bool
}

// IntervalData is used to store basic chart data
type IntervalData struct {
	Timestamp string
	Value     float64
}

// AdvancedIntervalData is used to store basic chart data
type AdvancedIntervalData struct {
	Timestamp string
	Value     float64
	Amount    float64
	Direction string
}

// SeriesData is used to store time series (OHLC) data
type SeriesData struct {
	Timestamp string
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}
