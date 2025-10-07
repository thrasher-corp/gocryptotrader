package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetConvertCoinList(t *testing.T) {
	t.Parallel()

	_, err := e.GetConvertCoinList(t.Context(), "", currency.EMPTYCODE, true)
	assert.ErrorIs(t, err, errUnsupportedAccountType)

	_, err = e.GetConvertCoinList(t.Context(), UTA, currency.EMPTYCODE, true)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	buylist, err := e.GetConvertCoinList(t.Context(), UTA, currency.USDT, true)
	require.NoError(t, err)
	assert.NotEmpty(t, buylist)

	sellList, err := e.GetConvertCoinList(t.Context(), UTA, currency.EMPTYCODE, false)
	require.NoError(t, err)
	assert.NotEmpty(t, sellList)
}

func TestRequestQuote(t *testing.T) {
	t.Parallel()

	_, err := e.RequestQuote(t.Context(), &RequestQuoteRequest{})
	assert.ErrorIs(t, err, errUnsupportedAccountType)

	_, err = e.RequestQuote(t.Context(), &RequestQuoteRequest{AccountType: UTA})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.RequestQuote(t.Context(), &RequestQuoteRequest{AccountType: UTA, FromCoin: currency.BTC})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.RequestQuote(t.Context(), &RequestQuoteRequest{
		AccountType: UTA,
		FromCoin:    currency.BTC,
		ToCoin:      currency.BTC,
	})
	assert.ErrorIs(t, err, errCurrencyCodesEqual)

	_, err = e.RequestQuote(t.Context(), &RequestQuoteRequest{
		AccountType: UTA,
		FromCoin:    currency.BTC,
		ToCoin:      currency.USDT,
		RequestCoin: currency.WOO,
	})
	assert.ErrorIs(t, err, errRequestCoinInvalid)

	_, err = e.RequestQuote(t.Context(), &RequestQuoteRequest{
		AccountType: UTA,
		FromCoin:    currency.BTC,
		ToCoin:      currency.USDT,
	})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	quote, err := e.RequestQuote(t.Context(), &RequestQuoteRequest{
		AccountType:   UTA,
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

	_, err = e.GetConvertStatus(t.Context(), UTA, "")
	assert.ErrorIs(t, err, errQuoteTransactionIDEmpty)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	status, err := e.GetConvertStatus(t.Context(), UTA, "10414247553864074960678912")
	require.NoError(t, err)
	assert.NotEmpty(t, status)
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()

	_, err := e.GetConvertHistory(t.Context(), []WalletAccountType{""}, 0, 0)
	assert.ErrorIs(t, err, errUnsupportedAccountType)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	history, err := e.GetConvertHistory(t.Context(), []WalletAccountType{UTA}, 0, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, history)

	if mockTests {
		require.Equal(t, 4, len(history), "GetConvertHistory must return 4 items in mock test")
		exp := ConvertHistoryResponse{
			AccountType:           UTA,
			ExchangeTransactionID: "104231555340214158196736",
			UserID:                "74199870",
			FromCoin:              currency.NewCode("UXLINK"),
			FromCoinType:          "crypto",
			FromAmount:            7.9952,
			ToCoin:                currency.USDT,
			ToCoinType:            "crypto",
			ToAmount:              2.84509740190888,
			ExchangeStatus:        "success",
			ExtendedInfo:          ExtendedInfoHistoryResponse{},
			ConvertRate:           0.35585068565,
			CreatedAt:             types.Time(time.UnixMilli(1754880224953)),
		}
		require.Equal(t, exp, history[0])
	}
}
