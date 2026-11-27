package kline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetOHLC(t *testing.T) {
	t.Parallel()
	if (&Item{Candles: []Candle{{Open: 1337}}}).GetOHLC() == nil {
		t.Fatal("unexpected value")
	}
}

func TestGetAverageTrueRange(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetAverageTrueRange(0)
	require.ErrorIs(t, err, errNilOHLC)

	ohlc = &OHLC{}
	_, err = ohlc.GetAverageTrueRange(0)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetAverageTrueRange(9)
	require.ErrorIs(t, err, errNoData)

	ohlc.High = append(ohlc.High, 1337)
	_, err = ohlc.GetAverageTrueRange(9)
	require.ErrorIs(t, err, errNoData)

	ohlc.Low = append(ohlc.Low, 1337)
	_, err = ohlc.GetAverageTrueRange(9)
	require.ErrorIs(t, err, errNoData)

	ohlc.Close = append(ohlc.Close, 1337)
	_, err = ohlc.GetAverageTrueRange(9)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetAverageTrueRange(1)
	require.NoError(t, err)

	wrap := Item{Candles: []Candle{{High: 1337, Low: 1337, Close: 1337}}}
	_, err = wrap.GetAverageTrueRange(1)
	require.NoError(t, err)
}

func TestGetBollingerBands(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetBollingerBands(0, 0, 0, 5)
	require.ErrorIs(t, err, errNilOHLC)

	ohlc = &OHLC{}
	_, err = ohlc.GetBollingerBands(0, 0, 0, 5)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetBollingerBands(9, 0, 0, 5)
	require.ErrorIs(t, err, errInvalidDeviationMultiplier)

	_, err = ohlc.GetBollingerBands(9, 1, 0, 5)
	require.ErrorIs(t, err, errInvalidDeviationMultiplier)

	_, err = ohlc.GetBollingerBands(9, 1, 1, 5)
	require.ErrorIs(t, err, errNoData)

	ohlc.Close = append(ohlc.Close, 1337, 1337, 1337, 1337, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetBollingerBands(10, 1, 1, 5)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetBollingerBands(9, 1, 1, 5)
	require.NoError(t, err)

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetBollingerBands(9, 1, 1, 5)
	require.NoError(t, err)
}

func TestGetCorrelationCoefficient(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetCorrelationCoefficient(nil, 0)
	require.ErrorIs(t, err, errNilOHLC)

	ohlc = &OHLC{}
	_, err = ohlc.GetCorrelationCoefficient(nil, 0)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetCorrelationCoefficient(nil, 1)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetCorrelationCoefficient(nil, 2)
	require.ErrorIs(t, err, errNilOHLC)

	_, err = ohlc.GetCorrelationCoefficient(&OHLC{}, 9)
	require.ErrorIs(t, err, errNoData)

	ohlc.Close = append(ohlc.Close, 1337, 1337)

	_, err = ohlc.GetCorrelationCoefficient(&OHLC{}, 9)
	require.ErrorIs(t, err, errNoData)

	_, err = ohlc.GetCorrelationCoefficient(&OHLC{Close: []float64{1337}}, 2)
	require.ErrorIs(t, err, errInvalidPeriod)

	ohlc.Close = append(ohlc.Close, 1337)
	_, err = ohlc.GetCorrelationCoefficient(&OHLC{Close: []float64{1337, 1337}}, 2)
	require.ErrorIs(t, err, errInvalidDataSetLengths)

	_, err = ohlc.GetCorrelationCoefficient(&OHLC{Close: []float64{1337, 1337, 1337}}, 2)
	require.NoError(t, err)

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetCorrelationCoefficient(&Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}}}, 2)
	require.NoError(t, err)
}

func TestGetSimpleMovingAverage(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetSimpleMovingAverage(nil, 0)
	require.ErrorIs(t, err, errNilOHLC)

	ohlc = &OHLC{}
	_, err = ohlc.GetSimpleMovingAverage(nil, 0)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetSimpleMovingAverage(nil, 9)
	require.ErrorIs(t, err, errNoData)

	_, err = ohlc.GetSimpleMovingAverage([]float64{1337}, 9)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetSimpleMovingAverage([]float64{1337, 1337}, 2)
	require.NoError(t, err)

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetSimpleMovingAverageOnClose(2)
	require.NoError(t, err)
}

func TestGetExponentialMovingAverage(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetExponentialMovingAverage(nil, 0)
	require.ErrorIs(t, err, errNilOHLC)

	ohlc = &OHLC{}
	_, err = ohlc.GetExponentialMovingAverage(nil, 0)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetExponentialMovingAverage(nil, 9)
	require.ErrorIs(t, err, errNoData)

	_, err = ohlc.GetExponentialMovingAverage([]float64{1337}, 9)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetExponentialMovingAverage([]float64{1337, 1337, 1337}, 2)
	require.NoError(t, err)

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetExponentialMovingAverageOnClose(2)
	require.NoError(t, err)
}

func TestGetMovingAverageConvergenceDivergence(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetMovingAverageConvergenceDivergence(nil, 0, 0, 0)
	require.ErrorIs(t, err, errNilOHLC)

	ohlc = &OHLC{}
	_, err = ohlc.GetMovingAverageConvergenceDivergence(nil, 0, 0, 0)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetMovingAverageConvergenceDivergence(nil, 1, 0, 0)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetMovingAverageConvergenceDivergence(nil, 1, 1, 0)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetMovingAverageConvergenceDivergence(nil, 1, 2, 0)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetMovingAverageConvergenceDivergence(nil, 1, 2, 1)
	require.ErrorIs(t, err, errNoData)

	_, err = ohlc.GetMovingAverageConvergenceDivergence([]float64{1337}, 1, 2, 2)
	require.ErrorIs(t, err, errNotEnoughData)

	_, err = ohlc.GetMovingAverageConvergenceDivergence([]float64{1337, 1337, 1337, 1337, 1337, 1337, 1337, 1337}, 1, 2, 1)
	require.NoError(t, err)

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetMovingAverageConvergenceDivergenceOnClose(1, 2, 1)
	require.NoError(t, err)
}

func TestGetMoneyFlowIndex(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetMoneyFlowIndex(0)
	require.ErrorIs(t, err, errNilOHLC)

	ohlc = &OHLC{}
	_, err = ohlc.GetMoneyFlowIndex(0)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetMoneyFlowIndex(9)
	require.ErrorIs(t, err, errNoData)

	ohlc.High = append(ohlc.High, 1337, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetMoneyFlowIndex(9)
	require.ErrorIs(t, err, errNoData)

	ohlc.Low = append(ohlc.Low, 1337, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetMoneyFlowIndex(9)
	require.ErrorIs(t, err, errNoData)

	ohlc.Close = append(ohlc.Close, 1337, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetMoneyFlowIndex(9)
	require.ErrorIs(t, err, errNoData)

	ohlc.Volume = append(ohlc.Volume, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetMoneyFlowIndex(5)
	require.ErrorIs(t, err, errInvalidDataSetLengths)

	ohlc.Volume = append(ohlc.Volume, 1337)
	_, err = ohlc.GetMoneyFlowIndex(6)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetMoneyFlowIndex(3)
	require.NoError(t, err)

	wrap := Item{Candles: []Candle{
		{Close: 1337, High: 1337, Low: 1337, Volume: 1337},
		{Close: 1337, High: 1337, Low: 1337, Volume: 1337},
		{Close: 1337, High: 1337, Low: 1337, Volume: 1337},
	}}
	_, err = wrap.GetMoneyFlowIndex(2)
	require.NoError(t, err)
}

func TestGetOnBalanceVolume(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetOnBalanceVolume()
	require.ErrorIs(t, err, errNilOHLC)

	ohlc = &OHLC{}
	_, err = ohlc.GetOnBalanceVolume()
	require.ErrorIs(t, err, errNoData)

	ohlc.Close = append(ohlc.Close, 1337, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetOnBalanceVolume()
	require.ErrorIs(t, err, errNoData)

	ohlc.Volume = append(ohlc.Volume, 0.00000001)
	_, err = ohlc.GetOnBalanceVolume()
	require.NoError(t, err)

	wrap := Item{Candles: []Candle{{Close: 1337, Volume: 1337}}}
	_, err = wrap.GetOnBalanceVolume()
	require.NoError(t, err)
}

func TestGetRelativeStrengthIndex(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetRelativeStrengthIndex(nil, 0)
	require.ErrorIs(t, err, errNilOHLC)

	ohlc = &OHLC{}
	_, err = ohlc.GetRelativeStrengthIndex(nil, 0)
	require.ErrorIs(t, err, errInvalidPeriod)

	_, err = ohlc.GetRelativeStrengthIndex(nil, 9)
	require.ErrorIs(t, err, errNotEnoughData)

	_, err = ohlc.GetRelativeStrengthIndex([]float64{1337, 1337, 1337}, 9)
	require.ErrorIs(t, err, errInvalidPeriod)

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetRelativeStrengthIndexOnClose(2)
	require.NoError(t, err)
}
