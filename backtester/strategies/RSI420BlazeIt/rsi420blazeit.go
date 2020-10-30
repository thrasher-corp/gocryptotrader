package RSI420BlazeIt

import (
	"github.com/thrasher-corp/gct-ta/indicators"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/signal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Strategy struct{}

const name = "rsi420blazeit"

func (s *Strategy) Name() string {
	return name
}

func (s *Strategy) OnSignal(d portfolio.DataHandler, _ portfolio2.PortfolioHandler) (signal.SignalEvent, error) {
	es := event.Signal{
		Event: event.Event{Time: d.Latest().GetTime(),
			CurrencyPair: d.Latest().Pair()},
	}
	if d.Offset() <= 14 {
		return &es, nil
	}
	dataRange := d.StreamClose()[:d.Offset()]

	rsi := indicators.RSI(dataRange, 14)
	lastSI := rsi[len(rsi)-1]
	if lastSI >= 70 {
		es.SetDirection(order.Sell)
	} else if lastSI <= 30 {
		es.SetDirection(order.Buy)
	} else {
		es.SetDirection(common.DoNothing)
	}

	return &es, nil
}
