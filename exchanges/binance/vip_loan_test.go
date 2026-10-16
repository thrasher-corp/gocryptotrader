package binance

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
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetVIPBorrowInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetVIPBorrowInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetVIPBorrowInterestRate must reject no loan coins")
	_, err = e.GetVIPBorrowInterestRate(t.Context(), []currency.Code{currency.ETH, currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetVIPBorrowInterestRate must reject an empty loan coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetVIPBorrowInterestRate(t.Context(), []currency.Code{currency.ETH, currency.BTC})
	require.NoError(t, err, "GetVIPBorrowInterestRate must not error")
	if mockTests {
		exp := []VIPLoanBorrowInterestRate{
			{
				Asset:                      currency.ETH,
				FlexibleDailyInterestRate:  0.000127,
				FlexibleYearlyInterestRate: 0.046355,
				Time:                       types.Time(time.Unix(1744190254, 0)),
			},
			{
				Asset:                      currency.BTC,
				FlexibleDailyInterestRate:  0.000101,
				FlexibleYearlyInterestRate: 0.036865,
				Time:                       types.Time(time.Unix(1744190254, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetVIPBorrowInterestRate should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetVIPBorrowInterestRate should return rates")
}

func TestGetVIPCollateralAssetData(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetVIPCollateralAssetData(t.Context(), currency.BTC)
	require.NoError(t, err, "GetVIPCollateralAssetData must not error")
	if mockTests {
		exp := &VIPCollateralAssetDataResponse{
			Rows: []VIPCollateralAsset{
				{
					CollateralCoin:        currency.BTC,
					FirstCollateralRatio:  "100%",
					FirstCollateralRange:  "1-10000000",
					SecondCollateralRatio: "80%",
					SecondCollateralRange: "10000000-100000000",
					ThirdCollateralRatio:  "60%",
					ThirdCollateralRange:  "100000000-1000000000",
					FourthCollateralRatio: "40%",
					FourthCollateralRange: ">1000000000",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetVIPCollateralAssetData should decode every field")
		return
	}
	assert.NotNil(t, result, "GetVIPCollateralAssetData should return a result")
}

func TestGetVIPLoanableAssetsData(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetVIPLoanableAssetsData(t.Context(), currency.BTC, 2)
	require.NoError(t, err, "GetVIPLoanableAssetsData must not error")
	if mockTests {
		exp := &VIPLoanableAssetsDataResponse{
			Rows: []VIPLoanableAsset{
				{
					LoanCoin:                    currency.BTC,
					FlexibleDailyInterestRate:   0.000101,
					FlexibleYearlyInterestRate:  0.036865,
					ThirtyDayDailyInterestRate:  0.000136,
					ThirtyDayYearlyInterestRate: 0.0345,
					SixtyDayDailyInterestRate:   0.000145,
					SixtyDayYearlyInterestRate:  0.04103,
					MinimumLimit:                100,
					MaximumLimit:                1000000,
					VIPLevel:                    2,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetVIPLoanableAssetsData should decode every field")
		return
	}
	assert.NotNil(t, result, "GetVIPLoanableAssetsData should return a result")
}

func TestGetVIPLoanInterestRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetVIPLoanInterestRateHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetVIPLoanInterestRateHistory must reject a nil request")
	_, err = e.GetVIPLoanInterestRateHistory(t.Context(), &VIPLoanInterestRateHistoryRequest{Limit: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetVIPLoanInterestRateHistory must reject an empty coin")
	startTime, endTime := getTime()
	_, err = e.GetVIPLoanInterestRateHistory(t.Context(), &VIPLoanInterestRateHistoryRequest{Coin: currency.BTC, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetVIPLoanInterestRateHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetVIPLoanInterestRateHistory(t.Context(), &VIPLoanInterestRateHistoryRequest{
		Coin:      currency.BTC,
		StartTime: startTime,
		EndTime:   endTime,
		Current:   1,
		Limit:     100,
	})
	require.NoError(t, err, "GetVIPLoanInterestRateHistory must not error")
	if mockTests {
		exp := &VIPLoanInterestRateHistoryResponse{
			Rows: []VIPLoanInterestRate{
				{
					Coin:                   currency.BTC,
					AnnualizedInterestRate: 0.0647,
					Time:                   types.Time(time.Unix(1744156800, 0)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetVIPLoanInterestRateHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetVIPLoanInterestRateHistory should return a result")
}

func TestVIPLoanBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.VIPLoanBorrow(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "VIPLoanBorrow must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *VIPLoanBorrowRequest
		err  error
	}{
		{"loan account ID", &VIPLoanBorrowRequest{LoanCoin: currency.ETH, LoanAmount: 123, CollateralAccountIDs: []uint64{1234}, CollateralCoins: []currency.Code{currency.LTC}, LoanTerm: 30}, errAccountIDRequired},
		{"loan coin", &VIPLoanBorrowRequest{LoanAccountID: 1234, LoanAmount: 123, CollateralAccountIDs: []uint64{1234}, CollateralCoins: []currency.Code{currency.LTC}, LoanTerm: 30}, currency.ErrCurrencyCodeEmpty},
		{"loan amount", &VIPLoanBorrowRequest{LoanAccountID: 1234, LoanCoin: currency.ETH, CollateralAccountIDs: []uint64{1234}, CollateralCoins: []currency.Code{currency.LTC}, LoanTerm: 30}, limits.ErrAmountBelowMin},
		{"collateral account IDs", &VIPLoanBorrowRequest{LoanAccountID: 1234, LoanCoin: currency.ETH, LoanAmount: 123, CollateralCoins: []currency.Code{currency.LTC}, LoanTerm: 30}, errAccountIDRequired},
		{"collateral coins", &VIPLoanBorrowRequest{LoanAccountID: 1234, LoanCoin: currency.ETH, LoanAmount: 123, CollateralAccountIDs: []uint64{1234}, LoanTerm: 30}, currency.ErrCurrencyCodeEmpty},
		{"fixed rate term", &VIPLoanBorrowRequest{LoanAccountID: 1234, LoanCoin: currency.ETH, LoanAmount: 123, CollateralAccountIDs: []uint64{1234}, CollateralCoins: []currency.Code{currency.LTC}}, errLoanTermMustBeSet},
		{"zero collateral account ID", &VIPLoanBorrowRequest{LoanAccountID: 1234, LoanCoin: currency.ETH, LoanAmount: 123, CollateralAccountIDs: []uint64{1234, 0}, CollateralCoins: []currency.Code{currency.LTC}, IsFlexibleRate: true}, errAccountIDRequired},
		{"empty collateral coin", &VIPLoanBorrowRequest{LoanAccountID: 1234, LoanCoin: currency.ETH, LoanAmount: 123, CollateralAccountIDs: []uint64{1234}, CollateralCoins: []currency.Code{currency.EMPTYCODE}, IsFlexibleRate: true}, currency.ErrCurrencyCodeEmpty},
	} {
		_, err = e.VIPLoanBorrow(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "VIPLoanBorrow must reject an invalid %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.VIPLoanBorrow(t.Context(), &VIPLoanBorrowRequest{
		LoanAccountID:        1234,
		LoanCoin:             currency.ETH,
		LoanAmount:           123,
		CollateralAccountIDs: []uint64{1234},
		CollateralCoins:      []currency.Code{currency.LTC},
		LoanTerm:             30,
	})
	require.NoError(t, err, "VIPLoanBorrow must not error")
	if mockTests {
		exp := &VIPLoanBorrowResponse{
			LoanAccountID:       "1234",
			RequestID:           "12345678",
			LoanCoin:            currency.ETH,
			IsFlexibleRate:      "No",
			LoanAmount:          123,
			CollateralAccountID: "1234",
			CollateralCoin:      "LTC",
			LoanTerm:            "30",
		}
		assert.Equal(t, exp, result, "VIPLoanBorrow should decode every field")
		return
	}
	assert.NotEmpty(t, result.RequestID, "VIPLoanBorrow should return a request ID")
}

func TestVIPLoanRenew(t *testing.T) {
	t.Parallel()
	_, err := e.VIPLoanRenew(t.Context(), 0, 60)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "VIPLoanRenew must reject an empty order ID")
	_, err = e.VIPLoanRenew(t.Context(), 1234, 0)
	require.ErrorIs(t, err, errLoanTermMustBeSet, "VIPLoanRenew must reject an empty loan term")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.VIPLoanRenew(t.Context(), 1234, 60)
	require.NoError(t, err, "VIPLoanRenew must not error")
	if mockTests {
		exp := &VIPLoanRenewResponse{
			LoanAccountID:       "12345678",
			LoanCoin:            currency.BTC,
			LoanAmount:          100.55,
			CollateralAccountID: "12345677,12345678,12345679",
			CollateralCoin:      "BUSD,USDT,ETH",
			LoanTerm:            "60",
		}
		assert.Equal(t, exp, result, "VIPLoanRenew should decode every field")
		return
	}
	assert.NotEmpty(t, result.LoanAccountID, "VIPLoanRenew should return the loan account ID")
}

func TestVIPLoanRepay(t *testing.T) {
	t.Parallel()
	_, err := e.VIPLoanRepay(t.Context(), 0, 0.2)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "VIPLoanRepay must reject an empty order ID")
	_, err = e.VIPLoanRepay(t.Context(), 1234, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "VIPLoanRepay must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.VIPLoanRepay(t.Context(), 1234, 0.2)
	require.NoError(t, err, "VIPLoanRepay must not error")
	if mockTests {
		exp := &VIPLoanRepayResponse{
			LoanCoin:           currency.BUSD,
			RepayAmount:        0.2,
			RemainingPrincipal: 100.5,
			RemainingInterest:  0.03,
			CollateralCoin:     "BNB,BTC,ETH",
			CurrentLTV:         0.25,
			RepayStatus:        "Repaid",
		}
		assert.Equal(t, exp, result, "VIPLoanRepay should decode every field")
		return
	}
	assert.NotEmpty(t, result.RepayStatus, "VIPLoanRepay should return a repayment status")
}

func TestCheckVIPLoanCollateralAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CheckVIPLoanCollateralAccount(t.Context(), 1223, 40)
	require.NoError(t, err, "CheckVIPLoanCollateralAccount must not error")
	if mockTests {
		exp := &VIPLoanCollateralAccountResponse{
			Rows: []VIPLoanCollateralAccount{
				{
					CollateralAccountID: "40",
					CollateralCoin:      "BNB,BTC,ETH",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "CheckVIPLoanCollateralAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "CheckVIPLoanCollateralAccount should return a result")
}

func TestGetVIPLoanAccruedInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetVIPLoanAccruedInterest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetVIPLoanAccruedInterest must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetVIPLoanAccruedInterest(t.Context(), &VIPLoanHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetVIPLoanAccruedInterest must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetVIPLoanAccruedInterest(t.Context(), &VIPLoanHistoryRequest{
		OrderID:   12345,
		LoanCoin:  currency.BTC,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10,
	})
	require.NoError(t, err, "GetVIPLoanAccruedInterest must not error")
	if mockTests {
		exp := &VIPLoanAccruedInterestResponse{
			Rows: []VIPLoanAccruedInterest{
				{
					LoanCoin:           currency.BTC,
					PrincipalAmount:    10000,
					InterestAmount:     1.2,
					AnnualInterestRate: 0.001273,
					AccrualTime:        types.Time(time.Unix(1744156800, 0)),
					OrderID:            12345,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetVIPLoanAccruedInterest should decode every field")
		return
	}
	assert.NotNil(t, result, "GetVIPLoanAccruedInterest should return a result")
}

func TestGetVIPLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetVIPLoanOngoingOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetVIPLoanOngoingOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetVIPLoanOngoingOrders(t.Context(), &VIPLoanOngoingOrdersRequest{
		OrderID:             1232,
		CollateralAccountID: 21231,
		LoanCoin:            currency.BTC,
		CollateralCoin:      currency.ETH,
		Limit:               10,
	})
	require.NoError(t, err, "GetVIPLoanOngoingOrders must not error")
	if mockTests {
		exp := &VIPLoanOngoingOrdersResponse{
			Rows: []VIPLoanOngoingOrder{
				{
					OrderID:                          1232,
					LoanCoin:                         currency.BTC,
					TotalDebt:                        10000,
					LoanRate:                         0.0123,
					ResidualInterest:                 10.27687923,
					CollateralAccountID:              "21231,23456789",
					CollateralCoin:                   "ETH",
					TotalCollateralValueAfterHaircut: 25000.27565492,
					LockedCollateralValue:            25000.27565492,
					CurrentLTV:                       0.57,
					ExpirationTime:                   types.Time(time.Unix(1746748800, 0)),
					LoanDate:                         types.Time(time.Unix(1744156800, 0)),
					LoanTerm:                         "30days",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetVIPLoanOngoingOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetVIPLoanOngoingOrders should return a result")
}

func TestGetVIPLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetVIPLoanRepaymentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetVIPLoanRepaymentHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetVIPLoanRepaymentHistory(t.Context(), &VIPLoanHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetVIPLoanRepaymentHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetVIPLoanRepaymentHistory(t.Context(), &VIPLoanHistoryRequest{
		OrderID:   1234,
		LoanCoin:  currency.ETH,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     20,
	})
	require.NoError(t, err, "GetVIPLoanRepaymentHistory must not error")
	if mockTests {
		exp := &VIPLoanRepaymentHistoryResponse{
			Rows: []VIPLoanRepayment{
				{
					LoanCoin:       currency.ETH,
					RepayAmount:    10000,
					CollateralCoin: "BNB,BTC,ETH",
					RepayStatus:    "Repaid",
					LoanDate:       types.Time(time.Unix(1676851200, 0)),
					RepayTime:      types.Time(time.Unix(1744156800, 0)),
					OrderID:        "1234",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetVIPLoanRepaymentHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetVIPLoanRepaymentHistory should return a result")
}

func TestGetVIPApplicationStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetVIPApplicationStatus(t.Context(), 10, 20)
	require.NoError(t, err, "GetVIPApplicationStatus must not error")
	if mockTests {
		exp := &VIPLoanApplicationStatusResponse{
			Rows: []VIPLoanApplication{
				{
					LoanAccountID:       "12345678",
					OrderID:             "12345678",
					RequestID:           "12345678",
					LoanCoin:            currency.BTC,
					LoanAmount:          100.55,
					CollateralAccountID: "12345678,12345679",
					CollateralCoin:      "BUSD,USDT,ETH",
					LoanTerm:            "30",
					Status:              "Accruing_Interest",
					LoanDate:            types.Time(time.Unix(1676851200, 0)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetVIPApplicationStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetVIPApplicationStatus should return a result")
}
