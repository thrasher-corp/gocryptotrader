package bybit

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

func TestGetLegacyCryptoLoanCollateralCoins(t *testing.T) {
	t.Parallel()
	result, err := e.GetLegacyCryptoLoanCollateralCoins(t.Context(), "PRO1", currency.ETH)
	require.NoError(t, err, "GetLegacyCryptoLoanCollateralCoins must not error")
	if mockTests {
		exp := &LegacyCryptoLoanCollateralCoinsResponse{
			VIPCoinList: []LegacyCryptoLoanVIPCollateralCoins{
				{
					List: []LegacyCryptoLoanCollateralCoin{
						{
							CollateralAccuracy: 8,
							Currency:           currency.ETH,
							InitialLTV:         0.8,
							MarginCallLTV:      0.87,
							LiquidationLTV:     0.95,
							MaxLimit:           32000,
						},
					},
					VIPLevel: "PRO1",
				},
			},
		}
		assert.Equal(t, exp, result, "GetLegacyCryptoLoanCollateralCoins should decode every field")
		return
	}
	require.NotEmpty(t, result.VIPCoinList, "GetLegacyCryptoLoanCollateralCoins must return VIP levels")
	assert.NotEmpty(t, result.VIPCoinList[0].List, "GetLegacyCryptoLoanCollateralCoins should return collateral coins")
}

func TestGetLegacyCryptoLoanBorrowableCoins(t *testing.T) {
	t.Parallel()
	result, err := e.GetLegacyCryptoLoanBorrowableCoins(t.Context(), "VIP0", currency.USDT)
	require.NoError(t, err, "GetLegacyCryptoLoanBorrowableCoins must not error")
	if mockTests {
		exp := &LegacyCryptoLoanBorrowableCoinsResponse{
			VIPCoinList: []LegacyCryptoLoanVIPBorrowableCoins{
				{
					List: []LegacyCryptoLoanBorrowableCoin{
						{
							BorrowingAccuracy:          4,
							Currency:                   currency.USDT,
							FlexibleHourlyInterestRate: 0.0000049141,
							HourlyInterestRate7Day:     0.0000180692,
							HourlyInterestRate14Day:    0.0000207796,
							HourlyInterestRate30Day:    0.00002349,
							HourlyInterestRate90Day:    0.0000251,
							HourlyInterestRate180Day:   0.0000273,
							MaxBorrowingAmount:         16000000,
							MinBorrowingAmount:         20,
						},
					},
					VIPLevel: "VIP0",
				},
			},
		}
		assert.Equal(t, exp, result, "GetLegacyCryptoLoanBorrowableCoins should decode every field")
		return
	}
	require.NotEmpty(t, result.VIPCoinList, "GetLegacyCryptoLoanBorrowableCoins must return VIP levels")
	assert.NotEmpty(t, result.VIPCoinList[0].List, "GetLegacyCryptoLoanBorrowableCoins should return borrowable coins")
}

func TestGetLegacyCryptoLoanBorrowableCollateralisableLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetLegacyCryptoLoanBorrowableCollateralisableLimit(t.Context(), currency.EMPTYCODE, currency.BTC)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLegacyCryptoLoanBorrowableCollateralisableLimit must reject an empty loan currency")
	_, err = e.GetLegacyCryptoLoanBorrowableCollateralisableLimit(t.Context(), currency.USDT, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLegacyCryptoLoanBorrowableCollateralisableLimit must reject an empty collateral currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLegacyCryptoLoanBorrowableCollateralisableLimit(t.Context(), currency.USDT, currency.BTC)
	require.NoError(t, err, "GetLegacyCryptoLoanBorrowableCollateralisableLimit must not error")
	if mockTests {
		exp := &LegacyCryptoLoanBorrowableCollateralisableLimitResponse{
			CollateralCurrency:  currency.BTC,
			LoanCurrency:        currency.USDT,
			MaxCollateralAmount: 164.95773205552675,
			MaxLoanAmount:       8000000,
			MinCollateralAmount: 0.000412394330138818,
			MinLoanAmount:       20,
		}
		assert.Equal(t, exp, result, "GetLegacyCryptoLoanBorrowableCollateralisableLimit should decode every field")
		return
	}
	assert.Equal(t, currency.USDT, result.LoanCurrency, "GetLegacyCryptoLoanBorrowableCollateralisableLimit should return the loan currency")
}

func TestRepayLegacyCryptoLoan(t *testing.T) {
	t.Parallel()
	_, err := e.RepayLegacyCryptoLoan(t.Context(), "", 100)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "RepayLegacyCryptoLoan must reject an empty order ID")
	_, err = e.RepayLegacyCryptoLoan(t.Context(), "1794267532472646144", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RepayLegacyCryptoLoan must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RepayLegacyCryptoLoan(t.Context(), "1794267532472646144", 100)
	require.NoError(t, err, "RepayLegacyCryptoLoan must not error")
	if mockTests {
		exp := &CryptoLoanRepayResponse{
			RepayID: "1794271131730737664",
		}
		assert.Equal(t, exp, result, "RepayLegacyCryptoLoan should decode every field")
		return
	}
	assert.NotEmpty(t, result.RepayID, "RepayLegacyCryptoLoan should return a repayment ID")
}

func TestGetLegacyCryptoLoanUnpaidLoans(t *testing.T) {
	t.Parallel()
	_, err := e.GetLegacyCryptoLoanUnpaidLoans(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLegacyCryptoLoanUnpaidLoans must reject a nil request")
	_, err = e.GetLegacyCryptoLoanUnpaidLoans(t.Context(), &LegacyCryptoLoanUnpaidLoansRequest{LoanTermType: "1"})
	require.ErrorIs(t, err, errLoanTermRequired, "GetLegacyCryptoLoanUnpaidLoans must reject a fixed term query without a loan term")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLegacyCryptoLoanUnpaidLoans(t.Context(), &LegacyCryptoLoanUnpaidLoansRequest{
		OrderID:            "1793683005081680384",
		LoanCurrency:       currency.USDT,
		CollateralCurrency: currency.BTC,
		LoanTermType:       "1",
		LoanTerm:           30,
		Limit:              10,
		Cursor:             "1793683005081680385",
	})
	require.NoError(t, err, "GetLegacyCryptoLoanUnpaidLoans must not error")
	if mockTests {
		exp := &LegacyCryptoLoanUnpaidLoansResponse{
			List: []LegacyCryptoLoanUnpaidLoan{
				{
					CollateralAmount:        0.0964687,
					CollateralCurrency:      currency.BTC,
					CurrentLTV:              0.4161,
					ExpirationTime:          types.Time(time.Unix(1731149999, 0)),
					HourlyInterestRate:      0.0000010633,
					LoanCurrency:            currency.USDT,
					LoanTerm:                30,
					OrderID:                 "1793683005081680384",
					ResidualInterest:        0.04016,
					ResidualPenaltyInterest: 0.0012,
					TotalDebt:               1888.005198,
				},
			},
			NextPageCursor: "1793683005081680383",
		}
		assert.Equal(t, exp, result, "GetLegacyCryptoLoanUnpaidLoans should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLegacyCryptoLoanUnpaidLoans should return a result")
}

func TestGetLegacyCryptoLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLegacyCryptoLoanRepaymentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLegacyCryptoLoanRepaymentHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLegacyCryptoLoanRepaymentHistory(t.Context(), &LegacyCryptoLoanRepaymentHistoryRequest{
		OrderID:      "1794267532472646144",
		RepayID:      "1794271131730737664",
		LoanCurrency: currency.USDT,
		Limit:        10,
		Cursor:       "1794271131730737666",
	})
	require.NoError(t, err, "GetLegacyCryptoLoanRepaymentHistory must not error")
	if mockTests {
		exp := &LegacyCryptoLoanRepaymentHistoryResponse{
			List: []LegacyCryptoLoanRepayment{
				{
					CollateralCurrency: currency.BTC,
					CollateralReturn:   0.0494727,
					LoanCurrency:       currency.USDT,
					LoanTerm:           7,
					OrderID:            "1794267532472646144",
					RepayAmount:        1888.005198,
					RepayID:            "1794271131730737664",
					RepayStatus:        1,
					RepayTime:          types.Time(time.UnixMilli(1728629786875)),
					RepayType:          "1",
				},
			},
			NextPageCursor: "1794271131730737663",
		}
		assert.Equal(t, exp, result, "GetLegacyCryptoLoanRepaymentHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLegacyCryptoLoanRepaymentHistory should return a result")
}

func TestGetLegacyCryptoLoanCompletedLoanHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLegacyCryptoLoanCompletedLoanHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLegacyCryptoLoanCompletedLoanHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLegacyCryptoLoanCompletedLoanHistory(t.Context(), &LegacyCryptoLoanCompletedLoanHistoryRequest{
		OrderID:            "1793569729874260992",
		LoanCurrency:       currency.ETH,
		CollateralCurrency: currency.BTC,
		Limit:              10,
		Cursor:             "1793569729874260993",
	})
	require.NoError(t, err, "GetLegacyCryptoLoanCompletedLoanHistory must not error")
	if mockTests {
		exp := &LegacyCryptoLoanCompletedLoanHistoryResponse{
			List: []LegacyCryptoLoanCompletedLoan{
				{
					BorrowTime:              types.Time(time.UnixMilli(1728546174028)),
					CollateralCurrency:      currency.BTC,
					ExpirationTime:          types.Time(time.Unix(1729148399, 0)),
					HourlyInterestRate:      0.0000010241,
					InitialCollateralAmount: 0.0494727,
					InitialLoanAmount:       1,
					LoanCurrency:            currency.ETH,
					LoanTerm:                7,
					OrderID:                 "1793569729874260992",
					RepaidInterest:          0.00000515,
					RepaidPenaltyInterest:   0.0000012,
					Status:                  1,
				},
			},
			NextPageCursor: "1793569729874260991",
		}
		assert.Equal(t, exp, result, "GetLegacyCryptoLoanCompletedLoanHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLegacyCryptoLoanCompletedLoanHistory should return a result")
}

func TestGetLegacyCryptoLoanMaxCollateralReductionAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetLegacyCryptoLoanMaxCollateralReductionAmount(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetLegacyCryptoLoanMaxCollateralReductionAmount must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLegacyCryptoLoanMaxCollateralReductionAmount(t.Context(), "1794267532472646144")
	require.NoError(t, err, "GetLegacyCryptoLoanMaxCollateralReductionAmount must not error")
	if mockTests {
		exp := &CryptoLoanMaxCollateralReductionAmountResponse{
			MaxCollateralAmount: 0.00210611,
		}
		assert.Equal(t, exp, result, "GetLegacyCryptoLoanMaxCollateralReductionAmount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLegacyCryptoLoanMaxCollateralReductionAmount should return a result")
}

func TestAdjustLegacyCryptoLoanCollateralAmount(t *testing.T) {
	t.Parallel()
	_, err := e.AdjustLegacyCryptoLoanCollateralAmount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AdjustLegacyCryptoLoanCollateralAmount must reject a nil request")
	_, err = e.AdjustLegacyCryptoLoanCollateralAmount(t.Context(), &AdjustLegacyCryptoLoanCollateralAmountRequest{Amount: 0.001, Direction: "1"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AdjustLegacyCryptoLoanCollateralAmount must reject an empty order ID")
	_, err = e.AdjustLegacyCryptoLoanCollateralAmount(t.Context(), &AdjustLegacyCryptoLoanCollateralAmountRequest{OrderID: "1794267532472646144", Direction: "1"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "AdjustLegacyCryptoLoanCollateralAmount must reject a zero amount")
	_, err = e.AdjustLegacyCryptoLoanCollateralAmount(t.Context(), &AdjustLegacyCryptoLoanCollateralAmountRequest{OrderID: "1794267532472646144", Amount: 0.001})
	require.ErrorIs(t, err, errCollateralAdjustmentDirectionRequired, "AdjustLegacyCryptoLoanCollateralAmount must reject an empty direction")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AdjustLegacyCryptoLoanCollateralAmount(t.Context(), &AdjustLegacyCryptoLoanCollateralAmountRequest{
		OrderID:   "1794267532472646144",
		Amount:    0.001,
		Direction: "1",
	})
	require.NoError(t, err, "AdjustLegacyCryptoLoanCollateralAmount must not error")
	if mockTests {
		exp := &AdjustLegacyCryptoLoanCollateralAmountResponse{
			AdjustID: "1794318409405331968",
		}
		assert.Equal(t, exp, result, "AdjustLegacyCryptoLoanCollateralAmount should decode every field")
		return
	}
	assert.NotEmpty(t, result.AdjustID, "AdjustLegacyCryptoLoanCollateralAmount should return an adjustment ID")
}

func TestGetLegacyCryptoLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLegacyCryptoLoanLTVAdjustmentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLegacyCryptoLoanLTVAdjustmentHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLegacyCryptoLoanLTVAdjustmentHistory(t.Context(), &LegacyCryptoLoanLTVAdjustmentHistoryRequest{
		OrderID:            "1794267532472646144",
		AdjustID:           "1794318409405331968",
		CollateralCurrency: currency.BTC,
		Limit:              10,
		Cursor:             "1844656778923966467",
	})
	require.NoError(t, err, "GetLegacyCryptoLoanLTVAdjustmentHistory must not error")
	if mockTests {
		exp := &LegacyCryptoLoanLTVAdjustmentHistoryResponse{
			List: []LegacyCryptoLoanLTVAdjustment{
				{
					CollateralCurrency: currency.BTC,
					OrderID:            "1794267532472646144",
					AdjustID:           "1794318409405331968",
					AdjustTime:         types.Time(time.UnixMilli(1728635422814)),
					PreLTV:             0.6546,
					AfterLTV:           0.7164,
					Direction:          1,
					Amount:             0.001,
				},
			},
			NextPageCursor: "1844656778923966466",
		}
		assert.Equal(t, exp, result, "GetLegacyCryptoLoanLTVAdjustmentHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLegacyCryptoLoanLTVAdjustmentHistory should return a result")
}
