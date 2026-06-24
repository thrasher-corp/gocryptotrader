package gateio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errTradFiTypeRequired      = errors.New("tradfi transaction type required")
	errTradFiCloseTypeRequired = errors.New("tradfi close type required (1=full close, 2=partial close)")
)

// GetTradFiMT5Account retrieves the MT5 account information for the authenticated user.
func (e *Exchange) GetTradFiMT5Account(ctx context.Context) (*TradFiMT5Account, error) {
	var resp tradFiResponse[*TradFiMT5Account]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiUsersMT5AccountEPL, http.MethodGet, "tradfi/users/mt5-account", nil, nil, &resp)
}

// GetTradFiSymbolCategories retrieves the list of trading symbol categories.
func (e *Exchange) GetTradFiSymbolCategories(ctx context.Context) ([]*TradFiCategory, error) {
	var resp tradFiResponse[*TradFiCategoryList]
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, tradfiSymbolsCategoriesEPL, "tradfi/symbols/categories", &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data.List, nil
}

// GetTradFiSymbols retrieves the full list of tradable symbols.
func (e *Exchange) GetTradFiSymbols(ctx context.Context) ([]*TradFiSymbol, error) {
	var resp tradFiResponse[*TradFiSymbolList]
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, tradfiSymbolsEPL, "tradfi/symbols", &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data.List, nil
}

// GetTradFiSymbolDetails retrieves detailed contract information for one or more symbols.
func (e *Exchange) GetTradFiSymbolDetails(ctx context.Context, symbols currency.Pairs) ([]*TradFiSymbolDetail, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("%w: at least one tradfi symbol required", currency.ErrCurrencyPairsEmpty)
	}
	params := url.Values{}
	params.Set("symbols", symbols.Join())
	var resp tradFiResponse[*TradFiSymbolDetailList]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiSymbolsDetailEPL, http.MethodGet, "tradfi/symbols/detail", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data.List, nil
}

// GetTradFiKlines retrieves candlestick data for a trading symbol.
func (e *Exchange) GetTradFiKlines(ctx context.Context, symbol currency.Pair, arg *GetTradFiKlinesRequest) ([]*TradFiKline, error) {
	if symbol.IsEmpty() {
		return nil, fmt.Errorf("%w: tradfi symbol required", currency.ErrSymbolStringEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.KlineType == "" {
		return nil, fmt.Errorf("%w: tradfi kline type required", kline.ErrInvalidInterval)
	}
	params := url.Values{}
	params.Set("kline_type", arg.KlineType)
	if !arg.BeginTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.BeginTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	if !arg.BeginTime.IsZero() {
		params.Set("begin_time", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp tradFiResponse[*TradFiKlineList]
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, tradfiKlinesEPL, common.EncodeURLValues("tradfi/symbols/"+symbol.String()+"/klines", params), &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data.List, nil
}

// GetTradFiSymbolTicker retrieves the latest ticker for a trading symbol.
func (e *Exchange) GetTradFiSymbolTicker(ctx context.Context, symbol string) (*TradFiTicker, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w: tradfi symbol required", currency.ErrSymbolStringEmpty)
	}
	var resp tradFiResponse[*TradFiTicker]
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, tradfiTickersEPL, "tradfi/symbols/"+symbol+"/tickers", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// ActivateTradFiUser activates the TradFi service for the authenticated user.
func (e *Exchange) ActivateTradFiUser(ctx context.Context) (*TradFiUserInfo, error) {
	var resp tradFiResponse[*TradFiUserInfo]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiUsersEPL, http.MethodPost, "tradfi/users", nil, nil, &resp)
}

// GetTradFiUserAssets retrieves the account balance and margin information.
func (e *Exchange) GetTradFiUserAssets(ctx context.Context) (*TradFiUserAssets, error) {
	var resp tradFiResponse[*TradFiUserAssets]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiUsersAssetsEPL, http.MethodGet, "tradfi/users/assets", nil, nil, &resp)
}

// GetTradFiTransactions retrieves fund transfer in/out records.
func (e *Exchange) GetTradFiTransactions(ctx context.Context, arg *GetTradFiTransactionsRequest) (*TradFiTransactionListData, error) {
	params := url.Values{}
	if arg != nil {
		if !arg.BeginTime.IsZero() && !arg.EndTime.IsZero() {
			if err := common.StartEndTimeCheck(arg.BeginTime, arg.EndTime); err != nil {
				return nil, err
			}
		}
		if !arg.BeginTime.IsZero() {
			params.Set("begin_time", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
		}
		if !arg.EndTime.IsZero() {
			params.Set("end_time", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
		}
		if arg.Type != "" {
			params.Set("type", arg.Type)
		}
		if arg.Page > 0 {
			params.Set("page", strconv.FormatUint(arg.Page, 10))
		}
		if arg.PageSize > 0 {
			params.Set("page_size", strconv.FormatUint(arg.PageSize, 10))
		}
	}
	var resp tradFiResponse[*TradFiTransactionListData]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiGetTransactionsEPL, http.MethodGet, "tradfi/transactions", params, nil, &resp)
}

// CreateTradFiTransaction deposits or withdraws funds to/from the TradFi account.
func (e *Exchange) CreateTradFiTransaction(ctx context.Context, arg *TradFiTransactionRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Asset.IsEmpty() {
		return fmt.Errorf("%w; tradfi asset required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Change <= 0 {
		return fmt.Errorf("%w: tradfi change amount required", limits.ErrAmountBelowMin)
	}
	if arg.Type == "" {
		return errTradFiTypeRequired
	}
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiCreateTransactionsEPL, http.MethodPost, "tradfi/transactions", nil, arg, &resp)
}

// GetTradFiActiveOrders retrieves the list of active (pending) orders.
func (e *Exchange) GetTradFiActiveOrders(ctx context.Context) ([]*TradFiOrder, error) {
	var resp tradFiResponse[*TradFiOrderList]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiGetOrdersEPL, http.MethodGet, "tradfi/orders", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data.List, nil
}

// CreateTradFiOrder places a new order.
func (e *Exchange) CreateTradFiOrder(ctx context.Context, arg *TradFiOrderRequest) (*TradFiCreateOrderResult, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, fmt.Errorf("%w: tradfi symbol required", currency.ErrSymbolStringEmpty)
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if arg.Volume <= 0 {
		return nil, fmt.Errorf("%w: tradfi order volume required", limits.ErrAmountBelowMin)
	}
	if arg.Side != 1 && arg.Side != 2 {
		return nil, fmt.Errorf("%w; order side required (1=sell, 2=buy)", order.ErrSideIsInvalid)
	}
	var resp tradFiResponse[*TradFiCreateOrderResult]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiCreateOrdersEPL, http.MethodPost, "tradfi/orders", nil, arg, &resp)
}

// UpdateTradFiOrder modifies an existing pending order's price, take profit, or stop loss.
func (e *Exchange) UpdateTradFiOrder(ctx context.Context, orderID int64, arg *TradFiOrderUpdateRequest) (*TradFiUpdatedOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Price == "" {
		return nil, fmt.Errorf("%w: tradfi order volume required", limits.ErrPriceBelowMin)
	}
	var resp tradFiResponse[*TradFiUpdatedOrder]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiUpdateOrdersEPL, http.MethodPut, "tradfi/orders/"+strconv.FormatInt(orderID, 10), nil, arg, &resp)
}

// CancelTradFiOrder cancels a pending order by its order ID.
func (e *Exchange) CancelTradFiOrder(ctx context.Context, orderID int64) error {
	if orderID == 0 {
		return order.ErrOrderIDNotSet
	}
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiDeleteOrdersEPL, http.MethodDelete, "tradfi/orders/"+strconv.FormatInt(orderID, 10), nil, nil, &resp)
}

// GetTradFiOrderHistory retrieves historical (completed) orders.
func (e *Exchange) GetTradFiOrderHistory(ctx context.Context, arg *GetTradFiOrderHistoryRequest) ([]*TradFiHistoricalOrder, error) {
	params := url.Values{}
	if arg != nil {
		if !arg.BeginTime.IsZero() && !arg.EndTime.IsZero() {
			if err := common.StartEndTimeCheck(arg.BeginTime, arg.EndTime); err != nil {
				return nil, err
			}
		}
		if !arg.BeginTime.IsZero() {
			params.Set("begin_time", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
		}
		if !arg.EndTime.IsZero() {
			params.Set("end_time", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
		}
		if !arg.Symbol.IsEmpty() {
			params.Set("symbol", arg.Symbol.String())
		}
		if arg.Side > 0 {
			params.Set("side", strconv.FormatUint(arg.Side, 10))
		}
	}
	var resp tradFiResponse[*TradFiOrderHistoryList]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiOrdersHistoryEPL, http.MethodGet, "tradfi/orders/history", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data.List, nil
}

// GetTradFiActivePositions retrieves the list of currently open positions.
func (e *Exchange) GetTradFiActivePositions(ctx context.Context) ([]*TradFiPosition, error) {
	var resp tradFiResponse[*TradFiPositionList]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiGetPositionsEPL, http.MethodGet, "tradfi/positions", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResults
	}
	return resp.Data.List, nil
}

// UpdateTradFiPosition modifies the take profit or stop loss of an open position.
func (e *Exchange) UpdateTradFiPosition(ctx context.Context, positionID int64, arg *TradFiPositionUpdateRequest) error {
	if positionID == 0 {
		return order.ErrOrderIDNotSet
	}
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiUpdatePositionsEPL, http.MethodPut, "tradfi/positions/"+strconv.FormatInt(positionID, 10), nil, arg, &resp)
}

// CloseTradFiPosition fully or partially closes an open position.
func (e *Exchange) CloseTradFiPosition(ctx context.Context, positionID int64, arg *TradFiClosePositionRequest) error {
	if positionID == 0 {
		return order.ErrOrderIDNotSet
	}
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.CloseType != 1 && arg.CloseType != 2 {
		return errTradFiCloseTypeRequired
	}
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiCreatePositionsEPL, http.MethodPost, "tradfi/positions/"+strconv.FormatInt(positionID, 10)+"/close", nil, arg, &resp)
}

// GetTradFiPositionHistory retrieves the history of closed positions.
func (e *Exchange) GetTradFiPositionHistory(ctx context.Context, arg *GetTradFiPositionHistoryRequest) (*TradFiHistoricalPositionListData, error) {
	params := url.Values{}
	if arg != nil {
		if !arg.BeginTime.IsZero() && !arg.EndTime.IsZero() {
			if err := common.StartEndTimeCheck(arg.BeginTime, arg.EndTime); err != nil {
				return nil, err
			}
		}
		if arg.Page > 0 {
			params.Set("page", strconv.FormatUint(arg.Page, 10))
		}
		if arg.PageSize > 0 {
			params.Set("page_size", strconv.FormatUint(arg.PageSize, 10))
		}
		if !arg.BeginTime.IsZero() {
			params.Set("begin_time", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
		}
		if !arg.EndTime.IsZero() {
			params.Set("end_time", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
		}
		if !arg.Symbol.IsEmpty() {
			params.Set("symbol", arg.Symbol.String())
		}
		if arg.PositionDir != "" {
			params.Set("position_dir", arg.PositionDir)
		}
	}
	var resp tradFiResponse[*TradFiHistoricalPositionListData]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiPositionsHistoryEPL, http.MethodGet, "tradfi/positions/history", params, nil, &resp)
}
