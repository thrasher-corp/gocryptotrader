package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FuturesFundingRateResponse holds a contract's current funding rate (Get Current Funding Rate)
type FuturesFundingRateResponse struct {
	// Symbol is the contract's funding rate symbol, such as .XBTUSDTMFPI8H
	Symbol      string `json:"symbol"`
	Granularity uint64 `json:"granularity"` // milliseconds
	// TimePoint is the previous cycle's settlement time
	TimePoint types.Time `json:"timePoint"`
	Value     float64    `json:"value"`
	// PredictedValue is documented but missing from live responses, so it can read zero
	PredictedValue float64 `json:"predictedValue"`
	// DailyInterestRate is sent but not documented
	DailyInterestRate float64 `json:"dailyInterestRate"`
	FundingRateCap    float64 `json:"fundingRateCap"`
	FundingRateFloor  float64 `json:"fundingRateFloor"`
	// Period is 1 when the funding fee is charged in the current cycle and 0 when it is deferred
	Period uint64 `json:"period"`
	// FundingTime is the next settlement time
	FundingTime types.Time `json:"fundingTime"`
	// LastTimeFundingRate is sent but not documented
	LastTimeFundingRate float64 `json:"lastTimeFundingRate"`
}

// FuturesPublicFundingHistory holds the funding rate a contract settled at one time point (Get Public Funding History)
type FuturesPublicFundingHistory struct {
	Symbol      string     `json:"symbol"`
	FundingRate float64    `json:"fundingRate"`
	Timepoint   types.Time `json:"timepoint"` // lower case p, unlike the other funding endpoints
}

// FuturesPrivateFundingHistoryResponse holds a page of the account's funding fee settlements (Get Private Funding
// History)
type FuturesPrivateFundingHistoryResponse struct {
	DataList []FuturesPrivateFundingHistory `json:"dataList"`
	HasMore  bool                           `json:"hasMore"`
}

// FuturesPrivateFundingHistory holds one funding fee settlement of a position
type FuturesPrivateFundingHistory struct {
	ID               uint64        `json:"id"`
	Symbol           string        `json:"symbol"`
	TimePoint        types.Time    `json:"timePoint"`
	FundingRate      float64       `json:"fundingRate"`
	MarkPrice        float64       `json:"markPrice"`
	PositionQuantity int64         `json:"positionQty"` // negative for a short position
	PositionCost     float64       `json:"positionCost"`
	Funding          float64       `json:"funding"` // positive when the account received the fee
	SettleCurrency   currency.Code `json:"settleCurrency"`
	// Context is JSON text holding the position's margin mode and side
	Context    string `json:"context"`
	MarginMode string `json:"marginMode"`
}
