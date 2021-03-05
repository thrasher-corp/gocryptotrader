package rsi

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Name is the strategy name
	Name         = "rsi"
	rsiPeriodKey = "rsi-period"
	rsiLowKey    = "rsi-low"
	rsiHighKey   = "rsi-high"
	description  = `The relative strength index is a technical indicator used in the analysis of financial markets. It is intended to chart the current and historical strength or weakness of a stock or market based on the closing prices of a recent trading period`
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	rsiPeriod float64
	rsiLow    float64
	rsiHigh   float64
}

// Name returns the name of the strategy
func (s *Strategy) Name() string {
	return Name
}

// Description provides a nice overview of the strategy
// be it definition of terms or to highlight its purpose
func (s *Strategy) Description() string {
	return description
}

// OnSignal handles a data event and returns what action the strategy believes should occur
// For rsi, this means returning a buy signal when rsi is at or below a certain level, and a
// sell signal when it is at or above a certain level
func (s *Strategy) OnSignal(d data.Handler, _ portfolio.Handler) (signal.Event, error) {
	if d == nil {
		return nil, common.ErrNilEvent
	}
	es, err := s.GetBaseData(d)
	if err != nil {
		return nil, err
	}
	es.SetPrice(d.Latest().ClosePrice())
	offset := d.Offset()

	if offset <= int(s.rsiPeriod) {
		es.AppendReason("Not enough data for signal generation")
		es.SetDirection(common.DoNothing)
		return &es, nil
	}

	dataRange := d.StreamClose()
	var massagedData []float64
	massagedData, err = s.massageMissingData(dataRange, es.GetTime())
	if err != nil {
		return nil, err
	}
	rsi := indicators.RSI(massagedData, int(s.rsiPeriod))
	latestRSIValue := rsi[len(rsi)-1]
	if !d.HasDataAtTime(d.Latest().GetTime()) {
		es.SetDirection(common.MissingData)
		es.AppendReason(fmt.Sprintf("missing data at %v, cannot perform any actions. RSI %v", d.Latest().GetTime(), latestRSIValue))
		return &es, nil
	}

	switch {
	case latestRSIValue >= s.rsiHigh:
		es.SetDirection(order.Sell)
	case latestRSIValue <= s.rsiLow:
		es.SetDirection(order.Buy)
	default:
		es.SetDirection(common.DoNothing)
	}
	es.AppendReason(fmt.Sprintf("RSI at %.2f", latestRSIValue))

	return &es, nil
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
// There is nothing actually stopping this strategy from considering multiple currencies at once
// but for demonstration purposes, this strategy does not
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return false
}

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
// For rsi, multi-currency signal processing is unsupported for demonstration purposes
func (s *Strategy) OnSimultaneousSignals(_ []data.Handler, _ portfolio.Handler) ([]signal.Event, error) {
	return nil, base.ErrSimultaneousProcessingNotSupported
}

// SetCustomSettings allows a user to modify the RSI limits in their config
func (s *Strategy) SetCustomSettings(customSettings map[string]interface{}) error {
	for k, v := range customSettings {
		switch k {
		case rsiHighKey:
			rsiHigh, ok := v.(float64)
			if !ok || rsiHigh <= 0 {
				return fmt.Errorf("%w provided rsi-high value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.rsiHigh = rsiHigh
		case rsiLowKey:
			rsiLow, ok := v.(float64)
			if !ok || rsiLow <= 0 {
				return fmt.Errorf("%w provided rsi-low value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.rsiLow = rsiLow
		case rsiPeriodKey:
			rsiPeriod, ok := v.(float64)
			if !ok || rsiPeriod <= 0 {
				return fmt.Errorf("%w provided rsi-period value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.rsiPeriod = rsiPeriod
		default:
			return fmt.Errorf("%w unrecognised custom setting key %v with value %v. Cannot apply", base.ErrInvalidCustomSettings, k, v)
		}
	}

	return nil
}

// SetDefaults sets the custom settings to their default values
func (s *Strategy) SetDefaults() {
	s.rsiHigh = 70
	s.rsiLow = 30
	s.rsiPeriod = 14
}

// massageMissingData will replace missing data with the previous candle's data
// this will ensure that RSI can be calculated correctly
// the decision to handle missing data occurs at the strategy level, not all strategies
// may wish to modify data
func (s *Strategy) massageMissingData(data []float64, t time.Time) ([]float64, error) {
	var resp []float64
	var missingDataStreak float64
	for i := range data {
		if data[i] == 0 && i > int(s.rsiPeriod) {
			data[i] = data[i-1]
			missingDataStreak++
		} else {
			missingDataStreak = 0
		}
		if missingDataStreak >= s.rsiPeriod {
			return nil, fmt.Errorf("missing data exceeds RSI period length of %v at %s and will distort results. %w",
				s.rsiPeriod,
				t.Format(gctcommon.SimpleTimeFormat),
				base.ErrTooMuchBadData)
		}
		resp = append(resp, data[i])
	}
	return resp, nil
}
