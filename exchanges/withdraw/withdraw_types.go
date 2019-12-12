package withdraw

import "github.com/thrasher-corp/gocryptotrader/currency"

// GenericWithdrawRequestInfo stores genric withdraw request info
type GenericWithdrawRequestInfo struct {
	// General withdraw information
	Currency        currency.Code
	Description     string
	OneTimePassword int64
	AccountID       string
	PIN             int64
	TradePassword   string
	Amount          float64
}

// CryptoWithdrawRequest stores the info required for a crypto withdrawal request
type CryptoWithdrawRequest struct {
	GenericWithdrawRequestInfo
	// Crypto related information
	Address    string
	AddressTag string
	FeeAmount  float64
}

// FiatWithdrawRequest used for fiat withdrawal requests
type FiatWithdrawRequest struct {
	GenericWithdrawRequestInfo
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
