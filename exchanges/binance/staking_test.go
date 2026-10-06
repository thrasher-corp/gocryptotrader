package binance

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

func TestGetETHStakingAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetETHStakingAccount(t.Context())
	require.NoError(t, err, "GetETHStakingAccount must not error")
	if mockTests {
		exp := &ETHStakingAccountResponse{
			HoldingInETH: 1.22330928,
			Holdings: ETHStakingHoldings{
				WBETHAmount: 1.10928781,
				BETHAmount:  0.09002112,
			},
			ThirtyDaysProfitInETH: 0.22330928,
			Profit: ETHStakingProfit{
				AmountFromWBETH: 0.12330928,
				AmountFromBETH:  0.1,
			},
		}
		assert.Equal(t, exp, result, "GetETHStakingAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetETHStakingAccount should return a result")
}

func TestGetBETHRewardsDistributionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBETHRewardsDistributionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBETHRewardsDistributionHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetBETHRewardsDistributionHistory(t.Context(), &EarnHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBETHRewardsDistributionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBETHRewardsDistributionHistory(t.Context(), &EarnHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetBETHRewardsDistributionHistory must not error")
	if mockTests {
		exp := &BETHRewardsDistributionHistoryResponse{
			Rows: []BETHRewardsDistribution{
				{
					Time:                 types.Time(time.Unix(1744156800, 0)),
					Asset:                currency.BETH,
					Holding:              2.3223,
					Amount:               0.23223,
					AnnualPercentageRate: 0.5,
					Status:               "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetBETHRewardsDistributionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBETHRewardsDistributionHistory should return a result")
}

func TestGetCurrentETHStakingQuota(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrentETHStakingQuota(t.Context())
	require.NoError(t, err, "GetCurrentETHStakingQuota must not error")
	if mockTests {
		exp := &ETHStakingQuotaResponse{
			LeftStakingPersonalQuota:    1000,
			LeftRedemptionPersonalQuota: 1000,
			MinimumStakeAmount:          0.0001,
			MinimumRedeemAmount:         0.00000001,
			RedeemPeriod:                20,
			Stakeable:                   true,
			Redeemable:                  true,
			CommissionFee:               0.05,
			Calculating:                 true,
		}
		assert.Equal(t, exp, result, "GetCurrentETHStakingQuota should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrentETHStakingQuota should return a result")
}

func TestGetETHRedemptionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetETHRedemptionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetETHRedemptionHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetETHRedemptionHistory(t.Context(), &StakingRedemptionHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetETHRedemptionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetETHRedemptionHistory(t.Context(), &StakingRedemptionHistoryRequest{RedeemID: 1234567, StartTime: startTime, EndTime: endTime, Size: 10})
	require.NoError(t, err, "GetETHRedemptionHistory must not error")
	if mockTests {
		exp := &ETHRedemptionHistoryResponse{
			Rows: []ETHRedemption{
				{
					Time:             types.Time(time.Unix(1744156800, 0)),
					ArrivalTime:      types.Time(time.Unix(1745884800, 0)),
					Asset:            currency.NewCode("WBETH"),
					Amount:           1.5,
					DistributeAsset:  currency.ETH,
					DistributeAmount: 1.6013,
					ConversionRatio:  1.06753333,
					Status:           "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetETHRedemptionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetETHRedemptionHistory should return a result")
}

func TestGetETHStakingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetETHStakingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetETHStakingHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetETHStakingHistory(t.Context(), &StakingSubscriptionHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetETHStakingHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetETHStakingHistory(t.Context(), &StakingSubscriptionHistoryRequest{PurchaseID: 1234567, StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetETHStakingHistory must not error")
	if mockTests {
		exp := &ETHStakingHistoryResponse{
			Rows: []ETHStakingSubscription{
				{
					Time:             types.Time(time.Unix(1744156800, 0)),
					Asset:            currency.ETH,
					Amount:           1.6013,
					DistributeAsset:  currency.NewCode("WBETH"),
					DistributeAmount: 1.5,
					ConversionRatio:  1.06753333,
					Status:           "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetETHStakingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetETHStakingHistory should return a result")
}

func TestGetWBETHRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetWBETHRateHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWBETHRateHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetWBETHRateHistory(t.Context(), &EarnHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWBETHRateHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWBETHRateHistory(t.Context(), &EarnHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetWBETHRateHistory must not error")
	if mockTests {
		exp := &WBETHRateHistoryResponse{
			Rows: []WBETHRate{
				{
					AnnualPercentageRate: 0.0285,
					ExchangeRate:         1.06753333,
					Time:                 types.Time(time.Unix(1744156800, 0)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetWBETHRateHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetWBETHRateHistory should return a result")
}

func TestGetWBETHRewardHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetWBETHRewardHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWBETHRewardHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetWBETHRewardHistory(t.Context(), &EarnHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWBETHRewardHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWBETHRewardHistory(t.Context(), &EarnHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 10})
	require.NoError(t, err, "GetWBETHRewardHistory must not error")
	if mockTests {
		exp := &WBETHRewardHistoryResponse{
			EstimatedRewardsInETH: 1.2323092,
			Rows: []WBETHReward{
				{
					Time:                 types.Time(time.Unix(1744156800, 0)),
					AmountInETH:          0.00023223,
					Holding:              2.3223,
					HoldingInETH:         2.4791,
					AnnualPercentageRate: 0.0285,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetWBETHRewardHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetWBETHRewardHistory should return a result")
}

func TestGetWBETHUnwrapHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetWBETHUnwrapHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWBETHUnwrapHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetWBETHUnwrapHistory(t.Context(), &EarnHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWBETHUnwrapHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWBETHUnwrapHistory(t.Context(), &EarnHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 10})
	require.NoError(t, err, "GetWBETHUnwrapHistory must not error")
	if mockTests {
		exp := &WBETHWrapHistoryResponse{
			Rows: []WBETHWrap{
				{
					Time:         types.Time(time.Unix(1744156800, 0)),
					FromAsset:    currency.NewCode("WBETH"),
					FromAmount:   1,
					ToAsset:      currency.BETH,
					ToAmount:     1.06753333,
					ExchangeRate: 1.06753333,
					Status:       "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetWBETHUnwrapHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetWBETHUnwrapHistory should return a result")
}

func TestGetWBETHWrapHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetWBETHWrapHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWBETHWrapHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetWBETHWrapHistory(t.Context(), &EarnHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWBETHWrapHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWBETHWrapHistory(t.Context(), &EarnHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetWBETHWrapHistory must not error")
	if mockTests {
		exp := &WBETHWrapHistoryResponse{
			Rows: []WBETHWrap{
				{
					Time:         types.Time(time.Unix(1744156800, 0)),
					FromAsset:    currency.BETH,
					FromAmount:   1.06753333,
					ToAsset:      currency.NewCode("WBETH"),
					ToAmount:     1,
					ExchangeRate: 1.06753333,
					Status:       "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetWBETHWrapHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetWBETHWrapHistory should return a result")
}

func TestRedeemETH(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemETH(t.Context(), 0, currency.NewCode("WBETH"))
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RedeemETH must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RedeemETH(t.Context(), 0.123, currency.NewCode("WBETH"))
	require.NoError(t, err, "RedeemETH must not error")
	if mockTests {
		exp := &ETHRedemptionResponse{
			Success:         true,
			ETHAmount:       0.1313066,
			RedeemID:        1234567,
			ConversionRatio: 1.06753333,
			ArrivalTime:     types.Time(time.Unix(1745884800, 0)),
		}
		assert.Equal(t, exp, result, "RedeemETH should decode every field")
		return
	}
	assert.True(t, result.Success, "RedeemETH should succeed")
}

func TestSubscribeETHStaking(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeETHStaking(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubscribeETHStaking must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubscribeETHStaking(t.Context(), 0.123)
	require.NoError(t, err, "SubscribeETHStaking must not error")
	if mockTests {
		exp := &ETHStakingSubscriptionResponse{
			Success:         true,
			WBETHAmount:     0.1152189,
			PurchaseID:      1234567,
			ConversionRatio: 1.06753333,
		}
		assert.Equal(t, exp, result, "SubscribeETHStaking should decode every field")
		return
	}
	assert.True(t, result.Success, "SubscribeETHStaking should succeed")
}

func TestWrapBETH(t *testing.T) {
	t.Parallel()
	_, err := e.WrapBETH(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "WrapBETH must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WrapBETH(t.Context(), 0.001)
	require.NoError(t, err, "WrapBETH must not error")
	if mockTests {
		exp := &WrapBETHResponse{
			Success:      true,
			WBETHAmount:  0.00093674,
			ExchangeRate: 1.06753333,
		}
		assert.Equal(t, exp, result, "WrapBETH should decode every field")
		return
	}
	assert.True(t, result.Success, "WrapBETH should succeed")
}

func TestClaimBoostRewards(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ClaimBoostRewards(t.Context())
	require.NoError(t, err, "ClaimBoostRewards must not error")
	if mockTests {
		exp := &EarnSuccessResponse{
			Success: true,
		}
		assert.Equal(t, exp, result, "ClaimBoostRewards should decode every field")
		return
	}
	assert.True(t, result.Success, "ClaimBoostRewards should succeed")
}

func TestGetBNSOLRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBNSOLRateHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBNSOLRateHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetBNSOLRateHistory(t.Context(), &EarnHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBNSOLRateHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBNSOLRateHistory(t.Context(), &EarnHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetBNSOLRateHistory must not error")
	if mockTests {
		exp := &BNSOLRateHistoryResponse{
			Rows: []BNSOLRate{
				{
					AnnualPercentageRate: 0.0712,
					ExchangeRate:         1.05923481,
					BoostRewards: []BNSOLBoostReward{
						{
							BoostAPR:     0.12,
							RewardsAsset: currency.SOL,
						},
					},
					Time: types.Time(time.Unix(1744156800, 0)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetBNSOLRateHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBNSOLRateHistory should return a result")
}

func TestGetBNSOLRewardsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBNSOLRewardsHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBNSOLRewardsHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetBNSOLRewardsHistory(t.Context(), &EarnHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBNSOLRewardsHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBNSOLRewardsHistory(t.Context(), &EarnHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetBNSOLRewardsHistory must not error")
	if mockTests {
		exp := &BNSOLRewardsHistoryResponse{
			EstimatedRewardsInSOL: 1.2323092,
			Rows: []BNSOLReward{
				{
					Time:                 types.Time(time.Unix(1744156800, 0)),
					AmountInSOL:          0.00045223,
					Holding:              2.3223,
					HoldingInSOL:         2.4599,
					AnnualPercentageRate: 0.0712,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetBNSOLRewardsHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBNSOLRewardsHistory should return a result")
}

func TestGetBoostRewardsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBoostRewardsHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBoostRewardsHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetBoostRewardsHistory(t.Context(), &BoostRewardsHistoryRequest{StartTime: startTime, EndTime: endTime})
	require.ErrorIs(t, err, errRewardTypeMissing, "GetBoostRewardsHistory must reject an empty type")
	_, err = e.GetBoostRewardsHistory(t.Context(), &BoostRewardsHistoryRequest{Type: "CLAIM", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBoostRewardsHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBoostRewardsHistory(t.Context(), &BoostRewardsHistoryRequest{Type: "CLAIM", StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetBoostRewardsHistory must not error")
	if mockTests {
		exp := &BoostRewardsHistoryResponse{
			Rows: []BoostReward{
				{
					Time:         types.Time(time.Unix(1744156800, 0)),
					Token:        currency.SOL,
					Amount:       1.20291028,
					BNSOLHolding: 2.0928798,
					Status:       "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetBoostRewardsHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBoostRewardsHistory should return a result")
}

func TestGetSOLRedemptionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSOLRedemptionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSOLRedemptionHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetSOLRedemptionHistory(t.Context(), &StakingRedemptionHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSOLRedemptionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSOLRedemptionHistory(t.Context(), &StakingRedemptionHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetSOLRedemptionHistory must not error")
	if mockTests {
		exp := &SOLRedemptionHistoryResponse{
			Rows: []SOLRedemption{
				{
					Time:             types.Time(time.Unix(1744156800, 0)),
					ArrivalTime:      types.Time(time.Unix(1744416000, 0)),
					Asset:            currency.NewCode("BNSOL"),
					Amount:           1.2,
					DistributeAsset:  currency.SOL,
					DistributeAmount: 1.27108177,
					ExchangeRate:     1.05923481,
					Status:           "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetSOLRedemptionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSOLRedemptionHistory should return a result")
}

func TestGetSOLStakingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSOLStakingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSOLStakingHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetSOLStakingHistory(t.Context(), &StakingSubscriptionHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSOLStakingHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSOLStakingHistory(t.Context(), &StakingSubscriptionHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetSOLStakingHistory must not error")
	if mockTests {
		exp := &SOLStakingHistoryResponse{
			Rows: []SOLStakingSubscription{
				{
					Time:             types.Time(time.Unix(1744156800, 0)),
					Asset:            currency.SOL,
					Amount:           1.27108177,
					DistributeAsset:  currency.NewCode("BNSOL"),
					DistributeAmount: 1.2,
					ExchangeRate:     1.05923481,
					Status:           "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetSOLStakingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSOLStakingHistory should return a result")
}

func TestGetSOLStakingQuotaDetails(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSOLStakingQuotaDetails(t.Context())
	require.NoError(t, err, "GetSOLStakingQuotaDetails must not error")
	if mockTests {
		exp := &SOLStakingQuotaResponse{
			LeftStakingPersonalQuota:    1000,
			LeftRedemptionPersonalQuota: 1000,
			MinimumStakeAmount:          0.01,
			MinimumRedeemAmount:         0.00000001,
			RedeemPeriod:                4,
			Stakeable:                   true,
			Redeemable:                  true,
			SoldOut:                     true,
			CommissionFee:               0.25,
			NextEpochTime:               types.Time(time.Unix(1744243200, 0)),
			Calculating:                 true,
		}
		assert.Equal(t, exp, result, "GetSOLStakingQuotaDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSOLStakingQuotaDetails should return a result")
}

func TestGetUnclaimedRewards(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUnclaimedRewards(t.Context())
	require.NoError(t, err, "GetUnclaimedRewards must not error")
	if mockTests {
		exp := []SOLStakingUnclaimedReward{
			{
				Amount:       1.00000011,
				RewardsAsset: currency.SOL,
			},
		}
		assert.Equal(t, exp, result, "GetUnclaimedRewards should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUnclaimedRewards should return a result")
}

func TestRedeemSOL(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemSOL(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RedeemSOL must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RedeemSOL(t.Context(), 1.2)
	require.NoError(t, err, "RedeemSOL must not error")
	if mockTests {
		exp := &SOLRedemptionResponse{
			Success:      true,
			SOLAmount:    1.27108177,
			RedeemID:     1234567,
			ExchangeRate: 1.05923481,
			ArrivalTime:  types.Time(time.Unix(1744416000, 0)),
		}
		assert.Equal(t, exp, result, "RedeemSOL should decode every field")
		return
	}
	assert.True(t, result.Success, "RedeemSOL should succeed")
}

func TestGetSOLStakingAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSOLStakingAccount(t.Context())
	require.NoError(t, err, "GetSOLStakingAccount must not error")
	if mockTests {
		exp := &SOLStakingAccountResponse{
			BNSOLAmount:           1.10928781,
			HoldingInSOL:          1.17499026,
			ThirtyDaysProfitInSOL: 0.22330928,
		}
		assert.Equal(t, exp, result, "GetSOLStakingAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSOLStakingAccount should return a result")
}

func TestSubscribeToSOLStaking(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeToSOLStaking(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubscribeToSOLStaking must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubscribeToSOLStaking(t.Context(), 1.2)
	require.NoError(t, err, "SubscribeToSOLStaking must not error")
	if mockTests {
		exp := &SOLStakingSubscriptionResponse{
			Success:      true,
			BNSOLAmount:  1.1328923,
			PurchaseID:   1234567,
			ExchangeRate: 1.05923481,
		}
		assert.Equal(t, exp, result, "SubscribeToSOLStaking should decode every field")
		return
	}
	assert.True(t, result.Success, "SubscribeToSOLStaking should succeed")
}
