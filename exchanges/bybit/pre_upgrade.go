package bybit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetPreUpgradeOrderHistory returns the orders placed before the account was upgraded to a unified account (Get
// Pre-upgrade Order History). Bybit plans to drop the spot category at the end of August 2026
func (e *Exchange) GetPreUpgradeOrderHistory(ctx context.Context, req *PreUpgradeOrderHistoryRequest) (*PreUpgradeOrderHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	// Bybit takes both bounds of the window or neither
	if !req.StartTime.IsZero() || !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if req.OrderFilter != "" {
		params.Set("orderFilter", req.OrderFilter)
	}
	if req.OrderStatus != "" {
		params.Set("orderStatus", req.OrderStatus)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *PreUpgradeOrderHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/order/history", params, nil, &resp, defaultEPL)
}

// GetPreUpgradeTradeHistory returns the executions made before the account was upgraded to a unified account, newest
// first (Get Pre-upgrade Trade History). Bybit plans to drop the spot category at the end of August 2026
func (e *Exchange) GetPreUpgradeTradeHistory(ctx context.Context, req *PreUpgradeTradeHistoryRequest) (*PreUpgradeTradeHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if req.ExecutionType != "" {
		params.Set("execType", req.ExecutionType)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *PreUpgradeTradeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/execution/list", params, nil, &resp, defaultEPL)
}

// GetPreUpgradeClosedPNL returns the closed profit and loss records made before the account was upgraded to a unified
// account, newest first (Get Pre-upgrade Closed PnL)
func (e *Exchange) GetPreUpgradeClosedPNL(ctx context.Context, req *PreUpgradeClosedPNLRequest) (*PreUpgradeClosedPNLResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	params.Set("symbol", req.Symbol)
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *PreUpgradeClosedPNLResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/position/closed-pnl", params, nil, &resp, defaultEPL)
}

// GetPreUpgradeTransactionLog returns the USDC derivatives wallet transaction logs made before the account was
// upgraded to a unified account (Get Pre-upgrade Transaction Log)
func (e *Exchange) GetPreUpgradeTransactionLog(ctx context.Context, req *PreUpgradeTransactionLogRequest) (*PreUpgradeTransactionLogResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *PreUpgradeTransactionLogResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/account/transaction-log", params, nil, &resp, defaultEPL)
}

// GetPreUpgradeOptionDeliveryRecord returns the option deliveries made before the account was upgraded to a unified
// account, newest first (Get Pre-upgrade Delivery Record)
func (e *Exchange) GetPreUpgradeOptionDeliveryRecord(ctx context.Context, req *PreUpgradeDeliveryRecordRequest) (*PreUpgradeDeliveryRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.ExpiryDate.IsZero() {
		params.Set("expDate", expiryDate(req.ExpiryDate))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *PreUpgradeDeliveryRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/asset/delivery-record", params, nil, &resp, defaultEPL)
}

// GetPreUpgradeUSDCSessionSettlement returns the USDC perpetual session settlements made before the account was
// upgraded to a unified account (Get Pre-upgrade USDC Session Settlement)
func (e *Exchange) GetPreUpgradeUSDCSessionSettlement(ctx context.Context, req *PreUpgradeSessionSettlementRequest) (*PreUpgradeSessionSettlementResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *PreUpgradeSessionSettlementResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/asset/settlement-record", params, nil, &resp, defaultEPL)
}
