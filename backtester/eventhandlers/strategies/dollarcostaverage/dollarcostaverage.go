package dollarcostaverage

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const name = "dollarcostaverage"

type Strategy struct {
	base.Strategy
}

func (s *Strategy) Name() string {
	return name
}

func (s *Strategy) OnSignal(d data.Handler, p portfolio.Handler) (signal.SignalEvent, error) {
	es := s.GetBase(d)

	es.SetPrice(d.Latest().Price())
	es.SetDirection(order.Buy)
	es.AppendWhy("DCA purchases on every iteration")
	return &es, nil
}

func (s *Strategy) SupportsMultiCurrency() bool {
	return true
}

func (s *Strategy) OnSignals(d []data.Handler, p portfolio.Handler) ([]signal.SignalEvent, error) {
	var resp []signal.SignalEvent
	for i := range d {
		es := s.GetBase(d[i])
		es.SetPrice(d[i].Latest().Price())
		es.SetDirection(order.Buy)
		es.AppendWhy("DCA purchases on every iteration")
		resp = append(resp, &es)
	}

	return resp, nil
}

// SetCustomSettings not required for DCA
func (s *Strategy) SetCustomSettings(_ map[string]interface{}) error {
	return nil
}

// SetDefaults not required for DCA
func (s *Strategy) SetDefaults() {
}
