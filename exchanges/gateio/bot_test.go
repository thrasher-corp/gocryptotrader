package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestBotResponseError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, botResponseError(0, "success", "trace"), "a zero code should not error")
	assert.NoError(t, botResponseError(200, "success", "trace"), "a 200 code should not error")
	err := botResponseError(40001, "strategy not found", "trace_011")
	require.Error(t, err, "a failure code must error")
	assert.ErrorContains(t, err, "strategy not found", "error should carry the message")
	assert.ErrorContains(t, err, "trace_011", "error should carry the trace ID")
}

func TestBotResponsesAsError(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		ok, failed func() error
	}{
		"BotDiscoverResponse":         {ok: (&BotDiscoverResponse{Code: 200}).AsError, failed: (&BotDiscoverResponse{Code: 4, TraceID: "trace-1"}).AsError},
		"BotCreateResponse":           {ok: (&BotCreateResponse{Code: 200}).AsError, failed: (&BotCreateResponse{Code: 4, TraceID: "trace-1"}).AsError},
		"BotPortfolioRunningResponse": {ok: (&BotPortfolioRunningResponse{Code: 200}).AsError, failed: (&BotPortfolioRunningResponse{Code: 4, TraceID: "trace-1"}).AsError},
		"BotPortfolioDetailResponse":  {ok: (&BotPortfolioDetailResponse{Code: 200}).AsError, failed: (&BotPortfolioDetailResponse{Code: 4, TraceID: "trace-1"}).AsError},
		"BotPortfolioStopResponse":    {ok: (&BotPortfolioStopResponse{Code: 200}).AsError, failed: (&BotPortfolioStopResponse{Code: 4, TraceID: "trace-1"}).AsError},
	} {
		assert.NoErrorf(t, tc.ok(), "%s should not error on success", name)
		assert.ErrorContainsf(t, tc.failed(), "trace-1", "%s should error with the trace ID on failure", name)
	}
}

func TestGetBotStrategyRecommendations(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBotStrategyRecommendations(t.Context(), &GetBotStrategyRecommendationsRequest{
		Market:       "BTC_USDT",
		StrategyType: BotStrategySpotGrid,
		Scene:        BotSceneTop1,
	})
	require.NoError(t, err, "GetBotStrategyRecommendations must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetBotStrategyRecommendations should return recommendations")
		return
	}
	exp := &BotDiscoverData{
		Scene: BotSceneTop1,
		Recommendations: []*BotRecommendation{{
			RecommendationID:      "spot_grid|BTC_USDT|bt_001",
			Market:                "BTC_USDT",
			StrategyType:          BotStrategySpotGrid,
			StrategyName:          "BTC_USDT Spot Grid",
			BacktestAPR:           "0.35",
			MaxDrawdown:           "0.12",
			Summary:               "Recommended spot grid for BTC_USDT",
			StrategyParamsPreview: `{"low_price":"90000","high_price":"110000","grid_num":"20"}`,
		}},
		UnsupportedFilters: []string{"direction"},
	}
	assert.Equal(t, exp, result, "GetBotStrategyRecommendations should decode every field")

	// Without parameters Gate infers the scene from its own logic
	result, err = e.GetBotStrategyRecommendations(t.Context(), nil)
	require.NoError(t, err, "GetBotStrategyRecommendations must not error without parameters")
	assert.Equal(t, exp, result, "GetBotStrategyRecommendations should decode every field without parameters")
}

func TestCreateSpotGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSpotGridBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSpotGridBot must reject a nil request")
	_, err = e.CreateSpotGridBot(t.Context(), &SpotGridCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired, "CreateSpotGridBot must require a market")

	arg := &SpotGridCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateSpotGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMoneyRequired, "CreateSpotGridBot must require an investment")
	arg.CreateParams.Money = 1000
	_, err = e.CreateSpotGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotLowPriceRequired, "CreateSpotGridBot must require a low price")
	arg.CreateParams.LowPrice = 90000
	_, err = e.CreateSpotGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotHighPriceRequired, "CreateSpotGridBot must require a high price")
	arg.CreateParams.HighPrice = 110000
	_, err = e.CreateSpotGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotGridNumRequired, "CreateSpotGridBot must require a grid number")
	arg.CreateParams.GridNumber = 20
	arg.CreateParams.PriceType = 2
	_, err = e.CreateSpotGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotPriceTypeInvalid, "CreateSpotGridBot must reject an invalid price type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.CreateParams.PriceType = BotPriceTypeGeometric
	result, err := e.CreateSpotGridBot(t.Context(), arg)
	require.NoError(t, err, "CreateSpotGridBot must not error")
	if mockTests {
		exp := &BotCreateData{StrategyID: "sg_001", StrategyType: BotStrategySpotGrid, Market: "BTC_USDT", Status: "running", JumpURL: "https://www.gate.com/bot/spot-grid/sg_001"}
		assert.Equal(t, exp, result, "CreateSpotGridBot should decode every field")
		return
	}
	assert.NotEmpty(t, result.StrategyID, "CreateSpotGridBot should return a strategy ID")
}

func TestCreateMarginGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateMarginGridBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateMarginGridBot must reject a nil request")
	_, err = e.CreateMarginGridBot(t.Context(), &MarginGridCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired, "CreateMarginGridBot must require a market")

	arg := &MarginGridCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMoneyRequired, "CreateMarginGridBot must require an investment")
	arg.CreateParams.Money = 1000
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotLowPriceRequired, "CreateMarginGridBot must require a low price")
	arg.CreateParams.LowPrice = 90000
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotHighPriceRequired, "CreateMarginGridBot must require a high price")
	arg.CreateParams.HighPrice = 110000
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotGridNumRequired, "CreateMarginGridBot must require a grid number")
	arg.CreateParams.GridNumber = 20
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotLeverageRequired, "CreateMarginGridBot must require a leverage")
	arg.CreateParams.Leverage = 3
	arg.CreateParams.PriceType = 2
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotPriceTypeInvalid, "CreateMarginGridBot must reject an invalid price type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.CreateParams.PriceType = BotPriceTypeArithmetic
	arg.CreateParams.Direction = order.Long.Lower()
	result, err := e.CreateMarginGridBot(t.Context(), arg)
	require.NoError(t, err, "CreateMarginGridBot must not error")
	if mockTests {
		exp := &BotCreateData{StrategyID: "mg_001", StrategyType: BotStrategyMarginGrid, Market: "BTC_USDT", Status: "running", JumpURL: "https://www.gate.com/bot/margin-grid/mg_001"}
		assert.Equal(t, exp, result, "CreateMarginGridBot should decode every field")
		return
	}
	assert.NotEmpty(t, result.StrategyID, "CreateMarginGridBot should return a strategy ID")
}

func TestCreateInfiniteGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateInfiniteGridBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateInfiniteGridBot must reject a nil request")
	_, err = e.CreateInfiniteGridBot(t.Context(), &InfiniteGridCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired, "CreateInfiniteGridBot must require a market")

	arg := &InfiniteGridCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateInfiniteGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMoneyRequired, "CreateInfiniteGridBot must require an investment")
	arg.CreateParams.Money = 1000
	_, err = e.CreateInfiniteGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotPriceFloorRequired, "CreateInfiniteGridBot must require a price floor")
	arg.CreateParams.PriceFloor = 80000
	_, err = e.CreateInfiniteGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotProfitPerGridRequired, "CreateInfiniteGridBot must require a profit per grid")
	arg.CreateParams.ProfitPerGrid = 0.003
	invalidPriceType := BotPriceType(2)
	arg.CreateParams.PriceType = &invalidPriceType
	_, err = e.CreateInfiniteGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotPriceTypeInvalid, "CreateInfiniteGridBot must reject an invalid price type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.CreateParams.PriceType = nil
	result, err := e.CreateInfiniteGridBot(t.Context(), arg)
	require.NoError(t, err, "CreateInfiniteGridBot must not error")
	if mockTests {
		exp := &BotCreateData{StrategyID: "ig_001", StrategyType: BotStrategyInfiniteGrid, Market: "BTC_USDT", Status: "running", JumpURL: "https://www.gate.com/bot/infinite-grid/ig_001"}
		assert.Equal(t, exp, result, "CreateInfiniteGridBot should decode every field")
		return
	}
	assert.NotEmpty(t, result.StrategyID, "CreateInfiniteGridBot should return a strategy ID")
}

func TestCreateFuturesGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFuturesGridBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFuturesGridBot must reject a nil request")
	_, err = e.CreateFuturesGridBot(t.Context(), &FuturesGridCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired, "CreateFuturesGridBot must require a market")

	arg := &FuturesGridCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMoneyRequired, "CreateFuturesGridBot must require an investment")
	arg.CreateParams.Money = 1000
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotLowPriceRequired, "CreateFuturesGridBot must require a low price")
	arg.CreateParams.LowPrice = 90000
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotHighPriceRequired, "CreateFuturesGridBot must require a high price")
	arg.CreateParams.HighPrice = 110000
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotGridNumRequired, "CreateFuturesGridBot must require a grid number")
	arg.CreateParams.GridNumber = 20
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotLeverageRequired, "CreateFuturesGridBot must require a leverage")
	arg.CreateParams.Leverage = 5
	arg.CreateParams.PriceType = 2
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotPriceTypeInvalid, "CreateFuturesGridBot must reject an invalid price type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.CreateParams.PriceType = BotPriceTypeArithmetic
	arg.CreateParams.Direction = order.Long.Lower()
	result, err := e.CreateFuturesGridBot(t.Context(), arg)
	require.NoError(t, err, "CreateFuturesGridBot must not error")
	if mockTests {
		exp := &BotCreateData{StrategyID: "fg_001", StrategyType: BotStrategyFuturesGrid, Market: "BTC_USDT", Status: "running", JumpURL: "https://www.gate.com/bot/futures-grid/fg_001"}
		assert.Equal(t, exp, result, "CreateFuturesGridBot should decode every field")
		return
	}
	assert.NotEmpty(t, result.StrategyID, "CreateFuturesGridBot should return a strategy ID")
}

func TestInfiniteGridCreateParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	arithmetic := BotPriceTypeArithmetic
	geometric := BotPriceTypeGeometric
	for _, tc := range []struct {
		name        string
		params      *InfiniteGridCreateParams
		contains    []string
		notContains []string
	}{
		{
			name:        "unset price type is omitted",
			params:      &InfiniteGridCreateParams{},
			notContains: []string{`"price_type"`},
		},
		{
			name:     "arithmetic price type is included",
			params:   &InfiniteGridCreateParams{PriceType: &arithmetic},
			contains: []string{`"price_type":0`},
		},
		{
			name:     "geometric price type is included",
			params:   &InfiniteGridCreateParams{PriceType: &geometric},
			contains: []string{`"price_type":1`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.params)
			require.NoError(t, err, "Marshal must not error")
			for _, exp := range tc.contains {
				assert.Containsf(t, string(got), exp, "payload should contain %s", exp)
			}
			for _, unexp := range tc.notContains {
				assert.NotContainsf(t, string(got), unexp, "payload should not contain %s", unexp)
			}
		})
	}
}

func TestCreateSpotMartingaleBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSpotMartingaleBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSpotMartingaleBot must reject a nil request")
	_, err = e.CreateSpotMartingaleBot(t.Context(), &SpotMartingaleCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired, "CreateSpotMartingaleBot must require a market")

	arg := &SpotMartingaleCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateSpotMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotInvestAmountRequired, "CreateSpotMartingaleBot must require an investment")
	arg.CreateParams.InvestAmount = 1000
	_, err = e.CreateSpotMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotPriceDeviationRequired, "CreateSpotMartingaleBot must require a price deviation")
	arg.CreateParams.PriceDeviation = 0.02
	_, err = e.CreateSpotMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMaxOrdersRequired, "CreateSpotMartingaleBot must require a maximum order count")
	arg.CreateParams.MaxOrders = 5
	_, err = e.CreateSpotMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotTakeProfitRequired, "CreateSpotMartingaleBot must require a take profit ratio")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.CreateParams.TakeProfitRatio = 0.05
	result, err := e.CreateSpotMartingaleBot(t.Context(), arg)
	require.NoError(t, err, "CreateSpotMartingaleBot must not error")
	if mockTests {
		exp := &BotCreateData{StrategyID: "sm_001", StrategyType: BotStrategySpotMartingale, Market: "BTC_USDT", Status: "running", JumpURL: "https://www.gate.com/bot/spot-martingale/sm_001"}
		assert.Equal(t, exp, result, "CreateSpotMartingaleBot should decode every field")
		return
	}
	assert.NotEmpty(t, result.StrategyID, "CreateSpotMartingaleBot should return a strategy ID")
}

func TestCreateContractMartingaleBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateContractMartingaleBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateContractMartingaleBot must reject a nil request")
	_, err = e.CreateContractMartingaleBot(t.Context(), &ContractMartingaleCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired, "CreateContractMartingaleBot must require a market")

	arg := &ContractMartingaleCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotInvestAmountRequired, "CreateContractMartingaleBot must require an investment")
	arg.CreateParams.InvestAmount = 1000
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotPriceDeviationRequired, "CreateContractMartingaleBot must require a price deviation")
	arg.CreateParams.PriceDeviation = 0.02
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMaxOrdersRequired, "CreateContractMartingaleBot must require a maximum order count")
	arg.CreateParams.MaxOrders = 5
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotTakeProfitRequired, "CreateContractMartingaleBot must require a take profit ratio")
	arg.CreateParams.TakeProfitRatio = 0.05
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotDirectionRequired, "CreateContractMartingaleBot must require a direction")
	arg.CreateParams.Direction = order.Buy.Lower()
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotLeverageRequired, "CreateContractMartingaleBot must require a leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.CreateParams.Leverage = 5
	result, err := e.CreateContractMartingaleBot(t.Context(), arg)
	require.NoError(t, err, "CreateContractMartingaleBot must not error")
	if mockTests {
		exp := &BotCreateData{StrategyID: "cm_001", StrategyType: BotStrategyContractMartingale, Market: "BTC_USDT", Status: "running", JumpURL: "https://www.gate.com/bot/contract-martingale/cm_001"}
		assert.Equal(t, exp, result, "CreateContractMartingaleBot should decode every field")
		return
	}
	assert.NotEmpty(t, result.StrategyID, "CreateContractMartingaleBot should return a strategy ID")
}

func TestGetBotRunningStrategies(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBotRunningStrategies(t.Context(), BotStrategySpotGrid, "BTC_USDT", 1, 10)
	require.NoError(t, err, "GetBotRunningStrategies must not error")
	if mockTests {
		exp := &BotPortfolioRunningData{
			Items: []*BotPortfolioRunningItem{{
				StrategyID:   "sg_001",
				StrategyType: BotStrategySpotGrid,
				StrategyName: "BTC_USDT Spot Grid",
				Market:       "BTC_USDT",
				Status:       "running",
				PNL:          120.5,
				PNLRate:      0.12,
				InvestAmount: 1000,
				CreatedAt:    "2026-01-01 00:00:00",
			}},
			Page:     1,
			PageSize: 10,
			Total:    1,
		}
		assert.Equal(t, exp, result, "GetBotRunningStrategies should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBotRunningStrategies should return running strategies")
}

func TestGetBotStrategyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetBotStrategyDetail(t.Context(), "", BotStrategySpotGrid)
	require.ErrorIs(t, err, errBotStrategyIDRequired, "GetBotStrategyDetail must require a strategy ID")
	_, err = e.GetBotStrategyDetail(t.Context(), "sg_001", "")
	require.ErrorIs(t, err, errBotStrategyTypeRequired, "GetBotStrategyDetail must require a strategy type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBotStrategyDetail(t.Context(), "sg_001", BotStrategySpotGrid)
	require.NoError(t, err, "GetBotStrategyDetail must not error")
	if mockTests {
		exp := &BotPortfolioDetailData{
			StrategyID:   "sg_001",
			StrategyType: BotStrategySpotGrid,
			Market:       "BTC_USDT",
			Status:       "running",
			BaseInfo: &BotPortfolioBaseInfo{
				StrategyName:    "BTC_USDT Spot Grid",
				CreatedAt:       "2026-01-01 00:00:00",
				RunningDuration: 86400,
				InvestAmount:    1000,
				TotalProfit:     120.5,
				ProfitRate:      0.12,
			},
			Metrics: &BotPortfolioMetrics{
				GridProfit:                100,
				FloatingPNL:               20.5,
				ArbitrageCount:            42,
				PriceRange:                "90000-110000",
				GridCount:                 20,
				EstimatedLiquidationPrice: 85000,
				PriceFloor:                80000,
				GridProfitRate:            0.1,
				RealisedPNL:               100,
				FinishedRounds:            3,
				AverageCost:               99000,
				TakeProfitPrice:           115000,
				MaintenanceMarginRatio:    0.05,
			},
			Position: &BotPortfolioPosition{
				Amount:        0.01,
				EntryPrice:    100000,
				QuoteAmount:   500,
				PositionValue: 1000,
				Margin:        200,
				Side:          "long",
			},
			StopSupported: true,
		}
		assert.Equal(t, exp, result, "GetBotStrategyDetail should decode every field")
		return
	}
	assert.NotEmpty(t, result.StrategyID, "GetBotStrategyDetail should return the strategy")
}

func TestStopBotStrategy(t *testing.T) {
	t.Parallel()
	_, err := e.StopBotStrategy(t.Context(), "", BotStrategySpotGrid)
	require.ErrorIs(t, err, errBotStrategyIDRequired, "StopBotStrategy must require a strategy ID")
	_, err = e.StopBotStrategy(t.Context(), "sg_001", "")
	require.ErrorIs(t, err, errBotStrategyTypeRequired, "StopBotStrategy must require a strategy type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopBotStrategy(t.Context(), "sg_001", BotStrategySpotGrid)
	require.NoError(t, err, "StopBotStrategy must not error")
	if !mockTests {
		assert.NotEmpty(t, result.StrategyID, "StopBotStrategy should return the strategy")
		return
	}
	exp := &BotPortfolioStopData{StrategyID: "sg_001", StrategyType: BotStrategySpotGrid, Status: "stopping", ResultMessage: "Strategy is being stopped"}
	assert.Equal(t, exp, result, "StopBotStrategy should decode every field")

	_, err = e.StopBotStrategy(t.Context(), "sg_002", BotStrategySpotGrid)
	assert.ErrorContains(t, err, "trace_011", "StopBotStrategy should return the bot error with its trace ID")
}
