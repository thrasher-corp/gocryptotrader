package gate

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetDualInvestmentProductList(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualInvestmentProductList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDualInvestmentProductList must reject a nil request")

	putPlan := &DualInvestmentPlanResponse{
		ID:               123123412341,
		InstrumentName:   "BTC 81,000",
		Type:             "put",
		InvestCurrency:   currency.USDT,
		ExerciseCurrency: currency.BTC,
		ExercisePrice:    81000,
		DeliveryTime:     types.Time(time.Unix(1744272000, 0)),
		APYDisplay:       0.9401,
		MinAmount:        2,
		EndTime:          types.Time(time.Unix(1744268400, 0)),
		Status:           "ONGOING",
	}
	callPlan := *putPlan
	callPlan.InstrumentName = "BTC 81,500"
	callPlan.Type = "call"
	callPlan.InvestCurrency = currency.BTC
	callPlan.ExerciseCurrency = currency.USDT
	callPlan.ExercisePrice = 81500.5
	callPlan.APYDisplay = 8.3846
	callPlan.MinAmount = 0.00002435
	for _, tc := range []struct {
		request *DualInvestmentProductListRequest
		exp     *DualInvestmentPlanResponse
	}{
		{&DualInvestmentProductListRequest{Sort: "apy", Type: "put", QuoteCurrency: currency.USDT, PlanID: 123123412341, Page: 1, PageSize: 1000}, putPlan},
		{&DualInvestmentProductListRequest{Sort: "short-period", Type: "put", Coin: currency.BTC, QuoteCurrency: currency.USDT, PlanID: 123123412341, Page: 1, PageSize: 1000}, putPlan},
		{&DualInvestmentProductListRequest{Sort: "apy", Type: "call", QuoteCurrency: currency.USDT, PlanID: 123123412341, Page: 1, PageSize: 1000}, &callPlan},
	} {
		result, err := e.GetDualInvestmentProductList(t.Context(), tc.request)
		require.NoErrorf(t, err, "GetDualInvestmentProductList must not error for sort %s type %s", tc.request.Sort, tc.request.Type)
		if mockTests {
			assert.Equalf(t, []*DualInvestmentPlanResponse{tc.exp}, result, "GetDualInvestmentProductList should decode every field for sort %s type %s", tc.request.Sort, tc.request.Type)
			continue
		}
		assert.NotNilf(t, result, "GetDualInvestmentProductList should return a result for sort %s type %s", tc.request.Sort, tc.request.Type)
	}
}

func TestGetDualInvestmentOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualInvestmentOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDualInvestmentOrderList must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.GetDualInvestmentOrderList(t.Context(), &DualInvestmentOrderListRequest{From: endTime, To: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDualInvestmentOrderList must reject a reversed time range")

	t.Run("future settlement window", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
		ex.SkipAuthCheck = true
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
		var query url.Values
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v4/earn/dual/orders", r.URL.Path, "request path should be the dual investment orders endpoint")
			query = r.URL.Query()
			_, err := fmt.Fprint(w, `[]`)
			assert.NoError(t, err, "writing dual investment orders should not error")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
		from, to := time.Now().Add(time.Hour*24), time.Now().Add(time.Hour*24*30)
		for _, tc := range []struct {
			name     string
			from, to time.Time
		}{
			{name: "future window", from: from, to: to},
			{name: "single settlement time", from: to, to: to},
		} {
			result, err := ex.GetDualInvestmentOrderList(t.Context(), &DualInvestmentOrderListRequest{From: tc.from, To: tc.to, Page: 2})
			require.NoErrorf(t, err, "GetDualInvestmentOrderList must accept a %s", tc.name)
			assert.Emptyf(t, result, "GetDualInvestmentOrderList should return the empty listing for a %s", tc.name)
			assert.Equalf(t, url.Values{"from": {strconv.FormatInt(tc.from.Unix(), 10)}, "to": {strconv.FormatInt(tc.to.Unix(), 10)}, "page": {"2"}}, query, "GetDualInvestmentOrderList should send the %s", tc.name)
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualInvestmentOrderList(t.Context(), &DualInvestmentOrderListRequest{From: startTime, To: endTime, Type: "call", Status: "HOLD", Coin: currency.ETH, Limit: 100})
	require.NoError(t, err, "GetDualInvestmentOrderList must not error")
	if mockTests {
		exp := []*DualInvestmentOrderDetailResponse{{
			ID:               373,
			PlanID:           176,
			InvestAmount:     1.5,
			CreateTime:       types.Time(time.Unix(1744103900, 0)),
			CompleteTime:     types.Time(time.Unix(1744190000, 0)),
			Status:           "INIT",
			InvestCurrency:   currency.ETH,
			ExerciseCurrency: currency.USDT,
			ExercisePrice:    2500,
			DeliveryTime:     types.Time(time.Unix(1744272000, 0)),
			APYDisplay:       0.68,
			Text:             "t-custom-text",
		}}
		assert.Equal(t, exp, result, "GetDualInvestmentOrderList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDualInvestmentOrderList should return a result")
}

func TestPlaceDualInvestmentOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceDualInvestmentOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceDualInvestmentOrder must reject a nil request")

	_, err = e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderRequest{Text: "t-abc"})
	require.ErrorIs(t, err, errPlanIDRequired, "PlaceDualInvestmentOrder must require a plan ID")

	_, err = e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderRequest{PlanID: "12321"})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceDualInvestmentOrder must require an amount")

	_, err = e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderRequest{PlanID: "12321", Amount: 1, Text: "abc"})
	require.ErrorIs(t, err, errInvalidTextPrefix, "PlaceDualInvestmentOrder must require custom text to start with t-")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceDualInvestmentOrder(t.Context(), &DualInvestmentOrderRequest{PlanID: "12321", Amount: 1223.213})
	require.NoError(t, err, "PlaceDualInvestmentOrder must not error")
	if mockTests {
		exp := &DualInvestmentOrderDetailResponse{
			ID:               374,
			PlanID:           12321,
			InvestAmount:     1223.213,
			CreateTime:       types.Time(time.Unix(1744103900, 0)),
			Status:           "INIT",
			InvestCurrency:   currency.USDT,
			ExerciseCurrency: currency.BTC,
			ExercisePrice:    81000,
			DeliveryTime:     types.Time(time.Unix(1744272000, 0)),
			APYDisplay:       0.9401,
		}
		assert.Equal(t, exp, result, "PlaceDualInvestmentOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "PlaceDualInvestmentOrder should return the order")
}

func TestGetDualCurrencyEarningAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualCurrencyEarningAssets(t.Context())
	require.NoError(t, err, "GetDualCurrencyEarningAssets must not error")
	if mockTests {
		exp := &DualCurrencyEarningAssetResponse{
			UserAssetUSDT:         30.13,
			UserAssetBTC:          0.00032798,
			UserTotalInterestUSDT: 581.19003892,
			UserTotalInterestBTC:  0.00632655,
		}
		assert.Equal(t, exp, result, "GetDualCurrencyEarningAssets should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDualCurrencyEarningAssets should return a result")
}

func TestGetDualCurrencyEarlyRedemptionPreview(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualCurrencyEarlyRedemptionPreview(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetDualCurrencyEarlyRedemptionPreview must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualCurrencyEarlyRedemptionPreview(t.Context(), "12345678")
	require.NoError(t, err, "GetDualCurrencyEarlyRedemptionPreview must not error")
	if mockTests {
		exp := &DualCurrencyEarlyRedemptionResponse{
			CreateTime:          types.Time(time.Unix(1775027285, 0)),
			DeliveryTime:        types.Time(time.Unix(1775289600, 0)),
			ExercisePrice:       71303.44279,
			InvestAmount:        1,
			InvestCurrency:      currency.BTC,
			Name:                "BTC_USDT-20260404-71303.44279-C",
			OrderID:             12345678,
			RequestID:           "tIvdY7nh",
			SettlePrice:         68781.0063,
			SettlementAmount:    0.99486528,
			SettlementCurrency:  currency.BTC,
			SettlementPrincipal: 0.99486528,
			Type:                "call",
			MoneyBackTime:       types.Time(time.Unix(1775027958, 0)),
		}
		assert.Equal(t, exp, result, "GetDualCurrencyEarlyRedemptionPreview should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDualCurrencyEarlyRedemptionPreview should return a result")
}

func TestRedeemDualCurrencyOrder(t *testing.T) {
	t.Parallel()
	err := e.RedeemDualCurrencyOrder(t.Context(), "", "OepRSEfv")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "RedeemDualCurrencyOrder must require an order ID")

	err = e.RedeemDualCurrencyOrder(t.Context(), "9487", "")
	require.ErrorIs(t, err, errOrderRefundRequestIDRequired, "RedeemDualCurrencyOrder must require a request ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.RedeemDualCurrencyOrder(t.Context(), "9487", "OepRSEfv")
	assert.NoError(t, err, "RedeemDualCurrencyOrder should not error")
}

func TestModifyDualCurrencyOrderReinvest(t *testing.T) {
	t.Parallel()
	err := e.ModifyDualCurrencyOrderReinvest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifyDualCurrencyOrderReinvest must reject a nil request")

	err = e.ModifyDualCurrencyOrderReinvest(t.Context(), &DualModifyOrderReinvestRequest{Status: 1})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ModifyDualCurrencyOrderReinvest must require an order ID")

	err = e.ModifyDualCurrencyOrderReinvest(t.Context(), &DualModifyOrderReinvestRequest{OrderID: 9497, Status: 2})
	require.ErrorIs(t, err, errInvalidReinvestStatus, "ModifyDualCurrencyOrderReinvest must reject a status other than 0 or 1")

	err = e.ModifyDualCurrencyOrderReinvest(t.Context(), &DualModifyOrderReinvestRequest{OrderID: 9497, Status: 1, EffectiveDuration: fractionalSeconds})
	require.ErrorIs(t, err, errNotWholeSeconds, "ModifyDualCurrencyOrderReinvest must reject an effective duration with a fraction of a second")

	err = e.ModifyDualCurrencyOrderReinvest(t.Context(), &DualModifyOrderReinvestRequest{OrderID: 9497, Status: 1, EffectiveDuration: negativeSeconds})
	require.ErrorIs(t, err, errNotWholeSeconds, "ModifyDualCurrencyOrderReinvest must reject a negative effective duration")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	// Status 0 switches reinvestment off, so it must reach the API rather than be dropped as empty
	for _, status := range []uint8{1, 0} {
		err = e.ModifyDualCurrencyOrderReinvest(t.Context(), &DualModifyOrderReinvestRequest{OrderID: 9497, Status: status})
		assert.NoErrorf(t, err, "ModifyDualCurrencyOrderReinvest should not error for status %d", status)
	}
}

func TestGetDualCurrencyRecommendedProjects(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	project := &DualRecommendedProjectResponse{
		ID:               110669,
		Category:         1,
		Type:             "call",
		InvestCurrency:   currency.BTC,
		ExerciseCurrency: currency.USDT,
		APYDisplay:       2.6,
		ExercisePrice:    71303.44279,
		DeliveryTime:     types.Time(time.Unix(1775289600, 0)),
		MinAmount:        1,
		MaxAmount:        100000,
		MinCopies:        1,
		MaxCopies:        100000,
		InvestDays:       2,
		InvestHours:      70.43458031,
	}
	result, err := e.GetDualCurrencyRecommendedProjects(t.Context(), "senior", currency.EMPTYCODE, "", nil)
	require.NoError(t, err, "GetDualCurrencyRecommendedProjects must not error")
	if mockTests {
		assert.Equal(t, []*DualRecommendedProjectResponse{project}, result, "GetDualCurrencyRecommendedProjects should decode every field")
	} else {
		assert.NotNil(t, result, "GetDualCurrencyRecommendedProjects should return a result")
	}

	result, err = e.GetDualCurrencyRecommendedProjects(t.Context(), "normal", currency.BTC, "call", []uint64{110669, 110670})
	require.NoError(t, err, "GetDualCurrencyRecommendedProjects must not error when excluding products")
	if mockTests {
		exp := *project
		exp.ID = 110671
		assert.Equal(t, []*DualRecommendedProjectResponse{&exp}, result, "GetDualCurrencyRecommendedProjects should decode every field when excluding products")
		return
	}
	assert.NotNil(t, result, "GetDualCurrencyRecommendedProjects should return a result when excluding products")
}

func TestCreateAutoInvestPlan(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAutoInvestPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateAutoInvestPlan must reject a nil request")

	arg := &CreateAutoInvestPlanRequest{}
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateAutoInvestPlan must require a pricing currency")

	arg.PlanMoney = currency.USDT
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CreateAutoInvestPlan must require a plan amount")

	arg.PlanAmount = 10
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, errPeriodTypeRequired, "CreateAutoInvestPlan must require a period type")

	arg.PlanPeriodType = "monthly"
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPlanPeriodDay, "CreateAutoInvestPlan must require a day of the month for monthly plans")

	arg.PlanPeriodType = "weekly"
	arg.PlanPeriodDay = 8
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPlanPeriodDay, "CreateAutoInvestPlan must require a weekday for weekly plans")

	arg.PlanPeriodType = "daily"
	arg.PlanPeriodHour = 24
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidPlanPeriodHour, "CreateAutoInvestPlan must reject an hour beyond 23")

	arg.PlanPeriodHour = 0
	_, err = e.CreateAutoInvestPlan(t.Context(), arg)
	require.ErrorIs(t, err, errNoValidParameterPassed, "CreateAutoInvestPlan must require portfolio items")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateAutoInvestPlan(t.Context(), &CreateAutoInvestPlanRequest{
		PlanMoney:      currency.USDT,
		PlanAmount:     10,
		PlanPeriodType: "monthly",
		PlanPeriodDay:  1,
		Items:          []*AutoInvestPlanItem{{Asset: currency.BTC, Ratio: 100}},
	})
	require.NoError(t, err, "CreateAutoInvestPlan must not error")
	if mockTests {
		exp := &AutoInvestPlanResponse{
			ID:         142579,
			Amount:     10,
			Money:      currency.USDT,
			NextTime:   types.Time(time.Unix(1775005200, 0)),
			PeriodType: "monthly",
			PeriodDay:  1,
			FundFlow:   "auto_invest",
			FundSource: "spot",
		}
		assert.Equal(t, exp, result, "CreateAutoInvestPlan should decode every field")
		return
	}
	assert.NotNil(t, result, "CreateAutoInvestPlan should return the plan")
}

func TestValidateAutoInvestPlanItems(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		items []*AutoInvestPlanItem
		err   error
	}{
		{name: "no items", err: errNoValidParameterPassed},
		{name: "nil item", items: []*AutoInvestPlanItem{nil}, err: common.ErrNilPointer},
		{name: "empty asset", items: []*AutoInvestPlanItem{{Ratio: 100}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing ratio", items: []*AutoInvestPlanItem{{Asset: currency.BTC}}, err: order.ErrAmountMustBeSet},
		{name: "valid", items: []*AutoInvestPlanItem{{Asset: currency.BTC, Ratio: 50}, {Asset: currency.ETH, Ratio: 50}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateAutoInvestPlanItems(tc.items)
			if tc.err == nil {
				assert.NoError(t, err, "validateAutoInvestPlanItems should accept a complete portfolio")
				return
			}
			assert.ErrorIs(t, err, tc.err, "validateAutoInvestPlanItems should return the expected error")
		})
	}
}

func TestUpdateAutoInvestPlan(t *testing.T) {
	t.Parallel()
	err := e.UpdateAutoInvestPlan(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateAutoInvestPlan must reject a nil request")

	err = e.UpdateAutoInvestPlan(t.Context(), &AutoInvestPlanUpdateRequest{FundSource: "earn"})
	require.ErrorIs(t, err, errPlanIDRequired, "UpdateAutoInvestPlan must require a plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.UpdateAutoInvestPlan(t.Context(), &AutoInvestPlanUpdateRequest{PlanID: 142582, FundSource: "earn", FundFlow: "auto_invest"})
	assert.NoError(t, err, "UpdateAutoInvestPlan should not error")
}

func TestStopAutoInvestPlan(t *testing.T) {
	t.Parallel()
	err := e.StopAutoInvestPlan(t.Context(), 0)
	require.ErrorIs(t, err, errPlanIDRequired, "StopAutoInvestPlan must require a plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.StopAutoInvestPlan(t.Context(), 142582)
	assert.NoError(t, err, "StopAutoInvestPlan should not error")
}

func TestAddAutoInvestPlanPosition(t *testing.T) {
	t.Parallel()
	err := e.AddAutoInvestPlanPosition(t.Context(), 0, 10)
	require.ErrorIs(t, err, errPlanIDRequired, "AddAutoInvestPlanPosition must require a plan ID")

	err = e.AddAutoInvestPlanPosition(t.Context(), 142583, 0)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "AddAutoInvestPlanPosition must require an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.AddAutoInvestPlanPosition(t.Context(), 142583, 12.345)
	assert.NoError(t, err, "AddAutoInvestPlanPosition should not error")
}

func TestGetAutoInvestSupportedCoins(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestSupportedCoins(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAutoInvestSupportedCoins must not error")
	if mockTests {
		exp := []*AutoInvestCoinItemResponse{
			{Key: currency.BTC, Value: "BTC", AssetIconURL: "https://icon.staticimgs.com/images/coin_icon/64/btc.png?v=1743408000"},
			{Key: currency.ETH, Value: "ETH", AssetIconURL: "https://icon.staticimgs.com/images/coin_icon/64/eth.png?v=1743408000", Sort: 1},
		}
		assert.Equal(t, exp, result, "GetAutoInvestSupportedCoins should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestSupportedCoins should return a result")
}

func TestGetAutoInvestMinimumAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetAutoInvestMinimumAmount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAutoInvestMinimumAmount must reject a nil request")

	_, err = e.GetAutoInvestMinimumAmount(t.Context(), &AutoInvestMinAmountRequest{Items: []*AutoInvestPlanItem{{Asset: currency.BTC, Ratio: 33}}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAutoInvestMinimumAmount must require a pricing currency")

	_, err = e.GetAutoInvestMinimumAmount(t.Context(), &AutoInvestMinAmountRequest{Money: currency.USDT})
	require.ErrorIs(t, err, errNoValidParameterPassed, "GetAutoInvestMinimumAmount must require portfolio items")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestMinimumAmount(t.Context(), &AutoInvestMinAmountRequest{
		Money: currency.USDT,
		Items: []*AutoInvestPlanItem{
			{Asset: currency.BTC, Ratio: 33},
			{Asset: currency.ETH, Ratio: 33},
			{Asset: currency.SOL, Ratio: 34},
		},
	})
	require.NoError(t, err, "GetAutoInvestMinimumAmount must not error")
	if mockTests {
		assert.Equal(t, &AutoInvestMinAmountResponse{MinAmount: 7.06}, result, "GetAutoInvestMinimumAmount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestMinimumAmount should return a result")
}

func TestGetAutoInvestPlanExecutionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetAutoInvestPlanExecutionRecords(t.Context(), 0, 0, 0)
	require.ErrorIs(t, err, errPlanIDRequired, "GetAutoInvestPlanExecutionRecords must require a plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestPlanExecutionRecords(t.Context(), 12345, 0, 0)
	require.NoError(t, err, "GetAutoInvestPlanExecutionRecords must not error")
	if mockTests {
		exp := &AutoInvestPlanExecutionRecordsResponse{
			Page:      1,
			PageSize:  10,
			TotalPage: 30,
			Total:     299,
			List: []*AutoInvestPlanExecutionRecord{{
				ID:            1770805384918111,
				Type:          "Auto Invest",
				Money:         currency.USDT,
				Asset:         currency.BTC,
				UserID:        21245786,
				PlanID:        12345,
				PlanVersion:   2,
				Amount:        32.60292,
				CreateTime:    types.Time(time.Unix(1773734410, 0)),
				UpdateTime:    types.Time(time.Unix(1773734410, 0)),
				Status:        "Failed",
				StatusType:    2,
				Side:          2,
				StatusMessage: "put order fail",
				Detail:        "insufficient balance",
			}},
		}
		assert.Equal(t, exp, result, "GetAutoInvestPlanExecutionRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestPlanExecutionRecords should return a result")
}

func TestGetAutoInvestPlanOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetAutoInvestPlanOrderDetails(t.Context(), 0, 1)
	require.ErrorIs(t, err, errPlanIDRequired, "GetAutoInvestPlanOrderDetails must require a plan ID")

	_, err = e.GetAutoInvestPlanOrderDetails(t.Context(), 1, 0)
	require.ErrorIs(t, err, errRecordIDRequired, "GetAutoInvestPlanOrderDetails must require a record ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestPlanOrderDetails(t.Context(), 12345, 67890)
	require.NoError(t, err, "GetAutoInvestPlanOrderDetails must not error")
	if mockTests {
		exp := []*AutoInvestOrderItemResponse{{
			ID:         6787627,
			Type:       "Buy",
			Amount:     0.031,
			PlanID:     12345,
			Side:       2,
			Asset:      currency.USDT,
			RecordID:   67890,
			TotalMoney: 2.72583,
			Market:     currency.NewPairWithDelimiter("SOL", "USDT", "/"),
			Price:      87.93,
			CreateTime: types.Time(time.Unix(1773656162, 0)),
			Total:      2.72583,
			FundFlow:   "auto_invest",
			Status:     1,
		}}
		assert.Equal(t, exp, result, "GetAutoInvestPlanOrderDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestPlanOrderDetails should return a result")
}

func TestGetAutoInvestCurrencyConfig(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestCurrencyConfig(t.Context())
	require.NoError(t, err, "GetAutoInvestCurrencyConfig must not error")
	if mockTests {
		exp := []*AutoInvestConfigItemResponse{{Coin: currency.BTC, MaxLimit: 10}, {Coin: currency.USDT, MaxLimit: 100000}}
		assert.Equal(t, exp, result, "GetAutoInvestCurrencyConfig should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestCurrencyConfig should return a result")
}

// autoInvestPlanDetailsFixture returns the auto invest plan recorded for the plan detail and plan list fixtures
func autoInvestPlanDetailsFixture() *AutoInvestPlanDetailsResponse {
	return &AutoInvestPlanDetailsResponse{
		ID:         12345,
		Version:    1,
		Name:       "Auto Invest BTC",
		CreateTime: types.Time(time.Unix(1773733133, 0)),
		UpdateTime: types.Time(time.Unix(1773733133, 0)),
		UserID:     2124569786,
		Money:      currency.USDT,
		Amount:     100,
		PeriodType: "monthly",
		PeriodDay:  1,
		PeriodHour: 1,
		Portfolio: []*AutoInvestPortfolioItem{
			{Asset: currency.BTC, Ratio: 66, CumulativeInvestment: 200, CumulativeHolding: 0.0025, AveragePrice: 80000},
			{Asset: currency.ETH, Ratio: 34, CumulativeInvestment: 100, CumulativeHolding: 0.04, AveragePrice: 2500},
		},
		NextTime:   types.Time(time.Unix(1775005200, 0)),
		Period:     3,
		FundSource: "spot",
		FundFlow:   "auto_invest",
	}
}

func TestGetAutoInvestPlanDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetAutoInvestPlanDetails(t.Context(), 0)
	require.ErrorIs(t, err, errPlanIDRequired, "GetAutoInvestPlanDetails must require a plan ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestPlanDetails(t.Context(), 12345)
	require.NoError(t, err, "GetAutoInvestPlanDetails must not error")
	if mockTests {
		assert.Equal(t, autoInvestPlanDetailsFixture(), result, "GetAutoInvestPlanDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestPlanDetails should return a result")
}

func TestGetAutoInvestPlanList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAutoInvestPlanList(t.Context(), "", 0, 0)
	require.ErrorIs(t, err, errPlanStatusRequired, "GetAutoInvestPlanList must require a status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoInvestPlanList(t.Context(), "active", 1, 100)
	require.NoError(t, err, "GetAutoInvestPlanList must not error")
	if mockTests {
		exp := &AutoInvestPlanListResponse{
			Page:       1,
			PageSize:   100,
			PageCount:  1,
			TotalCount: 1,
			List:       []*AutoInvestPlanDetailsResponse{autoInvestPlanDetailsFixture()},
		}
		assert.Equal(t, exp, result, "GetAutoInvestPlanList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoInvestPlanList should return a result")
}

func TestGetFixedTermProducts(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedTermProducts(t.Context(), currency.EMPTYCODE, 0, 0, 10)
	require.ErrorIs(t, err, errPageRequired, "GetFixedTermProducts must require a page")

	_, err = e.GetFixedTermProducts(t.Context(), currency.EMPTYCODE, 0, 1, 0)
	require.ErrorIs(t, err, errInvalidLimit, "GetFixedTermProducts must require a limit")

	result, err := e.GetFixedTermProducts(t.Context(), currency.EMPTYCODE, 0, 1, 10)
	require.NoError(t, err, "GetFixedTermProducts must not error")
	if mockTests {
		product := &FixedTermProduct{
			ID:                11,
			Name:              "USDT-TEST",
			Asset:             currency.USDT,
			LockUpPeriodDays:  16,
			MinLendAmount:     1,
			UserMaxLendAmount: 100000,
			TotalLendAmount:   100000000,
			YearRate:          0.03,
			Status:            2,
			CreateTime:        time.Date(2024, 12, 13, 3, 29, 41, 103000000, time.UTC),
		}
		otherProduct := *product
		otherProduct.ID = 10
		otherProduct.Name = "\u793a\u4f8b"
		otherProduct.LockUpPeriodDays = 15
		assert.Equal(t, &FixedTermProductsResponse{List: []*FixedTermProduct{product, &otherProduct}, Total: 12}, result, "GetFixedTermProducts should decode every field")
	} else {
		require.NotNil(t, result, "GetFixedTermProducts must return a result")
		assert.NotEmpty(t, result.List, "GetFixedTermProducts should return products")
	}

	result, err = e.GetFixedTermProducts(t.Context(), currency.USDT, 2, 1, 10)
	require.NoError(t, err, "GetFixedTermProducts must not error for VIP products")
	if mockTests {
		exp := &FixedTermProductsResponse{
			List: []*FixedTermProduct{{
				ID:                33,
				Name:              "USDT - V5-7",
				Asset:             currency.USDT,
				LockUpPeriodDays:  7,
				MinLendAmount:     500,
				UserMaxLendAmount: 30000000,
				TotalLendAmount:   30000000,
				YearRate:          0.037,
				Type:              2,
				Reinvest:          true,
				RedeemAccount:     1,
				MinVIP:            5,
				MaxVIP:            16,
				Status:            2,
				CreateTime:        time.Date(2024, 12, 30, 2, 55, 35, 950000000, time.UTC),
				UserMaxLendVolume: 30000000,
				SaleStatus:        1,
			}},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetFixedTermProducts should decode every field for VIP products")
		return
	}
	assert.NotNil(t, result, "GetFixedTermProducts should return a result for VIP products")
}

func TestGetFixedTermProductsAPIError(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	// Fixed-term endpoints report failures inside a 200 response envelope
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v4/earn/fixed-term/product", r.URL.Path, "request path should be the fixed-term product list")
		_, err := fmt.Fprint(w, `{"code":-1,"message":"Service is busy. Please try again later.","data":null,"timestamp":1791854276}`)
		assert.NoError(t, err, "writing the fixed-term error envelope should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	_, err := ex.GetFixedTermProducts(t.Context(), currency.EMPTYCODE, 0, 1, 10)
	assert.ErrorContains(t, err, "Service is busy", "GetFixedTermProducts should surface the envelope error")
}

func TestGetFixedTermProductsByAsset(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedTermProductsByAsset(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFixedTermProductsByAsset must require a currency")

	regular := &FixedTermProductSimple{
		ID:               68,
		Asset:            currency.USDT,
		LockUpPeriodDays: 1,
		YearRate:         0.0289,
		Type:             1,
		PreRedeem:        true,
		Reinvest:         true,
		SimpleEarn:       true,
		RedeemAccount:    1,
		SaleStatus:       1,
	}
	result, err := e.GetFixedTermProductsByAsset(t.Context(), currency.USDT, "")
	require.NoError(t, err, "GetFixedTermProductsByAsset must not error")
	if mockTests {
		assert.Equal(t, &FixedTermProductsByAssetResponse{List: []*FixedTermProductSimple{regular}}, result, "GetFixedTermProductsByAsset should decode every field")
	} else {
		assert.NotNil(t, result, "GetFixedTermProductsByAsset should return a result")
	}

	result, err = e.GetFixedTermProductsByAsset(t.Context(), currency.USDT, "0")
	require.NoError(t, err, "GetFixedTermProductsByAsset must not error for every product type")
	if mockTests {
		vip := &FixedTermProductSimple{
			ID:               284,
			Asset:            currency.USDT,
			LockUpPeriodDays: 7,
			YearRate:         0.037,
			Type:             2,
			Reinvest:         true,
			RedeemAccount:    1,
			MinVIP:           3,
			MaxVIP:           14,
			SaleStatus:       2,
		}
		assert.Equal(t, &FixedTermProductsByAssetResponse{List: []*FixedTermProductSimple{regular, vip}}, result, "GetFixedTermProductsByAsset should decode every field for every product type")
		return
	}
	assert.NotNil(t, result, "GetFixedTermProductsByAsset should return a result for every product type")
}

func TestSetFixedTermSubscriptionOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SetFixedTermSubscriptionOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetFixedTermSubscriptionOrder must reject a nil request")

	_, err = e.SetFixedTermSubscriptionOrder(t.Context(), &FixedTermSubscriptionRequest{Amount: 1})
	require.ErrorIs(t, err, errProductIDRequired, "SetFixedTermSubscriptionOrder must require a product ID")

	_, err = e.SetFixedTermSubscriptionOrder(t.Context(), &FixedTermSubscriptionRequest{ProductID: 1})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SetFixedTermSubscriptionOrder must require an amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetFixedTermSubscriptionOrder(t.Context(), &FixedTermSubscriptionRequest{ProductID: 1, Amount: 1})
	require.NoError(t, err, "SetFixedTermSubscriptionOrder must not error")
	if mockTests {
		assert.Equal(t, &OrderIDResponse{OrderID: 5846120275}, result, "SetFixedTermSubscriptionOrder should decode the order ID")
		return
	}
	assert.NotNil(t, result, "SetFixedTermSubscriptionOrder should return the order ID")
}

func TestGetFixedTermSubscriptionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedTermSubscriptionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedTermSubscriptionOrders must reject a nil request")

	arg := &FixedTermSubscriptionOrdersRequest{ProductID: 11}
	_, err = e.GetFixedTermSubscriptionOrders(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "GetFixedTermSubscriptionOrders must require an order type")

	arg.OrderType = 1
	_, err = e.GetFixedTermSubscriptionOrders(t.Context(), arg)
	require.ErrorIs(t, err, errPageRequired, "GetFixedTermSubscriptionOrders must require a page")

	arg.Page = 1
	_, err = e.GetFixedTermSubscriptionOrders(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidLimit, "GetFixedTermSubscriptionOrders must require a limit")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.Limit = 10
	result, err := e.GetFixedTermSubscriptionOrders(t.Context(), arg)
	require.NoError(t, err, "GetFixedTermSubscriptionOrders must not error")
	if mockTests {
		settlement := time.Date(2025, 8, 8, 0, 0, 0, 0, time.UTC)
		exp := &FixedTermSubscriptionOrdersResponse{
			List: []*FixedTermLendOrder{{
				ID:                   83402,
				Business:             1,
				OrderID:              5846120275,
				UserID:               2024103749,
				Asset:                currency.USDT,
				ProductID:            11,
				LockUpPeriodDays:     23,
				Principal:            44.12,
				YearRate:             0.3,
				ProductType:          2,
				Interest:             0.83404931,
				Status:               1,
				ReinvestStatus:       1,
				RedeemAccountType:    1,
				RedeemType:           2,
				RedeemTime:           settlement,
				FinishTime:           settlement,
				CreateTime:           time.Date(2025, 7, 15, 3, 55, 7, 304000000, time.UTC),
				YearRatePercent:      30,
				TotalYearRatePercent: 63.13,
				TotalInterest:        0.83996629,
				ProductInfo:          &FixedTermProductInfo{PreRedeem: true, Reinvest: true, RedeemAccount: 1, MinVIP: 1, MaxVIP: 16},
				BonusInfo: &FixedTermBonusInfo{
					ID:                   142,
					ProductID:            11,
					Asset:                currency.USDT,
					BonusAsset:           currency.GT,
					KYCLimit:             "1,2,3,4",
					LadderAPR:            []*FixedTermLadderAPR{{APR: 0.1, Right: 1}, {APR: 0.2, Left: 1, Right: 2}},
					TotalBonusAmount:     100000,
					UserTotalBonusAmount: 100000,
					Status:               2,
					StartTime:            time.Date(2025, 6, 20, 6, 57, 46, 0, time.UTC),
					EndTime:              time.Date(2026, 6, 29, 16, 0, 0, 0, time.UTC),
					CreateTime:           time.Date(2025, 6, 20, 6, 57, 54, 796000000, time.UTC),
					StartAt:              types.Time(time.Unix(1750402666, 0)),
					EndAt:                types.Time(time.Unix(1782748800, 0)),
					TotalIssuedAmount:    19.49,
					BonusAssetPrice:      17.66,
					ProductAssetPrice:    1,
					ProductYearRate:      0.3,
				},
				CouponInfo: &FixedTermCouponInfo{
					ID:              63,
					Business:        1,
					UserID:          2024103749,
					Asset:           currency.USDT,
					OrderID:         5846120275,
					FinancialRateID: 1506,
					BuyLimitLow:     1,
					BuyLimitHigh:    3,
					RateDays:        30,
					RateRatio:       3.13,
					CouponDays:      23,
					CouponPrincipal: 3,
					CouponYearRate:  0.0313,
					CouponInterest:  0.00591698,
					Status:          2,
					FinishTime:      settlement,
					CreateTime:      time.Date(2025, 7, 15, 3, 55, 7, 285000000, time.UTC),
				},
				RedeemAt: types.Time(time.Unix(1754611200, 0)),
				FinishAt: types.Time(time.Unix(1754611200, 0)),
				CreateAt: types.Time(time.Unix(1752551707, 0)),
				Icon:     "https://icon.gateimg.com/images/coin_icon/64/usdt.png",
			}},
			Total: 51,
		}
		assert.Equal(t, exp, result, "GetFixedTermSubscriptionOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFixedTermSubscriptionOrders should return a result")
}

func TestRedeemFixedTermOrder(t *testing.T) {
	t.Parallel()
	err := e.RedeemFixedTermOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "RedeemFixedTermOrder must require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.RedeemFixedTermOrder(t.Context(), "5862476630")
	assert.NoError(t, err, "RedeemFixedTermOrder should not error")
}

func TestGetFixedTermSubscriptionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedTermSubscriptionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFixedTermSubscriptionHistory must reject a nil request")

	arg := &FixedTermHistoryRequest{}
	_, err = e.GetFixedTermSubscriptionHistory(t.Context(), arg)
	require.ErrorIs(t, err, errHistoryTypeRequired, "GetFixedTermSubscriptionHistory must require a history type")

	arg.Type = 5
	_, err = e.GetFixedTermSubscriptionHistory(t.Context(), arg)
	require.ErrorIs(t, err, errHistoryTypeRequired, "GetFixedTermSubscriptionHistory must reject an unknown history type")

	arg.Type = 1
	_, err = e.GetFixedTermSubscriptionHistory(t.Context(), arg)
	require.ErrorIs(t, err, errPageRequired, "GetFixedTermSubscriptionHistory must require a page")

	arg.Page = 1
	_, err = e.GetFixedTermSubscriptionHistory(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidLimit, "GetFixedTermSubscriptionHistory must require a limit")

	startTime, endTime := getTime()
	_, err = e.GetFixedTermSubscriptionHistory(t.Context(), &FixedTermHistoryRequest{Type: 1, Page: 1, Limit: 10, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFixedTermSubscriptionHistory must reject a reversed time range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	record := &FixedTermHistoryRecord{
		ID:               10001,
		OrderID:          5846120275,
		UserID:           2024103749,
		Asset:            currency.USDT,
		UniqueTime:       "2025-07-15",
		ProductID:        11,
		TotalPrincipal:   44.12,
		Amount:           44.12,
		AssetPrice:       1,
		Status:           1,
		Detail:           "subscribe",
		CreateTime:       time.Date(2025, 7, 15, 3, 55, 7, 304000000, time.UTC),
		CreateAt:         types.Time(time.Unix(1752551707, 0)),
		LockUpPeriodDays: 23,
	}
	arg.Limit = 10
	result, err := e.GetFixedTermSubscriptionHistory(t.Context(), arg)
	require.NoError(t, err, "GetFixedTermSubscriptionHistory must not error")
	if mockTests {
		assert.Equal(t, &FixedTermHistoryResponse{List: []*FixedTermHistoryRecord{record}, Total: 1}, result, "GetFixedTermSubscriptionHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetFixedTermSubscriptionHistory should return a result")
	}

	result, err = e.GetFixedTermSubscriptionHistory(t.Context(), &FixedTermHistoryRequest{
		Type:           3,
		Page:           1,
		Limit:          10,
		ProductID:      11,
		Asset:          currency.USDT,
		StartTime:      startTime,
		EndTime:        endTime,
		BusinessFilter: []FixedTermBusinessFilterRequest{{Business: 1}},
	})
	require.NoError(t, err, "GetFixedTermSubscriptionHistory must not error with filters")
	if mockTests {
		bonus := *record
		bonus.ID = 10002
		bonus.UniqueTime = "2025-04-09"
		bonus.BonusID = 142
		bonus.BonusAsset = currency.GT
		bonus.Amount = 0.00091
		bonus.AssetPrice = 17.66
		bonus.Detail = "bonus"
		bonus.CreateTime = time.Date(2025, 4, 9, 0, 0, 0, 0, time.UTC)
		bonus.CreateAt = types.Time(time.Unix(1744156800, 0))
		assert.Equal(t, &FixedTermHistoryResponse{List: []*FixedTermHistoryRecord{&bonus}, Total: 1}, result, "GetFixedTermSubscriptionHistory should decode every field with filters")
		return
	}
	assert.NotNil(t, result, "GetFixedTermSubscriptionHistory should return a result with filters")
}

func TestSetFixedTermBusinessFilter(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	require.NoError(t, setFixedTermBusinessFilter(params, nil), "setFixedTermBusinessFilter must not error without filters")
	assert.Empty(t, params, "setFixedTermBusinessFilter should not set a parameter without filters")

	require.NoError(t, setFixedTermBusinessFilter(params, []FixedTermBusinessFilterRequest{{Business: 1}, {Business: 2, SubBusiness: 13}}), "setFixedTermBusinessFilter must not error")
	assert.Equal(t, `[{"business":1,"sub_business":0},{"business":2,"sub_business":13}]`, params.Get("business_filter"), "setFixedTermBusinessFilter should encode the filters as a JSON array")
}
