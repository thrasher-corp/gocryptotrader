package withdraw

import (
	"errors"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/management/banking"
)

var (
	validFiatRequest = &Request{
		Fiat:        &FiatRequest{},
		Exchange:    "test-exchange",
		Currency:    currency.AUD,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Fiat,
	}

	invalidRequest = &Request{
		Type: Fiat,
	}

	invalidCurrencyFiatRequest = &Request{
		Fiat: &FiatRequest{
			Bank: &banking.Account{},
		},
		Currency: currency.BTC,
		Amount:   1,
		Type:     Fiat,
	}

	validCryptoRequest = &Request{
		Crypto: &CryptoRequest{
			Address: core.BitcoinDonationAddress,
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

func TestMain(m *testing.M) {
	banking.Accounts = append(banking.Accounts,
		banking.Account{
			Enabled:             true,
			ID:                  "test-bank-01",
			BankName:            "Test Bank",
			BankAddress:         "42 Bank Street",
			BankPostalCode:      "13337",
			BankPostalCity:      "Satoshiville",
			BankCountry:         "Japan",
			AccountName:         "Satoshi Nakamoto",
			AccountNumber:       "0234",
			BSBNumber:           "123456",
			SWIFTCode:           "91272837",
			IBAN:                "98218738671897",
			SupportedCurrencies: "USD",
			SupportedExchanges:  "test-exchange",
		},
	)

	os.Exit(m.Run())
}

func TestValidateFiat(t *testing.T) {
	testCases := []struct {
		name          string
		request       *Request
		requestType   RequestType
		bankAccountID string
		output        interface{}
	}{
		{
			"Valid",
			validFiatRequest,
			Fiat,
			"test-bank-01",
			nil,
		},
		{
			"Invalid",
			invalidRequest,
			Fiat,
			"",
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
			"",
			errors.New("requested currency is not fiat, Bank Account is disabled, Bank Account Number cannot be empty, IBAN/SWIFT values not set"),
		},
	}

	for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {
			if test.requestType < 3 {
				test.request.Type = test.requestType
			}
			if test.bankAccountID != "" {
				v, err := banking.GetBankAccountByID(test.bankAccountID)
				if err != nil {
					t.Fatal(err)
				}
				test.request.Fiat.Bank = v
			}
			err := Valid(test.request)
			if err != nil {
				t.Log(err)
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
