package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestTransferCollateralToIsolatedMargin(t *testing.T) {
	t.Parallel()
	_, err := e.TransferCollateralToIsolatedMargin(t.Context(), BTCUSDT, currency.EMPTYCODE, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.TransferCollateralToIsolatedMargin(t.Context(), BTCUSDT, currency.USDT, 10)
	require.NoError(t, err, "TransferCollateralToIsolatedMargin must not error")
}

func TestTransferCollateralFromIsolatedMargin(t *testing.T) {
	t.Parallel()
	_, err := e.TransferCollateralFromIsolatedMargin(t.Context(), BTCUSDT, currency.EMPTYCODE, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.TransferCollateralFromIsolatedMargin(t.Context(), BTCUSDT, currency.USDT, 1)
	require.NoError(t, err, "TransferCollateralFromIsolatedMargin must not error")
}

func TestGetIsolatedMarginAccountBalanceChangeHistory(t *testing.T) {
	t.Parallel()
	tn := time.Now()
	_, err := e.GetIsolatedMarginAccountBalanceChangeHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, tn.Add(time.Hour), tn, 0, 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedMarginAccountBalanceChangeHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err, "GetIsolatedMarginAccountBalanceChangeHistory must not error")
}

func TestGetIsolatedMarginFundingAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetIsolatedMarginFundingAccountList(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "GetIsolatedMarginFundingAccountList must not error")
}

func TestGetIsolatedMarginUserAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.GetIsolatedMarginUserAutoRepaymentSetting(t.Context())
	require.NoError(t, err, "GetIsolatedMarginUserAutoRepaymentSetting must not error")
}

func TestUpdateIsolatedMarginUsersAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.UpdateIsolatedMarginUsersAutoRepaymentSetting(t.Context(), false)
	require.NoError(t, err, "UpdateIsolatedMarginUsersAutoRepaymentSetting must not error")
}

func TestGetIsolatedMarginMaxTransferableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginMaxTransferableAmount(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedMarginMaxTransferableAmount(t.Context(), currency.USDT, BTCUSDT)
	require.NoError(t, err, "GetIsolatedMarginMaxTransferableAmount must not error")
}

func TestGetIsolatedMarginLendingMarkets(t *testing.T) {
	t.Parallel()
	markets, err := e.GetIsolatedMarginLendingMarkets(t.Context())
	require.NoError(t, err, "GetIsolatedMarginLendingMarkets must not error")
	require.NotEmpty(t, markets, "GetIsolatedMarginLendingMarkets must return some markets")
}

func TestGetIsolatedMarginLendingMarketDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginLendingMarketDetails(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	market, err := e.GetIsolatedMarginLendingMarketDetails(t.Context(), BTCUSDT)
	require.NoError(t, err, "GetIsolatedMarginLendingMarketDetails must not error")
	require.NotNil(t, market, "GetIsolatedMarginLendingMarketDetails must return a market")
}

func TestGetIsolatedMarginEstimatedInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginEstimatedInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty)

	_, err = e.GetIsolatedMarginEstimatedInterestRate(t.Context(), currency.Currencies{currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetIsolatedMarginEstimatedInterestRate(t.Context(), currency.Currencies{
		currency.USDT,
		currency.BTC,
		currency.ETH,
		currency.XRP,
		currency.LTC,
		currency.DOGE,
		currency.BCH,
		currency.SOL,
		currency.ADA,
		currency.DOT,
		currency.MATIC,
	})
	require.ErrorIs(t, err, errTooManyCurrencyCodes)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	got, err := e.GetIsolatedMarginEstimatedInterestRate(t.Context(), currency.Currencies{currency.BTC, currency.USDT})
	require.NoError(t, err)
	val, ok := got["BTC"]
	require.True(t, ok, "result map must contain BTC key")
	require.Positive(t, val.Float64(), "estimated interest rate must not be 0")
}

func TestGetIsolatedMarginLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetIsolatedMarginLoans(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.NoError(t, err)
}

func TestIsolatedMarginBorrowOrRepay(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), nil), errNilArgument)
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		Currency: currency.BTC,
		Type:     "borrow",
		Amount:   1,
	}), currency.ErrCurrencyPairEmpty)
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Type:         "borrow",
		Amount:       1,
	}), currency.ErrCurrencyCodeEmpty)
	assert.ErrorContains(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Currency:     currency.BTC,
		Type:         "invalid",
		Amount:       1,
	}), "invalid isolated margin loan type")
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Currency:     currency.BTC,
		Type:         "borrow",
		Amount:       0,
	}), errInvalidAmount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	assert.NoError(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Currency:     currency.BTC,
		Type:         "repay",
		Amount:       0.00004,
	}))
}

func TestGetIsolatedMarginLoanRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetIsolatedMarginLoanRecords(t.Context(), "", currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.NoError(t, err)
}

func TestGetIsolatedMarginInterestDeductionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginInterestDeductionRecords(t.Context(), BTCUSDT, currency.BTC, 0, 101, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidLimit)
	tn := time.Now()
	_, err = e.GetIsolatedMarginInterestDeductionRecords(t.Context(), BTCUSDT, currency.BTC, 0, 0, tn.Add(time.Hour), tn)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedMarginInterestDeductionRecords(t.Context(), currency.EMPTYPAIR, currency.EMPTYCODE, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
}

func TestGetIsolatedMarginMaxBorrowableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginMaxBorrowableAmount(t.Context(), currency.EMPTYCODE, BTCUSDT)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetIsolatedMarginMaxBorrowableAmount(t.Context(), currency.BTC, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedMarginMaxBorrowableAmount(t.Context(), currency.BTC, BTCUSDT)
	require.NoError(t, err, "GetIsolatedMarginMaxBorrowableAmount must not error")
}

func TestGetIsolatedMarginUserLeverageTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginUserLeverageTiers(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	tiers, err := e.GetIsolatedMarginUserLeverageTiers(t.Context(), BTCUSDT)
	require.NoError(t, err, "GetIsolatedMarginUserLeverageTiers must not error")
	require.NotEmpty(t, tiers, "GetIsolatedMarginUserLeverageTiers must return some tiers")
}

func TestGetIsolatedMarginMarketLeverageTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginMarketLeverageTiers(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	tiers, err := e.GetIsolatedMarginMarketLeverageTiers(t.Context(), BTCUSDT)
	require.NoError(t, err, "GetIsolatedMarginMarketLeverageTiers must not error")
	require.NotEmpty(t, tiers, "GetIsolatedMarginMarketLeverageTiers must return some tiers")
}

func TestSetUserMarketLeverageMultiplier(t *testing.T) {
	t.Parallel()
	err := e.SetUserMarketLeverageMultiplier(t.Context(), currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	err = e.SetUserMarketLeverageMultiplier(t.Context(), BTCUSDT, 0)
	require.ErrorIs(t, err, errInvalidLeverage)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetUserMarketLeverageMultiplier(t.Context(), BTCUSDT, 20)
	require.NoError(t, err, "SetUserMarketLeverageMultiplier must not error")
}

func TestGetIsolatedMarginAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetIsolatedMarginAccountList(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetIsolatedMarginAccountList must not error")
}

func TestGetIsolatedMarginPoolLoans(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginPoolLoans(t.Context(), currency.BTC, 1, 101)
	require.ErrorIs(t, err, errInvalidLimit)

	got, err := e.GetIsolatedMarginPoolLoans(t.Context(), currency.BTC, 0, 0)
	if err != nil {
		require.ErrorContains(t, err, "504")
		return
	}
	require.NoError(t, err, "GetIsolatedMarginPoolLoans must not error")
	require.NotEmpty(t, got, "GetIsolatedMarginPoolLoans must return some loans")
}
