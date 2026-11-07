package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Bot results follow protobuf's JSON mapping: 64-bit integers such as bot IDs, timestamps and some counts arrive quoted,
// and enums arrive as their names (BOT_MODE_MIX, FUTURE_GRID_STATUS_COMPLETED) rather than the numbers requests take.
// Bot IDs have 18 digits, more than a float64 holds, so they stay strings, or a types.PreciseNumber where one can
// arrive as a bare number (CreateBotResponse)

// GetFuturesComboBotLimitRequest holds the futures combo bot settings to check against their accepted ranges before
// creating the bot with them
type GetFuturesComboBotLimitRequest struct {
	CreateFuturesComboBotRequest
	NeedToSlippage bool
}

// CreateFuturesComboBotRequest holds the settings of a new futures combo bot
type CreateFuturesComboBotRequest struct {
	// Leverage must be at least 1
	Leverage float64
	// InitialMargin is the initial investment in the quote currency
	InitialMargin float64
	// AdjustPositionMode is the rebalancing trigger: 1 time, 2 percentage, 3 time or percentage, 4 manual, 5 on
	// settings change, 6 on transfer
	AdjustPositionMode uint64
	SymbolSettings     []FuturesComboSymbolSettingRequest
	// AdjustPositionPercent is the allocation drift that triggers a rebalance, as a ratio from 0.01 to 0.5; modes that
	// rebalance on percentage need it
	AdjustPositionPercent float64
	// AdjustPositionTimeInterval is sent in whole seconds; modes that rebalance on time need it
	AdjustPositionTimeInterval time.Duration
	// SLPercent and TPPercent are ratios of the total margin, and TrailingStopPercent a ratio: 0.2 is 20%
	SLPercent           float64
	TPPercent           float64
	TrailingStopPercent float64
}

// FuturesComboSymbolSettingRequest holds one symbol of a futures combo bot's portfolio
type FuturesComboSymbolSettingRequest struct {
	Symbol string
	// TargetPositionPercent is the symbol's portfolio weight as a ratio; the weights of every symbol sum to 1
	TargetPositionPercent float64
	// Side is 1 long or 2 short
	Side uint64
}

// BotLimitRange is the accepted range of a bot parameter
type BotLimitRange struct {
	Min types.Number `json:"min"`
	Max types.Number `json:"max"`
}

// FuturesComboBotLimitResponse holds the accepted ranges of a futures combo bot's settings
type FuturesComboBotLimitResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode   uint64 `json:"status_code"`
	DebugMessage string `json:"debug_msg"`
	// CheckCode is LIMIT_CHECK_CODE_SUCCESS_UNSPECIFIED when every setting is in range, and otherwise names the setting
	// out of range
	CheckCode             string        `json:"check_code"`
	InitialMargin         BotLimitRange `json:"init_margin"`
	Leverage              BotLimitRange `json:"leverage"`
	AdjustPositionPercent BotLimitRange `json:"adjust_position_percent"`
	// AdjustPositionTimeInterval is in seconds
	AdjustPositionTimeInterval BotLimitRange `json:"adjust_position_time_interval"`
	SLPercent                  BotLimitRange `json:"sl_percent"`
	TPPercent                  BotLimitRange `json:"tp_percent"`
	TrailingStopPercent        BotLimitRange `json:"trailing_stop_percent"`
}

// CreateBotResponse holds the outcome of creating a futures combo, futures Martingale or DCA bot
type CreateBotResponse struct {
	// StatusCode is 0 or 200 on success and 421 when the user is banned
	StatusCode uint64 `json:"status_code"`
	// BotID decodes from a quoted or a bare integer: the futures pages quote it, while Create DCA Bot types it as an
	// integer. Its 18 digits need more precision than a float64 holds
	BotID types.PreciseNumber `json:"bot_id"`
	// BanReasonText is the localised ban reason, sent with status code 421
	BanReasonText string `json:"ban_reason_text"`
	DebugMessage  string `json:"debug_msg"`
}

// CloseBotResponse holds the outcome of closing a futures combo, futures Martingale or spot grid bot
type CloseBotResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode   uint64 `json:"status_code"`
	DebugMessage string `json:"debug_msg"`
}

// FuturesComboBotDetailResponse holds a futures combo bot's detail
type FuturesComboBotDetailResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode   uint64                `json:"status_code"`
	DebugMessage string                `json:"debug_msg"`
	Detail       FuturesComboBotDetail `json:"detail"`
}

// FuturesComboBotDetail holds a futures combo bot's settings, state and profit and loss
type FuturesComboBotDetail struct {
	BotID string `json:"bot_id"`
	// BotMode is an enum name such as BOT_MODE_LONG, BOT_MODE_SHORT or BOT_MODE_MIX
	BotMode string `json:"bot_mode"`
	// BotDisplayStatus is an enum name such as BOT_DISPLAY_STATUS_RUNNING
	BotDisplayStatus string `json:"bot_display_status"`
	// BotName lists the portfolio's symbols
	BotName  string       `json:"bot_name"`
	Leverage types.Number `json:"leverage"`
	// TotalMargin includes later top-ups
	TotalMargin types.Number `json:"total_margin"`
	Equity      types.Number `json:"equity"`
	TotalPNL    types.Number `json:"total_pnl"`
	// TotalPNLPercent, ROI and TotalAPR are ratios: 0.0505 is 5.05%
	TotalPNLPercent   types.Number `json:"total_pnl_per"`
	ROI               types.Number `json:"roi"`
	TotalAPR          types.Number `json:"total_apr"`
	RealisedPNL       types.Number `json:"realized_pnl"`
	UnrealisedPNL     types.Number `json:"unrealized_pnl"`
	TotalExecutionFee types.Number `json:"total_exec_fee"`
	// FundingFee is positive when received and negative when paid
	FundingFee            types.Number                `json:"funding_fee"`
	TotalPositionValue    types.Number                `json:"total_position_value"`
	MarginBalance         types.Number                `json:"margin_balance"`
	AvailableBalance      types.Number                `json:"available_balance"`
	InitialMarginRate     types.Number                `json:"imr"`
	MaintenanceMarginRate types.Number                `json:"mmr"`
	SymbolSettings        []FuturesComboSymbolSetting `json:"symbol_settings"`
	// AdjustPositionMode is an enum name such as ADJUST_POSITION_MODE_TIME_OR_PERCENT
	AdjustPositionMode    string       `json:"adjust_position_mode"`
	AdjustPositionPercent types.Number `json:"adjust_position_percent"`
	// AdjustPositionTimeInterval is in seconds
	AdjustPositionTimeInterval uint64       `json:"adjust_position_time_interval"`
	AdjustedPositionNumber     types.Number `json:"adjusted_position_num"`
	LastAdjustPositionTime     types.Time   `json:"last_adjust_position_time"`
	SLPercent                  types.Number `json:"sl_percent"`
	TPPercent                  types.Number `json:"tp_percent"`
	TrailingStopPercent        types.Number `json:"trailing_stop_percent"`
	// StopType is an enum name such as BOT_STOP_TYPE_BY_USER
	StopType string `json:"stop_type"`
	// CloseCode is an enum name such as BOT_CLOSE_CODE_CANCELED_MANUALLY
	CloseCode      string       `json:"close_code"`
	ExitEquity     types.Number `json:"exit_equity"`
	SLExitEquity   types.Number `json:"sl_exit_equity"`
	TPExitEquity   types.Number `json:"tp_exit_equity"`
	MidwayTransfer types.Number `json:"midway_transfer"`
	// SettlementAssets describes the assets settled on close, such as "496.24566099 USDT"
	SettlementAssets string       `json:"settlement_assets"`
	FollowNumber     uint64       `json:"follow_num"`
	UsedRewardAmount types.Number `json:"used_reward_amount"`
	// UsedRewardID is the ID of the first insurance voucher used
	UsedRewardID string       `json:"used_reward_id"`
	TotalBonus   types.Number `json:"total_bonus"`
	// RunTimeDuration is in seconds
	RunTimeDuration       types.Number `json:"run_time_duration"`
	CreateTime            types.Time   `json:"create_time"`
	EndTime               types.Time   `json:"end_time"`
	LastModifySettingTime types.Time   `json:"last_modify_setting_time"`
	WebsocketToken        string       `json:"ws_token"`
}

// FuturesComboSymbolSetting holds one symbol of a futures combo bot's portfolio
type FuturesComboSymbolSetting struct {
	Symbol                string        `json:"symbol"`
	BaseToken             currency.Code `json:"base_token"`
	QuoteToken            currency.Code `json:"quote_token"`
	TargetPositionPercent types.Number  `json:"target_position_percent"`
	// Side is an enum name: SIDE_LONG or SIDE_SHORT
	Side     string `json:"side"`
	SymbolID uint64 `json:"symbol_id"`
}

// ValidateFuturesGridInputRequest holds the futures grid bot settings to check against their accepted ranges
type ValidateFuturesGridInputRequest struct {
	Symbol string
	// CellNumber is the number of grid levels, at least 2
	CellNumber uint64
	MinPrice   float64
	MaxPrice   float64
	// Leverage must be at least 1
	Leverage float64
	// GridType is 1 arithmetic or 2 geometric
	GridType uint64
	// GridMode is 1 neutral, 2 long or 3 short
	GridMode        uint64
	StopLossPrice   float64
	TakeProfitPrice float64
	// TPSLType picks how take-profit and stop-loss trigger: 1 both by percentage, 2 both by price, 3 take-profit by
	// price and stop-loss by percentage, 4 take-profit by percentage and stop-loss by price
	TPSLType uint64
	// EntryPrice delays activation until the price reaches it
	EntryPrice float64
	// StopLossPercent and TakeProfitPercent are percentages, which this page's docs scale as 10 for 10% but Create
	// Grid Bot's docs as 0.1 for 10%
	StopLossPercent     float64
	TakeProfitPercent   float64
	TrailingStopPercent float64
	InitialMargin       float64
	MoveUpPrice         float64
	MoveDownPrice       float64
}

// GridInputRange is the accepted range of a grid bot parameter
type GridInputRange struct {
	From types.Number `json:"from"`
	To   types.Number `json:"to"`
}

// ValidateFuturesGridInputResponse holds the accepted ranges of a futures grid bot's settings
type ValidateFuturesGridInputResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode   uint64 `json:"status_code"`
	DebugMessage string `json:"debug_msg"`
	// CheckCode is FGRID_CHECK_CODE_SUCCESS or FGRID_CHECK_CODE_UNSPECIFIED when every setting is in range, and
	// otherwise names the setting out of range
	CheckCode             string         `json:"check_code"`
	Investment            GridInputRange `json:"investment"`
	Profit                GridInputRange `json:"profit"`
	CellNumber            GridInputRange `json:"cell_number"`
	MinPrice              GridInputRange `json:"min_price"`
	MaxPrice              GridInputRange `json:"max_price"`
	Leverage              GridInputRange `json:"leverage"`
	StopLoss              GridInputRange `json:"stop_loss"`
	TakeProfit            GridInputRange `json:"take_profit"`
	TakeProfitPrice       GridInputRange `json:"take_profit_price"`
	StopLossPrice         GridInputRange `json:"stop_loss_price"`
	EntryPrice            GridInputRange `json:"entry_price"`
	TrailingStopPercent   GridInputRange `json:"trailing_stop_per"`
	LongLiquidationPrice  types.Number   `json:"long_liq_price"`
	ShortLiquidationPrice types.Number   `json:"short_liq_price"`
	MoveUpPrice           GridInputRange `json:"move_up_price"`
	MoveDownPrice         GridInputRange `json:"move_down_price"`
	// CellGTV0Percent is the extra grid count VIP users may set, as a percentage of the non-VIP limit
	CellGTV0Percent types.Number `json:"cell_gtv0_per"`
}

// CreateFuturesGridBotRequest holds the settings of a new futures grid bot
type CreateFuturesGridBotRequest struct {
	Symbol string
	// GridMode is 1 neutral, 2 long or 3 short
	GridMode uint64
	MinPrice float64
	MaxPrice float64
	// CellNumber is the number of grid levels, at least 2
	CellNumber uint64
	// Leverage must be at least 1
	Leverage float64
	// GridType is 1 arithmetic or 2 geometric
	GridType uint64
	// TotalInvestment is the initial investment in the quote currency
	TotalInvestment float64
	// TakeProfitPercent and StopLossPercent are ratios: 0.2 is 20%
	TakeProfitPercent float64
	StopLossPercent   float64
	TakeProfitPrice   float64
	StopLossPrice     float64
	// TPSLType picks how take-profit and stop-loss trigger: 1 both by percentage, 2 both by price, 3 take-profit by
	// price and stop-loss by percentage, 4 take-profit by percentage and stop-loss by price
	TPSLType uint64
	// EntryPrice delays activation until the price reaches it
	EntryPrice float64
	// TrailingStopPercent is a ratio: 0.05 is 5%
	TrailingStopPercent float64
	// MoveUpPrice and MoveDownPrice shift the grid; geometric grids do not take them
	MoveUpPrice   float64
	MoveDownPrice float64
}

// CreateFuturesGridBotResponse holds the outcome of creating a futures grid bot
type CreateFuturesGridBotResponse struct {
	// StatusCode is 0 or 200 on success and 421 when the user is banned
	StatusCode uint64 `json:"status_code"`
	BotID      string `json:"bot_id"`
	// CheckCode names a setting out of range, as Validate Grid Input does
	CheckCode string `json:"check_code"`
	// BanReasonText is the localised ban reason, sent with status code 421
	BanReasonText string `json:"ban_reason_text"`
	DebugMessage  string `json:"debug_msg"`
}

// CloseFuturesGridBotResponse holds the outcome of closing a futures grid bot
type CloseFuturesGridBotResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode   uint64 `json:"status_code"`
	BotID        string `json:"bot_id"`
	DebugMessage string `json:"debug_msg"`
}

// FuturesGridBotDetailResponse holds a futures grid bot's detail
type FuturesGridBotDetailResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode   uint64               `json:"status_code"`
	DebugMessage string               `json:"debug_msg"`
	Detail       FuturesGridBotDetail `json:"detail"`
}

// FuturesGridBotDetail holds a futures grid bot's settings, state and profit and loss
type FuturesGridBotDetail struct {
	BotID      string        `json:"bot_id"`
	Symbol     string        `json:"symbol"`
	BaseToken  currency.Code `json:"base_token"`
	QuoteToken currency.Code `json:"quote_token"`
	// Status is an enum name such as FUTURE_GRID_STATUS_RUNNING
	Status string `json:"status"`
	// GridMode is an enum name such as FUTURE_GRID_MODE_LONG
	GridMode string `json:"grid_mode"`
	// GridType is an enum name such as FUTURE_GRID_TYPE_GEOMETRIC
	GridType   string       `json:"grid_type"`
	MinPrice   types.Number `json:"min_price"`
	MaxPrice   types.Number `json:"max_price"`
	CellNumber uint64       `json:"cell_number"`
	// Leverage is the initial leverage and RealLeverage the effective leverage
	Leverage     types.Number `json:"leverage"`
	RealLeverage types.Number `json:"real_leverage"`
	// TotalInvestment includes later top-ups
	TotalInvestment types.Number `json:"total_investment"`
	TotalValue      types.Number `json:"total_value"`
	// CurrentPosition is in contracts
	CurrentPosition types.Number `json:"current_position"`
	PNL             types.Number `json:"pnl"`
	// PNLPercent, GridAPR and TotalAPR are ratios: 0.0505 is 5.05%
	PNLPercent    types.Number `json:"pnl_per"`
	GridProfit    types.Number `json:"grid_profit"`
	GridAPR       types.Number `json:"grid_apr"`
	TotalAPR      types.Number `json:"total_apr"`
	RealisedPNL   types.Number `json:"realised_pnl"`
	UnrealisedPNL types.Number `json:"unrealised_pnl"`
	// FundingFee is positive when received and negative when paid
	FundingFee        types.Number `json:"funding_fee"`
	PositionBalance   types.Number `json:"position_balance"`
	AvailableBalance  types.Number `json:"available_balance"`
	TotalOrderBalance types.Number `json:"total_order_balance"`
	Equity            types.Number `json:"equity"`
	LastPrice         types.Number `json:"last_price"`
	MarkPrice         types.Number `json:"mark_price"`
	LiquidationPrice  types.Number `json:"liquidation_price"`
	StopLossPercent   types.Number `json:"stop_loss_per"`
	TakeProfitPercent types.Number `json:"take_profit_per"`
	StopLossPrice     types.Number `json:"stop_loss_price"`
	TakeProfitPrice   types.Number `json:"take_profit_price"`
	// TPSLType is an enum name such as TP_SL_TYPE_TP_PERCENT_SL_PRICE
	TPSLType   string       `json:"tp_sl_type"`
	EntryPrice types.Number `json:"entry_price"`
	// RealClosePrice is the price when the bot stopped
	RealClosePrice         types.Number `json:"real_close_price"`
	MinPricePrecision      types.Number `json:"min_price_precision"`
	TickSize               types.Number `json:"tick_size"`
	TrailingStopPercent    types.Number `json:"trailing_stop_per"`
	TrailingStopExitEquity types.Number `json:"trailing_stop_exit_equity"`
	// CloseReason is an enum name such as F_GRID_BOT_STOP_TYPE_TRIGGER_FBU_FAIL
	CloseReason string `json:"close_reason"`
	// BotCloseCode is an enum name such as BOT_CLOSE_CODE_FAILED_INITIATION
	BotCloseCode string `json:"bot_close_code"`
	// FuturesPositionSide is an enum name such as FUTURES_POSITION_SIDE_LONG
	FuturesPositionSide string `json:"futures_pos_side"`
	// ArbitrageNumber counts grid fills
	ArbitrageNumber       uint64       `json:"arbitrage_num"`
	ArbitrageNumber24Hour uint64       `json:"arbitrage_num_24"`
	MinProfit             types.Number `json:"min_profit"`
	MaxProfit             types.Number `json:"max_profit"`
	MoveUpPrice           types.Number `json:"move_up_price"`
	MoveDownPrice         types.Number `json:"move_down_price"`
	CurrentMinPrice       types.Number `json:"curr_min_price"`
	CurrentMaxPrice       types.Number `json:"curr_max_price"`
	InitialMarginRate     types.Number `json:"i_m_rate"`
	MaintenanceMarginRate types.Number `json:"m_m_rate"`
	// AccountType is an enum name such as BOT_ACCOUNT_TYPE_FUND
	AccountType string `json:"account_type"`
	// AllowFollow reports whether copy trading is allowed
	AllowFollow             types.Boolean `json:"allow_follow"`
	FollowNumber            uint64        `json:"follow_num"`
	CopyTradeIsMaster       bool          `json:"copy_trade_is_master"`
	CopyTradeFollowerNumber types.Number  `json:"copy_trade_follower_num"`
	UsedRewardAmount        types.Number  `json:"used_reward_amount"`
	// UsedRewardID is the ID of the first insurance voucher used
	UsedRewardID    string       `json:"used_reward_id"`
	InitialBonus    types.Number `json:"init_bonus"`
	UsedBonusAmount types.Number `json:"used_bonus_amount"`
	// SettlementAssets describes the assets settled on close, such as "950 USDT"
	SettlementAssets         string       `json:"settlement_assets"`
	CumulativeWithdrewAmount types.Number `json:"cum_withdrew_amount"`
	CurrentProfit            types.Number `json:"current_profit"`
	CurrentPercent           types.Number `json:"current_per"`
	TakerFeeRate             types.Number `json:"taker_fee_rate"`
	MakerFeeRate             types.Number `json:"maker_fee_rate"`
	// OperationTime is the milliseconds since the bot was created
	OperationTime    types.Number `json:"operation_time"`
	CreateTime       types.Time   `json:"create_time"`
	ModifyTime       types.Time   `json:"modify_time"`
	EndTime          types.Time   `json:"end_time"`
	ADLRankIndicator uint64       `json:"adl_rank_indicator"`
}

// GetFuturesMartingaleBotLimitRequest holds the futures Martingale bot settings to check against their accepted ranges
type GetFuturesMartingaleBotLimitRequest struct {
	Symbol string
	// MartingaleMode is 1 long or 2 short
	MartingaleMode uint64
	// Leverage must be at least 1
	Leverage float64
	// PriceFloatPercent is the adverse price move that adds to the position, as a ratio: 0.015 is 1.5%
	PriceFloatPercent float64
	// AddPositionPercent scales each add by the base position: 1 is 1x
	AddPositionPercent float64
	// AddPositionNumber is the most adds per round
	AddPositionNumber uint64
	InitialMargin     float64
	// RoundTPPercent and SLPercent are ratios: 0.03 is 3%
	RoundTPPercent float64
	SLPercent      float64
	EntryPrice     float64
	NeedToSlippage bool
}

// FuturesMartingaleBotLimitResponse holds the accepted ranges of a futures Martingale bot's settings
type FuturesMartingaleBotLimitResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode   uint64 `json:"status_code"`
	DebugMessage string `json:"debug_msg"`
	// CheckCode is F_MART_LIMIT_CHECK_CODE_F_MART_CHECK_CODE_SUCCESS_UNSPECIFIED when every setting is in range, and
	// otherwise names the setting out of range
	CheckCode          string        `json:"check_code"`
	PriceFloatPercent  BotLimitRange `json:"price_float_percent"`
	AddPositionPercent BotLimitRange `json:"add_position_percent"`
	AddPositionNumber  BotLimitRange `json:"add_position_num"`
	InitialMargin      BotLimitRange `json:"init_margin"`
	RoundTPPercent     BotLimitRange `json:"round_tp_percent"`
	SLPercent          BotLimitRange `json:"sl_percent"`
	EntryPrice         BotLimitRange `json:"entry_price"`
	Leverage           BotLimitRange `json:"leverage"`
}

// CreateFuturesMartingaleBotRequest holds the settings of a new futures Martingale bot
type CreateFuturesMartingaleBotRequest struct {
	Symbol string
	// MartingaleMode is 1 long, buying dips, or 2 short, selling rallies
	MartingaleMode uint64
	// Leverage must be at least 1
	Leverage float64
	// PriceFloatPercent is the adverse price move that adds to the position, as a ratio: 0.015 is 1.5%
	PriceFloatPercent float64
	// AddPositionPercent scales each add by the base position: 1.1 is 1.1x
	AddPositionPercent float64
	// AddPositionNumber is the most adds per round
	AddPositionNumber uint64
	// InitialMargin is the initial investment in the quote currency
	InitialMargin float64
	// RoundTPPercent is a round's take-profit as a ratio: 0.03 is 3%
	RoundTPPercent float64
	// AutoCycleToggle is 1 to start a new round after each take-profit or 2 to stop after one
	AutoCycleToggle uint64
	// SLPercent is a ratio of the total margin: 0.2 is 20%
	SLPercent  float64
	EntryPrice float64
}

// FuturesMartingaleBotDetailResponse holds a futures Martingale bot's detail
type FuturesMartingaleBotDetailResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode              uint64                     `json:"status_code"`
	DebugMessage            string                     `json:"debug_msg"`
	FuturesMartingaleDetail FuturesMartingaleBotDetail `json:"fmart_detail"`
}

// FuturesMartingaleBotDetail holds a futures Martingale bot's settings, round progress and profit and loss
type FuturesMartingaleBotDetail struct {
	BotID      string        `json:"bot_id"`
	Symbol     string        `json:"symbol"`
	BaseToken  currency.Code `json:"base_token"`
	QuoteToken currency.Code `json:"quote_token"`
	// BotDisplayStatus is an enum name such as F_MART_BOT_DISPLAY_STATUS_RUNNING
	BotDisplayStatus string `json:"bot_display_status"`
	// FuturesMartingaleMode is an enum name such as F_MART_MODE_MARTINGALE_MODE_LONG
	FuturesMartingaleMode string       `json:"fmart_mode"`
	Leverage              types.Number `json:"leverage"`
	ActualLeverage        types.Number `json:"actual_leverage"`
	// SingleRoundTargetProfitPercent, PriceFloatPercent, SLPercent, PNLPercent and TotalAPR are ratios: 0.03 is 3%
	SingleRoundTargetProfitPercent types.Number `json:"single_round_target_profit_per"`
	AddPositionNumber              types.Number `json:"add_pos_num"`
	AddPositionPercent             types.Number `json:"add_pos_per"`
	PriceFloatPercent              types.Number `json:"price_float_per"`
	EntryPrice                     types.Number `json:"entry_price"`
	SLPercent                      types.Number `json:"sl_per"`
	// TotalMargin includes later top-ups
	TotalMargin   types.Number `json:"total_margin"`
	PNL           types.Number `json:"pnl"`
	PNLPercent    types.Number `json:"pnl_per"`
	RealisedPNL   types.Number `json:"realized_pnl"`
	UnrealisedPNL types.Number `json:"unrealized_pnl"`
	// NetFundingFee is positive when received and negative when paid
	NetFundingFee              types.Number `json:"net_funding_fee"`
	TotalAPR                   types.Number `json:"total_apr"`
	CompletedRounds            uint64       `json:"completed_rounds"`
	CurrentRound               uint64       `json:"current_round"`
	CurrentAddedPositionNumber uint64       `json:"current_added_pos_num"`
	// PositionSize is in contracts
	PositionSize              types.Number `json:"pos_size"`
	AveragePositionPrice      types.Number `json:"avg_pos_price"`
	PositionBalance           types.Number `json:"pos_balance"`
	OrderBalance              types.Number `json:"order_balance"`
	AvailableBalance          types.Number `json:"available_balance"`
	MarkPrice                 types.Number `json:"mark_price"`
	CurrentPrice              types.Number `json:"current_price"`
	EstimatedLiquidationPrice types.Number `json:"estimated_liq_price"`
	// AutoCycle is an enum name such as AUTO_CYCLE_TOGGLE_AUTO_CYCLE_TOGGLE_ENABLED
	AutoCycle string `json:"auto_cycle"`
	// StopType is an enum name such as F_MART_BOT_STOP_TYPE_STOP_TYPE_BY_USER
	StopType string `json:"stop_type"`
	// CloseCode is an enum name such as BOT_CLOSE_CODE_CANCELED_MANUALLY
	CloseCode  string       `json:"close_code"`
	ClosePrice types.Number `json:"close_price"`
	TickSize   types.Number `json:"tick_size"`
	// SettlementAssets describes the assets settled on close
	SettlementAssets string       `json:"settlement_assets"`
	FollowNumber     uint64       `json:"follow_num"`
	UsedRewardAmount types.Number `json:"used_reward_amount"`
	// UsedRewardID is the ID of the first insurance voucher used
	UsedRewardID string       `json:"used_reward_id"`
	TotalBonus   types.Number `json:"total_bonus"`
	// OperationTime is the seconds since the bot was created
	OperationTime types.Number `json:"operation_time"`
	CreateTime    types.Time   `json:"create_time"`
	ModifyTime    types.Time   `json:"modify_time"`
	EndTime       types.Time   `json:"end_time"`
	// RunningDuration is in seconds
	RunningDuration  types.Number `json:"running_duration"`
	ADLRankIndicator uint64       `json:"adl_rank_indicator"`
}

// ValidateSpotGridInputRequest holds the spot grid bot settings to check against their accepted ranges
type ValidateSpotGridInputRequest struct {
	Symbol string
	// CellNumber is the number of grid intervals, at least 2
	CellNumber uint64
	MinPrice   float64
	MaxPrice   float64
	// InvestMode is 0 quote only, the default, 1 base only or 2 base and quote; BaseInvestment and QuoteInvestment
	// follow it
	InvestMode      uint64
	BaseInvestment  float64
	QuoteInvestment float64
	// StopLoss, TakeProfit and EntryPrice are prices
	StopLoss   float64
	TakeProfit float64
	EntryPrice float64
	// TrailingStopPercent is a ratio from 0 to 0.99: 0.05 is 5%
	TrailingStopPercent float64
	// EnableTrailing needs CellNumber of at least 5
	EnableTrailing bool
	LimitUpPrice   float64
}

// ValidateSpotGridInputResponse holds the accepted ranges of a spot grid bot's settings
type ValidateSpotGridInputResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode   uint64 `json:"status_code"`
	DebugMessage string `json:"debug_msg"`
	// CheckCode is SPOT_CHECK_CODE_SUCCESS_UNSPECIFIED when every setting is in range, and otherwise names the setting
	// out of range
	CheckCode  string         `json:"check_code"`
	Investment GridInputRange `json:"investment"`
	// Profit is the per-grid profit rate as a ratio
	Profit         GridInputRange `json:"profit"`
	CellNumber     GridInputRange `json:"cell_number"`
	MinPrice       GridInputRange `json:"min_price"`
	MaxPrice       GridInputRange `json:"max_price"`
	StopLoss       GridInputRange `json:"stop_loss"`
	TakeProfit     GridInputRange `json:"take_profit"`
	EntryPrice     GridInputRange `json:"entry_price"`
	BaseInvestment GridInputRange `json:"base_investment"`
	// CellDistance is the price distance between grid levels in the quote currency
	CellDistance        types.Number   `json:"cell_distance"`
	TrailingStopPercent GridInputRange `json:"ts_percent"`
	LimitUpPrice        GridInputRange `json:"limit_up_price"`
}

// CreateSpotGridBotRequest holds the settings of a new spot grid bot
type CreateSpotGridBotRequest struct {
	Symbol   string
	MaxPrice float64
	MinPrice float64
	// CellNumber is the number of grid intervals, at least 2
	CellNumber uint64
	// InvestMode is 0 quote only, the default, 1 base only or 2 base and quote; BaseInvestment and QuoteInvestment
	// follow it
	InvestMode      uint64
	BaseInvestment  float64
	QuoteInvestment float64
	// EntryPrice delays activation until the price reaches it
	EntryPrice      float64
	StopLossPrice   float64
	TakeProfitPrice float64
	// TrailingStopPercent is a ratio from 0 to 0.99: 0.05 is 5%
	TrailingStopPercent float64
	// EnableTrailing shifts the grid as the price moves and needs CellNumber of at least 5
	EnableTrailing bool
	// LimitUpPrice caps how far a trailing grid shifts up
	LimitUpPrice float64
}

// CreateSpotGridBotResponse holds the outcome of creating a spot grid bot
type CreateSpotGridBotResponse struct {
	// StatusCode is 0 or 200 on success and 421 when the user is banned
	StatusCode uint64 `json:"status_code"`
	GridID     string `json:"grid_id"`
	// BanReasonText is the localised ban reason, sent with status code 421
	BanReasonText string `json:"ban_reason_text"`
	DebugMessage  string `json:"debug_msg"`
}

// SpotGridBotDetailResponse holds a spot grid bot's detail
type SpotGridBotDetailResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode   uint64            `json:"status_code"`
	DebugMessage string            `json:"debug_msg"`
	Detail       SpotGridBotDetail `json:"detail"`
}

// SpotGridBotDetail holds a spot grid bot's settings, state and profit
type SpotGridBotDetail struct {
	GridID     string        `json:"grid_id"`
	Symbol     string        `json:"symbol"`
	BaseToken  currency.Code `json:"base_token"`
	QuoteToken currency.Code `json:"quote_token"`
	// Status is RAW, REJECTED, NEW, INITIALIZING, RUNNING, CANCELLING or COMPLETED
	Status   string       `json:"status"`
	MaxPrice types.Number `json:"max_price"`
	MinPrice types.Number `json:"min_price"`
	// CellNumber is the number of active grid intervals
	CellNumber uint64 `json:"cell_number"`
	// TotalInvestment, TotalProfit and GridProfit are in the quote currency
	TotalInvestment types.Number `json:"total_investment"`
	TotalProfit     types.Number `json:"total_profit"`
	GridProfit      types.Number `json:"grid_profit"`
	// GridAPR and TotalAPR are ratios: 0.15 is 15%
	GridAPR               types.Number `json:"grid_apr"`
	TotalAPR              types.Number `json:"total_apr"`
	ArbitrageNumber       uint64       `json:"arbitrage_num"`
	ArbitrageNumber24Hour uint64       `json:"arbitrage_num_24"`
	CurrentPrice          types.Number `json:"current_price"`
	// CurrentProfit and CurrentPercent are only set while the bot runs
	CurrentProfit   types.Number `json:"current_profit"`
	CurrentPercent  types.Number `json:"current_per"`
	Equity          types.Number `json:"equity"`
	EntryPrice      types.Number `json:"entry_price"`
	StopLossPrice   types.Number `json:"stop_loss_price"`
	TakeProfitPrice types.Number `json:"take_profit_price"`
	// CloseReason is an enum name such as CLOSED_MANUALLY
	CloseReason string `json:"close_reason"`
	// BotCloseCode is an enum name such as BOT_CLOSE_CODE_CANCELED_MANUALLY
	BotCloseCode string `json:"bot_close_code"`
	// TrailingStopPercent is a ratio: 0.05 is 5%
	TrailingStopPercent   types.Number `json:"ts_percent"`
	IsSupportTrailingStop bool         `json:"is_support_ts"`
	EnableTrailing        bool         `json:"enable_trailing"`
	LimitUpPrice          types.Number `json:"limit_up_price"`
	// OriginalMaxPrice, OriginalMinPrice and OriginalCellNumber are the grid before trailing shifted it
	OriginalMaxPrice   types.Number `json:"ori_max_price"`
	OriginalMinPrice   types.Number `json:"ori_min_price"`
	OriginalCellNumber uint64       `json:"ori_cell_number"`
	// AllowFollow reports whether copy trading is allowed
	AllowFollow  types.Boolean `json:"allow_follow"`
	FollowNumber uint64        `json:"follow_num"`
	// OperationTime is the milliseconds since the bot was created
	OperationTime    types.Number `json:"operation_time"`
	CreateTime       types.Time   `json:"create_time"`
	EndTime          types.Time   `json:"end_time"`
	UsedRewardAmount types.Number `json:"used_reward_amount"`
	// SettlementAssets describes the assets settled on close
	SettlementAssets string `json:"settlement_assets"`
	// AccountType is an enum name such as BOT_ACCOUNT_TYPE_FUND
	AccountType              string       `json:"account_type"`
	CumulativeWithdrewAmount types.Number `json:"cum_withdrew_amount"`
	// SpotSymbolStatus, UsedRewardID, TakerFeeRate, MakerFeeRate and TrailingStopExitEquity are missing from the
	// documented fields but present in the documented response
	SpotSymbolStatus       string       `json:"spot_symbol_status"`
	UsedRewardID           string       `json:"used_reward_id"`
	TakerFeeRate           types.Number `json:"taker_fee_rate"`
	MakerFeeRate           types.Number `json:"maker_fee_rate"`
	TrailingStopExitEquity types.Number `json:"ts_exit_equity"`
}

// CreateDCABotRequest holds the settings of a new DCA bot, which buys a fixed amount of each pair at every interval
type CreateDCABotRequest struct {
	// Frequency is sent in whole seconds, at least 10
	Frequency time.Duration
	// QuoteCoin pays for the purchases, such as USDT or USDC
	QuoteCoin currency.Code
	// Pairs holds up to 5 pairs
	Pairs []DCABotPairRequest
	// MaxInvestAmount closes the bot once its total spend reaches it
	MaxInvestAmount float64
}

// DCABotPairRequest holds a coin a DCA bot buys and the quote amount it spends each round
type DCABotPairRequest struct {
	Base   currency.Code
	Amount float64
}

// CloseDCABotResponse holds the outcome of closing a DCA bot
type CloseDCABotResponse struct {
	// StatusCode is 0 or 200 on success
	StatusCode uint64 `json:"status_code"`
	BotID      string `json:"bot_id"`
	// CloseMode is the settlement mode's enum name: DCA_BASE_MODE or DCA_QUOTE_MODE
	CloseMode    string `json:"close_mode"`
	DebugMessage string `json:"debug_msg"`
}
