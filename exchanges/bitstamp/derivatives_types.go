package bitstamp

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Position statuses
const (
	PositionStatusOpen              = "OPEN"
	PositionStatusWaitingSettlement = "WAITING_SETTLEMENT"
	PositionStatusSettled           = "SETTLED"
	PositionStatusLiquidating       = "LIQUIDATING"
)

// Position sides
const (
	PositionSideLong  = "LONG"
	PositionSideShort = "SHORT"
)

// Position settlement types
const (
	SettlementTypePeriodic = "PERIODIC"
	SettlementTypeClosed   = "CLOSED"
)

// Derivatives trade sides; a self trade matches two of the account's own orders
const (
	TradeSideBuy  = "BUY"
	TradeSideSell = "SELL"
	TradeSideSelf = "SELF"
)

// Derivatives trade types
const (
	TradeTypeTrade                     = "TRADE"
	TradeTypeLiquidation               = "LIQUIDATION"
	TradeTypeADL                       = "ADL"
	TradeTypeAssignmentProgram         = "ASSIGNMENT_PROGRAM"
	TradeTypeMarketWidePositionClosure = "MARKET_WIDE_POSITION_CLOSURE"
)

// Reference index publishing schedule types
const (
	ScheduleTypeOpenHours   = "OPEN_HOURS"
	ScheduleTypeClosedHours = "CLOSED_HOURS"
)

// Outcomes of checking whether an estimated order can be placed
const (
	OrderPlaceableUnspecified          = "UNSPECIFIED"
	OrderPlaceableYes                  = "YES"
	OrderPlaceableNoMMRBreach          = "NO_MMR_BREACH"
	OrderPlaceableNoIMRBreach          = "NO_IMR_BREACH"
	OrderPlaceableNoInsufficientMargin = "NO_INSUFFICIENT_MARGIN"
)

// MarginTiersResponse holds the margin tiers of a derivatives market
type MarginTiersResponse struct {
	Market string       `json:"market"`
	Tiers  []MarginTier `json:"tiers"`
}

// MarginTier holds the margin requirements for a range of position sizes
// The margin rates are fractions of the position value, e.g. 0.1 for 10%, although the API documents them as percentages
type MarginTier struct {
	Tier                              uint64       `json:"tier,string"`
	SizeLimitLow                      types.Number `json:"size_limit_low"`
	SizeLimitHigh                     types.Number `json:"size_limit_high"`
	MaxLeverage                       types.Number `json:"max_leverage"`
	InitialMarginRate                 types.Number `json:"initial_margin_rate"`
	MaintenanceMarginRate             types.Number `json:"maintenance_margin_rate"`
	CloseOutMarginRate                types.Number `json:"close_out_margin_rate"`
	InitialMarginPreviousLevelMax     types.Number `json:"initial_margin_previous_level_max"`
	MaintenanceMarginPreviousLevelMax types.Number `json:"maintenance_margin_previous_level_max"`
	CloseOutMarginPreviousLevelMax    types.Number `json:"close_out_margin_previous_level_max"`
}

// MarketHoursResponse holds the reference index publishing schedule of a derivatives market
type MarketHoursResponse struct {
	Market                        string                        `json:"market"`
	TradingHours                  string                        `json:"trading_hours"`
	ReferenceIndexPublishingHours ReferenceIndexPublishingHours `json:"reference_index_publishing_hours"`
}

// ReferenceIndexPublishingHours holds the status and schedule of a reference index
// The current open and close times are zero when the reference index is not publishing
type ReferenceIndexPublishingHours struct {
	IsReferenceIndexPublishing bool            `json:"is_reference_index_publishing"`
	CurrentOpen                time.Time       `json:"current_open"`
	CurrentClose               time.Time       `json:"current_close"`
	NextOpen                   time.Time       `json:"next_open"`
	NextClose                  time.Time       `json:"next_close"`
	Schedule                   []ScheduleEntry `json:"schedule"`
}

// ScheduleEntry holds a period in which a reference index is or is not publishing
type ScheduleEntry struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Type  string    `json:"type"`
}

// PositionResponse holds an open derivatives position
type PositionResponse struct {
	ID                        string        `json:"id"`
	Market                    string        `json:"market"`
	MarketType                string        `json:"market_type"`
	MarginMode                string        `json:"margin_mode"`
	SettlementCurrency        currency.Code `json:"settlement_currency"`
	EntryPrice                types.Number  `json:"entry_price"`
	PNLPercentage             types.Number  `json:"pnl_percentage"`
	PNLRealised               types.Number  `json:"pnl_realized"`
	PNLSettledSinceInception  types.Number  `json:"pnl_settled_since_inception"`
	Leverage                  types.Number  `json:"leverage"`
	PNL                       types.Number  `json:"pnl"`
	CumulativePricePNL        types.Number  `json:"cumulative_price_pnl"`
	CumulativeTradingFees     types.Number  `json:"cumulative_trading_fees"`
	CumulativeLiquidationFees types.Number  `json:"cumulative_liquidation_fees"`
	CumulativeFunding         types.Number  `json:"cumulative_funding"`
	CumulativeSocialisedLoss  types.Number  `json:"cumulative_socialized_loss"`
	Size                      types.Number  `json:"size"`
	PNLUnrealised             types.Number  `json:"pnl_unrealized"`
	PNLInSettlement           types.Number  `json:"pnl_in_settlement"`
	ImpliedLeverage           types.Number  `json:"implied_leverage"`
	InitialMargin             types.Number  `json:"initial_margin"`
	InitialMarginRatio        types.Number  `json:"initial_margin_ratio"`
	CurrentMargin             types.Number  `json:"current_margin"`
	CollateralReserved        types.Number  `json:"collateral_reserved"`
	MaintenanceMargin         types.Number  `json:"maintenance_margin"`
	MaintenanceMarginRatio    types.Number  `json:"maintenance_margin_ratio"`
	EstimatedLiquidationPrice types.Number  `json:"estimated_liquidation_price"`
	EstimatedClosingFeeAmount types.Number  `json:"estimated_closing_fee_amount"`
	MarkPrice                 types.Number  `json:"mark_price"`
	CurrentValue              types.Number  `json:"current_value"`
	EntryValue                types.Number  `json:"entry_value"`
	StrikePrice               types.Number  `json:"strike_price"`
	Side                      string        `json:"side"`
	MarginTier                string        `json:"margin_tier"`
}

// PositionStatusResponse holds the status of an open or closed derivatives position
// Fields describing the current state of a position are only populated for open positions and the exit and closing
// fields are only populated for closed positions
type PositionStatusResponse struct {
	PositionResponse
	TimeOpened      types.Time   `json:"time_opened"`
	Status          string       `json:"status"`
	ExitPrice       types.Number `json:"exit_price"`
	SettlementPrice types.Number `json:"settlement_price"`
	AmountDelta     types.Number `json:"amount_delta"`
	TimeClosed      types.Time   `json:"time_closed"`
}

// PositionHistoryRequest holds the parameters for fetching position history
// Pair is optional and restricts the results to a single market
type PositionHistoryRequest struct {
	Pair    currency.Pair
	Offset  uint64
	Limit   uint64
	SinceID string
	Sort    string
}

// PositionHistoryResponse holds a historical derivatives position
type PositionHistoryResponse struct {
	ID                        string        `json:"id"`
	Market                    string        `json:"market"`
	MarketType                string        `json:"market_type"`
	MarginMode                string        `json:"margin_mode"`
	PNLCurrency               currency.Code `json:"pnl_currency"`
	EntryPrice                types.Number  `json:"entry_price"`
	PNLPercentage             types.Number  `json:"pnl_percentage"`
	PNLRealised               types.Number  `json:"pnl_realized"`
	PNLSettled                types.Number  `json:"pnl_settled"`
	Leverage                  types.Number  `json:"leverage"`
	PNL                       types.Number  `json:"pnl"`
	CumulativePricePNL        types.Number  `json:"cumulative_price_pnl"`
	CumulativeTradingFees     types.Number  `json:"cumulative_trading_fees"`
	CumulativeLiquidationFees types.Number  `json:"cumulative_liquidation_fees"`
	CumulativeFunding         types.Number  `json:"cumulative_funding"`
	CumulativeSocialisedLoss  types.Number  `json:"cumulative_socialized_loss"`
	AmountDelta               types.Number  `json:"amount_delta"`
	TimeOpened                types.Time    `json:"time_opened"`
	TimeClosed                types.Time    `json:"time_closed"`
	Status                    string        `json:"status"`
	ExitPrice                 types.Number  `json:"exit_price"`
	SettlementPrice           types.Number  `json:"settlement_price"`
}

// ClosePositionsRequest holds the parameters for closing positions with market orders
// Pair and MarginMode are optional and restrict the positions which are closed
type ClosePositionsRequest struct {
	Pair       currency.Pair
	MarginMode string
}

// ClosePositionsResponse holds the positions which were and were not closed
type ClosePositionsResponse struct {
	Closed []ClosedPositionResponse `json:"closed"`
	Failed []ClosedPositionResponse `json:"failed"`
}

// ClosedPositionResponse holds a position closed by a close position request
type ClosedPositionResponse struct {
	PositionHistoryResponse
	ClosingFeeAmount types.Number `json:"closing_fee_amount"`
}

// SettlementTransactionsRequest holds the parameters for fetching position settlement transactions
// TransactionID is optional and restricts the results to the settlement transactions of a single market transaction
type SettlementTransactionsRequest struct {
	TransactionID string
	Offset        uint64
	Limit         uint64
	SinceID       string
	Sort          string
	Since         time.Time
	Until         time.Time
}

// SettlementTransactionResponse holds a position settlement transaction
type SettlementTransactionResponse struct {
	TransactionID              string        `json:"transaction_id"`
	PositionID                 string        `json:"position_id"`
	SettlementTime             types.Time    `json:"settlement_time"`
	SettlementType             string        `json:"settlement_type"`
	SettlementPrice            types.Number  `json:"settlement_price"`
	Market                     string        `json:"market"`
	MarketType                 string        `json:"market_type"`
	PNLCurrency                currency.Code `json:"pnl_currency"`
	PNLSettled                 types.Number  `json:"pnl_settled"`
	PNLComponentPrice          types.Number  `json:"pnl_component_price"`
	PNLComponentFees           types.Number  `json:"pnl_component_fees"`
	PNLComponentFunding        types.Number  `json:"pnl_component_funding"`
	PNLComponentSocialisedLoss types.Number  `json:"pnl_component_socialized_loss"`
	MarginMode                 string        `json:"margin_mode"`
	Size                       types.Number  `json:"size"`
	StrikePrice                types.Number  `json:"strike_price"`
	FeesComponentTrading       types.Number  `json:"fees_component_trading"`
	FeesComponentLiquidation   types.Number  `json:"fees_component_liquidation"`
}

// DerivativesTradeHistoryRequest holds the parameters for fetching derivatives trade history
// Either Pair or OrderID must be supplied
type DerivativesTradeHistoryRequest struct {
	Pair    currency.Pair
	OrderID uint64
	Limit   uint64
	Sort    string
	Since   time.Time
	Until   time.Time
	AfterID uint64
}

// DerivativesTradeResponse holds a derivatives trade
type DerivativesTradeResponse struct {
	TradeID             uint64         `json:"trade_id,string"`
	OrderID             uint64         `json:"order_id,string"`
	SelfTradeOrderID    string         `json:"self_trade_order_id"`
	PositionID          string         `json:"position_id"`
	SelfTradePositionID string         `json:"self_trade_position_id"`
	DateTime            types.DateTime `json:"datetime"`
	Fee                 types.Number   `json:"fee"`
	LiquidationFee      types.Number   `json:"liquidation_fee"`
	FeeCurrency         currency.Code  `json:"fee_currency"`
	Market              string         `json:"market"`
	MarginMode          string         `json:"margin_mode"`
	Leverage            types.Number   `json:"leverage"`
	Side                string         `json:"side"`
	Type                string         `json:"type"`
	SelfTradeType       string         `json:"self_trade_type"`
	Price               types.Number   `json:"price"`
	Amount              types.Number   `json:"amount"`
	TradeUTI            string         `json:"trade_uti"`
}

// MarginInfoResponse holds the account's derivatives margin information
type MarginInfoResponse struct {
	AccountMargin          types.Number      `json:"account_margin"`
	AccountMarginAvailable types.Number      `json:"account_margin_available"`
	AccountMarginReserved  types.Number      `json:"account_margin_reserved"`
	AccountMarginCurrency  currency.Code     `json:"account_margin_currency"`
	Assets                 []AssetMarginInfo `json:"assets"`
	InitialMarginRatio     types.Number      `json:"initial_margin_ratio"`
	MaintenanceMarginRatio types.Number      `json:"maintenance_margin_ratio"`
	ImpliedLeverage        types.Number      `json:"implied_leverage"`
}

// AssetMarginInfo holds the margin information of a collateral currency
type AssetMarginInfo struct {
	Asset           currency.Code `json:"asset"`
	TotalAmount     types.Number  `json:"total_amount"`
	Available       types.Number  `json:"available"`
	Reserved        types.Number  `json:"reserved"`
	MarginAvailable types.Number  `json:"margin_available"`
}

// OrderImpactRequest holds the parameters for estimating the margin impact of an order
// AdditionalCollateral holds collateral amounts to include in the estimate as if they were already posted
type OrderImpactRequest struct {
	Pair                 currency.Pair
	OrderType            string
	Amount               float64
	Side                 order.Side
	MarginMode           string
	Leverage             float64
	Price                float64
	ReduceOnly           bool
	AdditionalCollateral map[currency.Code]float64
}

// OrderImpactResponse holds the estimated margin impact of an order
// EstimatedLiquidationPrices is keyed by market name
type OrderImpactResponse struct {
	MarginCurrency             currency.Code           `json:"margin_currency"`
	AdditionalRequiredMargin   types.Number            `json:"additional_required_margin"`
	MarginRequiredForOrder     types.Number            `json:"margin_required_for_order"`
	EstimatedLiquidationPrices map[string]types.Number `json:"estimated_liquidation_prices"`
	EstimatedFee               types.Number            `json:"estimated_fee"`
	IsOrderPlaceable           string                  `json:"is_order_placeable"`
	MarginTier                 OrderImpactMarginTier   `json:"margin_tier"`
}

// OrderImpactMarginTier holds the margin tier breakdown of an estimated order impact
type OrderImpactMarginTier struct {
	Breakdown []MarginTierExposure `json:"breakdown"`
}

// MarginTierExposure holds the exposure and leverage of a margin tier
type MarginTierExposure struct {
	Tier     types.Number `json:"tier"`
	Exposure types.Number `json:"exposure"`
	Leverage types.Number `json:"leverage"`
}

// CollateralChangeImpactRequest holds the parameters for estimating the impact of changing collateral
// Exactly one of TargetCollateral or CollateralDeltas must be supplied and Pair is only used with isolated margin
type CollateralChangeImpactRequest struct {
	MarginMode       string
	Pair             currency.Pair
	TargetCollateral map[currency.Code]float64
	CollateralDeltas map[currency.Code]float64
}

// CollateralChangeImpactResponse holds the estimated impact of changing collateral
// EstimatedLiquidationPrices is keyed by position ID
type CollateralChangeImpactResponse struct {
	MarginCurrency                  currency.Code           `json:"margin_currency"`
	EstimatedInitialMarginRatio     types.Number            `json:"estimated_initial_margin_ratio"`
	EstimatedMaintenanceMarginRatio types.Number            `json:"estimated_maintenance_margin_ratio"`
	EstimatedLiquidationPrices      map[string]types.Number `json:"estimated_liquidation_prices"`
	TotalEstimatedMargin            types.Number            `json:"total_estimated_margin"`
}

// AdjustPositionCollateralRequest holds the parameters for setting the collateral amount of an isolated margin position
type AdjustPositionCollateralRequest struct {
	PositionID string
	NewAmount  float64
}

// LeverageSettingsRequest holds the optional filters for fetching leverage settings
type LeverageSettingsRequest struct {
	MarginMode string
	Pair       currency.Pair
}

// CollateralCurrencyResponse holds a currency which may be used as collateral and its haircut
type CollateralCurrencyResponse struct {
	Currency currency.Code `json:"currency"`
	Haircut  types.Number  `json:"haircut"`
}

// LeverageSettingRequest holds the parameters for updating the leverage of a derivatives market
type LeverageSettingRequest struct {
	Pair       currency.Pair
	MarginMode string
	Leverage   float64
}

// LeverageSettingResponse holds the leverage setting of a derivatives market
type LeverageSettingResponse struct {
	MarginMode      string       `json:"margin_mode"`
	Market          string       `json:"market"`
	LeverageCurrent types.Number `json:"leverage_current"`
	LeverageMax     types.Number `json:"leverage_max"`
}

// closePositionsBody is the JSON body of a close positions request
type closePositionsBody struct {
	Market     string `json:"market,omitempty"`
	MarginMode string `json:"margin_mode,omitempty"`
	OrderType  string `json:"order_type"`
}

// closePositionBody is the JSON body of a close position request
type closePositionBody struct {
	PositionID string `json:"position_id"`
}

// orderImpactBody is the JSON body of an estimated order impact request
type orderImpactBody struct {
	Market               string            `json:"market"`
	OrderType            string            `json:"order_type"`
	Amount               float64           `json:"amount"`
	OrderSide            string            `json:"order_side"`
	MarginMode           string            `json:"margin_mode"`
	Leverage             float64           `json:"leverage"`
	Price                float64           `json:"price,omitempty"`
	ReduceOnly           bool              `json:"reduce_only,omitempty"`
	AdditionalCollateral map[string]string `json:"additional_collateral,omitempty"`
}

// collateralChangeImpactBody is the JSON body of a collateral change impact request
type collateralChangeImpactBody struct {
	MarginMode       string            `json:"margin_mode"`
	Market           string            `json:"market,omitempty"`
	TargetCollateral map[string]string `json:"target_collateral,omitempty"`
	CollateralDeltas map[string]string `json:"collateral_deltas,omitempty"`
}

// adjustPositionCollateralBody is the JSON body of an adjust position collateral request
type adjustPositionCollateralBody struct {
	PositionID string  `json:"position_id"`
	NewAmount  float64 `json:"new_amount"`
}

// leverageSettingBody is the JSON body of a leverage setting update
type leverageSettingBody struct {
	MarginMode string  `json:"margin_mode"`
	Market     string  `json:"market"`
	Leverage   float64 `json:"leverage,omitempty"`
}

// Margin modes used by derivatives markets
const (
	MarginModeCross    = "CROSS"
	MarginModeIsolated = "ISOLATED"
)
