package banking

import (
	"testing"
)

func TestGetBankAccountByID(t *testing.T) {
	Accounts = append(Accounts,
		Account{
			ID:                  "test-bank-01",
			BankName:            "Test Bank",
			BankAddress:         "42 Bank Street",
			BankPostalCode:      "13337",
			BankPostalCity:      "Satoshiville",
			BankCountry:         "Japan",
			AccountName:         "Satoshi Nakamoto",
			AccountNumber:       "0234",
			SWIFTCode:           "91272837",
			IBAN:                "98218738671897",
			SupportedCurrencies: "USD",
			SupportedExchanges:  "Kraken,Bitstamp",
		},
	)

	_, err := GetBankAccountByID("test-bank-01")
	if err != nil {
		t.Error(err)
	}

	_, err = GetBankAccountByID("invalid-test-bank-01")
	if err == nil {
		t.Error("error expected for invalid account received nil")
	}
}
