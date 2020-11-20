package dollarcostaverage

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const name = "dollarcostaverage"

type Strategy struct {
	base.Strategy
}

func (s *Strategy) Name() string {
	return name
}

func (s *Strategy) OnSignal(d interfaces.DataHandler, p portfolio.PortfolioHandler) (signal.SignalEvent, error) {
	es := s.GetBase(d)

	es.SetPrice(d.Latest().Price())
	es.SetDirection(order.Buy)
	es.SetWhy("DCA purchases on every iteration")
	return &es, nil
}

// SetCustomSettings not required for DCA
func (s *Strategy) SetCustomSettings(_ map[string]interface{}) error {
	return nil
}

// SetDefaults not required for DCA
func (s *Strategy) SetDefaults() {
}
