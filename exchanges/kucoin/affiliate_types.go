package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AffiliateInviteesRequest holds the filters of GetAffiliateInvitees
type AffiliateInviteesRequest struct {
	// UserType is 1 (ordinary user), 2 (ordinary affiliate) or 3 (multi-level affiliate)
	UserType string
	// ReferralCode is the referral code the users registered with
	ReferralCode        string
	UIDs                []string
	RegistrationStartAt time.Time
	RegistrationEndAt   time.Time
	Page                uint64
	PageSize            uint64
}

// AffiliateInviteesResponse is a page of the users an affiliate invited
type AffiliateInviteesResponse struct {
	CurrentPage uint64             `json:"currentPage"`
	PageSize    uint64             `json:"pageSize"`
	TotalNumber uint64             `json:"totalNum"`
	TotalPage   uint64             `json:"totalPage"`
	Items       []AffiliateInvitee `json:"items"`
}

// AffiliateInvitee is a user an affiliate invited
type AffiliateInvitee struct {
	UID string `json:"uid"`
	// NickName is a masked phone number or email when the user has no nickname
	NickName              string       `json:"nickName"`
	ReferralCode          string       `json:"referralCode"`
	Country               string       `json:"country"`
	RegistrationTime      types.Time   `json:"registrationTime"`
	CompletedKYC          bool         `json:"completedKyc"`
	CompletedFirstDeposit bool         `json:"completedFirstDeposit"`
	CompletedFirstTrade   bool         `json:"completedFirstTrade"`
	Past7DayFees          types.Number `json:"past7dFees"`
	Past7DayCommission    types.Number `json:"past7dCommission"`
	TotalCommission       types.Number `json:"totalCommission"`
	// MyCommissionRate is the affiliate's commission rate in percent, where 70 is 70%
	MyCommissionRate types.Number `json:"myCommissionRate"`
	// CashbackRate is the invitee's cashback rate in percent, where 10 is 10%
	CashbackRate types.Number `json:"cashbackRate"`
	// Currency is the unit of the fees and commissions: USDT or USDC
	Currency currency.Code `json:"currency"`
}

// AffiliateCommissionRequest holds the filters of GetAffiliateCommission
type AffiliateCommissionRequest struct {
	// SiteType is the site the commission comes from; the API defaults it to all
	SiteType string
	// RebateType is 1 (direct) or 2 (secondary); 0, the API default, returns both
	RebateType    uint64
	RebateStartAt time.Time
	RebateEndAt   time.Time
	Page          uint64
	PageSize      uint64
	UserID        string
	// DataType is trade (the API default) or kumining
	DataType string
}

// AffiliateCommissionResponse is a page of an affiliate's commission payouts
type AffiliateCommissionResponse struct {
	CurrentPage uint64                `json:"currentPage"`
	PageSize    uint64                `json:"pageSize"`
	TotalNumber uint64                `json:"totalNum"`
	TotalPage   uint64                `json:"totalPage"`
	Items       []AffiliateCommission `json:"items"`
}

// AffiliateCommission is a commission payout of an affiliate
type AffiliateCommission struct {
	SiteType string `json:"siteType"`
	// RebateType is 1 (direct) or 2 (secondary)
	RebateType uint64 `json:"rebateType"`
	// PayoutTime is the T+1 settlement time; commission periods follow UTC+8 days
	PayoutTime      types.Time `json:"payoutTime"`
	PeriodStartTime types.Time `json:"periodStartTime"`
	PeriodEndTime   types.Time `json:"periodEndTime"`
	// Status is 1 (pending), 2 (processing), 3 (completed) or 4 (rejected)
	Status      uint64       `json:"status"`
	TakerVolume types.Number `json:"takerVolume"`
	MakerVolume types.Number `json:"makerVolume"`
	Commission  types.Number `json:"commission"`
	// Currency is the unit of the volumes and commission: USDT or USDC
	Currency       currency.Code `json:"currency"`
	KuminingVolume types.Number  `json:"kuminingVolume"`
	// DataType is trade or kumining
	DataType string `json:"dataType"`
}

// AffiliateTradeHistoryRequest holds the filters of GetAffiliateTradeHistory
type AffiliateTradeHistoryRequest struct {
	// UID is the invitee's UID
	UID string
	// TradeType is SPOT or FEATURE (futures)
	TradeType    string
	TradeStartAt time.Time
	TradeEndAt   time.Time
	Page         uint64
	PageSize     uint64
}

// AffiliateTradeHistoryResponse is a page of an invitee's trades
type AffiliateTradeHistoryResponse struct {
	CurrentPage uint64           `json:"currentPage"`
	PageSize    uint64           `json:"pageSize"`
	TotalNumber uint64           `json:"totalNum"`
	TotalPage   uint64           `json:"totalPage"`
	Items       []AffiliateTrade `json:"items"`
}

// AffiliateTrade is a trade of an invitee
type AffiliateTrade struct {
	TradeTime     types.Time    `json:"tradeTime"`
	TradeType     string        `json:"tradeType"`
	TradeCurrency currency.Code `json:"tradeCurrency"`
	TradeAmount   types.Number  `json:"tradeAmount"`
	// TradeAmountU and FeeU are converted into Currency, USDT or USDC
	TradeAmountU types.Number  `json:"tradeAmountU"`
	FeeU         types.Number  `json:"feeU"`
	Commission   types.Number  `json:"commission"`
	Currency     currency.Code `json:"currency"`
}

// AffiliateTransactionsRequest holds the filters of GetAffiliateTransactions
type AffiliateTransactionsRequest struct {
	// UID is the invitee's UID
	UID string
	// TradeType is SPOT or FEATURE (futures)
	TradeType    string
	TradeStartAt time.Time
	TradeEndAt   time.Time
	// LastID is the LastID of the last record of the previous page, empty for the first page. The docs type it as an
	// integer, but the IDs it carries are strings
	LastID string
	// Direction is PRE (previous page) or NEXT (next page, the API default)
	Direction string
	PageSize  uint64
}

// AffiliateTransaction is a trade of an invitee
type AffiliateTransaction struct {
	UID           string        `json:"uid"`
	TradeTime     types.Time    `json:"tradeTime"`
	TradeType     string        `json:"tradeType"`
	TradeCurrency currency.Code `json:"tradeCurrency"`
	TradeAmount   types.Number  `json:"tradeAmount"`
	// TradeAmountU and FeeU are converted into Currency, USDT or USDC
	TradeAmountU types.Number  `json:"tradeAmountU"`
	FeeU         types.Number  `json:"feeU"`
	Commission   types.Number  `json:"commission"`
	Currency     currency.Code `json:"currency"`
	LastID       string        `json:"lastId"`
}

// AffiliateKuminingRequest holds the filters of GetAffiliateKumining
type AffiliateKuminingRequest struct {
	UID     string
	StartAt time.Time
	EndAt   time.Time
	// LastID is the LastID of the last record of the previous page, empty for the first page
	LastID string
	// Direction is PRE (previous page) or NEXT (next page)
	Direction string
	PageSize  uint64
}

// AffiliateKumining is a KuMining purchase record
type AffiliateKumining struct {
	UID       string        `json:"uid"`
	GoodsName string        `json:"goodsName"`
	Amount    types.Number  `json:"amount"`
	Currency  currency.Code `json:"currency"`
	LastID    string        `json:"lastId"`
	PayTime   types.Time    `json:"payTime"`
}
