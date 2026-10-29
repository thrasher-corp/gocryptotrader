package gate

import (
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FuturesContractResponse represents futures contract detailed data.
type FuturesContractResponse struct {
	Name              currency.Pair `json:"name"`
	Type              string        `json:"type"`
	QuantoMultiplier  types.Number  `json:"quanto_multiplier"`
	RefDiscountRate   types.Number  `json:"ref_discount_rate"`
	OrderPriceDeviate types.Number  `json:"order_price_deviate"`
	MaintenanceRate   types.Number  `json:"maintenance_rate"`
	// Deprecated: Gate has deprecated it
	MarkType              string       `json:"mark_type"`
	LastPrice             types.Number `json:"last_price"`
	MarkPrice             types.Number `json:"mark_price"`
	IndexPrice            types.Number `json:"index_price"`
	FundingRateIndicative types.Number `json:"funding_rate_indicative"`
	MarkPriceRound        types.Number `json:"mark_price_round"`
	FundingOffset         types.Number `json:"funding_offset"`
	Delisting             bool         `json:"in_delisting"`
	// Deprecated: Gate has deprecated it
	RiskLimitBase   types.Number          `json:"risk_limit_base"`
	InterestRate    types.Number          `json:"interest_rate"`
	OrderPriceRound types.Number          `json:"order_price_round"`
	OrderSizeMin    types.Number          `json:"order_size_min"`
	RefRebateRate   types.Number          `json:"ref_rebate_rate"`
	FundingInterval types.DurationSeconds `json:"funding_interval"`
	// Deprecated: Gate has deprecated it
	RiskLimitStep types.Number `json:"risk_limit_step"`
	LeverageMin   types.Number `json:"leverage_min"`
	LeverageMax   types.Number `json:"leverage_max"`
	// Deprecated: use the risk limit tiers
	RiskLimitMax       types.Number `json:"risk_limit_max"`
	MakerFeeRate       types.Number `json:"maker_fee_rate"`
	TakerFeeRate       types.Number `json:"taker_fee_rate"`
	FundingRate        types.Number `json:"funding_rate"`
	OrderSizeMax       types.Number `json:"order_size_max"`
	FundingNextApply   types.Time   `json:"funding_next_apply"`
	ShortUsers         types.Number `json:"short_users"`
	ConfigChangeTime   types.Time   `json:"config_change_time"`
	TradeSize          types.Number `json:"trade_size"`
	PositionSize       types.Number `json:"position_size"`
	LongUsers          types.Number `json:"long_users"`
	FundingImpactValue types.Number `json:"funding_impact_value"`
	OrdersLimit        types.Number `json:"orders_limit"`
	TradeID            uint64       `json:"trade_id"`
	OrderbookID        uint64       `json:"orderbook_id"`
	EnableBonus        bool         `json:"enable_bonus"`
	EnableCredit       bool         `json:"enable_credit"`
	CreateTime         types.Time   `json:"create_time"`
	// Deprecated: Gate has deprecated it
	FundingCapRatio      types.Number `json:"funding_cap_ratio"`
	Status               string       `json:"status"`
	LaunchTime           types.Time   `json:"launch_time"`
	DelistingTime        types.Time   `json:"delisting_time"`
	DelistedTime         types.Time   `json:"delisted_time"`
	EnableDecimal        bool         `json:"enable_decimal"`
	MarketOrderSlipRatio types.Number `json:"market_order_slip_ratio"`
	MarketOrderSizeMax   types.Number `json:"market_order_size_max"`
	FundingRateLimit     types.Number `json:"funding_rate_limit"`
	ContractType         string       `json:"contract_type"`
	EnableCircuitBreaker bool         `json:"enable_circuit_breaker"`
	// The fields below are undocumented but returned live
	CrossLeverageDefault types.Number `json:"cross_leverage_default"`
	VoucherLeverage      types.Number `json:"voucher_leverage"`
	IsPreMarket          bool         `json:"is_pre_market"`
}

// FuturesADLRiskStatesResponse holds the auto-deleveraging risk state of every futures market of a settlement currency
type FuturesADLRiskStatesResponse struct {
	Settle string                          `json:"settle"`
	States map[string]*FuturesADLRiskState `json:"states"`
}

// FuturesADLRiskState holds a futures market's auto-deleveraging risk state
type FuturesADLRiskState struct {
	State        string     `json:"state"`
	CalculatedAt types.Time `json:"calculated_at_ms"`
}

// FuturesTradeByTimeRangeResponse represents one of the user's futures fills queried by time range
type FuturesTradeByTimeRangeResponse struct {
	TradeID    string       `json:"trade_id"`
	CreateTime types.Time   `json:"create_time"`
	Contract   string       `json:"contract"`
	OrderID    string       `json:"order_id"`
	Size       types.Number `json:"size"`
	CloseSize  types.Number `json:"close_size"`
	Price      types.Number `json:"price"`
	Role       string       `json:"role"`
	Text       string       `json:"text"`
	Fee        types.Number `json:"fee"`
	PointFee   types.Number `json:"point_fee"`
}

// FuturesPremiumIndexKlineResponse represents premium index K-Line information.
type FuturesPremiumIndexKlineResponse struct {
	UnixTimestamp types.Time   `json:"t"`
	ClosePrice    types.Number `json:"c"`
	HighestPrice  types.Number `json:"h"`
	LowestPrice   types.Number `json:"l"`
	OpenPrice     types.Number `json:"o"`
}

// FuturesFundingRateResponse represents futures funding rate response.
type FuturesFundingRateResponse struct {
	Timestamp types.Time   `json:"t"`
	Rate      types.Number `json:"r"`
}

// batchFundingRatesRequest holds the contracts a batch funding rate query asks for
type batchFundingRatesRequest struct {
	Contracts []string `json:"contracts"`
}

// BatchFundingRatesResponse holds a contract's funding rate history from a batch query
type BatchFundingRatesResponse struct {
	Contract string                        `json:"contract"`
	Data     []*FuturesFundingRateResponse `json:"data"`
}

// ContractStatResponse represents futures stats
type ContractStatResponse struct {
	Time                   types.Time   `json:"time"`
	LongShortTaker         types.Number `json:"lsr_taker"`
	LongShortAccount       types.Number `json:"lsr_account"`
	LongLiquidationSize    types.Number `json:"long_liq_size"`
	ShortLiquidationSize   types.Number `json:"short_liq_size"`
	OpenInterest           types.Number `json:"open_interest"`
	ShortLiquidationUSD    types.Number `json:"short_liq_usd"`
	MarkPrice              types.Number `json:"mark_price"`
	TopLongShortSize       types.Number `json:"top_lsr_size"`
	ShortLiquidationAmount types.Number `json:"short_liq_amount"`
	LongLiquidationAmount  types.Number `json:"long_liq_amount"`
	LastFundingRate        types.Number `json:"last_funding_rate"`
	OpenInterestUSD        types.Number `json:"open_interest_usd"`
	TopLongShortAccount    types.Number `json:"top_lsr_account"`
	LongLiquidationUSD     types.Number `json:"long_liq_usd"`
	TopLongSize            types.Number `json:"top_long_size"`
	TopShortSize           types.Number `json:"top_short_size"`
	ShortLiquidationUSDNew types.Number `json:"short_liq_usd_new"`
	LongLiquidationUSDNew  types.Number `json:"long_liq_usd_new"`
	TopLongAccount         types.Number `json:"top_long_account"`
	TopShortAccount        types.Number `json:"top_short_account"`
	LongTakerSize          types.Number `json:"long_taker_size"`
	ShortTakerSize         types.Number `json:"short_taker_size"`
	LongUsers              types.Number `json:"long_users"`
	ShortUsers             types.Number `json:"short_users"`
}

// IndexConstituentResponse represents index constituents
type IndexConstituentResponse struct {
	Index        string                     `json:"index"`
	Constituents []*IndexConstituentSources `json:"constituents"`
}

// IndexConstituentSources holds the symbols an exchange contributes to an index
type IndexConstituentSources struct {
	Exchange string   `json:"exchange"`
	Symbols  []string `json:"symbols"`
}

// LiquidationHistoryResponse represents a public futures liquidation order
type LiquidationHistoryResponse struct {
	Time       types.Time   `json:"time"`
	Contract   string       `json:"contract"`
	Size       types.Number `json:"size"`
	OrderSize  types.Number `json:"order_size"`
	OrderPrice types.Number `json:"order_price"`
	FillPrice  types.Number `json:"fill_price"`
	// RemainingAmount is documented as a reserved field with no current meaning
	RemainingAmount types.Number `json:"left"`
}

// positionReply decodes a position reply, which the docs define as an object but which has also been handled as a
// single element array
type positionReply struct {
	Position *PositionResponse
}

// UnmarshalJSON implements json.Unmarshaler
func (p *positionReply) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] != '[' {
		p.Position = new(PositionResponse)
		return json.Unmarshal(data, p.Position)
	}
	var positions []*PositionResponse
	if err := json.Unmarshal(data, &positions); err != nil {
		return err
	}
	if len(positions) != 1 {
		return fmt.Errorf("%w: expected one position, received %d", common.ErrInvalidResponse, len(positions))
	}
	p.Position = positions[0]
	return nil
}

// FuturesHistoricalPositionResponse represents a historical futures position snapshot queried by time range
type FuturesHistoricalPositionResponse struct {
	Contract                currency.Pair `json:"contract"`
	Size                    types.Number  `json:"size"`
	Leverage                types.Number  `json:"leverage"`
	RiskLimit               types.Number  `json:"risk_limit"`
	LeverageMax             types.Number  `json:"leverage_max"`
	MaintenanceRate         types.Number  `json:"maintenance_rate"`
	Margin                  types.Number  `json:"margin"`
	LiquidationPrice        types.Number  `json:"liq_price"`
	RealisedPNL             types.Number  `json:"realised_pnl"`
	HistoryPNL              types.Number  `json:"history_pnl"`
	LastClosePNL            types.Number  `json:"last_close_pnl"`
	RealisedPNLPoint        types.Number  `json:"realised_point"`
	RealisedPNLHistoryPoint types.Number  `json:"history_point"`
	Mode                    string        `json:"mode"`
	CrossLeverageLimit      types.Number  `json:"cross_leverage_limit"`
	EntryPrice              types.Number  `json:"entry_price"`
	Time                    types.Time    `json:"time"`
}

// FuturesLeverageResponse holds a contract's leverage for a position margin mode and side
type FuturesLeverageResponse struct {
	Leverage types.Number `json:"Lever"`
}

// FuturesPositionMarginModeRequest holds a request to switch a futures position between isolated and cross margin
type FuturesPositionMarginModeRequest struct {
	Mode     string        `json:"mode"` // ISOLATED or CROSS
	Contract currency.Pair `json:"contract"`
}

// FuturesOrderCreateRequest represents future order creation parameters
type FuturesOrderCreateRequest struct {
	Contract                  currency.Pair `json:"contract"`
	Size                      types.Number  `json:"size"` // positive long, negative short, zero to close a position
	Iceberg                   types.Number  `json:"iceberg"`
	Price                     types.Number  `json:"price"` // zero with an ioc time in force places a market order
	TimeInForce               string        `json:"tif"`
	Text                      string        `json:"text,omitempty"`  // errors when empty; Either populated or omitted
	ClosePosition             bool          `json:"close,omitempty"` // Size needs to be zero if true
	ReduceOnly                bool          `json:"reduce_only,omitempty"`
	AutoSize                  string        `json:"auto_size,omitempty"` // either close_long or close_short, requires zero in size field
	Settle                    currency.Code `json:"-"`                   // Used in URL. REST Calls only.
	SelfTradePreventionAction string        `json:"stp_act,omitempty"`
	PositionID                uint64        `json:"pid,omitempty"`
	MarketOrderSlipRatio      types.Number  `json:"market_order_slip_ratio,omitempty"`
	PositionMarginMode        string        `json:"pos_margin_mode,omitempty"`
	ActionMode                string        `json:"action_mode,omitempty"`
	TakeProfitTriggerPrice    types.Number  `json:"tpsl_tp_trigger_price,omitempty"`
	StopLossTriggerPrice      types.Number  `json:"tpsl_sl_trigger_price,omitempty"`
}

// MarshalJSON sends size, iceberg and price as decimal strings even when zero, since Gate reads a zero price
// with an ioc time in force as a market order and a zero size as a close order, whereas types.Number marshals
// zero as an empty string
func (c FuturesOrderCreateRequest) MarshalJSON() ([]byte, error) { //nolint:gocritic // hugeParam: a value receiver lets json call this for unaddressable values, such as orders marshalled by value
	type alias FuturesOrderCreateRequest
	return json.Marshal(&struct {
		Contract currency.Pair `json:"contract"`
		Size     string        `json:"size"`
		Iceberg  string        `json:"iceberg"`
		Price    string        `json:"price"`
		*alias
	}{
		Contract: c.Contract,
		Size:     strconv.FormatFloat(c.Size.Float64(), 'f', -1, 64),
		Iceberg:  strconv.FormatFloat(c.Iceberg.Float64(), 'f', -1, 64),
		Price:    strconv.FormatFloat(c.Price.Float64(), 'f', -1, 64),
		alias:    (*alias)(&c),
	})
}

// BatchFuturesOrderResponse represents one order's outcome in a batch futures order placement or amendment
type BatchFuturesOrderResponse struct {
	Succeeded bool   `json:"succeeded"`
	Label     string `json:"label"`
	Detail    string `json:"detail"`
	FuturesOrderResponse
}

// CancelFuturesOrdersRequest holds the filters for cancelling all open futures orders
type CancelFuturesOrdersRequest struct {
	Contract          currency.Pair // Cancels the open orders of every contract when empty
	Side              string        // bid or ask; both are cancelled when empty
	ActionMode        string        // ACK, RESULT or FULL
	ExcludeReduceOnly bool
	Text              string
}

// AmendFuturesOrderRequest represents amend futures order parameter
type AmendFuturesOrderRequest struct {
	Size       types.Number `json:"size,omitempty"` // keeps the order's sign; a size at or below the filled size cancels the order
	Price      types.Number `json:"price,omitempty"`
	AmendText  string       `json:"amend_text,omitempty"`
	Text       string       `json:"text,omitempty"`
	ActionMode string       `json:"action_mode,omitempty"`
}

// BatchAmendFuturesOrderRequest holds one order's amendment in a batch futures order amendment
type BatchAmendFuturesOrderRequest struct {
	OrderID    uint64       `json:"order_id,omitempty"`
	Text       string       `json:"text,omitempty"`
	Size       types.Number `json:"size,omitempty"`
	Price      types.Number `json:"price,omitempty"`
	AmendText  string       `json:"amend_text,omitempty"`
	ActionMode string       `json:"action_mode,omitempty"`
}

// CancelFuturesOrderResultResponse represents one order's outcome in a futures batch cancellation by order ID
type CancelFuturesOrderResultResponse struct {
	UserID    uint64 `json:"user_id"`
	ID        string `json:"id"`
	Succeeded bool   `json:"succeeded"`
	Message   string `json:"message"`
}

// FuturesAutoDeleverageResponse represents an auto-deleveraging of one of the user's futures positions
type FuturesAutoDeleverageResponse struct {
	Time               types.Time    `json:"time"`
	User               uint64        `json:"user"`
	OrderID            uint64        `json:"order_id"`
	Contract           currency.Pair `json:"contract"`
	Leverage           types.Number  `json:"leverage"`
	CrossLeverageLimit types.Number  `json:"cross_leverage_limit"`
	EntryPrice         types.Number  `json:"entry_price"`
	FillPrice          types.Number  `json:"fill_price"`
	TradeSize          types.Number  `json:"trade_size"`
	PositionSize       types.Number  `json:"position_size"`
}

// CountdownRequest represents the body of a futures countdown cancel request
type CountdownRequest struct {
	Countdown types.DurationSeconds `json:"timeout"` // Whole seconds of at least 5; zero cancels the countdown
	Contract  currency.Pair         `json:"contract,omitzero"`
}

// FuturesTradingFeeResponse holds a futures contract's trading fee rates
type FuturesTradingFeeResponse struct {
	TakerFee types.Number `json:"taker_fee"`
	MakerFee types.Number `json:"maker_fee"`
}

// FuturesBBOOrderRequest holds a futures order priced at a level of the order book
type FuturesBBOOrderRequest struct {
	Contract                  currency.Pair `json:"contract"`
	Size                      int64         `json:"size"`      // positive buys, negative sells, zero closes a position
	Direction                 string        `json:"direction"` // sell prices against the bids, buy against the asks
	Iceberg                   uint64        `json:"iceberg,omitempty"`
	Level                     uint64        `json:"level"` // order book level to price at, from 1 to 20
	ClosePosition             bool          `json:"close,omitempty"`
	ReduceOnly                bool          `json:"reduce_only,omitempty"`
	TimeInForce               string        `json:"tif,omitempty"`
	Text                      string        `json:"text,omitempty"`
	AutoSize                  string        `json:"auto_size,omitempty"`
	SelfTradePreventionAction string        `json:"stp_act,omitempty"`
	PositionID                uint64        `json:"pid,omitempty"`
}

// CreateTrailOrderRequest holds a futures trailing stop order creation request
type CreateTrailOrderRequest struct {
	Contract           currency.Pair `json:"contract"`
	Amount             types.Number  `json:"amount"` // positive buys, negative sells
	ActivationPrice    types.Number  `json:"activation_price,omitempty"`
	IsGreaterThanEqual bool          `json:"is_gte"`
	PriceType          uint8         `json:"price_type,omitempty"`   // 1: latest price, 2: index price, 3: mark price
	PriceOffset        string        `json:"price_offset,omitempty"` // a callback ratio such as 0.1% or a price distance such as 0.1
	ReduceOnly         bool          `json:"reduce_only,omitempty"`
	PositionRelated    bool          `json:"position_related,omitempty"`
	Text               string        `json:"text,omitempty"`
	PositionMarginMode string        `json:"pos_margin_mode,omitempty"`
	PositionMode       string        `json:"position_mode,omitempty"`
}

// CreateTrailOrderResponse holds the ID of a newly created trail order
type CreateTrailOrderResponse struct {
	ID NumericString `json:"id"`
}

// StopTrailOrderRequest identifies a trail order to stop by its ID or custom text
type StopTrailOrderRequest struct {
	ID   uint64 `json:"id,omitempty"`
	Text string `json:"text,omitempty"`
}

// StopAllTrailOrdersRequest holds the filters for stopping trail orders in batch
type StopAllTrailOrdersRequest struct {
	Contract        currency.Pair `json:"contract,omitzero"`
	RelatedPosition uint8         `json:"related_position,omitempty"` // 1: long, 2: short
}

// TrailOrderResponse represents a futures trailing stop order
type TrailOrderResponse struct {
	ID                 NumericString `json:"id"`
	UserID             NumericString `json:"user_id"`
	User               NumericString `json:"user"`
	Contract           currency.Pair `json:"contract"`
	Settle             string        `json:"settle"`
	Amount             types.Number  `json:"amount"`
	IsGreaterThanEqual bool          `json:"is_gte"`
	ActivationPrice    types.Number  `json:"activation_price"`
	PriceType          uint8         `json:"price_type"`
	PriceOffset        string        `json:"price_offset"`
	Text               string        `json:"text"`
	ReduceOnly         bool          `json:"reduce_only"`
	PositionRelated    bool          `json:"position_related"`
	CreatedAt          types.Time    `json:"created_at"`
	ActivatedAt        types.Time    `json:"activated_at"`
	FinishedAt         types.Time    `json:"finished_at"`
	CreateTime         types.Time    `json:"create_time"`
	ActiveTime         types.Time    `json:"active_time"`
	FinishTime         types.Time    `json:"finish_time"`
	Reason             string        `json:"reason"`
	SuborderText       string        `json:"suborder_text"`
	IsDualMode         bool          `json:"is_dual_mode"`
	TriggerPrice       types.Number  `json:"trigger_price"`
	SuborderID         NumericString `json:"suborder_id"`
	SideLabel          string        `json:"side_label"`
	OriginalStatus     uint64        `json:"original_status"`
	Status             string        `json:"status"`
	PositionSideOutput string        `json:"position_side_output"`
	UpdatedAt          types.Time    `json:"updated_at"`
	ExtremumPrice      types.Number  `json:"extremum_price"`
	StatusCode         string        `json:"status_code"`
	CreatedAtPrecise   types.Time    `json:"created_at_precise"`
	FinishedAtPrecise  types.Time    `json:"finished_at_precise"`
	ActivatedAtPrecise types.Time    `json:"activated_at_precise"`
	StatusLabel        string        `json:"status_label"`
	PositionMarginMode string        `json:"pos_margin_mode"`
	PositionMode       string        `json:"position_mode"`
	ErrorLabel         string        `json:"error_label"`
	Leverage           types.Number  `json:"leverage"`
}

// trailOrderReply decodes a single trail order reply, which the schema nests under "order" while the documented
// examples send the order itself
type trailOrderReply struct {
	Order *TrailOrderResponse
}

// UnmarshalJSON implements json.Unmarshaler
func (t *trailOrderReply) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var nested struct {
		Order *TrailOrderResponse `json:"order"`
	}
	if err := json.Unmarshal(data, &nested); err != nil {
		return err
	}
	if nested.Order != nil {
		t.Order = nested.Order
		return nil
	}
	t.Order = new(TrailOrderResponse)
	return json.Unmarshal(data, t.Order)
}

// GetTrailOrdersRequest holds the filters for listing futures trail orders
type GetTrailOrdersRequest struct {
	Contract        currency.Pair
	IsFinished      bool
	StartAt         time.Time
	EndAt           time.Time
	PageNumber      uint64
	PageSize        uint64
	SortBy          uint8 // 1: creation time, 2: end time
	HideCancelled   bool
	RelatedPosition uint8 // 1: long, 2: short
	SortByTrigger   bool
	ReduceOnly      uint8 // 1: yes, 2: no
	Side            uint8 // 1: long, 2: short
}

// UpdateTrailOrderRequest holds a futures trail order amendment, where empty values leave a parameter unchanged
type UpdateTrailOrderRequest struct {
	ID                 uint64       `json:"id"`
	Amount             types.Number `json:"amount,omitempty"`
	ActivationPrice    string       `json:"activation_price,omitempty"` // "0" triggers immediately
	IsGreaterThanEqual string       `json:"is_gte_str,omitempty"`       // "true" or "false"
	PriceType          uint8        `json:"price_type,omitempty"`
	PriceOffset        string       `json:"price_offset,omitempty"`
}

// trailOrdersReply holds the trail orders a list or stop-all request returns
type trailOrdersReply struct {
	Orders []*TrailOrderResponse `json:"orders"`
}

// trailOrderChangeLogReply holds a trail order's change log
type trailOrderChangeLogReply struct {
	ChangeLog []*TrailOrderChangeResponse `json:"change_log"`
}

// TrailOrderChangeResponse represents a creation or modification of a futures trail order
type TrailOrderChangeResponse struct {
	UpdatedAt          types.Time   `json:"updated_at"`
	Amount             types.Number `json:"amount"`
	IsGreaterThanEqual bool         `json:"is_gte"`
	ActivationPrice    types.Number `json:"activation_price"`
	PriceType          uint8        `json:"price_type"`
	PriceOffset        string       `json:"price_offset"`
	IsCreate           bool         `json:"is_create"`
}

// CreateChaseOrderRequest represents a chase limit order creation request.
type CreateChaseOrderRequest struct {
	Contract           currency.Pair `json:"contract"`
	Amount             types.Number  `json:"amount"`                 // positive buys, negative sells
	PriceLimit         types.Number  `json:"price_limit"`            // the furthest price to chase to; zero sets no limit
	OffsetLimit        types.Number  `json:"offset_limit,omitempty"` // the furthest distance to chase from the best price; exclusive of PriceLimit
	ReduceOnly         bool          `json:"reduce_only,omitempty"`
	Text               string        `json:"text,omitempty"`
	IsDualMode         bool          `json:"is_dual_mode,omitempty"`
	PriceType          int64         `json:"price_type,omitempty"`     // 1: best bid or ask, 2: a distance from the best bid or ask
	PriceGapType       int64         `json:"price_gap_type,omitempty"` // 1: absolute, 2: percentage; used with a PriceType of 2
	PriceGapValue      types.Number  `json:"price_gap_value,omitempty"`
	PositionMarginMode string        `json:"pos_margin_mode,omitempty"`
	PositionMode       string        `json:"position_mode,omitempty"`
	Settle             currency.Code `json:"-"`
}

// MarshalJSON sends the price limit as a decimal string even when zero, since Gate requires the field and reads
// "0" as no limit, whereas types.Number marshals zero as an empty string
func (c CreateChaseOrderRequest) MarshalJSON() ([]byte, error) { //nolint:gocritic // hugeParam: a value receiver lets json call this for unaddressable values, such as requests marshalled by value
	type alias CreateChaseOrderRequest
	return json.Marshal(&struct {
		Contract   currency.Pair `json:"contract"`
		Amount     types.Number  `json:"amount"`
		PriceLimit string        `json:"price_limit"`
		*alias
	}{
		Contract:   c.Contract,
		Amount:     c.Amount,
		PriceLimit: strconv.FormatFloat(c.PriceLimit.Float64(), 'f', -1, 64),
		alias:      (*alias)(&c),
	})
}

// CreateChaseOrderResponse holds the ID of a newly created chase order
type CreateChaseOrderResponse struct {
	ID NumericString `json:"id"`
}

// StopChaseOrderRequest identifies a running chase order to stop by its ID or custom text
type StopChaseOrderRequest struct {
	ID   string `json:"id,omitempty"`
	Text string `json:"text,omitempty"`
}

// StopAllChaseOrdersRequest holds the filters for stopping chase orders in batch
type StopAllChaseOrdersRequest struct {
	Contract           currency.Pair `json:"contract,omitzero"`
	PositionMarginMode string        `json:"pos_margin_mode,omitempty"`
}

// GetChaseOrdersRequest holds the filters for listing futures chase orders
type GetChaseOrdersRequest struct {
	Contract      currency.Pair
	IsFinished    bool
	StartAt       time.Time // required when IsFinished is set
	EndAt         time.Time // required when IsFinished is set
	PageNumber    uint64
	PageSize      uint64 // 1 to 100
	SortBy        uint8  // 1: creation time, 2: finish time
	HideCancelled bool
	ReduceOnly    uint8 // 1: reduce only orders, 2: other orders
	Side          uint8 // 1: long, 2: short
}

// chaseOrderReply holds the chase order a detail or stop request returns
type chaseOrderReply struct {
	Order *ChaseOrderResponse `json:"order"`
}

// chaseOrdersReply holds the chase orders a list or stop-all request returns
type chaseOrdersReply struct {
	Orders []*ChaseOrderResponse `json:"orders"`
}

// ChaseOrderResponse represents a chase limit order detail or list item.
type ChaseOrderResponse struct {
	ID                 NumericString `json:"id"`
	User               NumericString `json:"user"`
	Contract           currency.Pair `json:"contract"`
	Settle             string        `json:"settle"`
	Amount             types.Number  `json:"amount"`
	PriceLimit         types.Number  `json:"price_limit"`
	ReduceOnly         bool          `json:"reduce_only"`
	Text               string        `json:"text"`
	CreateTime         types.Time    `json:"create_time"`
	FinishTime         types.Time    `json:"finish_time"`
	OriginalStatus     uint64        `json:"original_status"`
	Status             string        `json:"status"`
	Reason             string        `json:"reason"`
	FillAmount         types.Number  `json:"fill_amount"`
	AverageFillPrice   types.Number  `json:"average_fill_price"`
	SuborderID         NumericString `json:"suborder_id"`
	IsDualMode         bool          `json:"is_dual_mode"`
	SideLabel          string        `json:"side_label"`
	PositionSideOutput string        `json:"position_side_output"`
	ChasePrice         types.Number  `json:"chase_price"`
	IntervalSeconds    uint64        `json:"interval_sec"`
	UpdatedAt          types.Time    `json:"updated_at"`
	SuborderPrice      types.Number  `json:"suborder_price"`
	SuborderOngoing    bool          `json:"suborder_ongoing"`
	SuborderFinishAs   string        `json:"suborder_finish_as"`
	PriceType          uint64        `json:"price_type"`
	PriceGapType       NumericString `json:"price_gap_type"`
	PriceGapValue      types.Number  `json:"price_gap_value"`
	StatusCode         string        `json:"status_code"`
	CreateTimePrecise  types.Time    `json:"create_time_precise"`
	FinishTimePrecise  types.Time    `json:"finish_time_precise"`
	PositionMarginMode string        `json:"pos_margin_mode"`
	PositionMode       string        `json:"position_mode"`
	Leverage           types.Number  `json:"leverage"`
	ErrorLabel         string        `json:"error_label"`
}

// AmendFuturesPriceTriggeredOrderRequest holds a futures price triggered order amendment
type AmendFuturesPriceTriggeredOrderRequest struct {
	OrderID      uint64       `json:"order_id"`
	Size         *int64       `json:"size,omitempty"`   // 0 fully closes the position
	Amount       types.Number `json:"amount,omitempty"` // A decimal size, which takes precedence over Size
	Price        *float64     `json:"price,omitempty"`  // 0 places a market order once triggered
	TriggerPrice types.Number `json:"trigger_price,omitempty"`
	PriceType    *int64       `json:"price_type,omitempty"` // 0: latest trade price, 1: mark price, 2: index price
	AutoSize     string       `json:"auto_size,omitempty"`
	Close        bool         `json:"close,omitempty"`
}

// MarshalJSON sends the price as the documented decimal string, keeping an explicit zero so that a market
// order can be requested
func (a AmendFuturesPriceTriggeredOrderRequest) MarshalJSON() ([]byte, error) {
	type alias AmendFuturesPriceTriggeredOrderRequest
	var price *string
	if a.Price != nil {
		formatted := strconv.FormatFloat(*a.Price, 'f', -1, 64)
		price = &formatted
	}
	return json.Marshal(&struct {
		OrderID uint64       `json:"order_id"`
		Size    *int64       `json:"size,omitempty"`
		Amount  types.Number `json:"amount,omitempty"`
		Price   *string      `json:"price,omitempty"`
		*alias
	}{
		OrderID: a.OrderID,
		Size:    a.Size,
		Amount:  a.Amount,
		Price:   price,
		alias:   (*alias)(&a),
	})
}
