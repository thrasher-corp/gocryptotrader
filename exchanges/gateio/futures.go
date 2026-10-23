package gateio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetAllFutureContracts retrieves the futures contracts of a settlement currency, all of which are returned when no
// limit is supplied
func (e *Exchange) GetAllFutureContracts(ctx context.Context, settle currency.Code, limit, offset uint64) ([]*FuturesContractResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var contracts []*FuturesContractResponse
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesContractsEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/contracts", params), &contracts)
}

// GetAllFutureContractsIncludingDelisted retrieves the futures contracts of a settlement currency, including
// delisted ones
func (e *Exchange) GetAllFutureContractsIncludingDelisted(ctx context.Context, settle currency.Code, limit, offset uint64) ([]*FuturesContractResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var contracts []*FuturesContractResponse
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesContractsAllEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/contracts_all", params), &contracts)
}

// GetFuturesContract returns a single futures contract info for the specified settle and Currency Pair (contract << in this case)
func (e *Exchange) GetFuturesContract(ctx context.Context, settle currency.Code, contract string) (*FuturesContractResponse, error) {
	if contract == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var futureContract *FuturesContractResponse
	return futureContract, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesContractsEPL, "futures/"+settle.Item.Lower+"/contracts/"+contract, &futureContract)
}

// GetFuturesADLRiskStates retrieves the auto-deleveraging risk state of every futures market of a settlement currency
func (e *Exchange) GetFuturesADLRiskStates(ctx context.Context, settle currency.Code) (*FuturesADLRiskStatesResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var resp *FuturesADLRiskStatesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesADLRiskStatesEPL, "futures/"+settle.Item.Lower+"/adl_risk_states", &resp)
}

// GetFuturesOrderbook retrieves futures order book data
func (e *Exchange) GetFuturesOrderbook(ctx context.Context, settle currency.Code, contract currency.Pair, interval string, limit uint64, withOrderbookID bool) (*OrderbookResponse, error) {
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if withOrderbookID {
		params.Set("with_id", "true")
	}
	var response *OrderbookResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/order_book", params), &response)
}

// GetFuturesTradingHistory retrieves futures trading history
func (e *Exchange) GetFuturesTradingHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, lastID string, from, to time.Time) ([]*TradingHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.Upper().String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var response []*TradingHistoryItemResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTradingHistoryFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/trades", params), &response)
}

// GetFuturesCandlesticks retrieves specified contract candlesticks. A limit cannot be combined with a time range,
// and timezone, one of all, utc0 or utc8, defaults to utc0 when empty.
func (e *Exchange) GetFuturesCandlesticks(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64, interval kline.Interval, timezone string) ([]*FuturesCandlestickResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if limit > 0 && (!from.IsZero() || !to.IsZero()) {
		return nil, errLimitWithTimeRange
	}
	if timezone != "" && timezone != "all" && timezone != utc0TimeZone && timezone != utc8TimeZone {
		return nil, fmt.Errorf("%w: %q", errInvalidTimezone, timezone)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if interval.Duration().Microseconds() != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	if timezone != "" {
		params.Set("timezone", timezone)
	}
	var candlesticks []*FuturesCandlestickResponse
	return candlesticks, e.SendHTTPRequest(ctx, exchange.RestFutures, publicCandleSticksFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/candlesticks", params), &candlesticks)
}

// PremiumIndexKline retrieves premium Index K-Line
// Maximum of 1000 points can be returned in a query. Be sure not to exceed the limit when specifying from, to and interval
// A limit cannot be combined with a time range
func (e *Exchange) PremiumIndexKline(ctx context.Context, settleCurrency currency.Code, contract currency.Pair, from, to time.Time, limit uint64, interval kline.Interval) ([]*FuturesPremiumIndexKlineResponse, error) {
	if settleCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", errEmptyOrInvalidSettlementCurrency)
	}
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if limit > 0 && (!from.IsZero() || !to.IsZero()) {
		return nil, errLimitWithTimeRange
	}
	intervalString, err := getIntervalString(interval)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("interval", intervalString)
	var resp []*FuturesPremiumIndexKlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicPremiumIndexEPL, common.EncodeURLValues("futures/"+settleCurrency.Item.Lower+"/premium_index", params), &resp)
}

// GetFuturesTickers retrieves futures ticker information for a specific settle and contract info.
func (e *Exchange) GetFuturesTickers(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*FuturesTickerResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var tickers []*FuturesTickerResponse
	return tickers, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/tickers", params), &tickers)
}

// GetFutureFundingRates retrieves funding rate information.
func (e *Exchange) GetFutureFundingRates(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, from, to time.Time) ([]*FuturesFundingRateResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var rates []*FuturesFundingRateResponse
	return rates, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFundingRatesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/funding_rate", params), &rates)
}

// GetBatchFuturesFundingRates retrieves the funding rate history of several perpetual contracts at once
func (e *Exchange) GetBatchFuturesFundingRates(ctx context.Context, settle currency.Code, contracts currency.Pairs) ([]*BatchFundingRatesResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if len(contracts) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	names := make([]string, len(contracts))
	for i := range contracts {
		if contracts[i].IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		names[i] = contracts[i].String()
	}
	var resp []*BatchFundingRatesResponse
	return resp, e.sendPublicPostRequest(ctx, publicBatchFundingRatesEPL, "futures/"+settle.Item.Lower+"/funding_rates", &batchFundingRatesRequest{Contracts: names}, &resp)
}

// GetFuturesInsuranceBalanceHistory retrieves futures insurance balance history
func (e *Exchange) GetFuturesInsuranceBalanceHistory(ctx context.Context, settle currency.Code, limit uint64) ([]*InsuranceBalanceResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var balances []*InsuranceBalanceResponse
	return balances, e.SendHTTPRequest(ctx, exchange.RestSpot, publicInsuranceFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/insurance", params), &balances)
}

// GetFutureStats retrieves futures stats
func (e *Exchange) GetFutureStats(ctx context.Context, settle currency.Code, contract currency.Pair, from time.Time, interval kline.Interval, limit uint64) ([]*ContractStatResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var stats []*ContractStatResponse
	return stats, e.SendHTTPRequest(ctx, exchange.RestSpot, publicStatsFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/contract_stats", params), &stats)
}

// GetIndexConstituent retrieves index constituents
func (e *Exchange) GetIndexConstituent(ctx context.Context, settle currency.Code, index string) (*IndexConstituentResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if index == "" {
		return nil, fmt.Errorf("%w: index pair string is required", currency.ErrCurrencyPairEmpty)
	}
	indexString := strings.ToUpper(index)
	var constituents *IndexConstituentResponse
	return constituents, e.SendHTTPRequest(ctx, exchange.RestSpot, publicIndexConstituentsEPL, "futures/"+settle.Item.Lower+"/index_constituents/"+indexString, &constituents)
}

// GetLiquidationHistory retrieves the public liquidation orders of a settlement currency, optionally for one contract.
// A time range may span at most an hour.
func (e *Exchange) GetLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64) ([]*LiquidationHistoryResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if !from.IsZero() && !to.IsZero() && to.Sub(from) > time.Hour {
		return nil, errLiquidationTimeRangeTooLarge
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var histories []*LiquidationHistoryResponse
	return histories, e.SendHTTPRequest(ctx, exchange.RestSpot, publicLiquidationHistoryEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/liq_orders", params), &histories)
}

// QueryFuturesAccount retrieves futures account
func (e *Exchange) QueryFuturesAccount(ctx context.Context, settle currency.Code) (*FuturesAccountResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var response *FuturesAccountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAccountEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/accounts", nil, nil, &response)
}

// GetFuturesAccountBooks retrieves account books. A contract only filters records from 2023-10-30 onwards.
func (e *Exchange) GetFuturesAccountBooks(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time, changingType string) ([]*AccountBookItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if changingType != "" {
		params.Set("type", changingType)
	}
	var response []*AccountBookItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAccountBooksEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/account_book", params, nil, &response)
}

// GetAllFuturesPositionsOfUsers list all positions of users. Every position is returned when no limit is supplied,
// and a supplied limit may not exceed 100.
func (e *Exchange) GetAllFuturesPositionsOfUsers(ctx context.Context, settle currency.Code, realPositionsOnly bool, limit, offset uint64) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if limit > 100 {
		return nil, fmt.Errorf("%w: %d exceeds the maximum of 100 positions", errInvalidLimit, limit)
	}
	params := url.Values{}
	if realPositionsOnly {
		params.Set("holding", "true")
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionsEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/positions", params, nil, &response)
}

// GetFuturesHistoricalPositions retrieves a contract's historical position snapshots by time range
func (e *Exchange) GetFuturesHistoricalPositions(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time) ([]*FuturesHistoricalPositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var resp []*FuturesHistoricalPositionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionsTimeRangeEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/positions_timerange", params, nil, &resp)
}

// GetSinglePosition returns a single position. A contract holding both a long and a short position in hedge mode is
// queried with GetPositionDetailInDualMode.
func (e *Exchange) GetSinglePosition(ctx context.Context, settle currency.Code, contract currency.Pair) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response *PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/positions/"+contract.String(), nil, nil, &response)
}

// GetFuturesLeverage retrieves a contract's leverage for a position margin mode, isolated or cross, and a hedge mode
// side, dual_long or dual_short
func (e *Exchange) GetFuturesLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, positionMarginMode, dualSide string) (*FuturesLeverageResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if positionMarginMode != "isolated" && positionMarginMode != "cross" {
		return nil, fmt.Errorf("%w: %q", errInvalidPositionMarginMode, positionMarginMode)
	}
	if dualSide != "dual_long" && dualSide != "dual_short" {
		return nil, fmt.Errorf("%w: 'dual_side' should be 'dual_short' or 'dual_long'", order.ErrSideIsInvalid)
	}
	params := url.Values{}
	params.Set("pos_margin_mode", positionMarginMode)
	params.Set("dual_side", dualSide)
	var resp *FuturesLeverageResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetLeverageEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/get_leverage/"+contract.String(), params, nil, &resp)
}

// UpdateFuturesPositionMargin adds margin to a futures position, or removes it with a negative change
func (e *Exchange) UpdateFuturesPositionMargin(ctx context.Context, settle currency.Code, change float64, contract currency.Pair) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if change == 0 {
		return nil, errInvalidMarginChange
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	var response *PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateMarginEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/positions/"+contract.String()+"/margin", params, nil, &response)
}

// UpdateFuturesPositionLeverage update position leverage. A non-zero leverage sets isolated margin and ignores the
// cross leverage limit, while a zero leverage sets cross margin at the cross leverage limit.
func (e *Exchange) UpdateFuturesPositionLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, leverage, crossLeverageLimit float64, positionID uint64) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage < 0 {
		return nil, fmt.Errorf("%w: %f", order.ErrSubmitLeverageNotSupported, leverage)
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	if leverage == 0 && crossLeverageLimit > 0 {
		params.Set("cross_leverage_limit", strconv.FormatFloat(crossLeverageLimit, 'f', -1, 64))
	}
	if positionID > 0 {
		params.Set("pid", strconv.FormatUint(positionID, 10))
	}
	var resp positionReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateLeverageEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/positions/"+contract.String()+"/leverage", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Position == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Position, nil
}

// SetFuturesLeverage sets a contract's leverage for a margin mode, isolated or cross, optionally for one hedge mode
// side, dual_long or dual_short
func (e *Exchange) SetFuturesLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, leverage float64, marginMode, dualSide string) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage <= 0 {
		return nil, fmt.Errorf("%w: %f", errInvalidLeverage, leverage)
	}
	if marginMode != "isolated" && marginMode != "cross" {
		return nil, fmt.Errorf("%w: %q", errInvalidPositionMarginMode, marginMode)
	}
	if dualSide != "" && dualSide != "dual_long" && dualSide != "dual_short" {
		return nil, fmt.Errorf("%w: 'dual_side' should be 'dual_short' or 'dual_long'", order.ErrSideIsInvalid)
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	params.Set("margin_mode", marginMode)
	if dualSide != "" {
		params.Set("dual_side", dualSide)
	}
	var resp *PositionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSetLeverageEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/positions/"+contract.String()+"/set_leverage", params, nil, &resp)
}

// SwitchFuturesPositionMarginMode switches a futures position between ISOLATED and CROSS margin
func (e *Exchange) SwitchFuturesPositionMarginMode(ctx context.Context, settle currency.Code, arg *FuturesPositionMarginModeRequest) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *PositionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSwitchMarginModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/positions/cross_mode", nil, arg, &resp)
}

// SwitchFuturesDualModePositionMarginMode switches a contract's hedge mode positions between ISOLATED and CROSS margin
func (e *Exchange) SwitchFuturesDualModePositionMarginMode(ctx context.Context, settle currency.Code, arg *FuturesPositionMarginModeRequest) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp []*PositionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSwitchDualModeMarginModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/dual_comp/positions/cross_mode", nil, arg, &resp)
}

// validate checks a position margin mode switch before it is sent
func (f *FuturesPositionMarginModeRequest) validate() error {
	if err := common.NilGuard(f); err != nil {
		return err
	}
	if f.Mode != "ISOLATED" && f.Mode != "CROSS" {
		return fmt.Errorf("%w: %q", errInvalidPositionMarginMode, f.Mode)
	}
	if f.Contract.IsEmpty() {
		return fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	return nil
}

// EnableOrDisableDualMode enable or disable dual mode
// Before setting dual mode, make sure all positions are closed and no orders are open
func (e *Exchange) EnableOrDisableDualMode(ctx context.Context, settle currency.Code, dualMode bool) (*FuturesAccountResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("dual_mode", strconv.FormatBool(dualMode))
	var response *FuturesAccountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualToggleDualModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/dual_mode", params, nil, &response)
}

// SetFuturesPositionMode sets the position holding mode to single, dual or dual_plus, superseding
// EnableOrDisableDualMode. Every position must be closed and every order cancelled beforehand.
func (e *Exchange) SetFuturesPositionMode(ctx context.Context, settle currency.Code, positionMode string) (*FuturesAccountResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if positionMode != "single" && positionMode != "dual" && positionMode != "dual_plus" {
		return nil, fmt.Errorf("%w: %q", errInvalidPositionMode, positionMode)
	}
	params := url.Values{}
	params.Set("position_mode", positionMode)
	var resp *FuturesAccountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSetPositionModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/set_position_mode", params, nil, &resp)
}

// GetPositionDetailInDualMode retrieve position detail in dual mode
func (e *Exchange) GetPositionDetailInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionsDualModeEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/dual_comp/positions/"+contract.String(), nil, nil, &response)
}

// UpdatePositionMarginInDualMode adds margin to one side of a hedge mode position, or removes it with a negative
// change
func (e *Exchange) UpdatePositionMarginInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, change float64, dualSide string) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if change == 0 {
		return nil, errInvalidMarginChange
	}
	if dualSide != "dual_long" && dualSide != "dual_short" {
		return nil, fmt.Errorf("%w: 'dual_side' should be 'dual_short' or 'dual_long'", order.ErrSideIsInvalid)
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	params.Set("dual_side", dualSide)
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateMarginDualModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/margin", params, nil, &response)
}

// UpdatePositionLeverageInDualMode update position leverage in dual mode
func (e *Exchange) UpdatePositionLeverageInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, leverage, crossLeverageLimit float64) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage < 0 {
		return nil, fmt.Errorf("%w: %f", order.ErrSubmitLeverageNotSupported, leverage)
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	if leverage == 0 && crossLeverageLimit > 0 {
		params.Set("cross_leverage_limit", strconv.FormatFloat(crossLeverageLimit, 'f', -1, 64))
	}
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateLeverageDualModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/leverage", params, nil, &response)
}

// UpdatePositionRiskLimitInDualMode update position risk limit in dual mode
// Risk Limit as of GateIO: https://www.gate.com/en/help/futures/futures-logic/22162
func (e *Exchange) UpdatePositionRiskLimitInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, riskLimit float64) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if riskLimit <= 0 {
		return nil, errInvalidRiskLimit
	}
	params := url.Values{}
	params.Set("risk_limit", strconv.FormatFloat(riskLimit, 'f', -1, 64))
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateRiskDualModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/risk_limit", params, nil, &response)
}

// PlaceFuturesOrder creates futures order
// Create a futures order
// Creating futures orders requires size, which is number of contracts instead of currency amount. You can use quanto_multiplier in contract detail response to know how much currency 1 size contract represents
// Zero-filled order cannot be retrieved 10 minutes after order cancellation. You will get a 404 not found for such orders
// Set reduce_only to true can keep the position from changing side when reducing position size
// In single position mode, to close a position, you need to set size to 0 and close to true
// In dual position mode, to close one side position, you need to set auto_size side, reduce_only to true and size to 0
func (e *Exchange) PlaceFuturesOrder(ctx context.Context, arg *FuturesOrderCreateRequest) (*FuturesOrderResponse, error) {
	if err := arg.validate(true); err != nil {
		return nil, err
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitOrderEPL, http.MethodPost, "futures/"+arg.Settle.Item.Lower+"/orders", nil, &arg, &response)
}

// GetFuturesOrders retrieves list of futures orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) GetFuturesOrders(ctx context.Context, contract currency.Pair, status, lastID string, settle currency.Code, limit, offset uint64) ([]*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w, only 'open' and 'finished' status are supported", errInvalidOrderStatus)
	}
	params := url.Values{}
	params.Set("status", status)
	if !contract.IsEmpty() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	var response []*FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetOrdersEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/orders", params, nil, &response)
}

// CancelMultipleFuturesOpenOrders cancels every open futures order matching the request's filters
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) CancelMultipleFuturesOpenOrders(ctx context.Context, settle currency.Code, arg *CancelFuturesOrdersRequest) ([]*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Side != "" && arg.Side != order.Bid.Lower() && arg.Side != order.Ask.Lower() {
		return nil, fmt.Errorf("%w: %q; only 'bid' and 'ask' are supported", order.ErrSideIsInvalid, arg.Side)
	}
	params := url.Values{}
	if arg.Contract.IsPopulated() {
		params.Set("contract", arg.Contract.String())
	}
	if arg.Side != "" {
		params.Set("side", arg.Side)
	}
	if arg.ActionMode != "" {
		params.Set("action_mode", arg.ActionMode)
	}
	if arg.ExcludeReduceOnly {
		params.Set("exclude_reduce_only", "true")
	}
	if arg.Text != "" {
		params.Set("text", arg.Text)
	}
	var response []*FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelOrdersEPL, http.MethodDelete, "futures/"+settle.Item.Lower+"/orders", params, nil, &response)
}

// GetFuturesOrdersByTimeRange retrieves futures orders by time range, including those older than the six months
// GetFuturesOrders covers
func (e *Exchange) GetFuturesOrdersByTimeRange(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time) ([]*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var resp []*FuturesOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetOrdersByTimeRangeEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/orders_timerange", params, nil, &resp)
}

// PlaceBatchFuturesOrders creates a list of futures orders
// Up to 10 orders per request
// If any of the order's parameters are missing or in the wrong format, all of them will not be executed, and a http status 400 error will be returned directly
// If the parameters are checked and passed, all are executed. Even if there is a business logic error in the middle (such as insufficient funds), it will not affect other execution orders
// The returned result is in array format, and the order corresponds to the orders in the request body
// In the returned result, the succeeded field of type bool indicates whether the execution was successful or not
// If the execution is successful, the normal order content is included; if the execution fails, the label field is included to indicate the cause of the error
// In the rate limiting, each order is counted individually
func (e *Exchange) PlaceBatchFuturesOrders(ctx context.Context, settle currency.Code, args []FuturesOrderCreateRequest) ([]*BatchFuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if len(args) == 0 {
		return nil, errOrdersEmpty
	}
	if len(args) > 10 {
		return nil, errTooManyOrderRequest
	}
	for x := range args {
		if err := args[x].validate(true); err != nil {
			return nil, err
		}
	}
	var response []*BatchFuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitBatchOrdersEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/batch_orders", nil, &args, &response)
}

// GetSingleFuturesOrder retrieves a single order by its identifier
func (e *Exchange) GetSingleFuturesOrder(ctx context.Context, settle currency.Code, orderID string) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualFetchOrderEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// CancelSingleFuturesOrder cancels a single order by its ID or custom text. An action mode of ACK, RESULT or FULL
// selects the fields returned.
func (e *Exchange) CancelSingleFuturesOrder(ctx context.Context, settle currency.Code, orderID, actionMode string) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if actionMode != "" {
		params.Set("action_mode", actionMode)
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelOrderEPL, http.MethodDelete, "futures/"+settle.Item.Lower+"/orders/"+orderID, params, nil, &response)
}

// AmendFuturesOrder amends an existing futures order's size, price or both
func (e *Exchange) AmendFuturesOrder(ctx context.Context, settle currency.Code, orderID string, arg *AmendFuturesOrderRequest) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Size == 0 && arg.Price == 0 {
		return nil, fmt.Errorf("%w: size or price is required", errNoOrderAmendment)
	}
	if arg.Price < 0 {
		return nil, fmt.Errorf("%w: %v", limits.ErrPriceBelowMin, arg.Price)
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAmendOrderEPL, http.MethodPut, "futures/"+settle.Item.Lower+"/orders/"+orderID, nil, arg, &response)
}

// GetMyFuturesTradingHistory retrieves the account's futures fills from the past six months; older fills are
// retrieved with GetMyFuturesTradingHistoryByTimeRange
func (e *Exchange) GetMyFuturesTradingHistory(ctx context.Context, settle currency.Code, lastID, orderID string, contract currency.Pair, limit, offset uint64) ([]*TradingHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if orderID != "" {
		params.Set("order", orderID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	var response []*TradingHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualTradingHistoryEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/my_trades", params, nil, &response)
}

// GetMyFuturesTradingHistoryByTimeRange retrieves the account's futures fills by time range, optionally only those
// in a role of maker or taker
func (e *Exchange) GetMyFuturesTradingHistoryByTimeRange(ctx context.Context, settle currency.Code, contract currency.Pair, role string, limit, offset uint64, from, to time.Time) ([]*FuturesTradeByTimeRangeResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if role != "" && role != "maker" && role != "taker" {
		return nil, fmt.Errorf("%w: %q", errInvalidTradeRole, role)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if role != "" {
		params.Set("role", role)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var resp []*FuturesTradeByTimeRangeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualTradingHistoryByTimeRangeEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/my_trades_timerange", params, nil, &resp)
}

// GetFuturesPositionCloseHistory lists position close history, optionally filtered by a side of long or short and
// by profit or loss
func (e *Exchange) GetFuturesPositionCloseHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time, side, pnl string) ([]*PositionCloseHistoryResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if side != "" && side != order.Long.Lower() && side != order.Short.Lower() {
		return nil, fmt.Errorf("%w: %q; only 'long' and 'short' are supported", order.ErrSideIsInvalid, side)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if side != "" {
		params.Set("side", side)
	}
	if pnl != "" {
		params.Set("pnl", pnl)
	}
	var response []*PositionCloseHistoryResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualClosePositionEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/position_close", params, nil, &response)
}

// GetFuturesLiquidationHistory lists the account's liquidations, optionally those at a given time
func (e *Exchange) GetFuturesLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to, at time.Time) ([]*LiquidationHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var response []*LiquidationHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualLiquidationHistoryEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/liquidates", params, nil, &response)
}

// GetFuturesAutoDeleverages lists the auto-deleveraging of the account's positions, optionally those at a given time
func (e *Exchange) GetFuturesAutoDeleverages(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to, at time.Time) ([]*FuturesAutoDeleverageResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var resp []*FuturesAutoDeleverageResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAutoDeleveragesEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/auto_deleverages", params, nil, &resp)
}

// CountdownCancelFuturesOrders sets a countdown, of at least 5 seconds, after which the account's open futures
// orders are cancelled unless the countdown is renewed. A zero timeout cancels the countdown.
func (e *Exchange) CountdownCancelFuturesOrders(ctx context.Context, settle currency.Code, arg *CountdownRequest) (*TriggerTimeResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", errEmptyOrInvalidSettlementCurrency)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if err := validateCountdownTimeout(arg.Countdown.Duration()); err != nil {
		return nil, err
	}
	return e.sendCountdownCancelOrdersRequest(ctx, perpetualCancelTriggerOrdersEPL, "futures/"+settle.Item.Lower+"/countdown_cancel_all", arg)
}

func (e *Exchange) sendCountdownCancelOrdersRequest(ctx context.Context, epl request.EndpointLimit, path string, arg any) (*TriggerTimeResponse, error) {
	var response *TriggerTimeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodPost, path, nil, arg, &response)
}

// GetFuturesTradingFees retrieves the account's futures trading fee rates keyed by contract, optionally for one
// contract
func (e *Exchange) GetFuturesTradingFees(ctx context.Context, settle currency.Code, contract currency.Pair) (map[string]*FuturesTradingFeeResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var resp map[string]*FuturesTradingFeeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualTradingFeeEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/fee", params, nil, &resp)
}

// CancelBatchFuturesOrders cancels up to 20 futures orders by their IDs
func (e *Exchange) CancelBatchFuturesOrders(ctx context.Context, settle currency.Code, orderIDs []string) ([]*CancelFuturesOrderResultResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if len(orderIDs) == 0 {
		return nil, errNoOrdersToCancel
	}
	if len(orderIDs) > 20 {
		return nil, fmt.Errorf("%w: %d exceeds the maximum of 20", errTooManyOrders, len(orderIDs))
	}
	if slices.Contains(orderIDs, "") {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []*CancelFuturesOrderResultResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelBatchOrdersEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/batch_cancel_orders", nil, orderIDs, &resp)
}

// AmendBatchFuturesOrders amends up to 10 futures orders, each identified by its order ID or custom text
func (e *Exchange) AmendBatchFuturesOrders(ctx context.Context, settle currency.Code, args []*BatchAmendFuturesOrderRequest) ([]*BatchFuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if len(args) == 0 {
		return nil, errOrdersEmpty
	}
	if len(args) > 10 {
		return nil, fmt.Errorf("%w: %d exceeds the maximum of 10", errTooManyOrders, len(args))
	}
	for _, arg := range args {
		if err := common.NilGuard(arg); err != nil {
			return nil, err
		}
		if arg.OrderID == 0 && arg.Text == "" {
			return nil, fmt.Errorf("%w: order ID or text is required", order.ErrOrderIDNotSet)
		}
		if arg.Size == 0 && arg.Price == 0 {
			return nil, fmt.Errorf("%w: size or price is required", errNoOrderAmendment)
		}
	}
	var resp []*BatchFuturesOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAmendBatchOrdersEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/batch_amend_orders", nil, args, &resp)
}

// PlaceFuturesBBOOrder places a futures order priced at a level of the order book, the bids for a sell and the asks
// for a buy
func (e *Exchange) PlaceFuturesBBOOrder(ctx context.Context, settle currency.Code, arg *FuturesBBOOrderRequest) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Size == 0 && !arg.ClosePosition && arg.AutoSize == "" {
		return nil, errInvalidOrderSize
	}
	if arg.Direction != order.Buy.Lower() && arg.Direction != order.Sell.Lower() {
		return nil, fmt.Errorf("%w: %q; only 'buy' and 'sell' are supported", order.ErrSideIsInvalid, arg.Direction)
	}
	if arg.Level < 1 || arg.Level > 20 {
		return nil, fmt.Errorf("%w: %d", errInvalidOrderBookLevel, arg.Level)
	}
	if arg.TimeInForce != "" {
		if _, err := timeInForceFromString(arg.TimeInForce); err != nil {
			return nil, err
		}
	}
	var resp *FuturesOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitBBOOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/bbo_orders", nil, arg, &resp)
}

// CreateTrailOrder creates a futures trailing stop order
func (e *Exchange) CreateTrailOrder(ctx context.Context, settle currency.Code, arg *CreateTrailOrderRequest) (*CreateTrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Amount == 0 {
		return nil, errInvalidOrderSize
	}
	if arg.PriceType > 3 {
		return nil, fmt.Errorf("%w: %d", errInvalidTriggerPriceType, arg.PriceType)
	}
	var resp gateioAPIResponse[*CreateTrailOrderResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCreateTrailOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/create", nil, arg, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// StopTrailOrder stops a futures trail order identified by its ID or custom text
func (e *Exchange) StopTrailOrder(ctx context.Context, settle currency.Code, arg *StopTrailOrderRequest) (*TrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ID == 0 && arg.Text == "" {
		return nil, errTrailOrderIDOrTextRequired
	}
	var resp trailOrderReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualStopTrailOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/stop", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// StopAllTrailOrders stops the futures trail orders matching the request's filters
func (e *Exchange) StopAllTrailOrders(ctx context.Context, settle currency.Code, arg *StopAllTrailOrdersRequest) ([]*TrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.RelatedPosition > 2 {
		return nil, fmt.Errorf("%w: related position %d", errInvalidTrailOrderFilter, arg.RelatedPosition)
	}
	var resp trailOrdersReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualStopAllTrailOrdersEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/stop_all", nil, arg, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetTrailOrders retrieves the futures trail orders matching the request's filters
func (e *Exchange) GetTrailOrders(ctx context.Context, settle currency.Code, arg *GetTrailOrdersRequest) ([]*TrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	switch {
	case arg.SortBy > 2:
		return nil, fmt.Errorf("%w: sort by %d", errInvalidTrailOrderFilter, arg.SortBy)
	case arg.RelatedPosition > 2:
		return nil, fmt.Errorf("%w: related position %d", errInvalidTrailOrderFilter, arg.RelatedPosition)
	case arg.ReduceOnly > 2:
		return nil, fmt.Errorf("%w: reduce only %d", errInvalidTrailOrderFilter, arg.ReduceOnly)
	case arg.Side > 2:
		return nil, fmt.Errorf("%w: side %d", errInvalidTrailOrderFilter, arg.Side)
	}
	if !arg.StartAt.IsZero() && !arg.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartAt, arg.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.Contract.IsPopulated() {
		params.Set("contract", arg.Contract.String())
	}
	if arg.IsFinished {
		params.Set("is_finished", "true")
	}
	if !arg.StartAt.IsZero() {
		params.Set("start_at", strconv.FormatInt(arg.StartAt.Unix(), 10))
	}
	if !arg.EndAt.IsZero() {
		params.Set("end_at", strconv.FormatInt(arg.EndAt.Unix(), 10))
	}
	if arg.PageNumber > 0 {
		params.Set("page_num", strconv.FormatUint(arg.PageNumber, 10))
	}
	if arg.PageSize > 0 {
		params.Set("page_size", strconv.FormatUint(arg.PageSize, 10))
	}
	if arg.SortBy > 0 {
		params.Set("sort_by", strconv.FormatUint(uint64(arg.SortBy), 10))
	}
	if arg.HideCancelled {
		params.Set("hide_cancel", "true")
	}
	if arg.RelatedPosition > 0 {
		params.Set("related_position", strconv.FormatUint(uint64(arg.RelatedPosition), 10))
	}
	if arg.SortByTrigger {
		params.Set("sort_by_trigger", "true")
	}
	if arg.ReduceOnly > 0 {
		params.Set("reduce_only", strconv.FormatUint(uint64(arg.ReduceOnly), 10))
	}
	if arg.Side > 0 {
		params.Set("side", strconv.FormatUint(uint64(arg.Side), 10))
	}
	var resp trailOrdersReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetTrailOrdersEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/list", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetTrailOrderDetail retrieves a futures trail order's detail
func (e *Exchange) GetTrailOrderDetail(ctx context.Context, settle currency.Code, orderID uint64) (*TrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == 0 {
		return nil, fmt.Errorf("%w: trail order ID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatUint(orderID, 10))
	var resp gateioAPIResponse[*trailOrderReply]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetTrailOrderDetailEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/detail", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil || resp.Data.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data.Order, nil
}

// UpdateTrailOrder amends a futures trail order, leaving parameters with empty values unchanged
func (e *Exchange) UpdateTrailOrder(ctx context.Context, settle currency.Code, arg *UpdateTrailOrderRequest) (*TrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ID == 0 {
		return nil, fmt.Errorf("%w: trail order ID is required", order.ErrOrderIDNotSet)
	}
	if arg.IsGreaterThanEqual != "" && arg.IsGreaterThanEqual != "true" && arg.IsGreaterThanEqual != "false" {
		return nil, fmt.Errorf("%w: is_gte_str %q", errInvalidTrailOrderFilter, arg.IsGreaterThanEqual)
	}
	if arg.PriceType > 3 {
		return nil, fmt.Errorf("%w: %d", errInvalidTriggerPriceType, arg.PriceType)
	}
	var resp trailOrderReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateTrailOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/update", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// GetTrailOrderChangeLog retrieves the creation and modification records of a futures trail order
func (e *Exchange) GetTrailOrderChangeLog(ctx context.Context, settle currency.Code, orderID, pageNumber, pageSize uint64) ([]*TrailOrderChangeResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == 0 {
		return nil, fmt.Errorf("%w: trail order ID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatUint(orderID, 10))
	if pageNumber > 0 {
		params.Set("page_num", strconv.FormatUint(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatUint(pageSize, 10))
	}
	var resp trailOrderChangeLogReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetTrailOrderChangeLogEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/change_log", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.ChangeLog, nil
}

// CreateChaseOrder creates a futures chase limit order that automatically adjusts its price to track the market
// until the full amount is filled or the order is stopped. A chase may be bounded by a price limit or by an offset
// limit, but not both.
func (e *Exchange) CreateChaseOrder(ctx context.Context, arg *CreateChaseOrderRequest) (*CreateChaseOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Amount == 0 {
		return nil, errInvalidOrderSize
	}
	if arg.PriceLimit != 0 && arg.OffsetLimit != 0 {
		return nil, errChasePriceAndOffsetLimit
	}
	var resp *CreateChaseOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCreateChaseOrderEPL, http.MethodPost, "futures/"+arg.Settle.Item.Lower+"/autoorder/v1/chase/create", nil, arg, &resp)
}

// StopChaseOrder stops a running futures chase limit order. Either the order ID or the custom
// text label must be provided to identify the order to stop.
func (e *Exchange) StopChaseOrder(ctx context.Context, settle currency.Code, arg *StopChaseOrderRequest) (*ChaseOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ID == "" && arg.Text == "" {
		return nil, errChaseOrderIDOrTextRequired
	}
	var resp chaseOrderReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualStopChaseOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/chase/stop", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// StopAllChaseOrders stops all running futures chase limit orders for the given settlement
// currency. The request may optionally scope the cancellation to a specific contract or margin mode.
func (e *Exchange) StopAllChaseOrders(ctx context.Context, settle currency.Code, arg *StopAllChaseOrdersRequest) ([]*ChaseOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	var resp chaseOrdersReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualStopAllChaseOrdersEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/chase/stop_all", nil, arg, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetChaseOrders retrieves a page of futures chase limit orders, sorted by creation time (1) or finish time (2).
// Finished orders are listed within a required time range.
func (e *Exchange) GetChaseOrders(ctx context.Context, settle currency.Code, arg *GetChaseOrdersRequest) ([]*ChaseOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SortBy != 1 && arg.SortBy != 2 {
		return nil, errInvalidChaseSortBy
	}
	if arg.IsFinished && (arg.StartAt.IsZero() || arg.EndAt.IsZero()) {
		return nil, errChaseTimeRangeRequired
	}
	if !arg.StartAt.IsZero() && !arg.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartAt, arg.EndAt); err != nil {
			return nil, err
		}
	}
	switch {
	case arg.PageSize > 100:
		return nil, fmt.Errorf("%w: %d exceeds the maximum page size of 100", errInvalidLimit, arg.PageSize)
	case arg.ReduceOnly > 2:
		return nil, fmt.Errorf("%w: reduce only %d", errInvalidChaseOrderFilter, arg.ReduceOnly)
	case arg.Side > 2:
		return nil, fmt.Errorf("%w: side %d", errInvalidChaseOrderFilter, arg.Side)
	}
	params := url.Values{}
	if arg.Contract.IsPopulated() {
		params.Set("contract", arg.Contract.String())
	}
	if arg.IsFinished {
		params.Set("is_finished", "true")
	}
	if !arg.StartAt.IsZero() {
		params.Set("start_at", strconv.FormatInt(arg.StartAt.Unix(), 10))
	}
	if !arg.EndAt.IsZero() {
		params.Set("end_at", strconv.FormatInt(arg.EndAt.Unix(), 10))
	}
	if arg.PageNumber > 0 {
		params.Set("page_num", strconv.FormatUint(arg.PageNumber, 10))
	}
	if arg.PageSize > 0 {
		params.Set("page_size", strconv.FormatUint(arg.PageSize, 10))
	}
	params.Set("sort_by", strconv.FormatUint(uint64(arg.SortBy), 10))
	if arg.HideCancelled {
		params.Set("hide_cancel", "true")
	}
	if arg.ReduceOnly > 0 {
		params.Set("reduce_only", strconv.FormatUint(uint64(arg.ReduceOnly), 10))
	}
	if arg.Side > 0 {
		params.Set("side", strconv.FormatUint(uint64(arg.Side), 10))
	}
	var resp chaseOrdersReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetChaseOrdersEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/autoorder/v1/chase/list", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetChaseOrderDetail retrieves the full detail of a single futures chase limit order by its ID.
func (e *Exchange) GetChaseOrderDetail(ctx context.Context, settle currency.Code, orderID string) (*ChaseOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w: chase order ID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("id", orderID)
	var resp chaseOrderReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetChaseOrderDetailEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/autoorder/v1/chase/detail", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// CreatePriceTriggeredFuturesOrder create a price-triggered order
func (e *Exchange) CreatePriceTriggeredFuturesOrder(ctx context.Context, settle currency.Code, arg *FuturesPriceTriggeredOrderRequest) (*TriggerOrderIDResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var response *TriggerOrderIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitTriggerOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/price_orders", nil, arg, &response)
}

// ListAllFuturesAutoOrders lists futures price triggered orders, optionally only those with an open or finished
// status
func (e *Exchange) ListAllFuturesAutoOrders(ctx context.Context, status string, settle currency.Code, contract currency.Pair, limit, offset uint64) ([]*PriceTriggeredOrderResponse, error) {
	if status != "" && status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status: %s", errInvalidOrderStatus, status)
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualListOpenOrdersEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/price_orders", params, nil, &response)
}

// CancelAllFuturesOpenOrders cancels every open futures price triggered order, optionally for one contract
func (e *Exchange) CancelAllFuturesOpenOrders(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelOpenOrdersEPL, http.MethodDelete, "futures/"+settle.Item.Lower+"/price_orders", params, nil, &response)
}

// GetSingleFuturesPriceTriggeredOrder retrieves a single price triggered order
func (e *Exchange) GetSingleFuturesPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetTriggerOrderEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// CancelFuturesPriceTriggeredOrder cancel a price-triggered order
func (e *Exchange) CancelFuturesPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelTriggerOrderEPL, http.MethodDelete, "futures/"+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// AmendFuturesPriceTriggeredOrder amends a pending futures price triggered order
func (e *Exchange) AmendFuturesPriceTriggeredOrder(ctx context.Context, settle currency.Code, arg *AmendFuturesPriceTriggeredOrderRequest) (*TriggerOrderIDResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Size == nil && arg.Amount == 0 && arg.Price == nil && arg.TriggerPrice == 0 && arg.PriceType == nil && arg.AutoSize == "" && !arg.Close {
		return nil, errNoOrderAmendment
	}
	if arg.Price != nil && *arg.Price < 0 {
		return nil, fmt.Errorf("%w: %v", limits.ErrPriceBelowMin, *arg.Price)
	}
	if arg.PriceType != nil && (*arg.PriceType < 0 || *arg.PriceType > 2) {
		return nil, fmt.Errorf("%w: %d; 0: latest trade price, 1: mark price, 2: index price", errInvalidTriggerPriceType, *arg.PriceType)
	}
	var resp *TriggerOrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAmendTriggerOrderEPL, http.MethodPut, "futures/"+settle.Item.Lower+"/price_orders/amend", nil, arg, &resp)
}

// validate validates the FuturesOrderCreateRequest
func (c *FuturesOrderCreateRequest) validate(isRest bool) error {
	if err := common.NilGuard(c); err != nil {
		return err
	}
	return validateOrderCreateParams(c.Contract, c.Size, c.Price, c.ClosePosition, c.AutoSize, c.TimeInForce, c.Text, c.Settle, isRest)
}
