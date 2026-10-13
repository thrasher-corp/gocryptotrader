package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CreateSubUserRequest holds the details of a sub-account to create
type CreateSubUserRequest struct {
	SubName  string // 7 to 32 characters without spaces, with at least one letter and one digit
	Password string // 7 to 24 letters and digits, with at least one of each
	Remarks  string // 1 to 24 characters
	Access   string // Spot, Futures or Margin, combined with commas
	Mode     string // CLASSIC or UNIFIED
}

// CreateSubUserResponse holds a created sub-account
type CreateSubUserResponse struct {
	UID     uint64 `json:"uid"`
	SubName string `json:"subName"`
	Remarks string `json:"remarks"`
	Access  string `json:"access"`
}

// AllSubAccountsInfoV2Response holds a page of sub-accounts
type AllSubAccountsInfoV2Response struct {
	CurrentPage uint64              `json:"currentPage"`
	PageSize    uint64              `json:"pageSize"`
	TotalNumber uint64              `json:"totalNum"`
	TotalPage   uint64              `json:"totalPage"`
	Items       []SubAccountSummary `json:"items"`
}

// SubAccountSummary holds a sub-account
type SubAccountSummary struct {
	UserID           string     `json:"userId"`
	UID              uint64     `json:"uid"`
	SubName          string     `json:"subName"`
	Status           uint64     `json:"status"` // 2 enabled, 3 frozen
	Type             uint64     `json:"type"`
	Access           string     `json:"access"`
	CreatedAt        types.Time `json:"createdAt"`
	Remarks          string     `json:"remarks"`
	TradeTypes       []string   `json:"tradeTypes"`
	OpenedTradeTypes []string   `json:"openedTradeTypes"` // Permissions the sub-account has activated
	HostedStatus     string     `json:"hostedStatus"`
}

// SubAccountDetailBalanceRequest holds the sub-account and filters for GetSubAccountDetailBalance
type SubAccountDetailBalanceRequest struct {
	SubUserID         string
	IncludeBaseAmount bool // Include currencies without assets
	BaseCurrency      currency.Code
	BaseAmount        float64 // Only currencies with at least this balance
}

// SubAccountDetailBalanceResponse holds the funding, spot and margin balances of a sub-account
type SubAccountDetailBalanceResponse SubAccountBalance

// SubAccountBalance holds the funding, spot and margin balances of a sub-account
type SubAccountBalance struct {
	SubUserID       string                      `json:"subUserId"`
	SubName         string                      `json:"subName"`
	MainAccounts    []SubAccountCurrencyBalance `json:"mainAccounts"`   // Funding account
	TradeAccounts   []SubAccountCurrencyBalance `json:"tradeAccounts"`  // Spot account
	MarginAccounts  []SubAccountCurrencyBalance `json:"marginAccounts"` // Margin account
	TradeHFAccounts []string                    `json:"tradeHFAccounts"`
}

// SubAccountCurrencyBalance holds a currency's balance in a sub-account
type SubAccountCurrencyBalance struct {
	Currency          currency.Code `json:"currency"`
	Balance           types.Number  `json:"balance"`
	Available         types.Number  `json:"available"`
	Holds             types.Number  `json:"holds"`
	BaseCurrency      currency.Code `json:"baseCurrency"`
	BaseCurrencyPrice types.Number  `json:"baseCurrencyPrice"`
	BaseAmount        types.Number  `json:"baseAmount"`
	Tag               string        `json:"tag"`
}

// AllSubAccountsBalanceV2Response holds a page of sub-account balances
type AllSubAccountsBalanceV2Response struct {
	CurrentPage uint64              `json:"currentPage"`
	PageSize    uint64              `json:"pageSize"`
	TotalNumber uint64              `json:"totalNum"`
	TotalPage   uint64              `json:"totalPage"`
	Items       []SubAccountBalance `json:"items"`
}

// AllFuturesSubAccountBalancesResponse holds the futures accounts of the master and sub-accounts
type AllFuturesSubAccountBalancesResponse struct {
	Summary  FuturesSubAccountBalanceSummary `json:"summary"`
	Accounts []FuturesSubAccountBalance      `json:"accounts"`
}

// FuturesSubAccountBalanceSummary holds the totals of the futures accounts of the master and sub-accounts
type FuturesSubAccountBalanceSummary struct {
	AccountEquityTotal    float64       `json:"accountEquityTotal"`
	UnrealisedPNLTotal    float64       `json:"unrealisedPNLTotal"`
	MarginBalanceTotal    float64       `json:"marginBalanceTotal"`
	PositionMarginTotal   float64       `json:"positionMarginTotal"`
	OrderMarginTotal      float64       `json:"orderMarginTotal"`
	FrozenFundsTotal      float64       `json:"frozenFundsTotal"`
	AvailableBalanceTotal float64       `json:"availableBalanceTotal"`
	Currency              currency.Code `json:"currency"`
}

// FuturesSubAccountBalance holds the futures account of the master or a sub-account
type FuturesSubAccountBalance struct {
	AccountName      string        `json:"accountName"` // main for the master account
	AccountEquity    float64       `json:"accountEquity"`
	UnrealisedPNL    float64       `json:"unrealisedPNL"`
	MarginBalance    float64       `json:"marginBalance"`
	PositionMargin   float64       `json:"positionMargin"`
	OrderMargin      float64       `json:"orderMargin"`
	FrozenFunds      float64       `json:"frozenFunds"`
	AvailableBalance float64       `json:"availableBalance"`
	Currency         currency.Code `json:"currency"`
}

// SubAccountAPIKey holds a sub-account API key
type SubAccountAPIKey struct {
	SubName     string     `json:"subName"`
	Remark      string     `json:"remark"`
	APIKey      string     `json:"apiKey"`
	APIVersion  uint64     `json:"apiVersion"`
	Permission  string     `json:"permission"`  // Comma-separated, e.g. General,Spot,Futures
	IPWhitelist string     `json:"ipWhitelist"` // Comma-separated
	CreatedAt   types.Time `json:"createdAt"`
	UID         uint64     `json:"uid"`
	IsMaster    bool       `json:"isMaster"`
}

// CreateSubAccountAPIKeyRequest holds the details of a sub-account API key to create
type CreateSubAccountAPIKeyRequest struct {
	SubName     string
	Passphrase  string // 7 to 32 characters without spaces
	Remark      string // 1 to 24 characters
	Permission  string // General, Spot, Futures, Margin, Unified or InnerTransfer, combined with commas
	IPWhitelist string // Up to 20 IPs, combined with commas
	Expire      string // Days until the key expires: 30, 90, 180 or 360; -1 never expires
}

// CreateSubAccountAPIKeyResponse holds a created sub-account API key
type CreateSubAccountAPIKeyResponse struct {
	SubName     string     `json:"subName"`
	Remark      string     `json:"remark"`
	APIKey      string     `json:"apiKey"`
	APISecret   string     `json:"apiSecret"`
	APIVersion  uint64     `json:"apiVersion"`
	Passphrase  string     `json:"passphrase"`
	Permission  string     `json:"permission"`
	IPWhitelist string     `json:"ipWhitelist"`
	CreatedAt   types.Time `json:"createdAt"`
}

// ModifySubAccountAPIKeyRequest holds the sub-account API key to modify and its new settings
type ModifySubAccountAPIKeyRequest struct {
	SubName     string
	APIKey      string
	Passphrase  string
	Permission  string // General, Spot, Futures, Margin, Unified or InnerTransfer, combined with commas
	IPWhitelist string // Up to 20 IPs, combined with commas
	Expire      string // Days until the key expires: 30, 90, 180 or 360; -1 never expires
}

// ModifySubAccountAPIKeyResponse holds a modified sub-account API key
type ModifySubAccountAPIKeyResponse struct {
	SubName     string `json:"subName"`
	APIKey      string `json:"apiKey"`
	Permission  string `json:"permission"`
	IPWhitelist string `json:"ipWhitelist"`
}

// DeleteSubAccountAPIKeyRequest holds the sub-account API key to delete
type DeleteSubAccountAPIKeyRequest struct {
	SubName    string
	APIKey     string
	Passphrase string
}

// DeleteSubAccountAPIKeyResponse holds a deleted sub-account API key
type DeleteSubAccountAPIKeyResponse struct {
	SubName string `json:"subName"`
	APIKey  string `json:"apiKey"`
}
