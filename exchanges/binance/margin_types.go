package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AdjustCrossMarginMaxLeverageResponse holds the result of adjusting the cross margin max leverage
type AdjustCrossMarginMaxLeverageResponse struct {
	Success bool `json:"success"`
}

// IsolatedMarginAccountInfoResponse holds isolated margin accounts and their totals in BTC
type IsolatedMarginAccountInfoResponse struct {
	Assets              []IsolatedMarginAccountAsset `json:"assets"`
	TotalAssetOfBTC     types.Number                 `json:"totalAssetOfBtc"`
	TotalLiabilityOfBTC types.Number                 `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC  types.Number                 `json:"totalNetAssetOfBtc"`
}

// IsolatedMarginAccountAsset holds the isolated margin account of one symbol
type IsolatedMarginAccountAsset struct {
	BaseAsset         IsolatedMarginAssetBalance `json:"baseAsset"`
	QuoteAsset        IsolatedMarginAssetBalance `json:"quoteAsset"`
	Symbol            string                     `json:"symbol"`
	IsolatedCreated   bool                       `json:"isolatedCreated"`
	Enabled           bool                       `json:"enabled"`
	MarginLevel       types.Number               `json:"marginLevel"`
	MarginLevelStatus string                     `json:"marginLevelStatus"` // EXCESSIVE, NORMAL, MARGIN_CALL, PRE_LIQUIDATION or FORCE_LIQUIDATION
	MarginRatio       types.Number               `json:"marginRatio"`
	IndexPrice        types.Number               `json:"indexPrice"`
	LiquidatePrice    types.Number               `json:"liquidatePrice"`
	LiquidateRate     types.Number               `json:"liquidateRate"`
	TradeEnabled      bool                       `json:"tradeEnabled"`
}

// IsolatedMarginAssetBalance holds the balance of the base or quote asset of an isolated margin account
type IsolatedMarginAssetBalance struct {
	Asset         currency.Code `json:"asset"`
	BorrowEnabled bool          `json:"borrowEnabled"`
	Borrowed      types.Number  `json:"borrowed"`
	Free          types.Number  `json:"free"`
	Interest      types.Number  `json:"interest"`
	Locked        types.Number  `json:"locked"`
	NetAsset      types.Number  `json:"netAsset"`
	NetAssetOfBTC types.Number  `json:"netAssetOfBtc"`
	RepayEnabled  bool          `json:"repayEnabled"`
	TotalAsset    types.Number  `json:"totalAsset"`
}

// IsolatedMarginAccountToggleResponse holds the result of enabling or disabling an isolated margin account
type IsolatedMarginAccountToggleResponse struct {
	Success bool   `json:"success"`
	Symbol  string `json:"symbol"`
}

// MarginAccountSummaryResponse holds the margin levels at which Binance treats the margin account as normal, calls for
// margin and force liquidates it
type MarginAccountSummaryResponse struct {
	NormalBar           types.Number `json:"normalBar"`
	MarginCallBar       types.Number `json:"marginCallBar"`
	ForceLiquidationBar types.Number `json:"forceLiquidationBar"`
}

// MarginCapitalFlowRequest holds the filters for GetCrossOrIsolatedMarginCapitalFlow
type MarginCapitalFlowRequest struct {
	Asset     currency.Code
	Symbol    currency.Pair // Selects isolated margin flows
	Type      string        // For example TRANSFER, BORROW or REPAY
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64 // Returns flows with a greater ID within the window
	Limit     uint64 // Default 500, maximum 1000
}

// MarginCapitalFlow holds one cross or isolated margin capital flow
type MarginCapitalFlow struct {
	ID            uint64        `json:"id"`
	TransactionID uint64        `json:"tranId"`
	Timestamp     types.Time    `json:"timestamp"`
	Asset         currency.Code `json:"asset"`
	Symbol        string        `json:"symbol"`
	Type          string        `json:"type"`
	Amount        types.Number  `json:"amount"`
	Note          string        `json:"note"` // INSTITUTIONAL_LOAN_TRANSFER, INSTITUTIONAL_LOAN_BORROW or INSTITUTIONAL_LOAN_REPAY for institutional loan flows
}

// CrossMarginAccountDetailResponse holds the cross margin account details
type CrossMarginAccountDetailResponse struct {
	Created                    bool                   `json:"created"`
	BorrowEnabled              bool                   `json:"borrowEnabled"`
	MarginLevel                types.Number           `json:"marginLevel"`
	CollateralMarginLevel      types.Number           `json:"collateralMarginLevel"`
	TotalAssetOfBTC            types.Number           `json:"totalAssetOfBtc"`
	TotalLiabilityOfBTC        types.Number           `json:"totalLiabilityOfBtc"`
	TotalNetAssetOfBTC         types.Number           `json:"totalNetAssetOfBtc"`
	TotalCollateralValueInUSDT types.Number           `json:"TotalCollateralValueInUSDT"`
	TradeEnabled               bool                   `json:"tradeEnabled"`
	TransferInEnabled          bool                   `json:"transferInEnabled"`
	TransferOutEnabled         bool                   `json:"transferOutEnabled"`
	AccountType                string                 `json:"accountType"` // MARGIN_1 for Cross Margin Classic, MARGIN_2 for Cross Margin Pro
	UserAssets                 []CrossMarginUserAsset `json:"userAssets"`
}

// CrossMarginUserAsset holds the balance of one asset in the cross margin account
type CrossMarginUserAsset struct {
	Asset    currency.Code `json:"asset"`
	Borrowed types.Number  `json:"borrowed"`
	Free     types.Number  `json:"free"`
	Interest types.Number  `json:"interest"`
	Locked   types.Number  `json:"locked"`
	NetAsset types.Number  `json:"netAsset"`
}

// CrossMarginFeeData holds the cross margin fee data of a coin at a VIP level
type CrossMarginFeeData struct {
	VIPLevel        uint64        `json:"vipLevel"`
	Coin            currency.Code `json:"coin"`
	TransferIn      bool          `json:"transferIn"`
	Borrowable      bool          `json:"borrowable"`
	DailyInterest   types.Number  `json:"dailyInterest"`
	YearlyInterest  types.Number  `json:"yearlyInterest"`
	BorrowLimit     types.Number  `json:"borrowLimit"`
	MarginablePairs []string      `json:"marginablePairs"`
}

// IsolatedMarginAccountLimitResponse holds the number of enabled isolated margin accounts and the maximum allowed
type IsolatedMarginAccountLimitResponse struct {
	EnabledAccount uint64 `json:"enabledAccount"`
	MaxAccount     uint64 `json:"maxAccount"`
}

// IsolatedMarginFeeData holds the isolated margin fee data of a symbol at a VIP level
type IsolatedMarginFeeData struct {
	VIPLevel uint64                  `json:"vipLevel"`
	Symbol   string                  `json:"symbol"`
	Leverage types.Number            `json:"leverage"`
	Data     []IsolatedMarginFeeCoin `json:"data"`
}

// IsolatedMarginFeeCoin holds the fee data of the base or quote coin of an isolated margin symbol
type IsolatedMarginFeeCoin struct {
	Coin          currency.Code `json:"coin"`
	DailyInterest types.Number  `json:"dailyInterest"`
	BorrowLimit   types.Number  `json:"borrowLimit"`
}

// HourlyInterestRate holds the next hourly interest rate estimate of an asset
type HourlyInterestRate struct {
	Asset                  currency.Code `json:"asset"`
	NextHourlyInterestRate types.Number  `json:"nextHourlyInterestRate"`
}

// UserMarginInterestHistoryRequest holds the filters for GetUserMarginInterestHistory
type UserMarginInterestHistoryRequest struct {
	Asset          currency.Code
	IsolatedSymbol currency.Pair // Empty for cross margin
	StartTime      time.Time
	EndTime        time.Time
	Current        uint64 // Page number, from 1
	Size           uint64 // Page size, default 10, maximum 100
}

// UserMarginInterestHistoryResponse holds a page of the margin interest history
type UserMarginInterestHistoryResponse struct {
	Rows  []UserMarginInterestHistory `json:"rows"`
	Total uint64                      `json:"total"`
}

// UserMarginInterestHistory holds one margin interest charge
type UserMarginInterestHistory struct {
	TransactionID       uint64        `json:"txId"`
	InterestAccuredTime types.Time    `json:"interestAccuredTime"`
	Asset               currency.Code `json:"asset"`
	RawAsset            currency.Code `json:"rawAsset"` // Not returned for isolated margin
	Principal           types.Number  `json:"principal"`
	Interest            types.Number  `json:"interest"`
	InterestRate        types.Number  `json:"interestRate"`
	Type                string        `json:"type"`           // PERIODIC, ON_BORROW, PERIODIC_CONVERTED, ON_BORROW_CONVERTED or PORTFOLIO
	IsolatedSymbol      string        `json:"isolatedSymbol"` // Not returned for cross margin
}

// MarginBorrowRepayRecordsRequest holds the filters for GetBorrowOrRepayRecordsInMarginAccount
type MarginBorrowRepayRecordsRequest struct {
	Type           string // BORROW or REPAY
	Asset          currency.Code
	IsolatedSymbol currency.Pair // Empty for cross margin
	TransactionID  uint64
	StartTime      time.Time
	EndTime        time.Time
	Current        uint64 // Page number, from 1
	Size           uint64 // Page size, default 10, maximum 100
}

// MarginBorrowRepayRecordsResponse holds a page of margin borrow or repay records
type MarginBorrowRepayRecordsResponse struct {
	Rows  []MarginBorrowRepayRecord `json:"rows"`
	Total uint64                    `json:"total"`
}

// MarginBorrowRepayRecord holds one margin borrow or repayment
type MarginBorrowRepayRecord struct {
	Type           string        `json:"type"`           // AUTO or MANUAL; cross margin repayments also BNB_AUTO_REPAY or POINT_AUTO_REPAY
	IsolatedSymbol string        `json:"isolatedSymbol"` // Not returned for cross margin
	Amount         types.Number  `json:"amount"`
	Asset          currency.Code `json:"asset"`
	Interest       types.Number  `json:"interest"`  // Interest repaid
	Principal      types.Number  `json:"principal"` // Principal repaid
	Status         string        `json:"status"`    // PENDING, CONFIRMED or FAILED
	Timestamp      types.Time    `json:"timestamp"`
	TransactionID  uint64        `json:"txId"`
}

// MarginAccountBorrowRepayRequest holds the parameters for MarginAccountBorrowRepay
type MarginAccountBorrowRepayRequest struct {
	Asset      currency.Code
	IsIsolated bool
	Symbol     currency.Pair // Required for isolated margin and not sent for cross margin
	Amount     float64
	Type       string // BORROW or REPAY
}

// MarginAccountBorrowRepayResponse holds the transaction ID of a margin borrow or repayment
type MarginAccountBorrowRepayResponse struct {
	TransactionID uint64 `json:"tranId"`
}

// MarginInterestRateHistoryRequest holds the filters for GetMarginInterestRateHistory
type MarginInterestRateHistoryRequest struct {
	Asset     currency.Code
	VIPLevel  *uint64 // nil returns the user's own level
	StartTime time.Time
	EndTime   time.Time
}

// MarginInterestRate holds the daily margin interest rate of an asset at a VIP level
type MarginInterestRate struct {
	Asset             currency.Code `json:"asset"`
	DailyInterestRate types.Number  `json:"dailyInterestRate"`
	Timestamp         types.Time    `json:"timestamp"`
	VIPLevel          uint64        `json:"vipLevel"`
}

// MarginMaxBorrowableResponse holds the maximum amount of an asset the margin account can borrow
type MarginMaxBorrowableResponse struct {
	Amount      types.Number `json:"amount"`      // Limited by the account level and the system's available inventory
	BorrowLimit types.Number `json:"borrowLimit"` // Limited by the account level only
}

// CrossMarginCollateralRatio holds the collateral discount rate tiers shared by a group of assets
type CrossMarginCollateralRatio struct {
	Collaterals []CrossMarginCollateral `json:"collaterals"`
	AssetNames  []currency.Code         `json:"assetNames"`
}

// CrossMarginCollateral holds the discount rate applied to collateral within a USD value range
type CrossMarginCollateral struct {
	MinUSDValue  types.Number `json:"minUsdValue"`
	MaxUSDValue  types.Number `json:"maxUsdValue"`
	DiscountRate types.Number `json:"discountRate"`
}

// CrossMarginPair holds a cross margin pair
type CrossMarginPair struct {
	Base          currency.Code `json:"base"`
	ID            uint64        `json:"id"`
	IsBuyAllowed  bool          `json:"isBuyAllowed"`
	IsMarginTrade bool          `json:"isMarginTrade"`
	IsSellAllowed bool          `json:"isSellAllowed"`
	Quote         currency.Code `json:"quote"`
	Symbol        string        `json:"symbol"`
	DelistTime    types.Time    `json:"delistTime"`
}

// IsolatedMarginSymbol holds an isolated margin symbol
type IsolatedMarginSymbol struct {
	Base          currency.Code `json:"base"`
	IsBuyAllowed  bool          `json:"isBuyAllowed"`
	IsMarginTrade bool          `json:"isMarginTrade"`
	IsSellAllowed bool          `json:"isSellAllowed"`
	Quote         currency.Code `json:"quote"`
	Symbol        string        `json:"symbol"`
}

// MarginAssetInfo holds a margin asset
type MarginAssetInfo struct {
	AssetFullName  string        `json:"assetFullName"`
	AssetName      currency.Code `json:"assetName"`
	IsBorrowable   bool          `json:"isBorrowable"`
	IsMortgageable bool          `json:"isMortgageable"`
	UserMinBorrow  types.Number  `json:"userMinBorrow"`
	UserMinRepay   types.Number  `json:"userMinRepay"`
	DelistTime     types.Time    `json:"delistTime"`
}

// MarginDelistSchedule holds the cross margin assets and isolated margin symbols delisted at a time
type MarginDelistSchedule struct {
	DelistTime            types.Time      `json:"delistTime"`
	CrossMarginAssets     []currency.Code `json:"crossMarginAssets"`
	IsolatedMarginSymbols []string        `json:"isolatedMarginSymbols"`
}

// IsolatedMarginTier holds one isolated margin tier of a symbol
type IsolatedMarginTier struct {
	Symbol                  string       `json:"symbol"`
	Tier                    uint64       `json:"tier"`
	EffectiveMultiple       types.Number `json:"effectiveMultiple"`
	InitialRiskRatio        types.Number `json:"initialRiskRatio"`
	LiquidationRiskRatio    types.Number `json:"liquidationRiskRatio"`
	BaseAssetMaxBorrowable  types.Number `json:"baseAssetMaxBorrowable"`
	QuoteAssetMaxBorrowable types.Number `json:"quoteAssetMaxBorrowable"`
}

// MarginAvailableInventoryResponse holds the inventory available to borrow per asset
type MarginAvailableInventoryResponse struct {
	Assets     map[currency.Code]types.Number `json:"assets"`
	UpdateTime types.Time                     `json:"updateTime"`
}

// MarginPriceIndexResponse holds the margin price index of a symbol
type MarginPriceIndexResponse struct {
	CalculationTime types.Time   `json:"calcTime"`
	Price           types.Number `json:"price"`
	Symbol          string       `json:"symbol"`
}

// ForceLiquidationRecordRequest holds the filters for GetForceLiquidationRecord
type ForceLiquidationRecordRequest struct {
	StartTime      time.Time
	EndTime        time.Time
	IsolatedSymbol currency.Pair // Empty for cross margin
	Current        uint64        // Page number, from 1
	Size           uint64        // Page size, default 10, maximum 100
}

// ForceLiquidationRecordResponse holds a page of force liquidation records
type ForceLiquidationRecordResponse struct {
	Rows  []ForceLiquidationRecord `json:"rows"`
	Total uint64                   `json:"total"`
}

// ForceLiquidationRecord holds one force liquidation order
type ForceLiquidationRecord struct {
	AveragePrice     types.Number `json:"avgPrice"`
	ExecutedQuantity types.Number `json:"executedQty"`
	OrderID          uint64       `json:"orderId"`
	Price            types.Number `json:"price"`
	Quantity         types.Number `json:"qty"`
	Side             string       `json:"side"`
	Symbol           string       `json:"symbol"`
	TimeInForce      string       `json:"timeInForce"`
	IsIsolated       bool         `json:"isIsolated"`
	UpdatedTime      types.Time   `json:"updatedTime"`
}

// SmallLiabilityExchangeCoin holds a liability MarginSmallLiabilityExchange can exchange
type SmallLiabilityExchangeCoin struct {
	Asset             currency.Code `json:"asset"`
	Interest          types.Number  `json:"interest"`
	Principal         types.Number  `json:"principal"`
	LiabilityAsset    currency.Code `json:"liabilityAsset"`
	LiabilityQuantity float64       `json:"liabilityQty"`
}

// SmallLiabilityExchangeHistoryRequest holds the filters for GetSmallLiabilityExchangeHistory
type SmallLiabilityExchangeHistoryRequest struct {
	Current   uint64 // Page number, from 1
	Size      uint64 // Page size, maximum 100
	StartTime time.Time
	EndTime   time.Time
}

// SmallLiabilityExchangeHistoryResponse holds a page of the small liability exchange history
type SmallLiabilityExchangeHistoryResponse struct {
	Total uint64                         `json:"total"`
	Rows  []SmallLiabilityExchangeRecord `json:"rows"`
}

// SmallLiabilityExchangeRecord holds one small liability exchange
type SmallLiabilityExchangeRecord struct {
	Asset        currency.Code `json:"asset"`
	Amount       types.Number  `json:"amount"`
	TargetAsset  currency.Code `json:"targetAsset"`
	TargetAmount types.Number  `json:"targetAmount"`
	BusinessType string        `json:"bizType"`
	Timestamp    types.Time    `json:"timestamp"`
}

// MarginTradeOrderResponse holds a margin account order as the order queries return it
type MarginTradeOrderResponse struct {
	ClientOrderID            string       `json:"clientOrderId"`
	CummulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"` // Negative for some historical orders whose value is unavailable
	ExecutedQuantity         types.Number `json:"executedQty"`
	IcebergQuantity          types.Number `json:"icebergQty"`
	IsWorking                bool         `json:"isWorking"`
	OrderID                  uint64       `json:"orderId"`
	OriginalQuantity         types.Number `json:"origQty"`
	Price                    types.Number `json:"price"`
	Side                     string       `json:"side"`
	Status                   string       `json:"status"`
	StopPrice                types.Number `json:"stopPrice"`
	Symbol                   string       `json:"symbol"`
	IsIsolated               bool         `json:"isIsolated"`
	Time                     types.Time   `json:"time"`
	TimeInForce              string       `json:"timeInForce"`
	Type                     string       `json:"type"`
	SelfTradePreventionMode  string       `json:"selfTradePreventionMode"`
	UpdateTime               types.Time   `json:"updateTime"`
}

// MarginCancelledOrder holds an order, or an order list such as OCO, cancelled by
// CancelAllOpenMarginAccountOrdersOnSymbol; the order list fields are set only for order lists
type MarginCancelledOrder struct {
	Symbol                   string                       `json:"symbol"`
	IsIsolated               bool                         `json:"isIsolated"`
	OriginalClientOrderID    string                       `json:"origClientOrderId"`
	OrderID                  uint64                       `json:"orderId"`
	OrderListID              int64                        `json:"orderListId"` // -1 when the order is not part of an order list
	ClientOrderID            string                       `json:"clientOrderId"`
	Price                    types.Number                 `json:"price"`
	OriginalQuantity         types.Number                 `json:"origQty"`
	ExecutedQuantity         types.Number                 `json:"executedQty"`
	CummulativeQuoteQuantity types.Number                 `json:"cummulativeQuoteQty"`
	Status                   string                       `json:"status"`
	TimeInForce              string                       `json:"timeInForce"`
	Type                     string                       `json:"type"`
	Side                     string                       `json:"side"`
	SelfTradePreventionMode  string                       `json:"selfTradePreventionMode"`
	ContingencyType          string                       `json:"contingencyType"`
	ListStatusType           string                       `json:"listStatusType"`
	ListOrderStatus          string                       `json:"listOrderStatus"`
	ListClientOrderID        string                       `json:"listClientOrderId"`
	TransactionTime          types.Time                   `json:"transactionTime"`
	Orders                   []MarginOrderListOrder       `json:"orders"`
	OrderReports             []MarginCancelledOrderReport `json:"orderReports"`
}

// MarginOrderListOrder holds an order of a margin order list
type MarginOrderListOrder struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// MarginCancelledOrderReport holds the report of an order cancelled with its order list by
// CancelAllOpenMarginAccountOrdersOnSymbol
type MarginCancelledOrderReport struct {
	Symbol                   string       `json:"symbol"`
	OriginalClientOrderID    string       `json:"origClientOrderId"`
	OrderID                  uint64       `json:"orderId"`
	OrderListID              uint64       `json:"orderListId"`
	ClientOrderID            string       `json:"clientOrderId"`
	Price                    types.Number `json:"price"`
	OriginalQuantity         types.Number `json:"origQty"`
	ExecutedQuantity         types.Number `json:"executedQty"`
	CummulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                   string       `json:"status"`
	TimeInForce              string       `json:"timeInForce"`
	Type                     string       `json:"type"`
	Side                     string       `json:"side"`
	StopPrice                types.Number `json:"stopPrice"`
	IcebergQuantity          types.Number `json:"icebergQty"`
}

// MarginOrderListResponse holds a margin order list as the order list queries return it. ContingencyType is OCO, OTO, OTOCO,
// OPO or OPOCO; OPO and OPOCO lists can be created only through the Binance website
type MarginOrderListResponse struct {
	OrderListID       uint64                 `json:"orderListId"`
	ContingencyType   string                 `json:"contingencyType"`
	ListStatusType    string                 `json:"listStatusType"`
	ListOrderStatus   string                 `json:"listOrderStatus"`
	ListClientOrderID string                 `json:"listClientOrderId"`
	TransactionTime   types.Time             `json:"transactionTime"`
	Symbol            string                 `json:"symbol"`
	IsIsolated        bool                   `json:"isIsolated"`
	Orders            []MarginOrderListOrder `json:"orders"`
}

// MarginOCOCancelRequest holds the parameters for CancelMarginAccountOCOOrder
type MarginOCOCancelRequest struct {
	Symbol            currency.Pair
	IsIsolated        bool
	OrderListID       uint64
	ListClientOrderID string
	NewClientOrderID  string
}

// MarginOCOCancelResponse holds an order list cancelled by CancelMarginAccountOCOOrder
type MarginOCOCancelResponse struct {
	OrderListID       uint64                  `json:"orderListId"`
	ContingencyType   string                  `json:"contingencyType"` // OCO, OTO, OTOCO, OPO or OPOCO
	ListStatusType    string                  `json:"listStatusType"`
	ListOrderStatus   string                  `json:"listOrderStatus"`
	ListClientOrderID string                  `json:"listClientOrderId"`
	TransactionTime   types.Time              `json:"transactionTime"`
	Symbol            string                  `json:"symbol"`
	IsIsolated        bool                    `json:"isIsolated"`
	Orders            []MarginOrderListOrder  `json:"orders"`
	OrderReports      []MarginOCOCancelReport `json:"orderReports"`
}

// MarginOCOCancelReport holds the report of an order cancelled with its order list
type MarginOCOCancelReport struct {
	Symbol                   string       `json:"symbol"`
	OriginalClientOrderID    string       `json:"origClientOrderId"`
	OrderID                  uint64       `json:"orderId"`
	OrderListID              uint64       `json:"orderListId"`
	ClientOrderID            string       `json:"clientOrderId"`
	Price                    types.Number `json:"price"`
	OriginalQuantity         types.Number `json:"origQty"`
	ExecutedQuantity         types.Number `json:"executedQty"`
	CummulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                   string       `json:"status"`
	TimeInForce              string       `json:"timeInForce"`
	Type                     string       `json:"type"`
	Side                     string       `json:"side"`
	StopPrice                types.Number `json:"stopPrice"`
	SelfTradePreventionMode  string       `json:"selfTradePreventionMode"`
}

// MarginAccountOrderRequest holds the parameters for PostMarginAccountOrder
type MarginAccountOrderRequest struct {
	Symbol                  currency.Pair
	IsIsolated              bool
	Side                    string // BUY or SELL
	OrderType               string // LIMIT, MARKET, STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, TAKE_PROFIT_LIMIT or LIMIT_MAKER
	Quantity                float64
	QuoteOrderQuantity      float64
	Price                   float64
	StopPrice               float64 // For STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT and TAKE_PROFIT_LIMIT orders
	NewClientOrderID        string
	IcebergQuantity         float64 // For LIMIT, STOP_LOSS_LIMIT and TAKE_PROFIT_LIMIT orders
	NewOrderResponseType    string  // ACK, RESULT or FULL
	SideEffectType          string  // NO_SIDE_EFFECT, MARGIN_BUY, AUTO_REPAY or AUTO_BORROW_REPAY
	TimeInForce             string
	SelfTradePreventionMode string
	TrailingDelta           uint64 // In basis points, for STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT and TAKE_PROFIT_LIMIT orders
	AutoRepayAtCancel       *bool  // Whether a MARGIN_BUY or AUTO_BORROW_REPAY order's debt is repaid when it is cancelled; nil leaves Binance's default of true
}

// MarginAccountNewOrderResponse holds a margin account order placed by PostMarginAccountOrder
type MarginAccountNewOrderResponse struct {
	Symbol                   string            `json:"symbol"`
	OrderID                  uint64            `json:"orderId"`
	ClientOrderID            string            `json:"clientOrderId"`
	IsIsolated               bool              `json:"isIsolated"`
	TransactTime             types.Time        `json:"transactTime"`
	Price                    types.Number      `json:"price"`
	OriginalQuantity         types.Number      `json:"origQty"`
	ExecutedQuantity         types.Number      `json:"executedQty"`
	CummulativeQuoteQuantity types.Number      `json:"cummulativeQuoteQty"`
	Status                   string            `json:"status"`
	TimeInForce              string            `json:"timeInForce"`
	Type                     string            `json:"type"`
	Side                     string            `json:"side"`
	SelfTradePreventionMode  string            `json:"selfTradePreventionMode"`
	MarginBuyBorrowAmount    types.Number      `json:"marginBuyBorrowAmount"` // Documented as a number here and as a string for OCO orders; only returned when the order borrows
	MarginBuyBorrowAsset     currency.Code     `json:"marginBuyBorrowAsset"`
	Fills                    []MarginOrderFill `json:"fills"`
}

// MarginOrderFill holds a fill of a margin account order
type MarginOrderFill struct {
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`      // null from 2026-10-14
	CommissionAsset currency.Code `json:"commissionAsset"` // null from 2026-10-14
	TradeID         uint64        `json:"tradeId"`
}

// MarginOrderCancelRequest holds the parameters for CancelMarginAccountOrder
type MarginOrderCancelRequest struct {
	Symbol                currency.Pair
	IsIsolated            bool
	OrderID               uint64
	OriginalClientOrderID string
	NewClientOrderID      string
}

// MarginOrderCancelResponse holds a margin account order cancelled by CancelMarginAccountOrder
type MarginOrderCancelResponse struct {
	Symbol                   string       `json:"symbol"`
	OrderID                  types.Number `json:"orderId"` // Documented as a string here and as a number by the other order endpoints
	OriginalClientOrderID    string       `json:"origClientOrderId"`
	ClientOrderID            string       `json:"clientOrderId"`
	Price                    types.Number `json:"price"`
	OriginalQuantity         types.Number `json:"origQty"`
	ExecutedQuantity         types.Number `json:"executedQty"`
	CummulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                   string       `json:"status"`
	TimeInForce              string       `json:"timeInForce"`
	Type                     string       `json:"type"`
	Side                     string       `json:"side"`
	IsIsolated               bool         `json:"isIsolated"`
}

// MarginOCOOrderRequest holds the parameters for NewMarginAccountOCOOrder
type MarginOCOOrderRequest struct {
	Symbol                  currency.Pair
	IsIsolated              bool
	ListClientOrderID       string
	Side                    string // BUY or SELL
	Quantity                float64
	LimitClientOrderID      string
	Price                   float64
	LimitIcebergQuantity    float64
	StopClientOrderID       string
	StopPrice               float64
	StopLimitPrice          float64 // Requires StopLimitTimeInForce
	StopIcebergQuantity     float64
	StopLimitTimeInForce    string // GTC, FOK or IOC
	NewOrderResponseType    string // ACK, RESULT or FULL
	SideEffectType          string // NO_SIDE_EFFECT, MARGIN_BUY, AUTO_REPAY or AUTO_BORROW_REPAY
	SelfTradePreventionMode string
	AutoRepayAtCancel       *bool // Whether a MARGIN_BUY or AUTO_BORROW_REPAY list's debt is repaid when it is cancelled; nil leaves Binance's default of true
}

// MarginOCOOrderResponse holds a margin OCO order list placed by NewMarginAccountOCOOrder
type MarginOCOOrderResponse struct {
	OrderListID           uint64                 `json:"orderListId"`
	ContingencyType       string                 `json:"contingencyType"`
	ListStatusType        string                 `json:"listStatusType"`
	ListOrderStatus       string                 `json:"listOrderStatus"`
	ListClientOrderID     string                 `json:"listClientOrderId"`
	TransactionTime       types.Time             `json:"transactionTime"`
	Symbol                string                 `json:"symbol"`
	MarginBuyBorrowAmount types.Number           `json:"marginBuyBorrowAmount"` // Only returned when the order list borrows
	MarginBuyBorrowAsset  currency.Code          `json:"marginBuyBorrowAsset"`  // Only returned when the order list borrows
	IsIsolated            bool                   `json:"isIsolated"`
	Orders                []MarginOrderListOrder `json:"orders"`
	OrderReports          []MarginOCOOrderReport `json:"orderReports"`
}

// MarginOCOOrderReport holds the report of an order placed with its OCO order list
type MarginOCOOrderReport struct {
	Symbol                   string       `json:"symbol"`
	OrderID                  uint64       `json:"orderId"`
	OrderListID              uint64       `json:"orderListId"`
	ClientOrderID            string       `json:"clientOrderId"`
	TransactTime             types.Time   `json:"transactTime"`
	Price                    types.Number `json:"price"`
	OriginalQuantity         types.Number `json:"origQty"`
	ExecutedQuantity         types.Number `json:"executedQty"`
	CummulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                   string       `json:"status"`
	TimeInForce              string       `json:"timeInForce"`
	Type                     string       `json:"type"`
	Side                     string       `json:"side"`
	StopPrice                types.Number `json:"stopPrice"`
	SelfTradePreventionMode  string       `json:"selfTradePreventionMode"`
}

// MarginManualLiquidationResponse holds the liability repaid by a margin manual liquidation
type MarginManualLiquidationResponse struct {
	Asset             currency.Code `json:"asset"`
	Interest          types.Number  `json:"interest"`
	Principal         types.Number  `json:"principal"`
	LiabilityAsset    currency.Code `json:"liabilityAsset"`
	LiabilityQuantity float64       `json:"liabilityQty"`
}

// MarginOrderCountUsage holds the margin account's order count against one rate limit interval
type MarginOrderCountUsage struct {
	RateLimitType  string `json:"rateLimitType"`
	Interval       string `json:"interval"`
	IntervalNumber uint64 `json:"intervalNum"`
	Limit          uint64 `json:"limit"`
	Count          uint64 `json:"count"`
}

// MarginAccountAllOCORequest holds the filters for GetMarginAccountAllOCO
type MarginAccountAllOCORequest struct {
	IsIsolated bool
	Symbol     currency.Pair
	FromID     uint64
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint64 // Default 500, maximum 1000
}

// MarginAccountAllOrdersRequest holds the filters for GetMarginAccountAllOrders
type MarginAccountAllOrdersRequest struct {
	Symbol     currency.Pair
	IsIsolated bool
	OrderID    uint64 // Returns orders from this order ID onwards
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint64 // Default 500, maximum 500
}

// MarginAccountTradeListRequest holds the filters for GetMarginAccountTradeList
type MarginAccountTradeListRequest struct {
	Symbol     currency.Pair
	IsIsolated bool
	OrderID    uint64
	StartTime  time.Time
	EndTime    time.Time
	FromID     uint64 // Returns trades from this trade ID onwards
	Limit      uint64 // Default 500, maximum 1000
}

// MarginAccountTrade holds a margin account trade
type MarginAccountTrade struct {
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	ID              uint64        `json:"id"`
	IsBestMatch     bool          `json:"isBestMatch"`
	IsBuyer         bool          `json:"isBuyer"`
	IsMaker         bool          `json:"isMaker"`
	OrderID         uint64        `json:"orderId"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Symbol          string        `json:"symbol"`
	IsIsolated      bool          `json:"isIsolated"`
	Time            types.Time    `json:"time"`
}

// CrossMarginTransferHistoryRequest holds the filters for GetCrossMarginTransferHistory
type CrossMarginTransferHistoryRequest struct {
	Asset          currency.Code
	Type           string // ROLL_IN or ROLL_OUT
	StartTime      time.Time
	EndTime        time.Time
	Current        uint64 // Page number, from 1
	Size           uint64 // Page size, default 10, maximum 100
	IsolatedSymbol currency.Pair
}

// CrossMarginTransferHistoryResponse holds a page of the cross margin transfer history
type CrossMarginTransferHistoryResponse struct {
	Rows  []CrossMarginTransfer `json:"rows"`
	Total uint64                `json:"total"`
}

// CrossMarginTransfer holds one transfer into or out of a margin account
type CrossMarginTransfer struct {
	Amount        types.Number  `json:"amount"`
	Asset         currency.Code `json:"asset"`
	Status        string        `json:"status"`
	Timestamp     types.Time    `json:"timestamp"`
	TransactionID uint64        `json:"txId"`
	Type          string        `json:"type"`      // ROLL_IN or ROLL_OUT
	TransferFrom  string        `json:"transFrom"` // An account type such as SPOT, ISOLATED_MARGIN or CROSS_MARGIN
	TransferTo    string        `json:"transTo"`   // An account type such as SPOT, ISOLATED_MARGIN or CROSS_MARGIN
	FromSymbol    string        `json:"fromSymbol"`
	ToSymbol      string        `json:"toSymbol"`
}

// MaxTransferOutAmountResponse holds the maximum amount of an asset that can be transferred out of a margin account
type MaxTransferOutAmountResponse struct {
	Amount types.Number `json:"amount"`
}

// MarginListenTokenRequest holds the parameters of the Create Margin Account listenToken request; a cross margin token
// is created unless IsIsolated is set
type MarginListenTokenRequest struct {
	// Symbol is required for an isolated margin account
	Symbol     currency.Pair
	IsIsolated bool
	// Validity is how long the token lasts: 24 hours when zero, which is also the maximum
	Validity time.Duration
}

// MarginListenTokenResponse is the Create Margin Account listenToken response
type MarginListenTokenResponse struct {
	Token          string     `json:"token"`
	ExpirationTime types.Time `json:"expirationTime"`
}
