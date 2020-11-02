package buyandhold

import (
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/signal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Strategy struct{}

const name = "buyandhold"

func (s *Strategy) Name() string {
	return name
}

func (s *Strategy) OnSignal(d portfolio.DataHandler, p portfolio2.PortfolioHandler) (signal.SignalEvent, error) {
	es := signal.Signal{
		Event: event.Event{Time: d.Latest().GetTime(),
			CurrencyPair: d.Latest().Pair()},
	}

	es.SetPrice(d.Latest().LatestPrice())
	es.SetDirection(order.Buy)
	return &es, nil
}
