package gate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetLendingCurrencyList(t *testing.T) {
	t.Parallel()
	result, err := e.GetLendingCurrencyList(t.Context())
	require.NoError(t, err, "GetLendingCurrencyList must not error")
	if mockTests {
		require.Len(t, result, 5, "GetLendingCurrencyList must return every recorded currency")
		exp := &LendingCurrencyDetailResponse{Currency: currency.USDT, MinLendAmount: 1, MaxLendAmountUSDT: 100000000, MaxHourlyRate: 0.00057, MinHourlyRate: 0.00000011}
		assert.Equal(t, exp, result[0], "GetLendingCurrencyList should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetLendingCurrencyList should return currencies")
}

func TestGetLendingCurrencyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetLendingCurrencyDetail(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLendingCurrencyDetail must require a currency")

	result, err := e.GetLendingCurrencyDetail(t.Context(), currency.ETH)
	require.NoError(t, err, "GetLendingCurrencyDetail must not error")
	if mockTests {
		exp := &LendingCurrencyDetailResponse{Currency: currency.ETH, MinLendAmount: 0.001, MaxLendAmountUSDT: 500000000, MaxHourlyRate: 0.00057, MinHourlyRate: 0.00000011}
		assert.Equal(t, exp, result, "GetLendingCurrencyDetail should decode every field")
		return
	}
	require.NotNil(t, result, "GetLendingCurrencyDetail must return a result")
	assert.True(t, result.Currency.Equal(currency.ETH), "GetLendingCurrencyDetail should return the requested currency")
}

func TestCreateLendingOrRedemption(t *testing.T) {
	t.Parallel()
	err := e.CreateLendingOrRedemption(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateLendingOrRedemption must reject a nil request")

	arg := &LendingOrRedemptionRequest{}
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateLendingOrRedemption must require a currency")

	arg.Currency = currency.USDT
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateLendingOrRedemption must require an amount")

	arg.Amount = 1
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, errLoanTypeIsRequired, "CreateLendingOrRedemption must require an operation type")

	arg.Type = "lend"
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateLendingOrRedemption must require a minimum rate for lending")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.MinHourlyRate = 0.00000211
	err = e.CreateLendingOrRedemption(t.Context(), arg)
	require.NoError(t, err, "CreateLendingOrRedemption must not error when lending")

	err = e.CreateLendingOrRedemption(t.Context(), &LendingOrRedemptionRequest{Currency: currency.USDT, Amount: 1, Type: "redeem"})
	assert.NoError(t, err, "CreateLendingOrRedemption should not require a minimum rate when redeeming")
}

func TestGetUserLendingOrderList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserLendingOrderList(t.Context(), currency.USDT, 10, 100)
	require.NoError(t, err, "GetUserLendingOrderList must not error")
	if mockTests {
		exp := []*LendOrderDetailResponse{{
			Currency:       currency.USDT,
			CurrentAmount:  20.999992,
			Amount:         20.999992,
			MinRate:        0.1,
			InterestStatus: "interest_dividend",
			CreateTime:     types.Time(time.UnixMilli(1673247054000)),
			UpdateTime:     types.Time(time.UnixMilli(1673247054000)),
		}}
		assert.Equal(t, exp, result, "GetUserLendingOrderList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserLendingOrderList should return a result")
}

func TestAmendUserLendingInformation(t *testing.T) {
	t.Parallel()
	err := e.AmendUserLendingInformation(t.Context(), currency.EMPTYCODE, 123.45)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AmendUserLendingInformation must require a currency")

	err = e.AmendUserLendingInformation(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "AmendUserLendingInformation must require a minimum rate")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.AmendUserLendingInformation(t.Context(), currency.USDT, 0.0001)
	assert.NoError(t, err, "AmendUserLendingInformation should not error")
}

func TestGetLendingTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetLendingTransactionRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLendingTransactionRecords must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetLendingTransactionRecords(t.Context(), &LendingTransactionRecordsRequest{Currency: currency.USDT, Page: 1, Limit: 100, From: endTime, To: startTime, Type: "lend"})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLendingTransactionRecords must reject a reversed time range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLendingTransactionRecords(t.Context(), &LendingTransactionRecordsRequest{Currency: currency.USDT, Page: 10, Limit: 100, From: startTime, To: endTime, Type: "redeem"})
	require.NoError(t, err, "GetLendingTransactionRecords must not error")
	if mockTests {
		exp := []*LendingTransactionRecordResponse{{
			Type:             "redeem",
			Currency:         currency.USDT,
			Amount:           1,
			LastWalletAmount: 0.2,
			CreateTime:       types.Time(time.UnixMilli(1744150000000)),
		}}
		assert.Equal(t, exp, result, "GetLendingTransactionRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLendingTransactionRecords should return a result")
}

func TestGetUserTotalInterestIncomePerCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTotalInterestIncomePerCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUserTotalInterestIncomePerCurrency must require a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserTotalInterestIncomePerCurrency(t.Context(), currency.USDT)
	require.NoError(t, err, "GetUserTotalInterestIncomePerCurrency must not error")
	if mockTests {
		assert.Equal(t, &CurrencyAndInterestIncomeResponse{Currency: currency.USDT, Interest: 123.345}, result, "GetUserTotalInterestIncomePerCurrency should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserTotalInterestIncomePerCurrency should return a result")
}

func TestGetUserDividendRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserDividendRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserDividendRecords must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetUserDividendRecords(t.Context(), &UserDividendRecordsRequest{Currency: currency.USDT, Page: 1, Limit: 100, From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserDividendRecords must reject a reversed time range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserDividendRecords(t.Context(), &UserDividendRecordsRequest{Currency: currency.AE, From: startTime, To: endTime})
	require.NoError(t, err, "GetUserDividendRecords must not error")
	if mockTests {
		exp := []*UserDividendRecordsResponse{{
			Status:         1,
			Currency:       currency.AE,
			ActualRate:     0.0005,
			Interest:       0.05,
			InterestStatus: "interest_dividend",
			CreateTime:     types.Time(time.UnixMilli(1744150000000)),
		}}
		assert.Equal(t, exp, result, "GetUserDividendRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserDividendRecords should return a result")
}

func TestGetCurrencyInterestCompoundingStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyInterestCompoundingStatus(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCurrencyInterestCompoundingStatus must require a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrencyInterestCompoundingStatus(t.Context(), currency.USDT)
	require.NoError(t, err, "GetCurrencyInterestCompoundingStatus must not error")
	if mockTests {
		assert.Equal(t, &CurrencyInterestStatusResponse{Currency: currency.USDT, InterestStatus: "interest_dividend"}, result, "GetCurrencyInterestCompoundingStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrencyInterestCompoundingStatus should return a result")
}

func TestGetUniLoanCurrencyAnnualisedTrendChart(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUniLoanCurrencyAnnualisedTrendChart(t.Context(), time.Time{}, endTime, currency.AE)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUniLoanCurrencyAnnualisedTrendChart must require a start time")

	_, err = e.GetUniLoanCurrencyAnnualisedTrendChart(t.Context(), endTime, startTime, currency.AE)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUniLoanCurrencyAnnualisedTrendChart must reject a reversed time range")

	_, err = e.GetUniLoanCurrencyAnnualisedTrendChart(t.Context(), endTime.Add(-31*24*time.Hour), endTime, currency.AE)
	require.ErrorIs(t, err, errUniChartRangeExceeded, "GetUniLoanCurrencyAnnualisedTrendChart must reject a range beyond 30 days")

	_, err = e.GetUniLoanCurrencyAnnualisedTrendChart(t.Context(), startTime, endTime, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUniLoanCurrencyAnnualisedTrendChart must require a currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUniLoanCurrencyAnnualisedTrendChart(t.Context(), startTime, endTime, currency.AE)
	require.NoError(t, err, "GetUniLoanCurrencyAnnualisedTrendChart must not error")
	if mockTests {
		assert.Equal(t, []*UniLoanAssetResponse{{Time: types.Time(time.Unix(1744156800, 0)), Value: 0.01}}, result, "GetUniLoanCurrencyAnnualisedTrendChart should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUniLoanCurrencyAnnualisedTrendChart should return a result")
}

func TestGetCurrencyEstimatedAnnualisedInterestRate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrencyEstimatedAnnualisedInterestRate(t.Context())
	require.NoError(t, err, "GetCurrencyEstimatedAnnualisedInterestRate must not error")
	if mockTests {
		assert.Equal(t, []*CurrencyEstimatedAnnualInterestRateResponse{{Currency: currency.USDT, EstimatedAnnualisedRate: 0.0226}}, result, "GetCurrencyEstimatedAnnualisedInterestRate should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrencyEstimatedAnnualisedInterestRate should return a result")
}
