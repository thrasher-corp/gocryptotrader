package htx

import (
	"context"
	"fmt"
	"math"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// GetHistoricalFundingRates returns historical funding rates for coin- and USDT-margined perpetual contracts.
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.CoinMarginedFutures && r.Asset != asset.USDTMarginedFutures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !r.StartDate.IsZero() && !r.EndDate.IsZero() {
		if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
			return nil, err
		}
	}
	if r.IncludePayments {
		return nil, fmt.Errorf("include payments %w", common.ErrNotYetImplemented)
	}

	const pageSize = int64(50)
	result := &fundingrate.HistoricalRates{
		Exchange:  e.Name,
		Asset:     r.Asset,
		Pair:      r.Pair,
		StartDate: r.StartDate,
		EndDate:   r.EndDate,
	}
	for page := int64(1); ; page++ {
		var history HistoricalFundingRateData
		var err error
		switch r.Asset {
		case asset.CoinMarginedFutures:
			history, err = e.GetHistoricalFundingRatesForPair(ctx, r.Pair, pageSize, page)
		case asset.USDTMarginedFutures:
			history, err = e.GetLinearSwapHistoricalFundingRates(ctx, r.Pair, pageSize, page)
		}
		if err != nil {
			return nil, err
		}
		var reachedStartDate bool
		for _, rate := range history.Data.Data {
			rateTime := rate.FundingTime.Time()
			if !r.StartDate.IsZero() && rateTime.Before(r.StartDate) {
				reachedStartDate = true
				continue
			}
			if !r.EndDate.IsZero() && rateTime.After(r.EndDate) {
				continue
			}
			result.FundingRates = append(result.FundingRates, fundingrate.Rate{
				Time: rateTime,
				Rate: decimal.NewFromFloat(rate.FundingRate.Float64()),
			})
		}
		if reachedStartDate || history.Data.TotalPage == 0 || page >= history.Data.TotalPage {
			break
		}
	}
	if len(result.FundingRates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}
	if r.IncludePredictedRate {
		latest, err := e.GetLatestFundingRates(ctx, &fundingrate.LatestRateRequest{
			Asset:                r.Asset,
			Pair:                 r.Pair,
			IncludePredictedRate: true,
		})
		if err != nil {
			return nil, err
		}
		if len(latest) != 0 {
			result.LatestRate = latest[0].LatestRate
			result.PredictedUpcomingRate = latest[0].PredictedUpcomingRate
			result.TimeOfNextRate = latest[0].TimeOfNextRate
		}
	}
	return result, nil
}

// SetLeverage changes the account leverage for an HTX derivatives contract.
func (e *Exchange) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, amount float64, _ order.Side) error {
	if pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if amount <= 0 || amount != math.Trunc(amount) || amount > math.MaxUint64 {
		return fmt.Errorf("%w: %v", errInvalidLeverage, amount)
	}
	leverage := uint64(amount)
	switch item {
	case asset.Futures:
		if marginType != margin.Isolated && marginType != margin.Unset {
			return fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
		}
		return e.FSwitchLeverage(ctx, pair.Base, leverage)
	case asset.CoinMarginedFutures:
		if marginType != margin.Isolated && marginType != margin.Unset {
			return fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
		}
		return e.SwitchCoinMarginedLeverage(ctx, pair, leverage)
	case asset.USDTMarginedFutures:
		switch marginType {
		case margin.Isolated, margin.Unset:
			return e.SwitchLinearSwapLeverage(ctx, pair, leverage, false)
		case margin.Multi:
			return e.SwitchLinearSwapLeverage(ctx, pair, leverage, true)
		default:
			return fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}
