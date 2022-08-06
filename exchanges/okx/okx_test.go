package okx

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
)

var ok Okx

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	exchCfg, err := cfg.GetExchangeConfig("okx")
	if err != nil {
		log.Fatal(err)
	}
	ok.SkipAuthCheck = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.ClientID = passphrase
	ok.SetDefaults()

	if apiKey != "" && apiSecret != "" {
		exchCfg.API.AuthenticatedSupport = true
		exchCfg.API.AuthenticatedWebsocketSupport = true
	}

	ok.Websocket = sharedtestvalues.NewTestWebsocket()
	err = ok.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	ok.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	ok.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := ok.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = ok.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func areTestAPIKeysSet() bool {
	return ok.ValidateAPICredentials(ok.GetDefaultCredentials()) == nil
}

var marketDataResponseJSON = `{
	"instType": "SWAP",
	"instId": "LTC-USD-SWAP",
	"last": "9999.99",
	"lastSz": "0.1",
	"askPx": "9999.99",
	"askSz": "11",
	"bidPx": "8888.88",
	"bidSz": "5",
	"open24h": "9000",
	"high24h": "10000",
	"low24h": "8888.88",
	"volCcy24h": "2222",
	"vol24h": "2222",
	"sodUtc0": "2222",
	"sodUtc8": "2222",
	"ts": "1597026383085"
  }`

func TestGetTickers(t *testing.T) {
	t.Parallel()
	var resp TickerResponse
	if er := json.Unmarshal([]byte(marketDataResponseJSON), &resp); er != nil {
		t.Error("Okx decerializing to MarketDataResponse error", er)
	}
	_, er := ok.GetTickers(context.Background(), "OPTION", "", "SOL-USD")
	if er != nil {
		t.Error("Okx GetTickers() error", er)
	}
}

func TestGetIndexTicker(t *testing.T) {
	t.Parallel()
	_, er := ok.GetIndexTickers(context.Background(), "USDT", "")
	if er != nil {
		t.Error("OKX GetIndexTicker() error", er)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetTicker(context.Background(), "NEAR-USDT-SWAP"); er != nil {
		t.Error("Okx GetTicker() error", er)
	}
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	instrumentID, er := ok.GetInstrumentIDFromPair(currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	if er != nil {
		t.Error("Okx GetInstrumentIDFromPair() error", er)
	}
	_, er = ok.GetOrderBookDepth(context.Background(), instrumentID, 10)
	if er != nil {
		t.Error("OKX GetOrderBookDepth() error", er)
	}
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, er := ok.GetCandlesticks(context.Background(), "BTC-USDT", kline.OneHour, time.Unix(time.Now().Unix()-3600, 0), time.Now(), 30)
	if er != nil {
		t.Error("Okx GetCandlesticks() error", er)
	}
}

func TestGetCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, er := ok.GetCandlesticksHistory(context.Background(), "BTC-USDT", kline.OneHour, time.Unix(time.Now().Unix()-3600, 0), time.Now(), 30)
	if er != nil {
		t.Error("Okx GetCandlesticksHistory() error", er)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, er := ok.GetTrades(context.Background(), "BTC-USDT", 30)
	if er != nil {
		t.Error("Okx GetTrades() error", er)
	}
}

var tradeHistoryJson = `{
	"instId": "BTC-USDT",
	"side": "sell",
	"sz": "0.00001",
	"px": "29963.2",
	"tradeId": "242720720",
	"ts": "1654161646974"
}`

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	var resp TradeResponse
	if er := json.Unmarshal([]byte(tradeHistoryJson), &resp); er != nil {
		t.Error("Okx decerializing to TradeResponse struct error", er)
	}
	if _, er := ok.GetTradesHistory(context.Background(), "BTC-USDT", "", "", 0); er != nil {
		t.Error("Okx GetTradeHistory() error", er)
	}
}

func TestGet24HTotalVolume(t *testing.T) {
	t.Parallel()
	_, er := ok.Get24HTotalVolume(context.Background())
	if er != nil {
		t.Error("Okx Get24HTotalVolume() error", er)
	}
}

func TestGetOracle(t *testing.T) {
	t.Parallel()
	_, er := ok.GetOracle(context.Background())
	if er != nil {
		t.Error("Okx GetOracle() error", er)
	}
}

func TestGetExchangeRate(t *testing.T) {
	t.Parallel()
	_, er := ok.GetExchangeRate(context.Background())
	if er != nil {
		t.Error("Okx GetExchangeRate() error", er)
	}
}

func TestGetIndexComponents(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	instID, er := ok.GetInstrumentIDFromPair(currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	if er != nil {
		t.Error("Okx GetInstrumentIDFromPair() error", er)
	}
	_, er = ok.GetIndexComponents(context.Background(), instID)
	if er != nil {
		t.Error("Okx GetIndexComponents() error", er)
	}
}

var blockTickerItemJson = `{
	"instType":"SWAP",
	"instId":"LTC-USD-SWAP",
	"volCcy24h":"2222",
	"vol24h":"2222",
	"ts":"1597026383085"
 }`

func TestGetBlockTickers(t *testing.T) {
	t.Parallel()
	var resp BlockTicker
	if er := json.Unmarshal([]byte(blockTickerItemJson), &resp); er != nil {
		t.Error("Okx Decerializing to BlockTickerItem error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetBlockTickers(context.Background(), "SWAP", ""); er != nil {
		t.Error("Okx GetBlockTickers() error", er)
	}
}

func TestGetBlockTicker(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() {
		t.SkipNow()
	}

	if _, er := ok.GetBlockTicker(context.Background(), "BTC-USDT"); er != nil {
		t.Error("Okx GetBlockTicker() error", er)
	}
}

var blockTradeItemJson = `{
	"instId":"BTC-USDT-SWAP",
	"tradeId":"90167",
	"px":"42000",
	"sz":"100",
	"side":"sell",
	"ts":"1642670926504"
}`

func TestGetBlockTrade(t *testing.T) {
	t.Parallel()
	var resp BlockTrade
	if er := json.Unmarshal([]byte(blockTradeItemJson), &resp); er != nil {
		t.Error("Okx Decerializing to BlockTrade error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetBlockTrades(context.Background(), "BTC-USDT"); er != nil {
		t.Error("Okx GetBlockTrades() error", er)
	}
}

func TestGetInstrument(t *testing.T) {
	t.Parallel()
	_, er := ok.GetInstruments(context.Background(), &InstrumentsFetchParams{
		InstrumentType: "MARGIN",
	})
	if er != nil {
		t.Error("Okx GetInstruments() error", er)
	}
	instruments, er := ok.GetInstruments(context.Background(), &InstrumentsFetchParams{
		InstrumentType: "OPTION",
		Underlying:     "BTC-USD",
	})
	for x := range instruments {
		print(instruments[x].InstrumentID, " , ")
	}

	if er != nil {
		t.Error("Okx GetInstruments() error", er)
	}
}

var deliveryHistoryData = `{
    "code":"0",
    "msg":"",
    "data":[
        {
            "ts":"1597026383085",
            "details":[
                {
                    "type":"delivery",
                    "instId":"ZIL-BTC",
                    "px":"0.016"
                }
            ]
        },
        {
            "ts":"1597026383085",
            "details":[
                {
                    "instId":"BTC-USD-200529-6000-C",
                    "type":"exercised",
                    "px":"0.016"
                },
                {
                    "instId":"BTC-USD-200529-8000-C",
                    "type":"exercised",
                    "px":"0.016"
                }
            ]
        }
    ]
}`

func TestGetDeliveryHistory(t *testing.T) {
	t.Parallel()
	var repo DeliveryHistoryResponse
	if err := json.Unmarshal([]byte(deliveryHistoryData), &repo); err != nil {
		t.Error("Okx error", err)
	}
	_, er := ok.GetDeliveryHistory(context.Background(), "FUTURES", "BTC-USDT", time.Time{}, time.Time{}, 100)
	if er != nil {
		t.Error("okx GetDeliveryHistory() error", er)
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetOpenInterest(context.Background(), "FUTURES", "BTC-USDT", ""); er != nil {
		t.Error("Okx GetOpenInterest() error", er)
	}
}

func TestGetFundingRate(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetFundingRate(context.Background(), "BTC-USD-SWAP"); er != nil {
		t.Error("okx GetFundingRate() error", er)
	}
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetFundingRateHistory(context.Background(), "BTC-USD-SWAP", time.Time{}, time.Time{}, 10); er != nil {
		t.Error("Okx GetFundingRateHistory() error", er)
	}
}

func TestGetLimitPrice(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetLimitPrice(context.Background(), "BTC-USD-SWAP"); er != nil {
		t.Error("okx GetLimitPrice() error", er)
	}
}

func TestGetOptionMarketData(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetOptionMarketData(context.Background(), "BTC-USD", time.Time{}); er != nil {
		t.Error("Okx GetOptionMarketData() error", er)
	}
}

var estimatedDeliveryResponseString = `{
    "code":"0",
    "msg":"",
    "data":[
    {
        "instType":"FUTURES",
        "instId":"BTC-USDT-201227",
        "settlePx":"200",
        "ts":"1597026383085"
    }
  ]
}`

func TestGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	var result DeliveryEstimatedPriceResponse
	er := json.Unmarshal([]byte(estimatedDeliveryResponseString), (&result))
	if er != nil {
		t.Error("Okx GetEstimatedDeliveryPrice() error", er)
	}
	if _, er := ok.GetEstimatedDeliveryPrice(context.Background(), "BTC-USD"); er != nil && !(strings.Contains(er.Error(), "Instrument ID does not exist.")) {
		t.Error("Okx GetEstimatedDeliveryPrice() error", er)
	}
}

func TestGetDiscountRateAndInterestFreeQuota(t *testing.T) {
	t.Parallel()
	_, er := ok.GetDiscountRateAndInterestFreeQuota(context.Background(), "BTC", 0)
	if er != nil {
		t.Error("Okx GetDiscountRateAndInterestFreeQuota() error", er)
	}
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetSystemTime(context.Background()); er != nil {
		t.Error("Okx GetSystemTime() error", er)
	}
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetLiquidationOrders(context.Background(), &LiquidationOrderRequestParams{
		InstrumentType: "MARGIN",
		Underlying:     "BTC-USD",
		Currency:       currency.BTC,
	}); er != nil {
		t.Error("Okx GetLiquidationOrders() error", er)
	}
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetMarkPrice(context.Background(), "MARGIN", "", ""); er != nil {
		t.Error("Okx GetMarkPrice() error", er)
	}
}

func TestGetPositionTiers(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetPositionTiers(context.Background(), "FUTURES", "cross", "BTC-USDT", "", ""); er != nil {
		t.Error("Okx GetPositionTiers() error", er)
	}
}

func TestGetInterestRateAndLoanQuota(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetInterestRateAndLoanQuota(context.Background()); er != nil {
		t.Error("Okx GetInterestRateAndLoanQuota() error", er)
	}
}

func TestGetInterestRateAndLoanQuotaForVIPLoans(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetInterestRateAndLoanQuotaForVIPLoans(context.Background()); er != nil {
		t.Error("Okx GetInterestRateAndLoanQuotaForVIPLoans() error", er)
	}
}

func TestGetPublicUnderlyings(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetPublicUnderlyings(context.Background(), "swap"); er != nil {
		t.Error("Okx GetPublicUnderlyings() error", er)
	}
}

func TestGetInsuranceFundInformations(t *testing.T) {
	t.Parallel()
	// getting the Underlyings usig the Get public Underlyinggs method for specific instrument type.
	var underlyings []string
	var er error
	if underlyings, er = ok.GetPublicUnderlyings(context.Background(), "futures"); er != nil {
		t.Error("Okx GetPublicUnderlyings() error", er)
		t.SkipNow()
	}
	if _, er := ok.GetInsuranceFundInformations(context.Background(), InsuranceFundInformationRequestParams{
		InstrumentType: "FUTURES",
		Underlying:     underlyings[0],
	}); er != nil {
		t.Error("Okx GetInsuranceFundInformations() error", er)
	}
}

var currencyConvertJson = `{
	"instId": "BTC-USD-SWAP",
	"px": "35000",
	"sz": "311",
	"type": "1",
	"unit": "coin"
}`

func TestCurrencyUnitConvert(t *testing.T) {
	t.Parallel()
	var resp UnitConvertResponse
	if er := json.Unmarshal([]byte(currencyConvertJson), &resp); er != nil {
		t.Error("Okx Decerializing to UnitConvertResponse error", er)
	}
	if _, er := ok.CurrencyUnitConvert(context.Background(), "BTC-USD-SWAP", 1, 3500, CurrencyToContract, ""); er != nil {
		t.Error("Okx CurrencyUnitConvert() error", er)
	}
}

// Trading related enndpoints test functions.
func TestGetSupportCoins(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetSupportCoins(context.Background()); er != nil {
		t.Error("Okx GetSupportCoins() error", er)
	}
}

func TestGetTakerVolume(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetTakerVolume(context.Background(), "BTC", "SPOT", time.Time{}, time.Time{}, kline.OneDay); er != nil {
		t.Error("Okx GetTakerVolume() error", er)
	}
}
func TestGetMarginLendingRatio(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetMarginLendingRatio(context.Background(), "BTC", time.Time{}, time.Time{}, kline.OneDay); er != nil {
		t.Error("Okx GetMarginLendingRatio() error", er)
	}
}

func TestGetLongShortRatio(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetLongShortRatio(context.Background(), "BTC", time.Time{}, time.Time{}, kline.OneDay); er != nil {
		t.Error("Okx GetLongShortRatio() error", er)
	}
}

func TestGetContractsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetContractsOpenInterestAndVolume(context.Background(), "BTC", time.Time{}, time.Time{}, kline.OneDay); er != nil {
		t.Error("Okx GetContractsOpenInterestAndVolume() error", er)
	}
}

func TestGetOptionsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetOptionsOpenInterestAndVolume(context.Background(), "BTC", kline.OneDay); er != nil {
		t.Error("Okx GetOptionsOpenInterestAndVolume() error", er)
	}
}

func TestGetPutCallRatio(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetPutCallRatio(context.Background(), "BTC", kline.OneDay); er != nil {
		t.Error("Okx GetPutCallRatio() error", er)
	}
}

func TestGetOpenInterestAndVolumeExpiry(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetOpenInterestAndVolumeExpiry(context.Background(), "BTC", kline.OneDay); er != nil {
		t.Error("Okx GetOpenInterestAndVolume() error", er)
	}
}

func TestGetOpenInterestAndVolumeStrike(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetOpenInterestAndVolumeStrike(context.Background(), "BTC", time.Now(), kline.OneDay); er != nil {
		t.Error("Okx GetOpenInterestAndVolumeStrike() error", er)
	}
}

func TestGetTakerFlow(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetTakerFlow(context.Background(), "BTC", kline.OneDay); er != nil {
		t.Error("Okx GetTakerFlow() error", er)
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.PlaceOrder(context.Background(), PlaceOrderRequestParam{
		InstrumentID:        "MATIC-USDC",
		TradeMode:           "cross",
		Side:                "sell",
		OrderType:           "optimal_limit_ioc",
		QuantityToBuyOrSell: 1,
		OrderPrice:          1,
	}); er != nil {
		t.Error("Okx PlaceOrder() error", er)
	}
}

func TestPlaceMultipleOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.PlaceMultipleOrders(context.Background(),
		[]PlaceOrderRequestParam{
			{
				InstrumentID:        "GNX-BTC",
				TradeMode:           "cross",
				Side:                "sell",
				OrderType:           "limit",
				QuantityToBuyOrSell: 1,
				OrderPrice:          1,
			},
		}); er != nil {
		t.Error("Okx PlaceOrderRequestParam() error", er)
	}
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.CancelSingleOrder(context.Background(),
		CancelOrderRequestParam{
			InstrumentID: "BTC-USD-190927",
			OrderID:      "2510789768709120",
		}); er != nil {
		t.Error("Okx CancelOrder() error", er)
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.CancelMultipleOrders(context.Background(), []CancelOrderRequestParam{{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
	}}); er != nil {
		t.Error("Okx CancelMultipleOrders() error", er)
	}
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.AmendOrder(context.Background(), &AmendOrderRequestParams{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	}); er != nil {
		t.Error("Okx AmendOrder() error", er)
	}
}
func TestAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.AmendMultipleOrders(context.Background(), []AmendOrderRequestParams{{
		InstrumentID: "BTC-USDT",
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	}}); er != nil {
		t.Error("Okx AmendMultipleOrders() error", er)
	}
}

func TestClosePositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.ClosePositions(context.Background(), &ClosePositionsRequestParams{
		InstrumentID: "BTC-USDT",
		MarginMode:   "cross",
	}); er != nil && !strings.Contains(er.Error(), "Operation is not supported under the current account mode") {
		t.Error("Okc ClosePositions() error", er)
	}
}

var orderDetail = `{"instType": "FUTURES","instId": "BTC-USD-200329","ccy": "","ordId": "312269865356374016","clOrdId": "b1","tag": "","px": "999","sz": "3","pnl": "5","ordType": "limit","side": "buy","posSide": "long","tdMode": "isolated","accFillSz": "0","fillPx": "0","tradeId": "0","fillSz": "0","fillTime": "0","state": "live","avgPx": "0","lever": "20","tpTriggerPx": "","tpTriggerPxType": "last","tpOrdPx": "","slTriggerPx": "","slTriggerPxType": "last","slOrdPx": "","feeCcy": "","fee": "","rebateCcy": "","rebate": "","tgtCcy":"","category": "","uTime": "1597026383085","cTime": "1597026383085"}`

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	var odetail OrderDetail
	if er := json.Unmarshal([]byte(orderDetail), &odetail); er != nil {
		t.Error("Okx OrderDetail error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetOrderDetail(context.Background(), &OrderDetailRequestParam{
		InstrumentID: "BTC-USDT",
		OrderID:      "2510789768709120",
	}); !strings.Contains(er.Error(), "Order does not exist") {
		t.Error("Okx GetOrderDetail() error", er)
	}
}

const pendingOrderItemJSON = `{"accFillSz": "0","avgPx": "","cTime": "1618235248028","category": "normal","ccy": "","clOrdId": "","fee": "0","feeCcy": "BTC","fillPx": "","fillSz": "0","fillTime": "","instId": "BTC-USDT","instType": "SPOT","lever": "5.6","ordId": "301835739059335168","ordType": "limit","pnl": "0","posSide": "net","px": "59200","rebate": "0","rebateCcy": "USDT","side": "buy","slOrdPx": "","slTriggerPx": "","slTriggerPxType": "last","state": "live","sz": "1","tag": "","tgtCcy": "","tdMode": "cross","source":"","tpOrdPx": "","tpTriggerPx": "","tpTriggerPxType": "last","tradeId": "","uTime": "1618235248028"}`

func TestGetOrderList(t *testing.T) {
	t.Parallel()
	var pending PendingOrderItem
	if er := json.Unmarshal([]byte(pendingOrderItemJSON), &pending); er != nil {
		t.Error("Okx PendingPrderItem error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetOrderList(context.Background(), &OrderListRequestParams{}); er != nil {
		t.Error("Okx GetOrderList() error", er)
	}
}

var orderHistory = `{
	"instType": "FUTURES",
	"instId": "BTC-USD-200329",
	"ccy": "",
	"ordId": "312269865356374016",
	"clOrdId": "b1",
	"tag": "",
	"px": "999",
	"sz": "3",
	"ordType": "limit",
	"side": "buy",
	"posSide": "long",
	"tdMode": "isolated",
	"accFillSz": "0",
	"fillPx": "0",
	"tradeId": "0",
	"fillSz": "0",
	"fillTime": "0",
	"state": "filled",
	"avgPx": "0",
	"lever": "20",
	"tpTriggerPx": "",
	"tpTriggerPxType": "last",
	"tpOrdPx": "",
	"slTriggerPx": "",
	"slTriggerPxType": "last",
	"slOrdPx": "",
	"feeCcy": "",
	"fee": "",
	"rebateCcy": "",
	"source":"",
	"rebate": "",
	"tgtCcy":"",
	"pnl": "",
	"category": "",
	"uTime": "1597026383085",
	"cTime": "1597026383085"
  }`

func TestGet7And3MonthDayOrderHistory(t *testing.T) {
	t.Parallel()
	var history PendingOrderItem
	if er := json.Unmarshal([]byte(orderHistory), &history); er != nil {
		t.Error("Okx OrderHistory error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.Get7DayOrderHistory(context.Background(), &OrderHistoryRequestParams{
		OrderListRequestParams: OrderListRequestParams{InstrumentType: "MARGIN"},
	}); er != nil {
		t.Error("Okx Get7DayOrderHistory() error", er)
	}
	if _, er := ok.Get3MonthOrderHistory(context.Background(), &OrderHistoryRequestParams{
		OrderListRequestParams: OrderListRequestParams{InstrumentType: "MARGIN"},
	}); er != nil {
		t.Error("Okx Get3MonthOrderHistory() error", er)
	}
}

var transactionhistoryJSON = `{
	"instType":"FUTURES",
	"instId":"BTC-USD-200329",
	"tradeId":"123",
	"ordId":"123445",
	"clOrdId": "b16",
	"billId":"1111",
	"tag":"",
	"fillPx":"999",
	"fillSz":"3",
	"side":"buy",
	"posSide":"long",
	"execType":"M",             
	"feeCcy":"",
	"fee":"",
	"ts":"1597026383085"
}`

func TestTransactionHistory(t *testing.T) {
	t.Parallel()
	var transactionhist TransactionDetail
	if er := json.Unmarshal([]byte(transactionhistoryJSON), &transactionhist); er != nil {
		t.Error("Okx Transaction Detail error", er.Error())
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetTransactionDetailsLast3Days(context.Background(), &TransactionDetailRequestParams{
		InstrumentType: "MARGIN",
	}); er != nil {
		t.Error("Okx GetTransactionDetailsLast3Days() error", er)
	}
	if _, er := ok.GetTransactionDetailsLast3Months(context.Background(), &TransactionDetailRequestParams{
		InstrumentType: "MARGIN",
	}); er != nil {
		t.Error("Okx GetTransactionDetailsLast3Days() error", er)
	}
}
func TestStopOrder(t *testing.T) {
	t.Parallel()
	t.SkipNow()
	// this test function had to be re modified.
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.StopOrderParams(context.Background(), &StopOrderParams{
		TakeProfitTriggerPriceType: "index",
	}); !strings.Contains(er.Error(), "Unsupported operation") {
		t.Error("Okx StopOrderParams() error", er)
	}
	if _, er := ok.PlaceTrailingStopOrder(context.Background(), &TrailingStopOrderRequestParam{
		CallbackRatio: 0.01,
	}); er != nil {
		t.Error("Okx PlaceTrailingStopOrder error", er)
	}
	if _, er := ok.PlaceIceburgOrder(context.Background(), &IceburgOrder{
		PriceLimit:    100.22,
		AverageAmount: 9999.9,
		PriceRatio:    "0.04",
	}); er != nil {
		t.Error("Okx PlaceIceburgOrder() error", er)
	}
	if _, er := ok.PlaceTWAPOrder(context.Background(), &TWAPOrderRequestParams{
		PriceLimit:    100.22,
		AverageAmount: 9999.9,
		PriceRatio:    "0.4",
		Timeinterval:  kline.ThreeDay,
	}); er != nil {
		t.Error("Okx PlaceTWAPOrder() error", er)
	}
	if _, er := ok.TriggerAlogOrder(context.Background(), &TriggerAlogOrderParams{
		TriggerPriceType: "mark",
	}); er != nil {
		t.Error("Okx TriggerAlogOrder() error", er)
	}
}

func TestCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.CancelAlgoOrder(context.Background(), []AlgoOrderCancelParams{
		{
			InstrumentID: "BTC-USDT",
			AlgoOrderID:  "90994943",
		},
	}); er != nil {
		t.Error("Okx CancelAlgoOrder() error", er)
	}
}

func TestCancelAdvanceAlgoOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.CancelAdvanceAlgoOrder(context.Background(), []AlgoOrderCancelParams{{
		InstrumentID: "BTC-USDT",
		AlgoOrderID:  "90994943",
	}}); er != nil {
		t.Error("Okx CancelAdvanceAlgoOrder() error", er)
	}
}

var algoOrderResponse = `{
	"instType": "FUTURES",
	"instId": "BTC-USD-200329",
	"ordId": "312269865356374016",
	"ccy": "BTC",
	"algoId": "1234",
	"sz": "999",
	"ordType": "oco",
	"side": "buy",
	"posSide": "long",
	"tdMode": "cross",
	"tgtCcy": "",
	"state": "1",
	"lever": "20",
	"tpTriggerPx": "",
	"tpTriggerPxType": "",
	"tpOrdPx": "",
	"slTriggerPx": "",
	"slTriggerPxType": "",
	"triggerPx": "99",
	"triggerPxType": "last",
	"ordPx": "12",
	"actualSz": "",
	"actualPx": "",
	"actualSide": "",
	"pxVar":"",
	"pxSpread":"",
	"pxLimit":"",
	"szLimit":"",
	"timeInterval":"",
	"triggerTime": "1597026383085",
	"callbackRatio":"",
	"callbackSpread":"",
	"activePx":"",
	"moveTriggerPx":"",
	"cTime": "1597026383000"
  }`

func TestGetAlgoOrderList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	var order AlgoOrderResponse
	if er := json.Unmarshal([]byte(algoOrderResponse), &order); er != nil {
		t.Error("Okx Unmarshaling AlgoOrder Response error", er)
	}
	if _, er := ok.GetAlgoOrderList(context.Background(), "conditional", "", "", "", time.Time{}, time.Time{}, 20); er != nil {
		t.Error("Okx GetAlgoOrderList() error", er)
	}
}

//
func TestGetAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	var order AlgoOrderResponse
	if er := json.Unmarshal([]byte(algoOrderResponse), &order); er != nil {
		t.Error("Okx Unmarshaling AlgoOrder Response error", er)
	}
	if _, er := ok.GetAlgoOrderHistory(context.Background(), "conditional", "effective", "", "", "", time.Time{}, time.Time{}, 20); er != nil {
		t.Error("Okx GetAlgoOrderList() error", er)
	}
}

func TestGetCounterparties(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetCounterparties(context.Background()); er != nil {
		t.Error("Okx GetCounterparties() error", er)
	}
}

var createRFQInputJson = `
{
    "anonymous": true,
    "counterparties":[
        "Trader1",
        "Trader2"
    ],
    "clRfqId":"rfq01",
    "legs":[
        {
            "sz":"25",
            "side":"buy",
            "instId":"BTCUSD-221208-100000-C"
        },
        {
            "sz":"150",
            "side":"buy",
            "instId":"ETH-USDT",
            "tgtCcy":"base_ccy"
        }
    ]
}`
var createRFQOutputJson = `{
	"cTime":"1611033737572",
	"uTime":"1611033737572",
	"traderCode":"SATOSHI",
	"rfqId":"22534",
	"clRfqId":"rfq01",
	"state":"active",
	"validUntil":"1611033857557",
	"counterparties":[
		"Trader1",
		"Trader2"
	],
	"legs":[
		{
			"instId":"BTCUSD-221208-100000-C",
			"sz":"25",
			"side":"buy",
			"tgtCcy":""
		},
		{
			"instId":"ETH-USDT",
			"sz":"150",
			"side":"buy",
			"tgtCcy":"base_ccy"     
		}
	]
}`

func TestCreateRFQ(t *testing.T) {
	t.Parallel()
	var input CreateRFQInput
	if er := json.Unmarshal([]byte(createRFQInputJson), &input); er != nil {
		t.Error("Okx Decerializing to CreateRFQInput", er)
	}
	var resp RFQResponse
	if er := json.Unmarshal([]byte(createRFQOutputJson), &resp); er != nil {
		t.Error("Okx Decerializing to CreateRFQResponse", er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := ok.CreateRFQ(context.Background(), input); er != nil {
		t.Error("Okx CreateRFQ() error", er)
	}
}

func TestCancelRFQ(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	_, er := ok.CancelRFQ(context.Background(), CancelRFQRequestParam{})
	if er != nil && !errors.Is(er, errMissingRFQIDANDClientSuppliedRFQID) {
		t.Errorf("Okx CancelRFQ() expecting %v, but found %v", errMissingRFQIDANDClientSuppliedRFQID, er)
	}
	_, er = ok.CancelRFQ(context.Background(), CancelRFQRequestParam{
		ClientSuppliedRFQID: "somersdjskfjsdkfj",
	})
	if er != nil {
		t.Error("Okx CancelRFQ() error", er)
	}
}

func TestMultipleCancelRFQ(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	_, er := ok.CancelMultipleRFQs(context.Background(), CancelRFQRequestsParam{})
	if er != nil && !errors.Is(er, errMissingRFQIDANDClientSuppliedRFQID) {
		t.Errorf("Okx CancelMultipleRFQs() expecting %v, but found %v", errMissingRFQIDANDClientSuppliedRFQID, er)
	}
	_, er = ok.CancelMultipleRFQs(context.Background(), CancelRFQRequestsParam{
		ClientSuppliedRFQID: []string{"somersdjskfjsdkfj"},
	})
	if er != nil && !strings.Contains(er.Error(), "Either parameter rfqIds or clRfqIds is required") {
		t.Error("Okx CancelMultipleRFQs() error", er)
	}
}

var executeQuoteJson = `{
	"blockTdId":"180184",
	"rfqId":"1419",
	"clRfqId":"r0001",
	"quoteId":"1046",
	"clQuoteId":"q0001",
	"tTraderCode":"Trader1",
	"mTraderCode":"Trader2",
	"cTime":"1649670009",
	"legs":[
		{
			"px":"0.1",
			"sz":"25",
			"instId":"BTC-USD-20220114-13250-C",
			"side":"sell",
			"fee":"-1.001",
			"feeCcy":"BTC",
			"tradeId":"10211"
		},
		{
			"px":"0.2",
			"sz":"25",
			"instId":"BTC-USDT",
			"side":"buy",
			"fee":"-1.001",
			"feeCcy":"BTC",
			"tradeId":"10212"
		}
	]
}`

func TestExecuteQuote(t *testing.T) {
	t.Parallel()
	var resp ExecuteQuoteResponse
	if er := json.Unmarshal([]byte(executeQuoteJson), &resp); er != nil {
		t.Error("Okx Decerialing error", er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := ok.ExecuteQuote(context.Background(), ExecuteQuoteParams{}); er != nil && !errors.Is(er, errMissingRfqIDOrQuoteID) {
		t.Errorf("Okx ExecuteQuote() expected %v, but found %v", errMissingRfqIDOrQuoteID, er)
	}
	if _, er := ok.ExecuteQuote(context.Background(), ExecuteQuoteParams{
		RfqID:   "22540",
		QuoteID: "84073",
	}); er != nil {
		t.Error("Okx ExecuteQuote() error", er)
	}
}

var createQuoteJson = `{
	"cTime":"1611038342698",
	"uTime":"1611038342698",
	"quoteId":"84069", 
	"clQuoteId":"q002",
	"rfqId":"22537",
	"quoteSide":"buy",
	"state":"active",
	"validUntil":"1611038442838",
	"legs":[
			{
				"px":"39450.0",
				"sz":"200000",
				"instId":"BTC-USDT-SWAP",
				"side":"buy",
				"tgtCcy":""
			}            
	]
}`

func TestCreateQuote(t *testing.T) {
	t.Parallel()
	var resp QuoteResponse
	if er := json.Unmarshal([]byte(createQuoteJson), &resp); er != nil {
		t.Error("Okx Decerializing to CreateQuoteResponse error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.CreateQuote(context.Background(), CreateQuoteParams{}); er != nil && !errors.Is(er, errMissingRfqID) {
		t.Errorf("Okx CreateQuote() expecting %v, but found %v", errMissingRfqID, er)
	}
}

func TestCancelQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.CancelQuote(context.Background(), CancelQuoteRequestParams{}); er != nil && !errors.Is(er, errMissingQuoteIDOrClientSuppliedQuoteID) {
		t.Error("Okx CancelQuote() error", er)
	}
}

func TestCancelMultipleQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.CancelMultipleQuote(context.Background(), CancelQuotesRequestParams{}); er != nil && !errors.Is(errMissingEitherQuoteIDAOrlientSuppliedQuoteIDs, er) {
		t.Error("Okx CancelQuote() error", er)
	}
}

func TestCancelAllQuotes(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if time, er := ok.CancelAllQuotes(context.Background()); er != nil && !strings.Contains(er.Error(), "Cancellation failed as you do not have any active Quotes.") {
		t.Error("Okx CancelAllQuotes() error", er)
	} else if er == nil && time.IsZero() {
		t.Error("Okx CancelAllQuotes() zero timestamp message ")
	}
}

func TestGetRFQs(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetRfqs(context.Background(), RfqRequestParams{}); er != nil {
		t.Error("Okx GetRfqs() error", er)
	}
}

func TestGetQuotes(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetQuotes(context.Background(), QuoteRequestParams{}); er != nil {
		t.Error("Okx GetQuotes() error", er)
	}
}

var rfqTradeResponseJson = `{
	"rfqId": "1234567",
	"clRfqId": "",
	"quoteId": "0T533T0",
	"clQuoteId": "",
	"blockTdId": "439121886014849024",
	"legs": [
		{
			"instId": "BTC-USDT",
			"side": "sell",
			"sz": "0.532",
			"px": "100",
			"tradeId": "439121886014849026",
			"fee": "-0.0266",
			"feeCcy": "USDT"
		}
	],
	"cTime": "1650966816550",
	"tTraderCode": "SATS",
	"mTraderCode": "MIKE"
}`

func TestGetRFQTrades(t *testing.T) {
	t.Parallel()
	var resp RfqTradeResponse
	if er := json.Unmarshal([]byte(rfqTradeResponseJson), &resp); er != nil {
		t.Error("Okx Decerializing to RFQTradeResponse error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetRFQTrades(context.Background(), RFQTradesRequestParams{}); er != nil {
		t.Error("Okx GetRFQTrades() error", er)
	}
}

var publicTradesResponseJson = `{
	"blockTdId": "439161457415012352",
	"legs": [
		{
			"instId": "BTC-USD-210826",
			"side": "sell",
			"sz": "100",
			"px": "11000",
			"tradeId": "439161457415012354"
		}
	],
	"cTime": "1650976251241"
}`

func TestGetPublicTrades(t *testing.T) {
	t.Parallel()
	var resp PublicTradesResponse
	if er := json.Unmarshal([]byte(publicTradesResponseJson), &resp); er != nil {
		t.Error("Okx Decerializing to PublicTradesResponse error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetPublicTrades(context.Background(), "", "", 10); er != nil {
		t.Error("Okx GetPublicTrades() error", er)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetCurrencies(context.Background()); er != nil {
		t.Error("Okx  GetCurrencies() error", er)
	}
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetBalance(context.Background(), ""); er != nil {
		t.Error("Okx GetBalance() error", er)
	}
}

func TestGetAccountAssetValuation(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetAccountAssetValuation(context.Background(), ""); er != nil {
		t.Error("Okx  GetAccountAssetValuation() error", er)
	}
}

var fundingTransferRequest = `{
    "ccy":"USDT",
    "type":"4",
    "amt":"1.5",
    "from":"6",
    "to":"6",
    "subAcct":"mini"
}`

var fundingTransferResponseMessage = `{
	"transId": "754147",
	"ccy": "USDT",
	"clientId": "",
	"from": "6",
	"amt": "0.1",
	"to": "18"
  }`

func TestFundingTransfer(t *testing.T) {
	t.Parallel()
	var fundReq FundingTransferRequestInput
	if er := json.Unmarshal([]byte(fundingTransferRequest), &fundReq); er != nil {
		t.Error("Okx FundingTransferRequestInput{} unmarshal  error", er)
	}
	var fundResponse FundingTransferResponse
	if er := json.Unmarshal([]byte(fundingTransferResponseMessage), &fundResponse); er != nil {
		t.Error("okx FundingTransferRequestInput{} unmarshal error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.FundingTransfer(context.Background(), &FundingTransferRequestInput{
		Amount:   12.000,
		To:       "6",
		From:     "6",
		Currency: "BTC",
	}); er != nil {
		t.Error("Okx FundingTransfer() error", er)
	}
}

var fundingRateTransferResponseJSON = `{
	"amt": "1.5",
	"ccy": "USDT",
	"clientId": "",
	"from": "18",
	"instId": "",
	"state": "success",
	"subAcct": "test",
	"to": "6",
	"toInstId": "",
	"transId": "1",
	"type": "1"
}`

func TestGetFundsTransferState(t *testing.T) {
	t.Parallel()
	var transResponse TransferFundRateResponse
	if er := json.Unmarshal([]byte(fundingRateTransferResponseJSON), &transResponse); er != nil {
		t.Error("Okx TransferFundRateResponse{} unmarshal error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetFundsTransferState(context.Background(), "abcdefg", "", 2); er != nil {
		t.Error("Okx GetFundsTransferState() error", er)
	}
}

var assetBillDetailResponse = `{
	"billId": "12344",
	"ccy": "BTC",
	"clientId": "",
	"balChg": "2",
	"bal": "12",
	"type": "1",
	"ts": "1597026383085"
}`

func TestGetAssetBillsDetails(t *testing.T) {
	t.Parallel()
	var response AssetBillDetail
	er := json.Unmarshal([]byte(assetBillDetailResponse), &response)
	if er != nil {
		t.Error("Okx Unmarshaling error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, er = ok.GetAssetBillsDetails(context.Background(), "", 0, "", "", time.Time{}, time.Time{}, 5)
	if er != nil {
		t.Error("Okx GetAssetBillsDetail() error", er)
	}
}

var lightningDepositResponseString = `{
	"cTime": "1631171307612",
	"invoice": "lnbc100u1psnnvhtpp5yq2x3q5hhrzsuxpwx7ptphwzc4k4wk0j3stp0099968m44cyjg9sdqqcqzpgxqzjcsp5hzzdszuv6yv6yw5svctl8kc8uv6y77szv5kma2kuculj86tk3yys9qyyssqd8urqgcensh9l4zwlwr3lxlcdqrlflvvlwldutm6ljx486h7lylqmd06kky6scas7warx69sregzrx20ffmsr4sp865x3wasrjd8ttgqrlx3tr"
}`

func TestGetLightningDeposits(t *testing.T) {
	t.Parallel()
	var response LightningDepositItem
	er := json.Unmarshal([]byte(lightningDepositResponseString), &response)
	if er != nil {
		t.Error("Okx Unamrshaling to LightningDepositItem error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er = ok.GetLightningDeposits(context.Background(), "BTC", 1.00, 0); er != nil {
		t.Error("Okx GetLightningDeposits() error", er)
	}
}

var depositAddressResponseItemString = `{
	"chain": "BTC-OKC",
	"ctAddr": "",
	"ccy": "BTC",
	"to": "6",
	"addr": "0x66d0edc2e63b6b992381ee668fbcb01f20ae0428",
	"selected": true
}`

func TestGetCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	var response CurrencyDepositResponseItem
	er := json.Unmarshal([]byte(depositAddressResponseItemString), &response)
	if er != nil {
		t.Error("Okx unmarshaling to CurrencyDepositResponseItem error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetCurrencyDepositAddress(context.Background(), "BTC"); er != nil {
		t.Error("Okx GetCurrencyDepositAddress() error", er)
	}
}

var depositHistoryResponseString = `{"amt": "0.01044408","txId": "1915737_3_0_0_asset","ccy": "BTC","chain":"BTC-Bitcoin","from": "13801825426","to": "","ts": "1597026383085","state": "2","depId": "4703879"}`

func TestGetCurrencyDepositHistory(t *testing.T) {
	t.Parallel()
	var response DepositHistoryResponseItem
	er := json.Unmarshal([]byte(depositHistoryResponseString), &response)
	if er != nil {
		t.Error("Okx DepositHistoryResponseItem unmarshaling error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetCurrencyDepositHistory(context.Background(), "BTC", "", "", 0, time.Time{}, time.Time{}, 5); er != nil {
		t.Error("Okx GetCurrencyDepositHistory() error", er)
	}
}

var withdrawalResponseString = `{
	"amt": "0.1",
	"wdId": "67485",
	"ccy": "BTC",
	"clientId": "",
	"chain": "BTC-Bitcoin"
}`

func TestWithdrawal(t *testing.T) {
	t.Parallel()
	var response WithdrawalResponse
	er := json.Unmarshal([]byte(withdrawalResponseString), &response)
	if er != nil {
		t.Error("Okx WithdrawalResponse unmarshaling json error", er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	_, er = ok.Withdrawal(context.Background(), WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: "BTC", WithdrawalDestination: "4", ToAddress: "17DKe3kkkkiiiiTvAKKi2vMPbm1Bz3CMKw"})
	if er != nil {
		t.Error("Okx Withdrawal error", er)
	}
}

var lightningWithdrawalResponseJson = `{
	"wdId": "121212",
	"cTime": "1597026383085"
}`

func TestLightningWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	var response LightningWithdrawalResponse
	if er := json.Unmarshal([]byte(lightningWithdrawalResponseJson), &response); er != nil {
		t.Error("Binanceus LightningWithdrawalResponse Json Conversion error ", er)
	}
	_, er := ok.LightningWithdrawal(context.Background(), LightningWithdrawalRequestInput{
		Currency: currency.BTC.String(),
		Invoice:  "lnbc100u1psnnvhtpp5yq2x3q5hhrzsuxpwx7ptphwzc4k4wk0j3stp0099968m44cyjg9sdqqcqzpgxqzjcsp5hz",
	})
	if !strings.Contains(er.Error(), `401 raw response: {"msg":"Invalid Authority","code":"50114"}`) {
		t.Error("Binanceus LightningWithdrawal() error", er)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := ok.CancelWithdrawal(context.Background(), "fjasdfkjasdk"); er != nil {
		t.Error("Okx CancelWithdrawal() error", er.Error())
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetWithdrawalHistory(context.Background(), "BTC", "", "", "", 0, time.Time{}, time.Time{}, 10); er != nil {
		t.Error("Okx GetWithdrawalHistory() error", er)
	}
}

func TestSmallAssetsConvert(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.SmallAssetsConvert(context.Background(), []string{"BTC", "USDT"}); er != nil {
		t.Error("Okx SmallAssetsConvert() error", er)
	}
}

var savingBalanceResponse = `{
	"earnings": "0.0010737388791526",
	"redemptAmt": "0.0000000000000000",
	"rate": "0.0100000000000000",
	"ccy": "USDT",
	"amt": "11.0010737453457821",
	"loanAmt": "11.0010630707982819",
	"pendingAmt": "0.0000106745475002"
}`

func TestGetSavingBalance(t *testing.T) {
	t.Parallel()
	var resp SavingBalanceResponse
	er := json.Unmarshal([]byte(savingBalanceResponse), &resp)
	if er != nil {
		t.Error("Okx Saving Balance Unmarshaling error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetSavingBalance(context.Background(), "BTC"); er != nil {
		t.Error("Okx GetSavingBalance() error", er)
	}
}

var redemptionOrPurchaseSavingJson = `{
	"ccy":"BTC",
	"amt":"1",
	"side":"purchase",
	"rate": "0.01"
}`

func TestSavingsPurchase(t *testing.T) {
	t.Parallel()
	var resp SavingsPurchaseRedemptionResponse
	if er := json.Unmarshal([]byte(redemptionOrPurchaseSavingJson), &resp); er != nil {
		t.Error("Okx Unmarshaling purchase or redemption error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.SavingsPurchase(context.Background(), &SavingsPurchaseRedemptionInput{
		Amount:   123.4,
		Currency: "BTC",
		Rate:     1,
	}); er != nil {
		t.Error("Okx SavingsPurchase() error", er)
	}
	if _, er := ok.SavingsRedemption(context.Background(), &SavingsPurchaseRedemptionInput{
		Amount:   123.4,
		Currency: "BTC",
		Rate:     1,
	}); er != nil {
		t.Error("Okx SavingsPurchase() error", er)
	}
}

var setLendingRate = `{"ccy": "BTC","rate": "0.02"}`

func TestSetLendingRate(t *testing.T) {
	t.Parallel()
	var resp LendingRate
	if er := json.Unmarshal([]byte(setLendingRate), &resp); er != nil {
		t.Error("Okx Unmarshaling LendingRate error", er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := ok.SetLendingRate(context.Background(), LendingRate{Currency: "BTC", Rate: 2}); er != nil &&
		!strings.Contains(er.Error(), "You do not have assets in this currency") {
		t.Error("Okx SetLendingRate() error", er)
	}
}

var lendinghistoryJSON = `{"ccy": "BTC","amt": "0.01","earnings": "0.001","rate": "0.01","ts": "1597026383085"}`

func TestGetLendingHistory(t *testing.T) {
	t.Parallel()
	var res LendingHistory
	if er := json.Unmarshal([]byte(lendinghistoryJSON), &res); er != nil {
		t.Error("Okx Unmarshaling Lending History error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetLendingHistory(context.Background(), "USDT", time.Time{}, time.Time{}, 10); er != nil {
		t.Error("Okx GetLendingHostory() error", er)
	}
}

var publicBorrowInfoJson = `{
	"ccy": "BTC",
	"amt": "0.01",
	"rate": "0.001",
	"ts": "1597026383085"
}`

func TestGetPublicBorrowInfo(t *testing.T) {
	t.Parallel()
	var resp LendingHistory
	if er := json.Unmarshal([]byte(publicBorrowInfoJson), &resp); er != nil {
		t.Error("Okx Unmarshaling to LendingHistory error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetPublicBorrowInfo(context.Background(), ""); er != nil {
		t.Error("Okx GetPublicBorrowInfo() error", er)
	}
}

var convertCurrencyResponseJson = `{
	"min": "0.0001",
	"max": "0.5",
	"ccy": "BTC"
}`

func TestGetConvertCurrencies(t *testing.T) {
	t.Parallel()
	var resp ConvertCurrency
	if er := json.Unmarshal([]byte(convertCurrencyResponseJson), &resp); er != nil {
		t.Error("Okx Unmarshaling Json error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetConvertCurrencies(context.Background()); er != nil {
		t.Error("Okx GetConvertCurrencies() error", er)
	}
}

var convertCurrencyPairResponseJson = `{
	"baseCcy": "BTC",
	"baseCcyMax": "0.5",
	"baseCcyMin": "0.0001",
	"instId": "BTC-USDT",
	"quoteCcy": "USDT",
	"quoteCcyMax": "10000",
	"quoteCcyMin": "1"
}`

func TestGetConvertCurrencyPair(t *testing.T) {
	t.Parallel()
	var resp ConvertCurrencyPair
	if er := json.Unmarshal([]byte(convertCurrencyPairResponseJson), &resp); er != nil {
		t.Error("Okx Unmarshaling ConvertCurrencyPair error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetConvertCurrencyPair(context.Background(), "USDT", "BTC"); er != nil {
		t.Error("Okx GetConvertCurrencyPair() error", er)
	}
}

var estimateQuoteJSON = `{
	"baseCcy": "ETH",
	"baseSz": "0.01023052",
	"clQReqId": "",
	"cnvtPx": "2932.40104429",
	"origRfqSz": "30",
	"quoteCcy": "USDT",
	"quoteId": "quoterETH-USDT16461885104612381",
	"quoteSz": "30",
	"quoteTime": "1646188510461",
	"rfqSz": "30",
	"rfqSzCcy": "USDT",
	"side": "buy",
	"ttlMs": "10000"
}`

func TestEstimateQuote(t *testing.T) {
	t.Parallel()
	var estimate EstimateQuoteResponse
	if er := json.Unmarshal([]byte(estimateQuoteJSON), &estimate); er != nil {
		t.Error("Okx Umarshaling EstimateQuoteResponse error", er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := ok.EstimateQuote(context.Background(), EstimateQuoteRequestInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          "Buy",
		RFQAmount:     30,
		RFQSzCurrency: "USDT",
	}); er != nil {
		t.Error("Okx EstimateQuote() error", er)
	}
}

var convertTradeJsonResponse = `{
	"baseCcy": "ETH",
	"clTReqId": "",
	"fillBaseSz": "0.01023052",
	"fillPx": "2932.40104429",
	"fillQuoteSz": "30",
	"instId": "ETH-USDT",
	"quoteCcy": "USDT",
	"quoteId": "quoterETH-USDT16461885104612381",
	"side": "buy",
	"state": "fullyFilled",
	"tradeId": "trader16461885203381437",
	"ts": "1646188520338"
}`

func TestConvertTrade(t *testing.T) {
	t.Parallel()
	var convert ConvertTradeResponse
	if er := json.Unmarshal([]byte(convertTradeJsonResponse), &convert); er != nil {
		t.Error("Okx Unmarshaling to ConvertTradeResponse error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.ConvertTrade(context.Background(), ConvertTradeInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          "Buy",
		Size:          2,
		SizeCurrency:  "USDT",
		QuoteID:       "quoterETH-USDT16461885104612381",
	}); er != nil && !errors.Is(er, errNoValidResponseFromServer) {
		t.Error("Okx ConvertTrade() error", er)
	}
}

var convertHistoryResponseJson = `{
	"instId": "ETH-USDT",
	"side": "buy",
	"fillPx": "2932.401044",
	"baseCcy": "ETH",
	"quoteCcy": "USDT",
	"fillBaseSz": "0.01023052",
	"state": "fullyFilled",
	"tradeId": "trader16461885203381437",
	"fillQuoteSz": "30",
	"ts": "1646188520000"
}`

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()
	var convertHistory ConvertHistory
	if er := json.Unmarshal([]byte(convertHistoryResponseJson), &convertHistory); er != nil {
		t.Error("Okx Unmarshaling ConvertHistory error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetConvertHistory(context.Background(), time.Time{}, time.Time{}, 10, ""); er != nil {
		t.Error("Okx GetConvertHistory() error", er)
	}
}

var accountBalanceInformation = ` {
	"adjEq": "10679688.0460531643092577",
	"details": [
		{
			"availBal": "",
			"availEq": "9930359.9998",
			"cashBal": "9930359.9998",
			"ccy": "USDT",
			"crossLiab": "0",
			"disEq": "9439737.0772999514",
			"eq": "9930359.9998",
			"eqUsd": "9933041.196999946",
			"frozenBal": "0",
			"interest": "0",
			"isoEq": "0",
			"isoLiab": "0",
			"isoUpl":"0",
			"liab": "0",
			"maxLoan": "10000",
			"mgnRatio": "",
			"notionalLever": "",
			"ordFrozen": "0",
			"twap": "0",
			"uTime": "1620722938250",
			"upl": "0",
			"uplLiab": "0",
			"stgyEq":"0"
		},
		{
			"availBal": "",
			"availEq": "33.6799714158199414",
			"cashBal": "33.2009985",
			"ccy": "BTC",
			"crossLiab": "0",
			"disEq": "1239950.9687532129092577",
			"eq": "33.771820625136023",
			"eqUsd": "1239950.9687532129092577",
			"frozenBal": "0.0918492093160816",
			"interest": "0",
			"isoEq": "0",
			"isoLiab": "0",
			"isoUpl":"0",
			"liab": "0",
			"maxLoan": "1453.92289531493594",
			"mgnRatio": "",
			"notionalLever": "",
			"ordFrozen": "0",
			"twap": "0",
			"uTime": "1620722938250",
			"upl": "0.570822125136023",
			"uplLiab": "0",
			"stgyEq":"0"
		}
	],
	"imr": "3372.2942371050594217",
	"isoEq": "0",
	"mgnRatio": "70375.35408747017",
	"mmr": "134.8917694842024",
	"notionalUsd": "33722.9423710505978888",
	"ordFroz": "0",
	"totalEq": "11172992.1657531589092577",
	"uTime": "1623392334718"
}`

func TestGetNonZeroAccountBalance(t *testing.T) {
	var account Account
	if er := json.Unmarshal([]byte(accountBalanceInformation), &account); er != nil {
		t.Error("Okx Unmarshaling to account error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetNonZeroBalances(context.Background(), ""); er != nil {
		t.Error("Okx GetBalance() error", er)
	}
}

var accountPositin = ` {
	"adl":"1",
	"availPos":"1",
	"avgPx":"2566.31",
	"cTime":"1619507758793",
	"ccy":"ETH",
	"deltaBS":"",
	"deltaPA":"",
	"gammaBS":"",
	"gammaPA":"",
	"imr":"",
	"instId":"ETH-USD-210430",
	"instType":"FUTURES",
	"interest":"0",
	"usdPx":"",
	"last":"2566.22",
	"lever":"10",
	"liab":"",
	"liabCcy":"",
	"liqPx":"2352.8496681818233",
	"markPx":"2353.849",
	"margin":"0.0003896645377994",
	"mgnMode":"isolated",
	"mgnRatio":"11.731726509588816",
	"mmr":"0.0000311811092368",
	"notionalUsd":"2276.2546609009605",
	"optVal":"",
	"pTime":"1619507761462",
	"pos":"1",
	"posCcy":"",
	"posId":"307173036051017730",
	"posSide":"long",
	"thetaBS":"",
	"thetaPA":"",
	"tradeId":"109844",
	"uTime":"1619507761462",
	"upl":"-0.0000009932766034",
	"uplRatio":"-0.0025490556801078",
	"vegaBS":"",
	"vegaPA":""
  }`

func TestGetPositions(t *testing.T) {
	t.Parallel()
	var accountPosition AccountPosition
	if er := json.Unmarshal([]byte(accountPositin), &accountPosition); er != nil {
		t.Error("Okx Unmarshaling to AccountPosition error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetPositions(context.Background(), "", "", ""); er != nil {
		t.Error("Okx GetPositions() error", er)
	}
}

var accountPositionHistoryJson = `{
	"cTime": "1654177169995",
	"ccy": "BTC",
	"closeAvgPx": "29786.5999999789081085",
	"closeTotalPos": "1",
	"instId": "BTC-USD-SWAP",
	"instType": "SWAP",
	"lever": "10.0",
	"mgnMode": "cross",
	"openAvgPx": "29783.8999999995535393",
	"openMaxPos": "1",
	"pnl": "0.00000030434156",
	"pnlRatio": "0.000906447858888",
	"posId": "452587086133239818",
	"posSide": "long",
	"triggerPx": "",
	"type": "allClose",
	"uTime": "1654177174419",
	"uly": "BTC-USD"
}`

func TestGetPositionsHistory(t *testing.T) {
	t.Parallel()
	var accountHistory AccountPositionHistory
	if er := json.Unmarshal([]byte(accountPositionHistoryJson), &accountHistory); er != nil {
		t.Error("Okx Unmarshal AccountPositionHistory error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetPositionsHistory(context.Background(), "", "", "", 0, time.Time{}, time.Time{}, 10); er != nil {
		t.Error("Okx GetPositionsHistory() error", er)
	}
}

var accountAndPositionRiskJson = `{
	"adjEq":"174238.6793649711331679",
	"balData":[
		{
			"ccy":"BTC",
			"disEq":"78846.7803721021362242",
			"eq":"1.3863533369419636"
		},
		{
			"ccy":"USDT",
			"disEq":"73417.2495112863300127",
			"eq":"73323.395564963177146"
		}
	],
	"posData":[
		{
			"baseBal": "0.4",
			"ccy": "",
			"instId": "BTC-USDT",
			"instType": "MARGIN",
			"mgnMode": "isolated",
			"notionalCcy": "0",
			"notionalUsd": "0",
			"pos": "0",
			"posCcy": "",
			"posId": "310388685292318723",
			"posSide": "net",
			"quoteBal": "0"
		}
	],
	"ts":"1620282889345"
}`

func TestGetAccountAndPositionRisk(t *testing.T) {
	t.Parallel()
	var accountAndPositionRisk AccountAndPositionRisk
	if er := json.Unmarshal([]byte(accountAndPositionRiskJson), &accountAndPositionRisk); er != nil {
		t.Error("Okx Decerializing AccountAndPositionRisk error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetAccountAndPositionRisk(context.Background(), ""); er != nil {
		t.Error("Okx GetAccountAndPositionRisk() error", er)
	}
}

func TestGetBillsDetail(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetBillsDetailLast7Days(context.Background(), BillsDetailQueryParameter{}); er != nil {
		t.Error("Okx GetBillsDetailLast7Days() error", er)
	}
}

func TestGetAccountConfiguration(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetAccountConfiguration(context.Background()); er != nil {
		t.Error("Okx GetAccountConfiguration() error", er)
	}
}

func TestSetPositionMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.SetPositionMode(context.Background(), "net_mode"); er != nil {
		t.Error("Okx SetPositionMode() error", er)
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.SetLeverage(context.Background(), SetLeverageInput{
		Leverage:     30,
		MarginMode:   "isolated",
		InstrumentID: "BTC-USDT",
		PositionSide: "long",
	}); er != nil && !errors.Is(er, errNoValidResponseFromServer) && !strings.Contains(er.Error(), "System error, please try again later.") {
		t.Error("Okx SetLeverage() error", er)
	}
}

func TestGetMaximumBuySellAmountOROpenAmount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetMaximumBuySellAmountOROpenAmount(context.Background(), "BTC-USDT", "cross", "BTC", "", 5); er != nil {
		t.Error("Okx GetMaximumBuySellAmountOROpenAmount() error", er)
	}
}

func TestGetMaximumAvailableTradableAmount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetMaximumAvailableTradableAmount(context.Background(), "BTC-USDT", "BTC", "cross", true, 123); er != nil {
		t.Error("Okx GetMaximumAvailableTradableAmount() error", er)
	}
}

func TestIncreaseDecreaseMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.IncreaseDecreaseMargin(context.Background(), IncreaseDecreaseMarginInput{
		InstrumentID: "BTC-USDT",
		PositionSide: "long",
		Type:         "add",
		Amount:       1000,
		Currency:     "USD",
	}); er != nil && !strings.Contains(er.Error(), "Unsupported operation") {
		t.Error("Okx IncreaseDecreaseMargin() error", er)
	}
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetLeverage(context.Background(), "BTC-USDT", "cross"); er != nil {
		t.Error("Okx GetLeverage() error", er)
	}
}

func TestGetMaximumLoanOfInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetMaximumLoanOfInstrument(context.Background(), "ZRX-BTC", "isolated", "ZRX"); er != nil {
		t.Error("Okx GetMaximumLoanOfInstrument() error", er)
	}
}

func TestGetFeeRate(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetTradeFee(context.Background(), "SPOT", "", ""); er != nil {
		t.Error("Okx GetTradeFeeRate() error", er)
	}
}

func TestGetInterestAccruedData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetInterestAccruedData(context.Background(), 0, "", "", "", time.Time{}, time.Time{}, 10); er != nil {
		t.Error("Okx GetInterestAccruedData() error", er)
	}
}

func TestGetInterestRate(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetInterestRate(context.Background(), ""); er != nil {
		t.Error("Okx GetInterestRate() error", er)
	}
}

func TestSetGeeks(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.SetGeeks(context.Background(), "PA"); er != nil {
		t.Error("Okx SetGeeks() error", er)
	}
}

func TestIsolatedMarginTradingSettings(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.IsolatedMarginTradingSettings(context.Background(), IsolatedMode{
		IsoMode:        "autonomy",
		InstrumentType: "MARGIN",
	}); er != nil && !strings.Contains(er.Error(), "Operation is not supported under the current account mode") {
		t.Error("Okx IsolatedMarginTradingSettings() error", er)
	}
}

func TestGetMaximumWithdrawals(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetMaximumWithdrawals(context.Background(), "BTC"); er != nil {
		t.Error("Okx GetMaximumWithdrawals() error", er)
	}
}

func TestGetAccountRiskState(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetAccountRiskState(context.Background()); er != nil {
		t.Error("Okx GetAccountRiskState() error", er)
	}
}

func TestVIPLoansBorrowAndRepay(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.VIPLoansBorrowAndRepay(context.Background(), LoanBorrowAndReplayInput{Currency: "BTC", Side: "borrow", Amount: 12}); er != nil &&
		!strings.Contains(er.Error(), "Your account does not support VIP loan") {
		t.Error("Okx VIPLoansBorrowAndRepay() error", er)
	}
}

func TestGetBorrowAndRepayHistoryForVIPLoans(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetBorrowAndRepayHistoryForVIPLoans(context.Background(), "", time.Time{}, time.Time{}, 12); er != nil {
		t.Error("Okx GetBorrowAndRepayHistoryForVIPLoans() error", er)
	}
}

func TestGetBorrowInterestAndLimit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetBorrowInterestAndLimit(context.Background(), 1, "BTC"); er != nil {
		t.Error("Okx GetBorrowInterestAndLimit() error", er)
	}
}

var positionBuilderJson = `{
	"imr": "0.005432310199023",
	"mmr": "0.0041787001530946",
	"mr1": "0.0041787001530946",
	"mr2": "0.0000734347499275",
	"mr3": "0",
	"mr4": "0",
	"mr5": "0",
	"mr6": "0.0028031968471",
	"mr7": "0.0022",
	"posData": [
		{
			"delta": "-0.008926024905498",
			"gamma": "-0.0707804093543001",
			"instId": "BTC-USD-220325-50000-C",
			"instType": "OPTION",
			"notionalUsd": "3782.9800000000005",
			"pos": "-1",
			"theta": "0.000093015207115",
			"vega": "-0.0000382697346669"
		}
	],
	"riskUnit": "BTC-USD",
	"ts": "1646639497536"
}`

func TestPositionBuilder(t *testing.T) {
	t.Parallel()
	var resp PositionBuilderResponse
	if er := json.Unmarshal([]byte(positionBuilderJson), &resp); er != nil {
		t.Error("Okx Decerializing to PositionBuilderResponse error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.PositionBuilder(context.Background(), PositionBuilderInput{
		ImportExistingPosition: true,
	}); er != nil {
		t.Error("Okx PositionBuilder() error", er)
	}
}

func TestGetGreeks(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetGreeks(context.Background(), ""); er != nil {
		t.Error("Okx GetGreeks() error", er)
	}
}

// Subaccount endpoint tests.

var subaccountsResponseJson = `{
	"enable":true,
	"subAcct":"test-1",
	"type":"1",
	"label":"trade futures",
	"mobile":"1818181",
	"gAuth":true,
	"canTransOut": true,
	"ts":"1597026383085"
 }`

func TestViewSubaccountList(t *testing.T) {
	t.Parallel()
	var resp SubaccountInfo
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if er := json.Unmarshal([]byte(subaccountsResponseJson), &resp); er != nil {
		t.Error("Okx Decerializing to SubaccountInfo error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.ViewSubaccountList(context.Background(), true, "", time.Time{}, time.Time{}, 10); er != nil {
		t.Error("Okx ViewSubaccountList() error", er)
	}
}

var subaccountBalanceResponseJson = `{"adjEq": "10679688.0460531643092577","details": [{"availBal": "","availEq": "9930359.9998","cashBal": "9930359.9998","ccy": "USDT","crossLiab": "0","disEq": "9439737.0772999514","eq": "9930359.9998","eqUsd": "9933041.196999946","frozenBal": "0","interest": "0","isoEq": "0","isoLiab": "0","liab": "0","maxLoan": "10000","mgnRatio": "","notionalLever": "","ordFrozen": "0","twap": "0","uTime": "1620722938250","upl": "0","uplLiab": "0"}],"imr": "3372.2942371050594217","isoEq": "0","mgnRatio": "70375.35408747017","mmr": "134.8917694842024","notionalUsd": "33722.9423710505978888","ordFroz": "0","totalEq": "11172992.1657531589092577","uTime": "1623392334718"}`

func TestGetSubaccountTradingBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	var resp SubaccountBalanceResponse
	if er := json.Unmarshal([]byte(subaccountBalanceResponseJson), &resp); er != nil {
		t.Error("Okx ", er)
	}
	if _, er := ok.GetSubaccountTradingBalance(context.Background(), ""); er != nil && !errors.Is(er, errMissingRequiredParameterSubaccountName) {
		t.Errorf("Okx GetSubaccountTradingBalance() expecting \"%v\", but found \"%v\"", errMissingRequiredParameterSubaccountName, er)
	}
	if _, er := ok.GetSubaccountTradingBalance(context.Background(), "test1"); er != nil {
		t.Error("Okx GetSubaccountTradingBalance() error", er)
	}
}

var fundingBalanceJson = `{
	"availBal": "37.11827078",
	"bal": "37.11827078",
	"ccy": "ETH",
	"frozenBal": "0"
}`

func TestGetSubaccountFundingBalance(t *testing.T) {
	t.Parallel()
	var resp FundingBalance
	if er := json.Unmarshal([]byte(fundingBalanceJson), &resp); er != nil {
		t.Error("okx Decerializing to FundingBalance error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetSubaccountFundingBalance(context.Background(), "test1", ""); er != nil && !strings.Contains(er.Error(), "\"msg\":\"Not Found\"") {
		t.Error("Okx GetSubaccountFundingBalance() error", er)
	}
}

var historyOfSubaccountTransfer = `{
	"billId": "12344",
	"type":"1",
	"ccy": "BTC",
	"amt":"2",
	"subAcct":"test-1",
	"ts":"1597026383085"
}`

func TestHistoryOfSubaccountTransfer(t *testing.T) {
	t.Parallel()
	var resp SubaccountBillItem
	if er := json.Unmarshal([]byte(historyOfSubaccountTransfer), &resp); er != nil {
		t.Error("Okx Decerializing to SubaccountBillItem error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.HistoryOfSubaccountTransfer(context.Background(), "", 0, "", time.Time{}, time.Time{}, 10); er != nil {
		t.Error("Okx HistoryOfSubaccountTransfer() error", er)
	}
}

func TestMasterAccountsManageTransfersBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.MasterAccountsManageTransfersBetweenSubaccounts(context.Background(), "BTC", 1200, 9, 9, "", "", true); er != nil && !errors.Is(er, errInvalidInvalidSubaccount) {
		t.Error("Okx MasterAccountsManageTransfersBetweenSubaccounts() error", er)
	}
	if _, er := ok.MasterAccountsManageTransfersBetweenSubaccounts(context.Background(), "BTC", 1200, 8, 8, "", "", true); er != nil && !errors.Is(er, errInvalidInvalidSubaccount) {
		t.Error("Okx MasterAccountsManageTransfersBetweenSubaccounts() error", er)
	}
}

func TestSetPermissionOfTransferOut(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.SetPermissionOfTransferOut(context.Background(), PermissingOfTransfer{SubAcct: "Test1"}); er != nil && !strings.Contains(er.Error(), "Sub-account does not exist") {
		t.Error("Okx SetPermissionOfTransferOut() error", er)
	}
}

func TestGetCustodyTradingSubaccountList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetCustodyTradingSubaccountList(context.Background(), ""); er != nil {
		t.Error("Okx GetCustodyTradingSubaccountList() error", er)
	}
}

var gridTradingPlaceOrder = `{"instId": "BTC-USDT-SWAP","algoOrdType": "contract_grid","maxPx": "5000","minPx": "400","gridNum": "10","runType": "1","sz": "200", "direction": "long","lever": "2"}`

func TestPlaceGridAlgoOrder(t *testing.T) {
	t.Parallel()
	var input GridAlgoOrder
	if er := json.Unmarshal([]byte(gridTradingPlaceOrder), &input); er != nil {
		t.Error("Okx Decerializing to GridALgoOrder error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.PlaceGridAlgoOrder(context.Background(), input); er != nil {
		t.Error("Okx PlaceGridAlgoOrder() error", er)
	}
}

var gridOrderAmendAlgo = `{
    "algoId":"448965992920907776",
    "instId":"BTC-USDT",
    "slTriggerPx":"1200",
    "tpTriggerPx":""
}`

func TestAmendGridAlgoOrder(t *testing.T) {
	t.Parallel()
	var input GridAlgoOrderAmend
	if er := json.Unmarshal([]byte(gridOrderAmendAlgo), &input); er != nil {
		t.Error("Okx Decerializing to GridAlgoOrderAmend error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.AmendGridAlgoOrder(context.Background(), input); er != nil {
		t.Error("Okx AmendGridAlgoOrder() error", er)
	}
}

func TestStopGridAlgoOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.StopGridAlgoOrder(context.Background(), []StopGridAlgoOrderRequest{}); er != nil {
		t.Error("Okx StopGridAlgoOrder() error", er)
	}
}

func TestGetGridAlgoOrdersList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetGridAlgoOrdersList(context.Background(), "grid", "", "", "", "", "", 100); er != nil {
		t.Error("Okx GetGridAlgoOrdersList() error", er)
	}
}

func TestGetGridAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetGridAlgoOrderHistory(context.Background(), "contract_grid", "", "", "", "", "", 100); er != nil {
		t.Error("Okx GetGridAlgoOrderHistory() error", er)
	}
}

func TestGetGridAlgoOrderDetails(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetGridAlgoOrderDetails(context.Background(), "grid", ""); er != nil && !errors.Is(er, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderDetails() expecting %v, but found %v error", errMissingAlgoOrderID, er)
	}
	if _, er := ok.GetGridAlgoOrderDetails(context.Background(), "grid", "7878"); er != nil && !errors.Is(er, errNoValidResponseFromServer) {
		t.Error("Okx GetGridAlgoOrderDetails() error", er)
	}
}

func TestGetGridAlgoSubOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetGridAlgoSubOrders(context.Background(), "", "", "", "", "", "", 10); er != nil && !errors.Is(er, errMissingAlgoOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", er, errMissingAlgoOrderType)
	}
	if _, er := ok.GetGridAlgoSubOrders(context.Background(), "grid", "", "", "", "", "", 10); er != nil && !errors.Is(er, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", er, errMissingAlgoOrderID)
	}
	if _, er := ok.GetGridAlgoSubOrders(context.Background(), "grid", "1234", "", "", "", "", 10); er != nil && !errors.Is(er, errMissingSubOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", er, errMissingSubOrderType)
	}
	if _, er := ok.GetGridAlgoSubOrders(context.Background(), "grid", "1234", "live", "", "", "", 10); er != nil && !errors.Is(er, errMissingSubOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", er, errMissingSubOrderType)
	}
}

var spotGridAlgoOrderPosition = `{"adl": "1","algoId": "449327675342323712","avgPx": "29215.0142857142857149","cTime": "1653400065917","ccy": "USDT","imr": "2045.386","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29206.7","lever": "5","liqPx": "661.1684795867162","markPx": "29213.9","mgnMode": "cross","mgnRatio": "217.19370606167573","mmr": "40.907720000000005","notionalUsd": "10216.70307","pos": "35","posSide": "net","uTime": "1653400066938","upl": "1.674999999999818","uplRatio": "0.0008190504784478"}`

func TestGetGridAlgoOrderPositions(t *testing.T) {
	t.Parallel()
	var resp AlgoOrderPosition
	if er := json.Unmarshal([]byte(spotGridAlgoOrderPosition), &resp); er != nil {
		t.Error("Okx Decerializing to AlgoOrderPosition error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetGridAlgoOrderPositions(context.Background(), "", ""); er != nil && !errors.Is(er, errMissingAlgoOrderType) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderType, er)
	}
	if _, er := ok.GetGridAlgoOrderPositions(context.Background(), "contract_grid", ""); er != nil && !errors.Is(er, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderID, er)
	}
	if _, er := ok.GetGridAlgoOrderPositions(context.Background(), "contract_grid", ""); er != nil && !errors.Is(er, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderID, er)
	}
}

func TestSpotGridWithdrawProfit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.SpotGridWithdrawProfit(context.Background(), ""); er != nil && !errors.Is(er, errMissingAlgoOrderID) {
		t.Errorf("Okx SpotGridWithdrawProfit() expecting %v, but found %v", errMissingAlgoOrderID, er)
	}
	if _, er := ok.SpotGridWithdrawProfit(context.Background(), "1234"); er != nil && strings.Contains(er.Error(), "Policy type is not grid policy") {
		t.Skip("Policy type is not grid policy")
	} else if er != nil {
		t.Error("Okx SpotGridWithdrawProfit() error", er)
	}
}

var systemStatusResponseJson = `{"title": "Spot System Upgrade","state": "scheduled","begin": "1620723600000","end": "1620724200000","href": "","serviceType": "1","system": "classic","scheDesc": ""}`

func TestSystemStatusResponse(t *testing.T) {
	t.Parallel()
	var resp SystemStatusResponse
	if er := json.Unmarshal([]byte(systemStatusResponseJson), &resp); er != nil {
		t.Error("Okx Decerializing to SystemStatusResponse error", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.SystemStatusResponse(context.Background(), ""); er != nil {
		t.Error("Okx SystemStatusResponse() error", er)
	}
}

/**********************************  Wrapper Functions **************************************/

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	if _, er := ok.FetchTradablePairs(context.Background(), asset.Futures); er != nil {
		t.Error("Okx FetchTradablePairs() error", er)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	if er := ok.UpdateTradablePairs(context.Background(), true); er != nil {
		t.Error("Okx UpdateTradablePairs() error", er)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	if _, er := ok.UpdateTicker(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Futures); er != nil {
		t.Error("Okx UpdateTicker() error", er)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	if er := ok.UpdateTickers(context.Background(), asset.Futures); er != nil {
		t.Error("Okx UpdateTicker() error", er)
	}
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	if _, er := ok.FetchTicker(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.PerpetualSwap); er != nil {
		t.Error("Okx FetchTicker() error", er)
	}
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.FetchOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot); er != nil {
		t.Error("Okx FetchOrderbook() error", er)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.UpdateOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot); er != nil {
		t.Error("Okx UpdateOrderbook() error", er)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.UpdateAccountInfo(context.Background(), asset.Spot); er != nil {
		t.Error("Okx UpdateAccountInfo() error", er)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.FetchAccountInfo(context.Background(), asset.Spot); er != nil {
		t.Error("Okx FetchAccountInfo() error", er)
	}
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetFundingHistory(context.Background()); er != nil {
		t.Error("Okx GetFundingHistory() error", er)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetWithdrawalsHistory(context.Background(), currency.BTC); er != nil {
		t.Error("Okx GetWithdrawalsHistory() error", er)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	if _, er := ok.GetRecentTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.PerpetualSwap); er != nil {
		t.Error("Okx GetRecentTrades() error", er)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.LTC,
			Quote: currency.BTC,
		},
		Exchange:  ok.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "yeneOrder",
		AssetType: asset.Spot,
	}
	_, err := ok.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error("Okx SubmitOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}
	err := ok.CancelOrder(context.Background(), orderCancellation)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("CancelExchangeOrder() error", err)
	case !areTestAPIKeysSet() && err == nil:
		t.Error("CancelExchangeOrder() expecting an error when no keys are set")
	case err != nil:
		t.Error("Mock CancelExchangeOrder() error", err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	var orderCancellationParams = []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          currency.NewPair(currency.LTC, currency.BTC),
			AssetType:     asset.Spot,
		},
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          currency.NewPair(currency.LTC, currency.BTC),
			AssetType:     asset.PerpetualSwap,
		},
	}
	_, err := ok.CancelBatchOrders(context.Background(), orderCancellationParams)
	if err != nil {
		t.Error("Okx CancelBatchOrders() error", err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.ModifyOrder(context.Background(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("Okx ModifyOrder() error cannot be nil")
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("Okx GetOrderInfo() skipping test: api keys not set")
	}
	tradablePairs, err := ok.FetchTradablePairs(context.Background(),
		asset.Futures)
	if err != nil {
		t.Error("Okx GetOrderInfo() error", err)
	}
	if len(tradablePairs) == 0 {
		t.Fatal("Okx GetOrderInfo() no tradable pairs")
	}
	cp, err := currency.NewPairFromString(tradablePairs[0])
	if err != nil {
		t.Error("Okx GetOrderinfo() error", err)
	}
	_, err = ok.GetOrderInfo(context.Background(),
		"123", cp, asset.Futures)
	if err != nil && !strings.Contains(err.Error(), "Order does not exist") {
		t.Errorf("Okx GetOrderInfo() expecting %s, but found %v", "Order does not exist", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetDepositAddress(context.Background(), currency.USDT, "", currency.BNB.String()); err != nil && !errors.Is(err, errDepositAddressNotFound) {
		t.Error("Okx GetDepositAddress() error", err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	withdrawCryptoRequest := withdraw.Request{
		Exchange: ok.Name,
		Amount:   0.00000000001,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}
	if _, err := ok.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest); err != nil && !strings.Contains(err.Error(), "Invalid Authority") {
		t.Error("Okx WithdrawCryptoCurrencyFunds() error", err)
	}
}

func TestGetPairFromInstrumentID(t *testing.T) {
	t.Parallel()
	instruments := []string{
		"BTC-USDT",
		"BTC-USDT-SWAP",
		"BTC-USDT-ER33234",
	}
	if _, er := ok.GetPairFromInstrumentID(instruments[0]); er != nil {
		t.Error("Okx GetPairFromInstrumentID() error", er)
	}
	if _, ere := ok.GetPairFromInstrumentID(instruments[1]); ere != nil {
		t.Error("Okx GetPairFromInstrumentID() error", ere)
	}
	if _, erf := ok.GetPairFromInstrumentID(instruments[2]); erf != nil {
		t.Error("Okx GetPairFromInstrumentID() error", erf)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	pair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		Pairs:     currency.Pairs{pair},
		AssetType: asset.Spot,
	}
	if _, err = ok.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Okx GetActiveOrders() error", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}
	_, err := ok.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err == nil {
		t.Errorf("Okx GetOrderHistory() Expected: %v. received nil", err)
	} else if err != nil && !errors.Is(err, errMissingAtLeast1CurrencyPair) {
		t.Errorf("Okx GetOrderHistory() Expected: %v, but found %v", errMissingAtLeast1CurrencyPair, err)
	}
	getOrdersRequest.Pairs = []currency.Pair{
		currency.NewPair(currency.LTC,
			currency.BTC)}
	if _, er := ok.GetOrderHistory(context.Background(), &getOrdersRequest); er != nil {
		t.Error("Okx GetOrderHistory() error", er)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	startTime := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2021, 2, 15, 0, 0, 0, 0, time.UTC)
	_, er := ok.GetHistoricCandles(context.Background(), pair, asset.Spot, startTime, endTime, kline.Interval(time.Hour*5))
	if er != nil && !strings.Contains(er.Error(), "interval not supported") {
		t.Errorf("Okx GetHistoricCandles() expected %s, but found %v", "interval not supported", er)
	}
	_, er = ok.GetHistoricCandles(context.Background(), pair, asset.Spot, time.Time{}, time.Time{}, kline.Interval(time.Hour*4))
	if er != nil {
		t.Error("Okx GetHistoricCandles() error", er)
	}
}

var wsInstrumentResp = `{"arg": {"channel": "instruments","instType": "FUTURES"},"data": [{"instType": "FUTURES","instId": "BTC-USD-191115","uly": "BTC-USD","category": "1","baseCcy": "","quoteCcy": "","settleCcy": "BTC","ctVal": "10","ctMult": "1","ctValCcy": "USD","optType": "","stk": "","listTime": "","expTime": "","tickSz": "0.01","lotSz": "1","minSz": "1","ctType": "linear","alias": "this_week","state": "live","maxLmtSz":"10000","maxMktSz":"99999","maxTwapSz":"99999","maxIcebergSz":"99999","maxTriggerSz":"9999","maxStopSz":"9999"}]}`

func TestWSInstruments(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(wsInstrumentResp)); er != nil {
		t.Errorf("%s Websocket Instruments Push Data error %v", ok.Name, er)
	}
}

var tickerChannelJSON = `{"arg": {"channel": "tickers","instId": "LTC-USD-200327"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","last": "9999.99","lastSz": "0.1","askPx": "9999.99","askSz": "11","bidPx": "8888.88","bidSz": "5","open24h": "9000","high24h": "10000","low24h": "8888.88","volCcy24h": "2222","vol24h": "2222","sodUtc0": "2222","sodUtc8": "2222","ts": "1597026383085"}]}`

func TestTickerChannel(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(tickerChannelJSON)); er != nil {
		t.Error("Okx TickerChannel push data error", er)
	}
}

var openInterestChannel = `{"arg": {"channel": "open-interest","instId": "LTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","oi": "5000","oiCcy": "555.55","ts": "1597026383085"}]}`

func TestOpenInterestPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(openInterestChannel)); er != nil {
		t.Error("Okx Open Interest Push Data error", er)
	}
}

var candlesticksPushData = `{"arg": {"channel": "candle1D","instId": "BTC-USD-191227"},"data": [["1597026383085","8533.02","8553.74","8527.17","8548.26","45247","529.5858061"]]}`

func TestCandlestickPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(candlesticksPushData)); er != nil {
		t.Error("Okx Candlestick Push Data error", er)
	}
}

var tradePushDataJSON = `{"arg": {"channel": "trades","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","tradeId": "130639474","px": "42219.9","sz": "0.12060306","side": "buy","ts": "1630048897897"}]}`

func TestTradePushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(tradePushDataJSON)); er != nil {
		t.Error("Okx Trade Push Data error", er)
	}
}

var estimatedDeliveryAndExercisePricePushDataJSON = `{"arg": {"args": "estimated-price","instType": "FUTURES","uly": "BTC-USD"},"data": [{"instType": "FUTURES","instId": "BTC-USD-170310","settlePx": "200","ts": "1597026383085"}]}`

func TestEstimatedDeliveryAndExercisePricePushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(estimatedDeliveryAndExercisePricePushDataJSON)); er != nil {
		t.Error("Okx Estimated Delivery and Exercise Price Push Data error", er)
	}
}

var markPricePushData = `{"arg": {"channel": "mark-price","instId": "LTC-USD-190628"},"data": [{"instType": "FUTURES","instId": "LTC-USD-190628","markPx": "0.1","ts": "1597026383085"}]}`

func TestMarkPricePushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(markPricePushData)); er != nil {
		t.Error("Okx Mark Price Push Data error", er)
	}
}

var markPriceCandlestickPushData = `{"arg": {"channel": "mark-price-candle1D","instId": "BTC-USD-190628"},"data": [["1597026383085", "3.721", "3.743", "3.677", "3.708"],["1597026383085", "3.731", "3.799", "3.494", "3.72"]]}`

func TestMarkPriceCandlestickPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(markPriceCandlestickPushData)); er != nil {
		t.Error("Okx Mark Price Candlestick Push Data error", er)
	}
}

var priceLimitPushDataJSON = `{"arg": {"channel": "mark-price","instId": "BTC-USDT"},"data": [{"instType": "MARGIN","instId": "BTC-USDT","markPx": "42310.6","ts": "1630049139746"}]}`

func TestPriceLimitPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(priceLimitPushDataJSON)); er != nil {
		t.Error("Okx Price Limit Push Data error", er)
	}
}

var snapshotOrderBookPushData = `{"arg":{"channel":"books","instId":"TRX-USD-220812"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`

func TestSnapshotOrderBookPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(snapshotOrderBookPushData)); er != nil {
		t.Error("Okx Snapshot order book push data error", er)
	}
}

var updateOrderBookPushDataJSON = `{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"update","data":[{"asks":[["23209.9","0","0","0"],["23211.2","0.02871319","0","2"],["23331.6","0.0008665","0","2"]],"bids":[["23187.6","0.05","0","1"],["23185.5","0","0","0"],["23119.3","0","0","0"],["23071.5","0.00011777","0","1"]],"ts":"1659794601607","checksum":-350736830}]}`

func TestUpdateOrderBookPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(updateOrderBookPushDataJSON)); er != nil {
		t.Error("Okx Update Order Book Push Data error", er)
	}
}

var optionSummaryPushDataJSON = `{"arg": {"channel": "opt-summary","uly": "BTC-USD"},"data": [{"instType": "OPTION","instId": "BTC-USD-200103-5500-C","uly": "BTC-USD","delta": "0.7494223636","gamma": "-0.6765419039","theta": "-0.0000809873","vega": "0.0000077307","deltaBS": "0.7494223636","gammaBS": "-0.6765419039","thetaBS": "-0.0000809873","vegaBS": "0.0000077307","realVol": "0","bidVol": "","askVol": "1.5625","markVol": "0.9987","lever": "4.0342","fwdPx": "39016.8143629068452065","ts": "1597026383085"}]}`

func TestOptionSummaryPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(optionSummaryPushDataJSON)); er != nil {
		t.Error("Okx Option Summary Push Data error", er)
	}
}

var fundingRatePushDataJSON = `{"arg": {"channel": "funding-rate","instId": "BTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "BTC-USD-SWAP","fundingRate": "0.018","nextFundingRate": "","fundingTime": "1597026383085"}]}`

func TestFundingRatePushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(fundingRatePushDataJSON)); er != nil {
		t.Error("Okx Funding Rate Push Data error", er)
	}
}

var indexCandlestickPushDataJSON = `{"arg": {"channel": "index-candle30m","instId": "BTC-USD"},"data": [["1597026383085", "3811.31", "3811.31", "3811.31", "3811.31"]]}`

func TestIndexCandlestickPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(indexCandlestickPushDataJSON)); er != nil {
		t.Error("Okx Index Candlestick Push Data error", er)
	}
}

var indexTickerPushDataJSON = `{"arg": {"channel": "index-tickers","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","idxPx": "0.1","high24h": "0.5","low24h": "0.1","open24h": "0.1","sodUtc0": "0.1","sodUtc8": "0.1","ts": "1597026383085"}]}`

func TestIndexTickersPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(indexTickerPushDataJSON)); er != nil {
		t.Error("Okx Index Ticker Push Data error", er)
	}
}

var statusPushDataJSON = `{"arg": {"channel": "status"},"data": [{"title": "Spot System Upgrade","state": "scheduled","begin": "1610019546","href": "","end": "1610019546","serviceType": "1","system": "classic","scheDesc": "","ts": "1597026383085"}]}`

func TestStatusPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(statusPushDataJSON)); er != nil {
		t.Error("Okx Status Push Data error", er)
	}
}

var publicStructBlockTradesPushDataJSON = `{"arg":{"channel":"public-struc-block-trades"},"data":[{"cTime":"1608267227834","blockTdId":"1802896","legs":[{"px":"0.323","sz":"25.0","instId":"BTC-USD-20220114-13250-C","side":"sell","tradeId":"15102"},{"px":"0.666","sz":"25","instId":"BTC-USD-20220114-21125-C","side":"buy","tradeId":"15103"}]}]}`

func TestPublicStructBlockTrades(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(publicStructBlockTradesPushDataJSON)); er != nil {
		t.Error("Okx Public Struct Block Trades error", er)
	}
}

var blockTickerPushDataJSON = `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`

func TestBlockTickerPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(blockTickerPushDataJSON)); er != nil {
		t.Error("Okx Block Tickers push data error", er)
	}
}

var accountPushDataJSON = `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`

func TestAccountPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(accountPushDataJSON)); er != nil {
		t.Error("Okx Account Push Data error", er)
	}
}

var positionPushDataJSON = `{"arg":{"channel":"positions","instType":"FUTURES"},"data":[{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`
var positionPushDataWithUnderlyingJSON = `{"arg": {"channel": "positions","uid": "77982378738415879","instType": "ANY"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`

func TestPositionPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(positionPushDataJSON)); er != nil {
		t.Error("Okx Account Push Data error", er)
	}
	if er := ok.WsHandleData([]byte(positionPushDataWithUnderlyingJSON)); er != nil {
		t.Error("Okx Account Push Data error", er)
	}
}

var balanceAndPositionJSON = `{"arg": {"channel": "balance_and_position","uid": "77982378738415879"},"data": [{"pTime": "1597026383085","eventType": "snapshot","balData": [{"ccy": "BTC","cashBal": "1","uTime": "1597026383085"}],"posData": [{"posId": "1111111111","tradeId": "2","instId": "BTC-USD-191018","instType": "FUTURES","mgnMode": "cross","posSide": "long","pos": "10","ccy": "BTC","posCcy": "","avgPx": "3320","uTIme": "1597026383085"}]}]}`

func TestBalanceAndPosition(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(balanceAndPositionJSON)); er != nil {
		t.Error("Okx Balance And Position error", er)
	}
}

var orderPushDataJSON = `{"arg": {"channel": "balance_and_position","uid": "77982378738415879"},"data": [{"pTime": "1597026383085","eventType": "snapshot","balData": [{"ccy": "BTC","cashBal": "1","uTime": "1597026383085"}],"posData": [{"posId": "1111111111","tradeId": "2","instId": "BTC-USD-191018","instType": "FUTURES","mgnMode": "cross","posSide": "long","pos": "10","ccy": "BTC","posCcy": "","avgPx": "3320","uTIme": "1597026383085"}]}]}`

func TestOrderPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(orderPushDataJSON)); er != nil {
		t.Error("Okx Order Push Data error", er)
	}
}

var algoOrdersPushDataJSON = `{"arg": {"channel": "orders-algo","uid": "77982378738415879","instType": "FUTURES","instId": "BTC-USD-200329"},"data": [{"instType": "FUTURES","instId": "BTC-USD-200329","ordId": "312269865356374016","ccy": "BTC","algoId": "1234","px": "999","sz": "3","tdMode": "cross","tgtCcy": "","notionalUsd": "","ordType": "trigger","side": "buy","posSide": "long","state": "live","lever": "20","tpTriggerPx": "","tpTriggerPxType": "","tpOrdPx": "","slTriggerPx": "","slTriggerPxType": "","triggerPx": "99","triggerPxType": "last","ordPx": "12","actualSz": "","actualPx": "","tag": "adadadadad","actualSide": "","triggerTime": "1597026383085","cTime": "1597026383000"}]}`

func TestAlgoOrderPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(algoOrdersPushDataJSON)); er != nil {
		t.Error("Okx Algo Order Push Data error", er)
	}
}

var advancedAlgoOrderPushDataJSON = `{"arg":{"channel":"algo-advance","uid": "77982378738415879","instType":"SPOT","instId":"BTC-USDT"},"data":[{"actualPx":"","actualSide":"","actualSz":"0","algoId":"355056228680335360","cTime":"1630924001545","ccy":"","count":"1","instId":"BTC-USDT","instType":"SPOT","lever":"0","notionalUsd":"","ordPx":"","ordType":"iceberg","pTime":"1630924295204","posSide":"net","pxLimit":"10","pxSpread":"1","pxVar":"","side":"buy","slOrdPx":"","slTriggerPx":"","state":"pause","sz":"0.1","szLimit":"0.1","tdMode":"cash","timeInterval":"","tpOrdPx":"","tpTriggerPx":"","tag": "adadadadad","triggerPx":"","triggerTime":"","callbackRatio":"","callbackSpread":"","activePx":"","moveTriggerPx":""}]}`

func TestAdvancedAlgoOrderPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(advancedAlgoOrderPushDataJSON)); er != nil {
		t.Error("Okx Advanced Algo Orders Push Data error", er)
	}
}

var positionRiskPushDataJSON = `{"arg": {"channel": "liquidation-warning","uid": "77982378738415879","instType": "ANY"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`

func TestPositionRiskPushDataJSON(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(positionRiskPushDataJSON)); er != nil {
		t.Error("Okx Position Risk Push Data error", er)
	}
}

var accountGreeksPushData = `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"",    "vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`

func TestAccountGreeksPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(accountGreeksPushData)); er != nil {
		t.Error("Okx Account Greeks Push Data error", er)
	}
}

var rfqsPushDataJSON = `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"",    "vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`

func TestRfqs(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(rfqsPushDataJSON)); er != nil {
		t.Error("Okx RFQS Push Data error", er)
	}
}

var quotesPushDataJSON = `{"arg":{"channel":"quotes"},"data":[{"validUntil":"1608997227854","uTime":"1608267227834","cTime":"1608267227834","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-220114-25000-C","side":"sell","tgtCcy":""},{"px":"0.0045","sz":"25","instId":"BTC-USD-220114-35000-C","side":"buy","tgtCcy":""}],"quoteId":"25092","rfqId":"18753","traderCode":"SATS","quoteSide":"sell","state":"canceled","clQuoteId":""}]}`

func TestQuotesPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(quotesPushDataJSON)); er != nil {
		t.Error("Okx Quotes Push Data error", er)
	}
}

var structureBlockTradesPushDataJSON = `{"arg":{"channel":"struc-block-trades"},"data":[{"cTime":"1608267227834","rfqId":"18753","clRfqId":"","quoteId":"25092","clQuoteId":"","blockTdId":"180184","tTraderCode":"ANAND","mTraderCode":"WAGMI","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-20220630-60000-C","side":"sell","fee":"0.1001","feeCcy":"BTC","tradeId":"10211","tgtCcy":""},{"px":"0.0033","sz":"25","instId":"BTC-USD-20220630-50000-C","side":"buy","fee":"0.1001","feeCcy":"BTC","tradeId":"10212","tgtCcy":""}]}]}`

func TestStructureBlockTradesPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(structureBlockTradesPushDataJSON)); er != nil {
		t.Error("Okx Structure Block Trades error", er)
	}
}

var spotGridAlgoOrdersPushDataJSON = `{"arg": {"channel": "grid-orders-spot","instType": "ANY"},"data": [{"algoId": "448965992920907776","algoOrdType": "grid","annualizedRate": "0","arbitrageNum": "0","baseSz": "0","cTime": "1653313834104","cancelType": "0","curBaseSz": "0.001776289214","curQuoteSz": "46.801755866","floatProfit": "-0.4953878967772","gridNum": "6","gridProfit": "0","instId": "BTC-USDC","instType": "SPOT","investment": "100","maxPx": "33444.8","minPx": "24323.5","pTime": "1653476023742","perMaxProfitRate": "0.060375293181491054543","perMinProfitRate": "0.0455275366818586","pnlRatio": "0","quoteSz": "100","runPx": "30478.1","runType": "1","singleAmt": "0.00059261","slTriggerPx": "","state": "running","stopResult": "0","stopType": "0","totalAnnualizedRate": "-0.9643551057262827","totalPnl": "-0.4953878967772","tpTriggerPx": "","tradeNum": "3","triggerTime": "1653378736894","uTime": "1653378736894"}]}`

func TestSpotGridAlgoOrdersPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(spotGridAlgoOrdersPushDataJSON)); er != nil {
		t.Error("Okx Spot Grid Algo Orders Push Data error", er)
	}
}

var contractGridAlgoOrdersPushDataJSON = `{"arg": {"channel": "grid-orders-contract","instType": "ANY"},"data": [{"actualLever": "1.02","algoId": "449327675342323712","algoOrdType": "contract_grid","annualizedRate": "0.7572437878956523","arbitrageNum": "1","basePos": true,"cTime": "1653400065912","cancelType": "0","direction": "long","eq": "10129.419829834853","floatProfit": "109.537858234853","gridNum": "50","gridProfit": "19.8819716","instId": "BTC-USDT-SWAP","instType": "SWAP","investment": "10000","lever": "5","liqPx": "603.2149534767834","maxPx": "100000","minPx": "10","pTime": "1653484573918","perMaxProfitRate": "995.7080916791230692","perMinProfitRate": "0.0946277854875634","pnlRatio": "0.0129419829834853","runPx": "29216.3","runType": "1","singleAmt": "1","slTriggerPx": "","state": "running","stopType": "0","sz": "10000","tag": "","totalAnnualizedRate": "4.929207431970923","totalPnl": "129.419829834853","tpTriggerPx": "","tradeNum": "37","triggerTime": "1653400066940","uTime": "1653484573589","uly": "BTC-USDT"}]}`

func TestContractGridAlgoOrdersPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(contractGridAlgoOrdersPushDataJSON)); er != nil {
		t.Error("Okx Contract Grid Algo Order Push Data error", er)
	}
}

var gridPositionsPushDataJSON = `{"arg": {"channel": "grid-positions","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"adl": "1","algoId": "449327675342323712","avgPx": "29181.4638888888888895","cTime": "1653400065917","ccy": "USDT","imr": "2089.2690000000002","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29852.7","lever": "5","liqPx": "604.7617536513744","markPx": "29849.7","mgnMode": "cross","mgnRatio": "217.71740878394456","mmr": "41.78538","notionalUsd": "10435.794191550001","pTime": "1653536068723","pos": "35","posSide": "net","uTime": "1653445498682","upl": "232.83263888888962","uplRatio": "0.1139826489932205"}]}`

func GridPositionsPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(gridPositionsPushDataJSON)); er != nil {
		t.Error("Okx Grid Positions Push Data error", er)
	}
}

var gridSubOrdersPushDataJSON = `{"arg": {"channel": "grid-sub-orders","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"accFillSz": "0","algoId": "449327675342323712","algoOrdType": "contract_grid","avgPx": "0","cTime": "1653445498664","ctVal": "0.01","fee": "0","feeCcy": "USDT","groupId": "-1","instId": "BTC-USDT-SWAP","instType": "SWAP","lever": "5","ordId": "449518234142904321","ordType": "limit","pTime": "1653486524502","pnl": "","posSide": "net","px": "28007.2","side": "buy","state": "live","sz": "1","tag":"","tdMode": "cross","uTime": "1653445498674"}]}`

func TestGridSubOrdersPushData(t *testing.T) {
	t.Parallel()
	if er := ok.WsHandleData([]byte(gridSubOrdersPushDataJSON)); er != nil {
		t.Error("Okx Grid Sub orders Push Data error", er)
	}
}
