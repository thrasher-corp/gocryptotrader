package coinbase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errOrderIDRequired                   = errors.New("order ID required")
	errClientOrderIDRequired             = errors.New("client order ID required")
	errClientOrderIDTooLong              = errors.New("client order ID longer than 128 characters")
	errInvalidOrderSide                  = errors.New("invalid order side, want BUY or SELL")
	errOrderConfigurationRequired        = errors.New("order configuration required")
	errInvalidAttachedOrderConfiguration = errors.New("ineligible attached order configuration")
	errAttachedOrderSizeSet              = errors.New("attached order size must be unset, as it takes its parent's")
	errInvalidOrderLeverage              = errors.New("order leverage must not be negative")
	errInvalidOrderMarginType            = errors.New("invalid order margin type")
	errInvalidStopDirection              = errors.New("invalid stop direction")
	errInvalidSORPreference              = errors.New("invalid smart order routing preference")
	errInvalidPredictionSide             = errors.New("invalid prediction side")
	errInvalidEquityTradingSession       = errors.New("invalid equity trading session")
	errInvalidDisplayedOrderConfig       = errors.New("invalid displayed order config")
	errInvalidCostBasisMethod            = errors.New("invalid cost basis method")
	errInvalidScaledPriceDistribution    = errors.New("invalid scaled order price distribution")
	errInvalidScaledSizeDistribution     = errors.New("invalid scaled order size distribution")
	errInvalidOrderStatus                = errors.New("invalid order status")
	errInvalidOrderTimeInForce           = errors.New("invalid order time in force")
	errInvalidOrderType                  = errors.New("invalid order type")
	errInvalidOrderProductType           = errors.New("invalid order product type")
	errInvalidOrderPlacementSource       = errors.New("invalid order placement source")
	errInvalidOrderContractExpiryType    = errors.New("invalid order contract expiry type")
	errInvalidOrdersSortBy               = errors.New("invalid order list sort field")
	errInvalidFillsSortBy                = errors.New("invalid fill list sort field")
	errOrderNotCreated                   = errors.New("order not created")
	errOrderNotEdited                    = errors.New("order not edited")
	errOrderEditPreviewFailed            = errors.New("order edit preview reported failures")
	errOrderPreviewFailed                = errors.New("order preview reported failures")
	errCancelOrderFailed                 = errors.New("order cancel request not submitted")
)

var (
	orderSides               = []string{OrderSideBuy, OrderSideSell}
	orderMarginTypes         = []string{OrderMarginTypeCross, OrderMarginTypeIsolated}
	stopDirections           = []string{StopDirectionStopUp, StopDirectionStopDown}
	sorPreferences           = []string{SORPreferenceUnspecified, SORPreferenceEnabled, SORPreferenceDisabled}
	predictionSides          = []string{PredictionSideUnknown, PredictionSideYes, PredictionSideNo}
	scaledPriceDistributions = []string{ScaledPriceDistributionFlat, ScaledPriceDistributionLinearIncreasing, ScaledPriceDistributionLinearDecreasing, ScaledPriceDistributionCustom}
	scaledSizeDistributions  = []string{ScaledSizeDistributionUnknown, ScaledSizeDistributionIncreasing, ScaledSizeDistributionDecreasing, ScaledSizeDistributionEvenlySplit, ScaledSizeDistributionCustom}
	orderTimeInForces        = []string{TimeInForceUnknown, TimeInForceGoodUntilDateTime, TimeInForceGoodUntilCancelled, TimeInForceImmediateOrCancel, TimeInForceFillOrKill}
	ordersSortFields         = []string{OrdersSortByUnknown, OrdersSortByLimitPrice, OrdersSortByLastFillTime, OrdersSortByLastUpdateTime}
	fillsSortFields          = []string{FillsSortByUnknown, FillsSortByPrice, FillsSortByTradeTime}
	equityTradingSessions    = []string{EquityTradingSessionUnknown, EquityTradingSessionNormal, EquityTradingSessionAfterHours, EquityTradingSessionMultiSession, EquityTradingSessionOvernight, EquityTradingSessionPreMarket}
	displayedOrderConfigs    = []string{DisplayedOrderConfigUnknown, DisplayedOrderConfigInstantGFD, DisplayedOrderConfigLimitGFD, DisplayedOrderConfigLimitGTC, DisplayedOrderConfigMarketGFD, DisplayedOrderConfigExerciseGFD}
	costBasisMethods         = []string{CostBasisMethodUnspecified, CostBasisMethodHIFO, CostBasisMethodLIFO, CostBasisMethodFIFO, CostBasisMethodSpecificID}
	orderPlacementSources    = []string{OrderPlacementSourceUnknown, OrderPlacementSourceRetailSimple, OrderPlacementSourceRetailAdvanced, OrderPlacementSourceRetailAdmin, OrderPlacementSourceRetailRaise}
	orderContractExpiryTypes = []string{"UNKNOWN_CONTRACT_EXPIRY_TYPE", ContractExpiryTypeExpiring, ContractExpiryTypePerpetual}
	orderProductTypes        = []string{"UNKNOWN_PRODUCT_TYPE", ProductTypeSpot, ProductTypeFuture, ProductTypeEquity, "OPTION_GROUP", "FUTURE_GROUP"}
	orderStatuses            = []string{OrderStatusPending, OrderStatusOpen, OrderStatusFilled, OrderStatusCancelled, OrderStatusExpired, OrderStatusFailed, OrderStatusUnknown, OrderStatusQueued, OrderStatusCancelQueued, OrderStatusEditQueued}
	orderTypes               = []string{OrderTypeUnknown, OrderTypeMarket, OrderTypeLimit, OrderTypeStop, OrderTypeStopLimit, OrderTypeBracket, OrderTypeTWAP, OrderTypeRollOpen, OrderTypeRollClose, OrderTypeLiquidation, OrderTypeScaled}
)

// CancelOrders sends cancel requests for orders, by their IDs, and reports whether Coinbase submitted each; a
// submitted request completes asynchronously. When any request is not submitted, the response is returned with the
// error of each, order.ErrOrderNotFound for an order Coinbase does not know
func (e *Exchange) CancelOrders(ctx context.Context, orderIDs []string) (*CancelOrdersResponse, error) {
	if len(orderIDs) == 0 || slices.Contains(orderIDs, "") {
		return nil, errOrderIDRequired
	}
	var resp *CancelOrdersResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"orders/batch_cancel", nil, &CancelOrdersWireRequest{OrderIDs: orderIDs}, &resp); err != nil {
		return nil, orderNotFoundError(err)
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, resp.Err()
}

// ClosePosition places an order closing the account's open position in a US derivative product, Size contracts of
// it when Size is set. When Coinbase creates no order, the response is returned with errOrderNotCreated and its
// reasons
func (e *Exchange) ClosePosition(ctx context.Context, req *ClosePositionRequest) (*CreateOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.ClientOrderID == "" {
		return nil, errClientOrderIDRequired
	}
	if req.ProductID == "" {
		return nil, errProductIDRequired
	}
	if err := checkOrderNumbers(errInvalidAmount, req.Size); err != nil {
		return nil, err
	}
	body := &ClosePositionWireRequest{ClientOrderID: req.ClientOrderID, ProductID: req.ProductID, Size: orderDecimalParam(req.Size)}
	var resp *CreateOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"orders/close_position", nil, body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, resp.Err()
}

// CreateOrder places an order; a client order ID already used returns the order it identifies rather than creating
// another. An equity order names the equity by its product ID, not its display ticker, and its equity metadata
// selects its session and time in force: a market_market_ioc order executes with MARKET_GFD in the normal session
// only, and a limit_limit_gtc order with LIMIT_GFD or LIMIT_GTC; outside the normal session an equity order must be a
// limit order for whole shares sized in the base currency. When Coinbase creates no order, the response is returned
// with errOrderNotCreated and its reasons
func (e *Exchange) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	var resp *CreateOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"orders", nil, body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, resp.Err()
}

// EditOrder changes an open order's price and size, and its attached take profit/stop loss order. When Coinbase does
// not place the edit, the response is returned with errOrderNotEdited and its reasons, order.ErrOrderNotFound for an
// order Coinbase does not know
func (e *Exchange) EditOrder(ctx context.Context, req *EditOrderRequest) (*EditOrderResponse, error) {
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	var resp *EditOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"orders/edit", nil, body, &resp); err != nil {
		return nil, orderNotFoundError(err)
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	if err := resp.Err(); err != nil {
		return resp, err
	}
	// The user channel reports no order's size, so the edited size is recorded for its updates
	e.orderSizes.set(req.OrderID, req.Size, 0)
	return resp, nil
}

// EditOrderPreview previews an order edit, returning the order's totals as edited. When the edit would fail, the
// response is returned with errOrderEditPreviewFailed and its reasons, order.ErrOrderNotFound for an order Coinbase
// does not know
func (e *Exchange) EditOrderPreview(ctx context.Context, req *EditOrderRequest) (*EditOrderPreviewResponse, error) {
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	var resp *EditOrderPreviewResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"orders/edit_preview", nil, body, &resp); err != nil {
		return nil, orderNotFoundError(err)
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, resp.Err()
}

// GetOrder returns an order by its ID; an order Coinbase does not know returns order.ErrOrderNotFound
func (e *Exchange) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	if orderID == "" {
		return nil, errOrderIDRequired
	}
	var resp *GetOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"orders/historical/"+orderID, nil, nil, &resp); err != nil {
		return nil, orderNotFoundError(err)
	}
	if resp == nil || resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// ListFills returns a page of the account's fills, 100 unless the request's limit says otherwise, filtered by the
// request; the response's cursor starts the next page
func (e *Exchange) ListFills(ctx context.Context, req *ListFillsRequest) (*ListFillsResponse, error) {
	params, err := listFillsParams(req)
	if err != nil {
		return nil, err
	}
	var resp *ListFillsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"orders/historical/fills", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ListOrders returns a page of the account's orders, filtered by the request; Coinbase lists only orders placed
// through Advanced Trade unless the request selects another placement source. While the response has a next page,
// its cursor starts it
func (e *Exchange) ListOrders(ctx context.Context, req *ListOrdersRequest) (*ListOrdersResponse, error) {
	params, err := listOrdersParams(req)
	if err != nil {
		return nil, err
	}
	var resp *ListOrdersResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"orders/historical/batch", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// PreviewOrder previews an order as CreateOrder would place it, returning its totals, commission and a preview ID to
// place it with. When the order would fail, the response is returned with errOrderPreviewFailed and its reasons
func (e *Exchange) PreviewOrder(ctx context.Context, req *PreviewOrderRequest) (*PreviewOrderResponse, error) {
	body, err := req.body()
	if err != nil {
		return nil, err
	}
	var resp *PreviewOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"orders/preview", nil, body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, resp.Err()
}

// orderNotFoundError marks the error Coinbase reports for an order it does not know with order.ErrOrderNotFound,
// which the wrapper and engine tell a missing order by
func orderNotFoundError(err error) error {
	if apiErr, ok := errors.AsType[*APIError](err); ok && apiErr.ErrorType == "NOT_FOUND" {
		return fmt.Errorf("%w: %w", order.ErrOrderNotFound, err)
	}
	return err
}

// editOrderFailure returns a failed edit's error: the sentinel with each reason Coinbase gave, also wrapping
// order.ErrOrderNotFound when one says Coinbase does not know the order
func editOrderFailure(sentinel error, editErrs []EditOrderError) error {
	var reasons []string
	var notFound bool
	for i := range editErrs {
		for _, reason := range []string{editErrs[i].EditFailureReason, editErrs[i].PreviewFailureReason} {
			if reason == "" || reason == editOrderFailureReasonUnknown || reason == previewFailureReasonUnknown {
				continue
			}
			notFound = notFound || slices.Contains(editOrderNotFoundReasons, reason)
			reasons = append(reasons, reason)
		}
	}
	if len(reasons) == 0 {
		return fmt.Errorf("%w: no reason given", sentinel)
	}
	if notFound {
		return fmt.Errorf("%w: %w: %s", sentinel, order.ErrOrderNotFound, strings.Join(reasons, ", "))
	}
	return fmt.Errorf("%w: %s", sentinel, strings.Join(reasons, ", "))
}

// body validates the order and returns the body Create Order sends
func (r *CreateOrderRequest) body() (*OrderWireRequest, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.ClientOrderID == "" {
		return nil, errClientOrderIDRequired
	}
	if n := utf8.RuneCountInString(r.ClientOrderID); n > maximumClientOrderIDLength {
		return nil, fmt.Errorf("%w: %d characters", errClientOrderIDTooLong, n)
	}
	if err := checkOrderEnum(r.SORPreference, errInvalidSORPreference, sorPreferences); err != nil {
		return nil, err
	}
	body, err := (&PreviewOrderRequest{
		ProductID:                  r.ProductID,
		Side:                       r.Side,
		OrderConfiguration:         r.OrderConfiguration,
		Leverage:                   r.Leverage,
		MarginType:                 r.MarginType,
		AttachedOrderConfiguration: r.AttachedOrderConfiguration,
		EquityOrderMetadata:        r.EquityOrderMetadata,
		PredictionMetadata:         r.PredictionMetadata,
		CostBasisMethod:            r.CostBasisMethod,
	}).body()
	if err != nil {
		return nil, err
	}
	body.ClientOrderID = r.ClientOrderID
	body.PreviewID = r.PreviewID
	body.SORPreference = r.SORPreference
	return body, nil
}

// body validates the order and returns the body Preview Order sends
func (r *PreviewOrderRequest) body() (*OrderWireRequest, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.ProductID == "" {
		return nil, errProductIDRequired
	}
	if !slices.Contains(orderSides, r.Side) {
		return nil, fmt.Errorf("%w: %q", errInvalidOrderSide, r.Side)
	}
	if !isOrderConfigurationSet(r.OrderConfiguration) {
		return nil, errOrderConfigurationRequired
	}
	configuration, err := newOrderConfigurationBody(r.OrderConfiguration)
	if err != nil {
		return nil, err
	}
	attached, err := attachedOrderConfigurationBody(r.AttachedOrderConfiguration)
	if err != nil {
		return nil, err
	}
	if r.Leverage < 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidOrderLeverage, r.Leverage)
	}
	if err := checkOrderEnum(r.MarginType, errInvalidOrderMarginType, orderMarginTypes); err != nil {
		return nil, err
	}
	if err := checkOrderEnum(r.CostBasisMethod, errInvalidCostBasisMethod, costBasisMethods); err != nil {
		return nil, err
	}
	if err := r.EquityOrderMetadata.validate(); err != nil {
		return nil, err
	}
	prediction, err := r.PredictionMetadata.body()
	if err != nil {
		return nil, err
	}
	return &OrderWireRequest{
		ProductID:                  r.ProductID,
		Side:                       r.Side,
		OrderConfiguration:         configuration,
		Leverage:                   orderDecimalParam(r.Leverage),
		MarginType:                 r.MarginType,
		AttachedOrderConfiguration: attached,
		PredictionMetadata:         prediction,
		EquityOrderMetadata:        r.EquityOrderMetadata,
		CostBasisMethod:            r.CostBasisMethod,
	}, nil
}

// body validates the edit and returns the body Edit Order and Edit Order Preview send
func (r *EditOrderRequest) body() (*EditOrderWireRequest, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.OrderID == "" {
		return nil, errOrderIDRequired
	}
	if r.Price <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidPrice, r.Price)
	}
	if r.Size <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, r.Size)
	}
	if err := checkOrderNumbers(errInvalidPrice, r.StopPrice, r.AverageEntryPrice); err != nil {
		return nil, err
	}
	attached, err := editAttachedOrderConfigurationBody(r.AttachedOrderConfiguration)
	if err != nil {
		return nil, err
	}
	return &EditOrderWireRequest{
		OrderID:                    r.OrderID,
		Price:                      formatFloat(r.Price),
		Size:                       formatFloat(r.Size),
		AttachedOrderConfiguration: attached,
		CancelAttachedOrder:        r.CancelAttachedOrder,
		StopPrice:                  orderDecimalParam(r.StopPrice),
		AverageEntryPrice:          orderDecimalParam(r.AverageEntryPrice),
	}, nil
}

// validate checks an equity order's session and time in force; nil metadata leaves both to Coinbase's defaults
func (m *EquityOrderMetadata) validate() error {
	if m == nil {
		return nil
	}
	if err := checkOrderEnum(m.EquityTradingSession, errInvalidEquityTradingSession, equityTradingSessions); err != nil {
		return err
	}
	return checkOrderEnum(m.DisplayedOrderConfig, errInvalidDisplayedOrderConfig, displayedOrderConfigs)
}

// body validates the prediction metadata and returns the body an order request sends; nil metadata sends none
func (m *PredictionMetadataRequest) body() (*PredictionMetadataWire, error) {
	if m == nil {
		return nil, nil
	}
	if err := checkOrderEnum(m.PredictionSide, errInvalidPredictionSide, predictionSides); err != nil {
		return nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, m.PreviewOrderEstimatedAverageFilledPrice); err != nil {
		return nil, err
	}
	return &PredictionMetadataWire{
		PredictionSide:                          m.PredictionSide,
		PreviewOrderEstimatedAverageFilledPrice: orderDecimalParam(m.PreviewOrderEstimatedAverageFilledPrice),
		SupportsFractionalBaseSize:              m.SupportsFractionalBaseSize,
	}, nil
}

// isOrderConfigurationSet reports whether an order configuration is set, as a nil one, typed or not, has no body
func isOrderConfigurationSet(configuration OrderConfigurationRequest) bool {
	return common.NilGuard(configuration) == nil
}

// newOrderConfigurationBody validates an order configuration and returns its body
func newOrderConfigurationBody(configuration OrderConfigurationRequest) (OrderConfigurationWire, error) {
	if err := common.NilGuard(configuration); err != nil {
		return nil, err
	}
	key, leg, err := configuration.orderConfiguration()
	if err != nil {
		return nil, err
	}
	return OrderConfigurationWire{key: leg}, nil
}

// attachedOrderConfigurationBody returns the body of the take profit/stop loss order a new order attaches, which can
// only be a trigger bracket without a size, as it takes its parent's; a nil configuration, typed or not, attaches none
func attachedOrderConfigurationBody(configuration OrderConfigurationRequest) (OrderConfigurationWire, error) {
	if !isOrderConfigurationSet(configuration) {
		return nil, nil
	}
	bracket, ok := configuration.(*TriggerBracketGTCRequest)
	if !ok {
		return nil, fmt.Errorf("%w: %T, want *TriggerBracketGTCRequest", errInvalidAttachedOrderConfiguration, configuration)
	}
	if bracket.BaseSize != 0 {
		return nil, errAttachedOrderSizeSet
	}
	return newOrderConfigurationBody(configuration)
}

// editAttachedOrderConfigurationBody returns the body of the attached order configuration an edit sets, which can be a
// trigger bracket, limit or stop limit configuration; a nil configuration, typed or not, sets none
func editAttachedOrderConfigurationBody(configuration OrderConfigurationRequest) (OrderConfigurationWire, error) {
	if !isOrderConfigurationSet(configuration) {
		return nil, nil
	}
	switch configuration.(type) {
	case *TriggerBracketGTCRequest, *LimitLimitGTCRequest, *StopLimitStopLimitGTCRequest:
		return newOrderConfigurationBody(configuration)
	}
	return nil, fmt.Errorf("%w: %T, want *TriggerBracketGTCRequest, *LimitLimitGTCRequest or *StopLimitStopLimitGTCRequest", errInvalidAttachedOrderConfiguration, configuration)
}

// orderConfiguration validates the market order and returns its body
func (r *MarketMarketIOCRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.QuoteSize, r.BaseSize); err != nil {
		return "", nil, err
	}
	return "market_market_ioc", &OrderConfigurationLegWire{QuoteSize: orderDecimalParam(r.QuoteSize), BaseSize: orderDecimalParam(r.BaseSize)}, nil
}

// orderConfiguration validates the fill or kill market order and returns its body
func (r *MarketMarketFOKRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.QuoteSize, r.BaseSize); err != nil {
		return "", nil, err
	}
	return "market_market_fok", &OrderConfigurationLegWire{QuoteSize: orderDecimalParam(r.QuoteSize), BaseSize: orderDecimalParam(r.BaseSize)}, nil
}

// orderConfiguration validates the smart order routed limit order and returns its body
func (r *SORLimitIOCRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.QuoteSize, r.BaseSize); err != nil {
		return "", nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, r.LimitPrice); err != nil {
		return "", nil, err
	}
	return "sor_limit_ioc", &OrderConfigurationLegWire{
		QuoteSize:  orderDecimalParam(r.QuoteSize),
		BaseSize:   orderDecimalParam(r.BaseSize),
		LimitPrice: orderDecimalParam(r.LimitPrice),
	}, nil
}

// orderConfiguration validates the good til cancelled limit order and returns its body
func (r *LimitLimitGTCRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.QuoteSize, r.BaseSize); err != nil {
		return "", nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, r.LimitPrice); err != nil {
		return "", nil, err
	}
	return "limit_limit_gtc", &OrderConfigurationLegWire{
		QuoteSize:  orderDecimalParam(r.QuoteSize),
		BaseSize:   orderDecimalParam(r.BaseSize),
		LimitPrice: orderDecimalParam(r.LimitPrice),
		PostOnly:   r.PostOnly,
	}, nil
}

// orderConfiguration validates the good til date limit order and returns its body
func (r *LimitLimitGTDRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.QuoteSize, r.BaseSize); err != nil {
		return "", nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, r.LimitPrice); err != nil {
		return "", nil, err
	}
	return "limit_limit_gtd", &OrderConfigurationLegWire{
		QuoteSize:  orderDecimalParam(r.QuoteSize),
		BaseSize:   orderDecimalParam(r.BaseSize),
		LimitPrice: orderDecimalParam(r.LimitPrice),
		EndTime:    orderTimeParam(r.EndTime),
		PostOnly:   r.PostOnly,
	}, nil
}

// orderConfiguration validates the fill or kill limit order and returns its body
func (r *LimitLimitFOKRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.QuoteSize, r.BaseSize); err != nil {
		return "", nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, r.LimitPrice); err != nil {
		return "", nil, err
	}
	return "limit_limit_fok", &OrderConfigurationLegWire{
		QuoteSize:  orderDecimalParam(r.QuoteSize),
		BaseSize:   orderDecimalParam(r.BaseSize),
		LimitPrice: orderDecimalParam(r.LimitPrice),
	}, nil
}

// orderConfiguration validates the time weighted average price order and returns its body
func (r *TWAPLimitGTDRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.QuoteSize, r.BaseSize, r.BucketSize); err != nil {
		return "", nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, r.LimitPrice); err != nil {
		return "", nil, err
	}
	// checkWindow refuses a start after now, which a scheduled order's window has, so only the order is checked
	if !r.StartTime.IsZero() && r.StartTime.After(r.EndTime) && !r.EndTime.IsZero() {
		return "", nil, fmt.Errorf("%w: %s to %s", common.ErrStartAfterEnd, r.StartTime, r.EndTime)
	}
	if r.BucketDuration < 0 {
		return "", nil, fmt.Errorf("%w: bucket duration %s", errInvalidDuration, r.BucketDuration)
	}
	var numberOfBuckets string
	if r.NumberOfBuckets != 0 {
		numberOfBuckets = strconv.FormatUint(r.NumberOfBuckets, 10)
	}
	return "twap_limit_gtd", &OrderConfigurationLegWire{
		QuoteSize:       orderDecimalParam(r.QuoteSize),
		BaseSize:        orderDecimalParam(r.BaseSize),
		StartTime:       orderTimeParam(r.StartTime),
		EndTime:         orderTimeParam(r.EndTime),
		LimitPrice:      orderDecimalParam(r.LimitPrice),
		NumberOfBuckets: numberOfBuckets,
		BucketSize:      orderDecimalParam(r.BucketSize),
		BucketDuration:  orderDurationParam(r.BucketDuration),
	}, nil
}

// orderConfiguration validates the good til cancelled stop limit order and returns its body
func (r *StopLimitStopLimitGTCRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.BaseSize); err != nil {
		return "", nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, r.LimitPrice, r.StopPrice); err != nil {
		return "", nil, err
	}
	if err := checkOrderEnum(r.StopDirection, errInvalidStopDirection, stopDirections); err != nil {
		return "", nil, err
	}
	return "stop_limit_stop_limit_gtc", &OrderConfigurationLegWire{
		BaseSize:      orderDecimalParam(r.BaseSize),
		LimitPrice:    orderDecimalParam(r.LimitPrice),
		StopPrice:     orderDecimalParam(r.StopPrice),
		StopDirection: r.StopDirection,
	}, nil
}

// orderConfiguration validates the good til date stop limit order and returns its body
func (r *StopLimitStopLimitGTDRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.BaseSize); err != nil {
		return "", nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, r.LimitPrice, r.StopPrice); err != nil {
		return "", nil, err
	}
	if err := checkOrderEnum(r.StopDirection, errInvalidStopDirection, stopDirections); err != nil {
		return "", nil, err
	}
	return "stop_limit_stop_limit_gtd", &OrderConfigurationLegWire{
		BaseSize:      orderDecimalParam(r.BaseSize),
		LimitPrice:    orderDecimalParam(r.LimitPrice),
		StopPrice:     orderDecimalParam(r.StopPrice),
		EndTime:       orderTimeParam(r.EndTime),
		StopDirection: r.StopDirection,
	}, nil
}

// orderConfiguration validates the good til cancelled bracket order and returns its body
func (r *TriggerBracketGTCRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.BaseSize); err != nil {
		return "", nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, r.LimitPrice, r.StopTriggerPrice); err != nil {
		return "", nil, err
	}
	return "trigger_bracket_gtc", &OrderConfigurationLegWire{
		BaseSize:         orderDecimalParam(r.BaseSize),
		LimitPrice:       orderDecimalParam(r.LimitPrice),
		StopTriggerPrice: orderDecimalParam(r.StopTriggerPrice),
	}, nil
}

// orderConfiguration validates the good til date bracket order and returns its body
func (r *TriggerBracketGTDRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.BaseSize); err != nil {
		return "", nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, r.LimitPrice, r.StopTriggerPrice); err != nil {
		return "", nil, err
	}
	return "trigger_bracket_gtd", &OrderConfigurationLegWire{
		BaseSize:         orderDecimalParam(r.BaseSize),
		LimitPrice:       orderDecimalParam(r.LimitPrice),
		StopTriggerPrice: orderDecimalParam(r.StopTriggerPrice),
		EndTime:          orderTimeParam(r.EndTime),
	}, nil
}

// orderConfiguration validates the scaled order and its child orders and returns its body
func (r *ScaledLimitGTCRequest) orderConfiguration() (string, *OrderConfigurationLegWire, error) {
	if err := checkOrderNumbers(errInvalidAmount, r.QuoteSize, r.BaseSize); err != nil {
		return "", nil, err
	}
	if err := checkOrderNumbers(errInvalidPrice, r.MinimumPrice, r.MaximumPrice); err != nil {
		return "", nil, err
	}
	if err := checkOrderEnum(r.PriceDistribution, errInvalidScaledPriceDistribution, scaledPriceDistributions); err != nil {
		return "", nil, err
	}
	if err := checkOrderEnum(r.SizeDistribution, errInvalidScaledSizeDistribution, scaledSizeDistributions); err != nil {
		return "", nil, err
	}
	var children []OrderConfigurationLegWire
	if len(r.Orders) != 0 {
		children = make([]OrderConfigurationLegWire, len(r.Orders))
		for i := range r.Orders {
			_, child, err := r.Orders[i].orderConfiguration()
			if err != nil {
				return "", nil, fmt.Errorf("scaled order child %d: %w", i, err)
			}
			children[i] = *child
		}
	}
	return "scaled_limit_gtc", &OrderConfigurationLegWire{
		Orders:            children,
		QuoteSize:         orderDecimalParam(r.QuoteSize),
		BaseSize:          orderDecimalParam(r.BaseSize),
		NumberOfOrders:    r.NumberOfOrders,
		MinimumPrice:      orderDecimalParam(r.MinimumPrice),
		MaximumPrice:      orderDecimalParam(r.MaximumPrice),
		PriceDistribution: r.PriceDistribution,
		SizeDistribution:  r.SizeDistribution,
		SizeDifference:    orderDecimalParam(r.SizeDifference),
		SizeRatio:         orderDecimalParam(r.SizeRatio),
	}, nil
}

// listFillsParams validates a fill list request and returns its query, each list param repeated per value
func listFillsParams(req *ListFillsRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := checkWindow(req.StartSequenceTimestamp, req.EndSequenceTimestamp); err != nil {
		return nil, err
	}
	if err := checkOrderEnum(req.SortBy, errInvalidFillsSortBy, fillsSortFields); err != nil {
		return nil, err
	}
	if err := checkOrderEnums(req.OrderTypes, errInvalidOrderType, orderTypes); err != nil {
		return nil, err
	}
	if err := checkOrderEnum(req.OrderSide, errInvalidOrderSide, orderSides); err != nil {
		return nil, err
	}
	if err := checkOrderEnums(req.ProductTypes, errInvalidOrderProductType, orderProductTypes); err != nil {
		return nil, err
	}
	params := url.Values{}
	addOrderParams(params, "order_ids", req.OrderIDs)
	addOrderParams(params, "trade_ids", req.TradeIDs)
	addOrderParams(params, "product_ids", req.ProductIDs)
	if !req.StartSequenceTimestamp.IsZero() {
		params.Set("start_sequence_timestamp", formatTime(req.StartSequenceTimestamp))
	}
	if !req.EndSequenceTimestamp.IsZero() {
		params.Set("end_sequence_timestamp", formatTime(req.EndSequenceTimestamp))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.SortBy != "" {
		params.Set("sort_by", req.SortBy)
	}
	for _, c := range req.AssetFilters {
		params.Add("asset_filters", c.Upper().String())
	}
	addOrderParams(params, "order_types", req.OrderTypes)
	if req.OrderSide != "" {
		params.Set("order_side", req.OrderSide)
	}
	addOrderParams(params, "product_types", req.ProductTypes)
	if req.ProofToken != "" {
		params.Set("proof_token", req.ProofToken)
	}
	return params, nil
}

// listOrdersParams validates an order list request and returns its query, each list param repeated per value
func listOrdersParams(req *ListOrdersRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := checkWindow(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}
	if err := checkOrderEnum(req.ProductType, errInvalidOrderProductType, orderProductTypes); err != nil {
		return nil, err
	}
	if err := checkOrderEnums(req.OrderStatuses, errInvalidOrderStatus, orderStatuses); err != nil {
		return nil, err
	}
	if err := checkOrderEnums(req.TimeInForces, errInvalidOrderTimeInForce, orderTimeInForces); err != nil {
		return nil, err
	}
	if err := checkOrderEnums(req.OrderTypes, errInvalidOrderType, orderTypes); err != nil {
		return nil, err
	}
	if err := checkOrderEnum(req.OrderSide, errInvalidOrderSide, orderSides); err != nil {
		return nil, err
	}
	if err := checkOrderEnum(req.OrderPlacementSource, errInvalidOrderPlacementSource, orderPlacementSources); err != nil {
		return nil, err
	}
	if err := checkOrderEnum(req.ContractExpiryType, errInvalidOrderContractExpiryType, orderContractExpiryTypes); err != nil {
		return nil, err
	}
	if err := checkOrderEnum(req.SortBy, errInvalidOrdersSortBy, ordersSortFields); err != nil {
		return nil, err
	}
	params := url.Values{}
	addOrderParams(params, "order_ids", req.OrderIDs)
	addOrderParams(params, "product_ids", req.ProductIDs)
	if req.ProductType != "" {
		params.Set("product_type", req.ProductType)
	}
	addOrderParams(params, "order_status", req.OrderStatuses)
	addOrderParams(params, "time_in_forces", req.TimeInForces)
	addOrderParams(params, "order_types", req.OrderTypes)
	if req.OrderSide != "" {
		params.Set("order_side", req.OrderSide)
	}
	if !req.StartDate.IsZero() {
		params.Set("start_date", formatTime(req.StartDate))
	}
	if !req.EndDate.IsZero() {
		params.Set("end_date", formatTime(req.EndDate))
	}
	if req.OrderPlacementSource != "" {
		params.Set("order_placement_source", req.OrderPlacementSource)
	}
	if req.ContractExpiryType != "" {
		params.Set("contract_expiry_type", req.ContractExpiryType)
	}
	for _, c := range req.AssetFilters {
		params.Add("asset_filters", c.Upper().String())
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.SortBy != "" {
		params.Set("sort_by", req.SortBy)
	}
	if req.UseSimplifiedTotalValueCalculation != nil {
		params.Set("use_simplified_total_value_calculation", strconv.FormatBool(*req.UseSimplifiedTotalValueCalculation))
	}
	if req.ProofToken != "" {
		params.Set("proof_token", req.ProofToken)
	}
	return params, nil
}

// addOrderParams adds a list param once per value, as Coinbase takes them
func addOrderParams(params url.Values, key string, values []string) {
	for _, v := range values {
		params.Add(key, v)
	}
}

// checkOrderEnum returns the sentinel wrapped with the value when it is set and not one of valid; an unset value leaves
// the choice to Coinbase's default
func checkOrderEnum(value string, sentinel error, valid []string) error {
	if value != "" && !slices.Contains(valid, value) {
		return fmt.Errorf("%w: %q", sentinel, value)
	}
	return nil
}

// checkOrderEnums returns the sentinel wrapped with the first of a list param's values that is not one of valid; an
// empty value is refused, as it selects nothing
func checkOrderEnums(values []string, sentinel error, valid []string) error {
	for _, v := range values {
		if !slices.Contains(valid, v) {
			return fmt.Errorf("%w: %q", sentinel, v)
		}
	}
	return nil
}

// checkOrderNumbers returns the sentinel wrapped with the first negative, NaN or infinite amount or price; zero leaves
// one unset
func checkOrderNumbers(sentinel error, values ...float64) error {
	for _, v := range values {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%w: %v", sentinel, v)
		}
	}
	return nil
}

// orderDecimalParam formats an optional amount or price as the decimal string Coinbase takes; zero formats as empty,
// which omitempty leaves unsent
func orderDecimalParam(v float64) string {
	if v == 0 {
		return ""
	}
	return formatFloat(v)
}

// orderTimeParam formats an optional time as the RFC 3339 UTC timestamp Coinbase takes; the zero time formats as
// empty, which omitempty leaves unsent
func orderTimeParam(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return formatTime(t)
}

// orderDurationParam formats an optional duration in seconds suffixed with s, as Coinbase takes durations; zero
// formats as empty, which omitempty leaves unsent
func orderDurationParam(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return strconv.FormatFloat(d.Seconds(), 'f', -1, 64) + "s"
}
