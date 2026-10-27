package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AccountBalanceResponse holds a portfolio margin asset balance across the margin, UM and CM accounts
type AccountBalanceResponse struct {
	Asset              currency.Code `json:"asset"`
	TotalWalletBalance types.Number  `json:"totalWalletBalance"`
	// CrossMarginAsset is crossMarginFree plus crossMarginLocked; it is only returned when no asset is requested
	CrossMarginAsset          types.Number `json:"crossMarginAsset"`
	CrossMarginBorrowed       types.Number `json:"crossMarginBorrowed"`
	CrossMarginFree           types.Number `json:"crossMarginFree"`
	CrossMarginInterest       types.Number `json:"crossMarginInterest"`
	CrossMarginLocked         types.Number `json:"crossMarginLocked"`
	USDTMarginedWalletBalance types.Number `json:"umWalletBalance"`
	USDTMarginedUnrealisedPNL types.Number `json:"umUnrealizedPNL"`
	CoinMarginedWalletBalance types.Number `json:"cmWalletBalance"`
	CoinMarginedUnrealisedPNL types.Number `json:"cmUnrealizedPNL"`
	UpdateTime                types.Time   `json:"updateTime"`
	NegativeBalance           types.Number `json:"negativeBalance"`
}

// PortfolioMarginAccountInformationResponse holds portfolio margin account information, with amounts in USD
type PortfolioMarginAccountInformationResponse struct {
	UniMMR               types.Number `json:"uniMMR"`
	AccountEquity        types.Number `json:"accountEquity"`
	ActualEquity         types.Number `json:"actualEquity"` // account equity without the collateral rate applied
	AccountInitialMargin types.Number `json:"accountInitialMargin"`
	// AccountMaintenanceMargin is the portfolio margin account maintenance margin
	AccountMaintenanceMargin types.Number `json:"accountMaintMargin"`
	AccountStatus            string       `json:"accountStatus"`
	// VirtualMaxWithdrawAmount is the maximum amount that can be transferred out
	VirtualMaxWithdrawAmount types.Number `json:"virtualMaxWithdrawAmount"`
	TotalAvailableBalance    types.Number `json:"totalAvailableBalance"`
	TotalMarginOpenLoss      types.Number `json:"totalMarginOpenLoss"`
	UpdateTime               types.Time   `json:"updateTime"`
}

// PortfolioMarginTransactionResponse holds the transaction ID returned by portfolio margin transfer, borrow and repay
// endpoints
type PortfolioMarginTransactionResponse struct {
	TransactionID uint64 `json:"tranId"`
}

// PortfolioMarginAutoRepayFuturesStatusResponse holds whether futures negative balances are repaid automatically
type PortfolioMarginAutoRepayFuturesStatusResponse struct {
	AutoRepay bool `json:"autoRepay"`
}

// PortfolioMarginMessageResponse holds the message returned by portfolio margin endpoints that only report success
type PortfolioMarginMessageResponse struct {
	Message string `json:"msg"`
}

// PortfolioMarginCoinMarginedInitialLeverageResponse holds a COIN-M symbol's initial leverage after a change
type PortfolioMarginCoinMarginedInitialLeverageResponse struct {
	Leverage    uint64       `json:"leverage"`
	MaxQuantity types.Number `json:"maxQty"` // maximum quantity of the base asset
	Symbol      string       `json:"symbol"`
}

// PortfolioMarginPositionModeResponse holds whether hedge mode (dual side position) is enabled
type PortfolioMarginPositionModeResponse struct {
	DualSidePosition bool `json:"dualSidePosition"`
}

// SuccessResponse holds the code and message returned by portfolio margin endpoints that only report success
type SuccessResponse struct {
	// Code is quoted by the cancel all CM open conditional orders endpoint and bare everywhere else
	Code    types.Number `json:"code"`
	Message string       `json:"msg"`
}

// PortfolioMarginUSDTMarginedInitialLeverageResponse holds a USDⓈ-M symbol's initial leverage after a change
type PortfolioMarginUSDTMarginedInitialLeverageResponse struct {
	Leverage         uint64       `json:"leverage"`
	MaxNotionalValue types.Number `json:"maxNotionalValue"`
	Symbol           string       `json:"symbol"`
}

// PortfolioMarginCoinMarginedNotionalAndLeverageResponse holds a COIN-M symbol's notional and leverage brackets
type PortfolioMarginCoinMarginedNotionalAndLeverageResponse struct {
	Symbol   string                                       `json:"symbol"`
	Brackets []PortfolioMarginCoinMarginedNotionalBracket `json:"brackets"`
}

// PortfolioMarginCoinMarginedNotionalBracket holds a COIN-M leverage bracket, with caps and floors in base asset
// quantity
type PortfolioMarginCoinMarginedNotionalBracket struct {
	Bracket                uint64  `json:"bracket"`
	InitialLeverage        uint64  `json:"initialLeverage"`
	QuantityCap            float64 `json:"qtyCap"`
	QuantityFloor          float64 `json:"qtyFloor"`
	MaintenanceMarginRatio float64 `json:"maintMarginRatio"`
	Cumulative             float64 `json:"cum"` // auxiliary number for quick maintenance margin calculation
}

// PortfolioMarginAccountDetailAsset holds a UM or CM account asset
type PortfolioMarginAccountDetailAsset struct {
	Asset                  currency.Code `json:"asset"`
	CrossWalletBalance     types.Number  `json:"crossWalletBalance"`
	CrossUnrealisedPNL     types.Number  `json:"crossUnPnl"`
	MaintenanceMargin      types.Number  `json:"maintMargin"`
	InitialMargin          types.Number  `json:"initialMargin"`
	PositionInitialMargin  types.Number  `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number  `json:"openOrderInitialMargin"`
	UpdateTime             types.Time    `json:"updateTime"`
}

// PortfolioMarginCoinMarginedAccountDetailPosition holds a CM account position
type PortfolioMarginCoinMarginedAccountDetailPosition struct {
	Symbol                 string       `json:"symbol"`
	PositionAmount         types.Number `json:"positionAmt"`
	InitialMargin          types.Number `json:"initialMargin"`
	MaintenanceMargin      types.Number `json:"maintMargin"`
	UnrealisedProfit       types.Number `json:"unrealizedProfit"`
	PositionInitialMargin  types.Number `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number `json:"openOrderInitialMargin"`
	Leverage               types.Number `json:"leverage"`
	PositionSide           string       `json:"positionSide"`
	EntryPrice             types.Number `json:"entryPrice"`
	MaxQuantity            types.Number `json:"maxQty"` // maximum quantity of the base asset
	UpdateTime             types.Time   `json:"updateTime"`
}

// PortfolioMarginCoinMarginedAccountDetailResponse holds CM account assets and positions
type PortfolioMarginCoinMarginedAccountDetailResponse struct {
	Assets    []PortfolioMarginAccountDetailAsset                `json:"assets"`
	Positions []PortfolioMarginCoinMarginedAccountDetailPosition `json:"positions"`
}

// PortfolioMarginIncomeHistoryRequest holds the parameters for a UM or CM income history query
type PortfolioMarginIncomeHistoryRequest struct {
	Symbol     currency.Pair
	IncomeType string
	StartTime  time.Time
	EndTime    time.Time
	Page       uint64
	Limit      uint64
}

// IncomeItemResponse holds a UM or CM income history record
type IncomeItemResponse struct {
	Symbol     string        `json:"symbol"`
	IncomeType string        `json:"incomeType"`
	Income     types.Number  `json:"income"`
	Asset      currency.Code `json:"asset"`
	Info       string        `json:"info"`
	Time       types.Time    `json:"time"`
	// TransactionID is unique per user within an income type
	TransactionID string `json:"tranId"`
	TradeID       string `json:"tradeId"`
}

// PortfolioMarginMarginInterestHistoryRequest holds the parameters for a margin borrow or loan interest history query
type PortfolioMarginMarginInterestHistoryRequest struct {
	Asset     currency.Code
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
	// Archived queries data older than six months
	Archived bool
}

// PortfolioMarginMarginInterestHistoryResponse holds a page of margin borrow or loan interest records
type PortfolioMarginMarginInterestHistoryResponse struct {
	Rows  []PortfolioMarginMarginInterest `json:"rows"`
	Total uint64                          `json:"total"`
}

// PortfolioMarginMarginInterest holds a margin borrow or loan interest record
type PortfolioMarginMarginInterest struct {
	TransactionID       uint64        `json:"txId"`
	InterestAccuredTime types.Time    `json:"interestAccuredTime"`
	Asset               currency.Code `json:"asset"`
	RawAsset            currency.Code `json:"rawAsset"`
	Principal           types.Number  `json:"principal"`
	Interest            types.Number  `json:"interest"`
	InterestRate        types.Number  `json:"interestRate"` // daily rate
	// Type is PERIODIC, ON_BORROW, PERIODIC_CONVERTED, ON_BORROW_CONVERTED or PORTFOLIO
	Type string `json:"type"`
}

// PortfolioMarginUSDTMarginedAccountDetailPosition holds a UM account position
type PortfolioMarginUSDTMarginedAccountDetailPosition struct {
	Symbol                 string       `json:"symbol"`
	InitialMargin          types.Number `json:"initialMargin"`
	MaintenanceMargin      types.Number `json:"maintMargin"`
	UnrealisedProfit       types.Number `json:"unrealizedProfit"`
	PositionInitialMargin  types.Number `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number `json:"openOrderInitialMargin"`
	Leverage               types.Number `json:"leverage"`
	EntryPrice             types.Number `json:"entryPrice"`
	// MaxNotional is the maximum notional available at the current leverage
	MaxNotional    types.Number `json:"maxNotional"`
	BidNotional    types.Number `json:"bidNotional"`
	AskNotional    types.Number `json:"askNotional"`
	PositionSide   string       `json:"positionSide"`
	PositionAmount types.Number `json:"positionAmt"`
	UpdateTime     types.Time   `json:"updateTime"`
}

// PortfolioMarginUSDTMarginedAccountDetailResponse holds UM account assets and positions
type PortfolioMarginUSDTMarginedAccountDetailResponse struct {
	Assets    []PortfolioMarginAccountDetailAsset                `json:"assets"`
	Positions []PortfolioMarginUSDTMarginedAccountDetailPosition `json:"positions"`
}

// PortfolioMarginCommissionRateResponse holds a UM or CM symbol's commission rates
type PortfolioMarginCommissionRateResponse struct {
	Symbol              string       `json:"symbol"`
	MakerCommissionRate types.Number `json:"makerCommissionRate"`
	TakerCommissionRate types.Number `json:"takerCommissionRate"`
}

// PortfolioMarginMarginMaxBorrowResponse holds the maximum amount of an asset the margin account can borrow
type PortfolioMarginMarginMaxBorrowResponse struct {
	// Amount is the current maximum borrowable amount given system availability
	Amount types.Number `json:"amount"`
	// BorrowLimit is the maximum borrowable amount allowed by the account level
	BorrowLimit types.Number `json:"borrowLimit"`
}

// PortfolioMarginTradingQuantitativeRulesIndicatorsResponse holds the UM trading quantitative rules indicators
type PortfolioMarginTradingQuantitativeRulesIndicatorsResponse struct {
	// Indicators is keyed by symbol, with account wide indicators under ACCOUNT
	Indicators map[string][]PortfolioMarginTradingIndicator `json:"indicators"`
	UpdateTime types.Time                                   `json:"updateTime"`
}

// PortfolioMarginTradingIndicator holds a trading quantitative rules indicator
type PortfolioMarginTradingIndicator struct {
	IsLocked           bool       `json:"isLocked"`
	PlannedRecoverTime types.Time `json:"plannedRecoverTime"`
	Indicator          string     `json:"indicator"`
	Value              float64    `json:"value"`
	TriggerValue       float64    `json:"triggerValue"`
}

// PortfolioMarginCoinMarginedPositionInformationResponse holds a CM position
type PortfolioMarginCoinMarginedPositionInformationResponse struct {
	Symbol           string       `json:"symbol"`
	PositionAmount   types.Number `json:"positionAmt"`
	EntryPrice       types.Number `json:"entryPrice"`
	MarkPrice        types.Number `json:"markPrice"`
	UnrealisedProfit types.Number `json:"unRealizedProfit"`
	LiquidationPrice types.Number `json:"liquidationPrice"`
	Leverage         types.Number `json:"leverage"`
	PositionSide     string       `json:"positionSide"`
	UpdateTime       types.Time   `json:"updateTime"`
	MaxQuantity      types.Number `json:"maxQty"` // maximum quantity of the base asset
	NotionalValue    types.Number `json:"notionalValue"`
}

// PortfolioMarginMarginLoanRepayRecordRequest holds the parameters for a margin loan or repay record query
type PortfolioMarginMarginLoanRepayRecordRequest struct {
	Asset currency.Code
	// TransactionID is the tranId returned by the borrow or repay endpoint and takes precedence over StartTime
	TransactionID uint64
	StartTime     time.Time
	EndTime       time.Time
	Current       uint64
	Size          uint64
	// Archived queries data older than six months
	Archived bool
}

// PortfolioMarginMarginLoanRecordResponse holds a page of margin loan records
type PortfolioMarginMarginLoanRecordResponse struct {
	Rows  []PortfolioMarginMarginLoanRecord `json:"rows"`
	Total uint64                            `json:"total"`
}

// PortfolioMarginMarginLoanRecord holds a margin loan record
type PortfolioMarginMarginLoanRecord struct {
	TransactionID uint64        `json:"txId"`
	Asset         currency.Code `json:"asset"`
	Principal     types.Number  `json:"principal"`
	Timestamp     types.Time    `json:"timestamp"`
	Status        string        `json:"status"`
}

// PortfolioMarginMarginMaxWithdrawResponse holds the maximum amount of an asset that can be withdrawn from the margin
// account
type PortfolioMarginMarginMaxWithdrawResponse struct {
	Amount types.Number `json:"amount"`
}

// PortfolioMarginMarginRepayRecordResponse holds a page of margin repay records
type PortfolioMarginMarginRepayRecordResponse struct {
	Rows  []PortfolioMarginMarginRepayRecord `json:"rows"`
	Total uint64                             `json:"total"`
}

// PortfolioMarginMarginRepayRecord holds a margin repay record
type PortfolioMarginMarginRepayRecord struct {
	Amount        types.Number  `json:"amount"` // total amount repaid
	Asset         currency.Code `json:"asset"`
	Interest      types.Number  `json:"interest"`
	Principal     types.Number  `json:"principal"`
	Status        string        `json:"status"`
	Timestamp     types.Time    `json:"timestamp"`
	TransactionID uint64        `json:"txId"`
}

// PortfolioMarginNegativeBalanceInterestHistoryRequest holds the parameters for a negative balance interest history
// query, shared by portfolio margin and portfolio margin pro
type PortfolioMarginNegativeBalanceInterestHistoryRequest struct {
	Asset     currency.Code
	StartTime time.Time
	EndTime   time.Time
	Size      uint64
}

// PortfolioMarginNegativeBalanceInterestResponse holds a portfolio margin negative balance interest record
type PortfolioMarginNegativeBalanceInterestResponse struct {
	Asset               currency.Code `json:"asset"`
	Interest            types.Number  `json:"interest"`
	InterestAccuredTime types.Time    `json:"interestAccuredTime"`
	InterestRate        types.Number  `json:"interestRate"` // daily rate
	Principal           types.Number  `json:"principal"`
}

// PortfolioMarginUSDTMarginedPositionInformationResponse holds a UM position
type PortfolioMarginUSDTMarginedPositionInformationResponse struct {
	EntryPrice       types.Number `json:"entryPrice"`
	Leverage         types.Number `json:"leverage"`
	MarkPrice        types.Number `json:"markPrice"`
	MaxNotionalValue types.Number `json:"maxNotionalValue"`
	PositionAmount   types.Number `json:"positionAmt"`
	Notional         types.Number `json:"notional"`
	Symbol           string       `json:"symbol"`
	UnrealisedProfit types.Number `json:"unRealizedProfit"`
	LiquidationPrice types.Number `json:"liquidationPrice"`
	PositionSide     string       `json:"positionSide"`
	UpdateTime       types.Time   `json:"updateTime"`
}

// PortfolioMarginNegativeBalanceExchangeRecordResponse holds negative balance auto exchange records
type PortfolioMarginNegativeBalanceExchangeRecordResponse struct {
	Total uint64                                         `json:"total"`
	Rows  []PortfolioMarginNegativeBalanceExchangeRecord `json:"rows"`
}

// PortfolioMarginNegativeBalanceExchangeRecord holds a negative balance auto exchange
type PortfolioMarginNegativeBalanceExchangeRecord struct {
	StartTime types.Time                                     `json:"startTime"`
	EndTime   types.Time                                     `json:"endTime"`
	Details   []PortfolioMarginNegativeBalanceExchangeDetail `json:"details"`
}

// PortfolioMarginNegativeBalanceExchangeDetail holds an asset's negative balance in an auto exchange
type PortfolioMarginNegativeBalanceExchangeDetail struct {
	Asset                currency.Code `json:"asset"`
	NegativeBalance      float64       `json:"negativeBalance"`
	NegativeMaxThreshold uint64        `json:"negativeMaxThreshold"`
}

// PortfolioMarginRateLimitResponse holds a portfolio margin order rate limit
type PortfolioMarginRateLimitResponse struct {
	RateLimitType  string `json:"rateLimitType"`
	Interval       string `json:"interval"`
	IntervalNumber uint64 `json:"intervalNum"`
	Limit          uint64 `json:"limit"`
}

// PortfolioMarginUSDTMarginedNotionalAndLeverageResponse holds a UM symbol's notional and leverage brackets
type PortfolioMarginUSDTMarginedNotionalAndLeverageResponse struct {
	Symbol string `json:"symbol"`
	// NotionalCoefficient is the user's bracket multiplier, present when it differs from 1
	NotionalCoefficient types.Number                                 `json:"notionalCoef"`
	Brackets            []PortfolioMarginUSDTMarginedNotionalBracket `json:"brackets"`
}

// PortfolioMarginUSDTMarginedNotionalBracket holds a UM leverage bracket, with caps and floors in notional value
type PortfolioMarginUSDTMarginedNotionalBracket struct {
	Bracket                uint64  `json:"bracket"`
	InitialLeverage        uint64  `json:"initialLeverage"`
	NotionalCap            float64 `json:"notionalCap"`
	NotionalFloor          float64 `json:"notionalFloor"`
	MaintenanceMarginRatio float64 `json:"maintMarginRatio"`
	Cumulative             float64 `json:"cum"` // auxiliary number for quick maintenance margin calculation
}

// PortfolioMarginCoinMarginedConditionalOrderRequest holds the parameters for a new CM conditional order
type PortfolioMarginCoinMarginedConditionalOrderRequest struct {
	Symbol currency.Pair
	Side   order.Side
	// PositionSide is BOTH in one-way mode and LONG or SHORT in hedge mode, where it is required
	PositionSide string
	// StrategyType is STOP, STOP_MARKET, TAKE_PROFIT, TAKE_PROFIT_MARKET or TRAILING_STOP_MARKET
	StrategyType string
	TimeInForce  string
	Quantity     float64
	ReduceOnly   bool
	Price        float64
	// WorkingType selects the price that triggers StopPrice: MARK_PRICE or CONTRACT_PRICE (default)
	WorkingType         string
	PriceProtect        bool
	NewClientStrategyID string
	StopPrice           float64
	ActivationPrice     float64
	// CallbackRate is a percentage from 0.1 to 5, used with TRAILING_STOP_MARKET orders
	CallbackRate float64
}

// PortfolioMarginCoinMarginedConditionalOrderResponse holds the fields returned for every CM conditional order
type PortfolioMarginCoinMarginedConditionalOrderResponse struct {
	NewClientStrategyID string       `json:"newClientStrategyId"`
	StrategyID          uint64       `json:"strategyId"`
	StrategyStatus      string       `json:"strategyStatus"`
	StrategyType        string       `json:"strategyType"`
	OriginalQuantity    types.Number `json:"origQty"`
	Price               types.Number `json:"price"`
	ReduceOnly          bool         `json:"reduceOnly"`
	Side                string       `json:"side"`
	PositionSide        string       `json:"positionSide"`
	// StopPrice does not apply to TRAILING_STOP_MARKET orders
	StopPrice   types.Number `json:"stopPrice"`
	Symbol      string       `json:"symbol"`
	TimeInForce string       `json:"timeInForce"`
	// ActivatePrice and PriceRate (the callback rate) are only returned for TRAILING_STOP_MARKET orders
	ActivatePrice types.Number `json:"activatePrice"`
	PriceRate     types.Number `json:"priceRate"`
	BookTime      types.Time   `json:"bookTime"` // order placement time
	UpdateTime    types.Time   `json:"updateTime"`
}

// NewPortfolioMarginCoinMarginedConditionalOrderResponse holds a new CM conditional order
type NewPortfolioMarginCoinMarginedConditionalOrderResponse struct {
	PortfolioMarginCoinMarginedConditionalOrderResponse
	Pair         string `json:"pair"`
	WorkingType  string `json:"workingType"`
	PriceProtect bool   `json:"priceProtect"`
}

// CancelPortfolioMarginCoinMarginedConditionalOrderResponse holds a cancelled CM conditional order
type CancelPortfolioMarginCoinMarginedConditionalOrderResponse struct {
	PortfolioMarginCoinMarginedConditionalOrderResponse
	WorkingType  string `json:"workingType"`
	PriceProtect bool   `json:"priceProtect"`
}

// PortfolioMarginCoinMarginedOrderDetailResponse holds a CM order returned by the order query endpoints
type PortfolioMarginCoinMarginedOrderDetailResponse struct {
	AveragePrice     types.Number `json:"avgPrice"`
	ClientOrderID    string       `json:"clientOrderId"`
	CumulativeBase   types.Number `json:"cumBase"`
	ExecutedQuantity types.Number `json:"executedQty"`
	OrderID          uint64       `json:"orderId"`
	OriginalQuantity types.Number `json:"origQty"`
	OriginalType     string       `json:"origType"`
	Price            types.Number `json:"price"`
	ReduceOnly       bool         `json:"reduceOnly"`
	Side             string       `json:"side"`
	Status           string       `json:"status"`
	Symbol           string       `json:"symbol"`
	Pair             string       `json:"pair"`
	PositionSide     string       `json:"positionSide"`
	Time             types.Time   `json:"time"`
	TimeInForce      string       `json:"timeInForce"`
	Type             string       `json:"type"`
	UpdateTime       types.Time   `json:"updateTime"`
}

// PortfolioMarginCoinMarginedOrderRequest holds the parameters for a new CM order
type PortfolioMarginCoinMarginedOrderRequest struct {
	Symbol currency.Pair
	Side   order.Side
	// PositionSide is BOTH in one-way mode and LONG or SHORT in hedge mode, where it is required
	PositionSide string
	// OrderType is LIMIT or MARKET
	OrderType   string
	TimeInForce string
	Quantity    float64
	// ReduceOnly cannot be sent in hedge mode
	ReduceOnly bool
	Price      float64
	// PriceMatch cannot be sent with Price
	PriceMatch           string
	NewClientOrderID     string
	NewOrderResponseType string
}

// PortfolioMarginCoinMarginedOrderResponse holds a CM order returned when it is placed or cancelled
type PortfolioMarginCoinMarginedOrderResponse struct {
	ClientOrderID      string       `json:"clientOrderId"`
	CumulativeQuantity types.Number `json:"cumQty"`
	ExecutedQuantity   types.Number `json:"executedQty"`
	OrderID            uint64       `json:"orderId"`
	OriginalQuantity   types.Number `json:"origQty"`
	Price              types.Number `json:"price"`
	ReduceOnly         bool         `json:"reduceOnly"`
	Side               string       `json:"side"`
	PositionSide       string       `json:"positionSide"`
	Status             string       `json:"status"`
	Symbol             string       `json:"symbol"`
	Pair               string       `json:"pair"`
	TimeInForce        string       `json:"timeInForce"`
	Type               string       `json:"type"`
	UpdateTime         types.Time   `json:"updateTime"`
}

// PortfolioMarginCancelledMarginOrderResponse holds an order or OCO order list cancelled by the cancel all margin open
// orders endpoint; order lists carry the list fields, Orders and OrderReports, and plain orders carry an OrderListID of
// -1
type PortfolioMarginCancelledMarginOrderResponse struct {
	Symbol                  string                                            `json:"symbol"`
	OriginalClientOrderID   string                                            `json:"origClientOrderId"`
	OrderID                 uint64                                            `json:"orderId"`
	OrderListID             int64                                             `json:"orderListId"` // -1 unless the entry is an order list
	ClientOrderID           string                                            `json:"clientOrderId"`
	Price                   types.Number                                      `json:"price"`
	OriginalQuantity        types.Number                                      `json:"origQty"`
	ExecutedQuantity        types.Number                                      `json:"executedQty"`
	CumulativeQuoteQuantity types.Number                                      `json:"cummulativeQuoteQty"`
	Status                  string                                            `json:"status"`
	TimeInForce             string                                            `json:"timeInForce"`
	Type                    string                                            `json:"type"`
	Side                    string                                            `json:"side"`
	ContingencyType         string                                            `json:"contingencyType"`
	ListStatusType          string                                            `json:"listStatusType"`
	ListOrderStatus         string                                            `json:"listOrderStatus"`
	ListClientOrderID       string                                            `json:"listClientOrderId"`
	TransactionTime         types.Time                                        `json:"transactionTime"`
	Orders                  []PortfolioMarginMarginOneCancelsTheOtherOrderLeg `json:"orders"`
	OrderReports            []PortfolioMarginCancelledMarginOrderReport       `json:"orderReports"`
}

// PortfolioMarginCancelledMarginOrderReport holds a cancelled OCO order leg returned by the cancel all margin open
// orders endpoint
type PortfolioMarginCancelledMarginOrderReport struct {
	PortfolioMarginCancelMarginOneCancelsTheOtherOrderReport
	IcebergQuantity types.Number `json:"icebergQty"`
}

// PortfolioMarginMarginOneCancelsTheOtherOrderLeg holds an order of a margin OCO order list
type PortfolioMarginMarginOneCancelsTheOtherOrderLeg struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// PortfolioMarginMarginOneCancelsTheOtherOrderResponse holds a margin OCO order list
type PortfolioMarginMarginOneCancelsTheOtherOrderResponse struct {
	OrderListID       uint64                                            `json:"orderListId"`
	ContingencyType   string                                            `json:"contingencyType"`
	ListStatusType    string                                            `json:"listStatusType"`
	ListOrderStatus   string                                            `json:"listOrderStatus"`
	ListClientOrderID string                                            `json:"listClientOrderId"`
	TransactionTime   types.Time                                        `json:"transactionTime"`
	Symbol            string                                            `json:"symbol"`
	Orders            []PortfolioMarginMarginOneCancelsTheOtherOrderLeg `json:"orders"`
}

// PortfolioMarginCancelMarginOneCancelsTheOtherOrderRequest holds the parameters for cancelling a margin OCO order list
type PortfolioMarginCancelMarginOneCancelsTheOtherOrderRequest struct {
	Symbol currency.Pair
	// OrderListID or ListClientOrderID identifies the order list
	OrderListID       uint64
	ListClientOrderID string
	// NewClientOrderID identifies this cancellation
	NewClientOrderID string
}

// PortfolioMarginCancelMarginOneCancelsTheOtherOrderResponse holds a cancelled margin OCO order list
type PortfolioMarginCancelMarginOneCancelsTheOtherOrderResponse struct {
	PortfolioMarginMarginOneCancelsTheOtherOrderResponse
	OrderReports []PortfolioMarginCancelMarginOneCancelsTheOtherOrderReport `json:"orderReports"`
}

// PortfolioMarginCancelMarginOneCancelsTheOtherOrderReport holds a cancelled margin OCO order leg
type PortfolioMarginCancelMarginOneCancelsTheOtherOrderReport struct {
	Symbol                  string       `json:"symbol"`
	OriginalClientOrderID   string       `json:"origClientOrderId"`
	OrderID                 uint64       `json:"orderId"`
	OrderListID             uint64       `json:"orderListId"`
	ClientOrderID           string       `json:"clientOrderId"`
	Price                   types.Number `json:"price"`
	OriginalQuantity        types.Number `json:"origQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	CumulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                  string       `json:"status"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	Side                    string       `json:"side"`
	StopPrice               types.Number `json:"stopPrice"`
}

// PortfolioMarginMarginOrderResponse holds a margin order returned by the margin order query endpoints
type PortfolioMarginMarginOrderResponse struct {
	ClientOrderID string `json:"clientOrderId"`
	// CumulativeQuoteQuantity is negative for some historical orders whose data is unavailable
	CumulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	IcebergQuantity         types.Number `json:"icebergQty"`
	IsWorking               bool         `json:"isWorking"`
	OrderID                 uint64       `json:"orderId"`
	OriginalQuantity        types.Number `json:"origQty"`
	Price                   types.Number `json:"price"`
	Side                    string       `json:"side"`
	Status                  string       `json:"status"`
	StopPrice               types.Number `json:"stopPrice"`
	Symbol                  string       `json:"symbol"`
	Time                    types.Time   `json:"time"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	UpdateTime              types.Time   `json:"updateTime"`
	AccountID               uint64       `json:"accountId"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	// PreventedMatchID and PreventedQuantity are null unless self-trade prevention expired the order
	PreventedMatchID  types.Number `json:"preventedMatchId"`
	PreventedQuantity types.Number `json:"preventedQuantity"`
}

// PortfolioMarginMarginOrderRequest holds the parameters for a new portfolio margin cross margin order
type PortfolioMarginMarginOrderRequest struct {
	Symbol currency.Pair
	Side   order.Side
	// OrderType is LIMIT, MARKET, STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, TAKE_PROFIT_LIMIT or LIMIT_MAKER
	OrderType          string
	Quantity           float64
	QuoteOrderQuantity float64
	Price              float64
	// StopPrice is used with STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT and TAKE_PROFIT_LIMIT orders
	StopPrice            float64
	NewClientOrderID     string
	NewOrderResponseType string
	// IcebergQuantity is used with LIMIT, STOP_LOSS_LIMIT and TAKE_PROFIT_LIMIT orders
	IcebergQuantity float64
	// SideEffectType is NO_SIDE_EFFECT (default), MARGIN_BUY, AUTO_REPAY or AUTO_BORROW_REPAY
	SideEffectType          string
	TimeInForce             string
	SelfTradePreventionMode string
	// AutoRepayAtCancel sets whether debt from a MARGIN_BUY or AUTO_BORROW_REPAY order is repaid when the order is
	// cancelled; nil leaves the default of true
	AutoRepayAtCancel *bool
}

// PortfolioMarginNewMarginOrderResponse holds a new margin order
type PortfolioMarginNewMarginOrderResponse struct {
	Symbol                  string       `json:"symbol"`
	OrderID                 uint64       `json:"orderId"`
	ClientOrderID           string       `json:"clientOrderId"`
	TransactTime            types.Time   `json:"transactTime"`
	Price                   types.Number `json:"price"`
	OriginalQuantity        types.Number `json:"origQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	CumulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                  string       `json:"status"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	Side                    string       `json:"side"`
	// MarginBuyBorrowAmount and MarginBuyBorrowAsset are only returned when the order borrowed
	MarginBuyBorrowAmount types.Number                     `json:"marginBuyBorrowAmount"`
	MarginBuyBorrowAsset  currency.Code                    `json:"marginBuyBorrowAsset"`
	Fills                 []PortfolioMarginMarginOrderFill `json:"fills"`
}

// PortfolioMarginMarginOrderFill holds a fill of a new margin order
type PortfolioMarginMarginOrderFill struct {
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
}

// PortfolioMarginCancelMarginOrderRequest holds the parameters for cancelling a margin order
type PortfolioMarginCancelMarginOrderRequest struct {
	Symbol currency.Pair
	// OrderID or OriginalClientOrderID identifies the order
	OrderID               uint64
	OriginalClientOrderID string
	// NewClientOrderID identifies this cancellation
	NewClientOrderID string
}

// PortfolioMarginCancelMarginOrderResponse holds a cancelled margin order
type PortfolioMarginCancelMarginOrderResponse struct {
	Symbol                  string       `json:"symbol"`
	OrderID                 uint64       `json:"orderId"`
	OriginalClientOrderID   string       `json:"origClientOrderId"`
	ClientOrderID           string       `json:"clientOrderId"`
	Price                   types.Number `json:"price"`
	OriginalQuantity        types.Number `json:"origQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	CumulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                  string       `json:"status"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	Side                    string       `json:"side"`
}

// PortfolioMarginUSDTMarginedOrderDetailResponse holds a UM order returned by the order query endpoints
type PortfolioMarginUSDTMarginedOrderDetailResponse struct {
	AveragePrice            types.Number `json:"avgPrice"`
	ClientOrderID           string       `json:"clientOrderId"`
	CumulativeQuote         types.Number `json:"cumQuote"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	OrderID                 uint64       `json:"orderId"`
	OriginalQuantity        types.Number `json:"origQty"`
	OriginalType            string       `json:"origType"`
	Price                   types.Number `json:"price"`
	ReduceOnly              bool         `json:"reduceOnly"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"positionSide"`
	Status                  string       `json:"status"`
	Symbol                  string       `json:"symbol"`
	Time                    types.Time   `json:"time"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	UpdateTime              types.Time   `json:"updateTime"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	GoodTillDate            types.Time   `json:"goodTillDate"` // automatic cancellation time of a GTD order
	PriceMatch              string       `json:"priceMatch"`
}

// PortfolioMarginUSDTMarginedOrderRequest holds the parameters for a new UM order
type PortfolioMarginUSDTMarginedOrderRequest struct {
	Symbol currency.Pair
	Side   order.Side
	// PositionSide is BOTH in one-way mode and LONG or SHORT in hedge mode, where it is required
	PositionSide string
	// OrderType is LIMIT or MARKET
	OrderType   string
	TimeInForce string
	Quantity    float64
	// ReduceOnly cannot be sent in hedge mode
	ReduceOnly           bool
	Price                float64
	NewClientOrderID     string
	NewOrderResponseType string
	// PriceMatch cannot be sent with Price
	PriceMatch string
	// SelfTradePreventionMode is only effective with the IOC, GTC and GTD time in force values
	SelfTradePreventionMode string
	// GoodTillDate is required for GTD orders; only its seconds are used
	GoodTillDate time.Time
}

// PortfolioMarginUSDTMarginedOrderResponse holds a UM order returned when it is placed or cancelled
type PortfolioMarginUSDTMarginedOrderResponse struct {
	ClientOrderID           string       `json:"clientOrderId"`
	CumulativeQuantity      types.Number `json:"cumQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	OrderID                 uint64       `json:"orderId"`
	OriginalQuantity        types.Number `json:"origQty"`
	Price                   types.Number `json:"price"`
	ReduceOnly              bool         `json:"reduceOnly"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"positionSide"`
	Status                  string       `json:"status"`
	Symbol                  string       `json:"symbol"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	GoodTillDate            types.Time   `json:"goodTillDate"` // automatic cancellation time of a GTD order
	UpdateTime              types.Time   `json:"updateTime"`
	PriceMatch              string       `json:"priceMatch"`
}

// PortfolioMarginCoinMarginedAccountTradeListRequest holds the parameters for a CM account trade list query; Symbol and
// Pair are mutually exclusive, and FromID cannot be combined with Pair or a time range
type PortfolioMarginCoinMarginedAccountTradeListRequest struct {
	Symbol currency.Pair
	// Pair, such as BTCUSD, is the base of a COIN-M contract pair and returns trades of every symbol of the pair
	Pair      currency.Code
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64
	Limit     uint64
}

// PortfolioMarginCoinMarginedAccountTradeResponse holds a CM account trade
type PortfolioMarginCoinMarginedAccountTradeResponse struct {
	Symbol              string        `json:"symbol"`
	ID                  uint64        `json:"id"`
	OrderID             uint64        `json:"orderId"`
	Pair                string        `json:"pair"`
	Side                string        `json:"side"`
	Price               types.Number  `json:"price"`
	QuantityInContracts types.Number  `json:"qty"`
	RealisedPNL         types.Number  `json:"realizedPnl"`
	MarginAsset         currency.Code `json:"marginAsset"`
	BaseQuantity        types.Number  `json:"baseQty"`
	Commission          types.Number  `json:"commission"`
	CommissionAsset     currency.Code `json:"commissionAsset"`
	Time                types.Time    `json:"time"`
	PositionSide        string        `json:"positionSide"`
	Buyer               bool          `json:"buyer"`
	Maker               bool          `json:"maker"`
}

// PortfolioMarginCoinMarginedAutoDeleveragingQuantileEstimationResponse holds a CM symbol's auto-deleveraging quantiles
type PortfolioMarginCoinMarginedAutoDeleveragingQuantileEstimationResponse struct {
	Symbol                   string                                              `json:"symbol"`
	AutoDeleveragingQuantile PortfolioMarginCoinMarginedAutoDeleveragingQuantile `json:"adlQuantile"`
}

// PortfolioMarginCoinMarginedAutoDeleveragingQuantile holds auto-deleveraging queue quantiles from 0 (low) to 4 (high)
// per position side; a cross margined hedge mode position reports HEDGE instead of BOTH
type PortfolioMarginCoinMarginedAutoDeleveragingQuantile struct {
	Long  uint64 `json:"LONG"`
	Short uint64 `json:"SHORT"`
	Hedge uint64 `json:"HEDGE"` // only a sign, its value is meaningless
	Both  uint64 `json:"BOTH"`
}

// PortfolioMarginMarginOneCancelsTheOtherOrderRequest holds the parameters for a new margin OCO order list
type PortfolioMarginMarginOneCancelsTheOtherOrderRequest struct {
	Symbol            currency.Pair
	ListClientOrderID string
	Side              order.Side
	// Quantity applies to both legs
	Quantity             float64
	LimitClientOrderID   string
	Price                float64
	LimitIcebergQuantity float64
	StopClientOrderID    string
	StopPrice            float64
	// StopLimitPrice requires StopLimitTimeInForce
	StopLimitPrice       float64
	StopIcebergQuantity  float64
	StopLimitTimeInForce string
	NewOrderResponseType string
	// SideEffectType is NO_SIDE_EFFECT (default), MARGIN_BUY or AUTO_REPAY
	SideEffectType string
}

// PortfolioMarginNewMarginOneCancelsTheOtherOrderResponse holds a new margin OCO order list
type PortfolioMarginNewMarginOneCancelsTheOtherOrderResponse struct {
	PortfolioMarginMarginOneCancelsTheOtherOrderResponse
	// MarginBuyBorrowAmount and MarginBuyBorrowAsset are only returned when the order list borrowed
	MarginBuyBorrowAmount types.Number                                            `json:"marginBuyBorrowAmount"`
	MarginBuyBorrowAsset  currency.Code                                           `json:"marginBuyBorrowAsset"`
	OrderReports          []PortfolioMarginNewMarginOneCancelsTheOtherOrderReport `json:"orderReports"`
}

// PortfolioMarginNewMarginOneCancelsTheOtherOrderReport holds an order of a new margin OCO order list
type PortfolioMarginNewMarginOneCancelsTheOtherOrderReport struct {
	Symbol                  string       `json:"symbol"`
	OrderID                 uint64       `json:"orderId"`
	OrderListID             uint64       `json:"orderListId"`
	ClientOrderID           string       `json:"clientOrderId"`
	TransactTime            types.Time   `json:"transactTime"`
	Price                   types.Number `json:"price"`
	OriginalQuantity        types.Number `json:"origQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	CumulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                  string       `json:"status"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	Side                    string       `json:"side"`
	StopPrice               types.Number `json:"stopPrice"`
}

// PortfolioMarginMarginTradeListRequest holds the parameters for a margin account trade list query
type PortfolioMarginMarginTradeListRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64
	Limit     uint64
}

// PortfolioMarginMarginTradeResponse holds a margin account trade
type PortfolioMarginMarginTradeResponse struct {
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
	Time            types.Time    `json:"time"`
}

// PortfolioMarginCoinMarginedConditionalOrdersRequest holds the parameters for a query of all CM conditional orders
type PortfolioMarginCoinMarginedConditionalOrdersRequest struct {
	Symbol     currency.Pair
	StrategyID uint64
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint64
}

// PortfolioMarginCoinMarginedConditionalOrderRecordResponse holds a CM conditional order with the order it triggered,
// if any
type PortfolioMarginCoinMarginedConditionalOrderRecordResponse struct {
	PortfolioMarginCoinMarginedConditionalOrderResponse
	// OrderID, Status and Type describe the order placed when the strategy triggered
	OrderID     uint64     `json:"orderId"`
	Status      string     `json:"status"`
	TriggerTime types.Time `json:"triggerTime"`
	Type        string     `json:"type"`
}

// PortfolioMarginCoinMarginedOrdersRequest holds the parameters for a query of all CM orders; Symbol or Pair is
// required
type PortfolioMarginCoinMarginedOrdersRequest struct {
	Symbol currency.Pair
	// Pair, such as BTCUSD, is the base of a COIN-M contract pair
	Pair currency.Code
	// OrderID returns orders from this order ID onwards
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// PortfolioMarginMarginOrdersRequest holds the parameters for a query of all margin account orders
type PortfolioMarginMarginOrdersRequest struct {
	Symbol currency.Pair
	// OrderID returns orders from this order ID onwards
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// PortfolioMarginUSDTMarginedOrdersRequest holds the parameters for a query of all UM orders
type PortfolioMarginUSDTMarginedOrdersRequest struct {
	Symbol currency.Pair
	// OrderID returns orders from this order ID onwards
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// PortfolioMarginCoinMarginedConditionalOrderHistoryResponse holds a CM conditional order from the order history
type PortfolioMarginCoinMarginedConditionalOrderHistoryResponse struct {
	PortfolioMarginCoinMarginedConditionalOrderRecordResponse
	WorkingType  string `json:"workingType"`
	PriceProtect bool   `json:"priceProtect"`
	PriceMatch   string `json:"priceMatch"`
}

// PortfolioMarginMarginAllOneCancelsTheOtherOrdersRequest holds the parameters for a query of all margin OCO order
// lists
type PortfolioMarginMarginAllOneCancelsTheOtherOrdersRequest struct {
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// PortfolioMarginForceOrdersRequest holds the parameters for a UM or CM force orders query
type PortfolioMarginForceOrdersRequest struct {
	Symbol currency.Pair
	// AutoCloseType is LIQUIDATION or ADL; both are returned when it is empty
	AutoCloseType string
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64
}

// PortfolioMarginCoinMarginedForceOrderResponse holds a CM liquidation or auto-deleveraging order
type PortfolioMarginCoinMarginedForceOrderResponse struct {
	OrderID          uint64       `json:"orderId"`
	Symbol           string       `json:"symbol"`
	Pair             string       `json:"pair"`
	Status           string       `json:"status"`
	ClientOrderID    string       `json:"clientOrderId"`
	Price            types.Number `json:"price"`
	AveragePrice     types.Number `json:"avgPrice"`
	OriginalQuantity types.Number `json:"origQty"`
	ExecutedQuantity types.Number `json:"executedQty"`
	CumulativeBase   types.Number `json:"cumBase"`
	TimeInForce      string       `json:"timeInForce"`
	Type             string       `json:"type"`
	ReduceOnly       bool         `json:"reduceOnly"`
	Side             string       `json:"side"`
	PositionSide     string       `json:"positionSide"`
	OriginalType     string       `json:"origType"`
	Time             types.Time   `json:"time"`
	UpdateTime       types.Time   `json:"updateTime"`
}

// PortfolioMarginMarginForceOrdersRequest holds the parameters for a margin force orders query
type PortfolioMarginMarginForceOrdersRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
}

// PortfolioMarginMarginForceOrderResponse holds a page of margin force orders
type PortfolioMarginMarginForceOrderResponse struct {
	Rows  []PortfolioMarginMarginForceOrder `json:"rows"`
	Total uint64                            `json:"total"`
}

// PortfolioMarginMarginForceOrder holds a margin liquidation order
type PortfolioMarginMarginForceOrder struct {
	AveragePrice     types.Number `json:"avgPrice"`
	ExecutedQuantity types.Number `json:"executedQty"`
	OrderID          uint64       `json:"orderId"`
	Price            types.Number `json:"price"`
	Quantity         types.Number `json:"qty"`
	Side             string       `json:"side"`
	Symbol           string       `json:"symbol"`
	TimeInForce      string       `json:"timeInForce"`
	UpdatedTime      types.Time   `json:"updatedTime"`
}

// PortfolioMarginUSDTMarginedForceOrderResponse holds a UM liquidation or auto-deleveraging order
type PortfolioMarginUSDTMarginedForceOrderResponse struct {
	OrderID          uint64       `json:"orderId"`
	Symbol           string       `json:"symbol"`
	Status           string       `json:"status"`
	ClientOrderID    string       `json:"clientOrderId"`
	Price            types.Number `json:"price"`
	AveragePrice     types.Number `json:"avgPrice"`
	OriginalQuantity types.Number `json:"origQty"`
	ExecutedQuantity types.Number `json:"executedQty"`
	CumulativeQuote  types.Number `json:"cumQuote"`
	TimeInForce      string       `json:"timeInForce"`
	Type             string       `json:"type"`
	ReduceOnly       bool         `json:"reduceOnly"`
	Side             string       `json:"side"`
	PositionSide     string       `json:"positionSide"`
	OriginalType     string       `json:"origType"`
	Time             types.Time   `json:"time"`
	UpdateTime       types.Time   `json:"updateTime"`
}

// PortfolioMarginUSDTMarginedAccountTradeListRequest holds the parameters for a UM account trade list query; FromID
// cannot be combined with a time range
type PortfolioMarginUSDTMarginedAccountTradeListRequest struct {
	Symbol    currency.Pair
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64
	Limit     uint64
}

// PortfolioMarginUSDTMarginedAccountTradeResponse holds a UM account trade
type PortfolioMarginUSDTMarginedAccountTradeResponse struct {
	Symbol          string        `json:"symbol"`
	ID              uint64        `json:"id"`
	OrderID         uint64        `json:"orderId"`
	Side            string        `json:"side"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	RealisedPNL     types.Number  `json:"realizedPnl"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Time            types.Time    `json:"time"`
	Buyer           bool          `json:"buyer"`
	Maker           bool          `json:"maker"`
	PositionSide    string        `json:"positionSide"`
}

// PortfolioMarginUSDTMarginedAutoDeleveragingQuantileEstimationResponse holds a UM symbol's auto-deleveraging quantiles
type PortfolioMarginUSDTMarginedAutoDeleveragingQuantileEstimationResponse struct {
	Symbol                   string                                              `json:"symbol"`
	AutoDeleveragingQuantile PortfolioMarginUSDTMarginedAutoDeleveragingQuantile `json:"adlQuantile"`
}

// PortfolioMarginUSDTMarginedAutoDeleveragingQuantile holds auto-deleveraging queue quantiles from 0 (low) to 4 (high)
// per position side
type PortfolioMarginUSDTMarginedAutoDeleveragingQuantile struct {
	Long  uint64 `json:"LONG"`
	Short uint64 `json:"SHORT"`
	Both  uint64 `json:"BOTH"`
}

// PortfolioMarginUSDTMarginedAlgoOrderRequest holds the parameters for a new UM algo (conditional) order
type PortfolioMarginUSDTMarginedAlgoOrderRequest struct {
	// AlgoType only supports CONDITIONAL
	AlgoType string
	Symbol   currency.Pair
	Side     order.Side
	// PositionSide is BOTH in one-way mode and LONG or SHORT in hedge mode
	PositionSide string
	// OrderType is STOP, TAKE_PROFIT, STOP_MARKET, TAKE_PROFIT_MARKET or TRAILING_STOP_MARKET
	OrderType   string
	TimeInForce string
	// Quantity cannot be sent with ClosePosition
	Quantity     float64
	Price        float64
	TriggerPrice float64
	// WorkingType selects the price that triggers TriggerPrice: MARK_PRICE or CONTRACT_PRICE (default)
	WorkingType string
	// PriceMatch cannot be sent with Price
	PriceMatch string
	// ClosePosition closes the whole position with a STOP_MARKET or TAKE_PROFIT_MARKET order
	ClosePosition bool
	PriceProtect  bool
	// ReduceOnly cannot be sent in hedge mode or with ClosePosition
	ReduceOnly bool
	// ActivatePrice and CallbackRate (a percentage from 0.1 to 10) are used with TRAILING_STOP_MARKET orders
	ActivatePrice           float64
	CallbackRate            float64
	ClientAlgoID            string
	NewOrderResponseType    string
	SelfTradePreventionMode string
	// GoodTillDate is required for GTD orders
	GoodTillDate time.Time
}

// PortfolioMarginUSDTMarginedAlgoOrderResponse holds a UM algo order as placing one and the open algo orders query
// return it
type PortfolioMarginUSDTMarginedAlgoOrderResponse struct {
	AlgoID                  uint64       `json:"algoId"`
	ClientAlgoID            string       `json:"clientAlgoId"`
	AlgoType                string       `json:"algoType"`
	OrderType               string       `json:"orderType"`
	Symbol                  string       `json:"symbol"`
	Side                    string       `json:"side"`
	PositionSide            string       `json:"positionSide"`
	TimeInForce             string       `json:"timeInForce"`
	Quantity                types.Number `json:"quantity"`
	AlgoStatus              string       `json:"algoStatus"`
	TriggerPrice            types.Number `json:"triggerPrice"`
	Price                   types.Number `json:"price"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	WorkingType             string       `json:"workingType"`
	PriceMatch              string       `json:"priceMatch"`
	ClosePosition           bool         `json:"closePosition"`
	PriceProtect            bool         `json:"priceProtect"`
	ReduceOnly              bool         `json:"reduceOnly"`
	ActivatePrice           types.Number `json:"activatePrice"`
	CallbackRate            types.Number `json:"callbackRate"`
	CreateTime              types.Time   `json:"createTime"`
	UpdateTime              types.Time   `json:"updateTime"`
	TriggerTime             types.Time   `json:"triggerTime"`
	GoodTillDate            types.Time   `json:"goodTillDate"`
}

// CancelPortfolioMarginUSDTMarginedAlgoOrderResponse holds the result of a UM algo order cancellation
type CancelPortfolioMarginUSDTMarginedAlgoOrderResponse struct {
	Complete bool `json:"complete"`
}

// PortfolioMarginUSDTMarginedAlgoOrderDetailResponse holds a UM algo order as the order and history queries return it,
// with the order its trigger placed
type PortfolioMarginUSDTMarginedAlgoOrderDetailResponse struct {
	PortfolioMarginUSDTMarginedAlgoOrderResponse
	ActualOrderID string       `json:"actualOrderId"` // empty until the algo order triggers
	ActualPrice   types.Number `json:"actualPrice"`
}

// PortfolioMarginUSDTMarginedAlgoOrderHistoryRequest holds the parameters of Query UM Algo Order History
type PortfolioMarginUSDTMarginedAlgoOrderHistoryRequest struct {
	Symbol    currency.Pair
	AlgoID    uint64 // returns orders from this algo ID on
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 500 when zero, at most 1000
}
