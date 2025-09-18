package withdraw

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/cache"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

// RequestType used for easy matching of int type to Word
type RequestType int64

const (
	// Crypto request type
	Crypto RequestType = iota
	// Fiat request type
	Fiat
	// Unknown request type
	Unknown
)

const (
	// ErrStrAmountMustBeGreaterThanZero message to return when requested amount is less than 0
	ErrStrAmountMustBeGreaterThanZero = "amount must be greater than 0"
	// ErrStrAddressNotSet message to return when address is empty
	ErrStrAddressNotSet = "address cannot be empty"
	// ErrStrNoCurrencySet message to return when no currency is set
	ErrStrNoCurrencySet = "currency not set"
	// ErrStrCurrencyNotCrypto message to return when requested currency is not crypto
	ErrStrCurrencyNotCrypto = "requested currency is not a cryptocurrency"
	// ErrStrCurrencyNotFiat message to return when requested currency is not fiat
	ErrStrCurrencyNotFiat = "requested currency is not fiat"
	// ErrStrFeeCannotBeNegative message to return when fee amount is negative
	ErrStrFeeCannotBeNegative = "fee amount cannot be negative"
)

var (
	// ErrRequestCannotBeNil message to return when a request is nil
	ErrRequestCannotBeNil = errors.New("request cannot be nil")
	// ErrInvalidRequest message to return when a request type is invalid
	ErrInvalidRequest = errors.New("invalid request type")
	// ErrStrAddressNotWhiteListed occurs when a withdrawal attempts to withdraw from a non-whitelisted address
	ErrStrAddressNotWhiteListed = errors.New("address is not whitelisted for withdrawals")
	// ErrStrExchangeNotSupportedByAddress message to return when attemptign to withdraw to an unsupported exchange
	ErrStrExchangeNotSupportedByAddress = errors.New("address is not supported by exchange")
	// CacheSize cache size to use for withdrawal request history
	CacheSize uint64 = 25
	// Cache LRU cache for recent requests
	Cache = cache.New(CacheSize)
	// DryRunID uuid to use for dryruns
	DryRunID, _ = uuid.FromString("3e7e2c25-5a0b-429b-95a1-0960079dce56")
)

// CryptoRequest stores the info required for a crypto withdrawal request
type CryptoRequest struct {
	Address    string
	AddressTag string
	Chain      string
	FeeAmount  float64
}

// FiatRequest used for fiat withdrawal requests
type FiatRequest struct {
	Bank banking.Account

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

// TravelAddress holds the address information required for travel rule compliance
type TravelAddress struct {
	Address1   string
	Address2   string
	Address3   string
	City       string
	State      string
	Country    string
	PostalCode string
}

// TravelRule stores the information that may need to be provided to comply with local regulations
type TravelRule struct {
	BeneficiaryWalletType           string
	IsSelf                          bool
	BeneficiaryName                 string
	BeneficiaryAddress              TravelAddress
	BeneficiaryFinancialInstitution string
	TransferPurpose                 string
}

// Request holds complete details for request
type Request struct {
	Exchange    string        `json:"exchange"`
	Currency    currency.Code `json:"currency"`
	Description string        `json:"description"`
	Amount      float64       `json:"amount"`
	Type        RequestType   `json:"type"`

	ClientOrderID string `json:"clientID"`

	WalletID string `json:"walletID"`

	// Used exclusively in OKX to classify internal represented by '3' or on chain represented by '4'
	InternalTransfer bool

	TradePassword   string
	OneTimePassword int64
	PIN             int64

	Crypto CryptoRequest `json:"crypto"`
	Fiat   FiatRequest   `json:"fiat"`

	Travel TravelRule `json:"travel_rule"`

	IdempotencyToken string
}

// Response holds complete details for Response
type Response struct {
	ID uuid.UUID `json:"id"`

	Exchange       ExchangeResponse `json:"exchange"`
	RequestDetails Request          `json:"request_details"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ExchangeResponse holds information returned from an exchange
type ExchangeResponse struct {
	Name   string `json:"name"`
	UUID   uuid.UUID
	ID     string `json:"id"`
	Status string `json:"status"`
}
