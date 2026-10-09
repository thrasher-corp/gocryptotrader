package kraken

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errTimeAndIDBound         = errors.New("a bound cannot be both a time and an ID")
	errCursorWithoutPaging    = errors.New("a cursor requires cursor pagination")
	errOffsetWithCursor       = errors.New("an offset cannot be combined with cursor pagination")
	errInvalidLimit           = errors.New("invalid limit")
	errTooManyIDs             = errors.New("too many IDs")
	errTradeIDEmpty           = errors.New("trade ID cannot be empty")
	errLedgerIDEmpty          = errors.New("ledger ID cannot be empty")
	errFilterFormConflict     = errors.New("names cannot be combined with classified names")
	errAssetClassEmpty        = errors.New("asset class cannot be empty")
	errExportReportTypeEmpty  = errors.New("export report type cannot be empty")
	errExportDescriptionEmpty = errors.New("export description cannot be empty")
	errExportIDEmpty          = errors.New("export report ID cannot be empty")
	errExportRemovalTypeEmpty = errors.New("export removal type cannot be empty")
	errExportNotArchive       = errors.New("export response is not an archive")
)

// BalanceRequest holds the parameters of Get Account Balance, Get Extended Balance and Get Credit Lines
type BalanceRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// ExtendedBalance holds an asset's balance with its credit and held amounts. The amount available to trade is Balance
// plus Credit, less CreditUsed and HoldTrade
type ExtendedBalance struct {
	Balance types.Number `json:"balance"`
	// Credit and CreditUsed are only sent for an account with a credit line
	Credit     types.Number `json:"credit"`
	CreditUsed types.Number `json:"credit_used"`
	// HoldTrade is held by open spot orders, margin orders excluded
	HoldTrade types.Number `json:"hold_trade"`
}

// CreditLinesResponse holds a VIP account's credit lines, keyed by asset name
type CreditLinesResponse struct {
	AssetDetails  map[string]CreditLineAsset `json:"asset_details"`
	LimitsMonitor CreditLimitsMonitor        `json:"limits_monitor"`
}

// CreditLineAsset holds an asset's balance and credit line
type CreditLineAsset struct {
	Balance types.Number `json:"balance"`
	// HoldTrade is held by open orders
	HoldTrade types.Number `json:"hold_trade"`
	// Credit is the credit limit; it, CreditUsed and the rates are only sent for an asset with a credit line
	Credit     types.Number `json:"credit"`
	CreditUsed types.Number `json:"credit_used"`
	// CollateralValue is the factor, from 0 to 1, the asset counts towards collateral at; it is only sent for eligible
	// collateral
	CollateralValue float64 `json:"collateral_value"`
	// RolloverRate is charged on drawn credit and ReserveRate on undrawn credit, in percent per rollover period
	RolloverRate types.Number `json:"rollover_fees"`
	ReserveRate  types.Number `json:"reserve_fees"`
}

// CreditLimitsMonitor holds an account's credit totals, valued in USD
type CreditLimitsMonitor struct {
	TotalCreditUSD     types.Number `json:"total_credit_usd"`
	TotalCreditUsedUSD types.Number `json:"total_credit_used_usd"`
	// TotalCollateralValueUSD sums each asset's balance in USD times its collateral value
	TotalCollateralValueUSD types.Number `json:"total_collateral_value_usd"`
	// EquityUSD is total collateral less total credit
	EquityUSD types.Number `json:"equity_usd"`
	// OngoingBalance is total collateral divided by total credit
	OngoingBalance types.Number `json:"ongoing_balance"`
	// DebtToEquity is total credit used divided by equity
	DebtToEquity types.Number `json:"debt_to_equity"`
}

// TradeBalanceRequest holds the parameters of Get Trade Balance
type TradeBalanceRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	// Asset is the asset balances are valued in, ZUSD when it is empty
	Asset currency.Code
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// TradeBalanceResponse holds an account's collateral balances, margin position valuations, equity and margin level,
// valued in the requested asset
type TradeBalanceResponse struct {
	// EquivalentBalance is the combined balance of every currency
	EquivalentBalance types.Number `json:"eb"`
	// TradeBalance is the combined balance of every equity currency
	TradeBalance types.Number `json:"tb"`
	// MarginAmount, UnrealisedNetPNL, CostBasis and Valuation are those of open positions
	MarginAmount     types.Number `json:"m"`
	UnrealisedNetPNL types.Number `json:"n"`
	CostBasis        types.Number `json:"c"`
	Valuation        types.Number `json:"v"`
	// Equity is TradeBalance plus UnrealisedNetPNL
	Equity types.Number `json:"e"`
	// FreeMargin is equity less initial margin, the most margin available to open new positions
	FreeMargin types.Number `json:"mf"`
	// FreeMarginForOrders is the margin available to new orders
	FreeMarginForOrders types.Number `json:"mfo"`
	// MarginLevel is equity divided by initial margin, times 100; it is only sent while margin positions are open
	MarginLevel types.Number `json:"ml"`
	// UnexecutedValue is the value of unfilled and partially filled orders
	UnexecutedValue types.Number `json:"uv"`
}

// OpenOrdersRequest holds the parameters of Get Open Orders
type OpenOrdersRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	// IncludeTrades adds the IDs of each order's trades
	IncludeTrades bool
	// UserReference and ClientOrderID filter the orders to those placed with them
	UserReference int32
	ClientOrderID string
	// WithoutTakerConsolidation lists a taker trade's fills separately rather than consolidated into one trade
	WithoutTakerConsolidation bool
	// WithCursor pages through the orders; set Cursor to the previous page's Cursor.Next to request the next page
	WithCursor bool
	Cursor     string
	// Limit is the number of orders: with WithCursor up to 100 a page and 50 when it is 0, otherwise every open order
	// when it is 0, up to 1000
	Limit uint64
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// OpenOrdersResponse holds open orders keyed by order ID
type OpenOrdersResponse struct {
	Open map[string]OrderInfo `json:"open"`
	// Cursor is only set with cursor pagination while more orders remain
	Cursor PageCursor `json:"cursor"`
}

// PageCursor holds the token of a cursor paginated response's next page
type PageCursor struct {
	// Next is the next request's cursor; it is empty on the last page
	Next string `json:"next"`
}

// OrderInfo holds an order's details. CloseTime and Reason are only set for a closed order
type OrderInfo struct {
	// ReferralOrderID is the order that created this one
	ReferralOrderID string `json:"refid"`
	UserReference   int32  `json:"userref"`
	ClientOrderID   string `json:"cl_ord_id"`
	// Status is pending, open, closed, canceled or expired
	Status    string     `json:"status"`
	OpenTime  types.Time `json:"opentm"`
	StartTime types.Time `json:"starttm"`
	// ExpireTime is zero for an order without an expiry
	ExpireTime  types.Time       `json:"expiretm"`
	Description OrderDescription `json:"descr"`
	// TimeInForce is gtc, ioc, gtd or fok
	TimeInForce    string       `json:"time_in_force"`
	Volume         types.Number `json:"vol"`
	VolumeExecuted types.Number `json:"vol_exec"`
	Cost           types.Number `json:"cost"`
	Fee            types.Number `json:"fee"`
	AveragePrice   types.Number `json:"price"`
	StopPrice      types.Number `json:"stopprice"`
	// LimitPrice is the limit price a limit based order took when it triggered
	LimitPrice types.Number `json:"limitprice"`
	// DisplayVolume is the quantity an iceberg order shows in the book, and DisplayVolumeRemaining the part of it
	// still shown
	DisplayVolume          types.Number `json:"displayvol"`
	DisplayVolumeRemaining types.Number `json:"displayvolremain"`
	ExternalOrderID        string       `json:"ext_ord_id"`
	// OriginalOrderID is the order an edit replaced
	OriginalOrderID string `json:"link_id"`
	ReduceOnly      bool   `json:"reduce_only"`
	// Trigger is the price signal that triggers stop-loss and take-profit orders: last, the default, or index
	Trigger string `json:"trigger"`
	// Margin is set for an order funded on margin
	Margin bool `json:"margin"`
	// Miscellaneous is a comma separated list of notes: stopped, touched, liquidated, partial or amended
	Miscellaneous string `json:"misc"`
	// SenderSubID identifies an institutional account's sub-account or trader for self trade prevention
	SenderSubID string `json:"sender_sub_id"`
	// OrderFlags is a comma separated list of flags: post, fcib, fciq, nompp or viqc
	OrderFlags string     `json:"oflags"`
	TradeIDs   []string   `json:"trades"`
	CloseTime  types.Time `json:"closetm"`
	// Reason explains the status, such as User requested
	Reason string `json:"reason"`
}

// OrderDescription describes an order
type OrderDescription struct {
	Pair string `json:"pair"`
	// Side is buy or sell
	Side      string `json:"type"`
	OrderType string `json:"ordertype"`
	// Price is the limit price, or a triggered order type's trigger price, which an order placed with a relative price,
	// as every trailing stop is, gives as its offset, such as +50.0000%
	Price OrderPrice `json:"price"`
	// SecondaryPrice is the limit price of a stop-loss-limit, take-profit-limit or trailing-stop-limit order
	SecondaryPrice OrderPrice `json:"price2"`
	// Leverage is none, or a ratio such as 5:1
	Leverage string `json:"leverage"`
	// Summary describes the order in words, such as "buy 1.25000000 XBTUSD @ limit 30010.0"
	Summary string `json:"order"`
	// ConditionalClose describes the order's conditional close, if it has one
	ConditionalClose string `json:"close"`
	AssetClass       string `json:"aclass"`
}

// ClosedOrdersRequest holds the parameters of Get Closed Orders
type ClosedOrdersRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	// IncludeTrades adds the IDs of each order's trades
	IncludeTrades bool
	// UserReference and ClientOrderID filter the orders to those placed with them
	UserReference int32
	ClientOrderID string
	// Start and End bound the orders by time, exclusively and inclusively; StartOrderID and EndOrderID bound them by
	// an order's opening time instead
	Start        time.Time
	End          time.Time
	StartOrderID string
	EndOrderID   string
	// Offset pages through the orders the deprecated way, which WithCursor replaces
	Offset uint64
	// WithCursor pages through the orders; set Cursor to the previous page's Cursor.Next to request the next page
	WithCursor bool
	Cursor     string
	// TimeFilter is the time Start and End compare: open, close, or both, the default, which needs an order to open
	// after Start and close by End
	TimeFilter string
	// WithoutTakerConsolidation lists a taker trade's fills separately rather than consolidated into one trade
	WithoutTakerConsolidation bool
	// WithoutCount skips counting the matching orders, which is noticeably faster for an account with many
	WithoutCount bool
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// ClosedOrdersResponse holds a page of up to 50 closed orders keyed by order ID
type ClosedOrdersResponse struct {
	Closed map[string]OrderInfo `json:"closed"`
	// Count is the number of matching orders, 0 with WithoutCount or cursor pagination
	Count uint64 `json:"count"`
	// Cursor is only set with cursor pagination while more orders remain
	Cursor PageCursor `json:"cursor"`
}

// QueryOrdersRequest holds the parameters of Query Orders Info
type QueryOrdersRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	// OrderIDs are the up to 50 orders to query
	OrderIDs []string
	// IncludeTrades adds the IDs of each order's trades
	IncludeTrades bool
	// UserReference and ClientOrderID filter the orders to those placed with them
	UserReference int32
	ClientOrderID string
	// WithoutTakerConsolidation lists a taker trade's fills separately rather than consolidated into one trade
	WithoutTakerConsolidation bool
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// OrderAmendsRequest holds the parameters of Get Order Amends
type OrderAmendsRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	OrderID   string
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// OrderAmendsResponse holds an order's amends in ascending time order, the order as entered first
type OrderAmendsResponse struct {
	// Count is the number of entries, the order as entered included
	Count  uint64       `json:"count"`
	Amends []OrderAmend `json:"amends"`
}

// OrderAmend holds an order's values after an amend
type OrderAmend struct {
	AmendID string `json:"amend_id"`
	// AmendType is original for the order as entered, user for an amend the user requested, or restated for the
	// engine's order maintenance
	AmendType     string       `json:"amend_type"`
	OrderQuantity types.Number `json:"order_qty"`
	// DisplayQuantity is the quantity an iceberg order shows in the book
	DisplayQuantity   types.Number `json:"display_qty"`
	RemainingQuantity types.Number `json:"remaining_qty"`
	LimitPrice        types.Number `json:"limit_price"`
	TriggerPrice      types.Number `json:"trigger_price"`
	Reason            string       `json:"reason"`
	// PostOnly is set when the order could not take liquidity
	PostOnly  bool       `json:"post_only"`
	Timestamp types.Time `json:"timestamp"`
}

// TradesHistoryRequest holds the parameters of Get Trades History
type TradesHistoryRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	// TradeType filters the trades by position: all, the default, any position, closed position, closing position or
	// no position
	TradeType string
	// IncludeTrades adds the closing trades of each position a trade opened
	IncludeTrades bool
	// Start and End bound the trades by time, exclusively and inclusively; StartTradeID and EndTradeID bound them by a
	// trade's time instead
	Start        time.Time
	End          time.Time
	StartTradeID string
	EndTradeID   string
	// Offset pages through the trades the deprecated way, which WithCursor replaces
	Offset uint64
	// WithoutCount skips counting the matching trades, which is noticeably faster for an account with many
	WithoutCount bool
	// WithoutTakerConsolidation lists a taker trade's fills separately rather than consolidated into one trade
	WithoutTakerConsolidation bool
	// IncludeLedgers adds the IDs of each trade's ledger entries, which slows the request
	IncludeLedgers bool
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
	// AssetClass is the class of Pair: forex, the default, equity_pair, futures_contract, synthetic_pair or
	// external_pair
	AssetClass string
	Pair       currency.Pair
	// Limit is the number of trades a page, 50 when it is 0; Kraken lowers a limit above 100 to 100
	Limit uint64
	// WithCursor pages through the trades; set Cursor to the previous page's Cursor.Next to request the next page
	WithCursor bool
	Cursor     string
}

// TradesHistoryResponse holds a page of trades keyed by trade ID
type TradesHistoryResponse struct {
	// Count is the number of matching trades, 0 with WithoutCount or cursor pagination
	Count  uint64               `json:"count"`
	Trades map[string]TradeInfo `json:"trades"`
	// Cursor is only set with cursor pagination while more trades remain
	Cursor PageCursor `json:"cursor"`
}

// TradeInfo holds a trade's details. Amounts are in the pair's precision rather than its assets'. The fields from
// PositionStatus on are only set for a trade that opened a position; the Closed fields, NetPNL and ClosingTradeIDs
// describe the part of it since closed
type TradeInfo struct {
	OrderID    string     `json:"ordertxid"`
	PositionID string     `json:"postxid"`
	Pair       string     `json:"pair"`
	Time       types.Time `json:"time"`
	// Side is buy or sell
	Side      string `json:"type"`
	OrderType string `json:"ordertype"`
	// Price is the average price the order executed at
	Price  types.Number `json:"price"`
	Cost   types.Number `json:"cost"`
	Fee    types.Number `json:"fee"`
	Volume types.Number `json:"vol"`
	// Margin is the initial margin
	Margin   types.Number `json:"margin"`
	Leverage types.Number `json:"leverage"`
	// Miscellaneous is a comma separated list of notes, such as closing for a trade that closes a position
	Miscellaneous       string   `json:"misc"`
	LedgerIDs           []string `json:"ledgers"`
	TradeID             uint64   `json:"trade_id"`
	Maker               bool     `json:"maker"`
	AssetClass          string   `json:"aclass"`
	ExternalExecutionID string   `json:"ext_exec_id"`
	// TradeOrderType is the order type the trade executed as, which differs from OrderType when execution converted it
	TradeOrderType     string       `json:"tradeordertype"`
	PositionStatus     string       `json:"posstatus"`
	ClosedAveragePrice types.Number `json:"cprice"`
	ClosedCost         types.Number `json:"ccost"`
	ClosedFee          types.Number `json:"cfee"`
	ClosedVolume       types.Number `json:"cvol"`
	// ClosedMargin is the margin closing freed
	ClosedMargin    types.Number `json:"cmargin"`
	NetPNL          types.Number `json:"net"`
	ClosingTradeIDs []string     `json:"trades"`
}

// QueryTradesRequest holds the parameters of Query Trades Info
type QueryTradesRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	// TradeIDs are the up to 20 trades to query
	TradeIDs []string
	// IncludeTrades adds the closing trades of each position a trade opened
	IncludeTrades bool
	// IncludeLedgers adds the IDs of each trade's ledger entries
	IncludeLedgers bool
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// OpenPositionsRequest holds the parameters of Get Open Positions
type OpenPositionsRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	// TradeIDs limits the positions to those these trades opened
	TradeIDs []string
	// IncludeCalculations adds the value and unrealised profit or loss of each position
	IncludeCalculations bool
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// OpenPosition holds an open margin position
type OpenPosition struct {
	OrderID    string `json:"ordertxid"`
	AssetClass string `json:"class"`
	// Status is open or closed
	Status string     `json:"posstatus"`
	Pair   string     `json:"pair"`
	Time   types.Time `json:"time"`
	// Side is buy or sell
	Side         string       `json:"type"`
	OrderType    string       `json:"ordertype"`
	Cost         types.Number `json:"cost"`
	Fee          types.Number `json:"fee"`
	Volume       types.Number `json:"vol"`
	VolumeClosed types.Number `json:"vol_closed"`
	// Margin is the initial margin consumed
	Margin types.Number `json:"margin"`
	// Value and UnrealisedPNL are those of the remaining position, only sent with IncludeCalculations
	Value         types.Number `json:"value"`
	UnrealisedPNL types.Number `json:"net"`
	// Terms describes the funding cost and term, such as "0.0100% per 4 hours"
	Terms            string     `json:"terms"`
	NextRolloverTime types.Time `json:"rollovertm"`
	Miscellaneous    string     `json:"misc"`
	// OrderFlags is a comma separated list of the opening order's flags
	OrderFlags string `json:"oflags"`
}

// ConsolidatedPosition holds the open margin positions of a pair
type ConsolidatedPosition struct {
	Pair       string `json:"pair"`
	AssetClass string `json:"class"`
	// Positions is the number of positions consolidated
	Positions types.Number `json:"positions"`
	// Side is buy or sell
	Side string `json:"type"`
	// Leverage is the positions' average leverage, or n/a
	Leverage     string       `json:"leverage"`
	Cost         types.Number `json:"cost"`
	Fee          types.Number `json:"fee"`
	Volume       types.Number `json:"vol"`
	VolumeClosed types.Number `json:"vol_closed"`
	// Margin is the initial margin consumed
	Margin types.Number `json:"margin"`
	// Value and UnrealisedPNL are those of the remaining positions, only sent with IncludeCalculations
	Value         types.Number `json:"value"`
	UnrealisedPNL types.Number `json:"net"`
}

// LedgersRequest holds the parameters of Get Ledgers Info
type LedgersRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	// Assets filters the entries to these currency class assets; every asset's are returned when it and
	// ClassifiedAssets are empty
	Assets []currency.Code
	// ClassifiedAssets filters the entries to assets of other classes in place of Assets, such as tokenized_asset
	// xStocks, whose names Kraken matches case sensitively
	ClassifiedAssets []ClassifiedAsset
	// AssetClass filters the entries by class the deprecated way, which ClassifiedAssets replaces
	AssetClass string
	// LedgerType filters the entries by type: all, the default, trade, deposit, withdrawal, transfer, margin,
	// adjustment, rollover, credit, settled, staking, dividend, sale or nft_rebate
	LedgerType string
	// Start and End bound the entries by time, exclusively and inclusively; StartLedgerID and EndLedgerID bound them
	// by an entry's time instead
	Start         time.Time
	End           time.Time
	StartLedgerID string
	EndLedgerID   string
	// Offset pages through the entries the deprecated way, which WithCursor replaces
	Offset uint64
	// WithCursor pages through the entries; set Cursor to the previous page's Cursor.Next to request the next page
	WithCursor bool
	Cursor     string
	// WithoutCount skips counting the matching entries, which is noticeably faster for an account with many
	WithoutCount bool
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// ClassifiedAsset names an asset, or a pair, together with its asset class
type ClassifiedAsset struct {
	Name string `json:"asset"`
	// AssetClass is currency, forex, equity, equity_pair, nft, derivatives, tokenized_asset or futures_contract
	AssetClass string `json:"aclass"`
}

// LedgersResponse holds a page of up to 50 ledger entries keyed by ledger ID
type LedgersResponse struct {
	Ledger map[string]LedgerEntry `json:"ledger"`
	// Count is the number of matching entries, 0 with WithoutCount or cursor pagination, for which Kraken sends null
	Count uint64 `json:"count"`
	// Cursor is only set with cursor pagination while more entries remain
	Cursor PageCursor `json:"cursor"`
}

// LedgerEntry holds a ledger entry
type LedgerEntry struct {
	// ReferenceID is the trade, deposit, withdrawal or other transaction that caused the entry
	ReferenceID string        `json:"refid"`
	Time        types.Time    `json:"time"`
	Type        string        `json:"type"`
	Subtype     string        `json:"subtype"`
	AssetClass  string        `json:"aclass"`
	Asset       currency.Code `json:"asset"`
	Amount      types.Number  `json:"amount"`
	Fee         types.Number  `json:"fee"`
	// Balance is the asset's balance after the entry
	Balance types.Number `json:"balance"`
	// AmountToken is Amount in a tokenised asset's raw base units, kept exact; only Query Ledgers sends it
	AmountToken types.PreciseNumber `json:"amount_token"`
}

// QueryLedgersRequest holds the parameters of Query Ledgers
type QueryLedgersRequest struct {
	// AccountID selects a wallet listed by ListWalletAccounts; the default wallet is used when it is empty
	AccountID string
	// LedgerIDs are the up to 20 entries to query
	LedgerIDs []string
	// IncludeTrades sends the deprecated trades flag, which Kraken says not to rely on
	IncludeTrades bool
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// TradeVolumeRequest holds the parameters of Get Trade Volume
type TradeVolumeRequest struct {
	// Pairs adds the fees of these forex pairs
	Pairs currency.Pairs
	// ClassifiedPairs adds the fees of pairs of other classes in place of Pairs, such as equity_pair or derivatives;
	// Kraken expresses each pair's amounts in its asset
	ClassifiedPairs []ClassifiedAsset
	// FeeInfo sends the legacy fee-info flag, which Kraken accepts for backwards compatibility
	FeeInfo bool
	// FeeSchedule adds each pair's full fee schedule
	FeeSchedule bool
	// RebaseMultiplier shows xStocks rebased, the default, in terms of the underlying equity, or base, in SPV tokens
	RebaseMultiplier string
}

// TradeVolumeResponse holds an account's 30 day trading volume and the fees of the requested pairs. Fees are
// percentages
type TradeVolumeResponse struct {
	// Currency is the asset Volume is expressed in
	Currency   currency.Code     `json:"currency"`
	AssetClass string            `json:"asset_class"`
	Volume     types.Number      `json:"volume"`
	Inputs     TradeVolumeInputs `json:"inputs"`
	// Fees holds each requested pair's taker fees, and FeesMaker the maker fees of a pair on a maker/taker schedule;
	// both are nil when no pair is requested
	Fees      map[string]TradeVolumeFee `json:"fees"`
	FeesMaker map[string]TradeVolumeFee `json:"fees_maker"`
	// Subaccounts is only sent to a domain's master account
	Subaccounts []TradeVolumeSubaccount `json:"volume_subaccounts"`
	// Schedules is only sent with FeeSchedule
	Schedules []TradeVolumeFeeSchedule `json:"schedules"`
}

// TradeVolumeInputs holds the domain wide values fee tiers are evaluated against
type TradeVolumeInputs struct {
	DomainSpotVolume30Day    types.Number `json:"domain_spot_volume_30d"`
	DomainFuturesVolume30Day types.Number `json:"domain_futures_volume_30d"`
	DomainAssetsOnPlatform   types.Number `json:"domain_assets_on_platform"`
}

// TradeVolumeFee holds a pair's current fee and volume tier
type TradeVolumeFee struct {
	Fee types.Number `json:"fee"`
	// MinimumFee is the fee of the highest volume tier and MaximumFee that of the lowest
	MinimumFee types.Number `json:"minfee"`
	MaximumFee types.Number `json:"maxfee"`
	// NextFee is the fee of the next volume tier, 0 at the lowest fee tier
	NextFee types.Number `json:"nextfee"`
	// TierVolume and TierFuturesVolume are the spot and futures volume thresholds of the current tier
	TierVolume        types.Number `json:"tiervolume"`
	TierFuturesVolume types.Number `json:"tierfuturesvolume"`
	// NextVolume and NextFuturesVolume are those of the next tier; NextVolume is 0 at the highest volume tier
	NextVolume        types.Number `json:"nextvolume"`
	NextFuturesVolume types.Number `json:"nextfuturesvolume"`
	// VolumeOffset is added to the trade volume when fees are computed
	VolumeOffset types.Number `json:"volumeoffset"`
}

// TradeVolumeSubaccount holds a subaccount's trade volume
type TradeVolumeSubaccount struct {
	// IIBAN is the subaccount's internal IBAN
	IIBAN  string       `json:"iiban"`
	Volume types.Number `json:"volume"`
}

// TradeVolumeFeeSchedule holds a pair's full fee schedule
type TradeVolumeFeeSchedule struct {
	Pair  string               `json:"pair"`
	Class string               `json:"class"`
	Tiers []TradeVolumeFeeTier `json:"tiers"`
}

// TradeVolumeFeeTier holds a fee schedule tier and the thresholds it requires
type TradeVolumeFeeTier struct {
	MakerFee          types.Number `json:"maker_fee"`
	TakerFee          types.Number `json:"taker_fee"`
	MinimumSpotVolume types.Number `json:"min_spot_volume"`
	// MinimumFuturesVolume is a 30 day volume
	MinimumFuturesVolume    types.Number `json:"min_futures_volume"`
	MinimumAssetsOnPlatform types.Number `json:"min_assets_on_platform"`
	// Active is set for the user's current tier
	Active bool `json:"active"`
}

// ExportReportRequest holds the parameters of Request Export Report
type ExportReportRequest struct {
	// Report is trades or ledgers
	Report string
	// Format is CSV, the default, or TSV
	Format      string
	Description string
	// Fields limits the report to these fields, every one when it is empty: ordertxid, time, ordertype, price, cost,
	// fee, vol, margin, misc and ledgers for trades, and refid, time, type, subtype, aclass, asset, amount, fee,
	// balance and wallet for ledgers
	Fields []string
	// Start defaults to the first of the current month and End to now
	Start time.Time
	End   time.Time
}

// ExportReportResponse holds a requested export report's ID
type ExportReportResponse struct {
	ID string `json:"id"`
}

// ExportReport holds an export report's status. Kraken deprecated Flags, ExpireTime and AssetClass
type ExportReport struct {
	ID          string `json:"id"`
	Description string `json:"descr"`
	Format      string `json:"format"`
	Report      string `json:"report"`
	Subtype     string `json:"subtype"`
	// Status is Queued, Processing or Processed
	Status string `json:"status"`
	// Error is why the report failed, such as EExport:Exported size too big
	Error       string     `json:"error"`
	Flags       string     `json:"flags"`
	Fields      string     `json:"fields"`
	CreatedTime types.Time `json:"createdtm"`
	ExpireTime  types.Time `json:"expiretm"`
	// StartTime and CompletedTime are when processing began and finished
	StartTime     types.Time `json:"starttm"`
	CompletedTime types.Time `json:"completedtm"`
	// DataStartTime and DataEndTime bound the report's data
	DataStartTime types.Time `json:"datastarttm"`
	DataEndTime   types.Time `json:"dataendtm"`
	AssetClass    string     `json:"aclass"`
	Asset         string     `json:"asset"`
	AssetClasses  []string   `json:"asset_classes"`
	EndTime       types.Time `json:"endtm"`
	// PendingDeletion is set while the report waits to be deleted
	PendingDeletion bool `json:"delete"`
}

// DeleteExportReportResponse holds whether an export report was deleted or cancelled
type DeleteExportReportResponse struct {
	Deleted   bool `json:"delete"`
	Cancelled bool `json:"cancel"`
}

// APIKeyInfoRequest holds the parameters of Get API Key Info
type APIKeyInfoRequest struct {
	// OneTimePassword is the two-factor password, only needed when the key has two-factor authentication
	OneTimePassword string
}

// APIKeyInfoResponse holds the details of the API key that signed the request
type APIKeyInfoResponse struct {
	Name string `json:"api_key_name"`
	Key  string `json:"api_key"`
	// Nonce is the key's current nonce, decoded as an integer since a nanosecond nonce exceeds float64's precision
	Nonce uint64 `json:"nonce,string"`
	// NonceWindow is the key's custom nonce window, 0 when it has none
	NonceWindow uint64 `json:"nonce_window"`
	// Permissions are values such as query-funds, withdraw-funds, query-open-trades and modify-trades
	Permissions []string `json:"permissions"`
	// IIBAN is the internal IBAN of the key's account
	IIBAN string `json:"iban"`
	// ValidUntil, QueryFrom and QueryTo are zero when not set
	ValidUntil   types.Time `json:"valid_until"`
	QueryFrom    types.Time `json:"query_from"`
	QueryTo      types.Time `json:"query_to"`
	CreatedTime  types.Time `json:"created_time"`
	ModifiedTime types.Time `json:"modified_time"`
	// IPAllowlist is empty when any IP address may use the key
	IPAllowlist []string `json:"ip_allowlist"`
	// LastUsed is zero for a key never used
	LastUsed types.Time `json:"last_used"`
}

// apiKeyInfoCamelCaseNames maps the camelCase names of Kraken's Get API Key Info example response to the snake_case
// names its schema documents
var apiKeyInfoCamelCaseNames = map[string]string{
	"apiKeyName":   "api_key_name",
	"apiKey":       "api_key",
	"nonceWindow":  "nonce_window",
	"validUntil":   "valid_until",
	"queryFrom":    "query_from",
	"queryTo":      "query_to",
	"createdTime":  "created_time",
	"modifiedTime": "modified_time",
	"ipAllowlist":  "ip_allowlist",
	"lastUsed":     "last_used",
}

// UnmarshalJSON decodes the snake_case names Kraken's schema documents, or the camelCase names of its example
// response, since the two disagree
func (a *APIKeyInfoResponse) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for camelCase, snakeCase := range apiKeyInfoCamelCaseNames {
		if value, ok := fields[camelCase]; ok {
			delete(fields, camelCase)
			fields[snakeCase] = value
		}
	}
	normalised, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	type plain APIKeyInfoResponse
	return json.Unmarshal(normalised, (*plain)(a))
}

// WalletAccountsResponse holds the authenticated user's wallet accounts
type WalletAccountsResponse struct {
	Accounts []WalletAccount `json:"accounts"`
	// Cursor's Next is empty when there are no further pages; Kraken documents no parameter that requests them
	Cursor PageCursor `json:"cursor"`
}

// WalletAccount holds a wallet account, whose AccountID selects it on the endpoints that take one
type WalletAccount struct {
	AccountID string             `json:"account_id"`
	Flags     WalletAccountFlags `json:"flags"`
	// Status is active, disabled, closed or unknown
	Status string `json:"status"`
	// Type is main, spot, pay, prop_paper, prop_real or unknown
	Type string `json:"type"`
	Name string `json:"name"`
}

// WalletAccountFlags holds a wallet account's flags
type WalletAccountFlags struct {
	// UserDefined is set for a wallet the user created rather than the system
	UserDefined bool `json:"user_defined"`
	Active      bool `json:"active"`
}
