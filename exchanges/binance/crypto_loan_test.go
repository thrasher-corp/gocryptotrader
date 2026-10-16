package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestCheckCollateralRepayRate(t *testing.T) {
	t.Parallel()
	_, err := e.CheckCollateralRepayRate(t.Context(), currency.EMPTYCODE, currency.USDT)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CheckCollateralRepayRate must reject an empty loan coin")
	_, err = e.CheckCollateralRepayRate(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CheckCollateralRepayRate must reject an empty collateral coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CheckCollateralRepayRate(t.Context(), currency.BTC, currency.USDT)
	require.NoError(t, err, "CheckCollateralRepayRate must not error")
	if mockTests {
		exp := &FlexibleLoanCollateralRepayRateResponse{
			LoanCoin:       currency.BTC,
			CollateralCoin: currency.USDT,
			Rate:           0.00001612,
		}
		assert.Equal(t, exp, result, "CheckCollateralRepayRate should decode every field")
		return
	}
	assert.Positive(t, result.Rate.Float64(), "CheckCollateralRepayRate should return a rate")
}

func TestFlexibleLoanAdjustLTV(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanAdjustLTV(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FlexibleLoanAdjustLTV must reject a nil request")
	_, err = e.FlexibleLoanAdjustLTV(t.Context(), &FlexibleLoanAdjustLTVRequest{CollateralCoin: currency.BTC, AdjustmentAmount: 1, Direction: "REDUCED"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FlexibleLoanAdjustLTV must reject an empty loan coin")
	_, err = e.FlexibleLoanAdjustLTV(t.Context(), &FlexibleLoanAdjustLTVRequest{LoanCoin: currency.USDT, AdjustmentAmount: 1, Direction: "REDUCED"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FlexibleLoanAdjustLTV must reject an empty collateral coin")
	_, err = e.FlexibleLoanAdjustLTV(t.Context(), &FlexibleLoanAdjustLTVRequest{LoanCoin: currency.USDT, CollateralCoin: currency.BTC, Direction: "REDUCED"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FlexibleLoanAdjustLTV must reject a zero amount")
	_, err = e.FlexibleLoanAdjustLTV(t.Context(), &FlexibleLoanAdjustLTVRequest{LoanCoin: currency.USDT, CollateralCoin: currency.BTC, AdjustmentAmount: 1})
	require.ErrorIs(t, err, errLTVAdjustmentDirectionRequired, "FlexibleLoanAdjustLTV must reject an empty direction")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FlexibleLoanAdjustLTV(t.Context(), &FlexibleLoanAdjustLTVRequest{
		LoanCoin:         currency.USDT,
		CollateralCoin:   currency.BTC,
		AdjustmentAmount: 1,
		Direction:        "REDUCED",
	})
	require.NoError(t, err, "FlexibleLoanAdjustLTV must not error")
	if mockTests {
		exp := &FlexibleLoanAdjustLTVResponse{
			LoanCoin:         currency.USDT,
			CollateralCoin:   currency.BTC,
			Direction:        "REDUCED",
			AdjustmentAmount: 1,
			CurrentLTV:       0.62,
			Status:           "Succeeds",
		}
		assert.Equal(t, exp, result, "FlexibleLoanAdjustLTV should decode every field")
		return
	}
	assert.NotEmpty(t, result.Status, "FlexibleLoanAdjustLTV should return a status")
}

func TestFlexibleLoanAdjustLTVResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := FlexibleLoanAdjustLTVResponse{
		LoanCoin:         currency.USDT,
		CollateralCoin:   currency.BTC,
		Direction:        "REDUCED",
		AdjustmentAmount: 1,
		CurrentLTV:       0.62,
		Status:           "Succeeds",
	}
	for _, tc := range []struct {
		name string
		data string
	}{
		{"documented adjustmentAmount", `{"loanCoin":"USDT","collateralCoin":"BTC","direction":"REDUCED","adjustmentAmount":"1","currentLTV":"0.62","status":"Succeeds"}`},
		{"v1 amount", `{"loanCoin":"USDT","collateralCoin":"BTC","direction":"REDUCED","amount":"1","currentLTV":"0.62","status":"Succeeds"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got FlexibleLoanAdjustLTVResponse
			require.NoError(t, json.Unmarshal([]byte(tc.data), &got), "Unmarshal must not error")
			assert.Equal(t, exp, got, "Unmarshal should decode every field")
		})
	}
	var got FlexibleLoanAdjustLTVResponse
	assert.Error(t, json.Unmarshal([]byte(`{"amount":true}`), &got), "Unmarshal should reject a non-numeric amount")
}

func TestFlexibleLoanBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanBorrow(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FlexibleLoanBorrow must reject a nil request")
	_, err = e.FlexibleLoanBorrow(t.Context(), &FlexibleLoanBorrowRequest{CollateralCoin: currency.USDC, LoanAmount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FlexibleLoanBorrow must reject an empty loan coin")
	_, err = e.FlexibleLoanBorrow(t.Context(), &FlexibleLoanBorrowRequest{LoanCoin: currency.ATOM, LoanAmount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FlexibleLoanBorrow must reject an empty collateral coin")
	_, err = e.FlexibleLoanBorrow(t.Context(), &FlexibleLoanBorrowRequest{LoanCoin: currency.ATOM, CollateralCoin: currency.USDC})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FlexibleLoanBorrow must reject no loan or collateral amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FlexibleLoanBorrow(t.Context(), &FlexibleLoanBorrowRequest{
		LoanCoin:       currency.ATOM,
		LoanAmount:     1,
		CollateralCoin: currency.USDC,
	})
	require.NoError(t, err, "FlexibleLoanBorrow must not error")
	if mockTests {
		exp := &FlexibleLoanBorrowResponse{
			LoanCoin:         currency.ATOM,
			LoanAmount:       1,
			CollateralCoin:   currency.USDC,
			CollateralAmount: 7.69,
			Status:           "Succeeds",
		}
		assert.Equal(t, exp, result, "FlexibleLoanBorrow should decode every field")
		return
	}
	assert.NotEmpty(t, result.Status, "FlexibleLoanBorrow should return a status")
}

func TestFlexibleLoanRepay(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanRepay(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FlexibleLoanRepay must reject a nil request")
	_, err = e.FlexibleLoanRepay(t.Context(), &FlexibleLoanRepayRequest{CollateralCoin: currency.USDC, RepayAmount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FlexibleLoanRepay must reject an empty loan coin")
	_, err = e.FlexibleLoanRepay(t.Context(), &FlexibleLoanRepayRequest{LoanCoin: currency.ATOM, RepayAmount: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FlexibleLoanRepay must reject an empty collateral coin")
	_, err = e.FlexibleLoanRepay(t.Context(), &FlexibleLoanRepayRequest{LoanCoin: currency.ATOM, CollateralCoin: currency.USDC, FullRepayment: true})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FlexibleLoanRepay must reject a zero repay amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *FlexibleLoanRepayRequest
		exp  *FlexibleLoanRepayResponse
	}{
		{
			name: "partial with loan asset",
			req:  &FlexibleLoanRepayRequest{LoanCoin: currency.ATOM, CollateralCoin: currency.USDC, RepayAmount: 1, CollateralReturn: new(false), RepaymentType: 1},
			exp: &FlexibleLoanRepayResponse{
				LoanCoin:            currency.ATOM,
				CollateralCoin:      currency.USDC,
				RemainingDebt:       0.5,
				RemainingCollateral: 7.69,
				CurrentLTV:          0.33,
				RepayStatus:         "REPAID",
			},
		},
		{
			name: "full with collateral",
			req:  &FlexibleLoanRepayRequest{LoanCoin: currency.ATOM, CollateralCoin: currency.USDC, RepayAmount: 1.5, CollateralReturn: new(true), FullRepayment: true, RepaymentType: 2},
			exp: &FlexibleLoanRepayResponse{
				LoanCoin:       currency.ATOM,
				CollateralCoin: currency.USDC,
				FullRepayment:  true,
				RepayStatus:    "REPAID",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.FlexibleLoanRepay(t.Context(), tc.req)
			require.NoError(t, err, "FlexibleLoanRepay must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "FlexibleLoanRepay should decode every field")
				return
			}
			assert.NotEmpty(t, result.RepayStatus, "FlexibleLoanRepay should return a repayment status")
		})
	}
}

func TestFlexibleLoanAssetsData(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FlexibleLoanAssetsData(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "FlexibleLoanAssetsData must not error")
	if mockTests {
		exp := &FlexibleLoanAssetsDataResponse{
			Rows: []FlexibleLoanAsset{
				{
					LoanCoin:             currency.ATOM,
					FlexibleInterestRate: 0.00000491,
					FlexibleMinimumLimit: 100,
					FlexibleMaximumLimit: 1000000,
				},
				{
					LoanCoin:             currency.USDT,
					FlexibleInterestRate: 0.00000512,
					FlexibleMinimumLimit: 10,
					FlexibleMaximumLimit: 5000000,
				},
			},
			Total: 2,
		}
		assert.Equal(t, exp, result, "FlexibleLoanAssetsData should decode every field")
		return
	}
	assert.NotEmpty(t, result.Rows, "FlexibleLoanAssetsData should return loanable assets")
}

func TestFlexibleLoanBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanBorrowHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FlexibleLoanBorrowHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.FlexibleLoanBorrowHistory(t.Context(), &FlexibleLoanHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "FlexibleLoanBorrowHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FlexibleLoanBorrowHistory(t.Context(), &FlexibleLoanHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Limit: 100})
	require.NoError(t, err, "FlexibleLoanBorrowHistory must not error")
	if mockTests {
		exp := &FlexibleLoanBorrowHistoryResponse{
			Rows: []FlexibleLoanBorrowRecord{
				{
					LoanCoin:                currency.ATOM,
					InitialLoanAmount:       1,
					CollateralCoin:          currency.USDC,
					InitialCollateralAmount: 7.69,
					BorrowTime:              types.Time(time.Unix(1744156800, 0)),
					Status:                  "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "FlexibleLoanBorrowHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "FlexibleLoanBorrowHistory should return a result")
}

func TestFlexibleCollateralAssetsData(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FlexibleCollateralAssetsData(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err, "FlexibleCollateralAssetsData must not error")
	if mockTests {
		exp := &FlexibleCollateralAssetsDataResponse{
			Rows: []FlexibleLoanCollateralAsset{
				{
					CollateralCoin: currency.USDC,
					InitialLTV:     0.65,
					MarginCallLTV:  0.75,
					LiquidationLTV: 0.83,
					MaximumLimit:   1000000,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "FlexibleCollateralAssetsData should decode every field")
		return
	}
	assert.NotEmpty(t, result.Rows, "FlexibleCollateralAssetsData should return collateral assets")
}

func TestGetFlexibleLoanLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleLoanLiquidationHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleLoanLiquidationHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetFlexibleLoanLiquidationHistory(t.Context(), &FlexibleLoanHistoryRequest{LoanCoin: currency.BTC, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFlexibleLoanLiquidationHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanLiquidationHistory(t.Context(), &FlexibleLoanHistoryRequest{LoanCoin: currency.BTC, StartTime: startTime, EndTime: endTime, Current: 1, Limit: 100})
	require.NoError(t, err, "GetFlexibleLoanLiquidationHistory must not error")
	if mockTests {
		exp := &FlexibleLoanLiquidationHistoryResponse{
			Rows: []FlexibleLoanLiquidation{
				{
					LoanCoin:                    currency.BTC,
					LiquidationDebt:             0.01,
					CollateralCoin:              currency.USDT,
					LiquidationCollateralAmount: 750,
					ReturnCollateralAmount:      0.2,
					LiquidationFee:              1.2,
					LiquidationStartingPrice:    75012.5,
					LiquidationStartingTime:     types.Time(time.Unix(1744156800, 0)),
					Status:                      "Liquidated",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetFlexibleLoanLiquidationHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFlexibleLoanLiquidationHistory should return a result")
}

func TestFlexibleLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanLTVAdjustmentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FlexibleLoanLTVAdjustmentHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.FlexibleLoanLTVAdjustmentHistory(t.Context(), &FlexibleLoanHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "FlexibleLoanLTVAdjustmentHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FlexibleLoanLTVAdjustmentHistory(t.Context(), &FlexibleLoanHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Limit: 100})
	require.NoError(t, err, "FlexibleLoanLTVAdjustmentHistory must not error")
	if mockTests {
		exp := &FlexibleLoanLTVAdjustmentHistoryResponse{
			Rows: []FlexibleLoanLTVAdjustment{
				{
					LoanCoin:         currency.USDT,
					CollateralCoin:   currency.BTC,
					Direction:        "ADDITIONAL",
					CollateralAmount: 0.005,
					PreLTV:           0.78,
					AfterLTV:         0.56,
					AdjustTime:       types.Time(time.Unix(1744156800, 0)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "FlexibleLoanLTVAdjustmentHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "FlexibleLoanLTVAdjustmentHistory should return a result")
}

func TestFlexibleLoanOngoingOrders(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanOngoingOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FlexibleLoanOngoingOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FlexibleLoanOngoingOrders(t.Context(), &FlexibleLoanOngoingOrdersRequest{})
	require.NoError(t, err, "FlexibleLoanOngoingOrders must not error")
	if mockTests {
		exp := &FlexibleLoanOngoingOrdersResponse{
			Rows: []FlexibleLoanOngoingOrder{
				{
					LoanCoin:         currency.ATOM,
					TotalDebt:        1.0001,
					CollateralCoin:   currency.USDC,
					CollateralAmount: 7.69,
					CurrentLTV:       0.57,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "FlexibleLoanOngoingOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "FlexibleLoanOngoingOrders should return a result")
}

func TestFlexibleLoanRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FlexibleLoanRepayHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FlexibleLoanRepayHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.FlexibleLoanRepayHistory(t.Context(), &FlexibleLoanHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "FlexibleLoanRepayHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FlexibleLoanRepayHistory(t.Context(), &FlexibleLoanHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Limit: 100})
	require.NoError(t, err, "FlexibleLoanRepayHistory must not error")
	if mockTests {
		exp := &FlexibleLoanRepayHistoryResponse{
			Rows: []FlexibleLoanRepayment{
				{
					LoanCoin:         currency.ATOM,
					RepayAmount:      0.5,
					CollateralCoin:   currency.USDC,
					CollateralReturn: 3.84,
					RepayStatus:      "REPAID",
					RepayTime:        types.Time(time.Unix(1744156800, 0)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "FlexibleLoanRepayHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "FlexibleLoanRepayHistory should return a result")
}

func TestCryptoLoanIncomeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.CryptoLoanIncomeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CryptoLoanIncomeHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.CryptoLoanIncomeHistory(t.Context(), &CryptoLoanIncomeHistoryRequest{Asset: currency.USDT, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "CryptoLoanIncomeHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CryptoLoanIncomeHistory(t.Context(), &CryptoLoanIncomeHistoryRequest{Asset: currency.USDT, StartTime: startTime, EndTime: endTime, Limit: 100})
	require.NoError(t, err, "CryptoLoanIncomeHistory must not error")
	if mockTests {
		exp := []CryptoLoanIncome{
			{
				Asset:         currency.USDT,
				Type:          "borrowIn",
				Amount:        100,
				Timestamp:     types.Time(time.Unix(1744156800, 0)),
				TransactionID: "80423589583",
			},
		}
		assert.Equal(t, exp, result, "CryptoLoanIncomeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "CryptoLoanIncomeHistory should return a result")
}

func TestCryptoLoanBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := e.CryptoLoanBorrowHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CryptoLoanBorrowHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.CryptoLoanBorrowHistory(t.Context(), &CryptoLoanHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "CryptoLoanBorrowHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CryptoLoanBorrowHistory(t.Context(), &CryptoLoanHistoryRequest{
		LoanCoin:       currency.USDT,
		CollateralCoin: currency.BTC,
		StartTime:      startTime,
		EndTime:        endTime,
		Current:        1,
		Limit:          100,
	})
	require.NoError(t, err, "CryptoLoanBorrowHistory must not error")
	if mockTests {
		exp := &CryptoLoanBorrowHistoryResponse{
			Rows: []CryptoLoanBorrowRecord{
				{
					OrderID:                 100000001,
					LoanCoin:                currency.USDT,
					InitialLoanAmount:       10000,
					HourlyInterestRate:      0.000057,
					LoanTerm:                7,
					CollateralCoin:          currency.BTC,
					InitialCollateralAmount: 0.2,
					BorrowTime:              types.Time(time.Unix(1744156800, 0)),
					Status:                  "Repaid",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "CryptoLoanBorrowHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "CryptoLoanBorrowHistory should return a result")
}

func TestCryptoLoanLTVAdjustmentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.CryptoLoanLTVAdjustmentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CryptoLoanLTVAdjustmentHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.CryptoLoanLTVAdjustmentHistory(t.Context(), &CryptoLoanHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "CryptoLoanLTVAdjustmentHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CryptoLoanLTVAdjustmentHistory(t.Context(), &CryptoLoanHistoryRequest{
		LoanCoin:       currency.USDT,
		CollateralCoin: currency.BTC,
		StartTime:      startTime,
		EndTime:        endTime,
		Current:        1,
		Limit:          100,
	})
	require.NoError(t, err, "CryptoLoanLTVAdjustmentHistory must not error")
	if mockTests {
		exp := &CryptoLoanLTVAdjustmentHistoryResponse{
			Rows: []CryptoLoanLTVAdjustment{
				{
					LoanCoin:       currency.USDT,
					CollateralCoin: currency.BTC,
					Direction:      "ADDITIONAL",
					Amount:         0.01,
					PreLTV:         0.78,
					AfterLTV:       0.56,
					AdjustTime:     types.Time(time.Unix(1744156800, 0)),
					OrderID:        756783308056935434,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "CryptoLoanLTVAdjustmentHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "CryptoLoanLTVAdjustmentHistory should return a result")
}

func TestCryptoLoanRepaymentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.CryptoLoanRepaymentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CryptoLoanRepaymentHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.CryptoLoanRepaymentHistory(t.Context(), &CryptoLoanHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "CryptoLoanRepaymentHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CryptoLoanRepaymentHistory(t.Context(), &CryptoLoanHistoryRequest{
		LoanCoin:       currency.USDT,
		CollateralCoin: currency.BTC,
		StartTime:      startTime,
		EndTime:        endTime,
		Current:        1,
		Limit:          100,
	})
	require.NoError(t, err, "CryptoLoanRepaymentHistory must not error")
	if mockTests {
		exp := &CryptoLoanRepaymentHistoryResponse{
			Rows: []CryptoLoanRepayment{
				{
					LoanCoin:         currency.USDT,
					RepayAmount:      10000,
					CollateralCoin:   currency.BTC,
					CollateralUsed:   0.0123,
					CollateralReturn: 0.1877,
					RepayType:        "2",
					RepayStatus:      "Repaid",
					RepayTime:        types.Time(time.Unix(1744156800, 0)),
					OrderID:          756783308056935434,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "CryptoLoanRepaymentHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "CryptoLoanRepaymentHistory should return a result")
}
