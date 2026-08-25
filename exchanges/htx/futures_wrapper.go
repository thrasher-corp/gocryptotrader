package htx

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"
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

	const pageSize = int64(50)
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
		windows, err := getV3HistoryWindows(r.StartDate, r.EndDate)
		if err != nil {
			return nil, err
		}
		for _, window := range windows {
			var from string
			for {
				history, err := e.GetV5FundingRateHistory(ctx, &V5FundingRateHistoryRequest{
					ContractCode: codeValue,
					StartTime:    window.start,
					EndTime:      window.end,
					From:         from,
					Limit:        uint64(pageSize),
					Direction:    "next",
				})
				if err != nil {
					return nil, err
				}
				for i := range history.Data {
					result.FundingRates = append(result.FundingRates, fundingrate.Rate{
						Time: history.Data[i].FundingTime.Time(),
						Rate: decimal.NewFromFloat(history.Data[i].FundingRate.Float64()),
					})
				}
				if len(history.Data) < int(pageSize) {
					break
				}
				nextFrom := history.Data[len(history.Data)-1].ID
				if nextFrom == from {
					break
				}
				from = nextFrom
			}
		}
	} else {
		for page := int64(1); ; page++ {
			history, err := e.GetHistoricalFundingRatesForPair(ctx, r.Pair, pageSize, page)
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
func (e *Exchange) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, amount float64, side order.Side) error {
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
			return e.SwitchLinearSwapLeverage(ctx, pair, leverage, false, side)
		case margin.Multi:
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
	switch resp.Data.AssetMode {
	case 1:
		return collateral.MultiMode, nil
	case 0, 2:
		return collateral.SingleMode, nil
	default:
		return collateral.UnsetMode, fmt.Errorf("%w %d", collateral.ErrInvalidCollateralMode, resp.Data.AssetMode)
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
		account, err := e.FGetAccountInfo(ctx, pair.Base)
		if err != nil {
			return 0, err
		}
		for i := range account.AccData {
			if account.AccData[i].Symbol.Equal(pair.Base) {
				return account.AccData[i].LeverageRate, nil
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
		for i := range account.Data {
			if account.Data[i].ContractCode == contractCode ||
				(account.Data[i].ContractCode == "" && account.Data[i].Symbol.Equal(pair.Base)) {
				return account.Data[i].LeverageRate, nil
			}
		}
	case asset.USDTMarginedFutures:
		marginMode := "cross"
		switch marginType {
		case margin.Unset, margin.Multi:
		case margin.Isolated:
			marginMode = "isolated"
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
		leverage, err := e.GetV5Leverage(ctx, pair, marginMode, positionSide)
		if err != nil {
			return 0, err
		}
		contractCode, err := e.FormatSymbol(pair, item)
		if err != nil {
			return 0, err
		}
		for i := range leverage.Data {
			if leverage.Data[i].ContractCode == contractCode &&
				leverage.Data[i].MarginMode == marginMode &&
				(positionSide == "" || leverage.Data[i].PositionSide == positionSide) {
				return float64(leverage.Data[i].LeverageRate), nil
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
