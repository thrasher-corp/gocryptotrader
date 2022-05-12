package common

import (
	"fmt"
	"testing"
)

func TestDataTypeConversion(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got, err := DataTypeToInt(ti.dataType)
			if ti.expectErr {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil || got != ti.want {
					t.Error(fmt.Errorf("%s: expected %d, got %d, err: %v", ti.dataType, ti.want, got, err))
				}
			}
		})
	}
}

func TestFitStringToLimit(t *testing.T) {
	t.Parallel()
	for _, ti := range []struct {
		str      string
		sep      string
		limit    int
		expected string
		upper    bool
	}{
		{
			str:      "good",
			sep:      " ",
			limit:    5,
			expected: "GOOD ",
			upper:    true,
		},
		{
			str:      "negative limit",
			sep:      " ",
			limit:    -1,
			expected: "negative limit",
		},
		{
			str:      "long spacer",
			sep:      "--",
			limit:    14,
			expected: "long spacer---",
		},
		{
			str:      "zero limit",
			sep:      "--",
			limit:    0,
			expected: "",
		},
		{
			str:      "over limit",
			sep:      "--",
			limit:    6,
			expected: "ove...",
		},
		{
			str:      "hi",
			sep:      " ",
			limit:    1,
			expected: "h",
		},
	} {
		test := ti
		t.Run(test.str, func(t *testing.T) {
			t.Parallel()
			result := FitStringToLimit(test.str, test.sep, test.limit, test.upper)
			if result != test.expected {
				t.Errorf("received '%v' expected '%v'", result, test.expected)
			}
		})
	}
}

func TestLogo(t *testing.T) {
	colourLogo := Logo()
	if colourLogo == "" {
		t.Error("expected a logo")
	}
	PurgeColours()
	if len(colourLogo) == len(Logo()) {
		t.Error("expected logo with colours removed")
	}
}

func TestPurgeColours(t *testing.T) {
	PurgeColours()
	if ColourSuccess != "" {
		t.Error("expected purged colour")
	}
}
