package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetBotStrategyRecommendations(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	result, err := e.GetBotStrategyRecommendations(t.Context(), nil)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetBotStrategyRecommendations(t.Context(), &GetBotStrategyRecommendationsRequest{
		Market:       "BTC_USDT",
		StrategyType: BotStrategySpotGrid,
		Scene:        BotSceneTop1,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSpotGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSpotGridBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CreateSpotGridBot(t.Context(), &SpotGridCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired)

	arg := &SpotGridCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateSpotGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMoneyRequired)

	arg.CreateParams.Money = 1000
	_, err = e.CreateSpotGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotLowPriceRequired)

	arg.CreateParams.LowPrice = 90000
	_, err = e.CreateSpotGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotHighPriceRequired)

	arg.CreateParams.HighPrice = 110000
	_, err = e.CreateSpotGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotGridNumRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.CreateParams.GridNumber = 20
	result, err := e.CreateSpotGridBot(t.Context(), arg)
	require.NoError(t, err)
	assert.NotEmpty(t, result.StrategyID)
}

func TestCreateMarginGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateMarginGridBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CreateMarginGridBot(t.Context(), &MarginGridCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired)

	arg := &MarginGridCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMoneyRequired)

	arg.CreateParams.Money = "1000"
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotLowPriceRequired)

	arg.CreateParams.LowPrice = 90000
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotHighPriceRequired)

	arg.CreateParams.HighPrice = 110000
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotGridNumRequired)

	arg.CreateParams.GridNum = 20
	_, err = e.CreateMarginGridBot(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.CreateParams.Leverage = 3
	arg.CreateParams.Direction = order.Long.Lower()
	result, err := e.CreateMarginGridBot(t.Context(), arg)
	require.NoError(t, err)
	assert.NotEmpty(t, result.StrategyID)
}

func TestCreateInfiniteGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateInfiniteGridBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CreateInfiniteGridBot(t.Context(), &InfiniteGridCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired)

	arg := &InfiniteGridCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateInfiniteGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMoneyRequired)

	arg.CreateParams.Money = "1000"
	_, err = e.CreateInfiniteGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotPriceFloorRequired)

	arg.CreateParams.PriceFloor = "80000"
	_, err = e.CreateInfiniteGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotProfitPerGridRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.CreateParams.ProfitPerGrid = "0.003"
	result, err := e.CreateInfiniteGridBot(t.Context(), arg)
	require.NoError(t, err)
	assert.NotEmpty(t, result.StrategyID)
}

func TestCreateFuturesGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFuturesGridBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CreateFuturesGridBot(t.Context(), &FuturesGridCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired)

	arg := &FuturesGridCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMoneyRequired)

	arg.CreateParams.Money = 1000
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotLowPriceRequired)

	arg.CreateParams.LowPrice = 90000
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotHighPriceRequired)

	arg.CreateParams.HighPrice = 110000
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotGridNumRequired)

	arg.CreateParams.GridNum = 20
	_, err = e.CreateFuturesGridBot(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.CreateParams.Leverage = 5
	arg.CreateParams.Direction = order.Long.Lower()
	result, err := e.CreateFuturesGridBot(t.Context(), arg)
	require.NoError(t, err)
	assert.NotEmpty(t, result.StrategyID)
}

func TestCreateSpotMartingaleBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSpotMartingaleBot(t.Context(), &SpotMartingaleCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired)

	arg := &SpotMartingaleCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateSpotMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotInvestAmountRequired)

	arg.CreateParams.InvestAmount = 1000
	_, err = e.CreateSpotMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotPriceDeviationRequired)

	arg.CreateParams.PriceDeviation = 0.02
	_, err = e.CreateSpotMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMaxOrdersRequired)

	arg.CreateParams.MaxOrders = 5
	_, err = e.CreateSpotMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotTakeProfitRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.CreateParams.TakeProfitRatio = 0.05
	result, err := e.CreateSpotMartingaleBot(t.Context(), arg)
	require.NoError(t, err)
	assert.NotEmpty(t, result.StrategyID)
}

func TestCreateContractMartingaleBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateContractMartingaleBot(t.Context(), &ContractMartingaleCreateRequest{})
	require.ErrorIs(t, err, errBotMarketRequired)

	arg := &ContractMartingaleCreateRequest{Market: "BTC_USDT"}
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotInvestAmountRequired)

	arg.CreateParams.InvestAmount = "1000"
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotPriceDeviationRequired)

	arg.CreateParams.PriceDeviation = "0.02"
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotMaxOrdersRequired)

	arg.CreateParams.MaxOrders = 5
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotTakeProfitRequired)

	arg.CreateParams.TakeProfitRatio = "0.05"
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, errBotDirectionRequired)

	arg.CreateParams.Direction = order.Buy.Lower()
	_, err = e.CreateContractMartingaleBot(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg.CreateParams.Leverage = "5"
	result, err := e.CreateContractMartingaleBot(t.Context(), arg)
	require.NoError(t, err)
	assert.NotEmpty(t, result.StrategyID, "strategy ID should not be empty")
}

func TestGetBotRunningStrategies(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	result, err := e.GetBotRunningStrategies(t.Context(), "", "", 1, 20)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetBotRunningStrategies(t.Context(), BotStrategySpotGrid, "BTC_USDT", 1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBotStrategyDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetBotStrategyDetail(t.Context(), "", BotStrategySpotGrid)
	require.ErrorIs(t, err, errBotStrategyIDRequired)

	_, err = e.GetBotStrategyDetail(t.Context(), "sg_001", "")
	require.ErrorIs(t, err, errBotStrategyTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBotStrategyDetail(t.Context(), "sg_001", BotStrategySpotGrid)
	require.NoError(t, err)
	assert.NotEmpty(t, result.StrategyID)
}

func TestStopBotStrategy(t *testing.T) {
	t.Parallel()
	_, err := e.StopBotStrategy(t.Context(), "", BotStrategySpotGrid)
	require.ErrorIs(t, err, errBotStrategyIDRequired)

	_, err = e.StopBotStrategy(t.Context(), "sg_001", "")
	require.ErrorIs(t, err, errBotStrategyTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.StopBotStrategy(t.Context(), "sg_001", BotStrategySpotGrid)
	require.NoError(t, err)
	assert.NotEmpty(t, result.StrategyID)
}
