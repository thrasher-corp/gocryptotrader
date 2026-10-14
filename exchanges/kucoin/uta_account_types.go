package kucoin

import (
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTAAccountOverviewResponse holds the account-level summary of a unified trading account's funds, valued in USD
type UTAAccountOverviewResponse struct {
	AccountType string `json:"accountType"`
	// RiskRatio is (maintenance margin + estimated liquidation fees) / adjusted equity
	RiskRatio types.Number `json:"riskRatio"`
	// Equity is undiscounted: the sum of each currency's equity at its index price
	Equity    types.Number `json:"equity"`
	Liability types.Number `json:"liability"`
	// AvailableMargin is the adjusted equity less the initial margin, and is the futures buying power
	AvailableMargin types.Number `json:"availableMargin"`
	// AdjustedEquity is the effective margin: equity after each collateral currency's collateral ratio discount
	AdjustedEquity    types.Number `json:"adjustedEquity"`
	InitialMargin     types.Number `json:"im"`
	MaintenanceMargin types.Number `json:"mm"`
}

// UTAAccountCurrencyAssetsResponse holds a unified trading account's assets per currency
type UTAAccountCurrencyAssetsResponse struct {
	AccountType string             `json:"accountType"`
	Timestamp   types.Time         `json:"ts"`
	Accounts    []UTAAccountAssets `json:"accounts"`
}

// UTAAccountAssets holds the currency assets of a unified trading account
type UTAAccountAssets struct {
	Currencies []UTACurrencyAsset `json:"currencies"`
}

// UTACurrencyAsset holds a unified trading account's asset in one currency. Its fields describe currency-level funds
// and are not a measure of futures buying power, which is UTAAccountOverviewResponse.AvailableMargin
type UTACurrencyAsset struct {
	Currency currency.Code `json:"currency"`
	// Equity is the balance plus unrealised PNL, including futures positions' unrealised PNL
	Equity types.Number `json:"equity"`
	// Hold is frozen by spot orders and the fees and taxes reserved for open orders
	Hold    types.Number `json:"hold"`
	Balance types.Number `json:"balance"`
	// Available is the equity less the hold
	Available types.Number `json:"available"`
	// Liability is abs(min(cross margin currency equity, 0)), across spot margin, futures and other products
	Liability types.Number `json:"liability"`
	// PotentialBorrow is reserved by open orders and becomes a liability when they fill
	PotentialBorrow types.Number `json:"potentialBorrow"`
	// CollateralStatus is the platform-level collateral status: 1 normal, 2 approaching the collateral cap, 3 cap
	// exceeded (still backing existing liabilities and positions, but not new orders)
	CollateralStatus string `json:"collateralStatus"`
}

// UTAClassicAccountCurrencyAssetsRequest holds the parameters of GetUTAClassicAccountCurrencyAssets
type UTAClassicAccountCurrencyAssetsRequest struct {
	// AccountType is required: FUNDING, SPOT, FUTURES, CROSS or ISOLATED
	AccountType string
	// AccountSubtype selects an isolated margin account by its symbol, such as AAVE-USDT
	AccountSubtype string
	// Currency limits the balances to one currency; every currency is returned when empty
	Currency currency.Code
}

// UTAClassicAccountCurrencyAssetsResponse holds a classic account type's assets per currency
type UTAClassicAccountCurrencyAssetsResponse struct {
	AccountType string                    `json:"accountType"`
	Timestamp   types.Time                `json:"ts"`
	Accounts    []UTAClassicAccountAssets `json:"accounts"`
}

// UTAClassicAccountAssets holds the currency assets of a classic account group
type UTAClassicAccountAssets struct {
	// AccountSubtype is the symbol of an isolated margin account, such as BTC-USDT. KuCoin's V1 docs spell it
	// accountSubType in places, which decodes into it as well
	AccountSubtype string `json:"accountSubtype"`
	// RiskRatio is returned for isolated margin accounts and, in the docs' example, futures accounts
	RiskRatio  types.Number              `json:"riskRatio"`
	Currencies []UTAClassicCurrencyAsset `json:"currencies"`
}

// UTAClassicCurrencyAsset holds a classic account's asset in one currency. The cross and isolated margin fields are only
// returned for futures accounts
type UTAClassicCurrencyAsset struct {
	Currency                 currency.Code `json:"currency"`
	Locked                   types.Number  `json:"locked"`
	Available                types.Number  `json:"available"`
	Balance                  types.Number  `json:"balance"`
	Equity                   types.Number  `json:"equity"`
	CrossOrderMargin         types.Number  `json:"crossOrderMargin"`
	CrossPositionMargin      types.Number  `json:"crossPosMargin"`
	CrossUnrealisedPNL       types.Number  `json:"crossUnPnl"`
	IsolatedFundingFeeMargin types.Number  `json:"isolatedFundingFeeMargin"`
	IsolatedOrderMargin      types.Number  `json:"isolatedOrderMargin"`
	IsolatedPositionMargin   types.Number  `json:"isolatedPosMargin"`
	IsolatedUnrealisedPNL    types.Number  `json:"isolatedUnPnl"`
	TotalCrossMargin         types.Number  `json:"totalCrossMargin"`
	Liability                types.Number  `json:"liability"`
}

// UTAAPIKeyInfoResponse holds the details of the API key that signed the request
type UTAAPIKeyInfoResponse struct {
	UID uint64 `json:"uid"`
	// ParentUID is the master account's UID, and is not returned for a master account's own key
	ParentUID uint64 `json:"parentUid"`
	// Region is the account's KYC region as a two-letter country code
	Region string `json:"region"`
	// KYCStatus is 0 or 1
	KYCStatus uint64 `json:"kycStatus"`
	// SubName is not returned for a master account
	SubName    string `json:"subName"`
	Remark     string `json:"remark"`
	APIKey     string `json:"apiKey"`
	APIVersion uint64 `json:"apiVersion"`
	// Permission is a comma separated list, such as General,Spot,Margin,Unified,Futures,InnerTransfer,Transfer,Earn
	Permission string `json:"permission"`
	// IPWhitelist is a comma separated list of the IPs allowed to use the key
	IPWhitelist string     `json:"ipWhitelist"`
	IsMaster    bool       `json:"isMaster"`
	CreatedAt   types.Time `json:"createdAt"`
	// ExpiredAt is zero when the key does not expire
	ExpiredAt types.Time `json:"expiredAt"`
	// ThirdPartyApp is empty when the key is not associated with a third-party app
	ThirdPartyApp string `json:"thirdPartyApp"`
	// SiteType is global, turkey, australia, europe or thailand
	SiteType string `json:"siteType"`
}

// UTAAccountModeResponse holds the account's mode and the UIDs of its unified and classic sub-accounts
type UTAAccountModeResponse struct {
	// SelfAccountMode is CLASSIC or UNIFIED
	SelfAccountMode   string   `json:"selfAccountMode"`
	UnifiedSubAccount []uint64 `json:"unifiedSubAccount"`
	ClassicSubAccount []uint64 `json:"classicSubAccount"`
}

// UTALeverageRequest holds the parameters of GetUTALeverage
type UTALeverageRequest struct {
	// Currency selects a cross margin currency when TradeType is MARGIN; every currency is returned when empty
	Currency currency.Code
	// Symbol selects a futures symbol when TradeType is FUTURES; every symbol is returned when empty
	Symbol string
	// TradeType is required: MARGIN or FUTURES
	TradeType string
	// MarginMode is CROSS or ISOLATED
	MarginMode string
}

// UTALeverage holds the leverage of a cross margin currency or a futures symbol
type UTALeverage struct {
	// Currency is returned for cross margin
	Currency currency.Code `json:"currency"`
	Leverage types.Number  `json:"leverage"`
	// MarginMode is CROSS or ISOLATE
	MarginMode string `json:"marginMode"`
	// Symbol is returned for futures
	Symbol string `json:"symbol"`
}

// ModifyUTAFuturesLeverageResponse holds the outcome of a futures leverage change
type ModifyUTAFuturesLeverageResponse struct {
	Code     string       `json:"code"`
	Leverage types.Number `json:"leverage"`
}

// ModifyUTACrossMarginLeverageResponse holds a cross margin currency's leverage after a change
type ModifyUTACrossMarginLeverageResponse struct {
	Currency currency.Code `json:"currency"`
	Leverage types.Number  `json:"leverage"`
}

// UTABorrowingRatesAndLimitsResponse holds a currency's borrowing rates, limits and reserved borrowing
type UTABorrowingRatesAndLimitsResponse struct {
	CurrentRateHourly types.Number `json:"currentRateHourly"`
	CurrentRateDaily  types.Number `json:"currentRateDaily"`
	// BorrowLimitTotal is shared by the master account and its sub-accounts
	BorrowLimitTotal types.Number `json:"borrowLimitTotal"`
	// BorrowLimitTotalHold is reserved across the master account and its sub-accounts
	BorrowLimitTotalHold types.Number `json:"borrowLimitTotalHold"`
	// BorrowLimitHold is reserved by this UID
	BorrowLimitHold types.Number `json:"borrowLimitHold"`
	// InterestFreeBorrowLimit is this UID's interest-free limit, which only covers liabilities from futures positions'
	// unrealised PNL
	InterestFreeBorrowLimit types.Number `json:"interestFreeBorrowLimit"`
}

// UTAAccountLedgerRequest holds the parameters of GetUTAAccountLedger
type UTAAccountLedgerRequest struct {
	// AccountType is required: FUNDING, SPOT, FUTURES, CROSS, ISOLATED or UNIFIED
	AccountType string
	// BusinessType filters the records, such as TRANSFER or TRADE_EXCHANGE
	BusinessType string
	Currency     currency.Code
	// Direction is IN or OUT
	Direction string
	StartAt   time.Time
	EndAt     time.Time
	// PageSize is at most 100 for FUTURES, 200 for UNIFIED and SPOT, and 500 for FUNDING, CROSS and ISOLATED
	PageSize uint64
	// LastID is the previous page's LastID; the latest records are returned when empty. It is text, as the ledger
	// entry IDs it refers to are hexadecimal for FUNDING
	LastID string
}

// UTAAccountLedgerResponse holds a page of account ledger entries
type UTAAccountLedgerResponse struct {
	// LastID is empty or zero when there are no further records
	LastID UTAID            `json:"lastId"`
	Items  []UTALedgerEntry `json:"items"`
}

// UTALedgerEntry holds an account ledger entry
type UTALedgerEntry struct {
	AccountType string `json:"accountType"`
	// ID is hexadecimal for FUNDING entries, and a bare number for SPOT ones in the docs' example
	ID           UTAID         `json:"id"`
	Currency     currency.Code `json:"currency"`
	Direction    string        `json:"direction"`
	BusinessType string        `json:"businessType"`
	Amount       types.Number  `json:"amount"`
	Balance      types.Number  `json:"balance"`
	Fee          types.Number  `json:"fee"`
	Tax          types.Number  `json:"tax"`
	Remark       string        `json:"remark"`
	Timestamp    types.Time    `json:"ts"`
}

// UTAID is an identifier KuCoin sends as a JSON string or as a bare JSON integer, such as account ledger entry IDs and
// cursors, and the trade IDs of the private websocket channels. The digits of a bare integer are kept as text, since
// the IDs pass the integers a float64 holds exactly
type UTAID string

// UnmarshalJSON decodes an identifier sent as a JSON string, a bare JSON integer or null
func (i *UTAID) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var id string
		if err := json.Unmarshal(data, &id); err != nil {
			return err
		}
		*i = UTAID(id)
		return nil
	}
	if string(data) == "null" {
		*i = ""
		return nil
	}
	if _, err := strconv.ParseUint(string(data), 10, 64); err != nil {
		return fmt.Errorf("%w: identifier %s", common.ErrMalformedData, data)
	}
	*i = UTAID(data)
	return nil
}

// UTAInterestHistoryRequest holds the parameters of GetUTAInterestHistory. The past 7 days are returned when no time
// range is given
type UTAInterestHistoryRequest struct {
	// AccountType is required: UNIFIED
	AccountType string
	Currency    currency.Code
	StartTime   time.Time
	EndTime     time.Time
	Page        uint64
	Size        uint64
}

// UTAInterestHistoryResponse holds a page of interest accrual records
type UTAInterestHistoryResponse struct {
	Items  []UTAInterestRecord `json:"items"`
	LastID uint64              `json:"lastId"`
}

// UTAInterestRecord holds an interest accrual record
type UTAInterestRecord struct {
	// Liability is the liability at the accrual time
	Liability          types.Number  `json:"liability"`
	Interest           types.Number  `json:"interest"`
	HourlyInterestRate types.Number  `json:"hourlyInterestRate"`
	Currency           currency.Code `json:"currency"`
	Timestamp          types.Time    `json:"ts"`
	// InterestFreeLiability is the interest-exempt part of the liability at the accrual time
	InterestFreeLiability types.Number `json:"interestFreeLiability"`
}

// UTAAPIRateLimitResponse holds the API rate limits of UIDs
type UTAAPIRateLimitResponse struct {
	List []UTAAPIRateLimit `json:"list"`
}

// UTAAPIRateLimit holds the API rate limit of a UID
type UTAAPIRateLimit struct {
	UID string `json:"uid"`
	// Rate is in requests per second
	Rate uint64 `json:"rate"`
}

// UTAAPIRateLimitCapResponse holds the master account's API rate limit quota and its allocation to sub-accounts, in
// requests per second
type UTAAPIRateLimitCapResponse struct {
	VIPLevel uint64 `json:"vipLevel"`
	// MainRateLimit is the allocated quota plus the remaining quota
	MainRateLimit uint64 `json:"mainRateLimit"`
	// SubRateLimit is the most a sub-account can be given
	SubRateLimit   uint64 `json:"subRateLimit"`
	AllocatedQuota uint64 `json:"allocatedQuota"`
	// RemainingQuota can still be allocated to sub-accounts, and is the master account's own rate limit
	RemainingQuota uint64 `json:"remainingQuota"`
	// DefaultQuota is given to new sub-accounts
	DefaultQuota uint64 `json:"defaultQuota"`
}

// UTAFeeRateResponse holds the actual fee rates of symbols
type UTAFeeRateResponse struct {
	TradeType string             `json:"tradeType"`
	List      []UTASymbolFeeRate `json:"list"`
}

// UTASymbolFeeRate holds a symbol's actual fee rates
type UTASymbolFeeRate struct {
	Symbol       string       `json:"symbol"`
	TakerFeeRate types.Number `json:"takerFeeRate"`
	MakerFeeRate types.Number `json:"makerFeeRate"`
}

// SetUTAKCSFeeDeductionResponse holds whether KCS fee deduction is enabled
type SetUTAKCSFeeDeductionResponse struct {
	Enabled bool `json:"enabled"`
}

// UTAThirdPartyCustodyCurrencyLimit holds the remaining custody quota of a settlement currency at a custodian
type UTAThirdPartyCustodyCurrencyLimit struct {
	Custodian string        `json:"custodian"`
	Currency  currency.Code `json:"currency"`
	// CustodyQuota is shared by every custodian sub-account under the master account
	CustodyQuota types.Number `json:"custodyQuota"`
}
