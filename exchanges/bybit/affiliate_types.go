package bybit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// AffiliateUserListRequest holds the parameters of GetAffiliateUserList; every field is optional
type AffiliateUserListRequest struct {
	// Size is the page size, from 0 to 100
	Size   uint64
	Cursor string
	// NeedDeposit, Need30Day and Need365Day ask for the deposit, 30 day and 365 day trading figures
	NeedDeposit bool
	Need30Day   bool
	Need365Day  bool
	// StartDate and EndDate set the period of the volume figures; only their UTC dates are sent
	StartDate time.Time
	EndDate   time.Time
}

// AffiliateUserListResponse holds a page of an affiliate's users
type AffiliateUserListResponse struct {
	List           []AffiliateUser `json:"list"`
	NextPageCursor string          `json:"nextPageCursor"`
}

// AffiliateUser holds an affiliate's user. Volumes are in USDT and include derivatives, options and spot; trading
// volumes update at T+1
type AffiliateUser struct {
	UserID string `json:"userId"`
	// RegisterTime is the registration date, as YYYY-MM-DD
	RegisterTime string `json:"registerTime"`
	// Source is the referral code the user registered with
	Source  string `json:"source"`
	Remarks string `json:"remarks"`
	IsKYC   bool   `json:"isKyc"`
	// The 30Day and 365Day fields are only returned when requested; the DepositAmount fields need NeedDeposit
	TakerVolume30Day    types.Number `json:"takerVol30Day"`
	MakerVolume30Day    types.Number `json:"makerVol30Day"`
	TradeVolume30Day    types.Number `json:"tradeVol30Day"`
	DepositAmount30Day  types.Number `json:"depositAmount30Day"`
	TakerVolume365Day   types.Number `json:"takerVol365Day"`
	MakerVolume365Day   types.Number `json:"makerVol365Day"`
	TradeVolume365Day   types.Number `json:"tradeVol365Day"`
	DepositAmount365Day types.Number `json:"depositAmount365Day"`
	// TakerVolume, MakerVolume and TradeVolume cover StartDate to EndDate
	TakerVolume types.Number `json:"takerVol"`
	MakerVolume types.Number `json:"makerVol"`
	TradeVolume types.Number `json:"tradeVol"`
	// StartDate and EndDate are the requested period, as YYYY-MM-DD
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	// TradFiTradeVolume and CommissionsVolume are only returned for a requested period, for which the 30 and 365 day
	// TradFi volumes are 0
	TradFiTradeVolume       types.Number `json:"tradfiTradeVol"`
	TradFiTradeVolume30Day  types.Number `json:"tradfiTradeVol30Day"`
	TradFiTradeVolume365Day types.Number `json:"tradfiTradeVol365Day"`
	// CommissionsVolume, Commissions30Day and Commissions365Day hold the commission earned in each coin
	CommissionsVolume map[currency.Code]types.Number `json:"commissionsVol"`
	Commissions30Day  map[currency.Code]types.Number `json:"commissions30Day"`
	Commissions365Day map[currency.Code]types.Number `json:"commissions365Day"`
}

// AffiliateUserInfoRequest holds the parameters of GetAffiliateUserInfo
type AffiliateUserInfoRequest struct {
	// UID is the master account UID of the affiliate's client, and is required
	UID  string
	Coin currency.Code
	// Business is 1 derivatives, 2 spot, 3 ByFi, 4 USDC or 5 options
	Business string
}

// AffiliateUserInfoResponse holds the trading volume, deposits and commissions of an affiliate's user. Volumes are in
// USDT and include derivatives, options and spot; they cover the user's whole activity on Bybit, not only the
// activity that earns commission
type AffiliateUserInfoResponse struct {
	UID string `json:"uid"`
	// VIPLevel is the VIP tier, such as 0 (no VIP), 1 to 4 or 99 (VVIP)
	VIPLevel         string       `json:"vipLevel"`
	TakerVolume30Day types.Number `json:"takerVol30Day"`
	MakerVolume30Day types.Number `json:"makerVol30Day"`
	TradeVolume30Day types.Number `json:"tradeVol30Day"`
	// DepositAmount30Day updates every 5 minutes
	DepositAmount30Day types.Number `json:"depositAmount30Day"`
	// The 365Day fields cover the user's lifetime since becoming the affiliate's client
	TakerVolume365Day   types.Number `json:"takerVol365Day"`
	MakerVolume365Day   types.Number `json:"makerVol365Day"`
	TradeVolume365Day   types.Number `json:"tradeVol365Day"`
	DepositAmount365Day types.Number `json:"depositAmount365Day"`
	// TotalWalletBalance is the wallet balance range: 1 under 100 USDT, 2 from 100 to under 250 USDT, 3 from 250 to
	// under 500 USDT, or 4 at least 500 USDT
	TotalWalletBalance string `json:"totalWalletBalance"`
	// DepositUpdateTime and VolumeUpdateTime are kept as Bybit sends them: its docs describe UTC millisecond
	// timestamps, but its example response gives UTC date times such as 2026-02-04 00:00:00
	DepositUpdateTime string `json:"depositUpdateTime"`
	VolumeUpdateTime  string `json:"volUpdateTime"`
	// KYCLevel is 0 no KYC, 1 KYC level 1 or 2 KYC level 2; Bybit capitalises its key
	KYCLevel                uint64       `json:"KycLevel"`
	TradFiTradeVolume30Day  types.Number `json:"tradfiTradeVol30Day"`
	TradFiTradeVolume365Day types.Number `json:"tradfiTradeVol365Day"`
	// Commissions30Day and Commissions365Day hold the commission earned in each coin
	Commissions30Day   map[currency.Code]types.Number `json:"commissions30Day"`
	Commissions365Day  map[currency.Code]types.Number `json:"commissions365Day"`
	PaySendAmount30Day types.Number                   `json:"paySendAmount30Day"`
	// PayFTT and CardFTT are the first transaction amounts of Bybit Pay and Bybit Card
	PayFTT  types.Number `json:"payFtt"`
	CardFTT types.Number `json:"cardFtt"`
}

// AffiliateSubAffiliateListRequest holds the parameters of GetAffiliateSubAffiliateList; every field is optional
type AffiliateSubAffiliateListRequest struct {
	Cursor string
	// Size is the page size, from 0 to 100
	Size uint64
	// StartDate and EndDate are set together, no more than 3 months apart, and default to the previous day; only
	// their UTC dates are sent
	StartDate time.Time
	EndDate   time.Time
	// SubAffiliateID limits the result to one sub-affiliate
	SubAffiliateID uint64
}

// AffiliateSubAffiliateListResponse holds a page of an affiliate's sub-affiliates
type AffiliateSubAffiliateListResponse struct {
	List           []SubAffiliate `json:"list"`
	NextPageCursor string         `json:"nextPageCursor"`
}

// SubAffiliate holds a sub-affiliate and its commissions for the requested period
type SubAffiliate struct {
	SubAffiliateID string `json:"subAffId"`
	UserID         string `json:"userId"`
	Name           string `json:"name"`
	// Email is partially masked
	Email string `json:"email"`
	// CommissionsVolume holds the commission in each of BTC, ETH, MNT, USDC and USDT
	CommissionsVolume  map[currency.Code]types.Number `json:"commissionsVol"`
	CommissionsForUSDT types.Number                   `json:"commissionsForUsdt"`
	// BecameAffiliateTime, StartDate and EndDate are UTC
	BecameAffiliateTime types.DateTime `json:"becameAffTime"`
	StartDate           types.DateTime `json:"startDate"`
	EndDate             types.DateTime `json:"endDate"`
}
