package gateio

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errTradFiTransactionTypeRequired  = errors.New("transaction type required")
	errInvalidTradFiCloseType         = errors.New("close type must be 1 (partial close) or 2 (full close)")
	errTradFiPositionIDRequired       = errors.New("position ID required")
	errTradFiOrderLogIDRequired       = errors.New("order log ID required")
	errTradFiSymbolOrCategoryRequired = errors.New("at least one symbol or category code required")
	errInvalidTradFiPositionFilter    = errors.New("position direction must be long or short")
)

// GetTradFiMT5Account retrieves the MT5 account information for the authenticated user
func (e *Exchange) GetTradFiMT5Account(ctx context.Context) (*TradFiMT5Account, error) {
	var resp tradFiResponse[*TradFiMT5Account]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiUsersMT5AccountEPL, http.MethodGet, "tradfi/users/mt5-account", nil, nil, &resp)
}

// GetTradFiSymbolCategories retrieves the trading symbol categories
func (e *Exchange) GetTradFiSymbolCategories(ctx context.Context) ([]*TradFiCategory, error) {
	var resp tradFiResponse[tradFiList[*TradFiCategory]]
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, tradfiSymbolsCategoriesEPL, "tradfi/symbols/categories", &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetTradFiSymbolCommissions retrieves the commission charged per lot for the given symbols, categories or both; when
// both are supplied the symbols are filtered by the categories
func (e *Exchange) GetTradFiSymbolCommissions(ctx context.Context, symbols, categoryCodes []string) ([]*TradFiSymbolCommission, error) {
	if len(symbols) == 0 && len(categoryCodes) == 0 {
		return nil, errTradFiSymbolOrCategoryRequired
	}
	params := url.Values{}
	if len(symbols) != 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	if len(categoryCodes) != 0 {
		params.Set("category_code", strings.Join(categoryCodes, ","))
	}
	var resp tradFiResponse[tradFiList[*TradFiSymbolCommission]]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiSymbolsCommissionsEPL, http.MethodGet, "tradfi/symbols/commissions", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetTradFiSymbols retrieves every trading symbol
func (e *Exchange) GetTradFiSymbols(ctx context.Context) ([]*TradFiSymbol, error) {
	var resp tradFiResponse[tradFiList[*TradFiSymbol]]
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, tradfiSymbolsEPL, "tradfi/symbols", &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetTradFiSymbolDetails retrieves detailed contract information for up to 10 symbols
func (e *Exchange) GetTradFiSymbolDetails(ctx context.Context, symbols []string) ([]*TradFiSymbolDetail, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("%w: at least one tradfi symbol required", currency.ErrSymbolStringEmpty)
	}
	params := url.Values{}
	params.Set("symbols", strings.Join(symbols, ","))
	var resp tradFiResponse[tradFiList[*TradFiSymbolDetail]]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiSymbolsDetailEPL, http.MethodGet, "tradfi/symbols/detail", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetTradFiKlines retrieves candlesticks for a trading symbol
func (e *Exchange) GetTradFiKlines(ctx context.Context, symbol string, arg *GetTradFiKlinesRequest) ([]*TradFiKline, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w: tradfi symbol required", currency.ErrSymbolStringEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	klineType, err := klineIntervalToTypeString(arg.Interval)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("kline_type", klineType)
	if err := setTradFiTimeRangeParams(params, arg.BeginTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp tradFiResponse[tradFiList[*TradFiKline]]
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, tradfiKlinesEPL, common.EncodeURLValues("tradfi/symbols/"+symbol+"/klines", params), &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetTradFiSymbolTicker retrieves the latest ticker for a trading symbol
func (e *Exchange) GetTradFiSymbolTicker(ctx context.Context, symbol string) (*TradFiTicker, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w: tradfi symbol required", currency.ErrSymbolStringEmpty)
	}
	var resp tradFiResponse[*TradFiTicker]
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, tradfiTickersEPL, "tradfi/symbols/"+symbol+"/tickers", &resp)
}

// ActivateTradFiUser opens the TradFi service for the authenticated user
func (e *Exchange) ActivateTradFiUser(ctx context.Context) (*TradFiUserInfo, error) {
	var resp tradFiResponse[*TradFiUserInfo]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiUsersEPL, http.MethodPost, "tradfi/users", nil, nil, &resp)
}

// GetTradFiUserAssets retrieves the account balance and margin information
func (e *Exchange) GetTradFiUserAssets(ctx context.Context) (*TradFiUserAssets, error) {
	var resp tradFiResponse[*TradFiUserAssets]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiUsersAssetsEPL, http.MethodGet, "tradfi/users/assets", nil, nil, &resp)
}

// CreateTradFiTransaction transfers funds into or out of the TradFi account
func (e *Exchange) CreateTradFiTransaction(ctx context.Context, arg *TradFiTransactionRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Asset.IsEmpty() {
		return fmt.Errorf("%w: tradfi asset required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Change <= 0 {
		return fmt.Errorf("%w: tradfi change amount required", limits.ErrAmountBelowMin)
	}
	if arg.Type == "" {
		return errTradFiTransactionTypeRequired
	}
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiCreateTransactionsEPL, http.MethodPost, "tradfi/transactions", nil, arg, &resp)
}

// GetTradFiTransactions retrieves fund transfer in and out records
func (e *Exchange) GetTradFiTransactions(ctx context.Context, arg *GetTradFiTransactionsRequest) (*TradFiTransactionListData, error) {
	params := url.Values{}
	if arg != nil {
		if err := setTradFiTimeRangeParams(params, arg.BeginTime, arg.EndTime); err != nil {
			return nil, err
		}
		if arg.Type != "" {
			params.Set("type", arg.Type)
		}
		setTradFiPageParams(params, arg.Page, arg.PageSize)
	}
	var resp tradFiResponse[*TradFiTransactionListData]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiGetTransactionsEPL, http.MethodGet, "tradfi/transactions", params, nil, &resp)
}

// CreateTradFiOrder places an order. The returned ID is a queue task ID, which GetTradFiOrderLog resolves to the order.
func (e *Exchange) CreateTradFiOrder(ctx context.Context, arg *TradFiOrderRequest) (*TradFiCreateOrderResult, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, fmt.Errorf("%w: tradfi symbol required", currency.ErrSymbolStringEmpty)
	}
	if _, err := tradFiSideCode(arg.Side); err != nil {
		return nil, err
	}
	if arg.PriceType != order.Market && arg.PriceType != order.Trigger {
		return nil, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, arg.PriceType)
	}
	if arg.Price <= 0 {
		return nil, fmt.Errorf("%w: tradfi order price required", limits.ErrPriceBelowMin)
	}
	if arg.Volume <= 0 {
		return nil, fmt.Errorf("%w: tradfi order volume required", limits.ErrAmountBelowMin)
	}
	var resp tradFiResponse[*TradFiCreateOrderResult]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiCreateOrdersEPL, http.MethodPost, "tradfi/orders", nil, arg, &resp)
}

// GetTradFiActiveOrders retrieves the active orders
func (e *Exchange) GetTradFiActiveOrders(ctx context.Context) ([]*TradFiOrder, error) {
	var resp tradFiResponse[tradFiList[*TradFiOrder]]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiGetOrdersEPL, http.MethodGet, "tradfi/orders", nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// UpdateTradFiOrder modifies an active order's price, take profit price and stop loss price
func (e *Exchange) UpdateTradFiOrder(ctx context.Context, orderID uint64, arg *TradFiOrderUpdateRequest) (*TradFiUpdatedOrder, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Price <= 0 {
		return nil, fmt.Errorf("%w: tradfi order price required", limits.ErrPriceBelowMin)
	}
	var resp tradFiResponse[*TradFiUpdatedOrder]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiUpdateOrdersEPL, http.MethodPut, "tradfi/orders/"+strconv.FormatUint(orderID, 10), nil, arg, &resp)
}

// CancelTradFiOrder cancels an active order
func (e *Exchange) CancelTradFiOrder(ctx context.Context, orderID uint64) error {
	if orderID == 0 {
		return order.ErrOrderIDNotSet
	}
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiDeleteOrdersEPL, http.MethodDelete, "tradfi/orders/"+strconv.FormatUint(orderID, 10), nil, nil, &resp)
}

// GetTradFiOrderHistory retrieves completed orders
func (e *Exchange) GetTradFiOrderHistory(ctx context.Context, arg *GetTradFiOrderHistoryRequest) ([]*TradFiHistoricalOrder, error) {
	params := url.Values{}
	if arg != nil {
		if err := setTradFiTimeRangeParams(params, arg.BeginTime, arg.EndTime); err != nil {
			return nil, err
		}
		if arg.Symbol != "" {
			params.Set("symbol", arg.Symbol)
		}
		if arg.Side != order.UnknownSide {
			side, err := tradFiSideCode(arg.Side)
			if err != nil {
				return nil, err
			}
			params.Set("side", strconv.FormatUint(uint64(side), 10))
		}
	}
	var resp tradFiResponse[tradFiList[*TradFiHistoricalOrder]]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiOrdersHistoryEPL, http.MethodGet, "tradfi/orders/history", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetTradFiOrderLog retrieves the order recorded against the log ID that CreateTradFiOrder returns
func (e *Exchange) GetTradFiOrderLog(ctx context.Context, logID uint64) (*TradFiOrderLogResponse, error) {
	if logID == 0 {
		return nil, errTradFiOrderLogIDRequired
	}
	var resp tradFiResponse[*TradFiOrderLogResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiOrderLogEPL, http.MethodGet, "tradfi/orders/log/"+strconv.FormatUint(logID, 10), nil, nil, &resp)
}

// GetTradFiActivePositions retrieves the open positions
func (e *Exchange) GetTradFiActivePositions(ctx context.Context) ([]*TradFiPosition, error) {
	var resp tradFiResponse[tradFiList[*TradFiPosition]]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiGetPositionsEPL, http.MethodGet, "tradfi/positions", nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// UpdateTradFiPosition modifies the take profit and stop loss prices of an open position
func (e *Exchange) UpdateTradFiPosition(ctx context.Context, positionID uint64, arg *TradFiPositionUpdateRequest) error {
	if positionID == 0 {
		return errTradFiPositionIDRequired
	}
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiUpdatePositionsEPL, http.MethodPut, "tradfi/positions/"+strconv.FormatUint(positionID, 10), nil, arg, &resp)
}

// CloseTradFiPosition fully or partially closes an open position
func (e *Exchange) CloseTradFiPosition(ctx context.Context, positionID uint64, arg *TradFiClosePositionRequest) error {
	if positionID == 0 {
		return errTradFiPositionIDRequired
	}
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if err := validateTradFiCloseType(arg.CloseType, arg.CloseVolume); err != nil {
		return err
	}
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiCreatePositionsEPL, http.MethodPost, "tradfi/positions/"+strconv.FormatUint(positionID, 10)+"/close", nil, arg, &resp)
}

// GetTradFiPositionHistory retrieves closed positions
func (e *Exchange) GetTradFiPositionHistory(ctx context.Context, arg *GetTradFiPositionHistoryRequest) (*TradFiHistoricalPositionListData, error) {
	params := url.Values{}
	if arg != nil {
		if err := setTradFiTimeRangeParams(params, arg.BeginTime, arg.EndTime); err != nil {
			return nil, err
		}
		setTradFiPageParams(params, arg.Page, arg.PageSize)
		if arg.Symbol != "" {
			params.Set("symbol", arg.Symbol)
		}
		switch arg.PositionDirection {
		case order.UnknownSide:
		case order.Long, order.Short:
			params.Set("position_dir", arg.PositionDirection.Title())
		default:
			return nil, fmt.Errorf("%w: %s", errInvalidTradFiPositionFilter, arg.PositionDirection)
		}
	}
	var resp tradFiResponse[*TradFiHistoricalPositionListData]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, tradfiPositionsHistoryEPL, http.MethodGet, "tradfi/positions/history", params, nil, &resp)
}

// tradFiSideCode converts an order side to the numeric side code of the TradFi and Stock APIs
func tradFiSideCode(side order.Side) (uint8, error) {
	switch side {
	case order.Sell:
		return 1, nil
	case order.Buy:
		return 2, nil
	default:
		return 0, fmt.Errorf("%w: %s", order.ErrSideIsInvalid, side)
	}
}

// validateTradFiCloseType checks a TradFi or Stock close position request, where only a partial close carries a volume
func validateTradFiCloseType(closeType uint64, closeVolume types.Number) error {
	switch closeType {
	case 1:
		if closeVolume <= 0 {
			return fmt.Errorf("%w: partial close volume required", limits.ErrAmountBelowMin)
		}
		return nil
	case 2:
		return nil
	default:
		return fmt.Errorf("%w: %d", errInvalidTradFiCloseType, closeType)
	}
}

// setTradFiTimeRangeParams sets the begin_time and end_time query parameters, in seconds, of the TradFi and Stock APIs
func setTradFiTimeRangeParams(params url.Values, begin, end time.Time) error {
	if !begin.IsZero() && !end.IsZero() {
		if err := common.StartEndTimeCheck(begin, end); err != nil {
			return err
		}
	}
	if !begin.IsZero() {
		params.Set("begin_time", strconv.FormatInt(begin.Unix(), 10))
	}
	if !end.IsZero() {
		params.Set("end_time", strconv.FormatInt(end.Unix(), 10))
	}
	return nil
}

// setTradFiPageParams sets the page and page_size query parameters of the TradFi and Stock APIs
func setTradFiPageParams(params url.Values, page, pageSize uint64) {
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatUint(pageSize, 10))
	}
}
