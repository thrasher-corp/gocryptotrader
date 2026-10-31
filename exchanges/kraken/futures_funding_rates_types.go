package kraken

import "time"

// FuturesHistoricalFundingRatesResponse holds a perpetual's funding rates in ascending time order
type FuturesHistoricalFundingRatesResponse struct {
	Rates      []FuturesFundingRate `json:"rates"`
	ServerTime time.Time            `json:"serverTime"`
}

// FuturesFundingRate holds the funding rate of a period
type FuturesFundingRate struct {
	// Timestamp is the start of the period
	Timestamp   time.Time `json:"timestamp"`
	FundingRate float64   `json:"fundingRate"`
	// RelativeFundingRate is FundingRate as a fraction of the price
	RelativeFundingRate float64 `json:"relativeFundingRate"`
}
