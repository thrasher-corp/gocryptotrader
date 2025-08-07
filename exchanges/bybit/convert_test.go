package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetConvertCoinList(t *testing.T) {
	t.Parallel()

	_, err := e.GetConvertCoinList(t.Context(), "", currency.EMPTYCODE, true)
	assert.ErrorIs(t, err, errUnsupportedAccountType)

	_, err = e.GetConvertCoinList(t.Context(), Uta, currency.EMPTYCODE, true)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	list, err := e.GetConvertCoinList(t.Context(), Uta, currency.EMPTYCODE, false)
	require.NoError(t, err)
	assert.NotEmpty(t, list)
}

func TestRequestAQuote(t *testing.T) {
	t.Parallel()

	_, err := e.RequestAQuote(t.Context(), &RequestAQuoteParams{})
	assert.ErrorIs(t, err, errUnsupportedAccountType)

	_, err = e.RequestAQuote(t.Context(), &RequestAQuoteParams{AccountType: Uta})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.RequestAQuote(t.Context(), &RequestAQuoteParams{AccountType: Uta, From: currency.BTC})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.RequestAQuote(t.Context(), &RequestAQuoteParams{
		AccountType: Uta,
		From:        currency.BTC,
		To:          currency.BTC,
	})
	assert.ErrorIs(t, err, errCurrencyCodesEqual)

	_, err = e.RequestAQuote(t.Context(), &RequestAQuoteParams{
		AccountType: Uta,
		From:        currency.BTC,
		To:          currency.USDT,
		RequestCoin: currency.WOO,
	})
	assert.ErrorIs(t, err, errRequestCoinInvalid)

	_, err = e.RequestAQuote(t.Context(), &RequestAQuoteParams{
		AccountType: Uta,
		From:        currency.BTC,
		To:          currency.USDT,
	})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	quote, err := e.RequestAQuote(t.Context(), &RequestAQuoteParams{
		AccountType: Uta,
		From:        currency.BTC,
		To:          currency.USDT,
		Amount:      69.420,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, quote.QuoteTxID)
}

func TestConfirmAQuote(t *testing.T) {
	t.Parallel()

	_, err := e.ConfirmAQuote(t.Context(), "")
	assert.ErrorIs(t, err, errQuoteTxIDEmpty)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	quote, err := e.ConfirmAQuote(t.Context(), "10414247553864074960678912")
	require.NoError(t, err)
	assert.NotEmpty(t, quote.QuoteTxID)
}

func TestGetConvertStatus(t *testing.T) {
	t.Parallel()

	_, err := e.GetConvertStatus(t.Context(), "", "")
	assert.ErrorIs(t, err, errUnsupportedAccountType)

	_, err = e.GetConvertStatus(t.Context(), Uta, "")
	assert.ErrorIs(t, err, errQuoteTxIDEmpty)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	status, err := e.GetConvertStatus(t.Context(), Uta, "10414247553864074960678912")
	require.NoError(t, err)
	assert.NotEmpty(t, status)
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()

	_, err := e.GetConvertHistory(t.Context(), []WalletAccountType{""}, 0, 0)
	assert.ErrorIs(t, err, errUnsupportedAccountType)

	if mockTests {
		t.Skip(skipAuthenticatedFunctionsForMockTesting)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	history, err := e.GetConvertHistory(t.Context(), nil, 0, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, history)
}
