package bybit

import (
	"fmt"
	"io"
	"net/http"
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

func TestTransferPrivateWealthManagementFunds(t *testing.T) {
	t.Parallel()
	_, err := e.TransferPrivateWealthManagementFunds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "TransferPrivateWealthManagementFunds must reject a nil request")
	req := &PrivateWealthManagementFundTransferRequest{}
	_, err = e.TransferPrivateWealthManagementFunds(t.Context(), req)
	require.ErrorIs(t, err, errMissingTransferID, "TransferPrivateWealthManagementFunds must reject an empty transfer ID")
	req.TransferID = "4fdf-re-4343-frewr"
	_, err = e.TransferPrivateWealthManagementFunds(t.Context(), req)
	require.ErrorIs(t, err, errMissingUserID, "TransferPrivateWealthManagementFunds must reject an empty source user ID")
	req.FromUserID = 800001
	_, err = e.TransferPrivateWealthManagementFunds(t.Context(), req)
	require.ErrorIs(t, err, errMissingUserID, "TransferPrivateWealthManagementFunds must reject an empty destination user ID")
	req.ToUserID = 800002
	_, err = e.TransferPrivateWealthManagementFunds(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "TransferPrivateWealthManagementFunds must reject a zero amount")
	req.Amount = 1
	_, err = e.TransferPrivateWealthManagementFunds(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "TransferPrivateWealthManagementFunds must reject an empty coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.Coin = currency.BTC
	result, err := e.TransferPrivateWealthManagementFunds(t.Context(), req)
	require.NoError(t, err, "TransferPrivateWealthManagementFunds must not error")
	if mockTests {
		exp := &PrivateWealthManagementFundTransferResponse{
			TransferID: "4fdf-re-4343-frewr",
			Status:     "PROCESSING",
		}
		assert.Equal(t, exp, result, "TransferPrivateWealthManagementFunds should decode every field")
		return
	}
	assert.Equal(t, req.TransferID, result.TransferID, "TransferPrivateWealthManagementFunds should return the transfer ID")
}

func TestGetPrivateWealthManagementFundTransferRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetPrivateWealthManagementFundTransferRecords(t.Context(), "", 800001)
	require.ErrorIs(t, err, errMissingTransferID, "GetPrivateWealthManagementFundTransferRecords must reject an empty transfer ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPrivateWealthManagementFundTransferRecords(t.Context(), "4fdf-re-4343-frewr", 800001)
	require.NoError(t, err, "GetPrivateWealthManagementFundTransferRecords must not error")
	if mockTests {
		exp := &PrivateWealthManagementFundTransferRecordsResponse{
			TransferID: "4fdf-re-4343-frewr",
			Status:     "SUCCESS",
			FromUserID: 800001,
			ToUserID:   800002,
			Amount:     1,
			Coin:       currency.BTC,
		}
		assert.Equal(t, exp, result, "GetPrivateWealthManagementFundTransferRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPrivateWealthManagementFundTransferRecords should return a result")
}

func TestGetAllPrivateWealthManagementInvestmentPlans(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPrivateWealthManagementInvestmentPlans(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllPrivateWealthManagementInvestmentPlans must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	req := &PrivateWealthManagementInvestmentPlansRequest{PlanID: "10001", Status: "Active", Limit: 20}
	result, err := e.GetAllPrivateWealthManagementInvestmentPlans(t.Context(), req)
	require.NoError(t, err, "GetAllPrivateWealthManagementInvestmentPlans must not error")
	if mockTests {
		exp := &PrivateWealthManagementInvestmentPlansResponse{
			List: []PrivateWealthManagementInvestmentPlan{
				{
					PlanID:             "10001",
					PlanName:           "Conservative Growth Plan",
					PlanType:           "stable",
					Status:             "Active",
					Source:             "consultant",
					CurrentAssetUSD:    200137.5,
					AccumulateYieldUSD: 2137.5,
					InvestmentDistribution: []PrivateWealthManagementInvestmentDistribution{
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
		assert.Equal(t, exp, result, "GetAllPrivateWealthManagementInvestmentPlans should decode every field")
	}
	if result.NextPageCursor == "" {
		return
	}
	req.Cursor = result.NextPageCursor
	_, err = e.GetAllPrivateWealthManagementInvestmentPlans(t.Context(), req)
	assert.NoError(t, err, "GetAllPrivateWealthManagementInvestmentPlans should not error for the next page")
}

func TestGetPrivateWealthManagementInvestmentPlanDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetPrivateWealthManagementInvestmentPlanDetail(t.Context(), "")
	require.ErrorIs(t, err, errPlanIDRequired, "GetPrivateWealthManagementInvestmentPlanDetail must reject an empty plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPrivateWealthManagementInvestmentPlanDetail(t.Context(), "10001")
	require.NoError(t, err, "GetPrivateWealthManagementInvestmentPlanDetail must not error")
	if mockTests {
		exp := &PrivateWealthManagementInvestmentPlanDetailResponse{
			PlanID:             "10001",
			PlanName:           "Conservative Growth Plan",
			PlanType:           "stable",
			Status:             "Active",
			CurrentAssetUSD:    200137.5,
			AccumulateYieldUSD: 2137.5,
			WeightedAverageAPR: 0.086,
			CurrentAssets: []PrivateWealthManagementCoinAmount{
				{
					Coin:   currency.USDT,
					Amount: 150000,
				},
			},
			Positions: PrivateWealthManagementPlanPositions{
				MultiCoinsEarning: PrivateWealthManagementMultiCoinsEarningPositions{
					TotalInvestmentUSD: 60000,
					AccumulateYieldUSD: 500,
					WeightedAverageAPR: 0.052,
					Items: []PrivateWealthManagementMultiCoinsEarningPosition{
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
				FixedYield: PrivateWealthManagementFixedYieldPositions{
					TotalInvestmentUSD: 50000,
					AccumulateYieldUSD: 800,
					WeightedAverageAPR: 0.08,
					Items: []PrivateWealthManagementFixedYieldPosition{
						{
							Category:        "fundPool",
							ProductID:       "430",
							Coin:            currency.USDT,
							CurrentAmount:   50000,
							AccumulateYield: 800,
							APR:             0.08,
							DurationInDays:  30,
							MaturityTime:    types.Time(time.Unix(1700500000, 0)),
							AutoReinvest:    true,
							PositionID:      "122",
							Status:          "1",
						},
					},
				},
				EquityFunds: PrivateWealthManagementEquityFundPositions{
					TotalInvestmentUSD: 70000,
					AccumulateYieldUSD: 700,
					WeightedAverageAPR: 0.12,
					Items: []PrivateWealthManagementEquityFundPosition{
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
				OnchainEarn: PrivateWealthManagementOnchainEarnPositions{
					TotalInvestmentUSD: 18000,
					AccumulateYieldUSD: 137.5,
					Items: []PrivateWealthManagementOnchainEarnPosition{
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
				FundingAccount: []PrivateWealthManagementCoinAmount{
					{
						Coin:   currency.USDT,
						Amount: 165.5,
					},
				},
			},
			CreatedTime: types.Time(time.Unix(1700000000, 0)),
		}
		assert.Equal(t, exp, result, "GetPrivateWealthManagementInvestmentPlanDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPrivateWealthManagementInvestmentPlanDetail should return a result")
}

func TestGetPendingPrivateWealthManagementInvestmentPlanDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetPendingPrivateWealthManagementInvestmentPlanDetail(t.Context(), "")
	require.ErrorIs(t, err, errPlanIDRequired, "GetPendingPrivateWealthManagementInvestmentPlanDetail must reject an empty plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPendingPrivateWealthManagementInvestmentPlanDetail(t.Context(), "10003")
	require.NoError(t, err, "GetPendingPrivateWealthManagementInvestmentPlanDetail must not error")
	if mockTests {
		exp := &PrivateWealthManagementPendingInvestmentPlanDetailResponse{
			PlanID:                   "10003",
			PlanName:                 "Aggressive Growth Plan",
			PlanType:                 "advanced",
			Status:                   "PendingSubscription",
			Source:                   "consultant",
			TotalConfiguredAmountUSD: 500000,
			Products: PrivateWealthManagementPendingPlanProducts{
				MultiCoinsEarning: PrivateWealthManagementPendingProductGroup{
					ConfiguredAmountUSD: 100000,
					Items: []PrivateWealthManagementPendingProductConfig{
						{
							Category:         "flexibleSavings",
							ProductID:        "430",
							Coin:             currency.USDT,
							ConfiguredAmount: 100000,
							APR:              0.05,
						},
					},
				},
				FixedYield: PrivateWealthManagementPendingFixedYieldGroup{
					ConfiguredAmountUSD: 80000,
					Items: []PrivateWealthManagementPendingFixedYieldConfig{
						{
							Category:         "fundPoolPremium",
							ProductID:        "431",
							Coin:             currency.USDT,
							ConfiguredAmount: 80000,
							APR:              0.08,
							DurationInDays:   30,
						},
					},
				},
				EquityFunds: PrivateWealthManagementPendingEquityFundGroup{
					ConfiguredAmountUSD: 270000,
					Items: []PrivateWealthManagementPendingEquityFundConfig{
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
							Introduction: PrivateWealthManagementFundIntroduction{
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
				OnchainEarn: PrivateWealthManagementPendingProductGroup{
					ConfiguredAmountUSD: 50000,
					Items: []PrivateWealthManagementPendingProductConfig{
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
		assert.Equal(t, exp, result, "GetPendingPrivateWealthManagementInvestmentPlanDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPendingPrivateWealthManagementInvestmentPlanDetail should return a result")
}

func TestClaimPrivateWealthManagementWithdrawableFunds(t *testing.T) {
	t.Parallel()
	_, err := e.ClaimPrivateWealthManagementWithdrawableFunds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ClaimPrivateWealthManagementWithdrawableFunds must reject a nil request")
	req := &PrivateWealthManagementClaimRequest{}
	_, err = e.ClaimPrivateWealthManagementWithdrawableFunds(t.Context(), req)
	require.ErrorIs(t, err, errPlanIDRequired, "ClaimPrivateWealthManagementWithdrawableFunds must reject an empty plan ID")
	req.PlanID = "10001"
	_, err = e.ClaimPrivateWealthManagementWithdrawableFunds(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "ClaimPrivateWealthManagementWithdrawableFunds must reject an empty order link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.OrderLinkID = "claim-order-001"
	req.ToAccountType = "FUND"
	result, err := e.ClaimPrivateWealthManagementWithdrawableFunds(t.Context(), req)
	require.NoError(t, err, "ClaimPrivateWealthManagementWithdrawableFunds must not error")
	if mockTests {
		exp := &PrivateWealthManagementClaimResponse{
			PlanID:        "10001",
			ToAccountType: 6,
			Status:        "Success",
			CreatedTime:   types.Time(time.Unix(1701400000, 0)),
		}
		assert.Equal(t, exp, result, "ClaimPrivateWealthManagementWithdrawableFunds should decode every field")
		return
	}
	assert.Equal(t, req.PlanID, result.PlanID, "ClaimPrivateWealthManagementWithdrawableFunds should return the plan ID")
}

func TestGetPrivateWealthManagementAssetTrend(t *testing.T) {
	t.Parallel()
	_, err := e.GetPrivateWealthManagementAssetTrend(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPrivateWealthManagementAssetTrend must reject a nil request")
	_, err = e.GetPrivateWealthManagementAssetTrend(t.Context(), &PrivateWealthManagementAssetTrendRequest{})
	require.ErrorIs(t, err, errPlanIDRequired, "GetPrivateWealthManagementAssetTrend must reject an empty plan ID")
	_, err = e.GetPrivateWealthManagementAssetTrend(t.Context(), &PrivateWealthManagementAssetTrendRequest{PlanID: "10001", StartTime: time.UnixMilli(1730592000000), EndTime: time.UnixMilli(1730419200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPrivateWealthManagementAssetTrend must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPrivateWealthManagementAssetTrend(t.Context(), &PrivateWealthManagementAssetTrendRequest{PlanID: "10001", StartTime: time.UnixMilli(1730419200000), EndTime: time.UnixMilli(1730592000000)})
	require.NoError(t, err, "GetPrivateWealthManagementAssetTrend must not error")
	if mockTests {
		exp := &PrivateWealthManagementAssetTrendResponse{
			PlanID: "10001",
			DataPoints: []PrivateWealthManagementAssetTrendPoint{
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
		assert.Equal(t, exp, result, "GetPrivateWealthManagementAssetTrend should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPrivateWealthManagementAssetTrend should return a result")
}

func TestGetPrivateWealthManagementFundHistoricalNAV(t *testing.T) {
	t.Parallel()
	_, err := e.GetPrivateWealthManagementFundHistoricalNAV(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPrivateWealthManagementFundHistoricalNAV must reject a nil request")
	_, err = e.GetPrivateWealthManagementFundHistoricalNAV(t.Context(), &PrivateWealthManagementFundHistoricalNAVRequest{})
	require.ErrorIs(t, err, errFundIDRequired, "GetPrivateWealthManagementFundHistoricalNAV must reject an empty fund ID")
	_, err = e.GetPrivateWealthManagementFundHistoricalNAV(t.Context(), &PrivateWealthManagementFundHistoricalNAVRequest{FundID: "2001", StartTime: time.UnixMilli(1730592000000), EndTime: time.UnixMilli(1730419200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPrivateWealthManagementFundHistoricalNAV must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPrivateWealthManagementFundHistoricalNAV(t.Context(), &PrivateWealthManagementFundHistoricalNAVRequest{FundID: "2001", StartTime: time.UnixMilli(1730419200000), EndTime: time.UnixMilli(1730592000000)})
	require.NoError(t, err, "GetPrivateWealthManagementFundHistoricalNAV must not error")
	if mockTests {
		exp := &PrivateWealthManagementFundHistoricalNAVResponse{
			FundID:     "2001",
			FundName:   "Market Neutral Alpha",
			Coin:       currency.USDT,
			CurrentNAV: 1.035,
			DataPoints: []PrivateWealthManagementFundNAVPoint{
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
		assert.Equal(t, exp, result, "GetPrivateWealthManagementFundHistoricalNAV should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPrivateWealthManagementFundHistoricalNAV should return a result")
}

func TestSubscribePrivateWealthManagementInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribePrivateWealthManagementInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubscribePrivateWealthManagementInvestmentPlan must reject a nil request")
	req := &PrivateWealthManagementSubscribeRequest{}
	_, err = e.SubscribePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanIDRequired, "SubscribePrivateWealthManagementInvestmentPlan must reject an empty plan ID")
	req.PlanID = "10003"
	_, err = e.SubscribePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "SubscribePrivateWealthManagementInvestmentPlan must reject an empty order link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.OrderLinkID = "pwm-subscribe-001"
	req.AccountType = "FUND"
	result, err := e.SubscribePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.NoError(t, err, "SubscribePrivateWealthManagementInvestmentPlan must not error")
	if mockTests {
		exp := &PrivateWealthManagementSubscribeResponse{
			PlanID:      "10003",
			Status:      "Active",
			OrderLinkID: "pwm-subscribe-001",
		}
		assert.Equal(t, exp, result, "SubscribePrivateWealthManagementInvestmentPlan should decode every field")
		return
	}
	assert.Equal(t, req.PlanID, result.PlanID, "SubscribePrivateWealthManagementInvestmentPlan should return the plan ID")
}

func TestInvestMoreInPrivateWealthManagementInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.InvestMoreInPrivateWealthManagementInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "InvestMoreInPrivateWealthManagementInvestmentPlan must reject a nil request")
	req := &PrivateWealthManagementInvestMoreRequest{}
	_, err = e.InvestMoreInPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanIDRequired, "InvestMoreInPrivateWealthManagementInvestmentPlan must reject an empty plan ID")
	req.PlanID = "10001"
	_, err = e.InvestMoreInPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "InvestMoreInPrivateWealthManagementInvestmentPlan must reject an empty category")
	req.Category = "equityFund"
	_, err = e.InvestMoreInPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "InvestMoreInPrivateWealthManagementInvestmentPlan must reject an empty product ID")
	req.ProductID = "2001"
	_, err = e.InvestMoreInPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "InvestMoreInPrivateWealthManagementInvestmentPlan must reject a zero amount")
	req.Amount = 20000
	_, err = e.InvestMoreInPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "InvestMoreInPrivateWealthManagementInvestmentPlan must reject an empty order link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.OrderLinkID = "pwm-invest-more-001"
	req.AccountType = "FUND"
	result, err := e.InvestMoreInPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.NoError(t, err, "InvestMoreInPrivateWealthManagementInvestmentPlan must not error")
	if mockTests {
		exp := &PrivateWealthManagementInvestMoreResponse{
			PlanID:      "10001",
			Category:    "equityFund",
			ProductID:   "2001",
			Coin:        currency.USDT,
			Amount:      20000,
			Status:      "Pending",
			OrderLinkID: "pwm-invest-more-001",
			OrderID:     "ORD20241115002",
		}
		assert.Equal(t, exp, result, "InvestMoreInPrivateWealthManagementInvestmentPlan should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "InvestMoreInPrivateWealthManagementInvestmentPlan should return an order ID")
}

func TestRedeemPrivateWealthManagementInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemPrivateWealthManagementInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RedeemPrivateWealthManagementInvestmentPlan must reject a nil request")
	req := &PrivateWealthManagementRedeemRequest{}
	_, err = e.RedeemPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanIDRequired, "RedeemPrivateWealthManagementInvestmentPlan must reject an empty plan ID")
	req.PlanID = "10001"
	_, err = e.RedeemPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "RedeemPrivateWealthManagementInvestmentPlan must reject an empty category")
	req.Category = "equityFund"
	_, err = e.RedeemPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "RedeemPrivateWealthManagementInvestmentPlan must reject an empty product ID")
	req.ProductID = "2001"
	_, err = e.RedeemPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "RedeemPrivateWealthManagementInvestmentPlan must reject a request without shares or an amount")
	req.Shares = 3000
	_, err = e.RedeemPrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "RedeemPrivateWealthManagementInvestmentPlan must reject an empty order link ID")
	unsent := newTransportTestExchange(t, func(http.ResponseWriter, *http.Request) {
		assert.Fail(t, "RedeemPrivateWealthManagementInvestmentPlan should not send a redemption its product's rules refuse")
	})
	for _, tc := range []struct {
		name string
		req  *PrivateWealthManagementRedeemRequest
		err  error
	}{
		{name: "an equity fund without shares", req: &PrivateWealthManagementRedeemRequest{PlanID: "10001", Category: "equityFund", ProductID: "2001", Amount: 3000, OrderLinkID: "pwm-redeem-003"}, err: order.ErrAmountIsInvalid},
		{name: "a FundPool product without shares or an amount", req: &PrivateWealthManagementRedeemRequest{PlanID: "10001", Category: "fundPool", ProductID: "430", PositionID: "77", OrderLinkID: "pwm-redeem-004"}, err: order.ErrAmountIsInvalid},
		{name: "a FundPool product without a position ID", req: &PrivateWealthManagementRedeemRequest{PlanID: "10001", Category: "fundPool", ProductID: "430", Amount: 5000, OrderLinkID: "pwm-redeem-005"}, err: errPositionIDRequired},
		{name: "an On-chain Earn product without a position ID", req: &PrivateWealthManagementRedeemRequest{PlanID: "10001", Category: "onchainEarn", ProductID: "25", Amount: 1, OrderLinkID: "pwm-redeem-006"}, err: errPositionIDRequired},
	} {
		_, err = unsent.RedeemPrivateWealthManagementInvestmentPlan(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "RedeemPrivateWealthManagementInvestmentPlan must reject %s", tc.name)
	}
	// Which categories besides equity funds count as fund products is not documented, so shares alone are sent for them
	var sentBody string
	sent := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the request body should not error")
		sentBody = string(b)
		_, err = fmt.Fprint(w, `{"retCode":0,"retMsg":"OK","result":{"orderId":"1","planId":"10001","category":"fundPool","productId":"430","shares":"10","orderLinkId":"pwm-redeem-007"},"retExtInfo":{},"time":1760000000000}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	_, err = sent.RedeemPrivateWealthManagementInvestmentPlan(t.Context(), &PrivateWealthManagementRedeemRequest{PlanID: "10001", Category: "fundPool", ProductID: "430", Shares: 10, PositionID: "77", OrderLinkID: "pwm-redeem-007"})
	require.NoError(t, err, "RedeemPrivateWealthManagementInvestmentPlan must send a FundPool redemption of shares")
	assert.JSONEq(t, `{"planId":"10001","category":"fundPool","productId":"430","shares":"10","positionId":"77","orderLinkId":"pwm-redeem-007"}`, sentBody, "RedeemPrivateWealthManagementInvestmentPlan should send the shares and position given")
	_, err = sent.RedeemPrivateWealthManagementInvestmentPlan(t.Context(), &PrivateWealthManagementRedeemRequest{PlanID: "10001", Category: "equityFund", ProductID: "2001", Shares: 0.5, OrderLinkID: "pwm-redeem-008"})
	require.NoError(t, err, "RedeemPrivateWealthManagementInvestmentPlan must send an equity fund redemption of part of a share")
	assert.JSONEq(t, `{"planId":"10001","category":"equityFund","productId":"2001","shares":"0.5","orderLinkId":"pwm-redeem-008"}`, sentBody, "RedeemPrivateWealthManagementInvestmentPlan should send the fraction of a share given")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PrivateWealthManagementRedeemRequest
		exp  *PrivateWealthManagementRedeemResponse
	}{
		{
			name: "fund shares",
			req:  &PrivateWealthManagementRedeemRequest{PlanID: "10001", Category: "equityFund", ProductID: "2001", Shares: 3000, OrderLinkID: "pwm-redeem-001"},
			exp: &PrivateWealthManagementRedeemResponse{
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
			req:  &PrivateWealthManagementRedeemRequest{PlanID: "10001", Category: "fundPool", ProductID: "430", Amount: 5000, OrderLinkID: "pwm-redeem-002", PositionID: "122"},
			exp: &PrivateWealthManagementRedeemResponse{
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
			result, err := e.RedeemPrivateWealthManagementInvestmentPlan(t.Context(), tc.req)
			require.NoError(t, err, "RedeemPrivateWealthManagementInvestmentPlan must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "RedeemPrivateWealthManagementInvestmentPlan should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderID, "RedeemPrivateWealthManagementInvestmentPlan should return an order ID")
		})
	}
}

func TestGetPrivateWealthManagementInvestmentPlanOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetPrivateWealthManagementInvestmentPlanOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPrivateWealthManagementInvestmentPlanOrders must reject a nil request")
	_, err = e.GetPrivateWealthManagementInvestmentPlanOrders(t.Context(), &PrivateWealthManagementInvestmentPlanOrdersRequest{StartTime: time.UnixMilli(1739520000000), EndTime: time.UnixMilli(1739433600000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPrivateWealthManagementInvestmentPlanOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	req := &PrivateWealthManagementInvestmentPlanOrdersRequest{
		PlanID:      "10001",
		Category:    "flexibleSavings",
		Type:        "Subscribe",
		Status:      "Completed",
		StartTime:   time.UnixMilli(1739433600000),
		EndTime:     time.UnixMilli(1739520000000),
		Limit:       20,
		OrderLinkID: "pwm-subscribe-001",
	}
	result, err := e.GetPrivateWealthManagementInvestmentPlanOrders(t.Context(), req)
	require.NoError(t, err, "GetPrivateWealthManagementInvestmentPlanOrders must not error")
	if mockTests {
		exp := &PrivateWealthManagementInvestmentPlanOrdersResponse{
			List: []PrivateWealthManagementInvestmentPlanOrder{
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
		assert.Equal(t, exp, result, "GetPrivateWealthManagementInvestmentPlanOrders should decode every field")
	}
	if result.NextPageCursor == "" {
		return
	}
	req.Cursor = result.NextPageCursor
	_, err = e.GetPrivateWealthManagementInvestmentPlanOrders(t.Context(), req)
	assert.NoError(t, err, "GetPrivateWealthManagementInvestmentPlanOrders should not error for the next page")
}

func TestGetPrivateWealthManagementSubscribableProductInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetPrivateWealthManagementSubscribableProductInfo(t.Context())
	require.NoError(t, err, "GetPrivateWealthManagementSubscribableProductInfo must not error")
	if mockTests {
		exp := &PrivateWealthManagementSubscribableProductInfoResponse{
			Products: []PrivateWealthManagementProductGroup{
				{
					Type: "equityFund",
					Cards: []PrivateWealthManagementProductCard{
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
					Cards: []PrivateWealthManagementProductCard{
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
					Cards: []PrivateWealthManagementProductCard{
						{
							Category:            "fundPoolPremium",
							ProductID:           "6",
							Coin:                currency.USDT,
							APR:                 0.026,
							Tags:                []string{},
							MinInvestmentAmount: 300000,
							MaxInvestmentAmount: 600000,
							DurationInDays:      180,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetPrivateWealthManagementSubscribableProductInfo should decode every field")
		return
	}
	require.NotEmpty(t, result.Products, "GetPrivateWealthManagementSubscribableProductInfo must return product groups")
	assert.NotEmpty(t, result.Products[0].Cards, "GetPrivateWealthManagementSubscribableProductInfo should return products")
}

func TestCreatePrivateWealthManagementCustomisedInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePrivateWealthManagementCustomisedInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePrivateWealthManagementCustomisedInvestmentPlan must reject a nil request")
	req := &PrivateWealthManagementCustomisedPlanRequest{}
	_, err = e.CreatePrivateWealthManagementCustomisedInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errNoProductsProvided, "CreatePrivateWealthManagementCustomisedInvestmentPlan must reject a plan without products")
	req.Products = []PrivateWealthManagementCustomisedPlanProduct{{}}
	_, err = e.CreatePrivateWealthManagementCustomisedInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errCategoryNotSet, "CreatePrivateWealthManagementCustomisedInvestmentPlan must reject a product without a category")
	req.Products[0].Category = "equityFund"
	_, err = e.CreatePrivateWealthManagementCustomisedInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreatePrivateWealthManagementCustomisedInvestmentPlan must reject a product without an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreatePrivateWealthManagementCustomisedInvestmentPlan(t.Context(), &PrivateWealthManagementCustomisedPlanRequest{
		AccountType: "FUND",
		Products: []PrivateWealthManagementCustomisedPlanProduct{
			{Category: "equityFund", FundName: "BTC_DeltaNeutral2", Amount: 100000},
			{Category: "flexibleSavings", ProductID: "3", Amount: 50000},
		},
		OrderLinkID: "pwm-custom-001",
	})
	require.NoError(t, err, "CreatePrivateWealthManagementCustomisedInvestmentPlan must not error")
	if mockTests {
		exp := &PrivateWealthManagementCustomisedPlanResponse{
			PlanID:      "10050",
			PlanName:    "PWM-10050",
			Status:      "Active",
			OrderLinkID: "pwm-custom-001",
		}
		assert.Equal(t, exp, result, "CreatePrivateWealthManagementCustomisedInvestmentPlan should decode every field")
		return
	}
	assert.NotEmpty(t, result.PlanID, "CreatePrivateWealthManagementCustomisedInvestmentPlan should return a plan ID")
}

func TestGetAllPrivateWealthManagementFunds(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPrivateWealthManagementFunds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllPrivateWealthManagementFunds must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *PrivateWealthManagementFundsRequest
		exp  *PrivateWealthManagementFundsResponse
	}{
		{
			name: "list",
			req:  &PrivateWealthManagementFundsRequest{Coin: currency.BTC, Status: "Active", Limit: 20},
			exp: &PrivateWealthManagementFundsResponse{
				List: []PrivateWealthManagementFund{
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
			req:  &PrivateWealthManagementFundsRequest{FundID: "124"},
			exp: &PrivateWealthManagementFundsResponse{
				List: []PrivateWealthManagementFund{
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
			result, err := e.GetAllPrivateWealthManagementFunds(t.Context(), tc.req)
			require.NoError(t, err, "GetAllPrivateWealthManagementFunds must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAllPrivateWealthManagementFunds should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetAllPrivateWealthManagementFunds(t.Context(), tc.req)
			assert.NoError(t, err, "GetAllPrivateWealthManagementFunds should not error for the next page")
		})
	}
}

func TestSettlePrivateWealthManagementFundProfit(t *testing.T) {
	t.Parallel()
	_, err := e.SettlePrivateWealthManagementFundProfit(t.Context(), "", "settle-001")
	require.ErrorIs(t, err, errFundIDRequired, "SettlePrivateWealthManagementFundProfit must reject an empty fund ID")
	_, err = e.SettlePrivateWealthManagementFundProfit(t.Context(), "12323", "")
	require.ErrorIs(t, err, errRequestLinkIDRequired, "SettlePrivateWealthManagementFundProfit must reject an empty request link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SettlePrivateWealthManagementFundProfit(t.Context(), "12323", "settle-001")
	require.NoError(t, err, "SettlePrivateWealthManagementFundProfit must not error")
	if mockTests {
		exp := &PrivateWealthManagementSettleFundProfitResponse{
			FundID:            "12323",
			Status:            "Processing",
			TotalProfitShared: 2.73,
			InstitutionIncome: 1.5,
			Coin:              currency.BTC,
			CreatedTime:       types.Time(time.Unix(1700000000, 0)),
		}
		assert.Equal(t, exp, result, "SettlePrivateWealthManagementFundProfit should decode every field")
		return
	}
	assert.Equal(t, "12323", result.FundID, "SettlePrivateWealthManagementFundProfit should return the fund ID")
}

func TestCreatePrivateWealthManagementFund(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePrivateWealthManagementFund(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePrivateWealthManagementFund must reject a nil request")
	req := &PrivateWealthManagementCreateFundRequest{}
	_, err = e.CreatePrivateWealthManagementFund(t.Context(), req)
	require.ErrorIs(t, err, errFundNameRequired, "CreatePrivateWealthManagementFund must reject an empty fund name")
	req.FundName = "BTC Alpha Fund"
	_, err = e.CreatePrivateWealthManagementFund(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreatePrivateWealthManagementFund must reject an empty coin")
	req.Coin = currency.BTC
	_, err = e.CreatePrivateWealthManagementFund(t.Context(), req)
	require.ErrorIs(t, err, errRequestLinkIDRequired, "CreatePrivateWealthManagementFund must reject an empty request link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.ProfitShareRate = 20
	req.ManagementFeeRate = 2
	req.FundIntroduction = "intro-btc-alpha"
	req.RequestLinkID = "create-fund-001"
	result, err := e.CreatePrivateWealthManagementFund(t.Context(), req)
	require.NoError(t, err, "CreatePrivateWealthManagementFund must not error")
	if mockTests {
		exp := &PrivateWealthManagementCreateFundResponse{
			FundID:            "12345",
			FundName:          "BTC Alpha Fund",
			Coin:              currency.BTC,
			Status:            "PendingSubscribe",
			ProfitShareRate:   20,
			ManagementFeeRate: 2,
			AccountUID:        "800003",
			CreatedTime:       types.Time(time.Unix(1640000000, 0)),
		}
		assert.Equal(t, exp, result, "CreatePrivateWealthManagementFund should decode every field")
		return
	}
	assert.NotEmpty(t, result.FundID, "CreatePrivateWealthManagementFund should return a fund ID")
}

func TestCreatePrivateWealthManagementInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePrivateWealthManagementInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreatePrivateWealthManagementInvestmentPlan must reject a nil request")
	req := &PrivateWealthManagementCreateInvestmentPlanRequest{}
	_, err = e.CreatePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errAccountUIDRequired, "CreatePrivateWealthManagementInvestmentPlan must reject an empty account UID")
	req.AccountUID = "800001"
	_, err = e.CreatePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanNameRequired, "CreatePrivateWealthManagementInvestmentPlan must reject an empty plan name")
	req.PlanName = "Conservative Growth Plan"
	_, err = e.CreatePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanTypeRequired, "CreatePrivateWealthManagementInvestmentPlan must reject an empty plan type")
	req.PlanType = "stable"
	_, err = e.CreatePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errNoFundsProvided, "CreatePrivateWealthManagementInvestmentPlan must reject a plan without funds")
	req.InvestmentDistribution = []PrivateWealthManagementPlanFundAllocation{{}}
	_, err = e.CreatePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errRequestLinkIDRequired, "CreatePrivateWealthManagementInvestmentPlan must reject an empty request link ID")
	req.RequestLinkID = "create-plan-001"
	_, err = e.CreatePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errProductIDRequired, "CreatePrivateWealthManagementInvestmentPlan must reject a fund without a product ID")
	req.InvestmentDistribution[0].ProductID = "12345"
	_, err = e.CreatePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreatePrivateWealthManagementInvestmentPlan must reject a fund without an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.InvestmentDistribution[0].Amount = 100000
	result, err := e.CreatePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.NoError(t, err, "CreatePrivateWealthManagementInvestmentPlan must not error")
	if mockTests {
		exp := &PrivateWealthManagementCreateInvestmentPlanResponse{
			PlanID:      "10088",
			PlanType:    "stable",
			AccountUID:  "800001",
			Status:      "PendingSubscription",
			CreatedTime: types.Time(time.Unix(1640000000, 0)),
		}
		assert.Equal(t, exp, result, "CreatePrivateWealthManagementInvestmentPlan should decode every field")
		return
	}
	assert.NotEmpty(t, result.PlanID, "CreatePrivateWealthManagementInvestmentPlan should return a plan ID")
}

func TestGetPrivateWealthManagementAssetManagerInvestmentPlans(t *testing.T) {
	t.Parallel()
	_, err := e.GetPrivateWealthManagementAssetManagerInvestmentPlans(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPrivateWealthManagementAssetManagerInvestmentPlans must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *PrivateWealthManagementAssetManagerInvestmentPlansRequest
		exp  *PrivateWealthManagementAssetManagerInvestmentPlansResponse
	}{
		{
			name: "list",
			req:  &PrivateWealthManagementAssetManagerInvestmentPlansRequest{Status: "Active", SubscriptionUID: "12323434", Limit: 20},
			exp: &PrivateWealthManagementAssetManagerInvestmentPlansResponse{
				List: []PrivateWealthManagementAssetManagerInvestmentPlan{
					{
						PlanID:             "10001",
						PlanName:           "Conservative Growth Plan",
						PlanType:           "stable",
						SubscriptionUID:    "12323434",
						Status:             "Active",
						Source:             "institution",
						CurrentAssetUSD:    200137.5,
						AccumulateYieldUSD: 2137.5,
						InvestmentDistribution: []PrivateWealthManagementInvestmentDistribution{
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
			req:  &PrivateWealthManagementAssetManagerInvestmentPlansRequest{PlanID: "10088"},
			exp: &PrivateWealthManagementAssetManagerInvestmentPlansResponse{
				List: []PrivateWealthManagementAssetManagerInvestmentPlan{
					{
						PlanID:          "10088",
						PlanName:        "Balanced Plan",
						PlanType:        "advanced",
						SubscriptionUID: "800001",
						Status:          "PendingSubscription",
						Source:          "institution",
						CurrentAssetUSD: 100000,
						InvestmentDistribution: []PrivateWealthManagementInvestmentDistribution{
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
			result, err := e.GetPrivateWealthManagementAssetManagerInvestmentPlans(t.Context(), tc.req)
			require.NoError(t, err, "GetPrivateWealthManagementAssetManagerInvestmentPlans must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPrivateWealthManagementAssetManagerInvestmentPlans should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetPrivateWealthManagementAssetManagerInvestmentPlans(t.Context(), tc.req)
			assert.NoError(t, err, "GetPrivateWealthManagementAssetManagerInvestmentPlans should not error for the next page")
		})
	}
}

func TestManagePrivateWealthManagementInvestmentPlan(t *testing.T) {
	t.Parallel()
	_, err := e.ManagePrivateWealthManagementInvestmentPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ManagePrivateWealthManagementInvestmentPlan must reject a nil request")
	req := &PrivateWealthManagementManageInvestmentPlanRequest{}
	_, err = e.ManagePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errPlanIDRequired, "ManagePrivateWealthManagementInvestmentPlan must reject an empty plan ID")
	req.PlanID = "10088"
	_, err = e.ManagePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errRequestLinkIDRequired, "ManagePrivateWealthManagementInvestmentPlan must reject an empty request link ID")
	req.RequestLinkID = "manage-plan-001"
	req.UpdateFunds = []PrivateWealthManagementFundAllocation{{}}
	_, err = e.ManagePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, errFundIDRequired, "ManagePrivateWealthManagementInvestmentPlan must reject a fund without a fund ID")
	req.UpdateFunds[0].FundID = "430"
	_, err = e.ManagePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "ManagePrivateWealthManagementInvestmentPlan must reject a fund without an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.UpdateStatus = "Closed"
	req.UpdateFunds = []PrivateWealthManagementFundAllocation{{FundID: "430", Amount: 100000}, {FundID: "2005", Amount: 270000}}
	result, err := e.ManagePrivateWealthManagementInvestmentPlan(t.Context(), req)
	require.NoError(t, err, "ManagePrivateWealthManagementInvestmentPlan must not error")
	if mockTests {
		exp := &PrivateWealthManagementManageInvestmentPlanResponse{
			PlanID:      "10088",
			Status:      "Closed",
			UpdatedTime: types.Time(time.Unix(1701700000, 0)),
			UpdateFunds: []PrivateWealthManagementUpdatedFund{
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
		assert.Equal(t, exp, result, "ManagePrivateWealthManagementInvestmentPlan should decode every field")
		return
	}
	assert.Equal(t, req.PlanID, result.PlanID, "ManagePrivateWealthManagementInvestmentPlan should return the plan ID")
}

func TestGetAllPrivateWealthManagementFundOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPrivateWealthManagementFundOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllPrivateWealthManagementFundOrders must reject a nil request")
	_, err = e.GetAllPrivateWealthManagementFundOrders(t.Context(), &PrivateWealthManagementFundOrdersRequest{StartTime: time.UnixMilli(1700200000000), EndTime: time.UnixMilli(1700000000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllPrivateWealthManagementFundOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *PrivateWealthManagementFundOrdersRequest
		exp  *PrivateWealthManagementFundOrdersResponse
	}{
		{
			name: "list",
			req:  &PrivateWealthManagementFundOrdersRequest{FundID: "100001", StartTime: time.UnixMilli(1700000000000), EndTime: time.UnixMilli(1700200000000), Limit: 20},
			exp: &PrivateWealthManagementFundOrdersResponse{
				List: []PrivateWealthManagementFundOrder{
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
			req:  &PrivateWealthManagementFundOrdersRequest{OrderType: "Redeem", Status: "Pending Review"},
			exp: &PrivateWealthManagementFundOrdersResponse{
				List: []PrivateWealthManagementFundOrder{
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
			result, err := e.GetAllPrivateWealthManagementFundOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetAllPrivateWealthManagementFundOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAllPrivateWealthManagementFundOrders should decode every field")
			}
			if result.NextPageCursor == "" {
				return
			}
			tc.req.Cursor = result.NextPageCursor
			_, err = e.GetAllPrivateWealthManagementFundOrders(t.Context(), tc.req)
			assert.NoError(t, err, "GetAllPrivateWealthManagementFundOrders should not error for the next page")
		})
	}
}

func TestManagePrivateWealthManagementFundOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ManagePrivateWealthManagementFundOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ManagePrivateWealthManagementFundOrder must reject a nil request")
	req := &PrivateWealthManagementManageOrderRequest{}
	_, err = e.ManagePrivateWealthManagementFundOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ManagePrivateWealthManagementFundOrder must reject an empty order ID")
	req.OrderID = "500002"
	_, err = e.ManagePrivateWealthManagementFundOrder(t.Context(), req)
	require.ErrorIs(t, err, errActionRequired, "ManagePrivateWealthManagementFundOrder must reject an empty action")
	req.Action = "approve"
	_, err = e.ManagePrivateWealthManagementFundOrder(t.Context(), req)
	require.ErrorIs(t, err, errRequestLinkIDRequired, "ManagePrivateWealthManagementFundOrder must reject an empty request link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PrivateWealthManagementManageOrderRequest
		exp  *PrivateWealthManagementManageOrderResponse
	}{
		{
			name: "approve subscription",
			req:  &PrivateWealthManagementManageOrderRequest{OrderID: "500002", Action: "approve", RequestLinkID: "manage-order-001"},
			exp: &PrivateWealthManagementManageOrderResponse{
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
			req:  &PrivateWealthManagementManageOrderRequest{OrderID: "500003", Action: "reject", RequestLinkID: "manage-order-002"},
			exp: &PrivateWealthManagementManageOrderResponse{
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
			result, err := e.ManagePrivateWealthManagementFundOrder(t.Context(), tc.req)
			require.NoError(t, err, "ManagePrivateWealthManagementFundOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ManagePrivateWealthManagementFundOrder should decode every field")
				return
			}
			assert.Equal(t, tc.req.OrderID, result.OrderID, "ManagePrivateWealthManagementFundOrder should return the order ID")
		})
	}
}

func TestCreatePrivateWealthManagementFundSubAccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePrivateWealthManagementFundSubAccount(t.Context(), "", "create-sub-001")
	require.ErrorIs(t, err, errFundIDRequired, "CreatePrivateWealthManagementFundSubAccount must reject an empty fund ID")
	_, err = e.CreatePrivateWealthManagementFundSubAccount(t.Context(), "100001", "")
	require.ErrorIs(t, err, errRequestLinkIDRequired, "CreatePrivateWealthManagementFundSubAccount must reject an empty request link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreatePrivateWealthManagementFundSubAccount(t.Context(), "100001", "create-sub-001")
	require.NoError(t, err, "CreatePrivateWealthManagementFundSubAccount must not error")
	if mockTests {
		exp := &PrivateWealthManagementCreateFundSubAccountResponse{
			FundID:        "100001",
			SubAccountUID: "800003",
			CreatedTime:   types.Time(time.Unix(1700600000, 0)),
			Status:        "success",
		}
		assert.Equal(t, exp, result, "CreatePrivateWealthManagementFundSubAccount should decode every field")
		return
	}
	assert.Equal(t, "100001", result.FundID, "CreatePrivateWealthManagementFundSubAccount should return the fund ID")
}
