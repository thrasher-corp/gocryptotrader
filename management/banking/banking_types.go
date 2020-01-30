package banking

import (
	"sync"
)

const (
	ErrBankAccountNotFound = "bank account ID: %v not found"
	ErrAccountCannotBeNil  = "Account cannot be nil"
	ErrBankAccountDisabled = "account is disabled"
)

// Account holds differing bank account details by supported funding
// currency
type Account struct {
	Enabled             bool   `json:"enabled"`
	ID                  string `json:"id,omitempty"`
	BankName            string `json:"bankName"`
	BankAddress         string `json:"bankAddress"`
	BankPostalCode      string `json:"bankPostalCode"`
	BankPostalCity      string `json:"bankPostalCity"`
	BankCountry         string `json:"bankCountry"`
	AccountName         string `json:"accountName"`
	AccountNumber       string `json:"accountNumber"`
	SWIFTCode           string `json:"swiftCode"`
	IBAN                string `json:"iban"`
	BSBNumber           string `json:"bsbNumber,omitempty"`
	BankCode         	float64  `json:",omitempty"`
	SupportedCurrencies string `json:"supportedCurrencies"`
	SupportedExchanges  string `json:"supportedExchanges,omitempty"`
}

var Accounts []Account
var m = &sync.Mutex{}
