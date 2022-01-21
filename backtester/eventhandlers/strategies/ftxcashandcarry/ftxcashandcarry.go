package ftxcashandcarry

import (
	"errors"
	"fmt"
	"strings"

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
func (s *Strategy) OnSignal(d data.Handler, _ funding.IFundTransferer, p portfolio.Handler) (signal.Event, error) {
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

func sortSignals(d []data.Handler) (map[currency.Pair]cashCarrySignals, error) {
	var response = make(map[currency.Pair]cashCarrySignals)
	for i := range d {
		l := d[i].Latest()
		if !strings.EqualFold(l.GetExchange(), exchangeName) {
			return nil, fmt.Errorf("%w, received '%v'", errOnlyFTXSupported, l.GetExchange())
		}
		a := l.GetAssetType()
		switch {
		case a == asset.Spot:
			entry := response[l.Pair()]
			entry.spotSignal = d[i]
			response[l.Pair()] = entry
		case a.IsFutures():
			u, err := l.GetUnderlyingPair()
			if err != nil {
				return nil, err
			}
			entry := response[u]
			entry.futureSignal = d[i]
			response[l.Pair()] = entry
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

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, _ funding.IFundTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	var response []signal.Event
	sortedSignals, err := sortSignals(d)
	if err != nil {
		return nil, err
	}
	for _, v := range sortedSignals {
		spotSignal, err := s.GetBaseData(v.spotSignal)
		if err != nil {
			return nil, err
		}
		futuresSignal, err := s.GetBaseData(v.futureSignal)
		if err != nil {
			return nil, err
		}
		spotSignal.SetPrice(v.spotSignal.Latest().GetClosePrice())
		futuresSignal.SetPrice(v.futureSignal.Latest().GetClosePrice())
		// first the spot purchase
		spotSignal.SetDirection(order.Buy)
		// second the futures purchase, using the newly acquired asset
		// as collateral to short
		futuresSignal.SetDirection(order.Sell)
		futuresSignal.RequiresCollateral = true
		futuresSignal.CollateralCurrency = futuresSignal.UnderlyingPair.Base
		response = append(response, &spotSignal, &futuresSignal)
	}
	return response, nil
}

// SetCustomSettings not required for DCA
func (s *Strategy) SetCustomSettings(_ map[string]interface{}) error {
	return base.ErrCustomSettingsUnsupported
}

// SetDefaults not required for DCA
func (s *Strategy) SetDefaults() {}
