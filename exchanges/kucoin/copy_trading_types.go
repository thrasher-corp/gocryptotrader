package kucoin

import (
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CopyTradingOrderRequest holds the parameters of a lead trader's copy trading futures order, shared by Add Order and
// Add Order Test
type CopyTradingOrderRequest struct {
	// ClientOrderID is a unique ID of up to 40 letters, digits, underscores and hyphens
	ClientOrderID string
	// Side may be left empty when CloseOrder is set; KuCoin then works it out from the position
	Side   string
	Symbol string
	// Leverage sets the margin frozen for the order. Add Order may leave it at zero; Add Order Test needs it unless
	// CloseOrder is set
	Leverage uint64
	Type     string
	// Stop is down or up; a stop order needs StopPriceType and StopPrice as well
	Stop string
	// StopPriceType is TP for the last trade price or MP for the mark price
	StopPriceType string
	StopPrice     float64
	ReduceOnly    bool
	// CloseOrder closes the whole position; Side, Size and Leverage may then be left empty
	CloseOrder bool
	MarginMode string
	// Price is required for limit orders
	Price float64
	// Size is a number of lots; it may be left at zero when CloseOrder is set
	Size         uint64
	TimeInForce  string
	PostOnly     bool
	PositionSide string
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// Payload validates the request and returns the order body Add Order and Add Order Test share
func (r *CopyTradingOrderRequest) Payload() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.ClientOrderID == "":
		return nil, order.ErrClientOrderIDMustBeSet
	case r.Symbol == "":
		return nil, currency.ErrSymbolStringEmpty
	case r.Type == "":
		return nil, order.ErrTypeIsInvalid
	case r.Side == "" && !r.CloseOrder:
		return nil, order.ErrSideIsInvalid
	case r.Size == 0 && !r.CloseOrder:
		return nil, order.ErrAmountIsInvalid
	case r.Type == "limit" && r.Price <= 0:
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	case r.Stop != "" && r.StopPriceType == "":
		return nil, errInvalidStopPriceType
	case r.Stop != "" && r.StopPrice <= 0:
		return nil, errStopPriceRequired
	}
	body := make(map[string]any)
	body["clientOid"] = r.ClientOrderID
	body["symbol"] = r.Symbol
	body["type"] = r.Type
	if r.Side != "" {
		body["side"] = r.Side
	}
	if r.Leverage != 0 {
		body["leverage"] = r.Leverage
	}
	if r.Stop != "" {
		body["stop"] = r.Stop
	}
	if r.StopPriceType != "" {
		body["stopPriceType"] = r.StopPriceType
	}
	if r.StopPrice != 0 {
		body["stopPrice"] = strconv.FormatFloat(r.StopPrice, 'f', -1, 64)
	}
	if r.ReduceOnly {
		body["reduceOnly"] = true
	}
	if r.CloseOrder {
		body["closeOrder"] = true
	}
	if r.MarginMode != "" {
		body["marginMode"] = r.MarginMode
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.Size != 0 {
		body["size"] = r.Size
	}
	if r.TimeInForce != "" {
		body["timeInForce"] = r.TimeInForce
	}
	if r.PostOnly {
		body["postOnly"] = true
	}
	if r.PositionSide != "" {
		body["positionSide"] = r.PositionSide
	}
	if r.FundsType != "" {
		body["fundsType"] = r.FundsType
	}
	return body, nil
}

// CopyTradingOrderResponse holds the IDs of a copy trading futures order KuCoin accepted
type CopyTradingOrderResponse struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOid"`
}

// CopyTradingTPSLOrderRequest holds the parameters of Add Take Profit And Stop Loss Order, a copy trading futures
// order that carries its own take profit and stop loss triggers
type CopyTradingTPSLOrderRequest struct {
	// ClientOrderID is a unique ID of up to 40 letters, digits, underscores and hyphens
	ClientOrderID string
	// Side may be left empty when CloseOrder is set; KuCoin then works it out from the position
	Side   string
	Symbol string
	// Leverage may be left at zero when CloseOrder is set
	Leverage uint64
	Type     string
	// StopPriceType is TP for the last trade price or MP for the mark price
	StopPriceType string
	ReduceOnly    bool
	// CloseOrder closes the whole position; Side, Size and Leverage may then be left empty
	CloseOrder bool
	MarginMode string
	// Price is required for limit orders
	Price float64
	// Size is a number of lots; it may be left at zero when CloseOrder is set
	Size        uint64
	TimeInForce string
	PostOnly    bool
	// TriggerStopUpPrice is the take profit trigger price
	TriggerStopUpPrice float64
	// TriggerStopDownPrice is the stop loss trigger price
	TriggerStopDownPrice float64
	PositionSide         string
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// Payload validates the request and returns the Add Take Profit And Stop Loss Order body
func (r *CopyTradingTPSLOrderRequest) Payload() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.ClientOrderID == "":
		return nil, order.ErrClientOrderIDMustBeSet
	case r.Symbol == "":
		return nil, currency.ErrSymbolStringEmpty
	case r.Type == "":
		return nil, order.ErrTypeIsInvalid
	case r.Side == "" && !r.CloseOrder:
		return nil, order.ErrSideIsInvalid
	case r.Size == 0 && !r.CloseOrder:
		return nil, order.ErrAmountIsInvalid
	case r.Leverage == 0 && !r.CloseOrder:
		return nil, errInvalidLeverage
	case r.Type == "limit" && r.Price <= 0:
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	body := make(map[string]any)
	body["clientOid"] = r.ClientOrderID
	body["symbol"] = r.Symbol
	body["type"] = r.Type
	if r.Side != "" {
		body["side"] = r.Side
	}
	if r.Leverage != 0 {
		body["leverage"] = r.Leverage
	}
	if r.StopPriceType != "" {
		body["stopPriceType"] = r.StopPriceType
	}
	if r.ReduceOnly {
		body["reduceOnly"] = true
	}
	if r.CloseOrder {
		body["closeOrder"] = true
	}
	if r.MarginMode != "" {
		body["marginMode"] = r.MarginMode
	}
	if r.Price != 0 {
		body["price"] = strconv.FormatFloat(r.Price, 'f', -1, 64)
	}
	if r.Size != 0 {
		body["size"] = r.Size
	}
	if r.TimeInForce != "" {
		body["timeInForce"] = r.TimeInForce
	}
	if r.PostOnly {
		body["postOnly"] = true
	}
	if r.TriggerStopUpPrice != 0 {
		body["triggerStopUpPrice"] = strconv.FormatFloat(r.TriggerStopUpPrice, 'f', -1, 64)
	}
	if r.TriggerStopDownPrice != 0 {
		body["triggerStopDownPrice"] = strconv.FormatFloat(r.TriggerStopDownPrice, 'f', -1, 64)
	}
	if r.PositionSide != "" {
		body["positionSide"] = r.PositionSide
	}
	if r.FundsType != "" {
		body["fundsType"] = r.FundsType
	}
	return body, nil
}

// CancelCopyTradingOrderByIDResponse holds the IDs of the copy trading futures orders KuCoin is cancelling
type CancelCopyTradingOrderByIDResponse struct {
	CancelledOrderIDs []string `json:"cancelledOrderIds"`
}

// CancelCopyTradingOrderByClientOrderIDRequest holds the parameters of Cancel Order By ClientOid
type CancelCopyTradingOrderByClientOrderIDRequest struct {
	Symbol        string
	ClientOrderID string
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// CancelCopyTradingOrderByClientOrderIDResponse holds the client order ID of the copy trading futures order KuCoin is
// cancelling
type CancelCopyTradingOrderByClientOrderIDResponse struct {
	ClientOrderID string `json:"clientOid"`
}

// CopyTradingMaxOpenSizeRequest holds the parameters of Get Max Open Size
type CopyTradingMaxOpenSizeRequest struct {
	Symbol   string
	Price    float64
	Leverage uint64
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// CopyTradingMaxOpenSizeResponse holds the largest copy trading futures position, in lots, that can be opened on each
// side
type CopyTradingMaxOpenSizeResponse struct {
	Symbol          string       `json:"symbol"`
	MaxBuyOpenSize  types.Number `json:"maxBuyOpenSize"`
	MaxSellOpenSize types.Number `json:"maxSellOpenSize"`
}

// CopyTradingMaxWithdrawMarginRequest holds the parameters of Get Max Withdraw Margin
type CopyTradingMaxWithdrawMarginRequest struct {
	Symbol string
	// PositionSide is BOTH in one-way mode, the default, and LONG or SHORT in hedge mode, where it is required
	PositionSide string
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// AddCopyTradingIsolatedMarginRequest holds the parameters of Add Isolated Margin
type AddCopyTradingIsolatedMarginRequest struct {
	Symbol string
	Margin float64
	// BusinessNumber is a unique ID of up to 36 characters, so that KuCoin processes the deposit only once
	BusinessNumber string
	// PositionSide is BOTH in one-way mode, the default, and LONG or SHORT in hedge mode, where it is required
	PositionSide string
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// AddCopyTradingIsolatedMarginResponse holds the copy trading futures position after margin was added to it
type AddCopyTradingIsolatedMarginResponse struct {
	ID                           string       `json:"id"`
	Symbol                       string       `json:"symbol"`
	AutoDeposit                  bool         `json:"autoDeposit"`
	MaintenanceMarginRequirement types.Number `json:"maintMarginReq"`
	RiskLimit                    uint64       `json:"riskLimit"`
	RealLeverage                 types.Number `json:"realLeverage"`
	CrossMode                    bool         `json:"crossMode"`
	MarginMode                   string       `json:"marginMode"`
	PositionSide                 string       `json:"positionSide"`
	Leverage                     types.Number `json:"leverage"`
	// DeleveragePercentage is the position's auto-deleveraging ranking percentile
	DeleveragePercentage float64    `json:"delevPercentage"`
	OpeningTimestamp     types.Time `json:"openingTimestamp"`
	CurrentTimestamp     types.Time `json:"currentTimestamp"`
	// CurrentQuantity is the position size in lots, negative for a short position
	CurrentQuantity   int64        `json:"currentQty"`
	CurrentCost       types.Number `json:"currentCost"`
	CurrentCommission types.Number `json:"currentComm"`
	UnrealisedCost    types.Number `json:"unrealisedCost"`
	RealisedGrossCost types.Number `json:"realisedGrossCost"`
	RealisedCost      types.Number `json:"realisedCost"`
	IsOpen            bool         `json:"isOpen"`
	MarkPrice         types.Number `json:"markPrice"`
	MarkValue         types.Number `json:"markValue"`
	PositionCost      types.Number `json:"posCost"`
	// PositionCross is the margin added to the position
	PositionCross types.Number `json:"posCross"`
	// PositionInitial is the position's leverage margin
	PositionInitial types.Number `json:"posInit"`
	// PositionCommission is the bankruptcy cost
	PositionCommission types.Number `json:"posComm"`
	// PositionLoss is the funding fees paid out
	PositionLoss   types.Number `json:"posLoss"`
	PositionMargin types.Number `json:"posMargin"`
	// PositionMaintenance is the position's maintenance margin
	PositionMaintenance types.Number `json:"posMaint"`
	MaintenanceMargin   types.Number `json:"maintMargin"`
	RealisedGrossPNL    types.Number `json:"realisedGrossPnl"`
	RealisedPNL         types.Number `json:"realisedPnl"`
	UnrealisedPNL       types.Number `json:"unrealisedPnl"`
	// UnrealisedPNLPercent is a ratio, 0.01 for 1%
	UnrealisedPNLPercent types.Number `json:"unrealisedPnlPcnt"`
	// UnrealisedROEPercent is the return on equity as a ratio, 0.01 for 1%
	UnrealisedROEPercent types.Number  `json:"unrealisedRoePcnt"`
	AverageEntryPrice    types.Number  `json:"avgEntryPrice"`
	LiquidationPrice     types.Number  `json:"liquidationPrice"`
	BankruptPrice        types.Number  `json:"bankruptPrice"`
	SettleCurrency       currency.Code `json:"settleCurrency"`
}

// RemoveCopyTradingIsolatedMarginRequest holds the parameters of Remove Isolated Margin
type RemoveCopyTradingIsolatedMarginRequest struct {
	Symbol string
	// WithdrawAmount is in USDT for USDT-margined contracts and in coins for coin-margined ones
	WithdrawAmount float64
	// PositionSide is BOTH in one-way mode, the default, and LONG or SHORT in hedge mode, where it is required
	PositionSide string
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// CopyTradingAutoDepositStatusRequest holds the parameters of Modify Isolated Margin Auto-Deposit Status
type CopyTradingAutoDepositStatusRequest struct {
	Symbol string
	Status bool
	// PositionSide is BOTH in one-way mode, the default, and LONG or SHORT in hedge mode, where it is required
	PositionSide string
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// SwitchCopyTradingMarginModeRequest holds the parameters of Switch Margin Mode
type SwitchCopyTradingMarginModeRequest struct {
	Symbol string
	// MarginMode is ISOLATED or CROSS
	MarginMode string
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// SwitchCopyTradingMarginModeResponse holds a copy trading futures symbol's margin mode after the switch
type SwitchCopyTradingMarginModeResponse struct {
	Symbol     string `json:"symbol"`
	MarginMode string `json:"marginMode"`
}

// CopyTradingCrossMarginLeverageRequest holds the parameters of Modify Cross Margin Leverage
type CopyTradingCrossMarginLeverageRequest struct {
	Symbol   string
	Leverage uint64
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// CopyTradingCrossMarginRequirementRequest holds the parameters of Get Cross Margin Requirement
type CopyTradingCrossMarginRequirementRequest struct {
	// Symbols takes up to 50 symbols
	Symbols  []string
	Leverage uint64
	// PositionValue is in USDT for USDT-margined contracts, USDC for USDC-margined ones and USD for coin-margined ones
	PositionValue float64
}

// CopyTradingCrossMarginRequirementResponse holds the cross margin rates of a copy trading futures symbol at a position value
type CopyTradingCrossMarginRequirementResponse struct {
	Symbol string `json:"symbol"`
	// Size is the position size in lots; the copy trading example quotes it and the futures one does not
	Size          types.Number `json:"size"`
	PositionValue types.Number `json:"positionValue"`
	// Price is the latest trade price the requirement was calculated at
	Price                 types.Number `json:"price"`
	InitialMarginRate     types.Number `json:"imr"`
	MaintenanceMarginRate types.Number `json:"mmr"`
}

// SwitchCopyTradingPositionModeResponse holds the copy trading futures position mode after the switch: 0 for one-way
// mode and 1 for hedge mode
type SwitchCopyTradingPositionModeResponse struct {
	PositionMode string `json:"positionMode"`
}

// CopyTradingPageRequest holds the funds type filter and page of the copy trading futures order and fill lists. Zero
// values are left out, so KuCoin applies its defaults
type CopyTradingPageRequest struct {
	// FundsType is selfFunds or leadTrial, or all for the order lists, which default to all; fills default to selfFunds
	FundsType   string
	CurrentPage uint64
	PageSize    uint64
}

// values returns the query of a copy trading order or fill list request, leaving out what is unset
func (r *CopyTradingPageRequest) values() (url.Values, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if r.FundsType != "" {
		params.Set("fundsType", r.FundsType)
	}
	if r.CurrentPage != 0 {
		params.Set("currentPage", strconv.FormatUint(r.CurrentPage, 10))
	}
	if r.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(r.PageSize, 10))
	}
	return params, nil
}

// CopyTradingOrdersResponse holds a page of copy trading futures orders
type CopyTradingOrdersResponse struct {
	CurrentPage uint64             `json:"currentPage"`
	PageSize    uint64             `json:"pageSize"`
	TotalNumber uint64             `json:"totalNum"`
	TotalPage   uint64             `json:"totalPage"`
	Items       []CopyTradingOrder `json:"items"`
}

// CopyTradingOrder holds a copy trading futures order
type CopyTradingOrder struct {
	ID                  string       `json:"id"`
	Symbol              string       `json:"symbol"`
	Type                string       `json:"type"`
	Side                string       `json:"side"`
	Price               types.Number `json:"price"`
	Size                uint64       `json:"size"`
	Value               types.Number `json:"value"`
	DealValue           types.Number `json:"dealValue"`
	DealSize            uint64       `json:"dealSize"`
	SelfTradePrevention string       `json:"stp"`
	Stop                string       `json:"stop"`
	StopPriceType       string       `json:"stopPriceType"`
	StopTriggered       bool         `json:"stopTriggered"`
	StopPrice           types.Number `json:"stopPrice"`
	TimeInForce         string       `json:"timeInForce"`
	PostOnly            bool         `json:"postOnly"`
	Hidden              bool         `json:"hidden"`
	Iceberg             bool         `json:"iceberg"`
	Leverage            types.Number `json:"leverage"`
	ForceHold           bool         `json:"forceHold"`
	CloseOrder          bool         `json:"closeOrder"`
	ReduceOnly          bool         `json:"reduceOnly"`
	VisibleSize         uint64       `json:"visibleSize"`
	// CancelAfter is the automatic cancellation delay in seconds; -1 when none is set
	CancelAfter   int64      `json:"cancelAfter"`
	Channel       string     `json:"channel"`
	ClientOrderID string     `json:"clientOid"`
	Remark        string     `json:"remark"`
	Tags          string     `json:"tags"`
	CancelExist   bool       `json:"cancelExist"`
	CreatedAt     types.Time `json:"createdAt"`
	IsActive      bool       `json:"isActive"`
	// SettleCurrency is the settlement currency
	SettleCurrency currency.Code `json:"settleCurrency"`
	// Shortcut is set when the order uses shortcut take profit or stop loss settings
	Shortcut             bool         `json:"shortcut"`
	TriggerStopUpPrice   types.Number `json:"triggerStopUpPrice"`
	TriggerStopDownPrice types.Number `json:"triggerStopDownPrice"`
	// TriggerOrderID is the order that triggered this conditional order; "0" when none did
	TriggerOrderID      string                  `json:"triggerOrderId"`
	FromReduce          bool                    `json:"fromReduce"`
	IsTrialFunds        bool                    `json:"isTrialFunds"`
	MarginMode          string                  `json:"marginMode"`
	DisplayType         string                  `json:"displayType"`
	AdvancedDisplayType string                  `json:"advancedDisplayType"`
	PositionSide        string                  `json:"positionSide"`
	AdvancedType        string                  `json:"advancedType"`
	ActivationPrice     types.Number            `json:"activationPrice"`
	CallbackRate        types.Number            `json:"callbackRate"`
	AdvancedStatus      string                  `json:"advancedStatus"`
	TrialCoupon         *CopyTradingTrialCoupon `json:"trialCoupon"`
}

// CopyTradingTrialCoupon holds the trial fund coupon a copy trading futures order used
type CopyTradingTrialCoupon struct {
	ID       uint64        `json:"id"`
	Code     string        `json:"code"`
	Currency currency.Code `json:"currency"`
	// FaceValue is documented as a decimal; the example sends it bare where other decimals here are quoted
	FaceValue        types.Number `json:"faceValue"`
	ExpiredPeriod    types.Time   `json:"expiredPeriod"`
	WithdrawHelpText string       `json:"withDrawHelpText"`
}

// CopyTradingFillsResponse holds a page of copy trading futures fills
type CopyTradingFillsResponse struct {
	Items []CopyTradingFill `json:"items"`
	// Offset is the self-funded fill offset, which KuCoin keeps only for response compatibility
	Offset uint64 `json:"offset"`
	// TrialOffset is the trial-funded fill offset, which KuCoin keeps only for response compatibility
	TrialOffset uint64 `json:"trialOffset"`
	// HasNext is set when another page may be available
	HasNext bool `json:"hasNext"`
}

// CopyTradingFill holds a copy trading futures fill
type CopyTradingFill struct {
	TradeID      string        `json:"tradeId"`
	OrderID      string        `json:"orderId"`
	Symbol       string        `json:"symbol"`
	Side         string        `json:"side"`
	PositionSide string        `json:"positionSide"`
	Price        types.Number  `json:"price"`
	Size         types.Number  `json:"size"`
	Fee          types.Number  `json:"fee"`
	FeeCurrency  currency.Code `json:"feeCurrency"`
	// Liquidity is maker or taker
	Liquidity    string     `json:"liquidity"`
	FundsType    string     `json:"fundsType"`
	TradeTime    types.Time `json:"tradeTime"`
	IsTrialFunds bool       `json:"isTrialFunds"`
}

// CopyTradingPositionResponse holds a copy trading futures position
type CopyTradingPositionResponse struct {
	ID                           string       `json:"id"`
	Symbol                       string       `json:"symbol"`
	AutoDeposit                  bool         `json:"autoDeposit"`
	MaintenanceMarginRequirement types.Number `json:"maintMarginReq"`
	// RiskLimit is the risk limit level
	RiskLimit    uint64       `json:"riskLimit"`
	RealLeverage types.Number `json:"realLeverage"`
	MarginMode   string       `json:"marginMode"`
	PositionSide string       `json:"positionSide"`
	Leverage     types.Number `json:"leverage"`
	// DeleveragePercentage is the position's auto-deleveraging queue percentile, from 0 to 1
	DeleveragePercentage float64    `json:"delevPercentage"`
	OpeningTimestamp     types.Time `json:"openingTimestamp"`
	CurrentTimestamp     types.Time `json:"currentTimestamp"`
	// CurrentQuantity is the position size in lots, negative for a short position
	CurrentQuantity   int64        `json:"currentQty"`
	CurrentCost       types.Number `json:"currentCost"`
	CurrentCommission types.Number `json:"currentComm"`
	UnrealisedCost    types.Number `json:"unrealisedCost"`
	RealisedGrossCost types.Number `json:"realisedGrossCost"`
	RealisedCost      types.Number `json:"realisedCost"`
	IsOpen            bool         `json:"isOpen"`
	MarkPrice         types.Number `json:"markPrice"`
	MarkValue         types.Number `json:"markValue"`
	PositionCost      types.Number `json:"posCost"`
	// PositionCross is the margin added to the position
	PositionCross types.Number `json:"posCross"`
	// PositionInitial is the position's initial margin
	PositionInitial types.Number `json:"posInit"`
	// PositionCommission is the estimated bankruptcy commission
	PositionCommission types.Number `json:"posComm"`
	// PositionLoss is the margin funding fees took from the position
	PositionLoss   types.Number `json:"posLoss"`
	PositionMargin types.Number `json:"posMargin"`
	// PositionMaintenance is the position's maintenance margin, fees included
	PositionMaintenance types.Number `json:"posMaint"`
	// PositionProfit is the funding income accumulated by the position
	PositionProfit types.Number `json:"posProfit"`
	// PositionCommissionFunding is the funding commission accumulated by the position
	PositionCommissionFunding types.Number `json:"posCommFunding"`
	MaintenanceMargin         types.Number `json:"maintMargin"`
	RealisedGrossPNL          types.Number `json:"realisedGrossPnl"`
	RealisedPNL               types.Number `json:"realisedPnl"`
	UnrealisedPNL             types.Number `json:"unrealisedPnl"`
	// UnrealisedPNLPercent is a ratio, 0.01 for 1%
	UnrealisedPNLPercent types.Number `json:"unrealisedPnlPcnt"`
	// UnrealisedROEPercent is the return on equity as a ratio, 0.01 for 1%
	UnrealisedROEPercent types.Number  `json:"unrealisedRoePcnt"`
	AverageEntryPrice    types.Number  `json:"avgEntryPrice"`
	LiquidationPrice     types.Number  `json:"liquidationPrice"`
	BankruptPrice        types.Number  `json:"bankruptPrice"`
	SettleCurrency       currency.Code `json:"settleCurrency"`
	// TrialCode is the trial fund coupon code of a position opened with lead trial funds
	TrialCode    string `json:"trialCode"`
	IsTrialFunds bool   `json:"isTrialFunds"`
	CrossMode    bool   `json:"crossMode"`
}

// CopyTradingAccountOverviewResponse holds a lead trader's copy trading futures account overview
type CopyTradingAccountOverviewResponse struct {
	// Currency is the settlement currency
	Currency  currency.Code            `json:"currency"`
	SelfFunds CopyTradingFundsOverview `json:"selfFunds"`
	LeadTrial CopyTradingFundsOverview `json:"leadTrial"`
	// TrialFund is the current or most recently used lead trial fund; nil when there is none
	TrialFund *CopyTradingTrialFund `json:"trialFund"`
}

// CopyTradingFundsOverview holds the balances of a copy trading futures funds pool
type CopyTradingFundsOverview struct {
	AccountEquity    types.Number `json:"accountEquity"`
	AvailableBalance types.Number `json:"availableBalance"`
	// MarginUsed is the margin positions and open orders hold
	MarginUsed    types.Number `json:"marginUsed"`
	UnrealisedPNL types.Number `json:"unrealisedPnl"`
}

// CopyTradingTrialFund holds a lead trial fund of a copy trading futures account
type CopyTradingTrialFund struct {
	TrialFundID string       `json:"trialFundId"`
	RewardID    string       `json:"rewardId"`
	Amount      types.Number `json:"amount"`
	// Status is the trial fund's lifecycle status
	Status           string     `json:"status"`
	ExpiresAt        types.Time `json:"expiresAt"`
	MaxLeverage      uint64     `json:"maxLeverage"`
	SupportContracts []string   `json:"supportContracts"`
	// WithdrawAllowed is set when the trial fund's profit may be withdrawn now
	WithdrawAllowed   bool         `json:"withdrawAllowed"`
	MaxWithdrawAmount types.Number `json:"maxWithdrawAmount"`
	// WithdrawThresholdTrade is the trading volume needed before profit may be withdrawn
	WithdrawThresholdTrade types.Number `json:"withdrawThresholdTrade"`
	// WithdrawThresholdProfit is the profit needed before it may be withdrawn
	WithdrawThresholdProfit types.Number `json:"withdrawThresholdProfit"`
}

// BatchSwitchCopyTradingMarginModeRequest holds the parameters of Batch switch copy trading futures margin mode
type BatchSwitchCopyTradingMarginModeRequest struct {
	// Items takes from 1 to 100 changes
	Items []CopyTradingMarginModeChange
	// FundsType applies to every item; it is selfFunds or leadTrial, and KuCoin uses selfFunds when it is empty
	FundsType string
}

// CopyTradingMarginModeChange holds the margin mode one symbol of a batch margin mode switch changes to
type CopyTradingMarginModeChange struct {
	Symbol string
	// MarginMode is CROSS or ISOLATED
	MarginMode string
	// FundsType is selfFunds or leadTrial; KuCoin uses selfFunds when it is empty
	FundsType string
}

// BatchSwitchCopyTradingMarginModeResponse holds the outcome of each change of a batch margin mode switch
type BatchSwitchCopyTradingMarginModeResponse struct {
	FundsType string                        `json:"fundsType"`
	Items     []CopyTradingMarginModeResult `json:"items"`
}

// CopyTradingMarginModeResult holds the outcome of one change of a batch margin mode switch. A change can fail while
// the batch succeeds: Status then says so, and ErrorCode and ErrorMessage say why
type CopyTradingMarginModeResult struct {
	Symbol              string `json:"symbol"`
	RequestedMarginMode string `json:"requestedMarginMode"`
	// MarginMode is the symbol's margin mode after the change was processed
	MarginMode   string `json:"marginMode"`
	Status       string `json:"status"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}
