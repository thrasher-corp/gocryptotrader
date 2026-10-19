package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/types"
)

// Intraday margin settings, which SetIntradayMarginSetting sets
const (
	// IntradayMarginSettingStandard keeps the overnight margin requirement all day
	IntradayMarginSettingStandard = "INTRADAY_MARGIN_SETTING_STANDARD"
	// IntradayMarginSettingIntraday opts in to more leverage in the intraday window, 8am to 4pm ET on weekdays that
	// are not market holidays
	IntradayMarginSettingIntraday = "INTRADAY_MARGIN_SETTING_INTRADAY"
)

// Margin profile types, which select the margin profile GetCurrentMarginWindow returns the window of
const (
	MarginProfileTypeRetailRegular         = "MARGIN_PROFILE_TYPE_RETAIL_REGULAR"
	MarginProfileTypeRetailIntradayMargin1 = "MARGIN_PROFILE_TYPE_RETAIL_INTRADAY_MARGIN_1"
)

// Margin window types, the margin window a US derivatives account is in
const (
	MarginWindowTypeOvernight  = "MARGIN_WINDOW_TYPE_OVERNIGHT"
	MarginWindowTypeWeekend    = "MARGIN_WINDOW_TYPE_WEEKEND"
	MarginWindowTypeIntraday   = "MARGIN_WINDOW_TYPE_INTRADAY"
	MarginWindowTypeTransition = "MARGIN_WINDOW_TYPE_TRANSITION"
)

// FCM margin window types, the margin window a balance summary's margin window measure is for
const (
	FCMMarginWindowTypeOvernight  = "FCM_MARGIN_WINDOW_TYPE_OVERNIGHT"
	FCMMarginWindowTypeWeekend    = "FCM_MARGIN_WINDOW_TYPE_WEEKEND"
	FCMMarginWindowTypeIntraday   = "FCM_MARGIN_WINDOW_TYPE_INTRADAY"
	FCMMarginWindowTypeTransition = "FCM_MARGIN_WINDOW_TYPE_TRANSITION"
)

// Margin level types, how near a US derivatives account is to liquidation
const (
	MarginLevelTypeBase        = "MARGIN_LEVEL_TYPE_BASE"
	MarginLevelTypeWarning     = "MARGIN_LEVEL_TYPE_WARNING"
	MarginLevelTypeDanger      = "MARGIN_LEVEL_TYPE_DANGER"
	MarginLevelTypeLiquidation = "MARGIN_LEVEL_TYPE_LIQUIDATION"
)

// Futures position sides
const (
	FuturesPositionSideLong  = "LONG"
	FuturesPositionSideShort = "SHORT"
)

// Futures sweep statuses
const (
	// FuturesSweepStatusPending is a sweep that has not started processing, which CancelPendingFuturesSweep can cancel
	FuturesSweepStatusPending = "PENDING"
	// FuturesSweepStatusProcessing is a sweep being processed, which can no longer be cancelled
	FuturesSweepStatusProcessing = "PROCESSING"
)

// futuresSweepResponse reports whether a sweep was scheduled or cancelled
type futuresSweepResponse struct {
	Success bool `json:"success"`
}

// CurrentMarginWindowResponse is the margin window a US derivatives account is in, and whether Coinbase has turned
// intraday margin off
type CurrentMarginWindowResponse struct {
	MarginWindow MarginWindow `json:"margin_window"`
	// IsIntradayMarginKillswitchEnabled is set while Coinbase has turned intraday margin off, and
	// IsIntradayMarginEnrolmentKillswitchEnabled while it has turned enrolment in it off
	IsIntradayMarginKillswitchEnabled          bool `json:"is_intraday_margin_killswitch_enabled"`
	IsIntradayMarginEnrolmentKillswitchEnabled bool `json:"is_intraday_margin_enrollment_killswitch_enabled"`
}

// MarginWindow is a margin window and when it ends
type MarginWindow struct {
	// MarginWindowType is one of the MarginWindowType values
	MarginWindowType string    `json:"margin_window_type"`
	EndTime          time.Time `json:"end_time"`
}

// futuresBalanceSummaryResponse holds a US derivatives balance summary
type futuresBalanceSummaryResponse struct {
	BalanceSummary *FuturesBalanceSummary `json:"balance_summary"`
}

// FuturesBalanceSummary is a US derivatives account's balances and margin. Futures are margined from a futures
// account at Coinbase Financial Markets (CFM), which cash moves to from the spot account at Coinbase Inc. (CBI) as
// margin requires; the margin ratio is AvailableMargin divided by LiquidationThreshold
type FuturesBalanceSummary struct {
	// FuturesBuyingPower is the cash available to trade US derivatives
	FuturesBuyingPower Amount `json:"futures_buying_power"`
	// TotalUSDBalance is the USD held across the CFM futures account and the CBI spot account
	TotalUSDBalance Amount `json:"total_usd_balance"`
	CBIUSDBalance   Amount `json:"cbi_usd_balance"`
	// CFMUSDBalance is the USD held in the futures account, which cannot trade spot
	CFMUSDBalance Amount `json:"cfm_usd_balance"`
	// TotalOpenOrdersHoldAmount is held for spot and futures open orders
	TotalOpenOrdersHoldAmount Amount `json:"total_open_orders_hold_amount"`
	UnrealisedPNL             Amount `json:"unrealized_pnl"`
	// DailyRealisedPNL is the profit or loss realised on the current trade date
	DailyRealisedPNL Amount `json:"daily_realized_pnl"`
	// InitialMargin is the margin the futures positions require; funds held for it cannot trade spot
	InitialMargin Amount `json:"initial_margin"`
	// AvailableMargin is the spot and futures USD and the futures profit and loss, less holds for open orders
	AvailableMargin Amount `json:"available_margin"`
	// LiquidationThreshold is the available margin at which positions are liquidated
	LiquidationThreshold Amount `json:"liquidation_threshold"`
	// LiquidationBufferAmount is AvailableMargin less LiquidationThreshold, and LiquidationBufferPercentage the same
	// as a percentage; positions and open orders are liquidated as they reach 0
	LiquidationBufferAmount     Amount       `json:"liquidation_buffer_amount"`
	LiquidationBufferPercentage types.Number `json:"liquidation_buffer_percentage"`
	// IntradayMarginWindowMeasure and OvernightMarginWindowMeasure are the margin health in each window of an
	// account opted in to intraday margin
	IntradayMarginWindowMeasure  *MarginWindowMeasure `json:"intraday_margin_window_measure"`
	OvernightMarginWindowMeasure *MarginWindowMeasure `json:"overnight_margin_window_measure"`
	TotalPendingTransfersAmount  Amount               `json:"total_pending_transfers_amount"`
	// FundingPNL is the funding profit or loss of US perpetual style futures
	FundingPNL Amount `json:"funding_pnl"`
}

// MarginWindowMeasure is a US derivatives account's margin health in a margin window
type MarginWindowMeasure struct {
	// MarginWindowType is one of the FCMMarginWindowType values
	MarginWindowType string `json:"margin_window_type"`
	// MarginLevel is one of the MarginLevelType values
	MarginLevel        string       `json:"margin_level"`
	InitialMargin      types.Number `json:"initial_margin"`
	MaintenanceMargin  types.Number `json:"maintenance_margin"`
	LiquidationBuffer  types.Number `json:"liquidation_buffer"`
	TotalHold          types.Number `json:"total_hold"`
	FuturesBuyingPower types.Number `json:"futures_buying_power"`
}

// futuresPositionsResponse holds US derivatives positions
type futuresPositionsResponse struct {
	Positions []FuturesPosition `json:"positions"`
}

// futuresPositionResponse holds a US derivatives position
type futuresPositionResponse struct {
	Position *FuturesPosition `json:"position"`
}

// FuturesPosition is a US derivatives position
type FuturesPosition struct {
	ProductID      string    `json:"product_id"`
	ExpirationTime time.Time `json:"expiration_time"`
	// Side is one of the FuturesPositionSide values
	Side              string       `json:"side"`
	NumberOfContracts types.Number `json:"number_of_contracts"`
	CurrentPrice      types.Number `json:"current_price"`
	AverageEntryPrice types.Number `json:"avg_entry_price"`
	UnrealisedPNL     types.Number `json:"unrealized_pnl"`
	// DailyRealisedPNL is the profit or loss the product's trades realised on the current trade date
	DailyRealisedPNL types.Number `json:"daily_realized_pnl"`
}

// scheduleFuturesSweepBody is ScheduleFuturesSweep's body; an unset amount sweeps every available excess fund
type scheduleFuturesSweepBody struct {
	USDAmount string `json:"usd_amount,omitempty"`
}

// futuresSweepsResponse holds US derivatives sweeps
type futuresSweepsResponse struct {
	Sweeps []FuturesSweep `json:"sweeps"`
}

// FuturesSweep is a sweep of funds from the US derivatives futures account to the USD spot wallet
type FuturesSweep struct {
	ID              string `json:"id"`
	RequestedAmount Amount `json:"requested_amount"`
	// ShouldSweepAll is set for a sweep of every available fund
	ShouldSweepAll bool `json:"should_sweep_all"`
	// Status is one of the FuturesSweepStatus values
	Status string `json:"status"`
	// ScheduledTime is when the sweep was requested
	ScheduledTime time.Time `json:"scheduled_time"`
}

// intradayMarginSettingBody is the intraday margin setting SetIntradayMarginSetting sends and
// GetIntradayMarginSetting returns
type intradayMarginSettingBody struct {
	Setting string `json:"setting"`
}
