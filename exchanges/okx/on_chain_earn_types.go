package okx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// OnChainEarnOffersRequest holds the filters for on-chain earn offers
type OnChainEarnOffersRequest struct {
	ProductID    string
	ProtocolType string
	Currency     currency.Code
}

// OnChainEarnOfferResponse holds an on-chain earn product open to purchase
type OnChainEarnOfferResponse struct {
	Currency                 currency.Code                `json:"ccy"`
	ProductID                string                       `json:"productId"`
	Protocol                 string                       `json:"protocol"`
	ProtocolType             string                       `json:"protocolType"`
	Term                     types.Number                 `json:"term"`
	APY                      types.Number                 `json:"apy"`
	EarlyRedeem              bool                         `json:"earlyRedeem"`
	InvestData               []OnChainEarnOfferInvestment `json:"investData"`
	EarningData              []OnChainEarnOfferEarning    `json:"earningData"`
	State                    string                       `json:"state"`
	RedeemPeriod             []string                     `json:"redeemPeriod"`
	FastRedemptionDailyLimit types.Number                 `json:"fastRedemptionDailyLimit"`
}

// OnChainEarnOfferInvestment holds a currency an on-chain earn offer accepts and its subscription bounds
type OnChainEarnOfferInvestment struct {
	Currency      currency.Code `json:"ccy"`
	Balance       types.Number  `json:"bal"`
	MinimumAmount types.Number  `json:"minAmt"`
	MaximumAmount types.Number  `json:"maxAmt"`
}

// OnChainEarnOfferEarning holds a currency an on-chain earn offer pays earnings in
type OnChainEarnOfferEarning struct {
	Currency    currency.Code `json:"ccy"`
	EarningType string        `json:"earningType"`
}

// OnChainEarnPurchaseRequest holds an on-chain earn purchase
type OnChainEarnPurchaseRequest struct {
	ProductID  string                      `json:"productId"`
	InvestData []OnChainEarnCurrencyAmount `json:"investData"`
	Term       uint64                      `json:"term,string,omitempty"`
	Tag        string                      `json:"tag,omitempty"`
}

// OnChainEarnCurrencyAmount holds a currency and amount to invest in an on-chain earn product
type OnChainEarnCurrencyAmount struct {
	Currency currency.Code `json:"ccy"`
	Amount   float64       `json:"amt,string"`
}

// OnChainEarnOrderIDResponse holds the order an on-chain earn purchase, redemption or cancellation acted on
type OnChainEarnOrderIDResponse struct {
	OrderID string `json:"ordId"`
	Tag     string `json:"tag"`
}

// OnChainEarnRedeemRequest holds an on-chain earn redemption
type OnChainEarnRedeemRequest struct {
	OrderID          string `json:"ordId"`
	ProtocolType     string `json:"protocolType"`
	AllowEarlyRedeem bool   `json:"allowEarlyRedeem,omitempty"`
}

// OnChainEarnCancelRequest holds an on-chain earn purchase or redemption to cancel
type OnChainEarnCancelRequest struct {
	OrderID      string `json:"ordId"`
	ProtocolType string `json:"protocolType"`
}

// OnChainEarnActiveOrdersRequest holds the filters for active on-chain earn orders
type OnChainEarnActiveOrdersRequest struct {
	ProductID    string
	ProtocolType string
	Currency     currency.Code
	State        string
}

// OnChainEarnActiveOrderResponse holds an active on-chain earn order
type OnChainEarnActiveOrderResponse struct {
	Currency                 currency.Code               `json:"ccy"`
	OrderID                  string                      `json:"ordId"`
	ProductID                string                      `json:"productId"`
	State                    string                      `json:"state"`
	Protocol                 string                      `json:"protocol"`
	ProtocolType             string                      `json:"protocolType"`
	Term                     types.Number                `json:"term"`
	APY                      types.Number                `json:"apy"`
	InvestData               []OnChainEarnInvestment     `json:"investData"`
	EarningData              []OnChainEarnOrderEarning   `json:"earningData"`
	FastRedemptionData       []OnChainEarnFastRedemption `json:"fastRedemptionData"`
	PurchasedTime            types.Time                  `json:"purchasedTime"`
	EstimatedSettlementTime  types.Time                  `json:"estSettlementTime"`
	CancelRedemptionDeadline types.Time                  `json:"cancelRedemptionDeadline"`
	Tag                      string                      `json:"tag"`
}

// OnChainEarnInvestment holds a currency and amount an on-chain earn order invests
type OnChainEarnInvestment struct {
	Currency currency.Code `json:"ccy"`
	Amount   types.Number  `json:"amt"`
}

// OnChainEarnOrderEarning holds the earnings an active on-chain earn order accrues in one currency
type OnChainEarnOrderEarning struct {
	Currency    currency.Code `json:"ccy"`
	EarningType string        `json:"earningType"`
	Earnings    types.Number  `json:"earnings"`
}

// OnChainEarnFastRedemption holds the amount of one currency an on-chain earn order is fast redeeming
type OnChainEarnFastRedemption struct {
	Currency        currency.Code `json:"ccy"`
	RedeemingAmount types.Number  `json:"redeemingAmt"`
}

// OnChainEarnOrderHistoryRequest holds the filters and order ID cursors for on-chain earn order history
type OnChainEarnOrderHistoryRequest struct {
	ProductID    string
	ProtocolType string
	Currency     currency.Code
	// After returns orders earlier than this order ID
	After string
	// Before returns orders newer than this order ID
	Before string
	Limit  uint64
}

// OnChainEarnOrderHistoryResponse holds a completed on-chain earn order
type OnChainEarnOrderHistoryResponse struct {
	Currency      currency.Code                    `json:"ccy"`
	OrderID       string                           `json:"ordId"`
	ProductID     string                           `json:"productId"`
	State         string                           `json:"state"`
	Protocol      string                           `json:"protocol"`
	ProtocolType  string                           `json:"protocolType"`
	Term          types.Number                     `json:"term"`
	APY           types.Number                     `json:"apy"`
	InvestData    []OnChainEarnInvestment          `json:"investData"`
	EarningData   []OnChainEarnOrderHistoryEarning `json:"earningData"`
	PurchasedTime types.Time                       `json:"purchasedTime"`
	RedeemedTime  types.Time                       `json:"redeemedTime"`
	Tag           string                           `json:"tag"`
}

// OnChainEarnOrderHistoryEarning holds the earnings a completed on-chain earn order realised in one currency
type OnChainEarnOrderHistoryEarning struct {
	Currency         currency.Code `json:"ccy"`
	EarningType      string        `json:"earningType"`
	RealisedEarnings types.Number  `json:"realizedEarnings"`
}
