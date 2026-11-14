package binanceus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errRailRequired             = errors.New("custodial partner rail is required")
	errOriginTransferIDRequired = errors.New("origin transfer ID is required")
)

// GetCustodialAccountBalance returns the balances of the Binance.US exchange wallet and of the custodial sub-account
// held with rail, the custodial partner
func (e *Exchange) GetCustodialAccountBalance(ctx context.Context, rail string) (*CustodialAccountBalanceResponse, error) {
	params, err := custodialParams(rail)
	if err != nil {
		return nil, err
	}
	var resp *CustodialAccountBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/custodian/balance", params, defaultRate, &resp)
}

// GetCustodialSupportedAssetList returns the assets that can be transferred from rail, the custodial partner, and the
// assets that can be settled to it
func (e *Exchange) GetCustodialSupportedAssetList(ctx context.Context, rail string) (*CustodialSupportedAssetListResponse, error) {
	params, err := custodialParams(rail)
	if err != nil {
		return nil, err
	}
	var resp *CustodialSupportedAssetListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/custodian/supportedAssetList", params, defaultRate, &resp)
}

// TransferFromExchangeWallet transfers an asset from the Binance.US exchange wallet to the custodial sub-account
func (e *Exchange) TransferFromExchangeWallet(ctx context.Context, req *CustodialAssetTransferRequest) (*TransferFromExchangeWalletResponse, error) {
	params, err := custodialAssetTransferParams(req)
	if err != nil {
		return nil, err
	}
	var resp *TransferFromExchangeWalletResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/custodian/walletTransfer", params, defaultRate, &resp)
}

// TransferFromCustodian transfers an asset from the custodial partner's account to the custodial sub-account
func (e *Exchange) TransferFromCustodian(ctx context.Context, req *CustodialAssetTransferRequest) (*TransferFromCustodianResponse, error) {
	params, err := custodialAssetTransferParams(req)
	if err != nil {
		return nil, err
	}
	var resp *TransferFromCustodianResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/custodian/custodianTransfer", params, defaultRate, &resp)
}

// UndoCustodialTransfer undoes a transfer from rail, the custodial partner, by the transfer's ID
func (e *Exchange) UndoCustodialTransfer(ctx context.Context, rail, originTransferID string) (*UndoCustodialTransferResponse, error) {
	params, err := custodialParams(rail)
	if err != nil {
		return nil, err
	}
	if originTransferID == "" {
		return nil, errOriginTransferIDRequired
	}
	params.Set("originTransferId", originTransferID)
	var resp *UndoCustodialTransferResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/custodian/undoTransfer", params, defaultRate, &resp)
}

// GetExchangeWalletTransfer returns transfers from the exchange wallet to the custodial sub-account, a page at a time
func (e *Exchange) GetExchangeWalletTransfer(ctx context.Context, req *ExchangeWalletTransferRequest) (*ExchangeWalletTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	if req.TransferID != "" {
		params.Set("transferId", req.TransferID)
	}
	if req.ClientOrderID != "" {
		params.Set("clientOrderId", req.ClientOrderID)
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *ExchangeWalletTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/custodian/walletTransferHistory", params, defaultRate, &resp)
}

// GetCustodianTransfer returns transfers from the custodial partner, a page at a time: custodian transfers, ExpressTrade
// transfers and undone transfers
func (e *Exchange) GetCustodianTransfer(ctx context.Context, req *CustodianTransferRequest) (*CustodianTransferResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	if req.TransferID != "" {
		params.Set("transferId", req.TransferID)
	}
	if req.ClientOrderID != "" {
		params.Set("clientOrderId", req.ClientOrderID)
	}
	if req.ExpressTradeTransfer {
		params.Set("expressTradeTransfer", "true")
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp *CustodianTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/custodian/custodianTransferHistory", params, defaultRate, &resp)
}

// CustodialNewOrder places an order in the custodial sub-account
func (e *Exchange) CustodialNewOrder(ctx context.Context, req *CustodialNewOrderRequest) (*CustodialNewOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	params.Set("symbol", symbol)
	params.Set("side", string(req.Side))
	params.Set("type", string(req.Type))
	if req.TimeInForce != "" {
		params.Set("timeInForce", string(req.TimeInForce))
	}
	if req.Quantity > 0 {
		params.Set("quantity", formatFloat(req.Quantity))
	}
	if req.QuoteOrderQuantity > 0 {
		params.Set("quoteOrderQty", formatFloat(req.QuoteOrderQuantity))
	}
	if req.Price > 0 {
		params.Set("price", formatFloat(req.Price))
	}
	if req.StopPrice > 0 {
		params.Set("stopPrice", formatFloat(req.StopPrice))
	}
	if req.IcebergQuantity > 0 {
		params.Set("icebergQty", formatFloat(req.IcebergQuantity))
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if req.AllowExpressTrade {
		params.Set("allowExpressTrade", "true")
	}
	var resp *CustodialNewOrderResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/custodian/order", params, orderRate, &resp)
}

// CustodialNewOneCancelsTheOtherOrder places a one-cancels-the-other (OCO) order list in the custodial sub-account,
// where the activation of one order immediately cancels the other. An OCO counts as two orders against the order rate
// limit
func (e *Exchange) CustodialNewOneCancelsTheOtherOrder(ctx context.Context, req *CustodialNewOneCancelsTheOtherOrderRequest) (*CustodialOneCancelsTheOtherOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Price <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	if req.StopPrice <= 0 {
		return nil, errStopPriceRequired
	}
	if req.StopLimitPrice > 0 && req.StopLimitTimeInForce == "" {
		return nil, errStopLimitTimeInForceRequired
	}
	params.Set("symbol", symbol)
	params.Set("side", string(req.Side))
	params.Set("quantity", formatFloat(req.Quantity))
	params.Set("price", formatFloat(req.Price))
	params.Set("stopPrice", formatFloat(req.StopPrice))
	if req.LimitClientOrderID != "" {
		params.Set("limitClientOrderId", req.LimitClientOrderID)
	}
	if req.LimitIcebergQuantity > 0 {
		params.Set("limitIcebergQty", formatFloat(req.LimitIcebergQuantity))
	}
	if req.StopClientOrderID != "" {
		params.Set("stopClientOrderId", req.StopClientOrderID)
	}
	if req.StopLimitPrice > 0 {
		params.Set("stopLimitPrice", formatFloat(req.StopLimitPrice))
	}
	if req.StopIcebergQuantity > 0 {
		params.Set("stopIcebergQty", formatFloat(req.StopIcebergQuantity))
	}
	if req.StopLimitTimeInForce != "" {
		params.Set("stopLimitTimeInForce", string(req.StopLimitTimeInForce))
	}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if req.AllowExpressTrade {
		params.Set("allowExpressTrade", "true")
	}
	var resp *CustodialOneCancelsTheOtherOrderResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/custodian/ocoOrder", params, oneCancelsTheOtherOrderRate, &resp)
}

// GetCustodialOpenOrders returns the custodial sub-account's open orders on symbol, or on every symbol for an empty
// pair
func (e *Exchange) GetCustodialOpenOrders(ctx context.Context, rail string, symbol currency.Pair) ([]CustodialOrderResponse, error) {
	params, err := custodialParams(rail)
	if err != nil {
		return nil, err
	}
	if !symbol.IsEmpty() {
		symbolValue, err := e.formatSymbol(symbol)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []CustodialOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/custodian/openOrders", params, defaultRate, &resp)
}

// GetCustodialOrder returns a custodial order
func (e *Exchange) GetCustodialOrder(ctx context.Context, req *CustodialOrderRequest) (*CustodialOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params.Set("symbol", symbol)
	params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	var resp *CustodialOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/custodian/order", params, defaultRate, &resp)
}

// GetCustodialOrderHistory returns the custodial sub-account's orders on the request's symbol, or on every symbol when
// it is empty
func (e *Exchange) GetCustodialOrderHistory(ctx context.Context, req *CustodialOrderHistoryRequest) ([]CustodialOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	if !req.Symbol.IsEmpty() {
		symbol, err := e.formatSymbol(req.Symbol)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.FromID != 0 {
		params.Set("fromId", strconv.FormatUint(req.FromID, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CustodialOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/custodian/orderHistory", params, defaultRate, &resp)
}

// GetCustodialTradeHistory returns the custodial sub-account's trades: from the request's FromID on when it is set,
// and otherwise the most recent
func (e *Exchange) GetCustodialTradeHistory(ctx context.Context, req *CustodialTradeHistoryRequest) ([]CustodialTradeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	if !req.Symbol.IsEmpty() {
		symbol, err := e.formatSymbol(req.Symbol)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.FromID != 0 {
		params.Set("fromId", strconv.FormatUint(req.FromID, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CustodialTradeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/custodian/tradeHistory", params, defaultRate, &resp)
}

// CancelCustodialOrder cancels an active custodial order by OrderID or OriginalClientOrderID
func (e *Exchange) CancelCustodialOrder(ctx context.Context, req *CancelCustodialOrderRequest) (*CancelCustodialOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.OrderID == 0 && req.OriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params.Set("symbol", symbol)
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if req.OriginalClientOrderID != "" {
		params.Set("origClientOrderId", req.OriginalClientOrderID)
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	var resp *CancelCustodialOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/sapi/v1/custodian/cancelOrder", params, cancelRate, &resp)
}

// CancelCustodialOpenOrdersForSymbol cancels every active custodial order on a symbol, OCO orders included
func (e *Exchange) CancelCustodialOpenOrdersForSymbol(ctx context.Context, rail string, symbol currency.Pair) ([]CancelCustodialOrderResponse, error) {
	params, err := custodialParams(rail)
	if err != nil {
		return nil, err
	}
	symbolValue, err := e.formatSymbol(symbol)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbolValue)
	var resp []CancelCustodialOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/sapi/v1/custodian/cancelOrdersBySymbol", params, cancelRate, &resp)
}

// CancelCustodialOneCancelsTheOtherOrder cancels an entire custodial order list
func (e *Exchange) CancelCustodialOneCancelsTheOtherOrder(ctx context.Context, req *CancelCustodialOneCancelsTheOtherOrderRequest) (*CustodialOneCancelsTheOtherOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	symbol, err := e.formatSymbol(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.OrderListID == nil {
		return nil, fmt.Errorf("%w: orderListId is required", order.ErrOrderIDNotSet)
	}
	params.Set("symbol", symbol)
	params.Set("orderListId", strconv.FormatUint(*req.OrderListID, 10))
	if req.ListClientOrderID != "" {
		params.Set("listClientOrderId", req.ListClientOrderID)
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	var resp *CustodialOneCancelsTheOtherOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/sapi/v1/custodian/cancelOcoOrder", params, cancelRate, &resp)
}

// GetCustodialSettlementSettings returns the settlement settings with rail, the custodial partner
func (e *Exchange) GetCustodialSettlementSettings(ctx context.Context, rail string) (*CustodialSettlementSettingsResponse, error) {
	params, err := custodialParams(rail)
	if err != nil {
		return nil, err
	}
	var resp *CustodialSettlementSettingsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/custodian/settlementSetting", params, defaultRate, &resp)
}

// GetCustodialSettlementHistory returns the settlements to the custodial partner, a page at a time
func (e *Exchange) GetCustodialSettlementHistory(ctx context.Context, req *CustodialSettlementHistoryRequest) (*CustodialSettlementHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Page > 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	var resp *CustodialSettlementHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/custodian/settlementHistory", params, defaultRate, &resp)
}

// custodialParams returns the parameters of a custodial request holding its rail, the custodial partner every custodial
// endpoint requires
func custodialParams(rail string) (url.Values, error) {
	if rail == "" {
		return nil, errRailRequired
	}
	params := url.Values{}
	params.Set("rail", rail)
	return params, nil
}

// custodialAssetTransferParams validates a transfer to the custodial sub-account and builds its parameters, which a
// transfer from the exchange wallet and one from the custodial partner share
func custodialAssetTransferParams(req *CustodialAssetTransferRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := custodialParams(req.Rail)
	if err != nil {
		return nil, err
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params.Set("asset", req.Asset.Upper().String())
	params.Set("amount", formatFloat(req.Amount))
	if req.ClientOrderID != "" {
		params.Set("clientOrderId", req.ClientOrderID)
	}
	return params, nil
}
