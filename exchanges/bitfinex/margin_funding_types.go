package bitfinex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// FundingOfferFlagHidden keeps a funding offer out of the funding book; see https://docs.bitfinex.com/docs/flag-values
const FundingOfferFlagHidden uint64 = 1 << 6

// Keep funding types, telling KeepFunding which funding taken it names
const (
	// KeepFundingTypeCredit names funding used in positions, as GetFundingCredits returns it
	KeepFundingTypeCredit = "credit"
	// KeepFundingTypeLoan names funding not used in positions, as GetFundingLoans returns it
	KeepFundingTypeLoan = "loan"
)

// SubmitFundingOfferRequest holds a funding offer to submit
type SubmitFundingOfferRequest struct {
	// Type is one of the funding offer types
	Type string
	// Symbol is a funding currency, such as fUSD
	Symbol string
	// Amount is positive to provide funding and negative to take it
	Amount float64
	// Rate is daily; 0.01 is one percent. For the flash return rate types it is the offset from the flash return rate,
	// which only FundingOfferTypeFlashReturnRateDeltaFixed takes below zero; zero offers at the flash return rate
	Rate float64
	// Period is in days, from 2 to 120
	Period uint64
	// Flags is the sum of the offer's flags, such as FundingOfferFlagHidden
	Flags uint64
}

// FundingAutoRenewRequest holds a currency's funding auto-renew setting
type FundingAutoRenewRequest struct {
	// Activate turns auto-renew on, and off when false
	Activate bool
	Currency currency.Code
	// Amount is the most auto-renewed, at least 50 USD's worth; zero renews the amount currently provided
	Amount float64
	// Rate is what the docs call the percentage rate to renew at; zero renews at the flash return rate
	Rate float64
	// Period is in days; zero renews for 2 days
	Period uint64
}

// FundingAutoRenewSettings is a currency's funding auto-renew settings, as FundingAutoRenew applies them
type FundingAutoRenewSettings struct {
	Currency currency.Code `tuple:"0"`
	// Period is in days
	Period uint64 `tuple:"1"`
	// Rate is daily, 0.01 being one percent; zero renews at the flash return rate
	Rate float64 `tuple:"2"`
	// Threshold is the most auto-renewed
	Threshold float64 `tuple:"3"`
}

// UnmarshalJSON decodes a funding auto-renew settings array
func (f *FundingAutoRenewSettings) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, f) }

// KeepFundingRequest holds the credits or loans whose funding to keep, or stop keeping
type KeepFundingRequest struct {
	// Type is KeepFundingTypeCredit or KeepFundingTypeLoan
	Type string
	// IDs toggles keeping each of these credits or loans
	IDs []uint64
	// Changes sets whether to keep each credit or loan by its ID, rather than toggling it
	Changes map[uint64]bool
}

// FundingOffersHistoryRequest holds the currency, window and limit to get past funding offers for, each optional
type FundingOffersHistoryRequest struct {
	// Symbol is a funding currency, such as fUSD; empty covers every currency
	Symbol string
	Start  time.Time
	End    time.Time
	// Limit caps the offers returned, 25 by default and at most 500
	Limit uint64
}

// FundingLoansHistoryRequest holds the currency, window and limit to get past funding loans for, each optional
type FundingLoansHistoryRequest struct {
	// Symbol is a funding currency, such as fUSD; empty covers every currency
	Symbol string
	Start  time.Time
	End    time.Time
	// Limit caps the loans returned, 25 by default and at most 500
	Limit uint64
}

// FundingCreditsHistoryRequest holds the currency, window and limit to get past funding credits for, each optional
type FundingCreditsHistoryRequest struct {
	// Symbol is a funding currency, such as fUSD; empty covers every currency
	Symbol string
	Start  time.Time
	End    time.Time
	// Limit caps the credits returned, 25 by default and at most 500
	Limit uint64
}

// AccountFundingTradesRequest holds the currency, window and limit to get the account's funding trades for, each
// optional
type AccountFundingTradesRequest struct {
	// Symbol is a funding currency, such as fUSD; empty covers every currency
	Symbol string
	Start  time.Time
	End    time.Time
	// Limit caps the trades returned
	Limit uint64
}

type submitFundingOfferBody struct {
	Type   string `json:"type"`
	Symbol string `json:"symbol"`
	Amount string `json:"amount"`
	// Rate is always sent, as zero offers at the flash return rate
	Rate   string `json:"rate"`
	Period uint64 `json:"period"`
	Flags  uint64 `json:"flags,omitempty"`
}

// fundingIDBody names a funding offer, loan or credit by its ID
type fundingIDBody struct {
	ID uint64 `json:"id"`
}

type cancelAllFundingOffersBody struct {
	Currency string `json:"currency,omitempty"`
}

type fundingAutoRenewBody struct {
	// Status is always sent, as 0 deactivates auto-renew
	Status   uint64 `json:"status"`
	Currency string `json:"currency"`
	Amount   string `json:"amount,omitempty"`
	Rate     string `json:"rate,omitempty"`
	Period   uint64 `json:"period,omitempty"`
}

type keepFundingBody struct {
	Type    string            `json:"type"`
	IDs     []uint64          `json:"id,omitempty"`
	Changes map[uint64]uint64 `json:"changes,omitempty"`
}
