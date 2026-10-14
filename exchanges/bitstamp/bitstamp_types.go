package bitstamp

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// TransactionType is the type of user transaction
type TransactionType uint8

// User transaction types
const (
	TransactionTypeDeposit                       TransactionType = 0
	TransactionTypeWithdrawal                    TransactionType = 1
	TransactionTypeMarketTrade                   TransactionType = 2
	TransactionTypeSubAccountTransfer            TransactionType = 14
	TransactionTypeStakingCredit                 TransactionType = 25
	TransactionTypeSentToStaking                 TransactionType = 26
	TransactionTypeStakingReward                 TransactionType = 27
	TransactionTypeReferralReward                TransactionType = 32
	TransactionTypeSettlementTransfer            TransactionType = 33
	TransactionTypeInterAccountTransfer          TransactionType = 35
	TransactionTypeSimpleBuy                     TransactionType = 36
	TransactionTypeSmallBalanceConversion        TransactionType = 53
	TransactionTypeSmallBalanceConversionAlt     TransactionType = 55
	TransactionTypeDerivativesPeriodicSettlement TransactionType = 58
	TransactionTypeInsuranceFundClaim            TransactionType = 59
	TransactionTypeInsuranceFundPremium          TransactionType = 60
	TransactionTypeCollateralLiquidation         TransactionType = 61
)

// Market types
const (
	MarketTypeSpot      = "SPOT"
	MarketTypePerpetual = "PERPETUAL"
)

// Margin modes used by derivatives markets
const (
	MarginModeCross    = "CROSS"
	MarginModeIsolated = "ISOLATED"
)

// Order subtypes used by derivatives markets
const (
	OrderSubtypeLimit                   = "LIMIT"
	OrderSubtypeMarket                  = "MARKET"
	OrderSubtypeInstant                 = "INSTANT"
	OrderSubtypeCash                    = "CASH"
	OrderSubtypeStopMarket              = "STOP_MARKET"
	OrderSubtypeStopLimit               = "STOP_LIMIT"
	OrderSubtypeStopLoss                = "STOP_LOSS"
	OrderSubtypeTakeProfit              = "TAKE_PROFIT"
	OrderSubtypeStopLossLimit           = "STOP_LOSS_LIMIT"
	OrderSubtypeTakeProfitLimit         = "TAKE_PROFIT_LIMIT"
	OrderSubtypeTrailingStopLoss        = "TRAILING_STOP_LOSS"
	OrderSubtypeTrailingTakeProfit      = "TRAILING_TAKE_PROFIT"
	OrderSubtypeTrailingStopLossLimit   = "TRAILING_STOP_LOSS_LIMIT"
	OrderSubtypeTrailingTakeProfitLimit = "TRAILING_TAKE_PROFIT_LIMIT"
)

// Stop order trigger price types
const (
	TriggerLastTradedPrice = "LAST_TRADED_PRICE"
	TriggerIndexPrice      = "INDEX_PRICE"
	TriggerMarkPrice       = "MARK_PRICE"
)

// Order event sources used for websocket gap recovery
const (
	OrderSourceOrderbook           = "orderbook"
	OrderSourceStopOrder           = "stop_order"
	OrderSourceTokenisedSecurities = "tokenized_securities" //nolint:gosec // G101 false positive; an order source rather than a credential
)

// Public transaction time periods
const (
	TransactionPeriodMinute = "minute"
	TransactionPeriodHour   = "hour"
	TransactionPeriodDay    = "day"
)

// Bank withdrawal types
const (
	BankWithdrawalTypeSEPA          = "sepa"
	BankWithdrawalTypeInternational = "international"
)

// Earn products, terms and settings
const (
	EarnTypeStaking   = "STAKING"
	EarnTypeLending   = "LENDING"
	EarnTermFlexible  = "FLEXIBLE"
	EarnTermFixed     = "FIXED"
	EarnSettingOptIn  = "OPT_IN"
	EarnSettingOptOut = "OPT_OUT"
)

// Sort orders
const (
	SortAscending  = "asc"
	SortDescending = "desc"
)

// Market broker entities used to filter markets
const (
	EntityUKLtd     = "UK_LTD"
	EntityEuropeSA  = "EUROPE_SA"
	EntityUSInc     = "US_INC"
	EntityBVILtd    = "BVI_LTD"
	EntityEuropeBEL = "EUROPE_BEL"
	EntityUK        = "UK"
	EntitySIBroker  = "SI_BROKER"
	EntitySIMTF     = "SI_MTF"
	EntityAsiaSG    = "ASIA_SG"
)

// Availability of market trading and of currency or network deposits and withdrawals
const (
	StatusEnabled  = "Enabled"
	StatusDisabled = "Disabled"
)

// Currency types
const (
	CurrencyTypeCrypto = "crypto"
	CurrencyTypeFiat   = "fiat"
)

// Market asset classes
const (
	AssetClassCrypto      = "CRYPTO"
	AssetClassETF         = "ETF"
	AssetClassCommodities = "COMMODITIES"
	AssetClassFX          = "FX"
)

// Order statuses returned by the order status endpoint
const (
	OrderStatusOpen      = "Open"
	OrderStatusFinished  = "Finished"
	OrderStatusExpired   = "Expired"
	OrderStatusCancelled = "Canceled"
)

// Statuses of cancelled orders
const (
	CancelStatusCancelled = "Canceled"
	CancelStatusPending   = "Cancel pending"
)

// Instant convert address formats
const (
	AddressFormatP2SHP2WSH = "P2SHP2WSH"
	AddressFormatBECH32    = "BECH32"
)

// Travel Rule identity document types
const (
	IDTypeDriversLicence = "drivers_licence"
	IDTypeIDCard         = "id_card"
	IDTypePassport       = "passport"
	IDTypeMisc           = "misc"
)

// Cryptocurrency deposit review statuses and the reasons a review is pending
const (
	DepositStatusPending                          = "PENDING"
	DepositStatusFinalised                        = "FINALIZED"
	DepositStatusInProcessing                     = "IN_PROCESSING"
	DepositStatusReverted                         = "REVERTED"
	DepositStatusRejected                         = "REJECTED"
	DepositPendingReasonAddressVerificationNeeded = "ADDRESS_VERIFICATION_NEEDED"
	DepositPendingReasonAdditionalInfoMissing     = "ADDITIONAL_INFORMATION_MISSING"
)

// Satoshi test statuses
const (
	SatoshiTestStatusPending  = "pending"
	SatoshiTestStatusApproved = "approved"
	SatoshiTestStatusExpired  = "expired"
)

// Extended public key registration statuses
const (
	XpubRegistrationStatusPending = "PENDING"
	XpubRegistrationStatusActive  = "ACTIVE"
	XpubRegistrationStatusRevoked = "REVOKED"
	XpubRegistrationStatusFailed  = "FAILED"
)

// Earn transaction types and statuses
const (
	EarnTransactionTypeStake          = "STAKE"
	EarnTransactionTypeUnstake        = "UNSTAKE"
	EarnTransactionTypeRewardReceived = "REWARD_RECEIVED"
	EarnTransactionTypeOptIn          = "OPT_IN"
	EarnTransactionTypeOptOut         = "OPT_OUT"
	EarnTransactionTypeSubscribe      = "SUBSCRIBE"
	EarnTransactionTypeUnsubscribe    = "UNSUBSCRIBE"
	EarnTransactionTypeAllocation     = "ALLOCATION"
	EarnTransactionTypeDeallocation   = "DEALLOCATION"
	EarnTransactionStatusPending      = "PENDING"
	EarnTransactionStatusCompleted    = "COMPLETED"
	EarnTransactionStatusCancelled    = "CANCELED"
	EarnTransactionStatusFailed       = "FAILED"
)

// WithdrawalType is the type of a withdrawal request
// Cryptocurrency withdrawals use a type number per currency, so only the bank withdrawal types are named
type WithdrawalType uint64

// Bank withdrawal request types
const (
	WithdrawalTypeSEPA WithdrawalType = 0
	WithdrawalTypeWire WithdrawalType = 2
)

// WithdrawalStatus is the status of a withdrawal request
type WithdrawalStatus uint64

// Withdrawal request statuses
const (
	WithdrawalStatusOpen      WithdrawalStatus = 0
	WithdrawalStatusInProcess WithdrawalStatus = 1
	WithdrawalStatusFinished  WithdrawalStatus = 2
	WithdrawalStatusCancelled WithdrawalStatus = 3
	WithdrawalStatusFailed    WithdrawalStatus = 4
	WithdrawalStatusReversed  WithdrawalStatus = 11
)

// OrderbookGrouping determines how orders are aggregated in orderbook responses
type OrderbookGrouping uint8

// Orderbook groupings
const (
	OrderbookUngrouped OrderbookGrouping = iota
	OrderbookGroupedByPrice
	OrderbookUngroupedWithOrderIDs
)

// CurrencyResponse holds information for a listed currency
type CurrencyResponse struct {
	Name            string            `json:"name"`
	Currency        currency.Code     `json:"currency"`
	Type            string            `json:"type"`
	Symbol          string            `json:"symbol"`
	Decimals        uint8             `json:"decimals"`
	Logo            string            `json:"logo"`
	AvailableSupply types.Number      `json:"available_supply"`
	Deposit         string            `json:"deposit"`
	Withdrawal      string            `json:"withdrawal"`
	Networks        []CurrencyNetwork `json:"networks"`
}

// CurrencyNetwork holds deposit and withdrawal details for a currency network
type CurrencyNetwork struct {
	Network                 string       `json:"network"`
	WithdrawalMinimumAmount types.Number `json:"withdrawal_minimum_amount"`
	WithdrawalDecimals      uint8        `json:"withdrawal_decimals"`
	Deposit                 string       `json:"deposit"`
	Withdrawal              string       `json:"withdrawal"`
}

// TickerResponse holds ticker information for a market
type TickerResponse struct {
	Last                       types.Number `json:"last"`
	High                       types.Number `json:"high"`
	Low                        types.Number `json:"low"`
	VolumeWeightedAveragePrice types.Number `json:"vwap"`
	Volume                     types.Number `json:"volume"`
	BestBid                    types.Number `json:"bid"`
	BestAsk                    types.Number `json:"ask"`
	Timestamp                  types.Time   `json:"timestamp"`
	Open                       types.Number `json:"open"`
	Open24Hour                 types.Number `json:"open_24"`
	PercentChange24Hour        types.Number `json:"percent_change_24"`
	Side                       OrderSide    `json:"side"`
	MarketType                 string       `json:"market_type"`
	MarkPrice                  types.Number `json:"mark_price"`
	IndexPrice                 types.Number `json:"index_price"`
	OpenInterest               types.Number `json:"open_interest"`
	OpenInterestValue          types.Number `json:"open_interest_value"`
	// Market is only populated by the all markets ticker endpoint
	Market string `json:"market"`
}

// OrderbookResponse holds orderbook information
type OrderbookResponse struct {
	Timestamp      types.Time       `json:"timestamp"`
	Microtimestamp types.Time       `json:"microtimestamp"`
	Bids           []OrderbookLevel `json:"bids"`
	Asks           []OrderbookLevel `json:"asks"`
}

// OrderbookLevel holds an orderbook price level
// OrderID is only populated when the orderbook is requested with OrderbookUngroupedWithOrderIDs
type OrderbookLevel struct {
	Price   float64
	Amount  float64
	OrderID uint64
}

// TransactionResponse holds a public market trade
type TransactionResponse struct {
	Date    types.Time   `json:"date"`
	TradeID uint64       `json:"tid,string"`
	Price   types.Number `json:"price"`
	Side    OrderSide    `json:"type"`
	Amount  types.Number `json:"amount"`
}

// MarketResponse holds market information
type MarketResponse struct {
	Name                        string        `json:"name"`
	MarketSymbol                string        `json:"market_symbol"`
	BaseCurrency                currency.Code `json:"base_currency"`
	BaseDecimals                uint8         `json:"base_decimals"`
	CounterCurrency             currency.Code `json:"counter_currency"`
	CounterDecimals             uint8         `json:"counter_decimals"`
	MinimumOrderValue           types.Number  `json:"minimum_order_value"`
	MaximumOrderValue           types.Number  `json:"maximum_order_value"`
	MinimumOrderAmount          types.Number  `json:"minimum_order_amount"`
	MaximumOrderAmount          types.Number  `json:"maximum_order_amount"`
	Trading                     string        `json:"trading"`
	InstantOrderCounterDecimals uint8         `json:"instant_order_counter_decimals"`
	InstantAndMarketOrders      string        `json:"instant_and_market_orders"`
	Description                 string        `json:"description"`
	MarketType                  string        `json:"market_type"`
	UnderlyingAsset             string        `json:"underlying_asset"`
	PayoffType                  string        `json:"payoff_type"`
	ContractSize                types.Number  `json:"contract_size"`
	MaxLeverage                 types.Number  `json:"max_leverage"`
	TickSize                    types.Number  `json:"tick_size"`
	ISIN                        string        `json:"isin"`
	AssetClass                  string        `json:"asset_class"`
	HasMarketHours              bool          `json:"has_market_hours"`
	Exchange                    string        `json:"exchange"`
}

// OHLCRequest holds the parameters for fetching candle data
// When both Start and End are set the API ignores Start and returns up to Limit candles ending at End
type OHLCRequest struct {
	Pair                 currency.Pair
	Step                 kline.Interval
	Limit                uint64
	Start                time.Time
	End                  time.Time
	ExcludeCurrentCandle bool
}

// OHLCResponse holds candle data
type OHLCResponse struct {
	Data OHLCData `json:"data"`
}

// OHLCData holds the candles for a market
type OHLCData struct {
	// Pair is only populated for spot markets and Market is only populated for perpetual markets
	Pair   string   `json:"pair"`
	Market string   `json:"market"`
	OHLC   []Candle `json:"ohlc"`
}

// Candle holds an individual candle
type Candle struct {
	Timestamp types.Time   `json:"timestamp"`
	Open      types.Number `json:"open"`
	High      types.Number `json:"high"`
	Low       types.Number `json:"low"`
	Close     types.Number `json:"close"`
	Volume    types.Number `json:"volume"`
}

// EURUSDConversionRateResponse holds the EUR/USD buy and sell conversion rates
type EURUSDConversionRateResponse struct {
	Buy  types.Number `json:"buy"`
	Sell types.Number `json:"sell"`
}

// FundingRateResponse holds the current funding rate for a perpetual market
type FundingRateResponse struct {
	FundingRate     types.Number `json:"funding_rate"`
	Timestamp       types.Time   `json:"timestamp"`
	Market          string       `json:"market"`
	NextFundingTime types.Time   `json:"next_funding_time"`
}

// FundingRateHistoryRequest holds the parameters for fetching funding rate history
type FundingRateHistoryRequest struct {
	Pair  currency.Pair
	Limit uint64
	Since time.Time
	Until time.Time
}

// FundingRateHistoryResponse holds the funding rate history for a perpetual market
type FundingRateHistoryResponse struct {
	Market             string        `json:"market"`
	FundingRateHistory []FundingRate `json:"funding_rate_history"`
}

// FundingRate holds a historical funding rate
type FundingRate struct {
	FundingRate types.Number `json:"funding_rate"`
	Timestamp   types.Time   `json:"timestamp"`
}

// PaginationRequest holds the paging parameters of list endpoints
type PaginationRequest struct {
	Page    uint64
	PerPage uint64
}

// VASPListResponse holds Virtual Asset Service Providers used for Travel Rule compliance
type VASPListResponse struct {
	Data       []VASP     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// VASP holds a Virtual Asset Service Provider
type VASP struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// Pagination holds paging information
type Pagination struct {
	Page  uint64 `json:"page"`
	Size  uint64 `json:"size"`
	Count uint64 `json:"count"`
}

// AccountBalanceResponse holds the balance of a currency
type AccountBalanceResponse struct {
	Currency  currency.Code `json:"currency"`
	Total     types.Number  `json:"total"`
	Available types.Number  `json:"available"`
	Reserved  types.Number  `json:"reserved"`
}

// TradingFeeResponse holds the trading fees for a market
type TradingFeeResponse struct {
	Market string         `json:"market"`
	Fees   MakerTakerFees `json:"fees"`
}

// MakerTakerFees holds maker and taker fee percentages
type MakerTakerFees struct {
	Maker types.Number `json:"maker"`
	Taker types.Number `json:"taker"`
}

// WithdrawalFeeResponse holds the withdrawal fee for a currency network
type WithdrawalFeeResponse struct {
	Currency currency.Code `json:"currency"`
	Fee      types.Number  `json:"fee"`
	Network  string        `json:"network"`
}

// OrderStatusRequest holds the parameters for fetching an order's status
// Either OrderID or ClientOrderID must be supplied
type OrderStatusRequest struct {
	OrderID          uint64
	ClientOrderID    string
	OmitTransactions bool
}

// OrderStatusResponse holds the status of an order
type OrderStatusResponse struct {
	ID              uint64             `json:"id"`
	DateTime        types.DateTime     `json:"datetime"`
	Side            OrderSide          `json:"type"`
	Subtype         string             `json:"subtype"`
	Status          string             `json:"status"`
	Market          string             `json:"market"`
	Transactions    []OrderTransaction `json:"transactions"`
	AmountRemaining types.Number       `json:"amount_remaining"`
	ClientOrderID   string             `json:"client_order_id"`
	MarginMode      string             `json:"margin_mode"`
	Leverage        types.Number       `json:"leverage"`
	StopPrice       types.Number       `json:"stop_price"`
	Trigger         string             `json:"trigger"`
	ActivationPrice types.Number       `json:"activation_price"`
	TrailingDelta   uint64             `json:"trailing_delta"`
}

// OrderTransaction holds a transaction which executed part of an order
// Amounts holds the signed amount of each currency involved in the transaction, keyed by currency
type OrderTransaction struct {
	TradeID  uint64
	Price    float64
	Fee      float64
	DateTime time.Time
	Type     TransactionType
	Amounts  map[currency.Code]float64
}

// OrderEventsRequest holds the parameters for fetching order events for websocket gap recovery
type OrderEventsRequest struct {
	Pair    currency.Pair
	SinceID string
	UntilID string
	// OrderSource is only used when fetching account order events
	OrderSource string
}

// OrderEventResponse holds an order event used for websocket gap recovery
type OrderEventResponse struct {
	Event       string `json:"event"`
	EventID     string `json:"event_id"`
	OrderSource string `json:"order_source"`
	// TradeAccountID is only populated for account order events and is 0 for the main account
	TradeAccountID uint64         `json:"trade_account_id"`
	Data           OrderEventData `json:"data"`
}

// OrderEventData holds the order details of an order event, from the websocket order channels or their REST recovery
// RemainingAmount is the amount left to execute, and AmountTraded is the amount executed by the event rather than in total
type OrderEventData struct {
	ID              uint64       `json:"id"`
	Side            OrderSide    `json:"order_type"`
	OrderSubtype    uint8        `json:"order_subtype"`
	DateTime        types.Time   `json:"datetime"`
	Microtimestamp  types.Time   `json:"microtimestamp"`
	RemainingAmount types.Number `json:"amount_str"`
	AmountTraded    types.Number `json:"amount_traded"`
	AmountAtCreate  types.Number `json:"amount_at_create"`
	Price           types.Number `json:"price_str"`
	IsLiquidation   bool         `json:"is_liquidation"`
	// The following fields are only populated for account order events
	TradeAccountID  uint64       `json:"trade_account_id"`
	ClientOrderID   string       `json:"client_order_id"`
	OriginalOrderID uint64       `json:"orig_order_id"`
	ReduceOnly      bool         `json:"reduce_only"`
	StopPrice       types.Number `json:"stop_price"`
	ActivationPrice types.Number `json:"activation_price"`
	TrailingDelta   types.Number `json:"trailing_delta"`
}

// OpenOrderResponse holds an open order
type OpenOrderResponse struct {
	ID              uint64         `json:"id,string"`
	DateTime        types.DateTime `json:"datetime"`
	Side            OrderSide      `json:"type"`
	Subtype         string         `json:"subtype"`
	Price           types.Number   `json:"price"`
	RemainingAmount types.Number   `json:"amount"`
	AmountAtCreate  types.Number   `json:"amount_at_create"`
	Market          string         `json:"market"`
	LimitPrice      types.Number   `json:"limit_price"`
	ClientOrderID   string         `json:"client_order_id"`
	MarginMode      string         `json:"margin_mode"`
	Leverage        types.Number   `json:"leverage"`
	ReservedMargin  types.Number   `json:"reserved_margin"`
	StopPrice       types.Number   `json:"stop_price"`
	Trigger         string         `json:"trigger"`
	ActivationPrice types.Number   `json:"activation_price"`
	TrailingDelta   uint64         `json:"trailing_delta"`
	ReduceOnly      bool           `json:"reduce_only"`
}

// LimitOrderRequest holds the parameters for placing a limit order
// MarginMode and Leverage are required for derivatives markets. At most one of DailyOrder, ImmediateOrCancel,
// FillOrKill, MakerOrCancel and GoodTillDate may be set, and none of them may be combined with LimitPrice
type LimitOrderRequest struct {
	Pair              currency.Pair
	Side              order.Side
	Amount            float64
	Price             float64
	LimitPrice        float64
	DailyOrder        bool
	ImmediateOrCancel bool
	FillOrKill        bool
	MakerOrCancel     bool
	GoodTillDate      bool
	ExpireTime        time.Time
	ClientOrderID     string
	Subtype           string
	MarginMode        string
	Leverage          float64
	StopPrice         float64
	Trigger           string
	ActivationPrice   float64
	TrailingDelta     uint64
	ReduceOnly        bool
}

// InstantOrderRequest holds the parameters for placing an instant order
// Buy amounts are in the counter currency; sell amounts are in the base currency unless AmountInCounter is set
// MarginMode and Leverage are required for derivatives buy orders, and ReduceOnly is only supported by sell orders
// whose amount is in the base currency
type InstantOrderRequest struct {
	Pair            currency.Pair
	Side            order.Side
	Amount          float64
	AmountInCounter bool
	ClientOrderID   string
	MarginMode      string
	Leverage        float64
	ReduceOnly      bool
}

// MarketOrderRequest holds the parameters for placing a market order
// MarginMode and Leverage are required for derivatives markets
type MarketOrderRequest struct {
	Pair            currency.Pair
	Side            order.Side
	Amount          float64
	ClientOrderID   string
	Subtype         string
	MarginMode      string
	Leverage        float64
	StopPrice       float64
	Trigger         string
	ActivationPrice float64
	TrailingDelta   uint64
	ReduceOnly      bool
}

// OrderResponse holds the details of a newly placed order
type OrderResponse struct {
	ID              uint64         `json:"id,string"`
	Subtype         string         `json:"subtype"`
	Market          string         `json:"market"`
	DateTime        types.DateTime `json:"datetime"`
	Side            OrderSide      `json:"type"`
	Price           types.Number   `json:"price"`
	Amount          types.Number   `json:"amount"`
	ClientOrderID   string         `json:"client_order_id"`
	MarginMode      string         `json:"margin_mode"`
	Leverage        types.Number   `json:"leverage"`
	StopPrice       types.Number   `json:"stop_price"`
	Trigger         string         `json:"trigger"`
	ActivationPrice types.Number   `json:"activation_price"`
	TrailingDelta   uint64         `json:"trailing_delta"`
}

// CancelOrderRequest holds the parameters for cancelling an order
// Either OrderID or ClientOrderID must be supplied
type CancelOrderRequest struct {
	OrderID       uint64
	ClientOrderID string
}

// CancelOrderResponse holds the details of a cancelled order
type CancelOrderResponse struct {
	ID     uint64       `json:"id"`
	Amount types.Number `json:"amount"`
	Price  types.Number `json:"price"`
	Side   OrderSide    `json:"type"`
	Market string       `json:"market"`
	Status string       `json:"status"`
}

// CancelAllOrdersResponse holds the result of cancelling all open orders
type CancelAllOrdersResponse struct {
	Cancelled []CancelledOrder `json:"canceled"`
	Success   bool             `json:"success"`
}

// CancelledOrder holds an order cancelled by a cancel all orders request
type CancelledOrder struct {
	ID     uint64       `json:"id"`
	Amount types.Number `json:"amount"`
	Price  types.Number `json:"price"`
	Side   OrderSide    `json:"type"`
	Market string       `json:"market"`
}

// ReplaceOrderRequest holds the parameters for replacing an order
// Either OrderID or OriginalClientOrderID must be supplied
type ReplaceOrderRequest struct {
	OrderID               uint64
	OriginalClientOrderID string
	ClientOrderID         string
	Amount                float64
	Price                 float64
}

// ReplaceOrderResponse holds the details of a replacement order
type ReplaceOrderResponse struct {
	OrderID               uint64       `json:"order_id"`
	Side                  OrderSide    `json:"order_type"`
	Market                string       `json:"market"`
	Amount                types.Number `json:"amount"`
	Price                 types.Number `json:"price"`
	DateTime              DateTime     `json:"datetime"`
	OriginalOrderID       uint64       `json:"orig_order_id"`
	OriginalClientOrderID string       `json:"orig_client_order_id"`
	Status                string       `json:"status"`
}

// TradingMarketResponse holds a market which can be traded by the account
type TradingMarketResponse struct {
	Name      string `json:"name"`
	URLSymbol string `json:"url_symbol"`
}

// MaxOrderAmountRequest holds the parameters for estimating the maximum order amount for a derivatives market
// AdditionalCollateral holds collateral amounts to include in the estimate as if they were already posted
type MaxOrderAmountRequest struct {
	Pair                 currency.Pair
	MarginMode           string
	Leverage             float64
	OrderType            string
	Side                 order.Side
	Price                float64
	StopPrice            float64
	ActivationPrice      float64
	TrailingDelta        uint64
	AdditionalCollateral map[currency.Code]float64
}

// MaxOrderAmountResponse holds the maximum order amount and value for a derivatives market
type MaxOrderAmountResponse struct {
	MaximumOrderAmount         types.Number  `json:"maximum_order_amount"`
	MaximumOrderValue          types.Number  `json:"maximum_order_value"`
	MaximumOrderAmountCurrency currency.Code `json:"maximum_order_amount_currency"`
	MaximumOrderValueCurrency  currency.Code `json:"maximum_order_value_currency"`
}

// WithdrawalRequestsRequest holds the parameters for fetching withdrawal requests
// TimeDelta returns withdrawal requests created within the supplied duration, between 1 and 50000000 seconds
type WithdrawalRequestsRequest struct {
	ID        uint64
	TimeDelta time.Duration
	Limit     uint64
	Offset    uint64
}

// WithdrawalRequestResponse holds a withdrawal request
// BitstampTransactionID is Bitstamp's own transaction ID, while TransactionID is the blockchain transaction ID of a
// cryptocurrency withdrawal and is empty until the withdrawal has been processed
type WithdrawalRequestResponse struct {
	ID                    uint64           `json:"id"`
	DateTime              types.DateTime   `json:"datetime"`
	Type                  WithdrawalType   `json:"type"`
	Currency              currency.Code    `json:"currency"`
	Network               string           `json:"network"`
	Amount                types.Number     `json:"amount"`
	Status                WithdrawalStatus `json:"status"`
	BitstampTransactionID uint64           `json:"txid"`
	Address               string           `json:"address"`
	TransactionID         string           `json:"transaction_id"`
}

// BankWithdrawalRequest holds the parameters for a SEPA or international bank withdrawal
// Name, Address, PostalCode, City and Country describe the account holder, and the Bank fields describe the target bank
type BankWithdrawalRequest struct {
	Amount          float64
	AccountCurrency currency.Code
	Name            string
	IBAN            string
	BIC             string
	Address         string
	PostalCode      string
	City            string
	Country         string
	Type            string
	Comment         string

	// The following fields only apply to international withdrawals
	BankName                       string
	BankAddress                    string
	BankPostalCode                 string
	BankCity                       string
	BankCountry                    string
	Currency                       currency.Code
	IntermediaryRoutingNumberOrBIC string
}

// BankWithdrawalResponse holds the ID of a bank withdrawal request
type BankWithdrawalResponse struct {
	WithdrawalID uint64 `json:"withdrawal_id"`
}

// CancelWithdrawalResponse holds the details of a cancelled withdrawal request
type CancelWithdrawalResponse struct {
	ID              uint64        `json:"id,string"`
	Amount          types.Number  `json:"amount"`
	Currency        currency.Code `json:"currency"`
	AccountCurrency currency.Code `json:"account_currency"`
	Type            string        `json:"type"`
}

// WithdrawalStatusResponse holds the status of a bank withdrawal request
type WithdrawalStatusResponse struct {
	Status string `json:"status"`
}

// CustomerInfo holds Travel Rule details for a retail or corporate customer
type CustomerInfo struct {
	RetailInfo    *RetailInfo    `json:"retail_info,omitempty"`
	CorporateInfo *CorporateInfo `json:"corporate_info,omitempty"`
}

// RetailInfo holds Travel Rule details for a retail customer
// FirstName and LastName are required; DateOfBirth is sent as a date only
type RetailInfo struct {
	FirstName            string
	LastName             string
	DateOfBirth          time.Time
	PlaceOfBirth         string
	IDType               string
	IDNumber             string
	StreetAndHouseNumber string
	City                 string
	PostalCode           string
	Country              string
}

// CorporateInfo holds Travel Rule details for a corporate customer
// CompanyName is required
type CorporateInfo struct {
	CompanyName string `json:"company_name"`
	Address     string `json:"address,omitempty"`
	Country     string `json:"country,omitempty"`
	LEI         string `json:"lei,omitempty"`
	CompanyID   string `json:"company_id,omitempty"`
}

// CryptoWithdrawalRequest holds the parameters for a cryptocurrency withdrawal
type CryptoWithdrawalRequest struct {
	Currency        currency.Code
	Network         string
	Amount          float64
	Address         string
	MemoID          string
	DestinationTag  string
	TransferID      uint64
	OriginatorInfo  *CustomerInfo
	BeneficiaryInfo *CustomerInfo
	// BeneficiaryThirdParty is sent when set, and must be false when withdrawing to an address in the account holder's name
	BeneficiaryThirdParty *bool
	BeneficiaryID         string
	VASPUUID              string
}

// RippleIOUWithdrawalRequest holds the parameters for a Ripple IOU withdrawal
type RippleIOUWithdrawalRequest struct {
	Currency        currency.Code
	Amount          float64
	Address         string
	OriginatorInfo  *CustomerInfo
	BeneficiaryInfo *CustomerInfo
	// BeneficiaryThirdParty is sent when set, and must be false when withdrawing to an address in the account holder's name
	BeneficiaryThirdParty *bool
	BeneficiaryID         string
	VASPUUID              string
}

// CryptoWithdrawalResponse holds the ID of a cryptocurrency withdrawal request
type CryptoWithdrawalResponse struct {
	ID uint64 `json:"id"`
}

// DepositAddressResponse holds a cryptocurrency deposit address
type DepositAddressResponse struct {
	Address        string `json:"address"`
	MemoID         string `json:"memo_id"`
	DestinationTag uint64 `json:"destination_tag"`
	TransferID     uint64 `json:"transfer_id"`
}

// UnconfirmedDepositResponse holds an unconfirmed bitcoin deposit
type UnconfirmedDepositResponse struct {
	Amount         types.Number `json:"amount"`
	Address        string       `json:"address"`
	Confirmations  uint64       `json:"confirmations"`
	MemoID         string       `json:"memo_id"`
	DestinationTag uint64       `json:"destination_tag"`
	TransferID     uint64       `json:"transfer_id"`
}

// RippleIOUDepositAddressResponse holds a Ripple IOU deposit address
type RippleIOUDepositAddressResponse struct {
	Address        string `json:"address"`
	DestinationTag uint64 `json:"destination_tag"`
}

// TransferRequest holds the parameters for transferring balances between the main account and a sub account
type TransferRequest struct {
	Amount     float64
	Currency   currency.Code
	SubAccount uint64
}

// InstantConvertAddressResponse holds an instant convert address
type InstantConvertAddressResponse struct {
	Address string `json:"address"`
}

// InstantConvertAddressInfoResponse holds the transactions of an instant convert address
type InstantConvertAddressInfoResponse struct {
	Address      string                      `json:"address"`
	CurrencyPair currency.Pair               `json:"currency_pair"`
	Transactions []InstantConvertTransaction `json:"transactions"`
}

// InstantConvertTransaction holds a conversion order for an instant convert address
type InstantConvertTransaction struct {
	OrderID uint64                `json:"order_id"`
	Count   uint64                `json:"count"`
	Trades  []InstantConvertTrade `json:"trades"`
}

// InstantConvertTrade holds a trade executed by an instant convert order
type InstantConvertTrade struct {
	ExchangeRate types.Number `json:"exchange_rate"`
	BTCAmount    types.Number `json:"btc_amount"`
	Fees         types.Number `json:"fees"`
}

// WebsocketTokenResponse holds the token and user ID for subscribing to private websocket channels
// The token is only valid for ValidSeconds and cannot be reused after it expires
type WebsocketTokenResponse struct {
	Token        string `json:"token"`
	ValidSeconds uint64 `json:"valid_sec"`
	UserID       uint64 `json:"user_id"`
}

// UserTransactionsRequest holds the parameters for fetching user transactions
// Pair is optional and restricts results to a single market
type UserTransactionsRequest struct {
	Pair    currency.Pair
	Offset  uint64
	Limit   uint64
	Sort    string
	Since   time.Time
	Until   time.Time
	SinceID uint64
}

// UserTransactionResponse holds a user transaction
// Amounts holds the signed amount of each currency involved in the transaction and ExchangeRates holds the
// execution rate for each market, both keyed by the names Bitstamp returns
type UserTransactionResponse struct {
	ID               uint64
	DateTime         time.Time
	Type             TransactionType
	Fee              float64
	OrderID          uint64
	SelfTrade        bool
	SelfTradeOrderID uint64
	Amounts          map[currency.Code]float64
	ExchangeRates    map[currency.Pair]float64
}

// CryptoTransactionsRequest holds the parameters for fetching cryptocurrency transactions
type CryptoTransactionsRequest struct {
	Limit       uint64
	Offset      uint64
	IncludeIOUs bool
	Since       time.Time
	Until       time.Time
}

// CryptoTransactionsResponse holds cryptocurrency deposits and withdrawals
type CryptoTransactionsResponse struct {
	RippleIOUTransactions []CryptoTransaction `json:"ripple_iou_transactions"`
	Deposits              []CryptoDeposit     `json:"deposits"`
	Withdrawals           []CryptoTransaction `json:"withdrawals"`
}

// CryptoTransaction holds a cryptocurrency withdrawal or Ripple IOU transaction
type CryptoTransaction struct {
	Currency           currency.Code `json:"currency"`
	Network            string        `json:"network"`
	DestinationAddress string        `json:"destinationAddress"`
	TxID               string        `json:"txid"`
	Amount             types.Number  `json:"amount"`
	DateTime           types.Time    `json:"datetime"`
}

// CryptoDeposit holds a cryptocurrency deposit
type CryptoDeposit struct {
	ID                 uint64        `json:"id"`
	Network            string        `json:"network"`
	Currency           currency.Code `json:"currency"`
	OriginatorAddress  string        `json:"originator_address"`
	TxID               string        `json:"txid"`
	Amount             types.Number  `json:"amount"`
	DateTime           types.Time    `json:"datetime"`
	Status             string        `json:"status"`
	PendingReason      string        `json:"pending_reason"`
	DestinationAddress string        `json:"destinationAddress"`
}

// CryptoDepositsRequest holds the parameters for fetching cryptocurrency deposits with their review status
type CryptoDepositsRequest struct {
	Offset uint64
	Limit  uint64
	Since  time.Time
	Until  time.Time
	Status string
}

// CryptoDepositResponse holds a cryptocurrency deposit and its review status
type CryptoDepositResponse struct {
	ID                 uint64        `json:"id"`
	Network            string        `json:"network"`
	Currency           currency.Code `json:"currency"`
	DestinationAddress string        `json:"destination_address"`
	OriginatorAddress  string        `json:"originator_address"`
	TxID               string        `json:"txid"`
	Amount             types.Number  `json:"amount"`
	DateTime           types.Time    `json:"datetime"`
	Status             string        `json:"status"`
	PendingReason      string        `json:"pending_reason"`
}

// DepositOriginatorRequest holds Travel Rule originator details for a pending cryptocurrency deposit
// OriginatorThirdParty is sent when set, and must be false when the deposit came from an address in the account holder's name
type DepositOriginatorRequest struct {
	OriginatorThirdParty *bool         `json:"originator_thirdparty,omitempty"`
	OriginatorID         string        `json:"originator_id,omitempty"`
	OriginatorInfo       *CustomerInfo `json:"originator_info,omitempty"`
	BeneficiaryInfo      *CustomerInfo `json:"beneficiary_info,omitempty"`
	VASPUUID             string        `json:"vasp_uuid,omitempty"`
}

// ContactRequest holds the details of a Travel Rule contact to create
type ContactRequest struct {
	RetailInfo    *RetailInfo    `json:"retail_info,omitempty"`
	CorporateInfo *CorporateInfo `json:"corporate_info,omitempty"`
	Description   string         `json:"description"`
}

// ContactResponse holds a Travel Rule contact
type ContactResponse struct {
	ID            string         `json:"id"`
	Description   string         `json:"description"`
	RetailInfo    *RetailInfo    `json:"retail_info"`
	CorporateInfo *CorporateInfo `json:"corporate_info"`
}

// CounterpartyAddressRequest holds Travel Rule counterparty details for a cryptocurrency address
type CounterpartyAddressRequest struct {
	Address           string `json:"address"`
	Network           string `json:"network"`
	MemoID            string `json:"memo_id,omitempty"`
	DestinationTag    string `json:"destination_tag,omitempty"`
	TransferID        uint64 `json:"transfer_id,omitempty"`
	ContactThirdParty bool   `json:"contact_thirdparty"`
	ContactUUID       string `json:"contact_uuid,omitempty"`
	VASPUUID          string `json:"vasp_uuid,omitempty"`
}

// CounterpartyAddressResponse holds the Travel Rule counterparty details registered for a cryptocurrency address
type CounterpartyAddressResponse struct {
	Address           string `json:"address"`
	Network           string `json:"network"`
	MemoID            string `json:"memo_id"`
	DestinationTag    string `json:"destination_tag"`
	TransferID        uint64 `json:"transfer_id"`
	ContactThirdParty bool   `json:"contact_thirdparty"`
	ContactUUID       string `json:"contact_uuid"`
	VASPUUID          string `json:"vasp_uuid"`
}

// SatoshiTestsRequest holds the filters for fetching Satoshi tests
type SatoshiTestsRequest struct {
	Network string
	Address string
	Status  string
}

// SatoshiTestRequest holds the parameters for creating a Satoshi test
type SatoshiTestRequest struct {
	Address  string
	Network  string
	Currency currency.Code
}

// SatoshiTestResponse holds a Satoshi test used to verify ownership of an external address
type SatoshiTestResponse struct {
	ID             string        `json:"id"`
	Network        string        `json:"network"`
	Currency       currency.Code `json:"currency"`
	Amount         types.Number  `json:"amount"`
	UserAddress    string        `json:"user_address"`
	DepositAddress string        `json:"deposit_address"`
	Status         string        `json:"status"`
	Expires        types.Time    `json:"expires"`
}

// XpubRegistrationsRequest holds the filters for fetching xpub registrations
type XpubRegistrationsRequest struct {
	Network string
	Status  string
	Offset  uint64
	Limit   uint64
}

// XpubRegistrationRequest holds the parameters for registering an extended public key
type XpubRegistrationRequest struct {
	Network           string `json:"network"`
	ExtendedPublicKey string `json:"extended_public_key"`
	Label             string `json:"label,omitempty"`
}

// XpubRegistrationResponse holds an extended public key registration
type XpubRegistrationResponse struct {
	ID          string                `json:"id"`
	Network     string                `json:"network"`
	Label       string                `json:"label"`
	Status      string                `json:"status"`
	Proof       XpubRegistrationProof `json:"proof"`
	CreatedAt   DateTime              `json:"created_at"`
	ActivatedAt DateTime              `json:"activated_at"`
	RevokedAt   DateTime              `json:"revoked_at"`
}

// XpubRegistrationProof holds the proof of control for an extended public key registration
type XpubRegistrationProof struct {
	ID     string `json:"id"`
	Method string `json:"method"`
}

// XpubRevocationResponse holds the status of a revoked extended public key registration
type XpubRevocationResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// AddressVerificationResponse holds the ownership verification status of a cryptocurrency address
type AddressVerificationResponse struct {
	Network    string   `json:"network"`
	Address    string   `json:"address"`
	Verified   bool     `json:"verified"`
	Method     string   `json:"method"`
	VerifiedAt DateTime `json:"verified_at"`
	RevokedAt  DateTime `json:"revoked_at"`
}

// EarnSubscriptionRequest holds the parameters for subscribing to or unsubscribing from an Earn product
type EarnSubscriptionRequest struct {
	Currency currency.Code
	EarnType string
	EarnTerm string
	Amount   float64
}

// EarnSubscriptionSettingRequest holds the parameters for opting in to or out of an Earn product
type EarnSubscriptionSettingRequest struct {
	Setting  string
	Currency currency.Code
	EarnType string
}

// EarnTransactionsRequest holds the parameters for fetching Earn transactions
type EarnTransactionsRequest struct {
	Limit         uint64
	Offset        uint64
	Currency      currency.Code
	QuoteCurrency currency.Code
}

// EarnTransactionResponse holds an Earn transaction
type EarnTransactionResponse struct {
	DateTime      types.DateTime `json:"datetime"`
	Type          string         `json:"type"`
	Amount        types.Number   `json:"amount"`
	Currency      currency.Code  `json:"currency"`
	Value         types.Number   `json:"value"`
	QuoteCurrency currency.Code  `json:"quote_currency"`
	Status        string         `json:"status"`
}

// EarnSubscriptionResponse holds an Earn subscription
type EarnSubscriptionResponse struct {
	Currency                  currency.Code `json:"currency"`
	Type                      string        `json:"type"`
	Term                      string        `json:"term"`
	EstimatedAnnualYield      types.Number  `json:"estimated_annual_yield"`
	DistributionPeriod        string        `json:"distribution_period"`
	ActivationPeriod          string        `json:"activation_period"`
	MinimumSubscriptionAmount types.Number  `json:"minimum_subscription_amount"`
	Amount                    types.Number  `json:"amount"`
	AvailableAmount           types.Number  `json:"available_amount"`
	AmountEarned              types.Number  `json:"amount_earned"`
}

// RevokeAPIKeysResponse holds the API keys revoked by a revoke all API keys request
type RevokeAPIKeysResponse struct {
	RevokedAPIKeys []string `json:"revoked_api_keys"`
}

// errorResponse holds the fields Bitstamp uses to report a failed request
// Each field is decoded when formatted since reasons may be strings, lists or objects mapping fields to messages and
// codes may be strings or numbers
type errorResponse struct {
	Status              json.RawMessage `json:"status"`
	Reason              json.RawMessage `json:"reason"`
	Code                json.RawMessage `json:"code"`
	Error               json.RawMessage `json:"error"`
	Message             json.RawMessage `json:"message"`
	Field               json.RawMessage `json:"field"`
	ResponseCode        json.RawMessage `json:"response_code"`
	ResponseExplanation json.RawMessage `json:"response_explanation"`
	Errors              json.RawMessage `json:"errors"`
}

// derivativesOrderParams holds the optional order parameters shared by limit, instant and market orders
type derivativesOrderParams struct {
	ClientOrderID   string
	Subtype         string
	MarginMode      string
	Leverage        float64
	StopPrice       float64
	Trigger         string
	ActivationPrice float64
	TrailingDelta   uint64
	ReduceOnly      bool
}

// maxOrderAmountBody is the JSON body of a maximum order amount request
type maxOrderAmountBody struct {
	Market               string            `json:"market"`
	MarginMode           string            `json:"margin_mode"`
	Leverage             float64           `json:"leverage"`
	OrderType            string            `json:"order_type"`
	Side                 string            `json:"side"`
	Price                float64           `json:"price,omitempty"`
	StopPrice            float64           `json:"stop_price,omitempty"`
	ActivationPrice      float64           `json:"activation_price,omitempty"`
	TrailingDelta        uint64            `json:"trailing_delta,omitempty"`
	AdditionalCollateral map[string]string `json:"additional_collateral,omitempty"`
}

// satoshiTestBody is the JSON body of a Satoshi test request
type satoshiTestBody struct {
	Address  string `json:"address"`
	Network  string `json:"network"`
	Currency string `json:"currency"`
}

// earnSubscriptionBody is the JSON body of an Earn subscribe or unsubscribe request
type earnSubscriptionBody struct {
	Currency string  `json:"currency"`
	EarnType string  `json:"earn_type"`
	EarnTerm string  `json:"earn_term"`
	Amount   float64 `json:"amount"`
}

// earnSubscriptionSettingBody is the JSON body of an Earn subscription setting request
type earnSubscriptionSettingBody struct {
	Setting  string `json:"setting"`
	Currency string `json:"currency"`
	EarnType string `json:"earn_type"`
}

// retailInfoJSON is the JSON representation of RetailInfo
type retailInfoJSON struct {
	FirstName            string `json:"first_name"`
	LastName             string `json:"last_name"`
	DateOfBirth          string `json:"date_of_birth,omitempty"`
	PlaceOfBirth         string `json:"place_of_birth,omitempty"`
	IDType               string `json:"id_type,omitempty"`
	IDNumber             string `json:"id_number,omitempty"`
	StreetAndHouseNumber string `json:"street_and_house_number,omitempty"`
	City                 string `json:"city,omitempty"`
	PostalCode           string `json:"zip,omitempty"`
	Country              string `json:"country,omitempty"`
}

// travelRuleParams holds the optional Travel Rule parameters shared by cryptocurrency withdrawals
type travelRuleParams struct {
	OriginatorInfo        *CustomerInfo
	BeneficiaryInfo       *CustomerInfo
	BeneficiaryThirdParty *bool
	BeneficiaryID         string
	VASPUUID              string
}

// OrderSide decodes an order side sent as 0 for buy or 1 for sell, either quoted or bare
type OrderSide order.Side

// DateTime decodes a date and time sent either in RFC 3339 form or in Bitstamp's "2006-01-02 15:04:05" form, with or
// without fractional seconds, since the API documents both forms as date-time values; null and an empty string decode as
// the zero time
type DateTime time.Time

// unconfirmedDeposits decodes unconfirmed deposits sent as either a list or a single object
type unconfirmedDeposits []UnconfirmedDepositResponse
