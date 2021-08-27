package strongestrsigetstheworm

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Name is the strategy name
	Name                 = "strongestrsigetstheworm"
	rsiPeriodKey         = "rsi-period"
	rsiLowKey            = "rsi-low"
	rsiHighKey           = "rsi-high"
	mandatoryCurrencyKey = "mandatory-currency"
	description          = `This is an example strategy to highlight more complex strategy design`
)

var (
	errStrategyBTCExclusive                       = errors.New("strategy requires all currency pairs contain BTC")
	errStrategyOnlySupportsSimultaneousProcessing = errors.New("strategy only supports simultaneous processing")
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	rsiPeriod         decimal.Decimal
	rsiLow            decimal.Decimal
	rsiHigh           decimal.Decimal
	mandatoryCurrency currency.Code
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
func (s *Strategy) OnSignal(_ data.Handler, _ funding.IFundTransferer) (signal.Event, error) {
	return nil, errStrategyOnlySupportsSimultaneousProcessing
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
// There is nothing actually stopping this strategy from considering multiple currencies at once
// but for demonstration purposes, this strategy does not
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return true
}

type rsiFundEvent struct {
	event signal.Event
	rsi   decimal.Decimal
	funds funding.IPairReader
}

// ByPrice used for sorting orders by order date
type byPrice []rsiFundEvent

func (b byPrice) Len() int           { return len(b) }
func (b byPrice) Less(i, j int) bool { return b[i].rsi.LessThan(b[j].rsi) }
func (b byPrice) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// sortOrdersByPrice the caller function to sort orders
func sortByRSI(o []rsiFundEvent, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(byPrice(o)))
	} else {
		sort.Sort(byPrice(o))
	}
}

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, f funding.IFundTransferer) ([]signal.Event, error) {
	var rsiFundEvents []rsiFundEvent
	var resp []signal.Event
	var errs gctcommon.Errors
	for i := range d {
		p := d[i].Latest().Pair()
		if p.Base != s.mandatoryCurrency && p.Quote != s.mandatoryCurrency {
			return nil, errStrategyBTCExclusive
		}

		if d == nil {
			return nil, common.ErrNilEvent
		}
		es, err := s.GetBaseData(d[i])
		if err != nil {
			return nil, err
		}
		es.SetPrice(d[i].Latest().ClosePrice())
		offset := d[i].Offset()

		if offset <= int(s.rsiPeriod.IntPart()) {
			es.AppendReason("Not enough data for signal generation")
			es.SetDirection(common.DoNothing)
			resp = append(resp, &es)
			continue
		}

		dataRange := d[i].StreamClose()
		var massagedData []float64
		massagedData, err = s.massageMissingData(dataRange, es.GetTime())
		if err != nil {
			return nil, err
		}
		rsi := indicators.RSI(massagedData, int(s.rsiPeriod.IntPart()))
		latestRSIValue := decimal.NewFromFloat(rsi[len(rsi)-1])
		if !d[i].HasDataAtTime(d[i].Latest().GetTime()) {
			es.SetDirection(common.MissingData)
			es.AppendReason(fmt.Sprintf("missing data at %v, cannot perform any actions. RSI %v", d[i].Latest().GetTime(), latestRSIValue))
			resp = append(resp, &es)
			continue
		}

		switch {
		case latestRSIValue.GreaterThanOrEqual(s.rsiHigh):
			es.SetDirection(order.Sell)
		case latestRSIValue.LessThanOrEqual(s.rsiLow):
			es.SetDirection(order.Buy)
		default:
			es.SetDirection(common.DoNothing)
		}
		es.AppendReason(fmt.Sprintf("RSI at %v", latestRSIValue))

		funds, err := f.GetFundingForEvent(&es)
		if err != nil {
			return nil, err
		}
		rsiFundEvents = append(rsiFundEvents, rsiFundEvent{
			event: &es,
			rsi:   latestRSIValue,
			funds: funds,
		})
	}

	if len(rsiFundEvents) == 0 {
		return resp, nil
	}
	sortByRSI(rsiFundEvents, true)
	strongestSignal := rsiFundEvents[0]
	strongestSignalFunds, err := f.GetFundingForEvent(strongestSignal.event)
	if err != nil {
		return nil, err
	}

	if strongestSignal.rsi.GreaterThan(s.rsiHigh) && strongestSignalFunds.Base.MatchesCurrency(currency.BTC) {
		// we are selling, send all matching to the strongest base
		sortByRSI(rsiFundEvents, false)
		if err != nil {
			return nil, err
		}
		for i := range rsiFundEvents {
			if rsiFundEvents[i] == strongestSignal {
				continue
			}
			evFunds, err := f.GetFundingForEvent(rsiFundEvents[i].event)
			if err != nil {
				return nil, err
			}
			if evFunds.Base.MatchesCurrency(s.mandatoryCurrency) {
				baseFunds := evFunds.BaseAvailable()
				if baseFunds.LessThanOrEqual(decimal.Zero) {
					continue
				}
				err = f.Transfer(baseFunds, evFunds.Base, strongestSignalFunds.Base)
				if err != nil {
					return nil, err
				}
				rsiFundEvents[i].event.AppendReason(fmt.Sprintf("sent %v %v funds to %v %v %v",
					baseFunds,
					strongestSignal.event.Pair().Base,
					strongestSignal.event.GetExchange(),
					strongestSignal.event.GetAssetType(),
					strongestSignal.event.Pair()))

				rsiFundEvents[i].event.SetDirection(common.DoNothing)
				strongestSignal.event.AppendReason(fmt.Sprintf("received %v %v funds to sell, from %v %v %v",
					baseFunds,
					rsiFundEvents[i].event.Pair().Base,
					rsiFundEvents[i].event.GetExchange(),
					rsiFundEvents[i].event.GetAssetType(),
					rsiFundEvents[i].event.Pair()))
			} else if evFunds.Quote.MatchesCurrency(s.mandatoryCurrency) {
				quoteFunds := evFunds.QuoteAvailable()
				if quoteFunds.LessThanOrEqual(decimal.Zero) {
					continue
				}
				err = f.Transfer(quoteFunds, evFunds.Quote, strongestSignalFunds.Base)
				if err != nil {
					return nil, err
				}
				rsiFundEvents[i].event.AppendReason(fmt.Sprintf("sent funds %v %v  to %v %v %v",
					quoteFunds,
					strongestSignal.event.Pair().Base,
					strongestSignal.event.GetExchange(),
					strongestSignal.event.GetAssetType(),
					strongestSignal.event.Pair()))

				rsiFundEvents[i].event.SetDirection(common.DoNothing)
				strongestSignal.event.AppendReason(fmt.Sprintf("received funds %v %v  to sell, from %v %v %v",
					quoteFunds,
					rsiFundEvents[i].event.Pair().Quote,
					rsiFundEvents[i].event.GetExchange(),
					rsiFundEvents[i].event.GetAssetType(),
					rsiFundEvents[i].event.Pair()))
			}
		}
	} else if strongestSignal.rsi.LessThan(s.rsiLow) {
		// we are buying, send all matching quote funds to leader
		for i := range rsiFundEvents {
			if rsiFundEvents[i] == strongestSignal {
				continue
			}
			evFunds, err := f.GetFundingForEvent(rsiFundEvents[i].event)
			if err != nil {
				return nil, err
			}
			if evFunds.Base.MatchesItemCurrency(strongestSignalFunds.Quote) {
				baseFunds := evFunds.BaseAvailable()
				if baseFunds.LessThanOrEqual(decimal.Zero) {
					continue
				}
				err = f.Transfer(baseFunds, evFunds.Base, strongestSignalFunds.Quote)
				if err != nil {
					return nil, err
				}
				rsiFundEvents[i].event.AppendReason(fmt.Sprintf("sent %v %v funds to %v %v %v",
					baseFunds,
					strongestSignal.event.Pair().Quote,
					strongestSignal.event.GetExchange(),
					strongestSignal.event.GetAssetType(),
					strongestSignal.event.Pair()))

				rsiFundEvents[i].event.SetDirection(common.DoNothing)
				strongestSignal.event.AppendReason(fmt.Sprintf("received %v %v funds to buy, from %v %v %v",
					baseFunds,
					rsiFundEvents[i].event.Pair().Base,
					rsiFundEvents[i].event.GetExchange(),
					rsiFundEvents[i].event.GetAssetType(),
					rsiFundEvents[i].event.Pair()))
			} else if evFunds.Quote.MatchesItemCurrency(strongestSignalFunds.Quote) {
				quoteFunds := evFunds.QuoteAvailable()
				if quoteFunds.LessThanOrEqual(decimal.Zero) {
					continue
				}
				err = f.Transfer(quoteFunds, evFunds.Quote, strongestSignalFunds.Quote)
				if err != nil {
					return nil, err
				}
				rsiFundEvents[i].event.AppendReason(fmt.Sprintf("sent %v %v funds to %v %v %v",
					quoteFunds,
					strongestSignal.event.Pair().Quote,
					strongestSignal.event.GetExchange(),
					strongestSignal.event.GetAssetType(),
					strongestSignal.event.Pair()))

				rsiFundEvents[i].event.SetDirection(common.DoNothing)
				strongestSignal.event.AppendReason(fmt.Sprintf("received %v %v funds to buy, from %v %v %v",
					quoteFunds,
					rsiFundEvents[i].event.Pair().Quote,
					rsiFundEvents[i].event.GetExchange(),
					rsiFundEvents[i].event.GetAssetType(),
					rsiFundEvents[i].event.Pair()))
			}
		}
	}

	for i := range rsiFundEvents {
		resp = append(resp, rsiFundEvents[i].event)
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return resp, nil
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
		case mandatoryCurrencyKey:
			currStr, ok := v.(string)
			if !ok || currStr == "" {
				return fmt.Errorf("%w mandatory currency is a mandatory field for this strategy, see readme for details: %v", base.ErrInvalidCustomSettings, v)
			}
			s.mandatoryCurrency = currency.NewCode(currStr)
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
	s.mandatoryCurrency = currency.BTC
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
