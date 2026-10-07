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

// testPortfolio is the Auto-Invest portfolio used by the plan and one-time transaction tests
var testPortfolio = []PortfolioDetail{{TargetAsset: currency.BTC, Percentage: 60}, {TargetAsset: currency.ETH, Percentage: 40}}

func TestGetTargetAssetList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTargetAssetList(t.Context(), currency.BTC, 10, 40)
	require.NoError(t, err, "GetTargetAssetList must not error")
	if mockTests {
		exp := &AutoInvestTargetAssetListResponse{
			TargetAssets: []currency.Code{
				currency.BTC,
			},
			AutoInvestAssetList: []AutoInvestTargetAsset{
				{
					TargetAsset: currency.BTC,
					ROIAndDimensionTypeList: []AutoInvestROIDimension{
						{
							SimulateROI:    5.004,
							DimensionValue: 3,
							DimensionUnit:  "year",
						},
						{
							SimulateROI:    0.14,
							DimensionValue: 7,
							DimensionUnit:  "day",
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetTargetAssetList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTargetAssetList should return a result")
}

func TestGetTargetAssetROIData(t *testing.T) {
	t.Parallel()
	_, err := e.GetTargetAssetROIData(t.Context(), currency.EMPTYCODE, "THREE_YEAR")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTargetAssetROIData must reject an empty target asset")
	_, err = e.GetTargetAssetROIData(t.Context(), currency.ETH, "")
	require.ErrorIs(t, err, errHistoricalROITypeRequired, "GetTargetAssetROIData must reject an empty ROI type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTargetAssetROIData(t.Context(), currency.ETH, "THREE_YEAR")
	require.NoError(t, err, "GetTargetAssetROIData must not error")
	if mockTests {
		exp := []AutoInvestTargetAssetROI{
			{
				Date:        types.Time(time.Unix(1648378800, 0)),
				SimulateROI: 1.75,
			},
			{
				Date:        types.Time(time.Unix(1648478800, 0)),
				SimulateROI: 2.9,
			},
		}
		assert.Equal(t, exp, result, "GetTargetAssetROIData should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTargetAssetROIData should return a result")
}

func TestGetAllSourceAssetAndTargetAsset(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllSourceAssetAndTargetAsset(t.Context())
	require.NoError(t, err, "GetAllSourceAssetAndTargetAsset must not error")
	if mockTests {
		exp := &AutoInvestAllAssetsResponse{
			TargetAssets: []currency.Code{
				currency.BTC,
				currency.BNB,
			},
			SourceAssets: []currency.Code{
				currency.USDT,
				currency.USDC,
			},
		}
		assert.Equal(t, exp, result, "GetAllSourceAssetAndTargetAsset should decode every field")
		return
	}
	assert.NotEmpty(t, result.SourceAssets, "GetAllSourceAssetAndTargetAsset should return source assets")
}

func TestGetSourceAssetList(t *testing.T) {
	t.Parallel()
	_, err := e.GetSourceAssetList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSourceAssetList must reject a nil request")
	_, err = e.GetSourceAssetList(t.Context(), &AutoInvestSourceAssetListRequest{TargetAsset: currency.BTC})
	require.ErrorIs(t, err, errUsageTypeRequired, "GetSourceAssetList must reject an empty usage type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSourceAssetList(t.Context(), &AutoInvestSourceAssetListRequest{
		TargetAsset:          currency.BTC,
		IndexID:              1,
		UsageType:            "RECURRING",
		FlexibleAllowedToUse: true,
		SourceType:           "MAIN_SITE",
	})
	require.NoError(t, err, "GetSourceAssetList must not error")
	if mockTests {
		exp := &AutoInvestSourceAssetListResponse{
			FeeRate: 0.002,
			TaxRate: 0.001,
			SourceAssets: []AutoInvestSourceAsset{
				{
					SourceAsset:        currency.USDT,
					AssetMinimumAmount: 0.1,
					AssetMaximumAmount: 1000000,
					Scale:              2,
					FlexibleAmount:     1111,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSourceAssetList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSourceAssetList should return a result")
}

func TestInvestmentPlanCreation(t *testing.T) {
	t.Parallel()
	_, err := e.InvestmentPlanCreation(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "InvestmentPlanCreation must reject a nil request")
	schedule := InvestmentPlanSchedule{
		SubscriptionAmount:    4,
		SubscriptionCycle:     "MONTHLY",
		SubscriptionStartDay:  1,
		SubscriptionStartTime: 8,
		SourceAsset:           currency.USDT,
		FlexibleAllowedToUse:  true,
		Details:               testPortfolio,
	}
	_, err = e.InvestmentPlanCreation(t.Context(), &InvestmentPlanCreationRequest{PlanType: "PORTFOLIO", InvestmentPlanSchedule: schedule})
	require.ErrorIs(t, err, errSourceTypeRequired, "InvestmentPlanCreation must reject an empty source type")
	_, err = e.InvestmentPlanCreation(t.Context(), &InvestmentPlanCreationRequest{SourceType: "MAIN_SITE", InvestmentPlanSchedule: schedule})
	require.ErrorIs(t, err, errPlanTypeRequired, "InvestmentPlanCreation must reject an empty plan type")
	invalid := schedule
	invalid.SubscriptionAmount = 0
	_, err = e.InvestmentPlanCreation(t.Context(), &InvestmentPlanCreationRequest{SourceType: "MAIN_SITE", PlanType: "PORTFOLIO", InvestmentPlanSchedule: invalid})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "InvestmentPlanCreation must reject a zero subscription amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.InvestmentPlanCreation(t.Context(), &InvestmentPlanCreationRequest{
		SourceType:             "MAIN_SITE",
		RequestID:              "MAIN_SITE12345",
		PlanType:               "PORTFOLIO",
		InvestmentPlanSchedule: schedule,
	})
	require.NoError(t, err, "InvestmentPlanCreation must not error")
	if mockTests {
		exp := &AutoInvestPlanResponse{
			PlanID:                12345,
			NextExecutionDateTime: types.Time(time.Unix(1746057600, 0)),
		}
		assert.Equal(t, exp, result, "InvestmentPlanCreation should decode every field")
		return
	}
	assert.NotZero(t, result.PlanID, "InvestmentPlanCreation should return a plan ID")
}

func TestInvestmentPlanAdjustment(t *testing.T) {
	t.Parallel()
	_, err := e.InvestmentPlanAdjustment(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "InvestmentPlanAdjustment must reject a nil request")
	schedule := InvestmentPlanSchedule{
		SubscriptionAmount:       4,
		SubscriptionCycle:        "WEEKLY",
		SubscriptionStartWeekday: "MON",
		SourceAsset:              currency.USDT,
		Details:                  testPortfolio,
	}
	_, err = e.InvestmentPlanAdjustment(t.Context(), &InvestmentPlanAdjustmentRequest{InvestmentPlanSchedule: schedule})
	require.ErrorIs(t, err, errPlanIDRequired, "InvestmentPlanAdjustment must reject an empty plan ID")
	for _, tc := range []struct {
		name   string
		modify func(*InvestmentPlanSchedule)
		err    error
	}{
		{"subscription amount", func(s *InvestmentPlanSchedule) { s.SubscriptionAmount = 0 }, limits.ErrAmountBelowMin},
		{"subscription cycle", func(s *InvestmentPlanSchedule) { s.SubscriptionCycle = "" }, errInvalidSubscriptionCycle},
		{"source asset", func(s *InvestmentPlanSchedule) { s.SourceAsset = currency.EMPTYCODE }, currency.ErrCurrencyCodeEmpty},
		{"details", func(s *InvestmentPlanSchedule) { s.Details = nil }, errPortfolioDetailRequired},
		{"detail target asset", func(s *InvestmentPlanSchedule) { s.Details = []PortfolioDetail{{Percentage: 100}} }, currency.ErrCurrencyCodeEmpty},
		{"detail percentage", func(s *InvestmentPlanSchedule) { s.Details = []PortfolioDetail{{TargetAsset: currency.BTC}} }, errInvalidPercentageAmount},
	} {
		invalid := schedule
		tc.modify(&invalid)
		_, err = e.InvestmentPlanAdjustment(t.Context(), &InvestmentPlanAdjustmentRequest{PlanID: 1234232, InvestmentPlanSchedule: invalid})
		require.ErrorIsf(t, err, tc.err, "InvestmentPlanAdjustment must reject an invalid %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.InvestmentPlanAdjustment(t.Context(), &InvestmentPlanAdjustmentRequest{PlanID: 1234232, InvestmentPlanSchedule: schedule})
	require.NoError(t, err, "InvestmentPlanAdjustment must not error")
	if mockTests {
		exp := &AutoInvestPlanResponse{
			PlanID:                1234232,
			NextExecutionDateTime: types.Time(time.Unix(1744588800, 0)),
		}
		assert.Equal(t, exp, result, "InvestmentPlanAdjustment should decode every field")
		return
	}
	assert.NotZero(t, result.PlanID, "InvestmentPlanAdjustment should return the plan ID")
}

func TestChangePlanStatus(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePlanStatus(t.Context(), 0, "PAUSED")
	require.ErrorIs(t, err, errPlanIDRequired, "ChangePlanStatus must reject an empty plan ID")
	_, err = e.ChangePlanStatus(t.Context(), 12345, "")
	require.ErrorIs(t, err, errPlanStatusRequired, "ChangePlanStatus must reject an empty status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ChangePlanStatus(t.Context(), 12345, "PAUSED")
	require.NoError(t, err, "ChangePlanStatus must not error")
	if mockTests {
		exp := &AutoInvestPlanStatusResponse{
			PlanID:                12345,
			NextExecutionDateTime: types.Time(time.Unix(1744588800, 0)),
			Status:                "PAUSED",
		}
		assert.Equal(t, exp, result, "ChangePlanStatus should decode every field")
		return
	}
	assert.Equal(t, "PAUSED", result.Status, "ChangePlanStatus should return the new status")
}

func TestGetListOfPlans(t *testing.T) {
	t.Parallel()
	_, err := e.GetListOfPlans(t.Context(), "")
	require.ErrorIs(t, err, errPlanTypeRequired, "GetListOfPlans must reject an empty plan type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetListOfPlans(t.Context(), "SINGLE")
	require.NoError(t, err, "GetListOfPlans must not error")
	if mockTests {
		exp := &AutoInvestPlanListResponse{
			PlanValueInUSD: 123,
			PlanValueInBTC: 0.0016,
			PNLInUSD:       12.3,
			ROI:            0.111,
			Plans: []AutoInvestPlan{
				{
					PlanID:                 12345,
					PlanType:               "SINGLE",
					EditAllowed:            true,
					CreationDateTime:       types.Time(time.Unix(1744156800, 0)),
					FirstExecutionDateTime: types.Time(time.Unix(1744156800, 0)),
					NextExecutionDateTime:  types.Time(time.Unix(1744761600, 0)),
					Status:                 "ONGOING",
					LastUpdatedDateTime:    types.Time(time.Unix(1744156800, 0)),
					TargetAsset:            currency.BTC,
					TotalTargetAmount:      0.00111,
					SourceAsset:            currency.USDT,
					TotalInvestedInUSD:     110.7,
					SubscriptionAmount:     10,
					SubscriptionCycle:      "MONTHLY",
					SubscriptionStartDay:   1,
					SubscriptionStartTime:  1,
					SourceWallet:           "SPOT_WALLET",
					PlanValueInUSD:         123,
					PNLInUSD:               12.3,
					ROI:                    0.111,
				},
				{
					PlanID:                   12346,
					PlanType:                 "SINGLE",
					CreationDateTime:         types.Time(time.Unix(1744156800, 0)),
					FirstExecutionDateTime:   types.Time(time.Unix(1744156800, 0)),
					NextExecutionDateTime:    types.Time(time.Unix(1744761600, 0)),
					Status:                   "PAUSED",
					LastUpdatedDateTime:      types.Time(time.Unix(1744156800, 0)),
					TargetAsset:              currency.ETH,
					TotalTargetAmount:        0.0123,
					SourceAsset:              currency.USDT,
					TotalInvestedInUSD:       20,
					SubscriptionAmount:       5,
					SubscriptionCycle:        "WEEKLY",
					SubscriptionStartWeekday: "MON",
					SubscriptionStartTime:    2,
					SourceWallet:             "SPOT_WALLET_FLEXIBLE_SAVINGS",
					FlexibleAllowedToUse:     true,
					PlanValueInUSD:           19.5,
					PNLInUSD:                 -0.5,
					ROI:                      -0.025,
				},
			},
		}
		assert.Equal(t, exp, result, "GetListOfPlans should decode every field")
		return
	}
	assert.NotNil(t, result, "GetListOfPlans should return a result")
}

func TestGetHoldingDetailsOfPlan(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetHoldingDetailsOfPlan(t.Context(), 1234, "")
	require.NoError(t, err, "GetHoldingDetailsOfPlan must not error")
	if mockTests {
		exp := &AutoInvestPlanHoldingResponse{
			PlanID:                 1234,
			PlanType:               "INDEX",
			EditAllowed:            true,
			FlexibleAllowedToUse:   true,
			CreationDateTime:       types.Time(time.Unix(1744156800, 0)),
			FirstExecutionDateTime: types.Time(time.Unix(1744156800, 0)),
			NextExecutionDateTime:  types.Time(time.Unix(1744761600, 0)),
			Status:                 "ONGOING",
			TargetAsset:            currency.BTC,
			SourceAsset:            currency.USDT,
			PlanValueInUSD:         101.2,
			PNLInUSD:               1.2,
			ROI:                    0.012,
			TotalInvestedInUSD:     100,
			Details: []AutoInvestPlanHoldingDetail{
				{
					TargetAsset:         currency.ADA,
					AveragePriceInUSD:   0.64,
					TotalInvestedInUSD:  100,
					PurchasedAmount:     156.25,
					PurchasedAmountUnit: currency.ADA,
					PNLInUSD:            1.2,
					ROI:                 0.012,
					Percentage:          50,
					AssetStatus:         "ACTIVE",
					AvailableAmount:     150.25,
					AvailableAmountUnit: currency.ADA,
					RedeemedAmout:       6,
					RedeemedAmoutUnit:   currency.ADA,
					AssetValueInUSD:     101.2,
				},
			},
		}
		assert.Equal(t, exp, result, "GetHoldingDetailsOfPlan should decode every field")
		return
	}
	assert.NotNil(t, result, "GetHoldingDetailsOfPlan should return a result")
}

func TestGetSubscriptionsTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubscriptionsTransactionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubscriptionsTransactionHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetSubscriptionsTransactionHistory(t.Context(), &AutoInvestSubscriptionHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSubscriptionsTransactionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSubscriptionsTransactionHistory(t.Context(), &AutoInvestSubscriptionHistoryRequest{
		PlanID:      1232,
		StartTime:   startTime,
		EndTime:     endTime,
		TargetAsset: currency.BTC,
		PlanType:    "PORTFOLIO",
		Size:        20,
	})
	require.NoError(t, err, "GetSubscriptionsTransactionHistory must not error")
	if mockTests {
		exp := []AutoInvestSubscriptionTransaction{
			{
				ID:                  1111,
				TargetAsset:         currency.BTC,
				PlanType:            "PORTFOLIO",
				PlanName:            "BTC and ETH",
				PlanID:              1232,
				TransactionDateTime: types.Time(time.Unix(1744156800, 0)),
				TransactionStatus:   "FAILED",
				FailedType:          "INSUFFICIENT_BALANCE",
				SourceAsset:         currency.USDT,
				SourceAssetAmount:   297.12345,
				TargetAssetAmount:   0.005,
				SourceWallet:        "SPOT_WALLET_FLEXIBLE_SAVINGS",
				FlexibleUsed:        true,
				TransactionFee:      0.002,
				TransactionFeeUnit:  currency.USDT,
				ExecutionPrice:      59424.69,
				ExecutionType:       "RECURRING",
				SubscriptionCycle:   "WEEKLY",
			},
		}
		assert.Equal(t, exp, result, "GetSubscriptionsTransactionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSubscriptionsTransactionHistory should return a result")
}

func TestGetIndexDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexDetail(t.Context(), 0)
	require.ErrorIs(t, err, errIndexIDIsRequired, "GetIndexDetail must reject an empty index ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIndexDetail(t.Context(), 1)
	require.NoError(t, err, "GetIndexDetail must not error")
	if mockTests {
		exp := &AutoInvestIndexDetailResponse{
			IndexID:   1,
			IndexName: "BINANCE TOP 10 EW",
			Status:    "RUNNING",
			AssetAllocation: []AutoInvestAssetAllocation{
				{
					TargetAsset: currency.ADA,
					Allocation:  10,
				},
				{
					TargetAsset: currency.BTC,
					Allocation:  10,
				},
			},
		}
		assert.Equal(t, exp, result, "GetIndexDetail should decode every field")
		return
	}
	assert.NotEmpty(t, result.AssetAllocation, "GetIndexDetail should return the asset allocation")
}

func TestGetIndexLinkedPlanPositionDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexLinkedPlanPositionDetails(t.Context(), 0)
	require.ErrorIs(t, err, errIndexIDIsRequired, "GetIndexLinkedPlanPositionDetails must reject an empty index ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIndexLinkedPlanPositionDetails(t.Context(), 1)
	require.NoError(t, err, "GetIndexLinkedPlanPositionDetails must not error")
	if mockTests {
		exp := &IndexLinkedPlanPositionResponse{
			IndexID:              1,
			TotalInvestedInUSD:   114.555,
			CurrentInvestedInUSD: 101.2,
			PNLInUSD:             -13.355,
			ROI:                  -0.1166,
			AssetAllocation: []AutoInvestAssetAllocation{
				{
					TargetAsset: currency.ADA,
					Allocation:  10,
				},
			},
			Details: []IndexLinkedPlanAsset{
				{
					TargetAsset:          currency.ADA,
					AveragePriceInUSD:    0.64,
					TotalInvestedInUSD:   11.4555,
					CurrentInvestedInUSD: 10.12,
					PurchasedAmount:      17.89,
					PNLInUSD:             -1.3355,
					ROI:                  -0.1166,
					Percentage:           10,
					AvailableAmount:      15.89,
					RedeemedAmount:       2,
					AssetValueInUSD:      10.12,
				},
			},
		}
		assert.Equal(t, exp, result, "GetIndexLinkedPlanPositionDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIndexLinkedPlanPositionDetails should return a result")
}

func TestOneTimeTransaction(t *testing.T) {
	t.Parallel()
	_, err := e.OneTimeTransaction(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "OneTimeTransaction must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *OneTimeTransactionRequest
		err  error
	}{
		{"source type", &OneTimeTransactionRequest{SubscriptionAmount: 12, SourceAsset: currency.USDT, Details: testPortfolio}, errSourceTypeRequired},
		{"subscription amount", &OneTimeTransactionRequest{SourceType: "MAIN_SITE", SourceAsset: currency.USDT, Details: testPortfolio}, limits.ErrAmountBelowMin},
		{"source asset", &OneTimeTransactionRequest{SourceType: "MAIN_SITE", SubscriptionAmount: 12, Details: testPortfolio}, currency.ErrCurrencyCodeEmpty},
		{"plan, index or details", &OneTimeTransactionRequest{SourceType: "MAIN_SITE", SubscriptionAmount: 12, SourceAsset: currency.USDT}, errPortfolioDetailRequired},
		{"detail percentage", &OneTimeTransactionRequest{SourceType: "MAIN_SITE", SubscriptionAmount: 12, SourceAsset: currency.USDT, Details: []PortfolioDetail{{TargetAsset: currency.BTC}}}, errInvalidPercentageAmount},
	} {
		_, err = e.OneTimeTransaction(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "OneTimeTransaction must reject an invalid %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.OneTimeTransaction(t.Context(), &OneTimeTransactionRequest{
		SourceType:         "MAIN_SITE",
		SubscriptionAmount: 12,
		SourceAsset:        currency.USDT,
		Details:            testPortfolio,
	})
	require.NoError(t, err, "OneTimeTransaction must not error")
	if mockTests {
		exp := &AutoInvestOneTimeTransactionResponse{
			TransactionID: 12345,
			WaitSecond:    5,
		}
		assert.Equal(t, exp, result, "OneTimeTransaction should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "OneTimeTransaction should return a transaction ID")
}

func TestGetOneTimeTransactionStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetOneTimeTransactionStatus(t.Context(), 0, "")
	require.ErrorIs(t, err, errTransactionIDRequired, "GetOneTimeTransactionStatus must reject neither a transaction nor request ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOneTimeTransactionStatus(t.Context(), 1234, "")
	require.NoError(t, err, "GetOneTimeTransactionStatus must not error")
	if mockTests {
		exp := &AutoInvestOneTimeTransactionStatusResponse{
			TransactionID: 1234,
			Status:        "SUCCESS",
		}
		assert.Equal(t, exp, result, "GetOneTimeTransactionStatus should decode every field")
		return
	}
	assert.NotEmpty(t, result.Status, "GetOneTimeTransactionStatus should return a status")
}

func TestIndexLinkedPlanRedemption(t *testing.T) {
	t.Parallel()
	_, err := e.IndexLinkedPlanRedemption(t.Context(), 0, 30, "")
	require.ErrorIs(t, err, errIndexIDIsRequired, "IndexLinkedPlanRedemption must reject an empty index ID")
	_, err = e.IndexLinkedPlanRedemption(t.Context(), 1, 0, "")
	require.ErrorIs(t, err, errInvalidPercentageAmount, "IndexLinkedPlanRedemption must reject a zero percentage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.IndexLinkedPlanRedemption(t.Context(), 1, 30, "")
	require.NoError(t, err, "IndexLinkedPlanRedemption must not error")
	if mockTests {
		exp := &AutoInvestRedemptionResponse{
			RedemptionID: 19,
		}
		assert.Equal(t, exp, result, "IndexLinkedPlanRedemption should decode every field")
		return
	}
	assert.NotZero(t, result.RedemptionID, "IndexLinkedPlanRedemption should return a redemption ID")
}

func TestGetIndexLinkedPlanRedemption(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexLinkedPlanRedemption(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIndexLinkedPlanRedemption must reject a nil request")
	_, err = e.GetIndexLinkedPlanRedemption(t.Context(), &IndexLinkedPlanRedemptionHistoryRequest{Asset: currency.ETH})
	require.ErrorIs(t, err, errRequestIDRequired, "GetIndexLinkedPlanRedemption must reject an empty request ID")
	startTime, endTime := getTime()
	_, err = e.GetIndexLinkedPlanRedemption(t.Context(), &IndexLinkedPlanRedemptionHistoryRequest{RequestID: "123123", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetIndexLinkedPlanRedemption must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIndexLinkedPlanRedemption(t.Context(), &IndexLinkedPlanRedemptionHistoryRequest{
		RequestID: "123123",
		StartTime: startTime,
		EndTime:   endTime,
		Asset:     currency.ETH,
		Size:      10,
	})
	require.NoError(t, err, "GetIndexLinkedPlanRedemption must not error")
	if mockTests {
		exp := []IndexLinkedPlanRedemption{
			{
				IndexID:            1,
				IndexName:          "BINANCE TOP 10 EW",
				RedemptionID:       11,
				Status:             "SUCCESS",
				Asset:              currency.ETH,
				Amount:             0.005,
				RedemptionDateTime: types.Time(time.Unix(1744156800, 0)),
				TransactionFee:     0.00001,
				TransactionFeeUnit: currency.ETH,
			},
		}
		assert.Equal(t, exp, result, "GetIndexLinkedPlanRedemption should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIndexLinkedPlanRedemption should return a result")
}

func TestGetIndexLinkedPlanRebalanceDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexLinkedPlanRebalanceDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIndexLinkedPlanRebalanceDetails must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetIndexLinkedPlanRebalanceDetails(t.Context(), &EarnHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetIndexLinkedPlanRebalanceDetails must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIndexLinkedPlanRebalanceDetails(t.Context(), &EarnHistoryRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 100})
	require.NoError(t, err, "GetIndexLinkedPlanRebalanceDetails must not error")
	if mockTests {
		exp := []IndexLinkedPlanRebalance{
			{
				IndexID:          1,
				IndexName:        "BINANCE TOP 10 EW",
				RebalanceID:      11,
				Status:           "SUCCESS",
				RebalanceFee:     10,
				RebalanceFeeUnit: currency.USDT,
				TransactionDetails: []IndexLinkedPlanRebalanceTransaction{
					{
						Asset:               currency.BTC,
						TransactionDateTime: types.Time(time.Unix(1744156800, 0)),
						RebalanceDirection:  "BUY",
						RebalanceAmount:     0.005,
					},
					{
						Asset:               currency.ETH,
						TransactionDateTime: types.Time(time.Unix(1744156800, 0)),
						RebalanceDirection:  "SELL",
						RebalanceAmount:     0.05,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetIndexLinkedPlanRebalanceDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIndexLinkedPlanRebalanceDetails should return a result")
}
