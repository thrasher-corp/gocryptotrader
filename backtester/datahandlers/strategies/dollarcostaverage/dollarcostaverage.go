package dollarcostaverage

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/signal"
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

	return &es, nil
}
