package kucoin

import "time"

var (
	validGranularity = []string{
		"1", "5", "15", "30", "60", "120", "240", "480", "720", "1440", "10080",
	}
)

// Contract store contract details
type Contract struct {
	Symbol                  string             `json:"symbol"`
	RootSymbol              string             `json:"rootSymbol"`
	ContractType            string             `json:"type"`
	FirstOpenDate           kucoinTimeMilliSec `json:"firstOpenDate"`
	ExpireDate              kucoinTimeMilliSec `json:"expireDate"`
	SettleDate              kucoinTimeMilliSec `json:"settleDate"`
	BaseCurrency            string             `json:"baseCurrency"`
	QuoteCurrency           string             `json:"quoteCurrency"`
	SettleCurrency          string             `json:"settleCurrency"`
	MaxOrderQty             float64            `json:"maxOrderQty"`
	MaxPrice                float64            `json:"maxPrice"`
	LotSize                 float64            `json:"lotSize"`
	TickSize                float64            `json:"tickSize"`
	IndexPriceTickSize      float64            `json:"indexPriceTickSize"`
	Multiplier              float64            `json:"multiplier"`
	InitialMargin           float64            `json:"initialMargin"`
	MaintainMargin          float64            `json:"maintainMargin"`
	MaxRiskLimit            float64            `json:"maxRiskLimit"`
	MinRiskLimit            float64            `json:"minRiskLimit"`
	RiskStep                float64            `json:"riskStep"`
	MakerFeeRate            float64            `json:"makerFeeRate"`
	TakerFeeRate            float64            `json:"takerFeeRate"`
	TakerFixFee             float64            `json:"takerFixFee"`
	MakerFixFee             float64            `json:"makerFixFee"`
	SettlementFee           float64            `json:"settlementFee"`
	IsDeleverage            bool               `json:"isDeleverage"`
	IsQuanto                bool               `json:"isQuanto"`
	IsInverse               bool               `json:"isInverse"`
	MarkMethod              string             `json:"markMethod"`
	FairMethod              string             `json:"fairMethod"`
	FundingBaseSymbol       string             `json:"fundingBaseSymbol"`
	FundingQuoteSymbol      string             `json:"fundingQuoteSymbol"`
	FundingRateSymbol       string             `json:"fundingRateSymbol"`
	IndexSymbol             string             `json:"indexSymbol"`
	SettlementSymbol        string             `json:"settlementSymbol"`
	Status                  string             `json:"status"`
	FundingFeeRate          float64            `json:"fundingFeeRate"`
	PredictedFundingFeeRate float64            `json:"predictedFundingFeeRate"`
	OpenInterest            string             `json:"openInterest"`
	TurnoverOf24h           float64            `json:"turnoverOf24h"`
	VolumeOf24h             float64            `json:"volumeOf24h"`
	MarkPrice               float64            `json:"markPrice"`
	IndexPrice              float64            `json:"indexPrice"`
	LastTradePrice          float64            `json:"lastTradePrice"`
	NextFundingRateTime     float64            `json:"nextFundingRateTime"`
	MaxLeverage             float64            `json:"maxLeverage"`
	SourceExchanges         []string           `json:"sourceExchanges"`
	PremiumsSymbol1M        string             `json:"premiumsSymbol1M"`
	PremiumsSymbol8H        string             `json:"premiumsSymbol8H"`
	FundingBaseSymbol1M     string             `json:"fundingBaseSymbol1M"`
	FundingQuoteSymbol1M    string             `json:"fundingQuoteSymbol1M"`
	LowPrice                float64            `json:"lowPrice"`
	HighPrice               float64            `json:"highPrice"`
	PriceChgPct             float64            `json:"priceChgPct"`
	PriceChg                float64            `json:"priceChg"`
}

// FuturesTicker stores ticker data
type FuturesTicker struct {
	Sequence     int64             `json:"sequence"`
	Symbol       string            `json:"symbol"`
	Side         string            `json:"side"`
	Size         float64           `json:"size"`
	Price        float64           `json:"price,string"`
	BestBidSize  float64           `json:"bestBidSize"`
	BestBidPrice float64           `json:"bestBidPrice,string"`
	BestAskSize  float64           `json:"bestAskSize"`
	BestAskPrice float64           `json:"bestAskPrice,string"`
	TradeId      string            `json:"tradeId"`
	FilledTime   kucoinTimeNanoSec `json:"time"`
}

type futuresOrderbookResponse struct {
	Data struct {
		Asks     [][2]float64      `json:"asks"`
		Bids     [][2]float64      `json:"bids"`
		Time     kucoinTimeNanoSec `json:"ts"`
		Sequence int64             `json:"sequence"`
		Symbol   string            `json:"symbol"`
	} `json:"result"`
	Error
}

// FuturesTrade stores trade data
type FuturesTrade struct {
	Sequence     int64             `json:"sequence"`
	TradeID      string            `json:"tradeId"`
	TakerOrderId string            `json:"takerOrderId"`
	MakerOrderId string            `json:"makerOrderId"`
	Price        float64           `json:"price,string"`
	Size         float64           `json:"size"`
	Side         string            `json:"side"`
	FilledTime   kucoinTimeNanoSec `json:"ts"`
}

// FuturesInterestRate stores interest rate data
type FuturesInterestRate struct {
	Symbol      string             `json:"symbol"`
	TimePoint   kucoinTimeMilliSec `json:"timePoint"`
	Value       float64            `json:"value"`
	Granularity int64              `json:"granularity"`
}

// Decomposion stores decomposion data
type Decomposion struct {
	Exchange string  `json:"exchange"`
	Price    float64 `json:"price"`
	Weight   float64 `json:"weight"`
}

// FuturesIndex stores index data
type FuturesIndex struct {
	FuturesInterestRate
	DecomposionList []Decomposion `json:"decomposionList"`
}

// FuturesMarkPrice stores mark price data
type FuturesMarkPrice struct {
	FuturesInterestRate
	IndexPrice float64 `json:"indexPrice"`
}

// FuturesFundingRate stores funding rate data
type FuturesFundingRate struct {
	FuturesInterestRate
	PredictedValue float64 `json:"predictedValue"`
}

// FuturesKline stores kline data
type FuturesKline struct {
	StartTime time.Time
	Open      float64
	Close     float64
	High      float64
	Low       float64
	Volume    float64
}

type FuturesOrder struct {
	ID             string             `json:"id"`
	Symbol         string             `json:"symbol"`
	OrderType      string             `json:"type"`
	Side           string             `json:"side"`
	Price          string             `json:"price"`
	Size           float64            `json:"size"`
	Value          float64            `json:"value,string"`
	DealValue      float64            `json:"dealValue,string"`
	DealSize       float64            `json:"dealSize"`
	Stp            string             `json:"stp"`
	Stop           string             `json:"stop"`
	StopPriceType  string             `json:"stopPriceType"`
	StopTriggered  bool               `json:"stopTriggered"`
	StopPrice      float64            `json:"stopPrice"`
	TimeInForce    string             `json:"timeInForce"`
	PostOnly       bool               `json:"postOnly"`
	Hidden         bool               `json:"hidden"`
	Iceberg        bool               `json:"iceberg"`
	Leverage       float64            `json:"leverage,string"`
	ForceHold      bool               `json:"forceHold"`
	CloseOrder     bool               `json:"closeOrder"`
	VisibleSize    float64            `json:"visibleSize"`
	ClientOid      string             `json:"clientOid"`
	Remark         string             `json:"remark"`
	Tags           string             `json:"tags"`
	IsActive       bool               `json:"isActive"`
	CancelExist    bool               `json:"cancelExist"`
	CreatedAt      kucoinTimeMilliSec `json:"createdAt"`
	UpdatedAt      kucoinTimeMilliSec `json:"updatedAt"`
	EndAt          kucoinTimeMilliSec `json:"endAt"`
	OrderTime      kucoinTimeNanoSec  `json:"orderTime"`
	SettleCurrency string             `json:"settleCurrency"`
	Status         string             `json:"status"`
	FilledValue    float64            `json:"filledValue,string"`
	FilledSize     float64            `json:"filledSize"`
	ReduceOnly     bool               `json:"reduceOnly"`
}
