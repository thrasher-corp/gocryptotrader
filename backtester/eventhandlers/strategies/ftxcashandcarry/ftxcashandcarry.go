package ftxcashandcarry

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Name returns the name of the strategy
func (s *Strategy) Name() string {
	return Name
}

// Description provides a nice overview of the strategy
// be it definition of terms or to highlight its purpose
func (s *Strategy) Description() string {
	return description
}

// OnSignal handles a data event and returns what action the strategy believes should occur
// For rsi, this means returning a buy signal when rsi is at or below a certain level, and a
// sell signal when it is at or above a certain level
func (s *Strategy) OnSignal(data.Handler, funding.IFundTransferer, portfolio.Handler) (signal.Event, error) {
	return nil, base.ErrSimultaneousProcessingOnly
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
// There is nothing actually stopping this strategy from considering multiple currencies at once
// but for demonstration purposes, this strategy does not
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return true
}

type cashCarrySignals struct {
	spotSignal   data.Handler
	futureSignal data.Handler
}

var errNotSetup = errors.New("sent incomplete signals")

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, f funding.IFundTransferer, p portfolio.Handler) ([]signal.Event, error) {
	var response []signal.Event
	sortedSignals, err := sortSignals(d, f)
	if err != nil {
		return nil, err
	}

	for _, v := range sortedSignals {
		pos, err := p.GetPositions(v.futureSignal.Latest())
		if err != nil {
			return nil, err
		}
		spotSignal, err := s.GetBaseData(v.spotSignal)
		if err != nil {
			return nil, err
		}
		futuresSignal, err := s.GetBaseData(v.futureSignal)
		if err != nil {
			return nil, err
		}

		fp := v.futureSignal.Latest().GetClosePrice()
		sp := v.spotSignal.Latest().GetClosePrice()
		switch {
		case len(pos) == 0:
			// check to see if order is appropriate to action
			spotSignal.SetPrice(v.spotSignal.Latest().GetClosePrice())
			spotSignal.AppendReason(fmt.Sprintf("signalling purchase of %v", spotSignal.Pair()))
			// first the spot purchase
			spotSignal.SetDirection(order.Buy)
			// second the futures purchase, using the newly acquired asset
			// as collateral to short
			futuresSignal.SetDirection(order.Short)
			futuresSignal.SetPrice(v.futureSignal.Latest().GetClosePrice())
			futuresSignal.CollateralCurrency = spotSignal.CurrencyPair.Base
			spotSignal.AppendReason(fmt.Sprintf("signalling shorting %v", futuresSignal.Pair()))
			// set the FillDependentEvent to use the futures signal
			// as the futures signal relies on a completed spot order purchase
			// to use as collateral
			spotSignal.FillDependentEvent = &futuresSignal
			response = append(response, &spotSignal)
		case len(pos) > 0 && v.futureSignal.IsLastEvent():
			futuresSignal.SetDirection(common.ClosePosition)
			futuresSignal.AppendReason("closing position on last event")
			futuresSignal.SetDirection(order.Long)
			response = append(response, &futuresSignal)
		case len(pos) > 0 && pos[len(pos)-1].Status == order.Open:
			if fp.Sub(sp).Div(sp).GreaterThan(s.closeShortDistancePercentage) {
				futuresSignal.SetDirection(common.ClosePosition)
				futuresSignal.AppendReason("closing position after reaching close short distance percentage")
				futuresSignal.SetDirection(order.Long)
				response = append(response, &futuresSignal)
			}
		case len(pos) > 0 && pos[len(pos)-1].Status == order.Closed:
			if fp.Sub(sp).Div(sp).GreaterThan(s.openShortDistancePercentage) {
				futuresSignal.SetDirection(order.Short)
				futuresSignal.SetPrice(v.futureSignal.Latest().GetClosePrice())
				futuresSignal.AppendReason("opening position after reaching open short distance percentage")
				response = append(response, &futuresSignal)
			}
		}
	}
	return response, nil
}

func sortSignals(d []data.Handler, f funding.IFundTransferer) (map[currency.Pair]cashCarrySignals, error) {
	var response = make(map[currency.Pair]cashCarrySignals)
	for i := range d {
		l := d[i].Latest()
		if !strings.EqualFold(l.GetExchange(), exchangeName) {
			return nil, fmt.Errorf("%w, received '%v'", errOnlyFTXSupported, l.GetExchange())
		}
		a := l.GetAssetType()
		switch {
		case a == asset.Spot:
			entry := response[l.Pair().Format("", false)]
			entry.spotSignal = d[i]
			response[l.Pair().Format("", false)] = entry
		case a.IsFutures():
			u, err := l.GetUnderlyingPair()
			if err != nil {
				return nil, err
			}
			entry := response[u.Format("", false)]
			entry.futureSignal = d[i]
			response[u.Format("", false)] = entry
		default:
			return nil, errFuturesOnly
		}
	}
	// validate that each set of signals is matched
	for _, v := range response {
		if v.futureSignal == nil {
			return nil, errNotSetup

		}
		if v.spotSignal == nil {
			return nil, errNotSetup
		}
	}

	return response, nil
}

// SetCustomSettings not required for DCA
func (s *Strategy) SetCustomSettings(customSettings map[string]interface{}) error {
	for k, v := range customSettings {
		switch k {
		case openShortDistancePercentageString:
			rsiHigh, ok := v.(float64)
			if !ok || rsiHigh <= 0 {
				return fmt.Errorf("%w provided rsi-high value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.openShortDistancePercentage = decimal.NewFromFloat(rsiHigh)
		case closeShortDistancePercentageString:
			rsiLow, ok := v.(float64)
			if !ok || rsiLow <= 0 {
				return fmt.Errorf("%w provided rsi-low value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.closeShortDistancePercentage = decimal.NewFromFloat(rsiLow)
		default:
			return fmt.Errorf("%w unrecognised custom setting key %v with value %v. Cannot apply", base.ErrInvalidCustomSettings, k, v)
		}
	}

	return nil
}

// SetDefaults not required for DCA
func (s *Strategy) SetDefaults() {
	s.closeShortDistancePercentage = decimal.NewFromInt(5)
	s.closeShortDistancePercentage = decimal.NewFromInt(5)
}
