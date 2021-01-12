package dollarcostaverage

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Name is the strategy name
const Name = "dollarcostaverage"

type Strategy struct {
	base.Strategy
}

func (s *Strategy) Name() string {
	return Name
}

func (s *Strategy) OnSignal(d data.Handler, _ portfolio.Handler) (signal.SignalEvent, error) {
	if d == nil {
		return nil, errors.New("received nil data")
	}
	es, _ := s.GetBase(d)

	if !d.HasDataAtTime(d.Latest().GetTime()) {
		es.SetDirection(common.MissingData)
		es.AppendWhy(fmt.Sprintf("missing data at %v, cannot perform any actions", d.Latest().GetTime()))
		return &es, nil
	}

	es.SetPrice(d.Latest().Price())
	es.SetDirection(order.Buy)
	es.AppendWhy("DCA purchases on every iteration")
	return &es, nil
}

func (s *Strategy) SupportsMultiCurrency() bool {
	return true
}

// OnSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
// For dollarcostaverage, the strategy is always "buy", so it uses the OnSignal function
func (s *Strategy) OnSignals(d []data.Handler, p portfolio.Handler) ([]signal.SignalEvent, error) {
	var resp []signal.SignalEvent
	for i := range d {
		sigEvent, err := s.OnSignal(d[i], nil)
		if err != nil {
			return nil, err
		}
		resp = append(resp, sigEvent)
	}

	return resp, nil
}

// SetCustomSettings not required for DCA
func (s *Strategy) SetCustomSettings(_ map[string]interface{}) error {
	return errors.New("unsupported")
}

// SetDefaults not required for DCA
func (s *Strategy) SetDefaults() {}
