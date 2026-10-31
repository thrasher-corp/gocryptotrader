package cryptodotcom

import (
	"fmt"
	"math"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Trading bot types
const (
	TradingBotTypeDCA              = "DCA"
	TradingBotTypeTWAP             = "TWAP"
	TradingBotTypeGrid             = "GRID"
	TradingBotTypeFundingArbitrage = "FUNDING_ARBITRAGE"
)

// Trading bot states: a paused bot places no orders until it is resumed, and a terminated one none again
const (
	TradingBotStateRunning    = "RUNNING"
	TradingBotStatePaused     = "PAUSED"
	TradingBotStateTerminated = "TERMINATED"
)

// DCA bot investment frequency modes: orders at a fixed interval, or monthly
const (
	TradingBotInvestmentFrequencyModeFixedDuration = "FIXED_DURATION"
	TradingBotInvestmentFrequencyModeMonthly       = "MONTHLY"
)

// Grid bot modes: grid levels at equal price differences, or at equal price ratios
const (
	TradingBotGridModeArithmetic = "ARITHMETIC"
	TradingBotGridModeGeometric  = "GEOMETRIC"
)

// TradingBotSettings is a bot type's settings, which CreateTradingBot takes and GetTradingBots reports:
// *TradingBotDCASettings, *TradingBotTWAPSettings, *TradingBotGridSettings or *TradingBotFundingArbitrageSettings
type TradingBotSettings interface {
	// BotType returns the type of bot the settings are for
	BotType() string
	validateSettings() error
}

// TradingBotUpdateSettings is the settings UpdateTradingBot can change, for a DCA bot *TradingBotDCAUpdateSettings and
// for a grid bot *TradingBotGridUpdateSettings; TWAP and funding arbitrage bots cannot change theirs
type TradingBotUpdateSettings interface {
	// BotType returns the type of bot the settings are for
	BotType() string
	updateSettingsParams() (any, error)
}

// TradingBotAllocation is the percentage of a DCA bot's investment that goes to an instrument; a bot's allocations add
// up to 100
type TradingBotAllocation struct {
	InstrumentName      string       `json:"instrument_name"`
	AllocatedPercentage types.Number `json:"allocated_percentage,omitempty"`
}

// TradingBotDCASettings is a DCA bot's settings: it buys or sells one instrument, or a basket of them by allocation, at
// an interval. A bot that is not a basket requires Side, OrderType and AutoStake, and a limit order Price; Quantity,
// per order, or TotalInvestment is always required. InvestmentFrequency, the interval between orders, is required in
// the fixed duration mode, and InvestmentDuration counts months in the monthly mode and milliseconds in the fixed
// duration one. Basket bots do not support stop or take profit prices
type TradingBotDCASettings struct {
	IsBasket                bool
	Allocations             []TradingBotAllocation
	InvestmentCurrency      currency.Code
	Side                    string
	OrderType               string
	AutoStake               bool
	Price                   types.Number
	Quantity                types.Number
	TotalInvestment         types.Number
	InvestmentFrequencyMode string
	InvestmentFrequency     time.Duration
	FirstInvestmentTime     time.Time
	InvestmentDuration      uint64
	MaxOrders               uint64
	StopPrice               types.Number
	TakeProfitPrice         types.Number
	SimplifiedMode          bool
}

// tradingBotDCASettingsWire is a DCA bot's settings as Crypto.com takes and reports them, the first investment time and
// the frequency in milliseconds
type tradingBotDCASettingsWire struct {
	IsBasket                        bool                   `json:"is_basket"`
	Allocations                     []TradingBotAllocation `json:"allocations"`
	InvestmentCurrency              currency.Code          `json:"investment_currency"`
	Side                            string                 `json:"side,omitempty"`
	OrderType                       string                 `json:"type,omitempty"`
	AutoStake                       bool                   `json:"auto_stake"`
	Price                           types.Number           `json:"price,omitempty"`
	Quantity                        types.Number           `json:"quantity,omitempty"`
	TotalInvestment                 types.Number           `json:"total_investment,omitempty"`
	InvestmentFrequencyMode         string                 `json:"investment_frequency_mode"`
	InvestmentFrequencyMilliseconds uint64                 `json:"investment_frequency,omitempty"`
	FirstInvestmentTimeMilliseconds uint64                 `json:"first_investment_time,omitempty"`
	InvestmentDuration              uint64                 `json:"investment_duration,omitempty"`
	MaxOrders                       uint64                 `json:"max_order,omitempty,string"`
	StopPrice                       types.Number           `json:"stop_price,omitempty"`
	TakeProfitPrice                 types.Number           `json:"take_profit_price,omitempty"`
	SimplifiedMode                  bool                   `json:"simplified_mode,omitempty"`
}

// BotType returns TradingBotTypeDCA
func (*TradingBotDCASettings) BotType() string { return TradingBotTypeDCA }

// MarshalJSON encodes the settings as Crypto.com takes them, the first investment time and the frequency in
// milliseconds
func (s *TradingBotDCASettings) MarshalJSON() ([]byte, error) {
	firstInvestmentTime, err := unixMilliParam(s.FirstInvestmentTime)
	if err != nil {
		return nil, err
	}
	frequency, err := investmentFrequencyParam(s.InvestmentFrequency)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&tradingBotDCASettingsWire{
		IsBasket:                        s.IsBasket,
		Allocations:                     s.Allocations,
		InvestmentCurrency:              s.InvestmentCurrency,
		Side:                            s.Side,
		OrderType:                       s.OrderType,
		AutoStake:                       s.AutoStake,
		Price:                           s.Price,
		Quantity:                        s.Quantity,
		TotalInvestment:                 s.TotalInvestment,
		InvestmentFrequencyMode:         s.InvestmentFrequencyMode,
		InvestmentFrequencyMilliseconds: frequency,
		FirstInvestmentTimeMilliseconds: firstInvestmentTime,
		InvestmentDuration:              s.InvestmentDuration,
		MaxOrders:                       s.MaxOrders,
		StopPrice:                       s.StopPrice,
		TakeProfitPrice:                 s.TakeProfitPrice,
		SimplifiedMode:                  s.SimplifiedMode,
	})
}

// UnmarshalJSON decodes the settings as Crypto.com reports them
func (s *TradingBotDCASettings) UnmarshalJSON(data []byte) error {
	var w tradingBotDCASettingsWire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	*s = TradingBotDCASettings{
		IsBasket:                w.IsBasket,
		Allocations:             w.Allocations,
		InvestmentCurrency:      w.InvestmentCurrency,
		Side:                    w.Side,
		OrderType:               w.OrderType,
		AutoStake:               w.AutoStake,
		Price:                   w.Price,
		Quantity:                w.Quantity,
		TotalInvestment:         w.TotalInvestment,
		InvestmentFrequencyMode: w.InvestmentFrequencyMode,
		InvestmentFrequency:     millisecondsDuration(w.InvestmentFrequencyMilliseconds),
		FirstInvestmentTime:     unixMilliTime(w.FirstInvestmentTimeMilliseconds),
		InvestmentDuration:      w.InvestmentDuration,
		MaxOrders:               w.MaxOrders,
		StopPrice:               w.StopPrice,
		TakeProfitPrice:         w.TakeProfitPrice,
		SimplifiedMode:          w.SimplifiedMode,
	}
	return nil
}

// TradingBotTWAPSettings is a TWAP bot's settings: it trades a quantity in slices between its start and end times, and
// requires a limit order's Price
type TradingBotTWAPSettings struct {
	InstrumentName string
	Side           string
	OrderType      string
	Price          types.Number
	Quantity       types.Number
	StartTime      time.Time
	EndTime        time.Time
}

// tradingBotTWAPSettingsWire is a TWAP bot's settings as Crypto.com takes and reports them, its times in milliseconds
type tradingBotTWAPSettingsWire struct {
	InstrumentName        string       `json:"instrument_name"`
	Side                  string       `json:"side"`
	OrderType             string       `json:"type"`
	Price                 types.Number `json:"price,omitempty"`
	Quantity              types.Number `json:"quantity,omitempty"`
	StartTimeMilliseconds uint64       `json:"start_time_millis"`
	EndTimeMilliseconds   uint64       `json:"end_time_millis"`
}

// BotType returns TradingBotTypeTWAP
func (*TradingBotTWAPSettings) BotType() string { return TradingBotTypeTWAP }

// MarshalJSON encodes the settings as Crypto.com takes them, the times in milliseconds
func (s *TradingBotTWAPSettings) MarshalJSON() ([]byte, error) {
	start, err := unixMilliParam(s.StartTime)
	if err != nil {
		return nil, err
	}
	end, err := unixMilliParam(s.EndTime)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&tradingBotTWAPSettingsWire{
		InstrumentName:        s.InstrumentName,
		Side:                  s.Side,
		OrderType:             s.OrderType,
		Price:                 s.Price,
		Quantity:              s.Quantity,
		StartTimeMilliseconds: start,
		EndTimeMilliseconds:   end,
	})
}

// UnmarshalJSON decodes the settings as Crypto.com reports them
func (s *TradingBotTWAPSettings) UnmarshalJSON(data []byte) error {
	var w tradingBotTWAPSettingsWire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	*s = TradingBotTWAPSettings{
		InstrumentName: w.InstrumentName,
		Side:           w.Side,
		OrderType:      w.OrderType,
		Price:          w.Price,
		Quantity:       w.Quantity,
		StartTime:      unixMilliTime(w.StartTimeMilliseconds),
		EndTime:        unixMilliTime(w.EndTimeMilliseconds),
	}
	return nil
}

// TradingBotGridSettings is a grid bot's settings: it places buy and sell orders at grid levels between its lower and
// upper prices, at least 2 levels; trailing up raises the grid with the price, up to TrailingUpPriceCap
type TradingBotGridSettings struct {
	InstrumentName     string       `json:"instrument_name"`
	TotalInvestment    types.Number `json:"total_investment,omitempty"`
	LowerPrice         types.Number `json:"lower_price,omitempty"`
	UpperPrice         types.Number `json:"upper_price,omitempty"`
	GridMode           string       `json:"grid_mode"`
	GridLevels         uint64       `json:"grid_no"`
	MarginEnabled      bool         `json:"margin_enabled"`
	TrailingUpEnabled  bool         `json:"trailing_up_enabled"`
	StopPrice          types.Number `json:"stop_price,omitempty"`
	TakeProfitPrice    types.Number `json:"take_profit_price,omitempty"`
	TrailingUpPriceCap types.Number `json:"trailing_up_price_cap,omitempty"`
}

// BotType returns TradingBotTypeGrid
func (*TradingBotGridSettings) BotType() string { return TradingBotTypeGrid }

// TradingBotFundingArbitrageSettings is a funding arbitrage bot's settings: it holds a spot position and an opposing
// perpetual swap position to earn the funding rate
type TradingBotFundingArbitrageSettings struct {
	TotalInvestment    types.Number  `json:"total_investment,omitempty"`
	InvestmentCurrency currency.Code `json:"investment_currency"`
	SpotLeg            TradingBotLeg `json:"spot_leg"`
	PerpetualLeg       TradingBotLeg `json:"perp_leg"`
}

// BotType returns TradingBotTypeFundingArbitrage
func (*TradingBotFundingArbitrageSettings) BotType() string { return TradingBotTypeFundingArbitrage }

// TradingBotLeg is a funding arbitrage bot's position in an instrument
type TradingBotLeg struct {
	InstrumentName string `json:"instrument_name"`
	Side           string `json:"side"`
}

// TradingBotDCAUpdateSettings is the settings of a DCA bot UpdateTradingBot can change; settings left unset are kept,
// and AutoStake is a pointer so auto-staking can be turned off. InvestmentDuration counts months or milliseconds, as
// TradingBotDCASettings'
type TradingBotDCAUpdateSettings struct {
	Allocations         []TradingBotAllocation
	OrderType           string
	Price               types.Number
	MaxOrders           uint64
	StopPrice           types.Number
	TakeProfitPrice     types.Number
	AutoStake           *bool
	InvestmentDuration  uint64
	InvestmentFrequency time.Duration
	Quantity            types.Number
}

// tradingBotDCAUpdateSettingsWire is a DCA bot's settings update as Crypto.com takes it, the frequency in milliseconds
type tradingBotDCAUpdateSettingsWire struct {
	Allocations                     []TradingBotAllocation `json:"allocations,omitempty"`
	OrderType                       string                 `json:"type,omitempty"`
	Price                           types.Number           `json:"price,omitempty"`
	MaxOrders                       uint64                 `json:"max_order,omitempty,string"`
	StopPrice                       types.Number           `json:"stop_price,omitempty"`
	TakeProfitPrice                 types.Number           `json:"take_profit_price,omitempty"`
	AutoStake                       *bool                  `json:"auto_stake,omitempty"`
	InvestmentDuration              uint64                 `json:"investment_duration,omitempty"`
	InvestmentFrequencyMilliseconds uint64                 `json:"investment_frequency,omitempty"`
	Quantity                        types.Number           `json:"quantity,omitempty"`
}

// BotType returns TradingBotTypeDCA
func (*TradingBotDCAUpdateSettings) BotType() string { return TradingBotTypeDCA }

// TradingBotGridUpdateSettings is the settings of a grid bot UpdateTradingBot can change; prices left unset are kept
type TradingBotGridUpdateSettings struct {
	StopPrice          types.Number `json:"stop_price,omitempty"`
	TakeProfitPrice    types.Number `json:"take_profit_price,omitempty"`
	TrailingUpPriceCap types.Number `json:"trailing_up_price_cap,omitempty"`
}

// BotType returns TradingBotTypeGrid
func (*TradingBotGridUpdateSettings) BotType() string { return TradingBotTypeGrid }

// CreateTradingBotRequest creates a trading bot of a type with that type's settings; DisplayName is 4 to 15 characters
type CreateTradingBotRequest struct {
	BotType           string             `json:"bot_type"`
	NotifyOnBotChange bool               `json:"notify_on_bot_change"`
	DisplayName       string             `json:"display_name,omitempty"`
	Settings          TradingBotSettings `json:"settings"`
}

// CreateTradingBotResponse identifies a trading bot created
type CreateTradingBotResponse struct {
	BotID string `json:"bot_id"`
}

// UpdateTradingBotRequest updates a running or paused trading bot's notifications and display name, and a DCA or grid
// bot's settings, which are left as they are when Settings is nil
type UpdateTradingBotRequest struct {
	BotID             string
	BotType           string
	NotifyOnBotChange bool
	DisplayName       string
	Settings          TradingBotUpdateSettings
}

// updateTradingBotParams is the params of an update-trading-bot request; Settings holds the settings update as
// Crypto.com takes it
type updateTradingBotParams struct {
	BotID             uint64 `json:"bot_id"`
	BotType           string `json:"bot_type"`
	NotifyOnBotChange bool   `json:"notify_on_bot_change"`
	DisplayName       string `json:"display_name,omitempty"`
	Settings          any    `json:"settings,omitempty"`
}

// TerminateTradingBotRequest terminates a trading bot, cancelling its open orders; ClosePosition also closes a funding
// arbitrage bot's positions
type TerminateTradingBotRequest struct {
	BotID         string
	BotType       string
	ClosePosition bool
}

// terminateTradingBotParams is the params of a terminate-trading-bot request
type terminateTradingBotParams struct {
	BotID         uint64 `json:"bot_id"`
	BotType       string `json:"bot_type"`
	ClosePosition bool   `json:"close_position,omitempty"`
}

// TradingBotRequest identifies the trading bot to pause or resume
type TradingBotRequest struct {
	BotID   string
	BotType string
}

// tradingBotParams is the params of a pause-trading-bot or resume-trading-bot request
type tradingBotParams struct {
	BotID   uint64 `json:"bot_id"`
	BotType string `json:"bot_type"`
}

// TradingBotsRequest filters the trading bots by type, which is required, and by ID or by state, one of which is
// required; Page counts from 0, and PageSize is 20 by default and at most 500
type TradingBotsRequest struct {
	BotID    string
	BotTypes []string
	States   []string
	Page     uint64
	PageSize uint64
}

// tradingBotsParams is the params of a get-trading-bots request
type tradingBotsParams struct {
	BotID    uint64   `json:"bot_id,omitempty"`
	BotTypes []string `json:"bot_types"`
	States   []string `json:"state,omitempty"`
	Page     uint64   `json:"page,omitempty"`
	PageSize uint64   `json:"page_size,omitempty"`
}

// TradingBotResponse is a trading bot; Settings holds its type's settings, and is nil for a bot type this package does
// not know. Reason is set for a terminated bot, and EndTime is unset while a bot is active
type TradingBotResponse struct {
	BotID            string                     `json:"bot_id"`
	BotType          string                     `json:"bot_type"`
	DisplayName      string                     `json:"display_name"`
	State            string                     `json:"state"`
	Reason           string                     `json:"reason"`
	CreateTime       types.Time                 `json:"create_time"`
	UpdateTime       types.Time                 `json:"update_time"`
	EndTime          types.Time                 `json:"end_time"`
	Migrated         bool                       `json:"migrated"`
	Settings         TradingBotSettings         `json:"-"`
	ExecutionSummary TradingBotExecutionSummary `json:"execution_summary"`
}

// UnmarshalJSON decodes a trading bot, its settings into its bot type's settings
func (b *TradingBotResponse) UnmarshalJSON(data []byte) error {
	type Alias TradingBotResponse
	aux := struct {
		*Alias
		Settings json.RawMessage `json:"settings"`
	}{Alias: (*Alias)(b)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return b.decodeSettings(aux.Settings)
}

// decodeSettings decodes a bot's settings into its bot type's settings; a type this package does not know is left
// without settings rather than failing the whole list of bots
func (b *TradingBotResponse) decodeSettings(data json.RawMessage) error {
	b.Settings = nil
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var settings TradingBotSettings
	switch b.BotType {
	case TradingBotTypeDCA:
		settings = new(TradingBotDCASettings)
	case TradingBotTypeTWAP:
		settings = new(TradingBotTWAPSettings)
	case TradingBotTypeGrid:
		settings = new(TradingBotGridSettings)
	case TradingBotTypeFundingArbitrage:
		settings = new(TradingBotFundingArbitrageSettings)
	default:
		return nil
	}
	if err := json.Unmarshal(data, settings); err != nil {
		return fmt.Errorf("%s bot settings: %w", b.BotType, err)
	}
	b.Settings = settings
	return nil
}

// TradingBotExecutionSummary summarises a trading bot's executions; Crypto.com documents its fields, which vary by bot
// type, for DCA bots only, so other types may report fields not decoded here
type TradingBotExecutionSummary struct {
	ExecutedOrders uint64              `json:"executed_order,string"`
	TotalInvested  types.PreciseNumber `json:"total_invested"`
	TotalQuantity  types.PreciseNumber `json:"total_quantity"`
	AveragePrice   types.PreciseNumber `json:"avg_price"`
}

// tradingBotListResponse is the result of get-trading-bots
type tradingBotListResponse struct {
	BotList []TradingBotResponse `json:"bot_list"`
}

// TradingBotExecutionsRequest filters a trading bot's executions by order status and, for a basket DCA bot, by the
// basket's instruments; Page and PageSize page them as TradingBotsRequest's do
type TradingBotExecutionsRequest struct {
	BotID                 string
	OrderStatuses         []string
	BasketInstrumentNames []string
	Page                  uint64
	PageSize              uint64
}

// tradingBotExecutionsParams is the params of a get-trading-bot-executions request
type tradingBotExecutionsParams struct {
	BotID           uint64                  `json:"bot_id"`
	OrderStatuses   []string                `json:"order_status,omitempty"`
	BasketBotFilter *tradingBotBasketFilter `json:"basket_bot_filter,omitempty"`
	Page            uint64                  `json:"page,omitempty"`
	PageSize        uint64                  `json:"page_size,omitempty"`
}

// tradingBotBasketFilter limits a basket DCA bot's executions to some of its instruments
type tradingBotBasketFilter struct {
	IncludeChildSymbols []string `json:"include_child_symbols"`
}

// TradingBotExecutionResponse is an order a trading bot placed; InstrumentName is the instrument's display name, such
// as BTC/USD, and Symbol the instrument name the other methods take, such as BTC_USD
type TradingBotExecutionResponse struct {
	InstrumentName     string              `json:"instrument_name"`
	Symbol             string              `json:"symbol"`
	ExecutionTime      types.Time          `json:"execution_time"`
	ScheduleTime       types.Time          `json:"schedule_time"`
	OrderStatus        string              `json:"order_status"`
	Side               string              `json:"side"`
	Price              types.PreciseNumber `json:"price"`
	Quantity           types.PreciseNumber `json:"quantity"`
	AveragePrice       types.PreciseNumber `json:"avg_price"`
	CumulativeQuantity types.PreciseNumber `json:"cum_qty"`
	OrderID            string              `json:"order_id"`
	UpdateTime         types.Time          `json:"update_time"`
	ExecutedPercentage types.Number        `json:"executed_percentage"`
}

// tradingBotExecutionListResponse is the result of get-trading-bot-executions
type tradingBotExecutionListResponse struct {
	ExecutionList []TradingBotExecutionResponse `json:"execution_list"`
}

// millisecondsDuration returns a whole number of milliseconds as a duration, capped at the longest duration
func millisecondsDuration(ms uint64) time.Duration {
	if ms > math.MaxInt64/uint64(time.Millisecond) {
		return math.MaxInt64
	}
	return time.Duration(ms) * time.Millisecond
}

// unixMilliTime returns Unix milliseconds as a time, and the zero time for zero, which Crypto.com sends for a time unset
func unixMilliTime(ms uint64) time.Time {
	if ms == 0 || ms > math.MaxInt64 {
		return time.Time{}
	}
	return time.UnixMilli(int64(ms))
}
