package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func TestCanTransact(t *testing.T) {
	t.Parallel()
	for _, ti := range []struct {
		side     gctorder.Side
		expected bool
	}{
		{
			side:     gctorder.UnknownSide,
			expected: false,
		},
		{
			side:     gctorder.Buy,
			expected: true,
		},
		{
			side:     gctorder.Sell,
			expected: true,
		},
		{
			side:     gctorder.Bid,
			expected: true,
		},
		{
			side:     gctorder.Ask,
			expected: true,
		},
		{
			// while anyside can work in GCT, it's a no for the backtester
			side:     gctorder.AnySide,
			expected: false,
		},
		{
			side:     gctorder.Long,
			expected: true,
		},
		{
			side:     gctorder.Short,
			expected: true,
		},
		{
			side:     gctorder.ClosePosition,
			expected: true,
		},
		{
			side:     gctorder.DoNothing,
			expected: false,
		},
		{
			side:     gctorder.TransferredFunds,
			expected: false,
		},
		{
			side:     gctorder.CouldNotBuy,
			expected: false,
		},
		{
			side:     gctorder.CouldNotSell,
			expected: false,
		},
		{
			side:     gctorder.CouldNotShort,
			expected: false,
		},
		{
			side:     gctorder.CouldNotLong,
			expected: false,
		},
		{
			side:     gctorder.CouldNotCloseShort,
			expected: false,
		},
		{
			side:     gctorder.CouldNotCloseLong,
			expected: false,
		},
		{
			side:     gctorder.MissingData,
			expected: false,
		},
	} {
		t.Run(ti.side.String(), func(t *testing.T) {
			t.Parallel()
			if CanTransact(ti.side) != ti.expected {
				t.Errorf("received '%v' expected '%v'", ti.side, ti.expected)
			}
		})
	}
}

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
	for _, tc := range []struct {
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
		t.Run(tc.str, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, FitStringToLimit(tc.str, tc.sep, tc.limit, tc.upper))
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
	if CMDColours.Success != "" {
		t.Error("expected purged colour")
	}
}

func TestGenerateFileName(t *testing.T) {
	t.Parallel()
	_, err := GenerateFileName("", "")
	assert.ErrorIs(t, err, errCannotGenerateFileName)

	_, err = GenerateFileName("hello", "")
	assert.ErrorIs(t, err, errCannotGenerateFileName)

	_, err = GenerateFileName("", "moto")
	assert.ErrorIs(t, err, errCannotGenerateFileName)

	_, err = GenerateFileName("hello", "moto")
	assert.NoError(t, err)

	name, err := GenerateFileName("......HELL0.  +  _", "moto.")
	require.NoError(t, err, "GenerateFileName must not error")
	assert.Equal(t, "hell0_.moto", name)
}

func TestRegisterBacktesterSubLoggers(t *testing.T) {
	t.Parallel()
	err := RegisterBacktesterSubLoggers()
	assert.NoError(t, err)

	err = RegisterBacktesterSubLoggers()
	assert.ErrorIs(t, err, log.ErrSubLoggerAlreadyRegistered)
}
