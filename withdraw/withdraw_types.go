package withdraw

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Type uint8
const (
	Crypto Type = iota
	Fiat
)

// CryptoRequest stores the info required for a crypto withdrawal request
type CryptoRequest struct {
	Address    string
	AddressTag string
	FeeAmount  float64
}

// FiatRequest used for fiat withdrawal requests
type FiatRequest struct {
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

type Request struct {
	Currency    currency.Code
	Description string
	Amount      float64
	Type        Type

	TradePassword string
	OneTimePassword int64
	PIN int64

	Crypto *CryptoRequest
	Fiat   *FiatRequest
}


type Response struct {
	ID uuid.UUID

	ExchangeID	string
	Status string

	RequestDetails	*Request

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"created_at"`
}

