package top2bottom2

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Name is the strategy name
	Name         = "top2bottom2"
	mfiPeriodKey = "mfi-period"
	mfiLowKey    = "mfi-low"
	mfiHighKey   = "mfi-high"
	description  = `This is an example strategy to highlight more complex strategy design. All signals are processed and then ranked. Only the top 2 and bottom 2 proceed further`
)

var (
	errStrategyOnlySupportsSimultaneousProcessing = errors.New("strategy only supports simultaneous processing")
	errStrategyCurrencyRequirements               = errors.New("top2bottom2 strategy requires at least 4 currencies")
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	mfiPeriod decimal.Decimal
	mfiLow    decimal.Decimal
	mfiHigh   decimal.Decimal
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
// however,this complex strategy cannot function on an individual basis
func (s *Strategy) OnSignal(_ data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) (signal.Event, error) {
	return nil, errStrategyOnlySupportsSimultaneousProcessing
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
// There is nothing actually stopping this strategy from considering multiple currencies at once
// but for demonstration purposes, this strategy does not
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return true
}

type mfiFundEvent struct {
	event signal.Event
	mfi   decimal.Decimal
	funds funding.IFundReader
}

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, f funding.IFundingTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	if len(d) < 4 {
		return nil, errStrategyCurrencyRequirements
	}
	mfiFundEvents := make([]mfiFundEvent, 0, len(d))
	var resp []signal.Event
	for i := range d {
		if d == nil {
			return nil, common.ErrNilEvent
		}
		es, err := s.GetBaseData(d[i])
		if err != nil {
			return nil, err
		}
		latest, err := d[i].Latest()
		if err != nil {
			return nil, err
		}
		es.SetPrice(latest.GetClosePrice())
		offset := latest.GetOffset()

		if offset <= s.mfiPeriod.IntPart() {
			es.AppendReason("Not enough data for signal generation")
			es.SetDirection(order.DoNothing)
			resp = append(resp, &es)
			continue
		}

		history, err := d[i].History()
		if err != nil {
			return nil, err
		}
		var (
			closeData  = make([]decimal.Decimal, len(history))
			volumeData = make([]decimal.Decimal, len(history))
			highData   = make([]decimal.Decimal, len(history))
			lowData    = make([]decimal.Decimal, len(history))
		)
		for i := range history {
			closeData[i] = history[i].GetClosePrice()
			volumeData[i] = history[i].GetVolume()
			highData[i] = history[i].GetHighPrice()
			lowData[i] = history[i].GetLowPrice()
		}
		backfilledCloseData, err := s.backfillMissingData(closeData, es.GetTime())
		if err != nil {
			return nil, err
		}
		backfilledVolumeData, err := s.backfillMissingData(volumeData, es.GetTime())
		if err != nil {
			return nil, err
		}
		backfilledHighData, err := s.backfillMissingData(highData, es.GetTime())
		if err != nil {
			return nil, err
		}
		backfilledLowData, err := s.backfillMissingData(lowData, es.GetTime())
		if err != nil {
			return nil, err
		}
		mfi := indicators.MFI(backfilledHighData, backfilledLowData, backfilledCloseData, backfilledVolumeData, int(s.mfiPeriod.IntPart()))
		latestMFI := decimal.NewFromFloat(mfi[len(mfi)-1])
		hasDataAtTime, err := d[i].HasDataAtTime(latest.GetTime())
		if err != nil {
			return nil, err
		}
		if !hasDataAtTime {
			es.SetDirection(order.MissingData)
			es.AppendReasonf("missing data at %v, cannot perform any actions. MFI %v", latest.GetTime(), latestMFI)
			resp = append(resp, &es)
			continue
		}

		es.SetDirection(order.DoNothing)
		es.AppendReasonf("MFI at %v", latestMFI)

		funds, err := f.GetFundingForEvent(&es)
		if err != nil {
			return nil, err
		}
		mfiFundEvents = append(mfiFundEvents, mfiFundEvent{
			event: &es,
			mfi:   latestMFI,
			funds: funds.FundReader(),
		})
	}

	return s.selectTopAndBottomPerformers(mfiFundEvents, resp)
}

func (s *Strategy) selectTopAndBottomPerformers(mfiFundEvents []mfiFundEvent, resp []signal.Event) ([]signal.Event, error) {
	if len(mfiFundEvents) == 0 {
		return resp, nil
	}
	slices.SortFunc(mfiFundEvents, func(a, b mfiFundEvent) int { return b.mfi.Compare(a.mfi) })
	buyingOrSelling := false
	for i := range mfiFundEvents {
		if i < 2 && mfiFundEvents[i].mfi.GreaterThanOrEqual(s.mfiHigh) {
			mfiFundEvents[i].event.SetDirection(order.Sell)
			buyingOrSelling = true
		} else if i >= 2 {
			break
		}
	}
	slices.Reverse(mfiFundEvents)
	for i := range mfiFundEvents {
		if i < 2 && mfiFundEvents[i].mfi.LessThanOrEqual(s.mfiLow) {
			mfiFundEvents[i].event.SetDirection(order.Buy)
			buyingOrSelling = true
		} else if i >= 2 {
			break
		}
	}
	for i := range mfiFundEvents {
		if buyingOrSelling && mfiFundEvents[i].event.GetDirection() == order.DoNothing {
			mfiFundEvents[i].event.AppendReason("MFI was not in the top or bottom two ranks")
		}
		resp = append(resp, mfiFundEvents[i].event)
	}
	return resp, nil
}

// SetCustomSettings allows a user to modify the MFI limits in their config
func (s *Strategy) SetCustomSettings(customSettings map[string]any) error {
	for k, v := range customSettings {
		switch k {
		case mfiHighKey:
			mfiHigh, ok := v.(float64)
			if !ok || mfiHigh <= 0 {
				return fmt.Errorf("%w provided mfi-high value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.mfiHigh = decimal.NewFromFloat(mfiHigh)
		case mfiLowKey:
			mfiLow, ok := v.(float64)
			if !ok || mfiLow <= 0 {
				return fmt.Errorf("%w provided mfi-low value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.mfiLow = decimal.NewFromFloat(mfiLow)
		case mfiPeriodKey:
			mfiPeriod, ok := v.(float64)
			if !ok || mfiPeriod <= 0 {
				return fmt.Errorf("%w provided mfi-period value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.mfiPeriod = decimal.NewFromFloat(mfiPeriod)
		default:
			return fmt.Errorf("%w unrecognised custom setting key %v with value %v. Cannot apply", base.ErrInvalidCustomSettings, k, v)
		}
	}

	return nil
}

// SetDefaults sets the custom settings to their default values
func (s *Strategy) SetDefaults() {
	s.mfiHigh = decimal.NewFromInt(70)
	s.mfiLow = decimal.NewFromInt(30)
	s.mfiPeriod = decimal.NewFromInt(14)
}

// backfillMissingData will replace missing data with the previous candle's data
// this will ensure that mfi can be calculated correctly
// the decision to handle missing data occurs at the strategy level, not all strategies
// may wish to modify data
func (s *Strategy) backfillMissingData(d []decimal.Decimal, t time.Time) ([]float64, error) {
	resp := make([]float64, len(d))
	var missingDataStreak int64
	for i := range d {
		if d[i].IsZero() && i > int(s.mfiPeriod.IntPart()) {
			d[i] = d[i-1]
			missingDataStreak++
		} else {
			missingDataStreak = 0
		}
		if missingDataStreak >= s.mfiPeriod.IntPart() {
			return nil, fmt.Errorf("missing data exceeds mfi period length of %v at %s and will distort results. %w",
				s.mfiPeriod,
				t.Format(time.DateTime),
				base.ErrTooMuchBadData)
		}
		resp[i] = d[i].InexactFloat64()
	}
	return resp, nil
}
