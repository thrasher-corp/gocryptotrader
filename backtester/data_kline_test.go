package backtest

import (
	"reflect"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestDataFromKline_StreamHigh(t *testing.T) {
	type fields struct {
		Item kline.Item
		Data Data
	}
	tests := []struct {
		name   string
		fields fields
		want   []float64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DataFromKline{
				Item: tt.fields.Item,
				Data: tt.fields.Data,
			}
			if got := d.StreamHigh(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StreamHigh() = %v, want %v", got, tt.want)
			}
		})
	}
}
