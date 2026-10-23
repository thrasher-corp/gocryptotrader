package gateio

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// LendingCurrencyDetailResponse holds a lending currency detail information
type LendingCurrencyDetailResponse struct {
	Currency          currency.Code `json:"currency"`
	MinLendAmount     types.Number  `json:"min_lend_amount"`
	MaxLendAmountUSDT types.Number  `json:"max_lend_amount"`
	MaxHourlyRate     types.Number  `json:"max_rate"`
	MinHourlyRate     types.Number  `json:"min_rate"`
}

// LendingOrRedemptionRequest holds a lending or redemption request parameters
type LendingOrRedemptionRequest struct {
	Currency      currency.Code `json:"currency"`
	Amount        types.Number  `json:"amount"`
	MinHourlyRate types.Number  `json:"min_rate,omitempty"` // Required for lending
	Type          string        `json:"type"`               // lend or redeem
}

// LendOrderDetailResponse represents a user's lend order item detail
type LendOrderDetailResponse struct {
	Currency           currency.Code `json:"currency"`
	CurrentAmount      types.Number  `json:"current_amount"`
	Amount             types.Number  `json:"amount"`
	LentAmount         types.Number  `json:"lent_amount"`
	FrozenAmount       types.Number  `json:"frozen_amount"`
	MinRate            types.Number  `json:"min_rate"`
	InterestStatus     string        `json:"interest_status"`
	ReinvestLeftAmount types.Number  `json:"reinvest_left_amount"`
	CreateTime         types.Time    `json:"create_time"`
	UpdateTime         types.Time    `json:"update_time"`
}

// LendingTransactionRecordsRequest holds lending transaction record filters
type LendingTransactionRecordsRequest struct {
	Currency currency.Code
	Page     uint64
	Limit    uint64
	From     time.Time
	To       time.Time
	Type     string // lend or redeem
}

// LendingTransactionRecordResponse holds a lending order transaction records
type LendingTransactionRecordResponse struct {
	Type             string        `json:"type"`
	Currency         currency.Code `json:"currency"`
	Amount           types.Number  `json:"amount"`
	LastWalletAmount types.Number  `json:"last_wallet_amount"`
	LastLentAmount   types.Number  `json:"last_lent_amount"`
	LastFrozenAmount types.Number  `json:"last_frozen_amount"`
	CreateTime       types.Time    `json:"create_time"`
}

// CurrencyAndInterestIncomeResponse holds a currency and interest income detail
type CurrencyAndInterestIncomeResponse struct {
	Currency currency.Code `json:"currency"`
	Interest types.Number  `json:"interest"`
}

// UserDividendRecordsRequest holds lending interest record filters
type UserDividendRecordsRequest struct {
	Currency currency.Code
	Page     uint64
	Limit    uint64
	From     time.Time
	To       time.Time
}

// UserDividendRecordsResponse holds user's dividend records
type UserDividendRecordsResponse struct {
	Status         uint8         `json:"status"` // 0 failure, 1 success
	Currency       currency.Code `json:"currency"`
	ActualRate     types.Number  `json:"actual_rate"`
	Interest       types.Number  `json:"interest"`
	InterestStatus string        `json:"interest_status"`
	CreateTime     types.Time    `json:"create_time"`
}

// CurrencyInterestStatusResponse holds a currency and it's interest status of user's account
type CurrencyInterestStatusResponse struct {
	Currency       currency.Code `json:"currency"`
	InterestStatus string        `json:"interest_status"` // interest_dividend or interest_reinvest
}

// UniLoanAssetResponse holds a uni loan asset annualised trend chart data
type UniLoanAssetResponse struct {
	Time  types.Time   `json:"time"`
	Value types.Number `json:"value"`
}

// CurrencyEstimatedAnnualInterestRateResponse holds user's account currency and estimated annualised interest create details
type CurrencyEstimatedAnnualInterestRateResponse struct {
	Currency                currency.Code `json:"currency"`
	EstimatedAnnualisedRate types.Number  `json:"est_rate"`
}
