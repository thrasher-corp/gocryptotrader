package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SpotReferralCustomerIDResponse is the customer ID a referred spot user set
type SpotReferralCustomerIDResponse struct {
	CustomerID string `json:"customerId"`
}

// ReferralCustomerEmail is the customer ID a partner set for a referred user's email
type ReferralCustomerEmail struct {
	CustomerID string `json:"customerId"`
	Email      string `json:"email"`
}

// ReferralCustomerEmailResponse is the customer ID a partner set for a referred user's email
type ReferralCustomerEmailResponse struct {
	CustomerID string `json:"customerId"`
	Email      string `json:"email"`
}

// SpotReferralNewUserResponse is whether a spot user is new to a partner's referral and whether rebates apply
type SpotReferralNewUserResponse struct {
	APIAgentCode  string `json:"apiAgentCode"`
	RebateWorking bool   `json:"rebateWorking"`
	IfNewUser     bool   `json:"ifNewUser"`
	ReferrerID    uint64 `json:"referrerId"`
}

// ReferralRebateRecord is a rebate the account earned from a spot trade
type ReferralRebateRecord struct {
	Income types.Number  `json:"income"`
	Asset  currency.Code `json:"asset"`
	Symbol string        `json:"symbol"`
	Time   types.Time    `json:"time"`
}

// PartnerRebateRecordRequest holds the parameters of Query Partner Rebate Recent Record
type PartnerRebateRecordRequest struct {
	CustomerID string
	// StartTime and EndTime are at most 7 days apart and both set or both empty
	StartTime time.Time
	EndTime   time.Time
	// Limit is required and at most 500
	Limit uint64
}

// PartnerRebateRecord is a rebate a partner earned from a referred user's spot trade
type PartnerRebateRecord struct {
	CustomerID      string        `json:"customerId"`
	Email           string        `json:"email"`
	Income          types.Number  `json:"income"`
	Asset           currency.Code `json:"asset"`
	Symbol          string        `json:"symbol"`
	Time            types.Time    `json:"time"`
	OrderID         uint64        `json:"orderId"`
	TradeID         uint64        `json:"tradeId"`
	DistributeTime  types.Time    `json:"distributeTime"`
	CommissionAsset currency.Code `json:"commissionAsset"`
	Commission      types.Number  `json:"commission"`
	ConvertPrice    types.Number  `json:"convertPrice"`
}

// FuturesReferralCustomerIDResponse is the customer ID a referred futures user set for a broker
type FuturesReferralCustomerIDResponse struct {
	BrokerID   string `json:"brokerId"`
	CustomerID string `json:"customerId"`
}

// ReferralCustomerEmailRequest holds the parameters of Get Futures Client Email Customized Id
type ReferralCustomerEmailRequest struct {
	// CustomerID and Email filter the results; at most one may be set
	CustomerID string
	Email      string
	Page       uint64
	// Limit is at most 1000 and defaults to 100
	Limit uint64
}

// ReferralRebateOverviewResponse is a broker's futures referral rebate overview
type ReferralRebateOverviewResponse struct {
	BrokerID                  string        `json:"brokerId"`
	NewTraderRebateCommission types.Number  `json:"newTraderRebateCommission"`
	OldTraderRebateCommission types.Number  `json:"oldTraderRebateCommission"`
	TotalTradeUser            uint64        `json:"totalTradeUser"`
	Unit                      currency.Code `json:"unit"`
	TotalTradeVolume          types.Number  `json:"totalTradeVol"`
	TotalRebateVolume         types.Number  `json:"totalRebateVol"`
	Time                      types.Time    `json:"time"`
}

// ReferralVolumeRequest holds the parameters the futures referral statistics share
type ReferralVolumeRequest struct {
	// CoinMargined selects COIN-M futures (type 2) instead of USDⓈ-M futures (type 1)
	CoinMargined bool
	StartTime    time.Time
	EndTime      time.Time
	// Limit is at most 1000 and defaults to 500
	Limit uint64
}

// ReferralRebateVolume is a broker's futures referral rebate volume at a time
type ReferralRebateVolume struct {
	Unit         currency.Code `json:"unit"`
	RebateVolume types.Number  `json:"rebateVol"`
	Time         types.Time    `json:"time"`
}

// ReferralTraderDetail is a referred trader's futures trade and rebate volume at a time
type ReferralTraderDetail struct {
	CustomerID   string        `json:"customerId"`
	Unit         currency.Code `json:"unit"`
	TradeVolume  types.Number  `json:"tradeVol"`
	RebateVolume types.Number  `json:"rebateVol"`
	Time         types.Time    `json:"time"`
}

// ReferralTraderNumber is the number of new and existing traders a broker referred at a time
type ReferralTraderNumber struct {
	NewTrader types.Number `json:"newTrader"`
	OldTrader types.Number `json:"oldTrader"`
	Time      types.Time   `json:"time"`
}

// ReferralTradeVolume is referred users' futures trade volume at a time
type ReferralTradeVolume struct {
	Unit        currency.Code `json:"unit"`
	TradeVolume types.Number  `json:"tradeVol"`
	Time        types.Time    `json:"time"`
}

// FuturesReferralNewUserResponse is whether a futures user is new to a broker's referral and whether rebates apply
type FuturesReferralNewUserResponse struct {
	BrokerID      string `json:"brokerId"`
	RebateWorking bool   `json:"rebateWorking"`
	IfNewUser     bool   `json:"ifNewUser"`
}

// PAPIClientNewUserStatusResponse holds whether a referred portfolio margin user is new to the broker
type PAPIClientNewUserStatusResponse struct {
	BrokerID      string `json:"brokerId"`
	RebateWorking bool   `json:"rebateWorking"`
	IfNewUser     bool   `json:"ifNewUser"`
}

// PAPICustomisedIDResponse holds a referred portfolio margin user's customised ID
type PAPICustomisedIDResponse struct {
	BrokerID   string `json:"brokerId"`
	CustomerID string `json:"customerId"`
}
