package rsi

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Name is the strategy name
const (
	Name         = "rsi"
	rsiPeriodKey = "rsi-period"
	rsiLowKey    = "rsi-low"
	rsiHighKey   = "rsi-high"
)

type Strategy struct {
	base.Strategy
	rsiPeriod float64
	rsiLow    float64
	rsiHigh   float64
}

func (s *Strategy) Name() string {
	return Name
}

func (s *Strategy) OnSignal(d data.Handler, _ portfolio.Handler) (signal.Event, error) {
	if d == nil {
		return nil, errors.New("received nil data")
	}
	es, _ := s.GetBase(d)
	es.SetPrice(d.Latest().Price())

	if !d.HasDataAtTime(d.Latest().GetTime()) {
		es.SetDirection(common.MissingData)
		es.AppendWhy(fmt.Sprintf("missing data at %v, cannot perform any actions", d.Latest().GetTime()))
		return &es, nil
	}

	if d.Offset() <= int(s.rsiPeriod) {
		es.AppendWhy("Not enough data for signal generation")
		return &es, errors.New(es.Why)
	}
	dataRange := d.StreamClose()[:d.Offset()]

	rsi := indicators.RSI(dataRange, int(s.rsiPeriod))
	lastSI := rsi[len(rsi)-1]
	switch {
	case lastSI >= s.rsiHigh:
		es.SetDirection(order.Sell)
	case lastSI <= s.rsiLow:
		es.SetDirection(order.Buy)
	default:
		es.SetDirection(common.DoNothing)
	}
	es.AppendWhy(fmt.Sprintf("RSI at %.2f", lastSI))

	return &es, nil
}

// SupportsMultiCurrency highlights whether the strategy can handle multiple currency calculation
// There is nothing actually stopping this strategy from considering multiple currencies at once
// but for demonstration purposes, this strategy does not
func (s *Strategy) SupportsMultiCurrency() bool {
	return false
}

// OnSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
// For rsi, multi-currency signal processing is unsupported for demonstration purposes
func (s *Strategy) OnSignals(_ []data.Handler, _ portfolio.Handler) ([]signal.Event, error) {
	return nil, errors.New("unsupported")
}

func (s *Strategy) SetCustomSettings(customSettings map[string]interface{}) error {
	for k, v := range customSettings {
		switch k {
		case rsiHighKey:
			rsiHigh, ok := v.(float64)
			if !ok {
				return fmt.Errorf("provided rsi-high value could not be parsed: %v", v)
			}
			s.rsiHigh = rsiHigh
		case rsiLowKey:
			rsiLow, ok := v.(float64)
			if !ok {
				return fmt.Errorf("provided rsi-low value could not be parsed: %v", v)
			}
			s.rsiLow = rsiLow
		case rsiPeriodKey:
			rsiPeriod, ok := v.(float64)
			if !ok {
				return fmt.Errorf("provided rsi-period value could not be parsed: %v", v)
			}
			s.rsiPeriod = rsiPeriod
		default:
			return fmt.Errorf("unrecognised custom setting key %v with value %v. Cannot apply", k, v)
		}
	}

	return nil
}

func (s *Strategy) SetDefaults() {
	s.rsiHigh = 70
	s.rsiLow = 30
	s.rsiPeriod = 14
}
