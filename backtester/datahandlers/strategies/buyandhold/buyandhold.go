package buyandhold

import (
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/signal"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const name = "buyandhold"

type Strategy struct {
	base.Strategy
}

func (s *Strategy) Name() string {
	return name
}

func (s *Strategy) OnSignal(d portfolio.DataHandler, p portfolio2.PortfolioHandler) (signal.SignalEvent, error) {
	es := s.GetBase(d)
	es.SetPrice(d.Latest().LatestPrice())
	es.SetDirection(order.Buy)

	return &es, nil
}
