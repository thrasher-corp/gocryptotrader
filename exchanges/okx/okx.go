package okx

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Okx is the overarching type across this package
type Okx struct {
	exchange.Base

	messageIDSeq           common.Counter
	instrumentsInfoMapLock sync.Mutex
	instrumentsInfoMap     map[string][]Instrument
}

const (
	baseURL = "https://www.okx.com/"
	apiURL  = baseURL + apiPath

	apiPath      = "api/v5/"
	websocketURL = "wss://ws.okx.com:8443/ws/v5/"

	apiWebsocketPublicURL  = websocketURL + "public"
	apiWebsocketPrivateURL = websocketURL + "private"
)

/************************************ MarketData Endpoints *************************************************/

// PlaceOrder places an order
func (ok *Okx) PlaceOrder(ctx context.Context, arg *PlaceOrderRequestParam) (*OrderData, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	var resp *OrderData
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, placeOrderEPL, http.MethodPost, "trade/order", &arg, &resp, request.AuthenticatedRequest)
	if err != nil {
		if resp != nil && resp.StatusMessage != "" {
			return nil, fmt.Errorf("%w; %w", err, getStatusError(resp.StatusCode, resp.StatusMessage))
		}
		return nil, err
	}
	return resp, nil
}

// PlaceMultipleOrders  to place orders in batches. Maximum 20 orders can be placed at a time. Request parameters should be passed in the form of an array
func (ok *Okx) PlaceMultipleOrders(ctx context.Context, args []PlaceOrderRequestParam) ([]OrderData, error) {
	if len(args) == 0 {
		return nil, order.ErrSubmissionIsNil
	}
	for x := range args {
		if err := args[x].Validate(); err != nil {
			return nil, err
		}
	}
	var resp []OrderData
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, placeMultipleOrdersEPL, http.MethodPost, "trade/batch-orders", &args, &resp, request.AuthenticatedRequest)
	if err != nil {
		if len(resp) == 0 {
			return nil, err
		}
		var errs error
		for x := range resp {
			errs = common.AppendError(errs, getStatusError(resp[x].StatusCode, resp[x].StatusMessage))
		}
		return nil, common.AppendError(err, errs)
	}
	return resp, nil
}

// CancelSingleOrder cancel an incomplete order
func (ok *Okx) CancelSingleOrder(ctx context.Context, arg *CancelOrderRequestParam) (*OrderData, error) {
	if *arg == (CancelOrderRequestParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OrderData
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelOrderEPL, http.MethodPost, "trade/cancel-order", &arg, &resp, request.AuthenticatedRequest)
	if err != nil {
		if resp != nil && resp.StatusMessage != "" {
			return nil, fmt.Errorf("%w; %w", err, getStatusError(resp.StatusCode, resp.StatusMessage))
		}
		return nil, err
	}
	return resp, nil
}

// CancelMultipleOrders cancel incomplete orders in batches. Maximum 20 orders can be canceled at a time.
// Request parameters should be passed in the form of an array
func (ok *Okx) CancelMultipleOrders(ctx context.Context, args []CancelOrderRequestParam) ([]*OrderData, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for x := range args {
		arg := args[x]
		if arg.InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if arg.OrderID == "" && arg.ClientOrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
	}
	var resp []*OrderData
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelMultipleOrdersEPL, http.MethodPost, "trade/cancel-batch-orders", args, &resp, request.AuthenticatedRequest)
	if err != nil {
		if len(resp) == 0 {
			return nil, err
		}
		var errs error
		for x := range resp {
			if resp[x].StatusCode != 0 {
				errs = common.AppendError(errs, getStatusError(resp[x].StatusCode, resp[x].StatusMessage))
			}
		}
		return nil, common.AppendError(err, errs)
	}
	return resp, nil
}

// AmendOrder an incomplete order
func (ok *Okx) AmendOrder(ctx context.Context, arg *AmendOrderRequestParams) (*OrderData, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.ClientOrderID == "" && arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.NewQuantity <= 0 && arg.NewPrice <= 0 {
		return nil, errInvalidNewSizeOrPriceInformation
	}
	var resp *OrderData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendOrderEPL, http.MethodPost, "trade/amend-order", arg, &resp, request.AuthenticatedRequest)
}

// AmendMultipleOrders amend incomplete orders in batches. Maximum 20 orders can be amended at a time. Request parameters should be passed in the form of an array
func (ok *Okx) AmendMultipleOrders(ctx context.Context, args []AmendOrderRequestParams) ([]OrderData, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for x := range args {
		if args[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[x].ClientOrderID == "" && args[x].OrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
		if args[x].NewQuantity <= 0 && args[x].NewPrice <= 0 {
			return nil, errInvalidNewSizeOrPriceInformation
		}
	}
	var resp []OrderData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendMultipleOrdersEPL, http.MethodPost, "trade/amend-batch-orders", &args, &resp, request.AuthenticatedRequest)
}

// ClosePositions close all positions of an instrument via a market order
func (ok *Okx) ClosePositions(ctx context.Context, arg *ClosePositionsRequestParams) (*ClosePositionResponse, error) {
	if *arg == (ClosePositionsRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	switch arg.MarginMode {
	case "", TradeModeCross, TradeModeIsolated:
	default:
		return nil, margin.ErrMarginTypeUnsupported
	}
	var resp *ClosePositionResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, closePositionEPL, http.MethodPost, "trade/close-position", arg, &resp, request.AuthenticatedRequest)
}

// GetOrderDetail retrieves order details given instrument id and order identification
func (ok *Okx) GetOrderDetail(ctx context.Context, arg *OrderDetailRequestParam) (*OrderDetail, error) {
	if *arg == (OrderDetailRequestParam{}) {
		return nil, common.ErrEmptyParams
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
	var resp *OrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOrderDetEPL, http.MethodGet, common.EncodeURLValues("trade/order", params), nil, &resp, request.AuthenticatedRequest)
}

// GetOrderList retrieves all incomplete orders under the current account
func (ok *Okx) GetOrderList(ctx context.Context, arg *OrderListRequestParams) ([]OrderDetail, error) {
	if *arg == (OrderListRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", arg.InstrumentType)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.OrderType != "" {
		params.Set("orderType", strings.ToLower(arg.OrderType))
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp []OrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOrderListEPL, http.MethodGet, common.EncodeURLValues("trade/orders-pending", params), nil, &resp, request.AuthenticatedRequest)
}

// Get7DayOrderHistory retrieves the completed order data for the last 7 days, and the incomplete orders that have been cancelled are only reserved for 2 hours
func (ok *Okx) Get7DayOrderHistory(ctx context.Context, arg *OrderHistoryRequestParams) ([]OrderDetail, error) {
	return ok.getOrderHistory(ctx, arg, "trade/orders-history", getOrderHistory7DaysEPL)
}

// Get3MonthOrderHistory retrieves the completed order data for the last 7 days, and the incomplete orders that have been cancelled are only reserved for 2 hours
func (ok *Okx) Get3MonthOrderHistory(ctx context.Context, arg *OrderHistoryRequestParams) ([]OrderDetail, error) {
	return ok.getOrderHistory(ctx, arg, "trade/orders-history-archive", getOrderHistory3MonthsEPL)
}

// getOrderHistory retrieves the order history of the past limited times
func (ok *Okx) getOrderHistory(ctx context.Context, arg *OrderHistoryRequestParams, route string, rateLimit request.EndpointLimit) ([]OrderDetail, error) {
	if *arg == (OrderHistoryRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(arg.InstrumentType))
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.OrderType != "" {
		params.Set("orderType", strings.ToLower(arg.OrderType))
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if !arg.Start.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.Start.UnixMilli(), 10))
	}
	if !arg.End.IsZero() {
		params.Set("end", strconv.FormatInt(arg.End.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	if arg.Category != "" {
		params.Set("category", strings.ToLower(arg.Category))
	}
	var resp []OrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rateLimit, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, request.AuthenticatedRequest)
}

// GetTransactionDetailsLast3Days retrieves recently-filled transaction details in the last 3 day
func (ok *Okx) GetTransactionDetailsLast3Days(ctx context.Context, arg *TransactionDetailRequestParams) ([]TransactionDetail, error) {
	return ok.getTransactionDetails(ctx, arg, "trade/fills", getTransactionDetail3DaysEPL)
}

// GetTransactionDetailsLast3Months retrieve recently-filled transaction details in the last 3 months
func (ok *Okx) GetTransactionDetailsLast3Months(ctx context.Context, arg *TransactionDetailRequestParams) ([]TransactionDetail, error) {
	return ok.getTransactionDetails(ctx, arg, "trade/fills-history", getTransactionDetail3MonthsEPL)
}

// GetTransactionDetails retrieves recently-filled transaction details
func (ok *Okx) getTransactionDetails(ctx context.Context, arg *TransactionDetailRequestParams, route string, rateLimit request.EndpointLimit) ([]TransactionDetail, error) {
	if *arg == (TransactionDetailRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	arg.InstrumentType = strings.ToUpper(arg.InstrumentType)
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if !arg.Begin.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.Begin.UnixMilli(), 10))
	}
	if !arg.End.IsZero() {
		params.Set("end", strconv.FormatInt(arg.End.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	var resp []TransactionDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rateLimit, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, request.AuthenticatedRequest)
}

// PlaceAlgoOrder order includes trigger, oco, chase, conditional, iceberg, twap and trailing orders.
// chase order only applicable to futures and swap orders
func (ok *Okx) PlaceAlgoOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if *arg == (AlgoOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	arg.TradeMode = strings.ToLower(arg.TradeMode)
	switch arg.TradeMode {
	case TradeModeCross, TradeModeIsolated, TradeModeCash:
	default:
		return nil, errInvalidTradeModeValue
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Side != order.Buy.Lower() &&
		arg.Side != order.Sell.Lower() {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Size <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp *AlgoOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeAlgoOrderEPL, http.MethodPost, "trade/order-algo", arg, &resp, request.AuthenticatedRequest)
}

// PlaceStopOrder places a stop order.
// The order type should be "conditional" because stop orders are used for conditional take-profit or stop-loss scenarios.
func (ok *Okx) PlaceStopOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if *arg == (AlgoOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.TakeProfitTriggerPrice <= 0 {
		return nil, fmt.Errorf("%w, take profit trigger price is required", order.ErrPriceBelowMin)
	}
	if arg.TakeProfitTriggerPriceType == "" {
		return nil, order.ErrUnknownPriceType
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// PlaceTrailingStopOrder to place trailing stop order
func (ok *Okx) PlaceTrailingStopOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if *arg == (AlgoOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.OrderType != "move_order_stop" {
		return nil, fmt.Errorf("%w: order type value 'move_order_stop' is only supported for move_order_stop orders", order.ErrTypeIsInvalid)
	}
	if arg.CallbackRatio == 0 && arg.CallbackSpreadVariance == 0 {
		return nil, fmt.Errorf(" %w \"callbackRatio\" or \"callbackSpread\" required", errPriceTrackingNotSet)
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// PlaceIcebergOrder to place iceberg algo order
func (ok *Okx) PlaceIcebergOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if *arg == (AlgoOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.OrderType != "iceberg" {
		return nil, fmt.Errorf("%w: order type value 'iceberg' is only supported for iceberg orders", order.ErrTypeIsInvalid)
	}
	if arg.SizeLimit <= 0 {
		return nil, errMissingSizeLimit
	}
	if arg.LimitPrice <= 0 {
		return nil, errInvalidPriceLimit
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// PlaceTWAPOrder to place TWAP algo orders
func (ok *Okx) PlaceTWAPOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if *arg == (AlgoOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.OrderType != "twap" {
		return nil, fmt.Errorf("%w: order type value 'twap' is only supported for twap orders", order.ErrTypeIsInvalid)
	}
	if arg.SizeLimit <= 0 {
		return nil, errMissingSizeLimit
	}
	if arg.LimitPrice <= 0 {
		return nil, errInvalidPriceLimit
	}
	if IntervalFromString(arg.TimeInterval, true) == "" {
		return nil, errMissingIntervalValue
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// PlaceTakeProfitStopLossOrder places conditional and oco orders
// When placing net TP/SL order (ordType=conditional) and both take-profit and stop-loss parameters are sent,
// only stop-loss logic will be performed and take-profit logic will be ignored
func (ok *Okx) PlaceTakeProfitStopLossOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if *arg == (AlgoOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.OrderType != orderConditional {
		return nil, fmt.Errorf("%w for TPSL: %q", order.ErrTypeIsInvalid, arg.OrderType)
	}
	if arg.StopLossTriggerPrice <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	switch arg.StopLossTriggerPriceType {
	case "", "last", "index", "mark":
	default:
		return nil, fmt.Errorf("%w, only 'last', 'index', and 'mark' are supported", order.ErrUnknownPriceType)
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// PlaceChaseAlgoOrder places an order that adjusts the price of an open limit order to match the current market price
func (ok *Okx) PlaceChaseAlgoOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if *arg == (AlgoOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.OrderType != orderChase {
		return nil, fmt.Errorf("%w: order type value 'chase' is only supported for chase orders", order.ErrTypeIsInvalid)
	}
	if (arg.MaxChaseType == "" || arg.MaxChaseValue == 0) &&
		(arg.MaxChaseType != "" || arg.MaxChaseValue != 0) {
		return nil, fmt.Errorf("%w, either non or both MaxChaseType and MaxChaseValue has to be provided", errPriceTrackingNotSet)
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// PlaceTriggerAlgoOrder fetches algo trigger orders for SWAP market types
func (ok *Okx) PlaceTriggerAlgoOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if *arg == (AlgoOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.OrderType != orderTrigger {
		return nil, fmt.Errorf("%w for Trigger: %q", order.ErrTypeIsInvalid, arg.OrderType)
	}
	if arg.TriggerPrice <= 0 {
		return nil, fmt.Errorf("%w, trigger price must be greater than 0", order.ErrPriceBelowMin)
	}
	switch arg.TriggerPriceType {
	case "", "last", "index", "mark":
	default:
		return nil, fmt.Errorf("%w, only last, index and mark trigger price types are allowed", order.ErrUnknownPriceType)
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// CancelAdvanceAlgoOrder Cancel unfilled algo orders
// A maximum of 10 orders can be canceled at a time.
// Request parameters should be passed in the form of an array
func (ok *Okx) CancelAdvanceAlgoOrder(ctx context.Context, args []AlgoOrderCancelParams) (*AlgoOrder, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	return ok.cancelAlgoOrder(ctx, args, "trade/cancel-advance-algos", cancelAdvanceAlgoOrderEPL)
}

// CancelAlgoOrder to cancel unfilled algo orders (not including Iceberg order, TWAP order, Trailing Stop order).
// A maximum of 10 orders can be canceled at a time.
// Request parameters should be passed in the form of an array
func (ok *Okx) CancelAlgoOrder(ctx context.Context, args []AlgoOrderCancelParams) (*AlgoOrder, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	return ok.cancelAlgoOrder(ctx, args, "trade/cancel-algos", cancelAlgoOrderEPL)
}

// cancelAlgoOrder to cancel unfilled algo orders
func (ok *Okx) cancelAlgoOrder(ctx context.Context, args []AlgoOrderCancelParams, route string, rateLimit request.EndpointLimit) (*AlgoOrder, error) {
	for x := range args {
		if args[x] == (AlgoOrderCancelParams{}) {
			return nil, common.ErrEmptyParams
		}
		if args[x].AlgoOrderID == "" {
			return nil, fmt.Errorf("%w, AlgoOrderID is required", order.ErrOrderIDNotSet)
		} else if args[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
	}
	var resp *AlgoOrder
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, rateLimit, http.MethodPost, route, &args, &resp, request.AuthenticatedRequest)
	if err != nil {
		if resp != nil && resp.StatusMessage != "" {
			return nil, fmt.Errorf("%w; %w", err, getStatusError(resp.StatusCode, resp.StatusMessage))
		}
		return nil, err
	}
	return resp, nil
}

// AmendAlgoOrder amend unfilled algo orders (Support stop order only, not including Move_order_stop order, Trigger order, Iceberg order, TWAP order, Trailing Stop order).
// Only applicable to Futures and Perpetual swap
func (ok *Okx) AmendAlgoOrder(ctx context.Context, arg *AmendAlgoOrderParam) (*AmendAlgoResponse, error) {
	if arg == nil {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.AlgoID == "" && arg.ClientSuppliedAlgoOrderID == "" {
		return nil, fmt.Errorf("%w either AlgoID or ClientSuppliedAlgoOrderID is required", order.ErrOrderIDNotSet)
	}
	var resp *AmendAlgoResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendAlgoOrderEPL, http.MethodPost, "trade/amend-algos", arg, &resp, request.AuthenticatedRequest)
}

// GetAlgoOrderDetail retrieves algo order details
func (ok *Okx) GetAlgoOrderDetail(ctx context.Context, algoID, clientSuppliedAlgoID string) (*AlgoOrderDetail, error) {
	if algoID == "" && clientSuppliedAlgoID == "" {
		return nil, fmt.Errorf("%w either AlgoID or ClientSuppliedAlgoID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	params.Set("algoClOrdId", clientSuppliedAlgoID)
	var resp *AlgoOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderDetailEPL, http.MethodGet, common.EncodeURLValues("trade/order-algo", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAlgoOrderList retrieves a list of untriggered Algo orders under the current account
func (ok *Okx) GetAlgoOrderList(ctx context.Context, orderType, algoOrderID, clientOrderID, instrumentType, instrumentID string, after, before time.Time, limit int64) ([]AlgoOrderResponse, error) {
	orderType = strings.ToLower(orderType)
	if orderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	params := url.Values{}
	params.Set("ordType", orderType)
	if algoOrderID != "" {
		params.Set("algoId", algoOrderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if !before.IsZero() && before.Before(time.Now()) {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() && after.Before(time.Now()) {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AlgoOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderListEPL, http.MethodGet, common.EncodeURLValues("trade/orders-algo-pending", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAlgoOrderHistory load a list of all algo orders under the current account in the last 3 months
func (ok *Okx) GetAlgoOrderHistory(ctx context.Context, orderType, state, algoOrderID, instrumentType, instrumentID string, after, before time.Time, limit int64) ([]AlgoOrderResponse, error) {
	if orderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if algoOrderID == "" && !slices.Contains([]string{"effective", "order_failed", "canceled"}, state) {
		return nil, errMissingEitherAlgoIDOrState
	}
	params := url.Values{}
	params.Set("ordType", strings.ToLower(orderType))
	if algoOrderID != "" {
		params.Set("algoId", algoOrderID)
	} else {
		params.Set("state", state)
	}
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if !before.IsZero() && before.Before(time.Now()) {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() && after.Before(time.Now()) {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AlgoOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("trade/orders-algo-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetEasyConvertCurrencyList retrieve list of small convertibles and mainstream currencies. Only applicable to the crypto balance less than $10
func (ok *Okx) GetEasyConvertCurrencyList(ctx context.Context, source string) (*EasyConvertDetail, error) {
	params := url.Values{}
	if source != "" {
		params.Set("source", source)
	}
	var resp *EasyConvertDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEasyConvertCurrencyListEPL, http.MethodGet,
		common.EncodeURLValues("trade/easy-convert-currency-list", params), nil, &resp, request.AuthenticatedRequest)
}

// PlaceEasyConvert converts small currencies to mainstream currencies. Only applicable to the crypto balance less than $10
func (ok *Okx) PlaceEasyConvert(ctx context.Context, arg PlaceEasyConvertParam) ([]EasyConvertItem, error) {
	if len(arg.FromCurrency) == 0 {
		return nil, fmt.Errorf("%w, missing FromCurrency", currency.ErrCurrencyCodeEmpty)
	}
	if arg.ToCurrency == "" {
		return nil, fmt.Errorf("%w, missing ToCurrency", currency.ErrCurrencyCodeEmpty)
	}
	var resp []EasyConvertItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeEasyConvertEPL, http.MethodPost, "trade/easy-convert", &arg, &resp, request.AuthenticatedRequest)
}

// GetEasyConvertHistory retrieves the history and status of easy convert trades
func (ok *Okx) GetEasyConvertHistory(ctx context.Context, after, before time.Time, limit int64) ([]EasyConvertItem, error) {
	params := url.Values{}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.Unix(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []EasyConvertItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEasyConvertHistoryEPL, http.MethodGet,
		common.EncodeURLValues("trade/easy-convert-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetOneClickRepayCurrencyList retrieves list of debt currency data and repay currencies. Debt currencies include both cross and isolated debts.
// debt level "cross", and "isolated" are allowed
func (ok *Okx) GetOneClickRepayCurrencyList(ctx context.Context, debtType string) ([]CurrencyOneClickRepay, error) {
	params := url.Values{}
	if debtType != "" {
		params.Set("debtType", debtType)
	}
	var resp []CurrencyOneClickRepay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, oneClickRepayCurrencyListEPL, http.MethodGet,
		common.EncodeURLValues("trade/one-click-repay-currency-list", params), nil, &resp, request.AuthenticatedRequest)
}

// TradeOneClickRepay trade one-click repay to repay cross debts. Isolated debts are not applicable. The maximum repayment amount is based on the remaining available balance of funding and trading accounts
func (ok *Okx) TradeOneClickRepay(ctx context.Context, arg TradeOneClickRepayParam) ([]CurrencyOneClickRepay, error) {
	if len(arg.DebtCurrency) == 0 {
		return nil, fmt.Errorf("%w, missing 'debtCcy'", currency.ErrCurrencyCodeEmpty)
	}
	if arg.RepayCurrency == "" {
		return nil, fmt.Errorf("%w, missing 'repayCcy'", currency.ErrCurrencyCodeEmpty)
	}
	var resp []CurrencyOneClickRepay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, tradeOneClickRepayEPL, http.MethodPost, "trade/one-click-repay", &arg, &resp, request.AuthenticatedRequest)
}

// GetOneClickRepayHistory get the history and status of one-click repay trades
func (ok *Okx) GetOneClickRepayHistory(ctx context.Context, after, before time.Time, limit int64) ([]CurrencyOneClickRepay, error) {
	params := url.Values{}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.Unix(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CurrencyOneClickRepay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOneClickRepayHistoryEPL, http.MethodGet, common.EncodeURLValues("trade/one-click-repay-history", params), nil, &resp, request.AuthenticatedRequest)
}

// CancelAllMMPOrders cancel all the MMP pending orders of an instrument family.
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required
func (ok *Okx) CancelAllMMPOrders(ctx context.Context, instrumentType, instrumentFamily string, lockInterval int64) (*CancelMMPResponse, error) {
	if instrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	if instrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	if lockInterval < 0 || lockInterval > 10000 {
		return nil, fmt.Errorf("%w, LockInterval value range should be between 0 and 10000", errMissingIntervalValue)
	}
	arg := &struct {
		InstrumentType   string `json:"instType,omitempty"`
		InstrumentFamily string `json:"instFamily,omitempty"`
		LockInterval     int64  `json:"lockInterval,omitempty"`
	}{
		InstrumentType:   instrumentType,
		InstrumentFamily: instrumentFamily,
		LockInterval:     lockInterval,
	}
	var resp *CancelMMPResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, tradeOneClickRepayEPL, http.MethodPost, "trade/mass-cancel", arg, &resp, request.AuthenticatedRequest)
}

// CancelAllDelayed cancel all pending orders after the countdown timeout.
// Applicable to all trading symbols through order book (except Spread trading)
func (ok *Okx) CancelAllDelayed(ctx context.Context, timeout int64, orderTag string) (*CancelResponse, error) {
	if (timeout != 0) && (timeout < 10 || timeout > 120) {
		return nil, fmt.Errorf("%w, Range of value can be 0, [10, 120]", errCountdownTimeoutRequired)
	}
	arg := &struct {
		TimeOut  int64  `json:"timeOut,string"`
		OrderTag string `json:"tag,omitempty"`
	}{
		TimeOut:  timeout,
		OrderTag: orderTag,
	}
	var resp *CancelResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllAfterCountdownEPL, http.MethodPost, "trade/cancel-all-after", arg, &resp, request.AuthenticatedRequest)
}

// GetTradeAccountRateLimit get account rate limit related information.
// Only new order requests and amendment order requests will be counted towards this limit. For batch order requests consisting of multiple orders, each order will be counted individually
func (ok *Okx) GetTradeAccountRateLimit(ctx context.Context) (*AccountRateLimit, error) {
	var resp *AccountRateLimit
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTradeAccountRateLimitEPL, http.MethodGet, "trade/account-rate-limit", nil, &resp, request.AuthenticatedRequest)
}

// PreCheckOrder returns the account information before and after placing a potential order
// Only applicable to Multi-currency margin mode, and Portfolio margin mode
func (ok *Okx) PreCheckOrder(ctx context.Context, arg *OrderPreCheckParams) (*OrderPreCheckResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
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
		return nil, order.ErrAmountBelowMin
	}
	var resp *OrderPreCheckResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, orderPreCheckEPL, http.MethodPost, "trade/order-precheck", arg, &resp, request.AuthenticatedRequest)
}

/*************************************** Block trading ********************************/

// GetCounterparties retrieves the list of counterparties that the user has permissions to trade with
func (ok *Okx) GetCounterparties(ctx context.Context) ([]CounterpartiesResponse, error) {
	var resp []CounterpartiesResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getCounterpartiesEPL, http.MethodGet, "rfq/counterparties", nil, &resp, request.AuthenticatedRequest)
}

// CreateRFQ Creates a new RFQ
func (ok *Okx) CreateRFQ(ctx context.Context, arg *CreateRFQInput) (*RFQResponse, error) {
	if len(arg.CounterParties) == 0 {
		return nil, errInvalidCounterParties
	}
	if len(arg.Legs) == 0 {
		return nil, errMissingLegs
	}
	var resp *RFQResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, createRFQEPL, http.MethodPost, "rfq/create-rfq", &arg, &resp, request.AuthenticatedRequest)
}

// CancelRFQ cancels a request for quotation
func (ok *Okx) CancelRFQ(ctx context.Context, rfqID, clientRFQID string) (*CancelRFQResponse, error) {
	if rfqID == "" && clientRFQID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CancelRFQResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelRFQEPL, http.MethodPost, "rfq/cancel-rfq", &CancelRFQRequestParam{
		RFQID:       rfqID,
		ClientRFQID: clientRFQID,
	}, &resp, request.AuthenticatedRequest)
}

// CancelMultipleRFQs cancel multiple active RFQs in a single batch. Maximum 100 RFQ orders can be canceled at a time
func (ok *Okx) CancelMultipleRFQs(ctx context.Context, arg *CancelRFQRequestsParam) ([]CancelRFQResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.RFQIDs) == 0 && len(arg.ClientRFQIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	} else if len(arg.RFQIDs)+len(arg.ClientRFQIDs) > 100 {
		return nil, errMaxRFQOrdersToCancel
	}
	var resp []CancelRFQResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelMultipleRFQEPL, http.MethodPost, "rfq/cancel-batch-rfqs", &arg, &resp, request.AuthenticatedRequest)
}

// CancelAllRFQs cancels all active RFQs
func (ok *Okx) CancelAllRFQs(ctx context.Context) (types.Time, error) {
	resp := &tsResp{}
	return resp.Timestamp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllRFQsEPL, http.MethodPost, "rfq/cancel-all-rfqs", nil, resp, request.AuthenticatedRequest)
}

// ExecuteQuote executes a Quote. It is only used by the creator of the RFQ
func (ok *Okx) ExecuteQuote(ctx context.Context, rfqID, quoteID string) (*ExecuteQuoteResponse, error) {
	if rfqID == "" || quoteID == "" {
		return nil, errMissingRFQIDOrQuoteID
	}
	var resp *ExecuteQuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, executeQuoteEPL, http.MethodPost, "rfq/execute-quote", &ExecuteQuoteParams{
		RFQID:   rfqID,
		QuoteID: quoteID,
	}, &resp, request.AuthenticatedRequest)
}

// GetQuoteProducts retrieve the products which makers want to quote and receive RFQs for, and the corresponding price and size limit
func (ok *Okx) GetQuoteProducts(ctx context.Context) ([]QuoteProduct, error) {
	var resp []QuoteProduct
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getQuoteProductsEPL, http.MethodGet, "rfq/maker-instrument-settings", nil, &resp, request.AuthenticatedRequest)
}

// SetQuoteProducts customize the products which makers want to quote and receive RFQs for, and the corresponding price and size limit
func (ok *Okx) SetQuoteProducts(ctx context.Context, args []SetQuoteProductParam) (*SetQuoteProductsResult, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for x := range args {
		args[x].InstrumentType = strings.ToUpper(args[x].InstrumentType)
		if !slices.Contains([]string{instTypeSwap, instTypeSpot, instTypeFutures, instTypeOption}, args[x].InstrumentType) {
			return nil, fmt.Errorf("%w received %v", errInvalidInstrumentType, args[x].InstrumentType)
		}
		if len(args[x].Data) == 0 {
			return nil, errMissingMakerInstrumentSettings
		}
		for y := range args[x].Data {
			if slices.Contains([]string{instTypeSwap, instTypeFutures, instTypeOption}, args[x].InstrumentType) && args[x].Data[y].Underlying == "" {
				return nil, fmt.Errorf("%w, for instrument type %s and %s", errInvalidUnderlying, args[x].InstrumentType, args[x].Data[x].Underlying)
			}
			if (args[x].InstrumentType == instTypeSpot) && args[x].Data[x].InstrumentID == "" {
				return nil, fmt.Errorf("%w, for instrument type %s and %s", errMissingInstrumentID, args[x].InstrumentType, args[x].Data[x].InstrumentID)
			}
		}
	}
	var resp *SetQuoteProductsResult
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setQuoteProductsEPL, http.MethodPost, "rfq/maker-instrument-settings", &args, &resp, request.AuthenticatedRequest)
}

// ResetRFQMMPStatus reset the MMP status to be inactive
func (ok *Okx) ResetRFQMMPStatus(ctx context.Context) (types.Time, error) {
	resp := &tsResp{}
	return resp.Timestamp, ok.SendHTTPRequest(ctx, exchange.RestSpot, resetRFQMMPEPL, http.MethodPost, "rfq/mmp-reset", nil, resp, request.AuthenticatedRequest)
}

// CreateQuote allows the user to Quote an RFQ that they are a counterparty to. The user MUST quote
// the entire RFQ and not part of the legs or part of the quantity. Partial quoting or partial fills are not allowed
func (ok *Okx) CreateQuote(ctx context.Context, arg *CreateQuoteParams) (*QuoteResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	arg.QuoteSide = strings.ToLower(arg.QuoteSide)
	switch {
	case arg.RFQID == "":
		return nil, errMissingRFQID
	case arg.QuoteSide != order.Buy.Lower() && arg.QuoteSide != order.Sell.Lower():
		return nil, order.ErrSideIsInvalid
	case len(arg.Legs) == 0:
		return nil, errMissingLegs
	}
	for x := range arg.Legs {
		switch {
		case arg.Legs[x].InstrumentID == "":
			return nil, errMissingInstrumentID
		case arg.Legs[x].SizeOfQuoteLeg <= 0:
			return nil, errMissingSizeOfQuote
		case arg.Legs[x].Price <= 0:
			return nil, errMissingLegsQuotePrice
		case arg.Legs[x].Side == order.UnknownSide:
			return nil, order.ErrSideIsInvalid
		}
	}
	var resp *QuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, createQuoteEPL, http.MethodPost, "rfq/create-quote", &arg, &resp, request.AuthenticatedRequest)
}

// CancelQuote cancels an existing active quote you have created in response to an RFQ
func (ok *Okx) CancelQuote(ctx context.Context, quoteID, clientQuoteID string) (*CancelQuoteResponse, error) {
	if clientQuoteID == "" && quoteID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CancelQuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelQuoteEPL, http.MethodPost, "rfq/cancel-quote", &CancelQuoteRequestParams{
		QuoteID:       quoteID,
		ClientQuoteID: clientQuoteID,
	}, &resp, request.AuthenticatedRequest)
}

// CancelMultipleQuote cancels multiple active quotes in a single batch, with a maximum of 100 quote orders cancellable at once
func (ok *Okx) CancelMultipleQuote(ctx context.Context, arg CancelQuotesRequestParams) ([]CancelQuoteResponse, error) {
	if len(arg.QuoteIDs) == 0 && len(arg.ClientQuoteIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []CancelQuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelMultipleQuotesEPL, http.MethodPost, "rfq/cancel-batch-quotes", &arg, &resp, request.AuthenticatedRequest)
}

// CancelAllRFQQuotes cancels all active quote orders
func (ok *Okx) CancelAllRFQQuotes(ctx context.Context) (types.Time, error) {
	resp := &tsResp{}
	return resp.Timestamp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllQuotesEPL, http.MethodPost, "rfq/cancel-all-quotes", nil, resp, request.AuthenticatedRequest)
}

// GetRFQs retrieves details of RFQs where the user is a counterparty, either as the creator or the recipient
func (ok *Okx) GetRFQs(ctx context.Context, arg *RFQRequestParams) ([]RFQResponse, error) {
	if *arg == (RFQRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	params := url.Values{}
	if arg.RFQID != "" {
		params.Set("rfqId", arg.RFQID)
	}
	if arg.ClientRFQID != "" {
		params.Set("clRFQId", arg.ClientRFQID)
	}
	if arg.State != "" {
		params.Set("state", strings.ToLower(arg.State))
	}
	if arg.BeginningID != "" {
		params.Set("beginId", arg.BeginningID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp []RFQResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getRFQsEPL, http.MethodGet, common.EncodeURLValues("rfq/rfqs", params), nil, &resp, request.AuthenticatedRequest)
}

// GetQuotes retrieves all Quotes where the user is a counterparty, either as the creator or the receiver
func (ok *Okx) GetQuotes(ctx context.Context, arg *QuoteRequestParams) ([]QuoteResponse, error) {
	if *arg == (QuoteRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	params := url.Values{}
	if arg.RFQID != "" {
		params.Set("rfqId", arg.RFQID)
	}
	if arg.ClientRFQID != "" {
		params.Set("clRFQId", arg.ClientRFQID)
	}
	if arg.QuoteID != "" {
		params.Set("quoteId", arg.QuoteID)
	}
	if arg.ClientQuoteID != "" {
		params.Set("clQuoteId", arg.ClientQuoteID)
	}
	if arg.State != "" {
		params.Set("state", strings.ToLower(arg.State))
	}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp []QuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getQuotesEPL, http.MethodGet, common.EncodeURLValues("rfq/quotes", params), nil, &resp, request.AuthenticatedRequest)
}

// GetRFQTrades retrieves executed trades where the user is a counterparty, either as the creator or the receiver
func (ok *Okx) GetRFQTrades(ctx context.Context, arg *RFQTradesRequestParams) ([]RFQTradeResponse, error) {
	if *arg == (RFQTradesRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	params := url.Values{}
	if arg.RFQID != "" {
		params.Set("rfqId", arg.RFQID)
	}
	if arg.ClientRFQID != "" {
		params.Set("clRFQId", arg.ClientRFQID)
	}
	if arg.QuoteID != "" {
		params.Set("quoteId", arg.QuoteID)
	}
	if arg.ClientQuoteID != "" {
		params.Set("clQuoteId", arg.ClientQuoteID)
	}
	if arg.State != "" {
		params.Set("state", strings.ToLower(arg.State))
	}
	if arg.BlockTradeID != "" {
		params.Set("blockTdId", arg.BlockTradeID)
	}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp []RFQTradeResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTradesEPL, http.MethodGet, common.EncodeURLValues("rfq/trades", params), nil, &resp, request.AuthenticatedRequest)
}

// GetPublicRFQTrades retrieves recent executed block trades
func (ok *Okx) GetPublicRFQTrades(ctx context.Context, beginID, endID string, limit int64) ([]PublicTradesResponse, error) {
	params := url.Values{}
	if beginID != "" {
		params.Set("beginId", beginID)
	}
	if endID != "" {
		params.Set("endId", endID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PublicTradesResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPublicTradesEPL, http.MethodGet, common.EncodeURLValues("rfq/public-trades", params), nil, &resp, request.UnauthenticatedRequest)
}

/*************************************** Funding Tradings ********************************/

// GetFundingCurrencies retrieve a list of all currencies
func (ok *Okx) GetFundingCurrencies(ctx context.Context, ccy currency.Code) ([]CurrencyResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []CurrencyResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getCurrenciesEPL, http.MethodGet,
		common.EncodeURLValues("asset/currencies", params), nil, &resp, request.AuthenticatedRequest)
}

// GetBalance retrieves the funding account balances of all the assets and the amount that is available or on hold
func (ok *Okx) GetBalance(ctx context.Context, ccy currency.Code) ([]AssetBalance, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []AssetBalance
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBalanceEPL, http.MethodGet, common.EncodeURLValues("asset/balances", params), nil, &resp, request.AuthenticatedRequest)
}

// GetNonTradableAssets retrieves non tradable assets
func (ok *Okx) GetNonTradableAssets(ctx context.Context, ccy currency.Code) ([]NonTradableAsset, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []NonTradableAsset
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getNonTradableAssetsEPL, http.MethodGet, common.EncodeURLValues("asset/non-tradable-assets", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAccountAssetValuation view account asset valuation
func (ok *Okx) GetAccountAssetValuation(ctx context.Context, ccy currency.Code) ([]AccountAssetValuation, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.Upper().String())
	}
	var resp []AccountAssetValuation
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountAssetValuationEPL, http.MethodGet, common.EncodeURLValues("asset/asset-valuation", params), nil, &resp, request.AuthenticatedRequest)
}

// FundingTransfer transfer of funds between your funding account and trading account,
// and from the master account to sub-accounts
func (ok *Okx) FundingTransfer(ctx context.Context, arg *FundingTransferRequestInput) ([]FundingTransferResponse, error) {
	if *arg == (FundingTransferRequestInput{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, funding amount must be greater than 0", order.ErrAmountBelowMin)
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.RemittingAccountType != "6" && arg.RemittingAccountType != "18" {
		return nil, fmt.Errorf("%w, remitting account type 6: Funding account 18: Trading account", errAddressRequired)
	}
	if arg.BeneficiaryAccountType != "6" && arg.BeneficiaryAccountType != "18" {
		return nil, fmt.Errorf("%w, beneficiary account type 6: Funding account 18: Trading account", errAddressRequired)
	}
	var resp []FundingTransferResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, fundsTransferEPL, http.MethodPost, "asset/transfer", arg, &resp, request.AuthenticatedRequest)
}

// GetFundsTransferState get funding rate response
func (ok *Okx) GetFundsTransferState(ctx context.Context, transferID, clientID string, transferType int64) ([]TransferFundRateResponse, error) {
	if transferID == "" && clientID == "" {
		return nil, fmt.Errorf("%w, 'transfer id' or 'client id' is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if transferID != "" {
		params.Set("transId", transferID)
	}
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if transferType > 0 && transferType <= 4 {
		params.Set("type", strconv.FormatInt(transferType, 10))
	}
	var resp []TransferFundRateResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFundsTransferStateEPL, http.MethodGet, common.EncodeURLValues("asset/transfer-state", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAssetBillsDetails query the billing record, you can get the latest 1 month historical data
// Bills type possible values are listed here: https://www.okx.com/docs-v5/en/#funding-account-rest-api-asset-bills-details
func (ok *Okx) GetAssetBillsDetails(ctx context.Context, ccy currency.Code, clientID string, after, before time.Time, billType, limit int64) ([]AssetBillDetail, error) {
	params := url.Values{}
	if billType > 0 {
		params.Set("type", strconv.FormatInt(billType, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AssetBillDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, assetBillsDetailsEPL, http.MethodGet, common.EncodeURLValues("asset/bills", params), nil, &resp, request.AuthenticatedRequest)
}

// GetLightningDeposits users can create up to 10 thousand different invoices within 24 hours.
// this method fetches list of lightning deposits filtered by a currency and amount
func (ok *Okx) GetLightningDeposits(ctx context.Context, ccy currency.Code, amount float64, to int64) ([]LightningDepositItem, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	params.Set("amt", strconv.FormatFloat(amount, 'f', 0, 64))
	if to == 6 || to == 18 {
		params.Set("to", strconv.FormatInt(to, 10))
	}
	var resp []LightningDepositItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lightningDepositsEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-lightning", params), nil, &resp, request.AuthenticatedRequest)
}

// GetCurrencyDepositAddress retrieve the deposit addresses of currencies, including previously-used addresses
func (ok *Okx) GetCurrencyDepositAddress(ctx context.Context, ccy currency.Code) ([]CurrencyDepositResponseItem, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	var resp []CurrencyDepositResponseItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDepositAddressEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-address", params), nil, &resp, request.AuthenticatedRequest)
}

// GetCurrencyDepositHistory retrieves deposit records and withdrawal status information depending on the currency, timestamp, and chronological order.
// Possible deposit 'type' are Deposit Type '3': internal transfer '4': deposit from chain
func (ok *Okx) GetCurrencyDepositHistory(ctx context.Context, ccy currency.Code, depositID, transactionID, fromWithdrawalID, depositType string, after, before time.Time, state, limit int64) ([]DepositHistoryResponseItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if depositID != "" {
		params.Set("depId", depositID)
	}
	if fromWithdrawalID != "" {
		params.Set("fromWdId", fromWithdrawalID)
	}
	if transactionID != "" {
		params.Set("txId", transactionID)
	}
	if depositType != "" {
		params.Set("type", depositType)
	}
	params.Set("state", strconv.FormatInt(state, 10))
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []DepositHistoryResponseItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDepositHistoryEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-history", params), nil, &resp, request.AuthenticatedRequest)
}

// Withdrawal to perform a withdrawal action. Sub-account does not support withdrawal
func (ok *Okx) Withdrawal(ctx context.Context, arg *WithdrawalInput) (*WithdrawalResponse, error) {
	if *arg == (WithdrawalInput{}) {
		return nil, common.ErrEmptyParams
	}
	switch {
	case arg.Currency.IsEmpty():
		return nil, currency.ErrCurrencyCodeEmpty
	case arg.Amount <= 0:
		return nil, fmt.Errorf("%w, withdrawal amount required", order.ErrAmountBelowMin)
	case arg.WithdrawalDestination == "":
		return nil, fmt.Errorf("%w, withdrawal destination required", errAddressRequired)
	case arg.ToAddress == "":
		return nil, fmt.Errorf("%w, missing verified digital currency address \"toAddr\" information", errAddressRequired)
	}
	var resp *WithdrawalResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, withdrawalEPL, http.MethodPost, "asset/withdrawal", &arg, &resp, request.AuthenticatedRequest)
}

/*
 This API function service is only open to some users. If you need this function service, please send an email to `liz.jensen@okg.com` to apply
*/

// LightningWithdrawal to withdraw a currency from an invoice
func (ok *Okx) LightningWithdrawal(ctx context.Context, arg *LightningWithdrawalRequestInput) (*LightningWithdrawalResponse, error) {
	if *arg == (LightningWithdrawalRequestInput{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	} else if arg.Invoice == "" {
		return nil, errInvoiceTextMissing
	}
	var resp *LightningWithdrawalResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lightningWithdrawalsEPL, http.MethodPost, "asset/withdrawal-lightning", &arg, &resp, request.AuthenticatedRequest)
}

// CancelWithdrawal cancels a normal withdrawal request but cannot be used to cancel Lightning withdrawals
func (ok *Okx) CancelWithdrawal(ctx context.Context, withdrawalID string) (string, error) {
	if withdrawalID == "" {
		return "", errMissingValidWithdrawalID
	}
	arg := &withdrawData{
		WithdrawalID: withdrawalID,
	}
	var resp withdrawData
	return resp.WithdrawalID, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelWithdrawalEPL, http.MethodPost, "asset/cancel-withdrawal", arg, &resp, request.AuthenticatedRequest)
}

// GetWithdrawalHistory retrieves the withdrawal records according to the currency, withdrawal status, and time range in reverse chronological order.
// The 100 most recent records are returned by default
func (ok *Okx) GetWithdrawalHistory(ctx context.Context, ccy currency.Code, withdrawalID, clientID, transactionID, state string, after, before time.Time, limit int64) ([]WithdrawalHistoryResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if withdrawalID != "" {
		params.Set("wdId", withdrawalID)
	}
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if transactionID != "" {
		params.Set("txId", transactionID)
	}
	if state != "" {
		params.Set("state", state)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []WithdrawalHistoryResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getWithdrawalHistoryEPL, http.MethodGet, common.EncodeURLValues("asset/withdrawal-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetDepositWithdrawalStatus retrieves the detailed status and estimated completion time for deposits and withdrawals
func (ok *Okx) GetDepositWithdrawalStatus(ctx context.Context, ccy currency.Code, withdrawalID, transactionID, addressTo, chain string) ([]DepositWithdrawStatus, error) {
	if withdrawalID == "" && transactionID == "" {
		return nil, fmt.Errorf("%w, either withdrawal id or transaction id is required", order.ErrOrderIDNotSet)
	}
	if withdrawalID == "" {
		return nil, errMissingValidWithdrawalID
	}
	params := url.Values{}
	params.Set("wdId", withdrawalID)
	if transactionID != "" {
		params.Set("txId", transactionID)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if addressTo != "" {
		params.Set("to", addressTo)
	}
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp []DepositWithdrawStatus
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDepositWithdrawalStatusEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-withdraw-status", params), nil, &resp, request.AuthenticatedRequest)
}

// SmallAssetsConvert Convert small assets in funding account to OKB. Only one convert is allowed within 24 hours
func (ok *Okx) SmallAssetsConvert(ctx context.Context, currency []string) (*SmallAssetConvertResponse, error) {
	var resp *SmallAssetConvertResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, smallAssetsConvertEPL, http.MethodPost, "asset/convert-dust-assets", map[string][]string{"ccy": currency}, &resp, request.AuthenticatedRequest)
}

// GetPublicExchangeList retrieves exchanges
func (ok *Okx) GetPublicExchangeList(ctx context.Context) ([]ExchangeInfo, error) {
	var resp []ExchangeInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPublicExchangeListEPL, http.MethodGet, "asset/exchange-list", nil, &resp, request.UnauthenticatedRequest)
}

// GetSavingBalance returns saving balance, and only assets in the funding account can be used for saving
func (ok *Okx) GetSavingBalance(ctx context.Context, ccy currency.Code) ([]SavingBalanceResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []SavingBalanceResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSavingBalanceEPL, http.MethodGet, common.EncodeURLValues("finance/savings/balance", params), nil, &resp, request.AuthenticatedRequest)
}

// SavingsPurchaseOrRedemption creates a purchase or redemption instance
func (ok *Okx) SavingsPurchaseOrRedemption(ctx context.Context, arg *SavingsPurchaseRedemptionInput) (*SavingsPurchaseRedemptionResponse, error) {
	if *arg == (SavingsPurchaseRedemptionInput{}) {
		return nil, common.ErrEmptyParams
	}
	arg.ActionType = strings.ToLower(arg.ActionType)
	switch {
	case arg.Currency.IsEmpty():
		return nil, currency.ErrCurrencyCodeEmpty
	case arg.Amount <= 0:
		return nil, order.ErrAmountBelowMin
	case arg.ActionType != "purchase" && arg.ActionType != "redempt":
		return nil, fmt.Errorf("%w, side has to be either 'redempt' or 'purchase'", order.ErrSideIsInvalid)
	case arg.ActionType == "purchase" && (arg.Rate < 0.01 || arg.Rate > 3.65):
		return nil, fmt.Errorf("%w, the rate value range is between 0.01 and 3.65", errRateRequired)
	}
	var resp *SavingsPurchaseRedemptionResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, savingsPurchaseRedemptionEPL, http.MethodPost, "finance/savings/purchase-redempt", &arg, &resp, request.AuthenticatedRequest)
}

// GetLendingHistory lending history
func (ok *Okx) GetLendingHistory(ctx context.Context, ccy currency.Code, before, after time.Time, limit int64) ([]LendingHistory, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LendingHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLendingHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/savings/lending-history", params), nil, &resp, request.AuthenticatedRequest)
}

// SetLendingRate sets an assets lending rate
func (ok *Okx) SetLendingRate(ctx context.Context, arg *LendingRate) (*LendingRate, error) {
	if *arg == (LendingRate{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	} else if arg.Rate < 0.01 || arg.Rate > 3.65 {
		return nil, fmt.Errorf("%w, rate value range is between 1 percent (0.01) and 365 percent (3.65)", errRateRequired)
	}
	var resp *LendingRate
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setLendingRateEPL, http.MethodPost, "finance/savings/set-lending-rate", &arg, &resp, request.AuthenticatedRequest)
}

// GetPublicBorrowInfo returns the public borrow info
func (ok *Okx) GetPublicBorrowInfo(ctx context.Context, ccy currency.Code) ([]PublicBorrowInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []PublicBorrowInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPublicBorrowInfoEPL, http.MethodGet, common.EncodeURLValues("finance/savings/lending-rate-summary", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPublicBorrowHistory return list of publix borrow history
func (ok *Okx) GetPublicBorrowHistory(ctx context.Context, ccy currency.Code, before, after time.Time, limit int64) ([]PublicBorrowHistory, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PublicBorrowHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPublicBorrowHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/savings/lending-rate-history", params), nil, &resp, request.UnauthenticatedRequest)
}

/***********************************Convert Endpoints | Authenticated s*****************************************/

// ApplyForMonthlyStatement requests a monthly statement for any month within the past year
func (ok *Okx) ApplyForMonthlyStatement(ctx context.Context, month string) (types.Time, error) {
	if month == "" {
		return types.Time{}, errMonthNameRequired
	}
	resp := &tsResp{}
	return resp.Timestamp, ok.SendHTTPRequest(ctx, exchange.RestSpot, applyForMonthlyStatementEPL,
		http.MethodPost, "asset/monthly-statement", &map[string]string{"month": month}, resp, request.AuthenticatedRequest)
}

// GetMonthlyStatement retrieves monthly statements for the past year.
// Month is in the form of Jan, Feb, March etc.
func (ok *Okx) GetMonthlyStatement(ctx context.Context, month string) ([]MonthlyStatement, error) {
	if month == "" {
		return nil, errMonthNameRequired
	}
	params := url.Values{}
	params.Set("month", month)
	var resp []MonthlyStatement
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMonthlyStatementEPL, http.MethodGet,
		common.EncodeURLValues("asset/monthly-statement", params), nil, &resp, request.AuthenticatedRequest)
}

// GetConvertCurrencies retrieves the currency conversion information
func (ok *Okx) GetConvertCurrencies(ctx context.Context) ([]ConvertCurrency, error) {
	var resp []ConvertCurrency
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getConvertCurrenciesEPL, http.MethodGet, "asset/convert/currencies", nil, &resp, request.AuthenticatedRequest)
}

// GetConvertCurrencyPair retrieves the currency conversion response detail given the 'currency from' and 'currency to'
func (ok *Okx) GetConvertCurrencyPair(ctx context.Context, fromCcy, toCcy currency.Code) (*ConvertCurrencyPair, error) {
	if fromCcy.IsEmpty() {
		return nil, fmt.Errorf("%w, source currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if toCcy.IsEmpty() {
		return nil, fmt.Errorf("%w, target currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("fromCcy", fromCcy.String())
	params.Set("toCcy", toCcy.String())
	var resp *ConvertCurrencyPair
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getConvertCurrencyPairEPL, http.MethodGet, common.EncodeURLValues("asset/convert/currency-pair", params), nil, &resp, request.AuthenticatedRequest)
}

// EstimateQuote retrieves quote estimation detail result given the base and quote currency
func (ok *Okx) EstimateQuote(ctx context.Context, arg *EstimateQuoteRequestInput) (*EstimateQuoteResponse, error) {
	if *arg == (EstimateQuoteRequestInput{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.BaseCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, base currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.QuoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, quote currency is required", currency.ErrCurrencyCodeEmpty)
	}
	arg.Side = strings.ToLower(arg.Side)
	switch arg.Side {
	case order.Buy.Lower(), order.Sell.Lower():
	default:
		return nil, order.ErrSideIsInvalid
	}
	if arg.RFQAmount <= 0 {
		return nil, fmt.Errorf("%w, RFQ amount required", order.ErrAmountBelowMin)
	}
	if arg.RFQSzCurrency == "" {
		return nil, fmt.Errorf("%w, missing RFQ currency", currency.ErrCurrencyCodeEmpty)
	}
	var resp *EstimateQuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, estimateQuoteEPL, http.MethodPost, "asset/convert/estimate-quote", arg, &resp, request.AuthenticatedRequest)
}

// ConvertTrade converts a base currency to quote currency
func (ok *Okx) ConvertTrade(ctx context.Context, arg *ConvertTradeInput) (*ConvertTradeResponse, error) {
	if *arg == (ConvertTradeInput{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.BaseCurrency == "" {
		return nil, fmt.Errorf("%w, base currency required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.QuoteCurrency == "" {
		return nil, fmt.Errorf("%w, quote currency required", currency.ErrCurrencyCodeEmpty)
	}
	arg.Side = strings.ToLower(arg.Side)
	switch arg.Side {
	case order.Buy.Lower(), order.Sell.Lower():
	default:
		return nil, order.ErrSideIsInvalid
	}
	if arg.Size <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.SizeCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.QuoteID == "" {
		return nil, fmt.Errorf("%w, quote id required", order.ErrOrderIDNotSet)
	}
	var resp *ConvertTradeResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, convertTradeEPL, http.MethodPost, "asset/convert/trade", &arg, &resp, request.AuthenticatedRequest)
}

// GetConvertHistory gets the recent history
func (ok *Okx) GetConvertHistory(ctx context.Context, before, after time.Time, limit int64, tag string) ([]ConvertHistory, error) {
	params := url.Values{}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if tag != "" {
		params.Set("tag", tag)
	}
	var resp []ConvertHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getConvertHistoryEPL, http.MethodGet, common.EncodeURLValues("asset/convert/history", params), nil, &resp, request.AuthenticatedRequest)
}

/********************************** Account endpoints ***************************************************/

// GetAccountInstruments retrieve available instruments info of current account
func (ok *Okx) GetAccountInstruments(ctx context.Context, instrumentType asset.Item, underlying, instrumentFamily, instrumentID string) ([]AccountInstrument, error) {
	if instrumentType == asset.Empty {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	switch instrumentType {
	case asset.Margin, asset.PerpetualSwap, asset.Futures:
		if underlying == "" {
			return nil, fmt.Errorf("%w, underlying is required", errInvalidUnderlying)
		}
		params.Set("uly", underlying)
	case asset.Options:
		if underlying == "" && instrumentFamily == "" {
			return nil, errInstrumentFamilyOrUnderlyingRequired
		}
		if underlying != "" {
			params.Set("uly", underlying)
		}
		if instrumentFamily != "" {
			params.Set("instFamily", instrumentFamily)
		}
	}
	instTypeString, err := assetTypeString(instrumentType)
	if err != nil {
		return nil, err
	}
	params.Set("instType", instTypeString)
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var resp []AccountInstrument
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountInstrumentsEPL, http.MethodGet, common.EncodeURLValues("account/instruments", params), nil, &resp, request.AuthenticatedRequest)
}

// AccountBalance retrieves a list of assets (with non-zero balance), remaining balance, and available amount in the trading account.
// Interest-free quota and discount rates are public data and not displayed on the account interface
func (ok *Okx) AccountBalance(ctx context.Context, ccy currency.Code) ([]Account, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []Account
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountBalanceEPL, http.MethodGet, common.EncodeURLValues("account/balance", params), nil, &resp, request.AuthenticatedRequest)
}

// GetPositions retrieves information on your positions. When the account is in net mode, net positions will be displayed, and when the account is in long/short mode, long or short positions will be displayed
func (ok *Okx) GetPositions(ctx context.Context, instrumentType, instrumentID, positionID string) ([]AccountPosition, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if positionID != "" {
		params.Set("posId", positionID)
	}
	var resp []AccountPosition
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPositionsEPL, http.MethodGet, common.EncodeURLValues("account/positions", params), nil, &resp, request.AuthenticatedRequest)
}

// GetPositionsHistory retrieves the updated position data for the last 3 months
func (ok *Okx) GetPositionsHistory(ctx context.Context, instrumentType, instrumentID, marginMode, positionID string, closePositionType, limit int64, after, before time.Time) ([]AccountPositionHistory, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if marginMode != "" {
		params.Set("mgnMode", marginMode) // margin mode 'cross' and 'isolated' are allowed
	}
	// The type of closing position
	// 1：Close position partially;2：Close all;3：Liquidation;4：Partial liquidation; 5：ADL;
	// It is the latest type if there are several types for the same position.
	if closePositionType > 0 {
		params.Set("type", strconv.FormatInt(closePositionType, 10))
	}
	if positionID != "" {
		params.Set("posId", positionID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AccountPositionHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPositionsHistoryEPL, http.MethodGet, common.EncodeURLValues("account/positions-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAccountAndPositionRisk  get account and position risks
func (ok *Okx) GetAccountAndPositionRisk(ctx context.Context, instrumentType string) ([]AccountAndPositionRisk, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", strings.ToUpper(instrumentType))
	}
	var resp []AccountAndPositionRisk
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountAndPositionRiskEPL, http.MethodGet, common.EncodeURLValues("account/account-position-risk", params), nil, &resp, request.AuthenticatedRequest)
}

// GetBillsDetailLast7Days The bill refers to all transaction records that result in changing the balance of an account. Pagination is supported, and the response is sorted with the most recent first. This endpoint can retrieve data from the last 7 days
func (ok *Okx) GetBillsDetailLast7Days(ctx context.Context, arg *BillsDetailQueryParameter) ([]BillsDetailResponse, error) {
	return ok.GetBillsDetail(ctx, arg, "account/bills", getBillsDetailsEPL)
}

// GetBillsDetail3Months retrieves the account’s bills.
// The bill refers to all transaction records that result in changing the balance of an account.
// Pagination is supported, and the response is sorted with most recent first.
// This endpoint can retrieve data from the last 3 months
func (ok *Okx) GetBillsDetail3Months(ctx context.Context, arg *BillsDetailQueryParameter) ([]BillsDetailResponse, error) {
	return ok.GetBillsDetail(ctx, arg, "account/bills-archive", getBillsDetailArchiveEPL)
}

// ApplyBillDetails apply for bill data since 1 February, 2021 except for the current quarter.
// Quarter, valid value is Q1, Q2, Q3, Q4
func (ok *Okx) ApplyBillDetails(ctx context.Context, year, quarter string) ([]BillsDetailResp, error) {
	if year == "" {
		return nil, errYearRequired
	}
	if quarter == "" {
		return nil, errQuarterValueRequired
	}
	var resp []BillsDetailResp
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, billHistoryArchiveEPL, http.MethodPost, "account/bills-history-archive", map[string]string{"year": year, "quarter": quarter}, &resp, request.AuthenticatedRequest)
}

// GetBillsHistoryArchive retrieves bill data archive
func (ok *Okx) GetBillsHistoryArchive(ctx context.Context, year, quarter string) ([]BillsArchiveInfo, error) {
	if year == "" {
		return nil, errYearRequired
	}
	if quarter == "" {
		return nil, errQuarterValueRequired
	}
	params := url.Values{}
	params.Set("year", year)
	params.Set("quarter", quarter)
	var resp []BillsArchiveInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBillHistoryArchiveEPL, http.MethodGet, common.EncodeURLValues("account/bills-history-archive", params), nil, &resp, request.AuthenticatedRequest)
}

// GetBillsDetail retrieves the bills of the account
func (ok *Okx) GetBillsDetail(ctx context.Context, arg *BillsDetailQueryParameter, route string, epl request.EndpointLimit) ([]BillsDetailResponse, error) {
	if *arg == (BillsDetailQueryParameter{}) {
		return nil, common.ErrEmptyParams
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", strings.ToUpper(arg.InstrumentType))
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	if arg.MarginMode != "" {
		params.Set("mgnMode", arg.MarginMode)
	}
	if arg.ContractType != "" {
		params.Set("ctType", arg.ContractType)
	}
	if arg.BillType >= 1 && arg.BillType <= 13 {
		params.Set("type", strconv.Itoa(int(arg.BillType)))
	}
	if arg.BillSubType != 0 {
		params.Set("subType", strconv.FormatInt(arg.BillSubType, 10))
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if !arg.BeginTime.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("end", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp []BillsDetailResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, request.AuthenticatedRequest)
}

// GetAccountConfiguration retrieves current account configuration
func (ok *Okx) GetAccountConfiguration(ctx context.Context) (*AccountConfigurationResponse, error) {
	var resp *AccountConfigurationResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountConfigurationEPL, http.MethodGet, "account/config", nil, &resp, request.AuthenticatedRequest)
}

// SetPositionMode FUTURES and SWAP support both long/short mode and net mode. In net mode, users can only have positions in one direction; In long/short mode, users can hold positions in long and short directions.
// Position mode 'long_short_mode': long/short, only applicable to  FUTURES/SWAP'net_mode': net
func (ok *Okx) SetPositionMode(ctx context.Context, positionMode string) (*PositionMode, error) {
	if positionMode != "long_short_mode" && positionMode != "net_mode" {
		return nil, errInvalidPositionMode
	}
	var resp *PositionMode
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setPositionModeEPL, http.MethodPost, "account/set-position-mode", &PositionMode{
		PositionMode: positionMode,
	}, &resp, request.AuthenticatedRequest)
}

// SetLeverageRate sets a leverage setting for instrument id
func (ok *Okx) SetLeverageRate(ctx context.Context, arg *SetLeverageInput) (*SetLeverageResponse, error) {
	if *arg == (SetLeverageInput{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentID == "" && arg.Currency.IsEmpty() {
		return nil, errEitherInstIDOrCcyIsRequired
	}
	switch arg.AssetType {
	case asset.Futures, asset.PerpetualSwap:
		if arg.PositionSide == "" && arg.MarginMode == "isolated" {
			return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.PositionSide)
		}
	}
	arg.PositionSide = strings.ToLower(arg.PositionSide)
	var resp *SetLeverageResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setLeverageEPL, http.MethodPost, "account/set-leverage", &arg, &resp, request.AuthenticatedRequest)
}

// GetMaximumBuySellAmountOROpenAmount retrieves the maximum buy or sell amount for a sell id
func (ok *Okx) GetMaximumBuySellAmountOROpenAmount(ctx context.Context, ccy currency.Code, instrumentID, tradeMode, leverage string, price float64, unSpotOffset bool) ([]MaximumBuyAndSell, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	switch tradeMode {
	case TradeModeCross, TradeModeIsolated, TradeModeCash:
	default:
		return nil, fmt.Errorf("%w, trade mode: %s", errInvalidTradeModeValue, tradeMode)
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("tdMode", tradeMode)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if price > 0 {
		params.Set("px", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if leverage != "" {
		params.Set("leverage", leverage)
	}
	if unSpotOffset {
		params.Set("unSpotOffset", "true")
	}
	var resp []MaximumBuyAndSell
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMaximumBuyOrSellAmountEPL, http.MethodGet, common.EncodeURLValues("account/max-size", params), nil, &resp, request.AuthenticatedRequest)
}

// GetMaximumAvailableTradableAmount retrieves the maximum tradable amount for specific instrument id, and/or currency
func (ok *Okx) GetMaximumAvailableTradableAmount(ctx context.Context, ccy currency.Code, instrumentID, tradeMode, quickMarginType string, reduceOnly, upSpotOffset bool, price float64) ([]MaximumTradableAmount, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if tradeMode == "" {
		// Supported tradeMode values are 'cross', 'isolated', 'cash', and 'spot_isolated'
		return nil, fmt.Errorf("%w, possible values are 'cross', 'isolated', 'cash', and 'spot_isolated'", errInvalidTradeModeValue)
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("tdMode", tradeMode)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if reduceOnly {
		params.Set("reduceOnly", "true")
	}
	if price != 0 {
		params.Set("px", strconv.FormatFloat(price, 'f', 0, 64))
	}
	if quickMarginType != "" {
		params.Set("quickMgnType", quickMarginType)
	}
	if upSpotOffset {
		params.Set("upSpotOffset", "true")
	}
	var resp []MaximumTradableAmount
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMaximumAvailableTradableAmountEPL, http.MethodGet, common.EncodeURLValues("account/max-avail-size", params), nil, &resp, request.AuthenticatedRequest)
}

// IncreaseDecreaseMargin Increase or decrease the margin of the isolated position. Margin reduction may result in the change of the actual leverage
func (ok *Okx) IncreaseDecreaseMargin(ctx context.Context, arg *IncreaseDecreaseMarginInput) (*IncreaseDecreaseMargin, error) {
	if *arg == (IncreaseDecreaseMarginInput{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if !slices.Contains([]string{positionSideLong, positionSideShort, positionSideNet}, arg.PositionSide) {
		return nil, fmt.Errorf("%w, position side is required", order.ErrSideIsInvalid)
	}
	if arg.MarginBalanceType != marginBalanceAdd && arg.MarginBalanceType != marginBalanceReduce {
		return nil, fmt.Errorf("%w, missing valid 'type', 'add': add margin 'reduce': reduce margin are allowed", order.ErrTypeIsInvalid)
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp *IncreaseDecreaseMargin
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, increaseOrDecreaseMarginEPL, http.MethodPost, "account/position/margin-balance", &arg, &resp, request.AuthenticatedRequest)
}

// GetLeverageRate retrieves leverage data for different instrument id or margin mode
func (ok *Okx) GetLeverageRate(ctx context.Context, instrumentID, marginMode string, ccy currency.Code) ([]LeverageResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	switch marginMode {
	case TradeModeCross, TradeModeIsolated, TradeModeCash:
	default:
		return nil, margin.ErrMarginTypeUnsupported
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("mgnMode", marginMode)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []LeverageResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeverageEPL, http.MethodGet, common.EncodeURLValues("account/leverage-info", params), nil, &resp, request.AuthenticatedRequest)
}

// GetLeverageEstimatedInfo retrieves leverage estimated information.
// Instrument type: possible values are MARGIN, SWAP, FUTURES
func (ok *Okx) GetLeverageEstimatedInfo(ctx context.Context, instrumentType, marginMode, leverage, positionSide, instrumentID string, ccy currency.Code) ([]LeverageEstimatedInfo, error) {
	if instrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	switch marginMode {
	case TradeModeCross, TradeModeIsolated:
	default:
		return nil, margin.ErrMarginTypeUnsupported
	}
	if leverage == "" {
		return nil, errInvalidLeverage
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	params.Set("lever", leverage)
	params.Set("mgnMode", marginMode)
	params.Set("ccy", ccy.String())
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if positionSide != "" {
		params.Set("posSide", positionSide)
	}
	var resp []LeverageEstimatedInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeverateEstimatedInfoEPL, http.MethodGet, common.EncodeURLValues("account/adjust-leverage-info", params), nil, &resp, request.AuthenticatedRequest)
}

// GetMaximumLoanOfInstrument returns list of maximum loan of instruments
func (ok *Okx) GetMaximumLoanOfInstrument(ctx context.Context, instrumentID, marginMode string, mgnCurrency currency.Code) ([]MaximumLoanInstrument, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if marginMode == "" {
		return nil, margin.ErrInvalidMarginType
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("mgnMode", marginMode)
	if !mgnCurrency.IsEmpty() {
		params.Set("mgnCcy", mgnCurrency.String())
	}
	var resp []MaximumLoanInstrument
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTheMaximumLoanOfInstrumentEPL, http.MethodGet, common.EncodeURLValues("account/max-loan", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFee returns Cryptocurrency trade fee, and offline trade fee
func (ok *Okx) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	// Here the Asset Type for the instrument Type is needed for getting the CryptocurrencyTrading Fee.
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		uly, err := ok.GetUnderlying(feeBuilder.Pair, asset.Spot)
		if err != nil {
			return 0, err
		}
		responses, err := ok.GetTradeFee(ctx, instTypeSpot, uly, "", "", "")
		if err != nil {
			return 0, err
		} else if len(responses) == 0 {
			return 0, common.ErrNoResponse
		}
		if feeBuilder.IsMaker {
			if feeBuilder.Pair.Quote.Equal(currency.USDC) {
				fee = responses[0].FeeRateMakerUSDC.Float64()
			} else if fee = responses[0].FeeRateMaker.Float64(); fee == 0 {
				fee = responses[0].FeeRateMakerUSDT.Float64()
			}
		} else {
			if feeBuilder.Pair.Quote.Equal(currency.USDC) {
				fee = responses[0].FeeRateTakerUSDC.Float64()
			} else if fee = responses[0].FeeRateTaker.Float64(); fee == 0 {
				fee = responses[0].FeeRateTakerUSDT.Float64()
			}
		}
		if fee != 0 {
			fee = -fee // A negative fee rate indicates a commission charge; a positive rate indicates a rebate.
		}
		return fee * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
	case exchange.OfflineTradeFee:
		return 0.0015 * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	default:
		return fee, errFeeTypeUnsupported
	}
}

// GetTradeFee queries the trade fee rates for various instrument types and their respective IDs
func (ok *Okx) GetTradeFee(ctx context.Context, instrumentType, instrumentID, underlying, instrumentFamily, ruleType string) ([]TradeFeeRate, error) {
	if instrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if underlying != "" {
		params.Set("uly", underlying)
	}
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	if ruleType != "" {
		params.Set("ruleType", ruleType)
	}
	var resp []TradeFeeRate
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFeeRatesEPL, http.MethodGet, common.EncodeURLValues("account/trade-fee", params), nil, &resp, request.AuthenticatedRequest)
}

// GetInterestAccruedData retrieves data on accrued interest
func (ok *Okx) GetInterestAccruedData(ctx context.Context, loanType, limit int64, ccy currency.Code, instrumentID, marginMode string, after, before time.Time) ([]InterestAccruedData, error) {
	params := url.Values{}
	if loanType == 1 || loanType == 2 {
		params.Set("type", strconv.FormatInt(loanType, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if marginMode != "" {
		params.Set("mgnMode", strings.ToLower(marginMode))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []InterestAccruedData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getInterestAccruedDataEPL, http.MethodGet, common.EncodeURLValues("account/interest-accrued", params), nil, &resp, request.AuthenticatedRequest)
}

// GetInterestRate get the user's current leveraged currency borrowing interest rate
func (ok *Okx) GetInterestRate(ctx context.Context, ccy currency.Code) ([]InterestRateResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []InterestRateResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getInterestRateEPL, http.MethodGet, common.EncodeURLValues("account/interest-rate", params), nil, &resp, request.AuthenticatedRequest)
}

// SetGreeks set the display type of Greeks. PA: Greeks in coins BS: Black-Scholes Greeks in dollars
func (ok *Okx) SetGreeks(ctx context.Context, greeksType string) (*GreeksType, error) {
	greeksType = strings.ToUpper(greeksType)
	if greeksType != "PA" && greeksType != "BS" {
		return nil, errMissingValidGreeksType
	}
	input := &GreeksType{
		GreeksType: greeksType,
	}
	var resp *GreeksType
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setGreeksEPL, http.MethodPost, "account/set-greeks", input, &resp, request.AuthenticatedRequest)
}

// IsolatedMarginTradingSettings configures the currency margin and sets the isolated margin trading mode for futures or perpetual contracts
func (ok *Okx) IsolatedMarginTradingSettings(ctx context.Context, arg *IsolatedMode) (*IsolatedMode, error) {
	if *arg == (IsolatedMode{}) {
		return nil, common.ErrEmptyParams
	}
	arg.IsoMode = strings.ToLower(arg.IsoMode)
	if arg.IsoMode != "automatic" &&
		arg.IsoMode != "autonomy" {
		return nil, errMissingIsolatedMarginTradingSetting
	}
	if arg.InstrumentType != instTypeMargin &&
		arg.InstrumentType != instTypeContract {
		return nil, fmt.Errorf("%w, received '%v' only margin and contract instrument types are allowed", errInvalidInstrumentType, arg.InstrumentType)
	}
	var resp *IsolatedMode
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, isolatedMarginTradingSettingsEPL, http.MethodPost, "account/set-isolated-mode", &arg, &resp, request.AuthenticatedRequest)
}

// ManualBorrowAndRepayInQuickMarginMode initiates a new manual borrow and repayment process in Quick Margin mode
func (ok *Okx) ManualBorrowAndRepayInQuickMarginMode(ctx context.Context, arg *BorrowAndRepay) (*BorrowAndRepay, error) {
	if *arg == (BorrowAndRepay{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.LoanCcy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side == "" {
		return nil, fmt.Errorf("%w, possible values are 'borrow' and 'repay'", order.ErrSideIsInvalid)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	var resp *BorrowAndRepay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, manualBorrowAndRepayEPL, http.MethodPost, "account/quick-margin-borrow-repay", arg, &resp, request.AuthenticatedRequest)
}

// GetBorrowAndRepayHistoryInQuickMarginMode retrieves borrow and repay history in quick margin mode
func (ok *Okx) GetBorrowAndRepayHistoryInQuickMarginMode(ctx context.Context, instrumentID currency.Pair, ccy currency.Code, side, afterPaginationID, beforePaginationID string, beginTime, endTime time.Time, limit int64) ([]BorrowRepayHistoryItem, error) {
	params := url.Values{}
	if !instrumentID.IsEmpty() {
		params.Set("instId", instrumentID.String())
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if side != "" {
		params.Set("side", side)
	}
	if afterPaginationID != "" {
		params.Set("after", afterPaginationID)
	}
	if beforePaginationID != "" {
		params.Set("before", beforePaginationID)
	}
	if !beginTime.IsZero() {
		params.Set("begin", strconv.FormatInt(beginTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BorrowRepayHistoryItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBorrowAndRepayHistoryEPL, http.MethodGet, common.EncodeURLValues("account/quick-margin-borrow-repay-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetMaximumWithdrawals retrieves the maximum transferable amount from a trading account to a funding account for quick margin borrowing and repayment
func (ok *Okx) GetMaximumWithdrawals(ctx context.Context, ccy currency.Code) ([]MaximumWithdrawal, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []MaximumWithdrawal
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMaximumWithdrawalsEPL, http.MethodGet, common.EncodeURLValues("account/max-withdrawal", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAccountRiskState gets the account risk status.
// only applicable to Portfolio margin account
func (ok *Okx) GetAccountRiskState(ctx context.Context) ([]AccountRiskState, error) {
	var resp []AccountRiskState
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountRiskStateEPL, http.MethodGet, "account/risk-state", nil, &resp, request.AuthenticatedRequest)
}

// VIPLoansBorrowAndRepay creates VIP borrow or repay for a currency
func (ok *Okx) VIPLoansBorrowAndRepay(ctx context.Context, arg *LoanBorrowAndReplayInput) (*LoanBorrowAndReplay, error) {
	if *arg == (LoanBorrowAndReplayInput{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp *LoanBorrowAndReplay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, vipLoansBorrowAnsRepayEPL, http.MethodPost, "account/borrow-repay", &arg, &resp, request.AuthenticatedRequest)
}

// GetBorrowAndRepayHistoryForVIPLoans retrieves borrow and repay history for VIP loans
func (ok *Okx) GetBorrowAndRepayHistoryForVIPLoans(ctx context.Context, ccy currency.Code, after, before time.Time, limit int64) ([]BorrowRepayHistory, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BorrowRepayHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBorrowAnsRepayHistoryHistoryEPL, http.MethodGet, common.EncodeURLValues("account/borrow-repay-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetVIPInterestAccruedData retrieves VIP interest accrued data
func (ok *Okx) GetVIPInterestAccruedData(ctx context.Context, ccy currency.Code, orderID string, after, before time.Time, limit int64) ([]VIPInterestData, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []VIPInterestData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getVIPInterestAccruedDataEPL, http.MethodGet, common.EncodeURLValues("account/vip-interest-accrued", params), nil, &resp, request.AuthenticatedRequest)
}

// GetVIPInterestDeductedData retrieves a VIP interest deducted data
func (ok *Okx) GetVIPInterestDeductedData(ctx context.Context, ccy currency.Code, orderID string, after, before time.Time, limit int64) ([]VIPInterestData, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []VIPInterestData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getVIPInterestDeductedDataEPL, http.MethodGet, common.EncodeURLValues("account/vip-interest-deducted", params), nil, &resp, request.AuthenticatedRequest)
}

// GetVIPLoanOrderList retrieves VIP loan order list
// state: possible values are 1:Borrowing 2:Borrowed 3:Repaying 4:Repaid 5:Borrow failed
func (ok *Okx) GetVIPLoanOrderList(ctx context.Context, orderID, state string, ccy currency.Code, after, before time.Time, limit int64) ([]VIPLoanOrder, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if state != "" {
		params.Set("state", state)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []VIPLoanOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getVIPLoanOrderListEPL, http.MethodGet, common.EncodeURLValues("account/vip-loan-order-list", params), nil, &resp, request.AuthenticatedRequest)
}

// GetVIPLoanOrderDetail retrieves list of loan order details
func (ok *Okx) GetVIPLoanOrderDetail(ctx context.Context, orderID string, ccy currency.Code, after, before time.Time, limit int64) (*VIPLoanOrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("ordId", orderID)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *VIPLoanOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getVIPLoanOrderDetailEPL, http.MethodGet, common.EncodeURLValues("account/vip-loan-order-detail", params), nil, &resp, request.AuthenticatedRequest)
}

// GetBorrowInterestAndLimit borrow interest and limit
func (ok *Okx) GetBorrowInterestAndLimit(ctx context.Context, loanType int64, ccy currency.Code) ([]BorrowInterestAndLimitResponse, error) {
	params := url.Values{}
	if loanType == 1 || loanType == 2 {
		params.Set("type", strconv.FormatInt(loanType, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []BorrowInterestAndLimitResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBorrowInterestAndLimitEPL, http.MethodGet, common.EncodeURLValues("account/interest-limits", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFixedLoanBorrowLimit retrieves a fixed loadn borrow limit information
func (ok *Okx) GetFixedLoanBorrowLimit(ctx context.Context) (*FixedLoanBorrowLimitInformation, error) {
	var resp *FixedLoanBorrowLimitInformation
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFixedLoanBorrowLimitEPL, http.MethodGet, "account/fixed-loan/borrowing-limit", nil, &resp, request.AuthenticatedRequest)
}

// GetFixedLoanBorrowQuote retrieves a fixed loan borrow quote information
func (ok *Okx) GetFixedLoanBorrowQuote(ctx context.Context, borrowingCurrency currency.Code, borrowType, term, orderID string, amount, maxRate float64) (*FixedLoanBorrowQuote, error) {
	if borrowType == "" {
		return nil, errBorrowTypeRequired
	}

	switch borrowType {
	case "normal":
		if borrowingCurrency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if amount <= 0 {
			return nil, order.ErrAmountBelowMin
		}
		if maxRate <= 0 {
			return nil, errMaxRateRequired
		}
		if term == "" {
			return nil, errLendingTermIsRequired
		}
	case "reborrow":
		if orderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
	}

	params := url.Values{}
	params.Set("type", borrowType)
	if !borrowingCurrency.IsEmpty() {
		params.Set("ccy", borrowingCurrency.String())
	}
	if amount > 0 {
		params.Set("amt", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	if maxRate > 0 {
		params.Set("maxRate", strconv.FormatFloat(maxRate, 'f', -1, 64))
	}
	if term != "" {
		params.Set("term", term)
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	var resp *FixedLoanBorrowQuote
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFixedLoanBorrowQuoteEPL, http.MethodGet, common.EncodeURLValues("account/fixed-loan/borrowing-quote", params), nil, &resp, request.AuthenticatedRequest)
}

// PlaceFixedLoanBorrowingOrder for new borrowing orders, they belong to the IOC (immediately close and cancel the remaining) type. For renewal orders, they belong to the FOK (Fill-or-kill) type
func (ok *Okx) PlaceFixedLoanBorrowingOrder(ctx context.Context, ccy currency.Code, amount, maxRate, reborrowRate float64, term string, reborrow bool) (*OrderIDResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if maxRate <= 0 {
		return nil, errMaxRateRequired
	}
	if term == "" {
		return nil, errLendingTermIsRequired
	}
	arg := &struct {
		Currency     string  `json:"ccy"`
		Amount       float64 `json:"amt,string"`
		MaxRate      float64 `jsons:"maxRate,string"`
		Term         string  `json:"term"`
		Reborrow     bool    `json:"reborrow,omitempty"`
		ReborrowRate float64 `json:"reborrowRate,string,omitempty"`
	}{
		Currency:     ccy.String(),
		Amount:       amount,
		MaxRate:      maxRate,
		Term:         term,
		Reborrow:     reborrow,
		ReborrowRate: reborrowRate,
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeFixedLoanBorrowingOrderEPL, http.MethodPost, "account/fixed-loan/borrowing-order", arg, &resp, request.AuthenticatedRequest)
}

// AmendFixedLoanBorrowingOrder amends a fixed loan borrowing order
func (ok *Okx) AmendFixedLoanBorrowingOrder(ctx context.Context, orderID string, reborrow bool, renewMaxRate float64) (*OrderIDResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	arg := &struct {
		OrderID      string  `json:"ordId"`
		Reborrow     bool    `json:"reborrow,omitempty"`
		RenewMaxRate float64 `json:"renewMaxRate,omitempty,string"`
	}{
		OrderID:      orderID,
		Reborrow:     reborrow,
		RenewMaxRate: renewMaxRate,
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendFixedLaonBorrowingOrderEPL, http.MethodPost, "account/fixed-loan/amend-borrowing-order", arg, &resp, request.AuthenticatedRequest)
}

// ManualRenewFixedLoanBorrowingOrder manual renew fixed loan borrowing order
func (ok *Okx) ManualRenewFixedLoanBorrowingOrder(ctx context.Context, orderID string, maxRate float64) (*OrderIDResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if maxRate <= 0 {
		return nil, errMaxRateRequired
	}
	arg := &struct {
		OrderID string  `json:"ordId"`
		MaxRate float64 `json:"maxRate,string"`
	}{
		OrderID: orderID,
		MaxRate: maxRate,
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, manualRenewFixedLoanBorrowingOrderEPL, http.MethodPost, "account/fixed-loan/manual-reborrow", arg, &resp, request.AuthenticatedRequest)
}

// RepayFixedLoanBorrowingOrder repays fixed loan borrowing order
func (ok *Okx) RepayFixedLoanBorrowingOrder(ctx context.Context, orderID string) (*OrderIDResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, repayFixedLoanBorrowingOrderEPL, http.MethodPost, "account/fixed-loan/repay-borrowing-order", map[string]string{"ordId": orderID}, &resp, request.AuthenticatedRequest)
}

// ConvertFixedLoanToMarketLoan converts fixed loan to market loan
func (ok *Okx) ConvertFixedLoanToMarketLoan(ctx context.Context, orderID string) (*OrderIDResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, convertFixedLoanToMarketLoanEPL, http.MethodPost, "account/fixed-loan/convert-to-market-loan", nil, &resp, request.AuthenticatedRequest)
}

// ReduceLiabilitiesForFixedLoan provide the function of "setting pending repay state / canceling pending repay state" for fixed loan order
func (ok *Okx) ReduceLiabilitiesForFixedLoan(ctx context.Context, orderID string, pendingRepay bool) (*ReduceLiabilities, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	arg := &struct {
		OrderID          string `json:"ordId"`
		PendingRepayment bool   `json:"pendingRepay,string"`
	}{
		OrderID:          orderID,
		PendingRepayment: pendingRepay,
	}
	var resp *ReduceLiabilities
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, reduceLiabilitiesForFixedLoanEPL, http.MethodPost, "account/fixed-loan/reduce-liabilities", arg, &resp, request.AuthenticatedRequest)
}

// GetFixedLoanBorrowOrderList retrieves fixed loan borrow order list
// State '1': Borrowing '2': Borrowed '3': Settled (Repaid) '4': Borrow failed '5': Overdue '6': Settling '7': Reborrowing '8': Pending repay
func (ok *Okx) GetFixedLoanBorrowOrderList(ctx context.Context, ccy currency.Code, orderID, state, term string, after, before time.Time, limit int64) ([]FixedLoanBorrowOrderDetail, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if state != "" {
		params.Set("state", state)
	}
	if term != "" {
		params.Set("term", term)
	}
	if !after.IsZero() && !before.IsZero() {
		err := common.StartEndTimeCheck(after, before)
		if err != nil {
			return nil, err
		}
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FixedLoanBorrowOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFixedLoanBorrowOrderListEPL, http.MethodGet, common.EncodeURLValues("account/fixed-loan/borrowing-orders-list", params), nil, &resp, request.AuthenticatedRequest)
}

// ManualBorrowOrRepay borrow or repay assets. only applicable to Spot mode (enabled borrowing)
func (ok *Okx) ManualBorrowOrRepay(ctx context.Context, ccy currency.Code, side string, amount float64) (*BorrowOrRepay, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if side == "" {
		return nil, errLendingSideRequired
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	arg := &struct {
		Currency string  `json:"ccy"`
		Side     string  `json:"side"`
		Amount   float64 `json:"amt"`
	}{
		Currency: ccy.String(),
		Side:     side,
		Amount:   amount,
	}
	var resp *BorrowOrRepay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, manualBorrowOrRepayEPL, http.MethodPost, "account/spot-manual-borrow-repay", arg, &resp, request.AuthenticatedRequest)
}

// SetAutoRepay represents an auto-repay. Only applicable to Spot mode (enabled borrowing)
func (ok *Okx) SetAutoRepay(ctx context.Context, autoRepay bool) (*AutoRepay, error) {
	arg := AutoRepay{
		AutoRepay: autoRepay,
	}
	var resp *AutoRepay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setAutoRepayEPL, http.MethodPost, "account/set-auto-repay", arg, &resp, request.AuthenticatedRequest)
}

// GetBorrowRepayHistory retrieve the borrow/repay history under Spot mode
func (ok *Okx) GetBorrowRepayHistory(ctx context.Context, ccy currency.Code, eventType string, after, before time.Time, limit int64) ([]BorrowRepayItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if eventType != "" {
		params.Set("type", eventType)
	}
	if !after.IsZero() && !before.IsZero() {
		err := common.StartEndTimeCheck(after, before)
		if err != nil {
			return nil, err
		}
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BorrowRepayItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBorrowRepayHistoryEPL, http.MethodGet, common.EncodeURLValues("account/spot-borrow-repay-history", params), nil, &resp, request.AuthenticatedRequest)
}

// NewPositionBuilder calculates portfolio margin information for virtual position/assets or current position of the user.
// You can add up to 200 virtual positions and 200 virtual assets in one request
func (ok *Okx) NewPositionBuilder(ctx context.Context, arg *PositionBuilderParam) (*PositionBuilderDetail, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	var resp *PositionBuilderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, newPositionBuilderEPL, http.MethodPost, "account/position-builder", arg, &resp, request.AuthenticatedRequest)
}

// SetRiskOffsetAmount set risk offset amount. This does not represent the actual spot risk offset amount. Only applicable to Portfolio Margin Mode
func (ok *Okx) SetRiskOffsetAmount(ctx context.Context, ccy currency.Code, clientSpotInUseAmount float64) (*RiskOffsetAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if clientSpotInUseAmount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	arg := &struct {
		Currency              string  `json:"ccy"`
		ClientSpotInUseAmount float64 `json:"clSpotInUseAmt,string"`
	}{
		Currency:              ccy.String(),
		ClientSpotInUseAmount: clientSpotInUseAmount,
	}
	var resp *RiskOffsetAmount
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setRiskOffsetAmountEPL, http.MethodPost, "account/set-riskOffset-amt", arg, &resp, request.AuthenticatedRequest)
}

// GetGreeks retrieves a greeks list of all assets in the account
func (ok *Okx) GetGreeks(ctx context.Context, ccy currency.Code) ([]GreeksItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []GreeksItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getGreeksEPL, http.MethodGet, common.EncodeURLValues("account/greeks", params), nil, &resp, request.AuthenticatedRequest)
}

// GetPMPositionLimitation retrieve cross position limitation of SWAP/FUTURES/OPTION under Portfolio margin mode
func (ok *Okx) GetPMPositionLimitation(ctx context.Context, instrumentType, underlying, instrumentFamily string) ([]PMLimitationResponse, error) {
	if instrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	if underlying == "" && instrumentFamily == "" {
		return nil, errInstrumentFamilyOrUnderlyingRequired
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(instrumentType))
	if underlying != "" {
		params.Set("uly", underlying)
	}
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	var resp []PMLimitationResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPMLimitationEPL, http.MethodGet, common.EncodeURLValues("account/position-tiers", params), nil, &resp, request.AuthenticatedRequest)
}

// SetRiskOffsetType configure the risk offset type in portfolio margin mode.
// riskOffsetType possible values are:
// 1: Spot-derivatives (USDT) risk offset
// 2: Spot-derivatives (Crypto) risk offset
// 3:Derivatives only mode
func (ok *Okx) SetRiskOffsetType(ctx context.Context, riskOffsetType string) (*RiskOffsetType, error) {
	if riskOffsetType == "" {
		return nil, errors.New("missing risk offset type")
	}
	var resp *RiskOffsetType
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setRiskOffsetLimiterEPL, http.MethodPost, "account/set-riskOffset-type", &map[string]string{"type": riskOffsetType}, &resp, request.AuthenticatedRequest)
}

// ActivateOption activates option
func (ok *Okx) ActivateOption(ctx context.Context) (types.Time, error) {
	resp := &tsResp{}
	return resp.Timestamp, ok.SendHTTPRequest(ctx, exchange.RestSpot, activateOptionEPL, http.MethodPost, "account/activate-option", nil, resp, request.AuthenticatedRequest)
}

// SetAutoLoan only applicable to Multi-currency margin and Portfolio margin
func (ok *Okx) SetAutoLoan(ctx context.Context, autoLoan bool) (*AutoLoan, error) {
	var resp *AutoLoan
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setAutoLoanEPL, http.MethodPost, "account/set-auto-loan", &AutoLoan{AutoLoan: autoLoan}, &resp, request.AuthenticatedRequest)
}

// SetAccountMode to set on the Web/App for the first set of every account mode.
// Account mode 1: Simple mode 2: Single-currency margin mode  3: Multi-currency margin code  4: Portfolio margin mode
func (ok *Okx) SetAccountMode(ctx context.Context, accountLevel string) (*AccountMode, error) {
	var resp *AccountMode
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setAccountLevelEPL, http.MethodPost, "account/set-account-level", &map[string]string{"acctLv": accountLevel}, &resp, request.AuthenticatedRequest)
}

// ResetMMPStatus reset the MMP status to be inactive.
// you can unfreeze by this endpoint once MMP is triggered.
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required
func (ok *Okx) ResetMMPStatus(ctx context.Context, instrumentType, instrumentFamily string) (*MMPStatusResponse, error) {
	if instrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	arg := &struct {
		InstrumentType   string `json:"instType,omitempty"`
		InstrumentFamily string `json:"instFamily"`
	}{
		InstrumentType:   instrumentType,
		InstrumentFamily: instrumentFamily,
	}
	var resp *MMPStatusResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, resetMMPStatusEPL, http.MethodPost, "account/mmp-reset", arg, &resp, request.AuthenticatedRequest)
}

// SetMMP set MMP configure
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required
func (ok *Okx) SetMMP(ctx context.Context, arg *MMPConfig) (*MMPConfig, error) {
	if *arg == (MMPConfig{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	if arg.QuantityLimit <= 0 {
		return nil, errInvalidQuantityLimit
	}
	var resp *MMPConfig
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setMMPEPL, http.MethodPost, "account/mmp-config", arg, &resp, request.AuthenticatedRequest)
}

// GetMMPConfig retrieves MMP configure information
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required
func (ok *Okx) GetMMPConfig(ctx context.Context, instrumentFamily string) ([]MMPConfigDetail, error) {
	params := url.Values{}
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	var resp []MMPConfigDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMMPConfigEPL, http.MethodGet, common.EncodeURLValues("account/mmp-config", params), nil, &resp, request.AuthenticatedRequest)
}

/********************************** Subaccount Endpoints ***************************************************/

// ViewSubAccountList applies to master accounts only
func (ok *Okx) ViewSubAccountList(ctx context.Context, enable bool, subaccountName string, after, before time.Time, limit int64) ([]SubaccountInfo, error) {
	params := url.Values{}
	if enable {
		params.Set("enable", "true")
	}
	if subaccountName != "" {
		params.Set("subAcct", subaccountName)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubaccountInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, viewSubaccountListEPL, http.MethodGet, common.EncodeURLValues("users/subaccount/list", params), nil, &resp, request.AuthenticatedRequest)
}

// ResetSubAccountAPIKey applies to master accounts only and master accounts APIKey must be linked to IP addresses
func (ok *Okx) ResetSubAccountAPIKey(ctx context.Context, arg *SubAccountAPIKeyParam) (*SubAccountAPIKeyResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.SubAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	if arg.APIKey == "" {
		return nil, errInvalidAPIKey
	}
	if arg.IP != "" && net.ParseIP(arg.IP).To4() == nil {
		return nil, errInvalidIPAddress
	}
	if arg.APIKeyPermission == "" && len(arg.Permissions) != 0 {
		for x := range arg.Permissions {
			if !slices.Contains([]string{"read", "withdraw", "trade", "read_only"}, arg.Permissions[x]) {
				return nil, errInvalidAPIKeyPermission
			}
			if x != 0 {
				arg.APIKeyPermission += ","
			}
			arg.APIKeyPermission += arg.Permissions[x]
		}
	} else if !slices.Contains([]string{"read", "withdraw", "trade", "read_only"}, arg.APIKeyPermission) {
		return nil, errInvalidAPIKeyPermission
	}
	var resp *SubAccountAPIKeyResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, resetSubAccountAPIKeyEPL, http.MethodPost, "users/subaccount/modify-apikey", &arg, &resp, request.AuthenticatedRequest)
}

// GetSubaccountTradingBalance query detailed balance info of Trading Account of a sub-account via the master account (applies to master accounts only)
func (ok *Okx) GetSubaccountTradingBalance(ctx context.Context, subaccountName string) ([]SubaccountBalanceResponse, error) {
	if subaccountName == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subAcct", subaccountName)
	var resp []SubaccountBalanceResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSubaccountTradingBalanceEPL, http.MethodGet, common.EncodeURLValues("account/subaccount/balances", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSubaccountFundingBalance query detailed balance info of Funding Account of a sub-account via the master account (applies to master accounts only)
func (ok *Okx) GetSubaccountFundingBalance(ctx context.Context, subaccountName string, ccy currency.Code) ([]FundingBalance, error) {
	if subaccountName == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subAcct", subaccountName)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []FundingBalance
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSubaccountFundingBalanceEPL, http.MethodGet,
		common.EncodeURLValues("asset/subaccount/balances", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSubAccountMaximumWithdrawal retrieve the maximum withdrawal information of a sub-account via the master account (applies to master accounts only). If no currency is specified, the transferable amount of all owned currencies will be returned
func (ok *Okx) GetSubAccountMaximumWithdrawal(ctx context.Context, subAccountName string, ccy currency.Code) ([]SubAccountMaximumWithdrawal, error) {
	if subAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subAcct", subAccountName)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []SubAccountMaximumWithdrawal
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSubAccountMaxWithdrawalEPL, http.MethodGet,
		common.EncodeURLValues("account/subaccount/max-withdrawal", params), nil, &resp, request.AuthenticatedRequest)
}

// HistoryOfSubaccountTransfer retrieves subaccount transfer histories; applies to master accounts only.
// retrieve the transfer data for the last 3 months
func (ok *Okx) HistoryOfSubaccountTransfer(ctx context.Context, ccy currency.Code, subaccountType, subaccountName string, before, after time.Time, limit int64) ([]SubaccountBillItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if subaccountType != "" {
		params.Set("type", subaccountType)
	}
	if subaccountName != "" {
		params.Set("subacct", subaccountName)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubaccountBillItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, historyOfSubaccountTransferEPL, http.MethodGet, common.EncodeURLValues("asset/subaccount/bills", params), nil, &resp, request.AuthenticatedRequest)
}

// GetHistoryOfManagedSubAccountTransfer retrieves managed sub-account transfers.
// nly applicable to the trading team's master account to getting transfer records of managed sub accounts entrusted to oneself
func (ok *Okx) GetHistoryOfManagedSubAccountTransfer(ctx context.Context, ccy currency.Code, transferType, subAccountName, subAccountUID string, after, before time.Time, limit int64) ([]SubAccountTransfer, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if transferType != "" {
		params.Set("type", transferType)
	}
	if subAccountName != "" {
		params.Set("subAcct", subAccountName)
	}
	if subAccountUID != "" {
		params.Set("subUid", subAccountUID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubAccountTransfer
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, managedSubAccountTransferEPL, http.MethodGet, common.EncodeURLValues("asset/subaccount/managed-subaccount-bills", params), nil, &resp, request.AuthenticatedRequest)
}

// MasterAccountsManageTransfersBetweenSubaccounts master accounts manage the transfers between sub-accounts applies to master accounts only
func (ok *Okx) MasterAccountsManageTransfersBetweenSubaccounts(ctx context.Context, arg *SubAccountAssetTransferParams) ([]TransferIDInfo, error) {
	if *arg == (SubAccountAssetTransferParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.From == 0 {
		return nil, errInvalidSubaccount
	}
	if arg.To != 6 && arg.To != 18 {
		return nil, errInvalidSubaccount
	}
	if arg.FromSubAccount == "" {
		return nil, errInvalidSubAccountName
	}
	if arg.ToSubAccount == "" {
		return nil, errInvalidSubAccountName
	}
	var resp []TransferIDInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, masterAccountsManageTransfersBetweenSubaccountEPL, http.MethodPost, "asset/subaccount/transfer", &arg, &resp, request.AuthenticatedRequest)
}

// SetPermissionOfTransferOut set permission of transfer out for sub-account(only applicable to master account). Sub-account can transfer out to master account by default
func (ok *Okx) SetPermissionOfTransferOut(ctx context.Context, arg *PermissionOfTransfer) ([]PermissionOfTransfer, error) {
	if *arg == (PermissionOfTransfer{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.SubAcct == "" {
		return nil, errInvalidSubAccountName
	}
	var resp []PermissionOfTransfer
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setPermissionOfTransferOutEPL, http.MethodPost, "users/subaccount/set-transfer-out", &arg, &resp, request.AuthenticatedRequest)
}

// GetCustodyTradingSubaccountList the trading team uses this interface to view the list of sub-accounts currently under escrow
// usersEntrustSubaccountList ="users/entrust-subaccount-list"
func (ok *Okx) GetCustodyTradingSubaccountList(ctx context.Context, subaccountName string) ([]SubaccountName, error) {
	params := url.Values{}
	if subaccountName != "" {
		params.Set("setAcct", subaccountName)
	}
	var resp []SubaccountName
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getCustodyTradingSubaccountListEPL, http.MethodGet, common.EncodeURLValues("users/entrust-subaccount-list", params), nil, &resp, request.AuthenticatedRequest)
}

// SetSubAccountVIPLoanAllocation set the VIP loan allocation of sub-accounts. Only Applicable to master account API keys with Trade access
func (ok *Okx) SetSubAccountVIPLoanAllocation(ctx context.Context, arg *SubAccountLoanAllocationParam) (bool, error) {
	if len(arg.Alloc) == 0 {
		return false, common.ErrEmptyParams
	}
	for a := range arg.Alloc {
		if arg.Alloc[a] == (subAccountVIPLoanAllocationInfo{}) {
			return false, common.ErrEmptyParams
		}
		if arg.Alloc[a].SubAcct == "" {
			return false, errInvalidSubAccountName
		}
		if arg.Alloc[a].LoanAlloc < 0 {
			return false, errInvalidLoanAllocationValue
		}
	}
	resp := &struct {
		Result bool `json:"result"`
	}{}
	return resp.Result, ok.SendHTTPRequest(ctx, exchange.RestSpot, setSubAccountVIPLoanAllocationEPL, http.MethodPost, "account/subaccount/set-loan-allocation", arg, resp, request.AuthenticatedRequest)
}

// GetSubAccountBorrowInterestAndLimit retrieves sub-account borrow interest and limit
// Only applicable to master account API keys. Only return VIP loan information
func (ok *Okx) GetSubAccountBorrowInterestAndLimit(ctx context.Context, subAccount string, ccy currency.Code) ([]SubAccounBorrowInterestAndLimit, error) {
	if subAccount == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subAcct", subAccount)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []SubAccounBorrowInterestAndLimit
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSubAccountBorrowInterestAndLimitEPL, http.MethodGet, common.EncodeURLValues("account/subaccount/interest-limits", params), nil, &resp, request.AuthenticatedRequest)
}

/*************************************** Grid Trading Endpoints ***************************************************/

// PlaceGridAlgoOrder place spot grid algo order
func (ok *Okx) PlaceGridAlgoOrder(ctx context.Context, arg *GridAlgoOrder) (*GridAlgoOrderIDResponse, error) {
	if *arg == (GridAlgoOrder{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	arg.AlgoOrdType = strings.ToLower(arg.AlgoOrdType)
	if arg.AlgoOrdType != AlgoOrdTypeGrid && arg.AlgoOrdType != AlgoOrdTypeContractGrid {
		return nil, errMissingAlgoOrderType
	}
	if arg.MaxPrice <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if arg.MinPrice <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if arg.GridQuantity < 0 {
		return nil, errInvalidGridQuantity
	}
	isSpotGridOrder := arg.QuoteSize > 0 || arg.BaseSize > 0
	if !isSpotGridOrder {
		if arg.Size <= 0 {
			return nil, fmt.Errorf("%w: parameter Size is required", order.ErrAmountMustBeSet)
		}
		arg.Direction = strings.ToLower(arg.Direction)
		if !slices.Contains([]string{positionSideLong, positionSideShort, "neutral"}, arg.Direction) {
			return nil, errMissingRequiredArgumentDirection
		}
		if arg.Leverage == "" {
			return nil, errInvalidLeverage
		}
	}
	var resp *GridAlgoOrderIDResponse
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, gridTradingEPL, http.MethodPost, "tradingBot/grid/order-algo", &arg, &resp, request.AuthenticatedRequest)
	if err != nil {
		if resp != nil && resp.StatusMessage != "" {
			return nil, fmt.Errorf("%w; %w", err, getStatusError(resp.StatusCode, resp.StatusMessage))
		}
		return nil, err
	}
	return resp, nil
}

// AmendGridAlgoOrder supported contract grid algo order amendment
func (ok *Okx) AmendGridAlgoOrder(ctx context.Context, arg *GridAlgoOrderAmend) (*GridAlgoOrderIDResponse, error) {
	if *arg == (GridAlgoOrderAmend{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	var resp *GridAlgoOrderIDResponse
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, amendGridAlgoOrderEPL, http.MethodPost, "tradingBot/grid/amend-order-algo", &arg, &resp, request.AuthenticatedRequest)
	if err != nil {
		if resp != nil && resp.StatusMessage == "" {
			return nil, fmt.Errorf("%w; %w", err, getStatusError(resp.StatusCode, resp.StatusMessage))
		}
		return nil, err
	}
	return resp, nil
}

// StopGridAlgoOrder stop a batch of grid algo orders.
// A maximum of 10 orders can be canceled per request
func (ok *Okx) StopGridAlgoOrder(ctx context.Context, arg []StopGridAlgoOrderRequest) ([]GridAlgoOrderIDResponse, error) {
	if len(arg) == 0 {
		return nil, common.ErrEmptyParams
	}
	for x := range arg {
		if (arg[x]) == (StopGridAlgoOrderRequest{}) {
			return nil, common.ErrEmptyParams
		}
		if arg[x].AlgoID == "" {
			return nil, errAlgoIDRequired
		}
		if arg[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		arg[x].AlgoOrderType = strings.ToLower(arg[x].AlgoOrderType)
		if arg[x].AlgoOrderType != AlgoOrdTypeGrid && arg[x].AlgoOrderType != AlgoOrdTypeContractGrid {
			return nil, errMissingAlgoOrderType
		}
		if arg[x].StopType != 1 && arg[x].StopType != 2 {
			return nil, errMissingValidStopType
		}
	}
	var resp []GridAlgoOrderIDResponse
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, stopGridAlgoOrderEPL, http.MethodPost, "tradingBot/grid/stop-order-algo", arg, &resp, request.AuthenticatedRequest)
	if err != nil {
		if len(resp) == 0 {
			return nil, err
		}
		var errs error
		for x := range resp {
			if resp[x].StatusMessage != "" {
				errs = common.AppendError(errs, getStatusError(resp[x].StatusCode, resp[x].StatusMessage))
			}
		}
		return nil, common.AppendError(err, errs)
	}
	return resp, nil
}

// ClosePositionForContractID close position when the contract grid stop type is 'keep position'
func (ok *Okx) ClosePositionForContractID(ctx context.Context, arg *ClosePositionParams) (*ClosePositionContractGridResponse, error) {
	if *arg == (ClosePositionParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if !arg.MarketCloseAllPositions && arg.Size <= 0 {
		return nil, fmt.Errorf("%w 'size' is required", order.ErrAmountMustBeSet)
	}
	if !arg.MarketCloseAllPositions && arg.Price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	var resp *ClosePositionContractGridResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, closePositionForForContractGridEPL, http.MethodPost, "tradingBot/grid/close-position", arg, &resp, request.AuthenticatedRequest)
}

// CancelClosePositionOrderForContractGrid cancels close position order for contract grid
func (ok *Okx) CancelClosePositionOrderForContractGrid(ctx context.Context, arg *CancelClosePositionOrder) (*ClosePositionContractGridResponse, error) {
	if *arg == (CancelClosePositionOrder{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *ClosePositionContractGridResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelClosePositionOrderForContractGridEPL, http.MethodPost, "tradingBot/grid/cancel-close-order", arg, &resp, request.AuthenticatedRequest)
}

// InstantTriggerGridAlgoOrder triggers grid algo order
func (ok *Okx) InstantTriggerGridAlgoOrder(ctx context.Context, algoID string) (*TriggeredGridAlgoOrderInfo, error) {
	var resp *TriggeredGridAlgoOrderInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, instantTriggerGridAlgoOrderEPL, http.MethodPost, "tradingBot/grid/order-instant-trigger", &map[string]string{"algoId": algoID}, &resp, request.AuthenticatedRequest)
}

// GetGridAlgoOrdersList retrieves list of pending grid algo orders with the complete data
func (ok *Okx) GetGridAlgoOrdersList(ctx context.Context, algoOrderType, algoID,
	instrumentID, instrumentType,
	after, before string, limit int64,
) ([]GridAlgoOrderResponse, error) {
	return ok.getGridAlgoOrders(ctx, algoOrderType, algoID,
		instrumentID, instrumentType,
		after, before, "tradingBot/grid/orders-algo-pending", limit)
}

// GetGridAlgoOrderHistory retrieves list of grid algo orders with the complete data including the stopped orders
func (ok *Okx) GetGridAlgoOrderHistory(ctx context.Context, algoOrderType, algoID,
	instrumentID, instrumentType,
	after, before string, limit int64,
) ([]GridAlgoOrderResponse, error) {
	return ok.getGridAlgoOrders(ctx, algoOrderType, algoID,
		instrumentID, instrumentType,
		after, before, "tradingBot/grid/orders-algo-history", limit)
}

// getGridAlgoOrderList retrieves list of grid algo orders with the complete data
func (ok *Okx) getGridAlgoOrders(ctx context.Context, algoOrderType, algoID,
	instrumentID, instrumentType,
	after, before, route string, limit int64,
) ([]GridAlgoOrderResponse, error) {
	algoOrderType = strings.ToLower(algoOrderType)
	if algoOrderType != AlgoOrdTypeGrid && algoOrderType != AlgoOrdTypeContractGrid {
		return nil, errMissingAlgoOrderType
	}
	params := url.Values{}
	params.Set("algoOrdType", algoOrderType)
	if algoID != "" {
		params.Set("algoId", algoID)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if instrumentType != "" {
		params.Set("instType", strings.ToUpper(instrumentType))
	}
	if after != "" {
		params.Set("after", after)
	}
	if before != "" {
		params.Set("before", before)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	epl := getGridAlgoOrderListEPL
	if route == "tradingBot/grid/orders-algo-history" {
		epl = getGridAlgoOrderHistoryEPL
	}
	var resp []GridAlgoOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, request.AuthenticatedRequest)
}

// GetGridAlgoOrderDetails retrieves grid algo order details
func (ok *Okx) GetGridAlgoOrderDetails(ctx context.Context, algoOrderType, algoID string) (*GridAlgoOrderResponse, error) {
	if algoOrderType != AlgoOrdTypeGrid &&
		algoOrderType != AlgoOrdTypeContractGrid {
		return nil, errMissingAlgoOrderType
	}
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	params := url.Values{}
	params.Set("algoOrdType", algoOrderType)
	params.Set("algoId", algoID)
	var resp *GridAlgoOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoOrderDetailsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/orders-algo-details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetGridAlgoSubOrders retrieves grid algo sub orders
func (ok *Okx) GetGridAlgoSubOrders(ctx context.Context, algoOrderType, algoID, subOrderType, groupID, after, before string, limit int64) ([]GridAlgoOrderResponse, error) {
	if algoOrderType != AlgoOrdTypeGrid &&
		algoOrderType != AlgoOrdTypeContractGrid {
		return nil, errMissingAlgoOrderType
	}
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	if subOrderType != "live" && subOrderType != order.Filled.String() {
		return nil, errMissingSubOrderType
	}
	params := url.Values{}
	params.Set("algoOrdType", algoOrderType)
	params.Set("algoId", algoID)
	params.Set("type", subOrderType)
	if groupID != "" {
		params.Set("groupId", groupID)
	}
	if after != "" {
		params.Set("after", after)
	}
	if before != "" {
		params.Set("before", before)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []GridAlgoOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoSubOrdersEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/sub-orders", params), nil, &resp, request.AuthenticatedRequest)
}

// GetGridAlgoOrderPositions retrieves grid algo order positions
func (ok *Okx) GetGridAlgoOrderPositions(ctx context.Context, algoOrderType, algoID string) ([]AlgoOrderPosition, error) {
	if algoOrderType != AlgoOrdTypeGrid && algoOrderType != AlgoOrdTypeContractGrid {
		return nil, errInvalidAlgoOrderType
	}
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	params := url.Values{}
	params.Set("algoOrdType", algoOrderType)
	params.Set("algoId", algoID)
	var resp []AlgoOrderPosition
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoOrderPositionsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/positions", params), nil, &resp, request.AuthenticatedRequest)
}

// SpotGridWithdrawProfit returns the spot grid orders withdrawal profit given an instrument id
func (ok *Okx) SpotGridWithdrawProfit(ctx context.Context, algoID string) (*AlgoOrderWithdrawalProfit, error) {
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	input := &struct {
		AlgoID string `json:"algoId"`
	}{
		AlgoID: algoID,
	}
	var resp *AlgoOrderWithdrawalProfit
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, spotGridWithdrawIncomeEPL, http.MethodPost, "tradingBot/grid/withdraw-income", input, &resp, request.AuthenticatedRequest)
}

// ComputeMarginBalance computes margin balance with 'add' and 'reduce' balance type
func (ok *Okx) ComputeMarginBalance(ctx context.Context, arg MarginBalanceParam) (*ComputeMarginBalance, error) {
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AdjustMarginBalanceType != "add" && arg.AdjustMarginBalanceType != marginBalanceReduce {
		return nil, errInvalidMarginTypeAdjust
	}
	var resp *ComputeMarginBalance
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, computeMarginBalanceEPL, http.MethodPost, "tradingBot/grid/compute-margin-balance", &arg, &resp, request.AuthenticatedRequest)
}

// AdjustMarginBalance retrieves adjust margin balance with 'add' and 'reduce' balance type
func (ok *Okx) AdjustMarginBalance(ctx context.Context, arg *MarginBalanceParam) (*AdjustMarginBalanceResponse, error) {
	if *arg == (MarginBalanceParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.AdjustMarginBalanceType != "add" && arg.AdjustMarginBalanceType != marginBalanceReduce {
		return nil, errInvalidMarginTypeAdjust
	}
	if arg.Percentage <= 0 && arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, either percentage or amount is required", order.ErrAmountIsInvalid)
	}
	var resp *AdjustMarginBalanceResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, adjustMarginBalanceEPL, http.MethodPost, "tradingBot/grid/margin-balance", &arg, &resp, request.AuthenticatedRequest)
}

// GetGridAIParameter retrieves grid AI parameter
func (ok *Okx) GetGridAIParameter(ctx context.Context, algoOrderType, instrumentID, direction, duration string) ([]GridAIParameterResponse, error) {
	if !slices.Contains([]string{"moon_grid", "contract_grid", "grid"}, algoOrderType) {
		return nil, errInvalidAlgoOrderType
	}
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if algoOrderType == "contract_grid" && !slices.Contains([]string{positionSideLong, positionSideShort, "neutral"}, direction) {
		return nil, fmt.Errorf("%w, required for 'contract_grid' algo order type", errMissingRequiredArgumentDirection)
	}
	params := url.Values{}
	params.Set("direction", direction)
	params.Set("algoOrdType", algoOrderType)
	params.Set("instId", instrumentID)
	if !slices.Contains([]string{"", "7D", "30D", "180D"}, duration) {
		return nil, errInvalidDuration
	}
	var resp []GridAIParameterResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getGridAIParameterEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/ai-param", params), nil, &resp, request.UnauthenticatedRequest)
}

// ComputeMinInvestment computes minimum investment
func (ok *Okx) ComputeMinInvestment(ctx context.Context, arg *ComputeInvestmentDataParam) (*InvestmentResult, error) {
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	switch arg.AlgoOrderType {
	case "grid", "contract_grid":
	default:
		return nil, errInvalidAlgoOrderType
	}
	if arg.MaxPrice <= 0 {
		return nil, fmt.Errorf("%w, maxPrice = %f", order.ErrPriceBelowMin, arg.MaxPrice)
	}
	if arg.MinPrice <= 0 {
		return nil, fmt.Errorf("%w, minPrice = %f", order.ErrPriceBelowMin, arg.MaxPrice)
	}
	if arg.GridNumber == 0 {
		return nil, fmt.Errorf("%w, grid number is required", errInvalidGridQuantity)
	}
	if arg.RunType == "" {
		return nil, errRunTypeRequired
	}
	if arg.AlgoOrderType == "contract_grid" {
		switch arg.Direction {
		case positionSideLong, positionSideShort, "neutral":
		default:
			return nil, fmt.Errorf("%w, invalid grid direction %s", errMissingRequiredArgumentDirection, arg.Direction)
		}
		if arg.Leverage <= 0 {
			return nil, errInvalidLeverage
		}
	}
	for x := range arg.InvestmentData {
		if arg.InvestmentData[x].Amount <= 0 {
			return nil, fmt.Errorf("%w, investment amt = %f", order.ErrAmountBelowMin, arg.InvestmentData[x].Amount)
		}
		if arg.InvestmentData[x].Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
	}
	var resp *InvestmentResult
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, computeMinInvestmentEPL, http.MethodPost, "tradingBot/grid/min-investment", arg, &resp, request.UnauthenticatedRequest)
}

// RSIBackTesting relative strength index(RSI) backtesting
// Parameters:
//
//	TriggerCondition: possible values are "cross_up" "cross_down" "above" "below" "cross" Default is cross_down
//
// Threshold: The value should be an integer between 1 to 100
func (ok *Okx) RSIBackTesting(ctx context.Context, instrumentID, triggerCondition, duration string, threshold, timePeriod int64, timeFrame kline.Interval) (*RSIBacktestingResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if threshold > 100 || threshold < 1 {
		return nil, errors.New("threshold should be an integer between 1 to 100")
	}
	timeFrameString := IntervalFromString(timeFrame, false)
	if timeFrameString == "" {
		return nil, errors.New("timeframe is required")
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("timeframe", timeFrameString)
	params.Set("thold", strconv.FormatInt(threshold, 10))
	if timePeriod > 0 {
		params.Set("timePeriod", strconv.FormatInt(timePeriod, 10))
	}
	if triggerCondition != "" {
		params.Set("triggerCond", triggerCondition)
	}
	if duration != "" {
		params.Set("duration", duration)
	}
	var resp *RSIBacktestingResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rsiBackTestingEPL, http.MethodGet, common.EncodeURLValues("tradingBot/public/rsi-back-testing", params), nil, &resp, request.UnauthenticatedRequest)
}

// ****************************************** Signal bot trading **************************************************

// GetSignalBotOrderDetail create and customize your own signals while gaining access to a diverse selection of signals from top providers.
// Empower your trading strategies and stay ahead of the game with our comprehensive signal trading platform
func (ok *Okx) GetSignalBotOrderDetail(ctx context.Context, algoOrderType, algoID string) (*SignalBotOrderDetail, error) {
	if algoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	params.Set("algoOrdType", algoOrderType)
	var resp *SignalBotOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, signalBotOrderDetailsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/orders-algo-details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSignalOrderPositions retrieves signal bot order positions
func (ok *Okx) GetSignalOrderPositions(ctx context.Context, algoOrderType, algoID string) (*SignalBotPosition, error) {
	if algoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	params.Set("algoOrdType", algoOrderType)
	var resp *SignalBotPosition
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, signalBotOrderPositionsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/positions", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSignalBotSubOrders retrieves historical filled sub orders and designated sub orders
func (ok *Okx) GetSignalBotSubOrders(ctx context.Context, algoID, algoOrderType, subOrderType, clientOrderID, afterPaginationID, beforePaginationID string, begin, end time.Time, limit int64) ([]SubOrder, error) {
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	if algoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if subOrderType == "" && clientOrderID == "" {
		return nil, fmt.Errorf("%w, either client order ID or sub-order state is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	params.Set("algoOrdType", algoOrderType)
	if subOrderType != "" {
		params.Set("type", subOrderType)
	} else {
		params.Set("clOrdId", clientOrderID)
	}
	if afterPaginationID != "" {
		params.Set("after", afterPaginationID)
	}
	if beforePaginationID != "" {
		params.Set("before", beforePaginationID)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, signalBotSubOrdersEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/sub-orders", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSignalBotEventHistory retrieves signal bot event history
func (ok *Okx) GetSignalBotEventHistory(ctx context.Context, algoID string, after, before time.Time, limit int64) ([]SignalBotEventHistory, error) {
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SignalBotEventHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, signalBotEventHistoryEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/event-history", params), nil, &resp, request.AuthenticatedRequest)
}

// ****************************************** Recurring Buy *****************************************

// PlaceRecurringBuyOrder recurring buy is a strategy for investing a fixed amount in crypto at fixed intervals.
// An appropriate recurring approach in volatile markets allows you to buy crypto at lower costs. Learn more
// The API endpoints of Recurring buy require authentication
func (ok *Okx) PlaceRecurringBuyOrder(ctx context.Context, arg *PlaceRecurringBuyOrderParam) (*RecurringOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.StrategyName == "" {
		return nil, errStrategyNameRequired
	}
	if len(arg.RecurringList) == 0 {
		return nil, fmt.Errorf("%w, no recurring list is provided", common.ErrEmptyParams)
	}
	for x := range arg.RecurringList {
		if arg.RecurringList[x].Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
	}
	if arg.RecurringDay == "" {
		return nil, errRecurringDayRequired
	}
	if arg.RecurringTime > 23 || arg.RecurringTime < 0 {
		return nil, errRecurringBuyTimeRequired
	}
	if arg.TradeMode == "" {
		return nil, errInvalidTradeModeValue
	}
	var resp *RecurringOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeRecurringBuyOrderEPL, http.MethodPost, "tradingBot/recurring/order-algo", arg, &resp, request.AuthenticatedRequest)
}

// AmendRecurringBuyOrder amends recurring order
func (ok *Okx) AmendRecurringBuyOrder(ctx context.Context, arg *AmendRecurringOrderParam) (*RecurringOrderResponse, error) {
	if (*arg) == (AmendRecurringOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.AlgoID == "" {
		return nil, errAlgoIDRequired
	}
	if arg.StrategyName == "" {
		return nil, errStrategyNameRequired
	}
	var resp *RecurringOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendRecurringBuyOrderEPL, http.MethodPost, "tradingBot/recurring/amend-order-algo", arg, &resp, request.AuthenticatedRequest)
}

// StopRecurringBuyOrder stops recurring buy order. A maximum of 10 orders can be stopped per request
func (ok *Okx) StopRecurringBuyOrder(ctx context.Context, arg []StopRecurringBuyOrder) ([]RecurringOrderResponse, error) {
	if len(arg) == 0 {
		return nil, common.ErrEmptyParams
	}
	for x := range arg {
		if arg[x].AlgoID == "" {
			return nil, errAlgoIDRequired
		}
	}
	var resp []RecurringOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, stopRecurringBuyOrderEPL, http.MethodPost, "tradingBot/recurring/stop-order-algo", arg, &resp, request.AuthenticatedRequest)
}

// GetRecurringBuyOrderList retrieves recurring buy order list
func (ok *Okx) GetRecurringBuyOrderList(ctx context.Context, algoID, algoOrderState string, after, before time.Time, limit int64) ([]RecurringOrderItem, error) {
	params := url.Values{}
	if algoOrderState != "" {
		params.Set("state", algoOrderState)
	}
	if algoID != "" {
		params.Set("algoId", algoID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []RecurringOrderItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuyOrderListEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/orders-algo-pending", params), nil, &resp, request.AuthenticatedRequest)
}

// GetRecurringBuyOrderHistory retrieves recurring buy order history
func (ok *Okx) GetRecurringBuyOrderHistory(ctx context.Context, algoID string, after, before time.Time, limit int64) ([]RecurringOrderItem, error) {
	params := url.Values{}
	if algoID != "" {
		params.Set("algoId", algoID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []RecurringOrderItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuyOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/orders-algo-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetRecurringOrderDetails retrieves a single recurring order detail
func (ok *Okx) GetRecurringOrderDetails(ctx context.Context, algoID, algoOrderState string) (*RecurringOrderDeail, error) {
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	if algoOrderState != "" {
		params.Set("state", algoOrderState)
	}
	var resp *RecurringOrderDeail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuyOrderDetailEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/orders-algo-details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetRecurringSubOrders retrieves recurring buy sub orders
func (ok *Okx) GetRecurringSubOrders(ctx context.Context, algoID, orderID string, after, before time.Time, limit int64) ([]RecurringBuySubOrder, error) {
	if algoID == "" {
		return nil, errAlgoIDRequired
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []RecurringBuySubOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuySubOrdersEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/sub-orders", params), nil, &resp, request.AuthenticatedRequest)
}

// ****************************************** Earn **************************************************

// GetExistingLeadingPositions retrieves leading positions that are not closed
func (ok *Okx) GetExistingLeadingPositions(ctx context.Context, instrumentType, instrumentID string, before, after time.Time, limit int64) ([]PositionInfo, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PositionInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getExistingLeadingPositionsEPL, http.MethodGet, common.EncodeURLValues("copytrading/current-subpositions", params), nil, &resp, request.AuthenticatedRequest)
}

// GetLeadingPositionsHistory leading trader retrieves the completed leading position of the last 3 months.
// Returns reverse chronological order with subPosId
func (ok *Okx) GetLeadingPositionsHistory(ctx context.Context, instrumentType, instrumentID string, before, after time.Time, limit int64) ([]PositionInfo, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PositionInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadingPositionHistoryEPL, http.MethodGet, common.EncodeURLValues("copytrading/subpositions-history", params), nil, &resp, request.AuthenticatedRequest)
}

// PlaceLeadingStopOrder holds leading trader sets TP/SL for the current leading position that are not closed
func (ok *Okx) PlaceLeadingStopOrder(ctx context.Context, arg *TPSLOrderParam) (*PositionIDInfo, error) {
	if *arg == (TPSLOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.SubPositionID == "" {
		return nil, errSubPositionIDRequired
	}
	var resp *PositionIDInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeLeadingStopOrderEPL, http.MethodPost, "copytrading/algo-order", arg, &resp, request.AuthenticatedRequest)
}

// CloseLeadingPosition close a leading position once a time
func (ok *Okx) CloseLeadingPosition(ctx context.Context, arg *CloseLeadingPositionParam) (*PositionIDInfo, error) {
	if *arg == (CloseLeadingPositionParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.SubPositionID == "" {
		return nil, errSubPositionIDRequired
	}
	var resp *PositionIDInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, closeLeadingPositionEPL, http.MethodPost, "copytrading/close-subposition", arg, &resp, request.AuthenticatedRequest)
}

// GetLeadingInstrument retrieves leading instruments
func (ok *Okx) GetLeadingInstrument(ctx context.Context, instrumentType string) ([]LeadingInstrumentItem, error) {
	params := url.Values{}
	if instrumentType == "" {
		params.Set("instType", instrumentType)
	}
	var resp []LeadingInstrumentItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadingInstrumentsEPL, http.MethodGet, common.EncodeURLValues("copytrading/instruments", params), nil, &resp, request.AuthenticatedRequest)
}

// AmendLeadingInstruments amend current leading instruments, need to set initial leading instruments while applying to become a leading trader.
// All non-leading contracts can't have position or pending orders for the current request when setting non-leading contracts as leading contracts
func (ok *Okx) AmendLeadingInstruments(ctx context.Context, instrumentID, instrumentType string) ([]LeadingInstrumentItem, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	var resp []LeadingInstrumentItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadingInstrumentsEPL, http.MethodPost, "copytrading/set-instruments", &struct {
		InstrumentType string `json:"instType,omitempty"`
		InstrumentID   string `json:"instId"`
	}{
		InstrumentID:   instrumentID,
		InstrumentType: instrumentType,
	}, &resp, request.AuthenticatedRequest)
}

// GetProfitSharingDetails gets profits shared details for the last 3 months
func (ok *Okx) GetProfitSharingDetails(ctx context.Context, instrumentType string, before, after time.Time, limit int64) ([]ProfitSharingItem, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []ProfitSharingItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getProfitSharingLimitEPL, http.MethodGet, common.EncodeURLValues("copytrading/profit-sharing-details", params), nil, &resp, request.AuthenticatedRequest)
}

// GetTotalProfitSharing gets the total amount of profit shared since joining the platform.
// Instrument type 'SPOT' 'SWAP' It returns all types by default
func (ok *Okx) GetTotalProfitSharing(ctx context.Context, instrumentType string) ([]TotalProfitSharing, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []TotalProfitSharing
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTotalProfitSharingEPL, http.MethodGet, common.EncodeURLValues("copytrading/total-profit-sharing", params), nil, &resp, request.AuthenticatedRequest)
}

// GetUnrealizedProfitSharingDetails gets leading trader gets the profit sharing details that are expected to be shared in the next settlement cycle.
// The unrealized profit sharing details will update once there copy position is closed
func (ok *Okx) GetUnrealizedProfitSharingDetails(ctx context.Context, instrumentType string) ([]ProfitSharingItem, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []ProfitSharingItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTotalProfitSharingEPL, http.MethodGet, common.EncodeURLValues("copytrading/unrealized-profit-sharing-details", params), nil, &resp, request.AuthenticatedRequest)
}

// SetFirstCopySettings set first copy settings for the certain lead trader. You need to first copy settings after stopping copying
func (ok *Okx) SetFirstCopySettings(ctx context.Context, arg *FirstCopySettings) (*ResponseResult, error) {
	err := validateFirstCopySettings(arg)
	if err != nil {
		return nil, err
	}
	var resp *ResponseResult
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setFirstCopySettingsEPL, http.MethodPost, "copytrading/first-copy-settings", arg, &resp, request.AuthenticatedRequest)
}

// AmendCopySettings amends need to use this endpoint for amending copy settings
func (ok *Okx) AmendCopySettings(ctx context.Context, arg *FirstCopySettings) (*ResponseResult, error) {
	err := validateFirstCopySettings(arg)
	if err != nil {
		return nil, err
	}
	var resp *ResponseResult
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendFirstCopySettingsEPL, http.MethodPost, "copytrading/amend-copy-settings", arg, &resp, request.AuthenticatedRequest)
}

func validateFirstCopySettings(arg *FirstCopySettings) error {
	if *arg == (FirstCopySettings{}) {
		return common.ErrEmptyParams
	}
	if arg.UniqueCode == "" {
		return errUniqueCodeRequired
	}
	if arg.CopyInstrumentIDType == "" {
		return errCopyInstrumentIDTypeRequired
	}
	if arg.CopyTotalAmount <= 0 {
		return fmt.Errorf("%w, copyTotalAmount value %f", order.ErrAmountBelowMin, arg.CopyTotalAmount)
	}
	if arg.SubPosCloseType == "" {
		return errSubPositionCloseTypeRequired
	}
	return nil
}

// StopCopying need to use this endpoint for amending copy settings
func (ok *Okx) StopCopying(ctx context.Context, arg *StopCopyingParameter) (*ResponseResult, error) {
	if *arg == (StopCopyingParameter{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.UniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	if arg.SubPositionCloseType == "" {
		return nil, errSubPositionCloseTypeRequired
	}
	var resp *ResponseResult
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, stopCopyingEPL, http.MethodPost, "copytrading/stop-copy-trading", arg, &resp, request.AuthenticatedRequest)
}

// GetCopySettings retrieve the copy settings about certain lead trader
func (ok *Okx) GetCopySettings(ctx context.Context, instrumentType, uniqueCode string) (*CopySetting, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	if instrumentType == "" {
		params.Set("instType", instrumentType)
	}
	var resp *CopySetting
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getCopySettingsEPL, http.MethodGet, common.EncodeURLValues("copytrading/copy-settings", params), nil, &resp, request.AuthenticatedRequest)
}

// GetMultipleLeverages retrieve leverages that belong to the lead trader and you
func (ok *Okx) GetMultipleLeverages(ctx context.Context, marginMode, uniqueCode, instrumentID string) ([]Leverages, error) {
	if marginMode == "" {
		return nil, margin.ErrInvalidMarginType
	}
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("mgnMode", marginMode)
	params.Set("uniqueCode", uniqueCode)
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var resp []Leverages
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMultipleLeveragesEPL, http.MethodGet, common.EncodeURLValues("copytrading/batch-leverage-info", params), nil, &resp, request.AuthenticatedRequest)
}

// SetMultipleLeverages set Multiple leverages
func (ok *Okx) SetMultipleLeverages(ctx context.Context, arg *SetLeveragesParam) (*SetMultipleLeverageResponse, error) {
	if *arg == (SetLeveragesParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.MarginMode == "" {
		return nil, margin.ErrInvalidMarginType
	}
	if arg.Leverage <= 0 {
		return nil, errInvalidLeverage
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	var resp *SetMultipleLeverageResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setBatchLeverageEPL, http.MethodPost, "copytrading/batch-set-leverage", arg, &resp, request.AuthenticatedRequest)
}

// GetMyLeadTraders retrieve my lead traders
func (ok *Okx) GetMyLeadTraders(ctx context.Context, instrumentType string) ([]CopyTradingLeadTrader, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []CopyTradingLeadTrader
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMyLeadTradersEPL, http.MethodGet, common.EncodeURLValues("copytrading/current-lead-traders", params), nil, &resp, request.AuthenticatedRequest)
}

// GetHistoryLeadTraders retrieve my history lead traders
func (ok *Okx) GetHistoryLeadTraders(ctx context.Context, instrumentType, after, before string, limit int64) ([]CopyTradingLeadTrader, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if after != "" {
		params.Set("after", after)
	}
	if before != "" {
		params.Set("before", before)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CopyTradingLeadTrader
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMyLeadTradersEPL, http.MethodGet, common.EncodeURLValues("copytrading/lead-traders-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetLeadTradersRanks retrieves lead trader ranks
func (ok *Okx) GetLeadTradersRanks(ctx context.Context, req *LeadTraderRanksRequest) ([]LeadTradersRank, error) {
	params := url.Values{}
	if req.InstrumentType != "" {
		params.Set("instType", req.InstrumentType)
	}
	if req.SortType != "" {
		params.Set("sortType", req.SortType)
	}
	if req.HasVacancy {
		params.Set("state", "1")
	}
	if req.MinLeadDays != 0 {
		params.Set("minLeadDays", strconv.FormatUint(req.MinLeadDays, 10))
	}
	if req.MinAssets > 0 {
		params.Set("minAssets", strconv.FormatFloat(req.MinAssets, 'f', -1, 64))
	}
	if req.MaxAssets > 0 {
		params.Set("maxAssets", strconv.FormatFloat(req.MaxAssets, 'f', -1, 64))
	}
	if req.MinAssetsUnderManagement > 0 {
		params.Set("minAum", strconv.FormatFloat(req.MinAssetsUnderManagement, 'f', -1, 64))
	}
	if req.MaxAssetsUnderManagement > 0 {
		params.Set("maxAum", strconv.FormatFloat(req.MaxAssetsUnderManagement, 'f', -1, 64))
	}
	if req.DataVersion != 0 {
		params.Set("dataVer", strconv.FormatUint(req.DataVersion, 10))
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []LeadTradersRank
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderRanksEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-lead-traders", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetWeeklyTraderProfitAndLoss retrieve lead trader weekly pnl. Results are returned in counter chronological order
func (ok *Okx) GetWeeklyTraderProfitAndLoss(ctx context.Context, instrumentType, uniqueCode string) ([]TraderWeeklyProfitAndLoss, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []TraderWeeklyProfitAndLoss
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderWeeklyPNLEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-weekly-pnl", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetDailyLeadTraderPNL retrieve lead trader daily pnl. Results are returned in counter chronological order.
// Last days "1": last 7 days  "2": last 30 days "3": last 90 days  "4": last 365 days
func (ok *Okx) GetDailyLeadTraderPNL(ctx context.Context, instrumentType, uniqueCode, lastDays string) ([]TraderWeeklyProfitAndLoss, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	if lastDays == "" {
		return nil, errLastDaysRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	params.Set("lastDays", lastDays)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []TraderWeeklyProfitAndLoss
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderDailyPNLEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-weekly-pnl", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLeadTraderStats retrieves key data related to lead trader performance
func (ok *Okx) GetLeadTraderStats(ctx context.Context, instrumentType, uniqueCode, lastDays string) ([]LeadTraderStat, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	if lastDays == "" {
		return nil, errLastDaysRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	params.Set("lastDays", lastDays)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []LeadTraderStat
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderStatsEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-stats", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLeadTraderCurrencyPreferences retrieves the most frequently traded crypto of this lead trader. Results are sorted by ratio from large to small
func (ok *Okx) GetLeadTraderCurrencyPreferences(ctx context.Context, instrumentType, uniqueCode, lastDays string) ([]LeadTraderCurrencyPreference, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	if lastDays == "" {
		return nil, errLastDaysRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	params.Set("lastDays", lastDays)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []LeadTraderCurrencyPreference
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderCurrencyPreferencesEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-preference-currency", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLeadTraderCurrentLeadPositions get current leading positions of lead trader
// Instrument type "SPOT" "SWAP"
func (ok *Okx) GetLeadTraderCurrentLeadPositions(ctx context.Context, instrumentType, uniqueCode, afterSubPositionID,
	beforeSubPositionID string, limit int64,
) ([]LeadTraderCurrentLeadPosition, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if afterSubPositionID != "" {
		params.Set("after", afterSubPositionID)
	}
	if beforeSubPositionID != "" {
		params.Set("before", beforeSubPositionID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LeadTraderCurrentLeadPosition
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTraderCurrentLeadPositionsEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-current-subpositions", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLeadTraderLeadPositionHistory retrieve the lead trader completed leading position of the last 3 months. Returns reverse chronological order with subPosId
func (ok *Okx) GetLeadTraderLeadPositionHistory(ctx context.Context, instrumentType, uniqueCode, afterSubPositionID, beforeSubPositionID string, limit int64) ([]LeadPosition, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if afterSubPositionID != "" {
		params.Set("after", afterSubPositionID)
	}
	if beforeSubPositionID != "" {
		params.Set("before", beforeSubPositionID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LeadPosition
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderLeadPositionHistoryEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-subpositions-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// ****************************************** Earn **************************************************

// GetOffers retrieves list of offers for different protocols
func (ok *Okx) GetOffers(ctx context.Context, productID, protocolType string, ccy currency.Code) ([]Offer, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	if protocolType != "" {
		params.Set("protocolType", protocolType)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []Offer
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOfferEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/offers", params), nil, &resp, request.AuthenticatedRequest)
}

// Purchase invest on specific product
func (ok *Okx) Purchase(ctx context.Context, arg *PurchaseRequestParam) (*OrderIDResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.ProductID == "" {
		return nil, fmt.Errorf("%w, missing product id", errMissingRequiredParameter)
	}
	for x := range arg.InvestData {
		if arg.InvestData[x].Currency.IsEmpty() {
			return nil, fmt.Errorf("%w, currency information for investment is required", currency.ErrCurrencyCodeEmpty)
		}
		if arg.InvestData[x].Amount <= 0 {
			return nil, order.ErrAmountBelowMin
		}
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, purchaseEPL, http.MethodPost, "finance/staking-defi/purchase", &arg, &resp, request.AuthenticatedRequest)
}

// Redeem redemption of investment
func (ok *Okx) Redeem(ctx context.Context, arg *RedeemRequestParam) (*OrderIDResponse, error) {
	if *arg == (RedeemRequestParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.OrderID == "" {
		return nil, fmt.Errorf("%w, missing 'orderId'", order.ErrOrderIDNotSet)
	}
	if arg.ProtocolType != "staking" && arg.ProtocolType != "defi" {
		return nil, errInvalidProtocolType
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, redeemEPL, http.MethodPost, "finance/staking-defi/redeem", &arg, &resp, request.AuthenticatedRequest)
}

// CancelPurchaseOrRedemption cancels Purchases or Redemptions
// after cancelling, returning funds will go to the funding account
func (ok *Okx) CancelPurchaseOrRedemption(ctx context.Context, arg *CancelFundingParam) (*OrderIDResponse, error) {
	if *arg == (CancelFundingParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.OrderID == "" {
		return nil, fmt.Errorf("%w, missing 'orderId'", order.ErrOrderIDNotSet)
	}
	if arg.ProtocolType != "staking" && arg.ProtocolType != "defi" {
		return nil, errInvalidProtocolType
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelPurchaseOrRedemptionEPL, http.MethodPost, "finance/staking-defi/cancel", &arg, &resp, request.AuthenticatedRequest)
}

// GetEarnActiveOrders retrieves active orders
func (ok *Okx) GetEarnActiveOrders(ctx context.Context, productID, protocolType, state string, ccy currency.Code) ([]ActiveFundingOrder, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}

	if protocolType != "" {
		// protocol type 'staking' and 'defi' is allowed by default
		if protocolType != "staking" && protocolType != "defi" {
			return nil, errInvalidProtocolType
		}
		params.Set("protocolType", protocolType)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if state != "" {
		params.Set("state", state)
	}
	var resp []ActiveFundingOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEarnActiveOrdersEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/orders-active", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFundingOrderHistory retrieves funding order history
// valid protocol types are 'staking' and 'defi'
func (ok *Okx) GetFundingOrderHistory(ctx context.Context, productID, protocolType string, ccy currency.Code, after, before time.Time, limit int64) ([]ActiveFundingOrder, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	if protocolType != "" {
		params.Set("protocolType", protocolType)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []ActiveFundingOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFundingOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/orders-history", params), nil, &resp, request.AuthenticatedRequest)
}

// **************************************************************** ETH Staking ****************************************************************

// GetProductInfo retrieves ETH staking products
func (ok *Okx) GetProductInfo(ctx context.Context) (*ProductInfo, error) {
	var resp *ProductInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getProductInfoEPL, http.MethodGet,
		"finance/staking-defi/eth/product-info", nil, &resp, request.AuthenticatedRequest)
}

// PurchaseETHStaking staking ETH for BETH
// Only the assets in the funding account can be used
func (ok *Okx) PurchaseETHStaking(ctx context.Context, amount float64) error {
	if amount <= 0 {
		return order.ErrAmountBelowMin
	}
	var resp []string
	return ok.SendHTTPRequest(ctx, exchange.RestSpot, purchaseETHStakingEPL, http.MethodPost, "finance/staking-defi/eth/purchase", map[string]string{"amt": strconv.FormatFloat(amount, 'f', -1, 64)}, &resp, request.AuthenticatedRequest)
}

// RedeemETHStaking only the assets in the funding account can be used. If your BETH is in your trading account, you can make funding transfer first
func (ok *Okx) RedeemETHStaking(ctx context.Context, amount float64) error {
	if amount <= 0 {
		return order.ErrAmountBelowMin
	}
	var resp []string
	return ok.SendHTTPRequest(ctx, exchange.RestSpot, redeemETHStakingEPL, http.MethodPost, "finance/staking-defi/eth/redeem",
		map[string]string{"amt": strconv.FormatFloat(amount, 'f', -1, 64)}, &resp, request.AuthenticatedRequest)
}

// GetBETHAssetsBalance balance is a snapshot summarized all BETH assets in trading and funding accounts. Also, the snapshot updates hourly
func (ok *Okx) GetBETHAssetsBalance(ctx context.Context) (*BETHAssetsBalance, error) {
	var resp *BETHAssetsBalance
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBETHBalanceEPL, http.MethodGet,
		"finance/staking-defi/eth/balance", nil, &resp, request.AuthenticatedRequest)
}

// GetPurchaseAndRedeemHistory retrieves purchase and redeem history
// kind possible values are 'purchase' and 'redeem'
// Status 'pending' 'success' 'failed'
func (ok *Okx) GetPurchaseAndRedeemHistory(ctx context.Context, kind, status string, after, before time.Time, limit int64) ([]PurchaseRedeemHistory, error) {
	if kind == "" {
		return nil, fmt.Errorf("%w, possible values are 'purchase' and 'redeem'", errLendingTermIsRequired)
	}
	params := url.Values{}
	params.Set("type", kind)
	if status != "" {
		params.Set("status", status)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PurchaseRedeemHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPurchaseRedeemHistoryEPL, http.MethodGet,
		common.EncodeURLValues("finance/staking-defi/eth/purchase-redeem-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAPYHistory retrieves Annual percentage yield(APY) history
func (ok *Okx) GetAPYHistory(ctx context.Context, days int64) ([]APYItem, error) {
	if days == 0 || days > 365 {
		return nil, errors.New("field days is required; possible values from 1 to 365")
	}
	var resp []APYItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAPYHistoryEPL, http.MethodGet, fmt.Sprintf("finance/staking-defi/eth/apy-history?days=%d", days), nil, &resp, request.UnauthenticatedRequest)
}

// GetTickers retrieves the latest price snapshots best bid/ ask price, and trading volume in the last 24 hours
func (ok *Okx) GetTickers(ctx context.Context, instType, uly, instFamily string) ([]TickerResponse, error) {
	if instType == "" {
		return nil, errInvalidInstrumentType
	}
	params := url.Values{}
	params.Set("instType", instType)
	if instFamily != "" {
		params.Set("instFamily", instFamily)
	}
	if uly != "" {
		params.Set("uly", uly)
	}
	var response []TickerResponse
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTickersEPL, http.MethodGet, common.EncodeURLValues("market/tickers", params), nil, &response, request.UnauthenticatedRequest)
}

// GetTicker retrieves the latest price snapshot, best bid/ask price, and trading volume in the last 24 hours
func (ok *Okx) GetTicker(ctx context.Context, instrumentID string) (*TickerResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *TickerResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTickerEPL, http.MethodGet, common.EncodeURLValues("market/ticker", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPremiumHistory returns premium data in the past 6 months
func (ok *Okx) GetPremiumHistory(ctx context.Context, instrumentID string, after, before time.Time, limit int64) ([]PremiumInfo, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if !after.IsZero() && !before.IsZero() {
		err := common.StartEndTimeCheck(after, before)
		if err != nil {
			return nil, err
		}
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PremiumInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPremiumHistoryEPL, http.MethodGet, common.EncodeURLValues("public/premium-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetIndexTickers Retrieves index tickers
func (ok *Okx) GetIndexTickers(ctx context.Context, quoteCurrency currency.Code, instID string) ([]IndexTicker, error) {
	if instID == "" && quoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, QuoteCurrency or InstrumentID is required", errEitherInstIDOrCcyIsRequired)
	}
	params := url.Values{}
	if !quoteCurrency.IsEmpty() {
		params.Set("quoteCcy", quoteCurrency.String())
	}
	if instID != "" {
		params.Set("instId", instID)
	}
	var resp []IndexTicker
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getIndexTickersEPL, http.MethodGet, common.EncodeURLValues("market/index-tickers", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetInstrumentTypeFromAssetItem returns a string representation of asset.Item; which is an equivalent term for InstrumentType in Okx exchange
func GetInstrumentTypeFromAssetItem(a asset.Item) string {
	switch a {
	case asset.PerpetualSwap:
		return instTypeSwap
	case asset.Options:
		return instTypeOption
	default:
		return strings.ToUpper(a.String())
	}
}

// GetUnderlying returns the instrument ID for the corresponding asset pairs and asset type( Instrument Type )
func (ok *Okx) GetUnderlying(pair currency.Pair, a asset.Item) (string, error) {
	if !pair.IsPopulated() {
		return "", currency.ErrCurrencyPairsEmpty
	}
	format, err := ok.GetPairFormat(a, false)
	if err != nil {
		return "", err
	}
	return pair.Base.String() + format.Delimiter + pair.Quote.String(), nil
}

// GetPairFromInstrumentID returns a currency pair give an instrument ID and asset Item, which represents the instrument type
func (ok *Okx) GetPairFromInstrumentID(instrumentID string) (currency.Pair, error) {
	codes := strings.Split(instrumentID, currency.DashDelimiter)
	if len(codes) >= 2 {
		instrumentID = codes[0] + currency.DashDelimiter + strings.Join(codes[1:], currency.DashDelimiter)
	}
	return currency.NewPairFromString(instrumentID)
}

// GetOrderBookDepth returns the recent order asks and bids before specified timestamp
func (ok *Okx) GetOrderBookDepth(ctx context.Context, instrumentID string, depth int64) (*OrderBookResponseDetail, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if depth > 0 {
		params.Set("sz", strconv.FormatInt(depth, 10))
	}
	var resp *OrderBookResponseDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOrderBookEPL, http.MethodGet, common.EncodeURLValues("market/books", params), nil, &resp, request.UnauthenticatedRequest)
}

// IntervalFromString returns a kline.Interval instance from string
func IntervalFromString(interval kline.Interval, appendUTC bool) string {
	switch interval {
	case kline.OneMin:
		return "1m"
	case kline.ThreeMin:
		return "3m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1H"
	case kline.TwoHour:
		return "2H"
	case kline.FourHour:
		return "4H"
	}

	duration := ""
	switch interval {
	case kline.SixHour: // NOTE: Cases here and below can either be local Hong Kong time or UTC time.
		duration = "6H"
	case kline.TwelveHour:
		duration = "12H"
	case kline.OneDay:
		duration = "1D"
	case kline.TwoDay:
		duration = "2D"
	case kline.ThreeDay:
		duration = "3D"
	case kline.FiveDay:
		duration = "5D"
	case kline.OneWeek:
		duration = "1W"
	case kline.OneMonth:
		duration = "1M"
	case kline.ThreeMonth:
		duration = "3M"
	case kline.SixMonth:
		duration = "6M"
	case kline.OneYear:
		duration = "1Y"
	default:
		return duration
	}
	if appendUTC {
		duration += "utc"
	}
	return duration
}

// GetCandlesticks retrieve the candlestick charts. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar
func (ok *Okx) GetCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, before, after time.Time, limit int64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, "market/candles", getCandlesticksEPL)
}

// GetCandlesticksHistory retrieve history candlestick charts from recent years
func (ok *Okx) GetCandlesticksHistory(ctx context.Context, instrumentID string, interval kline.Interval, before, after time.Time, limit int64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, "market/history-candles", getCandlestickHistoryEPL)
}

// GetIndexCandlesticks retrieve the candlestick charts of the index. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar.
// the response is a list of Candlestick data
func (ok *Okx) GetIndexCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, before, after time.Time, limit int64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, "market/index-candles", getIndexCandlesticksEPL)
}

// GetMarkPriceCandlesticks retrieve the candlestick charts of mark price. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar
func (ok *Okx) GetMarkPriceCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, before, after time.Time, limit int64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, "market/mark-price-candles", getCandlestickHistoryEPL)
}

// GetHistoricIndexCandlesticksHistory retrieve the candlestick charts of the index from recent years
func (ok *Okx) GetHistoricIndexCandlesticksHistory(ctx context.Context, instrumentID string, after, before time.Time, bar kline.Interval, limit int64) ([]CandlestickHistoryItem, error) {
	return ok.getHistoricCandlesticks(ctx, instrumentID, "market/history-index-candles", after, before, bar, limit, getIndexCandlesticksHistoryEPL)
}

// GetMarkPriceCandlestickHistory retrieve the candlestick charts of the mark price from recent years
func (ok *Okx) GetMarkPriceCandlestickHistory(ctx context.Context, instrumentID string, after, before time.Time, bar kline.Interval, limit int64) ([]CandlestickHistoryItem, error) {
	return ok.getHistoricCandlesticks(ctx, instrumentID, "market/history-mark-price-candles", after, before, bar, limit, getMarkPriceCandlesticksHistoryEPL)
}

func (ok *Okx) getHistoricCandlesticks(ctx context.Context, instrumentID, path string, after, before time.Time, bar kline.Interval, limit int64, epl request.EndpointLimit) ([]CandlestickHistoryItem, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	barString := IntervalFromString(bar, false)
	if barString != "" {
		params.Set("bar", barString)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CandlestickHistoryItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, common.EncodeURLValues(path, params), nil, &resp, request.UnauthenticatedRequest)
}

// GetEconomicCalendarData retrieves the macro-economic calendar data within 3 months. Historical data from 3 months ago is only available to users with trading fee tier VIP1 and above
func (ok *Okx) GetEconomicCalendarData(ctx context.Context, region, importance string, before, after time.Time, limit int64) ([]EconomicCalendar, error) {
	params := url.Values{}
	if region != "" {
		params.Set("region", region)
	}
	if importance != "" {
		params.Set("importance", importance)
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []EconomicCalendar
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEconomicCalendarEPL, http.MethodGet, common.EncodeURLValues("public/economic-calendar", params), nil, &resp, request.AuthenticatedRequest)
}

// GetCandlestickData handles fetching the data for both the default GetCandlesticks, GetCandlesticksHistory, and GetIndexCandlesticks() methods
func (ok *Okx) GetCandlestickData(ctx context.Context, instrumentID string, interval kline.Interval, before, after time.Time, limit int64, route string, rateLimit request.EndpointLimit) ([]CandleStick, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("limit", strconv.FormatInt(limit, 10))
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	bar := IntervalFromString(interval, true)
	if bar != "" {
		params.Set("bar", bar)
	}
	var resp []CandleStick
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rateLimit, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, request.UnauthenticatedRequest)
}

// GetTrades retrieve the recent transactions of an instrument
func (ok *Okx) GetTrades(ctx context.Context, instrumentID string, limit int64) ([]TradeResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []TradeResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTradesRequestEPL, http.MethodGet, common.EncodeURLValues("market/trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetTradesHistory retrieves the recent transactions of an instrument from the last 3 months with pagination
func (ok *Okx) GetTradesHistory(ctx context.Context, instrumentID, before, after string, limit int64) ([]TradeResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if before != "" {
		params.Set("before", before)
	}
	if after != "" {
		params.Set("after", after)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []TradeResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTradesHistoryEPL, http.MethodGet, common.EncodeURLValues("market/history-trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOptionTradesByInstrumentFamily retrieve the recent transactions of an instrument under same instFamily. The maximum is 100
func (ok *Okx) GetOptionTradesByInstrumentFamily(ctx context.Context, instrumentFamily string) ([]InstrumentFamilyTrade, error) {
	if instrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	var resp []InstrumentFamilyTrade
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, optionInstrumentTradeFamilyEPL, http.MethodGet, "market/option/instrument-family-trades?instFamily="+instrumentFamily, nil, &resp, request.UnauthenticatedRequest)
}

// GetOptionTrades retrieves option trades
// Option type, 'C': Call 'P': put
func (ok *Okx) GetOptionTrades(ctx context.Context, instrumentID, instrumentFamily, optionType string) ([]OptionTrade, error) {
	if instrumentID == "" && instrumentFamily == "" {
		return nil, errInstrumentIDorFamilyRequired
	}
	params := url.Values{}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	if optionType != "" {
		params.Set("optType", optionType)
	}
	var resp []OptionTrade
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, optionTradesEPL, http.MethodGet, common.EncodeURLValues("public/option-trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// Get24HTotalVolume The 24-hour trading volume is calculated on a rolling basis, using USD as the pricing unit
func (ok *Okx) Get24HTotalVolume(ctx context.Context) (*TradingVolumeIn24HR, error) {
	var resp *TradingVolumeIn24HR
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, get24HTotalVolumeEPL, http.MethodGet, "market/platform-24-volume", nil, &resp, request.UnauthenticatedRequest)
}

// GetOracle Get the crypto price of signing using Open Oracle smart contract
func (ok *Okx) GetOracle(ctx context.Context) (*OracleSmartContractResponse, error) {
	var resp *OracleSmartContractResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOracleEPL, http.MethodGet, "market/open-oracle", nil, &resp, request.UnauthenticatedRequest)
}

// GetExchangeRate this interface provides the average exchange rate data for 2 weeks
// from USD to CNY
func (ok *Okx) GetExchangeRate(ctx context.Context) (*UsdCnyExchangeRate, error) {
	var resp *UsdCnyExchangeRate
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getExchangeRateRequestEPL, http.MethodGet, "market/exchange-rate", nil, &resp, request.UnauthenticatedRequest)
}

// GetIndexComponents returns the index component information data on the market
func (ok *Okx) GetIndexComponents(ctx context.Context, index string) (*IndexComponent, error) {
	if index == "" {
		return nil, errIndexComponentNotFound
	}
	params := url.Values{}
	params.Set("index", index)
	var resp *IndexComponent
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getIndexComponentsEPL, http.MethodGet, common.EncodeURLValues("market/index-components", params), nil, &resp, request.UnauthenticatedRequest)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errIndexComponentNotFound
	}
	return resp, nil
}

// GetBlockTickers retrieves the latest block trading volume in the last 24 hours.
// Instrument Type Is Mandatory, and Underlying is Optional
func (ok *Okx) GetBlockTickers(ctx context.Context, instrumentType, underlying string) ([]BlockTicker, error) {
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	if underlying != "" {
		params.Set("uly", underlying)
	}
	var resp []BlockTicker
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBlockTickersEPL, http.MethodGet, common.EncodeURLValues("market/block-tickers", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetBlockTicker retrieves the latest block trading volume in the last 24 hours
func (ok *Okx) GetBlockTicker(ctx context.Context, instrumentID string) (*BlockTicker, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *BlockTicker
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBlockTickersEPL, http.MethodGet, common.EncodeURLValues("market/block-ticker", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPublicBlockTrades retrieves the recent block trading transactions of an instrument. Descending order by tradeId
func (ok *Okx) GetPublicBlockTrades(ctx context.Context, instrumentID string) ([]BlockTrade, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp []BlockTrade
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBlockTradesEPL, http.MethodGet, common.EncodeURLValues("public/block-trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// ********************************************* Spread Trading ***********************************************

// Spread Trading: As Introduced in the Okx exchange. URL: https://www.okx.com/docs-v5/en/#spread-trading-introduction

// PlaceSpreadOrder places new spread order
func (ok *Okx) PlaceSpreadOrder(ctx context.Context, arg *SpreadOrderParam) (*SpreadOrderResponse, error) {
	err := ok.validatePlaceSpreadOrderParam(arg)
	if err != nil {
		return nil, err
	}
	var resp *SpreadOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeSpreadOrderEPL, http.MethodPost, "sprd/order", arg, &resp, request.AuthenticatedRequest)
}

func (ok *Okx) validatePlaceSpreadOrderParam(arg *SpreadOrderParam) error {
	if *arg == (SpreadOrderParam{}) {
		return common.ErrEmptyParams
	}
	if arg.SpreadID == "" {
		return fmt.Errorf("%w, spread ID missing", errMissingInstrumentID)
	}
	if arg.OrderType == "" {
		return fmt.Errorf("%w spread order type is required", order.ErrTypeIsInvalid)
	}
	if arg.Size <= 0 {
		return order.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return order.ErrPriceBelowMin
	}
	arg.Side = strings.ToLower(arg.Side)
	switch arg.Side {
	case order.Buy.Lower(), order.Sell.Lower():
	default:
		return fmt.Errorf("%w %s", order.ErrSideIsInvalid, arg.Side)
	}
	return nil
}

// CancelSpreadOrder cancels an incomplete spread order
func (ok *Okx) CancelSpreadOrder(ctx context.Context, orderID, clientOrderID string) (*SpreadOrderResponse, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	arg := make(map[string]string)
	if orderID != "" {
		arg["ordId"] = orderID
	}
	if clientOrderID != "" {
		arg["clOrdId"] = clientOrderID
	}
	var resp *SpreadOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelSpreadOrderEPL, http.MethodPost, "sprd/cancel-order", arg, &resp, request.AuthenticatedRequest)
}

// CancelAllSpreadOrders cancels all spread orders and return success message
// spreadID is optional
// the function returns success status and error message
func (ok *Okx) CancelAllSpreadOrders(ctx context.Context, spreadID string) (bool, error) {
	arg := make(map[string]string, 1)
	if spreadID != "" {
		arg["sprdId"] = spreadID
	}
	resp := &struct {
		Result bool `json:"result"`
	}{}
	return resp.Result, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllSpreadOrderEPL, http.MethodPost, "sprd/mass-cancel", arg, resp, request.AuthenticatedRequest)
}

// AmendSpreadOrder amends incomplete spread order
func (ok *Okx) AmendSpreadOrder(ctx context.Context, arg *AmendSpreadOrderParam) (*SpreadOrderResponse, error) {
	if *arg == (AmendSpreadOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.NewPrice == 0 && arg.NewSize == 0 {
		return nil, errSizeOrPriceIsRequired
	}
	var resp *SpreadOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendSpreadOrderEPL, http.MethodPost, "sprd/amend-order", arg, &resp, request.AuthenticatedRequest)
}

// GetSpreadOrderDetails retrieves spread order details
func (ok *Okx) GetSpreadOrderDetails(ctx context.Context, orderID, clientOrderID string) (*SpreadOrder, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	var resp *SpreadOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrderDetailsEPL, http.MethodGet, common.EncodeURLValues("sprd/order", params), nil, &resp, request.AuthenticatedRequest)
}

// GetActiveSpreadOrders retrieves list of incomplete spread orders
func (ok *Okx) GetActiveSpreadOrders(ctx context.Context, spreadID, orderType, state, beginID, endID string, limit int64) ([]SpreadOrder, error) {
	params := url.Values{}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	if orderType != "" {
		params.Set("ordType", orderType)
	}
	if state != "" {
		params.Set("state", state)
	}
	if beginID != "" {
		params.Set("beginId", beginID)
	}
	if endID != "" {
		params.Set("endId", endID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SpreadOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getActiveSpreadOrdersEPL, http.MethodGet, common.EncodeURLValues("sprd/orders-pending", params), nil, &resp, request.AuthenticatedRequest)
}

// GetCompletedSpreadOrdersLast7Days retrieve the completed order data for the last 7 days, and the incomplete orders (filledSz =0 & state = canceled) that have been canceled are only reserved for 2 hours. Results are returned in counter chronological order
func (ok *Okx) GetCompletedSpreadOrdersLast7Days(ctx context.Context, spreadID, orderType, state, beginID, endID string, begin, end time.Time, limit int64) ([]SpreadOrder, error) {
	params := url.Values{}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	if orderType != "" {
		params.Set("ordType", orderType)
	}
	if state != "" {
		params.Set("state", state)
	}
	if beginID != "" {
		params.Set("beginId", beginID)
	}
	if endID != "" {
		params.Set("endId", endID)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SpreadOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrders7DaysEPL, http.MethodGet, common.EncodeURLValues("sprd/orders-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetSpreadTradesOfLast7Days retrieve historical transaction details for the last 7 days. Results are returned in counter chronological order
func (ok *Okx) GetSpreadTradesOfLast7Days(ctx context.Context, spreadID, tradeID, orderID, beginID, endID string, begin, end time.Time, limit int64) ([]SpreadTrade, error) {
	params := url.Values{}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	if tradeID != "" {
		params.Set("tradeId", tradeID)
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if beginID != "" {
		params.Set("beginId", beginID)
	}
	if endID != "" {
		params.Set("endId", endID)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SpreadTrade
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrderTradesEPL, http.MethodGet, common.EncodeURLValues("sprd/trades", params), nil, &resp, request.AuthenticatedRequest)
}

// GetPublicSpreads retrieve all available spreads based on the request parameters
func (ok *Okx) GetPublicSpreads(ctx context.Context, baseCurrency, instrumentID, spreadID, state string) ([]SpreadInstrument, error) {
	params := url.Values{}
	if baseCurrency != "" {
		params.Set("baseCcy", baseCurrency)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	if state != "" {
		params.Set("state", state)
	}
	var resp []SpreadInstrument
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadsEPL, http.MethodGet, common.EncodeURLValues("sprd/spreads", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPublicSpreadOrderBooks retrieve the order book of the spread
func (ok *Okx) GetPublicSpreadOrderBooks(ctx context.Context, spreadID string, orderbookSize int64) ([]SpreadOrderbook, error) {
	if spreadID == "" {
		return nil, fmt.Errorf("%w, spread ID missing", errMissingInstrumentID)
	}
	params := url.Values{}
	params.Set("sprdId", spreadID)
	if orderbookSize != 0 {
		params.Set("size", strconv.FormatInt(orderbookSize, 10))
	}
	var resp []SpreadOrderbook
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrderbookEPL, http.MethodGet, common.EncodeURLValues("sprd/books", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPublicSpreadTickers retrieve the latest price snapshot, best bid/ask price, and trading volume in the last 24 hours
func (ok *Okx) GetPublicSpreadTickers(ctx context.Context, spreadID string) ([]SpreadTicker, error) {
	if spreadID == "" {
		return nil, fmt.Errorf("%w, spread ID is required", errMissingInstrumentID)
	}
	params := url.Values{}
	params.Set("sprdId", spreadID)
	var resp []SpreadTicker
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadTickerEPL, http.MethodGet, common.EncodeURLValues("sprd/ticker", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetPublicSpreadTrades retrieve the recent transactions of an instrument (at most 500 records per request). Results are returned in counter chronological order
func (ok *Okx) GetPublicSpreadTrades(ctx context.Context, spreadID string) ([]SpreadPublicTradeItem, error) {
	params := url.Values{}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	var resp []SpreadPublicTradeItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadPublicTradesEPL, http.MethodGet, common.EncodeURLValues("sprd/public-trades", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetSpreadCandlesticks retrieves candlestick charts for a given spread instrument
func (ok *Okx) GetSpreadCandlesticks(ctx context.Context, spreadID string, interval kline.Interval, before, after time.Time, limit uint64) ([]SpreadCandlestick, error) {
	if spreadID == "" {
		return nil, fmt.Errorf("%w, spread ID is required", errMissingInstrumentID)
	}
	params := url.Values{}
	params.Set("sprdId", spreadID)
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if bar := IntervalFromString(interval, true); bar != "" {
		params.Set("bar", bar)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []SpreadCandlestick
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadCandlesticksEPL, http.MethodGet, common.EncodeURLValues("market/sprd-candles", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetSpreadCandlesticksHistory retrieves candlestick chart history for a given spread instrument for a period of up to 3 months
func (ok *Okx) GetSpreadCandlesticksHistory(ctx context.Context, spreadID string, interval kline.Interval, before, after time.Time, limit uint64) ([]SpreadCandlestick, error) {
	if spreadID == "" {
		return nil, fmt.Errorf("%w, spread ID is required", errMissingInstrumentID)
	}
	params := url.Values{}
	params.Set("sprdId", spreadID)
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if bar := IntervalFromString(interval, true); bar != "" {
		params.Set("bar", bar)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []SpreadCandlestick
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadCandlesticksHistoryEPL, http.MethodGet, common.EncodeURLValues("market/sprd-history-candles", params), nil, &resp, request.UnauthenticatedRequest)
}

// CancelAllSpreadOrdersAfterCountdown cancel all pending orders after the countdown timeout. Only applicable to spread trading
func (ok *Okx) CancelAllSpreadOrdersAfterCountdown(ctx context.Context, timeoutDuration int64) (*SpreadOrderCancellationResponse, error) {
	if (timeoutDuration != 0) && (timeoutDuration < 10 || timeoutDuration > 120) {
		return nil, fmt.Errorf("%w, range of value can be 0, [10, 120]", errCountdownTimeoutRequired)
	}
	arg := &struct {
		TimeOut int64 `json:"timeOut,string"`
	}{
		TimeOut: timeoutDuration,
	}
	var resp *SpreadOrderCancellationResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllSpreadOrdersAfterEPL, http.MethodPost, "sprd/cancel-all-after", arg, &resp, request.AuthenticatedRequest)
}

/************************************ Public Data Endpoints *************************************************/

// GetInstruments retrieve a list of instruments with open contracts
func (ok *Okx) GetInstruments(ctx context.Context, arg *InstrumentsFetchParams) ([]Instrument, error) {
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	if arg.InstrumentType == instTypeOption &&
		arg.InstrumentFamily == "" && arg.Underlying == "" {
		return nil, errInstrumentFamilyOrUnderlyingRequired
	}
	params := url.Values{}
	arg.InstrumentType = strings.ToUpper(arg.InstrumentType)
	params.Set("instType", arg.InstrumentType)
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	var resp []Instrument
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getInstrumentsEPL, http.MethodGet,
		common.EncodeURLValues("public/instruments", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetDeliveryHistory retrieves the estimated delivery price of the last 3 months, which will only have a return value one hour before the delivery/exercise
func (ok *Okx) GetDeliveryHistory(ctx context.Context, instrumentType, underlying, instrumentFamily string, after, before time.Time, limit int64) ([]DeliveryHistory, error) {
	if instrumentType == "" {
		return nil, errInvalidInstrumentType
	}
	switch instrumentType {
	case instTypeFutures, instTypeOption:
		if underlying == "" && instrumentFamily == "" {
			return nil, errInstrumentFamilyOrUnderlyingRequired
		}
	}
	if limit > 100 {
		return nil, errLimitValueExceedsMaxOf100
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	if underlying != "" {
		params.Set("uly", underlying)
	}
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []DeliveryHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDeliveryExerciseHistoryEPL, http.MethodGet,
		common.EncodeURLValues("public/delivery-exercise-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOpenInterestData retrieves the total open interest for contracts on OKX
func (ok *Okx) GetOpenInterestData(ctx context.Context, instType, uly, instrumentFamily, instID string) ([]OpenInterest, error) {
	if instType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	if instType == instTypeOption && uly == "" && instrumentFamily == "" {
		return nil, errInstrumentFamilyOrUnderlyingRequired
	}
	params := url.Values{}
	instType = strings.ToUpper(instType)
	params.Set("instType", instType)
	if uly != "" {
		params.Set("uly", uly)
	}
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	if instID != "" {
		params.Set("instId", instID)
	}
	var resp []OpenInterest
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOpenInterestEPL, http.MethodGet, common.EncodeURLValues("public/open-interest", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetSingleFundingRate returns the latest funding rate
func (ok *Okx) GetSingleFundingRate(ctx context.Context, instrumentID string) (*FundingRateResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *FundingRateResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFundingEPL, http.MethodGet, common.EncodeURLValues("public/funding-rate", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetFundingRateHistory retrieves funding rate history. This endpoint can retrieve data from the last 3 months
func (ok *Okx) GetFundingRateHistory(ctx context.Context, instrumentID string, before, after time.Time, limit int64) ([]FundingRateResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FundingRateResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFundingRateHistoryEPL, http.MethodGet, common.EncodeURLValues("public/funding-rate-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetLimitPrice retrieves the highest buy limit and lowest sell limit of the instrument
func (ok *Okx) GetLimitPrice(ctx context.Context, instrumentID string) (*LimitPriceResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *LimitPriceResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLimitPriceEPL, http.MethodGet, common.EncodeURLValues("public/price-limit", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOptionMarketData retrieves option market data
func (ok *Okx) GetOptionMarketData(ctx context.Context, underlying, instrumentFamily string, expTime time.Time) ([]OptionMarketDataResponse, error) {
	if underlying == "" && instrumentFamily == "" {
		return nil, errInstrumentFamilyOrUnderlyingRequired
	}
	params := url.Values{}
	if underlying != "" {
		params.Set("uly", underlying)
	}
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	if !expTime.IsZero() {
		params.Set("expTime", fmt.Sprintf("%d%d%d", expTime.Year(), expTime.Month(), expTime.Day()))
	}
	var resp []OptionMarketDataResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOptionMarketDateEPL, http.MethodGet, common.EncodeURLValues("public/opt-summary", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetEstimatedDeliveryPrice retrieves the estimated delivery price which will only have a return value one hour before the delivery/exercise
func (ok *Okx) GetEstimatedDeliveryPrice(ctx context.Context, instrumentID string) ([]DeliveryEstimatedPrice, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp []DeliveryEstimatedPrice
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEstimatedDeliveryPriceEPL, http.MethodGet, common.EncodeURLValues("public/estimated-price", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetDiscountRateAndInterestFreeQuota retrieves discount rate level and interest-free quota
func (ok *Okx) GetDiscountRateAndInterestFreeQuota(ctx context.Context, ccy currency.Code, discountLevel int8) ([]DiscountRate, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if discountLevel > 0 {
		params.Set("discountLv", strconv.FormatInt(int64(discountLevel), 10))
	}
	var response []DiscountRate
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDiscountRateAndInterestFreeQuotaEPL, http.MethodGet, common.EncodeURLValues("public/discount-rate-interest-free-quota", params), nil, &response, request.UnauthenticatedRequest)
}

// GetSystemTime retrieve API server time
func (ok *Okx) GetSystemTime(ctx context.Context) (types.Time, error) {
	resp := &tsResp{}
	return resp.Timestamp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSystemTimeEPL, http.MethodGet, "public/time", nil, resp, request.UnauthenticatedRequest)
}

// GetLiquidationOrders retrieves information on liquidation orders in the last day
func (ok *Okx) GetLiquidationOrders(ctx context.Context, arg *LiquidationOrderRequestParams) (*LiquidationOrder, error) {
	arg.InstrumentType = strings.ToUpper(arg.InstrumentType)
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	arg.MarginMode = strings.ToLower(arg.MarginMode)
	if arg.MarginMode != "" {
		params.Set("mgnMode", arg.MarginMode)
	}
	switch {
	case arg.InstrumentType == instTypeMargin && arg.InstrumentID != "":
		params.Set("instId", arg.InstrumentID)
	case arg.InstrumentType == instTypeMargin && arg.Currency.String() != "":
		params.Set("ccy", arg.Currency.String())
	default:
		return nil, errEitherInstIDOrCcyIsRequired
	}
	if arg.InstrumentType != instTypeMargin && arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.InstrumentType == instTypeFutures && arg.Alias != "" {
		params.Set("alias", arg.Alias)
	}
	if !arg.Before.IsZero() {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if !arg.After.IsZero() {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if arg.Limit > 0 && arg.Limit < 100 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp *LiquidationOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLiquidationOrdersEPL, http.MethodGet, common.EncodeURLValues("public/liquidation-orders", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetMarkPrice  retrieve mark price
func (ok *Okx) GetMarkPrice(ctx context.Context, instrumentType, underlying, instrumentFamily, instrumentID string) ([]MarkPrice, error) {
	if instrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(instrumentType))
	if underlying != "" {
		params.Set("uly", underlying)
	}
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var response []MarkPrice
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMarkPriceEPL, http.MethodGet, common.EncodeURLValues("public/mark-price", params), nil, &response, request.UnauthenticatedRequest)
}

// GetPositionTiers retrieves position tiers information，maximum leverage depends on your borrowings and margin ratio
func (ok *Okx) GetPositionTiers(ctx context.Context, instrumentType, tradeMode, underlying, instrumentFamily, instrumentID, tiers string, ccy currency.Code) ([]PositionTiers, error) {
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	tradeMode = strings.ToLower(tradeMode)
	switch tradeMode {
	case TradeModeCross, TradeModeIsolated:
	default:
		return nil, errInvalidTradeMode
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(instrumentType))
	params.Set("tdMode", tradeMode)
	if underlying != "" {
		params.Set("uly", underlying)
	}

	switch instrumentType {
	case instTypeSwap, instTypeFutures, instTypeOption:
		if instrumentFamily == "" && underlying == "" {
			return nil, errInstrumentFamilyOrUnderlyingRequired
		}
		if ccy.IsEmpty() && instrumentID == "" {
			return nil, errEitherInstIDOrCcyIsRequired
		}
	}
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	if underlying != "" {
		params.Set("uly", underlying)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if tiers != "" {
		params.Set("tiers", tiers)
	}
	var response []PositionTiers
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPositionTiersEPL, http.MethodGet, common.EncodeURLValues("public/position-tiers", params), nil, &response, request.UnauthenticatedRequest)
}

// GetInterestRateAndLoanQuota retrieves an interest rate and loan quota information for various currencies
func (ok *Okx) GetInterestRateAndLoanQuota(ctx context.Context) ([]InterestRateLoanQuotaItem, error) {
	var resp []InterestRateLoanQuotaItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getInterestRateAndLoanQuotaEPL, http.MethodGet, "public/interest-rate-loan-quota", nil, &resp, request.UnauthenticatedRequest)
}

// GetInterestRateAndLoanQuotaForVIPLoans retrieves an interest rate and loan quota information for VIP users of various currencies
func (ok *Okx) GetInterestRateAndLoanQuotaForVIPLoans(ctx context.Context) ([]VIPInterestRateAndLoanQuotaInformation, error) {
	var response []VIPInterestRateAndLoanQuotaInformation
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getInterestRateAndLoanQuoteForVIPLoansEPL, http.MethodGet, "public/vip-interest-rate-loan-quota", nil, &response, request.UnauthenticatedRequest)
}

// GetPublicUnderlyings returns list of underlyings for various instrument types
func (ok *Okx) GetPublicUnderlyings(ctx context.Context, instrumentType string) ([]string, error) {
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(instrumentType))
	var resp []string
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getUnderlyingEPL, http.MethodGet, common.EncodeURLValues("public/underlying", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetInsuranceFundInformation returns insurance fund balance information
func (ok *Okx) GetInsuranceFundInformation(ctx context.Context, arg *InsuranceFundInformationRequestParams) (*InsuranceFundInformation, error) {
	if *arg == (InsuranceFundInformationRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.InstrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(arg.InstrumentType))
	arg.InsuranceType = strings.ToLower(arg.InsuranceType)
	if arg.InsuranceType != "" {
		params.Set("type", arg.InsuranceType)
	}
	switch arg.InstrumentType {
	case instTypeFutures, instTypeSwap, instTypeOption:
		if arg.Underlying == "" && arg.InstrumentFamily == "" {
			return nil, errInstrumentFamilyOrUnderlyingRequired
		}
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.InstrumentFamily != "" {
		params.Set("instFamily", arg.InstrumentFamily)
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	if !arg.Before.IsZero() {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if !arg.After.IsZero() {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp *InsuranceFundInformation
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getInsuranceFundEPL, http.MethodGet, common.EncodeURLValues("public/insurance-fund", params), nil, &resp, request.UnauthenticatedRequest)
}

// CurrencyUnitConvert convert currency to contract, or contract to currency
func (ok *Okx) CurrencyUnitConvert(ctx context.Context, instrumentID string, quantity, orderPrice float64, convertType uint64, unitOfCcy currency.Code, operationTypeOpen bool) (*UnitConvertResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if quantity <= 0 {
		return nil, errMissingQuantity
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("sz", strconv.FormatFloat(quantity, 'f', 0, 64))
	if orderPrice > 0 {
		params.Set("px", strconv.FormatFloat(orderPrice, 'f', 0, 64))
	}
	if convertType > 0 {
		params.Set("type", strconv.FormatUint(convertType, 10))
	}
	switch unitOfCcy {
	case currency.USDC, currency.USDT:
		params.Set("unit", "usds")
	default:
		params.Set("unit", "coin")
	}
	// Applicable to FUTURES and SWAP orders
	if operationTypeOpen {
		params.Set("opType", "open")
	} else {
		params.Set("opType", "close")
	}
	var resp *UnitConvertResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, unitConvertEPL, http.MethodGet,
		common.EncodeURLValues("public/convert-contract-coin", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOptionsTickBands retrieves option tick bands information.
// Instrument type OPTION
func (ok *Okx) GetOptionsTickBands(ctx context.Context, instrumentType, instrumentFamily string) ([]OptionTickBand, error) {
	if instrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	var resp []OptionTickBand
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, optionTickBandsEPL, http.MethodGet,
		common.EncodeURLValues("public/instrument-tick-bands", params), nil, &resp, request.UnauthenticatedRequest)
}

// Trading Data Endpoints

// GetSupportCoins retrieves the currencies supported by the trading data endpoints
func (ok *Okx) GetSupportCoins(ctx context.Context) (*SupportedCoinsData, error) {
	var resp *SupportedCoinsData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSupportCoinEPL, http.MethodGet, "rubik/stat/trading-data/support-coin", nil, &resp, request.UnauthenticatedRequest)
}

// GetTakerVolume retrieves the taker volume for both buyers and sellers
func (ok *Okx) GetTakerVolume(ctx context.Context, ccy currency.Code, instrumentType, instrumentFamily string, begin, end time.Time, period kline.Interval) ([]TakerVolume, error) {
	if instrumentType == "" {
		return nil, fmt.Errorf("%w, empty instrument type", errInvalidInstrumentType)
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(instrumentType))
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	interval := IntervalFromString(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	var response []TakerVolume
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTakerVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/taker-volume", params), nil, &response, request.UnauthenticatedRequest)
}

// GetMarginLendingRatio retrieves the ratio of cumulative amount between currency margin quote currency and base currency
func (ok *Okx) GetMarginLendingRatio(ctx context.Context, ccy currency.Code, begin, end time.Time, period kline.Interval) ([]MarginLendRatioItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	interval := IntervalFromString(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var response []MarginLendRatioItem
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMarginLendingRatioEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/margin/loan-ratio", params), nil, &response, request.UnauthenticatedRequest)
}

// GetLongShortRatio retrieves the ratio of users with net long vs net short positions for futures and perpetual swaps
func (ok *Okx) GetLongShortRatio(ctx context.Context, ccy currency.Code, begin, end time.Time, period kline.Interval) ([]LongShortRatio, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	interval := IntervalFromString(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var response []LongShortRatio
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLongShortRatioEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/contracts/long-short-account-ratio", params), nil, &response, request.UnauthenticatedRequest)
}

// GetContractsOpenInterestAndVolume retrieves the open interest and trading volume for futures and perpetual swaps
func (ok *Okx) GetContractsOpenInterestAndVolume(ctx context.Context, ccy currency.Code, begin, end time.Time, period kline.Interval) ([]OpenInterestVolume, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	interval := IntervalFromString(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var response []OpenInterestVolume
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getContractsOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/contracts/open-interest-volume", params), nil, &response, request.UnauthenticatedRequest)
}

// GetOptionsOpenInterestAndVolume retrieves the open interest and trading volume for options
func (ok *Okx) GetOptionsOpenInterestAndVolume(ctx context.Context, ccy currency.Code, period kline.Interval) ([]OpenInterestVolume, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	interval := IntervalFromString(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var response []OpenInterestVolume
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOptionsOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume", params), nil, &response, request.UnauthenticatedRequest)
}

// GetPutCallRatio retrieves the open interest ration and trading volume ratio of calls vs puts
func (ok *Okx) GetPutCallRatio(ctx context.Context, ccy currency.Code, period kline.Interval) ([]OpenInterestVolumeRatio, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	interval := IntervalFromString(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var response []OpenInterestVolumeRatio
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPutCallRatioEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume-ratio", params), nil, &response, request.UnauthenticatedRequest)
}

// GetOpenInterestAndVolumeExpiry retrieves the open interest and trading volume of calls and puts for each upcoming expiration
func (ok *Okx) GetOpenInterestAndVolumeExpiry(ctx context.Context, ccy currency.Code, period kline.Interval) ([]ExpiryOpenInterestAndVolume, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	interval := IntervalFromString(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var resp []ExpiryOpenInterestAndVolume
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume-expiry", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetOpenInterestAndVolumeStrike retrieves the taker volume for both buyers and sellers of calls and puts
func (ok *Okx) GetOpenInterestAndVolumeStrike(ctx context.Context, ccy currency.Code,
	expTime time.Time, period kline.Interval,
) ([]StrikeOpenInterestAndVolume, error) {
	if expTime.IsZero() {
		return nil, errMissingExpiryTimeParameter
	}
	params := url.Values{}
	params.Set("expTime", expTime.UTC().Format("20060102"))
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	interval := IntervalFromString(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var resp []StrikeOpenInterestAndVolume
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume-strike", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetTakerFlow shows the relative buy/sell volume for calls and puts
// It shows whether traders are bullish or bearish on price and volatility
func (ok *Okx) GetTakerFlow(ctx context.Context, ccy currency.Code, period kline.Interval) (*CurrencyTakerFlow, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	interval := IntervalFromString(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var resp *CurrencyTakerFlow
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTakerFlowEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/taker-block-volume", params), nil, &resp, request.UnauthenticatedRequest)
}

// ********************************************************** Affiliate **********************************************************************

// The Affiliate API offers affiliate users a flexible function to query the invitee information
// Simply enter the UID of your direct invitee to access their relevant information, empowering your affiliate business growth and day-to-day business operation
// If you have additional data requirements regarding the Affiliate API, please don't hesitate to contact your BD
// We will reach out to you through your BD to provide more comprehensive API support

// GetInviteesDetail retrieves affiliate invitees details
func (ok *Okx) GetInviteesDetail(ctx context.Context, uid string) (*AffilateInviteesDetail, error) {
	if uid == "" {
		return nil, errUserIDRequired
	}
	var resp *AffilateInviteesDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAffilateInviteesDetailEPL, http.MethodGet, "affiliate/invitee/detail?uid="+uid, nil, &resp, request.AuthenticatedRequest)
}

// GetUserAffiliateRebateInformation this endpoint is used to get the user's affiliate rebate information for affiliate
func (ok *Okx) GetUserAffiliateRebateInformation(ctx context.Context, apiKey string) (*AffilateRebateInfo, error) {
	if apiKey == "" {
		return nil, errInvalidAPIKey
	}
	var resp *AffilateRebateInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getUserAffiliateRebateInformationEPL, http.MethodGet, "users/partner/if-rebate?apiKey="+apiKey, nil, &resp, request.AuthenticatedRequest)
}

// Status

// SystemStatusResponse retrieves the system status.
// state supports valid values 'scheduled', 'ongoing', 'pre_open', 'completed', and 'canceled'
func (ok *Okx) SystemStatusResponse(ctx context.Context, state string) ([]SystemStatusResponse, error) {
	params := url.Values{}
	if state != "" {
		params.Set("state", state)
	}
	var resp []SystemStatusResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEventStatusEPL, http.MethodGet, common.EncodeURLValues("system/status", params), nil, &resp, request.UnauthenticatedRequest)
}

// -------------------------------------------------------  Lending Orders  ------------------------------------------------------

// PlaceLendingOrder places a lending order
func (ok *Okx) PlaceLendingOrder(ctx context.Context, arg *LendingOrderParam) (*LendingOrderResponse, error) {
	if *arg == (LendingOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.Rate <= 0 {
		return nil, errRateRequired
	}
	if arg.Term == "" {
		return nil, errLendingTermIsRequired
	}
	var resp *LendingOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeLendingOrderEPL, http.MethodPost, "finance/fixed-loan/lending-order", arg, &resp, request.AuthenticatedRequest)
}

// AmendLendingOrder amends a lending order
func (ok *Okx) AmendLendingOrder(ctx context.Context, orderID string, changeAmount, rate float64, autoRenewal bool) (string, error) {
	if orderID == "" {
		return "", order.ErrOrderIDNotSet
	}
	arg := &struct {
		OrderID      string  `json:"ordId"`
		ChangeAmount float64 `json:"changeAmt,omitempty,string"`
		Rate         float64 `json:"rate,omitempty,string"`
		AutoRenewal  bool    `json:"autoRenewal,omitempty"`
	}{
		OrderID:      orderID,
		ChangeAmount: changeAmount,
		Rate:         rate,
		AutoRenewal:  autoRenewal,
	}
	var resp OrderIDResponse
	return resp.OrderID, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendLendingOrderEPL, http.MethodPost, "finance/fixed-loan/amend-lending-order", arg, &resp, request.AuthenticatedRequest)
}

// Note: the documentation for Amending lending order has similar url, request method, and parameters to the placing order. Therefore, the implementation is skipped for now.

// GetLendingOrders retrieves list of lending orders.
// State: possible values are 'pending', 'earning', 'expired', 'settled'
func (ok *Okx) GetLendingOrders(ctx context.Context, orderID, state string, ccy currency.Code, startAt, endAt time.Time, limit int64) ([]LendingOrderDetail, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("after", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("before", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if state != "" {
		params.Set("state", state)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LendingOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lendingOrderListEPL, http.MethodGet,
		common.EncodeURLValues("finance/fixed-loan/lending-orders-list", params), nil, &resp, request.AuthenticatedRequest)
}

// GetLendingSubOrderList retrieves a lending sub-orders list
func (ok *Okx) GetLendingSubOrderList(ctx context.Context, orderID, state string, startAt, endAt time.Time, limit int64) ([]LendingSubOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("ordId", orderID)
	if state != "" {
		params.Set("state", state)
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("after", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("before", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LendingSubOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lendingSubOrderListEPL, http.MethodGet, common.EncodeURLValues("finance/fixed-loan/lending-sub-orders", params), nil, &resp, request.AuthenticatedRequest)
}

// Trading Statistics endpoints

// GetFuturesContractsOpenInterestHistory retrieve the contract open interest statistics of futures and perp. This endpoint returns a maximum of 1440 records
func (ok *Okx) GetFuturesContractsOpenInterestHistory(ctx context.Context, instrumentID string, period kline.Interval, startAt, endAt time.Time, limit int64) ([]ContractOpenInterestHistoryItem, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if period != kline.Interval(0) {
		params.Set("period", IntervalFromString(period, true))
	}
	if !startAt.IsZero() {
		params.Set("begin", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("end", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []ContractOpenInterestHistoryItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rubikGetContractOpenInterestHistoryEPL, http.MethodGet,
		common.EncodeURLValues("rubik/stat/contracts/open-interest-history", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetFuturesContractTakerVolume retrieve the contract taker volume for both buyers and sellers. This endpoint returns a maximum of 1440 records.
// The unit of buy/sell volume, the default is 1. '0': Crypto '1': Contracts '2': U
func (ok *Okx) GetFuturesContractTakerVolume(ctx context.Context, instrumentID string, period kline.Interval, unit, limit int64, startAt, endAt time.Time) ([]ContractTakerVolume, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if period != kline.Interval(0) {
		params.Set("period", IntervalFromString(period, true))
	}
	if !startAt.IsZero() {
		params.Set("begin", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("end", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if unit != 1 {
		params.Set("unit", strconv.FormatInt(unit, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []ContractTakerVolume
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rubikContractTakerVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/taker-volume-contract", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetFuturesContractLongShortAccountRatio retrieve the account long/short ratio of a contract. This endpoint returns a maximum of 1440 records
func (ok *Okx) GetFuturesContractLongShortAccountRatio(ctx context.Context, instrumentID string, period kline.Interval, startAt, endAt time.Time, limit int64) ([]TopTraderContractsLongShortRatio, error) {
	return ok.getTopTradersFuturesContractLongShortRatio(ctx, instrumentID, "rubik/stat/contracts/long-short-account-ratio-contract", period, startAt, endAt, limit)
}

// GetTopTradersFuturesContractLongShortAccountRatio retrieve the account net long/short ratio of a contract for top traders.
// Top traders refer to the top 5% of traders with the largest open position value.
// This endpoint returns a maximum of 1440 records
func (ok *Okx) GetTopTradersFuturesContractLongShortAccountRatio(ctx context.Context, instrumentID string, period kline.Interval, startAt, endAt time.Time, limit int64) ([]TopTraderContractsLongShortRatio, error) {
	return ok.getTopTradersFuturesContractLongShortRatio(ctx, instrumentID, "rubik/stat/contracts/long-short-account-ratio-contract-top-trader", period, startAt, endAt, limit)
}

// GetTopTradersFuturesContractLongShortPositionRatio retrieve the position long/short ratio of a contract for top traders. Top traders refer to the top 5% of traders with the largest open position value. This endpoint returns a maximum of 1440 records
func (ok *Okx) GetTopTradersFuturesContractLongShortPositionRatio(ctx context.Context, instrumentID string, period kline.Interval, startAt, endAt time.Time, limit int64) ([]TopTraderContractsLongShortRatio, error) {
	return ok.getTopTradersFuturesContractLongShortRatio(ctx, instrumentID, "rubik/stat/contracts/long-short-position-ratio-contract-top-trader", period, startAt, endAt, limit)
}

func (ok *Okx) getTopTradersFuturesContractLongShortRatio(ctx context.Context, instrumentID, path string, period kline.Interval, startAt, endAt time.Time, limit int64) ([]TopTraderContractsLongShortRatio, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if period != kline.Interval(0) {
		params.Set("period", IntervalFromString(period, true))
	}
	if !startAt.IsZero() {
		params.Set("begin", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("end", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []TopTraderContractsLongShortRatio
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rubikTopTradersContractLongShortRatioEPL, http.MethodGet, common.EncodeURLValues(path, params), nil, &resp, request.UnauthenticatedRequest)
}

// GetAnnouncements get announcements, the response is sorted by pTime with the most recent first. The sort will not be affected if the announcement is updated. Every page has 20 records
//
// There are differences between public endpoint and private endpoint.
// For public endpoint, the response is restricted based on your request IP.
// For private endpoint, the response is restricted based on your country of residence
func (ok *Okx) GetAnnouncements(ctx context.Context, announcementType string, page int64) (*AnnouncementDetail, error) {
	params := url.Values{}
	if announcementType != "" {
		params.Set("annType", announcementType)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *AnnouncementDetail
	if ok.AreCredentialsValid(ctx) {
		return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAnnouncementsEPL, http.MethodGet, common.EncodeURLValues("support/announcements", params), nil, &resp, request.AuthenticatedRequest)
	}
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAnnouncementsEPL, http.MethodGet, common.EncodeURLValues("support/announcements", params), nil, &resp, request.UnauthenticatedRequest)
}

// GetAnnouncementTypes represents a list of announcement types
func (ok *Okx) GetAnnouncementTypes(ctx context.Context) ([]AnnouncementTypeInfo, error) {
	var resp []AnnouncementTypeInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAnnouncementTypeEPL, http.MethodGet,
		"support/announcement-types", nil, &resp, request.UnauthenticatedRequest)
}

// Fiat endpoints

// GetDepositOrderDetail retrieves fiat deposit order detail
func (ok *Okx) GetDepositOrderDetail(ctx context.Context, orderID string) (*FiatOrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("ordID", orderID)
	var resp *FiatOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDepositOrderDetailEPL, http.MethodGet, common.EncodeURLValues("fiat/deposit", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFiatDepositOrderHistory retrieves fiat deposit order history
func (ok *Okx) GetFiatDepositOrderHistory(ctx context.Context, ccy currency.Code, paymentMethod, state string, after, before time.Time, limit int64) ([]FiatOrderDetail, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if paymentMethod != "" {
		params.Set("paymentMethod", paymentMethod)
	}
	if state != "" {
		params.Set("state", state)
	}
	if !after.IsZero() && !before.IsZero() {
		err := common.StartEndTimeCheck(after, before)
		if err != nil {
			return nil, err
		}
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FiatOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDepositOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("fiat/deposit-order-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetWithdrawalOrderDetail retrieves fiat withdrawal order detail
func (ok *Okx) GetWithdrawalOrderDetail(ctx context.Context, orderID string) (*FiatOrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("ordId", orderID)
	var resp *FiatOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getWithdrawalOrderDetailEPL, http.MethodGet,
		common.EncodeURLValues("fiat/withdrawal", params), &map[string]string{"ordId": orderID}, &resp, request.AuthenticatedRequest)
}

// GetFiatWithdrawalOrderHistory retrieves fiat withdrawal order history
func (ok *Okx) GetFiatWithdrawalOrderHistory(ctx context.Context, ccy currency.Code, paymentMethod, state string, after, before time.Time, limit int64) ([]FiatOrderDetail, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if paymentMethod != "" {
		params.Set("paymentMethod", paymentMethod)
	}
	if state != "" {
		params.Set("state", state)
	}
	if !after.IsZero() && !before.IsZero() {
		err := common.StartEndTimeCheck(after, before)
		if err != nil {
			return nil, err
		}
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FiatOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFiatWithdrawalOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("fiat/withdrawal-order-history", params), nil, &resp, request.AuthenticatedRequest)
}

// CancelWithdrawalOrder cancel a pending fiat withdrawal order, currently only applicable to TRY
func (ok *Okx) CancelWithdrawalOrder(ctx context.Context, orderID string) (*OrderIDAndState, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *OrderIDAndState
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelWithdrawalOrderEPL, http.MethodPost, "fiat/cancel-withdrawal", &map[string]string{"ordId": orderID}, &resp, request.AuthenticatedRequest)
}

// CreateWithdrawalOrder initiate a fiat withdrawal request (Authenticated endpoint, Only for API keys with "Withdrawal" access)
func (ok *Okx) CreateWithdrawalOrder(ctx context.Context, ccy currency.Code, paymentAccountID, paymentMethod, clientID string, amount float64) (*FiatOrderDetail, error) {
	if paymentAccountID == "" {
		return nil, fmt.Errorf("%w, payment account ID is required", errIDNotSet)
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if paymentMethod == "" {
		return nil, errPaymentMethodRequired
	}
	if clientID == "" {
		return nil, fmt.Errorf("%w, client ID is required", errIDNotSet)
	}
	arg := &struct {
		PaymentMethod string  `json:"paymentMethod"`
		PaymentAcctID string  `json:"paymentAcctId"`
		ClientID      string  `json:"clientId"`
		Amount        float64 `json:"amt,string"`
		Currency      string  `json:"ccy"`
	}{
		PaymentMethod: paymentMethod,
		PaymentAcctID: paymentAccountID,
		ClientID:      clientID,
		Amount:        amount,
		Currency:      ccy.String(),
	}
	var resp *FiatOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, createWithdrawalOrderEPL, http.MethodPost, "fiat/create-withdrawal", arg, &resp, request.AuthenticatedRequest)
}

// GetFiatWithdrawalPaymentMethods to display all the available fiat withdrawal payment methods
func (ok *Okx) GetFiatWithdrawalPaymentMethods(ctx context.Context, ccy currency.Code) (*FiatWithdrawalPaymentMethods, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	var resp *FiatWithdrawalPaymentMethods
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getWithdrawalPaymentMethodsEPL, http.MethodGet,
		common.EncodeURLValues("fiat/withdrawal-payment-methods", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFiatDepositPaymentMethods to display all the available fiat deposit payment methods
func (ok *Okx) GetFiatDepositPaymentMethods(ctx context.Context, ccy currency.Code) (*FiatDepositPaymentMethods, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	var resp *FiatDepositPaymentMethods
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFiatDepositPaymentMethodsEPL, http.MethodGet,
		common.EncodeURLValues("fiat/deposit-payment-methods", params), nil, &resp, request.AuthenticatedRequest)
}

/*
SendHTTPRequest sends an http request, optionally with a JSON payload
URL arguments must be encoded in the request path
result must be a pointer
The response will be unmarshalled first into []any{result}, which matches most APIs, and fallback to directly into result
*/
func (ok *Okx) SendHTTPRequest(ctx context.Context, ep exchange.URL, f request.EndpointLimit, httpMethod, requestPath string, data, result any, requestType request.AuthType) (err error) {
	endpoint, err := ok.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var resp struct {
		Code types.Number    `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	newRequest := func() (*request.Item, error) {
		var payload []byte
		if data != nil {
			payload, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
		}
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		if simulate, okay := ctx.Value(testNetVal).(bool); okay && simulate {
			headers["x-simulated-trading"] = "1"
		}
		if requestType == request.AuthenticatedRequest {
			creds, err := ok.GetCredentials(ctx)
			if err != nil {
				return nil, err
			}
			signPath := "/" + apiPath + requestPath
			utcTime := time.Now().UTC().Format(time.RFC3339)
			hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(utcTime+httpMethod+signPath+string(payload)), []byte(creds.Secret))
			if err != nil {
				return nil, err
			}
			headers["OK-ACCESS-KEY"] = creds.Key
			headers["OK-ACCESS-SIGN"] = base64.StdEncoding.EncodeToString(hmac)
			headers["OK-ACCESS-TIMESTAMP"] = utcTime
			headers["OK-ACCESS-PASSPHRASE"] = creds.ClientID
		}
		return &request.Item{
			Method:        strings.ToUpper(httpMethod),
			Path:          endpoint + requestPath,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &resp,
			Verbose:       ok.Verbose,
			HTTPDebugging: ok.HTTPDebugging,
			HTTPRecording: ok.HTTPRecording,
		}, nil
	}
	if err := ok.SendPayload(ctx, f, newRequest, requestType); err != nil {
		return err
	}
	if resp.Code.Int64() != 0 {
		if requestType == request.AuthenticatedRequest {
			err = request.ErrAuthRequestFailed
		}
		if resp.Msg != "" {
			return common.AppendError(err, fmt.Errorf("error code: `%d`; message: %q", resp.Code.Int64(), resp.Msg))
		}
		if mErr, ok := ErrorCodes[resp.Code.String()]; ok {
			return common.AppendError(err, mErr)
		}
		return common.AppendError(err, fmt.Errorf("error code: `%d`", resp.Code.Int64()))
	}

	// First see if resp.Data can unmarshal into a slice of result, which is true for most APIs
	if sliceErr := json.Unmarshal(resp.Data, &[]any{result}); sliceErr != nil {
		// Otherwise, resp.Data should unmarshal directly into result; e.g. index-components, support-coin, and taker-block-volume
		if directErr := json.Unmarshal(resp.Data, result); directErr != nil {
			return fmt.Errorf("cannot unmarshal as a slice of result (error: %w) or as a reference to result (error: %w)", sliceErr, directErr)
		}
	}

	return nil
}

func getStatusError(statusCode int64, statusMessage string) error {
	if statusCode == 0 {
		return nil
	}
	return fmt.Errorf("status code: `%d` status message: %q", statusCode, statusMessage)
}
