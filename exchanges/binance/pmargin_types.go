package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AccountBalance holds a portfolio margin asset balance across the margin, UM and CM accounts
type AccountBalance struct {
	Asset              currency.Code `json:"asset"`
	TotalWalletBalance types.Number  `json:"totalWalletBalance"`
	// CrossMarginAsset is crossMarginFree plus crossMarginLocked; it is only returned when no asset is requested
	CrossMarginAsset    types.Number `json:"crossMarginAsset"`
	CrossMarginBorrowed types.Number `json:"crossMarginBorrowed"`
	CrossMarginFree     types.Number `json:"crossMarginFree"`
	CrossMarginInterest types.Number `json:"crossMarginInterest"`
	CrossMarginLocked   types.Number `json:"crossMarginLocked"`
	UMWalletBalance     types.Number `json:"umWalletBalance"`
	UMUnrealizedPNL     types.Number `json:"umUnrealizedPNL"`
	CMWalletBalance     types.Number `json:"cmWalletBalance"`
	CMUnrealizedPNL     types.Number `json:"cmUnrealizedPNL"`
	UpdateTime          types.Time   `json:"updateTime"`
	NegativeBalance     types.Number `json:"negativeBalance"`
}

// AccountBalanceResponse holds the balances returned by the account balance endpoint, which answers with a single
// object when an asset is requested and an array otherwise
type AccountBalanceResponse []AccountBalance

// UnmarshalJSON decodes either response shape into a slice of balances
func (a *AccountBalanceResponse) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) != 0 && data[0] == '[' {
		var balances []AccountBalance
		if err := json.Unmarshal(data, &balances); err != nil {
			return err
		}
		*a = balances
		return nil
	}
	var balance AccountBalance
	if err := json.Unmarshal(data, &balance); err != nil {
		return err
	}
	*a = AccountBalanceResponse{balance}
	return nil
}

// PMAccountInformationResponse holds portfolio margin account information, with amounts in USD
type PMAccountInformationResponse struct {
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

// PMTransactionResponse holds the transaction ID returned by portfolio margin transfer, borrow and repay endpoints
type PMTransactionResponse struct {
	TransactionID uint64 `json:"tranId"`
}

// PMAutoRepayFuturesStatusResponse holds whether futures negative balances are repaid automatically
type PMAutoRepayFuturesStatusResponse struct {
	AutoRepay bool `json:"autoRepay"`
}

// PMMessageResponse holds the message returned by portfolio margin endpoints that only report success
type PMMessageResponse struct {
	Message string `json:"msg"`
}

// CMInitialLeverageResponse holds a COIN-M symbol's initial leverage after a change
type CMInitialLeverageResponse struct {
	Leverage    uint64       `json:"leverage"`
	MaxQuantity types.Number `json:"maxQty"` // maximum quantity of the base asset
	Symbol      string       `json:"symbol"`
}

// PMPositionModeResponse holds whether hedge mode (dual side position) is enabled
type PMPositionModeResponse struct {
	DualSidePosition bool `json:"dualSidePosition"`
}

// SuccessResponse holds the code and message returned by portfolio margin endpoints that only report success
type SuccessResponse struct {
	// Code is quoted by the cancel all CM open conditional orders endpoint and bare everywhere else
	Code    types.Number `json:"code"`
	Message string       `json:"msg"`
}

// UMInitialLeverageResponse holds a USDⓈ-M symbol's initial leverage after a change
type UMInitialLeverageResponse struct {
	Leverage         uint64       `json:"leverage"`
	MaxNotionalValue types.Number `json:"maxNotionalValue"`
	Symbol           string       `json:"symbol"`
}

// CMNotionalAndLeverage holds a COIN-M symbol's notional and leverage brackets
type CMNotionalAndLeverage struct {
	Symbol   string              `json:"symbol"`
	Brackets []CMNotionalBracket `json:"brackets"`
}

// CMNotionalBracket holds a COIN-M leverage bracket, with caps and floors in base asset quantity
type CMNotionalBracket struct {
	Bracket                uint64  `json:"bracket"`
	InitialLeverage        uint64  `json:"initialLeverage"`
	QuantityCap            float64 `json:"qtyCap"`
	QuantityFloor          float64 `json:"qtyFloor"`
	MaintenanceMarginRatio float64 `json:"maintMarginRatio"`
	Cumulative             float64 `json:"cum"` // auxiliary number for quick maintenance margin calculation
}

// PMAccountDetailAsset holds a UM or CM account asset
type PMAccountDetailAsset struct {
	Asset                  currency.Code `json:"asset"`
	CrossWalletBalance     types.Number  `json:"crossWalletBalance"`
	CrossUnrealizedPNL     types.Number  `json:"crossUnPnl"`
	MaintenanceMargin      types.Number  `json:"maintMargin"`
	InitialMargin          types.Number  `json:"initialMargin"`
	PositionInitialMargin  types.Number  `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number  `json:"openOrderInitialMargin"`
	UpdateTime             types.Time    `json:"updateTime"`
}

// CMAccountDetailPosition holds a CM account position
type CMAccountDetailPosition struct {
	Symbol                 string       `json:"symbol"`
	PositionAmount         types.Number `json:"positionAmt"`
	InitialMargin          types.Number `json:"initialMargin"`
	MaintenanceMargin      types.Number `json:"maintMargin"`
	UnrealizedProfit       types.Number `json:"unrealizedProfit"`
	PositionInitialMargin  types.Number `json:"positionInitialMargin"`
	OpenOrderInitialMargin types.Number `json:"openOrderInitialMargin"`
	Leverage               types.Number `json:"leverage"`
	PositionSide           string       `json:"positionSide"`
	EntryPrice             types.Number `json:"entryPrice"`
	MaxQuantity            types.Number `json:"maxQty"` // maximum quantity of the base asset
	UpdateTime             types.Time   `json:"updateTime"`
}

// CMAccountDetailResponse holds CM account assets and positions
type CMAccountDetailResponse struct {
	Assets    []PMAccountDetailAsset    `json:"assets"`
	Positions []CMAccountDetailPosition `json:"positions"`
}

// PMIncomeHistoryRequest holds the parameters for a UM or CM income history query
type PMIncomeHistoryRequest struct {
	Symbol     currency.Pair
	IncomeType string
	StartTime  time.Time
	EndTime    time.Time
	Page       uint64
	Limit      uint64
}

// IncomeItem holds a UM or CM income history record
type IncomeItem struct {
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

// PMMarginInterestHistoryRequest holds the parameters for a margin borrow or loan interest history query
type PMMarginInterestHistoryRequest struct {
	Asset     currency.Code
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
	// Archived queries data older than six months
	Archived bool
}

// PMMarginInterestHistoryResponse holds a page of margin borrow or loan interest records
type PMMarginInterestHistoryResponse struct {
	Rows  []PMMarginInterest `json:"rows"`
	Total uint64             `json:"total"`
}

// PMMarginInterest holds a margin borrow or loan interest record
type PMMarginInterest struct {
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

// UMAccountDetailPosition holds a UM account position
type UMAccountDetailPosition struct {
	Symbol                 string       `json:"symbol"`
	InitialMargin          types.Number `json:"initialMargin"`
	MaintenanceMargin      types.Number `json:"maintMargin"`
	UnrealizedProfit       types.Number `json:"unrealizedProfit"`
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

// UMAccountDetailResponse holds UM account assets and positions
type UMAccountDetailResponse struct {
	Assets    []PMAccountDetailAsset    `json:"assets"`
	Positions []UMAccountDetailPosition `json:"positions"`
}

// PMCommissionRateResponse holds a UM or CM symbol's commission rates
type PMCommissionRateResponse struct {
	Symbol              string       `json:"symbol"`
	MakerCommissionRate types.Number `json:"makerCommissionRate"`
	TakerCommissionRate types.Number `json:"takerCommissionRate"`
}

// PMMarginMaxBorrowResponse holds the maximum amount of an asset the margin account can borrow
type PMMarginMaxBorrowResponse struct {
	// Amount is the current maximum borrowable amount given system availability
	Amount types.Number `json:"amount"`
	// BorrowLimit is the maximum borrowable amount allowed by the account level
	BorrowLimit types.Number `json:"borrowLimit"`
}

// PMTradingQuantitativeRulesIndicatorsResponse holds the UM trading quantitative rules indicators
type PMTradingQuantitativeRulesIndicatorsResponse struct {
	// Indicators is keyed by symbol, with account wide indicators under ACCOUNT
	Indicators map[string][]PMTradingIndicator `json:"indicators"`
	UpdateTime types.Time                      `json:"updateTime"`
}

// PMTradingIndicator holds a trading quantitative rules indicator
type PMTradingIndicator struct {
	IsLocked           bool       `json:"isLocked"`
	PlannedRecoverTime types.Time `json:"plannedRecoverTime"`
	Indicator          string     `json:"indicator"`
	Value              float64    `json:"value"`
	TriggerValue       float64    `json:"triggerValue"`
}

// CMPositionInformation holds a CM position
type CMPositionInformation struct {
	Symbol           string       `json:"symbol"`
	PositionAmount   types.Number `json:"positionAmt"`
	EntryPrice       types.Number `json:"entryPrice"`
	MarkPrice        types.Number `json:"markPrice"`
	UnrealizedProfit types.Number `json:"unRealizedProfit"`
	LiquidationPrice types.Number `json:"liquidationPrice"`
	Leverage         types.Number `json:"leverage"`
	PositionSide     string       `json:"positionSide"`
	UpdateTime       types.Time   `json:"updateTime"`
	MaxQuantity      types.Number `json:"maxQty"` // maximum quantity of the base asset
	NotionalValue    types.Number `json:"notionalValue"`
}

// PMMarginLoanRepayRecordRequest holds the parameters for a margin loan or repay record query
type PMMarginLoanRepayRecordRequest struct {
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

// PMMarginLoanRecordResponse holds a page of margin loan records
type PMMarginLoanRecordResponse struct {
	Rows  []PMMarginLoanRecord `json:"rows"`
	Total uint64               `json:"total"`
}

// PMMarginLoanRecord holds a margin loan record
type PMMarginLoanRecord struct {
	TransactionID uint64        `json:"txId"`
	Asset         currency.Code `json:"asset"`
	Principal     types.Number  `json:"principal"`
	Timestamp     types.Time    `json:"timestamp"`
	Status        string        `json:"status"`
}

// PMMarginMaxWithdrawResponse holds the maximum amount of an asset that can be withdrawn from the margin account
type PMMarginMaxWithdrawResponse struct {
	Amount types.Number `json:"amount"`
}

// PMMarginRepayRecordResponse holds a page of margin repay records
type PMMarginRepayRecordResponse struct {
	Rows  []PMMarginRepayRecord `json:"rows"`
	Total uint64                `json:"total"`
}

// PMMarginRepayRecord holds a margin repay record
type PMMarginRepayRecord struct {
	Amount        types.Number  `json:"amount"` // total amount repaid
	Asset         currency.Code `json:"asset"`
	Interest      types.Number  `json:"interest"`
	Principal     types.Number  `json:"principal"`
	Status        string        `json:"status"`
	Timestamp     types.Time    `json:"timestamp"`
	TransactionID uint64        `json:"txId"`
}

// PMNegativeBalanceInterestHistoryRequest holds the parameters for a negative balance interest history query, shared
// by portfolio margin and portfolio margin pro
type PMNegativeBalanceInterestHistoryRequest struct {
	Asset     currency.Code
	StartTime time.Time
	EndTime   time.Time
	Size      uint64
}

// PortfolioMarginNegativeBalanceInterest holds a portfolio margin negative balance interest record
type PortfolioMarginNegativeBalanceInterest struct {
	Asset               currency.Code `json:"asset"`
	Interest            types.Number  `json:"interest"`
	InterestAccuredTime types.Time    `json:"interestAccuredTime"`
	InterestRate        types.Number  `json:"interestRate"` // daily rate
	Principal           types.Number  `json:"principal"`
}

// UMPositionInformation holds a UM position
type UMPositionInformation struct {
	EntryPrice       types.Number `json:"entryPrice"`
	Leverage         types.Number `json:"leverage"`
	MarkPrice        types.Number `json:"markPrice"`
	MaxNotionalValue types.Number `json:"maxNotionalValue"`
	PositionAmount   types.Number `json:"positionAmt"`
	Notional         types.Number `json:"notional"`
	Symbol           string       `json:"symbol"`
	UnrealizedProfit types.Number `json:"unRealizedProfit"`
	LiquidationPrice types.Number `json:"liquidationPrice"`
	PositionSide     string       `json:"positionSide"`
	UpdateTime       types.Time   `json:"updateTime"`
}

// PMNegativeBalanceExchangeRecordResponse holds negative balance auto exchange records
type PMNegativeBalanceExchangeRecordResponse struct {
	Total uint64                            `json:"total"`
	Rows  []PMNegativeBalanceExchangeRecord `json:"rows"`
}

// PMNegativeBalanceExchangeRecord holds a negative balance auto exchange
type PMNegativeBalanceExchangeRecord struct {
	StartTime types.Time                        `json:"startTime"`
	EndTime   types.Time                        `json:"endTime"`
	Details   []PMNegativeBalanceExchangeDetail `json:"details"`
}

// PMNegativeBalanceExchangeDetail holds an asset's negative balance in an auto exchange
type PMNegativeBalanceExchangeDetail struct {
	Asset                currency.Code `json:"asset"`
	NegativeBalance      float64       `json:"negativeBalance"`
	NegativeMaxThreshold uint64        `json:"negativeMaxThreshold"`
}

// PMRateLimit holds a portfolio margin order rate limit
type PMRateLimit struct {
	RateLimitType  string `json:"rateLimitType"`
	Interval       string `json:"interval"`
	IntervalNumber uint64 `json:"intervalNum"`
	Limit          uint64 `json:"limit"`
}

// UMNotionalAndLeverage holds a UM symbol's notional and leverage brackets
type UMNotionalAndLeverage struct {
	Symbol string `json:"symbol"`
	// NotionalCoefficient is the user's bracket multiplier, present when it differs from 1
	NotionalCoefficient types.Number        `json:"notionalCoef"`
	Brackets            []UMNotionalBracket `json:"brackets"`
}

// UMNotionalBracket holds a UM leverage bracket, with caps and floors in notional value
type UMNotionalBracket struct {
	Bracket                uint64  `json:"bracket"`
	InitialLeverage        uint64  `json:"initialLeverage"`
	NotionalCap            float64 `json:"notionalCap"`
	NotionalFloor          float64 `json:"notionalFloor"`
	MaintenanceMarginRatio float64 `json:"maintMarginRatio"`
	Cumulative             float64 `json:"cum"` // auxiliary number for quick maintenance margin calculation
}

// CMConditionalOrderRequest holds the parameters for a new CM conditional order
type CMConditionalOrderRequest struct {
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

// CMConditionalOrderResponse holds the fields returned for every CM conditional order
type CMConditionalOrderResponse struct {
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

// NewCMConditionalOrderResponse holds a new CM conditional order
type NewCMConditionalOrderResponse struct {
	CMConditionalOrderResponse
	Pair         string `json:"pair"`
	WorkingType  string `json:"workingType"`
	PriceProtect bool   `json:"priceProtect"`
}

// CancelCMConditionalOrderResponse holds a cancelled CM conditional order
type CancelCMConditionalOrderResponse struct {
	CMConditionalOrderResponse
	WorkingType  string `json:"workingType"`
	PriceProtect bool   `json:"priceProtect"`
}

// CMOrderDetailResponse holds a CM order returned by the order query endpoints
type CMOrderDetailResponse struct {
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

// CMOrderRequest holds the parameters for a new CM order
type CMOrderRequest struct {
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
	PriceMatch       string
	NewClientOrderID string
	NewOrderRespType string
}

// CMOrderResponse holds a CM order returned when it is placed or cancelled
type CMOrderResponse struct {
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

// PMCancelledMarginOrder holds an order or OCO order list cancelled by the cancel all margin open orders endpoint;
// order lists carry the list fields, Orders and OrderReports, and plain orders carry an OrderListID of -1
type PMCancelledMarginOrder struct {
	Symbol                   string                         `json:"symbol"`
	OriginalClientOrderID    string                         `json:"origClientOrderId"`
	OrderID                  uint64                         `json:"orderId"`
	OrderListID              int64                          `json:"orderListId"`
	ClientOrderID            string                         `json:"clientOrderId"`
	Price                    types.Number                   `json:"price"`
	OriginalQuantity         types.Number                   `json:"origQty"`
	ExecutedQuantity         types.Number                   `json:"executedQty"`
	CummulativeQuoteQuantity types.Number                   `json:"cummulativeQuoteQty"`
	Status                   string                         `json:"status"`
	TimeInForce              string                         `json:"timeInForce"`
	Type                     string                         `json:"type"`
	Side                     string                         `json:"side"`
	ContingencyType          string                         `json:"contingencyType"`
	ListStatusType           string                         `json:"listStatusType"`
	ListOrderStatus          string                         `json:"listOrderStatus"`
	ListClientOrderID        string                         `json:"listClientOrderId"`
	TransactionTime          types.Time                     `json:"transactionTime"`
	Orders                   []PMMarginOCOOrderLeg          `json:"orders"`
	OrderReports             []PMCancelledMarginOrderReport `json:"orderReports"`
}

// PMCancelledMarginOrderReport holds a cancelled OCO order leg returned by the cancel all margin open orders endpoint
type PMCancelledMarginOrderReport struct {
	PMCancelMarginOCOOrderReport
	IcebergQuantity types.Number `json:"icebergQty"`
}

// PMMarginOCOOrderLeg holds an order of a margin OCO order list
type PMMarginOCOOrderLeg struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// PMMarginOCOOrderResponse holds a margin OCO order list
type PMMarginOCOOrderResponse struct {
	OrderListID       int64                 `json:"orderListId"`
	ContingencyType   string                `json:"contingencyType"`
	ListStatusType    string                `json:"listStatusType"`
	ListOrderStatus   string                `json:"listOrderStatus"`
	ListClientOrderID string                `json:"listClientOrderId"`
	TransactionTime   types.Time            `json:"transactionTime"`
	Symbol            string                `json:"symbol"`
	Orders            []PMMarginOCOOrderLeg `json:"orders"`
}

// PMCancelMarginOCOOrderRequest holds the parameters for cancelling a margin OCO order list
type PMCancelMarginOCOOrderRequest struct {
	Symbol currency.Pair
	// OrderListID or ListClientOrderID identifies the order list
	OrderListID       uint64
	ListClientOrderID string
	// NewClientOrderID identifies this cancellation
	NewClientOrderID string
}

// PMCancelMarginOCOOrderResponse holds a cancelled margin OCO order list
type PMCancelMarginOCOOrderResponse struct {
	PMMarginOCOOrderResponse
	OrderReports []PMCancelMarginOCOOrderReport `json:"orderReports"`
}

// PMCancelMarginOCOOrderReport holds a cancelled margin OCO order leg
type PMCancelMarginOCOOrderReport struct {
	Symbol                   string       `json:"symbol"`
	OriginalClientOrderID    string       `json:"origClientOrderId"`
	OrderID                  uint64       `json:"orderId"`
	OrderListID              int64        `json:"orderListId"`
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
}

// PMMarginOrderResponse holds a margin order returned by the margin order query endpoints
type PMMarginOrderResponse struct {
	ClientOrderID string `json:"clientOrderId"`
	// CummulativeQuoteQuantity is negative for some historical orders whose data is unavailable
	CummulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
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
	Time                     types.Time   `json:"time"`
	TimeInForce              string       `json:"timeInForce"`
	Type                     string       `json:"type"`
	UpdateTime               types.Time   `json:"updateTime"`
	AccountID                uint64       `json:"accountId"`
	SelfTradePreventionMode  string       `json:"selfTradePreventionMode"`
	// PreventedMatchID and PreventedQuantity are null unless self-trade prevention expired the order
	PreventedMatchID  types.Number `json:"preventedMatchId"`
	PreventedQuantity types.Number `json:"preventedQuantity"`
}

// PMMarginOrderRequest holds the parameters for a new portfolio margin cross margin order
type PMMarginOrderRequest struct {
	Symbol currency.Pair
	Side   order.Side
	// OrderType is LIMIT, MARKET, STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, TAKE_PROFIT_LIMIT or LIMIT_MAKER
	OrderType          string
	Quantity           float64
	QuoteOrderQuantity float64
	Price              float64
	// StopPrice is used with STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT and TAKE_PROFIT_LIMIT orders
	StopPrice        float64
	NewClientOrderID string
	NewOrderRespType string
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

// PMNewMarginOrderResponse holds a new margin order
type PMNewMarginOrderResponse struct {
	Symbol                   string       `json:"symbol"`
	OrderID                  uint64       `json:"orderId"`
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
	// MarginBuyBorrowAmount and MarginBuyBorrowAsset are only returned when the order borrowed
	MarginBuyBorrowAmount types.Number        `json:"marginBuyBorrowAmount"`
	MarginBuyBorrowAsset  currency.Code       `json:"marginBuyBorrowAsset"`
	Fills                 []PMMarginOrderFill `json:"fills"`
}

// PMMarginOrderFill holds a fill of a new margin order
type PMMarginOrderFill struct {
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
}

// PMCancelMarginOrderRequest holds the parameters for cancelling a margin order
type PMCancelMarginOrderRequest struct {
	Symbol currency.Pair
	// OrderID or OriginalClientOrderID identifies the order
	OrderID               uint64
	OriginalClientOrderID string
	// NewClientOrderID identifies this cancellation
	NewClientOrderID string
}

// PMCancelMarginOrderResponse holds a cancelled margin order
type PMCancelMarginOrderResponse struct {
	Symbol                   string       `json:"symbol"`
	OrderID                  uint64       `json:"orderId"`
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
}

// UMOrderDetailResponse holds a UM order returned by the order query endpoints
type UMOrderDetailResponse struct {
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

// UMOrderRequest holds the parameters for a new UM order
type UMOrderRequest struct {
	Symbol currency.Pair
	Side   order.Side
	// PositionSide is BOTH in one-way mode and LONG or SHORT in hedge mode, where it is required
	PositionSide string
	// OrderType is LIMIT or MARKET
	OrderType   string
	TimeInForce string
	Quantity    float64
	// ReduceOnly cannot be sent in hedge mode
	ReduceOnly       bool
	Price            float64
	NewClientOrderID string
	NewOrderRespType string
	// PriceMatch cannot be sent with Price
	PriceMatch string
	// SelfTradePreventionMode is only effective with the IOC, GTC and GTD time in force values
	SelfTradePreventionMode string
	// GoodTillDate is required for GTD orders; only its seconds are used
	GoodTillDate time.Time
}

// UMOrderResponse holds a UM order returned when it is placed or cancelled
type UMOrderResponse struct {
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

// CMAccountTradeListRequest holds the parameters for a CM account trade list query; Symbol and Pair are mutually
// exclusive, and FromID cannot be combined with Pair or a time range
type CMAccountTradeListRequest struct {
	Symbol currency.Pair
	// Pair, such as BTCUSD, is the base of a COIN-M contract pair and returns trades of every symbol of the pair
	Pair      currency.Code
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64
	Limit     uint64
}

// CMAccountTrade holds a CM account trade
type CMAccountTrade struct {
	Symbol          string        `json:"symbol"`
	ID              uint64        `json:"id"`
	OrderID         uint64        `json:"orderId"`
	Pair            string        `json:"pair"`
	Side            string        `json:"side"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	RealizedPNL     types.Number  `json:"realizedPnl"`
	MarginAsset     currency.Code `json:"marginAsset"`
	BaseQuantity    types.Number  `json:"baseQty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Time            types.Time    `json:"time"`
	PositionSide    string        `json:"positionSide"`
	Buyer           bool          `json:"buyer"`
	Maker           bool          `json:"maker"`
}

// CMADLQuantileEstimation holds a CM symbol's auto-deleveraging quantiles
type CMADLQuantileEstimation struct {
	Symbol      string        `json:"symbol"`
	ADLQuantile CMADLQuantile `json:"adlQuantile"`
}

// CMADLQuantile holds auto-deleveraging queue quantiles from 0 (low) to 4 (high) per position side; a cross margined
// hedge mode position reports HEDGE instead of BOTH
type CMADLQuantile struct {
	Long  uint64 `json:"LONG"`
	Short uint64 `json:"SHORT"`
	Hedge uint64 `json:"HEDGE"` // only a sign, its value is meaningless
	Both  uint64 `json:"BOTH"`
}

// PMMarginOCOOrderRequest holds the parameters for a new margin OCO order list
type PMMarginOCOOrderRequest struct {
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
	NewOrderRespType     string
	// SideEffectType is NO_SIDE_EFFECT (default), MARGIN_BUY or AUTO_REPAY
	SideEffectType string
}

// PMNewMarginOCOOrderResponse holds a new margin OCO order list
type PMNewMarginOCOOrderResponse struct {
	PMMarginOCOOrderResponse
	// MarginBuyBorrowAmount and MarginBuyBorrowAsset are only returned when the order list borrowed
	MarginBuyBorrowAmount types.Number                `json:"marginBuyBorrowAmount"`
	MarginBuyBorrowAsset  currency.Code               `json:"marginBuyBorrowAsset"`
	OrderReports          []PMNewMarginOCOOrderReport `json:"orderReports"`
}

// PMNewMarginOCOOrderReport holds an order of a new margin OCO order list
type PMNewMarginOCOOrderReport struct {
	Symbol                   string       `json:"symbol"`
	OrderID                  uint64       `json:"orderId"`
	OrderListID              int64        `json:"orderListId"`
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
}

// PMMarginTradeListRequest holds the parameters for a margin account trade list query
type PMMarginTradeListRequest struct {
	Symbol    currency.Pair
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64
	Limit     uint64
}

// PMMarginTrade holds a margin account trade
type PMMarginTrade struct {
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

// CMConditionalOrdersRequest holds the parameters for a query of all CM conditional orders
type CMConditionalOrdersRequest struct {
	Symbol     currency.Pair
	StrategyID uint64
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint64
}

// CMConditionalOrderRecord holds a CM conditional order with the order it triggered, if any
type CMConditionalOrderRecord struct {
	CMConditionalOrderResponse
	// OrderID, Status and Type describe the order placed when the strategy triggered
	OrderID     uint64     `json:"orderId"`
	Status      string     `json:"status"`
	TriggerTime types.Time `json:"triggerTime"`
	Type        string     `json:"type"`
}

// CMOrdersRequest holds the parameters for a query of all CM orders; Symbol or Pair is required
type CMOrdersRequest struct {
	Symbol currency.Pair
	// Pair, such as BTCUSD, is the base of a COIN-M contract pair
	Pair currency.Code
	// OrderID returns orders from this order ID onwards
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// PMMarginOrdersRequest holds the parameters for a query of all margin account orders
type PMMarginOrdersRequest struct {
	Symbol currency.Pair
	// OrderID returns orders from this order ID onwards
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// UMOrdersRequest holds the parameters for a query of all UM orders
type UMOrdersRequest struct {
	Symbol currency.Pair
	// OrderID returns orders from this order ID onwards
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// CMConditionalOrderHistoryResponse holds a CM conditional order from the order history
type CMConditionalOrderHistoryResponse struct {
	CMConditionalOrderRecord
	WorkingType  string `json:"workingType"`
	PriceProtect bool   `json:"priceProtect"`
	PriceMatch   string `json:"priceMatch"`
}

// PMMarginAllOCORequest holds the parameters for a query of all margin OCO order lists
type PMMarginAllOCORequest struct {
	FromID    uint64
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64
}

// PMForceOrdersRequest holds the parameters for a UM or CM force orders query
type PMForceOrdersRequest struct {
	Symbol currency.Pair
	// AutoCloseType is LIQUIDATION or ADL; both are returned when it is empty
	AutoCloseType string
	StartTime     time.Time
	EndTime       time.Time
	Limit         uint64
}

// CMForceOrder holds a CM liquidation or auto-deleveraging order
type CMForceOrder struct {
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

// PMMarginForceOrdersRequest holds the parameters for a margin force orders query
type PMMarginForceOrdersRequest struct {
	StartTime time.Time
	EndTime   time.Time
	Current   uint64
	Size      uint64
}

// PMMarginForceOrderResponse holds a page of margin force orders
type PMMarginForceOrderResponse struct {
	Rows  []PMMarginForceOrder `json:"rows"`
	Total uint64               `json:"total"`
}

// PMMarginForceOrder holds a margin liquidation order
type PMMarginForceOrder struct {
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

// UMForceOrder holds a UM liquidation or auto-deleveraging order
type UMForceOrder struct {
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

// UMAccountTradeListRequest holds the parameters for a UM account trade list query; FromID cannot be combined with
// a time range
type UMAccountTradeListRequest struct {
	Symbol    currency.Pair
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64
	Limit     uint64
}

// UMAccountTrade holds a UM account trade
type UMAccountTrade struct {
	Symbol          string        `json:"symbol"`
	ID              uint64        `json:"id"`
	OrderID         uint64        `json:"orderId"`
	Side            string        `json:"side"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	RealizedPNL     types.Number  `json:"realizedPnl"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Time            types.Time    `json:"time"`
	Buyer           bool          `json:"buyer"`
	Maker           bool          `json:"maker"`
	PositionSide    string        `json:"positionSide"`
}

// UMADLQuantileEstimation holds a UM symbol's auto-deleveraging quantiles
type UMADLQuantileEstimation struct {
	Symbol      string        `json:"symbol"`
	ADLQuantile UMADLQuantile `json:"adlQuantile"`
}

// UMADLQuantile holds auto-deleveraging queue quantiles from 0 (low) to 4 (high) per position side
type UMADLQuantile struct {
	Long  uint64 `json:"LONG"`
	Short uint64 `json:"SHORT"`
	Both  uint64 `json:"BOTH"`
}

// UMAlgoOrderRequest holds the parameters for a new UM algo (conditional) order
type UMAlgoOrderRequest struct {
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
	NewOrderRespType        string
	SelfTradePreventionMode string
	// GoodTillDate is required for GTD orders
	GoodTillDate time.Time
}

// UMAlgoOrderResponse holds a new UM algo order
type UMAlgoOrderResponse struct {
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

// CancelUMAlgoOrderResponse holds the result of a UM algo order cancellation
type CancelUMAlgoOrderResponse struct {
	Complete bool `json:"complete"`
}

// UMAlgoOrder holds an open UM algo order
type UMAlgoOrder struct {
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
	CreateTime              types.Time   `json:"createTime"`
	UpdateTime              types.Time   `json:"updateTime"`
	TriggerTime             types.Time   `json:"triggerTime"`
	GoodTillDate            types.Time   `json:"goodTillDate"`
}

// UMAlgoOrderDetailResponse holds a UM algo order as the order and history queries return it, with the order its
// trigger placed
type UMAlgoOrderDetailResponse struct {
	UMAlgoOrder
	ActualOrderID string       `json:"actualOrderId"` // empty until the algo order triggers
	ActualPrice   types.Number `json:"actualPrice"`
}

// UMAlgoOrderHistoryRequest holds the parameters of Query UM Algo Order History
type UMAlgoOrderHistoryRequest struct {
	Symbol    currency.Pair
	AlgoID    uint64 // returns orders from this algo ID on
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 500 when zero, at most 1000
}
