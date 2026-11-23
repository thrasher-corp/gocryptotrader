package okx

import (
	"strings"
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

func TestGetFlexibleLoanBorrowableCurrencies(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanBorrowableCurrencies(contextGenerate())
	require.NoError(t, err, "GetFlexibleLoanBorrowableCurrencies must not error")
	if mockTests {
		exp := []FlexibleLoanBorrowableCurrencyResponse{{BorrowCurrency: currency.USDT}, {BorrowCurrency: currency.USDC}}
		assert.Equal(t, exp, result, "GetFlexibleLoanBorrowableCurrencies should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetFlexibleLoanBorrowableCurrencies should return currencies")
	}
}

func TestGetFlexibleLoanCollateralAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleLoanCollateralAssets(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleLoanCollateralAssets must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanCollateralAssets(contextGenerate(), &FlexibleLoanCollateralAssetsRequest{Currency: currency.BTC, OrderID: "12345"})
	require.NoError(t, err, "GetFlexibleLoanCollateralAssets must not error")
	if mockTests {
		exp := []FlexibleLoanCollateralAssetsResponse{
			{Assets: []FlexibleLoanCollateralAsset{{Currency: currency.BTC, Amount: 1.7921483143067599, NotionalUSD: 158292.621793314105231}}},
		}
		assert.Equal(t, exp, result, "GetFlexibleLoanCollateralAssets should decode every field")
	} else {
		assert.NotNil(t, result, "GetFlexibleLoanCollateralAssets should return collateral")
	}
}

func TestGetFlexibleLoanMaximumLoanAmount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *FlexibleLoanMaximumLoanRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing borrow currency", arg: &FlexibleLoanMaximumLoanRequest{OrderID: "12345"}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing supplementary collateral currency", arg: &FlexibleLoanMaximumLoanRequest{BorrowCurrency: currency.USDT, SupplementaryCollateral: []FlexibleLoanCurrencyAmount{{Amount: 1}}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing supplementary collateral amount", arg: &FlexibleLoanMaximumLoanRequest{BorrowCurrency: currency.USDT, SupplementaryCollateral: []FlexibleLoanCurrencyAmount{{Currency: currency.BTC}}}, err: limits.ErrAmountBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetFlexibleLoanMaximumLoanAmount(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetFlexibleLoanMaximumLoanAmount should reject an invalid request")
		})
	}
	for _, tc := range []struct {
		name string
		arg  *FlexibleLoanMaximumLoanRequest
		body string
	}{
		{
			name: "every optional parameter",
			arg:  &FlexibleLoanMaximumLoanRequest{BorrowCurrency: currency.USDT, OrderID: "12345", SupplementaryCollateral: []FlexibleLoanCurrencyAmount{{Currency: currency.BTC, Amount: 0.5}, {Currency: currency.ETH, Amount: 2}}},
			body: `{"borrowCcy":"USDT","ordId":"12345","supCollateral":[{"ccy":"BTC","amt":"0.5"},{"ccy":"ETH","amt":"2"}]}`,
		},
		{name: "required parameters only", arg: &FlexibleLoanMaximumLoanRequest{BorrowCurrency: currency.USDT}, body: `{"borrowCcy":"USDT"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newRequestPinExchange(t, "/finance/flexible-loan/max-loan", tc.body, []map[string]string{{"borrowCcy": "USDT", "maxLoan": "1"}})
			_, err := ex.GetFlexibleLoanMaximumLoanAmount(t.Context(), tc.arg)
			require.NoError(t, err, "GetFlexibleLoanMaximumLoanAmount must not error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanMaximumLoanAmount(contextGenerate(), &FlexibleLoanMaximumLoanRequest{BorrowCurrency: currency.USDT, OrderID: "12345"})
	require.NoError(t, err, "GetFlexibleLoanMaximumLoanAmount must not error")
	if mockTests {
		exp := []FlexibleLoanMaximumLoanResponse{{BorrowCurrency: currency.USDT, MaximumLoan: 0.01113, NotionalUSD: 0.01113356, RemainingQuota: 3395000}}
		assert.Equal(t, exp, result, "GetFlexibleLoanMaximumLoanAmount should decode every field")
	} else {
		assert.NotNil(t, result, "GetFlexibleLoanMaximumLoanAmount should return the maximum loan")
	}
}

func TestGetFlexibleLoanMaximumCollateralRedeemAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleLoanMaximumCollateralRedeemAmount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleLoanMaximumCollateralRedeemAmount must reject a nil request")
	_, err = e.GetFlexibleLoanMaximumCollateralRedeemAmount(t.Context(), &FlexibleLoanMaximumCollateralRedeemRequest{OrderID: "12345"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFlexibleLoanMaximumCollateralRedeemAmount must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanMaximumCollateralRedeemAmount(contextGenerate(), &FlexibleLoanMaximumCollateralRedeemRequest{Currency: currency.USDT, OrderID: "12345"})
	require.NoError(t, err, "GetFlexibleLoanMaximumCollateralRedeemAmount must not error")
	if mockTests {
		exp := []FlexibleLoanMaximumCollateralRedeemResponse{{Currency: currency.USDT, MaximumRedeemAmount: 1}}
		assert.Equal(t, exp, result, "GetFlexibleLoanMaximumCollateralRedeemAmount should decode every field")
	} else {
		assert.NotNil(t, result, "GetFlexibleLoanMaximumCollateralRedeemAmount should return the maximum")
	}
}

func TestAdjustFlexibleLoanCollateral(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *FlexibleLoanAdjustCollateralRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing type", arg: &FlexibleLoanAdjustCollateralRequest{CollateralCurrency: currency.BTC, CollateralAmount: 0.1}, err: errMissingRequiredParameter},
		{name: "missing collateral currency", arg: &FlexibleLoanAdjustCollateralRequest{Type: "add", CollateralAmount: 0.1}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing collateral amount", arg: &FlexibleLoanAdjustCollateralRequest{Type: "add", CollateralCurrency: currency.BTC}, err: limits.ErrAmountBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := e.AdjustFlexibleLoanCollateral(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "AdjustFlexibleLoanCollateral should reject an invalid request")
		})
	}
	for _, tc := range []struct {
		name string
		arg  *FlexibleLoanAdjustCollateralRequest
		body string
	}{
		{
			name: "every optional parameter",
			arg:  &FlexibleLoanAdjustCollateralRequest{Type: "add", CollateralCurrency: currency.BTC, CollateralAmount: 0.1, OrderID: "12345"},
			body: `{"type":"add","collateralCcy":"BTC","collateralAmt":"0.1","ordId":"12345"}`,
		},
		{
			name: "required parameters only",
			arg:  &FlexibleLoanAdjustCollateralRequest{Type: "reduce", CollateralCurrency: currency.USDT, CollateralAmount: 1},
			body: `{"type":"reduce","collateralCcy":"USDT","collateralAmt":"1"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newRequestPinExchange(t, "/finance/flexible-loan/adjust-collateral", tc.body, []any{})
			require.NoError(t, ex.AdjustFlexibleLoanCollateral(t.Context(), tc.arg), "AdjustFlexibleLoanCollateral must not error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err := e.AdjustFlexibleLoanCollateral(contextGenerate(), &FlexibleLoanAdjustCollateralRequest{Type: "add", CollateralCurrency: currency.BTC, CollateralAmount: 0.1, OrderID: "12345"})
	require.NoError(t, err, "AdjustFlexibleLoanCollateral must not error")
}

func TestBorrowFlexibleLoan(t *testing.T) {
	t.Parallel()
	tooManyCollaterals := make([]FlexibleLoanCurrencyAmount, maxFlexibleLoanBorrowCollateralEntries+1)
	for i := range tooManyCollaterals {
		tooManyCollaterals[i] = FlexibleLoanCurrencyAmount{Currency: currency.BTC, Amount: 0.01}
	}
	loan := FlexibleLoanCurrencyAmount{Currency: currency.USDT, Amount: 1000}
	for _, tc := range []struct {
		name string
		arg  *FlexibleLoanBorrowRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing loan currency", arg: &FlexibleLoanBorrowRequest{LoanData: FlexibleLoanCurrencyAmount{Amount: 1000}, ClientOrderID: "my-borrow-001"}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing loan amount", arg: &FlexibleLoanBorrowRequest{LoanData: FlexibleLoanCurrencyAmount{Currency: currency.USDT}, ClientOrderID: "my-borrow-001"}, err: limits.ErrAmountBelowMin},
		{name: "new loan without collateral", arg: &FlexibleLoanBorrowRequest{LoanData: loan, ClientOrderID: "my-borrow-001"}, err: errMissingRequiredParameter},
		{name: "more collateral entries than the maximum", arg: &FlexibleLoanBorrowRequest{LoanData: loan, CollateralData: tooManyCollaterals, ClientOrderID: "my-borrow-001"}, err: errFlexibleLoanCollateralEntriesExceedMaximum},
		{name: "missing collateral currency", arg: &FlexibleLoanBorrowRequest{LoanData: loan, CollateralData: []FlexibleLoanCurrencyAmount{{Amount: 0.02}}, ClientOrderID: "my-borrow-001"}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing collateral amount", arg: &FlexibleLoanBorrowRequest{LoanData: loan, CollateralData: []FlexibleLoanCurrencyAmount{{Currency: currency.BTC}}, ClientOrderID: "my-borrow-001"}, err: limits.ErrAmountBelowMin},
		{name: "missing client order ID", arg: &FlexibleLoanBorrowRequest{LoanData: loan, CollateralData: []FlexibleLoanCurrencyAmount{{Currency: currency.BTC, Amount: 0.02}}}, err: order.ErrClientOrderIDMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.BorrowFlexibleLoan(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "BorrowFlexibleLoan should reject an invalid request")
		})
	}
	maximumCollateralsBody := `{"loanData":{"ccy":"USDT","amt":"1000"},"collateralData":[` + strings.Repeat(`{"ccy":"BTC","amt":"0.01"},`, maxFlexibleLoanBorrowCollateralEntries-1) + `{"ccy":"BTC","amt":"0.01"}],"clOrdId":"my-borrow-004"}`
	for _, tc := range []struct {
		name string
		arg  *FlexibleLoanBorrowRequest
		body string
	}{
		{
			name: "every optional parameter",
			arg: &FlexibleLoanBorrowRequest{
				OrderID:        "680234819012345678",
				LoanData:       FlexibleLoanCurrencyAmount{Currency: currency.USDT, Amount: 500},
				CollateralData: []FlexibleLoanCurrencyAmount{{Currency: currency.BTC, Amount: 0.01}, {Currency: currency.ETH, Amount: 0.5}},
				EMode:          "1",
				ClientOrderID:  "my-borrow-002",
			},
			body: `{"ordId":"680234819012345678","loanData":{"ccy":"USDT","amt":"500"},"collateralData":[{"ccy":"BTC","amt":"0.01"},{"ccy":"ETH","amt":"0.5"}],"eMode":"1","clOrdId":"my-borrow-002"}`,
		},
		{
			name: "borrow more on an existing loan",
			arg:  &FlexibleLoanBorrowRequest{OrderID: "680234819012345678", LoanData: FlexibleLoanCurrencyAmount{Currency: currency.USDT, Amount: 500}, ClientOrderID: "my-borrow-003"},
			body: `{"ordId":"680234819012345678","loanData":{"ccy":"USDT","amt":"500"},"clOrdId":"my-borrow-003"}`,
		},
		{
			name: "collateral entries at the maximum",
			arg:  &FlexibleLoanBorrowRequest{LoanData: loan, CollateralData: tooManyCollaterals[:maxFlexibleLoanBorrowCollateralEntries], ClientOrderID: "my-borrow-004"},
			body: maximumCollateralsBody,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newRequestPinExchange(t, "/finance/flexible-loan/borrow", tc.body, []map[string]string{{"ordId": "680234819012345678", "eMode": "0", "clOrdId": tc.arg.ClientOrderID}})
			_, err := ex.BorrowFlexibleLoan(t.Context(), tc.arg)
			require.NoError(t, err, "BorrowFlexibleLoan must not error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BorrowFlexibleLoan(contextGenerate(), &FlexibleLoanBorrowRequest{
		LoanData:       FlexibleLoanCurrencyAmount{Currency: currency.USDT, Amount: 1000},
		CollateralData: []FlexibleLoanCurrencyAmount{{Currency: currency.BTC, Amount: 0.02}},
		EMode:          "0",
		ClientOrderID:  "my-borrow-001",
	})
	require.NoError(t, err, "BorrowFlexibleLoan must not error")
	if mockTests {
		exp := &FlexibleLoanBorrowResponse{OrderID: "680234819012345678", EMode: "0", ClientOrderID: "my-borrow-001"}
		assert.Equal(t, exp, result, "BorrowFlexibleLoan should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "BorrowFlexibleLoan should return an order ID")
	}
}

func TestRepayFlexibleLoan(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *FlexibleLoanRepayRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing order ID", arg: &FlexibleLoanRepayRequest{Currency: currency.USDT, Amount: 1000, ClientOrderID: "my-repay-001"}, err: order.ErrOrderIDNotSet},
		{name: "missing currency", arg: &FlexibleLoanRepayRequest{OrderID: "680234819012345678", Amount: 1000, ClientOrderID: "my-repay-001"}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing amount", arg: &FlexibleLoanRepayRequest{OrderID: "680234819012345678", Currency: currency.USDT, ClientOrderID: "my-repay-001"}, err: limits.ErrAmountBelowMin},
		{name: "missing client order ID", arg: &FlexibleLoanRepayRequest{OrderID: "680234819012345678", Currency: currency.USDT, Amount: 1000}, err: order.ErrClientOrderIDMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.RepayFlexibleLoan(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "RepayFlexibleLoan should reject an invalid request")
		})
	}
	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/finance/flexible-loan/repay", `{"ordId":"680234819012345678","ccy":"USDT","amt":"250.5","clOrdId":"my-repay-002"}`, []map[string]string{{"ordId": "680234819012345678"}})
		_, err := ex.RepayFlexibleLoan(t.Context(), &FlexibleLoanRepayRequest{OrderID: "680234819012345678", Currency: currency.USDT, Amount: 250.5, ClientOrderID: "my-repay-002"})
		require.NoError(t, err, "RepayFlexibleLoan must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RepayFlexibleLoan(contextGenerate(), &FlexibleLoanRepayRequest{OrderID: "680234819012345678", Currency: currency.USDT, Amount: 1000, ClientOrderID: "my-repay-001"})
	require.NoError(t, err, "RepayFlexibleLoan must not error")
	if mockTests {
		exp := &FlexibleLoanRepayResponse{OrderID: "680234819012345678", Currency: currency.USDT, Amount: 1000, ClientOrderID: "my-repay-001"}
		assert.Equal(t, exp, result, "RepayFlexibleLoan should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "RepayFlexibleLoan should return the order ID")
	}
}

func TestGetFlexibleLoanEModeInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetFlexibleLoanEModeInfo(contextGenerate())
	require.NoError(t, err, "GetFlexibleLoanEModeInfo must not error")
	if mockTests {
		exp := []FlexibleLoanEModeInfoResponse{
			{CollateralCurrency: currency.BETH, BorrowCurrency: currency.ETH},
			{CollateralCurrency: currency.NewCode("OKSOL"), BorrowCurrency: currency.SOL},
		}
		assert.Equal(t, exp, result, "GetFlexibleLoanEModeInfo should decode every field")
	} else {
		require.NotEmpty(t, result, "GetFlexibleLoanEModeInfo must return currency pairs")
		assert.False(t, result[0].CollateralCurrency.IsEmpty(), "GetFlexibleLoanEModeInfo should return a collateral currency")
		assert.False(t, result[0].BorrowCurrency.IsEmpty(), "GetFlexibleLoanEModeInfo should return a borrow currency")
	}
}

func TestGetFlexibleLoanInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanInfo(contextGenerate(), "")
	require.NoError(t, err, "GetFlexibleLoanInfo must not error")
	if mockTests {
		exp := []FlexibleLoanInfoResponse{
			{
				OrderID:               "12345",
				LoanNotionalUSD:       0.8661285,
				LoanData:              []FlexibleLoanAsset{{Currency: currency.USDC, Amount: 0.86590608}},
				CollateralNotionalUSD: 1.5078763,
				CollateralData: []FlexibleLoanAsset{
					{Currency: currency.COMP, Amount: 0.0000097},
					{Currency: currency.STX, Amount: 0.78},
					{Currency: currency.DOT, Amount: 0.001},
					{Currency: currency.LUNA, Amount: 0.05357864},
				},
				CurrentLTV:     0.5742,
				MarginCallLTV:  0.7374,
				LiquidationLTV: 0.8374,
			},
			{
				OrderID:               "12346",
				LoanNotionalUSD:       1000.1,
				LoanData:              []FlexibleLoanAsset{{Currency: currency.USDT, Amount: 1000}},
				CollateralNotionalUSD: 1766.53,
				CollateralData:        []FlexibleLoanAsset{{Currency: currency.BTC, Amount: 0.02}},
				RiskWarningData:       FlexibleLoanRiskWarning{InstrumentID: "BTC-USDT", LiquidationPrice: 59713.26},
				CurrentLTV:            0.5662,
				MarginCallLTV:         0.7,
				LiquidationLTV:        0.85,
			},
		}
		assert.Equal(t, exp, result, "GetFlexibleLoanInfo should decode every field")
	} else {
		assert.NotNil(t, result, "GetFlexibleLoanInfo should return loans")
	}
}

func TestGetFlexibleLoanHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleLoanHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleLoanHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanHistory(contextGenerate(), &FlexibleLoanHistoryRequest{Type: "collateral_locked", After: "17316594851045087", Before: "17316594851045085", Limit: 1, OrderID: "12345"})
	require.NoError(t, err, "GetFlexibleLoanHistory must not error")
	if mockTests {
		exp := []FlexibleLoanHistoryResponse{
			{ReferenceID: "17316594851045086", Type: "collateral_locked", Currency: currency.DOT, Amount: -0.001, Timestamp: types.Time(time.UnixMilli(1731659485000))},
		}
		assert.Equal(t, exp, result, "GetFlexibleLoanHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetFlexibleLoanHistory should return loan history")
	}
}

func TestGetFlexibleLoanAccruedInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleLoanAccruedInterest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleLoanAccruedInterest must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleLoanAccruedInterest(contextGenerate(), &FlexibleLoanAccruedInterestRequest{Currency: currency.USDC, After: "17319133035195745", Before: "17319133035195743", Limit: 1, OrderID: "12345"})
	require.NoError(t, err, "GetFlexibleLoanAccruedInterest must not error")
	if mockTests {
		exp := []FlexibleLoanAccruedInterestResponse{
			{
				ReferenceID:  "17319133035195744",
				Currency:     currency.USDC,
				Loan:         0.86599309,
				Interest:     0.00004054,
				InterestRate: 0.41,
				Timestamp:    types.Time(time.UnixMilli(1731913200000)),
			},
		}
		assert.Equal(t, exp, result, "GetFlexibleLoanAccruedInterest should decode every field")
	} else {
		assert.NotNil(t, result, "GetFlexibleLoanAccruedInterest should return accrued interest")
	}
}

func TestValidateFlexibleLoanCurrencyAmounts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		entries []FlexibleLoanCurrencyAmount
		err     error
	}{
		{name: "no entries"},
		{name: "valid entries", entries: []FlexibleLoanCurrencyAmount{{Currency: currency.BTC, Amount: 0.5}, {Currency: currency.ETH, Amount: 2}}},
		{name: "missing currency in the last entry", entries: []FlexibleLoanCurrencyAmount{{Currency: currency.BTC, Amount: 0.5}, {Amount: 2}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "zero amount", entries: []FlexibleLoanCurrencyAmount{{Currency: currency.BTC}}, err: limits.ErrAmountBelowMin},
		{name: "negative amount", entries: []FlexibleLoanCurrencyAmount{{Currency: currency.BTC, Amount: -1}}, err: limits.ErrAmountBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateFlexibleLoanCurrencyAmounts("collateralData", tc.entries)
			if tc.err == nil {
				assert.NoError(t, err, "validateFlexibleLoanCurrencyAmounts should accept valid entries")
				return
			}
			assert.ErrorIs(t, err, tc.err, "validateFlexibleLoanCurrencyAmounts should reject an invalid entry")
		})
	}
}
