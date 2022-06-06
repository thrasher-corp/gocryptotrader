package kline

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gct-ta/indicators"
)

var (
	errInvalidPeriod              = errors.New("invalid period")
	errNoData                     = errors.New("no data")
	errInvalidDeviationMultiplier = errors.New("invalid deviation multiplier")
	errNilOHLC                    = errors.New("nil OHLC data")
	errInvalidDataSetLengths      = errors.New("invalid data set lengths")
	errNotEnoughData              = errors.New("not enough data to derive signal")
)

// OHLC is a connector for technical analysis usage
type OHLC struct {
	Open   []float64
	High   []float64
	Low    []float64
	Close  []float64
	Volume []float64
}

// GetOHLC returns the entire subset of candles as a friendly type for gct
// technical analysis usage.
func (k *Item) GetOHLC() *OHLC {
	ohlc := &OHLC{
		Open:   make([]float64, len(k.Candles)),
		High:   make([]float64, len(k.Candles)),
		Low:    make([]float64, len(k.Candles)),
		Close:  make([]float64, len(k.Candles)),
		Volume: make([]float64, len(k.Candles)),
	}
	for x := range k.Candles {
		ohlc.Open[x] = k.Candles[x].Open
		ohlc.High[x] = k.Candles[x].High
		ohlc.Low[x] = k.Candles[x].Low
		ohlc.Close[x] = k.Candles[x].Close
		ohlc.Volume[x] = k.Candles[x].Volume
	}
	return ohlc
}

// GetAverageTrueRange returns the Average True Range for the given period.
func (k *Item) GetAverageTrueRange(period int) ([]float64, error) {
	return k.GetOHLC().GetAverageTrueRange(period)
}

// GetAverageTrueRange returns the Average True Range for the given period.
func (o *OHLC) GetAverageTrueRange(period int) ([]float64, error) {
	if o == nil {
		return nil, fmt.Errorf("get average true range %w", errNilOHLC)
	}
	if period <= 0 {
		return nil, fmt.Errorf("get average true range %w", errInvalidPeriod)
	}
	if len(o.High) == 0 {
		return nil, fmt.Errorf("get average true range high %w", errNoData)
	}
	if len(o.Low) == 0 {
		return nil, fmt.Errorf("get average true range low %w", errNoData)
	}
	if len(o.Close) == 0 {
		return nil, fmt.Errorf("get average true range close %w", errNoData)
	}
	return indicators.ATR(o.High, o.Low, o.Close, period), nil
}

// GetBollingerBands returns Bollinger Bands for the given period.
func (k *Item) GetBollingerBands(period int, nbDevUp, nbDevDown float64, m indicators.MaType) (*Bollinger, error) {
	return k.GetOHLC().GetBollingerBands(period, nbDevUp, nbDevDown, m)
}

// Bollinger defines a return type for the bollinger bands
type Bollinger struct {
	Upper  []float64
	Middle []float64
	Lower  []float64
}

// GetBollingerBands returns Bollinger Bands for the given period.
func (o *OHLC) GetBollingerBands(period int, nbDevUp, nbDevDown float64, m indicators.MaType) (*Bollinger, error) {
	if o == nil {
		return nil, fmt.Errorf("get bollinger bands %w", errNilOHLC)
	}
	if period <= 0 {
		return nil, fmt.Errorf("get bollinger bands %w", errInvalidPeriod)
	}
	if nbDevUp <= 0 {
		return nil, fmt.Errorf("get bollinger bands %w upper limit", errInvalidDeviationMultiplier)
	}
	if nbDevDown <= 0 {
		return nil, fmt.Errorf("get bollinger bands %w lower limit", errInvalidDeviationMultiplier)
	}
	if len(o.Close) == 0 {
		return nil, fmt.Errorf("get bollinger bands close %w", errNoData)
	}
	if period > len(o.Close) { // TODO: Investigate the panic when this protection is removed.
		return nil, fmt.Errorf("get bollinger bands %w '%v' should not exceed close data length '%v'",
			errInvalidPeriod, period, len(o.Close))
	}
	var bands Bollinger
	bands.Upper, bands.Middle, bands.Lower = indicators.BBANDS(o.Close,
		period,
		nbDevUp,
		nbDevDown,
		m)
	return &bands, nil
}

// GetCorrelationCoefficient returns GetCorrelation Coefficient against another
// candle data set for the given period.
func (k *Item) GetCorrelationCoefficient(other *Item, period int) ([]float64, error) {
	return k.GetOHLC().GetCorrelationCoefficient(other.GetOHLC(), period)
}

// GetCorrelationCoefficient returns GetCorrelation Coefficient against another
// candle data set for the given period.
func (o *OHLC) GetCorrelationCoefficient(other *OHLC, period int) ([]float64, error) {
	if o == nil {
		return nil, fmt.Errorf("get correlation coefficient %w", errNilOHLC)
	}
	if period <= 0 {
		return nil, fmt.Errorf("get correlation coefficient %w", errInvalidPeriod)
	}
	if other == nil {
		return nil, fmt.Errorf("get correlation coefficient %w", errNilOHLC)
	}
	if len(o.Close) == 0 {
		return nil, fmt.Errorf("get correlation coefficient close %w", errNoData)
	}
	if len(other.Close) == 0 {
		return nil, fmt.Errorf("get correlation coefficient comparison close %w", errNoData)
	}
	return indicators.CorrelationCoefficient(o.Close, other.Close, period), nil
}

// GetSimpleMovingAverageOnClose returns MA the close prices set for the given
// period.
func (k *Item) GetSimpleMovingAverageOnClose(period int) ([]float64, error) {
	ohlc := k.GetOHLC()
	return ohlc.GetSimpleMovingAverage(ohlc.Close, period)
}

// GetSimpleMovingAverage returns MA for the supplied price set for the given
// period.
func (o *OHLC) GetSimpleMovingAverage(option []float64, period int) ([]float64, error) {
	if o == nil {
		return nil, fmt.Errorf("get simple moving average %w", errNilOHLC)
	}
	if period <= 0 {
		return nil, fmt.Errorf("get simple moving average %w", errInvalidPeriod)
	}
	if len(option) == 0 {
		return nil, fmt.Errorf("get simple moving average %w", errNoData)
	}
	return indicators.SMA(option, period), nil
}

// GetExponentialMovingAverage returns the EMA on the close price set for the
// given period.
func (k *Item) GetExponentialMovingAverageOnClose(period int) ([]float64, error) {
	ohlc := k.GetOHLC()
	return ohlc.GetExponentialMovingAverage(ohlc.Close, period)
}

// GetExponentialMovingAverage returns the EMA on the supplied price set for the
// given period.
func (o *OHLC) GetExponentialMovingAverage(option []float64, period int) ([]float64, error) {
	if o == nil {
		return nil, fmt.Errorf("get exponential moving average %w", errNilOHLC)
	}
	if period <= 0 {
		return nil, fmt.Errorf("get exponential moving average %w", errInvalidPeriod)
	}
	if len(option) == 0 {
		return nil, fmt.Errorf("get exponential moving average %w", errNoData)
	}
	return indicators.EMA(option, period), nil
}

// GetMovingAverageConvergenceDivergence returns the
// MACD (macd, signal period vals, histogram) for the given price
// set and the parameters fast, slow signal time periods.
func (k *Item) GetMovingAverageConvergenceDivergenceOnClose(fast, slow, signal int) (*MACD, error) {
	ohlc := k.GetOHLC()
	return ohlc.GetMovingAverageConvergenceDivergence(ohlc.Close, fast, slow, signal)
}

// MACD defines MACD values
type MACD struct {
	MACD       []float64
	SignalVals []float64
	Histogram  []float64
}

// GetMovingAverageConvergenceDivergence returns the
// MACD (macd, signal period vals, histogram) for the given price
// set and the parameters fast, slow signal time periods.
func (o *OHLC) GetMovingAverageConvergenceDivergence(option []float64, fast, slow, signal int) (*MACD, error) {
	if o == nil {
		return nil, fmt.Errorf("get macd %w", errNilOHLC)
	}
	if fast <= 0 {
		return nil, fmt.Errorf("get macd %w fast", errInvalidPeriod)
	}
	if slow <= 0 {
		return nil, fmt.Errorf("get macd %w slow", errInvalidPeriod)
	}
	if signal <= 0 {
		return nil, fmt.Errorf("get macd %w signal", errInvalidPeriod)
	}
	if len(option) == 0 {
		return nil, fmt.Errorf("get macd %w", errNoData)
	}

	if len(option) < slow+signal-2 {
		return nil, fmt.Errorf("get macd %w %v data points are less than minimum %v length requirement derived from the slow %v and signal %v period subtract two, increase end date or scale down granularity",
			errNotEnoughData,
			len(option),
			slow+signal-2,
			slow,
			signal)
	}
	var macd MACD
	macd.MACD, macd.SignalVals, macd.Histogram = indicators.MACD(option,
		fast,
		slow,
		signal)
	return &macd, nil
}

// GetMoneyFlowIndex returns Money Flow Index for the given period.
func (k *Item) GetMoneyFlowIndex(period int) ([]float64, error) {
	return k.GetOHLC().GetMoneyFlowIndex(period)
}

// GetMoneyFlowIndex returns Money Flow Index for the given period.
func (o *OHLC) GetMoneyFlowIndex(period int) ([]float64, error) {
	if o == nil {
		return nil, fmt.Errorf("get money flow index %w", errNilOHLC)
	}
	if period <= 0 {
		return nil, fmt.Errorf("get money flow index %w", errInvalidPeriod)
	}
	highLen := len(o.High)
	if highLen == 0 {
		return nil, fmt.Errorf("get money flow index %w for high", errNoData)
	}
	lowLen := len(o.Low)
	if lowLen == 0 {
		return nil, fmt.Errorf("get money flow index %w for low", errNoData)
	}
	closeLen := len(o.Close)
	if closeLen == 0 {
		return nil, fmt.Errorf("get money flow index %w for close", errNoData)
	}
	volLen := len(o.Volume)
	if volLen == 0 {
		return nil, fmt.Errorf("get money flow index %w for volume", errNoData)
	}
	if highLen != closeLen || lowLen != closeLen || volLen != closeLen {
		// TODO: Investigate the panic when this protection is removed.
		// This is very unstable with incorrect lengths.
		return nil, fmt.Errorf("get money flow index %w", errInvalidDataSetLengths)
	}

	if period >= len(o.Close) {
		// TODO: Investigate the panic when this protection is removed.
		return nil, fmt.Errorf("get money flow index %w '%v' should not exceed or equal close data length '%v'",
			errInvalidPeriod, period, len(o.Close))
	}
	return indicators.MFI(o.High, o.Low, o.Close, o.Volume, period), nil
}

// GetOnBalanceVolume returns On Balance Volume.
func (k *Item) GetOnBalanceVolume() ([]float64, error) {
	return k.GetOHLC().GetOnBalanceVolume()
}

// GetOnBalanceVolume returns On Balance Volume.
func (o *OHLC) GetOnBalanceVolume() ([]float64, error) {
	if o == nil {
		return nil, fmt.Errorf("get on balance volume %w", errNilOHLC)
	}
	if len(o.Close) == 0 {
		return nil, fmt.Errorf("get on balance volume %w for close", errNoData)
	}
	if len(o.Volume) == 0 {
		return nil, fmt.Errorf("get on balance volume %w for volume", errNoData)
	}
	return indicators.OBV(o.Close, o.Volume), nil
}

// GetRelativeStrengthIndex returns the relative strength index from the the
// given price set and period.
func (k *Item) GetRelativeStrengthIndexOnClose(period int) ([]float64, error) {
	ohlc := k.GetOHLC()
	return ohlc.GetRelativeStrengthIndex(ohlc.Close, period)
}

// GetRelativeStrengthIndex returns the relative strength index from the the
// given price set and period.
func (o *OHLC) GetRelativeStrengthIndex(option []float64, period int) ([]float64, error) {
	if o == nil {
		return nil, fmt.Errorf("get relative strength index %w", errNilOHLC)
	}
	if period <= 0 {
		return nil, fmt.Errorf("get relative strength index %w", errInvalidPeriod)
	}
	if len(option) == 0 {
		return nil, fmt.Errorf("get relative strength index %w", errNoData)
	}
	return indicators.RSI(option, period), nil
}
