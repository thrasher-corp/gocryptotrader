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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errCrossexExchangeTypeRequired = errors.New("crossex exchange type required")
	errCrossexSymbolRequired       = errors.New("crossex symbol required")
	errCrossexFromAccountRequired  = errors.New("crossex from account required")
	errCrossexToAccountRequired    = errors.New("crossex to account required")
	errCrossexQuoteIDRequired      = errors.New("crossex quote ID required")
	errCrossexLeverageRequired     = errors.New("crossex leverage required")
)

// GetCrossexSymbols retrieves symbol information for CrossEx trading pairs.
// Optionally filter by one or more symbol names.
func (e *Exchange) GetCrossexSymbols(ctx context.Context, symbols []string) ([]*CrossexSymbol, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp []*CrossexSymbol
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("crossex/symbols", params), &resp)
}

// GetCrossexRiskLimits retrieves risk limit information for CrossEx trading pairs.
// Optionally filter by one or more symbol names.
func (e *Exchange) GetCrossexRiskLimits(ctx context.Context, symbols []string) ([]*CrossexRiskLimit, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp []*CrossexRiskLimit
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("crossex/risk_limit", params), &resp)
}

// GetCrossexTransferCoins retrieves the list of currencies supported for CrossEx transfers.
// Optionally filter by coin code.
func (e *Exchange) GetCrossexTransferCoins(ctx context.Context, coin currency.Code) ([]*CrossexTransferCoin, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	var resp []*CrossexTransferCoin
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/transfers/coin", params), nil, nil, &resp)
}

// GetCrossexTransferHistory retrieves the fund transfer history for the authenticated user.
func (e *Exchange) GetCrossexTransferHistory(ctx context.Context, arg *GetCrossexTransferHistoryRequest) ([]*CrossexTransferRecord, error) {
	params := url.Values{}
	if arg != nil {
		if !arg.Coin.IsEmpty() {
			params.Set("coin", arg.Coin.String())
		}
		if arg.OrderID != "" {
			params.Set("order_id", arg.OrderID)
		}
		if arg.ClientOrderID != "" {
			params.Set("client_order_id", arg.ClientOrderID)
		}
		if arg.To > 0 {
			params.Set("to", strconv.FormatInt(arg.To, 10))
		}
		if arg.From > 0 {
			params.Set("from", strconv.FormatInt(arg.From, 10))
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
	}
	var resp []*CrossexTransferRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/transfers", params), nil, nil, &resp)
}

// CrossexFundTransfer initiates a fund transfer to or from the CrossEx account.
func (e *Exchange) CrossexFundTransfer(ctx context.Context, arg *CrossexTransferRequest) (*CrossexTransferResponse, error) {
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: crossex amount required", order.ErrAmountMustBeSet)
	}
	if arg.From == "" {
		return nil, errCrossexFromAccountRequired
	}
	if arg.To == "" {
		return nil, errCrossexToAccountRequired
	}
	var resp CrossexTransferResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "crossex/transfers", nil, arg, &resp)
}

// CreateCrossexOrder places a new order on the CrossEx platform.
func (e *Exchange) CreateCrossexOrder(ctx context.Context, arg *CrossexOrderCreateRequest) (*CrossexOrder, error) {
	if arg.ExchangeType == "" {
		return nil, errCrossexExchangeTypeRequired
	}
	if arg.Symbol == "" {
		return nil, errCrossexSymbolRequired
	}
	if arg.Side == order.UnknownSide {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	var resp CrossexOrder
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "crossex/orders", nil, arg, &resp)
}

// GetCrossexOrderDetails retrieves details for a specific CrossEx order.
func (e *Exchange) GetCrossexOrderDetails(ctx context.Context, orderID string) (*CrossexOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp CrossexOrder
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "crossex/orders/"+orderID, nil, nil, &resp)
}

// ModifyCrossexOrder modifies an existing CrossEx order's quantity or price.
func (e *Exchange) ModifyCrossexOrder(ctx context.Context, orderID string, arg *CrossexOrderUpdateRequest) (*CrossexOrderActionResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp CrossexOrderActionResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPut, "crossex/orders/"+orderID, nil, arg, &resp)
}

// CancelCrossexOrder cancels an existing CrossEx order.
func (e *Exchange) CancelCrossexOrder(ctx context.Context, orderID string) (*CrossexOrderActionResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp CrossexOrderActionResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodDelete, "crossex/orders/"+orderID, nil, nil, &resp)
}

// GetCrossexConvertQuote retrieves a flash swap quote for a CrossEx currency conversion.
func (e *Exchange) GetCrossexConvertQuote(ctx context.Context, arg *CrossexConvertQuoteRequest) (*CrossexConvertQuoteResponse, error) {
	if arg.ExchangeType == "" {
		return nil, errCrossexExchangeTypeRequired
	}
	if arg.FromCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: crossex from coin required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.ToCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: crossex to coin required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.FromAmount <= 0 {
		return nil, fmt.Errorf("%w: crossex amount required", order.ErrAmountMustBeSet)
	}
	var resp CrossexConvertQuoteResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "crossex/convert/quote", nil, arg, &resp)
}

// ExecuteCrossexConvertOrder executes a CrossEx flash swap using a previously obtained quote ID.
func (e *Exchange) ExecuteCrossexConvertOrder(ctx context.Context, quoteID string) (*CrossexConvertOrderResponse, error) {
	if quoteID == "" {
		return nil, errCrossexQuoteIDRequired
	}
	body := CrossexConvertOrderRequest{QuoteID: quoteID}
	var resp CrossexConvertOrderResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "crossex/convert/orders", nil, &body, &resp)
}

// GetCrossexAccountAssets retrieves the CrossEx account asset information.
// Optionally filter by exchange type.
func (e *Exchange) GetCrossexAccountAssets(ctx context.Context, exchangeType string) ([]*CrossexAccount, error) {
	params := url.Values{}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossexAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/accounts", params), nil, nil, &resp)
}

// UpdateCrossexAccount modifies the CrossEx account's position mode or account mode.
func (e *Exchange) UpdateCrossexAccount(ctx context.Context, arg *CrossexAccountUpdateRequest) (*CrossexAccountUpdateResponse, error) {
	var resp CrossexAccountUpdateResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPut, "crossex/accounts", nil, arg, &resp)
}

// GetCrossexContractLeverage retrieves the leverage multiplier for CrossEx contract trading pairs.
// Optionally filter by one or more symbol names.
func (e *Exchange) GetCrossexContractLeverage(ctx context.Context, symbols []string) (map[string]string, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp map[string]string
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/positions/leverage", params), nil, nil, &resp)
}

// SetCrossexContractLeverage sets the leverage multiplier for a CrossEx contract trading pair.
func (e *Exchange) SetCrossexContractLeverage(ctx context.Context, arg *CrossexLeverageRequest) (*CrossexLeverageResponse, error) {
	if arg.Symbol == "" {
		return nil, errCrossexSymbolRequired
	}
	if arg.Leverage <= 0 {
		return nil, errCrossexLeverageRequired
	}
	var resp CrossexLeverageResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "crossex/positions/leverage", nil, arg, &resp)
}

// GetCrossexMarginLeverage retrieves the leverage multiplier for CrossEx leveraged (margin) trading pairs.
// Optionally filter by one or more symbol names.
func (e *Exchange) GetCrossexMarginLeverage(ctx context.Context, symbols []string) (map[string]string, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp map[string]string
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/margin_positions/leverage", params), nil, nil, &resp)
}

// SetCrossexMarginLeverage sets the leverage multiplier for a CrossEx leveraged (margin) trading pair.
func (e *Exchange) SetCrossexMarginLeverage(ctx context.Context, arg *CrossexLeverageRequest) (*CrossexLeverageResponse, error) {
	if arg.Symbol == "" {
		return nil, errCrossexSymbolRequired
	}
	if arg.Leverage <= 0 {
		return nil, errCrossexLeverageRequired
	}
	var resp CrossexLeverageResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "crossex/margin_positions/leverage", nil, arg, &resp)
}

// CloseCrossexPosition fully closes an open CrossEx contract position.
func (e *Exchange) CloseCrossexPosition(ctx context.Context, arg *CrossexClosePositionRequest) (*CrossexOrderActionResponse, error) {
	if arg.Symbol == "" {
		return nil, errCrossexSymbolRequired
	}
	var resp CrossexOrderActionResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "crossex/positions", nil, arg, &resp)
}

// GetCrossexInterestRates retrieves margin asset interest rates.
// Optionally filter by coin and exchange type.
func (e *Exchange) GetCrossexInterestRates(ctx context.Context, coin currency.Code, exchangeType string) ([]*CrossexInterestRate, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossexInterestRate
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/interest_rate", params), nil, nil, &resp)
}

// GetCrossexUserFeeRates retrieves the fee rates for the authenticated CrossEx user.
func (e *Exchange) GetCrossexUserFeeRates(ctx context.Context) ([]*CrossexFee, error) {
	var resp []*CrossexFee
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "crossex/fee", nil, nil, &resp)
}

// GetCrossexContractPositions retrieves the authenticated user's open CrossEx contract positions.
// Optionally filter by symbol and exchange type.
func (e *Exchange) GetCrossexContractPositions(ctx context.Context, symbol, exchangeType string) ([]*CrossexPosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossexPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/positions", params), nil, nil, &resp)
}

// GetCrossexMarginPositions retrieves the authenticated user's open CrossEx leveraged (margin) positions.
// Optionally filter by symbol and exchange type.
func (e *Exchange) GetCrossexMarginPositions(ctx context.Context, symbol, exchangeType string) ([]*CrossexMarginPosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossexMarginPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/margin_positions", params), nil, nil, &resp)
}

// GetCrossexADLRank retrieves the ADL (Auto-Deleveraging) position reduction ranking for a CrossEx symbol.
func (e *Exchange) GetCrossexADLRank(ctx context.Context, symbol string) ([]*CrossexAdlRank, error) {
	if symbol == "" {
		return nil, errCrossexSymbolRequired
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []*CrossexAdlRank
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/adl_rank", params), nil, nil, &resp)
}

// GetCrossexOpenOrders retrieves all currently open CrossEx orders.
func (e *Exchange) GetCrossexOpenOrders(ctx context.Context, arg *GetCrossexOpenOrdersRequest) ([]*CrossexOrder, error) {
	params := url.Values{}
	if arg != nil {
		if arg.Symbol != "" {
			params.Set("symbol", arg.Symbol)
		}
		if arg.ExchangeType != "" {
			params.Set("exchange_type", arg.ExchangeType)
		}
		if arg.BusinessType != "" {
			params.Set("business_type", arg.BusinessType)
		}
	}
	var resp []*CrossexOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/open_orders", params), nil, nil, &resp)
}

// GetCrossexOrderHistory retrieves the CrossEx order history for the authenticated user.
func (e *Exchange) GetCrossexOrderHistory(ctx context.Context, arg *GetCrossexOrderHistoryRequest) ([]*CrossexOrder, error) {
	params := url.Values{}
	if arg != nil {
		if arg.Page > 0 {
			params.Set("page", strconv.FormatUint(arg.Page, 10))
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Symbol != "" {
			params.Set("symbol", arg.Symbol)
		}
		if arg.From > 0 {
			params.Set("from", strconv.FormatInt(arg.From, 10))
		}
		if arg.To > 0 {
			params.Set("to", strconv.FormatInt(arg.To, 10))
		}
		if arg.Attribute != "" {
			params.Set("attribute", arg.Attribute)
		}
	}
	var resp []*CrossexOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/history_orders", params), nil, nil, &resp)
}

// GetCrossexContractPositionHistory retrieves closed CrossEx contract position history.
func (e *Exchange) GetCrossexContractPositionHistory(ctx context.Context, arg *GetCrossexPositionHistoryRequest) ([]*CrossexHistoricalPosition, error) {
	params := url.Values{}
	if arg != nil {
		if arg.Page > 0 {
			params.Set("page", strconv.FormatUint(arg.Page, 10))
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Symbol != "" {
			params.Set("symbol", arg.Symbol)
		}
		if arg.From > 0 {
			params.Set("from", strconv.FormatInt(arg.From, 10))
		}
		if arg.To > 0 {
			params.Set("to", strconv.FormatInt(arg.To, 10))
		}
	}
	var resp []*CrossexHistoricalPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/history_positions", params), nil, nil, &resp)
}

// GetCrossexMarginPositionHistory retrieves closed CrossEx leveraged (margin) position history.
func (e *Exchange) GetCrossexMarginPositionHistory(ctx context.Context, arg *GetCrossexPositionHistoryRequest) ([]*CrossexHistoricalMarginPosition, error) {
	params := url.Values{}
	if arg != nil {
		if arg.Page > 0 {
			params.Set("page", strconv.FormatUint(arg.Page, 10))
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Symbol != "" {
			params.Set("symbol", arg.Symbol)
		}
		if arg.From > 0 {
			params.Set("from", strconv.FormatInt(arg.From, 10))
		}
		if arg.To > 0 {
			params.Set("to", strconv.FormatInt(arg.To, 10))
		}
	}
	var resp []*CrossexHistoricalMarginPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/history_margin_positions", params), nil, nil, &resp)
}

// GetCrossexMarginInterestHistory retrieves the leveraged interest deduction history.
func (e *Exchange) GetCrossexMarginInterestHistory(ctx context.Context, arg *GetCrossexMarginInterestHistoryRequest) ([]*CrossexMarginInterestRecord, error) {
	params := url.Values{}
	if arg != nil {
		if arg.Symbol != "" {
			params.Set("symbol", arg.Symbol)
		}
		if arg.From > 0 {
			params.Set("from", strconv.FormatInt(arg.From, 10))
		}
		if arg.To > 0 {
			params.Set("to", strconv.FormatInt(arg.To, 10))
		}
		if arg.Page > 0 {
			params.Set("page", strconv.FormatUint(arg.Page, 10))
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.ExchangeType != "" {
			params.Set("exchange_type", arg.ExchangeType)
		}
	}
	var resp []*CrossexMarginInterestRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/history_margin_interests", params), nil, nil, &resp)
}

// GetCrossexTradeHistory retrieves the trade history for the authenticated CrossEx user.
func (e *Exchange) GetCrossexTradeHistory(ctx context.Context, arg *GetCrossexTradeHistoryRequest) ([]*CrossexTrade, error) {
	params := url.Values{}
	if arg != nil {
		if arg.Page > 0 {
			params.Set("page", strconv.FormatUint(arg.Page, 10))
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Symbol != "" {
			params.Set("symbol", arg.Symbol)
		}
		if arg.From > 0 {
			params.Set("from", strconv.FormatInt(arg.From, 10))
		}
		if arg.To > 0 {
			params.Set("to", strconv.FormatInt(arg.To, 10))
		}
	}
	var resp []*CrossexTrade
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/history_trades", params), nil, nil, &resp)
}

// GetCrossexAccountBook retrieves the account asset change history for the authenticated CrossEx user.
func (e *Exchange) GetCrossexAccountBook(ctx context.Context, arg *GetCrossexAccountBookRequest) ([]*CrossexAccountBookRecord, error) {
	params := url.Values{}
	if arg != nil {
		if arg.Page > 0 {
			params.Set("page", strconv.FormatUint(arg.Page, 10))
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if !arg.Coin.IsEmpty() {
			params.Set("coin", arg.Coin.String())
		}
		if arg.StatementType != "" {
			params.Set("statement_type", arg.StatementType)
		}
		if arg.From > 0 {
			params.Set("from", strconv.FormatInt(arg.From, 10))
		}
		if arg.To > 0 {
			params.Set("to", strconv.FormatInt(arg.To, 10))
		}
	}
	var resp []*CrossexAccountBookRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/account_book", params), nil, nil, &resp)
}

// GetCrossexCoinDiscountRates retrieves the currency discount rates for CrossEx assets.
// Optionally filter by coin and exchange type.
func (e *Exchange) GetCrossexCoinDiscountRates(ctx context.Context, coin, exchangeType string) ([]*CrossexCoinDiscountRate, error) {
	params := url.Values{}
	if coin != "" {
		params.Set("coin", coin)
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossexCoinDiscountRate
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, common.EncodeURLValues("crossex/coin_discount_rate", params), nil, nil, &resp)
}
