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

// affiliateCommissionTestCase is a GetAffiliateCommission request and the page it should return
type affiliateCommissionTestCase struct {
	req *AffiliateCommissionRequest
	exp *AffiliateCommissionResponse
}

func TestGetAffiliateInvitees(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliateInvitees(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAffiliateInvitees must reject a nil request")
	start, end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 7, 31, 0, 0, 0, 0, time.UTC)
	_, err = e.GetAffiliateInvitees(t.Context(), &AffiliateInviteesRequest{RegistrationStartAt: end, RegistrationEndAt: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAffiliateInvitees must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAffiliateInvitees(t.Context(), &AffiliateInviteesRequest{UserType: "1", ReferralCode: "GCTREF01", UIDs: []string{"100000011", "100000012"}, RegistrationStartAt: start, RegistrationEndAt: end, Page: 1, PageSize: 10})
	require.NoError(t, err, "GetAffiliateInvitees must not error")
	if mockTests {
		exp := &AffiliateInviteesResponse{
			CurrentPage: 1,
			PageSize:    10,
			TotalNumber: 2,
			TotalPage:   1,
			Items: []AffiliateInvitee{
				{
					UID:              "100000011",
					NickName:         "ex**@**.com",
					ReferralCode:     "GCTREF01",
					Country:          "United States of America",
					RegistrationTime: types.Time(time.UnixMilli(1752737999000)),
					MyCommissionRate: 29,
					CashbackRate:     11,
					Currency:         currency.USDT,
				},
				{
					UID:                   "100000012",
					NickName:              "us**@**.com",
					ReferralCode:          "GCTREF01",
					Country:               "Nigeria",
					RegistrationTime:      types.Time(time.UnixMilli(1750245613000)),
					CompletedKYC:          true,
					CompletedFirstDeposit: true,
					CompletedFirstTrade:   true,
					Past7DayFees:          12.45,
					Past7DayCommission:    3.61,
					TotalCommission:       27.83,
					MyCommissionRate:      29,
					CashbackRate:          11,
					Currency:              currency.USDT,
				},
			},
		}
		assert.Equal(t, exp, result, "GetAffiliateInvitees should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAffiliateInvitees should return a page")
}

func TestGetAffiliateCommission(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliateCommission(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAffiliateCommission must reject a nil request")
	start, end := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	_, err = e.GetAffiliateCommission(t.Context(), &AffiliateCommissionRequest{RebateStartAt: end, RebateEndAt: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAffiliateCommission must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for name, tc := range map[string]affiliateCommissionTestCase{
		"trade": {
			req: &AffiliateCommissionRequest{SiteType: "all", RebateType: 1, RebateStartAt: start, RebateEndAt: end, Page: 1, PageSize: 10, UserID: "100000011", DataType: "trade"},
			exp: &AffiliateCommissionResponse{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []AffiliateCommission{
					{
						SiteType:        "global",
						RebateType:      1,
						PayoutTime:      types.Time(time.UnixMilli(1786032000000)),
						PeriodStartTime: types.Time(time.UnixMilli(1785945600000)),
						PeriodEndTime:   types.Time(time.UnixMilli(1786031999999)),
						Status:          3,
						TakerVolume:     0.39,
						MakerVolume:     1.25,
						Commission:      0.0021,
						Currency:        currency.USDT,
						DataType:        "trade",
					},
				},
			},
		},
		"kumining": {
			req: &AffiliateCommissionRequest{DataType: "kumining"},
			exp: &AffiliateCommissionResponse{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []AffiliateCommission{
					{
						SiteType:        "global",
						RebateType:      1,
						PayoutTime:      types.Time(time.UnixMilli(1786032000000)),
						PeriodStartTime: types.Time(time.UnixMilli(1785945600000)),
						PeriodEndTime:   types.Time(time.UnixMilli(1786031999999)),
						Status:          3,
						Commission:      1.21,
						Currency:        currency.USDT,
						KuminingVolume:  120.5,
						DataType:        "kumining",
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAffiliateCommission(t.Context(), tc.req)
			require.NoError(t, err, "GetAffiliateCommission must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAffiliateCommission should decode every field")
				return
			}
			assert.NotNil(t, result, "GetAffiliateCommission should return a page")
		})
	}
}

func TestGetAffiliateTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliateTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAffiliateTradeHistory must reject a nil request")
	_, err = e.GetAffiliateTradeHistory(t.Context(), &AffiliateTradeHistoryRequest{TradeType: "SPOT"})
	require.ErrorIs(t, err, errUIDRequired, "GetAffiliateTradeHistory must reject an empty UID")
	start, end := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 7, 31, 0, 0, 0, 0, time.UTC)
	_, err = e.GetAffiliateTradeHistory(t.Context(), &AffiliateTradeHistoryRequest{UID: "100000011", TradeStartAt: end, TradeEndAt: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAffiliateTradeHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAffiliateTradeHistory(t.Context(), &AffiliateTradeHistoryRequest{UID: "100000011", TradeType: "SPOT", TradeStartAt: start, TradeEndAt: end, Page: 1, PageSize: 10})
	require.NoError(t, err, "GetAffiliateTradeHistory must not error")
	if mockTests {
		exp := &AffiliateTradeHistoryResponse{
			CurrentPage: 1,
			PageSize:    10,
			TotalNumber: 2,
			TotalPage:   1,
			Items: []AffiliateTrade{
				{
					TradeTime:            types.Time(time.UnixMilli(1752560532000)),
					TradeType:            "SPOT",
					TradeCurrency:        currency.TRUMP,
					TradeAmount:          4.96,
					ConvertedTradeAmount: 4.96,
					ConvertedFee:         0.00496,
					Commission:           0.0014384,
					Currency:             currency.USDT,
				},
				{
					TradeTime:            types.Time(time.UnixMilli(1752204768000)),
					TradeType:            "SPOT",
					TradeCurrency:        currency.DOGE,
					TradeAmount:          1.98,
					ConvertedTradeAmount: 1.98,
					ConvertedFee:         0.00198,
					Commission:           0.0005742,
					Currency:             currency.USDT,
				},
			},
		}
		assert.Equal(t, exp, result, "GetAffiliateTradeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAffiliateTradeHistory should return a page")
}

func TestGetAffiliateTransactions(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliateTransactions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAffiliateTransactions must reject a nil request")
	start, end := time.Date(2020, 12, 31, 16, 0, 0, 0, time.UTC), time.Date(2021, 1, 1, 16, 0, 0, 0, time.UTC)
	_, err = e.GetAffiliateTransactions(t.Context(), &AffiliateTransactionsRequest{TradeEndAt: end})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetAffiliateTransactions must reject a missing start time")
	_, err = e.GetAffiliateTransactions(t.Context(), &AffiliateTransactionsRequest{TradeStartAt: end, TradeEndAt: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAffiliateTransactions must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAffiliateTransactions(t.Context(), &AffiliateTransactionsRequest{UID: "100000011", TradeType: "SPOT", TradeStartAt: start, TradeEndAt: end, LastID: "6498040334d4050001b0aa7b", Direction: "NEXT", PageSize: 10})
	require.NoError(t, err, "GetAffiliateTransactions must not error")
	if mockTests {
		exp := []AffiliateTransactionResponse{
			{
				UID:                  "100000011",
				TradeTime:            types.Time(time.UnixMilli(1609516800000)),
				TradeType:            "SPOT",
				TradeCurrency:        currency.BTC,
				TradeAmount:          0.1,
				ConvertedTradeAmount: 10000,
				ConvertedFee:         2.82,
				Commission:           1.41,
				Currency:             currency.USDT,
				LastID:               "6498040334d4050001b0aa7c",
			},
		}
		assert.Equal(t, exp, result, "GetAffiliateTransactions should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAffiliateTransactions should return a transaction list")
}

func TestGetAffiliateKumining(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliateKumining(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAffiliateKumining must reject a nil request")
	start, end := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	_, err = e.GetAffiliateKumining(t.Context(), &AffiliateKuminingRequest{StartAt: end, EndAt: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAffiliateKumining must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAffiliateKumining(t.Context(), &AffiliateKuminingRequest{UID: "100000011", StartAt: start, EndAt: end, LastID: "123lk4jh1234lkhj1234lkjg", Direction: "NEXT", PageSize: 10})
	require.NoError(t, err, "GetAffiliateKumining must not error")
	if mockTests {
		exp := []AffiliateKuminingResponse{
			{
				UID:       "100000011",
				GoodsName: "BTC-10TH",
				Amount:    0.7,
				Currency:  currency.USDT,
				LastID:    "123lk4jh1234lkhj1234lkjh",
				PayTime:   types.Time(time.UnixMilli(1762846542000)),
			},
		}
		assert.Equal(t, exp, result, "GetAffiliateKumining should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAffiliateKumining should return a record list")
}
