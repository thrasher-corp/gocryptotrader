package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Sentinel errors for futures position requests
var (
	errMarginModeRequired     = errors.New("margin mode is required")
	errPositionModeRequired   = errors.New("position mode is required")
	errBusinessNumberRequired = errors.New("business number (bizNo) is required")
	errPositionValueRequired  = errors.New("position value is required")
	errRiskLimitLevelRequired = errors.New("risk limit level is required")
)

// GetFuturesMarginMode returns a contract's margin mode (Get Margin Mode)
func (e *Exchange) GetFuturesMarginMode(ctx context.Context, symbol string) (*FuturesMarginModeResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesMarginModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesMarginModeEPL, http.MethodGet, common.EncodeURLValues("/v2/position/getMarginMode", params), nil, &resp)
}

// SwitchFuturesMarginMode switches a contract's margin mode to ISOLATED or CROSS (Switch Margin Mode)
func (e *Exchange) SwitchFuturesMarginMode(ctx context.Context, symbol, marginMode string) (*FuturesMarginModeResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if marginMode == "" {
		return nil, errMarginModeRequired
	}
	body := make(map[string]any, 2)
	body["symbol"] = symbol
	body["marginMode"] = marginMode
	var resp *FuturesMarginModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, switchFuturesMarginModeEPL, http.MethodPost, "/v2/position/changeMarginMode", body, &resp)
}

// BatchSwitchFuturesMarginMode switches the margin mode of several contracts to ISOLATED or CROSS, returning the
// contracts KuCoin could not switch with their reasons (Batch Switch Margin Mode)
func (e *Exchange) BatchSwitchFuturesMarginMode(ctx context.Context, marginMode string, symbols []string) (*BatchSwitchFuturesMarginModeResponse, error) {
	if marginMode == "" {
		return nil, errMarginModeRequired
	}
	if err := checkSymbols(symbols); err != nil {
		return nil, err
	}
	body := make(map[string]any, 2)
	body["marginMode"] = marginMode
	body["symbols"] = symbols
	var resp *BatchSwitchFuturesMarginModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, batchSwitchFuturesMarginModeEPL, http.MethodPost, "/v2/position/batchChangeMarginMode", body, &resp)
}

// GetFuturesPositionMode returns the account's position mode, one-way or hedge, which applies to every contract (Get
// Position Mode)
func (e *Exchange) GetFuturesPositionMode(ctx context.Context) (*FuturesPositionModeResponse, error) {
	var resp *FuturesPositionModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesPositionModeEPL, http.MethodGet, "/v2/position/getPositionMode", nil, &resp)
}

// SwitchFuturesPositionMode switches the account's position mode for every contract: 0 for one-way mode and 1 for
// hedge mode (Switch Position Mode)
func (e *Exchange) SwitchFuturesPositionMode(ctx context.Context, positionMode string) (*FuturesPositionModeResponse, error) {
	if positionMode == "" {
		return nil, errPositionModeRequired
	}
	body := make(map[string]any, 1)
	body["positionMode"] = positionMode
	var resp *FuturesPositionModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, switchFuturesPositionModeEPL, http.MethodPost, "/v2/position/switchPositionMode", body, &resp)
}

// GetMaximumOpenPositionSize returns the largest positions a contract can open at a price and leverage (Get Max Open
// Size)
func (e *Exchange) GetMaximumOpenPositionSize(ctx context.Context, symbol string, price float64, leverage uint64) (*FuturesMaxOpenSizeResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if leverage == 0 {
		return nil, errInvalidLeverage
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	params.Set("leverage", strconv.FormatUint(leverage, 10))
	var resp *FuturesMaxOpenSizeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getMaximumOpenPositionSizeEPL, http.MethodGet, common.EncodeURLValues("/v2/getMaxOpenSize", params), nil, &resp)
}

// GetFuturesPositionDetails returns a contract's positions: one in one-way mode, and up to one per side in hedge mode
// (Get Position Details)
func (e *Exchange) GetFuturesPositionDetails(ctx context.Context, symbol string) ([]FuturesPositionDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []FuturesPositionDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesPositionDetailsEPL, http.MethodGet, common.EncodeURLValues("/v2/position", params), nil, &resp)
}

// GetFuturesPositionList returns the positions settled in a currency, such as USDT or XBT, or every position when
// settleCurrency is empty (Get Position List)
func (e *Exchange) GetFuturesPositionList(ctx context.Context, settleCurrency currency.Code) ([]FuturesPosition, error) {
	params := url.Values{}
	if !settleCurrency.IsEmpty() {
		params.Set("currency", settleCurrency.Upper().String())
	}
	var resp []FuturesPosition
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesPositionListEPL, http.MethodGet, common.EncodeURLValues("/v1/positions", params), nil, &resp)
}

// GetPositionHistory returns a page of closed futures positions (Get Positions History)
func (e *Exchange) GetPositionHistory(ctx context.Context, r *GetPositionHistoryRequest) (*FuturesPositionHistoryResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if !r.From.IsZero() && !r.To.IsZero() {
		if err := common.StartEndTimeCheck(r.From, r.To); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if r.Symbol != "" {
		params.Set("symbol", r.Symbol)
	}
	if !r.From.IsZero() {
		params.Set("from", strconv.FormatInt(r.From.UnixMilli(), 10))
	}
	if !r.To.IsZero() {
		params.Set("to", strconv.FormatInt(r.To.UnixMilli(), 10))
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.PageID != 0 {
		params.Set("pageId", strconv.FormatUint(r.PageID, 10))
	}
	var resp *FuturesPositionHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getPositionHistoryEPL, http.MethodGet, common.EncodeURLValues("/v1/history-positions", params), nil, &resp)
}

// GetMaxWithdrawMargin returns the most margin an isolated margin position can withdraw: USDT for USDT-margined
// contracts and coins for coin-margined contracts. positionSide is BOTH, the one-way mode default, LONG or SHORT;
// hedge mode requires LONG or SHORT (Get Max Withdraw Margin)
func (e *Exchange) GetMaxWithdrawMargin(ctx context.Context, symbol, positionSide string) (float64, error) {
	if symbol == "" {
		return 0, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if positionSide != "" {
		params.Set("positionSide", positionSide)
	}
	var resp types.Number
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getMaxWithdrawMarginEPL, http.MethodGet, common.EncodeURLValues("/v1/margin/maxWithdrawMargin", params), nil, &resp); err != nil {
		return 0, err
	}
	return resp.Float64(), nil
}

// GetFuturesCrossMarginLeverage returns a contract's cross margin leverage (Get Cross Margin Leverage)
func (e *Exchange) GetFuturesCrossMarginLeverage(ctx context.Context, symbol string) (*FuturesCrossMarginLeverageResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesCrossMarginLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesCrossMarginLeverageEPL, http.MethodGet, common.EncodeURLValues("/v2/getCrossUserLeverage", params), nil, &resp)
}

// ModifyFuturesCrossMarginLeverage sets a contract's cross margin leverage, returning whether KuCoin accepted it
// (Modify Cross Margin Leverage)
func (e *Exchange) ModifyFuturesCrossMarginLeverage(ctx context.Context, symbol string, leverage float64) (bool, error) {
	if symbol == "" {
		return false, currency.ErrSymbolStringEmpty
	}
	if leverage <= 0 {
		return false, errInvalidLeverage
	}
	body := make(map[string]any, 2)
	body["symbol"] = symbol
	body["leverage"] = strconv.FormatFloat(leverage, 'f', -1, 64)
	var resp bool
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, modifyFuturesCrossMarginLeverageEPL, http.MethodPost, "/v2/changeCrossUserLeverage", body, &resp)
}

// AddMargin adds margin to an isolated margin position, returning the position (Add Isolated Margin)
func (e *Exchange) AddMargin(ctx context.Context, r *AddMarginRequest) (*AddMarginResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Symbol == "":
		return nil, currency.ErrSymbolStringEmpty
	case r.Margin <= 0:
		return nil, fmt.Errorf("%w: margin must be positive", limits.ErrAmountBelowMin)
	case r.BusinessNumber == "":
		return nil, errBusinessNumberRequired
	}
	body := make(map[string]any, 3)
	body["symbol"] = r.Symbol
	body["margin"] = r.Margin
	body["bizNo"] = r.BusinessNumber
	if r.PositionSide != "" {
		body["positionSide"] = r.PositionSide
	}
	var resp *AddMarginResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, addMarginEPL, http.MethodPost, "/v1/position/margin/deposit-margin", body, &resp)
}

// RemoveMarginManually withdraws margin from an isolated margin position, returning the amount withdrawn: USDT for
// USDT-margined contracts and coins for coin-margined contracts (Remove Isolated Margin)
func (e *Exchange) RemoveMarginManually(ctx context.Context, r *RemoveMarginManuallyRequest) (float64, error) {
	if err := common.NilGuard(r); err != nil {
		return 0, err
	}
	if r.Symbol == "" {
		return 0, currency.ErrSymbolStringEmpty
	}
	if r.WithdrawAmount <= 0 {
		return 0, fmt.Errorf("%w: withdraw amount must be positive", limits.ErrAmountBelowMin)
	}
	body := make(map[string]any, 2)
	body["symbol"] = r.Symbol
	body["withdrawAmount"] = strconv.FormatFloat(r.WithdrawAmount, 'f', -1, 64)
	if r.PositionSide != "" {
		body["positionSide"] = r.PositionSide
	}
	var resp types.Number
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestFutures, removeMarginManuallyEPL, http.MethodPost, "/v1/margin/withdrawMargin", body, &resp); err != nil {
		return 0, err
	}
	return resp.Float64(), nil
}

// GetFuturesCrossMarginRiskLimit returns the largest cross margin positions contracts can open with a margin and
// leverage. Cross margin risk limits follow a smooth curve rather than fixed levels (Get Cross Margin Risk Limit)
func (e *Exchange) GetFuturesCrossMarginRiskLimit(ctx context.Context, r *GetFuturesCrossMarginRiskLimitRequest) ([]FuturesCrossMarginRiskLimit, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if err := checkSymbols(r.Symbols); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", strings.Join(r.Symbols, ","))
	if r.TotalMargin != 0 {
		params.Set("totalMargin", strconv.FormatFloat(r.TotalMargin, 'f', -1, 64))
	}
	if r.Leverage != 0 {
		params.Set("leverage", strconv.FormatUint(r.Leverage, 10))
	}
	var resp []FuturesCrossMarginRiskLimit
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesCrossMarginRiskLimitEPL, http.MethodGet, common.EncodeURLValues("/v2/batchGetCrossOrderLimit", params), nil, &resp)
}

// GetFuturesCrossMarginRequirement returns the size and cross margin rates of a position of a value in each contract
// (Get Cross Margin Requirement)
func (e *Exchange) GetFuturesCrossMarginRequirement(ctx context.Context, r *GetFuturesCrossMarginRequirementRequest) ([]FuturesCrossMarginRequirement, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if err := checkSymbols(r.Symbols); err != nil {
		return nil, err
	}
	if r.PositionValue <= 0 {
		return nil, errPositionValueRequired
	}
	body := make(map[string]any, 3)
	body["symbol"] = strings.Join(r.Symbols, ",")
	body["positionValue"] = strconv.FormatFloat(r.PositionValue, 'f', -1, 64)
	if r.Leverage != 0 {
		body["leverage"] = strconv.FormatFloat(r.Leverage, 'f', -1, 64)
	}
	var resp []FuturesCrossMarginRequirement
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getFuturesCrossMarginRequirementEPL, http.MethodPost, "/v2/getCrossModeMarginRequirement", body, &resp)
}

// checkSymbols returns an error when a symbol list or one of its symbols is empty
func checkSymbols(symbols []string) error {
	if len(symbols) == 0 {
		return currency.ErrSymbolStringEmpty
	}
	for i := range symbols {
		if symbols[i] == "" {
			return fmt.Errorf("%w: symbol %d", currency.ErrSymbolStringEmpty, i)
		}
	}
	return nil
}

// GetFuturesRiskLimitLevel returns a contract's isolated margin risk limit levels (Get Isolated Margin Risk Limit)
func (e *Exchange) GetFuturesRiskLimitLevel(ctx context.Context, symbol string) ([]FuturesRiskLimitLevel, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp []FuturesRiskLimitLevel
	return resp, e.SendHTTPRequest(ctx, exchange.RestFutures, getFuturesRiskLimitLevelEPL, "/v1/contracts/risk-limit/"+symbol, &resp)
}

// UpdateFuturesRiskLimitLevel moves an isolated margin position to a risk limit level, cancelling its open orders.
// KuCoin applies the change asynchronously, so the result only says whether it accepted the request; the position
// websocket channel reports the outcome (Modify Isolated Margin Risk Limit)
func (e *Exchange) UpdateFuturesRiskLimitLevel(ctx context.Context, symbol string, level uint64) (bool, error) {
	if symbol == "" {
		return false, currency.ErrSymbolStringEmpty
	}
	if level == 0 {
		return false, errRiskLimitLevelRequired
	}
	body := make(map[string]any, 2)
	body["symbol"] = symbol
	body["level"] = level
	var resp bool
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, updateFuturesRiskLimitLevelEPL, http.MethodPost, "/v1/position/risk-limit-level/change", body, &resp)
}
