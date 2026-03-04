package binanceus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestFormatToInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected kline.Interval
	}{
		{name: "one minute", input: "1m", expected: kline.OneMin},
		{name: "three minute", input: "3m", expected: kline.ThreeMin},
		{name: "five minute", input: "5m", expected: kline.FiveMin},
		{name: "fifteen minute", input: "15m", expected: kline.FifteenMin},
		{name: "thirty minute", input: "30m", expected: kline.ThirtyMin},
		{name: "one hour", input: "1h", expected: kline.OneHour},
		{name: "two hour", input: "2h", expected: kline.TwoHour},
		{name: "four hour", input: "4h", expected: kline.FourHour},
		{name: "six hour", input: "6h", expected: kline.SixHour},
		{name: "eight hour", input: "8h", expected: kline.EightHour},
		{name: "twelve hour", input: "12h", expected: kline.TwelveHour},
		{name: "one day", input: "1d", expected: kline.OneDay},
		{name: "three day", input: "3d", expected: kline.ThreeDay},
		{name: "one week", input: "1w", expected: kline.OneWeek},
		{name: "one month", input: "1M", expected: kline.OneMonth},
		{name: "invalid empty", input: "", expected: 0},
		{name: "invalid casing", input: "1H", expected: 0},
		{name: "invalid value", input: "10m", expected: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, formatToInterval(tc.input))
		})
	}
}
