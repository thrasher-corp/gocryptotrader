package kucoin

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UTAWithdrawalQuotasRequest holds the parameters of GetUTAWithdrawalQuotas
type UTAWithdrawalQuotasRequest struct {
	// Chain is the chain's lowercase identifier, such as trx
	Chain string
	// Currency is required
	Currency currency.Code
	// IsInner selects the quotas of an internal withdrawal
	IsInner bool
	// WithdrawType is required: ADDRESS for an on-chain withdrawal, or MAIL, PHONE or UID for an internal one
	WithdrawType string
}

// UTAWithdrawalQuotasResponse holds a currency's withdrawal quotas, fees and limits on a chain
type UTAWithdrawalQuotasResponse struct {
	Currency currency.Code `json:"currency"`
	// Chain is the chain's identifier used in requests, such as trx
	Chain string `json:"chain"`
	// ChainName is the chain's display name, such as TRC20
	ChainName         string `json:"chainName"`
	IsWithdrawEnabled bool   `json:"isWithdrawEnabled"`
	// Reason says why withdrawals are unavailable
	Reason                  string       `json:"reason"`
	AvailableWithdrawAmount types.Number `json:"availableWithdrawAmount"`
	RemainingQuotaAmount    types.Number `json:"remainingQuotaAmount"`
	LockedAmount            types.Number `json:"lockedAmount"`
	// WithdrawMinSize depends on the requested withdrawal type
	WithdrawMinSize     types.Number `json:"withdrawMinSize"`
	MinWithdrawFee      types.Number `json:"minWithdrawFee"`
	MinInnerWithdrawFee types.Number `json:"minInnerWithdrawFee"`
	Precision           uint64       `json:"precision"`
	// QuotaCurrency is the currency the withdrawal quota is counted in
	QuotaCurrency            currency.Code `json:"quotaCurrency"`
	LimitQuotaCurrencyAmount types.Number  `json:"limitQuotaCurrencyAmount"`
	UsedQuotaCurrencyAmount  types.Number  `json:"usedQuotaCurrencyAmount"`
	LimitBTCAmount           types.Number  `json:"limitBTCAmount"`
	UsedBTCAmount            types.Number  `json:"usedBTCAmount"`
}

// UTAWithdrawalHistoryResponse holds a page of withdrawal records
type UTAWithdrawalHistoryResponse struct {
	CurrentPage uint64                `json:"currentPage"`
	PageSize    uint64                `json:"pageSize"`
	TotalNumber uint64                `json:"totalNum"`
	TotalPage   uint64                `json:"totalPage"`
	Items       []UTAWithdrawalRecord `json:"items"`
}

// UTAWithdrawalRecord holds a withdrawal record. A manually cancelled withdrawal has the status FAILURE and the failure
// reason withdraw.manual.cancel
type UTAWithdrawalRecord struct {
	ID       string        `json:"id"`
	Currency currency.Code `json:"currency"`
	// Chain is the chain's identifier used in requests, such as trx
	Chain   string `json:"chain"`
	Status  string `json:"status"`
	Address string `json:"address"`
	Memo    string `json:"memo"`
	// IsInner is true for an internal transfer
	IsInner              bool         `json:"isInner"`
	Amount               types.Number `json:"amount"`
	Fee                  types.Number `json:"fee"`
	FailureReason        string       `json:"failureReason"`
	FailureReasonMessage string       `json:"failureReasonMsg"`
	WalletTransactionID  string       `json:"walletTxId"`
	CreatedAt            types.Time   `json:"createdAt"`
	UpdatedAt            types.Time   `json:"updatedAt"`
	Remark               string       `json:"remark"`
	SubStatus            string       `json:"subStatus"`
}

// UTAWithdrawRequest holds the parameters of UTAWithdraw
type UTAWithdrawRequest struct {
	// Currency is required
	Currency currency.Code
	// Chain is the chain's identifier; set it for a currency with several chains rather than relying on the default
	Chain string
	// Amount is required and must match the currency's precision
	Amount float64
	// ToAddress is required. With the withdrawal address whitelist enabled, it and the chain must match a saved
	// address, case-sensitively
	ToAddress string
	// Memo is required by some chains and receiving platforms; a missing or wrong memo can lose the funds
	Memo   string
	Remark string
	// IsInner marks an internal withdrawal, which the UID, MAIL and PHONE withdrawal types make
	IsInner bool
	// WithdrawType is required: ADDRESS for an on-chain withdrawal, or UID, MAIL or PHONE for an internal one, which is
	// limited to 3 requests per 10 seconds and 50 per 24 hours
	WithdrawType string
	// FeeDeductType is INTERNAL or EXTERNAL. When empty, the fee is taken from the main account if its balance covers it,
	// and otherwise from the withdrawal amount
	FeeDeductType string
}

// UTAWithdrawResponse holds a withdrawal's ID
type UTAWithdrawResponse struct {
	WithdrawalID string `json:"withdrawalId"`
}

// CancelUTAWithdrawalResponse holds a cancelled withdrawal's ID
type CancelUTAWithdrawalResponse struct {
	WithdrawID string `json:"withdrawId"`
}
