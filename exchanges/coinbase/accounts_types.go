package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Account types
const (
	AccountTypeCrypto           = "ACCOUNT_TYPE_CRYPTO"
	AccountTypeFiat             = "ACCOUNT_TYPE_FIAT"
	AccountTypeVault            = "ACCOUNT_TYPE_VAULT"
	AccountTypePerpetualFutures = "ACCOUNT_TYPE_PERP_FUTURES"
)

// Account platforms, which name the venue an account's balance trades on
const (
	// AccountPlatformConsumer holds spot balances
	AccountPlatformConsumer = "ACCOUNT_PLATFORM_CONSUMER"
	// AccountPlatformCFMConsumer holds US derivatives balances
	AccountPlatformCFMConsumer = "ACCOUNT_PLATFORM_CFM_CONSUMER"
	// AccountPlatformINTX holds International Exchange balances
	AccountPlatformINTX = "ACCOUNT_PLATFORM_INTX"
	// AccountPlatformUnspecified is the platform an account reports by default, which, as an empty platform, holds spot
	// balances
	AccountPlatformUnspecified = "ACCOUNT_PLATFORM_UNSPECIFIED"
)

// ListAccountsRequest selects a page of the accounts ListAccounts returns
type ListAccountsRequest struct {
	// Limit is how many accounts the page holds, at most 250; Coinbase holds 49 when it is unset
	Limit uint64
	// Cursor is a previous page's cursor, which the page starts after
	Cursor string
	// RetailPortfolioID selects a portfolio's accounts for a legacy API key; Coinbase deprecates it, as a CDP API key
	// only lists the accounts of the portfolio it is permissioned for
	RetailPortfolioID string
}

// AccountsResponse is a page of accounts
type AccountsResponse struct {
	Accounts []Account `json:"accounts"`
	// HasNext is set when more accounts follow, from the page after Cursor
	HasNext bool   `json:"has_next"`
	Cursor  string `json:"cursor"`
	// Size is how many accounts the page holds
	Size uint64 `json:"size"`
}

// AccountResponse holds the account GetAccount returns
type AccountResponse struct {
	Account *Account `json:"account"`
}

// Account is a portfolio's balance of a currency
type Account struct {
	UUID             string        `json:"uuid"`
	Name             string        `json:"name"`
	Currency         currency.Code `json:"currency"`
	AvailableBalance Amount        `json:"available_balance"`
	// Default is set for the user's primary account
	Default bool `json:"default"`
	// Active is set while the account can be used
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// DeletedAt is zero unless the account has been deleted
	DeletedAt time.Time `json:"deleted_at"`
	Type      string    `json:"type"`
	// Ready is set when the account can trade
	Ready bool `json:"ready"`
	// Hold is the amount held for pending transfers, which the available balance leaves out
	Hold              Amount `json:"hold"`
	RetailPortfolioID string `json:"retail_portfolio_id"`
	// Platform is the venue the account's balance trades on, spot, US derivatives or International Exchange
	Platform string `json:"platform"`
}
