package bybit

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

func TestGetAffiliateUserList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliateUserList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAffiliateUserList must reject a nil request")
	start, end := time.Date(2025, 8, 21, 0, 0, 0, 0, time.UTC), time.Date(2025, 10, 22, 0, 0, 0, 0, time.UTC)
	_, err = e.GetAffiliateUserList(t.Context(), &AffiliateUserListRequest{StartDate: end, EndDate: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAffiliateUserList must reject a start date after the end date")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *AffiliateUserListRequest
		exp  *AffiliateUserListResponse
	}{
		{
			name: "period",
			req:  &AffiliateUserListRequest{Size: 2, Cursor: "16196", NeedDeposit: true, Need30Day: true, Need365Day: true, StartDate: start, EndDate: end},
			exp: &AffiliateUserListResponse{
				List: []AffiliateUser{
					{
						UserID:              "100000014",
						RegisterTime:        "2024-10-29",
						Source:              "Default",
						Remarks:             "vip lead",
						IsKYC:               true,
						TakerVolume30Day:    12861.362976,
						MakerVolume30Day:    262.60865,
						TradeVolume30Day:    13123.971626,
						DepositAmount30Day:  1500,
						TakerVolume365Day:   208971.63737375,
						MakerVolume365Day:   33392.64275,
						TradeVolume365Day:   242364.28012375,
						DepositAmount365Day: 25000,
						TakerVolume:         194231.4175,
						MakerVolume:         32886.108,
						TradeVolume:         227117.5255,
						StartDate:           "2025-08-21",
						EndDate:             "2025-10-22",
						TradFiTradeVolume:   1250.5,
						CommissionsVolume: map[currency.Code]types.Number{
							currency.BTC:            0,
							currency.ETH:            0,
							currency.NewCode("MNT"): 0,
							currency.USDC:           0,
							currency.USDT:           0.00835765,
						},
						Commissions30Day: map[currency.Code]types.Number{
							currency.BTC:            0,
							currency.ETH:            0,
							currency.NewCode("MNT"): 0.0621748,
							currency.USDC:           0,
							currency.USDT:           2.64288011,
						},
						Commissions365Day: map[currency.Code]types.Number{
							currency.BTC:            0.00000002,
							currency.ETH:            0.00000063,
							currency.NewCode("MNT"): 0.1210605,
							currency.USDC:           0.13462624,
							currency.USDT:           2.79509816,
						},
					},
					{
						UserID:            "100000015",
						RegisterTime:      "2023-06-28",
						Source:            "GCTREF01",
						TakerVolume365Day: 147664.35398115,
						MakerVolume365Day: 74696.351,
						TradeVolume365Day: 222360.70498115,
						TakerVolume:       30936.86124,
						MakerVolume:       36.032,
						TradeVolume:       30972.89324,
						StartDate:         "2025-08-21",
						EndDate:           "2025-10-22",
						CommissionsVolume: map[currency.Code]types.Number{
							currency.BTC:            0,
							currency.ETH:            0,
							currency.NewCode("MNT"): 2.35583601,
							currency.USDC:           0,
							currency.USDT:           3.1648939,
						},
						Commissions30Day: map[currency.Code]types.Number{
							currency.BTC:            0,
							currency.ETH:            0,
							currency.NewCode("MNT"): 0,
							currency.USDC:           0,
							currency.USDT:           0,
						},
						Commissions365Day: map[currency.Code]types.Number{
							currency.BTC:            0,
							currency.ETH:            0,
							currency.NewCode("MNT"): 8.90403178,
							currency.USDC:           0,
							currency.USDT:           23.89337689,
						},
					},
				},
				NextPageCursor: "16197",
			},
		},
		{
			name: "no period",
			req:  &AffiliateUserListRequest{Size: 1, Need30Day: true, Need365Day: true},
			exp: &AffiliateUserListResponse{
				List: []AffiliateUser{
					{
						UserID:                  "100000014",
						RegisterTime:            "2024-10-29",
						Source:                  "Default",
						Remarks:                 "vip lead",
						IsKYC:                   true,
						TakerVolume30Day:        12861.362976,
						MakerVolume30Day:        262.60865,
						TradeVolume30Day:        13123.971626,
						TakerVolume365Day:       208971.63737375,
						MakerVolume365Day:       33392.64275,
						TradeVolume365Day:       242364.28012375,
						TradFiTradeVolume30Day:  845.25,
						TradFiTradeVolume365Day: 9120.75,
						Commissions30Day: map[currency.Code]types.Number{
							currency.BTC:            0,
							currency.ETH:            0,
							currency.NewCode("MNT"): 0.0621748,
							currency.USDC:           0,
							currency.USDT:           2.64288011,
						},
						Commissions365Day: map[currency.Code]types.Number{
							currency.BTC:            0.00000002,
							currency.ETH:            0.00000063,
							currency.NewCode("MNT"): 0.1210605,
							currency.USDC:           0.13462624,
							currency.USDT:           2.79509816,
						},
					},
				},
				NextPageCursor: "16196",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAffiliateUserList(t.Context(), tc.req)
			require.NoError(t, err, "GetAffiliateUserList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAffiliateUserList should decode every field")
				return
			}
			assert.NotNil(t, result, "GetAffiliateUserList should return a result")
		})
	}
}

func TestGetAffiliateUserInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliateUserInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAffiliateUserInfo must reject a nil request")
	_, err = e.GetAffiliateUserInfo(t.Context(), &AffiliateUserInfoRequest{Coin: currency.USDT})
	require.ErrorIs(t, err, errClientUIDRequired, "GetAffiliateUserInfo must reject a missing client UID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAffiliateUserInfo(t.Context(), &AffiliateUserInfoRequest{UID: "100000013", Coin: currency.USDT, Business: "1"})
	require.NoError(t, err, "GetAffiliateUserInfo must not error")
	if mockTests {
		exp := &AffiliateUserInfoResponse{
			UID:                     "100000013",
			VIPLevel:                "5",
			TakerVolume30Day:        17061.64983,
			MakerVolume30Day:        10756.454142,
			TradeVolume30Day:        27818.103972,
			DepositAmount30Day:      2500,
			TakerVolume365Day:       1183752.53919162,
			MakerVolume365Day:       44349.42819772,
			TradeVolume365Day:       1228101.96738934,
			DepositAmount365Day:     50000,
			TotalWalletBalance:      "4",
			DepositUpdateTime:       "2026-02-04 00:00:00",
			VolumeUpdateTime:        "2026-02-04 00:00:00",
			KYCLevel:                2,
			TradFiTradeVolume30Day:  1828890.6352,
			TradFiTradeVolume365Day: 1828890.6352,
			Commissions30Day: map[currency.Code]types.Number{
				currency.BTC:            0,
				currency.ETH:            0,
				currency.NewCode("MNT"): 2.14012088,
				currency.USDC:           0,
				currency.USDT:           17.0461748,
			},
			Commissions365Day: map[currency.Code]types.Number{
				currency.BTC:            0.00012928,
				currency.ETH:            0.00099384,
				currency.NewCode("MNT"): 95.72886531,
				currency.USDC:           2.81382591,
				currency.USDT:           130.48078429,
			},
			PaySendAmount30Day:       1000,
			PayFirstTimeTransaction:  100,
			CardFirstTimeTransaction: 50,
		}
		assert.Equal(t, exp, result, "GetAffiliateUserInfo should decode every field")
		return
	}
	assert.Equal(t, "100000013", result.UID, "GetAffiliateUserInfo should return the client")
}

func TestGetAffiliateSubAffiliateList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAffiliateSubAffiliateList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAffiliateSubAffiliateList must reject a nil request")
	start, end := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	_, err = e.GetAffiliateSubAffiliateList(t.Context(), &AffiliateSubAffiliateListRequest{StartDate: start})
	require.ErrorIs(t, err, errDateRangeIncomplete, "GetAffiliateSubAffiliateList must reject a start date without an end date")
	_, err = e.GetAffiliateSubAffiliateList(t.Context(), &AffiliateSubAffiliateListRequest{EndDate: end})
	require.ErrorIs(t, err, errDateRangeIncomplete, "GetAffiliateSubAffiliateList must reject an end date without a start date")
	_, err = e.GetAffiliateSubAffiliateList(t.Context(), &AffiliateSubAffiliateListRequest{StartDate: end, EndDate: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAffiliateSubAffiliateList must reject a start date after the end date")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAffiliateSubAffiliateList(t.Context(), &AffiliateSubAffiliateListRequest{
		Cursor:         "eyJsYXN0SWQiOiIxMDAwMCJ9",
		Size:           1,
		StartDate:      start,
		EndDate:        end,
		SubAffiliateID: 10001,
	})
	require.NoError(t, err, "GetAffiliateSubAffiliateList must not error")
	if mockTests {
		exp := &AffiliateSubAffiliateListResponse{
			List: []SubAffiliate{
				{
					SubAffiliateID: "10001",
					UserID:         "100000016",
					Name:           "sub_affiliate_nick",
					Email:          "sub***@example.com",
					CommissionsVolume: map[currency.Code]types.Number{
						currency.BTC:            0.005,
						currency.ETH:            0.2,
						currency.NewCode("MNT"): 40,
						currency.USDC:           100,
						currency.USDT:           400,
					},
					CommissionsForUSDT:  550,
					BecameAffiliateTime: types.DateTime(time.Date(2023, 6, 1, 10, 0, 0, 0, time.UTC)),
					StartDate:           types.DateTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
					EndDate:             types.DateTime(time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)),
				},
			},
			NextPageCursor: "eyJsYXN0SWQiOiIxMDAwMSJ9",
		}
		assert.Equal(t, exp, result, "GetAffiliateSubAffiliateList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAffiliateSubAffiliateList should return a result")
}
