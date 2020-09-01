package backtest

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func TestDataFromTick_Load(t *testing.T) {
	type fields struct {
		ticks []*ticker.Price
		Data  Data
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"valid",
			fields{
				ticks: genTickerPrice(),
			},
			false,
		}, {
			"invalid",
			fields{},
			true,
		},
	}
	for x := range tests {
		test := tests[x]
		t.Run(test.name, func(t *testing.T) {
			d := &DataFromTick{
				ticks: test.fields.ticks,
				Data:  test.fields.Data,
			}
			if err := d.Load(); (err != nil) != test.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func genTickerPrice() (out []*ticker.Price) {
	for x := 0; x < 100; x++ {
		out = append(out, &ticker.Price{
			Ask: float64(x),
			Bid: float64(x),
		})
	}
	return out
}
