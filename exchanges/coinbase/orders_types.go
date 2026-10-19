package coinbase

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Order margin types; Coinbase applies cross margin to an order that sets none
const (
	OrderMarginTypeCross    = "CROSS"
	OrderMarginTypeIsolated = "ISOLATED"
	// OrderMarginTypeUnknown is the margin type Coinbase reports for an order without margin, such as a spot order;
	// the docs do not list it
	OrderMarginTypeUnknown = "UNKNOWN_MARGIN_TYPE"
)

// Stop directions, the way the last trade price must cross a stop limit order's stop price to trigger it
const (
	StopDirectionStopUp   = "STOP_DIRECTION_STOP_UP"
	StopDirectionStopDown = "STOP_DIRECTION_STOP_DOWN"
)

// Smart order routing preferences; an unspecified preference follows the account's settings
const (
	SORPreferenceUnspecified = "SOR_PREFERENCE_UNSPECIFIED"
	SORPreferenceEnabled     = "SOR_ENABLED"
	SORPreferenceDisabled    = "SOR_DISABLED"
)

// Prediction market sides; a prediction market has only a YES book, so a NO order is a short position on it
const (
	PredictionSideUnknown = "PREDICTION_SIDE_UNKNOWN"
	PredictionSideYes     = "PREDICTION_SIDE_YES"
	PredictionSideNo      = "PREDICTION_SIDE_NO"
)

// Equity trading sessions an equity order may execute in; only the normal session, the default, takes market orders,
// and the others take only limit orders for whole shares sized in the base currency
const (
	EquityTradingSessionUnknown      = "UNKNOWN_EQUITY_TRADING_SESSION"
	EquityTradingSessionNormal       = "EQUITY_TRADING_SESSION_NORMAL"
	EquityTradingSessionAfterHours   = "EQUITY_TRADING_SESSION_AFTER_HOURS"
	EquityTradingSessionMultiSession = "EQUITY_TRADING_SESSION_MULTI_SESSION"
	EquityTradingSessionOvernight    = "EQUITY_TRADING_SESSION_OVERNIGHT"
	EquityTradingSessionPreMarket    = "EQUITY_TRADING_SESSION_PRE_MARKET"
)

// Displayed order configurations, an equity order's time in force: MARKET_GFD for a market order, which Coinbase
// assumes when a market order sets none, and LIMIT_GFD or LIMIT_GTC for a limit order. INSTANT_GFD is a legacy Simple
// Trade value and EXERCISE_GFD applies only to equity option exercises
const (
	DisplayedOrderConfigUnknown     = "UNKNOWN_DISPLAYED_ORDER_CONFIG"
	DisplayedOrderConfigInstantGFD  = "INSTANT_GFD"
	DisplayedOrderConfigLimitGFD    = "LIMIT_GFD"
	DisplayedOrderConfigLimitGTC    = "LIMIT_GTC"
	DisplayedOrderConfigMarketGFD   = "MARKET_GFD"
	DisplayedOrderConfigExerciseGFD = "EXERCISE_GFD"
)

// Cost basis methods, which match an order's fills to tax lots
const (
	CostBasisMethodUnspecified = "COST_BASIS_METHOD_UNSPECIFIED"
	CostBasisMethodHIFO        = "COST_BASIS_METHOD_HIFO"
	CostBasisMethodLIFO        = "COST_BASIS_METHOD_LIFO"
	CostBasisMethodFIFO        = "COST_BASIS_METHOD_FIFO"
	CostBasisMethodSpecificID  = "COST_BASIS_METHOD_SPEC_ID"
)

// Scaled order price distributions, how a scaled order spreads its child orders' prices between its minimum and
// maximum prices; Coinbase uses FLAT by default
const (
	ScaledPriceDistributionFlat             = "FLAT"
	ScaledPriceDistributionLinearIncreasing = "LINEAR_INCREASING"
	ScaledPriceDistributionLinearDecreasing = "LINEAR_DECREASING"
	ScaledPriceDistributionCustom           = "CUSTOM_PRICE_DISTRIBUTION"
)

// Scaled order size distributions, how a scaled order spreads its size across its child orders
const (
	ScaledSizeDistributionUnknown     = "UNKNOWN_DISTRIBUTION"
	ScaledSizeDistributionIncreasing  = "INCREASING"
	ScaledSizeDistributionDecreasing  = "DECREASING"
	ScaledSizeDistributionEvenlySplit = "EVENLY_SPLIT"
	ScaledSizeDistributionCustom      = "CUSTOM_SIZE_DISTRIBUTION"
)

// Order statuses
const (
	OrderStatusPending      = "PENDING"
	OrderStatusOpen         = "OPEN"
	OrderStatusFilled       = "FILLED"
	OrderStatusCancelled    = "CANCELLED"
	OrderStatusExpired      = "EXPIRED"
	OrderStatusFailed       = "FAILED"
	OrderStatusUnknown      = "UNKNOWN_ORDER_STATUS"
	OrderStatusQueued       = "QUEUED"
	OrderStatusCancelQueued = "CANCEL_QUEUED"
	OrderStatusEditQueued   = "EDIT_QUEUED"
)

// Order times in force
const (
	TimeInForceUnknown            = "UNKNOWN_TIME_IN_FORCE"
	TimeInForceGoodUntilDateTime  = "GOOD_UNTIL_DATE_TIME"
	TimeInForceGoodUntilCancelled = "GOOD_UNTIL_CANCELLED"
	TimeInForceImmediateOrCancel  = "IMMEDIATE_OR_CANCEL"
	TimeInForceFillOrKill         = "FILL_OR_KILL"
)

// Order types
const (
	OrderTypeUnknown   = "UNKNOWN_ORDER_TYPE"
	OrderTypeMarket    = "MARKET"
	OrderTypeLimit     = "LIMIT"
	OrderTypeStop      = "STOP"
	OrderTypeStopLimit = "STOP_LIMIT"
	// OrderTypeBracket is a limit order with an embedded stop trigger price that exits the position
	OrderTypeBracket = "BRACKET"
	// OrderTypeTWAP splits an order into smaller ones executed over a duration
	OrderTypeTWAP = "TWAP"
	// OrderTypeRollOpen and OrderTypeRollClose are the open and close steps of a futures contract roll
	OrderTypeRollOpen    = "ROLL_OPEN"
	OrderTypeRollClose   = "ROLL_CLOSE"
	OrderTypeLiquidation = "LIQUIDATION"
	// OrderTypeScaled splits an order into child limit orders at increasing or decreasing prices
	OrderTypeScaled = "SCALED"
	// OrderTypeTakeProfitStopLoss is the type Coinbase reports for an attached take profit/stop loss order; the docs
	// do not list it, so list filters refuse it
	OrderTypeTakeProfitStopLoss = "TAKE_PROFIT_STOP_LOSS"
)

// Order trigger statuses, which say whether a stop order's stop price has been reached
const (
	TriggerStatusUnknown          = "UNKNOWN_TRIGGER_STATUS"
	TriggerStatusInvalidOrderType = "INVALID_ORDER_TYPE"
	TriggerStatusStopPending      = "STOP_PENDING"
	TriggerStatusStopTriggered    = "STOP_TRIGGERED"
)

// Order reject reasons
const (
	RejectReasonUnspecified       = "REJECT_REASON_UNSPECIFIED"
	RejectReasonHoldFailure       = "HOLD_FAILURE"
	RejectReasonTooManyOpenOrders = "TOO_MANY_OPEN_ORDERS"
	RejectReasonInsufficientFunds = "REJECT_REASON_INSUFFICIENT_FUNDS"
	RejectReasonRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)

// Order placement sources, where an order was placed from
const (
	OrderPlacementSourceUnknown        = "UNKNOWN_PLACEMENT_SOURCE"
	OrderPlacementSourceRetailSimple   = "RETAIL_SIMPLE"
	OrderPlacementSourceRetailAdvanced = "RETAIL_ADVANCED"
	OrderPlacementSourceRetailAdmin    = "RETAIL_ADMIN"
	OrderPlacementSourceRetailRaise    = "RETAIL_RAISE"
)

// Order list sort fields; Coinbase sorts by creation time by default, and pages through any other sort unstably
const (
	OrdersSortByUnknown        = "UNKNOWN_SORT_BY"
	OrdersSortByLimitPrice     = "LIMIT_PRICE"
	OrdersSortByLastFillTime   = "LAST_FILL_TIME"
	OrdersSortByLastUpdateTime = "LAST_UPDATE_TIME"
)

// Fill list sort fields; Coinbase sorts by creation time by default, and pages through any other sort unstably
const (
	FillsSortByUnknown   = "UNKNOWN_SORT_BY"
	FillsSortByPrice     = "PRICE"
	FillsSortByTradeTime = "TRADE_TIME"
)

// Fill liquidity indicators, whether a fill made or took liquidity
const (
	LiquidityIndicatorUnknown = "UNKNOWN_LIQUIDITY_INDICATOR"
	LiquidityIndicatorMaker   = "MAKER"
	LiquidityIndicatorTaker   = "TAKER"
)

// Fill sources, the venue mechanism a fill came from: the central limit order book or a request for quote
const (
	FillSourceUnknown = "FILL_SOURCE_UNKNOWN"
	FillSourceCLOB    = "FILL_SOURCE_CLOB"
	FillSourceRFQ     = "FILL_SOURCE_RFQ"
)

// Order data sources, the backend system that served an order or fill
const (
	OrderDataSourceUnknown               = "ORDER_DATA_SOURCE_UNKNOWN"
	OrderDataSourceTradingProxy          = "ORDER_DATA_SOURCE_TRADING_PROXY"
	OrderDataSourceAccountHistoryService = "ORDER_DATA_SOURCE_ACCOUNT_HISTORY_SERVICE"
)

// Fill trade types: FILL for a regular fill, and REVERSAL, CORRECTION or SYNTHETIC for an adjusted one
const (
	FillTradeTypeFill       = "FILL"
	FillTradeTypeReversal   = "REVERSAL"
	FillTradeTypeCorrection = "CORRECTION"
	FillTradeTypeSynthetic  = "SYNTHETIC"
)

// Cancel failure reasons, why Coinbase did not submit an order's cancel request
const (
	// CancelFailureReasonUnknown is the reason a submitted cancel request reports
	CancelFailureReasonUnknown        = "UNKNOWN_CANCEL_FAILURE_REASON"
	CancelFailureReasonInvalidRequest = "INVALID_CANCEL_REQUEST"
	// CancelFailureReasonUnknownOrder is the reason Coinbase reports for an order it does not know
	CancelFailureReasonUnknownOrder             = "UNKNOWN_CANCEL_ORDER"
	CancelFailureReasonCommanderRejected        = "COMMANDER_REJECTED_CANCEL_ORDER"
	CancelFailureReasonDuplicateRequest         = "DUPLICATE_CANCEL_REQUEST"
	CancelFailureReasonInvalidProductID         = "INVALID_CANCEL_PRODUCT_ID"
	CancelFailureReasonInvalidFCMTradingSession = "INVALID_CANCEL_FCM_TRADING_SESSION"
	CancelFailureReasonNotAllowed               = "NOT_ALLOWED_TO_CANCEL"
	CancelFailureReasonOrderFullyFilled         = "ORDER_IS_FULLY_FILLED"
	CancelFailureReasonOrderBeingReplaced       = "ORDER_IS_BEING_REPLACED"
)

// Order preview warnings
const (
	PreviewWarningUnknown                       = "UNKNOWN"
	PreviewWarningBigOrder                      = "BIG_ORDER"
	PreviewWarningSmallOrder                    = "SMALL_ORDER"
	PreviewWarningDurationExtendedByMarketClose = "DURATION_EXTENDED_BY_MARKET_CLOSE"
	// PreviewWarningOpenOrdersExceedCompliancePurchaseLimit warns that open orders exceed the account's compliance
	// purchase limit, so Coinbase may cancel some
	PreviewWarningOpenOrdersExceedCompliancePurchaseLimit = "OPEN_ORDERS_EXCEED_COMPLIANCE_PURCHASE_LIMIT_MAY_CANCEL"
)

const (
	// maximumClientOrderIDLength is the most characters Coinbase accepts in a new order's client order ID
	maximumClientOrderIDLength = 128

	// The failure reasons Coinbase reports when it has none to give, which a failure's error leaves out
	newOrderFailureReasonUnknown  = "UNKNOWN_FAILURE_REASON"
	previewFailureReasonUnknown   = "UNKNOWN_PREVIEW_FAILURE_REASON"
	editOrderFailureReasonUnknown = "UNKNOWN_EDIT_ORDER_FAILURE_REASON"
)

// editOrderNotFoundReasons are the edit failure reasons Coinbase reports for an order it does not know
var editOrderNotFoundReasons = []string{"ORDER_NOT_FOUND", "FUTURES_VENUE_ORDER_NOT_FOUND"}

// cancelOrdersBody is the body Cancel Orders sends
type cancelOrdersBody struct {
	OrderIDs []string `json:"order_ids"`
}

// CancelOrdersResponse holds the result of each order's cancel request
type CancelOrdersResponse struct {
	Results []CancelOrderResult `json:"results"`
}

// Err joins the errors of the cancel requests Coinbase did not submit, or returns nil when it submitted them all
func (r *CancelOrdersResponse) Err() error {
	var errs []error
	for i := range r.Results {
		if err := r.Results[i].Err(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// CancelOrderResult is the result of an order's cancel request; a submitted request completes asynchronously, so the
// order's status reports whether it was cancelled
type CancelOrderResult struct {
	Success bool `json:"success"`
	// FailureReason is UNKNOWN_CANCEL_FAILURE_REASON for a submitted request
	FailureReason string `json:"failure_reason"`
	OrderID       string `json:"order_id"`
}

// Err returns nil for a submitted cancel request, order.ErrOrderNotFound for an order Coinbase does not know and
// errCancelOrderFailed with Coinbase's reason for any other failure
func (r *CancelOrderResult) Err() error {
	switch {
	case r.Success:
		return nil
	case r.FailureReason == CancelFailureReasonUnknownOrder:
		return fmt.Errorf("%w: order %q: %s", order.ErrOrderNotFound, r.OrderID, r.FailureReason)
	default:
		return fmt.Errorf("%w: order %q: %s", errCancelOrderFailed, r.OrderID, r.FailureReason)
	}
}

// ClosePositionRequest is a position ClosePosition closes
type ClosePositionRequest struct {
	// ClientOrderID identifies the closing order
	ClientOrderID string
	// ProductID is a US derivative's product ID, such as BIT-28JUL23-CDE
	ProductID string
	// Size is how many contracts to close
	Size float64
}

// closePositionBody is the body Close Position sends
type closePositionBody struct {
	ClientOrderID string `json:"client_order_id"`
	ProductID     string `json:"product_id"`
	Size          string `json:"size,omitempty"`
}

// CreateOrderRequest is an order CreateOrder places
type CreateOrderRequest struct {
	// ClientOrderID identifies the order, in at most 128 characters; an ID already used returns the order it
	// identifies instead of creating another
	ClientOrderID string
	// ProductID is the product's canonical ID, an equity's rather than its display ticker
	ProductID string
	// Side is BUY or SELL
	Side               string
	OrderConfiguration OrderConfigurationRequest
	// Leverage is the order's leverage; Coinbase uses 1 when it is unset
	Leverage float64
	// MarginType is CROSS, the default, or ISOLATED
	MarginType string
	// PreviewID links the order to the preview PreviewOrder returned for it
	PreviewID string
	// AttachedOrderConfiguration attaches a take profit/stop loss order to the order: a *TriggerBracketGTCRequest
	// without a size, as the attached order takes the order's own. Equities take no attached orders
	AttachedOrderConfiguration OrderConfigurationRequest
	// SORPreference enables or disables smart order routing for the order
	SORPreference       string
	PredictionMetadata  *PredictionMetadataRequest
	EquityOrderMetadata *EquityOrderMetadata
	// CostBasisMethod chooses how the order's fills match tax lots
	CostBasisMethod string
}

// orderBody is the body Create Order and Preview Order send; Preview Order takes no client order ID, preview ID or
// smart order routing preference
type orderBody struct {
	ClientOrderID              string                  `json:"client_order_id,omitempty"`
	ProductID                  string                  `json:"product_id"`
	Side                       string                  `json:"side"`
	OrderConfiguration         orderConfigurationBody  `json:"order_configuration"`
	Leverage                   string                  `json:"leverage,omitempty"`
	MarginType                 string                  `json:"margin_type,omitempty"`
	PreviewID                  string                  `json:"preview_id,omitempty"`
	AttachedOrderConfiguration orderConfigurationBody  `json:"attached_order_configuration,omitempty"`
	SORPreference              string                  `json:"sor_preference,omitempty"`
	PredictionMetadata         *predictionMetadataBody `json:"prediction_metadata,omitempty"`
	EquityOrderMetadata        *EquityOrderMetadata    `json:"equity_order_metadata,omitempty"`
	CostBasisMethod            string                  `json:"cost_basis_method,omitempty"`
}

// OrderConfigurationRequest is the configuration of an order to place, preview or attach to another: one of
// *MarketMarketIOCRequest, *MarketMarketFOKRequest, *SORLimitIOCRequest, *LimitLimitGTCRequest,
// *LimitLimitGTDRequest, *LimitLimitFOKRequest, *TWAPLimitGTDRequest, *StopLimitStopLimitGTCRequest,
// *StopLimitStopLimitGTDRequest, *TriggerBracketGTCRequest, *TriggerBracketGTDRequest or *ScaledLimitGTCRequest.
// Sizes and prices left zero are not sent
type OrderConfigurationRequest interface {
	// orderConfiguration validates the configuration and returns the key and body Coinbase takes it under
	orderConfiguration() (string, *orderConfigurationLegBody, error)
}

// MarketMarketIOCRequest is a market order, which fills what it can at once at the best prices and cancels the rest;
// a buy is sized in the quote or base currency, a sell in the base currency
type MarketMarketIOCRequest struct {
	QuoteSize float64
	BaseSize  float64
}

// MarketMarketFOKRequest is a market order that fills completely at once or not at all; Coinbase documents it for
// perpetuals only
type MarketMarketFOKRequest struct {
	QuoteSize float64
	BaseSize  float64
}

// SORLimitIOCRequest is a limit order, routed by smart order routing, that fills what it can at once at its limit
// price or better and cancels the rest
type SORLimitIOCRequest struct {
	QuoteSize  float64
	BaseSize   float64
	LimitPrice float64
}

// LimitLimitGTCRequest is a limit order that rests on the book until it fills or is cancelled; an equity's sets its
// time in force, LIMIT_GFD or LIMIT_GTC, in the order's equity metadata
type LimitLimitGTCRequest struct {
	QuoteSize  float64
	BaseSize   float64
	LimitPrice float64
	// PostOnly refuses the order rather than let it take liquidity
	PostOnly bool
}

// LimitLimitGTDRequest is a limit order that rests on the book until it fills, is cancelled or its end time passes
type LimitLimitGTDRequest struct {
	QuoteSize  float64
	BaseSize   float64
	LimitPrice float64
	EndTime    time.Time
	// PostOnly refuses the order rather than let it take liquidity
	PostOnly bool
}

// LimitLimitFOKRequest is a limit order that fills completely at once at its limit price or better, or not at all
type LimitLimitFOKRequest struct {
	QuoteSize  float64
	BaseSize   float64
	LimitPrice float64
}

// TWAPLimitGTDRequest is a time weighted average price order, which executes in suborders, its buckets, between its
// start and end times, at its limit price or better
type TWAPLimitGTDRequest struct {
	QuoteSize  float64
	BaseSize   float64
	StartTime  time.Time
	EndTime    time.Time
	LimitPrice float64
	// NumberOfBuckets is how many suborders the order splits into, each BucketSize; together they make the order's
	// size
	NumberOfBuckets uint64
	BucketSize      float64
	// BucketDuration is how long each suborder executes over, sent in whole and fractional seconds
	BucketDuration time.Duration
}

// StopLimitStopLimitGTCRequest is a stop limit order: a limit order Coinbase places when the last trade price crosses
// its stop price in its stop direction, which then rests until it fills or is cancelled
type StopLimitStopLimitGTCRequest struct {
	BaseSize   float64
	LimitPrice float64
	StopPrice  float64
	// StopDirection is STOP_DIRECTION_STOP_UP, triggering when the last trade price rises above the stop price, or
	// STOP_DIRECTION_STOP_DOWN, when it falls below it
	StopDirection string
}

// StopLimitStopLimitGTDRequest is a stop limit order whose limit order rests until it fills, is cancelled or its end
// time passes
type StopLimitStopLimitGTDRequest struct {
	BaseSize   float64
	LimitPrice float64
	StopPrice  float64
	EndTime    time.Time
	// StopDirection is STOP_DIRECTION_STOP_UP, triggering when the last trade price rises above the stop price, or
	// STOP_DIRECTION_STOP_DOWN, when it falls below it
	StopDirection string
}

// TriggerBracketGTCRequest is a bracket order: a limit order with an embedded stop, which places a stop limit order
// priced 5% beyond StopTriggerPrice, higher for a buy and lower for a sell, once the price reaches it. As an attached
// take profit/stop loss order it takes no size, as it takes its parent's
type TriggerBracketGTCRequest struct {
	BaseSize   float64
	LimitPrice float64
	// StopTriggerPrice is the quote price the position exits at
	StopTriggerPrice float64
}

// TriggerBracketGTDRequest is a bracket order that rests until it fills, is cancelled or its end time passes
type TriggerBracketGTDRequest struct {
	BaseSize   float64
	LimitPrice float64
	// StopTriggerPrice is the quote price the position exits at
	StopTriggerPrice float64
	EndTime          time.Time
}

// ScaledLimitGTCRequest is a scaled order, which splits itself into child limit orders across a price range. Orders
// sets each child's size and price, as a custom distribution takes them; otherwise Coinbase infers the children's
// prices from the minimum and maximum prices and the price distribution
type ScaledLimitGTCRequest struct {
	Orders            []LimitLimitGTCRequest
	QuoteSize         float64
	BaseSize          float64
	NumberOfOrders    uint64
	MinimumPrice      float64
	MaximumPrice      float64
	PriceDistribution string
	SizeDistribution  string
	SizeDifference    float64
	SizeRatio         float64
}

// orderConfigurationBody is an order configuration as Coinbase takes it: one configuration under its key
type orderConfigurationBody map[string]*orderConfigurationLegBody

// orderConfigurationLegBody is a configuration of an order configuration body; each configuration sets the fields its
// type takes, and omitempty leaves the rest out
type orderConfigurationLegBody struct {
	QuoteSize         string                      `json:"quote_size,omitempty"`
	BaseSize          string                      `json:"base_size,omitempty"`
	LimitPrice        string                      `json:"limit_price,omitempty"`
	StopPrice         string                      `json:"stop_price,omitempty"`
	StopTriggerPrice  string                      `json:"stop_trigger_price,omitempty"`
	StopDirection     string                      `json:"stop_direction,omitempty"`
	StartTime         string                      `json:"start_time,omitempty"`
	EndTime           string                      `json:"end_time,omitempty"`
	PostOnly          bool                        `json:"post_only,omitempty"`
	NumberOfBuckets   string                      `json:"number_buckets,omitempty"`
	BucketSize        string                      `json:"bucket_size,omitempty"`
	BucketDuration    string                      `json:"bucket_duration,omitempty"`
	Orders            []orderConfigurationLegBody `json:"orders,omitempty"`
	NumberOfOrders    uint64                      `json:"num_orders,omitempty"`
	MinimumPrice      string                      `json:"min_price,omitempty"`
	MaximumPrice      string                      `json:"max_price,omitempty"`
	PriceDistribution string                      `json:"price_distribution,omitempty"`
	SizeDistribution  string                      `json:"size_distribution,omitempty"`
	SizeDifference    string                      `json:"size_diff,omitempty"`
	SizeRatio         string                      `json:"size_ratio,omitempty"`
}

// PredictionMetadataRequest describes a prediction market order
type PredictionMetadataRequest struct {
	// PredictionSide is PREDICTION_SIDE_YES or PREDICTION_SIDE_NO
	PredictionSide                          string
	PreviewOrderEstimatedAverageFilledPrice float64
	SupportsFractionalBaseSize              bool
}

// predictionMetadataBody is the prediction metadata an order request sends
type predictionMetadataBody struct {
	PredictionSide                          string `json:"prediction_side,omitempty"`
	PreviewOrderEstimatedAverageFilledPrice string `json:"preview_order_est_average_filled_price,omitempty"`
	SupportsFractionalBaseSize              bool   `json:"supports_fractional_base_size,omitempty"`
}

// EquityOrderMetadata is an equity order's trading session and time in force; Coinbase executes an order in the
// normal session by default
type EquityOrderMetadata struct {
	EquityTradingSession string `json:"equity_trading_session,omitempty"`
	// DisplayedOrderConfig is the order's time in force: MARKET_GFD with market_market_ioc, LIMIT_GFD or LIMIT_GTC
	// with limit_limit_gtc
	DisplayedOrderConfig string `json:"displayed_order_config,omitempty"`
}

// CreateOrderResponse is Coinbase's answer to an order placement: the order it created, or why it created none, and
// the order's configuration
type CreateOrderResponse struct {
	Success            bool                `json:"success"`
	SuccessResponse    *CreateOrderSuccess `json:"success_response"`
	ErrorResponse      *CreateOrderFailure `json:"error_response"`
	OrderConfiguration *OrderConfiguration `json:"order_configuration"`
}

// Err returns errOrderNotCreated with Coinbase's reasons when no order was created
func (r *CreateOrderResponse) Err() error {
	if r.Success {
		return nil
	}
	if r.ErrorResponse == nil {
		return fmt.Errorf("%w: no reason given", errOrderNotCreated)
	}
	f := r.ErrorResponse
	reason := f.NewOrderFailureReason
	if reason == "" || reason == newOrderFailureReasonUnknown {
		reason = f.FailureReason
	}
	if reason == "" || reason == newOrderFailureReasonUnknown {
		reason = f.PreviewFailureReason
	}
	var parts []string
	if reason != "" && reason != newOrderFailureReasonUnknown && reason != previewFailureReasonUnknown {
		parts = append(parts, reason)
	}
	if f.Message != "" {
		parts = append(parts, f.Message)
	}
	if f.ErrorDetails != "" && f.ErrorDetails != f.Message {
		parts = append(parts, f.ErrorDetails)
	}
	if len(parts) == 0 {
		return fmt.Errorf("%w: no reason given", errOrderNotCreated)
	}
	return fmt.Errorf("%w: %s", errOrderNotCreated, strings.Join(parts, ": "))
}

// CreateOrderSuccess identifies an order Coinbase created
type CreateOrderSuccess struct {
	OrderID       string `json:"order_id"`
	ProductID     string `json:"product_id"`
	Side          string `json:"side"`
	ClientOrderID string `json:"client_order_id"`
	// AttachedOrderID is undocumented; Coinbase sends it, empty for an order without an attached order
	AttachedOrderID string `json:"attached_order_id"`
}

// CreateOrderFailure is why Coinbase created no order
type CreateOrderFailure struct {
	// FailureReason is deprecated in favour of NewOrderFailureReason, which Coinbase may leave out
	FailureReason string `json:"error"`
	// Message describes the failure generally, and ErrorDetails specifically
	Message      string `json:"message"`
	ErrorDetails string `json:"error_details"`
	// PreviewFailureReason is deprecated in favour of NewOrderFailureReason
	PreviewFailureReason  string `json:"preview_failure_reason"`
	NewOrderFailureReason string `json:"new_order_failure_reason"`
}

// OrderConfiguration is an order's configuration as Coinbase reports it, one of its configurations set
type OrderConfiguration struct {
	MarketMarketIOC       *MarketMarketIOC       `json:"market_market_ioc"`
	MarketMarketFOK       *MarketMarketFOK       `json:"market_market_fok"`
	SORLimitIOC           *SORLimitIOC           `json:"sor_limit_ioc"`
	LimitLimitGTC         *LimitLimitGTC         `json:"limit_limit_gtc"`
	LimitLimitGTD         *LimitLimitGTD         `json:"limit_limit_gtd"`
	LimitLimitFOK         *LimitLimitFOK         `json:"limit_limit_fok"`
	TWAPLimitGTD          *TWAPLimitGTD          `json:"twap_limit_gtd"`
	StopLimitStopLimitGTC *StopLimitStopLimitGTC `json:"stop_limit_stop_limit_gtc"`
	StopLimitStopLimitGTD *StopLimitStopLimitGTD `json:"stop_limit_stop_limit_gtd"`
	TriggerBracketGTC     *TriggerBracketGTC     `json:"trigger_bracket_gtc"`
	TriggerBracketGTD     *TriggerBracketGTD     `json:"trigger_bracket_gtd"`
	ScaledLimitGTC        *ScaledLimitGTC        `json:"scaled_limit_gtc"`
}

// MarketMarketIOC is a market order's configuration
type MarketMarketIOC struct {
	QuoteSize types.Number `json:"quote_size"`
	BaseSize  types.Number `json:"base_size"`
	// CurrencySize is the quote size in the user's native currency, unset when the order is not sized in the quote
	// currency or no rate is available
	CurrencySize *Amount `json:"currency_size"`
	// RFQEnabled, RFQDisabled, ReduceOnly and OCOReference are undocumented; Coinbase sends them
	RFQEnabled   bool   `json:"rfq_enabled"`
	RFQDisabled  bool   `json:"rfq_disabled"`
	ReduceOnly   bool   `json:"reduce_only"`
	OCOReference string `json:"oco_ref"`
}

// MarketMarketFOK is a fill or kill market order's configuration
type MarketMarketFOK struct {
	QuoteSize types.Number `json:"quote_size"`
	BaseSize  types.Number `json:"base_size"`
}

// SORLimitIOC is a smart order routed immediate or cancel limit order's configuration
type SORLimitIOC struct {
	QuoteSize  types.Number `json:"quote_size"`
	BaseSize   types.Number `json:"base_size"`
	LimitPrice types.Number `json:"limit_price"`
	// RFQDisabled is undocumented; Coinbase has sent it
	RFQDisabled bool `json:"rfq_disabled"`
}

// LimitLimitGTC is a good til cancelled limit order's configuration, and a scaled order's child's
type LimitLimitGTC struct {
	QuoteSize  types.Number `json:"quote_size"`
	BaseSize   types.Number `json:"base_size"`
	LimitPrice types.Number `json:"limit_price"`
	PostOnly   bool         `json:"post_only"`
	// CurrencySize is the quote size in the user's native currency, unset when the order is not sized in the quote
	// currency or no rate is available
	CurrencySize *Amount `json:"currency_size"`
	// RFQDisabled, ReduceOnly and OCOReference are undocumented; Coinbase sends them
	RFQDisabled  bool   `json:"rfq_disabled"`
	ReduceOnly   bool   `json:"reduce_only"`
	OCOReference string `json:"oco_ref"`
}

// LimitLimitGTD is a good til date limit order's configuration
type LimitLimitGTD struct {
	QuoteSize  types.Number `json:"quote_size"`
	BaseSize   types.Number `json:"base_size"`
	LimitPrice types.Number `json:"limit_price"`
	EndTime    time.Time    `json:"end_time"`
	PostOnly   bool         `json:"post_only"`
	// CurrencySize is the quote size in the user's native currency, unset when the order is not sized in the quote
	// currency or no rate is available
	CurrencySize *Amount `json:"currency_size"`
	// RFQDisabled and ReduceOnly are undocumented; Coinbase has sent them
	RFQDisabled bool `json:"rfq_disabled"`
	ReduceOnly  bool `json:"reduce_only"`
}

// LimitLimitFOK is a fill or kill limit order's configuration
type LimitLimitFOK struct {
	QuoteSize  types.Number `json:"quote_size"`
	BaseSize   types.Number `json:"base_size"`
	LimitPrice types.Number `json:"limit_price"`
	// RFQDisabled is undocumented; Coinbase has sent it
	RFQDisabled bool `json:"rfq_disabled"`
}

// TWAPLimitGTD is a time weighted average price order's configuration
type TWAPLimitGTD struct {
	QuoteSize       types.Number `json:"quote_size"`
	BaseSize        types.Number `json:"base_size"`
	StartTime       time.Time    `json:"start_time"`
	EndTime         time.Time    `json:"end_time"`
	LimitPrice      types.Number `json:"limit_price"`
	NumberOfBuckets types.Number `json:"number_buckets"`
	BucketSize      types.Number `json:"bucket_size"`
	BucketDuration  Duration     `json:"bucket_duration"`
}

// StopLimitStopLimitGTC is a good til cancelled stop limit order's configuration
type StopLimitStopLimitGTC struct {
	BaseSize      types.Number `json:"base_size"`
	LimitPrice    types.Number `json:"limit_price"`
	StopPrice     types.Number `json:"stop_price"`
	StopDirection string       `json:"stop_direction"`
}

// StopLimitStopLimitGTD is a good til date stop limit order's configuration
type StopLimitStopLimitGTD struct {
	BaseSize      types.Number `json:"base_size"`
	LimitPrice    types.Number `json:"limit_price"`
	StopPrice     types.Number `json:"stop_price"`
	EndTime       time.Time    `json:"end_time"`
	StopDirection string       `json:"stop_direction"`
}

// TriggerBracketGTC is a good til cancelled bracket order's configuration, and an attached take profit/stop loss
// order's, which carries no size of its own
type TriggerBracketGTC struct {
	BaseSize         types.Number `json:"base_size"`
	LimitPrice       types.Number `json:"limit_price"`
	StopTriggerPrice types.Number `json:"stop_trigger_price"`
	// TakeProfitPrice, StopLossPrice, ReduceOnly, OCOReference and TrailingOffset are undocumented; Coinbase sends
	// them
	TakeProfitPrice types.Number `json:"take_profit_price"`
	StopLossPrice   types.Number `json:"stop_loss_price"`
	ReduceOnly      bool         `json:"reduce_only"`
	OCOReference    string       `json:"oco_ref"`
	TrailingOffset  types.Number `json:"trailing_offset"`
}

// TriggerBracketGTD is a good til date bracket order's configuration
type TriggerBracketGTD struct {
	BaseSize         types.Number `json:"base_size"`
	LimitPrice       types.Number `json:"limit_price"`
	StopTriggerPrice types.Number `json:"stop_trigger_price"`
	EndTime          time.Time    `json:"end_time"`
}

// ScaledLimitGTC is a scaled order's configuration
type ScaledLimitGTC struct {
	Orders            []LimitLimitGTC `json:"orders"`
	QuoteSize         types.Number    `json:"quote_size"`
	BaseSize          types.Number    `json:"base_size"`
	NumberOfOrders    uint64          `json:"num_orders"`
	MinimumPrice      types.Number    `json:"min_price"`
	MaximumPrice      types.Number    `json:"max_price"`
	PriceDistribution string          `json:"price_distribution"`
	SizeDistribution  string          `json:"size_distribution"`
	SizeDifference    types.Number    `json:"size_diff"`
	SizeRatio         types.Number    `json:"size_ratio"`
}

// EditOrderRequest is an edit EditOrder makes to an open order, or EditOrderPreview previews
type EditOrderRequest struct {
	OrderID string
	// Price and Size are the order's new price and size, both required, so an unchanged one is sent as it stands
	Price float64
	Size  float64
	// AttachedOrderConfiguration changes the order's attached take profit/stop loss order: a
	// *TriggerBracketGTCRequest, *LimitLimitGTCRequest or *StopLimitStopLimitGTCRequest
	AttachedOrderConfiguration OrderConfigurationRequest
	// CancelAttachedOrder drops both legs of the order's attached take profit/stop loss order, leaving a plain limit
	// order
	CancelAttachedOrder bool
	// StopPrice is the new stop price of a take profit/stop loss or stop loss order
	StopPrice float64
	// AverageEntryPrice is the position's average entry price, which the preview's estimated profit and loss uses
	AverageEntryPrice float64
}

// editOrderBody is the body Edit Order and Edit Order Preview send
type editOrderBody struct {
	OrderID                    string                 `json:"order_id"`
	Price                      string                 `json:"price"`
	Size                       string                 `json:"size"`
	AttachedOrderConfiguration orderConfigurationBody `json:"attached_order_configuration,omitempty"`
	CancelAttachedOrder        bool                   `json:"cancel_attached_order,omitempty"`
	StopPrice                  string                 `json:"stop_price,omitempty"`
	AverageEntryPrice          string                 `json:"average_entry_price,omitempty"`
}

// EditOrderResponse is Coinbase's answer to an order edit: whether it placed the edit, and why not
type EditOrderResponse struct {
	Success bool             `json:"success"`
	Errors  []EditOrderError `json:"errors"`
}

// Err returns errOrderNotEdited with Coinbase's reasons when the edit was not placed, also wrapping
// order.ErrOrderNotFound for an order Coinbase does not know
func (r *EditOrderResponse) Err() error {
	if r.Success {
		return nil
	}
	return editOrderFailure(errOrderNotEdited, r.Errors)
}

// EditOrderError is a reason an order edit failed, or would fail
type EditOrderError struct {
	EditFailureReason    string `json:"edit_failure_reason"`
	PreviewFailureReason string `json:"preview_failure_reason"`
}

// EditOrderPreviewResponse is an order edit's preview: the order's totals as edited, and why the edit would fail
type EditOrderPreviewResponse struct {
	Errors             []EditOrderError `json:"errors"`
	Slippage           types.Number     `json:"slippage"`
	OrderTotal         types.Number     `json:"order_total"`
	CommissionTotal    types.Number     `json:"commission_total"`
	QuoteSize          types.Number     `json:"quote_size"`
	BaseSize           types.Number     `json:"base_size"`
	BestBid            types.Number     `json:"best_bid"`
	BestAsk            types.Number     `json:"best_ask"`
	AverageFilledPrice types.Number     `json:"average_filled_price"`
	OrderMarginTotal   types.Number     `json:"order_margin_total"`
	// CommissionDetailTotal breaks the commission down
	CommissionDetailTotal *CommissionDetailTotal `json:"commission_detail_total"`
	// PNLConfiguration estimates the profit or loss of an order with an attached take profit/stop loss, before fees
	// and slippage
	PNLConfiguration *PNLConfiguration `json:"pnl_configuration"`
}

// Err returns errOrderEditPreviewFailed with Coinbase's reasons when the edit would fail, also wrapping
// order.ErrOrderNotFound for an order Coinbase does not know
func (r *EditOrderPreviewResponse) Err() error {
	if len(r.Errors) == 0 {
		return nil
	}
	return editOrderFailure(errOrderEditPreviewFailed, r.Errors)
}

// CommissionDetailTotal breaks an order's or fill's commission down
type CommissionDetailTotal struct {
	// TotalCommission is the sum of the commissions below
	TotalCommission types.Number `json:"total_commission"`
	// GSTCommission is the goods and services tax part of the commission
	GSTCommission         types.Number `json:"gst_commission"`
	WithholdingCommission types.Number `json:"withholding_commission"`
	ClientCommission      types.Number `json:"client_commission"`
	VenueCommission       types.Number `json:"venue_commission"`
	RegulatoryCommission  types.Number `json:"regulatory_commission"`
	ClearingCommission    types.Number `json:"clearing_commission"`
}

// PNLConfiguration is an order's estimated profit or loss, before fees and slippage
type PNLConfiguration struct {
	TriggerBracketPNL *TriggerBracketPNL `json:"trigger_bracket_pnl"`
}

// TriggerBracketPNL is a bracket order's estimated profit or loss at each of its prices
type TriggerBracketPNL struct {
	// TakeProfitPNL and StopLossPNL are the profit or loss should an attached order fill at its take profit or stop
	// loss price
	TakeProfitPNL types.Number `json:"take_profit_pnl"`
	StopLossPNL   types.Number `json:"stop_loss_pnl"`
}

// getOrderResponse holds the order Get Order returns
type getOrderResponse struct {
	Order *Order `json:"order"`
}

// Order is an order as Coinbase reports it
type Order struct {
	OrderID            string              `json:"order_id"`
	ProductID          string              `json:"product_id"`
	UserID             string              `json:"user_id"`
	OrderConfiguration *OrderConfiguration `json:"order_configuration"`
	Side               string              `json:"side"`
	ClientOrderID      string              `json:"client_order_id"`
	Status             string              `json:"status"`
	TimeInForce        string              `json:"time_in_force"`
	CreatedTime        time.Time           `json:"created_time"`
	// CompletionPercentage is how much of the order has filled, in percent
	CompletionPercentage types.Number `json:"completion_percentage"`
	// FilledSize is in the base currency, and FilledValue in the quote currency
	FilledSize         types.Number `json:"filled_size"`
	AverageFilledPrice types.Number `json:"average_filled_price"`
	// Fee is deprecated in favour of TotalFees
	Fee           types.Number `json:"fee"`
	NumberOfFills types.Number `json:"number_of_fills"`
	FilledValue   types.Number `json:"filled_value"`
	// PendingCancel is set while a cancel request for the order is in progress
	PendingCancel bool `json:"pending_cancel"`
	// SizeInQuote is set for an order sized in the quote currency
	SizeInQuote         bool         `json:"size_in_quote"`
	TotalFees           types.Number `json:"total_fees"`
	SizeInclusiveOfFees bool         `json:"size_inclusive_of_fees"`
	// TotalValueAfterFees is FilledValue plus TotalFees for a buy and minus them for a sell
	TotalValueAfterFees types.Number `json:"total_value_after_fees"`
	TriggerStatus       string       `json:"trigger_status"`
	OrderType           string       `json:"order_type"`
	RejectReason        string       `json:"reject_reason"`
	// Settled is set once the order has filled completely
	Settled              bool   `json:"settled"`
	ProductType          string `json:"product_type"`
	RejectMessage        string `json:"reject_message"`
	CancelMessage        string `json:"cancel_message"`
	OrderPlacementSource string `json:"order_placement_source"`
	// OutstandingHoldAmount is the funds the order still holds, zero once its hold is released
	OutstandingHoldAmount types.Number `json:"outstanding_hold_amount"`
	IsLiquidation         bool         `json:"is_liquidation"`
	LastFillTime          time.Time    `json:"last_fill_time"`
	// EditHistory holds the order's latest five edits
	EditHistory []OrderEdit  `json:"edit_history"`
	Leverage    types.Number `json:"leverage"`
	MarginType  string       `json:"margin_type"`
	// RetailPortfolioID is the portfolio the order belongs to
	RetailPortfolioID string `json:"retail_portfolio_id"`
	// OriginatingOrderID is an attached order's parent, and AttachedOrderID a parent's attached order
	OriginatingOrderID         string              `json:"originating_order_id"`
	AttachedOrderID            string              `json:"attached_order_id"`
	AttachedOrderConfiguration *OrderConfiguration `json:"attached_order_configuration"`
	// CurrentPendingReplace is the price and size of an edit in progress
	CurrentPendingReplace *OrderEdit             `json:"current_pending_replace"`
	CommissionDetailTotal *CommissionDetailTotal `json:"commission_detail_total"`
	// WorkableSize is how much of the originating order has filled, and WorkableSizeCompletionPercentage the percentage
	WorkableSize                     types.Number         `json:"workable_size"`
	WorkableSizeCompletionPercentage types.Number         `json:"workable_size_completion_pct"`
	ProductDetails                   *OrderProductDetails `json:"product_details"`
	CostBasisMethod                  string               `json:"cost_basis_method"`
	// DisplayedOrderConfig is an equity order's time in force
	DisplayedOrderConfig string `json:"displayed_order_config"`
	// EquityTradingSession is the session an equity order may execute in
	EquityTradingSession string `json:"equity_trading_session"`
	PredictionSide       string `json:"prediction_side"`
	// LastUpdateTime is the time of the order's latest fill, or its creation time before it fills
	LastUpdateTime time.Time `json:"last_update_time"`
	// TotalValueAfterFeesNative, OutstandingHoldAmountNative and TotalFeesNative are in the user's native currency,
	// unset when no rate is available; the hold converts from the quote currency for a buy and the base for a sell
	TotalValueAfterFeesNative   *Amount `json:"total_value_after_fees_native"`
	OutstandingHoldAmountNative *Amount `json:"outstanding_hold_amount_native"`
	TotalFeesNative             *Amount `json:"total_fees_native"`
	// OrderDataSource and FundingSource are undocumented; Coinbase sends them
	OrderDataSource string `json:"order_data_source"`
	FundingSource   string `json:"funding_source"`
	// OptionLegs, Holds and ComboLegs are undocumented and have only been seen empty, so they are kept as sent
	OptionLegs json.RawMessage `json:"option_legs"`
	Holds      json.RawMessage `json:"holds"`
	ComboLegs  json.RawMessage `json:"combo_legs"`
}

// OrderEdit is an edit of an order's price and size
type OrderEdit struct {
	Price                  types.Number `json:"price"`
	Size                   types.Number `json:"size"`
	ReplaceAcceptTimestamp time.Time    `json:"replace_accept_timestamp"`
}

// OrderProductDetails holds the details of an order's product that depend on its type
type OrderProductDetails struct {
	EquityDetails *EquityOrderProductDetails `json:"equity_details"`
}

// EquityOrderProductDetails identifies an equity order's product
type EquityOrderProductDetails struct {
	BaseCBRN string `json:"base_cbrn"`
	Ticker   string `json:"ticker"`
	QuoteID  string `json:"quote_id"`
}

// ListFillsRequest selects the fills ListFills returns
type ListFillsRequest struct {
	OrderIDs   []string
	TradeIDs   []string
	ProductIDs []string
	// StartSequenceTimestamp and EndSequenceTimestamp bound the fills' trade times
	StartSequenceTimestamp time.Time
	EndSequenceTimestamp   time.Time
	// Limit is how many fills the page holds; Coinbase holds 100 when it is unset
	Limit uint64
	// Cursor pages from where a previous page's cursor left off
	Cursor string
	// SortBy sorts by PRICE or TRADE_TIME, which pages unstably; Coinbase sorts by creation time by default
	SortBy string
	// AssetFilters selects fills whose base, quote or underlying asset is one of these
	AssetFilters []currency.Code
	OrderTypes   []string
	// OrderSide is BUY or SELL
	OrderSide    string
	ProductTypes []string
	// ProofToken is a two factor proof the EU's strong customer authentication can require to list fills, which the
	// response's ProofTokenRequired asks for
	ProofToken string
}

// ListFillsResponse is a page of fills
type ListFillsResponse struct {
	Fills []Fill `json:"fills"`
	// Cursor, when set, starts the next page
	Cursor string `json:"cursor"`
	// ProofTokenRequired is set when the EU's strong customer authentication needs a proof token to list fills
	ProofTokenRequired bool `json:"proof_token_required"`
}

// Fill is a fill of an order
type Fill struct {
	EntryID string `json:"entry_id"`
	// TradeID is unique to a FILL trade type; an adjusted fill shares its fill's
	TradeID   string    `json:"trade_id"`
	OrderID   string    `json:"order_id"`
	TradeTime time.Time `json:"trade_time"`
	// TradeType is FILL for a regular fill and REVERSAL, CORRECTION or SYNTHETIC for an adjusted one
	TradeType  string       `json:"trade_type"`
	Price      types.Number `json:"price"`
	Size       types.Number `json:"size"`
	Commission types.Number `json:"commission"`
	ProductID  string       `json:"product_id"`
	// SequenceTimestamp is when the fill was posted
	SequenceTimestamp  time.Time `json:"sequence_timestamp"`
	LiquidityIndicator string    `json:"liquidity_indicator"`
	// SizeInQuote is set for a fill of an order sized in the quote currency
	SizeInQuote bool   `json:"size_in_quote"`
	UserID      string `json:"user_id"`
	// Side is the fill's order's side
	Side              string `json:"side"`
	RetailPortfolioID string `json:"retail_portfolio_id"`
	FillSource        string `json:"fillSource"`
	// CommissionDetailTotal breaks the fill's commission down
	CommissionDetailTotal *CommissionDetailTotal `json:"commission_detail_total"`
	OrderDataSource       string                 `json:"order_data_source"`
	// FutureLegs are a futures combo fill's legs, empty for any other fill
	FutureLegs []FillFutureLeg `json:"future_legs"`
	// OptionLegs is undocumented and has only been seen empty, so it is kept as sent
	OptionLegs json.RawMessage `json:"option_legs"`
	// RealisedProfitLoss and ProductType are undocumented; Coinbase sends them
	RealisedProfitLoss types.Number `json:"realized_pl"`
	ProductType        string       `json:"product_type"`
}

// FillFutureLeg is a leg of a futures combo fill
type FillFutureLeg struct {
	ProductID string `json:"product_id"`
	// ComboID is shared by every leg of the combo fill
	ComboID     string       `json:"combo_id"`
	TradeID     string       `json:"trade_id"`
	FilledPrice types.Number `json:"filled_price"`
	FilledSize  types.Number `json:"filled_size"`
	Side        string       `json:"side"`
}

// ListOrdersRequest selects the orders ListOrders returns
type ListOrdersRequest struct {
	OrderIDs   []string
	ProductIDs []string
	// ProductType selects one product type's orders, EQUITY for equities; Coinbase lists every type by default
	ProductType   string
	OrderStatuses []string
	TimeInForces  []string
	OrderTypes    []string
	// OrderSide is BUY or SELL
	OrderSide string
	// StartDate and EndDate bound the orders' creation times, the start inclusive and the end exclusive
	StartDate time.Time
	EndDate   time.Time
	// OrderPlacementSource selects orders placed from one source; Coinbase lists RETAIL_ADVANCED orders by default
	OrderPlacementSource string
	// ContractExpiryType selects expiring or perpetual futures, for ProductType FUTURE
	ContractExpiryType string
	// AssetFilters selects orders whose base, quote or underlying asset is one of these
	AssetFilters []currency.Code
	// Limit is how many orders the page holds; Coinbase has no default
	Limit uint64
	// Cursor pages from where a previous page's cursor left off
	Cursor string
	// SortBy sorts by LIMIT_PRICE, LAST_FILL_TIME or LAST_UPDATE_TIME, which pages unstably; Coinbase sorts by
	// creation time by default
	SortBy string
	// UseSimplifiedTotalValueCalculation selects how orders' total values are calculated; Coinbase simplifies them
	// when it is unset
	UseSimplifiedTotalValueCalculation *bool
	// ProofToken is a two factor proof the EU's strong customer authentication can require to list orders, which the
	// response's ProofTokenRequired asks for
	ProofToken string
}

// ListOrdersResponse is a page of orders
type ListOrdersResponse struct {
	Orders []Order `json:"orders"`
	// Sequence is deprecated; it was the database sequence the orders were read at
	Sequence types.Number `json:"sequence"`
	HasNext  bool         `json:"has_next"`
	// Cursor starts the next page while HasNext is set
	Cursor string `json:"cursor"`
	// ProofTokenRequired is set when the EU's strong customer authentication needs a proof token to list orders
	ProofTokenRequired bool `json:"proof_token_required"`
}

// PreviewOrderRequest is an order PreviewOrder previews, as CreateOrder would place it
type PreviewOrderRequest struct {
	// ProductID is the product's canonical ID, an equity's rather than its display ticker
	ProductID string
	// Side is BUY or SELL
	Side               string
	OrderConfiguration OrderConfigurationRequest
	// Leverage is the order's leverage; Coinbase uses 1 when it is unset
	Leverage float64
	// MarginType is CROSS, the default, or ISOLATED
	MarginType string
	// AttachedOrderConfiguration attaches a take profit/stop loss order to the order: a *TriggerBracketGTCRequest
	// without a size, as the attached order takes the order's own
	AttachedOrderConfiguration OrderConfigurationRequest
	EquityOrderMetadata        *EquityOrderMetadata
	PredictionMetadata         *PredictionMetadataRequest
	// CostBasisMethod chooses how the order's fills match tax lots
	CostBasisMethod string
}

// PreviewOrderResponse is an order's preview: its totals, and why it would fail
type PreviewOrderResponse struct {
	OrderTotal types.Number `json:"order_total"`
	// CommissionTotal is the commission charged, an amount rather than a rate
	CommissionTotal types.Number `json:"commission_total"`
	// Errors are the reasons the order would fail
	Errors    []string     `json:"errs"`
	Warnings  []string     `json:"warning"`
	QuoteSize types.Number `json:"quote_size"`
	BaseSize  types.Number `json:"base_size"`
	BestBid   types.Number `json:"best_bid"`
	BestAsk   types.Number `json:"best_ask"`
	// IsMax is set when the order is sized to the whole tradable balance
	IsMax            bool         `json:"is_max"`
	OrderMarginTotal types.Number `json:"order_margin_total"`
	Leverage         types.Number `json:"leverage"`
	LongLeverage     types.Number `json:"long_leverage"`
	ShortLeverage    types.Number `json:"short_leverage"`
	Slippage         types.Number `json:"slippage"`
	// PreviewID links an order CreateOrder places to this preview
	PreviewID string `json:"preview_id"`
	// CurrentLiquidationBuffer and ProjectedLiquidationBuffer are giving way to MarginRatioData
	CurrentLiquidationBuffer   types.Number `json:"current_liquidation_buffer"`
	ProjectedLiquidationBuffer types.Number `json:"projected_liquidation_buffer"`
	MaximumLeverage            types.Number `json:"max_leverage"`
	// PNLConfiguration estimates the profit or loss of an order with an attached take profit/stop loss, before fees
	// and slippage
	PNLConfiguration *PNLConfiguration `json:"pnl_configuration"`
	// TWAPBucketMetadata is the size and duration of each suborder of a time weighted average price order
	TWAPBucketMetadata                 *TWAPBucketMetadata `json:"twap_bucket_metadata"`
	PositionNotionalLimit              types.Number        `json:"position_notional_limit"`
	MaximumNotionalAtRequestedLeverage types.Number        `json:"max_notional_at_requested_leverage"`
	MarginRatioData                    *MarginRatioData    `json:"margin_ratio_data"`
	// CommissionDetailTotal breaks the commission down
	CommissionDetailTotal *CommissionDetailTotal `json:"commission_detail_total"`
	// ScaledMetadata is the size and price of each child order of a scaled order
	ScaledMetadata      *ScaledOrderMetadata `json:"scaled_metadata"`
	ComplianceLimitData *ComplianceLimitData `json:"compliance_limit_data"`
	// EstimatedAverageFilledPrice is the price the order is estimated to fill at
	EstimatedAverageFilledPrice types.Number             `json:"est_average_filled_price"`
	PredictionOrderMetadata     *PredictionOrderMetadata `json:"prediction_order_metadata"`
	// PredictedLiquidationPrice is a US derivative order's predicted liquidation price
	PredictedLiquidationPrice types.Number `json:"predicted_liquidation_price"`
	// AvailableBuyingPower is the futures buying power the order's funding shortfall is calculated from
	AvailableBuyingPower types.Number `json:"available_buying_power"`
	// MaximumLoss is the order's greatest possible loss, its total for an equity option buy
	MaximumLoss types.Number `json:"max_loss"`
	// EquityOrderMetadata, FillLikelihood, OptionContractLimit, OptionPremiumLimitUSD, OptionGrossContractLimit and
	// OptionGrossContractsHeld are undocumented; Coinbase sends them
	EquityOrderMetadata      *EquityOrderMetadata `json:"equity_order_metadata"`
	FillLikelihood           string               `json:"fill_likelihood"`
	OptionContractLimit      types.Number         `json:"option_contract_limit"`
	OptionPremiumLimitUSD    types.Number         `json:"option_premium_limit_usd"`
	OptionGrossContractLimit types.Number         `json:"option_gross_contract_limit"`
	OptionGrossContractsHeld types.Number         `json:"option_gross_contracts_held"`
}

// Err returns errOrderPreviewFailed with Coinbase's reasons when the order would fail
func (r *PreviewOrderResponse) Err() error {
	if len(r.Errors) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", errOrderPreviewFailed, strings.Join(r.Errors, ", "))
}

// TWAPBucketMetadata is the size and duration of each suborder of a time weighted average price order
type TWAPBucketMetadata struct {
	BucketDuration  Duration     `json:"bucket_duration"`
	BucketSize      types.Number `json:"bucket_size"`
	NumberOfBuckets types.Number `json:"number_buckets"`
	StartTime       time.Time    `json:"start_time"`
	EndTime         time.Time    `json:"end_time"`
}

// MarginRatioData is an order's current and projected margin ratios, which replace the liquidation buffers
type MarginRatioData struct {
	CurrentMarginRatio   types.Number `json:"current_margin_ratio"`
	ProjectedMarginRatio types.Number `json:"projected_margin_ratio"`
}

// ScaledOrderMetadata is how a scaled order distributes itself across its child orders
type ScaledOrderMetadata struct {
	ScaledOrderDistribution []ScaledOrderDistribution `json:"scaled_order_distribution"`
}

// ScaledOrderDistribution is a scaled order's child order's size and price, and why it would fail
type ScaledOrderDistribution struct {
	Size     types.Number `json:"size"`
	Price    types.Number `json:"price"`
	Errors   []string     `json:"errs"`
	Warnings []string     `json:"warning"`
}

// ComplianceLimitData is the account's compliance purchase limit and what remains of it
type ComplianceLimitData struct {
	TotalLimit      *Amount `json:"total_limit"`
	RemainingLimits *Amount `json:"remaining_limits"`
}

// PredictionOrderMetadata is a prediction market order's preview, from the user's side of the market
type PredictionOrderMetadata struct {
	// ContractSubtotal is the contracts' base value
	ContractSubtotal types.Number `json:"contract_subtotal"`
	// UserNetTotal is what the user pays for a buy, or receives for a sell, after fees
	UserNetTotal types.Number `json:"user_net_total"`
	// SlippagePercentage is the slippage tolerance of the order's price tier
	SlippagePercentage types.Number `json:"slippage_percentage"`
	// MinimumContracts is the fewest contracts a buy sized in the quote currency receives after slippage
	MinimumContracts types.Number `json:"minimum_contracts"`
	// MinimumTotal is the least a sell sized in the base currency receives after slippage
	MinimumTotal *Amount `json:"minimum_total"`
	// MaximumTotal is the most a buy sized in the base currency pays after slippage
	MaximumTotal *Amount `json:"maximum_total"`
	// MaximumContracts is the most contracts a sell sized in the quote currency sells after slippage
	MaximumContracts types.Number `json:"maximum_contracts"`
}
