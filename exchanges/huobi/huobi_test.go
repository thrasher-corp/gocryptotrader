package huobi

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	testSymbol              = "btcusdt"
)

var (
	h               = &HUOBI{}
	wsSetupRan      bool
	futuresTestPair = currency.NewPair(currency.BTC, currency.NewCode("CW")) // represents this week - NQ (next quarter) is erroring out.
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
	request.MaxRequestJobs = 100
	err = h.Setup(hConfig)
	if err != nil {
		log.Fatal("Huobi setup error", err)
	}

	err = h.UpdateTradablePairs(context.Background(), true)
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
		t.Skip(stream.WebsocketNotEnabled)
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

func TestStart(t *testing.T) {
	t.Parallel()
	err := h.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Errorf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = h.Start(context.Background(), &testWg)
	if err != nil {
		t.Error(err)
	}
	testWg.Wait()
}

func TestGetCurrenciesIncludingChains(t *testing.T) {
	t.Parallel()
	r, err := h.GetCurrenciesIncludingChains(context.Background(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
	if len(r) == 1 {
		t.Error("expected 1 result")
	}
	r, err = h.GetCurrenciesIncludingChains(context.Background(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
	if len(r) < 1 {
		t.Error("expected >= 1 results")
	}
}

func TestFGetContractInfo(t *testing.T) {
	t.Parallel()
	_, err := h.FGetContractInfo(context.Background(), "", "", currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestFIndexPriceInfo(t *testing.T) {
	t.Parallel()
	_, err := h.FIndexPriceInfo(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestFContractPriceLimitations(t *testing.T) {
	t.Parallel()
	_, err := h.FContractPriceLimitations(context.Background(),
		"BTC", "this_week", currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestFContractOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := h.FContractOpenInterest(context.Background(),
		"BTC", "this_week", currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := h.FGetEstimatedDeliveryPrice(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := h.FGetMarketDepth(context.Background(), futuresTestPair, "step5")
	if err != nil {
		t.Error(err)
	}
}

func TestFGetKlineData(t *testing.T) {
	t.Parallel()
	_, err := h.FGetKlineData(context.Background(), futuresTestPair, "5min", 5, time.Now().Add(-time.Minute*5), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestFGetMarketOverviewData(t *testing.T) {
	t.Parallel()
	_, err := h.FGetMarketOverviewData(context.Background(), futuresTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestFLastTradeData(t *testing.T) {
	t.Parallel()
	_, err := h.FLastTradeData(context.Background(), futuresTestPair)
	if err != nil {
		t.Error(err)
	}
}

func TestFRequestPublicBatchTrades(t *testing.T) {
	t.Parallel()
	_, err := h.FRequestPublicBatchTrades(context.Background(), futuresTestPair, 50)
	if err != nil {
		t.Error(err)
	}
}

func TestFQueryInsuranceAndClawbackData(t *testing.T) {
	t.Parallel()
	_, err := h.FQueryInsuranceAndClawbackData(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestFQueryHistoricalInsuranceData(t *testing.T) {
	t.Parallel()
	_, err := h.FQueryHistoricalInsuranceData(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestFQueryTieredAdjustmentFactor(t *testing.T) {
	t.Parallel()
	_, err := h.FQueryTieredAdjustmentFactor(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestFQueryHisOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := h.FQueryHisOpenInterest(context.Background(),
		"BTC", "this_week", "60min", "cont", 3)
	if err != nil {
		t.Error(err)
	}
}

func TestFQuerySystemStatus(t *testing.T) {
	t.Parallel()

	_, err := h.FQuerySystemStatus(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestFQueryTopAccountsRatio(t *testing.T) {
	t.Parallel()
	_, err := h.FQueryTopAccountsRatio(context.Background(), "BTC", "5min")
	if err != nil {
		t.Error(err)
	}
}

func TestFQueryTopPositionsRatio(t *testing.T) {
	t.Parallel()
	_, err := h.FQueryTopPositionsRatio(context.Background(), "BTC", "5min")
	if err != nil {
		t.Error(err)
	}
}

func TestFLiquidationOrders(t *testing.T) {
	t.Parallel()
	if _, err := h.FLiquidationOrders(context.Background(), currency.BTC, "filled", 0, 0, "", 0); err != nil {
		t.Error(err)
	}
}

func TestFIndexKline(t *testing.T) {
	t.Parallel()
	_, err := h.FIndexKline(context.Background(), futuresTestPair, "5min", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetBasisData(t *testing.T) {
	t.Parallel()
	_, err := h.FGetBasisData(context.Background(), futuresTestPair, "5min", "open", 3)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetAccountInfo(context.Background(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetPositionsInfo(context.Background(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetAllSubAccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetAllSubAccountAssets(context.Background(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetSingleSubAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetSingleSubAccountInfo(context.Background(), "", "154263566")
	if err != nil {
		t.Error(err)
	}
}

func TestFGetSingleSubPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetSingleSubPositions(context.Background(), "", "154263566")
	if err != nil {
		t.Error(err)
	}
}

func TestFGetFinancialRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetFinancialRecords(context.Background(),
		"BTC", "closeLong", 2, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetSettlementRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetSettlementRecords(context.Background(),
		currency.BTC, 0, 0, time.Now().Add(-48*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestFContractTradingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FContractTradingFee(context.Background(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetTransferLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetTransferLimits(context.Background(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetPositionLimits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetPositionLimits(context.Background(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetAssetsAndPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetAssetsAndPositions(context.Background(), currency.HT)
	if err != nil {
		t.Error(err)
	}
}

func TestFTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FTransfer(context.Background(),
		"154263566", "HT", "sub_to_master", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetTransferRecords(context.Background(),
		"HT", "master_to_sub", 90, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetAvailableLeverage(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestFOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	tradablePairs, err := h.CurrencyPairs.GetPairs(asset.Futures, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	_, err = h.FOrder(context.Background(),
		currency.EMPTYPAIR, tradablePairs[0].Base.Upper().String(),
		"quarter", "123", "BUY", "open", "limit", 1, 1, 1)
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Error(err)
	}
}

func TestFCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.FCancelOrder(context.Background(), currency.BTC, "123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestFCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	tradablePairs, err := h.CurrencyPairs.GetPairs(asset.Futures, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	_, err = h.FCancelAllOrders(context.Background(), tradablePairs[0], "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestFFlashCloseOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.FFlashCloseOrder(context.Background(),
		currency.EMPTYPAIR, "BTC", "quarter", "BUY", "lightning", "", 1)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetOrderInfo(context.Background(), "BTC", "", "123")
	if err != nil {
		t.Error(err)
	}
}

func TestFOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FOrderDetails(context.Background(),
		"BTC", "123", "quotation", time.Now().Add(-1*time.Hour), 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FGetOpenOrders(context.Background(), currency.BTC, 1, 2)
	if err != nil {
		t.Error(err)
	}
}

func TestFGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	tradablePairs, err := h.CurrencyPairs.GetPairs(asset.Futures, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	_, err = h.FGetOrderHistory(context.Background(),
		currency.EMPTYPAIR, tradablePairs[0].Base.Upper().String(),
		"all", "all", "limit",
		[]order.Status{},
		5, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestFTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.FTradeHistory(context.Background(),
		currency.EMPTYPAIR, "BTC", "all", 10, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestFPlaceTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.FPlaceTriggerOrder(context.Background(),
		currency.EMPTYPAIR, "EOS", "quarter", "greaterOrEqual",
		"limit", "buy", "close", 1.1, 1.05, 5, 2)
	if err != nil {
		t.Error(err)
	}
}

func TestFCancelTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.FCancelTriggerOrder(context.Background(), "ETH", "123")
	if err != nil {
		t.Error(err)
	}
}

func TestFCancelAllTriggerOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.FCancelAllTriggerOrders(context.Background(),
		currency.EMPTYPAIR, "BTC", "this_week")
	if err != nil {
		t.Error(err)
	}
}

func TestFQueryTriggerOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.FQueryTriggerOpenOrders(context.Background(),
		currency.EMPTYPAIR, "BTC", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestFQueryTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.FQueryTriggerOrderHistory(context.Background(),
		currency.EMPTYPAIR, "EOS", "all", "all", 10, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := h.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickerSpot(t *testing.T) {
	t.Parallel()
	_, err := h.UpdateTicker(context.Background(), currency.NewPairWithDelimiter("INV", "ALID", "-"), asset.Spot)
	if err == nil {
		t.Error("expected invalid pair")
	}
	_, err = h.UpdateTicker(context.Background(), currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickerCMF(t *testing.T) {
	t.Parallel()
	_, err := h.UpdateTicker(context.Background(), currency.NewPairWithDelimiter("INV", "ALID", "_"), asset.CoinMarginedFutures)
	if err == nil {
		t.Error("expected invalid contract code")
	}
	_, err = h.UpdateTicker(context.Background(), currency.NewPairWithDelimiter("BTC", "USD", "_"), asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickerFutures(t *testing.T) {
	t.Parallel()
	tradablePairs, err := h.CurrencyPairs.GetPairs(asset.Futures, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	_, err = h.UpdateTicker(context.Background(), tradablePairs[0], asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbookSpot(t *testing.T) {
	t.Parallel()
	sp, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Error(err)
	}
	_, err = h.UpdateOrderbook(context.Background(), sp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbookCMF(t *testing.T) {
	t.Parallel()
	cp1, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.UpdateOrderbook(context.Background(), cp1, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbookFuture(t *testing.T) {
	t.Parallel()
	tradablePairs, err := h.CurrencyPairs.GetPairs(asset.Futures, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	_, err = h.UpdateOrderbook(context.Background(), tradablePairs[0], asset.Futures)
	if err != nil {
		t.Error(err)
	}
	tradablePairs, err = h.CurrencyPairs.GetPairs(asset.CoinMarginedFutures, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	_, err = h.UpdateOrderbook(context.Background(), tradablePairs[0], asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := h.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}

	cp1, err := currency.NewPairFromString("ADA-USD")
	if err != nil {
		t.Error(err)
	}
	getOrdersRequest.Pairs = []currency.Pair{cp1}
	getOrdersRequest.AssetType = asset.CoinMarginedFutures
	_, err = h.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
	tradablePairs, err := h.CurrencyPairs.GetPairs(asset.Futures, false)
	if err != nil {
		t.Error(err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("no tradable pairs")
	}
	getOrdersRequest.Pairs = []currency.Pair{tradablePairs[0]}
	getOrdersRequest.AssetType = asset.Futures
	_, err = h.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.CancelAllOrders(context.Background(),
		&order.Cancel{AssetType: asset.Futures})
	if err != nil {
		t.Error(err)
	}
}

func TestQuerySwapIndexPriceInfo(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.QuerySwapIndexPriceInfo(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestSwapOpenInterestInformation(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.SwapOpenInterestInformation(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapMarketDepth(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapMarketDepth(context.Background(), cp, "step0")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapKlineData(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapKlineData(context.Background(),
		cp, "5min", 5, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapMarketOverview(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapMarketOverview(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLastTrade(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetLastTrade(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBatchTrades(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetBatchTrades(context.Background(), cp, 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetInsuranceData(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetInsuranceData(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricalInsuranceData(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetHistoricalInsuranceData(context.Background(), cp, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTieredAjustmentFactorInfo(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetTieredAjustmentFactorInfo(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenInterestInfo(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetOpenInterestInfo(context.Background(),
		cp, "5min", "cryptocurrency", 50)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTraderSentimentIndexAccount(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetTraderSentimentIndexAccount(context.Background(), cp, "5min")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTraderSentimentIndexPosition(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetTraderSentimentIndexPosition(context.Background(), cp, "5min")
	if err != nil {
		t.Error(err)
	}
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}

	if _, err = h.GetLiquidationOrders(context.Background(), cp, "closed", 0, 0, "", 0); err != nil {
		t.Error(err)
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetHistoricalFundingRates(context.Background(), cp, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetPremiumIndexKlineData(context.Background(), cp, "5min", 15)
	if err != nil {
		t.Error(err)
	}
}

func TestGetEstimatedFundingRates(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetPremiumIndexKlineData(context.Background(), cp, "5min", 15)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBasisData(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetBasisData(context.Background(), cp, "5min", "close", 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSystemStatusInfo(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSystemStatusInfo(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapPriceLimits(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapPriceLimits(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginRates(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetMarginRates(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapAccountInfo(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapPositionsInfo(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapAssetsAndPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapAssetsAndPositions(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapAllSubAccAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapAllSubAccAssets(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccPositionInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSubAccPositionInfo(context.Background(), cp, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountFinancialRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetAccountFinancialRecords(context.Background(), cp, "3,4", 15, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapSettlementRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapSettlementRecords(context.Background(),
		cp, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetAvailableLeverage(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapOrderLimitInfo(context.Background(), cp, "limit")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapTradingFeeInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapTradingFeeInfo(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapTransferLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapTransferLimitInfo(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapPositionLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapPositionLimitInfo(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestAccountTransferData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.AccountTransferData(context.Background(),
		cp, "123", "master_to_sub", 15)
	if err != nil {
		t.Error(err)
	}
}

func TestAccountTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.AccountTransferRecords(context.Background(),
		cp, "master_to_sub", 12, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.PlaceSwapOrders(context.Background(),
		cp, "", "buy", "open", "limit", 0.01, 1, 1)
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSwapOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.CancelSwapOrder(context.Background(), "test123", "", cp)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.CancelAllSwapOrders(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceLightningCloseOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.PlaceLightningCloseOrder(context.Background(),
		cp, "buy", "lightning", 5, 1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapOrderInfo(context.Background(), cp, "123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapOrderDetails(context.Background(),
		cp, "123", "10", "cancelledOrder", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapOpenOrders(context.Background(), cp, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapOrderHistory(context.Background(),
		cp, "all", "all",
		[]order.Status{order.PartiallyCancelled, order.Active}, 25, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapTradeHistory(context.Background(),
		cp, "liquidateShort", 10, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.PlaceSwapTriggerOrder(context.Background(),
		cp, "greaterOrEqual", "buy", "open", "optimal_5", 5, 3, 1, 1)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.CancelSwapTriggerOrder(context.Background(), cp, "test123")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllSwapTriggerOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.CancelAllSwapTriggerOrders(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString("ETH-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSwapTriggerOrderHistory(context.Background(),
		cp, "open", "all", 15, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSwapMarkets(t *testing.T) {
	t.Parallel()
	_, err := h.GetSwapMarkets(context.Background(), currency.EMPTYPAIR)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetSpotKline(context.Background(), KlinesRequestParams{
		Symbol: cp,
		Period: "1min",
	})
	if err != nil {
		t.Errorf("Huobi TestGetSpotKline: %s", err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Error(err)
	}

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err = h.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	if err != nil {
		t.Error(err)
	}

	_, err = h.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	if err != nil {
		t.Error(err)
	}

	pairs, err := h.CurrencyPairs.GetPairs(asset.Futures, false)
	if err != nil {
		t.Error(err)
	}
	err = h.CurrencyPairs.EnablePair(asset.Futures, pairs[0])
	if err != nil && !errors.Is(err, currency.ErrPairAlreadyEnabled) {
		t.Error(err)
	}
	_, err = h.GetHistoricCandles(context.Background(), pairs[0], asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	if err != nil {
		t.Error(err)
	}

	pairs, err = h.CurrencyPairs.GetPairs(asset.CoinMarginedFutures, false)
	if err != nil {
		t.Error(err)
	}
	err = h.CurrencyPairs.EnablePair(asset.CoinMarginedFutures, pairs[0])
	if err != nil && !errors.Is(err, currency.ErrPairAlreadyEnabled) {
		t.Error(err)
	}
	_, err = h.GetHistoricCandles(context.Background(), pairs[0], asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Error(err)
	}
	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err = h.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	if !errors.Is(err, common.ErrFunctionNotSupported) {
		t.Error(err)
	}

	pairs, err := h.CurrencyPairs.GetPairs(asset.Futures, false)
	if err != nil {
		t.Error(err)
	}
	err = h.CurrencyPairs.EnablePair(asset.Futures, pairs[0])
	if err != nil && !errors.Is(err, currency.ErrPairAlreadyEnabled) {
		t.Error(err)
	}
	_, err = h.GetHistoricCandlesExtended(context.Background(), pairs[0], asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	if err != nil {
		t.Error(err)
	}

	// demonstrate that adjusting time doesn't wreck non-day intervals
	_, err = h.GetHistoricCandlesExtended(context.Background(), pairs[0], asset.Futures, kline.OneHour, endTime.AddDate(0, 0, -1), endTime)
	if err != nil {
		t.Error(err)
	}

	pairs, err = h.CurrencyPairs.GetPairs(asset.CoinMarginedFutures, false)
	if err != nil {
		t.Error(err)
	}
	err = h.CurrencyPairs.EnablePair(asset.CoinMarginedFutures, pairs[0])
	if err != nil && !errors.Is(err, currency.ErrPairAlreadyEnabled) {
		t.Error(err)
	}
	_, err = h.GetHistoricCandlesExtended(context.Background(), pairs[0], asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), time.Now())
	if err != nil {
		t.Error(err)
	}

	_, err = h.GetHistoricCandlesExtended(context.Background(), pairs[0], asset.CoinMarginedFutures, kline.OneHour, endTime.AddDate(0, 0, -1), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketDetailMerged(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetMarketDetailMerged(context.Background(), cp)
	if err != nil {
		t.Errorf("Huobi TestGetMarketDetailMerged: %s", err)
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetDepth(context.Background(),
		&OrderBookDataRequestParams{
			Symbol: cp,
			Type:   OrderBookDataRequestParamsTypeStep1,
		})
	if err != nil {
		t.Errorf("Huobi TestGetDepth: %s", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetTrades(context.Background(), cp)
	if err != nil {
		t.Errorf("Huobi TestGetTrades: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetLatestSpotPrice(context.Background(), cp)
	if err != nil {
		t.Errorf("Huobi GetLatestSpotPrice: %s", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetTradeHistory(context.Background(), cp, 50)
	if err != nil {
		t.Errorf("Huobi TestGetTradeHistory: %s", err)
	}
}

func TestGetMarketDetail(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetMarketDetail(context.Background(), cp)
	if err != nil {
		t.Errorf("Huobi TestGetTradeHistory: %s", err)
	}
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := h.GetSymbols(context.Background())
	if err != nil {
		t.Errorf("Huobi TestGetSymbols: %s", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := h.GetCurrencies(context.Background())
	if err != nil {
		t.Errorf("Huobi TestGetCurrencies: %s", err)
	}
}

func TestGet24HrMarketSummary(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("ethusdt")
	if err != nil {
		t.Error(err)
	}
	_, err = h.Get24HrMarketSummary(context.Background(), cp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := h.GetTickers(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	st, err := h.GetCurrentServerTime(context.Background())
	if err != nil {
		t.Errorf("Huobi TestGetTimestamp: %s", err)
	}

	if st.IsZero() {
		t.Error("expected a time")
	}
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := h.GetServerTime(context.Background(), asset.Spot)
	if !errors.Is(err, nil) {
		t.Errorf("received: '%v' but expected: '%v'", err, nil)
	}

	if st.IsZero() {
		t.Error("expected a time")
	}
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.GetAccounts(context.Background())
	if err != nil {
		t.Errorf("Huobi GetAccounts: %s", err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	result, err := h.GetAccounts(context.Background())
	if err != nil {
		t.Errorf("Huobi GetAccounts: %s", err)
	}

	userID := strconv.FormatInt(result[0].ID, 10)
	_, err = h.GetAccountBalance(context.Background(), userID)
	if err != nil {
		t.Errorf("Huobi GetAccountBalance: %s", err)
	}
}

func TestGetAggregatedBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.GetAggregatedBalance(context.Background())
	if err != nil {
		t.Errorf("Huobi GetAggregatedBalance: %s", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	cp, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	arg := SpotNewOrderRequestParams{
		Symbol:    cp,
		AccountID: 1997024,
		Amount:    0.01,
		Price:     10.1,
		Type:      SpotNewOrderRequestTypeBuyLimit,
	}

	_, err = h.SpotNewOrder(context.Background(), &arg)
	if err != nil {
		t.Errorf("Huobi SpotNewOrder: %s", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.CancelExistingOrder(context.Background(), 1337)
	if err == nil {
		t.Error("Huobi TestCancelExistingOrder Expected error")
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.GetOrder(context.Background(), 1337)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginLoanOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetMarginLoanOrders(context.Background(),
		cp, "", "", "", "", "", "", "")
	if err != nil {
		t.Errorf("Huobi TestGetMarginLoanOrders: %s", err)
	}
}

func TestGetMarginAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	cp, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetMarginAccountBalance(context.Background(), cp)
	if err != nil {
		t.Errorf("Huobi TestGetMarginAccountBalance: %s", err)
	}
}

func TestCancelWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	_, err := h.CancelWithdraw(context.Background(), 1337)
	if err == nil {
		t.Error("Huobi TestCancelWithdraw Expected error")
	}
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
	if err != nil {
		t.Error(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(h) {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if _, err := h.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if _, err := h.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if _, err := h.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if _, err := h.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := h.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := h.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := h.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := h.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := h.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := h.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
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
	if sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)

	accounts, err := h.GetAccounts(context.Background())
	if err != nil {
		t.Errorf("Failed to get accounts. Err: %s", err)
	}

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
	if sharedtestvalues.AreAPICredentialsSet(h) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := h.CancelOrder(context.Background(), orderCancellation)

	if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(h) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
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
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !sharedtestvalues.AreAPICredentialsSet(h) {
		_, err := h.UpdateAccountInfo(context.Background(),
			asset.CoinMarginedFutures)
		if err == nil {
			t.Error("GetAccountInfo() Expected error")
		}
		_, err = h.UpdateAccountInfo(context.Background(), asset.Futures)
		if err == nil {
			t.Error("GetAccountInfo() Expected error")
		}
	} else {
		_, err := h.UpdateAccountInfo(context.Background(),
			asset.CoinMarginedFutures)
		if err != nil {
			// Spot and Futures have separate api keys. Please ensure that the correct keys are provided
			t.Error(err)
		}
		_, err = h.UpdateAccountInfo(context.Background(), asset.Futures)
		if err != nil {
			// Spot and Futures have separate api keys. Please ensure that the correct keys are provided
			t.Error(err)
		}
	}
}

func TestGetSpotAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		// Spot and Futures have separate api keys. Please ensure that the correct keys are provided
		t.Error(err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	_, err := h.ModifyOrder(context.Background(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
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

	_, err := h.WithdrawCryptocurrencyFunds(context.Background(),
		&withdrawCryptoRequest)
	if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(h) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	var withdrawFiatRequest = withdraw.Request{}
	_, err := h.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	var withdrawFiatRequest = withdraw.Request{}
	_, err := h.WithdrawFiatFundsToInternationalBank(context.Background(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestQueryDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := h.QueryDepositAddress(context.Background(), currency.USDT)
	if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(h) && err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := h.GetDepositAddress(context.Background(), currency.USDT, "", "uSdTeRc20")
	if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(h) && err != nil {
		t.Error(err)
	}
}

func TestQueryWithdrawQuota(t *testing.T) {
	t.Parallel()

	_, err := h.QueryWithdrawQuotas(context.Background(),
		currency.BTC.Lower().String())
	if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(h) && err != nil {
		t.Error(err)
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
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsUnsubscribe(t *testing.T) {
	pressXToJSON := []byte(`{
  "id": "id4",
  "status": "ok",
  "unsubbed": "market.btcusdt.trade.detail",
  "ts": 1494326028889
}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Error(err)
	}
}

func TestWsMarketDepth(t *testing.T) {
	pressXToJSON := []byte(`{
  "ch": "market.htusdt.depth.step0",
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
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsBestBidOffer(t *testing.T) {
	pressXToJSON := []byte(`{
	  "ch": "market.btcusdt.bbo",
	  "ts": 1489474082831,
	  "tick": {
		"symbol": "btcusdt",
		"quoteTime": "1489474082811",
		"bid": "10008.31",
		"bidSize": "0.01",
		"ask": "10009.54",
		"askSize": "0.3"
	  }
	}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTradeDetail(t *testing.T) {
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
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Error(err)
	}
}

func TestWsSubsbOp(t *testing.T) {
	pressXToJSON := []byte(`{
	  "op": "unsub",
	  "topic": "accounts",
	  "cid": "123"
	}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = []byte(`{
	  "op": "sub",
	  "cid": "123",
	  "err-code": 0,
	  "ts": 1489474081631,
	  "topic": "accounts"
	}`)
	err = h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsMarketByPrice(t *testing.T) {
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
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Error(err)
	}
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
	testCases := []struct {
		name     string
		interval kline.Interval
		output   string
	}{
		{
			"OneMin",
			kline.OneMin,
			"1min",
		},
		{
			"FourHour",
			kline.FourHour,
			"4hour",
		},
		{
			"OneDay",
			kline.OneDay,
			"1day",
		},
		{
			"OneWeek",
			kline.OneWeek,
			"1week",
		},
		{
			"OneMonth",
			kline.OneMonth,
			"1mon",
		},
		{
			"OneYear",
			kline.OneYear,
			"1year",
		},
		{
			"AllOthers",
			kline.TwoWeek,
			"",
		},
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			ret := h.FormatExchangeKlineInterval(test.interval)

			if ret != test.output {
				t.Errorf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	fPairs, err := h.CurrencyPairs.GetPairs(asset.Futures, false)
	if err != nil {
		t.Error(err)
	}
	currencyPair, err = fPairs.GetRandomPair()
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetRecentTrades(context.Background(), currencyPair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	currencyPair, err = currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetRecentTrades(context.Background(), currencyPair, asset.CoinMarginedFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Error(err)
	}
	_, err = h.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	r, err := h.GetAvailableTransferChains(context.Background(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
	if len(r) < 2 {
		t.Error("expected more than one result")
	}
}

func TestFormatFuturesPair(t *testing.T) {
	r, err := h.formatFuturesPair(futuresTestPair)
	if err != nil {
		t.Error(err)
	}
	if r != "BTC_CW" {
		t.Errorf("expected BTC_CW, got %s", r)
	}
	availInstruments, err := h.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}
	if len(availInstruments) == 0 {
		t.Error("expected instruments, got 0")
	}
	// test getting a tradable pair in the format of BTC210827 but make it lower
	// case to test correct formatting
	r, err = h.formatFuturesPair(availInstruments[0])
	if err != nil {
		t.Error(err)
	}

	// Test for upper case 'BTC' not lower case 'btc', disregarded numerals
	// as they not deterministic from this endpoint.
	if !strings.Contains(r, "BTC") {
		t.Errorf("expected %s, got %s", "BTC220708", r)
	}
}

func TestSearchForExistedWithdrawsAndDeposits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.SearchForExistedWithdrawsAndDeposits(context.Background(), currency.BTC, "deposit", "", 0, 100)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrderBatch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h, canManipulateRealOrders)
	_, err := h.CancelOrderBatch(context.Background(), []string{"1234"}, nil)
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}
