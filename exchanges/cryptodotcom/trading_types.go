package cryptodotcom

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Isolated margin transfer directions: credit the isolated position from the account, or debit it back
const (
	IsolatedMarginTransferDirectionCredit = "CREDIT"
	IsolatedMarginTransferDirectionDebit  = "DEBIT"
)

// Journal types, the kinds of transaction private/get-transactions reports and filters by
const (
	JournalTypeTrading               = "TRADING"
	JournalTypeTradeFee              = "TRADE_FEE"
	JournalTypeOnChainWithdrawal     = "ONCHAIN_WITHDRAWAL"
	JournalTypeOnChainDeposit        = "ONCHAIN_DEPOSIT"
	JournalTypeFunding               = "FUNDING"
	JournalTypeRealisedPNL           = "REALIZED_PNL"
	JournalTypeInsuranceFund         = "INSURANCE_FUND"
	JournalTypeSocialisedLoss        = "SOCIALIZED_LOSS"
	JournalTypeLiquidationFee        = "LIQUIDATION_FEE"
	JournalTypeSessionReset          = "SESSION_RESET"
	JournalTypeAdjustment            = "ADJUSTMENT"
	JournalTypeSessionSettle         = "SESSION_SETTLE"
	JournalTypeUncoveredLoss         = "UNCOVERED_LOSS"
	JournalTypeAdminAdjustment       = "ADMIN_ADJUSTMENT"
	JournalTypeDelist                = "DELIST"
	JournalTypeSettlementFee         = "SETTLEMENT_FEE"
	JournalTypeAutoConversion        = "AUTO_CONVERSION"
	JournalTypeManualConversion      = "MANUAL_CONVERSION"
	JournalTypeSubAccountTransaction = "SUBACCOUNT_TX"
	JournalTypeFiatWithdrawalCancel  = "FIAT_WITHDRAWAL_CANCEL"
	JournalTypeMarginTradeInterest   = "MARGIN_TRADE_INTEREST"
)

// CreateOrderRequest is a LIMIT or MARKET order to create, alone, in an order list or in a batch; trigger orders are
// created with CreateAdvancedOrder
type CreateOrderRequest struct {
	// InstrumentName is required, except in a batch, whose orders inherit the batch's
	InstrumentName string
	// Side is SideBuy or SideSell
	Side string
	// OrderType is OrderTypeLimit or OrderTypeMarket
	OrderType string
	// Price is a LIMIT order's limit price, which a MARKET order does not take
	Price types.Number
	// Quantity is required, except for a MARKET BUY that gives Notional, the amount to spend, in its place
	Quantity types.Number
	Notional types.Number
	// ClientOrderID, at most 36 characters, identifies the order to the caller; without one Crypto.com uses the
	// request's nonce, which can repeat across orders
	ClientOrderID string
	// TimeInForce defaults to TimeInForceGoodTillCancel for a LIMIT order, the only one ExecutionInstructionPostOnly
	// allows
	TimeInForce string
	// ExecutionInstructions may hold ExecutionInstructionPostOnly or ExecutionInstructionSmartPostOnly, not both, and
	// ExecutionInstructionReduceOnly, ExecutionInstructionIsolatedMargin and ExecutionInstructionMarginOrder
	ExecutionInstructions []string
	// SpotMargin is SpotMarginSpot or SpotMarginMargin
	SpotMargin string
	// FeeInstrumentName is the currency, such as CRO, the fees are deducted in
	FeeInstrumentName currency.Code
	// SelfTradePreventionID, up to 32767, needs SelfTradePreventionScope; SelfTradePreventionInstruction is required
	// with a scope other than SelfTradePreventionScopeReset
	SelfTradePreventionID          uint64
	SelfTradePreventionInstruction string
	SelfTradePreventionScope       string
	IsolatedMarginAmount           types.Number
	IsolationID                    string
	Leverage                       types.Number
}

// AmendOrderRequest amends an order's price and quantity, both required even when unchanged; the order is named by
// OrderID or, failing that, OriginalClientOrderID
type AmendOrderRequest struct {
	OrderID               string `json:"order_id,omitempty"`
	OriginalClientOrderID string `json:"orig_client_oid,omitempty"`
	// ClientOrderID, at most 36 characters, is the amended order's client order ID
	ClientOrderID string       `json:"client_oid,omitempty"`
	NewPrice      types.Number `json:"new_price"`
	NewQuantity   types.Number `json:"new_quantity"`
}

// CancelOrderRequest names an order to cancel, alone or in a batch, by its order ID or client order ID
type CancelOrderRequest struct {
	OrderID       string `json:"order_id,omitempty"`
	ClientOrderID string `json:"client_oid,omitempty"`
}

// CancelAllOrdersRequest selects the orders to cancel: those of an instrument, or of every instrument without one, and
// of CancelOrderTypeLimit or CancelOrderTypeAll
type CancelAllOrdersRequest struct {
	InstrumentName string `json:"instrument_name,omitempty"`
	OrderType      string `json:"type,omitempty"`
}

// ClosePositionRequest closes a position, partly when Quantity is given, with a LIMIT order at Price or a MARKET order;
// an isolated position is named by its IsolationID, without which the cross-margin position is closed
type ClosePositionRequest struct {
	InstrumentName string       `json:"instrument_name"`
	OrderType      string       `json:"type"`
	Price          types.Number `json:"price,omitempty"`
	Quantity       types.Number `json:"quantity,omitempty"`
	IsolationID    string       `json:"isolation_id,omitempty"`
}

// OpenOrdersRequest selects the open orders of an instrument, or of every instrument without one
type OpenOrdersRequest struct {
	InstrumentName string `json:"instrument_name,omitempty"`
}

// OrderDetailRequest names an order by its order ID or client order ID
type OrderDetailRequest struct {
	OrderID       string `json:"order_id,omitempty"`
	ClientOrderID string `json:"client_oid,omitempty"`
}

// CancelOrderListEntryRequest names an order of an order list to cancel, by its order ID or client order ID
type CancelOrderListEntryRequest struct {
	InstrumentName string `json:"instrument_name,omitempty"`
	OrderID        string `json:"order_id,omitempty"`
	ClientOrderID  string `json:"client_oid,omitempty"`
}

// ChangeAccountLeverageRequest sets the maximum leverage, 1 to 100, of the logged-in account; each instrument's own
// maximum still applies when lower
type ChangeAccountLeverageRequest struct {
	AccountID string `json:"account_id"`
	Leverage  uint64 `json:"leverage"`
}

// ChangeAccountSettingsRequest changes the account's self-trade prevention settings and maximum leverage; unset fields
// are left unchanged. SelfTradePreventionScopeReset resets the self-trade prevention settings, ignoring the
// instruction and ID
type ChangeAccountSettingsRequest struct {
	SelfTradePreventionScope       string `json:"stp_scope,omitempty"`
	SelfTradePreventionInstruction string `json:"stp_inst,omitempty"`
	SelfTradePreventionID          uint64 `json:"stp_id,string,omitempty"`
	Leverage                       uint64 `json:"leverage,omitempty"`
}

// CreateIsolatedMarginTransferRequest moves Amount USD between the account and an isolated position, in the direction
// IsolatedMarginTransferDirectionCredit or IsolatedMarginTransferDirectionDebit
type CreateIsolatedMarginTransferRequest struct {
	IsolationID string       `json:"isolation_id"`
	Direction   string       `json:"direction"`
	Amount      types.Number `json:"amount"`
}

// ChangeIsolatedMarginLeverageRequest sets an isolated position's maximum leverage
type ChangeIsolatedMarginLeverageRequest struct {
	IsolationID string       `json:"isolation_id"`
	Leverage    types.Number `json:"leverage"`
}

// UserOrderHistoryRequest selects past orders: of an instrument and isolated position when given, within the window,
// at most Limit, up to 100, of them
type UserOrderHistoryRequest struct {
	InstrumentName string
	StartTime      time.Time
	EndTime        time.Time
	Limit          uint64
	IsolationID    string
}

// UserTradesRequest selects the account's trades: of an instrument and isolated position when given, at most Limit,
// up to 100, before EndTime, which defaults to now, and from StartTime, which defaults to a day before EndTime
type UserTradesRequest struct {
	InstrumentName string
	StartTime      time.Time
	EndTime        time.Time
	Limit          uint64
	IsolationID    string
}

// TransactionsRequest selects the account's transactions: of an instrument, journal type and isolated position when
// given, at most Limit, up to 100, before EndTime, which defaults to now, and from StartTime, which defaults to a day
// before EndTime
type TransactionsRequest struct {
	InstrumentName string
	// JournalType is one of the JournalType constants
	JournalType string
	StartTime   time.Time
	EndTime     time.Time
	Limit       uint64
	IsolationID string
}

// BatchOrderRequest cancels and creates orders of one instrument in one websocket request: at least one cancel or
// create, and at most 10 creates, which inherit InstrumentName. The self-trade prevention fields apply to the batch
type BatchOrderRequest struct {
	InstrumentName                        string
	Cancels                               []CancelOrderRequest
	Creates                               []CreateOrderRequest
	SelfTradePreventionCancelType         string
	SelfTradePreventionAccountEnforcement bool
	SelfTradePreventionGroup              string
}

// AccountSettingsResponse is the account's maximum leverage and self-trade prevention settings
type AccountSettingsResponse struct {
	Leverage                       types.Number `json:"leverage"`
	SelfTradePreventionID          uint64       `json:"stp_id,string"`
	SelfTradePreventionScope       string       `json:"stp_scope"`
	SelfTradePreventionInstruction string       `json:"stp_inst"`
}

// FeeRateResponse is the account's 30-day trading volume tiers and the fee rates they give, in basis points
type FeeRateResponse struct {
	SpotTier                                 uint64       `json:"spot_tier,string"`
	DerivativesTier                          uint64       `json:"deriv_tier,string"`
	EffectiveSpotMakerRateBasisPoints        types.Number `json:"effective_spot_maker_rate_bps"`
	EffectiveSpotTakerRateBasisPoints        types.Number `json:"effective_spot_taker_rate_bps"`
	EffectiveDerivativesMakerRateBasisPoints types.Number `json:"effective_deriv_maker_rate_bps"`
	EffectiveDerivativesTakerRateBasisPoints types.Number `json:"effective_deriv_taker_rate_bps"`
}

// InstrumentFeeRateResponse is the account's fee rates for an instrument, in basis points
type InstrumentFeeRateResponse struct {
	InstrumentName                string       `json:"instrument_name"`
	EffectiveMakerRateBasisPoints types.Number `json:"effective_maker_rate_bps"`
	EffectiveTakerRateBasisPoints types.Number `json:"effective_taker_rate_bps"`
}

// UserTradeResponse is one of the account's trades; negative fees and fee credits were deducted, the fees net of the
// credits used. The match count and index are only reported for some trades, the index only for a maker's
type UserTradeResponse struct {
	AccountID string `json:"account_id"`
	// EventDate is the day of the trade as YYYY-MM-DD, which CreateTime times
	EventDate           string              `json:"event_date"`
	JournalType         string              `json:"journal_type"`
	TradedQuantity      types.PreciseNumber `json:"traded_quantity"`
	TradedPrice         types.PreciseNumber `json:"traded_price"`
	Fees                types.PreciseNumber `json:"fees"`
	FeeCredits          types.PreciseNumber `json:"fee_credits"`
	OrderID             string              `json:"order_id"`
	TradeID             string              `json:"trade_id"`
	TradeMatchID        string              `json:"trade_match_id"`
	ClientOrderID       string              `json:"client_oid"`
	TakerSide           string              `json:"taker_side"`
	Side                string              `json:"side"`
	InstrumentName      string              `json:"instrument_name"`
	FeeInstrumentName   currency.Code       `json:"fee_instrument_name"`
	CreateTime          types.Time          `json:"create_time"`
	CreateTimeNano      types.Time          `json:"create_time_ns"`
	TransactionTimeNano types.Time          `json:"transact_time_ns"`
	MatchCount          uint64              `json:"match_count,string"`
	MatchIndex          uint64              `json:"match_index,string"`
	IsolationID         string              `json:"isolation_id"`
	IsolationType       string              `json:"isolation_type"`
}

// TransactionResponse is one of the account's transactions; the order, trade and taker side fields are only set for
// trades
type TransactionResponse struct {
	AccountID string `json:"account_id"`
	// EventDate is the day of the transaction as YYYY-MM-DD, which EventTime times
	EventDate           string              `json:"event_date"`
	JournalType         string              `json:"journal_type"`
	JournalID           NumericID           `json:"journal_id"`
	TransactionQuantity types.PreciseNumber `json:"transaction_qty"`
	TransactionCost     types.PreciseNumber `json:"transaction_cost"`
	RealisedPNL         types.PreciseNumber `json:"realized_pnl"`
	OrderID             string              `json:"order_id"`
	TradeID             string              `json:"trade_id"`
	TradeMatchID        string              `json:"trade_match_id"`
	EventTime           types.Time          `json:"event_timestamp_ms"`
	EventTimeNano       types.Time          `json:"event_timestamp_ns"`
	ClientOrderID       string              `json:"client_oid"`
	TakerSide           string              `json:"taker_side"`
	Side                string              `json:"side"`
	InstrumentName      string              `json:"instrument_name"`
	IsolationID         string              `json:"isolation_id"`
	IsolationType       string              `json:"isolation_type"`
}

// BatchOrderResponse is the outcome of each cancel and create of a batch; Crypto.com reports no index for them, and
// its documented example lists them in the request's order
type BatchOrderResponse struct {
	Cancels []OrderListEntryResponse `json:"cancels"`
	Creates []OrderListEntryResponse `json:"creates"`
}

// createOrderParams is the params creating an order, and each order of an order list or batch; a batch's orders leave
// the instrument out, as they inherit the batch's
type createOrderParams struct {
	InstrumentName                 string       `json:"instrument_name,omitempty"`
	Side                           string       `json:"side"`
	OrderType                      string       `json:"type"`
	Price                          types.Number `json:"price,omitempty"`
	Quantity                       types.Number `json:"quantity,omitempty"`
	Notional                       types.Number `json:"notional,omitempty"`
	ClientOrderID                  string       `json:"client_oid,omitempty"`
	TimeInForce                    string       `json:"time_in_force,omitempty"`
	ExecutionInstructions          []string     `json:"exec_inst,omitempty"`
	SpotMargin                     string       `json:"spot_margin,omitempty"`
	FeeInstrumentName              string       `json:"fee_instrument_name,omitempty"`
	SelfTradePreventionID          uint64       `json:"stp_id,string,omitempty"`
	SelfTradePreventionInstruction string       `json:"stp_inst,omitempty"`
	SelfTradePreventionScope       string       `json:"stp_scope,omitempty"`
	IsolatedMarginAmount           types.Number `json:"isolated_margin_amount,omitempty"`
	IsolationID                    string       `json:"isolation_id,omitempty"`
	Leverage                       types.Number `json:"leverage,omitempty"`
}

// createOrderListParams is the params of a create-order-list request
type createOrderListParams struct {
	ContingencyType string              `json:"contingency_type"`
	OrderList       []createOrderParams `json:"order_list"`
}

// cancelOrderListParams is the params of a cancel-order-list request
type cancelOrderListParams struct {
	ContingencyType string                        `json:"contingency_type"`
	OrderList       []CancelOrderListEntryRequest `json:"order_list"`
}

// instrumentFeeRateParams is the params of a get-instrument-fee-rate request
type instrumentFeeRateParams struct {
	InstrumentName string `json:"instrument_name"`
}

// tradingHistoryParams is the params of an order history, trades or transactions query, its window in Unix
// nanoseconds
type tradingHistoryParams struct {
	InstrumentName string `json:"instrument_name,omitempty"`
	JournalType    string `json:"journal_type,omitempty"`
	StartTime      uint64 `json:"start_time,string,omitempty"`
	EndTime        uint64 `json:"end_time,string,omitempty"`
	Limit          uint64 `json:"limit,omitempty"`
	IsolationID    string `json:"isolation_id,omitempty"`
}

// batchOrderParams is the params of a batch-order-request
type batchOrderParams struct {
	InstrumentName                        string               `json:"instrument_name"`
	Cancels                               []CancelOrderRequest `json:"cancels,omitempty"`
	Creates                               []createOrderParams  `json:"creates,omitempty"`
	SelfTradePreventionCancelType         string               `json:"stp_cancel_type,omitempty"`
	SelfTradePreventionAccountEnforcement bool                 `json:"stp_account_enforce_flag,omitempty"`
	SelfTradePreventionGroup              string               `json:"stp_group,omitempty"`
}
