package gateio

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestTransferCollateralToIsolatedMargin(t *testing.T) {
	t.Parallel()
	_, err := e.TransferCollateralToIsolatedMargin(t.Context(), BTCUSDT, currency.EMPTYCODE, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "empty currency code must return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.TransferCollateralToIsolatedMargin(t.Context(), BTCUSDT, currency.USDT, -10)
	require.NoError(t, err, "TransferCollateralToIsolatedMargin must not error")
}

func TestTransferCollateralFromIsolatedMargin(t *testing.T) {
	t.Parallel()
	_, err := e.TransferCollateralFromIsolatedMargin(t.Context(), BTCUSDT, currency.EMPTYCODE, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "empty currency code must return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.TransferCollateralFromIsolatedMargin(t.Context(), BTCUSDT, currency.USDT, -1)
	require.NoError(t, err, "TransferCollateralFromIsolatedMargin must not error")
}

func TestGetIsolatedMarginAccountBalanceChangeHistory(t *testing.T) {
	t.Parallel()
	tn := time.Now()
	_, err := e.GetIsolatedMarginAccountBalanceChangeHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, tn.Add(time.Hour), tn, 0, 0, "")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "start time after end time must return the expected error")

	_, err = e.GetIsolatedMarginAccountBalanceChangeHistory(t.Context(), currency.BTC, BTCUSDT, time.Time{}, time.Time{}, 1, 100, "invalid")
	require.ErrorIs(t, err, errInvalidIsolatedMarginAccountType, "invalid account type must return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedMarginAccountBalanceChangeHistory(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 0, "margin_out")
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
	_, err := e.GetIsolatedMarginMaxTransferableAmount(t.Context(), currency.EMPTYCODE, BTCUSDT)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "empty currency code must return the expected error")

	_, err = e.GetIsolatedMarginMaxTransferableAmount(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "empty currency pair must return the expected error")

	for _, pair := range []currency.Pair{
		currency.NewPair(currency.BTC, currency.BTC),
		currency.NewPair(currency.BTC, currency.EMPTYCODE),
	} {
		_, err = e.GetIsolatedMarginMaxTransferableAmount(t.Context(), currency.BTC, pair)
		require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "malformed currency pair must return the expected error")
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedMarginMaxTransferableAmount(t.Context(), currency.USDT, BTCUSDT)
	require.NoError(t, err, "GetIsolatedMarginMaxTransferableAmount must not error")
}

func TestIsolatedMarginLendingMarketIsTradable(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)

	for _, tc := range []struct {
		name         string
		status       string
		delistedTime time.Time
		expected     bool
	}{
		{name: "enabled without delisting", status: "enabled", expected: true},
		{name: "disabled", status: "disabled", expected: false},
		{name: "past delisting", status: "enabled", delistedTime: now.Add(-time.Second), expected: false},
		{name: "delisting now", status: "enabled", delistedTime: now, expected: true},
		{name: "future delisting", status: "enabled", delistedTime: now.Add(time.Second), expected: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			market := IsolatedMarginLendingMarket{Status: tc.status, DelistedTime: types.Time(tc.delistedTime)}
			assert.Equal(t, tc.expected, market.IsTradable(now), "tradability should match the market status and delisting time")
		})
	}
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
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "empty currency pair must return the expected error")

	market, err := e.GetIsolatedMarginLendingMarketDetails(t.Context(), BTCUSDT)
	require.NoError(t, err, "GetIsolatedMarginLendingMarketDetails must not error")
	require.NotNil(t, market, "GetIsolatedMarginLendingMarketDetails must return a market")
}

func TestGetIsolatedMarginEstimatedInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginEstimatedInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "nil currencies must return the expected error")

	_, err = e.GetIsolatedMarginEstimatedInterestRate(t.Context(), currency.Currencies{currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "empty currency code must return the expected error")

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
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "too many currency codes must return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	got, err := e.GetIsolatedMarginEstimatedInterestRate(t.Context(), currency.Currencies{currency.BTC, currency.USDT})
	require.NoError(t, err, "GetIsolatedMarginEstimatedInterestRate must not error")
	val, ok := got["BTC"]
	require.True(t, ok, "result map must contain BTC key")
	require.Positive(t, val.Float64(), "estimated interest rate must not be 0")
}

func TestGetIsolatedMarginLoans(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginLoans(t.Context(), currency.BTC, currency.NewBTCUSD(), 0, 101)
	require.ErrorIs(t, err, errInvalidLimit, "limit above maximum must return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedMarginLoans(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0)
	require.NoError(t, err, "GetIsolatedMarginLoans must not error")
}

func TestIsolatedMarginBorrowOrRepay(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), nil), errNilArgument, "nil request should return the expected error")
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		Currency: currency.BTC,
		Type:     "borrow",
		Amount:   1,
	}), currency.ErrCurrencyPairEmpty, "empty currency pair should return the expected error")
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Type:         "borrow",
		Amount:       1,
	}), currency.ErrCurrencyCodeEmpty, "empty currency code should return the expected error")
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Currency:     currency.BTC,
		Type:         "invalid",
		Amount:       1,
	}), errInvalidIsolatedMarginLoanType, "invalid loan type should return the expected error")
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Currency:     currency.BTC,
		Type:         "borrow",
		Amount:       0,
	}), errInvalidAmount, "zero amount should return the expected error")
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Currency:     currency.BTC,
		Type:         "repay",
		Amount:       1,
		RepaidAll:    true,
	}), errAmountOverriddenByRepaidAll, "amount with full repayment should return the expected error")
	assert.ErrorIs(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Currency:     currency.BTC,
		Type:         "borrow",
		RepaidAll:    true,
	}), errInvalidRepaidAllOperation, "full repayment on a borrow should return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	assert.NoError(t, e.IsolatedMarginBorrowOrRepay(t.Context(), &IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Currency:     currency.BTC,
		Type:         "repay",
		Amount:       0.00004,
	}), "IsolatedMarginBorrowOrRepay should not error")
}

func TestIsolatedBorrowRepayRequestFullRepayment(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(&IsolatedBorrowRepayRequest{
		CurrencyPair: BTCUSDT,
		Currency:     currency.BTC,
		Type:         "repay",
		RepaidAll:    true,
	})
	require.NoError(t, err, "marshalling a full repayment request must not error")
	assert.JSONEq(t, `{"currency_pair":"BTC_USDT","currency":"BTC","type":"repay","amount":"","repaid_all":true}`, string(payload), "full repayment request should include repaid_all")
}

func TestGetIsolatedMarginLoanRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginLoanRecords(t.Context(), currency.BTC, currency.NewBTCUSDT(), 0, 101, "")
	require.ErrorIs(t, err, errInvalidLimit, "limit above maximum must return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedMarginLoanRecords(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0, "")
	require.NoError(t, err, "GetIsolatedMarginLoanRecords must not error")
}

func TestGetIsolatedMarginInterestDeductionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginInterestDeductionRecords(t.Context(), currency.BTC, BTCUSDT, 0, 1001, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidLimit, "limit above maximum must return the expected error")
	tn := time.Now()
	_, err = e.GetIsolatedMarginInterestDeductionRecords(t.Context(), currency.BTC, BTCUSDT, 0, 0, tn.Add(time.Hour), tn)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "start time after end time must return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedMarginInterestDeductionRecords(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err, "GetIsolatedMarginInterestDeductionRecords must not error")
}

func TestGetIsolatedMarginMaxBorrowableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginMaxBorrowableAmount(t.Context(), currency.EMPTYCODE, BTCUSDT)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "empty currency code must return the expected error")

	_, err = e.GetIsolatedMarginMaxBorrowableAmount(t.Context(), currency.BTC, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "empty currency pair must return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedMarginMaxBorrowableAmount(t.Context(), currency.BTC, BTCUSDT)
	require.NoError(t, err, "GetIsolatedMarginMaxBorrowableAmount must not error")
}

func TestGetIsolatedMarginUserLeverageTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginUserLeverageTiers(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "empty currency pair should return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	tiers, err := e.GetIsolatedMarginUserLeverageTiers(t.Context(), BTCUSDT)
	require.NoError(t, err, "GetIsolatedMarginUserLeverageTiers must not error")
	require.NotEmpty(t, tiers, "GetIsolatedMarginUserLeverageTiers must return some tiers")
}

func TestGetIsolatedMarginMarketLeverageTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginMarketLeverageTiers(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "empty currency pair should return the expected error")

	tiers, err := e.GetIsolatedMarginMarketLeverageTiers(t.Context(), BTCUSDT)
	require.NoError(t, err, "GetIsolatedMarginMarketLeverageTiers must not error")
	require.NotEmpty(t, tiers, "GetIsolatedMarginMarketLeverageTiers must return some tiers")
}

func TestSetUserMarketLeverageMultiplier(t *testing.T) {
	t.Parallel()
	err := e.SetUserMarketLeverageMultiplier(t.Context(), currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "empty currency pair must return the expected error")
	err = e.SetUserMarketLeverageMultiplier(t.Context(), BTCUSDT, 0)
	require.ErrorIs(t, err, errInvalidLeverage, "zero leverage must return the expected error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetUserMarketLeverageMultiplier(t.Context(), BTCUSDT, 1)
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
	ctx := request.WithHeaders(t.Context(), http.Header{
		"User-Agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36"},
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		"Accept-Language":           {"en-US,en;q=0.9"},
		"Sec-Ch-Ua":                 {`"Chromium";v="148", "Google Chrome";v="148", "Not/A)Brand";v="99"`},
		"Sec-Ch-Ua-Mobile":          {"?0"},
		"Sec-Ch-Ua-Platform":        {`"Windows"`},
		"Sec-Fetch-Dest":            {"document"},
		"Sec-Fetch-Mode":            {"navigate"},
		"Sec-Fetch-Site":            {"none"},
		"Sec-Fetch-User":            {"?1"},
		"Upgrade-Insecure-Requests": {"1"},
	})
	_, err := e.GetIsolatedMarginPoolLoans(ctx, currency.BTC, 18446744073709551615, 100)
	require.ErrorContains(t, err, "Type mismatch", "page value above the API range must return a type mismatch")

	got, err := e.GetIsolatedMarginPoolLoans(ctx, currency.BTC, 0, 100)
	if err != nil {
		require.ErrorContains(t, err, "504", "pool loan request failure must be a gateway timeout")
		return
	}
	require.NoError(t, err, "GetIsolatedMarginPoolLoans must not error")
	require.NotEmpty(t, got, "GetIsolatedMarginPoolLoans must return some loans")
}
