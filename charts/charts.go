package charts

import (
	"bytes"
	"errors"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

//go:generate go run generate/generate.go

// New returns a new chart instance
func New(name, template, outputpath string) (chart Chart) {
	switch template {
	case "basic":
		chart.template = "basic.tmpl"
	case "timeseries":
		chart.template = "timeseries.tmpl"
	case "timeseries-markets":
		chart.template = "timeseries-markers.tmpl"
	}
	chart.output = name
	if outputpath == "" {
		chart.OutputPath = "output"
	} else {
		chart.OutputPath = outputpath
	}
	return chart
}

// Generate chart output
func (c *Chart) Generate() (*os.File, error) {
	var list []string
	if c.TemplatePath == "yeah the original f" {
		baseTemplate, err := writeTemplate(templateList["base.tmpl"])
		if err != nil {
			return nil, err
		}

		data, ok := templateList[c.template]
		if !ok {
			return nil, errors.New("template: " + c.template + " not found")
		}
		templateFile, err := writeTemplate(data)
		if err != nil {
			return nil, err
		}
		list = []string{
			filepath.Join(templateFile.Name()),
			filepath.Join(baseTemplate.Name()),
		}
	} else {
		list = []string{
			filepath.Join(c.TemplatePath, c.template),
			filepath.Join(c.TemplatePath, "base.tmpl"),
		}
	}
	var out *os.File
	tmpl, err := template.ParseFiles(list...)
	if err != nil {
		return nil, err
	}

	if c.WriteFile {
		if filepath.Ext(c.output) != ".html" {
			c.output += ".html"
		}
		var createErr error
		f, createErr := os.Create(filepath.Join(c.OutputPath, c.output))
		if createErr != nil {
			return nil, err
		}
		defer func() {
			err = f.Close()
			if err != nil {
				log.Warnln(log.Global, err)
			}
		}()
		c.w = f
		out = f
	} else {
		c.w = bytes.NewBuffer(tempByte)
	}

	if c.Data.Height == 0 {
		c.Data.Height = 1024
	}
	if c.Data.Width == 0 {
		c.Data.Width = 768
	}

	err = tmpl.Execute(c.w, c.Data)
	if err != nil {
		return nil, err
	}

	return out, err
}

// ToFile sets WriteFile to true
// this allows chaining a Generate() call if you wish to write a file after creation of instance
func (c *Chart) ToFile() *Chart {
	c.WriteFile = true
	return c
}

// Result returns byte array copy of chart
func (c *Chart) Result() ([]byte, error) {
	return ioutil.ReadAll(c.w)
}

// KlineItemToSeriesData converts from a kline.Item to SeriesData slice
func KlineItemToSeriesData(item *kline.Item) ([]SeriesData, error) {
	if len(item.Candles) == 0 {
		return nil, errors.New("no candle data found")
	}

	out := make([]SeriesData, len(item.Candles))
	for x := range item.Candles {
		out[x] = SeriesData{
			Timestamp: item.Candles[x].Time.Format("2006-01-02"),
			Open:      item.Candles[x].Open,
			High:      item.Candles[x].High,
			Low:       item.Candles[x].Low,
			Close:     item.Candles[x].Close,
			Volume:    item.Candles[x].Volume,
		}
	}
	return out, nil
}
