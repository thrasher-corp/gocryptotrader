package kline

import "github.com/thrasher-corp/gct-ta/indicators"

// OHLC is a connector for technical analysis usage
type OHLC struct {
	Open   []float64
	High   []float64
	Low    []float64
	Close  []float64
	Volume []float64
}

// GetOHLC returns the entire subset of candles as a friendly type for technical
// gct technical analysis usage.
func (i *Item) GetOHLC() *OHLC {
	open := make([]float64, len(i.Candles))
	high := make([]float64, len(i.Candles))
	low := make([]float64, len(i.Candles))
	close := make([]float64, len(i.Candles))
	volume := make([]float64, len(i.Candles))
	for x := range i.Candles {
		open[x] = i.Candles[x].Open
		high[x] = i.Candles[x].High
		low[x] = i.Candles[x].Low
		close[x] = i.Candles[x].Close
		volume[x] = i.Candles[x].Volume
	}
	return &OHLC{open, high, low, close, volume}
}

// GetAverageTrueRange returns the Average True Range for the given period.
func (o *OHLC) GetAverageTrueRange(period int) ([]float64, error) {
	return indicators.ATR(o.High, o.Low, o.Close, period), nil
}

// GetBollingerBands returns Bollinger Bands for the given period.
func (o *OHLC) GetBollingerBands(options []float64, period int, nbDevUp, nbDevDown float64, m indicators.MaType) (upper, middle, lower []float64, err error) {
	upper, middle, lower = indicators.BBANDS(options, period, nbDevUp, nbDevDown, m)
	return
}

// GetCorrelationCoefficient returns GetCorrelation Coefficient against another
// candle data set for the given period.
func (o *OHLC) GetCorrelationCoefficient(other *OHLC, period int) ([]float64, error) {
	return indicators.CorrelationCoefficient(o.Close, other.Close, period), nil
}

// GetSimpleMovingAverage returns MA for the supplied price set for the given
// period.
func (o *OHLC) GetSimpleMovingAverage(option []float64, period int) ([]float64, error) {
	return indicators.SMA(o.Close, period), nil
}

// GetExponentialMovingAverage returns the EMA on the supplied price set for the
// given period.
func (o *OHLC) GetExponentialMovingAverage(option []float64, period int) ([]float64, error) {
	return indicators.EMA(o.Close, period), nil
}

// GetMovingAverageConvergenceDivergence returns the
// MACD (macd, signal period vals, histogram) for the given price
// set and the paramaters fast, slow signal time periods.
func (o *OHLC) GetMovingAverageConvergenceDivergence(option []float64, fast, slow, signal int) (macd, signalVals, histogram []float64, err error) {
	macd, signalVals, histogram = indicators.MACD(option, fast, slow, signal)
	return
}

// GetMoneyFlowIndex returns Money Flow Index for the given period.
func (o *OHLC) GetMoneyFlowIndex(period int) ([]float64, error) {
	return indicators.MFI(o.High, o.Low, o.Close, o.Volume, period), nil
}

// GetOnBalanceVolume returns On Balance Volume.
func (o *OHLC) GetOnBalanceVolume() ([]float64, error) {
	return indicators.OBV(o.Close, o.Volume), nil
}

// GetRelativeStrengthIndex returns the relative strength index from the the
// given price set and period.
func (o *OHLC) GetRelativeStrengthIndex(option []float64, period int) ([]float64, error) {
	return indicators.RSI(option, period), nil
}
