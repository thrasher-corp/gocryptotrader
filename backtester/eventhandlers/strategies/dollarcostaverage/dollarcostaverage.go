package dollarcostaverage

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Name is the strategy name
	Name        = "dollarcostaverage"
	description = `Dollar-cost averaging (DCA) is an investment strategy in which an investor divides up the total amount to be invested across periodic purchases of a target asset in an effort to reduce the impact of volatility on the overall purchase. The purchases occur regardless of the asset's price and at regular intervals. In effect, this strategy removes much of the detailed work of attempting to time the market in order to make purchases of equities at the best prices.`
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
}

// Name returns the name
func (s *Strategy) Name() string {
	return Name
}

// Description provides a nice overview of the strategy
// be it definition of terms or to highlight its purpose
func (s *Strategy) Description() string {
	return description
}

// OnSignal handles a data event and returns what action the strategy believes should occur
// For dollarcostaverage, this means returning a buy signal on every event
func (s *Strategy) OnSignal(d data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) (signal.Event, error) {
	if d == nil {
		return nil, common.ErrNilEvent
	}
	es, err := s.GetBaseData(d)
	if err != nil {
		return nil, err
	}

	latest, err := d.Latest()
	if err != nil {
		return nil, err
	}
	hasDataAtTime, err := d.HasDataAtTime(latest.GetTime())
	if err != nil {
		return nil, err
	}
	if !hasDataAtTime {
		es.SetDirection(order.MissingData)
		es.AppendReasonf("missing data at %v, cannot perform any actions", latest.GetTime())
		return &es, nil
	}

	es.SetPrice(latest.GetClosePrice())
	es.SetDirection(order.Buy)
	es.AppendReason("DCA purchases on every iteration")
	return &es, nil
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return true
}

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
// For dollarcostaverage, the strategy is always "buy", so it uses the OnSignal function
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	var resp []signal.Event
	var errs error
	for i := range d {
		sigEvent, err := s.OnSignal(d[i], nil, nil)
		if err != nil {
			errs = gctcommon.AppendError(errs, err)
		} else {
			resp = append(resp, sigEvent)
		}
	}
	return resp, errs
}

// SetCustomSettings not required for DCA
func (s *Strategy) SetCustomSettings(_ map[string]any) error {
	return base.ErrCustomSettingsUnsupported
}

// SetDefaults not required for DCA
func (s *Strategy) SetDefaults() {}
