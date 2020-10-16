package charts

import (
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestChartResult(t *testing.T) {
	ohlcvKline, _ := KlineItemToSeriesData(genOHCLVData(1))
	type fields struct {
		template     string
		TemplatePath string
		output       string
		outputPath   string
		Data         Output
		w            io.ReadWriter
		writeFile    bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			"valid",
			fields{
				output:       "basic.html",
				outputPath:   "Output",
				template:     "basic.tmpl",
				TemplatePath: "templates",
				writeFile:    false,
				Data: Output{
					Data: genIntervalData(1),
					Page: Page{
						Height: 1920,
						Width:  1080,
						Watermark:
						Watermark{
							watermark,
							true,
						},
					},
				},
			},
			basicTestData,
			false,
		},
		{
			"valid-timeseries-from-map",
			fields{
				output:       "candlestickseries.html",
				outputPath:   "Output",
				template:     "candlestickseries.tmpl",
				TemplatePath: "",
				writeFile:    false,
				Data: Output{
					Data: ohlcvKline,
					Page: Page{
						Height: 1920,
						Width:  1080,
					},
				},
			},
			candleStickTestData,
			false,
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			c := &Chart{
				Config: Config{
					Template:     tt.fields.template,
					TemplatePath: tt.fields.TemplatePath,
					File:         tt.fields.output,
					Path:         tt.fields.outputPath,
					w:            tt.fields.w,
					WriteFile:    tt.fields.writeFile,
				},
				Output: tt.fields.Data,
			}
			f, err := c.Generate()
			if err != nil {
				t.Fatal(err)
			}
			got, err := c.Result()
			if (err != nil) != tt.wantErr {
				t.Errorf("Result() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Result() got = %v, want %v", got, tt.want)
			}
			if f != nil {
				err = f.Close()
				if err != nil {
					t.Error("failed to close file manual removal may be required")
				}
				err = os.Remove(f.Name())
				if err != nil {
					t.Error("failed to remove file manual removal may be required")
				}
			}
		})
	}
}

func genIntervalData(totalCandles int) []IntervalData {
	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]IntervalData, totalCandles)
	out[0] = IntervalData{Timestamp: start.Format("2006-01-02"), Value: 0}
	for x := 1; x < totalCandles; x++ {
		out[x] = IntervalData{
			Timestamp: start.Add(time.Hour * 24 * time.Duration(x)).Format("2006-01-02"),
			Value:     out[x-1].Value + rand.Float64(), // nolint:gosec // no need to import crypo/rand for testing
		}
	}

	return out
}

func genOHCLVData(totalCandles int) *kline.Item {
	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)

	var outItem kline.Item
	outItem.Interval = kline.OneDay
	outItem.Asset = asset.Spot
	outItem.Pair = currency.NewPair(currency.BTC, currency.USDT)
	outItem.Exchange = "test"

	outItem.Candles = make([]kline.Candle, 365)
	outItem.Candles[0] = kline.Candle{
		Time:   start,
		Open:   0,
		High:   10 + rand.Float64(), // nolint:gosec // no need to import crypo/rand for testing
		Low:    10 + rand.Float64(), // nolint:gosec // no need to import crypo/rand for testing
		Close:  10 + rand.Float64(), // nolint:gosec // no need to import crypo/rand for testing
		Volume: 10,
	}

	for x := 1; x < totalCandles; x++ {
		outItem.Candles[x] = kline.Candle{
			Time:   start.Add(time.Hour * 24 * time.Duration(x)),
			Open:   outItem.Candles[x-1].Close,
			High:   outItem.Candles[x-1].Open + rand.Float64(), // nolint:gosec // no need to import crypo/rand for testing
			Low:    outItem.Candles[x-1].Open - rand.Float64(), // nolint:gosec // no need to import crypo/rand for testing
			Close:  outItem.Candles[x-1].Open + rand.Float64(), // nolint:gosec // no need to import crypo/rand for testing
			Volume: float64(rand.Int63n(150)),                  // nolint:gosec // no need to import crypo/rand for testing
		}
	}

	return &outItem
}

func TestChartGenerate(t *testing.T) {
	ohlcvKline, _ := KlineItemToSeriesData(genOHCLVData(365))
	type fields struct {
		template     string
		TemplatePath string
		output       string
		OutputPath   string
		Data         Output
		w            io.ReadWriter
		WriteFile    bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantOut *os.File
		wantErr bool
	}{
		{
			"basic",
			fields{
				output:     "basic.html",
				OutputPath: "Output",
				template:   "basic.tmpl",
				WriteFile:  true,
				Data: Output{
					Data: genIntervalData(365),
				},
			},
			&os.File{},
			false,
		},
		{
			"basic-invalid",
			fields{
				output:       "basic.html",
				OutputPath:   "Output",
				template:     "basic.tmpl",
				TemplatePath: filepath.Join("generate"),
				WriteFile:    true,
				Data: Output{
					Data: genIntervalData(365),
				},
			},
			&os.File{},
			true,
		},
		{
			"timeseries",
			fields{
				output:       "candlestickseries.html",
				OutputPath:   "Output",
				template:     "candlestickseries.tmpl",
				TemplatePath: "templates",
				WriteFile:    true,
				Data: Output{
					Data: ohlcvKline,
				},
			},
			nil,
			false,
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			c := &Chart{
				Config: Config{
					Template:     tt.fields.template,
					TemplatePath: tt.fields.TemplatePath,
					File:         tt.fields.output,
					Path:         tt.fields.OutputPath,
					w:            tt.fields.w,
					WriteFile:    tt.fields.WriteFile,
				},
				Output: tt.fields.Data,
			}
			_, err := c.Generate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// if f != nil {
			// 	_ = f.Close()
			// 	_ = os.Remove(f.Name())
			// }
		})
	}
}

func TestChart_ToFile(t *testing.T) {
	tests := []struct {
		name string
		want *Chart
	}{
		{
			"valid",
			&Chart{
				Config: Config{
					WriteFile: true,
				},
			},
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			c := &Chart{}
			if got := c.ToFile(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKlineItemToSeriesData(t *testing.T) {
	type args struct {
		item *kline.Item
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"valid",
			args{
				genOHCLVData(1),
			},
			false,
		},
		{
			"valid",
			args{
				&kline.Item{},
			},
			true,
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			_, err := KlineItemToSeriesData(tt.args.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("KlineItemToSeriesData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestTestTestTest(t *testing.T) {
	c := Chart{
		Config: Config{
			Path:         "output",
			File:         "basic.html",
			Template:     "basic.tmpl",
			TemplatePath: "templates",
			WriteFile:    true,
		},
		Output: Output{
			Page: Page{
				Width: 1920,
				Height: 1080,
				Watermark:
					Watermark{
						watermark,
						false,
				},
			},
		},
	}
	var err error
	c.Output.Data = genIntervalData(365)//KlineItemToSeriesData(genOHCLVData(365))
	_, err = c.Generate()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNew(t *testing.T) {
	type args struct {
		name     string
		template string
		outPath  string
	}
	tests := []struct {
		name      string
		args      args
		wantChart *Chart
		wantErr   bool
	}{
		{
			"basic",
			args{
				name:     "basic",
				template: "basic",
				outPath:  "Output",
			},
			&Chart{
				Config: Config{
					Template: "basic.tmpl",
					File:     "basic",
					Path:     "Output",
				},
			},
			false,
		},
		{
			"candlestickseries",
			args{
				name:     "candlestickseries",
				template: "candlestickseries",
				outPath:  "Output",
			},
			&Chart{
				Config: Config{
					Template: "candlestickseries.tmpl",
					File:     "candlestickseries",
					Path:     "Output",
				},
			},
			false,
		},
		{
			"candlestickseries-markers",
			args{
				name:     "candlestickseries-markers",
				template: "candlestickseries-markers",
				outPath:  "",
			},
			&Chart{
				Config: Config{
					Template: "candlestickseries-markers.tmpl",
					File:     "candlestickseries-markers",
					Path:     "Output",
				},
			},
			false,
		},
		{
			"invalid",
			args{
				name:     "invalid",
				template: "invalid",
				outPath:  "Output",
			},
			nil,
			true,
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			gotChart, err := New(tt.args.name, tt.args.template, tt.args.outPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotChart, tt.wantChart) {
				t.Errorf("New() gotChart = %v, want %v", gotChart, tt.wantChart)
			}
		})
	}
}
