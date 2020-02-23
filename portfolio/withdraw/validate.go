package withdraw

import (
	"errors"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// Validate takes interface and passes to asset type to check the request meets requirements to submit
func Validate(request *Request) (err error) {
	if request == nil {
		return ErrRequestCannotBeNil
	}

	var allErrors []string
	if request.Amount <= 0 {
		allErrors = append(allErrors, ErrStrAmountMustBeGreaterThanZero)
	}

	if (request.Currency == currency.Code{}) {
		allErrors = append(allErrors, ErrStrNoCurrencySet)
	}

	switch request.Type {
	case Fiat:
		if request.Fiat == nil {
			return ErrInvalidRequest
		}
		if (request.Currency != currency.Code{}) && !request.Currency.IsFiatCurrency() {
			allErrors = append(allErrors, ErrStrCurrencyNotFiat)
		}
		allErrors = append(allErrors, validateFiat(request)...)
	case Crypto:
		if request.Crypto == nil {
			return ErrInvalidRequest
		}
		if (request.Currency != currency.Code{}) && !request.Currency.IsCryptocurrency() {
			allErrors = append(allErrors, ErrStrCurrencyNotCrypto)
		}
		allErrors = append(allErrors, validateCrypto(request)...)
	default:
		allErrors = append(allErrors, "invalid request type")
	}
	if len(allErrors) > 0 {
		return errors.New(strings.Join(allErrors, ", "))
	}
	return nil
}

// validateFiat takes interface and passes to asset type to check the request meets requirements to submit
func validateFiat(request *Request) (err []string) {
	errBank := request.Fiat.Bank.ValidateForWithdrawal(request.Exchange, request.Currency)
	if errBank != nil {
		err = append(err, errBank...)
	}
	return err
}

// validateCrypto checks if Crypto request is valid and meets the minimum requirements to submit a crypto withdrawal request
func validateCrypto(request *Request) (err []string) {
	if !portfolio.IsWhiteListed(request.Crypto.Address) {
		err = append(err, ErrStrAddressNotWhiteListed)
	}

	if !portfolio.IsExchangeSupported(request.Exchange, request.Crypto.Address) {
		err = append(err, ErrStrExchangeNotSupportedByAddress)
	}

	if request.Crypto.Address == "" {
		err = append(err, ErrStrAddressNotSet)
	}

	if request.Crypto.FeeAmount < 0 {
		err = append(err, ErrStrFeeCannotBeNegative)
	}
	return
}
