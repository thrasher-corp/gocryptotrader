package okx

import (
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SubAccountListRequest holds the filters for GetSubAccountList; a nil Enabled lists normal and frozen sub-accounts
type SubAccountListRequest struct {
	Enabled        *bool
	SubAccountName string
	StartTime      time.Time
	EndTime        time.Time
	Limit          uint64
}

// SubAccountResponse holds a sub-account
type SubAccountResponse struct {
	Type                 string     `json:"type"`
	Enabled              bool       `json:"enable"`
	SubAccountName       string     `json:"subAcct"`
	UID                  string     `json:"uid"`
	Label                string     `json:"label"`
	MobileNumber         string     `json:"mobile"`
	GoogleAuthenticator  bool       `json:"gAuth"`
	FrozenFunctions      []string   `json:"frozenFunc"`
	CanTransferOut       bool       `json:"canTransOut"`
	CreationTime         types.Time `json:"ts"`
	SubAccountLevel      string     `json:"subAcctLv"`
	FirstLevelSubAccount string     `json:"firstLvSubAcct"`
	DirectMarketAccess   bool       `json:"ifDma"`
}

// CreateSubAccountRequest holds the parameters for CreateSubAccount
type CreateSubAccountRequest struct {
	SubAccountName string `json:"subAcct"`
	Type           string `json:"type"`
	Label          string `json:"label,omitempty"`
	Password       string `json:"pwd,omitempty"` // Required for KYB users only
}

// CreateSubAccountResponse holds a created sub-account
type CreateSubAccountResponse struct {
	SubAccountName string     `json:"subAcct"`
	Label          string     `json:"label"`
	UID            string     `json:"uid"`
	CreationTime   types.Time `json:"ts"`
}

// CreateSubAccountAPIKeyRequest holds the parameters for CreateSubAccountAPIKey
type CreateSubAccountAPIKeyRequest struct {
	SubAccountName string   `json:"subAcct"`
	Label          string   `json:"label"`
	Passphrase     string   `json:"passphrase"`
	Permissions    []string `json:"-"`
	IPAddresses    []string `json:"-"`
}

// MarshalJSON sends the permissions and IP addresses as the comma-separated lists OKX expects
func (r *CreateSubAccountAPIKeyRequest) MarshalJSON() ([]byte, error) {
	type Alias CreateSubAccountAPIKeyRequest
	return json.Marshal(&struct {
		*Alias
		Permissions string `json:"perm,omitempty"`
		IPAddresses string `json:"ip,omitempty"`
	}{
		Alias:       (*Alias)(r),
		Permissions: strings.Join(r.Permissions, ","),
		IPAddresses: strings.Join(r.IPAddresses, ","),
	})
}

// CreateSubAccountAPIKeyResponse holds a created sub-account API key and its secrets
type CreateSubAccountAPIKeyResponse struct {
	SubAccountName string     `json:"subAcct"`
	Label          string     `json:"label"`
	APIKey         string     `json:"apiKey"`
	SecretKey      string     `json:"secretKey"`
	Passphrase     string     `json:"passphrase"`
	Permissions    string     `json:"perm"` // Comma-separated
	IPAddresses    string     `json:"ip"`   // Comma-separated
	CreationTime   types.Time `json:"ts"`
}

// SubAccountAPIKeysRequest holds the parameters for GetSubAccountAPIKeys
type SubAccountAPIKeysRequest struct {
	SubAccountName string
	APIKey         string
}

// SubAccountAPIKeyResponse holds a sub-account API key
type SubAccountAPIKeyResponse struct {
	Label        string     `json:"label"`
	APIKey       string     `json:"apiKey"`
	Permissions  string     `json:"perm"` // Comma-separated
	IPAddresses  string     `json:"ip"`   // Comma-separated
	CreationTime types.Time `json:"ts"`
}

// ResetSubAccountAPIKeyRequest holds the parameters for ResetSubAccountAPIKey. OKX resets the label, permissions and
// IP addresses it is sent and keeps the rest; UnlinkIPAddresses sends an empty IP list, which unlinks every address
type ResetSubAccountAPIKeyRequest struct {
	SubAccountName    string   `json:"subAcct"`
	APIKey            string   `json:"apiKey"`
	Label             string   `json:"label,omitempty"`
	Permissions       []string `json:"-"`
	IPAddresses       []string `json:"-"`
	UnlinkIPAddresses bool     `json:"-"`
}

// MarshalJSON sends the permissions and IP addresses as the comma-separated lists OKX expects
func (r *ResetSubAccountAPIKeyRequest) MarshalJSON() ([]byte, error) {
	type Alias ResetSubAccountAPIKeyRequest
	body := &struct {
		*Alias
		Permissions string  `json:"perm,omitempty"`
		IPAddresses *string `json:"ip,omitempty"`
	}{
		Alias:       (*Alias)(r),
		Permissions: strings.Join(r.Permissions, ","),
	}
	if r.UnlinkIPAddresses || len(r.IPAddresses) > 0 {
		ipAddresses := strings.Join(r.IPAddresses, ",")
		body.IPAddresses = &ipAddresses
	}
	return json.Marshal(body)
}

// ResetSubAccountAPIKeyResponse holds a reset sub-account API key
type ResetSubAccountAPIKeyResponse struct {
	SubAccountName string     `json:"subAcct"`
	APIKey         string     `json:"apiKey"`
	Label          string     `json:"label"`
	Permissions    string     `json:"perm"` // Comma-separated
	IPAddresses    string     `json:"ip"`   // Comma-separated
	CreationTime   types.Time `json:"ts"`
}

// DeleteSubAccountAPIKeyRequest holds the parameters for DeleteSubAccountAPIKey
type DeleteSubAccountAPIKeyRequest struct {
	SubAccountName string `json:"subAcct"`
	APIKey         string `json:"apiKey"`
}

// SubAccountNameResponse holds a sub-account's name
type SubAccountNameResponse struct {
	SubAccountName string `json:"subAcct"`
}

// SubAccountTradingBalanceResponse holds a sub-account's trading account balance
type SubAccountTradingBalanceResponse struct {
	UpdateTime                   types.Time                `json:"uTime"`
	TotalEquity                  types.Number              `json:"totalEq"`
	IsolatedEquity               types.Number              `json:"isoEq"`
	AdjustedEquity               types.Number              `json:"adjEq"`
	AvailableEquity              types.Number              `json:"availEq"`
	OrderFrozen                  types.Number              `json:"ordFroz"`
	InitialMarginRequirement     types.Number              `json:"imr"`
	MaintenanceMarginRequirement types.Number              `json:"mmr"`
	BorrowFrozen                 types.Number              `json:"borrowFroz"`
	MarginRatio                  types.Number              `json:"mgnRatio"`
	NotionalUSD                  types.Number              `json:"notionalUsd"`
	NotionalUSDForBorrow         types.Number              `json:"notionalUsdForBorrow"`
	NotionalUSDForSwap           types.Number              `json:"notionalUsdForSwap"`
	NotionalUSDForFutures        types.Number              `json:"notionalUsdForFutures"`
	NotionalUSDForOption         types.Number              `json:"notionalUsdForOption"`
	UnrealisedPNL                types.Number              `json:"upl"`
	Delta                        types.Number              `json:"delta"`
	DeltaLeverage                types.Number              `json:"deltaLever"`
	DeltaNeutralStatus           string                    `json:"deltaNeutralStatus"`
	Details                      []SubAccountBalanceDetail `json:"details"`
}

// SubAccountBalanceDetail holds a currency's balance in a sub-account's trading account
type SubAccountBalanceDetail struct {
	Currency                       currency.Code `json:"ccy"`
	Equity                         types.Number  `json:"eq"`
	CashBalance                    types.Number  `json:"cashBal"`
	UpdateTime                     types.Time    `json:"uTime"`
	IsolatedEquity                 types.Number  `json:"isoEq"`
	AvailableEquity                types.Number  `json:"availEq"`
	DiscountEquity                 types.Number  `json:"disEq"`
	FixedBalance                   types.Number  `json:"fixedBal"`
	AvailableBalance               types.Number  `json:"availBal"`
	FrozenBalance                  types.Number  `json:"frozenBal"`
	OrderFrozen                    types.Number  `json:"ordFrozen"`
	Liabilities                    types.Number  `json:"liab"`
	UnrealisedPNL                  types.Number  `json:"upl"`
	UnrealisedPNLLiabilities       types.Number  `json:"uplLiab"`
	CrossLiabilities               types.Number  `json:"crossLiab"`
	RewardBalance                  types.Number  `json:"rewardBal"`
	IsolatedLiabilities            types.Number  `json:"isoLiab"`
	MarginRatio                    types.Number  `json:"mgnRatio"`
	InitialMarginRequirement       types.Number  `json:"imr"`
	MaintenanceMarginRequirement   types.Number  `json:"mmr"`
	Interest                       types.Number  `json:"interest"`
	TWAP                           types.Number  `json:"twap"` // Forced repayment risk level, 0 to 5
	ForcedRepaymentType            string        `json:"frpType"`
	MaximumLoan                    types.Number  `json:"maxLoan"`
	EquityUSD                      types.Number  `json:"eqUsd"`
	BorrowFrozen                   types.Number  `json:"borrowFroz"`
	NotionalLeverage               types.Number  `json:"notionalLever"`
	StrategyEquity                 types.Number  `json:"stgyEq"`
	IsolatedUnrealisedPNL          types.Number  `json:"isoUpl"`
	SpotInUseAmount                types.Number  `json:"spotInUseAmt"`
	ClientSpotInUseAmount          types.Number  `json:"clSpotInUseAmt"`
	MaximumSpotInUse               types.Number  `json:"maxSpotInUse"`
	SpotIsolatedBalance            types.Number  `json:"spotIsoBal"`
	SmartSyncEquity                types.Number  `json:"smtSyncEq"`
	SpotCopyTradingEquity          types.Number  `json:"spotCopyTradingEq"`
	SpotBalance                    types.Number  `json:"spotBal"`
	OpenAveragePrice               types.Number  `json:"openAvgPx"`
	AccumulatedAveragePrice        types.Number  `json:"accAvgPx"`
	SpotUnrealisedPNL              types.Number  `json:"spotUpl"`
	SpotUnrealisedPNLRatio         types.Number  `json:"spotUplRatio"`
	TotalPNL                       types.Number  `json:"totalPnl"`
	TotalPNLRatio                  types.Number  `json:"totalPnlRatio"`
	CollateralRestriction          string        `json:"colRes"`
	CollateralBorrowAutoConversion types.Number  `json:"colBorrAutoConversion"` // Auto conversion risk level, 0 to 5
	CollateralRestrict             bool          `json:"collateralRestrict"`    // Deprecated by OKX for CollateralRestriction
	CollateralEnabled              bool          `json:"collateralEnabled"`
	AutoLendStatus                 string        `json:"autoLendStatus"`
	AutoLendMatchedAmount          types.Number  `json:"autoLendMtAmt"`
}

// SubAccountFundingBalanceRequest holds the parameters for GetSubAccountFundingBalance
type SubAccountFundingBalanceRequest struct {
	SubAccountName string
	Currencies     currency.Currencies
}

// SubAccountMaximumWithdrawalRequest holds the parameters for GetSubAccountMaximumWithdrawal
type SubAccountMaximumWithdrawalRequest struct {
	SubAccountName string
	Currencies     currency.Currencies
}

// SubAccountMaximumWithdrawalResponse holds a currency's maximum withdrawal from a sub-account
type SubAccountMaximumWithdrawalResponse struct {
	Currency                                     currency.Code `json:"ccy"`
	MaximumWithdrawal                            types.Number  `json:"maxWd"`
	MaximumWithdrawalIncludingBorrowed           types.Number  `json:"maxWdEx"`
	SpotOffsetMaximumWithdrawal                  types.Number  `json:"spotOffsetMaxWd"`
	SpotOffsetMaximumWithdrawalIncludingBorrowed types.Number  `json:"spotOffsetMaxWdEx"`
}

// SubAccountTransferHistoryRequest holds the filters for GetSubAccountTransferHistory
type SubAccountTransferHistoryRequest struct {
	Currency       currency.Code
	TransferType   string
	SubAccountName string
	StartTime      time.Time
	EndTime        time.Time
	Limit          uint64
}

// SubAccountTransferResponse holds a transfer between the master account and a sub-account
type SubAccountTransferResponse struct {
	BillID         string        `json:"billId"`
	Currency       currency.Code `json:"ccy"`
	Amount         types.Number  `json:"amt"`
	BillType       string        `json:"type"`
	SubAccountName string        `json:"subAcct"`
	Timestamp      types.Time    `json:"ts"`
}

// ManagedSubAccountTransferHistoryRequest holds the filters for GetManagedSubAccountTransferHistory
type ManagedSubAccountTransferHistoryRequest struct {
	Currency       currency.Code
	TransferType   string
	SubAccountName string
	SubAccountUID  string
	StartTime      time.Time
	EndTime        time.Time
	Limit          uint64
}

// ManagedSubAccountTransferResponse holds a transfer between the master account and a managed sub-account
type ManagedSubAccountTransferResponse struct {
	BillID         string        `json:"billId"`
	Currency       currency.Code `json:"ccy"`
	Amount         types.Number  `json:"amt"`
	BillType       string        `json:"type"`
	SubAccountName string        `json:"subAcct"`
	SubAccountUID  string        `json:"subUid"`
	Timestamp      types.Time    `json:"ts"`
}

// TransferBetweenSubAccountsRequest holds the parameters for TransferBetweenSubAccounts
type TransferBetweenSubAccountsRequest struct {
	Currency               currency.Code `json:"ccy"`
	Amount                 float64       `json:"amt,string"`
	RemittingAccountType   string        `json:"from"`
	BeneficiaryAccountType string        `json:"to"`
	FromSubAccount         string        `json:"fromSubAccount"`
	ToSubAccount           string        `json:"toSubAccount"`
	LoanTransfer           bool          `json:"loanTrans,omitempty"`
	OmitPositionRisk       bool          `json:"omitPosRisk,string,omitempty"`
}

// TransferIDResponse holds a transfer's ID
type TransferIDResponse struct {
	TransferID string `json:"transId"`
}

// PermissionOfTransferOutRequest holds the parameters for SetPermissionOfTransferOut
type PermissionOfTransferOutRequest struct {
	SubAccountNames []string `json:"-"`
	CanTransferOut  bool     `json:"canTransOut"`
}

// MarshalJSON sends the sub-account names as the comma-separated list OKX expects
func (r *PermissionOfTransferOutRequest) MarshalJSON() ([]byte, error) {
	type Alias PermissionOfTransferOutRequest
	return json.Marshal(&struct {
		*Alias
		SubAccountNames string `json:"subAcct"`
	}{
		Alias:           (*Alias)(r),
		SubAccountNames: strings.Join(r.SubAccountNames, ","),
	})
}

// PermissionOfTransferOutResponse holds a sub-account's permission to transfer out
type PermissionOfTransferOutResponse struct {
	SubAccountName string `json:"subAcct"`
	CanTransferOut bool   `json:"canTransOut"`
}

// CustodyTradingSubAccountListRequest holds the filters for GetCustodyTradingSubAccountList
type CustodyTradingSubAccountListRequest struct {
	SubAccountName string
}
