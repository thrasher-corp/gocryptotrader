package dollarcostaverage

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Name is the strategy name
const Name = "dollarcostaverage"

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
}

// Name returns the name
func (s *Strategy) Name() string {
	return Name
}

// OnSignal handles a data event and returns what action the strategy believes should occur
// For dollarcostaverage, this means returning a buy signal on every event
func (s *Strategy) OnSignal(d data.Handler, _ portfolio.Handler) (signal.Event, error) {
	if d == nil {
		return nil, common.ErrNilEvent
	}
	es, err := s.GetBase(d)
	if err != nil {
		return nil, err
	}

	if !d.HasDataAtTime(d.Latest().GetTime()) {
		es.SetDirection(common.MissingData)
		es.AppendReason(fmt.Sprintf("missing data at %v, cannot perform any actions", d.Latest().GetTime()))
		return &es, nil
	}

	es.SetPrice(d.Latest().ClosePrice())
	es.SetDirection(order.Buy)
	es.AppendReason("DCA purchases on every iteration")
	return &es, nil
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return true
}

// OnSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
// For dollarcostaverage, the strategy is always "buy", so it uses the OnSignal function
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, p portfolio.Handler) ([]signal.Event, error) {
	var resp []signal.Event
	var errs gctcommon.Errors
	for i := range d {
		sigEvent, err := s.OnSignal(d[i], nil)
		if err != nil {
			errs = append(errs, err)
		} else {
			resp = append(resp, sigEvent)
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return resp, nil
}

// SetCustomSettings not required for DCA
func (s *Strategy) SetCustomSettings(_ map[string]interface{}) error {
	return base.ErrCustomSettingsUnsupported
}

// SetDefaults not required for DCA
func (s *Strategy) SetDefaults() {}
