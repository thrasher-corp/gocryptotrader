package banking

import (
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestMain(m *testing.M) {
	Accounts = append(Accounts,
		Account{
			Enabled:             true,
			ID:                  "test-bank-01",
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
			SupportedCurrencies: "USD",
			SupportedExchanges:  "test-exchange",
		},
	)
	os.Exit(m.Run())
}

func TestGetBankAccountByID(t *testing.T) {
	_, err := GetBankAccountByID("test-bank-01")
	if err != nil {
		t.Error(err)
	}

	_, err = GetBankAccountByID("invalid-test-bank-01")
	if err == nil {
		t.Error("error expected for invalid account received nil")
	}
}

func TestAccount_Validate(t *testing.T) {

}

func TestAccount_ValidateForWithdrawal(t *testing.T) {
	v, err := GetBankAccountByID("test-bank-01")
	if err != nil {
		t.Error(err)
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