package withdraw

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	validFiatRequest = &Request{
		Fiat: &FiatRequest{
			BankAccountName:   "test-bank-account",
			BankAccountNumber: "test-bank-number",
			BankName:          "test-bank-name",
			BSB:               "123456",
			SwiftCode:         "",
			IBAN:              "",
		},
		Currency:    currency.AUD,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Fiat,
	}

	invalidRequest = &Request{
		Type: Fiat,
	}

	invalidCurrencyFiatRequest = &Request{
		Fiat:     &FiatRequest{},
		Currency: currency.BTC,
		Amount:   1,
		Type:     Fiat,
	}

	validCryptoRequest = &Request{
		Crypto: &CryptoRequest{
			Address: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		},
		Currency:    currency.BTC,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Crypto,
	}

	invalidCurrencyCryptoRequest = &Request{
		Crypto:   &CryptoRequest{},
		Currency: currency.AUD,
		Amount:   0,
		Type:     Crypto,
	}

	invalidCryptoAddressRequest = &Request{
		Crypto: &CryptoRequest{
			Address: "1D10TH0RS3",
		},
		Currency:    currency.BTC,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Crypto,
	}
)

// func TestValid(t *testing.T) {
// 	testCases := []struct {
// 		name    string
// 		request interface{}
// 		output  interface{}
// 	}{
// 		{
// 			"Fiat",
// 			validFiatRequest,
// 			nil,
// 		},
// 		{
// 			"Crypto",
// 			validCryptoRequest,
// 			nil,
// 		},
// 		{
// 			"Invalid",
// 			nil,
// 			ErrInvalidRequest,
// 		},
// 	}
// 	for _, tests := range testCases {
// 		test := tests
// 		t.Run(test.name, func(t *testing.T) {
// 			err := Valid(test.request.(*Request))
// 			if err != nil {
// 				// if test.output.(error).Error() != err.Error() {
// 				 	t.Fatal(err)
// 				// }
// 			}
// 		})
// 	}
// }

func TestValidateFiat(t *testing.T) {
	testCases := []struct {
		name        string
		request     *Request
		requestType RequestType
		output      interface{}
	}{
		{
			"Valid",
			validFiatRequest,
			Fiat,
			nil,
		},
		{
			"Invalid",
			invalidRequest,
			Fiat,
			errors.New("invalid request type"),
		},
		{
			name:        "NoRequest",
			request:     nil,
			requestType: 3,
			output:      ErrRequestCannotBeNil,
		},
		{
			"CryptoCurrency",
			invalidCurrencyFiatRequest,
			Fiat,
			errors.New("requested currency is not fiat, Bank Account Number cannot be empty, IBAN or Swift must be set"),
		},
	}

	for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {
			if test.requestType < 3 {
				test.request.Type = test.requestType
			}
			err := Valid(test.request)
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
		request *Request
		output  interface{}
	}{
		{
			"Valid",
			validCryptoRequest,
			nil,
		},
		{
			"Invalid",
			invalidRequest,
			ErrInvalidRequest,
		},
		{
			"NoRequest",
			nil,
			ErrRequestCannotBeNil,
		},
		{
			"FiatCurrency",
			invalidCurrencyCryptoRequest,
			errors.New("amount must be greater than 0, requested currency is not a cryptocurrency, Address cannot be empty"),
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
			err := Valid(test.request)
			if err != nil {
				if test.output.(error).Error() != err.Error() {
					t.Fatal(err)
				}
			}
		})
	}
}
