package kucoin

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WithdrawalQuotasResponse is a currency's withdrawal quota
type WithdrawalQuotasResponse struct {
	Currency       currency.Code `json:"currency"`
	LimitBTCAmount types.Number  `json:"limitBTCAmount"`
	UsedBTCAmount  types.Number  `json:"usedBTCAmount"`
	// QuotaCurrency is the currency the intraday limit is counted in
	QuotaCurrency currency.Code `json:"quotaCurrency"`
	// LimitQuotaCurrencyAmount is the intraday withdrawal limit, in QuotaCurrency
	LimitQuotaCurrencyAmount types.Number `json:"limitQuotaCurrencyAmount"`
	// UsedQuotaCurrencyAmount is the amount withdrawn today, in QuotaCurrency
	UsedQuotaCurrencyAmount types.Number `json:"usedQuotaCurrencyAmount"`
	// RemainAmount is what can still be withdrawn today
	RemainAmount    types.Number `json:"remainAmount"`
	AvailableAmount types.Number `json:"availableAmount"`
	WithdrawMinFee  types.Number `json:"withdrawMinFee"`
	// InnerWithdrawMinFee is the fee for internal withdrawals
	InnerWithdrawMinFee types.Number `json:"innerWithdrawMinFee"`
	WithdrawMinSize     types.Number `json:"withdrawMinSize"`
	IsWithdrawEnabled   bool         `json:"isWithdrawEnabled"`
	Precision           uint64       `json:"precision"`
	Chain               string       `json:"chain"`
	// Reason explains any restriction and is usually empty
	Reason string `json:"reason"`
	// LockedAmount includes the amounts locked into USDT for each currency
	LockedAmount types.Number `json:"lockedAmount"`
}

// WithdrawV3Request holds the Withdraw (V3) parameters; Currency, ToAddress, Amount and WithdrawType are required
type WithdrawV3Request struct {
	Currency currency.Code
	// Chain is the currency's chain ID; KuCoin recommends setting it for a currency on several chains
	Chain string
	// Amount is a multiple of the currency's precision
	Amount float64
	// Memo is the address tag the receiving address needs, if any
	Memo    string
	IsInner bool
	Remark  string
	// FeeDeductType is INTERNAL to deduct the fee from the amount or EXTERNAL to deduct it from the main account;
	// unset, KuCoin deducts it from the main account when its balance is enough and from the amount otherwise
	FeeDeductType string
	// ToAddress is the address, UID, email or phone number that WithdrawType names
	ToAddress string
	// WithdrawType is ADDRESS, UID, MAIL or PHONE
	WithdrawType string
}

// WithdrawalResponse identifies the withdrawal KuCoin accepted
type WithdrawalResponse struct {
	WithdrawalID string `json:"withdrawalId"`
}

// WithdrawalHistoryRequest holds the Get Withdrawal History parameters; Currency is required
type WithdrawalHistoryRequest struct {
	Currency currency.Code
	// Status is REVIEW, PROCESSING, WALLET_PROCESSING, SUCCESS or FAILURE
	Status      string
	StartAt     time.Time
	EndAt       time.Time
	CurrentPage uint64
	// PageSize is between 10 and 500
	PageSize uint64
}

// WithdrawalHistoryResponse is a page of withdrawals, latest first
type WithdrawalHistoryResponse struct {
	CurrentPage uint64       `json:"currentPage"`
	PageSize    uint64       `json:"pageSize"`
	TotalNumber uint64       `json:"totalNum"`
	TotalPage   uint64       `json:"totalPage"`
	Items       []Withdrawal `json:"items"`
}

// Withdrawal is a withdrawal (Get Withdrawal History)
type Withdrawal struct {
	ID       string        `json:"id"`
	Currency currency.Code `json:"currency"`
	Chain    string        `json:"chain"`
	Status   string        `json:"status"`
	Address  string        `json:"address"`
	Memo     string        `json:"memo"`
	IsInner  bool          `json:"isInner"`
	Amount   types.Number  `json:"amount"`
	Fee      types.Number  `json:"fee"`
	// WalletTxID is empty for internal withdrawals
	WalletTxID string     `json:"walletTxId"`
	CreatedAt  types.Time `json:"createdAt"`
	UpdatedAt  types.Time `json:"updatedAt"`
	Remark     string     `json:"remark"`
	// FailureReason is SYSTEM_EXCEPTION, NO_BALANCE, CANCEL, REJECTED or WALLET_FAILURE
	FailureReason        string `json:"failureReason"`
	FailureReasonMessage string `json:"failureReasonMsg"`
	// Arrears is in the docs' response example but not their schema
	Arrears bool `json:"arrears"`
}

// WithdrawalHistoryByIDResponse is a withdrawal (Get Withdrawal History By ID)
type WithdrawalHistoryByIDResponse struct {
	ID  string `json:"id"`
	UID uint64 `json:"uid"`
	// Currency is the currency's code, which never changes
	Currency  currency.Code `json:"currency"`
	ChainID   string        `json:"chainId"`
	ChainName string        `json:"chainName"`
	// CurrencyName changes when KuCoin renames the currency
	CurrencyName string `json:"currencyName"`
	// Status is REVIEW, PROCESSING, WALLET_PROCESSING, SUCCESS or FAILURE
	Status string `json:"status"`
	// FailureReason is SYSTEM_EXCEPTION, NO_BALANCE, CANCEL, REJECTED or WALLET_FAILURE
	FailureReason        string       `json:"failureReason"`
	FailureReasonMessage string       `json:"failureReasonMsg"`
	Address              string       `json:"address"`
	Memo                 string       `json:"memo"`
	IsInner              bool         `json:"isInner"`
	Amount               types.Number `json:"amount"`
	Fee                  types.Number `json:"fee"`
	WalletTxID           string       `json:"walletTxId"`
	AddressRemark        string       `json:"addressRemark"`
	Remark               string       `json:"remark"`
	CreatedAt            types.Time   `json:"createdAt"`
	// CancelType is CANCELABLE, CANCELING or NON_CANCELABLE
	CancelType string `json:"cancelType"`
	// Taxes is only returned to users in some regions
	Taxes          []string `json:"taxes"`
	TaxDescription string   `json:"taxDescription"`
	TxID           string   `json:"txId"`
	// ReturnStatus is NOT_RETURN, PROCESSING or SUCCESS
	ReturnStatus   string        `json:"returnStatus"`
	ReturnAmount   types.Number  `json:"returnAmount"`
	ReturnCurrency currency.Code `json:"returnCurrency"`
}
