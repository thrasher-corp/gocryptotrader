package arbitrage

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Name is the strategy name
	Name        = "arbitrage"
	minProfitKey = "min-profit"
	description = `TODO To demonstrate the arbitrage strategy using API tick data and custom settings`
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	minProfit decimal.Decimal
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
// For arbitrage, this means returning a buy or signal on every event based on difference in price
func (s *Strategy) OnSignal(d data.Handler, _ funding.IFundTransferer) (signal.Event, error) {return nil, nil}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return true
}

// OnSimultaneousSignals analyses multiple data points simultaneously.
// For arbitrage, the strategy executes two trades for the pair in opposite directions
// if the close price difference between the pair is greater than min-profit, e.g., 0.3%
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, _ funding.IFundTransferer) ([]signal.Event, error) {
	var resp []signal.Event
	var errs gctcommon.Errors
	var d1, d2 data.Handler
	
	if len(d) == 2 {
		d1 = d[0]
		d2 = d[1]
	} else {
		d1 = nil
		d2 = nil
		return nil, common.ErrNilArguments
	}
	

	if d1 == nil || d2 == nil {
		return nil, common.ErrNilEvent
	}
	es1, err1 := s.GetBaseData(d1)
	if err1 != nil {
		return nil, err1
	}

	es2, err2 := s.GetBaseData(d2)
	if err2 != nil {
		return nil, err2
	}

	if !d1.HasDataAtTime(d1.Latest().GetTime()) {
		es1.SetDirection(common.MissingData)
		es1.AppendReason(fmt.Sprintf("missing data at %v, cannot perform any actions", d1.Latest().GetTime()))
		resp = append(resp, &es1)
		return resp, nil
	}

	if !d2.HasDataAtTime(d2.Latest().GetTime()) {
		es2.SetDirection(common.MissingData)
		es2.AppendReason(fmt.Sprintf("missing data at %v, cannot perform any actions", d2.Latest().GetTime()))
		resp = append(resp, &es2)
		return resp, nil
	}

	latestReturnValue := s.calculateReturns(d1, d2)

	switch {
	// Case where price for d1 is bigger than d2 and min profit
	case latestReturnValue[0].GreaterThanOrEqual(latestReturnValue[1]) && latestReturnValue[0].GreaterThanOrEqual(s.minProfit):
		es1.SetDirection(order.Sell)
		es2.SetDirection(order.Buy)
	// Case where price for d2 is bigger than d1 and min profit
	case latestReturnValue[1].GreaterThanOrEqual(latestReturnValue[0]) && latestReturnValue[1].GreaterThanOrEqual(s.minProfit):
		es1.SetDirection(order.Buy)
		es2.SetDirection(order.Sell)
	default:
		es1.SetDirection(common.DoNothing)
		es2.SetDirection(common.DoNothing)
	}
	es1.AppendReason(fmt.Sprintf("Pair return difference at %v", latestReturnValue))
	es2.AppendReason(fmt.Sprintf("Pair return difference at %v", latestReturnValue))


	if err1 != nil || err2 != nil {
		errs = append(errs, fmt.Errorf("%v %v %v %w", d1.Latest().GetExchange(), d1.Latest().GetAssetType(), d1.Latest().Pair(), err1))
		errs = append(errs, fmt.Errorf("%v %v %v %w", d2.Latest().GetExchange(), d2.Latest().GetAssetType(), d2.Latest().Pair(), err2))
	} else {
		resp = append(resp, &es1)
		resp = append(resp, &es2)
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return resp, nil
}

// SetCustomSettings allows a user to modify the arbitrage parameters in their config
func (s *Strategy) SetCustomSettings(customSettings map[string]interface{}) error {
	for k, v := range customSettings {
		switch k {
		case minProfitKey:
			minProfit, ok := v.(float64)
			if !ok || minProfit <= 0 {
				return fmt.Errorf("%w provided min-profit value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.minProfit = decimal.NewFromFloat(minProfit)
		default:
			return fmt.Errorf("%w unrecognised custom setting key %v with value %v. Cannot apply", base.ErrInvalidCustomSettings, k, v)
		}
	}

	return nil
}

// SetDefaults sets the custom settings to their default values
func (s *Strategy) SetDefaults() {
	s.minProfit = decimal.NewFromFloat(0.3)
}

// calculateReturns Calculates the profitability of crossing the exchanges in both directions (buy on exchange 2 + sell
// on exchange 1 | buy on exchange 1 + sell on exchange 2) using the last candle close price on each.
func (s *Strategy) calculateReturns(d1 data.Handler, d2 data.Handler) ([]decimal.Decimal) {
	var ret []decimal.Decimal
	d1_close := d1.Latest().ClosePrice()
	d2_close := d2.Latest().ClosePrice()
	return_d1_vs_d2 := d1_close.Div(d2_close.Add(decimal.NewFromFloat(1)))
	return_d2_vs_d1 := d2_close.Div(d1_close.Add(decimal.NewFromFloat(1)))
	ret = append(ret, return_d1_vs_d2)
	ret = append(ret, return_d2_vs_d1)
	return ret
}


