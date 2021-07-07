package banking

import (
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	validAccount = Account{
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
	}
	invalidAccount = Account{
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
	}
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestGetBankAccountByID(t *testing.T) {
	t.Parallel()
	SetAccounts(validAccount, invalidAccount)
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
	t.Parallel()
	testBankAccounts := []Account{
		validAccount, invalidAccount,
	}
	invalid := testBankAccounts[1]
	if err := invalid.Validate(); err == nil {
		t.Error(err)
	}

	invalid = testBankAccounts[0]
	invalid.SupportedCurrencies = "AUD"
	invalid.BSBNumber = ""
	if err := invalid.Validate(); err == nil {
		t.Error("Expected error when Currency is AUD but no BSB set")
	}

	invalid = testBankAccounts[0]
	invalid.SupportedExchanges = ""
	if err := invalid.Validate(); err != nil {
		t.Error("Expected error when Currency is AUD but no BSB set")
	}
	if invalid.SupportedExchanges != "ALL" {
		t.Error("expected SupportedExchanges to return \"ALL\" after validation")
	}

	invalid = testBankAccounts[0]
	invalid.SWIFTCode = ""
	invalid.IBAN = ""
	invalid.SupportedCurrencies = "USD"
	if err := invalid.Validate(); err == nil {
		t.Error("Expected error when no Swift/IBAN set")
	}
}

func TestAccount_ValidateForWithdrawal(t *testing.T) {
	t.Parallel()
	acc := validAccount
	errWith := acc.ValidateForWithdrawal("test-exchange", currency.AUD)
	if errWith != nil {
		t.Fatal(errWith)
	}
	acc.BSBNumber = ""
	errWith = acc.ValidateForWithdrawal("test-exchange", currency.AUD)
	if errWith != nil {
		if errWith[0] != ErrBSBRequiredForAUD {
			t.Fatal(errWith)
		}
	}
	acc.SWIFTCode = ""
	acc.IBAN = ""
	errWith = acc.ValidateForWithdrawal("test-exchange", currency.USD)
	if errWith != nil {
		if errWith[0] != ErrIBANSwiftNotSet {
			t.Fatal(errWith)
		}
	}
	errWith = acc.ValidateForWithdrawal("test-exchange-nope", currency.AUD)
	if errWith != nil {
		if errWith[0] != "Exchange test-exchange-nope not supported by bank account" {
			t.Fatal(errWith)
		}
	}
	acc.AccountNumber = ""
	errWith = acc.ValidateForWithdrawal("test-exchange", currency.AUD)
	if errWith != nil {
		if errWith[0] != ErrAccountCannotBeEmpty {
			t.Fatal(errWith)
		}
	}
	acc.Enabled = false
	errWith = acc.ValidateForWithdrawal("test-exchange", currency.AUD)
	if errWith != nil {
		if errWith[0] != ErrBankAccountDisabled {
			t.Fatal(errWith)
		}
	}
}

func TestSetAccounts(t *testing.T) {
	SetAccounts()
	if len(accounts) != 0 {
		t.Error("expected 0")
	}
	SetAccounts(validAccount, invalidAccount)
	if len(accounts) != 2 {
		t.Error("expected 2")
	}
}

func TestAppendAccounts(t *testing.T) {
	SetAccounts()
	if len(accounts) != 0 {
		t.Error("expected 0")
	}
	AppendAccounts(validAccount, invalidAccount)
	if len(accounts) != 2 {
		t.Error("expected 2")
	}
}
