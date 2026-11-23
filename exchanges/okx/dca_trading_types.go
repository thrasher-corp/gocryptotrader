package okx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AlgoOrdTypeSpotDCA algo order type, spot_dca
const AlgoOrdTypeSpotDCA = "spot_dca"

// AlgoOrdTypeContractDCA algo order type, contract_dca
const AlgoOrdTypeContractDCA = "contract_dca"

// PlaceDCAAlgoOrderRequest holds the parameters to place a spot or contract DCA algo order. A nil AllowReinvest keeps
// OKX's default of reinvesting profit.
type PlaceDCAAlgoOrderRequest struct {
	InstrumentID         string                       `json:"instId"`
	AlgoOrderType        string                       `json:"algoOrdType"`
	InitialOrderAmount   float64                      `json:"initOrdAmt,string"`
	AllowReinvest        *bool                        `json:"allowReinvest,omitempty"`
	SafetyOrderAmount    float64                      `json:"safetyOrdAmt,string,omitempty"`
	MaxSafetyOrders      uint64                       `json:"maxSafetyOrds,string"`
	PriceSteps           float64                      `json:"pxSteps,string,omitempty"`
	PriceStepsMultiplier float64                      `json:"pxStepsMult,string,omitempty"`
	VolumeMultiplier     float64                      `json:"volMult,string,omitempty"`
	TakeProfitPercentage float64                      `json:"tpPct,string"`
	StopLossPercentage   float64                      `json:"slPct,string,omitempty"`
	StopLossMode         string                       `json:"slMode,omitempty"`
	Direction            string                       `json:"direction,omitempty"`
	Leverage             float64                      `json:"lever,string,omitempty"`
	TriggerParameters    []TradingBotTriggerParameter `json:"triggerParams"`
	ProfitSharingRatio   float64                      `json:"profitSharingRatio,string,omitempty"`
	TrackingMode         string                       `json:"trackingMode,omitempty"`
	Tag                  string                       `json:"tag,omitempty"`
	AlgoClientOrderID    string                       `json:"algoClOrdId,omitempty"`
	TradeQuoteCurrency   currency.Code                `json:"tradeQuoteCcy,omitzero"`
}

// AmendSpotDCABasicParametersRequest holds the new settings of a spot DCA algo order
type AmendSpotDCABasicParametersRequest struct {
	AlgoID               string                       `json:"algoId"`
	PriceSteps           float64                      `json:"pxSteps,string"`
	PriceStepsMultiplier float64                      `json:"pxStepsMult,string"`
	VolumeMultiplier     float64                      `json:"volMult,string"`
	TakeProfitPercentage float64                      `json:"tpPct,string"`
	StopLossPercentage   float64                      `json:"slPct,string"`
	InitialOrderAmount   float64                      `json:"initOrdAmt,string"`
	SafetyOrderAmount    float64                      `json:"safetyOrdAmt,string"`
	MaxSafetyOrders      uint64                       `json:"maxSafetyOrds,string"`
	ReserveFunds         bool                         `json:"reserveFunds"`
	TriggerParameters    []TradingBotTriggerParameter `json:"triggerParams"`
}

// StopDCAAlgoOrderRequest holds a DCA algo order to stop
type StopDCAAlgoOrderRequest struct {
	AlgoID        string `json:"algoId"`
	AlgoOrderType string `json:"algoOrdType"`
	StopType      string `json:"stopType"`
}

// DCAAlgoOrderListRequest holds the filters for listing ongoing or historic DCA algo orders
type DCAAlgoOrderListRequest struct {
	AlgoOrderType string
	AlgoID        string
	After         string // Returns records earlier than this algo ID
	Before        string // Returns records newer than this algo ID
	Limit         uint64
}

// DCAAlgoOrderResponse holds a DCA algo order
type DCAAlgoOrderResponse struct {
	AlgoID               string                       `json:"algoId"`
	AlgoOrderType        string                       `json:"algoOrdType"`
	InstrumentID         string                       `json:"instId"`
	CopyType             string                       `json:"copyType"`
	State                string                       `json:"state"`
	CancelType           string                       `json:"cancelType"`
	Direction            string                       `json:"direction"`
	Leverage             types.Number                 `json:"lever"`
	InitialOrderAmount   types.Number                 `json:"initOrdAmt"`
	SafetyOrderAmount    types.Number                 `json:"safetyOrdAmt"`
	MaxSafetyOrders      types.Number                 `json:"maxSafetyOrds"`
	PriceSteps           types.Number                 `json:"pxSteps"`
	PriceStepsMultiplier types.Number                 `json:"pxStepsMult"`
	VolumeMultiplier     types.Number                 `json:"volMult"`
	TakeProfitPriceRange types.Number                 `json:"tpPxRange"`
	StopLossPercentage   types.Number                 `json:"slPct"`
	StopLossMode         string                       `json:"slMode"`
	AllowReinvest        bool                         `json:"allowReinvest"`
	TotalPNL             types.Number                 `json:"totalPnl"`
	PNLRatio             types.Number                 `json:"pnlRatio"`
	TotalFundingFee      types.Number                 `json:"totalFundingFee"`
	FundingFee           types.Number                 `json:"fundingFee"`
	InvestmentAmount     types.Number                 `json:"investmentAmt"`
	InvestmentCurrency   currency.Code                `json:"investmentCcy"`
	ArbitragePNL         types.Number                 `json:"arbitragePnL"`
	TransferInMargin     types.Number                 `json:"transferInMargin"`
	ProfitSharingRatio   types.Number                 `json:"profitSharingRatio"`
	TrackingMode         string                       `json:"trackingMode"`
	TriggerParameters    []TradingBotTriggerParameter `json:"triggerParams"`
	CreationTime         types.Time                   `json:"cTime"`
	UpdateTime           types.Time                   `json:"uTime"`
	ContractValue        types.Number                 `json:"ctVal"`
	TradeQuoteCurrency   currency.Code                `json:"tradeQuoteCcy"`
}

// DCASubOrdersRequest holds the filters for listing a DCA algo order's sub orders
type DCASubOrdersRequest struct {
	AlgoID        string
	AlgoOrderType string
	CycleID       string
	After         string // Returns records earlier than this order ID
	Before        string // Returns records newer than this order ID
	Limit         uint64
}

// DCASubOrderResponse holds a DCA algo order's sub order
type DCASubOrderResponse struct {
	CycleID            string        `json:"cycleId"`
	OrderID            string        `json:"ordId"`
	AverageFillPrice   types.Number  `json:"avgFillPx"`
	Direction          string        `json:"direction"`
	Side               string        `json:"side"`
	OrderType          string        `json:"ordType"`
	Price              types.Number  `json:"px"`
	Size               types.Number  `json:"sz"`
	FilledSize         types.Number  `json:"filledSz"`
	State              string        `json:"state"`
	Fee                types.Number  `json:"fee"`
	Rebate             types.Number  `json:"rebate"`
	RebateCurrency     currency.Code `json:"rebateCcy"`
	Leverage           types.Number  `json:"lever"`
	InstrumentID       string        `json:"instId"`
	ContractValue      types.Number  `json:"ctVal"`
	FillTime           types.Time    `json:"fillTime"`
	CreationTime       types.Time    `json:"cTime"`
	UpdateTime         types.Time    `json:"uTime"`
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy"`
}

// AddDCAInvestmentRequest holds a manual buy adding investment to a DCA algo order
type AddDCAInvestmentRequest struct {
	AlgoID             string        `json:"algoId"`
	AlgoOrderType      string        `json:"algoOrdType"`
	Price              float64       `json:"price,string"`
	Amount             float64       `json:"amt,string"`
	OrderType          string        `json:"ordType,omitempty"`
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy,omitzero"`
}

// AddDCAInvestmentResponse holds the result of adding investment to a DCA algo order
type AddDCAInvestmentResponse struct {
	AlgoID            string       `json:"algoId"`
	AlgoClientOrderID string       `json:"algoClOrdId"`
	AlgoOrderType     string       `json:"algoOrdType"`
	Tag               string       `json:"tag"`
	DiffAmount        types.Number `json:"diffAmount"`
	// StatusCode is never negative on the wire; it stays int64 to match getStatusError
	StatusCode    uint64 `json:"sCode,string"`
	StatusMessage string `json:"sMsg"`
}

// AmendDCAReinvestmentRequest holds whether a contract DCA algo order reinvests its profit
type AmendDCAReinvestmentRequest struct {
	AlgoID        string `json:"algoId"`
	AlgoOrderType string `json:"algoOrdType"`
	AllowReinvest bool   `json:"allowReinvest"`
}

// AmendDCATakeProfitRequest holds a contract DCA algo order's new take profit price
type AmendDCATakeProfitRequest struct {
	AlgoID          string  `json:"algoId"`
	AlgoOrderType   string  `json:"algoOrdType"`
	TakeProfitPrice float64 `json:"tpPrice,string"`
}

// DCASettingResponse holds the result of changing a DCA algo order's settings or margin
type DCASettingResponse struct {
	AlgoID        string `json:"algoId"`
	AlgoOrderType string `json:"algoOrdType"`
	// StatusCode is never negative on the wire; it stays int64 to match getStatusError
	StatusCode    uint64 `json:"sCode,string"`
	StatusMessage string `json:"sMsg"`
}

// DCAAlgoOrderRequest identifies a DCA algo order
type DCAAlgoOrderRequest struct {
	AlgoID        string
	AlgoOrderType string
}

// DCAPositionDetailsResponse holds a DCA algo order's position in its current cycle
type DCAPositionDetailsResponse struct {
	AlgoID             string        `json:"algoId"`
	AlgoClientOrderID  string        `json:"algoClOrdId"`
	AlgoOrderType      string        `json:"algoOrdType"`
	InstrumentID       string        `json:"instId"`
	CurrentCycleID     string        `json:"curCycleld"` // OKX's docs spell the key with a lower case L
	StartTime          types.Time    `json:"startTime"`
	FilledManualOrders types.Number  `json:"fillManualOrds"`
	FilledSafetyOrders types.Number  `json:"fillSafetyOrds"`
	FundingFee         types.Number  `json:"fundingFee"`
	InitialPrice       types.Number  `json:"initPx"`
	NotionalUSD        types.Number  `json:"notionalUsd"`
	AveragePrice       types.Number  `json:"avgPx"`
	UnrealisedPNL      types.Number  `json:"upl"`
	LiquidationPrice   types.Number  `json:"liqPx"`
	Size               types.Number  `json:"sz"`
	BaseSize           types.Number  `json:"baseSz"`
	QuoteSize          types.Number  `json:"quoteSz"`
	StopLossPrice      types.Number  `json:"slPx"`
	TakeProfitPrice    types.Number  `json:"tpPx"`
	Fee                types.Number  `json:"fee"`
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy"`
}

// DCACycleListRequest holds the filters for listing a DCA algo order's cycles
type DCACycleListRequest struct {
	AlgoID        string
	AlgoOrderType string
	InstrumentID  string
	After         string // Returns records earlier than this cycle ID
	Before        string // Returns records newer than this cycle ID
	Limit         uint64
}

// DCACycleResponse holds a DCA algo order's cycle
type DCACycleResponse struct {
	AlgoID            string       `json:"algoId"`
	AlgoClientOrderID string       `json:"algoClOrdId"`
	CycleID           string       `json:"cycleId"`
	CurrentCycle      bool         `json:"currentCycle"`
	RealisedPNL       types.Number `json:"realizedPnl"`
	StartTime         types.Time   `json:"startTime"`
	EndTime           types.Time   `json:"endTime"`
	Fee               types.Number `json:"fee"`
	AveragePrice      types.Number `json:"avgPx"`
	TakeProfitPrice   types.Number `json:"tpPx"`
}

// DCAMarginRequest holds an amount of margin to add to or reduce from a contract DCA algo order
type DCAMarginRequest struct {
	AlgoID string  `json:"algoId"`
	Amount float64 `json:"amt,string"`
}
