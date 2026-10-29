package gate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errInvalidMMPWindow       = errors.New("market maker protection window must be whole milliseconds of at most 5 seconds")
	errInvalidMMPFrozenPeriod = errors.New("market maker protection frozen period must be non-negative whole milliseconds")
	errInvalidMMPLimit        = errors.New("invalid market maker protection limit")
)

// GetAllOptionsUnderlyings retrieves all option underlyings
func (e *Exchange) GetAllOptionsUnderlyings(ctx context.Context) ([]*OptionUnderlyingResponse, error) {
	var response []*OptionUnderlyingResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicUnderlyingOptionsEPL, "options/underlyings", &response)
}

// GetExpirationTimes returns every expiration time listed for the provided underlying, in the unsorted order Gate
// returns them
func (e *Exchange) GetExpirationTimes(ctx context.Context, underlying string) ([]time.Time, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	var timestamps []types.Time
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, publicExpirationOptionsEPL, common.EncodeURLValues("options/expirations", params), &timestamps); err != nil {
		return nil, err
	}
	expirations := make([]time.Time, len(timestamps))
	for i := range timestamps {
		expirations[i] = timestamps[i].Time()
	}
	return expirations, nil
}

// GetAllContractOfUnderlyingWithinExpiryDate retrieves list of contracts of the specified underlying and expiry time.
func (e *Exchange) GetAllContractOfUnderlyingWithinExpiryDate(ctx context.Context, underlying string, expTime time.Time) ([]*OptionContractResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if !expTime.IsZero() {
		params.Set("expiration", strconv.FormatInt(expTime.Unix(), 10))
	}
	var contracts []*OptionContractResponse
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicContractsOptionsEPL, common.EncodeURLValues("options/contracts", params), &contracts)
}

// GetOptionsSpecifiedContractDetail query specified contract detail
func (e *Exchange) GetOptionsSpecifiedContractDetail(ctx context.Context, contract currency.Pair) (*OptionContractResponse, error) {
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var contr *OptionContractResponse
	return contr, e.SendHTTPRequest(ctx, exchange.RestSpot, publicContractsOptionsEPL, "options/contracts/"+contract.String(), &contr)
}

// GetSettlementHistory retrieves list of settlement history
func (e *Exchange) GetSettlementHistory(ctx context.Context, underlying string, offset, limit uint64, from, to time.Time) ([]*OptionSettlementResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var settlements []*OptionSettlementResponse
	return settlements, e.SendHTTPRequest(ctx, exchange.RestSpot, publicSettlementOptionsEPL, common.EncodeURLValues("options/settlements", params), &settlements)
}

// GetOptionsSpecifiedContractsSettlement retrieves a contract's settlement at the given settlement time
func (e *Exchange) GetOptionsSpecifiedContractsSettlement(ctx context.Context, contract currency.Pair, underlying string, at time.Time) (*OptionSettlementResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if at.IsZero() {
		return nil, fmt.Errorf("%w: settlement time is required", common.ErrDateUnset)
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	params.Set("at", strconv.FormatInt(at.Unix(), 10))
	var settlement *OptionSettlementResponse
	return settlement, e.SendHTTPRequest(ctx, exchange.RestSpot, publicSettlementOptionsEPL, common.EncodeURLValues("options/settlements/"+contract.String(), params), &settlement)
}

// GetMyOptionsSettlements retrieves accounts option settlements.
func (e *Exchange) GetMyOptionsSettlements(ctx context.Context, underlying string, contract currency.Pair, offset, limit uint64, from, to time.Time) ([]*MyOptionSettlementResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var settlements []*MyOptionSettlementResponse
	return settlements, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsSettlementsEPL, http.MethodGet, "options/my_settlements", params, nil, &settlements)
}

// GetOptionsOrderbook returns the orderbook data for the given contract.
func (e *Exchange) GetOptionsOrderbook(ctx context.Context, contract currency.Pair, interval string, limit uint64, withOrderbookID bool) (*OrderbookResponse, error) {
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", strings.ToUpper(contract.String()))
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	var response *OrderbookResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookOptionsEPL, common.EncodeURLValues("options/order_book", params), &response)
}

// GetOptionsTickers lists tickers of options contracts
func (e *Exchange) GetOptionsTickers(ctx context.Context, underlying string) ([]*OptionsTickerResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", strings.ToUpper(underlying))
	var response []*OptionsTickerResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickerOptionsEPL, common.EncodeURLValues("options/tickers", params), &response)
}

// GetOptionUnderlyingTickers retrieves options underlying ticker
func (e *Exchange) GetOptionUnderlyingTickers(ctx context.Context, underlying string) (*OptionsUnderlyingTickerResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	var respos *OptionsUnderlyingTickerResponse
	return respos, e.SendHTTPRequest(ctx, exchange.RestSpot, publicUnderlyingTickerOptionsEPL, "options/underlying/tickers/"+underlying, &respos)
}

// GetOptionFuturesCandlesticks retrieves option futures candlesticks
func (e *Exchange) GetOptionFuturesCandlesticks(ctx context.Context, contract currency.Pair, interval kline.Interval, from, to time.Time, limit uint64) ([]*FuturesCandlestickResponse, error) {
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	var candles []*FuturesCandlestickResponse
	return candles, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleSticksOptionsEPL, common.EncodeURLValues("options/candlesticks", params), &candles)
}

// GetOptionFuturesMarkPriceCandlesticks retrieves index price candlesticks of an underlying
func (e *Exchange) GetOptionFuturesMarkPriceCandlesticks(ctx context.Context, underlying string, limit uint64, from, to time.Time, interval kline.Interval) ([]*FuturesCandlestickResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	var candles []*FuturesCandlestickResponse
	return candles, e.SendHTTPRequest(ctx, exchange.RestSpot, publicMarkpriceCandleSticksOptionsEPL, common.EncodeURLValues("options/underlying/candlesticks", params), &candles)
}

// GetOptionsTradeHistory retrieves options trade history
func (e *Exchange) GetOptionsTradeHistory(ctx context.Context, contract currency.Pair, callType string, offset, limit uint64, from, to time.Time) ([]*OptionTradeResponse, error) {
	params := url.Values{}
	callType = strings.ToUpper(callType)
	if callType == "C" || callType == "P" {
		params.Set("type", callType)
	}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var trades []*OptionTradeResponse
	return trades, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTradeHistoryOptionsEPL, common.EncodeURLValues("options/trades", params), &trades)
}

// GetOptionAccount lists option accounts
func (e *Exchange) GetOptionAccount(ctx context.Context) (*OptionAccountResponse, error) {
	var resp *OptionAccountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsAccountsEPL, http.MethodGet, "options/accounts", nil, nil, &resp)
}

// GetAccountChangingHistory retrieves list of account changing history
// possible change type values are: "dnw": Deposit & Withdrawal, "prem": Trading premium, "fee": Trading fee, "refr": Referrer rebate, and "set": Settlement P&L
func (e *Exchange) GetAccountChangingHistory(ctx context.Context, offset, limit uint64, from, to time.Time, changingType string) ([]*AccountBookResponse, error) {
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if changingType != "" {
		params.Set("type", changingType)
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var accountBook []*AccountBookResponse
	return accountBook, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsAccountBooksEPL, http.MethodGet, "options/account_book", params, nil, &accountBook)
}

// GetUsersPositionSpecifiedUnderlying lists user's positions of specified underlying
func (e *Exchange) GetUsersPositionSpecifiedUnderlying(ctx context.Context, underlying string) ([]*UsersPositionForUnderlyingResponse, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	var response []*UsersPositionForUnderlyingResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsPositions, http.MethodGet, "options/positions", params, nil, &response)
}

// GetSpecifiedContractPosition retrieves specified contract position
func (e *Exchange) GetSpecifiedContractPosition(ctx context.Context, contract currency.Pair) (*UsersPositionForUnderlyingResponse, error) {
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response *UsersPositionForUnderlyingResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsPositions, http.MethodGet, "options/positions/"+contract.String(), nil, nil, &response)
}

// GetUsersLiquidationHistoryForSpecifiedUnderlying retrieves user's liquidation history of specified underlying
func (e *Exchange) GetUsersLiquidationHistoryForSpecifiedUnderlying(ctx context.Context, underlying string, contract currency.Pair) ([]*ContractClosePositionResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*ContractClosePositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsLiquidationHistoryEPL, http.MethodGet, "options/position_close", params, nil, &response)
}

// PlaceOptionOrder creates an options order
// A zero size with Close set closes the position, and a zero price with an IOC time in force places a market order
func (e *Exchange) PlaceOptionOrder(ctx context.Context, arg *OptionOrderRequest) (*OptionOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Close != (arg.OrderSize == 0) {
		return nil, fmt.Errorf("%w: size must be zero when closing a position and non-zero otherwise", errInvalidOrderSize)
	}
	if arg.Price < 0 {
		return nil, fmt.Errorf("%w: price must not be negative", limits.ErrPriceBelowMin)
	}
	var response *OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsSubmitOrderEPL, http.MethodPost, "options/orders", nil, arg, &response)
}

// GetOptionFuturesOrders retrieves options orders with the given status
func (e *Exchange) GetOptionFuturesOrders(ctx context.Context, contract currency.Pair, underlying, status string, offset, limit uint64, from, to time.Time) ([]*OptionOrderResponse, error) {
	status = strings.ToLower(status)
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w: %q; only 'open' and 'finished' are supported", errInvalidOrderStatus, status)
	}
	params := url.Values{}
	params.Set("status", status)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsOrdersEPL, http.MethodGet, "options/orders", params, nil, &response)
}

// CancelMultipleOptionOpenOrders cancels all open orders matched
func (e *Exchange) CancelMultipleOptionOpenOrders(ctx context.Context, contract currency.Pair, underlying, side string) ([]*OptionOrderResponse, error) {
	// Dropping an unrecognised side would widen the cancellation to both sides of the book
	if side != "" && side != order.Ask.Lower() && side != order.Bid.Lower() {
		return nil, fmt.Errorf("%w: %q; only 'ask' and 'bid' are supported", order.ErrSideIsInvalid, side)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	if side != "" {
		params.Set("side", side)
	}
	var response []*OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsCancelOrdersEPL, http.MethodDelete, "options/orders", params, nil, &response)
}

// AmendOptionOrder amends the price and size of an open options order
func (e *Exchange) AmendOptionOrder(ctx context.Context, orderID string, arg *AmendOptionOrderRequest) (*OptionOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Price <= 0 {
		return nil, fmt.Errorf("%w: price must be positive", limits.ErrPriceBelowMin)
	}
	if arg.Size == 0 {
		return nil, fmt.Errorf("%w: size must not be zero", errInvalidOrderSize)
	}
	var response *OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsAmendOrderEPL, http.MethodPut, "options/orders/"+orderID, nil, arg, &response)
}

// GetSingleOptionOrder retrieves a single option order
func (e *Exchange) GetSingleOptionOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var o *OptionOrderResponse
	return o, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsOrderEPL, http.MethodGet, "options/orders/"+orderID, nil, nil, &o)
}

// CancelOptionSingleOrder cancel a single order.
func (e *Exchange) CancelOptionSingleOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsCancelOrderEPL, http.MethodDelete, "options/orders/"+orderID, nil, nil, &response)
}

// CountdownCancelOptionOrders sets a countdown after which Gate cancels open options orders unless the countdown is
// renewed before it ends; a zero timeout clears the countdown
func (e *Exchange) CountdownCancelOptionOrders(ctx context.Context, arg *CountdownCancelOptionOrdersRequest) (*TriggerTimeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if err := validateCountdownTimeout(arg.Countdown.Duration()); err != nil {
		return nil, err
	}
	var response *TriggerTimeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsCountdownCancelEPL, http.MethodPost, "options/countdown_cancel_all", nil, arg, &response)
}

// GetMyOptionsTradingHistory retrieves authenticated account's option trading history
func (e *Exchange) GetMyOptionsTradingHistory(ctx context.Context, underlying string, contract currency.Pair, offset, limit uint64, from, to time.Time) ([]*OptionTradingHistoryResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var resp []*OptionTradingHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsTradingHistoryEPL, http.MethodGet, "options/my_trades", params, nil, &resp)
}

// SetOptionsMarketMakerProtection sets the market maker protection of an options underlying
func (e *Exchange) SetOptionsMarketMakerProtection(ctx context.Context, arg *OptionsMarketMakerProtectionRequest) (*OptionsMarketMakerProtectionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Underlying == "" {
		return nil, errInvalidUnderlying
	}
	// Gate takes whole milliseconds, and truncating a sub-millisecond window to zero would disable protection
	if arg.Window < 0 || arg.Window > 5*time.Second || arg.Window%time.Millisecond != 0 {
		return nil, fmt.Errorf("%w: %s", errInvalidMMPWindow, arg.Window)
	}
	if arg.FrozenPeriod < 0 || arg.FrozenPeriod%time.Millisecond != 0 {
		return nil, fmt.Errorf("%w: %s", errInvalidMMPFrozenPeriod, arg.FrozenPeriod)
	}
	if arg.QuantityLimit <= 0 {
		return nil, fmt.Errorf("%w: quantity limit must be positive", errInvalidMMPLimit)
	}
	if arg.DeltaLimit <= 0 {
		return nil, fmt.Errorf("%w: delta limit must be positive", errInvalidMMPLimit)
	}
	var response *OptionsMarketMakerProtectionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsSetMMPEPL, http.MethodPost, "options/mmp", nil, arg, &response)
}

// GetOptionsMarketMakerProtection retrieves the market maker protection of an options underlying, or of every
// underlying when underlying is empty
func (e *Exchange) GetOptionsMarketMakerProtection(ctx context.Context, underlying string) ([]*OptionsMarketMakerProtectionResponse, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	var response []*OptionsMarketMakerProtectionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsGetMMPEPL, http.MethodGet, "options/mmp", params, nil, &response)
}

// ResetOptionsMarketMakerProtection resets the market maker protection of an options underlying, unfreezing trading
// after the protection triggers
func (e *Exchange) ResetOptionsMarketMakerProtection(ctx context.Context, underlying string) (*OptionsMarketMakerProtectionResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	var response *OptionsMarketMakerProtectionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsResetMMPEPL, http.MethodPost, "options/mmp/reset", nil, map[string]string{"underlying": underlying}, &response)
}
