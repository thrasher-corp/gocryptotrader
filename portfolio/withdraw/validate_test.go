package withdraw

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

const (
	testBTCAddress = "0xTHISISALEGITBTCADDRESSHONEST"
)

var (
	validFiatRequest = &Request{
		Fiat:        FiatRequest{},
		Exchange:    "test-exchange",
		Currency:    currency.AUD,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Fiat,
	}

	invalidRequest = &Request{
		Exchange: "Binance",
		Type:     Fiat,
	}

	invalidCurrencyFiatRequest = &Request{
		Exchange: "Binance",
		Fiat: FiatRequest{
			Bank: banking.Account{},
		},
		Currency: currency.BTC,
		Amount:   1,
		Type:     Fiat,
	}

	validCryptoRequest = &Request{
		Exchange: "Binance",
		Crypto: CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
		Currency:    currency.BTC,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Crypto,
	}

	invalidCryptoNilRequest = &Request{
		Exchange:    "test",
		Currency:    currency.BTC,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Crypto,
	}

	invalidCryptoNegativeFeeRequest = &Request{
		Exchange: "Binance",
		Crypto: CryptoRequest{
			Address:   core.BitcoinDonationAddress,
			FeeAmount: -0.1,
		},
		Currency:    currency.BTC,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Crypto,
	}

	invalidCurrencyCryptoRequest = &Request{
		Exchange: "Binance",
		Crypto: CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
		Currency: currency.AUD,
		Amount:   0,
		Type:     Crypto,
	}

	invalidCryptoNoAddressRequest = &Request{
		Exchange:    "test",
		Crypto:      CryptoRequest{},
		Currency:    currency.BTC,
		Description: "Test Withdrawal",
		Amount:      0.1,
		Type:        Crypto,
	}

	invalidType = &Request{
		Exchange: "test",
		Type:     Unknown,
		Currency: currency.BTC,
		Amount:   0.1,
	}
)

func TestMain(m *testing.M) {
	var p portfolio.Base
	err := p.AddAddress(core.BitcoinDonationAddress, "test", currency.BTC, 1500)
	if err != nil {
		fmt.Printf("failed to add portfolio address with reason: %v, unable to continue tests", err)
		os.Exit(0)
	}
	p.Addresses[0].WhiteListed = true
	p.Addresses[0].ColdStorage = true
	p.Addresses[0].SupportedExchanges = "BTC Markets,Binance"

	err = p.AddAddress(testBTCAddress, "test", currency.BTC, 1500)
	if err != nil {
		fmt.Printf("failed to add portfolio address with reason: %v, unable to continue tests", err)
		os.Exit(0)
	}
	p.Addresses[1].SupportedExchanges = "BTC Markets,Binance"
	banking.AppendAccounts(banking.Account{
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
	err := invalidType.Validate()
	if err != nil {
		if err.Error() != ErrInvalidRequest.Error() {
			t.Fatal(err)
		}
	}
}

func TestExchangeNameUnset(t *testing.T) {
	r := Request{}
	err := r.Validate()
	if err != nil {
		if err != common.ErrExchangeNameNotSet {
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
		output        any
		validate      validate.Checker
	}{
		{
			"Valid",
			validFiatRequest,
			Fiat,
			"test-bank-01",
			nil,
			nil,
		},
		{
			"Valid-With-Good-Exchange-Option",
			validFiatRequest,
			Fiat,
			"test-bank-01",
			nil,
			validate.Check(func() error { return nil }),
		},
		{
			"Valid-With-bad-Exchange-Option",
			validFiatRequest,
			Fiat,
			"test-bank-01",
			errors.New("error"),
			validate.Check(func() error { return errors.New("error") }),
		},
		{
			"Invalid",
			invalidRequest,
			Fiat,
			"",
			errors.New(ErrStrAmountMustBeGreaterThanZero + ", " +
				ErrStrNoCurrencySet + ", " +
				banking.ErrBankAccountDisabled + ", " +
				fmt.Sprintf("Exchange %s not supported by bank account",
					invalidRequest.Exchange) + ", " +
				banking.ErrAccountCannotBeEmpty + ", " +
				banking.ErrIBANSwiftNotSet),
			nil,
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
			errors.New(ErrStrCurrencyNotFiat + ", " +
				banking.ErrBankAccountDisabled + ", " +
				fmt.Sprintf("Exchange %s not supported by bank account",
					invalidRequest.Exchange) + ", " +
				banking.ErrAccountCannotBeEmpty + ", " +
				banking.ErrCurrencyNotSupportedByAccount + ", " +
				banking.ErrIBANSwiftNotSet),
			nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.requestType < 3 {
				test.request.Type = test.requestType
			}
			if test.bankAccountID != "" {
				v, err := banking.GetBankAccountByID(test.bankAccountID)
				if err != nil {
					t.Fatal(err)
				}
				test.request.Fiat.Bank = *v
			}
			err := test.request.Validate(test.validate)
			if err != nil {
				errOutput, ok := test.output.(error)
				if !ok {
					t.Fatalf("Test Name %s unable to type assert error", test.name)
				}
				if errOutput.Error() != err.Error() {
					t.Fatalf("Test Name %s expecting error [%s] but received [%s]", test.name, errOutput.Error(), err)
				}
			}
		})
	}
}

func TestValidateCrypto(t *testing.T) {
	testCases := []struct {
		name    string
		request *Request
		output  any
	}{
		{
			"Valid",
			validCryptoRequest,
			nil,
		},
		{
			"Invalid",
			invalidRequest,
			errors.New(ErrStrAmountMustBeGreaterThanZero + ", " +
				ErrStrNoCurrencySet + ", " +
				banking.ErrBankAccountDisabled + ", " +
				fmt.Sprintf("Exchange %s not supported by bank account",
					invalidRequest.Exchange) + ", " +
				banking.ErrAccountCannotBeEmpty + ", " +
				banking.ErrIBANSwiftNotSet),
		},
		{
			"Invalid-Nil",
			invalidCryptoNilRequest,
			errors.New(ErrStrAddressNotSet),
		},
		{
			"NoRequest",
			nil,
			ErrRequestCannotBeNil,
		},
		{
			"FiatCurrency",
			invalidCurrencyCryptoRequest,
			errors.New(ErrStrAmountMustBeGreaterThanZero + ", " +
				ErrStrCurrencyNotCrypto),
		},
		{
			"NoAddress",
			invalidCryptoNoAddressRequest,
			errors.New(
				ErrStrAddressNotSet),
		},
		{
			"NegativeFee",
			invalidCryptoNegativeFeeRequest,
			errors.New(ErrStrFeeCannotBeNegative),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := test.request.Validate()
			if err != nil {
				tErr, _ := test.output.(error)
				if err.Error() != tErr.Error() {
					t.Fatalf("Test Name %s expecting error [%v] but received [%s]",
						test.name,
						tErr,
						err)
				}
			}
		})
	}
}
