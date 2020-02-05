package withdraw

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

const (
	// ErrStrAmountMustBeGreaterThanZero message to return when withdraw amount is less than 0
	ErrStrAmountMustBeGreaterThanZero = "amount must be greater than 0"
	// ErrStrAddressisInvalid message to return when address is invalid for crypto request
	ErrStrAddressisInvalid = "address is not valid"
	// ErrStrAddressNotSet message to return when address is empty
	ErrStrAddressNotSet = "address cannot be empty"
	// ErrStrNoCurrencySet message to return when no currency is set
	ErrStrNoCurrencySet = "currency not set"
)

var (
	// ErrRequestCannotBeNil message to return when a request is nil
	ErrRequestCannotBeNil = errors.New("request cannot be nil")
	// ErrInvalidRequest message to return when a request is invalid
	ErrInvalidRequest = errors.New("invalid request type")
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
