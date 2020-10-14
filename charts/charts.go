package charts

import (
	"bytes"
	"errors"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// New returns a new chart instance
func New(name, template, outPath string) (chart *Chart, err error) {
	chart = new(Chart)
	switch template {
	case "basic":
		chart.Template = "basic.tmpl"
	case "timeseries":
		chart.Template = "timeseries.tmpl"
	case "timeseries-markers":
		chart.Template = "timeseries-markers.tmpl"
	default:
		return nil, errors.New("invalid Template")
	}
	chart.Config.File = name
	if outPath == "" {
		chart.Path = "Output"
	} else {
		chart.Path = outPath
	}
	return chart, nil
}

// Generate chart Output
func (c *Chart) Generate() (*os.File, error) {
	var list []string
	if c.TemplatePath == "" {
		baseTemplate, err := writeTemplate(templateList["base.tmpl"])
		if err != nil {
			return nil, err
		}

		data, ok := templateList[c.Template]
		if !ok {
			return nil, errors.New("Template: " + c.Template + " not found")
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
			filepath.Join(c.TemplatePath, c.Template),
			filepath.Join(c.TemplatePath, "base.tmpl"),
		}
	}

	var out *os.File
	tmpl, err := template.ParseFiles(list...)
	if err != nil {
		return nil, err
	}

	if c.WriteFile {
		if filepath.Ext(c.Config.File) != ".html" {
			c.Config.File += ".html"
		}
		var createErr error
		f, createErr := os.Create(filepath.Join(c.Path, c.Config.File))
		if createErr != nil {
			return nil, createErr
		}
		createErr = c.writeJavascriptLibrary()
		if createErr != nil {
			log.Errorf(log.Global, "failed to write javascript library: %v", createErr)
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

	if c.Output.Page.Height == 0 {
		c.Output.Page.Height = 1024
	}
	if c.Output.Page.Width == 0 {
		c.Output.Page.Width = 768
	}

	err = tmpl.Execute(c.w, c.Output)
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

func (c *Chart) writeJavascriptLibrary() error {
	outfile := filepath.Join(c.Path, tvScriptName)
	if c.TemplatePath == "" {
		f, err := os.Create(outfile)
		if err != nil {
			return err
		}
		n, err := f.Write(templateList["lightweight-charts.standalone.production.js"])
		if err != nil {
			return err
		}
		if n != len(templateList["lightweight-charts.standalone.production.js"]) {
			return errors.New("write length mismatch")
		}
		return f.Close()
	}
	_, err := file.Copy(filepath.Join(c.TemplatePath, tvScriptName), outfile)
	return err
}
