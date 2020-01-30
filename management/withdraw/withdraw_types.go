package withdraw

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/cache"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/management/banking"
)

type RequestType uint8

const (
	Crypto RequestType = iota
	Fiat
)

const (
	ErrStrAmountMustBeGreaterThanZero = "amount must be greater than 0"
	// ErrStrAddressisInvalid message to return when address is invalid for crypto request
	ErrStrAddressisInvalid = "address is not valid"
	// ErrStrAddressNotSet message to return when address is empty
	ErrStrAddressNotSet = "address cannot be empty"
	// ErrStrNoCurrencySet message to return when no currency is set
	ErrStrNoCurrencySet = "currency not set"
	// ErrStrCurrencyNotCrypto message to return when requested currency is not crypto
	ErrStrCurrencyNotCrypto = "requested currency is not a cryptocurrency"
	// ErrStrCurrencyNotFiat message to return when requested currency is not fiat
	ErrStrCurrencyNotFiat = "requested currency is not fiat"
)

var (
	// ErrRequestCannotBeNil message to return when a request is nil
	ErrRequestCannotBeNil = errors.New("request cannot be nil")
	// ErrInvalidRequest message to return when a request type is invalid
	ErrInvalidRequest = errors.New("invalid request type")
	// Cache LRU cache for recent requests
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
	Bank *banking.Account

	IsExpressWire bool
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
	Exchange    string
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

	Exchange       *ExchangeResponse
	RequestDetails *Request

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ExchangeResponse struct {
	Name   string
	ID     string
	Status string
}
