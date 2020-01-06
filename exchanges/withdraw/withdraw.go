package withdraw

import (
	"errors"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Valid takes interface and passes to asset type to check the request meets requirements to submit
func Valid(request interface{}) error {
	switch request := request.(type) {
	case *FiatRequest:
		return ValidateFiat(request)
	case *CryptoRequest:
		return ValidateCrypto(request)
	default:
		return ErrInvalidRequest
	}
}

// ValidateFiat checks if Fiat request is valid
func ValidateFiat(request *FiatRequest) (err error) {
	if request == nil {
		return ErrRequestCannotBeNil
	}

	var allErrors []string
	if (request.Currency != currency.Code{}) {
		if !request.Currency.IsFiatCurrency() {
			allErrors = append(allErrors, "currency is not a fiat currency")
		}
	} else {
		allErrors = append(allErrors, ErrStrNoCurrencySet)
	}

	if request.Amount <= 0 {
		allErrors = append(allErrors, ErrStrAmountMustBeGreaterThanZero)
	}

	if len(allErrors) > 0 {
		err = errors.New(strings.Join(allErrors, ", "))
	}
	return err
}

// ValidateCrypto checks if Crypto request is valid
func ValidateCrypto(request *CryptoRequest) (err error) {
	if request == nil {
		return ErrRequestCannotBeNil
	}

	var allErrors []string
	if (request.Currency != currency.Code{}) {
		if !request.Currency.IsCryptocurrency() {
			allErrors = append(allErrors, "currency is not a crypto currency")
		}
	} else {
		allErrors = append(allErrors, ErrStrNoCurrencySet)
	}

	if request.Amount <= 0 {
		allErrors = append(allErrors, ErrStrAmountMustBeGreaterThanZero)
	}

	if request.Address == "" {
		allErrors = append(allErrors, ErrStrAddressNotSet)
	}

	if len(allErrors) > 0 {
		err = errors.New(strings.Join(allErrors, ", "))
	}
	return err
}
