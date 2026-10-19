package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// Coinbase App fiat deposit and withdrawal statuses
const (
	AppFiatTransferStatusCreated   = "created"
	AppFiatTransferStatusCompleted = "completed"
	AppFiatTransferStatusCancelled = "canceled"
)

// Coinbase App transfer types, which tell a deposit from a withdrawal
const (
	AppTransferTypeDeposit    = "TRANSFER_TYPE_DEPOSIT"
	AppTransferTypeWithdrawal = "TRANSFER_TYPE_WITHDRAWAL"
)

// AppFiatTransferRequest deposits funds from a payment method into a fiat account, or withdraws funds from a fiat
// account to a payment method
type AppFiatTransferRequest struct {
	AccountID string
	Amount    float64
	Currency  currency.Code
	// PaymentMethodID is the ID of the bank account or other payment method the funds move from or to, which Advanced
	// Trade's List Payment Methods returns
	PaymentMethodID string
	// Commit completes the transfer at once. An uncommitted transfer, which Coinbase makes by default since 11 April
	// 2025, shows the transfer's amounts and fees and completes only once it is committed
	Commit bool
}

// appFiatTransferBody is Deposit Funds' and Withdraw Funds' request body; commit is always sent, so a transfer does not
// depend on Coinbase's default, which changed to false on 11 April 2025
type appFiatTransferBody struct {
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
	Commit        bool   `json:"commit"`
}

// AppFiatTransferIDRequest selects one of an account's deposits or withdrawals
type AppFiatTransferIDRequest struct {
	AccountID  string
	TransferID string
}

// AppListFiatTransfersRequest selects a page of an account's deposits or withdrawals
type AppListFiatTransfersRequest struct {
	AccountID string
	// Page selects the page; nil selects the first
	Page *AppPageRequest
}

// appTransferResponse holds the transfer Deposit Funds, Withdraw Funds and their commits return, which has no data
// envelope
type appTransferResponse struct {
	Transfer *AppTransfer `json:"transfer"`
}

// AppTransfer is a deposit or withdrawal as Deposit Funds, Withdraw Funds, Commit Deposit and Commit Withdrawal
// return it, its amounts holding a value rather than an amount
type AppTransfer struct {
	// UserEnteredAmount is the amount the request asked for
	UserEnteredAmount Amount `json:"user_entered_amount"`
	Amount            Amount `json:"amount"`
	Total             Amount `json:"total"`
	Subtotal          Amount `json:"subtotal"`
	IdempotencyKey    string `json:"idem"`
	Committed         bool   `json:"committed"`
	// ID is the transfer's ID, which CommitDeposit and CommitWithdrawal take
	ID      string `json:"id"`
	Instant bool   `json:"instant"`
	// Source is where the funds move from, and Target where they move to
	Source        AppTransferParty `json:"source"`
	Target        AppTransferParty `json:"target"`
	PayoutAt      time.Time        `json:"payout_at"`
	Status        string           `json:"status"`
	UserReference string           `json:"user_reference"`
	// Type is TRANSFER_TYPE_DEPOSIT or TRANSFER_TYPE_WITHDRAWAL
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// UserWarnings, CancellationReason and NextStep are documented only as empty or null, so they are kept as sent
	UserWarnings           []json.RawMessage `json:"user_warnings"`
	Fees                   []AppTransferFee  `json:"fees"`
	TotalFee               AppTransferFee    `json:"total_fee"`
	CancellationReason     json.RawMessage   `json:"cancellation_reason"`
	HoldDays               uint64            `json:"hold_days"`
	NextStep               json.RawMessage   `json:"nextStep"`
	CheckoutURL            string            `json:"checkout_url"`
	RequiresCompletionStep bool              `json:"requires_completion_step"`
}

// AppTransferParty is where a transfer's funds move from or to: an external payment method such as a bank account,
// or a Coinbase ledger account
type AppTransferParty struct {
	// Type is EXTERNAL_PAYMENT_METHOD or LEDGER_ACCOUNT
	Type string `json:"type"`
	// Network is the network the funds move on, such as ach, or internal_retail within Coinbase
	Network               string                    `json:"network"`
	PaymentMethodID       string                    `json:"payment_method_id"`
	ExternalPaymentMethod *AppExternalPaymentMethod `json:"external_payment_method"`
	LedgerAccount         *AppLedgerAccount         `json:"ledger_account"`
}

// AppExternalPaymentMethod is the payment method outside Coinbase a transfer's funds move from or to
type AppExternalPaymentMethod struct {
	PaymentMethodID string `json:"payment_method_id"`
}

// AppLedgerAccount is the Coinbase account a transfer's funds move from or to
type AppLedgerAccount struct {
	AccountID string                `json:"account_id"`
	Currency  currency.Code         `json:"currency"`
	Owner     AppLedgerAccountOwner `json:"owner"`
}

// AppLedgerAccountOwner is the owner of a ledger account, such as a RETAIL user
type AppLedgerAccountOwner struct {
	ID       string `json:"id"`
	UUID     string `json:"uuid"`
	UserUUID string `json:"user_uuid"`
	Type     string `json:"type"`
}

// AppTransferFee is a fee a transfer pays, or the total of its fees
type AppTransferFee struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Amount      Amount `json:"amount"`
	// Type is who charges the fee, such as COINBASE
	Type string `json:"type"`
}

// AppFiatTransfer is a fiat deposit or withdrawal as List and Show return it; Resource tells them apart. A committed
// transfer also has a transaction
type AppFiatTransfer struct {
	ID string `json:"id"`
	// Status is created, completed or canceled
	Status        string               `json:"status"`
	PaymentMethod AppResourceReference `json:"payment_method"`
	// Transaction is the transfer's transaction, which an uncommitted transfer does not have
	Transaction *AppResourceReference `json:"transaction"`
	Amount      AppMoney              `json:"amount"`
	// Subtotal is the amount without fees
	Subtotal     AppMoney  `json:"subtotal"`
	Fee          AppMoney  `json:"fee"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Resource     string    `json:"resource"`
	ResourcePath string    `json:"resource_path"`
	Committed    bool      `json:"committed"`
	// PayoutAt is when a transfer that is not executed at once will be
	PayoutAt time.Time `json:"payout_at"`
}

// AppResourceReference names another Coinbase App resource, such as a payment method or a transaction
type AppResourceReference struct {
	ID           string `json:"id"`
	Resource     string `json:"resource"`
	ResourcePath string `json:"resource_path"`
}
