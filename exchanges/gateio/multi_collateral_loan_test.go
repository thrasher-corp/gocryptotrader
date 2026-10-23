package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceMultiCollateralLoanOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceMultiCollateralLoanOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceMultiCollateralLoanOrder must reject a nil request")

	arg := &MultiCollateralLoanOrderRequest{BorrowAmount: 123}
	_, err = e.PlaceMultiCollateralLoanOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceMultiCollateralLoanOrder must require a borrow currency")

	_, err = e.PlaceMultiCollateralLoanOrder(t.Context(), &MultiCollateralLoanOrderRequest{BorrowCurrency: currency.ETH})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceMultiCollateralLoanOrder must require a borrow amount")

	arg.BorrowCurrency = currency.ETH
	arg.OrderType = "fixed"
	_, err = e.PlaceMultiCollateralLoanOrder(t.Context(), arg)
	require.ErrorIs(t, err, errFixedRateTermRequired, "PlaceMultiCollateralLoanOrder must require a term for fixed rate orders")

	arg.FixedType = "7d"
	_, err = e.PlaceMultiCollateralLoanOrder(t.Context(), arg)
	require.ErrorIs(t, err, errFixedRateRequired, "PlaceMultiCollateralLoanOrder must require a rate for fixed rate orders")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceMultiCollateralLoanOrder(t.Context(), &MultiCollateralLoanOrderRequest{BorrowCurrency: currency.ETH, BorrowAmount: 123})
	require.NoError(t, err, "PlaceMultiCollateralLoanOrder must not error")
	if mockTests {
		assert.Equal(t, &OrderIDResponse{OrderID: 10005578}, result, "PlaceMultiCollateralLoanOrder should decode the order ID")
		return
	}
	assert.NotNil(t, result, "PlaceMultiCollateralLoanOrder should return the order ID")
}

// multiCollateralLoanOrderFixture returns the multi-collateral loan order recorded for the order list and detail fixtures
func multiCollateralLoanOrderFixture() *MultiCollateralLoanOrderDetailResponse {
	return &MultiCollateralLoanOrderDetailResponse{
		OrderID:                 "10005578",
		OrderType:               "fixed",
		FixedType:               "7d",
		FixedRate:               0.00001,
		ExpireTime:              types.Time(time.Unix(1703820105, 0)),
		AutoRenew:               true,
		AutoRepay:               true,
		CurrentLTV:              0.0001004349664281,
		Status:                  "lent",
		BorrowTime:              types.Time(time.Unix(1702615021, 0)),
		TotalLeftRepayUSDT:      106.491212982,
		TotalLeftCollateralUSDT: 1060300.18,
		BorrowCurrencies: []*BorrowCurrencyInfo{
			{Currency: currency.GT, IndexPrice: 10.6491, LeftRepayPrincipal: 10, LeftRepayInterest: 0.00002, LeftRepayUSDT: 106.491212982},
		},
		CollateralCurrencies: []*CollateralCurrencyInfo{
			{Currency: currency.BTC, IndexPrice: 112794.7, LeftCollateral: 9.4, LeftCollateralUSDT: 1060270.18},
			{Currency: currency.USDT, IndexPrice: 1, LeftCollateral: 30, LeftCollateralUSDT: 30},
		},
	}
}

func TestGetMultiCollateralLoanOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct{ sortType, orderType string }{{"", "current"}, {"time_desc", ""}, {"time_desc", "current"}} {
		result, err := e.GetMultiCollateralLoanOrders(t.Context(), 1, 100, tc.sortType, tc.orderType)
		require.NoErrorf(t, err, "GetMultiCollateralLoanOrders must not error for sort %q and order type %q", tc.sortType, tc.orderType)
		if mockTests {
			assert.Equalf(t, []*MultiCollateralLoanOrderDetailResponse{multiCollateralLoanOrderFixture()}, result, "GetMultiCollateralLoanOrders should decode every field for sort %q and order type %q", tc.sortType, tc.orderType)
			continue
		}
		assert.NotNilf(t, result, "GetMultiCollateralLoanOrders should return a result for sort %q and order type %q", tc.sortType, tc.orderType)
	}
}

func TestGetMultiCollateralLoanOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCollateralLoanOrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetMultiCollateralLoanOrderDetail must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMultiCollateralLoanOrderDetail(t.Context(), "1234556")
	require.NoError(t, err, "GetMultiCollateralLoanOrderDetail must not error")
	if mockTests {
		exp := multiCollateralLoanOrderFixture()
		exp.OrderID = "1234556"
		assert.Equal(t, exp, result, "GetMultiCollateralLoanOrderDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMultiCollateralLoanOrderDetail should return a result")
}

func TestRepayMultiCollateralLoan(t *testing.T) {
	t.Parallel()
	_, err := e.RepayMultiCollateralLoan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RepayMultiCollateralLoan must reject a nil request")

	_, err = e.RepayMultiCollateralLoan(t.Context(), &MultiCollateralLoanRepaymentRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "RepayMultiCollateralLoan must require an order ID")

	arg := &MultiCollateralLoanRepaymentRequest{OrderID: 1234}
	_, err = e.RepayMultiCollateralLoan(t.Context(), arg)
	require.ErrorIs(t, err, errNoValidParameterPassed, "RepayMultiCollateralLoan must require repay items")

	arg.RepayItems = []LoanRepaymentMultiCollateralAssetItem{{RepaidAll: true}}
	_, err = e.RepayMultiCollateralLoan(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "RepayMultiCollateralLoan must require each repay item's currency")

	arg.RepayItems = []LoanRepaymentMultiCollateralAssetItem{{Currency: currency.USDT}}
	_, err = e.RepayMultiCollateralLoan(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "RepayMultiCollateralLoan must require an amount for partial repayments")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.RepayItems = []LoanRepaymentMultiCollateralAssetItem{{Currency: currency.USDT, RepaidAll: true}, {Currency: currency.BTC, Amount: 1}}
	result, err := e.RepayMultiCollateralLoan(t.Context(), arg)
	require.NoError(t, err, "RepayMultiCollateralLoan must not error")
	if mockTests {
		exp := &MultiCollateralLoanRepaymentResponse{
			OrderID: 1234,
			RepaidCurrencies: []*MultiCollateralRepaidCurrency{
				{Succeeded: true, Currency: currency.USDT, RepaidPrincipal: 10, RepaidInterest: 0.0001},
				{Label: "INVALID_PARAM_VALUE", Message: "Invalid parameter value", Currency: currency.BTC},
			},
		}
		assert.Equal(t, exp, result, "RepayMultiCollateralLoan should decode every field")
		return
	}
	assert.NotNil(t, result, "RepayMultiCollateralLoan should return a result")
}

func TestGetMultiCurrencyCollateralRepaymentRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMultiCurrencyCollateralRepaymentRecords must reject a nil request")

	_, err = e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), &MultiCollateralRepaymentRecordsRequest{BorrowCurrency: currency.USDT, Limit: 10})
	require.ErrorIs(t, err, errLoanTypeIsRequired, "GetMultiCurrencyCollateralRepaymentRecords must require an operation type")

	startTime, endTime := getTime()
	_, err = e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), &MultiCollateralRepaymentRecordsRequest{Type: "repay", From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMultiCurrencyCollateralRepaymentRecords must reject a reversed time range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMultiCurrencyCollateralRepaymentRecords(t.Context(), &MultiCollateralRepaymentRecordsRequest{Type: "repay", BorrowCurrency: currency.USDT, Limit: 10})
	require.NoError(t, err, "GetMultiCurrencyCollateralRepaymentRecords must not error")
	if mockTests {
		exp := []*MultiCurrencyCollateralRepaymentResponse{{
			OrderID:    10005679,
			RecordID:   1348,
			InitLTV:    0.2141,
			BeforeLTV:  0.215,
			AfterLTV:   0.312,
			BorrowTime: types.Time(time.Unix(1702995889, 0)),
			RepayTime:  types.Time(time.Unix(1703053927, 0)),
			BorrowCurrencies: []*CurrencyDetail{
				{Currency: currency.USDT, IndexPrice: 1, BeforeAmount: 1, BeforeAmountUSDT: 1, AfterAmount: 0.999017, AfterAmountUSDT: 0.999017},
			},
			CollateralCurrencies: []*CurrencyDetail{
				{Currency: currency.ETC, IndexPrice: 0.6014228107, BeforeAmount: 1000, BeforeAmountUSDT: 601.4228107, AfterAmount: 1000, AfterAmountUSDT: 601.4228107},
			},
			RepaidCurrencies: []*RepayRecordRepaidCurrency{
				{Currency: currency.USDT, IndexPrice: 1, RepaidAmount: 0.001, RepaidPrincipal: 0.000983, RepaidInterest: 0.000017, RepaidAmountUSDT: 0.001},
			},
			TotalInterestList: []*RepayRecordTotalInterest{
				{Currency: currency.USDT, IndexPrice: 1, Amount: 0.000017, AmountUSDT: 0.000017},
			},
			LeftRepayInterestList: []*CurrencyDetail{
				{Currency: currency.USDT, IndexPrice: 1, BeforeAmount: 0.000017, BeforeAmountUSDT: 0.000017},
			},
		}}
		assert.Equal(t, exp, result, "GetMultiCurrencyCollateralRepaymentRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMultiCurrencyCollateralRepaymentRecords should return a result")
}

func TestAddOrWithdrawCollateral(t *testing.T) {
	t.Parallel()
	_, err := e.AddOrWithdrawCollateral(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddOrWithdrawCollateral must reject a nil request")

	_, err = e.AddOrWithdrawCollateral(t.Context(), &AddOrWithdrawCollateralRequest{OperationType: "redeem"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AddOrWithdrawCollateral must require an order ID")

	_, err = e.AddOrWithdrawCollateral(t.Context(), &AddOrWithdrawCollateralRequest{OrderID: 12344})
	require.ErrorIs(t, err, errLoanTypeIsRequired, "AddOrWithdrawCollateral must require an operation type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AddOrWithdrawCollateral(t.Context(), &AddOrWithdrawCollateralRequest{
		OrderID:       12344,
		OperationType: "redeem",
		Collaterals:   []CurrencyAndAmountResponse{{Currency: currency.BTC, Amount: 0.5}},
	})
	require.NoError(t, err, "AddOrWithdrawCollateral must not error")
	if mockTests {
		exp := &CollateralAddOrRemoveResponse{
			OrderID:              12344,
			CollateralCurrencies: []*CollateralOperationResult{{Succeeded: true, Currency: currency.BTC, Amount: 0.5}},
		}
		assert.Equal(t, exp, result, "AddOrWithdrawCollateral should decode every field")
		return
	}
	assert.NotNil(t, result, "AddOrWithdrawCollateral should return a result")
}

func TestGetMultiCollateralAdjustmentRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCollateralAdjustmentRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMultiCollateralAdjustmentRecords must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetMultiCollateralAdjustmentRecords(t.Context(), &MultiCollateralAdjustmentRecordsRequest{Page: 1, Limit: 10, From: endTime, To: startTime, CollateralCurrency: currency.ETH})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMultiCollateralAdjustmentRecords must reject a reversed time range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMultiCollateralAdjustmentRecords(t.Context(), &MultiCollateralAdjustmentRecordsRequest{Page: 1, Limit: 10, From: startTime, To: endTime, CollateralCurrency: currency.ETH})
	require.NoError(t, err, "GetMultiCollateralAdjustmentRecords must not error")
	if mockTests {
		exp := []*MultiCollateralAdjustmentRecordResponse{{
			OrderID:     10000417,
			RecordID:    10000452,
			BeforeLTV:   0.0003934555562148,
			AfterLTV:    0.0001967277781074,
			OperateTime: types.Time(time.Unix(1744150000, 0)),
			BorrowCurrencies: []*CurrencyDetail{
				{Currency: currency.USDT, IndexPrice: 1, BeforeAmount: 1000, BeforeAmountUSDT: 1000, AfterAmount: 1000, AfterAmountUSDT: 1000},
			},
			CollateralCurrencies: []*CurrencyDetail{
				{Currency: currency.ETH, IndexPrice: 1500, BeforeAmount: 0.5, BeforeAmountUSDT: 750, AfterAmount: 1, AfterAmountUSDT: 1500},
			},
		}}
		assert.Equal(t, exp, result, "GetMultiCollateralAdjustmentRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMultiCollateralAdjustmentRecords should return a result")
}

func TestGetMultiCollateralCurrencyQuota(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCollateralCurrencyQuota(t.Context(), "", []currency.Code{currency.BTC})
	require.ErrorIs(t, err, errCurrencyTypeRequired, "GetMultiCollateralCurrencyQuota must require a quota type")

	_, err = e.GetMultiCollateralCurrencyQuota(t.Context(), "collateral", nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMultiCollateralCurrencyQuota must require a currency")

	_, err = e.GetMultiCollateralCurrencyQuota(t.Context(), "borrow", []currency.Code{currency.BTC, currency.ETH})
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetMultiCollateralCurrencyQuota must accept a single borrow currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMultiCollateralCurrencyQuota(t.Context(), "collateral", []currency.Code{currency.BTC})
	require.NoError(t, err, "GetMultiCollateralCurrencyQuota must not error")
	if mockTests {
		exp := []*MultiCollateralCurrencyQuotaResponse{{
			Currency:           currency.BTC,
			IndexPrice:         35306.1,
			LeftQuota:          2768152.4958445218723677,
			LeftQuotaUSDT:      97732668833.536273678,
			LeftQuotaFixed:     1000,
			LeftQuotaUSDTFixed: 35306100,
		}}
		assert.Equal(t, exp, result, "GetMultiCollateralCurrencyQuota should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMultiCollateralCurrencyQuota should return a result")
}

func TestGetMultiCollateralSupportedCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetMultiCollateralSupportedCurrencies(t.Context())
	require.NoError(t, err, "GetMultiCollateralSupportedCurrencies must not error")
	if mockTests {
		exp := &MultiCollateralSupportedCurrenciesResponse{
			LoanCurrencies:       []*MultiLoanItem{{Currency: currency.BTC, Price: 1212}, {Currency: currency.GT, Price: 12}},
			CollateralCurrencies: []*MultiCollateralItem{{Currency: currency.BTC, IndexPrice: 1212, Discount: 0.7}},
		}
		assert.Equal(t, exp, result, "GetMultiCollateralSupportedCurrencies should decode every field")
		return
	}
	require.NotNil(t, result, "GetMultiCollateralSupportedCurrencies must return a result")
	assert.NotEmpty(t, result.LoanCurrencies, "GetMultiCollateralSupportedCurrencies should return loan currencies")
}

func TestGetMultiCollateralisationRatio(t *testing.T) {
	t.Parallel()
	result, err := e.GetMultiCollateralisationRatio(t.Context())
	require.NoError(t, err, "GetMultiCollateralisationRatio must not error")
	if mockTests {
		assert.Equal(t, &MultiCollateralisationRatioResponse{InitLTV: 0.7, AlertLTV: 0.8, LiquidateLTV: 0.9}, result, "GetMultiCollateralisationRatio should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMultiCollateralisationRatio should return a result")
}

func TestGetMultiCollateralFixedRates(t *testing.T) {
	t.Parallel()
	result, err := e.GetMultiCollateralFixedRates(t.Context())
	require.NoError(t, err, "GetMultiCollateralFixedRates must not error")
	if mockTests {
		exp := []*CollateralFixRateResponse{{Currency: currency.BTC, Rate7Day: 0.000023, Rate30Day: 0.1, UpdateTime: types.Time(time.Unix(1703820105, 0))}}
		assert.Equal(t, exp, result, "GetMultiCollateralFixedRates should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetMultiCollateralFixedRates should return rates")
}

func TestGetMultiCollateralCurrentRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultiCollateralCurrentRates(t.Context(), nil, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMultiCollateralCurrentRates must require a currency")

	_, err = e.GetMultiCollateralCurrentRates(t.Context(), make([]currency.Code, 101), "")
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "GetMultiCollateralCurrentRates must reject more than 100 currencies")

	result, err := e.GetMultiCollateralCurrentRates(t.Context(), []currency.Code{currency.BTC, currency.GT}, "")
	require.NoError(t, err, "GetMultiCollateralCurrentRates must not error")
	if mockTests {
		exp := []*CollateralCurrentRateResponse{{Currency: currency.BTC, CurrentRate: 0.000023}, {Currency: currency.GT, CurrentRate: 0.000015}}
		assert.Equal(t, exp, result, "GetMultiCollateralCurrentRates should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetMultiCollateralCurrentRates should return a rate for each currency")
}
