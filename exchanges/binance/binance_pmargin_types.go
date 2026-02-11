package binance

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UMCMOrder represents a portfolio margin USDT Margined or Coin Margined order.
type UMCMOrder struct {
	OrderID       int64        `json:"orderId"`
	ClientOrderID string       `json:"clientOrderId"`
	CumQty        types.Number `json:"cumQty"`
	ExecutedQty   types.Number `json:"executedQty"`
	AvgPrice      types.Number `json:"avgPrice"`
	OrigQty       types.Number `json:"origQty"`
	Price         types.Number `json:"price"`
	ReduceOnly    bool         `json:"reduceOnly"`
	Side          string       `json:"side"`
	PositionSide  string       `json:"positionSide"`
	Status        string       `json:"status"`
	Symbol        string       `json:"symbol"`
	TimeInForce   string       `json:"timeInForce"`
	Type          string       `json:"type"`
	UpdateTime    types.Time   `json:"updateTime"`

	// Used By USDT Margined Futures only
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	GoodTillDate            types.Time   `json:"goodTillDate"`
	CumQuote                types.Number `json:"cumQuote"`

	// Used By Coin Margined Futures only
	Pair    string `json:"pair"`
	CumBase string `json:"cumBase"`
}

// UMOrderParam request parameters for UM order
type UMOrderParam struct {
	Symbol                  currency.Pair `json:"symbol"`
	Side                    string        `json:"side"`
	PositionSide            string        `json:"positionSide,omitempty"`
	OrderType               string        `json:"type"`
	TimeInForce             string        `json:"timeInForce,omitempty"`
	Quantity                float64       `json:"quantity,omitempty"`
	ReduceOnly              bool          `json:"reduceOnly,omitempty"`
	Price                   float64       `json:"price,omitempty"`
	NewClientOrderID        string        `json:"newClientOrderId,omitempty"`
	NewOrderRespType        string        `json:"newOrderRespType,omitempty"`
	SelfTradePreventionMode string        `json:"selfTradePreventionMode,omitempty"`
	GoodTillDate            int64         `json:"goodTillDate,omitempty"`
}

// MarginOrderParam represents request parameter for margin trade order
type MarginOrderParam struct {
	Symbol                  string  `json:"symbol"`
	Side                    string  `json:"side"`
	OrderType               string  `json:"type"`
	Amount                  float64 `json:"quantity,omitempty"`
	QuoteOrderQty           float64 `json:"quoteOrderQty,omitempty"`
	Price                   float64 `json:"price,omitempty"`
	StopPrice               float64 `json:"stopPrice,omitempty"` // Used with STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, and TAKE_PROFIT_LIMIT orders.
	NewClientOrderID        string  `json:"newClientOrderId,omitempty"`
	NewOrderRespType        string  `json:"newOrderRespType,omitempty"`
	IcebergQuantity         float64 `json:"icebergQty,omitempty"`
	SideEffectType          string  `json:"sideEffectType,omitempty"`
	TimeInForce             string  `json:"timeInForce,omitempty"`
	SelfTradePreventionMode string  `json:"selfTradePreventionMode,omitempty"`
}

// MarginOrderResp represents a margin order response.
type MarginOrderResp struct {
	Symbol                  string            `json:"symbol"`
	OrderID                 int64             `json:"orderId"`
	ClientOrderID           string            `json:"clientOrderId"`
	OrigClientOrderID       string            `json:"origClientOrderId"`
	TransactTime            types.Time        `json:"transactTime"`
	Price                   types.Number      `json:"price"`
	SelfTradePreventionMode string            `json:"selfTradePreventionMode"`
	OrigQty                 types.Number      `json:"origQty"`
	ExecutedQty             types.Number      `json:"executedQty"`
	CummulativeQuoteQty     types.Number      `json:"cummulativeQuoteQty"`
	Status                  string            `json:"status"`
	TimeInForce             order.TimeInForce `json:"timeInForce"`
	Type                    string            `json:"type"`
	Side                    string            `json:"side"`
	MarginBuyBorrowAmount   float64           `json:"marginBuyBorrowAmount"`
	MarginBuyBorrowAsset    string            `json:"marginBuyBorrowAsset"`
	Fills                   []struct {
		Price           types.Number `json:"price"`
		Qty             types.Number `json:"qty"`
		Commission      types.Number `json:"commission"`
		CommissionAsset string       `json:"commissionAsset"`
	} `json:"fills"`
}

// MarginAccOrdersList represents a list of margin account order details.
type MarginAccOrdersList []struct {
	Symbol              string            `json:"symbol"`
	OrigClientOrderID   string            `json:"origClientOrderId,omitempty"`
	OrderID             int64             `json:"orderId,omitempty"`
	OrderListID         int64             `json:"orderListId"`
	ClientOrderID       string            `json:"clientOrderId,omitempty"`
	Price               types.Number      `json:"price,omitempty"`
	OrigQty             types.Number      `json:"origQty,omitempty"`
	ExecutedQty         types.Number      `json:"executedQty,omitempty"`
	CummulativeQuoteQty types.Number      `json:"cummulativeQuoteQty,omitempty"`
	Status              string            `json:"status,omitempty"`
	TimeInForce         order.TimeInForce `json:"timeInForce,omitempty"`
	Type                string            `json:"type,omitempty"`
	Side                string            `json:"side,omitempty"`
	ContingencyType     string            `json:"contingencyType,omitempty"`
	ListStatusType      string            `json:"listStatusType,omitempty"`
	ListOrderStatus     string            `json:"listOrderStatus,omitempty"`
	ListClientOrderID   string            `json:"listClientOrderId,omitempty"`
	TransactionTime     types.Time        `json:"transactionTime,omitempty"`
	Orders              []struct {
		Symbol        string `json:"symbol"`
		OrderID       int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
	} `json:"orders,omitempty"`
	OrderReports []OrderResponse `json:"orderReports,omitempty"`
}

// ConditionalOrder represents a USDT/Coin margined conditional order instance.
type ConditionalOrder struct {
	NewClientStrategyID string            `json:"newClientStrategyId"`
	StrategyID          int               `json:"strategyId"`
	StrategyStatus      string            `json:"strategyStatus"`
	StrategyType        string            `json:"strategyType"`
	OrigQty             types.Number      `json:"origQty"`
	Price               types.Number      `json:"price"`
	ReduceOnly          bool              `json:"reduceOnly"`
	Side                string            `json:"side"`
	PositionSide        string            `json:"positionSide"`
	StopPrice           types.Number      `json:"stopPrice"`
	Symbol              string            `json:"symbol"`
	TimeInForce         order.TimeInForce `json:"timeInForce"`
	ActivatePrice       types.Number      `json:"activatePrice"` // activation price, only return with TRAILING_STOP_MARKET order
	PriceRate           types.Number      `json:"priceRate"`     // callback rate, only return with TRAILING_STOP_MARKET order
	BookTime            types.Time        `json:"bookTime"`      // order place time
	UpdateTime          types.Time        `json:"updateTime"`
	WorkingType         string            `json:"workingType"`
	PriceProtect        bool              `json:"priceProtect"`

	// Returned for USDT Margined Futures orders only
	SelfTradePreventionMode string     `json:"selfTradePreventionMode"`
	GoodTillDate            types.Time `json:"goodTillDate"` // order pre-set auto cancel time for TIF GTD order

	Pair string `json:"pair"`
}

// ConditionalOrderParam represents a conditional order parameter for coin/usdt margined futures.
type ConditionalOrderParam struct {
	Symbol              currency.Pair `json:"symbol"`
	Side                string        `json:"side"`
	PositionSide        string        `json:"positionSide,omitempty"` // Default BOTH for One-way Mode ; LONG or SHORT for Hedge Mode. It must be sent in Hedge Mode.
	StrategyType        string        `json:"strategyType"`           // "STOP", "STOP_MARKET", "TAKE_PROFIT", "TAKE_PROFIT_MARKET", and "TRAILING_STOP_MARKET"
	TimeInForce         string        `json:"timeInForce,omitempty"`
	Quantity            float64       `json:"quantity,omitempty"`
	ReduceOnly          bool          `json:"reduceOnly,omitempty"`
	Price               float64       `json:"price,omitempty"`
	WorkingType         string        `json:"workingType,omitempty"`
	PriceProtect        bool          `json:"priceProtect,omitempty"`
	NewClientStrategyID string        `json:"newClientStrategyID,omitempty"`
	StopPrice           float64       `json:"stopPrice,omitempty"`
	ActivationPrice     float64       `json:"activationPrice,omitempty"`
	CallbackRate        float64       `json:"callbackRate,omitempty"`

	// User in USDT margined futures only
	SelfTradePreventionMode string `json:"selfTradePreventionMode,omitempty"`
	GoodTillDate            int64  `json:"goodTillDate,omitempty"`
}

// SuccessResponse represents a success code and message; used when cancelling orders in portfolio margin endpoints.
type SuccessResponse struct {
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}

// MarginOrder represents a margin account order
type MarginOrder struct {
	ClientOrderID           string       `json:"clientOrderId"`
	CummulativeQuoteQty     types.Number `json:"cummulativeQuoteQty"`
	ExecutedQty             types.Number `json:"executedQty"`
	IcebergQty              types.Number `json:"icebergQty"`
	IsWorking               bool         `json:"isWorking"`
	OrderID                 int          `json:"orderId"`
	OrigQty                 types.Number `json:"origQty"`
	Price                   types.Number `json:"price"`
	Side                    string       `json:"side"`
	Status                  string       `json:"status"`
	StopPrice               types.Number `json:"stopPrice"`
	Symbol                  string       `json:"symbol"`
	Time                    types.Time   `json:"time"`
	TimeInForce             string       `json:"timeInForce"`
	Type                    string       `json:"type"`
	UpdateTime              types.Time   `json:"updateTime"`
	AccountID               int64        `json:"accountId"`
	SelfTradePreventionMode string       `json:"selfTradePreventionMode"`
	PreventedMatchID        any          `json:"preventedMatchId"`
	PreventedQuantity       any          `json:"preventedQuantity"`
}

// AccountBalance represents an account balance information for an asset from all margin and futures accounts.
type AccountBalance struct {
	Asset               string       `json:"asset"`
	TotalWalletBalance  types.Number `json:"totalWalletBalance"`  // wallet balance =  cross margin free + cross margin locked + UM wallet balance + CM wallet balance
	CrossMarginAsset    types.Number `json:"crossMarginAsset"`    // crossMarginAsset = crossMarginFree + crossMarginLocked
	CrossMarginBorrowed types.Number `json:"crossMarginBorrowed"` // principal of cross margin
	CrossMarginFree     types.Number `json:"crossMarginFree"`     // free asset of cross margin
	CrossMarginInterest types.Number `json:"crossMarginInterest"` // interest of cross margin
	CrossMarginLocked   types.Number `json:"crossMarginLocked"`   // lock asset of cross margin
	UmWalletBalance     types.Number `json:"umWalletBalance"`     // wallet balance of um
	UmUnrealizedPNL     types.Number `json:"umUnrealizedPNL"`     // unrealized profit of um
	CmWalletBalance     types.Number `json:"cmWalletBalance"`     // wallet balance of cm
	CmUnrealizedPNL     string       `json:"cmUnrealizedPNL"`     // unrealized profit of cm
	UpdateTime          types.Time   `json:"updateTime"`
	NegativeBalance     types.Number `json:"negativeBalance"`
}

// AccountBalanceResponse takes an instance object or slice of instances of AccountBalance as a slice.
type AccountBalanceResponse []AccountBalance

// AccountInformation represents a portfolio margin account information.
type AccountInformation struct {
	UniMMR                   string       `json:"uniMMR"`        // Portfolio margin account maintenance margin rate
	AccountEquity            types.Number `json:"accountEquity"` // Account equity, in USD value
	ActualEquity             types.Number `json:"actualEquity"`  // Account equity calculated without discount on collateral rate, in USD value
	AccountInitialMargin     types.Number `json:"accountInitialMargin"`
	AccountMaintMargin       types.Number `json:"accountMaintMargin"`       // Portfolio margin account maintenance margin, unit：USD
	AccountStatus            string       `json:"accountStatus"`            // Portfolio margin account status:"NORMAL", "MARGIN_CALL", "SUPPLY_MARGIN", "REDUCE_ONLY", "ACTIVE_LIQUIDATION", "FORCE_LIQUIDATION", "BANKRUPTED"
	VirtualMaxWithdrawAmount types.Number `json:"virtualMaxWithdrawAmount"` // Portfolio margin maximum amount for transfer out in USD
	TotalAvailableBalance    string       `json:"totalAvailableBalance"`
	TotalMarginOpenLoss      string       `json:"totalMarginOpenLoss"` // in USD margin open order
	UpdateTime               types.Time   `json:"updateTime"`          // last update time
}

// MaxBorrow represents borrowable amount information.
type MaxBorrow struct {
	Amount                  float64 `json:"amount"`      // account's currently max borrowable amount with sufficient system availability
	AccountLevelBorrowLimit float64 `json:"borrowLimit"` // max borrowable amount limited by the account level
}

// UMPositionInformation represents a UM position information.
type UMPositionInformation struct {
	EntryPrice       types.Number `json:"entryPrice"`
	Leverage         types.Number `json:"leverage"`
	MarkPrice        types.Number `json:"markPrice"`
	MaxNotionalValue types.Number `json:"maxNotionalValue"`
	PositionAmt      types.Number `json:"positionAmt"`
	Notional         types.Number `json:"notional"`
	Symbol           string       `json:"symbol"`
	UnRealizedProfit types.Number `json:"unRealizedProfit"`
	LiquidationPrice types.Number `json:"liquidationPrice"`
	PositionSide     string       `json:"positionSide"`
	UpdateTime       types.Time   `json:"updateTime"`
}

// CMPositionInformation represents a Coin Margined Futures position information.
type CMPositionInformation []struct {
	Symbol           string       `json:"symbol"`
	PositionAmt      types.Number `json:"positionAmt"`
	EntryPrice       types.Number `json:"entryPrice"`
	MarkPrice        types.Number `json:"markPrice"`
	LiquidationPrice types.Number `json:"liquidationPrice"`
	UnRealizedProfit string       `json:"unRealizedProfit"`
	Leverage         string       `json:"leverage"`
	PositionSide     string       `json:"positionSide"`
	UpdateTime       types.Time   `json:"updateTime"`
	MaxQty           types.Number `json:"maxQty"`
	NotionalValue    types.Number `json:"notionalValue"`
	BreakEvenPrice   types.Number `json:"breakEvenPrice"`
}

// InitialLeverage represents a leverage information for USDT Margined symbol.
type InitialLeverage struct {
	Leverage         int          `json:"leverage"`
	MaxNotionalValue types.Number `json:"maxNotionalValue"`
	Symbol           string       `json:"symbol"`
}

// CMInitialLeverage represents a leverage information for Coin Margined symbol
type CMInitialLeverage struct {
	Leverage    int          `json:"leverage"`
	MaxQuantity types.Number `json:"maxQty"`
	Symbol      string       `json:"symbol"`
}

// DualPositionMode represents a user's position mode
type DualPositionMode struct {
	DualPositionMode bool `json:"dualSidePosition"` // "true": Hedge Mode; "false": One-way Mode
}

// UMCMAccountTradeItem represents an account trade list
type UMCMAccountTradeItem struct {
	Symbol          string       `json:"symbol"`
	ID              int64        `json:"id"`
	OrderID         int64        `json:"orderId"`
	Side            types.Number `json:"side"`
	Price           types.Number `json:"price"`
	Qty             types.Number `json:"qty"`
	RealizedPnl     types.Number `json:"realizedPnl"`
	MarginAsset     string       `json:"marginAsset"`
	QuoteQty        types.Number `json:"quoteQty"`
	Commission      types.Number `json:"commission"`
	CommissionAsset string       `json:"commissionAsset"`
	Time            types.Time   `json:"time"`
	Buyer           bool         `json:"buyer"`
	Maker           bool         `json:"maker"`
	PositionSide    string       `json:"positionSide"`

	// used with the CM trade info
	Pair    types.Number `json:"pair"`
	BaseQty types.Number `json:"baseQty"`
}

// NotionalAndLeverage represents notional and leverage brackets
type NotionalAndLeverage struct {
	Symbol       string `json:"symbol"`
	NotionalCoef string `json:"notionalCoef"`
	Brackets     []struct {
		Bracket          float64 `json:"bracket"`
		InitialLeverage  float64 `json:"initialLeverage"`
		NotionalCap      float64 `json:"notionalCap"`
		NotionalFloor    float64 `json:"notionalFloor"`
		MaintMarginRatio float64 `json:"maintMarginRatio"`
		Cum              float64 `json:"cum"`
	} `json:"brackets"`
}

// CMNotionalAndLeverage represents notional and leverage brackets for Coin Margined Futures.
type CMNotionalAndLeverage struct {
	Symbol       string `json:"symbol"`
	NotionalCoef string `json:"notionalCoef"`
	Brackets     []struct {
		Bracket          float64 `json:"bracket"`
		InitialLeverage  float64 `json:"initialLeverage"`
		QuantityCap      float64 `json:"qtyCap"`
		QuantityFloor    float64 `json:"qtyFloor"`
		MaintMarginRatio float64 `json:"maintMarginRatio"`
		Cum              float64 `json:"cum"`
	} `json:"brackets"`
}

// MarginForceOrder user's margin force order
type MarginForceOrder struct {
	Rows []struct {
		OrderID     int64        `json:"orderId"`
		AvgPrice    types.Number `json:"avgPrice"`
		ExecutedQty types.Number `json:"executedQty"`
		Price       types.Number `json:"price"`
		Qty         types.Number `json:"qty"`
		Side        string       `json:"side"`
		Symbol      string       `json:"symbol"`
		TimeInForce string       `json:"timeInForce"`
		UpdatedTime types.Time   `json:"updatedTime"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// ForceOrder represents a USDT Margined force order instance.
type ForceOrder struct {
	OrderID       int64        `json:"orderId"`
	Symbol        string       `json:"symbol"`
	Status        string       `json:"status"`
	ClientOrderID string       `json:"clientOrderId"`
	Price         types.Number `json:"price"`
	AvgPrice      types.Number `json:"avgPrice"`
	OrigQty       types.Number `json:"origQty"`
	ExecutedQty   types.Number `json:"executedQty"`
	TimeInForce   string       `json:"timeInForce"`
	Type          string       `json:"type"`
	ReduceOnly    bool         `json:"reduceOnly"`
	Side          string       `json:"side"`
	PositionSide  string       `json:"positionSide"`
	OrigType      string       `json:"origType"`
	Time          types.Time   `json:"time"`
	UpdateTime    types.Time   `json:"updateTime"`

	// used by usdt margined futures
	CumQuote types.Number `json:"cumQuote"`

	// used by coin margined futures
	Pair    string       `json:"pair"`
	CumBase types.Number `json:"cumBase"`
}

// CommissionRate represents a user's commission rate
type CommissionRate struct {
	Symbol              string       `json:"symbol"`
	MakerCommissionRate types.Number `json:"makerCommissionRate"`
	TakerCommissionRate types.Number `json:"takerCommissionRate"`
}

// MarginLoanRecord represents a margin loan record.
type MarginLoanRecord struct {
	Rows []struct {
		TransactionID int64      `json:"txId"`
		Asset         string     `json:"asset"`
		Principal     string     `json:"principal"`
		Timestamp     types.Time `json:"timestamp"`
		Status        string     `json:"status"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// MarginRepayRecord represents a margin repay record.
type MarginRepayRecord struct {
	Rows []struct {
		Amount        types.Number `json:"amount"`
		Asset         string       `json:"asset"`
		Interest      types.Number `json:"interest"`
		Principal     types.Number `json:"principal"`
		Status        string       `json:"status"`
		Timestamp     types.Time   `json:"timestamp"`
		TransactionID int64        `json:"txId"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// MarginBorrowOrLoanInterest represents margin borrow/loan interest history
type MarginBorrowOrLoanInterest struct {
	Rows []struct {
		TransactionID       int64        `json:"txId"`
		InterestAccuredTime types.Time   `json:"interestAccuredTime"`
		Asset               string       `json:"asset"`
		RawAsset            string       `json:"rawAsset"`
		Principal           string       `json:"principal"`
		Interest            types.Number `json:"interest"`
		InterestRate        types.Number `json:"interestRate"`
		Type                string       `json:"type"`
	} `json:"rows"`
	Total int64 `json:"total"`
}

// PortfolioMarginNegativeBalanceInterest represents interest history of negative balance.
type PortfolioMarginNegativeBalanceInterest struct {
	Asset               string       `json:"asset"`
	Interest            string       `json:"interest"`
	InterestAccuredTime types.Time   `json:"interestAccuredTime"`
	InterestRate        types.Number `json:"interestRate"`
	Principal           string       `json:"principal"`
}

// IncomeItem represents a USDT margined income item.
type IncomeItem struct {
	Symbol       string       `json:"symbol"`
	IncomeType   string       `json:"incomeType"`
	IncomeAmount types.Number `json:"income"`
	IncomeAsset  string       `json:"asset"`
	ExtraInfo    string       `json:"info"`
	Time         types.Time   `json:"time"`
	TranferID    string       `json:"tranId"`
	TradeID      string       `json:"tradeId"`
}

// AccountDetail represents account asset and position information.
type AccountDetail struct {
	TradeGroupID int `json:"tradeGroupId"`
	Assets       []struct {
		Asset                  string       `json:"asset"`
		CrossWalletBalance     types.Number `json:"crossWalletBalance"`
		CrossUnPnl             types.Number `json:"crossUnPnl"`
		MaintMargin            types.Number `json:"maintMargin"`
		InitialMargin          types.Number `json:"initialMargin"`
		PositionInitialMargin  types.Number `json:"positionInitialMargin"`
		OpenOrderInitialMargin types.Number `json:"openOrderInitialMargin"`
		UpdateTime             types.Time   `json:"updateTime"`
	} `json:"assets"`
	Positions []struct {
		Symbol                 string       `json:"symbol"`
		InitialMargin          types.Number `json:"initialMargin"`
		MaintMargin            types.Number `json:"maintMargin"`
		UnrealizedProfit       types.Number `json:"unrealizedProfit"`
		PositionInitialMargin  types.Number `json:"positionInitialMargin"`
		OpenOrderInitialMargin types.Number `json:"openOrderInitialMargin"`
		Leverage               types.Number `json:"leverage"`
		EntryPrice             types.Number `json:"entryPrice"`
		PositionSide           string       `json:"positionSide"`
		PositionAmt            types.Number `json:"positionAmt"`
		UpdateTime             types.Time   `json:"updateTime"`
		BreakEvenPrice         types.Number `json:"breakEvenPrice"`

		// Used USDT Margined Futures
		MaxNotional string `json:"maxNotional"`
		BidNotional string `json:"bidNotional"`
		AskNotional string `json:"askNotional"`

		// Used Coin Margined Futures
		MaxQty types.Number `json:"maxQty"`
	} `json:"positions"`
}

// AutoRepayStatus represents an auto-repay status
type AutoRepayStatus struct {
	AutoRepay bool `json:"autoRepay"` // "true" for turn on the auto-repay futures; "false" for turn off the auto-repay futures
}

// ADLQuantileEstimation represents an ADL quantile estimation instance.
type ADLQuantileEstimation struct {
	Symbol string `json:"symbol"`
	// if the positions of the symbol are crossed margined in Hedge Mode, "LONG" and "SHORT" will be returned a same quantile value, and "HEDGE" will be returned instead of "BOTH".
	ADLQuantile struct {
		Long  float64 `json:"LONG"`  // adl quantile for "LONG" position in hedge mode
		Short float64 `json:"SHORT"` // adl qauntile for "SHORT" position in hedge mode
		Hedge float64 `json:"HEDGE"` // only a sign, ignore the value
	} `json:"adlQuantile,omitempty"` // adl qunatile for position in one-way mode
}

// PortfolioMarginAssetIndexPrice holds a portfolio margin asset index price in usd
type PortfolioMarginAssetIndexPrice struct {
	Asset                string       `json:"asset"`
	AssetIndexPriceInUSD types.Number `json:"assetIndexPrice"`
	Time                 types.Time   `json:"time"`
}
