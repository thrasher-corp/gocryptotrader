package charts

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

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
	if outputpath != "" {
		chart.OutputPath = "output"
	} else {
		chart.OutputPath = outputpath
	}
	return chart
}

// Generate chart output
func (c *Chart) Generate() error {
	if c.TemplatePath == "" {
		c.TemplatePath = "templates"
	}
	list := []string{
		filepath.Join(c.TemplatePath, c.template),
		filepath.Join(c.TemplatePath, "base.tmpl"),
	}
	tmpl, err := template.ParseFiles(list...)
	if err != nil {
		return err
	}

	if c.WriteFile {
		wd, _ := os.Getwd()
		outPath := filepath.Join(wd, c.OutputPath)
		err := common.CreateDir(outPath)
		if err != nil {
			return err
		}
		if filepath.Ext(c.output) != ".html" {
			c.output += ".html"
		}
		f, err := os.Create(filepath.Join(outPath, c.output))
		defer func() {
			err = f.Close()
			if err != nil {
				fmt.Println(err)
			}
		}()
		c.w = f
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
		return err
	}

	return nil
}

func (c *Chart) ToFile() *Chart {
	c.WriteFile = true
	return c
}

// Result returns byte array copy of chart
func (c *Chart) Result() ([]byte, error) {
	if c.WriteFile {
		return []byte{}, errors.New("")
	}
	return ioutil.ReadAll(c.w)
}

// KlineItemToSeriesData converts from a kline.Item to SeriesData slice
func KlineItemToSeriesData(item kline.Item) ([]SeriesData, error) {
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
