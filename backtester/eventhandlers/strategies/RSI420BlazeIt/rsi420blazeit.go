package RSI420BlazeIt

import (
	"fmt"

	"github.com/thrasher-corp/gct-ta/indicators"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const name = "rsi420blazeit"

type Strategy struct {
	base.Strategy
}

func (s *Strategy) Name() string {
	return name
}

func (s *Strategy) OnSignal(d portfolio.DataHandler, _ portfolio2.PortfolioHandler) (signal.SignalEvent, error) {
	es := s.GetBase(d)
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
	es.SetWhy(fmt.Sprintf("RSI at %v", lastSI))

	return &es, nil
}
