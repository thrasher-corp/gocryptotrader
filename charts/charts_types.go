package charts

import "io"

const (
	watermark    = "GoCryptoTrader"
	tvScriptName = "lightweight-charts.standalone.production.js"
)

var tempByte []byte

// Chart configuration options
type Chart struct {
	Config
	Data
}

// Config handles chart configurable options
type Config struct {
	template     string
	TemplatePath string
	output       string
	OutputPath   string

	w         io.ReadWriter
	WriteFile bool
}

// Data holds page related configuration data that is passed to template generation
type Data struct {
	PageTitle string
	size
	Pair string
	Data interface{}
}

type size struct {
	Width  float64
	Height float64
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
