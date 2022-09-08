package kucoin

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

type RiskLimitInfo struct {
	Symbol string `json:"symbol"`
	Level  int64  `json:"symbol"`
	Level  int64  `json:"symbol"`
	Level  int64  `json:"symbol"`
}

/*
   "symbol": "ADAUSDTM",
    "level": 1,
    "maxRiskLimit": 500,
    "minRiskLimit": 0,
    "maxLeverage": 20,
    "initialMarginRate": 0.05,
    "maintenanceMarginRate": 0.025
*/
