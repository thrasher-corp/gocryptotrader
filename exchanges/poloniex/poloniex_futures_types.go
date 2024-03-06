package poloniex

import (
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Contracts represents a list of open contract.
type Contracts struct {
	Code string         `json:"code"`
	Data []ContractItem `json:"data"`
}

// ContractItem represents a single open contract instance.
type ContractItem struct {
	Symbol                  string               `json:"symbol"`
	TakerFixFee             float64              `json:"takerFixFee"`
	NextFundingRateTime     convert.ExchangeTime `json:"nextFundingRateTime"`
	MakerFixFee             float64              `json:"makerFixFee"`
	Type                    string               `json:"type"`
	PredictedFundingFeeRate float64              `json:"predictedFundingFeeRate"`
	TurnoverOf24H           float64              `json:"turnoverOf24h"`
	InitialMargin           float64              `json:"initialMargin"`
	IsDeleverage            bool                 `json:"isDeleverage"`
	CreatedAt               convert.ExchangeTime `json:"createdAt"`
	FundingBaseSymbol       string               `json:"fundingBaseSymbol"`
	LowPriceOf24H           float64              `json:"lowPriceOf24h"`
	LastTradePrice          float64              `json:"lastTradePrice"`
	IndexPriceTickSize      float64              `json:"indexPriceTickSize"`
	FairMethod              string               `json:"fairMethod"`
	TakerFeeRate            float64              `json:"takerFeeRate"`
	Order                   int64                `json:"order"`
	UpdatedAt               convert.ExchangeTime `json:"updatedAt"`
	DisplaySettleCurrency   string               `json:"displaySettleCurrency"`
	IndexPrice              float64              `json:"indexPrice"`
	Multiplier              float64              `json:"multiplier"`
	MinOrderQty             float64              `json:"minOrderQty"`
	MaxLeverage             float64              `json:"maxLeverage"`
	FundingQuoteSymbol      string               `json:"fundingQuoteSymbol"`
	QuoteCurrency           string               `json:"quoteCurrency"`
	MaxOrderQty             float64              `json:"maxOrderQty"`
	MaxPrice                float64              `json:"maxPrice"`
	MaintainMargin          float64              `json:"maintainMargin"`
	Status                  string               `json:"status"`
	DisplayNameMap          struct {
		ContractNameKoKR string `json:"contractName_ko-KR"`
		ContractNameZhCN string `json:"contractName_zh-CN"`
		ContractNameEnUS string `json:"contractName_en-US"`
	} `json:"displayNameMap"`
	OpenInterest      string  `json:"openInterest"`
	HighPriceOf24H    float64 `json:"highPriceOf24h"`
	FundingFeeRate    float64 `json:"fundingFeeRate"`
	VolumeOf24H       float64 `json:"volumeOf24h"`
	RiskStep          float64 `json:"riskStep"`
	IsQuanto          bool    `json:"isQuanto"`
	MaxRiskLimit      float64 `json:"maxRiskLimit"`
	RootSymbol        string  `json:"rootSymbol"`
	BaseCurrency      string  `json:"baseCurrency"`
	FirstOpenDate     int64   `json:"firstOpenDate"`
	TickSize          float64 `json:"tickSize"`
	MarkMethod        string  `json:"markMethod"`
	IndexSymbol       string  `json:"indexSymbol"`
	MarkPrice         float64 `json:"markPrice"`
	MinRiskLimit      float64 `json:"minRiskLimit"`
	SettlementFixFee  float64 `json:"settlementFixFee"`
	SettlementSymbol  string  `json:"settlementSymbol"`
	PriceChgPctOf24H  float64 `json:"priceChgPctOf24h"`
	FundingRateSymbol string  `json:"fundingRateSymbol"`
	MakerFeeRate      float64 `json:"makerFeeRate"`
	IsInverse         bool    `json:"isInverse"`
	LotSize           float64 `json:"lotSize"`
	SettleCurrency    string  `json:"settleCurrency"`
	SettlementFeeRate float64 `json:"settlementFeeRate"`
}

// TickerInfo represents a ticker information for a single symbol.
type TickerInfo struct {
	Sequence     int64                `json:"sequence"`
	Symbol       string               `json:"symbol"`
	Side         string               `json:"side"`
	Size         float64              `json:"size"`
	Price        types.Number         `json:"price"`
	BestBidSize  float64              `json:"bestBidSize"`
	BestBidPrice types.Number         `json:"bestBidPrice"`
	BestAskSize  float64              `json:"bestAskSize"`
	BestAskPrice types.Number         `json:"bestAskPrice"`
	TradeID      string               `json:"tradeId"`
	Timestamp    convert.ExchangeTime `json:"ts"`
}
