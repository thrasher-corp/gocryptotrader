package charts

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func New(name, template, outputpath string) (chart Chart) {
	switch template {
	case "basic":
		chart.template = "basic.tmpl"

	}
	chart.output = name
	if outputpath != "" {
		chart.OutputPath = "output"
	} else {
		chart.OutputPath = outputpath
	}
	return chart
}

func (c *Chart) Generate() error {
	list := []string{
		filepath.Join("templates", c.template),
		filepath.Join("templates", "base.tmpl"),
	}

	tmpl, err := template.ParseFiles(list...)
	if err != nil {
	return err
	}

	if c.writeFile {
		wd, _ := os.Getwd()
		outPath := filepath.Join(wd, c.OutputPath)
		err := common.CreateDir(outPath)
		if err != nil {
			return err
		}
		f, err := os.Create(filepath.Join(outPath, c.output))
		defer func() {
			err = f.Close()
			if err != nil {
				fmt.Println(err)
			}
		}()
		c.w = f
	}
	err = tmpl.Execute(c.w, c.Data)
	if err != nil {
		return err
	}

	return nil
}

func (c *Chart) Result() ([]byte, error) {
	return ioutil.ReadAll(c.w)
}

func KlineItemToSeriesData(item kline.Item) ([]SeriesData, error) {
	if len(item.Candles) == 0 {
		return nil, errors.New("no candle data found")
	}

	out := make([]SeriesData, len(item.Candles))
	for x := range item.Candles {
		out[x] = SeriesData{
			Timestamp: item.Candles[x].Time.Format("2006-01-02"),
			Open: item.Candles[x].Open,
			High: item.Candles[x].High,
			Low: item.Candles[x].Low,
			Close: item.Candles[x].Close,
			Volume: item.Candles[x].Volume,
		}
	}
	return out, nil
}