package kucoin

import "time"

// Contract store contract details
type Contract struct {
	Symbol             string  `json:"symbol"`
	ContractType       string  `json:"type"`
	FirstOpenDate      string  `json:"firstOpenDate"`
	ExpireDate         string  `json:"expireDate"`
	SettleDate         string  `json:"settleDate"`
	BaseCurrency       string  `json:"baseCurrency"`
	QuoteCurrency      string  `json:"quoteCurrency"`
	SettleCurrency     string  `json:"settleCurrency"`
	MaxOrderQty        float64 `json:"maxOrderQty"`
	MaxPrice           float64 `json:"maxPrice"`
	LotSize            float64 `json:"lotSize"`
	TickSize           float64 `json:"tickSize"`
	IndexPriceTickSize float64 `json:"indexPriceTickSize"`
	Multiplier         float64 `json:"multiplier"`
	MakerFeeRate       float64 `json:"makerFeeRate"`
	TakerFeeRate       float64 `json:"takerFeeRate"`
	SettlementFeeRate  float64 `json:"settlementFeeRate"`
	IsInverse          bool    `json:"isInverse"`
	FundingBaseSymbol  string  `json:"fundingBaseSymbol"`
	FundingQuoteSymbol string  `json:"fundingQuoteSymbol"`
	FundingRateSymbol  string  `json:"fundingRateSymbol"`
	IndexSymbol        string  `json:"indexSymbol"`
	SettlementSymbol   string  `json:"settlementSymbol"`
	Status             string  `json:"status"`
}

// RiskLimitInfo store contract risk limit details
type RiskLimitInfo struct {
	Symbol                string  `json:"symbol"`
	Level                 int64   `json:"level"`
	MaxRiskLimit          int64   `json:"maxRiskLimit"`
	MinRiskLimit          int64   `json:"minRiskLimit"`
	MaxLeverage           float64 `json:"maxLeverage"`
	InitialMarginRate     float64 `json:"initialMarginRate"`
	MaintenanceMarginRate float64 `json:"maintenanceMarginRate"`
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
