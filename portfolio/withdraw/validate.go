package withdraw

import (
	"errors"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
)

// Validate takes interface and passes to asset type to check the request meets requirements to submit
func (r *Request) Validate(opt ...validate.Checker) (err error) {
	if r == nil {
		return ErrRequestCannotBeNil
	}

	if r.Exchange == "" {
		return common.ErrExchangeNameNotSet
	}

	var allErrors []string
	if r.Amount <= 0 {
		allErrors = append(allErrors, ErrStrAmountMustBeGreaterThanZero)
	}

	if r.Currency.Equal(currency.EMPTYCODE) {
		allErrors = append(allErrors, ErrStrNoCurrencySet)
	}

	switch r.Type {
	case Fiat:
		if !r.Currency.Equal(currency.EMPTYCODE) && !r.Currency.IsFiatCurrency() {
			allErrors = append(allErrors, ErrStrCurrencyNotFiat)
		}
		allErrors = append(allErrors, r.validateFiat()...)
	case Crypto:
		if !r.Currency.Equal(currency.EMPTYCODE) && !r.Currency.IsCryptocurrency() {
			allErrors = append(allErrors, ErrStrCurrencyNotCrypto)
		}
		allErrors = append(allErrors, r.validateCrypto()...)
	default:
		allErrors = append(allErrors, "invalid request type")
	}

	for _, o := range opt {
		if o == nil {
			continue
		}
		err := o.Check()
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if len(allErrors) > 0 {
		return errors.New(strings.Join(allErrors, ", "))
	}
	return nil
}

// validateFiat takes interface and passes to asset type to check the request meets requirements to submit
func (r *Request) validateFiat() []string {
	var resp []string
	errBank := r.Fiat.Bank.ValidateForWithdrawal(r.Exchange, r.Currency)
	if errBank != nil {
		resp = append(resp, errBank...)
	}
	return resp
}

// validateCrypto checks if Crypto request is valid and meets the minimum requirements to submit a crypto withdrawal request
func (r *Request) validateCrypto() []string {
	var resp []string

	if r.Crypto.Address == "" {
		resp = append(resp, ErrStrAddressNotSet)
	}

	if r.Crypto.FeeAmount < 0 {
		resp = append(resp, ErrStrFeeCannotBeNegative)
	}
	return resp
}
