package withdraw

import (
	"errors"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// Valid takes interface and passes to asset type to check the request meets requirements to submit
func Valid(request interface{}) error {
	switch request.(type) {
	case *FiatRequest:
		return ValidateFiat(request.(*FiatRequest))
	case *CryptoRequest:
		return ValidateCrypto(request.(*CryptoRequest))
	default:
		return nil
	}
}

// ValidateFiat checks if Fiat request is valid
func ValidateFiat(request *FiatRequest) error {
	if request == nil {
		return nil
	}
	return nil
}

// ValidateCrypto checks if Crypto request is valid
func ValidateCrypto(request *CryptoRequest) (err error) {
	if request == nil {
		return nil
	}

	var allErrors []string
	if !request.Currency.IsCryptocurrency() {
		allErrors = append(allErrors, "currency is not a crypto currency")
	}

	if request.Amount < 0 {
		allErrors = append(allErrors, "amount must be greater than 0")
	}

	if request.Address == "" {
		allErrors = append(allErrors, "address cannot be empty")
	}

	v,_ := common.IsValidCryptoAddress(request.Address, request.Currency.String())
	if !v {
		allErrors = append(allErrors, "address is not valid")
	}

	return errors.New(strings.Join(allErrors, "\n"))
}
