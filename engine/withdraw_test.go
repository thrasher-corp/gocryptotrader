package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	exchangeName  = "BTC Markets"
	bankAccountID = "test-bank-01"
)

var (
	settings = Settings{
		ConfigFile:          filepath.Join("..", "testdata", "configtest.json"),
		EnableDryRun:        true,
		DataDir:             filepath.Join("..", "testdata", "gocryptotrader"),
		Verbose:             false,
		EnableGRPC:          false,
		EnableDeprecatedRPC: false,
		EnableWebsocketRPC:  false,
	}
)

func setupEngine() (err error) {
	Bot, err = NewFromSettings(&settings)
	if err != nil {
		return err
	}
	return Bot.Start()
}

func cleanup() {
	err := os.RemoveAll(settings.DataDir)
	if err != nil {
		fmt.Printf("Clean up failed to remove file: %v manual removal may be required", err)
	}
}

func TestSubmitWithdrawal(t *testing.T) {
	err := setupEngine()
	if err != nil {
		t.Fatal(err)
	}

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
			SupportedExchanges:  exchangeName,
		},
	)

	bank, err := banking.GetBankAccountByID(bankAccountID)
	if err != nil {
		t.Fatal(err)
	}
	req := &withdraw.Request{
		Exchange:    exchangeName,
		Currency:    currency.AUD,
		Description: exchangeName,
		Amount:      1.0,
		Type:        1,
		Fiat: &withdraw.FiatRequest{
			Bank: bank,
		},
	}

	_, err = SubmitWithdrawal(exchangeName, req)
	if err != nil {
		t.Fatal(err)
	}

	_, err = SubmitWithdrawal(exchangeName, nil)
	if err != nil {
		if err.Error() != withdraw.ErrRequestCannotBeNil.Error() {
			t.Fatal(err)
		}
	}
	cleanup()
}

func TestWithdrawEventtByID(t *testing.T) {
	tempResp := &withdraw.Response{
		ID: withdraw.DryRunID,
	}
	_, err := WithdrawEventByID(withdraw.DryRunID.String())
	if err != nil {
		if err.Error() != fmt.Errorf(ErrWithdrawRequestNotFound, withdraw.DryRunID.String()).Error() {
			t.Fatal(err)
		}
	}
	withdraw.Cache.Add(withdraw.DryRunID.String(), tempResp)
	v, err := WithdrawEventByID(withdraw.DryRunID.String())
	if err != nil {
		if err != fmt.Errorf(ErrWithdrawRequestNotFound, withdraw.DryRunID.String()) {
			t.Fatal(err)
		}
	}
	if v == nil {
		t.Fatal("expected WithdrawEventByID() to return data from cache")
	}
}
