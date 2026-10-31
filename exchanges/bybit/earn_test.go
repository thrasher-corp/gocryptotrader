package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetEarnProductInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnProductInfo(t.Context(), "", currency.BTC)
	require.ErrorIs(t, err, errCategoryNotSet, "GetEarnProductInfo must reject an empty category")

	for _, tc := range []struct {
		category string
		coin     currency.Code
		exp      *EarnProductInfoResponse
	}{
		{
			category: "FlexibleSaving",
			coin:     currency.BTC,
			exp: &EarnProductInfoResponse{
				List: []EarnProduct{
					{
						Category:       "FlexibleSaving",
						EstimateAPR:    "0.1%",
						Coin:           currency.BTC,
						MinStakeAmount: 0.002,
						MaxStakeAmount: 1500,
						Precision:      8,
						ProductID:      "3",
						Status:         "Available",
						BonusEvents: []EarnBonusEvent{
							{
								APR:          "1.5%",
								Coin:         currency.BTC,
								Announcement: "https://announcements.bybit.com/en/article/flexible-savings-btc-bonus",
							},
						},
						HasTieredAPR: true,
						TierAPRDetails: []EarnTierAPRDetail{
							{
								Max:         0.003,
								EstimateAPR: "0.4%",
							},
							{
								Min:         0.003,
								Max:         -1,
								EstimateAPR: "0.1%",
							},
						},
						RemainingPoolAmount: 7980.26721726,
					},
				},
			},
		},
		{
			category: "OnChain",
			exp: &EarnProductInfoResponse{
				List: []EarnProduct{
					{
						Category:                "OnChain",
						EstimateAPR:             "2.17%",
						Coin:                    currency.ETH,
						MinStakeAmount:          0.1,
						MaxStakeAmount:          10000,
						Precision:               8,
						ProductID:               "1",
						Status:                  "Available",
						BonusEvents:             []EarnBonusEvent{},
						MinRedeemAmount:         0.01,
						MaxRedeemAmount:         1000,
						Duration:                "Flexible",
						SwapCoin:                currency.NewCode("STETH"),
						SwapCoinPrecision:       8,
						StakeExchangeRate:       0.99928374,
						RedeemExchangeRate:      1.000717,
						RewardDistributionType:  "Other",
						RewardIntervalMinute:    1440,
						RedeemProcessingMinute:  11520,
						StakeTime:               types.Time(time.UnixMilli(1791535836334)),
						InterestCalculationTime: types.Time(time.Unix(1791676800, 0)),
						TierAPRDetails:          []EarnTierAPRDetail{},
					},
					{
						Category:                "OnChain",
						EstimateAPR:             "1.2%",
						Coin:                    currency.BTC,
						MinStakeAmount:          0.1,
						MaxStakeAmount:          200,
						Precision:               8,
						ProductID:               "11",
						Status:                  "NotAvailable",
						BonusEvents:             []EarnBonusEvent{},
						Duration:                "Fixed",
						Term:                    45,
						RewardDistributionType:  "Simple",
						RewardIntervalMinute:    64800,
						RedeemProcessingMinute:  64800,
						StakeTime:               types.Time(time.Unix(1791590400, 0)),
						InterestCalculationTime: types.Time(time.Unix(1791590400, 0)),
						TierAPRDetails:          []EarnTierAPRDetail{},
					},
				},
			},
		},
	} {
		t.Run(tc.category, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetEarnProductInfo(t.Context(), tc.category, tc.coin)
			require.NoError(t, err, "GetEarnProductInfo must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetEarnProductInfo should decode every field")
				return
			}
			require.NotEmpty(t, result.List, "GetEarnProductInfo must return products")
			assert.Equal(t, tc.category, result.List[0].Category, "GetEarnProductInfo should return products of the category")
		})
	}
}

func TestPlaceEarnOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceEarnOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceEarnOrder must reject a nil request")
	req := &PlaceEarnOrderRequest{}
	_, err = e.PlaceEarnOrder(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "PlaceEarnOrder must reject an empty category")
	req.Category = "FlexibleSaving"
	_, err = e.PlaceEarnOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "PlaceEarnOrder must reject an empty order type")
	req.OrderType = "Stake"
	_, err = e.PlaceEarnOrder(t.Context(), req)
	require.ErrorIs(t, err, errMissingAccountType, "PlaceEarnOrder must reject an empty account type")
	req.AccountType = "FUND"
	_, err = e.PlaceEarnOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceEarnOrder must reject a zero amount")
	req.Amount = 0.35
	_, err = e.PlaceEarnOrder(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceEarnOrder must reject an empty coin")
	req.Coin = currency.BTC
	_, err = e.PlaceEarnOrder(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "PlaceEarnOrder must reject an empty product ID")
	req.ProductID = "430"
	_, err = e.PlaceEarnOrder(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "PlaceEarnOrder must reject an empty order link ID")
	req.OrderLinkID = "btc-earn-001"
	req.InterestCard = &EarnInterestCard{}
	_, err = e.PlaceEarnOrder(t.Context(), req)
	require.ErrorIs(t, err, errAwardIDRequired, "PlaceEarnOrder must reject an interest card without an award ID")
	req.InterestCard.AwardID = 1001
	_, err = e.PlaceEarnOrder(t.Context(), req)
	require.ErrorIs(t, err, errSpecCodeRequired, "PlaceEarnOrder must reject an interest card without a spec code")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PlaceEarnOrderRequest
		exp  *PlaceEarnOrderResponse
	}{
		{
			name: "FlexibleSaving stake with interest card",
			req: &PlaceEarnOrderRequest{
				Category:     "FlexibleSaving",
				OrderType:    "Stake",
				AccountType:  "FUND",
				Amount:       0.35,
				Coin:         currency.BTC,
				ProductID:    "430",
				OrderLinkID:  "btc-earn-001",
				InterestCard: &EarnInterestCard{AwardID: 1001, SpecCode: "FS_APY_3PCT_30D"},
			},
			exp: &PlaceEarnOrderResponse{
				OrderID:     "0572b030-6a0b-423f-88c4-b6ce31c0c82d",
				OrderLinkID: "btc-earn-001",
			},
		},
		{
			name: "OnChain redeem",
			req: &PlaceEarnOrderRequest{
				Category:         "OnChain",
				OrderType:        "Redeem",
				AccountType:      "FUND",
				Amount:           10,
				Coin:             currency.SOL,
				ProductID:        "3",
				OrderLinkID:      "sol-onchain-redeem-001",
				RedeemPositionID: "326",
				ToAccountType:    "FUND",
			},
			exp: &PlaceEarnOrderResponse{
				OrderID:     "7f8b2a64-21cd-4e5b-9c3a-0d2f6e8a1b43",
				OrderLinkID: "sol-onchain-redeem-001",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceEarnOrder(t.Context(), tc.req)
			require.NoError(t, err, "PlaceEarnOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceEarnOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderID, "PlaceEarnOrder should return an order ID")
		})
	}
}

func TestGetEarnOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEarnOrderHistory must reject a nil request")
	_, err = e.GetEarnOrderHistory(t.Context(), &EarnOrderHistoryRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetEarnOrderHistory must reject an empty category")
	_, err = e.GetEarnOrderHistory(t.Context(), &EarnOrderHistoryRequest{Category: "FlexibleSaving", StartTime: time.UnixMilli(1760000000000), EndTime: time.UnixMilli(1759395200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEarnOrderHistory must reject a start time after the end time")
	_, err = e.GetEarnOrderHistory(t.Context(), &EarnOrderHistoryRequest{Category: "OnChain", ProductID: "8"})
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired, "GetEarnOrderHistory must require an order ID or order link ID for OnChain")
	// Either ID alone is enough for OnChain
	onChain := restReplyTestInstance(t, map[string]string{"/v5/earn/order": `{"retCode":0,"retMsg":"OK","result":{"list":[],"nextPageCursor":""},"retExtInfo":{},"time":1760000000000}`})
	for _, req := range []*EarnOrderHistoryRequest{{Category: "OnChain", OrderID: "1"}, {Category: "OnChain", OrderLinkID: "link-1"}} {
		_, err = onChain.GetEarnOrderHistory(t.Context(), req)
		assert.NoErrorf(t, err, "GetEarnOrderHistory should accept OnChain with order ID %q and link ID %q", req.OrderID, req.OrderLinkID)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *EarnOrderHistoryRequest
		exp  *EarnOrderHistoryResponse
	}{
		{
			name: "OnChain order",
			req:  &EarnOrderHistoryRequest{Category: "OnChain", OrderID: "9640dc23-df1a-448a-ad24-e1a48028a51f", OrderLinkID: "eth-onchain-stake-001"},
			exp: &EarnOrderHistoryResponse{
				List: []EarnOrder{
					{
						Coin:               currency.ETH,
						OrderValue:         1.5,
						OrderType:          "Stake",
						OrderID:            "9640dc23-df1a-448a-ad24-e1a48028a51f",
						OrderLinkID:        "eth-onchain-stake-001",
						Status:             "Pending",
						CreatedAt:          types.Time(time.Unix(1759983689, 0)),
						ProductID:          "1",
						UpdatedAt:          types.Time(time.Unix(1759983749, 0)),
						SwapOrderValue:     1.49892561,
						EstimateRedeemTime: types.Time(time.Unix(1760675689, 0)),
						EstimateStakeTime:  types.Time(time.Unix(1759987289, 0)),
					},
				},
			},
		},
		{
			name: "FlexibleSaving orders",
			req:  &EarnOrderHistoryRequest{Category: "FlexibleSaving", ProductID: "428", StartTime: time.UnixMilli(1759395200000), EndTime: time.UnixMilli(1760000000000), Limit: 20},
			exp: &EarnOrderHistoryResponse{
				NextPageCursor: "eyJpZCI6MTAwMjA5Nn0=",
				List: []EarnOrder{
					{
						Coin:        currency.USDT,
						OrderValue:  1000,
						OrderType:   "Stake",
						OrderID:     "ad98d473-4e17-46da-ab30-5563f62a97fa",
						OrderLinkID: "usdt-earn-001",
						Status:      "Success",
						CreatedAt:   types.Time(time.Unix(1759983689, 0)),
						ProductID:   "428",
						UpdatedAt:   types.Time(time.Unix(1759983689, 0)),
					},
					{
						Coin:       currency.USDT,
						OrderValue: 250,
						OrderType:  "Redeem",
						OrderID:    "05a7012d-c4d6-493a-8c6b-023a1038944a",
						Status:     "Success",
						CreatedAt:  types.Time(time.Unix(1759993805, 0)),
						ProductID:  "428",
						UpdatedAt:  types.Time(time.Unix(1759993806, 0)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetEarnOrderHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetEarnOrderHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetEarnOrderHistory should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetEarnOrderHistory(t.Context(), tc.req)
			assert.NoError(t, err, "GetEarnOrderHistory should not error for the next page")
		})
	}
}

func TestGetEarnStakedPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnStakedPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEarnStakedPosition must reject a nil request")
	_, err = e.GetEarnStakedPosition(t.Context(), &EarnStakedPositionRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetEarnStakedPosition must reject an empty category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *EarnStakedPositionRequest
		exp  *EarnStakedPositionResponse
	}{
		{
			name: "OnChain",
			req:  &EarnStakedPositionRequest{Category: "OnChain", ProductID: "8", Coin: currency.BTC},
			exp: &EarnStakedPositionResponse{
				List: []EarnStakedPosition{
					{
						Coin:                            currency.BTC,
						ProductID:                       "8",
						Amount:                          0.1,
						TotalPNL:                        0.000027397260273973,
						ID:                              "326",
						Status:                          "Active",
						OrderID:                         "1a5a8945-e042-4dd5-a93f-c0f0577377ad",
						EstimateRedeemTime:              types.Time(time.Unix(1744675200, 0)),
						EstimateStakeTime:               types.Time(time.Unix(1739948177, 0)),
						EstimateInterestCalculationTime: types.Time(time.Unix(1744243200, 0)),
						SettlementTime:                  types.Time(time.Unix(1744675200, 0)),
						AutoReinvest:                    "Enable",
						AvailableAmount:                 0.09,
						FreezeDetails: []EarnFreezeDetail{
							{
								Amount:      0.01,
								Description: "Locked in Fixed-Rate Loan",
							},
						},
					},
				},
			},
		},
		{
			name: "FlexibleSaving",
			req:  &EarnStakedPositionRequest{Category: "FlexibleSaving", Coin: currency.USDT},
			exp: &EarnStakedPositionResponse{
				List: []EarnStakedPosition{
					{
						Coin:            currency.USDT,
						ProductID:       "428",
						Amount:          5000,
						ClaimableYield:  0.0608,
						AvailableAmount: 4900,
						FreezeDetails:   []EarnFreezeDetail{},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetEarnStakedPosition(t.Context(), tc.req)
			require.NoError(t, err, "GetEarnStakedPosition must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetEarnStakedPosition should decode every field")
				return
			}
			assert.NotNil(t, result, "GetEarnStakedPosition should return a result")
		})
	}
}

func TestModifyEarnPosition(t *testing.T) {
	t.Parallel()
	err := e.ModifyEarnPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifyEarnPosition must reject a nil request")
	req := &ModifyEarnPositionRequest{}
	err = e.ModifyEarnPosition(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "ModifyEarnPosition must reject an empty category")
	req.Category = "OnChain"
	err = e.ModifyEarnPosition(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "ModifyEarnPosition must reject an empty product ID")
	req.ProductID = 8
	err = e.ModifyEarnPosition(t.Context(), req)
	require.ErrorIs(t, err, errPositionIDRequired, "ModifyEarnPosition must reject an empty position ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.ModifyEarnPosition(t.Context(), &ModifyEarnPositionRequest{Category: "OnChain", ProductID: 8, PositionID: 326, AutoReinvest: true})
	assert.NoError(t, err, "ModifyEarnPosition should not error")
}

func TestGetEarnYieldHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnYieldHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEarnYieldHistory must reject a nil request")
	_, err = e.GetEarnYieldHistory(t.Context(), &EarnYieldHistoryRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetEarnYieldHistory must reject an empty category")
	_, err = e.GetEarnYieldHistory(t.Context(), &EarnYieldHistoryRequest{Category: "OnChain", ProductID: "8"})
	require.ErrorIs(t, err, errOnChainProductIDRejected, "GetEarnYieldHistory must reject a product ID for OnChain")
	_, err = e.GetEarnYieldHistory(t.Context(), &EarnYieldHistoryRequest{Category: "FlexibleSaving", StartTime: time.UnixMilli(1760000000000), EndTime: time.UnixMilli(1759395200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEarnYieldHistory must reject a start time after the end time")

	start, end := time.UnixMilli(1759395200000), time.UnixMilli(1760000000000)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		// Yields are only kept for three months
		end = time.Now()
		start = end.AddDate(0, 0, -7)
	}
	req := &EarnYieldHistoryRequest{Category: "FlexibleSaving", ProductID: "428", StartTime: start, EndTime: end, Limit: 20}
	result, err := e.GetEarnYieldHistory(t.Context(), req)
	require.NoError(t, err, "GetEarnYieldHistory must not error")
	if mockTests {
		exp := &EarnYieldHistoryResponse{
			NextPageCursor: "eyJpZCI6MTAwMjA5N30=",
			Yield: []EarnYield{
				{
					ProductID:              "428",
					Coin:                   currency.USDT,
					ID:                     "1002096",
					Amount:                 0.0608,
					YieldType:              "Normal",
					DistributionMode:       "Manual",
					EffectiveStakingAmount: 1000,
					OrderID:                "05a7012d-c4d6-493a-8c6b-023a1038944a",
					Status:                 "Success",
					CreatedAt:              types.Time(time.Unix(1759993805, 0)),
				},
				{
					ProductID:              "428",
					Coin:                   currency.USDT,
					ID:                     "1002097",
					Amount:                 0.0125,
					YieldType:              "Bonus",
					DistributionMode:       "Auto",
					EffectiveStakingAmount: 1000,
					Status:                 "Pending",
					CreatedAt:              types.Time(time.Unix(1759995605, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetEarnYieldHistory should decode every field")
	}
	if result.NextPageCursor == "" {
		return
	}
	req.Cursor = result.NextPageCursor
	_, err = e.GetEarnYieldHistory(t.Context(), req)
	assert.NoError(t, err, "GetEarnYieldHistory should not error for the next page")
}

func TestGetEarnHourlyYieldHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnHourlyYieldHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEarnHourlyYieldHistory must reject a nil request")
	_, err = e.GetEarnHourlyYieldHistory(t.Context(), &EarnYieldHistoryRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetEarnHourlyYieldHistory must reject an empty category")
	_, err = e.GetEarnHourlyYieldHistory(t.Context(), &EarnYieldHistoryRequest{Category: "FlexibleSaving", StartTime: time.UnixMilli(1760000000000), EndTime: time.UnixMilli(1759395200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEarnHourlyYieldHistory must reject a start time after the end time")

	start, end := time.UnixMilli(1759395200000), time.UnixMilli(1760000000000)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		end = time.Now()
		start = end.AddDate(0, 0, -7)
	}
	req := &EarnYieldHistoryRequest{Category: "FlexibleSaving", ProductID: "428", StartTime: start, EndTime: end, Limit: 20}
	result, err := e.GetEarnHourlyYieldHistory(t.Context(), req)
	require.NoError(t, err, "GetEarnHourlyYieldHistory must not error")
	if mockTests {
		exp := &EarnHourlyYieldHistoryResponse{
			NextPageCursor: "eyJpZCI6MjA5ODExNDR9",
			List: []EarnHourlyYield{
				{
					ProductID:              "428",
					Coin:                   currency.USDT,
					ID:                     "20981144",
					Amount:                 0.06081050228310502,
					EffectiveStakingAmount: 1000,
					Status:                 "Success",
					HourlyDate:             types.Time(time.Unix(1759989600, 0)),
					CreatedAt:              types.Time(time.Unix(1759989603, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetEarnHourlyYieldHistory should decode every field")
	}
	if result.NextPageCursor == "" {
		return
	}
	req.Cursor = result.NextPageCursor
	_, err = e.GetEarnHourlyYieldHistory(t.Context(), req)
	assert.NoError(t, err, "GetEarnHourlyYieldHistory should not error for the next page")
}

func TestGetEarnAPRHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnAPRHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEarnAPRHistory must reject a nil request")
	_, err = e.GetEarnAPRHistory(t.Context(), &EarnAPRHistoryRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetEarnAPRHistory must reject an empty category")
	_, err = e.GetEarnAPRHistory(t.Context(), &EarnAPRHistoryRequest{Category: "OnChain"})
	require.ErrorIs(t, err, errProductIDRequired, "GetEarnAPRHistory must reject an empty product ID")
	_, err = e.GetEarnAPRHistory(t.Context(), &EarnAPRHistoryRequest{Category: "OnChain", ProductID: "8", StartTime: time.UnixMilli(1791331200000), EndTime: time.UnixMilli(1791072000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEarnAPRHistory must reject a start time after the end time")

	req := &EarnAPRHistoryRequest{Category: "OnChain", ProductID: "8", StartTime: time.UnixMilli(1791072000000), EndTime: time.UnixMilli(1791331200000)}
	if !mockTests {
		products, err := e.GetEarnProductInfo(t.Context(), "OnChain", currency.EMPTYCODE)
		require.NoError(t, err, "GetEarnProductInfo must not error")
		require.NotEmpty(t, products.List, "GetEarnProductInfo must return products")
		req.ProductID = products.List[0].ProductID
		// The API only keeps six months of APRs
		req.EndTime = time.Now()
		req.StartTime = req.EndTime.AddDate(0, 0, -3)
	}
	result, err := e.GetEarnAPRHistory(t.Context(), req)
	require.NoError(t, err, "GetEarnAPRHistory must not error")
	if mockTests {
		exp := &EarnAPRHistoryResponse{
			List: []EarnAPR{
				{
					Timestamp: types.Time(time.Unix(1791244800, 0)),
					APR:       "14.142001%",
				},
				{
					Timestamp: types.Time(time.Unix(1791158400, 0)),
					APR:       "14.172735%",
				},
				{
					Timestamp: types.Time(time.Unix(1791072000, 0)),
					APR:       "14.236871%",
				},
			},
		}
		assert.Equal(t, exp, result, "GetEarnAPRHistory should decode every field")
		return
	}
	require.NotEmpty(t, result.List, "GetEarnAPRHistory must return APRs")
	assert.NotEmpty(t, result.List[0].APR, "GetEarnAPRHistory should return an APR")
}

func TestGetEarnCouponList(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnCouponList(t.Context(), "")
	require.ErrorIs(t, err, errCategoryNotSet, "GetEarnCouponList must reject an empty category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEarnCouponList(t.Context(), "DualAssets")
	require.NoError(t, err, "GetEarnCouponList must not error")
	if mockTests {
		exp := &EarnCouponListResponse{
			InterestCards: []EarnCouponInterestCard{
				{
					AwardID:                 1001,
					SpecCode:                "DA_APY_3PCT_30D",
					Coin:                    currency.USDT,
					APY:                     0.03,
					Duration:                30,
					ClaimedAt:               types.Time(time.Unix(1759900800, 0)),
					ExpireAt:                types.Time(time.Unix(1762492800, 0)),
					UsedAt:                  types.Time(time.Unix(1759987200, 0)),
					Status:                  "InUse",
					CurrentPNL:              1.25,
					LimitPNL:                100,
					PositionEffectiveAmount: 5000,
					ProductID:               5512,
					Category:                "DualAssets",
				},
			},
			AwardCards: []EarnCouponAwardCard{
				{
					AwardID:            2002,
					SpecCode:           "DA_TRIAL_100USDT",
					ClaimedAt:          types.Time(time.Unix(1759900800, 0)),
					UsedAt:             types.Time(time.Unix(1759987200, 0)),
					ExpireAt:           types.Time(time.Unix(1762492800, 0)),
					Status:             "AlreadyUsed",
					Amount:             100,
					LimitPNLPercentage: 0.23,
					BaseCoin:           currency.BTC,
					QuoteCoin:          currency.USDT,
					Direction:          1,
					Category:           "DualAssets",
				},
			},
		}
		assert.Equal(t, exp, result, "GetEarnCouponList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetEarnCouponList should return a result")
}

func TestGetBYUSDTProductInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetBYUSDTProductInfo(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBYUSDTProductInfo must reject an empty coin")

	// Signed requests return account specific fields, so the method signs whenever credentials are set
	unsigned := testInstance(t)
	unsigned.SetCredentials(nil)
	for _, tc := range []struct {
		name string
		ex   *Exchange
	}{
		{name: "credentials", ex: e},
		{name: "no credentials", ex: unsigned},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := tc.ex.GetBYUSDTProductInfo(t.Context(), currency.NewCode("BYUSDT"))
			require.NoError(t, err, "GetBYUSDTProductInfo must not error")
			if mockTests {
				exp := &BYUSDTProductInfoResponse{
					ProductID:         "1",
					Coin:              currency.NewCode("BYUSDT"),
					MintFeeRateE8:     5000000,
					RedeemFeeRateE8:   100000,
					MinInvestment:     100,
					UserHolding:       588,
					LeftQuota:         999412,
					CanMint:           true,
					SavingsBalance:    1412,
					APRE8:             1890000,
					BonusAPRE8:        1000000,
					BonusMaxAmount:    100000,
					BaseCoinPrecision: 4,
					TokenPrecision:    4,
				}
				assert.Equal(t, exp, result, "GetBYUSDTProductInfo should decode every field")
				return
			}
			assert.NotEmpty(t, result.ProductID, "GetBYUSDTProductInfo should return a product ID")
			assert.True(t, result.Coin.Equal(currency.NewCode("BYUSDT")), "GetBYUSDTProductInfo should return the BYUSDT product")
		})
	}
}

func TestPlaceBYUSDTOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceBYUSDTOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceBYUSDTOrder must reject a nil request")
	req := &PlaceBYUSDTOrderRequest{}
	_, err = e.PlaceBYUSDTOrder(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceBYUSDTOrder must reject an empty coin")
	req.Coin = currency.NewCode("BYUSDT")
	_, err = e.PlaceBYUSDTOrder(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "PlaceBYUSDTOrder must reject an empty order link ID")
	req.OrderLinkID = "my-order-001"
	_, err = e.PlaceBYUSDTOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "PlaceBYUSDTOrder must reject an empty order type")
	req.OrderType = "Mint"
	_, err = e.PlaceBYUSDTOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceBYUSDTOrder must reject a zero amount")
	req.Amount = 100
	_, err = e.PlaceBYUSDTOrder(t.Context(), req)
	require.ErrorIs(t, err, errMissingAccountType, "PlaceBYUSDTOrder must reject an empty account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.AccountType = "FlexibleSaving"
	result, err := e.PlaceBYUSDTOrder(t.Context(), req)
	require.NoError(t, err, "PlaceBYUSDTOrder must not error")
	if mockTests {
		exp := &PlaceBYUSDTOrderResponse{
			OrderID:     "550e8400-e29b-41d4-a716-446655440000",
			OrderLinkID: "my-order-001",
		}
		assert.Equal(t, exp, result, "PlaceBYUSDTOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "PlaceBYUSDTOrder should return an order ID")
}

func TestGetBYUSDTOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetBYUSDTOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBYUSDTOrderList must reject a nil request")
	_, err = e.GetBYUSDTOrderList(t.Context(), &BYUSDTOrderListRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBYUSDTOrderList must reject an empty coin")
	_, err = e.GetBYUSDTOrderList(t.Context(), &BYUSDTOrderListRequest{Coin: currency.NewCode("BYUSDT"), StartTime: time.Unix(1775260800, 0), EndTime: time.Unix(1775088000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBYUSDTOrderList must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *BYUSDTOrderListRequest
		exp  *BYUSDTOrderListResponse
	}{
		{
			name: "list",
			req:  &BYUSDTOrderListRequest{Coin: currency.NewCode("BYUSDT"), OrderType: "Redeem", StartTime: time.Unix(1775088000, 0), EndTime: time.Unix(1775260800, 0), Limit: 20},
			exp: &BYUSDTOrderListResponse{
				List: []BYUSDTOrder{
					{
						OrderID:     "9985caf3-7b24-452e-bb28-afbc5b52805f",
						OrderLinkID: "byusdt-order-001",
						OrderType:   "Redeem",
						FromCoin:    currency.NewCode("BYUSDT"),
						ToCoin:      currency.USDT,
						FromAmount:  108,
						ToAmount:    107.9892,
						ServiceFee:  0.0108,
						FromAccount: "UNIFIED",
						ToAccount:   "FlexibleSaving",
						Status:      "Success",
						CreatedTime: types.Time(time.Unix(1775179628, 0)),
					},
					{
						OrderID:           "63581995-4e3b-4ffa-9a20-c8415e7ed558",
						OrderType:         "Redeem",
						FromCoin:          currency.NewCode("BYUSDT"),
						ToCoin:            currency.USDT,
						FromAmount:        12.5,
						ToAmount:          12.5,
						FromAccount:       "TRADFI",
						ToAccount:         "TRADFI",
						ExternalEventType: "RPL_CLOSE",
						Status:            "Processing",
						CreatedTime:       types.Time(time.Unix(1775177332, 0)),
					},
				},
				NextPageCursor: "eyJpZCI6NjM1ODE5OTV9",
			},
		},
		{
			name: "single order",
			req:  &BYUSDTOrderListRequest{Coin: currency.NewCode("BYUSDT"), OrderID: "9985caf3-7b24-452e-bb28-afbc5b52805f", OrderLinkID: "byusdt-order-001"},
			exp: &BYUSDTOrderListResponse{
				List: []BYUSDTOrder{
					{
						OrderID:     "9985caf3-7b24-452e-bb28-afbc5b52805f",
						OrderLinkID: "byusdt-order-001",
						OrderType:   "Mint",
						FromCoin:    currency.USDT,
						ToCoin:      currency.NewCode("BYUSDT"),
						FromAmount:  108,
						ToAmount:    108,
						FromAccount: "FlexibleSaving",
						ToAccount:   "UNIFIED",
						Status:      "Success",
						CreatedTime: types.Time(time.Unix(1775179628, 0)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetBYUSDTOrderList(t.Context(), tc.req)
			require.NoError(t, err, "GetBYUSDTOrderList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetBYUSDTOrderList should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetBYUSDTOrderList(t.Context(), tc.req)
			assert.NoError(t, err, "GetBYUSDTOrderList should not error for the next page")
		})
	}
}

func TestGetBYUSDTPositionInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetBYUSDTPositionInfo(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBYUSDTPositionInfo must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBYUSDTPositionInfo(t.Context(), currency.NewCode("BYUSDT"))
	require.NoError(t, err, "GetBYUSDTPositionInfo must not error")
	if mockTests {
		exp := &BYUSDTPositionInfoResponse{
			TotalAmount:    696,
			TotalYield:     1.2345,
			YesterdayYield: 0.1142,
			APRE8:          60000000,
			BonusAPRE8:     20000000,
			BonusMaxAmount: 1000,
			HasQuota:       true,
		}
		assert.Equal(t, exp, result, "GetBYUSDTPositionInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBYUSDTPositionInfo should return a result")
}

func TestGetBYUSDTDailyYieldRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetBYUSDTDailyYieldRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBYUSDTDailyYieldRecords must reject a nil request")
	_, err = e.GetBYUSDTDailyYieldRecords(t.Context(), &BYUSDTYieldRecordsRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBYUSDTDailyYieldRecords must reject an empty coin")
	_, err = e.GetBYUSDTDailyYieldRecords(t.Context(), &BYUSDTYieldRecordsRequest{Coin: currency.NewCode("BYUSDT"), StartTime: time.Unix(1775260800, 0), EndTime: time.Unix(1775088000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBYUSDTDailyYieldRecords must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	req := &BYUSDTYieldRecordsRequest{Coin: currency.NewCode("BYUSDT"), StartTime: time.Unix(1775088000, 0), EndTime: time.Unix(1775260800, 0), Limit: 5}
	result, err := e.GetBYUSDTDailyYieldRecords(t.Context(), req)
	require.NoError(t, err, "GetBYUSDTDailyYieldRecords must not error")
	if mockTests {
		exp := &BYUSDTDailyYieldRecordsResponse{
			List: []BYUSDTDailyYield{
				{
					Yield:       0.5,
					BonusYield:  0.2,
					Status:      "Success",
					CreatedTime: types.Time(time.Unix(1775088000, 0)),
				},
				{
					Yield:       0.48,
					Status:      "Processing",
					CreatedTime: types.Time(time.Unix(1775174400, 0)),
				},
			},
			NextPageCursor: "eyJpZCI6MTIzNDU2fQ==",
		}
		assert.Equal(t, exp, result, "GetBYUSDTDailyYieldRecords should decode every field")
	}
	if result.NextPageCursor == "" {
		return
	}
	req.Cursor = result.NextPageCursor
	_, err = e.GetBYUSDTDailyYieldRecords(t.Context(), req)
	assert.NoError(t, err, "GetBYUSDTDailyYieldRecords should not error for the next page")
}

func TestGetBYUSDTHourlyYieldRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetBYUSDTHourlyYieldRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBYUSDTHourlyYieldRecords must reject a nil request")
	_, err = e.GetBYUSDTHourlyYieldRecords(t.Context(), &BYUSDTYieldRecordsRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBYUSDTHourlyYieldRecords must reject an empty coin")
	_, err = e.GetBYUSDTHourlyYieldRecords(t.Context(), &BYUSDTYieldRecordsRequest{Coin: currency.NewCode("BYUSDT"), StartTime: time.Unix(1775260800, 0), EndTime: time.Unix(1775088000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBYUSDTHourlyYieldRecords must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	req := &BYUSDTYieldRecordsRequest{Coin: currency.NewCode("BYUSDT"), StartTime: time.Unix(1775088000, 0), EndTime: time.Unix(1775260800, 0), Limit: 20}
	result, err := e.GetBYUSDTHourlyYieldRecords(t.Context(), req)
	require.NoError(t, err, "GetBYUSDTHourlyYieldRecords must not error")
	if mockTests {
		exp := &BYUSDTHourlyYieldRecordsResponse{
			List: []BYUSDTHourlyYield{
				{
					EffectiveAmount: 500,
					Yield:           0.02,
					APRE8:           500000000,
					HourlyDate:      types.Time(time.Unix(1775088000, 0)),
					CreatedTime:     types.Time(time.Unix(1775091600, 0)),
				},
				{
					EffectiveAmount: 500,
					Yield:           0.01,
					RewardType:      1,
					APRE8:           200000000,
					HourlyDate:      types.Time(time.Unix(1775088000, 0)),
					CreatedTime:     types.Time(time.Unix(1775091600, 0)),
				},
			},
			NextPageCursor: "eyJpZCI6MTIzNDU3fQ==",
		}
		assert.Equal(t, exp, result, "GetBYUSDTHourlyYieldRecords should decode every field")
	}
	if result.NextPageCursor == "" {
		return
	}
	req.Cursor = result.NextPageCursor
	_, err = e.GetBYUSDTHourlyYieldRecords(t.Context(), req)
	assert.NoError(t, err, "GetBYUSDTHourlyYieldRecords should not error for the next page")
}

func TestGetBYUSDTHistoryAPR(t *testing.T) {
	t.Parallel()
	_, err := e.GetBYUSDTHistoryAPR(t.Context(), currency.EMPTYCODE, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetBYUSDTHistoryAPR must reject an empty coin")
	_, err = e.GetBYUSDTHistoryAPR(t.Context(), currency.NewCode("BYUSDT"), 0)
	require.ErrorIs(t, err, errAPRRangeRequired, "GetBYUSDTHistoryAPR must reject an empty range")

	result, err := e.GetBYUSDTHistoryAPR(t.Context(), currency.NewCode("BYUSDT"), 1)
	require.NoError(t, err, "GetBYUSDTHistoryAPR must not error")
	if mockTests {
		exp := &BYUSDTHistoryAPRResponse{
			List: []BYUSDTHistoryAPR{
				{
					Timestamp: types.Time(time.Unix(1791331200, 0)),
					APRE8:     2160000,
				},
				{
					Timestamp: types.Time(time.Unix(1791417600, 0)),
					APRE8:     2010000,
				},
			},
		}
		assert.Equal(t, exp, result, "GetBYUSDTHistoryAPR should decode every field")
		return
	}
	require.NotEmpty(t, result.List, "GetBYUSDTHistoryAPR must return APRs")
	assert.False(t, result.List[0].Timestamp.Time().IsZero(), "GetBYUSDTHistoryAPR should return a timestamp")
}

func TestGetFixedSavingProductInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetFixedSavingProductInfo(t.Context(), currency.USDT)
	require.NoError(t, err, "GetFixedSavingProductInfo must not error")
	if mockTests {
		exp := &FixedSavingProductInfoResponse{
			List: []FixedSavingProduct{
				{
					ProductID:               "4",
					Category:                "FundPool",
					Coin:                    currency.USDT,
					Duration:                "7d",
					Status:                  "SoldOut",
					TieredAPYList:           []FixedSavingTieredAPY{},
					MinStakeAmount:          500,
					MaxStakeAmount:          9000000,
					Precision:               4,
					SubscribeStartAt:        types.Time(time.Unix(1697019748, 0)),
					AllowEarlyRedemption:    true,
					EarlyRedemptionAPY:      "0.30%",
					RedemptionLimitDuration: "1d",
					AllowAutoReinvest:       true,
					InterestCoinAPYList: []FixedSavingInterestCoinAPY{
						{
							Coin:              currency.USDT,
							APY:               "2.50%",
							ExpectUnitEarning: 0.0004,
							CurrentPrice:      1,
						},
					},
					IsVIP:      true,
					CreditTime: types.Time(time.Unix(1792202700, 0)),
				},
				{
					ProductID: "427",
					Category:  "FixedTermSaving",
					Coin:      currency.USDT,
					Duration:  "30d",
					Status:    "Available",
					TieredAPYList: []FixedSavingTieredAPY{
						{
							Max: 500,
							APY: "5.00%",
						},
						{
							Min: 500,
							Max: -1,
							APY: "2.50%",
						},
					},
					MinStakeAmount:   50,
					MaxStakeAmount:   10000,
					Precision:        4,
					SubscribeStartAt: types.Time(time.Unix(1686704400, 0)),
					SubscribeEndAt:   types.Time(time.Unix(1909094399, 0)),
					InterestCoinAPYList: []FixedSavingInterestCoinAPY{
						{
							Coin:              currency.USDT,
							APY:               "2.50%",
							ExpectUnitEarning: 0.002,
							CurrentPrice:      1,
						},
					},
					CreditTime:               types.Time(time.Unix(1778718600, 0)),
					SpecialUserGroupRequired: true,
					SpecialUserGroupInfo:     "Limited Offer",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFixedSavingProductInfo should decode every field")
		return
	}
	require.NotEmpty(t, result.List, "GetFixedSavingProductInfo must return products")
	assert.True(t, result.List[0].Coin.Equal(currency.USDT), "GetFixedSavingProductInfo should return products of the coin")
}

func TestPlaceFixedSavingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceFixedSavingOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceFixedSavingOrder must reject a nil request")
	req := &PlaceFixedSavingOrderRequest{}
	_, err = e.PlaceFixedSavingOrder(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "PlaceFixedSavingOrder must reject an empty product ID")
	req.ProductID = "27"
	_, err = e.PlaceFixedSavingOrder(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "PlaceFixedSavingOrder must reject an empty category")
	req.Category = "FundPool"
	_, err = e.PlaceFixedSavingOrder(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceFixedSavingOrder must reject an empty coin")
	req.Coin = currency.USDT
	_, err = e.PlaceFixedSavingOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceFixedSavingOrder must reject a zero amount")
	req.Amount = 201
	_, err = e.PlaceFixedSavingOrder(t.Context(), req)
	require.ErrorIs(t, err, errMissingAccountType, "PlaceFixedSavingOrder must reject an empty account type")
	req.AccountType = "FUND"
	_, err = e.PlaceFixedSavingOrder(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "PlaceFixedSavingOrder must reject an empty order link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.OrderLinkID = "usdt-fixdearn-001"
	req.AutoInvest = true
	result, err := e.PlaceFixedSavingOrder(t.Context(), req)
	require.NoError(t, err, "PlaceFixedSavingOrder must not error")
	if mockTests {
		exp := &PlaceFixedSavingOrderResponse{
			OrderID:     "86468516-5497-4a2b-8c4c-f9c58e01e12c",
			OrderLinkID: "usdt-fixdearn-001",
		}
		assert.Equal(t, exp, result, "PlaceFixedSavingOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "PlaceFixedSavingOrder should return an order ID")
}

func TestRedeemFixedSaving(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemFixedSaving(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RedeemFixedSaving must reject a nil request")
	req := &RedeemFixedSavingRequest{}
	_, err = e.RedeemFixedSaving(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "RedeemFixedSaving must reject an empty product ID")
	req.ProductID = "27"
	_, err = e.RedeemFixedSaving(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "RedeemFixedSaving must reject an empty category")
	req.Category = "FundPool"
	_, err = e.RedeemFixedSaving(t.Context(), req)
	require.ErrorIs(t, err, errPositionIDRequired, "RedeemFixedSaving must reject an empty position ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.PositionID = "19454"
	result, err := e.RedeemFixedSaving(t.Context(), req)
	require.NoError(t, err, "RedeemFixedSaving must not error")
	if mockTests {
		exp := &RedeemFixedSavingResponse{
			RedeemAmount:      0.1,
			EstimatedEarnings: 0.00002191,
		}
		assert.Equal(t, exp, result, "RedeemFixedSaving should decode every field")
		return
	}
	assert.Positive(t, result.RedeemAmount.Float64(), "RedeemFixedSaving should return the redeemed amount")
}

func TestGetFixedSavingPositionInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedSavingPositionInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedSavingPositionInfo must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFixedSavingPositionInfo(t.Context(), &FixedSavingPositionInfoRequest{ProductID: "27", Category: "FundPool", Coin: currency.USDT})
	require.NoError(t, err, "GetFixedSavingPositionInfo must not error")
	if mockTests {
		exp := &FixedSavingPositionInfoResponse{
			List: []FixedSavingPosition{
				{
					PositionID:      "19454",
					ProductID:       "27",
					Category:        "FundPool",
					Coin:            currency.USDT,
					Amount:          201,
					EffectiveAmount: 200,
					Duration:        "3d",
					Status:          "Active",
					SettlementTime:  types.Time(time.Unix(1776385800, 0)),
					CreatedAt:       types.Time(time.Unix(1776068177, 0)),
					OrderID:         "86468516-5497-4a2b-8c4c-f9c58e01e12c",
					EarlyRedeemInfo: &FixedSavingEarlyRedeemInfo{
						AllowEarlyRedeem:        true,
						EarlyRedeemEarning:      0.0016,
						ReturnCoin:              currency.USDT,
						RedemptionLimitDuration: "1d",
					},
					AllowAutoReinvest: true,
					AutoReinvest:      "Enable",
					InterestCoinAPYList: []FixedSavingPositionInterestCoinAPY{
						{
							Coin:                currency.USDT,
							APY:                 "9.99%",
							ExpectReturnEarning: 0.1651,
							Price:               1,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetFixedSavingPositionInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFixedSavingPositionInfo should return a result")
}

func TestGetFixedSavingOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedSavingOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedSavingOrderList must reject a nil request")
	_, err = e.GetFixedSavingOrderList(t.Context(), &FixedSavingOrderListRequest{ProductID: "546"})
	require.ErrorIs(t, err, errCategoryNotSet, "GetFixedSavingOrderList must reject a product ID without a category")
	_, err = e.GetFixedSavingOrderList(t.Context(), &FixedSavingOrderListRequest{StartTime: time.UnixMilli(1750900000000), EndTime: time.UnixMilli(1750600000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFixedSavingOrderList must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *FixedSavingOrderListRequest
		exp  *FixedSavingOrderListResponse
	}{
		{
			name: "list",
			req:  &FixedSavingOrderListRequest{OrderType: "Redeem", ProductID: "546", Category: "FixedTermSaving", StartTime: time.UnixMilli(1750600000000), EndTime: time.UnixMilli(1750900000000), Limit: 20},
			exp: &FixedSavingOrderListResponse{
				List: []FixedSavingOrder{
					{
						OrderID:        "6f2530d6-46b9-41f9-880a-4addbd152398",
						OrderLinkID:    "usdt-fixedearn-redeem-001",
						OrderType:      "Redeem",
						Status:         "Complete",
						ProductID:      "546",
						Category:       "FixedTermSaving",
						Coin:           currency.USDT,
						Amount:         100.056,
						Duration:       "1d",
						AccountType:    "FUND",
						SettlementTime: types.Time(time.Unix(1750811400, 0)),
						CreatedAt:      types.Time(time.Unix(1750648976, 0)),
						YieldInfoList: []FixedSavingYieldInfo{
							{
								Coin:      currency.USDT,
								Amount:    0.0063,
								Status:    "Distributed",
								CreatedAt: types.Time(time.Unix(1750811401, 0)),
								APY:       "2.33%",
							},
						},
					},
				},
				NextPageCursor: "eyJpZCI6NjU0MzIxfQ==",
			},
		},
		{
			name: "single order",
			req:  &FixedSavingOrderListRequest{OrderID: "86468516-5497-4a2b-8c4c-f9c58e01e12c"},
			exp: &FixedSavingOrderListResponse{
				List: []FixedSavingOrder{
					{
						OrderID:        "86468516-5497-4a2b-8c4c-f9c58e01e12c",
						OrderLinkID:    "usdt-fixdearn-001",
						OrderType:      "Stake",
						Status:         "Active",
						ProductID:      "27",
						Category:       "FundPool",
						Coin:           currency.USDT,
						Amount:         201,
						Duration:       "3d",
						AccountType:    "UNIFIED",
						SettlementTime: types.Time(time.Unix(1776385800, 0)),
						CreatedAt:      types.Time(time.Unix(1776068177, 0)),
						YieldInfoList:  []FixedSavingYieldInfo{},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFixedSavingOrderList(t.Context(), tc.req)
			require.NoError(t, err, "GetFixedSavingOrderList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFixedSavingOrderList should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetFixedSavingOrderList(t.Context(), tc.req)
			assert.NoError(t, err, "GetFixedSavingOrderList should not error for the next page")
		})
	}
}

func TestSetFixedSavingAutoInvest(t *testing.T) {
	t.Parallel()
	err := e.SetFixedSavingAutoInvest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetFixedSavingAutoInvest must reject a nil request")
	req := &SetFixedSavingAutoInvestRequest{}
	err = e.SetFixedSavingAutoInvest(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "SetFixedSavingAutoInvest must reject an empty product ID")
	req.ProductID = "27"
	err = e.SetFixedSavingAutoInvest(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "SetFixedSavingAutoInvest must reject an empty category")
	req.Category = "FundPool"
	err = e.SetFixedSavingAutoInvest(t.Context(), req)
	require.ErrorIs(t, err, errPositionIDRequired, "SetFixedSavingAutoInvest must reject an empty position ID")
	req.PositionID = "19454"
	err = e.SetFixedSavingAutoInvest(t.Context(), req)
	require.ErrorIs(t, err, errAutoInvestStatusRequired, "SetFixedSavingAutoInvest must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.Status = "Enable"
	err = e.SetFixedSavingAutoInvest(t.Context(), req)
	assert.NoError(t, err, "SetFixedSavingAutoInvest should not error")
}

func TestGetFlexibleSavingAutoSavings(t *testing.T) {
	t.Parallel()
	_, err := e.GetFlexibleSavingAutoSavings(t.Context(), []currency.Code{currency.BTC, currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFlexibleSavingAutoSavings must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name  string
		coins []currency.Code
		exp   *FlexibleSavingAutoSavingsResponse
	}{
		{
			name: "all coins",
			exp: &FlexibleSavingAutoSavingsResponse{
				SelectedAll: true,
				Coins: []FlexibleSavingAutoSavingsCoin{
					{
						Coin:             currency.BTC,
						IsSelected:       true,
						MaxStakingAmount: 10,
						APR:              0.05,
					},
					{
						Coin:             currency.USDT,
						IsSelected:       true,
						MaxStakingAmount: 100000,
						APR:              0.064,
					},
				},
			},
		},
		{
			name:  "BTC and ETH",
			coins: []currency.Code{currency.BTC, currency.ETH},
			exp: &FlexibleSavingAutoSavingsResponse{
				Coins: []FlexibleSavingAutoSavingsCoin{
					{
						Coin:             currency.BTC,
						IsSelected:       true,
						MaxStakingAmount: 10,
						APR:              0.05,
					},
					{
						Coin:             currency.ETH,
						MaxStakingAmount: 100,
						APR:              0.04,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFlexibleSavingAutoSavings(t.Context(), tc.coins)
			require.NoError(t, err, "GetFlexibleSavingAutoSavings must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFlexibleSavingAutoSavings should decode every field")
				return
			}
			assert.NotEmpty(t, result.Coins, "GetFlexibleSavingAutoSavings should return coins")
		})
	}
}

func TestSetFlexibleSavingAutoSavings(t *testing.T) {
	t.Parallel()
	err := e.SetFlexibleSavingAutoSavings(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetFlexibleSavingAutoSavings must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.SetFlexibleSavingAutoSavings(t.Context(), &SetFlexibleSavingAutoSavingsRequest{Coin: currency.BTC, IsSelected: true, PurchaseImmediately: true})
	assert.NoError(t, err, "SetFlexibleSavingAutoSavings should not error")
}

func TestGetAirdropProducts(t *testing.T) {
	t.Parallel()
	// Signed requests return account specific fields, so the method signs whenever credentials are set
	unsigned := testInstance(t)
	unsigned.SetCredentials(nil)
	for _, tc := range []struct {
		name string
		ex   *Exchange
	}{
		{name: "credentials", ex: e},
		{name: "no credentials", ex: unsigned},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := tc.ex.GetAirdropProducts(t.Context())
			require.NoError(t, err, "GetAirdropProducts must not error")
			if mockTests {
				exp := &AirdropProductsResponse{
					Products: []AirdropProduct{
						{
							CoinName: currency.NewCode("RLUSD"),
							Yields: []AirdropYield{
								{
									CoinName:    currency.XRP,
									APY:         "2.160508%",
									PersonalAPY: "4.321016%",
									Multiplier:  2,
								},
								{
									CoinName:    currency.NewCode("RLUSD"),
									APY:         "3.75%",
									PersonalAPY: "3.75%",
									Multiplier:  1,
								},
							},
							Status:          "Online",
							APY:             "5.910508%",
							PersonalAPY:     "8.071016%",
							AnnouncementURL: "https://bybit.com/en/earn/hold-to-earn/rlusd",
						},
						{
							CoinName: currency.NewCode("USD1"),
							Yields: []AirdropYield{
								{
									CoinName:    currency.NewCode("WLFI"),
									APY:         "6.27778%",
									PersonalAPY: "6.27778%",
									Multiplier:  1,
								},
							},
							Status:          "Ended",
							APY:             "6.27778%",
							PersonalAPY:     "6.27778%",
							AnnouncementURL: "https://www.bybit.com/en/earn/usd1-page",
						},
					},
				}
				assert.Equal(t, exp, result, "GetAirdropProducts should decode every field")
				return
			}
			require.NotEmpty(t, result.Products, "GetAirdropProducts must return products")
			assert.NotEmpty(t, result.Products[0].Yields, "GetAirdropProducts should return yield coins")
		})
	}
}

func TestGetAirdropDailyPNLRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetAirdropDailyPNLRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAirdropDailyPNLRecords must reject a nil request")
	_, err = e.GetAirdropDailyPNLRecords(t.Context(), &AirdropDailyPNLRecordsRequest{})
	require.ErrorIs(t, err, errLimitRequired, "GetAirdropDailyPNLRecords must reject an empty limit")
	_, err = e.GetAirdropDailyPNLRecords(t.Context(), &AirdropDailyPNLRecordsRequest{Limit: 20, TimeStart: time.Unix(1747728000, 0), TimeEnd: time.Unix(1747468800, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAirdropDailyPNLRecords must reject a start time after the end time")

	start, end := time.Unix(1747468800, 0), time.Unix(1747728000, 0)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		// A start time more than three months ago is rejected
		end = time.Now()
		start = end.AddDate(0, 0, -3)
	}
	req := &AirdropDailyPNLRecordsRequest{TimeStart: start, TimeEnd: end, Limit: 20}
	result, err := e.GetAirdropDailyPNLRecords(t.Context(), req)
	require.NoError(t, err, "GetAirdropDailyPNLRecords must not error")
	if mockTests {
		exp := &AirdropDailyPNLRecordsResponse{
			NextCursor: "eyJsYXN0SWQiOjEwMDE5MH0=",
			AirdropDailyPNLs: []AirdropDailyPNL{
				{
					CoinName:        currency.NewCode("USDE"),
					YieldCoinName:   currency.NewCode("USDE"),
					EffectiveAmount: 10000,
					PNL:             0.2739726,
					APY:             "10%",
					CreatedAt:       types.Time(time.Unix(1747641600, 0)),
				},
				{
					CoinName:        currency.NewCode("USD1"),
					YieldCoinName:   currency.NewCode("WLFI"),
					EffectiveAmount: 5000,
					PNL:             0.85994246,
					APY:             "6.27778%",
					CreatedAt:       types.Time(time.Unix(1747555200, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetAirdropDailyPNLRecords should decode every field")
	}
	if result.NextCursor == "" {
		return
	}
	req.Cursor = result.NextCursor
	_, err = e.GetAirdropDailyPNLRecords(t.Context(), req)
	assert.NoError(t, err, "GetAirdropDailyPNLRecords should not error for the next page")
}

func TestEarnYieldHistoryResponseUnmarshal(t *testing.T) {
	t.Parallel()
	exp := EarnYieldHistoryResponse{
		NextPageCursor: "eyJpZCI6MTAwMjA5N30=",
		Yield: []EarnYield{
			{
				ProductID:              "428",
				Coin:                   currency.USDT,
				ID:                     "1002096",
				Amount:                 0.0608,
				YieldType:              "Normal",
				DistributionMode:       "Manual",
				EffectiveStakingAmount: 1000,
				OrderID:                "05a7012d-c4d6-493a-8c6b-023a1038944a",
				Status:                 "Success",
				CreatedAt:              types.Time(time.UnixMilli(1759993805000)),
			},
		},
	}
	row := `{"productId":"428","coin":"USDT","id":"1002096","amount":"0.0608","yieldType":"Normal","distributionMode":"Manual","effectiveStakingAmount":"1000","orderId":"05a7012d-c4d6-493a-8c6b-023a1038944a","status":"Success","createdAt":"1759993805000"}`
	// The docs' response example names the yields yield and their table list
	for _, key := range []string{"yield", "list"} {
		var result EarnYieldHistoryResponse
		require.NoErrorf(t, json.Unmarshal([]byte(`{"`+key+`":[`+row+`],"nextPageCursor":"eyJpZCI6MTAwMjA5N30="}`), &result), "Unmarshal must not error for %s", key)
		assert.Equalf(t, exp, result, "EarnYieldHistoryResponse should decode the yields from %s", key)
	}
	// An empty yield is still the response's yields, whatever list holds
	var emptyYield EarnYieldHistoryResponse
	require.NoError(t, json.Unmarshal([]byte(`{"yield":[],"list":[`+row+`]}`), &emptyYield), "Unmarshal must not error for an empty yield")
	assert.Empty(t, emptyYield.Yield, "EarnYieldHistoryResponse should take an empty yield over list")
	// Yields under yield win over yields under list, and a response decoded into a used one replaces its yields
	reused := EarnYieldHistoryResponse{Yield: []EarnYield{{ID: "old"}}}
	require.NoError(t, json.Unmarshal([]byte(`{"yield":[`+row+`],"list":[{"id":"other"}],"nextPageCursor":"eyJpZCI6MTAwMjA5N30="}`), &reused), "Unmarshal must not error for both keys")
	assert.Equal(t, exp, reused, "EarnYieldHistoryResponse should take the yields under yield when both keys are sent")
	reused.Yield = []EarnYield{{ID: "old"}}
	require.NoError(t, json.Unmarshal([]byte(`{"list":[`+row+`],"nextPageCursor":"eyJpZCI6MTAwMjA5N30="}`), &reused), "Unmarshal must not error for a used response")
	assert.Equal(t, exp, reused, "EarnYieldHistoryResponse should replace a used response's yields with those under list")
	require.NoError(t, json.Unmarshal([]byte(`{"nextPageCursor":""}`), &reused), "Unmarshal must not error without either key")
	assert.Equal(t, exp.Yield, reused.Yield, "EarnYieldHistoryResponse should keep its yields when the response sends neither key")
	assert.Error(t, json.Unmarshal([]byte(`{"list":"bad"}`), &reused), "Unmarshal should reject yields that are not a list")
}
