package engine

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okx"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	withdrawManagerTestExchangeName = "okx"
)

func withdrawManagerTestHelper(t *testing.T) (*ExchangeManager, *portfolioManager) {
	t.Helper()
	em := NewExchangeManager()
	b := new(okx.Exchange)
	cfg, err := exchange.GetDefaultConfig(t.Context(), b)
	if err != nil {
		t.Fatal(err)
	}
	err = b.Setup(cfg)
	if err != nil {
		t.Fatal(err)
	}
	err = em.Add(b)
	require.NoError(t, err)

	pm, err := setupPortfolioManager(em, 0, &portfolio.Base{Addresses: []portfolio.Address{}})
	if err != nil {
		t.Fatal(err)
	}

	return em, pm
}

func TestSubmitWithdrawal(t *testing.T) {
	t.Parallel()
	em, pm := withdrawManagerTestHelper(t)
	m, err := SetupWithdrawManager(em, pm, false)
	if err != nil {
		t.Fatal(err)
	}
	bank := banking.Account{
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
	}

	banking.AppendAccounts(bank)

	req := &withdraw.Request{
		Exchange:    withdrawManagerTestExchangeName,
		Currency:    currency.AUD,
		Description: withdrawManagerTestExchangeName,
		Amount:      1.0,
		Type:        withdraw.Fiat,
		Fiat: withdraw.FiatRequest{
			Bank: bank,
		},
	}
	_, err = m.SubmitWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)

	req.Type = withdraw.Crypto
	req.Currency = currency.BTC
	req.Crypto.Address = "1337"
	_, err = m.SubmitWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, withdraw.ErrStrAddressNotWhiteListed)

	var wg sync.WaitGroup
	err = pm.Start(&wg)
	if err != nil {
		t.Error(err)
	}
	err = pm.AddAddress("1337", "", req.Currency, 1337)
	if err != nil {
		t.Error(err)
	}
	adds := pm.GetAddresses()
	adds[0].WhiteListed = true
	assert.NoError(t, err)

	_, err = m.SubmitWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, withdraw.ErrStrExchangeNotSupportedByAddress)

	adds[0].SupportedExchanges = withdrawManagerTestExchangeName
	_, err = m.SubmitWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)

	_, err = m.SubmitWithdrawal(t.Context(), nil)
	assert.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil)

	m.isDryRun = true
	_, err = m.SubmitWithdrawal(t.Context(), req)
	assert.NoError(t, err)
}

func TestWithdrawEventByID(t *testing.T) {
	t.Parallel()
	em, pm := withdrawManagerTestHelper(t)
	m, err := SetupWithdrawManager(em, pm, false)
	if err != nil {
		t.Fatal(err)
	}
	tempResp := &withdraw.Response{
		ID: withdraw.DryRunID,
	}
	_, err = m.WithdrawalEventByID(withdraw.DryRunID.String())
	assert.ErrorIs(t, err, ErrWithdrawRequestNotFound)

	withdraw.Cache.Add(withdraw.DryRunID.String(), tempResp)
	v, err := m.WithdrawalEventByID(withdraw.DryRunID.String())
	assert.NoError(t, err)

	if v == nil {
		t.Error("expected WithdrawalEventByID() to return data from cache")
	}
}

func TestWithdrawalEventByExchange(t *testing.T) {
	t.Parallel()
	em, pm := withdrawManagerTestHelper(t)
	m, err := SetupWithdrawManager(em, pm, false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = (*WithdrawManager)(nil).WithdrawalEventByExchange("xxx", 0)
	assert.ErrorIs(t, err, ErrNilSubsystem)

	_, err = m.WithdrawalEventByExchange("xxx", 0)
	assert.ErrorIs(t, err, ErrExchangeNotFound)
}

func TestWithdrawEventByDate(t *testing.T) {
	t.Parallel()
	em, pm := withdrawManagerTestHelper(t)
	m, err := SetupWithdrawManager(em, pm, false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = (*WithdrawManager)(nil).WithdrawEventByDate("xxx", time.Now(), time.Now(), 1)
	assert.ErrorIs(t, err, ErrNilSubsystem)

	_, err = m.WithdrawEventByDate("xxx", time.Now(), time.Now(), 1)
	assert.ErrorIs(t, err, ErrExchangeNotFound)
}

func TestWithdrawalEventByExchangeID(t *testing.T) {
	t.Parallel()
	em, _ := withdrawManagerTestHelper(t)
	m, err := SetupWithdrawManager(em, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = (*WithdrawManager)(nil).WithdrawalEventByExchangeID("xxx", "xxx")
	assert.ErrorIs(t, err, ErrNilSubsystem)

	_, err = m.WithdrawalEventByExchangeID("xxx", "xxx")
	assert.ErrorIs(t, err, ErrExchangeNotFound)
}
