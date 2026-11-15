package gate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
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

func TestBotDiscoverResponseAsError(t *testing.T) {
	t.Parallel()
	t.Run("success code", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, (&BotDiscoverResponse{Code: 200}).AsError(), "AsError should not error on success")
	})
	t.Run("failure code", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, (&BotDiscoverResponse{Code: 4, TraceID: "trace-1"}).AsError(), "bot api error code: 4 message:  trace id: trace-1", "AsError should error with the code and trace ID on failure")
	})
}

func TestBotCreateResponseAsError(t *testing.T) {
	t.Parallel()
	t.Run("success code", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, (&BotCreateResponse{Code: 200}).AsError(), "AsError should not error on success")
	})
	t.Run("failure code", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, (&BotCreateResponse{Code: 4, TraceID: "trace-1"}).AsError(), "bot api error code: 4 message:  trace id: trace-1", "AsError should error with the code and trace ID on failure")
	})
}

func TestBotPortfolioRunningResponseAsError(t *testing.T) {
	t.Parallel()
	t.Run("success code", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, (&BotPortfolioRunningResponse{Code: 200}).AsError(), "AsError should not error on success")
	})
	t.Run("failure code", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, (&BotPortfolioRunningResponse{Code: 4, TraceID: "trace-1"}).AsError(), "bot api error code: 4 message:  trace id: trace-1", "AsError should error with the code and trace ID on failure")
	})
}

func TestBotPortfolioDetailResponseAsError(t *testing.T) {
	t.Parallel()
	t.Run("success code", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, (&BotPortfolioDetailResponse{Code: 200}).AsError(), "AsError should not error on success")
	})
	t.Run("failure code", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, (&BotPortfolioDetailResponse{Code: 4, TraceID: "trace-1"}).AsError(), "bot api error code: 4 message:  trace id: trace-1", "AsError should error with the code and trace ID on failure")
	})
}

func TestBotPortfolioStopResponseAsError(t *testing.T) {
	t.Parallel()
	t.Run("success code", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, (&BotPortfolioStopResponse{Code: 200}).AsError(), "AsError should not error on success")
	})
	t.Run("failure code", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, (&BotPortfolioStopResponse{Code: 4, TraceID: "trace-1"}).AsError(), "bot api error code: 4 message:  trace id: trace-1", "AsError should error with the code and trace ID on failure")
	})
}

func TestGetBotStrategyRecommendations(t *testing.T) {
	t.Parallel()
	t.Run("null reply returns an error", func(t *testing.T) {
		t.Parallel()
		_, err := newFixedResponseExchange(t, "/api/v4/bot/strategy/recommend", `null`).GetBotStrategyRecommendations(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrNoResponse, "GetBotStrategyRecommendations should report a null reply")
	})

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
	t.Run("null reply returns an error", func(t *testing.T) {
		t.Parallel()
		arg := &SpotGridCreateRequest{
			Market:       "BTC_USDT",
			CreateParams: SpotGridCreateParams{Money: 1000, LowPrice: 90000, HighPrice: 110000, GridNumber: 20, PriceType: BotPriceTypeGeometric},
		}
		_, err := newFixedResponseExchange(t, "/api/v4/bot/spot-grid/create", `null`).CreateSpotGridBot(t.Context(), arg)
		assert.ErrorIs(t, err, common.ErrNoResponse, "CreateSpotGridBot should report a null reply")
	})

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

func TestCreateBot(t *testing.T) {
	t.Parallel()
	newRequest := func() *SpotGridCreateRequest {
		return &SpotGridCreateRequest{
			StrategyType: BotStrategySpotGrid,
			Market:       "BTC_USDT",
			CreateParams: SpotGridCreateParams{Money: 1000, LowPrice: 90000, HighPrice: 110000, GridNumber: 20, PriceType: BotPriceTypeGeometric},
		}
	}

	// Serial, as it calls the shared e
	t.Run("creates the strategy", func(t *testing.T) {
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.createBot(t.Context(), botSpotGridCreateEPL, "bot/spot-grid/create", newRequest())
		require.NoError(t, err, "createBot must not error")
		if mockTests {
			exp := &BotCreateData{StrategyID: "sg_001", StrategyType: BotStrategySpotGrid, Market: "BTC_USDT", Status: "running", JumpURL: "https://www.gate.com/bot/spot-grid/sg_001"}
			assert.Equal(t, exp, result, "createBot should decode every field")
			return
		}
		assert.NotEmpty(t, result.StrategyID, "createBot should return a strategy ID")
	})

	t.Run("request error is returned", func(t *testing.T) {
		t.Parallel()
		_, err := unsignedExchange(t).createBot(t.Context(), botSpotGridCreateEPL, "bot/spot-grid/create", newRequest())
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "createBot should return the request's error")
	})

	t.Run("failure code returns the bot error", func(t *testing.T) {
		t.Parallel()
		ex := newFixedResponseExchange(t, "/api/v4/bot/spot-grid/create", `{"code":40001,"message":"strategy not found","data":null,"trace_id":"trace_011"}`)
		_, err := ex.createBot(t.Context(), botSpotGridCreateEPL, "bot/spot-grid/create", newRequest())
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "createBot must return a failed request")
		assert.ErrorContains(t, err, "bot api error code: 40001 message: strategy not found trace id: trace_011", "createBot should return the bot error with its trace ID")
	})

	t.Run("null reply returns an error", func(t *testing.T) {
		t.Parallel()
		_, err := newFixedResponseExchange(t, "/api/v4/bot/spot-grid/create", `null`).createBot(t.Context(), botSpotGridCreateEPL, "bot/spot-grid/create", newRequest())
		assert.ErrorIs(t, err, common.ErrNoResponse, "createBot should report a null reply")
	})
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

	geometric := *arg
	geometric.CreateParams.PriceType = BotPriceTypeGeometric
	t.Run("geometric price type is allowed", func(t *testing.T) {
		t.Parallel()
		_, err := unsignedExchange(t).CreateMarginGridBot(t.Context(), &geometric)
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "CreateMarginGridBot should allow the geometric price type and fail only at signing")
	})

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

	geometric := *arg
	geometricPriceType := BotPriceTypeGeometric
	geometric.CreateParams.PriceType = &geometricPriceType
	t.Run("geometric price type is allowed", func(t *testing.T) {
		t.Parallel()
		_, err := unsignedExchange(t).CreateInfiniteGridBot(t.Context(), &geometric)
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "CreateInfiniteGridBot should allow the geometric price type and fail only at signing")
	})

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

	geometric := *arg
	geometric.CreateParams.PriceType = BotPriceTypeGeometric
	t.Run("geometric price type is allowed", func(t *testing.T) {
		t.Parallel()
		_, err := unsignedExchange(t).CreateFuturesGridBot(t.Context(), &geometric)
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "CreateFuturesGridBot should allow the geometric price type and fail only at signing")
	})

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
	t.Run("null reply returns an error", func(t *testing.T) {
		t.Parallel()
		_, err := newFixedResponseExchange(t, "/api/v4/bot/portfolio/running", `null`).GetBotRunningStrategies(t.Context(), BotStrategySpotGrid, "BTC_USDT", 1, 10)
		assert.ErrorIs(t, err, common.ErrNoResponse, "GetBotRunningStrategies should report a null reply")
	})

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
	t.Run("null reply returns an error", func(t *testing.T) {
		t.Parallel()
		_, err := newFixedResponseExchange(t, "/api/v4/bot/portfolio/detail", `null`).GetBotStrategyDetail(t.Context(), "sg_001", BotStrategySpotGrid)
		assert.ErrorIs(t, err, common.ErrNoResponse, "GetBotStrategyDetail should report a null reply")
	})

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
		exp := &BotPortfolioDetail{
			StrategyID:   "sg_001",
			StrategyType: BotStrategySpotGrid,
			Market:       "BTC_USDT",
			Status:       "running",
			BaseInfo: &BotPortfolioBaseInfo{
				StrategyName:    "BTC_USDT Spot Grid",
				CreatedAt:       "2026-01-01 00:00:00",
				RunningDuration: types.DurationSeconds(24 * time.Hour),
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
	t.Run("null reply returns an error", func(t *testing.T) {
		t.Parallel()
		_, err := newFixedResponseExchange(t, "/api/v4/bot/portfolio/stop", `null`).StopBotStrategy(t.Context(), "sg_001", BotStrategySpotGrid)
		assert.ErrorIs(t, err, common.ErrNoResponse, "StopBotStrategy should report a null reply")
	})

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
