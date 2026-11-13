package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTAAffiliateInviteesRequest holds the parameters for GetUTAAffiliateInvitees
type UTAAffiliateInviteesRequest struct {
	// UserType is 1 for ordinary users, 2 for ordinary affiliates and 3 for multi-level affiliates
	UserType string
	// ReferralCode is the code the invitees registered with
	ReferralCode        string
	UIDs                []string
	RegistrationStartAt time.Time
	RegistrationEndAt   time.Time
	Page                uint64
	PageSize            uint64
}

// UTAAffiliateInviteesResponse holds a page of the affiliate's invitees
type UTAAffiliateInviteesResponse struct {
	CurrentPage uint64                `json:"currentPage"`
	PageSize    uint64                `json:"pageSize"`
	TotalNumber uint64                `json:"totalNum"`
	TotalPage   uint64                `json:"totalPage"`
	Items       []UTAAffiliateInvitee `json:"items"`
}

// UTAAffiliateInvitee holds an invitee's registration and the commission it brings
type UTAAffiliateInvitee struct {
	UID string `json:"uid"`
	// NickName falls back to the invitee's masked phone number or email
	NickName              string     `json:"nickName"`
	ReferralCode          string     `json:"referralCode"`
	Country               string     `json:"country"`
	RegistrationTime      types.Time `json:"registrationTime"`
	CompletedKYC          bool       `json:"completedKyc"`
	CompletedFirstDeposit bool       `json:"completedFirstDeposit"`
	CompletedFirstTrade   bool       `json:"completedFirstTrade"`
	// Past7DayFees, Past7DayCommission and TotalCommission are in Currency
	Past7DayFees       types.Number `json:"past7dFees"`
	Past7DayCommission types.Number `json:"past7dCommission"`
	TotalCommission    types.Number `json:"totalCommission"`
	// MyCommissionRate and CashbackRate are percentages, where 29 is 29%; the docs swap the descriptions of these two
	// fields and TotalCommission
	MyCommissionRate types.Number `json:"myCommissionRate"`
	CashbackRate     types.Number `json:"cashbackRate"`
	// Currency is USDT or USDC
	Currency currency.Code `json:"currency"`
}

// UTAAffiliateCommissionRequest holds the parameters for GetUTAAffiliateCommission
type UTAAffiliateCommissionRequest struct {
	// SiteType defaults to all
	SiteType string
	// RebateType is 1 for direct rebates and 2 for secondary rebates; 0 asks for both
	RebateType uint64
	// RebateStartAt and RebateEndAt bound the commission payout time to at most a year; RebateEndAt defaults to now
	RebateStartAt time.Time
	RebateEndAt   time.Time
	Page          uint64
	PageSize      uint64
	// DataType is trade, the default, for trading commission, or kumining
	DataType string
}

// UTAAffiliateCommissionResponse holds a page of the affiliate's commission. The docs type the data as an array, but
// their example and KuCoin's SDK show this page
type UTAAffiliateCommissionResponse struct {
	CurrentPage uint64                   `json:"currentPage"`
	PageSize    uint64                   `json:"pageSize"`
	TotalNumber uint64                   `json:"totalNum"`
	TotalPage   uint64                   `json:"totalPage"`
	Items       []UTAAffiliateCommission `json:"items"`
}

// UTAAffiliateCommission holds the commission of a calculation period
type UTAAffiliateCommission struct {
	SiteType string `json:"siteType"`
	// RebateType is 1 for a direct rebate and 2 for a secondary rebate
	RebateType uint64 `json:"rebateType"`
	// PayoutTime is when the commission is paid, a day after the period ends
	PayoutTime      types.Time `json:"payoutTime"`
	PeriodStartTime types.Time `json:"periodStartTime"`
	PeriodEndTime   types.Time `json:"periodEndTime"`
	// Status is 1 for pending, 2 for processing, 3 for completed and 4 for rejected
	Status         uint64       `json:"status"`
	TakerVolume    types.Number `json:"takerVolume"`
	MakerVolume    types.Number `json:"makerVolume"`
	KuminingVolume types.Number `json:"kuminingVolume"`
	Commission     types.Number `json:"commission"`
	// Currency, USDT or USDC, denominates the volumes and the commission
	Currency currency.Code `json:"currency"`
	// DataType is trade or kumining
	DataType string `json:"dataType"`
}

// UTAAffiliateTradeHistoryRequest holds the parameters for GetUTAAffiliateTradeHistory
type UTAAffiliateTradeHistoryRequest struct {
	// UID, the invitee's, is required
	UID string
	// TradeType is SPOT or, for futures, FEATURE, as the docs spell it
	TradeType    string
	TradeStartAt time.Time
	TradeEndAt   time.Time
	Page         uint64
	PageSize     uint64
}

// UTAAffiliateTradeHistoryResponse holds a page of an invitee's trades
type UTAAffiliateTradeHistoryResponse struct {
	CurrentPage uint64              `json:"currentPage"`
	PageSize    uint64              `json:"pageSize"`
	TotalNumber uint64              `json:"totalNum"`
	TotalPage   uint64              `json:"totalPage"`
	Items       []UTAAffiliateTrade `json:"items"`
}

// UTAAffiliateTrade holds an invitee's trade and the commission it earned
type UTAAffiliateTrade struct {
	TradeTime types.Time `json:"tradeTime"`
	TradeType string     `json:"tradeType"`
	// TradeCurrency is the traded currency, which the docs call the trading pair symbol
	TradeCurrency currency.Code `json:"tradeCurrency"`
	// TradeAmount is in TradeCurrency, and ConvertedTradeAmount and ConvertedFee in Currency
	TradeAmount          types.Number `json:"tradeAmount"`
	ConvertedTradeAmount types.Number `json:"tradeAmountU"`
	ConvertedFee         types.Number `json:"feeU"`
	Commission           types.Number `json:"commission"`
	// Currency is USDT or USDC
	Currency currency.Code `json:"currency"`
}

// UTAAffiliateTransactionsRequest holds the parameters for GetUTAAffiliateTransactions
type UTAAffiliateTransactionsRequest struct {
	UID string
	// TradeType is SPOT or FUTURES
	TradeType string
	// TradeStartAt and TradeEndAt are required
	TradeStartAt time.Time
	TradeEndAt   time.Time
	// LastID is the previous page's last LastID; the docs type it as an integer, but KuCoin returns it as a string
	LastID string
	// Direction is NEXT, the default, or PRE
	Direction string
	PageSize  uint64
}

// UTAAffiliateTransactionResponse holds an invitee's trade and the commission it earned
type UTAAffiliateTransactionResponse struct {
	UID       string     `json:"uid"`
	TradeTime types.Time `json:"tradeTime"`
	TradeType string     `json:"tradeType"`
	// TradeCurrency is the traded currency
	TradeCurrency currency.Code `json:"tradeCurrency"`
	// TradeAmount is in TradeCurrency, and ConvertedTradeAmount and ConvertedFee in Currency
	TradeAmount          types.Number `json:"tradeAmount"`
	ConvertedTradeAmount types.Number `json:"tradeAmountU"`
	ConvertedFee         types.Number `json:"feeU"`
	Commission           types.Number `json:"commission"`
	// Currency is USDT or USDC
	Currency currency.Code `json:"currency"`
	// LastID pages on from this transaction
	LastID string `json:"lastId"`
}

// UTAAffiliateKuminingRequest holds the parameters for GetUTAAffiliateKumining
type UTAAffiliateKuminingRequest struct {
	UIDs    []string
	StartAt time.Time
	EndAt   time.Time
	// LastID is the previous page's last LastID
	LastID string
	// Direction is NEXT or PRE
	Direction string
	PageSize  uint64
}

// UTAAffiliateKuminingResponse holds an invitee's KuMining purchase
type UTAAffiliateKuminingResponse struct {
	UID string `json:"uid"`
	// PayTime is the trade time
	PayTime   types.Time    `json:"payTime"`
	GoodsName string        `json:"goodsName"`
	Amount    types.Number  `json:"amount"`
	Currency  currency.Code `json:"currency"`
	// LastID pages on from this purchase
	LastID string `json:"lastId"`
}
