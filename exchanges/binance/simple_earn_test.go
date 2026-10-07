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

func TestGetSimpleEarnCollateralRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetSimpleEarnCollateralRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSimpleEarnCollateralRecord must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetSimpleEarnCollateralRecord(t.Context(), &SimpleEarnCollateralRecordRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSimpleEarnCollateralRecord must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSimpleEarnCollateralRecord(t.Context(), &SimpleEarnCollateralRecordRequest{ProductID: "USDT001", StartTime: startTime, EndTime: endTime, Size: 10})
	require.NoError(t, err, "GetSimpleEarnCollateralRecord must not error")
	if mockTests {
		exp := &SimpleEarnCollateralRecordResponse{
			Rows: []SimpleEarnCollateralRecord{
				{
					Amount:      100,
					ProductID:   "USDT001",
					Asset:       currency.USDT,
					CreateTime:  types.Time(time.Unix(1744156800, 0)),
					Type:        "REPAY",
					ProductName: "USDT",
					OrderID:     26055,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetSimpleEarnCollateralRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSimpleEarnCollateralRecord should return a result")
}

func TestGetFlexiblePersonalLeftQuota(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexiblePersonalLeftQuota(t.Context(), "")
	require.ErrorIs(t, err, errProductIDRequired, "GetFlexiblePersonalLeftQuota must reject an empty product ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexiblePersonalLeftQuota(t.Context(), "BTC001")
	require.NoError(t, err, "GetFlexiblePersonalLeftQuota must not error")
	if mockTests {
		exp := &SimpleEarnPersonalLeftQuotaResponse{
			LeftPersonalQuota: 1000,
		}
		assert.Equal(t, exp, result, "GetFlexiblePersonalLeftQuota should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFlexiblePersonalLeftQuota should return a result")
}

func TestGetFlexibleProductPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleProductPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleProductPosition must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleProductPosition(t.Context(), &SimpleEarnFlexiblePositionRequest{Asset: currency.BTC, Size: 10})
	require.NoError(t, err, "GetFlexibleProductPosition must not error")
	if mockTests {
		exp := &SimpleEarnFlexiblePositionResponse{
			Rows: []SimpleEarnFlexiblePosition{
				{
					TotalAmount: 0.7546,
					TierAnnualPercentageRate: map[string]float64{
						"0-5BTC":  0.05,
						"5-10BTC": 0.03,
					},
					LatestAnnualPercentageRate:     0.02599895,
					YesterdayAirdropPercentageRate: 0.02599895,
					Asset:                          currency.BTC,
					AirDropAsset:                   currency.BETH,
					CanRedeem:                      true,
					CollateralAmount:               0.23223121,
					ProductID:                      "BTC001",
					YesterdayRealTimeRewards:       0.00010293,
					CumulativeBonusRewards:         0.00022759,
					CumulativeRealTimeRewards:      0.00022759,
					CumulativeTotalRewards:         0.00045518,
					AutoSubscribe:                  true,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetFlexibleProductPosition should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFlexibleProductPosition should return a result")
}

func TestGetFlexibleRedemptionRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleRedemptionRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleRedemptionRecord must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetFlexibleRedemptionRecord(t.Context(), &SimpleEarnFlexibleRedemptionRecordRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFlexibleRedemptionRecord must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleRedemptionRecord(t.Context(), &SimpleEarnFlexibleRedemptionRecordRequest{RedeemID: 1234, Asset: currency.LTC, StartTime: startTime, EndTime: endTime, Size: 12})
	require.NoError(t, err, "GetFlexibleRedemptionRecord must not error")
	if mockTests {
		exp := &SimpleEarnFlexibleRedemptionRecordResponse{
			Rows: []SimpleEarnFlexibleRedemption{
				{
					Amount:             10.54,
					Asset:              currency.LTC,
					Time:               types.Time(time.Unix(1744156800, 0)),
					ProjectID:          "LTC001",
					RedeemID:           1234,
					DestinationAccount: "SPOT",
					Status:             "PAID",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetFlexibleRedemptionRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFlexibleRedemptionRecord should return a result")
}

func TestGetFlexibleRewardHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleRewardHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleRewardHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetFlexibleRewardHistory(t.Context(), &SimpleEarnFlexibleRewardHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFlexibleRewardHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleRewardHistory(t.Context(), &SimpleEarnFlexibleRewardHistoryRequest{
		ProductID: "BTC001",
		Asset:     currency.BTC,
		Type:      "REALTIME",
		StartTime: startTime,
		EndTime:   endTime,
		Current:   1,
		Size:      10,
	})
	require.NoError(t, err, "GetFlexibleRewardHistory must not error")
	if mockTests {
		exp := &SimpleEarnFlexibleRewardHistoryResponse{
			Rows: []SimpleEarnFlexibleReward{
				{
					Asset:     currency.BTC,
					Rewards:   0.00006408,
					ProjectID: "BTC001",
					Type:      "REALTIME",
					Time:      types.Time(time.Unix(1744156800, 0)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetFlexibleRewardHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFlexibleRewardHistory should return a result")
}

func TestGetFlexibleSubscriptionPreview(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleSubscriptionPreview(t.Context(), "", 0.0001)
	require.ErrorIs(t, err, errProductIDRequired, "GetFlexibleSubscriptionPreview must reject an empty product ID")
	_, err = e.GetFlexibleSubscriptionPreview(t.Context(), "BTC001", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "GetFlexibleSubscriptionPreview must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleSubscriptionPreview(t.Context(), "BTC001", 0.0001)
	require.NoError(t, err, "GetFlexibleSubscriptionPreview must not error")
	if mockTests {
		exp := &SimpleEarnFlexibleSubscriptionPreviewResponse{
			TotalAmount:                   0.0101,
			RewardAsset:                   currency.BTC,
			AirDropAsset:                  currency.BETH,
			EstimatedDailyBonusRewards:    0.00000001,
			EstimatedDailyRealTimeRewards: 0.00000003,
			EstimatedDailyAirdropRewards:  0.00000002,
		}
		assert.Equal(t, exp, result, "GetFlexibleSubscriptionPreview should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFlexibleSubscriptionPreview should return a result")
}

func TestGetFlexibleSubscriptionRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleSubscriptionRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFlexibleSubscriptionRecord must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetFlexibleSubscriptionRecord(t.Context(), &SimpleEarnFlexibleSubscriptionRecordRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFlexibleSubscriptionRecord must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFlexibleSubscriptionRecord(t.Context(), &SimpleEarnFlexibleSubscriptionRecordRequest{Asset: currency.ETH, StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetFlexibleSubscriptionRecord must not error")
	if mockTests {
		exp := &SimpleEarnFlexibleSubscriptionRecordResponse{
			Rows: []SimpleEarnFlexibleSubscription{
				{
					Amount:            100,
					Asset:             currency.ETH,
					Time:              types.Time(time.Unix(1744156800, 0)),
					PurchaseID:        26055,
					ProductID:         "ETH001",
					Type:              "AUTO",
					SourceAccount:     "SPOT",
					AmountFromSpot:    30,
					AmountFromFunding: 70,
					Status:            "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetFlexibleSubscriptionRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFlexibleSubscriptionRecord should return a result")
}

func TestGetLockedPersonalLeftQuota(t *testing.T) {
	t.Parallel()
	_, err := e.GetLockedPersonalLeftQuota(t.Context(), "")
	require.ErrorIs(t, err, errProjectIDRequired, "GetLockedPersonalLeftQuota must reject an empty project ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLockedPersonalLeftQuota(t.Context(), "Axs*90")
	require.NoError(t, err, "GetLockedPersonalLeftQuota must not error")
	if mockTests {
		exp := &SimpleEarnPersonalLeftQuotaResponse{
			LeftPersonalQuota: 2000,
		}
		assert.Equal(t, exp, result, "GetLockedPersonalLeftQuota should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLockedPersonalLeftQuota should return a result")
}

func TestGetLockedProductPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetLockedProductPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLockedProductPosition must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLockedProductPosition(t.Context(), &SimpleEarnLockedPositionRequest{Asset: currency.ETH, Size: 12})
	require.NoError(t, err, "GetLockedProductPosition must not error")
	if mockTests {
		exp := &SimpleEarnLockedPositionResponse{
			Rows: []SimpleEarnLockedPosition{
				{
					PositionID:                 123123,
					ParentPositionID:           123122,
					ProjectID:                  "Eth*90",
					Asset:                      currency.ETH,
					Amount:                     1.22092029,
					PurchaseTime:               types.Time(time.Unix(1744156800, 0)),
					Duration:                   90,
					AccrualDays:                4,
					RewardAsset:                currency.ETH,
					APY:                        0.2032,
					RewardAmount:               0.00517181,
					ExtraRewardAsset:           currency.BNB,
					ExtraRewardAPR:             0.0203,
					EstimatedExtraRewardAmount: 0.00517181,
					BoostRewardAsset:           currency.ETH,
					BoostAPR:                   0.0121,
					TotalBoostRewardAmount:     0.00398201,
					NextPay:                    0.00129295,
					NextPayDate:                types.Time(time.Unix(1744243200, 0)),
					PayPeriod:                  1,
					RedeemAmountEarly:          1.21092029,
					RewardsEndDate:             types.Time(time.Unix(1751932800, 0)),
					DeliverDate:                types.Time(time.Unix(1752019200, 0)),
					RedeemPeriod:               1,
					RedeemingAmount:            0.2323,
					RedeemTo:                   "FLEXIBLE",
					PartialAmountDeliverDate:   types.Time(time.Unix(1752019200, 0)),
					CanRedeemEarly:             true,
					CanFastRedemption:          true,
					AutoSubscribe:              true,
					Type:                       "AUTO",
					Status:                     "HOLDING",
					CanReStake:                 true,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetLockedProductPosition should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLockedProductPosition should return a result")
}

func TestGetLockedRedemptionRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetLockedRedemptionRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLockedRedemptionRecord must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetLockedRedemptionRecord(t.Context(), &SimpleEarnLockedRedemptionRecordRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLockedRedemptionRecord must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLockedRedemptionRecord(t.Context(), &SimpleEarnLockedRedemptionRecordRequest{RedeemID: 1234, Asset: currency.LTC, StartTime: startTime, EndTime: endTime, Size: 12})
	require.NoError(t, err, "GetLockedRedemptionRecord must not error")
	if mockTests {
		exp := &SimpleEarnLockedRedemptionRecordResponse{
			Rows: []SimpleEarnLockedRedemption{
				{
					PositionID:                 123123,
					RedeemID:                   1234,
					Time:                       types.Time(time.Unix(1744156800, 0)),
					Asset:                      currency.LTC,
					LockPeriod:                 30,
					Amount:                     21.23223,
					OriginalAmount:             21.23224232,
					Type:                       "MATURE",
					DeliverDate:                types.Time(time.Unix(1744156800, 0)),
					LossAmount:                 0.00001232,
					IsComplete:                 true,
					RewardAsset:                currency.LTC,
					RewardAmount:               0.17181528,
					ExtraRewardAsset:           currency.BNB,
					EstimatedExtraRewardAmount: 0.01718152,
					Status:                     "PAID",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetLockedRedemptionRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLockedRedemptionRecord should return a result")
}

func TestGetLockedRewardHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLockedRewardHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLockedRewardHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetLockedRewardHistory(t.Context(), &SimpleEarnLockedRewardHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLockedRewardHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLockedRewardHistory(t.Context(), &SimpleEarnLockedRewardHistoryRequest{PositionID: 12345, Asset: currency.BTC, StartTime: startTime, EndTime: endTime, Current: 10, Size: 40})
	require.NoError(t, err, "GetLockedRewardHistory must not error")
	if mockTests {
		exp := &SimpleEarnLockedRewardHistoryResponse{
			Rows: []SimpleEarnLockedReward{
				{
					PositionID: 12345,
					Time:       types.Time(time.Unix(1744156800, 0)),
					Asset:      currency.BTC,
					LockPeriod: 30,
					Amount:     0.00021312,
					Type:       "Locked Rewards",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetLockedRewardHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLockedRewardHistory should return a result")
}

func TestGetLockedSubscriptionPreview(t *testing.T) {
	t.Parallel()
	_, err := e.GetLockedSubscriptionPreview(t.Context(), "", 0.1234, false)
	require.ErrorIs(t, err, errProjectIDRequired, "GetLockedSubscriptionPreview must reject an empty project ID")
	_, err = e.GetLockedSubscriptionPreview(t.Context(), "Axs*90", 0, false)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "GetLockedSubscriptionPreview must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLockedSubscriptionPreview(t.Context(), "Axs*90", 0.1234, false)
	require.NoError(t, err, "GetLockedSubscriptionPreview must not error")
	if mockTests {
		exp := []SimpleEarnLockedSubscriptionPreview{
			{
				RewardAsset:                     currency.AXS,
				TotalRewardAmount:               0.00517181,
				ExtraRewardAsset:                currency.BNB,
				EstimatedTotalExtraRewardAmount: 0.00051718,
				BoostRewardAsset:                currency.AXS,
				EstimatedDailyRewardAmount:      0.00005746,
				NextPay:                         0.00005746,
				NextPayDate:                     types.Time(time.Unix(1744243200, 0)),
				ValueDate:                       types.Time(time.Unix(1744243200, 0)),
				RewardsEndDate:                  types.Time(time.Unix(1752019200, 0)),
				DeliverDate:                     types.Time(time.Unix(1752105600, 0)),
				NextSubscriptionDate:            types.Time(time.Unix(1752105600, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetLockedSubscriptionPreview should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLockedSubscriptionPreview should return a result")
}

func TestGetLockedSubscriptionRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetLockedSubscriptionRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLockedSubscriptionRecord must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetLockedSubscriptionRecord(t.Context(), &SimpleEarnLockedSubscriptionRecordRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLockedSubscriptionRecord must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLockedSubscriptionRecord(t.Context(), &SimpleEarnLockedSubscriptionRecordRequest{Asset: currency.ETH, StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetLockedSubscriptionRecord must not error")
	if mockTests {
		exp := &SimpleEarnLockedSubscriptionRecordResponse{
			Rows: []SimpleEarnLockedSubscription{
				{
					PositionID:        123123,
					PurchaseID:        "26055",
					ProjectID:         "Eth*90",
					Time:              types.Time(time.Unix(1744156800, 0)),
					Asset:             currency.ETH,
					Amount:            1.22092029,
					LockPeriod:        90,
					Type:              "AUTO",
					SourceAccount:     "SPOTANDFUNDING",
					AmountFromSpot:    0.5,
					AmountFromFunding: 0.72092029,
					Status:            "SUCCESS",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetLockedSubscriptionRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLockedSubscriptionRecord should return a result")
}

func TestGetSimpleEarnRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSimpleEarnRateHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSimpleEarnRateHistory must reject a nil request")
	_, err = e.GetSimpleEarnRateHistory(t.Context(), &SimpleEarnRateHistoryRequest{Size: 100})
	require.ErrorIs(t, err, errProductIDRequired, "GetSimpleEarnRateHistory must reject an empty product ID")
	startTime, endTime := getTime()
	_, err = e.GetSimpleEarnRateHistory(t.Context(), &SimpleEarnRateHistoryRequest{ProductID: "BTC001", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSimpleEarnRateHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSimpleEarnRateHistory(t.Context(), &SimpleEarnRateHistoryRequest{
		ProductID: "BTC001",
		APRPeriod: "YEAR",
		StartTime: startTime,
		EndTime:   endTime,
		Current:   1,
		Size:      100,
	})
	require.NoError(t, err, "GetSimpleEarnRateHistory must not error")
	if mockTests {
		exp := &SimpleEarnRateHistoryResponse{
			Rows: []SimpleEarnRate{
				{
					ProductID:            "BTC001",
					Asset:                currency.BTC,
					AnnualPercentageRate: 0.0412,
					Time:                 types.Time(time.Unix(1744156800, 0)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetSimpleEarnRateHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSimpleEarnRateHistory should return a result")
}

func TestGetSimpleEarnFlexibleProductList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSimpleEarnFlexibleProductList(t.Context(), currency.BTC, 2, 10)
	require.NoError(t, err, "GetSimpleEarnFlexibleProductList must not error")
	if mockTests {
		exp := &SimpleEarnFlexibleProductListResponse{
			Rows: []SimpleEarnFlexibleProduct{
				{
					Asset:                      currency.BTC,
					LatestAnnualPercentageRate: 0.05,
					TierAnnualPercentageRate: map[string]float64{
						"0-5BTC":  0.05,
						"5-10BTC": 0.03,
					},
					AirDropPercentageRate: 0.05,
					CanPurchase:           true,
					CanRedeem:             true,
					Hot:                   true,
					MinimumPurchaseAmount: 0.01,
					ProductID:             "BTC001",
					SubscriptionStartTime: types.Time(time.Unix(1646182276, 0)),
					Status:                "PURCHASING",
				},
				{
					Asset:                      currency.BTC,
					LatestAnnualPercentageRate: 0.03,
					TierAnnualPercentageRate: map[string]float64{
						"0-1BTC": 0.03,
					},
					AirDropPercentageRate: 0.01,
					CanRedeem:             true,
					IsSoldOut:             true,
					MinimumPurchaseAmount: 0.001,
					ProductID:             "BTC002",
					SubscriptionStartTime: types.Time(time.Unix(1646182276, 0)),
					Status:                "END",
				},
			},
			Total: 12,
		}
		assert.Equal(t, exp, result, "GetSimpleEarnFlexibleProductList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSimpleEarnFlexibleProductList should return a result")
}

func TestGetSimpleEarnLockedProducts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSimpleEarnLockedProducts(t.Context(), currency.BTC, 2, 10)
	require.NoError(t, err, "GetSimpleEarnLockedProducts must not error")
	if mockTests {
		exp := &SimpleEarnLockedProductListResponse{
			Rows: []SimpleEarnLockedProduct{
				{
					ProjectID: "Btc*90",
					Detail: SimpleEarnLockedProductDetail{
						Asset:                 currency.BTC,
						RewardAsset:           currency.BTC,
						Duration:              90,
						Renewable:             true,
						APR:                   0.0206,
						Status:                "PURCHASING",
						SubscriptionStartTime: types.Time(time.Unix(1646182276, 0)),
						ExtraRewardAsset:      currency.BNB,
						ExtraRewardAPR:        0.023,
						BoostRewardAsset:      currency.BTC,
						BoostAPR:              0.0121,
						BoostEndTime:          types.Time(time.Unix(1746748800, 0)),
					},
					Quota: SimpleEarnLockedProductQuota{
						TotalPersonalQuota: 2,
						Minimum:            0.001,
					},
				},
				{
					ProjectID: "Btc*120",
					Detail: SimpleEarnLockedProductDetail{
						Asset:                 currency.BTC,
						RewardAsset:           currency.BTC,
						Duration:              120,
						IsSoldOut:             true,
						APR:                   0.0254,
						Status:                "END",
						SubscriptionStartTime: types.Time(time.Unix(1646182276, 0)),
						ExtraRewardAsset:      currency.BNB,
						ExtraRewardAPR:        0.021,
						BoostRewardAsset:      currency.BTC,
						BoostAPR:              0.0101,
						BoostEndTime:          types.Time(time.Unix(1746748800, 0)),
					},
					Quota: SimpleEarnLockedProductQuota{
						TotalPersonalQuota: 1,
						Minimum:            0.001,
					},
				},
			},
			Total: 12,
		}
		assert.Equal(t, exp, result, "GetSimpleEarnLockedProducts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSimpleEarnLockedProducts should return a result")
}

func TestRedeemFlexibleProduct(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemFlexibleProduct(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RedeemFlexibleProduct must reject a nil request")
	_, err = e.RedeemFlexibleProduct(t.Context(), &RedeemFlexibleProductRequest{Amount: 0.1234})
	require.ErrorIs(t, err, errProductIDRequired, "RedeemFlexibleProduct must reject an empty product ID")
	_, err = e.RedeemFlexibleProduct(t.Context(), &RedeemFlexibleProductRequest{ProductID: "BTC001"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RedeemFlexibleProduct must reject a zero amount unless redeeming all")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RedeemFlexibleProduct(t.Context(), &RedeemFlexibleProductRequest{ProductID: "BTC001", Amount: 0.1234, DestinationAccount: "FUND"})
	require.NoError(t, err, "RedeemFlexibleProduct must not error")
	if mockTests {
		exp := &SimpleEarnRedemptionResponse{
			RedeemID: 40607,
			Success:  true,
		}
		assert.Equal(t, exp, result, "RedeemFlexibleProduct should decode every field")
		return
	}
	assert.True(t, result.Success, "RedeemFlexibleProduct should succeed")
}

func TestRedeemLockedProduct(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemLockedProduct(t.Context(), 0)
	require.ErrorIs(t, err, errPositionIDRequired, "RedeemLockedProduct must reject an empty position ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RedeemLockedProduct(t.Context(), 12345)
	require.NoError(t, err, "RedeemLockedProduct must not error")
	if mockTests {
		exp := &SimpleEarnRedemptionResponse{
			RedeemID: 40608,
			Success:  true,
		}
		assert.Equal(t, exp, result, "RedeemLockedProduct should decode every field")
		return
	}
	assert.True(t, result.Success, "RedeemLockedProduct should succeed")
}

func TestSetFlexibleAutoSubscribe(t *testing.T) {
	t.Parallel()
	_, err := e.SetFlexibleAutoSubscribe(t.Context(), "", true)
	require.ErrorIs(t, err, errProductIDRequired, "SetFlexibleAutoSubscribe must reject an empty product ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetFlexibleAutoSubscribe(t.Context(), "BTC001", true)
	require.NoError(t, err, "SetFlexibleAutoSubscribe must not error")
	if mockTests {
		exp := &EarnSuccessResponse{
			Success: true,
		}
		assert.Equal(t, exp, result, "SetFlexibleAutoSubscribe should decode every field")
		return
	}
	assert.True(t, result.Success, "SetFlexibleAutoSubscribe should succeed")
}

func TestSetLockedAutoSubscribe(t *testing.T) {
	t.Parallel()
	_, err := e.SetLockedAutoSubscribe(t.Context(), 0, true)
	require.ErrorIs(t, err, errPositionIDRequired, "SetLockedAutoSubscribe must reject an empty position ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetLockedAutoSubscribe(t.Context(), 12345, true)
	require.NoError(t, err, "SetLockedAutoSubscribe must not error")
	if mockTests {
		exp := &EarnSuccessResponse{
			Success: true,
		}
		assert.Equal(t, exp, result, "SetLockedAutoSubscribe should decode every field")
		return
	}
	assert.True(t, result.Success, "SetLockedAutoSubscribe should succeed")
}

func TestSetLockedProductRedeemOption(t *testing.T) {
	t.Parallel()
	_, err := e.SetLockedProductRedeemOption(t.Context(), 0, "FLEXIBLE")
	require.ErrorIs(t, err, errPositionIDRequired, "SetLockedProductRedeemOption must reject an empty position ID")
	_, err = e.SetLockedProductRedeemOption(t.Context(), 12345, "")
	require.ErrorIs(t, err, errRedemptionAccountRequired, "SetLockedProductRedeemOption must reject an empty redeem destination")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetLockedProductRedeemOption(t.Context(), 12345, "FLEXIBLE")
	require.NoError(t, err, "SetLockedProductRedeemOption must not error")
	if mockTests {
		exp := &EarnSuccessResponse{
			Success: true,
		}
		assert.Equal(t, exp, result, "SetLockedProductRedeemOption should decode every field")
		return
	}
	assert.True(t, result.Success, "SetLockedProductRedeemOption should succeed")
}

func TestSimpleAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.SimpleAccount(t.Context())
	require.NoError(t, err, "SimpleAccount must not error")
	if mockTests {
		exp := &SimpleAccountResponse{
			TotalAmountInBTC:          0.01067982,
			TotalAmountInUSDT:         1077.1328923,
			TotalFlexibleAmountInBTC:  0.00532991,
			TotalFlexibleAmountInUSDT: 537.56644615,
			TotalLockedInBTC:          0.00534991,
			TotalLockedInUSDT:         539.56644615,
		}
		assert.Equal(t, exp, result, "SimpleAccount should decode every field")
		return
	}
	assert.NotNil(t, result, "SimpleAccount should return a result")
}

func TestSubscribeToFlexibleProducts(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeToFlexibleProducts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubscribeToFlexibleProducts must reject a nil request")
	_, err = e.SubscribeToFlexibleProducts(t.Context(), &SubscribeFlexibleProductRequest{Amount: 1})
	require.ErrorIs(t, err, errProductIDRequired, "SubscribeToFlexibleProducts must reject an empty product ID")
	_, err = e.SubscribeToFlexibleProducts(t.Context(), &SubscribeFlexibleProductRequest{ProductID: "BTC001"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubscribeToFlexibleProducts must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubscribeToFlexibleProducts(t.Context(), &SubscribeFlexibleProductRequest{ProductID: "BTC001", Amount: 1, AutoSubscribe: new(true), SourceAccount: "FUND"})
	require.NoError(t, err, "SubscribeToFlexibleProducts must not error")
	if mockTests {
		exp := &SimpleEarnFlexibleSubscriptionResponse{
			PurchaseID: 40607,
			Success:    true,
		}
		assert.Equal(t, exp, result, "SubscribeToFlexibleProducts should decode every field")
		return
	}
	assert.True(t, result.Success, "SubscribeToFlexibleProducts should succeed")
}

func TestSubscribeToLockedProducts(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeToLockedProducts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubscribeToLockedProducts must reject a nil request")
	_, err = e.SubscribeToLockedProducts(t.Context(), &SubscribeLockedProductRequest{Amount: 1})
	require.ErrorIs(t, err, errProjectIDRequired, "SubscribeToLockedProducts must reject an empty project ID")
	_, err = e.SubscribeToLockedProducts(t.Context(), &SubscribeLockedProductRequest{ProjectID: "Axs*90"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubscribeToLockedProducts must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubscribeToLockedProducts(t.Context(), &SubscribeLockedProductRequest{
		ProjectID:     "Axs*90",
		Amount:        1,
		AutoSubscribe: new(true),
		SourceAccount: "SPOT",
		RedeemTo:      "FLEXIBLE",
	})
	require.NoError(t, err, "SubscribeToLockedProducts must not error")
	if mockTests {
		exp := &SimpleEarnLockedSubscriptionResponse{
			PurchaseID: 40608,
			PositionID: "12345",
			Success:    true,
		}
		assert.Equal(t, exp, result, "SubscribeToLockedProducts should decode every field")
		return
	}
	assert.True(t, result.Success, "SubscribeToLockedProducts should succeed")
}
