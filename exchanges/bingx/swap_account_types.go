package bingx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SwapAccountDataResponse holds an asset of the perpetual account from Query account data
type SwapAccountDataResponse struct {
	UserID           string        `json:"userId"`
	ShortUID         string        `json:"shortUid"`
	Asset            currency.Code `json:"asset"`
	Balance          types.Number  `json:"balance"`
	Equity           types.Number  `json:"equity"`
	UnrealisedProfit types.Number  `json:"unrealizedProfit"`
	RealisedProfit   types.Number  `json:"realisedProfit"`
	AvailableMargin  types.Number  `json:"availableMargin"`
	UsedMargin       types.Number  `json:"usedMargin"`
	FreezedMargin    types.Number  `json:"freezedMargin"` // frozen margin
}

// SwapPositionDataRequest holds the parameters of Query position data
type SwapPositionDataRequest struct {
	Symbol   currency.Pair
	Currency currency.Code // settlement currency, USDT or USDC; every settlement currency when empty
}

// SwapPositionDataResponse holds a perpetual position from Query position data
type SwapPositionDataResponse struct {
	Symbol             string        `json:"symbol"`
	Currency           currency.Code `json:"currency"` // settlement currency
	PositionID         string        `json:"positionId"`
	PositionSide       PositionSide  `json:"positionSide"`
	Isolated           bool          `json:"isolated"` // true in isolated margin mode, false in cross margin mode
	PositionAmount     types.Number  `json:"positionAmt"`
	AvailableAmount    types.Number  `json:"availableAmt"`
	UnrealisedProfit   types.Number  `json:"unrealizedProfit"`
	RealisedProfit     types.Number  `json:"realisedProfit"`
	InitialMargin      types.Number  `json:"initialMargin"`
	Margin             types.Number  `json:"margin"`
	LiquidationPrice   float64       `json:"liquidationPrice"`
	AveragePrice       types.Number  `json:"avgPrice"` // average opening price
	Leverage           uint64        `json:"leverage"`
	PositionValue      types.Number  `json:"positionValue"`
	MarkPrice          types.Number  `json:"markPrice"`
	RiskRate           types.Number  `json:"riskRate"` // the position is liquidated or reduced when it reaches 100%
	MaxMarginReduction types.Number  `json:"maxMarginReduction"`
	PNLRatio           types.Number  `json:"pnlRatio"` // unrealised profit and loss ratio
	UpdateTime         types.Time    `json:"updateTime"`
}

// SwapFundFlowRequest holds the parameters of Get Account Profit and Loss Fund Flow
type SwapFundFlowRequest struct {
	Symbol     currency.Pair
	Currency   currency.Code // settlement currency, USDT or USDC; USDT when empty unless Symbol is a USDC contract
	IncomeType IncomeType    // every type when empty
	StartTime  time.Time
	EndTime    time.Time
	Limit      uint64 // 100 when zero, at most 1000
}

// SwapFundFlowResponse holds a fund flow record from Get Account Profit and Loss Fund Flow
type SwapFundFlowResponse struct {
	Symbol     string        `json:"symbol"`
	IncomeType IncomeType    `json:"incomeType"`
	Income     types.Number  `json:"income"` // positive for an inflow, negative for an outflow
	Asset      currency.Code `json:"asset"`
	Info       string        `json:"info"`
	Time       types.Time    `json:"time"`
	TransferID string        `json:"tranId"`
	TradeID    string        `json:"tradeId"` // the trade that caused the flow
}

// SwapTradingCommissionRateResponse holds the reply of Query Trading Commission Rate
type SwapTradingCommissionRateResponse struct {
	Commission *SwapCommissionRate `json:"commission"`
}

// SwapCommissionRate holds the account's perpetual taker and maker fee rates
type SwapCommissionRate struct {
	TakerCommissionRate float64 `json:"takerCommissionRate"`
	MakerCommissionRate float64 `json:"makerCommissionRate"`
}
