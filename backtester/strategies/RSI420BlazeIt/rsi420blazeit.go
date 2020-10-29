package RSI420BlazeIt

import (
	"github.com/thrasher-corp/gct-ta/indicators"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/signal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
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
	log.Debugf(log.Global, "STREAM CLOSE at: %v", d.StreamClose())

	rsi := indicators.RSI(d.StreamClose(), 14)
	lastSI := rsi[len(rsi)-1]
	log.Debugf(log.Global, "RSI at: %v", lastSI)
	if lastSI >= 70 {
		es.SetDirection(order.Buy)
	} else if lastSI <= 30 {
		es.SetDirection(order.Sell)
	} else {
		es.SetDirection(common.DoNothing)
	}

	return &es, nil
}
