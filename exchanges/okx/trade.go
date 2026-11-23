package okx

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errInvalidNewSizeOrPriceInformation = errors.New("invalid new size or price information")
	errInvalidLockInterval              = errors.New("invalid lock interval")
	// errConflictingAmendPrices reports an amendment stating its new price more than one way, as OKX takes one of
	// newPx, newPxUsd and newPxVol
	errConflictingAmendPrices = fmt.Errorf("%w: an amendment states one of a new price, a new USD price and a new implied volatility", errInvalidNewSizeOrPriceInformation)
)

const (
	maxOrdersPerBatch          = 20
	maxEasyConvertCurrencies   = 5
	maxOneClickRepayCurrencies = 5
	maxMassCancelLockInterval  = 10000
)

// PlaceOrder places an order
func (e *Exchange) PlaceOrder(ctx context.Context, arg *PlaceOrderRequest) (*OrderActionResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	var resp *OrderActionResponse
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, placeOrderEPL, http.MethodPost, "trade/order", arg, &resp, request.AuthenticatedRequest)
	if err != nil {
		if resp != nil && resp.StatusMessage != "" {
			return nil, fmt.Errorf("%w; %w", err, getStatusError(resp.StatusCode, resp.StatusMessage))
		}
		return nil, err
	}
	return resp, nil
}

// PlaceMultipleOrders places up to 20 orders in one request
func (e *Exchange) PlaceMultipleOrders(ctx context.Context, args []PlaceOrderRequest) ([]OrderActionResponse, error) {
	if len(args) == 0 {
		return nil, order.ErrSubmissionIsNil
	}
	if len(args) > maxOrdersPerBatch {
		return nil, fmt.Errorf("%w, cannot place more than %d orders", errExceedLimit, maxOrdersPerBatch)
	}
	for x := range args {
		if err := args[x].Validate(); err != nil {
			return nil, err
		}
	}
	epl := batchEndpointLimit(placeOrderEPL, placeMultipleOrdersEPL, len(args))
	var resp []OrderActionResponse
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodPost, "trade/batch-orders", args, &resp, request.AuthenticatedRequest)
	if err != nil {
		if len(resp) == 0 {
			return nil, err
		}
		var errs error
		for x := range resp {
			errs = common.AppendError(errs, getStatusError(resp[x].StatusCode, resp[x].StatusMessage))
		}
		return resp, common.AppendError(err, errs)
	}
	return resp, nil
}

// CancelSingleOrder cancels an incomplete order
func (e *Exchange) CancelSingleOrder(ctx context.Context, arg *CancelOrderRequest) (*OrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OrderActionResponse
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, cancelOrderEPL, http.MethodPost, "trade/cancel-order", arg, &resp, request.AuthenticatedRequest)
	if err != nil {
		if resp != nil && resp.StatusMessage != "" {
			return nil, fmt.Errorf("%w; %w", err, getStatusError(resp.StatusCode, resp.StatusMessage))
		}
		return nil, err
	}
	return resp, nil
}

// CancelMultipleOrders cancels up to 20 incomplete orders in one request
func (e *Exchange) CancelMultipleOrders(ctx context.Context, args []CancelOrderRequest) ([]*OrderActionResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	if len(args) > maxOrdersPerBatch {
		return nil, fmt.Errorf("%w, cannot cancel more than %d orders", errExceedLimit, maxOrdersPerBatch)
	}
	for x := range args {
		if args[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[x].OrderID == "" && args[x].ClientOrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
	}
	epl := batchEndpointLimit(cancelOrderEPL, cancelMultipleOrdersEPL, len(args))
	var resp []*OrderActionResponse
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodPost, "trade/cancel-batch-orders", args, &resp, request.AuthenticatedRequest)
	if err != nil {
		if len(resp) == 0 {
			return nil, err
		}
		var errs error
		for x := range resp {
			if resp[x] != nil && resp[x].StatusCode != 0 {
				errs = common.AppendError(errs, getStatusError(resp[x].StatusCode, resp[x].StatusMessage))
			}
		}
		// A partially successful batch returns its per-order results with the
		// error, so callers can report the cancellations that succeeded
		return resp, common.AppendError(err, errs)
	}
	return resp, nil
}

// validate checks the parameters OKX requires to amend an order over REST, which identifies the instrument by its
// instrument ID
func (arg *AmendOrderRequest) validate() error {
	if arg.InstrumentID == "" {
		return errMissingInstrumentID
	}
	return arg.validateAmendment()
}

// validateAmendment checks the amendment fields REST and websocket amendments share: the order, and a new size or a
// new price, which an option may state in USD or as implied volatility, but in only one of those ways
func (arg *AmendOrderRequest) validateAmendment() error {
	if arg.ClientOrderID == "" && arg.OrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if arg.NewSize <= 0 && arg.NewPrice <= 0 && arg.NewPriceUSD <= 0 && arg.NewPriceVolatility <= 0 {
		return errInvalidNewSizeOrPriceInformation
	}
	var prices int
	for _, price := range []float64{arg.NewPrice, arg.NewPriceUSD, arg.NewPriceVolatility} {
		if price != 0 {
			prices++
		}
	}
	if prices > 1 {
		return errConflictingAmendPrices
	}
	return nil
}

// AmendOrder amends an incomplete order
func (e *Exchange) AmendOrder(ctx context.Context, arg *AmendOrderRequest) (*OrderActionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *OrderActionResponse
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, amendOrderEPL, http.MethodPost, "trade/amend-order", arg, &resp, request.AuthenticatedRequest)
	if err != nil {
		if resp != nil && resp.StatusMessage != "" {
			// A rejected amend reports its reason in the row's sCode and sMsg,
			// not in the top-level reply
			return nil, fmt.Errorf("%w; %w", err, getStatusError(resp.StatusCode, resp.StatusMessage))
		}
		return nil, err
	}
	return resp, nil
}

// AmendMultipleOrders amends up to 20 incomplete orders in one request
func (e *Exchange) AmendMultipleOrders(ctx context.Context, args []AmendOrderRequest) ([]OrderActionResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	if len(args) > maxOrdersPerBatch {
		return nil, fmt.Errorf("%w, cannot amend more than %d orders", errExceedLimit, maxOrdersPerBatch)
	}
	for x := range args {
		if err := args[x].validate(); err != nil {
			return nil, err
		}
	}
	epl := batchEndpointLimit(amendOrderEPL, amendMultipleOrdersEPL, len(args))
	var resp []OrderActionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodPost, "trade/amend-batch-orders", args, &resp, request.AuthenticatedRequest)
}

// ClosePositions closes the position of an instrument with a market order
func (e *Exchange) ClosePositions(ctx context.Context, arg *ClosePositionsRequest) (*ClosePositionResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	switch arg.MarginMode {
	case TradeModeCross, TradeModeIsolated:
	default:
		return nil, fmt.Errorf("%w: %q, the margin mode is required, use %q or %q", margin.ErrMarginTypeUnsupported, arg.MarginMode, TradeModeCross, TradeModeIsolated)
	}
	// posSide is required only in long/short mode, which this layer cannot
	// see, so an unset value passes and a set one is normalised and checked
	arg.PositionSide = strings.ToLower(arg.PositionSide)
	switch arg.PositionSide {
	case "", positionSideLong, positionSideShort, positionSideNet:
	default:
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.PositionSide)
	}
	var resp *ClosePositionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, closePositionEPL, http.MethodPost, "trade/close-position", arg, &resp, request.AuthenticatedRequest)
}

// GetOrderDetail retrieves an order's details by its order ID or client order
// ID
func (e *Exchange) GetOrderDetail(ctx context.Context, arg *OrderDetailRequest) (*OrderDetailResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	switch {
	case arg.OrderID == "" && arg.ClientOrderID == "":
		return nil, order.ErrOrderIDNotSet
	case arg.ClientOrderID == "":
		params.Set("ordId", arg.OrderID)
	default:
		params.Set("clOrdId", arg.ClientOrderID)
	}
	params.Set("instId", arg.InstrumentID)
	var resp *OrderDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOrderDetEPL, http.MethodGet, common.EncodeURLValues("trade/order", params), nil, &resp, request.AuthenticatedRequest)
}

// GetOrderList retrieves the account's incomplete orders
func (e *Exchange) GetOrderList(ctx context.Context, arg *OrderListRequest) ([]OrderDetailResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", arg.InstrumentType)
	}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.OrderType != "" {
		params.Set("ordType", strings.ToLower(arg.OrderType))
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []OrderDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOrderListEPL, http.MethodGet, common.EncodeURLValues("trade/orders-pending", params), nil, &resp, request.AuthenticatedRequest)
}

// Get7DayOrderHistory retrieves the orders completed in the last 7 days; incomplete orders cancelled without fills are
// kept for 2 hours only
func (e *Exchange) Get7DayOrderHistory(ctx context.Context, arg *OrderHistoryRequest) ([]OrderDetailResponse, error) {
	return e.getOrderHistory(ctx, arg, "trade/orders-history", getOrderHistory7DaysEPL)
}

// Get3MonthOrderHistory retrieves the orders completed in the last 3 months, leaving out orders cancelled without fills
func (e *Exchange) Get3MonthOrderHistory(ctx context.Context, arg *OrderHistoryRequest) ([]OrderDetailResponse, error) {
	return e.getOrderHistory(ctx, arg, "trade/orders-history-archive", getOrderHistory3MonthsEPL)
}

// getOrderHistory retrieves completed orders from the given history listing
func (e *Exchange) getOrderHistory(ctx context.Context, arg *OrderHistoryRequest, route string, rateLimit request.EndpointLimit) ([]OrderDetailResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	if !arg.Begin.IsZero() && !arg.End.IsZero() && arg.Begin.After(arg.End) {
		return nil, fmt.Errorf("%w: begin %s, end %s", common.ErrStartAfterEnd, arg.Begin, arg.End)
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(arg.InstrumentType))
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.OrderType != "" {
		params.Set("ordType", strings.ToLower(arg.OrderType))
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	if arg.Category != "" {
		params.Set("category", strings.ToLower(arg.Category))
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if !arg.Begin.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.Begin.UnixMilli(), 10))
	}
	if !arg.End.IsZero() {
		params.Set("end", strconv.FormatInt(arg.End.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []OrderDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, rateLimit, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, request.AuthenticatedRequest)
}

// GetTransactionDetailsLast3Days retrieves the transactions filled in the last 3 days
func (e *Exchange) GetTransactionDetailsLast3Days(ctx context.Context, arg *TransactionDetailRequest) ([]TransactionDetailResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	return e.getTransactionDetails(ctx, arg, "trade/fills", getTransactionDetail3DaysEPL)
}

// GetTransactionDetailsLast3Months retrieves the transactions filled in the last 3 months
func (e *Exchange) GetTransactionDetailsLast3Months(ctx context.Context, arg *TransactionDetailRequest) ([]TransactionDetailResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	return e.getTransactionDetails(ctx, arg, "trade/fills-history", getTransactionDetail3MonthsEPL)
}

// getTransactionDetails retrieves filled transactions from the given fills listing
func (e *Exchange) getTransactionDetails(ctx context.Context, arg *TransactionDetailRequest, route string, rateLimit request.EndpointLimit) ([]TransactionDetailResponse, error) {
	if !arg.Begin.IsZero() && !arg.End.IsZero() && arg.Begin.After(arg.End) {
		return nil, fmt.Errorf("%w: begin %s, end %s", common.ErrStartAfterEnd, arg.Begin, arg.End)
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", strings.ToUpper(arg.InstrumentType))
	}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.OrderID != "" {
		params.Set("ordId", arg.OrderID)
	}
	if arg.SubType != "" {
		params.Set("subType", arg.SubType)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if !arg.Begin.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.Begin.UnixMilli(), 10))
	}
	if !arg.End.IsZero() {
		params.Set("end", strconv.FormatInt(arg.End.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []TransactionDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, rateLimit, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, request.AuthenticatedRequest)
}

// GetEasyConvertCurrencyList retrieves the small currencies, worth less than $10, easy convert can convert and the
// mainstream currencies it converts them to
func (e *Exchange) GetEasyConvertCurrencyList(ctx context.Context, arg *EasyConvertCurrencyListRequest) (*EasyConvertCurrencyListResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	params := url.Values{}
	if arg.Source != "" {
		params.Set("source", arg.Source)
	}
	var resp *EasyConvertCurrencyListResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getEasyConvertCurrencyListEPL, http.MethodGet, common.EncodeURLValues("trade/easy-convert-currency-list", params), nil, &resp, request.AuthenticatedRequest)
}

// PlaceEasyConvert converts up to 5 small currencies to a mainstream currency
func (e *Exchange) PlaceEasyConvert(ctx context.Context, arg *PlaceEasyConvertRequest) ([]EasyConvertResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if len(arg.FromCurrencies) == 0 {
		return nil, fmt.Errorf("%w, missing from currencies", currency.ErrCurrencyCodeEmpty)
	}
	if len(arg.FromCurrencies) > maxEasyConvertCurrencies {
		return nil, fmt.Errorf("%w, cannot convert more than %d currencies", errExceedLimit, maxEasyConvertCurrencies)
	}
	for x := range arg.FromCurrencies {
		if arg.FromCurrencies[x].IsEmpty() {
			return nil, fmt.Errorf("%w, empty from currency", currency.ErrCurrencyCodeEmpty)
		}
	}
	if arg.ToCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, missing to currency", currency.ErrCurrencyCodeEmpty)
	}
	var resp []EasyConvertResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, placeEasyConvertEPL, http.MethodPost, "trade/easy-convert", arg, &resp, request.AuthenticatedRequest)
}

// GetEasyConvertHistory retrieves the easy convert trades of the past 7 days
func (e *Exchange) GetEasyConvertHistory(ctx context.Context, arg *EasyConvertHistoryRequest) ([]EasyConvertResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	params, err := timePageParams(arg.Before, arg.After, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []EasyConvertResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getEasyConvertHistoryEPL, http.MethodGet, common.EncodeURLValues("trade/easy-convert-history", params), nil, &resp, request.AuthenticatedRequest)
}

// timePageParams returns the query of a listing paged by Unix millisecond before and after cursors; records newer than
// before and older than after are returned, so before must not be later than after
func timePageParams(before, after time.Time, limit uint64) (url.Values, error) {
	if !before.IsZero() && !after.IsZero() && before.After(after) {
		return nil, fmt.Errorf("%w: before %s is after %s", common.ErrStartAfterEnd, before, after)
	}
	params := url.Values{}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}

// GetOneClickRepayCurrencyList retrieves the cross and isolated debt currencies and the currencies that can repay them
func (e *Exchange) GetOneClickRepayCurrencyList(ctx context.Context, arg *OneClickRepayCurrencyListRequest) ([]OneClickRepayCurrencyListResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	params := url.Values{}
	if arg.DebtType != "" {
		params.Set("debtType", arg.DebtType)
	}
	var resp []OneClickRepayCurrencyListResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, oneClickRepayCurrencyListEPL, http.MethodGet, common.EncodeURLValues("trade/one-click-repay-currency-list", params), nil, &resp, request.AuthenticatedRequest)
}

// TradeOneClickRepay repays up to 5 cross debt currencies with a repay currency, up to the available balance of the
// funding and trading accounts
func (e *Exchange) TradeOneClickRepay(ctx context.Context, arg *TradeOneClickRepayRequest) ([]OneClickRepayResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if len(arg.DebtCurrencies) == 0 {
		return nil, fmt.Errorf("%w, missing debt currencies", currency.ErrCurrencyCodeEmpty)
	}
	if len(arg.DebtCurrencies) > maxOneClickRepayCurrencies {
		return nil, fmt.Errorf("%w, cannot repay more than %d debt currencies", errExceedLimit, maxOneClickRepayCurrencies)
	}
	for x := range arg.DebtCurrencies {
		if arg.DebtCurrencies[x].IsEmpty() {
			return nil, fmt.Errorf("%w, empty debt currency", currency.ErrCurrencyCodeEmpty)
		}
	}
	if arg.RepayCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, missing repay currency", currency.ErrCurrencyCodeEmpty)
	}
	var resp []OneClickRepayResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, tradeOneClickRepayEPL, http.MethodPost, "trade/one-click-repay", arg, &resp, request.AuthenticatedRequest)
}

// GetOneClickRepayHistory retrieves the one-click repay trades of the past 7 days
func (e *Exchange) GetOneClickRepayHistory(ctx context.Context, arg *OneClickRepayHistoryRequest) ([]OneClickRepayResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	params, err := timePageParams(arg.Before, arg.After, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []OneClickRepayResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOneClickRepayHistoryEPL, http.MethodGet, common.EncodeURLValues("trade/one-click-repay-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetOneClickRepayCurrencyListV2 retrieves the debt currencies and the currencies that can repay them through the new
// one-click repay
func (e *Exchange) GetOneClickRepayCurrencyListV2(ctx context.Context) (*OneClickRepayCurrencyListV2Response, error) {
	var resp *OneClickRepayCurrencyListV2Response
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOneClickRepayCurrencyListV2EPL, http.MethodGet, "trade/one-click-repay-currency-list-v2", nil, &resp, request.AuthenticatedRequest)
}

// TradeOneClickRepayV2 repays a debt currency with the repay currencies in priority order through the new one-click
// repay
func (e *Exchange) TradeOneClickRepayV2(ctx context.Context, arg *TradeOneClickRepayV2Request) (*TradeOneClickRepayV2Response, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.DebtCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, missing debt currency", currency.ErrCurrencyCodeEmpty)
	}
	if len(arg.RepayCurrencies) == 0 {
		return nil, fmt.Errorf("%w, missing repay currencies", currency.ErrCurrencyCodeEmpty)
	}
	for x := range arg.RepayCurrencies {
		if arg.RepayCurrencies[x].IsEmpty() {
			return nil, fmt.Errorf("%w, empty repay currency", currency.ErrCurrencyCodeEmpty)
		}
	}
	var resp *TradeOneClickRepayV2Response
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, tradeOneClickRepayV2EPL, http.MethodPost, "trade/one-click-repay-v2", arg, &resp, request.AuthenticatedRequest)
}

// GetOneClickRepayHistoryV2 retrieves the new one-click repay trades of the past 7 days
func (e *Exchange) GetOneClickRepayHistoryV2(ctx context.Context, arg *OneClickRepayHistoryRequest) ([]OneClickRepayHistoryV2Response, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	params, err := timePageParams(arg.Before, arg.After, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []OneClickRepayHistoryV2Response
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOneClickRepayHistoryV2EPL, http.MethodGet, common.EncodeURLValues("trade/one-click-repay-history-v2", params), nil, &resp, request.AuthenticatedRequest)
}

// CancelAllMMPOrders cancels all the MMP pending orders of an instrument family; only applicable to options in
// Portfolio Margin mode with the MMP privilege
func (e *Exchange) CancelAllMMPOrders(ctx context.Context, arg *MassCancelOrderRequest) (*MassCancelOrderResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	if arg.InstrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	if arg.LockInterval > maxMassCancelLockInterval {
		return nil, fmt.Errorf("%w: %d, the range is 0 to %d milliseconds", errInvalidLockInterval, arg.LockInterval, maxMassCancelLockInterval)
	}
	var resp *MassCancelOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, massCancelMMPOrderEPL, http.MethodPost, "trade/mass-cancel", arg, &resp, request.AuthenticatedRequest)
}

// CancelAllDelayed cancels all pending orders, spread orders aside, once the countdown times out; a zero countdown
// disables it
func (e *Exchange) CancelAllDelayed(ctx context.Context, arg *CancelAllAfterRequest) (*CancelAllAfterResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.TimeOut != 0 && (arg.TimeOut < 10 || arg.TimeOut > 120) {
		return nil, fmt.Errorf("%w, range of value can be 0, [10, 120]", errCountdownTimeoutRequired)
	}
	var resp *CancelAllAfterResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllAfterCountdownEPL, http.MethodPost, "trade/cancel-all-after", arg, &resp, request.AuthenticatedRequest)
}

// GetTradeAccountRateLimit retrieves the account's order rate limit, which counts new and amended orders, each order
// of a batch individually
func (e *Exchange) GetTradeAccountRateLimit(ctx context.Context) (*AccountRateLimitResponse, error) {
	var resp *AccountRateLimitResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getTradeAccountRateLimitEPL, http.MethodGet, "trade/account-rate-limit", nil, &resp, request.AuthenticatedRequest)
}

// PreCheckOrder returns the account information before and after placing a potential order; only applicable to
// Multi-currency margin and Portfolio margin modes
func (e *Exchange) PreCheckOrder(ctx context.Context, arg *OrderPreCheckRequest) (*OrderPreCheckResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%T: %w", arg, common.ErrNilPointer)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.TradeMode == "" {
		return nil, errInvalidTradeModeValue
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Size <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *OrderPreCheckResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, orderPreCheckEPL, http.MethodPost, "trade/order-precheck", arg, &resp, request.AuthenticatedRequest)
}
