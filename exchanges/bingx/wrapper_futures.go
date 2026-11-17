package bingx

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const (
	// fundingRateHistoryLimit is the most funding rate records a request returns
	fundingRateHistoryLimit = 1000
	// fundingFlowLimit is the most fund flow records a request returns
	fundingFlowLimit = 1000
	// fundingFlowRetention is how long BingX keeps the fund flow records funding payments come from
	fundingFlowRetention = 90 * 24 * time.Hour
	// fundingFlowWindow is short enough for a window's funding fees to fit one request: a contract pays at most two a
	// funding time, one for each hedge mode side, and funding times are at least an hour apart
	fundingFlowWindow = time.Hour * fundingFlowLimit / 2
)

var (
	errInvalidLeverage           = errors.New("leverage must be a whole number of at least 1")
	errPositionDirectionRequired = errors.New("position direction required")
	errMarginChangeRequired      = errors.New("the new allocated margin equals the original")
	errFundingPaymentsNotKept    = errors.New("funding payments start before the fund flow BingX keeps")
	errInvalidMarginAmount       = errors.New("invalid margin amount")
)

// GetFuturesContractDetails returns the contracts of a futures asset with their funding rate limits and current rates:
// USDT- and USDC-settled linear perpetuals, settled and margined in their quote, and inverse coin-M perpetuals,
// settled and margined in their base, whose multiplier is a contract's USD value
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	switch {
	case !item.IsFutures():
		return nil, futures.ErrNotFuturesAsset
	case item != asset.USDTMarginedFutures && item != asset.CoinMarginedFutures:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, item)
	}
	format, err := e.GetPairFormat(item, false)
	if err != nil {
		return nil, err
	}
	switch item {
	case asset.USDTMarginedFutures:
		contracts, err := e.GetSwapContracts(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		rates, err := e.GetSwapMarkPriceAndFundingRate(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		bySymbol := make(map[string]*SwapMarkPriceAndFundingRateResponse, len(rates))
		for i := range rates {
			bySymbol[rates[i].Symbol] = &rates[i]
		}
		resp := make([]futures.Contract, 0, len(contracts))
		for i := range contracts {
			c := &contracts[i]
			name, err := symbolPair(c.Symbol, format)
			if err != nil {
				e.logOrderIssue(item, c.Symbol, err)
				continue
			}
			contract := futures.Contract{
				Exchange:           e.Name,
				Name:               name,
				Underlying:         currency.NewPair(c.Asset, c.Currency),
				Asset:              item,
				StartDate:          c.LaunchTime.Time(),
				IsActive:           c.Status == SwapContractStatusOnline && c.APIStateOpen.Bool(),
				Status:             swapContractStatusString(c.Status),
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: c.Currency,
				MarginCurrency:     c.Currency,
			}
			if r, ok := bySymbol[c.Symbol]; ok {
				contract.FundingRateFloor = r.MinFundingRate.Decimal()
				contract.FundingRateCeiling = r.MaxFundingRate.Decimal()
				contract.LatestRate = fundingrate.Rate{Time: swapFundingPeriodStart(r), Rate: r.LastFundingRate.Decimal()}
			}
			resp = append(resp, contract)
		}
		return resp, nil
	default:
		contracts, err := e.GetCoinMarginedContracts(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		rates, err := e.GetCoinMarginedPriceAndFundingRate(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		bySymbol := make(map[string]*CoinMarginedPriceAndFundingRateResponse, len(rates))
		for i := range rates {
			bySymbol[rates[i].Symbol] = &rates[i]
		}
		resp := make([]futures.Contract, 0, len(contracts))
		for i := range contracts {
			c := &contracts[i]
			name, err := symbolPair(c.Symbol, format)
			if err != nil {
				e.logOrderIssue(item, c.Symbol, err)
				continue
			}
			contract := futures.Contract{
				Exchange:           e.Name,
				Name:               name,
				Underlying:         name,
				Asset:              item,
				StartDate:          c.TimeOnline.Time(),
				IsActive:           c.Status == coinMarginedContractStatusOnline,
				Status:             strconv.FormatUint(c.Status, 10),
				Type:               futures.Perpetual,
				SettlementType:     futures.Inverse,
				SettlementCurrency: name.Base,
				MarginCurrency:     name.Base,
				Multiplier:         c.MinTickSize.Float64(),
			}
			if r, ok := bySymbol[c.Symbol]; ok {
				// Coin-M reports no funding interval, so the start of the period the rate applies to is unknown
				contract.LatestRate = fundingrate.Rate{Rate: r.LastFundingRate.Decimal()}
			}
			resp = append(resp, contract)
		}
		return resp, nil
	}
}

// swapContractStatusString returns the name of a USDT-M contract status, or the code of an undocumented one
func swapContractStatusString(status SwapContractStatus) string {
	switch status {
	case SwapContractStatusOffline:
		return "offline"
	case SwapContractStatusOnline:
		return "online"
	case SwapContractStatusPreOnline:
		return "pre-online"
	case SwapContractStatusForbiddenToOpenPositions:
		return "forbidden to open positions"
	default:
		return strconv.FormatUint(uint64(status), 10)
	}
}

// swapFundingPeriodStart returns the start of the funding period a USDT-M contract's current rate applies to
func swapFundingPeriodStart(r *SwapMarkPriceAndFundingRateResponse) time.Time {
	return r.NextFundingTime.Time().Add(-swapFundingInterval(r))
}

// swapFundingInterval returns a USDT-M contract's funding interval, at most the longest duration
func swapFundingInterval(r *SwapMarkPriceAndFundingRateResponse) time.Duration {
	return time.Duration(min(r.FundingIntervalHours, uint64(math.MaxInt64/time.Hour))) * time.Hour
}

// IsPerpetualFutureCurrency reports whether a pair is a perpetual future, which every BingX futures contract is
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	switch a {
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		if cp.IsEmpty() {
			return false, currency.ErrCurrencyPairEmpty
		}
		return true, nil
	case asset.Spot:
		return false, nil
	default:
		return false, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
}

// GetCollateralCurrencyForContract returns the currency a contract is margined in: a USDT-M contract's settlement
// currency, the quote its symbol ends in, or a coin-M contract's base coin
func (e *Exchange) GetCollateralCurrencyForContract(a asset.Item, cp currency.Pair) (currency.Code, asset.Item, error) {
	return contractSettlement(a, cp)
}

// GetCurrencyForRealisedPNL returns the currency and asset a contract's realised profit and loss is paid in: its
// settlement currency, into the perpetual account it trades in
func (e *Exchange) GetCurrencyForRealisedPNL(a asset.Item, cp currency.Pair) (currency.Code, asset.Item, error) {
	return contractSettlement(a, cp)
}

// contractSettlement returns a contract's settlement currency and the asset its account is: a USDT-M contract's quote,
// as every USDT-M symbol ends in its settlement currency, or a coin-M contract's base
func contractSettlement(a asset.Item, cp currency.Pair) (currency.Code, asset.Item, error) {
	if cp.IsEmpty() {
		return currency.EMPTYCODE, asset.Empty, currency.ErrCurrencyPairEmpty
	}
	switch a {
	case asset.USDTMarginedFutures:
		return cp.Quote, a, nil
	case asset.CoinMarginedFutures:
		return cp.Base, a, nil
	case asset.Spot:
		return currency.EMPTYCODE, asset.Empty, futures.ErrNotFuturesAsset
	default:
		return currency.EMPTYCODE, asset.Empty, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
}

// GetLatestFundingRates returns the current funding rate of a perpetual, or of every enabled perpetual of the asset
// without a pair, from one request. BingX's current rate accrues until the next funding time settles it, so it is the
// latest rate, of the period ending then, and the predicted rate of the next funding time. Coin-M reports no funding
// interval, so a coin-M rate's period start is unknown and its latest rate is undated
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	var resp []fundingrate.LatestRateResponse
	checked := time.Now()
	switch r.Asset {
	case asset.USDTMarginedFutures:
		rates, err := e.GetSwapMarkPriceAndFundingRate(ctx, r.Pair)
		if err != nil {
			return nil, err
		}
		resp = make([]fundingrate.LatestRateResponse, 0, len(rates))
		for i := range rates {
			pair, ok, err := e.fundingRatePair(r, rates[i].Symbol)
			if err != nil {
				return nil, err
			}
			if ok {
				resp = append(resp, latestFundingRate(e.Name, r, pair, swapFundingPeriodStart(&rates[i]), rates[i].NextFundingTime.Time(), rates[i].LastFundingRate, checked))
			}
		}
	case asset.CoinMarginedFutures:
		rates, err := e.GetCoinMarginedPriceAndFundingRate(ctx, r.Pair)
		if err != nil {
			return nil, err
		}
		resp = make([]fundingrate.LatestRateResponse, 0, len(rates))
		for i := range rates {
			pair, ok, err := e.fundingRatePair(r, rates[i].Symbol)
			if err != nil {
				return nil, err
			}
			if ok {
				resp = append(resp, latestFundingRate(e.Name, r, pair, time.Time{}, rates[i].NextFundingTime.Time(), rates[i].LastFundingRate, checked))
			}
		}
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, r.Asset)
	}
	return resp, nil
}

// fundingRatePair returns the pair of a funding rate: the requested pair, or the enabled pair of the symbol when every
// symbol's rate was requested, and false for a symbol without one
func (e *Exchange) fundingRatePair(r *fundingrate.LatestRateRequest, symbol string) (currency.Pair, bool, error) {
	if !r.Pair.IsEmpty() {
		return r.Pair, true, nil
	}
	pair, ok, err := e.availablePair(symbol, r.Asset)
	if err != nil || !ok {
		return currency.EMPTYPAIR, false, err
	}
	enabled, err := e.IsPairEnabled(pair, r.Asset)
	return pair, enabled, err
}

// latestFundingRate returns a perpetual's current funding rate as its latest rate, of the period from periodStart to
// nextFundingTime, and, when requested, as the predicted rate settled at nextFundingTime
func latestFundingRate(exchangeName string, r *fundingrate.LatestRateRequest, pair currency.Pair, periodStart, nextFundingTime time.Time, rate types.Number, checked time.Time) fundingrate.LatestRateResponse {
	resp := fundingrate.LatestRateResponse{
		Exchange:       exchangeName,
		Asset:          r.Asset,
		Pair:           pair,
		LatestRate:     fundingrate.Rate{Time: periodStart, Rate: rate.Decimal()},
		TimeOfNextRate: nextFundingTime,
		TimeChecked:    checked,
	}
	if r.IncludePredictedRate {
		resp.PredictedUpcomingRate = fundingrate.Rate{Time: nextFundingTime, Rate: rate.Decimal()}
	}
	return resp
}

// GetHistoricalFundingRates returns a USDT-M perpetual's settled funding rates from StartDate to EndDate, paging back
// from the latest, with its current rate as the latest rate and, when requested, the predicted rate and the funding
// fees paid or received, which BingX keeps for 90 days. Coin-M serves no funding rate history
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch r.Asset {
	case asset.USDTMarginedFutures:
	case asset.CoinMarginedFutures:
		return nil, fmt.Errorf("%w: BingX serves no %s funding rate history", common.ErrFunctionNotSupported, r.Asset)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	settlement := r.Pair.Quote
	if !r.PaymentCurrency.IsEmpty() && !r.PaymentCurrency.Equal(settlement) {
		return nil, fmt.Errorf("%w: %s funding is paid in %s", common.ErrFunctionNotSupported, r.Pair, settlement)
	}
	resp := &fundingrate.HistoricalRates{
		Exchange:  e.Name,
		Asset:     r.Asset,
		Pair:      r.Pair,
		StartDate: r.StartDate,
		EndDate:   r.EndDate,
	}
	var err error
	if resp.FundingRates, err = e.swapFundingRates(ctx, r.Pair, r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	current, err := e.GetSwapMarkPriceAndFundingRate(ctx, r.Pair)
	if err != nil {
		return nil, err
	}
	if len(current) == 0 {
		return nil, fmt.Errorf("%w: no current rate for %s", fundingrate.ErrNoFundingRatesFound, r.Pair)
	}
	resp.LatestRate = fundingrate.Rate{Time: swapFundingPeriodStart(&current[0]), Rate: current[0].LastFundingRate.Decimal()}
	resp.TimeOfNextRate = current[0].NextFundingTime.Time()
	if r.IncludePredictedRate {
		resp.PredictedUpcomingRate = fundingrate.Rate{Time: resp.TimeOfNextRate, Rate: resp.LatestRate.Rate}
	}
	if !r.IncludePayments {
		return resp, nil
	}
	paymentsStart := r.StartDate
	if kept := time.Now().Add(-fundingFlowRetention); paymentsStart.Before(kept) {
		if !r.RespectHistoryLimits {
			return nil, fmt.Errorf("%w: %s is before %s", errFundingPaymentsNotKept, r.StartDate.UTC(), kept.UTC())
		}
		paymentsStart = kept
	}
	resp.PaymentCurrency = settlement
	interval := swapFundingInterval(&current[0])
	for windowStart := paymentsStart; windowStart.Before(r.EndDate); windowStart = windowStart.Add(fundingFlowWindow) {
		flows, err := e.GetSwapFundFlow(ctx, &SwapFundFlowRequest{
			Symbol:     r.Pair,
			Currency:   settlement,
			IncomeType: IncomeTypeFundingFee,
			StartTime:  windowStart,
			EndTime:    earlierTime(windowStart.Add(fundingFlowWindow), r.EndDate),
			Limit:      fundingFlowLimit,
		})
		if err != nil {
			return nil, err
		}
		for i := range flows {
			addFundingPayment(resp, flows[i].Time.Time(), flows[i].Income.Decimal(), interval)
		}
	}
	return resp, nil
}

// swapFundingRates returns a USDT-M perpetual's funding rates from start to end in ascending time. BingX returns the
// latest records of a range, newest first and both bounds included, so pages are requested back from end until one
// is empty or reaches start, as a page can come back short of the limit with older records still in the range
func (e *Exchange) swapFundingRates(ctx context.Context, pair currency.Pair, start, end time.Time) ([]fundingrate.Rate, error) {
	var rates []fundingrate.Rate
	for pageEnd := end; !pageEnd.Before(start); {
		page, err := e.GetSwapFundingRateHistory(ctx, &SwapFundingRateHistoryRequest{Symbol: pair, StartTime: start, EndTime: pageEnd, Limit: fundingRateHistoryLimit})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		for i := range page {
			rates = append(rates, fundingrate.Rate{Time: page[i].FundingTime.Time(), Rate: page[i].FundingRate.Decimal()})
		}
		oldest := slices.MinFunc(page, func(a, b SwapFundingRateHistoryResponse) int {
			return a.FundingTime.Time().Compare(b.FundingTime.Time())
		})
		next := oldest.FundingTime.Time().Add(-time.Millisecond)
		if !next.Before(pageEnd) {
			break
		}
		pageEnd = next
	}
	slices.SortFunc(rates, func(a, b fundingrate.Rate) int { return a.Time.Compare(b.Time) })
	return slices.CompactFunc(rates, func(a, b fundingrate.Rate) bool { return a.Time.Equal(b.Time) }), nil
}

// addFundingPayment adds a funding fee to the rate it was paid at, the one nearest it, and to the payment sum. Fees more
// than half a funding interval from every rate fall outside the rates fetched. The rates are in ascending time order
func addFundingPayment(resp *fundingrate.HistoricalRates, at time.Time, amount decimal.Decimal, interval time.Duration) {
	i, _ := slices.BinarySearchFunc(resp.FundingRates, at, func(r fundingrate.Rate, t time.Time) int { return r.Time.Compare(t) })
	nearest := -1
	for _, j := range []int{i - 1, i} {
		if j < 0 || j >= len(resp.FundingRates) {
			continue
		}
		if nearest < 0 || resp.FundingRates[j].Time.Sub(at).Abs() < resp.FundingRates[nearest].Time.Sub(at).Abs() {
			nearest = j
		}
	}
	if nearest < 0 || resp.FundingRates[nearest].Time.Sub(at).Abs() >= interval/2 {
		return
	}
	resp.FundingRates[nearest].Payment = resp.FundingRates[nearest].Payment.Add(amount)
	resp.PaymentSum = resp.PaymentSum.Add(amount)
}

// GetOpenInterest returns the open interest of each pair, or of every enabled pair of both futures assets without
// any: USDT-M open interest is its value in the settlement currency, as BingX reports it, and coin-M open interest is
// in the base coin. Coin-M pairs come from one request for every symbol when more than one is asked for
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.USDTMarginedFutures && k[i].Asset != asset.CoinMarginedFutures {
			// Checked before any request, so no request is made for a list that cannot be answered in full
			return nil, fmt.Errorf("%w: %s %s", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	if len(k) == 0 {
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			pairs, err := e.GetEnabledPairs(a)
			if errors.Is(err, asset.ErrNotEnabled) {
				// A disabled asset has no enabled pairs
				continue
			}
			if err != nil {
				return nil, err
			}
			for _, p := range pairs {
				k = append(k, key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a})
			}
		}
	}
	var coinMarginedPairs int
	for i := range k {
		if k[i].Asset == asset.CoinMarginedFutures {
			coinMarginedPairs++
		}
	}
	var coinMargined map[string]*CoinMarginedOpenInterestResponse
	if coinMarginedPairs > 1 {
		all, err := e.GetCoinMarginedOpenInterest(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}
		coinMargined = make(map[string]*CoinMarginedOpenInterestResponse, len(all))
		for i := range all {
			coinMargined[all[i].Symbol] = &all[i]
		}
	}
	resp := make([]futures.OpenInterest, 0, len(k))
	for i := range k {
		pair := k[i].Pair()
		var openInterest float64
		if k[i].Asset == asset.USDTMarginedFutures {
			oi, err := e.GetSwapOpenInterest(ctx, pair)
			if err != nil {
				return nil, err
			}
			openInterest = oi.OpenInterest.Float64()
		} else {
			symbol, err := e.FormatSymbol(pair, asset.CoinMarginedFutures)
			if err != nil {
				return nil, err
			}
			oi, ok := coinMargined[symbol]
			if coinMargined == nil {
				listed, err := e.GetCoinMarginedOpenInterest(ctx, pair)
				if err != nil {
					return nil, err
				}
				if len(listed) != 0 {
					oi, ok = &listed[0], true
				}
			}
			if !ok {
				return nil, fmt.Errorf("%w: no %s open interest for %s", currency.ErrPairNotFound, k[i].Asset, pair)
			}
			openInterest = oi.OpenInterest.Float64()
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, k[i].Asset, pair),
			OpenInterest: openInterest,
		})
	}
	return resp, nil
}

// leveragePositionSides returns the position sides a leverage change applies to: BOTH in USDT-M one-way mode, and in
// hedge mode, which coin-M always takes, the side of the order direction, or both sides without one
func (e *Exchange) leveragePositionSides(ctx context.Context, a asset.Item, side order.Side) ([]PositionSide, error) {
	if a == asset.USDTMarginedFutures {
		hedgeMode, err := e.swapHedgeMode(ctx)
		if err != nil {
			return nil, err
		}
		if !hedgeMode {
			return []PositionSide{PositionSideBoth}, nil
		}
	}
	switch {
	case side.IsLong():
		return []PositionSide{PositionSideLong}, nil
	case side.IsShort():
		return []PositionSide{PositionSideShort}, nil
	default:
		return []PositionSide{PositionSideLong, PositionSideShort}, nil
	}
}

// SetLeverage sets a perpetual's leverage, a whole number, for the position sides leveragePositionSides returns; BingX
// sets leverage per symbol, whatever the margin type. A USDT-M change BingX refuses for the account's position mode is
// made once more after looking the mode up again, as a change of mode made elsewhere leaves the cached one stale. Every
// request goes to the account of the credentials ctx names when the change starts
func (e *Exchange) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, amount float64, side order.Side) error {
	if item != asset.USDTMarginedFutures && item != asset.CoinMarginedFutures {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, item)
	}
	if pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if amount < 1 || amount != math.Trunc(amount) || amount > math.MaxUint32 {
		return fmt.Errorf("%w: %v", errInvalidLeverage, amount)
	}
	ctx, err := e.credentialsContext(ctx)
	if err != nil {
		return err
	}
	err = e.setPerpetualLeverage(ctx, item, pair, uint64(amount), side)
	if item == asset.USDTMarginedFutures && isPositionModeRejection(err) {
		e.forgetSwapPositionMode(ctx)
		err = e.setPerpetualLeverage(ctx, item, pair, uint64(amount), side)
	}
	return err
}

// setPerpetualLeverage sets a perpetual's leverage for each position side leveragePositionSides returns
func (e *Exchange) setPerpetualLeverage(ctx context.Context, item asset.Item, pair currency.Pair, leverage uint64, side order.Side) error {
	sides, err := e.leveragePositionSides(ctx, item, side)
	if err != nil {
		return err
	}
	for _, positionSide := range sides {
		if item == asset.USDTMarginedFutures {
			_, err = e.SetSwapLeverage(ctx, pair, positionSide, leverage)
		} else {
			_, err = e.SetCoinMarginedLeverage(ctx, pair, positionSide, leverage)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// GetLeverage returns a perpetual's leverage for the side of the order direction; without a direction the long and
// short leverage must agree, as they do in one-way mode
func (e *Exchange) GetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, side order.Side) (float64, error) {
	if item != asset.USDTMarginedFutures && item != asset.CoinMarginedFutures {
		return 0, fmt.Errorf("%w: %s", asset.ErrNotSupported, item)
	}
	if pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	var long, short uint64
	if item == asset.USDTMarginedFutures {
		resp, err := e.GetSwapLeverage(ctx, pair)
		if err != nil {
			return 0, err
		}
		long, short = resp.LongLeverage, resp.ShortLeverage
	} else {
		resp, err := e.GetCoinMarginedLeverage(ctx, pair)
		if err != nil {
			return 0, err
		}
		long, short = resp.LongLeverage, resp.ShortLeverage
	}
	switch {
	case side.IsLong():
		return float64(long), nil
	case side.IsShort():
		return float64(short), nil
	case long == short:
		return float64(long), nil
	default:
		return 0, fmt.Errorf("%w: %s %s has long leverage %d and short leverage %d", errPositionDirectionRequired, item, pair, long, short)
	}
}

// formatMarginType returns the margin type BingX takes: isolated, or cross margin for GCT's multi margin
func formatMarginType(mt margin.Type) (MarginType, error) {
	switch mt {
	case margin.Isolated:
		return MarginTypeIsolated, nil
	case margin.Multi:
		return MarginTypeCrossed, nil
	default:
		return "", fmt.Errorf("%w: %s", margin.ErrInvalidMarginType, mt)
	}
}

// convertMarginType converts a margin type BingX reports: isolated, separate isolated, which keeps each position's
// margin apart, or cross margin
func convertMarginType(mt MarginType) (margin.Type, error) {
	switch mt {
	case MarginTypeIsolated, MarginTypeSeparateIsolated:
		return margin.Isolated, nil
	case MarginTypeCrossed:
		return margin.Multi, nil
	default:
		return margin.Unset, fmt.Errorf("%w: %q", margin.ErrInvalidMarginType, mt)
	}
}

// SetMarginType sets a perpetual's margin type, isolated or cross
func (e *Exchange) SetMarginType(ctx context.Context, item asset.Item, pair currency.Pair, tp margin.Type) error {
	if item != asset.USDTMarginedFutures && item != asset.CoinMarginedFutures {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, item)
	}
	mt, err := formatMarginType(tp)
	if err != nil {
		return err
	}
	if item == asset.USDTMarginedFutures {
		return e.ChangeSwapMarginType(ctx, pair, mt)
	}
	return e.SetCoinMarginedMarginType(ctx, pair, mt)
}

// ChangePositionMargin adds margin to or removes margin from an isolated position: the difference between its
// original and its new allocated margin. MarginSide is the position side, LONG or SHORT in hedge mode, which coin-M
// always takes, and BOTH or empty in USDT-M one-way mode
func (e *Exchange) ChangePositionMargin(ctx context.Context, req *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Asset != asset.USDTMarginedFutures && req.Asset != asset.CoinMarginedFutures {
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.Asset)
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.MarginType == margin.Multi {
		return nil, fmt.Errorf("%w: cross margin positions share the account's margin", margin.ErrMarginTypeUnsupported)
	}
	if req.NewAllocatedMargin == 0 {
		return nil, margin.ErrNewAllocatedMarginRequired
	}
	if req.OriginalAllocatedMargin == 0 {
		return nil, margin.ErrOriginalPositionMarginRequired
	}
	for _, m := range []float64{req.NewAllocatedMargin, req.OriginalAllocatedMargin} {
		if math.IsNaN(m) || math.IsInf(m, 0) {
			return nil, fmt.Errorf("%w: %v", errInvalidMarginAmount, m)
		}
	}
	// Decimal arithmetic keeps the change as precise as the amounts, which float subtraction would not
	change := decimal.MustFromFloat(req.NewAllocatedMargin).Sub(decimal.MustFromFloat(req.OriginalAllocatedMargin))
	if change.IsZero() {
		return nil, errMarginChangeRequired
	}
	amount := change.Abs().InexactFloat64()
	positionSide := PositionSide(strings.ToUpper(req.MarginSide))
	var err error
	if req.Asset == asset.USDTMarginedFutures {
		adjustment := SwapMarginAdjustmentTypeIncrease
		if change.IsNegative() {
			adjustment = SwapMarginAdjustmentTypeDecrease
		}
		_, err = e.ModifySwapIsolatedPositionMargin(ctx, &ModifySwapIsolatedPositionMarginRequest{Symbol: req.Pair, Amount: amount, Type: adjustment, PositionSide: positionSide})
	} else {
		if positionSide == "" {
			return nil, errPositionSideRequired
		}
		adjustment := CoinMarginedMarginAdjustmentTypeIncrease
		if change.IsNegative() {
			adjustment = CoinMarginedMarginAdjustmentTypeDecrease
		}
		err = e.AdjustCoinMarginedIsolatedMargin(ctx, &AdjustCoinMarginedIsolatedMarginRequest{Symbol: req.Pair, Amount: amount, Type: adjustment, PositionSide: positionSide})
	}
	if err != nil {
		return nil, err
	}
	return &margin.PositionChangeResponse{
		Exchange:        e.Name,
		Pair:            req.Pair,
		Asset:           req.Asset,
		MarginType:      req.MarginType,
		AllocatedMargin: req.NewAllocatedMargin,
	}, nil
}

// summaryPositionSide returns the side of a symbol's position a summary is of, from the sides of its open positions: a
// direction selects the hedge mode long or short position, and without one the one-way position, or in hedge mode its
// one open side; a direction is required while both sides are open
func summaryPositionSide(direction order.Side, sides []PositionSide) (PositionSide, error) {
	switch {
	case direction.IsLong():
		return PositionSideLong, nil
	case direction.IsShort():
		return PositionSideShort, nil
	}
	long, short := slices.Contains(sides, PositionSideLong), slices.Contains(sides, PositionSideShort)
	switch {
	case long && short:
		return "", fmt.Errorf("%w: both a long and a short position are open", errPositionDirectionRequired)
	case long:
		return PositionSideLong, nil
	case short:
		return PositionSideShort, nil
	default:
		return PositionSideBoth, nil
	}
}

// GetFuturesPositionSummary returns a perpetual position's summary, its collateral being its settlement currency's
// balance. A direction selects a hedge mode side, and without one, as the RPC server sends, the summary is of the
// one-way position or of the one open hedge mode side; a direction is required while both sides are open. A USDT-M
// summary reports the account's collateral mode, and a coin-M summary's sizes count contracts
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.CalculateOffline {
		return nil, common.ErrCannotCalculateOffline
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch req.Asset {
	case asset.USDTMarginedFutures:
		return e.swapPositionSummary(ctx, req)
	case asset.CoinMarginedFutures:
		return e.coinMarginedPositionSummary(ctx, req)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.Asset)
	}
}

// swapPositionSummary returns a USDT-M position's summary
func (e *Exchange) swapPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	symbol, err := e.FormatSymbol(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	settlement := req.Pair.Quote
	positions, err := e.GetSwapPositionData(ctx, &SwapPositionDataRequest{Symbol: req.Pair, Currency: settlement})
	if err != nil {
		return nil, err
	}
	var sides []PositionSide
	for i := range positions {
		if positions[i].Symbol == symbol {
			sides = append(sides, positions[i].PositionSide)
		}
	}
	positionSide, err := summaryPositionSide(req.Direction, sides)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", symbol, err)
	}
	i := slices.IndexFunc(positions, func(p SwapPositionDataResponse) bool {
		return p.Symbol == symbol && p.PositionSide == positionSide
	})
	if i < 0 {
		return nil, fmt.Errorf("%w: %s %s %s", futures.ErrNoPositionsFound, req.Asset, req.Pair, positionSide)
	}
	position := &positions[i]
	balances, err := e.GetSwapAccountData(ctx)
	if err != nil {
		return nil, err
	}
	j := slices.IndexFunc(balances, func(b SwapAccountDataResponse) bool { return b.Asset.Equal(settlement) })
	if j < 0 {
		return nil, fmt.Errorf("%w: %s %s collateral %s", currency.ErrCurrencyNotFound, req.Asset, req.Pair, settlement)
	}
	collateralMode, err := e.GetCollateralMode(ctx, req.Asset)
	if err != nil {
		return nil, err
	}
	summary := &futures.PositionSummary{
		Pair:                      req.Pair,
		Asset:                     req.Asset,
		MarginType:                isolatedMarginType(position.Isolated),
		CollateralMode:            collateralMode,
		Currency:                  settlement,
		NotionalSize:              position.PositionValue.Decimal(),
		Leverage:                  decimal.NewFromInt(int64(min(position.Leverage, math.MaxInt64))),
		InitialMarginRequirement:  position.InitialMargin.Decimal(),
		EstimatedLiquidationPrice: decimal.MustFromFloat(position.LiquidationPrice),
		CollateralUsed:            position.Margin.Decimal(),
		MarkPrice:                 position.MarkPrice.Decimal(),
		CurrentSize:               position.PositionAmount.Decimal(),
		ContractSettlementType:    futures.Linear,
		AverageOpenPrice:          position.AveragePrice.Decimal(),
		UnrealisedPNL:             position.UnrealisedProfit.Decimal(),
		RealisedPNL:               position.RealisedProfit.Decimal(),
		TotalCollateral:           balances[j].Balance.Decimal(),
		FreeCollateral:            balances[j].AvailableMargin.Decimal(),
	}
	if position.Isolated {
		summary.IsolatedMargin = position.Margin.Decimal()
	}
	summary.FrozenBalance = summary.TotalCollateral.Sub(summary.FreeCollateral)
	return summary, nil
}

// coinMarginedPositionSummary returns a coin-M position's summary
func (e *Exchange) coinMarginedPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	symbol, err := e.FormatSymbol(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	positions, err := e.GetCoinMarginedPositions(ctx, req.Pair)
	if err != nil {
		return nil, err
	}
	var sides []PositionSide
	for i := range positions {
		if positions[i].Symbol == symbol {
			sides = append(sides, convertPositionSide(string(positions[i].PositionSide)))
		}
	}
	positionSide, err := summaryPositionSide(req.Direction, sides)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", symbol, err)
	}
	i := slices.IndexFunc(positions, func(p CoinMarginedPositionResponse) bool {
		return p.Symbol == symbol && convertPositionSide(string(p.PositionSide)) == positionSide
	})
	if i < 0 {
		return nil, fmt.Errorf("%w: %s %s %s", futures.ErrNoPositionsFound, req.Asset, req.Pair, positionSide)
	}
	position := &positions[i]
	settlement := req.Pair.Base
	balances, err := e.GetCoinMarginedAccountAssets(ctx, req.Pair)
	if err != nil {
		return nil, err
	}
	j := slices.IndexFunc(balances, func(b CoinMarginedAccountAssetResponse) bool { return b.Asset.Equal(settlement) })
	if j < 0 {
		return nil, fmt.Errorf("%w: %s %s collateral %s", currency.ErrCurrencyNotFound, req.Asset, req.Pair, settlement)
	}
	summary := &futures.PositionSummary{
		Pair:                      req.Pair,
		Asset:                     req.Asset,
		MarginType:                isolatedMarginType(position.Isolated),
		CollateralMode:            collateral.SingleMode,
		Currency:                  settlement,
		Leverage:                  decimal.NewFromInt(int64(min(position.Leverage, math.MaxInt64))),
		InitialMarginRequirement:  position.InitialMargin.Decimal(),
		EstimatedLiquidationPrice: decimal.MustFromFloat(position.LiquidationPrice),
		CollateralUsed:            position.InitialMargin.Decimal(),
		MarkPrice:                 position.MarkPrice.Decimal(),
		CurrentSize:               position.PositionAmount.Decimal(),
		ContractSettlementType:    futures.Inverse,
		AverageOpenPrice:          position.AveragePrice.Decimal(),
		UnrealisedPNL:             position.UnrealisedProfit.Decimal(),
		TotalCollateral:           balances[j].Balance.Decimal(),
		FreeCollateral:            balances[j].AvailableMargin.Decimal(),
	}
	if position.Isolated {
		summary.IsolatedMargin = position.InitialMargin.Decimal()
	}
	summary.FrozenBalance = summary.TotalCollateral.Sub(summary.FreeCollateral)
	return summary, nil
}

// isolatedMarginType returns the margin type of a position BingX reports as isolated or not
func isolatedMarginType(isolated bool) margin.Type {
	if isolated {
		return margin.Isolated
	}
	return margin.Multi
}

// GetFuturesPositionOrders returns the filled and cancelled orders of each pair's position from the request's start to
// its end, now when unset, collected in the seven day windows the order history endpoints take, with the pair's
// margin type
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var settlementType futures.ContractSettlementType
	switch req.Asset {
	case asset.USDTMarginedFutures:
		settlementType = futures.Linear
	case asset.CoinMarginedFutures:
		settlementType = futures.Inverse
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.Asset)
	}
	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	end := req.EndDate
	if end.IsZero() {
		end = time.Now()
	}
	if err := common.StartEndTimeCheck(req.StartDate, end); err != nil {
		return nil, err
	}
	history := &order.MultiOrderRequest{AssetType: req.Asset, StartTime: req.StartDate, EndTime: end}
	resp := make([]futures.PositionResponse, len(req.Pairs))
	for i, p := range req.Pairs {
		var orders []order.Detail
		var mt margin.Type
		var err error
		if req.Asset == asset.USDTMarginedFutures {
			if orders, err = e.swapOrderHistory(ctx, history, p, 0); err != nil {
				return nil, err
			}
			marginType, err := e.GetSwapMarginType(ctx, p)
			if err != nil {
				return nil, err
			}
			mt, err = convertMarginType(marginType.MarginType)
			if err != nil {
				e.logOrderIssue(req.Asset, p.String(), err)
			}
		} else {
			if orders, err = e.coinMarginedOrderHistory(ctx, history, p, 0); err != nil {
				return nil, err
			}
			marginTypes, err := e.GetCoinMarginedMarginType(ctx, p)
			if err != nil {
				return nil, err
			}
			mt, err = convertMarginType(marginTypes[0].MarginType)
			if err != nil {
				e.logOrderIssue(req.Asset, p.String(), err)
			}
		}
		for j := range orders {
			orders[j].MarginType = mt
		}
		resp[i] = futures.PositionResponse{
			Pair:                   p,
			Asset:                  req.Asset,
			ContractSettlementType: settlementType,
			Orders:                 orders,
		}
	}
	return resp, nil
}

// SetCollateralMode sets the USDT-M account's collateral mode: multi-assets mode, in which every asset the multi-assets
// rules list margins every contract, or single asset mode, in which each contract is margined in its settlement
// currency. Coin-M contracts are margined in their base coin alone
func (e *Exchange) SetCollateralMode(ctx context.Context, a asset.Item, mode collateral.Mode) error {
	switch a {
	case asset.USDTMarginedFutures:
	case asset.CoinMarginedFutures:
		return fmt.Errorf("%w: %s contracts are margined in their base coin alone", common.ErrFunctionNotSupported, a)
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	var assetMode AssetMode
	switch mode {
	case collateral.MultiMode:
		assetMode = AssetModeMultiAssets
	case collateral.SingleMode:
		assetMode = AssetModeSingleAsset
	default:
		return fmt.Errorf("%w: %s", order.ErrCollateralInvalid, mode)
	}
	_, err := e.SwitchSwapMultiAssetsMode(ctx, assetMode)
	return err
}

// GetCollateralMode returns the USDT-M account's collateral mode: multi-assets mode or single asset mode
func (e *Exchange) GetCollateralMode(ctx context.Context, a asset.Item) (collateral.Mode, error) {
	switch a {
	case asset.USDTMarginedFutures:
	case asset.CoinMarginedFutures:
		return collateral.UnknownMode, fmt.Errorf("%w: %s contracts are margined in their base coin alone", common.ErrFunctionNotSupported, a)
	default:
		return collateral.UnknownMode, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	resp, err := e.GetSwapMultiAssetsMode(ctx)
	if err != nil {
		return collateral.UnknownMode, err
	}
	switch resp.AssetMode {
	case AssetModeMultiAssets:
		return collateral.MultiMode, nil
	case AssetModeSingleAsset:
		return collateral.SingleMode, nil
	default:
		return collateral.UnknownMode, fmt.Errorf("%w: %q", order.ErrCollateralInvalid, resp.AssetMode)
	}
}

// GetMarginRatesHistory is not supported: BingX offers no margin lending
func (e *Exchange) GetMarginRatesHistory(context.Context, *margin.RateHistoryRequest) (*margin.RateHistoryResponse, error) {
	return nil, common.ErrFunctionNotSupported
}
