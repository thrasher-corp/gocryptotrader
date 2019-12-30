package withdraw

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

const (
	ErrStrAmountMustBeGreaterThanZero = "amount must be greater than 0"
	ErrStrAddressisInvalid            = "address is not valid"
	ErrStrNoCurrecnySet               = "currency not set"
	ErrStrAddressNotSet               = "address cannot be empty"
)

var (
	ErrRequestCannotBeNil = errors.New("request cannot be nil")
	ErrInvalidRequest     = errors.New("invalid request type")
)

// GenericInfo stores genric withdraw request info
type GenericInfo struct {
	// General withdraw information
	Currency        currency.Code
	Description     string
	OneTimePassword int64
	AccountID       string
	PIN             int64
	TradePassword   string
	Amount          float64
}

// CryptoRequest stores the info required for a crypto withdrawal request
type CryptoRequest struct {
	GenericInfo
	// Crypto related information
	Address    string
	AddressTag string
	FeeAmount  float64
}

// FiatRequest used for fiat withdrawal requests
type FiatRequest struct {
	GenericInfo
	// FIAT related information
	BankAccountName   string
	BankAccountNumber string
	BankName          string
	BankAddress       string
	BankCity          string
	BankCountry       string
	BankPostalCode    string
	BSB               string
	SwiftCode         string
	IBAN              string
	BankCode          float64
	IsExpressWire     bool
	// Intermediary bank information
	RequiresIntermediaryBank      bool
	IntermediaryBankAccountNumber float64
	IntermediaryBankName          string
	IntermediaryBankAddress       string
	IntermediaryBankCity          string
	IntermediaryBankCountry       string
	IntermediaryBankPostalCode    string
	IntermediarySwiftCode         string
	IntermediaryBankCode          float64
	IntermediaryIBAN              string
	WireCurrency                  string
}
