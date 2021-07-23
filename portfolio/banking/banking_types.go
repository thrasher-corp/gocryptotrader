package banking

import (
	"sync"
)

const (
	// ErrBankAccountNotFound message to return when bank account was not found
	ErrBankAccountNotFound = "bank account ID: %v not found"
	// ErrAccountCannotBeEmpty message to return when bank account number is empty
	ErrAccountCannotBeEmpty = "Bank Account Number cannot be empty"
	// ErrBankAccountDisabled message to return when bank account is disabled
	ErrBankAccountDisabled = "Bank Account is disabled"
	// ErrBSBRequiredForAUD message to return when currency is AUD but no bsb is set
	ErrBSBRequiredForAUD = "BSB must be set for AUD values"
	// ErrIBANSwiftNotSet message to return when no iban or swift value set
	ErrIBANSwiftNotSet = "IBAN/SWIFT values not set"
	// ErrCurrencyNotSupportedByAccount message to return when the requested
	// currency is not supported by the bank account
	ErrCurrencyNotSupportedByAccount = "requested currency is not supported by account"
)

var (
	accounts []Account
	m        sync.Mutex
)

// Account holds differing bank account details by supported funding
// currency
type Account struct {
	Enabled             bool    `json:"enabled"`
	ID                  string  `json:"id,omitempty"`
	BankName            string  `json:"bankName"`
	BankAddress         string  `json:"bankAddress"`
	BankPostalCode      string  `json:"bankPostalCode"`
	BankPostalCity      string  `json:"bankPostalCity"`
	BankCountry         string  `json:"bankCountry"`
	AccountName         string  `json:"accountName"`
	AccountNumber       string  `json:"accountNumber"`
	SWIFTCode           string  `json:"swiftCode"`
	IBAN                string  `json:"iban"`
	BSBNumber           string  `json:"bsbNumber,omitempty"`
	BankCode            float64 `json:"bank_code,omitempty"`
	SupportedCurrencies string  `json:"supportedCurrencies"`
	SupportedExchanges  string  `json:"supportedExchanges,omitempty"`
}
