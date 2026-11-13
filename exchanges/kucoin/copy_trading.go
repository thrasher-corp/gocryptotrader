package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Copy trading validation errors
var (
	errMarginModeChangesRequired  = errors.New("at least one margin mode change is required")
	errMarginModeChangeIncomplete = errors.New("margin mode change needs a symbol and a margin mode")
)

// AddCopyTradingOrder places a lead trader's copy trading futures order (Add Order)
func (e *Exchange) AddCopyTradingOrder(ctx context.Context, r *CopyTradingOrderRequest) (*CopyTradingOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *CopyTradingOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, addCopyTradingOrderEPL, http.MethodPost, "/v1/copy-trade/futures/orders", body, &resp)
}

// AddCopyTradingOrderTest validates a copy trading futures order and its signature without placing it (Add Order
// Test)
func (e *Exchange) AddCopyTradingOrderTest(ctx context.Context, r *CopyTradingOrderRequest) (*CopyTradingOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	// Add Order Test requires the leverage Add Order leaves optional; closing a position still does without it
	if r.Leverage == 0 && !r.CloseOrder {
		return nil, errInvalidLeverage
	}
	var resp *CopyTradingOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, addCopyTradingOrderTestEPL, http.MethodPost, "/v1/copy-trade/futures/orders/test", body, &resp)
}

// AddCopyTradingTPSLOrder places a copy trading futures order with take profit and stop loss triggers (Add Take Profit
// And Stop Loss Order)
func (e *Exchange) AddCopyTradingTPSLOrder(ctx context.Context, r *CopyTradingTPSLOrderRequest) (*CopyTradingOrderResponse, error) {
	body, err := r.Payload()
	if err != nil {
		return nil, err
	}
	var resp *CopyTradingOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, addCopyTradingTPSLOrderEPL, http.MethodPost, "/v1/copy-trade/futures/st-orders", body, &resp)
}

// CancelCopyTradingOrderByID cancels a copy trading futures order, a stop order included, by its order ID (Cancel
// Order By OrderId). KuCoin cancels asynchronously, so the order's status confirms the cancellation
func (e *Exchange) CancelCopyTradingOrderByID(ctx context.Context, orderID, fundsType string) (*CancelCopyTradingOrderByIDResponse, error) {
	// The docs mark orderId optional, but an order ID is what this cancels by
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	if fundsType != "" {
		params.Set("fundsType", fundsType)
	}
	var resp *CancelCopyTradingOrderByIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, cancelCopyTradingOrderByIDEPL, http.MethodDelete, common.EncodeURLValues("/v1/copy-trade/futures/orders", params), nil, &resp)
}

// CancelCopyTradingOrderByClientOrderID cancels a copy trading futures order, a stop order included, by its client
// order ID (Cancel Order By ClientOid). KuCoin cancels asynchronously, so the order's status confirms the cancellation
func (e *Exchange) CancelCopyTradingOrderByClientOrderID(ctx context.Context, r *CancelCopyTradingOrderByClientOrderIDRequest) (*CancelCopyTradingOrderByClientOrderIDResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	params := url.Values{}
	params.Set("symbol", r.Symbol)
	params.Set("clientOid", r.ClientOrderID)
	if r.FundsType != "" {
		params.Set("fundsType", r.FundsType)
	}
	var resp *CancelCopyTradingOrderByClientOrderIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, cancelCopyTradingOrderByClientOrderIDEPL, http.MethodDelete, common.EncodeURLValues("/v1/copy-trade/futures/orders/client-order", params), nil, &resp)
}

// GetCopyTradingMaxOpenSize returns the largest copy trading futures position that can be opened at a price and
// leverage (Get Max Open Size)
func (e *Exchange) GetCopyTradingMaxOpenSize(ctx context.Context, r *CopyTradingMaxOpenSizeRequest) (*CopyTradingMaxOpenSizeResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if r.Leverage == 0 {
		return nil, errInvalidLeverage
	}
	params := url.Values{}
	params.Set("symbol", r.Symbol)
	params.Set("price", strconv.FormatFloat(r.Price, 'f', -1, 64))
	params.Set("leverage", strconv.FormatUint(r.Leverage, 10))
	if r.FundsType != "" {
		params.Set("fundsType", r.FundsType)
	}
	var resp *CopyTradingMaxOpenSizeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getCopyTradingMaxOpenSizeEPL, http.MethodGet, common.EncodeURLValues("/v1/copy-trade/futures/get-max-open-size", params), nil, &resp)
}

// GetCopyTradingMaxWithdrawMargin returns the most margin that can be withdrawn from a copy trading futures position:
// USDT for USDT-margined contracts and coins for coin-margined ones (Get Max Withdraw Margin)
func (e *Exchange) GetCopyTradingMaxWithdrawMargin(ctx context.Context, r *CopyTradingMaxWithdrawMarginRequest) (float64, error) {
	if err := common.NilGuard(r); err != nil {
		return 0, err
	}
	if r.Symbol == "" {
		return 0, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", r.Symbol)
	if r.PositionSide != "" {
		params.Set("positionSide", r.PositionSide)
	}
	if r.FundsType != "" {
		params.Set("fundsType", r.FundsType)
	}
	var resp types.Number
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getCopyTradingMaxWithdrawMarginEPL, http.MethodGet, common.EncodeURLValues("/v1/copy-trade/futures/position/margin/max-withdraw-margin", params), nil, &resp); err != nil {
		return 0, err
	}
	return resp.Float64(), nil
}

// AddCopyTradingIsolatedMargin adds margin to an isolated margin copy trading futures position (Add Isolated Margin)
func (e *Exchange) AddCopyTradingIsolatedMargin(ctx context.Context, r *AddCopyTradingIsolatedMarginRequest) (*AddCopyTradingIsolatedMarginResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.Margin <= 0 {
		return nil, fmt.Errorf("%w: margin must be positive", limits.ErrAmountBelowMin)
	}
	if r.BusinessNumber == "" {
		return nil, errBusinessNumberRequired
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	body["margin"] = r.Margin // documented as a JSON number, unlike most decimals
	body["bizNo"] = r.BusinessNumber
	if r.PositionSide != "" {
		body["positionSide"] = r.PositionSide
	}
	if r.FundsType != "" {
		body["fundsType"] = r.FundsType
	}
	var resp *AddCopyTradingIsolatedMarginResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, addCopyTradingIsolatedMarginEPL, http.MethodPost, "/v1/copy-trade/futures/position/margin/deposit-margin", body, &resp)
}

// RemoveCopyTradingIsolatedMargin withdraws margin from an isolated margin copy trading futures position, returning
// the amount withdrawn: USDT for USDT-margined contracts and coins for coin-margined ones (Remove Isolated Margin)
func (e *Exchange) RemoveCopyTradingIsolatedMargin(ctx context.Context, r *RemoveCopyTradingIsolatedMarginRequest) (float64, error) {
	if err := common.NilGuard(r); err != nil {
		return 0, err
	}
	if r.Symbol == "" {
		return 0, currency.ErrSymbolStringEmpty
	}
	if r.WithdrawAmount <= 0 {
		return 0, fmt.Errorf("%w: withdraw amount must be positive", limits.ErrAmountBelowMin)
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	body["withdrawAmount"] = r.WithdrawAmount // documented as a JSON number, unlike most decimals
	if r.PositionSide != "" {
		body["positionSide"] = r.PositionSide
	}
	if r.FundsType != "" {
		body["fundsType"] = r.FundsType
	}
	var resp types.Number
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestFutures, removeCopyTradingIsolatedMarginEPL, http.MethodPost, "/v1/copy-trade/futures/position/margin/withdraw-margin", body, &resp); err != nil {
		return 0, err
	}
	return resp.Float64(), nil
}

// ModifyCopyTradingIsolatedMarginRiskLimit changes the risk limit level of an isolated margin copy trading futures
// position, which cancels its open orders (Modify Isolated Margin Risk Limit). The reply only says whether KuCoin
// accepted the request; the position change websocket events carry the outcome
func (e *Exchange) ModifyCopyTradingIsolatedMarginRiskLimit(ctx context.Context, symbol string, level uint64) (bool, error) {
	if symbol == "" {
		return false, currency.ErrSymbolStringEmpty
	}
	if level == 0 {
		return false, errRiskLimitLevelRequired
	}
	body := make(map[string]any)
	body["symbol"] = symbol
	body["level"] = level
	var resp bool
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, modifyCopyTradingIsolatedMarginRiskLimitEPL, http.MethodPost, "/v1/copy-trade/futures/position/risk-limit-level/change", body, &resp)
}

// ModifyCopyTradingAutoDepositStatus turns automatic margin deposits for an isolated margin copy trading futures
// position on or off (Modify Isolated Margin Auto-Deposit Status). KuCoin no longer recommends isolated margin
func (e *Exchange) ModifyCopyTradingAutoDepositStatus(ctx context.Context, r *CopyTradingAutoDepositStatusRequest) (bool, error) {
	if err := common.NilGuard(r); err != nil {
		return false, err
	}
	if r.Symbol == "" {
		return false, currency.ErrSymbolStringEmpty
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	body["status"] = r.Status
	if r.PositionSide != "" {
		body["positionSide"] = r.PositionSide
	}
	if r.FundsType != "" {
		body["fundsType"] = r.FundsType
	}
	var resp bool
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, modifyCopyTradingAutoDepositStatusEPL, http.MethodPost, "/v1/copy-trade/futures/position/margin/auto-deposit-status", body, &resp)
}

// SwitchCopyTradingMarginMode switches a copy trading futures symbol between isolated and cross margin (Switch Margin
// Mode)
func (e *Exchange) SwitchCopyTradingMarginMode(ctx context.Context, r *SwitchCopyTradingMarginModeRequest) (*SwitchCopyTradingMarginModeResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.MarginMode == "" {
		return nil, errMarginModeRequired
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	body["marginMode"] = r.MarginMode
	if r.FundsType != "" {
		body["fundsType"] = r.FundsType
	}
	var resp *SwitchCopyTradingMarginModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, switchCopyTradingMarginModeEPL, http.MethodPost, "/v1/copy-trade/futures/position/changeMarginMode", body, &resp)
}

// ModifyCopyTradingCrossMarginLeverage changes a copy trading futures symbol's cross margin leverage (Modify Cross
// Margin Leverage)
func (e *Exchange) ModifyCopyTradingCrossMarginLeverage(ctx context.Context, r *CopyTradingCrossMarginLeverageRequest) (bool, error) {
	if err := common.NilGuard(r); err != nil {
		return false, err
	}
	if r.Symbol == "" {
		return false, currency.ErrSymbolStringEmpty
	}
	if r.Leverage == 0 {
		return false, errInvalidLeverage
	}
	body := make(map[string]any)
	body["symbol"] = r.Symbol
	body["leverage"] = strconv.FormatUint(r.Leverage, 10)
	if r.FundsType != "" {
		body["fundsType"] = r.FundsType
	}
	var resp bool
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, modifyCopyTradingCrossMarginLeverageEPL, http.MethodPost, "/v2/copy-trade/futures/changeCrossUserLeverage", body, &resp)
}

// GetCopyTradingCrossMarginRequirement returns the cross margin rates of copy trading futures symbols at a position
// value (Get Cross Margin Requirement)
func (e *Exchange) GetCopyTradingCrossMarginRequirement(ctx context.Context, r *CopyTradingCrossMarginRequirementRequest) ([]CopyTradingCrossMarginRequirementResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if len(r.Symbols) == 0 || slices.Contains(r.Symbols, "") {
		return nil, currency.ErrSymbolStringEmpty
	}
	if r.Leverage == 0 {
		return nil, errInvalidLeverage
	}
	if r.PositionValue <= 0 {
		return nil, errPositionValueRequired
	}
	body := make(map[string]any)
	body["symbol"] = strings.Join(r.Symbols, ",")
	body["leverage"] = strconv.FormatUint(r.Leverage, 10)
	body["positionValue"] = strconv.FormatFloat(r.PositionValue, 'f', -1, 64)
	var resp []CopyTradingCrossMarginRequirementResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getCopyTradingCrossMarginRequirementEPL, http.MethodPost, "/v2/copy-trade/getCrossModeMarginRequirement", body, &resp)
}

// SwitchCopyTradingPositionMode switches every copy trading futures symbol between one-way mode, positionMode 0, and
// hedge mode, positionMode 1 (Switch Position Mode)
func (e *Exchange) SwitchCopyTradingPositionMode(ctx context.Context, positionMode, fundsType string) (*SwitchCopyTradingPositionModeResponse, error) {
	if positionMode == "" {
		return nil, errPositionModeRequired
	}
	body := make(map[string]any)
	body["positionMode"] = positionMode
	if fundsType != "" {
		body["fundsType"] = fundsType
	}
	var resp *SwitchCopyTradingPositionModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, switchCopyTradingPositionModeEPL, http.MethodPost, "/v2/copy-trade/position/switchPositionMode", body, &resp)
}

// GetCopyTradingActiveOrders returns a page of a lead trader's active copy trading futures orders (Get active copy
// trading futures orders)
func (e *Exchange) GetCopyTradingActiveOrders(ctx context.Context, r *CopyTradingPageRequest) (*CopyTradingOrdersResponse, error) {
	params, err := r.values()
	if err != nil {
		return nil, err
	}
	var resp *CopyTradingOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getCopyTradingActiveOrdersEPL, http.MethodGet, common.EncodeURLValues("/v1/copy-trade/futures/orders", params), nil, &resp)
}

// GetCopyTradingFills returns a page of a lead trader's copy trading futures fills (Get copy trading futures fills)
func (e *Exchange) GetCopyTradingFills(ctx context.Context, r *CopyTradingPageRequest) (*CopyTradingFillsResponse, error) {
	params, err := r.values()
	if err != nil {
		return nil, err
	}
	var resp *CopyTradingFillsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getCopyTradingFillsEPL, http.MethodGet, common.EncodeURLValues("/v1/copy-trade/futures/fills", params), nil, &resp)
}

// GetCopyTradingPositions returns a lead trader's copy trading futures positions, of one funds type or, when fundsType
// is empty or all, of both (Get copy trading futures positions)
func (e *Exchange) GetCopyTradingPositions(ctx context.Context, fundsType string) ([]CopyTradingPositionResponse, error) {
	params := url.Values{}
	if fundsType != "" {
		params.Set("fundsType", fundsType)
	}
	var resp []CopyTradingPositionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getCopyTradingPositionsEPL, http.MethodGet, common.EncodeURLValues("/v1/copy-trade/futures/positions", params), nil, &resp)
}

// GetCopyTradingOrderHistory returns a page of a lead trader's past copy trading futures orders (Get copy trading
// futures order history)
func (e *Exchange) GetCopyTradingOrderHistory(ctx context.Context, r *CopyTradingPageRequest) (*CopyTradingOrdersResponse, error) {
	params, err := r.values()
	if err != nil {
		return nil, err
	}
	var resp *CopyTradingOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getCopyTradingOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("/v1/copy-trade/futures/orders/history", params), nil, &resp)
}

// GetCopyTradingAccountOverview returns a lead trader's copy trading futures account overview, for self funds and lead
// trial funds (Get copy trading futures account overview)
func (e *Exchange) GetCopyTradingAccountOverview(ctx context.Context) (*CopyTradingAccountOverviewResponse, error) {
	var resp *CopyTradingAccountOverviewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, getCopyTradingAccountOverviewEPL, http.MethodGet, "/v1/copy-trade/futures/account-overview", nil, &resp)
}

// BatchSwitchCopyTradingMarginMode switches the margin mode of up to 100 copy trading futures symbols (Batch switch
// copy trading futures margin mode). Each change succeeds or fails on its own: check every item's Status
func (e *Exchange) BatchSwitchCopyTradingMarginMode(ctx context.Context, r *BatchSwitchCopyTradingMarginModeRequest) (*BatchSwitchCopyTradingMarginModeResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if len(r.Items) == 0 {
		return nil, errMarginModeChangesRequired
	}
	items := make([]map[string]any, len(r.Items))
	for i := range r.Items {
		if r.Items[i].Symbol == "" || r.Items[i].MarginMode == "" {
			return nil, fmt.Errorf("%w: item %d", errMarginModeChangeIncomplete, i)
		}
		item := make(map[string]any)
		item["symbol"] = r.Items[i].Symbol
		item["marginMode"] = r.Items[i].MarginMode
		if r.Items[i].FundsType != "" {
			item["fundsType"] = r.Items[i].FundsType
		}
		items[i] = item
	}
	body := make(map[string]any)
	body["items"] = items
	if r.FundsType != "" {
		body["fundsType"] = r.FundsType
	}
	var resp *BatchSwitchCopyTradingMarginModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFutures, batchSwitchCopyTradingMarginModeEPL, http.MethodPost, "/v1/copy-trade/futures/position/changeMarginModeBatch", body, &resp)
}
