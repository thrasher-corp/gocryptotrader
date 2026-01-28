package huobi

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var (
	e                  *Exchange
	btcFutureDatedPair currency.Pair
	btccwPair          = currency.NewPair(currency.BTC, currency.NewCode("CW"))
	btcusdPair         = currency.NewPairWithDelimiter("BTC", "USD", "-")
	btcusdtPair        = currency.NewPairWithDelimiter("BTC", "USDT", "-")
	ethusdPair         = currency.NewPairWithDelimiter("ETH", "USD", "-")
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("HUOBI Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	os.Exit(m.Run())
}

func TestGetCurrenciesIncludingChains(t *testing.T) {
	t.Parallel()
	r, err := e.GetCurrenciesIncludingChains(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.Greater(t, len(r), 1, "should get more than one currency back")
	r, err = e.GetCurrenciesIncludingChains(t.Context(), currency.USDT)
	require.NoError(t, err)
	assert.Equal(t, 1, len(r), "Should only get one currency back")
}

func TestFGetContractInfo(t *testing.T) {
	t.Parallel()
	_, err := e.FGetContractInfo(t.Context(), "", "", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFIndexPriceInfo(t *testing.T) {
	t.Parallel()
	_, err := e.FIndexPriceInfo(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFContractPriceLimitations(t *testing.T) {
	t.Parallel()
	_, err := e.FContractPriceLimitations(t.Context(),
		"BTC", "this_week", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFContractOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.FContractOpenInterest(t.Context(), "BTC", "this_week", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := e.FGetEstimatedDeliveryPrice(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFGetMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := e.FGetMarketDepth(t.Context(), btccwPair, "step5")
	require.NoError(t, err)
}

func TestFGetKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.FGetKlineData(t.Context(), btccwPair, "5min", 5, time.Now().Add(-time.Minute*5), time.Now())
	require.NoError(t, err)
}

func TestFGetMarketOverviewData(t *testing.T) {
	t.Parallel()
	_, err := e.FGetMarketOverviewData(t.Context(), btccwPair)
	require.NoError(t, err)
}

func TestFLastTradeData(t *testing.T) {
	t.Parallel()
	_, err := e.FLastTradeData(t.Context(), btccwPair)
	require.NoError(t, err)
}

func TestFRequestPublicBatchTrades(t *testing.T) {
	t.Parallel()
	_, err := e.FRequestPublicBatchTrades(t.Context(), btccwPair, 50)
	require.NoError(t, err)
}

func TestFQueryTieredAdjustmentFactor(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryTieredAdjustmentFactor(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFQueryHisOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryHisOpenInterest(t.Context(), "BTC", "this_week", "60min", "cont", 3)
	require.NoError(t, err)
}

func TestFQuerySystemStatus(t *testing.T) {
	t.Parallel()
	_, err := e.FQuerySystemStatus(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFQueryTopAccountsRatio(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryTopAccountsRatio(t.Context(), "BTC", "5min")
	require.NoError(t, err)
}

func TestFQueryTopPositionsRatio(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryTopPositionsRatio(t.Context(), "BTC", "5min")
	require.NoError(t, err)
}

func TestFLiquidationOrders(t *testing.T) {
	t.Parallel()
	if _, err := e.FLiquidationOrders(t.Context(), currency.BTC, "filled", 0, 0, "", 0); err != nil {
		t.Error(err)
	}
}

func TestFIndexKline(t *testing.T) {
	t.Parallel()
	_, err := e.FIndexKline(t.Context(), btccwPair, "5min", 5)
	require.NoError(t, err)
}

func TestFGetBasisData(t *testing.T) {
	t.Parallel()
	_, err := e.FGetBasisData(t.Context(), btccwPair, "5min", "open", 3)
	require.NoError(t, err)
}

func TestFGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetAccountInfo(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetPositionsInfo(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetAllSubAccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetAllSubAccountAssets(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetSingleSubAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetSingleSubAccountInfo(t.Context(), "", "154263566")
	require.NoError(t, err)
}

func TestFGetSingleSubPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetSingleSubPositions(t.Context(), "", "154263566")
	require.NoError(t, err)
}

func TestFGetFinancialRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetFinancialRecords(t.Context(),
		"BTC", "closeLong", 2, 0, 0)
	require.NoError(t, err)
}

func TestFGetSettlementRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetSettlementRecords(t.Context(),
		currency.BTC, 0, 0, time.Now().Add(-48*time.Hour), time.Now())
	require.NoError(t, err)
}

func TestFContractTradingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FContractTradingFee(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetTransferLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetTransferLimits(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetPositionLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetPositionLimits(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetAssetsAndPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetAssetsAndPositions(t.Context(), currency.HT)
	require.NoError(t, err)
}

func TestFTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FTransfer(t.Context(), "154263566", "HT", "sub_to_master", 5)
	require.NoError(t, err)
}

func TestFGetTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetTransferRecords(t.Context(), "HT", "master_to_sub", 90, 0, 0)
	require.NoError(t, err)
}

func TestFGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetAvailableLeverage(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "123", "BUY", "open", "limit", 1, 1, 1)
	require.NoError(t, err)
}

func TestFPlaceBatchOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	var req []fBatchOrderData
	order1 := fBatchOrderData{
		Symbol:         "btc",
		ContractType:   "quarter",
		ClientOrderID:  "",
		Price:          5,
		Volume:         1,
		Direction:      "buy",
		Offset:         "open",
		LeverageRate:   1,
		OrderPriceType: "limit",
	}
	order2 := fBatchOrderData{
		Symbol:         "xrp",
		ContractType:   "this_week",
		ClientOrderID:  "",
		Price:          10000,
		Volume:         1,
		Direction:      "sell",
		Offset:         "open",
		LeverageRate:   1,
		OrderPriceType: "limit",
	}
	req = append(req, order1, order2)
	_, err := e.FPlaceBatchOrder(t.Context(), req)
	require.NoError(t, err)
}

func TestFCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FCancelOrder(t.Context(), currency.BTC, "123", "")
	require.NoError(t, err)
}

func TestFCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	updatePairsOnce(t, e)
	_, err := e.FCancelAllOrders(t.Context(), btcFutureDatedPair, "", "")
	require.NoError(t, err)
}

func TestFFlashCloseOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FFlashCloseOrder(t.Context(),
		currency.EMPTYPAIR, "BTC", "quarter", "BUY", "lightning", "", 1)
	require.NoError(t, err)
}

func TestFGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetOrderInfo(t.Context(), "BTC", "", "123")
	require.NoError(t, err)
}

func TestFOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FOrderDetails(t.Context(), "BTC", "123", "quotation", time.Now().Add(-1*time.Hour), 0, 0)
	require.NoError(t, err)
}

func TestFGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetOpenOrders(t.Context(), currency.BTC, 1, 2)
	require.NoError(t, err)
}

func TestFGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FGetOrderHistory(t.Context(),
		currency.EMPTYPAIR, "BTC",
		"all", "all", "limit",
		[]order.Status{},
		5, 0, 0)
	require.NoError(t, err)
}

func TestFTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.FTradeHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", 10, 0, 0)
	require.NoError(t, err)
}

func TestFPlaceTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FPlaceTriggerOrder(t.Context(), currency.EMPTYPAIR, "EOS", "quarter", "greaterOrEqual", "limit", "buy", "close", 1.1, 1.05, 5, 2)
	require.NoError(t, err)
}

func TestFCancelTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FCancelTriggerOrder(t.Context(), "ETH", "123")
	require.NoError(t, err)
}

func TestFCancelAllTriggerOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FCancelAllTriggerOrders(t.Context(), currency.EMPTYPAIR, "BTC", "this_week")
	require.NoError(t, err)
}

func TestFQueryTriggerOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FQueryTriggerOpenOrders(t.Context(), currency.EMPTYPAIR, "BTC", 0, 0)
	require.NoError(t, err)
}

func TestFQueryTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.FQueryTriggerOrderHistory(t.Context(), currency.EMPTYPAIR, "EOS", "all", "all", 10, 0, 0)
	require.NoError(t, err)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)
}

func TestUpdateTickerSpot(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("INV", "ALID", "-"), asset.Spot)
	assert.ErrorContains(t, err, "invalid symbol")
	_, err = e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Spot)
	require.NoError(t, err)
}

func TestUpdateTickerCMF(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("INV", "ALID", "_"), asset.CoinMarginedFutures)
	assert.ErrorContains(t, err, "symbol data error")
	_, err = e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("BTC", "USD", "_"), asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateTickerFutures(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), btccwPair, asset.Futures)
	require.NoError(t, err)
}

func TestUpdateOrderbookSpot(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btcusdtPair, asset.Spot)
	require.NoError(t, err)
}

func TestUpdateOrderbookCMF(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateOrderbookFuture(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), btccwPair, asset.Futures)
	require.NoError(t, err)
	_, err = e.UpdateOrderbook(t.Context(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	updatePairsOnce(t, e)
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewBTCUSDT()},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)

	getOrdersRequest.Pairs = []currency.Pair{btcusdPair}
	getOrdersRequest.AssetType = asset.CoinMarginedFutures
	_, err = e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)
	getOrdersRequest.Pairs = []currency.Pair{btcFutureDatedPair}
	getOrdersRequest.AssetType = asset.Futures
	_, err = e.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures})
	require.NoError(t, err)
}

func TestQuerySwapIndexPriceInfo(t *testing.T) {
	t.Parallel()
	_, err := e.QuerySwapIndexPriceInfo(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestSwapOpenInterestInformation(t *testing.T) {
	t.Parallel()
	_, err := e.SwapOpenInterestInformation(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetSwapMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarketDepth(t.Context(), btcusdPair, "step0")
	require.NoError(t, err)
}

func TestGetSwapKlineData(t *testing.T) {
	t.Parallel()
	r, err := e.GetSwapKlineData(t.Context(), btcusdPair, "5min", 5, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotEmpty(t, r.Data, "GetSwapKlineData should return some data")
}

func TestGetSwapMarketOverview(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarketOverview(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetLastTrade(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastTrade(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetBatchTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetBatchTrades(t.Context(), btcusdPair, 5)
	require.NoError(t, err)
}

func TestGetTieredAjustmentFactorInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetTieredAjustmentFactorInfo(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetOpenInterestInfo(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	_, err := e.GetOpenInterestInfo(t.Context(), btcusdPair, "5min", "cryptocurrency", 50)
	require.NoError(t, err)
}

func TestGetTraderSentimentIndexAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetTraderSentimentIndexAccount(t.Context(), btcusdPair, "5min")
	require.NoError(t, err)
}

func TestGetTraderSentimentIndexPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetTraderSentimentIndexPosition(t.Context(), btcusdPair, "5min")
	require.NoError(t, err)
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidationOrders(t.Context(), btcusdPair, "closed", time.Now().AddDate(0, 0, -2), time.Now(), "", 0)
	assert.NoError(t, err, "GetLiquidationOrders should not error")
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalFundingRatesForPair(t.Context(), btcusdPair, 0, 0)
	require.NoError(t, err)
}

func TestGetPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "5min", 15)
	require.NoError(t, err)
}

func TestGetEstimatedFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "5min", 15)
	require.NoError(t, err)
}

func TestGetBasisData(t *testing.T) {
	t.Parallel()
	_, err := e.GetBasisData(t.Context(), btcusdPair, "5min", "close", 5)
	require.NoError(t, err)
}

func TestGetSystemStatusInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetSystemStatusInfo(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetSwapPriceLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapPriceLimits(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetMarginRates(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetMarginRates(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetSwapAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapAccountInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapPositionsInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapAssetsAndPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapAssetsAndPositions(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapAllSubAccAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapAllSubAccAssets(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSubAccPositionInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubAccPositionInfo(t.Context(), ethusdPair, 0)
	require.NoError(t, err)
}

func TestGetAccountFinancialRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAccountFinancialRecords(t.Context(), ethusdPair, "3,4", 15, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapSettlementRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	r, err := e.GetSwapSettlementRecords(t.Context(), ethusdPair, time.Now().AddDate(0, -1, 0), time.Now(), 0, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, r.Data, "GetSwapSettlementRecords should return some data")
}

func TestGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAvailableLeverage(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapOrderLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderLimitInfo(t.Context(), ethusdPair, "limit")
	require.NoError(t, err)
}

func TestGetSwapTradingFeeInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTradingFeeInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapTransferLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTransferLimitInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapPositionLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapPositionLimitInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestAccountTransferData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.AccountTransferData(t.Context(), ethusdPair, "123", "master_to_sub", 15)
	require.NoError(t, err)
}

func TestAccountTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.AccountTransferRecords(t.Context(), ethusdPair, "master_to_sub", 12, 0, 0)
	require.NoError(t, err)
}

func TestPlaceSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceSwapOrders(t.Context(), ethusdPair, "", "buy", "open", "limit", 0.01, 1, 1)
	require.NoError(t, err)
}

func TestPlaceSwapBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	var req BatchOrderRequestType
	order1 := batchOrderData{
		ContractCode:   "ETH-USD",
		ClientOrderID:  "",
		Price:          5,
		Volume:         1,
		Direction:      "buy",
		Offset:         "open",
		LeverageRate:   1,
		OrderPriceType: "limit",
	}
	order2 := batchOrderData{
		ContractCode:   "BTC-USD",
		ClientOrderID:  "",
		Price:          2.5,
		Volume:         1,
		Direction:      "buy",
		Offset:         "open",
		LeverageRate:   1,
		OrderPriceType: "limit",
	}
	req.Data = append(req.Data, order1, order2)

	_, err := e.PlaceSwapBatchOrders(t.Context(), req)
	require.NoError(t, err)
}

func TestCancelSwapOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelSwapOrder(t.Context(), "test123", "", ethusdPair)
	require.NoError(t, err)
}

func TestCancelAllSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelAllSwapOrders(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestPlaceLightningCloseOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceLightningCloseOrder(t.Context(), ethusdPair, "buy", "lightning", 5, 1)
	require.NoError(t, err)
}

func TestGetSwapOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderInfo(t.Context(), ethusdPair, "123", "")
	require.NoError(t, err)
}

func TestGetSwapOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderDetails(t.Context(), ethusdPair, "123", "10", "cancelledOrder", 0, 0)
	require.NoError(t, err)
}

func TestGetSwapOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOpenOrders(t.Context(), ethusdPair, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderHistory(t.Context(), ethusdPair, "all", "all", []order.Status{order.PartiallyCancelled, order.Active}, 25, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTradeHistory(t.Context(), ethusdPair, "liquidateShort", 10, 0, 0)
	require.NoError(t, err)
}

func TestPlaceSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceSwapTriggerOrder(t.Context(), ethusdPair, "greaterOrEqual", "buy", "open", "optimal_5", 5, 3, 1, 1)
	require.NoError(t, err)
}

func TestCancelSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelSwapTriggerOrder(t.Context(), ethusdPair, "test123")
	require.NoError(t, err)
}

func TestCancelAllSwapTriggerOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelAllSwapTriggerOrders(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTriggerOrderHistory(t.Context(), ethusdPair, "open", "all", 15, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapMarkets(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarkets(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotKline(t.Context(), KlinesRequestParams{Symbol: btcusdtPair, Period: "1min"})
	require.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	updatePairsOnce(t, e)

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err := e.GetHistoricCandles(t.Context(), btcusdtPair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcusdtPair, asset.Spot, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	updatePairsOnce(t, e)

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err := e.GetHistoricCandlesExtended(t.Context(), btcusdtPair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	// demonstrate that adjusting time doesn't wreck non-day intervals
	_, err = e.GetHistoricCandlesExtended(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneHour, endTime.AddDate(0, 0, -1), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), time.Now())
	require.NoError(t, err)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneHour, endTime.AddDate(0, 0, -1), time.Now())
	require.NoError(t, err)
}

func TestGetMarketDetailMerged(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketDetailMerged(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepth(t.Context(),
		&OrderBookDataRequestParams{
			Symbol: btcusdtPair,
			Type:   OrderBookDataRequestParamsTypeStep1,
		})
	require.NoError(t, err)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestSpotPrice(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeHistory(t.Context(), btcusdtPair, 50)
	require.NoError(t, err)
}

func TestGetMarketDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketDetail(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbols(t.Context())
	require.NoError(t, err)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencies(t.Context())
	require.NoError(t, err)
}

func TestGet24HrMarketSummary(t *testing.T) {
	t.Parallel()
	_, err := e.Get24HrMarketSummary(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickers(t.Context())
	require.NoError(t, err)
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	st, err := e.GetCurrentServerTime(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, st, "GetCurrentServerTime should return a time")
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, st, "GetServerTime should return a time")
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.GetAccounts(t.Context())
	require.NoError(t, err)
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetAccounts(t.Context())
	require.NoError(t, err, "GetAccounts must not error")

	userID := strconv.FormatInt(result[0].ID, 10)
	_, err = e.GetAccountBalance(t.Context(), userID)
	require.NoError(t, err, "GetAccountBalance must not error")
}

func TestGetAggregatedBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAggregatedBalance(t.Context())
	require.NoError(t, err)
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg := SpotNewOrderRequestParams{
		Symbol:    btcusdtPair,
		AccountID: 1997024,
		Amount:    0.01,
		Price:     10.1,
		Type:      SpotNewOrderRequestTypeBuyLimit,
	}

	_, err := e.SpotNewOrder(t.Context(), &arg)
	require.NoError(t, err)
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelExistingOrder(t.Context(), 1337)
	assert.Error(t, err)
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.GetOrder(t.Context(), 1337)
	require.NoError(t, err)
}

func TestGetMarginLoanOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetMarginLoanOrders(t.Context(), btcusdtPair, "", "", "", "", "", "", "")
	require.NoError(t, err)
}

func TestGetMarginAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetMarginAccountBalance(t.Context(), btcusdtPair)
	require.NoError(t, err)
}

func TestCancelWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelWithdraw(t.Context(), 1337)
	require.Error(t, err)
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.LTC.String(),
			"_"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := setFeeBuilder()
	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err)
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	_, err := e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	_, err = e.GetFee(feeBuilder)
	require.NoError(t, err)
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := e.FormatWithdrawPermissions()
	assert.Equal(t, expectedResult, withdrawPermissions)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewBTCUSDT()},
		Side:      order.AnySide,
	}

	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

// TestSubmitOrder and below can impact your orders on the exchange. Enable canManipulateRealOrders to run them
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	accounts, err := e.GetAccounts(t.Context())
	require.NoError(t, err, "GetAccounts must not error")

	orderSubmission := &order.Submit{
		Exchange: e.Name,
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.USDT,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     5,
		Amount:    1,
		ClientID:  strconv.FormatInt(accounts[0].ID, 10),
		AssetType: asset.Spot,
	}
	response, err := e.SubmitOrder(t.Context(), orderSubmission)
	require.NoError(t, err)
	assert.Equal(t, order.New, response.Status, "response status should be correct")
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      btcusdtPair,
		AssetType: asset.Spot,
	}

	err := e.CancelOrder(t.Context(), orderCancellation)
	require.NoError(t, err)
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	_, err := e.CancelAllOrders(t.Context(), &orderCancellation)
	require.NoError(t, err)
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	for _, a := range []asset.Item{asset.Spot, asset.CoinMarginedFutures, asset.Futures} {
		_, err := e.UpdateAccountBalances(t.Context(), a)
		assert.NoErrorf(t, err, "UpdateAccountBalances should not error for asset %s", a)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	_, err := e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot})
	require.Error(t, err, "ModifyOrder must error without any order details")
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdrawCryptoRequest)
	require.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero)
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestQueryDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.QueryDepositAddress(t.Context(), currency.USDT)
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.USDT, "", "uSdTeRc20")
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestQueryWithdrawQuota(t *testing.T) {
	t.Parallel()
	_, err := e.QueryWithdrawQuotas(t.Context(), currency.BTC.Lower().String())
	if sharedtestvalues.AreAPICredentialsSet(e) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestWSCandles(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.kline.1min", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.CandlesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsCandles.json", e.wsHandleData)
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	cAny := <-e.Websocket.DataHandler.C
	c, ok := cAny.Data.(websocket.KlineData)
	require.True(t, ok, "Must get the correct type from DataHandler")
	exp := websocket.KlineData{
		Timestamp:  time.UnixMilli(1489474082831),
		Pair:       btcusdtPair,
		AssetType:  asset.Spot,
		Exchange:   e.Name,
		OpenPrice:  7962.62,
		ClosePrice: 8014.56,
		HighPrice:  14962.77,
		LowPrice:   5110.14,
		Volume:     4.4,
		Interval:   "0s",
	}
	assert.Equal(t, exp, c)
}

func TestWSOrderbook(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.depth.step0", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.OrderbookChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsOrderbook.json", e.wsHandleData)
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	dAny := <-e.Websocket.DataHandler.C
	d, ok := dAny.Data.(*orderbook.Depth)
	require.True(t, ok, "Must get the correct type from DataHandler")
	require.NotNil(t, d)
	l, err := d.GetAskLength()
	require.NoError(t, err, "GetAskLength must not error")
	assert.Equal(t, 2, l, "Ask length should be correct")
	liq, _, err := d.TotalAskAmounts()
	require.NoError(t, err, "TotalAskAmount must not error")
	assert.Equal(t, 0.502591, liq, "Ask Liquidity should be correct")
	l, err = d.GetBidLength()
	require.NoError(t, err, "GetBidLength must not error")
	assert.Equal(t, 2, l, "Bid length should be correct")
	liq, _, err = d.TotalBidAmounts()
	require.NoError(t, err, "TotalBidAmount must not error")
	assert.Equal(t, 0.56281, liq, "Bid Liquidity should be correct")
}

// TestWSHandleAllTradesMsg ensures wsHandleAllTrades sends trade.Data to the ws.DataHandler
func TestWSHandleAllTradesMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.trade.detail", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.AllTradesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.json", e.wsHandleData)
	e.Websocket.DataHandler.Close()
	exp := []trade.Data{
		{
			Exchange:     e.Name,
			CurrencyPair: btcusdtPair,
			Timestamp:    time.UnixMilli(1630994963173).UTC(),
			Price:        52648.62,
			Amount:       0.006754,
			Side:         order.Buy,
			TID:          "102523573486",
			AssetType:    asset.Spot,
		},
		{
			Exchange:     e.Name,
			CurrencyPair: btcusdtPair,
			Timestamp:    time.UnixMilli(1630994963184).UTC(),
			Price:        52648.73,
			Amount:       0.006755,
			Side:         order.Sell,
			TID:          "102523573487",
			AssetType:    asset.Spot,
		},
	}
	require.Len(t, e.Websocket.DataHandler.C, 2, "Must see correct number of trades")
	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case trade.Data:
			i := 1 - len(e.Websocket.DataHandler.C)
			require.Equalf(t, exp[i], v, "Trade [%d] must be correct", i)
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T(%s)", v, v)
		}
	}
	require.Empty(t, e.Websocket.DataHandler.C, "Must not see any errors going to datahandler")
}

func TestWSTicker(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.detail", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.TickerChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsTicker.json", e.wsHandleData)
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	tickAny := <-e.Websocket.DataHandler.C
	tick, ok := tickAny.Data.(*ticker.Price)
	require.True(t, ok, "Must get the correct type from DataHandler")
	require.NotNil(t, tick)
	exp := &ticker.Price{
		High:         52924.14,
		Low:          51000,
		Bid:          0,
		Volume:       13991.028076056185,
		QuoteVolume:  7.27676440200527e+08,
		Open:         51823.62,
		Close:        52379.99,
		Pair:         btcusdtPair,
		ExchangeName: e.Name,
		AssetType:    asset.Spot,
		LastUpdated:  time.UnixMilli(1630998026649),
	}
	assert.Equal(t, exp, tick)
}

func TestWSAccountUpdate(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "accounts.update#2", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.MyAccountChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	testexch.FixtureToDataHandler(t, "testdata/wsMyAccount.json", e.wsHandleData)
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 3, "Must see correct number of records")
	exp := []WsAccountUpdate{
		{Currency: "btc", AccountID: 123456, Balance: 23.111, ChangeType: "transfer", AccountType: "trade", ChangeTime: types.Time(time.UnixMilli(1568601800000)), SeqNum: 1},
		{Currency: "btc", AccountID: 33385, Available: 2028.69, ChangeType: "order.match", AccountType: "trade", ChangeTime: types.Time(time.UnixMilli(1574393385167)), SeqNum: 2},
		{Currency: "usdt", AccountID: 14884859, Available: 20.29388158, Balance: 20.29388158, AccountType: "trade", SeqNum: 3},
	}
	for _, ex := range exp {
		uAny := <-e.Websocket.DataHandler.C
		u, ok := uAny.Data.(WsAccountUpdate)
		require.True(t, ok, "Must get the correct type from DataHandler")
		require.NotNil(t, u)
		assert.Equal(t, ex, u)
	}
}

func TestWSOrderUpdate(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "orders#*", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.MyOrdersChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	errs := testexch.FixtureToDataHandlerWithErrors(t, "testdata/wsMyOrders.json", e.wsHandleData)
	e.Websocket.DataHandler.Close()
	require.Equal(t, 1, len(errs), "Must receive the correct number of errors back")
	require.ErrorContains(t, errs[0].Err, "error with order \"test1\": invalid.client.order.id (NT) (2002)")
	require.Len(t, e.Websocket.DataHandler.C, 4, "Must see correct number of records")
	exp := []*order.Detail{
		{
			Exchange:      e.Name,
			Pair:          btcusdtPair,
			Side:          order.Buy,
			Status:        order.Rejected,
			ClientOrderID: "test1",
			AssetType:     asset.Spot,
			LastUpdated:   time.UnixMicro(1583853365586000),
		},
		{
			Exchange:      e.Name,
			Pair:          btcusdtPair,
			Side:          order.Buy,
			Status:        order.Cancelled,
			ClientOrderID: "test2",
			AssetType:     asset.Spot,
			LastUpdated:   time.UnixMicro(1583853365586000),
		},
		{
			Exchange:      e.Name,
			Pair:          btcusdtPair,
			Side:          order.Sell,
			Status:        order.New,
			ClientOrderID: "test3",
			AssetType:     asset.Spot,
			Price:         77,
			Amount:        2,
			Type:          order.Limit,
			OrderID:       "27163533",
			LastUpdated:   time.UnixMicro(1583853365586000),
		},
		{
			Exchange:    e.Name,
			Pair:        btcusdtPair,
			Side:        order.Buy,
			Status:      order.New,
			AssetType:   asset.Spot,
			Price:       70000,
			Amount:      0.000157,
			Type:        order.Limit,
			OrderID:     "1199329381585359",
			LastUpdated: time.UnixMicro(1731039387696000),
		},
	}
	for _, ex := range exp {
		m := <-e.Websocket.DataHandler.C
		require.IsType(t, &order.Detail{}, m.Data, "Must get the correct type from DataHandler")
		d, _ := m.Data.(*order.Detail)
		require.NotNil(t, d)
		assert.Equal(t, ex, d, "Order Detail should match")
	}
}

func TestWSMyTrades(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "trade.clearing#btcusdt#1", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.MyTradesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	testexch.FixtureToDataHandler(t, "testdata/wsMyTrades.json", e.wsHandleData)
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	m := <-e.Websocket.DataHandler.C
	exp := &order.Detail{
		Exchange:      e.Name,
		Pair:          btcusdtPair,
		Side:          order.Buy,
		Status:        order.PartiallyFilled,
		ClientOrderID: "a001",
		OrderID:       "99998888",
		AssetType:     asset.Spot,
		Date:          time.UnixMicro(1583853365586000),
		LastUpdated:   time.UnixMicro(1583853365996000),
		Price:         10000,
		Amount:        1,
		Trades: []order.TradeHistory{
			{
				Price:     9999.99,
				Amount:    0.96,
				Fee:       19.88,
				Exchange:  e.Name,
				TID:       "919219323232",
				Side:      order.Buy,
				IsMaker:   false,
				Timestamp: time.UnixMicro(1583853365996000),
			},
		},
	}
	require.IsType(t, &order.Detail{}, m.Data, "Must get the correct type from DataHandler")
	d, _ := m.Data.(*order.Detail)
	require.NotNil(t, d)
	assert.Equal(t, exp, d, "Order Detail should match")
}

func TestStringToOrderStatus(t *testing.T) {
	t.Parallel()
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []TestCases{
		{Case: "submitted", Result: order.New},
		{Case: "canceled", Result: order.Cancelled},
		{Case: "partial-filled", Result: order.PartiallyFilled},
		{Case: "partial-canceled", Result: order.PartiallyCancelled},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := stringToOrderStatus(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestStringToOrderSide(t *testing.T) {
	t.Parallel()
	type TestCases struct {
		Case   string
		Result order.Side
	}
	testCases := []TestCases{
		{Case: "buy-limit", Result: order.Buy},
		{Case: "sell-limit", Result: order.Sell},
		{Case: "woah-nelly", Result: order.UnknownSide},
	}
	for i := range testCases {
		result, _ := stringToOrderSide(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestStringToOrderType(t *testing.T) {
	t.Parallel()
	type TestCases struct {
		Case   string
		Result order.Type
	}
	testCases := []TestCases{
		{Case: "buy-limit", Result: order.Limit},
		{Case: "sell-market", Result: order.Market},
		{Case: "woah-nelly", Result: order.UnknownType},
	}
	for i := range testCases {
		result, _ := stringToOrderType(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		interval kline.Interval
		output   string
	}{
		{kline.OneMin, "1min"},
		{kline.FourHour, "4hour"},
		{kline.OneDay, "1day"},
		{kline.OneWeek, "1week"},
		{kline.OneMonth, "1mon"},
		{kline.OneYear, "1year"},
		{kline.TwoWeek, ""},
	} {
		assert.Equalf(t, tt.output, e.FormatExchangeKlineInterval(tt.interval), "FormatExchangeKlineInterval should return correctly for %s", tt.output)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), btcusdtPair, asset.Spot)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btccwPair, asset.Futures)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), btcusdtPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	c, err := e.GetAvailableTransferChains(t.Context(), currency.USDT)
	require.NoError(t, err)
	require.Greater(t, len(c), 2, "Must get more than 2 chains")
}

func TestFormatFuturesPair(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)

	r, err := e.formatFuturesPair(btccwPair, false)
	require.NoError(t, err)
	assert.Equal(t, "BTC_CW", r)

	// pair in the format of BTC210827 but make it lower case to test correct formatting
	r, err = e.formatFuturesPair(btcFutureDatedPair.Lower(), false)
	require.NoError(t, err)
	assert.Len(t, r, 9, "Should be an 9 character string")
	assert.Equal(t, "BTC2", r[0:4], "Should start with btc and a date this millennium")

	r, err = e.formatFuturesPair(btccwPair, true)
	require.NoError(t, err)
	assert.Len(t, r, 9, "Should be an 9 character string")
	assert.Equal(t, "BTC2", r[0:4], "Should start with btc and a date this millennium")

	r, err = e.formatFuturesPair(currency.NewBTCUSDT(), false)
	require.NoError(t, err)
	assert.Equal(t, "BTC-USDT", r)
}

func TestSearchForExistedWithdrawsAndDeposits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.SearchForExistedWithdrawsAndDeposits(t.Context(), currency.BTC, "deposit", "", 0, 100)
	require.NoError(t, err)
}

func TestCancelOrderBatch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelOrderBatch(t.Context(), []string{"1234"}, nil)
	require.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      currency.NewBTCUSDT(),
		},
	})
	require.NoError(t, err)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")
	updatePairsOnce(t, e)

	_, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSD(),
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.CoinMarginedFutures,
		Pair:                 currency.NewBTCUSD(),
		IncludePredictedRate: true,
	})
	require.NoError(t, err)

	err = e.CurrencyPairs.EnablePair(asset.CoinMarginedFutures, currency.NewBTCUSD())
	require.ErrorIs(t, err, currency.ErrPairAlreadyEnabled)

	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.CoinMarginedFutures,
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Binary, currency.NewBTCUSDT())
	require.NoError(t, err)
	assert.False(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewBTCUSDT())
	require.NoError(t, err)
	assert.True(t, is)
}

func TestGetSwapFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapFundingRates(t.Context())
	require.NoError(t, err)
}

func TestGetBatchCoinMarginSwapContracts(t *testing.T) {
	t.Parallel()
	resp, err := e.GetBatchCoinMarginSwapContracts(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetBatchLinearSwapContracts(t *testing.T) {
	t.Parallel()
	resp, err := e.GetBatchLinearSwapContracts(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetBatchFuturesContracts(t *testing.T) {
	t.Parallel()
	resp, err := e.GetBatchFuturesContracts(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		err := e.UpdateTickers(t.Context(), a)
		require.NoErrorf(t, err, "asset %s", a)
		avail, err := e.GetAvailablePairs(a)
		require.NoError(t, err)
		for _, p := range avail {
			_, err = ticker.GetTicker(e.Name, p, a)
			assert.NoErrorf(t, err, "Could not get ticker for %s %s", a, p)
		}
	}
}

var expiryWindows = map[string]uint{
	"CW": 14,
	"NW": 21,
	"CQ": 190,
	"NQ": 282,
}

// TestPairFromContractExpiryCode ensures at least some contract codes are available and loaded with sane dates
// Expectations are relaxed because dates are unpredictable and codes disappear intermittently
func TestPairFromContractExpiryCode(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")

	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)

	tz, err := time.LoadLocation("Asia/Singapore") // Huobi HQ and apparent local time for when codes become effective
	require.NoError(t, err, "LoadLocation must not error")

	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, tz) // Do not use Truncate; https://github.com/golang/go/issues/55921

	require.NotEmpty(t, e.futureContractCodes, "At least one contract code must be loaded")

	for cType, cachedContract := range e.futureContractCodes {
		t.Run(cType, func(t *testing.T) {
			t.Parallel()
			p, err := e.pairFromContractExpiryCode(currency.Pair{
				Base:  currency.BTC,
				Quote: currency.NewCode(cType),
			})
			require.NoError(t, err)
			assert.Equal(t, currency.BTC, p.Base, "pair Base should be BTC")
			assert.Equal(t, cachedContract, p.Quote, "pair Quote should match futureContractCodes value")
			exp, err := time.ParseInLocation("060102", p.Quote.String(), tz)
			require.NoError(t, err, "currency code must be a parsable date")
			require.Falsef(t, exp.Before(today), "expiry must be today or after; Got: %q", exp)
			diff := uint(exp.Sub(today).Hours() / 24)
			require.LessOrEqualf(t, diff, expiryWindows[cType], "expiry must be within expected update window; Today: %q, Expiry: %q",
				today.Format(time.DateOnly),
				exp.Format(time.DateOnly),
			)
		})
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)

	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	resp, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USD.Item,
		Asset: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  btccwPair.Base.Item,
		Quote: btccwPair.Quote.Item,
		Asset: asset.Futures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestContractOpenInterestUSDT(t *testing.T) {
	t.Parallel()
	resp, err := e.ContractOpenInterestUSDT(t.Context(), currency.EMPTYPAIR, currency.EMPTYPAIR, "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	cp := currency.NewBTCUSDT()
	resp, err = e.ContractOpenInterestUSDT(t.Context(), cp, currency.EMPTYPAIR, "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.ContractOpenInterestUSDT(t.Context(), currency.EMPTYPAIR, cp, "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.ContractOpenInterestUSDT(t.Context(), cp, currency.EMPTYPAIR, "this_week", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.ContractOpenInterestUSDT(t.Context(), currency.EMPTYPAIR, currency.EMPTYPAIR, "", "swap")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		if (a == asset.Futures || a == asset.CoinMarginedFutures) && !pairs[0].Quote.Equal(currency.USD) && !pairs[0].Quote.Equal(currency.USDT) {
			require.ErrorIs(t, err, common.ErrNotYetImplemented)
		} else {
			require.NoError(t, err)
			assert.NotEmpty(t, resp)
		}
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := e.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{}
	for _, s := range e.Features.Subscriptions {
		if s.Asset == asset.Empty {
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.QualifiedChannel = channelName(s)
			exp = append(exp, s)
			continue
		}
		for _, a := range e.GetAssetTypes(true) {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			pairs = common.SortStrings(pairs).Format(currency.PairFormat{Uppercase: false, Delimiter: ""})
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.Asset = a
			if isWildcardChannel(s) {
				s.Pairs = pairs
				s.QualifiedChannel = channelName(s)
				exp = append(exp, s)
				continue
			}
			for i, p := range pairs {
				s := s.Clone() //nolint:govet // Intentional lexical scope shadow
				s.QualifiedChannel = channelName(s, p)
				switch s.Channel {
				case subscription.OrderbookChannel:
					s.QualifiedChannel += ".step0"
				case subscription.CandlesChannel:
					s.QualifiedChannel += ".1min"
				}
				s.Pairs = pairs[i : i+1]
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

func wsFixture(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	action, _ := jsonparser.GetString(msg, "action")
	ch, _ := jsonparser.GetString(msg, "ch")
	if action == "req" && ch == "auth" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"action":"req","code":200,"ch":"auth","data":{}}`))
	}
	if action == "sub" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"action":"sub","code":200,"ch":"`+ch+`"}`))
	}
	id, _ := jsonparser.GetString(msg, "id")
	sub, _ := jsonparser.GetString(msg, "sub")
	if id != "" && sub != "" {
		return w.WriteMessage(gws.TextMessage, []byte(`{"id":"`+id+`","status":"ok","subbed":"`+sub+`"}`))
	}
	return fmt.Errorf("%w: %s", errors.New("Unhandled mock websocket message"), msg)
}

// TestSubscribe exercises live public subscriptions
func TestSubscribe(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	subs, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err, "ExpandTemplates must not error")
	testexch.SetupWs(t, e)
	err = e.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")
	got := e.Websocket.GetSubscriptions()
	require.Equal(t, 8, len(got), "Must get correct number of subscriptions")
	for _, s := range got {
		assert.Equal(t, subscription.SubscribedState, s.State())
	}
}

// TestAuthSubscribe exercises mock subscriptions including private
func TestAuthSubscribe(t *testing.T) {
	t.Parallel()
	subCfg := e.Features.Subscriptions
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
	h.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := subCfg.ExpandTemplates(h)
	require.NoError(t, err, "ExpandTemplates must not error")
	err = h.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")
	got := h.Websocket.GetSubscriptions()
	require.Equal(t, 11, len(got), "Must get correct number of subscriptions")
	for _, s := range got {
		assert.Equal(t, subscription.SubscribedState, s.State())
	}
}

func TestChannelName(t *testing.T) {
	assert.Equal(t, "market.BTC-USD.kline", channelName(&subscription.Subscription{Channel: subscription.CandlesChannel}, btcusdPair))
	assert.Equal(t, "trade.clearing#*#1", channelName(&subscription.Subscription{Channel: subscription.MyTradesChannel}, btcusdPair))
	assert.Panics(t, func() { channelName(&subscription.Subscription{Channel: wsOrderbookChannel}, btcusdPair) })
}

func TestIsWildcardChannel(t *testing.T) {
	assert.False(t, isWildcardChannel(&subscription.Subscription{Channel: subscription.CandlesChannel}))
	assert.True(t, isWildcardChannel(&subscription.Subscription{Channel: subscription.MyOrdersChannel}))
	assert.Panics(t, func() { channelName(&subscription.Subscription{Channel: wsOrderbookChannel}) })
}

func TestGetErrResp(t *testing.T) {
	err := getErrResp([]byte(`{"status":"error","err-code":"bad-request","err-msg":"invalid topic promiscuous.drop🐻s.nearby"}`))
	assert.ErrorContains(t, err, "invalid topic promiscuous.drop🐻s.nearby (bad-request)", "V1 errors should return correctly")
	err = getErrResp([]byte(`{"status":"ok","subbed":"market.btcusdt.trade.detail"}`))
	assert.NoError(t, err, "V1 success should not error")

	err = getErrResp([]byte(`{"action":"sub","code":2001,"ch":"naughty.drop🐻s.locally","message":"invalid.ch"}`))
	assert.ErrorContains(t, err, "invalid.ch (2001)", "V2 errors should return correctly")

	err = getErrResp([]byte(`{"action":"sub","code":200,"ch":"orders#btcusdt","data":{}}`))
	assert.NoError(t, err, "V2 success should not error")
}

func TestBootstrap(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")

	c, err := e.Bootstrap(t.Context())
	require.NoError(t, err)
	assert.True(t, c, "Bootstrap should return true to continue")

	e.futureContractCodes = nil
	e.Features.Enabled.AutoPairUpdates = false
	_, err = e.Bootstrap(t.Context())
	require.NoError(t, err)
	require.NotNil(t, e.futureContractCodes)
}

var (
	updatePairsMutex         sync.Mutex
	futureContractCodesCache map[string]currency.Code
)

// updatePairsOnce updates the pairs once, and ensures a future dated contract is enabled
func updatePairsOnce(tb testing.TB, h *Exchange) {
	tb.Helper()

	updatePairsMutex.Lock()
	defer updatePairsMutex.Unlock()

	testexch.UpdatePairsOnce(tb, h)

	h.futureContractCodesMutex.Lock()
	if len(h.futureContractCodes) == 0 {
		// Restored pairs from cache, so haven't populated futureContract Codes
		require.NotEmpty(tb, futureContractCodesCache, "futureContractCodesCache must not be empty")
		h.futureContractCodes = futureContractCodesCache
	} else {
		futureContractCodesCache = h.futureContractCodes
	}
	h.futureContractCodesMutex.Unlock()

	if btcFutureDatedPair.Equal(currency.EMPTYPAIR) {
		p, err := h.pairFromContractExpiryCode(btccwPair)
		require.NoError(tb, err, "pairFromContractCode must not error")
		btcFutureDatedPair = p
	}

	err := h.CurrencyPairs.EnablePair(asset.Futures, btcFutureDatedPair) // Must enable every time we refresh the CurrencyPairs from cache
	require.NoError(tb, common.ExcludeError(err, currency.ErrPairAlreadyEnabled))
}
