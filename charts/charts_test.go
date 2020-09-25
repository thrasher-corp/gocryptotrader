package charts

import (
	"io"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestChart_Generate(t *testing.T) {
	ohlcvKline, _  := KlineItemToSeriesData(genOHCLVData())
	type fields struct {
		template   string
		output     string
		outputPath string
		Data       Data
		w          io.ReadWriter
		writeFile  bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"basic",
			fields{
				output:     "basic.html",
				outputPath: "output",
				template:   "basic.tmpl",
				writeFile:  true,
				Data: Data{
					Data: genIntervalData(),
				},
			},
			false,
		},
		{
			"timeseries",
			fields{
				output:     "timeseries.html",
				outputPath: "output",
				template:   "timeseries.tmpl",
				writeFile:  true,
				Data: Data{
					Data: ohlcvKline,
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chart{
				template:   tt.fields.template,
				output:     tt.fields.output,
				outputPath: tt.fields.outputPath,
				Data:       tt.fields.Data,
				w:          tt.fields.w,
				writeFile:  tt.fields.writeFile,
			}
			if err := c.Generate(); (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChart_Result(t *testing.T) {
	type fields struct {
		template  string
		output    string
		Data      Data
		w         io.ReadWriter
		writeFile bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			"valid",
			fields{},
			[]byte{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chart{
				template:  tt.fields.template,
				output:    tt.fields.output,
				Data:      tt.fields.Data,
				w:         tt.fields.w,
				writeFile: tt.fields.writeFile,
			}
			got, err := c.Result()
			if (err != nil) != tt.wantErr {
				t.Errorf("Result() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Result() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewBasic(t *testing.T) {
	tests := []struct {
		name string
		want Chart
	}{
		{
			"basic",
			Chart{
				output:   "basic",
				template: "basic.tmpl",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.name, tt.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBasic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func genIntervalData() []IntervalData {
	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]IntervalData, 365)
	out[0] = IntervalData{ Timestamp: start.Format("2006-01-02"), Value: 0}
	for x := 1; x < 365; x++ {
		out[x] = IntervalData{
			Timestamp:   start.Add(time.Hour * 24 * time.Duration(x)).Format("2006-01-02"),
			Value:   out[x-1].Value + rand.Float64(),
		}
	}

	return out
}


func genOHCLVData() kline.Item {
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
		High:   10 + rand.Float64(),
		Low:    10 + rand.Float64(),
		Close:  10 + rand.Float64(),
		Volume: 10,
	}

	for x := 1; x < 365; x++ {
		outItem.Candles[x] = kline.Candle{
			Time:   start.Add(time.Hour * 24 * time.Duration(x)),
			Open:   outItem.Candles[x-1].Close,
			High:   outItem.Candles[x-1].Open + rand.Float64(),
			Low:    outItem.Candles[x-1].Open - rand.Float64(),
			Close:  outItem.Candles[x-1].Open + rand.Float64(),
			Volume: float64(rand.Int63n(150)),
		}
	}

	return outItem
}
