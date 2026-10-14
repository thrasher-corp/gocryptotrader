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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errCrossExchangeExchangeTypeRequired = errors.New("crossex exchange type required")
	errCrossExchangeFromAccountRequired  = errors.New("crossex from account required")
	errCrossExchangeToAccountRequired    = errors.New("crossex to account required")
	errCrossExchangeLeverageRequired     = errors.New("crossex leverage required")
	errCrossExchangePositionSideRequired = errors.New("crossex position side required to close a margin position")
	errCrossExchangeMarginModeRequired   = errors.New("crossex margin mode required")
	errCrossExchangeMarginRequired       = errors.New("crossex margin adjustment required")
	errCrossExchangeTimeInForceConflict  = errors.New("crossex retail price improvement orders cannot set another time in force")
)

func formatCrossExchangeSymbols(symbols []CrossExchangeSymbolIdentifier) (string, error) {
	formatted := make([]string, len(symbols))
	for x := range symbols {
		if symbols[x].IsEmpty() {
			return "", fmt.Errorf("invalid crossex symbol at index %d: %w", x, currency.ErrSymbolStringEmpty)
		}
		var err error
		formatted[x], err = symbols[x].Format()
		if err != nil {
			return "", fmt.Errorf("invalid crossex symbol at index %d: %w", x, err)
		}
	}
	return strings.Join(formatted, ","), nil
}

// crossExchangeHistoryParams builds the symbol, millisecond time range and paging filters the CrossEx record
// endpoints share
func crossExchangeHistoryParams(symbol CrossExchangeSymbolIdentifier, startTime, endTime time.Time, page, limit uint64) (url.Values, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		formatted, err := symbol.Format()
		if err != nil {
			return nil, err
		}
		params.Set("symbol", formatted)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("to", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}

// GetCrossExchangeSymbols retrieves symbol information for CrossEx trading pairs.
func (e *Exchange) GetCrossExchangeSymbols(ctx context.Context, symbols []CrossExchangeSymbolIdentifier) ([]*CrossExchangeSymbol, error) {
	params := url.Values{}
	if len(symbols) != 0 {
		formatted, err := formatCrossExchangeSymbols(symbols)
		if err != nil {
			return nil, err
		}
		params.Set("symbols", formatted)
	}
	var resp []*CrossExchangeSymbol
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, crossexSymbolsEPL, common.EncodeURLValues("crossex/rule/symbols", params), &resp)
}

// GetCrossExchangeRiskLimits retrieves risk limit information for CrossEx futures and margin trading pairs.
func (e *Exchange) GetCrossExchangeRiskLimits(ctx context.Context, symbols []CrossExchangeSymbolIdentifier) ([]*CrossExchangeRiskLimit, error) {
	if len(symbols) == 0 {
		return nil, currency.ErrSymbolStringEmpty
	}
	formatted, err := formatCrossExchangeSymbols(symbols)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbols", formatted)
	var resp []*CrossExchangeRiskLimit
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, crossexRiskLimitEPL, common.EncodeURLValues("crossex/rule/risk_limits", params), &resp)
}

// GetCrossExchangeTransferCoins retrieves the list of currencies supported for CrossEx transfers.
func (e *Exchange) GetCrossExchangeTransferCoins(ctx context.Context, coin currency.Code) ([]*CrossExchangeTransferCoin, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	var resp []*CrossExchangeTransferCoin
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, crossexTransfersCoinEPL, common.EncodeURLValues("crossex/transfers/coin", params), &resp)
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
	var resp *CrossExchangeTransferResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCreateTransfersEPL, http.MethodPost, "crossex/transfers", nil, arg, &resp)
}

// GetCrossExchangeTransferHistory retrieves the fund transfer history for the authenticated user.
func (e *Exchange) GetCrossExchangeTransferHistory(ctx context.Context, arg *GetCrossExchangeTransferHistoryRequest) ([]*CrossExchangeTransferRecord, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := crossExchangeHistoryParams(CrossExchangeSymbolIdentifier{}, arg.StartTime, arg.EndTime, arg.Page, arg.Limit)
	if err != nil {
		return nil, err
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.String())
	}
	if arg.OrderID != "" {
		params.Set("order_id", arg.OrderID)
	}
	var resp []*CrossExchangeTransferRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetTransfersEPL, http.MethodGet, "crossex/transfers", params, nil, &resp)
}

// CreateCrossExchangeOrder places a new order on the CrossEx platform. Orders default to LIMIT when OrderType is
// unset; spot and margin market buys are sized by QuoteQuantity and every other order by Quantity.
func (e *Exchange) CreateCrossExchangeOrder(ctx context.Context, arg *CrossExchangeOrderCreateRequest) (*CrossExchangeOrderActionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side != order.Buy && arg.Side != order.Sell {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType != order.UnknownType && arg.OrderType != order.Limit && arg.OrderType != order.Market {
		return nil, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, arg.OrderType)
	}
	if arg.OrderType == order.Market && arg.Side == order.Buy && (arg.Symbol.Asset == asset.Spot || arg.Symbol.Asset == asset.Margin) {
		if arg.QuoteQuantity <= 0 {
			return nil, fmt.Errorf("%w: crossex quote quantity required for spot and margin market buys", order.ErrAmountMustBeSet)
		}
	} else if arg.Quantity <= 0 {
		return nil, fmt.Errorf("%w: crossex quantity required", order.ErrAmountMustBeSet)
	}
	if arg.OrderType != order.Market && arg.Price <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	if arg.RetailPriceImprovement && arg.TimeInForce != order.UnknownTIF {
		return nil, errCrossExchangeTimeInForceConflict
	}
	var resp *CrossExchangeOrderActionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCreateOrdersEPL, http.MethodPost, "crossex/orders", nil, arg, &resp)
}

// CancelBatchCrossExchangeOrders cancels multiple CrossEx orders by order ID or client-defined text ID, preferring
// the order ID when both are set.
func (e *Exchange) CancelBatchCrossExchangeOrders(ctx context.Context, args []*CrossExchangeBatchCancelOrderRequest) ([]*CrossExchangeBatchCancelOrderResponse, error) {
	if len(args) == 0 {
		return nil, errNoValidParameterPassed
	}
	for x := range args {
		if err := common.NilGuard(args[x]); err != nil {
			return nil, err
		}
		if args[x].OrderID == "" && args[x].Text == "" {
			return nil, order.ErrOrderIDNotSet
		}
	}
	var resp []*CrossExchangeBatchCancelOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCancelBatchOrdersEPL, http.MethodPost, "crossex/batch_cancel_orders", nil, &args, &resp)
}

// CancelCrossExchangeOrder cancels an existing CrossEx order by its order ID or client-defined text ID.
func (e *Exchange) CancelCrossExchangeOrder(ctx context.Context, orderID string) (*CrossExchangeOrderActionResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CrossExchangeOrderActionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexDeleteOrdersEPL, http.MethodDelete, "crossex/orders/"+orderID, nil, nil, &resp)
}

// ModifyCrossExchangeOrder modifies an existing CrossEx order's quantity or price, identified by its order ID or
// client-defined text ID.
func (e *Exchange) ModifyCrossExchangeOrder(ctx context.Context, orderID string, arg *CrossExchangeOrderUpdateRequest) (*CrossExchangeOrderActionResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Quantity <= 0 && arg.Price <= 0 {
		return nil, fmt.Errorf("%w: crossex quantity or price required", errNoValidParameterPassed)
	}
	var resp *CrossExchangeOrderActionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexUpdateOrdersEPL, http.MethodPut, "crossex/orders/"+orderID, nil, arg, &resp)
}

// GetCrossExchangeOrderDetails retrieves details for a specific CrossEx order by its order ID or client-defined text
// ID.
func (e *Exchange) GetCrossExchangeOrderDetails(ctx context.Context, orderID string) (*CrossExchangeOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CrossExchangeOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetOrdersEPL, http.MethodGet, "crossex/orders/"+orderID, nil, nil, &resp)
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
	var resp *CrossExchangeConvertQuoteResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexConvertQuoteEPL, http.MethodPost, "crossex/convert/quote", nil, arg, &resp)
}

// ExecuteCrossExchangeConvertOrder executes a CrossEx flash swap using a previously obtained quote ID.
func (e *Exchange) ExecuteCrossExchangeConvertOrder(ctx context.Context, quoteID string) (*CrossExchangeConvertOrderResponse, error) {
	if quoteID == "" {
		return nil, errQuoteIDRequired
	}
	var resp *CrossExchangeConvertOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexConvertOrdersEPL, http.MethodPost, "crossex/convert/orders", nil, &CrossExchangeConvertOrderRequest{QuoteID: quoteID}, &resp)
}

// UpdateCrossExchangeAccount modifies the CrossEx account's futures position mode or account mode.
func (e *Exchange) UpdateCrossExchangeAccount(ctx context.Context, arg *CrossExchangeAccountUpdateRequest) (*CrossExchangeAccountUpdateResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	var resp *CrossExchangeAccountUpdateResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexUpdateAccountsEPL, http.MethodPut, "crossex/accounts", nil, arg, &resp)
}

// GetCrossExchangeAccountAssets retrieves the CrossEx account asset information. The exchange type is only required
// in isolated exchange mode.
func (e *Exchange) GetCrossExchangeAccountAssets(ctx context.Context, exchangeType string) (*CrossExchangeAccount, error) {
	params := url.Values{}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp *CrossExchangeAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetAccountsEPL, http.MethodGet, "crossex/accounts", params, nil, &resp)
}

// SetCrossExchangeContractLeverage sets the leverage multiplier for a CrossEx contract trading pair.
func (e *Exchange) SetCrossExchangeContractLeverage(ctx context.Context, arg *CrossExchangeLeverageRequest) (*CrossExchangeLeverageResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Leverage <= 0 {
		return nil, errCrossExchangeLeverageRequired
	}
	var resp *CrossExchangeLeverageResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCreatePositionsLeverageEPL, http.MethodPost, "crossex/positions/leverage", nil, arg, &resp)
}

// GetCrossExchangeContractLeverage retrieves the leverage multiplier for CrossEx contract trading pairs, keyed by
// symbol.
func (e *Exchange) GetCrossExchangeContractLeverage(ctx context.Context, symbols []CrossExchangeSymbolIdentifier) (map[string]types.Number, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		formatted, err := formatCrossExchangeSymbols(symbols)
		if err != nil {
			return nil, err
		}
		params.Set("symbols", formatted)
	}
	var resp map[string]types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetPositionsLeverageEPL, http.MethodGet, "crossex/positions/leverage", params, nil, &resp)
}

// SetCrossExchangeMarginLeverage sets the leverage multiplier for a CrossEx leveraged (margin) trading pair.
func (e *Exchange) SetCrossExchangeMarginLeverage(ctx context.Context, arg *CrossExchangeLeverageRequest) (*CrossExchangeLeverageResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Leverage <= 0 {
		return nil, errCrossExchangeLeverageRequired
	}
	var resp *CrossExchangeLeverageResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCreateMarginPositionsLeverageEPL, http.MethodPost, "crossex/margin_positions/leverage", nil, arg, &resp)
}

// GetCrossExchangeMarginLeverage retrieves the leverage multiplier for CrossEx leveraged (margin) trading pairs,
// keyed by symbol.
func (e *Exchange) GetCrossExchangeMarginLeverage(ctx context.Context, symbols []CrossExchangeSymbolIdentifier) (map[string]types.Number, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		formatted, err := formatCrossExchangeSymbols(symbols)
		if err != nil {
			return nil, err
		}
		params.Set("symbols", formatted)
	}
	var resp map[string]types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetMarginPositionsLeverageEPL, http.MethodGet, "crossex/margin_positions/leverage", params, nil, &resp)
}

// CloseCrossExchangePosition fully closes a FUTURE or MARGIN position when its symbol has no pending orders
// and its notional is below minNotional or its quantity is below minSize.
func (e *Exchange) CloseCrossExchangePosition(ctx context.Context, arg *CrossExchangeClosePositionRequest) (*CrossExchangeOrderActionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Symbol.Asset == asset.Margin && arg.PositionSide == "" {
		return nil, errCrossExchangePositionSideRequired
	}
	var resp *CrossExchangeOrderActionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexPositionEPL, http.MethodPost, "crossex/position", nil, arg, &resp)
}

// SetCrossExchangePositionMarginMode changes a futures trading pair's margin mode. Gate only supports Hyperliquid
// futures, and only while the pair has no open orders or positions.
func (e *Exchange) SetCrossExchangePositionMarginMode(ctx context.Context, arg *CrossExchangeMarginModeRequest) (*CrossExchangeMarginModeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.MarginMode == "" {
		return nil, errCrossExchangeMarginModeRequired
	}
	var resp *CrossExchangeMarginModeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexUpdatePositionsMarginModeEPL, http.MethodPost, "crossex/positions/margin_mode", nil, arg, &resp)
}

// GetCrossExchangePositionMarginMode retrieves a futures trading pair's margin mode.
func (e *Exchange) GetCrossExchangePositionMarginMode(ctx context.Context, symbol CrossExchangeSymbolIdentifier) (*CrossExchangeMarginModeResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	formatted, err := symbol.Format()
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", formatted)
	var resp *CrossExchangeMarginModeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexGetPositionsMarginModeEPL, http.MethodGet, "crossex/positions/margin_mode", params, nil, &resp)
}

// UpdateCrossExchangeIsolatedMargin increases or decreases an isolated futures position's margin. Gate only
// supports Hyperliquid isolated futures positions.
func (e *Exchange) UpdateCrossExchangeIsolatedMargin(ctx context.Context, arg *CrossExchangeIsolatedMarginRequest) (*CrossExchangeIsolatedMarginResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Margin == 0 {
		return nil, errCrossExchangeMarginRequired
	}
	var resp *CrossExchangeIsolatedMarginResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexUpdatePositionsMarginEPL, http.MethodPost, "crossex/positions/margin", nil, arg, &resp)
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
func (e *Exchange) GetCrossExchangeContractPositions(ctx context.Context, symbol CrossExchangeSymbolIdentifier, exchangeType string) ([]*CrossExchangePosition, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		formatted, err := symbol.Format()
		if err != nil {
			return nil, err
		}
		params.Set("symbol", formatted)
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossExchangePosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexPositionsEPL, http.MethodGet, "crossex/positions", params, nil, &resp)
}

// GetCrossExchangeMarginPositions retrieves the authenticated user's open CrossEx leveraged (margin) positions.
func (e *Exchange) GetCrossExchangeMarginPositions(ctx context.Context, symbol CrossExchangeSymbolIdentifier, exchangeType string) ([]*CrossExchangeMarginPosition, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		formatted, err := symbol.Format()
		if err != nil {
			return nil, err
		}
		params.Set("symbol", formatted)
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossExchangeMarginPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexMarginPositionsEPL, http.MethodGet, "crossex/margin_positions", params, nil, &resp)
}

// GetCrossExchangeADLRank retrieves the ADL (Auto-Deleveraging) position reduction ranking for a CrossEx symbol.
func (e *Exchange) GetCrossExchangeADLRank(ctx context.Context, symbol CrossExchangeSymbolIdentifier) ([]*CrossExchangeADLRank, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	formatted, err := symbol.Format()
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", formatted)
	var resp []*CrossExchangeADLRank
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexADLRankEPL, http.MethodGet, "crossex/adl_rank", params, nil, &resp)
}

// GetCrossExchangeOpenOrders retrieves all currently open CrossEx orders.
func (e *Exchange) GetCrossExchangeOpenOrders(ctx context.Context, arg *GetCrossExchangeOpenOrdersRequest) ([]*CrossExchangeOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !arg.Symbol.IsEmpty() {
		formatted, err := arg.Symbol.Format()
		if err != nil {
			return nil, err
		}
		params.Set("symbol", formatted)
	}
	if arg.ExchangeType != "" {
		params.Set("exchange_type", arg.ExchangeType)
	}
	if arg.BusinessType != "" {
		params.Set("business_type", arg.BusinessType)
	}
	var resp []*CrossExchangeOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexOpenOrdersEPL, http.MethodGet, "crossex/open_orders", params, nil, &resp)
}

// GetCrossExchangeOrderHistory retrieves the CrossEx order history for the authenticated user.
func (e *Exchange) GetCrossExchangeOrderHistory(ctx context.Context, arg *GetCrossExchangeOrderHistoryRequest) ([]*CrossExchangeOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := crossExchangeHistoryParams(arg.Symbol, arg.StartTime, arg.EndTime, arg.Page, arg.Limit)
	if err != nil {
		return nil, err
	}
	if len(arg.Attributes) != 0 {
		params.Set("attributes", strings.Join(arg.Attributes, ","))
	}
	var resp []*CrossExchangeOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexHistoryOrdersEPL, http.MethodGet, "crossex/history_orders", params, nil, &resp)
}

// GetCrossExchangeContractPositionHistory retrieves closed CrossEx contract position history.
func (e *Exchange) GetCrossExchangeContractPositionHistory(ctx context.Context, arg *GetCrossExchangePositionHistoryRequest) ([]*CrossExchangeHistoricalPosition, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := crossExchangeHistoryParams(arg.Symbol, arg.StartTime, arg.EndTime, arg.Page, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []*CrossExchangeHistoricalPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexHistoryPositionsEPL, http.MethodGet, "crossex/history_positions", params, nil, &resp)
}

// GetCrossExchangeMarginPositionHistory retrieves closed CrossEx leveraged (margin) position history.
func (e *Exchange) GetCrossExchangeMarginPositionHistory(ctx context.Context, arg *GetCrossExchangePositionHistoryRequest) ([]*CrossExchangeHistoricalMarginPosition, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := crossExchangeHistoryParams(arg.Symbol, arg.StartTime, arg.EndTime, arg.Page, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []*CrossExchangeHistoricalMarginPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexHistoryMarginPositionsEPL, http.MethodGet, "crossex/history_margin_positions", params, nil, &resp)
}

// GetCrossExchangeMarginInterestHistory retrieves the leveraged interest deduction history.
func (e *Exchange) GetCrossExchangeMarginInterestHistory(ctx context.Context, arg *GetCrossExchangeMarginInterestHistoryRequest) ([]*CrossExchangeMarginInterestRecord, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := crossExchangeHistoryParams(arg.Symbol, arg.StartTime, arg.EndTime, arg.Page, arg.Limit)
	if err != nil {
		return nil, err
	}
	if arg.ExchangeType != "" {
		params.Set("exchange_type", arg.ExchangeType)
	}
	var resp []*CrossExchangeMarginInterestRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexHistoryMarginInterestsEPL, http.MethodGet, "crossex/history_margin_interests", params, nil, &resp)
}

// GetCrossExchangeTradeHistory retrieves the trade history for the authenticated CrossEx user.
func (e *Exchange) GetCrossExchangeTradeHistory(ctx context.Context, arg *GetCrossExchangeTradeHistoryRequest) ([]*CrossExchangeTrade, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := crossExchangeHistoryParams(arg.Symbol, arg.StartTime, arg.EndTime, arg.Page, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []*CrossExchangeTrade
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexHistoryTradesEPL, http.MethodGet, "crossex/history_trades", params, nil, &resp)
}

// GetCrossExchangeAccountBook retrieves the account asset change history for the authenticated CrossEx user.
func (e *Exchange) GetCrossExchangeAccountBook(ctx context.Context, arg *GetCrossExchangeAccountBookRequest) ([]*CrossExchangeAccountBookRecord, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := crossExchangeHistoryParams(CrossExchangeSymbolIdentifier{}, arg.StartTime, arg.EndTime, arg.Page, arg.Limit)
	if err != nil {
		return nil, err
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.String())
	}
	if arg.StatementType != "" {
		params.Set("statement_type", arg.StatementType)
	}
	var resp []*CrossExchangeAccountBookRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexAccountBookEPL, http.MethodGet, "crossex/account_book", params, nil, &resp)
}

// GetCrossExchangeCoinDiscountRates retrieves the currency discount rates for CrossEx assets.
func (e *Exchange) GetCrossExchangeCoinDiscountRates(ctx context.Context, coin currency.Code, exchangeType string) ([]*CrossExchangeCoinDiscountRate, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if exchangeType != "" {
		params.Set("exchange_type", exchangeType)
	}
	var resp []*CrossExchangeCoinDiscountRate
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexCoinDiscountRateEPL, http.MethodGet, "crossex/coin_discount_rate", params, nil, &resp)
}

// GetCrossExchangeMarketTickers retrieves exchange tickers for CrossEx spot and futures symbols, or for every
// symbol when none are given. Margin symbols are not accepted.
func (e *Exchange) GetCrossExchangeMarketTickers(ctx context.Context, symbols []CrossExchangeSymbolIdentifier) ([]*CrossExchangeMarketTickerResponse, error) {
	params := url.Values{}
	if len(symbols) != 0 {
		for x := range symbols {
			if symbols[x].Asset == asset.Margin {
				return nil, fmt.Errorf("%w: crossex tickers do not accept margin symbol %s", asset.ErrNotSupported, symbols[x])
			}
		}
		formatted, err := formatCrossExchangeSymbols(symbols)
		if err != nil {
			return nil, err
		}
		params.Set("symbols", formatted)
	}
	var resp []*CrossExchangeMarketTickerResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexMarketTickersEPL, http.MethodGet, "crossex/market/tickers", params, nil, &resp)
}

// GetCrossExchangeFundingInfo retrieves exchange funding rates, funding intervals and next funding times for CrossEx
// futures symbols, or for every futures symbol when none are given.
func (e *Exchange) GetCrossExchangeFundingInfo(ctx context.Context, symbols []CrossExchangeSymbolIdentifier) ([]*CrossExchangeFundingInfoResponse, error) {
	params := url.Values{}
	if len(symbols) != 0 {
		formatted, err := formatCrossExchangeSymbols(symbols)
		if err != nil {
			return nil, err
		}
		params.Set("symbols", formatted)
	}
	var resp []*CrossExchangeFundingInfoResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, crossexMarketFundingInfoEPL, http.MethodGet, "crossex/market/funding_info", params, nil, &resp)
}
