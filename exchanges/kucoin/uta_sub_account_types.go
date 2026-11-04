package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTASubAccountCurrencyAssetsRequest holds the parameters of GetUTASubAccountCurrencyAssets; every field is optional
type UTASubAccountCurrencyAssetsRequest struct {
	// UID selects one sub-account; every sub-account is returned when empty
	UID      string
	PageSize uint64
	// LastID is the ID of the previous page's last record; the latest records are returned when zero
	LastID uint64
}

// UTASubAccountCurrencyAssetsResponse holds sub-accounts' assets per account and currency
type UTASubAccountCurrencyAssetsResponse struct {
	Timestamp types.Time                `json:"ts"`
	UserList  []UTASubAccountUserAssets `json:"userList"`
}

// UTASubAccountUserAssets holds a sub-account's assets per account
type UTASubAccountUserAssets struct {
	UID      uint64                `json:"uid"`
	Accounts []UTASubAccountAssets `json:"accounts"`
}

// UTASubAccountAssets holds a sub-account's currency assets in one account
type UTASubAccountAssets struct {
	// AccountType is FUNDING, UNIFIED, FUTURES, ISOLATED, CROSS or SPOT
	AccountType string `json:"accountType"`
	// SubAccountType identifies an isolated margin account by its symbol, such as AAVE-USDT
	SubAccountType string                       `json:"subAccountType"`
	Currencies     []UTASubAccountCurrencyAsset `json:"currencies"`
}

// UTASubAccountCurrencyAsset holds a sub-account's asset in one currency; the fields an account type does not use are
// zero
type UTASubAccountCurrencyAsset struct {
	Currency           currency.Code `json:"currency"`
	Locked             types.Number  `json:"locked"`
	Available          types.Number  `json:"available"`
	Balance            types.Number  `json:"balance"`
	Equity             types.Number  `json:"equity"`
	Liability          types.Number  `json:"liability"`
	LiabilityInterest  types.Number  `json:"liabilityInterest"`
	LiabilityPrincipal types.Number  `json:"liabilityPrincipal"`
	UnrealisedPNL      types.Number  `json:"unrealisedPnl"`
}

// UTASubAccountListResponse holds a page of sub-accounts
type UTASubAccountListResponse struct {
	CurrentPage uint64          `json:"currentPage"`
	PageSize    uint64          `json:"pageSize"`
	TotalNumber uint64          `json:"totalNum"`
	TotalPage   uint64          `json:"totalPage"`
	Items       []UTASubAccount `json:"items"`
}

// UTASubAccount holds a sub-account's details
type UTASubAccount struct {
	UserID  string `json:"userId"`
	UID     uint64 `json:"uid"`
	SubName string `json:"subName"`
	// Status is 2 (enabled) or 3 (frozen)
	Status uint64 `json:"status"`
	// Type is 0 (normal), 1 (robot), 2 (novice) or 5 (hosted, for asset management)
	Type uint64 `json:"type"`
	// Access is All, Futures or Margin; it is All for types 0, 1, 2 and 5
	Access    string     `json:"access"`
	CreatedAt types.Time `json:"createdAt"`
	Remarks   string     `json:"remarks"`
	// TradeTypes are the trading permissions assigned to the sub-account: Spot, Futures or Margin
	TradeTypes []string `json:"tradeTypes"`
	// OpenedTradeTypes are the assigned trading permissions the sub-account has activated
	OpenedTradeTypes []string `json:"openedTradeTypes"`
	// HostedStatus is 0 to 5
	HostedStatus uint64 `json:"hostedStatus"`
}

// UTASubAccountAPIResponse holds the details of a sub-account API key
type UTASubAccountAPIResponse struct {
	SubName    string `json:"subName"`
	Remark     string `json:"remark"`
	APIKey     string `json:"apiKey"`
	APIVersion uint64 `json:"apiVersion"`
	// Permission is a comma separated list of General, Spot, Futures, Margin, Unified and InnerTransfer
	Permission string `json:"permission"`
	// IPWhitelist is a comma separated list of the IPs allowed to use the key
	IPWhitelist string     `json:"ipWhitelist"`
	CreatedAt   types.Time `json:"createdAt"`
	UID         uint64     `json:"uid"`
	IsMaster    bool       `json:"isMaster"`
}

// UTASubAccountTransferPermissionResponse holds whether a sub-account's API key can initiate SUB_TO_SUB transfers
type UTASubAccountTransferPermissionResponse struct {
	SubUID   string `json:"subUid"`
	SubToSub bool   `json:"subToSub"`
}

// AddUTASubAccountRequest holds the parameters of AddUTASubAccount
type AddUTASubAccountRequest struct {
	// Password is required: 7 to 24 letters and numbers, not only numbers, and no special characters
	Password string
	// Remarks is 1 to 24 characters
	Remarks string
	// SubName is required: 7 to 32 characters with at least one letter and one number, and no spaces. The sub-account
	// logs in with it
	SubName string
	// Access is required: a comma separated list of the case-sensitive permissions Spot, Futures and Margin
	Access string
	// Mode is CLASSIC or UNIFIED
	Mode string
}

// AddUTASubAccountResponse holds a new sub-account's details
type AddUTASubAccountResponse struct {
	UID     uint64 `json:"uid"`
	SubName string `json:"subName"`
	Remarks string `json:"remarks"`
	Access  string `json:"access"`
}

// AddUTASubAccountAPIRequest holds the parameters of AddUTASubAccountAPI
type AddUTASubAccountAPIRequest struct {
	// SubName is required
	SubName string
	// Passphrase is required: 7 to 32 characters without spaces
	Passphrase string
	// Remark is required: 1 to 24 characters
	Remark string
	// Permission is a comma separated list of General, Spot, Margin, Unified, Futures, InnerTransfer, Transfer and Earn
	Permission string
	// IPWhitelist is a comma separated list of up to 20 IPs
	IPWhitelist string
	// Expire is the key's lifetime in days: -1 (no expiry), 30, 90, 180 or 360
	Expire string
}

// AddUTASubAccountAPIResponse holds a new sub-account API key, including its secret
type AddUTASubAccountAPIResponse struct {
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

// ModifyUTASubAccountAPIRequest holds the parameters of ModifyUTASubAccountAPI
type ModifyUTASubAccountAPIRequest struct {
	// SubName is required
	SubName string
	// APIKey is required: the sub-account API key to modify
	APIKey string
	// Passphrase is required: the key's passphrase
	Passphrase string
	// Permission is a comma separated list of General, Spot, Futures, Unified, Margin and InnerTransfer
	Permission string
	// IPWhitelist is a comma separated list of up to 20 IPs
	IPWhitelist string
	// Expire is the key's lifetime in days: -1 (no expiry), 30, 90, 180 or 360
	Expire string
}

// ModifyUTASubAccountAPIResponse holds a sub-account API key's details after a change
type ModifyUTASubAccountAPIResponse struct {
	SubName     string `json:"subName"`
	APIKey      string `json:"apiKey"`
	Permission  string `json:"permission"`
	IPWhitelist string `json:"ipWhitelist"`
}

// SetUTASubAccountsAPIRateLimitRequest holds the API rate limits to set for sub-accounts
type SetUTASubAccountsAPIRateLimitRequest struct {
	Limits []UTASubAccountAPIRateLimit
}

// UTASubAccountAPIRateLimit is the API rate limit to set for a sub-account
type UTASubAccountAPIRateLimit struct {
	UID string
	// Rate is in requests per second
	Rate uint64
}

// SetUTASubAccountsAPIRateLimitResponse holds the outcome of each sub-account's API rate limit change
type SetUTASubAccountsAPIRateLimitResponse struct {
	Items []UTASubAccountAPIRateLimitResult `json:"items"`
}

// UTASubAccountAPIRateLimitResult holds the outcome of setting a sub-account's API rate limit. Code is 200000 when the
// limit was set; any other code means it was not, and Message says why
type UTASubAccountAPIRateLimitResult struct {
	UID string `json:"uid"`
	// Rate is in requests per second
	Rate    uint64 `json:"rate"`
	Code    string `json:"code"`
	Message string `json:"msg"`
}

// DeleteUTASubAccountAPIRequest holds the parameters of DeleteUTASubAccountAPI; every field is required
type DeleteUTASubAccountAPIRequest struct {
	APIKey  string
	SubName string
	// Passphrase is the API key's passphrase
	Passphrase string
}

// DeleteUTASubAccountAPIResponse holds a deleted sub-account API key's details
type DeleteUTASubAccountAPIResponse struct {
	SubName string `json:"subName"`
	APIKey  string `json:"apiKey"`
}
