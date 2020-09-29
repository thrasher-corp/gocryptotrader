package charts

import (
	"io"
)

// Chart configuration options
type Chart struct {
	template   string
	TemplatePath string
	output     string
	OutputPath string

	Data
	w         io.ReadWriter
	writeFile bool
}

// Data holds page related configuration data that is passed to template generation
type Data struct {
	PageTitle string
	size
	Pair      string
	Data      interface{}
}

type size struct {
	Width  float64
	Height float64
}

// IntervalData is used to store basic chart data
type IntervalData struct {

}

// AdvancedIntervalData is used to store basic chart data
type AdvancedIntervalData struct {
	Timestamp string
	Value     float64
	Amount 	  float64
	Direction string

}

// SeriesData is used to store timeseries (OHLC) data
type SeriesData struct {
	Timestamp string
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

var tempByte []byte