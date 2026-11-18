package kucoin

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

func TestGetBorrowInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetBorrowInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBorrowInterestRate must reject a nil request")
	_, err = e.GetBorrowInterestRate(t.Context(), &BorrowInterestRateRequest{Currencies: []currency.Code{currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBorrowInterestRate must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	var vipLevel uint64
	for _, tc := range []struct {
		name string
		req  *BorrowInterestRateRequest
		exp  *BorrowInterestRateResponse
	}{
		{
			name: "own VIP level",
			req:  &BorrowInterestRateRequest{},
			exp: &BorrowInterestRateResponse{
				VIPLevel: 2,
				Items: []BorrowInterestRate{
					{Currency: currency.BTC, HourlyBorrowRate: 0.00000088, AnnualizedBorrowRate: 0.0077},
					{Currency: currency.ETH, HourlyBorrowRate: 0.00000421, AnnualizedBorrowRate: 0.0369},
				},
			},
		},
		{
			name: "VIP level 0",
			req:  &BorrowInterestRateRequest{VIPLevel: &vipLevel, Currencies: []currency.Code{currency.BTC, currency.ETH}},
			exp: &BorrowInterestRateResponse{
				Items: []BorrowInterestRate{
					{Currency: currency.BTC, HourlyBorrowRate: 0.00000098, AnnualizedBorrowRate: 0.0086},
					{Currency: currency.ETH, HourlyBorrowRate: 0.00000468, AnnualizedBorrowRate: 0.041},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetBorrowInterestRate(t.Context(), tc.req)
			require.NoError(t, err, "GetBorrowInterestRate must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetBorrowInterestRate should decode every field")
				return
			}
			assert.NotEmpty(t, result.Items, "GetBorrowInterestRate should return interest rates")
		})
	}
}

func TestPostMarginBorrowOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PostMarginBorrowOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PostMarginBorrowOrder must reject a nil request")
	_, err = e.PostMarginBorrowOrder(t.Context(), &MarginBorrowOrderRequest{Size: 0.05, TimeInForce: "IOC"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PostMarginBorrowOrder must reject an empty currency")
	_, err = e.PostMarginBorrowOrder(t.Context(), &MarginBorrowOrderRequest{Currency: currency.USDT, TimeInForce: "IOC"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PostMarginBorrowOrder must reject a zero size")
	_, err = e.PostMarginBorrowOrder(t.Context(), &MarginBorrowOrderRequest{Currency: currency.USDT, Size: 0.05})
	require.ErrorIs(t, err, errTimeInForceRequired, "PostMarginBorrowOrder must reject an empty time in force")
	_, err = e.PostMarginBorrowOrder(t.Context(), &MarginBorrowOrderRequest{Currency: currency.BTC, Size: 0.001, TimeInForce: "IOC", IsIsolated: true})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "PostMarginBorrowOrder must reject an isolated borrow without a symbol")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.PostMarginBorrowOrder(t.Context(), &MarginBorrowOrderRequest{Currency: currency.BTC, Size: 0.001, TimeInForce: "IOC", Symbol: "BTC-USDT", IsIsolated: true, IsHF: true})
		return err
	})
	assert.JSONEq(t, `{"currency":"BTC","size":0.001,"timeInForce":"IOC","symbol":"BTC-USDT","isIsolated":true,"isHf":true}`, sent.body, "PostMarginBorrowOrder should send the size as the number KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *MarginBorrowOrderRequest
		exp  *MarginBorrowOrderResponse
	}{
		{
			name: "cross",
			req:  &MarginBorrowOrderRequest{Currency: currency.USDT, Size: 0.05, TimeInForce: "IOC"},
			exp:  &MarginBorrowOrderResponse{OrderNumber: "67187162c0d6990007717b15", ActualSize: 0.05},
		},
		{
			name: "isolated",
			req:  &MarginBorrowOrderRequest{Currency: currency.BTC, Size: 0.001, TimeInForce: "IOC", Symbol: "BTC-USDT", IsIsolated: true, IsHF: true},
			exp:  &MarginBorrowOrderResponse{OrderNumber: "67187155b088e70007149585", ActualSize: 0.001},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PostMarginBorrowOrder(t.Context(), tc.req)
			require.NoError(t, err, "PostMarginBorrowOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PostMarginBorrowOrder should decode every field")
				return
			}
			require.NotNil(t, result, "PostMarginBorrowOrder must return a result")
			assert.NotEmpty(t, result.OrderNumber, "PostMarginBorrowOrder should return an order number")
		})
	}
}

func TestGetMarginBorrowingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginBorrowingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginBorrowingHistory must reject a nil request")
	_, err = e.GetMarginBorrowingHistory(t.Context(), &MarginBorrowingHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMarginBorrowingHistory must reject an empty currency")
	_, err = e.GetMarginBorrowingHistory(t.Context(), &MarginBorrowingHistoryRequest{Currency: currency.BTC, IsIsolated: true})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetMarginBorrowingHistory must reject an isolated query without a symbol")
	_, err = e.GetMarginBorrowingHistory(t.Context(), &MarginBorrowingHistoryRequest{Currency: currency.BTC, StartTime: time.UnixMilli(1729656000000), EndTime: time.UnixMilli(1729652400000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginBorrowingHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *MarginBorrowingHistoryRequest
		exp  *MarginBorrowingHistoryResponse
	}{
		{
			name: "cross",
			req:  &MarginBorrowingHistoryRequest{Currency: currency.USDT},
			exp: &MarginBorrowingHistoryResponse{
				Timestamp:   types.Time(time.UnixMilli(1729657580449)),
				CurrentPage: 1,
				PageSize:    50,
				TotalNumber: 2,
				TotalPage:   1,
				Items: []MarginBorrowOrder{
					{
						OrderNumber: "67187162c0d6990007717b15",
						Currency:    currency.USDT,
						Size:        10,
						ActualSize:  10,
						Status:      "SUCCESS",
						CreatedTime: types.Time(time.UnixMilli(1729655138000)),
					},
					{
						OrderNumber: "67187155b088e70007149585",
						Currency:    currency.USDT,
						Size:        0.1,
						Status:      "FAILED",
						CreatedTime: types.Time(time.UnixMilli(1729655125000)),
					},
				},
			},
		},
		{
			name: "isolated",
			req: &MarginBorrowingHistoryRequest{
				Currency:    currency.BTC,
				IsIsolated:  true,
				Symbol:      "BTC-USDT",
				OrderNumber: "67187162c0d6990007717b16",
				StartTime:   time.UnixMilli(1729652400000),
				EndTime:     time.UnixMilli(1729656000000),
				CurrentPage: 1,
				PageSize:    10,
			},
			exp: &MarginBorrowingHistoryResponse{
				Timestamp:   types.Time(time.UnixMilli(1729657580449)),
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []MarginBorrowOrder{
					{
						OrderNumber: "67187162c0d6990007717b16",
						Symbol:      "BTC-USDT",
						Currency:    currency.BTC,
						Size:        0.001,
						ActualSize:  0.001,
						Status:      "SUCCESS",
						CreatedTime: types.Time(time.UnixMilli(1729655138000)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetMarginBorrowingHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetMarginBorrowingHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetMarginBorrowingHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetMarginBorrowingHistory should return a result")
		})
	}
}

func TestPostRepayment(t *testing.T) {
	t.Parallel()
	_, err := e.PostRepayment(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PostRepayment must reject a nil request")
	_, err = e.PostRepayment(t.Context(), &MarginRepaymentRequest{Size: 0.05})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PostRepayment must reject an empty currency")
	_, err = e.PostRepayment(t.Context(), &MarginRepaymentRequest{Currency: currency.ETH})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PostRepayment must reject a zero size")
	_, err = e.PostRepayment(t.Context(), &MarginRepaymentRequest{Currency: currency.BTC, Size: 0.001, IsIsolated: true})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "PostRepayment must reject an isolated repayment without a symbol")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.PostRepayment(t.Context(), &MarginRepaymentRequest{Currency: currency.BTC, Size: 0.001, Symbol: "BTC-USDT", IsIsolated: true, IsHF: true})
		return err
	})
	assert.JSONEq(t, `{"currency":"BTC","size":0.001,"symbol":"BTC-USDT","isIsolated":true,"isHf":true}`, sent.body, "PostRepayment should send the size as the number KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *MarginRepaymentRequest
		exp  *MarginRepaymentResponse
	}{
		{
			name: "cross",
			req:  &MarginRepaymentRequest{Currency: currency.USDT, Size: 0.05},
			exp: &MarginRepaymentResponse{
				Timestamp:   types.Time(time.UnixMilli(1729655606816)),
				OrderNumber: "671873361d5bd400075096ad",
				ActualSize:  0.05,
			},
		},
		{
			name: "isolated",
			req:  &MarginRepaymentRequest{Currency: currency.BTC, Size: 0.001, Symbol: "BTC-USDT", IsIsolated: true, IsHF: true},
			exp: &MarginRepaymentResponse{
				Timestamp:   types.Time(time.UnixMilli(1729655606817)),
				OrderNumber: "671873361d5bd400075096ae",
				ActualSize:  0.001,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PostRepayment(t.Context(), tc.req)
			require.NoError(t, err, "PostRepayment must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PostRepayment should decode every field")
				return
			}
			require.NotNil(t, result, "PostRepayment must return a result")
			assert.NotEmpty(t, result.OrderNumber, "PostRepayment should return an order number")
		})
	}
}

func TestGetRepaymentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetRepaymentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRepaymentHistory must reject a nil request")
	_, err = e.GetRepaymentHistory(t.Context(), &MarginRepaymentHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetRepaymentHistory must reject an empty currency")
	_, err = e.GetRepaymentHistory(t.Context(), &MarginRepaymentHistoryRequest{Currency: currency.BTC, IsIsolated: true})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetRepaymentHistory must reject an isolated query without a symbol")
	_, err = e.GetRepaymentHistory(t.Context(), &MarginRepaymentHistoryRequest{Currency: currency.BTC, StartTime: time.UnixMilli(1729659600000), EndTime: time.UnixMilli(1729652400000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetRepaymentHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *MarginRepaymentHistoryRequest
		exp  *MarginRepaymentHistoryResponse
	}{
		{
			name: "cross",
			req:  &MarginRepaymentHistoryRequest{Currency: currency.USDT},
			exp: &MarginRepaymentHistoryResponse{
				Timestamp:   types.Time(time.UnixMilli(1729663471891)),
				CurrentPage: 1,
				PageSize:    50,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []MarginRepaymentOrder{
					{
						OrderNumber: "671873361d5bd400075096ad",
						Currency:    currency.USDT,
						Size:        10,
						Principal:   9.99986518,
						Interest:    0.00013482,
						Status:      "SUCCESS",
						CreatedTime: types.Time(time.UnixMilli(1729655606000)),
					},
				},
			},
		},
		{
			name: "isolated",
			req: &MarginRepaymentHistoryRequest{
				Currency:    currency.BTC,
				IsIsolated:  true,
				Symbol:      "BTC-USDT",
				OrderNumber: "671873361d5bd400075096ae",
				StartTime:   time.UnixMilli(1729652400000),
				EndTime:     time.UnixMilli(1729659600000),
				CurrentPage: 1,
				PageSize:    10,
			},
			exp: &MarginRepaymentHistoryResponse{
				Timestamp:   types.Time(time.UnixMilli(1729663471891)),
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []MarginRepaymentOrder{
					{
						OrderNumber: "671873361d5bd400075096ae",
						Symbol:      "BTC-USDT",
						Currency:    currency.BTC,
						Size:        0.001,
						Principal:   0.00099986,
						Interest:    0.00000014,
						Status:      "SUCCESS",
						CreatedTime: types.Time(time.UnixMilli(1729655606000)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRepaymentHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetRepaymentHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetRepaymentHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetRepaymentHistory should return a result")
		})
	}
}

func TestGetCrossIsolatedMarginInterestRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossIsolatedMarginInterestRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossIsolatedMarginInterestRecords must reject a nil request")
	_, err = e.GetCrossIsolatedMarginInterestRecords(t.Context(), &MarginInterestRecordsRequest{IsIsolated: true})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetCrossIsolatedMarginInterestRecords must reject an isolated query without a symbol")
	_, err = e.GetCrossIsolatedMarginInterestRecords(t.Context(), &MarginInterestRecordsRequest{StartTime: time.UnixMilli(1729666800000), EndTime: time.UnixMilli(1729652400000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCrossIsolatedMarginInterestRecords must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *MarginInterestRecordsRequest
		exp  *MarginInterestRecordsResponse
	}{
		{
			name: "cross",
			req:  &MarginInterestRecordsRequest{},
			exp: &MarginInterestRecordsResponse{
				Timestamp:   types.Time(time.UnixMilli(1729665170701)),
				CurrentPage: 1,
				PageSize:    50,
				TotalNumber: 3,
				TotalPage:   1,
				Items: []MarginInterestRecord{
					{Currency: currency.USDT, DayRatio: 0.000296, InterestAmount: 0.00000001, CreatedTime: types.Time(time.UnixMilli(1729663213375))},
					{Currency: currency.USDT, DayRatio: 0.000296, InterestAmount: 0.00000001, CreatedTime: types.Time(time.UnixMilli(1729659618802))},
					{Currency: currency.USDT, DayRatio: 0.000296, InterestAmount: 0.00000001, CreatedTime: types.Time(time.UnixMilli(1729656028077))},
				},
			},
		},
		{
			name: "isolated",
			req: &MarginInterestRecordsRequest{
				Currency:    currency.BTC,
				IsIsolated:  true,
				Symbol:      "BTC-USDT",
				StartTime:   time.UnixMilli(1729652400000),
				EndTime:     time.UnixMilli(1729666800000),
				CurrentPage: 1,
				PageSize:    10,
			},
			exp: &MarginInterestRecordsResponse{
				Timestamp:   types.Time(time.UnixMilli(1729665170701)),
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []MarginInterestRecord{
					{Currency: currency.BTC, DayRatio: 0.000096, InterestAmount: 0.00000001, CreatedTime: types.Time(time.UnixMilli(1729663213375))},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetCrossIsolatedMarginInterestRecords(t.Context(), tc.req)
			require.NoError(t, err, "GetCrossIsolatedMarginInterestRecords must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCrossIsolatedMarginInterestRecords should decode every field")
				return
			}
			assert.NotNil(t, result, "GetCrossIsolatedMarginInterestRecords should return a result")
		})
	}
}

func TestModifyLeverageMultiplier(t *testing.T) {
	t.Parallel()
	err := e.ModifyLeverageMultiplier(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifyLeverageMultiplier must reject a nil request")
	err = e.ModifyLeverageMultiplier(t.Context(), &ModifyLeverageMultiplierRequest{Leverage: 3.5, IsIsolated: true})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "ModifyLeverageMultiplier must reject an isolated change without a symbol")
	err = e.ModifyLeverageMultiplier(t.Context(), &ModifyLeverageMultiplierRequest{})
	require.ErrorIs(t, err, errInvalidLeverage, "ModifyLeverageMultiplier must reject a zero leverage")
	err = e.ModifyLeverageMultiplier(t.Context(), &ModifyLeverageMultiplierRequest{Leverage: 2.555})
	require.ErrorIs(t, err, errInvalidLeverage, "ModifyLeverageMultiplier must reject a leverage with more than two decimal places")

	sent := sentRequest(t, func(ex *Exchange) error {
		return ex.ModifyLeverageMultiplier(t.Context(), &ModifyLeverageMultiplierRequest{Symbol: "BTC-USDT", IsIsolated: true, Leverage: 3.5})
	})
	assert.JSONEq(t, `{"leverage":"3.5","symbol":"BTC-USDT","isIsolated":true}`, sent.body, "ModifyLeverageMultiplier should send the leverage as the string KuCoin documents")
	sent = sentRequest(t, func(ex *Exchange) error {
		return ex.ModifyLeverageMultiplier(t.Context(), &ModifyLeverageMultiplierRequest{Leverage: 2.55})
	})
	assert.JSONEq(t, `{"leverage":"2.55"}`, sent.body, "ModifyLeverageMultiplier should send a leverage with two decimal places as given")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *ModifyLeverageMultiplierRequest
	}{
		{name: "cross", req: &ModifyLeverageMultiplierRequest{Leverage: 5}},
		{name: "isolated", req: &ModifyLeverageMultiplierRequest{Symbol: "BTC-USDT", IsIsolated: true, Leverage: 3.5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, e.ModifyLeverageMultiplier(t.Context(), tc.req), "ModifyLeverageMultiplier should not error")
		})
	}
}
