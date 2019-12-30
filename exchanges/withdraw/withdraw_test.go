package withdraw

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	validFiatRequest = &FiatRequest{
		GenericInfo: GenericInfo{
			Currency:    currency.AUD,
			Description: "Test Withdrawal",
			Amount:      0.1,
		},
		BankAccountName:   "test-bank-account",
		BankAccountNumber: "test-bank-number",
		BankName:          "test-bank-name",
		BSB:               "",
		SwiftCode:         "",
		IBAN:              "",
	}

	invalidFiatRequest         = &FiatRequest{}
	invalidCurrencyFiatRequest = &FiatRequest{
		GenericInfo: GenericInfo{
			Currency: currency.BTC,
			Amount:   1,
		},
	}

	validCryptoRequest = &CryptoRequest{
		GenericInfo: GenericInfo{
			Currency:    currency.BTC,
			Description: "Test Withdrawal",
			Amount:      0.1,
		},
		Address: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
	}
	invalidCryptoRequest         = &CryptoRequest{}
	invalidCurrencyCryptoRequest = &CryptoRequest{
		GenericInfo: GenericInfo{
			Currency: currency.AUD,
			Amount:   0,
		},
	}
	invalidCryptoAddressRequest = &CryptoRequest{
		GenericInfo: GenericInfo{
			Currency:    currency.BTC,
			Description: "Test Withdrawal",
			Amount:      0.1,
		},
		Address: "1D10TH0RS3",
	}
)

func TestValid(t *testing.T) {
	testCases := []struct {
		name    string
		request interface{}
		output  interface{}
	}{
		{
			"Fiat",
			validFiatRequest,
			nil,
		},
		{
			"Crypto",
			validCryptoRequest,
			nil,
		},
		{
			"Invalid",
			nil,
			ErrInvalidRequest,
		},
	}
	for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {
			err := Valid(test.request)
			if err != nil {
				if test.output.(error).Error() != err.Error() {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestValidateFiat(t *testing.T) {
	testCases := []struct {
		name    string
		request *FiatRequest
		output  interface{}
	}{
		{
			"Valid",
			validFiatRequest,
			nil,
		},
		{
			"Invalid",
			invalidFiatRequest,
			errors.New("currency not set, amount must be greater than 0"),
		},
		{
			"NoRequest",
			nil,
			ErrRequestCannotBeNil,
		},
		{
			"CryptoCurrency",
			invalidCurrencyFiatRequest,
			errors.New("currency is not a fiat currency"),
		},
	}

	for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {
			err := ValidateFiat(test.request)
			if err != nil {
				if test.output.(error).Error() != err.Error() {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestValidateCrypto(t *testing.T) {
	testCases := []struct {
		name    string
		request *CryptoRequest
		output  interface{}
	}{
		{
			"Valid",
			validCryptoRequest,
			nil,
		},
		{
			"Invalid",
			invalidCryptoRequest,
			errors.New("currency not set, amount must be greater than 0, address cannot be empty"),
		},
		{
			"NoRequest",
			nil,
			ErrRequestCannotBeNil,
		},
		{
			"FiatCurrency",
			invalidCurrencyCryptoRequest,
			errors.New("currency is not a crypto currency, amount must be greater than 0, address cannot be empty"),
		},
		{
			"InvalidAddress",
			invalidCryptoAddressRequest,
			errors.New(ErrStrAddressisInvalid),
		},
	}

	for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {
			err := ValidateCrypto(test.request)
			if err != nil {
				if test.output.(error).Error() != err.Error() {
					t.Fatal(err)
				}
			}
		})
	}
}
