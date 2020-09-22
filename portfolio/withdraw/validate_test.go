package withdraw

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

const (
	testBTCAddress = "0xTHISISALEGITBTCADDRESSHONEST"
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

	invalidCryptoNilRequest = &Request{
		Currency:    currency.BTC,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Crypto,
	}

	invalidCryptoNegativeFeeRequest = &Request{
		Crypto: &CryptoRequest{
			Address:   core.BitcoinDonationAddress,
			FeeAmount: -0.1,
		},
		Currency:    currency.BTC,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Crypto,
	}

	invalidCurrencyCryptoRequest = &Request{
		Crypto: &CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
		Currency: currency.AUD,
		Amount:   0,
		Type:     Crypto,
	}

	invalidCryptoNoAddressRequest = &Request{
		Crypto:      &CryptoRequest{},
		Currency:    currency.BTC,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Crypto,
	}

	invalidCryptoNonWhiteListedAddressRequest = &Request{
		Crypto: &CryptoRequest{
			Address: testBTCAddress,
		},
		Currency:    currency.BTC,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Crypto,
	}

	invalidType = &Request{
		Type:     Unknown,
		Currency: currency.BTC,
		Amount:   0.1,
	}
)

func TestMain(m *testing.M) {
	err := portfolio.Portfolio.AddAddress(core.BitcoinDonationAddress, "test", currency.BTC, 1500)
	if err != nil {
		fmt.Printf("failed to add portfolio address with reason: %v, unable to continue tests", err)
		os.Exit(0)
	}
	portfolio.Portfolio.Addresses[0].WhiteListed = true
	portfolio.Portfolio.Addresses[0].ColdStorage = true
	portfolio.Portfolio.Addresses[0].SupportedExchanges = "BTC Markets,Binance"

	err = portfolio.Portfolio.AddAddress(testBTCAddress, "test", currency.BTC, 1500)
	if err != nil {
		fmt.Printf("failed to add portfolio address with reason: %v, unable to continue tests", err)
		os.Exit(0)
	}
	portfolio.Portfolio.Addresses[1].SupportedExchanges = "BTC Markets,Binance"

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
			SupportedCurrencies: "AUD,USD",
			SupportedExchanges:  "test-exchange",
		},
	)

	os.Exit(m.Run())
}

func TestValid(t *testing.T) {
	err := Validate(invalidType)
	if err != nil {
		if err.Error() != ErrInvalidRequest.Error() {
			t.Fatal(err)
		}
	}
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
			errors.New(ErrStrCurrencyNotFiat + ", " + banking.ErrBankAccountDisabled + ", " + banking.ErrAccountCannotBeEmpty + ", " + banking.ErrCurrencyNotSupportedByAccount + ", " + banking.ErrIBANSwiftNotSet),
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
			err := Validate(test.request)
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
			"Invalid-Nil",
			invalidCryptoNilRequest,
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
			errors.New(ErrStrAmountMustBeGreaterThanZero + ", " + ErrStrCurrencyNotCrypto),
		},
		{
			"NoAddress",
			invalidCryptoNoAddressRequest,
			errors.New(ErrStrAddressNotWhiteListed + ", " + ErrStrExchangeNotSupportedByAddress + ", " + ErrStrAddressNotSet),
		},
		{
			"NonWhiteListed",
			invalidCryptoNonWhiteListedAddressRequest,
			errors.New(ErrStrAddressNotWhiteListed),
		},
		{
			"NegativeFee",
			invalidCryptoNegativeFeeRequest,
			errors.New(ErrStrFeeCannotBeNegative),
		},
	}

	for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.request)
			if err != nil {
				if err.Error() != test.output.(error).Error() {
					t.Fatal(err)
				}
			}
		})
	}
}
