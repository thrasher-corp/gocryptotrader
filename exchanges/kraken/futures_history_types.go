package kraken

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var errInvalidAccountLogRange = errors.New("invalid account log entry ID range")

// FuturesHistoryExecutionEventsRequest holds the parameters of Get execution events. Every history event listing pages
// the same way
type FuturesHistoryExecutionEventsRequest struct {
	// Pair limits the listing to a market's events
	Pair currency.Pair
	// Since and Before bound the events' times
	Since  time.Time
	Before time.Time
	// Ascending lists the oldest events first, where the newest come first by default
	Ascending bool
	// ContinuationToken continues the listing from a previous response's ContinuationToken, requested with the same
	// Ascending
	ContinuationToken string
	// Count is the maximum number of events to list, which Kraken caps at 1000
	Count uint64
}

// FuturesHistoryExecutionEventsResponse holds a page of the account's executions
type FuturesHistoryExecutionEventsResponse struct {
	AccountUID string                           `json:"accountUid"`
	Length     uint64                           `json:"len"`
	Elements   []FuturesHistoryExecutionElement `json:"elements"`
	// ContinuationToken continues the listing as the next request's ContinuationToken; it is empty once no more events
	// match
	ContinuationToken string `json:"continuationToken"`
}

// FuturesHistoryExecutionElement is an execution event
type FuturesHistoryExecutionElement struct {
	UID       string                       `json:"uid"`
	Timestamp types.Time                   `json:"timestamp"`
	Event     FuturesHistoryExecutionEvent `json:"event"`
}

// FuturesHistoryExecutionEvent holds an execution. Kraken documents its key as execution where public market history
// sends Execution; keys match case-insensitively, so either decodes
type FuturesHistoryExecutionEvent struct {
	Execution FuturesHistoryExecutionDetails `json:"execution"`
}

// FuturesHistoryExecutionDetails holds an execution
type FuturesHistoryExecutionDetails struct {
	Execution            FuturesHistoryExecution `json:"execution"`
	TakerReducedQuantity types.Number            `json:"takerReducedQuantity"`
}

// FuturesHistoryExecution is an execution of one of the account's orders
type FuturesHistoryExecution struct {
	UID       string              `json:"uid"`
	Order     FuturesHistoryOrder `json:"order"`
	Timestamp types.Time          `json:"timestamp"`
	Quantity  types.Number        `json:"quantity"`
	Price     types.Number        `json:"price"`
	MarkPrice types.Number        `json:"markPrice"`
	// ExecutionType is maker or taker
	ExecutionType  string                        `json:"executionType"`
	LimitFilled    bool                          `json:"limitFilled"`
	OldTakerOrder  *FuturesHistoryOrder          `json:"oldTakerOrder"`
	USDValue       types.Number                  `json:"usdValue"`
	OrderData      *FuturesHistoryOrderData      `json:"orderData"`
	PartnerTradeID string                        `json:"partnerTradeId"`
	RegulatoryData *FuturesHistoryRegulatoryData `json:"regulatoryData"`
}

// FuturesHistoryOrder is one of the account's orders as its history records it
type FuturesHistoryOrder struct {
	UID         string `json:"uid"`
	AccountUID  string `json:"accountUid"`
	Tradeable   string `json:"tradeable"`
	PositionUID string `json:"positionUid"`
	// Direction is Buy or Sell
	Direction      string       `json:"direction"`
	Quantity       types.Number `json:"quantity"`
	FilledQuantity types.Number `json:"filled"`
	Timestamp      types.Time   `json:"timestamp"`
	LimitPrice     types.Number `json:"limitPrice"`
	// OrderType is Limit, IoC, Post, Liquidation, Assignment, Stop, Unwind, Market, Block, CoveredLiquidation,
	// HedgeImmediateOrCancel, HedgeAssignment, FillOrKill, Rfq or PartialLiquidation
	OrderType             string                       `json:"orderType"`
	ClientOrderID         string                       `json:"clientId"`
	ReduceOnly            bool                         `json:"reduceOnly"`
	LastUpdateTime        types.Time                   `json:"lastUpdateTimestamp"`
	SpotData              *FuturesHistorySpotOrderData `json:"spotData"`
	ParentOfGroupID       string                       `json:"parentOfGroupId"`
	MemberOfGroupID       string                       `json:"memberOfGroupId"`
	PartnerOrderID        string                       `json:"partnerOrderId"`
	RegulatoryExternalUID string                       `json:"regulatoryExternalUid"`
}

// FuturesHistorySpotOrderData holds a spot order's preferences
type FuturesHistorySpotOrderData struct {
	Margin bool `json:"margin"`
	// FeePreference and QuantityPreference are Base or Quote; FeePreference may be empty
	FeePreference      string `json:"feePreference"`
	QuantityPreference string `json:"quantityPreference"`
}

// FuturesHistoryOrderData holds the fee and position details an execution recorded for its order
type FuturesHistoryOrderData struct {
	Fee            types.Number                 `json:"fee"`
	PositionSize   types.Number                 `json:"positionSize"`
	FeeCalculation FuturesHistoryFeeCalculation `json:"feeCalculationInfo"`
	RealisedPNL    types.Number                 `json:"realizedPnl"`
}

// FuturesHistoryFeeCalculation holds how an execution's fee was calculated
type FuturesHistoryFeeCalculation struct {
	PercentageFee             types.Number `json:"percentageFee"`
	UserFeeDiscountApplied    types.Number `json:"userFeeDiscountApplied"`
	MarketShareRebateCredited types.Number `json:"marketShareRebateCredited"`
}

// FuturesHistoryRegulatoryData holds an execution's regulatory reporting details
type FuturesHistoryRegulatoryData struct {
	Venue        string `json:"venue"`
	Counterparty string `json:"counterparty"`
	ExternalUID  string `json:"externalUid"`
}

// FuturesHistoryOrderEventsRequest holds the parameters of Get order events, filtered by market and paged as
// FuturesHistoryExecutionEventsRequest describes
type FuturesHistoryOrderEventsRequest struct {
	Pair              currency.Pair
	Since             time.Time
	Before            time.Time
	Ascending         bool
	ContinuationToken string
	Count             uint64
	// Opened includes, when true, or excludes, when false, the orders placed within the window
	Opened *bool
	// Closed includes, when true, or excludes, when false, the orders closed, cancelled or rejected within the window
	Closed *bool
}

// FuturesHistoryOrderEventsResponse holds a page of the account's order events
type FuturesHistoryOrderEventsResponse struct {
	AccountUID        string                       `json:"accountUid"`
	Length            uint64                       `json:"len"`
	ServerTime        time.Time                    `json:"serverTime"`
	Elements          []FuturesHistoryOrderElement `json:"elements"`
	ContinuationToken string                       `json:"continuationToken"`
}

// FuturesHistoryOrderElement is an order event
type FuturesHistoryOrderElement struct {
	UID       string                   `json:"uid"`
	Timestamp types.Time               `json:"timestamp"`
	Event     FuturesHistoryOrderEvent `json:"event"`
}

// FuturesHistoryOrderEvent holds an order event as the field of its kind, so exactly one is set, or none for a kind
// Kraken has added since
type FuturesHistoryOrderEvent struct {
	OrderPlaced       *FuturesHistoryOrderPlaced       `json:"OrderPlaced"`
	OrderUpdated      *FuturesHistoryOrderUpdated      `json:"OrderUpdated"`
	OrderRejected     *FuturesHistoryOrderRejected     `json:"OrderRejected"`
	OrderCancelled    *FuturesHistoryOrderCancelled    `json:"OrderCancelled"`
	OrderNotFound     *FuturesHistoryOrderNotFound     `json:"OrderNotFound"`
	OrderEditRejected *FuturesHistoryOrderEditRejected `json:"OrderEditRejected"`
}

// FuturesHistoryOrderPlaced is a placed order. Its reducedQuantity, which Kraken documents as always empty, is left out
type FuturesHistoryOrderPlaced struct {
	Order  FuturesHistoryOrder `json:"order"`
	Reason string              `json:"reason"`
}

// FuturesHistoryOrderUpdated is an order's edit or fill, as Reason says
type FuturesHistoryOrderUpdated struct {
	OldOrder        FuturesHistoryOrder `json:"oldOrder"`
	NewOrder        FuturesHistoryOrder `json:"newOrder"`
	Reason          string              `json:"reason"`
	ReducedQuantity types.Number        `json:"reducedQuantity"`
}

// FuturesHistoryOrderRejected is a rejected order
type FuturesHistoryOrderRejected struct {
	Order      FuturesHistoryOrder `json:"order"`
	OrderError string              `json:"orderError"`
	Reason     string              `json:"reason"`
}

// FuturesHistoryOrderCancelled is a cancelled order
type FuturesHistoryOrderCancelled struct {
	Order  FuturesHistoryOrder `json:"order"`
	Reason string              `json:"reason"`
}

// FuturesHistoryOrderNotFound is an order Kraken could not find
type FuturesHistoryOrderNotFound struct {
	AccountUID string `json:"accountUid"`
	// OrderID is formatted as Uuid(uuid=2ceb1d31-f619-457b-870c-fd4ddbb10d45)
	OrderID string `json:"orderId"`
}

// FuturesHistoryOrderEditRejected is a rejected order edit
type FuturesHistoryOrderEditRejected struct {
	OldOrder       FuturesHistoryOrder `json:"oldOrder"`
	AttemptedOrder FuturesHistoryOrder `json:"attemptedOrder"`
	OrderError     string              `json:"orderError"`
}

// FuturesHistoryTriggerEventsRequest holds the parameters of Get trigger events, filtered by market and paged as
// FuturesHistoryExecutionEventsRequest describes
type FuturesHistoryTriggerEventsRequest struct {
	Pair              currency.Pair
	Since             time.Time
	Before            time.Time
	Ascending         bool
	ContinuationToken string
	Count             uint64
	// Opened includes, when true, or excludes, when false, the triggers placed within the window
	Opened *bool
	// Closed includes, when true, or excludes, when false, the triggers closed, cancelled or rejected within the window
	Closed *bool
}

// FuturesHistoryTriggerEventsResponse holds a page of the account's trigger order events
type FuturesHistoryTriggerEventsResponse struct {
	AccountUID        string                         `json:"accountUid"`
	Length            uint64                         `json:"len"`
	ServerTime        time.Time                      `json:"serverTime"`
	Elements          []FuturesHistoryTriggerElement `json:"elements"`
	ContinuationToken string                         `json:"continuationToken"`
}

// FuturesHistoryTriggerElement is a trigger order event
type FuturesHistoryTriggerElement struct {
	UID       string                     `json:"uid"`
	Timestamp types.Time                 `json:"timestamp"`
	Event     FuturesHistoryTriggerEvent `json:"event"`
}

// FuturesHistoryTriggerEvent holds a trigger order event as the field of its kind, so exactly one is set, or none for
// a kind Kraken has added since
type FuturesHistoryTriggerEvent struct {
	OrderTriggerPlaced       *FuturesHistoryTriggerPlaced       `json:"OrderTriggerPlaced"`
	OrderTriggerCancelled    *FuturesHistoryTriggerCancelled    `json:"OrderTriggerCancelled"`
	OrderTriggerUpdated      *FuturesHistoryTriggerUpdated      `json:"OrderTriggerUpdated"`
	OrderTriggerActivated    *FuturesHistoryTriggerActivated    `json:"OrderTriggerActivated"`
	OrderTriggerEditRejected *FuturesHistoryTriggerEditRejected `json:"OrderTriggerEditRejected"`
}

// FuturesHistoryTrigger is one of the account's trigger orders as its history records it
type FuturesHistoryTrigger struct {
	UID        string `json:"uid"`
	AccountID  uint64 `json:"accountId"`
	AccountUID string `json:"accountUid"`
	Tradeable  string `json:"tradeable"`
	// Direction is Buy or Sell
	Direction      string                       `json:"direction"`
	Quantity       types.Number                 `json:"quantity"`
	Timestamp      types.Time                   `json:"timestamp"`
	LimitPrice     types.Number                 `json:"limitPrice"`
	OrderType      string                       `json:"orderType"`
	ClientOrderID  string                       `json:"clientId"`
	ReduceOnly     bool                         `json:"reduceOnly"`
	LastUpdateTime types.Time                   `json:"lastUpdateTimestamp"`
	TriggerOptions FuturesHistoryTriggerOptions `json:"triggerOptions"`
}

// FuturesHistoryTriggerOptions holds when a trigger order fires and how its prices follow the market
type FuturesHistoryTriggerOptions struct {
	TriggerPrice types.Number `json:"triggerPrice"`
	// TriggerSignal is MarkPrice, LastPrice or SpotPrice
	TriggerSignal string `json:"triggerSignal"`
	// TriggerSide is Above or Below
	TriggerSide         string                            `json:"triggerSide"`
	TrailingStopOptions FuturesHistoryTrailingStopOptions `json:"trailingStopOptions"`
	LimitPriceOffset    FuturesHistoryPriceOffset         `json:"limitPriceOffset"`
}

// FuturesHistoryTrailingStopOptions holds how far a trailing stop's trigger price may deviate from the market
type FuturesHistoryTrailingStopOptions struct {
	MaximumDeviation types.Number `json:"maxDeviation"`
	// Unit is Percent or QuoteCurrency
	Unit string `json:"unit"`
}

// FuturesHistoryPriceOffset is a triggered order's limit price offset from its trigger price
type FuturesHistoryPriceOffset struct {
	PriceOffset types.Number `json:"priceOffset"`
	// Unit is Percent or QuoteCurrency
	Unit string `json:"unit"`
}

// FuturesHistoryTriggerPlaced is a placed trigger order
type FuturesHistoryTriggerPlaced struct {
	Order  FuturesHistoryTrigger `json:"order"`
	Reason string                `json:"reason"`
}

// FuturesHistoryTriggerCancelled is a cancelled trigger order
type FuturesHistoryTriggerCancelled struct {
	Order  FuturesHistoryTrigger `json:"order"`
	Reason string                `json:"reason"`
}

// FuturesHistoryTriggerUpdated is an edited trigger order
type FuturesHistoryTriggerUpdated struct {
	OldOrderTrigger FuturesHistoryTrigger `json:"oldOrderTrigger"`
	NewOrderTrigger FuturesHistoryTrigger `json:"newOrderTrigger"`
	Reason          string                `json:"reason"`
}

// FuturesHistoryTriggerActivated is a trigger order that fired
type FuturesHistoryTriggerActivated struct {
	Order FuturesHistoryTrigger `json:"order"`
}

// FuturesHistoryTriggerEditRejected is a rejected trigger order edit
type FuturesHistoryTriggerEditRejected struct {
	AttemptedOrderTrigger FuturesHistoryTrigger `json:"attemptedOrderTrigger"`
	OldOrderTrigger       FuturesHistoryTrigger `json:"oldOrderTrigger"`
	Reason                string                `json:"reason"`
	OrderError            string                `json:"orderError"`
}

// FuturesHistoryPositionEventsRequest holds the parameters of Get position update events, filtered by market and paged
// as FuturesHistoryExecutionEventsRequest describes. Setting position change filters, Opened to NoChange, lists the
// events matching any of them, as setting update reason filters, Trades to Settlement, does; setting both kinds lists
// the events matching one of each, and setting none lists every event
type FuturesHistoryPositionEventsRequest struct {
	Pair               currency.Pair
	Since              time.Time
	Before             time.Time
	Ascending          bool
	ContinuationToken  string
	Count              uint64
	Opened             bool
	Closed             bool
	Increased          bool
	Decreased          bool
	Reversed           bool
	NoChange           bool
	Trades             bool
	FundingRealisation bool
	Settlement         bool
}

// FuturesHistoryPositionEventsResponse holds a page of the account's position updates
type FuturesHistoryPositionEventsResponse struct {
	AccountUID        string                          `json:"accountUid"`
	Length            uint64                          `json:"len"`
	ServerTime        time.Time                       `json:"serverTime"`
	Elements          []FuturesHistoryPositionElement `json:"elements"`
	ContinuationToken string                          `json:"continuationToken"`
}

// FuturesHistoryPositionElement is a position update event
type FuturesHistoryPositionElement struct {
	UID       string                      `json:"uid"`
	Timestamp types.Time                  `json:"timestamp"`
	Event     FuturesHistoryPositionEvent `json:"event"`
}

// FuturesHistoryPositionEvent holds a position update
type FuturesHistoryPositionEvent struct {
	PositionUpdate FuturesHistoryPositionUpdate `json:"PositionUpdate"`
}

// FuturesHistoryPositionUpdate is a change to a position through a trade, a funding realisation or a settlement
type FuturesHistoryPositionUpdate struct {
	AccountUID           string        `json:"accountUid"`
	Tradeable            string        `json:"tradeable"`
	OldPosition          types.Number  `json:"oldPosition"`
	OldAverageEntryPrice types.Number  `json:"oldAverageEntryPrice"`
	NewPosition          types.Number  `json:"newPosition"`
	NewAverageEntryPrice types.Number  `json:"newAverageEntryPrice"`
	FillTime             types.Time    `json:"fillTime"`
	Fee                  types.Number  `json:"fee"`
	FeeCurrency          currency.Code `json:"feeCurrency"`
	RealisedPNL          types.Number  `json:"realizedPnL"`
	// PositionChange is open, close, increase, decrease, reverse or noChange
	PositionChange string       `json:"positionChange"`
	ExecutionUID   string       `json:"executionUid"`
	ExecutionPrice types.Number `json:"executionPrice"`
	ExecutionSize  types.Number `json:"executionSize"`
	// TradeType is userExecution, liquidation, partialLiquidation, assignment or unwind
	TradeType              string       `json:"tradeType"`
	FundingRealisationTime types.Time   `json:"fundingRealizationTime"`
	RealisedFunding        types.Number `json:"realizedFunding"`
	SettlementPrice        types.Number `json:"settlementPrice"`
	Timestamp              types.Time   `json:"timestamp"`
	// UpdateReason is trade, fundingRealisation or settlement
	UpdateReason string `json:"updateReason"`
}

// FuturesHistoryAccountLogRequest holds the parameters of Get account log, whose time and entry ID bounds combine
type FuturesHistoryAccountLogRequest struct {
	Since  time.Time
	Before time.Time
	// FromID and ToID bound the entries' IDs inclusively; IDs start at 1
	FromID uint64
	ToID   uint64
	// Ascending lists the oldest entries first, where the newest come first by default
	Ascending bool
	// EntryTypes limits the entries to these types, such as futures trade or funding rate change
	EntryTypes []string
	// Count is the number of entries to list, 500 when it is 0; larger counts cost more of the history rate limit
	Count uint64
	// ConversionDetails adds each conversion's exchange rate and fee
	ConversionDetails bool
}

// FuturesHistoryAccountLogResponse holds the account's log entries
type FuturesHistoryAccountLogResponse struct {
	AccountUID string                          `json:"accountUid"`
	Logs       []FuturesHistoryAccountLogEntry `json:"logs"`
}

// FuturesHistoryAccountLogEntry is a change to a wallet's balance or a position's size. Kraken sends null for the
// fields that do not apply to an entry, which decode as zero
type FuturesHistoryAccountLogEntry struct {
	// Asset is a currency, such as bch, or for a position its contract, such as pi_bchusd
	Asset      string        `json:"asset"`
	BookingUID string        `json:"booking_uid"`
	Collateral currency.Code `json:"collateral"`
	Contract   string        `json:"contract"`
	Date       time.Time     `json:"date"`
	// ExecutionID is the UID of the associated execution or transfer; a cross-exchange transfer's reference is not
	// always a UUID
	ExecutionID string  `json:"execution"`
	Fee         float64 `json:"fee"`
	// FundingRate is the absolute funding rate when the entry was booked
	FundingRate float64 `json:"funding_rate"`
	ID          uint64  `json:"id"`
	// EntryType describes the entry, such as futures trade or funding rate change
	EntryType            string  `json:"info"`
	MarginAccount        string  `json:"margin_account"`
	MarkPrice            float64 `json:"mark_price"`
	NewAverageEntryPrice float64 `json:"new_average_entry_price"`
	// NewBalance is the wallet's balance or the position's size after the entry, as OldBalance is before it
	NewBalance                 float64 `json:"new_balance"`
	OldAverageEntryPrice       float64 `json:"old_average_entry_price"`
	OldBalance                 float64 `json:"old_balance"`
	RealisedFunding            float64 `json:"realized_funding"`
	RealisedPNL                float64 `json:"realized_pnl"`
	TradePrice                 float64 `json:"trade_price"`
	ConversionSpreadPercentage float64 `json:"conversion_spread_percentage"`
	LiquidationFee             float64 `json:"liquidation_fee"`
	// ExchangeRate is the USD price of ExchangeRateFrom a conversion used. It, ConversionFee and ExchangeRateFrom are
	// only sent with FuturesHistoryAccountLogRequest's ConversionDetails
	ExchangeRate float64 `json:"exchange_rate"`
	// ConversionFee is a percentage, 0.05 being 0.05%
	ConversionFee    float64       `json:"conversion_fee"`
	ExchangeRateFrom currency.Code `json:"exchange_rate_from"`
}

// FuturesHistoryAccountLogCSVRequest holds the parameters of Account log (CSV)
type FuturesHistoryAccountLogCSVRequest struct {
	// ConversionDetails adds each conversion's exchange rate, its base currency and fee
	ConversionDetails bool
}

// FuturesHistoryMarketEventsRequest holds the parameters of Get public execution events, Get public order events and
// Get public mark price events, paged as FuturesHistoryExecutionEventsRequest describes
type FuturesHistoryMarketEventsRequest struct {
	// Pair is the market to list
	Pair              currency.Pair
	Since             time.Time
	Before            time.Time
	Ascending         bool
	ContinuationToken string
	Count             uint64
}

// FuturesHistoryPublicExecutionEventsResponse holds a page of a market's trades
type FuturesHistoryPublicExecutionEventsResponse struct {
	Length            uint64                                 `json:"len"`
	Elements          []FuturesHistoryPublicExecutionElement `json:"elements"`
	ContinuationToken string                                 `json:"continuationToken"`
}

// FuturesHistoryPublicExecutionElement is a public trade event
type FuturesHistoryPublicExecutionElement struct {
	UID       string                             `json:"uid"`
	Timestamp types.Time                         `json:"timestamp"`
	Event     FuturesHistoryPublicExecutionEvent `json:"event"`
}

// FuturesHistoryPublicExecutionEvent holds a public trade
type FuturesHistoryPublicExecutionEvent struct {
	Execution FuturesHistoryPublicExecutionDetails `json:"Execution"`
}

// FuturesHistoryPublicExecutionDetails holds a public trade
type FuturesHistoryPublicExecutionDetails struct {
	Execution            FuturesHistoryPublicExecution `json:"execution"`
	TakerReducedQuantity types.Number                  `json:"takerReducedQuantity"`
}

// FuturesHistoryPublicExecution is a trade between a maker and a taker order; the taker's Direction is the trade's side
type FuturesHistoryPublicExecution struct {
	UID           string                     `json:"uid"`
	MakerOrder    FuturesHistoryPublicOrder  `json:"makerOrder"`
	TakerOrder    FuturesHistoryPublicOrder  `json:"takerOrder"`
	Timestamp     types.Time                 `json:"timestamp"`
	Quantity      types.Number               `json:"quantity"`
	Price         types.Number               `json:"price"`
	MarkPrice     types.Number               `json:"markPrice"`
	LimitFilled   bool                       `json:"limitFilled"`
	OldTakerOrder *FuturesHistoryPublicOrder `json:"oldTakerOrder"`
	USDValue      types.Number               `json:"usdValue"`
}

// FuturesHistoryPublicOrder is an order as public market history shows it
type FuturesHistoryPublicOrder struct {
	UID       string `json:"uid"`
	Tradeable string `json:"tradeable"`
	// Direction is Buy or Sell
	Direction      string       `json:"direction"`
	Quantity       types.Number `json:"quantity"`
	Timestamp      types.Time   `json:"timestamp"`
	LimitPrice     types.Number `json:"limitPrice"`
	OrderType      string       `json:"orderType"`
	ReduceOnly     bool         `json:"reduceOnly"`
	LastUpdateTime types.Time   `json:"lastUpdateTimestamp"`
}

// FuturesHistoryPublicOrderEventsResponse holds a page of a market's order events
type FuturesHistoryPublicOrderEventsResponse struct {
	Length            uint64                             `json:"len"`
	Elements          []FuturesHistoryPublicOrderElement `json:"elements"`
	ContinuationToken string                             `json:"continuationToken"`
}

// FuturesHistoryPublicOrderElement is a public order event
type FuturesHistoryPublicOrderElement struct {
	UID       string                         `json:"uid"`
	Timestamp types.Time                     `json:"timestamp"`
	Event     FuturesHistoryPublicOrderEvent `json:"event"`
}

// FuturesHistoryPublicOrderEvent holds a public order event as the field of its kind, so exactly one is set, or none
// for a kind Kraken has added since. Kraken documents placed, updated and cancelled orders, and sends rejected ones too
type FuturesHistoryPublicOrderEvent struct {
	OrderPlaced    *FuturesHistoryPublicOrderPlaced    `json:"OrderPlaced"`
	OrderUpdated   *FuturesHistoryPublicOrderUpdated   `json:"OrderUpdated"`
	OrderCancelled *FuturesHistoryPublicOrderCancelled `json:"OrderCancelled"`
	OrderRejected  *FuturesHistoryPublicOrderRejected  `json:"OrderRejected"`
}

// FuturesHistoryPublicOrderPlaced is a placed order. Its reducedQuantity, which Kraken documents as always empty, is
// left out
type FuturesHistoryPublicOrderPlaced struct {
	Order  FuturesHistoryPublicOrder `json:"order"`
	Reason string                    `json:"reason"`
}

// FuturesHistoryPublicOrderUpdated is an order's edit or fill, as Reason says
type FuturesHistoryPublicOrderUpdated struct {
	OldOrder        FuturesHistoryPublicOrder `json:"oldOrder"`
	NewOrder        FuturesHistoryPublicOrder `json:"newOrder"`
	Reason          string                    `json:"reason"`
	ReducedQuantity types.Number              `json:"reducedQuantity"`
}

// FuturesHistoryPublicOrderCancelled is a cancelled order
type FuturesHistoryPublicOrderCancelled struct {
	Order  FuturesHistoryPublicOrder `json:"order"`
	Reason string                    `json:"reason"`
}

// FuturesHistoryPublicOrderRejected is a rejected order
type FuturesHistoryPublicOrderRejected struct {
	Order      FuturesHistoryPublicOrder `json:"order"`
	OrderError string                    `json:"orderError"`
	Reason     string                    `json:"reason"`
}

// FuturesHistoryPublicMarkPriceEventsResponse holds a page of a market's mark prices
type FuturesHistoryPublicMarkPriceEventsResponse struct {
	Length            uint64                                 `json:"len"`
	Elements          []FuturesHistoryPublicMarkPriceElement `json:"elements"`
	ContinuationToken string                                 `json:"continuationToken"`
}

// FuturesHistoryPublicMarkPriceElement is a mark price event. Kraken sends it with an empty UID
type FuturesHistoryPublicMarkPriceElement struct {
	UID       string                             `json:"uid"`
	Timestamp types.Time                         `json:"timestamp"`
	Event     FuturesHistoryPublicMarkPriceEvent `json:"event"`
}

// FuturesHistoryPublicMarkPriceEvent holds a mark price change, which Kraken documents as the event itself but sends
// under MarkPriceChanged
type FuturesHistoryPublicMarkPriceEvent struct {
	MarkPriceChanged FuturesHistoryMarkPriceChanged `json:"MarkPriceChanged"`
}

// FuturesHistoryMarkPriceChanged is a market's new mark price
type FuturesHistoryMarkPriceChanged struct {
	Price types.Number `json:"price"`
}
