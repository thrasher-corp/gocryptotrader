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
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errCrossExchangeExchangeTypeRequired = errors.New("crossex exchange type required")
	errCrossExchangeFromAccountRequired  = errors.New("crossex from account required")
	errCrossExchangeToAccountRequired    = errors.New("crossex to account required")
	errCrossExchangeQuoteIDRequired      = errors.New("crossex quote ID required")
	errCrossExchangeLeverageRequired     = errors.New("crossex leverage required")
)

// GetCrossExchangeSymbols retrieves symbol information for CrossEx trading pairs.
func (e *Exchange) GetCrossExchangeSymbols(ctx context.Context, symbols []string) ([]*CrossExchangeSymbol, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp []*CrossExchangeSymbol
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, crossexSymbolsEPL, common.EncodeURLValues("crossex/symbols", params), &resp)
}

// GetCrossExchangeRiskLimits retrieves risk limit information for CrossEx trading pairs.
func (e *Exchange) GetCrossExchangeRiskLimits(ctx context.Context, symbols []string) ([]*CrossExchangeRiskLimit, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp []*CrossExchangeRiskLimit
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, crossexRiskLimitEPL, common.EncodeURLValues("crossex/risk_limit", params), &resp)
}

// GetCrossExchangeTransferCoins retrieves the list of currencies supported for CrossEx transfers.
func (e *Exchange) GetCrossExchangeTransferCoins(ctx context.Context, coin currency.Code) ([]*CrossExchangeTransferCoin, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	var resp []*CrossExchangeTransferCoin
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexTransfersCoinEPL, http.MethodGet, "crossex/transfers/coin", params, nil, &resp)
}

// GetCrossExchangeTransferHistory retrieves the fund transfer history for the authenticated user.
func (e *Exchange) GetCrossExchangeTransferHistory(ctx context.Context, arg *GetCrossExchangeTransferHistoryRequest) ([]*CrossExchangeTransferRecord, error) {
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
	var resp []*CrossExchangeTransferRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetTransfersEPL, http.MethodGet, "crossex/transfers", params, nil, &resp)
}

// CrossExchangeFundTransfer initiates a fund transfer to or from the CrossEx account.
func (e *Exchange) CrossExchangeFundTransfer(ctx context.Context, arg *CrossExchangeTransferRequest) (*CrossExchangeTransferResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: crossex amount required", order.ErrAmountMustBeSet)
	}
	if arg.From == "" {
		return nil, errCrossExchangeFromAccountRequired
	}
	if arg.To == "" {
		return nil, errCrossExchangeToAccountRequired
	}
	var resp CrossExchangeTransferResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCreateTransfersEPL, http.MethodPost, "crossex/transfers", nil, arg, &resp)
}

// CreateCrossExchangeOrder places a new order on the CrossEx platform.
func (e *Exchange) CreateCrossExchangeOrder(ctx context.Context, arg *CrossExchangeOrderCreateRequest) (*CrossExchangeOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ExchangeType == "" {
		return nil, errCrossExchangeExchangeTypeRequired
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == order.UnknownSide {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	var resp CrossExchangeOrder
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCreateOrdersEPL, http.MethodPost, "crossex/orders", nil, arg, &resp)
}

// GetCrossExchangeOrderDetails retrieves details for a specific CrossEx order.
func (e *Exchange) GetCrossExchangeOrderDetails(ctx context.Context, orderID string) (*CrossExchangeOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp CrossExchangeOrder
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetOrdersEPL, http.MethodGet, "crossex/orders/"+orderID, nil, nil, &resp)
}

// ModifyCrossExchangeOrder modifies an existing CrossEx order's quantity or price.
func (e *Exchange) ModifyCrossExchangeOrder(ctx context.Context, orderID string, arg *CrossExchangeOrderUpdateRequest) (*CrossExchangeOrderActionResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp CrossExchangeOrderActionResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexUpdateOrdersEPL, http.MethodPut, "crossex/orders/"+orderID, nil, arg, &resp)
}

// CancelCrossExchangeOrder cancels an existing CrossEx order.
func (e *Exchange) CancelCrossExchangeOrder(ctx context.Context, orderID string) (*CrossExchangeOrderActionResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp CrossExchangeOrderActionResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexDeleteOrdersEPL, http.MethodDelete, "crossex/orders/"+orderID, nil, nil, &resp)
}

// GetCrossExchangeConvertQuote retrieves a flash swap quote for a CrossEx currency conversion.
func (e *Exchange) GetCrossExchangeConvertQuote(ctx context.Context, arg *CrossExchangeConvertQuoteRequest) (*CrossExchangeConvertQuoteResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ExchangeType == "" {
		return nil, errCrossExchangeExchangeTypeRequired
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
	var resp CrossExchangeConvertQuoteResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexConvertQuoteEPL, http.MethodPost, "crossex/convert/quote", nil, arg, &resp)
}

// ExecuteCrossExchangeConvertOrder executes a CrossEx flash swap using a previously obtained quote ID.
func (e *Exchange) ExecuteCrossExchangeConvertOrder(ctx context.Context, quoteID string) (*CrossExchangeConvertOrderResponse, error) {
	if quoteID == "" {
		return nil, errCrossExchangeQuoteIDRequired
	}
	body := CrossExchangeConvertOrderRequest{QuoteID: quoteID}
	var resp CrossExchangeConvertOrderResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexConvertOrdersEPL, http.MethodPost, "crossex/convert/orders", nil, &body, &resp)
}

// GetCrossExchangeAccountAssets retrieves the CrossEx account asset information.
func (e *Exchange) GetCrossExchangeAccountAssets(ctx context.Context, exchangeType string) ([]*CrossExchangeAccount, error) {
	params := url.Values{}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossExchangeAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetAccountsEPL, http.MethodGet, "crossex/accounts", params, nil, &resp)
}

// UpdateCrossExchangeAccount modifies the CrossEx account's position mode or account mode.
func (e *Exchange) UpdateCrossExchangeAccount(ctx context.Context, arg *CrossExchangeAccountUpdateRequest) (*CrossExchangeAccountUpdateResponse, error) {
	var resp CrossExchangeAccountUpdateResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexUpdateAccountsEPL, http.MethodPut, "crossex/accounts", nil, arg, &resp)
}

// GetCrossExchangeContractLeverage retrieves the leverage multiplier for CrossEx contract trading pairs.
func (e *Exchange) GetCrossExchangeContractLeverage(ctx context.Context, symbols []string) (map[string]types.Number, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp map[string]types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetPositionsLeverageEPL, http.MethodGet, "crossex/positions/leverage", params, nil, &resp)
}

// SetCrossExchangeContractLeverage sets the leverage multiplier for a CrossEx contract trading pair.
func (e *Exchange) SetCrossExchangeContractLeverage(ctx context.Context, arg *CrossExchangeLeverageRequest) (*CrossExchangeLeverageResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Leverage <= 0 {
		return nil, errCrossExchangeLeverageRequired
	}
	var resp CrossExchangeLeverageResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCreatePositionsLeverageEPL, http.MethodPost, "crossex/positions/leverage", nil, arg, &resp)
}

// GetCrossExchangeMarginLeverage retrieves the leverage multiplier for CrossEx leveraged (margin) trading pairs.
func (e *Exchange) GetCrossExchangeMarginLeverage(ctx context.Context, symbols []string) (map[string]types.Number, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp map[string]types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetMarginPositionsLeverageEPL, http.MethodGet, "crossex/margin_positions/leverage", params, nil, &resp)
}

// SetCrossExchangeMarginLeverage sets the leverage multiplier for a CrossEx leveraged (margin) trading pair.
func (e *Exchange) SetCrossExchangeMarginLeverage(ctx context.Context, arg *CrossExchangeLeverageRequest) (*CrossExchangeLeverageResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Leverage <= 0 {
		return nil, errCrossExchangeLeverageRequired
	}
	var resp CrossExchangeLeverageResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCreateMarginPositionsLeverageEPL, http.MethodPost, "crossex/margin_positions/leverage", nil, arg, &resp)
}

// CloseCrossExchangePosition fully closes an open CrossEx contract position.
func (e *Exchange) CloseCrossExchangePosition(ctx context.Context, arg *CrossExchangeClosePositionRequest) (*CrossExchangeOrderActionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp CrossExchangeOrderActionResponse
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexPositionEPL, http.MethodPost, "crossex/position", nil, arg, &resp)
}

// GetCrossExchangeInterestRates retrieves margin asset interest rates.
func (e *Exchange) GetCrossExchangeInterestRates(ctx context.Context, coin currency.Code, exchangeType string) ([]*CrossExchangeInterestRate, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossExchangeInterestRate
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexInterestRateEPL, http.MethodGet, "crossex/interest_rate", params, nil, &resp)
}

// GetCrossExchangeUserFeeRates retrieves the fee rates for the authenticated CrossEx user.
func (e *Exchange) GetCrossExchangeUserFeeRates(ctx context.Context) ([]*CrossExchangeFee, error) {
	var resp []*CrossExchangeFee
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexFeeEPL, http.MethodGet, "crossex/fee", nil, nil, &resp)
}

// GetCrossExchangeContractPositions retrieves the authenticated user's open CrossEx contract positions.
func (e *Exchange) GetCrossExchangeContractPositions(ctx context.Context, symbol, exchangeType string) ([]*CrossExchangePosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossExchangePosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexPositionsEPL, http.MethodGet, "crossex/positions", params, nil, &resp)
}

// GetCrossExchangeMarginPositions retrieves the authenticated user's open CrossEx leveraged (margin) positions.
func (e *Exchange) GetCrossExchangeMarginPositions(ctx context.Context, symbol, exchangeType string) ([]*CrossExchangeMarginPosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossExchangeMarginPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexMarginPositionsEPL, http.MethodGet, "crossex/margin_positions", params, nil, &resp)
}

// GetCrossExchangeADLRank retrieves the ADL (Auto-Deleveraging) position reduction ranking for a CrossEx symbol.
func (e *Exchange) GetCrossExchangeADLRank(ctx context.Context, symbol string) ([]*CrossExchangeAdlRank, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []*CrossExchangeAdlRank
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexAdlRankEPL, http.MethodGet, "crossex/adl_rank", params, nil, &resp)
}

// GetCrossExchangeOpenOrders retrieves all currently open CrossEx orders.
func (e *Exchange) GetCrossExchangeOpenOrders(ctx context.Context, arg *GetCrossExchangeOpenOrdersRequest) ([]*CrossExchangeOrder, error) {
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
	var resp []*CrossExchangeOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexOpenOrdersEPL, http.MethodGet, "crossex/open_orders", params, nil, &resp)
}

// GetCrossExchangeOrderHistory retrieves the CrossEx order history for the authenticated user.
func (e *Exchange) GetCrossExchangeOrderHistory(ctx context.Context, arg *GetCrossExchangeOrderHistoryRequest) ([]*CrossExchangeOrder, error) {
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
	var resp []*CrossExchangeOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexHistoryOrdersEPL, http.MethodGet, "crossex/history_orders", params, nil, &resp)
}

// GetCrossExchangeContractPositionHistory retrieves closed CrossEx contract position history.
func (e *Exchange) GetCrossExchangeContractPositionHistory(ctx context.Context, arg *GetCrossExchangePositionHistoryRequest) ([]*CrossExchangeHistoricalPosition, error) {
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
	var resp []*CrossExchangeHistoricalPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexHistoryPositionsEPL, http.MethodGet, "crossex/history_positions", params, nil, &resp)
}

// GetCrossExchangeMarginPositionHistory retrieves closed CrossEx leveraged (margin) position history.
func (e *Exchange) GetCrossExchangeMarginPositionHistory(ctx context.Context, arg *GetCrossExchangePositionHistoryRequest) ([]*CrossExchangeHistoricalMarginPosition, error) {
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
	var resp []*CrossExchangeHistoricalMarginPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexHistoryMarginPositionsEPL, http.MethodGet, "crossex/history_margin_positions", params, nil, &resp)
}

// GetCrossExchangeMarginInterestHistory retrieves the leveraged interest deduction history.
func (e *Exchange) GetCrossExchangeMarginInterestHistory(ctx context.Context, arg *GetCrossExchangeMarginInterestHistoryRequest) ([]*CrossExchangeMarginInterestRecord, error) {
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
	var resp []*CrossExchangeMarginInterestRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexHistoryMarginInterestsEPL, http.MethodGet, "crossex/history_margin_interests", params, nil, &resp)
}

// GetCrossExchangeTradeHistory retrieves the trade history for the authenticated CrossEx user.
func (e *Exchange) GetCrossExchangeTradeHistory(ctx context.Context, arg *GetCrossExchangeTradeHistoryRequest) ([]*CrossExchangeTrade, error) {
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
	var resp []*CrossExchangeTrade
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexHistoryTradesEPL, http.MethodGet, "crossex/history_trades", params, nil, &resp)
}

// GetCrossExchangeAccountBook retrieves the account asset change history for the authenticated CrossEx user.
func (e *Exchange) GetCrossExchangeAccountBook(ctx context.Context, arg *GetCrossExchangeAccountBookRequest) ([]*CrossExchangeAccountBookRecord, error) {
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
	var resp []*CrossExchangeAccountBookRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexAccountBookEPL, http.MethodGet, "crossex/account_book", params, nil, &resp)
}

// GetCrossExchangeCoinDiscountRates retrieves the currency discount rates for CrossEx assets.
func (e *Exchange) GetCrossExchangeCoinDiscountRates(ctx context.Context, coin, exchangeType string) ([]*CrossExchangeCoinDiscountRate, error) {
	params := url.Values{}
	if coin != "" {
		params.Set("coin", coin)
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossExchangeCoinDiscountRate
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCoinDiscountRateEPL, http.MethodGet, "crossex/coin_discount_rate", params, nil, &resp)
}
