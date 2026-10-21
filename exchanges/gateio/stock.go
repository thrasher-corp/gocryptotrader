package gateio

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var errStockReferenceIDRequired = errors.New("reference ID required")

// GetStockUserAssets retrieves the stock account's assets
func (e *Exchange) GetStockUserAssets(ctx context.Context, arg *GetStockUserAssetsRequest) (*StockUserAssetsResponse, error) {
	params := url.Values{}
	if arg != nil {
		setStockPNLParams(params, arg.PNLCalculationCostType, arg.PNLCalculationPriceType)
	}
	var resp tradFiResponse[*StockUserAssetsResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockUserAssetsEPL, http.MethodGet, "stock/users/assets", params, nil, &resp)
}

// GetStockSymbols retrieves stock symbols
func (e *Exchange) GetStockSymbols(ctx context.Context, arg *GetStockSymbolsRequest) (*StockSymbolsResponse, error) {
	params := url.Values{}
	if arg != nil {
		if len(arg.Symbols) != 0 {
			params.Set("symbols", strings.Join(arg.Symbols, ","))
		}
		if arg.Exchange != "" {
			params.Set("exchange", arg.Exchange)
		}
		if arg.WithMultilingualDescriptions {
			params.Set("with_desc_i18n", "true")
		}
		setTradFiPageParams(params, arg.Page, arg.PageSize)
	}
	var resp tradFiResponse[*StockSymbolsResponse]
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, stockSymbolsEPL, common.EncodeURLValues("stock/symbols", params), &resp)
}

// GetStockSymbolDetails retrieves the trading rules of stock symbols
func (e *Exchange) GetStockSymbolDetails(ctx context.Context, arg *GetStockSymbolDetailsRequest) (*StockSymbolDetailsResponse, error) {
	params := url.Values{}
	if arg != nil {
		if len(arg.Symbols) != 0 {
			params.Set("symbols", strings.Join(arg.Symbols, ","))
		}
		if arg.Exchange != "" {
			params.Set("exchange", arg.Exchange)
		}
		setTradFiPageParams(params, arg.Page, arg.PageSize)
	}
	var resp tradFiResponse[*StockSymbolDetailsResponse]
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, stockSymbolDetailsEPL, common.EncodeURLValues("stock/symbols/detail", params), &resp)
}

// GetStockOrderbook retrieves a stock symbol's orderbook
func (e *Exchange) GetStockOrderbook(ctx context.Context, symbol string) (*StockOrderbookResponse, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w: stock symbol required", currency.ErrSymbolStringEmpty)
	}
	var resp tradFiResponse[*StockOrderbookResponse]
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, stockOrderbookEPL, "stock/market/"+symbol+"/orderbook", &resp)
}

// CreateStockOrder places a stock order
func (e *Exchange) CreateStockOrder(ctx context.Context, arg *StockOrderRequest) (*StockCreateOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, fmt.Errorf("%w: stock symbol required", currency.ErrSymbolStringEmpty)
	}
	if _, err := tradFiSideCode(arg.Side); err != nil {
		return nil, err
	}
	switch arg.PriceType {
	case order.Market:
	case order.Limit:
		if arg.Price <= 0 {
			return nil, order.ErrPriceMustBeSetIfLimitOrder
		}
	default:
		return nil, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, arg.PriceType)
	}
	if arg.Volume <= 0 {
		return nil, fmt.Errorf("%w: stock order volume required", limits.ErrAmountBelowMin)
	}
	if arg.TimeInForce != order.UnknownTIF && arg.TimeInForce != order.GoodTillDay {
		return nil, fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, arg.TimeInForce)
	}
	var resp tradFiResponse[*StockCreateOrderResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockCreateOrderEPL, http.MethodPost, "stock/orders", nil, arg, &resp)
}

// GetStockOpenOrders retrieves open stock orders, optionally for a single symbol
func (e *Exchange) GetStockOpenOrders(ctx context.Context, symbol string) ([]*StockOpenOrderResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp tradFiResponse[tradFiList[*StockOpenOrderResponse]]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockOpenOrdersEPL, http.MethodGet, "stock/orders", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// CancelAllStockOrders cancels every open stock order
func (e *Exchange) CancelAllStockOrders(ctx context.Context) error {
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockCancelAllOrdersEPL, http.MethodDelete, "stock/orders", nil, nil, &resp)
}

// GetStockOrderHistory retrieves the stock order history
func (e *Exchange) GetStockOrderHistory(ctx context.Context, arg *GetStockOrderHistoryRequest) (*StockOrderHistoryResponse, error) {
	params := url.Values{}
	if arg != nil {
		if arg.Symbol != "" {
			params.Set("symbol", arg.Symbol)
		}
		if len(arg.OrderIDs) != 0 {
			params.Set("order_ids", strings.Join(arg.OrderIDs, ","))
		}
		if err := setTradFiTimeRangeParams(params, arg.BeginTime, arg.EndTime); err != nil {
			return nil, err
		}
		if arg.Side != order.UnknownSide {
			side, err := tradFiSideCode(arg.Side)
			if err != nil {
				return nil, err
			}
			params.Set("side", strconv.FormatUint(uint64(side), 10))
		}
		setTradFiPageParams(params, arg.Page, arg.PageSize)
	}
	var resp tradFiResponse[*StockOrderHistoryResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockOrderHistoryEPL, http.MethodGet, "stock/orders/history", params, nil, &resp)
}

// UpdateStockOrder modifies an open stock order's volume and price
func (e *Exchange) UpdateStockOrder(ctx context.Context, orderID string, arg *StockOrderUpdateRequest) (*StockOrderIDResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Volume <= 0 {
		return nil, fmt.Errorf("%w: stock order volume required", limits.ErrAmountBelowMin)
	}
	if arg.Price <= 0 {
		return nil, fmt.Errorf("%w: stock order price required", limits.ErrPriceBelowMin)
	}
	var resp tradFiResponse[*StockOrderIDResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockUpdateOrderEPL, http.MethodPut, "stock/orders/"+orderID, nil, arg, &resp)
}

// CancelStockOrder cancels an open stock order
func (e *Exchange) CancelStockOrder(ctx context.Context, orderID string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockCancelOrderEPL, http.MethodDelete, "stock/orders/"+orderID, nil, nil, &resp)
}

// GetStockPositions retrieves the stock positions
func (e *Exchange) GetStockPositions(ctx context.Context, arg *GetStockPositionsRequest) ([]*StockPositionResponse, error) {
	params := url.Values{}
	if arg != nil {
		setStockPNLParams(params, arg.PNLCalculationCostType, arg.PNLCalculationPriceType)
		if arg.Symbol != "" {
			params.Set("symbol", arg.Symbol)
		}
		if arg.Exchange != "" {
			params.Set("exchange", arg.Exchange)
		}
	}
	var resp tradFiResponse[tradFiList[*StockPositionResponse]]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockPositionsEPL, http.MethodGet, "stock/positions", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// CloseStockPosition fully or partially closes a stock position
func (e *Exchange) CloseStockPosition(ctx context.Context, arg *StockClosePositionRequest) (*StockOrderIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, fmt.Errorf("%w: stock symbol required", currency.ErrSymbolStringEmpty)
	}
	if err := validateTradFiCloseType(arg.CloseType, arg.CloseVolume); err != nil {
		return nil, err
	}
	var resp tradFiResponse[*StockOrderIDResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockClosePositionEPL, http.MethodPost, "stock/positions/close", nil, arg, &resp)
}

// CreateStockTransaction transfers USDT between the spot account and the stock account
func (e *Exchange) CreateStockTransaction(ctx context.Context, arg *StockTransactionRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Asset.IsEmpty() {
		return fmt.Errorf("%w: stock transfer asset required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Change <= 0 {
		return fmt.Errorf("%w: stock transfer amount required", limits.ErrAmountBelowMin)
	}
	if arg.Type == "" {
		return errTradFiTransactionTypeRequired
	}
	if arg.ReferenceID == "" {
		return errStockReferenceIDRequired
	}
	var resp tradFiResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockCreateTransactionEPL, http.MethodPost, "stock/transactions", nil, arg, &resp)
}

// GetStockTransactions retrieves stock account transactions, including fund transfers, trades, fees, dividends, rewards
// and stock transfers
func (e *Exchange) GetStockTransactions(ctx context.Context, arg *GetStockTransactionsRequest) (*StockTransactionsResponse, error) {
	params := url.Values{}
	if arg != nil {
		if err := setTradFiTimeRangeParams(params, arg.BeginTime, arg.EndTime); err != nil {
			return nil, err
		}
		if arg.ReferenceID != "" {
			params.Set("ref_id", arg.ReferenceID)
		}
		if arg.Type != "" {
			params.Set("type", arg.Type)
		}
		setTradFiPageParams(params, arg.Page, arg.PageSize)
	}
	var resp tradFiResponse[*StockTransactionsResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, stockTransactionsEPL, http.MethodGet, "stock/transactions", params, nil, &resp)
}

// GetStockExchanges retrieves the supported stock markets and whether each supports stock transfers
func (e *Exchange) GetStockExchanges(ctx context.Context) ([]*StockExchangeResponse, error) {
	var resp tradFiResponse[tradFiList[*StockExchangeResponse]]
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, stockExchangesEPL, "stock/exchanges", &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetStockFeeRates retrieves the Japanese and Korean stock fee rates for each VIP level
func (e *Exchange) GetStockFeeRates(ctx context.Context) ([]*StockFeeRateResponse, error) {
	var resp tradFiResponse[tradFiList[*StockFeeRateResponse]]
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, stockFeeRateEPL, "stock/fee-rate", &resp); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// setStockPNLParams sets the profit and loss calculation query parameters of the Stock assets and positions endpoints
func setStockPNLParams(params url.Values, costType, priceType uint64) {
	if costType > 0 {
		params.Set("pnl_calc_type", strconv.FormatUint(costType, 10))
	}
	if priceType > 0 {
		params.Set("pnl_calc_price", strconv.FormatUint(priceType, 10))
	}
}
