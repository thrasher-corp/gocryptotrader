package RSI420BlazeIt

import (
	"fmt"

	"github.com/thrasher-corp/gct-ta/indicators"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const name = "rsi420blazeit"

type Strategy struct {
	base.Strategy
	rsiPeriod float64
	rsiLow    float64
	rsiHigh   float64
}

func (s *Strategy) Name() string {
	return name
}

func (s *Strategy) OnSignal(d interfaces.DataHandler, _ portfolio2.PortfolioHandler) (signal.SignalEvent, error) {
	es := s.GetBase(d)
	es.SetPrice(d.Latest().Price())

	if d.Offset() <= int(s.rsiPeriod) {
		return &es, nil
	}
	dataRange := d.StreamClose()[:d.Offset()]

	rsi := indicators.RSI(dataRange, int(s.rsiPeriod))
	lastSI := rsi[len(rsi)-1]
	if lastSI >= s.rsiHigh {
		es.SetDirection(order.Sell)
	} else if lastSI <= s.rsiLow {
		es.SetDirection(order.Buy)
	} else {
		es.SetDirection(common.DoNothing)
	}
	es.SetWhy(fmt.Sprintf("RSI at %v", lastSI))

	return &es, nil
}

func (s *Strategy) SetCustomSettings(customSettings map[string]interface{}) error {
	if rsiLowInterface, ok := customSettings["rsi-low"]; ok {
		rsiLow, ok := rsiLowInterface.(float64)
		if !ok {
			return fmt.Errorf("provided rsi-low value could not be parsed: %v", rsiLowInterface)
		}
		s.rsiLow = rsiLow
	}

	if rsiHighInterface, ok := customSettings["rsi-high"]; ok {
		rsiHigh, ok := rsiHighInterface.(float64)
		if !ok {
			return fmt.Errorf("provided rsi-high value could not be parsed: %v", rsiHighInterface)
		}
		s.rsiHigh = rsiHigh
	}

	if rsiPeriodInterface, ok := customSettings["rsi-period"]; ok {
		rsiPeriod, ok := rsiPeriodInterface.(float64)
		if !ok {
			return fmt.Errorf("provided rsi-period value could not be parsed: %v", rsiPeriodInterface)
		}
		s.rsiPeriod = rsiPeriod
	}

	return nil
}

func (s *Strategy) SetDefaults() {
	s.rsiHigh = 70
	s.rsiLow = 30
	s.rsiPeriod = 14
}
