package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// utaAffiliateTestStart and utaAffiliateTestEnd bound the window the filtered UTA affiliate requests query
var (
	utaAffiliateTestStart = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	utaAffiliateTestEnd   = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
)

func TestGetUTAAffiliateInvitees(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAAffiliateInvitees(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAAffiliateInvitees must reject a nil request")
	_, err = e.GetUTAAffiliateInvitees(t.Context(), &UTAAffiliateInviteesRequest{RegistrationStartAt: utaAffiliateTestEnd, RegistrationEndAt: utaAffiliateTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAAffiliateInvitees must reject a start after the end")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	exp := &UTAAffiliateInviteesResponse{
		CurrentPage: 1,
		PageSize:    10,
		TotalNumber: 2,
		TotalPage:   1,
		Items: []UTAAffiliateInvitee{
			{
				UID:                   "100000001",
				NickName:              "do**@**.com",
				ReferralCode:          "CX87KDUK",
				Country:               "United States of America",
				RegistrationTime:      types.Time(time.Unix(1788737999, 0)),
				CompletedKYC:          true,
				CompletedFirstDeposit: true,
				CompletedFirstTrade:   true,
				Past7DayFees:          12.48,
				Past7DayCommission:    3.62,
				TotalCommission:       15.27,
				MyCommissionRate:      29,
				CashbackRate:          11,
				Currency:              currency.USDT,
			},
			{
				UID:              "100000002",
				NickName:         "sa**@**.com",
				ReferralCode:     "CX87KDUK",
				Country:          "Nigeria",
				RegistrationTime: types.Time(time.Unix(1789245613, 0)),
				MyCommissionRate: 29,
				CashbackRate:     11,
				Currency:         currency.USDC,
			},
		},
	}
	for _, tc := range []struct {
		name string
		req  *UTAAffiliateInviteesRequest
	}{
		{name: "every invitee", req: &UTAAffiliateInviteesRequest{}},
		{name: "filtered", req: &UTAAffiliateInviteesRequest{
			UserType:            "1",
			ReferralCode:        "CX87KDUK",
			UIDs:                []string{"100000001", "100000002"},
			RegistrationStartAt: utaAffiliateTestStart,
			RegistrationEndAt:   utaAffiliateTestEnd,
			Page:                1,
			PageSize:            10,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAAffiliateInvitees(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAAffiliateInvitees must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTAAffiliateInvitees should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTAAffiliateInvitees should return a page")
		})
	}
}

func TestGetUTAAffiliateCommission(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAAffiliateCommission(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAAffiliateCommission must reject a nil request")
	_, err = e.GetUTAAffiliateCommission(t.Context(), &UTAAffiliateCommissionRequest{RebateStartAt: utaAffiliateTestEnd, RebateEndAt: utaAffiliateTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAAffiliateCommission must reject a start after the end")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	for _, tc := range []struct {
		name string
		req  *UTAAffiliateCommissionRequest
		exp  *UTAAffiliateCommissionResponse
	}{
		{name: "trade commission", req: &UTAAffiliateCommissionRequest{}, exp: &UTAAffiliateCommissionResponse{
			CurrentPage: 1,
			PageSize:    10,
			TotalNumber: 1,
			TotalPage:   1,
			Items: []UTAAffiliateCommission{
				{
					SiteType:        "global",
					RebateType:      1,
					PayoutTime:      types.Time(time.Unix(1786032000, 0)),
					PeriodStartTime: types.Time(time.Unix(1785945600, 0)),
					PeriodEndTime:   types.Time(time.UnixMilli(1786031999999)),
					Status:          3,
					TakerVolume:     1520.39,
					MakerVolume:     860.25,
					Commission:      0.86,
					Currency:        currency.USDT,
					DataType:        "trade",
				},
			},
		}},
		{
			name: "kumining commission",
			req: &UTAAffiliateCommissionRequest{
				SiteType:      "global",
				RebateType:    2,
				RebateStartAt: utaAffiliateTestStart,
				RebateEndAt:   utaAffiliateTestEnd,
				Page:          1,
				PageSize:      10,
				DataType:      "kumining",
			},
			exp: &UTAAffiliateCommissionResponse{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []UTAAffiliateCommission{
					{
						SiteType:        "global",
						RebateType:      2,
						PayoutTime:      types.Time(time.Unix(1788393600, 0)),
						PeriodStartTime: types.Time(time.Unix(1788307200, 0)),
						PeriodEndTime:   types.Time(time.UnixMilli(1788393599999)),
						Status:          1,
						KuminingVolume:  350.5,
						Commission:      1.75,
						Currency:        currency.USDT,
						DataType:        "kumining",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAAffiliateCommission(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAAffiliateCommission must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAAffiliateCommission should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTAAffiliateCommission should return a page")
		})
	}
}

func TestGetUTAAffiliateTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAAffiliateTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAAffiliateTradeHistory must reject a nil request")
	_, err = e.GetUTAAffiliateTradeHistory(t.Context(), &UTAAffiliateTradeHistoryRequest{})
	require.ErrorIs(t, err, errUTAInviteeUIDRequired, "GetUTAAffiliateTradeHistory must reject an empty UID")
	_, err = e.GetUTAAffiliateTradeHistory(t.Context(), &UTAAffiliateTradeHistoryRequest{UID: "100000001", TradeStartAt: utaAffiliateTestEnd, TradeEndAt: utaAffiliateTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAAffiliateTradeHistory must reject a start after the end")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	exp := &UTAAffiliateTradeHistoryResponse{
		CurrentPage: 1,
		PageSize:    10,
		TotalNumber: 2,
		TotalPage:   1,
		Items: []UTAAffiliateTrade{
			{
				TradeTime:     types.Time(time.Unix(1788560532, 0)),
				TradeType:     "SPOT",
				TradeCurrency: currency.NewCode("TRUMP"),
				TradeAmount:   0.62,
				TradeAmountU:  4.96,
				FeeU:          0.01,
				Commission:    0.0029,
				Currency:      currency.USDT,
			},
			{
				TradeTime:     types.Time(time.Unix(1788604768, 0)),
				TradeType:     "SPOT",
				TradeCurrency: currency.DOGE,
				TradeAmount:   10,
				TradeAmountU:  1.98,
				FeeU:          0.002,
				Commission:    0.0006,
				Currency:      currency.USDT,
			},
		},
	}
	for _, tc := range []struct {
		name string
		req  *UTAAffiliateTradeHistoryRequest
	}{
		{name: "every trade", req: &UTAAffiliateTradeHistoryRequest{UID: "100000001"}},
		{name: "filtered", req: &UTAAffiliateTradeHistoryRequest{
			UID:          "100000001",
			TradeType:    "SPOT",
			TradeStartAt: utaAffiliateTestStart,
			TradeEndAt:   utaAffiliateTestEnd,
			Page:         1,
			PageSize:     10,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAAffiliateTradeHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAAffiliateTradeHistory must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTAAffiliateTradeHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTAAffiliateTradeHistory should return a page")
		})
	}
}

func TestGetUTAAffiliateTransactions(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAAffiliateTransactions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAAffiliateTransactions must reject a nil request")
	_, err = e.GetUTAAffiliateTransactions(t.Context(), &UTAAffiliateTransactionsRequest{TradeEndAt: utaAffiliateTestEnd})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUTAAffiliateTransactions must reject an unset start")
	_, err = e.GetUTAAffiliateTransactions(t.Context(), &UTAAffiliateTransactionsRequest{TradeStartAt: utaAffiliateTestEnd, TradeEndAt: utaAffiliateTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAAffiliateTransactions must reject a start after the end")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	exp := []UTAAffiliateTransaction{
		{
			UID:           "100000001",
			TradeTime:     types.Time(time.Unix(1788516800, 0)),
			TradeType:     "SPOT",
			TradeCurrency: currency.BTC,
			TradeAmount:   0.1,
			TradeAmountU:  8350,
			FeeU:          8.35,
			Commission:    2.42,
			Currency:      currency.USDT,
			LastID:        "6498040334d4050001b0aa7c",
		},
	}
	for _, tc := range []struct {
		name string
		req  *UTAAffiliateTransactionsRequest
	}{
		{name: "window only", req: &UTAAffiliateTransactionsRequest{TradeStartAt: utaAffiliateTestStart, TradeEndAt: utaAffiliateTestEnd}},
		{name: "filtered", req: &UTAAffiliateTransactionsRequest{
			UID:          "100000001",
			TradeType:    "SPOT",
			TradeStartAt: utaAffiliateTestStart,
			TradeEndAt:   utaAffiliateTestEnd,
			LastID:       "6498040334d4050001b0aa7b",
			Direction:    "NEXT",
			PageSize:     10,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAAffiliateTransactions(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAAffiliateTransactions must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTAAffiliateTransactions should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTAAffiliateTransactions should return transactions")
		})
	}
}

func TestGetUTAAffiliateKumining(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAAffiliateKumining(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAAffiliateKumining must reject a nil request")
	_, err = e.GetUTAAffiliateKumining(t.Context(), &UTAAffiliateKuminingRequest{StartAt: utaAffiliateTestEnd, EndAt: utaAffiliateTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAAffiliateKumining must reject a start after the end")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	exp := []UTAAffiliateKumining{
		{
			UID:       "100000001",
			PayTime:   types.Time(time.Unix(1788846542, 0)),
			GoodsName: "BTC-10TH",
			Amount:    0.7,
			Currency:  currency.USDT,
			LastID:    "123lk4jh1234lkhj1234lkjh",
		},
	}
	for _, tc := range []struct {
		name string
		req  *UTAAffiliateKuminingRequest
	}{
		{name: "every purchase", req: &UTAAffiliateKuminingRequest{}},
		{name: "filtered", req: &UTAAffiliateKuminingRequest{
			UIDs:      []string{"100000001", "100000002"},
			StartAt:   utaAffiliateTestStart,
			EndAt:     utaAffiliateTestEnd,
			LastID:    "123lk4jh1234lkhj1234lkjg",
			Direction: "NEXT",
			PageSize:  10,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAAffiliateKumining(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAAffiliateKumining must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTAAffiliateKumining should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTAAffiliateKumining should return purchases")
		})
	}
}
