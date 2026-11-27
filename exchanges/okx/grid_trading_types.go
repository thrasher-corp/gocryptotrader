package okx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AlgoOrdTypeGrid algo order type,grid
const AlgoOrdTypeGrid = "grid"

// AlgoOrdTypeContractGrid algo order type, contract_grid
const AlgoOrdTypeContractGrid = "contract_grid"

// TradingBotAlgoIDRequest holds the algo ID a trading bot request acts on
type TradingBotAlgoIDRequest struct {
	AlgoID string `json:"algoId"`
}

// TradingBotAlgoIDResponse holds the algo ID a trading bot request acted on
type TradingBotAlgoIDResponse struct {
	AlgoID string `json:"algoId"`
}

// TradingBotOrderResponse holds the result of placing, amending or stopping a trading bot algo order
type TradingBotOrderResponse struct {
	AlgoID            string `json:"algoId"`
	AlgoClientOrderID string `json:"algoClOrdId"`
	Tag               string `json:"tag"`
	// StatusCode is never negative on the wire; it stays int64 to match
	// getStatusError and the package's other sCode declarations.
	StatusCode    uint64 `json:"sCode,string"`
	StatusMessage string `json:"sMsg"`
}

// TradingBotTriggerParameter holds a condition that starts or stops a grid or DCA bot. OKX returns TriggerTime and
// TriggerType; requests leave them unset.
type TradingBotTriggerParameter struct {
	TriggerAction    string       `json:"triggerAction"`
	TriggerStrategy  string       `json:"triggerStrategy"`
	DelaySeconds     types.Number `json:"delaySeconds,omitempty"`
	Timeframe        string       `json:"timeframe,omitempty"`
	Threshold        types.Number `json:"thold,omitempty"`
	TriggerCondition string       `json:"triggerCond,omitempty"`
	TimePeriod       types.Number `json:"timePeriod,omitempty"`
	TriggerPrice     types.Number `json:"triggerPx,omitempty"`
	StopType         string       `json:"stopType,omitempty"`
	TriggerTime      types.Time   `json:"triggerTime,omitzero"`
	TriggerType      string       `json:"triggerType,omitempty"`
}

// TradingBotPositionResponse holds a position held by a contract grid or signal bot
type TradingBotPositionResponse struct {
	AlgoID                       string        `json:"algoId"`
	AlgoClientOrderID            string        `json:"algoClOrdId"`
	InstrumentType               string        `json:"instType"`
	InstrumentID                 string        `json:"instId"`
	CreationTime                 types.Time    `json:"cTime"`
	UpdateTime                   types.Time    `json:"uTime"`
	AveragePrice                 types.Number  `json:"avgPx"`
	Currency                     currency.Code `json:"ccy"`
	Leverage                     types.Number  `json:"lever"`
	LiquidationPrice             types.Number  `json:"liqPx"`
	PositionSide                 string        `json:"posSide"`
	Position                     types.Number  `json:"pos"`
	MarginMode                   string        `json:"mgnMode"`
	MarginRatio                  types.Number  `json:"mgnRatio"`
	InitialMarginRequirement     types.Number  `json:"imr"`
	MaintenanceMarginRequirement types.Number  `json:"mmr"`
	UnrealisedPNL                types.Number  `json:"upl"`
	UnrealisedPNLRatio           types.Number  `json:"uplRatio"`
	LastTradedPrice              types.Number  `json:"last"`
	NotionalUSD                  types.Number  `json:"notionalUsd"`
	AutoDeleveraging             types.Number  `json:"adl"`
	MarkPrice                    types.Number  `json:"markPx"`
}

// PlaceGridAlgoOrderRequest holds the parameters to place a spot or contract grid algo order
type PlaceGridAlgoOrderRequest struct {
	InstrumentID           string                       `json:"instId"`
	AlgoOrderType          string                       `json:"algoOrdType"`
	MaxPrice               float64                      `json:"maxPx,string"`
	MinPrice               float64                      `json:"minPx,string"`
	GridNumber             uint64                       `json:"gridNum,string"`
	RunType                string                       `json:"runType,omitempty"`
	TakeProfitTriggerPrice float64                      `json:"tpTriggerPx,string,omitempty"`
	StopLossTriggerPrice   float64                      `json:"slTriggerPx,string,omitempty"`
	AlgoClientOrderID      string                       `json:"algoClOrdId,omitempty"`
	Tag                    string                       `json:"tag,omitempty"`
	ProfitSharingRatio     float64                      `json:"profitSharingRatio,string,omitempty"`
	TriggerParameters      []TradingBotTriggerParameter `json:"triggerParams,omitempty"`

	// Spot grid
	QuoteSize          float64       `json:"quoteSz,string,omitempty"`
	BaseSize           float64       `json:"baseSz,string,omitempty"`
	TradeQuoteCurrency currency.Code `json:"tradeQuoteCcy,omitzero"`

	// Contract grid
	Size            float64 `json:"sz,string,omitempty"`
	Direction       string  `json:"direction,omitempty"`
	Leverage        float64 `json:"lever,string,omitempty"`
	BasePosition    bool    `json:"basePos,omitempty"`
	TakeProfitRatio float64 `json:"tpRatio,string,omitempty"`
	StopLossRatio   float64 `json:"slRatio,string,omitempty"`
}

// AmendGridAlgoOrderBasicParametersRequest holds the new price range and grid quantity of a grid algo order
type AmendGridAlgoOrderBasicParametersRequest struct {
	AlgoID      string  `json:"algoId"`
	MinPrice    float64 `json:"minPx,string"`
	MaxPrice    float64 `json:"maxPx,string"`
	GridNumber  uint64  `json:"gridNum,string"`
	TopUpAmount float64 `json:"topupAmount,string,omitempty"`
}

// AmendGridAlgoOrderBasicParametersResponse holds the investment top up a grid algo order amendment requires
type AmendGridAlgoOrderBasicParametersResponse struct {
	AlgoID              string       `json:"algoId"`
	RequiredTopUpAmount types.Number `json:"requiredTopupAmount"`
	MaxTopUpAmount      types.Number `json:"maxTopupAmount"`
}

// AmendGridAlgoOrderRequest holds the parameters to amend a grid algo order. A nil trigger price or ratio is left
// unchanged; a pointer to zero sends "", which cancels it.
type AmendGridAlgoOrderRequest struct {
	AlgoID                 string                       `json:"algoId"`
	InstrumentID           string                       `json:"instId"`
	StopLossTriggerPrice   *types.Number                `json:"slTriggerPx,omitempty"`
	TakeProfitTriggerPrice *types.Number                `json:"tpTriggerPx,omitempty"`
	TakeProfitRatio        *types.Number                `json:"tpRatio,omitempty"`
	StopLossRatio          *types.Number                `json:"slRatio,omitempty"`
	TopUpAmount            float64                      `json:"topUpAmt,string,omitempty"`
	TriggerParameters      []TradingBotTriggerParameter `json:"triggerParams,omitempty"`
}

// StopGridAlgoOrderRequest holds a grid algo order to stop
type StopGridAlgoOrderRequest struct {
	AlgoID        string `json:"algoId"`
	InstrumentID  string `json:"instId"`
	AlgoOrderType string `json:"algoOrdType"`
	StopType      string `json:"stopType"`
}

// ClosePositionForContractGridRequest holds the parameters to close a contract grid's position
type ClosePositionForContractGridRequest struct {
	AlgoID                  string  `json:"algoId"`
	MarketCloseAllPositions bool    `json:"mktClose"`
	Size                    float64 `json:"sz,string,omitempty"`
	Price                   float64 `json:"px,string,omitempty"`
}

// CancelClosePositionOrderRequest holds a contract grid close position order to cancel
type CancelClosePositionOrderRequest struct {
	AlgoID  string `json:"algoId"`
	OrderID string `json:"ordId"`
}

// ContractGridCloseOrderResponse holds a contract grid's close position order
type ContractGridCloseOrderResponse struct {
	AlgoID            string `json:"algoId"`
	OrderID           string `json:"ordId"`
	AlgoClientOrderID string `json:"algoClOrdId"`
	Tag               string `json:"tag"`
}

// InstantTriggerGridAlgoOrderRequest holds a grid algo order to trigger immediately
type InstantTriggerGridAlgoOrderRequest struct {
	AlgoID      string  `json:"algoId"`
	TopUpAmount float64 `json:"topUpAmt,string,omitempty"`
}

// InstantTriggerGridAlgoOrderResponse holds the grid algo order triggered
type InstantTriggerGridAlgoOrderResponse struct {
	AlgoID            string `json:"algoId"`
	AlgoClientOrderID string `json:"algoClOrdId"`
}

// GridAlgoOrderListRequest holds the filters for listing pending or historic grid algo orders
type GridAlgoOrderListRequest struct {
	AlgoOrderType  string
	AlgoID         string
	InstrumentID   string
	InstrumentType string
	After          string // Returns records earlier than this algo ID
	Before         string // Returns records newer than this algo ID
	Limit          uint64
}

// GridAlgoOrderRequest identifies a grid algo order
type GridAlgoOrderRequest struct {
	AlgoOrderType string
	AlgoID        string
}

// GridRebateTransfer holds a grid algo order's rebate transfer
type GridRebateTransfer struct {
	Rebate         types.Number  `json:"rebate"`
	RebateCurrency currency.Code `json:"rebateCcy"`
}

// GridAlgoOrderResponse holds a grid algo order; the details endpoint returns the fields the list endpoints omit
type GridAlgoOrderResponse struct {
	AlgoID                 string                       `json:"algoId"`
	AlgoClientOrderID      string                       `json:"algoClOrdId"`
	InstrumentType         string                       `json:"instType"`
	InstrumentID           string                       `json:"instId"`
	CreationTime           types.Time                   `json:"cTime"`
	UpdateTime             types.Time                   `json:"uTime"`
	AlgoOrderType          string                       `json:"algoOrdType"`
	State                  string                       `json:"state"`
	RebateTransfers        []GridRebateTransfer         `json:"rebateTrans"`
	TriggerParameters      []TradingBotTriggerParameter `json:"triggerParams"`
	MaxPrice               types.Number                 `json:"maxPx"`
	MinPrice               types.Number                 `json:"minPx"`
	GridNumber             types.Number                 `json:"gridNum"`
	RunType                string                       `json:"runType"`
	TakeProfitTriggerPrice types.Number                 `json:"tpTriggerPx"`
	StopLossTriggerPrice   types.Number                 `json:"slTriggerPx"`
	TradeNumber            types.Number                 `json:"tradeNum"`
	ArbitrageNumber        types.Number                 `json:"arbitrageNum"`
	SingleAmount           types.Number                 `json:"singleAmt"`
	PerMinProfitRate       types.Number                 `json:"perMinProfitRate"`
	PerMaxProfitRate       types.Number                 `json:"perMaxProfitRate"`
	RunPrice               types.Number                 `json:"runPx"`
	TotalPNL               types.Number                 `json:"totalPnl"`
	PNLRatio               types.Number                 `json:"pnlRatio"`
	Investment             types.Number                 `json:"investment"`
	GridProfit             types.Number                 `json:"gridProfit"`
	FloatProfit            types.Number                 `json:"floatProfit"`
	TotalAnnualisedRate    types.Number                 `json:"totalAnnualizedRate"`
	AnnualisedRate         types.Number                 `json:"annualizedRate"`
	CancelType             string                       `json:"cancelType"`
	StopType               string                       `json:"stopType"`
	ActiveOrderNumber      types.Number                 `json:"activeOrdNum"`
	QuoteSize              types.Number                 `json:"quoteSz"`
	BaseSize               types.Number                 `json:"baseSz"`
	CurrentQuoteSize       types.Number                 `json:"curQuoteSz"`
	CurrentBaseSize        types.Number                 `json:"curBaseSz"`
	Profit                 types.Number                 `json:"profit"`
	StopResult             string                       `json:"stopResult"`
	Direction              string                       `json:"direction"`
	BasePosition           bool                         `json:"basePos"`
	Size                   types.Number                 `json:"sz"`
	Leverage               types.Number                 `json:"lever"`
	ActualLeverage         types.Number                 `json:"actualLever"`
	LiquidationPrice       types.Number                 `json:"liqPx"`
	Underlying             string                       `json:"uly"`
	InstrumentFamily       string                       `json:"instFamily"`
	OrderFrozen            types.Number                 `json:"ordFrozen"`
	AvailableEquity        types.Number                 `json:"availEq"`
	Equity                 types.Number                 `json:"eq"`
	Tag                    string                       `json:"tag"`
	ProfitSharingRatio     types.Number                 `json:"profitSharingRatio"`
	CopyType               string                       `json:"copyType"`
	TakeProfitRatio        types.Number                 `json:"tpRatio"`
	StopLossRatio          types.Number                 `json:"slRatio"`
	Fee                    types.Number                 `json:"fee"`
	FeeCurrency            currency.Code                `json:"feeCcy"`
	FundingFee             types.Number                 `json:"fundingFee"`
	TradeQuoteCurrency     currency.Code                `json:"tradeQuoteCcy"`
}

// GridAlgoSubOrdersRequest holds the filters for listing a grid algo order's sub orders
type GridAlgoSubOrdersRequest struct {
	AlgoOrderType string
	AlgoID        string
	SubOrderType  string // Sub order state: live or filled
	GroupID       string
	After         string // Returns records earlier than this order ID
	Before        string // Returns records newer than this order ID
	Limit         uint64
}

// GridSubOrderResponse holds a grid algo order's sub order
type GridSubOrderResponse struct {
	AlgoID              string        `json:"algoId"`
	AlgoClientOrderID   string        `json:"algoClOrdId"`
	InstrumentType      string        `json:"instType"`
	InstrumentID        string        `json:"instId"`
	AlgoOrderType       string        `json:"algoOrdType"`
	GroupID             string        `json:"groupId"`
	OrderID             string        `json:"ordId"`
	CreationTime        types.Time    `json:"cTime"`
	UpdateTime          types.Time    `json:"uTime"`
	TradeMode           string        `json:"tdMode"`
	Currency            currency.Code `json:"ccy"`
	OrderType           string        `json:"ordType"`
	Size                types.Number  `json:"sz"`
	State               string        `json:"state"`
	Side                order.Side    `json:"side"`
	Price               types.Number  `json:"px"`
	Fee                 types.Number  `json:"fee"`
	FeeCurrency         currency.Code `json:"feeCcy"`
	Rebate              types.Number  `json:"rebate"`
	RebateCurrency      currency.Code `json:"rebateCcy"`
	AveragePrice        types.Number  `json:"avgPx"`
	AccumulatedFillSize types.Number  `json:"accFillSz"`
	PositionSide        string        `json:"posSide"`
	PNL                 types.Number  `json:"pnl"`
	ContractValue       types.Number  `json:"ctVal"`
	Leverage            types.Number  `json:"lever"`
	Tag                 string        `json:"tag"`
	PushTime            types.Time    `json:"pTime"` // Only set by the grid sub orders websocket channel
}

// SpotGridWithdrawIncomeResponse holds the profit withdrawn from a spot grid algo order
type SpotGridWithdrawIncomeResponse struct {
	AlgoID            string       `json:"algoId"`
	AlgoClientOrderID string       `json:"algoClOrdId"`
	Profit            types.Number `json:"profit"`
}

// ComputeGridMarginBalanceRequest holds the parameters to compute a contract grid's adjustable margin balance
type ComputeGridMarginBalanceRequest struct {
	AlgoID                  string  `json:"algoId"`
	AdjustMarginBalanceType string  `json:"type"`
	Amount                  float64 `json:"amt,string,omitempty"`
}

// ComputeGridMarginBalanceResponse holds a contract grid's adjustable margin balance
type ComputeGridMarginBalanceResponse struct {
	MaxAmount types.Number `json:"maxAmt"`
	Leverage  types.Number `json:"lever"`
}

// AdjustGridMarginBalanceRequest holds the parameters to adjust a contract grid's margin balance by an amount or a
// percentage
type AdjustGridMarginBalanceRequest struct {
	AlgoID                  string  `json:"algoId"`
	AdjustMarginBalanceType string  `json:"type"`
	Amount                  float64 `json:"amt,string,omitempty"`
	Percent                 float64 `json:"percent,string,omitempty"`
}

// AdjustGridMarginBalanceResponse holds the contract grid whose margin balance was adjusted
type AdjustGridMarginBalanceResponse struct {
	AlgoID            string `json:"algoId"`
	AlgoClientOrderID string `json:"algoClOrdId"`
}

// AddGridInvestmentRequest holds an investment to add to a grid algo order. A nil AllowReinvestProfit keeps OKX's
// default of reinvesting.
type AddGridInvestmentRequest struct {
	AlgoID              string  `json:"algoId"`
	Amount              float64 `json:"amt,string"`
	AllowReinvestProfit *bool   `json:"allowReinvestProfit,omitempty,string"`
}

// GridAIParameterRequest holds the parameters to retrieve a grid's AI parameters
type GridAIParameterRequest struct {
	AlgoOrderType string
	InstrumentID  string
	Direction     string // Required for contract grids
	Duration      string
}

// GridAIParameterResponse holds a grid's AI parameters
type GridAIParameterResponse struct {
	InstrumentID       string        `json:"instId"`
	AlgoOrderType      string        `json:"algoOrdType"`
	Duration           string        `json:"duration"`
	GridNumber         types.Number  `json:"gridNum"`
	MaxPrice           types.Number  `json:"maxPx"`
	MinPrice           types.Number  `json:"minPx"`
	PerMaxProfitRate   types.Number  `json:"perMaxProfitRate"`
	PerMinProfitRate   types.Number  `json:"perMinProfitRate"`
	PerGridProfitRatio types.Number  `json:"perGridProfitRatio"`
	AnnualisedRate     types.Number  `json:"annualizedRate"`
	MinInvestment      types.Number  `json:"minInvestment"`
	Currency           currency.Code `json:"ccy"`
	RunType            string        `json:"runType"`
	Direction          string        `json:"direction"`
	Leverage           types.Number  `json:"lever"`
	SourceCurrency     currency.Code `json:"sourceCcy"`
}

// InvestmentData holds an investment amount in a currency
type InvestmentData struct {
	Amount   types.Number  `json:"amt"`
	Currency currency.Code `json:"ccy"`
}

// ComputeMinInvestmentRequest holds the grid parameters to compute the minimum investment for
type ComputeMinInvestmentRequest struct {
	InstrumentID    string           `json:"instId"`
	AlgoOrderType   string           `json:"algoOrdType"`
	MaxPrice        float64          `json:"maxPx,string"`
	MinPrice        float64          `json:"minPx,string"`
	GridNumber      uint64           `json:"gridNum,string"`
	RunType         string           `json:"runType"`
	Direction       string           `json:"direction,omitempty"`
	Leverage        float64          `json:"lever,string,omitempty"`
	BasePosition    bool             `json:"basePos,omitempty"`
	InvestmentType  string           `json:"investmentType,omitempty"`
	TriggerStrategy string           `json:"triggerStrategy,omitempty"`
	TopUpAmount     float64          `json:"topUpAmt,string,omitempty"`
	InvestmentData  []InvestmentData `json:"investmentData,omitempty"`
}

// ComputeMinInvestmentResponse holds a grid's minimum investment
type ComputeMinInvestmentResponse struct {
	MinInvestmentData []InvestmentData `json:"minInvestmentData"`
	SingleAmount      types.Number     `json:"singleAmt"`
}

// RSIBackTestingRequest holds the parameters to back test a relative strength index (RSI) trigger
type RSIBackTestingRequest struct {
	InstrumentID     string
	Timeframe        kline.Interval
	Threshold        uint64 // An integer between 1 and 100
	TimePeriod       uint64
	TriggerCondition string
	Duration         string
}

// RSIBackTestingResponse holds the number of times an RSI trigger fired
type RSIBackTestingResponse struct {
	TriggerNumber types.Number `json:"triggerNum"`
}

// MaxGridQuantityRequest holds the parameters to retrieve a grid's maximum grid quantity
type MaxGridQuantityRequest struct {
	InstrumentID  string
	RunType       string
	AlgoOrderType string
	MaxPrice      float64
	MinPrice      float64
	Leverage      float64 // Required for contract grids
}

// MaxGridQuantityResponse holds a grid's maximum grid quantity
type MaxGridQuantityResponse struct {
	MaxGridQuantity types.Number `json:"maxGridQty"`
}

// CopyGridAlgoOrderRequest holds the parameters to copy a lead trader's grid algo order. A nil AutoReserve leaves
// it unsent, as spot grids require.
type CopyGridAlgoOrderRequest struct {
	InstrumentID      string  `json:"instId"`
	AlgoOrderType     string  `json:"algoOrdType"`
	SourceAlgoID      string  `json:"sourceAlgoId"`
	QuoteSize         float64 `json:"quoteSz,string,omitempty"`
	Leverage          float64 `json:"lever,string,omitempty"`
	AutoReserve       *bool   `json:"autoReserve,omitempty"`
	Size              float64 `json:"sz,string,omitempty"`
	ActualMarginSize  float64 `json:"actualMarginSz,string,omitempty"`
	ExtraMarginSize   float64 `json:"extraMarginSz,string,omitempty"`
	AlgoClientOrderID string  `json:"algoClOrdId,omitempty"`
	Tag               string  `json:"tag,omitempty"`
}
