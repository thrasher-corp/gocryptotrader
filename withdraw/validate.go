package withdraw

import (
	"errors"
	"strings"
)

// Valid takes interface and passes to asset type to check the request meets requirements to submit
func Valid(request *Request) (err error) {
	var allErrors []string
	if request.Amount <= 0 {
		allErrors = append(allErrors, "amount cannot be empty")
	}

	switch request.Type {
	case Fiat:
		allErrors = append(allErrors, ValidateFiat(request.Fiat)...)
	case Crypto:
		allErrors = append(allErrors, ValidateCrypto(request.Crypto)...)
		default:
			allErrors = append(allErrors, "invalid request type")

	}
	if len(allErrors) > 0 {
		err = errors.New(strings.Join(allErrors, "\n"))
	}

	return
}

// Valid takes interface and passes to asset type to check the request meets requirements to submit
func ValidateFiat(request *FiatRequest) (err []string) {
	if request == nil {
		return
	}

	if request.BankAccountNumber == "" {
		err = append(err, "BankAccountNumber cannot be empty")
	}

	if request.BSB == "" {

	}

	if request.IBAN == "" && request.SwiftCode == "" {
		err = append(err, "BankAccountNumber cannot be empty")
	}

	return err
}

// ValidateCrypto checks if Crypto request is valid and meets the minimum requirements to submit a crypto withdrawal request
func ValidateCrypto(request *CryptoRequest) (err []string) {
	if request == nil {
		err = append(err, "Cryptorequest cannot be nil on a crypto request")
		return
	}

	if request.Address == "" {
		err = append(err, "Address cannot be empty")
	}

	if request.FeeAmount < 0 {
		err = append(err, "FeeAmount cannot be a negative number")
	}

	return
}
