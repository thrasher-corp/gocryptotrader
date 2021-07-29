package common

import (
	"fmt"
	"testing"
)

func TestDataTypeConversion(t *testing.T) {
	for _, ti := range []struct {
		title     string
		dataType  string
		want      int64
		expectErr bool
	}{
		{
			title:    "Candle data type",
			dataType: CandleStr,
			want:     DataCandle,
		},
		{
			title:    "Trade data type",
			dataType: TradeStr,
			want:     DataTrade,
		},
		{
			title:     "Unknown data type",
			dataType:  "unknown",
			want:      0,
			expectErr: true,
		},
	} {
		t.Run(ti.title, func(t *testing.T) {
			got, err := DataTypeToInt(ti.dataType)
			if ti.expectErr {
				if err == nil {
					t.Errorf("expected error")
				}
			} else {
				if err != nil || got != ti.want {
					t.Error(fmt.Errorf("%s: expected %d, got %d, err: %v", ti.dataType, ti.want, got, err))
				}
			}
		})
	}
}
