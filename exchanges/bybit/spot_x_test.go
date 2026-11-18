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

func TestGetLaunchpoolProjectList(t *testing.T) {
	t.Parallel()
	_, err := e.GetLaunchpoolProjectList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLaunchpoolProjectList must reject a nil request")

	result, err := e.GetLaunchpoolProjectList(t.Context(), &SpotXProjectListRequest{Status: 2, Limit: 2})
	require.NoError(t, err, "GetLaunchpoolProjectList must not error")
	if mockTests {
		exp := &LaunchpoolProjectListResponse{
			List: []LaunchpoolProject{
				{
					ProjectID:             "20260327073057",
					ActivityCoin:          currency.NewCode("BASED"),
					RewardCoin:            currency.NewCode("BASED"),
					Status:                2,
					TotalReward:           3000000,
					StakeStartTime:        types.Time(time.Unix(1774864800, 0)),
					StakeEndTime:          types.Time(time.Unix(1775469600, 0)),
					StakeStartTimeUTC0:    "2026-03-30 10:00:00",
					StakeEndTimeUTC0:      "2026-04-06 10:00:00",
					TotalParticipantCount: 25281,
					Pools: []LaunchpoolPool{
						{
							PoolID:            "202603270730571",
							StakeCoin:         currency.NewCode("BASED"),
							APR:               215.39047045,
							TotalStakedAmount: 19064213.89623133,
							ParticipantCount:  2615,
						},
						{
							PoolID:            "202603270730572",
							StakeCoin:         currency.NewCode("MNT"),
							APR:               13.31002752,
							TotalStakedAmount: 59895661.45836238,
							ParticipantCount:  22666,
						},
					},
				},
				{
					ProjectID:             "20260130095445",
					ActivityCoin:          currency.NewCode("ZAMA"),
					RewardCoin:            currency.NewCode("ZAMA"),
					Status:                2,
					TotalReward:           12000000,
					StakeStartTime:        types.Time(time.Unix(1770019200, 0)),
					StakeEndTime:          types.Time(time.Unix(1770544800, 0)),
					StakeStartTimeUTC0:    "2026-02-02 08:00:00",
					StakeEndTimeUTC0:      "2026-02-08 10:00:00",
					TotalParticipantCount: 36230,
					Pools: []LaunchpoolPool{
						{
							PoolID:            "202601300954452",
							StakeCoin:         currency.NewCode("MNT"),
							APR:               37.4328208,
							TotalStakedAmount: 52359693.1301973,
							ParticipantCount:  33561,
						},
						{
							PoolID:            "202601300954451",
							StakeCoin:         currency.NewCode("ZAMA"),
							APR:               186.87651818,
							TotalStakedAmount: 83706916.70862648,
							ParticipantCount:  2669,
						},
					},
				},
			},
			NextPageCursor: "eyJpZCI6MTEyfQ",
		}
		assert.Equal(t, exp, result, "GetLaunchpoolProjectList should decode every field")
	}
	require.NotEmpty(t, result.List, "GetLaunchpoolProjectList must return projects")

	// Launchpool matches the activity coin filter against the reward coin
	single, err := e.GetLaunchpoolProjectList(t.Context(), &SpotXProjectListRequest{Status: 2, ProjectID: result.List[0].ProjectID, ActivityCoin: result.List[0].RewardCoin})
	require.NoError(t, err, "GetLaunchpoolProjectList must not error for one project")
	require.Len(t, single.List, 1, "GetLaunchpoolProjectList must return one project")
	assert.Equal(t, result.List[0], single.List[0], "GetLaunchpoolProjectList should return the listed project")

	require.NotEmpty(t, result.NextPageCursor, "GetLaunchpoolProjectList must return a cursor while more projects exist")
	next, err := e.GetLaunchpoolProjectList(t.Context(), &SpotXProjectListRequest{Status: 2, Limit: 2, Cursor: result.NextPageCursor})
	require.NoError(t, err, "GetLaunchpoolProjectList must not error for the next page")
	require.NotEmpty(t, next.List, "GetLaunchpoolProjectList must return the next page")
	assert.NotEqual(t, result.List[0].ProjectID, next.List[0].ProjectID, "GetLaunchpoolProjectList should return other projects on the next page")
}

func TestGetLaunchpoolUserActivityLog(t *testing.T) {
	t.Parallel()
	_, err := e.GetLaunchpoolUserActivityLog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLaunchpoolUserActivityLog must reject a nil request")
	_, err = e.GetLaunchpoolUserActivityLog(t.Context(), &LaunchpoolUserActivityLogRequest{StartTime: time.UnixMilli(1704067200000)})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetLaunchpoolUserActivityLog must reject a start time without an end time")
	_, err = e.GetLaunchpoolUserActivityLog(t.Context(), &LaunchpoolUserActivityLogRequest{StartTime: time.UnixMilli(1705000000000), EndTime: time.UnixMilli(1704067200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLaunchpoolUserActivityLog must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	pledge, success := uint64(0), uint64(1)
	for _, tc := range []struct {
		name string
		req  *LaunchpoolUserActivityLogRequest
		exp  *LaunchpoolUserActivityLogResponse
	}{
		{
			name: "successful stakes",
			req:  &LaunchpoolUserActivityLogRequest{Type: &pledge, Status: &success},
			exp: &LaunchpoolUserActivityLogResponse{
				List: []LaunchpoolActivity{
					{
						StakeCoin:   currency.USDT,
						RewardCoin:  currency.BTC,
						Amount:      1000,
						Status:      1,
						CreatedTime: types.Time(time.Unix(1704067200, 0)),
						OrderID:     "ORDER_123456",
					},
				},
				Total:    1,
				PageSize: 10,
				Current:  1,
			},
		},
		{
			name: "window",
			req:  &LaunchpoolUserActivityLogRequest{StakeCoin: currency.USDT, StartTime: time.UnixMilli(1704067200000), EndTime: time.UnixMilli(1705000000000), PageSize: 10, Current: 1},
			exp: &LaunchpoolUserActivityLogResponse{
				List: []LaunchpoolActivity{
					{
						StakeCoin:   currency.USDT,
						RewardCoin:  currency.BTC,
						Amount:      1000,
						Status:      1,
						CreatedTime: types.Time(time.Unix(1704067200, 0)),
						OrderID:     "ORDER_123456",
					},
					{
						StakeCoin:   currency.USDT,
						RewardCoin:  currency.BTC,
						Amount:      0.00012,
						Type:        2,
						CreatedTime: types.Time(time.Unix(1704153600, 0)),
						OrderID:     "ORDER_123789",
					},
				},
				Total:    25,
				PageSize: 10,
				Current:  1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetLaunchpoolUserActivityLog(t.Context(), tc.req)
			require.NoError(t, err, "GetLaunchpoolUserActivityLog must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetLaunchpoolUserActivityLog should decode every field")
				return
			}
			assert.NotNil(t, result, "GetLaunchpoolUserActivityLog should return a result")
		})
	}
}

func TestGetLaunchpoolCurrentStaking(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLaunchpoolCurrentStaking(t.Context())
	require.NoError(t, err, "GetLaunchpoolCurrentStaking must not error")
	if mockTests {
		exp := &LaunchpoolCurrentStakingResponse{
			TotalInvestmentUSD: 15000,
			TotalEarningsUSD:   120.5,
			TodayEarningsUSD:   8.2,
			List: []LaunchpoolStakingPosition{
				{
					StakeCoin:      currency.USDT,
					RewardCoin:     currency.BTC,
					StakeAmount:    10000,
					TotalReward:    0.0012,
					AutoRedeemDate: types.Time(time.Unix(1706745600, 0)),
				},
				{
					StakeCoin:   currency.NewCode("MNT"),
					RewardCoin:  currency.BTC,
					StakeAmount: 50000,
					TotalReward: 0.0005,
				},
			},
		}
		assert.Equal(t, exp, result, "GetLaunchpoolCurrentStaking should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLaunchpoolCurrentStaking should return a result")
}

func TestGetLaunchpoolUserHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLaunchpoolUserHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLaunchpoolUserHistory must reject a nil request")
	_, err = e.GetLaunchpoolUserHistory(t.Context(), &LaunchpoolUserHistoryRequest{EndTime: time.UnixMilli(1706745600000)})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetLaunchpoolUserHistory must reject an end time without a start time")
	_, err = e.GetLaunchpoolUserHistory(t.Context(), &LaunchpoolUserHistoryRequest{StartTime: time.UnixMilli(1706745600000), EndTime: time.UnixMilli(1701388800000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLaunchpoolUserHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLaunchpoolUserHistory(t.Context(), &LaunchpoolUserHistoryRequest{
		StakeCoin:  currency.USDT,
		RewardCoin: currency.BTC,
		StartTime:  time.UnixMilli(1701388800000),
		EndTime:    time.UnixMilli(1706745600000),
		PageSize:   10,
		Current:    1,
	})
	require.NoError(t, err, "GetLaunchpoolUserHistory must not error")
	if mockTests {
		exp := &LaunchpoolUserHistoryResponse{
			List: []LaunchpoolHistoricalPosition{
				{
					StakeCoin:      currency.USDT,
					RewardCoin:     currency.BTC,
					TotalReward:    0.0025,
					StakeStartTime: types.Time(time.Unix(1704067200, 0)),
					StakeEndTime:   types.Time(time.Unix(1706745600, 0)),
				},
				{
					StakeCoin:      currency.USDT,
					RewardCoin:     currency.BTC,
					TotalReward:    0.0008,
					StakeStartTime: types.Time(time.Unix(1701388800, 0)),
					StakeEndTime:   types.Time(time.Unix(1704067200, 0)),
				},
			},
			Total:    12,
			PageSize: 10,
			Current:  1,
		}
		assert.Equal(t, exp, result, "GetLaunchpoolUserHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLaunchpoolUserHistory should return a result")
}

func TestGetPuzzleProjectList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPuzzleProjectList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPuzzleProjectList must reject a nil request")

	result, err := e.GetPuzzleProjectList(t.Context(), &SpotXProjectListRequest{Status: 2, Limit: 2})
	require.NoError(t, err, "GetPuzzleProjectList must not error")
	if mockTests {
		exp := &PuzzleProjectListResponse{
			List: []PuzzleProject{
				{
					ProjectID:        "9739263906356",
					ActivityCoin:     currency.NewCode("NVDAX"),
					RewardCoin:       currency.USDT,
					Status:           2,
					StartTime:        types.Time(time.Unix(1788523200, 0)),
					EndTime:          types.Time(time.Unix(1790769600, 0)),
					StartTimeUTC0:    "2026-09-04 12:00:00",
					EndTimeUTC0:      "2026-09-30 12:00:00",
					TotalReward:      100000,
					ParticipantCount: 44667,
				},
				{
					ProjectID:        "9002488396764",
					ActivityCoin:     currency.NewCode("CAP"),
					RewardCoin:       currency.NewCode("CAP"),
					Status:           2,
					StartTime:        types.Time(time.Unix(1784023200, 0)),
					EndTime:          types.Time(time.Unix(1785146400, 0)),
					StartTimeUTC0:    "2026-07-14 10:00:00",
					EndTimeUTC0:      "2026-07-27 10:00:00",
					TotalReward:      15000000,
					ParticipantCount: 34382,
				},
			},
			NextPageCursor: "eyJpZCI6MTQ0fQ",
		}
		assert.Equal(t, exp, result, "GetPuzzleProjectList should decode every field")
	}
	require.NotEmpty(t, result.List, "GetPuzzleProjectList must return projects")

	single, err := e.GetPuzzleProjectList(t.Context(), &SpotXProjectListRequest{Status: 2, ProjectID: result.List[0].ProjectID, ActivityCoin: result.List[0].ActivityCoin})
	require.NoError(t, err, "GetPuzzleProjectList must not error for one project")
	require.Len(t, single.List, 1, "GetPuzzleProjectList must return one project")
	assert.Equal(t, result.List[0], single.List[0], "GetPuzzleProjectList should return the listed project")

	require.NotEmpty(t, result.NextPageCursor, "GetPuzzleProjectList must return a cursor while more projects exist")
	next, err := e.GetPuzzleProjectList(t.Context(), &SpotXProjectListRequest{Status: 2, Limit: 2, Cursor: result.NextPageCursor})
	require.NoError(t, err, "GetPuzzleProjectList must not error for the next page")
	require.NotEmpty(t, next.List, "GetPuzzleProjectList must return the next page")
	assert.NotEqual(t, result.List[0].ProjectID, next.List[0].ProjectID, "GetPuzzleProjectList should return other projects on the next page")
}

func TestGetTokenSplashProjectList(t *testing.T) {
	t.Parallel()
	_, err := e.GetTokenSplashProjectList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTokenSplashProjectList must reject a nil request")

	result, err := e.GetTokenSplashProjectList(t.Context(), &SpotXProjectListRequest{Status: 2, Limit: 2})
	require.NoError(t, err, "GetTokenSplashProjectList must not error")
	if mockTests {
		exp := &TokenSplashProjectListResponse{
			List: []TokenSplashProject{
				{
					Status:                    2,
					ProjectID:                 "20260924122414",
					ActivityCoin:              currency.NewCode("KII"),
					RewardCoin:                currency.NewCode("KII"),
					TotalReward:               2000000,
					ParticipantCount:          4966,
					RegistrationStartTime:     types.Time(time.Unix(1790330400, 0)),
					ActivityEndTime:           types.Time(time.Unix(1791198000, 0)),
					RegistrationStartTimeUTC0: "2026-09-25 10:00:00 UTC",
					ActivityEndTimeUTC0:       "2026-10-05 11:00:00 UTC",
				},
				{
					Status:                    2,
					ProjectID:                 "20260924054648",
					ActivityCoin:              currency.USDC,
					RewardCoin:                currency.USDC,
					TotalReward:               100000,
					ParticipantCount:          2005,
					RegistrationStartTime:     types.Time(time.Unix(1790244000, 0)),
					ActivityEndTime:           types.Time(time.Unix(1790852400, 0)),
					RegistrationStartTimeUTC0: "2026-09-24 10:00:00 UTC",
					ActivityEndTimeUTC0:       "2026-10-01 11:00:00 UTC",
				},
			},
			NextPageCursor: "eyJpZCI6NTY0fQ",
		}
		assert.Equal(t, exp, result, "GetTokenSplashProjectList should decode every field")
	}
	require.NotEmpty(t, result.List, "GetTokenSplashProjectList must return projects")

	single, err := e.GetTokenSplashProjectList(t.Context(), &SpotXProjectListRequest{Status: 2, ProjectID: result.List[0].ProjectID, ActivityCoin: result.List[0].ActivityCoin})
	require.NoError(t, err, "GetTokenSplashProjectList must not error for one project")
	require.Len(t, single.List, 1, "GetTokenSplashProjectList must return one project")
	assert.Equal(t, result.List[0], single.List[0], "GetTokenSplashProjectList should return the listed project")

	require.NotEmpty(t, result.NextPageCursor, "GetTokenSplashProjectList must return a cursor while more projects exist")
	next, err := e.GetTokenSplashProjectList(t.Context(), &SpotXProjectListRequest{Status: 2, Limit: 2, Cursor: result.NextPageCursor})
	require.NoError(t, err, "GetTokenSplashProjectList must not error for the next page")
	require.NotEmpty(t, next.List, "GetTokenSplashProjectList must return the next page")
	assert.NotEqual(t, result.List[0].ProjectID, next.List[0].ProjectID, "GetTokenSplashProjectList should return other projects on the next page")
}

func TestGetTokenSplashUserActivityParams(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTokenSplashUserActivityParams(t.Context(), "TOKENSPLASH_BTC_2024Q1", currency.BTC)
	require.NoError(t, err, "GetTokenSplashUserActivityParams must not error")
	if mockTests {
		exp := &TokenSplashUserActivityParamsResponse{
			List: []TokenSplashUserActivity{
				{
					ProjectID:    "TOKENSPLASH_BTC_2024Q1",
					ActivityCoin: currency.BTC,
					TradeTask: TokenSplashTradeTask{
						TradeRequiredAmount:   0.5,
						TradeUnit:             currency.BTC,
						TradeToken:            "BTC",
						TradedAmount:          0.3,
						MaxRewardAmount:       200,
						EstimatedRewardAmount: 120,
						RewardCoin:            currency.USDT,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetTokenSplashUserActivityParams should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTokenSplashUserActivityParams should return a result")
}
