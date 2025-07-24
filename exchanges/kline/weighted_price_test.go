package kline

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var accuracy10dp = 1 / math.Pow10(10)

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()
	c := Candle{}
	assert.Zero(t, c.GetAveragePrice(), "GetAveragePrice should return zero")

	c.High = 20
	assert.Equal(t, 5.0, c.GetAveragePrice(), "GetAveragePrice should return correct value")
}

func TestGetAveragePrice_OHLC(t *testing.T) {
	t.Parallel()
	var ohlc *OHLC
	_, err := ohlc.GetAveragePrice(-1)
	assert.ErrorIs(t, err, errNilOHLC, "GetAveragePrice should error on nil")

	ohlc = &OHLC{}
	_, err = ohlc.GetAveragePrice(-1)
	assert.ErrorIs(t, err, errInvalidElement, "GetAveragePrice should error on negative index")

	_, err = ohlc.GetAveragePrice(0)
	assert.ErrorIs(t, err, errElementExceedsDataLength, "GetAveragePrice should error on empty timeseries")

	ohlc.High = append(ohlc.High, 20)
	ohlc.Open = append(ohlc.Open, 0)
	ohlc.Low = append(ohlc.Low, 0)
	ohlc.Close = append(ohlc.Close, 0)
	avgPrice, err := ohlc.GetAveragePrice(0)
	assert.NoError(t, err, "GetAveragePrice should not error")
	assert.Equal(t, 5.0, avgPrice, "GetAveragePrice should return correct value")
}

var twapdataset = []Candle{
	{Time: time.Date(2020, 6, 17, 0, 0, 0, 0, time.UTC), Close: 351.59, Open: 355.15, High: 355.40, Low: 351.09},
	{Time: time.Date(2020, 6, 16, 0, 0, 0, 0, time.UTC), Close: 352.08, Open: 351.46, High: 353.20, Low: 344.72},
	{Time: time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC), Close: 342.99, Open: 333.25, High: 345.68, Low: 332.58},
	{Time: time.Date(2020, 6, 12, 0, 0, 0, 0, time.UTC), Close: 338.80, Open: 344.72, High: 347.80, Low: 334.22},
	{Time: time.Date(2020, 6, 11, 0, 0, 0, 0, time.UTC), Close: 335.90, Open: 349.31, High: 351.06, Low: 335.48},
	{Time: time.Date(2020, 6, 10, 0, 0, 0, 0, time.UTC), Close: 352.84, Open: 347.90, High: 354.77, Low: 346.09},
	{Time: time.Date(2020, 6, 9, 0, 0, 0, 0, time.UTC), Close: 343.99, Open: 332.14, High: 345.61, Low: 332.01},
	{Time: time.Date(2020, 6, 8, 0, 0, 0, 0, time.UTC), Close: 333.46, Open: 330.25, High: 333.60, Low: 327.32},
	{Time: time.Date(2020, 6, 5, 0, 0, 0, 0, time.UTC), Close: 331.50, Open: 323.35, High: 331.75, Low: 323.23},
	{Time: time.Date(2020, 6, 4, 0, 0, 0, 0, time.UTC), Close: 322.32, Open: 324.39, High: 325.62, Low: 320.78},
	{Time: time.Date(2020, 6, 3, 0, 0, 0, 0, time.UTC), Close: 325.12, Open: 324.66, High: 326.20, Low: 322.30},
	{Time: time.Date(2020, 6, 2, 0, 0, 0, 0, time.UTC), Close: 323.34, Open: 320.75, High: 323.44, Low: 318.93},
	{Time: time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC), Close: 321.85, Open: 317.75, High: 322.35, Low: 317.21},
	{Time: time.Date(2020, 5, 29, 0, 0, 0, 0, time.UTC), Close: 317.94, Open: 319.25, High: 321.15, Low: 316.47},
	{Time: time.Date(2020, 5, 28, 0, 0, 0, 0, time.UTC), Close: 318.25, Open: 316.77, High: 323.44, Low: 315.63},
	{Time: time.Date(2020, 5, 27, 0, 0, 0, 0, time.UTC), Close: 318.11, Open: 316.14, High: 318.71, Low: 313.09},
	{Time: time.Date(2020, 5, 26, 0, 0, 0, 0, time.UTC), Close: 316.73, Open: 323.50, High: 324.24, Low: 316.50},
	{Time: time.Date(2020, 5, 22, 0, 0, 0, 0, time.UTC), Close: 318.89, Open: 315.77, High: 319.23, Low: 315.35},
	{Time: time.Date(2020, 5, 21, 0, 0, 0, 0, time.UTC), Close: 316.85, Open: 318.66, High: 320.89, Low: 315.87},
	{Time: time.Date(2020, 5, 20, 0, 0, 0, 0, time.UTC), Close: 319.23, Open: 316.68, High: 319.52, Low: 316.20},
	{Time: time.Date(2020, 5, 19, 0, 0, 0, 0, time.UTC), Close: 313.14, Open: 315.03, High: 318.52, Low: 313.01},
	{Time: time.Date(2020, 5, 18, 0, 0, 0, 0, time.UTC), Close: 314.96, Open: 313.17, High: 316.50, Low: 310.32},
}

func TestGetTWAP_OHLC(t *testing.T) {
	t.Parallel()
	var ohlc *OHLC
	_, err := ohlc.GetTWAP()
	assert.ErrorIs(t, err, errNilOHLC, "GetTWAP should error on nil")

	ohlc = &OHLC{}
	_, err = ohlc.GetTWAP()
	assert.ErrorIs(t, err, errNoData, "GetTWAP should error on no data")

	ohlc.Open = append(ohlc.Open, 20)
	ohlc.High = append(ohlc.High, 20)
	ohlc.Low = append(ohlc.Low, 20)
	ohlc.Close = append(ohlc.Close, 20, 20)
	_, err = ohlc.GetTWAP()
	assert.ErrorIs(t, err, errDataLengthMismatch, "GetTWAP should error on length mismatch")

	i := Item{}
	i.Candles = twapdataset

	ohlc = i.GetOHLC()
	twap, err := ohlc.GetTWAP()
	assert.NoError(t, err, "GetTWAP should not error")
	assert.InDelta(t, 328.147840909091, twap, accuracy10dp, "TWAP should be correct")
}

func TestGetTWAP(t *testing.T) {
	t.Parallel()
	candles := Item{}
	_, err := candles.GetTWAP()
	assert.ErrorIs(t, err, errNoData, "GetTWAP should error on no data")

	candles.Candles = twapdataset
	twap, err := candles.GetTWAP()
	assert.NoError(t, err, "GetTWAP should not error")
	assert.InDelta(t, 328.147840909091, twap, accuracy10dp, "TWAP should be correct")
}

var vwapdataset = []Candle{
	{Time: time.Date(2019, 10, 10, 9, 31, 0, 0, time.UTC), Open: 245.2903, High: 245.516, Low: 244.7652, Close: 244.8702, Volume: 103033},
	{Time: time.Date(2019, 10, 10, 9, 32, 0, 0, time.UTC), Open: 245.0807, High: 245.0807, Low: 244.55, Close: 244.66, Volume: 21168},
	{Time: time.Date(2019, 10, 10, 9, 33, 0, 0, time.UTC), Open: 244.58, High: 245.8, Low: 244.55, Close: 245.6, Volume: 36544},
	{Time: time.Date(2019, 10, 10, 9, 34, 0, 0, time.UTC), Open: 245.7097, High: 246.09, Low: 245.57, Close: 245.92, Volume: 30057},
	{Time: time.Date(2019, 10, 10, 9, 35, 0, 0, time.UTC), Open: 245.62, High: 245.62, Low: 245.62, Close: 245.62, Volume: 26301},
	{Time: time.Date(2019, 10, 10, 9, 36, 0, 0, time.UTC), Open: 245.7126, High: 246.44, Low: 245.7126, Close: 246.188, Volume: 31494},
	{Time: time.Date(2019, 10, 10, 9, 37, 0, 0, time.UTC), Open: 246.46, High: 246.46, Low: 246.45, Close: 246.45, Volume: 24271},
	{Time: time.Date(2019, 10, 10, 9, 38, 0, 0, time.UTC), Open: 246.755, High: 246.755, Low: 246.25, Close: 246.25, Volume: 37951},
	{Time: time.Date(2019, 10, 10, 9, 39, 0, 0, time.UTC), Open: 246.2818, High: 246.655, Low: 246.2818, Close: 246.655, Volume: 15324},
	{Time: time.Date(2019, 10, 10, 9, 40, 0, 0, time.UTC), Open: 246.78, High: 246.78, Low: 246.56, Close: 246.762, Volume: 23285},
	{Time: time.Date(2019, 10, 10, 9, 41, 0, 0, time.UTC), Open: 246.75, High: 246.75, Low: 246.38, Close: 246.5, Volume: 23365},
	{Time: time.Date(2019, 10, 10, 9, 42, 0, 0, time.UTC), Open: 246.17, High: 246.17, Low: 246.17, Close: 246.17, Volume: 16130},
	{Time: time.Date(2019, 10, 10, 9, 43, 0, 0, time.UTC), Open: 246.135, High: 246.135, Low: 245.82, Close: 245.82, Volume: 27227},
	{Time: time.Date(2019, 10, 10, 9, 44, 0, 0, time.UTC), Open: 245.9335, High: 245.9335, Low: 245.91, Close: 245.91, Volume: 14464},
	{Time: time.Date(2019, 10, 10, 9, 45, 0, 0, time.UTC), Open: 246.41, High: 246.41, Low: 246.41, Close: 246.41, Volume: 17156},
	{Time: time.Date(2019, 10, 10, 9, 46, 0, 0, time.UTC), Open: 246.44, High: 246.46, Low: 246.1683, Close: 246.1683, Volume: 23938},
	{Time: time.Date(2019, 10, 10, 9, 47, 0, 0, time.UTC), Open: 246.2857, High: 246.57, Low: 246.2857, Close: 246.57, Volume: 70833},
	{Time: time.Date(2019, 10, 10, 9, 48, 0, 0, time.UTC), Open: 246.6, High: 247.47, Low: 246.6, Close: 247.47, Volume: 59743},
	{Time: time.Date(2019, 10, 10, 9, 49, 0, 0, time.UTC), Open: 247.49, High: 247.65, Low: 247.49, Close: 247.65, Volume: 71995},
	{Time: time.Date(2019, 10, 10, 9, 50, 0, 0, time.UTC), Open: 247.685, High: 247.801, Low: 247.65, Close: 247.69, Volume: 46038},
	{Time: time.Date(2019, 10, 10, 9, 51, 0, 0, time.UTC), Open: 247.95, High: 248.74, Low: 247.95, Close: 248.74, Volume: 103773},
	{Time: time.Date(2019, 10, 10, 9, 52, 0, 0, time.UTC), Open: 248.56, High: 248.56, Low: 247.95, Close: 247.95, Volume: 73810},
	{Time: time.Date(2019, 10, 10, 9, 53, 0, 0, time.UTC), Open: 247.93, High: 247.93, Low: 247.6614, Close: 247.6614, Volume: 29784},
	{Time: time.Date(2019, 10, 10, 9, 54, 0, 0, time.UTC), Open: 247.74, High: 247.76, Low: 247.65, Close: 247.76, Volume: 37138},
	{Time: time.Date(2019, 10, 10, 9, 55, 0, 0, time.UTC), Open: 247.93, High: 248.03, Low: 247.93, Close: 248.03, Volume: 53166},
	{Time: time.Date(2019, 10, 10, 9, 56, 0, 0, time.UTC), Open: 247.91, High: 248.44, Low: 247.91, Close: 248.44, Volume: 40789},
	{Time: time.Date(2019, 10, 10, 9, 57, 0, 0, time.UTC), Open: 248.52, High: 248.52, Low: 248.3154, Close: 248.3154, Volume: 51988},
	{Time: time.Date(2019, 10, 10, 9, 58, 0, 0, time.UTC), Open: 248.4409, High: 248.62, Low: 248.4409, Close: 248.62, Volume: 53405},
	{Time: time.Date(2019, 10, 10, 9, 59, 0, 0, time.UTC), Open: 248.9199, High: 248.9199, Low: 248.9199, Close: 248.9199, Volume: 85348},
	{Time: time.Date(2019, 10, 10, 10, 0, 0, 0, time.UTC), Open: 248.91, High: 249.08, Low: 248.42, Close: 248.72, Volume: 58270},
}
var expectVWAPs = []float64{245.05046666666664, 245.00156932123465, 245.07320400593073, 245.19714781780763, 245.248374356565, 245.35797872352975, 245.45540807301208, 245.57298124760712, 245.61797546720302, 245.6901232761351, 245.7435986712912, 245.76128302894574, 245.771994363731, 245.7768929849006, 245.80115004533573, 245.82471633454026, 245.90964645148168, 246.0356579876492, 246.20233204964117, 246.29892677543359, 246.57315726207088, 246.70305234595537, 246.73669536160304, 246.7746731036053, 246.83849361010806, 246.89338504378165, 246.96313273581723, 247.03640100225914, 247.16505290840146, 247.23522648930867}

func TestGetVWAPs(t *testing.T) {
	t.Parallel()
	candles := Item{}
	_, err := candles.GetVWAPs()
	require.ErrorIs(t, err, errNoData)
	candles.Candles = vwapdataset
	vwap, err := candles.GetVWAPs()
	assert.NoError(t, err, "GetVWAPs should not error")
	assert.Len(t, vwap, len(expectVWAPs), "Should get correct number of vwaps")
	for i := range vwap {
		assert.InDelta(t, expectVWAPs[i], vwap[i], accuracy10dp, "VWAPS should be correct")
	}
}

func TestGetVWAPs_OHLC(t *testing.T) {
	t.Parallel()
	var ohlc *OHLC
	_, err := ohlc.GetVWAPs()
	assert.ErrorIs(t, err, errNilOHLC, "GetVWAPs should error correctly with no data")

	ohlc = &OHLC{}
	_, err = ohlc.GetVWAPs()
	assert.ErrorIs(t, err, errNoData, "GetVWAP should error on no data")

	ohlc.Open = append(ohlc.Open, 20)
	ohlc.High = append(ohlc.High, 20)
	ohlc.Low = append(ohlc.Low, 20)
	ohlc.Close = append(ohlc.Close, 20, 20)

	_, err = ohlc.GetVWAPs()
	assert.ErrorIs(t, err, errDataLengthMismatch, "GetVWAP should error on length mismatch")

	ohlc = (&Item{Candles: vwapdataset}).GetOHLC()

	vwap, err := ohlc.GetVWAPs()
	assert.NoError(t, err, "GetVWAPs should not error")
	assert.Len(t, vwap, len(expectVWAPs), "Should get correct number of vwaps")
	for i := range vwap {
		assert.InDelta(t, expectVWAPs[i], vwap[i], accuracy10dp, "VWAPS should be correct")
	}
}

func TestGetTypicalPrice_OHLC(t *testing.T) {
	t.Parallel()
	var ohlc *OHLC
	_, err := ohlc.GetTypicalPrice(-1)
	assert.ErrorIs(t, err, errNilOHLC, "GetTypicalPrice should error correctly with no data")

	ohlc = &OHLC{}
	_, err = ohlc.GetTypicalPrice(-1)
	assert.ErrorIs(t, err, errInvalidElement, "GetTypicalPrice should error on negative index")

	_, err = ohlc.GetTypicalPrice(0)
	assert.ErrorIs(t, err, errElementExceedsDataLength, "GetTypicalPrice should error on empty timeseries")

	ohlc.High = append(ohlc.High, 15)
	ohlc.Low = append(ohlc.Low, 0)
	ohlc.Close = append(ohlc.Close, 0)
	avgPrice, err := ohlc.GetTypicalPrice(0)
	assert.NoError(t, err, "GetTypicalPrice should not error")
	assert.Equal(t, 5.0, avgPrice, "GetTypicalPrice should return correct value")
}
