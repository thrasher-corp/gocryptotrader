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

// Description describes the strategy
func (s *Strategy) Description() string {
	return description
}

// OnSignal handles a data event and returns what action the strategy believes should occur
// For rsi, this means returning a buy signal when rsi is at or below a certain level, and a
// sell signal when it is at or above a certain level
func (s *Strategy) OnSignal(data.Handler, funding.IFundingTransferer, portfolio.Handler) (signal.Event, error) {
	return nil, base.ErrSimultaneousProcessingOnly
}

// SupportsSimultaneousProcessing this strategy only supports simultaneous signal processing
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
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, f funding.IFundingTransferer, p portfolio.Handler) ([]signal.Event, error) {
	if len(d) == 0 {
		return nil, errNoSignals
	}
	if f == nil {
		return nil, fmt.Errorf("%w missing funding transferred", common.ErrNilArguments)
	}
	if p == nil {
		return nil, fmt.Errorf("%w missing portfolio handler", common.ErrNilArguments)
	}
	var response []signal.Event
	sortedSignals, err := sortSignals(d)
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

		spotSignal.SetDirection(order.DoNothing)
		futuresSignal.SetDirection(order.DoNothing)
		fp := v.futureSignal.Latest().GetClosePrice()
		sp := v.spotSignal.Latest().GetClosePrice()
		diffBetweenFuturesSpot := fp.Sub(sp).Div(sp).Mul(decimal.NewFromInt(100))
		futuresSignal.AppendReasonf("Futures Spot Difference: %v%%", diffBetweenFuturesSpot)
		if len(pos) > 0 && pos[len(pos)-1].Status == order.Open {
			futuresSignal.AppendReasonf("Unrealised PNL: %v %v", pos[len(pos)-1].UnrealisedPNL, pos[len(pos)-1].CollateralCurrency)
		}
		if f.HasExchangeBeenLiquidated(&spotSignal) || f.HasExchangeBeenLiquidated(&futuresSignal) {
			spotSignal.AppendReason("cannot transact, has been liquidated")
			futuresSignal.AppendReason("cannot transact, has been liquidated")
			response = append(response, &spotSignal, &futuresSignal)
			continue
		}
		signals, err := s.createSignals(pos, &spotSignal, &futuresSignal, diffBetweenFuturesSpot, v.futureSignal.IsLastEvent())
		if err != nil {
			return nil, err
		}
		response = append(response, signals...)
	}
	return response, nil
}

// createSignals creates signals based on the relationships between
// futures and spot signals
func (s *Strategy) createSignals(pos []order.PositionStats, spotSignal, futuresSignal *signal.Signal, diffBetweenFuturesSpot decimal.Decimal, isLastEvent bool) ([]signal.Event, error) {
	if spotSignal == nil {
		return nil, fmt.Errorf("%w missing spot signal", common.ErrNilArguments)
	}
	if futuresSignal == nil {
		return nil, fmt.Errorf("%w missing futures signal", common.ErrNilArguments)
	}
	var response []signal.Event
	switch {
	case len(pos) == 0,
		pos[len(pos)-1].Status == order.Closed &&
			diffBetweenFuturesSpot.GreaterThan(s.openShortDistancePercentage):
		// check to see if order is appropriate to action
		spotSignal.SetPrice(spotSignal.ClosePrice)
		spotSignal.AppendReasonf("Signalling purchase of %v", spotSignal.Pair())
		// first the spot purchase
		spotSignal.SetDirection(order.Buy)
		// second the futures purchase, using the newly acquired asset
		// as collateral to short
		futuresSignal.SetDirection(order.Short)
		futuresSignal.SetPrice(futuresSignal.ClosePrice)
		futuresSignal.AppendReason("Shorting to perform cash and carry")
		futuresSignal.CollateralCurrency = spotSignal.CurrencyPair.Base
		futuresSignal.MatchesOrderAmount = true
		spotSignal.AppendReasonf("Signalling shorting of %v after spot order placed", futuresSignal.Pair())
		// set the FillDependentEvent to use the futures signal
		// as the futures signal relies on a completed spot order purchase
		// to use as collateral
		spotSignal.FillDependentEvent = futuresSignal
		// only appending spotSignal as futuresSignal will be raised later
		response = append(response, spotSignal)
	case pos[len(pos)-1].Status == order.Open &&
		isLastEvent:
		// closing positions on last event
		spotSignal.SetDirection(order.ClosePosition)
		spotSignal.AppendReason("Selling asset on last event")
		futuresSignal.SetDirection(order.ClosePosition)
		futuresSignal.AppendReason("Closing position on last event")
		response = append(response, futuresSignal, spotSignal)
	case pos[len(pos)-1].Status == order.Open &&
		diffBetweenFuturesSpot.LessThanOrEqual(s.closeShortDistancePercentage):
		// closing positions when custom threshold met
		spotSignal.SetDirection(order.ClosePosition)
		spotSignal.AppendReasonf("Closing position. Met threshold of %v", s.closeShortDistancePercentage)
		futuresSignal.SetDirection(order.ClosePosition)
		futuresSignal.AppendReasonf("Closing position. Met threshold %v", s.closeShortDistancePercentage)
		response = append(response, futuresSignal, spotSignal)
	default:
		response = append(response, spotSignal, futuresSignal)
	}
	return response, nil
}

// sortSignals links spot and futures signals in order to create cash
// and carry signals
func sortSignals(d []data.Handler) (map[currency.Pair]cashCarrySignals, error) {
	if len(d) == 0 {
		return nil, errNoSignals
	}
	var response = make(map[currency.Pair]cashCarrySignals, len(d))
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
			u := l.GetUnderlyingPair()
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
			return nil, fmt.Errorf("%w missing future signal", errNotSetup)
		}
		if v.spotSignal == nil {
			return nil, fmt.Errorf("%w missing spot signal", errNotSetup)
		}
	}

	return response, nil
}

// SetCustomSettings can override default settings
func (s *Strategy) SetCustomSettings(customSettings map[string]interface{}) error {
	for k, v := range customSettings {
		switch k {
		case openShortDistancePercentageString:
			osdp, ok := v.(float64)
			if !ok || osdp <= 0 {
				return fmt.Errorf("%w provided openShortDistancePercentage value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.openShortDistancePercentage = decimal.NewFromFloat(osdp)
		case closeShortDistancePercentageString:
			csdp, ok := v.(float64)
			if !ok || csdp <= 0 {
				return fmt.Errorf("%w provided closeShortDistancePercentage value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.closeShortDistancePercentage = decimal.NewFromFloat(csdp)
		default:
			return fmt.Errorf("%w unrecognised custom setting key %v with value %v. Cannot apply", base.ErrInvalidCustomSettings, k, v)
		}
	}

	return nil
}

// SetDefaults sets default values for overridable custom settings
func (s *Strategy) SetDefaults() {
	s.openShortDistancePercentage = decimal.Zero
	s.closeShortDistancePercentage = decimal.Zero
}
