package withdrawalmanager

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

const (
	bankAccountID = "test-bank-01"
	exchangeName  = "Binance"
)

var em *exchangemanager.Manager

func TestMain(m *testing.M) {
	em = exchangemanager.Setup()
	b := new(binance.Binance)
	b.SetDefaults()
	em.Add(b)
	os.Exit(m.Run())
}

func TestSubmitWithdrawal(t *testing.T) {
	t.Parallel()
	m, err := Setup(em, false)
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
			SupportedCurrencies: "AUD,USD",
			SupportedExchanges:  "Binance",
		},
	)
	bank, err := banking.GetBankAccountByID(bankAccountID)
	if err != nil {
		t.Error(err)
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
	_, err = m.SubmitWithdrawal(req)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	req.Type = withdraw.Crypto
	req.Currency = currency.BTC
	_, err = m.SubmitWithdrawal(req)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}

	_, err = m.SubmitWithdrawal(nil)
	if !errors.Is(err, withdraw.ErrRequestCannotBeNil) {
		t.Errorf("received %v, expected %v", err, withdraw.ErrRequestCannotBeNil)
	}

	m.isDryRun = true
	_, err = m.SubmitWithdrawal(req)
	if !errors.Is(err, nil) {
		t.Errorf("received %v, expected %v", err, nil)
	}
}

func TestWithdrawEventByID(t *testing.T) {
	t.Parallel()
	m, err := Setup(em, false)
	if err != nil {
		t.Fatal(err)
	}
	tempResp := &withdraw.Response{
		ID: withdraw.DryRunID,
	}
	_, err = m.WithdrawalEventByID(withdraw.DryRunID.String())
	if !errors.Is(err, ErrWithdrawRequestNotFound) {
		t.Errorf("received %v, expected %v", err, ErrWithdrawRequestNotFound)
	}

	withdraw.Cache.Add(withdraw.DryRunID.String(), tempResp)
	v, err := m.WithdrawalEventByID(withdraw.DryRunID.String())
	if !errors.Is(err, nil) {
		t.Errorf("expected %v, received %v", nil, err)
	}
	if v == nil {
		t.Error("expected WithdrawalEventByID() to return data from cache")
	}
}

func TestWithdrawalEventByExchange(t *testing.T) {
	t.Parallel()
	m, err := Setup(em, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.WithdrawalEventByExchange(exchangeName, 1)
	if err == nil {
		t.Error(err)
	}
}

func TestWithdrawEventByDate(t *testing.T) {
	t.Parallel()
	m, err := Setup(em, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.WithdrawEventByDate(exchangeName, time.Now(), time.Now(), 1)
	if err == nil {
		t.Error(err)
	}
}

func TestWithdrawalEventByExchangeID(t *testing.T) {
	t.Parallel()
	m, err := Setup(em, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.WithdrawalEventByExchangeID(exchangeName, exchangeName)
	if err == nil {
		t.Error(err)
	}
}
