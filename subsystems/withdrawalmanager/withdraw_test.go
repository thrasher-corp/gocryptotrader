package withdrawalmanager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
)

func cleanup() {
	err := os.RemoveAll(settings.DataDir)
	if err != nil {
		fmt.Printf("Clean up failed to remove file: %v manual removal may be required", err)
	}
}

func TestSubmitWithdrawal(t *testing.T) {
	w := Manager{exchangeManager: &exchangemanager.Manager{}}
	exch, err := w.exchangeManager.NewExchangeByName(exchangeName)
	if err != nil {
		t.Error(err)
	}
	w.exchangeManager.Add(exch)
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
	_, err := WithdrawalEventByID(withdraw.DryRunID.String())
	if err != nil {
		if err.Error() != fmt.Errorf(ErrWithdrawRequestNotFound, withdraw.DryRunID.String()).Error() {
			t.Fatal(err)
		}
	}
	withdraw.Cache.Add(withdraw.DryRunID.String(), tempResp)
	v, err := WithdrawalEventByID(withdraw.DryRunID.String())
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
	_, err := WithdrawalEventByExchange(exchangeName, 1)
	if err == nil {
		t.Fatal(err)
	}
}

func TestWithdrawEventByDate(t *testing.T) {
	_, err := WithdrawEventByDate(exchangeName, time.Now(), time.Now(), 1)
	if err == nil {
		t.Fatal(err)
	}
}

func TestWithdrawalEventByExchangeID(t *testing.T) {
	_, err := WithdrawalEventByExchangeID(exchangeName, exchangeName)
	if err == nil {
		t.Fatal(err)
	}
}

func TestParseEvents(t *testing.T) {
	var testData []*withdraw.Response
	for x := 0; x < 5; x++ {
		test := fmt.Sprintf("test-%v", x)
		resp := &withdraw.Response{
			ID: withdraw.DryRunID,
			Exchange: withdraw.ExchangeResponse{
				Name:   test,
				ID:     test,
				Status: test,
			},
			RequestDetails: withdraw.Request{
				Exchange:    test,
				Description: test,
				Amount:      1.0,
			},
		}
		if x%2 == 0 {
			resp.RequestDetails.Currency = currency.AUD
			resp.RequestDetails.Type = 1
			resp.RequestDetails.Fiat = withdraw.FiatRequest{
				Bank: banking.Account{
					Enabled:             false,
					ID:                  fmt.Sprintf("test-%v", x),
					BankName:            fmt.Sprintf("test-%v-bank", x),
					AccountName:         "hello",
					AccountNumber:       fmt.Sprintf("test-%v", x),
					BSBNumber:           "123456",
					SupportedCurrencies: "BTC-AUD",
					SupportedExchanges:  exchangeName,
				},
			}
		} else {
			resp.RequestDetails.Currency = currency.BTC
			resp.RequestDetails.Type = 0
			resp.RequestDetails.Crypto.Address = test
			resp.RequestDetails.Crypto.FeeAmount = 0
			resp.RequestDetails.Crypto.AddressTag = test
		}
		testData = append(testData, resp)
	}
	v := ParseMultipleEvents(testData)
	if reflect.TypeOf(v).String() != "*gctrpc.WithdrawalEventsByExchangeResponse" {
		t.Fatal("expected type to be *gctrpc.WithdrawalEventsByExchangeResponse")
	}
	if testData == nil || len(testData) < 2 {
		t.Fatal("expected at least 2")
	}

	v = ParseSingleEvents(testData[0])
	if reflect.TypeOf(v).String() != "*gctrpc.WithdrawalEventsByExchangeResponse" {
		t.Fatal("expected type to be *gctrpc.WithdrawalEventsByExchangeResponse")
	}

	v = ParseSingleEvents(testData[1])
	if v.Event[0].Request.Type != 0 {
		t.Fatal("Expected second entry in slice to return a Request.Type of Crypto")
	}
}
