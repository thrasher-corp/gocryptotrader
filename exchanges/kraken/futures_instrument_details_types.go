package kraken

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FuturesInstrumentsRequest holds the parameters of Get instruments
type FuturesInstrumentsRequest struct {
	// ContractTypes limits the instruments to futures_inverse, futures_vanilla, flexible_futures, options or all.
	// Every futures type, which excludes options, is returned when it is empty
	ContractTypes []string
	// Expired returns only expired instruments rather than only listed ones
	Expired bool
}

// FuturesInstrumentsResponse holds the specifications of the listed markets, in no particular order
type FuturesInstrumentsResponse struct {
	Instruments []FuturesInstrument `json:"instruments"`
	ServerTime  time.Time           `json:"serverTime"`
}

// FuturesInstrument holds a market's specification. Kraken documents its tags as never populated, so they are left
// out, and documents OptionType, StrikePrice and RebateLevels for Get trading instruments only, but sends them here too
type FuturesInstrument struct {
	// MakerProtectionMilliseconds is how long an order that could take liquidity is held before matching, so resting
	// orders can react; it is 0 for a market without maker protection
	MakerProtectionMilliseconds uint64 `json:"makerProtectionMillis"`
	// Category is the market's category, such as Layer 1, DeFi or Forex
	Category     string  `json:"category"`
	ContractSize float64 `json:"contractSize"`
	// ContractValueTradePrecision is the number of decimal places an order size may have; a negative precision requires
	// a multiple of a power of ten, such as 10 for -1
	ContractValueTradePrecision int64   `json:"contractValueTradePrecision"`
	Description                 string  `json:"description"`
	FundingRateCoefficient      float64 `json:"fundingRateCoefficient"`
	// ImpactMidSize is the book depth used to calculate the mid price the mark price derives from
	ImpactMidSize   float64   `json:"impactMidSize"`
	ISIN            string    `json:"isin"`
	LastTradingTime time.Time `json:"lastTradingTime"`
	// MarginSchedules holds each platform's margin schedule, keyed by platform, such as europa or dlt
	MarginSchedules    map[string]FuturesMarginSchedule `json:"marginSchedules"`
	RetailMarginLevels []FuturesMarginLevel             `json:"retailMarginLevels"`
	// MarginLevels are the professional clients' margin levels
	MarginLevels []FuturesMarginLevel `json:"marginLevels"`
	// MaxPositionSize is the largest position an account may hold, in contracts
	MaxPositionSize float64 `json:"maxPositionSize"`
	// MaxOpenInterestUSD caps the market's open interest, and is 0 for an uncapped market
	MaxOpenInterestUSD float64 `json:"maxOpenInterestUsd"`
	// MaxOpenInterestShare is the largest share of the market's open interest one user, with every subaccount, may hold
	// on either side
	MaxOpenInterestShare float64 `json:"maxOpenInterestShare"`
	// MaxRelativeFundingRate caps the absolute value of a perpetual's relative funding rate, and
	// MinRelativeFundingRate is a pre-IPO perpetual's funding rate floor, which may be negative; equal rates fix the
	// funding rate
	MaxRelativeFundingRate float64   `json:"maxRelativeFundingRate"`
	MinRelativeFundingRate float64   `json:"minRelativeFundingRate"`
	OpeningDate            time.Time `json:"openingDate"`
	PostOnly               bool      `json:"postOnly"`
	// FeeScheduleUID has not identified the schedule fees are charged by since 22 June 2026; GetTradeVolume returns
	// the fee rates charged
	FeeScheduleUID string `json:"feeScheduleUid"`
	Symbol         string `json:"symbol"`
	// Pair is the market's base and quote, such as BTC:USD
	Pair     string        `json:"pair"`
	Base     currency.Code `json:"base"`
	Quote    currency.Code `json:"quote"`
	TickSize float64       `json:"tickSize"`
	// Tradeable is true for every instrument returned, expired ones included
	Tradeable bool `json:"tradeable"`
	// Type is flexible_futures, futures_inverse, futures_vanilla or options
	Type string `json:"type"`
	// Underlying is the index a single-collateral market tracks
	Underlying string `json:"underlying"`
	// UnderlyingFuture is the future an option is written on
	UnderlyingFuture string `json:"underlyingFuture"`
	// TraditionalFinance is true for a non-crypto market
	TraditionalFinance bool `json:"tradfi"`
	// MTF is true while the market has Multilateral Trading Facility status
	MTF                bool     `json:"mtf"`
	PlatformsPermitted []string `json:"platformsPermitted"`
	// CountriesBanned holds the ISO 3166-1 alpha-2 codes of the countries banned from trading the market
	CountriesBanned []string `json:"countriesBanned"`
	IsExpired       bool     `json:"isExpired"`
	// OptionType is call or put
	OptionType  string  `json:"optionType"`
	StrikePrice float64 `json:"strikePrice"`
	// RebateLevels maps market share levels to rebate percentages, both decimals
	RebateLevels map[string]types.Number `json:"rebateLevels"`
}

// FuturesMarginSchedule holds a platform's margin levels for retail and professional clients
type FuturesMarginSchedule struct {
	Retail       []FuturesMarginLevel `json:"retail"`
	Professional []FuturesMarginLevel `json:"professional"`
}

// FuturesMarginLevel holds the initial and maintenance margin rates that apply from a position size, in Contracts for a
// single-collateral market or in NonContractUnits, such as the quote currency of a linear future, for a
// multi-collateral one
type FuturesMarginLevel struct {
	Contracts         uint64  `json:"contracts"`
	NonContractUnits  float64 `json:"numNonContractUnits"`
	InitialMargin     float64 `json:"initialMargin"`
	MaintenanceMargin float64 `json:"maintenanceMargin"`
}

// FuturesTradingInstrumentsRequest holds the parameters of Get trading instruments
type FuturesTradingInstrumentsRequest struct {
	// ContractTypes limits the instruments to futures_inverse, futures_vanilla, flexible_futures, options or all.
	// Every futures type, which excludes options, is returned when it is empty
	ContractTypes []string
}

// FuturesTradingInstrumentsResponse holds the specifications of the markets the account can access, in no particular
// order
type FuturesTradingInstrumentsResponse struct {
	Instruments []FuturesTradingInstrument `json:"instruments"`
	ServerTime  time.Time                  `json:"serverTime"`
}

// FuturesTradingInstrument holds the specification of a market the account can access. Kraken documents its
// minimumTradeSize as never populated, so it is left out
type FuturesTradingInstrument struct {
	// FundingRateCoefficient is set for a perpetual
	FundingRateCoefficient float64 `json:"fundingRateCoefficient"`
	// LastTradingTime is set for a fixed maturity market
	LastTradingTime time.Time `json:"lastTradingTime"`
	// ImpactMidSize is the book depth used to calculate the mid price the mark price derives from
	ImpactMidSize float64 `json:"impactMidSize"`
	// MaxPositionSize is the market-wide position size limit
	MaxPositionSize float64 `json:"maxPositionSize"`
	// MaxOpenInterestUSD caps the market's open interest, and is 0 for an uncapped market
	MaxOpenInterestUSD float64 `json:"maxOpenInterestUsd"`
	// MaxOpenInterestShare is the largest share of the market's open interest one user, with every subaccount, may hold
	// on either side
	MaxOpenInterestShare float64   `json:"maxOpenInterestShare"`
	OpeningDate          time.Time `json:"openingDate"`
	// MarginLevels is the margin schedule that applies to the account, for a futures market
	MarginLevels []FuturesMarginLevel `json:"marginLevels"`
	// MaxRelativeFundingRate caps the absolute value of a perpetual's relative funding rate, and
	// MinRelativeFundingRate is a pre-IPO perpetual's funding rate floor, which may be negative; equal rates fix the
	// funding rate
	MaxRelativeFundingRate float64 `json:"maxRelativeFundingRate"`
	MinRelativeFundingRate float64 `json:"minRelativeFundingRate"`
	Symbol                 string  `json:"symbol"`
	// Pair is the market's base and quote, such as BTC:USD
	Pair     string        `json:"pair"`
	Base     currency.Code `json:"base"`
	Quote    currency.Code `json:"quote"`
	TickSize float64       `json:"tickSize"`
	// Type is flexible_futures, futures_inverse, futures_vanilla or options
	Type string `json:"type"`
	// Underlying is the index a single-collateral market tracks
	Underlying string `json:"underlying"`
	ISIN       string `json:"isin"`
	// ContractValueTradePrecision is the number of decimal places an order size may have; a negative precision requires
	// a multiple of a power of ten, such as 10 for -1
	ContractValueTradePrecision int64  `json:"contractValueTradePrecision"`
	Description                 string `json:"description"`
	// MakerProtectionMilliseconds is how long an order that could take liquidity is held before matching, so resting
	// orders can react; it is 0 for a market without maker protection
	MakerProtectionMilliseconds uint64 `json:"makerProtectionMillis"`
	PostOnly                    bool   `json:"postOnly"`
	// FeeScheduleUID has not identified the schedule fees are charged by since 22 June 2026; GetTradeVolume returns
	// the fee rates charged
	FeeScheduleUID string `json:"feeScheduleUid"`
	// OptionType is call or put
	OptionType  string  `json:"optionType"`
	StrikePrice float64 `json:"strikePrice"`
	// UnderlyingFuture is the future an option is written on
	UnderlyingFuture string `json:"underlyingFuture"`
	// RebateLevels maps market share levels to rebate percentages, both decimals
	RebateLevels map[string]types.Number `json:"rebateLevels"`
	// MTF is true for a market provided under the Multilateral Trading Facility licence
	MTF bool `json:"mtf"`
	// TraditionalFinance is true for a non-crypto market
	TraditionalFinance bool `json:"tradfi"`
	// Restricted is true when the account may only place orders that reduce its position
	Restricted bool `json:"restricted"`
	IsExpired  bool `json:"isExpired"`
}

// FuturesInstrumentStatusListRequest holds the parameters of Get instrument status list
type FuturesInstrumentStatusListRequest struct {
	// ContractTypes limits the statuses to futures_inverse, futures_vanilla, flexible_futures, options or all. Every
	// futures type, which excludes options, is returned when it is empty
	ContractTypes []string
}

// FuturesInstrumentStatusListResponse holds the markets' price dislocation and volatility statuses
type FuturesInstrumentStatusListResponse struct {
	InstrumentStatus []FuturesInstrumentStatus `json:"instrumentStatus"`
	ServerTime       time.Time                 `json:"serverTime"`
}

// FuturesInstrumentStatusResponse holds a market's price dislocation and volatility status
type FuturesInstrumentStatusResponse struct {
	FuturesInstrumentStatus
	ServerTime time.Time `json:"serverTime"`
}

// FuturesInstrumentStatus holds a market's price dislocation and volatility status
type FuturesInstrumentStatus struct {
	Symbol                  string `json:"tradeable"`
	ExperiencingDislocation bool   `json:"experiencingDislocation"`
	// PriceDislocationDirection is ABOVE_UPPER_BOUND or BELOW_LOWER_BOUND during a price dislocation
	PriceDislocationDirection     string `json:"priceDislocationDirection"`
	ExperiencingExtremeVolatility bool   `json:"experiencingExtremeVolatility"`
	// ExtremeVolatilityInitialMarginMultiplier is the factor the initial margin is multiplied by during extreme
	// volatility
	ExtremeVolatilityInitialMarginMultiplier uint64 `json:"extremeVolatilityInitialMarginMultiplier"`
}
