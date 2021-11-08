package futures

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Name is the strategy name
	Name                 = "futures-rsi"
	rsiPeriodKey         = "rsi-period"
	rsiLowKey            = "rsi-low"
	rsiHighKey           = "rsi-high"
	rsiStopLoss          = "rsi-stop-loss"
	rsiTakeProfit        = "rsi-take-profit"
	rsiTrailingStop      = "rsi-trailing-stop"
	rsiHighestUnrealised = "rsi-highest-unrealised"
	rsiLowestUnrealised  = "rsi-lowest-unrealised"
	description          = `The relative strength index is a technical indicator used in the analysis of financial markets. It is intended to chart the current and historical strength or weakness of a stock or market based on the closing prices of a recent trading period`
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	rsiPeriod         decimal.Decimal
	rsiLow            decimal.Decimal
	rsiHigh           decimal.Decimal
	stopLoss          decimal.Decimal
	takeProfit        decimal.Decimal
	trailingStop      decimal.Decimal
	highestUnrealised decimal.Decimal
	lowestUnrealised  decimal.Decimal
}

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
	if d == nil {
		return nil, common.ErrNilEvent
	}
	latest := d.Latest()
	if latest.GetAssetType() != asset.Futures {
		return nil, errors.New("can only work with futures")
	}

	es, err := s.GetBaseData(d)
	if err != nil {
		return nil, err
	}
	es.SetPrice(d.Latest().ClosePrice())

	if offset := d.Offset(); offset <= int(s.rsiPeriod.IntPart()) {
		es.AppendReason("Not enough data for signal generation")
		es.SetDirection(common.DoNothing)
		return &es, nil
	}

	dataRange := d.StreamClose()
	var massagedData []float64
	massagedData, err = s.massageMissingData(dataRange, es.GetTime())
	if err != nil {
		return nil, err
	}
	rsi := indicators.RSI(massagedData, int(s.rsiPeriod.IntPart()))
	latestRSIValue := decimal.NewFromFloat(rsi[len(rsi)-1])
	if !d.HasDataAtTime(d.Latest().GetTime()) {
		es.SetDirection(common.MissingData)
		es.AppendReason(fmt.Sprintf("missing data at %v, cannot perform any actions. RSI %v", d.Latest().GetTime(), latestRSIValue))
		return &es, nil
	}

	currentOrders, err := p.GetLatestOrderSnapshotForEvent(&es)
	if err != nil {
		return nil, err
	}

	var unrealisedOrder *order.Detail
	for i := range currentOrders.Orders {
		if currentOrders.Orders[i].FuturesOrder != nil {
			if currentOrders.Orders[i].FuturesOrder.ClosingPosition == nil {
				if currentOrders.Orders[i].FuturesOrder.Side == order.Short || currentOrders.Orders[i].FuturesOrder.Side == order.Long {
					unrealisedOrder = currentOrders.Orders[i].FuturesOrder.OpeningPosition
				}
			}
		}
	}
	if unrealisedOrder == nil {
		switch {
		case latestRSIValue.GreaterThanOrEqual(s.rsiHigh):
			es.SetDirection(order.Short)
		case latestRSIValue.LessThanOrEqual(s.rsiLow):
			es.SetDirection(order.Long)
		default:
			es.SetDirection(common.DoNothing)
		}
		es.AppendReason(fmt.Sprintf("RSI at %v", latestRSIValue))
	} else {
		p := decimal.NewFromFloat(unrealisedOrder.Price)
		if latestRSIValue.LessThanOrEqual(s.rsiLow) ||
			latestRSIValue.GreaterThanOrEqual(s.rsiHigh) ||
			(!s.stopLoss.IsZero() && latest.ClosePrice().LessThanOrEqual(s.stopLoss)) ||
			(!s.takeProfit.IsZero() && latest.ClosePrice().GreaterThanOrEqual(s.takeProfit)) ||
			(!s.trailingStop.IsZero() && latest.ClosePrice().Sub(p).Div(p).Mul(decimal.NewFromInt(100)).LessThanOrEqual(s.trailingStop)) ||
			unrealisedOrder.UnrealisedPNL.GreaterThanOrEqual(s.highestUnrealised) ||
			unrealisedOrder.UnrealisedPNL.LessThanOrEqual(s.lowestUnrealised) {
			// set up the counter order to close the position
			es.SetAmount(decimal.NewFromFloat(unrealisedOrder.Amount))
			if unrealisedOrder.Side == order.Short {
				es.SetDirection(order.Long)
			} else if unrealisedOrder.Side == order.Long {
				es.SetDirection(order.Short)
			}
			es.SetCloseOrderID(unrealisedOrder.ID)
		}
	}

	return &es, nil
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
// There is nothing actually stopping this strategy from considering multiple currencies at once
// but for demonstration purposes, this strategy does not
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return false
}

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, _ funding.IFundTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	return nil, base.ErrSimultaneousProcessingNotSupported
}

// SetCustomSettings allows a user to modify the RSI limits in their config
func (s *Strategy) SetCustomSettings(customSettings map[string]interface{}) error {
	for k, v := range customSettings {
		switch k {
		case rsiHighKey:
			rsiHigh, ok := v.(float64)
			if !ok || rsiHigh <= 0 {
				return fmt.Errorf("%w provided rsi-high value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.rsiHigh = decimal.NewFromFloat(rsiHigh)
		case rsiLowKey:
			rsiLow, ok := v.(float64)
			if !ok || rsiLow <= 0 {
				return fmt.Errorf("%w provided rsi-low value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.rsiLow = decimal.NewFromFloat(rsiLow)
		case rsiPeriodKey:
			rsiPeriod, ok := v.(float64)
			if !ok || rsiPeriod <= 0 {
				return fmt.Errorf("%w provided rsi-period value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.rsiPeriod = decimal.NewFromFloat(rsiPeriod)
		case rsiStopLoss:
			sl, ok := v.(float64)
			if !ok || sl <= 0 {
				return fmt.Errorf("%w provided rsi-period value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.stopLoss = decimal.NewFromFloat(sl)
		case rsiTakeProfit:
			tp, ok := v.(float64)
			if !ok || tp <= 0 {
				return fmt.Errorf("%w provided rsi-period value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.takeProfit = decimal.NewFromFloat(tp)
		case rsiTrailingStop:
			ts, ok := v.(float64)
			if !ok || ts <= 0 {
				return fmt.Errorf("%w provided rsi-period value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.trailingStop = decimal.NewFromFloat(ts)
		case rsiHighestUnrealised:
			ts, ok := v.(float64)
			if !ok || ts <= 0 {
				return fmt.Errorf("%w provided rsi-period value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.highestUnrealised = decimal.NewFromFloat(ts)
		case rsiLowestUnrealised:
			ts, ok := v.(float64)
			if !ok || ts <= 0 {
				return fmt.Errorf("%w provided rsi-period value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.lowestUnrealised = decimal.NewFromFloat(ts)
		default:
			return fmt.Errorf("%w unrecognised custom setting key %v with value %v. Cannot apply", base.ErrInvalidCustomSettings, k, v)
		}
	}

	return nil
}

// SetDefaults sets the custom settings to their default values
func (s *Strategy) SetDefaults() {
	s.rsiHigh = decimal.NewFromInt(70)
	s.rsiLow = decimal.NewFromInt(30)
	s.rsiPeriod = decimal.NewFromInt(14)
}

// massageMissingData will replace missing data with the previous candle's data
// this will ensure that RSI can be calculated correctly
// the decision to handle missing data occurs at the strategy level, not all strategies
// may wish to modify data
func (s *Strategy) massageMissingData(data []decimal.Decimal, t time.Time) ([]float64, error) {
	var resp []float64
	var missingDataStreak int64
	for i := range data {
		if data[i].IsZero() && i > int(s.rsiPeriod.IntPart()) {
			data[i] = data[i-1]
			missingDataStreak++
		} else {
			missingDataStreak = 0
		}
		if missingDataStreak >= s.rsiPeriod.IntPart() {
			return nil, fmt.Errorf("missing data exceeds RSI period length of %v at %s and will distort results. %w",
				s.rsiPeriod,
				t.Format(gctcommon.SimpleTimeFormat),
				base.ErrTooMuchBadData)
		}
		d, _ := data[i].Float64()
		resp = append(resp, d)
	}
	return resp, nil
}
