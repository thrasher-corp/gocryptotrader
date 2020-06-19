package gct

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/ta/indicators"
)

var (
	atrPayload         = &indicators.ATR{Array: oneElement}
	bbandsPayload      = &indicators.BBands{Array: threeElement}
	emaPayload         = &indicators.EMA{Array: oneElement}
	macdPayload        = &indicators.MACD{Array: threeElement}
	mfiPayload         = &indicators.MFI{Array: oneElement}
	obvPayload         = &indicators.OBV{Array: oneElement}
	rsiPayload         = &indicators.RSI{Array: oneElement}
	smaPayload         = &indicators.SMA{Array: oneElement}
	correlationPayload = &indicators.Correlation{Array: oneElement}
	ohlcPayload        = &OHLCV{Map: ohlcdata}
	unhandled          = &objects.Array{}

	oneElement = objects.Array{
		Value: []objects.Object{
			&objects.Float{Value: 1},
			&objects.Float{Value: 2},
			&objects.Float{Value: 3},
			&objects.Float{Value: 4},
			&objects.Float{Value: 5},
		},
	}

	threeElement = objects.Array{
		Value: []objects.Object{
			&objects.Array{
				Value: []objects.Object{
					&objects.Float{Value: 11},
					&objects.Float{Value: 12},
					&objects.Float{Value: 13},
				},
			},
			&objects.Array{
				Value: []objects.Object{
					&objects.Float{Value: 21},
					&objects.Float{Value: 22},
					&objects.Float{Value: 23},
				},
			},
			&objects.Array{
				Value: []objects.Object{
					&objects.Float{Value: 31},
					&objects.Float{Value: 32},
					&objects.Float{Value: 33},
				},
			},
			&objects.Array{
				Value: []objects.Object{
					&objects.Float{Value: 41},
					&objects.Float{Value: 42},
					&objects.Float{Value: 43},
				},
			},
			&objects.Array{
				Value: []objects.Object{
					&objects.Float{Value: 51},
					&objects.Float{Value: 52},
					&objects.Float{Value: 53},
				},
			},
		},
	}

	ohlcv = []objects.Object{
		&objects.Time{Value: time.Now()},
		&objects.Float{Value: 100},
		&objects.Float{Value: 100},
		&objects.Float{Value: 100},
		&objects.Float{Value: 100},
		&objects.Float{Value: 1},
	}

	ohlcdata = objects.Map{
		Value: map[string]objects.Object{
			"exchange":  &objects.String{Value: "exchange"},
			"pair":      &objects.String{Value: "BTC-USD"},
			"asset":     &objects.String{Value: asset.Spot.String()},
			"intervals": &objects.String{Value: time.Minute.String()},
			"candles": &objects.Array{
				Value: []objects.Object{
					&objects.Array{
						Value: ohlcv,
					},
					&objects.Array{
						Value: ohlcv,
					},
					&objects.Array{
						Value: ohlcv,
					},
					&objects.Array{
						Value: ohlcv,
					},
					&objects.Array{
						Value: ohlcv,
					},
				},
			},
		},
	}
)

func TestCommonWriteToCSV(t *testing.T) {
	t.Parallel()

	OutputDir = filepath.Join(os.TempDir(), "script-temp")
	defer func() {
		err := os.RemoveAll(OutputDir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	_, err := WriteAsCSV()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	_, err = WriteAsCSV(nil)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	_, err = WriteAsCSV(&objects.String{Value: "something.txt"})
	if err == nil {
		t.Fatal(err)
	}

	_, err = WriteAsCSV(&objects.String{Value: "something.txt"},
		&objects.String{Value: "extra string"})
	if err == nil {
		t.Fatal(err)
	}

	_, err = WriteAsCSV(&objects.String{Value: "script-temp.csv"}, unhandled)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	_, err = WriteAsCSV(&objects.String{Value: "script-temp.csv"},
		atrPayload,
		bbandsPayload,
		emaPayload,
		macdPayload,
		mfiPayload,
		obvPayload,
		rsiPayload,
		smaPayload,
		correlationPayload,
		ohlcPayload)
	if err != nil {
		t.Fatal(err)
	}

	_, err = WriteAsCSV(atrPayload)
	if err == nil {
		t.Fatal(err)
	}

	_, err = WriteAsCSV(&objects.String{Value: "test.gct-script-temp2"},
		atrPayload)
	if err != nil {
		t.Fatal(err)
	}
}
