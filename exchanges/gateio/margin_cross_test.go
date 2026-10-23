package gateio

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestCurrencySupportedByCrossMargin(t *testing.T) {
	t.Parallel()
	result, err := e.CurrencySupportedByCrossMargin(t.Context())
	require.NoError(t, err, "CurrencySupportedByCrossMargin must not error")
	if mockTests {
		require.Len(t, result, 7, "CurrencySupportedByCrossMargin must return every recorded currency")
		exp := &CrossMarginCurrenciesResponse{
			Name:                 currency.BTC,
			Rate:                 0.0002,
			CurrencyPrecision:    0.000001,
			Discount:             1,
			MinBorrowAmount:      0.01,
			UserMaxBorrowAmount:  1000000,
			TotalMaxBorrowAmount: 10000000,
			Price:                1,
			Loanable:             true,
			Status:               1,
		}
		assert.Equal(t, exp, result[0], "CurrencySupportedByCrossMargin should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CurrencySupportedByCrossMargin should return currencies")
}

func TestGetCrossMarginSupportedCurrencyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossMarginSupportedCurrencyDetail(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCrossMarginSupportedCurrencyDetail must reject an empty currency")

	result, err := e.GetCrossMarginSupportedCurrencyDetail(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCrossMarginSupportedCurrencyDetail must not error")
	if mockTests {
		exp := &CrossMarginCurrenciesResponse{
			Name:                 currency.BTC,
			CurrencyPrecision:    0.000001,
			Discount:             1,
			MinBorrowAmount:      0.00001,
			UserMaxBorrowAmount:  120000000,
			TotalMaxBorrowAmount: 300000000,
			Price:                82139.13,
			Loanable:             true,
			Status:               1,
		}
		assert.Equal(t, exp, result, "GetCrossMarginSupportedCurrencyDetail should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Name, "GetCrossMarginSupportedCurrencyDetail should return the requested currency")
}

func TestGetCrossMarginAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginAccounts(t.Context())
	require.NoError(t, err, "GetCrossMarginAccounts must not error")
	if mockTests {
		exp := &CrossMarginAccountResponse{
			UserID: 10001,
			Balances: map[string]CrossMarginCurrencyBalance{
				"BTC": {Available: 0.5, Freeze: 0.1, Borrowed: 0.02, Interest: 0.0001},
			},
			Total:                       48050,
			Borrowed:                    1600,
			Interest:                    8,
			Risk:                        25.6,
			TotalInitialMargin:          160,
			TotalMarginBalance:          48050,
			TotalMaintenanceMargin:      80,
			TotalInitialMarginRate:      300.31,
			TotalMaintenanceMarginRate:  600.62,
			TotalAvailableMargin:        47890,
			TotalPortfolioMarginAccount: 48050,
		}
		assert.Equal(t, exp, result, "GetCrossMarginAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossMarginAccounts should return an account")
}

func TestGetCrossMarginAccountChangeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetCrossMarginAccountChangeHistory(t.Context(), currency.BTC, endTime, startTime, 0, 6, "in")
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCrossMarginAccountChangeHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginAccountChangeHistory(t.Context(), currency.BTC, startTime, endTime, 1, 6, "in")
	require.NoError(t, err, "GetCrossMarginAccountChangeHistory must not error")
	if mockTests {
		exp := []*CrossMarginAccountHistoryItemResponse{{
			ID:       "123456",
			Time:     types.Time(time.UnixMilli(1744150000000)),
			Currency: currency.BTC,
			Change:   1.03,
			Balance:  4.59316525194,
			Type:     "in",
		}}
		assert.Equal(t, exp, result, "GetCrossMarginAccountChangeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossMarginAccountChangeHistory should return a result")
}

func TestCreateCrossMarginBorrowLoanNullReply(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v4/margin/cross/loans", r.URL.Path, "request path should be the cross margin loans")
		_, err := w.Write([]byte(`null`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	_, err := ex.CreateCrossMarginBorrowLoan(t.Context(), &CrossMarginBorrowLoanRequest{Currency: currency.BTC, Amount: 3})
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateCrossMarginBorrowLoan should reject a null reply")
}

func TestCreateCrossMarginBorrowLoan(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCrossMarginBorrowLoan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateCrossMarginBorrowLoan must reject a nil request")

	_, err = e.CreateCrossMarginBorrowLoan(t.Context(), &CrossMarginBorrowLoanRequest{Currency: currency.EMPTYCODE, Amount: 3})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateCrossMarginBorrowLoan must reject an empty currency")

	_, err = e.CreateCrossMarginBorrowLoan(t.Context(), &CrossMarginBorrowLoanRequest{Currency: currency.BTC, Amount: 0})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateCrossMarginBorrowLoan must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateCrossMarginBorrowLoan(t.Context(), &CrossMarginBorrowLoanRequest{Currency: currency.BTC, Amount: 3})
	require.NoError(t, err, "CreateCrossMarginBorrowLoan must not error")
	if mockTests {
		exp := &CrossMarginLoanResponse{
			ID:             "17",
			CreateTime:     types.Time(time.UnixMilli(1744150000000)),
			UpdateTime:     types.Time(time.UnixMilli(1744150000000)),
			Currency:       currency.BTC,
			Amount:         3,
			Status:         2,
			UnpaidInterest: 0.0001,
		}
		assert.Equal(t, exp, result, "CreateCrossMarginBorrowLoan should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateCrossMarginBorrowLoan should return the loan")
}

func TestGetCrossMarginBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossMarginBorrowHistory(t.Context(), 6, currency.BTC, 0, 0, false)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetCrossMarginBorrowHistory must reject an unknown status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginBorrowHistory(t.Context(), 1, currency.BTC, 0, 0, false)
	require.NoError(t, err, "GetCrossMarginBorrowHistory must not error")
	if mockTests {
		exp := []*CrossMarginLoanResponse{{
			ID:         "19",
			CreateTime: types.Time(time.UnixMilli(1744150000000)),
			UpdateTime: types.Time(time.UnixMilli(1744150000000)),
			Currency:   currency.BTC,
			Amount:     100,
			Text:       "web",
			Status:     1,
		}}
		assert.Equal(t, exp, result, "GetCrossMarginBorrowHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossMarginBorrowHistory should return a result")
}

func TestGetSingleBorrowLoanDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleBorrowLoanDetail(t.Context(), "")
	require.ErrorIs(t, err, errInvalidLoanID, "GetSingleBorrowLoanDetail must reject an empty loan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSingleBorrowLoanDetail(t.Context(), "1234")
	require.NoError(t, err, "GetSingleBorrowLoanDetail must not error")
	if mockTests {
		exp := &CrossMarginLoanResponse{
			ID:             "1234",
			CreateTime:     types.Time(time.UnixMilli(1744103854944)),
			UpdateTime:     types.Time(time.UnixMilli(1744150000000)),
			Currency:       currency.BTC,
			Amount:         0.5,
			Text:           "web",
			Status:         2,
			Repaid:         0.1,
			RepaidInterest: 0.00001,
			UnpaidInterest: 0.00004,
		}
		assert.Equal(t, exp, result, "GetSingleBorrowLoanDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSingleBorrowLoanDetail should return the loan")
}

func TestExecuteRepayment(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteRepayment(t.Context(), CurrencyAndAmountResponse{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ExecuteRepayment must reject an empty currency")

	_, err = e.ExecuteRepayment(t.Context(), CurrencyAndAmountResponse{Currency: currency.BTC})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "ExecuteRepayment must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ExecuteRepayment(t.Context(), CurrencyAndAmountResponse{Currency: currency.USDT, Amount: 1234.55})
	require.NoError(t, err, "ExecuteRepayment must not error")
	if mockTests {
		exp := []*CrossMarginLoanResponse{{
			ID:             "18",
			CreateTime:     types.Time(time.UnixMilli(1744103854944)),
			UpdateTime:     types.Time(time.UnixMilli(1744150000000)),
			Currency:       currency.USDT,
			Amount:         1234.55,
			Text:           "web",
			Status:         3,
			Repaid:         1234.55,
			RepaidInterest: 0.04,
		}}
		assert.Equal(t, exp, result, "ExecuteRepayment should decode every field")
		return
	}
	assert.NotNil(t, result, "ExecuteRepayment should return the repaid loans")
}

func TestGetCrossMarginRepayments(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginRepayments(t.Context(), currency.BTC, "123", 1000, 10, true)
	require.NoError(t, err, "GetCrossMarginRepayments must not error")
	if mockTests {
		exp := []*CrossMarginLoanResponse{{
			ID:             "123",
			CreateTime:     types.Time(time.UnixMilli(1744103854944)),
			UpdateTime:     types.Time(time.UnixMilli(1744150000000)),
			Currency:       currency.BTC,
			Amount:         0.5,
			Text:           "web",
			Status:         3,
			Repaid:         0.5,
			RepaidInterest: 0.00002,
		}}
		assert.Equal(t, exp, result, "GetCrossMarginRepayments should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossMarginRepayments should return a result")
}

func TestGetMaxTransferableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxTransferableAmountForSpecificCrossMarginCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaxTransferableAmountForSpecificCrossMarginCurrency must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaxTransferableAmountForSpecificCrossMarginCurrency(t.Context(), currency.BTC)
	require.NoError(t, err, "GetMaxTransferableAmountForSpecificCrossMarginCurrency must not error")
	if mockTests {
		exp := &CurrencyAndAmountResponse{Currency: currency.BTC, Amount: 1.5}
		assert.Equal(t, exp, result, "GetMaxTransferableAmountForSpecificCrossMarginCurrency should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Currency, "GetMaxTransferableAmountForSpecificCrossMarginCurrency should return the requested currency")
}

func TestGetMaxBorrowableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxBorrowableAmountForSpecificCrossMarginCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaxBorrowableAmountForSpecificCrossMarginCurrency must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaxBorrowableAmountForSpecificCrossMarginCurrency(t.Context(), currency.BTC)
	require.NoError(t, err, "GetMaxBorrowableAmountForSpecificCrossMarginCurrency must not error")
	if mockTests {
		exp := &CurrencyAndAmountResponse{Currency: currency.BTC, Amount: 0.8}
		assert.Equal(t, exp, result, "GetMaxBorrowableAmountForSpecificCrossMarginCurrency should decode every field")
		return
	}
	assert.Equal(t, currency.BTC, result.Currency, "GetMaxBorrowableAmountForSpecificCrossMarginCurrency should return the requested currency")
}
