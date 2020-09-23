package charts

import (
	"io"
	"time"
)

type Chart struct {
	template string
	output string

	Data data
	w io.ReadWriter
	writeFile bool
}

type Data struct {
	PageTitle string
	Pair string
	data []data
}

type data struct {
	Timestamp time.Time
	Value     float64
}
