package htx

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// appendFuturesCandles normalises the candlestick shape shared by HTX delivery
// and perpetual futures while enforcing the caller's requested time window.
func appendFuturesCandles(destination []kline.Candle, candles []FuturesKline, start, end time.Time) []kline.Candle {
	for i := range candles {
		timestamp := candles[i].IDTimestamp.Time()
		if timestamp.Before(start) || timestamp.After(end) {
			continue
		}
		destination = append(destination, kline.Candle{
			Time:   timestamp,
			Open:   candles[i].Open,
			High:   candles[i].High,
			Low:    candles[i].Low,
			Close:  candles[i].Close,
			Volume: candles[i].Volume,
		})
	}
	return destination
}

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

	const pageSize = uint64(50)
	result := &fundingrate.HistoricalRates{
		Exchange:  e.Name,
		Asset:     r.Asset,
		Pair:      r.Pair,
		StartDate: r.StartDate,
		EndDate:   r.EndDate,
	}
	if r.Asset == asset.USDTMarginedFutures {
		codeValue, err := e.FormatSymbol(r.Pair, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		end := r.EndDate
		if end.IsZero() {
			end = time.Now().UTC()
		}
		start := r.StartDate
		if start.IsZero() {
			start = end.Add(-48 * time.Hour)
		}
		if err := common.StartEndTimeCheck(start, end); err != nil {
			return nil, err
		}
		// Funding history has its own cursor and no documented 90-day order-history cap.
		var from string
		for {
			history, err := e.GetV5FundingRateHistory(ctx, &V5FundingRateHistoryRequest{
				ContractCode: codeValue,
				StartTime:    start,
				EndTime:      end,
				From:         from,
				Limit:        pageSize,
				Direction:    "next",
			})
			if err != nil {
				return nil, err
			}
			if history == nil {
				return nil, errEmptyResult
			}
			for i := range history.FundingRates {
				rateTime := history.FundingRates[i].FundingTime.Time()
				if rateTime.Before(start) || rateTime.After(end) {
					continue
				}
				result.FundingRates = append(result.FundingRates, fundingrate.Rate{
					Time: history.FundingRates[i].FundingTime.Time(),
					Rate: decimal.MustFromFloat(history.FundingRates[i].FundingRate.Float64()),
				})
			}
			if len(history.FundingRates) < int(pageSize) {
				break
			}
			nextFrom := history.FundingRates[len(history.FundingRates)-1].ID
			if nextFrom == from {
				break
			}
			from = nextFrom
		}
	} else {
		for page := uint64(1); ; page++ {
			history, err := e.GetHistoricalFundingRatesForPair(ctx, r.Pair, pageSize, page)
			if err != nil {
				return nil, err
			}
			var reachedStartDate bool
			for _, rate := range history.FundingHistory.FundingRates {
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
					Rate: decimal.MustFromFloat(rate.FundingRate.Float64()),
				})
			}
			if reachedStartDate || history.FundingHistory.TotalPage == 0 || page >= history.FundingHistory.TotalPage {
				break
			}
		}
	}
	slices.SortFunc(result.FundingRates, func(a, b fundingrate.Rate) int { return a.Time.Compare(b.Time) })
	result.FundingRates = slices.CompactFunc(result.FundingRates, func(a, b fundingrate.Rate) bool { return a.Time.Equal(b.Time) })
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
func (e *Exchange) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, amount float64, side order.Side) error {
	if pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	// MaxUint64 rounds up to 2^64 as a float64, so the upper bound must be exclusive.
	if amount <= 0 || amount != math.Trunc(amount) || amount >= math.Exp2(64) {
		return fmt.Errorf("%w: %v", errInvalidLeverage, amount)
	}
	leverage := uint64(amount)
	switch item {
	case asset.Futures:
		if marginType != margin.Isolated && marginType != margin.Unset {
			return fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
		}
		return e.FuturesSwitchLeverage(ctx, pair.Base, leverage)
	case asset.CoinMarginedFutures:
		if marginType != margin.Isolated && marginType != margin.Unset {
			return fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
		}
		return e.SwitchCoinMarginedLeverage(ctx, pair, leverage)
	case asset.USDTMarginedFutures:
		switch marginType {
		case margin.Isolated:
			return e.SwitchLinearSwapLeverage(ctx, pair, leverage, false, side)
		case margin.Unset, margin.Multi:
			return e.SwitchLinearSwapLeverage(ctx, pair, leverage, true, side)
		default:
			return fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

func settlementCurrencyForContract(item asset.Item, pair currency.Pair) (currency.Code, error) {
	if pair.IsEmpty() {
		return currency.EMPTYCODE, currency.ErrCurrencyPairEmpty
	}
	switch item {
	case asset.Futures, asset.CoinMarginedFutures:
		return pair.Base, nil
	case asset.USDTMarginedFutures:
		return pair.Quote, nil
	default:
		return currency.EMPTYCODE, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetCollateralCurrencyForContract returns the currency used as collateral for an HTX contract.
func (e *Exchange) GetCollateralCurrencyForContract(item asset.Item, pair currency.Pair) (currency.Code, asset.Item, error) {
	code, err := settlementCurrencyForContract(item, pair)
	if err != nil {
		return currency.EMPTYCODE, asset.Empty, err
	}
	return code, item, nil
}

// GetCurrencyForRealisedPNL returns the wallet credited with realised contract profit and loss.
func (e *Exchange) GetCurrencyForRealisedPNL(item asset.Item, pair currency.Pair) (currency.Code, asset.Item, error) {
	code, err := settlementCurrencyForContract(item, pair)
	if err != nil {
		return currency.EMPTYCODE, asset.Empty, err
	}
	return code, item, nil
}

// SetCollateralMode changes the account-wide USDT-margined collateral mode.
func (e *Exchange) SetCollateralMode(ctx context.Context, item asset.Item, mode collateral.Mode) error {
	if item != asset.USDTMarginedFutures {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	var assetMode uint64
	switch mode {
	case collateral.MultiMode:
		assetMode = 1
	case collateral.SingleMode:
		assetMode = 2
	default:
		return fmt.Errorf("%w %v", collateral.ErrInvalidCollateralMode, mode)
	}
	resp, err := e.SetV5AssetMode(ctx, assetMode)
	if err != nil {
		return err
	}
	if resp == nil {
		return errEmptyResult
	}
	if resp.Mode.AssetMode != assetMode && (mode != collateral.SingleMode || resp.Mode.AssetMode != 0) {
		return fmt.Errorf("%w: requested %d, received %d", collateral.ErrInvalidCollateralMode, assetMode, resp.Mode.AssetMode)
	}
	return nil
}

// GetCollateralMode returns the account-wide USDT-margined collateral mode.
func (e *Exchange) GetCollateralMode(ctx context.Context, item asset.Item) (collateral.Mode, error) {
	if item != asset.USDTMarginedFutures {
		return collateral.UnsetMode, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	resp, err := e.GetV5AssetMode(ctx)
	if err != nil {
		return collateral.UnsetMode, err
	}
	if resp == nil {
		return collateral.UnsetMode, errEmptyResult
	}
	switch resp.Mode.AssetMode {
	case 1:
		return collateral.MultiMode, nil
	case 0, 2:
		return collateral.SingleMode, nil
	default:
		return collateral.UnsetMode, fmt.Errorf("%w %d", collateral.ErrInvalidCollateralMode, resp.Mode.AssetMode)
	}
}

// GetLeverage gets the configured leverage for an HTX derivatives contract.
func (e *Exchange) GetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, orderSide order.Side) (float64, error) {
	if pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	switch item {
	case asset.Futures:
		if marginType != margin.Isolated && marginType != margin.Unset {
			return 0, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
		}
		account, err := e.FuturesGetAccountInfo(ctx, pair.Base)
		if err != nil {
			return 0, err
		}
		for i := range account.Accounts {
			if account.Accounts[i].Symbol.Equal(pair.Base) {
				return account.Accounts[i].LeverageRate, nil
			}
		}
	case asset.CoinMarginedFutures:
		if marginType != margin.Isolated && marginType != margin.Unset {
			return 0, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
		}
		account, err := e.GetSwapAccountInfo(ctx, pair)
		if err != nil {
			return 0, err
		}
		contractCode, err := e.FormatSymbol(pair, item)
		if err != nil {
			return 0, err
		}
		for i := range account.Accounts {
			if account.Accounts[i].ContractCode == contractCode ||
				(account.Accounts[i].ContractCode == "" && account.Accounts[i].Symbol.Equal(pair.Base)) {
				return account.Accounts[i].LeverageRate, nil
			}
		}
	case asset.USDTMarginedFutures:
		marginMode := marginModeCross
		switch marginType {
		case margin.Unset, margin.Multi:
		case margin.Isolated:
			marginMode = marginModeIsolated
		default:
			return 0, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, marginType)
		}
		var positionSide string
		switch {
		case orderSide == order.UnknownSide:
		case orderSide.IsLong():
			positionSide = "long"
		case orderSide.IsShort():
			positionSide = "short"
		default:
			return 0, order.ErrSideIsInvalid
		}
		// Query every position side so one-way accounts can return their shared "both" setting.
		leverage, err := e.GetV5Leverage(ctx, pair, marginMode, "")
		if err != nil {
			return 0, err
		}
		if leverage == nil {
			return 0, errEmptyResult
		}
		contractCode, err := e.FormatSymbol(pair, item)
		if err != nil {
			return 0, err
		}
		for i := range leverage.Leverage {
			if leverage.Leverage[i].ContractCode == contractCode &&
				leverage.Leverage[i].MarginMode == marginMode &&
				(positionSide == "" || leverage.Leverage[i].PositionSide == "both" || leverage.Leverage[i].PositionSide == positionSide) {
				// Isolated hedge positions have a separate leverage for each side.
				if positionSide == "" && marginMode == marginModeIsolated && leverage.Leverage[i].PositionSide != "both" {
					return 0, order.ErrSideIsInvalid
				}
				return float64(leverage.Leverage[i].LeverageRate), nil
			}
		}
	default:
		return 0, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	return 0, fmt.Errorf("%w %v %s", futures.ErrPositionNotFound, item, pair)
}

// ChangePositionMargin adjusts the margin allocated to an isolated USDT-margined position.
func (e *Exchange) ChangePositionMargin(ctx context.Context, change *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if change == nil {
		return nil, fmt.Errorf("%w PositionChangeRequest", common.ErrNilPointer)
	}
	if change.Asset != asset.USDTMarginedFutures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, change.Asset)
	}
	if change.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if change.MarginType != margin.Isolated {
		return nil, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, change.MarginType)
	}
	if change.NewAllocatedMargin == 0 {
		return nil, margin.ErrNewAllocatedMarginRequired
	}
	if change.OriginalAllocatedMargin == 0 {
		return nil, margin.ErrOriginalPositionMarginRequired
	}
	if change.NewAllocatedMargin == change.OriginalAllocatedMargin {
		return &margin.PositionChangeResponse{
			Exchange:        e.Name,
			Pair:            change.Pair,
			Asset:           change.Asset,
			AllocatedMargin: change.NewAllocatedMargin,
			MarginType:      change.MarginType,
		}, nil
	}
	amount := change.NewAllocatedMargin - change.OriginalAllocatedMargin
	changeType := "add"
	if amount < 0 {
		changeType = "reduce"
		amount = -amount
	}
	positionSide := change.MarginSide
	if positionSide == "" {
		positionSide = "both"
	}
	contractCode, err := e.FormatSymbol(change.Pair, change.Asset)
	if err != nil {
		return nil, err
	}
	if _, err := e.AdjustV5PositionMargin(ctx, &V5AdjustPositionMarginRequest{
		ContractCode: contractCode,
		PositionSide: positionSide,
		Type:         changeType,
		Amount:       types.Number(amount),
	}); err != nil {
		return nil, err
	}
	return &margin.PositionChangeResponse{
		Exchange:        e.Name,
		Pair:            change.Pair,
		Asset:           change.Asset,
		AllocatedMargin: change.NewAllocatedMargin,
		MarginType:      change.MarginType,
	}, nil
}
