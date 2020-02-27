package banking

import (
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	testBankAccounts = []Account{
		{
			Enabled:             true,
			ID:                  "valid-test-bank-01",
			BankName:            "Test Bank",
			BankAddress:         "42 Bank Street",
			BankPostalCode:      "13337",
			BankPostalCity:      "Satoshiville",
			BankCountry:         "Japan",
			AccountName:         "Satoshi Nakamoto",
			AccountNumber:       "0234",
			SWIFTCode:           "91272837",
			BSBNumber:           "123456",
			IBAN:                "98218738671897",
			SupportedCurrencies: "AUD,USD",
			SupportedExchanges:  "test-exchange",
		},
		{
			Enabled:             false,
			ID:                  "invalid-test-bank-01",
			BankName:            "",
			BankAddress:         "",
			BankPostalCode:      "",
			BankPostalCity:      "",
			BankCountry:         "",
			AccountName:         "",
			AccountNumber:       "",
			SWIFTCode:           "",
			BSBNumber:           "",
			IBAN:                "",
			SupportedCurrencies: "",
			SupportedExchanges:  "",
		},
	}
)

func TestMain(m *testing.M) {
	Accounts = append(Accounts, testBankAccounts...)
	os.Exit(m.Run())
}

func TestGetBankAccountByID(t *testing.T) {
	_, err := GetBankAccountByID("valid-test-bank-01")
	if err != nil {
		t.Error(err)
	}

	_, err = GetBankAccountByID("invalid-test-")
	if err == nil {
		t.Error("error expected for invalid account received nil")
	}
}

func TestAccount_Validate(t *testing.T) {
	valid, err := GetBankAccountByID("valid-test-bank-01")
	if err != nil {
		t.Fatal(err)
	}
	if err = valid.Validate(); err != nil {
		t.Error(err)
	}

	invalid := testBankAccounts[1]
	if err = invalid.Validate(); err == nil {
		t.Error(err)
	}

	invalid = testBankAccounts[0]
	invalid.SupportedCurrencies = "AUD"
	invalid.BSBNumber = ""
	if err = invalid.Validate(); err == nil {
		t.Error("Expected error when Currency is AUD but no BSB set")
	}

	invalid = testBankAccounts[0]
	invalid.SupportedExchanges = ""
	if err = invalid.Validate(); err != nil {
		t.Error("Expected error when Currency is AUD but no BSB set")
	}
	if invalid.SupportedExchanges != "ALL" {
		t.Error("expected SupportedExchanges to return \"ALL\" after validation")
	}

	invalid = testBankAccounts[0]
	invalid.SWIFTCode = ""
	invalid.IBAN = ""
	invalid.SupportedCurrencies = "USD"
	if err = invalid.Validate(); err == nil {
		t.Error("Expected error when no Swift/IBAN set")
	}
}

func TestAccount_ValidateForWithdrawal(t *testing.T) {
	v, err := GetBankAccountByID("valid-test-bank-01")
	if err != nil {
		t.Fatal(err)
	}
	errWith := v.ValidateForWithdrawal("test-exchange", currency.AUD)
	if errWith != nil {
		t.Fatal(errWith)
	}
	v.BSBNumber = ""
	errWith = v.ValidateForWithdrawal("test-exchange", currency.AUD)
	if errWith != nil {
		if errWith[0] != ErrBSBRequiredforAUD {
			t.Fatal(errWith)
		}
	}
	v.SWIFTCode = ""
	v.IBAN = ""
	errWith = v.ValidateForWithdrawal("test-exchange", currency.USD)
	if errWith != nil {
		if errWith[0] != ErrIBANSwiftNotSet {
			t.Fatal(errWith)
		}
	}
	errWith = v.ValidateForWithdrawal("test-exchange-nope", currency.AUD)
	if errWith != nil {
		if errWith[0] != "Exchange test-exchange-nope not supported by bank account" {
			t.Fatal(errWith)
		}
	}
	v.AccountNumber = ""
	errWith = v.ValidateForWithdrawal("test-exchange", currency.AUD)
	if errWith != nil {
		if errWith[0] != ErrAccountCannotBeEmpty {
			t.Fatal(errWith)
		}
	}
	v.Enabled = false
	errWith = v.ValidateForWithdrawal("test-exchange", currency.AUD)
	if errWith != nil {
		if errWith[0] != ErrBankAccountDisabled {
			t.Fatal(errWith)
		}
	}
}
