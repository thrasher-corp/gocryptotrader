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

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	buylist, err := e.GetConvertCoinList(t.Context(), Uta, currency.USDT, true)
	require.NoError(t, err)
	assert.NotEmpty(t, buylist)

	sellList, err := e.GetConvertCoinList(t.Context(), Uta, currency.EMPTYCODE, false)
	require.NoError(t, err)
	assert.NotEmpty(t, sellList)
}

func TestRequestQuote(t *testing.T) {
	t.Parallel()

	_, err := e.RequestQuote(t.Context(), &RequestQuoteRequest{})
	assert.ErrorIs(t, err, errUnsupportedAccountType)

	_, err = e.RequestQuote(t.Context(), &RequestQuoteRequest{AccountType: Uta})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.RequestQuote(t.Context(), &RequestQuoteRequest{AccountType: Uta, FromCoin: currency.BTC})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.RequestQuote(t.Context(), &RequestQuoteRequest{
		AccountType: Uta,
		FromCoin:    currency.BTC,
		ToCoin:      currency.BTC,
	})
	assert.ErrorIs(t, err, errCurrencyCodesEqual)

	_, err = e.RequestQuote(t.Context(), &RequestQuoteRequest{
		AccountType: Uta,
		FromCoin:    currency.BTC,
		ToCoin:      currency.USDT,
		RequestCoin: currency.WOO,
	})
	assert.ErrorIs(t, err, errRequestCoinInvalid)

	_, err = e.RequestQuote(t.Context(), &RequestQuoteRequest{
		AccountType: Uta,
		FromCoin:    currency.BTC,
		ToCoin:      currency.USDT,
	})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	quote, err := e.RequestQuote(t.Context(), &RequestQuoteRequest{
		AccountType:   Uta,
		FromCoin:      currency.XRP,
		ToCoin:        currency.USDT,
		RequestAmount: 0.0088,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, quote.QuoteTransactionID)
}

func TestConfirmQuote(t *testing.T) {
	t.Parallel()

	_, err := e.ConfirmQuote(t.Context(), "")
	assert.ErrorIs(t, err, errQuoteTransactionIDEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	quote, err := e.ConfirmQuote(t.Context(), "10175108571334212336947200")
	require.NoError(t, err)
	assert.NotEmpty(t, quote.QuoteTransactionID)
}

func TestGetConvertStatus(t *testing.T) {
	t.Parallel()

	_, err := e.GetConvertStatus(t.Context(), "", "")
	assert.ErrorIs(t, err, errUnsupportedAccountType)

	_, err = e.GetConvertStatus(t.Context(), Uta, "")
	assert.ErrorIs(t, err, errQuoteTransactionIDEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	status, err := e.GetConvertStatus(t.Context(), Uta, "10414247553864074960678912")
	require.NoError(t, err)
	assert.NotEmpty(t, status)
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()

	_, err := e.GetConvertHistory(t.Context(), []WalletAccountType{""}, 0, 0)
	assert.ErrorIs(t, err, errUnsupportedAccountType)

	if mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	history, err := e.GetConvertHistory(t.Context(), []WalletAccountType{Uta}, 0, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, history)
}
