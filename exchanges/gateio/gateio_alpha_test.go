package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetAlphaAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlphaAccounts(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlphaAccountTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlphaAccountTransactionHistory(t.Context(), time.Time{}, time.Time{}, 1, 10)
	require.ErrorIs(t, err, errStartTimeRequired)

	startTime, endTime := getTime()
	_, err = e.GetAlphaAccountTransactionHistory(t.Context(), endTime, startTime, 1, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlphaAccountTransactionHistory(t.Context(), startTime, endTime, 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateAlphaCurrencyQuoteID(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAlphaCurrencyQuoteID(t.Context(), &AlphaCurrencyQuoteInfoRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg := &AlphaCurrencyQuoteInfoRequest{Currency: currency.BTC}
	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell
	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1
	_, err = e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.ErrorIs(t, err, errGasModeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.GasMode = "custom"
	result, err := e.CreateAlphaCurrencyQuoteID(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceAlphaTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceAlphaTradeOrder(t.Context(), &AlphaCurrencyQuoteInfoRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg := &AlphaCurrencyQuoteInfoRequest{Currency: currency.BTC}
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, errGasModeRequired)

	arg.GasMode = "custom"
	_, err = e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, errQuoteIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.QuoteID = "123345678"
	result, err := e.PlaceAlphaTradeOrder(t.Context(), arg)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlphaOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlphaOrders(t.Context(), currency.EMPTYCODE, order.Sell, 0, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetAlphaOrders(t.Context(), currency.ETH, order.Long, 0, time.Time{}, time.Time{}, 0, 10)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	startTime, endTime := getTime()
	_, err = e.GetAlphaOrders(t.Context(), currency.ETH, order.Sell, 0, endTime, startTime, 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlphaOrders(t.Context(), currency.ETH, order.Sell, 1, startTime, endTime, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlphaOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlphaOrderByID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlphaOrderByID(t.Context(), "123345678")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlphaCurrenciesDetail(t *testing.T) {
	t.Parallel()
	result, err := e.GetAlphaCurrenciesDetail(t.Context(), currency.EMPTYCODE, 100, 10)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	result, err = e.GetAlphaCurrenciesDetail(t.Context(), currency.NewCode("memeboxtrump"), 100, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlphaCurrencyTicker(t *testing.T) {
	t.Parallel()
	result, err := e.GetAlphaCurrencyTicker(t.Context(), currency.EMPTYCODE, 100, 10)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	result, err = e.GetAlphaCurrencyTicker(t.Context(), currency.NewCode("memeboxtrump"), 100, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
