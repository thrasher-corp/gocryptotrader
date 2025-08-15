package binancecashandcarry

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
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
		return nil, base.ErrNoDataToProcess
	}
	if f == nil {
		return nil, fmt.Errorf("%w missing funding transferred", gctcommon.ErrNilPointer)
	}
	if p == nil {
		return nil, fmt.Errorf("%w missing portfolio handler", gctcommon.ErrNilPointer)
	}
	var response []signal.Event
	sortedSignals, err := sortSignals(d)
	if err != nil {
		return nil, err
	}

	for i := range sortedSignals {
		var latestSpot, latestFuture data.Event
		latestSpot, err = sortedSignals[i].spotSignal.Latest()
		if err != nil {
			return nil, err
		}
		latestFuture, err = sortedSignals[i].futureSignal.Latest()
		if err != nil {
			return nil, err
		}
		var pos []futures.Position
		pos, err = p.GetPositions(latestFuture)
		if err != nil {
			return nil, err
		}
		var spotSignal, futuresSignal signal.Signal
		spotSignal, err = s.GetBaseData(sortedSignals[i].spotSignal)
		if err != nil {
			return nil, err
		}
		futuresSignal, err = s.GetBaseData(sortedSignals[i].futureSignal)
		if err != nil {
			return nil, err
		}

		spotSignal.SetDirection(order.DoNothing)
		futuresSignal.SetDirection(order.DoNothing)
		fp := latestFuture.GetClosePrice()
		sp := latestSpot.GetClosePrice()
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
		var isLastEvent bool
		var signals []signal.Event
		isLastEvent, err = sortedSignals[i].futureSignal.IsLastEvent()
		if err != nil {
			return nil, err
		}
		signals, err = s.createSignals(pos, &spotSignal, &futuresSignal, diffBetweenFuturesSpot, isLastEvent)
		if err != nil {
			return nil, err
		}
		response = append(response, signals...)
	}
	return response, nil
}

// CloseAllPositions is this strategy's implementation on how to
// unwind all positions in the event of a closure
func (s *Strategy) CloseAllPositions(h []holdings.Holding, prices []data.Event) ([]signal.Event, error) {
	var spotSignals, futureSignals []signal.Event
	signalTime := time.Now().UTC()
	for i := range h {
		for j := range prices {
			if prices[j].GetExchange() != h[i].Exchange ||
				prices[j].GetAssetType() != h[i].Asset ||
				!prices[j].Pair().Equal(h[i].Pair) {
				continue
			}
			sig := &signal.Signal{
				Base: &event.Base{
					Offset:         h[i].Offset + 1,
					Exchange:       h[i].Exchange,
					Time:           signalTime,
					Interval:       prices[j].GetInterval(),
					CurrencyPair:   h[i].Pair,
					UnderlyingPair: prices[j].GetUnderlyingPair(),
					AssetType:      h[i].Asset,
					Reasons:        []string{"closing position on close"},
				},
				OpenPrice:          prices[j].GetOpenPrice(),
				HighPrice:          prices[j].GetHighPrice(),
				LowPrice:           prices[j].GetLowPrice(),
				ClosePrice:         prices[j].GetClosePrice(),
				Volume:             prices[j].GetVolume(),
				Amount:             h[i].BaseSize,
				Direction:          order.ClosePosition,
				CollateralCurrency: h[i].Pair.Base,
			}
			if prices[j].GetAssetType().IsFutures() {
				futureSignals = append(futureSignals, sig)
			} else {
				spotSignals = append(spotSignals, sig)
			}
		}
	}
	// close out future positions first
	return append(futureSignals, spotSignals...), nil
}

// createSignals creates signals based on the relationships between
// futures and spot signals
func (s *Strategy) createSignals(pos []futures.Position, spotSignal, futuresSignal *signal.Signal, diffBetweenFuturesSpot decimal.Decimal, isLastEvent bool) ([]signal.Event, error) {
	if spotSignal == nil {
		return nil, fmt.Errorf("%w missing spot signal", gctcommon.ErrNilPointer)
	}
	if futuresSignal == nil {
		return nil, fmt.Errorf("%w missing futures signal", gctcommon.ErrNilPointer)
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
		diffBetweenFuturesSpot.LessThanOrEqual(s.closeShortDistancePercentage):
		// closing positions when custom threshold met
		spotSignal.SetDirection(order.ClosePosition)
		spotSignal.AppendReasonf("Closing position. Met threshold of %v", s.closeShortDistancePercentage)
		futuresSignal.SetDirection(order.ClosePosition)
		futuresSignal.AppendReasonf("Closing position. Met threshold %v", s.closeShortDistancePercentage)
		response = append(response, futuresSignal, spotSignal)
	case pos[len(pos)-1].Status == order.Open &&
		isLastEvent:
		// closing positions on last event
		spotSignal.SetDirection(order.ClosePosition)
		spotSignal.AppendReason("Selling asset on last event")
		futuresSignal.SetDirection(order.ClosePosition)
		futuresSignal.AppendReason("Closing position on last event")
		response = append(response, futuresSignal, spotSignal)
	default:
		response = append(response, spotSignal, futuresSignal)
	}
	return response, nil
}

// sortSignals links spot and futures signals in order to create cash
// and carry signals
func sortSignals(d []data.Handler) ([]cashCarrySignals, error) {
	if len(d) == 0 {
		return nil, base.ErrNoDataToProcess
	}
	carryMap := make(map[*currency.Item]map[*currency.Item]cashCarrySignals, len(d))
	for i := range d {
		l, err := d[i].Latest()
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(l.GetExchange(), exchangeName) {
			return nil, fmt.Errorf("%w, received '%v'", errOnlyBinanceSupported, l.GetExchange())
		}
		a := l.GetAssetType()
		switch {
		case a == asset.Spot:
			b := carryMap[l.Pair().Base.Item]
			if b == nil {
				carryMap[l.Pair().Base.Item] = make(map[*currency.Item]cashCarrySignals)
			}
			entry := carryMap[l.Pair().Base.Item][l.Pair().Quote.Item]
			entry.spotSignal = d[i]
			carryMap[l.Pair().Base.Item][l.Pair().Quote.Item] = entry
		case a.IsFutures():
			u := l.GetUnderlyingPair()
			b := carryMap[u.Base.Item]
			if b == nil {
				carryMap[u.Base.Item] = make(map[*currency.Item]cashCarrySignals)
			}
			entry := carryMap[u.Base.Item][u.Quote.Item]
			entry.futureSignal = d[i]
			carryMap[u.Base.Item][u.Quote.Item] = entry
		default:
			return nil, errFuturesOnly
		}
	}

	var resp []cashCarrySignals
	// validate that each set of signals is matched
	for _, b := range carryMap {
		for _, v := range b {
			if v.futureSignal == nil {
				return nil, fmt.Errorf("%w missing future signal", errNotSetup)
			}
			if v.spotSignal == nil {
				return nil, fmt.Errorf("%w missing spot signal", errNotSetup)
			}
			resp = append(resp, v)
		}
	}

	return resp, nil
}

// SetCustomSettings can override default settings
func (s *Strategy) SetCustomSettings(customSettings map[string]any) error {
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
