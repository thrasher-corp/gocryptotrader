package cryptodotcom

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Payment networks, the rails fiat is deposited and withdrawn on, each with its currency
const (
	PaymentNetworkOpenPaydSEPA = "openpayd_exchange_sepa" // EUR
	PaymentNetworkUSDSWIFT     = "usd_swift"              // USD
	PaymentNetworkUSDFedwire   = "usd_fedwire"            // USD
	PaymentNetworkUSDCubix     = "usd_cubix"              // USD
	PaymentNetworkUKFPS        = "uk_fps"                 // GBP
	PaymentNetworkAEDIPI       = "aed_ipi"                // AED
	// PaymentNetworkUSACH is used throughout the docs' examples, though missing from their table of payment networks
	PaymentNetworkUSACH = "us_ach" // USD
)

// Fiat transaction statuses; the docs' examples of created withdrawals report FiatTransactionStatusDebitProcessing,
// which the histories' lists of statuses leave out
const (
	FiatTransactionStatusPending         = "pending"
	FiatTransactionStatusDebitProcessing = "debit_processing"
	FiatTransactionStatusCompleted       = "completed"
	FiatTransactionStatusFailed          = "failed"
	FiatTransactionStatusCancelled       = "cancelled"
	FiatTransactionStatusRefunded        = "refunded"
)

// Bank account statuses; an expired bank account's Plaid token has expired, so it must be relinked before it can be
// withdrawn to again
const (
	FiatBankAccountStatusPending            = "pending"
	FiatBankAccountStatusSubmitted          = "submitted"
	FiatBankAccountStatusPendingAuthorise   = "pending_authorize"
	FiatBankAccountStatusAuthenticated      = "authenticated"
	FiatBankAccountStatusTransactionEnabled = "transaction_enabled"
	FiatBankAccountStatusPendingProof       = "pending_proof"
	FiatBankAccountStatusProofSubmitted     = "proof_submitted"
	FiatBankAccountStatusApproved           = "approved"
	FiatBankAccountStatusRejected           = "rejected"
	FiatBankAccountStatusEscalated          = "escalated"
	FiatBankAccountStatusCompleted          = "completed"
	FiatBankAccountStatusDeleted            = "deleted"
	FiatBankAccountStatusExpired            = "expired"
)

// fiatAccessesResponse is the result of a fiat-access request
type fiatAccessesResponse struct {
	Accesses []FiatAccessResponse `json:"accesses"`
}

// FiatAccessResponse is whether a currency can be deposited and withdrawn on a payment network
type FiatAccessResponse struct {
	Currency          currency.Code `json:"currency"`
	PaymentNetwork    string        `json:"payment_network"`
	DepositEnabled    bool          `json:"is_deposit_enable"`
	WithdrawalEnabled bool          `json:"is_withdraw_enable"`
}

// FiatPaymentNetworksRequest filters by payment network
type FiatPaymentNetworksRequest struct {
	// PaymentNetworks are PaymentNetwork constants, every payment network when empty
	PaymentNetworks []string
}

// fiatPaymentNetworksParams is the params of a fiat-deposit-info or fiat-get-bank-accounts request
type fiatPaymentNetworksParams struct {
	PaymentNetworks string `json:"payment_networks,omitempty"`
}

// fiatDepositInfoListResponse is the result of a fiat-deposit-info request
type fiatDepositInfoListResponse struct {
	DepositInfo []FiatDepositInfoResponse `json:"deposit_info_list"`
}

// FiatDepositInfoResponse is the bank account to deposit a currency to on a payment network
type FiatDepositInfoResponse struct {
	PaymentNetwork string                 `json:"payment_network"`
	Currency       currency.Code          `json:"currency"`
	BankDetails    FiatDepositBankDetails `json:"bank_details"`
}

// FiatDepositBankDetails is a bank account fiat is deposited to; each payment network sets only the details it uses
type FiatDepositBankDetails struct {
	AccountHolderName string `json:"account_holder_name"`
	BankName          string `json:"bank_name"`
	BankAddress       string `json:"bank_address"`
	BankCountry       string `json:"bank_country"`
	RoutingNumber     string `json:"routing_number"`
	AccountNumber     string `json:"account_number"`
	RecipientName     string `json:"recipient_name"`
	RecipientAddress  string `json:"recipient_address"`
	// BankIdentifierCode is the bank's SWIFT code
	BankIdentifierCode             string `json:"bic_code"`
	InternationalBankAccountNumber string `json:"iban_code"`
	ReferenceCode                  string `json:"reference_code"`
	SortCode                       string `json:"sort_code"`
	CubixPartnerName               string `json:"cubix_partner_name"`
	CubixAccountName               string `json:"cubix_account_name"`
	CubixAccountID                 string `json:"cubix_account_id"`
	// CubixAccountNumber is the last four digits of the Cubix account number
	CubixAccountNumber string `json:"cubix_account_number"`
	AccountType        string `json:"account_type"`
	// Meta is left undecoded, as the docs leave its shape undescribed
	Meta json.RawMessage `json:"meta"`
}

// FiatHistoryRequest pages through a fiat deposit or withdrawal history; the page and page size are required
type FiatHistoryRequest struct {
	// Page counts from 0
	Page     uint64
	PageSize uint64
	// StartTime is required with an end time
	StartTime time.Time
	EndTime   time.Time
	// PaymentNetworks are PaymentNetwork constants, every payment network when empty
	PaymentNetworks []string
}

// fiatHistoryParams is the params of a fiat-deposit-history or fiat-withdraw-history request
type fiatHistoryParams struct {
	Page            uint64 `json:"page,string"`
	PageSize        uint64 `json:"page_size,string"`
	StartTime       uint64 `json:"start_time,omitempty,string"`
	EndTime         uint64 `json:"end_time,omitempty,string"`
	PaymentNetworks string `json:"payment_networks,omitempty"`
}

// FiatTransactionHistoryResponse is a page of fiat deposits or withdrawals
type FiatTransactionHistoryResponse struct {
	Page         uint64            `json:"page,string"`
	PageSize     uint64            `json:"page_size,string"`
	Total        uint64            `json:"total,string"`
	Transactions []FiatTransaction `json:"transaction_history_list"`
}

// FiatTransaction is a fiat deposit or withdrawal; the account ID, amounts in USD and client withdrawal ID are only
// reported for withdrawals
type FiatTransaction struct {
	ID             string              `json:"id"`
	AccountID      string              `json:"account_id"`
	Currency       currency.Code       `json:"currency"`
	Amount         types.PreciseNumber `json:"amount"`
	AmountInUSD    types.PreciseNumber `json:"amount_in_usd"`
	FeeCurrency    currency.Code       `json:"fee_currency"`
	FeeAmount      types.PreciseNumber `json:"fee_amount"`
	FeeAmountInUSD types.PreciseNumber `json:"fee_amount_in_usd"`
	// Fee is the docs' alternative to FeeAmount
	Fee            types.PreciseNumber `json:"fee"`
	PaymentNetwork string              `json:"payment_network"`
	// Status is a FiatTransactionStatus constant
	Status             string     `json:"status"`
	CreatedAt          types.Time `json:"created_at"`
	UpdatedAt          types.Time `json:"updated_at"`
	CompletedAt        types.Time `json:"completed_at"`
	ClientWithdrawalID string     `json:"client_wid"`
	// VendorTransactionID is the ID the payment service provider that processed the transaction gave it
	VendorTransactionID string                `json:"vendor_transaction_id"`
	Sender              *FiatTransactionParty `json:"sender"`
	Beneficiary         *FiatTransactionParty `json:"beneficiary"`
}

// FiatTransactionParty is a bank account a fiat transaction is sent from or to
type FiatTransactionParty struct {
	AccountIdentifierValue string `json:"account_identifier_value"`
}

// FiatWithdrawalRequest withdraws fiat to a linked bank account; the bank account ID, amount, currency and payment
// network are required
type FiatWithdrawalRequest struct {
	ClientWithdrawalID string `json:"client_wid,omitempty"`
	// BankAccountID is the ID of a bank account from GetFiatBankAccounts, not its account ID
	BankAccountID  string        `json:"account_id"`
	Amount         types.Number  `json:"amount"`
	Currency       currency.Code `json:"currency"`
	PaymentNetwork string        `json:"payment_network"`
	// IntermediateBank is only for PaymentNetworkUSDSWIFT, and replaces the intermediary bank stored for the bank
	// account
	IntermediateBank *FiatIntermediateBank `json:"intermediate_bank,omitempty"`
	BeneficiaryID    string                `json:"beneficiary_id,omitempty"`
	AuthorisationID  string                `json:"authorization_id,omitempty"`
	// BankIdentifierValue and AccountIdentifierValue are only for PaymentNetworkAEDIPI
	BankIdentifierValue    string `json:"bank_identifier_value,omitempty"`
	AccountIdentifierValue string `json:"account_identifier_value,omitempty"`
}

// FiatIntermediateBank is the intermediary bank a SWIFT withdrawal passes through
type FiatIntermediateBank struct {
	// BankIdentifierType is the kind of bank identifier, such as SWIFT or ROUTING
	BankIdentifierType  string `json:"bank_identifier_type,omitempty"`
	BankIdentifierValue string `json:"bank_identifier_value,omitempty"`
	BankName            string `json:"bank_name,omitempty"`
	AddressLine1        string `json:"address_1,omitempty"`
	AddressLine2        string `json:"address_2,omitempty"`
}

// FiatWithdrawalResponse is a created fiat withdrawal
type FiatWithdrawalResponse struct {
	ID                 string `json:"id"`
	ClientWithdrawalID string `json:"client_wid"`
	BankAccountID      string `json:"account_id"`
	// Status is a FiatTransactionStatus constant
	Status          string              `json:"status"`
	PaymentNetwork  string              `json:"payment_network"`
	Currency        currency.Code       `json:"currency"`
	Amount          types.PreciseNumber `json:"amount"`
	AmountInUSD     types.PreciseNumber `json:"amount_in_usd"`
	FeeCurrency     currency.Code       `json:"fee_currency"`
	FeeAmount       types.PreciseNumber `json:"fee_amount"`
	BeneficiaryID   string              `json:"beneficiary_id"`
	AuthorisationID string              `json:"authorization_id"`
}

// fiatPaymentNetworkParams is the params of a fiat-transaction-quota or fiat-transaction-limit request
type fiatPaymentNetworkParams struct {
	PaymentNetwork string `json:"payment_network"`
}

// FiatCurrencyAmount is an amount of a currency
type FiatCurrencyAmount struct {
	Currency currency.Code `json:"currency"`
	// Amount is nil for no limit, which Crypto.com sends as null
	Amount *types.Number `json:"amount"`
}

// fiatTransactionQuotaDetailsResponse is the result of a fiat-transaction-quota request
type fiatTransactionQuotaDetailsResponse struct {
	Details *FiatTransactionQuotaResponse `json:"details"`
}

// FiatTransactionQuotaResponse is how much can be withdrawn on a payment network each day and month, in USD and in the
// network's currency, and how much of that is used. The transaction counts per period and remaining are nil for no
// limit, which Crypto.com sends as null, as 0 would mean none
type FiatTransactionQuotaResponse struct {
	Currency                                  currency.Code      `json:"currency"`
	PaymentNetwork                            string             `json:"payment_network"`
	NetworkName                               string             `json:"network_name"`
	MonthlyQuotaInUSD                         FiatCurrencyAmount `json:"monthly_quota_in_usd"`
	DailyQuotaInUSD                           FiatCurrencyAmount `json:"daily_quota_in_usd"`
	UsedMonthlyQuotaInUSD                     FiatCurrencyAmount `json:"used_monthly_quota_in_usd"`
	UsedDailyQuotaInUSD                       FiatCurrencyAmount `json:"used_daily_quota_in_usd"`
	RemainingMonthlyQuotaInUSD                FiatCurrencyAmount `json:"remaining_monthly_quota_in_usd"`
	RemainingDailyQuotaInUSD                  FiatCurrencyAmount `json:"remaining_daily_quota_in_usd"`
	MinimumWithdrawalAmountInUSD              FiatCurrencyAmount `json:"minimum_withdrawal_amount_in_usd"`
	MonthlyQuota                              FiatCurrencyAmount `json:"monthly_quota"`
	DailyQuota                                FiatCurrencyAmount `json:"daily_quota"`
	UsedMonthlyQuota                          FiatCurrencyAmount `json:"used_monthly_quota"`
	UsedDailyQuota                            FiatCurrencyAmount `json:"used_daily_quota"`
	RemainingMonthlyQuota                     FiatCurrencyAmount `json:"remaining_monthly_quota"`
	RemainingDailyQuota                       FiatCurrencyAmount `json:"remaining_daily_quota"`
	MinimumWithdrawalAmount                   FiatCurrencyAmount `json:"minimum_withdrawal_amount"`
	CurrencyDailyQuota                        FiatCurrencyAmount `json:"currency_daily_quota"`
	CurrencyMonthlyQuota                      FiatCurrencyAmount `json:"currency_monthly_quota"`
	CurrencyUsedDailyQuota                    FiatCurrencyAmount `json:"currency_used_daily_quota"`
	CurrencyUsedMonthlyQuota                  FiatCurrencyAmount `json:"currency_used_monthly_quota"`
	CurrencyRemainingDailyQuota               FiatCurrencyAmount `json:"currency_remaining_daily_quota"`
	CurrencyRemainingMonthlyQuota             FiatCurrencyAmount `json:"currency_remaining_monthly_quota"`
	TransactionsPerDay                        *types.Number      `json:"transactions_per_day"`
	TransactionsPerMonth                      *types.Number      `json:"transactions_per_month"`
	TransactionsDailyCount                    types.Number       `json:"transactions_daily_count"`
	TransactionsMonthlyCount                  types.Number       `json:"transactions_monthly_count"`
	RemainingTransactionsDailyCount           *types.Number      `json:"remaining_transactions_daily_count"`
	RemainingTransactionsMonthlyCount         *types.Number      `json:"remaining_transactions_monthly_count"`
	CurrencyTransactionsPerDay                *types.Number      `json:"currency_transactions_per_day"`
	CurrencyTransactionsPerMonth              *types.Number      `json:"currency_transactions_per_month"`
	CurrencyTransactionsDailyCount            types.Number       `json:"currency_transactions_daily_count"`
	CurrencyTransactionsMonthlyCount          types.Number       `json:"currency_transactions_monthly_count"`
	CurrencyRemainingTransactionsDailyCount   *types.Number      `json:"currency_remaining_transactions_daily_count"`
	CurrencyRemainingTransactionsMonthlyCount *types.Number      `json:"currency_remaining_transactions_monthly_count"`
}

// FiatTransactionLimitResponse is the limits of deposits and payments on a payment network
type FiatTransactionLimitResponse struct {
	Deposit FiatDepositLimits `json:"deposit"`
	Payment FiatPaymentLimits `json:"payment"`
}

// FiatDepositLimits is the limits of deposits on a payment network; a zero quota or transaction count means no limit
type FiatDepositLimits struct {
	MinimumDepositAmount                   FiatCurrencyAmount `json:"min_deposit_amount"`
	DailyMaximumDepositAmount              FiatCurrencyAmount `json:"daily_max_deposit_amount"`
	MonthlyMaximumDepositAmount            FiatCurrencyAmount `json:"monthly_max_deposit_amount"`
	CurrencyDailyMaximumDepositAmount      FiatCurrencyAmount `json:"currency_daily_max_deposit_amount"`
	CurrencyMonthlyMaximumDepositAmount    FiatCurrencyAmount `json:"currency_monthly_max_deposit_amount"`
	DailyQuota                             types.Number       `json:"daily_quota"`
	MonthlyQuota                           types.Number       `json:"monthly_quota"`
	DailyTransactionCount                  types.Number       `json:"daily_transaction_count"`
	MonthlyTransactionCount                types.Number       `json:"monthly_transaction_count"`
	CurrencyDailyTransactionCount          types.Number       `json:"currency_daily_transaction_count"`
	CurrencyMonthlyTransactionCount        types.Number       `json:"currency_monthly_transaction_count"`
	FeeAmount                              FiatCurrencyAmount `json:"fee_amount"`
	DailyMaximumTransactionCount           types.Number       `json:"daily_max_transaction_count"`
	MonthlyMaximumTransactionCount         types.Number       `json:"monthly_max_transaction_count"`
	CurrencyDailyMaximumTransactionCount   types.Number       `json:"currency_daily_max_transaction_count"`
	CurrencyMonthlyMaximumTransactionCount types.Number       `json:"currency_monthly_max_transaction_count"`
}

// FiatPaymentLimits is the limits of payments on a payment network and how long they take; a zero maximum transaction
// count means no limit
type FiatPaymentLimits struct {
	Name                  string `json:"name"`
	FullName              string `json:"full_name"`
	ReviewTimeDescription string `json:"review_time_description"`
	// ReviewTime is how long a payment's review takes
	ReviewTime                  FiatDurationRange `json:"review_time"`
	BankTransferTimeDescription string            `json:"bank_transfer_time_description"`
	// BankTransferTime is how long a payment's bank transfer takes
	BankTransferTime                       FiatDurationRange  `json:"bank_transfer_time"`
	MinimumPaymentAmount                   FiatCurrencyAmount `json:"min_payment_amount"`
	DailyMaximumPaymentAmount              FiatCurrencyAmount `json:"daily_max_payment_amount"`
	MonthlyMaximumPaymentAmount            FiatCurrencyAmount `json:"monthly_max_payment_amount"`
	AutoApproveMaximumPaymentAmount        FiatCurrencyAmount `json:"auto_approve_max_payment_amount"`
	CurrencyDailyMaximumPaymentAmount      FiatCurrencyAmount `json:"currency_daily_max_payment_amount"`
	CurrencyMonthlyMaximumPaymentAmount    FiatCurrencyAmount `json:"currency_monthly_max_payment_amount"`
	FeeAmount                              FiatCurrencyAmount `json:"fee_amount"`
	RefundFeeAmount                        FiatCurrencyAmount `json:"refund_fee_amount"`
	DailyTransactionCount                  types.Number       `json:"daily_transaction_count"`
	MonthlyTransactionCount                types.Number       `json:"monthly_transaction_count"`
	CurrencyDailyTransactionCount          types.Number       `json:"currency_daily_transaction_count"`
	CurrencyMonthlyTransactionCount        types.Number       `json:"currency_monthly_transaction_count"`
	DailyMaximumTransactionCount           types.Number       `json:"daily_max_transaction_count"`
	MonthlyMaximumTransactionCount         types.Number       `json:"monthly_max_transaction_count"`
	CurrencyDailyMaximumTransactionCount   types.Number       `json:"currency_daily_max_transaction_count"`
	CurrencyMonthlyMaximumTransactionCount types.Number       `json:"currency_monthly_max_transaction_count"`
}

// FiatDurationRange is the shortest and longest a step of a payment takes
type FiatDurationRange struct {
	Minimum types.Number `json:"min"`
	Maximum types.Number `json:"max"`
	// Unit is the minimum's and maximum's unit, such as working days
	Unit string `json:"unit"`
}

// FiatBankAccountRequest links a bank account through Plaid; the Plaid account ID, Plaid public token and currency
// are required
type FiatBankAccountRequest struct {
	// PlaidAccountID is Plaid's ID of the bank account the user selected in Plaid Link
	PlaidAccountID string `json:"plaid_account_id"`
	// PlaidPublicToken is the public token Plaid Link's onSuccess callback gives, not the link token that starts Plaid
	// Link
	PlaidPublicToken string        `json:"plaid_public_token"`
	Currency         currency.Code `json:"currency"`
	// BankAccountID relinks an existing bank account, and is left empty to link a new one
	BankAccountID string `json:"bank_account_id,omitempty"`
}

// fiatBankAccountsResponse is the result of a fiat-create-bank-account or fiat-get-bank-accounts request
type fiatBankAccountsResponse struct {
	BankAccounts []FiatBankAccountResponse `json:"bank_accounts_list"`
}

// FiatBankAccountResponse is a bank account linked for fiat withdrawals; a bank account just linked reports only its ID,
// status, bank name, currency, holder name, masked account identifier, supported payment networks and times
type FiatBankAccountResponse struct {
	// OK, Code and Message are mostly null
	OK      string `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
	// ID identifies the bank account to withdraw to
	ID string `json:"id"`
	// AccountID cannot be withdrawn to
	AccountID string `json:"account_id"`
	UserUUID  string `json:"user_uuid"`
	// Status is a FiatBankAccountStatus constant
	Status      string `json:"status"`
	BankName    string `json:"bank_name"`
	BankCity    string `json:"bank_city"`
	BankCountry string `json:"bank_country"`
	// BankIdentifierType is the kind of the masked bank identifier, such as cubix_account_number_last_4_digits or
	// bic_swift
	BankIdentifierType    string `json:"bank_identifier_type"`
	BankIdentifierValue   string `json:"bank_identifier_value"`
	BankAccountHolderName string `json:"bank_account_holder_name"`
	// AccountIdentifierType is the kind of the masked account identifier, such as cubix_account_id or iban
	AccountIdentifierType         string                  `json:"account_identifier_type"`
	AccountIdentifierValue        string                  `json:"account_identifier_value"`
	AccountHolderName             string                  `json:"account_holder_name"`
	AccountType                   string                  `json:"account_type"`
	Currency                      currency.Code           `json:"currency"`
	VerifiedBy                    string                  `json:"verified_by"`
	Reason                        string                  `json:"reason"`
	SupportedPaymentNetworks      []string                `json:"supported_payment_networks"`
	WithdrawalPaymentNetworks     []string                `json:"withdrawal_payment_networks"`
	PaymentNetworkIdentifierValue string                  `json:"payment_network_identifier_value"`
	BankDetails                   *FiatBankAccountDetails `json:"bank_details"`
	CreatedAt                     types.Time              `json:"created_at"`
	UpdatedAt                     types.Time              `json:"updated_at"`
}

// FiatBankAccountDetails is the addresses and intermediary banks of a linked bank account
type FiatBankAccountDetails struct {
	InstitutionalStreetAddress string `json:"institutional_street_address"`
	BankStreetAddress          string `json:"bank_street_address"`
	AccountHolderAddress       string `json:"account_holder_address"`
	// BankAccountAddress, IntermediateBank and IntermediateBankOptions are left undecoded, as the docs leave their
	// shapes undescribed
	BankAccountAddress      json.RawMessage   `json:"bank_account_address"`
	IntermediateBank        json.RawMessage   `json:"intermediate_bank"`
	IntermediateBankOptions []json.RawMessage `json:"intermediate_bank_options"`
}
