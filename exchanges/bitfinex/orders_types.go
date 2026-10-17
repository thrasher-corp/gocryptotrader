package bitfinex

import (
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Ledger entry categories, which GetLedgers filters by
const (
	LedgerCategoryExchange uint64 = 5
	// LedgerCategoryPositionChange is a position modified, closed or liquidated
	LedgerCategoryPositionChange   uint64 = 22
	LedgerCategoryPositionClaim    uint64 = 23
	LedgerCategoryPositionTransfer uint64 = 25
	LedgerCategoryPositionSwap     uint64 = 26
	// LedgerCategoryPositionFundingCost is a position's funding cost or interest charged
	LedgerCategoryPositionFundingCost uint64 = 27
	// LedgerCategoryMarginPayment is a margin, swap or interest payment
	LedgerCategoryMarginPayment           uint64 = 28
	LedgerCategoryDerivativesFunding      uint64 = 29
	LedgerCategorySettlement              uint64 = 31
	LedgerCategoryTransfer                uint64 = 51
	LedgerCategoryDeposit                 uint64 = 101
	LedgerCategoryWithdrawal              uint64 = 104
	LedgerCategoryCancelledWithdrawal     uint64 = 105
	LedgerCategoryTradingFee              uint64 = 201
	LedgerCategoryTradingRebate           uint64 = 202
	LedgerCategoryHiddenOrderFee          uint64 = 204
	LedgerCategoryOTCTradeFee             uint64 = 207
	LedgerCategorySwapFee                 uint64 = 222
	LedgerCategoryClaimingFee             uint64 = 224
	LedgerCategoryUsedMarginFundingCharge uint64 = 226
	LedgerCategoryUnusedMarginFundingFee  uint64 = 228
	// LedgerCategoryEarnedFee is an earned fee or affiliate rebate
	LedgerCategoryEarnedFee            uint64 = 241
	LedgerCategoryETHFXLoyaltyFee      uint64 = 243
	LedgerCategoryDepositFee           uint64 = 251
	LedgerCategoryWithdrawalFee        uint64 = 254
	LedgerCategoryWithdrawalExpressFee uint64 = 255
	LedgerCategoryMinerFee             uint64 = 258
	LedgerCategoryStakingPayment       uint64 = 262
	LedgerCategoryAdjustment           uint64 = 401
	LedgerCategoryExpense              uint64 = 501
	// LedgerCategoryCurrencyConversionFee is a currency conversion or computation fee
	LedgerCategoryCurrencyConversionFee uint64 = 905
	LedgerCategoryMonthlyProfitPayment  uint64 = 907
	LedgerCategoryLosses                uint64 = 911
)

// OpenOrdersRequest filters the active orders GetOpenOrders returns; an empty request returns them all
type OpenOrdersRequest struct {
	IDs     []uint64
	GroupID uint64
	// ClientOrderID filters by client order ID, and needs ClientOrderIDDate, the UTC day it belongs to
	ClientOrderID     uint64
	ClientOrderIDDate time.Time
}

// OpenOrdersBySymbolRequest filters the active orders on a trading pair GetOpenOrdersBySymbol returns
type OpenOrdersBySymbolRequest struct {
	// Symbol is a trading pair, such as tBTCUSD
	Symbol  string
	IDs     []uint64
	GroupID uint64
	// ClientOrderID filters by client order ID, and needs ClientOrderIDDate, the UTC day it belongs to
	ClientOrderID     uint64
	ClientOrderIDDate time.Time
}

// openOrdersBody is the body GetOpenOrders and GetOpenOrdersBySymbol send
type openOrdersBody struct {
	IDs               []uint64 `json:"id,omitempty"`
	GroupID           uint64   `json:"gid,omitempty"`
	ClientOrderID     uint64   `json:"cid,omitempty"`
	ClientOrderIDDate string   `json:"cid_date,omitempty"`
}

// OrderMeta is the meta data an order is placed or updated with
type OrderMeta struct {
	AffiliateCode string
	// MakeVisible shows a hidden order, one with OrderFlagHidden, in the book once it is hit
	MakeVisible bool
	// ProtectSelfMatch cancels the order rather than let it match one of the account's own orders
	ProtectSelfMatch bool
}

// orderMetaBody is the meta object of an order placement or update, whose options Bitfinex enables with 1
type orderMetaBody struct {
	AffiliateCode    string `json:"aff_code,omitempty"`
	MakeVisible      uint8  `json:"make_visible,omitempty"`
	ProtectSelfMatch uint8  `json:"protect_selfmatch,omitempty"`
}

// PlaceOrderRequest is an order PlaceOrder submits
type PlaceOrderRequest struct {
	// Type is one of the order types; prefixed with ExchangeOrderTypePrefix the order trades on the exchange wallet,
	// otherwise on margin
	Type string
	// Symbol is a trading pair, such as tBTCUSD
	Symbol string
	// Amount is positive to buy and negative to sell
	Amount float64
	// Price is the order's price, or the stop price of a stop order; market orders ignore it
	Price float64
	// Leverage, from 1 to 100, applies to derivative orders only; Bitfinex uses 10 when it is unset
	Leverage uint64
	// TrailingPrice is a trailing stop order's distance from the market price
	TrailingPrice float64
	// AuxiliaryLimitPrice is a stop limit order's limit price
	AuxiliaryLimitPrice float64
	// OCOStopPrice is the stop price of the order OrderFlagOneCancelsOther pairs this one with
	OCOStopPrice float64
	GroupID      uint64
	// ClientOrderID should be unique within its UTC day
	ClientOrderID uint64
	// Flags is the sum of the order's flags
	Flags uint64
	// TimeInForce is when Bitfinex cancels the order, to the second
	TimeInForce time.Time
	Meta        *OrderMeta
}

// placeOrderBody is the body PlaceOrder sends, and an order submission of OrderMultiOp
type placeOrderBody struct {
	Type                string         `json:"type"`
	Symbol              string         `json:"symbol"`
	Amount              string         `json:"amount"`
	Price               string         `json:"price"`
	Leverage            uint64         `json:"lev,omitempty"`
	TrailingPrice       string         `json:"price_trailing,omitempty"`
	AuxiliaryLimitPrice string         `json:"price_aux_limit,omitempty"`
	OCOStopPrice        string         `json:"price_oco_stop,omitempty"`
	GroupID             uint64         `json:"gid,omitempty"`
	ClientOrderID       uint64         `json:"cid,omitempty"`
	Flags               uint64         `json:"flags,omitempty"`
	TimeInForce         string         `json:"tif,omitempty"`
	Meta                *orderMetaBody `json:"meta,omitempty"`
}

// UpdateOrderRequest is an update UpdateOrder makes to an order; the order keeps what the request leaves unset
type UpdateOrderRequest struct {
	ID uint64
	// Amount replaces what remains of the order: positive to buy and negative to sell
	Amount float64
	Price  float64
	// Delta is added to what remains of the order
	Delta         float64
	ClientOrderID uint64
	// ClientOrderIDDate is the UTC day ClientOrderID belongs to
	ClientOrderIDDate time.Time
	GroupID           uint64
	// Flags replaces the order's flags when set, so zero clears them
	Flags *uint64
	// Leverage, from 1 to 100, is required to update a derivative order
	Leverage            uint64
	TrailingPrice       float64
	AuxiliaryLimitPrice float64
	// TimeInForce is when Bitfinex cancels the order, to the second; an order's time in force can be changed but not
	// removed
	TimeInForce time.Time
	Meta        *OrderMeta
}

// updateOrderBody is the body UpdateOrder sends, and an order update of OrderMultiOp
type updateOrderBody struct {
	ID                  uint64         `json:"id"`
	Amount              string         `json:"amount,omitempty"`
	Price               string         `json:"price,omitempty"`
	Delta               string         `json:"delta,omitempty"`
	ClientOrderID       uint64         `json:"cid,omitempty"`
	ClientOrderIDDate   string         `json:"cid_date,omitempty"`
	GroupID             uint64         `json:"gid,omitempty"`
	Flags               *uint64        `json:"flags,omitempty"`
	Leverage            uint64         `json:"lev,omitempty"`
	TrailingPrice       string         `json:"price_trailing,omitempty"`
	AuxiliaryLimitPrice string         `json:"price_aux_limit,omitempty"`
	TimeInForce         string         `json:"tif,omitempty"`
	Meta                *orderMetaBody `json:"meta,omitempty"`
}

// CancelExistingOrderRequest picks the order CancelExistingOrder cancels: by ID, or by client order ID with the UTC
// day it belongs to
type CancelExistingOrderRequest struct {
	ID                uint64
	ClientOrderID     uint64
	ClientOrderIDDate time.Time
}

// cancelOrderBody is the body CancelExistingOrder sends, and an order cancellation of OrderMultiOp
type cancelOrderBody struct {
	ID                uint64 `json:"id,omitempty"`
	ClientOrderID     uint64 `json:"cid,omitempty"`
	ClientOrderIDDate string `json:"cid_date,omitempty"`
}

// DatedClientOrderID is a client order ID with the UTC day it belongs to, which together identify an order
type DatedClientOrderID struct {
	ID   uint64
	Date time.Time
}

// CancelOrdersRequest picks the orders CancelOrders cancels, by their IDs, group IDs or client order IDs, or all of
// them
type CancelOrdersRequest struct {
	IDs            []uint64
	GroupIDs       []uint64
	ClientOrderIDs []DatedClientOrderID
	// All cancels every open order, trading and derivative alike
	All bool
}

// cancelOrdersBody is the body CancelOrders sends, and a multiple cancellation of OrderMultiOp; each client order ID
// is a [cid, cid_date] pair
type cancelOrdersBody struct {
	IDs            []uint64 `json:"id,omitempty"`
	GroupIDs       []uint64 `json:"gid,omitempty"`
	ClientOrderIDs [][2]any `json:"cid,omitempty"`
	All            uint8    `json:"all,omitempty"`
}

// OrderOperation is an operation OrderMultiOp sends: a *PlaceOrderRequest submits an order, an *UpdateOrderRequest
// updates one, a *CancelExistingOrderRequest cancels one and a *CancelOrdersRequest cancels several
type OrderOperation interface {
	// operation returns the operation's abbreviation and the body its own endpoint takes
	operation() (abbreviation string, body any, err error)
}

// orderMultiOpBody is the body OrderMultiOp sends, each operation an [abbreviation, body] pair
type orderMultiOpBody struct {
	Ops [][2]any `json:"ops"`
}

// OrderOperationOrders holds the orders an operation of OrderMultiOp affected
type OrderOperationOrders struct {
	Orders []Order
}

// UnmarshalJSON decodes an operation notification's data, which the docs show as a list of orders, as Submit Order and
// Cancel Orders (multiple) return, but which may be one order, as Update Order and Cancel Order return
func (o *OrderOperationOrders) UnmarshalJSON(data []byte) error {
	// An order begins with its ID, a list of orders with an order
	if _, dataType, _, err := jsonparser.Get(data, "[0]"); err == nil && dataType == jsonparser.Number {
		var single Order
		if err := single.UnmarshalJSON(data); err != nil {
			return err
		}
		o.Orders = []Order{single}
		return nil
	}
	// A fresh slice, as decoding into the old one would keep its orders' fields wherever the new ones are null
	var orders []Order
	if err := json.Unmarshal(data, &orders); err != nil {
		return err
	}
	o.Orders = orders
	return nil
}

// OrdersHistoryRequest filters the closed and cancelled orders GetOrdersHistory returns
type OrdersHistoryRequest struct {
	// Start and End bound the orders' creation times; either may be unset
	Start time.Time
	End   time.Time
	// Limit is at most 2500
	Limit uint64
	IDs   []uint64
}

// OrdersHistoryBySymbolRequest filters the closed and cancelled orders on a trading pair GetOrdersHistoryBySymbol
// returns
type OrdersHistoryBySymbolRequest struct {
	// Symbol is a trading pair, such as tBTCUSD
	Symbol string
	// Start and End bound the orders' creation times; either may be unset
	Start time.Time
	End   time.Time
	// Limit is at most 2500
	Limit uint64
	IDs   []uint64
}

// ordersHistoryBody is the body the orders history and OTC orders history endpoints send
type ordersHistoryBody struct {
	historyBody
	IDs []uint64 `json:"id,omitempty"`
}

// OrderTradesRequest picks the order whose trades GetOrderTrades returns
type OrderTradesRequest struct {
	// Symbol is the order's trading pair, such as tBTCUSD
	Symbol  string
	OrderID uint64
}

// AccountTradesRequest filters the trades GetAccountTrades returns
type AccountTradesRequest struct {
	// Start and End bound the trades' execution times; either may be unset
	Start time.Time
	End   time.Time
	// Limit is at most 2500
	Limit uint64
	// Sort, by execution time, is 1 for oldest first or -1 for newest first
	Sort int64
}

// AccountTradesBySymbolRequest filters the trades on a trading pair GetAccountTradesBySymbol returns
type AccountTradesBySymbolRequest struct {
	// Symbol is a trading pair, such as tBTCUSD
	Symbol string
	// Start and End bound the trades' execution times; either may be unset
	Start time.Time
	End   time.Time
	// Limit is at most 2500
	Limit uint64
	// Sort, by execution time, is 1 for oldest first or -1 for newest first
	Sort int64
}

// OTCOrdersHistoryRequest filters the over the counter orders GetOTCOrdersHistory returns
type OTCOrdersHistoryRequest struct {
	// Symbol is a trading pair, such as tBTCUSD, or empty for every pair
	Symbol string
	Start  time.Time
	End    time.Time
	// Limit is at most 500; Bitfinex returns 25 orders when it is unset
	Limit uint64
	IDs   []uint64
}

// OTCOrder is an over the counter order between the account and a counterparty
type OTCOrder struct {
	ID        uint64     `tuple:"0"`
	Symbol    string     `tuple:"1"`
	CreatedAt types.Time `tuple:"2"`
	UpdatedAt types.Time `tuple:"3"`
	// IsInitiator is true when the account initiated the order and false when the counterparty did
	IsInitiator          types.Boolean `tuple:"5"`
	InitiatorNickname    string        `tuple:"6"`
	CounterpartyNickname string        `tuple:"7"`
	// Amount is positive for a buy and negative for a sell
	Amount float64 `tuple:"9"`
	Price  float64 `tuple:"10"`
	// Status is PENDING, CANCELED, REJECTED or COMPLETED
	Status string `tuple:"12"`
	// TimeInForce is when Bitfinex cancels the order
	TimeInForce types.Time `tuple:"13"`
}

// UnmarshalJSON decodes an OTC order array
func (o *OTCOrder) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, o) }

// LedgersRequest filters the ledger entries GetLedgers returns
type LedgersRequest struct {
	// Currency is the entries' currency, or empty for every currency
	Currency currency.Code
	// Category is one of the ledger categories, or zero for every category
	Category uint64
	Start    time.Time
	End      time.Time
	// Limit is at most 2500
	Limit uint64
	// Wallet is a wallet, such as exchange, margin or funding, or empty for every wallet
	Wallet string
}

// ledgersBody is the body GetLedgers sends
type ledgersBody struct {
	historyBody
	Category uint64 `json:"category,omitempty"`
	Wallet   string `json:"wallet,omitempty"`
}

// LedgerEntry is a change to one of the account's wallet balances
type LedgerEntry struct {
	ID       uint64        `tuple:"0"`
	Currency currency.Code `tuple:"1"`
	// Wallet is exchange, margin, funding or contribution, or empty when Bitfinex sends none
	Wallet  string     `tuple:"2"`
	Time    types.Time `tuple:"3"`
	Amount  float64    `tuple:"5"`
	Balance float64    `tuple:"6"`
	// Description describes the change, such as "Settlement @ 185.79 on wallet margin"
	Description string `tuple:"8"`
}

// UnmarshalJSON decodes a ledger entry array
func (l *LedgerEntry) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, l) }
