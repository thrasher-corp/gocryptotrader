package bank

import (
	"fmt"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
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

// SetAccounts safely overwrites bank account slice
func SetAccounts(accs ...Account) {
	m.Lock()
	defer m.Unlock()
	accounts = accs
}

// AppendAccounts safely adds to bank account slice
func AppendAccounts(accs ...Account) {
	m.Lock()
	defer m.Unlock()
accountRange:
	for j := range accs {
		for i := range accounts {
			if accounts[i].AccountNumber == accs[j].AccountNumber {
				continue accountRange
			}
		}
		accounts = append(accounts, accs[j])
	}
}

// GetBankAccountByID Returns a bank account based on its ID
func GetBankAccountByID(id string) (*Account, error) {
	m.Lock()
	defer m.Unlock()
	for x := range accounts {
		if strings.EqualFold(accounts[x].ID, id) {
			return &accounts[x], nil
		}
	}
	return nil, fmt.Errorf(ErrBankAccountNotFound, id)
}

// ExchangeSupported Checks if exchange is supported by bank account
func (b *Account) ExchangeSupported(exchange string) bool {
	exchList := strings.Split(b.SupportedExchanges, ",")
	return common.StringDataCompareInsensitive(exchList, exchange)
}

// Validate validates bank account settings
func (b *Account) Validate() error {
	if b.BankName == "" ||
		b.BankAddress == "" ||
		b.BankPostalCode == "" ||
		b.BankPostalCity == "" ||
		b.BankCountry == "" ||
		b.AccountName == "" ||
		b.SupportedCurrencies == "" {
		return fmt.Errorf(
			"banking details for %s is enabled but variables not set correctly",
			b.BankName)
	}

	if b.SupportedExchanges == "" {
		b.SupportedExchanges = "ALL"
	}

	if strings.Contains(strings.ToUpper(
		b.SupportedCurrencies),
		currency.AUD.String()) {
		if b.BSBNumber == "" {
			return fmt.Errorf(
				"banking details for %s is enabled but BSB/SWIFT values not set",
				b.BankName)
		}
	} else {
		if b.IBAN == "" && b.SWIFTCode == "" {
			return fmt.Errorf(
				"banking details for %s is enabled but SWIFT/IBAN values not set",
				b.BankName)
		}
	}
	return nil
}

// ValidateForWithdrawal confirms bank account meets minimum requirements to submit
// a withdrawal request
func (b *Account) ValidateForWithdrawal(exchange string, cur currency.Code) (err []string) {
	if !b.Enabled {
		err = append(err, ErrBankAccountDisabled)
	}
	if !b.ExchangeSupported(exchange) {
		err = append(err, "Exchange "+exchange+" not supported by bank account")
	}

	if b.AccountNumber == "" {
		err = append(err, ErrAccountCannotBeEmpty)
	}

	if !common.StringDataCompareInsensitive(strings.Split(b.SupportedCurrencies, ","), cur.String()) {
		err = append(err, ErrCurrencyNotSupportedByAccount)
	}

	if cur.Upper() == currency.AUD {
		if b.BSBNumber == "" {
			err = append(err, ErrBSBRequiredForAUD)
		}
	} else {
		if b.IBAN == "" && b.SWIFTCode == "" {
			err = append(err, ErrIBANSwiftNotSet)
		}
	}
	return
}
