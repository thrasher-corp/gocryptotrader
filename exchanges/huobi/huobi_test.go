package huobi

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var (
	h                  = &HUOBI{}
	wsSetupRan         bool
	btcFutureDatedPair currency.Pair
	btccwPair          = currency.NewPair(currency.BTC, currency.NewCode("CW"))
	btcusdPair         = currency.NewPairWithDelimiter("BTC", "USD", "-")
	btcusdtPair        = currency.NewPairWithDelimiter("BTC", "USDT", "-")
	ethusdPair         = currency.NewPairWithDelimiter("ETH", "USD", "-")
)

func TestMain(m *testing.M) {
	h.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Huobi load config error", err)
	}
	hConfig, err := cfg.GetExchangeConfig("Huobi")
	if err != nil {
		log.Fatal("Huobi Setup() init error")
	}
	hConfig.API.AuthenticatedSupport = true
	hConfig.API.AuthenticatedWebsocketSupport = true
	hConfig.API.Credentials.Key = apiKey
	hConfig.API.Credentials.Secret = apiSecret
	h.Websocket = sharedtestvalues.NewTestWebsocket()
	err = h.Setup(hConfig)
	if err != nil {
		log.Fatal("Huobi setup error", err)
	}

	os.Exit(m.Run())
}

func setupWsTests(t *testing.T) {
	t.Helper()
	if wsSetupRan {
		return
	}
	if !h.Websocket.IsEnabled() && !h.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(h) {
		t.Skip(stream.ErrWebsocketNotEnabled.Error())
	}
	comms = make(chan WsMessage, sharedtestvalues.WebsocketChannelOverrideCapacity)
	go h.wsReadData()
	var dialer websocket.Dialer
	err := h.wsAuthenticatedDial(&dialer)
	if err != nil {
		t.Fatal(err)
	}
	err = h.wsLogin(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	wsSetupRan = true
}

func TestGetCurrenciesIncludingChains(t *testing.T) {
	t.Parallel()
	r, err := h.GetCurrenciesIncludingChains(context.Background(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.Greater(t, len(r), 1, "should get more than one currency back")
	r, err = h.GetCurrenciesIncludingChains(context.Background(), currency.USDT)
	require.NoError(t, err)
	assert.Equal(t, 1, len(r), "Should only get one currency back")
}

func TestFGetContractInfo(t *testing.T) {
	t.Parallel()
	_, err := h.FGetContractInfo(context.Background(), "", "", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFIndexPriceInfo(t *testing.T) {
	t.Parallel()
	_, err := h.FIndexPriceInfo(context.Background(), currency.BTC)
	require.NoError(t, err)
}

func TestFContractPriceLimitations(t *testing.T) {
	t.Parallel()
	_, err := h.FContractPriceLimitations(context.Background(),
		"BTC", "this_week", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFContractOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := h.FContractOpenInterest(context.Background(), "BTC", "this_week", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := h.FGetEstimatedDeliveryPrice(context.Background(), currency.BTC)
	require.NoError(t, err)
}

func TestFGetMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := h.FGetMarketDepth(context.Background(), btccwPair, "step5")
	require.NoError(t, err)
}

func TestFGetKlineData(t *testing.T) {
	t.Parallel()
	_, err := h.FGetKlineData(context.Background(), btccwPair, "5min", 5, time.Now().Add(-time.Minute*5), time.Now())
	require.NoError(t, err)
}

func TestFGetMarketOverviewData(t *testing.T) {
	t.Parallel()
	_, err := h.FGetMarketOverviewData(context.Background(), btccwPair)
	require.NoError(t, err)
}

func TestFLastTradeData(t *testing.T) {
	t.Parallel()
	_, err := h.FLastTradeData(context.Background(), btccwPair)
	require.NoError(t, err)
}

func TestFRequestPublicBatchTrades(t *testing.T) {
	t.Parallel()
	_, err := h.FRequestPublicBatchTrades(context.Background(), btccwPair, 50)
	require.NoError(t, err)
}

func TestFQueryTieredAdjustmentFactor(t *testing.T) {
	t.Parallel()
	_, err := h.FQueryTieredAdjustmentFactor(context.Background(), currency.BTC)
	require.NoError(t, err)
}

func TestFQueryHisOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := h.FQueryHisOpenInterest(context.Background(), "BTC", "this_week", "60min", "cont", 3)
	require.NoError(t, err)
}

func TestFQuerySystemStatus(t *testing.T) {
	t.Parallel()
	_, err := h.FQuerySystemStatus(context.Background(), currency.BTC)
	require.NoError(t, err)
}

func TestFQueryTopAccountsRatio(t *testing.T) {
	t.Parallel()
	_, err := h.FQueryTopAccountsRatio(context.Background(), "BTC", "5min")
	require.NoError(t, err)
}

func TestFQueryTopPositionsRatio(t *testing.T) {
	t.Parallel()
	_, err := h.FQueryTopPositionsRatio(context.Background(), "BTC", "5min")
	require.NoError(t, err)
}

func TestFLiquidationOrders(t *testing.T) {
	t.Parallel()
	if _, err := h.FLiquidationOrders(context.Background(), currency.BTC, "filled", 0, 0, "", 0); err != nil {
		t.Error(err)
	}
}

func TestFIndexKline(t *testing.T) {
	t.Parallel()
	_, err := h.FIndexKline(context.Background(), btccwPair, "5min", 5)
	require.NoError(t, err)
}

func TestFGetBasisData(t *testing.T) {
	t.Parallel()
	_, err := h.FGetBasisData(context.Background(), btccwPair, "5min", "open", 3)
	require.NoError(t, err)
}

func TestFGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetAccountInfo(context.Background(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetPositionsInfo(context.Background(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetAllSubAccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetAllSubAccountAssets(context.Background(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetSingleSubAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetSingleSubAccountInfo(context.Background(), "", "154263566")
	require.NoError(t, err)
}

func TestFGetSingleSubPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetSingleSubPositions(context.Background(), "", "154263566")
	require.NoError(t, err)
}

func TestFGetFinancialRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetFinancialRecords(context.Background(),
		"BTC", "closeLong", 2, 0, 0)
	require.NoError(t, err)
}

func TestFGetSettlementRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetSettlementRecords(context.Background(),
		currency.BTC, 0, 0, time.Now().Add(-48*time.Hour), time.Now())
	require.NoError(t, err)
}

func TestFContractTradingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FContractTradingFee(context.Background(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetTransferLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetTransferLimits(context.Background(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetPositionLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetPositionLimits(context.Background(), currency.EMPTYCODE)
	require.NoError(t, err)
}

func TestFGetAssetsAndPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetAssetsAndPositions(context.Background(), currency.HT)
	require.NoError(t, err)
}

func TestFTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FTransfer(context.Background(), "154263566", "HT", "sub_to_master", 5)
	require.NoError(t, err)
}

func TestFGetTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetTransferRecords(context.Background(), "HT", "master_to_sub", 90, 0, 0)
	require.NoError(t, err)
}

func TestFGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetAvailableLeverage(context.Background(), currency.BTC)
	require.NoError(t, err)
}

func TestFOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.FOrder(context.Background(), currency.EMPTYPAIR, "BTC", "quarter", "123", "BUY", "open", "limit", 1, 1, 1)
	require.NoError(t, err)
}

func TestFPlaceBatchOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
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
	_, err := h.FPlaceBatchOrder(context.Background(), req)
	require.NoError(t, err)
}

func TestFCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.FCancelOrder(context.Background(), currency.BTC, "123", "")
	require.NoError(t, err)
}

func TestFCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	updatePairsOnce(t)
	_, err := h.FCancelAllOrders(context.Background(), btcFutureDatedPair, "", "")
	require.NoError(t, err)
}

func TestFFlashCloseOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.FFlashCloseOrder(context.Background(),
		currency.EMPTYPAIR, "BTC", "quarter", "BUY", "lightning", "", 1)
	require.NoError(t, err)
}

func TestFGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetOrderInfo(context.Background(), "BTC", "", "123")
	require.NoError(t, err)
}

func TestFOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FOrderDetails(context.Background(), "BTC", "123", "quotation", time.Now().Add(-1*time.Hour), 0, 0)
	require.NoError(t, err)
}

func TestFGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetOpenOrders(context.Background(), currency.BTC, 1, 2)
	require.NoError(t, err)
}

func TestFGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FGetOrderHistory(context.Background(),
		currency.EMPTYPAIR, "BTC",
		"all", "all", "limit",
		[]order.Status{},
		5, 0, 0)
	require.NoError(t, err)
}

func TestFTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.FTradeHistory(context.Background(), currency.EMPTYPAIR, "BTC", "all", 10, 0, 0)
	require.NoError(t, err)
}

func TestFPlaceTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.FPlaceTriggerOrder(context.Background(), currency.EMPTYPAIR, "EOS", "quarter", "greaterOrEqual", "limit", "buy", "close", 1.1, 1.05, 5, 2)
	require.NoError(t, err)
}

func TestFCancelTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.FCancelTriggerOrder(context.Background(), "ETH", "123")
	require.NoError(t, err)
}

func TestFCancelAllTriggerOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.FCancelAllTriggerOrders(context.Background(), currency.EMPTYPAIR, "BTC", "this_week")
	require.NoError(t, err)
}

func TestFQueryTriggerOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.FQueryTriggerOpenOrders(context.Background(), currency.EMPTYPAIR, "BTC", 0, 0)
	require.NoError(t, err)
}

func TestFQueryTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.FQueryTriggerOrderHistory(context.Background(), currency.EMPTYPAIR, "EOS", "all", "all", 10, 0, 0)
	require.NoError(t, err)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := h.FetchTradablePairs(context.Background(), asset.Futures)
	require.NoError(t, err)
}

func TestUpdateTickerSpot(t *testing.T) {
	t.Parallel()
	_, err := h.UpdateTicker(context.Background(), currency.NewPairWithDelimiter("INV", "ALID", "-"), asset.Spot)
	assert.ErrorContains(t, err, "invalid symbol")
	_, err = h.UpdateTicker(context.Background(), currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Spot)
	require.NoError(t, err)
}

func TestUpdateTickerCMF(t *testing.T) {
	t.Parallel()
	_, err := h.UpdateTicker(context.Background(), currency.NewPairWithDelimiter("INV", "ALID", "_"), asset.CoinMarginedFutures)
	assert.ErrorContains(t, err, "symbol data error")
	_, err = h.UpdateTicker(context.Background(), currency.NewPairWithDelimiter("BTC", "USD", "_"), asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateTickerFutures(t *testing.T) {
	t.Parallel()
	_, err := h.UpdateTicker(context.Background(), btccwPair, asset.Futures)
	require.NoError(t, err)
}

func TestUpdateOrderbookSpot(t *testing.T) {
	t.Parallel()
	_, err := h.UpdateOrderbook(context.Background(), btcusdtPair, asset.Spot)
	require.NoError(t, err)
}

func TestUpdateOrderbookCMF(t *testing.T) {
	t.Parallel()
	_, err := h.UpdateOrderbook(context.Background(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestUpdateOrderbookFuture(t *testing.T) {
	t.Parallel()
	_, err := h.UpdateOrderbook(context.Background(), btccwPair, asset.Futures)
	require.NoError(t, err)
	_, err = h.UpdateOrderbook(context.Background(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	updatePairsOnce(t)
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := h.GetOrderHistory(context.Background(), &getOrdersRequest)
	require.NoError(t, err)

	cp1, err := currency.NewPairFromString("ADA-USD")
	require.NoError(t, err)
	getOrdersRequest.Pairs = []currency.Pair{cp1}
	getOrdersRequest.AssetType = asset.CoinMarginedFutures
	_, err = h.GetOrderHistory(context.Background(), &getOrdersRequest)
	require.NoError(t, err)
	getOrdersRequest.Pairs = []currency.Pair{btcFutureDatedPair}
	getOrdersRequest.AssetType = asset.Futures
	_, err = h.GetOrderHistory(context.Background(), &getOrdersRequest)
	require.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.CancelAllOrders(context.Background(), &order.Cancel{AssetType: asset.Futures})
	require.NoError(t, err)
}

func TestQuerySwapIndexPriceInfo(t *testing.T) {
	t.Parallel()
	_, err := h.QuerySwapIndexPriceInfo(context.Background(), btcusdPair)
	require.NoError(t, err)
}

func TestSwapOpenInterestInformation(t *testing.T) {
	t.Parallel()
	_, err := h.SwapOpenInterestInformation(context.Background(), btcusdPair)
	require.NoError(t, err)
}

func TestGetSwapMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := h.GetSwapMarketDepth(context.Background(), btcusdPair, "step0")
	require.NoError(t, err)
}

func TestGetSwapKlineData(t *testing.T) {
	t.Parallel()
	_, err := h.GetSwapKlineData(context.Background(), btcusdPair, "5min", 5, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
}

func TestGetSwapMarketOverview(t *testing.T) {
	t.Parallel()
	_, err := h.GetSwapMarketOverview(context.Background(), btcusdPair)
	require.NoError(t, err)
}

func TestGetLastTrade(t *testing.T) {
	t.Parallel()
	_, err := h.GetLastTrade(context.Background(), btcusdPair)
	require.NoError(t, err)
}

func TestGetBatchTrades(t *testing.T) {
	t.Parallel()
	_, err := h.GetBatchTrades(context.Background(), btcusdPair, 5)
	require.NoError(t, err)
}

func TestGetTieredAjustmentFactorInfo(t *testing.T) {
	t.Parallel()
	_, err := h.GetTieredAjustmentFactorInfo(context.Background(), btcusdPair)
	require.NoError(t, err)
}

func TestGetOpenInterestInfo(t *testing.T) {
	t.Parallel()
	_, err := h.GetOpenInterestInfo(context.Background(), btcusdPair, "5min", "cryptocurrency", 50)
	require.NoError(t, err)
}

func TestGetTraderSentimentIndexAccount(t *testing.T) {
	t.Parallel()
	_, err := h.GetTraderSentimentIndexAccount(context.Background(), btcusdPair, "5min")
	require.NoError(t, err)
}

func TestGetTraderSentimentIndexPosition(t *testing.T) {
	t.Parallel()
	_, err := h.GetTraderSentimentIndexPosition(context.Background(), btcusdPair, "5min")
	require.NoError(t, err)
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := h.GetLiquidationOrders(context.Background(), btcusdPair, "closed", 0, 0, "", 0)
	require.NoError(t, err)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := h.GetHistoricalFundingRatesForPair(context.Background(), btcusdPair, 0, 0)
	require.NoError(t, err)
}

func TestGetPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	_, err := h.GetPremiumIndexKlineData(context.Background(), btcusdPair, "5min", 15)
	require.NoError(t, err)
}

func TestGetEstimatedFundingRates(t *testing.T) {
	t.Parallel()
	_, err := h.GetPremiumIndexKlineData(context.Background(), btcusdPair, "5min", 15)
	require.NoError(t, err)
}

func TestGetBasisData(t *testing.T) {
	t.Parallel()
	_, err := h.GetBasisData(context.Background(), btcusdPair, "5min", "close", 5)
	require.NoError(t, err)
}

func TestGetSystemStatusInfo(t *testing.T) {
	t.Parallel()
	_, err := h.GetSystemStatusInfo(context.Background(), btcusdPair)
	require.NoError(t, err)
}

func TestGetSwapPriceLimits(t *testing.T) {
	t.Parallel()
	_, err := h.GetSwapPriceLimits(context.Background(), btcusdPair)
	require.NoError(t, err)
}

func TestGetMarginRates(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetMarginRates(context.Background(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetSwapAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapAccountInfo(context.Background(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapPositionsInfo(context.Background(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapAssetsAndPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapAssetsAndPositions(context.Background(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapAllSubAccAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapAllSubAccAssets(context.Background(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSubAccPositionInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSubAccPositionInfo(context.Background(), ethusdPair, 0)
	require.NoError(t, err)
}

func TestGetAccountFinancialRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetAccountFinancialRecords(context.Background(), ethusdPair, "3,4", 15, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapSettlementRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapSettlementRecords(context.Background(), ethusdPair, time.Time{}, time.Time{}, 0, 0)
	require.NoError(t, err)
}

func TestGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetAvailableLeverage(context.Background(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapOrderLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapOrderLimitInfo(context.Background(), ethusdPair, "limit")
	require.NoError(t, err)
}

func TestGetSwapTradingFeeInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapTradingFeeInfo(context.Background(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapTransferLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapTransferLimitInfo(context.Background(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapPositionLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapPositionLimitInfo(context.Background(), ethusdPair)
	require.NoError(t, err)
}

func TestAccountTransferData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.AccountTransferData(context.Background(), ethusdPair, "123", "master_to_sub", 15)
	require.NoError(t, err)
}

func TestAccountTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.AccountTransferRecords(context.Background(), ethusdPair, "master_to_sub", 12, 0, 0)
	require.NoError(t, err)
}

func TestPlaceSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.PlaceSwapOrders(context.Background(), ethusdPair, "", "buy", "open", "limit", 0.01, 1, 1)
	require.NoError(t, err)
}

func TestPlaceSwapBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
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

	_, err := h.PlaceSwapBatchOrders(context.Background(), req)
	require.NoError(t, err)
}

func TestCancelSwapOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.CancelSwapOrder(context.Background(), "test123", "", ethusdPair)
	require.NoError(t, err)
}

func TestCancelAllSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.CancelAllSwapOrders(context.Background(), ethusdPair)
	require.NoError(t, err)
}

func TestPlaceLightningCloseOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.PlaceLightningCloseOrder(context.Background(), ethusdPair, "buy", "lightning", 5, 1)
	require.NoError(t, err)
}

func TestGetSwapOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapOrderInfo(context.Background(), ethusdPair, "123", "")
	require.NoError(t, err)
}

func TestGetSwapOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapOrderDetails(context.Background(), ethusdPair, "123", "10", "cancelledOrder", 0, 0)
	require.NoError(t, err)
}

func TestGetSwapOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapOpenOrders(context.Background(), ethusdPair, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapOrderHistory(context.Background(), ethusdPair, "all", "all", []order.Status{order.PartiallyCancelled, order.Active}, 25, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapTradeHistory(context.Background(), ethusdPair, "liquidateShort", 10, 0, 0)
	require.NoError(t, err)
}

func TestPlaceSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.PlaceSwapTriggerOrder(context.Background(), ethusdPair, "greaterOrEqual", "buy", "open", "optimal_5", 5, 3, 1, 1)
	require.NoError(t, err)
}

func TestCancelSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.CancelSwapTriggerOrder(context.Background(), ethusdPair, "test123")
	require.NoError(t, err)
}

func TestCancelAllSwapTriggerOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.CancelAllSwapTriggerOrders(context.Background(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetSwapTriggerOrderHistory(context.Background(), ethusdPair, "open", "all", 15, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapMarkets(t *testing.T) {
	t.Parallel()
	_, err := h.GetSwapMarkets(context.Background(), currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := h.GetSpotKline(context.Background(), KlinesRequestParams{Symbol: btcusdtPair, Period: "1min"})
	require.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()

	updatePairsOnce(t)

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err := h.GetHistoricCandles(context.Background(), btcusdtPair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	require.NoError(t, err)

	_, err = h.GetHistoricCandles(context.Background(), btcusdtPair, asset.Spot, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	_, err = h.GetHistoricCandles(context.Background(), btcFutureDatedPair, asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	_, err = h.GetHistoricCandles(context.Background(), btcusdPair, asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()

	updatePairsOnce(t)

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err := h.GetHistoricCandlesExtended(context.Background(), btcusdtPair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	_, err = h.GetHistoricCandlesExtended(context.Background(), btcFutureDatedPair, asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	// demonstrate that adjusting time doesn't wreck non-day intervals
	_, err = h.GetHistoricCandlesExtended(context.Background(), btcFutureDatedPair, asset.Futures, kline.OneHour, endTime.AddDate(0, 0, -1), endTime)
	require.NoError(t, err)

	_, err = h.GetHistoricCandlesExtended(context.Background(), btcusdPair, asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), time.Now())
	require.NoError(t, err)

	_, err = h.GetHistoricCandlesExtended(context.Background(), btcusdPair, asset.CoinMarginedFutures, kline.OneHour, endTime.AddDate(0, 0, -1), time.Now())
	require.NoError(t, err)
}

func TestGetMarketDetailMerged(t *testing.T) {
	t.Parallel()
	_, err := h.GetMarketDetailMerged(context.Background(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := h.GetDepth(context.Background(),
		&OrderBookDataRequestParams{
			Symbol: btcusdtPair,
			Type:   OrderBookDataRequestParamsTypeStep1,
		})
	require.NoError(t, err)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := h.GetTrades(context.Background(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := h.GetLatestSpotPrice(context.Background(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := h.GetTradeHistory(context.Background(), btcusdtPair, 50)
	require.NoError(t, err)
}

func TestGetMarketDetail(t *testing.T) {
	t.Parallel()
	_, err := h.GetMarketDetail(context.Background(), btcusdtPair)
	require.NoError(t, err)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := h.GetSymbols(context.Background())
	require.NoError(t, err)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := h.GetCurrencies(context.Background())
	require.NoError(t, err)
}

func TestGet24HrMarketSummary(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("ethusdt")
	require.NoError(t, err)
	_, err = h.Get24HrMarketSummary(context.Background(), cp)
	require.NoError(t, err)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := h.GetTickers(context.Background())
	require.NoError(t, err)
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	st, err := h.GetCurrentServerTime(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, st, "GetCurrentServerTime should return a time")
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := h.GetServerTime(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, st, "GetServerTime should return a time")
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.GetAccounts(context.Background())
	require.NoError(t, err)
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	result, err := h.GetAccounts(context.Background())
	require.NoError(t, err, "GetAccounts must not error")

	userID := strconv.FormatInt(result[0].ID, 10)
	_, err = h.GetAccountBalance(context.Background(), userID)
	require.NoError(t, err, "GetAccountBalance must not error")
}

func TestGetAggregatedBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetAggregatedBalance(context.Background())
	require.NoError(t, err)
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	arg := SpotNewOrderRequestParams{
		Symbol:    btcusdtPair,
		AccountID: 1997024,
		Amount:    0.01,
		Price:     10.1,
		Type:      SpotNewOrderRequestTypeBuyLimit,
	}

	_, err := h.SpotNewOrder(context.Background(), &arg)
	require.NoError(t, err)
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.CancelExistingOrder(context.Background(), 1337)
	assert.Error(t, err)
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.GetOrder(context.Background(), 1337)
	require.NoError(t, err)
}

func TestGetMarginLoanOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetMarginLoanOrders(context.Background(), btcusdtPair, "", "", "", "", "", "", "")
	require.NoError(t, err)
}

func TestGetMarginAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetMarginAccountBalance(context.Background(), btcusdtPair)
	require.NoError(t, err)
}

func TestCancelWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.CancelWithdraw(context.Background(), 1337)
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

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	_, err := h.GetFeeByType(context.Background(), feeBuilder)
	require.NoError(t, err)
	if !sharedtestvalues.AreAPICredentialsSet(h) {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	_, err := h.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	_, err = h.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	_, err = h.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	_, err = h.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = h.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = h.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = h.GetFee(feeBuilder)
	require.NoError(t, err)

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	_, err = h.GetFee(feeBuilder)
	require.NoError(t, err)

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	_, err = h.GetFee(feeBuilder)
	require.NoError(t, err)
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := h.FormatWithdrawPermissions()
	assert.Equal(t, expectedResult, withdrawPermissions)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.MultiOrderRequest{
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)},
		Side:      order.AnySide,
	}

	_, err := h.GetActiveOrders(context.Background(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(h) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	accounts, err := h.GetAccounts(context.Background())
	require.NoError(t, err, "GetAccounts must not error")

	var orderSubmission = &order.Submit{
		Exchange: h.Name,
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
	response, err := h.SubmitOrder(context.Background(), orderSubmission)
	require.NoError(t, err)
	assert.Equal(t, order.New, response.Status, "response status should be correct")
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          btcusdtPair,
		AssetType:     asset.Spot,
	}

	err := h.CancelOrder(context.Background(), orderCancellation)
	require.NoError(t, err)
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	_, err := h.CancelAllOrders(context.Background(), &orderCancellation)
	require.NoError(t, err)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	for _, a := range []asset.Item{asset.Spot, asset.CoinMarginedFutures, asset.Futures} {
		_, err := h.UpdateAccountInfo(context.Background(), a)
		assert.NoErrorf(t, err, "UpdateAccountInfo should not error for asset %s", a)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)
	_, err := h.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Spot})
	require.Error(t, err, "ModifyOrder must error without any order details")
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    h.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := h.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest)
	require.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero)
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := h.WithdrawFiatFunds(context.Background(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := h.WithdrawFiatFundsToInternationalBank(context.Background(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestQueryDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := h.QueryDepositAddress(context.Background(), currency.USDT)
	if sharedtestvalues.AreAPICredentialsSet(h) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := h.GetDepositAddress(context.Background(), currency.USDT, "", "uSdTeRc20")
	if sharedtestvalues.AreAPICredentialsSet(h) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
	}
}

func TestQueryWithdrawQuota(t *testing.T) {
	t.Parallel()
	_, err := h.QueryWithdrawQuotas(context.Background(), currency.BTC.Lower().String())
	if sharedtestvalues.AreAPICredentialsSet(h) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
	}
}

// TestWsGetAccountsList connects to WS, logs in, gets account list
func TestWsGetAccountsList(t *testing.T) {
	setupWsTests(t)
	if _, err := h.wsGetAccountsList(context.Background()); err != nil {
		t.Error(err)
	}
}

// TestWsGetOrderList connects to WS, logs in, gets order list
func TestWsGetOrderList(t *testing.T) {
	setupWsTests(t)
	p, err := currency.NewPairFromString("ethbtc")
	if err != nil {
		t.Error(err)
	}
	_, err = h.wsGetOrdersList(context.Background(), 1, p)
	if err != nil {
		t.Error(err)
	}
}

// TestWsGetOrderDetails connects to WS, logs in, gets order details
func TestWsGetOrderDetails(t *testing.T) {
	setupWsTests(t)
	orderID := "123"
	_, err := h.wsGetOrderDetails(context.Background(), orderID)
	if err != nil {
		t.Error(err)
	}
}

func TestWsSubResponse(t *testing.T) {
	pressXToJSON := []byte(`{
  "op": "sub",
  "cid": "123",
  "err-code": 0,
  "ts": 1489474081631,
  "topic": "accounts"
}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsKline(t *testing.T) {
	err := h.Websocket.AddSubscriptions(h.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.kline.1min", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.CandlesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	pressXToJSON := []byte(`{
  "ch": "market.btcusdt.kline.1min",
  "ts": 1489474082831,
  "tick": {
    "id": 1489464480,
    "amount": 0.0,
    "count": 0,
    "open": 7962.62,
    "close": 7962.62,
    "low": 7962.62,
    "high": 7962.62,
    "vol": 0.0
  }
}`)
	err = h.wsHandleData(pressXToJSON)
	require.NoError(t, err)
}

func TestWsKlineArray(t *testing.T) {
	pressXToJSON := []byte(`{
  "status": "ok",
  "rep": "market.btcusdt.kline.1min",
  "data": [
    {
      "amount": 1.6206,
      "count":  3,
      "id":     1494465840,
      "open":   9887.00,
      "close":  9885.00,
      "low":    9885.00,
      "high":   9887.00,
      "vol":    16021.632026
    },
    {
      "amount": 2.2124,
      "count":  6,
      "id":     1494465900,
      "open":   9885.00,
      "close":  9880.00,
      "low":    9880.00,
      "high":   9885.00,
      "vol":    21859.023500
    }
  ]
}`)
	err := h.wsHandleData(pressXToJSON)
	require.NoError(t, err)
}

func TestWsMarketDepth(t *testing.T) {
	err := h.Websocket.AddSubscriptions(h.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.depth.step0", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.OrderbookChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	pressXToJSON := []byte(`{
  "ch": "market.btcusdt.depth.step0",
  "ts": 1572362902027,
  "tick": {
    "bids": [
      [3.7721, 344.86],
      [3.7709, 46.66]
    ],
    "asks": [
      [3.7745, 15.44],
      [3.7746, 70.52]
    ],
    "version": 100434317651,
    "ts": 1572362902012
  }
}`)
	err = h.wsHandleData(pressXToJSON)
	require.NoError(t, err)
}

func TestWsTradeDetail(t *testing.T) {
	err := h.Websocket.AddSubscriptions(h.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.trade.detail", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.AllTradesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	pressXToJSON := []byte(`{
	  "ch": "market.btcusdt.trade.detail",
	  "ts": 1489474082831,
	  "tick": {
			"id": 14650745135,
			"ts": 1533265950234,
			"data": [
				{
					"amount": 0.0099,
					"ts": 1533265950234,
					"id": 146507451359183894799,
					"tradeId": 102043495674,
					"price": 401.74,
					"direction": "buy"
				}
			]
	  }
	}`)
	err = h.wsHandleData(pressXToJSON)
	require.NoError(t, err)
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{
  "rep": "market.btcusdt.detail",
  "id": "id11",
	"data":{
		"amount": 12224.2922,
		"open":   9790.52,
		"close":  10195.00,
		"high":   10300.00,
		"ts":     1494496390000,
		"id":     1494496390,
		"count":  15195,
		"low":    9657.00,
		"vol":    121906001.754751
	  }
}`)
	err := h.wsHandleData(pressXToJSON)
	require.NoError(t, err)
}

func TestWsAccountUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
	  "op": "notify",
	  "ts": 1522856623232,
	  "topic": "accounts",
	  "data": {
		"event": "order.place",
		"list": [
		  {
			"account-id": 419013,
			"currency": "usdt",
			"type": "trade",
			"balance": "500009195917.4362872650"
		  }
		]
	  }
	}`)
	err := h.wsHandleData(pressXToJSON)
	require.NoError(t, err)
}

func TestWsOrderUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
  "op": "notify",
  "topic": "orders.htusdt",
  "ts": 1522856623232,
  "data": {
    "seq-id": 94984,
    "order-id": 2039498445,
    "symbol": "btcusdt",
    "account-id": 100077,
    "order-amount": "5000.000000000000000000",
    "order-price": "1.662100000000000000",
    "created-at": 1522858623622,
    "order-type": "buy-limit",
    "order-source": "api",
    "order-state": "filled",
    "role": "taker",
    "price": "1.662100000000000000",
    "filled-amount": "5000.000000000000000000",
    "unfilled-amount": "0.000000000000000000",
    "filled-cash-amount": "8301.357280000000000000",
    "filled-fees": "8.000000000000000000"
  }
}`)
	err := h.wsHandleData(pressXToJSON)
	require.NoError(t, err)
}

func TestWsMarketByPrice(t *testing.T) {
	err := h.Websocket.AddSubscriptions(h.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.mbp.150", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.OrderbookChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	pressXToJSON := []byte(`{
		"ch": "market.btcusdt.mbp.150",
		"ts": 1573199608679,
		"tick": {
			"seqNum": 100020146795,
			"prevSeqNum": 100020146794,
			"bids": [],
			"asks": [
				[645.140000000000000000, 26.755973959140651643]
			]
		}
	}`)
	err = h.wsHandleData(pressXToJSON)
	require.NoError(t, err)
	pressXToJSON = []byte(`{
		"id": "id2",
		"rep": "market.btcusdt.mbp.150",
		"status": "ok",
		"data": {
			"seqNum": 100020142010,
			"bids": [
				[618.37, 71.594],
				[423.33, 77.726],
				[223.18, 47.997],
				[219.34, 24.82],
				[210.34, 94.463]
		],
			"asks": [
				[650.59, 14.909733438479636],
				[650.63, 97.996],
				[650.77, 97.465],
				[651.23, 83.973],
				[651.42, 34.465]
			]
		}
	}`)
	err = h.wsHandleData(pressXToJSON)
	require.NoError(t, err)
}

func TestWsOrdersUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
		"op": "notify",
		"ts": 1522856623232,
		"topic": "orders.btcusdt.update",
		"data": {
		"unfilled-amount": "0.000000000000000000",
			"filled-amount": "5000.000000000000000000",
			"price": "1.662100000000000000",
			"order-id": 2039498445,
			"symbol": "btcusdt",
			"match-id": 94984,
			"filled-cash-amount": "8301.357280000000000000",
			"role": "taker|maker",
			"order-state": "filled",
			"client-order-id": "a0001",
			"order-type": "buy-limit"
	}
	}`)
	err := h.wsHandleData(pressXToJSON)
	require.NoError(t, err)
}

func TestStringToOrderStatus(t *testing.T) {
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

func Test_FormatExchangeKlineInterval(t *testing.T) {
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
		assert.Equalf(t, tt.output, h.FormatExchangeKlineInterval(tt.interval), "FormatExchangeKlineInterval should return correctly for %s", tt.output)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := h.GetRecentTrades(context.Background(), btcusdtPair, asset.Spot)
	require.NoError(t, err)
	_, err = h.GetRecentTrades(context.Background(), btccwPair, asset.Futures)
	require.NoError(t, err)
	_, err = h.GetRecentTrades(context.Background(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := h.GetHistoricTrades(context.Background(), btcusdtPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	c, err := h.GetAvailableTransferChains(context.Background(), currency.USDT)
	require.NoError(t, err)
	require.Greater(t, len(c), 2, "Must get more than 2 chains")
}

func TestFormatFuturesPair(t *testing.T) {
	updatePairsOnce(t)

	r, err := h.formatFuturesPair(btccwPair, false)
	require.NoError(t, err)
	assert.Equal(t, "BTC_CW", r)

	// pair in the format of BTC210827 but make it lower case to test correct formatting
	r, err = h.formatFuturesPair(btcFutureDatedPair.Lower(), false)
	require.NoError(t, err)
	assert.Len(t, r, 9, "Should be an 9 character string")
	assert.Equal(t, "BTC2", r[0:4], "Should start with btc and a date this millenium")

	r, err = h.formatFuturesPair(btccwPair, true)
	require.NoError(t, err)
	assert.Len(t, r, 9, "Should be an 9 character string")
	assert.Equal(t, "BTC2", r[0:4], "Should start with btc and a date this millenium")

	r, err = h.formatFuturesPair(currency.NewPair(currency.BTC, currency.USDT), false)
	require.NoError(t, err)
	assert.Equal(t, "BTC-USDT", r)
}

func TestSearchForExistedWithdrawsAndDeposits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.SearchForExistedWithdrawsAndDeposits(context.Background(), currency.BTC, "deposit", "", 0, 100)
	require.NoError(t, err)
}

func TestCancelOrderBatch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.CancelOrderBatch(context.Background(), []string{"1234"}, nil)
	require.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.BTC, currency.USDT),
		},
	})
	require.NoError(t, err)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)
	_, err := h.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	require.NoError(t, err)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := h.GetFuturesContractDetails(context.Background(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = h.GetFuturesContractDetails(context.Background(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = h.GetFuturesContractDetails(context.Background(), asset.CoinMarginedFutures)
	require.ErrorIs(t, err, nil)
	_, err = h.GetFuturesContractDetails(context.Background(), asset.Futures)
	require.ErrorIs(t, err, nil)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := h.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewPair(currency.BTC, currency.USD),
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = h.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.CoinMarginedFutures,
		Pair:                 currency.NewPair(currency.BTC, currency.USD),
		IncludePredictedRate: true,
	})
	require.NoError(t, err)

	err = h.CurrencyPairs.EnablePair(asset.CoinMarginedFutures, currency.NewPair(currency.BTC, currency.USD))
	require.ErrorIs(t, err, currency.ErrPairAlreadyEnabled)

	_, err = h.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Asset:                asset.CoinMarginedFutures,
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := h.IsPerpetualFutureCurrency(asset.Binary, currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.False(t, is)

	is, err = h.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	assert.True(t, is)
}

func TestGetSwapFundingRates(t *testing.T) {
	t.Parallel()
	_, err := h.GetSwapFundingRates(context.Background())
	require.NoError(t, err)
}

func TestGetBatchCoinMarginSwapContracts(t *testing.T) {
	t.Parallel()
	resp, err := h.GetBatchCoinMarginSwapContracts(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetBatchLinearSwapContracts(t *testing.T) {
	t.Parallel()
	resp, err := h.GetBatchLinearSwapContracts(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetBatchFuturesContracts(t *testing.T) {
	t.Parallel()
	resp, err := h.GetBatchFuturesContracts(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t)
	for _, a := range h.GetAssetTypes(false) {
		err := h.UpdateTickers(context.Background(), a)
		require.NoErrorf(t, err, "asset %s", a)
		avail, err := h.GetAvailablePairs(a)
		require.NoError(t, err)
		for _, p := range avail {
			_, err = ticker.GetTicker(h.Name, p, a)
			assert.NoErrorf(t, err, "Could not get ticker for %s %s", a, p)
		}
	}
}

func TestPairFromContractExpiryCode(t *testing.T) {
	t.Parallel()

	h := new(HUOBI) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(h), "Test Instance Setup must not fail")

	_, err := h.FetchTradablePairs(context.Background(), asset.Futures)
	require.NoError(t, err)

	n := time.Now().Truncate(24 * time.Hour)
	for _, cType := range contractExpiryNames {
		p, err := h.pairFromContractExpiryCode(currency.Pair{
			Base:  currency.BTC,
			Quote: currency.NewCode(cType),
		})
		if cType == "NQ" && err != nil {
			continue // Next Quarter is intermittently present
		}
		require.NoErrorf(t, err, "pairFromContractExpiryCode must not error for %s code", cType)
		assert.Equal(t, currency.BTC, p.Base, "pair Base should be the same")
		h.futureContractCodesMutex.RLock()
		exp, ok := h.futureContractCodes[cType]
		h.futureContractCodesMutex.RUnlock()
		require.True(t, ok, "%s type must be in contractExpiryNames", cType)
		assert.Equal(t, currency.BTC, p.Base, "pair Base should be the same")
		assert.Equal(t, exp, p.Quote, "pair Quote should be the same")
		d, err := time.Parse("060102", p.Quote.String())
		require.NoError(t, err, "currency code must be a parsable date")
		require.Falsef(t, d.Before(n), "%s expiry must be today or after", cType)
		switch cType {
		case "CW", "NW":
			require.True(t, d.Before(n.Add(24*time.Hour*14)), "%s expiry must be within 2 weeks", cType)
		case "CQ", "NQ":
			require.True(t, d.Before(n.Add(24*time.Hour*90*2)), "%s expiry must be within 2 quarters", cType)
		}
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t)

	_, err := h.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	resp, err := h.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USD.Item,
		Asset: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = h.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  btccwPair.Base.Item,
		Quote: btccwPair.Quote.Item,
		Asset: asset.Futures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = h.GetOpenInterest(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestContractOpenInterestUSDT(t *testing.T) {
	t.Parallel()
	resp, err := h.ContractOpenInterestUSDT(context.Background(), currency.EMPTYPAIR, currency.EMPTYPAIR, "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	cp := currency.NewPair(currency.BTC, currency.USDT)
	resp, err = h.ContractOpenInterestUSDT(context.Background(), cp, currency.EMPTYPAIR, "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = h.ContractOpenInterestUSDT(context.Background(), currency.EMPTYPAIR, cp, "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = h.ContractOpenInterestUSDT(context.Background(), cp, currency.EMPTYPAIR, "this_week", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = h.ContractOpenInterestUSDT(context.Background(), currency.EMPTYPAIR, currency.EMPTYPAIR, "", "swap")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t)
	for _, a := range h.GetAssetTypes(false) {
		pairs, err := h.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := h.GetCurrencyTradeURL(context.Background(), a, pairs[0])
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

	h := new(HUOBI)
	require.NoError(t, testexch.Setup(h), "Test instance Setup must not error")
	subs, err := h.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{}
	for _, s := range h.Features.Subscriptions {
		if s.Authenticated && !h.Websocket.CanUseAuthenticatedEndpoints() {
			continue
		}
		for _, a := range h.GetAssetTypes(true) {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err := h.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			pairs = common.SortStrings(pairs).Format(currency.PairFormat{Uppercase: false, Delimiter: ""})
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.Asset = a
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

// TestSubscribe exercises live public subscriptions
func TestSubscribe(t *testing.T) {
	t.Parallel()
	h := new(HUOBI)
	require.NoError(t, testexch.Setup(h), "Test instance Setup must not error")
	subs, err := h.Features.Subscriptions.ExpandTemplates(h)
	require.NoError(t, err, "ExpandTemplates must not error")
	testexch.SetupWs(t, h)
	err = h.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")
	got := h.Websocket.GetSubscriptions()
	require.Equal(t, 4, len(got), "Must get correct number of subscriptions")
	for _, s := range got {
		assert.Equal(t, subscription.SubscribedState, s.State())
	}
}

func TestChannelName(t *testing.T) {
	p := currency.NewPair(currency.BTC, currency.USD)
	assert.Equal(t, "market.BTCUSD.kline", channelName(&subscription.Subscription{Channel: subscription.CandlesChannel}, p))
	assert.Panics(t, func() { channelName(&subscription.Subscription{Channel: wsOrderbookChannel}, p) })
	assert.Panics(t, func() { channelName(&subscription.Subscription{Channel: subscription.MyAccountChannel}, p) }, "Should panic on V2 endpoints until implemented")
}

func TestBootstrap(t *testing.T) {
	t.Parallel()
	h := new(HUOBI) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(h), "Test Instance Setup must not fail")

	c, err := h.Bootstrap(context.Background())
	require.NoError(t, err)
	assert.True(t, c, "Bootstrap should return true to continue")

	h.futureContractCodes = nil
	h.Features.Enabled.AutoPairUpdates = false
	_, err = h.Bootstrap(context.Background())
	require.NoError(t, err)
	require.NotNil(t, h.futureContractCodes)
}

var updatePairsMutex sync.Mutex

func updatePairsOnce(tb testing.TB) {
	tb.Helper()

	updatePairsMutex.Lock()
	defer updatePairsMutex.Unlock()

	testexch.UpdatePairsOnce(tb, h)

	p, err := h.pairFromContractExpiryCode(btccwPair)
	require.NoError(tb, err, "pairFromContractCode must not error")
	err = h.CurrencyPairs.EnablePair(asset.Futures, p)
	require.NoError(tb, common.ExcludeError(err, currency.ErrPairAlreadyEnabled))
	btcFutureDatedPair = p
}
