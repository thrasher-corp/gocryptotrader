package withdraw

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/cache"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

type RequestType uint8

const (
	Crypto RequestType = iota
	Fiat
)

const (
	ErrStrAmountMustBeGreaterThanZero = "amount must be greater than 0"
	ErrStrAddressisInvalid            = "address is not valid"
	ErrStrNoCurrencySet               = "currency not set"
	ErrStrAddressNotSet               = "address cannot be empty"
	ErrStrCurrencyNotFiat             = "requested currency is not fiat"
	ErrStrCurrencyNotCrypto           = "requested currency is not a cryptocurrency"
)

var (
	ErrRequestCannotBeNil = errors.New("request cannot be nil")
	ErrInvalidRequest     = errors.New("invalid request type")
	Cache = cache.New(50)
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
	Type        RequestType

	TradePassword   string
	OneTimePassword int64
	PIN             int64

	Crypto *CryptoRequest
	Fiat   *FiatRequest
}

type Response struct {
	ID uuid.UUID

	ExchangeID string
	Status     string

	RequestDetails *Request

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
