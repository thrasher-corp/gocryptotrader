package withdrawalmanager

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

const (
	bankAccountID = "test-bank-01"
	exchangeName  = "Binance"
)

var (
	settings = engine.Settings{
		ConfigFile:          filepath.Join("..", "testdata", "configtest.json"),
		EnableDryRun:        true,
		DataDir:             filepath.Join("..", "testdata", "gocryptotrader"),
		Verbose:             false,
		EnableGRPC:          false,
		EnableDeprecatedRPC: false,
		EnableWebsocketRPC:  false,
	}
	em exchangemanager.Manager
	w  Manager
)

func cleanup() {
	err := os.RemoveAll(settings.DataDir)
	if err != nil {
		fmt.Printf("Clean up failed to remove file: %v manual removal may be required", err)
	}
}

func TestMain(m *testing.M) {
	em = exchangemanager.Manager{}
	exch, err := em.NewExchangeByName(exchangeName)
	if err != nil {
		log.Fatal(err)
	}
	em.Add(exch)
	w = Manager{exchangeManager: &em}
}

func TestSubmitWithdrawal(t *testing.T) {
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
			SupportedExchanges:  "Binance",
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
		Fiat: withdraw.FiatRequest{
			Bank: *bank,
		},
	}

	_, err = w.SubmitWithdrawal(req)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.SubmitWithdrawal(nil)
	if err != nil {
		if errors.Is(withdraw.ErrRequestCannotBeNil, err) {
			t.Fatal(err)
		}
	}
	cleanup()
}

func TestWithdrawEventByID(t *testing.T) {
	tempResp := &withdraw.Response{
		ID: withdraw.DryRunID,
	}
	_, err := w.WithdrawalEventByID(withdraw.DryRunID.String())
	if err != nil {
		if err.Error() != fmt.Errorf(ErrWithdrawRequestNotFound, withdraw.DryRunID.String()).Error() {
			t.Fatal(err)
		}
	}
	withdraw.Cache.Add(withdraw.DryRunID.String(), tempResp)
	v, err := w.WithdrawalEventByID(withdraw.DryRunID.String())
	if err != nil {
		if err != fmt.Errorf(ErrWithdrawRequestNotFound, withdraw.DryRunID.String()) {
			t.Fatal(err)
		}
	}
	if v == nil {
		t.Fatal("expected WithdrawalEventByID() to return data from cache")
	}
}

func TestWithdrawalEventByExchange(t *testing.T) {
	_, err := w.WithdrawalEventByExchange(exchangeName, 1)
	if err == nil {
		t.Fatal(err)
	}
}

func TestWithdrawEventByDate(t *testing.T) {
	_, err := w.WithdrawEventByDate(exchangeName, time.Now(), time.Now(), 1)
	if err == nil {
		t.Fatal(err)
	}
}

func TestWithdrawalEventByExchangeID(t *testing.T) {
	_, err := w.WithdrawalEventByExchangeID(exchangeName, exchangeName)
	if err == nil {
		t.Fatal(err)
	}
}
