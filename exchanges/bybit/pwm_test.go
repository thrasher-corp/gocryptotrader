package bybit

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

func TestTransferPWMFunds(t *testing.T) {
	t.Parallel()
	_, err := e.TransferPWMFunds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "TransferPWMFunds must reject a nil request")
	req := &PWMFundTransferRequest{}
	_, err = e.TransferPWMFunds(t.Context(), req)
	require.ErrorIs(t, err, errMissingTransferID, "TransferPWMFunds must reject an empty transfer ID")
	req.TransferID = "4fdf-re-4343-frewr"
	_, err = e.TransferPWMFunds(t.Context(), req)
	require.ErrorIs(t, err, errMissingUserID, "TransferPWMFunds must reject an empty source user ID")
	req.FromUserID = 800001
	_, err = e.TransferPWMFunds(t.Context(), req)
	require.ErrorIs(t, err, errMissingUserID, "TransferPWMFunds must reject an empty destination user ID")
	req.ToUserID = 800002
	_, err = e.TransferPWMFunds(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "TransferPWMFunds must reject a zero amount")
	req.Amount = 1
	_, err = e.TransferPWMFunds(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "TransferPWMFunds must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.Coin = currency.BTC
	result, err := e.TransferPWMFunds(t.Context(), req)
	require.NoError(t, err, "TransferPWMFunds must not error")
	if mockTests {
		exp := &PWMFundTransferResponse{
			TransferID: "4fdf-re-4343-frewr",
			Status:     "PROCESSING",
		}
		assert.Equal(t, exp, result, "TransferPWMFunds should decode every field")
		return
	}
	assert.Equal(t, req.TransferID, result.TransferID, "TransferPWMFunds should return the transfer ID")
}

func TestGetPWMFundTransferRecords(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPWMFundTransferRecords(t.Context(), "4fdf-re-4343-frewr", 800001)
	require.NoError(t, err, "GetPWMFundTransferRecords must not error")
	if mockTests {
		exp := &PWMFundTransferRecordsResponse{
			TransferID: "4fdf-re-4343-frewr",
			Status:     "SUCCESS",
			FromUserID: 800001,
			ToUserID:   800002,
			Amount:     1,
			Coin:       currency.BTC,
		}
		assert.Equal(t, exp, result, "GetPWMFundTransferRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPWMFundTransferRecords should return a result")
}

func TestGetAllPWMInvestmentPlans(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPWMInvestmentPlans(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllPWMInvestmentPlans must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	req := &PWMInvestmentPlansRequest{PlanID: "10001", Status: "Active", Limit: 20}
	result, err := e.GetAllPWMInvestmentPlans(t.Context(), req)
	require.NoError(t, err, "GetAllPWMInvestmentPlans must not error")
	if mockTests {
		exp := &PWMInvestmentPlansResponse{
			List: []PWMInvestmentPlan{
				{
					PlanID:             "10001",
					PlanName:           "Conservative Growth Plan",
					PlanType:           "stable",
					Status:             "Active",
					Source:             "consultant",
					CurrentAssetUSD:    200137.5,
					AccumulateYieldUSD: 2137.5,
					InvestmentDistribution: []PWMInvestmentDistribution{
						{
							Category:      "fixedYield",
							Coin:          currency.USDT,
							ProductID:     "430",
							CurrentAmount: 50000,
						},
						{
							Category:      "equityFund",
							Coin:          currency.USDT,
							ProductID:     "2001",
							CurrentAmount: 150137.5,
						},
					},
					CreatedTime: types.Time(time.Unix(1700000000, 0)),
				},
			},
			NextPageCursor: "eyJpZCI6MTAwMDF9",
		}
		assert.Equal(t, exp, result, "GetAllPWMInvestmentPlans should decode every field")
	}
	if result.NextPageCursor == "" {
		return
	}
	req.Cursor = result.NextPageCursor
	_, err = e.GetAllPWMInvestmentPlans(t.Context(), req)
	assert.NoError(t, err, "GetAllPWMInvestmentPlans should not error for the next page")
}

func TestGetPWMInvestmentPlanDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetPWMInvestmentPlanDetail(t.Context(), "")
	require.ErrorIs(t, err, errPlanIDRequired, "GetPWMInvestmentPlanDetail must reject an empty plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPWMInvestmentPlanDetail(t.Context(), "10001")
	require.NoError(t, err, "GetPWMInvestmentPlanDetail must not error")
	if mockTests {
		exp := &PWMInvestmentPlanDetailResponse{
			PlanID:             "10001",
			PlanName:           "Conservative Growth Plan",
			PlanType:           "stable",
			Status:             "Active",
			CurrentAssetUSD:    200137.5,
			AccumulateYieldUSD: 2137.5,
			WeightedAverageAPR: 0.086,
			CurrentAssets: []PWMCoinAmount{
				{
					Coin:   currency.USDT,
					Amount: 150000,
				},
			},
			Positions: PWMPlanPositions{
				MultiCoinsEarning: PWMMultiCoinsEarningPositions{
					TotalInvestmentUSD: 60000,
					AccumulateYieldUSD: 500,
					WeightedAverageAPR: 0.052,
					Items: []PWMMultiCoinsEarningPosition{
						{
							Category:        "flexibleSavings",
							ProductID:       "430",
							Coin:            currency.USDT,
							CurrentAmount:   30000,
							AccumulateYield: 250,
							APR:             0.05,
							PositionID:      "121",
							Status:          "1",
						},
					},
				},
				FixedYield: PWMFixedYieldPositions{
					TotalInvestmentUSD: 50000,
					AccumulateYieldUSD: 800,
					WeightedAverageAPR: 0.08,
					Items: []PWMFixedYieldPosition{
						{
							Category:        "fundPool",
							ProductID:       "430",
							Coin:            currency.USDT,
							CurrentAmount:   50000,
							AccumulateYield: 800,
							APR:             0.08,
							Duration:        30,
							MaturityTime:    types.Time(time.Unix(1700500000, 0)),
							AutoReinvest:    true,
							PositionID:      "122",
							Status:          "1",
						},
					},
				},
				EquityFunds: PWMEquityFundPositions{
					TotalInvestmentUSD: 70000,
					AccumulateYieldUSD: 700,
					WeightedAverageAPR: 0.12,
					Items: []PWMEquityFundPosition{
						{
							Category:  "equityFund",
							ProductID: "2001",
							FundName:  "Market Neutral Alpha",
							Coin:      currency.USDT,
							Tags: []string{
								"Delta Neutral",
								"Funding Rate",
							},
							NAV:             1.035,
							UserShares:      68000,
							ShareValue:      1.029,
							HoldingValue:    69972,
							AccumulateYield: 700,
							APR30Day:        0.12,
							APRTotal:        0.105,
							SharpRatio:      2.5,
							MaxDrawdown:     -0.032,
							CreatedTime:     types.Time(time.Unix(1695000000, 0)),
							RunningDays:     58,
							PositionID:      "123",
							Status:          "1",
						},
					},
				},
				OnchainEarn: PWMOnchainEarnPositions{
					TotalInvestmentUSD: 18000,
					AccumulateYieldUSD: 137.5,
					Items: []PWMOnchainEarnPosition{
						{
							Category:    "onchainEarn",
							ProductID:   "25",
							Coin:        currency.USDT,
							StakeAmount: 18000,
							APR:         0.075,
							PositionID:  "124",
							Status:      "2",
						},
					},
				},
				FundingAccount: []PWMCoinAmount{
					{
						Coin:   currency.USDT,
						Amount: 165.5,
					},
				},
			},
			CreatedTime: types.Time(time.Unix(1700000000, 0)),
		}
		assert.Equal(t, exp, result, "GetPWMInvestmentPlanDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPWMInvestmentPlanDetail should return a result")
}

func TestGetPendingPWMInvestmentPlanDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetPendingPWMInvestmentPlanDetail(t.Context(), "")
	require.ErrorIs(t, err, errPlanIDRequired, "GetPendingPWMInvestmentPlanDetail must reject an empty plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPendingPWMInvestmentPlanDetail(t.Context(), "10003")
	require.NoError(t, err, "GetPendingPWMInvestmentPlanDetail must not error")
	if mockTests {
		exp := &PWMPendingInvestmentPlanDetailResponse{
			PlanID:                   "10003",
			PlanName:                 "Aggressive Growth Plan",
			PlanType:                 "advanced",
			Status:                   "PendingSubscription",
			Source:                   "consultant",
			TotalConfiguredAmountUSD: 500000,
			Products: PWMPendingPlanProducts{
				MultiCoinsEarning: PWMPendingProductGroup{
					ConfiguredAmountUSD: 100000,
					Items: []PWMPendingProductConfig{
						{
							Category:         "flexibleSavings",
							ProductID:        "430",
							Coin:             currency.USDT,
							ConfiguredAmount: 100000,
							APR:              0.05,
						},
					},
				},
				FixedYield: PWMPendingFixedYieldGroup{
					ConfiguredAmountUSD: 80000,
					Items: []PWMPendingFixedYieldConfig{
						{
							Category:         "fundPoolPremium",
							ProductID:        "431",
							Coin:             currency.USDT,
							ConfiguredAmount: 80000,
							APR:              0.08,
							Duration:         30,
						},
					},
				},
				EquityFunds: PWMPendingEquityFundGroup{
					ConfiguredAmountUSD: 270000,
					Items: []PWMPendingEquityFundConfig{
						{
							Category:         "equityFund",
							ProductID:        "2005",
							FundName:         "Market Neutral Alpha",
							Coin:             currency.USDT,
							ConfiguredAmount: 270000,
							Tags: []string{
								"Delta Neutral",
								"Funding Rate",
							},
							Introduction: PWMFundIntroduction{
								Description:            "A market-neutral strategy fund that generates stable returns through long-short hedging",
								HistoricalYieldRateMax: 0.15,
								HistoricalYieldRateMin: 0.08,
								SharpRatio:             2.3,
								MaxDrawback:            -0.035,
								LockupPeriod:           "30 days",
							},
						},
					},
				},
				OnchainEarn: PWMPendingProductGroup{
					ConfiguredAmountUSD: 50000,
					Items: []PWMPendingProductConfig{
						{
							Category:         "onchainEarn",
							ProductID:        "25",
							Coin:             currency.USDT,
							ConfiguredAmount: 50000,
							APR:              0.075,
						},
					},
				},
			},
			CreatedTime: types.Time(time.Unix(1700800000, 0)),
		}
		assert.Equal(t, exp, result, "GetPendingPWMInvestmentPlanDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPendingPWMInvestmentPlanDetail should return a result")
}

func TestClaimPWMWithdrawableFunds(t *testing.T) {
	t.Parallel()
	_, err := e.ClaimPWMWithdrawableFunds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ClaimPWMWithdrawableFunds must reject a nil request")
	req := &PWMClaimRequest{}
	_, err = e.ClaimPWMWithdrawableFunds(t.Context(), req)
	require.ErrorIs(t, err, errPlanIDRequired, "ClaimPWMWithdrawableFunds must reject an empty plan ID")
	req.PlanID = "10001"
	_, err = e.ClaimPWMWithdrawableFunds(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "ClaimPWMWithdrawableFunds must reject an empty order link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.OrderLinkID = "claim-order-001"
	req.ToAccountType = "FUND"
	result, err := e.ClaimPWMWithdrawableFunds(t.Context(), req)
	require.NoError(t, err, "ClaimPWMWithdrawableFunds must not error")
	if mockTests {
		exp := &PWMClaimResponse{
			PlanID:        "10001",
			ToAccountType: 6,
			Status:        "Success",
			CreatedTime:   types.Time(time.Unix(1701400000, 0)),
		}
		assert.Equal(t, exp, result, "ClaimPWMWithdrawableFunds should decode every field")
		return
	}
	assert.Equal(t, req.PlanID, result.PlanID, "ClaimPWMWithdrawableFunds should return the plan ID")
}

func TestGetPWMAssetTrend(t *testing.T) {
	t.Parallel()
	_, err := e.GetPWMAssetTrend(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPWMAssetTrend must reject a nil request")
	_, err = e.GetPWMAssetTrend(t.Context(), &PWMAssetTrendRequest{})
	require.ErrorIs(t, err, errPlanIDRequired, "GetPWMAssetTrend must reject an empty plan ID")
	_, err = e.GetPWMAssetTrend(t.Context(), &PWMAssetTrendRequest{PlanID: "10001", StartTime: time.UnixMilli(1730592000000), EndTime: time.UnixMilli(1730419200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPWMAssetTrend must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPWMAssetTrend(t.Context(), &PWMAssetTrendRequest{PlanID: "10001", StartTime: time.UnixMilli(1730419200000), EndTime: time.UnixMilli(1730592000000)})
	require.NoError(t, err, "GetPWMAssetTrend must not error")
	if mockTests {
		exp := &PWMAssetTrendResponse{
			PlanID: "10001",
			DataPoints: []PWMAssetTrendPoint{
				{
					Date:          "2024-11-01",
					AssetValueUSD: 198500,
				},
				{
					Date:          "2024-11-02",
					AssetValueUSD: 199100,
				},
			},
		}
		assert.Equal(t, exp, result, "GetPWMAssetTrend should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPWMAssetTrend should return a result")
}

func TestGetPWMFundHistoricalNAV(t *testing.T) {
	t.Parallel()
	_, err := e.GetPWMFundHistoricalNAV(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPWMFundHistoricalNAV must reject a nil request")
	_, err = e.GetPWMFundHistoricalNAV(t.Context(), &PWMFundHistoricalNAVRequest{})
	require.ErrorIs(t, err, errFundIDRequired, "GetPWMFundHistoricalNAV must reject an empty fund ID")
	_, err = e.GetPWMFundHistoricalNAV(t.Context(), &PWMFundHistoricalNAVRequest{FundID: "2001", StartTime: time.UnixMilli(1730592000000), EndTime: time.UnixMilli(1730419200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPWMFundHistoricalNAV must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPWMFundHistoricalNAV(t.Context(), &PWMFundHistoricalNAVRequest{FundID: "2001", StartTime: time.UnixMilli(1730419200000), EndTime: time.UnixMilli(1730592000000)})
	require.NoError(t, err, "GetPWMFundHistoricalNAV must not error")
	if mockTests {
		exp := &PWMFundHistoricalNAVResponse{
			FundID:     "2001",
			FundName:   "Market Neutral Alpha",
			Coin:       currency.USDT,
			CurrentNAV: 1.035,
			DataPoints: []PWMFundNAVPoint{
				{
					Date: "2024-11-01",
					NAV:  1.028,
				},
				{
					Date: "2024-11-02",
					NAV:  1.031,
				},
			},
		}
		assert.Equal(t, exp, result, "GetPWMFundHistoricalNAV should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPWMFundHistoricalNAV should return a result")
}

func TestSubscribePWMInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribePWMInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubscribePWMInvestmentPlan must reject a nil request")
	req := &PWMSubscribeRequest{}
	_, err = e.SubscribePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanIDRequired, "SubscribePWMInvestmentPlan must reject an empty plan ID")
	req.PlanID = "10003"
	_, err = e.SubscribePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "SubscribePWMInvestmentPlan must reject an empty order link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.OrderLinkID = "pwm-subscribe-001"
	req.AccountType = "FUND"
	result, err := e.SubscribePWMInvestmentPlan(t.Context(), req)
	require.NoError(t, err, "SubscribePWMInvestmentPlan must not error")
	if mockTests {
		exp := &PWMSubscribeResponse{
			PlanID:      "10003",
			Status:      "Active",
			OrderLinkID: "pwm-subscribe-001",
		}
		assert.Equal(t, exp, result, "SubscribePWMInvestmentPlan should decode every field")
		return
	}
	assert.Equal(t, req.PlanID, result.PlanID, "SubscribePWMInvestmentPlan should return the plan ID")
}

func TestInvestMoreInPWMInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.InvestMoreInPWMInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "InvestMoreInPWMInvestmentPlan must reject a nil request")
	req := &PWMInvestMoreRequest{}
	_, err = e.InvestMoreInPWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanIDRequired, "InvestMoreInPWMInvestmentPlan must reject an empty plan ID")
	req.PlanID = "10001"
	_, err = e.InvestMoreInPWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "InvestMoreInPWMInvestmentPlan must reject an empty category")
	req.Category = "equityFund"
	_, err = e.InvestMoreInPWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "InvestMoreInPWMInvestmentPlan must reject an empty product ID")
	req.ProductID = "2001"
	_, err = e.InvestMoreInPWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "InvestMoreInPWMInvestmentPlan must reject a zero amount")
	req.Amount = 20000
	_, err = e.InvestMoreInPWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "InvestMoreInPWMInvestmentPlan must reject an empty order link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.OrderLinkID = "pwm-invest-more-001"
	req.AccountType = "FUND"
	result, err := e.InvestMoreInPWMInvestmentPlan(t.Context(), req)
	require.NoError(t, err, "InvestMoreInPWMInvestmentPlan must not error")
	if mockTests {
		exp := &PWMInvestMoreResponse{
			PlanID:      "10001",
			Category:    "equityFund",
			ProductID:   "2001",
			Coin:        currency.USDT,
			Amount:      20000,
			Status:      "Pending",
			OrderLinkID: "pwm-invest-more-001",
			OrderID:     "ORD20241115002",
		}
		assert.Equal(t, exp, result, "InvestMoreInPWMInvestmentPlan should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "InvestMoreInPWMInvestmentPlan should return an order ID")
}

func TestRedeemPWMInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemPWMInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RedeemPWMInvestmentPlan must reject a nil request")
	req := &PWMRedeemRequest{}
	_, err = e.RedeemPWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanIDRequired, "RedeemPWMInvestmentPlan must reject an empty plan ID")
	req.PlanID = "10001"
	_, err = e.RedeemPWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "RedeemPWMInvestmentPlan must reject an empty category")
	req.Category = "equityFund"
	_, err = e.RedeemPWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "RedeemPWMInvestmentPlan must reject an empty product ID")
	req.ProductID = "2001"
	_, err = e.RedeemPWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "RedeemPWMInvestmentPlan must reject a request without shares or an amount")
	req.Shares = 3000
	_, err = e.RedeemPWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "RedeemPWMInvestmentPlan must reject an empty order link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PWMRedeemRequest
		exp  *PWMRedeemResponse
	}{
		{
			name: "fund shares",
			req:  &PWMRedeemRequest{PlanID: "10001", Category: "equityFund", ProductID: "2001", Shares: 3000, OrderLinkID: "pwm-redeem-001"},
			exp: &PWMRedeemResponse{
				OrderID:         "ORD20241115003",
				PlanID:          "10001",
				Category:        "equityFund",
				ProductID:       "2001",
				Shares:          3000,
				EstimatedAmount: 3087,
				Coin:            currency.USDT,
				Status:          "Pending",
				OrderLinkID:     "pwm-redeem-001",
			},
		},
		{
			name: "product amount",
			req:  &PWMRedeemRequest{PlanID: "10001", Category: "fundPool", ProductID: "430", Amount: 5000, OrderLinkID: "pwm-redeem-002", PositionID: "122"},
			exp: &PWMRedeemResponse{
				OrderID:         "ORD20241115004",
				PlanID:          "10001",
				Category:        "fundPool",
				ProductID:       "430",
				Amount:          5000,
				EstimatedAmount: 5000,
				Coin:            currency.USDT,
				Status:          "Success",
				OrderLinkID:     "pwm-redeem-002",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.RedeemPWMInvestmentPlan(t.Context(), tc.req)
			require.NoError(t, err, "RedeemPWMInvestmentPlan must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "RedeemPWMInvestmentPlan should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderID, "RedeemPWMInvestmentPlan should return an order ID")
		})
	}
}

func TestGetPWMInvestmentPlanOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetPWMInvestmentPlanOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPWMInvestmentPlanOrders must reject a nil request")
	_, err = e.GetPWMInvestmentPlanOrders(t.Context(), &PWMInvestmentPlanOrdersRequest{StartTime: time.UnixMilli(1739520000000), EndTime: time.UnixMilli(1739433600000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPWMInvestmentPlanOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	req := &PWMInvestmentPlanOrdersRequest{
		PlanID:      "10001",
		Category:    "flexibleSavings",
		Type:        "Subscribe",
		Status:      "Completed",
		StartTime:   time.UnixMilli(1739433600000),
		EndTime:     time.UnixMilli(1739520000000),
		Limit:       20,
		OrderLinkID: "pwm-subscribe-001",
	}
	result, err := e.GetPWMInvestmentPlanOrders(t.Context(), req)
	require.NoError(t, err, "GetPWMInvestmentPlanOrders must not error")
	if mockTests {
		exp := &PWMInvestmentPlanOrdersResponse{
			List: []PWMInvestmentPlanOrder{
				{
					OrderID:     "a285b975-9968-4ba0-bd78-58430ead715f",
					PlanID:      "10001",
					Type:        "Subscribe",
					AccountType: "Funding",
					Coin:        currency.USDT,
					Amount:      100,
					Category:    "flexibleSavings",
					ProductID:   "11123",
					Status:      "Completed",
					OrderTime:   types.Time(time.Unix(1739519687, 0)),
				},
			},
			NextPageCursor: "eyJpZCI6Mjg1fQ==",
		}
		assert.Equal(t, exp, result, "GetPWMInvestmentPlanOrders should decode every field")
	}
	if result.NextPageCursor == "" {
		return
	}
	req.Cursor = result.NextPageCursor
	_, err = e.GetPWMInvestmentPlanOrders(t.Context(), req)
	assert.NoError(t, err, "GetPWMInvestmentPlanOrders should not error for the next page")
}

func TestGetPWMSubscribableProductInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetPWMSubscribableProductInfo(t.Context())
	require.NoError(t, err, "GetPWMSubscribableProductInfo must not error")
	if mockTests {
		exp := &PWMSubscribableProductInfoResponse{
			Products: []PWMProductGroup{
				{
					Type: "equityFund",
					Cards: []PWMProductCard{
						{
							Category:     "equityFund",
							FundName:     "BTC_DeltaNeutral2",
							Coin:         currency.BTC,
							APRRangeLow:  0.01,
							APRRangeHigh: 0.11,
							Tags: []string{
								"30% High Water Mark",
								"5% Drawdown protection",
							},
							Introduction:        "The BTC Fund employs a composite arbitrage strategy combined with intelligent risk control systems.",
							AUM:                 507,
							MinInvestmentAmount: 10,
							MaxInvestmentAmount: 100,
							MaxDrawdown:         0.004,
							SharpRatio:          7,
							EstimatedAPR:        0.02,
						},
					},
				},
				{
					Type: "multiCoinEarning",
					Cards: []PWMProductCard{
						{
							Category:            "flexibleSavings",
							ProductID:           "3",
							Coin:                currency.BTC,
							APR:                 0.001,
							Tags:                []string{},
							MinInvestmentAmount: 1,
							MaxInvestmentAmount: 1500,
						},
					},
				},
				{
					Type: "fixedYield",
					Cards: []PWMProductCard{
						{
							Category:            "fundPoolPremium",
							ProductID:           "6",
							Coin:                currency.USDT,
							APR:                 0.026,
							Tags:                []string{},
							MinInvestmentAmount: 300000,
							MaxInvestmentAmount: 600000,
							Duration:            180,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetPWMSubscribableProductInfo should decode every field")
		return
	}
	require.NotEmpty(t, result.Products, "GetPWMSubscribableProductInfo must return product groups")
	assert.NotEmpty(t, result.Products[0].Cards, "GetPWMSubscribableProductInfo should return products")
}

func TestCreatePWMCustomisedInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePWMCustomisedInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePWMCustomisedInvestmentPlan must reject a nil request")
	req := &PWMCustomisedPlanRequest{}
	_, err = e.CreatePWMCustomisedInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errNoProductsProvided, "CreatePWMCustomisedInvestmentPlan must reject a plan without products")
	req.Products = []PWMCustomisedPlanProduct{{}}
	_, err = e.CreatePWMCustomisedInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "CreatePWMCustomisedInvestmentPlan must reject a product without a category")
	req.Products[0].Category = "equityFund"
	_, err = e.CreatePWMCustomisedInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreatePWMCustomisedInvestmentPlan must reject a product without an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreatePWMCustomisedInvestmentPlan(t.Context(), &PWMCustomisedPlanRequest{
		AccountType: "FUND",
		Products: []PWMCustomisedPlanProduct{
			{Category: "equityFund", FundName: "BTC_DeltaNeutral2", Amount: 100000},
			{Category: "flexibleSavings", ProductID: "3", Amount: 50000},
		},
		OrderLinkID: "pwm-custom-001",
	})
	require.NoError(t, err, "CreatePWMCustomisedInvestmentPlan must not error")
	if mockTests {
		exp := &PWMCustomisedPlanResponse{
			PlanID:      "10050",
			PlanName:    "PWM-10050",
			Status:      "Active",
			OrderLinkID: "pwm-custom-001",
		}
		assert.Equal(t, exp, result, "CreatePWMCustomisedInvestmentPlan should decode every field")
		return
	}
	assert.NotEmpty(t, result.PlanID, "CreatePWMCustomisedInvestmentPlan should return a plan ID")
}

func TestGetAllPWMFunds(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPWMFunds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllPWMFunds must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *PWMFundsRequest
		exp  *PWMFundsResponse
	}{
		{
			name: "list",
			req:  &PWMFundsRequest{Coin: currency.BTC, Status: "Active", Limit: 20},
			exp: &PWMFundsResponse{
				List: []PWMFund{
					{
						FundID:      "123",
						FundName:    "BTC Alpha Fund",
						Coin:        currency.BTC,
						Status:      "Active",
						TotalEquity: 10.5,
						TotalShares: 1000,
						CurrentNAV:  1.05,
						CurrentAPR:  0.12,
						AccountUID:  "456789",
						SubAccountList: []string{
							"456790",
							"456791",
						},
						ProfitShareRate:   20,
						ManagementFeeRate: 2,
						UncollectedProfit: 0.5,
						CollectedProfit:   1.2,
						TotalLoan:         0.3,
						CreatedTime:       types.Time(time.Unix(1640000000, 0)),
					},
				},
				NextPageCursor: "122",
			},
		},
		{
			name: "single fund",
			req:  &PWMFundsRequest{FundID: "124"},
			exp: &PWMFundsResponse{
				List: []PWMFund{
					{
						FundID:            "124",
						FundName:          "ETH Delta Fund",
						Coin:              currency.ETH,
						Status:            "Closing",
						TotalEquity:       120,
						TotalShares:       118.5,
						CurrentNAV:        1.0127,
						CurrentAPR:        0.08,
						AccountUID:        "456800",
						SubAccountList:    []string{},
						ProfitShareRate:   25,
						ManagementFeeRate: 1.5,
						CollectedProfit:   3.4,
						CreatedTime:       types.Time(time.Unix(1650000000, 0)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAllPWMFunds(t.Context(), tc.req)
			require.NoError(t, err, "GetAllPWMFunds must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAllPWMFunds should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetAllPWMFunds(t.Context(), tc.req)
			assert.NoError(t, err, "GetAllPWMFunds should not error for the next page")
		})
	}
}

func TestSettlePWMFundProfit(t *testing.T) {
	t.Parallel()
	_, err := e.SettlePWMFundProfit(t.Context(), "", "settle-001")
	require.ErrorIs(t, err, errFundIDRequired, "SettlePWMFundProfit must reject an empty fund ID")
	_, err = e.SettlePWMFundProfit(t.Context(), "12323", "")
	require.ErrorIs(t, err, errRequestLinkIDRequired, "SettlePWMFundProfit must reject an empty request link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SettlePWMFundProfit(t.Context(), "12323", "settle-001")
	require.NoError(t, err, "SettlePWMFundProfit must not error")
	if mockTests {
		exp := &PWMSettleFundProfitResponse{
			FundID:            "12323",
			Status:            "Processing",
			TotalProfitShared: 2.73,
			InstitutionIncome: 1.5,
			Coin:              currency.BTC,
			CreatedTime:       types.Time(time.Unix(1700000000, 0)),
		}
		assert.Equal(t, exp, result, "SettlePWMFundProfit should decode every field")
		return
	}
	assert.Equal(t, "12323", result.FundID, "SettlePWMFundProfit should return the fund ID")
}

func TestCreatePWMFund(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePWMFund(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePWMFund must reject a nil request")
	req := &PWMCreateFundRequest{}
	_, err = e.CreatePWMFund(t.Context(), req)
	require.ErrorIs(t, err, errFundNameRequired, "CreatePWMFund must reject an empty fund name")
	req.FundName = "BTC Alpha Fund"
	_, err = e.CreatePWMFund(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreatePWMFund must reject an empty coin")
	req.Coin = currency.BTC
	_, err = e.CreatePWMFund(t.Context(), req)
	require.ErrorIs(t, err, errRequestLinkIDRequired, "CreatePWMFund must reject an empty request link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.ProfitShareRate = 20
	req.ManagementFeeRate = 2
	req.FundIntroduction = "intro-btc-alpha"
	req.RequestLinkID = "create-fund-001"
	result, err := e.CreatePWMFund(t.Context(), req)
	require.NoError(t, err, "CreatePWMFund must not error")
	if mockTests {
		exp := &PWMCreateFundResponse{
			FundID:            "12345",
			FundName:          "BTC Alpha Fund",
			Coin:              currency.BTC,
			Status:            "PendingSubscribe",
			ProfitShareRate:   20,
			ManagementFeeRate: 2,
			AccountUID:        "800003",
			CreatedTime:       types.Time(time.Unix(1640000000, 0)),
		}
		assert.Equal(t, exp, result, "CreatePWMFund should decode every field")
		return
	}
	assert.NotEmpty(t, result.FundID, "CreatePWMFund should return a fund ID")
}

func TestCreatePWMInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePWMInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePWMInvestmentPlan must reject a nil request")
	req := &PWMCreateInvestmentPlanRequest{}
	_, err = e.CreatePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errAccountUIDRequired, "CreatePWMInvestmentPlan must reject an empty account UID")
	req.AccountUID = "800001"
	_, err = e.CreatePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanNameRequired, "CreatePWMInvestmentPlan must reject an empty plan name")
	req.PlanName = "Conservative Growth Plan"
	_, err = e.CreatePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanTypeRequired, "CreatePWMInvestmentPlan must reject an empty plan type")
	req.PlanType = "stable"
	_, err = e.CreatePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errNoFundsProvided, "CreatePWMInvestmentPlan must reject a plan without funds")
	req.InvestmentDistribution = []PWMPlanFundAllocation{{}}
	_, err = e.CreatePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errRequestLinkIDRequired, "CreatePWMInvestmentPlan must reject an empty request link ID")
	req.RequestLinkID = "create-plan-001"
	_, err = e.CreatePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "CreatePWMInvestmentPlan must reject a fund without a product ID")
	req.InvestmentDistribution[0].ProductID = "12345"
	_, err = e.CreatePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreatePWMInvestmentPlan must reject a fund without an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.InvestmentDistribution[0].Amount = 100000
	result, err := e.CreatePWMInvestmentPlan(t.Context(), req)
	require.NoError(t, err, "CreatePWMInvestmentPlan must not error")
	if mockTests {
		exp := &PWMCreateInvestmentPlanResponse{
			PlanID:      "10088",
			PlanType:    "stable",
			AccountUID:  "800001",
			Status:      "PendingSubscription",
			CreatedTime: types.Time(time.Unix(1640000000, 0)),
		}
		assert.Equal(t, exp, result, "CreatePWMInvestmentPlan should decode every field")
		return
	}
	assert.NotEmpty(t, result.PlanID, "CreatePWMInvestmentPlan should return a plan ID")
}

func TestGetPWMAssetManagerInvestmentPlans(t *testing.T) {
	t.Parallel()
	_, err := e.GetPWMAssetManagerInvestmentPlans(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPWMAssetManagerInvestmentPlans must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *PWMAssetManagerInvestmentPlansRequest
		exp  *PWMAssetManagerInvestmentPlansResponse
	}{
		{
			name: "list",
			req:  &PWMAssetManagerInvestmentPlansRequest{Status: "Active", SubscriptionUID: "12323434", Limit: 20},
			exp: &PWMAssetManagerInvestmentPlansResponse{
				List: []PWMAssetManagerInvestmentPlan{
					{
						PlanID:             "10001",
						PlanName:           "Conservative Growth Plan",
						PlanType:           "stable",
						SubscriptionUID:    "12323434",
						Status:             "Active",
						Source:             "institution",
						CurrentAssetUSD:    200137.5,
						AccumulateYieldUSD: 2137.5,
						InvestmentDistribution: []PWMInvestmentDistribution{
							{
								Category:      "equityFund",
								Coin:          currency.USDT,
								ProductID:     "431",
								CurrentAmount: 50000,
							},
						},
						CreatedTime: types.Time(time.Unix(1700000000, 0)),
					},
				},
				NextPageCursor: "10000",
			},
		},
		{
			name: "single plan",
			req:  &PWMAssetManagerInvestmentPlansRequest{PlanID: "10088"},
			exp: &PWMAssetManagerInvestmentPlansResponse{
				List: []PWMAssetManagerInvestmentPlan{
					{
						PlanID:          "10088",
						PlanName:        "Balanced Plan",
						PlanType:        "advanced",
						SubscriptionUID: "800001",
						Status:          "PendingSubscription",
						Source:          "institution",
						CurrentAssetUSD: 100000,
						InvestmentDistribution: []PWMInvestmentDistribution{
							{
								Category:      "equityFund",
								Coin:          currency.BTC,
								ProductID:     "12345",
								CurrentAmount: 1.5,
							},
						},
						CreatedTime: types.Time(time.Unix(1700500000, 0)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPWMAssetManagerInvestmentPlans(t.Context(), tc.req)
			require.NoError(t, err, "GetPWMAssetManagerInvestmentPlans must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPWMAssetManagerInvestmentPlans should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetPWMAssetManagerInvestmentPlans(t.Context(), tc.req)
			assert.NoError(t, err, "GetPWMAssetManagerInvestmentPlans should not error for the next page")
		})
	}
}

func TestManagePWMInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.ManagePWMInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ManagePWMInvestmentPlan must reject a nil request")
	req := &PWMManageInvestmentPlanRequest{}
	_, err = e.ManagePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanIDRequired, "ManagePWMInvestmentPlan must reject an empty plan ID")
	req.PlanID = "10088"
	_, err = e.ManagePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errRequestLinkIDRequired, "ManagePWMInvestmentPlan must reject an empty request link ID")
	req.RequestLinkID = "manage-plan-001"
	req.UpdateFunds = []PWMFundAllocation{{}}
	_, err = e.ManagePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errFundIDRequired, "ManagePWMInvestmentPlan must reject a fund without a fund ID")
	req.UpdateFunds[0].FundID = "430"
	_, err = e.ManagePWMInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "ManagePWMInvestmentPlan must reject a fund without an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.UpdateStatus = "Closed"
	req.UpdateFunds = []PWMFundAllocation{{FundID: "430", Amount: 100000}, {FundID: "2005", Amount: 270000}}
	result, err := e.ManagePWMInvestmentPlan(t.Context(), req)
	require.NoError(t, err, "ManagePWMInvestmentPlan must not error")
	if mockTests {
		exp := &PWMManageInvestmentPlanResponse{
			PlanID:      "10088",
			Status:      "Closed",
			UpdatedTime: types.Time(time.Unix(1701700000, 0)),
			UpdateFunds: []PWMUpdatedFund{
				{
					FundID: "430",
					Amount: 100000,
				},
				{
					FundID: "2005",
					Amount: 270000,
				},
			},
		}
		assert.Equal(t, exp, result, "ManagePWMInvestmentPlan should decode every field")
		return
	}
	assert.Equal(t, req.PlanID, result.PlanID, "ManagePWMInvestmentPlan should return the plan ID")
}

func TestGetAllPWMFundOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPWMFundOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllPWMFundOrders must reject a nil request")
	_, err = e.GetAllPWMFundOrders(t.Context(), &PWMFundOrdersRequest{StartTime: time.UnixMilli(1700200000000), EndTime: time.UnixMilli(1700000000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllPWMFundOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *PWMFundOrdersRequest
		exp  *PWMFundOrdersResponse
	}{
		{
			name: "list",
			req:  &PWMFundOrdersRequest{FundID: "100001", StartTime: time.UnixMilli(1700000000000), EndTime: time.UnixMilli(1700200000000), Limit: 20},
			exp: &PWMFundOrdersResponse{
				List: []PWMFundOrder{
					{
						OrderID:     "768",
						FundID:      "100001",
						FundName:    "Alpha BTC Strategy Fund",
						AccountUID:  "800001",
						OrderType:   "Subscribe",
						Coin:        currency.BTC,
						Amount:      10,
						Status:      "Success",
						CreatedTime: types.Time(time.Unix(1700000000, 0)),
					},
					{
						OrderID:     "769",
						FundID:      "100001",
						FundName:    "Alpha BTC Strategy Fund",
						AccountUID:  "800002",
						OrderType:   "Redeem",
						Coin:        currency.BTC,
						Shares:      5000,
						Status:      "Processing",
						CreatedTime: types.Time(time.Unix(1700100000, 0)),
					},
				},
				NextPageCursor: "767",
			},
		},
		{
			name: "pending review redemptions",
			req:  &PWMFundOrdersRequest{OrderType: "Redeem", Status: "Pending Review"},
			exp: &PWMFundOrdersResponse{
				List: []PWMFundOrder{
					{
						OrderID:     "770",
						FundID:      "100001",
						FundName:    "Alpha BTC Strategy Fund",
						AccountUID:  "800002",
						OrderType:   "Redeem",
						Coin:        currency.BTC,
						Shares:      1200,
						Status:      "PendingReview",
						CreatedTime: types.Time(time.Unix(1700150000, 0)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAllPWMFundOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetAllPWMFundOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAllPWMFundOrders should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetAllPWMFundOrders(t.Context(), tc.req)
			assert.NoError(t, err, "GetAllPWMFundOrders should not error for the next page")
		})
	}
}

func TestManagePWMFundOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ManagePWMFundOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ManagePWMFundOrder must reject a nil request")
	req := &PWMManageOrderRequest{}
	_, err = e.ManagePWMFundOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ManagePWMFundOrder must reject an empty order ID")
	req.OrderID = "500002"
	_, err = e.ManagePWMFundOrder(t.Context(), req)
	require.ErrorIs(t, err, errActionRequired, "ManagePWMFundOrder must reject an empty action")
	req.Action = "approve"
	_, err = e.ManagePWMFundOrder(t.Context(), req)
	require.ErrorIs(t, err, errRequestLinkIDRequired, "ManagePWMFundOrder must reject an empty request link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PWMManageOrderRequest
		exp  *PWMManageOrderResponse
	}{
		{
			name: "approve subscription",
			req:  &PWMManageOrderRequest{OrderID: "500002", Action: "approve", RequestLinkID: "manage-order-001"},
			exp: &PWMManageOrderResponse{
				OrderID:     "500002",
				FundID:      "100001",
				AccountUID:  "800001",
				OrderStatus: "Pass",
				OrderType:   "Subscribe",
				Action:      "approve",
				Coin:        currency.BTC,
				Amount:      10,
				UpdatedTime: types.Time(time.Unix(1700100000, 0)),
			},
		},
		{
			name: "reject redemption",
			req:  &PWMManageOrderRequest{OrderID: "500003", Action: "reject", RequestLinkID: "manage-order-002"},
			exp: &PWMManageOrderResponse{
				OrderID:     "500003",
				FundID:      "100001",
				AccountUID:  "800001",
				OrderStatus: "Rejected",
				OrderType:   "Redeem",
				Action:      "reject",
				Coin:        currency.BTC,
				Shares:      5000,
				UpdatedTime: types.Time(time.Unix(1700100060, 0)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ManagePWMFundOrder(t.Context(), tc.req)
			require.NoError(t, err, "ManagePWMFundOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ManagePWMFundOrder should decode every field")
				return
			}
			assert.Equal(t, tc.req.OrderID, result.OrderID, "ManagePWMFundOrder should return the order ID")
		})
	}
}

func TestCreatePWMFundSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePWMFundSubAccount(t.Context(), "", "create-sub-001")
	require.ErrorIs(t, err, errFundIDRequired, "CreatePWMFundSubAccount must reject an empty fund ID")
	_, err = e.CreatePWMFundSubAccount(t.Context(), "100001", "")
	require.ErrorIs(t, err, errRequestLinkIDRequired, "CreatePWMFundSubAccount must reject an empty request link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreatePWMFundSubAccount(t.Context(), "100001", "create-sub-001")
	require.NoError(t, err, "CreatePWMFundSubAccount must not error")
	if mockTests {
		exp := &PWMCreateFundSubAccountResponse{
			FundID:        "100001",
			SubAccountUID: "800003",
			CreatedTime:   types.Time(time.Unix(1700600000, 0)),
			Status:        "success",
		}
		assert.Equal(t, exp, result, "CreatePWMFundSubAccount should decode every field")
		return
	}
	assert.Equal(t, "100001", result.FundID, "CreatePWMFundSubAccount should return the fund ID")
}
