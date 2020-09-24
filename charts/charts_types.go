package charts

import (
	"io"
)

type Chart struct {
	template   string
	output     string
	outputPath string

	Data      Data
	w         io.ReadWriter
	writeFile bool
}

type Data struct {
	PageTitle    string
	Pair         string
	Data 	interface{}
}

type IntervalData struct {
	Timestamp string
	Value     float64
}

type SeriesData struct {
	Timestamp string
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}