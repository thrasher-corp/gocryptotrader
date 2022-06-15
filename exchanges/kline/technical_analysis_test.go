package kline

import (
	"errors"
	"testing"
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
	if !errors.Is(err, errNilOHLC) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilOHLC)
	}

	ohlc = &OHLC{}
	_, err = ohlc.GetAverageTrueRange(0)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetAverageTrueRange(9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.High = append(ohlc.High, 1337)
	_, err = ohlc.GetAverageTrueRange(9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.Low = append(ohlc.Low, 1337)
	_, err = ohlc.GetAverageTrueRange(9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.Close = append(ohlc.Close, 1337)
	_, err = ohlc.GetAverageTrueRange(9)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetAverageTrueRange(1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	wrap := Item{Candles: []Candle{{High: 1337, Low: 1337, Close: 1337}}}
	_, err = wrap.GetAverageTrueRange(1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetBollingerBands(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetBollingerBands(0, 0, 0, 5)
	if !errors.Is(err, errNilOHLC) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilOHLC)
	}

	ohlc = &OHLC{}
	_, err = ohlc.GetBollingerBands(0, 0, 0, 5)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetBollingerBands(9, 0, 0, 5)
	if !errors.Is(err, errInvalidDeviationMultiplier) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidDeviationMultiplier)
	}

	_, err = ohlc.GetBollingerBands(9, 1, 0, 5)
	if !errors.Is(err, errInvalidDeviationMultiplier) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidDeviationMultiplier)
	}

	_, err = ohlc.GetBollingerBands(9, 1, 1, 5)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.Close = append(ohlc.Close, 1337, 1337, 1337, 1337, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetBollingerBands(10, 1, 1, 5)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetBollingerBands(9, 1, 1, 5)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetBollingerBands(9, 1, 1, 5)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetCorrelationCoefficient(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetCorrelationCoefficient(nil, 0)
	if !errors.Is(err, errNilOHLC) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilOHLC)
	}

	ohlc = &OHLC{}
	_, err = ohlc.GetCorrelationCoefficient(nil, 0)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetCorrelationCoefficient(nil, 1)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetCorrelationCoefficient(nil, 2)
	if !errors.Is(err, errNilOHLC) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilOHLC)
	}

	_, err = ohlc.GetCorrelationCoefficient(&OHLC{}, 9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.Close = append(ohlc.Close, 1337, 1337)

	_, err = ohlc.GetCorrelationCoefficient(&OHLC{}, 9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	_, err = ohlc.GetCorrelationCoefficient(&OHLC{Close: []float64{1337}}, 2)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	ohlc.Close = append(ohlc.Close, 1337)
	_, err = ohlc.GetCorrelationCoefficient(&OHLC{Close: []float64{1337, 1337}}, 2)
	if !errors.Is(err, errInvalidDataSetLengths) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidDataSetLengths)
	}

	_, err = ohlc.GetCorrelationCoefficient(&OHLC{Close: []float64{1337, 1337, 1337}}, 2)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetCorrelationCoefficient(&Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}}}, 2)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetSimpleMovingAverage(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetSimpleMovingAverage(nil, 0)
	if !errors.Is(err, errNilOHLC) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilOHLC)
	}

	ohlc = &OHLC{}
	_, err = ohlc.GetSimpleMovingAverage(nil, 0)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetSimpleMovingAverage(nil, 9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	_, err = ohlc.GetSimpleMovingAverage([]float64{1337}, 9)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetSimpleMovingAverage([]float64{1337, 1337}, 2)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetSimpleMovingAverageOnClose(2)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetExponentialMovingAverage(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetExponentialMovingAverage(nil, 0)
	if !errors.Is(err, errNilOHLC) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilOHLC)
	}

	ohlc = &OHLC{}
	_, err = ohlc.GetExponentialMovingAverage(nil, 0)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetExponentialMovingAverage(nil, 9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	_, err = ohlc.GetExponentialMovingAverage([]float64{1337}, 9)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetExponentialMovingAverage([]float64{1337, 1337, 1337}, 2)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetExponentialMovingAverageOnClose(2)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetMovingAverageConvergenceDivergence(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetMovingAverageConvergenceDivergence(nil, 0, 0, 0)
	if !errors.Is(err, errNilOHLC) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilOHLC)
	}

	ohlc = &OHLC{}
	_, err = ohlc.GetMovingAverageConvergenceDivergence(nil, 0, 0, 0)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetMovingAverageConvergenceDivergence(nil, 1, 0, 0)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetMovingAverageConvergenceDivergence(nil, 1, 1, 0)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetMovingAverageConvergenceDivergence(nil, 1, 2, 0)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetMovingAverageConvergenceDivergence(nil, 1, 2, 1)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	_, err = ohlc.GetMovingAverageConvergenceDivergence([]float64{1337}, 1, 2, 2)
	if !errors.Is(err, errNotEnoughData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughData)
	}

	_, err = ohlc.GetMovingAverageConvergenceDivergence([]float64{1337, 1337, 1337, 1337, 1337, 1337, 1337, 1337}, 1, 2, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetMovingAverageConvergenceDivergenceOnClose(1, 2, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetMoneyFlowIndex(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetMoneyFlowIndex(0)
	if !errors.Is(err, errNilOHLC) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilOHLC)
	}

	ohlc = &OHLC{}
	_, err = ohlc.GetMoneyFlowIndex(0)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetMoneyFlowIndex(9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.High = append(ohlc.High, 1337, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetMoneyFlowIndex(9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.Low = append(ohlc.Low, 1337, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetMoneyFlowIndex(9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.Close = append(ohlc.Close, 1337, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetMoneyFlowIndex(9)
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.Volume = append(ohlc.Volume, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetMoneyFlowIndex(5)
	if !errors.Is(err, errInvalidDataSetLengths) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidDataSetLengths)
	}

	ohlc.Volume = append(ohlc.Volume, 1337)
	_, err = ohlc.GetMoneyFlowIndex(6)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetMoneyFlowIndex(3)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	wrap := Item{Candles: []Candle{
		{Close: 1337, High: 1337, Low: 1337, Volume: 1337},
		{Close: 1337, High: 1337, Low: 1337, Volume: 1337},
		{Close: 1337, High: 1337, Low: 1337, Volume: 1337},
	}}
	_, err = wrap.GetMoneyFlowIndex(2)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetOnBalanceVolume(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetOnBalanceVolume()
	if !errors.Is(err, errNilOHLC) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilOHLC)
	}

	ohlc = &OHLC{}
	_, err = ohlc.GetOnBalanceVolume()
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.Close = append(ohlc.Close, 1337, 1337, 1337, 1337, 1337, 1337)
	_, err = ohlc.GetOnBalanceVolume()
	if !errors.Is(err, errNoData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoData)
	}

	ohlc.Volume = append(ohlc.Volume, 0.00000001)
	_, err = ohlc.GetOnBalanceVolume()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	wrap := Item{Candles: []Candle{{Close: 1337, Volume: 1337}}}
	_, err = wrap.GetOnBalanceVolume()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetRelativeStrengthIndex(t *testing.T) {
	t.Parallel()

	var ohlc *OHLC
	_, err := ohlc.GetRelativeStrengthIndex(nil, 0)
	if !errors.Is(err, errNilOHLC) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilOHLC)
	}

	ohlc = &OHLC{}
	_, err = ohlc.GetRelativeStrengthIndex(nil, 0)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	_, err = ohlc.GetRelativeStrengthIndex(nil, 9)
	if !errors.Is(err, errNotEnoughData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNotEnoughData)
	}

	_, err = ohlc.GetRelativeStrengthIndex([]float64{1337, 1337, 1337}, 9)
	if !errors.Is(err, errInvalidPeriod) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPeriod)
	}

	wrap := Item{Candles: []Candle{{Close: 1337}, {Close: 1337}, {Close: 1337}}}
	_, err = wrap.GetRelativeStrengthIndexOnClose(2)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}
