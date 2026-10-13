package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// ApplyForFastWithdrawalRequest holds the Apply for Fast Withdrawal parameters; Currency, Amount and Address are
// required. The first request returns the 2FA factors to verify; repeating it with Validation set completes the
// withdrawal
type ApplyForFastWithdrawalRequest struct {
	Currency currency.Code
	Amount   float64
	Address  string
	// Memo is the address remark (memo or tag); the API requires the key, so it is sent even when empty
	Memo string
	// Remark is sent even when empty, as the API requires the key
	Remark string
	// Chain is the currency's chain ID
	Chain string
	// WithdrawType is ADDRESS, MAIL, PHONE or UID
	WithdrawType string
	// FeeDeductType is EXTERNAL or INTERNAL
	FeeDeductType string
	IsInner       bool
	Wallet        string
	// TIN is the receiver's tax identification number
	TIN            string
	Country        string
	FirstName      string
	LastName       string
	IdentityNumber string
	IdentityType   string
	CompanyName    string
	// ReceiverType is COMPANY or INDIVIDUAL
	ReceiverType     string
	Questionnaire    string
	ReasonOfTransfer string
	// Validation carries the verified 2FA factors on the request that completes the withdrawal
	Validation *FastWithdrawalValidation
}

// FastWithdrawalValidation holds the 2FA factors a user verified for a fast withdrawal, and the transaction ID the
// first Apply for Fast Withdrawal request returned
type FastWithdrawalValidation struct {
	Factors       []FastWithdrawalFactor
	TransactionID string
}

// FastWithdrawalFactor is one verified 2FA factor of a fast withdrawal
type FastWithdrawalFactor struct {
	// FactorType is EMV, GAV or SMS
	FactorType  string
	FactorValue string
}

// ApplyForFastWithdrawalResponse is the outcome of a fast withdrawal request: the 2FA factors to verify and the
// transaction ID they belong to, or the withdrawal ID once they are verified
type ApplyForFastWithdrawalResponse struct {
	// AllFactors lists the factors that need to be validated
	AllFactors [][]string `json:"allFactors"`
	// SentFactors lists the factors whose codes were sent to the user
	SentFactors   [][]string `json:"sentFactors"`
	WithdrawalID  string     `json:"withdrawalId"`
	TransactionID string     `json:"transactionId"`
}

// GetBrokerProRebateRequest holds the Broker Pro Get Broker Rebate parameters, all of them required
type GetBrokerProRebateRequest struct {
	// Begin and End are sent as UTC days and may be at most six months apart
	Begin time.Time
	End   time.Time
	// TradeType is SPOT or FUTURE
	TradeType string
}

// BrokerRebateDownloadResponse holds the link to a broker's rebate orders CSV file, which is valid for one day
type BrokerRebateDownloadResponse struct {
	URL string `json:"url"`
}

// GetBrokerProCommissionRequest holds the optional Get Commission parameters
type GetBrokerProCommissionRequest struct {
	// SiteType is the site the commission originates from; the API defaults to all
	SiteType string
	// TradeType is all, SPOT or FUTURE
	TradeType string
	// RebateType is 1 or 2, which the docs describe as spot and futures here but as tagged and untagged commission
	// in the response
	RebateType string
	StartAt    time.Time
	EndAt      time.Time
	Page       uint64
	PageSize   uint64
}

// BrokerProCommissionResponse holds a page of Broker Pro commission payouts
type BrokerProCommissionResponse struct {
	CurrentPage uint64                `json:"currentPage"`
	PageSize    uint64                `json:"pageSize"`
	TotalNumber uint64                `json:"totalNum"`
	TotalPage   uint64                `json:"totalPage"`
	Items       []BrokerProCommission `json:"items"`
}

// BrokerProCommission is the commission of one calculation period
type BrokerProCommission struct {
	SiteType string `json:"siteType"`
	// RebateType is 0 (all), 1 (tagged commission) or 2 (untagged commission)
	RebateType      uint64     `json:"rebateType"`
	PayoutTime      types.Time `json:"payoutTime"`
	PeriodStartTime types.Time `json:"periodStartTime"`
	PeriodEndTime   types.Time `json:"periodEndTime"`
	// Status is the payout status: 0 (pending), 1 (in progress), 2 (completed) or 3 (not paid)
	Status           uint64        `json:"status"`
	TotalTradeUser   types.Number  `json:"totalTradeUser"`
	TagUser          types.Number  `json:"tagUser"`
	TagTradeVolume   types.Number  `json:"tagTradeVolume"`
	TagCommission    types.Number  `json:"tagCommission"`
	InvitedUser      types.Number  `json:"invitedUser"`
	NoTagTradeVolume types.Number  `json:"noTagTradeVolume"`
	NoTagCommission  types.Number  `json:"noTagCommission"`
	TotalCommission  types.Number  `json:"totalCommission"`
	Currency         currency.Code `json:"currency"`
}

// GetBrokerProUserListRequest holds the optional Get User List parameters
type GetBrokerProUserListRequest struct {
	// TradeType is all, SPOT or FUTURE
	TradeType string
	UID       string
	// ReferralCode is the referral code the user registered with the broker through
	ReferralCode string
	// Tag is the broker tag the user traded through
	Tag string
	// StartAt and EndAt bound the users' registration time
	StartAt  time.Time
	EndAt    time.Time
	Page     uint64
	PageSize uint64
}

// BrokerProUserListResponse holds a page of the users trading through a Broker Pro
type BrokerProUserListResponse struct {
	CurrentPage uint64          `json:"currentPage"`
	PageSize    uint64          `json:"pageSize"`
	TotalNumber uint64          `json:"totalNum"`
	TotalPage   uint64          `json:"totalPage"`
	Items       []BrokerProUser `json:"items"`
}

// BrokerProUser is a user trading through a Broker Pro, with the volume, fees and commission they contributed
type BrokerProUser struct {
	UID string `json:"uid"`
	// NickName defaults to the user's phone number or email
	NickName           string       `json:"nickName"`
	RegistrationTime   types.Time   `json:"registrationTime"`
	RateUp             bool         `json:"rateUp"`
	TotalCommission    types.Number `json:"totalCommission"`
	TotalTradingVolume types.Number `json:"totalTradingVolume"`
	TotalFee           types.Number `json:"totalFee"`
	InvitedByMe        bool         `json:"invitedByMe"`
	ReferralCode       string       `json:"rcode"`
	// Tags lists the broker tags the user traded through, separated by commas
	Tags                          string        `json:"tags"`
	SpotTradingVolumeWithTag      types.Number  `json:"spotTradingVolumeWithTag"`
	FutureTradingVolumeWithTag    types.Number  `json:"futureTradingVolumeWithTag"`
	TradingFeeWithTag             types.Number  `json:"tradingFeeWithTag"`
	CommissionWithTag             types.Number  `json:"commissionWithTag"`
	SpotTradingVolumeWithoutTag   types.Number  `json:"spotTradingVolumeWithoutTag"`
	FutureTradingVolumeWithoutTag types.Number  `json:"futureTradingVolumeWithoutTag"`
	TradingFeeWithoutTag          types.Number  `json:"tradingFeeWithoutTag"`
	CommissionWithoutTag          types.Number  `json:"commissionWithoutTag"`
	Currency                      currency.Code `json:"currency"`
	// FuturesTradingVolumeWithTag and FuturesTradingVolumeWithoutTag appear in the docs' example but not its schema
	FuturesTradingVolumeWithTag    types.Number `json:"futuresTradingVolumeWithTag"`
	FuturesTradingVolumeWithoutTag types.Number `json:"futuresTradingVolumeWithoutTag"`
	// RefferralCode keeps the API's spelling
	RefferralCode string `json:"refferralCode"`
	// CompletedKYC, CompletedFirstTrade and CompletedFirstDeposit are typed as strings in the docs' schema and as
	// JSON booleans in its example; types.Boolean decodes both. The same example sends null for the unset rcode, so a
	// flag KuCoin leaves unset decodes to nil rather than failing the whole page
	CompletedKYC        *types.Boolean `json:"completedKYC"`
	CompletedFirstTrade *types.Boolean `json:"completedFirstTrade"`
	// FirstTradeTime is a date formatted YYYY-MM-DD
	FirstTradeTime        string         `json:"firstTradeTime"`
	FirstTradeAmount      types.Number   `json:"firstTradeAmount"`
	CompletedFirstDeposit *types.Boolean `json:"completedFirstDeposit"`
	// FirstDepositTime is a date formatted YYYY-MM-DD
	FirstDepositTime   string       `json:"firstDepositTime"`
	FirstDepositAmount types.Number `json:"firstDepositAmount"`
	Country            string       `json:"country"`
}

// GetBrokerProUserTransactionsRequest holds the optional Get User Transactions parameters
type GetBrokerProUserTransactionsRequest struct {
	// TradeType is all, SPOT or FUTURE
	TradeType string
	UID       string
	StartAt   time.Time
	EndAt     time.Time
	// LastID is the LastID of the last transaction of the previous page; leave it empty for the first page
	LastID string
	// Direction is PRE (the previous page) or NEXT (the next page)
	Direction string
	PageSize  uint64
}

// BrokerProUserTransaction is a user's trading through a Broker Pro, with the volume, fees and commission it
// contributed
type BrokerProUserTransaction struct {
	TradeTime types.Time `json:"tradeTime"`
	UID       string     `json:"uid"`
	// NickName defaults to the user's phone number or email
	NickName                     string        `json:"nickName"`
	InvitedByMe                  bool          `json:"invitedByMe"`
	TotalCommission              types.Number  `json:"totalCommission"`
	TotalVolume                  types.Number  `json:"totalVolume"`
	TotalFee                     types.Number  `json:"totalFee"`
	ReferralCode                 string        `json:"rcode"`
	SpotTradingVolumeWithTag     types.Number  `json:"spotTradingVolumeWithTag"`
	FutureTradingVolumeWithTag   types.Number  `json:"futureTradingVolumeWithTag"`
	TradingFeeWithTag            types.Number  `json:"tradingFeeWithTag"`
	CommissionWithTag            types.Number  `json:"commissionWithTag"`
	SpotTradingVolumeWithNoTag   types.Number  `json:"spotTradingVolumeWithNoTag"`
	FutureTradingVolumeWithNoTag types.Number  `json:"futureTradingVolumeWithNoTag"`
	TradingFeeWithNoTag          types.Number  `json:"tradingFeeWithNoTag"`
	CommissionWithNoTag          types.Number  `json:"commissionWithNoTag"`
	Currency                     currency.Code `json:"currency"`
	// LastID is the offset ID that requests the page after this transaction
	LastID string `json:"lastId"`
	// The remaining fields appear in the docs' example but not its schema
	SpotTradingVolumeWithoutTag    types.Number `json:"spotTradingVolumeWithoutTag"`
	FuturesTradingVolumeWithoutTag types.Number `json:"futuresTradingVolumeWithoutTag"`
	FuturesTradingVolumeWithTag    types.Number `json:"futuresTradingVolumeWithTag"`
	TradingFeeWithoutTag           types.Number `json:"tradingFeeWithoutTag"`
	CommissionWithoutTag           types.Number `json:"commissionWithoutTag"`
}
